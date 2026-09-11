package repository

import (
	"context"
	"database/sql"
	"errors"

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

// UpsertByUserID update-in-place ngưỡng đang hiệu lực của user (BUSINESS-RULES.md mục 0).
// Bảng alert_thresholds KHÔNG có UNIQUE user_id (chủ đích, để mở rộng lịch sử sau này),
// nên không dùng ON CONFLICT: UPDATE trước, nếu chưa có row thì INSERT.
// Nhận querier để chạy trong transaction do service mở.
func (r *AlertThresholdRepo) UpsertByUserID(
	ctx context.Context,
	q querier,
	userID string,
	thresholdAQI int,
	isCustom bool,
) (*models.AlertThreshold, error) {
	var t models.AlertThreshold
	err := q.GetContext(ctx, &t,
		`UPDATE alert_thresholds
		 SET threshold_aqi = $2, is_custom = $3, updated_at = now()
		 WHERE user_id = $1
		 RETURNING `+alertThresholdColumns,
		userID, thresholdAQI, isCustom,
	)
	if err == nil {
		return &t, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = q.GetContext(ctx, &t,
		`INSERT INTO alert_thresholds (user_id, threshold_aqi, is_custom)
		 VALUES ($1, $2, $3)
		 RETURNING `+alertThresholdColumns,
		userID, thresholdAQI, isCustom,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
