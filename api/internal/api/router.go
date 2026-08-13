package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/devices", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Put("/{id}", h.update)
			r.Delete("/{id}", h.delete)
		})
	})

	// для докера, чтобы мог проверять живость
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return r
}
