package config_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nachoconques0/splitter_svc/backend/internal/config"
)

// required lists every variable that has no safe default, so a test that wants a
// valid environment can set them all and a test about one missing variable can
// drop exactly one.
var required = map[string]string{
	"DB_HOST":     "db",
	"DB_USER":     "splitter",
	"DB_PASSWORD": "s3cret",
	"DB_NAME":     "splitter",
}

// optional lists every variable that has a named default, so clearEnv can blank
// it and the defaults tests cannot be coloured by the developer's own shell.
var optional = []string{"SERVER_ADDRESS", "DB_PORT", "DB_SSLMODE", "MIGRATIONS_PATH", "LOG_LEVEL", "SHUTDOWN_TIMEOUT"}

func clearEnv(t *testing.T) {
	t.Helper()
	for name := range required {
		t.Setenv(name, "")
	}
	for _, name := range optional {
		t.Setenv(name, "")
	}
}

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for name, value := range env {
		t.Setenv(name, value)
	}
}

func TestLoadReportsEveryMissingVariableAtOnce(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantMissing []string
	}{
		{
			name:        "empty environment names all of them",
			env:         map[string]string{},
			wantMissing: []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME"},
		},
		{
			name:        "a partial environment names only what is absent",
			env:         map[string]string{"DB_HOST": "db", "DB_USER": "splitter"},
			wantMissing: []string{"DB_PASSWORD", "DB_NAME"},
		},
		{
			name:        "a variable set to the empty string counts as missing",
			env:         map[string]string{"DB_HOST": "db", "DB_USER": "splitter", "DB_PASSWORD": "", "DB_NAME": "splitter"},
			wantMissing: []string{"DB_PASSWORD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Not parallel: t.Setenv forbids it, and these sub-tests share one process environment.
			clearEnv(t)
			setEnv(t, tt.env)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() succeeded, want an error naming %v", tt.wantMissing)
			}

			var missing *config.MissingError
			if !errors.As(err, &missing) {
				t.Fatalf("Load() error = %v, want a *config.MissingError", err)
			}
			if got, want := len(missing.Names), len(tt.wantMissing); got != want {
				t.Fatalf("Load() named %d missing variables (%v), want %d (%v)", got, missing.Names, want, tt.wantMissing)
			}
			for _, name := range tt.wantMissing {
				if !slices.Contains(missing.Names, name) {
					t.Errorf("Load() did not name %q as missing; named %v", name, missing.Names)
				}
				if !strings.Contains(err.Error(), name) {
					t.Errorf("Load() error message %q does not mention %q", err.Error(), name)
				}
			}
		})
	}
}

// Load's whole result is compared, not the fields each case cares about, so a
// new setting cannot be added without a decision about what it defaults to.
func TestLoadResolvesTheConfiguration(t *testing.T) {
	base := config.Config{
		ServerAddress: ":8080",
		Database: config.Database{
			Host: "db", Port: "5432", User: "splitter", Password: "s3cret", Name: "splitter", SSLMode: "disable",
		},
		MigrationsPath:  "migrations",
		LogLevel:        "info",
		ShutdownTimeout: 10 * time.Second,
	}

	overridden := base
	overridden.ServerAddress = ":9090"
	overridden.Database.Port = "6543"
	overridden.Database.SSLMode = "require"
	overridden.MigrationsPath = "/srv/migrations"
	overridden.LogLevel = "debug"
	overridden.ShutdownTimeout = 30 * time.Second

	tests := []struct {
		name string
		env  map[string]string
		want config.Config
	}{
		{
			name: "the named defaults apply when only the required variables are set",
			env:  nil,
			want: base,
		},
		{
			name: "every default can be overridden",
			env: map[string]string{
				"SERVER_ADDRESS":   ":9090",
				"DB_PORT":          "6543",
				"DB_SSLMODE":       "require",
				"MIGRATIONS_PATH":  "/srv/migrations",
				"LOG_LEVEL":        "debug",
				"SHUTDOWN_TIMEOUT": "30s",
			},
			want: overridden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			setEnv(t, required)
			setEnv(t, tt.env)

			got, err := config.Load()
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// An unreadable duration is gathered with the missing names rather than masking
// them, so one run of Load reports everything the environment got wrong.
func TestLoadReportsAnUnparseableDurationAlongsideMissingVariables(t *testing.T) {
	clearEnv(t)
	setEnv(t, required)
	t.Setenv("SHUTDOWN_TIMEOUT", "ten seconds")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() succeeded with an unparseable SHUTDOWN_TIMEOUT, want an error")
	}
	if !strings.Contains(err.Error(), "SHUTDOWN_TIMEOUT") {
		t.Errorf("Load() error %q does not name SHUTDOWN_TIMEOUT", err)
	}

	t.Setenv("DB_HOST", "")
	_, err = config.Load()
	if err == nil {
		t.Fatal("Load() succeeded with DB_HOST missing, want an error")
	}
	var missing *config.MissingError
	if !errors.As(err, &missing) {
		t.Fatalf("Load() error = %v, want a *config.MissingError among the problems", err)
	}
	if !strings.Contains(err.Error(), "SHUTDOWN_TIMEOUT") {
		t.Errorf("Load() error %q reports the missing variable but drops SHUTDOWN_TIMEOUT", err)
	}
}

// DSN is the one derived value, and it has to survive a password containing URL
// metacharacters: a naive fmt.Sprintf would silently truncate the credentials.
func TestDatabaseDSNEscapesCredentials(t *testing.T) {
	clearEnv(t)
	setEnv(t, required)
	t.Setenv("DB_PASSWORD", "p@ss:w/rd?")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	want := "postgres://splitter:p%40ss%3Aw%2Frd%3F@db:5432/splitter?sslmode=disable"
	if got := cfg.Database.DSN(); got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}
