//go:build !windows

package tmuxserver

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// lxProbeRoot stages an isolated runtime root with a verified leaf for
// dial/probe/spawn tests. The test's t.TempDir owns its cleanup.
func lxProbeRoot(t *testing.T) (string, string) {
	t.Helper()
	short := shortTestTempDir(t)
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
	return root, SocketPath(runtime)
}

// lxBareRoot stages a runtime root with no leaf: the shape an unstaged
// root leaves behind (the delegated leaf custody refuses). The test's
// t.TempDir owns its cleanup.
func lxBareRoot(t *testing.T) (string, string) {
	t.Helper()
	short := shortTestTempDir(t)
	root := filepath.Join(short, "r")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = resolved
	return root, filepath.Join(root, RuntimeDirName, SocketName)
}

// stageStaleSocket binds a unix socket and closes the descriptor
// without unlinking the path: the shape a SIGKILLed server leaves
// behind (Go's net listener unlinks on close, so raw syscalls stage
// it). Dialing the staged path reports ECONNREFUSED.
func stageStaleSocket(t *testing.T, socket string) {
	t.Helper()
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(fd)
	if err := syscall.Bind(fd, &syscall.SockaddrUnix{Name: socket}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestUnixDialerLiveStaleUnknown(t *testing.T) {
	_, socket := lxProbeRoot(t)
	dialer := UnixDialer{}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	if got := dialer.Dial(socket); got != DialLive {
		listener.Close()
		t.Fatalf("live socket = %s, want live", got)
	}
	// A refused socket stages through a bound-then-closed descriptor;
	// closing the Go listener would unlink the path instead.
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	stageStaleSocket(t, socket)
	if got := dialer.Dial(socket); got != DialStale {
		t.Fatalf("stale socket = %s, want stale", got)
	}
}

func TestUnixDialerMissingDirectoryIsUnknown(t *testing.T) {
	dialer := UnixDialer{}
	root := shortTestTempDir(t)
	// A missing parent is a failed socket observation under pinned
	// SPEC.v0.7.0 §4.C's successful-status-only absence rule.
	if got := dialer.Dial(filepath.Join(root, "nodir", "ax.sock")); got != DialUnknown {
		t.Fatalf("missing directory = %s, want unknown", got)
	}
}

func TestServerProberMapsDialOutcomes(t *testing.T) {
	root, socket := lxProbeRoot(t)
	admission := RealmAdmission{Admitted: terminalbackend.Admitted{Capabilities: []string{"credential_capable_execution_realm"}}, RawGeneration: lxGeneration}
	cases := []struct {
		name    string
		dial    DialOutcome
		running bool
		unknown bool
	}{
		{"stale", DialStale, false, false},
		{"unknown", DialUnknown, false, true},
		{"live", DialLive, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prober := ServerProber{
				Root:     root,
				Platform: scalar.PlatformMacOS,
				Dialer:   &fakeDialer{outcomes: []DialOutcome{tc.dial}},
				Admit: func() (RealmAdmission, error) {
					return admission, nil
				},
			}
			report, err := prober.Probe(socket)
			if tc.unknown {
				if err == nil {
					t.Fatal("unknown dial reported")
				} else {
					requireLocalCode(t, err, "tmux_server_spawn_failed", "server probe")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if report.Running != tc.running {
				t.Fatalf("running = %v, want %v", report.Running, tc.running)
			}
			if tc.running && report.Admission.RawGeneration != lxGeneration {
				t.Fatalf("admission = %+v", report.Admission)
			}
		})
	}
}

func TestServerProberAdmissionFailurePassesThrough(t *testing.T) {
	root, socket := lxProbeRoot(t)
	boom := errors.New("reconcile failed")
	prober := ServerProber{
		Root:     root,
		Platform: scalar.PlatformMacOS,
		Dialer:   &fakeDialer{outcomes: []DialOutcome{DialLive}},
		Admit: func() (RealmAdmission, error) {
			return RealmAdmission{}, boom
		},
	}
	if _, err := prober.Probe(socket); err != boom {
		t.Fatalf("err = %v, want the admission failure", err)
	}
}

func TestServerProberEnforcesLengthAndCustody(t *testing.T) {
	root, socket := lxProbeRoot(t)
	prober := ServerProber{
		Root:     root,
		Platform: scalar.PlatformMacOS,
		Dialer:   &fakeDialer{outcomes: []DialOutcome{DialLive}},
		Admit: func() (RealmAdmission, error) {
			return RealmAdmission{}, nil
		},
	}
	longSocket := socket
	for len(longSocket) < 104 {
		longSocket += "x"
	}
	if _, err := prober.Probe(longSocket); err == nil {
		t.Fatal("overlong socket probed")
	} else {
		requireLocalCode(t, err, "tmux_socket_path_too_long", "socket path length")
	}
	if _, err := prober.Probe("/tmp/ax-ambient.sock"); err == nil {
		t.Fatal("ambient socket probed")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket placement")
	}
	// The compliant socket on an unstaged root: the delegated leaf
	// custody refuses, propagated.
	bare, bareSocket := lxBareRoot(t)
	bareProber := ServerProber{
		Root:     bare,
		Platform: scalar.PlatformMacOS,
		Dialer:   &fakeDialer{outcomes: []DialOutcome{DialLive}},
		Admit: func() (RealmAdmission, error) {
			return RealmAdmission{}, nil
		},
	}
	if _, err := bareProber.Probe(bareSocket); err == nil {
		t.Fatal("absent leaf probed")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_runtime_dir", "runtime absent")
	}
}

func TestServerProberRefusesSocketSubstitutionBeforeConnect(t *testing.T) {
	root, socket := lxProbeRoot(t)
	primary, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
	decoyPath := socket + ".decoy"
	decoy, err := net.Listen("unix", decoyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer decoy.Close()
	if err := os.Chmod(decoyPath, 0o600); err != nil {
		t.Fatal(err)
	}

	previous := lstatCustodySocket
	calls := 0
	lstatCustodySocket = func(path string) (os.FileInfo, error) {
		calls++
		if calls == 2 {
			if err := os.Rename(decoyPath, socket); err != nil {
				return nil, err
			}
		}
		return previous(path)
	}
	defer func() { lstatCustodySocket = previous }()

	dialer := &fakeDialer{outcomes: []DialOutcome{DialLive}}
	prober := ServerProber{
		Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer,
		Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil },
	}
	if _, err := prober.Probe(socket); err == nil {
		t.Fatal("socket replaced during custody validation was connected")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket identity")
	}
	if len(dialer.calls) != 0 {
		t.Fatalf("substituted socket connected before refusal: %q", dialer.calls)
	}
}

func TestServerProberRefusesSocketSymlinkBeforeConnect(t *testing.T) {
	root, socket := lxProbeRoot(t)
	target := socket + ".target"
	if err := os.WriteFile(target, []byte("not an endpoint"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, socket); err != nil {
		t.Fatal(err)
	}
	dialer := &fakeDialer{outcomes: []DialOutcome{DialLive}}
	prober := ServerProber{
		Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer,
		Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil },
	}
	if _, err := prober.Probe(socket); err == nil {
		t.Fatal("socket symlink was connected")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket kind")
	}
	if len(dialer.calls) != 0 {
		t.Fatalf("symlink connected before refusal: %q", dialer.calls)
	}
}

func TestServerProberRefusesForeignSocketBeforeConnect(t *testing.T) {
	root, socket := lxProbeRoot(t)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
	previous := lstatCustodySocket
	lstatCustodySocket = func(path string) (os.FileInfo, error) {
		info, err := previous(path)
		if err != nil || path != socket {
			return info, err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatalf("socket stat is %T, want *syscall.Stat_t", info.Sys())
		}
		foreign := *stat
		foreign.Uid++
		return socketFileInfo{FileInfo: info, stat: &foreign}, nil
	}
	defer func() { lstatCustodySocket = previous }()

	dialer := &fakeDialer{outcomes: []DialOutcome{DialLive}}
	prober := ServerProber{
		Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer,
		Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil },
	}
	if _, err := prober.Probe(socket); err == nil {
		t.Fatal("foreign-owned socket was connected")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ownership")
	}
	if len(dialer.calls) != 0 {
		t.Fatalf("foreign socket connected before refusal: %q", dialer.calls)
	}
}

func TestServerProberRefusesPermissiveSocketBeforeConnect(t *testing.T) {
	root, socket := lxProbeRoot(t)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := os.Chmod(socket, 0o644); err != nil {
		t.Fatal(err)
	}
	dialer := &fakeDialer{outcomes: []DialOutcome{DialLive}}
	prober := ServerProber{
		Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer,
		Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil },
	}
	if _, err := prober.Probe(socket); err == nil {
		t.Fatal("group/world-readable socket was connected")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket permissions")
	}
	if len(dialer.calls) != 0 {
		t.Fatalf("permissive socket connected before refusal: %q", dialer.calls)
	}
}

type socketFileInfo struct {
	os.FileInfo
	stat *syscall.Stat_t
}

func (info socketFileInfo) Sys() any { return info.stat }

func TestServerSpawnerRefusesAbsentLeaf(t *testing.T) {
	// The well-pinned spawn on an unstaged root: the delegated leaf
	// custody refuses before any exec.
	root, socket := lxBareRoot(t)
	good := []string{"tmux", "-S", socket, "new-session", "-d", "-s", lxInstance}
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: newFakeRunner().queue("new-session", "", 0)}
	if err := spawner.Spawn(socket, good); err == nil {
		t.Fatal("absent leaf spawned")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_runtime_dir", "runtime absent")
	}
}

