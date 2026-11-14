package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/service"
)

type mockPushSubscriptionService struct {
	upsertResult     *models.PushSubscription
	upsertErr        error
	deactivateResult *models.PushSubscription
	deactivateErr    error

	lastUpsertInput      service.UpsertPushSubscriptionInput
	lastDeactivateTarget string
}

func (m *mockPushSubscriptionService) UpsertSubscription(ctx context.Context, claims authpkg.Claims, in service.UpsertPushSubscriptionInput) (*models.PushSubscription, error) {
	m.lastUpsertInput = in
	return m.upsertResult, m.upsertErr
}

func (m *mockPushSubscriptionService) DeactivateSubscription(ctx context.Context, claims authpkg.Claims, endpoint string) (*models.PushSubscription, error) {
	m.lastDeactivateTarget = endpoint
	return m.deactivateResult, m.deactivateErr
}

func TestPushHandler_Unauthorized(t *testing.T) {
	h := NewPushHandler(&mockPushSubscriptionService{})

	for _, tc := range []struct {
		name   string
		method string
		call   func(http.ResponseWriter, *http.Request)
	}{
		{name: "upsert", method: http.MethodPost, call: h.Upsert},
		{name: "deactivate", method: http.MethodDelete, call: h.Deactivate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/push-subscriptions", strings.NewReader(`{}`))
			rec := httptest.NewRecorder()
			tc.call(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestPushHandler_Upsert_ValidationErrors(t *testing.T) {
	validP256dh := "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM"
	validAuth := "tBHItJI5svbpez7KI4CCXg"

	tests := []struct {
		name       string
		body       string
		wantFields []string
	}{
		{
			name:       "invalid json",
			body:       `{invalid`,
			wantFields: []string{"body"},
		},
		{
			name:       "missing all fields",
			body:       `{}`,
			wantFields: []string{"endpoint", "p256dh_key", "auth_key"},
		},
		{
			name:       "http endpoint rejected",
			body:       `{"endpoint":"http://example.com/push","p256dh_key":"` + validP256dh + `","auth_key":"` + validAuth + `"}`,
			wantFields: []string{"endpoint"},
		},
		{
			name:       "endpoint exceeds max length",
			body:       `{"endpoint":"https://example.com/` + strings.Repeat("a", 2040) + `","p256dh_key":"` + validP256dh + `","auth_key":"` + validAuth + `"}`,
			wantFields: []string{"endpoint"},
		},
		{
			name:       "invalid base64 keys and whitespace",
			body:       `{"endpoint":"https://fcm.googleapis.com/fcm/send/abc","p256dh_key":"not valid!","auth_key":"has space"}`,
			wantFields: []string{"p256dh_key", "auth_key"},
		},
		{
			name:       "key exceeds varchar 255",
			body:       `{"endpoint":"https://fcm.googleapis.com/fcm/send/abc","p256dh_key":"` + strings.Repeat("A", 256) + `","auth_key":"` + validAuth + `"}`,
			wantFields: []string{"p256dh_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewPushHandler(&mockPushSubscriptionService{})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/push-subscriptions", strings.NewReader(tt.body))
			req = req.WithContext(authpkg.ContextWithClaims(req.Context(), authpkg.Claims{Sub: "kc-1"}))
			rec := httptest.NewRecorder()

			h.Upsert(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (%s)", rec.Code, rec.Body.String())
			}

			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			errObj := body["error"].(map[string]any)
			if errObj["code"] != "VALIDATION_ERROR" {
				t.Fatalf("expected VALIDATION_ERROR, got %v", errObj["code"])
			}
			details := errObj["details"].([]any)
			gotFields := make(map[string]bool)
			for _, d := range details {
				gotFields[d.(map[string]any)["field"].(string)] = true
			}
			for _, wf := range tt.wantFields {
				if !gotFields[wf] {
					t.Errorf("expected field %q in details %v", wf, details)
				}
			}
		})
	}
}

func TestPushHandler_Upsert_Success(t *testing.T) {
	created := time.Date(2026, 11, 14, 14, 55, 0, 0, time.UTC)
	svc := &mockPushSubscriptionService{
		upsertResult: &models.PushSubscription{
			ID:           "d4e5f6a7-b8c9-4d0e-9f1a-2b3c4d5e6f70",
			UserID:       "user-1",
			Endpoint:     "https://fcm.googleapis.com/fcm/send/abc",
			P256dhKey:    "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM",
			AuthKey:      "tBHItJI5svbpez7KI4CCXg==",
			IsActive:     true,
			LastFailedAt: sql.NullTime{Valid: false},
			CreatedAt:    created,
		},
	}
	h := NewPushHandler(svc)

	payload := `{
		"endpoint": "  https://fcm.googleapis.com/fcm/send/abc  ",
		"p256dh_key": "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM",
		"auth_key": "tBHItJI5svbpez7KI4CCXg=="
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/push-subscriptions", strings.NewReader(payload))
	req = req.WithContext(authpkg.ContextWithClaims(req.Context(), authpkg.Claims{Sub: "kc-1"}))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if svc.lastUpsertInput.Endpoint != "https://fcm.googleapis.com/fcm/send/abc" {
		t.Errorf("expected trimmed endpoint passed to service, got %q", svc.lastUpsertInput.Endpoint)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] != "d4e5f6a7-b8c9-4d0e-9f1a-2b3c4d5e6f70" || resp["is_active"] != true {
		t.Errorf("unexpected response body: %v", resp)
	}
	if _, hasKey := resp["p256dh_key"]; hasKey {
		t.Errorf("response must not echo p256dh_key")
	}
	if _, hasAuth := resp["auth_key"]; hasAuth {
		t.Errorf("response must not echo auth_key")
	}
	if lastFailed, exists := resp["last_failed_at"]; !exists || lastFailed != nil {
		t.Errorf("expected last_failed_at=null, got %v (exists=%v)", lastFailed, exists)
	}
}

func TestPushHandler_Upsert_InternalError(t *testing.T) {
	svc := &mockPushSubscriptionService{upsertErr: errors.New("db down")}
	h := NewPushHandler(svc)

	payload := `{"endpoint":"https://fcm.googleapis.com/fcm/send/abc","p256dh_key":"BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM","auth_key":"tBHItJI5svbpez7KI4CCXg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/push-subscriptions", strings.NewReader(payload))
	req = req.WithContext(authpkg.ContextWithClaims(req.Context(), authpkg.Claims{Sub: "kc-1"}))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestPushHandler_Deactivate(t *testing.T) {
	failedAt := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 11, 14, 13, 0, 0, 0, time.UTC)

	t.Run("validation error", func(t *testing.T) {
		h := NewPushHandler(&mockPushSubscriptionService{})
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/push-subscriptions", strings.NewReader(`{"endpoint":"not-url"}`))
		req = req.WithContext(authpkg.ContextWithClaims(req.Context(), authpkg.Claims{Sub: "kc-1"}))
		rec := httptest.NewRecorder()

		h.Deactivate(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := NewPushHandler(&mockPushSubscriptionService{deactivateErr: service.ErrPushSubscriptionNotFound})
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/push-subscriptions", strings.NewReader(`{"endpoint":"https://fcm.googleapis.com/fcm/send/abc"}`))
		req = req.WithContext(authpkg.ContextWithClaims(req.Context(), authpkg.Claims{Sub: "kc-1"}))
		rec := httptest.NewRecorder()

		h.Deactivate(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
		var body map[string]map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["error"]["code"] != "SUBSCRIPTION_NOT_FOUND" {
			t.Errorf("expected SUBSCRIPTION_NOT_FOUND, got %v", body["error"]["code"])
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockPushSubscriptionService{
			deactivateResult: &models.PushSubscription{
				ID:           "sub-1",
				Endpoint:     "https://fcm.googleapis.com/fcm/send/abc",
				IsActive:     false,
				LastFailedAt: sql.NullTime{Time: failedAt, Valid: true},
				CreatedAt:    createdAt,
			},
		}
		h := NewPushHandler(svc)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/push-subscriptions", strings.NewReader(`{"endpoint":"https://fcm.googleapis.com/fcm/send/abc"}`))
		req = req.WithContext(authpkg.ContextWithClaims(req.Context(), authpkg.Claims{Sub: "kc-1"}))
		rec := httptest.NewRecorder()

		h.Deactivate(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["is_active"] != false {
			t.Errorf("expected is_active=false, got %v", resp["is_active"])
		}
		if resp["last_failed_at"] != "2026-11-14T14:00:00Z" {
			t.Errorf("unexpected last_failed_at: %v", resp["last_failed_at"])
		}
	})
}
