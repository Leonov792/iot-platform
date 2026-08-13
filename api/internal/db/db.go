package db

import (
	"context"
	"embed"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Connect открывает пул к постгресу.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	// без этого пинга сервер стартовал как ни в чём не бывало,
	// а падал уже на первом запросе. компилятор не поймает, вот и лови сам
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

// Migrate накатывает все sql-файлы из папки migrations по порядку.
// Схема простая: таблица schema_migrations, каждый файл — своя транзакция.
// Откатов нет, но для начала сойдёт. TODO: нормальные up/down, если доживу
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return err
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`, name).Scan(&applied)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		// embed ругался, пока файлы не легли ровно в подпапку migrations.
		// читаем по одному, чтобы не тащить всё в память
		sqlBytes, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(name) VALUES($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}
