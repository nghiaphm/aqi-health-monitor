package service

import (
	"context"
	"database/sql"
	"errors"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
)

var ErrPushSubscriptionNotFound = errors.New("push subscription not found")

type pushUserSyncer interface {
	FindOrCreate(ctx context.Context, claims authpkg.Claims) (*models.User, error)
}

type pushSubscriptionStore interface {
	UpsertByEndpoint(ctx context.Context, userID, endpoint, p256dhKey, authKey string) (*models.PushSubscription, error)
	DeactivateByUserAndEndpoint(ctx context.Context, userID, endpoint string) (*models.PushSubscription, error)
}

// PushSubscriptionService quản lý đăng ký/tắt mềm Web Push subscription theo user.
type PushSubscriptionService struct {
	users pushUserSyncer
	subs  pushSubscriptionStore
}

func NewPushSubscriptionService(users pushUserSyncer, subs pushSubscriptionStore) *PushSubscriptionService {
	return &PushSubscriptionService{users: users, subs: subs}
}

// UpsertPushSubscriptionInput là dữ liệu subscription đã được handler validate.
type UpsertPushSubscriptionInput struct {
	Endpoint  string
	P256dhKey string
	AuthKey   string
}

// UpsertSubscription đảm bảo user tồn tại rồi upsert atomic theo endpoint.
func (s *PushSubscriptionService) UpsertSubscription(
	ctx context.Context,
	claims authpkg.Claims,
	in UpsertPushSubscriptionInput,
) (*models.PushSubscription, error) {
	user, err := s.users.FindOrCreate(ctx, claims)
	if err != nil {
		return nil, err
	}
	return s.subs.UpsertByEndpoint(ctx, user.ID, in.Endpoint, in.P256dhKey, in.AuthKey)
}

// DeactivateSubscription tắt mềm subscription thuộc user hiện tại.
func (s *PushSubscriptionService) DeactivateSubscription(
	ctx context.Context,
	claims authpkg.Claims,
	endpoint string,
) (*models.PushSubscription, error) {
	user, err := s.users.FindOrCreate(ctx, claims)
	if err != nil {
		return nil, err
	}
	sub, err := s.subs.DeactivateByUserAndEndpoint(ctx, user.ID, endpoint)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPushSubscriptionNotFound
		}
		return nil, err
	}
	return sub, nil
}
