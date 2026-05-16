package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

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
