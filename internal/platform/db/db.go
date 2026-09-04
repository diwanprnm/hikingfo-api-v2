// Package db owns the Postgres pool and the goose migration runner.
// Contexts receive the *pgxpool.Pool from the composition root and build their
// own sqlc adapters over it; this package never imports context packages.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Options configures the pool.
type Options struct {
	DSN      string
	MaxConns int32
	MinConns int32
}

// Open creates a pgx pool and verifies connectivity.
func Open(ctx context.Context, o Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(o.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	if o.MaxConns > 0 {
		cfg.MaxConns = o.MaxConns
	}
	if o.MinConns > 0 {
		cfg.MinConns = o.MinConns
	}
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}
