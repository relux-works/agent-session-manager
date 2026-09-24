//go:build !windows

package tmuxserver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// Rework regressions for the rev1 review findings P1-A through P2-B.
// Every test drives a production entry; literals come from the pinned
// specification, the tmux manual, or observed tmux 3.6a behavior,
// never from production constants.

const (
	// lxRemoteHost is a second host identity for after-restore branch
	// tests: a winner naming it is remote.
	lxRemoteHost = "0198f4c8-8e50-7f66-8f70-777777777772"
)

func TestExecuteBoundaryWaitsForSignal(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	var waited []string
	fx.runner.onRun = func(argv []string) {
		if len(argv) > 3 && argv[3] == "wait-for" {
			waited = append([]string(nil), argv...)
		}
	}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.SafeBoundaryEvidenceID == "" || outcome.BoundaryObservedAt == "" {
		t.Fatalf("outcome = %+v", outcome)
	}
	want := []string{"tmux", "-S", fx.socket, "wait-for", "ax-boundary-" + lxQuiesce}
	if len(waited) != len(want) {
		t.Fatalf("wait argv = %q, want %q", waited, want)
	}
	for i := range want {
		if waited[i] != want[i] {
			t.Fatalf("wait argv = %q, want %q", waited, want)
		}
	}
	for _, element := range waited[4:] {
		if element == "-L" || element == "-S" || element == "-U" {
			t.Fatalf("blocking wait carries channel flag %q: %q", element, waited)
		}
	}
}

func TestExecuteBoundaryBindsProofKind(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	first, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "provider_quiescence", 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	second, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "provider_process_exit", 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.SafeBoundaryEvidenceID == second.SafeBoundaryEvidenceID {
		t.Fatalf("proof kind not bound: %s", first.SafeBoundaryEvidenceID)
	}
}

func TestObserveBoundaryBindsProviderRows(t *testing.T) {
	script := func(rows string) *fakeRunner {
		return newFakeRunner().queue("wait-for", "", 0).queue("list-panes", rows, 0)
	}
	observe := func(t *testing.T, runner *fakeRunner) string {
		t.Helper()
		reqBackend := effectBackend(t, terminalbackend.OperationWaitSafeBoundary, runner)
		reqBackend.quiescence = lxQuiesce
		reqBackend.channel = BoundaryChannelPrefix + lxQuiesce
		reqBackend.proofKind = terminstance.ProofProviderQuiescence
		reqBackend.waitCtx = context.Background()
		id, err := reqBackend.PerformEffect(context.Background(), terminalbackend.EffectSafeBoundaryObserved)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	if observe(t, script("sh\n")) == observe(t, script("zsh\n")) {
		t.Fatal("provider rows not bound into boundary evidence")
	}
}

func TestExecuteQuiesceBarrierSequence(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.QuiescenceGeneration != lxQuiesce || outcome.InputClosedAt == "" {
		t.Fatalf("outcome = %+v", outcome)
	}
	got := fx.runner.subcommands()
	if len(got) != 2 || got[0] != "lock-session" || got[1] != "detach-client" {
		t.Fatalf("calls = %v, want lock then detach", got)
	}
}

func TestExecuteAttachRefusesInputWhenQuiescing(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	// New input-authorized attaches refuse once quiescing records.
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("input attach admitted under quiescing")
	} else {
		requireLocalCode(t, err, "local_precondition_failed", "attach quiesced input")
	}
	// Read-only attaches stay admitted: quiesce retains observation.
	readonly, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = lxClientB
			object["input_authorized"] = false
			object["authorization"] = lxAttachAuth("local_only", false)
		}), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatalf("read-only attach refused under quiescing: %v", err)
	}
	if readonly.Attach == nil || readonly.Attach.InputAuthorized {
		t.Fatalf("attach = %+v", readonly.Attach)
	}
}

func TestExecuteQuiesceEmitsNoProviderInput(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	for _, call := range fx.runner.calls {
		for _, element := range call {
			if element == "send-keys" {
				t.Fatalf("quiesce emitted provider input: %q", call)
			}
		}
	}
}

