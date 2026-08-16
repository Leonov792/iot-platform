// Модульный драйвер Modbus TCP: долгоживущий poller, который опрашивает
// оборудование бассейна/спортзала, публикует показания датчиков в общую шину
// (через ingest-ручку go api) и принимает команды на реле/клапаны.
//
// Конфиг (JSON) задаётся через env MODBUS_CONFIG (путь к файлу), дефолт — modbus.json.
// Пример структуры — в modbus.example.json в этом же каталоге.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"iot-platform/api/internal/modbus"
)

// Point — датчик: читается из регистра, масштабируется в физическую величину.
type Point struct {
	Name     string  `json:"name"`
	Register string  `json:"register"` // input | holding | coil
	Address  uint16  `json:"address"`
	Scale    float64 `json:"scale"`
	Unit     string  `json:"unit,omitempty"`
}

// Relay — исполнительный механизм: реле/клапан/насос, включается выкл.
type Relay struct {
	Name     string `json:"name"`
	Register string `json:"register"` // coil | holding
	Address  uint16 `json:"address"`
}

// Device — одно Modbus-устройство (слейв).
type Device struct {
	ID     string  `json:"id"`
	Host   string  `json:"host"`
	Port   int     `json:"port"`
	Unit   byte    `json:"unit"`
	Points []Point `json:"points"`
	Relays []Relay `json:"relays"`
}

// Config — конфиг poller'а.
type Config struct {
	APIURL       string   `json:"api_url"`
	IngestToken  string   `json:"ingest_token"`
	WriteToken   string   `json:"write_token"`
	Listen       string   `json:"listen"`
	PollInterval int      `json:"poll_interval_ms"`
	Devices      []Device `json:"devices"`
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
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5000
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8090"
	}
	if cfg.WriteToken == "" {
		return cfg, errors.New("write_token обязателен")
	}
	if cfg.APIURL == "" || cfg.IngestToken == "" {
		return cfg, errors.New("api_url и ingest_token обязательны")
	}
	return cfg, nil
}

type poller struct {
	cfg     Config
	mu      sync.Mutex
	clients map[string]*modbus.Client
	relays  map[string]map[string]Relay // device_id -> relay name -> relay
	http    *http.Client
	logs    *logRing
}

func newPoller(cfg Config, client *http.Client) *poller {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	p := &poller{
		cfg:     cfg,
		clients: map[string]*modbus.Client{},
		relays:  map[string]map[string]Relay{},
		http:    client,
		logs:    newLogRing(200),
	}
	for _, d := range cfg.Devices {
		p.clients[d.ID] = modbus.NewClient(d.Host, d.Port, d.Unit, 3*time.Second)
		m := map[string]Relay{}
		for _, r := range d.Relays {
			m[r.Name] = r
		}
		p.relays[d.ID] = m
	}
	return p
}

func (p *poller) close() {
	for _, c := range p.clients {
		_ = c.Close()
	}
}

// logEntry — запись о Modbus-операции для админки.
type logEntry struct {
	TS     time.Time `json:"ts"`
	Device string    `json:"device"`
	Kind   string    `json:"kind"` // read | write
	Target string    `json:"target"`
	Value  any       `json:"value,omitempty"`
	Error  string    `json:"error,omitempty"`
}

// logRing — кольцевой буфер последних операций.
type logRing struct {
	mu      sync.Mutex
	entries []logEntry
	max     int
}

func newLogRing(max int) *logRing {
	return &logRing{max: max, entries: make([]logEntry, 0, max)}
}

func (l *logRing) add(e logEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, e)
	if len(l.entries) > l.max {
		l.entries = l.entries[len(l.entries)-l.max:]
	}
}

func (l *logRing) all() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]logEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

func (p *poller) record(kind, device, target string, value any, err error) {
	e := logEntry{TS: time.Now(), Device: device, Kind: kind, Target: target, Value: value}
	if err != nil {
		e.Error = err.Error()
	}
	p.logs.add(e)
}

