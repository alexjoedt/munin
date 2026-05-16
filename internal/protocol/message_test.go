package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestHeaderMarshalWire(t *testing.T) {
	tests := []struct {
		name string
		h    *Header
		want []byte
		ok   bool
	}{
		{
			name: "marshal header",
			h: &Header{
				MagicByte: magicByte,
				Version:   version,
				Type:      uint8(TypePublish),
				Length:    3,
			},
			want: []byte{0x39, 0x05, 0x00, 0x00, 0x01, 0x03, 0x00, 0x00, 0x00, 0x03},
			ok:   true,
		},
		{
			name: "nil header",
			h:    nil,
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.h.MarshalWire()
			if !tt.ok {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Fatalf("unexpected bytes: got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestHeaderUnmarshalWire(t *testing.T) {
	tests := []struct {
		name string
		h    *Header
		in   []byte
		ok   bool
	}{
		{
			name: "unmarshal header",
			h:    &Header{},
			in:   []byte{0x39, 0x05, 0x00, 0x00, 0x01, 0x03, 0x00, 0x00, 0x00, 0x03},
			ok:   true,
		},
		{
			name: "nil header",
			h:    nil,
			in:   []byte{0x39, 0x05, 0x00, 0x00, 0x01, 0x03, 0x00, 0x00, 0x00, 0x03},
			ok:   false,
		},
		{
			name: "short buffer",
			h:    &Header{},
			in:   []byte{0x39, 0x05, 0x00},
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.h.UnmarshalWire(tt.in)
			if !tt.ok {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if tt.h.MagicByte != magicByte {
				t.Fatalf("magic byte: expected %d, got %d", magicByte, tt.h.MagicByte)
			}

			if tt.h.Version != version {
				t.Fatalf("version: expected %d, got %d", version, tt.h.Version)
			}

			if tt.h.Type != uint8(TypePublish) {
				t.Fatalf("type: expected %d, got %d", TypePublish, tt.h.Type)
			}

			if tt.h.Length != 3 {
				t.Fatalf("length: expected %d, got %d", 3, tt.h.Length)
			}
		})
	}
}

func TestMessageMarshalWire(t *testing.T) {
	tests := []struct {
		name    string
		msgType Type
		payload []byte
	}{
		{"handshake", TypeHandshake, []byte(`{"message": "hello munin"}`)},
		{"heartbeat", TypeHeartbeat, []byte{}},
		{"publish", TypePublish, []byte(`{"topic":"t","data":"x"}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.msgType, tt.payload)

			data, err := msg.MarshalWire()
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			if len(data) != headerLength+len(tt.payload) {
				t.Fatalf("expected length %d, got %d", headerLength+len(tt.payload), len(data))
			}

			var expectedMagic [4]byte
			binary.LittleEndian.PutUint32(expectedMagic[:], magicByte)
			if !bytes.Equal(data[:4], expectedMagic[:]) {
				t.Errorf("magic bytes: expected %v, got %v", expectedMagic[:], data[:4])
			}

			if data[4] != version {
				t.Errorf("version: expected %d, got %d", version, data[4])
			}

			if data[5] != byte(tt.msgType) {
				t.Errorf("type: expected %d, got %d", tt.msgType, data[5])
			}

			var expectedLen [4]byte
			binary.BigEndian.PutUint32(expectedLen[:], uint32(len(tt.payload)))
			if !bytes.Equal(data[6:10], expectedLen[:]) {
				t.Errorf("length bytes: expected %v, got %v", expectedLen[:], data[6:10])
			}

			if !bytes.Equal(data[10:], tt.payload) {
				t.Errorf("payload: expected %s, got %s", tt.payload, data[10:])
			}
		})
	}
}

func TestMessageUnmarshalWire(t *testing.T) {
	tests := []struct {
		name    string
		msgType Type
		payload []byte
	}{
		{"handshake", TypeHandshake, []byte(`{"message":"hello munin"}`)},
		{"heartbeat", TypeHeartbeat, []byte{}},
		{"publish", TypePublish, []byte(`{"topic":"t","data":"x"}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire, marshalErr := NewMessage(tt.msgType, tt.payload).MarshalWire()
			if marshalErr != nil {
				t.Fatalf("marshal failed: %v", marshalErr)
			}

			var got Message
			unmarshalErr := got.UnmarshalWire(wire)
			if unmarshalErr != nil {
				t.Fatalf("unmarshal failed: %v", unmarshalErr)
			}

			if got.MagicByte != magicByte {
				t.Errorf("magic byte: expected %d, got %d", magicByte, got.MagicByte)
			}

			if got.Version != version {
				t.Errorf("version: expected %d, got %d", version, got.Version)
			}

			if got.Type != uint8(tt.msgType) {
				t.Errorf("type: expected %d, got %d", tt.msgType, got.Type)
			}

			if got.Length != uint32(len(tt.payload)) {
				t.Errorf("length: expected %d, got %d", len(tt.payload), got.Length)
			}

			if !bytes.Equal(got.Payload, tt.payload) {
				t.Errorf("payload: expected %s, got %s", tt.payload, got.Payload)
			}
		})
	}
}

func TestMessageMarshalWireErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "nil message",
			msg:  nil,
		},
		{
			name: "length mismatch",
			msg: &Message{
				Header: &Header{
					MagicByte: magicByte,
					Version:   version,
					Type:      uint8(TypePublish),
					Length:    10,
				},
				Payload: []byte("abc"),
			},
		},
		{
			name: "nil message header",
			msg: &Message{
				Payload: []byte("abc"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.msg.MarshalWire(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestMessageUnmarshalWireErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		in   []byte
	}{
		{
			name: "nil message",
			msg:  nil,
			in:   []byte{0x39, 0x05, 0x00, 0x00, 0x01, 0x03, 0x00, 0x00, 0x00, 0x03},
		},
		{
			name: "short header",
			msg:  &Message{},
			in:   []byte{0x39, 0x05},
		},
		{
			name: "short payload",
			msg:  &Message{},
			in:   []byte{0x39, 0x05, 0x00, 0x00, 0x01, 0x03, 0x00, 0x00, 0x00, 0x04, 0x41},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.msg.UnmarshalWire(tt.in); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
