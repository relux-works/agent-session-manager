package tmuxserver

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// This file executes the three lifecycle operations the landed engine
// scope refuses — attach, terminate-stale, and restore — over the same
// landed gates the engine uses (transition matrix, idempotency shapes,
// authorization, capability conferral, receipt and binding stores)
// plus the tmux backend. The recheck loop, receipt discipline, and
// error-to-observation mapping mirror the engine's second side of each
// rule; the census symmetry lines name both sides.

// sortedEffects copies effects into bytewise order for the wire.
func sortedEffects(effects []terminalbackend.SideEffect) []terminalbackend.SideEffect {
	ordered := append([]terminalbackend.SideEffect(nil), effects...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered
}

// sortedEvidence copies evidence IDs into bytewise order for the wire.
func sortedEvidence(evidence []string) []string {
	ordered := append([]string(nil), evidence...)
	sort.Strings(ordered)
	return ordered
}

// attachDescriptor derives the bound attach descriptor: socket,
// instance, and client joined by newlines. It is host-local (the
// socket path is usable only on the responding host), deterministic,
// and never persisted: the attach handler returns it without writing
// it anywhere, and a dedicated test scans the stores for its bytes.
// Length is bounded by construction — sun_path socket plus two UUIDs
// stay two orders of magnitude below the 4096 cap — so no length arm
// exists to measure; a test pins the maximum instead.
func attachDescriptor(socket, instance, client string) string {
	return socket + "\n" + instance + "\n" + client
}

// createDescriptor derives the unbound create-interactive descriptor:
// socket and instance only, because create carries no client. Same
// non-persistence and length properties as the bound shape.
func createDescriptor(socket, instance string) string {
	return socket + "\n" + instance
}

// executeAttach runs one attach: closed-shape admission, deadline,
// transition, landed attach authorization, capability conferral plus
// the matching-transport arm, the fast-path quiesce barrier check,
// live server attestation (the P3-A wiring: a fresh decoy admission
// refuses), the authoritative barrier recheck under the ordering and
// shared per-instance admission locks, the durable client receipt, and the caller-executed attach
// vector with its descriptor. The entry returns the vector; it never
// execs an interactive attach.
func (lc *Lifecycle) executeAttach(ctx context.Context, req OpRequest, socket string) (OpOutcome, error) {
	empty := OpOutcome{Operation: terminalbackend.OperationAttach}
	body, err := parseAttachBody(req.Body)
	if err != nil {
		return empty, err
	}
	if err := checkLifecycleBackend(body.TerminalBackendID); err != nil {
		return empty, err
	}
	deadline, err := body.Deadline.Time()
	if err != nil {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body deadline"}
	}
	if !lc.Now().Before(deadline) {
		return empty, &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
	}
	if _, _, err := terminalbackend.CheckTransition(string(terminalbackend.OperationAttach), req.Source, false); err != nil {
		return empty, err
	}
	if err := terminalbackend.CheckAttachRequest(body.Authorization, body.Transport, body.InputAuthorized, lc.Now()); err != nil {
		return empty, err
	}
	if err := terminalbackend.CheckOperation(string(terminalbackend.OperationAttach), req.Admitted); err != nil {
		return empty, err
	}
	if err := checkAttachTransportCapability(body.Transport, req.Admitted); err != nil {
		return empty, err
	}
	if err := lc.checkAttachQuiesced(body.TerminalInstanceID, body.InputAuthorized); err != nil {
		return empty, err
	}
	if lc.ServerAdmission == nil {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "lifecycle dependencies"}
	}
	// One wait context bounds both waits below — the server
	// admission and the barrier acquisition — by the operation
	// deadline and the caller's cancellation: a deadline cancels
	// waiting, not a committed effect, and a timed-out wait
	// commits nothing late.
	waitCtx, waitCancel := lc.waitUntil(ctx, deadline)
	defer waitCancel()
	admission, err := lc.admitServer(waitCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return empty, &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
		}
		return empty, rowError(terminalbackend.OperationAttach, err)
	}
	if err := CheckServerAttested(admission, body.BackendGeneration); err != nil {
		return empty, &Error{Code: terminalbackend.CodeUnauthorized, Detail: "attach server attestation"}
	}
	// Ordering contract, attach side: the checks above ran before
	// the (possibly blocking) server admission and the barrier and
	// file-lock waits, so every live fact they observed may have gone
	// stale. The effect boundary below re-observes every live
	// authorization fact under both locks immediately before the
	// receipt commit, and the lock holds through the receipt, the
	// returned vector, and the advisory memory record. The
	// enumeration, first-observed site to revalidation site:
	// closure (fast path above, rechecked here); server
	// generation (admission above, live CurrentGeneration here);
	// request deadline (entry above, live clock here — both lock
	// waits count against the deadline, and the deadline stays
	// distinct from the authorization expiry); authorization
	// expiry, transport binding, and input binding (entry
	// CheckAttachRequest above, live clock rechecked inside the
	// commit call itself, which holds no wait between its check
	// and its no-replace install, plus the post-commit result
	// binding for input). Transition, operation capability, and
	// backend identity are pure checks over request-immutable
	// members — no live fact, nothing to revalidate. The
	// attestation value is immutable once returned; its live
	// counterpart is the generation rechecked here. The overlap
	// possibility (below) is likewise a live fact: a peer client
	// may commit between server admission and these locks, so the
	// receipt census is read here, not earlier. Residual: a
	// rotation landing between this recheck and the commit —
	// inside one process or across processes — is not excluded;
	// the window holds no waits (file I/O only). That remainder
	// is the B42 in-process scope, extended to the generation
	// and deadline rechecks. The OS-backed per-instance admission
	// lock spans peer census through receipt install across
	// independent Lifecycle values and processes; quiesce ordering
	// remains process-local (bound B42).
	release, err := lc.acquireBarrier(waitCtx)
	if err != nil {
		return empty, &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
	}
	defer release()
	releaseAdmission, err := lc.Attach.AcquireAdmission(waitCtx, body.TerminalInstanceID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return empty, &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
		}
		return empty, rowError(terminalbackend.OperationAttach, err)
	}
	defer releaseAdmission()
	// File-lock acquisition is another wait boundary. Recheck facts
	// that could change while an independent attach held the lock.
	if err := lc.checkAttachQuiesced(body.TerminalInstanceID, body.InputAuthorized); err != nil {
		return empty, err
	}
	if !lc.Now().Before(deadline) {
		return empty, &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
	}
	if lc.CurrentGeneration() != body.BackendGeneration {
		return empty, &Error{Code: terminalbackend.CodeStaleGeneration, Detail: "backend_generation stale"}
	}
	if err := lc.checkAttachOverlap(body, req.Admitted); err != nil {
		return empty, err
	}
	reqBackend := lc.requestBackend(req, socket, body.SessionID, body.TerminalInstanceID)
	reqBackend.operation = terminalbackend.OperationAttach
	reqBackend.clientID = body.ClientID
	reqBackend.transport = body.Transport
	reqBackend.input = body.InputAuthorized
	reqBackend.authRaw = body.AuthRaw
	evidenceID, err := reqBackend.PerformEffect(ctx, terminalbackend.EffectAttachClientCreated)
	if err != nil {
		return empty, rowError(terminalbackend.OperationAttach, err)
	}
	argv, err := BuildCommand(DirectiveAttachSession, CommandArgs{
		RuntimeDir:     lc.RuntimeDir,
		Socket:         socket,
		InstanceID:     body.TerminalInstanceID,
		AttachInput:    body.InputAuthorized,
		HasAttachInput: true,
	})
	if err != nil {
		return empty, err
	}
	if err := terminalbackend.CheckAttachResult(body.InputAuthorized, body.InputAuthorized, body.Authorization); err != nil {
		return empty, err
	}
	// Only an input-authorized attach adopts the active state: a
	// read-only attach is observation, and recording active for it
	// would reopen quiesced input (the quiesce barrier reads this
	// same memory, so observation must leave it untouched).
	if body.InputAuthorized {
		_ = lc.recordAfter(body.TerminalInstanceID, terminalbackend.StateActive)
	}
	return OpOutcome{
		Operation: terminalbackend.OperationAttach,
		Attach: &AttachOutcome{
			ClientMirrorID:  body.ClientID,
			Descriptor:      attachDescriptor(socket, body.TerminalInstanceID, body.ClientID),
			InputAuthorized: body.InputAuthorized,
			EvidenceIDs:     []string{evidenceID},
		},
		Argv: argv,
	}, nil
}

