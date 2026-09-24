//go:build !windows

package tmuxserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestAttachRefusesAfterQuiescenceWhenAdmittedLate pins the P1-B
// ordering contract at the production entries: an attach paused in
// server admission while quiesce-input commits its durable report
// must refuse once released, never commit a writable receipt after
// quiescence. The attach holds no lock while blocked in admission,
// so quiesce completes; the authoritative recheck under the lock
// then observes the committed report.
func TestAttachRefusesAfterQuiescenceWhenAdmittedLate(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	attachBody := lxAttachBody(t, nil)
	quiesceBody := lxQuiesceBody(t, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	admitted := fullLifecycleAdmitted()
	fx.lc.ServerAdmission = func(context.Context) (RealmAdmission, error) {
		once.Do(func() { close(started) })
		<-release
		return RealmAdmission{Admitted: admitted, RawGeneration: lxGeneration}, nil
	}
	type attachResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan attachResult, 1)
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: attachBody, Source: "parked", Admitted: fullLifecycleAdmitted(),
		})
		done <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("attach never reached server admission")
	}
	quiesceOutcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: quiesceBody, Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		close(release)
		t.Fatalf("quiesce: %v", err)
	}
	if quiesceOutcome.Mutation == nil || quiesceOutcome.Mutation.After != terminalbackend.StateQuiescing {
		close(release)
		t.Fatalf("quiesce mutation = %+v", quiesceOutcome.Mutation)
	}
	if proven, err := fx.lc.States.QuiesceBarrierProven(lxInstance); err != nil || !proven {
		close(release)
		t.Fatalf("barrier proven = %v, %v, want the durable report", proven, err)
	}
	close(release)
	var res attachResult
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("attach never returned after release")
	}
	if res.err == nil {
		t.Fatalf("attach admitted after durable quiescence: %+v", res.outcome.Attach)
	}
	requireLocalCode(t, res.err, "local_precondition_failed", "attach quiesced input")
	if len(res.outcome.Argv) != 0 {
		t.Fatalf("refused attach built argv %q", res.outcome.Argv)
	}
	if res.outcome.Attach != nil && res.outcome.Attach.InputAuthorized {
		t.Fatalf("refused attach authorized input: %+v", res.outcome.Attach)
	}
}

// TestAttachQuiesceOrderingOverlapQuiesce pins that the quiesce
// report commit and the attach receipt commit are mutually
// exclusive: an attach launched from inside the report write (via
// the state crash seam) must not complete until quiesce releases
// the lock, and must then refuse through the recheck. Without
// either side of the lock, or without the recheck, the attach
// completes mid-commit and the test fails.
func TestAttachQuiesceOrderingOverlapQuiesce(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	attachBody := lxAttachBody(t, nil)
	quiesceBody := lxQuiesceBody(t, nil)
	type attachResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan attachResult, 1)
	var fired atomic.Bool
	fx.lc.States.WithHooks(&StateHooks{
		AfterStage: func(path string) error {
			if !strings.Contains(path, "/outcomes/") {
				return nil
			}
			if fired.Swap(true) {
				return nil
			}
			go func() {
				outcome, err := fx.lc.Execute(context.Background(), OpRequest{
					Operation: "attach", Body: attachBody, Source: "parked", Admitted: fullLifecycleAdmitted(),
				})
				done <- attachResult{outcome: outcome, err: err}
			}()
			select {
			case res := <-done:
				t.Errorf("attach completed during the quiesce report write: err=%v attach=%+v", res.err, res.outcome.Attach)
				done <- res
			case <-time.After(200 * time.Millisecond):
			}
			return nil
		},
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: quiesceBody, Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("quiesce: %v", err)
	}
	if outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	if !fired.Load() {
		t.Fatal("report hook never fired")
	}
	var res attachResult
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("attach never returned after quiescence")
	}
	if res.err == nil {
		t.Fatalf("attach admitted after durable quiescence: %+v", res.outcome.Attach)
	}
	requireLocalCode(t, res.err, "local_precondition_failed", "attach quiesced input")
}

