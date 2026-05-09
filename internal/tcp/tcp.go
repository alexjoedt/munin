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

type Server struct {
	listener net.Listener
	logger   *slog.Logger

	cancelFunc context.CancelFunc

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
	srvCtx, cancel := context.WithCancel(ctx)
	srv.cancelFunc = cancel

	network := networkFromAddr(addr)

	listener, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}
	srv.listener = listener

	return srv.serve(srvCtx, listener)
}

// serve accepts connections from l and dispatches each one to handler in
// its own goroutine. The listener is closed before serve returns.
func (srv *Server) serve(ctx context.Context, l net.Listener) error {
	// Stop accepting new connections when context is done
	go func() {
		<-ctx.Done()
		srv.logger.Info("server context canceled")
		_ = l.Close()
	}()

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
	// Do not close the underlying listener here
	if srv.cancelFunc != nil {
		srv.cancelFunc() // Stops the listener and prevents accepting new connections
	}

	done := make(chan struct{})
	go func() {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		for _, p := range srv.peers {
			srv.logger.Info("closing peer", "id", p.id)
			p.cancelFunc() // closing all peer
		}
	}()

	go func() {
		srv.wg.Wait()
		srv.logger.Info("all peer are done")
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown server: %w", ctx.Err())
	case <-done:
		return nil
	case <-time.After(shutdownTimeout):
		return fmt.Errorf("timeout")
	}
}

// networkFromAddr returns "unix" when addr looks like a file-system path
// (starts with "/" or "./"), and "tcp" otherwise.
func networkFromAddr(addr string) string {
	if strings.HasPrefix(addr, "/") || strings.HasPrefix(addr, "./") {
		return "unix"
	}
	return "tcp"
}
