package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// TariffProvider возвращает 24 почасовые цены на электроэнергию (EUR/MWh),
// индекс 0 = 00:00. Реализация — Energy-Charts (api.energy-charts.info).
type TariffProvider interface {
	Fetch(ctx context.Context) ([]float64, error)
}

// EnergyCharts — открытый API day-ahead цен (зона bzn, напр. "DE-LU").
type EnergyCharts struct {
	bzn  string
	http *http.Client
}

func NewEnergyCharts(bzn string, client *http.Client) *EnergyCharts {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &EnergyCharts{bzn: bzn, http: client}
}

// Fetch тянет цены и раскладывает их по 24 часам.
func (e *EnergyCharts) Fetch(ctx context.Context) ([]float64, error) {
	u := "https://api.energy-charts.info/price?bzn=" + url.QueryEscape(e.bzn)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("energy-charts ответил %d", resp.StatusCode)
	}

	var out struct {
		UnixSeconds []int64    `json:"unix_seconds"`
		Price       []*float64 `json:"price"` // может содержать null
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	prices := make([]float64, 24)
	for i, sec := range out.UnixSeconds {
		if i >= len(out.Price) {
			break
		}
		h := time.Unix(sec, 0).In(time.Local).Hour()
		if h >= 0 && h < 24 && out.Price[i] != nil {
			prices[h] = *out.Price[i]
		}
	}
	return prices, nil
}
