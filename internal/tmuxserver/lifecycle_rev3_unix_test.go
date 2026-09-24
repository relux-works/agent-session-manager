//go:build !windows

package tmuxserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// recordedAxpaneBinding is the bootstrap pair receipt the binding
// tests stage without its document: the receipt stands, the payload
// is absent.
func recordedAxpaneBinding() axpane.Binding {
	return axpane.Binding{
		SessionID:          lxSession,
		OperationID:        lxBootstrap,
		TerminalInstanceID: lxInstance,
		BindingDigest:      lxDigestA,
	}
}

// This file pins the revision-3 rework findings (rev2 P1-A…P1-D,
// P2-A, and the five surviving narrowings R1–R5) through the
// production entries: every test drives Lifecycle.Execute (or the
// public store lookup entry where the reviewer placed the killer)
// and asserts the literal effect, never a string report.

// TestExecuteReadOnlyAttachPreservesQuiescedInput is the P1-A
// regression: quiesce, then an admitted read-only attach, then an
// input-authorized attach. Observation must not reopen input: the
// third call refuses the failed local precondition and executes
// nothing.
func TestExecuteReadOnlyAttachPreservesQuiescedInput(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach",
		Body: lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = lxClientB
			object["input_authorized"] = false
			object["authorization"] = lxAttachAuth("local_only", false)
		}),
		Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("quiesced input reopened by read-only observation: argv=%q", outcome.Argv)
	}
	requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want exactly the quiesce pair", fx.runner.callCount())
	}
	if memory, found, err := fx.lc.States.Lookup(lxInstance); err != nil || !found || memory != terminalbackend.StateQuiescing {
		t.Fatalf("memory = %s, found=%v, err=%v, want quiescing", memory, found, err)
	}
}

// corruptOutcomeDoc overwrites the persisted outcome report for key
// with unparseable bytes: the receipt stands, the report is corrupt.
func corruptOutcomeDoc(t *testing.T, fx *lxFixture, key string) {
	t.Helper()
	sum := sha256.Sum256([]byte(key))
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "outcomes", hex.EncodeToString(sum[:])+".json")
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestExecuteQuiesceCorruptOutcomeRefusesIntegrity is the P2-A
// regression: quiesce succeeds, only the persisted report is
// corrupted, the clock advances, and the identical retry refuses the
// integrity failure instead of manufacturing a fresh closure time
// without closing input again.
func TestExecuteQuiesceCorruptOutcomeRefusesIntegrity(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	req := OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}
	first, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	corruptOutcomeDoc(t, fx, lxQuiesceKey())
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("corrupt replay manufactured closure time %s -> %s", first.InputClosedAt, second.InputClosedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "quiesce outcome image")
	if second.InputClosedAt != "" {
		t.Fatalf("refused report manufactures time: %+v", second)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want exactly the first close", fx.runner.callCount())
	}
}

// TestExecuteBoundaryCorruptOutcomeRefusesIntegrity is the P2-A twin
// for the boundary report: a corrupt boundary outcome refuses the
// integrity failure instead of manufacturing a fresh observed time.
func TestExecuteBoundaryCorruptOutcomeRefusesIntegrity(t *testing.T) {
	const kind = "ax_checkpoint_boundary"
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	req := OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, kind, 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	corruptOutcomeDoc(t, fx, lxBoundaryKey(kind))
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("corrupt replay manufactured boundary time %s", second.BoundaryObservedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "boundary outcome image")
	if second.BoundaryObservedAt != "" {
		t.Fatalf("refused report manufactures time: %+v", second)
	}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want exactly the first wait", fx.runner.callCount())
	}
}

// TestExecuteQuiesceForgedOutcomeKeyRefusesIntegrity pins the report
// mapping for a forged outcome key: the lookup gate refuses the key,
// and the quiesce retry maps that refusal to the integrity failure
// instead of recording fresh evidence beside the forgery.
func TestExecuteQuiesceForgedOutcomeKeyRefusesIntegrity(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	req := OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(lxQuiesceKey()))
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "outcomes", hex.EncodeToString(sum[:])+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document["idempotency_key"] = "review-forged-key"
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("forged replay reported time %s", second.InputClosedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "quiesce outcome image")
}

