package tmuxserver

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// DialOutcome is one unix-socket dial verdict: the socket answers, the
// socket is stale (connection refused: no listener), or the dial
// proves nothing (any other failure). Liveness is decided by connect,
// never by parsing daemon stderr: no gate here inspects source text.
type DialOutcome string

// Dial outcomes.
const (
	DialLive    DialOutcome = "live"
	DialStale   DialOutcome = "stale"
	DialUnknown DialOutcome = "unknown"
)

// Dialer dials the dedicated socket and reports the outcome. The
// production dialer connects over unix with a short timeout; tests
// inject fakes. A live outcome never proves attestation — only that a
// listener answers — and a stale outcome never proves absence of all
// servers, only that this socket has no listener.
type Dialer interface {
	Dial(socket string) DialOutcome
}

// UnixDialer is the production Dialer.
type UnixDialer struct {
	// Timeout bounds one dial. Zero means one second.
	Timeout time.Duration
}

// Dial connects to socket over unix: success is live; ECONNREFUSED or
// ENOENT is stale; every other failure (timeout, permission,
// impossible length) is unknown. ENOENT is stale rather than unknown
// because the caller length-gates and custody-checks first: under a
// verified parent, a missing socket proves no bind, not a broken
// tree. This function dials only.
func (dialer UnixDialer) Dial(socket string) DialOutcome {
	timeout := dialer.Timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	conn, err := net.DialTimeout("unix", socket, timeout)
	if err == nil {
		_ = conn.Close()
		return DialLive
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && (errno == syscall.ECONNREFUSED || errno == syscall.ENOENT) {
		return DialStale
	}
	return DialUnknown
}

// AdmitServer is the server attestation verdict over the exact live
// socket: the landed Reconcile admission the probe binds to the
// request generation. Production wires terminalbackend.Reconcile over
// the caller-supplied Manifest, Probe, evidence, generation, clock,
// and signature verifier; tests inject verdicts directly.
type AdmitServer func() (RealmAdmission, error)

// ReconcileAdmission closes AdmitServer over the landed admission:
// manifest, probe, evidence, the AX-known raw generation, the
// admission instant, and the trusted-registry signature verifier. A
// nil verifier fails closed inside Reconcile, never here.
func ReconcileAdmission(manifest terminalbackend.Manifest, probe terminalbackend.Probe, evidence []terminalbackend.Evidence, rawGeneration string, now time.Time, verify terminalbackend.SignatureVerifier) AdmitServer {
	return func() (RealmAdmission, error) {
		admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, rawGeneration, now, verify)
		if err != nil {
			return RealmAdmission{}, err
		}
		return RealmAdmission{Admitted: admitted, RawGeneration: rawGeneration}, nil
	}
}

// ServerProber is the production ProbeServer dependency: custody,
// length, dial, then admission. Before connect it enforces the
// before-connect custody rejection and the sun_path length refusal; a
// live socket reports the admission verdict, a stale socket reports
// not-running, and an unknown dial passes through as unknown (never
// unavailable, never absent).
type ServerProber struct {
	// Root is the Runtime IPC root the socket must sit under.
	Root string
	// Platform selects the sun_path limit.
	Platform scalar.Platform
	// Dialer connects to the socket. Nil refuses: no probe runs
	// without its dependency.
	Dialer Dialer
	// Admit reports the landed admission verdict. Nil refuses.
	Admit AdmitServer
}

// Probe implements Dependencies.ProbeServer.
func (prober ServerProber) Probe(socket string) (ServerReport, error) {
	if prober.Dialer == nil || prober.Admit == nil {
		return ServerReport{}, &Error{Code: CodeInvalidArguments, Detail: "probe dependencies"}
	}
	if err := CheckSocketLength(socket, prober.Platform); err != nil {
		return ServerReport{}, err
	}
	if err := CheckSocketCustody(socket, prober.Root, prober.Platform); err != nil {
		return ServerReport{}, err
	}
	switch prober.Dialer.Dial(socket) {
	case DialStale:
		return ServerReport{Running: false}, nil
	case DialUnknown:
		return ServerReport{}, &Error{Code: CodeSpawnFailed, Detail: "server probe"}
	case DialLive:
		admission, err := prober.Admit()
		if err != nil {
			return ServerReport{}, err
		}
		return ServerReport{Running: true, Admission: admission}, nil
	default:
		return ServerReport{}, &Error{Code: CodeSpawnFailed, Detail: "server probe"}
	}
}

// BrokerProber is the production ProbeBroker dependency. No Aqua broker
// IPC exists in the tree, so the production probe reports the miss:
// the zero report (no authenticated principal, no attested server),
// which the broker-or-refuse rule maps to the typed
// capability_unavailable. Contacting a broker is MAY in the spec, so
// an honest miss is compliant; the broker path stays reachable through
// injected Dependencies (proven by the acquisition suite), and a
// future broker-IPC task owns the transport. This bound is stated, not
// silent: Probe documents it on every call site.
type BrokerProber struct{}

