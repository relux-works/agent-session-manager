package fencing

import (
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// AuthorizeTerminateStale gates the terminal backend terminate-stale
// operation: terminate a process only for explicit force recovery
// after preserving diagnostics, and only when the target is fenced
// (its token is not the winning tuple). The caller needs no winning
// grant of its own: the stale-process host runs this during force
// recovery. A nil error authorizes the termination; any refusal names
// the Section 4.C terminate-stale row class
// (local_precondition_failed) or the fencing class that applies.
//
// Arm order is precedence: malformed calls die first, then the
// authority to judge staleness (an established, well-formed winner),
// then the explicit force recovery confirmation with its preserved
// diagnostics, then the target binding, then the live-owner guard.
func AuthorizeTerminateStale(presented PresentedToken, winner sessrepo.LeaseSummary, hasWinner bool, forceRecovery bool, diagnosticsPreserved bool) error {
	if _, err := scalar.ParseUUIDv7(presented.SessionID); err != nil {
		return refuse(ErrInvalidArguments, "terminate-stale target session %q is not a UUIDv7: %v", presented.SessionID, err)
	}
	if presented.Epoch == 0 {
		return refuse(ErrInvalidArguments, "terminate-stale target carries epoch 0, want an epoch at or above 1")
	}
	if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil {
		return refuse(ErrInvalidArguments, "terminate-stale target lease %q is not a UUIDv4: %v", presented.LeaseID, err)
	}
	if !hasWinner {
		return refuse(ErrLocalPreconditionFailed, "terminate-stale cannot prove its target is fenced without an established winning lease")
	}
	if winner.Epoch == 0 || !validLeaseID(winner.LeaseID) {
		return refuse(ErrInvalidArguments, "terminate-stale observation carries no well-formed winning lease")
	}
	if !forceRecovery {
		return refuse(ErrLocalPreconditionFailed, "terminate-stale requires explicit force recovery")
	}
	if !diagnosticsPreserved {
		return refuse(ErrLocalPreconditionFailed, "terminate-stale requires preserved diagnostics before termination")
	}
	if presented.SessionID != winner.SessionID {
		return refuse(ErrLeaseConflict, "terminate-stale target for session %s is not fenced by the winning lease for session %s", presented.SessionID, winner.SessionID)
	}
	if presented.Epoch == winner.Epoch && presented.LeaseID == winner.LeaseID {
		return refuse(ErrLocalPreconditionFailed, "terminate-stale must not terminate the live winning owner at epoch %d", winner.Epoch)
	}
	return nil
}
