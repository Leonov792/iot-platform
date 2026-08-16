package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WeatherProvider — прогноз OpenWeatherMap (5-day / 3-hour, метрика °C).
type WeatherProvider struct {
	apiKey string
	city   string
	http   *http.Client
}

func NewWeatherProvider(apiKey, city string, client *http.Client) *WeatherProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &WeatherProvider{apiKey: apiKey, city: city, http: client}
}

// FetchAverageTemp возвращает среднюю температуру на ближайшие сутки (для решения/лога).
func (w *WeatherProvider) FetchAverageTemp(ctx context.Context) (float64, error) {
	if w.apiKey == "" {
		return 0, fmt.Errorf("OWM_API_KEY не задан")
	}

	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric",
		w.city, w.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := w.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("openweathermap ответил %d", resp.StatusCode)
	}

	var out struct {
		List []struct {
			Main struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
		} `json:"list"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}

	const points = 8 // 8 точек по 3 часа = сутки
	if len(out.List) == 0 {
		return 0, fmt.Errorf("пустой прогноз")
	}

	sum, n := 0.0, 0
	for i, item := range out.List {
		if i >= points {
			break
		}
		sum += item.Main.Temp
		n++
	}
	return sum / float64(n), nil
}
