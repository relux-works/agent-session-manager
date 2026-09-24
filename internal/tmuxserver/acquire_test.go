package tmuxserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

var errProbeBoom = errors.New("probe boom")

func TestAcquireForegroundSpawnsDedicatedServer(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	req.Ambient = hostileAmbient()
	spy := &spawnSpy{}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{}, nil
	}
	req.Deps.Spawn = spy.spawn
	outcome, err := Acquire(req)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Via != "spawned" {
		t.Fatalf("via = %q, want spawned", outcome.Via)
	}
	if outcome.Socket != SocketPath(filepath.Join(root, RuntimeDirName)) {
		t.Fatalf("socket = %q", outcome.Socket)
	}
	if spy.calls != 1 || spy.socket != outcome.Socket {
		t.Fatalf("spawn calls = %d socket = %q", spy.calls, spy.socket)
	}
	if len(spy.argv) < 3 || spy.argv[0] != "tmux" || spy.argv[1] != "-S" || spy.argv[2] != outcome.Socket {
		t.Fatalf("spawn argv = %q", spy.argv)
	}
	for i, ambient := range []string{req.Ambient.TMUXEnv, req.Ambient.TMUXTmpDir, req.Ambient.InheritedSocket, req.Ambient.DefaultPath, req.Ambient.ConventionalName} {
		if ambient == "" {
			continue
		}
		if outcome.Socket == ambient {
			t.Fatalf("socket reuses ambient vector %d", i)
		}
		for _, arg := range spy.argv {
			if arg == ambient {
				t.Fatalf("argv carries ambient vector %d: %q", i, arg)
			}
		}
	}
	wantTail := []string{"ax", "pane", fixtureSession}
	tail := spy.argv[len(spy.argv)-3:]
	for i := range wantTail {
		if tail[i] != wantTail[i] {
			t.Fatalf("argv tail = %q, want %q", tail, wantTail)
		}
	}
	// The spawned vector is pinned element-by-element through the
	// entry, exactly as the direct BuildArgv test pins it: a dropped
	// flag or a retargeted tail reddens here, not only at the helper.
	wantVector := []string{"tmux", "-S", outcome.Socket, "new-session", "-d", "-s", fixtureSession, "ax", "pane", fixtureSession}
	if len(spy.argv) != len(wantVector) {
		t.Fatalf("spawn argv = %q, want %q", spy.argv, wantVector)
	}
	for i := range wantVector {
		if spy.argv[i] != wantVector[i] {
			t.Fatalf("spawn argv = %q, want %q", spy.argv, wantVector)
		}
	}
}

func socketDir(t *testing.T, root string) string {
	t.Helper()
	return runtimeDir(t, root)
}

func TestAcquireForegroundAttachesToRunningAttested(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{}
	probed := ""
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		probed = socket
		return ServerReport{Running: true, Admission: boundAdmission()}, nil
	}
	req.Deps.Spawn = spy.spawn
	outcome, err := Acquire(req)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Via != "attached-running" {
		t.Fatalf("via = %q", outcome.Via)
	}
	if probed != outcome.Socket || outcome.Socket != SocketPath(socketDir(t, root)) {
		t.Fatalf("probed = %q socket = %q", probed, outcome.Socket)
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on attach", spy.calls)
	}
}

func TestAcquireForegroundRefusesRunningUnattested(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{Running: true, Admission: RealmAdmission{
			Admitted:      terminalbackend.Admitted{Capabilities: []string{fixtureDecoyCapacity}},
			RawGeneration: fixtureGeneration,
		}}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_readiness_not_authorizing", "server attestation")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on unattested", spy.calls)
	}
}

// A stale admission — yesterday's live proof replayed against today's
// typed generation — fakes no running attested server: attach refuses
// on the generation arm with no spawn and no repair.
func TestAcquireForegroundRefusesWrongGenerationAdmission(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{Running: true, Admission: staleAdmission()}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_readiness_not_authorizing", "server generation")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on stale admission", spy.calls)
	}
}

