package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_PreflightAndAllowedOrigin(t *testing.T) {
	innerCalled := false
	h := CORS("http://localhost:3000, https://app.example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// 1. OPTIONS preflight returns 204 without calling inner handler
	reqOpt := httptest.NewRequest(http.MethodOptions, "/api/v1/push-subscriptions", nil)
	reqOpt.Header.Set("Origin", "http://localhost:3000")
	recOpt := httptest.NewRecorder()
	h.ServeHTTP(recOpt, reqOpt)

	if recOpt.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", recOpt.Code)
	}
	if recOpt.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("unexpected Allow-Origin: %q", recOpt.Header().Get("Access-Control-Allow-Origin"))
	}
	if innerCalled {
		t.Errorf("preflight OPTIONS should not invoke inner handler")
	}

	// 2. Disallowed origin does not receive Allow-Origin header
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/push-subscriptions", nil)
	reqBad.Header.Set("Origin", "https://evil.example.com")
	recBad := httptest.NewRecorder()
	h.ServeHTTP(recBad, reqBad)

	if recBad.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin must not receive Allow-Origin, got %q", recBad.Header().Get("Access-Control-Allow-Origin"))
	}
}
