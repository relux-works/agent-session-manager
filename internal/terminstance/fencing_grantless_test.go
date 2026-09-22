package terminstance

import (
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// grantlessShapes rosters the four grant-precondition members. Missing
// grant, missing clock, and unusable policy still refuse before the
// direction/tuple arms. A lapsed grant defers only its expiry outcome at
// the direct remote-owner question; the composed stale verdict still
// fences a stale incarnation.
func grantlessShapes() map[string]func(*fencing.Observation) {
	return map[string]func(*fencing.Observation){
		"no grant": func(observation *fencing.Observation) {
			observation.HasGrant = false
			observation.Grant = sessrepo.FencingGrant{}
		},
		"lapsed grant": func(observation *fencing.Observation) {
			observation.Grant.ValidatedAt = fixtureNow().Add(-2 * time.Hour)
		},
		"no clock reading": func(observation *fencing.Observation) {
			observation.Now = time.Time{}
		},
		"unusable policy": func(observation *fencing.Observation) {
			observation.Policy = sessrepo.FencingPolicy{}
		},
	}
}

// TestRV3F3_GrantLessStaleFences is the regression test for the
// rev2-F2/rev3-F3 class: a stale incarnation whose observation carries
// no LIVE fencing grant still fences. Every hard grant-precondition member
// (no grant, no clock reading, unusable policy) fences from active, parked
// and quiescing, under a local and a remote winner alike, for the
// older-epoch and the same-epoch lease-mismatch tokens. A lapsed grant
// under a remote winner is split across the two composed questions:
// Authorize surfaces the literal remote_owner park, then the
// grant-independent stale verdict fences the stale incarnation.
func TestRV3F3_GrantLessStaleFences(t *testing.T) {
	staleEpoch := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	losingLease := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLeaseB}
	tokens := map[string]fencing.PresentedToken{"stale epoch": staleEpoch, "losing lease": losingLease}
	winners := map[string]func() fencing.Observation{"local winner": fencingObservation, "remote winner": remoteWinnerObservation}
	states := []terminalbackend.InstanceState{terminalbackend.StateActive, terminalbackend.StateParked, terminalbackend.StateQuiescing}
	for shapeName, shape := range grantlessShapes() {
		for winnerName, winner := range winners {
			for tokenName, presented := range tokens {
				for _, current := range states {
					name := shapeName + "/" + winnerName + "/" + tokenName + "/" + string(current)
					t.Run(name, func(t *testing.T) {
						observation := winner()
						shape(&observation)
						if shapeName == "lapsed grant" && winnerName == "remote winner" {
							_, authErr := fencing.Authorize(fencing.OperationRestore, presented, observation)
							if authErr == nil {
								t.Fatal("direct gate question = nil, want remote_owner park")
							}
							reason, _, parked := fencing.ParkDetails(authErr)
							if !parked {
								t.Fatalf("direct gate question = %v, want remote_owner park", authErr)
							}
							requireLiteral(t, "direct park reason", string(reason), "remote_owner")
							state, transitioned, err := ObserveFencing(current, presented, observation)
							if err != nil {
								t.Fatalf("ObserveFencing() error = %v, want stale_fenced transition", err)
							}
							if !transitioned {
								t.Fatal("ObserveFencing() transitioned = false, want stale_fenced")
							}
							requireLiteral(t, "state", string(state), "stale_fenced")
							return
						}
						if _, err := fencing.Authorize(fencing.OperationRestore, presented, observation); err == nil || fencing.IsParked(err) {
							t.Fatalf("direct gate question = %v, want the grant-gated refusal (never a park)", err)
						}
						state, transitioned, err := ObserveFencing(current, presented, observation)
						if err != nil {
							t.Fatalf("ObserveFencing() error = %v, want the fencing transition", err)
						}
						if !transitioned {
							t.Fatalf("ObserveFencing() transitioned = false, want stale_fenced for the grant-less stale token")
						}
						requireLiteral(t, "state", string(state), "stale_fenced")
					})
				}
			}
		}
	}
}

