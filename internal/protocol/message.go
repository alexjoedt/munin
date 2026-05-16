package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

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
	if msg == nil {
		return nil, errors.New("message is nil")
	}

	payloadLen := len(msg.Payload)
	if payloadLen > math.MaxUint32 {
		return nil, fmt.Errorf("payload too large: %d bytes", payloadLen)
	}

	if msg.Length != uint32(payloadLen) {
		return nil, fmt.Errorf("message length mismatch: header=%d payload=%d", msg.Length, payloadLen)
	}

	buf := make([]byte, headerLength+payloadLen)

	binary.LittleEndian.PutUint32(buf, msg.MagicByte)
	buf[4] = msg.Version
	buf[5] = msg.Type
	binary.BigEndian.PutUint32(buf[6:10], uint32(payloadLen))
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
