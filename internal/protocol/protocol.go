package protocol

import (
	"encoding/binary"
)

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

type Message struct {
	MagicByte uint32 // 4 Byte
	Version   uint8  // 1 Byte
	Type      uint8  // 1 Byte
	Length    uint32 // 4 Byte
	Payload   []byte
}

type Type uint8

const (
	TypeUnknown Type = iota
	TypeHandshake
	TypeHeartbeat
	TypePublish
	TypeFetch
	TypeAck
)

// NewMessage is a helper for a munin message.
func NewMessage(t Type, payload []byte) *Message {
	return &Message{
		MagicByte: magicByte,
		Version:   version,
		Type:      uint8(t),
		Length:    uint32(len(payload)),
		Payload:   payload,
	}
}

// MarshalWire marshals [Message] in the wire format.
func (msg *Message) MarshalWire() ([]byte, error) {
	buf := make([]byte, headerLength+msg.Length)

	binary.LittleEndian.PutUint32(buf, msg.MagicByte)
	buf[4] = msg.Version
	buf[5] = msg.Type
	binary.BigEndian.PutUint32(buf[6:10], msg.Length)
	copy(buf[10:], msg.Payload)

	return buf, nil
}

func (msg *Message) UnmarshalWire(b []byte) error {
	offset := 0
	msg.MagicByte = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	msg.Version = b[offset]
	offset++

	msg.Type = b[offset]
	offset++
	msg.Length = binary.BigEndian.Uint32(b[offset : offset+4])
	offset += 4

	msg.Payload = b[offset : offset+int(msg.Length)]

	return nil
}
