package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
)

const (
	alertCooldown          = 6 * time.Hour
	alertEvalLockTTL       = 2 * time.Minute
	alertStationStaleLimit = 3 * time.Hour
	alertMaxDistanceMeters = 20000.0
)

type alertStationReader interface {
	FindByID(ctx context.Context, id string) (*models.Station, error)
}

type alertReadingReader interface {
	GetLatest(ctx context.Context, stationID string) (*models.AQIReading, error)
}

type alertCandidateFinder interface {
	FindNearestCandidatesByStation(ctx context.Context, stationID string, maxMeters float64) ([]repository.StationAlertCandidate, error)
}

type alertCooldownChecker interface {
	HasSentSince(ctx context.Context, userID, stationID string, since time.Time) (bool, error)
}

// AlertEvalLocker cung cấp distributed lock theo station_id để tránh chạy song song check_alerts.
type AlertEvalLocker interface {
	AcquireLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key, token string) error
}

// AlertDispatchInput chứa ngữ cảnh đầy đủ cho tầng gửi thông báo (S08-FG-02-T02).
type AlertDispatchInput struct {
	UserID        string
	UserEmail     string
	UserFullName  *string
	Station       *models.Station
	LocationID    string
	LocationLabel string
	Reading       *models.AQIReading
	ThresholdAQI  int
}

// AlertDispatchOutcome tóm tắt số lần thử gửi theo kết quả từng attempt.
type AlertDispatchOutcome struct {
	Sent    int
	Failed  int
	Skipped int
}

// AlertDispatcher gửi cảnh báo đa kênh cho 1 user đã vượt ngưỡng và qua cooldown.
type AlertDispatcher interface {
	Dispatch(ctx context.Context, in AlertDispatchInput) (AlertDispatchOutcome, error)
}

// CheckAlertsResult ghi nhận chi tiết kết quả 1 lần đánh giá cảnh báo theo trạm.
type CheckAlertsResult struct {
	StationID         string
	SkippedLock       bool
	SkippedInactive   bool
	SkippedStale      bool
	SkippedNoData     bool
	EvaluatedUsers    int
	BelowThreshold    int
	SkippedCooldown   int
	Triggered         int
	DispatchedSent    int
	DispatchedFailed  int
	DispatchedSkipped int
	UserErrors        int
}

// AlertService thực thi logic Alert Evaluation (check_alerts(station_id)).
type AlertService struct {
	stations   alertStationReader
	readings   alertReadingReader
	candidates alertCandidateFinder
	cooldowns  alertCooldownChecker
	locker     AlertEvalLocker
	dispatcher AlertDispatcher
	nowFunc    func() time.Time
}

func NewAlertService(
	stations alertStationReader,
	readings alertReadingReader,
	candidates alertCandidateFinder,
	cooldowns alertCooldownChecker,
	locker AlertEvalLocker,
	dispatcher AlertDispatcher,
) *AlertService {
	return &AlertService{
		stations:   stations,
		readings:   readings,
		candidates: candidates,
		cooldowns:  cooldowns,
		locker:     locker,
		dispatcher: dispatcher,
		nowFunc:    time.Now,
	}
}

func (s *AlertService) SetNowFunc(f func() time.Time) {
	s.nowFunc = f
}

func (s *AlertService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc().UTC()
	}
	return time.Now().UTC()
}

