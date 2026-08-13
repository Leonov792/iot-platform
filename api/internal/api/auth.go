package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/models"
	"iot-platform/api/internal/store"
)

type AuthHandler struct {
	users *store.UserStore
}

func NewAuthHandler(users *store.UserStore) *AuthHandler {
	return &AuthHandler{users: users}
}

// один тип под обе ручки — регистрация и логин принимают одинаково
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.Email == "" || len(req.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "почта пустая или пароль короче 6 символов")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог захэшировать пароль")
		return
	}

	u := models.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}
	if err := h.users.Create(r.Context(), u); err != nil {
		// TODO: отличать дубль почты от падения базы, пока сваливаю всё в 409
		writeErr(w, http.StatusConflict, "такой юзер уже есть или база отвалилась")
		return
	}

	token, err := auth.GenerateToken(u.ID, 24*time.Hour)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог подписать токен")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}

	u, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if err == store.ErrNotFound {
			// одинаковый текст для обеих ошибок, чтобы нельзя было перебирать почты
			writeErr(w, http.StatusUnauthorized, "неверный логин или пароль")
			return
		}
		writeErr(w, http.StatusInternalServerError, "база не отвечает")
		return
	}

	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "неверный логин или пароль")
		return
	}

	token, err := auth.GenerateToken(u.ID, 24*time.Hour)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог подписать токен")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
