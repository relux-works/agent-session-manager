//go:build !windows

package tmuxserver

import (
	"context"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// terminateFencing returns the fencing material for a fenced target:
// presented (session, stale epoch/lease) differs from the winning
// (session, current epoch/lease), with explicit force recovery and
// preserved diagnostics.
func terminateFencing() (fencing.PresentedToken, sessrepo.LeaseSummary) {
	presented := fencing.PresentedToken{SessionID: lxSession, Epoch: 9, LeaseID: lxLeaseB}
	winner := sessrepo.LeaseSummary{SessionID: lxSession, LeaseID: lxLease, Epoch: 7}
	return presented, winner
}

func TestExecuteCreateInteractive(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateActive {
		t.Fatalf("mutation = %+v", outcome.Mutation)
	}
	if outcome.Binding == nil || outcome.Binding.TerminalInstanceID != lxInstance {
		t.Fatalf("binding = %+v", outcome.Binding)
	}
	if outcome.Descriptor == nil || *outcome.Descriptor != fx.socket+"\n"+lxInstance {
		t.Fatalf("descriptor = %+v", outcome.Descriptor)
	}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want 1", fx.runner.callCount())
	}
	wantArgv := []string{"tmux", "-S", fx.socket, "new-session", "-d", "-s", lxInstance, "ax", "pane", lxSession}
	if got := fx.runner.calls[0]; len(got) != len(wantArgv) {
		t.Fatalf("argv = %q, want %q", got, wantArgv)
	} else {
		for i := range wantArgv {
			if got[i] != wantArgv[i] {
				t.Fatalf("argv = %q, want %q", got, wantArgv)
			}
		}
	}
	state, found, err := fx.lc.States.Lookup(lxInstance)
	if err != nil || !found || state != terminalbackend.StateActive {
		t.Fatalf("memory = %s, %v, %v", state, found, err)
	}
}

func TestExecuteCreateHeadless(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	raw := lxCreateBody(t, func(object map[string]any) { object["interactive"] = false })
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: raw, Source: "stopped", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation.After != terminalbackend.StateParked {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	if outcome.Descriptor != nil {
		t.Fatalf("headless descriptor = %q", *outcome.Descriptor)
	}
}

func TestExecuteCreateRefusals(t *testing.T) {
	t.Run("relay", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		raw := lxCreateBody(t, func(object map[string]any) { object["presentation_transport"] = "third_party_relay" })
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "create", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted()})
		requireLocalCode(t, err, "terminal_backend_unauthorized", "create relay transport")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused create execed %d commands", fx.runner.callCount())
		}
	})
	t.Run("headless-without-capability", func(t *testing.T) {
		fx := newLifecycleFixture(t, admittedWith("credential_capable_execution_realm"))
		raw := lxCreateBody(t, func(object map[string]any) { object["interactive"] = false })
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "create", Body: raw, Source: "absent", Admitted: admittedWith("credential_capable_execution_realm")})
		requireEngineCode(t, err, "terminal_backend_capability_unproven")
		if outcome.Mutation == nil || outcome.Mutation.After != terminalbackend.StateAbsent {
			t.Fatalf("mutation = %+v", outcome.Mutation)
		}
	})
	t.Run("bad-source", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "create", Body: lxCreateBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()})
		requireEngineCode(t, err, "local_precondition_failed")
	})
}

func TestExecuteCreateBindingAgreement(t *testing.T) {
	members := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"session", func(binding map[string]any) { binding["session_id"] = lxSessionB }},
		{"instance", func(binding map[string]any) { binding["terminal_instance_id"] = lxSessionB }},
		{"backend", func(binding map[string]any) { binding["terminal_backend_id"] = "ax.conpty" }},
		{"generation", func(binding map[string]any) { binding["backend_generation"] = "generation-two" }},
	}
	for _, tc := range members {
		t.Run(tc.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			raw := lxCreateBody(t, func(object map[string]any) {
				object["binding"] = lxBindingDoc(t, tc.mutate)
			})
			_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "create", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted()})
			requireLocalCode(t, err, "local_precondition_failed", "create binding agreement")
			if fx.runner.callCount() != 0 {
				t.Fatalf("disagreeing create execed %d commands", fx.runner.callCount())
			}
		})
	}
}

