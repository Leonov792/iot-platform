package store

import (
	"context"
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

func (s *UserStore) Create(ctx context.Context, u models.User) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ($1,$2,$3,$4)`,
		u.ID, u.Email, u.PasswordHash, u.CreatedAt)
	return err
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}
