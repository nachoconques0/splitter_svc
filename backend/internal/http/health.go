package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate go tool mockgen -source=health.go -destination=mocks/health.go -package=mocks

// Pinger reports whether the database answers.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthResponse reports the database too: a process cut off from Postgres can
// still serve this endpoint, so "ok" alone would be a lie.
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func health(logger *slog.Logger, pinger Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pinger.PingContext(c.Request.Context()); err != nil {
			logger.ErrorContext(c.Request.Context(), "health check could not reach the database", slog.Any("error", err))
			AbortWithError(c, http.StatusServiceUnavailable, ErrCodeDatabaseUnavailable, "the database is not reachable")
			return
		}

		c.JSON(http.StatusOK, HealthResponse{Status: "ok", Database: "ok"})
	}
}
