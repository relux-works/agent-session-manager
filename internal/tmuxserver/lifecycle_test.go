package tmuxserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// lifecycleShell builds a Lifecycle with temp stores for portable
// dispatch tests: custody never passes here, so only pre-custody arms
// are reachable (dependencies, vocabulary, scope, length).
func lifecycleShell(t *testing.T) *Lifecycle {
	t.Helper()
	receipts, err := terminstance.OpenReceiptStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := testAxpaneStore(t)
	if err != nil {
		t.Fatal(err)
	}
	attach, err := testAttachStore(t)
	if err != nil {
		t.Fatal(err)
	}
	states, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Short shared root (never the test-temp dir: its socket path
	// exceeds sun_path on macOS). The leaf stays absent by design.
	root := filepath.Join(os.TempDir(), "axshell")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return &Lifecycle{
		RuntimeDir: root + "/tmux",
		Root:       root,
		Platform:   scalar.PlatformMacOS,
		Runner:     newFakeRunner(),
		Receipts:   receipts,
		Bindings:   bindings,
		Attach:     attach,
		States:     states,
		CurrentLease: func() terminstance.LeaseView {
			return terminstance.LeaseView{LeaseID: lxLease, Epoch: 7}
		},
		CurrentGeneration: func() string { return lxGeneration },
		Now:               lxNow,
	}
}

func TestExecuteRefusesUnknownOperation(t *testing.T) {
	lc := lifecycleShell(t)
	if _, err := lc.Execute(context.Background(), OpRequest{Operation: "suspend"}); err == nil {
		t.Fatal("unknown operation admitted")
	} else {
		requireLandedCode(t, err, "terminal_backend_protocol_error")
	}
}

func TestExecuteRefusesNonLifecycleScope(t *testing.T) {
	lc := lifecycleShell(t)
	for _, operation := range []string{"manifest", "probe"} {
		if _, err := lc.Execute(context.Background(), OpRequest{Operation: operation}); err == nil {
			t.Fatalf("%s admitted to lifecycle", operation)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "lifecycle operation")
		}
	}
}

