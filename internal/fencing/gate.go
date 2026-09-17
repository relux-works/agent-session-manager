package fencing

import (
	"errors"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

var (
	// ErrLeaseConflict is the Section 15 lease_conflict refusal at exit
	// 10: the presented fencing proof is not the winning authority for
	// this session (same-epoch loser, epoch beyond the winner, foreign
	// session, lapsed grant, or absent/ambiguous/unverified ownership on
	// a non-launch entry), or a forged capability reached a consumer.
	ErrLeaseConflict = errors.New("lease_conflict")
	// ErrNotOwner is the Section 15 not_owner refusal at exit 10: the
	// winning lease holder is a remote host and this entry is not a
	// launch, so there is no parked direction to take.
	ErrNotOwner = errors.New("not_owner")
	// ErrStaleOwner is the Section 15 stale_owner refusal at exit 10:
	// the presented epoch precedes the winning epoch on a non-launch
	// entry. Launch entries park with the stale_owner reason instead.
	ErrStaleOwner = errors.New("stale_owner")
	// ErrParked marks a park decision: the launch is refused and the
	// session must park carrying the session.parked reason and winning
	// lease ID. It always wraps the underlying Section 15 cause as
	// well, so errors.Is reports both the decision and the class.
	ErrParked = errors.New("parked")
	// ErrInvalidArguments is the Section 15 invalid_arguments refusal
	// at exit 2: the gate call itself is malformed (unknown operation,
	// malformed presented token or winning summary, missing grant,
	// missing clock reading, unusable policy, or unknown provider
	// operation). The caller fixes the call, not the ownership.
	ErrInvalidArguments = errors.New("invalid_arguments")
	// ErrLocalPreconditionFailed is the Section 15
	// local_precondition_failed refusal at exit 3, the class the
	// Section 4.C terminate-stale row names: terminate-stale without
	// explicit force recovery, without preserved diagnostics, without
	// an established winner, or against the live winning owner.
	ErrLocalPreconditionFailed = errors.New("local_precondition_failed")
)

// refuse is the single refusal funnel every fencing rejection flows
// through. No production path wraps a sentinel with fmt.Errorf directly.
var refuse = func(sentinel error, format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, arguments...))
}

// ParkReason is one Section 5.2 session.parked reason: the exact
// vocabulary a park decision carries.
type ParkReason string

const (
	// ParkRemoteOwner parks because the winning lease holder is remote.
	ParkRemoteOwner ParkReason = "remote_owner"
	// ParkStaleOwner parks because the local fencing proof loses to the
	// winning lease while the winner is local (a stale cache).
	ParkStaleOwner ParkReason = "stale_owner"
	// ParkRestorePolicy parks because ownership cannot be verified:
	// no winning lease, unverified sync, ambiguous observations, or a
	// token from an epoch beyond the winner.
	ParkRestorePolicy ParkReason = "restore_policy"
	// ParkFailedHandoff parks because the caller's handoff already
	// failed: a losing force lease must never activate.
	ParkFailedHandoff ParkReason = "failed_handoff"
)

// parkRefusal carries a park decision: the session.parked reason, the
// winning lease ID the parked event repeats (empty when no winner is
// established), and the underlying Section 15 cause. It unwraps to both
// ErrParked and the cause.
type parkRefusal struct {
	reason         ParkReason
	winningLeaseID string
	cause          error
}

func (refusal *parkRefusal) Error() string {
	if refusal.winningLeaseID == "" {
		return fmt.Sprintf("%s: park (%s); no winning lease is established", ErrParked, refusal.reason)
	}
	return fmt.Sprintf("%s: park (%s) under winning lease %s: %v", ErrParked, refusal.reason, refusal.winningLeaseID, refusal.cause)
}

func (refusal *parkRefusal) Unwrap() []error {
	return []error{ErrParked, refusal.cause}
}

// IsParked reports whether err is a park decision rather than a hard
// refusal. A parked launch must emit the session.parked vocabulary,
// never start a runtime.
func IsParked(err error) bool {
	return errors.Is(err, ErrParked)
}

// ParkDetails extracts the session.parked vocabulary from a park
// decision: the reason and the winning lease ID (empty when no winner
// is established). It reports false for a hard refusal.
func ParkDetails(err error) (ParkReason, string, bool) {
	var parked *parkRefusal
	if !errors.As(err, &parked) {
		return "", "", false
	}
	return parked.reason, parked.winningLeaseID, true
}

