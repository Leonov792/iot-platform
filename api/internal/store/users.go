package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iot-platform/api/internal/models"
)

// ErrNotFound — возвращаем, когда записи нет, чтобы хендлеры отличали
// "нет юзера" от "база отвалилась"
var ErrNotFound = errors.New("не найдено")

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

const userCols = `id, email, password_hash, role, home_id, schedule, created_at`

func scanUser(row pgx.Row) (models.User, error) {
	var u models.User
	var schedule []byte
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.HomeID, &schedule, &u.CreatedAt)
	if err != nil {
		return u, err
	}
	if len(schedule) > 0 {
		_ = json.Unmarshal(schedule, &u.Schedule)
	}
	if u.Schedule == nil {
		u.Schedule = []models.ScheduleEntry{}
	}
	return u, nil
}

func (s *UserStore) Create(ctx context.Context, u models.User) error {
	schedule, err := json.Marshal(u.Schedule)
	if err != nil {
		return err
	}
	if u.Role == "" {
		u.Role = "owner"
	}
	if u.HomeID == "" {
		u.HomeID = u.ID
	}
	_, err = s.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, home_id, schedule, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		u.ID, u.Email, u.PasswordHash, u.Role, u.HomeID, schedule, u.CreatedAt)
	return err
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (models.User, error) {
	u, err := scanUser(s.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE email=$1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *UserStore) GetByID(ctx context.Context, id string) (models.User, error) {
	u, err := scanUser(s.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// ListByHome возвращает всех членов «дома» (включая владельца).
func (s *UserStore) ListByHome(ctx context.Context, homeID string) ([]models.User, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+userCols+` FROM users WHERE home_id=$1 ORDER BY created_at ASC`, homeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetRole меняет роль пользователя (только владелец дома).
func (s *UserStore) SetRole(ctx context.Context, userID, homeID, role string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE users SET role=$1 WHERE id=$2 AND home_id=$3`, role, userID, homeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetSchedule обновляет расписание доступа персонала.
func (s *UserStore) SetSchedule(ctx context.Context, userID, homeID string, schedule []models.ScheduleEntry) error {
	b, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE users SET schedule=$1 WHERE id=$2 AND home_id=$3`, b, userID, homeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