// TestInstanceStatesLookupOutcomeRefusesForgedKey is the R2 killer at
// the public store lookup entry: a report filed under one key hash
// that names another key refuses the key gate instead of replaying
// another key's evidence.
func TestInstanceStatesLookupOutcomeRefusesForgedKey(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	key := "review-original-key"
	if err := fx.lc.States.RecordOutcome(key, OperationOutcome{Operation: "quiesce-input", InputClosedAt: lxIssued}); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(key))
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "outcomes", hex.EncodeToString(sum[:])+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document["idempotency_key"] = "review-forged-key"
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fx.lc.States.LookupOutcome(key); err == nil {
		t.Fatal("forged lookup key admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance outcome key")
	}
}

// TestExecuteStatusNonMatchEachMember is the P1-B core: an
// exact-instance query drifted on exactly one recorded member reads
// the prescribed non-match result (absent, no wrapper, no provider,
// no attach, null lasts) and probes nothing. The recorded tuple
// decides; the query never echoes into the verdict.
func TestExecuteStatusNonMatchEachMember(t *testing.T) {
	const otherInstance = "0198f4c8-8e50-7f66-8f70-222222222222"
	cases := []struct {
		name   string
		member string
		value  string
	}{
		{"instance", "terminal_instance_id", otherInstance},
		{"implementation", "implementation_version", "9.9.9"},
		{"generation", "backend_generation", "generation-two"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			recordBinding(t, fx, lxDigestA)
			fx.runner.queue("list-panes", "ax\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "status",
				Body: lxStatusBody(t, true, false, func(object map[string]any) {
					object[tc.member] = tc.value
				}),
				Source: "parked", Admitted: fullLifecycleAdmitted(),
			})
			if err != nil {
				t.Fatalf("non-match refused: %v", err)
			}
			report := outcome.Status
			if report == nil {
				t.Fatal("status omits its report")
			}
			if report.IdentityMatch {
				t.Fatalf("query %s echoed as matching identity: %+v", tc.member, report)
			}
			if report.State != terminalbackend.StateAbsent || report.WrapperPresent ||
				report.ProviderPresent != nil || report.Attachable ||
				report.LastOperationID != nil || report.LastEffect != nil {
				t.Fatalf("non-match shape = %+v", report)
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("non-match probed %v", fx.runner.subcommands())
			}
		})
	}
}

// TestExecuteStatusExactUnboundSessionRefusesUnknown pins the session
// axis: an exact-instance query over a session with no recorded
// binding names no observable identity, so the observation is
// unknown — never a manufactured match — and probes nothing.
func TestExecuteStatusExactUnboundSessionRefusesUnknown(t *testing.T) {
	// Two unbound sessions: the narrowing row admits exactly one of
	// them, and the other still refuses (narrowness in the log).
	for _, session := range []string{lxSessionB, "0198f4c8-8e50-7f66-8f70-111111111113"} {
		t.Run(session, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			recordBinding(t, fx, lxDigestA)
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "status",
				Body: lxStatusBody(t, true, false, func(object map[string]any) {
					object["session_id"] = session
				}),
				Source: "parked", Admitted: fullLifecycleAdmitted(),
			})
			if err == nil {
				t.Fatalf("unbound exact query matched: %+v", outcome.Status)
			}
			requireEngineCode(t, err, "terminal_backend_unavailable")
			if fx.runner.callCount() != 0 {
				t.Fatalf("unknown observation probed %v", fx.runner.subcommands())
			}
		})
	}
}

