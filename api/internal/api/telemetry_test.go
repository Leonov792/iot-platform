package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"iot-platform/api/internal/models"
)

func TestIngest(t *testing.T) {
	h, _, _, ts, _ := newTestRouter()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/telemetry",
		map[string]any{"device_id": "sensor-1", "payload": map[string]any{"temp": 21.5}},
		map[string]string{"X-Ingest-Token": "test-token"})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("ждём 204, пришло %d: %s", rec.Code, rec.Body.String())
	}
	if len(ts.rows["sensor-1"]) != 1 {
		t.Fatalf("телеметрия не сохранилась: %+v", ts.rows)
	}
}

func TestIngestBadToken(t *testing.T) {
	h, _, _, _, _ := newTestRouter()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/telemetry",
		map[string]any{"device_id": "sensor-1"},
		map[string]string{"X-Ingest-Token": "wrong"})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ждём 401 на левый токен, пришло %d", rec.Code)
	}
}

func TestHistory(t *testing.T) {
	h, ds, _, ts, _ := newTestRouter()
	tok := authHeader("user-1")

	_ = ds.Create(context.Background(), models.Device{ID: "sensor-1", OwnerID: "user-1"})
	_ = ts.Insert(context.Background(), "sensor-1", map[string]any{"temp": 21.5})

	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices/sensor-1/telemetry", nil,
		map[string]string{"Authorization": tok})

	if rec.Code != http.StatusOK {
		t.Fatalf("ждём 200, пришло %d: %s", rec.Code, rec.Body.String())
	}

	var points []models.Telemetry
	if err := json.NewDecoder(rec.Body).Decode(&points); err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("ждём 1 точку, пришло %d", len(points))
	}
}

func TestHistoryForeignDevice(t *testing.T) {
	h, ds, _, _, _ := newTestRouter()
	tok := authHeader("user-1")

	_ = ds.Create(context.Background(), models.Device{ID: "sensor-x", OwnerID: "user-2"})

	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices/sensor-x/telemetry", nil,
		map[string]string{"Authorization": tok})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("ждём 404 на чужое устройство, пришло %d", rec.Code)
	}
}
