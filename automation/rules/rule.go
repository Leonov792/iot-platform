// Package rules — движок сценариев IF [Condition] THEN [Action] для умного дома.
package rules

// Rule — одно правило. Условие по телеметрии устройства, действия при срабатывании.
type Rule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Condition Condition `json:"condition"`
	Actions   []Action  `json:"actions"`
	Cooldown  string    `json:"cooldown,omitempty"` // минимальный интервал между срабатываниями, напр. "5m"
}

// Condition — сравнение поля телеметрии с порогом.
type Condition struct {
	DeviceID string  `json:"device_id"`
	Field    string  `json:"field"`
	Op       string  `json:"op"` // gt | lt | gte | lte | eq | neq
	Value    float64 `json:"value"`
}

// Action — что сделать. type: modbus_write | command.
type Action struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
	Relay    string `json:"relay,omitempty"`     // для modbus_write
	Action   string `json:"action,omitempty"`    // для command
	Value    any    `json:"value,omitempty"`     // значение (bool/число/строка)
	OffAfter string `json:"off_after,omitempty"` // "30s" — выключить реле через N секунд
}

// ValueBool приводит Value к bool для реле/клапанов.
func (a Action) ValueBool() bool {
	switch v := a.Value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1" || v == "on"
	case float64:
		return v != 0
	default:
		return false
	}
}