// corruptBindingDoc overwrites the binding document for the fixture
// pair with unparseable bytes: the receipt stands, the payload is
// corrupt.
func corruptBindingDoc(t *testing.T, fx *lxFixture) {
	t.Helper()
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "bindings", lxSession, lxBootstrap+".json")
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestExecuteStatusBindingTamperRefusesIntegrity pins the recorded
// tuple as the observation: a receipt without its payload, a corrupt
// payload, a payload naming another session or instance, and a
// payload whose digest mismatches its bytes each refuse the
// integrity failure before any probe runs.
func TestExecuteStatusBindingTamperRefusesIntegrity(t *testing.T) {
	setup := map[string]func(t *testing.T, fx *lxFixture){
		"missing-doc": func(t *testing.T, fx *lxFixture) {
			if _, _, err := fx.lc.Bindings.Bind(lxSession, lxBootstrap, recordedAxpaneBinding()); err != nil {
				t.Fatal(err)
			}
		},
		"corrupt-doc": func(t *testing.T, fx *lxFixture) {
			recordBinding(t, fx, lxDigestA)
			corruptBindingDoc(t, fx)
		},
		"foreign-session-doc": func(t *testing.T, fx *lxFixture) {
			recordBinding(t, fx, lxDigestA)
			swapped := lxBindingDoc(t, func(object map[string]any) {
				object["session_id"] = lxSessionB
			})
			if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, swapped); err != nil {
				t.Fatal(err)
			}
		},
		"foreign-instance-doc": func(t *testing.T, fx *lxFixture) {
			recordBinding(t, fx, lxDigestA)
			swapped := lxBindingDoc(t, func(object map[string]any) {
				object["terminal_instance_id"] = "0198f4c8-8e50-7f66-8f70-222222222222"
			})
			if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, swapped); err != nil {
				t.Fatal(err)
			}
		},
		"digest-mismatch-doc": func(t *testing.T, fx *lxFixture) {
			recordBinding(t, fx, lxDigestA)
			swapped := lxBindingDoc(t, func(object map[string]any) {
				object["binding_id"] = lxDigestB
			})
			if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, swapped); err != nil {
				t.Fatal(err)
			}
		},
	}
	names := []string{"missing-doc", "corrupt-doc", "foreign-session-doc", "foreign-instance-doc", "digest-mismatch-doc"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			setup[name](t, fx)
			fx.runner.queue("list-panes", "ax\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
			_, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "status", Body: lxStatusBody(t, true, false, nil),
				Source: "parked", Admitted: fullLifecycleAdmitted(),
			})
			if err == nil {
				t.Fatalf("%s admitted", name)
			}
			requireEngineCode(t, err, "terminal_backend_integrity_failure")
			if fx.runner.callCount() != 0 {
				t.Fatalf("%s probed %v", name, fx.runner.subcommands())
			}
		})
	}
}

