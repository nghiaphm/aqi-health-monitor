package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

type AQIReadingRepo struct {
	db *sqlx.DB
}

func NewAQIReadingRepo(db *sqlx.DB) *AQIReadingRepo {
	return &AQIReadingRepo{db: db}
}

// Insert lưu 1 reading + cập nhật stations.last_synced_at trong CÙNG 1 transaction
// (chọn phương án gộp transaction: cùng commit hoặc cùng rollback).
// Dùng ON CONFLICT (station_id, time) DO NOTHING — chạy trùng không ghi đè lịch sử.
// Trả về inserted=false khi reading đã tồn tại (vẫn coi là fetch thành công nên
// last_synced_at vẫn được cập nhật).
func (r *AQIReadingRepo) Insert(ctx context.Context, rd models.AQIReading) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO aqi_readings
		     (time, station_id, aqi, pm25, pm10, o3, no2, so2, co, dominant_pollutant)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (station_id, time) DO NOTHING`,
		rd.Time, rd.StationID, rd.AQI, rd.PM25, rd.PM10, rd.O3, rd.NO2, rd.SO2, rd.CO, rd.DominantPollutant,
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE stations SET last_synced_at = now() WHERE id = $1`, rd.StationID,
	); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return rows > 0, nil
}