// The realistic post-spawn state — the server runs, no evidence has
// been reconciled yet, the admission carries nothing — refuses
// through the entry with no second spawn: a running server with zero
// admission rows is still running, never absent.
func TestAcquireForegroundRefusesRunningWithEmptyAdmission(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{Running: true, Admission: RealmAdmission{}}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_readiness_not_authorizing", "server attestation")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on a running server", spy.calls)
	}
}

// Same rule one member over: a running server whose admission
// carries the realm row but no generation refuses on the generation
// arm through the entry with no second spawn.
func TestAcquireForegroundRefusesRunningWithEmptyGeneration(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{Running: true, Admission: RealmAdmission{
			Admitted: realmAdmitted(),
		}}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_readiness_not_authorizing", "server generation")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on a running server", spy.calls)
	}
}

func TestAcquireForegroundProbeFailurePassesThrough(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{}, errProbeBoom
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	if !errors.Is(err, errProbeBoom) {
		t.Fatalf("err = %v, want the probe failure", err)
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on unknown", spy.calls)
	}
}

func TestAcquireForegroundSpawnFailureReportsWithoutFallback(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	spy := &spawnSpy{err: errors.New("exec failed")}
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_server_spawn_failed", "server spawn")
	if spy.calls != 1 {
		t.Fatalf("spawn calls = %d, want exactly 1", spy.calls)
	}
}

// The session identity is validated at the foreground entry before
// the runtime directory is ensured and the server probed, so a
// malformed session refuses before any side effect: no directory, no
// probe, no spawn.
func TestAcquireForegroundRefusesMalformedSession(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	req.SessionID = "12345"
	spy := &spawnSpy{}
	probed := false
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		probed = true
		return ServerReport{}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_invalid_arguments", "session identity")
	if probed {
		t.Fatal("server was probed before a malformed session refused")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0", spy.calls)
	}
	if _, statErr := os.Lstat(filepath.Join(root, RuntimeDirName)); !os.IsNotExist(statErr) {
		t.Fatalf("malformed session created the directory: stat err = %v", statErr)
	}
}

// A malformed runtime root refuses through the foreground entry with
// no directory, no probe, and no spawn: the lexical gate precedes
// every side effect on the creating path too.
func TestAcquireForegroundRefusesBadRoot(t *testing.T) {
	root := runtimeRoot(t)
	for _, tc := range []struct {
		name string
		root string
	}{
		{"relative", "relative/root"},
		{"missing", filepath.Join(root, "no-such-root")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			leaf := filepath.Join(root, RuntimeDirName)
			req := validRequest(t, tc.root)
			spy := &spawnSpy{}
			probed := false
			req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
				probed = true
				return ServerReport{}, nil
			}
			req.Deps.Spawn = spy.spawn
			_, err := Acquire(req)
			requireLocalError(t, err, "tmux_invalid_arguments", "runtime root")
			if probed {
				t.Fatal("server was probed before a bad root refused")
			}
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0", spy.calls)
			}
			if _, statErr := os.Lstat(leaf); !os.IsNotExist(statErr) {
				t.Fatalf("bad root created the directory: stat err = %v", statErr)
			}
		})
	}
}

func TestAcquireBackgroundContactsBroker(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	req := validRequest(t, root)
	req.Caller = CallerBackground
	req.Platform = scalar.PlatformMacOS
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	probed := false
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	outcome, err := Acquire(req)
	if err != nil {
		t.Fatal(err)
	}
	if !probed {
		t.Fatal("broker was not probed")
	}
	if outcome.Via != "broker" {
		t.Fatalf("via = %q", outcome.Via)
	}
	if outcome.Socket != SocketPath(socketDir(t, root)) {
		t.Fatalf("socket = %q", outcome.Socket)
	}
	if len(outcome.Argv) != 0 {
		t.Fatalf("broker outcome carries argv %q", outcome.Argv)
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on broker contact", spy.calls)
	}
	probe.requireSilent(t)
}

