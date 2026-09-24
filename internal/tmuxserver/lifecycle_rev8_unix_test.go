//go:build !windows

package tmuxserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// This file pins the revision-8 rework findings through the
// production entries: the stop escalation revalidation (rev7 P1-A),
// the deadline/cancellation-bounded waits (rev7 P2), the attach
// overlap fail-closed gate (rev7 P1-B narrowed to this leaf's bound,
// with the liveness census owned by TASK-260922-vcx6yo), and the measured effect-boundary audit over
// every other multi-command or waiting effect. Every test drives
// Lifecycle.Execute (or the documented store entry) and asserts the
// literal effect — the absence of the forbidden command, receipt,
// or wait — never a string report.

// TestExecuteStopRefusesStaleFactsAfterPoll pins the escalation
// revalidation: the graceful interrupt runs, exactly one admission
// fact changes during the has-session poll, and the escalation must
// refuse without issuing kill-session. The graceful wait expiring
// is the escalation trigger — a graceful timeout escalates — but
// an operation deadline passed, an authorization lapsed, or a
// generation rotated during the poll refuses instead. The test
// asserts the kill absence, not merely the final error: the
// returned result already lists graceful_stop_requested, and no
// late refusal un-runs a kill.
func TestExecuteStopRefusesStaleFactsAfterPoll(t *testing.T) {
	for _, fact := range []string{"generation", "authorization", "deadline", "deadline-instant"} {
		t.Run(fact, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			generation := lxGeneration
			fx.lc.CurrentGeneration = func() string { return generation }
			raw := lxStopBody(t, 5, func(object map[string]any) {
				context := object["context"].(map[string]any)
				switch fact {
				case "authorization":
					context["authorization"].(map[string]any)["expires_at"] = "2026-09-01T12:00:00.003Z"
				case "deadline":
					context["deadline_at"] = "2026-09-01T12:00:00.003Z"
				case "deadline-instant":
					context["deadline_at"] = "2026-09-01T12:00:00.010Z"
				}
			})
			// The kill and the second confirm are queued so a
			// regression fails on the kill-absence assertion
			// with the forbidden command recorded — not on an
			// unscripted-call error.
			fx.runner.queue("send-keys", "", 0).queue("has-session", "", 0).
				queueFull("has-session", "", "can't find session: "+lxInstance, 1).
				queue("kill-session", "", 0)
			seen := false
			fx.runner.onRun = func(argv []string) {
				if len(argv) >= 4 && argv[3] == "has-session" && !seen {
					seen = true
					if fact == "generation" {
						generation = "generation-two"
					}
					fx.clock.Sleep(10 * time.Millisecond)
				}
			}
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "request-stop", Body: raw, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
			})
			for _, command := range fx.runner.subcommands() {
				if command == "kill-session" {
					t.Fatalf("kill-session issued after %s went stale during the poll: err=%v mutation=%+v", fact, err, outcome.Mutation)
				}
			}
			switch fact {
			case "generation":
				requireEngineCode(t, err, "terminal_backend_stale_generation")
			case "authorization":
				requireEngineCode(t, err, "terminal_backend_unauthorized")
			case "deadline", "deadline-instant":
				requireEngineCode(t, err, "stop_timeout")
			}
			if outcome.Mutation == nil {
				t.Fatalf("refused stop omits its mutation: err=%v", err)
			}
			if len(outcome.Mutation.Effects) != 1 || outcome.Mutation.Effects[0] != terminalbackend.EffectGracefulStopRequested {
				t.Fatalf("effects = %v, want exactly the graceful request", outcome.Mutation.Effects)
			}
			if outcome.Mutation.After != terminalbackend.StateUnavailable {
				t.Fatalf("after = %s, want unavailable", outcome.Mutation.After)
			}
		})
	}
}

