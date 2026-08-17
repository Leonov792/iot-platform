// HomeKit-мост: пробрасывает устройства умного дома (лампы, розетки, термостаты,
// датчики) в Apple HomeKit как аксессуары через HAP-протокол (brutella/hap).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brutella/hap"
)

type Config struct {
	APIURL       string
	Email        string
	Password     string
	Pin          string
	Store        string
	Addr         string
	SyncInterval string
}

func loadConfig() Config {
	return Config{
		APIURL:       getEnv("API_URL", "http://localhost:8080"),
		Email:        os.Getenv("HOMEBRIDGE_EMAIL"),
		Password:     os.Getenv("HOMEBRIDGE_PASSWORD"),
		Pin:          getEnv("HOMEBRIDGE_PIN", "00102003"),
		Store:        getEnv("HOMEBRIDGE_STORE", "./homekit-store"),
		Addr:         getEnv("HOMEBRIDGE_ADDR", ":51826"),
		SyncInterval: getEnv("SYNC_INTERVAL", "30s"),
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := loadConfig()
	if cfg.Email == "" || cfg.Password == "" {
		slog.Error("HOMEBRIDGE_EMAIL и HOMEBRIDGE_PASSWORD обязательны")
		os.Exit(1)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	api := NewAPIClient(cfg.APIURL, cfg.Email, cfg.Password, httpClient)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := api.login(ctx); err != nil {
		slog.Error("не вошёл в api", "err", err)
		os.Exit(1)
	}

	devices, err := api.ListDevices(ctx)
	if err != nil {
		slog.Error("не вытащил устройства", "err", err)
		os.Exit(1)
	}
	slog.Info("устройств для HomeKit", "count", len(devices))

	bridge, accs, bindings := buildBridge(api, devices)

	store := hap.NewFsStore(cfg.Store)
	server, err := hap.NewServer(store, bridge, accs...)
	if err != nil {
		slog.Error("не поднял HAP-сервер", "err", err)
		os.Exit(1)
	}
	server.Pin = cfg.Pin
	server.Addr = cfg.Addr

	// фоновый синк состояния (sensor/light/thermostat) из api -> HomeKit
	interval, err := time.ParseDuration(cfg.SyncInterval)
	if err != nil {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncDevices(api, bindings)
			}
		}
	}()

	slog.Info("HomeKit-мост запущен", "addr", cfg.Addr, "pin", cfg.Pin)
	if err := server.ListenAndServe(ctx); err != nil {
		slog.Error("HAP-сервер упал", "err", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
