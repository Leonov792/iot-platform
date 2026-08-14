package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"iot-platform/api/internal/auth"
)

type TelemetryHandler struct {
	telemetry telemetryStore
	devices   deviceStore
	ingestKey string
}

func NewTelemetryHandler(t telemetryStore, d deviceStore, ingestKey string) *TelemetryHandler {
	return &TelemetryHandler{telemetry: t, devices: d, ingestKey: ingestKey}
}

// ingest — гейтвей шлёт сюда распарсенную телеметрию. это не пользовательская ручка,
// гейтвей ходит со своим секретным заголовком, а не с jwt
func (h *TelemetryHandler) ingest(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestKey {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	var req struct {
		DeviceID string         `json:"device_id"`
		Payload  map[string]any `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.DeviceID == "" {
		writeErr(w, http.StatusBadRequest, "device_id обязателен")
		return
	}

	if err := h.telemetry.Insert(r.Context(), req.DeviceID, req.Payload); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог сохранить телеметрию")
		return
	}

	// если устройства ещё нет в базе — не падаем: эмулятор мог заслать раньше,
	// чем юзер создал устройство. просто игнорим ошибку
	_ = h.devices.Touch(r.Context(), req.DeviceID)

	w.WriteHeader(http.StatusNoContent)
}

// latestInternal — последняя точка телеметрии устройства.
// закрытая ручка для движка автоматизации/ИИ (ingest-токен).
func (h *TelemetryHandler) latestInternal(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestKey {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	id := chi.URLParam(r, "id")
	t, err := h.telemetry.Latest(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "телеметрии нет")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// history — отдаёт последние точки для графика. доступ членам дома
func (h *TelemetryHandler) history(w http.ResponseWriter, r *http.Request) {
	homeID, ok := auth.HomeIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	deviceID := chi.URLParam(r, "id")

	owned, err := h.devices.OwnedBy(r.Context(), deviceID, homeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "база не отвечает")
		return
	}
	if !owned {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}

	since := time.Now().Add(-24 * time.Hour) // дефолт — последние сутки
	if v := r.URL.Query().Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			since = t
		}
	}

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	points, err := h.telemetry.List(r.Context(), deviceID, since, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить историю")
		return
	}
	writeJSON(w, http.StatusOK, points)
}
