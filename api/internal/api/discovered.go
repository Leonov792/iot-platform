package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// DiscoveryHandler — найденные в сети устройства (сканер подсети).
// Список/approve/ignore — только владелец; upsert — закрытая ручка гейтвея.
type DiscoveryHandler struct {
	discovery discoveryStore
	ingestKey string
}

func NewDiscoveryHandler(discovery discoveryStore, ingestKey string) *DiscoveryHandler {
	return &DiscoveryHandler{discovery: discovery, ingestKey: ingestKey}
}

// list — GET /api/v1/discovered?status=pending
func (h *DiscoveryHandler) list(w http.ResponseWriter, r *http.Request) {
	_, _, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец смотрит найденные устройства")
		return
	}

	status := r.URL.Query().Get("status")
	devices, err := h.discovery.List(r.Context(), status)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить найденные устройства")
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

// setStatus — общий хелпер для approve/ignore.
func (h *DiscoveryHandler) setStatus(w http.ResponseWriter, r *http.Request, status string) {
	_, _, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец управляет найденными устройствами")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "кривой id")
		return
	}
	if err := h.discovery.SetStatus(r.Context(), id, status); err != nil {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *DiscoveryHandler) approve(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "approved")
}

func (h *DiscoveryHandler) ignore(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "ignored")
}

// upsertInternal — POST /internal/discovered (ingest-token). Гейтвей-сканер зовёт сюда.
func (h *DiscoveryHandler) upsertInternal(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestKey {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	var req struct {
		IP      string `json:"ip"`
		Port    int    `json:"port"`
		Service string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.IP == "" || req.Port == 0 || req.Service == "" {
		writeErr(w, http.StatusBadRequest, "ip, port и service обязательны")
		return
	}

	inserted, err := h.discovery.Upsert(r.Context(), req.IP, req.Port, req.Service)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог сохранить")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"inserted": inserted})
}
