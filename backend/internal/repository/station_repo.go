package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const stationColumns = `id, waqi_station_id, name, ST_Y(geom::geometry)::float8 AS latitude, ST_X(geom::geometry)::float8 AS longitude, city, is_active, last_synced_at`

type StationRepo struct {
	db *sqlx.DB
}

func NewStationRepo(db *sqlx.DB) *StationRepo {
	return &StationRepo{db: db}
}

// UpsertByWAQIID đồng bộ trạm từ /map/bounds/ — idempotent theo waqi_station_id.
// last_synced_at chỉ set khi INSERT (lần discovery đầu); các lần fetch_readings
// sau sẽ cập nhật mốc này (xem AQIReadingRepo.Insert).
func (r *StationRepo) UpsertByWAQIID(
	ctx context.Context,
	waqiStationID, name string,
	latitude, longitude float64,
	city *string,
) (*models.Station, error) {
	var s models.Station
	err := r.db.GetContext(ctx, &s,
		`INSERT INTO stations (waqi_station_id, name, geom, city, is_active, last_synced_at)
		 VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, true, now())
		 ON CONFLICT (waqi_station_id) DO UPDATE
		 SET name = EXCLUDED.name,
		     geom = EXCLUDED.geom,
		     city = COALESCE(EXCLUDED.city, stations.city),
		     is_active = true
		 RETURNING `+stationColumns,
		waqiStationID, name, longitude, latitude, city,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListActive trả các trạm đang hoạt động để fetch_readings duyệt.
func (r *StationRepo) ListActive(ctx context.Context) ([]models.Station, error) {
	var out []models.Station
	err := r.db.SelectContext(ctx, &out,
		"SELECT "+stationColumns+" FROM stations WHERE is_active = true ORDER BY waqi_station_id",
	)
	return out, err
}
