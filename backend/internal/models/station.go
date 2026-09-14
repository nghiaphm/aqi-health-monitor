package models

import (
	"database/sql"
)

// Station ánh xạ bảng stations — trạm quan trắc đồng bộ từ WAQI.
// Latitude/Longitude được repository đọc từ cột geom (PostGIS) qua ST_Y/ST_X.
type Station struct {
	ID           string         `db:"id"`
	WAQIStationID string        `db:"waqi_station_id"`
	Name         string         `db:"name"`
	Latitude     float64        `db:"latitude"`
	Longitude    float64        `db:"longitude"`
	City         sql.NullString `db:"city"`
	IsActive     bool           `db:"is_active"`
	LastSyncedAt sql.NullTime   `db:"last_synced_at"`
}
