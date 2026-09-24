package tmuxserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// effectBackend builds a request backend for PerformEffect unit tests.
func effectBackend(t *testing.T, operation terminalbackend.Operation, runner *fakeRunner) *backend {
	t.Helper()
	bindings, err := testAxpaneStore(t)
	if err != nil {
		t.Fatal(err)
	}
	attach, err := termbind.OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	states, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := newManualClock(lxNow())
	// The helper pins the command ladders, not the entry
	// revalidation: the post-wait boundaries carry a permissive
	// recheck here, and the production entries configure the
	// real one (pinned through Lifecycle.Execute).
	allow := func() error { return nil }
	return &backend{
		runner:      runner,
		runtimeDir:  "/root/tmux",
		socket:      "/root/tmux/ax.sock",
		operation:   operation,
		sessionID:   lxSession,
		instanceID:  lxInstance,
		bootstrapID: lxBootstrap,
		bindingID:   lxDigestA,
		recheck:     allow,
		now:         clock.Now,
		sleep:       clock.Sleep,
		poll:        time.Millisecond,
		deadline:    clock.Now().Add(time.Minute),
		admitted:    fullLifecycleAdmitted(),
		bindings:    bindings,
		attach:      attach,
		states:      states,
		killExit:    -1,
	}
}

func requireDigest(t *testing.T, id string) {
	t.Helper()
	if _, err := scalar.ParseDigest(id); err != nil {
		t.Fatalf("evidence %q: %v", id, err)
	}
}

func TestPerformEffectBindingPersisted(t *testing.T) {
	reqBackend := effectBackend(t, terminalbackend.OperationCreate, newFakeRunner())
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectBindingPersisted)
	if err != nil {
		t.Fatal(err)
	}
	if id != lxDigestA {
		t.Fatalf("evidence = %s", id)
	}
	// Identical retry replays the one receipt.
	id2, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectBindingPersisted)
	if err != nil || id2 != id {
		t.Fatalf("replay = %s, %v", id2, err)
	}
}

func TestPerformEffectWrapperStart(t *testing.T) {
	runner := newFakeRunner().queue("new-session", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationCreate, runner)
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectWrapperStarted)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
	calls := runner.subcommands()
	if len(calls) != 1 || calls[0] != "new-session" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestPerformEffectRefusesFailedExec(t *testing.T) {
	runner := newFakeRunner().queue("new-session", "", 1)
	reqBackend := effectBackend(t, terminalbackend.OperationCreate, runner)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectWrapperStarted); err == nil {
		t.Fatal("failed exec committed")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
}

func TestPerformEffectTransportProvesNothing(t *testing.T) {
	runner := newFakeRunner().failTransport("new-session", errFakeTransport)
	reqBackend := effectBackend(t, terminalbackend.OperationCreate, runner)
	_, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectWrapperStarted)
	if err != errFakeTransport {
		t.Fatalf("err = %v, want the transport failure", err)
	}
}

func TestPerformEffectInputClosed(t *testing.T) {
	runner := newFakeRunner().queue("lock-session", "", 0).queue("detach-client", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationQuiesceInput, runner)
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectInputClosed)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
	got := runner.subcommands()
	if len(got) != 2 || got[0] != "lock-session" || got[1] != "detach-client" {
		t.Fatalf("calls = %v, want lock then detach", got)
	}
}

func TestPerformEffectBoundaryRefusedExit(t *testing.T) {
	runner := newFakeRunner().queue("wait-for", "", 1)
	reqBackend := effectBackend(t, terminalbackend.OperationWaitSafeBoundary, runner)
	reqBackend.quiescence = lxQuiesce
	reqBackend.channel = BoundaryChannelPrefix + lxQuiesce
	reqBackend.waitCtx = context.Background()
	_, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectSafeBoundaryObserved)
	if err == nil {
		t.Fatal("failed boundary wait committed")
	}
}

func TestPerformEffectBoundaryObserved(t *testing.T) {
	runner := newFakeRunner().queue("wait-for", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationWaitSafeBoundary, runner)
	reqBackend.quiescence = lxQuiesce
	reqBackend.channel = BoundaryChannelPrefix + lxQuiesce
	reqBackend.waitCtx = context.Background()
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectSafeBoundaryObserved)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
}

func TestPerformEffectBoundaryTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := newFakeRunner().failTransport("wait-for", context.Canceled)
	reqBackend := effectBackend(t, terminalbackend.OperationWaitSafeBoundary, runner)
	reqBackend.quiescence = lxQuiesce
	reqBackend.channel = BoundaryChannelPrefix + lxQuiesce
	reqBackend.waitCtx = ctx
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectSafeBoundaryObserved); err == nil {
		t.Fatal("expired wait committed")
	} else {
		requireEngineCode(t, err, "quiesce_timeout")
	}
}

func TestPerformEffectStopLadder(t *testing.T) {
	// Graceful close on first confirm: send-keys, has (absent).
	runner := newFakeRunner().queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	reqBackend := effectBackend(t, terminalbackend.OperationRequestStop, runner)
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
	if got := runner.subcommands(); len(got) != 1 || got[0] != "has-session" {
		t.Fatalf("calls = %v", got)
	}
}

func TestPerformEffectStopEscalates(t *testing.T) {
	// Live, live, then absent after the kill escalation.
	runner := newFakeRunner().
		queue("has-session", "", 0).
		queue("kill-session", "", 0).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	reqBackend := effectBackend(t, terminalbackend.OperationRequestStop, runner)
	reqBackend.deadline = lxNow().Add(-time.Minute)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed); err != nil {
		t.Fatal(err)
	}
	got := runner.subcommands()
	want := []string{"has-session", "kill-session", "has-session"}
	if len(got) != 3 {
		t.Fatalf("calls = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("calls = %v, want %v", got, want)
		}
	}
}

func TestPerformEffectStopTimeout(t *testing.T) {
	runner := newFakeRunner().
		queue("has-session", "", 0).
		queue("kill-session", "", 0).
		queue("has-session", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationRequestStop, runner)
	reqBackend.deadline = lxNow().Add(-time.Minute)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed); err == nil {
		t.Fatal("stubborn session closed")
	} else {
		requireEngineCode(t, err, "stop_timeout")
	}
}

func TestPerformEffectStopEscalationKillFailure(t *testing.T) {
	// Live session plus a failed escalation kill: the coded process
	// failure, before any re-confirm — the kill did not prove the
	// incarnation dead.
	runner := newFakeRunner().
		queue("has-session", "", 0).
		queue("kill-session", "", 1).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	reqBackend := effectBackend(t, terminalbackend.OperationRequestStop, runner)
	reqBackend.deadline = lxNow().Add(-time.Minute)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed); err == nil {
		t.Fatal("failed escalation kill closed")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
	if got := runner.subcommands(); len(got) != 2 || got[1] != "kill-session" {
		t.Fatalf("calls = %v, want has-session then kill-session", got)
	}
}

func TestPerformEffectBindingConflict(t *testing.T) {
	// A changed binding for the recorded pair is the landed mismatch,
	// mapped with the persist detail — never a silent re-record.
	reqBackend := effectBackend(t, terminalbackend.OperationCreate, newFakeRunner())
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectBindingPersisted); err != nil {
		t.Fatal(err)
	}
	reqBackend.bootstrapID = lxQuiesce
	_, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectBindingPersisted)
	if err == nil {
		t.Fatal("conflicting binding persisted")
	}
	var engine *terminstance.Error
	if !errors.As(err, &engine) {
		t.Fatalf("want *terminstance.Error, got %T (%v)", err, err)
	}
	if engine.Code != "idempotency_mismatch" || engine.Detail != "backend binding conflict" {
		t.Fatalf("want mismatch at backend binding conflict, got %s at %s", engine.Code, engine.Detail)
	}
}

func TestPerformEffectTerminateKill(t *testing.T) {
	runner := newFakeRunner().queue("kill-session", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationTerminateStale, runner)
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectStaleIncarnationTerminated)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
	// The kill claimed the incarnation; the re-confirm proves absence.
	runner.queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed); err != nil {
		t.Fatal(err)
	}
}

func TestPerformEffectTerminateSelfConfirms(t *testing.T) {
	// Failed kill plus absent session: terminated however it closed.
	runner := newFakeRunner().
		queue("kill-session", "", 1).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	reqBackend := effectBackend(t, terminalbackend.OperationTerminateStale, runner)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectStaleIncarnationTerminated); err != nil {
		t.Fatal(err)
	}
}

