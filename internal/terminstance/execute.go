package terminstance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Backend is the modeled terminal backend caller: side-effect execution
// and status observation. No tmux/ConPTY process control and no provider
// process exist behind it; tests model success, coded row refusals,
// uncoded failures and crashes through this interface.
type Backend interface {
	// PerformEffect executes one side effect and reports its evidence
	// ID. A nil error proves the commit; a *Error with a row code
	// proves the backend refused before committing; any other error
	// proves nothing, not even non-commit.
	PerformEffect(ctx context.Context, effect terminalbackend.SideEffect) (string, error)
	// ObserveStatus reports the backend's current observation for one
	// status body. An error is unknown, never absent.
	ObserveStatus(ctx context.Context, body StatusBody) (StatusObservation, error)
}

// StatusObservation is the backend-reported status observation: the
// reported state and match verdict, the reported identity tuple the
// engine compares against the query, and the reported capability and
// last-operation facts the landed CheckStatusResult validates.
type StatusObservation struct {
	State             terminalbackend.InstanceState
	IdentityMatch     bool
	WrapperPresent    bool
	ProviderPresent   *bool
	Attachable        bool
	LastOperationID   *string
	LastEffect        *terminalbackend.SideEffect
	EvidenceIDs       []string
	ProviderRequested bool
	ProviderEvidenced bool
	AttachEvidenced   bool
	SessionID         string
	InstanceID        string
	BackendID         string
	ImplVersion       string
	ProtoVersion      string
	Generation        string
}

// StatusReport is the validated status outcome: the adopted observation.
// Status performs no transition bookkeeping of its own — no receipt, no
// effects, no before/after — so the report carries the single adopted
// state verbatim.
type StatusReport struct {
	State           terminalbackend.InstanceState
	IdentityMatch   bool
	WrapperPresent  bool
	ProviderPresent *bool
	Attachable      bool
	LastOperationID *string
	LastEffect      *terminalbackend.SideEffect
	EvidenceIDs     []string
}

// EngineHooks arms the engine seams tests observe: AfterReceipt fires
// after the durable receipt commits and before the first side effect,
// carrying the interim state (creating for create, the source
// otherwise); BeforeEffect and AfterEffect bracket each committed
// attempt. Hooks observe only: they return nothing and cannot alter
// the outcome.
type EngineHooks struct {
	AfterReceipt func(operation terminalbackend.Operation, key string, interim terminalbackend.InstanceState)
	BeforeEffect func(effect terminalbackend.SideEffect)
	AfterEffect  func(effect terminalbackend.SideEffect)
}

// Engine executes the §4.C operation rows. Store is the durable receipt
// table; Backend the modeled backend; CurrentLease and CurrentGeneration
// re-read the winning lease and the validated binding generation at
// every check point (function values, never snapshots, so a rotation
// between entry and any effect is caught); Now is the clock. Hooks may
// be nil.
type Engine struct {
	Store             *ReceiptStore
	Backend           Backend
	CurrentLease      func() LeaseView
	CurrentGeneration func() string
	Now               func() time.Time
	Hooks             *EngineHooks
}

// InterimState is the observable state between the durable idempotency
// receipt and the first side effect: creating for create, the source
// state for every other engine operation. creating is entered only by
// create, only after its receipt and before its first listed effect.
func InterimState(operation terminalbackend.Operation, source terminalbackend.InstanceState) terminalbackend.InstanceState {
	if operation == terminalbackend.OperationCreate {
		return terminalbackend.StateCreating
	}
	return source
}

// kindFor maps an engine operation to its AXAuthorization kind through the
// landed transition-table authorization column: the table names the kind
// and this function honors only the two the engine executes (create,
// control). The force_stale and restore kinds parse but no engine
// operation requests them — terminate-stale and restore recovery belongs
// to the story's final leaf — and attach and none are never
// AXAuthorization kinds. A drift between the table and this mapping fails
// TestKindForAgreesWithLandedTable; no second table is restated here.
func kindFor(operation terminalbackend.Operation) (AuthorizationKind, bool) {
	landed, err := terminalbackend.TransitionAuthorization(string(operation))
	if err != nil {
		return "", false
	}
	switch AuthorizationKind(landed) {
	case AuthorizationCreate:
		return AuthorizationCreate, true
	case AuthorizationControl:
		return AuthorizationControl, true
	default:
		return "", false
	}
}

