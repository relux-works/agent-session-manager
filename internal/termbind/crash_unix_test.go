//go:build darwin || linux

package termbind

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// Real process-termination evidence for the attach receipt seam and the
// lost-create recovery seam. The attach child runs the genuine production
// Attach entry and SIGKILLs itself in the AfterInstall hook, after the
// no-replace commit and before the directory sync. The recovery child
// durably binds the bootstrap pair, durably commits the backend effect to
// the observation file, and SIGKILLs itself before reading any result —
// the lost create result. Each parent proves the interrupted run left the
// complete durable fact behind and the retry converges to it.
const (
	crashChildMode  = "AX_TERMBIND_CRASH_MODE"
	crashChildStore = "AX_TERMBIND_CRASH_STORE"
	crashChildFile  = "AX_TERMBIND_CRASH_FILE"
)

func TestAttachCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildMode) == "attach" {
		attachCrashChild(t)
		return
	}
	storeDir := t.TempDir()
	if _, err := OpenAttachStore(storeDir); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestAttachCrashChildSelfTerminates$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildMode+"=attach",
		crashChildStore+"="+storeDir,
	)
	runErr := child.Run()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v, want a signal-kill exit", runErr)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v, want killed by SIGKILL", exitErr)
	}
	store, err := OpenAttachStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	proven, found, err := store.Lookup(fixtureSession, fixtureInstance, fixtureClient)
	if err != nil {
		t.Fatalf("Lookup() after kill error = %v", err)
	}
	if !found {
		t.Fatal("Lookup() after kill reports absence, want the committed receipt")
	}
	if proven.ClientID != fixtureClient || proven.Transport != "local_only" || !proven.InputAuthorized {
		t.Fatalf("Lookup() after kill = %+v, want the complete receipt", proven)
	}
	second, replayed, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow())
	if err != nil {
		t.Fatalf("Attach() retry after kill error = %v", err)
	}
	if !replayed || second != proven {
		t.Fatalf("Attach() retry = %+v replayed %v, want the one receipt", second, replayed)
	}
}

func attachCrashChild(t *testing.T) {
	t.Helper()
	store, err := OpenAttachStore(os.Getenv(crashChildStore))
	if err != nil {
		t.Fatalf("child OpenAttachStore() error = %v", err)
	}
	store.WithHooks(&AttachHooks{
		AfterInstall: func(path string) error {
			proc, err := os.FindProcess(os.Getpid())
			if err != nil {
				t.Fatalf("child find self: %v", err)
			}
			_ = proc.Signal(syscall.SIGKILL)
			select {}
		},
	})
	_, _, _ = store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow())
	t.Fatal("child Attach() returned past SIGKILL")
}

func TestRecoverCreateAfterKillRecoversChild(t *testing.T) {
	if os.Getenv(crashChildMode) == "recover" {
		recoverCrashChild(t)
		return
	}
	storeDir := t.TempDir()
	store, err := axpane.Open(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	observationFile := storeDir + "/backend-observation"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestRecoverCreateAfterKillRecoversChild$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildMode+"=recover",
		crashChildStore+"="+storeDir,
		crashChildFile+"="+observationFile,
	)
	runErr := child.Run()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v, want a signal-kill exit", runErr)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v, want killed by SIGKILL", exitErr)
	}
	// The parent never saw the child's result: it recovers from the
	// durable binding plus a status read over the durable backend effect.
	backend := &mockBackend{observe: func(terminstance.StatusBody) (terminstance.StatusObservation, error) {
		raw, err := os.ReadFile(observationFile)
		if err != nil {
			return terminstance.StatusObservation{}, err
		}
		if string(raw) != "committed" {
			return terminstance.StatusObservation{}, errors.New("backend effect lost")
		}
		return exactObservation(terminalbackend.StateActive, true), nil
	}}
	verdict, err := RecoverCreate(context.Background(), RecoveryDeps{Bindings: store, Engine: testEngine(backend)}, recoveryRequest(fixtureBootstrap, true))
	if err != nil {
		t.Fatalf("RecoverCreate() after kill error = %v", err)
	}
	if verdict.Outcome != RecoveryChild || verdict.Binding.TerminalInstanceID != fixtureInstance {
		t.Fatalf("verdict after kill = %+v, want the one recorded child", verdict)
	}
}

func recoverCrashChild(t *testing.T) {
	t.Helper()
	store, err := axpane.Open(os.Getenv(crashChildStore))
	if err != nil {
		t.Fatalf("child Open() error = %v", err)
	}
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, axpane.Binding{
		SessionID:          fixtureSession,
		OperationID:        fixtureBootstrap,
		TerminalInstanceID: fixtureInstance,
		BindingDigest:      seedDigest(0xB1),
	}); err != nil {
		t.Fatalf("child Bind() error = %v", err)
	}
	if err := os.WriteFile(os.Getenv(crashChildFile), []byte("committed"), 0o600); err != nil {
		t.Fatalf("child commit effect: %v", err)
	}
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("child find self: %v", err)
	}
	_ = proc.Signal(syscall.SIGKILL)
	select {}
}

func TestEmitAppendHookSeams(t *testing.T) {
	t.Parallel()
	t.Run("before-write aborts without an event", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		repository.BeforeWrite = func() error { return errors.New("injected write fault") }
		if _, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), emitFacts(world), emitAdmission(t, world)); err == nil {
			t.Fatal("EmitTerminalCreated(write fault) succeeded, want refusal")
		}
		repository.BeforeWrite = nil
		if len(mustListEvents(t, repository)) != 1 {
			t.Fatal("aborted emission left an event, want none")
		}
		if _, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), emitFacts(world), emitAdmission(t, world)); err != nil {
			t.Fatalf("EmitTerminalCreated(retry) error = %v", err)
		}
		if len(mustListEvents(t, repository)) != 2 {
			t.Fatal("retry left no event, want one")
		}
	})
	t.Run("after-commit keeps the commit and replays", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		params := emitParams(t, repository)
		repository.AfterCommit = func() error { return errors.New("injected commit fault") }
		if _, _, _, err := EmitTerminalCreated(repository, params, emitFacts(world), emitAdmission(t, world)); err == nil {
			t.Fatal("EmitTerminalCreated(commit fault) succeeded, want the hook error")
		}
		repository.AfterCommit = nil
		events := mustListEvents(t, repository)
		if len(events) != 2 {
			t.Fatalf("chain holds %d events, want the kept commit", len(events))
		}
		kept := events[1].EventID
		// The byte-identical retry replays the kept commit instead of
		// duplicating the chain.
		second, _, _, err := EmitTerminalCreated(repository, params, emitFacts(world), emitAdmission(t, world))
		if err != nil {
			t.Fatalf("EmitTerminalCreated(retry) error = %v", err)
		}
		if second.EventID != kept {
			t.Fatalf("retry event = %q, want the kept %q", second.EventID, kept)
		}
		if after := mustListEvents(t, repository); len(after) != 2 {
			t.Fatalf("chain holds %d events after replay, want 2", len(after))
		}
	})
}
