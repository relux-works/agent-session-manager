package tmuxserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// fakeRunner is a scripted Runner: responses are keyed by the tmux
// subcommand (argv[3]), served FIFO per key, with every argv recorded.
// An empty script for a key is a test failure (fail-closed scripts:
// unexpected execs redden instead of returning zero values).
type fakeRunner struct {
	mu        sync.Mutex
	script    map[string][]fakeResponse
	calls     [][]string
	onRun     func(argv []string)
	transport map[string]error
}

type fakeResponse struct {
	stdout string
	stderr string
	exit   int
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{script: map[string][]fakeResponse{}, transport: map[string]error{}}
}

// queue appends one scripted response for a subcommand.
func (runner *fakeRunner) queue(subcommand, stdout string, exit int) *fakeRunner {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.script[subcommand] = append(runner.script[subcommand], fakeResponse{stdout: stdout, exit: exit})
	return runner
}

// queueFull appends one scripted response with stderr for a
// subcommand. Absence-shaped scripts (has-session, list-panes) must
// carry the tmux stderr marker their verdict needs; a bare exit 1
// without stderr is unknown, never absent.
func (runner *fakeRunner) queueFull(subcommand, stdout, stderr string, exit int) *fakeRunner {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.script[subcommand] = append(runner.script[subcommand], fakeResponse{stdout: stdout, stderr: stderr, exit: exit})
	return runner
}

// failTransport arms a transport error for a subcommand (served once,
// FIFO ahead of queued responses when set).
func (runner *fakeRunner) failTransport(subcommand string, err error) *fakeRunner {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.transport[subcommand] = err
	return runner
}

func (runner *fakeRunner) Run(_ context.Context, argv []string) (RunResult, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.calls = append(runner.calls, append([]string(nil), argv...))
	if runner.onRun != nil {
		runner.onRun(argv)
	}
	if len(argv) < 4 || argv[0] != "tmux" || argv[1] != "-S" {
		return RunResult{}, fmt.Errorf("fake runner refuses unshaped argv %q", argv)
	}
	key := argv[3]
	if err, armed := runner.transport[key]; armed {
		delete(runner.transport, key)
		return RunResult{}, err
	}
	responses := runner.script[key]
	if len(responses) == 0 {
		return RunResult{}, fmt.Errorf("fake runner has no script for %q (call %d)", key, len(runner.calls))
	}
	next := responses[0]
	runner.script[key] = responses[1:]
	return RunResult{Stdout: []byte(next.stdout), Stderr: []byte(next.stderr), ExitCode: next.exit}, nil
}

func (runner *fakeRunner) callCount() int {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	return len(runner.calls)
}

