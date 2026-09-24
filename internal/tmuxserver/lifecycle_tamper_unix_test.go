//go:build !windows

package tmuxserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// lxLeaseC is a third lease identity: neither the current (lxLease)
// nor the stale target (lxLeaseB). Winner-tracking tests rotate the
// live lease to it so a comparison against a captured constant would
// accept the usually-winning value.
const lxLeaseC = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

// stubPerformer is a scripted effectPerformer: it reports the canned
// evidence ID (or error) for every effect and counts calls.
type stubPerformer struct {
	id    string
	err   error
	calls int
}

func (stub *stubPerformer) PerformEffect(_ context.Context, _ terminalbackend.SideEffect) (string, error) {
	stub.calls++
	return stub.id, stub.err
}

// TestRunLifecycleEffectsRefusesForgedEvidence pins the digest arm of
// the receipt-bound effect loop: a backend that reports a
// non-digest evidence ID fails the run with the integrity failure,
// not a commit. The failure lands pre-commit (after equals source),
// binds no completion, and records the source in the AX-side memory,
// so a retry reconciles instead of replaying a forgery.
func TestRunLifecycleEffectsRefusesForgedEvidence(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	deadline, err := time.Parse(time.RFC3339Nano, lxDeadline)
	if err != nil {
		t.Fatal(err)
	}
	expires, err := scalar.ParseTimestamp(lxExpires)
	if err != nil {
		t.Fatal(err)
	}
	mctx := terminstance.MutationContext{
		OperationID:           lxOperation,
		SessionID:             lxSession,
		TerminalInstanceID:    lxInstance,
		TerminalBackendID:     "ax.tmux",
		ImplementationVersion: lxImpl,
		ProtocolVersion:       lxProto,
		BackendGeneration:     lxGeneration,
		IdempotencyKey:        lxTerminateKey(),
		Authorization: terminstance.AXAuthorization{
			LeaseID:    lxLease,
			LeaseEpoch: 7,
			Kind:       terminstance.AuthorizationForceStale,
			ExpiresAt:  expires,
		},
	}
	stub := &stubPerformer{id: "not-a-digest"}
	outcome, err := fx.lc.runLifecycleEffects(
		context.Background(),
		terminalbackend.OperationTerminateStale,
		mctx,
		terminalbackend.StateStaleFenced,
		terminalbackend.StateStopped,
		[]terminalbackend.SideEffect{terminalbackend.EffectStaleIncarnationTerminated},
		deadline,
		terminstance.AuthorizationForceStale,
		stub,
		func(terminstance.Result) OpOutcome {
			t.Fatal("forged evidence completed")
			return OpOutcome{}
		},
	)
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "backend effect evidence")
	if stub.calls != 1 {
		t.Fatalf("calls = %d, want 1", stub.calls)
	}
	if outcome.Mutation == nil {
		t.Fatal("forged-evidence refusal carries no mutation")
	}
	if outcome.Mutation.After != terminalbackend.StateStaleFenced {
		t.Fatalf("after = %s, want stale_fenced", outcome.Mutation.After)
	}
	if len(outcome.Mutation.Effects) != 0 || len(outcome.Mutation.EvidenceIDs) != 0 {
		t.Fatalf("forged run performed %+v", outcome.Mutation)
	}
	if _, found, err := fx.lc.Receipts.Completed(lxTerminateKey()); err != nil || found {
		t.Fatalf("completed = %v, %v; want no completion", found, err)
	}
	state, _, _ := fx.lc.States.Lookup(lxInstance)
	if state != terminalbackend.StateStaleFenced {
		t.Fatalf("memory = %s, want stale_fenced", state)
	}
}

// terminateWithLease rebuilds the terminate request against a rotated
// live lease: the body authorization follows the new current lease
// and the caller-observed winner is the argument.
func terminateWithLease(t *testing.T, lease terminstance.LeaseView, winner sessrepo.LeaseSummary) OpRequest {
	t.Helper()
	presented, _ := terminateFencing()
	raw := lxTerminateBody(t, func(object map[string]any) {
		nested := object["context"].(map[string]any)["authorization"].(map[string]any)
		nested["lease_id"] = lease.LeaseID
		nested["lease_epoch"] = float64(lease.Epoch)
	})
	return OpRequest{
		Operation: "terminate-stale", Body: raw, Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	}
}

