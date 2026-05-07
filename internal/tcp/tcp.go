package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/rs/xid"
)

type Server struct {
	listener net.Listener

	wg    sync.WaitGroup
	mu    sync.RWMutex
	peers map[xid.ID]*Peer
}

func NewServer() *Server {
	return &Server{
		peers: make(map[xid.ID]*Peer),
	}
}

// ListenAndServe starts a listener. It determines on the addr if it uses
// a TCP socket or a Unix socket, then accepts connections and dispatches
// each one to handler in its own goroutine.
//
// ListenAndServe always returns a non-nil error.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	network := networkFromAddr(addr)

	listener, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}
	s.listener = listener

	return s.serve(ctx, listener)
}

// serve accepts connections from l and dispatches each one to handler in
// its own goroutine. The listener is closed before serve returns.
func (s *Server) serve(ctx context.Context, l net.Listener) error {

	// Stop accepting new connections when context is done
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				return nil
			default:
				return fmt.Errorf("accept connection: %w", err)
			}
		}

		fmt.Println("accpeted new connection")

		peerCtx, cancel := context.WithCancel(ctx)
		peer := &Peer{id: xid.New(), conn: conn, cancelFunc: cancel}

		s.mu.Lock()
		s.peers[peer.id] = peer
		s.mu.Unlock()

		s.wg.Go(func() {
			defer cancel()
			fmt.Println("Starting peer handler", peer.id)
			if err := peer.Serve(peerCtx); err != nil {
				if !errors.Is(err, ErrPeerClosed) {
					fmt.Printf("peer serve: %v\n", err)
				}
			} else {
				fmt.Printf("peer '%s' closed gracefully\n", peer.id)
			}
		})
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
