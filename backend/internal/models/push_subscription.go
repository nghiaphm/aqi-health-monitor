package models

import (
	"database/sql"
	"time"
)

// PushSubscription ánh xạ bảng push_subscriptions (migrations 000008 + 000011).
type PushSubscription struct {
	ID           string       `db:"id"`
	UserID       string       `db:"user_id"`
	Endpoint     string       `db:"endpoint"`
	P256dhKey    string       `db:"p256dh_key"`
	AuthKey      string       `db:"auth_key"`
	IsActive     bool         `db:"is_active"`
	LastFailedAt sql.NullTime `db:"last_failed_at"`
	CreatedAt    time.Time    `db:"created_at"`
}
