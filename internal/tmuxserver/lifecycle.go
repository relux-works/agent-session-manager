package tmuxserver

import (
	"context"
	"sync"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// Lifecycle executes the eight tmux lifecycle operations — create,
// attach, status, quiesce-input, wait-safe-boundary, request-stop,
// terminate-stale, restore — over the structured commands. Five
// operations (create, quiesce-input, wait-safe-boundary, request-stop,
// status) delegate to the landed terminstance engine with the tmux
// backend; attach, terminate-stale, and restore — which the engine
// scope gate refuses — execute here over the same landed gates
// (transition matrix, idempotency shapes, authorization, capability
// conferral, receipt and binding stores) plus the tmux backend.
//
// A Lifecycle is constructed whole: every store and function value is
// required (except Sleep/PollInterval, which default, and
// ServerAdmission, which only attach needs). The Execute entry refuses
// before any work when a dependency is missing.
type Lifecycle struct {
	// RuntimeDir is the tmux runtime directory (<root>/tmux) the
	// socket derives from.
	RuntimeDir string
	// Root is the Runtime IPC root the socket custody verifies.
	Root string
	// Platform selects the sun_path limit.
	Platform scalar.Platform
	// Runner executes the tmux commands.
	Runner Runner
	// Receipts is the durable idempotency receipt table.
	Receipts *terminstance.ReceiptStore
	// Bindings is the durable bootstrap binding store.
	Bindings *axpane.Store
	// Attach is the durable attach-client receipt store.
	Attach *termbind.AttachStore
	// States is the durable AX-side state memory.
	States *InstanceStates
	// CurrentLease and CurrentGeneration re-read the winning lease
	// and the validated binding generation at every check point.
	CurrentLease      func() terminstance.LeaseView
	CurrentGeneration func() string
	// Now is the clock.
	Now func() time.Time
	// Sleep waits between close-confirm polls. Nil means time.Sleep.
	Sleep func(time.Duration)
	// PollInterval separates close-confirm polls. Non-positive means
	// 50 milliseconds.
	PollInterval time.Duration
	// ServerAdmission reports the live server attestation the attach
	// handler binds to the request generation (the P3-A wiring: a
	// fresh decoy admission refuses here). Nil refuses attach only.
	// The context carries the operation deadline and the caller's
	// cancellation: a cooperative admission returns promptly when
	// it fires, and the attach entry additionally bounds the wait
	// itself, so even an adapter that ignores cancellation cannot
	// hold the request past its deadline or commit late.
	ServerAdmission func(ctx context.Context) (RealmAdmission, error)
	// LocalHostID is the AX host identity the after-restore
	// composition compares the winning lease holder against. Empty
	// cannot prove a local win, so the composition parks or offers
	// remote attach only, never resumes.
	LocalHostID string
	// LeaseRefresh attempts the §4.2 mesh lease refresh
	// (after-restore step 2) under a timeout context. Nil skips the
	// attempt; a failed attempt also falls back to local knowledge.
	// Only a successful refresh verifies: the fallback decides
	// unverified, so it parks instead of resuming.
	LeaseRefresh func(ctx context.Context, sessionID string) (RefreshWinner, error)
	// MeshRefreshTimeout bounds one refresh attempt. Non-positive
	// means DefaultMeshRefreshTimeout. The bound is enforced, not
	// advisory: the entry returns at the deadline even against an
	// adapter that ignores cancellation, and a late answer never
	// verifies.
	MeshRefreshTimeout time.Duration
	// Hooks observes receipts and effects. Nil disables observation.
	Hooks *terminstance.EngineHooks
	// RefreshHooks arms the race boundary of the refresh select.
	// Nil in production.
	RefreshHooks *RefreshHooks
	// barrierMu serializes the input-closure ordering contract: the
	// quiesce report commit, the boundary report commit, and the
	// attach writable-receipt commit (barrier recheck, client
	// receipt, returned vector, advisory memory) are mutually
	// exclusive, so whichever lands first wins and no writable
	// receipt commits after a durable closure report. The initial
	// attach fast-path check and the server admission run outside
	// the lock (admission may block); only the recheck-to-vector
	// section holds it. The quiesce engine closure runs inside the
	// lock (its tmux effects return immediately); the boundary wait
	// runs outside (it blocks on the wrapper signal) and only its
	// report commit takes the lock. Non-reentrant: hooks firing
	// inside a locked section must not call Execute. Attach-to-attach
	// overlap admission additionally uses AttachStore's per-instance
	// OS lock across processes. Attach-versus-quiesce ordering remains
	// in-process: concurrent ax processes serialize only through the
	// report files, not through this mutex (bound B42). The vector handoff the contract
	// covers ends at vector construction: a caller that execs a
	// pre-quiescence vector after quiescence commits holds a
	// pre-quiescence authorization, which quiesce does not revoke.
	barrierMu sync.Mutex
}

// RefreshHooks arms the race boundary of the mesh lease refresh:
// BeforeSelect runs after the adapter launches and before the entry
// selects over the answer and the deadline. A nil hook is a no-op.
// Tests stall past the bound to force the both-ready race
// deterministically: without the stall the deadline arm always wins
// and the answer-path expiry check never triggers.
type RefreshHooks struct {
	BeforeSelect func()
}

// DefaultMeshRefreshTimeout bounds the after-restore mesh lease
// refresh when the Lifecycle leaves it unset: five seconds, after
// which the branch proceeds on local knowledge instead of blocking
// forever.
const DefaultMeshRefreshTimeout = 5 * time.Second

// RefreshWinner is one mesh lease-refresh answer: the winning lease
// tuple and its holder host.
type RefreshWinner struct {
	LeaseID      string
	Epoch        uint64
	HolderHostID string
}

// OpRequest is one lifecycle operation request: the operation name,
// its closed body, the presented source state, the admitted
// capabilities, and the fencing material (for terminate-stale only).
type OpRequest struct {
	Operation            string
	Body                 []byte
	Source               string
	Admitted             terminalbackend.Admitted
	Presented            fencing.PresentedToken
	Winner               sessrepo.LeaseSummary
	HasWinner            bool
	ForceRecovery        bool
	DiagnosticsPreserved bool
}

// AttachOutcome is the attach success payload: the client mirror, the
// host-local descriptor, the authorized input boolean, and the
// evidence. The result input boolean always equals the request and the
// authorization; the landed CheckAttachResult proves it on every call.
type AttachOutcome struct {
	ClientMirrorID  string
	Descriptor      string
	InputAuthorized bool
	EvidenceIDs     []string
}

// OpOutcome is one executed operation: exactly one of Mutation,
// Status, or Attach is set, plus the per-operation success members.
// Mutation is set on every executed mutating path — success and
// coded failure alike, mirroring the engine — and nil only when the
// request was malformed before any result existed.
type OpOutcome struct {
	Operation              terminalbackend.Operation
	Mutation               *terminstance.Result
	Status                 *terminstance.StatusReport
	Attach                 *AttachOutcome
	Argv                   []string
	Binding                *termbind.Binding
	Descriptor             *string
	PriorBindingID         string
	RestoredParked         bool
	TargetClosed           bool
	ProcessClosed          bool
	StoreClosed            bool
	QuiescenceGeneration   string
	InputClosedAt          string
	SafeBoundaryEvidenceID string
	BoundaryObservedAt     string
}

// Execute runs one lifecycle operation to its outcome. Refusals
// precede side effects: malformed operations, unknown backends, and
// unsafe sockets refuse before any store or tmux interaction. The
// socket custody (before-connect rejection) and length refusal run on
// every operation, because every operation connects.
func (lc *Lifecycle) Execute(ctx context.Context, req OpRequest) (OpOutcome, error) {
	empty := OpOutcome{}
	if lc == nil || lc.Runner == nil || lc.Receipts == nil || lc.Bindings == nil ||
		lc.Attach == nil || lc.States == nil ||
		lc.CurrentLease == nil || lc.CurrentGeneration == nil || lc.Now == nil {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "lifecycle dependencies"}
	}
	operation, err := terminalbackend.ParseOperation(req.Operation)
	if err != nil {
		return empty, err
	}
	if _, err := DirectivesFor(operation); err != nil {
		return empty, err
	}
	socket := SocketPath(lc.RuntimeDir)
	if err := CheckSocketLength(socket, lc.Platform); err != nil {
		return empty, err
	}
	if err := CheckSocketCustody(socket, lc.Root, lc.Platform); err != nil {
		return empty, err
	}
	switch operation {
	case terminalbackend.OperationCreate:
		return lc.executeCreate(ctx, req, socket)
	case terminalbackend.OperationAttach:
		return lc.executeAttach(ctx, req, socket)
	case terminalbackend.OperationStatus:
		return lc.executeStatus(ctx, req, socket)
	case terminalbackend.OperationQuiesceInput:
		return lc.executeEngineOp(ctx, req, socket, operation)
	case terminalbackend.OperationWaitSafeBoundary:
		return lc.executeEngineOp(ctx, req, socket, operation)
	case terminalbackend.OperationRequestStop:
		return lc.executeEngineOp(ctx, req, socket, operation)
	case terminalbackend.OperationTerminateStale:
		return lc.executeTerminate(ctx, req, socket)
	case terminalbackend.OperationRestore:
		return lc.executeRestore(ctx, req, socket)
	default:
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "lifecycle operation"}
	}
}

