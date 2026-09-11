package handler

import (
	"encoding/json"
	"net/http"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/service"
)

type HealthProfileHandler struct {
	svc *service.HealthProfileService
}

func NewHealthProfileHandler(svc *service.HealthProfileService) *HealthProfileHandler {
	return &HealthProfileHandler{svc: svc}
}

// Upsert — POST /api/v1/health-profile. Shape response khớp 100% API-CONTRACT.md mục 4.2.
func (h *HealthProfileHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	claims, ok := authpkg.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
		return
	}

	var req upsertHealthProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Dữ liệu gửi lên không hợp lệ.",
			[]fieldError{{Field: "body", Reason: "invalid_json"}})
		return
	}

	var details []fieldError
	if !isValidConditionType(req.ConditionType) {
		details = append(details, fieldError{Field: "condition_type", Reason: "required; must be one of none|asthma|allergic_rhinitis"})
	}
	if !isValidAgeGroup(req.AgeGroup) {
		details = append(details, fieldError{Field: "age_group", Reason: "required; must be one of child|adult|elderly"})
	}

	sensitivity := "normal"
	if req.SensitivityLevel != nil {
		sensitivity = *req.SensitivityLevel
		if !isValidSensitivityLevel(sensitivity) {
			details = append(details, fieldError{Field: "sensitivity_level", Reason: "must be one of normal|high"})
		}
	}

	if req.CustomThresholdAQI != nil {
		v := *req.CustomThresholdAQI
		if v < 0 || v > 500 {
			details = append(details, fieldError{Field: "custom_threshold_aqi", Reason: "must be within [0,500]"})
		}
	}

	if len(details) > 0 {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Dữ liệu gửi lên không hợp lệ.", details)
		return
	}

	// Cấu trúc hợp lệ ([0,500]) nhưng < 50 → vi phạm nghiệp vụ, KHÔNG clamp (BUSINESS-RULES.md mục 1).
	if req.CustomThresholdAQI != nil && *req.CustomThresholdAQI < 50 {
		writeErrorWithDetails(w, http.StatusUnprocessableEntity, "THRESHOLD_TOO_LOW",
			"Ngưỡng cảnh báo không được dưới 50 (AQI dưới mức 'Trung bình' sẽ gây cảnh báo liên tục).",
			[]fieldError{{Field: "custom_threshold_aqi", Reason: "min=50"}})
		return
	}

	result, err := h.svc.UpsertProfile(r.Context(), claims, service.UpsertHealthProfileInput{
		ConditionType:      req.ConditionType,
		AgeGroup:           req.AgeGroup,
		SensitivityLevel:   sensitivity,
		CustomThresholdAQI: req.CustomThresholdAQI,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi hệ thống, vui lòng thử lại sau.")
		return
	}

	writeJSON(w, http.StatusOK, toUpsertHealthProfileResponse(result))
}

// ---------- request / response DTO (khớp API-CONTRACT.md mục 4.2) ----------

type upsertHealthProfileRequest struct {
	ConditionType      string  `json:"condition_type"`
	AgeGroup           string  `json:"age_group"`
	SensitivityLevel   *string `json:"sensitivity_level"`
	CustomThresholdAQI *int    `json:"custom_threshold_aqi"`
}

// upsertHealthProfileResponse tái dùng healthProfileResponse/alertThresholdResponse
// đã định nghĩa ở user_handler.go (cùng package) để tránh lệch shape giữa 2 endpoint.
type upsertHealthProfileResponse struct {
	healthProfileResponse
	AlertThreshold *alertThresholdResponse `json:"alert_threshold"`
}

type fieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func toUpsertHealthProfileResponse(result *service.HealthProfileResult) *upsertHealthProfileResponse {
	resp := &upsertHealthProfileResponse{
		healthProfileResponse: healthProfileResponse{
			ID:               result.Profile.ID,
			ConditionType:    result.Profile.ConditionType,
			AgeGroup:         result.Profile.AgeGroup,
			SensitivityLevel: result.Profile.SensitivityLevel,
			CreatedAt:        formatUTC(result.Profile.CreatedAt),
			UpdatedAt:        formatUTC(result.Profile.UpdatedAt),
		},
	}
	if result.Threshold != nil {
		resp.AlertThreshold = &alertThresholdResponse{
			ID:           result.Threshold.ID,
			ThresholdAQI: result.Threshold.ThresholdAQI,
			IsCustom:     result.Threshold.IsCustom,
			UpdatedAt:    formatUTC(result.Threshold.UpdatedAt),
		}
	}
	return resp
}

func isValidConditionType(v string) bool {
	switch v {
	case "none", "asthma", "allergic_rhinitis":
		return true
	default:
		return false
	}
}

func isValidAgeGroup(v string) bool {
	switch v {
	case "child", "adult", "elderly":
		return true
	default:
		return false
	}
}

func isValidSensitivityLevel(v string) bool {
	switch v {
	case "normal", "high":
		return true
	default:
		return false
	}
}

func writeErrorWithDetails(w http.ResponseWriter, status int, code, message string, details []fieldError) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}
