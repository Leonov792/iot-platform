package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"iot-platform/api/internal/auth"
)

func NewRouter(h *Handler, ah *AuthHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// открытые ручки — без токена
	r.Post("/api/v1/auth/register", ah.register)
	r.Post("/api/v1/auth/login", ah.login)

	r.Route("/api/v1", func(r chi.Router) {
		// всё, что ниже, требует авторизацию
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth)
			r.Route("/devices", func(r chi.Router) {
				r.Get("/", h.list)
				r.Post("/", h.create)
				r.Put("/{id}", h.update)
				r.Delete("/{id}", h.delete)
			})
		})
	})

	// для докера, чтобы мог проверять живость
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return r
}
