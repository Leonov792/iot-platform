package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	GatewayURL  string
	IngestToken string
	JWTSecret   string
}

// Load читает настройки из окружения. Дефолты под локальную сборку,
// чтобы не бегать с флагами каждый раз.
func Load() Config {
	return Config{
		Port:        getEnv("API_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://iot:iot@localhost:5432/iot?sslmode=disable"),
		GatewayURL:  getEnv("GATEWAY_URL", "http://localhost:4000"),
		IngestToken: getEnv("INGEST_TOKEN", "dev-ingest-token"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-tut-potom-zamenyu"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