// engineFor builds the per-call landed engine over a request-scoped
// tmux backend. The engine is a plain struct: constructing one per
// call keeps request identity out of shared state.
func (lc *Lifecycle) engineFor(reqBackend *backend) *terminstance.Engine {
	return &terminstance.Engine{
		Store:             lc.Receipts,
		Backend:           reqBackend,
		CurrentLease:      lc.CurrentLease,
		CurrentGeneration: lc.CurrentGeneration,
		Now:               lc.Now,
		Hooks:             lc.Hooks,
	}
}

// requestBackend builds the request-scoped tmux backend over the
// lifecycle stores and the admitted set.
func (lc *Lifecycle) requestBackend(req OpRequest, socket, session, instance string) *backend {
	sleep := lc.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	poll := lc.PollInterval
	if poll <= 0 {
		poll = 50 * time.Millisecond
	}
	return &backend{
		runner:     lc.Runner,
		runtimeDir: lc.RuntimeDir,
		socket:     socket,
		sessionID:  session,
		instanceID: instance,
		now:        lc.Now,
		sleep:      sleep,
		poll:       poll,
		admitted:   req.Admitted,
		bindings:   lc.Bindings,
		attach:     lc.Attach,
		states:     lc.States,
		killExit:   -1,
	}
}

// recordAfter records the adopted after-state in the AX-side memory on
// success paths. The memory is advisory everywhere: status
// reconciles the live probe against it, but attach admission never
// consults it — only the incarnation-scoped closure proof decides
// admission, so a stale or lost memory record can neither reopen a
// closed barrier nor close an open one. On the barrier successes
// (quiesce-input and wait-safe-boundary) the caller still checks the
// returned error and fails closed, keeping the status memory
// faithful to the closure the operation just proved.
func (lc *Lifecycle) recordAfter(instance string, after terminalbackend.InstanceState) error {
	if lc.States == nil {
		return nil
	}
	return lc.States.Record(instance, after)
}