// TestObserveStatusSessionScopedObservesRecordedTuple pins the
// session-scoped vehicle: the reported tuple is the recorded one even
// where the engine adopts — a query carrying drifted versions still
// observes the recorded implementation and generation.
func TestObserveStatusSessionScopedObservesRecordedTuple(t *testing.T) {
	runner := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	reqBackend, _ := statusBackend(t, runner, admittedWith("local_attach"))
	stageStatusBinding(t, reqBackend)
	body, err := terminstance.ParseStatusBody(lxStatusBody(t, false, false, func(object map[string]any) {
		object["implementation_version"] = "9.9.9"
	}))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := reqBackend.ObserveStatus(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	if !observed.IdentityMatch {
		t.Fatalf("session-scoped bound query did not adopt: %+v", observed)
	}
	if observed.ImplVersion != lxImpl {
		t.Fatalf("implementation = %s, want recorded %s", observed.ImplVersion, lxImpl)
	}
	if observed.Generation != lxGeneration {
		t.Fatalf("generation = %s, want recorded %s", observed.Generation, lxGeneration)
	}
	if observed.InstanceID != lxInstance || observed.SessionID != lxSession {
		t.Fatalf("observed = %+v", observed)
	}
}

// TestObserveStatusExactBackendDriftNonMatches pins the backend member
// at the backend entry: the lifecycle backend gate refuses foreign
// backends before observation, so the recorded-vs-query backend
// comparison is reachable only here, where drift reads non-match.
func TestObserveStatusExactBackendDriftNonMatches(t *testing.T) {
	runner := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	reqBackend, _ := statusBackend(t, runner, admittedWith("local_attach"))
	stageStatusBinding(t, reqBackend)
	body, err := terminstance.ParseStatusBody(lxStatusBody(t, true, false, nil))
	if err != nil {
		t.Fatal(err)
	}
	body.TerminalBackendID = "ax.conpty"
	observed, err := reqBackend.ObserveStatus(context.Background(), body)
	if err != nil {
		t.Fatalf("non-match refused: %v", err)
	}
	if observed.IdentityMatch || observed.State != terminalbackend.StateAbsent {
		t.Fatalf("backend drift matched: %+v", observed)
	}
}

// TestExecuteRestoreAcrossServerGeneration is the P1-C core: create,
// reboot the receipt scope, advance the server generation, and
// restore. The wrapper recreates once, the result carries the minted
// successor (new generation, supersedes link, fresh termbind-admitted
// identity), and the identical retry replays that same successor
// without executing again.
func TestExecuteRestoreAcrossServerGeneration(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	created, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	prior := created.Binding.BindingID
	rebooted, err := terminstance.OpenReceiptStore(filepath.Join(fx.data, "receipts-after-reboot"))
	if err != nil {
		t.Fatal(err)
	}
	fx.lc.Receipts = rebooted
	fx.lc.CurrentGeneration = func() string { return "generation-after-reboot" }
	fx.runner.queue("new-session", "", 0)
	restoreReq := OpRequest{
		Operation: "restore",
		Body: lxRestoreBody(t, prior, func(object map[string]any) {
			object["context"].(map[string]any)["backend_generation"] = "generation-after-reboot"
		}),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	}
	restored, err := fx.lc.Execute(context.Background(), restoreReq)
	if err != nil {
		t.Fatalf("valid restore to new generation failed after effects %v: %v", fx.runner.subcommands(), err)
	}
	successor := restored.Binding
	if successor == nil || successor.BackendGeneration != "generation-after-reboot" {
		t.Fatalf("restore lacks new-generation binding: %+v", successor)
	}
	if successor.BindingID == prior {
		t.Fatalf("successor reuses the prior identity: %+v", successor)
	}
	if !successor.HasSupersedes || successor.SupersedesBindingID != prior {
		t.Fatalf("successor omits the supersedes link: %+v", successor)
	}
	if successor.SessionID != lxSession || successor.TerminalInstanceID != lxInstance ||
		successor.TerminalBackendID != "ax.tmux" {
		t.Fatalf("successor = %+v", successor)
	}
	stored, found, err := fx.lc.States.LookupBindingDoc(lxSession, lxBootstrap)
	if err != nil || !found {
		t.Fatalf("successor not persisted: found=%v err=%v", found, err)
	}
	reparsed, err := termbind.ParseTerminalBinding(stored)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.BindingID != successor.BindingID {
		t.Fatalf("persisted identity %s != returned %s", reparsed.BindingID, successor.BindingID)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want create plus one restore", fx.runner.callCount())
	}
	replayed, err := fx.lc.Execute(context.Background(), restoreReq)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Binding == nil || replayed.Binding.BindingID != successor.BindingID {
		t.Fatalf("replay binding = %+v, want %s", replayed.Binding, successor.BindingID)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("replay executed: calls = %d", fx.runner.callCount())
	}
}

// TestExecuteRestoreSuccessorRetryConverges pins crash convergence
// across the succession: with the successor persist lost (the stored
// document still names the prior generation), the identical retry
// replays the receipt without executing and converges on the
// identical successor instead of forking a second identity.
func TestExecuteRestoreSuccessorRetryConverges(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	created, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	priorDoc, found, err := fx.lc.States.LookupBindingDoc(lxSession, lxBootstrap)
	if err != nil || !found {
		t.Fatalf("prior doc missing: found=%v err=%v", found, err)
	}
	prior := created.Binding.BindingID
	rebooted, err := terminstance.OpenReceiptStore(filepath.Join(fx.data, "receipts-after-reboot"))
	if err != nil {
		t.Fatal(err)
	}
	fx.lc.Receipts = rebooted
	fx.lc.CurrentGeneration = func() string { return "generation-after-reboot" }
	fx.runner.queue("new-session", "", 0)
	restoreReq := OpRequest{
		Operation: "restore",
		Body: lxRestoreBody(t, prior, func(object map[string]any) {
			object["context"].(map[string]any)["backend_generation"] = "generation-after-reboot"
		}),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	}
	first, err := fx.lc.Execute(context.Background(), restoreReq)
	if err != nil {
		t.Fatal(err)
	}
	// Lose the successor persist: the stored document falls back to
	// the prior generation while the receipt stands completed.
	if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, priorDoc); err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), restoreReq)
	if err != nil {
		t.Fatal(err)
	}
	if second.Binding == nil || second.Binding.BindingID != first.Binding.BindingID {
		t.Fatalf("retry forked: %v vs %v", second.Binding, first.Binding)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("retry executed: calls = %d", fx.runner.callCount())
	}
	stored, found, err := fx.lc.States.LookupBindingDoc(lxSession, lxBootstrap)
	if err != nil || !found {
		t.Fatalf("successor not re-persisted: found=%v err=%v", found, err)
	}
	reparsed, err := termbind.ParseTerminalBinding(stored)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.BindingID != first.Binding.BindingID {
		t.Fatalf("re-persisted identity %s != %s", reparsed.BindingID, first.Binding.BindingID)
	}
}

