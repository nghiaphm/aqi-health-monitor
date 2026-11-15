package models

import (
	"database/sql"
	"time"
)

const (
	AlertChannelWebPush = "web_push"
	AlertChannelEmail   = "email"

	AlertStatusSent   = "sent"
	AlertStatusFailed = "failed"
)

// AlertLog ánh xạ bảng alerts_log (migrations 000009 + 000010).
type AlertLog struct {
	ID         string         `db:"id"`
	UserID     string         `db:"user_id"`
	StationID  string         `db:"station_id"`
	LocationID sql.NullString `db:"location_id"`
	AQIValue   int            `db:"aqi_value"`
	Channel    string         `db:"channel"`
	SentAt     time.Time      `db:"sent_at"`
	Status     string         `db:"status"`
}
