//go:build !windows

package tmuxserver

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// lxCustodyRoot stages a runtime root with a verified leaf and returns
// the root and the derived socket.
func lxCustodyRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = resolved
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	runtime, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	return root, SocketPath(runtime)
}

func TestCheckSocketCustodyAdmitsCompliant(t *testing.T) {
	root, socket := lxCustodyRoot(t)
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err != nil {
		t.Fatalf("compliant socket refused: %v", err)
	}
}

func TestCheckSocketCustodyAdmitsLiveSocket(t *testing.T) {
	// Short root: the temp-dir socket path would exceed the platform
	// sun_path (bind: invalid argument), which is B18 itself.
	short, err := os.MkdirTemp("/tmp", "axsock")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(short) })
	root := filepath.Join(short, "r")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = resolved
	runtime, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	socket := SocketPath(runtime)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err != nil {
		t.Fatalf("live socket refused: %v", err)
	}
}

func TestCheckSocketCustodyRefusesUnsafeRoot(t *testing.T) {
	root, socket := lxCustodyRoot(t)
	if err := os.Chmod(root, 0o777); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(root, 0o700)
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("0777 root admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	}
}

func TestCheckSocketCustodyRefusesForeignRoot(t *testing.T) {
	root, socket := lxCustodyRoot(t)
	previous := effectiveUID
	effectiveUID = func() int { return previous() + 100000 }
	defer func() { effectiveUID = previous }()
	// The leaf custody delegates to the same seam, so a foreign root
	// refuses at the leaf ownership arm first; the bind step still
	// refuses rather than admitting.
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("foreign root admitted")
	}
}

func TestCheckSocketCustodyRefusesWritableAncestor(t *testing.T) {
	base := t.TempDir()
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	base = resolved
	middle := filepath.Join(base, "middle")
	if err := os.Mkdir(middle, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(middle, "root")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(middle, 0o777); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(middle, 0o755)
	socket := filepath.Join(root, RuntimeDirName, SocketName)
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("writable ancestor admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	}
}

func TestCheckSocketCustodyAdmitsStickyAncestor(t *testing.T) {
	base := t.TempDir()
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	base = resolved
	middle := filepath.Join(base, "middle")
	if err := os.Mkdir(middle, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(middle, "root")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(middle, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(middle, 0o755)
	socket := filepath.Join(root, RuntimeDirName, SocketName)
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err != nil {
		t.Fatalf("sticky ancestor refused: %v", err)
	}
}

func TestCheckSocketCustodyRefusesFileAncestor(t *testing.T) {
	base := t.TempDir()
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	base = resolved
	blocker := filepath.Join(base, "axevil")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(blocker, "root")
	socket := filepath.Join(root, RuntimeDirName, SocketName)
	// The leaf custody refuses first (no directory under a file), so
	// this input pins the refusal without naming the ancestor arm.
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("file ancestor admitted")
	}
}

func TestCheckSocketAncestorKindArm(t *testing.T) {
	// The ancestor kind arm refuses a regular file directly: a file
	// staged as an ancestor of a resolved root. The root itself lives
	// under the temp dir; the walk climbs the lexical path, so a
	// file on the chain refuses at the no-follow open.
	base := t.TempDir()
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	base = resolved
	fileAncestor := filepath.Join(base, "fileancestor")
	if err := os.WriteFile(fileAncestor, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	// Point the walk at a root whose ancestor chain contains the file
	// by nesting the root under a directory that IS the file: lexical
	// root = file/root. The no-follow open of the file ancestor
	// refuses, which the ancestor arm reports.
	nested := filepath.Join(fileAncestor, "root")
	if err := checkCustodyAncestors(nested); err == nil {
		t.Fatal("file ancestor admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	}
	if err := checkCustodyAncestors(root); err != nil {
		t.Fatalf("clean ancestors refused: %v", err)
	}
}

func TestCheckSocketCustodyRefusesNonSocketFile(t *testing.T) {
	root, socket := lxCustodyRoot(t)
	if err := os.WriteFile(socket, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("regular file at socket path admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket kind")
	}
}

func TestCheckSocketCustodyPropagatesLeafRefusal(t *testing.T) {
	root := t.TempDir()
	socket := filepath.Join(root, RuntimeDirName, SocketName)
	// No leaf: the delegated leaf custody refuses with its own code,
	// which the bind step propagates instead of masking.
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("absent leaf admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_runtime_dir", "runtime absent")
	}
}
