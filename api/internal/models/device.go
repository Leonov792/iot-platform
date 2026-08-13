package models

import "time"

// Device — устройство, которое шлёт телеметрию.
// Status пока строкой, потом наверное сделаю enum, но руки не доходят
type Device struct {
	ID        string
	Name      string
	Type      string
	Status    string
	CreatedAt time.Time
	LastSeen  time.Time // обновляется, когда прилетает пакет
}
