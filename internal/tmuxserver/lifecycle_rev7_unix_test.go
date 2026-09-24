//go:build !windows

package tmuxserver

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// TestAttachRefusesGenerationRotatedDuringLockWait pins the P1
// effect-boundary generation recheck: the server admission binds
// the request generation, then the generation rotates while the
// attach waits for the barrier lock. The post-lock recheck must
// refuse the stale generation instead of committing a writable
// receipt under it. The rotation lands strictly between the
// admission read and the lock release (channel-ordered), so the
// test is deterministic with no sleeps.
func TestAttachRefusesGenerationRotatedDuringLockWait(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	var changed atomic.Bool
	arrived := make(chan struct{}, 1)
	fx.lc.CurrentGeneration = func() string {
		if changed.Load() {
			return "generation-two"
		}
		return lxGeneration
	}
	admitted := fullLifecycleAdmitted()
	fx.lc.ServerAdmission = func(context.Context) (RealmAdmission, error) {
		generation := fx.lc.CurrentGeneration()
		arrived <- struct{}{}
		return RealmAdmission{Admitted: admitted, RawGeneration: generation}, nil
	}
	fx.lc.barrierMu.Lock()
	type attachResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan attachResult, 1)
	body := lxAttachBody(t, nil)
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: body, Source: "parked", Admitted: fullLifecycleAdmitted(),
		})
		done <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-arrived:
	case <-time.After(10 * time.Second):
		fx.lc.barrierMu.Unlock()
		t.Fatal("attach never reached server admission")
	}
	changed.Store(true)
	fx.lc.barrierMu.Unlock()
	var res attachResult
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("attach never returned after unlock")
	}
	requireLocalCode(t, res.err, "terminal_backend_stale_generation", "backend_generation stale")
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt", found, err)
	}
	if len(res.outcome.Argv) != 0 {
		t.Fatalf("refused attach built argv %q", res.outcome.Argv)
	}
	if res.outcome.Attach != nil && res.outcome.Attach.InputAuthorized {
		t.Fatalf("refused attach authorized input: %+v", res.outcome.Attach)
	}
}

// TestAttachRefusesDeadlineExpiredDuringAdmission pins the P2-A
// effect-boundary deadline recheck: the server admission advances
// the clock past the operation deadline (18:00) while the
// authorization stays valid (until the next day). The post-lock
// recheck must refuse the timeout — the operation expired even
// though its authorization did not — and commit no receipt.
func TestAttachRefusesDeadlineExpiredDuringAdmission(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	admitted := fullLifecycleAdmitted()
	fx.lc.ServerAdmission = func(context.Context) (RealmAdmission, error) {
		fx.clock.Sleep(7 * time.Hour)
		return RealmAdmission{Admitted: admitted, RawGeneration: lxGeneration}, nil
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt", found, err)
	}
	if len(outcome.Argv) != 0 {
		t.Fatalf("refused attach built argv %q", outcome.Argv)
	}
	if outcome.Attach != nil && outcome.Attach.InputAuthorized {
		t.Fatalf("refused attach authorized input: %+v", outcome.Attach)
	}
}

// TestAttachRefusesDeadlineOnTheInstantDuringAdmission pins the
// strict bound at the same recheck: now equal to the deadline
// when the lock releases refuses exactly like a late instant.
func TestAttachRefusesDeadlineOnTheInstantDuringAdmission(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	admitted := fullLifecycleAdmitted()
	fx.lc.ServerAdmission = func(context.Context) (RealmAdmission, error) {
		fx.clock.Sleep(6 * time.Hour)
		return RealmAdmission{Admitted: admitted, RawGeneration: lxGeneration}, nil
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt", found, err)
	}
	if len(outcome.Argv) != 0 {
		t.Fatalf("refused attach built argv %q", outcome.Argv)
	}
	if outcome.Attach != nil && outcome.Attach.InputAuthorized {
		t.Fatalf("refused attach authorized input: %+v", outcome.Attach)
	}
}

// TestAttachAuthorizationMembersRefuseWithoutReceipt pins the
// composed attach authorization negative: each member the landed
// gate binds — expiry, input, transport — refuses through the
// production attach entry with no durable receipt. The input
// member is the P2-B witness: asserting the returned error alone
// is not enough, because a weakened input arm commits the
// writable receipt before the result binding refuses — so the
// test asserts the receipt absence as well as the refusal.
func TestAttachAuthorizationMembersRefuseWithoutReceipt(t *testing.T) {
	for _, member := range []string{"expiry", "input", "transport"} {
		t.Run(member, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			body := lxAttachBody(t, func(object map[string]any) {
				auth := lxAttachAuth("local_only", true)
				switch member {
				case "expiry":
					auth["expires_at"] = fx.clock.Now().Format("2006-01-02T15:04:05.000Z")
				case "input":
					auth["input_authorized"] = false
				case "transport":
					auth["transport"] = "trusted_private_mesh"
				}
				object["authorization"] = auth
			})
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "attach", Body: body, Source: "parked", Admitted: fullLifecycleAdmitted(),
			})
			_, found, lookupErr := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient)
			if lookupErr != nil {
				t.Fatal(lookupErr)
			}
			if err == nil || found {
				t.Fatalf("unauthorized effect: member=%s err=%v receipt=%v attach=%+v", member, err, found, outcome.Attach)
			}
			requireLandedCode(t, err, "terminal_backend_unauthorized")
			if outcome.Attach != nil && outcome.Attach.InputAuthorized {
				t.Fatalf("refused attach authorized input: %+v", outcome.Attach)
			}
		})
	}
}