// CheckAlerts đánh giá cảnh báo cho toàn bộ user có trạm gần nhất <= 20km là stationID.
// Lỗi hạ tầng toàn cục trả về error để Asynq retry; lỗi gửi riêng lẻ của từng user được cô lập.
func (s *AlertService) CheckAlerts(ctx context.Context, stationID string) (CheckAlertsResult, error) {
	res := CheckAlertsResult{StationID: stationID}
	if stationID == "" {
		return res, errors.New("station_id is required")
	}

	if s.locker != nil {
		lockKey := fmt.Sprintf("alert:eval:lock:%s", stationID)
		token := uuid.NewString()
		acquired, err := s.locker.AcquireLock(ctx, lockKey, token, alertEvalLockTTL)
		if err != nil {
			return res, fmt.Errorf("acquire eval lock: %w", err)
		}
		if !acquired {
			res.SkippedLock = true
			slog.Info("check_alerts: skipped due to active lock", "station_id", stationID)
			return res, nil
		}
		defer func() {
			relCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer cancel()
			if relErr := s.locker.ReleaseLock(relCtx, lockKey, token); relErr != nil {
				slog.Warn("check_alerts: release lock failed", "station_id", stationID, "error", relErr)
			}
		}()
	}

	st, err := s.stations.FindByID(ctx, stationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res.SkippedInactive = true
			return res, nil
		}
		return res, fmt.Errorf("find station: %w", err)
	}
	if !st.IsActive {
		res.SkippedInactive = true
		return res, nil
	}
	if !st.LastSyncedAt.Valid || s.now().Sub(st.LastSyncedAt.Time.UTC()) > alertStationStaleLimit {
		res.SkippedStale = true
		slog.Info("check_alerts: skipped stale station", "station_id", stationID)
		return res, nil
	}

	rd, err := s.readings.GetLatest(ctx, stationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res.SkippedNoData = true
			return res, nil
		}
		return res, fmt.Errorf("get latest reading: %w", err)
	}

	rawCandidates, err := s.candidates.FindNearestCandidatesByStation(ctx, stationID, alertMaxDistanceMeters)
	if err != nil {
		return res, fmt.Errorf("find nearest alert candidates: %w", err)
	}

	candidates := dedupCandidatesByUser(rawCandidates)
	res.EvaluatedUsers = len(candidates)

	cooldownCutoff := s.now().Add(-alertCooldown)

	for _, cand := range candidates {
		// Quy tắc đã chốt: chỉ kích hoạt khi AQI thực sự vượt ngưỡng (>).
		if rd.AQI <= cand.ThresholdAQI {
			res.BelowThreshold++
			continue
		}

		sentRecently, cdErr := s.cooldowns.HasSentSince(ctx, cand.UserID, stationID, cooldownCutoff)
		if cdErr != nil {
			res.UserErrors++
			slog.Warn("check_alerts: cooldown check failed",
				"station_id", stationID, "user_id", cand.UserID, "error", cdErr)
			continue
		}
		if sentRecently {
			res.SkippedCooldown++
			continue
		}

		res.Triggered++

		if s.dispatcher == nil {
			continue
		}

		var fullName *string
		if cand.UserFullName.Valid {
			n := cand.UserFullName.String
			fullName = &n
		}

		outcome, dispErr := s.dispatcher.Dispatch(ctx, AlertDispatchInput{
			UserID:        cand.UserID,
			UserEmail:     cand.UserEmail,
			UserFullName:  fullName,
			Station:       st,
			LocationID:    cand.LocationID,
			LocationLabel: cand.LocationLabel,
			Reading:       rd,
			ThresholdAQI:  cand.ThresholdAQI,
		})
		if dispErr != nil {
			res.UserErrors++
			slog.Warn("check_alerts: dispatch failed",
				"station_id", stationID, "user_id", cand.UserID, "error", dispErr)
			continue
		}

		res.DispatchedSent += outcome.Sent
		res.DispatchedFailed += outcome.Failed
		res.DispatchedSkipped += outcome.Skipped
	}

	return res, nil
}

// dedupCandidatesByUser gộp nhiều location của cùng 1 user về đúng 1 candidate,
// ưu tiên giữ location có label "current" (nếu có) trước "home".
func dedupCandidatesByUser(rows []repository.StationAlertCandidate) []repository.StationAlertCandidate {
	if len(rows) == 0 {
		return nil
	}

	byUserIdx := make(map[string]int, len(rows))
	out := make([]repository.StationAlertCandidate, 0, len(rows))

	for _, r := range rows {
		idx, exists := byUserIdx[r.UserID]
		if !exists {
			byUserIdx[r.UserID] = len(out)
			out = append(out, r)
			continue
		}
		if out[idx].LocationLabel != "current" && r.LocationLabel == "current" {
			out[idx] = r
		}
	}

	return out
}

var releaseLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// RedisAlertLock triển khai AlertEvalLocker bằng Redis SET NX + Lua compare-and-delete.
type RedisAlertLock struct {
	rdb *redis.Client
}

func NewRedisAlertLock(rdb *redis.Client) *RedisAlertLock {
	return &RedisAlertLock{rdb: rdb}
}

func (l *RedisAlertLock) AcquireLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	return l.rdb.SetNX(ctx, key, token, ttl).Result()
}

func (l *RedisAlertLock) ReleaseLock(ctx context.Context, key, token string) error {
	return releaseLockScript.Run(ctx, l.rdb, []string{key}, token).Err()
}
