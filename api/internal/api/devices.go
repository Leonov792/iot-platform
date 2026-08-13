package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/gateway"
	"iot-platform/api/internal/models"
	"iot-platform/api/internal/store"
)

type Handler struct {
	devices *store.DeviceStore
	gateway *gateway.Client
}

func NewHandler(devices *store.DeviceStore, gw *gateway.Client) *Handler {
	return &Handler{devices: devices, gateway: gw}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.UserIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	devices, err := h.devices.List(r.Context(), ownerID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить устройства")
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.UserIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

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
	if d.State == nil {
		d.State = map[string]any{}
	}
	d.OwnerID = ownerID
	d.CreatedAt = time.Now()
	d.LastSeen = time.Now()

	if err := h.devices.Create(r.Context(), d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог создать устройство")
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.UserIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	id := chi.URLParam(r, "id")

	var d models.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if d.State == nil {
		d.State = map[string]any{}
	}
	d.ID = id
	d.OwnerID = ownerID

	if err := h.devices.Update(r.Context(), ownerID, d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог обновить устройство")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.UserIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.devices.Delete(r.Context(), ownerID, id); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог удалить устройство")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type commandReq struct {
	Action string `json:"action"`
	Value  any    `json:"value"`
}

// command — команда на устройство. идёт по цепочке фронт -> api -> гейтвей -> устройство
func (h *Handler) command(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.UserIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	id := chi.URLParam(r, "id")

	owned, err := h.devices.OwnedBy(r.Context(), id, ownerID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "база не отвечает")
		return
	}
	if !owned {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}

	var req commandReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.Action == "" {
		writeErr(w, http.StatusBadRequest, "action обязателен")
		return
	}

	if err := h.gateway.SendCommand(r.Context(), gateway.Command{
		DeviceID: id,
		Action:   req.Action,
		Value:    req.Value,
	}); err != nil {
		// гейтвей лежит — честно говорим 502, не врём клиенту
		writeErr(w, http.StatusBadGateway, "гейтвей не доступен: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "отправлено"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v) // ошибку тут забиваю — клиент уже отвалился, хрен с ним
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
