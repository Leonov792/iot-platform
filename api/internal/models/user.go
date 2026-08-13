package models

import "time"

// User — аккаунт. Храним хэш пароля, а не сам пароль, само собой.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // в ответах это поле никогда не светим
	CreatedAt    time.Time `json:"created_at"`
}
