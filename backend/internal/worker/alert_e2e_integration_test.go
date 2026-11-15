package worker

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/internal/service"
	"aqi-health-monitor/backend/pkg/waqi"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func inMemoryHTTPClient(h http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}
}

type memoryAlertLocker struct {
	mu    sync.Mutex
	locks map[string]string
}

func newMemoryAlertLocker() *memoryAlertLocker {
	return &memoryAlertLocker{locks: make(map[string]string)}
}

func (l *memoryAlertLocker) AcquireLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.locks[key]; exists {
		return false, nil
	}
	l.locks[key] = token
	return true, nil
}

func (l *memoryAlertLocker) ReleaseLock(ctx context.Context, key, token string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locks[key] == token {
		delete(l.locks, key)
	}
	return nil
}

type inlineCheckAlertsEnqueuer struct {
	handler   *CheckAlertsHandler
	calls     int
	lastErr   error
	stationID []string
}

func (e *inlineCheckAlertsEnqueuer) EnqueueCheckAlerts(ctx context.Context, stationID string) error {
	e.calls++
	e.stationID = append(e.stationID, stationID)
	task, err := NewCheckAlertsTask(stationID)
	if err != nil {
		e.lastErr = err
		return err
	}
	e.lastErr = e.handler.ProcessTask(ctx, task)
	return e.lastErr
}

