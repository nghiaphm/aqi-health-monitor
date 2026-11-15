package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
)

type mockAlertStationReader struct {
	st  *models.Station
	err error
}

func (m *mockAlertStationReader) FindByID(ctx context.Context, id string) (*models.Station, error) {
	return m.st, m.err
}

type mockAlertReadingReader struct {
	rd  *models.AQIReading
	err error
}

func (m *mockAlertReadingReader) GetLatest(ctx context.Context, stationID string) (*models.AQIReading, error) {
	return m.rd, m.err
}

type mockAlertCandidateFinder struct {
	candidates []repository.StationAlertCandidate
	err        error
	gotMaxM    float64
}

func (m *mockAlertCandidateFinder) FindNearestCandidatesByStation(ctx context.Context, stationID string, maxMeters float64) ([]repository.StationAlertCandidate, error) {
	m.gotMaxM = maxMeters
	return m.candidates, m.err
}

type mockAlertCooldownChecker struct {
	sentByUser map[string]bool
	errByUser  map[string]error
	gotSince   time.Time
}

func (m *mockAlertCooldownChecker) HasSentSince(ctx context.Context, userID, stationID string, since time.Time) (bool, error) {
	m.gotSince = since
	if m.errByUser != nil {
		if err, ok := m.errByUser[userID]; ok {
			return false, err
		}
	}
	return m.sentByUser[userID], nil
}

type mockAlertLocker struct {
	acquired bool
	acqErr   error
	released bool
	gotKey   string
	gotToken string
}

func (m *mockAlertLocker) AcquireLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	m.gotKey = key
	m.gotToken = token
	return m.acquired, m.acqErr
}

func (m *mockAlertLocker) ReleaseLock(ctx context.Context, key, token string) error {
	if key == m.gotKey && token == m.gotToken {
		m.released = true
	}
	return nil
}

type mockAlertDispatcher struct {
	calls      []AlertDispatchInput
	errByUser  map[string]error
	outcomeMap map[string]AlertDispatchOutcome
}

func (m *mockAlertDispatcher) Dispatch(ctx context.Context, in AlertDispatchInput) (AlertDispatchOutcome, error) {
	m.calls = append(m.calls, in)
	if m.errByUser != nil {
		if err, ok := m.errByUser[in.UserID]; ok {
			return AlertDispatchOutcome{}, err
		}
	}
	if m.outcomeMap != nil {
		if o, ok := m.outcomeMap[in.UserID]; ok {
			return o, nil
		}
	}
	return AlertDispatchOutcome{Sent: 1}, nil
}

func TestAlertService_CheckAlerts_LockBehavior(t *testing.T) {
	t.Run("skipped when lock already held", func(t *testing.T) {
		locker := &mockAlertLocker{acquired: false}
		svc := NewAlertService(&mockAlertStationReader{}, &mockAlertReadingReader{}, &mockAlertCandidateFinder{}, &mockAlertCooldownChecker{}, locker, nil)

		res, err := svc.CheckAlerts(context.Background(), "st-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.SkippedLock {
			t.Fatalf("expected SkippedLock=true, got %+v", res)
		}
		if locker.released {
			t.Errorf("must not release lock that was not acquired")
		}
	})

	t.Run("acquire error triggers retry", func(t *testing.T) {
		boom := errors.New("redis down")
		locker := &mockAlertLocker{acqErr: boom}
		svc := NewAlertService(&mockAlertStationReader{}, &mockAlertReadingReader{}, &mockAlertCandidateFinder{}, &mockAlertCooldownChecker{}, locker, nil)

		_, err := svc.CheckAlerts(context.Background(), "st-1")
		if !errors.Is(err, boom) {
			t.Fatalf("expected error %v, got %v", boom, err)
		}
	})
}

func TestAlertService_CheckAlerts_StationAndDataGuards(t *testing.T) {
	now := time.Date(2026, 11, 15, 5, 0, 0, 0, time.UTC)

	t.Run("inactive station skipped", func(t *testing.T) {
		locker := &mockAlertLocker{acquired: true}
		svc := NewAlertService(
			&mockAlertStationReader{st: &models.Station{ID: "st-1", IsActive: false}},
			&mockAlertReadingReader{}, &mockAlertCandidateFinder{}, &mockAlertCooldownChecker{}, locker, nil,
		)
		res, err := svc.CheckAlerts(context.Background(), "st-1")
		if err != nil || !res.SkippedInactive || !locker.released {
			t.Fatalf("unexpected result: res=%+v err=%v released=%v", res, err, locker.released)
		}
	})

	t.Run("stale station (>3h) skipped", func(t *testing.T) {
		svc := NewAlertService(
			&mockAlertStationReader{st: &models.Station{
				ID:           "st-1",
				IsActive:     true,
				LastSyncedAt: sql.NullTime{Time: now.Add(-3*time.Hour - time.Minute), Valid: true},
			}},
			&mockAlertReadingReader{}, &mockAlertCandidateFinder{}, &mockAlertCooldownChecker{}, nil, nil,
		)
		svc.SetNowFunc(func() time.Time { return now })

		res, err := svc.CheckAlerts(context.Background(), "st-1")
		if err != nil || !res.SkippedStale {
			t.Fatalf("expected SkippedStale=true, got res=%+v err=%v", res, err)
		}
	})

	t.Run("no reading skipped", func(t *testing.T) {
		svc := NewAlertService(
			&mockAlertStationReader{st: &models.Station{
				ID:           "st-1",
				IsActive:     true,
				LastSyncedAt: sql.NullTime{Time: now.Add(-10 * time.Minute), Valid: true},
			}},
			&mockAlertReadingReader{err: sql.ErrNoRows},
			&mockAlertCandidateFinder{}, &mockAlertCooldownChecker{}, nil, nil,
		)
		svc.SetNowFunc(func() time.Time { return now })

		res, err := svc.CheckAlerts(context.Background(), "st-1")
		if err != nil || !res.SkippedNoData {
			t.Fatalf("expected SkippedNoData=true, got res=%+v err=%v", res, err)
		}
	})
}

