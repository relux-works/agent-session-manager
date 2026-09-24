//go:build !windows

package tmuxserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// This file pins the revision-9 rework findings through the
// production entries: the peer-census namespace integrity (a
// directory at a receipt-shaped name fails closed instead of
// reading as no peers), the deadline-bounded read-only probes (the
// stop close-confirm poll, the terminate-stale self-confirm, and
// the status live probes honor the operation deadline in-flight
// and conclude unknown — never a verdict — on expiry), and the
// receipt-validated same-client replay (a recorded receipt replays
// with another peer present; a conflicting retry still refuses
// idempotency_mismatch). Every test drives Lifecycle.Execute and
// asserts the literal effect.

// blockingRunner is a context-honoring scripted Runner for the
// wait-bound probes: instant answers serve at once, while a blocked
// subcommand waits for its context or the test release — whichever
// fires first decides. After the release every blocked call answers
// at once, so a widened bound drains instead of hanging.
type blockingRunner struct {
	mu      sync.Mutex
	calls   [][]string
	instant map[string]fakeResponse
	blocked map[string]fakeResponse
	release chan struct{}
}

func newBlockingRunner(release chan struct{}) *blockingRunner {
	return &blockingRunner{
		instant: map[string]fakeResponse{},
		blocked: map[string]fakeResponse{},
		release: release,
	}
}

func (runner *blockingRunner) serve(subcommand, stdout, stderr string, exit int) *blockingRunner {
	runner.instant[subcommand] = fakeResponse{stdout: stdout, stderr: stderr, exit: exit}
	return runner
}

func (runner *blockingRunner) hold(subcommand, stdout, stderr string, exit int) *blockingRunner {
	runner.blocked[subcommand] = fakeResponse{stdout: stdout, stderr: stderr, exit: exit}
	return runner
}

func (runner *blockingRunner) Run(ctx context.Context, argv []string) (RunResult, error) {
	runner.mu.Lock()
	runner.calls = append(runner.calls, append([]string(nil), argv...))
	instant := runner.instant
	blocked := runner.blocked
	release := runner.release
	runner.mu.Unlock()
	key := ""
	if len(argv) >= 4 {
		key = argv[3]
	}
	if resp, ok := instant[key]; ok {
		return RunResult{Stdout: []byte(resp.stdout), Stderr: []byte(resp.stderr), ExitCode: resp.exit}, nil
	}
	resp, ok := blocked[key]
	if !ok {
		return RunResult{}, fmt.Errorf("blocking runner has no script for %q", argv)
	}
	select {
	case <-ctx.Done():
		return RunResult{}, ctx.Err()
	case <-release:
		return RunResult{Stdout: []byte(resp.stdout), Stderr: []byte(resp.stderr), ExitCode: resp.exit}, nil
	}
}

func (runner *blockingRunner) subcommands() []string {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	var subcommands []string
	for _, call := range runner.calls {
		if len(call) >= 4 {
			subcommands = append(subcommands, call[3])
		}
	}
	return subcommands
}

