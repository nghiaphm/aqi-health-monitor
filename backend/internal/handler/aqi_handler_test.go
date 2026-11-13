package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/service"
)

type mockAQIService struct {
	result *service.CurrentAQIResult
	err    error
}

func (m *mockAQIService) GetCurrent(ctx context.Context, claims authpkg.Claims, locationID *string) (*service.CurrentAQIResult, error) {
	return m.result, m.err
}

func TestAQIHandler_GetCurrent_Unauthorized(t *testing.T) {
	h := NewAQIHandler(&mockAQIService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/aqi/current", nil)
	rec := httptest.NewRecorder()

	h.GetCurrent(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"]["code"] != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %v", body["error"]["code"])
	}
}

func TestAQIHandler_GetCurrent_InvalidUUID(t *testing.T) {
	h := NewAQIHandler(&mockAQIService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/aqi/current?location_id=not-a-valid-uuid", nil)
	claims := authpkg.Claims{Sub: "user-1", Email: "test@example.com"}
	ctx := authpkg.ContextWithClaims(req.Context(), claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.GetCurrent(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok || errObj["code"] != "VALIDATION_ERROR" {
		t.Fatalf("expected code VALIDATION_ERROR, got %v", errObj["code"])
	}

	details, ok := errObj["details"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("expected 1 detail item, got %v", errObj["details"])
	}
	detailMap := details[0].(map[string]any)
	if detailMap["field"] != "location_id" {
		t.Errorf("expected detail field 'location_id', got %v", detailMap["field"])
	}
}

func TestAQIHandler_GetCurrent_ErrorMappings(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "location not found",
			serviceErr: service.ErrLocationNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "LOCATION_NOT_FOUND",
		},
		{
			name:       "no station coverage",
			serviceErr: service.ErrNoStationCoverage,
			wantStatus: http.StatusNotFound,
			wantCode:   "NO_STATION_COVERAGE",
		},
		{
			name:       "no data",
			serviceErr: service.ErrNoData,
			wantStatus: http.StatusNotFound,
			wantCode:   "NO_DATA",
		},
		{
			name:       "data stale",
			serviceErr: service.ErrDataStale,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "DATA_STALE",
		},
		{
			name:       "internal error",
			serviceErr: errors.New("db connection failure"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAQIService{err: tt.serviceErr}
			h := NewAQIHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/aqi/current", nil)
			claims := authpkg.Claims{Sub: "user-1", Email: "test@example.com"}
			req = req.WithContext(authpkg.ContextWithClaims(req.Context(), claims))

			rec := httptest.NewRecorder()
			h.GetCurrent(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			var body map[string]map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if body["error"]["code"] != tt.wantCode {
				t.Errorf("expected error code %q, got %q", tt.wantCode, body["error"]["code"])
			}
		})
	}
}

func TestAQIHandler_GetCurrent_Success(t *testing.T) {
	hanoi := "Hà Nội"
	dom := "pm25"
	pm25Val := 68.5
	pm10Val := 110.0
	measuredAt := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)

	mockResult := &service.CurrentAQIResult{
		Location: service.CurrentLocation{
			ID:        "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			Label:     "current",
			Latitude:  21.028511,
			Longitude: 105.854444,
			City:      &hanoi,
		},
		Station: service.CurrentStation{
			ID:        "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
			Name:      "Trạm Hoàn Kiếm",
			City:      &hanoi,
			DistanceM: 1420,
		},
		Reading: service.CurrentReading{
			AQI:               155,
			Level:             "unhealthy",
			MeasuredAt:        measuredAt,
			DominantPollutant: &dom,
			Pollutants: service.Pollutants{
				PM25: &pm25Val,
				PM10: &pm10Val,
				O3:   nil,
				NO2:  nil,
				SO2:  nil,
				CO:   nil,
			},
		},
		Advice: service.Advice{
			Summary: "Chất lượng không khí ở mức Xấu.",
			Actions: []string{"Đeo khẩu trang N95", "Đóng cửa sổ"},
		},
	}

	h := NewAQIHandler(&mockAQIService{result: mockResult})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/aqi/current", nil)
	claims := authpkg.Claims{Sub: "user-1", Email: "test@example.com"}
	req = req.WithContext(authpkg.ContextWithClaims(req.Context(), claims))

	rec := httptest.NewRecorder()
	h.GetCurrent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp currentAQIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Location.ID != mockResult.Location.ID {
		t.Errorf("location.id = %q, want %q", resp.Location.ID, mockResult.Location.ID)
	}
	if resp.Station.DistanceM != 1420 {
		t.Errorf("station.distance_m = %d, want 1420", resp.Station.DistanceM)
	}
	if resp.Reading.AQI != 155 {
		t.Errorf("reading.aqi = %d, want 155", resp.Reading.AQI)
	}
	if resp.Reading.Level != "unhealthy" {
		t.Errorf("reading.level = %q, want unhealthy", resp.Reading.Level)
	}
	if resp.Reading.MeasuredAt != "2026-09-07T10:00:00Z" {
		t.Errorf("reading.measured_at = %q, want 2026-09-07T10:00:00Z", resp.Reading.MeasuredAt)
	}
	if resp.Reading.Pollutants.PM25 == nil || *resp.Reading.Pollutants.PM25 != 68.5 {
		t.Errorf("pollutants.pm25 mismatch")
	}
	if resp.Reading.Pollutants.O3 != nil {
		t.Errorf("pollutants.o3 should be nil, got %v", resp.Reading.Pollutants.O3)
	}
	if len(resp.Advice.Actions) != 2 {
		t.Errorf("advice.actions length = %d, want 2", len(resp.Advice.Actions))
	}
}
