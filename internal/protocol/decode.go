package protocol

import (
	"fmt"
	"io"
)

// Unmarshal decodes data into v from its wire format.
// v must implement the Unmarshaller interface.
func Unmarshal(data []byte, v any) error {
	if data == nil {
		return fmt.Errorf("data must not be nil")
	}

	u, ok := v.(Unmarshaller)
	if !ok {
		return fmt.Errorf("type %T does not implement Unmarshaller interface", v)
	}
	return u.UnmarshalWire(data)
}

func ReadHeader(r io.Reader) (*Header, error) {
	lr := io.LimitReader(r, headerLength)
	buf := make([]byte, headerLength)

	n, err := lr.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	if n != headerLength {
		return nil, fmt.Errorf("read header, got %d bytes; expected %d", n, headerLength)
	}

	var header Header
	if err := Unmarshal(buf, &header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	return &header, nil
}
