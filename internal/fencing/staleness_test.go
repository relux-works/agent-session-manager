package fencing

import (
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// TestStalenessTupleArms pins the decided members: an older epoch and
// a same-epoch losing lease are stale relative to the winner, while
// the winning tuple and a future epoch are decided-not-stale. A
// future epoch is unknown to the gate (restore_policy), never stale.
func TestStalenessTupleArms(t *testing.T) {
	cases := []struct {
		name      string
		presented PresentedToken
		stale     bool
	}{
		{"older epoch is stale", PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}, true},
		{"same-epoch low loser is stale", PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: propLeaseLow}, true},
		{"same-epoch high loser is stale", PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseC}, true},
		{"winning tuple is not stale", fencePresented(), false},
		{"future epoch is not stale", PresentedToken{SessionID: fenceSessionA, Epoch: 9, LeaseID: fenceLeaseA}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stale, decided := StaleRelativeToWinner(tc.presented, fenceObservation())
			if !decided {
				t.Fatalf("StaleRelativeToWinner() decided = false, want the tuple verdict")
			}
			if stale != tc.stale {
				t.Errorf("StaleRelativeToWinner() stale = %v, want %v", stale, tc.stale)
			}
		})
	}
}

// TestStalenessIgnoresGrantPrecondition pins the reason the verdict
// exists: the grant arms (present, absent, lapsed, unreadable clock,
// unusable policy) never change the staleness fact. A grant
// authorizes; it does not make an incarnation less stale.
func TestStalenessIgnoresGrantPrecondition(t *testing.T) {
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	grants := map[string]func(*Observation){
		"fresh grant": func(observation *Observation) {},
		"no grant": func(observation *Observation) {
			observation.HasGrant = false
			observation.Grant = sessrepo.FencingGrant{}
		},
		"lapsed grant": func(observation *Observation) {
			observation.Grant.ValidatedAt = fenceValidatedAt.Add(-2 * fencePolicy.RefreshInterval)
		},
		"no clock reading": func(observation *Observation) {
			observation.Now = time.Time{}
		},
		"unusable policy": func(observation *Observation) {
			observation.Policy = sessrepo.FencingPolicy{}
		},
	}
	for name, mutate := range grants {
		t.Run(name+"/stale token is stale", func(t *testing.T) {
			observation := fenceObservation()
			mutate(&observation)
			stale, decided := StaleRelativeToWinner(stale, observation)
			if !decided || !stale {
				t.Errorf("StaleRelativeToWinner() = (%v, %v), want (true, true)", stale, decided)
			}
		})
		t.Run(name+"/winning token is not stale", func(t *testing.T) {
			observation := fenceObservation()
			mutate(&observation)
			stale, decided := StaleRelativeToWinner(fencePresented(), observation)
			if !decided || stale {
				t.Errorf("StaleRelativeToWinner() = (%v, %v), want (false, true)", stale, decided)
			}
		})
	}
}

// TestStalenessRemoteWinnerIsStillStale pins that direction is not
// staleness: a losing tuple under a remote winner is stale relative
// to the winner on any host, while the winning token observed from
// afar is not stale.
func TestStalenessRemoteWinnerIsStillStale(t *testing.T) {
	remote := fenceObservation()
	remote.LocalHostID = fenceHostB
	stale, decided := StaleRelativeToWinner(PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}, remote)
	if !decided || !stale {
		t.Errorf("StaleRelativeToWinner(stale, remote) = (%v, %v), want (true, true)", stale, decided)
	}
	stale, decided = StaleRelativeToWinner(fencePresented(), remote)
	if !decided || stale {
		t.Errorf("StaleRelativeToWinner(winning, remote) = (%v, %v), want (false, true)", stale, decided)
	}
}

