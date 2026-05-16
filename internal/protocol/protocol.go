package protocol

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