// TestObserveFencingRemoteWinnerLapsedGrantFencesStaleIncarnation keeps the
// composition property independently named: the direct remote-owner park is
// the offer signal, but a stale incarnation still reaches stale_fenced through
// the relative, grant-independent verdict.
func TestObserveFencingRemoteWinnerLapsedGrantFencesStaleIncarnation(t *testing.T) {
	tokens := map[string]fencing.PresentedToken{
		"stale_epoch":  {SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB},
		"losing_lease": {SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLeaseB},
	}
	for tokenName, presented := range tokens {
		for _, current := range []terminalbackend.InstanceState{
			terminalbackend.StateActive, terminalbackend.StateParked, terminalbackend.StateQuiescing,
		} {
			t.Run(tokenName+"/"+string(current), func(t *testing.T) {
				observation := remoteWinnerObservation()
				observation.Grant.ValidatedAt = fixtureNow().Add(-2 * time.Hour)
				state, transitioned, err := ObserveFencing(current, presented, observation)
				if err != nil {
					t.Fatalf("ObserveFencing() error = %v, want stale_fenced transition", err)
				}
				if !transitioned {
					t.Fatal("ObserveFencing() transitioned = false, want stale_fenced")
				}
				requireLiteral(t, "state", string(state), "stale_fenced")
			})
		}
	}
}

// TestRV3F3_GrantLessDecidedNotStaleLeavesState pins the decided
// members that must NOT fence without a grant: the winning token and
// a future-epoch token surface the grant refusal with no transition,
// under a local and a remote winner alike. A lapsed grant under a
// remote winner follows the remote_owner park arm for the non-stale
// winning/future tokens, while stale-shaped tokens are fenced by the
// relative verdict.
func TestRV3F3_GrantLessDecidedNotStaleLeavesState(t *testing.T) {
	winning := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLease}
	future := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 3, LeaseID: fixtureLeaseB}
	cases := []struct {
		name      string
		presented fencing.PresentedToken
		winner    func() fencing.Observation
	}{
		{"winning_token_local", winning, fencingObservation},
		{"winning_token_remote", winning, remoteWinnerObservation},
		{"future_epoch_local", future, fencingObservation},
		{"future_epoch_remote", future, remoteWinnerObservation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for shapeName, shape := range map[string]func(*fencing.Observation){"no grant": grantlessShapes()["no grant"], "lapsed grant": grantlessShapes()["lapsed grant"]} {
				observation := tc.winner()
				shape(&observation)
				state, transitioned, err := ObserveFencing(terminalbackend.StateActive, tc.presented, observation)
				if shapeName == "lapsed grant" && (tc.name == "winning_token_remote" || tc.name == "future_epoch_remote") {
					if err == nil {
						t.Fatal("ObserveFencing(lapsed grant) = nil, want the remote_owner park")
					}
					if !fencing.IsParked(err) {
						t.Fatalf("ObserveFencing(lapsed grant) = %v, want remote_owner park", err)
					}
					if transitioned {
						t.Fatal("ObserveFencing(lapsed grant) transitioned = true, want no transition")
					}
					requireLiteral(t, "lapsed remote state", string(state), "active")
					reason, _, parked := fencing.ParkDetails(err)
					if !parked {
						t.Fatalf("ParkDetails(lapsed grant) = %v, want remote_owner park", err)
					}
					requireLiteral(t, "lapsed remote park reason", string(reason), "remote_owner")
					continue
				}
				if err == nil {
					t.Fatalf("ObserveFencing(%s) = nil, want the grant refusal surfaced", shapeName)
				}
				if fencing.IsParked(err) {
					t.Fatalf("ObserveFencing(%s) parked, want the grant refusal: %v", shapeName, err)
				}
				if transitioned {
					t.Fatalf("ObserveFencing(%s) transitioned = true, want no transition for the decided-not-stale token", shapeName)
				}
				requireLiteral(t, "state", string(state), "active")
			}
		})
	}
}

