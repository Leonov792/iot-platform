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

func (s *DeviceStore) List(ctx context.Context, ownerID string) ([]models.Device, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, type, status, room, zone, state, owner_id, created_at, last_seen
		 FROM devices WHERE owner_id=$1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]models.Device, 0)
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Status, &d.Room, &d.Zone, &d.State, &d.OwnerID, &d.CreatedAt, &d.LastSeen); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *DeviceStore) Create(ctx context.Context, d models.Device) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO devices (id, name, type, status, room, zone, state, owner_id, created_at, last_seen)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		d.ID, d.Name, d.Type, d.Status, d.Room, d.Zone, d.State, d.OwnerID, d.CreatedAt, d.LastSeen)
	return err
}

func (s *DeviceStore) Get(ctx context.Context, id string) (models.Device, error) {
	var d models.Device
	err := s.db.QueryRow(ctx,
		`SELECT id, name, type, status, room, zone, state, owner_id, created_at, last_seen FROM devices WHERE id=$1`, id).
		Scan(&d.ID, &d.Name, &d.Type, &d.Status, &d.Room, &d.Zone, &d.State, &d.OwnerID, &d.CreatedAt, &d.LastSeen)
	return d, err
}

func (s *DeviceStore) Update(ctx context.Context, ownerID string, d models.Device) error {
	_, err := s.db.Exec(ctx,
		`UPDATE devices SET name=$1, type=$2, status=$3, room=$4, zone=$5, state=$6 WHERE id=$7 AND owner_id=$8`,
		d.Name, d.Type, d.Status, d.Room, d.Zone, d.State, d.ID, ownerID)
	return err
}

func (s *DeviceStore) Delete(ctx context.Context, ownerID, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM devices WHERE id=$1 AND owner_id=$2`, id, ownerID)
	return err
}

// OwnedBy — проверяем, что устройство принадлежит юзеру. юзается в истории и командах
func (s *DeviceStore) OwnedBy(ctx context.Context, deviceID, ownerID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM devices WHERE id=$1 AND owner_id=$2)`, deviceID, ownerID).Scan(&exists)
	return exists, err
}

// Touch — обновляем last_seen, когда прилетела телеметрия
func (s *DeviceStore) Touch(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `UPDATE devices SET last_seen=now() WHERE id=$1`, id)
	return err
}

// SetDeviceTokenHash — сохраняем sha256-хэш токена устройства (сам токен не храним)
func (s *DeviceStore) SetDeviceTokenHash(ctx context.Context, id, ownerID, hash string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE devices SET device_token_hash=$1 WHERE id=$2 AND owner_id=$3`, hash, id, ownerID)
	return err
}

// GetDeviceTokenHash возвращает хэш токена устройства по id (для проверки на хендшейке).
// если устройства нет или хэш пустой — отдаёт ErrNotFound.
func (s *DeviceStore) GetDeviceTokenHash(ctx context.Context, id string) (string, error) {
	var hash *string
	err := s.db.QueryRow(ctx,
		`SELECT device_token_hash FROM devices WHERE id=$1`, id).Scan(&hash)
	if err != nil {
		return "", err
	}
	if hash == nil || *hash == "" {
		return "", ErrNotFound
	}
	return *hash, nil
}
