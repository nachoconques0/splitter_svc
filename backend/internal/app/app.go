// Package app wires configuration, the database, the schema and the server.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nachoconques0/splitter_svc/backend/internal/config"
	billcontroller "github.com/nachoconques0/splitter_svc/backend/internal/controller/bill"
	personcontroller "github.com/nachoconques0/splitter_svc/backend/internal/controller/person"
	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	billrepository "github.com/nachoconques0/splitter_svc/backend/internal/repository/bill"
	personrepository "github.com/nachoconques0/splitter_svc/backend/internal/repository/person"
	billservice "github.com/nachoconques0/splitter_svc/backend/internal/service/bill"
	personservice "github.com/nachoconques0/splitter_svc/backend/internal/service/person"
)

// Run starts the service and blocks until ctx is cancelled.
func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	db, err := openDatabase(ctx, logger, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	// Between the database being reachable and the listener opening, so no request
	// can arrive against a half-written schema.
	if err := migrateUp(ctx, logger, cfg.Database, cfg.MigrationsPath); err != nil {
		return err
	}

	bills := billcontroller.New(logger, billservice.New(billrepository.New(db)))
	people := personcontroller.New(logger, personservice.New(personrepository.New(db)))

	gin.SetMode(gin.ReleaseMode)
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: splitterhttp.NewRouter(logger, db, bills, people),
	}

	return serve(ctx, logger, server, cfg.ShutdownTimeout)
}

// serve listens, and shuts down gracefully when ctx is cancelled.
func serve(ctx context.Context, logger *slog.Logger, server *http.Server, shutdownTimeout time.Duration) error {
	listening := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "listening", slog.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listening <- fmt.Errorf("serving: %w", err)
			return
		}
		listening <- nil
	}()

	select {
	case err := <-listening:
		return err
	case <-ctx.Done():
		logger.InfoContext(ctx, "shutting down", slog.Duration("grace", shutdownTimeout))

		// A fresh context: ctx is cancelled, and passing it on would abort the
		// in-flight requests we are trying to let finish.
		grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(grace); err != nil {
			return fmt.Errorf("shutting down: %w", err)
		}
		return nil
	}
}

// newLogger builds the structured logger at the configured level.
func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		parsed = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: parsed}))
}
