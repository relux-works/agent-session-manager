package fencing

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm drives
// Authorize itself at the after-restore boundary. The wrapper owns the
// interactive/attach choice; fencing must first return the remote-owner park
// so that choice remains reachable after the local grant lapses.
func TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm(t *testing.T) {
	observation := fenceObservation()
	observation.Winner.HolderHostID = fenceHostB
	observation.Now = fenceValidatedAt.Add(fencePolicy.RefreshInterval + time.Second)

	_, err := Authorize(OperationRestore, fencePresented(), observation)
	if err == nil {
		t.Fatal("Authorize() succeeded for a remote owner")
	}
	if strings.Contains(err.Error(), "lease_conflict") {
		t.Fatalf("Authorize() returned literal lease_conflict for the remote offer vector: %v", err)
	}
	reason, winningLease, parked := ParkDetails(err)
	if !parked || string(reason) != "remote_owner" || winningLease != fenceLeaseA {
		t.Fatalf("Authorize() = %v, want parked remote_owner under %s", err, fenceLeaseA)
	}
	if !errors.Is(err, ErrNotOwner) {
		t.Fatalf("Authorize() = %v, want the remote-owner cause", err)
	}
	if !strings.Contains(err.Error(), "not_owner") {
		t.Fatalf("Authorize() = %v, want literal not_owner cause", err)
	}
	t.Logf("remote interactive owner + lapsed local grant: park=%s lease=%s cause=not_owner", reason, winningLease)
}

// TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry keeps the
// direction-before-expiry arm ordering visible at every fencing entry:
// launch entries park remote_owner/not_owner and non-launch entries refuse
// not_owner, even when the local grant has lapsed.
func TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry(t *testing.T) {
	for name, entry := range fenceEntries() {
		t.Run(name, func(t *testing.T) {
			observation := fenceObservation()
			observation.Winner.HolderHostID = fenceHostB
			observation.Now = fenceValidatedAt.Add(fencePolicy.RefreshInterval + time.Second)
			_, err := entry.authorize(fencePresented(), observation)
			if entry.launch {
				mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, name+"/remote_lapsed")
			} else {
				mustRefuse(t, err, ErrNotOwner, name+"/remote_lapsed")
			}
			if err == nil || !strings.Contains(err.Error(), "not_owner") || strings.Contains(err.Error(), "lease_conflict") {
				t.Fatalf("%s/remote_lapsed = %v, want literal not_owner without lease_conflict", name, err)
			}
		})
	}
}

// TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict pins the
// neighboring local stale-token class: expiry still wins over tuple arms
// when direction is local, so every entry returns hard lease_conflict rather
// than a stale_owner park or refusal.
func TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict(t *testing.T) {
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	observation := fenceObservation()
	observation.Now = fenceValidatedAt.Add(fencePolicy.RefreshInterval + time.Second)
	for name, entry := range fenceEntries() {
		t.Run(name, func(t *testing.T) {
			_, err := entry.authorize(stale, observation)
			mustRefuse(t, err, ErrLeaseConflict, name+"/local_lapsed_stale")
			if !strings.Contains(err.Error(), "lease_conflict") || strings.Contains(err.Error(), "stale_owner") {
				t.Fatalf("%s/local_lapsed_stale = %v, want literal lease_conflict without stale_owner", name, err)
			}
		})
	}
}

func TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments(t *testing.T) {
	observation := fenceObservation()
	observation.Winner.HolderHostID = fenceHostB
	observation.Policy.RefreshInterval = 0

	_, err := Authorize(OperationRestore, fencePresented(), observation)
	mustRefuse(t, err, ErrInvalidArguments, "remote owner + unusable refresh policy")
	if IsParked(err) {
		t.Fatalf("remote owner + unusable refresh policy = %v, want literal invalid_arguments refusal", err)
	}
	if !strings.Contains(err.Error(), "invalid_arguments") {
		t.Fatalf("remote owner + unusable refresh policy = %v, want literal invalid_arguments refusal", err)
	}
	t.Logf("remote owner + unusable refresh policy: refusal=invalid_arguments")
}

// TestAuthorizeArmOrderingNeighbourVectors keeps the adjacent direct-gate
// outcomes visible: a live local grant authorizes, a remote owner with a live
// grant reaches the remote-owner arm, and a lapsed local grant remains
// lease_conflict.
func TestAuthorizeArmOrderingNeighbourVectors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Observation)
		check  func(*testing.T, LeaseToken, error)
	}{
		{
			name: "live_local_grant",
			check: func(t *testing.T, token LeaseToken, err error) {
				mustMint(t, token, err, "live_local_grant")
			},
		},
		{
			name: "remote_owner_live_grant",
			mutate: func(observation *Observation) {
				observation.Winner.HolderHostID = fenceHostB
			},
			check: func(t *testing.T, _ LeaseToken, err error) {
				mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, "remote_owner_live_grant")
				reason, _, parked := ParkDetails(err)
				if !parked || string(reason) != "remote_owner" || !strings.Contains(err.Error(), "not_owner") {
					t.Fatalf("remote_owner_live_grant = %v, want literal remote_owner/not_owner", err)
				}
			},
		},
		{
			name: "local_owner_lapsed_grant",
			mutate: func(observation *Observation) {
				observation.Now = fenceValidatedAt.Add(fencePolicy.RefreshInterval + time.Second)
			},
			check: func(t *testing.T, _ LeaseToken, err error) {
				mustRefuse(t, err, ErrLeaseConflict, "local_owner_lapsed_grant")
				if !strings.Contains(err.Error(), "lease_conflict") {
					t.Fatalf("local_owner_lapsed_grant = %v, want literal lease_conflict", err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observation := fenceObservation()
			if tc.mutate != nil {
				tc.mutate(&observation)
			}
			token, err := Authorize(OperationRestore, fencePresented(), observation)
			tc.check(t, token, err)
		})
	}
}