// TestExecuteStopEscalatesAfterGracefulTimeoutAlone is the control
// for the escalation revalidation: the graceful wait expiring with
// every other fact still valid escalates exactly once and closes.
// A graceful timeout escalates; only the other facts gate it.
func TestExecuteStopEscalatesAfterGracefulTimeoutAlone(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("send-keys", "", 0).queue("has-session", "", 0).
		queue("kill-session", "", 0).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	seen := false
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "has-session" && !seen {
			seen = true
			fx.clock.Sleep(10 * time.Millisecond)
		}
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "request-stop", Body: lxStopBody(t, 5, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.runner.subcommands(); len(got) != 4 || got[0] != "send-keys" || got[1] != "has-session" || got[2] != "kill-session" || got[3] != "has-session" {
		t.Fatalf("calls = %v, want the graceful poll, one escalation, and the re-confirm", got)
	}
	if outcome.Mutation.After != terminalbackend.StateStopped {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}

// TestExecuteQuiesceRefusesStaleFactsBetweenCommands pins the
// within-effect revalidation on the quiesce path: the lock runs,
// exactly one admission fact changes during its round trip, and
// the detach must refuse without issuing. The lock/detach pair is
// one effect with two destructive commands; the boundary between
// them re-proves the facts the engine checked before the effect.
// The deadline fact probes both the strictly-past instant and the
// on-deadline instant: both refuse, and the authorization expired
// thirty minutes before the recheck, so the minus-one-hour
// narrowing admits it while the unmutated clock still refuses.
func TestExecuteQuiesceRefusesStaleFactsBetweenCommands(t *testing.T) {
	for _, fact := range []string{"generation", "authorization", "deadline", "deadline-instant"} {
		t.Run(fact, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			generation := lxGeneration
			fx.lc.CurrentGeneration = func() string { return generation }
			raw := lxQuiesceBody(t, func(object map[string]any) {
				if fact == "authorization" {
					object["context"].(map[string]any)["authorization"].(map[string]any)["expires_at"] = "2026-09-01T13:30:00.000Z"
				}
			})
			// The detach is queued so a regression fails on
			// the detach-absence assertion with the
			// forbidden command recorded.
			fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
			fx.runner.onRun = func(argv []string) {
				if len(argv) < 4 || argv[3] != "lock-session" {
					return
				}
				switch fact {
				case "generation":
					generation = "generation-two"
				case "authorization":
					fx.clock.Sleep(2 * time.Hour)
				case "deadline":
					fx.clock.Sleep(7 * time.Hour)
				case "deadline-instant":
					fx.clock.Sleep(6 * time.Hour)
				}
			}
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "quiesce-input", Body: raw, Source: "active", Admitted: fullLifecycleAdmitted(),
			})
			for _, command := range fx.runner.subcommands() {
				if command == "detach-client" {
					t.Fatalf("detach-client issued after %s went stale during the lock: err=%v mutation=%+v", fact, err, outcome.Mutation)
				}
			}
			switch fact {
			case "generation":
				requireEngineCode(t, err, "terminal_backend_stale_generation")
			case "authorization":
				requireEngineCode(t, err, "terminal_backend_unauthorized")
			case "deadline", "deadline-instant":
				requireEngineCode(t, err, "terminal_backend_timeout")
			}
			if outcome.Mutation == nil {
				t.Fatalf("refused quiesce omits its mutation: err=%v", err)
			}
			if len(outcome.Mutation.Effects) != 0 {
				t.Fatalf("effects = %v, want none performed", outcome.Mutation.Effects)
			}
		})
	}
}

// TestBackendPostWaitBoundaryRefusesWithoutRecheck pins the
// fail-closed wiring: a post-wait destructive boundary with no
// configured revalidation refuses instead of running blind. The
// production entries always configure it; this helper-level pin
// documents the nil contract both guarded sites share.
func TestBackendPostWaitBoundaryRefusesWithoutRecheck(t *testing.T) {
	t.Run("escalation", func(t *testing.T) {
		runner := newFakeRunner().queue("has-session", "", 0).queue("kill-session", "", 0)
		reqBackend := effectBackend(t, terminalbackend.OperationRequestStop, runner)
		reqBackend.recheck = nil
		reqBackend.deadline = lxNow().Add(-time.Minute)
		_, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed)
		requireEngineCode(t, err, "terminal_backend_protocol_error")
		for _, command := range runner.subcommands() {
			if command == "kill-session" {
				t.Fatalf("kill-session issued with no revalidation configured")
			}
		}
	})
	t.Run("detach", func(t *testing.T) {
		runner := newFakeRunner().queue("lock-session", "", 0).queue("detach-client", "", 0)
		reqBackend := effectBackend(t, terminalbackend.OperationQuiesceInput, runner)
		reqBackend.recheck = nil
		_, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectInputClosed)
		requireEngineCode(t, err, "terminal_backend_protocol_error")
		for _, command := range runner.subcommands() {
			if command == "detach-client" {
				t.Fatalf("detach-client issued with no revalidation configured")
			}
		}
	})
}