// TestExecuteTerminateWinnerTracksCurrentLease proves the winner
// binding reads the live current lease on every call: the lease
// identity is its own arm (epoch agreement alone never authorizes),
// the usually-winning value refuses once the lease rotates past it,
// and a rotated lease authorizes its own winner. Every refusal lands
// before any exec.
func TestExecuteTerminateWinnerTracksCurrentLease(t *testing.T) {
	t.Run("winner-lease-identity", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		presented, _ := terminateFencing()
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
			Admitted: fullLifecycleAdmitted(), Presented: presented,
			Winner:    sessrepo.LeaseSummary{SessionID: lxSession, LeaseID: lxLeaseB, Epoch: 7},
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		})
		requireLocalCode(t, err, "local_precondition_failed", "terminate winner binding")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
		}
	})

	t.Run("winner-must-track-current", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		rotated := terminstance.LeaseView{LeaseID: lxLeaseC, Epoch: 3}
		fx.lc.CurrentLease = func() terminstance.LeaseView { return rotated }
		usually := sessrepo.LeaseSummary{SessionID: lxSession, LeaseID: lxLease, Epoch: 7}
		_, err := fx.lc.Execute(context.Background(), terminateWithLease(t, rotated, usually))
		requireLocalCode(t, err, "local_precondition_failed", "terminate winner binding")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
		}
	})

	t.Run("rotated-current-succeeds", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		rotated := terminstance.LeaseView{LeaseID: lxLeaseC, Epoch: 3}
		fx.lc.CurrentLease = func() terminstance.LeaseView { return rotated }
		fx.runner.queue("kill-session", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
		winner := sessrepo.LeaseSummary{SessionID: lxSession, LeaseID: lxLeaseC, Epoch: 3}
		outcome, err := fx.lc.Execute(context.Background(), terminateWithLease(t, rotated, winner))
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Mutation.After != terminalbackend.StateStopped || !outcome.TargetClosed {
			t.Fatalf("outcome = %+v", outcome)
		}
	})
}

// tamperImage builds the valid completed terminate image the tamper
// rows mutate one member of.
func tamperImage() terminstance.Result {
	return terminstance.Result{
		OperationID:           lxOperation,
		SessionID:             lxSession,
		TerminalInstanceID:    lxInstance,
		TerminalBackendID:     "ax.tmux",
		ImplementationVersion: lxImpl,
		ProtocolVersion:       lxProto,
		BackendGeneration:     lxGeneration,
		Before:                terminalbackend.StateStaleFenced,
		After:                 terminalbackend.StateStopped,
		Disposition:           terminstance.DispositionReplaySame,
	}
}

