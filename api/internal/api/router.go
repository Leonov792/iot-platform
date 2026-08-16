package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"iot-platform/api/internal/auth"
)

func NewRouter(h *Handler, ah *AuthHandler, th *TelemetryHandler, uh *UsersHandler, dh *DiscoveryHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		// открытые ручки — без токена
		r.Post("/auth/register", ah.register)
		r.Post("/auth/login", ah.login)

		// ингест телеметрии не под jwt: гейтвей ходит со своим токеном
		r.Post("/telemetry", th.ingest)

		// всё, что ниже, требует авторизацию
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth)
			r.Route("/devices", func(r chi.Router) {
				r.Get("/", h.list)
				r.Post("/", h.create)
				r.Put("/{id}", h.update)
				r.Delete("/{id}", h.delete)
				r.Post("/{id}/command", h.command)
				r.Get("/{id}/telemetry", th.history)
				r.Post("/{id}/token", h.generateDeviceToken)
			})
			r.Route("/users", func(r chi.Router) {
				r.Get("/", uh.list)
				r.Post("/", uh.createMember)
				r.Put("/{id}/role", uh.setRole)
				r.Put("/{id}/schedule", uh.setSchedule)
			})
			r.Route("/discovered", func(r chi.Router) {
				r.Get("/", dh.list)
				r.Post("/{id}/approve", dh.approve)
				r.Post("/{id}/ignore", dh.ignore)
			})
		})
	})

	// закрытые ручки для гейтвея (ingest-токен). сюда фронт не ходит
	r.Post("/internal/device/{id}/verify", h.verifyDeviceToken)
	r.Get("/internal/telemetry/{id}/latest", th.latestInternal)
	r.Post("/internal/discovered", dh.upsertInternal)
	r.Post("/internal/automation-events", h.ingestAutomationEvent)
	r.Post("/internal/devices/{id}/eco", h.setEco)

	// для докера, чтобы мог проверять живость
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return r
}
