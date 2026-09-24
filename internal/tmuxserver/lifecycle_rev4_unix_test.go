//go:build !windows

package tmuxserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Revision-4 P1-A: the quiesce input barrier is durable and fails
// closed across state-write failure and recovery. A failed barrier
// write never reports successful closure, and the attach entry
// recovers the barrier from the completed closure outcomes when the
// state memory holds no record — absence of memory is not proof of
// an open input.

func TestExecuteQuiesceStateWriteFailureRefusesWiredAttach(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterStage: func(path string) error {
		if strings.Contains(path, "/instances/") {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	quiesced, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatal("quiesce reported success after its barrier write failed")
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "quiesce state image")
	if quiesced.Mutation == nil {
		t.Fatal("failed quiesce omits its mutation")
	}
	if got := fx.runner.subcommands(); len(got) != 2 || got[0] != "lock-session" || got[1] != "detach-client" {
		t.Fatalf("closure effects = %v, want the executed lock and detach", got)
	}
	if _, found, err := fx.lc.States.Lookup(lxInstance); err != nil || found {
		t.Fatalf("memory = %v, %v, want no record after the failed write", found, err)
	}
	if _, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey()); err != nil || !found {
		t.Fatalf("outcome = %v, %v, want the recorded closure proof", found, err)
	}
	fx.lc.States.WithHooks(nil)
	// The input-closed instance refuses writable input even though
	// the memory never recorded the barrier.
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("writable attach admitted after closure with lost state")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
	// Observation survives the recovery: read-only attach stays
	// admitted while input stays refused.
	readonly, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = lxClientB
			object["input_authorized"] = false
			object["authorization"] = lxAttachAuth("local_only", false)
		}), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("read-only attach refused under the recovered barrier: %v", err)
	}
	if readonly.Attach == nil || readonly.Attach.InputAuthorized {
		t.Fatalf("attach = %+v", readonly.Attach)
	}
}

func TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterStage: func(path string) error {
		if strings.Contains(path, "/instances/") {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("quiesce reported success after its barrier write failed")
	}
	fx.lc.States.WithHooks(nil)
	fx.clock.Sleep(time.Second)
	// The identical retry replays the completed receipt without a
	// second effect, heals the barrier memory, and replays the
	// original closure time — not the retry time.
	retried, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if retried.InputClosedAt != "2026-09-01T12:00:00.000Z" {
		t.Fatalf("InputClosedAt = %q, want the original closure time", retried.InputClosedAt)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("retry executed %v, want no second effect", fx.runner.subcommands())
	}
	if state, found, err := fx.lc.States.Lookup(lxInstance); err != nil || !found || state != terminalbackend.StateQuiescing {
		t.Fatalf("memory = %s, %v, %v, want healed quiescing", state, found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("writable attach admitted after the healed barrier")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

func TestExecuteBoundaryBarrierSurvivesStateWriteFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterStage: func(path string) error {
		if strings.Contains(path, "/instances/") {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("wait-for", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("boundary reported success after its barrier write failed")
	} else {
		requireLocalCode(t, err, "terminal_backend_integrity_failure", "boundary state image")
	}
	fx.lc.States.WithHooks(nil)
	// No quiesce ran here: only the boundary outcome proves the
	// barrier, isolating the boundary recovery arm.
	if _, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey()); err != nil || found {
		t.Fatalf("quiesce outcome = %v, %v, want none", found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("writable attach admitted after boundary closure with lost state")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

func TestExecuteAttachErrorsOnCorruptBarrierScan(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	dir := filepath.Join(fx.data, "states", "tmuxlifecycle", "outcomes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "corrupt-barrier-probe.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A corrupt barrier report is unknown, never an open input.
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("attach admitted under a corrupt barrier scan")
	} else if !strings.Contains(err.Error(), "decode tmux barrier outcome") {
		t.Fatalf("attach error = %v, want the barrier decode failure", err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}

// Revision-4 clause witnesses: four admitting narrowings the
// reviewer planted against the status probes and the argv gate, each
// surviving the committed suite twice, plus the spawner-side custody
// narrowing that closes the SPW/B39 gap. Each test drives the
// production entry and refuses the exact admitted member.

// TestExecuteStatusAttachedExitIsUnknown pins that a nonzero
// list-sessions exit proves nothing even with plausible stdout:
// exit 1 with a well-formed attached-count row is unknown, never a
// count.
func TestExecuteStatusAttachedExitIsUnknown(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 1)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, true, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("exit-1 attached probe read a count")
	} else {
		requireEngineCode(t, err, "terminal_backend_unavailable")
	}
}

// TestExecuteStatusForeignAttachedRowIsUnknown pins that only the
// queried instance's own row counts: a well-formed row for another
// session with no row for this instance is unknown, never zero
// attached.
func TestExecuteStatusForeignAttachedRowIsUnknown(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", "review-foreign-instance|0\n", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, true, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("foreign attached row read a count")
	} else {
		requireEngineCode(t, err, "terminal_backend_unavailable")
	}
}

// TestExecuteStatusUnknownPanesExitIsUnknown pins that an unmarked
// nonzero list-panes exit proves nothing even with plausible
// stdout: exit 2 with a provider row is unknown, never presence.
func TestExecuteStatusUnknownPanesExitIsUnknown(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "sh\n", 2).queue("list-sessions", lxInstance+"|0\n", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, true, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("exit-2 panes probe read presence")
	} else {
		requireEngineCode(t, err, "terminal_backend_unavailable")
	}
}

// TestOSRunnerRefusesNulArgument pins the argv gate at the execution
// boundary: a NUL-containing argument refuses before any process
// starts, and the command is never built.
func TestOSRunnerRefusesNulArgument(t *testing.T) {
	called := false
	runner := OSRunner{Command: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		called = true
		return exec.CommandContext(ctx, "/usr/bin/true")
	}}
	if _, err := runner.Run(context.Background(), []string{"tmux", "review\x00arg"}); err == nil {
		t.Fatal("NUL argument admitted to execution")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command argv")
	}
	if called {
		t.Fatal("refused argv built a process")
	}
}

// TestServerSpawnerRefusesWritableRoot pins the before-bind custody
// rejection at the spawner entry: a world-writable root refuses
// before any spawn vector runs.
func TestServerSpawnerRefusesWritableRoot(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if err := os.Chmod(fx.root, 0o777); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("new-session", "", 0)
	spawner := ServerSpawner{Root: fx.root, Platform: fx.lc.Platform, Runner: fx.runner}
	err := spawner.Spawn(fx.socket, []string{"tmux", "-S", fx.socket, "new-session", "-d", "-s", lxInstance, "ax", "pane", lxSession})
	if err == nil {
		t.Fatal("writable root admitted by spawn")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused spawn executed %v", fx.runner.subcommands())
	}
}

func TestExecuteAttachErrorsOnEmptyBarrierTime(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"quiesce", lxQuiesceKey()},
		{"boundary", lxBoundaryKey("ax_checkpoint_boundary")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			if err := fx.lc.States.RecordOutcome(tc.key, OperationOutcome{Operation: tc.name}); err != nil {
				t.Fatal(err)
			}
			// A closure report with no closure time proves nothing: the
			// scan refuses it exactly like the exact read does.
			if _, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
			}); err == nil {
				t.Fatal("attach admitted under an empty barrier time")
			} else {
				requireLocalCode(t, err, "tmux_invalid_arguments", "instance outcome key")
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("refused attach executed %v", fx.runner.subcommands())
			}
		})
	}
}
