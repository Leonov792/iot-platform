// Сервис автоматизации: опрашивает телеметрию через go api, прогоняет правила
// IF [Condition] THEN [Action] и управляет оборудованием через modbus-poller и
// гейтвей. Сценарии: долив воды, хлор/pH, климат спортзала.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iot-platform/automation/rules"
)

type Config struct {
	APIURL       string       `json:"api_url"`
	IngestToken  string       `json:"ingest_token"`
	ModbusURL    string       `json:"modbus_url"`
	ModbusToken  string       `json:"modbus_write_token"`
	GatewayURL   string       `json:"gateway_url"`
	RulesToken   string       `json:"rules_token"`
	Listen       string       `json:"listen"`
	PollInterval string       `json:"poll_interval"`
	Rules        []rules.Rule `json:"rules"`
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
	if cfg.ModbusURL == "" || cfg.ModbusToken == "" {
		return cfg, errors.New("modbus_url и modbus_write_token обязательны")
	}
	if cfg.RulesToken == "" {
		return cfg, errors.New("rules_token обязателен (защита ручки правил)")
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8091"
	}
	if cfg.PollInterval == "" {
		cfg.PollInterval = "2s"
	}
	return cfg, nil
}

// httpExecutor исполняет действия правил по HTTP.
type httpExecutor struct {
	modbusURL  string
	modbusTok  string
	gatewayURL string
	client     *http.Client
}

func newHTTPExecutor(modbusURL, modbusTok, gatewayURL string, client *http.Client) *httpExecutor {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &httpExecutor{
		modbusURL:  modbusURL,
		modbusTok:  modbusTok,
		gatewayURL: gatewayURL,
		client:     client,
	}
}

func (h *httpExecutor) post(url, token string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Write-Token", token)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("http " + resp.Status)
	}
	return nil
}

func (h *httpExecutor) ModbusWrite(deviceID, relay string, value bool) error {
	return h.post(h.modbusURL+"/internal/write", h.modbusTok, map[string]any{
		"device_id": deviceID, "relay": relay, "value": value,
	})
}

func (h *httpExecutor) Command(deviceID, action string, value any) error {
	return h.post(h.gatewayURL+"/internal/command", "", map[string]any{
		"device_id": deviceID, "action": action, "value": value,
	})
}

type telemetryReader struct {
	apiURL string
	token  string
	client *http.Client
}

func newTelemetryReader(apiURL, token string, client *http.Client) *telemetryReader {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &telemetryReader{apiURL: apiURL, token: token, client: client}
}

// latest возвращает map[field]float64 последней телеметрии устройства.
func (r *telemetryReader) latest(ctx context.Context, deviceID string) (map[string]float64, error) {
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
	configPath := flag.String("config", envOr("AUTOMATION_CONFIG", "automation.json"), "путь к конфигу")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("не прочитал конфиг: %v", err)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	executor := newHTTPExecutor(cfg.ModbusURL, cfg.ModbusToken, cfg.GatewayURL, httpClient)
	reader := newTelemetryReader(cfg.APIURL, cfg.IngestToken, httpClient)

	engine := rules.NewEngine(cfg.Rules, executor)
	interval, err := time.ParseDuration(cfg.PollInterval)
	if err != nil {
		interval = 2 * time.Second
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// HTTP-ручки: конструктор автоматизаций читает/пишет правила
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Automation-Token") != cfg.RulesToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "неверный automation-токен"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, engine.Rules())
		case http.MethodPut:
			var newRules []rules.Rule
			if err := json.NewDecoder(r.Body).Decode(&newRules); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "кривой json"})
				return
			}
			engine.SetRules(newRules)
			cfg.Rules = newRules
			if err := persistConfig(*configPath, cfg); err != nil {
				log.Printf("[automation] не сохранил правила на диск: %v", err)
			}
			log.Printf("[automation] правила обновлены: %d", len(newRules))
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "метод не тот"})
		}
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: cfg.Listen, Handler: mux}
	go func() {
		log.Printf("[automation] http на %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[automation] http сдох: %v", err)
		}
	}()

	log.Printf("automation запущен: %d правил, интервал %s", len(cfg.Rules), interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("гашу automation...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			return
		case now := <-ticker.C:
			devices := engine.Devices()
			states := make(map[string]map[string]float64, len(devices))
			for _, id := range devices {
				payload, err := reader.latest(ctx, id)
				if err != nil {
					// нет телеметрии — пропускаем, не спамим логом каждый тик
					continue
				}
				states[id] = payload
			}
			engine.Evaluate(states, now)
		}
	}
}

// persistConfig перезаписывает конфиг с новым списком правил.
func persistConfig(path string, cfg Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
