package service

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SherClockHolmes/webpush-go"

	"aqi-health-monitor/backend/internal/models"
)

type mockNotifPushRepo struct {
	subs         []models.PushSubscription
	findErr      error
	markedFailed []string
	markedTimes  []time.Time
}

func (m *mockNotifPushRepo) FindActiveByUserID(ctx context.Context, userID string) ([]models.PushSubscription, error) {
	return m.subs, m.findErr
}

func (m *mockNotifPushRepo) MarkFailed(ctx context.Context, id string, failedAt time.Time) error {
	m.markedFailed = append(m.markedFailed, id)
	m.markedTimes = append(m.markedTimes, failedAt)
	return nil
}

type mockNotifAlertLogRepo struct {
	entries []models.AlertLog
	err     error
}

func (m *mockNotifAlertLogRepo) Insert(ctx context.Context, entry models.AlertLog) (*models.AlertLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	entry.ID = "log-id"
	m.entries = append(m.entries, entry)
	return &entry, nil
}

type mockWebPushSender struct {
	fn    func(sub models.PushSubscription, payload []byte) (int, error)
	calls int
}

func (m *mockWebPushSender) Send(ctx context.Context, sub models.PushSubscription, payload []byte) (int, error) {
	m.calls++
	return m.fn(sub, payload)
}

type mockEmailSender struct {
	fn    func(to, subject, htmlBody, textBody string) (int, error)
	calls int
}

func (m *mockEmailSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) (int, error) {
	m.calls++
	return m.fn(to, subject, htmlBody, textBody)
}

func sampleDispatchInput() AlertDispatchInput {
	name := "Nguyễn Văn A"
	return AlertDispatchInput{
		UserID:        "user-1",
		UserEmail:     "user@example.com",
		UserFullName:  &name,
		Station:       &models.Station{ID: "st-1", Name: "Hoàn Kiếm, Hà Nội"},
		LocationID:    "loc-1",
		LocationLabel: "current",
		Reading:       &models.AQIReading{StationID: "st-1", AQI: 165},
		ThresholdAQI:  100,
	}
}

