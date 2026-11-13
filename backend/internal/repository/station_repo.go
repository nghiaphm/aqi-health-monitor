package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const stationColumns = `id, waqi_station_id, name, ST_Y(geom::geometry)::float8 AS latitude, ST_X(geom::geometry)::float8 AS longitude, city, is_active, last_synced_at`

// NearestStation là trạm gần nhất được tìm thấy kèm khoảng cách tính theo mét.
type NearestStation struct {
	ID        string         `db:"id"`
	Name      string         `db:"name"`
	City      sql.NullString `db:"city"`
	DistanceM int            `db:"distance_m"`
}

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

// FindNearestWithin tìm trạm active gần nhất trong bán kính maxMeters (mét) tính từ (lat, lng).
// Tọa độ điểm user được truyền vào PostGIS dạng WGS84 ST_MakePoint(lng, lat).
// Trả về sql.ErrNoRows nếu không có trạm nào trong bán kính.
func (r *StationRepo) FindNearestWithin(ctx context.Context, lat, lng float64, maxMeters float64) (*NearestStation, error) {
	var ns NearestStation
	query := `
		SELECT 
			id, 
			name, 
			city, 
			ROUND(ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography))::int AS distance_m
		FROM stations
		WHERE is_active = true 
		  AND ST_DWithin(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) ASC
		LIMIT 1`
	err := r.db.GetContext(ctx, &ns, query, lng, lat, maxMeters)
	if err != nil {
		return nil, err
	}
	return &ns, nil
}
