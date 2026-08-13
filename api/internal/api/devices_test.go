package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"iot-platform/api/internal/models"
)

func TestDevicesRequireAuth(t *testing.T) {
	h, _, _, _, _ := newTestRouter()

	rec := doJSON(t, h, http.MethodGet, "/api/v1/devices", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ждём 401 без токена, пришло %d", rec.Code)
	}
}

func TestCreateAndListDevices(t *testing.T) {
	h, ds, _, _, _ := newTestRouter()
	tok := authHeader("user-1")

	rec := doJSON(t, h, http.MethodPost, "/api/v1/devices",
		map[string]any{"id": "lamp-1", "name": "Лампа", "type": "light", "room": "Зал"},
		map[string]string{"Authorization": tok})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ждём 201, пришло %d: %s", rec.Code, rec.Body.String())
	}

	// чужое устройство не должно попадать в список
	_ = ds.Create(context.Background(), models.Device{ID: "other", OwnerID: "user-2", Name: "чужая"})

	rec = doJSON(t, h, http.MethodGet, "/api/v1/devices", nil, map[string]string{"Authorization": tok})
	if rec.Code != http.StatusOK {
		t.Fatalf("ждём 200, пришло %d", rec.Code)
	}

	var list []models.Device
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "lamp-1" {
		t.Fatalf("ждём только lamp-1, пришло %+v", list)
	}
}

func TestCommandUpdatesState(t *testing.T) {
	h, ds, _, _, gw := newTestRouter()
	tok := authHeader("user-1")

	doJSON(t, h, http.MethodPost, "/api/v1/devices",
		map[string]any{"id": "lamp-1", "name": "Лампа", "type": "light"},
		map[string]string{"Authorization": tok})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/devices/lamp-1/command",
		map[string]any{"action": "on"}, map[string]string{"Authorization": tok})
	if rec.Code != http.StatusOK {
		t.Fatalf("ждём 200, пришло %d: %s", rec.Code, rec.Body.String())
	}

	if len(gw.sent) != 1 || gw.sent[0].Action != "on" || gw.sent[0].DeviceID != "lamp-1" {
		t.Fatalf("команда не ушла в гейтвей: %+v", gw.sent)
	}

	d, err := ds.Get(context.Background(), "lamp-1")
	if err != nil {
		t.Fatal(err)
	}
	if d.State["on"] != true {
		t.Fatalf("состояние не обновилось: %+v", d.State)
	}
}

func TestCommandForeignDevice(t *testing.T) {
	h, ds, _, _, _ := newTestRouter()
	tok := authHeader("user-1")

	_ = ds.Create(context.Background(), models.Device{ID: "lamp-x", OwnerID: "user-2"})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/devices/lamp-x/command",
		map[string]any{"action": "on"}, map[string]string{"Authorization": tok})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("ждём 404 на чужое устройство, пришло %d", rec.Code)
	}
}

func TestDeleteDevice(t *testing.T) {
	h, ds, _, _, _ := newTestRouter()
	tok := authHeader("user-1")

	doJSON(t, h, http.MethodPost, "/api/v1/devices",
		map[string]any{"id": "lamp-1", "name": "Лампа", "type": "light"},
		map[string]string{"Authorization": tok})

	rec := doJSON(t, h, http.MethodDelete, "/api/v1/devices/lamp-1", nil,
		map[string]string{"Authorization": tok})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("ждём 204, пришло %d", rec.Code)
	}

	if _, err := ds.Get(context.Background(), "lamp-1"); err == nil {
		t.Fatal("устройство не удалилось")
	}
}
