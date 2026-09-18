package termbind

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

func bindPair(t *testing.T, store *axpane.Store, sessionID, operationID, instanceID string) axpane.Binding {
	t.Helper()
	bound, _, err := store.Bind(sessionID, operationID, axpane.Binding{
		SessionID:          sessionID,
		OperationID:        operationID,
		TerminalInstanceID: instanceID,
		BindingDigest:      seedDigest(0xB1),
	})
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	return bound
}

func recoveryRequest(operationID string, windowOpen bool) RecoveryRequest {
	return RecoveryRequest{
		SessionID:             fixtureSession,
		BootstrapOperationID:  operationID,
		WindowOpen:            windowOpen,
		BackendID:             terminalbackend.BuiltinTmux,
		ImplementationVersion: fixtureImpl,
		ProtocolVersion:       fixtureProto,
		Generation:            fixtureRawGen,
		DeadlineAt:            fixtureDeadline,
	}
}

func exactObservation(state terminalbackend.InstanceState, match bool) terminstance.StatusObservation {
	observed := terminstance.StatusObservation{
		State:         state,
		IdentityMatch: match,
		SessionID:     fixtureSession,
		InstanceID:    fixtureInstance,
		BackendID:     terminalbackend.BuiltinTmux,
		ImplVersion:   fixtureImpl,
		ProtoVersion:  fixtureProto,
		Generation:    fixtureRawGen,
	}
	if match {
		observed.WrapperPresent = state == terminalbackend.StateParked || state == terminalbackend.StateActive || state == terminalbackend.StateQuiescing
	} else {
		observed.InstanceID = "0198f4c9-9999-7eee-8eee-1234567890ab"
	}
	return observed
}

func sessionObservation(state terminalbackend.InstanceState, sessionID string) terminstance.StatusObservation {
	return terminstance.StatusObservation{
		State:         state,
		IdentityMatch: sessionID == fixtureSession,
		SessionID:     sessionID,
		InstanceID:    fixtureInstance,
		BackendID:     terminalbackend.BuiltinTmux,
		ImplVersion:   fixtureImpl,
		ProtoVersion:  fixtureProto,
		Generation:    fixtureRawGen,
	}
}

func recoverWith(t *testing.T, store *axpane.Store, observe func(terminstance.StatusBody) (terminstance.StatusObservation, error), request RecoveryRequest) (RecoveryVerdict, error) {
	t.Helper()
	deps := RecoveryDeps{Bindings: store, Engine: testEngine(&mockBackend{observe: observe})}
	return RecoverCreate(context.Background(), deps, request)
}

func TestRecoverCreateProvesAbsence(t *testing.T) {
	t.Parallel()
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		return sessionObservation(terminalbackend.StateAbsent, fixtureSession), nil
	}, recoveryRequest(fixtureBootstrap, true))
	if err != nil {
		t.Fatalf("RecoverCreate() error = %v", err)
	}
	if verdict.Outcome != RecoveryAbsent || verdict.Retry != terminstance.DispositionReplaySame {
		t.Fatalf("verdict = %+v, want absent with replay_same", verdict)
	}
	if verdict.HasBinding {
		t.Fatalf("verdict carries a binding for proven absence: %+v", verdict)
	}
}

func TestRecoverCreateIdentifiesOneChild(t *testing.T) {
	t.Parallel()
	for _, state := range []terminalbackend.InstanceState{terminalbackend.StateActive, terminalbackend.StateParked} {
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()
			store, err := axpane.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			bound := bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
			verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
				return exactObservation(state, true), nil
			}, recoveryRequest(fixtureBootstrap, true))
			if err != nil {
				t.Fatalf("RecoverCreate() error = %v", err)
			}
			if verdict.Outcome != RecoveryChild || !verdict.HasBinding {
				t.Fatalf("verdict = %+v, want the one child", verdict)
			}
			if verdict.Binding != bound {
				t.Fatalf("verdict binding = %+v, want %+v", verdict.Binding, bound)
			}
			if verdict.State != state || verdict.Retry != terminstance.DispositionReplaySame {
				t.Fatalf("verdict = %+v, want state %q with replay_same", verdict, string(state))
			}
		})
	}
}

