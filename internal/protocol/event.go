package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/rs/xid"
)

var (
	ErrTopicExceedsMaxLength = errors.New("topic exceeds max length")
)

// Event is the payload enveloped in a [TypePublish] Message.
// Wire format:
//
//	| ID (12B) | TopicLen (2B) | Topic (N) | DataLen (4B) | Data (N) |
type Event struct {
	ID    xid.ID          `json:"id"`    // 12 Bytes
	Topic string          `json:"topic"` // N Bytes
	Data  json.RawMessage `json:"data"`  // N Bytes
}

func NewEvent(topic string, data []byte) *Event {
	return &Event{
		ID:    xid.New(),
		Topic: topic,
		Data:  data,
	}
}

// MarshalWire marshals [Event] into a binary format.
func (e *Event) MarshalWire() ([]byte, error) {
	topic := []byte(e.Topic)
	if len(topic) > math.MaxUint16 {
		return nil, fmt.Errorf("event: %w", ErrTopicExceedsMaxLength)
	}

	buf := make([]byte, 12+2+len(e.Topic)+4+len(e.Data))
	offset := 0

	// Write ID
	n, err := writeBytes(buf, offset, e.ID[:])
	if err != nil {
		return nil, fmt.Errorf("write id: %w", err)
	}
	offset += n

	beWriter := binary.BigEndian

	// Write Topic length
	beWriter.PutUint16(buf[offset:], uint16(len(e.Topic)))
	offset += 2

	// Write Topic
	n, err = writeBytes(buf, offset, []byte(e.Topic))
	if err != nil {
		return nil, fmt.Errorf("write topic length: %w", err)
	}
	offset += n

	// Write data length
	beWriter.PutUint32(buf[offset:], uint32(len(e.Data)))
	offset += 4

	// Write data
	_, err = writeBytes(buf, offset, e.Data)
	if err != nil {
		return nil, fmt.Errorf("write data: %w", err)
	}

	return buf, nil
}

func (e *Event) UnmarshalWire(b []byte) error {
	offset := 0

	// Read ID
	id, err := xid.FromBytes(b[:12])
	if err != nil {
		return fmt.Errorf("parse id: %w", err)
	}
	e.ID = id
	offset += 12

	// Read Topic length
	topicLen := int(binary.BigEndian.Uint16(b[offset : offset+2]))
	offset += 2

	// Read Topic
	e.Topic = string(b[offset : offset+topicLen])
	offset += topicLen

	// Read Data length
	dataLen := int(binary.BigEndian.Uint32(b[offset : offset+4]))
	offset += 4

	e.Data = b[offset : offset+dataLen]

	return nil
}

func writeBytes(buf []byte, offset int, b []byte) (int, error) {
	if buf == nil {
		return 0, fmt.Errorf("buf must not be nil")
	}

	if offset < 0 {
		return 0, fmt.Errorf("offset must be greater than 0: %d", offset)
	}

	if b == nil {
		return 0, fmt.Errorf("b must not be nil")
	}

	if offset >= len(buf) {
		return 0, fmt.Errorf("offset out of bounds: %d", offset)
	}

	if len(b) > len(buf) {
		return 0, fmt.Errorf("bytes to add are greater than buf")
	}

	return copy(buf[offset:], b), nil
}