// rowTimeoutCode is the §4.C timeout vocabulary per engine row: the wait
// row times out quiesce_timeout and the stop row stop_timeout, while
// create and quiesce-input time out terminal_backend_timeout.
//
// An engine deadline cancel before any attempt reports replay_same on
// every row, including stop: nothing was attempted, so the identical
// retry is safe. That differs deliberately from a backend-REPORTED
// stop_timeout, which maps through DispositionFor (status_first even
// uncommitted): a graceful wait that ran and timed out may have left
// the provider mid-shutdown, while a cancel before the first attempt
// proves nothing happened. The two directions are pinned together by
// TestExecuteEntryDeadlineCodes and the stop rows of
// TestExecuteBackendRowErrorSets.
func rowTimeoutCode(operation terminalbackend.Operation) string {
	switch operation {
	case terminalbackend.OperationWaitSafeBoundary:
		return CodeQuiesceTimeout
	case terminalbackend.OperationRequestStop:
		return CodeStopTimeout
	default:
		return CodeTimeout
	}
}

// engineScope admits exactly the operations this engine executes: the
// status, quiesce-input, wait-safe-boundary and request-stop rows plus
// the create receipt rule. Status has its own entry; attach, restore,
// terminate-stale, manifest and probe are refused here.
func engineScope(operation terminalbackend.Operation) error {
	switch operation {
	case terminalbackend.OperationCreate, terminalbackend.OperationQuiesceInput,
		terminalbackend.OperationWaitSafeBoundary, terminalbackend.OperationRequestStop:
		return nil
	default:
		return refuse(CodeProtocolError, "operation lifecycle scope")
	}
}

// checkCapabilityConditional evaluates the §4.C conditional capability
// dependencies at the operation call site, the half the landed
// CheckOperation explicitly leaves to the lifecycle owner: headless
// creation for non-interactive create, and safe-boundary observation
// (plus provider-process observation for provider proof kinds) for
// wait-safe-boundary. Quiesce-input and request-stop carry no conditional
// beyond conferral — input_quiescence and graceful_stop are their sole
// conferring capabilities, so the landed CheckOperation decides those
// rows alone and no second arm restates it. The credential-realm
// conditional is a stated bound: credential need is not modeled on this
// path.
func checkCapabilityConditional(operation terminalbackend.Operation, params Params, admitted terminalbackend.Admitted) error {
	switch operation {
	case terminalbackend.OperationCreate:
		if !params.Interactive && !admitted.Has("headless_creation") {
			return refuse(CodeCapabilityUnproven, "operation capability conditional")
		}
		return nil
	case terminalbackend.OperationQuiesceInput, terminalbackend.OperationRequestStop:
		return nil
	case terminalbackend.OperationWaitSafeBoundary:
		if !admitted.Has("safe_boundary_observation") {
			return refuse(CodeCapabilityUnproven, "operation capability conditional")
		}
		if RequiresProviderObservation(params.ProviderProofKind) && !admitted.Has("provider_process_observation") {
			return refuse(CodeCapabilityUnproven, "operation capability conditional")
		}
		return nil
	default:
		return refuse(CodeProtocolError, "operation lifecycle scope")
	}
}