// TestEffectBoundaryAuditCreateIssuesSingleCommand probes the
// create path for a post-wait destructive interval: the generation
// rotates during the wrapper's only command round trip, and the
// path must still succeed with exactly that one command. The
// script holds no second command, so any future post-wait command
// fails the unscripted call and reddens this probe.
func TestEffectBoundaryAuditCreateIssuesSingleCommand(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	generation := lxGeneration
	fx.lc.CurrentGeneration = func() string { return generation }
	fx.runner.queue("new-session", "", 0)
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "new-session" {
			generation = "generation-two"
		}
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.runner.subcommands(); len(got) != 1 || got[0] != "new-session" {
		t.Fatalf("calls = %v, want exactly the one wrapper command", got)
	}
	if outcome.Mutation.After != terminalbackend.StateActive {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}

// TestEffectBoundaryAuditRestoreIssuesSingleCommand probes the
// restore path the same way: a rotation during its only command
// round trip changes nothing further, because nothing further
// runs. Any future post-wait command reddens the unscripted call.
func TestEffectBoundaryAuditRestoreIssuesSingleCommand(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	generation := lxGeneration
	fx.lc.CurrentGeneration = func() string { return generation }
	fx.runner.queue("new-session", "", 0)
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "new-session" {
			generation = "generation-two"
		}
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.runner.subcommands(); len(got) != 1 || got[0] != "new-session" {
		t.Fatalf("calls = %v, want exactly the one wrapper command", got)
	}
	if outcome.Mutation.After != terminalbackend.StateParked {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}

// TestEffectBoundaryAuditTerminateIssuesNoPostWaitCommand probes
// the terminate-stale tail: the kill runs first (the loop
// rechecked before the effect), then the generation rotates
// during the re-confirm poll. No destructive command follows
// that wait — the tail only refuses — so the kill count must stay
// exactly one however the facts moved.
func TestEffectBoundaryAuditTerminateIssuesNoPostWaitCommand(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	generation := lxGeneration
	fx.lc.CurrentGeneration = func() string { return generation }
	fx.runner.queue("kill-session", "", 0).queue("has-session", "", 0)
	seen := false
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "has-session" && !seen {
			seen = true
			generation = "generation-two"
			fx.clock.Sleep(7 * time.Hour)
		}
	}
	presented, winner := terminateFencing()
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	requireLocalCode(t, err, "terminal_backend_process_failed", "backend effect refused")
	if outcome.Mutation == nil {
		t.Fatalf("refused terminate omits its mutation")
	}
	kills := 0
	for _, command := range fx.runner.subcommands() {
		if command == "kill-session" {
			kills++
		}
	}
	if kills != 1 {
		t.Fatalf("kill-session issued %d times, want exactly the pre-wait one: %v", kills, fx.runner.subcommands())
	}
	if got := fx.runner.subcommands(); len(got) != 2 || got[0] != "kill-session" || got[1] != "has-session" {
		t.Fatalf("calls = %v, want the kill plus the one re-confirm probe", got)
	}
}

// TestEffectBoundaryAuditBoundaryTailIsReadOnly probes the
// safe-boundary tail: a rotation during the wrapper-signal wait
// must not route to any destructive command — the only
// post-signal command is the read-only provider-rows probe, and
// the observed proof still commits.
func TestEffectBoundaryAuditBoundaryTailIsReadOnly(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	generation := lxGeneration
	fx.lc.CurrentGeneration = func() string { return generation }
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "wait-for" {
			generation = "generation-two"
		}
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "provider_quiescence", 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.runner.subcommands(); len(got) != 2 || got[0] != "wait-for" || got[1] != "list-panes" {
		t.Fatalf("calls = %v, want the signal wait plus the read-only rows probe", got)
	}
	if outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}

// TestAttachWaitBoundedByOperationDeadline pins the waiting half of
// "a deadline cancels waiting, not a committed effect": with a
// 200ms operation deadline and an hour-long authorization, a
// blocked server admission or a held barrier lock must refuse the
// timeout at the deadline — not when the waiter is released a
// second later — and the late release must commit nothing. The
// admission fake is deliberately non-cooperative (it ignores
// cancellation and blocks on the test's release channel), so the
// test proves the entry bounds the wait itself rather than relying
// on adapter cooperation. The release instant, not scheduler
// latency, decides the verdict.
func TestAttachWaitBoundedByOperationDeadline(t *testing.T) {
	for _, where := range []string{"admission", "barrier"} {
		t.Run(where, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			fx.lc.Now = time.Now
			now := time.Now().UTC()
			deadline := now.Add(200 * time.Millisecond)
			body := lxAttachBody(t, func(object map[string]any) {
				object["deadline_at"] = formatTimestamp(deadline)
				auth := lxAttachAuth("local_only", true)
				auth["issued_at"] = formatTimestamp(now.Add(-time.Hour))
				auth["expires_at"] = formatTimestamp(now.Add(time.Hour))
				object["authorization"] = auth
			})
			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			finished := make(chan struct{})
			previous := fx.lc.ServerAdmission
			fx.lc.ServerAdmission = func(ctx context.Context) (RealmAdmission, error) {
				entered <- struct{}{}
				if where == "admission" {
					<-release
					close(finished)
				}
				return previous(ctx)
			}
			held := false
			if where == "barrier" {
				fx.lc.barrierMu.Lock()
				held = true
			}
			released := false
			cleanup := func() {
				if where == "admission" && !released {
					released = true
					close(release)
				}
				if held {
					held = false
					fx.lc.barrierMu.Unlock()
				}
			}
			defer cleanup()
			type attachResult struct {
				outcome OpOutcome
				err     error
			}
			done := make(chan attachResult, 1)
			start := time.Now()
			go func() {
				outcome, err := fx.lc.Execute(context.Background(), OpRequest{
					Operation: "attach", Body: body, Source: "parked", Admitted: fullLifecycleAdmitted(),
				})
				done <- attachResult{outcome: outcome, err: err}
			}()
			select {
			case <-entered:
			case res := <-done:
				t.Fatalf("attach returned before reaching the wait: %v", res.err)
			case <-time.After(10 * time.Second):
				t.Fatal("attach never reached the wait")
			}
			// The release lands a second after the deadline:
			// a correct entry already refused by then, and
			// only a wait-ignoring entry is still blocked.
			releaseAt := time.NewTimer(time.Until(deadline) + time.Second)
			defer releaseAt.Stop()
			var res attachResult
			received := false
			elapsed := time.Since(start)
			select {
			case res = <-done:
				received = true
				elapsed = time.Since(start)
			case <-releaseAt.C:
			}
			cleanup()
			if where == "admission" {
				select {
				case <-finished:
				case <-time.After(10 * time.Second):
					t.Fatal("released admission never returned")
				}
			}
			if !received {
				res = <-done
				t.Fatalf("attach %s wait ignored the 200ms deadline (blocked until the release): err=%v", where, res.err)
			}
			requireLocalCode(t, res.err, "terminal_backend_timeout", "operation deadline")
			if elapsed >= time.Second {
				t.Fatalf("attach refused only after %v, want the deadline refusal before the release", elapsed)
			}
			if len(res.outcome.Argv) != 0 {
				t.Fatalf("refused attach built argv %q", res.outcome.Argv)
			}
			// No late receipt: the release and a settle
			// window must leave the store empty — nothing
			// commits after the caller timed out.
			time.Sleep(100 * time.Millisecond)
			if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient); err != nil || found {
				t.Fatalf("lookup = %v, %v; want no late receipt after the release", found, err)
			}
			t.Logf("%s wait refused at the deadline after %v", where, elapsed)
		})
	}
}

