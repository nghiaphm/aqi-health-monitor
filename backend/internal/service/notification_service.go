package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"

	"aqi-health-monitor/backend/internal/models"
)

const defaultTransientRetryDelay = 1 * time.Second

var errPermanentDelivery = errors.New("permanent delivery error")

type notificationPushRepo interface {
	FindActiveByUserID(ctx context.Context, userID string) ([]models.PushSubscription, error)
	MarkFailed(ctx context.Context, id string, failedAt time.Time) error
}

type notificationAlertLogRepo interface {
	Insert(ctx context.Context, entry models.AlertLog) (*models.AlertLog, error)
}

// WebPushSender gửi thông báo đẩy tới 1 subscription trình duyệt và trả HTTP status code.
type WebPushSender interface {
	Send(ctx context.Context, sub models.PushSubscription, payload []byte) (int, error)
}

// EmailSender gửi email cảnh báo và trả HTTP status code từ provider.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody, textBody string) (int, error)
}

// NotificationService triển khai AlertDispatcher: gửi độc lập Web Push & Email,
// áp dụng Phương án A (retry tức thời tối đa 1 lần cho lỗi tạm thời),
// soft-disable subscription khi gặp 404/410 và ghi 1 dòng alerts_log mỗi attempt.
type NotificationService struct {
	pushRepo   notificationPushRepo
	alertLogs  notificationAlertLogRepo
	webPush    WebPushSender
	email      EmailSender
	retryDelay time.Duration
	nowFunc    func() time.Time
}

func NewNotificationService(
	pushRepo notificationPushRepo,
	alertLogs notificationAlertLogRepo,
	webPush WebPushSender,
	email EmailSender,
) *NotificationService {
	return &NotificationService{
		pushRepo:   pushRepo,
		alertLogs:  alertLogs,
		webPush:    webPush,
		email:      email,
		retryDelay: defaultTransientRetryDelay,
		nowFunc:    time.Now,
	}
}

func (s *NotificationService) SetNowFunc(f func() time.Time) {
	s.nowFunc = f
}

func (s *NotificationService) SetRetryDelay(d time.Duration) {
	s.retryDelay = d
}

func (s *NotificationService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc().UTC()
	}
	return time.Now().UTC()
}

// Dispatch gửi cảnh báo độc lập qua Web Push và Email rồi ghi alerts_log cho từng attempt.
func (s *NotificationService) Dispatch(ctx context.Context, in AlertDispatchInput) (AlertDispatchOutcome, error) {
	var outcome AlertDispatchOutcome
	level := AQILevel(in.Reading.AQI)
	advice := AQIAdvice(level)
	levelLabel := aqiLevelVNLabel(level)

	s.dispatchWebPush(ctx, in, levelLabel, advice, &outcome)
	s.dispatchEmail(ctx, in, levelLabel, advice, &outcome)

	return outcome, nil
}

func (s *NotificationService) dispatchWebPush(
	ctx context.Context,
	in AlertDispatchInput,
	levelLabel string,
	advice Advice,
	outcome *AlertDispatchOutcome,
) {
	if s.pushRepo == nil || s.webPush == nil {
		outcome.Skipped++
		return
	}

	subs, err := s.pushRepo.FindActiveByUserID(ctx, in.UserID)
	if err != nil {
		slog.Warn("notification: load active push subscriptions failed", "user_id", in.UserID, "error", err)
		outcome.Skipped++
		return
	}
	// Push subscription không có / đều inactive → skip, không ghi alerts_log failed.
	if len(subs) == 0 {
		outcome.Skipped++
		return
	}

	payload, err := buildWebPushPayload(in, levelLabel, advice)
	if err != nil {
		slog.Warn("notification: marshal web push payload failed", "user_id", in.UserID, "error", err)
		outcome.Skipped++
		return
	}

	for _, sub := range subs {
		statusCode, sendErr := s.sendWithTransientRetry(ctx, func() (int, error) {
			return s.webPush.Send(ctx, sub, payload)
		})

		status := models.AlertStatusSent
		if sendErr != nil {
			status = models.AlertStatusFailed
			outcome.Failed++
			slog.Warn("notification: web push send failed",
				"user_id", in.UserID, "sub_id", sub.ID, "status_code", statusCode, "error", sendErr)

			if statusCode == http.StatusNotFound || statusCode == http.StatusGone {
				if markErr := s.pushRepo.MarkFailed(ctx, sub.ID, s.now()); markErr != nil {
					slog.Warn("notification: mark push subscription failed",
						"sub_id", sub.ID, "error", markErr)
				}
			}
		} else {
			outcome.Sent++
		}

		s.recordAlertLog(ctx, in, models.AlertChannelWebPush, status)
	}
}

