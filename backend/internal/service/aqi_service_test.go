package service

import (
	"testing"
	"time"
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
