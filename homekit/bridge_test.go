package main

import "testing"

func TestBuildBridgeMapping(t *testing.T) {
	api := NewAPIClient("http://localhost:8080", "a@b.c", "secret", nil)
	devices := []Device{
		{ID: "l1", Name: "Лампа", Type: "light"},
		{ID: "p1", Name: "Розетка", Type: "plug"},
		{ID: "t1", Name: "Термостат", Type: "thermostat"},
		{ID: "s1", Name: "Датчик", Type: "sensor"},
		{ID: "x1", Name: "Неизвестный", Type: "whatever"},
	}

	_, accs, bindings := buildBridge(api, devices)

	if len(accs) != 4 {
		t.Fatalf("ждём 4 аксессуара (unknown пропущен), пришло %d", len(accs))
	}
	if len(bindings) != 4 {
		t.Fatalf("ждём 4 биндинга, пришло %d", len(bindings))
	}

	if bindings["l1"].deviceType != "light" || bindings["l1"].brightness == nil {
		t.Fatalf("у лампы должен быть on+brightness: %+v", bindings["l1"])
	}
	if bindings["p1"].on == nil {
		t.Fatal("у розетки должен быть on")
	}
	if bindings["t1"].targetTemp == nil || bindings["t1"].currentTemp == nil {
		t.Fatal("у термостата должны быть current/target temp")
	}
	if bindings["s1"].humidity == nil || bindings["s1"].currentTemp == nil {
		t.Fatal("у датчика должны быть temp+humidity")
	}
	if _, ok := bindings["x1"]; ok {
		t.Fatal("неизвестный тип должен пропускаться")
	}
}
