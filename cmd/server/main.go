package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/alexjoedt/munin/internal/tcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "exited with an error: %w\n", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	logger.Info("Starting Munin server...")

	srv := tcp.NewServer()
	err := srv.ListenAndServe(context.Background(), ":8080")
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
