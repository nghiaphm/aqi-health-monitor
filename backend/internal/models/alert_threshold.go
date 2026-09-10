package models

import "time"

// AlertThreshold ánh xạ bảng alert_thresholds.
// MVP: 1 user chỉ có đúng 1 ngưỡng đang hiệu lực (upsert, update-in-place).
type AlertThreshold struct {
	ID           string    `db:"id"`
	UserID       string    `db:"user_id"`
	ThresholdAQI int       `db:"threshold_aqi"`
	IsCustom     bool      `db:"is_custom"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