func (s *NotificationService) dispatchEmail(
	ctx context.Context,
	in AlertDispatchInput,
	levelLabel string,
	advice Advice,
	outcome *AlertDispatchOutcome,
) {
	if s.email == nil || strings.TrimSpace(in.UserEmail) == "" {
		outcome.Skipped++
		return
	}

	subject, htmlBody, textBody := buildEmailContent(in, levelLabel, advice)
	statusCode, sendErr := s.sendWithTransientRetry(ctx, func() (int, error) {
		return s.email.Send(ctx, in.UserEmail, subject, htmlBody, textBody)
	})

	status := models.AlertStatusSent
	if sendErr != nil {
		status = models.AlertStatusFailed
		outcome.Failed++
		slog.Warn("notification: email send failed",
			"user_id", in.UserID, "status_code", statusCode, "error", sendErr)
	} else {
		outcome.Sent++
	}

	s.recordAlertLog(ctx, in, models.AlertChannelEmail, status)
}

func (s *NotificationService) recordAlertLog(ctx context.Context, in AlertDispatchInput, channel, status string) {
	if s.alertLogs == nil {
		return
	}
	entry := models.AlertLog{
		UserID:     in.UserID,
		StationID:  in.Station.ID,
		LocationID: sql.NullString{String: in.LocationID, Valid: in.LocationID != ""},
		AQIValue:   in.Reading.AQI,
		Channel:    channel,
		SentAt:     s.now(),
		Status:     status,
	}
	if _, err := s.alertLogs.Insert(ctx, entry); err != nil {
		slog.Warn("notification: insert alerts_log failed",
			"user_id", in.UserID, "station_id", in.Station.ID, "channel", channel, "status", status, "error", err)
	}
}

// sendWithTransientRetry áp dụng Phương án A: chỉ retry tức thời tối đa 1 lần cho lỗi tạm thời
// (network/timeout hoặc HTTP >= 500 / 408 / 429). Không retry 4xx (như 404/410) hoặc lỗi cấu hình.
func (s *NotificationService) sendWithTransientRetry(
	ctx context.Context,
	fn func() (int, error),
) (int, error) {
	statusCode, err := fn()
	if err == nil || !isTransientDeliveryError(statusCode, err) {
		return statusCode, err
	}

	if s.retryDelay > 0 {
		timer := time.NewTimer(s.retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return statusCode, ctx.Err()
		case <-timer.C:
		}
	}

	return fn()
}

func isTransientDeliveryError(statusCode int, err error) bool {
	if err == nil || errors.Is(err, errPermanentDelivery) || errors.Is(err, context.Canceled) {
		return false
	}
	if statusCode == 0 {
		return true
	}
	return statusCode >= http.StatusInternalServerError ||
		statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests
}

func aqiLevelVNLabel(level string) string {
	switch level {
	case "good":
		return "Tốt"
	case "moderate":
		return "Trung bình"
	case "unhealthy_sensitive":
		return "Kém"
	case "unhealthy":
		return "Xấu"
	case "very_unhealthy":
		return "Rất xấu"
	default:
		return "Nguy hiểm"
	}
}

func buildWebPushPayload(in AlertDispatchInput, levelLabel string, advice Advice) ([]byte, error) {
	payload := map[string]string{
		"title": fmt.Sprintf("Cảnh báo AQI: %s đạt %d", in.Station.Name, in.Reading.AQI),
		"body": fmt.Sprintf(
			"Chỉ số AQI hiện tại là %d (%s), vượt ngưỡng %d của bạn. %s",
			in.Reading.AQI, levelLabel, in.ThresholdAQI, advice.Summary,
		),
		"url": "/dashboard",
		"tag": "aqi-alert-" + in.Station.ID,
	}
	return json.Marshal(payload)
}