// recordFailure records the after-state of a failed operation. A
// failure with committed effects and an uncertain tail (After
// unavailable) overwrites with the honest unknown: the prior record
// no longer describes the instance. A pure refusal proves nothing —
// its After echoes the caller-carried source — so it installs that
// echo only when the memory holds no record at all (letting the
// uncertain-resume path reconcile a first-operation failure) and
// never overwrites an authoritative record with stale caller state.
// Failure reporting can never reopen a closed barrier: attach
// admission never consults this memory, and a refusal either
// preserves the prior record, installs a first echo, or records the
// non-admitting unknown.
func (lc *Lifecycle) recordFailure(instance string, after terminalbackend.InstanceState) {
	if lc.States == nil {
		return
	}
	if after == terminalbackend.StateUnavailable {
		_ = lc.recordAfter(instance, after)
		return
	}
	_, found, err := lc.States.Lookup(instance)
	if err != nil || found {
		return
	}
	_ = lc.recordAfter(instance, after)
}

// mutationResult assembles a lifecycle result with the engine's
// buildResult contract: repeated request identities, before/after
// states, bytewise-sorted effects and evidence, and one disposition.
func mutationResult(mctx terminstance.MutationContext, before, after terminalbackend.InstanceState, effects []terminalbackend.SideEffect, evidence []string, disposition terminstance.RetryDisposition) terminstance.Result {
	return terminstance.Result{
		OperationID:           mctx.OperationID,
		SessionID:             mctx.SessionID,
		TerminalInstanceID:    mctx.TerminalInstanceID,
		TerminalBackendID:     mctx.TerminalBackendID,
		ImplementationVersion: mctx.ImplementationVersion,
		ProtocolVersion:       mctx.ProtocolVersion,
		BackendGeneration:     mctx.BackendGeneration,
		Before:                before,
		After:                 after,
		Effects:               sortedEffects(effects),
		EvidenceIDs:           sortedEvidence(evidence),
		Disposition:           disposition,
	}
}

