package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/rs/xid"

	"github.com/alexjoedt/munin/internal/protocol"
)

// readTimeout is the maximum time to wait for a new line from a peer.
// It acts as an idle-connection detector: if a peer sends nothing within
// this window, the connection is considered stale and the read is retried.
// Any ongoing context cancellation will unblock the read before this fires.
const readTimeout = 30 * time.Second

type Peer struct {
	id         xid.ID
	conn       net.Conn
	logger     *slog.Logger
	cancelFunc context.CancelFunc
}

func newPeer(conn net.Conn, cancelFunc context.CancelFunc, logger *slog.Logger) *Peer {
	id := xid.New()
	return &Peer{
		id:         id,
		conn:       conn,
		cancelFunc: cancelFunc,
		logger:     logger.With("peer_id", id, "remote_addr", conn.RemoteAddr()),
	}
}

func (peer *Peer) Serve(ctx context.Context) error {
	pCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer peer.conn.Close()

	peer.logger.InfoContext(pCtx, "peer connected")

	// When the context is cancelled, unblock the blocking ReadString by
	// expiring the deadline immediately. The resulting timeout error is
	// detected below and causes the loop to exit cleanly.
	go func() {
		<-pCtx.Done()
		_ = peer.conn.SetDeadline(time.Now())
	}()

	r := bufio.NewReader(peer.conn)
	for {
		_ = peer.conn.SetDeadline(time.Now().Add(readTimeout))
		line, err := r.ReadString('\n')
		if err != nil {
			shouldRetry, peerErr := handleErr(err, peer.id.String())
			if shouldRetry && pCtx.Err() == nil {
				continue
			}
			if peerErr != nil {
				peer.logger.WarnContext(pCtx, "peer disconnected with error", "error", peerErr)
			} else {
				peer.logger.InfoContext(pCtx, "peer disconnected")
			}
			return peerErr
		}

		line = strings.TrimSuffix(line, "\n")
		if line == "q" {
			peer.logger.InfoContext(pCtx, "peer requested close")
			return nil
		}
	}
}

// Close cancels the peer's context, triggering a graceful shutdown.
func (peer *Peer) Close() {
	if peer.cancelFunc != nil {
		peer.cancelFunc()
	}
}

var (
	ErrPeerClosed = errors.New("peer closed the connection")
	ErrPeerReset  = errors.New("peer reset the connection")
)

// handleErr classifies a read error into one of three outcomes:
//   - true, nil  — transient error (e.g. idle timeout); caller should retry the read.
//   - false, nil — connection was closed cleanly (locally or by context); no action needed.
//   - false, err — connection was lost or an unexpected error occurred; caller should return the error.
func handleErr(err error, peerID string) (bool, error) {
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		// Remote peer closed the connection cleanly or dropped it mid-message.
		return false, &protocol.PeerErr{Err: fmt.Errorf("%w: %w", ErrPeerClosed, err), PeerID: peerID}
	case errors.Is(err, net.ErrClosed):
		// Connection was closed locally (e.g. context cancellation triggered SetDeadline).
		return false, nil
	}

	netErr, ok := errors.AsType[*net.OpError](err)
	if !ok {
		return false, err
	}

	switch {
	case errors.Is(netErr.Err, syscall.ECONNRESET):
		// Remote peer reset the connection abruptly.
		return false, &protocol.PeerErr{Err: fmt.Errorf("%w: %w", ErrPeerReset, err), PeerID: peerID}
	case netErr.Timeout():
		// Read deadline expired — idle timeout (retry) or context cancellation (checked by caller).
		return true, nil
	default:
		return false, &protocol.PeerErr{Err: err, PeerID: peerID}
	}
}
