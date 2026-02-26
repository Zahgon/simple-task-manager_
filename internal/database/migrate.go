package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB, migrationsPath string, log logger.Logger) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("creating migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("creating migration instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("running migrations: %w", err)
	}

	log.Info("Migrations applied successfully")
	return nil
}