// TestQuiesceBarrierWaitBoundedByOperationDeadline pins the uniform
// barrier contract on the quiesce span: with a 200ms operation
// deadline, a quiesce blocked on a held barrier refuses the
// timeout at the deadline — before any engine effect runs — not
// when the holder releases a second later.
func TestQuiesceBarrierWaitBoundedByOperationDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.Now = time.Now
	now := time.Now().UTC()
	deadline := now.Add(200 * time.Millisecond)
	auth := map[string]any{
		"lease_id":                  lxLease,
		"lease_epoch":               float64(7),
		"holder_host_id":            lxHost,
		"authorization_kind":        "control",
		"issued_at":                 formatTimestamp(now.Add(-time.Hour)),
		"expires_at":                formatTimestamp(now.Add(time.Hour)),
		"authorization_evidence_id": lxDigestA,
	}
	raw := lxQuiesceBody(t, func(object map[string]any) {
		context := object["context"].(map[string]any)
		context["deadline_at"] = formatTimestamp(deadline)
		context["authorization"] = auth
	})
	fx.lc.barrierMu.Lock()
	held := true
	defer func() {
		if held {
			fx.lc.barrierMu.Unlock()
		}
	}()
	// Queued for the regression shape only: a wait-ignoring
	// entry proceeds after the release and succeeds, which the
	// timeout assertion below reddens.
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	type quiesceResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan quiesceResult, 1)
	start := time.Now()
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "quiesce-input", Body: raw, Source: "active", Admitted: fullLifecycleAdmitted(),
		})
		done <- quiesceResult{outcome: outcome, err: err}
	}()
	// The release lands a second after the deadline: a correct
	// entry already refused by then, and only a wait-ignoring
	// entry is still blocked.
	releaseAt := time.NewTimer(time.Until(deadline) + time.Second)
	defer releaseAt.Stop()
	var res quiesceResult
	received := false
	select {
	case res = <-done:
		received = true
	case <-releaseAt.C:
	}
	held = false
	fx.lc.barrierMu.Unlock()
	if !received {
		res = <-done
		t.Fatalf("quiesce barrier wait ignored the 200ms deadline (blocked until the release): err=%v", res.err)
	}
	requireLocalCode(t, res.err, "terminal_backend_timeout", "operation deadline")
	if res.outcome.Mutation != nil {
		t.Fatalf("mutation = %+v, want no result before the refused wait", res.outcome.Mutation)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("quiesce refused only after %v, want the deadline refusal before the release", elapsed)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused quiesce execed %d commands", fx.runner.callCount())
	}
}

