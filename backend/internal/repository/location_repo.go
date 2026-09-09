package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

// locationColumns đọc tọa độ từ cột geom GEOGRAPHY(Point,4326) (lưu (lng,lat))
// qua ST_X/ST_Y và alias về latitude/longitude cho API.
const locationColumns = `id, user_id, label, ST_Y(geom::geometry)::float8 AS latitude, ST_X(geom::geometry)::float8 AS longitude, city, created_at`

type LocationRepo struct {
	db *sqlx.DB
}

func NewLocationRepo(db *sqlx.DB) *LocationRepo {
	return &LocationRepo{db: db}
}

func (r *LocationRepo) FindAllByUserID(ctx context.Context, userID string) ([]models.UserLocation, error) {
	var out []models.UserLocation
	err := r.db.SelectContext(ctx, &out,
		"SELECT "+locationColumns+" FROM user_locations WHERE user_id = $1 ORDER BY created_at ASC",
		userID,
	)
	return out, err
}
