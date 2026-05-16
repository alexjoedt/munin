package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/xid"
)

func TestEventUnmarshalBinary(t *testing.T) {
	fixedID := xid.ID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	anotherID := xid.ID{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B}

	tests := []struct {
		name        string
		binaryEvent []byte
		want        *Event
		wantErr     bool
	}{
		{
			name:        "event with topic and data",
			binaryEvent: eventBinary(fixedID, "users.created", []byte(`{"email": "test@munin.com"}`)),
			want:        &Event{ID: fixedID, Topic: "users.created", Data: []byte(`{"email": "test@munin.com"}`)},
		},
		{
			name:        "event with empty topic and empty data",
			binaryEvent: eventBinary(anotherID, "", []byte{}),
			want:        &Event{ID: anotherID, Topic: "", Data: []byte{}},
		},
		{
			name:        "event with topic and empty data",
			binaryEvent: eventBinary(anotherID, "users.updated", []byte{}),
			want:        &Event{ID: anotherID, Topic: "users.updated", Data: []byte{}},
		},
		{
			name:        "event with empty topic and data",
			binaryEvent: eventBinary(anotherID, "", []byte(`{"ok":true}`)),
			want:        &Event{ID: anotherID, Topic: "", Data: []byte(`{"ok":true}`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{}
			err := event.UnmarshalBinary(tt.binaryEvent)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("UnmarshalBinary(): error = nil; want non-nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalBinary(): error = %v; want nil", err)
			}

			if !reflect.DeepEqual(tt.want, event) {
				t.Fatalf("want %v, got %v", tt.want, event)
			}
		})
	}
}

func TestEventMarshalBinary(t *testing.T) {
	fixedID := xid.ID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}

	tests := []struct {
		name    string
		event   *Event
		want    []byte
		wantErr bool
		errIs   error
	}{
		{
			name: "marshal event with topic and data",
			event: &Event{
				ID:    fixedID,
				Topic: "orders.created",
				Data:  []byte(`{"order_id":42}`),
			},
			want: eventBinary(fixedID, "orders.created", []byte(`{"order_id":42}`)),
		},
		{
			name: "marshal event with empty topic and empty data",
			event: &Event{
				ID:    fixedID,
				Topic: "",
				Data:  []byte{},
			},
			want: eventBinary(fixedID, "", []byte{}),
		},
		{
			name: "topic exceeding max uint16 returns error",
			event: &Event{
				ID:    fixedID,
				Topic: strings.Repeat("a", int(math.MaxUint16)+1),
				Data:  []byte(`{"ok":true}`),
			},
			wantErr: true,
			errIs:   ErrTopicExceedsMaxLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.event.MarshalBinary()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("MarshalBinary() error = nil, want non-nil")
				}

				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Fatalf("MarshalBinary() error = %v, want error matching %v", err, tt.errIs)
				}

				return
			}

			if err != nil {
				t.Fatalf("MarshalBinary() unexpected error = %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Fatalf("MarshalBinary() bytes mismatch\nwant: % X\ngot:  % X", tt.want, got)
			}
		})
	}
}

func TestEventMarshalBinary_WireSegments(t *testing.T) {
	fixedID := xid.ID{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B}

	tests := []struct {
		name  string
		event *Event
	}{
		{
			name: "non-empty topic and data",
			event: &Event{
				ID:    fixedID,
				Topic: "invoice.paid",
				Data:  []byte(`{"invoice_id":"inv-1","paid":true}`),
			},
		},
		{
			name: "empty topic and data",
			event: &Event{
				ID:    fixedID,
				Topic: "",
				Data:  []byte{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.event.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary() unexpected error = %v", err)
			}

			topicBytes := []byte(tt.event.Topic)
			dataBytes := []byte(tt.event.Data)
			wantLen := 12 + 2 + len(topicBytes) + 4 + len(dataBytes)
			if len(got) != wantLen {
				t.Fatalf("MarshalBinary() len = %d, want %d", len(got), wantLen)
			}

			offset := 0

			if !bytes.Equal(got[offset:offset+12], tt.event.ID[:]) {
				t.Fatalf("ID mismatch\nwant: % X\ngot:  % X", tt.event.ID[:], got[offset:offset+12])
			}
			offset += 12

			topicLen := int(binary.BigEndian.Uint16(got[offset : offset+2]))
			if topicLen != len(topicBytes) {
				t.Fatalf("TopicLen mismatch = %d, want %d", topicLen, len(topicBytes))
			}
			offset += 2

			if !bytes.Equal(got[offset:offset+topicLen], topicBytes) {
				t.Fatalf("Topic mismatch\nwant: % X\ngot:  % X", topicBytes, got[offset:offset+topicLen])
			}
			offset += topicLen

			dataLen := int(binary.BigEndian.Uint32(got[offset : offset+4]))
			if dataLen != len(dataBytes) {
				t.Fatalf("DataLen mismatch = %d, want %d", dataLen, len(dataBytes))
			}
			offset += 4

			if !bytes.Equal(got[offset:offset+dataLen], dataBytes) {
				t.Fatalf("Data mismatch\nwant: % X\ngot:  % X", dataBytes, got[offset:offset+dataLen])
			}
		})
	}
}

