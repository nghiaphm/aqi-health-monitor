package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const alertThresholdColumns = "id, user_id, threshold_aqi, is_custom, created_at, updated_at"

type AlertThresholdRepo struct {
	db *sqlx.DB
}

func NewAlertThresholdRepo(db *sqlx.DB) *AlertThresholdRepo {
	return &AlertThresholdRepo{db: db}
}

// FindByUserID trả ngưỡng hiện đang hiệu lực của 1 user.
// MVP upsert 1 row/user nên LIMIT 1 chỉ là lưới an toàn. Không có → sql.ErrNoRows.
func (r *AlertThresholdRepo) FindByUserID(ctx context.Context, userID string) (*models.AlertThreshold, error) {
	var t models.AlertThreshold
	err := r.db.GetContext(ctx, &t,
		"SELECT "+alertThresholdColumns+" FROM alert_thresholds WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