func TestAcquireBackgroundMissReturnsTypedUnavailable(t *testing.T) {
	live := brokerReport()
	for _, tc := range []struct {
		name   string
		report BrokerReport
	}{
		{"no broker", BrokerReport{}},
		{"foreign-user broker", BrokerReport{
			Principal: BrokerPrincipal{UID: foreignUID(), Generation: fixtureGeneration},
			Server:    boundAdmission(),
		}},
		{"generation-unbound principal", BrokerReport{
			Principal: BrokerPrincipal{UID: live.Principal.UID, Generation: fixtureStaleGen},
			Server:    boundAdmission(),
		}},
		{"unattested server", BrokerReport{
			Principal: live.Principal,
			Server:    RealmAdmission{RawGeneration: fixtureGeneration},
		}},
		{"stale server admission", BrokerReport{
			Principal: live.Principal,
			Server:    staleAdmission(),
		}},
		{"zero server admission", BrokerReport{
			Principal: live.Principal,
			Server:    RealmAdmission{},
		}},
		{"admitted server without generation", BrokerReport{
			Principal: live.Principal,
			Server:    RealmAdmission{Admitted: realmAdmitted()},
		}},
		{"decoy-only server admission", BrokerReport{
			Principal: live.Principal,
			Server: RealmAdmission{
				Admitted:      terminalbackend.Admitted{Capabilities: []string{fixtureDecoyCapacity}},
				RawGeneration: fixtureGeneration,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := runtimeRoot(t)
			runtimeDir(t, root)
			req := validRequest(t, root)
			req.Caller = CallerBackground
			req.Platform = scalar.PlatformMacOS
			spy := &spawnSpy{}
			probe := &probeServerSpy{}
			// Even dependencies that would succeed must never run:
			// the miss path has no fallback to direct creation,
			// neither through the spawner nor through a re-entered
			// foreground probe.
			req.Deps.Spawn = spy.spawn
			req.Deps.ProbeServer = probe.probe
			req.Deps.ProbeBroker = func() (BrokerReport, error) {
				return tc.report, nil
			}
			_, err := Acquire(req)
			requireCapabilityUnavailable(t, err, "background", fixtureBroker, fixtureGeneration, fixtureRemedy)
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0 without fallback", spy.calls)
			}
			probe.requireSilent(t)
		})
	}
}

// The catalog-derived decoy set refuses through the background
// entry too: a decoy-only server admission on a broker miss returns
// the typed capability_unavailable with no spawn, for every non-realm
// Section 4.D member alone and all of them together.
func TestAcquireBackgroundRefusesEveryCatalogDecoy(t *testing.T) {
	decoys := catalogDecoyCapabilities(t)
	drive := func(t *testing.T, capabilities []string) {
		t.Helper()
		live := brokerReport()
		root := runtimeRoot(t)
		runtimeDir(t, root)
		req := validRequest(t, root)
		req.Caller = CallerBackground
		req.Platform = scalar.PlatformMacOS
		spy := &spawnSpy{}
		probe := &probeServerSpy{}
		req.Deps.Spawn = spy.spawn
		req.Deps.ProbeServer = probe.probe
		req.Deps.ProbeBroker = func() (BrokerReport, error) {
			return BrokerReport{
				Principal: live.Principal,
				Server: RealmAdmission{
					Admitted:      terminalbackend.Admitted{Capabilities: capabilities},
					RawGeneration: fixtureGeneration,
				},
			}, nil
		}
		_, err := Acquire(req)
		requireCapabilityUnavailable(t, err, "background", fixtureBroker, fixtureGeneration, fixtureRemedy)
		if spy.calls != 0 {
			t.Fatalf("spawn calls = %d, want 0", spy.calls)
		}
		probe.requireSilent(t)
	}
	for _, decoy := range decoys {
		t.Run(decoy, func(t *testing.T) {
			drive(t, []string{decoy})
		})
	}
	t.Run("all decoys together", func(t *testing.T) {
		drive(t, append([]string(nil), decoys...))
	})
}

