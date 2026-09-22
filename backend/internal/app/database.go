package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	// lib/pq is the driver golang-migrate's postgres backend is built on, so the
	// application and the migrations share one driver and cannot skew apart.
	_ "github.com/lib/pq"

	"github.com/nachoconques0/splitter_svc/backend/internal/config"
)

const (
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute

	// Compose reports Postgres healthy the moment it accepts connections, which
	// can still be a moment before it accepts ours.
	databaseReadyTimeout  = 30 * time.Second
	databaseRetryInterval = time.Second
)

// openDatabase connects, waiting for Postgres to start answering.
func openDatabase(ctx context.Context, logger *slog.Logger, cfg config.Database) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening the database: %w", err)
	}

	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)

	if err := waitForDatabase(ctx, logger, db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func waitForDatabase(ctx context.Context, logger *slog.Logger, db *sql.DB) error {
	deadline, cancel := context.WithTimeout(ctx, databaseReadyTimeout)
	defer cancel()

	ticker := time.NewTicker(databaseRetryInterval)
	defer ticker.Stop()

	for {
		err := db.PingContext(deadline)
		if err == nil {
			return nil
		}
		logger.InfoContext(ctx, "waiting for the database", slog.Any("error", err))

		select {
		case <-deadline.Done():
			return fmt.Errorf("database not reachable within %s: %w", databaseReadyTimeout, err)
		case <-ticker.C:
		}
	}
}