func TestExecuteCreateIdempotent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	raw := lxCreateBody(t, nil)
	first, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "create", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "create", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want 1", fx.runner.callCount())
	}
	if second.Mutation.After != first.Mutation.After || len(second.Mutation.EvidenceIDs) != len(first.Mutation.EvidenceIDs) {
		t.Fatalf("replay diverged: %+v vs %+v", second.Mutation, first.Mutation)
	}
}

func TestExecuteAttach(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Attach == nil || outcome.Attach.ClientMirrorID != lxClient || !outcome.Attach.InputAuthorized {
		t.Fatalf("attach = %+v", outcome.Attach)
	}
	if outcome.Attach.Descriptor != fx.socket+"\n"+lxInstance+"\n"+lxClient {
		t.Fatalf("descriptor = %q", outcome.Attach.Descriptor)
	}
	wantArgv := []string{"tmux", "-S", fx.socket, "attach-session", "-t", lxInstance}
	if len(outcome.Argv) != len(wantArgv) {
		t.Fatalf("argv = %q", outcome.Argv)
	}
	for i := range wantArgv {
		if outcome.Argv[i] != wantArgv[i] {
			t.Fatalf("argv = %q, want %q", outcome.Argv, wantArgv)
		}
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("attach execed %d commands", fx.runner.callCount())
	}
	state, found, err := fx.lc.States.Lookup(lxInstance)
	if err != nil || !found || state != terminalbackend.StateActive {
		t.Fatalf("memory = %s, %v, %v", state, found, err)
	}
}

func TestExecuteAttachRefusals(t *testing.T) {
	t.Run("fresh-decoy", func(t *testing.T) {
		admitted := admittedWith("local_attach")
		fx := newLifecycleFixture(t, admitted)
		fx.lc.ServerAdmission = func(context.Context) (RealmAdmission, error) {
			return RealmAdmission{Admitted: admittedWith("local_attach"), RawGeneration: lxGeneration}, nil
		}
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted})
		requireLocalCode(t, err, "terminal_backend_unauthorized", "attach server attestation")
	})
	t.Run("stale-admission", func(t *testing.T) {
		admitted := fullLifecycleAdmitted()
		fx := newLifecycleFixture(t, admitted)
		fx.lc.ServerAdmission = func(context.Context) (RealmAdmission, error) {
			return RealmAdmission{Admitted: admitted, RawGeneration: "generation-two"}, nil
		}
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted})
		requireLocalCode(t, err, "terminal_backend_unauthorized", "attach server attestation")
	})
	t.Run("wrong-transport-capability", func(t *testing.T) {
		admitted := admittedWith("remote_attach")
		fx := newLifecycleFixture(t, admitted)
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted})
		requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	})
	t.Run("relay", func(t *testing.T) {
		admitted := fullLifecycleAdmitted()
		fx := newLifecycleFixture(t, admitted)
		raw := lxAttachBody(t, func(object map[string]any) {
			object["transport"] = "third_party_relay"
			object["authorization"] = lxAttachAuth("third_party_relay", true)
		})
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: raw, Source: "parked", Admitted: admitted})
		requireLandedCode(t, err, "terminal_backend_unauthorized")
	})
	t.Run("bad-source", func(t *testing.T) {
		admitted := fullLifecycleAdmitted()
		fx := newLifecycleFixture(t, admitted)
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "stopped", Admitted: admitted})
		requireLandedCode(t, err, "local_precondition_failed")
	})
	t.Run("no-admission", func(t *testing.T) {
		admitted := fullLifecycleAdmitted()
		fx := newLifecycleFixture(t, admitted)
		fx.lc.ServerAdmission = nil
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted})
		requireLocalCode(t, err, "terminal_backend_protocol_error", "lifecycle dependencies")
	})
}

func TestExecuteAttachIdempotent(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	raw := lxAttachBody(t, nil)
	first, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: raw, Source: "parked", Admitted: admitted})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: raw, Source: "active", Admitted: admitted})
	if err != nil {
		t.Fatal(err)
	}
	if second.Attach.Descriptor != first.Attach.Descriptor || len(second.Attach.EvidenceIDs) != 1 {
		t.Fatalf("replay diverged: %+v", second.Attach)
	}
}

