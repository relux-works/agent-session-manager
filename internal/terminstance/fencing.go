package terminstance

import (
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// ObserveFencing projects one AX fencing observation onto the Terminal
// Instance lifecycle. It is the ONLY entry that produces stale_fenced:
// no backend operation targets that state (the landed CheckTransition
// proves it and the engine never synthesizes it), only AX fencing
// observation does.
//
// The verdict is read entirely from the landed fencing projection: the
// presented local token is authorized against the observation through
// fencing.Authorize with the restore operation, and a ParkStaleOwner
// park — the older-epoch arm and the same-epoch lease-mismatch arm
// alike — moves the local observation to stale_fenced. The operation
// must be launch-class (restore or activation): the input, mutation
// and checkpoint entries refuse a stale token instead of parking it,
// so non-live sources would surface a hard refusal instead of the
// stale_owner park vocabulary (live sources still fence through the
// verdict fallback either way). Restore is chosen over activation
// because it gates terminal restore, this story's domain; the two are
// equivalent on the stale arms.
//
// A stale token under a REMOTE winner fences too: the landed gate parks
// remote_owner before it compares tuples ("the operative fact beats the
// credential"), so the first verdict cannot distinguish a stale token
// from the winning token under a foreign winner. The second question
// asks the landed gate whether the presented token is stale RELATIVE TO
// THE WINNER by authorizing from the winner's own host (LocalHostID set
// to Winner.HolderHostID) and reading the park: stale_owner fences,
// anything else — including the winning token authorizing cleanly —
// leaves the state with the original remote park. Both questions run
// inside the landed Authorize; no local tuple comparison exists here.
//
// A stale token refused by the gate's GRANT precondition still fences:
// the grant arms (no grant, no clock reading, lapsed grant, unusable
// policy) fire before the direction/tuple arms, so without a second
// verdict a cold force-takeover incarnation — one that can never hold
// a live grant, because VerifyFencingToken refuses its stale epoch
// and grants never survive a controller restart — would never reach
// stale_fenced. The landed StaleRelativeToWinner verdict decides the
// direction/tuple arms over the verified well-formed winner without
// the grant precondition, and a decided stale verdict fences from a
// live incarnation exactly like the gate's own stale park. The
// relative question needs no such fallback: it carries the direct
// observation's grant facts unchanged and the grant arms do not read
// LocalHostID, so a direct remote park implies the relative question
// passes the grant arms.
//
// Only a live incarnation fences: creating, parked, active and
// quiescing move to stale_fenced, while absent and stopped (no
// incarnation exists for terminate-stale to target), stale_fenced
// itself (already fenced: the transition is idempotent) and
// unavailable (the observation is unknown and no verdict is projected
// over unknown) surface the refusal with no transition. Every other
// outcome leaves the state untouched: an authorized token reports no
// transition, and any refusal or park the verdict cannot decide
// (malformed, session mismatch, no winner, malformed winner,
// unverified, ambiguous, failed handoff) or decides not-stale
// (winning tuple, future epoch) surfaces its own error with no
// transition, because only proven staleness fences. A nil error with
// transitioned false is the authorized case.
//
// This function performs no lease comparison of its own: the tuple,
// grant, expiry and direction arms all run inside the landed Authorize
// and the landed StaleRelativeToWinner, so no second fencing model
// exists here. It reads Winner.LeaseID and Winner.Epoch only through
// those landed verdicts, never Winner.Checkpoint: lease records are
// immutable content-addressed blobs whose checkpoint member is null
// for every epoch-1 lease, so keying on it would be vacuous for every
// session that never changed hands.
func ObserveFencing(current terminalbackend.InstanceState, presented fencing.PresentedToken, observation fencing.Observation) (terminalbackend.InstanceState, bool, error) {
	if _, err := terminalbackend.ParseInstanceState(string(current)); err != nil {
		return current, false, wrapLanded(err)
	}
	_, authErr := fencing.Authorize(fencing.OperationRestore, presented, observation)
	if authErr == nil {
		return current, false, nil
	}
	reason, _, parked := fencing.ParkDetails(authErr)
	if parked && reason == fencing.ParkRemoteOwner {
		return observeRemoteWinner(current, presented, observation, authErr)
	}
	if parked && reason == fencing.ParkStaleOwner {
		return fenceLive(current, authErr)
	}
	if stale, decided := fencing.StaleRelativeToWinner(presented, observation); decided && stale {
		return fenceLive(current, authErr)
	}
	return current, false, authErr
}

// observeRemoteWinner resolves a remote-winner park: the winner is on
// another host and the presented token may be stale relative to it (the
// force-takeover case) or the winning token itself observed from afar.
// The landed gate decides from the winner's own host; only a stale
// verdict fences, and only from a live incarnation.
func observeRemoteWinner(current terminalbackend.InstanceState, presented fencing.PresentedToken, observation fencing.Observation, authErr error) (terminalbackend.InstanceState, bool, error) {
	relative := observation
	relative.LocalHostID = observation.Winner.HolderHostID
	_, relativeErr := fencing.Authorize(fencing.OperationRestore, presented, relative)
	if relativeErr == nil {
		return current, false, authErr
	}
	if relativeReason, _, relativeParked := fencing.ParkDetails(relativeErr); relativeParked && relativeReason == fencing.ParkStaleOwner {
		return fenceLive(current, authErr)
	}
	return current, false, authErr
}

// fenceLive moves a live incarnation to stale_fenced and leaves every
// other source untouched with the refusal or park that proved
// staleness. Absent and stopped name no incarnation, stale_fenced is
// already fenced, and unavailable is unknown: none of them transitions.
func fenceLive(current terminalbackend.InstanceState, authErr error) (terminalbackend.InstanceState, bool, error) {
	switch current {
	case terminalbackend.StateCreating, terminalbackend.StateParked,
		terminalbackend.StateActive, terminalbackend.StateQuiescing:
		return terminalbackend.StateStaleFenced, true, nil
	default:
		return current, false, authErr
	}
}
