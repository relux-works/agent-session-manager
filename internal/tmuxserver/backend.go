package tmuxserver

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// operationDirectives is the exact lifecycle-to-tmux mapping: every
// operation executes a fixed directive sequence and no other tmux
// command. The table is the contract; the drivers below execute it.
// The boundary list-panes member is conditional: it runs only for
// provider proof kinds, binding contemporaneous provider rows into
// the boundary evidence, and never fails the observation.
var operationDirectives = map[terminalbackend.Operation][]Directive{
	terminalbackend.OperationCreate:           {DirectiveNewSession},
	terminalbackend.OperationAttach:           {DirectiveAttachSession},
	terminalbackend.OperationStatus:           {DirectiveListPanes, DirectiveListSessions},
	terminalbackend.OperationQuiesceInput:     {DirectiveLockSession, DirectiveDetachClients},
	terminalbackend.OperationWaitSafeBoundary: {DirectiveWaitBoundary, DirectiveListPanes},
	terminalbackend.OperationRequestStop:      {DirectiveSendInterrupt, DirectiveHasSession, DirectiveKillSession, DirectiveHasSession},
	terminalbackend.OperationTerminateStale:   {DirectiveKillSession, DirectiveHasSession},
	terminalbackend.OperationRestore:          {DirectiveNewSession},
}

// DirectivesFor reports the fixed tmux directive sequence for one
// lifecycle operation. Manifest and probe are not lifecycle operations
// and refuse; every other unknown operation refuses as well.
func DirectivesFor(operation terminalbackend.Operation) ([]Directive, error) {
	directives, known := operationDirectives[operation]
	if !known {
		return nil, &Error{Code: terminalbackend.CodeProtocolError, Detail: "lifecycle operation"}
	}
	return append([]Directive(nil), directives...), nil
}

// backend is the tmux terminstance.Backend: side-effect execution and
// status observation over the structured commands. It is
// request-scoped — one backend per Execute call — because effects
// carry no identity of their own and the runner, socket, stores, and
// admission are per-request facts.
type backend struct {
	runner      Runner
	runtimeDir  string
	socket      string
	operation   terminalbackend.Operation
	sessionID   string
	instanceID  string
	bootstrapID string
	bindingID   string
	bindingDoc  []byte
	channel     string
	quiescence  string
	proofKind   terminstance.ProviderProofKind
	waitCtx     context.Context
	deadline    time.Time
	// opDeadline is the operation deadline for the
	// post-escalation observation: the graceful wait budget in
	// deadline is already exhausted when the stop escalation
	// fires, so the re-confirmation draws on the still-live
	// operation budget instead of reusing the expired wait. A
	// zero value (paths that never escalate) falls back to the
	// poll bound; the Execute stop path always sets it.
	opDeadline time.Time
	// recheck revalidates the live authorization facts (operation
	// deadline, AX authorization over the current lease, server
	// generation) at a destructive command boundary that follows a
	// wait. The lifecycle entry builds it over the same landed
	// gates the engine's per-effect loop applies; paths whose
	// effects issue no destructive command after any wait leave it
	// nil. A nil recheck at a guarded boundary refuses instead of
	// running blind: the backend never issues a post-wait
	// destructive command without its revalidation.
	recheck   func() error
	now       func() time.Time
	sleep     func(time.Duration)
	poll      time.Duration
	admitted  terminalbackend.Admitted
	bindings  *axpane.Store
	attach    *termbind.AttachStore
	states    *InstanceStates
	transport string
	input     bool
	authRaw   []byte
	clientID  string
	// killExit records the stale-kill exit for the close re-confirm:
	// a live session after a clean kill and a live session after a
	// failed kill refuse through the same arm, and the recorded exit
	// is the member split the narrowing mutant admits exactly one of.
	// -1 means no kill ran yet on this backend.
	killExit int
}

