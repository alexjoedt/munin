package protocol

import "encoding/binary"

const (
	headerLength = 10
	magicByte    = 0x0539
	version      = 1
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

// NewMessage is a helper for a munin message
func NewMessage(t Type, payload []byte) *Message {
	return &Message{
		MagicByte: magicByte,
		Version:   version,
		Type:      uint8(t),
		Length:    uint32(len(payload)),
		Payload:   payload,
	}
}

func (msg *Message) MarshalBinary() ([]byte, error) {
	buf := make([]byte, headerLength+msg.Length)

	binary.LittleEndian.PutUint32(buf, msg.MagicByte)
	buf[4] = msg.Version
	buf[5] = msg.Type
	binary.BigEndian.PutUint32(buf[6:10], msg.Length)
	copy(buf[10:], msg.Payload)

	return buf, nil
}
