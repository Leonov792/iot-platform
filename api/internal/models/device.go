package models

import "time"

// Device — устройство в умном доме.
// Status пока строкой, потом наверное сделаю enum, но руки не доходят.
// State — управляемое состояние: у лампы {on:true,brightness:80}, у термостата {target_temp:22}.
type Device struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`   // light | plug | thermostat | sensor
	Status    string         `json:"status"` // online | offline
	Room      string         `json:"room"`
	Zone      string         `json:"zone"` // home | pool | gym — для RBAC и сценариев
	State     map[string]any `json:"state"`
	OwnerID   string         `json:"owner_id"`
	CreatedAt time.Time      `json:"created_at"`
	LastSeen  time.Time      `json:"last_seen"`
}
