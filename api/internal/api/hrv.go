package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"iot-platform/api/internal/auth"
)

// HRVHandler — приём HRV (вариабельность ритма) с мобилки + чтение для wellness-сервиса.
type HRVHandler struct {
	hrv       hrvStore
	ingestKey string
}

func NewHRVHandler(hrv hrvStore, ingestKey string) *HRVHandler {
	return &HRVHandler{hrv: hrv, ingestKey: ingestKey}
}

// ingest — POST /api/v1/health/hrv (JWT): мобилка шлёт HRV каждые ~10 сек.
func (h *HRVHandler) ingest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	var req struct {
		Value float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.Value <= 0 {
		writeErr(w, http.StatusBadRequest, "value обязателен и > 0")
		return
	}

	if err := h.hrv.Insert(r.Context(), userID, req.Value); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог сохранить HRV")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// internal — GET /internal/hrv?user_id=X&minutes=30 (ingest-token): чтение для wellness.
func (h *HRVHandler) internal(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestKey {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeErr(w, http.StatusBadRequest, "user_id обязателен")
		return
	}

	minutes := 30
	if v := r.URL.Query().Get("minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minutes = n
		}
	}

	samples, err := h.hrv.Since(r.Context(), userID, time.Now().Add(-time.Duration(minutes)*time.Minute))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить HRV")
		return
	}
	writeJSON(w, http.StatusOK, samples)
}