// TestExecuteTerminateReplayRefusesTamperedImage pins the replay
// path's tamper-evidence member by member: every binding the stored
// completion repeats is rechecked against the request context, and a
// completion that disagrees on any one member refuses with that
// member's detail instead of replaying. The refusal carries the
// mutation with the unavailable after-state, and no tampered replay
// execs.
func TestExecuteTerminateReplayRefusesTamperedImage(t *testing.T) {
	rows := []struct {
		name   string
		tamper func(*terminstance.Result)
		code   string
		detail string
	}{
		{"operation", func(image *terminstance.Result) { image.OperationID = lxBootstrap }, "terminal_backend_protocol_error", "result operation binding"},
		{"session", func(image *terminstance.Result) { image.SessionID = lxSessionB }, "terminal_backend_protocol_error", "result session binding"},
		{"instance", func(image *terminstance.Result) { image.TerminalInstanceID = "0198f4c8-8e50-7f66-8f70-222222222222" }, "terminal_backend_protocol_error", "result instance binding"},
		{"backend", func(image *terminstance.Result) { image.TerminalBackendID = "ax.conpty" }, "terminal_backend_protocol_error", "result backend binding"},
		{"implementation", func(image *terminstance.Result) { image.ImplementationVersion = "9.9.9" }, "terminal_backend_protocol_error", "result implementation binding"},
		{"protocol", func(image *terminstance.Result) { image.ProtocolVersion = "9.9.9" }, "terminal_backend_protocol_error", "result protocol binding"},
		{"generation", func(image *terminstance.Result) { image.BackendGeneration = "generation-two" }, "terminal_backend_stale_generation", "result generation binding"},
		{"before", func(image *terminstance.Result) { image.Before = terminalbackend.StateActive }, "terminal_backend_protocol_error", "result before state"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			image := tamperImage()
			row.tamper(&image)
			encoded, err := json.Marshal(image)
			if err != nil {
				t.Fatal(err)
			}
			key := lxTerminateKey()
			if _, replayed, err := fx.lc.Receipts.Bind(key, terminalbackend.OperationTerminateStale, lxOperation); err != nil || replayed {
				t.Fatalf("plant bind = replayed %v, %v", replayed, err)
			}
			if err := fx.lc.Receipts.Complete(key, encoded); err != nil {
				t.Fatal(err)
			}
			presented, winner := terminateFencing()
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
				Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
				HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
			})
			if err == nil {
				t.Fatalf("tampered %s replayed: %+v", row.name, outcome.Mutation)
			}
			var engine *terminstance.Error
			if !errors.As(err, &engine) {
				t.Fatalf("want *terminstance.Error, got %T (%v)", err, err)
			}
			if engine.Code != row.code || engine.Detail != row.detail {
				t.Fatalf("want %s at %s, got %s at %s", row.code, row.detail, engine.Code, engine.Detail)
			}
			if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateUnavailable {
				t.Fatalf("mutation = %+v", outcome.Mutation)
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("tampered replay execed %d commands", fx.runner.callCount())
			}
		})
	}
}

// TestExecuteTerminateTargetSessionBinds pins the session arm of the
// target binding: a presented token for another session — even with a
// winner that agrees with the token — refuses, so the fencing
// decision cannot authorize a different session than the key names.
func TestExecuteTerminateTargetSessionBinds(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted:  fullLifecycleAdmitted(),
		Presented: fencing.PresentedToken{SessionID: lxSessionB, Epoch: 9, LeaseID: lxLeaseB},
		Winner:    sessrepo.LeaseSummary{SessionID: lxSessionB, LeaseID: lxLease, Epoch: 7},
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "local_precondition_failed", "terminate target binding")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteTerminateRefusesProviderOnlyCapability pins the stale
// arm of the dual conditional: provider observation alone confers
// the operation (landed CheckOperation passes) but never authorizes
// the kill without stale_process_termination.
func TestExecuteTerminateRefusesProviderOnlyCapability(t *testing.T) {
	fx := newLifecycleFixture(t, admittedWith("provider_process_observation"))
	presented, winner := terminateFencing()
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: admittedWith("provider_process_observation"), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteRestoreRefusesWithoutRebootCapability pins the reboot
// conditional: the credential realm alone confers restore (landed
// CheckOperation passes) but never authorizes it without
// reboot_restoration.
func TestExecuteRestoreRefusesWithoutRebootCapability(t *testing.T) {
	fx := newLifecycleFixture(t, admittedWith("credential_capable_execution_realm"))
	recordBinding(t, fx, lxDigestA)
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent",
		Admitted: admittedWith("credential_capable_execution_realm"),
	})
	requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteTerminateRefusesOnDeadline pins the entry deadline as a