// ExecuteMutating executes one create, quiesce-input, wait-safe-boundary
// or request-stop request from the source state. A nil error is success
// (after is the transition target, disposition replay_same: re-sending
// replays). A non-nil execution error carries the row code with after
// and disposition per the §4.C error rules: an error before the first
// side effect restores the source state, while an error after a
// committed effect whose result cannot be proven moves the local
// observation to unavailable with status_first (new_authorization when
// the failure names auth drift) and never claims absent. Malformed
// requests (unknown operation, unparseable source or context) return an
// empty result with the error, because a result repeating unvalidated
// identities would echo data the gates refused. Authorization and
// generation are rechecked immediately before each side effect, not
// once at entry; a deadline cancels waiting, never a committed effect.
//
// Emission bound: the §4.C row sets govern backend REPORTS, which the
// engine routes through CheckErrorAllowed (an outside-set code is
// refused as a protocol error). The engine's own AX-side seam verdicts
// are outside that rule: an uncoded effect outcome classifies
// terminal_backend_process_failed, and an uncertain retry (receipt
// without completion) whose status reconciliation proves nothing refuses
// terminal_backend_unavailable, both with after unavailable and
// status_first per the row recovery columns ("uncertainty requires
// status", "an error after an effect whose result cannot be proven").
// Those codes are the leaf's contract for failures the backend never
// coded, pinned literally below.
//
// Recovery: a bound key without its completion reconciles through the
// production status entry before it refuses. When status proves the
// presented source state — nothing happened — the same operation
// continues under the same receipt and completes it, so the §4.C
// recovery columns ("then same-key retry only if not closed", "Create
// is idempotent across controller crash and lost result") execute
// instead of refusing forever. When status proves anything else, or
// fails, the retry refuses uncertain: the effects may have committed
// and their evidence cannot be proven, so re-issuing them would risk a
// second allocation. No path binds a second receipt for one key.
func (engine *Engine) ExecuteMutating(ctx context.Context, operation terminalbackend.Operation, context MutationContext, params Params, source terminalbackend.InstanceState, admitted terminalbackend.Admitted) (Result, error) {
	empty := Result{}
	if engine == nil || engine.Store == nil || engine.Backend == nil ||
		engine.CurrentLease == nil || engine.CurrentGeneration == nil || engine.Now == nil {
		return empty, refuse(CodeProtocolError, "engine unavailable")
	}
	if _, err := terminalbackend.ParseOperation(string(operation)); err != nil {
		return empty, wrapLanded(err)
	}
	if err := engineScope(operation); err != nil {
		return empty, err
	}
	if _, err := terminalbackend.ParseInstanceState(string(source)); err != nil {
		return empty, wrapLanded(err)
	}
	if err := CheckMutationContext(operation, context, params); err != nil {
		return empty, err
	}
	if err := terminalbackend.CheckOperation(string(operation), admitted); err != nil {
		failure := wrapLanded(err)
		return buildResult(context, source, source, nil, nil, DispositionFor(failure.Code, false)), failure
	}
	if err := checkCapabilityConditional(operation, params, admitted); err != nil {
		failure := err.(*Error)
		return buildResult(context, source, source, nil, nil, DispositionFor(failure.Code, false)), failure
	}
	target, effects, err := terminalbackend.CheckTransition(string(operation), string(source), params.Interactive)
	if err != nil {
		failure := wrapLanded(err)
		return buildResult(context, source, source, nil, nil, DispositionFor(failure.Code, false)), failure
	}
	kind, _ := kindFor(operation)
	if err := CheckAuthorization(context.Authorization, kind, engine.CurrentLease(), engine.Now()); err != nil {
		failure := err.(*Error)
		return buildResult(context, source, source, nil, nil, DispositionFor(failure.Code, false)), failure
	}
	if engine.CurrentGeneration() != context.BackendGeneration {
		failure := refuse(CodeStaleGeneration, "backend_generation stale")
		return buildResult(context, source, source, nil, nil, DispositionFor(failure.Code, false)), failure
	}
	deadline, err := context.Deadline.Time()
	if err != nil {
		return empty, refuse(CodeProtocolError, "mutation context deadline")
	}
	if !engine.Now().Before(deadline) {
		failure := refuse(rowTimeoutCode(operation), "operation deadline")
		return buildResult(context, source, source, nil, nil, DispositionReplaySame), failure
	}
	key := context.IdempotencyKey
	_, replayed, err := engine.Store.Bind(key, operation, context.OperationID)
	if err != nil {
		failure, ok := err.(*Error)
		if !ok {
			// Uncoded store failure: the Lookup below decides what is
			// durable. Nothing linked (a pre-link stage, write, sync or
			// link failure — no side effect ran and no receipt stands)
			// restores the source state, so the identical retry is
			// safe. A standing receipt (a post-link hook or
			// directory-sync failure) or an unreadable store is
			// genuinely uncertain.
			if _, found, lookupErr := engine.Store.Lookup(key); lookupErr == nil && !found {
				prelink := refuse(CodeProcessFailed, "idempotency store failure")
				return buildResult(context, source, source, nil, nil, DispositionFor(prelink.Code, false)), prelink
			}
			uncoded := refuse(CodeProcessFailed, "idempotency store failure")
			return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionStatusFirst), uncoded
		}
		return buildResult(context, source, source, nil, nil, DispositionFor(failure.Code, false)), failure
	}
	if replayed {
		stored, found, err := engine.Store.Completed(key)
		if err != nil {
			uncoded := refuse(CodeProcessFailed, "idempotency store failure")
			return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionStatusFirst), uncoded
		}
		if !found {
			return engine.resumeUncertain(ctx, operation, context, source, admitted, target, effects, deadline, kind)
		}
		var result Result
		if err := json.Unmarshal(stored, &result); err != nil {
			failure := refuse(CodeIntegrityFailure, "idempotency result image")
			return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionStatusFirst), failure
		}
		if err := CheckResult(context, source, result); err != nil {
			failure, ok := err.(*Error)
			if !ok {
				uncoded := refuse(CodeIntegrityFailure, "idempotency result image")
				return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionStatusFirst), uncoded
			}
			return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionFor(failure.Code, true)), failure
		}
		return result, nil
	}
	if engine.Hooks != nil && engine.Hooks.AfterReceipt != nil {
		engine.Hooks.AfterReceipt(operation, key, InterimState(operation, source))
	}
	return engine.runEffects(ctx, operation, context, source, target, effects, deadline, kind)
}

