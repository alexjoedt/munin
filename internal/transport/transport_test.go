package transport

import "testing"

func TestSplitAddress(t *testing.T) {
	type netAddr struct {
		network string
		address string
	}

	tests := []struct {
		name     string
		addr     string
		expected netAddr
		wantErr  bool
	}{
		{
			name:     "tcp network without protocol",
			addr:     "127.0.0.1:8080",
			expected: netAddr{network: "tcp", address: "127.0.0.1:8080"},
		},
		{
			name:     "tcp network with protocol",
			addr:     "tcp://127.0.0.1:8080",
			expected: netAddr{network: "tcp", address: "127.0.0.1:8080"},
		},
		{
			name:     "unix socket without protocol",
			addr:     "/path/to/socket.sock",
			expected: netAddr{network: "unix", address: "/path/to/socket.sock"},
		},
		{
			name:     "unix socket without protocol",
			addr:     "unix:///path/to/socket.sock",
			expected: netAddr{network: "unix", address: "/path/to/socket.sock"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNet, gotAddr, err := SplitAddr(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Fatal("got error = nil; want non-nil error")
				}
			}

			if gotNet != tt.expected.network {
				t.Fatalf("got %s; want %s\n", gotNet, tt.expected.network)
			}

			if gotAddr != tt.expected.address {
				t.Fatalf("got %s; want %s\n", gotAddr, tt.expected.address)
			}
		})
	}

}
