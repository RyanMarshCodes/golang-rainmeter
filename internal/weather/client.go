package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	geocodeURL  = "https://geocoding-api.open-meteo.com/v1/search"
	forecastURL = "https://api.open-meteo.com/v1/forecast"
)

// Units is temperature/wind unit preference.
type Units string

const (
	UnitsF Units = "f"
	UnitsC Units = "c"
)

// Snapshot is the rendered weather state for a location.
type Snapshot struct {
	OK       bool
	Err      string
	Place    string // resolved display name from geocoder
	Query    string // original place / ZIP search string
	Updated  time.Time
	Current  Current
	Forecast []Day
}

// Current is live conditions.
type Current struct {
	TempC      float64
	FeelsC     float64
	Humidity   int
	WindKmh    float64
	WindDeg    int
	Code       int
	Label      string
	Icon       string // logical icon name (see icon-map.json)
}

// Day is one forecast column.
type Day struct {
	Date    time.Time
	Label   string // Mon / Tue / …
	HighC   float64
	LowC    float64
	Sunrise time.Time
	Sunset  time.Time
	SunOK   bool
	Code    int
	Icon    string
	Cond    string
}

// Client fetches and caches Open-Meteo data for a place name or US ZIP.
type Client struct {
	http *http.Client

	mu     sync.Mutex
	query  string
	lat    float64
	lon    float64
	place  string
	geoOK  bool
}

// NewClient returns an Open-Meteo client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 12 * time.Second},
	}
}

// Fetch resolves place (city, US ZIP, …; cached geo) and returns current + forecast.
func (c *Client) Fetch(query string, units Units) Snapshot {
	query = strings.TrimSpace(query)
	if query == "" {
		return Snapshot{Err: "place is empty"}
	}
	if units != UnitsC {
		units = UnitsF
	}

	lat, lon, place, err := c.resolve(query)
	if err != nil {
		return Snapshot{Query: query, Err: err.Error()}
	}

	snap, err := c.forecast(lat, lon, place, query, units)
	if err != nil {
		return Snapshot{Query: query, Place: place, Err: err.Error()}
	}
	return snap
}

func (c *Client) resolve(query string) (lat, lon float64, place string, err error) {
	c.mu.Lock()
	if c.geoOK && strings.EqualFold(c.query, query) {
		lat, lon, place = c.lat, c.lon, c.place
		c.mu.Unlock()
		return lat, lon, place, nil
	}
	c.mu.Unlock()

	q := url.Values{}
	q.Set("name", query)
	q.Set("count", "1")
	// US ZIPs resolve better with country filter; cities use open name search.
	if looksUSZIP(query) {
		q.Set("countryCode", "US")
	}

	var raw struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country_code"`
		} `json:"results"`
	}
	if err := getJSON(c.http, geocodeURL+"?"+q.Encode(), &raw); err != nil {
		return 0, 0, "", fmt.Errorf("geocode: %w", err)
	}
	if len(raw.Results) == 0 {
		return 0, 0, "", fmt.Errorf("no location for %q (try a city name; Canadian postals often fail)", query)
	}
	r := raw.Results[0]
	place = r.Name
	if r.Admin1 != "" {
		place = r.Name + ", " + r.Admin1
	}

	c.mu.Lock()
	c.query = query
	c.lat = r.Latitude
	c.lon = r.Longitude
	c.place = place
	c.geoOK = true
	c.mu.Unlock()
	return r.Latitude, r.Longitude, place, nil
}

func (c *Client) forecast(lat, lon float64, place, query string, units Units) (Snapshot, error) {
	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%f", lat))
	q.Set("longitude", fmt.Sprintf("%f", lon))
	q.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m,wind_direction_10m")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,sunrise,sunset")
	q.Set("timezone", "auto")
	q.Set("forecast_days", "4")
	if units == UnitsC {
		q.Set("temperature_unit", "celsius")
		q.Set("wind_speed_unit", "kmh")
	} else {
		q.Set("temperature_unit", "fahrenheit")
		q.Set("wind_speed_unit", "mph")
	}

	var raw struct {
		Current struct {
			Temp     float64 `json:"temperature_2m"`
			Humidity int     `json:"relative_humidity_2m"`
			Feels    float64 `json:"apparent_temperature"`
			Code     int     `json:"weather_code"`
			Wind     float64 `json:"wind_speed_10m"`
			WindDir  int     `json:"wind_direction_10m"`
		} `json:"current"`
		Daily struct {
			Time    []string  `json:"time"`
			Code    []int     `json:"weather_code"`
			High    []float64 `json:"temperature_2m_max"`
			Low     []float64 `json:"temperature_2m_min"`
			Sunrise []string  `json:"sunrise"`
			Sunset  []string  `json:"sunset"`
		} `json:"daily"`
	}
	if err := getJSON(c.http, forecastURL+"?"+q.Encode(), &raw); err != nil {
		return Snapshot{}, fmt.Errorf("forecast: %w", err)
	}

	label, icon := Describe(raw.Current.Code)
	snap := Snapshot{
		OK:      true,
		Place:   place,
		Query:   query,
		Updated: time.Now(),
		Current: Current{
			TempC:    raw.Current.Temp, // already in requested units; name kept for layout helpers
			FeelsC:   raw.Current.Feels,
			Humidity: raw.Current.Humidity,
			WindKmh:  raw.Current.Wind,
			WindDeg:  raw.Current.WindDir,
			Code:     raw.Current.Code,
			Label:    label,
			Icon:     icon,
		},
	}

	n := len(raw.Daily.Time)
	if n > 4 {
		n = 4
	}
	now := time.Now()
	for i := 0; i < n; i++ {
		t, err := time.ParseInLocation("2006-01-02", raw.Daily.Time[i], time.Local)
		if err != nil {
			t = now.AddDate(0, 0, i)
		}
		code := 0
		high, low := 0.0, 0.0
		if i < len(raw.Daily.Code) {
			code = raw.Daily.Code[i]
		}
		if i < len(raw.Daily.High) {
			high = raw.Daily.High[i]
		}
		if i < len(raw.Daily.Low) {
			low = raw.Daily.Low[i]
		}
		cond, ic := Describe(code)
		sunrise, sunOK := parseSunTime(raw.Daily.Sunrise, i)
		sunset, setOK := parseSunTime(raw.Daily.Sunset, i)
		snap.Forecast = append(snap.Forecast, Day{
			Date:    t,
			Label:   dayLabel(t),
			HighC:   high,
			LowC:    low,
			Sunrise: sunrise,
			Sunset:  sunset,
			SunOK:   sunOK && setOK,
			Code:    code,
			Icon:    ic,
			Cond:    cond,
		})
	}
	return snap, nil
}

func dayLabel(t time.Time) string {
	return t.Weekday().String()[:3]
}

func parseSunTime(times []string, i int) (time.Time, bool) {
	if i >= len(times) {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(times[i])
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		time.RFC3339,
	} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func looksUSZIP(s string) bool {
	if len(s) == 5 {
		_, err := strconv.Atoi(s)
		return err == nil
	}
	if prefix, suffix, ok := strings.Cut(s, "-"); ok && len(prefix) == 5 && len(suffix) == 4 {
		_, e1 := strconv.Atoi(prefix)
		_, e2 := strconv.Atoi(suffix)
		return e1 == nil && e2 == nil
	}
	return false
}

func getJSON(client *http.Client, rawURL string, dest any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "rmgo-weather/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
