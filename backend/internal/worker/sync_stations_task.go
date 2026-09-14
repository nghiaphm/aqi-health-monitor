package worker

import (
	"context"

	"github.com/hibiken/asynq"

	"aqi-health-monitor/backend/internal/service"
)

const TypeSyncStations = "sync_stations"

func NewSyncStationsTask() *asynq.Task {
	return asynq.NewTask(TypeSyncStations, nil)
}

type SyncStationsHandler struct {
	svc *service.AQIService
}

func NewSyncStationsHandler(svc *service.AQIService) *SyncStationsHandler {
	return &SyncStationsHandler{svc: svc}
}

// ProcessTask discovery + upsert trạm WAQI (thưa, không cần chạy mỗi giờ).
func (h *SyncStationsHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	_, err := h.svc.SyncStations(ctx)
	return err
}