// strict bound: the operation whose now equals the deadline refuses
// instead of binding a receipt it cannot execute.
func TestExecuteTerminateRefusesOnDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.clock.Sleep(6 * time.Hour)
	presented, winner := terminateFencing()
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateStaleFenced {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	// The entry deadline refuses before any store side effect: the
	// per-effect recheck would refuse the same verdict, but only the
	// entry arm leaves the key unbound.
	if _, found, err := fx.lc.Receipts.Lookup(lxTerminateKey()); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt", found, err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteRestoreRefusesOnDeadline pins the restore entry deadline
// the same way: now equal to the deadline refuses before any receipt.
func TestExecuteRestoreRefusesOnDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.clock.Sleep(6 * time.Hour)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateAbsent {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	// The entry deadline refuses before any store side effect: the
	// per-effect recheck would refuse the same verdict, but only the
	// entry arm leaves the key unbound.
	if _, found, err := fx.lc.Receipts.Lookup(lxCreateKey()); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt", found, err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteRestoreRefusesOnDeadlineEffect pins the per-effect
// deadline as a strict bound: when the clock reaches the deadline
// between the persist and the wrapper start, the second effect
// refuses instead of running. The persist already committed, so the
// after-state is unavailable and the retry reconciles through status.
func TestExecuteRestoreRefusesOnDeadlineEffect(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.lc.Hooks = &terminstance.EngineHooks{BeforeEffect: func(terminalbackend.SideEffect) {
		fx.clock.Sleep(6 * time.Hour)
	}}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateUnavailable {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if len(outcome.Mutation.Effects) != 1 || len(outcome.Mutation.EvidenceIDs) != 1 {
		t.Fatalf("performed = %+v", outcome.Mutation)
	}
	if _, found, err := fx.lc.Receipts.Completed(lxCreateKey()); err != nil || found {
		t.Fatalf("completed = %v, %v; want no completion", found, err)
	}
	state, _, _ := fx.lc.States.Lookup(lxInstance)
	if state != terminalbackend.StateUnavailable {
		t.Fatalf("memory = %s, want unavailable", state)
	}
}

// TestExecuteRestoreRefusesGenerationDrift pins the per-effect
// generation recheck: a generation that validates at entry and
// rotates before the first effect refuses instead of executing under
// a stale binding. Nothing committed, so the after-state is the
// source and no receipt completes.
func TestExecuteRestoreRefusesGenerationDrift(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	calls := 0
	fx.lc.CurrentGeneration = func() string {
		calls++
		if calls >= 2 {
			return "generation-two"
		}
		return lxGeneration
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "terminal_backend_stale_generation", "backend_generation stale")
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

// TestExecuteAttachRefusesMeshWithoutCapability pins the mesh arm of
// the matching-transport gate: mesh presentation without
// remote_attach refuses even though local_attach confers the
// operation.
func TestExecuteAttachRefusesMeshWithoutCapability(t *testing.T) {
	admitted := admittedWith("local_attach")
	fx := newLifecycleFixture(t, admitted)
	raw := lxAttachBody(t, func(object map[string]any) {
		object["transport"] = "trusted_private_mesh"
		object["authorization"] = lxAttachAuth("trusted_private_mesh", true)
	})
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: raw, Source: "parked", Admitted: admitted,
	})
	requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteAttachRefusesDeadline pins the attach entry deadline:
// past-deadline refuses, and the on-deadline instant refuses too —
// the bound is strict, matching the mutating entries.
func TestExecuteAttachRefusesDeadline(t *testing.T) {
	t.Run("past", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		raw := lxAttachBody(t, func(object map[string]any) { object["deadline_at"] = lxIssued })
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: raw, Source: "parked", Admitted: fullLifecycleAdmitted(),
		})
		requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused attach execed %d commands", fx.runner.callCount())
		}
	})
	t.Run("boundary", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		raw := lxAttachBody(t, func(object map[string]any) { object["deadline_at"] = "2026-09-01T12:00:00.000Z" })
		admissionCalls := 0
		admission := fx.lc.ServerAdmission
		fx.lc.ServerAdmission = func(ctx context.Context) (RealmAdmission, error) {
			admissionCalls++
			return admission(ctx)
		}
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: raw, Source: "parked", Admitted: fullLifecycleAdmitted(),
		})
		requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
		// The entry arm refuses before the server admission runs;
		// the post-lock arm would refuse the same verdict after
		// it. Only the no-admission assertion pins this refusal
		// to the entry site.
		if admissionCalls != 0 {
			t.Fatalf("admission ran %d times; want the entry refusal", admissionCalls)
		}
	})
}

