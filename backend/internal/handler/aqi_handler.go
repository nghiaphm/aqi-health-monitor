package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/service"
)

type currentAQIService interface {
	GetCurrent(ctx context.Context, claims authpkg.Claims, locationID *string) (*service.CurrentAQIResult, error)
}

type AQIHandler struct {
	svc currentAQIService
}

func NewAQIHandler(svc currentAQIService) *AQIHandler {
	return &AQIHandler{svc: svc}
}

// GetCurrent — GET /api/v1/aqi/current. Shape response khớp 100% API-CONTRACT.md mục 4.4.
func (h *AQIHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	claims, ok := authpkg.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
		return
	}

	var locationID *string
	qLoc := strings.TrimSpace(r.URL.Query().Get("location_id"))
	if qLoc != "" {
		if _, err := uuid.Parse(qLoc); err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Dữ liệu gửi lên không hợp lệ.",
				[]fieldError{{Field: "location_id", Reason: "must be a valid UUID"}})
			return
		}
		locationID = &qLoc
	}

	result, err := h.svc.GetCurrent(r.Context(), claims, locationID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLocationNotFound):
			writeError(w, http.StatusNotFound, "LOCATION_NOT_FOUND", "Không tìm thấy vị trí yêu cầu hoặc bạn chưa lưu vị trí này.")
		case errors.Is(err, service.ErrNoStationCoverage):
			writeError(w, http.StatusNotFound, "NO_STATION_COVERAGE", "Không có trạm quan trắc nào hoạt động trong bán kính 20 km.")
		case errors.Is(err, service.ErrNoData):
			writeError(w, http.StatusNotFound, "NO_DATA", "Trạm quan trắc chưa có dữ liệu đo nào.")
		case errors.Is(err, service.ErrDataStale):
			writeError(w, http.StatusUnprocessableEntity, "DATA_STALE", "Dữ liệu đo đã quá cũ (hơn 3 giờ), không đủ tin cậy để hiển thị hiện tại.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi hệ thống, vui lòng thử lại sau.")
		}
		return
	}

	writeJSON(w, http.StatusOK, toCurrentAQIResponse(result))
}

// ---------- response DTO (khớp 100% API-CONTRACT.md mục 4.4) ----------

type currentAQIResponse struct {
	Location currentAQILocationResponse `json:"location"`
	Station  currentAQIStationResponse  `json:"station"`
	Reading  currentAQIReadingResponse  `json:"reading"`
	Advice   currentAQIAdviceResponse   `json:"advice"`
}

type currentAQILocationResponse struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      *string `json:"city"`
}

type currentAQIStationResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	City      *string `json:"city"`
	DistanceM int     `json:"distance_m"`
}

type currentAQIPollutantsResponse struct {
	PM25 *float64 `json:"pm25"`
	PM10 *float64 `json:"pm10"`
	O3   *float64 `json:"o3"`
	NO2  *float64 `json:"no2"`
	SO2  *float64 `json:"so2"`
	CO   *float64 `json:"co"`
}

type currentAQIReadingResponse struct {
	AQI               int                          `json:"aqi"`
	Level             string                       `json:"level"`
	MeasuredAt        string                       `json:"measured_at"`
	DominantPollutant *string                      `json:"dominant_pollutant"`
	Pollutants        currentAQIPollutantsResponse `json:"pollutants"`
}

type currentAQIAdviceResponse struct {
	Summary string   `json:"summary"`
	Actions []string `json:"actions"`
}

func toCurrentAQIResponse(res *service.CurrentAQIResult) currentAQIResponse {
	return currentAQIResponse{
		Location: currentAQILocationResponse{
			ID:        res.Location.ID,
			Label:     res.Location.Label,
			Latitude:  res.Location.Latitude,
			Longitude: res.Location.Longitude,
			City:      res.Location.City,
		},
		Station: currentAQIStationResponse{
			ID:        res.Station.ID,
			Name:      res.Station.Name,
			City:      res.Station.City,
			DistanceM: res.Station.DistanceM,
		},
		Reading: currentAQIReadingResponse{
			AQI:               res.Reading.AQI,
			Level:             res.Reading.Level,
			MeasuredAt:        formatUTC(res.Reading.MeasuredAt),
			DominantPollutant: res.Reading.DominantPollutant,
			Pollutants: currentAQIPollutantsResponse{
				PM25: res.Reading.Pollutants.PM25,
				PM10: res.Reading.Pollutants.PM10,
				O3:   res.Reading.Pollutants.O3,
				NO2:  res.Reading.Pollutants.NO2,
				SO2:  res.Reading.Pollutants.SO2,
				CO:   res.Reading.Pollutants.CO,
			},
		},
		Advice: currentAQIAdviceResponse{
			Summary: res.Advice.Summary,
			Actions: res.Advice.Actions,
		},
	}
}