// TestBoundaryCommitWaitBoundedByOperationDeadline pins the uniform
// barrier contract on the boundary report commit: the wrapper
// signal commits outside the lock, the operation deadline passes
// during the wait, and the held barrier then refuses the row
// timeout against the already-committed proof — the success
// mutation stands, only the report is unrecorded. The deadline
// passed thirty minutes before the commit, so the plus-one-hour
// wait narrowing admits the wait while the unmutated bound still
// refuses; the holder releases a second after the verdict window,
// so a widened wait drains instead of hanging.
func TestBoundaryCommitWaitBoundedByOperationDeadline(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "wait-for" {
			fx.clock.Sleep(6*time.Hour + 30*time.Minute)
		}
	}
	fx.lc.barrierMu.Lock()
	held := true
	defer func() {
		if held {
			fx.lc.barrierMu.Unlock()
		}
	}()
	type boundaryResult struct {
		outcome OpOutcome
		err     error
	}
	done := make(chan boundaryResult, 1)
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "provider_quiescence", 60000, nil),
			Source: "quiescing", Admitted: fullLifecycleAdmitted(),
		})
		done <- boundaryResult{outcome: outcome, err: err}
	}()
	releaseAt := time.NewTimer(time.Second)
	defer releaseAt.Stop()
	var res boundaryResult
	received := false
	select {
	case res = <-done:
		received = true
	case <-releaseAt.C:
	}
	held = false
	fx.lc.barrierMu.Unlock()
	if !received {
		select {
		case res = <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("boundary report commit blocked on the held barrier past its expired deadline")
		}
	}
	requireLocalCode(t, res.err, "quiesce_timeout", "operation deadline")
	if res.outcome.Mutation == nil || res.outcome.Mutation.After != terminalbackend.StateQuiescing {
		t.Fatalf("mutation = %+v, want the committed proof", res.outcome.Mutation)
	}
	if res.outcome.BoundaryObservedAt != "" {
		t.Fatalf("observed_at = %q, want no unrecorded report", res.outcome.BoundaryObservedAt)
	}
}