// resumeUncertain reconciles a bound key without its completion through
// the production status entry, executing the §4.C recovery columns
// ("uncertainty requires status" on create and quiesce-input,
// "uncertainty uses status" on restore, "lost result requires status,
// then same-key retry only if not closed" on request-stop, "Create is
// idempotent on (session_id, bootstrap_operation_id) across controller
// crash and lost result"). The reconciliation query is the exact
// instance the request names — Session, instance, backend,
// implementation, protocol and generation — validated by the same
// ExecuteStatus arms every caller-driven status passes, so a lying or
// drifting backend cannot authorize a resumption.
//
// When status proves the presented source state, nothing happened and
// the same operation continues under the same receipt: the receipt is
// already bound, so no second receipt is created, and the resumed run
// completes it exactly once. For wait-safe-boundary the source is the
// target, so a proven quiescing resumes and the resumed observation
// supersedes any lost one; at most one completion exists per key either
// way. Re-issue under the same key relies on the backend's own
// key binding (§4.C: "AX and backend durably bind the pair before the
// first child side effect"; §13.13 safe_retry): the engine never issues
// the same logical operation under a second key, so no second process
// is allocated.
//
// When status proves anything else — the target (the effects may have
// committed but their evidence cannot be proven), a contradictory state,
// or nothing at all (the read fails) — the retry refuses uncertain
// with status_first. Re-issuing from a non-source observation would
// violate the transition matrix, and inventing evidence would fabricate
// a result; the caller reconciles through its own status read instead.
func (engine *Engine) resumeUncertain(ctx context.Context, operation terminalbackend.Operation, context MutationContext, source terminalbackend.InstanceState, admitted terminalbackend.Admitted, target terminalbackend.InstanceState, effects []terminalbackend.SideEffect, deadline time.Time, kind AuthorizationKind) (Result, error) {
	body := StatusBody{
		SessionID:                  context.SessionID,
		TerminalInstanceID:         context.TerminalInstanceID,
		HasTerminalInstanceID:      true,
		TerminalBackendID:          context.TerminalBackendID,
		ImplementationVersion:      context.ImplementationVersion,
		ProtocolVersion:            context.ProtocolVersion,
		BackendGeneration:          context.BackendGeneration,
		HasBackendGeneration:       true,
		IncludeProviderObservation: false,
		Deadline:                   context.Deadline,
	}
	report, err := engine.ExecuteStatus(ctx, body, source, admitted)
	if err != nil {
		failure := refuse(CodeUnavailable, "idempotency result uncertain")
		return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionStatusFirst), failure
	}
	if report.State != source {
		failure := refuse(CodeUnavailable, "idempotency result uncertain")
		return buildResult(context, source, terminalbackend.StateUnavailable, nil, nil, DispositionStatusFirst), failure
	}
	if engine.Hooks != nil && engine.Hooks.AfterReceipt != nil {
		engine.Hooks.AfterReceipt(operation, context.IdempotencyKey, InterimState(operation, source))
	}
	return engine.runEffects(ctx, operation, context, source, target, effects, deadline, kind)
}

