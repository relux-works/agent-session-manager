package tmuxserver

import (
	"testing"
)

func TestBuildArgvAddressesOnlyTheDerivedSocket(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	socket := SocketPath(dir)
	argv, err := BuildArgv(dir, socket, fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"tmux", "-S", socket, "new-session", "-d", "-s", fixtureSession, "ax", "pane", fixtureSession}
	if len(argv) != len(want) {
		t.Fatalf("argv = %q, want %q", argv, want)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv = %q, want %q", argv, want)
		}
	}
}

func TestBuildArgvRefusesAmbientSocket(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	hostile := hostileAmbient()
	for _, tc := range []struct {
		name   string
		socket string
	}{
		{"tmux env value", hostile.TMUXEnv},
		{"default path", hostile.DefaultPath},
		{"inherited socket", hostile.InheritedSocket},
		{"conventional name", hostile.ConventionalName},
		{"tmpdir", hostile.TMUXTmpDir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildArgv(dir, tc.socket, fixtureSession)
			requireLocalError(t, err, "tmux_ambient_server_reuse", "spawn socket")
		})
	}
}

// The landed argv shape gate is wired at the spawn vector: a runtime
// directory no filesystem names (invalid UTF-8) refuses there instead
// of reaching the spawner. Reachable only by direct call — through
// Acquire the directory always names a created directory.
func TestBuildArgvRefusesInvalidArgvShape(t *testing.T) {
	dir := string([]byte{0xff, 0xfe})
	_, err := BuildArgv(dir, SocketPath(dir), fixtureSession)
	requireLocalError(t, err, "tmux_invalid_arguments", "spawn argv")
}

func TestBuildArgvRefusesMalformedSession(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	for _, session := range []string{"", "12345", "not-a-uuid", "0198f4c8-7d40-7e55-8e6f-1234567890ag"} {
		_, err := BuildArgv(dir, SocketPath(dir), session)
		requireLocalError(t, err, "tmux_invalid_arguments", "session identity")
	}
}
