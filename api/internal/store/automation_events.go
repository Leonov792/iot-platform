package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AutomationEventStore — лог срабатываний правил автоматизации.
type AutomationEventStore struct {
	db *pgxpool.Pool
}

func NewAutomationEventStore(db *pgxpool.Pool) *AutomationEventStore {
	return &AutomationEventStore{db: db}
}

// InsertEvent пишет событие срабатывания правила.
func (s *AutomationEventStore) InsertEvent(ctx context.Context, ruleID, ruleName, deviceID string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO automation_events (rule_id, rule_name, device_id) VALUES ($1,$2,$3)`,
		ruleID, ruleName, deviceID)
	return err
}