// checkAttachQuiesced enforces the quiesce input barrier at the
// attach entry: tmux offers no session freeze (a locked session does
// not lock subsequently attaching clients), so once a closure
// report proves the input closed in the current incarnation, new
// input-authorized attaches refuse the failed local precondition.
// Read-only attaches stay admitted: quiesce stops input while
// retaining observation. The incarnation-scoped closure proof is the
// sole admission verdict: advisory memory, caller-carried source
// state, and failure reporting are never consulted for their value
// here, so none of them can reopen a closed barrier — a stale
// active record, a failure result echoing its source, and a lost
// state write all refuse exactly like the proof itself. Only a
// fresh create or restore success reopens the input, by rotating
// the incarnation the proof is scoped to. Store health still fails
// closed: an unreadable state record is unknown, never admission,
// so the memory read error propagates even though its value is
// ignored. A proof read failure is likewise an error, never
// admission: unknown is not a verdict.
func (lc *Lifecycle) checkAttachQuiesced(instanceID string, inputAuthorized bool) error {
	if !inputAuthorized {
		return nil
	}
	if _, _, err := lc.States.Lookup(instanceID); err != nil {
		return err
	}
	proven, err := lc.States.QuiesceBarrierProven(instanceID)
	if err != nil {
		return err
	}
	if proven {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "attach quiesced input"}
	}
	return nil
}