// TestAttachQuiesceOrderingOverlapBoundary pins the same mutual
// exclusion for the boundary report commit: the boundary wait runs
// outside the lock, but its report serializes with attach
// receipts, and an attach overlapping the commit refuses once the
// boundary proof lands.
func TestAttachQuiesceOrderingOverlapBoundary(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	attachBody := lxAttachBody(t, nil)
	boundaryBody := lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil)
	type attachResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan attachResult, 1)
	var fired atomic.Bool
	fx.lc.States.WithHooks(&StateHooks{
		AfterStage: func(path string) error {
			if !strings.Contains(path, "/outcomes/") {
				return nil
			}
			if fired.Swap(true) {
				return nil
			}
			go func() {
				outcome, err := fx.lc.Execute(context.Background(), OpRequest{
					Operation: "attach", Body: attachBody, Source: "parked", Admitted: fullLifecycleAdmitted(),
				})
				done <- attachResult{outcome: outcome, err: err}
			}()
			select {
			case res := <-done:
				t.Errorf("attach completed during the boundary report write: err=%v attach=%+v", res.err, res.outcome.Attach)
				done <- res
			case <-time.After(200 * time.Millisecond):
			}
			return nil
		},
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: boundaryBody, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("boundary: %v", err)
	}
	if outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
	if !fired.Load() {
		t.Fatal("report hook never fired")
	}
	var res attachResult
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("attach never returned after the boundary proof")
	}
	if res.err == nil {
		t.Fatalf("attach admitted after the durable boundary proof: %+v", res.outcome.Attach)
	}
	requireLocalCode(t, res.err, "local_precondition_failed", "attach quiesced input")
}

// TestExecuteAttachErrorsOnIncarnationReadFailure pins that an
// unreadable incarnation document fails attach closed: the
// barrier scan propagates the read failure instead of reading
// the incarnation as absent.
func TestExecuteAttachErrorsOnIncarnationReadFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	path := filepath.Join(fx.data, "states", "tmuxlifecycle", "incarnation", lxInstance+".json")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("unreadable incarnation admitted attach")
	} else if !strings.Contains(err.Error(), "read tmux instance incarnation") {
		t.Fatalf("attach error = %v, want the incarnation read failure", err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}

// TestExecuteAttachErrorsOnOutcomesDirReadFailure pins that an
// unreadable outcomes directory fails attach closed: the barrier
// scan propagates the directory read failure instead of reading
// the barrier as absent.
func TestExecuteAttachErrorsOnOutcomesDirReadFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	dir := filepath.Join(fx.data, "states", "tmuxlifecycle", "outcomes")
	if err := os.WriteFile(dir, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("unreadable outcomes directory admitted attach")
	} else if !strings.Contains(err.Error(), "read tmux operation outcomes:") {
		t.Fatalf("attach error = %v, want the outcomes directory read failure", err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}

// TestExecuteAttachErrorsOnOutcomeFileReadFailure pins that an
// unreadable outcome document fails attach closed: the barrier
// scan propagates the file read failure instead of skipping the
// document. The unreadable entry is a .json symlink to a
// directory, so the directory-entry pre-filter keeps it and the
// file read fails with EISDIR.
func TestExecuteAttachErrorsOnOutcomeFileReadFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	dir := filepath.Join(fx.data, "states", "tmuxlifecycle", "outcomes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "targetdir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "deadbeef.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("unreadable outcome document admitted attach")
	} else if !strings.Contains(err.Error(), "read tmux operation outcome:") {
		t.Fatalf("attach error = %v, want the outcome file read failure", err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}

// TestExecuteAttachRefusesWritableRootValidBody pins the attach
// entry of the unsafe-root custody class with a valid body: a
// writable root refuses before dispatch, and weakening the gate
// for attach alone would serve the attach instead of refusing.
func TestExecuteAttachRefusesWritableRootValidBody(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if err := os.Chmod(fx.root, 0o777); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(fx.root, 0o700) }()
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}
