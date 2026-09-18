package termbind

import (
	"context"
	"errors"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// RecoveryOutcome is the lost-create recovery verdict: proven absence, the
// one recorded child, or unavailable when the effect cannot be disproven.
type RecoveryOutcome string

// Recovery outcomes.
const (
	RecoveryAbsent      RecoveryOutcome = "absent"
	RecoveryChild       RecoveryOutcome = "child"
	RecoveryUnavailable RecoveryOutcome = "unavailable"
)

// RecoveryRequest carries one lost-create recovery query: the bootstrap
// pair, the caller-derived bootstrap window bit, the backend tuple and
// host-local generation for the status read, and the status deadline.
type RecoveryRequest struct {
	SessionID            string
	BootstrapOperationID string
	// WindowOpen is true while the bootstrap window is open: the sessstate
	// fold names no newest published checkpoint. The caller derives it
	// from the fold (the axpane precedent for caller-derived facts);
	// recovery tests close the loop by folding a real chain.
	WindowOpen                 bool
	BackendID                  string
	ImplementationVersion      string
	ProtocolVersion            string
	Generation                 string
	IncludeProviderObservation bool
	DeadlineAt                 string
}

// RecoveryDeps carries the recovery inputs: the durable bootstrap binding
// store, the status engine, and the admitted capabilities for the status
// provider conditional.
type RecoveryDeps struct {
	Bindings *axpane.Store
	Engine   *terminstance.Engine
	Admitted terminalbackend.Admitted
}

// RecoveryVerdict is the recovery decision: the outcome, the recorded
// binding and adopted backend state for the child, and the retry
// disposition. Absence and the child both carry replay_same (run or
// re-run the one operation); unavailable carries status_first.
type RecoveryVerdict struct {
	Outcome    RecoveryOutcome
	Binding    axpane.Binding
	HasBinding bool
	State      terminalbackend.InstanceState
	Retry      terminstance.RetryDisposition
}

// RecoverCreate recovers a lost create result by bootstrap operation ID
// from the durable (session_id, bootstrap_operation_id) binding plus one
// status read. It is read-only: it binds nothing and emits nothing.
//
//   - A changed operation inside the bootstrap window refuses
//     idempotency_mismatch; outside the window a new pair proceeds.
//   - No binding plus a status that proves absent yields absence with
//     replay_same: creating under this pair is safe.
//   - A binding plus an identity-matching status yields the ONE recorded
//     child with replay_same: reattach to it, or (when the backend shows
//     absent) re-run the same operation for the same recorded instance,
//     never a second child.
//   - Every other shape yields unavailable with status_first: a
//     contradiction (binding without a backend match), the creating
//     interim, a backend-reported unavailable, or a live backend instance
//     under an unbound pair (unadoptable: it cannot be proven ours).
//
// A binding read failure or a status read failure is an error, never
// absence: unknown is not absent.
func RecoverCreate(ctx context.Context, deps RecoveryDeps, request RecoveryRequest) (RecoveryVerdict, error) {
	empty := RecoveryVerdict{}
	if deps.Bindings == nil {
		return empty, errors.New("recovery carries no bootstrap binding store")
	}
	if deps.Engine == nil {
		return empty, errors.New("recovery carries no status engine")
	}
	binding, found, err := deps.Bindings.Lookup(request.SessionID, request.BootstrapOperationID)
	if err != nil {
		return empty, err
	}
	if !found {
		anchor, anchored, err := deps.Bindings.Status(request.SessionID)
		if err != nil {
			return empty, err
		}
		if anchored && anchor.OperationID != request.BootstrapOperationID && request.WindowOpen {
			return empty, &terminalbackend.Error{Code: terminalbackend.CodeIdempotencyMismatch, Detail: "bootstrap binding"}
		}
	}
	body, err := recoveryStatusBody(request, binding, found)
	if err != nil {
		return empty, err
	}
	report, err := deps.Engine.ExecuteStatus(ctx, body, "", deps.Admitted)
	if err != nil {
		return empty, err
	}
	if !found {
		if report.IdentityMatch && report.State == terminalbackend.StateAbsent {
			return RecoveryVerdict{Outcome: RecoveryAbsent, State: report.State, Retry: terminstance.DispositionReplaySame}, nil
		}
		return RecoveryVerdict{Outcome: RecoveryUnavailable, State: report.State, Retry: terminstance.DispositionStatusFirst}, nil
	}
	if !report.IdentityMatch {
		return RecoveryVerdict{Outcome: RecoveryUnavailable, State: report.State, Retry: terminstance.DispositionStatusFirst}, nil
	}
	switch report.State {
	case terminalbackend.StateCreating, terminalbackend.StateUnavailable:
		return RecoveryVerdict{Outcome: RecoveryUnavailable, State: report.State, Retry: terminstance.DispositionStatusFirst}, nil
	default:
		return RecoveryVerdict{Outcome: RecoveryChild, Binding: binding, HasBinding: true, State: report.State, Retry: terminstance.DispositionReplaySame}, nil
	}
}

// recoveryStatusBody builds the recovery status read: the exact-instance
// lookup over the recorded binding when the pair is bound, else the
// session-scoped lookup that can prove absence. The host-local generation
// passes through the landed bound gate; the deadline parses through the
// scalar owner.
func recoveryStatusBody(request RecoveryRequest, binding axpane.Binding, found bool) (terminstance.StatusBody, error) {
	empty := terminstance.StatusBody{}
	deadline, err := scalar.ParseTimestamp(request.DeadlineAt)
	if err != nil {
		return empty, err
	}
	body := terminstance.StatusBody{
		SessionID:                  request.SessionID,
		TerminalBackendID:          request.BackendID,
		ImplementationVersion:      request.ImplementationVersion,
		ProtocolVersion:            request.ProtocolVersion,
		IncludeProviderObservation: request.IncludeProviderObservation,
		Deadline:                   deadline,
	}
	if !found {
		return body, nil
	}
	if _, err := terminalbackend.GenerationDigest(request.Generation); err != nil {
		return empty, err
	}
	body.TerminalInstanceID = binding.TerminalInstanceID
	body.HasTerminalInstanceID = true
	body.BackendGeneration = request.Generation
	body.HasBackendGeneration = true
	return body, nil
}