// The broker miss refuses without fallback on every unix platform,
// not just macOS: a background restore worker on Linux or WSL2 meets
// the miss exactly like the macOS caller, and the no-fallback rule
// binds it the same way. One member per platform, so a
// platform-guarded fallback plant reddens on its own member.
func TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback(t *testing.T) {
	for _, platform := range []scalar.Platform{scalar.PlatformLinux, scalar.PlatformWSL2} {
		t.Run(string(platform), func(t *testing.T) {
			root := runtimeRoot(t)
			runtimeDir(t, root)
			req := validRequest(t, root)
			req.Caller = CallerBackground
			req.Platform = platform
			spy := &spawnSpy{}
			probe := &probeServerSpy{}
			req.Deps.Spawn = spy.spawn
			req.Deps.ProbeServer = probe.probe
			req.Deps.ProbeBroker = func() (BrokerReport, error) {
				return BrokerReport{}, nil
			}
			_, err := Acquire(req)
			requireCapabilityUnavailable(t, err, "background", fixtureBroker, fixtureGeneration, fixtureRemedy)
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0", spy.calls)
			}
			probe.requireSilent(t)
		})
	}
}

// Background acquisition verifies the runtime directory instead of
// creating it. A missing leaf is the ordinary post-reboot cold
// state — the Runtime IPC root is a per-user temporary directory —
// so absence returns the typed capability_unavailable with the
// request's realm/readiness details, and the refusal probes no
// broker, spawns nothing, and leaves no directory behind: a
// background caller performs no durable write.
func TestAcquireBackgroundVerifiesWithoutCreating(t *testing.T) {
	// The cold state is the same on every unix platform: $XDG_RUNTIME_DIR
	// is tmpfs on Linux/WSL2, so a background restore worker there meets
	// the absent leaf exactly like the macOS caller. Each platform maps
	// the absence to the typed unavailable — never the custody code.
	for _, platform := range []scalar.Platform{scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2} {
		t.Run(string(platform), func(t *testing.T) {
			root := runtimeRoot(t)
			req := validRequest(t, root)
			req.Caller = CallerBackground
			req.Platform = platform
			probed := false
			spy := &spawnSpy{}
			serverProbe := &probeServerSpy{}
			req.Deps.Spawn = spy.spawn
			req.Deps.ProbeServer = serverProbe.probe
			req.Deps.ProbeBroker = func() (BrokerReport, error) {
				probed = true
				return brokerReport(), nil
			}
			_, err := Acquire(req)
			requireCapabilityUnavailable(t, err, "background", fixtureBroker, fixtureGeneration, fixtureRemedy)
			if probed {
				t.Fatal("broker was probed for an absent runtime directory")
			}
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0 on absence", spy.calls)
			}
			serverProbe.requireSilent(t)
			if _, statErr := os.Lstat(filepath.Join(root, RuntimeDirName)); !os.IsNotExist(statErr) {
				t.Fatalf("background refusal created the directory: stat err = %v", statErr)
			}
		})
	}
}

// A missing runtime root is not a missing leaf: the root input names
// nothing the leaf could live under, so the background path refuses
// it as malformed input — the same code the commit side reports —
// without probing the broker.
func TestAcquireBackgroundMissingRootRefusesInvalid(t *testing.T) {
	root := filepath.Join(runtimeRoot(t), "no-such-root")
	req := validRequest(t, root)
	req.Caller = CallerBackground
	probed := false
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_invalid_arguments", "runtime root")
	if probed {
		t.Fatal("broker was probed past a missing root")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a missing root", spy.calls)
	}
	probe.requireSilent(t)
}