// effectEvidence derives one evidence digest over the canonical effect
// record: effect name, operation, instance, and outcome facts. The join
// is deterministic; nothing parses it back.
func effectEvidence(effect terminalbackend.SideEffect, operation terminalbackend.Operation, instance string, facts ...string) string {
	record := strings.Join(append([]string{string(effect), string(operation), instance}, facts...), "\n")
	return scalar.SHA256Digest([]byte(record)).String()
}

// PerformEffect executes one side effect and reports its evidence ID.
// A nil error proves the commit; a coded error proves the backend
// refused before committing; any other error proves nothing. Every
// tmux command runs through BuildCommand, so no effect addresses any
// server but the derived socket.
func (runner *backend) PerformEffect(ctx context.Context, effect terminalbackend.SideEffect) (string, error) {
	switch effect {
	case terminalbackend.EffectBindingPersisted:
		return runner.persistBinding()
	case terminalbackend.EffectWrapperStarted:
		return runner.startWrapper(ctx, terminalbackend.EffectWrapperStarted)
	case terminalbackend.EffectAttachClientCreated:
		return runner.createAttachClient()
	case terminalbackend.EffectInputClosed:
		return runner.closeInput(ctx)
	case terminalbackend.EffectSafeBoundaryObserved:
		return runner.observeBoundary()
	case terminalbackend.EffectGracefulStopRequested:
		return runner.execEvidence(ctx, DirectiveSendInterrupt, terminalbackend.EffectGracefulStopRequested)
	case terminalbackend.EffectProcessClosed:
		return runner.confirmClosed(ctx)
	case terminalbackend.EffectBackendStoreClosed:
		return effectEvidence(effect, runner.operation, runner.instanceID, "tmux-side closes done"), nil
	case terminalbackend.EffectStaleIncarnationTerminated:
		return runner.terminateStale(ctx)
	case terminalbackend.EffectWrapperRestored:
		return runner.startWrapper(ctx, terminalbackend.EffectWrapperRestored)
	default:
		return "", &terminstance.Error{Code: terminalbackend.CodeProtocolError, Detail: "backend effect vocabulary"}
	}
}

// persistBinding records the bootstrap binding for the create/restore
// pair. A changed operation in the bootstrap window refuses the landed
// idempotency_mismatch; an identical retry replays the one receipt.
// On create the carried binding document is kept alongside the
// receipt, so restore can return the required binding without
// minting; a replayed receipt heals a missing document from the same
// carried bytes, closing the crash window between receipt and
// document.
func (runner *backend) persistBinding() (string, error) {
	if runner.bindings == nil {
		return "", &terminstance.Error{Code: terminalbackend.CodeProtocolError, Detail: "backend binding store"}
	}
	recorded, replayed, err := runner.bindings.Bind(runner.sessionID, runner.bootstrapID, axpane.Binding{
		SessionID:          runner.sessionID,
		OperationID:        runner.bootstrapID,
		TerminalInstanceID: runner.instanceID,
		BindingDigest:      runner.bindingID,
	})
	if err != nil {
		if code := backendErrorCode(err); code != "" {
			return "", &terminstance.Error{Code: code, Detail: "backend binding conflict"}
		}
		return "", err
	}
	if len(runner.bindingDoc) > 0 && runner.states != nil {
		if !replayed {
			if err := runner.states.RecordBindingDoc(runner.sessionID, runner.bootstrapID, runner.bindingDoc); err != nil {
				return "", err
			}
		} else if _, found, err := runner.states.LookupBindingDoc(runner.sessionID, runner.bootstrapID); err != nil {
			return "", err
		} else if !found {
			if err := runner.states.RecordBindingDoc(runner.sessionID, runner.bootstrapID, runner.bindingDoc); err != nil {
				return "", err
			}
		}
	}
	return recorded.BindingDigest, nil
}

