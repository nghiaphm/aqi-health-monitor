package models

import "time"

// AQIReading ánh xạ bảng aqi_readings (hypertable).
// PK composite (station_id, time) — dùng cho insert idempotent ON CONFLICT DO NOTHING.
type AQIReading struct {
	Time              time.Time `db:"time"`
	StationID         string    `db:"station_id"`
	AQI               int       `db:"aqi"`
	PM25              *float64  `db:"pm25"`
	PM10              *float64  `db:"pm10"`
	O3                *float64  `db:"o3"`
	NO2               *float64  `db:"no2"`
	SO2               *float64  `db:"so2"`
	CO                *float64  `db:"co"`
	DominantPollutant *string   `db:"dominant_pollutant"`
}
