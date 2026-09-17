package fencing

import (
	"errors"
	"testing"
	"time"
)

// Ownership gate properties (TASK-260830-2atgj4): the gate halves of
// invariants 3 (clock non-authority) and 4 (zero duplicate authorized
// owners) over the landed Authorize* entries. The reducer halves live
// in internal/sessstate, the store halves in internal/sessrepo.
//
// Clock bound: the absolute wall-clock position is non-authoritative;
// the fencing-grant AGE relative to the refresh policy is
// authoritative by design (sessrepo.CheckFencingExpiry). The property
// therefore translates (ValidatedAt, Now) together and asserts
// identical verdicts at a fixed age, including a lapsed age and the
// exact interval boundary. PresentedToken and Observation carry no
// created_at member, so lease creation time is non-authoritative by
// construction.

// propLeaseLow is a valid fencing token sorting below the winning
// lease A: the below-winner half of the same-epoch loser pair.
const propLeaseLow = "00000000-1111-4000-8000-000000000001"

// propVerdict encodes an Authorize outcome for invariance
// comparison: authorized, refused with its Section 15 class, or
// parked with the session.parked reason, the cause class, and the
// repeated winning lease.
func propVerdict(err error) string {
	if err == nil {
		return "authorized"
	}
	classes := []error{ErrLeaseConflict, ErrNotOwner, ErrStaleOwner, ErrInvalidArguments, ErrLocalPreconditionFailed}
	classOf := func() string {
		for _, class := range classes {
			if errors.Is(err, class) {
				return class.Error()
			}
		}
		return "unknown"
	}
	if reason, leaseID, ok := ParkDetails(err); ok {
		return "parked:" + string(reason) + "/" + classOf() + "/" + leaseID
	}
	return "refused:" + classOf()
}

// TestOwnershipGateClockNonAuthority proves invariant 3 over the
// gates: translating the wall clock by -1h, 0, +1h, or +365d with
// the grant age fixed leaves every verdict — authorized, refused
// with its class, or parked with its reason — identical across all
// five operations. Ages cover fresh (30s), the exact interval
// boundary (60s, still current), zero (Now == ValidatedAt), and
// lapsed (61s). 4 ages x 4 translations x 5 operations = 80
// authorizations.
func TestOwnershipGateClockNonAuthority(t *testing.T) {
	translations := []time.Duration{-time.Hour, 0, time.Hour, 365 * 24 * time.Hour}
	ages := []time.Duration{0, 30 * time.Second, time.Minute, time.Minute + time.Second}
	entries := fenceEntries()
	for _, age := range ages {
		for name, entry := range entries {
			var baseline string
			var first bool
			for _, delta := range translations {
				observation := fenceObservation()
				observation.Grant.ValidatedAt = fenceValidatedAt.Add(delta)
				observation.Now = fenceValidatedAt.Add(delta).Add(age)
				token, err := entry.authorize(fencePresented(), observation)
				got := propVerdict(err)
				if err == nil {
					mustMint(t, token, err, name)
				}
				if !first {
					baseline, first = got, true
					continue
				}
				if got != baseline {
					t.Fatalf("op %s age %v translation %v: verdict %q, want translation-invariant %q",
						name, age, delta, got, baseline)
				}
			}
			// The age classes themselves are pinned: a current
			// grant authorizes, a lapsed grant refuses
			// lease_conflict on every entry including launch.
			want := "authorized"
			if age > fencePolicy.RefreshInterval {
				want = "refused:lease_conflict"
			}
			if baseline != want {
				t.Fatalf("op %s age %v: verdict %q, want %q", name, age, baseline, want)
			}
		}
	}
	t.Logf("gate-clock: %d ages x %d translations x %d operations, verdicts translation-invariant",
		len(ages), len(translations), len(entries))
}