// checkLifecycleBackend admits exactly the tmux backend identity: this
// lifecycle executes for ax.tmux and no other backend. A request
// naming any other backend refuses the failed local precondition,
// never executes against the wrong backend.
func checkLifecycleBackend(backendID string) error {
	if backendID != terminalbackend.BuiltinTmux {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "lifecycle backend"}
	}
	return nil
}

// checkBindingAgreement binds the create binding to its mutation
// context: session, instance, backend, and generation must all agree.
// Any disagreement refuses the failed local precondition: the binding
// names another creation.
func checkBindingAgreement(body CreateBody) error {
	if body.Binding.SessionID != body.Context.SessionID ||
		body.Binding.TerminalInstanceID != body.Context.TerminalInstanceID ||
		body.Binding.TerminalBackendID != body.Context.TerminalBackendID ||
		body.Binding.BackendGeneration != body.Context.BackendGeneration {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "create binding agreement"}
	}
	return nil
}

// rowError normalizes a backend- or store-reported code against one
// operation's allowed set: an in-row code propagates with its report,
// while an outside-set code is itself refused as the protocol error —
// the engine's CheckErrorAllowed arm, mirrored for the operations the
// engine scope leaves out.
func rowError(operation terminalbackend.Operation, err error) error {
	code := backendErrorCode(err)
	if code == "" {
		return err
	}
	if checkErr := terminalbackend.CheckErrorAllowed(string(operation), code); checkErr != nil {
		return &Error{Code: terminalbackend.CodeProtocolError, Detail: "operation error vocabulary"}
	}
	return err
}

// formatTimestamp renders Now in the scalar timestamp grammar: UTC
// RFC3339 with millisecond precision, always carrying the fractional
// digits the grammar requires.
func formatTimestamp(now time.Time) string {
	return now.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}

