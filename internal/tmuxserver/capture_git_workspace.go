package tmuxserver

import (
	"context"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/gitsnap"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// GitWorkspaceCaptureRequest binds one complete Git workspace assembly to an
// exact Terminal Instance status query. A running instance must supply the
// landed wait-safe-boundary body for a provider proof; a stopped instance must
// not supply one. ScratchRoot is part of Assembly and must be caller-owned.
type GitWorkspaceCaptureRequest struct {
	StatusBody   []byte
	BoundaryBody []byte
	Admitted     terminalbackend.Admitted
	Runner       gitsnap.AssemblyRunner
	Assembly     gitsnap.AssemblyOptions
}

// GitWorkspaceCaptureOutcome is returned only after the assembly and closing
// status read both succeed. The safe-boundary ID is empty for a stopped source.
type GitWorkspaceCaptureOutcome struct {
	Assembly               *gitsnap.ProvisionalAssembly
	InitialState           terminalbackend.InstanceState
	FinalState             terminalbackend.InstanceState
	SafeBoundaryEvidenceID string
}

// CaptureGitWorkspace is the production composition entry for complete Git
// transfer-object assembly. It admits only an exact status observation of a
// stopped instance or a current-incarnation quiescence with a provider
// safe-boundary receipt. It never opens input or changes the lifecycle state.
func (lc *Lifecycle) CaptureGitWorkspace(ctx context.Context, request GitWorkspaceCaptureRequest) (*GitWorkspaceCaptureOutcome, error) {
	if lc == nil {
		return nil, captureUnavailable("lifecycle is required")
	}
	statusBody, err := terminstance.ParseStatusBody(request.StatusBody)
	if err != nil {
		return nil, err
	}
	// ParseStatusBody admits the instance ID and backend generation as a pair,
	// so the instance-presence check is the complete capture identity gate.
	if !statusBody.HasTerminalInstanceID {
		return nil, captureUnavailable("capture requires an exact Terminal Instance identity")
	}
	statusRequest := OpRequest{Operation: "status", Body: request.StatusBody, Admitted: request.Admitted}
	initial, err := lc.Execute(ctx, statusRequest)
	if err != nil {
		return nil, err
	}
	// Lifecycle.Execute's status operation returns a report on every nil-error path.
	if !initial.Status.IdentityMatch {
		return nil, captureUnavailable("exact Terminal Instance status did not match")
	}
	if !captureStateAllowed(initial.Status.State) {
		switch initial.Status.State {
		case terminalbackend.StateAbsent:
			return nil, captureUnavailable(`capture is unsupported from Terminal Instance state "absent"`)
		case terminalbackend.StateParked:
			return nil, captureUnavailable(`capture is unsupported from Terminal Instance state "parked"`)
		case terminalbackend.StateActive:
			return nil, captureUnavailable(`capture is unsupported from Terminal Instance state "active"`)
		case terminalbackend.StateStaleFenced:
			return nil, captureUnavailable(`capture is unsupported from Terminal Instance state "stale_fenced"`)
		case terminalbackend.StateUnavailable:
			return nil, captureUnavailable(`capture is unsupported from Terminal Instance state "unavailable"`)
		default:
			return nil, fmt.Errorf("capture observed unknown Terminal Instance state %q", initial.Status.State)
		}
	}
	if request.Runner == nil {
		return nil, captureUnavailable("Git assembly runner is required")
	}

	var safeBoundary string
	var incarnation string
	if initial.Status.State == terminalbackend.StateStopped {
		if len(request.BoundaryBody) != 0 {
			return nil, captureUnavailable("stopped capture must not carry a boundary operation")
		}
		var found bool
		incarnation, found, err = lc.States.LookupIncarnation(statusBody.TerminalInstanceID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, captureUnavailable("stopped capture has no current incarnation record")
		}
	} else {
		if len(request.BoundaryBody) == 0 {
			return nil, captureUnavailable("quiescing capture requires a provider safe-boundary operation")
		}
		boundaryContext, boundaryParams, err := terminstance.ParseOperationBody("wait-safe-boundary", request.BoundaryBody)
		if err != nil {
			return nil, err
		}
		if !sameCaptureIdentity(statusBody, boundaryContext) {
			return nil, captureUnavailable("boundary operation identity differs from exact status")
		}
		if boundaryParams.ProviderProofKind != "provider_quiescence" && boundaryParams.ProviderProofKind != "provider_process_exit" {
			return nil, captureUnavailable("checkpoint boundary alone does not prove provider quiescence")
		}
		var found bool
		incarnation, found, err = lc.States.LookupIncarnation(statusBody.TerminalInstanceID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, captureUnavailable("quiescing capture has no current incarnation record")
		}
		quiesceKey, err := terminstance.KeySegments(terminalbackend.OperationQuiesceInput, boundaryContext, terminstance.Params{QuiescenceGeneration: boundaryParams.QuiescenceGeneration})
		if err != nil {
			return nil, err
		}
		quiesce, found, err := lc.States.LookupOutcome(quiesceKey)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, captureUnavailable("current-incarnation input-closure receipt is missing")
		}
		if found {
			if quiesce.Operation != "quiesce-input" {
				return nil, captureUnavailable("input-closure receipt has the wrong operation")
			}
			if _, err := scalar.ParseTimestamp(quiesce.InputClosedAt); err != nil {
				return nil, captureUnavailable("input-closure receipt has an invalid observation timestamp")
			}
			if quiesce.Incarnation != incarnation {
				return nil, captureUnavailable("input-closure receipt belongs to another incarnation")
			}
		}
		boundaryKey, err := terminstance.KeySegments(terminalbackend.OperationWaitSafeBoundary, boundaryContext, boundaryParams)
		if err != nil {
			return nil, err
		}
		boundary, err := lc.Execute(ctx, OpRequest{
			Operation: "wait-safe-boundary", Body: request.BoundaryBody,
			Source: "quiescing", Admitted: request.Admitted,
		})
		if err != nil {
			return nil, err
		}
		safeBoundary = boundary.SafeBoundaryEvidenceID
		if _, err := scalar.ParseDigest(safeBoundary); err != nil {
			return nil, fmt.Errorf("capture boundary evidence is malformed: %w", err)
		}
		boundaryReceipt, found, err := lc.States.LookupOutcome(boundaryKey)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("successful provider-boundary operation has no persisted receipt")
		}
		if boundaryReceipt.Operation != "wait-safe-boundary" {
			return nil, captureUnavailable("provider-boundary receipt has the wrong operation")
		}
		if boundaryReceipt.Incarnation != incarnation {
			return nil, captureUnavailable("provider-boundary receipt belongs to another incarnation")
		}
		if _, err := scalar.ParseTimestamp(boundaryReceipt.BoundaryObservedAt); err != nil {
			return nil, fmt.Errorf("capture provider-boundary receipt is malformed: %w", err)
		}
	}

	assembly, err := gitsnap.AssembleProvisional(ctx, request.Runner, request.Assembly)
	if err != nil {
		return nil, err
	}
	final, err := lc.Execute(ctx, statusRequest)
	if err != nil {
		return nil, err
	}
	// As above, a nil error from the status owner includes its report.
	if !final.Status.IdentityMatch {
		return nil, captureUnavailable("closing Terminal Instance status did not match")
	}
	currentIncarnation, found, err := lc.States.LookupIncarnation(statusBody.TerminalInstanceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, captureUnavailable("Terminal Instance incarnation record disappeared during capture")
	}
	if found && currentIncarnation != incarnation {
		return nil, captureUnavailable("Terminal Instance incarnation changed during capture")
	}
	if !captureStateAllowed(final.Status.State) {
		return nil, captureUnavailable("held capture left quiescing or stopped state")
	}
	if initial.Status.State == terminalbackend.StateStopped {
		if final.Status.State != terminalbackend.StateStopped {
			return nil, captureUnavailable("stopped capture did not remain stopped")
		}
	} else if final.Status.State == terminalbackend.StateQuiescing {
		proven, err := lc.States.QuiesceBarrierProven(statusBody.TerminalInstanceID)
		if err != nil {
			return nil, err
		}
		if !proven {
			return nil, captureUnavailable("input-closure barrier no longer proves capture hold")
		}
	}
	return &GitWorkspaceCaptureOutcome{
		Assembly: assembly, InitialState: initial.Status.State,
		FinalState: final.Status.State, SafeBoundaryEvidenceID: safeBoundary,
	}, nil
}

func captureStateAllowed(state terminalbackend.InstanceState) bool {
	return state == terminalbackend.StateQuiescing || state == terminalbackend.StateStopped
}

func sameCaptureIdentity(status terminstance.StatusBody, mutation terminstance.MutationContext) bool {
	return status.SessionID == mutation.SessionID &&
		status.TerminalInstanceID == mutation.TerminalInstanceID &&
		status.TerminalBackendID == mutation.TerminalBackendID &&
		status.ImplementationVersion == mutation.ImplementationVersion &&
		status.ProtocolVersion == mutation.ProtocolVersion &&
		status.BackendGeneration == mutation.BackendGeneration
}

func captureUnavailable(detail string) error {
	return &gitsnap.Refusal{Code: "capability_unavailable", Gate: gitsnap.GateCapturePolicy.Name, Detail: detail}
}