// closeInput cuts operator input to the instance: lock-session first,
// so attached clients cannot type into the pane during the detach
// window, then detach-client -s, which detaches every client
// currently attached to the session. A locked session does not lock
// subsequently attaching clients (verified on tmux 3.6a), so new
// input-authorized attaches are refused AX-side at the attach entry
// once the quiescing state records; read-only attaches stay admitted
// because quiesce retains observation. Provider input enters only
// through the request-stop effect (the ordered next step), so no
// send-keys runs here. Both commands must exit clean; the evidence
// binds both exits. The detach is a second destructive command after
// the lock's round trip, so the boundary between them revalidates
// the authorization facts the engine checked before the effect: a
// generation rotated, authorization lapsed, or deadline passed
// during the lock refuses before the detach issues.
func (runner *backend) closeInput(ctx context.Context) (string, error) {
	for i, directive := range []Directive{DirectiveLockSession, DirectiveDetachClients} {
		if i > 0 {
			if err := runner.recheckPostWait(); err != nil {
				return "", err
			}
		}
		argv, err := runner.command(directive)
		if err != nil {
			return "", err
		}
		result, err := runner.runner.Run(ctx, argv)
		if err != nil {
			return "", err
		}
		if result.ExitCode != 0 {
			return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
		}
	}
	return effectEvidence(terminalbackend.EffectInputClosed, runner.operation, runner.instanceID, "locked", "detached"), nil
}

// recheckPostWait runs the entry-configured authorization
// revalidation at a destructive command boundary that follows a
// wait. A missing revalidation refuses: the boundary never runs
// blind.
func (runner *backend) recheckPostWait() error {
	if runner.recheck == nil {
		return &terminstance.Error{Code: terminalbackend.CodeProtocolError, Detail: "backend authorization recheck"}
	}
	return runner.recheck()
}

// startWrapper execs the instance new-session vector for create
// (wrapper_started) and restore (wrapper_restored).
func (runner *backend) startWrapper(ctx context.Context, effect terminalbackend.SideEffect) (string, error) {
	return runner.execEvidence(ctx, DirectiveNewSession, effect)
}

// createAttachClient records the presentation client receipt for one
// (instance, client) pair. Every call requires the valid unexpired
// attach authorization the attach handler already admitted; the store
// rechecks it, so a timeout creates no authorized input unless the
// durable receipt says so.
func (runner *backend) createAttachClient() (string, error) {
	if runner.attach == nil {
		return "", &terminstance.Error{Code: terminalbackend.CodeProtocolError, Detail: "backend attach store"}
	}
	receipt, _, err := runner.attach.Attach(runner.sessionID, runner.instanceID, runner.clientID, runner.transport, runner.input, runner.authRaw, runner.now())
	if err != nil {
		if code := backendErrorCode(err); code != "" {
			return "", &terminstance.Error{Code: code, Detail: "backend attach conflict"}
		}
		return "", err
	}
	return effectEvidence(terminalbackend.EffectAttachClientCreated, runner.operation, runner.instanceID, receipt.ClientID, receipt.Transport), nil
}

// execEvidence runs one directive and evidences its success. A nonzero
// exit is a coded process failure: the command ran and the effect did
// not commit. A transport error is uncoded: it proves nothing.
func (runner *backend) execEvidence(ctx context.Context, directive Directive, effect terminalbackend.SideEffect) (string, error) {
	argv, err := runner.command(directive)
	if err != nil {
		return "", err
	}
	result, err := runner.runner.Run(ctx, argv)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
	}
	return effectEvidence(effect, runner.operation, runner.instanceID, strconv.Itoa(result.ExitCode)), nil
}

