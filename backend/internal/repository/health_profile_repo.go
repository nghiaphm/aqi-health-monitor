package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const healthProfileColumns = "id, user_id, condition_type, age_group, sensitivity_level, created_at, updated_at"

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