// TestOwnershipGateSingleAuthorizedOwner proves invariant 4 over the
// gates: across the presenter grid (exact winner, stale epoch,
// below- and above-winner same-epoch losers, future epoch, foreign
// session) exactly one presenter authorizes per operation — the exact
// winning tuple on the holder's host — and no two distinct
// presenters, hosts, or processes ever both pass an
// activation-class gate for the same session. 6 presenters x 5
// observations x 5 operations = 150 authorizations, exhaustive.
func TestOwnershipGateSingleAuthorizedOwner(t *testing.T) {
	presenters := map[string]PresentedToken{
		"exact":      fencePresented(),
		"stale":      {SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld},
		"loser_low":  {SessionID: fenceSessionA, Epoch: 2, LeaseID: propLeaseLow},
		"loser_high": {SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseC},
		"future":     {SessionID: fenceSessionA, Epoch: 9, LeaseID: fenceLeaseA},
		"foreign":    {SessionID: fenceSessionB, Epoch: 2, LeaseID: fenceLeaseA},
	}
	local := fenceObservation()
	remote := fenceObservation()
	remote.LocalHostID = fenceHostB
	ambiguous := fenceObservation()
	ambiguous.Ambiguous = true
	failed := fenceObservation()
	failed.HandoffFailed = true
	unverified := fenceObservation()
	unverified.Verified = false
	observations := map[string]Observation{
		"local": local, "remote": remote,
		"ambiguous": ambiguous, "failed_handoff": failed, "unverified": unverified,
	}
	// verdicts[observation][operation][presenter] is the recorded
	// table every assertion below reads.
	verdicts := map[string]map[string]map[string]string{}
	entries := fenceEntries()
	for obsName, observation := range observations {
		verdicts[obsName] = map[string]map[string]string{}
		for opName, entry := range entries {
			verdicts[obsName][opName] = map[string]string{}
			for presenterName, presented := range presenters {
				token, err := entry.authorize(presented, observation)
				verdicts[obsName][opName][presenterName] = propVerdict(err)
				if err == nil {
					mustMint(t, token, err, obsName+"/"+opName+"/"+presenterName)
				}
			}
		}
	}
	// The local holder's table is pinned cell by cell: the exact
	// winner authorizes on every operation; every other presenter
	// parks or refuses with its landed class.
	wantLocal := map[string]map[string]string{
		"exact": {
			"activation": "authorized", "input": "authorized", "mutation": "authorized",
			"checkpoint": "authorized", "restore": "authorized",
		},
		"stale": {
			"activation": "parked:stale_owner/stale_owner/" + fenceLeaseA,
			"restore":    "parked:stale_owner/stale_owner/" + fenceLeaseA,
			"input":      "refused:stale_owner", "mutation": "refused:stale_owner",
			"checkpoint": "refused:stale_owner",
		},
		"loser_low": {
			"activation": "parked:stale_owner/lease_conflict/" + fenceLeaseA,
			"restore":    "parked:stale_owner/lease_conflict/" + fenceLeaseA,
			"input":      "refused:lease_conflict", "mutation": "refused:lease_conflict",
			"checkpoint": "refused:lease_conflict",
		},
		"future": {
			"activation": "parked:restore_policy/lease_conflict/" + fenceLeaseA,
			"restore":    "parked:restore_policy/lease_conflict/" + fenceLeaseA,
			"input":      "refused:lease_conflict", "mutation": "refused:lease_conflict",
			"checkpoint": "refused:lease_conflict",
		},
		"foreign": {
			"activation": "refused:lease_conflict", "input": "refused:lease_conflict",
			"mutation": "refused:lease_conflict", "checkpoint": "refused:lease_conflict",
			"restore": "refused:lease_conflict",
		},
	}
	wantLocal["loser_high"] = wantLocal["loser_low"]
	for presenterName, ops := range wantLocal {
		for opName, want := range ops {
			if got := verdicts["local"][opName][presenterName]; got != want {
				t.Fatalf("local/%s/%s verdict = %q, want %q", opName, presenterName, got, want)
			}
		}
	}
	// Exactly one presenter authorizes per local operation; none
	// authorizes anywhere else.
	for opName := range entries {
		authorized := 0
		for presenterName := range presenters {
			if verdicts["local"][opName][presenterName] == "authorized" {
				authorized++
			}
		}
		if authorized != 1 {
			t.Fatalf("local/%s authorizes %d presenters, want exactly one", opName, authorized)
		}
		for _, obsName := range []string{"remote", "ambiguous", "failed_handoff", "unverified"} {
			for presenterName := range presenters {
				if verdicts[obsName][opName][presenterName] == "authorized" {
					t.Fatalf("%s/%s/%s authorizes, want no authorization off the local verified winner",
						obsName, opName, presenterName)
				}
			}
		}
	}
	// The remote holder parks remote_owner (launch) or refuses
	// not_owner (non-launch) even for the exact token: two hosts can
	// never both pass.
	for opName, entry := range entries {
		want := "refused:not_owner"
		if entry.launch {
			want = "parked:remote_owner/not_owner/" + fenceLeaseA
		}
		if got := verdicts["remote"][opName]["exact"]; got != want {
			t.Fatalf("remote/%s/exact verdict = %q, want %q", opName, got, want)
		}
	}
	// Pairwise mutual exclusion over the recorded table: no two
	// distinct presenters both authorize in any observation and
	// operation (two processes), and no presenter authorizes under
	// both host perspectives (two hosts).
	names := []string{"exact", "stale", "loser_low", "loser_high", "future", "foreign"}
	for obsName := range observations {
		for opName := range entries {
			for i := 0; i < len(names); i++ {
				for j := i + 1; j < len(names); j++ {
					if verdicts[obsName][opName][names[i]] == "authorized" &&
						verdicts[obsName][opName][names[j]] == "authorized" {
						t.Fatalf("%s/%s authorizes both %s and %s; at most one owner may pass",
							obsName, opName, names[i], names[j])
					}
				}
			}
		}
	}
	for _, presenterName := range names {
		for opName := range entries {
			if verdicts["local"][opName][presenterName] == "authorized" &&
				verdicts["remote"][opName][presenterName] == "authorized" {
				t.Fatalf("%s/%s authorizes on both hosts; two hosts must never both pass", opName, presenterName)
			}
		}
	}
	t.Logf("gate-single-owner: %d presenters x %d observations x %d operations, exactly one authorizes per local operation",
		len(presenters), len(observations), len(entries))
}