// lxAttachReceiptBytes reads the raw stored receipt for one client.
func lxAttachReceiptBytes(t *testing.T, fx *lxFixture, client string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance, client+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestAttachPeerDirectoryFailsClosed pins the census namespace
// integrity at the attach entry: after a valid writable attach, a
// directory replacing that client's receipt file — keeping the
// original bytes inside — is unreadable receipt evidence, not
// proof the client is gone, so a different writable client with
// multi_attach absent refuses with no receipt, no argv, and no
// authorized input. The precision leg proves entries outside the
// receipt namespace still skip instead of failing the census.
func TestAttachPeerDirectoryFailsClosed(t *testing.T) {
	t.Run("receipt-dir refuses", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		if _, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
		}); err != nil {
			t.Fatal(err)
		}
		receipt := filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance, lxClient+".json")
		raw, err := os.ReadFile(receipt)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(receipt); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(receipt, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(receipt, "original.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		second := lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = lxClientB
		})
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: second, Source: "active", Admitted: withoutCapability("multi_attach"),
		})
		if err == nil {
			t.Fatalf("attach admitted past a directory at the peer receipt path: %+v", outcome.Attach)
		}
		if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
			t.Fatalf("lookup = %v, %v; want no receipt past the corrupt census", found, err)
		}
		if len(outcome.Argv) != 0 {
			t.Fatalf("refused attach built argv %q", outcome.Argv)
		}
		if outcome.Attach != nil {
			t.Fatalf("refused attach authorized input: %+v", outcome.Attach)
		}
	})
	t.Run("non-receipt entries still skipped", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		if _, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
		}); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance)
		if err := os.Mkdir(filepath.Join(dir, "attach-old.tmp"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "scratch"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("operator notes"), 0o600); err != nil {
			t.Fatal(err)
		}
		second := lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = lxClientB
		})
		if _, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: second, Source: "active", Admitted: fullLifecycleAdmitted(),
		}); err != nil {
			t.Fatalf("attach refused past non-receipt entries: %v", err)
		}
	})
}

// realDeadlineBody rewrites the operation deadline and the AX
// authorization window of a closed body map onto the real clock: a
// 200ms operation deadline with an hour-long authorization.
func realDeadlineBody(object map[string]any, now time.Time, deadline time.Time) {
	context := object["context"].(map[string]any)
	context["deadline_at"] = formatTimestamp(deadline)
	context["authorization"] = map[string]any{
		"lease_id":                  lxLease,
		"lease_epoch":               float64(7),
		"holder_host_id":            lxHost,
		"authorization_kind":        "control",
		"issued_at":                 formatTimestamp(now.Add(-time.Hour)),
		"expires_at":                formatTimestamp(now.Add(time.Hour)),
		"authorization_evidence_id": lxDigestA,
	}
}

// TestExecuteStopPollHonorsOperationDeadline pins the in-flight
// bound on the stop close-confirm poll: with a 200ms operation
// deadline, a has-session that stays blocked until the test
// release a second later must conclude at the deadline — not at
// the release — with the honest unknown (process_failed,
// status_first, no kill, no closure), because absence is proven
// only by positive observation. The release instant, not
// scheduler latency, decides the verdict.
func TestExecuteStopPollHonorsOperationDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.Now = time.Now
	release := make(chan struct{})
	runner := newBlockingRunner(release).serve("send-keys", "", "", 0).serve("kill-session", "", "", 0).
		hold("has-session", "", "", 0)
	fx.lc.Runner = runner
	now := time.Now().UTC()
	deadline := now.Add(200 * time.Millisecond)
	raw := lxStopBody(t, 3600000, func(object map[string]any) {
		realDeadlineBody(object, now, deadline)
	})
	type stopResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan stopResult, 1)
	start := time.Now()
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "request-stop", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
		})
		done <- stopResult{outcome: outcome, err: err}
	}()
	releaseAt := time.NewTimer(time.Until(deadline) + time.Second)
	defer releaseAt.Stop()
	var res stopResult
	received := false
	select {
	case res = <-done:
		received = true
	case <-releaseAt.C:
	}
	close(release)
	if !received {
		res = <-done
		t.Fatalf("stop poll ignored the 200ms deadline (blocked until the release): err=%v", res.err)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("stop refused only after %v, want the bound verdict before the release", elapsed)
	}
	requireEngineCode(t, res.err, "terminal_backend_process_failed")
	if res.outcome.Mutation == nil {
		t.Fatalf("refused stop omits its mutation: err=%v", res.err)
	}
	if len(res.outcome.Mutation.Effects) != 1 || res.outcome.Mutation.Effects[0] != terminalbackend.EffectGracefulStopRequested {
		t.Fatalf("effects = %v, want exactly the graceful request", res.outcome.Mutation.Effects)
	}
	if res.outcome.Mutation.After != terminalbackend.StateUnavailable {
		t.Fatalf("after = %s, want unavailable", res.outcome.Mutation.After)
	}
	if string(res.outcome.Mutation.Disposition) != "status_first" {
		t.Fatalf("disposition = %q, want status_first", res.outcome.Mutation.Disposition)
	}
	if res.outcome.ProcessClosed || res.outcome.StoreClosed {
		t.Fatalf("timed-out poll reported closure: %+v", res.outcome)
	}
	for _, command := range runner.subcommands() {
		if command == "kill-session" {
			t.Fatalf("kill-session issued for an unproven poll: err=%v", res.err)
		}
	}
}

