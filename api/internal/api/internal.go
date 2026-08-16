package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// setEco — закрытая ручка (ingest-token): EcoPlanner пишет eco-флаг и план в state устройства.
func (h *Handler) setEco(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestToken {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	id := chi.URLParam(r, "id")
	var req struct {
		EcoMode bool `json:"eco_mode"`
		EcoPlan any  `json:"eco_plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}

	d, err := h.devices.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}
	if d.State == nil {
		d.State = map[string]any{}
	}
	d.State["eco_mode"] = req.EcoMode
	if req.EcoPlan != nil {
		d.State["eco_plan"] = req.EcoPlan
	}

	if err := h.devices.Update(r.Context(), d.OwnerID, d); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог обновить устройство")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ingestAutomationEvent — закрытая ручка (ingest-token): движок автоматизации
// шлёт сюда событие срабатывания правила (для бизнес-метрик Grafana).
func (h *Handler) ingestAutomationEvent(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestToken {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	var req struct {
		RuleID   string `json:"rule_id"`
		RuleName string `json:"rule_name"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.RuleID == "" {
		writeErr(w, http.StatusBadRequest, "rule_id обязателен")
		return
	}

	if h.events != nil {
		if err := h.events.InsertEvent(r.Context(), req.RuleID, req.RuleName, req.DeviceID); err != nil {
			writeErr(w, http.StatusInternalServerError, "не смог сохранить событие")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
