package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/service"
)

type LocationHandler struct {
	svc *service.LocationService
}

func NewLocationHandler(svc *service.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

// Upsert — POST /api/v1/locations. Shape response khớp 100% API-CONTRACT.md mục 4.3.
func (h *LocationHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	claims, ok := authpkg.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
		return
	}

	var req upsertLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Dữ liệu gửi lên không hợp lệ.",
			[]fieldError{{Field: "body", Reason: "invalid_json"}})
		return
	}

	var details []fieldError
	if req.Label != "current" && req.Label != "home" {
		details = append(details, fieldError{Field: "label", Reason: "required; must be one of current|home"})
	}
	if req.Latitude == nil {
		details = append(details, fieldError{Field: "latitude", Reason: "required"})
	} else if *req.Latitude < -90 || *req.Latitude > 90 {
		details = append(details, fieldError{Field: "latitude", Reason: "must be within [-90,90]"})
	}
	if req.Longitude == nil {
		details = append(details, fieldError{Field: "longitude", Reason: "required"})
	} else if *req.Longitude < -180 || *req.Longitude > 180 {
		details = append(details, fieldError{Field: "longitude", Reason: "must be within [-180,180]"})
	}

	var city *string
	if req.City != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.City))
		if len(normalized) > 100 {
			details = append(details, fieldError{Field: "city", Reason: "max=100 characters"})
		} else if normalized != "" {
			city = &normalized
		}
	}

	if len(details) > 0 {
		writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Dữ liệu gửi lên không hợp lệ.", details)
		return
	}

	loc, err := h.svc.UpsertByUserAndLabel(r.Context(), claims, service.UpsertLocationInput{
		Label:     req.Label,
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
		City:      city,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi hệ thống, vui lòng thử lại sau.")
		return
	}

	writeJSON(w, http.StatusOK, locationResponse{
		ID:        loc.ID,
		Label:     loc.Label,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		City:      nullStringPtr(loc.City),
		CreatedAt: formatUTC(loc.CreatedAt),
	})
}

// ---------- request DTO (khớp API-CONTRACT.md mục 4.3) ----------

type upsertLocationRequest struct {
	Label     string   `json:"label"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	City      *string  `json:"city"`
}