// observeBoundary waits for the wrapper's safe-boundary signal on the
// quiescence channel with the bare blocking `wait-for` vector. The
// wait context already bounds the lesser of the request deadline and
// the timeout; its expiry is the coded quiesce_timeout proving
// non-commit, never a silent continue. The requested provider proof
// kind is bound into the evidence, so each kind proves its own
// boundary and an identical retry replays the same proof. For
// provider proof kinds the contemporaneous provider rows are bound
// alongside: the signal is the proof, the rows are audit context,
// and a failed row read binds its negative instead of failing the
// observed proof.
func (runner *backend) observeBoundary() (string, error) {
	argv, err := runner.command(DirectiveWaitBoundary)
	if err != nil {
		return "", err
	}
	result, err := runner.runner.Run(runner.waitCtx, argv)
	if err != nil {
		if runner.waitCtx.Err() != nil {
			return "", &terminstance.Error{Code: terminstance.CodeQuiesceTimeout, Detail: "backend boundary wait"}
		}
		return "", err
	}
	if result.ExitCode != 0 {
		if runner.waitCtx.Err() != nil {
			return "", &terminstance.Error{Code: terminstance.CodeQuiesceTimeout, Detail: "backend boundary wait"}
		}
		return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
	}
	observed, err := runner.now().UTC().MarshalText()
	if err != nil {
		return "", err
	}
	facts := []string{string(runner.proofKind), runner.channel, string(observed)}
	if terminstance.RequiresProviderObservation(runner.proofKind) {
		facts = append(facts, runner.boundaryProviderRows())
	}
	return effectEvidence(terminalbackend.EffectSafeBoundaryObserved, runner.operation, runner.instanceID, facts...), nil
}

// boundaryProviderRows reads the contemporaneous provider rows for a
// provider-kind boundary as bound context. Presence binds the row
// set; absence binds the negative; a failed read binds unknown. The
// verdict never fails: the wrapper signal already proved the
// boundary, and a post-signal crash must not discard that proof.
func (runner *backend) boundaryProviderRows() string {
	ctx := runner.waitCtx
	if ctx == nil {
		ctx = context.Background()
	}
	present, rows, err := runner.probePanes(ctx, runner.instanceID)
	if err != nil {
		return "provider:unknown"
	}
	if !present {
		return "provider:absent"
	}
	return "provider:" + strings.Join(rows, "\n")
}

// Absence stderr markers, verified against tmux 3.6a. has-session
// against a missing session reports `can't find session`; list-panes
// against a missing target reports `can't find window`; either
// command against a missing server socket reports `error connecting
// to <socket> (No such file or directory)`. Under a derived and
// custody-verified socket, a missing socket proves no listener and
// therefore no session. Every tmux failure exits 1, so the exit code
// alone proves nothing: only these marker-paired exits prove
// absence. Unrecognized stderr — permission failures, refused
// connections, version-drifted strings — fails closed to unknown.
const (
	absentMarkerNoSession = "can't find session"
	absentMarkerNoWindow  = "can't find window"
	absentMarkerNoSocket  = "No such file or directory"
	absentMarkerNoConnect = "error connecting to"
)

// errProbeUnknown is the uncoded unknown-probe signal: the probe ran
// (or never answered) without proving presence or absence. Callers
// return it verbatim; the engine and the lifecycle effect loop map
// uncoded backend failures to the row-allowed process failure with
// status_first and the unavailable observation, never to a verdict.
var errProbeUnknown = errors.New("tmux probe proves neither presence nor absence")

// classifyAbsence sorts one probe run into present, absent, or
// unknown. Exit 0 is present. A negative exit (a killed child that a
// runner reported as data) is unknown, never absent. A positive exit
// proves absence only with an absence marker on stderr; without one
// it is unknown. Callers pass the markers their command emits.
func classifyAbsence(result RunResult, markers ...string) (present, absent bool) {
	if result.ExitCode == 0 {
		return true, false
	}
	if result.ExitCode < 0 {
		return false, false
	}
	stderr := string(result.Stderr)
	for _, marker := range markers {
		if marker == absentMarkerNoSocket {
			if strings.Contains(stderr, absentMarkerNoConnect) && strings.Contains(stderr, absentMarkerNoSocket) {
				return false, true
			}
			continue
		}
		if strings.Contains(stderr, marker) {
			return false, true
		}
	}
	return false, false
}