func TestExecuteQuiesceRefusesFailedClose(t *testing.T) {
	for _, tc := range []struct {
		name   string
		script func(*fakeRunner) *fakeRunner
	}{
		{"lock", func(r *fakeRunner) *fakeRunner { return r.queue("lock-session", "", 1) }},
		{"detach", func(r *fakeRunner) *fakeRunner { return r.queue("lock-session", "", 0).queue("detach-client", "", 1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			tc.script(fx.runner)
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
			})
			if err == nil {
				t.Fatal("failed close step committed")
			}
			// The quiesce row carries no process-failure member
			// (landed map, v0.6.0 §4.C), so the engine's
			// vocabulary arm reports the backend refusal as the
			// protocol error with the unavailable observation;
			// the refusal is this leaf's contract, the code
			// and observation the engine's.
			requireEngineCode(t, err, "terminal_backend_protocol_error")
			if outcome.Mutation == nil || outcome.Mutation.After != "unavailable" {
				t.Fatalf("mutation = %+v", outcome.Mutation)
			}
		})
	}
}

func TestExecuteReadOnlyAttachIsReadOnly(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	readonly, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, func(object map[string]any) {
			object["input_authorized"] = false
			object["authorization"] = lxAttachAuth("local_only", false)
		}), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, element := range readonly.Argv {
		if element == "-r" {
			found = true
		}
	}
	if !found {
		t.Fatalf("read-only attach vector lacks -r: %q", readonly.Argv)
	}
	writable, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = lxClientB
		}), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, element := range writable.Argv {
		if element == "-r" {
			t.Fatalf("writable attach vector carries -r: %q", writable.Argv)
		}
	}
}

func TestClassifyAbsenceMarkers(t *testing.T) {
	cases := []struct {
		name    string
		result  RunResult
		present bool
		absent  bool
	}{
		{"exit-zero", RunResult{ExitCode: 0}, true, false},
		{"negative", RunResult{ExitCode: -1}, false, false},
		{"negative-with-marker", RunResult{ExitCode: -1, Stderr: []byte("can't find session: s")}, false, false},
		{"bare-exit-one", RunResult{ExitCode: 1}, false, false},
		{"no-session", RunResult{ExitCode: 1, Stderr: []byte("can't find session: s")}, false, true},
		{"no-window", RunResult{ExitCode: 1, Stderr: []byte("can't find window: s")}, false, true},
		{"no-socket", RunResult{ExitCode: 1, Stderr: []byte("error connecting to /r/tmux/ax.sock (No such file or directory)")}, false, true},
		{"denied", RunResult{ExitCode: 1, Stderr: []byte("error connecting to /r/tmux/ax.sock (Permission denied)")}, false, false},
		{"refused", RunResult{ExitCode: 1, Stderr: []byte("error connecting to /r/tmux/ax.sock (Connection refused)")}, false, false},
		{"unmarked", RunResult{ExitCode: 1, Stderr: []byte("usage: has-session")}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			present, absent := classifyAbsence(tc.result, absentMarkerNoSession, absentMarkerNoWindow, absentMarkerNoSocket)
			if present != tc.present || absent != tc.absent {
				t.Fatalf("classify = (%v, %v), want (%v, %v)", present, absent, tc.present, tc.absent)
			}
		})
	}
}

func TestExecuteStopUnknownProbeIsNotClosure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		script func(*fakeRunner) *fakeRunner
	}{
		{"killed", func(r *fakeRunner) *fakeRunner {
			return r.queue("send-keys", "", 0).queue("has-session", "", -1)
		}},
		{"killed-with-marker", func(r *fakeRunner) *fakeRunner {
			return r.queue("send-keys", "", 0).queueFull("has-session", "", "can't find session: "+lxInstance, -1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			tc.script(fx.runner)
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "request-stop", Body: lxStopBody(t, 5, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
			})
			if err == nil {
				t.Fatalf("killed probe closed: %+v", outcome)
			}
			if outcome.ProcessClosed || outcome.StoreClosed {
				t.Fatalf("unknown probe reported closure: %+v", outcome)
			}
		})
	}
}

