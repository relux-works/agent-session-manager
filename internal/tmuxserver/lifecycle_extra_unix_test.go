//go:build !windows

package tmuxserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

func TestExecuteTerminateResumesAfterEffectFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	presented, winner := terminateFencing()
	req := OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	}
	// First attempt: the kill fails against a live session — coded,
	// pre-commit, so the receipt binds without its completion.
	fx.runner.queue("kill-session", "", 1).queue("has-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), req)
	requireLocalCode(t, err, "terminal_backend_process_failed", "backend effect refused")
	if outcome.Mutation.After != terminalbackend.StateStaleFenced {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	// Retry: status proves the source (fenced memory plus a live
	// session), so the same operation resumes under the same receipt.
	fx.runner.
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0).
		queue("kill-session", "", 0).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	resumed, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Mutation.After != terminalbackend.StateStopped || !resumed.TargetClosed {
		t.Fatalf("resumed = %+v", resumed.Mutation)
	}
}

func TestExecuteRestoreResumesAfterRecheckFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	req := OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	}
	// First attempt: the authorization is valid at entry and drifts
	// before the first effect — post-bind, pre-commit, so the receipt
	// binds without its completion and the memory records the source.
	good := terminstance.LeaseView{LeaseID: lxLease, Epoch: 7}
	bad := terminstance.LeaseView{LeaseID: lxLeaseB, Epoch: 7}
	calls := 0
	fx.lc.CurrentLease = func() terminstance.LeaseView {
		calls++
		if calls == 2 {
			return bad
		}
		return good
	}
	outcome, err := fx.lc.Execute(context.Background(), req)
	requireLocalCode(t, err, "terminal_backend_unauthorized", "lifecycle authorization recheck")
	if outcome.Mutation.After != terminalbackend.StateAbsent {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	// Retry: status proves absent, so the operation resumes under the
	// same receipt and the persist effect replays the recorded pair.
	fx.runner.queueFull("list-panes", "", "can't find window: "+lxInstance, 1).queue("new-session", "", 0)
	resumed, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Mutation.After != terminalbackend.StateParked || !resumed.RestoredParked {
		t.Fatalf("resumed = %+v", resumed.Mutation)
	}
}

func TestExecuteResumeRefusesWhenStatusProvesOtherwise(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	req := OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	}
	// First attempt: the wrapper start fails unprovably after the
	// persist committed — bound without completion, memory unavailable.
	fx.runner.failTransport("new-session", errFakeTransport)
	if _, err := fx.lc.Execute(context.Background(), req); err == nil {
		t.Fatal("unproven restore admitted")
	}
	// Retry reconciles against a live session: status proves parked,
	// not the presented absent, so the retry refuses uncertain.
	fx.runner.
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	outcome, err := fx.lc.Execute(context.Background(), req)
	requireLocalCode(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
	if outcome.Mutation.After != terminalbackend.StateUnavailable {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}

func TestExecuteTerminateIdempotent(t *testing.T) {
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
	second, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", fx.runner.callCount())
	}
	if second.Mutation.After != terminalbackend.StateStopped || !second.TargetClosed {
		t.Fatalf("replay = %+v", second.Mutation)
	}
}

func TestExecuteBoundaryWaitTimeout(t *testing.T) {
	// Deterministic wait expiry: the wait context derives from the
	// canceled parent, so it is already expired when the unscripted
	// runner reports its transport failure — the coded quiesce_timeout
	// proving non-commit, with the source state restored.
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome, err := fx.lc.Execute(ctx, OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	requireEngineCode(t, err, "quiesce_timeout")
	if outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}

func TestExecuteBoundaryPastDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	raw := lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, func(object map[string]any) {
		context := object["context"].(map[string]any)
		context["deadline_at"] = "2026-09-01T00:00:00.000Z"
	})
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	requireEngineCode(t, err, "quiesce_timeout")
	if fx.runner.callCount() != 0 {
		t.Fatalf("expired boundary execed %d commands", fx.runner.callCount())
	}
}

func TestAttachDescriptorNeverPersisted(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted,
	})
	if err != nil {
		t.Fatal(err)
	}
	scanStoresForSecret(t, fx.data, outcome.Attach.Descriptor, "attach descriptor")
	// The create-interactive descriptor is equally unpersisted.
	fx.runner.queue("new-session", "", 0)
	created, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: admitted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Descriptor == nil {
		t.Fatal("interactive create has no descriptor")
	}
	scanStoresForSecret(t, fx.data, *created.Descriptor, "create descriptor")
}

