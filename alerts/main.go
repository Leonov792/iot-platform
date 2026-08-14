// Уведомления о критических авариях: протечка в бассейне, перегрев сауны,
// задымление и т.п. Опрашивает телеметрию через go api и шлёт push (FCM) и
// сообщения в Telegram при пробитии порогов.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Alert struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Severity string  `json:"severity"` // info | warning | critical
	DeviceID string  `json:"device_id"`
	Field    string  `json:"field"`
	Op       string  `json:"op"`
	Value    float64 `json:"value"`
	Message  string  `json:"message"`
	Cooldown string  `json:"cooldown"`
}

type Config struct {
	APIURL        string   `json:"api_url"`
	IngestToken   string   `json:"ingest_token"`
	PollInterval  string   `json:"poll_interval"`
	TelegramToken string   `json:"telegram_token,omitempty"`
	TelegramChat  string   `json:"telegram_chat,omitempty"`
	FCMCreds      string   `json:"fcm_credentials,omitempty"` // путь к service account json
	FCMTopic      string   `json:"fcm_topic,omitempty"`
	Alerts        []Alert  `json:"alerts"`
}

func loadConfig(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.APIURL == "" || cfg.IngestToken == "" {
		return cfg, errors.New("api_url и ingest_token обязательны")
	}
	if cfg.PollInterval == "" {
		cfg.PollInterval = "5s"
	}
	return cfg, nil
}

// Notifier шлёт одно уведомление.
type Notifier interface {
	Send(title, body string) error
}

// MultiNotifier рассылает по всем включённым каналам.
type MultiNotifier struct {
	notifiers []Notifier
}

func (m *MultiNotifier) Send(title, body string) error {
	var first error
	for _, n := range m.notifiers {
		if err := n.Send(title, body); err != nil {
			log.Printf("[alerts] канал не доставил: %v", err)
			first = err
		}
	}
	return first
}

type watcher struct {
	alerts   []Alert
	notifier Notifier

	mu       sync.Mutex
	lastSent map[string]time.Time
}

func (w *watcher) evaluate(states map[string]map[string]float64, now time.Time) {
	for _, a := range w.alerts {
		payload, ok := states[a.DeviceID]
		if !ok {
			continue
		}
		val, ok := payload[a.Field]
		if !ok || !match(a.Op, val, a.Value) {
			continue
		}
		if !w.cooldownOK(a, now) {
			continue
		}

		w.mu.Lock()
		w.lastSent[a.ID] = now
		w.mu.Unlock()

		title := "[" + a.Severity + "] " + a.Name
		body := a.Message
		if body == "" {
			body = a.DeviceID + ":" + a.Field + " " + a.Op + " " + formatFloat(a.Value)
		}
		log.Printf("[alerts] %s — %s", title, body)
		_ = w.notifier.Send(title, body)
	}
}

func (w *watcher) cooldownOK(a Alert, now time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	last, ok := w.lastSent[a.ID]
	if !ok {
		return true
	}
	d := 5 * time.Minute
	if a.Cooldown != "" {
		if parsed, err := time.ParseDuration(a.Cooldown); err == nil {
			d = parsed
		}
	}
	return now.Sub(last) >= d
}

func match(op string, got, want float64) bool {
	switch op {
	case "gt":
		return got > want
	case "lt":
		return got < want
	case "gte":
		return got >= want
	case "lte":
		return got <= want
	case "eq":
		return got == want
	case "neq":
		return got != want
	default:
		return false
	}
}

func formatFloat(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

type reader struct {
	apiURL string
	token  string
	client *http.Client
}

func (r *reader) latest(ctx context.Context, deviceID string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		r.apiURL+"/internal/telemetry/"+deviceID+"/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Ingest-Token", r.token)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("http " + resp.Status)
	}

	var t struct {
		Payload map[string]any `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, err
	}

	out := make(map[string]float64, len(t.Payload))
	for k, v := range t.Payload {
		if f, ok := v.(float64); ok {
			out[k] = f
		}
	}
	return out, nil
}

func main() {
	configPath := flag.String("config", envOr("ALERTS_CONFIG", "alerts.json"), "путь к конфигу")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("не прочитал конфиг: %v", err)
	}

	notifiers := make([]Notifier, 0, 2)
	if cfg.TelegramToken != "" && cfg.TelegramChat != "" {
		notifiers = append(notifiers, newTelegramNotifier(cfg.TelegramToken, cfg.TelegramChat))
	}
	if cfg.FCMCreds != "" {
		fcm, err := newFCMNotifier(cfg.FCMCreds, cfg.FCMTopic)
		if err != nil {
			log.Printf("[alerts] не поднял FCM (пропускаю): %v", err)
		} else {
			notifiers = append(notifiers, fcm)
		}
	}
	if len(notifiers) == 0 {
		log.Println("[alerts] предупреждение: ни один канал уведомлений не настроен")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	r := &reader{apiURL: cfg.APIURL, token: cfg.IngestToken, client: client}
	w := &watcher{alerts: cfg.Alerts, notifier: &MultiNotifier{notifiers: notifiers}, lastSent: map[string]time.Time{}}

	interval, err := time.ParseDuration(cfg.PollInterval)
	if err != nil {
		interval = 5 * time.Second
	}

	devices := alertDevices(cfg.Alerts)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("alerts запущен: %d алертов, %d устройств", len(cfg.Alerts), len(devices))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("гашу alerts...")
			return
		case now := <-ticker.C:
			states := make(map[string]map[string]float64, len(devices))
			for _, id := range devices {
				payload, err := r.latest(ctx, id)
				if err != nil {
					continue
				}
				states[id] = payload
			}
			w.evaluate(states, now)
		}
	}
}

func alertDevices(alerts []Alert) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, a := range alerts {
		if !seen[a.DeviceID] {
			seen[a.DeviceID] = true
			out = append(out, a.DeviceID)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
