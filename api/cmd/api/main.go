package main

import (
	"context"
	"errors"
	"log"
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
	cfg := config.Load()
	auth.SetSecret([]byte(cfg.JWTSecret))

	// чтобы гаситься по ctrl+c, а не оставлять висящие соединения в постгресе
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("не подключился к постгресу: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("миграции не прошли: %v", err)
	}

	devices := store.NewDeviceStore(pool)
	users := store.NewUserStore(pool)
	telemetry := store.NewTelemetryStore(pool)
	commands := store.NewCommandLogStore(pool)

	gw := gateway.NewClient(cfg.GatewayURL)

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
		log.Printf("поднимаю http на %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("сервер сдох: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("гашу сервер...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("не успел погаситься нормально: %v", err)
	}
}