// runEffects performs the transition's side effects under an already
// bound receipt — freshly bound by ExecuteMutating or resumed by
// resumeUncertain after status proved the source — with the per-effect
// deadline, authorization and generation gates, then completes the key
// exactly once. Fresh and resumed runs share this loop, so the recheck
// and error rules exist once, never as a resumed twin.
func (engine *Engine) runEffects(ctx context.Context, operation terminalbackend.Operation, context MutationContext, source, target terminalbackend.InstanceState, effects []terminalbackend.SideEffect, deadline time.Time, kind AuthorizationKind) (Result, error) {
	key := context.IdempotencyKey
	committed := false
	evidence := []string(nil)
	performed := []terminalbackend.SideEffect(nil)
	for _, effect := range effects {
		if !engine.Now().Before(deadline) {
			failure := refuse(rowTimeoutCode(operation), "operation deadline")
			if committed {
				return buildResult(context, source, terminalbackend.StateUnavailable, performed, evidence, DispositionStatusFirst), failure
			}
			return buildResult(context, source, source, performed, evidence, DispositionReplaySame), failure
		}
		if err := CheckAuthorization(context.Authorization, kind, engine.CurrentLease(), engine.Now()); err != nil {
			failure := err.(*Error)
			after := source
			if committed {
				after = terminalbackend.StateUnavailable
			}
			return buildResult(context, source, after, performed, evidence, DispositionFor(failure.Code, committed)), failure
		}
		if engine.CurrentGeneration() != context.BackendGeneration {
			failure := refuse(CodeStaleGeneration, "backend_generation stale")
			after := source
			if committed {
				after = terminalbackend.StateUnavailable
			}
			return buildResult(context, source, after, performed, evidence, DispositionFor(failure.Code, committed)), failure
		}
		if engine.Hooks != nil && engine.Hooks.BeforeEffect != nil {
			engine.Hooks.BeforeEffect(effect)
		}
		evidenceID, err := engine.Backend.PerformEffect(ctx, effect)
		if engine.Hooks != nil && engine.Hooks.AfterEffect != nil {
			engine.Hooks.AfterEffect(effect)
		}
		if err != nil {
			code := ErrorCode(err)
			if code == "" {
				failure := refuse(CodeProcessFailed, "backend effect uncertain")
				return buildResult(context, source, terminalbackend.StateUnavailable, performed, evidence, DispositionStatusFirst), failure
			}
			if err := terminalbackend.CheckErrorAllowed(string(operation), code); err != nil {
				failure := refuse(CodeProtocolError, "operation error vocabulary")
				return buildResult(context, source, terminalbackend.StateUnavailable, performed, evidence, DispositionStatusFirst), failure
			}
			after := source
			if committed {
				after = terminalbackend.StateUnavailable
			}
			failure := refuse(code, "backend effect refused")
			return buildResult(context, source, after, performed, evidence, DispositionFor(code, committed)), failure
		}
		if _, err := scalar.ParseDigest(evidenceID); err != nil {
			failure := refuse(CodeIntegrityFailure, "backend effect evidence")
			return buildResult(context, source, terminalbackend.StateUnavailable, performed, evidence, DispositionStatusFirst), failure
		}
		committed = true
		performed = append(performed, effect)
		evidence = append(evidence, evidenceID)
	}
	result := buildResult(context, source, target, performed, evidence, DispositionReplaySame)
	encoded, err := json.Marshal(result)
	if err != nil {
		failure := refuse(CodeIntegrityFailure, "idempotency result image")
		result.Disposition = DispositionStatusFirst
		return result, failure
	}
	if err := engine.Store.Complete(key, encoded); err != nil {
		failure, ok := err.(*Error)
		if !ok {
			uncoded := refuse(CodeProcessFailed, "idempotency store failure")
			result.Disposition = DispositionStatusFirst
			return result, uncoded
		}
		result.Disposition = DispositionStatusFirst
		return result, failure
	}
	return result, nil
}

