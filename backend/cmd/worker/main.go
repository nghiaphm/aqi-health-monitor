package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"

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

	task := flag.String("task", "serve", "serve | sync_stations | fetch_readings")
	flag.Parse()

	if err := run(*task); err != nil {
		log.Fatal(err)
	}
}

func run(task string) error {
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
	aqiSvc := service.NewAQIService(
		waqiClient,
		repository.NewStationRepo(db),
		repository.NewAQIReadingRepo(db),
		rdb,
	)

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
		fmt.Printf("fetch_readings: total=%d success=%d skipped=%d failed=%d inserted=%d\n",
			res.Total, res.Success, res.Skipped, res.Failed, res.Inserted)
		return nil

	case "serve":
		srv := asynq.NewServer(
			asynq.RedisClientOpt{Addr: cfg.RedisAddr},
			asynq.Config{Concurrency: 4},
		)
		mux := asynq.NewServeMux()
		mux.Handle(worker.TypeSyncStations, worker.NewSyncStationsHandler(aqiSvc))
		mux.Handle(worker.TypeFetchReadings, worker.NewFetchReadingsHandler(aqiSvc))
		slog.Info("worker listening", "redis", cfg.RedisAddr)
		return srv.Run(mux)

	default:
		return fmt.Errorf("unknown -task %q (dùng: serve | sync_stations | fetch_readings)", task)
	}
}
