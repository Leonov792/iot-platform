package models

import "time"

// Device — устройство, которое шлёт телеметрию.
// Status пока строкой, потом наверное сделаю enum, но руки не доходят
type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"` // обновляется, когда прилетает пакет
}
