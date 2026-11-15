package repository

import (
	"context"
	"database/sql"

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

// FindByIDAndUserID trả về 1 vị trí của user theo id. Không tìm thấy trả về sql.ErrNoRows.
func (r *LocationRepo) FindByIDAndUserID(ctx context.Context, id, userID string) (*models.UserLocation, error) {
	var loc models.UserLocation
	err := r.db.GetContext(ctx, &loc,
		"SELECT "+locationColumns+" FROM user_locations WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// FindByUserAndLabel trả về vị trí của user theo nhãn (current | home). Không tìm thấy trả về sql.ErrNoRows.
func (r *LocationRepo) FindByUserAndLabel(ctx context.Context, userID, label string) (*models.UserLocation, error) {
	var loc models.UserLocation
	err := r.db.GetContext(ctx, &loc,
		"SELECT "+locationColumns+" FROM user_locations WHERE user_id = $1 AND label = $2",
		userID, label,
	)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// StationAlertCandidate là bản ghi vị trí user có trạm gần nhất trong <= 20km là stationID
// và user đã thiết lập alert_thresholds (chưa có threshold tự động bị loại bởi JOIN).
type StationAlertCandidate struct {
	LocationID    string         `db:"location_id"`
	UserID        string         `db:"user_id"`
	LocationLabel string         `db:"location_label"`
	UserEmail     string         `db:"user_email"`
	UserFullName  sql.NullString `db:"user_full_name"`
	ThresholdAQI  int            `db:"threshold_aqi"`
}

// FindNearestCandidatesByStation trả các vị trí user thỏa đồng thời:
//  1. Trạm active gần nhất trong bán kính maxMeters (20km) chính là stationID.
//  2. User đã có ít nhất 1 alert_thresholds đang hiệu lực.
//
// Kết quả sắp xếp ưu tiên label 'current' trước 'home' để service dedup xác định per-user.
func (r *LocationRepo) FindNearestCandidatesByStation(
	ctx context.Context,
	stationID string,
	maxMeters float64,
) ([]StationAlertCandidate, error) {
	var out []StationAlertCandidate
	query := `
		SELECT
			ul.id AS location_id,
			ul.user_id AS user_id,
			ul.label AS location_label,
			u.email AS user_email,
			u.full_name AS user_full_name,
			at.threshold_aqi AS threshold_aqi
		FROM user_locations ul
		JOIN users u ON u.id = ul.user_id
		JOIN LATERAL (
			SELECT t.threshold_aqi
			FROM alert_thresholds t
			WHERE t.user_id = ul.user_id
			ORDER BY t.updated_at DESC
			LIMIT 1
		) at ON true
		JOIN LATERAL (
			SELECT s.id
			FROM stations s
			WHERE s.is_active = true
			  AND ST_DWithin(s.geom, ul.geom, $2)
			ORDER BY ST_Distance(s.geom, ul.geom) ASC
			LIMIT 1
		) ns ON ns.id = $1
		ORDER BY ul.user_id, CASE WHEN ul.label = 'current' THEN 0 ELSE 1 END, ul.created_at ASC`
	err := r.db.SelectContext(ctx, &out, query, stationID, maxMeters)
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