func TestExecuteStatusPresent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, true, nil), Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status == nil || outcome.Status.State != terminalbackend.StateParked {
		t.Fatalf("status = %+v", outcome.Status)
	}
	if !outcome.Status.IdentityMatch || !outcome.Status.WrapperPresent || !outcome.Status.Attachable {
		t.Fatalf("status = %+v", outcome.Status)
	}
	if outcome.Status.ProviderPresent == nil || !*outcome.Status.ProviderPresent {
		t.Fatalf("provider = %+v", outcome.Status.ProviderPresent)
	}
}

func TestExecuteStatusAbsent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status.State != terminalbackend.StateAbsent || outcome.Status.ProviderPresent != nil {
		t.Fatalf("status = %+v", outcome.Status)
	}
}

func TestExecuteStatusProviderConditional(t *testing.T) {
	fx := newLifecycleFixture(t, admittedWith("local_attach"))
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, true, nil), Admitted: admittedWith("local_attach"),
	})
	requireEngineCode(t, err, "terminal_backend_capability_unproven")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused status probed %d commands", fx.runner.callCount())
	}
}

func TestExecuteQuiesce(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	if outcome.QuiescenceGeneration != lxQuiesce {
		t.Fatalf("quiescence = %q", outcome.QuiescenceGeneration)
	}
	assertTimestamp(t, outcome.InputClosedAt)
	state, _, _ := fx.lc.States.Lookup(lxInstance)
	if state != terminalbackend.StateQuiescing {
		t.Fatalf("memory = %s", state)
	}
}

func TestExecuteBoundary(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	if len(outcome.Mutation.EvidenceIDs) != 1 || outcome.SafeBoundaryEvidenceID != outcome.Mutation.EvidenceIDs[0] {
		t.Fatalf("evidence = %+v", outcome.Mutation)
	}
	assertTimestamp(t, outcome.BoundaryObservedAt)
}

func TestExecuteStop(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("send-keys", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "request-stop", Body: lxStopBody(t, 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation.After != terminalbackend.StateStopped {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	if !outcome.ProcessClosed || !outcome.StoreClosed {
		t.Fatalf("markers = %v/%v", outcome.ProcessClosed, outcome.StoreClosed)
	}
	state, _, _ := fx.lc.States.Lookup(lxInstance)
	if state != terminalbackend.StateStopped {
		t.Fatalf("memory = %s", state)
	}
}

func TestExecuteTerminate(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("kill-session", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	presented, winner := terminateFencing()
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation.After != terminalbackend.StateStopped || !outcome.TargetClosed {
		t.Fatalf("outcome = %+v", outcome)
	}
	state, _, _ := fx.lc.States.Lookup(lxInstance)
	if state != terminalbackend.StateStopped {
		t.Fatalf("memory = %s", state)
	}
	if _, found, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || found {
		t.Fatalf("incarnation found=%v, err=%v, want terminate to leave the incarnation alone", found, err)
	}
}

func TestExecuteTerminateRefusals(t *testing.T) {
	newTerminate := func(t *testing.T) (*lxFixture, OpRequest) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		presented, winner := terminateFencing()
		return fx, OpRequest{
			Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
			Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		}
	}
	t.Run("live-owner", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.Presented = fencing.PresentedToken{SessionID: lxSession, Epoch: 7, LeaseID: lxLease}
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "local_precondition_failed", "terminate fencing authorization")
	})
	t.Run("no-force", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.ForceRecovery = false
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "local_precondition_failed", "terminate fencing authorization")
	})
	t.Run("no-diagnostics", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.DiagnosticsPreserved = false
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "local_precondition_failed", "terminate fencing authorization")
	})
	t.Run("no-winner", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.HasWinner = false
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "local_precondition_failed", "terminate fencing authorization")
	})
	t.Run("malformed-presented", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.Presented = fencing.PresentedToken{SessionID: "nope", Epoch: 9, LeaseID: lxLeaseB}
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "terminal_backend_protocol_error", "terminate fencing material")
	})
	t.Run("target-mismatch", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.Presented = fencing.PresentedToken{SessionID: lxSession, Epoch: 8, LeaseID: lxLeaseB}
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "local_precondition_failed", "terminate target binding")
	})
	t.Run("winner-rotated", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.Winner = sessrepo.LeaseSummary{SessionID: lxSession, LeaseID: lxLease, Epoch: 8}
		_, err := fx.lc.Execute(context.Background(), req)
		requireLocalCode(t, err, "local_precondition_failed", "terminate winner binding")
	})
	t.Run("bad-source", func(t *testing.T) {
		fx, req := newTerminate(t)
		req.Source = "active"
		_, err := fx.lc.Execute(context.Background(), req)
		requireLandedCode(t, err, "local_precondition_failed")
	})
	t.Run("missing-capability", func(t *testing.T) {
		fx := newLifecycleFixture(t, admittedWith("stale_process_termination"))
		presented, winner := terminateFencing()
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
			Admitted: admittedWith("stale_process_termination"), Presented: presented, Winner: winner,
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		})
		requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	})
}