// scanStoresForSecret walks every durable store file and fails when
// any file contains the secret bytes.
func scanStoresForSecret(t *testing.T, data, secret, name string) {
	t.Helper()
	if secret == "" {
		t.Fatalf("empty %s", name)
	}
	err := filepath.Walk(data, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(raw), secret) {
			t.Fatalf("%s persisted in %s", name, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleIgnoresAmbientEnvironment(t *testing.T) {
	// B16 disposition, lifecycle side: the entries derive the socket
	// from the runtime directory and never read the process
	// environment, so an ax invocation from inside its own AX pane
	// (TMUX naming the derived socket in the real encoding) executes
	// normally here. The acquisition side keeps the over-strict
	// refusal (TestAcquireNestedInvocationRefusesAmbientCollision):
	// no specified flow nests acquisition, while lifecycle entries
	// take no ambient input at all.
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	t.Setenv("TMUX", fx.socket+",1234,0")
	t.Setenv("TMUX_TMPDIR", "/tmp/ambient")
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted,
	})
	if err != nil {
		t.Fatalf("nested lifecycle attach refused: %v", err)
	}
	if outcome.Attach == nil {
		t.Fatal("no attach outcome")
	}
}

func TestAcquireNestedInvocationRefusesAmbientCollision(t *testing.T) {
	// B16 disposition, acquisition side: an ax invocation from inside
	// its own AX pane refuses ambient collision (over-strict by
	// decision). The wrapper (ax pane) never resolves a server socket
	// — axpane carries no tmuxserver dependency — so no specified flow
	// needs nested acquisition, and admitting would trust ambient
	// bytes indistinguishable from the operator's default.
	root := lxShortRoot(t)
	socket := SocketPath(root + "/tmux")
	lookup := func(name string) (string, bool) {
		if name == "TMUX" {
			return socket + ",1234,0", true
		}
		return "", false
	}
	ambient := ObserveAmbient(lookup, "", "", "")
	for _, caller := range []Caller{CallerForeground, CallerBackground} {
		req := validRequest(t, root)
		req.Caller = caller
		req.Ambient = ambient
		_, err := Acquire(req)
		requireLocalCode(t, err, "tmux_ambient_server_reuse", "ambient collision")
	}
}

func TestLifecycleEmittedCodesAreRowAllowed(t *testing.T) {
	// Every code the three lifecycle-owned operations emit belongs to
	// the operation's allowed set, checked against the landed oracle.
	// Engine-delegated operations inherit the engine's conformance.
	emitted := map[terminalbackend.Operation][]string{
		terminalbackend.OperationAttach: {
			"terminal_backend_protocol_error", "terminal_backend_timeout",
			"terminal_backend_unauthorized", "local_precondition_failed",
			"idempotency_mismatch", "terminal_backend_stale_generation",
			"terminal_backend_capability_unproven", "terminal_backend_unavailable",
			"terminal_backend_process_failed", "terminal_backend_integrity_failure",
		},
		terminalbackend.OperationTerminateStale: {
			"terminal_backend_protocol_error", "terminal_backend_timeout",
			"terminal_backend_unauthorized", "local_precondition_failed",
			"idempotency_mismatch", "terminal_backend_stale_generation",
			"terminal_backend_capability_unproven",
			"terminal_backend_process_failed", "terminal_backend_integrity_failure",
		},
		terminalbackend.OperationRestore: {
			"terminal_backend_protocol_error", "terminal_backend_timeout",
			"terminal_backend_unauthorized", "local_precondition_failed",
			"idempotency_mismatch", "terminal_backend_stale_generation",
			"terminal_backend_capability_unproven", "terminal_backend_restore_mismatch",
			"terminal_backend_process_failed",
			"terminal_backend_integrity_failure",
		},
	}
	for operation, codes := range emitted {
		for _, code := range codes {
			if err := terminalbackend.CheckErrorAllowed(string(operation), code); err != nil {
				t.Fatalf("row %s emits outside-set %s", operation, code)
			}
		}
	}
	// The uncertain-retry verdict (terminal_backend_unavailable with
	// status_first) is the lifecycle's AX-side contract for unprovable
	// outcomes, outside the backend-report rule the rows govern —
	// mirroring the landed engine, which reports the same verdict on
	// quiesce-input, wait-safe-boundary, and request-stop, whose rows
	// likewise lack the member. The verdict is pinned through the
	// resume-refusal tests, not through this oracle.
}

func TestRecordFailureLeavesPresenceReconciliation(t *testing.T) {
	// The state memory is advisory: when Record fails, the operation
	// still succeeds and later status reconciles from live presence.
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterInstall: func(path string) error {
		// Only the advisory instance-state record fails: the
		// binding document the restore read needs still installs,
		// and its failure is a real failure with its own test.
		if strings.Contains(path, string(filepath.Separator)+"instances"+string(filepath.Separator)) {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("create failed on memory failure: %v", err)
	}
	if outcome.Mutation.After != terminalbackend.StateActive {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	fx.lc.States.WithHooks(nil)
	// No memory plus a live session reads parked (zero attached).
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
	status, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status.State != terminalbackend.StateParked {
		t.Fatalf("status = %s", status.Status.State)
	}
}

func TestQuiesceTimeoutCodeUnreachableOnWrongRows(t *testing.T) {
	// quiesce_timeout is admitted only on wait-safe-boundary and
	// stop_timeout only on request-stop: the oracle pins both.
	if err := terminalbackend.CheckErrorAllowed("wait-safe-boundary", "quiesce_timeout"); err != nil {
		t.Fatalf("boundary refuses its timeout: %v", err)
	}
	if err := terminalbackend.CheckErrorAllowed("request-stop", "stop_timeout"); err != nil {
		t.Fatalf("stop refuses its timeout: %v", err)
	}
	for _, operation := range []string{"create", "attach", "status", "quiesce-input", "terminate-stale", "restore"} {
		if err := terminalbackend.CheckErrorAllowed(operation, "quiesce_timeout"); err == nil {
			t.Fatalf("row %s admits quiesce_timeout", operation)
		}
		if err := terminalbackend.CheckErrorAllowed(operation, "stop_timeout"); err == nil {
			t.Fatalf("row %s admits stop_timeout", operation)
		}
	}
	_ = scalar.PlatformMacOS
}