// park builds the launch-class refusal for one ownership condition: the
// parked reason with the Section 15 cause the same condition refuses
// on a non-launch entry.
func park(reason ParkReason, winningLeaseID string, cause error) error {
	return &parkRefusal{reason: reason, winningLeaseID: winningLeaseID, cause: cause}
}

// Operation is one activation-class action the gate authorizes.
type Operation string

const (
	// OperationActivation gates provider activation and launch: only
	// the winning committed lease may create a destination runtime.
	OperationActivation Operation = "activation"
	// OperationInput gates provider input after attach (the
	// quiesce-input and attach direction): the owner revalidates
	// before accepting operator input.
	OperationInput Operation = "input"
	// OperationMutation gates owner-authored mutation: event append
	// and compare-and-swap succession carry the winning epoch.
	OperationMutation Operation = "mutation"
	// OperationCheckpoint gates checkpoint capture admission: a fresh
	// checkpoint is published under the current lease only.
	OperationCheckpoint Operation = "checkpoint"
	// OperationRestore gates terminal restore and the wrapper's first
	// start: ownership is compared before any provider resume.
	OperationRestore Operation = "restore"
)

// validOperation reports whether op names a gated action. Anything
// else is a malformed gate call, never an authorized action.
func validOperation(op Operation) bool {
	switch op {
	case OperationActivation, OperationInput, OperationMutation, OperationCheckpoint, OperationRestore:
		return true
	}
	return false
}

// launchClass reports whether op is a launch-class entry: activation
// and restore park when ownership is remote, ambiguous, unverified,
// or absent, while input, mutation, and checkpoint refuse outright.
func launchClass(op Operation) bool {
	return op == OperationActivation || op == OperationRestore
}

// PresentedToken is the caller's claimed fencing authority: the local
// fencing token the wrapper compares, or the token a takeover, input,
// mutation, or checkpoint caller holds. It is untrusted input: every
// member is grammar-checked before any comparison runs.
type PresentedToken struct {
	SessionID string
	Epoch     uint64
	LeaseID   string
}

// Observation carries the verified durable and sync facts the gate
// decides over. Observe builds it from the repository; callers that
// build it by hand (tests, modeled flows) must supply the same facts.
// The winner is trusted only through its grammar: a malformed winning
// summary refuses rather than authorizing against garbage.
type Observation struct {
	SessionID string
	// Winner is the winning lease summary with its holder. It is
	// meaningful only when HasWinner is set.
	Winner    sessrepo.LeaseSummary
	HasWinner bool
	// LocalHostID is the host asking for authorization. An unknown or
	// empty local identity never equals the winning holder, so it
	// fails closed to the remote arm without a grammar check.
	LocalHostID string
	// Verified reports that a mesh lease refresh proved current
	// lease knowledge. Unverified ownership parks or refuses.
	Verified bool
	// Ambiguous reports conflicting ownership observations (a union
	// rival the local store cannot order). Ambiguity parks or refuses.
	Ambiguous bool
	// HandoffFailed reports that the caller's own handoff already
	// failed (a losing force lease before activation). A failed
	// handoff parks with failed_handoff instead of activating.
	HandoffFailed bool
	// Grant is the process-local authorization derived from a winning
	// lease, meaningful only when HasGrant is set. The gate re-checks
	// its expiry through the sessrepo owner on every call.
	Grant    sessrepo.FencingGrant
	HasGrant bool
	// Policy carries the configured lease refresh interval the
	// expiry arm enforces.
	Policy sessrepo.FencingPolicy
	// Now is the caller's clock reading for the expiry arm. A grant
	// without a clock reading refuses rather than bypassing expiry.
	Now time.Time
}

// ObserveInput carries the caller-reported facts Observe combines with
// the durable winning lease: sync state, handoff outcome, the
// process-local fencing grant with its policy, and the clock reading.
type ObserveInput struct {
	LocalHostID   string
	Verified      bool
	Ambiguous     bool
	HandoffFailed bool
	Grant         sessrepo.FencingGrant
	HasGrant      bool
	Policy        sessrepo.FencingPolicy
	Now           time.Time
}

