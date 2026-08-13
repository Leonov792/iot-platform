package models

import "time"

// Telemetry — одна запись с датчика. Payload это распарсенный json от rust-парсера,
// типа {"temp":21.5,"humidity":40,"battery":87}
type Telemetry struct {
	ID       int64          `json:"id"`
	DeviceID string         `json:"device_id"`
	TS       time.Time      `json:"ts"`
	Payload  map[string]any `json:"payload"`
}