// TestRV3F3_GrantLessUndecidedLeavesState pins every ownership
// precondition through the composed entry: each undecidable shape
// surfaces the gate's own refusal or park with no transition, even
// with a stale-shaped token and no grant. Each subtest drives two
// stale-shaped tokens (the older epoch and the same-epoch mismatch,
// except epoch_zero which drives two zero-epoch leases) so a mutant
// deciding exactly one member still kills on the other.
func TestRV3F3_GrantLessUndecidedLeavesState(t *testing.T) {
	staleEpoch := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	losingLease := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 2, LeaseID: fixtureLeaseB}
	noGrant := func(observation *fencing.Observation) {
		observation.HasGrant = false
		observation.Grant = sessrepo.FencingGrant{}
	}
	parked := func(t *testing.T, err error, reason string) {
		t.Helper()
		if err == nil {
			t.Fatalf("ObserveFencing() = nil, want the %s park surfaced", reason)
		}
		gotReason, _, isParked := fencing.ParkDetails(err)
		if !isParked {
			t.Fatalf("ObserveFencing() = %v, want the %s park", err, reason)
		}
		requireLiteral(t, "park reason", string(gotReason), reason)
	}
	refused := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("ObserveFencing() = nil, want the refusal surfaced")
		}
		if fencing.IsParked(err) {
			t.Fatalf("ObserveFencing() parked, want the hard refusal: %v", err)
		}
	}
	leaves := func(t *testing.T, presented fencing.PresentedToken, mutate func(*fencing.PresentedToken, *fencing.Observation), assert func(*testing.T, error)) {
		t.Helper()
		observation := fencingObservation()
		noGrant(&observation)
		mutate(&presented, &observation)
		state, transitioned, err := ObserveFencing(terminalbackend.StateActive, presented, observation)
		assert(t, err)
		if transitioned {
			t.Error("ObserveFencing() transitioned = true, want no transition for the undecidable observation")
		}
		requireLiteral(t, "state", string(state), "active")
	}
	identity := func(*fencing.PresentedToken, *fencing.Observation) {}
	t.Run("unverified", func(t *testing.T) {
		mutate := func(_ *fencing.PresentedToken, observation *fencing.Observation) { observation.Verified = false }
		leaves(t, staleEpoch, mutate, func(t *testing.T, err error) { parked(t, err, "restore_policy") })
		leaves(t, losingLease, mutate, func(t *testing.T, err error) { parked(t, err, "restore_policy") })
	})
	t.Run("ambiguous", func(t *testing.T) {
		mutate := func(_ *fencing.PresentedToken, observation *fencing.Observation) { observation.Ambiguous = true }
		leaves(t, staleEpoch, mutate, func(t *testing.T, err error) { parked(t, err, "restore_policy") })
		leaves(t, losingLease, mutate, func(t *testing.T, err error) { parked(t, err, "restore_policy") })
	})
	t.Run("no_winner", func(t *testing.T) {
		mutate := func(_ *fencing.PresentedToken, observation *fencing.Observation) { observation.HasWinner = false }
		leaves(t, staleEpoch, mutate, func(t *testing.T, err error) { parked(t, err, "restore_policy") })
		leaves(t, losingLease, mutate, func(t *testing.T, err error) { parked(t, err, "restore_policy") })
	})
	t.Run("malformed_winner", func(t *testing.T) {
		mutate := func(_ *fencing.PresentedToken, observation *fencing.Observation) { observation.Winner.LeaseID = "nope" }
		leaves(t, staleEpoch, mutate, refused)
		leaves(t, losingLease, mutate, refused)
	})
	t.Run("session_mismatch", func(t *testing.T) {
		mutate := func(presented *fencing.PresentedToken, _ *fencing.Observation) { presented.SessionID = fixtureSessionB }
		leaves(t, staleEpoch, mutate, refused)
		leaves(t, losingLease, mutate, refused)
	})
	t.Run("failed_handoff", func(t *testing.T) {
		mutate := func(_ *fencing.PresentedToken, observation *fencing.Observation) { observation.HandoffFailed = true }
		leaves(t, staleEpoch, mutate, func(t *testing.T, err error) { parked(t, err, "failed_handoff") })
		leaves(t, losingLease, mutate, func(t *testing.T, err error) { parked(t, err, "failed_handoff") })
		_ = identity
	})
	t.Run("malformed_session", func(t *testing.T) {
		mutate := func(presented *fencing.PresentedToken, _ *fencing.Observation) { presented.SessionID = "nope" }
		leaves(t, staleEpoch, mutate, refused)
		leaves(t, losingLease, mutate, refused)
		// Agreed-malformed: both sides name the same malformed
		// session, so ONLY the grammar arm (not the session-match
		// arm) undecides this member.
		agreed := func(presented *fencing.PresentedToken, observation *fencing.Observation) {
			presented.SessionID = "nope"
			observation.SessionID = "nope"
		}
		leaves(t, staleEpoch, agreed, refused)
	})
	t.Run("epoch_zero", func(t *testing.T) {
		zeroB := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 0, LeaseID: fixtureLeaseB}
		zeroA := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 0, LeaseID: fixtureLease}
		leaves(t, zeroB, identity, refused)
		leaves(t, zeroA, identity, refused)
	})
	t.Run("malformed_lease", func(t *testing.T) {
		mutate := func(presented *fencing.PresentedToken, _ *fencing.Observation) { presented.LeaseID = "nope" }
		leaves(t, staleEpoch, mutate, refused)
		leaves(t, losingLease, mutate, refused)
	})
}