// TestExecuteRestorePriorValidationPrecedesEffects pins the P1-C
// ordering: a corrupt or missing prior refuses before the receipt
// binds and the wrapper recreates — the queued wrapper vector stays
// unconsumed.
func TestExecuteRestorePriorValidationPrecedesEffects(t *testing.T) {
	t.Run("corrupt-prior", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		corruptBindingDoc(t, fx)
		fx.runner.queue("new-session", "", 0)
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil),
			Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		if err == nil {
			t.Fatalf("corrupt prior admitted: %+v", outcome.Binding)
		}
		requireLocalCode(t, err, "terminal_backend_integrity_failure", "restore binding image")
		if outcome.Mutation == nil {
			t.Fatal("refused restore omits its mutation")
		}
		if fx.runner.callCount() != 0 {
			t.Fatalf("pre-effects refusal executed %v", fx.runner.subcommands())
		}
	})
	t.Run("missing-prior-doc", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		if _, _, err := fx.lc.Bindings.Bind(lxSession, lxBootstrap, recordedAxpaneBinding()); err != nil {
			t.Fatal(err)
		}
		fx.runner.queue("new-session", "", 0)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil),
			Source: "absent", Admitted: fullLifecycleAdmitted(),
		})
		if err == nil {
			t.Fatal("missing prior doc admitted")
		}
		requireLocalCode(t, err, "terminal_backend_process_failed", "backend binding read")
		if fx.runner.callCount() != 0 {
			t.Fatalf("pre-effects refusal executed %v", fx.runner.subcommands())
		}
	})
}

// TestExecuteRestoreRefusesSubstitutedInstance is the R1 killer: a
// binding document that substitutes another instance for the recorded
// one refuses the restore binding image before any effect.
func TestExecuteRestoreRefusesSubstitutedInstance(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	substituted := lxBindingDoc(t, func(object map[string]any) {
		object["terminal_instance_id"] = "0198f4c8-8e50-7f66-8f70-222222222222"
	})
	if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, substituted); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("substituted instance admitted: %+v", outcome.Binding)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "restore binding image")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore executed %v", fx.runner.subcommands())
	}
}

// TestExecuteStatusRefusesNegativeAttachedCount is the R3 killer: a
// negative attached-client count (exactly -1 here) is a malformed
// probe, never attachability evidence, and refuses unknown.
func TestExecuteStatusRefusesNegativeAttachedCount(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "ax\n", 0).queue("list-sessions", lxInstance+"|-1\n", 0)
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, false, nil),
		Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatal("negative attached-client count admitted")
	}
	requireEngineCode(t, err, "terminal_backend_unavailable")
}

