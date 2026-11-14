package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
)

type mockPushUserSyncer struct {
	user *models.User
	err  error
}

func (m *mockPushUserSyncer) FindOrCreate(ctx context.Context, claims authpkg.Claims) (*models.User, error) {
	return m.user, m.err
}

type mockPushSubStore struct {
	sub          *models.PushSubscription
	upsertErr    error
	deactivate   *models.PushSubscription
	deactivateEr error

	gotUserID   string
	gotEndpoint string
	gotP256dh   string
	gotAuth     string
}

func (m *mockPushSubStore) UpsertByEndpoint(ctx context.Context, userID, endpoint, p256dhKey, authKey string) (*models.PushSubscription, error) {
	m.gotUserID = userID
	m.gotEndpoint = endpoint
	m.gotP256dh = p256dhKey
	m.gotAuth = authKey
	return m.sub, m.upsertErr
}

func (m *mockPushSubStore) DeactivateByUserAndEndpoint(ctx context.Context, userID, endpoint string) (*models.PushSubscription, error) {
	m.gotUserID = userID
	m.gotEndpoint = endpoint
	return m.deactivate, m.deactivateEr
}

func TestPushSubscriptionService_UpsertSubscription_Success(t *testing.T) {
	want := &models.PushSubscription{
		ID:        "sub-1",
		UserID:    "user-123",
		Endpoint:  "https://fcm.googleapis.com/fcm/send/abc",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	store := &mockPushSubStore{sub: want}
	svc := NewPushSubscriptionService(&mockPushUserSyncer{user: &models.User{ID: "user-123"}}, store)

	got, err := svc.UpsertSubscription(context.Background(), authpkg.Claims{Sub: "kc-1"}, UpsertPushSubscriptionInput{
		Endpoint:  "https://fcm.googleapis.com/fcm/send/abc",
		P256dhKey: "p256dh-val",
		AuthKey:   "auth-val",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if store.gotUserID != "user-123" || store.gotP256dh != "p256dh-val" || store.gotAuth != "auth-val" {
		t.Errorf("unexpected store arguments: %+v", store)
	}
}

func TestPushSubscriptionService_UpsertSubscription_SyncError(t *testing.T) {
	boom := errors.New("sync failed")
	svc := NewPushSubscriptionService(&mockPushUserSyncer{err: boom}, &mockPushSubStore{})

	_, err := svc.UpsertSubscription(context.Background(), authpkg.Claims{Sub: "kc-1"}, UpsertPushSubscriptionInput{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected error %v, got %v", boom, err)
	}
}

func TestPushSubscriptionService_DeactivateSubscription_NotFound(t *testing.T) {
	store := &mockPushSubStore{deactivateEr: sql.ErrNoRows}
	svc := NewPushSubscriptionService(&mockPushUserSyncer{user: &models.User{ID: "user-123"}}, store)

	_, err := svc.DeactivateSubscription(context.Background(), authpkg.Claims{Sub: "kc-1"}, "https://fcm.googleapis.com/fcm/send/missing")
	if !errors.Is(err, ErrPushSubscriptionNotFound) {
		t.Fatalf("expected ErrPushSubscriptionNotFound, got %v", err)
	}
}

func TestPushSubscriptionService_DeactivateSubscription_Success(t *testing.T) {
	want := &models.PushSubscription{
		ID:       "sub-1",
		UserID:   "user-123",
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		IsActive: false,
	}
	store := &mockPushSubStore{deactivate: want}
	svc := NewPushSubscriptionService(&mockPushUserSyncer{user: &models.User{ID: "user-123"}}, store)

	got, err := svc.DeactivateSubscription(context.Background(), authpkg.Claims{Sub: "kc-1"}, want.Endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsActive {
		t.Errorf("expected IsActive=false, got true")
	}
	if store.gotUserID != "user-123" || store.gotEndpoint != want.Endpoint {
		t.Errorf("unexpected store arguments: %+v", store)
	}
}
