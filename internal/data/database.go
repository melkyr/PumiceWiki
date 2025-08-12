package data

import (
	"fmt"
	"go-wiki-app/internal/config"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

// NewDB creates a new database connection pool.
func NewDB(cfg config.DBConfig) (*sqlx.DB, error) {
	// sqlx.Connect opens a connection and pings it to verify it's alive.
	db, err := sqlx.Connect("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings to prevent overwhelming the database.
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMins) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTimeMins) * time.Minute)

	return db, nil
}

// ApplyMigrations runs all up migrations.
func ApplyMigrations(dsn string, migrationsPath string) error {
	// The migrate library needs the DSN in a URL format.
	// e.g., "mysql://user:pass@tcp(host:port)/dbname"
	migrateDSN := fmt.Sprintf("mysql://%s", dsn)

	// To ensure the path is correctly interpreted by the migrate library,
	// convert it to an absolute path and then format it as a file URL.
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for migrations: %w", err)
	}
	sourceURL := fmt.Sprintf("file://%s", absPath)

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