func TestExecuteRestore(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Mutation.After != terminalbackend.StateParked || !outcome.RestoredParked {
		t.Fatalf("outcome = %+v", outcome)
	}
	if outcome.PriorBindingID != lxDigestA {
		t.Fatalf("prior = %q", outcome.PriorBindingID)
	}
	wantArgv := []string{"tmux", "-S", fx.socket, "new-session", "-d", "-s", lxInstance, "ax", "pane", lxSession}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want 1", fx.runner.callCount())
	}
	if got := fx.runner.calls[0]; len(got) != len(wantArgv) {
		t.Fatalf("argv = %q, want %q", got, wantArgv)
	} else {
		for i := range wantArgv {
			if got[i] != wantArgv[i] {
				t.Fatalf("argv = %q, want %q", got, wantArgv)
			}
		}
	}
	state, _, _ := fx.lc.States.Lookup(lxInstance)
	if state != terminalbackend.StateParked {
		t.Fatalf("memory = %s", state)
	}
}

func TestExecuteRestoreMismatch(t *testing.T) {
	t.Run("wrong-digest", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: lxRestoreBody(t, lxDigestB, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		requireLocalCode(t, err, "terminal_backend_restore_mismatch", "restore prior binding")
		if fx.runner.callCount() != 0 {
			t.Fatalf("mismatched restore execed %d commands", fx.runner.callCount())
		}
	})
	t.Run("unrecorded", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		requireLocalCode(t, err, "terminal_backend_restore_mismatch", "restore prior binding")
	})
	t.Run("bad-source", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
		})
		requireLandedCode(t, err, "local_precondition_failed")
	})
}

func TestExecuteRefusesForeignBackend(t *testing.T) {
	cases := []struct {
		operation string
		body      func(*testing.T) []byte
		source    string
	}{
		{"create", func(t *testing.T) []byte { return foreignBackendBody(t, lxCreateBody(t, nil)) }, "absent"},
		{"attach", func(t *testing.T) []byte { return foreignBackendBody(t, lxAttachBody(t, nil)) }, "parked"},
		{"status", func(t *testing.T) []byte { return foreignBackendBody(t, lxStatusBody(t, true, false, nil)) }, ""},
		{"quiesce-input", func(t *testing.T) []byte { return foreignBackendBody(t, lxQuiesceBody(t, nil)) }, "active"},
		{"wait-safe-boundary", func(t *testing.T) []byte {
			return foreignBackendBody(t, lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil))
		}, "quiescing"},
		{"request-stop", func(t *testing.T) []byte { return foreignBackendBody(t, lxStopBody(t, 60000, nil)) }, "quiescing"},
		{"terminate-stale", func(t *testing.T) []byte { return foreignBackendBody(t, lxTerminateBody(t, nil)) }, "stale_fenced"},
		{"restore", func(t *testing.T) []byte { return foreignBackendBody(t, lxRestoreBody(t, lxDigestA, nil)) }, "absent"},
	}
	for _, tc := range cases {
		t.Run(tc.operation, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			presented, winner := terminateFencing()
			_, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: tc.operation, Body: tc.body(t), Source: tc.source,
				Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
				HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
			})
			requireLocalCode(t, err, "local_precondition_failed", "lifecycle backend")
			if fx.runner.callCount() != 0 {
				t.Fatalf("%s execed %d commands", tc.operation, fx.runner.callCount())
			}
		})
	}
}