// Observe loads the winning lease for sessionID and combines it with
// the caller's sync, handoff, grant, and clock facts into the
// Observation Authorize decides over. The winner is the last summary
// of the tuple-ordered ListLeases read, the same derivation
// WinningLease returns, so no second winner rule exists here. A
// session with no leases yet yields HasWinner false without an error:
// absence is data the gate parks or refuses on. A repository failure
// (including an unknown session) propagates unchanged.
func Observe(repository *sessrepo.Repository, sessionID string, input ObserveInput) (Observation, error) {
	leases, err := repository.ListLeases(sessionID)
	if err != nil {
		return Observation{}, err
	}
	observation := Observation{
		SessionID:     sessionID,
		LocalHostID:   input.LocalHostID,
		Verified:      input.Verified,
		Ambiguous:     input.Ambiguous,
		HandoffFailed: input.HandoffFailed,
		Grant:         input.Grant,
		HasGrant:      input.HasGrant,
		Policy:        input.Policy,
		Now:           input.Now,
	}
	if len(leases) > 0 {
		observation.Winner = leases[len(leases)-1]
		observation.HasWinner = true
	}
	return observation, nil
}

// AuthorizeActivation gates provider activation and launch through the
// fencing core.
func AuthorizeActivation(presented PresentedToken, observation Observation) (LeaseToken, error) {
	return Authorize(OperationActivation, presented, observation)
}

// AuthorizeInput gates provider input after attach through the fencing
// core.
func AuthorizeInput(presented PresentedToken, observation Observation) (LeaseToken, error) {
	return Authorize(OperationInput, presented, observation)
}

// AuthorizeMutation gates owner-authored mutation through the fencing
// core.
func AuthorizeMutation(presented PresentedToken, observation Observation) (LeaseToken, error) {
	return Authorize(OperationMutation, presented, observation)
}

// AuthorizeCheckpoint gates checkpoint capture admission through the
// fencing core.
func AuthorizeCheckpoint(presented PresentedToken, observation Observation) (LeaseToken, error) {
	return Authorize(OperationCheckpoint, presented, observation)
}

// AuthorizeRestore gates terminal restore and the wrapper's first
// start through the fencing core.
func AuthorizeRestore(presented PresentedToken, observation Observation) (LeaseToken, error) {
	return Authorize(OperationRestore, presented, observation)
}