// admissionResult is one server-admission answer from the waiter
// goroutine.
type admissionResult struct {
	admission RealmAdmission
	err       error
}

// admitServer runs the injected server admission under the wait
// context and reports its verdict: the entry takes whichever
// answers first — the admission or the deadline. The bound is
// enforced, never cooperative: an adapter that ignores
// cancellation still loses at the deadline, and its late answer is
// discarded, never committed. The abandoned waiter send never
// blocks — the channel is buffered — and its goroutine exits when
// the adapter returns, so nothing here waits past the bound and
// nothing commits after the caller timed out.
func (lc *Lifecycle) admitServer(ctx context.Context) (RealmAdmission, error) {
	// An already-fired wait never starts its admission: the
	// request is refused deterministically instead of racing the
	// waiter goroutine against the fired context.
	if err := ctx.Err(); err != nil {
		return RealmAdmission{}, err
	}
	answered := make(chan admissionResult, 1)
	go func() {
		admission, err := lc.ServerAdmission(ctx)
		answered <- admissionResult{admission: admission, err: err}
	}()
	select {
	case result := <-answered:
		return result.admission, result.err
	case <-ctx.Done():
		return RealmAdmission{}, ctx.Err()
	}
}

// checkAttachOverlap enforces the §4.C attach overlap capabilities
// at the effect boundary: a new client that may overlap a recorded
// peer needs multi_attach, and new concurrent input over a recorded
// input-authorized peer needs multiple_input_clients. Overlap cannot
// be ruled out from this protocol's evidence: the returned attach
// vector is executed by the caller, and AX has no positive client
// identity or detach signal that retires a receipt. A valid peer
// receipt proves an admitted client claim; if that client can no
// longer be shown live, its liveness is UNKNOWN and the receipt
// remains a possible peer until an authoritative retirement
// contract exists. A validated identical replay is
// exempt: the requesting client already holds a recorded receipt,
// and the durable store below replays it — or refuses
// idempotency_mismatch when transport or input changed — after
// rechecking the live authorization. The exemption keys on the
// recorded receipt, a Lookup-proven durable fact, never on the bare
// caller client ID: a client with no recorded receipt still gates
// below however it names itself. Generation and deadline already
// rechecked above; authorization rechecks in the commit. A
// peer-census read failure is an error, never admission: unknown
// is not a verdict. This conservative liveness rule deliberately
// retains detached or not-yet-executed receipts as possible peers.
func (lc *Lifecycle) checkAttachOverlap(body AttachBody, admitted terminalbackend.Admitted) error {
	peers, err := lc.Attach.Peers(body.SessionID, body.TerminalInstanceID, body.ClientID)
	if err != nil {
		return err
	}
	if len(peers) == 0 {
		return nil
	}
	_, found, err := lc.Attach.Lookup(body.SessionID, body.TerminalInstanceID, body.ClientID)
	if err != nil {
		return err
	}
	if found {
		return nil
	}
	if !admitted.Has("multi_attach") {
		return &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
	}
	if !body.InputAuthorized {
		return nil
	}
	for _, peer := range peers {
		if peer.InputAuthorized {
			if !admitted.Has("multiple_input_clients") {
				return &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
			}
			return nil
		}
	}
	return nil
}

// checkAttachTransportCapability enforces the matching-transport arm:
// local presentation needs local_attach, mesh presentation needs
// remote_attach. The relay member never reaches here — the landed
// attach authorization refuses it first — so no third arm exists.
func checkAttachTransportCapability(transport string, admitted terminalbackend.Admitted) error {
	switch transport {
	case string(terminalbackend.TransportLocalOnly):
		if !admitted.Has("local_attach") {
			return &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
		}
	case string(terminalbackend.TransportTrustedPrivateMesh):
		if !admitted.Has("remote_attach") {
			return &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
		}
	default:
		return &Error{Code: terminalbackend.CodeProtocolError, Detail: "presentation transport vocabulary"}
	}
	return nil
}

