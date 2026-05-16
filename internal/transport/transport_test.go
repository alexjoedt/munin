package transport

import (
	"path/filepath"
	"testing"
)

func TestParseAddr(t *testing.T) {
	relUnixPath := "./testdata/socket.sock"
	absUnixPath, err := filepath.Abs(relUnixPath)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", relUnixPath, err)
	}

	tests := []struct {
		name    string
		input   string
		want    *Addr
		wantErr bool
	}{
		{
			name:  "tcp address without scheme",
			input: "127.0.0.1:8080",
			want: &Addr{
				Network: "tcp",
				Host:    "127.0.0.1",
				Port:    8080,
			},
		},
		{
			name:  "tcp address with scheme",
			input: "tcp://127.0.0.1:8080",
			want: &Addr{
				Network: "tcp",
				Host:    "127.0.0.1",
				Port:    8080,
			},
		},
		{
			name:  "unix address absolute path",
			input: "/tmp/munin.sock",
			want: &Addr{
				Network: "unix",
				Path:    "/tmp/munin.sock",
			},
		},
		{
			name:  "unix address with scheme and relative path",
			input: "unix://" + relUnixPath,
			want: &Addr{
				Network: "unix",
				Path:    absUnixPath,
			},
		},
		{
			name:    "unsupported scheme",
			input:   "udp://127.0.0.1:8080",
			wantErr: true,
		},
		{
			name:    "tcp with invalid port",
			input:   "tcp://127.0.0.1:not-a-number",
			wantErr: true,
		},
		{
			name:    "tcp with port too large",
			input:   "tcp://127.0.0.1:65536",
			wantErr: true,
		},
		{
			name:    "tcp missing port",
			input:   "tcp://127.0.0.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAddr(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("got error = nil; want non-nil error")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseAddr(%q) returned error: %v", tt.input, err)
			}

			if got.Network != tt.want.Network {
				t.Fatalf("Network = %q, want %q", got.Network, tt.want.Network)
			}
			if got.Host != tt.want.Host {
				t.Fatalf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Fatalf("Port = %d, want %d", got.Port, tt.want.Port)
			}
			if got.Path != tt.want.Path {
				t.Fatalf("Path = %q, want %q", got.Path, tt.want.Path)
			}
		})
	}
}