// Authorize is the fencing core: every activation-class action passes
// here. The presented token must name this session and equal the
// winning (epoch, lease_id) tuple exactly; the fencing grant must be
// present and unexpired; ownership must be verified, unambiguous, and
// local. Launch-class entries (activation, restore) park with the
// session.parked vocabulary when ownership is remote, ambiguous,
// unverified, or absent; every other entry refuses outright. A passed
// gate mints the sealed LeaseToken the provider operations bind; a
// failed gate refuses with a Section 15 registered code and mints
// nothing.
//
// Arm order is precedence: malformed calls die first, then foreign
// sessions, then ownership presence and verifiability, then the
// fencing grant, then ownership direction (the operative fact beats
// the credential: a stale token under a remote winner parks remote,
// it does not report stale), then the exact tuple match.
func Authorize(operation Operation, presented PresentedToken, observation Observation) (LeaseToken, error) {
	if !validOperation(operation) {
		return LeaseToken{}, refuse(ErrInvalidArguments, "unknown fencing operation %q", string(operation))
	}
	if _, err := scalar.ParseUUIDv7(presented.SessionID); err != nil {
		return LeaseToken{}, refuse(ErrInvalidArguments, "presented fencing token session %q is not a UUIDv7: %v", presented.SessionID, err)
	}
	if presented.Epoch == 0 {
		return LeaseToken{}, refuse(ErrInvalidArguments, "presented fencing token carries epoch 0, want an epoch at or above 1")
	}
	if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil {
		return LeaseToken{}, refuse(ErrInvalidArguments, "presented fencing token lease %q is not a UUIDv4: %v", presented.LeaseID, err)
	}
	if presented.SessionID != observation.SessionID {
		return LeaseToken{}, refuse(ErrLeaseConflict, "presented fencing token for session %s does not authorize session %s", presented.SessionID, observation.SessionID)
	}
	if !observation.HasWinner {
		if launchClass(operation) {
			return LeaseToken{}, park(ParkRestorePolicy, "", ErrLeaseConflict)
		}
		return LeaseToken{}, refuse(ErrLeaseConflict, "no winning lease is established for session %s", observation.SessionID)
	}
	if observation.Winner.Epoch == 0 || !validLeaseID(observation.Winner.LeaseID) || !validHostID(observation.Winner.HolderHostID) {
		return LeaseToken{}, refuse(ErrInvalidArguments, "fencing observation for session %s carries no well-formed winning lease", observation.SessionID)
	}
	if !observation.Verified {
		if launchClass(operation) {
			return LeaseToken{}, park(ParkRestorePolicy, observation.Winner.LeaseID, ErrLeaseConflict)
		}
		return LeaseToken{}, refuse(ErrLeaseConflict, "ownership of session %s is unverified; refresh lease knowledge first", observation.SessionID)
	}
	if observation.Ambiguous {
		if launchClass(operation) {
			return LeaseToken{}, park(ParkRestorePolicy, observation.Winner.LeaseID, ErrLeaseConflict)
		}
		return LeaseToken{}, refuse(ErrLeaseConflict, "ownership of session %s is ambiguous; resolve the union rival first", observation.SessionID)
	}
	if observation.HandoffFailed {
		if launchClass(operation) {
			return LeaseToken{}, park(ParkFailedHandoff, observation.Winner.LeaseID, ErrLeaseConflict)
		}
		return LeaseToken{}, refuse(ErrLeaseConflict, "handoff for session %s already failed; a losing lease must never activate", observation.SessionID)
	}
	if !observation.HasGrant {
		return LeaseToken{}, refuse(ErrInvalidArguments, "fencing observation for session %s carries no grant; validate the winning token first", observation.SessionID)
	}
	if observation.Now.IsZero() {
		return LeaseToken{}, refuse(ErrInvalidArguments, "fencing observation for session %s carries no clock reading; expiry cannot be checked", observation.SessionID)
	}
	if err := sessrepo.CheckFencingExpiry(observation.Grant, observation.Policy, observation.Now); err != nil {
		if errors.Is(err, sessrepo.ErrFencingExpired) {
			return LeaseToken{}, refuse(ErrLeaseConflict, "fencing grant for session %s lapsed; revalidate the winning token", observation.SessionID)
		}
		return LeaseToken{}, refuse(ErrInvalidArguments, "fencing observation for session %s carries an unusable refresh policy", observation.SessionID)
	}
	if observation.Winner.HolderHostID != observation.LocalHostID {
		if launchClass(operation) {
			return LeaseToken{}, park(ParkRemoteOwner, observation.Winner.LeaseID, ErrNotOwner)
		}
		return LeaseToken{}, refuse(ErrNotOwner, "session %s is owned by remote host %s", observation.SessionID, observation.Winner.HolderHostID)
	}
	epochsEqual := presented.Epoch == observation.Winner.Epoch
	if !epochsEqual {
		if presented.Epoch < observation.Winner.Epoch {
			if launchClass(operation) {
				return LeaseToken{}, park(ParkStaleOwner, observation.Winner.LeaseID, ErrStaleOwner)
			}
			return LeaseToken{}, refuse(ErrStaleOwner, "presented epoch %d precedes winning epoch %d for session %s", presented.Epoch, observation.Winner.Epoch, observation.SessionID)
		}
		if launchClass(operation) {
			return LeaseToken{}, park(ParkRestorePolicy, observation.Winner.LeaseID, ErrLeaseConflict)
		}
		return LeaseToken{}, refuse(ErrLeaseConflict, "presented epoch %d names no known lease for session %s", presented.Epoch, observation.SessionID)
	}
	if presented.LeaseID != observation.Winner.LeaseID {
		if launchClass(operation) {
			return LeaseToken{}, park(ParkStaleOwner, observation.Winner.LeaseID, ErrLeaseConflict)
		}
		return LeaseToken{}, refuse(ErrLeaseConflict, "presented lease %s loses to winning lease %s at epoch %d", presented.LeaseID, observation.Winner.LeaseID, observation.Winner.Epoch)
	}
	return mintLeaseToken(observation.SessionID, observation.Winner.Epoch, observation.Winner.LeaseID), nil
}

// validLeaseID reports whether id parses as a fencing token UUID. The
// scalar owner holds the grammar; this is only the boolean access the
// combined winner arm needs.
func validLeaseID(id string) bool {
	_, err := scalar.ParseUUIDv4(id)
	return err == nil
}

// validHostID reports whether id parses as a host UUID. The scalar
// owner holds the grammar; this is only the boolean access the
// combined winner arm needs.
func validHostID(id string) bool {
	_, err := scalar.ParseUUIDv7(id)
	return err == nil
}