// executeCreate runs the full create body: closed-shape admission,
// backend and transport gates, binding agreement, then the landed
// engine with the tmux backend. The result echoes the carried binding
// and, exactly when interactive, the unbound attach descriptor.
func (lc *Lifecycle) executeCreate(ctx context.Context, req OpRequest, socket string) (OpOutcome, error) {
	empty := OpOutcome{Operation: terminalbackend.OperationCreate}
	body, err := parseCreateBody(req.Body)
	if err != nil {
		return empty, err
	}
	if err := checkLifecycleBackend(body.Context.TerminalBackendID); err != nil {
		return empty, err
	}
	if body.Transport == string(terminalbackend.TransportThirdPartyRelay) {
		return empty, &Error{Code: terminalbackend.CodeUnauthorized, Detail: "create relay transport"}
	}
	if err := checkBindingAgreement(body); err != nil {
		return empty, err
	}
	reqBackend := lc.requestBackend(req, socket, body.Context.SessionID, body.Context.TerminalInstanceID)
	reqBackend.operation = terminalbackend.OperationCreate
	reqBackend.bootstrapID = body.BootstrapOperationID
	reqBackend.bindingID = body.Binding.BindingID
	reqBackend.bindingDoc = append([]byte(nil), body.BindingRaw...)
	_, completedBefore, err := lc.Receipts.Completed(body.Context.IdempotencyKey)
	if err != nil {
		return empty, &Error{Code: terminstance.CodeProcessFailed, Detail: "idempotency store failure"}
	}
	result, err := lc.engineFor(reqBackend).ExecuteMutating(ctx, terminalbackend.OperationCreate, body.Context, terminstance.Params{
		BootstrapOperationID: body.BootstrapOperationID,
		Interactive:          body.Interactive,
	}, terminalbackend.InstanceState(req.Source), req.Admitted)
	outcome := OpOutcome{Operation: terminalbackend.OperationCreate, Binding: &body.Binding}
	if result.OperationID != "" {
		outcome.Mutation = &result
		if err != nil {
			lc.recordFailure(body.Context.TerminalInstanceID, result.After)
		} else {
			_ = lc.recordAfter(body.Context.TerminalInstanceID, result.After)
		}
	}
	if err != nil {
		return outcome, err
	}
	if !completedBefore {
		// A fresh create success opens a new incarnation: prior
		// closure reports are superseded, so the new
		// incarnation's input starts open. Replays skip the
		// rotation — replaying an old success must not erase a
		// newer closure — and a failed rotation fails the
		// operation closed: the incarnation keeps its prior
		// value, prior proofs stay valid, and recovery is a
		// fresh create or restore.
		if err := lc.States.RecordIncarnation(body.Context.TerminalInstanceID, body.Context.IdempotencyKey); err != nil {
			return outcome, &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "create incarnation image"}
		}
	}
	if body.Interactive {
		descriptor := createDescriptor(socket, body.Context.TerminalInstanceID)
		outcome.Descriptor = &descriptor
	}
	return outcome, nil
}

// executeStatus runs one status query through the landed engine with
// the tmux backend: no receipt, no side effects, no authorization.
func (lc *Lifecycle) executeStatus(ctx context.Context, req OpRequest, socket string) (OpOutcome, error) {
	empty := OpOutcome{Operation: terminalbackend.OperationStatus}
	body, err := terminstance.ParseStatusBody(req.Body)
	if err != nil {
		return empty, err
	}
	if err := checkLifecycleBackend(body.TerminalBackendID); err != nil {
		return empty, err
	}
	instance := body.TerminalInstanceID
	if !body.HasTerminalInstanceID {
		instance = ""
	}
	reqBackend := lc.requestBackend(req, socket, body.SessionID, instance)
	reqBackend.operation = terminalbackend.OperationStatus
	report, err := lc.engineFor(reqBackend).ExecuteStatus(ctx, body, terminalbackend.InstanceState(req.Source), req.Admitted)
	if err != nil {
		return empty, err
	}
	return OpOutcome{Operation: terminalbackend.OperationStatus, Status: &report}, nil
}