// readPoint читает значение датчика в зависимости от типа регистра.
func readPoint(c *modbus.Client, pt Point) (float64, error) {
	var raw uint16
	switch pt.Register {
	case "input":
		b, err := c.ReadInputRegisters(pt.Address, 1)
		if err != nil {
			return 0, err
		}
		if len(b) < 2 {
			return 0, errors.New("короткий ответ input-регистра")
		}
		raw = binary.BigEndian.Uint16(b)
	case "holding":
		b, err := c.ReadHoldingRegisters(pt.Address, 1)
		if err != nil {
			return 0, err
		}
		if len(b) < 2 {
			return 0, errors.New("короткий ответ holding-регистра")
		}
		raw = binary.BigEndian.Uint16(b)
	case "coil":
		b, err := c.ReadCoils(pt.Address, 1)
		if err != nil {
			return 0, err
		}
		if len(b) < 1 {
			return 0, errors.New("короткий ответ coil")
		}
		raw = 0
		if b[0]&0x01 != 0 {
			raw = 1
		}
	default:
		return 0, errors.New("неизвестный тип регистра " + pt.Register)
	}

	scale := pt.Scale
	if scale == 0 {
		scale = 1
	}
	return float64(raw) * scale, nil
}

// pollOne опрашивает одно устройство и публикует телеметрию.
func (p *poller) pollOne(d Device) {
	c := p.clients[d.ID]
	if c == nil {
		return
	}

	payload := map[string]any{}
	for _, pt := range d.Points {
		v, err := readPoint(c, pt)
		if err != nil {
			slog.Warn("точка не прочиталась", "device", d.ID, "point", pt.Name, "err", err)
			p.record("read", d.ID, pt.Name, nil, err)
			continue
		}
		payload[pt.Name] = v
	}
	if len(payload) == 0 {
		return
	}

	body, err := json.Marshal(map[string]any{"device_id": d.ID, "payload": payload})
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, p.cfg.APIURL+"/api/v1/telemetry", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ingest-Token", p.cfg.IngestToken)

	resp, err := p.http.Do(req)
	if err != nil {
		slog.Warn("не достучался до api", "device", d.ID, "err", err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		slog.Warn("api вернул ошибку на телеметрию", "device", d.ID, "status", resp.StatusCode)
	}
}

type writeReq struct {
	DeviceID string `json:"device_id"`
	Relay    string `json:"relay"`
	Value    bool   `json:"value"`
}

// handleWrite — команда на реле от правил/внешних систем.
func (p *poller) handleWrite(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Write-Token") != p.cfg.WriteToken {
		writeErr(w, http.StatusUnauthorized, "неверный write-токен")
		return
	}
	var req writeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}

	relays, ok := p.relays[req.DeviceID]
	if !ok {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}
	relay, ok := relays[req.Relay]
	if !ok {
		writeErr(w, http.StatusNotFound, "реле не найдено")
		return
	}
	c := p.clients[req.DeviceID]

	var err error
	switch relay.Register {
	case "coil":
		err = c.WriteSingleCoil(relay.Address, req.Value)
	case "holding":
		v := uint16(0)
		if req.Value {
			v = 1
		}
		err = c.WriteSingleRegister(relay.Address, v)
	default:
		err = errors.New("неизвестный тип реле " + relay.Register)
	}
	if err != nil {
		slog.Error("запись не прошла", "device", req.DeviceID, "relay", req.Relay, "err", err)
		p.record("write", req.DeviceID, req.Relay, req.Value, err)
		writeErr(w, http.StatusInternalServerError, "запись не прошла")
		return
	}

	slog.Info("записал реле", "device", req.DeviceID, "relay", req.Relay, "value", req.Value)
	p.record("write", req.DeviceID, req.Relay, req.Value, nil)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// handleLogs — последние Modbus-операции (для админки веб-дашборда).
func (p *poller) handleLogs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p.logs.all())
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := flag.String("config", envOr("MODBUS_CONFIG", "modbus.json"), "путь к конфигу")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		slog.Error("не прочитал конфиг", "err", err)
		os.Exit(1)
	}

	p := newPoller(cfg, &http.Client{Timeout: 5 * time.Second})
	defer p.close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/write", p.handleWrite)
	mux.HandleFunc("/logs", p.handleLogs)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: cfg.Listen, Handler: mux}
	go func() {
		slog.Info("modbus-poller слушает", "addr", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http сдох", "err", err)
			os.Exit(1)
		}
	}()

	interval := time.Duration(cfg.PollInterval) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("modbus-poller запущен", "devices", len(cfg.Devices), "interval", interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("гашу poller...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			return
		case <-ticker.C:
			for _, d := range cfg.Devices {
				p.pollOne(d)
			}
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