// TestExecuteTerminateConfirmHonorsOperationDeadline pins the same
// in-flight bound on the terminate-stale self-confirm: after a
// failed kill, a has-session that stays blocked until the release
// must conclude unknown at the 200ms operation deadline — never
// termination — with no termination evidence and no second kill.
func TestExecuteTerminateConfirmHonorsOperationDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.Now = time.Now
	release := make(chan struct{})
	runner := newBlockingRunner(release).serve("kill-session", "", "", 1).
		hold("has-session", "", "can't find session: "+lxInstance, 1)
	fx.lc.Runner = runner
	now := time.Now().UTC()
	deadline := now.Add(200 * time.Millisecond)
	raw := lxTerminateBody(t, func(object map[string]any) {
		realDeadlineBody(object, now, deadline)
		object["context"].(map[string]any)["authorization"].(map[string]any)["authorization_kind"] = "force_stale"
	})
	presented, winner := terminateFencing()
	type terminateResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan terminateResult, 1)
	start := time.Now()
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "terminate-stale", Body: raw, Source: "stale_fenced",
			Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
			HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
		})
		done <- terminateResult{outcome: outcome, err: err}
	}()
	releaseAt := time.NewTimer(time.Until(deadline) + time.Second)
	defer releaseAt.Stop()
	var res terminateResult
	received := false
	select {
	case res = <-done:
		received = true
	case <-releaseAt.C:
	}
	close(release)
	if !received {
		res = <-done
		t.Fatalf("terminate confirm ignored the 200ms deadline (blocked until the release): err=%v", res.err)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("terminate refused only after %v, want the bound verdict before the release", elapsed)
	}
	requireLocalCode(t, res.err, "terminal_backend_process_failed", "backend effect uncertain")
	if res.outcome.Mutation == nil {
		t.Fatalf("refused terminate omits its mutation: err=%v", res.err)
	}
	if len(res.outcome.Mutation.Effects) != 0 {
		t.Fatalf("effects = %v, want none performed", res.outcome.Mutation.Effects)
	}
	if res.outcome.TargetClosed {
		t.Fatalf("timed-out confirm reported termination: %+v", res.outcome)
	}
	if got := runner.subcommands(); len(got) != 2 || got[0] != "kill-session" || got[1] != "has-session" {
		t.Fatalf("calls = %v, want the failed kill plus the bounded confirm", got)
	}
}

// TestExecuteStatusProbeHonorsOperationDeadline pins the query
// deadline on the status live probes: a list-panes that stays
// blocked until the release must conclude unknown at the 200ms
// deadline — never absent — with no report.
func TestExecuteStatusProbeHonorsOperationDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.Now = time.Now
	recordBinding(t, fx, lxDigestA)
	release := make(chan struct{})
	runner := newBlockingRunner(release).serve("list-sessions", lxInstance+"|0\n", "", 0).
		hold("list-panes", "sh\n", "", 0)
	fx.lc.Runner = runner
	now := time.Now().UTC()
	deadline := now.Add(200 * time.Millisecond)
	raw := lxStatusBody(t, true, false, func(object map[string]any) {
		object["deadline_at"] = formatTimestamp(deadline)
	})
	type statusResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan statusResult, 1)
	start := time.Now()
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: raw, Admitted: fullLifecycleAdmitted(),
		})
		done <- statusResult{outcome: outcome, err: err}
	}()
	releaseAt := time.NewTimer(time.Until(deadline) + time.Second)
	defer releaseAt.Stop()
	var res statusResult
	received := false
	select {
	case res = <-done:
		received = true
	case <-releaseAt.C:
	}
	close(release)
	if !received {
		res = <-done
		t.Fatalf("status probe ignored the 200ms deadline (blocked until the release): err=%v", res.err)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("status refused only after %v, want the bound verdict before the release", elapsed)
	}
	requireEngineCode(t, res.err, "terminal_backend_unavailable")
	if res.outcome.Status != nil {
		t.Fatalf("timed-out probe reported %+v, want no report", res.outcome.Status)
	}
	if got := runner.subcommands(); len(got) != 1 || got[0] != "list-panes" {
		t.Fatalf("calls = %v, want exactly the bounded probe", got)
	}
}

