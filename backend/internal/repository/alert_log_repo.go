package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const alertLogColumns = `id, user_id, station_id, location_id, aqi_value, channel, sent_at, status`

type AlertLogRepo struct {
	db *sqlx.DB
}

func NewAlertLogRepo(db *sqlx.DB) *AlertLogRepo {
	return &AlertLogRepo{db: db}
}

// HasSentSince kiểm tra trong cooldown window (từ since tới hiện tại) đã có cảnh báo
// status='sent' cho cặp (user_id, station_id) hay chưa (dùng index idx_alerts_log_user_station_sent_at).
func (r *AlertLogRepo) HasSentSince(ctx context.Context, userID, stationID string, since time.Time) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS (
			SELECT 1
			FROM alerts_log
			WHERE user_id = $1
			  AND station_id = $2
			  AND status = 'sent'
			  AND sent_at > $3
		)`,
		userID, stationID, since.UTC(),
	)
	return exists, err
}

// Insert ghi 1 bản ghi kết quả gửi cảnh báo vào alerts_log.
func (r *AlertLogRepo) Insert(ctx context.Context, entry models.AlertLog) (*models.AlertLog, error) {
	var out models.AlertLog
	var err error
	if entry.SentAt.IsZero() {
		err = r.db.GetContext(ctx, &out,
			`INSERT INTO alerts_log (user_id, station_id, location_id, aqi_value, channel, status)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING `+alertLogColumns,
			entry.UserID, entry.StationID, entry.LocationID, entry.AQIValue, entry.Channel, entry.Status,
		)
	} else {
		err = r.db.GetContext(ctx, &out,
			`INSERT INTO alerts_log (user_id, station_id, location_id, aqi_value, channel, sent_at, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING `+alertLogColumns,
			entry.UserID, entry.StationID, entry.LocationID, entry.AQIValue, entry.Channel, entry.SentAt.UTC(), entry.Status,
		)
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}
