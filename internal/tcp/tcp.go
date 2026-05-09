package tcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
)

const (
	shutdownTimeout = 1 * time.Minute
)

var (
	ErrShutdownTimeout = errors.New("timeout while shutdown server")
)

type Server struct {
	logger *slog.Logger

	listener net.Listener

	closeOnce      sync.Once
	listenerClosed chan struct{}

	wg    sync.WaitGroup
	mu    sync.RWMutex
	peers map[xid.ID]*Peer
}

func NewServer(logger *slog.Logger) *Server {
	return &Server{
		logger: logger,
		peers:  make(map[xid.ID]*Peer),
	}
}

// ListenAndServe starts a listener. It determines on the addr if it uses
// a TCP socket or a Unix socket, then accepts connections and dispatches
// each one to handler in its own goroutine.
//
// ListenAndServe always returns a non-nil error.
func (srv *Server) ListenAndServe(ctx context.Context, addr string) error {

	network := networkFromAddr(addr)

	listener, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	srv.listener = listener
	srv.listenerClosed = make(chan struct{})

	stop := context.AfterFunc(ctx, srv.closeListener)
	defer stop()

	return srv.serve(ctx, listener)
}

// serve accepts connections from l and dispatches each one to handler in
// its own goroutine. The listener is closed before serve returns.
func (srv *Server) serve(ctx context.Context, l net.Listener) error {
	defer close(srv.listenerClosed)

	for {
		conn, err := l.Accept()
		if err != nil {
			switch {
			case errors.Is(err, net.ErrClosed):
				return nil
			default:
				srv.logger.ErrorContext(ctx, "accept connection", "error", err)
				continue
				// return fmt.Errorf("accept connection: %w", err)
			}
		}

		peerCtx, cancel := context.WithCancel(ctx)
		peer := newPeer(conn, cancel, srv.logger)

		srv.mu.Lock()
		srv.peers[peer.id] = peer
		srv.mu.Unlock()

		srv.wg.Go(func() {
			defer func() {
				cancel()
				srv.mu.Lock()
				delete(srv.peers, peer.id)
				srv.mu.Unlock()
			}()
			if serveErr := peer.Serve(peerCtx); serveErr != nil {
				if !errors.Is(serveErr, ErrPeerClosed) {
					srv.logger.ErrorContext(peerCtx, "peer serve error", "error", serveErr)
				}
			}
		})
	}
}

func (srv *Server) Shutdown(ctx context.Context) error {
	srv.logger.Info("shutdown server")

	// stop accepting new connections
	srv.closeListener()

	select {
	case <-srv.listenerClosed:
	case <-ctx.Done():
		return fmt.Errorf("shutdown server: %w", ctx.Err())
	}

	done := make(chan struct{})
	go func() {
		srv.wg.Wait() // wait for peers to close
		srv.logger.Info("all peer are done")
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown server: %w", ctx.Err())
	case <-done:
		return nil
	}
}

func (srv *Server) closeListener() {
	srv.closeOnce.Do(func() {
		if srv.listener != nil {
			_ = srv.listener.Close()
		}
	})
}

// networkFromAddr returns "unix" when addr looks like a file-system path
// (starts with "/" or "./"), and "tcp" otherwise.
func networkFromAddr(addr string) string {
	if strings.HasPrefix(addr, "/") || strings.HasPrefix(addr, "./") {
		return "unix"
	}
	return "tcp"
}
