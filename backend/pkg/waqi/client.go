// Package waqi là HTTP client tối thiểu cho 2 endpoint WAQI đã liệt kê trong
// RESEARCH.md mục 2.1: /map/bounds/ (station discovery) và /feed/@id/ (reading).
package waqi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.waqi.info"
	defaultTimeout = 10 * time.Second
)

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient tạo WAQI client. baseURL rỗng → dùng API thật; cho phép override
// qua env WAQI_BASE_URL để test offline bằng fixture cùng schema.
func NewClient(token, baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		token:      token,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// BoundsStation là 1 phần tử trong data[] của /map/bounds/.
type BoundsStation struct {
	UID  int
	Lat  float64
	Lon  float64
	Name string
}

// Feed là reading đã chuẩn hóa từ /feed/@id/.
// HasAQI=false khi WAQI trả aqi thiếu/không parse được (service sẽ bỏ qua, log WARNING).
type Feed struct {
	AQI               int
	HasAQI            bool
	DominantPollutant *string
	PM25              *float64
	PM10              *float64
	O3                *float64
	NO2               *float64
	SO2               *float64
	CO                *float64
	MeasuredAt        time.Time
}

type boundsResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
	Data   []struct {
		UID     int     `json:"uid"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
		Station struct {
			Name string `json:"name"`
		} `json:"station"`
	} `json:"data"`
}

type feedResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		AQI         json.RawMessage `json:"aqi"`
		DominentPol string          `json:"dominentpol"`
		IAQI        map[string]struct {
			V float64 `json:"v"`
		} `json:"iaqi"`
		Time struct {
			V int64 `json:"v"`
		} `json:"time"`
	} `json:"data"`
}

// FetchStationsInBounds gọi /map/bounds/?latlng=lat1,lng1,lat2,lng2&token=...
func (c *Client) FetchStationsInBounds(ctx context.Context, lat1, lng1, lat2, lng2 float64) ([]BoundsStation, error) {
	q := url.Values{}
	q.Set("latlng", fmt.Sprintf("%.4f,%.4f,%.4f,%.4f", lat1, lng1, lat2, lng2))
	q.Set("token", c.token)

	var out boundsResponse
	if err := c.getJSON(ctx, c.baseURL+"/map/bounds/?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	if out.Status != "ok" {
		return nil, fmt.Errorf("waqi map/bounds status=%q msg=%q", out.Status, out.Msg)
	}

	stations := make([]BoundsStation, 0, len(out.Data))
	for _, d := range out.Data {
		stations = append(stations, BoundsStation{
			UID:  d.UID,
			Lat:  d.Lat,
			Lon:  d.Lon,
			Name: d.Station.Name,
		})
	}
	return stations, nil
}

// FetchStationFeed gọi /feed/@id/?token=...
func (c *Client) FetchStationFeed(ctx context.Context, waqiStationID string) (*Feed, error) {
	u := fmt.Sprintf("%s/feed/@%s/?token=%s", c.baseURL, url.PathEscape(waqiStationID), url.QueryEscape(c.token))

	var out feedResponse
	if err := c.getJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	if out.Status != "ok" {
		return nil, fmt.Errorf("waqi feed status=%q msg=%q", out.Status, out.Msg)
	}

	feed := &Feed{}
	if out.Data.Time.V > 0 {
		feed.MeasuredAt = time.Unix(out.Data.Time.V, 0).UTC()
	} else {
		feed.MeasuredAt = time.Now().UTC()
	}
	if aqi, ok := parseAQI(out.Data.AQI); ok {
		feed.AQI = aqi
		feed.HasAQI = true
	}
	if out.Data.DominentPol != "" {
		p := out.Data.DominentPol
		feed.DominantPollutant = &p
	}
	feed.PM25 = iaqiValue(out.Data.IAQI, "pm25")
	feed.PM10 = iaqiValue(out.Data.IAQI, "pm10")
	feed.O3 = iaqiValue(out.Data.IAQI, "o3")
	feed.NO2 = iaqiValue(out.Data.IAQI, "no2")
	feed.SO2 = iaqiValue(out.Data.IAQI, "so2")
	feed.CO = iaqiValue(out.Data.IAQI, "co")
	return feed, nil
}

// parseAQI chấp nhận số hoặc chuỗi; trả ok=false với "-"/rỗng/không hợp lệ.
func parseAQI(raw json.RawMessage) (int, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return 0, false
	}
	s = strings.Trim(s, `"`)
	if s == "" || s == "-" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return int(f), true
}

func iaqiValue(m map[string]struct {
	V float64 `json:"v"`
}, key string) *float64 {
	v, ok := m[key]
	if !ok {
		return nil
	}
	x := v.V
	return &x
}

func (c *Client) getJSON(ctx context.Context, url string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("waqi request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("waqi http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("waqi http status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("waqi decode: %w", err)
	}
	return nil
}