// The acquisition entry derives the spec's socket path on both caller
// paths: the outcome socket equals <Root>/tmux/ax.sock with both leaves
// asserted as spec literals — not via the production constants — so a
// renamed constant or a caller-selectable leaf reddens here.
func TestAcquireOutcomeSocketEqualsSpecPath(t *testing.T) {
	t.Run("foreground", func(t *testing.T) {
		root := runtimeRoot(t)
		req := validRequest(t, root)
		spy := &spawnSpy{}
		req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
			return ServerReport{}, nil
		}
		req.Deps.Spawn = spy.spawn
		outcome, err := Acquire(req)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(root, "tmux", "ax.sock"); outcome.Socket != want {
			t.Fatalf("socket = %q, want %q", outcome.Socket, want)
		}
	})
	t.Run("background", func(t *testing.T) {
		root := runtimeRoot(t)
		runtimeDir(t, root)
		req := validRequest(t, root)
		req.Caller = CallerBackground
		spy := &spawnSpy{}
		probe := &probeServerSpy{}
		req.Deps.Spawn = spy.spawn
		req.Deps.ProbeServer = probe.probe
		req.Deps.ProbeBroker = func() (BrokerReport, error) {
			return brokerReport(), nil
		}
		outcome, err := Acquire(req)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(root, "tmux", "ax.sock"); outcome.Socket != want {
			t.Fatalf("socket = %q, want %q", outcome.Socket, want)
		}
		if spy.calls != 0 {
			t.Fatalf("spawn calls = %d, want 0 on broker contact", spy.calls)
		}
		probe.requireSilent(t)
	})
}

// A commit-phase custody refusal staged through the foreground entry:
// the suite widens the mode behind the entry's back, and acquisition
// refuses without probing or spawning.
func TestAcquireForegroundStagesCommitPhaseCustodyRefusal(t *testing.T) {
	root := runtimeRoot(t)
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	req := validRequest(t, root)
	spy := &spawnSpy{}
	probed := false
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		probed = true
		return ServerReport{}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err = Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
	if probed {
		t.Fatal("server was probed past a custody refusal")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a custody refusal", spy.calls)
	}
}

// The background verify wiring refuses a widened directory through the
// entry with no broker contact.
func TestAcquireBackgroundStagesVerifyCustodyRefusal(t *testing.T) {
	root := runtimeRoot(t)
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	req := validRequest(t, root)
	req.Caller = CallerBackground
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	probed := false
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err = Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
	if probed {
		t.Fatal("broker was probed past a custody refusal")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a custody refusal", spy.calls)
	}
	probe.requireSilent(t)
}

// A present-but-invalid typed detail refuses as malformed input: the
// generation passes the entry's non-empty check, the broker is probed
// for the miss, and only the refusal renderer — which enforces the
// landed detail grammar — refuses, after the probe.
func TestAcquireBackgroundInvalidGenerationRefusesAfterProbe(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	req := validRequest(t, root)
	req.Caller = CallerBackground
	req.Generation = "gen\xff"
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	probed := false
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return BrokerReport{}, nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_invalid_arguments", "realm details")
	if !probed {
		t.Fatal("broker was not probed before the detail refusal")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0", spy.calls)
	}
	probe.requireSilent(t)
}

func TestAcquireBackgroundProbeFailurePassesThrough(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	req := validRequest(t, root)
	req.Caller = CallerBackground
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		return BrokerReport{}, errProbeBoom
	}
	_, err := Acquire(req)
	if !errors.Is(err, errProbeBoom) {
		t.Fatalf("err = %v, want the probe failure, not unavailable", err)
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0", spy.calls)
	}
	probe.requireSilent(t)
}

func TestAcquireRefusesEveryAmbientOverrideVector(t *testing.T) {
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
			for _, caller := range []Caller{CallerForeground, CallerBackground} {
				t.Run(string(caller), func(t *testing.T) {
					root := runtimeRoot(t)
					req := validRequest(t, root)
					req.Caller = caller
					spy := &spawnSpy{}
					probe := &probeServerSpy{}
					req.SocketOverride = override
					req.Deps.ProbeServer = probe.probe
					req.Deps.Spawn = spy.spawn
					_, err := Acquire(req)
					requireLocalError(t, err, "tmux_ambient_server_reuse", "socket override")
					if spy.calls != 0 {
						t.Fatalf("spawn calls = %d, want 0 on ambient refusal", spy.calls)
					}
					probe.requireSilent(t)
					if _, statErr := os.Lstat(filepath.Join(root, RuntimeDirName)); !os.IsNotExist(statErr) {
						t.Fatalf("ambient refusal created the directory: stat err = %v", statErr)
					}
				})
			}
		})
	}
}

