package munin

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/alexjoedt/munin/internal/transport"
)

var (
	ErrConnectionClosed = errors.New("munin: connection closed")
)

type Conn struct {
	addr *transport.Addr
	conn net.Conn
}

func Dial(ctx context.Context, addr string) (*Conn, error) {
	address, err := transport.ParseAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("address: %w", err)
	}

	var dialer net.Dialer
	c, err := dialer.DialContext(ctx, address.Network, address.Address())
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return &Conn{
		addr: address,
		conn: c,
	}, nil
}
