package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"aqi-health-monitor/backend/internal/service"
)

const (
	TypeCheckAlerts       = "check_alerts"
	checkAlertsUniqueTTL  = 10 * time.Minute
	checkAlertsMaxRetries = 3
)

type CheckAlertsPayload struct {
	StationID string `json:"station_id"`
}

// NewCheckAlertsTask tạo task đánh giá cảnh báo cho 1 station, gắn Unique(10m) theo station_id
// để Asynq tự deduplicate nếu nhiều job check_alerts cùng trạm được enqueue chồng nhau.
func NewCheckAlertsTask(stationID string) (*asynq.Task, error) {
	if stationID == "" {
		return nil, fmt.Errorf("station_id is required")
	}
	payload, err := json.Marshal(CheckAlertsPayload{StationID: stationID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(
		TypeCheckAlerts,
		payload,
		asynq.Unique(checkAlertsUniqueTTL),
		asynq.MaxRetry(checkAlertsMaxRetries),
	), nil
}

type checkAlertsEvaluator interface {
	CheckAlerts(ctx context.Context, stationID string) (service.CheckAlertsResult, error)
}

type CheckAlertsHandler struct {
	svc checkAlertsEvaluator
}

func NewCheckAlertsHandler(svc checkAlertsEvaluator) *CheckAlertsHandler {
	return &CheckAlertsHandler{svc: svc}
}

func (h *CheckAlertsHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p CheckAlertsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil || p.StationID == "" {
		return fmt.Errorf("%w: invalid check_alerts payload", asynq.SkipRetry)
	}
	_, err := h.svc.CheckAlerts(ctx, p.StationID)
	return err
}

type taskEnqueueClient interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// AsynqAlertEnqueuer đẩy task check_alerts(station_id) vào hàng đợi Asynq.
// Nếu Redis đã giữ 1 task trùng trong cửa sổ Unique(10m), coi như đã deduplicate thành công.
type AsynqAlertEnqueuer struct {
	client taskEnqueueClient
}

func NewAsynqAlertEnqueuer(client *asynq.Client) *AsynqAlertEnqueuer {
	if client == nil {
		return &AsynqAlertEnqueuer{}
	}
	return &AsynqAlertEnqueuer{client: client}
}

func (e *AsynqAlertEnqueuer) EnqueueCheckAlerts(ctx context.Context, stationID string) error {
	if e.client == nil {
		return fmt.Errorf("asynq client is nil")
	}
	task, err := NewCheckAlertsTask(stationID)
	if err != nil {
		return err
	}
	if _, err := e.client.EnqueueContext(ctx, task); err != nil {
		if errors.Is(err, asynq.ErrDuplicateTask) {
			return nil
		}
		return err
	}
	return nil
}
