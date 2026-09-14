package worker

import (
	"context"

	"github.com/hibiken/asynq"

	"aqi-health-monitor/backend/internal/service"
)

const TypeFetchReadings = "fetch_readings"

func NewFetchReadingsTask() *asynq.Task {
	return asynq.NewTask(TypeFetchReadings, nil)
}

type FetchReadingsHandler struct {
	svc *service.AQIService
}

func NewFetchReadingsHandler(svc *service.AQIService) *FetchReadingsHandler {
	return &FetchReadingsHandler{svc: svc}
}

// ProcessTask fetch reading cho toàn bộ trạm active (lịch mỗi giờ ở Bước 8).
func (h *FetchReadingsHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	_, err := h.svc.FetchReadings(ctx)
	return err
}
