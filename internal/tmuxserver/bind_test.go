package tmuxserver

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestCheckSocketLengthBounds(t *testing.T) {
	cases := []struct {
		name     string
		platform scalar.Platform
		admit    int
		refuse   int
	}{
		{"darwin", scalar.PlatformMacOS, 103, 104},
		{"linux", scalar.PlatformLinux, 107, 108},
		{"wsl2", scalar.PlatformWSL2, 107, 108},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckSocketLength(strings.Repeat("a", tc.admit), tc.platform); err != nil {
				t.Fatalf("length %d refused: %v", tc.admit, err)
			}
			if err := CheckSocketLength(strings.Repeat("a", tc.refuse), tc.platform); err == nil {
				t.Fatalf("length %d admitted", tc.refuse)
			} else {
				requireLocalCode(t, err, "tmux_socket_path_too_long", "socket path length")
			}
		})
	}
}

func TestCheckSocketLengthCountsBytes(t *testing.T) {
	// Multibyte runes count their UTF-8 bytes: 52 two-byte runes are
	// 104 bytes and refuse on darwin, while 51 (102 bytes) admit.
	if err := CheckSocketLength(strings.Repeat("é", 51), scalar.PlatformMacOS); err != nil {
		t.Fatalf("102-byte socket refused: %v", err)
	}
	if err := CheckSocketLength(strings.Repeat("é", 52), scalar.PlatformMacOS); err == nil {
		t.Fatal("104-byte socket admitted")
	} else {
		requireLocalCode(t, err, "tmux_socket_path_too_long", "socket path length")
	}
}

func TestCheckSocketCustodyRefusesMisplacedSocket(t *testing.T) {
	root := t.TempDir()
	for _, socket := range []string{
		"/tmp/ax-ambient.sock",
		root + "/tmux/ax.sock.evil",
		root + "/other/ax.sock",
		root + "/tmux",
	} {
		if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
			t.Fatalf("misplaced socket %q admitted", socket)
		} else {
			requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket placement")
		}
	}
}