func TestExecuteStopUnmarkedExitIsNotClosure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("send-keys", "", 0).
		queueFull("has-session", "", "error connecting to "+fx.socket+" (Permission denied)", 1)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "request-stop", Body: lxStopBody(t, 5, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("unmarked probe closed: %+v", outcome)
	}
	if outcome.ProcessClosed || outcome.StoreClosed {
		t.Fatalf("unknown probe reported closure: %+v", outcome)
	}
}

func TestExecuteStatusUnknownProbeIsUnavailable(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queueFull("list-panes", "", "error connecting to "+fx.socket+" (Connection refused)", 1)
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, false, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatal("unmarked probe read absence")
	} else {
		requireEngineCode(t, err, "terminal_backend_unavailable")
	}
}

func TestExecuteTerminateUnknownConfirmIsNotTerminated(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	presented, winner := terminateFencing()
	fx.runner.queue("kill-session", "", 1).
		queueFull("has-session", "", "error connecting to "+fx.socket+" (Permission denied)", 1)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "terminate-stale", Body: lxTerminateBody(t, nil), Source: "stale_fenced",
		Admitted: fullLifecycleAdmitted(), Presented: presented, Winner: winner,
		HasWinner: true, ForceRecovery: true, DiagnosticsPreserved: true,
	})
	if err == nil {
		t.Fatalf("unknown confirm terminated: %+v", outcome)
	}
	if outcome.TargetClosed {
		t.Fatalf("unknown confirm reported termination: %+v", outcome)
	}
}

func TestCheckSocketCustodyRefusesIntermediateSymlink(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	alias := filepath.Join(filepath.Dir(fx.root), "alias")
	if err := os.Symlink(filepath.Dir(fx.root), alias); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(alias, filepath.Base(fx.root))
	if err := CheckSocketCustody(filepath.Join(root, "tmux", "ax.sock"), root, scalar.PlatformMacOS); err == nil {
		t.Fatal("intermediate symlink admitted at before-connect custody")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	}
}

func TestExecuteCustodyEntryRoots(t *testing.T) {
	for _, entry := range []string{"status", "restore"} {
		for _, which := range []string{"root", "ancestor"} {
			t.Run(entry+"/"+which, func(t *testing.T) {
				fx := newLifecycleFixture(t, fullLifecycleAdmitted())
				path := fx.root
				if which == "ancestor" {
					path = filepath.Dir(fx.root)
				}
				if err := os.Chmod(path, 0o777); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(path, 0o755)
				req := OpRequest{Operation: entry, Body: lxStatusBody(t, true, false, nil), Source: "parked", Admitted: fullLifecycleAdmitted()}
				if entry == "restore" {
					recordBinding(t, fx, lxDigestA)
					req.Body = lxRestoreBody(t, lxDigestA, nil)
					req.Source = "absent"
				}
				_, err := fx.lc.Execute(context.Background(), req)
				want := "tmux server refused: tmux_unsafe_socket_path at socket " + which
				if err == nil || err.Error() != want {
					t.Fatalf("want %s got %v", want, err)
				}
			})
		}
	}
}

func TestCheckSocketCustodyNoCacheAfterChange(t *testing.T) {
	root, socket := lxCustodyRoot(t)
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err != nil {
		t.Fatalf("compliant refused: %v", err)
	}
	if err := os.Chmod(root, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("widened root admitted after a passing check")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(root)
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(parent, 0o700)
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("widened ancestor admitted after a passing check")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	}
}

func TestCheckSocketCustodyIdentityMismatchRefuses(t *testing.T) {
	root, socket := lxCustodyRoot(t)
	previous := sameCustodyIdentity
	sameCustodyIdentity = func(a, b os.FileInfo) bool { return false }
	defer func() { sameCustodyIdentity = previous }()
	if err := CheckSocketCustody(socket, root, scalar.PlatformMacOS); err == nil {
		t.Fatal("swapped root admitted")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket root")
	}
}