// TestRV3F3_GrantLessNonLiveSourcesLeaveState pins the live-source
// gate under the verdict path: creating fences grant-less alongside
// the other live states, while absent, stopped, stale_fenced and
// unavailable surface the grant refusal with no transition — under a
// local and (for absent) a remote winner.
func TestRV3F3_GrantLessNonLiveSourcesLeaveState(t *testing.T) {
	presented := fencing.PresentedToken{SessionID: fixtureSessionA, Epoch: 1, LeaseID: fixtureLeaseB}
	observation := fencingObservation()
	observation.HasGrant = false
	observation.Grant = sessrepo.FencingGrant{}
	state, transitioned, err := ObserveFencing(terminalbackend.StateCreating, presented, observation)
	if err != nil {
		t.Fatalf("ObserveFencing(creating) error = %v, want the fencing transition", err)
	}
	if !transitioned {
		t.Fatal("ObserveFencing(creating) transitioned = false, want stale_fenced grant-less")
	}
	requireLiteral(t, "creating state", string(state), "stale_fenced")
	for _, current := range []terminalbackend.InstanceState{terminalbackend.StateAbsent, terminalbackend.StateStopped, terminalbackend.StateStaleFenced, terminalbackend.StateUnavailable} {
		state, transitioned, err := ObserveFencing(current, presented, observation)
		if err == nil {
			t.Errorf("ObserveFencing(%s) = nil, want the grant refusal surfaced", current)
			continue
		}
		if fencing.IsParked(err) {
			t.Errorf("ObserveFencing(%s) parked, want the grant refusal: %v", current, err)
			continue
		}
		if transitioned {
			t.Errorf("ObserveFencing(%s) transitioned = true, want no transition", current)
			continue
		}
		requireLiteral(t, "state", string(state), string(current))
	}
	remote := remoteWinnerObservation()
	remote.HasGrant = false
	remote.Grant = sessrepo.FencingGrant{}
	absent, transitioned, err := ObserveFencing(terminalbackend.StateAbsent, presented, remote)
	if err == nil || transitioned {
		t.Fatalf("ObserveFencing(absent, remote winner) = (%v, %v), want (no transition, error)", transitioned, err)
	}
	requireLiteral(t, "absent state", string(absent), "absent")
	if fencing.IsParked(err) {
		t.Fatalf("ObserveFencing(absent, remote winner) parked, want the grant refusal: %v", err)
	}
}