func TestExecuteRefusesMissingDependencies(t *testing.T) {
	lc := lifecycleShell(t)
	lc.Runner = nil
	if _, err := lc.Execute(context.Background(), OpRequest{Operation: "status"}); err == nil {
		t.Fatal("nil runner admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "lifecycle dependencies")
	}
	var nilLifecycle *Lifecycle
	if _, err := nilLifecycle.Execute(context.Background(), OpRequest{}); err == nil {
		t.Fatal("nil lifecycle admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "lifecycle dependencies")
	}
}

func TestExecuteRefusesLongSocket(t *testing.T) {
	lc := lifecycleShell(t)
	lc.RuntimeDir = "/root/" + strings.Repeat("d", 200)
	if _, err := lc.Execute(context.Background(), OpRequest{Operation: "status", Body: lxStatusBody(t, true, false, nil)}); err == nil {
		t.Fatal("overlong socket admitted")
	} else {
		requireLocalCode(t, err, "tmux_socket_path_too_long", "socket path length")
	}
}

func TestExecuteRefusesAtLimitSocket(t *testing.T) {
	// The length pre-gate is strict-below on every operation: a socket
	// at exactly the platform limit refuses before custody, which
	// would refuse the same requests with the absence verdict.
	for _, operation := range []string{"create", "attach", "status", "quiesce-input", "wait-safe-boundary", "request-stop", "terminate-stale", "restore"} {
		t.Run(operation, func(t *testing.T) {
			lc := lifecycleShell(t)
			lc.RuntimeDir = "/root/" + strings.Repeat("d", 90)
			if _, err := lc.Execute(context.Background(), OpRequest{Operation: operation}); err == nil {
				t.Fatalf("%s admitted the at-limit socket", operation)
			} else {
				requireLocalCode(t, err, "tmux_socket_path_too_long", "socket path length")
			}
		})
	}
}

func TestExecuteRefusesUnsafeSocket(t *testing.T) {
	// No leaf staged: the delegated leaf custody refuses, propagated —
	// identically on every operation, because custody precedes dispatch
	// and never sees the body.
	for _, operation := range []string{"create", "attach", "status", "quiesce-input", "wait-safe-boundary", "request-stop", "terminate-stale", "restore"} {
		t.Run(operation, func(t *testing.T) {
			lc := lifecycleShell(t)
			if _, err := lc.Execute(context.Background(), OpRequest{Operation: operation}); err == nil {
				t.Fatalf("%s admitted the unsafe socket", operation)
			} else {
				requireLocalCode(t, err, "tmux_unsafe_runtime_dir", "runtime absent")
			}
		})
	}
}

func TestRowErrorNormalizesOutsideSet(t *testing.T) {
	// In-row codes propagate with their report; outside-set codes are
	// themselves refused as the protocol error.
	inRow := rowError(terminalbackend.OperationTerminateStale,
		&terminalbackend.Error{Code: "terminal_backend_process_failed", Detail: "x"})
	var landed *terminalbackend.Error
	if !isLandedError(inRow, &landed) || landed.Code != "terminal_backend_process_failed" {
		t.Fatalf("in-row normalized: %v", inRow)
	}
	outRow := rowError(terminalbackend.OperationTerminateStale,
		&terminalbackend.Error{Code: "terminal_backend_restore_mismatch", Detail: "x"})
	requireLocalCode(t, outRow, "terminal_backend_protocol_error", "operation error vocabulary")
	if uncoded := rowError(terminalbackend.OperationAttach, errFakeTransport); uncoded != errFakeTransport {
		t.Fatalf("uncoded mapped: %v", uncoded)
	}
}

func TestRowErrorCoversEightRows(t *testing.T) {
	// rowError agrees with the landed allowed-set oracle on every row:
	// a code the row admits propagates; any other code normalizes.
	probes := []string{
		"terminal_backend_timeout", "terminal_backend_process_failed",
		"terminal_backend_restore_mismatch", "quiesce_timeout", "stop_timeout",
		"idempotency_mismatch", "terminal_backend_unavailable",
	}
	for _, operation := range []terminalbackend.Operation{
		terminalbackend.OperationCreate, terminalbackend.OperationAttach,
		terminalbackend.OperationStatus, terminalbackend.OperationQuiesceInput,
		terminalbackend.OperationWaitSafeBoundary, terminalbackend.OperationRequestStop,
		terminalbackend.OperationTerminateStale, terminalbackend.OperationRestore,
	} {
		for _, code := range probes {
			reported := &terminalbackend.Error{Code: code, Detail: "x"}
			got := rowError(operation, reported)
			allowed := terminalbackend.CheckErrorAllowed(string(operation), code) == nil
			var landed *terminalbackend.Error
			if allowed {
				if !isLandedError(got, &landed) || landed.Code != code {
					t.Fatalf("row %s normalized admitted %s: %v", operation, code, got)
				}
				continue
			}
			requireLocalCode(t, got, "terminal_backend_protocol_error", "operation error vocabulary")
		}
	}
}

func TestDescriptorsBoundedByConstruction(t *testing.T) {
	// Maximum descriptor: longest socket plus two UUIDs plus separators.
	socket := "/" + strings.Repeat("s", 106)
	bound := attachDescriptor(socket, lxInstance, lxClient)
	if len(bound) > 4096 {
		t.Fatalf("bound descriptor length %d exceeds 4096", len(bound))
	}
	if !strings.Contains(bound, socket) || !strings.Contains(bound, lxClient) {
		t.Fatalf("bound descriptor = %q", bound)
	}
	unbound := createDescriptor(socket, lxInstance)
	if len(unbound) > 4096 || strings.Contains(unbound, lxClient) {
		t.Fatalf("unbound descriptor = %q", unbound)
	}
}

func TestFormatTimestampParses(t *testing.T) {
	rendered := formatTimestamp(lxNow())
	if _, err := scalar.ParseTimestamp(rendered); err != nil {
		t.Fatalf("format %q: %v", rendered, err)
	}
	if rendered != "2026-09-01T12:00:00.000Z" {
		t.Fatalf("format = %q", rendered)
	}
	// Fractional instants keep their precision.
	precise, err := scalar.ParseTimestamp("2026-09-01T12:00:00.123456789Z")
	if err != nil {
		t.Fatal(err)
	}
	instant, err := precise.Time()
	if err != nil {
		t.Fatal(err)
	}
	if got := formatTimestamp(instant); got != "2026-09-01T12:00:00.123Z" {
		t.Fatalf("precise format = %q", got)
	}
}

func TestFormatTimestampUsesUTC(t *testing.T) {
	eastern := time.FixedZone("EST", -5*3600)
	now := time.Date(2026, 9, 1, 7, 0, 0, 0, eastern)
	if got := formatTimestamp(now); got != "2026-09-01T12:00:00.000Z" {
		t.Fatalf("utc format = %q", got)
	}
}