// TestExecuteTerminateRefusesAuthorizationExpiredAfterReceipt pins
// the per-effect authorization recheck as a negative witness: the
// authorization is valid at entry (12:00, expiring 13:00) and
// expires before the first effect runs (the receipt hook advances
// the clock to 14:00, still before the 18:00 deadline). The loop
// must refuse before any effect executes — zero tmux commands —
// rather than running the kill under the lapsed grant.
func TestExecuteTerminateRefusesAuthorizationExpiredAfterReceipt(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	raw := lxTerminateBody(t, func(object map[string]any) {
		object["context"].(map[string]any)["authorization"].(map[string]any)["expires_at"] = "2026-09-01T13:00:00.000Z"
	})
	presented, winner := terminateFencing()
	fx.lc.Hooks = &terminstance.EngineHooks{
		AfterReceipt: func(terminalbackend.Operation, string, terminalbackend.InstanceState) {
			fx.clock.Sleep(2 * time.Hour)
		},
	}
	fx.runner.queue("kill-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: raw, Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "terminal_backend_unauthorized", "lifecycle authorization recheck")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateStaleFenced {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteRestoreRefusesAuthorizationExpiredAfterReceipt pins the
// same per-effect recheck through the restore entry: the
// authorization lapses between the receipt and the first effect,
// so the loop refuses with nothing committed instead of running
// the persist under the stale grant.
func TestExecuteRestoreRefusesAuthorizationExpiredAfterReceipt(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	raw := lxRestoreBody(t, lxDigestA, func(object map[string]any) {
		object["context"].(map[string]any)["authorization"].(map[string]any)["expires_at"] = "2026-09-01T13:00:00.000Z"
	})
	fx.lc.Hooks = &terminstance.EngineHooks{
		AfterReceipt: func(terminalbackend.Operation, string, terminalbackend.InstanceState) {
			fx.clock.Sleep(2 * time.Hour)
		},
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "terminal_backend_unauthorized", "lifecycle authorization recheck")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateAbsent {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if _, found, err := fx.lc.Receipts.Completed(lxCreateKey()); err != nil || found {
		t.Fatalf("completed = %v, %v; want no completion", found, err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteEntryRefusesWrongAuthorizationKind pins the entry
// authorization kind as a genuinely admitting gate: a
// control-kind body refuses at the entry check before any store
// side effect — the effect loop would refuse the same verdict,
// but only the entry arm leaves the key unbound. A mutant
// rewiring the entry to the control kind admits the body past
// the entry, binds the receipt, and refuses only at the loop, so
// the no-bind assertion reddens.
func TestExecuteEntryRefusesWrongAuthorizationKind(t *testing.T) {
	t.Run("terminate", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		raw := lxTerminateBody(t, func(object map[string]any) {
			object["context"].(map[string]any)["authorization"].(map[string]any)["authorization_kind"] = "control"
		})
		presented, winner := terminateFencing()
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "terminate-stale", Body: raw, Source: "stale_fenced",
			Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		})
		requireEngineCode(t, err, "terminal_backend_unauthorized")
		if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateStaleFenced {
			t.Fatalf("mutation = %+v", outcome.Mutation)
		}
		if _, found, err := fx.lc.Receipts.Lookup(lxTerminateKey()); err != nil || found {
			t.Fatalf("lookup = %v, %v; want no receipt", found, err)
		}
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
		}
	})
	t.Run("restore", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		raw := lxRestoreBody(t, lxDigestA, func(object map[string]any) {
			object["context"].(map[string]any)["authorization"].(map[string]any)["authorization_kind"] = "control"
		})
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		requireEngineCode(t, err, "terminal_backend_unauthorized")
		if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateAbsent {
			t.Fatalf("mutation = %+v", outcome.Mutation)
		}
		if _, found, err := fx.lc.Receipts.Lookup(lxCreateKey()); err != nil || found {
			t.Fatalf("lookup = %v, %v; want no receipt", found, err)
		}
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused restore execed %d commands", fx.runner.callCount())
		}
	})
}