// TestExecuteEntryRefusesStaleGeneration pins the entry generation
// gate on both receipt operations: a context generation the live
// binding does not match refuses before any store side effect — the
// per-effect recheck would refuse the same verdict, but only the
// entry arm leaves the key unbound.
func TestExecuteEntryRefusesStaleGeneration(t *testing.T) {
	t.Run("terminate", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		raw := lxTerminateBody(t, func(object map[string]any) {
			object["context"].(map[string]any)["backend_generation"] = "generation-two"
		})
		presented, winner := terminateFencing()
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "terminate-stale", Body: raw, Source: "stale_fenced",
			Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		})
		requireLocalCode(t, err, "terminal_backend_stale_generation", "backend_generation stale")
		if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateStaleFenced {
			t.Fatalf("mutation = %+v", outcome.Mutation)
		}
		if _, found, err := fx.lc.Receipts.Lookup(lxTerminateKey()); err != nil || found {
			t.Fatalf("lookup = %v, %v; want no receipt", found, err)
		}
	})
	t.Run("restore", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		raw := lxRestoreBody(t, lxDigestA, func(object map[string]any) {
			object["context"].(map[string]any)["backend_generation"] = "generation-two"
		})
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		requireLocalCode(t, err, "terminal_backend_stale_generation", "backend_generation stale")
		if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateAbsent {
			t.Fatalf("mutation = %+v", outcome.Mutation)
		}
		if _, found, err := fx.lc.Receipts.Lookup(lxCreateKey()); err != nil || found {
			t.Fatalf("lookup = %v, %v; want no receipt", found, err)
		}
	})
}

// TestExecuteRefusesWrongIdempotencyKey pins the key-material gate on
// both receipt operations: a presented key the inputs do not derive
// refuses before capability, receipt, or exec.
func TestExecuteRefusesWrongIdempotencyKey(t *testing.T) {
	t.Run("terminate", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		raw := lxTerminateBody(t, func(object map[string]any) {
			object["context"].(map[string]any)["idempotency_key"] = "wrong/key"
		})
		presented, winner := terminateFencing()
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "terminate-stale", Body: raw, Source: "stale_fenced",
			Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		})
		requireLocalCode(t, err, "terminal_backend_protocol_error", "idempotency key material")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
		}
	})
	t.Run("restore", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		raw := lxRestoreBody(t, lxDigestA, func(object map[string]any) {
			object["context"].(map[string]any)["idempotency_key"] = "wrong/key"
		})
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		requireLocalCode(t, err, "terminal_backend_protocol_error", "idempotency key material")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused restore execed %d commands", fx.runner.callCount())
		}
	})
}

// TestExecuteTerminateRedriveMismatch pins the Bind coded-error
// mapping: the same key redriven with a different operation ID is
// the idempotency mismatch, not a replay and not a store failure,
// and it execs nothing further.
func TestExecuteTerminateRedriveMismatch(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("kill-session", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	presented, winner := terminateFencing()
	req := OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	req.Body = lxTerminateBody(t, func(object map[string]any) {
		object["context"].(map[string]any)["operation_id"] = lxBootstrap
	})
	outcome, err := fx.lc.Execute(context.Background(), req)
	requireEngineCode(t, err, "idempotency_mismatch")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateStaleFenced {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", fx.runner.callCount())
	}
}

// TestExecuteTerminateReplayRefusesCorruptImage pins the completion
// integrity arm: a bound key whose stored completion is not JSON
// refuses the image integrity failure with the unavailable
// after-state instead of replaying or resuming.
func TestExecuteTerminateReplayRefusesCorruptImage(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	key := lxTerminateKey()
	if _, replayed, err := fx.lc.Receipts.Bind(key, terminalbackend.OperationTerminateStale, lxOperation); err != nil || replayed {
		t.Fatalf("plant bind = replayed %v, %v", replayed, err)
	}
	if err := fx.lc.Receipts.Complete(key, []byte("not-json{{")); err != nil {
		t.Fatal(err)
	}
	presented, winner := terminateFencing()
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "idempotency result image")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateUnavailable {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("corrupt replay execed %d commands", fx.runner.callCount())
	}
}

