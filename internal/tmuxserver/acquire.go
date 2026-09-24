package tmuxserver

import (
	"errors"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Caller names the acquisition caller. The vocabulary is owned by
// internal/axpane, which already gates the wrapper decision over it;
// this package reuses its constants rather than retyping them.
type Caller = string

const (
	// CallerForeground is an interactive operator-side caller that
	// may create the dedicated server.
	CallerForeground = axpane.CallerForeground
	// CallerBackground is a background CLI, SSH RPC process,
	// daemon, or restore worker: it must never create a
	// credential-dependent tmux server, may contact an
	// already-running authenticated broker, and otherwise receives
	// capability_unavailable with no fallback to direct creation.
	CallerBackground = axpane.CallerBackground
)

// Acquisition outcomes name how the caller reached the dedicated
// server. Exactly one applies; there is no combined or fallback
// outcome.
const (
	// ViaSpawned created the dedicated -S server on the foreground
	// path and returns its spawn argv.
	ViaSpawned = "spawned"
	// ViaAttachedFound the dedicated server already running and
	// attested on the foreground path.
	ViaAttached = "attached-running"
	// ViaBroker contacted the already-running authenticated Aqua
	// broker and its attested AX tmux server on the background
	// path. No local spawn argv exists on this path.
	ViaBroker = "broker"
)

// realmCapability is the Section 4.D capability the background
// typed refusal names: the exact instance passes the AX sentinel
// and the provider-auth smoke without UI. The admission verdict is
// the landed terminalbackend owner's; this package consumes it
// through RealmAdmission and never re-decides it.
const realmCapability = "credential_capable_execution_realm"

// ServerReport is the live dedicated-server probe result: whether
// the private socket answers, with the landed admission verdict over
// the exact server when it does.
type ServerReport struct {
	Running   bool
	Admission RealmAdmission
}

// Dependencies carries every external interaction acquisition
// performs. This leaf ships the decision core and argv only: no
// production probe or spawner implementation exists in the tree. The
// exec/probe adapters — which report Reconcile/ResolveEvidence
// verdicts and spawn through the built argv — and the server socket's
// crash evidence are owned by the lifecycle leaf TASK-260830-1c28dz;
// tests wire fakes. No probe runs without its dependency: a nil entry
// refuses instead of reaching for a default.
type Dependencies struct {
	// ProbeServer reports the dedicated socket's live state.
	ProbeServer func(socket string) (ServerReport, error)
	// ProbeBroker reports the live Aqua broker state.
	ProbeBroker func() (BrokerReport, error)
	// Spawn creates the dedicated server with the given -S argv.
	Spawn func(socket string, argv []string) error
}

// Request is the complete acquisition surface. There is deliberately
// no member for a credential flag, a managername hint, or a cached
// observation: the background creation rule needs no credential flag
// because the background path never creates on any platform, and a
// hint or cached observation never authorizes, so none is an input.
type Request struct {
	Caller   Caller
	Platform scalar.Platform
	// Root is the Runtime IPC root of the SPEC 3.2 table (macOS:
	// per-user temporary directory; Linux/WSL2:
	// $XDG_RUNTIME_DIR/ax), resolved by the landed localstore
	// layout as PathRuntime and supplied by the caller — this leaf
	// resolves no root of its own, because the socket is runtime
	// IPC, never durable identity. The runtime leaf is not a
	// caller input: Acquire always passes RuntimeDirName, so the
	// derived socket is always <Root>/tmux/ax.sock and no caller
	// spelling can re-open the ambient or conventional names the
	// override gate refuses. Foreground acquisition ensures the
	// directory idempotently; background acquisition verifies the
	// existing directory and never creates it.
	Root string
	// Ambient carries the observed ambient socket facts for
	// evidence. Resolution never reads them; tests assert the
	// acquired socket equals none of them.
	Ambient Ambient
	// SocketOverride must be empty. Any caller-supplied socket —
	// whichever ambient vector supplied it — refuses.
	SocketOverride string
	// SessionID is the logical session the spawned server will
	// host, as canonical UUIDv7. It shapes spawn argv only.
	SessionID string
	// Generation, BrokerState, and Remediation are the typed
	// realm/readiness details the capability_unavailable refusal
	// carries. All three are required: an input that omits one
	// is malformed and refuses invalid_arguments instead of
	// emitting an untyped refusal. Generation is also the typed
	// generation every admission must bind to.
	Generation  string
	BrokerState string
	Remediation string
	Deps        Dependencies
}

// Outcome is the one acquisition result.
type Outcome struct {
	Via    string
	Socket string
	// Argv is the dedicated -S spawn argv. It is set only for
	// ViaSpawned: the broker path spawns nothing and the
	// attached path presents through sibling-owned attach
	// semantics, not through a second spawn vector.
	Argv []string
}

// Acquire resolves the dedicated server socket from the runtime
// directory and reaches the server through the caller-appropriate
// path. Foreground attaches to the running attested server or
// spawns the dedicated -S server; background contacts the
// already-running authenticated broker or receives the typed
// capability_unavailable refusal, and never falls back to direct
// server creation — there is no spawn call on the background path,
// and every background-path Acquire test arms both a spy spawner
// and a spy server probe that fail the run if either is ever
// invoked, so a fallback through either foreground dependency is
// visible.
//
// Refusals precede side effects: malformed input (caller, realm
// details, runtime root/name grammar, session identity), ambient
// reuse, and missing dependencies all refuse before any directory is
// created, and background acquisition verifies the runtime directory
// without creating it, so a background caller performs no durable
// write.
func Acquire(req Request) (Outcome, error) {
	if req.Caller != CallerForeground && req.Caller != CallerBackground {
		return Outcome{}, &Error{Code: CodeInvalidArguments, Detail: "caller"}
	}
	if req.Generation == "" || req.BrokerState == "" || req.Remediation == "" {
		return Outcome{}, &Error{Code: CodeInvalidArguments, Detail: "realm details"}
	}
	dir, err := lexicalRuntimePath(req.Root, RuntimeDirName, req.Platform)
	if err != nil {
		return Outcome{}, err
	}
	socket, err := ResolveSocket(dir, req.SocketOverride, req.Ambient)
	if err != nil {
		return Outcome{}, err
	}
	if req.Caller == CallerBackground {
		return acquireBackground(req, socket)
	}
	return acquireForeground(req, socket)
}

// acquireBackground implements the broker-or-refuse rule: it may
// contact the already-running authenticated broker and its attested
// server, and when neither exists it returns capability_unavailable
// with typed realm/readiness details. Direct server creation is not
// the else branch — this function has no path to the spawner, on
// any platform — and the runtime directory is verified, never
// created here. A missing leaf is the ordinary cold state (the
// Runtime IPC root is a per-user temporary directory), so absence
// maps to the typed unavailable with no broker probe and no spawn;
// a leaf that exists but fails custody keeps its custody code.
func acquireBackground(req Request, socket string) (Outcome, error) {
	if req.Deps.ProbeBroker == nil {
		return Outcome{}, &Error{Code: CodeInvalidArguments, Detail: "dependencies"}
	}
	verr := VerifyRuntimeDir(req.Root, RuntimeDirName, req.Platform)
	if runtimeAbsent(verr) {
		return Outcome{}, backgroundUnavailable(req, verr)
	}
	if verr != nil {
		return Outcome{}, verr
	}
	status, err := req.Deps.ProbeBroker()
	if err != nil {
		return Outcome{}, err
	}
	if err := CheckBrokerContact(status, req.Generation, currentUID()); err != nil {
		// The miss path creates nothing: it consults the creation
		// gate so the no-fallback rule is enforced by a call, not
		// by the absence of one. The gate refuses every background
		// caller; its refusal is expected here and the path reports
		// the typed unavailable the spec names. The else branch is
		// reachable only when the gate is weakened to admit a
		// background caller, and then it refuses with the creation
		// code instead of the typed unavailable, so the committed
		// background-miss tests redden.
		if cerr := checkCreationAllowed(req.Caller); cerr != nil {
			return Outcome{}, backgroundUnavailable(req, err)
		}
		return Outcome{}, &Error{Code: CodeBackgroundCreationRefused, Detail: "background fallback"}
	}
	return Outcome{Via: ViaBroker, Socket: socket}, nil
}

// acquireForeground attaches to the running attested dedicated
// server or spawns it. A running but unattested server refuses:
// acquisition never attaches to a server whose landed admission is
// missing or bound to another generation, and never repairs it here
// — lifecycle owns that decision. The session identity is validated
// here, before the directory is ensured and the server probed, so a
// malformed session refuses before any side effect; BuildArgv
// re-validates at the spawn site, and each side carries its own
// narrowing row.
func acquireForeground(req Request, socket string) (Outcome, error) {
	if req.Deps.ProbeServer == nil || req.Deps.Spawn == nil {
		return Outcome{}, &Error{Code: CodeInvalidArguments, Detail: "dependencies"}
	}
	if _, err := scalar.ParseUUIDv7(req.SessionID); err != nil {
		return Outcome{}, &Error{Code: CodeInvalidArguments, Detail: "session identity"}
	}
	ensured, err := EnsureRuntimeDir(req.Root, RuntimeDirName, req.Platform, nil)
	if err != nil {
		return Outcome{}, err
	}
	report, err := req.Deps.ProbeServer(socket)
	if err != nil {
		return Outcome{}, err
	}
	if report.Running {
		if err := CheckServerAttested(report.Admission, req.Generation); err != nil {
			return Outcome{}, err
		}
		return Outcome{Via: ViaAttached, Socket: socket}, nil
	}
	argv, err := BuildArgv(ensured, socket, req.SessionID)
	if err != nil {
		return Outcome{}, err
	}
	if err := req.Deps.Spawn(socket, argv); err != nil {
		return Outcome{}, &Error{Code: CodeSpawnFailed, Detail: "server spawn"}
	}
	return Outcome{Via: ViaSpawned, Socket: socket, Argv: argv}, nil
}

// runtimeAbsent reports whether err is the verify-side absent-leaf
// refusal. Only the missing leaf counts — never a missing root, a
// mode, ownership, kind, or symlink refusal: absence is the
// readiness fact the background path maps to
// capability_unavailable, while every unsafe-leaf refusal keeps its
// custody code. A nil error is not absence.
func runtimeAbsent(err error) bool {
	var local *Error
	if !errors.As(err, &local) {
		return false
	}
	return local.Code == CodeUnsafeRuntimeDir && local.Detail == "runtime absent"
}

// checkCreationAllowed is the single choke point before every
// direct server creation. A background caller never creates: on
// macOS because server creation is credential-sensitive, and on
// every platform because the background path must not fall back to
// direct creation. The macOS credential-dependent case is one
// fixture of this gate, not a narrower arm: narrowing the gate to
// macOS or to credential-dependent servers would admit background
// creation the no-fallback rule forbids. Foreground creation is
// allowed on every platform; the credential binding it needs is
// proven by the landed admission the server must present before
// any attach.
func checkCreationAllowed(caller Caller) error {
	if caller == CallerBackground {
		return &Error{Code: CodeBackgroundCreationRefused, Detail: "background creation"}
	}
	return nil
}

// backgroundUnavailable composes the typed capability_unavailable
// refusal the spec names for the background miss, with the exact
// Section 15.3 detail set. The constructor enforces the typed
// details; a failure to render them is itself malformed input, not
// an untyped refusal.
func backgroundUnavailable(req Request, cause error) error {
	details := axerror.RealmEvidence{
		Capability:           realmCapability,
		CallerRealm:          req.Caller,
		BrokerState:          req.BrokerState,
		TmuxServerGeneration: req.Generation,
		Remediation:          req.Remediation,
	}
	failure, err := axerror.NewRealmEvidenceUnavailable(axerror.Version130,
		"no authenticated broker or attested tmux server",
		axerror.NoIDs(), details, cause)
	if err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "realm details"}
	}
	return failure
}
