package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/service"
)

const maxPushEndpointLength = 2048
const maxPushKeyLength = 255

type pushSubscriptionService interface {
	UpsertSubscription(ctx context.Context, claims authpkg.Claims, in service.UpsertPushSubscriptionInput) (*models.PushSubscription, error)
	DeactivateSubscription(ctx context.Context, claims authpkg.Claims, endpoint string) (*models.PushSubscription, error)
}

type PushHandler struct {
	svc pushSubscriptionService
}

func NewPushHandler(svc pushSubscriptionService) *PushHandler {
	return &PushHandler{svc: svc}
}

// Upsert — POST /api/v1/push-subscriptions. Khớp API-CONTRACT.md mục 4.5.
func (h *PushHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	claims, ok := authpkg.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
		return
	}

	var req upsertPushSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Dữ liệu gửi lên không hợp lệ.",
			[]fieldError{{Field: "body", Reason: "invalid_json"}})
		return
	}

	var details []fieldError
	endpoint, endpointErr := validatePushEndpoint(req.Endpoint)
	if endpointErr != nil {
		details = append(details, *endpointErr)
	}
	p256dh, p256dhErr := validatePushKey("p256dh_key", req.P256dhKey)
	if p256dhErr != nil {
		details = append(details, *p256dhErr)
	}
	authKey, authErr := validatePushKey("auth_key", req.AuthKey)
	if authErr != nil {
		details = append(details, *authErr)
	}

	if len(details) > 0 {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Dữ liệu gửi lên không hợp lệ.", details)
		return
	}

	sub, err := h.svc.UpsertSubscription(r.Context(), claims, service.UpsertPushSubscriptionInput{
		Endpoint:  endpoint,
		P256dhKey: p256dh,
		AuthKey:   authKey,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi hệ thống, vui lòng thử lại sau.")
		return
	}

	writeJSON(w, http.StatusOK, toPushSubscriptionResponse(sub))
}

// Deactivate — DELETE /api/v1/push-subscriptions. Khớp API-CONTRACT.md mục 4.6.
func (h *PushHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	claims, ok := authpkg.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
		return
	}

	var req deactivatePushSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Dữ liệu gửi lên không hợp lệ.",
			[]fieldError{{Field: "body", Reason: "invalid_json"}})
		return
	}

	endpoint, endpointErr := validatePushEndpoint(req.Endpoint)
	if endpointErr != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Dữ liệu gửi lên không hợp lệ.",
			[]fieldError{*endpointErr})
		return
	}

	sub, err := h.svc.DeactivateSubscription(r.Context(), claims, endpoint)
	if err != nil {
		if errors.Is(err, service.ErrPushSubscriptionNotFound) {
			writeError(w, http.StatusNotFound, "SUBSCRIPTION_NOT_FOUND", "Không tìm thấy đăng ký thông báo đẩy thuộc tài khoản của bạn.")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi hệ thống, vui lòng thử lại sau.")
		return
	}

	writeJSON(w, http.StatusOK, toPushSubscriptionResponse(sub))
}

func validatePushEndpoint(raw string) (string, *fieldError) {
	endpoint := strings.TrimSpace(raw)
	if endpoint == "" {
		return "", &fieldError{Field: "endpoint", Reason: "required"}
	}
	if len(endpoint) > maxPushEndpointLength {
		return "", &fieldError{Field: "endpoint", Reason: "max=2048 characters"}
	}
	if strings.ContainsAny(endpoint, " \t\r\n") {
		return "", &fieldError{Field: "endpoint", Reason: "must be a valid https URL"}
	}
	u, err := url.ParseRequestURI(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", &fieldError{Field: "endpoint", Reason: "must be a valid https URL"}
	}
	return endpoint, nil
}

func validatePushKey(field, raw string) (string, *fieldError) {
	if raw == "" {
		return "", &fieldError{Field: field, Reason: "required"}
	}
	if strings.ContainsAny(raw, " \t\r\n") {
		return "", &fieldError{Field: field, Reason: "must not contain whitespace"}
	}
	if len(raw) > maxPushKeyLength {
		return "", &fieldError{Field: field, Reason: "max=255 characters"}
	}
	if !isValidBase64Any(raw) {
		return "", &fieldError{Field: field, Reason: "must be valid base64 or base64url"}
	}
	return raw, nil
}

func isValidBase64Any(s string) bool {
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil && len(b) > 0 {
			return true
		}
	}
	return false
}

// ---------- request / response DTO (khớp API-CONTRACT.md mục 3.6, 4.5, 4.6) ----------

type upsertPushSubscriptionRequest struct {
	Endpoint  string `json:"endpoint"`
	P256dhKey string `json:"p256dh_key"`
	AuthKey   string `json:"auth_key"`
}

type deactivatePushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
}

type pushSubscriptionResponse struct {
	ID           string  `json:"id"`
	Endpoint     string  `json:"endpoint"`
	IsActive     bool    `json:"is_active"`
	LastFailedAt *string `json:"last_failed_at"`
	CreatedAt    string  `json:"created_at"`
}

func toPushSubscriptionResponse(sub *models.PushSubscription) *pushSubscriptionResponse {
	var lastFailedAt *string
	if sub.LastFailedAt.Valid {
		formatted := formatUTC(sub.LastFailedAt.Time)
		lastFailedAt = &formatted
	}
	return &pushSubscriptionResponse{
		ID:           sub.ID,
		Endpoint:     sub.Endpoint,
		IsActive:     sub.IsActive,
		LastFailedAt: lastFailedAt,
		CreatedAt:    formatUTC(sub.CreatedAt),
	}
}