// TestStalenessUndecidedMembers pins every ownership precondition: a
// malformed presented token, a session mismatch, a missing or
// malformed winner, and an unverified, ambiguous, or failed-handoff
// observation are undecidable — the caller surfaces the gate's own
// refusal or park instead of projecting a verdict over them.
func TestStalenessUndecidedMembers(t *testing.T) {
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	cases := map[string]func(*PresentedToken, *Observation){
		"presented session grammar": func(presented *PresentedToken, _ *Observation) {
			presented.SessionID = "nope"
		},
		"presented epoch zero": func(presented *PresentedToken, _ *Observation) {
			presented.Epoch = 0
		},
		"presented lease grammar": func(presented *PresentedToken, _ *Observation) {
			presented.LeaseID = "nope"
		},
		"session mismatch": func(presented *PresentedToken, _ *Observation) {
			presented.SessionID = fenceSessionB
		},
		"no winner": func(_ *PresentedToken, observation *Observation) {
			observation.HasWinner = false
		},
		"winner epoch zero": func(_ *PresentedToken, observation *Observation) {
			observation.Winner.Epoch = 0
		},
		"winner lease grammar": func(_ *PresentedToken, observation *Observation) {
			observation.Winner.LeaseID = "nope"
		},
		"winner host grammar": func(_ *PresentedToken, observation *Observation) {
			observation.Winner.HolderHostID = "nope"
		},
		"unverified": func(_ *PresentedToken, observation *Observation) {
			observation.Verified = false
		},
		"ambiguous": func(_ *PresentedToken, observation *Observation) {
			observation.Ambiguous = true
		},
		"failed handoff": func(_ *PresentedToken, observation *Observation) {
			observation.HandoffFailed = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			presented, observation := stale, fenceObservation()
			mutate(&presented, &observation)
			stale, decided := StaleRelativeToWinner(presented, observation)
			if decided || stale {
				t.Errorf("StaleRelativeToWinner() = (%v, %v), want (false, false)", stale, decided)
			}
		})
	}
}

// TestStalenessAgreesWithGateOnHotPath pins that the verdict is the
// gate's own tuple fact, not a second staleness rule: wherever the
// landed Authorize reaches its tuple arms — the direct launch-class
// question and the relative-to-winner question under a remote winner —
// a stale_owner park agrees with a stale verdict, an authorization
// agrees with a decided-not-stale verdict, and a future-epoch
// restore_policy park agrees with a decided-not-stale verdict.
func TestStalenessAgreesWithGateOnHotPath(t *testing.T) {
	presenters := map[string]PresentedToken{
		"exact":     fencePresented(),
		"stale":     {SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld},
		"loser_low": {SessionID: fenceSessionA, Epoch: 2, LeaseID: propLeaseLow},
		"future":    {SessionID: fenceSessionA, Epoch: 9, LeaseID: fenceLeaseA},
	}
	for name, presented := range presenters {
		t.Run("direct/"+name, func(t *testing.T) {
			_, err := Authorize(OperationRestore, presented, fenceObservation())
			stale, decided := StaleRelativeToWinner(presented, fenceObservation())
			switch {
			case err == nil:
				if !decided || stale {
					t.Errorf("authorized but verdict = (%v, %v), want (false, true)", stale, decided)
				}
			default:
				reason, _, parked := ParkDetails(err)
				if !parked {
					t.Fatalf("Authorize() = %v, want a park on the hot path", err)
				}
				switch reason {
				case ParkStaleOwner:
					if !decided || !stale {
						t.Errorf("stale_owner park but verdict = (%v, %v), want (true, true)", stale, decided)
					}
				case ParkRestorePolicy:
					if !decided || stale {
						t.Errorf("restore_policy park but verdict = (%v, %v), want (false, true)", stale, decided)
					}
				default:
					t.Fatalf("Authorize() parked %s, want stale_owner or restore_policy", reason)
				}
			}
		})
		t.Run("relative/"+name, func(t *testing.T) {
			observation := fenceObservation()
			observation.LocalHostID = fenceHostB
			_, err := Authorize(OperationRestore, presented, observation)
			if reason, _, parked := ParkDetails(err); !parked || reason != ParkRemoteOwner {
				t.Fatalf("direct question = %v, want the remote_owner park", err)
			}
			relative := observation
			relative.LocalHostID = observation.Winner.HolderHostID
			_, relativeErr := Authorize(OperationRestore, presented, relative)
			stale, decided := StaleRelativeToWinner(presented, observation)
			if relativeErr == nil {
				if !decided || stale {
					t.Errorf("relative authorized but verdict = (%v, %v), want (false, true)", stale, decided)
				}
				return
			}
			reason, _, parked := ParkDetails(relativeErr)
			if !parked {
				t.Fatalf("relative question = %v, want a park", relativeErr)
			}
			switch reason {
			case ParkStaleOwner:
				if !decided || !stale {
					t.Errorf("relative stale_owner park but verdict = (%v, %v), want (true, true)", stale, decided)
				}
			case ParkRestorePolicy:
				if !decided || stale {
					t.Errorf("relative restore_policy park but verdict = (%v, %v), want (false, true)", stale, decided)
				}
			default:
				t.Fatalf("relative question parked %s, want stale_owner or restore_policy", reason)
			}
		})
	}
}