func TestRecoverCreateDuplicateRetrySameVerdict(t *testing.T) {
	t.Parallel()
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
	observe := func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		return exactObservation(terminalbackend.StateActive, true), nil
	}
	first, err := recoverWith(t, store, observe, recoveryRequest(fixtureBootstrap, true))
	if err != nil {
		t.Fatal(err)
	}
	second, err := recoverWith(t, store, observe, recoveryRequest(fixtureBootstrap, true))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("retry verdict = %+v, want %+v", second, first)
	}
}

func TestRecoverCreateChangedOperationInWindowRefuses(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	hasNewest, _ := foldNewest(t, repository)
	if hasNewest {
		t.Fatal("bootstrap chain folds a newest checkpoint, want the open window")
	}
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
	_, err = recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		t.Fatal("status runs past the window mismatch")
		return terminstance.StatusObservation{}, nil
	}, recoveryRequest(fixtureOtherOp, !hasNewest))
	requireCode(t, err, "idempotency_mismatch")
}

func checkpointChain(t *testing.T) *sessrepo.Repository {
	t.Helper()
	repository, _ := chainFixture(t)
	tail := chainTail(t, repository)
	next := int(tail.LeaseSequence) + 1
	predecessor := tail.EventID
	steps := []struct {
		eventType string
		payload   map[string]any
	}{
		{"terminal.created", map[string]any{"backend": "tmux", "terminal_id": "tmux-session-1"}},
		{"provider.launched", map[string]any{
			"provider_id": "codex", "provider_version": "0.147.0",
			"execution_profile": "standard", "profile_source_event_id": nil,
			"profile_mapping": "default",
		}},
		{"provider.identified", map[string]any{"provider_identity_record_id": seedDigest(0xD1), "confidence": "exact"}},
		{"session.idle", map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}},
		{"session.quiescing", map[string]any{"operation_id": fixtureBootstrap, "reason": "checkpoint", "input_blocked": true}},
		{"checkpoint.created", map[string]any{"checkpoint_id": seedDigest(0xC1), "kind": "manual"}},
	}
	for _, step := range steps {
		predecessor = appendEvent(t, repository, "1.0.0", step.eventType, 1, fixtureLeaseA, next, predecessor, step.payload)
		next++
	}
	return repository
}

func TestRecoverCreateChangedOperationPostWindowProceeds(t *testing.T) {
	t.Parallel()
	repository := checkpointChain(t)
	hasNewest, _ := foldNewest(t, repository)
	if !hasNewest {
		t.Fatal("checkpoint chain folds no newest, want the closed window")
	}
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
	verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		return sessionObservation(terminalbackend.StateAbsent, fixtureSession), nil
	}, recoveryRequest(fixtureOtherOp, !hasNewest))
	if err != nil {
		t.Fatalf("RecoverCreate(post-window) error = %v", err)
	}
	if verdict.Outcome != RecoveryAbsent {
		t.Fatalf("verdict = %+v, want absence for the new post-window pair", verdict)
	}
}

func TestRecoverCreateLostResult(t *testing.T) {
	t.Parallel()
	// The lost create result: the durable binding committed before the
	// first child side effect, the backend committed the effect, and the
	// result never reached the caller (dropped here to model the crash
	// between effect and result). Recovery must identify the ONE recorded
	// child, never a second one.
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bound := bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
	committed := exactObservation(terminalbackend.StateActive, true)
	_ = committed
	verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		return exactObservation(terminalbackend.StateActive, true), nil
	}, recoveryRequest(fixtureBootstrap, true))
	if err != nil {
		t.Fatalf("RecoverCreate() error = %v", err)
	}
	if verdict.Outcome != RecoveryChild || verdict.Binding.TerminalInstanceID != bound.TerminalInstanceID {
		t.Fatalf("verdict = %+v, want the one recorded child %+v", verdict, bound)
	}
}

