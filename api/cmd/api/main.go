package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iot-platform/api/internal/api"
	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/config"
	"iot-platform/api/internal/db"
	"iot-platform/api/internal/gateway"
	"iot-platform/api/internal/store"
)

func main() {
	// структурный JSON-логгер — обязательно для продакшена (см. README, «Логирование»)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()
	auth.SetSecret([]byte(cfg.JWTSecret))

	// чтобы гаситься по ctrl+c, а не оставлять висящие соединения в постгресе
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("не подключился к постгресу", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		slog.Error("миграции не прошли", "err", err)
		os.Exit(1)
	}

	devices := store.NewDeviceStore(pool)
	users := store.NewUserStore(pool)
	telemetry := store.NewTelemetryStore(pool)
	commands := store.NewCommandLogStore(pool)

	gw := gateway.NewClient(cfg.GatewayURL, &http.Client{Timeout: 5 * time.Second})

	h := api.NewHandler(devices, gw, users, commands, cfg.IngestToken)
	ah := api.NewAuthHandler(users)
	th := api.NewTelemetryHandler(telemetry, devices, cfg.IngestToken)
	uh := api.NewUsersHandler(users)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(h, ah, th, uh),
		// таймауты добавил не сразу: компилятор молчал, но голанци орал. вставил на всякий
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("поднимаю http", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("сервер сдох", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("гашу сервер...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("не успел погаситься нормально", "err", err)
	}
}
