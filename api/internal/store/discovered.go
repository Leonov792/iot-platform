package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DiscoveredDevice — найденное в сети устройство.
type DiscoveredDevice struct {
	ID        int64  `json:"id"`
	MAC       string `json:"mac,omitempty"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Service   string `json:"service"`
	Status    string `json:"status"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

// DiscoveryStore — хранение найденных устройств.
type DiscoveryStore struct {
	db *pgxpool.Pool
}

func NewDiscoveryStore(db *pgxpool.Pool) *DiscoveryStore {
	return &DiscoveryStore{db: db}
}

// Upsert — регистрирует устройство (по ip+port). Возвращает true, если устройство новое.
func (s *DiscoveryStore) Upsert(ctx context.Context, ip string, port int, service string) (bool, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO discovered_devices (ip, port, service)
		VALUES ($1, $2, $3)
		ON CONFLICT (ip, port)
		DO UPDATE SET last_seen = now()
		RETURNING (xmax = 0) AS inserted`, ip, port, service)

	var inserted bool
	if err := row.Scan(&inserted); err != nil {
		return false, err
	}
	return inserted, nil
}

// List возвращает найденные устройства, опционально фильтруя по статусу.
func (s *DiscoveryStore) List(ctx context.Context, status string) ([]DiscoveredDevice, error) {
	q := `SELECT id, COALESCE(mac, ''), ip, port, service, status,
		to_char(first_seen, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		to_char(last_seen, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM discovered_devices`
	args := []any{}
	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY last_seen DESC`

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DiscoveredDevice, 0)
	for rows.Next() {
		var d DiscoveredDevice
		if err := rows.Scan(&d.ID, &d.MAC, &d.IP, &d.Port, &d.Service, &d.Status, &d.FirstSeen, &d.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetStatus меняет статус (approved/ignored). Возвращает ErrNotFound, если нет записи.
func (s *DiscoveryStore) SetStatus(ctx context.Context, id int64, status string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE discovered_devices SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