// executeTerminate runs one terminate-stale: closed-shape admission,
// idempotency material, capability conferral plus the dual conditional,
// transition, force_stale authorization, generation recheck, landed
// fencing authorization, deadline, receipt discipline with
// status-reconciled resume, and the kill plus close-confirm effects.
func (lc *Lifecycle) executeTerminate(ctx context.Context, req OpRequest, socket string) (OpOutcome, error) {
	empty := OpOutcome{Operation: terminalbackend.OperationTerminateStale}
	body, err := parseTerminateBody(req.Body)
	if err != nil {
		return empty, err
	}
	operation := terminalbackend.OperationTerminateStale
	mctx := body.Context
	source := terminalbackend.InstanceState(req.Source)
	if err := checkLifecycleBackend(mctx.TerminalBackendID); err != nil {
		return empty, err
	}
	derived, err := terminalbackend.IdempotencyKey(string(operation), mctx.TerminalInstanceID, "terminate", body.StaleLeaseID, strconv.FormatUint(body.StaleEpoch, 10))
	if err != nil {
		return empty, err
	}
	if mctx.IdempotencyKey != derived {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "idempotency key material"}
	}
	if err := terminalbackend.CheckOperation(string(operation), req.Admitted); err != nil {
		failure := rowError(operation, err)
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if err := checkTerminateCapability(req.Admitted); err != nil {
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
		return OpOutcome{Operation: operation, Mutation: &result}, err
	}
	target, effects, err := terminalbackend.CheckTransition(string(operation), string(source), false)
	if err != nil {
		failure := rowError(operation, err)
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if err := terminstance.CheckAuthorization(mctx.Authorization, terminstance.AuthorizationForceStale, lc.CurrentLease(), lc.Now()); err != nil {
		failure := rowError(operation, err)
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if lc.CurrentGeneration() != mctx.BackendGeneration {
		failure := &Error{Code: terminalbackend.CodeStaleGeneration, Detail: "backend_generation stale"}
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if err := mapFencingError(fencing.AuthorizeTerminateStale(req.Presented, req.Winner, req.HasWinner, req.ForceRecovery, req.DiagnosticsPreserved)); err != nil {
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
		return OpOutcome{Operation: operation, Mutation: &result}, err
	}
	if err := checkTerminateTarget(mctx, body, req.Presented); err != nil {
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
		return OpOutcome{Operation: operation, Mutation: &result}, err
	}
	if err := checkTerminateWinner(mctx, req.Winner, lc.CurrentLease()); err != nil {
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
		return OpOutcome{Operation: operation, Mutation: &result}, err
	}
	deadline, err := mctx.Deadline.Time()
	if err != nil {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "mutation context deadline"}
	}
	if !lc.Now().Before(deadline) {
		failure := &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionReplaySame)
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	reqBackend := lc.requestBackend(req, socket, mctx.SessionID, mctx.TerminalInstanceID)
	reqBackend.operation = operation
	reqBackend.deadline = deadline
	return lc.executeWithReceipt(ctx, req, socket, operation, mctx, source, target, effects, deadline, terminstance.AuthorizationForceStale, reqBackend, func(result terminstance.Result) OpOutcome {
		return OpOutcome{Operation: operation, Mutation: &result, TargetClosed: true}
	})
}

// checkTerminateCapability enforces the dual conditional: stale
// termination needs stale_process_termination AND
// provider_process_observation. Each arm refuses alone.
func checkTerminateCapability(admitted terminalbackend.Admitted) error {
	if !admitted.Has("stale_process_termination") {
		return &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
	}
	if !admitted.Has("provider_process_observation") {
		return &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
	}
	return nil
}

// mapFencingError maps the landed terminate-stale authorization onto
// the row's allowed set: malformed fencing material is the protocol
// error, every other fencing refusal — no winner, no force, no
// diagnostics, target mismatch, live owner — is the failed local
// precondition. lease_conflict maps to the precondition too: the row
// has no lease_conflict member, and a target fenced by another
// session's winner fails the fenced-target precondition.
func mapFencingError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fencing.ErrInvalidArguments) {
		return &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate fencing material"}
	}
	return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "terminate fencing authorization"}
}

// checkTerminateTarget binds the presented fencing token to the
// request: session, stale lease, and stale epoch must all agree with
// the context and body. Any disagreement refuses the failed local
// precondition: the retry would authorize a different target than the
// key names.
func checkTerminateTarget(mctx terminstance.MutationContext, body TerminateBody, presented fencing.PresentedToken) error {
	if presented.SessionID != mctx.SessionID ||
		presented.LeaseID != body.StaleLeaseID ||
		presented.Epoch != body.StaleEpoch {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "terminate target binding"}
	}
	return nil
}

// checkTerminateWinner binds the caller-observed winner to the
// re-read current lease: the fencing authorization decides over the
// caller's winner, and only a winner that still wins authorizes. A
// rotated or misobserved winner refuses the failed local
// precondition; the caller re-observes and retries. Absence of a
// winner never reaches here — the fencing gate refuses it first.
func checkTerminateWinner(mctx terminstance.MutationContext, winner sessrepo.LeaseSummary, lease terminstance.LeaseView) error {
	if winner.SessionID != mctx.SessionID ||
		winner.LeaseID != lease.LeaseID ||
		winner.Epoch != lease.Epoch {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "terminate winner binding"}
	}
	return nil
}

