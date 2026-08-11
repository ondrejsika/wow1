// Package store provides the PostgreSQL-backed data access layer for wow1.
package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}

// Migrate applies all pending migrations from the embedded migrations
// filesystem to the database.
func Migrate(databaseURL string, migrationsFS embed.FS) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	// The golang-migrate pgx/v5 driver registers itself under the "pgx5"
	// scheme rather than "postgres"/"postgresql".
	migrateURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	migrateURL.Scheme = "pgx5"

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL.String())
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
