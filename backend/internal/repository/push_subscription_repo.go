package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const pushSubscriptionColumns = `id, user_id, endpoint, p256dh_key, auth_key, is_active, last_failed_at, created_at`

type PushSubscriptionRepo struct {
	db *sqlx.DB
}

func NewPushSubscriptionRepo(db *sqlx.DB) *PushSubscriptionRepo {
	return &PushSubscriptionRepo{db: db}
}

// UpsertByEndpoint upsert atomic theo UNIQUE(endpoint):
//   - Cùng user đăng ký lại: cập nhật keys, kích hoạt lại is_active=true và xóa last_failed_at.
//   - User khác trên cùng trình duyệt: reassign user_id về user hiện tại để tránh gửi nhầm cảnh báo.
func (r *PushSubscriptionRepo) UpsertByEndpoint(
	ctx context.Context,
	userID, endpoint, p256dhKey, authKey string,
) (*models.PushSubscription, error) {
	var sub models.PushSubscription
	err := r.db.GetContext(ctx, &sub,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh_key, auth_key, is_active, last_failed_at)
		 VALUES ($1, $2, $3, $4, true, NULL)
		 ON CONFLICT (endpoint) DO UPDATE
		 SET user_id = EXCLUDED.user_id,
		     p256dh_key = EXCLUDED.p256dh_key,
		     auth_key = EXCLUDED.auth_key,
		     is_active = true,
		     last_failed_at = NULL
		 RETURNING `+pushSubscriptionColumns,
		userID, endpoint, p256dhKey, authKey,
	)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// DeactivateByUserAndEndpoint tắt mềm subscription thuộc user hiện tại.
// Trả về sql.ErrNoRows nếu endpoint không tồn tại hoặc không thuộc userID.
func (r *PushSubscriptionRepo) DeactivateByUserAndEndpoint(
	ctx context.Context,
	userID, endpoint string,
) (*models.PushSubscription, error) {
	var sub models.PushSubscription
	err := r.db.GetContext(ctx, &sub,
		`UPDATE push_subscriptions
		 SET is_active = false
		 WHERE user_id = $1 AND endpoint = $2
		 RETURNING `+pushSubscriptionColumns,
		userID, endpoint,
	)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
