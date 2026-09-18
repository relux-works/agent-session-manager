package terminstance

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// remoteWinnerObservation builds a verified observation whose winner
// (epoch 2, lease A) is held by ANOTHER host: the force-takeover case
// where the prior owner is remote. The local host holds the stale
// epoch-1 lease B.
func remoteWinnerObservation() fencing.Observation {
	observation := fencingObservation()
	observation.Winner = sessrepo.LeaseSummary{SessionID: fixtureSessionA, LeaseID: fixtureLease, Epoch: 2, HolderHostID: "0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa2"}
	observation.Grant = sessrepo.FencingGrant{
		SessionID:   fixtureSessionA,
		Token:       sessrepo.FencingToken{Epoch: 1, LeaseID: fixtureLeaseB, HolderHostID: fixtureHost},
		ValidatedAt: fixtureNow(),
	}
	return observation
}

// TestObserveFencingRemoteWinnerStaleEpochFences proves a stale-epoch
// incarnation fences under a remote winner: the force takeover from
// another host moves the local observation to stale_fenced from every
// live state, through the production entry.
func TestObserveFencingRemoteWinnerStaleEpochFences(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	for _, current := range []terminalbackend.InstanceState{terminalbackend.StateActive, terminalbackend.StateParked, terminalbackend.StateQuiescing} {
		state, transitioned, err := ObserveFencing(current, presented, remoteWinnerObservation())
		if err != nil {
			t.Fatalf("ObserveFencing(%s) error = %v, want the fencing transition", current, err)
		}
		if !transitioned {
			t.Fatalf("ObserveFencing(%s) transitioned = false, want stale_fenced under the remote epoch-2 winner", current)
		}
		requireLiteral(t, "state", string(state), "stale_fenced")
	}
}

// TestObserveFencingRemoteWinnerLosingLeaseFences proves the
// same-epoch losing lease fences under a remote winner too, not only
// the older epoch.
func TestObserveFencingRemoteWinnerLosingLeaseFences(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLeaseB}
	for _, current := range []terminalbackend.InstanceState{terminalbackend.StateActive, terminalbackend.StateParked, terminalbackend.StateQuiescing} {
		state, transitioned, err := ObserveFencing(current, presented, remoteWinnerObservation())
		if err != nil {
			t.Fatalf("ObserveFencing(%s) error = %v, want the fencing transition", current, err)
		}
		if !transitioned {
			t.Fatalf("ObserveFencing(%s) transitioned = false, want stale_fenced for the losing lease under the remote winner", current)
		}
		requireLiteral(t, "state", string(state), "stale_fenced")
	}
}

// TestObserveFencingRemoteWinnerWinningTokenLeavesState proves the
// winning token under a foreign host is not stale: it surfaces the
// remote park with no transition from every live state.
func TestObserveFencingRemoteWinnerWinningTokenLeavesState(t *testing.T) {
	winning := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLease}
	for _, current := range []terminalbackend.InstanceState{terminalbackend.StateActive, terminalbackend.StateParked, terminalbackend.StateQuiescing} {
		state, transitioned, err := ObserveFencing(current, winning, remoteWinnerObservation())
		if err == nil {
			t.Fatalf("ObserveFencing(%s) = nil, want the remote park", current)
		}
		if transitioned {
			t.Fatalf("ObserveFencing(%s) transitioned = true, want no transition for the winning token", current)
		}
		requireLiteral(t, "state", string(state), string(current))
		reason, winner, parked := fencing.ParkDetails(err)
		if !parked {
			t.Fatalf("ParkDetails() parked = false, want the park: %v", err)
		}
		requireLiteral(t, "park reason", string(reason), "remote_owner")
		requireLiteral(t, "winning lease", winner, "f47ac10b-58cc-4372-a567-0e02b2c3d479")
	}
}

// TestObserveFencingRemoteWinnerFutureEpochLeavesState proves a
// future-epoch token under a remote winner is not stale either: the
// relative verdict parks restore_policy, never stale_owner, so no
// transition follows.
func TestObserveFencingRemoteWinnerFutureEpochLeavesState(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 3, LeaseID: fixtureLeaseB}
	state, transitioned, err := ObserveFencing(terminalbackend.StateActive, presented, remoteWinnerObservation())
	if err == nil {
		t.Fatal("ObserveFencing(future epoch) = nil, want the remote park")
	}
	if transitioned {
		t.Fatal("ObserveFencing(future epoch) transitioned = true, want no transition")
	}
	requireLiteral(t, "state", string(state), "active")
	reason, _, parked := fencing.ParkDetails(err)
	if !parked {
		t.Fatalf("ParkDetails() parked = false, want the park: %v", err)
	}
	requireLiteral(t, "park reason", string(reason), "remote_owner")
}

// TestObserveFencingNonLiveSourcesLeaveState proves only a live
// incarnation fences: creating fences alongside parked, active and
// quiescing, while absent and stopped (no incarnation exists for
// terminate-stale to target), stale_fenced (already fenced) and
// unavailable (unknown: no verdict is projected over unknown) surface
// the park with no transition.
func TestObserveFencingNonLiveSourcesLeaveState(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	state, transitioned, err := ObserveFencing(terminalbackend.StateCreating, presented, fencingObservation())
	if err != nil {
		t.Fatalf("ObserveFencing(creating) error = %v, want the fencing transition", err)
	}
	if !transitioned {
		t.Fatal("ObserveFencing(creating) transitioned = false, want stale_fenced for the materializing incarnation")
	}
	requireLiteral(t, "creating state", string(state), "stale_fenced")

	for _, current := range []terminalbackend.InstanceState{terminalbackend.StateAbsent, terminalbackend.StateStopped, terminalbackend.StateStaleFenced, terminalbackend.StateUnavailable} {
		state, transitioned, err := ObserveFencing(current, presented, fencingObservation())
		if err == nil {
			t.Errorf("ObserveFencing(%s) = nil, want the stale park surfaced", current)
			continue
		}
		if transitioned {
			t.Errorf("ObserveFencing(%s) transitioned = true, want no transition", current)
			continue
		}
		requireLiteral(t, "state", string(state), string(current))
		reason, _, parked := fencing.ParkDetails(err)
		if !parked {
			t.Errorf("ObserveFencing(%s) ParkDetails() parked = false, want the park: %v", current, err)
			continue
		}
		requireLiteral(t, "park reason", string(reason), "stale_owner")
	}

	// The same bound holds under a remote winner: the stale epoch-1
	// incarnation that does not exist surfaces the remote park without
	// fencing.
	absent, transitioned, err := ObserveFencing(terminalbackend.StateAbsent, presented, remoteWinnerObservation())
	if err == nil || transitioned {
		t.Fatalf("ObserveFencing(absent, remote winner) = (%v, %v), want (no transition, error)", transitioned, err)
	}
	requireLiteral(t, "absent state", string(absent), "absent")
	reason, _, parked := fencing.ParkDetails(err)
	if !parked {
		t.Fatalf("ParkDetails() parked = false, want the park: %v", err)
	}
	requireLiteral(t, "park reason", string(reason), "remote_owner")
}