// TestExecuteTerminateBindHookFailure pins the Bind store-failure
// mapping: a receipt that commits but reports failure is uncertainty
// with the unavailable after-state — the file exists, so the retry
// reconciles through status instead of replaying a completion that
// was never recorded.
func TestExecuteTerminateBindHookFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.Receipts.WithHooks(&terminstance.StoreHooks{AfterCommit: func(string) error {
		return errFakeTransport
	}})
	presented, winner := terminateFencing()
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "terminal_backend_process_failed", "idempotency store failure")
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateUnavailable {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("uncertain bind execed %d commands", fx.runner.callCount())
	}
}

// TestBoundWaitTakesLesser pins the boundary wait bound: the lesser
// of the request deadline and the timeout wins, and an invalid
// deadline falls back to the timeout alone.
func TestBoundWaitTakesLesser(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	deadline, err := scalar.ParseTimestamp(lxDeadline)
	if err != nil {
		t.Fatal(err)
	}
	mctx := terminstance.MutationContext{Deadline: deadline}
	wait, cancel := fx.lc.boundWait(context.Background(), mctx, 60000)
	defer cancel()
	got, ok := wait.Deadline()
	if !ok || !got.Equal(lxNow().Add(time.Minute)) {
		t.Fatalf("deadline = %v, %v; want now+60s bound", got, ok)
	}
	wait, cancel = fx.lc.boundWait(context.Background(), mctx, uint64(7*time.Hour/time.Millisecond))
	defer cancel()
	got, ok = wait.Deadline()
	want, _ := time.Parse(time.RFC3339Nano, lxDeadline)
	if !ok || !got.Equal(want) {
		t.Fatalf("deadline = %v, %v; want request deadline", got, ok)
	}
	broken := terminstance.MutationContext{}
	wait, cancel = fx.lc.boundWait(context.Background(), broken, 60000)
	defer cancel()
	got, ok = wait.Deadline()
	if !ok || !got.Equal(lxNow().Add(time.Minute)) {
		t.Fatalf("deadline = %v, %v; want timeout fallback", got, ok)
	}
}

// TestLesserDeadlineTakesLesser pins the stop poll bound the same
// way: lesser of the request deadline and the graceful timeout,
// invalid deadline falls back to the timeout.
func TestLesserDeadlineTakesLesser(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	deadline, err := scalar.ParseTimestamp(lxDeadline)
	if err != nil {
		t.Fatal(err)
	}
	mctx := terminstance.MutationContext{Deadline: deadline}
	if got := fx.lc.lesserDeadline(mctx, 60000); !got.Equal(lxNow().Add(time.Minute)) {
		t.Fatalf("lesser = %v; want now+60s bound", got)
	}
	want, _ := time.Parse(time.RFC3339Nano, lxDeadline)
	if got := fx.lc.lesserDeadline(mctx, uint64(7*time.Hour/time.Millisecond)); !got.Equal(want) {
		t.Fatalf("lesser = %v; want request deadline", got)
	}
	if got := fx.lc.lesserDeadline(terminstance.MutationContext{}, 60000); !got.Equal(lxNow().Add(time.Minute)) {
		t.Fatalf("lesser = %v; want timeout fallback", got)
	}
}

