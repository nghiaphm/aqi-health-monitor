package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/pkg/waqi"
)

var (
	ErrLocationNotFound  = errors.New("location not found")
	ErrNoStationCoverage = errors.New("no station coverage within 20km")
	ErrNoData            = errors.New("no data for station")
	ErrDataStale         = errors.New("data stale")
)

type CurrentLocation struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      *string `json:"city"`
}

type CurrentStation struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	City      *string `json:"city"`
	DistanceM int     `json:"distance_m"`
}

type Pollutants struct {
	PM25 *float64 `json:"pm25"`
	PM10 *float64 `json:"pm10"`
	O3   *float64 `json:"o3"`
	NO2  *float64 `json:"no2"`
	SO2  *float64 `json:"so2"`
	CO   *float64 `json:"co"`
}

type CurrentReading struct {
	AQI               int        `json:"aqi"`
	Level             string     `json:"level"`
	MeasuredAt        time.Time  `json:"measured_at"`
	DominantPollutant *string    `json:"dominant_pollutant"`
	Pollutants        Pollutants `json:"pollutants"`
}

type Advice struct {
	Summary string   `json:"summary"`
	Actions []string `json:"actions"`
}

type CurrentAQIResult struct {
	Location CurrentLocation `json:"location"`
	Station  CurrentStation  `json:"station"`
	Reading  CurrentReading  `json:"reading"`
	Advice   Advice          `json:"advice"`
}

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

// AQIService xử lý ingestion: đồng bộ trạm + fetch reading + ghi cache, và API tra cứu AQI hiện tại.
type AQIService struct {
	waqi      *waqi.Client
	stations  *repository.StationRepo
	readings  *repository.AQIReadingRepo
	redis     *redis.Client
	users     *UserSyncer
	locations *repository.LocationRepo
	nowFunc   func() time.Time
}

func NewAQIService(
	waqiClient *waqi.Client,
	stations *repository.StationRepo,
	readings *repository.AQIReadingRepo,
	rdb *redis.Client,
	users *UserSyncer,
	locations *repository.LocationRepo,
) *AQIService {
	return &AQIService{
		waqi:      waqiClient,
		stations:  stations,
		readings:  readings,
		redis:     rdb,
		users:     users,
		locations: locations,
		nowFunc:   time.Now,
	}
}

func (s *AQIService) SetNowFunc(f func() time.Time) {
	s.nowFunc = f
}

func (s *AQIService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc().UTC()
	}
	return time.Now().UTC()
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

// GetCurrent lấy AQI hiện tại cho 1 vị trí của user (hoặc vị trí mặc định 'current').
func (s *AQIService) GetCurrent(ctx context.Context, claims authpkg.Claims, locationID *string) (*CurrentAQIResult, error) {
	user, err := s.users.FindOrCreate(ctx, claims)
	if err != nil {
		return nil, err
	}

	var loc *models.UserLocation
	if locationID != nil {
		loc, err = s.locations.FindByIDAndUserID(ctx, *locationID, user.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLocationNotFound
		} else if err != nil {
			return nil, err
		}
	} else {
		loc, err = s.locations.FindByUserAndLabel(ctx, user.ID, "current")
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLocationNotFound
		} else if err != nil {
			return nil, err
		}
	}

	nearestStation, err := s.stations.FindNearestWithin(ctx, loc.Latitude, loc.Longitude, 20000)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoStationCoverage
	} else if err != nil {
		return nil, err
	}

	reading, err := s.getOrFetchReading(ctx, nearestStation.ID)
	if err != nil {
		return nil, err
	}

	// Rule DATA_STALE: reading cũ hơn 3 giờ so với hiện tại
	if s.now().Sub(reading.MeasuredAt) > 3*time.Hour {
		return nil, ErrDataStale
	}

	level := AQILevel(reading.AQI)
	advice := AQIAdvice(level)

	var locCity *string
	if loc.City.Valid {
		c := loc.City.String
		locCity = &c
	}
	var stationCity *string
	if nearestStation.City.Valid {
		c := nearestStation.City.String
		stationCity = &c
	}

	return &CurrentAQIResult{
		Location: CurrentLocation{
			ID:        loc.ID,
			Label:     loc.Label,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			City:      locCity,
		},
		Station: CurrentStation{
			ID:        nearestStation.ID,
			Name:      nearestStation.Name,
			City:      stationCity,
			DistanceM: nearestStation.DistanceM,
		},
		Reading: CurrentReading{
			AQI:               reading.AQI,
			Level:             level,
			MeasuredAt:        reading.MeasuredAt,
			DominantPollutant: reading.DominantPollutant,
			Pollutants: Pollutants{
				PM25: reading.PM25,
				PM10: reading.PM10,
				O3:   reading.O3,
				NO2:  reading.NO2,
				SO2:  reading.SO2,
				CO:   reading.CO,
			},
		},
		Advice: advice,
	}, nil
}

