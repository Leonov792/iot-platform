package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/go-chi/chi/v5"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/store"
)

// generateDeviceToken — владелец выпускает токен для устройства (аналог API-ключа).
// токен показывается один раз, в базе хранится только sha256.
func (h *Handler) generateDeviceToken(w http.ResponseWriter, r *http.Request) {
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

	token, err := newDeviceToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог сгенерировать токен")
		return
	}

	if err := h.devices.SetDeviceTokenHash(r.Context(), id, ownerID, hashDeviceToken(token)); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог сохранить токен")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// verifyDeviceToken — гейтвей проверяет токен устройства перед WebSocket-хендшейком.
// закрытая ручка: ходит только гейтвей со своим ingest-токеном.
func (h *Handler) verifyDeviceToken(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Ingest-Token") != h.ingestToken {
		writeErr(w, http.StatusUnauthorized, "неверный ingest-токен")
		return
	}

	id := chi.URLParam(r, "id")
	token := r.Header.Get("X-Device-Token")
	if token == "" {
		writeErr(w, http.StatusUnauthorized, "отсутствует device token")
		return
	}

	stored, err := h.devices.GetDeviceTokenHash(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "устройство не найдено или токен не задан")
			return
		}
		writeErr(w, http.StatusInternalServerError, "база не отвечает")
		return
	}

	if subtle.ConstantTimeCompare([]byte(stored), []byte(hashDeviceToken(token))) != 1 {
		writeErr(w, http.StatusUnauthorized, "неверный device token")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func newDeviceToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
