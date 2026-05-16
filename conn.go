package munin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/alexjoedt/munin/internal/protocol"
	"github.com/alexjoedt/munin/internal/transport"
)

var (
	ErrConnectionClosed = errors.New("munin: connection closed")
)

type Conn struct {
	addr *transport.Addr

	mu          sync.RWMutex
	conn        net.Conn
	isConnected atomic.Bool
	connC       chan struct{}
	done        chan struct{}

	cancelFn context.CancelFunc
}

func Dial(ctx context.Context, addr string) (*Conn, error) {
	address, err := transport.ParseAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("address: %w", err)
	}

	dCtx, cancel := context.WithCancel(ctx)
	var dialer net.Dialer
	c, err := dialer.DialContext(dCtx, address.Network, address.Address())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("dial: %w", err)
	}

	conn := &Conn{
		addr:        address,
		conn:        c,
		connC:       make(chan struct{}, 1),
		isConnected: atomic.Bool{},
		cancelFn:    cancel,
	}

	var stateFn StateFunc = conn.handshake

	go func() {
		defer close(conn.connC)
		for dCtx.Err() == nil {
			stateFn, err = stateFn(dCtx)
			if err != nil {
				return
			}

			if stateFn == nil {
				return
			}
		}
	}()

	<-conn.connC
	return conn, nil
}

func (conn *Conn) writeMsg(msg protocol.Marshaller) (int, error) {
	data, err := msg.MarshalWire()
	if err != nil {
		return 0, fmt.Errorf("marshal: %w", err)
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	return conn.conn.Write(data)
}

type StateFunc func(ctx context.Context) (StateFunc, error)

func (c *Conn) handshake(ctx context.Context) (StateFunc, error) {
	msg := protocol.NewMessage(protocol.TypeHandshake, nil)

	_, err := c.writeMsg(msg)
	if err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	buf := make([]byte, 10)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("handshake: read from conn: %w", err)
	}

	if n != 10 {
		return nil, fmt.Errorf("expected to read 10 bytes, got %d", n)
	}

	var header protocol.Header
	if err := protocol.Unmarshal(buf, &header); err != nil {
		return nil, fmt.Errorf("unmarshal header: %w", err)
	}

	return nil, nil
}

func (c *Conn) connected(ctx context.Context) (StateFunc, error) {
	return nil, nil
}
