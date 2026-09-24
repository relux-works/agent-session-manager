package tmuxserver

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestObserveAmbientRecordsWithoutUsing(t *testing.T) {
	hostile := hostileAmbient()
	observed := ObserveAmbient(func(name string) (string, bool) {
		switch name {
		case "TMUX":
			return hostile.TMUXEnv, true
		case "TMUX_TMPDIR":
			return hostile.TMUXTmpDir, true
		default:
			return "", false
		}
	}, hostile.InheritedSocket, hostile.DefaultPath, hostile.ConventionalName)
	if observed != hostile {
		t.Fatalf("observed = %+v, want %+v", observed, hostile)
	}
}

func TestObserveAmbientNilLookupKeepsOutOfBand(t *testing.T) {
	observed := ObserveAmbient(nil, "/tmp/x.sock", "/tmp/tmux-501/default", "default")
	if observed.TMUXEnv != "" || observed.TMUXTmpDir != "" {
		t.Fatalf("nil lookup observed env: %+v", observed)
	}
	if observed.InheritedSocket != "/tmp/x.sock" {
		t.Fatalf("observed = %+v", observed)
	}
}

func TestResolveSocketDerivesIgnoringAmbient(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	socket, err := ResolveSocket(dir, "", hostileAmbient())
	if err != nil {
		t.Fatal(err)
	}
	if socket != SocketPath(dir) {
		t.Fatalf("socket = %q", socket)
	}
	for _, ambient := range []string{
		hostileAmbient().TMUXEnv,
		hostileAmbient().TMUXTmpDir,
		hostileAmbient().InheritedSocket,
		hostileAmbient().DefaultPath,
		hostileAmbient().ConventionalName,
	} {
		if strings.Contains(socket, ambient) || socket == ambient {
			t.Fatalf("socket %q reuses ambient %q", socket, ambient)
		}
	}
}

func TestResolveSocketRefusesEveryOverrideVector(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	hostile := hostileAmbient()
	vectors := map[string]string{
		"environment variable": hostile.TMUXEnv,
		"tmux tmpdir":          hostile.TMUXTmpDir,
		"inherited socket":     hostile.InheritedSocket,
		"default path":         hostile.DefaultPath,
		"conventional name":    hostile.ConventionalName,
		"whitespace only":      "  ",
	}
	for name, override := range vectors {
		t.Run(name, func(t *testing.T) {
			_, err := ResolveSocket(dir, override, Ambient{})
			requireLocalError(t, err, "tmux_ambient_server_reuse", "socket override")
		})
	}
}

func TestResolveSocketRefusesAmbientCollision(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	derived := SocketPath(dir)
	for _, tc := range []struct {
		name    string
		ambient Ambient
	}{
		{"tmux env", Ambient{TMUXEnv: derived}},
		{"tmux tmpdir", Ambient{TMUXTmpDir: derived}},
		{"inherited socket", Ambient{InheritedSocket: derived}},
		{"default path", Ambient{DefaultPath: derived}},
		{"conventional name", Ambient{ConventionalName: derived}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveSocket(dir, "", tc.ambient)
			requireLocalError(t, err, "tmux_ambient_server_reuse", "ambient collision")
		})
	}
}

// The collision gate compares cleaned spellings: an ambient value that
// names the derived socket through a `/./` segment or a doubled
// separator refuses like the byte-identical spelling. Each alias is
// asserted to clean to the derived socket so the row cannot pass on a
// non-colliding value.
func TestResolveSocketRefusesAmbientCollisionUncleanSpellings(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	derived := SocketPath(dir)
	for _, tc := range []struct {
		name  string
		alias string
	}{
		{"dot segment", dir + "/./" + SocketName},
		{"doubled separator", dir + "//" + SocketName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := filepath.Clean(tc.alias); got != derived {
				t.Fatalf("alias %q cleans to %q, want the derived socket %q", tc.alias, got, derived)
			}
			if tc.alias == derived {
				t.Fatalf("alias %q is byte-identical; the row is vacuous", tc.alias)
			}
			_, err := ResolveSocket(dir, "", Ambient{DefaultPath: tc.alias})
			requireLocalError(t, err, "tmux_ambient_server_reuse", "ambient collision")
		})
	}
}

// The TMUX member collides in the encoding tmux(1) really produces:
// `<socket>,<pid>,<index>`. A value naming the derived socket in
// that encoding refuses; the suite asserts the encoding shape (a
// comma past the socket) so the row cannot pass on a bare path.
func TestResolveSocketRefusesTMUXEnvRealEncoding(t *testing.T) {
	dir := runtimeDir(t, runtimeRoot(t))
	derived := SocketPath(dir)
	encoded := derived + ",12345,0"
	if !strings.Contains(encoded, ",") {
		t.Fatalf("encoded value %q carries no TMUX separator", encoded)
	}
	if got := tmuxEnvSocket(encoded); got != derived {
		t.Fatalf("socket component = %q, want the derived socket %q", got, derived)
	}
	_, err := ResolveSocket(dir, "", Ambient{TMUXEnv: encoded})
	requireLocalError(t, err, "tmux_ambient_server_reuse", "ambient collision")
}
