//go:build !windows

package tmuxserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Revision-5 regression tests: the reviewer's four unchanged-production
// failures (two P1-A barrier sequences, two P1-B wrapper sequences),
// the composed-authorization joins, the incarnation rotation and its
// replay discipline, and the four P2-B clause gaps. Every test drives
// a production entry and asserts the literal refusal or the admitted
// effect.

// TestReviewExistingActiveMemoryCannotReopenQuiescedInput is the P1-A
// regression: an existing active record (the normal case) plus a
// quiesce whose effects ran and whose closure report persisted while
// the state write failed. A later writable attach must refuse: the
// incarnation-scoped closure proof decides admission, and the stale
// active memory is never consulted for its value.
func TestReviewExistingActiveMemoryCannotReopenQuiescedInput(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.lc.States.WithHooks(&StateHooks{AfterStage: func(path string) error {
		if strings.Contains(path, "/instances/") {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()})
	if err == nil {
		t.Fatal("quiesce unexpectedly succeeded")
	}
	fx.lc.States.WithHooks(nil)
	out, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, func(o map[string]any) { o["client_id"] = lxClientB }), Source: "parked", Admitted: fullLifecycleAdmitted()})
	if err == nil {
		t.Fatalf("writable attach admitted after durable closure with stale active memory: %+v argv=%v", out.Attach, out.Argv)
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

// TestReviewFailedStopCannotEraseQuiescence is the P1-A regression
// that needs no storage fault: a successful quiesce, then a
// request-stop with expired authorization carrying stale
// Source=active. The stop correctly fails, and its failure result
// must not overwrite the barrier: a later writable attach refuses,
// and the memory still records quiescing.
func TestReviewFailedStopCannotEraseQuiescence(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	body := lxStopBody(t, 60000, func(o map[string]any) {
		o["context"].(map[string]any)["authorization"].(map[string]any)["expires_at"] = "2026-09-01T01:00:00.000Z"
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: body, Source: "active", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("expired stop admitted")
	}
	if state, found, err := fx.lc.States.Lookup(lxInstance); err != nil || !found || state != terminalbackend.StateQuiescing {
		t.Fatalf("memory = %s, found=%v, err=%v, want quiescing preserved across the failed stop", state, found, err)
	}
	out, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()})
	if err == nil {
		t.Fatalf("failed stop erased successful quiescence: attach=%+v", out.Attach)
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

// TestExecuteFailedStopPreservesQuiescingMemory covers the second
// refusal path of the same invariant: the stop reaches the engine
// authorization (Source quiescing, expired grant) instead of the
// transition gate, and the pure refusal still preserves the
// authoritative memory record.
func TestExecuteFailedStopPreservesQuiescingMemory(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	body := lxStopBody(t, 60000, func(o map[string]any) {
		o["context"].(map[string]any)["authorization"].(map[string]any)["expires_at"] = "2026-09-01T01:00:00.000Z"
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: body, Source: "quiescing", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("expired stop admitted")
	}
	if state, found, err := fx.lc.States.Lookup(lxInstance); err != nil || !found || state != terminalbackend.StateQuiescing {
		t.Fatalf("memory = %s, found=%v, err=%v, want quiescing preserved across the authorization refusal", state, found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("writable attach admitted after the failed stop")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

// TestExecuteAdvisoryMemoryNeverDecidesAdmission proves the barrier
// construction directly: forged memory values in either direction
// change nothing. After a real closure, forcing the memory to active
// still refuses; without any closure, forcing the memory to
// quiescing still admits. Only the incarnation-scoped proof decides.
func TestExecuteAdvisoryMemoryNeverDecidesAdmission(t *testing.T) {
	t.Run("stale-active-cannot-reopen", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
		if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
			t.Fatal(err)
		}
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateActive); err != nil {
			t.Fatal(err)
		}
		if out, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
			t.Fatalf("stale active memory reopened the barrier: %+v", out.Attach)
		} else {
			requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
		}
	})
	t.Run("forged-quiescing-cannot-close", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err != nil {
			t.Fatal(err)
		}
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateQuiescing); err != nil {
			t.Fatal(err)
		}
		out, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, func(o map[string]any) { o["client_id"] = lxClientB }), Source: "parked", Admitted: fullLifecycleAdmitted()})
		if err != nil {
			t.Fatalf("forged quiescing memory closed an open input: %v", err)
		}
		if out.Attach == nil || !out.Attach.InputAuthorized {
			t.Fatalf("attach = %+v, want the writable admission", out.Attach)
		}
	})
}

