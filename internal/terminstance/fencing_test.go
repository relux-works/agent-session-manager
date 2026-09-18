package terminstance

import (
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// fencingObservation builds a verified local observation over the
// winning lease: epoch 2 held by the fixture host, with a live grant.
func fencingObservation() fencing.Observation {
	now := fixtureNow()
	return fencing.Observation{
		SessionID:   fixtureSessionA,
		Winner:      sessrepo.LeaseSummary{SessionID: fixtureSessionA, LeaseID: fixtureLease, Epoch: 2, HolderHostID: fixtureHost},
		HasWinner:   true,
		LocalHostID: fixtureHost,
		Verified:    true,
		Grant: sessrepo.FencingGrant{
			SessionID:   fixtureSessionA,
			Token:       sessrepo.FencingToken{Epoch: 2, LeaseID: fixtureLease, HolderHostID: fixtureHost},
			ValidatedAt: now,
		},
		HasGrant: true,
		Policy:   sessrepo.FencingPolicy{RefreshInterval: time.Hour},
		Now:      now,
	}
}

// TestObserveFencingStaleEpochFences proves the older-epoch arm: a
// presented token from epoch 1 under a winning epoch 2 moves the local
// observation to stale_fenced through the production entry.
func TestObserveFencingStaleEpochFences(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	state, transitioned, err := ObserveFencing(terminalbackend.StateActive, presented, fencingObservation())
	if err != nil {
		t.Fatalf("ObserveFencing() error = %v", err)
	}
	if !transitioned {
		t.Fatal("ObserveFencing() transitioned = false, want the fencing transition")
	}
	requireLiteral(t, "state", string(state), "stale_fenced")
}

// TestObserveFencingLeaseMismatchFences proves the same-epoch
// lease-mismatch arm: the winning epoch with a losing lease ID is
// fenced too, not only the older epoch.
func TestObserveFencingLeaseMismatchFences(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLeaseB}
	state, transitioned, err := ObserveFencing(terminalbackend.StateParked, presented, fencingObservation())
	if err != nil {
		t.Fatalf("ObserveFencing() error = %v", err)
	}
	if !transitioned {
		t.Fatal("ObserveFencing() transitioned = false, want the fencing transition")
	}
	requireLiteral(t, "state", string(state), "stale_fenced")
}

// TestObserveFencingNonStaleLeavesState proves only proven staleness
// fences: the winning token authorizes with no transition, and remote,
// unverified and winnerless observations surface their own refusal
// with no transition and no fencing claim.
func TestObserveFencingNonStaleLeavesState(t *testing.T) {
	observation := fencingObservation()
	winning := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLease}
	state, transitioned, err := ObserveFencing(terminalbackend.StateActive, winning, observation)
	if err != nil {
		t.Fatalf("ObserveFencing(winning) error = %v", err)
	}
	if transitioned {
		t.Error("ObserveFencing(winning) transitioned = true, want no transition")
	}
	requireLiteral(t, "state", string(state), "active")

	remote := observation
	remote.LocalHostID = fixtureSessionB
	if _, transitioned, err := ObserveFencing(terminalbackend.StateActive, winning, remote); err == nil || transitioned {
		t.Errorf("ObserveFencing(remote) = (%v, %v), want (no transition, error)", transitioned, err)
	}

	unverified := observation
	unverified.Verified = false
	if _, transitioned, err := ObserveFencing(terminalbackend.StateActive, winning, unverified); err == nil || transitioned {
		t.Errorf("ObserveFencing(unverified) = (%v, %v), want (no transition, error)", transitioned, err)
	}

	nowinner := observation
	nowinner.HasWinner = false
	if _, transitioned, err := ObserveFencing(terminalbackend.StateActive, winning, nowinner); err == nil || transitioned {
		t.Errorf("ObserveFencing(no winner) = (%v, %v), want (no transition, error)", transitioned, err)
	}
}

// TestObserveFencingStaleParkVocabulary proves the fencing verdict reads
// the landed stale_owner park: authorizing the stale token directly
// through the landed gate parks stale_owner with the winning lease.
func TestObserveFencingStaleParkVocabulary(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	_, err := fencing.Authorize(fencing.OperationRestore, presented, fencingObservation())
	if err == nil {
		t.Fatal("Authorize(stale) = nil, want the park")
	}
	reason, winner, parked := fencing.ParkDetails(err)
	if !parked {
		t.Fatalf("ParkDetails() parked = false, want the park: %v", err)
	}
	requireLiteral(t, "park reason", string(reason), "stale_owner")
	requireLiteral(t, "winning lease", winner, "f47ac10b-58cc-4372-a567-0e02b2c3d479")
}