func TestServerSpawnerPinAndCustody(t *testing.T) {
	root, socket := lxProbeRoot(t)
	runner := newFakeRunner().queue("new-session", "", 0)
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
	good, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	if err := spawner.Spawn(socket, good); err != nil {
		t.Fatalf("spawn refused: %v", err)
	}
	if runner.callCount() != 1 {
		t.Fatalf("spawn calls = %d, want 1", runner.callCount())
	}
	for _, bad := range [][]string{
		{"tmux", "-S", "/tmp/ax-ambient.sock", "new-session"},
		{"tmux", "new-session"},
		{"tmux", "-S"},
		{"other", "-S", socket, "new-session"},
	} {
		if err := spawner.Spawn(socket, bad); err == nil {
			t.Fatalf("spawn %q admitted", bad)
		} else {
			requireLocalCode(t, err, "tmux_ambient_server_reuse", "spawn socket")
		}
	}
	if err := spawner.Spawn("/tmp/ax-ambient.sock", good); err == nil {
		t.Fatal("ambient socket spawn admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket placement")
	}
}

func TestServerSpawnerUnlinksStaleSocket(t *testing.T) {
	root, socket := lxProbeRoot(t)
	stageStaleSocket(t, socket)
	var unlinked []string
	runner := newFakeRunner().queue("new-session", "", 0)
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner, Hooks: &SpawnHooks{AfterUnlink: func(socket string) {
		unlinked = append(unlinked, socket)
	}}}
	good, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	if err := spawner.Spawn(socket, good); err != nil {
		t.Fatalf("spawn over stale socket refused: %v", err)
	}
	if len(unlinked) != 1 || unlinked[0] != socket {
		t.Fatalf("unlinked = %q", unlinked)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("stale socket survives: %v", err)
	}
	// Retry converges: no stale file, spawn proceeds.
	runner.queue("new-session", "", 0)
	if err := spawner.Spawn(socket, good); err != nil {
		t.Fatalf("retry refused: %v", err)
	}
}