// TestAttachWaitCancelledReturnsPromptly pins the cancellation half
// of the barrier contract: an expired and context-cancelled attach
// waiting on a held barrier returns without any external unlock.
// The holder releases only after the verdict, for goroutine
// cleanup.
func TestAttachWaitCancelledReturnsPromptly(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	arrived := make(chan struct{}, 1)
	previous := fx.lc.ServerAdmission
	fx.lc.ServerAdmission = func(ctx context.Context) (RealmAdmission, error) {
		admission, err := previous(ctx)
		arrived <- struct{}{}
		return admission, err
	}
	fx.lc.barrierMu.Lock()
	defer fx.lc.barrierMu.Unlock()
	done := make(chan error, 1)
	body := lxAttachBody(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_, err := fx.lc.Execute(ctx, OpRequest{
			Operation: "attach", Body: body, Source: "parked", Admitted: fullLifecycleAdmitted(),
		})
		done <- err
	}()
	select {
	case <-arrived:
	case <-time.After(10 * time.Second):
		t.Fatal("attach never reached server admission")
	}
	fx.clock.Sleep(7 * time.Hour)
	cancel()
	select {
	case err := <-done:
		requireLocalCode(t, err, "terminal_backend_timeout", "operation deadline")
		if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient); err != nil || found {
			t.Fatalf("lookup = %v, %v; want no receipt", found, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("expired and cancelled attach stayed blocked on the held barrier")
	}
}

// TestQuiesceMalformedDeadlineRefusesBeforeBarrierWait probes the
// unparseable-deadline corner of the uniform barrier contract: the
// operation body parse refuses before any wait runs, so even a
// held barrier cannot hang the path. The holder releases only
// after the prompt verdict.
func TestQuiesceMalformedDeadlineRefusesBeforeBarrierWait(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.barrierMu.Lock()
	defer fx.lc.barrierMu.Unlock()
	raw := lxQuiesceBody(t, func(object map[string]any) {
		object["context"].(map[string]any)["deadline_at"] = "not-a-timestamp"
	})
	done := make(chan error, 1)
	go func() {
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "quiesce-input", Body: raw, Source: "active", Admitted: fullLifecycleAdmitted(),
		})
		done <- err
	}()
	select {
	case err := <-done:
		requireEngineCode(t, err, "terminal_backend_protocol_error")
		if fx.runner.callCount() != 0 {
			t.Fatalf("refused quiesce execed %d commands", fx.runner.callCount())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("malformed-deadline quiesce blocked on the held barrier")
	}
}

// withoutCapability returns the full lifecycle set minus one
// capability.
func withoutCapability(drop string) terminalbackend.Admitted {
	kept := []string(nil)
	for _, capability := range fullLifecycleAdmitted().Capabilities {
		if capability != drop {
			kept = append(kept, capability)
		}
	}
	return terminalbackend.Admitted{Capabilities: kept}
}