func TestAcquireRefusesAmbientCollision(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ambient func(derived string) Ambient
	}{
		{"tmux env", func(derived string) Ambient { return Ambient{TMUXEnv: derived} }},
		{"tmux tmpdir", func(derived string) Ambient { return Ambient{TMUXTmpDir: derived} }},
		{"inherited socket", func(derived string) Ambient { return Ambient{InheritedSocket: derived} }},
		{"default path", func(derived string) Ambient { return Ambient{DefaultPath: derived} }},
		{"conventional name", func(derived string) Ambient { return Ambient{ConventionalName: derived} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, caller := range []Caller{CallerForeground, CallerBackground} {
				t.Run(string(caller), func(t *testing.T) {
					root := runtimeRoot(t)
					dir := socketDir(t, root)
					req := validRequest(t, root)
					req.Caller = caller
					req.Ambient = tc.ambient(SocketPath(dir))
					spy := &spawnSpy{}
					probe := &probeServerSpy{}
					req.Deps.ProbeServer = probe.probe
					req.Deps.Spawn = spy.spawn
					_, err := Acquire(req)
					requireLocalError(t, err, "tmux_ambient_server_reuse", "ambient collision")
					if spy.calls != 0 {
						t.Fatalf("spawn calls = %d, want 0", spy.calls)
					}
					probe.requireSilent(t)
				})
			}
		})
	}
}

// The background path verifies custody through VerifyRuntimeDir, so a
// non-directory leaf refuses through the entry before any broker
// contact: the verify-side O_DIRECTORY refusal driven at the entry the
// background path uses.
func TestAcquireBackgroundRefusesNonDirectoryLeaf(t *testing.T) {
	root := runtimeRoot(t)
	leaf := filepath.Join(root, RuntimeDirName)
	if err := os.WriteFile(leaf, []byte("x"), 0o700); err != nil {
		t.Fatal(err)
	}
	req := validRequest(t, root)
	req.Caller = CallerBackground
	probed := false
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	if probed {
		t.Fatal("broker was probed past a non-directory leaf")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a non-directory leaf", spy.calls)
	}
	probe.requireSilent(t)
}

// The foreground path ensures custody through EnsureRuntimeDir, so a
// non-directory leaf refuses through the entry before any server
// contact: the commit-side O_DIRECTORY refusal driven at the entry
// the foreground path uses. The regular file carries 0700 so a
// weakened commit open admits it outright — probing the server and
// spawning into <root>/tmux/ax.sock where tmux is a regular file —
// instead of rerouting to the mode gate.
func TestAcquireForegroundRefusesNonDirectoryLeaf(t *testing.T) {
	root := runtimeRoot(t)
	leaf := filepath.Join(root, RuntimeDirName)
	if err := os.WriteFile(leaf, []byte("x"), 0o700); err != nil {
		t.Fatal(err)
	}
	req := validRequest(t, root)
	spy := &spawnSpy{}
	probed := false
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		probed = true
		return ServerReport{}, nil
	}
	req.Deps.Spawn = spy.spawn
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	if probed {
		t.Fatal("server was probed past a non-directory leaf")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a non-directory leaf", spy.calls)
	}
}

// A symlink leaf refuses through the background entry before any
// broker contact. The escape target is a compliant directory, so a
// verify-side plant that follows the leaf admits fully and the test
// reddens on the broker contact, not on a rerouted custody detail.
func TestAcquireBackgroundRefusesSymlinkLeaf(t *testing.T) {
	root := runtimeRoot(t)
	escape := t.TempDir()
	leaf := filepath.Join(root, RuntimeDirName)
	if err := os.Symlink(escape, leaf); err != nil {
		t.Fatal(err)
	}
	req := validRequest(t, root)
	req.Caller = CallerBackground
	probed := false
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	if probed {
		t.Fatal("broker was probed past a symlink leaf")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a symlink leaf", spy.calls)
	}
	probe.requireSilent(t)
}

// A symlinked runtime root refuses through the background entry
// before any broker contact: the pinned root handle never follows
// the last component, exactly as on the commit and verify sides.
func TestAcquireBackgroundRefusesSymlinkRoot(t *testing.T) {
	real := runtimeRoot(t)
	link := filepath.Join(t.TempDir(), "linkroot")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	req := validRequest(t, link)
	req.Caller = CallerBackground
	probed := false
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	if probed {
		t.Fatal("broker was probed past a symlinked root")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a symlinked root", spy.calls)
	}
	probe.requireSilent(t)
}