// TestExecuteStatusContradictions drives the observation verdicts
// through the status dispatch: resurrected and vanished wrappers
// read unavailable, malformed and contradictory probes refuse, and a
// live session without attach presentation reports parked but not
// attachable. Memory is staged per leg; the probe scripts are the
// only tmux the dispatch ever sees.
func TestExecuteStatusContradictions(t *testing.T) {
	t.Run("stopped-memory", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateStopped); err != nil {
			t.Fatal(err)
		}
		fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, true, nil), Admitted: fullLifecycleAdmitted(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Status.State != terminalbackend.StateUnavailable {
			t.Fatalf("state = %s, want unavailable", outcome.Status.State)
		}
	})
	t.Run("quiescing-memory", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateQuiescing); err != nil {
			t.Fatal(err)
		}
		fx.runner.queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Status.State != terminalbackend.StateUnavailable {
			t.Fatalf("state = %s, want unavailable", outcome.Status.State)
		}
	})
	t.Run("count-less-row", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|\n", 0)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, true, nil), Admitted: fullLifecycleAdmitted(),
		})
		requireEngineCode(t, err, "terminal_backend_unavailable")
	})
	t.Run("empty-rows", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		fx.runner.queue("list-panes", "\n", 0)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, true, nil), Admitted: fullLifecycleAdmitted(),
		})
		requireEngineCode(t, err, "terminal_backend_unavailable")
	})
	t.Run("unrequested-present", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Status.State != terminalbackend.StateParked {
			t.Fatalf("state = %s, want parked", outcome.Status.State)
		}
		if outcome.Status.ProviderPresent != nil {
			t.Fatalf("provider = %+v, want null", outcome.Status.ProviderPresent)
		}
	})
	t.Run("incapable-attach", func(t *testing.T) {
		admitted := admittedWith("provider_process_observation")
		fx := newLifecycleFixture(t, admitted)
		recordBinding(t, fx, lxDigestA)
		fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, true, nil), Admitted: admitted,
		})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Status.State != terminalbackend.StateParked || outcome.Status.Attachable {
			t.Fatalf("status = %+v", outcome.Status)
		}
	})
}

// TestExecuteQuiesceIdempotent pins the engine receipt through the
// quiesce dispatch: the second identical request replays the first
// mutation and locks nothing new.
func TestExecuteQuiesceIdempotent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	raw := lxQuiesceBody(t, nil)
	first, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: raw, Source: "active", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: raw, Source: "active", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", fx.runner.callCount())
	}
	if second.Mutation.After != first.Mutation.After || len(second.Mutation.EvidenceIDs) != len(first.Mutation.EvidenceIDs) {
		t.Fatalf("replay diverged: %+v vs %+v", second.Mutation, first.Mutation)
	}
	if first.InputClosedAt != second.InputClosedAt {
		t.Fatalf("replay changed closure time: %s vs %s", first.InputClosedAt, second.InputClosedAt)
	}
}

// TestExecuteBoundaryIdempotent pins the engine receipt through the
// boundary dispatch: the second identical request replays the first
// mutation and waits for nothing new.
func TestExecuteBoundaryIdempotent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	raw := lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil)
	first, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "wait-safe-boundary", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "wait-safe-boundary", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want 1", fx.runner.callCount())
	}
	if second.Mutation.After != first.Mutation.After || len(second.Mutation.EvidenceIDs) != len(first.Mutation.EvidenceIDs) {
		t.Fatalf("replay diverged: %+v vs %+v", second.Mutation, first.Mutation)
	}
	if first.BoundaryObservedAt != second.BoundaryObservedAt {
		t.Fatalf("replay changed boundary time: %s vs %s", first.BoundaryObservedAt, second.BoundaryObservedAt)
	}
	if first.SafeBoundaryEvidenceID != second.SafeBoundaryEvidenceID {
		t.Fatalf("replay changed proof: %s vs %s", first.SafeBoundaryEvidenceID, second.SafeBoundaryEvidenceID)
	}
}

// TestExecuteStopIdempotent pins the engine receipt through the stop
// dispatch: the second identical request replays the first mutation
// and closes nothing new.
func TestExecuteStopIdempotent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("send-keys", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	raw := lxStopBody(t, 60000, nil)
	first, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", fx.runner.callCount())
	}
	if second.Mutation.After != first.Mutation.After || len(second.Mutation.EvidenceIDs) != len(first.Mutation.EvidenceIDs) {
		t.Fatalf("replay diverged: %+v vs %+v", second.Mutation, first.Mutation)
	}
}

// TestExecuteTerminateTargetLeaseIdentity pins the stale-lease arm of
// the target binding: a presented token that agrees on session and
// epoch but names another lease refuses, so the fencing decision
// cannot authorize a different target than the key names.
func TestExecuteTerminateTargetLeaseIdentity(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	_, winner := terminateFencing()
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted:  fullLifecycleAdmitted(),
		Presented: fencing.PresentedToken{SessionID: lxSession, Epoch: 9, LeaseID: lxLease},
		Winner:    winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "local_precondition_failed", "terminate target binding")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused terminate execed %d commands", fx.runner.callCount())
	}
}
