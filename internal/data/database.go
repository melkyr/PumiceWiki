package data

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// NewDB creates a new database connection pool.
func NewDB(dsn string) (*sqlx.DB, error) {
	// sqlx.Connect opens a connection and pings it to verify it's alive.
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// ApplyMigrations runs all up migrations.
func ApplyMigrations(dsn string, migrationsPath string) error {
	// The migrate library needs the DSN in a URL format.
	// e.g., "sqlite3://wiki.db"
	migrateDSN := fmt.Sprintf("sqlite3://%s", dsn)

	// The migrationsPath needs to be in the format "file://path/to/migrations"
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.New(sourceURL, migrateDSN)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Up applies all available up migrations.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