func TestSocketCustodySymlinkRefusesAtProbeAndSpawn(t *testing.T) {
	short, err := os.MkdirTemp("/tmp", "axalias")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(short) })
	holder := filepath.Join(short, "holder")
	if err := os.Mkdir(holder, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(holder, "root")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(short, "alias")
	if err := os.Symlink(holder, alias); err != nil {
		t.Fatal(err)
	}
	// The aliased root spells through an intermediate symlink above
	// an otherwise valid root; the leaf arm follows it (first-leaf
	// behavior), and the ancestor arm refuses it here.
	aliasedRoot := filepath.Join(alias, "root")
	aliased := filepath.Join(aliasedRoot, RuntimeDirName, SocketName)
	prober := ServerProber{Root: aliasedRoot, Platform: scalar.PlatformMacOS, Dialer: &fakeDialer{outcomes: []DialOutcome{DialLive}}, Admit: func() (RealmAdmission, error) {
		return RealmAdmission{}, errors.New("admission must not run past refused custody")
	}}
	if _, err := prober.Probe(aliased); err == nil {
		t.Fatal("aliased root probed past custody")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	}
	spawner := ServerSpawner{Root: aliasedRoot, Platform: scalar.PlatformMacOS, Runner: newFakeRunner()}
	argv := []string{"tmux", "-S", aliased, "new-session", "-d", "-s", lxSession, "ax", "pane", lxSession}
	if err := spawner.Spawn(aliased, argv); err == nil {
		t.Fatal("aliased root spawned past custody")
	} else {
		requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket ancestor")
	}
}

func TestExecuteRestoreReturnsBinding(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Binding == nil {
		t.Fatal("successful restore omits required TerminalInstanceBinding")
	}
	if outcome.Binding.SessionID != lxSession ||
		outcome.Binding.TerminalInstanceID != lxInstance ||
		outcome.Binding.TerminalBackendID != "ax.tmux" ||
		outcome.Binding.BackendGeneration != lxGeneration {
		t.Fatalf("binding = %+v", outcome.Binding)
	}
	if !outcome.RestoredParked || outcome.PriorBindingID != lxDigestA {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteRestoreRefusesSwappedBindingDoc(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	swapped := lxBindingDoc(t, func(object map[string]any) {
		object["session_id"] = lxSessionB
	})
	if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, swapped); err != nil {
		t.Fatal(err)
	}
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("swapped binding doc admitted: %+v", outcome.Binding)
	} else {
		requireLocalCode(t, err, "terminal_backend_integrity_failure", "restore binding image")
	}
}

func TestExecuteCreatePersistsBindingDocForRestore(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("new-session", "", 0)
	created, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Binding == nil {
		t.Fatalf("create outcome = %+v", created)
	}
	// Reboot between create and restore: the receipt table is
	// reboot-volatile (a same-boot restore under the create key is
	// the specified idempotency_mismatch), while the binding store
	// is reboot-durable and still answers.
	rebooted, err := terminstance.OpenReceiptStore(filepath.Join(fx.data, "receipts2"))
	if err != nil {
		t.Fatal(err)
	}
	fx.lc.Receipts = rebooted
	fx.runner.queue("new-session", "", 0)
	restored, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, created.Binding.BindingID, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Binding == nil || restored.Binding.BindingID != created.Binding.BindingID {
		t.Fatalf("restored binding = %+v, want %s", restored.Binding, created.Binding.BindingID)
	}
}

