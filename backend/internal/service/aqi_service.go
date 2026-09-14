package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/pkg/waqi"
)

const cacheTTL = 20 * time.Minute

// cityBounds: bounding box gần đúng để discovery trạm cho 2 thành phố trong scope MVP
// (dùng cho /map/bounds/). Điều chỉnh khi đối chiếu mật độ trạm thực tế lần đầu.
type cityBounds struct {
	city                   string
	lat1, lng1, lat2, lng2 float64
}

var syncBounds = []cityBounds{
	{"hanoi", 20.85, 105.70, 21.20, 106.00},
	{"ho-chi-minh-city", 10.65, 106.55, 10.95, 106.85},
}

// SyncStationsResult là log tổng kết 1 lần sync_stations.
type SyncStationsResult struct {
	Discovered int
	Upserted   int
	Errors     int
}

// FetchReadingsResult là log tổng kết 1 lần fetch_readings.
type FetchReadingsResult struct {
	Total    int
	Success  int
	Skipped  int
	Failed   int
	Inserted int
}

// cachedReading là payload lưu Redis key aqi:current:{station_id} (cache-aside).
type cachedReading struct {
	AQI               int       `json:"aqi"`
	PM25              *float64  `json:"pm25"`
	PM10              *float64  `json:"pm10"`
	O3                *float64  `json:"o3"`
	NO2               *float64  `json:"no2"`
	SO2               *float64  `json:"so2"`
	CO                *float64  `json:"co"`
	DominantPollutant *string   `json:"dominant_pollutant"`
	MeasuredAt        time.Time `json:"measured_at"`
}

// AQIService xử lý ingestion: đồng bộ trạm + fetch reading + ghi cache.
type AQIService struct {
	waqi     *waqi.Client
	stations *repository.StationRepo
	readings *repository.AQIReadingRepo
	redis    *redis.Client
}

func NewAQIService(
	waqiClient *waqi.Client,
	stations *repository.StationRepo,
	readings *repository.AQIReadingRepo,
	rdb *redis.Client,
) *AQIService {
	return &AQIService{waqi: waqiClient, stations: stations, readings: readings, redis: rdb}
}

// SyncStations discovery trạm cho Hà Nội + TP.HCM rồi upsert theo waqi_station_id.
func (s *AQIService) SyncStations(ctx context.Context) (SyncStationsResult, error) {
	var result SyncStationsResult

	for _, b := range syncBounds {
		list, err := s.waqi.FetchStationsInBounds(ctx, b.lat1, b.lng1, b.lat2, b.lng2)
		if err != nil {
			result.Errors++
			slog.Warn("sync_stations: fetch bounds failed", "city", b.city, "error", err)
			continue
		}

		city := b.city
		for _, st := range list {
			result.Discovered++
			if st.UID == 0 || st.Name == "" {
				result.Errors++
				slog.Warn("sync_stations: skip invalid station entry", "city", b.city, "uid", st.UID)
				continue
			}
			waqiID := strconv.Itoa(st.UID)
			if _, err := s.stations.UpsertByWAQIID(ctx, waqiID, st.Name, st.Lat, st.Lon, &city); err != nil {
				result.Errors++
				slog.Warn("sync_stations: upsert failed", "waqi_station_id", waqiID, "error", err)
				continue
			}
			result.Upserted++
		}
	}

	slog.Info("sync_stations done",
		"discovered", result.Discovered,
		"upserted", result.Upserted,
		"errors", result.Errors,
	)
	return result, nil
}

// FetchReadings duyệt trạm active, lấy feed, lưu reading + cache.
// Record thiếu aqi hoặc lỗi 1 trạm → log WARNING và bỏ qua, KHÔNG fail cả job.
func (s *AQIService) FetchReadings(ctx context.Context) (FetchReadingsResult, error) {
	var result FetchReadingsResult

	stations, err := s.stations.ListActive(ctx)
	if err != nil {
		return result, err
	}
	result.Total = len(stations)

	for _, st := range stations {
		feed, err := s.waqi.FetchStationFeed(ctx, st.WAQIStationID)
		if err != nil {
			result.Failed++
			slog.Warn("fetch_readings: fetch feed failed", "waqi_station_id", st.WAQIStationID, "error", err)
			continue
		}
		if !feed.HasAQI {
			result.Skipped++
			slog.Warn("fetch_readings: missing aqi, skip record (BUSINESS-RULES.md mục 3)", "waqi_station_id", st.WAQIStationID)
			continue
		}

		reading := models.AQIReading{
			Time:              feed.MeasuredAt,
			StationID:         st.ID,
			AQI:               feed.AQI,
			PM25:              feed.PM25,
			PM10:              feed.PM10,
			O3:                feed.O3,
			NO2:               feed.NO2,
			SO2:               feed.SO2,
			CO:                feed.CO,
			DominantPollutant: feed.DominantPollutant,
		}

		inserted, err := s.readings.Insert(ctx, reading)
		if err != nil {
			result.Failed++
			slog.Warn("fetch_readings: insert failed", "waqi_station_id", st.WAQIStationID, "error", err)
			continue
		}

		if err := s.cacheReading(ctx, st.ID, reading); err != nil {
			slog.Warn("fetch_readings: cache failed", "station_id", st.ID, "error", err)
		}

		result.Success++
		if inserted {
			result.Inserted++
		}
	}

	slog.Info("fetch_readings done",
		"total", result.Total,
		"success", result.Success,
		"skipped", result.Skipped,
		"failed", result.Failed,
		"inserted", result.Inserted,
	)
	return result, nil
}

func (s *AQIService) cacheReading(ctx context.Context, stationID string, rd models.AQIReading) error {
	payload := cachedReading{
		AQI:               rd.AQI,
		PM25:              rd.PM25,
		PM10:              rd.PM10,
		O3:                rd.O3,
		NO2:               rd.NO2,
		SO2:               rd.SO2,
		CO:                rd.CO,
		DominantPollutant: rd.DominantPollutant,
		MeasuredAt:        rd.Time,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, cacheKey(stationID), b, cacheTTL).Err()
}

func cacheKey(stationID string) string {
	return "aqi:current:" + stationID
}
