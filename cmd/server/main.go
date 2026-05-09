package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer cancel()

	srv := tcp.NewServer(logger)
	errC := make(chan error, 1)
	go func() {
		<-ctx.Done()
		cancel()
		shutDownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		errC <- srv.Shutdown(shutDownCtx)
	}()

	err := srv.ListenAndServe(context.Background(), ":8080")
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return <-errC
}