// A handle/path mismatch refuses through the background entry before
// any broker contact: the verify-side binding check driven at the
// entry the background path uses, staged through the sameDirectory
// seam.
func TestAcquireBackgroundRefusesHandleMismatch(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	previous := sameDirectory
	sameDirectory = func(_, _ os.FileInfo) bool { return false }
	defer func() { sameDirectory = previous }()
	req := validRequest(t, root)
	req.Caller = CallerBackground
	probed := false
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	if probed {
		t.Fatal("broker was probed past a handle mismatch")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 past a handle mismatch", spy.calls)
	}
	probe.requireSilent(t)
}

// An unclean spelling of the derived socket refuses through the entry
// with no spawn: the cleaned-spelling collision gate driven at
// Acquire.
func TestAcquireRefusesAmbientCollisionUncleanSpelling(t *testing.T) {
	root := runtimeRoot(t)
	dir := socketDir(t, root)
	alias := dir + "/./" + SocketName
	if got := filepath.Clean(alias); got != SocketPath(dir) {
		t.Fatalf("alias %q cleans to %q, want the derived socket", alias, got)
	}
	for _, caller := range []Caller{CallerForeground, CallerBackground} {
		t.Run(string(caller), func(t *testing.T) {
			req := validRequest(t, root)
			req.Caller = caller
			req.Ambient = Ambient{DefaultPath: alias}
			spy := &spawnSpy{}
			probe := &probeServerSpy{}
			req.Deps.ProbeServer = probe.probe
			req.Deps.Spawn = spy.spawn
			_, err := Acquire(req)
			requireLocalError(t, err, "tmux_ambient_server_reuse", "ambient collision")
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0", spy.calls)
			}
			probe.requireSilent(t)
		})
	}
}

// A TMUX value naming the derived socket in the real
// `<socket>,<pid>,<index>` encoding refuses through the entry with
// no spawn: the collision gate compares the socket component.
func TestAcquireRefusesTMUXEnvCollisionRealEncoding(t *testing.T) {
	root := runtimeRoot(t)
	dir := socketDir(t, root)
	encoded := SocketPath(dir) + ",12345,0"
	for _, caller := range []Caller{CallerForeground, CallerBackground} {
		t.Run(string(caller), func(t *testing.T) {
			req := validRequest(t, root)
			req.Caller = caller
			req.Ambient = Ambient{TMUXEnv: encoded}
			spy := &spawnSpy{}
			probe := &probeServerSpy{}
			req.Deps.ProbeServer = probe.probe
			req.Deps.Spawn = spy.spawn
			_, err := Acquire(req)
			requireLocalError(t, err, "tmux_ambient_server_reuse", "ambient collision")
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0", spy.calls)
			}
			probe.requireSilent(t)
		})
	}
}

// The caller value is the gate input, so no caller dimension exists:
// an unknown caller is neither foreground nor background, and an
// admitted one always dispatches down the foreground path (its value
// differs from the background caller). The test arms the foreground
// post-admission spy set the admitted input would reach.
func TestAcquireRefusesUnknownCaller(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	req.Caller = "operator"
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_invalid_arguments", "caller")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on caller refusal", spy.calls)
	}
	probe.requireSilent(t)
}

func TestAcquireRefusesUnknownPlatform(t *testing.T) {
	root := runtimeRoot(t)
	for _, caller := range []Caller{CallerForeground, CallerBackground} {
		t.Run(string(caller), func(t *testing.T) {
			req := validRequest(t, root)
			req.Caller = caller
			req.Platform = scalar.Platform("plan9")
			spy := &spawnSpy{}
			probe := &probeServerSpy{}
			req.Deps.Spawn = spy.spawn
			req.Deps.ProbeServer = probe.probe
			_, err := Acquire(req)
			requireLocalError(t, err, "tmux_invalid_arguments", "runtime platform")
			if spy.calls != 0 {
				t.Fatalf("spawn calls = %d, want 0 on platform refusal", spy.calls)
			}
			probe.requireSilent(t)
		})
	}
}