// ExecuteStatus executes one status query: no receipt, no side effects,
// no authorization and no transition bookkeeping of its own. The
// reported state is adopted verbatim; the engine validates the report
// (the reported match must equal the compared tuple, the landed
// CheckStatusResult must admit it, evidence must be sorted-unique
// digests) and enforces the provider-observation capability conditional.
// A timeout or read failure is unknown, never absent: it returns an
// error with no report.
func (engine *Engine) ExecuteStatus(ctx context.Context, body StatusBody, source terminalbackend.InstanceState, admitted terminalbackend.Admitted) (StatusReport, error) {
	empty := StatusReport{}
	if engine == nil || engine.Backend == nil || engine.Now == nil {
		return empty, refuse(CodeProtocolError, "engine unavailable")
	}
	if source != "" {
		if _, _, err := terminalbackend.CheckTransition(string(terminalbackend.OperationStatus), string(source), false); err != nil {
			return empty, wrapLanded(err)
		}
	}
	deadline, err := body.Deadline.Time()
	if err != nil {
		return empty, refuse(CodeProtocolError, "status body deadline")
	}
	if !engine.Now().Before(deadline) {
		return empty, refuse(CodeTimeout, "operation deadline")
	}
	if body.IncludeProviderObservation && !admitted.Has("provider_process_observation") {
		return empty, refuse(CodeCapabilityUnproven, "operation capability conditional")
	}
	observed, err := engine.Backend.ObserveStatus(ctx, body)
	if err != nil {
		code := ErrorCode(err)
		if code == "" {
			return empty, refuse(CodeUnavailable, "status observation unknown")
		}
		if err := terminalbackend.CheckErrorAllowed(string(terminalbackend.OperationStatus), code); err != nil {
			return empty, refuse(CodeProtocolError, "operation error vocabulary")
		}
		return empty, refuse(code, "status observation unknown")
	}
	if _, err := terminalbackend.ParseInstanceState(string(observed.State)); err != nil {
		return empty, wrapLanded(err)
	}
	computed, err := statusIdentityMatch(body, observed)
	if err != nil {
		return empty, err
	}
	if observed.IdentityMatch != computed {
		return empty, refuse(CodeProtocolError, "status identity binding")
	}
	converted := terminalbackend.StatusResult{
		State:             observed.State,
		IdentityMatch:     observed.IdentityMatch,
		WrapperPresent:    observed.WrapperPresent,
		ProviderPresent:   observed.ProviderPresent,
		Attachable:        observed.Attachable,
		LastOperationID:   observed.LastOperationID,
		LastEffect:        observed.LastEffect,
		ProviderRequested: observed.ProviderRequested,
		ProviderEvidenced: observed.ProviderEvidenced,
		AttachEvidenced:   observed.AttachEvidenced,
	}
	if observed.LastOperationID != nil {
		if _, err := scalar.ParseUUIDv7(*observed.LastOperationID); err != nil {
			return empty, refuse(CodeProtocolError, "status report operation")
		}
	}
	if err := terminalbackend.CheckStatusResult(computed, converted); err != nil {
		return empty, wrapLanded(err)
	}
	if err := checkSortedUniqueStatusEvidence(observed.EvidenceIDs); err != nil {
		return empty, err
	}
	return StatusReport{
		State:           observed.State,
		IdentityMatch:   observed.IdentityMatch,
		WrapperPresent:  observed.WrapperPresent,
		ProviderPresent: observed.ProviderPresent,
		Attachable:      observed.Attachable,
		LastOperationID: observed.LastOperationID,
		LastEffect:      observed.LastEffect,
		EvidenceIDs:     append([]string(nil), observed.EvidenceIDs...),
	}, nil
}

// statusIdentityMatch compares the reported identity tuple against the
// query: exact-instance lookup requires Session, instance, backend,
// implementation, protocol and generation to all match; Session-scoped
// lookup requires the reported session to match and adopts the rest.
// Reported members are shape-validated first, so a malformed report is
// refused instead of compared.
func statusIdentityMatch(body StatusBody, observed StatusObservation) (bool, error) {
	if _, err := scalar.ParseUUIDv7(observed.SessionID); err != nil {
		return false, refuse(CodeProtocolError, "status report session")
	}
	if _, err := terminalbackend.ParseID(observed.BackendID); err != nil {
		return false, wrapLanded(err)
	}
	if err := checkVersionTuple(observed.BackendID, observed.ImplVersion, observed.ProtoVersion); err != nil {
		return false, err
	}
	if observed.SessionID != body.SessionID {
		return false, nil
	}
	if !body.HasTerminalInstanceID {
		return true, nil
	}
	if _, err := scalar.ParseUUIDv7(observed.InstanceID); err != nil {
		return false, refuse(CodeProtocolError, "status report instance")
	}
	if err := checkGenerationBound(observed.Generation); err != nil {
		return false, refuse(CodeProtocolError, "status report generation")
	}
	return observed.InstanceID == body.TerminalInstanceID &&
		observed.BackendID == body.TerminalBackendID &&
		observed.ImplVersion == body.ImplementationVersion &&
		observed.ProtoVersion == body.ProtocolVersion &&
		observed.Generation == body.BackendGeneration, nil
}