func eventBinary(id xid.ID, topic string, data []byte) []byte {
	topicBytes := []byte(topic)
	buf := make([]byte, 12+2+len(topicBytes)+4+len(data))
	offset := 0

	offset += copy(buf[offset:], id[:])
	binary.BigEndian.PutUint16(buf[offset:], uint16(len(topicBytes)))
	offset += 2
	offset += copy(buf[offset:], topicBytes)
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(data)))
	offset += 4
	copy(buf[offset:], data)

	return buf
}

func TestWriteBytes(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		offset  int
		b       []byte
		wantN   int
		wantErr bool
		wantBuf []byte // expected buf state after the call
	}{
		{
			name:    "nil buf returns error",
			buf:     nil,
			offset:  0,
			b:       []byte{0x01},
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "nil b returns error",
			buf:     make([]byte, 4),
			offset:  0,
			b:       nil,
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "negative offset returns error",
			buf:     make([]byte, 4),
			offset:  -1,
			b:       []byte{0x01},
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "offset out of bounds returns error",
			buf:     make([]byte, 4),
			offset:  5,
			b:       []byte{0x01},
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "b larger than buf returns error",
			buf:     make([]byte, 2),
			offset:  0,
			b:       []byte{0x01, 0x02, 0x03},
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "write at offset zero",
			buf:     make([]byte, 4),
			offset:  0,
			b:       []byte{0x01, 0x02},
			wantN:   2,
			wantErr: false,
			wantBuf: []byte{0x01, 0x02, 0x00, 0x00},
		},
		{
			name:    "write at non-zero offset",
			buf:     make([]byte, 4),
			offset:  2,
			b:       []byte{0x03, 0x04},
			wantN:   2,
			wantErr: false,
			wantBuf: []byte{0x00, 0x00, 0x03, 0x04},
		},
		{
			name:    "write fills entire buf",
			buf:     make([]byte, 3),
			offset:  0,
			b:       []byte{0xAA, 0xBB, 0xCC},
			wantN:   3,
			wantErr: false,
			wantBuf: []byte{0xAA, 0xBB, 0xCC},
		},
		{
			name:    "write single byte at last valid offset",
			buf:     make([]byte, 4),
			offset:  3,
			b:       []byte{0xFF},
			wantN:   1,
			wantErr: false,
			wantBuf: []byte{0x00, 0x00, 0x00, 0xFF},
		},
		{
			name:    "write overwrites existing content",
			buf:     []byte{0x01, 0x02, 0x03, 0x04},
			offset:  1,
			b:       []byte{0xAA, 0xBB},
			wantN:   2,
			wantErr: false,
			wantBuf: []byte{0x01, 0xAA, 0xBB, 0x04},
		},
		{
			name:    "offset equal to buf length returns error",
			buf:     make([]byte, 4),
			offset:  4,
			b:       []byte{0x01},
			wantN:   0,
			wantErr: true,
		},
		{
			// With the tightened guard (offset >= len(buf)), offset 0 on an
			// empty buf is out of bounds.
			name:    "zero offset on empty buf returns error",
			buf:     make([]byte, 0),
			offset:  0,
			b:       []byte{},
			wantN:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := writeBytes(tt.buf, tt.offset, tt.b)

			if (err != nil) != tt.wantErr {
				t.Errorf("writeBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if n != tt.wantN {
				t.Errorf("writeBytes() n = %d, want %d", n, tt.wantN)
			}

			if tt.wantBuf != nil {
				if len(tt.buf) != len(tt.wantBuf) {
					t.Fatalf("writeBytes() buf len = %d, want %d", len(tt.buf), len(tt.wantBuf))
				}
				for i := range tt.wantBuf {
					if tt.buf[i] != tt.wantBuf[i] {
						t.Errorf("writeBytes() buf[%d] = 0x%02X, want 0x%02X", i, tt.buf[i], tt.wantBuf[i])
					}
				}
			}
		})
	}
}