func TestAcquireRefusesWindowsPlatform(t *testing.T) {
	root := runtimeRoot(t)
	for _, caller := range []Caller{CallerForeground, CallerBackground} {
		req := validRequest(t, root)
		req.Caller = caller
		req.Platform = scalar.PlatformWindows
		req.Root = `C:\ax-runtime`
		spy := &spawnSpy{}
		probe := &probeServerSpy{}
		req.Deps.Spawn = spy.spawn
		req.Deps.ProbeServer = probe.probe
		_, err := Acquire(req)
		requireLocalError(t, err, "tmux_invalid_arguments", "runtime platform")
		if spy.calls != 0 {
			t.Fatalf("spawn calls = %d, want 0 on platform refusal", spy.calls)
		}
		probe.requireSilent(t)
	}
}

func TestAcquireRefusesMissingRealmDetails(t *testing.T) {
	root := runtimeRoot(t)
	for _, tc := range []struct {
		name   string
		mutate func(*Request)
	}{
		{"generation", func(r *Request) { r.Generation = "" }},
		{"broker state", func(r *Request) { r.BrokerState = "" }},
		{"remediation", func(r *Request) { r.Remediation = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, caller := range []Caller{CallerForeground, CallerBackground} {
				t.Run(string(caller), func(t *testing.T) {
					req := validRequest(t, root)
					req.Caller = caller
					tc.mutate(&req)
					spy := &spawnSpy{}
					probe := &probeServerSpy{}
					req.Deps.Spawn = spy.spawn
					req.Deps.ProbeServer = probe.probe
					_, err := Acquire(req)
					requireLocalError(t, err, "tmux_invalid_arguments", "realm details")
					if spy.calls != 0 {
						t.Fatalf("spawn calls = %d, want 0 on details refusal", spy.calls)
					}
					probe.requireSilent(t)
				})
			}
		})
	}
}

func TestAcquireRefusesMissingDependencies(t *testing.T) {
	root := runtimeRoot(t)
	foreground := validRequest(t, root)
	_, err := Acquire(foreground)
	requireLocalError(t, err, "tmux_invalid_arguments", "dependencies")
	background := validRequest(t, root)
	background.Caller = CallerBackground
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	background.Deps.Spawn = spy.spawn
	background.Deps.ProbeServer = probe.probe
	_, err = Acquire(background)
	requireLocalError(t, err, "tmux_invalid_arguments", "dependencies")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0 on missing dependencies", spy.calls)
	}
	probe.requireSilent(t)
}

func TestAcquireCreatesRuntimeDirIdempotently(t *testing.T) {
	root := runtimeRoot(t)
	make := func() Request {
		req := validRequest(t, root)
		spy := &spawnSpy{}
		req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
			return ServerReport{}, nil
		}
		req.Deps.Spawn = spy.spawn
		return req
	}
	first, err := Acquire(make())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Acquire(make())
	if err != nil {
		t.Fatal(err)
	}
	if first.Socket != second.Socket {
		t.Fatalf("sockets differ: %q vs %q", first.Socket, second.Socket)
	}
}

func TestAcquireBackgroundNilBrokerWithSpawnRefuses(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	req.Caller = CallerBackground
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_invalid_arguments", "dependencies")
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0", spy.calls)
	}
	probe.requireSilent(t)
}

func TestAcquireForegroundNilSpawnWithProbeRefuses(t *testing.T) {
	root := runtimeRoot(t)
	req := validRequest(t, root)
	req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
		return ServerReport{}, nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_invalid_arguments", "dependencies")
}

func TestCheckCreationAllowedRefusesBackground(t *testing.T) {
	requireLocalError(t, checkCreationAllowed(CallerBackground), "tmux_background_creation_refused", "background creation")
	if err := checkCreationAllowed(CallerForeground); err != nil {
		t.Fatalf("foreground creation refused: %v", err)
	}
}