// confirmClosed proves process closure for request-stop and
// terminate-stale. Positively observed absence (has-session exit with
// the no-session marker, or no server socket under the verified
// path) is the proof. A live session on request-stop escalates once
// to kill-session and re-confirms; a still-live session is the coded
// stop_timeout. An unproven session — transport failure, killed
// probe, or unmarked nonzero exit — is unknown at the deadline, never
// closure. A live session on terminate-stale is the coded process
// failure: the kill claimed the incarnation and it still answers.
// The confirm polls to the deadline before it concludes anything.
//
// The stop escalation is a new destructive command after the poll
// wait, so the poll bound (the graceful wait) expiring is exactly
// when the other authorization facts must be re-proved: the
// escalation revalidates the operation deadline, the AX
// authorization over the current lease, and the server generation
// before the kill issues. A graceful timeout escalates — the wait
// did its job — but an operation deadline passed, an authorization
// lapsed, or a generation rotated during the poll refuses without
// issuing the kill. The post-escalation re-confirmation observes
// under the operation deadline, not the exhausted graceful wait:
// reusing the expired wait would cancel the fresh probe before a
// cancellation-honouring executor could run it, making the success
// path unreachable. The terminate-stale tail issues no command
// after its poll, so it carries no revalidation: its refusal
// stands however the facts moved.
func (runner *backend) confirmClosed(ctx context.Context) (string, error) {
	absent, err := runner.pollSession(ctx, runner.deadline)
	if err != nil {
		return "", err
	}
	if absent {
		return effectEvidence(terminalbackend.EffectProcessClosed, runner.operation, runner.instanceID, "absent"), nil
	}
	if runner.operation == terminalbackend.OperationRequestStop {
		if err := runner.recheckPostWait(); err != nil {
			return "", err
		}
		argv, err := runner.command(DirectiveKillSession)
		if err != nil {
			return "", err
		}
		if result, err := runner.runner.Run(ctx, argv); err != nil {
			return "", err
		} else if result.ExitCode != 0 {
			return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
		}
		absent, err := runner.pollSession(ctx, runner.escalationDeadline())
		if err != nil {
			return "", err
		}
		if absent {
			return effectEvidence(terminalbackend.EffectProcessClosed, runner.operation, runner.instanceID, "escalated"), nil
		}
		return "", &terminstance.Error{Code: terminstance.CodeStopTimeout, Detail: "backend stop wait"}
	}
	// Terminate-stale re-confirm: a live session refuses whether the kill
	// exited clean or failed — the kill claimed the incarnation and it
	// still answers. The exit split names both members so each refuses
	// explicitly; the narrowing mutant admits exactly the failed-kill
	// member.
	if runner.killExit != 0 {
		return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
	}
	return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
}

// probeContext bounds one read-only probe by an absolute deadline:
// the duration from the backend clock to the deadline, enforced as
// a real-time context timeout, so a blocked probe returns at the
// bound even when the clock never advances. Duration-based, like
// the barrier waitUntil contract: fake-clock deadlines bound real
// waits. A zero deadline (paths with no poll contract) or a nil
// clock leaves the caller's context unmodified. Callers treat
// bound expiry as unknown — a timed-out probe proves neither
// presence nor absence — never as a verdict.
func probeContext(ctx context.Context, now func() time.Time, deadline time.Time) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if now == nil || deadline.IsZero() {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, deadline.Sub(now()))
}

// escalationDeadline selects the post-escalation observation
// budget: the operation deadline. A missing operation deadline
// falls back to the poll bound, preserving the single-budget poll
// for direct unit stagings that set no operation deadline; the
// Execute stop path always sets it.
func (runner *backend) escalationDeadline() time.Time {
	if runner.opDeadline.IsZero() {
		return runner.deadline
	}
	return runner.opDeadline
}