func (s *AQIService) getOrFetchReading(ctx context.Context, stationID string) (*cachedReading, error) {
	// 1. Check Redis cache-aside
	if s.redis != nil {
		val, err := s.redis.Get(ctx, cacheKey(stationID)).Bytes()
		if err == nil {
			var cached cachedReading
			if err := json.Unmarshal(val, &cached); err == nil {
				return &cached, nil
			}
		}
	}

	// 2. Cache miss -> query DB
	dbReading, err := s.readings.GetLatest(ctx, stationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoData
	} else if err != nil {
		return nil, err
	}

	res := &cachedReading{
		AQI:               dbReading.AQI,
		PM25:              dbReading.PM25,
		PM10:              dbReading.PM10,
		O3:                dbReading.O3,
		NO2:               dbReading.NO2,
		SO2:               dbReading.SO2,
		CO:                dbReading.CO,
		DominantPollutant: dbReading.DominantPollutant,
		MeasuredAt:        dbReading.Time,
	}

	// Ghi cache lại vào Redis
	if s.redis != nil {
		_ = s.cacheReading(ctx, stationID, *dbReading)
	}

	return res, nil
}

// AQILevel phân loại mức AQI thành 6 cấp độ chuẩn (API Contract mục 2.8).
func AQILevel(aqi int) string {
	switch {
	case aqi <= 50:
		return "good"
	case aqi <= 100:
		return "moderate"
	case aqi <= 150:
		return "unhealthy_sensitive"
	case aqi <= 200:
		return "unhealthy"
	case aqi <= 300:
		return "very_unhealthy"
	default:
		return "hazardous"
	}
}

// AQIAdvice trả khuyến nghị tổng hợp (summary) và danh sách hành động (actions) tương ứng với từng cấp độ AQI.
func AQIAdvice(level string) Advice {
	switch level {
	case "good":
		return Advice{
			Summary: "Chất lượng không khí Tốt — an toàn cho mọi hoạt động ngoài trời.",
			Actions: []string{
				"Tận hưởng các hoạt động ngoài trời",
				"Mở cửa thông gió tự nhiên",
			},
		}
	case "moderate":
		return Advice{
			Summary: "Chất lượng không khí ở mức Trung bình — có thể ảnh hưởng nhẹ tới nhóm người cực kỳ nhạy cảm.",
			Actions: []string{
				"Nhóm cực kỳ nhạy cảm nên chú ý triệu chứng khó thở hoặc ho",
				"Người bình thường có thể duy trì hoạt động ngoài trời",
			},
		}
	case "unhealthy_sensitive":
		return Advice{
			Summary: "Chất lượng không khí ở mức Kém — nhóm nhạy cảm (trẻ em, người già, bệnh hô hấp) bắt đầu xuất hiện triệu chứng.",
			Actions: []string{
				"Nhóm nhạy cảm hạn chế vận động mạnh ngoài trời",
				"Đeo khẩu trang lọc bụi khi ra đường",
				"Theo dõi các triệu chứng hô hấp",
			},
		}
	case "unhealthy":
		return Advice{
			Summary: "Chất lượng không khí ở mức Xấu — bắt đầu ảnh hưởng tới toàn bộ người dân, nhóm nhạy cảm bị ảnh hưởng nghiêm trọng hơn.",
			Actions: []string{
				"Hạn chế ra ngoài khi không cần thiết",
				"Đeo khẩu trang N95 nếu phải ra đường",
				"Đóng cửa sổ và bật máy lọc không khí",
			},
		}
	case "very_unhealthy":
		return Advice{
			Summary: "Chất lượng không khí ở mức Rất xấu — cảnh báo khẩn cấp, mọi người đều bị ảnh hưởng xấu tới sức khỏe.",
			Actions: []string{
				"Tránh mọi hoạt động thể chất ngoài trời",
				"Luôn đeo khẩu trang N95 khi bắt buộc phải ra ngoài",
				"Đóng kín cửa và duy trì máy lọc không khí trong nhà",
			},
		}
	default: // hazardous
		return Advice{
			Summary: "Chất lượng không khí ở mức Nguy hiểm — cảnh báo y tế cao nhất, nguy cơ ảnh hưởng nghiêm trọng đến hệ hô hấp.",
			Actions: []string{
				"Ở trong nhà và đóng kín toàn bộ cửa sổ",
				"Bật máy lọc không khí ở công suất tối đa",
				"Tránh tuyệt đối các hoạt động ngoài trời",
			},
		}
	}
}
