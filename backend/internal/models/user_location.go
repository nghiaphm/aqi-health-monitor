package models

import (
	"database/sql"
	"time"
)

// UserLocation ánh xạ bảng user_locations.
// Latitude/Longitude được repository đọc từ cột geom (PostGIS) qua ST_Y/ST_X.
type UserLocation struct {
	ID        string         `db:"id"`
	UserID    string         `db:"user_id"`
	Label     string         `db:"label"`
	Latitude  float64        `db:"latitude"`
	Longitude float64        `db:"longitude"`
	City      sql.NullString `db:"city"`
	CreatedAt time.Time      `db:"created_at"`
}