// executeRestore runs one restore: closed-shape admission, idempotency
// material, capability conferral plus the reboot conditional,
// transition, restore authorization, generation recheck, prior-binding
// mismatch, deadline, receipt discipline with status-reconciled
// resume, the persist plus wrapper-restored effects, and the required
// result binding. The wire result is always parked. The backend
// launches no provider: the wrapper owns launch gating through the
// after-restore composition (ExecuteWrapperRestore), and this entry
// only recreates the wrapper process.
//
// The result binding is the create-time document kept alongside the
// bootstrap receipt, re-admitted by termbind on the read — within one
// generation — or its minted successor across a reboot: the termbind
// owner succeeds the prior (the JCS identity rule stays
// termbind-private) and the successor persists under the same pair
// key. The prior-binding mismatch arm proves the carried prior equals
// the recorded digest; the document read proves a valid binding for
// this session, instance, and backend.
func (lc *Lifecycle) executeRestore(ctx context.Context, req OpRequest, socket string) (OpOutcome, error) {
	empty := OpOutcome{Operation: terminalbackend.OperationRestore}
	body, err := parseRestoreBody(req.Body)
	if err != nil {
		return empty, err
	}
	operation := terminalbackend.OperationRestore
	mctx := body.Context
	source := terminalbackend.InstanceState(req.Source)
	if err := checkLifecycleBackend(mctx.TerminalBackendID); err != nil {
		return empty, err
	}
	derived, err := terminalbackend.IdempotencyKey(string(operation), mctx.SessionID, body.BootstrapOperationID)
	if err != nil {
		return empty, err
	}
	if mctx.IdempotencyKey != derived {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "idempotency key material"}
	}
	if err := terminalbackend.CheckOperation(string(operation), req.Admitted); err != nil {
		failure := rowError(operation, err)
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if !req.Admitted.Has("reboot_restoration") {
		failure := &Error{Code: terminalbackend.CodeCapabilityUnproven, Detail: "operation capability conditional"}
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	target, effects, err := terminalbackend.CheckTransition(string(operation), string(source), false)
	if err != nil {
		failure := rowError(operation, err)
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if err := terminstance.CheckAuthorization(mctx.Authorization, terminstance.AuthorizationRestore, lc.CurrentLease(), lc.Now()); err != nil {
		failure := rowError(operation, err)
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if lc.CurrentGeneration() != mctx.BackendGeneration {
		failure := &Error{Code: terminalbackend.CodeStaleGeneration, Detail: "backend_generation stale"}
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if err := lc.checkPriorBinding(body); err != nil {
		var local *Error
		if errors.As(err, &local) && local.Code == terminalbackend.CodeRestoreMismatch {
			result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
			return OpOutcome{Operation: operation, Mutation: &result}, err
		}
		failure := &Error{Code: terminstance.CodeProcessFailed, Detail: "backend binding read"}
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	deadline, err := mctx.Deadline.Time()
	if err != nil {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "mutation context deadline"}
	}
	if !lc.Now().Before(deadline) {
		failure := &Error{Code: terminstance.CodeTimeout, Detail: "operation deadline"}
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionReplaySame)
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	// The prior binding validates before the receipt binds and the
	// wrapper recreates: a missing, corrupt, or disagreeing prior
	// refuses here, never after the irreversible effects.
	if _, err := lc.readRestorePrior(mctx, body); err != nil {
		result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
		return OpOutcome{Operation: operation, Mutation: &result}, err
	}
	reqBackend := lc.requestBackend(req, socket, mctx.SessionID, mctx.TerminalInstanceID)
	reqBackend.operation = operation
	reqBackend.bootstrapID = body.BootstrapOperationID
	// The persist effect replays the recorded pair receipt (verified
	// by the mismatch arm above); the candidate carries the matched
	// prior digest so the store admits its shape on the replay path.
	reqBackend.bindingID = body.PriorBindingID
	outcome, err := lc.executeWithReceipt(ctx, req, socket, operation, mctx, source, target, effects, deadline, terminstance.AuthorizationRestore, reqBackend, func(result terminstance.Result) OpOutcome {
		return OpOutcome{Operation: operation, Mutation: &result, PriorBindingID: body.PriorBindingID, RestoredParked: true}
	})
	if err != nil {
		return outcome, err
	}
	binding, err := lc.restoreBinding(mctx, body)
	if err != nil {
		return outcome, err
	}
	outcome.Binding = binding
	return outcome, nil
}

// readRestorePrior reads the create-time binding document kept under
// the (session, bootstrap) pair, re-admitted by the termbind owner,
// and proves it names this restore's session, instance, and backend.
// Generation is not compared here: after a reboot the restore carries
// the new server generation while the prior still names the old one,
// and succession (not agreement) resolves that drift. A missing
// document is the backend store failure — the receipt proved the
// prior, but the payload is gone, so the required result cannot be
// built. A document that fails termbind admission, or that names
// another session, instance, or backend than the mutation context, is
// the integrity failure: the store disagrees with the receipt it
// stands beside.
func (lc *Lifecycle) readRestorePrior(mctx terminstance.MutationContext, body RestoreBody) (termbind.Binding, error) {
	empty := termbind.Binding{}
	doc, found, err := lc.States.LookupBindingDoc(mctx.SessionID, body.BootstrapOperationID)
	if err != nil {
		return empty, &Error{Code: terminstance.CodeProcessFailed, Detail: "backend binding read"}
	}
	if !found {
		return empty, &Error{Code: terminstance.CodeProcessFailed, Detail: "backend binding read"}
	}
	binding, err := termbind.ParseTerminalBinding(doc)
	if err != nil {
		return empty, &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "restore binding image"}
	}
	if binding.SessionID != mctx.SessionID ||
		binding.TerminalInstanceID != mctx.TerminalInstanceID ||
		binding.TerminalBackendID != mctx.TerminalBackendID {
		return empty, &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "restore binding image"}
	}
	return binding, nil
}

// restoreBinding builds the required restore result binding after the
// effects commit. Within one generation the prior is the result: the
// re-admitted create-time document the pre-effects validation already
// proved. Across a reboot the mutation context carries the new server
// generation, and the result is the minted successor: the termbind
// owner succeeds the prior (new generation, supersedes link, fresh
// identity), the successor persists under the same pair key, and the
// same successor returns. The mint is deterministic, so a retry after
// a crash between the effects and the persist converges on the
// identical document instead of forking, and a retry after the
// persist replays the stored successor without minting again.
func (lc *Lifecycle) restoreBinding(mctx terminstance.MutationContext, body RestoreBody) (*termbind.Binding, error) {
	prior, err := lc.readRestorePrior(mctx, body)
	if err != nil {
		return nil, err
	}
	if prior.BackendGeneration == mctx.BackendGeneration {
		return &prior, nil
	}
	minted, raw, err := termbind.MintSuccessorBinding(prior, mctx.BackendGeneration)
	if err != nil {
		return nil, &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "restore binding image"}
	}
	if err := lc.States.RecordBindingDoc(mctx.SessionID, body.BootstrapOperationID, raw); err != nil {
		return nil, &Error{Code: terminstance.CodeProcessFailed, Detail: "backend binding write"}
	}
	return &minted, nil
}

// checkPriorBinding enforces the restore mismatch rule: the recorded
// bootstrap binding for the (session, bootstrap) pair must exist and
// its digest must equal the carried prior_binding_id. Substituting any
// other binding — including the configured default — refuses the
// restore mismatch. A binding read failure is an error, never a
// mismatch: unknown is not a verdict.
func (lc *Lifecycle) checkPriorBinding(body RestoreBody) error {
	recorded, found, err := lc.Bindings.Lookup(body.Context.SessionID, body.BootstrapOperationID)
	if err != nil {
		return err
	}
	if !found || recorded.BindingDigest != body.PriorBindingID {
		return &Error{Code: terminalbackend.CodeRestoreMismatch, Detail: "restore prior binding"}
	}
	return nil
}

// effectPerformer executes one lifecycle side effect and reports its
// evidence ID, the single capability the receipt-bound effect loop
// needs from the request backend.
type effectPerformer interface {
	PerformEffect(ctx context.Context, effect terminalbackend.SideEffect) (string, error)
}

// executeWithReceipt binds the idempotency receipt and runs the
// effects under it, mirroring the engine's bind discipline: an
// identical retry replays the one completion, a bound key without its
// completion reconciles through the production status entry before it
// refuses, and no path binds a second receipt for one key. complete
// maps the successful result into the operation outcome; failures
// return the outcome with the mutation set, mirroring the engine.
func (lc *Lifecycle) executeWithReceipt(ctx context.Context, req OpRequest, socket string, operation terminalbackend.Operation, mctx terminstance.MutationContext, source, target terminalbackend.InstanceState, effects []terminalbackend.SideEffect, deadline time.Time, kind terminstance.AuthorizationKind, reqBackend effectPerformer, complete func(terminstance.Result) OpOutcome) (OpOutcome, error) {
	key := mctx.IdempotencyKey
	_, replayed, err := lc.Receipts.Bind(key, operation, mctx.OperationID)
	if err != nil {
		if backendErrorCode(err) != "" {
			result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(err), false))
			return OpOutcome{Operation: operation, Mutation: &result}, rowError(operation, err)
		}
		if _, found, lookupErr := lc.Receipts.Lookup(key); lookupErr == nil && !found {
			failure := &Error{Code: terminstance.CodeProcessFailed, Detail: "idempotency store failure"}
			result := mutationResult(mctx, source, source, nil, nil, terminstance.DispositionFor(backendErrorCode(failure), false))
			return OpOutcome{Operation: operation, Mutation: &result}, failure
		}
		failure := &Error{Code: terminstance.CodeProcessFailed, Detail: "idempotency store failure"}
		result := mutationResult(mctx, source, terminalbackend.StateUnavailable, nil, nil, terminstance.DispositionStatusFirst)
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if replayed {
		// A replayed success replays the stored result without
		// touching the incarnation: replaying an old restore
		// must not supersede a newer closure. Only the fresh
		// path rotates, after its effects commit.
		stored, found, err := lc.Receipts.Completed(key)
		if err != nil {
			failure := &Error{Code: terminstance.CodeProcessFailed, Detail: "idempotency store failure"}
			result := mutationResult(mctx, source, terminalbackend.StateUnavailable, nil, nil, terminstance.DispositionStatusFirst)
			return OpOutcome{Operation: operation, Mutation: &result}, failure
		}
		if !found {
			return lc.resumeUncertain(ctx, req, socket, operation, mctx, source, target, effects, deadline, kind, reqBackend, complete)
		}
		var result terminstance.Result
		if err := json.Unmarshal(stored, &result); err != nil {
			failure := &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "idempotency result image"}
			rebuilt := mutationResult(mctx, source, terminalbackend.StateUnavailable, nil, nil, terminstance.DispositionStatusFirst)
			return OpOutcome{Operation: operation, Mutation: &rebuilt}, failure
		}
		if err := terminstance.CheckResult(mctx, source, result); err != nil {
			if backendErrorCode(err) == "" {
				failure := &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "idempotency result image"}
				rebuilt := mutationResult(mctx, source, terminalbackend.StateUnavailable, nil, nil, terminstance.DispositionStatusFirst)
				return OpOutcome{Operation: operation, Mutation: &rebuilt}, failure
			}
			rebuilt := mutationResult(mctx, source, terminalbackend.StateUnavailable, nil, nil, terminstance.DispositionFor(backendErrorCode(err), true))
			return OpOutcome{Operation: operation, Mutation: &rebuilt}, rowError(operation, err)
		}
		return complete(result), nil
	}
	if lc.Hooks != nil && lc.Hooks.AfterReceipt != nil {
		lc.Hooks.AfterReceipt(operation, key, terminstance.InterimState(operation, source))
	}
	return lc.runLifecycleEffects(ctx, operation, mctx, source, target, effects, deadline, kind, reqBackend, complete)
}

