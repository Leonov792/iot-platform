package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/gateway"
	"iot-platform/api/internal/models"
)

type Handler struct {
	devices     deviceStore
	gateway     commandSender
	users       userStore
	commands    commandLogger
	ingestToken string
}

func NewHandler(devices deviceStore, gw commandSender, users userStore, commands commandLogger, ingestToken string) *Handler {
	return &Handler{devices: devices, gateway: gw, users: users, commands: commands, ingestToken: ingestToken}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	homeID, ok := auth.HomeIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	role, _ := auth.RoleFromCtx(r.Context())

	devices, err := h.devices.List(r.Context(), homeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить устройства")
		return
	}

	// персонал видит только бассейн/спортзал
	if role == auth.RoleStaff {
		filtered := make([]models.Device, 0, len(devices))
		for _, d := range devices {
			if staffZone(d.Zone) {
				filtered = append(filtered, d)
			}
		}
		devices = filtered
	}

	writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец создаёт устройства")
		return
	}

	var d models.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if d.ID == "" {
		writeErr(w, http.StatusBadRequest, "id обязателен")
		return
	}
	if d.Status == "" {
		d.Status = "offline"
	}
	if d.State == nil {
		d.State = map[string]any{}
	}
	if d.Zone == "" {
		d.Zone = "home"
	}
	d.OwnerID = homeID
	d.CreatedAt = time.Now()
	d.LastSeen = time.Now()

	if err := h.devices.Create(r.Context(), d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог создать устройство")
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец меняет устройства")
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
	d.OwnerID = homeID

	if err := h.devices.Update(r.Context(), homeID, d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог обновить устройство")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец удаляет устройства")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.devices.Delete(r.Context(), homeID, id); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог удалить устройство")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type commandReq struct {
	Action string `json:"action"`
	Value  any    `json:"value"`
}

// command — команда на устройство. идёт по цепочке фронт -> api -> гейтвей -> устройство.
// перед отправкой проверяем право роли управлять этим устройством прямо сейчас.
func (h *Handler) command(w http.ResponseWriter, r *http.Request) {
	userID, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	id := chi.URLParam(r, "id")

	d, err := h.devices.Get(r.Context(), id)
	if err != nil || d.OwnerID != homeID {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}

	var schedule []models.ScheduleEntry
	if role == auth.RoleStaff {
		// расписание берём из базы, а не из токена — чтобы смена часов вступала сразу
		schedule = h.userSchedule(r.Context(), userID)
	}
	if !authorized(role, d, schedule, time.Now()) {
		writeErr(w, http.StatusForbidden, "нет доступа к этому устройству")
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

	// оптимистично двигаем состояние в базе, чтобы фронт сразу видел новое
	h.applyState(r.Context(), homeID, id, req.Action, req.Value)

	// логируем команду для предиктивного анализа (не блокируем ответ)
	if h.commands != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = h.commands.Insert(ctx, id, userID, req.Action, req.Value)
		}()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "отправлено"})
}

func (h *Handler) userSchedule(ctx context.Context, userID string) []models.ScheduleEntry {
	u, err := h.users.GetByID(ctx, userID)
	if err != nil {
		return nil
	}
	return u.Schedule
}

// applyState — кладём в state устройства то, что влезло из команды.
// если не получилось — не фатально, команда и так ушла на устройство
func (h *Handler) applyState(ctx context.Context, homeID, id, action string, value any) {
	d, err := h.devices.Get(ctx, id)
	if err != nil {
		return
	}
	if d.State == nil {
		d.State = map[string]any{}
	}

	switch action {
	case "on":
		d.State["on"] = true
	case "off":
		d.State["on"] = false
	case "set_brightness":
		if v, ok := value.(float64); ok {
			d.State["brightness"] = v
			d.State["on"] = v > 0
		}
	case "set_target":
		if v, ok := value.(float64); ok {
			d.State["target_temp"] = v
		}
	case "set_color":
		if v, ok := value.(string); ok {
			d.State["color"] = v
			d.State["on"] = true
		}
	}

	_ = h.devices.Update(ctx, homeID, d)
}

// authCtx достаёт из контекста user_id, home_id и роль одной пачкой
func authCtx(ctx context.Context) (userID, homeID, role string, ok bool) {
	userID, ok = auth.UserIDFromCtx(ctx)
	if !ok {
		return "", "", "", false
	}
	role, _ = auth.RoleFromCtx(ctx)
	homeID, ok = auth.HomeIDFromCtx(ctx)
	return
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v) // ошибку тут забиваю — клиент уже отвалился, хрен с ним
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
