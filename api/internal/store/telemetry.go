package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"iot-platform/api/internal/models"
)

type TelemetryStore struct {
	db *pgxpool.Pool
}

func NewTelemetryStore(db *pgxpool.Pool) *TelemetryStore {
	return &TelemetryStore{db: db}
}

func (s *TelemetryStore) Insert(ctx context.Context, deviceID string, payload map[string]any) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO telemetry (device_id, ts, payload) VALUES ($1, now(), $2)`,
		deviceID, payload)
	return err
}

// List отдаёт последние limit точек, но в хронологическом порядке — для графика.
// берём последние N через подзапрос, потом разворачиваем по ts ASC
func (s *TelemetryStore) List(ctx context.Context, deviceID string, since time.Time, limit int) ([]models.Telemetry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(ctx,
		`SELECT id, device_id, ts, payload FROM (
			SELECT id, device_id, ts, payload FROM telemetry
			WHERE device_id=$1 AND ts >= $2
			ORDER BY ts DESC LIMIT $3
		) q ORDER BY ts ASC`,
		deviceID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Telemetry, 0, limit)
	for rows.Next() {
		var t models.Telemetry
		if err := rows.Scan(&t.ID, &t.DeviceID, &t.TS, &t.Payload); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
