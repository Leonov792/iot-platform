package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CommandLogStore пишет лог команд устройства — для предиктивного анализа.
type CommandLogStore struct {
	db *pgxpool.Pool
}

func NewCommandLogStore(db *pgxpool.Pool) *CommandLogStore {
	return &CommandLogStore{db: db}
}

// Insert — асинхронно (не критично для доставки команды). value — nil, если нет.
func (s *CommandLogStore) Insert(ctx context.Context, deviceID, userID, action string, value any) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO device_commands (device_id, user_id, action, value) VALUES ($1,$2,$3,$4)`,
		deviceID, userID, action, value)
	return err
}
