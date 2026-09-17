package fencing

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// fenceWinner returns the epoch-2 winning summary the terminate tests
// judge staleness against.
func fenceWinner() sessrepo.LeaseSummary {
	return sessrepo.LeaseSummary{SessionID: fenceSessionA, LeaseID: fenceLeaseA, Epoch: 2, HolderHostID: fenceHostA}
}

// TestTerminateStaleAuthorizesFencedTarget proves explicit force
// recovery with preserved diagnostics authorizes terminating a
// fenced process: a lower-epoch target, a same-epoch divergent
// target, and a target from an epoch beyond the winner are all not
// the live owner.
func TestTerminateStaleAuthorizesFencedTarget(t *testing.T) {
	winner := fenceWinner()
	targets := map[string]PresentedToken{
		"lower_epoch":    {SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld},
		"divergent":      {SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseC},
		"beyond_winner":  {SessionID: fenceSessionA, Epoch: 9, LeaseID: fenceLeaseC},
		"prefix_sharing": {SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseC2},
	}
	for name, target := range targets {
		t.Run(name, func(t *testing.T) {
			if err := AuthorizeTerminateStale(target, winner, true, true, true); err != nil {
				t.Fatalf("AuthorizeTerminateStale(%s) error = %v, want authorization", name, err)
			}
		})
	}
}

// TestTerminateStaleRefusesWithoutForce proves terminate-stale
// without explicit force recovery refuses local_precondition_failed,
// even with diagnostics preserved.
func TestTerminateStaleRefusesWithoutForce(t *testing.T) {
	winner := fenceWinner()
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	vectors := map[string]bool{"diagnostics": true, "no_diagnostics": false}
	for name, diagnostics := range vectors {
		t.Run(name, func(t *testing.T) {
			err := AuthorizeTerminateStale(stale, winner, true, false, diagnostics)
			mustRefuse(t, err, ErrLocalPreconditionFailed, "no_force/"+name)
		})
	}
}

// TestTerminateStaleRefusesWithoutDiagnostics proves terminate-stale
// before diagnostics are preserved refuses
// local_precondition_failed, even under explicit force recovery.
func TestTerminateStaleRefusesWithoutDiagnostics(t *testing.T) {
	winner := fenceWinner()
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	vectors := map[string]bool{"force": true, "no_force": false}
	for name, force := range vectors {
		t.Run(name, func(t *testing.T) {
			err := AuthorizeTerminateStale(stale, winner, true, force, false)
			mustRefuse(t, err, ErrLocalPreconditionFailed, "no_diagnostics/"+name)
		})
	}
}

// TestTerminateStaleRefusesWithoutWinner proves staleness cannot be
// judged without an established winner: terminate-stale refuses
// local_precondition_failed.
func TestTerminateStaleRefusesWithoutWinner(t *testing.T) {
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	vectors := map[string]bool{"force": true, "no_force": false}
	for name, force := range vectors {
		t.Run(name, func(t *testing.T) {
			err := AuthorizeTerminateStale(stale, sessrepo.LeaseSummary{}, false, force, true)
			mustRefuse(t, err, ErrLocalPreconditionFailed, "no_winner/"+name)
		})
	}
}

// TestTerminateStaleRefusesLiveOwner proves terminate-stale must
// never terminate the live winning owner: an exact winning target
// refuses local_precondition_failed at both epoch 1 and epoch 2.
func TestTerminateStaleRefusesLiveOwner(t *testing.T) {
	t.Run("epoch_two", func(t *testing.T) {
		winner := fenceWinner()
		live := PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseA}
		err := AuthorizeTerminateStale(live, winner, true, true, true)
		mustRefuse(t, err, ErrLocalPreconditionFailed, "live_owner/epoch_two")
	})
	t.Run("epoch_one", func(t *testing.T) {
		winner := sessrepo.LeaseSummary{SessionID: fenceSessionA, LeaseID: fenceLeaseOld, Epoch: 1, HolderHostID: fenceHostA}
		live := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
		err := AuthorizeTerminateStale(live, winner, true, true, true)
		mustRefuse(t, err, ErrLocalPreconditionFailed, "live_owner/epoch_one")
	})
}

// TestTerminateStaleRefusesForeignSession proves a target bound to
// another session is not fenced by this winner: terminate-stale
// refuses lease_conflict.
func TestTerminateStaleRefusesForeignSession(t *testing.T) {
	winner := fenceWinner()
	vectors := map[string]PresentedToken{
		"prefix_sharing": {SessionID: fenceSessionB, Epoch: 2, LeaseID: fenceLeaseA},
		"fully_foreign":  {SessionID: fenceSessionC, Epoch: 9, LeaseID: fenceLeaseC},
	}
	for name, target := range vectors {
		t.Run(name, func(t *testing.T) {
			err := AuthorizeTerminateStale(target, winner, true, true, true)
			mustRefuse(t, err, ErrLeaseConflict, "foreign/"+name)
		})
	}
}

// TestTerminateStaleRefusesMalformedTarget proves malformed targets
// refuse invalid_arguments before any force or staleness question.
func TestTerminateStaleRefusesMalformedTarget(t *testing.T) {
	winner := fenceWinner()
	vectors := map[string]PresentedToken{
		"session_not_uuid": {SessionID: "not-a-uuid", Epoch: 2, LeaseID: fenceLeaseA},
		"epoch_zero":       {SessionID: fenceSessionA, Epoch: 0, LeaseID: fenceLeaseA},
		"lease_not_uuid":   {SessionID: fenceSessionA, Epoch: 2, LeaseID: "zzz"},
		"lease_empty":      {SessionID: fenceSessionA, Epoch: 2, LeaseID: ""},
	}
	for name, target := range vectors {
		t.Run(name, func(t *testing.T) {
			err := AuthorizeTerminateStale(target, winner, true, true, true)
			mustRefuse(t, err, ErrInvalidArguments, "malformed/"+name)
		})
	}
}

// TestTerminateStaleRefusesGarbageWinner proves staleness is never
// judged against a malformed winner: garbage authority refuses
// invalid_arguments.
func TestTerminateStaleRefusesGarbageWinner(t *testing.T) {
	stale := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	vectors := map[string]sessrepo.LeaseSummary{
		"epoch_zero_lease": {SessionID: fenceSessionA, LeaseID: "zzz", Epoch: 0, HolderHostID: fenceHostA},
		"lease_malformed":  {SessionID: fenceSessionA, LeaseID: "zzz", Epoch: 2, HolderHostID: fenceHostA},
	}
	for name, winner := range vectors {
		t.Run(name, func(t *testing.T) {
			err := AuthorizeTerminateStale(stale, winner, true, true, true)
			mustRefuse(t, err, ErrInvalidArguments, "garbage_winner/"+name)
		})
	}
}
