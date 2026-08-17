package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"iot-platform/api/internal/gateway"
	"iot-platform/api/internal/store"
)

// undoClimateWindow — окно, после которого климат не откатываем (экономия энергии).
const undoClimateWindow = 5 * time.Minute

// undoPlan строит компенсирующую команду для предыдущей команды.
// skip != "" — команду откатывать нельзя (вернётся причина). ok=false — компенсации нет.
func undoPlan(action string, previous map[string]any, deviceType string, age time.Duration) (gateway.Command, string, bool) {
	switch action {
	case "on", "off":
		on, ok := previous["on"].(bool)
		if !ok {
			on = false // нет данных — выключаем (безопасный дефолт)
		}
		a := "off"
		if on {
			a = "on"
		}
		return gateway.Command{Action: a}, "", true

	case "set_brightness":
		v, ok := previous["brightness"].(float64)
		if !ok {
			return gateway.Command{}, "нет прошлой яркости", false
		}
		return gateway.Command{Action: "set_brightness", Value: v}, "", true

	case "set_color":
		v, ok := previous["color"].(string)
		if !ok {
			return gateway.Command{}, "нет прошлого цвета", false
		}
		return gateway.Command{Action: "set_color", Value: v}, "", true

	case "set_target":
		// климат не гоним обратно, если прошло много времени (ИИ-слой решает по сроку)
		if deviceType == "thermostat" && age > undoClimateWindow {
			return gateway.Command{}, "климат не откатываем (прошло > 5 мин — экономия энергии)", false
		}
		v, ok := previous["target_temp"].(float64)
		if !ok {
			return gateway.Command{}, "нет прошлой температуры", false
		}
		return gateway.Command{Action: "set_target", Value: v}, "", true

	default:
		return gateway.Command{}, "нет компенсации для " + action, false
	}
}

// undo — «физический Ctrl+Z»: откатывает последнюю команду на устройство.
func (h *Handler) undo(w http.ResponseWriter, r *http.Request) {
	userID, homeID, _, ok := authCtx(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "не авторизован")
		return
	}

	id := chi.URLParam(r, "id")

	d, err := h.devices.Get(r.Context(), id)
	if err != nil || d.OwnerID != homeID {
		writeErr(w, http.StatusNotFound, "устройство не найдено")
		return
	}

	if h.history == nil {
		writeErr(w, http.StatusNotFound, "нечего откатывать")
		return
	}

	rec, err := h.history.Last(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "нечего откатывать")
			return
		}
		writeErr(w, http.StatusInternalServerError, "база не отвечает")
		return
	}

	cmd, skip, ok := undoPlan(rec.Action, rec.Previous, d.Type, time.Since(rec.CreatedAt))
	if !ok {
		writeJSON(w, http.StatusOK, map[string]string{"status": "skipped", "reason": skip})
		return
	}

	if err := h.gateway.SendCommand(r.Context(), cmd); err != nil {
		writeErr(w, http.StatusBadGateway, "гейтвей не доступен: "+err.Error())
		return
	}
	h.applyState(r.Context(), homeID, id, cmd.Action, cmd.Value)

	writeJSON(w, http.StatusOK, map[string]string{"status": "отменено", "action": cmd.Action})
}