// executeEngineOp runs one quiesce-input, wait-safe-boundary, or
// request-stop body through the landed engine with the tmux backend,
// adding the per-operation success members the engine's receipt scope
// leaves out.
func (lc *Lifecycle) executeEngineOp(ctx context.Context, req OpRequest, socket string, operation terminalbackend.Operation) (OpOutcome, error) {
	empty := OpOutcome{Operation: operation}
	mctx, params, err := terminstance.ParseOperationBody(string(operation), req.Body)
	if err != nil {
		return empty, err
	}
	if err := checkLifecycleBackend(mctx.TerminalBackendID); err != nil {
		return empty, err
	}
	reqBackend := lc.requestBackend(req, socket, mctx.SessionID, mctx.TerminalInstanceID)
	reqBackend.operation = operation
	switch operation {
	case terminalbackend.OperationWaitSafeBoundary:
		reqBackend.quiescence = params.QuiescenceGeneration
		reqBackend.channel = BoundaryChannelPrefix + params.QuiescenceGeneration
		reqBackend.proofKind = params.ProviderProofKind
		waitCtx, cancel := lc.boundWait(ctx, mctx, params.TimeoutMs)
		defer cancel()
		reqBackend.waitCtx = waitCtx
	case terminalbackend.OperationRequestStop:
		// The poll bound is the graceful wait: the lesser of the
		// request deadline and the graceful timeout. The
		// escalation revalidation below answers to the
		// operation deadline instead — a graceful timeout
		// escalates, an operation deadline passed refuses —
		// so the two bounds stay distinct. The post-escalation
		// re-confirmation observes under the operation
		// deadline as well: the graceful budget is exhausted
		// by then, and reusing it would cancel the fresh
		// probe before a cancellation-honouring executor
		// could run it.
		reqBackend.deadline = lc.lesserDeadline(mctx, params.GracefulTimeoutMs)
		if opDeadline, err := mctx.Deadline.Time(); err == nil {
			reqBackend.opDeadline = opDeadline
		}
	}
	if operation == terminalbackend.OperationRequestStop || operation == terminalbackend.OperationQuiesceInput {
		// Within-effect revalidation for the paths whose
		// effects issue a destructive command after a wait:
		// the stop escalation after its poll, the quiesce
		// detach after its lock round trip. The boundary
		// wait's tail is observation-only and create's
		// effects are single commands with no wait, so
		// those paths carry no revalidation.
		opDeadline, _ := mctx.Deadline.Time()
		reqBackend.recheck = lc.effectRecheck(operation, mctx, engineKind(operation), opDeadline)
	}
	if operation == terminalbackend.OperationQuiesceInput {
		// Ordering contract, quiesce side: hold the barrier lock
		// across the engine input closure and the report commit
		// below, so no attach receipt commits between the tmux
		// closure effects and their durable proof. The lock
		// releases at return. The boundary wait takes no such
		// span (it blocks on the wrapper signal); only its
		// report commit serializes, in its case branch. The
		// acquisition itself honors the operation deadline
		// and the caller's cancellation like every barrier
		// user: a deadline cancels waiting, not a committed
		// effect.
		waitCtx, waitCancel := lc.waitContext(ctx, mctx)
		defer waitCancel()
		release, err := lc.acquireBarrier(waitCtx)
		if err != nil {
			return empty, &Error{Code: rowTimeoutCode(operation), Detail: "operation deadline"}
		}
		defer release()
	}
	result, err := lc.engineFor(reqBackend).ExecuteMutating(ctx, operation, mctx, params, terminalbackend.InstanceState(req.Source), req.Admitted)
	outcome := OpOutcome{Operation: operation}
	if result.OperationID != "" {
		outcome.Mutation = &result
		if err != nil {
			// Failure results record only the honest
			// unknown: a pure refusal writes nothing
			// instead of overwriting an authoritative
			// record with stale caller state.
			lc.recordFailure(mctx.TerminalInstanceID, result.After)
		} else if operation != terminalbackend.OperationQuiesceInput && operation != terminalbackend.OperationWaitSafeBoundary {
			// Non-barrier successes record best-effort:
			// the memory is advisory and a lost write
			// never admits input.
			_ = lc.recordAfter(mctx.TerminalInstanceID, result.After)
		}
		// Barrier successes record below, checked, after the
		// outcome report commits: the report carries the closure
		// time the retry replays and the incarnation-scoped
		// proof the attach entry enforces, and the memory
		// carries the closure status reconciles against.
	}
	if err != nil {
		return outcome, err
	}
	switch operation {
	case terminalbackend.OperationQuiesceInput:
		outcome.QuiescenceGeneration = params.QuiescenceGeneration
		closedAt, err := lc.reportInputClosedAt(mctx.TerminalInstanceID, mctx.IdempotencyKey)
		if err != nil {
			return outcome, err
		}
		if err := lc.recordAfter(mctx.TerminalInstanceID, result.After); err != nil {
			return outcome, &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "quiesce state image"}
		}
		outcome.InputClosedAt = closedAt
	case terminalbackend.OperationWaitSafeBoundary:
		// The boundary transition carries exactly one effect, and the
		// engine evidences every committed effect exactly once, so a
		// successful boundary result carries exactly one evidence ID.
		outcome.SafeBoundaryEvidenceID = result.EvidenceIDs[0]
		// Ordering contract, boundary side: the wait above ran
		// outside the lock; only the report commit below
		// serializes with attach receipts. The lock releases at
		// return. The acquisition honors the operation
		// deadline and the caller's cancellation like every
		// barrier user; a wait refused here reports the row
		// timeout against the already-committed proof.
		waitCtx, waitCancel := lc.waitContext(ctx, mctx)
		defer waitCancel()
		release, err := lc.acquireBarrier(waitCtx)
		if err != nil {
			return outcome, &Error{Code: rowTimeoutCode(operation), Detail: "operation deadline"}
		}
		defer release()
		observedAt, err := lc.reportBoundaryObservedAt(mctx.TerminalInstanceID, mctx.IdempotencyKey)
		if err != nil {
			return outcome, err
		}
		if err := lc.recordAfter(mctx.TerminalInstanceID, result.After); err != nil {
			return outcome, &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "boundary state image"}
		}
		outcome.BoundaryObservedAt = observedAt
	case terminalbackend.OperationRequestStop:
		outcome.ProcessClosed = true
		outcome.StoreClosed = true
	}
	return outcome, nil
}

