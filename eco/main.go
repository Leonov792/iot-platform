// EcoPlanner: загружает прогноз погоды (OpenWeatherMap) и цены на электроэнергию
// (Energy-Charts), строит оптимальное окно нагрева бассейна (ночь 23:00–06:00 при
// дешёвом тарифе) и пишет eco_mode/eco_plan в состояние устройства через go api.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	APIURL      string
	IngestToken string
	DeviceID    string
	OWMKey      string
	OWMCity     string
	BZN         string
	HeatHours   int
	Interval    string
}

func loadConfig() Config {
	return Config{
		APIURL:      getEnv("API_URL", "http://localhost:8080"),
		IngestToken: getEnv("INGEST_TOKEN", "dev-ingest-token"),
		DeviceID:    getEnv("ECO_DEVICE_ID", "pool-heater"),
		OWMKey:      os.Getenv("OWM_API_KEY"),
		OWMCity:     getEnv("OWM_CITY", "Berlin"),
		BZN:         getEnv("ECO_BZN", "DE-LU"),
		HeatHours:   getEnvInt("ECO_HEAT_HOURS", 4),
		Interval:    getEnv("ECO_INTERVAL", "30m"),
	}
}

// state хранит последний план для ручки /v1/plan.
type state struct {
	mu   sync.Mutex
	plan Plan
}

func (s *state) set(p Plan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plan = p
}

func (s *state) get() Plan {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.plan
}

// pushEco пишет eco_mode + eco_plan в устройство через внутренний эндпоинт API.
func pushEco(ctx context.Context, cfg Config, client *http.Client, plan Plan, ecoMode bool) error {
	body, err := json.Marshal(map[string]any{"eco_mode": ecoMode, "eco_plan": plan})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		cfg.APIURL+"/internal/devices/"+cfg.DeviceID+"/eco", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ingest-Token", cfg.IngestToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("api ответил " + resp.Status)
	}
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := loadConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	weather := NewWeatherProvider(cfg.OWMKey, cfg.OWMCity, httpClient)
	tariffs := NewEnergyCharts(cfg.BZN, httpClient)
	st := &state{}

	interval, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		interval = 30 * time.Minute
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/plan", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st.get())
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: getEnv("ECO_PORT", "8096"), Handler: mux}
	go func() {
		slog.Info("eco слушает", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("eco http сдох", "err", err)
			os.Exit(1)
		}
	}()

	runOnce := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		prices, err := tariffs.Fetch(ctx)
		if err != nil {
			slog.Warn("не получил тарифы, фолбэк на погоду", "err", err)
			prices = nil
		}

		if avg, err := weather.FetchAverageTemp(ctx); err == nil {
			slog.Info("прогноз погоды", "avg_temp_24h", avg)
		} else {
			slog.Warn("не получил погоду", "err", err)
		}

		plan := BuildPlan(prices, cfg.HeatHours)
		st.set(plan)
		if err := pushEco(ctx, cfg, httpClient, plan, true); err != nil {
			slog.Warn("не записал eco-план в устройство", "err", err)
			return
		}
		slog.Info("eco-план применён", "device", cfg.DeviceID, "start", plan.StartHour, "end", plan.EndHour, "mode", plan.Mode)
	}

	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("гашу eco...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			return
		case <-ticker.C:
			runOnce()
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
