package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type Header struct {
	MagicByte uint32 // 4 Byte
	Version   uint8  // 1 Byte
	Type      uint8  // 1 Byte
	Length    uint32 // 4 Byte
}

// MarshalWire marshals [Header] in the wire format.
func (h *Header) MarshalWire() ([]byte, error) {
	if h == nil {
		return nil, errors.New("header is nil")
	}

	buf := make([]byte, headerLength)
	binary.LittleEndian.PutUint32(buf, h.MagicByte)
	buf[4] = h.Version
	buf[5] = h.Type
	binary.BigEndian.PutUint32(buf[6:10], h.Length)

	return buf, nil
}

// UnmarshalWire unmarshals [Header] from the wire format.
func (h *Header) UnmarshalWire(b []byte) error {
	if h == nil {
		return errors.New("header is nil")
	}

	if len(b) < headerLength {
		return fmt.Errorf("buffer too short for header: got=%d want>=%d", len(b), headerLength)
	}

	offset := 0
	h.MagicByte = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	h.Version = b[offset]
	offset++

	h.Type = b[offset]
	offset++

	h.Length = binary.BigEndian.Uint32(b[offset : offset+4])

	return nil
}

// Message is the message that represents the wire format, it acts as envelope to transport the message type payload.
//
//	| MagicByte (4B) | Version (1B) | Type (1B) | Length (4B) | Payload (N) |
type Message struct {
	*Header

	Payload []byte
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
	payloadLen := len(payload)
	length := uint32(math.MaxUint32)
	if payloadLen <= math.MaxUint32 {
		length = uint32(payloadLen)
	}

	return &Message{
		Header: &Header{
			MagicByte: magicByte,
			Version:   version,
			Type:      uint8(t),
			Length:    length,
		},
		Payload: payload,
	}
}

// MarshalWire marshals [Message] in the wire format.
func (msg *Message) MarshalWire() ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}
	if msg.Header == nil {
		return nil, errors.New("message header is nil")
	}

	payloadLen := len(msg.Payload)
	if payloadLen > math.MaxUint32 {
		return nil, fmt.Errorf("payload too large: %d bytes", payloadLen)
	}

	if msg.Length != uint32(payloadLen) {
		return nil, fmt.Errorf("message length mismatch: header=%d payload=%d", msg.Length, payloadLen)
	}

	headerData, err := msg.Header.MarshalWire()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, headerLength+payloadLen)
	copy(buf, headerData)
	copy(buf[10:], msg.Payload)

	return buf, nil
}

func (msg *Message) UnmarshalWire(b []byte) error {
	if msg == nil {
		return errors.New("message is nil")
	}

	if len(b) < headerLength {
		return fmt.Errorf("buffer too short for header: got=%d want>=%d", len(b), headerLength)
	}

	if msg.Header == nil {
		msg.Header = &Header{}
	}

	if err := msg.Header.UnmarshalWire(b[:headerLength]); err != nil {
		return err
	}

	payloadEnd := headerLength + int(msg.Length)
	if payloadEnd > len(b) {
		return fmt.Errorf("buffer too short for payload: got=%d need=%d", len(b), payloadEnd)
	}

	msg.Payload = append(msg.Payload[:0], b[headerLength:payloadEnd]...)

	return nil
}
