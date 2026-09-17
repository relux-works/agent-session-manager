package fencing

import (
	"errors"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// TestAuthorizeAdmitsWinnerExactEpochAllOperations drives the positive
// row of every gate: the winning lease with the exact epoch mints a
// sealed token through each of the five production entries, and the
// token binds the exact triple to quiesce, capture, and materialize.
func TestAuthorizeAdmitsWinnerExactEpochAllOperations(t *testing.T) {
	for name, entry := range fenceEntries() {
		t.Run(name, func(t *testing.T) {
			token, err := entry.authorize(fencePresented(), fenceObservation())
			mustMint(t, token, err, name)
		})
	}
}

// TestAuthorizeRefusesUnknownOperation proves the operation enum is
// closed: an action outside the five gated entries refuses
// invalid_arguments instead of authorizing.
func TestAuthorizeRefusesUnknownOperation(t *testing.T) {
	_, err := Authorize("bogus", fencePresented(), fenceObservation())
	mustRefuse(t, err, ErrInvalidArguments, "Authorize(bogus)")
}

// TestAuthorizeRefusesMalformedPresented drives every malformed
// presented-token vector through a launch and a non-launch entry:
// malformed claims refuse invalid_arguments on both, never a fencing
// class and never a park. The epoch-zero vectors pair the winner
// lease (which the epoch gate would stale-report past the grammar),
// a well-formed third lease (which proves the epoch gate fires
// regardless of lease identity), and a malformed lease (which
// refuses at grammar under mutants).
func TestAuthorizeRefusesMalformedPresented(t *testing.T) {
	vectors := map[string]PresentedToken{
		"session_not_uuid":       {SessionID: "not-a-uuid", Epoch: 2, LeaseID: fenceLeaseA},
		"session_empty":          {SessionID: "", Epoch: 2, LeaseID: fenceLeaseA},
		"epoch_zero":             {SessionID: fenceSessionA, Epoch: 0, LeaseID: fenceLeaseA},
		"epoch_zero_other_lease": {SessionID: fenceSessionA, Epoch: 0, LeaseID: fenceLeaseC},
		"epoch_zero_lease":       {SessionID: fenceSessionA, Epoch: 0, LeaseID: "zzz"},
		"lease_not_uuid":         {SessionID: fenceSessionA, Epoch: 2, LeaseID: "zzz"},
		"lease_empty":            {SessionID: fenceSessionA, Epoch: 2, LeaseID: ""},
	}
	for name, entry := range fenceEntries() {
		for vector, presented := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				_, err := entry.authorize(presented, fenceObservation())
				mustRefuse(t, err, ErrInvalidArguments, name+"/"+vector)
			})
		}
	}
}

// TestAuthorizeRefusesGarbageWinner proves the winner is not trusted
// past its grammar: a caller-built observation with a malformed
// winning summary refuses invalid_arguments instead of authorizing
// against garbage.
func TestAuthorizeRefusesGarbageWinner(t *testing.T) {
	vectors := map[string]sessrepo.LeaseSummary{
		// The narrowing vector the winner mutant admits past the arm.
		"epoch_zero_lease": {SessionID: fenceSessionA, LeaseID: "zzz", Epoch: 0, HolderHostID: fenceHostA},
		// A second garbage shape that still refuses under the mutant.
		"holder_malformed": {SessionID: fenceSessionA, LeaseID: fenceLeaseA, Epoch: 2, HolderHostID: "zzz"},
	}
	for name, entry := range fenceEntries() {
		for vector, winner := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				observation := fenceObservation()
				observation.Winner = winner
				_, err := entry.authorize(fencePresented(), observation)
				mustRefuse(t, err, ErrInvalidArguments, name+"/"+vector)
			})
		}
	}
}