// Probe implements Dependencies.ProbeBroker by reporting no broker.
func (BrokerProber) Probe() (BrokerReport, error) {
	return BrokerReport{}, nil
}

// SpawnHooks arms the crash boundary of the spawn commit. AfterUnlink
// runs after a stale socket is unlinked and before the server is
// spawned — only when a stale file was actually removed — so a crash
// there leaves a retry that binds clean. Hooks is nil in production.
type SpawnHooks struct {
	AfterUnlink func(socket string)
}

// ServerSpawner is the production Dependencies.Spawn dependency: the
// bind step. Before bind it enforces the length refusal, the
// before-bind custody rejection, and the exec-site socket pin (the
// argv must address exactly this socket — the second side of the pin
// gate whose first side is BuildCommand/BuildArgv); then it unlinks a
// stale socket, if any, and spawns through the runner. A crash between
// unlink and spawn leaves no socket behind, so a retry binds clean.
type ServerSpawner struct {
	// Root is the Runtime IPC root the socket must sit under.
	Root string
	// Platform selects the sun_path limit.
	Platform scalar.Platform
	// Runner executes the spawn vector. Nil refuses.
	Runner Runner
	// Hooks arms the crash boundary. Nil in production.
	Hooks *SpawnHooks
}

// Spawn implements Dependencies.Spawn.
func (spawner ServerSpawner) Spawn(socket string, argv []string) error {
	if spawner.Runner == nil {
		return &Error{Code: CodeInvalidArguments, Detail: "spawn dependencies"}
	}
	if err := CheckSocketLength(socket, spawner.Platform); err != nil {
		return err
	}
	if err := CheckSocketCustody(socket, spawner.Root, spawner.Platform); err != nil {
		return err
	}
	if err := checkSpawnAddresses(argv, socket); err != nil {
		return err
	}
	removed, err := unlinkStaleSocket(socket)
	if err != nil {
		return err
	}
	if removed && spawner.Hooks != nil && spawner.Hooks.AfterUnlink != nil {
		spawner.Hooks.AfterUnlink(socket)
	}
	result, err := spawner.Runner.Run(context.Background(), argv)
	if err != nil {
		return &Error{Code: CodeSpawnFailed, Detail: "server spawn"}
	}
	if result.ExitCode != 0 {
		return &Error{Code: CodeSpawnFailed, Detail: "server spawn"}
	}
	return nil
}

// checkSpawnAddresses is the exec-site socket pin: the spawn vector
// must be a tmux -S vector addressing exactly this socket. A vector
// for any other socket — however it was built — refuses here instead
// of spawning a substituted server. The argv shape gate runs first so
// a malformed vector reports the shape refusal, not the pin.
func checkSpawnAddresses(argv []string, socket string) error {
	if err := secprim.CheckArgv(argv); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "spawn argv"}
	}
	if len(argv) < 3 || argv[0] != "tmux" || argv[1] != "-S" || argv[2] != socket {
		return &Error{Code: CodeAmbientReuse, Detail: "spawn socket"}
	}
	return nil
}

// unlinkStaleSocket removes a stale socket file before bind and reports
// whether it removed one. Custody already proved the path is absent or
// a socket; anything else refuses instead of unlinking. Absence is a
// no-op: bind proceeds.
func unlinkStaleSocket(socket string) (bool, error) {
	info, err := os.Lstat(socket)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &Error{Code: CodeSpawnFailed, Detail: "server spawn"}
	}
	if info.Mode().Type() != os.ModeSocket {
		return false, &Error{Code: CodeUnsafeSocketPath, Detail: "socket kind"}
	}
	if err := os.Remove(socket); err != nil {
		return false, &Error{Code: CodeSpawnFailed, Detail: "server spawn"}
	}
	return true, nil
}

// Production wires the production exec/probe adapters into the
// acquisition Dependencies: the server prober (custody, length, dial,
// landed admission), the broker miss, and the bind-step spawner. The
// caller supplies the runtime root, platform, runner, dialer, and
// admission verdict; tests inject fakes for the runner and dialer and
// drive every gate deterministically with no tmux process anywhere.
func Production(root string, platform scalar.Platform, runner Runner, dialer Dialer, admit AdmitServer) Dependencies {
	prober := ServerProber{Root: root, Platform: platform, Dialer: dialer, Admit: admit}
	spawner := ServerSpawner{Root: root, Platform: platform, Runner: runner}
	return Dependencies{
		ProbeServer: prober.Probe,
		ProbeBroker: BrokerProber{}.Probe,
		Spawn:       spawner.Spawn,
	}
}
