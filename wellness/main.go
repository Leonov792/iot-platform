// Wellness: предсказание фазы пробуждения по HRV (вариабельность ритма).
// Если за 15 мин до будильника HRV резко падает — запускает «мягкий рассвет»
// (шторы 15% -> кофе -> тёплый пол +1°C). Обратная связь: пользователь может
// отменить сценарий, и сервис запомнит «негативный паттерн» (пропуск на 7 дней).
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
	"strings"
	"sync"
	"syscall"
	"time"
)

type Step struct {
	Delay    string `json:"delay"`
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
	Value    any    `json:"value,omitempty"`
}

type Config struct {
	APIURL      string  `json:"api_url"`
	IngestToken string  `json:"ingest_token"`
	GatewayURL  string  `json:"gateway_url"`
	UserID      string  `json:"user_id"`
	Alarm       string  `json:"alarm"`      // "07:00"
	WindowMin   int     `json:"window_min"` // 15
	Threshold   float64 `json:"threshold"`  // -1.0
	Steps       []Step  `json:"steps"`
	Listen      string  `json:"listen"`
	PollSec     int     `json:"poll_seconds"` // 10
}

func loadConfig(path string) (Config, error) {
	cfg := Config{
		Alarm:     "07:00",
		WindowMin: 15,
		Threshold: -1.0,
		PollSec:   10,
		Listen:    ":8097",
		Steps: []Step{
			{Delay: "0s", DeviceID: "curtains", Action: "set_brightness", Value: 15},
			{Delay: "2m", DeviceID: "coffee-machine", Action: "on", Value: true},
			{Delay: "3m", DeviceID: "floor-heater", Action: "set_target", Value: 23},
		},
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil // дефолты
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.APIURL == "" || cfg.IngestToken == "" || cfg.UserID == "" {
		return cfg, errors.New("api_url, ingest_token и user_id обязательны")
	}
	return cfg, nil
}

type service struct {
	cfg    Config
	client *http.Client

	mu           sync.Mutex
	triggeredDay string
	skipUntil    time.Time
}

func (s *service) shouldSkip(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Before(s.skipUntil) {
		return true
	}
	if s.triggeredDay == now.Format("2006-01-02") {
		return true
	}
	return false
}

func (s *service) markTriggered(now time.Time) {
	s.mu.Lock()
	s.triggeredDay = now.Format("2006-01-02")
	s.mu.Unlock()
}

// cancel — «негативный паттерн»: пользователь выключил сценарий, пропускаем 7 дней.
func (s *service) cancel() {
	s.mu.Lock()
	s.skipUntil = time.Now().Add(7 * 24 * time.Hour)
	s.mu.Unlock()
	slog.Info("мягкий рассвет отменён пользователем, пропускаю 7 дней")
}

// fetchHRV тянет отсчёты HRV через внутренний эндпоинт api.
func (s *service) fetchHRV(ctx context.Context, minutes int) ([]Sample, error) {
	url := s.cfg.APIURL + "/internal/hrv?user_id=" + s.cfg.UserID + "&minutes=" + strconv.Itoa(minutes)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Ingest-Token", s.cfg.IngestToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("api ответил " + resp.Status)
	}

	var samples []Sample
	if err := json.NewDecoder(resp.Body).Decode(&samples); err != nil {
		return nil, err
	}
	return samples, nil
}

// sendCommand шлёт команду на устройство через гейтвей.
func (s *service) sendCommand(deviceID, action string, value any) error {
	body, err := json.Marshal(map[string]any{"device_id": deviceID, "action": action, "value": value})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.cfg.GatewayURL+"/internal/command", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("гейтвей ответил " + resp.Status)
	}
	return nil
}

// runSoftDawn выполняет шаги с задержками.
func (s *service) runSoftDawn() {
	slog.Info("запускаю «мягкий рассвет»")
	for _, step := range s.cfg.Steps {
		step := step
		delay := time.Duration(0)
		if d, err := time.ParseDuration(step.Delay); err == nil {
			delay = d
		}
		time.AfterFunc(delay, func() {
			if err := s.sendCommand(step.DeviceID, step.Action, step.Value); err != nil {
				slog.Warn("шаг рассвета не ушёл", "device", step.DeviceID, "err", err)
			} else {
				slog.Info("шаг рассвета", "device", step.DeviceID, "action", step.Action)
			}
		})
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := envOr("WELLNESS_CONFIG", "wellness.json")
	cfg, err := loadConfig(configPath)
	if err != nil {
		slog.Error("не прочитал конфиг", "err", err)
		os.Exit(1)
	}

	s := &service{cfg: cfg, client: &http.Client{Timeout: 10 * time.Second}}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/cancel", func(w http.ResponseWriter, _ *http.Request) {
		s.cancel()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: cfg.Listen, Handler: mux}
	go func() {
		slog.Info("wellness слушает", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("wellness http сдох", "err", err)
			os.Exit(1)
		}
	}()

	ticker := time.NewTicker(time.Duration(cfg.PollSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("гашу wellness...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			return

		case now := <-ticker.C:
			if s.shouldSkip(now) {
				continue
			}
			alarm := nextAlarm(now, cfg.Alarm)
			samples, err := s.fetchHRV(ctx, cfg.WindowMin+5)
			if err != nil {
				continue
			}
			if ShouldTriggerSoftDawn(samples, alarm, now, time.Duration(cfg.WindowMin)*time.Minute, cfg.Threshold) {
				s.markTriggered(now)
				s.runSoftDawn()
			}
		}
	}
}

// nextAlarm возвращает ближайшее наступление времени hh:mm.
func nextAlarm(now time.Time, hhmm string) time.Time {
	parts := strings.Split(hhmm, ":")
	if len(parts) != 2 {
		return now
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	alarm := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
	if alarm.Before(now) {
		alarm = alarm.Add(24 * time.Hour)
	}
	return alarm
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
