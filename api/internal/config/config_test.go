package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	// чистим окружение, чтобы не зависеть от того, что налепил кто-то снаружи
	t.Setenv("API_PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("GATEWAY_URL", "")
	t.Setenv("INGEST_TOKEN", "")

	c := Load()

	if c.Port != "8080" {
		t.Fatalf("ждём порт 8080, пришло %q", c.Port)
	}
	if c.GatewayURL != "http://localhost:4000" {
		t.Fatalf("неожиданный gateway url: %q", c.GatewayURL)
	}
	if c.IngestToken != "dev-ingest-token" {
		t.Fatalf("неожиданный ingest token: %q", c.IngestToken)
	}
}
