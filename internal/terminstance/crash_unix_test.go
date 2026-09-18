//go:build darwin || linux

package terminstance

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Real process-termination evidence for the idempotency receipt seam:
// the child runs the genuine production ExecuteMutating entry and
// SIGKILLs itself in the store AfterCommit hook, after the no-replace
// receipt commit and before the first side effect. The parent then
// proves creating was never entered before the durable receipt (the
// pending receipt survived, the completion did not), the identical
// retry refuses uncertain while the observation is unknown instead of
// replaying or inventing absence, and the same-key retry resumes under
// the same receipt — exactly one result and one receipt — once a
// successful status read proves the source.
const (
	crashChildEnv   = "AX_TERMINSTANCE_CRASH_CHILD"
	crashChildStore = "AX_TERMINSTANCE_CRASH_STORE"
	crashChildOp    = "AX_TERMINSTANCE_CRASH_OP"
)

func TestExecuteCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildEnv) == "1" {
		executeCrashChild(t)
		return
	}
	storeDir := t.TempDir()
	store, err := OpenReceiptStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	_ = store
	runCrashChild(t, storeDir, "quiesce-input", "^TestExecuteCrashChildSelfTerminates$")

	reopened, err := OpenReceiptStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	context := testContext(t)
	receipt, found, err := reopened.Lookup(context.IdempotencyKey)
	if err != nil {
		t.Fatalf("Lookup() after kill error = %v", err)
	}
	if !found {
		t.Fatal("Lookup() after kill reports absence, want the committed receipt")
	}
	if receipt.Operation != terminalbackend.OperationQuiesceInput || receipt.OperationID != fixtureOperation {
		t.Fatalf("Lookup() after kill = %+v, want the quiesce receipt", receipt)
	}
	if _, found, err := reopened.Completed(context.IdempotencyKey); err != nil || found {
		t.Fatalf("Completed() after kill = (%v, %v), want no completion", found, err)
	}

	backend := newMockBackend()
	engine := &Engine{
		Store:             reopened,
		Backend:           backend,
		CurrentLease:      func() LeaseView { return testLease() },
		CurrentGeneration: func() string { return fixtureGen },
		Now:               fixtureNow,
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, testParams(), mustParseState(t, "active"), testAdmitted("input_quiescence"))
	requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on the uncertain retry", backend.performed())
	}

	backend.observation = matchingObservation()
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() after kill error = %v", err)
	}
	requireLiteral(t, "recovered state", string(report.State), "active")

	// The same-key retry after the status proof resumes under the same
	// receipt: one effect, one recorded result, one receipt.
	resumed, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, testParams(), mustParseState(t, "active"), testAdmitted("input_quiescence"))
	if err != nil {
		t.Fatalf("ExecuteMutating() same-key retry after kill+status error = %v", err)
	}
	requireLiteral(t, "resumed after", string(resumed.After), "quiescing")
	requireLiteral(t, "resumed disposition", string(resumed.Disposition), "replay_same")
	performed := backend.performed()
	if len(performed) != 1 || performed[0] != terminalbackend.EffectInputClosed {
		t.Errorf("performed = %v, want exactly the one resumed effect", performed)
	}
	if _, found, err := reopened.Completed(context.IdempotencyKey); err != nil || !found {
		t.Errorf("Completed(key) after resume = (%v, %v), want the one recorded result", found, err)
	}
	if got := countExportedReceipts(t, reopened); got != 1 {
		t.Errorf("exported receipts = %d, want exactly one (no second receipt)", got)
	}
}

