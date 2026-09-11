package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const healthProfileColumns = "id, user_id, condition_type, age_group, sensitivity_level, created_at, updated_at"

// querier cho phép repo dùng chung cho cả *sqlx.DB và *sqlx.Tx
// (để upsert health_profiles + alert_thresholds trong cùng 1 transaction).
type querier interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

type HealthProfileRepo struct {
	db *sqlx.DB
}

func NewHealthProfileRepo(db *sqlx.DB) *HealthProfileRepo {
	return &HealthProfileRepo{db: db}
}

// FindByUserID trả hồ sơ sức khỏe của 1 user. Không có → sql.ErrNoRows.
func (r *HealthProfileRepo) FindByUserID(ctx context.Context, userID string) (*models.HealthProfile, error) {
	var p models.HealthProfile
	err := r.db.GetContext(ctx, &p, "SELECT "+healthProfileColumns+" FROM health_profiles WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertByUserID update-in-place theo user_id (DB có UNIQUE user_id, dùng ON CONFLICT).
// Nhận querier để chạy trong transaction do service mở.
func (r *HealthProfileRepo) UpsertByUserID(
	ctx context.Context,
	q querier,
	userID, conditionType, ageGroup, sensitivityLevel string,
) (*models.HealthProfile, error) {
	var p models.HealthProfile
	err := q.GetContext(ctx, &p,
		`INSERT INTO health_profiles (user_id, condition_type, age_group, sensitivity_level)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO UPDATE
		 SET condition_type = EXCLUDED.condition_type,
		     age_group = EXCLUDED.age_group,
		     sensitivity_level = EXCLUDED.sensitivity_level,
		     updated_at = now()
		 RETURNING `+healthProfileColumns,
		userID, conditionType, ageGroup, sensitivityLevel,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
