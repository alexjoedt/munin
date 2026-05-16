package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexjoedt/munin/internal/transport"

	"github.com/alexjoedt/log"
)

const (
	shutdownTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "exited with an error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	handler := log.NewSlogHandler(
		log.WithFormat(log.FormatConsole),
		log.WithWriter(os.Stderr),
	)

	logger := slog.New(handler)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv := transport.NewServer(logger)
	errC := make(chan error, 1)
	go func() {
		<-ctx.Done()
		cancel()
		shutDownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancelShutdown()
		errC <- srv.Shutdown(shutDownCtx)
	}()

	err := srv.ListenAndServe(ctx, ":8080")
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return <-errC
}
