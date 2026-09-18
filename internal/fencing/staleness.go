package fencing

import (
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// StaleRelativeToWinner reports whether the presented fencing token is
// stale relative to the observation's winning lease: an older epoch,
// or the winning epoch under a losing lease ID. It decides ONLY the
// direction/tuple arms over a verified, well-formed winner, WITHOUT
// the grant precondition: a grant AUTHORIZES, it does not make an
// incarnation less stale, and grants are machine-local transient state
// (never persisted), so after a controller restart the prior owner has
// none while the observation itself still proves staleness (Section
// 13.7 cold force takeover: the prior owner becomes stale and MUST
// stop accepting input when it learns the winner).
//
// Direction is not staleness either: a losing tuple under a remote
// winner is stale relative to the winner on any host, the same fact
// the relative question answers through the gate when a grant exists.
// LocalHostID is therefore never read here.
//
// The ownership preconditions mirror Authorize's pre-grant arms
// exactly: the presented token must be well-formed and name this
// session, the winner must be established and well-formed, and the
// observation must be verified, unambiguous, and not a failed handoff.
// Anything else is undecidable (decided false, stale false): the
// caller surfaces the gate's own refusal or park instead of projecting
// a verdict over it. In particular a failed handoff keeps its
// failed_handoff park — the handoff arm is ownership history, not a
// grant precondition, so it stays outside this verdict.
//
// The verdict mints no authority: stale true never authorizes input,
// mutation, or activation. It is a projection input, consumed by the
// Terminal Instance lifecycle to decide stale_fenced. Wherever
// Authorize reaches its own tuple arms, the two agree (pinned by
// TestStalenessAgreesWithGateOnHotPath); no second staleness rule
// exists here.
func StaleRelativeToWinner(presented PresentedToken, observation Observation) (stale, decided bool) {
	if _, err := scalar.ParseUUIDv7(presented.SessionID); err != nil {
		return false, false
	}
	if presented.Epoch == 0 {
		return false, false
	}
	if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil {
		return false, false
	}
	if presented.SessionID != observation.SessionID {
		return false, false
	}
	if !observation.HasWinner {
		return false, false
	}
	if observation.Winner.Epoch == 0 || !validLeaseID(observation.Winner.LeaseID) || !validHostID(observation.Winner.HolderHostID) {
		return false, false
	}
	if !observation.Verified {
		return false, false
	}
	if observation.Ambiguous {
		return false, false
	}
	if observation.HandoffFailed {
		return false, false
	}
	if presented.Epoch != observation.Winner.Epoch {
		return presented.Epoch < observation.Winner.Epoch, true
	}
	return presented.LeaseID != observation.Winner.LeaseID, true
}
