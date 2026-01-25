package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if migrationsDir == "" {
		migrationsDir = "/migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			workDir, _ := os.Getwd()
			migrationsDir = filepath.Join(workDir, "internal", "db", "migrations")
		}
	}

	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve migrations path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found: %s", absPath)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite driver: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", absPath)
	if !strings.HasPrefix(absPath, "/") {
		sourceURL = fmt.Sprintf("file:///%s", absPath)
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := runMigrationsUp(m); err != nil {
		return err
	}
	return nil
}

func runMigrationsUp(m *migrate.Migrate) error {
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		// Handle dirty database state (e.g. migration failed mid-run, process killed)
		var dirtyErr migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			lastGood := dirtyErr.Version - 1
			if lastGood < 0 {
				lastGood = 0
			}
			if forceErr := m.Force(lastGood); forceErr != nil {
				return fmt.Errorf("failed to run migrations: %w; failed to force version %d: %v", err, lastGood, forceErr)
			}
			// Retry migrations from last known good version
			if retryErr := m.Up(); retryErr != nil && !errors.Is(retryErr, migrate.ErrNoChange) {
				return fmt.Errorf("failed to run migrations after dirty recovery: %w", retryErr)
			}
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
