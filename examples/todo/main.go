package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/kdnk/kdnk"
	"github.com/north-shore-software/kdnk/providers"
)

//go:embed web
var webAssets embed.FS

type Todo struct {
	Title         string  `json:"title" jsonschema:"What the task is"`
	Urgency       float64 `json:"urgency" jsonschema:"1-10, how soon it must happen"`
	ExpectedValue float64 `json:"expected_value" jsonschema:"1-10, how much it is worth"`
}

type Profile struct {
	Location  string  `json:"location" jsonschema:"Place name as the user gave it"`
	Latitude  float64 `json:"latitude" jsonschema:"From find_location"`
	Longitude float64 `json:"longitude" jsonschema:"From find_location"`
	Timezone  string  `json:"timezone" jsonschema:"IANA timezone from find_location"`
}

type FindLocationInput struct {
	Name string `json:"name" jsonschema:"Place name, e.g. Calgary"`
}

type TimeInput struct {
	Timezone string `json:"timezone,omitempty" jsonschema:"IANA timezone; UTC if empty"`
}

type WeatherInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	if err := kdnk.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *kdnk.Registry) error {
	p, err := provider()
	if err != nil {
		return err
	}
	web, err := fs.Sub(webAssets, "web")
	if err != nil {
		return err
	}
	r.Provider(p)
	r.SystemPrompt(strings.Join([]string{
		"You help the user prioritise tasks by expected value and urgency, both on a 1-10 scale.",
		"Use todos_create to record a task, todos_list and todos_get to read what is already recorded,",
		"todos_update to change a task, and todos_delete to remove one.",
		"When the user asks what to do on a given day, check their location with profile_get",
		"(ask for it and save it with find_location and profile_set if missing),",
		"use get_time and get_weather for that place, and avoid suggesting outdoor tasks on days with bad weather.",
		"get_weather only covers the next 7 days.",
		"After a tool returns, tell the user what happened in plain language.",
	}, " "))
	r.Web(web)
	kdnk.Collection[Todo](r, "todos")
	kdnk.Singleton[Profile](r, "profile")
	kdnk.Tool(r, "find_location", "Look up a place by name. Returns up to 5 matches with latitude, longitude and IANA timezone.", findLocation)
	kdnk.Tool(r, "get_time", "Get the current date, time and weekday in an IANA timezone, or in UTC if the timezone is empty or unknown.", getTime)
	kdnk.Tool(r, "get_weather", "Get the daily weather forecast at a latitude and longitude for today and the next 6 days. It cannot see further than 7 days ahead.", getWeather)
	return nil
}

func getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", req.URL.Host, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func findLocation(ctx context.Context, _ core.App, _ kdnk.Session, in FindLocationInput) (any, error) {
	var res struct {
		Results []struct {
			Name      string  `json:"name"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}
	q := url.Values{"name": {in.Name}, "count": {"5"}}
	if err := getJSON(ctx, "https://geocoding-api.open-meteo.com/v1/search?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	if len(res.Results) == 0 {
		return nil, fmt.Errorf("no place found for %q", in.Name)
	}
	out := make([]map[string]any, 0, len(res.Results))
	for _, p := range res.Results {
		out = append(out, map[string]any{
			"name":      p.Name,
			"region":    p.Admin1,
			"country":   p.Country,
			"latitude":  p.Latitude,
			"longitude": p.Longitude,
			"timezone":  p.Timezone,
		})
	}
	return out, nil
}

func getTime(_ context.Context, _ core.App, _ kdnk.Session, in TimeInput) (any, error) {
	loc := time.UTC
	if in.Timezone != "" {
		if l, err := time.LoadLocation(in.Timezone); err == nil {
			loc = l
		}
	}
	now := time.Now().In(loc)
	return map[string]any{
		"time":     now.Format(time.RFC3339),
		"date":     now.Format(time.DateOnly),
		"weekday":  now.Weekday().String(),
		"timezone": loc.String(),
	}, nil
}

func getWeather(ctx context.Context, _ core.App, _ kdnk.Session, in WeatherInput) (any, error) {
	var res struct {
		Timezone string `json:"timezone"`
		Daily    struct {
			Time        []string  `json:"time"`
			WeatherCode []int     `json:"weather_code"`
			TempMax     []float64 `json:"temperature_2m_max"`
			TempMin     []float64 `json:"temperature_2m_min"`
			PrecipProb  []float64 `json:"precipitation_probability_max"`
			PrecipSum   []float64 `json:"precipitation_sum"`
			WindMax     []float64 `json:"wind_speed_10m_max"`
		} `json:"daily"`
	}
	q := url.Values{
		"latitude":      {fmt.Sprint(in.Latitude)},
		"longitude":     {fmt.Sprint(in.Longitude)},
		"daily":         {"weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,precipitation_sum,wind_speed_10m_max"},
		"timezone":      {"auto"},
		"forecast_days": {"7"},
	}
	if err := getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	d := res.Daily
	n := len(d.Time)
	if len(d.WeatherCode) != n || len(d.TempMax) != n || len(d.TempMin) != n || len(d.PrecipProb) != n || len(d.PrecipSum) != n || len(d.WindMax) != n {
		return nil, errors.New("malformed forecast")
	}
	days := make([]map[string]any, 0, n)
	for i, date := range d.Time {
		t, err := time.Parse(time.DateOnly, date)
		if err != nil {
			return nil, err
		}
		days = append(days, map[string]any{
			"date":                          date,
			"weekday":                       t.Weekday().String(),
			"conditions":                    conditions(d.WeatherCode[i]),
			"temp_max_c":                    d.TempMax[i],
			"temp_min_c":                    d.TempMin[i],
			"precipitation_probability_pct": d.PrecipProb[i],
			"precipitation_mm":              d.PrecipSum[i],
			"wind_max_kmh":                  d.WindMax[i],
		})
	}
	return map[string]any{"timezone": res.Timezone, "days": days}, nil
}

func conditions(code int) string {
	switch {
	case code == 0:
		return "clear"
	case code <= 2:
		return "partly cloudy"
	case code == 3:
		return "overcast"
	case code == 45 || code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 71 && code <= 77:
		return "snow"
	case code >= 80 && code <= 82:
		return "rain showers"
	case code == 85 || code == 86:
		return "snow showers"
	case code >= 95:
		return "thunderstorm"
	}
	return fmt.Sprintf("weather code %d", code)
}

func provider() (providers.Provider, error) {
	switch set := os.Getenv("KDNK_MODEL_SET"); set {
	case "", "local":
		return &providers.Ollama{Model: "gemma4"}, nil
	case "cloud":
		return &providers.Gemini{Model: "gemini-3.6-flash"}, nil
	default:
		return nil, fmt.Errorf("unknown KDNK_MODEL_SET %q (want local or cloud)", set)
	}
}
