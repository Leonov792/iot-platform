package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommandRecord — одна команда из undo-стека.
type CommandRecord struct {
	Action    string         `json:"action"`
	Previous  map[string]any `json:"previous_state"`
	CreatedAt time.Time      `json:"created_at"`
}

// CommandHistoryStore — undo-стек команд (Event Sourcing).
type CommandHistoryStore struct {
	db *pgxpool.Pool
}

func NewCommandHistoryStore(db *pgxpool.Pool) *CommandHistoryStore {
	return &CommandHistoryStore{db: db}
}

// Append пишет команду с предыдущим состоянием устройства.
func (s *CommandHistoryStore) Append(ctx context.Context, deviceID, userID, action string, previous map[string]any) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO command_history (device_id, user_id, action, previous_state) VALUES ($1,$2,$3,$4)`,
		deviceID, userID, action, previous)
	return err
}

// Last возвращает последнюю команду для устройства от этого пользователя.
func (s *CommandHistoryStore) Last(ctx context.Context, deviceID, userID string) (CommandRecord, error) {
	var rec CommandRecord
	err := s.db.QueryRow(ctx,
		`SELECT action, previous_state, created_at FROM command_history
		 WHERE device_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT 1`,
		deviceID, userID).Scan(&rec.Action, &rec.Previous, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return rec, ErrNotFound
	}
	return rec, err
}
