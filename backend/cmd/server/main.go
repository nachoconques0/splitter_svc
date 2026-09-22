// Command server runs the bill splitting API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nachoconques0/splitter_svc/backend/internal/app"
)

func main() {
	// SIGTERM is how a container is asked to stop, and what makes the graceful
	// shutdown reachable.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		// Written to stderr rather than logged: a configuration failure happens
		// before there is a configured logger to fail through.
		fmt.Fprintln(os.Stderr, "server stopped:", err)
		os.Exit(1)
	}
}