// pollSession probes has-session until the session is absent or the
// deadline passes. Presence polls on; absence returns at once; an
// unproven probe (transport failure, killed child, unmarked exit,
// or a probe still blocked when the bound fires) polls on until
// the deadline and then reports unknown, never closure. The bound
// cancels the in-flight probe itself, not just the loop around it:
// clock checks after Run cannot cancel a Run that has not
// returned. Answered probes follow the clock exactly as before;
// only a probe that never answered concludes by the context.
// Cancellation aborts at once.
func (runner *backend) pollSession(ctx context.Context, deadline time.Time) (bool, error) {
	pollCtx, cancel := probeContext(ctx, runner.now, deadline)
	defer cancel()
	for {
		if ctx != nil && ctx.Err() != nil {
			return false, ctx.Err()
		}
		argv, err := runner.command(DirectiveHasSession)
		if err != nil {
			return false, err
		}
		result, err := runner.runner.Run(pollCtx, argv)
		if err != nil {
			if ctx != nil && ctx.Err() != nil {
				return false, ctx.Err()
			}
			if pollCtx.Err() != nil || !runner.now().Before(deadline) {
				return false, errProbeUnknown
			}
			runner.sleep(runner.poll)
			continue
		}
		present, absent := classifyAbsence(result, absentMarkerNoSession, absentMarkerNoSocket)
		if absent {
			return true, nil
		}
		if !present {
			if !runner.now().Before(deadline) {
				return false, errProbeUnknown
			}
			runner.sleep(runner.poll)
			continue
		}
		if !runner.now().Before(deadline) {
			return false, nil
		}
		runner.sleep(runner.poll)
	}
}

// terminateStale kills the fenced stale incarnation session. A clean
// kill proves termination; a failed kill self-confirms — a
// positively absent session is terminated however it closed — and
// only a live session after a failed kill is the coded process
// failure. An unproven confirm is unknown, never termination. The
// confirm is a read-only probe under the same deadline-bound
// context as the close-confirm poll: a confirm still blocked when
// the bound fires concludes unknown, never termination.
func (runner *backend) terminateStale(ctx context.Context) (string, error) {
	argv, err := runner.command(DirectiveKillSession)
	if err != nil {
		return "", err
	}
	result, err := runner.runner.Run(ctx, argv)
	if err != nil {
		return "", err
	}
	runner.killExit = result.ExitCode
	if result.ExitCode == 0 {
		return effectEvidence(terminalbackend.EffectStaleIncarnationTerminated, runner.operation, runner.instanceID, "killed"), nil
	}
	confirm, err := runner.command(DirectiveHasSession)
	if err != nil {
		return "", err
	}
	confirmCtx, cancel := probeContext(ctx, runner.now, runner.deadline)
	defer cancel()
	observed, err := runner.runner.Run(confirmCtx, confirm)
	if err != nil {
		return "", errProbeUnknown
	}
	present, absent := classifyAbsence(observed, absentMarkerNoSession, absentMarkerNoSocket)
	if absent {
		return effectEvidence(terminalbackend.EffectStaleIncarnationTerminated, runner.operation, runner.instanceID, "confirmed-absent"), nil
	}
	if !present {
		return "", errProbeUnknown
	}
	return "", &terminstance.Error{Code: terminstance.CodeProcessFailed, Detail: "backend effect refused"}
}

// command builds the argv for one directive over the request identity.
func (runner *backend) command(directive Directive) ([]string, error) {
	built, err := BuildCommand(directive, CommandArgs{
		RuntimeDir:           runner.runtimeDir,
		Socket:               runner.socket,
		SessionID:            runner.sessionID,
		InstanceID:           runner.instanceID,
		QuiescenceGeneration: runner.quiescence,
	})
	if err != nil {
		if code := backendErrorCode(err); code != "" {
			return nil, &terminstance.Error{Code: code, Detail: "backend command refused"}
		}
		return nil, err
	}
	return built, nil
}

// backendErrorCode reads the wire code off a backend-construction or
// store error: landed terminalbackend and terminstance refusals plus
// local tmuxserver refusals all carry one. Anything else is uncoded.
func backendErrorCode(err error) string {
	if code := terminstance.ErrorCode(err); code != "" {
		return code
	}
	var local *Error
	if errors.As(err, &local) {
		return local.Code
	}
	return ""
}
