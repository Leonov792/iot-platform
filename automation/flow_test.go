package main

import "testing"

func boolPtr(b bool) *bool { return &b }

func TestValidateFlowValid(t *testing.T) {
	f := &Flow{
		ID:   "fill",
		Name: "Долив воды",
		Nodes: []Node{
			{ID: "n1", Type: "trigger", DeviceID: "pool-pump", Field: "water_level", Op: "lt", Value: 20},
			{ID: "n2", Type: "action", DeviceID: "pool-pump", Relay: "fill_valve", On: boolPtr(true)},
		},
		Edges: []Edge{{From: "n1", To: "n2"}},
	}
	if errs := ValidateFlow(f); len(errs) != 0 {
		t.Fatalf("валидный flow не должен давать ошибок: %v", errs)
	}
}

func TestValidateFlowBadJSON(t *testing.T) {
	if _, err := ParseFlow([]byte("{not json")); err == nil {
		t.Fatal("битый json должен давать ошибку")
	}
}

func TestValidateFlowUnknownOp(t *testing.T) {
	f := &Flow{
		ID: "f",
		Nodes: []Node{
			{ID: "n1", Type: "trigger", DeviceID: "d", Field: "x", Op: "bogus"},
			{ID: "n2", Type: "action", DeviceID: "d", Relay: "r"},
		},
	}
	errs := ValidateFlow(f)
	if len(errs) == 0 {
		t.Fatal("неизвестный op должен давать ошибку")
	}
}

func TestValidateFlowDanglingEdge(t *testing.T) {
	f := &Flow{
		ID: "f",
		Nodes: []Node{
			{ID: "n1", Type: "trigger", DeviceID: "d", Field: "x", Op: "gt"},
			{ID: "n2", Type: "action", DeviceID: "d", Relay: "r"},
		},
		Edges: []Edge{{From: "n1", To: "ghost"}},
	}
	errs := ValidateFlow(f)
	if len(errs) == 0 {
		t.Fatal("ребро в неизвестную ноду должно давать ошибку")
	}
}

func TestValidateFlowNoTrigger(t *testing.T) {
	f := &Flow{
		ID:    "f",
		Nodes: []Node{{ID: "n2", Type: "action", DeviceID: "d", Relay: "r"}},
	}
	errs := ValidateFlow(f)
	if len(errs) == 0 {
		t.Fatal("без trigger должна быть ошибка")
	}
}

func TestFlowToRule(t *testing.T) {
	f := &Flow{
		ID:   "fill",
		Name: "Долив",
		Nodes: []Node{
			{ID: "n1", Type: "trigger", DeviceID: "pool-pump", Field: "water_level", Op: "lt", Value: 20, Cooldown: "2m"},
			{ID: "n2", Type: "action", DeviceID: "pool-pump", Relay: "fill_valve", On: boolPtr(true), OffAfter: "30s"},
		},
		Edges: []Edge{{From: "n1", To: "n2"}},
	}

	r, err := FlowToRule(f)
	if err != nil {
		t.Fatalf("не транслировалось: %v", err)
	}
	if r.ID != "fill" || r.Name != "Долив" {
		t.Fatalf("id/name не совпали: %+v", r)
	}
	if r.Condition.DeviceID != "pool-pump" || r.Condition.Field != "water_level" || r.Condition.Op != "lt" || r.Condition.Value != 20 {
		t.Fatalf("условие не то: %+v", r.Condition)
	}
	if len(r.Actions) != 1 {
		t.Fatalf("ждём 1 действие, пришло %d", len(r.Actions))
	}
	a := r.Actions[0]
	if a.Relay != "fill_valve" || a.Value != true || a.OffAfter != "30s" {
		t.Fatalf("действие не то: %+v", a)
	}
	if r.Cooldown != "2m" {
		t.Fatalf("cooldown не сохранился: %q", r.Cooldown)
	}
}

func TestFlowToRuleDefaultCooldown(t *testing.T) {
	f := &Flow{
		ID: "f",
		Nodes: []Node{
			{ID: "n1", Type: "trigger", DeviceID: "d", Field: "x", Op: "gt"},
			{ID: "n2", Type: "action", DeviceID: "d", Relay: "r"},
		},
	}
	r, err := FlowToRule(f)
	if err != nil {
		t.Fatalf("не транслировалось: %v", err)
	}
	if r.Cooldown != "1m" {
		t.Fatalf("дефолтный cooldown должен быть 1m, пришло %q", r.Cooldown)
	}
}
