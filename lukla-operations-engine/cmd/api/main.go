package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lukla/operations-engine/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := a.Run(ctx); err != nil {
		log.Fatalf("app exited with error: %v", err)
	}
}