// attachPeerFixture attaches client A with the given input shape
// and then an input-authorized client B, both with the full
// capability set, and returns the fixture plus A's exact body for
// the replay.
func attachPeerFixture(t *testing.T, inputA bool) (*lxFixture, []byte) {
	t.Helper()
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	first := lxAttachBody(t, func(object map[string]any) {
		object["input_authorized"] = inputA
		object["authorization"] = lxAttachAuth("local_only", inputA)
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: first, Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	second := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	return fx, first
}

// TestAttachSameClientRetryWithPeerPresent pins the
// receipt-validated replay: with another peer recorded, the same
// client re-attaching its identical body replays on the transport
// capability alone — in both the input-authorized and the
// read-only shapes — while a retry that changes transport or
// input still refuses idempotency_mismatch. The exemption keys on
// the recorded receipt, never on the bare client ID.
func TestAttachSameClientRetryWithPeerPresent(t *testing.T) {
	t.Run("input", func(t *testing.T) {
		fx, first := attachPeerFixture(t, true)
		before := lxAttachReceiptBytes(t, fx, lxClient)
		reduced := admittedWith("local_attach")
		replayed, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: first, Source: "active", Admitted: reduced,
		})
		if err != nil {
			t.Fatalf("same-client retry refused with a peer present: %v", err)
		}
		if replayed.Attach == nil || replayed.Attach.ClientMirrorID != lxClient || !replayed.Attach.InputAuthorized {
			t.Fatalf("attach = %+v, want the replayed writable receipt", replayed.Attach)
		}
		if after := lxAttachReceiptBytes(t, fx, lxClient); string(before) != string(after) {
			t.Fatalf("replay rewrote the receipt: before %s after %s", before, after)
		}
	})
	t.Run("readonly", func(t *testing.T) {
		fx, first := attachPeerFixture(t, false)
		before := lxAttachReceiptBytes(t, fx, lxClient)
		reduced := admittedWith("local_attach")
		replayed, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: first, Source: "active", Admitted: reduced,
		})
		if err != nil {
			t.Fatalf("same-client read-only retry refused with a peer present: %v", err)
		}
		if replayed.Attach == nil || replayed.Attach.ClientMirrorID != lxClient || replayed.Attach.InputAuthorized {
			t.Fatalf("attach = %+v, want the replayed read-only receipt", replayed.Attach)
		}
		if after := lxAttachReceiptBytes(t, fx, lxClient); string(before) != string(after) {
			t.Fatalf("replay rewrote the receipt: before %s after %s", before, after)
		}
	})
	t.Run("conflict", func(t *testing.T) {
		fx, _ := attachPeerFixture(t, true)
		flipped := lxAttachBody(t, func(object map[string]any) {
			object["input_authorized"] = false
			object["authorization"] = lxAttachAuth("local_only", false)
		})
		before := lxAttachReceiptBytes(t, fx, lxClient)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: flipped, Source: "active", Admitted: admittedWith("local_attach"),
		})
		requireEngineCode(t, err, "idempotency_mismatch")
		if after := lxAttachReceiptBytes(t, fx, lxClient); string(before) != string(after) {
			t.Fatalf("conflicting retry rewrote the receipt: before %s after %s", before, after)
		}
		if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || !found {
			t.Fatalf("peer lookup = %v, %v; want the peer receipt untouched", found, err)
		}
	})
}