// TestExecuteQuiesceStopRestoreReopensBarrier proves the legitimate
// reopening: quiesce, stop, then a fresh restore rotates the
// incarnation, so the prior closure no longer proves the barrier
// and a writable attach is admitted with the writable vector.
func TestExecuteQuiesceStopRestoreReopensBarrier(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("send-keys", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: lxStopBody(t, 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("new-session", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "stopped", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	if incarnation, found, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || !found || incarnation != lxCreateKey() {
		t.Fatalf("incarnation = %q, found=%v, err=%v, want the restore key", incarnation, found, err)
	}
	out, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatalf("writable attach refused after the restoring reopen: %v", err)
	}
	for _, element := range out.Argv {
		if element == "-r" {
			t.Fatalf("reopened attach carries the read-only vector: %q", out.Argv)
		}
	}
}

// TestExecuteRestoreReplayPreservesBarrier proves the replay
// discipline: a restore success, then a quiesce in the new
// incarnation, then an identical restore retry. The retry replays
// the stored success without rotating, so the newer closure still
// proves the barrier and a writable attach refuses.
func TestExecuteRestoreReplayPreservesBarrier(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	req := OpRequest{Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted()}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	replayed, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("restore replay refused: %v", err)
	}
	if replayed.Mutation == nil || replayed.Mutation.After != terminalbackend.StateParked {
		t.Fatalf("replay = %+v, want the parked success", replayed.Mutation)
	}
	if incarnation, _, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || incarnation != lxCreateKey() {
		t.Fatalf("incarnation = %q, err=%v, want the single fresh rotation untouched by the replay", incarnation, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("restore replay erased the newer closure")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

// TestExecuteCreateReplayPreservesBarrier is the create-side replay
// discipline: a create success, then a quiesce, then an identical
// create retry. The retry replays without rotating, so the closure
// still proves the barrier.
func TestExecuteCreateReplayPreservesBarrier(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	req := OpRequest{Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted()}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatalf("create replay refused: %v", err)
	}
	if incarnation, _, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || incarnation != lxCreateKey() {
		t.Fatalf("incarnation = %q, err=%v, want the single fresh rotation untouched by the replay", incarnation, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("create replay erased the newer closure")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

// TestExecuteCreateIncarnationWriteFailureFailsClosed pins the lost
// rotation bound: quiesce, stop, then a create whose incarnation
// install fails. The create refuses the integrity failure with the
// incarnation unchanged, so the prior closure still proves the
// barrier; the identical retry replays success without rotating
// (fail-closed, safe direction). Recovery through a new bootstrap
// is impossible under the landed one-bootstrap-per-session rule,
// pinned below; the stuck closure is over-strict in the safe
// direction until operator surgery clears the superseded reports.
func TestExecuteCreateIncarnationWriteFailureFailsClosed(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("send-keys", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: lxStopBody(t, 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.lc.States.WithHooks(&StateHooks{AfterStage: func(path string) error {
		if strings.Contains(path, "/incarnation/") {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("new-session", "", 0)
	createReq := OpRequest{Operation: "create", Body: lxCreateBody(t, nil), Source: "stopped", Admitted: fullLifecycleAdmitted()}
	if _, err := fx.lc.Execute(context.Background(), createReq); err == nil {
		t.Fatal("create with a lost rotation reported success")
	} else {
		requireLocalCode(t, err, "terminal_backend_integrity_failure", "create incarnation image")
	}
	fx.lc.States.WithHooks(nil)
	if _, found, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || found {
		t.Fatalf("incarnation found=%v, err=%v, want the rotation absent", found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("lost rotation reopened the prior closure")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
	// The identical retry replays the committed success but still
	// refuses to rotate: the stale closure keeps proving the
	// barrier (over-strict, safe direction) until a fresh key.
	if _, err := fx.lc.Execute(context.Background(), createReq); err != nil {
		t.Fatalf("create replay refused: %v", err)
	}
	if _, found, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || found {
		t.Fatalf("incarnation found=%v, err=%v, want the replay to skip the rotation", found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("create replay rotated past the prior closure")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
	// A fresh key cannot recover through any production entry: the
	// landed bootstrap store binds one bootstrap per session, so a
	// new pair refuses before any rotation could run. This pins the
	// mechanism behind the stuck-closed bound (owner: axpane).
	bootstrap2 := "0198f4c8-8e50-7f66-8f70-444444444442"
	if _, _, err := fx.lc.Bindings.Bind(lxSession, bootstrap2, axpane.Binding{SessionID: lxSession, OperationID: bootstrap2, TerminalInstanceID: lxInstance, BindingDigest: lxDigestA}); err == nil {
		t.Fatal("second bootstrap admitted for the session")
	} else {
		requireLandedCode(t, err, "idempotency_mismatch")
	}
}

// TestExecuteRestoreIncarnationWriteFailureFailsClosed is the
// restore-side lost rotation bound: quiesce, stop, then a restore
// whose incarnation install fails. The restore refuses the
// integrity failure, the identical retry replays success without
// rotating, and the prior closure keeps proving the barrier
// through both.
func TestExecuteRestoreIncarnationWriteFailureFailsClosed(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("send-keys", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "request-stop", Body: lxStopBody(t, 60000, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted()}); err != nil {
		t.Fatal(err)
	}
	fx.lc.States.WithHooks(&StateHooks{AfterStage: func(path string) error {
		if strings.Contains(path, "/incarnation/") {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("new-session", "", 0)
	restoreReq := OpRequest{Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "stopped", Admitted: fullLifecycleAdmitted()}
	if _, err := fx.lc.Execute(context.Background(), restoreReq); err == nil {
		t.Fatal("restore with a lost rotation reported success")
	} else {
		requireLocalCode(t, err, "terminal_backend_integrity_failure", "restore incarnation image")
	}
	fx.lc.States.WithHooks(nil)
	if _, found, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || found {
		t.Fatalf("incarnation found=%v, err=%v, want the rotation absent", found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("lost rotation reopened the prior closure")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
	if _, err := fx.lc.Execute(context.Background(), restoreReq); err != nil {
		t.Fatalf("restore replay refused: %v", err)
	}
	if _, found, err := fx.lc.States.LookupIncarnation(lxInstance); err != nil || found {
		t.Fatalf("incarnation found=%v, err=%v, want the replay to skip the rotation", found, err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}); err == nil {
		t.Fatal("restore replay rotated past the prior closure")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
}

// TestReviewWrapperBootstrapMustBindExecutedRestore is the P1-B
// regression: a valid launch decision for one bootstrap must not
// execute a restore naming another. The composed binding refuses
// before any effect.
func TestReviewWrapperBootstrapMustBindExecutedRestore(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(context.Context, string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	input := wDecideInput(t, true)
	input.BootstrapOperationID = lxOperation
	out, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{Decide: input, Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t)})
	if err == nil && out.Decision.Action == axpane.ActionLaunch && fx.runner.callCount() > 0 {
		t.Fatalf("launch decision for bootstrap %s executed restore for different bootstrap %s: commands=%v", input.BootstrapOperationID, lxBootstrap, fx.runner.subcommands())
	}
	if err == nil {
		t.Fatal("bootstrap-divergent restore admitted")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore bootstrap binding")
	if out.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", out.Decision.Action, out.Decision.Cause)
	}
	if out.Restore != nil {
		t.Fatalf("divergent restore executed: %+v", out.Restore)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed %v", fx.runner.subcommands())
	}
}

// TestReviewWrapperInstanceMustBindExecutedRestore is the P1-B
// regression for the instance member: a valid decision descriptor
// for one instance must not execute a restore targeting another.
func TestReviewWrapperInstanceMustBindExecutedRestore(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(context.Context, string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	input := wDecideInput(t, true)
	descriptor, err := axpane.BuildDescriptor(axpane.DescriptorParams{Binding: axpane.BindingRef{BindingDigest: input.HostBinding.TerminalBindingID, BackendID: input.HostBinding.BackendID, ImplementationVersion: input.HostBinding.ImplementationVersion, ProtocolVersion: input.HostBinding.ProtocolVersion, Generation: input.HostBinding.Generation}, InstanceID: lxClientB, Interactive: true, Columns: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	input.Descriptor = descriptor
	out, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{Decide: input, Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t)})
	if err == nil && out.Decision.Action == axpane.ActionLaunch && fx.runner.callCount() > 0 {
		t.Fatalf("launch decision for instance %s executed restore for different instance %s: commands=%v", lxClientB, lxInstance, fx.runner.subcommands())
	}
	if err == nil {
		t.Fatal("instance-divergent restore admitted")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore instance binding")
	if out.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", out.Decision.Action, out.Decision.Cause)
	}
	if out.Restore != nil {
		t.Fatalf("divergent restore executed: %+v", out.Restore)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed %v", fx.runner.subcommands())
	}
}

// wDivergentRestoreBody builds a restore body diverging on exactly
// one composed-binding member while staying closed and parseable:
// the join refuses before the backend entry ever runs.
func wDivergentRestoreBody(t *testing.T, mutate func(object map[string]any)) []byte {
	t.Helper()
	return lxRestoreBody(t, lxDigestA, mutate)
}

// TestWrapperRestoreRefusesDivergentSession pins the session arm of
// the composed binding: the decision authorizes lxSession while the
// restore names lxSessionB, so the entry refuses before any effect.
func TestWrapperRestoreRefusesDivergentSession(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(context.Context, string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	restore := wRestoreRequest(t)
	restore.Body = wDivergentRestoreBody(t, func(object map[string]any) {
		object["context"].(map[string]any)["session_id"] = lxSessionB
	})
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: restore,
	})
	if err == nil {
		t.Fatal("session-divergent restore admitted")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore session binding")
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore != nil {
		t.Fatalf("divergent restore executed: %+v", outcome.Restore)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed %v", fx.runner.subcommands())
	}
}

// TestWrapperRestoreRefusesDivergentGeneration pins the generation
// arm: the admitted descriptor binds lxGeneration while the restore
// names generation-two.
func TestWrapperRestoreRefusesDivergentGeneration(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(context.Context, string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	restore := wRestoreRequest(t)
	restore.Body = wDivergentRestoreBody(t, func(object map[string]any) {
		object["context"].(map[string]any)["backend_generation"] = "generation-two"
	})
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: restore,
	})
	if err == nil {
		t.Fatal("generation-divergent restore admitted")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore generation binding")
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore != nil {
		t.Fatalf("divergent restore executed: %+v", outcome.Restore)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed %v", fx.runner.subcommands())
	}
}

// TestWrapperRestoreRefusesDivergentBackend pins the backend arm:
// the admitted descriptor binds ax.tmux while the restore names
// ax.conpty. The wrapper detail proves the join fires before the
// backend entry's own lifecycle-backend refusal.
func TestWrapperRestoreRefusesDivergentBackend(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(context.Context, string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	restore := wRestoreRequest(t)
	restore.Body = wDivergentRestoreBody(t, func(object map[string]any) {
		object["context"].(map[string]any)["terminal_backend_id"] = "ax.conpty"
	})
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: restore,
	})
	if err == nil {
		t.Fatal("backend-divergent restore admitted")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore backend binding")
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore != nil {
		t.Fatalf("divergent restore executed: %+v", outcome.Restore)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed %v", fx.runner.subcommands())
	}
}

// TestExecuteStatusMalformedAttachedCountIsUnknown pins the
// parse-error sibling of the negative-count arm: a well-formed row
// whose count is not a number is unknown, never a count.
func TestExecuteStatusMalformedAttachedCountIsUnknown(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0junk\n", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, true, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("malformed attached count admitted")
	} else {
		requireEngineCode(t, err, "terminal_backend_unavailable")
	}
}

// TestExecuteAttachErrorsOnEmptyBarrierKey pins that a barrier
// report with no idempotency key at all is unknown, never an open
// input: the exactly-{} document errors instead of proving or
// disproving the barrier.
func TestExecuteAttachErrorsOnEmptyBarrierKey(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	dir := filepath.Join(fx.data, "states", "tmuxlifecycle", "outcomes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("empty-key report admitted writable attach")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance outcome key")
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}

// TestServerProberRefusesWritableRoot closes the prober-side custody
// cell: a 0777 runtime root refuses at Probe before any dial, the
// second side of the custody gate whose spawner side the prior
// revision closed.
func TestServerProberRefusesWritableRoot(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if err := os.Chmod(fx.root, 0o777); err != nil {
		t.Fatal(err)
	}
	prober := ServerProber{Root: fx.root, Platform: fx.lc.Platform, Dialer: &fakeDialer{outcomes: []DialOutcome{DialLive}}, Admit: func() (RealmAdmission, error) {
		return RealmAdmission{}, nil
	}}
	if _, err := prober.Probe(fx.socket); err == nil {
		t.Fatal("prober admitted writable root")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	}
}

// TestExecuteAttachErrorsOnStateReadFailure pins that an unreadable
// state record fails attach closed: the store health check
// propagates the read failure even though the admission verdict
// never consults the record's value.
func TestExecuteAttachErrorsOnStateReadFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	path := filepath.Join(fx.data, "states", "tmuxlifecycle", "instances", lxInstance+".json")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("unreadable state admitted attach")
	} else if !strings.Contains(err.Error(), "read tmux instance state") {
		t.Fatalf("attach error = %v, want the state read failure", err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused attach executed %v", fx.runner.subcommands())
	}
}

// TestInstanceIncarnationRoundTrip pins the incarnation store:
// rotate, replay, absence, and the closed refusal shapes.
func TestInstanceIncarnationRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenInstanceStates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.LookupIncarnation(lxInstance); err != nil || found {
		t.Fatalf("lookup = %v, %v, want absence", found, err)
	}
	if err := store.RecordIncarnation(lxInstance, lxCreateKey()); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.LookupIncarnation(lxInstance)
	if err != nil || !found || got != lxCreateKey() {
		t.Fatalf("lookup = %q, %v, %v, want the rotated key", got, found, err)
	}
	fresh := lxSession + "/0198f4c8-8e50-7f66-8f70-444444444442"
	if err := store.RecordIncarnation(lxInstance, fresh); err != nil {
		t.Fatal(err)
	}
	if got, _, err := store.LookupIncarnation(lxInstance); err != nil || got != fresh {
		t.Fatalf("lookup = %q, %v, want the fresh rotation", got, err)
	}
	if err := store.RecordIncarnation("not-a-uuid", lxCreateKey()); err == nil {
		t.Fatal("bad identity admitted")
	}
	if _, _, err := store.LookupIncarnation("not-a-uuid"); err == nil {
		t.Fatal("bad identity lookup admitted")
	}
	if err := store.RecordIncarnation(lxInstance, ""); err == nil {
		t.Fatal("empty key admitted")
	}
	incDir := filepath.Join(dir, "tmuxlifecycle", "incarnation")
	if err := os.MkdirAll(incDir, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := `{"schema":"urn:ax:schema:tmux-instance-incarnation","schema_version":"1.0.0","terminal_instance_id":"0198f4c8-8e50-7f66-8f70-111111111111","key":"stale-key"}`
	if err := os.WriteFile(filepath.Join(incDir, lxInstance+".json"), []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.LookupIncarnation(lxInstance); err == nil {
		t.Fatal("foreign incarnation document admitted")
	}
	empty := `{"schema":"urn:ax:schema:tmux-instance-incarnation","schema_version":"1.0.0","terminal_instance_id":"` + lxInstance + `","key":""}`
	if err := os.WriteFile(filepath.Join(incDir, lxInstance+".json"), []byte(empty), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.LookupIncarnation(lxInstance); err == nil {
		t.Fatal("empty-key incarnation document admitted")
	}
}

// TestQuiesceBarrierIgnoresSupersededIncarnation pins the
// incarnation scoping at the store: a closure report proves the
// barrier exactly in the incarnation it bound, and a rotation
// supersedes it without deleting it.
func TestQuiesceBarrierIgnoresSupersededIncarnation(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RecordOutcome(lxQuiesceKey(), OperationOutcome{Operation: "quiesce-input", InputClosedAt: "2026-09-01T12:00:00.000Z"}); err != nil {
		t.Fatal(err)
	}
	if proven, err := store.QuiesceBarrierProven(lxInstance); err != nil || !proven {
		t.Fatalf("proven = %v, %v, want the zero-incarnation proof", proven, err)
	}
	if err := store.RecordIncarnation(lxInstance, lxCreateKey()); err != nil {
		t.Fatal(err)
	}
	if proven, err := store.QuiesceBarrierProven(lxInstance); err != nil || proven {
		t.Fatalf("proven = %v, %v, want the superseded report ignored", proven, err)
	}
	if err := store.RecordOutcome(lxBoundaryKey("ax_checkpoint_boundary"), OperationOutcome{Operation: "wait-safe-boundary", BoundaryObservedAt: "2026-09-01T12:00:00.000Z", Incarnation: lxCreateKey()}); err != nil {
		t.Fatal(err)
	}
	if proven, err := store.QuiesceBarrierProven(lxInstance); err != nil || !proven {
		t.Fatalf("proven = %v, %v, want the current-incarnation proof", proven, err)
	}
}