// TestExecuteCrashCreateResumesAfterStatus proves the create row's
// crash recovery with a real SIGKILL at the receipt seam: the create
// receipt survives without its completion, the identical retry refuses
// uncertain before any status proof, status proves absence (the only
// proof of absence), and the identical retry then completes with
// exactly one result, one instance's effects, and one receipt.
func TestExecuteCrashCreateResumesAfterStatus(t *testing.T) {
	if os.Getenv(crashChildEnv) == "1" {
		executeCrashChild(t)
		return
	}
	storeDir := t.TempDir()
	if _, err := OpenReceiptStore(storeDir); err != nil {
		t.Fatal(err)
	}
	runCrashChild(t, storeDir, "create", "^TestExecuteCrashCreateResumesAfterStatus$")

	reopened, err := OpenReceiptStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	create := testCreateContext(t)
	receipt, found, err := reopened.Lookup(create.context.IdempotencyKey)
	if err != nil {
		t.Fatalf("Lookup() after kill error = %v", err)
	}
	if !found {
		t.Fatal("Lookup() after kill reports absence, want the committed create receipt")
	}
	if receipt.Operation != terminalbackend.OperationCreate || receipt.OperationID != fixtureOperation {
		t.Fatalf("Lookup() after kill = %+v, want the create receipt", receipt)
	}
	if _, found, err := reopened.Completed(create.context.IdempotencyKey); err != nil || found {
		t.Fatalf("Completed() after kill = (%v, %v), want no completion", found, err)
	}

	backend := newMockBackend()
	engine := &Engine{
		Store:             reopened,
		Backend:           backend,
		CurrentLease:      func() LeaseView { return testLease() },
		CurrentGeneration: func() string { return fixtureGen },
		Now:               fixtureNow,
	}
	source := mustParseState(t, "absent")
	admitted := testAdmitted("terminal_state_retention")
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, source, admitted)
	requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
	requireLiteral(t, "after", string(result.After), "unavailable")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on the uncertain retry", backend.performed())
	}

	backend.observation = StatusObservation{
		State: terminalbackend.StateAbsent, IdentityMatch: false,
		SessionID: fixtureSessionA, InstanceID: fixtureInstance,
		BackendID: fixtureBackend, ImplVersion: fixtureImpl,
		ProtoVersion: fixtureProto, Generation: "generation-two",
	}
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), source, testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() after kill error = %v", err)
	}
	requireLiteral(t, "recovered state", string(report.State), "absent")

	resumed, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() identical create retry after kill+status error = %v", err)
	}
	requireLiteral(t, "resumed after", string(resumed.After), "active")
	performed := backend.performed()
	if len(performed) != 2 || performed[0] != terminalbackend.EffectBindingPersisted || performed[1] != terminalbackend.EffectWrapperStarted {
		t.Errorf("performed = %v, want exactly one instance's [binding_persisted wrapper_started]", performed)
	}
	if _, found, err := reopened.Completed(create.context.IdempotencyKey); err != nil || !found {
		t.Errorf("Completed(key) after resume = (%v, %v), want the one recorded result", found, err)
	}
	if got := countExportedReceipts(t, reopened); got != 1 {
		t.Errorf("exported receipts = %d, want exactly one (no second receipt)", got)
	}
}

// runCrashChild runs the genuine production entry in a child process
// that SIGKILLs itself at the receipt seam, and proves the kill.
func runCrashChild(t *testing.T, storeDir, operation, runFilter string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run="+runFilter,
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildEnv+"=1",
		crashChildStore+"="+storeDir,
		crashChildOp+"="+operation,
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
}

func executeCrashChild(t *testing.T) {
	t.Helper()
	store, err := OpenReceiptStore(os.Getenv(crashChildStore))
	if err != nil {
		t.Fatalf("child OpenReceiptStore() error = %v", err)
	}
	store.WithHooks(&StoreHooks{
		AfterCommit: func(string) error {
			proc, err := os.FindProcess(os.Getpid())
			if err != nil {
				t.Fatalf("child find self: %v", err)
			}
			_ = proc.Signal(syscall.SIGKILL)
			select {}
		},
	})
	backend := newMockBackend()
	engine := &Engine{
		Store:             store,
		Backend:           backend,
		CurrentLease:      func() LeaseView { return testLease() },
		CurrentGeneration: func() string { return fixtureGen },
		Now:               fixtureNow,
	}
	switch os.Getenv(crashChildOp) {
	case "quiesce-input":
		_, _ = engine.ExecuteMutating(context.Background(), terminalbackend.OperationQuiesceInput, testContext(t), testParams(), mustParseState(t, "active"), testAdmitted("input_quiescence"))
	case "create":
		create := testCreateContext(t)
		_, _ = engine.ExecuteMutating(context.Background(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
	default:
		t.Fatalf("child operation = %q, want quiesce-input or create", os.Getenv(crashChildOp))
	}
	t.Fatal("child ExecuteMutating() returned past SIGKILL")
}
