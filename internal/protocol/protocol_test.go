package protocol

import (
	"bytes"
	"testing"
)

func TestMarhsalBinary(t *testing.T) {
	buf := []byte(`{"message": "hello munin"}`)
	msg := &Message{
		MagicByte: magicByte,
		Version:   version,
		Type:      uint8(TypeHandshake),
		Length:    uint32(len(buf)),
		Payload:   buf,
	}

	data, err := msg.MarshalBinary()
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		t.FailNow()
	}

	expectedLen := headerLength + len(msg.Payload)
	if len(data) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(data))
		t.FailNow()
	}

	if !bytes.Equal(data[:4], []byte{57, 5, 0, 0}) {
		t.Errorf("exptected '%s', got %s", []byte{57, 5, 0, 0}, data[:3])
		t.FailNow()
	}

	if data[4] != version {
		t.Errorf("expected %d, got %d", version, data[4])
		t.FailNow()
	}

	if data[5] != byte(TypeHandshake) {
		t.Errorf("expected %d, got %d", TypeHeartbeat, data[5])
		t.FailNow()
	}

	if !bytes.Equal(data[6:10], []byte{0, 0, 0, 26}) {
		t.Errorf("expected []byte{0, 0, 0, 26}, got %b", data[6:10])
		t.FailNow()
	}

	if !bytes.Equal(data[10:], buf) {
		t.Errorf("expteted %s, got %s", string(buf), string(data[10:]))
		t.FailNow()
	}
}
