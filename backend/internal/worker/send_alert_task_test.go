package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"

	"aqi-health-monitor/backend/internal/service"
)

type mockCheckAlertsEvaluator struct {
	gotStationID string
	result       service.CheckAlertsResult
	err          error
}

func (m *mockCheckAlertsEvaluator) CheckAlerts(ctx context.Context, stationID string) (service.CheckAlertsResult, error) {
	m.gotStationID = stationID
	return m.result, m.err
}

func TestNewCheckAlertsTask(t *testing.T) {
	if _, err := NewCheckAlertsTask(""); err == nil {
		t.Fatalf("expected error for empty stationID")
	}

	task, err := NewCheckAlertsTask("station-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TypeCheckAlerts {
		t.Errorf("expected type %q, got %q", TypeCheckAlerts, task.Type())
	}
}

func TestCheckAlertsHandler_ProcessTask(t *testing.T) {
	t.Run("invalid json skips retry", func(t *testing.T) {
		h := NewCheckAlertsHandler(&mockCheckAlertsEvaluator{})
		err := h.ProcessTask(context.Background(), asynq.NewTask(TypeCheckAlerts, []byte("{bad")))
		if !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("expected SkipRetry, got %v", err)
		}
	})

	t.Run("empty station_id skips retry", func(t *testing.T) {
		h := NewCheckAlertsHandler(&mockCheckAlertsEvaluator{})
		err := h.ProcessTask(context.Background(), asynq.NewTask(TypeCheckAlerts, []byte(`{"station_id":""}`)))
		if !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("expected SkipRetry, got %v", err)
		}
	})

	t.Run("valid task delegates station_id", func(t *testing.T) {
		eval := &mockCheckAlertsEvaluator{}
		h := NewCheckAlertsHandler(eval)
		task, _ := NewCheckAlertsTask("st-42")
		if err := h.ProcessTask(context.Background(), task); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eval.gotStationID != "st-42" {
			t.Errorf("expected station_id st-42, got %q", eval.gotStationID)
		}
	})

	t.Run("infra error returns for retry", func(t *testing.T) {
		boom := errors.New("db unreachable")
		h := NewCheckAlertsHandler(&mockCheckAlertsEvaluator{err: boom})
		task, _ := NewCheckAlertsTask("st-42")
		err := h.ProcessTask(context.Background(), task)
		if !errors.Is(err, boom) {
			t.Fatalf("expected %v, got %v", boom, err)
		}
	})
}

type mockEnqueueClient struct {
	gotTask *asynq.Task
	err     error
}

func (m *mockEnqueueClient) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	m.gotTask = task
	return &asynq.TaskInfo{ID: "task-1"}, m.err
}

func TestAsynqAlertEnqueuer_EnqueueCheckAlerts(t *testing.T) {
	t.Run("ignores ErrDuplicateTask from Unique(10m)", func(t *testing.T) {
		mc := &mockEnqueueClient{err: asynq.ErrDuplicateTask}
		enq := &AsynqAlertEnqueuer{client: mc}
		if err := enq.EnqueueCheckAlerts(context.Background(), "st-99"); err != nil {
			t.Fatalf("expected nil on ErrDuplicateTask, got %v", err)
		}
		if mc.gotTask == nil || mc.gotTask.Type() != TypeCheckAlerts {
			t.Fatalf("expected task type %q", TypeCheckAlerts)
		}
	})

	t.Run("propagates non-duplicate enqueue error", func(t *testing.T) {
		boom := errors.New("redis down")
		enq := &AsynqAlertEnqueuer{client: &mockEnqueueClient{err: boom}}
		if err := enq.EnqueueCheckAlerts(context.Background(), "st-99"); !errors.Is(err, boom) {
			t.Fatalf("expected %v, got %v", boom, err)
		}
	})
}
