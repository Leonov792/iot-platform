package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"iot-platform/api/internal/models"
)

type DeviceStore struct {
	db *pgxpool.Pool
}

func NewDeviceStore(db *pgxpool.Pool) *DeviceStore {
	return &DeviceStore{db: db}
}

func (s *DeviceStore) List(ctx context.Context) ([]models.Device, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, type, status, created_at, last_seen FROM devices ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]models.Device, 0)
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Status, &d.CreatedAt, &d.LastSeen); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *DeviceStore) Create(ctx context.Context, d models.Device) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO devices (id, name, type, status, created_at, last_seen) VALUES ($1,$2,$3,$4,$5,$6)`,
		d.ID, d.Name, d.Type, d.Status, d.CreatedAt, d.LastSeen)
	return err
}

func (s *DeviceStore) Update(ctx context.Context, d models.Device) error {
	_, err := s.db.Exec(ctx,
		`UPDATE devices SET name=$1, type=$2, status=$3 WHERE id=$4`,
		d.Name, d.Type, d.Status, d.ID)
	return err
}

func (s *DeviceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM devices WHERE id=$1`, id)
	return err
}
