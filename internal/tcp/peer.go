package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/rs/xid"

	"github.com/alexjoedt/munin/internal/protocol"
)

type Peer struct {
	id   xid.ID
	conn net.Conn

	cancelFunc context.CancelFunc
}

func (peer *Peer) Serve(ctx context.Context) error {
	pCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer peer.conn.Close()

	go func() {
		<-pCtx.Done()
		_ = peer.conn.SetDeadline(time.Now())
	}()

	const readTimeout = 30 * time.Second
	r := bufio.NewReader(peer.conn)
	for {
		_ = peer.conn.SetDeadline(time.Now().Add(readTimeout))
		line, err := r.ReadString('\n')
		if err != nil {
			ok, peerErr := handleErr(err, peer.id.String())
			if ok && pCtx.Err() == nil {
				continue
			}
			return peerErr
		}

		line = strings.TrimSuffix(line, "\n")
		if line == "q" {
			return nil
		}
	}
}

// Close cancels the peer's context.
func (peer *Peer) Close() {
	peer.cancelFunc()
}

var (
	ErrPeerClosed = errors.New("peer closed the connection")
	ErrPeerReset  = errors.New("peer reset the connection")
)

func handleErr(err error, peerID string) (bool, error) {
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		// Remote peer closed the connection cleanly or mid-message.
		return false, &protocol.PeerErr{Err: fmt.Errorf("%w: %w", ErrPeerClosed, err), PeerID: peerID}
	case errors.Is(err, net.ErrClosed):
		// Connection was closed locally (e.g., context cancellation triggered conn.Close).
		return false, nil
	}

	netErr, ok := errors.AsType[*net.OpError](err)
	if ok {
		// Connection reset by peer: treat as an abrupt peer close.
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return false, &protocol.PeerErr{Err: fmt.Errorf("%w: %w", ErrPeerReset, err), PeerID: peerID}
		}
		// Timeout errors are transient; signal the caller to retry.
		if netErr.Timeout() {
			return true, nil
		}
		return false, &protocol.PeerErr{Err: err, PeerID: peerID}
	}

	return false, err
}
