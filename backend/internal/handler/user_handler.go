package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/service"
)

type UserHandler struct {
	syncer *service.UserSyncer
}

func NewUserHandler(syncer *service.UserSyncer) *UserHandler {
	return &UserHandler{syncer: syncer}
}

// GetMe — GET /api/v1/me. Shape response khớp 100% API-CONTRACT.md mục 4.1.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := authpkg.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
		return
	}

	state, err := h.syncer.GetUserState(r.Context(), claims)
	if err != nil {
		// 5xx — API-CONTRACT.md: không định nghĩa code riêng cho lỗi server.
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi hệ thống, vui lòng thử lại sau.")
		return
	}

	writeJSON(w, http.StatusOK, toUserStateResponse(state))
}

// ---------- response DTO (khớp field API-CONTRACT.md mục 4.1) ----------

type userStateResponse struct {
	ID             string                  `json:"id"`
	Email          string                  `json:"email"`
	FullName       *string                 `json:"full_name"`
	CreatedAt      string                  `json:"created_at"`
	HealthProfile  *healthProfileResponse  `json:"health_profile"`
	AlertThreshold *alertThresholdResponse `json:"alert_threshold"`
	Locations      []locationResponse      `json:"locations"`
}

type healthProfileResponse struct {
	ID               string `json:"id"`
	ConditionType    string `json:"condition_type"`
	AgeGroup         string `json:"age_group"`
	SensitivityLevel string `json:"sensitivity_level"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type alertThresholdResponse struct {
	ID           string `json:"id"`
	ThresholdAQI int    `json:"threshold_aqi"`
	IsCustom     bool   `json:"is_custom"`
	UpdatedAt    string `json:"updated_at"`
}

type locationResponse struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      *string `json:"city"`
	CreatedAt string  `json:"created_at"`
}

func toUserStateResponse(state *service.UserState) *userStateResponse {
	resp := &userStateResponse{
		ID:        state.User.ID,
		Email:     state.User.Email,
		FullName:  nullStringPtr(state.User.FullName),
		CreatedAt: formatUTC(state.User.CreatedAt),
		Locations: make([]locationResponse, 0, len(state.Locations)),
	}

	if state.HealthProfile != nil {
		resp.HealthProfile = &healthProfileResponse{
			ID:               state.HealthProfile.ID,
			ConditionType:    state.HealthProfile.ConditionType,
			AgeGroup:         state.HealthProfile.AgeGroup,
			SensitivityLevel: state.HealthProfile.SensitivityLevel,
			CreatedAt:        formatUTC(state.HealthProfile.CreatedAt),
			UpdatedAt:        formatUTC(state.HealthProfile.UpdatedAt),
		}
	}

	if state.AlertThreshold != nil {
		resp.AlertThreshold = &alertThresholdResponse{
			ID:           state.AlertThreshold.ID,
			ThresholdAQI: state.AlertThreshold.ThresholdAQI,
			IsCustom:     state.AlertThreshold.IsCustom,
			UpdatedAt:    formatUTC(state.AlertThreshold.UpdatedAt),
		}
	}

	for _, loc := range state.Locations {
		resp.Locations = append(resp.Locations, locationResponse{
			ID:        loc.ID,
			Label:     loc.Label,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			City:      nullStringPtr(loc.City),
			CreatedAt: formatUTC(loc.CreatedAt),
		})
	}

	return resp
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func formatUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
