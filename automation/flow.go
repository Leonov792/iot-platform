package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"iot-platform/automation/rules"
)

// Flow — граф автоматизации в формате flow_json (ноды + рёбра, Node-RED-стиль).
// Один flow транслируется в одно правило движка rules.Rule.
type Flow struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node — нода графа. Type: trigger (условие) | action (действие).
type Node struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	DeviceID string  `json:"device_id"`
	Field    string  `json:"field,omitempty"`     // trigger: поле телеметрии
	Op       string  `json:"op,omitempty"`        // trigger: gt|lt|gte|lte|eq|neq
	Value    float64 `json:"value,omitempty"`     // trigger: порог
	Relay    string  `json:"relay,omitempty"`     // action: имя реле
	On       *bool   `json:"on,omitempty"`        // action: вкл/выкл (дефолт true)
	OffAfter string  `json:"off_after,omitempty"` // action: выключить через N (напр. "30s")
	Cooldown string  `json:"cooldown,omitempty"`  // trigger: интервал между срабатываниями
}

// Edge — связь ноды trigger -> action.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

var validOps = map[string]bool{
	"gt": true, "lt": true, "gte": true, "lte": true, "eq": true, "neq": true,
}

// ParseFlow разбирает flow_json.
func ParseFlow(data []byte) (*Flow, error) {
	var f Flow
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// ValidateFlow проверяет схему flow и возвращает список ошибок (пустой = валиден).
func ValidateFlow(f *Flow) []string {
	var errs []string

	if f == nil {
		return []string{"flow пустой"}
	}
	if f.ID == "" {
		errs = append(errs, "id обязателен")
	}

	known := make(map[string]bool, len(f.Nodes))
	triggers, actions := 0, 0

	for _, n := range f.Nodes {
		known[n.ID] = true
		switch n.Type {
		case "trigger":
			triggers++
			if n.DeviceID == "" {
				errs = append(errs, fmt.Sprintf("trigger %q: device_id обязателен", n.ID))
			}
			if n.Field == "" {
				errs = append(errs, fmt.Sprintf("trigger %q: field обязателен", n.ID))
			}
			if !validOps[n.Op] {
				errs = append(errs, fmt.Sprintf("trigger %q: неизвестный op %q", n.ID, n.Op))
			}
		case "action":
			actions++
			if n.DeviceID == "" {
				errs = append(errs, fmt.Sprintf("action %q: device_id обязателен", n.ID))
			}
			if n.Relay == "" {
				errs = append(errs, fmt.Sprintf("action %q: relay обязателен", n.ID))
			}
		default:
			errs = append(errs, fmt.Sprintf("нода %q: неизвестный type %q", n.ID, n.Type))
		}
	}

	if triggers != 1 {
		errs = append(errs, fmt.Sprintf("должен быть ровно один trigger, найдено %d", triggers))
	}
	if actions == 0 {
		errs = append(errs, "нужна хотя бы одна action-нода")
	}

	for _, e := range f.Edges {
		if !known[e.From] {
			errs = append(errs, fmt.Sprintf("ребро из неизвестной ноды %q", e.From))
		}
		if !known[e.To] {
			errs = append(errs, fmt.Sprintf("ребро в неизвестную ноду %q", e.To))
		}
	}

	return errs
}

// FlowToRule транслирует flow в rules.Rule (trigger -> условие, action-ноды -> действия).
func FlowToRule(f *Flow) (rules.Rule, error) {
	var trigger *Node
	actionNodes := make([]Node, 0)

	for i := range f.Nodes {
		n := f.Nodes[i]
		switch n.Type {
		case "trigger":
			trigger = &n
		case "action":
			actionNodes = append(actionNodes, n)
		}
	}

	if trigger == nil || len(actionNodes) == 0 {
		return rules.Rule{}, errors.New("невалидный flow: нужен trigger и action")
	}

	actions := make([]rules.Action, 0, len(actionNodes))
	for _, a := range actionNodes {
		on := true
		if a.On != nil {
			on = *a.On
		}
		actions = append(actions, rules.Action{
			Type:     "modbus_write",
			DeviceID: a.DeviceID,
			Relay:    a.Relay,
			Value:    on,
			OffAfter: a.OffAfter,
		})
	}

	cooldown := trigger.Cooldown
	if cooldown == "" {
		cooldown = "1m"
	}

	return rules.Rule{
		ID:   f.ID,
		Name: f.Name,
		Condition: rules.Condition{
			DeviceID: trigger.DeviceID,
			Field:    trigger.Field,
			Op:       trigger.Op,
			Value:    trigger.Value,
		},
		Actions:  actions,
		Cooldown: cooldown,
	}, nil
}
