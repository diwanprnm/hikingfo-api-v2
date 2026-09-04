// Package integration hosts testcontainers-based integration tests that run
// against a real Postgres with the goose migrations applied. Skipped with a
// short message when Docker is unavailable (CI always has Docker).
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"hikingfo/backend/internal/platform/db"
)

// postgresImage pins the integration-test image (matches compose.yaml).
const postgresImage = "postgres:16-alpine"

// TestMain reports Docker availability once per run; individual tests still
// skip individually (t.Skipf) so CI without Docker stays green.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// newPostgres starts a throwaway Postgres container and runs the migrations
// from backend/migrations. The pool is closed automatically via t.Cleanup.
func newPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	container, err := tcpostgres.Run(ctx, postgresImage)
	if err != nil {
		t.Skipf("integration test skipped: could not start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	// Windows Docker Desktop reports the container ready while postgres is
	// still finishing its init restart — early connections get EOF'd. Retry.
	var pool *pgxpool.Pool
	for i := 0; i < 15; i++ {
		pool, err = db.Open(ctx, db.Options{DSN: dsn, MaxConns: 8})
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
	if err != nil {
		t.Fatalf("open pool after retries: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// runMigrations applies backend/migrations via the same goose runner the API
// uses at boot.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("migrations dir: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqlDB, dir)
}