// resumeUncertain reconciles a bound key without its completion through
// the production status entry: when status proves the presented source
// state — nothing happened — the same operation continues under the
// same receipt; when status proves anything else, or fails, the retry
// refuses uncertain with status_first. No path binds a second receipt
// for one key.
func (lc *Lifecycle) resumeUncertain(ctx context.Context, req OpRequest, socket string, operation terminalbackend.Operation, mctx terminstance.MutationContext, source, target terminalbackend.InstanceState, effects []terminalbackend.SideEffect, deadline time.Time, kind terminstance.AuthorizationKind, reqBackend effectPerformer, complete func(terminstance.Result) OpOutcome) (OpOutcome, error) {
	body := terminstance.StatusBody{
		SessionID:                  mctx.SessionID,
		TerminalInstanceID:         mctx.TerminalInstanceID,
		HasTerminalInstanceID:      true,
		TerminalBackendID:          mctx.TerminalBackendID,
		ImplementationVersion:      mctx.ImplementationVersion,
		ProtocolVersion:            mctx.ProtocolVersion,
		BackendGeneration:          mctx.BackendGeneration,
		HasBackendGeneration:       true,
		IncludeProviderObservation: false,
		Deadline:                   mctx.Deadline,
	}
	statusBackend := lc.requestBackend(req, socket, mctx.SessionID, mctx.TerminalInstanceID)
	statusBackend.operation = terminalbackend.OperationStatus
	report, err := lc.engineFor(statusBackend).ExecuteStatus(ctx, body, source, req.Admitted)
	if err != nil || report.State != source {
		failure := &Error{Code: terminalbackend.CodeUnavailable, Detail: "idempotency result uncertain"}
		result := mutationResult(mctx, source, terminalbackend.StateUnavailable, nil, nil, terminstance.DispositionStatusFirst)
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if lc.Hooks != nil && lc.Hooks.AfterReceipt != nil {
		lc.Hooks.AfterReceipt(operation, mctx.IdempotencyKey, terminstance.InterimState(operation, source))
	}
	return lc.runLifecycleEffects(ctx, operation, mctx, source, target, effects, deadline, kind, reqBackend, complete)
}

// runLifecycleEffects performs the transition's side effects under an
// already-bound receipt with the per-effect deadline, authorization,
// and generation gates, then completes the key exactly once — the
// engine's runEffects contract, second side. Fresh and resumed runs
// share this loop. On success the after-state records in the AX-side
// memory and the outcome completes; every failure returns the outcome
// with the mutation set, mirroring the engine.
func (lc *Lifecycle) runLifecycleEffects(ctx context.Context, operation terminalbackend.Operation, mctx terminstance.MutationContext, source, target terminalbackend.InstanceState, effects []terminalbackend.SideEffect, deadline time.Time, kind terminstance.AuthorizationKind, reqBackend effectPerformer, complete func(terminstance.Result) OpOutcome) (OpOutcome, error) {
	key := mctx.IdempotencyKey
	committed := false
	evidence := []string(nil)
	performed := []terminalbackend.SideEffect(nil)
	fail := func(code, detail string) (OpOutcome, error) {
		after := source
		if committed {
			after = terminalbackend.StateUnavailable
		}
		failure := &Error{Code: code, Detail: detail}
		result := mutationResult(mctx, source, after, performed, evidence, terminstance.DispositionFor(code, committed))
		lc.recordFailure(mctx.TerminalInstanceID, after)
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	for _, effect := range effects {
		if !lc.Now().Before(deadline) {
			return fail(rowTimeoutCode(operation), "operation deadline")
		}
		if err := terminstance.CheckAuthorization(mctx.Authorization, kind, lc.CurrentLease(), lc.Now()); err != nil {
			code := backendErrorCode(err)
			if code == "" {
				code = terminalbackend.CodeUnauthorized
			}
			return fail(code, "lifecycle authorization recheck")
		}
		if lc.CurrentGeneration() != mctx.BackendGeneration {
			return fail(terminalbackend.CodeStaleGeneration, "backend_generation stale")
		}
		if lc.Hooks != nil && lc.Hooks.BeforeEffect != nil {
			lc.Hooks.BeforeEffect(effect)
		}
		evidenceID, err := reqBackend.PerformEffect(ctx, effect)
		if lc.Hooks != nil && lc.Hooks.AfterEffect != nil {
			lc.Hooks.AfterEffect(effect)
		}
		if err != nil {
			code := backendErrorCode(err)
			if code == "" {
				return fail(terminstance.CodeProcessFailed, "backend effect uncertain")
			}
			if checkErr := terminalbackend.CheckErrorAllowed(string(operation), code); checkErr != nil {
				return fail(terminalbackend.CodeProtocolError, "operation error vocabulary")
			}
			return fail(code, "backend effect refused")
		}
		if _, err := scalar.ParseDigest(evidenceID); err != nil {
			return fail(terminalbackend.CodeIntegrityFailure, "backend effect evidence")
		}
		committed = true
		performed = append(performed, effect)
		evidence = append(evidence, evidenceID)
	}
	result := mutationResult(mctx, source, target, performed, evidence, terminstance.DispositionReplaySame)
	encoded, err := json.Marshal(result)
	if err != nil {
		failure := &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "idempotency result image"}
		result.Disposition = terminstance.DispositionStatusFirst
		return OpOutcome{Operation: operation, Mutation: &result}, failure
	}
	if err := lc.Receipts.Complete(key, encoded); err != nil {
		if backendErrorCode(err) == "" {
			failure := &Error{Code: terminstance.CodeProcessFailed, Detail: "idempotency store failure"}
			result.Disposition = terminstance.DispositionStatusFirst
			return OpOutcome{Operation: operation, Mutation: &result}, failure
		}
		result.Disposition = terminstance.DispositionStatusFirst
		return OpOutcome{Operation: operation, Mutation: &result}, rowError(operation, err)
	}
	if operation == terminalbackend.OperationRestore {
		// A fresh restore success opens a new incarnation like a
		// fresh create: prior closure reports are superseded.
		// Terminate-stale never rotates — it closes the
		// incarnation toward stopped, which admits no attach —
		// and a failed rotation fails the operation closed.
		if err := lc.States.RecordIncarnation(mctx.TerminalInstanceID, key); err != nil {
			failure := &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "restore incarnation image"}
			return OpOutcome{Operation: operation, Mutation: &result}, failure
		}
	}
	_ = lc.recordAfter(mctx.TerminalInstanceID, target)
	return complete(result), nil
}

// rowTimeoutCode reports the §4.C timeout vocabulary per operation
// row: the wait row times out quiesce_timeout and the stop row
// stop_timeout, while every other row times out
// terminal_backend_timeout. This mirrors the landed engine's
// per-row mapping for the operations the engine scope leaves out
// and the within-effect revalidation; the literals are pinned by
// the row tests, never derived from the engine constant.
func rowTimeoutCode(operation terminalbackend.Operation) string {
	switch operation {
	case terminalbackend.OperationWaitSafeBoundary:
		return terminstance.CodeQuiesceTimeout
	case terminalbackend.OperationRequestStop:
		return terminstance.CodeStopTimeout
	default:
		return terminstance.CodeTimeout
	}
}
