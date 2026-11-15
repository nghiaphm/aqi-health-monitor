package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"

	"aqi-health-monitor/backend/internal/config"
	"aqi-health-monitor/backend/internal/database"
	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/internal/service"
	"aqi-health-monitor/backend/internal/worker"
	"aqi-health-monitor/backend/pkg/waqi"
)

func main() {
	// .env ở root repo (khi chạy go run từ backend/, root nằm ở ../.env).
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	task := flag.String("task", "serve", "serve | sync_stations | fetch_readings | check_alerts")
	stationID := flag.String("station_id", "", "station UUID dùng cho -task=check_alerts")
	flag.Parse()

	if err := run(*task, *stationID); err != nil {
		log.Fatal(err)
	}
}

func run(task, stationID string) error {
	cfg := config.Load()
	ctx := context.Background()

	db, err := database.OpenPostgres(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		return err
	}
	defer db.Close()

	rdb, err := database.OpenRedis(ctx, cfg.RedisAddr)
	if err != nil {
		return err
	}
	defer rdb.Close()

	waqiClient := waqi.NewClient(cfg.WAQIToken, cfg.WAQIBaseURL)
	locationRepo := repository.NewLocationRepo(db)
	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewHealthProfileRepo(db)
	thresholdRepo := repository.NewAlertThresholdRepo(db)
	stationRepo := repository.NewStationRepo(db)
	readingRepo := repository.NewAQIReadingRepo(db)
	pushRepo := repository.NewPushSubscriptionRepo(db)
	alertLogRepo := repository.NewAlertLogRepo(db)

	syncer := service.NewUserSyncer(userRepo, profileRepo, thresholdRepo, locationRepo)
	aqiSvc := service.NewAQIService(
		waqiClient,
		stationRepo,
		readingRepo,
		rdb,
		syncer,
		locationRepo,
	)

	webPushSender := service.NewVAPIDWebPushSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject, nil)
	emailSender := service.NewResendEmailSender(cfg.ResendAPIKey, cfg.ResendFromEmail, cfg.ResendBaseURL, nil)
	notifSvc := service.NewNotificationService(pushRepo, alertLogRepo, webPushSender, emailSender)
	alertSvc := service.NewAlertService(
		stationRepo,
		readingRepo,
		locationRepo,
		alertLogRepo,
		service.NewRedisAlertLock(rdb),
		notifSvc,
	)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	defer asynqClient.Close()
	aqiSvc.SetAlertEnqueuer(worker.NewAsynqAlertEnqueuer(asynqClient))

	switch task {
	case "sync_stations":
		res, err := aqiSvc.SyncStations(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("sync_stations: discovered=%d upserted=%d errors=%d\n", res.Discovered, res.Upserted, res.Errors)
		return nil

	case "fetch_readings":
		res, err := aqiSvc.FetchReadings(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("fetch_readings: total=%d success=%d skipped=%d failed=%d inserted=%d alert_enqueued=%d\n",
			res.Total, res.Success, res.Skipped, res.Failed, res.Inserted, res.AlertEnqueued)
		return nil

	case "check_alerts":
		if stationID == "" {
			return fmt.Errorf("-station_id is required when -task=check_alerts")
		}
		res, err := alertSvc.CheckAlerts(ctx, stationID)
		if err != nil {
			return err
		}
		fmt.Printf("check_alerts: station_id=%s evaluated=%d triggered=%d sent=%d failed=%d skipped=%d\n",
			res.StationID, res.EvaluatedUsers, res.Triggered, res.DispatchedSent, res.DispatchedFailed, res.DispatchedSkipped)
		return nil

	case "serve":
		redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
		scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{})
		if _, err := scheduler.Register("@hourly", worker.NewFetchReadingsTask(), asynq.Unique(55*time.Minute)); err != nil {
			return fmt.Errorf("register fetch_readings schedule: %w", err)
		}
		if _, err := scheduler.Register("@daily", worker.NewSyncStationsTask(), asynq.Unique(23*time.Hour)); err != nil {
			return fmt.Errorf("register sync_stations schedule: %w", err)
		}
		if err := scheduler.Start(); err != nil {
			return fmt.Errorf("start asynq scheduler: %w", err)
		}
		defer scheduler.Shutdown()

		srv := asynq.NewServer(
			redisOpt,
			asynq.Config{Concurrency: 4},
		)
		mux := asynq.NewServeMux()
		mux.Handle(worker.TypeSyncStations, worker.NewSyncStationsHandler(aqiSvc))
		mux.Handle(worker.TypeFetchReadings, worker.NewFetchReadingsHandler(aqiSvc))
		mux.Handle(worker.TypeCheckAlerts, worker.NewCheckAlertsHandler(alertSvc))
		slog.Info("worker listening", "redis", cfg.RedisAddr, "schedules", []string{"fetch_readings:@hourly", "sync_stations:@daily"})
		return srv.Run(mux)

	default:
		return fmt.Errorf("unknown -task %q (dùng: serve | sync_stations | fetch_readings | check_alerts)", task)
	}
}