func TestAlertService_CheckAlerts_ThresholdDedupCooldownIsolation(t *testing.T) {
	now := time.Date(2026, 11, 15, 5, 0, 0, 0, time.UTC)
	st := &models.Station{
		ID:           "st-1",
		Name:         "Cầu Giấy, Hà Nội",
		IsActive:     true,
		LastSyncedAt: sql.NullTime{Time: now.Add(-3 * time.Hour), Valid: true}, // Đúng mốc 3h: vẫn hợp lệ (chỉ >3h mới skip)
	}
	rd := &models.AQIReading{
		Time:      now.Add(-15 * time.Minute),
		StationID: "st-1",
		AQI:       120,
	}

	candidates := &mockAlertCandidateFinder{
		candidates: []repository.StationAlertCandidate{
			// User 1 có 2 locations (home trước, current sau) đều trỏ về st-1 → dedup giữ current, trigger vì 120 > 100
			{UserID: "u-1", LocationID: "loc-1-home", LocationLabel: "home", UserEmail: "u1@example.com", ThresholdAQI: 100},
			{UserID: "u-1", LocationID: "loc-1-cur", LocationLabel: "current", UserEmail: "u1@example.com", ThresholdAQI: 100},
			// User 2 có threshold == AQI (120 == 120) → KHÔNG vượt ngưỡng (quy tắc strictly >)
			{UserID: "u-2", LocationID: "loc-2-cur", LocationLabel: "current", UserEmail: "u2@example.com", ThresholdAQI: 120},
			// User 3 vượt ngưỡng nhưng đang trong cooldown 6h → skip
			{UserID: "u-3", LocationID: "loc-3-cur", LocationLabel: "current", UserEmail: "u3@example.com", ThresholdAQI: 80},
			// User 4 lỗi dispatch → được cô lập, không chặn User 5 và không làm fail cả job
			{UserID: "u-4", LocationID: "loc-4-cur", LocationLabel: "current", UserEmail: "u4@example.com", ThresholdAQI: 90},
			// User 5 vượt ngưỡng, dispatch thành công 2 attempts (1 sent, 1 failed)
			{UserID: "u-5", LocationID: "loc-5-home", LocationLabel: "home", UserEmail: "u5@example.com", ThresholdAQI: 119},
		},
	}

	cooldowns := &mockAlertCooldownChecker{
		sentByUser: map[string]bool{"u-3": true},
	}
	locker := &mockAlertLocker{acquired: true}
	dispatcher := &mockAlertDispatcher{
		errByUser: map[string]error{"u-4": errors.New("unexpected dispatch err")},
		outcomeMap: map[string]AlertDispatchOutcome{
			"u-1": {Sent: 1},
			"u-5": {Sent: 1, Failed: 1},
		},
	}

	svc := NewAlertService(
		&mockAlertStationReader{st: st},
		&mockAlertReadingReader{rd: rd},
		candidates,
		cooldowns,
		locker,
		dispatcher,
	)
	svc.SetNowFunc(func() time.Time { return now })

	res, err := svc.CheckAlerts(context.Background(), "st-1")
	if err != nil {
		t.Fatalf("expected nil job error when user error isolated, got %v", err)
	}
	if !locker.released {
		t.Errorf("expected lock to be released")
	}
	if candidates.gotMaxM != 20000.0 {
		t.Errorf("expected maxMeters=20000, got %v", candidates.gotMaxM)
	}
	if !cooldowns.gotSince.Equal(now.Add(-6 * time.Hour)) {
		t.Errorf("expected cooldown cutoff %v, got %v", now.Add(-6*time.Hour), cooldowns.gotSince)
	}
	if res.EvaluatedUsers != 5 || res.BelowThreshold != 1 || res.SkippedCooldown != 1 || res.Triggered != 3 {
		t.Fatalf("unexpected counts: %+v", res)
	}
	if res.UserErrors != 1 || res.DispatchedSent != 2 || res.DispatchedFailed != 1 {
		t.Fatalf("unexpected dispatch counts: %+v", res)
	}

	if len(dispatcher.calls) != 3 {
		t.Fatalf("expected 3 dispatch calls (u-1, u-4, u-5), got %d", len(dispatcher.calls))
	}
	if dispatcher.calls[0].UserID != "u-1" || dispatcher.calls[0].LocationID != "loc-1-cur" {
		t.Errorf("expected dedup to pick u-1 current location, got %+v", dispatcher.calls[0])
	}
}

func TestAlertService_CheckAlerts_InfraErrorPropagates(t *testing.T) {
	now := time.Date(2026, 11, 15, 5, 0, 0, 0, time.UTC)
	dbErr := errors.New("postgres timeout")
	svc := NewAlertService(
		&mockAlertStationReader{st: &models.Station{ID: "st-1", IsActive: true, LastSyncedAt: sql.NullTime{Time: now, Valid: true}}},
		&mockAlertReadingReader{rd: &models.AQIReading{StationID: "st-1", AQI: 150}},
		&mockAlertCandidateFinder{err: dbErr},
		&mockAlertCooldownChecker{},
		nil,
		nil,
	)
	svc.SetNowFunc(func() time.Time { return now })

	_, err := svc.CheckAlerts(context.Background(), "st-1")
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected infra error %v to propagate, got %v", dbErr, err)
	}
}