// TestExecuteCreateRefusesWritableRoot is the R5 killer: the custody
// gate runs before dispatch on every entry, so create refuses a
// group/world-writable root exactly like status and restore do.
func TestExecuteCreateRefusesWritableRoot(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if err := os.Chmod(fx.root, 0o777); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(fx.root, 0o700) }()
	fx.runner.queue("new-session", "", 0)
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatal("writable root admitted by create")
	}
	requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused create executed %v", fx.runner.subcommands())
	}
}

// lifecycleOperations is every operation the lifecycle entry serves:
// the custody pre-gate runs before dispatch on all of them.
var lifecycleOperations = []string{
	"create", "attach", "status", "quiesce-input",
	"wait-safe-boundary", "request-stop", "terminate-stale", "restore",
}

// TestExecuteEachOperationRefusesWritableRoot pins the entry
// dimension of the root custody class: every operation refuses a
// world-writable root before its body is ever parsed (empty bodies).
func TestExecuteEachOperationRefusesWritableRoot(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if err := os.Chmod(fx.root, 0o777); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(fx.root, 0o700) }()
	for _, operation := range lifecycleOperations {
		t.Run(operation, func(t *testing.T) {
			_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: operation})
			if err == nil {
				t.Fatalf("%s admitted a writable root", operation)
			}
			requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
		})
	}
}

// TestExecuteEachOperationRefusesWritableAncestor pins the entry
// dimension of the ancestor custody class: every operation refuses a
// writable ancestor before dispatch.
func TestExecuteEachOperationRefusesWritableAncestor(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	parent := filepath.Dir(fx.root)
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(parent, 0o700) }()
	for _, operation := range lifecycleOperations {
		t.Run(operation, func(t *testing.T) {
			_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: operation})
			if err == nil {
				t.Fatalf("%s admitted a writable ancestor", operation)
			}
			requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
		})
	}
}

// TestExecuteEachOperationRefusesNonSocketKind pins the entry
// dimension of the socket-kind class: every operation refuses a
// regular file at the socket path before dispatch.
func TestExecuteEachOperationRefusesNonSocketKind(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if err := os.WriteFile(fx.socket, []byte("not-a-socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, operation := range lifecycleOperations {
		t.Run(operation, func(t *testing.T) {
			_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: operation})
			if err == nil {
				t.Fatalf("%s admitted a non-socket file", operation)
			}
			requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket kind")
		})
	}
}

// TestExecuteEachOperationRefusesBadLeafMode pins the entry dimension
// of the leaf custody class: every operation refuses a world-writable
// leaf before dispatch.
func TestExecuteEachOperationRefusesBadLeafMode(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	leaf := filepath.Join(fx.root, "tmux")
	if err := os.Chmod(leaf, 0o777); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(leaf, 0o700) }()
	for _, operation := range lifecycleOperations {
		t.Run(operation, func(t *testing.T) {
			_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: operation})
			if err == nil {
				t.Fatalf("%s admitted a bad leaf mode", operation)
			}
			requireLocalCode(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
		})
	}
}

// TestExecuteCustodyNarrownessTwins pins the narrowness of the class
// admission rows: a 0770 root, a 0770 ancestor, a directory at the
// socket path, and a 0770 leaf each still refuse, so the rows that
// admit exactly one shape stay narrow.
func TestExecuteCustodyNarrownessTwins(t *testing.T) {
	t.Run("root-0770", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		if err := os.Chmod(fx.root, 0o770); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(fx.root, 0o700) }()
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "status"})
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	})
	t.Run("ancestor-0770", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		parent := filepath.Dir(fx.root)
		if err := os.Chmod(parent, 0o770); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(parent, 0o700) }()
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "status"})
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	})
	t.Run("directory-at-socket", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		if err := os.Mkdir(fx.socket, 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "status"})
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket kind")
	})
	t.Run("leaf-0770", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		leaf := filepath.Join(fx.root, "tmux")
		if err := os.Chmod(leaf, 0o770); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(leaf, 0o700) }()
		_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "status"})
		requireLocalCode(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
	})
}