// TestAttachOverlapRequiresMultiAttach pins the overlap
// fail-closed gate: the first client attaches without multi_attach
// (no peer exists yet), but the second client — which may overlap
// the recorded first — refuses without it and commits no receipt.
// With the capability the second client attaches; a third client
// without it refuses too, proving the gate counts past the first
// peer.
func TestAttachOverlapRequiresMultiAttach(t *testing.T) {
	admitted := withoutCapability("multi_attach")
	fx := newLifecycleFixture(t, admitted)
	first := lxAttachBody(t, func(object map[string]any) {
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: first, Source: "parked", Admitted: admitted,
	}); err != nil {
		t.Fatalf("first attach refused without peers: %v", err)
	}
	second := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: admitted,
	})
	requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no overlapping receipt", found, err)
	}
	if outcome.Attach != nil || len(outcome.Argv) != 0 {
		t.Fatalf("refused attach returned a vector: attach=%+v argv=%q", outcome.Attach, outcome.Argv)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatalf("second attach refused with multi_attach: %v", err)
	}
	third := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = "0198f4c8-8e50-7f66-8f70-555555555553"
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: third, Source: "active", Admitted: admitted,
	}); err == nil {
		t.Fatal("third attach admitted past two peers without multi_attach")
	} else {
		requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	}
}

// TestAttachOverlapInputRequiresMultipleInputClients pins the
// concurrent-input arm: over a recorded input-authorized peer, a
// new input-authorized attach refuses without
// multiple_input_clients (even with multi_attach) and commits no
// receipt or authorized input. A read-only second client stays
// admitted — observation is not input — and the full set admits
// the writable pair.
func TestAttachOverlapInputRequiresMultipleInputClients(t *testing.T) {
	admitted := withoutCapability("multiple_input_clients")
	fx := newLifecycleFixture(t, admitted)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted,
	}); err != nil {
		t.Fatalf("first writable attach refused: %v", err)
	}
	second := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: admitted,
	})
	requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no concurrent-input receipt", found, err)
	}
	if outcome.Attach != nil || len(outcome.Argv) != 0 {
		t.Fatalf("refused concurrent-input attach returned a vector: attach=%+v argv=%q", outcome.Attach, outcome.Argv)
	}
	readonly := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: readonly, Source: "active", Admitted: admitted,
	}); err != nil {
		t.Fatalf("read-only second attach refused over an input peer: %v", err)
	}
	// Precision: over two recorded peers the input arm still
	// refuses without the capability.
	third := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = "0198f4c8-8e50-7f66-8f70-555555555553"
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: third, Source: "active", Admitted: admitted,
	}); err == nil {
		t.Fatal("third writable attach admitted past peers without multiple_input_clients")
	} else {
		requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	}
	fx2 := newLifecycleFixture(t, fullLifecycleAdmitted())
	if _, err := fx2.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	granted, err := fx2.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("writable pair refused with multiple_input_clients: %v", err)
	}
	if granted.Attach == nil || !granted.Attach.InputAuthorized {
		t.Fatalf("attach = %+v, want the authorized pair", granted.Attach)
	}
}

// TestAttachSameClientRetryIgnoresOverlap pins the idempotent
// replay: the same client re-attaching over its own recorded
// receipt is not a new client, so the overlap capabilities are
// not consulted and the retry succeeds on the transport
// capability alone.
func TestAttachSameClientRetryIgnoresOverlap(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	reduced := admittedWith("local_attach")
	replayed, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "active", Admitted: reduced,
	})
	if err != nil {
		t.Fatalf("same-client retry refused without overlap capabilities: %v", err)
	}
	if replayed.Attach == nil || replayed.Attach.ClientMirrorID != lxClient || !replayed.Attach.InputAuthorized {
		t.Fatalf("attach = %+v, want the replayed writable receipt", replayed.Attach)
	}
}

// TestAttachOverlapPeerReadFailureFailsClosed pins the census
// health rule: a peer receipt that cannot be read is unknown, not
// absence, so the overlapping attach errors instead of admitting
// past the unreadable peer.
func TestAttachOverlapPeerReadFailureFailsClosed(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	peer := filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance, lxClient+".json")
	if err := os.WriteFile(peer, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	second := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("attach admitted past an unreadable peer: %+v", outcome.Attach)
	}
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt past the unreadable peer", found, err)
	}
}