func (runner *fakeRunner) subcommands() []string {
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

// manualClock is a fake Now/Sleep pair: Sleep advances the clock
// instead of waiting, so poll loops run deterministically.
type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func newManualClock(now time.Time) *manualClock {
	return &manualClock{now: now}
}

func (clock *manualClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *manualClock) Sleep(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
}

// lxFixture is one lifecycle test setup: runtime root with a verified
// leaf, durable stores, scripted runner, manual clock, and a
// whole Lifecycle. Unix-only: custody passes only where the unix
// commit arms run.
type lxFixture struct {
	root    string
	runtime string
	socket  string
	data    string
	runner  *fakeRunner
	clock   *manualClock
	lc      *Lifecycle
}

// lxShortRoot stages a sun_path-fitting runtime root under /tmp: the
// test-temp-dir socket path exceeds the darwin limit, so lifecycle
// fixtures (which pass the length gate) never use t.TempDir directly.
// The root is canonicalized before return: custody walks the lexical
// path without resolving, so the fixture stages the resolved root a
// production caller would pass (macOS /tmp is a symlink).
func lxShortRoot(t *testing.T) string {
	t.Helper()
	short, err := os.MkdirTemp("/tmp", "axl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(short) })
	root := filepath.Join(short, "r")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func newLifecycleFixture(t *testing.T, admitted terminalbackend.Admitted) *lxFixture {
	t.Helper()
	root := lxShortRoot(t)
	runtime, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	data := t.TempDir()
	receipts, err := terminstance.OpenReceiptStore(filepath.Join(data, "receipts"))
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := axpane.Open(filepath.Join(data, "bindings"))
	if err != nil {
		t.Fatal(err)
	}
	attach, err := termbind.OpenAttachStore(filepath.Join(data, "attachstore"))
	if err != nil {
		t.Fatal(err)
	}
	states, err := OpenInstanceStates(filepath.Join(data, "states"))
	if err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	clock := newManualClock(lxNow())
	lc := &Lifecycle{
		RuntimeDir: runtime,
		Root:       root,
		Platform:   scalar.PlatformMacOS,
		Runner:     runner,
		Receipts:   receipts,
		Bindings:   bindings,
		Attach:     attach,
		States:     states,
		CurrentLease: func() terminstance.LeaseView {
			return terminstance.LeaseView{LeaseID: lxLease, Epoch: 7}
		},
		CurrentGeneration: func() string { return lxGeneration },
		Now:               clock.Now,
		Sleep:             clock.Sleep,
		PollInterval:      time.Millisecond,
		ServerAdmission: func(context.Context) (RealmAdmission, error) {
			return RealmAdmission{Admitted: admitted, RawGeneration: lxGeneration}, nil
		},
	}
	return &lxFixture{root: root, runtime: runtime, socket: SocketPath(runtime), data: data, runner: runner, clock: clock, lc: lc}
}

// admittedWith returns an Admitted set holding exactly the capabilities.
func admittedWith(capabilities ...string) terminalbackend.Admitted {
	return terminalbackend.Admitted{Capabilities: capabilities}
}

// fullLifecycleAdmitted holds every capability the eight operations
// confer or condition on.
func fullLifecycleAdmitted() terminalbackend.Admitted {
	return admittedWith(
		"headless_creation",
		"credential_capable_execution_realm",
		"local_attach",
		"remote_attach",
		"multi_attach",
		"multiple_input_clients",
		"reboot_restoration",
		"input_quiescence",
		"safe_boundary_observation",
		"provider_process_observation",
		"graceful_stop",
		"stale_process_termination",
		"terminal_state_retention",
	)
}

// errFakeTransport is the canned transport failure.
var errFakeTransport = errors.New("fake runner transport failure")

// testAxpaneStore opens a bootstrap binding store under a temp dir.
func testAxpaneStore(t *testing.T) (*axpane.Store, error) {
	t.Helper()
	return axpane.Open(t.TempDir())
}

// testAttachStore opens an attach receipt store under a temp dir.
func testAttachStore(t *testing.T) (*termbind.AttachStore, error) {
	t.Helper()
	return termbind.OpenAttachStore(t.TempDir())
}

// isLandedError reports whether err carries a landed terminalbackend
// refusal, decoding it into target.
func isLandedError(err error, target **terminalbackend.Error) bool {
	return errors.As(err, target)
}

// requireLocalCode asserts a *tmuxserver.Error with the literal code
// and detail. The literals come from the pinned spec or the committed
// package vocabulary, never from production constants.
func requireLocalCode(t *testing.T, err error, code, detail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %s at %s, got nil", code, detail)
	}
	var local *Error
	if !errors.As(err, &local) {
		t.Fatalf("want *Error %s at %s, got %T (%v)", code, detail, err, err)
	}
	if local.Code != code || local.Detail != detail {
		t.Fatalf("want %s at %s, got %s at %s", code, detail, local.Code, local.Detail)
	}
}

// requireEngineCode asserts a *terminstance.Error with the literal code.
func requireEngineCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want engine error %s, got nil", code)
	}
	var failure *terminstance.Error
	if !errors.As(err, &failure) {
		t.Fatalf("want *terminstance.Error %s, got %T (%v)", code, err, err)
	}
	if failure.Code != code {
		t.Fatalf("want engine code %s, got %s (%v)", code, failure.Code, err)
	}
}

// requireLandedCode asserts a *terminalbackend.Error with the literal code.
func requireLandedCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want landed error %s, got nil", code)
	}
	var failure *terminalbackend.Error
	if !errors.As(err, &failure) {
		t.Fatalf("want *terminalbackend.Error %s, got %T (%v)", code, err, err)
	}
	if failure.Code != code {
		t.Fatalf("want landed code %s, got %s (%v)", code, failure.Code, err)
	}
}

func argvStrings(argv []string) string { return strings.Join(argv, " ") }

// recordBinding records the bootstrap binding receipt for the
// lifecycle (session, bootstrap) pair with the given digest, plus
// the create-time binding document restore returns: a real prior
// create persists both, and a restore staged without the document
// cannot build its required result.
func recordBinding(t *testing.T, fx *lxFixture, digest string) {
	t.Helper()
	_, _, err := fx.lc.Bindings.Bind(lxSession, lxBootstrap, axpane.Binding{
		SessionID:          lxSession,
		OperationID:        lxBootstrap,
		TerminalInstanceID: lxInstance,
		BindingDigest:      digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := fx.lc.States.RecordBindingDoc(lxSession, lxBootstrap, lxBindingDoc(t, nil)); err != nil {
		t.Fatal(err)
	}
}

// foreignBackendBody rewrites every terminal_backend_id in a body to
// ax.conpty: the flat member and the nested context member alike.
func foreignBackendBody(t *testing.T, raw []byte) []byte {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	swap := func(container map[string]json.RawMessage) {
		if _, known := container["terminal_backend_id"]; known {
			container["terminal_backend_id"] = json.RawMessage(`"ax.conpty"`)
		}
	}
	swap(object)
	if context, known := object["context"]; known {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(context, &nested); err != nil {
			t.Fatal(err)
		}
		swap(nested)
		rewritten, err := json.Marshal(nested)
		if err != nil {
			t.Fatal(err)
		}
		object["context"] = rewritten
	}
	rewritten, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return rewritten
}

// assertTimestamp pins a rendered timestamp against the scalar grammar.
func assertTimestamp(t *testing.T, rendered string) {
	t.Helper()
	if _, err := scalar.ParseTimestamp(rendered); err != nil {
		t.Fatalf("timestamp %q: %v", rendered, err)
	}
}
