// Package config reads the service's configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// Defaults for every variable that has a safe one. The database host and
// credentials have none: guessing them fails later and with a worse message.
const (
	defaultServerAddress   = ":8080"
	defaultDBPort          = "5432"
	defaultDBSSLMode       = "disable"
	defaultMigrationsPath  = "migrations"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = 10 * time.Second
)

// Config is the whole of the service's configuration.
type Config struct {
	ServerAddress   string
	Database        Database
	MigrationsPath  string
	LogLevel        string
	ShutdownTimeout time.Duration
}

// Database is everything needed to reach Postgres.
type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN builds the connection string. It goes through net/url rather than fmt.Sprintf: a password containing ':',
// '/' or '?' would otherwise be read as URL structure.
func (d Database) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     d.Host + ":" + d.Port,
		Path:     "/" + d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}
	return u.String()
}

// MissingError names every absent variable, not the first: startup failures
// arrive one restart at a time.
type MissingError struct {
	Names []string
}

func (e *MissingError) Error() string {
	return "missing required environment variables: " + strings.Join(e.Names, ", ")
}

// Load resolves the configuration, or reports everything wrong with it.
func Load() (Config, error) {
	var missing []string

	record := func(name, value string) string {
		if value == "" {
			missing = append(missing, name)
		}
		return value
	}

	required := func(name string) string { return record(name, strings.TrimSpace(os.Getenv(name))) }
	// Not trimmed: whitespace is legitimate in a generated secret.
	requiredSecret := func(name string) string { return record(name, os.Getenv(name)) }

	db := Database{
		Host:     required("DB_HOST"),
		Port:     optional("DB_PORT", defaultDBPort),
		User:     required("DB_USER"),
		Password: requiredSecret("DB_PASSWORD"),
		Name:     required("DB_NAME"),
		SSLMode:  optional("DB_SSLMODE", defaultDBSSLMode),
	}

	cfg := Config{
		ServerAddress:  optional("SERVER_ADDRESS", defaultServerAddress),
		Database:       db,
		MigrationsPath: optional("MIGRATIONS_PATH", defaultMigrationsPath),
		LogLevel:       optional("LOG_LEVEL", defaultLogLevel),
	}

	var problems []error
	if len(missing) > 0 {
		problems = append(problems, &MissingError{Names: missing})
	}

	timeout, err := time.ParseDuration(optional("SHUTDOWN_TIMEOUT", defaultShutdownTimeout.String()))
	if err != nil {
		problems = append(problems, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err))
	}
	cfg.ShutdownTimeout = timeout

	if len(problems) > 0 {
		return Config{}, errors.Join(problems...)
	}

	return cfg, nil
}

func optional(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