func TestNotificationService_Dispatch_DualChannelSuccess(t *testing.T) {
	now := time.Date(2026, 11, 15, 6, 0, 0, 0, time.UTC)
	pushRepo := &mockNotifPushRepo{
		subs: []models.PushSubscription{
			{ID: "sub-1", UserID: "user-1", Endpoint: "https://push.example/1", IsActive: true},
			{ID: "sub-2", UserID: "user-1", Endpoint: "https://push.example/2", IsActive: true},
		},
	}
	logRepo := &mockNotifAlertLogRepo{}
	var gotPayload []byte
	wp := &mockWebPushSender{
		fn: func(sub models.PushSubscription, payload []byte) (int, error) {
			gotPayload = payload
			return http.StatusCreated, nil
		},
	}
	em := &mockEmailSender{
		fn: func(to, subject, htmlBody, textBody string) (int, error) {
			if to != "user@example.com" || !strings.Contains(subject, "165") || !strings.Contains(htmlBody, "Hoàn Kiếm") {
				t.Errorf("unexpected email content: to=%s subject=%s", to, subject)
			}
			return http.StatusOK, nil
		},
	}

	svc := NewNotificationService(pushRepo, logRepo, wp, em)
	svc.SetRetryDelay(0)
	svc.SetNowFunc(func() time.Time { return now })

	out, err := svc.Dispatch(context.Background(), sampleDispatchInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Sent != 3 || out.Failed != 0 || out.Skipped != 0 {
		t.Fatalf("expected Sent=3 Failed=0 Skipped=0, got %+v", out)
	}
	if len(logRepo.entries) != 3 {
		t.Fatalf("expected 3 alerts_log entries (2 push attempts + 1 email), got %d", len(logRepo.entries))
	}

	var wpJSON map[string]string
	if err := json.Unmarshal(gotPayload, &wpJSON); err != nil {
		t.Fatalf("invalid push JSON payload: %v", err)
	}
	if wpJSON["url"] != "/dashboard" || wpJSON["tag"] != "aqi-alert-st-1" {
		t.Errorf("unexpected push payload fields: %+v", wpJSON)
	}
}

func TestNotificationService_Dispatch_NoActivePushSkipsWithoutFailedLog(t *testing.T) {
	pushRepo := &mockNotifPushRepo{subs: nil}
	logRepo := &mockNotifAlertLogRepo{}
	wp := &mockWebPushSender{
		fn: func(sub models.PushSubscription, payload []byte) (int, error) {
			t.Fatalf("must not send web push when no active subscriptions")
			return 0, nil
		},
	}
	em := &mockEmailSender{
		fn: func(to, subject, htmlBody, textBody string) (int, error) {
			return http.StatusOK, nil
		},
	}

	svc := NewNotificationService(pushRepo, logRepo, wp, em)
	svc.SetRetryDelay(0)

	out, err := svc.Dispatch(context.Background(), sampleDispatchInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Sent != 1 || out.Failed != 0 || out.Skipped != 1 {
		t.Fatalf("expected Sent=1 Failed=0 Skipped=1, got %+v", out)
	}
	if len(logRepo.entries) != 1 || logRepo.entries[0].Channel != models.AlertChannelEmail || logRepo.entries[0].Status != models.AlertStatusSent {
		t.Fatalf("expected only 1 email sent log entry, got %+v", logRepo.entries)
	}
}

func TestNotificationService_Dispatch_404And410SoftDisableNoRetry(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusGone} {
		now := time.Date(2026, 11, 15, 6, 30, 0, 0, time.UTC)
		pushRepo := &mockNotifPushRepo{
			subs: []models.PushSubscription{{ID: "sub-stale", UserID: "user-1", Endpoint: "https://push.example/gone", IsActive: true}},
		}
		logRepo := &mockNotifAlertLogRepo{}
		wp := &mockWebPushSender{
			fn: func(sub models.PushSubscription, payload []byte) (int, error) {
				return code, errors.New("gone")
			},
		}
		em := &mockEmailSender{
			fn: func(to, subject, htmlBody, textBody string) (int, error) {
				return http.StatusOK, nil
			},
		}

		svc := NewNotificationService(pushRepo, logRepo, wp, em)
		svc.SetRetryDelay(0)
		svc.SetNowFunc(func() time.Time { return now })

		out, err := svc.Dispatch(context.Background(), sampleDispatchInput())
		if err != nil {
			t.Fatalf("status %d: unexpected error: %v", code, err)
		}
		if wp.calls != 1 {
			t.Errorf("status %d: must NOT retry permanent 404/410, got %d calls", code, wp.calls)
		}
		if len(pushRepo.markedFailed) != 1 || pushRepo.markedFailed[0] != "sub-stale" || !pushRepo.markedTimes[0].Equal(now) {
			t.Errorf("status %d: expected MarkFailed('sub-stale', %v), got ids=%v times=%v", code, now, pushRepo.markedFailed, pushRepo.markedTimes)
		}
		if out.Sent != 1 || out.Failed != 1 {
			t.Errorf("status %d: expected partial failure Sent=1 Failed=1, got %+v", code, out)
		}
		if len(logRepo.entries) != 2 || logRepo.entries[0].Status != models.AlertStatusFailed || logRepo.entries[1].Status != models.AlertStatusSent {
			t.Errorf("status %d: unexpected logs: %+v", code, logRepo.entries)
		}
	}
}

func TestNotificationService_TransientRetryOptionA(t *testing.T) {
	t.Run("502 then 200 succeeds on single retry", func(t *testing.T) {
		pushRepo := &mockNotifPushRepo{subs: nil}
		logRepo := &mockNotifAlertLogRepo{}
		attempts := 0
		em := &mockEmailSender{
			fn: func(to, subject, htmlBody, textBody string) (int, error) {
				attempts++
				if attempts == 1 {
					return http.StatusBadGateway, errors.New("502 temporary")
				}
				return http.StatusOK, nil
			},
		}

		svc := NewNotificationService(pushRepo, logRepo, nil, em)
		svc.SetRetryDelay(0)

		out, _ := svc.Dispatch(context.Background(), sampleDispatchInput())
		if attempts != 2 || out.Sent != 1 || out.Failed != 0 {
			t.Fatalf("expected 2 attempts and Sent=1, got attempts=%d out=%+v", attempts, out)
		}
	})

	t.Run("503 twice stops after 1 retry and logs failed", func(t *testing.T) {
		pushRepo := &mockNotifPushRepo{subs: nil}
		logRepo := &mockNotifAlertLogRepo{}
		em := &mockEmailSender{
			fn: func(to, subject, htmlBody, textBody string) (int, error) {
				return http.StatusServiceUnavailable, errors.New("503 down")
			},
		}

		svc := NewNotificationService(pushRepo, logRepo, nil, em)
		svc.SetRetryDelay(0)

		out, _ := svc.Dispatch(context.Background(), sampleDispatchInput())
		if em.calls != 2 || out.Sent != 0 || out.Failed != 1 {
			t.Fatalf("expected 2 calls and Failed=1, got calls=%d out=%+v", em.calls, out)
		}
		if len(logRepo.entries) != 1 || logRepo.entries[0].Status != models.AlertStatusFailed {
			t.Fatalf("expected 1 failed log, got %+v", logRepo.entries)
		}
	})

	t.Run("400 bad request not retried", func(t *testing.T) {
		pushRepo := &mockNotifPushRepo{subs: nil}
		logRepo := &mockNotifAlertLogRepo{}
		em := &mockEmailSender{
			fn: func(to, subject, htmlBody, textBody string) (int, error) {
				return http.StatusBadRequest, errors.New("400 invalid")
			},
		}

		svc := NewNotificationService(pushRepo, logRepo, nil, em)
		svc.SetRetryDelay(0)

		out, _ := svc.Dispatch(context.Background(), sampleDispatchInput())
		if em.calls != 1 || out.Failed != 1 {
			t.Fatalf("expected 1 call without retry, got calls=%d out=%+v", em.calls, out)
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newInMemoryHTTPClient(h http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}
}

func TestResendEmailSender_HTTP(t *testing.T) {
	httpClient := newInMemoryHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/emails" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer re_test_key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["subject"] == "trigger-500" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))

	sender := NewResendEmailSender("re_test_key", "AQI <onboarding@resend.dev>", "https://resend.test", httpClient)
	code, err := sender.Send(context.Background(), "u@example.com", "Test", "<p>Hi</p>", "Hi")
	if err != nil || code != http.StatusOK {
		t.Fatalf("expected 200 OK, got code=%d err=%v", code, err)
	}

	code500, err500 := sender.Send(context.Background(), "u@example.com", "trigger-500", "<p>Hi</p>", "Hi")
	if err500 == nil || code500 != http.StatusInternalServerError {
		t.Fatalf("expected 500 error, got code=%d err=%v", code500, err500)
	}

	unconfigured := NewResendEmailSender("your_resend_key_here", "from", "https://resend.test", httpClient)
	_, permErr := unconfigured.Send(context.Background(), "u@example.com", "Test", "<p>Hi</p>", "Hi")
	if !errors.Is(permErr, errPermanentDelivery) {
		t.Fatalf("expected permanent delivery error when unconfigured, got %v", permErr)
	}
}

func TestVAPIDWebPushSender_HTTP(t *testing.T) {
	privVAPID, pubVAPID, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}

	// Tạo ECDH P-256 public key + auth secret giả lập trình duyệt thật để webpush-go mã hóa AES128GCM.
	clientPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	authBytes := make([]byte, 16)
	_, _ = rand.Read(authBytes)
	sub := models.PushSubscription{
		ID:        "sub-1",
		Endpoint:  "https://push.test/sub-1",
		P256dhKey: base64.RawURLEncoding.EncodeToString(clientPriv.PublicKey().Bytes()),
		AuthKey:   base64.RawURLEncoding.EncodeToString(authBytes),
	}

	var returnStatus int
	httpClient := newInMemoryHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" || r.Header.Get("Content-Encoding") != "aes128gcm" {
			t.Errorf("missing expected web push headers: auth=%q enc=%q", r.Header.Get("Authorization"), r.Header.Get("Content-Encoding"))
		}
		w.WriteHeader(returnStatus)
	}))

	sender := NewVAPIDWebPushSender(pubVAPID, privVAPID, "mailto:test@example.com", httpClient)

	returnStatus = http.StatusCreated
	code, sendErr := sender.Send(context.Background(), sub, []byte(`{"title":"AQI 160"}`))
	if sendErr != nil || code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got code=%d err=%v", code, sendErr)
	}

	returnStatus = http.StatusGone
	code410, err410 := sender.Send(context.Background(), sub, []byte(`{"title":"AQI 160"}`))
	if err410 == nil || code410 != http.StatusGone {
		t.Fatalf("expected 410 Gone error, got code=%d err=%v", code410, err410)
	}
}