// reportInputClosedAt replays the recorded input-closed time for one
// quiesce key, recording the current time on first success. The
// first-success record binds the current incarnation, so the report
// proves the barrier exactly for the incarnation it closed. A
// failed or forged lookup is corruption, never absence: the retry
// refuses the integrity failure instead of manufacturing a fresh
// closure time for an already-closed input. A record failure
// refuses the same way: the effect is committed but its time is
// unprovable, so the retry re-records rather than reporting an
// unrecorded time.
func (lc *Lifecycle) reportInputClosedAt(instanceID, key string) (string, error) {
	stored, found, err := lc.States.LookupOutcome(key)
	if err != nil {
		return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "quiesce outcome image"}
	}
	if found {
		if stored.InputClosedAt == "" {
			return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "quiesce outcome image"}
		}
		return stored.InputClosedAt, nil
	}
	incarnation, _, err := lc.States.LookupIncarnation(instanceID)
	if err != nil {
		return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "quiesce outcome image"}
	}
	rendered := formatTimestamp(lc.Now())
	if err := lc.States.RecordOutcome(key, OperationOutcome{Operation: string(terminalbackend.OperationQuiesceInput), InputClosedAt: rendered, Incarnation: incarnation}); err != nil {
		return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "quiesce outcome image"}
	}
	return rendered, nil
}

// reportBoundaryObservedAt replays the recorded boundary-observed
// time for one boundary key, recording the current time on first
// success, with the same corrupt-read and record-failure discipline
// as the quiesce report: no failed read becomes fresh evidence.
// The first-success record binds the current incarnation like the
// quiesce report.
func (lc *Lifecycle) reportBoundaryObservedAt(instanceID, key string) (string, error) {
	stored, found, err := lc.States.LookupOutcome(key)
	if err != nil {
		return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "boundary outcome image"}
	}
	if found {
		if stored.BoundaryObservedAt == "" {
			return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "boundary outcome image"}
		}
		return stored.BoundaryObservedAt, nil
	}
	incarnation, _, err := lc.States.LookupIncarnation(instanceID)
	if err != nil {
		return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "boundary outcome image"}
	}
	rendered := formatTimestamp(lc.Now())
	if err := lc.States.RecordOutcome(key, OperationOutcome{Operation: string(terminalbackend.OperationWaitSafeBoundary), BoundaryObservedAt: rendered, Incarnation: incarnation}); err != nil {
		return "", &Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "boundary outcome image"}
	}
	return rendered, nil
}

// boundWait bounds the safe-boundary wait by the lesser of the request
// deadline and the timeout: the wait context carries the minimum, and
// an already-invalid deadline falls back to the timeout alone (the
// engine refuses the malformed deadline before any effect runs, so the
// fallback never binds). The caller cancels the context.
func (lc *Lifecycle) boundWait(ctx context.Context, mctx terminstance.MutationContext, timeoutMs uint64) (context.Context, context.CancelFunc) {
	bound := lc.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	if deadline, err := mctx.Deadline.Time(); err == nil && deadline.Before(bound) {
		bound = deadline
	}
	return context.WithDeadline(ctx, bound)
}