func TestExecuteCreateRefusesBindingDocFailure(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterInstall: func(path string) error {
		if strings.Contains(path, string(filepath.Separator)+"bindings"+string(filepath.Separator)) {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("new-session", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "create", Body: lxCreateBody(t, nil), Source: "absent", Admitted: fullLifecycleAdmitted(),
	}); err == nil {
		t.Fatal("create committed without its binding document")
	} else {
		requireEngineCode(t, err, "terminal_backend_process_failed")
	}
}

func TestExecuteRestoreExpiredGrantRefusesUnauthorized(t *testing.T) {
	// The backend restore entry refuses an expired grant at restore
	// authorization, before any branch or effect: that refusal is the
	// separate restore-authorization contract working, and the
	// lapsed-grant remote offer composes above it (the wrapper entry
	// refreshes first, so the offer path never presents the lapsed
	// grant here). This test replaces the revoked
	// TestExecuteRestoreRemoteOfferOnLapsedGrant, which claimed a
	// lapsed-grant offer through this entry but staged a future
	// expiry: with a truly expired grant this entry refuses and must
	// keep refusing (rev2 P1-D).
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "restore",
		Body: lxRestoreBody(t, lxDigestA, func(object map[string]any) {
			object["context"].(map[string]any)["authorization"].(map[string]any)["expires_at"] = "2026-09-01T11:00:00.000Z"
		}),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatalf("expired grant admitted: %+v", outcome.Binding)
	}
	requireEngineCode(t, err, "terminal_backend_unauthorized")
	if fx.runner.callCount() != 0 {
		t.Fatalf("refused restore executed %v", fx.runner.subcommands())
	}
}

func TestExecuteBoundaryRefusesFailedWait(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 1)
	_, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err == nil {
		t.Fatal("failed boundary wait committed")
	}
	// The fixture clock leaves the wait context expired, so the
	// failed wait reports the timeout; the live-context refused
	// exit is pinned at the helper by
	// TestPerformEffectBoundaryRefusedExit.
	requireEngineCode(t, err, "quiesce_timeout")
}

func TestExecuteQuiesceReplayKeepsTimestamps(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	req := OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}
	first, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.InputClosedAt != second.InputClosedAt {
		t.Fatalf("same receipt changed closure time: %s -> %s", first.InputClosedAt, second.InputClosedAt)
	}
}

func TestExecuteBoundaryReplayKeepsProofAndTime(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("wait-for", "", 0)
	req := OpRequest{
		Operation: "wait-safe-boundary", Body: lxBoundaryBody(t, "ax_checkpoint_boundary", 60000, nil),
		Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}
	first, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.BoundaryObservedAt != second.BoundaryObservedAt {
		t.Fatalf("same receipt changed boundary time: %s -> %s", first.BoundaryObservedAt, second.BoundaryObservedAt)
	}
	if first.SafeBoundaryEvidenceID != second.SafeBoundaryEvidenceID {
		t.Fatalf("same receipt changed proof: %s -> %s", first.SafeBoundaryEvidenceID, second.SafeBoundaryEvidenceID)
	}
}

func TestOutcomeRecordCrashConverges(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.States.WithHooks(&StateHooks{AfterInstall: func(path string) error {
		if strings.Contains(path, string(filepath.Separator)+"outcomes"+string(filepath.Separator)) {
			return errFakeTransport
		}
		return nil
	}})
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	req := OpRequest{Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted()}
	first, err := fx.lc.Execute(context.Background(), req)
	// The receipt committed but the report did not persist: the
	// committed effect fails on the unprovable time (no invented
	// timestamp), and the retry re-records. A missing report
	// re-observes; a corrupt one refuses (rev2 P2-A).
	if err == nil {
		t.Fatalf("unrecorded closure time reported: %s", first.InputClosedAt)
	}
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "quiesce outcome image")
	if first.Mutation == nil || first.InputClosedAt != "" {
		t.Fatalf("refused report manufactures time: %+v", first)
	}
	fx.lc.States.WithHooks(nil)
	fx.clock.Sleep(time.Second)
	second, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", fx.runner.callCount())
	}
	fx.clock.Sleep(time.Second)
	third, err := fx.lc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if second.InputClosedAt != third.InputClosedAt {
		t.Fatalf("report not stable after record: %s -> %s", second.InputClosedAt, third.InputClosedAt)
	}
	if first.Mutation.EvidenceIDs[0] != second.Mutation.EvidenceIDs[0] {
		t.Fatalf("evidence changed across the crash window: %v vs %v", first.Mutation.EvidenceIDs, second.Mutation.EvidenceIDs)
	}
}

