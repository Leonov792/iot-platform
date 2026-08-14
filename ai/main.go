// Микросервис локального ИИ: Ollama-клиент, парсинг естественного языка в
// JSON-команды и предиктивный анализ привычек по логу команд.
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
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Port        string
	OllamaURL   string
	Model       string
	DatabaseURL string
}

func loadConfig() Config {
	return Config{
		Port:        getEnv("AI_PORT", "8095"),
		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		Model:       getEnv("OLLAMA_MODEL", "llama3.1"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://iot:iot@localhost:5432/iot?sslmode=disable"),
	}
}

type server struct {
	ollama    *OllamaClient
	predictor *Predictor
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// handleIntent — текст/голос -> JSON-массив команд.
func (s *server) handleIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.Text == "" {
		writeErr(w, http.StatusBadRequest, "text обязателен")
		return
	}

	cmds, err := parseIntent(r.Context(), s.ollama, req.Text)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "модель не ответила: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cmds)
}

// handleStatus — статус локального ИИ для админки.
func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	models, running, err := s.ollama.Status(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"online":  false,
			"model":   s.ollama.model,
			"error":   err.Error(),
			"models":  []string{},
			"running": []string{},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"online":  true,
		"model":   s.ollama.model,
		"models":  models,
		"running": running,
	})
}

// handleRecommendations — предиктивные рекомендации по привычкам.
func (s *server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if s.predictor == nil {
		writeErr(w, http.StatusServiceUnavailable, "нет доступа к базе")
		return
	}
	recs, err := s.predictor.Analyze(r.Context(), 30, 3)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог посчитать: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, recs)
}

func main() {
	configPath := flag.String("config", "", "не используется, всё из env")
	_ = configPath
	flag.Parse()

	cfg := loadConfig()

	ollama := NewOllamaClient(cfg.OllamaURL, cfg.Model)

	var predictor *Predictor
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			log.Printf("[ai] не подключился к базе (предиктив отключён): %v", err)
		} else {
			predictor = NewPredictor(pool)
			defer pool.Close()
		}
	}

	s := &server{ollama: ollama, predictor: predictor}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/intent", s.handleIntent)
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/recommendations", s.handleRecommendations)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("[ai] слушаю на %s (модель %s, ollama %s)", srv.Addr, cfg.Model, cfg.OllamaURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[ai] сервер сдох: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[ai] гашу...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
