package protocol

const (
	headerLength = 10
	magicByte    = 0x0539
	version      = 1
)

type Marshaller interface {
	MarshalWire() ([]byte, error)
}

type Unmarshaller interface {
	UnmarshalWire([]byte) error
}