func TestRecoverCreateNeverClaimsAbsent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		bound   bool
		observe func() terminstance.StatusObservation
		verdict RecoveryOutcome
	}{
		{"bound contradiction", true, func() terminstance.StatusObservation {
			return exactObservation(terminalbackend.StateAbsent, false)
		}, RecoveryUnavailable},
		{"creating interim", true, func() terminstance.StatusObservation {
			return exactObservation(terminalbackend.StateCreating, true)
		}, RecoveryUnavailable},
		{"backend unavailable", true, func() terminstance.StatusObservation {
			return exactObservation(terminalbackend.StateUnavailable, true)
		}, RecoveryUnavailable},
		{"unbound live instance", false, func() terminstance.StatusObservation {
			return sessionObservation(terminalbackend.StateActive, fixtureSession)
		}, RecoveryUnavailable},
		{"unbound session drift", false, func() terminstance.StatusObservation {
			return sessionObservation(terminalbackend.StateAbsent, "0198f4c8-8e50-7f66-8f70-1234567890ab")
		}, RecoveryUnavailable},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			store, err := axpane.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if testCase.bound {
				bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
			}
			verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
				return testCase.observe(), nil
			}, recoveryRequest(fixtureBootstrap, true))
			if err != nil {
				t.Fatalf("RecoverCreate() error = %v", err)
			}
			if verdict.Outcome != testCase.verdict || verdict.Retry != terminstance.DispositionStatusFirst {
				t.Fatalf("verdict = %+v, want unavailable with status_first", verdict)
			}
		})
	}
}

func TestRecoverCreateRecordedChildBackendAbsentReplays(t *testing.T) {
	t.Parallel()
	// The binding committed but the backend shows absent: the effect never
	// committed or the backend lost it. The binding still identifies the
	// ONE child, so the same operation replays for the same recorded
	// instance instead of minting a second one.
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bound := bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
	verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		return exactObservation(terminalbackend.StateAbsent, true), nil
	}, recoveryRequest(fixtureBootstrap, true))
	if err != nil {
		t.Fatalf("RecoverCreate() error = %v", err)
	}
	if verdict.Outcome != RecoveryChild || verdict.Binding != bound || verdict.Retry != terminstance.DispositionReplaySame {
		t.Fatalf("verdict = %+v, want the recorded child with replay_same", verdict)
	}
}

func TestRecoverGenerationBoundBeforeStatusRead(t *testing.T) {
	t.Parallel()
	// A 257-character generation on a bound recovery request refuses the
	// landed generation bound before the status read runs: the backend is
	// never contacted with an unbindable generation.
	store, err := axpane.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bindPair(t, store, fixtureSession, fixtureBootstrap, fixtureInstance)
	request := recoveryRequest(fixtureBootstrap, true)
	request.Generation = strings.Repeat("g", 257)
	reads := 0
	verdict, err := recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		reads++
		return exactObservation(terminalbackend.StateActive, true), nil
	}, request)
	if err == nil {
		t.Fatalf("RecoverCreate(generation 257) succeeded with verdict %+v after %d status reads, want the generation bound refusal before any read", verdict, reads)
	}
	requireCode(t, err, "terminal_backend_stale_generation")
	requireDetail(t, err, "backend_generation bound")
	if reads != 0 {
		t.Fatalf("status read ran %d times before the bound refused", reads)
	}
}

func TestRecoverCreateUnknownIsNotAbsent(t *testing.T) {
	t.Parallel()
	t.Run("status read failure", func(t *testing.T) {
		t.Parallel()
		store, err := axpane.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		_, err = recoverWith(t, store, func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
			return terminstance.StatusObservation{}, errors.New("backend read failed")
		}, recoveryRequest(fixtureBootstrap, true))
		if err == nil || !strings.Contains(err.Error(), "terminal_backend_unavailable") {
			t.Fatalf("RecoverCreate() error = %v, want the unknown-read refusal", err)
		}
	})
}
