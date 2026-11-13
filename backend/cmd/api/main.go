package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	authpkg "aqi-health-monitor/backend/internal/auth"
	cfgpkg "aqi-health-monitor/backend/internal/config"
	dbpkg "aqi-health-monitor/backend/internal/database"
	handlerpkg "aqi-health-monitor/backend/internal/handler"
	mwpkg "aqi-health-monitor/backend/internal/middleware"
	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/internal/service"
	"aqi-health-monitor/backend/pkg/waqi"
)

func main() {
	// .env ở root repo (khi chạy go run từ backend/, root nằm ở ../.env).
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := cfgpkg.Load()

	db, err := dbpkg.OpenPostgres(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	rdb, err := dbpkg.OpenRedis(ctx, cfg.RedisAddr)
	if err != nil {
		return err
	}
	defer rdb.Close()

	jwks := authpkg.NewJWKSClient(cfg.KeycloakIssuer)
	verifier := authpkg.NewTokenVerifier(cfg.KeycloakIssuer, jwks)

	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewHealthProfileRepo(db)
	thresholdRepo := repository.NewAlertThresholdRepo(db)
	locationRepo := repository.NewLocationRepo(db)
	stationRepo := repository.NewStationRepo(db)
	readingRepo := repository.NewAQIReadingRepo(db)

	syncer := service.NewUserSyncer(userRepo, profileRepo, thresholdRepo, locationRepo)
	userHandler := handlerpkg.NewUserHandler(syncer)

	healthProfileSvc := service.NewHealthProfileService(db, syncer, profileRepo, thresholdRepo)
	healthProfileHandler := handlerpkg.NewHealthProfileHandler(healthProfileSvc)

	locationSvc := service.NewLocationService(syncer, locationRepo)
	locationHandler := handlerpkg.NewLocationHandler(locationSvc)

	waqiClient := waqi.NewClient(cfg.WAQIToken, cfg.WAQIBaseURL)
	aqiSvc := service.NewAQIService(waqiClient, stationRepo, readingRepo, rdb, syncer, locationRepo)
	aqiHandler := handlerpkg.NewAQIHandler(aqiSvc)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/me", mwpkg.Auth(verifier, http.HandlerFunc(userHandler.GetMe)))
	mux.Handle("POST /api/v1/health-profile", mwpkg.Auth(verifier, http.HandlerFunc(healthProfileHandler.Upsert)))
	mux.Handle("POST /api/v1/locations", mwpkg.Auth(verifier, http.HandlerFunc(locationHandler.Upsert)))
	mux.Handle("GET /api/v1/aqi/current", mwpkg.Auth(verifier, http.HandlerFunc(aqiHandler.GetCurrent)))

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("api listening on :%s (issuer %s)", cfg.AppPort, cfg.KeycloakIssuer)
	return srv.ListenAndServe()
}
