package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate runs the goose migrations in dir against the pool. It is called by
// the composition root at boot (the API container is the migration runner;
// research.md §1, plan.md → cmd/api "migrations").
//
// goose's Up API consumes a *sql.DB; we adapt the pgx pool with the stdlib
// bridge so migrations share the same underlying pool rather than opening a
// second connection set.
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if dir == "" {
		return fmt.Errorf("db: migrate: empty migrations dir")
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, dir); err != nil {
		return fmt.Errorf("db: goose up: %w", err)
	}
	return nil
}
