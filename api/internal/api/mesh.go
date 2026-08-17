package api

import (
	"encoding/json"
	"net/http"

	"iot-platform/api/internal/auth"
)

// MeshHandler — модель дома для X-Ray AR (геометрия + якоря + зоны труб).
type MeshHandler struct {
	mesh meshStore
}

func NewMeshHandler(mesh meshStore) *MeshHandler {
	return &MeshHandler{mesh: mesh}
}

// put — PUT /api/v1/mesh (owner): сохранить модель дома.
func (h *MeshHandler) put(w http.ResponseWriter, r *http.Request) {
	_, homeID, role, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	if !canManageSystem(role) {
		writeErr(w, http.StatusForbidden, "только владелец сохраняет модель дома")
		return
	}

	var req struct {
		Mesh    any `json:"mesh"`
		Anchors any `json:"anchors"`
		Zones   any `json:"zones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "кривой json")
		return
	}

	if err := h.mesh.Put(r.Context(), homeID, req.Mesh, req.Anchors, req.Zones); err != nil {
		writeErr(w, http.StatusInternalServerError, "не смог сохранить модель")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// get — GET /api/v1/mesh (JWT): отдать модель дома клиенту (грузится один раз).
func (h *MeshHandler) get(w http.ResponseWriter, r *http.Request) {
	homeID, ok := auth.HomeIDFromCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	m, err := h.mesh.Get(r.Context(), homeID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "модель дома ещё не сохранена")
		return
	}
	writeJSON(w, http.StatusOK, m)
}
