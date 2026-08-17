package api

import (
	"testing"
	"time"
)

func TestUndoPlanOnOff(t *testing.T) {
	cmd, skip, ok := undoPlan("on", map[string]any{"on": false}, "light", 0)
	if !ok || skip != "" || cmd.Action != "off" {
		t.Fatalf("undo(on) должен дать off: %+v skip=%q ok=%v", cmd, skip, ok)
	}

	cmd, _, ok = undoPlan("off", map[string]any{"on": true}, "plug", 0)
	if !ok || cmd.Action != "on" {
		t.Fatalf("undo(off) должен дать on: %+v", cmd)
	}
}

func TestUndoPlanBrightness(t *testing.T) {
	cmd, _, ok := undoPlan("set_brightness", map[string]any{"brightness": float64(80)}, "light", 0)
	if !ok || cmd.Action != "set_brightness" || cmd.Value != float64(80) {
		t.Fatalf("undo(set_brightness) не восстановил яркость: %+v", cmd)
	}
}

func TestUndoPlanColor(t *testing.T) {
	cmd, _, ok := undoPlan("set_color", map[string]any{"color": "0000FF"}, "light", 0)
	if !ok || cmd.Action != "set_color" || cmd.Value != "0000FF" {
		t.Fatalf("undo(set_color) не восстановил цвет: %+v", cmd)
	}
}

func TestUndoPlanClimateWithinWindow(t *testing.T) {
	cmd, skip, ok := undoPlan("set_target", map[string]any{"target_temp": float64(22)}, "thermostat", 1*time.Minute)
	if !ok || skip != "" || cmd.Action != "set_target" || cmd.Value != float64(22) {
		t.Fatalf("климат в пределах окна должен откатываться: %+v skip=%q", cmd, skip)
	}
}

func TestUndoPlanClimateExpired(t *testing.T) {
	_, skip, ok := undoPlan("set_target", map[string]any{"target_temp": float64(22)}, "thermostat", 10*time.Minute)
	if ok {
		t.Fatalf("климат старше 5 мин не должен откатываться")
	}
	if skip == "" {
		t.Fatal("должна быть причина пропуска")
	}
}

func TestUndoPlanNoCompensation(t *testing.T) {
	_, skip, ok := undoPlan("set_volume", map[string]any{}, "light", 0)
	if ok {
		t.Fatal("неизвестная команда не должна откатываться")
	}
	if skip == "" {
		t.Fatal("должна быть причина")
	}
}