func TestExecuteRestoreIdempotent(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	raw := lxRestoreBody(t, lxDigestA, nil)
	first, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "restore", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.lc.Execute(context.Background(), OpRequest{Operation: "restore", Body: raw, Source: "absent", Admitted: fullLifecycleAdmitted()})
	if err != nil {
		t.Fatal(err)
	}
	if fx.runner.callCount() != 1 {
		t.Fatalf("calls = %d, want 1", fx.runner.callCount())
	}
	if second.Mutation.After != first.Mutation.After || len(second.Mutation.EvidenceIDs) != len(first.Mutation.EvidenceIDs) {
		t.Fatalf("replay diverged: %+v vs %+v", second.Mutation, first.Mutation)
	}
	if second.Binding == nil || second.Binding.BindingID != first.Binding.BindingID {
		t.Fatalf("replay binding = %+v", second.Binding)
	}
}

func TestExecuteStatusPresentIncludesIdentity(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|0\n", 0)
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	report := outcome.Status
	if report == nil {
		t.Fatal("status omits its report")
	}
	if !report.IdentityMatch || !report.WrapperPresent || len(report.EvidenceIDs) == 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.State != terminalbackend.StateParked {
		t.Fatalf("state = %s", report.State)
	}
}

func TestObserveStatusAbsentProbe(t *testing.T) {
	runner := newFakeRunner().queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
	reqBackend, _ := statusBackend(t, runner, admittedWith("local_attach"))
	stageStatusBinding(t, reqBackend)
	observed, err := reqBackend.ObserveStatus(context.Background(), exactStatusBody(t))
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != terminalbackend.StateAbsent || observed.WrapperPresent || len(observed.EvidenceIDs) == 0 {
		t.Fatalf("observed = %+v", observed)
	}
}

func TestInstanceStatesOutcomeRoundTrip(t *testing.T) {
	states, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := lxInstance + "/quiesce/" + lxQuiesce
	if _, found, err := states.LookupOutcome(key); err != nil || found {
		t.Fatalf("lookup = %v, %v, want absence", found, err)
	}
	want := OperationOutcome{Operation: "quiesce-input", InputClosedAt: "2026-09-01T12:00:00.000Z"}
	if err := states.RecordOutcome(key, want); err != nil {
		t.Fatal(err)
	}
	got, found, err := states.LookupOutcome(key)
	if err != nil || !found {
		t.Fatalf("lookup = %v, %v, want the record", found, err)
	}
	if got != want {
		t.Fatalf("outcome = %+v, want %+v", got, want)
	}
	if err := states.RecordOutcome("", want); err == nil {
		t.Fatal("empty outcome key recorded")
	}
	if err := states.RecordOutcome("", OperationOutcome{}); err == nil {
		t.Fatal("empty outcome key with empty operation recorded")
	}
	if _, _, err := states.LookupOutcome(""); err == nil {
		t.Fatal("empty outcome key looked up")
	}
}

func TestInstanceStatesBindingDocRoundTrip(t *testing.T) {
	states, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := states.LookupBindingDoc(lxSession, lxBootstrap); err != nil || found {
		t.Fatalf("lookup = %v, %v, want absence", found, err)
	}
	doc := lxBindingDoc(t, nil)
	if err := states.RecordBindingDoc(lxSession, lxBootstrap, doc); err != nil {
		t.Fatal(err)
	}
	got, found, err := states.LookupBindingDoc(lxSession, lxBootstrap)
	if err != nil || !found {
		t.Fatalf("lookup = %v, %v, want the document", found, err)
	}
	if string(got) != string(doc) {
		t.Fatalf("doc = %s, want %s", got, doc)
	}
	if err := states.RecordBindingDoc("not-a-uuid", lxBootstrap, doc); err == nil {
		t.Fatal("malformed binding identity recorded")
	}
	if err := states.RecordBindingDoc(lxSession, lxBootstrap, []byte("{broken")); err == nil {
		t.Fatal("non-JSON binding document recorded")
	}
	if _, _, err := states.LookupBindingDoc("not-a-uuid", lxBootstrap); err == nil {
		t.Fatal("malformed binding identity looked up")
	}
}
