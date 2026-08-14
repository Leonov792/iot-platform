package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/models"
)

// UsersHandler — управление членами «дома» (семья/персонал). Только владелец.
type UsersHandler struct {
	users userStore
}

func NewUsersHandler(users userStore) *UsersHandler {
	return &UsersHandler{users: users}
}

// list — все члены дома
func (h *UsersHandler) list(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец смотрит пользователей")
		return
	}

	users, err := h.users.ListByHome(r.Context(), homeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог вытащить пользователей")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

type createMemberReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // family | staff
}

// createMember — владелец добавляет семью/персонал в свой «дом»
func (h *UsersHandler) createMember(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец добавляет пользователей")
		return
	}

	var req createMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if req.Email == "" || len(req.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "почта пустая или пароль короче 6 символов")
		return
	}
	if req.Role != auth.RoleFamily && req.Role != auth.RoleStaff {
		writeErr(w, http.StatusBadRequest, "роль должна быть family или staff")
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
		Role:         req.Role,
		HomeID:       homeID,
		Schedule:     []models.ScheduleEntry{},
		CreatedAt:    time.Now(),
	}
	if err := h.users.Create(r.Context(), u); err != nil {
		writeErr(w, http.StatusConflict, "такой юзер уже есть или база отвалилась")
		return
	}

	u.PasswordHash = ""
	writeJSON(w, http.StatusCreated, u)
}

type setRoleReq struct {
	Role string `json:"role"`
}

func (h *UsersHandler) setRole(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец меняет роли")
		return
	}

	id := chi.URLParam(r, "id")
	var req setRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}
	if !auth.ValidRole(req.Role) {
		writeErr(w, http.StatusBadRequest, "неизвестная роль")
		return
	}

	if err := h.users.SetRole(r.Context(), id, homeID, req.Role); err != nil {
		writeErr(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// setSchedule — владелец задаёт окна доступа персонала по зонам
func (h *UsersHandler) setSchedule(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец меняет расписание")
		return
	}

	id := chi.URLParam(r, "id")
	var req struct {
		Schedule []models.ScheduleEntry `json:"schedule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}

	if err := h.users.SetSchedule(r.Context(), id, homeID, req.Schedule); err != nil {
		writeErr(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