// TestAlertFlow_E2E_IngestionToDelivery kiểm chứng end-to-end luồng:
// FetchReadings -> insert aqi_readings (inserted=true) -> enqueue check_alerts(station_id)
// -> CheckAlertsHandler -> AlertService (PostGIS nearest <= 20km + dedup current>home + threshold + cooldown)
// -> NotificationService (VAPID Web Push + Resend Email + 410 soft-disable) -> alerts_log.
func TestAlertFlow_E2E_IngestionToDelivery(t *testing.T) {
	dsn := os.Getenv("AQI_TEST_DSN")
	if dsn == "" {
		t.Skip("AQI_TEST_DSN not set; skipping real-DB alert E2E test")
	}

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	thresholdRepo := repository.NewAlertThresholdRepo(db)
	locationRepo := repository.NewLocationRepo(db)
	stationRepo := repository.NewStationRepo(db)
	readingRepo := repository.NewAQIReadingRepo(db)
	pushRepo := repository.NewPushSubscriptionRepo(db)
	alertLogRepo := repository.NewAlertLogRepo(db)

	// 1. Seed user, ngưỡng cảnh báo (100), và 2 vị trí đều gần st-near (~500m) để test dedup ưu tiên current.
	fullName := "Nguyễn Minh"
	u, err := userRepo.Create(ctx, "kc-e2e-user-1", "minh.e2e@example.com", &fullName)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := thresholdRepo.UpsertByUserID(ctx, db, u.ID, 100, true); err != nil {
		t.Fatalf("upsert threshold: %v", err)
	}
	hanoi := "Hanoi"
	homeLoc, err := locationRepo.UpsertByUserAndLabel(ctx, u.ID, "home", 21.0280, 105.8540, &hanoi)
	if err != nil {
		t.Fatalf("upsert home loc: %v", err)
	}
	currentLoc, err := locationRepo.UpsertByUserAndLabel(ctx, u.ID, "current", 21.0285, 105.8542, &hanoi)
	if err != nil {
		t.Fatalf("upsert current loc: %v", err)
	}
	_ = homeLoc

	// 2. Seed 2 trạm active: stNear cách ~50m, stFar cách ~30km (ngoài bán kính 20km).
	stNear, err := stationRepo.UpsertByWAQIID(ctx, "9101", "Hoan Kiem Station", 21.0289, 105.8545, &hanoi)
	if err != nil {
		t.Fatalf("upsert near station: %v", err)
	}
	stFar, err := stationRepo.UpsertByWAQIID(ctx, "9102", "Far Outskirts Station", 21.3000, 105.8545, &hanoi)
	if err != nil {
		t.Fatalf("upsert far station: %v", err)
	}
	_ = stFar

	// 3. Seed push subscriptions: subOK (201), subGone (410 soft-disable), subInactive (is_active=false -> skip).
	clientPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ecdh key: %v", err)
	}
	authBytes := make([]byte, 16)
	_, _ = rand.Read(authBytes)
	p256dh := base64.RawURLEncoding.EncodeToString(clientPriv.PublicKey().Bytes())
	auth := base64.RawURLEncoding.EncodeToString(authBytes)

	subOK, err := pushRepo.UpsertByEndpoint(ctx, u.ID, "https://push.test/ok", p256dh, auth)
	if err != nil {
		t.Fatalf("upsert subOK: %v", err)
	}
	subGone, err := pushRepo.UpsertByEndpoint(ctx, u.ID, "https://push.test/gone", p256dh, auth)
	if err != nil {
		t.Fatalf("upsert subGone: %v", err)
	}
	subInactive, err := pushRepo.UpsertByEndpoint(ctx, u.ID, "https://push.test/inactive", p256dh, auth)
	if err != nil {
		t.Fatalf("upsert subInactive: %v", err)
	}
	if _, err := pushRepo.DeactivateByUserAndEndpoint(ctx, u.ID, subInactive.Endpoint); err != nil {
		t.Fatalf("deactivate subInactive: %v", err)
	}

	// 4. Chuẩn bị WAQI mock feed (AQI=165 cho stNear, AQI=40 cho stFar), VAPID Web Push mock & Resend mock.
	now := time.Now().UTC().Truncate(time.Second)
	measuredAt := now.Add(-5 * time.Minute)
	waqiClient := waqi.NewClient("test-token", "https://waqi.test")
	waqiClient.SetHTTPClient(inMemoryHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aqi := 40
		if strings.Contains(r.URL.Path, "@9101") {
			aqi = 165
		}
		payload := map[string]any{
			"status": "ok",
			"data": map[string]any{
				"aqi":         aqi,
				"dominentpol": "pm25",
				"iaqi": map[string]any{
					"pm25": map[string]any{"v": 75.5},
				},
				"time": map[string]any{
					"v": measuredAt.Unix(),
				},
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	})))

	privVAPID, pubVAPID, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("vapid keys: %v", err)
	}

	var pushCalls, emailCalls int
	pushSender := service.NewVAPIDWebPushSender(pubVAPID, privVAPID, "mailto:alerts@test.local",
		inMemoryHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pushCalls++
			if strings.HasSuffix(r.URL.Path, "/gone") {
				w.WriteHeader(http.StatusGone)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})),
	)
	emailSender := service.NewResendEmailSender("re_test_key", "AQI <noreply@test.local>", "https://resend.test",
		inMemoryHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			emailCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"email_e2e_1"}`))
		})),
	)

	notifSvc := service.NewNotificationService(pushRepo, alertLogRepo, pushSender, emailSender)
	notifSvc.SetRetryDelay(0)
	alertSvc := service.NewAlertService(stationRepo, readingRepo, locationRepo, alertLogRepo, newMemoryAlertLocker(), notifSvc)

	handler := NewCheckAlertsHandler(alertSvc)
	enqueuer := &inlineCheckAlertsEnqueuer{handler: handler}

	aqiSvc := service.NewAQIService(waqiClient, stationRepo, readingRepo, nil, nil, locationRepo)
	aqiSvc.SetAlertEnqueuer(enqueuer)

	// 5. Lần FetchReadings thứ 1: insert readings mới → kích hoạt check_alerts cho cả 2 trạm.
	res1, err := aqiSvc.FetchReadings(ctx)
	if err != nil {
		t.Fatalf("fetch readings 1: %v", err)
	}
	if res1.Inserted != 2 || res1.AlertEnqueued != 2 || enqueuer.calls != 2 {
		t.Fatalf("expected 2 inserted & 2 enqueued, got res=%+v calls=%d", res1, enqueuer.calls)
	}
	if enqueuer.lastErr != nil {
		t.Fatalf("check_alerts handler returned error: %v", enqueuer.lastErr)
	}

	// Xác minh 1 Web Push OK, 1 Web Push 410, 1 Email OK (subInactive không gửi, không lỗi).
	if pushCalls != 2 || emailCalls != 1 {
		t.Fatalf("expected 2 push calls & 1 email call, got push=%d email=%d", pushCalls, emailCalls)
	}

	// Xác minh alerts_log ghi đúng 3 dòng: 2 web_push (sent, failed) + 1 email (sent), đều gắn currentLoc.ID.
	type logRow struct {
		StationID  string  `db:"station_id"`
		LocationID *string `db:"location_id"`
		AQIValue   int     `db:"aqi_value"`
		Channel    string  `db:"channel"`
		Status     string  `db:"status"`
	}
	var logs []logRow
	if err := db.SelectContext(ctx, &logs,
		`SELECT station_id, location_id, aqi_value, channel, status FROM alerts_log WHERE user_id = $1 ORDER BY channel ASC, status DESC`,
		u.ID,
	); err != nil {
		t.Fatalf("query alerts_log: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 alerts_log rows (web_push sent/failed + email sent), got %d: %+v", len(logs), logs)
	}
	for _, row := range logs {
		if row.StationID != stNear.ID || row.AQIValue != 165 {
			t.Errorf("unexpected station/aqi in alerts_log: %+v", row)
		}
		if row.LocationID == nil || *row.LocationID != currentLoc.ID {
			t.Errorf("expected dedup location_id=%s (current), got %v", currentLoc.ID, row.LocationID)
		}
	}

	// Xác minh subGone đã bị soft-disable kèm last_failed_at, còn subOK vẫn active.
	var subGoneActive, subOKActive bool
	var subGoneFailedAt *time.Time
	if err := db.QueryRowxContext(ctx, `SELECT is_active, last_failed_at FROM push_subscriptions WHERE id = $1`, subGone.ID).Scan(&subGoneActive, &subGoneFailedAt); err != nil {
		t.Fatalf("query subGone: %v", err)
	}
	if subGoneActive || subGoneFailedAt == nil {
		t.Fatalf("expected subGone soft-disabled with last_failed_at, got active=%v failedAt=%v", subGoneActive, subGoneFailedAt)
	}
	if err := db.QueryRowxContext(ctx, `SELECT is_active FROM push_subscriptions WHERE id = $1`, subOK.ID).Scan(&subOKActive); err != nil {
		t.Fatalf("query subOK: %v", err)
	}
	if !subOKActive {
		t.Fatalf("expected subOK to remain active")
	}

	// 6. Lần FetchReadings thứ 2 với cùng measuredAt: inserted=0 → KHÔNG enqueue check_alerts.
	res2, err := aqiSvc.FetchReadings(ctx)
	if err != nil {
		t.Fatalf("fetch readings 2: %v", err)
	}
	if res2.Inserted != 0 || res2.AlertEnqueued != 0 || enqueuer.calls != 2 {
		t.Fatalf("expected 0 inserted & 0 enqueued on duplicate reading, got res=%+v calls=%d", res2, enqueuer.calls)
	}

	// 7. Lần FetchReadings thứ 3 với reading mới hơn (+2 phút), vẫn trong cooldown 6h:
	// có enqueue check_alerts nhưng bị cooldown chặn → KHÔNG gửi thêm thông báo.
	measuredAt = now.Add(-2 * time.Minute)
	res3, err := aqiSvc.FetchReadings(ctx)
	if err != nil {
		t.Fatalf("fetch readings 3: %v", err)
	}
	if res3.Inserted != 2 || res3.AlertEnqueued != 2 {
		t.Fatalf("expected 2 inserted & 2 enqueued on newer reading, got res=%+v", res3)
	}
	if pushCalls != 2 || emailCalls != 1 {
		t.Fatalf("expected cooldown to prevent extra dispatches, got push=%d email=%d", pushCalls, emailCalls)
	}
	var logCount int
	if err := db.GetContext(ctx, &logCount, `SELECT count(*) FROM alerts_log WHERE user_id = $1`, u.ID); err != nil {
		t.Fatalf("count alerts_log: %v", err)
	}
	if logCount != 3 {
		t.Fatalf("expected alerts_log count to stay 3 under cooldown, got %d", logCount)
	}
}