// TestAuthorizeRefusesForeignSession drives the foreign-session row:
// a token minted for another session never authorizes here. The B
// session shares the first UUID segment with A (the vector the
// prefix mutant admits); the C session shares nothing.
func TestAuthorizeRefusesForeignSession(t *testing.T) {
	vectors := map[string]string{
		"prefix_sharing": fenceSessionB,
		"fully_foreign":  fenceSessionC,
	}
	for name, entry := range fenceEntries() {
		for vector, session := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				presented := fencePresented()
				presented.SessionID = session
				_, err := entry.authorize(presented, fenceObservation())
				mustRefuse(t, err, ErrLeaseConflict, name+"/"+vector)
			})
		}
	}
}

// TestAuthorizeParksAndRefusesAbsentWinner drives the absent-lease
// row: with no winning lease established, launch entries park with
// restore_policy and an empty winning lease ID, while input,
// mutation, and checkpoint refuse lease_conflict.
func TestAuthorizeParksAndRefusesAbsentWinner(t *testing.T) {
	vectors := map[string]bool{"verified": true, "unverified": false}
	for name, entry := range fenceEntries() {
		for vector, verified := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				observation := fenceObservation()
				observation.HasWinner = false
				observation.Winner = sessrepo.LeaseSummary{}
				observation.Verified = verified
				_, err := entry.authorize(fencePresented(), observation)
				if entry.launch {
					mustPark(t, err, ParkRestorePolicy, "", ErrLeaseConflict, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrLeaseConflict, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizeParksAndRefusesUnverified drives the unverified-sync
// row: without a mesh lease refresh proving current knowledge,
// launch entries park with restore_policy under the winner, while
// the other entries refuse lease_conflict.
func TestAuthorizeParksAndRefusesUnverified(t *testing.T) {
	vectors := map[string]bool{"unambiguous": false, "ambiguous": true}
	for name, entry := range fenceEntries() {
		for vector, ambiguous := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				observation := fenceObservation()
				observation.Verified = false
				observation.Ambiguous = ambiguous
				_, err := entry.authorize(fencePresented(), observation)
				if entry.launch {
					mustPark(t, err, ParkRestorePolicy, fenceLeaseA, ErrLeaseConflict, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrLeaseConflict, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizeParksAndRefusesAmbiguous drives the ambiguous-ownership
// row: conflicting union observations park or refuse exactly like
// unverified sync, naming the winner the park stands under.
func TestAuthorizeParksAndRefusesAmbiguous(t *testing.T) {
	vectors := map[string]bool{"verified": true, "unverified": false}
	for name, entry := range fenceEntries() {
		for vector, verified := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				observation := fenceObservation()
				observation.Ambiguous = true
				observation.Verified = verified
				_, err := entry.authorize(fencePresented(), observation)
				if entry.launch {
					mustPark(t, err, ParkRestorePolicy, fenceLeaseA, ErrLeaseConflict, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrLeaseConflict, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizeParksAndRefusesFailedHandoff drives the failed-handoff
// row: a caller whose own handoff already failed parks with
// failed_handoff on launch entries instead of activating its losing
// lease, and refuses lease_conflict elsewhere. The unverified vector
// dies at the earlier verifiability arm instead, pinning the
// precedence.
func TestAuthorizeParksAndRefusesFailedHandoff(t *testing.T) {
	for name, entry := range fenceEntries() {
		t.Run(name+"/verified", func(t *testing.T) {
			observation := fenceObservation()
			observation.HandoffFailed = true
			_, err := entry.authorize(fencePresented(), observation)
			if entry.launch {
				mustPark(t, err, ParkFailedHandoff, fenceLeaseA, ErrLeaseConflict, name+"/verified")
			} else {
				mustRefuse(t, err, ErrLeaseConflict, name+"/verified")
			}
		})
		t.Run(name+"/unverified", func(t *testing.T) {
			observation := fenceObservation()
			observation.HandoffFailed = true
			observation.Verified = false
			_, err := entry.authorize(fencePresented(), observation)
			if entry.launch {
				mustPark(t, err, ParkRestorePolicy, fenceLeaseA, ErrLeaseConflict, name+"/unverified")
			} else {
				mustRefuse(t, err, ErrLeaseConflict, name+"/unverified")
			}
		})
	}
}

// TestAuthorizeRefusesMissingGrant proves a gate call without the
// process-local fencing grant is malformed: the caller validates the
// winning token first. The unverified vector dies at the earlier
// verifiability arm instead, pinning the precedence.
func TestAuthorizeRefusesMissingGrant(t *testing.T) {
	for name, entry := range fenceEntries() {
		t.Run(name+"/verified", func(t *testing.T) {
			observation := fenceObservation()
			observation.HasGrant = false
			observation.Grant = sessrepo.FencingGrant{}
			_, err := entry.authorize(fencePresented(), observation)
			mustRefuse(t, err, ErrInvalidArguments, name+"/verified")
		})
		t.Run(name+"/unverified", func(t *testing.T) {
			observation := fenceObservation()
			observation.HasGrant = false
			observation.Grant = sessrepo.FencingGrant{}
			observation.Verified = false
			_, err := entry.authorize(fencePresented(), observation)
			if entry.launch {
				mustPark(t, err, ParkRestorePolicy, fenceLeaseA, ErrLeaseConflict, name+"/unverified")
			} else {
				mustRefuse(t, err, ErrLeaseConflict, name+"/unverified")
			}
		})
	}
}

// TestAuthorizeRefusesMissingClock proves expiry cannot be bypassed
// with a missing clock reading: a grant without Now refuses
// invalid_arguments instead of reading as fresh.
func TestAuthorizeRefusesMissingClock(t *testing.T) {
	vectors := map[string]time.Time{
		"zero_validated": {},
		"set_validated":  fenceValidatedAt,
	}
	for name, entry := range fenceEntries() {
		for vector, validated := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				observation := fenceObservation()
				observation.Now = time.Time{}
				observation.Grant.ValidatedAt = validated
				_, err := entry.authorize(fencePresented(), observation)
				mustRefuse(t, err, ErrInvalidArguments, name+"/"+vector)
			})
		}
	}
}

// TestAuthorizeRefusesExpiredGrant drives the expired-grant row
// through the sessrepo expiry owner: a lapsed grant refuses
// lease_conflict on every entry so the holder revalidates, a grant
// validated exactly at the interval boundary is still current, and
// an unusable refresh policy refuses invalid_arguments.
func TestAuthorizeRefusesExpiredGrant(t *testing.T) {
	t.Run("lapsed", func(t *testing.T) {
		for name, entry := range fenceEntries() {
			t.Run(name, func(t *testing.T) {
				observation := fenceObservation()
				observation.Now = fenceValidatedAt.Add(time.Minute + time.Second)
				_, err := entry.authorize(fencePresented(), observation)
				mustRefuse(t, err, ErrLeaseConflict, name+"/lapsed")
			})
		}
	})
	t.Run("never_validated", func(t *testing.T) {
		for name, entry := range fenceEntries() {
			t.Run(name, func(t *testing.T) {
				observation := fenceObservation()
				observation.Grant.ValidatedAt = time.Time{}
				_, err := entry.authorize(fencePresented(), observation)
				mustRefuse(t, err, ErrLeaseConflict, name+"/never_validated")
			})
		}
	})
	t.Run("boundary_fresh", func(t *testing.T) {
		for name, entry := range fenceEntries() {
			t.Run(name, func(t *testing.T) {
				observation := fenceObservation()
				observation.Now = fenceValidatedAt.Add(time.Minute)
				token, err := entry.authorize(fencePresented(), observation)
				mustMint(t, token, err, name+"/boundary_fresh")
			})
		}
	})
	t.Run("zero_policy", func(t *testing.T) {
		for name, entry := range fenceEntries() {
			t.Run(name, func(t *testing.T) {
				observation := fenceObservation()
				observation.Policy = sessrepo.FencingPolicy{}
				_, err := entry.authorize(fencePresented(), observation)
				mustRefuse(t, err, ErrInvalidArguments, name+"/zero_policy")
			})
		}
	})
}

// TestAuthorizeParksAndRefusesRemote drives the remote-ownership row:
// when the winning holder is not local, launch entries park with
// remote_owner under the winner, while the other entries refuse
// not_owner. Host B shares the first UUID segment with the local
// host (the vector the prefix mutant admits); host C shares nothing.
func TestAuthorizeParksAndRefusesRemote(t *testing.T) {
	vectors := map[string]string{"prefix_sharing": fenceHostB, "fully_remote": fenceHostC}
	for name, entry := range fenceEntries() {
		for vector, holder := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				observation := fenceObservation()
				observation.Winner.HolderHostID = holder
				_, err := entry.authorize(fencePresented(), observation)
				if entry.launch {
					mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrNotOwner, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizeRefusesLowerEpoch drives the lower-epoch row: a token
// from a superseded epoch refuses stale_owner on input, mutation,
// and checkpoint, and parks with stale_owner on launch entries. The
// exact-lease vector carries the winning lease at the lower epoch
// (which the weakened epoch gate would mint); the standard vector
// carries the old superseded lease.
func TestAuthorizeRefusesLowerEpoch(t *testing.T) {
	vectors := map[string]PresentedToken{
		"standard":    {SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld},
		"exact_lease": {SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseA},
	}
	for name, entry := range fenceEntries() {
		for vector, presented := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				_, err := entry.authorize(presented, fenceObservation())
				if entry.launch {
					mustPark(t, err, ParkStaleOwner, fenceLeaseA, ErrStaleOwner, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrStaleOwner, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizeRefusesEpochBeyondWinner drives the epoch-ahead row: a
// token from an epoch beyond the winner names no known lease. Launch
// entries park with restore_policy because the future token is
// unverifiable; the other entries refuse lease_conflict.
func TestAuthorizeRefusesEpochBeyondWinner(t *testing.T) {
	vectors := map[string]PresentedToken{
		"exact_lease": {SessionID: fenceSessionA, Epoch: 9, LeaseID: fenceLeaseA},
		"other_lease": {SessionID: fenceSessionA, Epoch: 9, LeaseID: fenceLeaseC},
	}
	for name, entry := range fenceEntries() {
		for vector, presented := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				_, err := entry.authorize(presented, fenceObservation())
				if entry.launch {
					mustPark(t, err, ParkRestorePolicy, fenceLeaseA, ErrLeaseConflict, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrLeaseConflict, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizeRefusesSameEpochLoser drives the same-epoch-loser row:
// a token that loses the lease tie at the winning epoch refuses
// lease_conflict on input, mutation, and checkpoint, and parks with
// stale_owner on launch entries. Lease C2 shares the first UUID
// segment with the winner (the vector the prefix mutant admits);
// lease C shares nothing (the vector the carve-out mutant admits).
func TestAuthorizeRefusesSameEpochLoser(t *testing.T) {
	vectors := map[string]string{"standard": fenceLeaseC, "prefix_sharing": fenceLeaseC2}
	for name, entry := range fenceEntries() {
		for vector, lease := range vectors {
			t.Run(name+"/"+vector, func(t *testing.T) {
				presented := fencePresented()
				presented.LeaseID = lease
				_, err := entry.authorize(presented, fenceObservation())
				if entry.launch {
					mustPark(t, err, ParkStaleOwner, fenceLeaseA, ErrLeaseConflict, name+"/"+vector)
				} else {
					mustRefuse(t, err, ErrLeaseConflict, name+"/"+vector)
				}
			})
		}
	}
}

// TestAuthorizePrecedence pins the arm order where two arms could
// fire: the operative ownership fact beats the presented credential,
// and malformed calls die before any ownership question is asked.
func TestAuthorizePrecedence(t *testing.T) {
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	t.Run("direction_beats_credential", func(t *testing.T) {
		observation := fenceObservation()
		observation.Winner.HolderHostID = fenceHostC
		_, err := AuthorizeActivation(stale, observation)
		mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, "activation/direction_beats_credential")
		_, err = AuthorizeMutation(stale, observation)
		mustRefuse(t, err, ErrNotOwner, "mutation/direction_beats_credential")
	})
	t.Run("verifiability_beats_credential", func(t *testing.T) {
		observation := fenceObservation()
		observation.Verified = false
		_, err := AuthorizeRestore(stale, observation)
		mustPark(t, err, ParkRestorePolicy, fenceLeaseA, ErrLeaseConflict, "restore/verifiability_beats_credential")
		_, err = AuthorizeInput(stale, observation)
		mustRefuse(t, err, ErrLeaseConflict, "input/verifiability_beats_credential")
	})
	t.Run("grammar_beats_ownership", func(t *testing.T) {
		observation := fenceObservation()
		observation.Winner.HolderHostID = fenceHostC
		malformed := PresentedToken{SessionID: "not-a-uuid", Epoch: 2, LeaseID: fenceLeaseA}
		_, err := AuthorizeActivation(malformed, observation)
		mustRefuse(t, err, ErrInvalidArguments, "activation/grammar_beats_ownership")
	})
	t.Run("absence_beats_stale_token", func(t *testing.T) {
		observation := fenceObservation()
		observation.HasWinner = false
		observation.Winner = sessrepo.LeaseSummary{}
		_, err := AuthorizeActivation(stale, observation)
		mustPark(t, err, ParkRestorePolicy, "", ErrLeaseConflict, "activation/absence_beats_stale_token")
	})
}

// TestObserveLoadsWinningLease drives the repository-backed
// observation entry: the winner is the store head, a session with no
// leases yields no winner without an error, and an unknown session
// propagates the repository refusal.
func TestObserveLoadsWinningLease(t *testing.T) {
	t.Run("winner_is_store_head", func(t *testing.T) {
		repository, _ := openFenceRepository(t)
		createFenceSession(t, repository)
		first := mustCreateFenceLease(t, repository, fenceSessionA, sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt})
		second := mustSwapFenceLease(t, repository, fenceSessionA, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
			CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: fenceLeaseA, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt},
			Reason:           "graceful_takeover",
			CheckpointID:     fenceZeroDigest,
		})
		observation, err := Observe(repository, fenceSessionA, ObserveInput{LocalHostID: fenceHostA, Verified: true, HasGrant: true, Grant: sessrepo.FencingGrant{SessionID: fenceSessionA, ValidatedAt: fenceValidatedAt}, Policy: fencePolicy, Now: fenceValidatedAt})
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		if !observation.HasWinner {
			t.Fatal("Observe reports no winner past a succession")
		}
		if observation.Winner.Epoch != 2 || observation.Winner.LeaseID != fenceLeaseA || observation.Winner.HolderHostID != fenceHostA || observation.Winner.RecordID != second.RecordID {
			t.Fatalf("Observe winner = %+v, want the store head %+v", observation.Winner, second)
		}
		token, err := AuthorizeMutation(fencePresented(), observation)
		mustMint(t, token, err, "Observe/AuthorizeMutation")
	})
	t.Run("no_leases_yields_no_winner", func(t *testing.T) {
		repository, _ := openFenceRepository(t)
		createFenceSession(t, repository)
		observation, err := Observe(repository, fenceSessionA, ObserveInput{LocalHostID: fenceHostA, Verified: true})
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		if observation.HasWinner {
			t.Fatalf("Observe winner = %+v, want no winner", observation.Winner)
		}
	})
	t.Run("unknown_session_propagates", func(t *testing.T) {
		repository, _ := openFenceRepository(t)
		_, err := Observe(repository, fenceSessionA, ObserveInput{LocalHostID: fenceHostA, Verified: true})
		if err == nil {
			t.Fatal("Observe(unknown session) succeeded")
		}
		if !errors.Is(err, sessrepo.ErrUnknownSession) {
			t.Fatalf("Observe(unknown session) error = %v, want the repository refusal", err)
		}
	})
}
