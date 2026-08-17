package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HRVSample — одна точка вариабельности сердечного ритма.
type HRVSample struct {
	Value float64   `json:"value"`
	TS    time.Time `json:"ts"`
}

// HRVStore — хранение HRV-отсчётов.
type HRVStore struct {
	db *pgxpool.Pool
}

func NewHRVStore(db *pgxpool.Pool) *HRVStore {
	return &HRVStore{db: db}
}

func (s *HRVStore) Insert(ctx context.Context, userID string, value float64) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO hrv_samples (user_id, value) VALUES ($1,$2)`, userID, value)
	return err
}

// Since возвращает отсчёты HRV после указанного момента (для wellness-сервиса).
func (s *HRVStore) Since(ctx context.Context, userID string, since time.Time) ([]HRVSample, error) {
	rows, err := s.db.Query(ctx,
		`SELECT value, ts FROM hrv_samples WHERE user_id=$1 AND ts >= $2 ORDER BY ts ASC`,
		userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]HRVSample, 0)
	for rows.Next() {
		var s HRVSample
		if err := rows.Scan(&s.Value, &s.TS); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
