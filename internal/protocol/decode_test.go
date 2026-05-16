package protocol

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type shortReadNoErrorReader struct {
	data []byte
}

func (r *shortReadNoErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	return n, nil
}

func TestReadHeader(t *testing.T) {
	expectedHeader := &Header{
		MagicByte: magicByte,
		Version:   version,
		Type:      uint8(TypeHandshake),
		Length:    24,
	}

	headerData, err := expectedHeader.MarshalWire()
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name           string
		reader         io.Reader
		wantHeader     *Header
		wantErr        bool
		wantErrContain string
	}{
		{
			name:       "valid header",
			reader:     bytes.NewReader(headerData),
			wantHeader: expectedHeader,
		},
		{
			name:           "short read without read error",
			reader:         &shortReadNoErrorReader{data: headerData[:headerLength-1]},
			wantErr:        true,
			wantErrContain: "read header, got 9 bytes; expected 10",
		},
		{
			name:           "reader returns error",
			reader:         errorReader{err: io.ErrUnexpectedEOF},
			wantErr:        true,
			wantErrContain: "read header: unexpected EOF",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotHeader, err := ReadHeader(tc.reader)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tc.wantErrContain != "" && !strings.Contains(err.Error(), tc.wantErrContain) {
					t.Fatalf("expected error to contain %q, got %q", tc.wantErrContain, err.Error())
				}

				if gotHeader != nil {
					t.Fatalf("expected nil header, got %v", gotHeader)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.wantHeader, gotHeader) {
				t.Fatalf("expected %v; got %v", tc.wantHeader, gotHeader)
			}
		})
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read(_ []byte) (int, error) {
	if r.err == nil {
		return 0, errors.New("errorReader err is nil")
	}

	return 0, r.err
}
