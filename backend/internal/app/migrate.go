package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/nachoconques0/splitter_svc/backend/internal/config"
)

// migrateUp brings the schema up before the server listens, on a pool of its
// own: postgres.WithInstance holds a connection for the driver's lifetime and
// only gives it back by closing the pool it came from.
func migrateUp(ctx context.Context, logger *slog.Logger, cfg config.Database, path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving migrations path %q: %w", path, err)
	}

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return fmt.Errorf("opening the database for migrations: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("preparing the migration driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://"+absolute, "postgres", driver)
	if err != nil {
		return fmt.Errorf("loading migrations from %s: %w", absolute, err)
	}
	defer func() {
		if sourceErr, dbErr := migrator.Close(); sourceErr != nil || dbErr != nil {
			logger.WarnContext(ctx, "closing the migrator", slog.Any("source_error", sourceErr), slog.Any("database_error", dbErr))
		}
	}()

	// golang-migrate would refuse a dirty schema anyway; checking first lets the
	// message name the version that needs a decision.
	version, dirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("reading the schema version: %w", err)
	}
	if dirty {
		return fmt.Errorf("schema is dirty at version %d: a previous migration failed part-way and must be resolved by hand before the service can start", version)
	}

	switch err := migrator.Up(); {
	case errors.Is(err, migrate.ErrNoChange):
		logger.InfoContext(ctx, "schema is up to date", slog.Uint64("version", uint64(version)))
	case err != nil:
		return fmt.Errorf("applying migrations: %w", err)
	default:
		applied, _, versionErr := migrator.Version()
		if versionErr != nil {
			applied = 0
		}
		logger.InfoContext(ctx, "schema migrated", slog.Uint64("from_version", uint64(version)), slog.Uint64("to_version", uint64(applied)))
	}

	return nil
}
