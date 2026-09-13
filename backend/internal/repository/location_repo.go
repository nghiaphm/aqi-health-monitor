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

// UpsertByUserAndLabel upsert ATOMIC theo (user_id, label) bằng ON CONFLICT
// (dựa trên constraint uq_user_locations_user_label, migration 000012) —
// tránh race condition khi 2 request đồng thời cùng user + label.
// ST_MakePoint nhận (lng, lat) theo đúng thứ tự WGS84 của PostGIS.
func (r *LocationRepo) UpsertByUserAndLabel(
	ctx context.Context,
	userID, label string,
	latitude, longitude float64,
	city *string,
) (*models.UserLocation, error) {
	var loc models.UserLocation
	err := r.db.GetContext(ctx, &loc,
		`INSERT INTO user_locations (user_id, label, geom, city)
		 VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5)
		 ON CONFLICT (user_id, label) DO UPDATE
		 SET geom = EXCLUDED.geom, city = EXCLUDED.city
		 RETURNING `+locationColumns,
		userID, label, longitude, latitude, city,
	)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}