func TestPerformEffectTerminateLiveRefuses(t *testing.T) {
	// Failed kill plus live session: the process failure, both at the
	// self-confirm and (after a clean kill) at the re-confirm.
	runner := newFakeRunner().
		queue("kill-session", "", 1).
		queue("has-session", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationTerminateStale, runner)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectStaleIncarnationTerminated); err == nil {
		t.Fatal("live session terminated")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
	runner2 := newFakeRunner().
		queue("kill-session", "", 0).
		queue("has-session", "", 0)
	reqBackend2 := effectBackend(t, terminalbackend.OperationTerminateStale, runner2)
	reqBackend2.deadline = lxNow().Add(-time.Minute)
	if _, err := reqBackend2.PerformEffect(context.Background(), terminalbackend.EffectStaleIncarnationTerminated); err != nil {
		t.Fatal(err)
	}
	if _, err := reqBackend2.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed); err == nil {
		t.Fatal("resurrected session closed")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
}

func TestPerformEffectTerminateReconfirmRefusesFailedKill(t *testing.T) {
	// Failed kill plus live session at the re-confirm: the recorded
	// failed-kill member refuses exactly like the clean-kill member —
	// the kill claimed the incarnation and it still answers.
	runner := newFakeRunner().
		queue("kill-session", "", 1).
		queue("has-session", "", 0).
		queue("has-session", "", 0)
	reqBackend := effectBackend(t, terminalbackend.OperationTerminateStale, runner)
	reqBackend.deadline = lxNow().Add(-time.Minute)
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectStaleIncarnationTerminated); err == nil {
		t.Fatal("live session terminated at self-confirm")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectProcessClosed); err == nil {
		t.Fatal("live session closed after failed kill")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
}

func TestPerformEffectNilStoresRefuse(t *testing.T) {
	// The durable effects fail closed without their stores: a missing
	// binding store or attach store is a local wiring refusal, never a
	// silent no-op.
	persist := effectBackend(t, terminalbackend.OperationCreate, newFakeRunner())
	persist.bindings = nil
	if _, err := persist.PerformEffect(context.Background(), terminalbackend.EffectBindingPersisted); err == nil {
		t.Fatal("persist without store committed")
	}
	client := effectBackend(t, terminalbackend.OperationAttach, newFakeRunner())
	client.attach = nil
	if _, err := client.PerformEffect(context.Background(), terminalbackend.EffectAttachClientCreated); err == nil {
		t.Fatal("attach without store committed")
	}
}

func TestPerformEffectStoreClosedIsLocal(t *testing.T) {
	runner := newFakeRunner()
	reqBackend := effectBackend(t, terminalbackend.OperationRequestStop, runner)
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectBackendStoreClosed)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
	if runner.callCount() != 0 {
		t.Fatalf("local effect execed %d commands", runner.callCount())
	}
}

func TestPerformEffectRefusesUnknownVocabulary(t *testing.T) {
	reqBackend := effectBackend(t, terminalbackend.OperationCreate, newFakeRunner())
	if _, err := reqBackend.PerformEffect(context.Background(), "server_rebooted"); err == nil {
		t.Fatal("unknown effect committed")
	} else {
		requireEngineCode(t, err, "terminal_backend_protocol_error")
	}
}

func TestPerformEffectAttachClient(t *testing.T) {
	reqBackend := effectBackend(t, terminalbackend.OperationAttach, newFakeRunner())
	reqBackend.clientID = lxClient
	reqBackend.transport = "local_only"
	reqBackend.input = true
	auth, err := parseAttachBody(lxAttachBody(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	reqBackend.authRaw = auth.AuthRaw
	id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectAttachClientCreated)
	if err != nil {
		t.Fatal(err)
	}
	requireDigest(t, id)
	// Identical retry replays; conflicting retry mismatches.
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectAttachClientCreated); err != nil {
		t.Fatalf("replay: %v", err)
	}
	// A conflicting retry carries matching authorization for the new
	// input boolean (the store rechecks auth before the conflict arm).
	falseAuth, err := parseAttachBody(lxAttachBody(t, func(object map[string]any) {
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	}))
	if err != nil {
		t.Fatal(err)
	}
	reqBackend.input = false
	reqBackend.authRaw = falseAuth.AuthRaw
	if _, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectAttachClientCreated); err == nil {
		t.Fatal("conflicting attach committed")
	} else {
		requireEngineCode(t, err, "idempotency_mismatch")
	}
}
