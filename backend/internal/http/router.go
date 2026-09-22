// Package http wires the routes and owns the error envelope every response shares.
package http

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

//go:generate go tool mockgen -source=router.go -destination=mocks/router.go -package=mocks

// Declared here so the dependency points one way: the controllers reach into
// this package for the error envelope, so it must not reach back.
type BillController interface {
	ShareSet(*gin.Context)
	ReplaceShareSet(*gin.Context)
}

// PersonController serves the Person routes. No update and no delete on purpose:
// a Share refers to a Person.
type PersonController interface {
	Create(*gin.Context)
	List(*gin.Context)
}

// NewRouter builds the engine. Nothing configures CORS: nginx serves the page
// and proxies /api, so every request the browser makes is same-origin.
func NewRouter(logger *slog.Logger, pinger Pinger, bills BillController, people PersonController) *gin.Engine {
	router := gin.New()

	// gin's Recovery aborts with a bare 500 and an empty body, which would make a
	// panic the one response not wearing the envelope. Its stack dump is
	// discarded because the stack is attached to the log record instead.
	router.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		withStack := logger.With(slog.String("stack", string(debug.Stack())))
		AbortWithInternalError(c, withStack, fmt.Errorf("panic: %v", recovered))
	}))

	// gin's 404 and 405 are bare text; the envelope keeps one error shape.
	router.NoRoute(func(c *gin.Context) {
		AbortWithError(c, http.StatusNotFound, ErrCodeNotFound, "no such endpoint")
	})
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		AbortWithError(c, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "that method is not allowed on this endpoint")
	})

	router.GET("/health", health(logger, pinger))
	router.GET("/bills/:id/shares", bills.ShareSet)
	router.PUT("/bills/:id/shares", bills.ReplaceShareSet)
	router.POST("/people", people.Create)
	router.GET("/people", people.List)

	return router
}