// TestExecuteQuiesceEmptyRecordedTimeRefusesIntegrity pins the
// recorded-but-timeless report: a quiesce outcome document that
// carries no closure time refuses the integrity failure instead of
// reporting an empty timestamp as the event time.
func TestExecuteQuiesceEmptyRecordedTimeRefusesIntegrity(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	req := OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), OperationOutcome{Operation: "quiesce-input"}); err != nil {
		t.Fatal(err)
	}
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("timeless replay reported %q", second.InputClosedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "quiesce outcome image")
}

// TestExecuteBoundaryEmptyRecordedTimeRefusesIntegrity is the
// boundary twin: a timeless boundary report refuses instead of
// reporting an empty observed time.
func TestExecuteBoundaryEmptyRecordedTimeRefusesIntegrity(t *testing.T) {
	const kind = "ax_checkpoint_boundary"
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	req := OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, kind, 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}
	if _, err := fx.lc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if err := fx.lc.States.RecordOutcome(lxBoundaryKey(kind), OperationOutcome{Operation: "wait-safe-boundary"}); err != nil {
		t.Fatal(err)
	}
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("timeless replay reported %q", second.BoundaryObservedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "boundary outcome image")
}

// TestExecuteBoundaryRecordCrashRefusesIntegrity pins the boundary
// record-failure arm: when the boundary outcome record fails, the
// committed effect refuses the integrity failure (no unrecorded
// time), and the retry after recovery re-records without a second
// wait.
func TestExecuteBoundaryRecordCrashRefusesIntegrity(t *testing.T) {
	const kind = "ax_checkpoint_boundary"
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterInstall: func(path string) error {
		if strings.Contains(path, string(filepath.Separator)+"outcomes"+string(filepath.Separator)) {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("wait-for", "", 0)
	req := OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, kind, 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}
	first, err := fx.lc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("unrecorded boundary time reported: %s", first.BoundaryObservedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "boundary outcome image")
	if first.Mutation == nil || first.BoundaryObservedAt != "" {
		t.Fatalf("refused report manufactures time: %+v", first)
	}
	fx.lc.States.WithHooks(nil)
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if second.BoundaryObservedAt == "" {
		t.Fatal("recovery retry reports no time")
	}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want exactly the first wait", fx.runner.callCount())
	}
}

// TestObserveStatusExactProtocolDriftNonMatches pins the protocol
// member at the backend entry: the closed-shape grammar refuses
// foreign protocol majors before observation, so the
// recorded-vs-query protocol comparison is reachable only here,
// where drift reads non-match.
func TestObserveStatusExactProtocolDriftNonMatches(t *testing.T) {
	runner := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	reqBackend, _ := statusBackend(t, runner, admittedWith("local_attach"))
	stageStatusBinding(t, reqBackend)
	body, err := terminstance.ParseStatusBody(lxStatusBody(t, true, false, nil))
	if err != nil {
		t.Fatal(err)
	}
	body.ProtocolVersion = "9.9.9"
	observed, err := reqBackend.ObserveStatus(context.Background(), body)
	if err != nil {
		t.Fatalf("non-match refused: %v", err)
	}
	if observed.IdentityMatch || observed.State != terminalbackend.StateAbsent {
		t.Fatalf("protocol drift matched: %+v", observed)
	}
}

// TestExecuteRestoreRefusesSubstitutedSession pins the session member
// of the prior agreement: a binding document naming another session
// refuses the restore binding image before any effect.
func TestExecuteRestoreRefusesSubstitutedSession(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	substituted := lxBindingDoc(t, func(object map[string]any) {
		object["session_id"] = lxSessionB
	})
	if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, substituted); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("substituted session admitted: %+v", outcome.Binding)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "restore binding image")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore executed %v", fx.runner.subcommands())
	}
}

// TestExecuteRestoreRefusesSubstitutedBackend pins the backend member
// of the prior agreement: a binding document naming another backend
// refuses the restore binding image before any effect.
func TestExecuteRestoreRefusesSubstitutedBackend(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	substituted := lxBindingDoc(t, func(object map[string]any) {
		object["terminal_backend_id"] = "ax.conpty"
	})
	if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, substituted); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("substituted backend admitted: %+v", outcome.Binding)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "restore binding image")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore executed %v", fx.runner.subcommands())
	}
}
