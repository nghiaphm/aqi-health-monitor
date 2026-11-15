package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/pkg/waqi"
)

func TestAQILevel(t *testing.T) {
	cases := []struct {
		aqi  int
		want string
	}{
		{0, "good"},
		{50, "good"},
		{51, "moderate"},
		{100, "moderate"},
		{101, "unhealthy_sensitive"},
		{150, "unhealthy_sensitive"},
		{151, "unhealthy"},
		{200, "unhealthy"},
		{201, "very_unhealthy"},
		{300, "very_unhealthy"},
		{301, "hazardous"},
		{500, "hazardous"},
	}

	for _, c := range cases {
		got := AQILevel(c.aqi)
		if got != c.want {
			t.Errorf("AQILevel(%d) = %q, want %q", c.aqi, got, c.want)
		}
	}
}

func TestAQIAdvice(t *testing.T) {
	levels := []string{"good", "moderate", "unhealthy_sensitive", "unhealthy", "very_unhealthy", "hazardous"}

	for _, lvl := range levels {
		adv := AQIAdvice(lvl)
		if adv.Summary == "" {
			t.Errorf("AQIAdvice(%q).Summary is empty", lvl)
		}
		if len(adv.Actions) == 0 {
			t.Errorf("AQIAdvice(%q).Actions is empty", lvl)
		}
	}

	// Verify exact contract wording for unhealthy level (API Contract mục 4.4)
	unhealthyAdv := AQIAdvice("unhealthy")
	expectedSummary := "Chất lượng không khí ở mức Xấu — bắt đầu ảnh hưởng tới toàn bộ người dân, nhóm nhạy cảm bị ảnh hưởng nghiêm trọng hơn."
	if unhealthyAdv.Summary != expectedSummary {
		t.Errorf("unhealthy advice summary mismatch: got %q, want %q", unhealthyAdv.Summary, expectedSummary)
	}
}

func TestAQIService_StaleRule(t *testing.T) {
	fixedNow := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	svc := &AQIService{
		nowFunc: func() time.Time {
			return fixedNow
		},
	}

	// Reading 2 giờ trước -> KHÔNG stale
	freshTime := fixedNow.Add(-2 * time.Hour)
	if svc.now().Sub(freshTime) > 3*time.Hour {
		t.Errorf("reading 2h old should not be stale")
	}

	// Reading 3 giờ 1 phút trước -> STALE
	staleTime := fixedNow.Add(-3*time.Hour - 1*time.Minute)
	if svc.now().Sub(staleTime) <= 3*time.Hour {
		t.Errorf("reading > 3h old must be stale")
	}
}

type mockAQIWAQIClient struct {
	feeds map[string]*waqi.Feed
}

func (m *mockAQIWAQIClient) FetchStationsInBounds(ctx context.Context, lat1, lng1, lat2, lng2 float64) ([]waqi.BoundsStation, error) {
	return nil, nil
}

func (m *mockAQIWAQIClient) FetchStationFeed(ctx context.Context, waqiStationID string) (*waqi.Feed, error) {
	return m.feeds[waqiStationID], nil
}

type mockAQIStationStore struct {
	active []models.Station
}

func (m *mockAQIStationStore) UpsertByWAQIID(ctx context.Context, waqiStationID, name string, lat, lon float64, city *string) (*models.Station, error) {
	return nil, nil
}

func (m *mockAQIStationStore) ListActive(ctx context.Context) ([]models.Station, error) {
	return m.active, nil
}

func (m *mockAQIStationStore) FindNearestWithin(ctx context.Context, lat, lon, maxMeters float64) (*repository.NearestStation, error) {
	return nil, nil
}

type mockAQIReadingStore struct {
	insertedByStation map[string]bool
}

func (m *mockAQIReadingStore) Insert(ctx context.Context, rd models.AQIReading) (bool, error) {
	return m.insertedByStation[rd.StationID], nil
}

func (m *mockAQIReadingStore) GetLatest(ctx context.Context, stationID string) (*models.AQIReading, error) {
	return nil, nil
}

type mockAlertCheckEnqueuer struct {
	enqueued []string
	failOn   string
}

func (m *mockAlertCheckEnqueuer) EnqueueCheckAlerts(ctx context.Context, stationID string) error {
	if m.failOn == stationID {
		return errors.New("queue down")
	}
	m.enqueued = append(m.enqueued, stationID)
	return nil
}

func TestAQIService_FetchReadings_EnqueuesAlertOnlyWhenInserted(t *testing.T) {
	now := time.Date(2026, 11, 15, 6, 0, 0, 0, time.UTC)
	svc := &AQIService{
		waqi: &mockAQIWAQIClient{
			feeds: map[string]*waqi.Feed{
				"@1": {AQI: 160, HasAQI: true, MeasuredAt: now},
				"@2": {AQI: 140, HasAQI: true, MeasuredAt: now},
				"@3": {AQI: 130, HasAQI: true, MeasuredAt: now},
			},
		},
		stations: &mockAQIStationStore{
			active: []models.Station{
				{ID: "st-new", WAQIStationID: "@1", Name: "New Station"},
				{ID: "st-dup", WAQIStationID: "@2", Name: "Duplicate Reading Station"},
				{ID: "st-enq-err", WAQIStationID: "@3", Name: "Enqueue Error Station"},
			},
		},
		readings: &mockAQIReadingStore{
			insertedByStation: map[string]bool{
				"st-new":     true,
				"st-dup":     false,
				"st-enq-err": true,
			},
		},
	}

	enq := &mockAlertCheckEnqueuer{failOn: "st-enq-err"}
	svc.SetAlertEnqueuer(enq)

	res, err := svc.FetchReadings(context.Background())
	if err != nil {
		t.Fatalf("unexpected FetchReadings error: %v", err)
	}
	if res.Total != 3 || res.Success != 3 || res.Inserted != 2 || res.AlertEnqueued != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(enq.enqueued) != 1 || enq.enqueued[0] != "st-new" {
		t.Fatalf("expected only st-new enqueued, got %+v", enq.enqueued)
	}
}