// lesserDeadline computes the lesser of the request deadline and the
// graceful timeout for the stop close-confirm poll. An invalid
// deadline falls back the same way as the boundary wait.
func (lc *Lifecycle) lesserDeadline(mctx terminstance.MutationContext, timeoutMs uint64) time.Time {
	bound := lc.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	if deadline, err := mctx.Deadline.Time(); err == nil && deadline.Before(bound) {
		return deadline
	}
	return bound
}

// engineKind resolves one engine operation's AX authorization kind
// through the landed transition table, mirroring the engine's own
// mapping: the table names the kind and this function honors it. A
// drift resolves to the empty kind, which the authorization check
// refuses exactly like the engine's entry gate would.
func engineKind(operation terminalbackend.Operation) terminstance.AuthorizationKind {
	landed, err := terminalbackend.TransitionAuthorization(string(operation))
	if err != nil {
		return ""
	}
	kind, err := terminstance.ParseAuthorizationKind(landed)
	if err != nil {
		return ""
	}
	return kind
}

// effectRecheck builds the within-effect authorization revalidation
// for one engine operation: the same three per-effect gates the
// landed loop applies before every effect — the operation deadline,
// the AX authorization over the live lease, and the live server
// generation — evaluated at the destructive command boundary that
// follows a wait. The timeout code is the operation row's own
// member. The authorization gate is the landed CheckAuthorization
// over the same request authorization, kind, lease, and clock the
// engine passes; nothing here re-implements it.
func (lc *Lifecycle) effectRecheck(operation terminalbackend.Operation, mctx terminstance.MutationContext, kind terminstance.AuthorizationKind, opDeadline time.Time) func() error {
	timeoutCode := rowTimeoutCode(operation)
	return func() error {
		if !lc.Now().Before(opDeadline) {
			return &terminstance.Error{Code: timeoutCode, Detail: "operation deadline"}
		}
		if err := terminstance.CheckAuthorization(mctx.Authorization, kind, lc.CurrentLease(), lc.Now()); err != nil {
			return err
		}
		if lc.CurrentGeneration() != mctx.BackendGeneration {
			return &terminstance.Error{Code: terminalbackend.CodeStaleGeneration, Detail: "backend_generation stale"}
		}
		return nil
	}
}

// waitUntil bounds one wait by an absolute operation deadline
// measured on the lifecycle clock, combined with the caller's
// context. The duration comes from the request deadline minus the
// lifecycle now, so fake-clock requests wait out their remaining
// real time and already-passed deadlines refuse at once; the
// caller cancels the context. A deadline cancels waiting, not a
// committed effect.
func (lc *Lifecycle) waitUntil(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, deadline.Sub(lc.Now()))
}

// waitContext bounds one engine-operation wait by the mutation
// context deadline. An unparseable deadline leaves the caller's
// context unmodified: the operation body parse refuses it before
// any wait runs, so the unmodified wait never binds an effect,
// and a dedicated probe proves the path still returns promptly
// under contention.
func (lc *Lifecycle) waitContext(ctx context.Context, mctx terminstance.MutationContext) (context.Context, context.CancelFunc) {
	deadline, err := mctx.Deadline.Time()
	if err != nil {
		return ctx, func() {}
	}
	return lc.waitUntil(ctx, deadline)
}

// acquireBarrier acquires the barrier mutex unless the wait context
// fires first, in which case no lock is held and the waiter reports
// the context error. A waiter that loses the race releases the
// mutex as soon as its orphan acquisition lands, so a timed-out
// waiter never steals the barrier from the next holder and never
// leaves a goroutine behind once the holder releases. Every
// barrier user — attach, the quiesce span, the boundary report
// commit — acquires through this contract.
func (lc *Lifecycle) acquireBarrier(ctx context.Context) (func(), error) {
	// An already-fired wait never starts its waiter: the
	// request is refused deterministically instead of racing the
	// orphan acquisition against the fired context.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	acquired := make(chan struct{})
	go func() {
		lc.barrierMu.Lock()
		close(acquired)
	}()
	select {
	case <-acquired:
		return lc.barrierMu.Unlock, nil
	case <-ctx.Done():
		go func() {
			<-acquired
			lc.barrierMu.Unlock()
		}()
		return nil, ctx.Err()
	}
}