func TestServerSpawnerRefusesNonSocketFile(t *testing.T) {
	root, socket := lxProbeRoot(t)
	if err := os.WriteFile(socket, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: newFakeRunner()}
	good, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	if err := spawner.Spawn(socket, good); err == nil {
		t.Fatal("non-socket file unlinked")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket kind")
	}
	if _, err := os.Lstat(socket); err != nil {
		t.Fatalf("non-socket file removed: %v", err)
	}
}

func TestServerSpawnerRefusesSocketSubstitutionBeforeUnlinkOrSpawn(t *testing.T) {
	root, socket := lxProbeRoot(t)
	primary, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
	decoyPath := socket + ".decoy"
	decoy, err := net.Listen("unix", decoyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer decoy.Close()
	if err := os.Chmod(decoyPath, 0o600); err != nil {
		t.Fatal(err)
	}

	previous := lstatCustodySocket
	calls := 0
	lstatCustodySocket = func(path string) (os.FileInfo, error) {
		if path == socket {
			calls++
			if calls == 2 {
				if err := os.Rename(decoyPath, socket); err != nil {
					return nil, err
				}
			}
		}
		return previous(path)
	}
	defer func() { lstatCustodySocket = previous }()

	runner := newFakeRunner()
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
	argv, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	err = spawner.Spawn(socket, argv)
	requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket identity")
	if calls != 2 {
		t.Fatalf("socket custody lstat calls = %d, want exactly two", calls)
	}
	if runner.callCount() != 0 {
		t.Fatalf("substituted socket spawn executed %d commands", runner.callCount())
	}
	if _, err := os.Lstat(socket); err != nil {
		t.Fatalf("refused substituted socket was unlinked: %v", err)
	}
}

func TestServerSpawnerRefusesSymlinkSocketBeforeMutation(t *testing.T) {
	root, socket := lxProbeRoot(t)
	target := socket + ".target"
	if err := os.WriteFile(target, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, socket); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
	argv, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	if err := spawner.Spawn(socket, argv); err == nil {
		t.Fatal("socket symlink was unlinked or spawned")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket kind")
	}
	if runner.callCount() != 0 {
		t.Fatalf("symlink socket spawned %d commands", runner.callCount())
	}
	if _, err := os.Lstat(socket); err != nil {
		t.Fatalf("refused socket symlink was unlinked: %v", err)
	}
}

func TestServerSpawnerRefusesForeignOrPermissiveSocketBeforeMutation(t *testing.T) {
	for _, refusal := range []struct {
		name    string
		detail  string
		mode    os.FileMode
		foreign bool
	}{
		{name: "foreign-owner", detail: "socket ownership", mode: 0o600, foreign: true},
		{name: "permissive-mode", detail: "socket permissions", mode: 0o644},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			root, socket := lxProbeRoot(t)
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err := os.Chmod(socket, refusal.mode); err != nil {
				t.Fatal(err)
			}
			if refusal.foreign {
				previous := lstatCustodySocket
				lstatCustodySocket = func(path string) (os.FileInfo, error) {
					info, err := previous(path)
					if err != nil || path != socket {
						return info, err
					}
					stat, ok := info.Sys().(*syscall.Stat_t)
					if !ok {
						t.Fatalf("socket stat is %T, want *syscall.Stat_t", info.Sys())
					}
					foreign := *stat
					foreign.Uid++
					return socketFileInfo{FileInfo: info, stat: &foreign}, nil
				}
				t.Cleanup(func() { lstatCustodySocket = previous })
			}
			runner := newFakeRunner()
			spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
			argv, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
			if err != nil {
				t.Fatal(err)
			}
			err = spawner.Spawn(socket, argv)
			requireLocalCode(t, err, "tmux_unsafe_socket_path", refusal.detail)
			if runner.callCount() != 0 {
				t.Fatalf("%s socket spawned %d commands", refusal.name, runner.callCount())
			}
			if _, err := os.Lstat(socket); err != nil {
				t.Fatalf("refused socket was unlinked: %v", err)
			}
		})
	}
}

func TestServerSpawnerMapsRunnerOutcome(t *testing.T) {
	root, socket := lxProbeRoot(t)
	good, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	failing := newFakeRunner().queue("new-session", "", 1)
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: failing}
	if err := spawner.Spawn(socket, good); err == nil {
		t.Fatal("failed spawn admitted")
	} else {
		requireLocalCode(t, err, "tmux_server_spawn_failed", "server spawn")
	}
	broken := newFakeRunner().failTransport("new-session", errors.New("no exec"))
	spawner = ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: broken}
	if err := spawner.Spawn(socket, good); err == nil {
		t.Fatal("transport failure admitted")
	} else {
		requireLocalCode(t, err, "tmux_server_spawn_failed", "server spawn")
	}
}
