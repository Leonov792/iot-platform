package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"iot-platform/api/internal/models"
	"iot-platform/api/internal/store"
)

type Handler struct {
	devices *store.DeviceStore
}

func NewHandler(devices *store.DeviceStore) *Handler {
	return &Handler{devices: devices}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	devices, err := h.devices.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить устройства")
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var d models.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if d.ID == "" {
		// id пока не генерирую, пусть клиент сам придумывает. потом uuid прикручу
		writeErr(w, http.StatusBadRequest, "id обязателен")
		return
	}
	if d.Status == "" {
		d.Status = "offline"
	}
	d.CreatedAt = time.Now()
	d.LastSeen = time.Now()

	if err := h.devices.Create(r.Context(), d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог создать устройство")
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var d models.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	d.ID = id

	if err := h.devices.Update(r.Context(), d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог обновить устройство")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.devices.Delete(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог удалить устройство")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v) // ошибку тут забиваю — клиент уже отвалился, хрен с ним
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
