package models

import "time"

// User — аккаунт. Храним хэш пароля, а не сам пароль, само собой.
// Role — owner | family | staff. HomeID — id владельца «дома» (у owner = свой id).
type User struct {
	ID           string          `json:"id"`
	Email        string          `json:"email"`
	PasswordHash string          `json:"-"` // в ответах это поле никогда не светим
	Role         string          `json:"role"`
	HomeID       string          `json:"home_id"`
	Schedule     []ScheduleEntry `json:"schedule"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ScheduleEntry — окно доступа персонала к зоне (pool/gym).
// Days — дни недели по time.Weekday (0=воскресенье .. 6=суббота).
type ScheduleEntry struct {
	Zone  string `json:"zone"`
	Days  []int  `json:"days"`
	Start string `json:"start"` // "08:00"
	End   string `json:"end"`   // "20:00"
}
