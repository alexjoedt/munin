package protocol

import (
	"errors"
	"fmt"
)

const (
	headerLength = 10
	magicByte    = 0x0539
	version      = 1
)

// Marshaller encodes a value into its wire format.
type Marshaller interface {
	MarshalWire() ([]byte, error)
}

// Unmarshaller decodes a value from its wire format.
type Unmarshaller interface {
	UnmarshalWire([]byte) error
}

// writeBytes copies b into buf starting at offset and returns bytes written.
func writeBytes(buf []byte, offset int, b []byte) (int, error) {
	if buf == nil {
		return 0, errors.New("buf must not be nil")
	}

	if offset < 0 {
		return 0, fmt.Errorf("offset must be greater than or equal to 0: %d", offset)
	}

	if b == nil {
		return 0, errors.New("b must not be nil")
	}

	if offset > len(buf) {
		return 0, fmt.Errorf("offset out of bounds: %d", offset)
	}

	if offset == len(buf) {
		if len(buf) == 0 || len(b) > 0 {
			return 0, fmt.Errorf("offset out of bounds: %d", offset)
		}

		return 0, nil
	}

	if len(b) > len(buf)-offset {
		return 0, errors.New("bytes to add exceed remaining buffer capacity")
	}

	return copy(buf[offset:], b), nil
}
