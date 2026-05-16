package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/xid"
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
	address, err := ParseAddr(addr)
	if err != nil {
		return fmt.Errorf("address: %w", err)
	}

	switch address.Network {
	case "unix":
		addr = address.Path
	case "tcp":
		addr = fmt.Sprintf("%s:%d", address.Host, address.Port)
	}

	listener, err := net.Listen(address.Network, addr)
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	srv.listener = listener
	srv.listenerClosed = make(chan struct{})

	stop := context.AfterFunc(ctx, srv.closeListener)
	defer stop()

	srv.logger.Info("Starting server", "address", address.String())
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
					srv.logger.Error("handle peer", "error", serveErr)
				}
			}
		})

	}
}

func (srv *Server) Shutdown(ctx context.Context) error {
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

type Addr struct {
	Network string

	Host string
	Port uint16

	// Path is the path to the unix domain socket
	Path string
}

func (addr *Addr) String() string {
	switch addr.Network {
	case "unix":
		return fmt.Sprintf("%s:://%s", addr.Network, addr.Path)
	case "tcp":
		return fmt.Sprintf("%s:://%s:%d", addr.Network, addr.Host, addr.Port)
	}
	return ""
}

func ParseAddr(addr string) (*Addr, error) {
	var address Addr
	switch {
	case isUnix(addr):
		address.Network = "unix"
		var err error
		path := strings.TrimPrefix(addr, "unix://")
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("invalid path '%s': %w", path, err)
		}
		address.Path = path
		return &address, nil
	case isTCP(addr):
		address.Network = "tcp"
		host, port, err := net.SplitHostPort(strings.TrimPrefix(addr, "tcp://"))
		if err != nil {
			return nil, fmt.Errorf("parsing tcp address: %s: %w", addr, err)
		}
		address.Host = host
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid port in adress '%s': %w", addr, err)
		}
		if p > math.MaxUint16 {
			return nil, fmt.Errorf("port number exceeds limit: %d", port)
		}
		address.Port = uint16(p)
		return &address, nil
	default:
		return nil, fmt.Errorf("unsupported adress: '%s'", addr)
	}
}

func isUnix(addr string) bool {
	return strings.HasPrefix(addr, "/") ||
		strings.HasPrefix(addr, "./") ||
		strings.HasPrefix(addr, "unix")
}

func isTCP(addr string) bool {
	u, err := url.Parse(addr)
	if err == nil && u.Scheme == "tcp" {
		return true
	}
	_, _, err = net.SplitHostPort(addr)
	return err == nil
}