func buildEmailContent(in AlertDispatchInput, levelLabel string, advice Advice) (string, string, string) {
	subject := fmt.Sprintf("[AQI Health Monitor] Cảnh báo AQI %d (%s) tại %s", in.Reading.AQI, levelLabel, in.Station.Name)

	greeting := "Xin chào,"
	if in.UserFullName != nil && strings.TrimSpace(*in.UserFullName) != "" {
		greeting = fmt.Sprintf("Xin chào %s,", strings.TrimSpace(*in.UserFullName))
	}

	var actionLines []string
	var actionHTML strings.Builder
	for _, a := range advice.Actions {
		actionLines = append(actionLines, "- "+a)
		actionHTML.WriteString("<li>")
		actionHTML.WriteString(html.EscapeString(a))
		actionHTML.WriteString("</li>")
	}

	textBody := fmt.Sprintf(
		"%s\n\nChỉ số chất lượng không khí tại trạm %s hiện là %d (%s), vượt ngưỡng bảo vệ %d của bạn.\n\n%s\n\nKhuyến nghị hành động:\n%s\n",
		greeting, in.Station.Name, in.Reading.AQI, levelLabel, in.ThresholdAQI, advice.Summary, strings.Join(actionLines, "\n"),
	)

	htmlBody := fmt.Sprintf(
		`<div style="font-family:sans-serif;line-height:1.5;color:#111827">`+
			`<p>%s</p>`+
			`<h2 style="margin:8px 0">Cảnh báo AQI đạt %d (%s)</h2>`+
			`<p>Trạm quan trắc gần bạn: <strong>%s</strong> (ngưỡng bảo vệ của bạn: <strong>%d</strong>).</p>`+
			`<p>%s</p>`+
			`<p><strong>Khuyến nghị hành động:</strong></p>`+
			`<ul>%s</ul>`+
			`</div>`,
		html.EscapeString(greeting),
		in.Reading.AQI,
		html.EscapeString(levelLabel),
		html.EscapeString(in.Station.Name),
		in.ThresholdAQI,
		html.EscapeString(advice.Summary),
		actionHTML.String(),
	)

	return subject, htmlBody, textBody
}

// ---------- Concrete Web Push (VAPID) & Email (Resend) Senders ----------

// VAPIDWebPushSender gửi Web Push chuẩn RFC 8291/8292 qua webpush-go.
type VAPIDWebPushSender struct {
	publicKey  string
	privateKey string
	subscriber string
	httpClient *http.Client
}

func NewVAPIDWebPushSender(publicKey, privateKey, subject string, httpClient *http.Client) *VAPIDWebPushSender {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	sub := strings.TrimSpace(subject)
	sub = strings.TrimPrefix(sub, "mailto:")
	return &VAPIDWebPushSender{
		publicKey:  strings.TrimSpace(publicKey),
		privateKey: strings.TrimSpace(privateKey),
		subscriber: sub,
		httpClient: httpClient,
	}
}

func (w *VAPIDWebPushSender) Send(ctx context.Context, sub models.PushSubscription, payload []byte) (int, error) {
	if w.publicKey == "" || w.privateKey == "" ||
		w.publicKey == "your_vapid_public_key_here" || w.privateKey == "your_vapid_private_key_here" {
		return 0, fmt.Errorf("%w: VAPID keys are not configured", errPermanentDelivery)
	}

	wpSub := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dhKey,
			Auth:   sub.AuthKey,
		},
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payload, wpSub, &webpush.Options{
		Subscriber:      w.subscriber,
		VAPIDPublicKey:  w.publicKey,
		VAPIDPrivateKey: w.privateKey,
		TTL:             3600,
		HTTPClient:      w.httpClient,
	})
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close()
			return resp.StatusCode, err
		}
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return resp.StatusCode, fmt.Errorf("web push status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.StatusCode, nil
}

// ResendEmailSender gửi email cảnh báo qua Resend HTTP API (/emails).
type ResendEmailSender struct {
	apiKey     string
	fromEmail  string
	baseURL    string
	httpClient *http.Client
}

func NewResendEmailSender(apiKey, fromEmail, baseURL string, httpClient *http.Client) *ResendEmailSender {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.resend.com"
	}
	return &ResendEmailSender{
		apiKey:     strings.TrimSpace(apiKey),
		fromEmail:  strings.TrimSpace(fromEmail),
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (r *ResendEmailSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) (int, error) {
	if r.apiKey == "" || r.apiKey == "your_resend_key_here" {
		return 0, fmt.Errorf("%w: RESEND_API_KEY is not configured", errPermanentDelivery)
	}

	bodyBytes, err := json.Marshal(map[string]any{
		"from":    r.fromEmail,
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
		"text":    textBody,
	})
	if err != nil {
		return 0, fmt.Errorf("%w: marshal email payload: %v", errPermanentDelivery, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, fmt.Errorf("%w: build resend request: %v", errPermanentDelivery, err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return resp.StatusCode, fmt.Errorf("resend status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return resp.StatusCode, nil
}
