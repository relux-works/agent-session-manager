package axpane

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func openPaneStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store
}

func bindCandidate() Binding {
	candidate := Binding{
		SessionID:          fixtureSession,
		OperationID:        fixtureBootstrap,
		TerminalInstanceID: fixtureInstance,
		CreatedAt:          fixtureCreatedAt,
	}
	candidate.BindingDigest = BindingDigest(candidate)
	return candidate
}

func TestBindInstallsAndStatusProves(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	if _, found, err := store.Status(fixtureSession); err != nil || found {
		t.Fatalf("Status() before bind = (%v, %v), want (false, nil)", found, err)
	}
	bound, reattached, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if reattached {
		t.Fatal("Bind() first call reports reattached")
	}
	if bound.SessionID != fixtureSession || bound.OperationID != fixtureBootstrap || bound.TerminalInstanceID != fixtureInstance {
		t.Fatalf("Bind() = %+v, want the candidate pair", bound)
	}
	if bound.CreatedAt == "" {
		t.Fatal("Bind() receipt carries no timestamp")
	}
	if _, err := scalar.ParseTimestamp(bound.CreatedAt); err != nil {
		t.Fatalf("Bind() receipt timestamp is not a scalar timestamp: %v", err)
	}
	proven, found, err := store.Status(fixtureSession)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !found {
		t.Fatal("Status() after bind reports absence")
	}
	if proven != bound {
		t.Fatalf("Status() = %+v, want %+v", proven, bound)
	}
}

func TestBindIdenticalRetryReattaches(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	first, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	// A retry carrying different child facts still reattaches to the
	// one recorded child: the stored receipt is authoritative.
	retry := bindCandidate()
	retry.TerminalInstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	retry.BindingDigest = BindingDigest(retry)
	second, reattached, err := store.Bind(fixtureSession, fixtureBootstrap, retry)
	if err != nil {
		t.Fatalf("Bind() retry error = %v", err)
	}
	if !reattached {
		t.Fatal("Bind() identical retry does not report reattached")
	}
	if second != first {
		t.Fatalf("Bind() retry = %+v, want the recorded %+v", second, first)
	}
}

func TestBindChangedOperationRefusesMismatch(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	other := bindCandidate()
	other.OperationID = fixtureOtherOp
	if _, _, err := store.Bind(fixtureSession, fixtureOtherOp, other); err == nil {
		t.Fatal("Bind() changed operation succeeded, want idempotency_mismatch")
	} else {
		var backend *terminalbackend.Error
		if !errors.As(err, &backend) || backend.Code != terminalbackend.CodeIdempotencyMismatch {
			t.Fatalf("Bind() changed operation error = %v, want idempotency_mismatch", err)
		}
	}
	proven, found, err := store.Status(fixtureSession)
	if err != nil || !found {
		t.Fatalf("Status() after mismatch = (%v, %v), want the recorded binding", found, err)
	}
	if proven.OperationID != fixtureBootstrap {
		t.Fatalf("Status() operation = %q, want the first operation", proven.OperationID)
	}
}

func TestBindRefusesMalformedCandidates(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	base := bindCandidate()
	cases := []struct {
		name      string
		sessionID string
		operation string
		mutate    func(*Binding)
	}{
		{"bad session", "not-a-uuid", fixtureBootstrap, func(*Binding) {}},
		{"bad operation", fixtureSession, "not-a-uuid", func(candidate *Binding) { candidate.OperationID = "not-a-uuid" }},
		{"candidate pair mismatch", fixtureSession, fixtureBootstrap, func(candidate *Binding) { candidate.OperationID = fixtureOtherOp }},
		{"bad instance", fixtureSession, fixtureBootstrap, func(candidate *Binding) { candidate.TerminalInstanceID = "1234" }},
		{"bad digest", fixtureSession, fixtureBootstrap, func(candidate *Binding) { candidate.BindingDigest = "nope" }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			candidate := base
			testCase.mutate(&candidate)
			if _, _, err := store.Bind(testCase.sessionID, testCase.operation, candidate); err == nil {
				t.Fatalf("Bind(%q, %q) succeeded, want refusal", testCase.sessionID, testCase.operation)
			}
		})
	}
}

func TestStatusUnknownIsNotAbsent(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	// Corrupt the anchor behind the store: Status and SessionBound
	// must fail, never report absence.
	dir := store.sessionDir(fixtureSession)
	if err := os.WriteFile(filepath.Join(dir, "binding.json"), []byte(`{"schema":42}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Status(fixtureSession); err == nil || found {
		t.Fatalf("Status() over torn bytes = (%v, %v), want an error", found, err)
	}
	if _, err := store.SessionBound(fixtureSession); err == nil {
		t.Fatal("SessionBound() over torn anchor succeeded, want the read error")
	}
	// Corrupt the pair receipt too: Lookup and the identical retry
	// must fail, never report absence or attach to a second child.
	if err := os.WriteFile(store.receiptPath(fixtureSession, fixtureBootstrap), []byte(`{"schema":42}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Lookup(fixtureSession, fixtureBootstrap); err == nil || found {
		t.Fatalf("Lookup() over torn bytes = (%v, %v), want an error", found, err)
	}
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err == nil {
		t.Fatal("Bind() over torn bytes succeeded, want the read error")
	}
}

func TestBindHookAfterStageAbortsWithoutInstall(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t).WithHooks(&Hooks{
		AfterStage: func(path string) error { return errFixtureConfig },
	})
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); !errors.Is(err, errFixtureConfig) {
		t.Fatalf("Bind() hook error = %v, want the hook error", err)
	}
	if _, found, err := store.Status(fixtureSession); err != nil || found {
		t.Fatalf("Status() after aborted stage = (%v, %v), want absence", found, err)
	}
	// The identical retry converges: staging garbage never blocks it.
	store.WithHooks(nil)
	if _, reattached, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil || reattached {
		t.Fatalf("Bind() retry = (reattached %v, %v), want a fresh install", reattached, err)
	}
}

func TestBindHookAfterInstallKeepsTheCommit(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t).WithHooks(&Hooks{
		AfterInstall: func(path string) error { return errFixtureConfig },
	})
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); !errors.Is(err, errFixtureConfig) {
		t.Fatalf("Bind() hook error = %v, want the hook error", err)
	}
	// The hook runs after the no-replace commit: the binding stands
	// and the identical retry reattaches to it.
	proven, found, err := store.Status(fixtureSession)
	if err != nil || !found {
		t.Fatalf("Status() after post-commit hook = (%v, %v), want the binding", found, err)
	}
	store.WithHooks(nil)
	second, reattached, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil || !reattached {
		t.Fatalf("Bind() retry = (reattached %v, %v), want reattach", reattached, err)
	}
	if second != proven {
		t.Fatalf("Bind() retry = %+v, want %+v", second, proven)
	}
}

// TestBindConcurrentSameOperationReattaches pins the linearizable
// race: a second wrapper committing between Status and link wins,
// and the loser reattaches to the winner's receipt instead of
// failing or installing a second child. The AfterStage hook plays
// the concurrent winner deterministically.
func TestBindConcurrentSameOperationReattaches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outer, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	winner := bindCandidate()
	winner.TerminalInstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	winner.BindingDigest = BindingDigest(winner)
	outer.WithHooks(&Hooks{
		AfterStage: func(path string) error {
			inner, err := Open(root)
			if err != nil {
				return err
			}
			if _, _, err := inner.Bind(fixtureSession, fixtureBootstrap, winner); err != nil {
				return err
			}
			return nil
		},
	})
	bound, reattached, err := outer.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil {
		t.Fatalf("Bind() race loser error = %v", err)
	}
	if !reattached {
		t.Fatal("Bind() race loser installs fresh, want reattach to the winner")
	}
	if bound.TerminalInstanceID != winner.TerminalInstanceID {
		t.Fatalf("Bind() race loser = %+v, want the winner's receipt", bound)
	}
	proven, found, err := outer.Status(fixtureSession)
	if err != nil || !found || proven != bound {
		t.Fatalf("Status() = (%+v, %v, %v), want the one receipt", proven, found, err)
	}
}

// TestBindConcurrentChangedOperationRefuses pins the losing side of
// the commit race: a concurrent winner with a changed operation is
// the bootstrap-window mismatch, never a reattach.
func TestBindConcurrentChangedOperationRefuses(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outer, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	winner := bindCandidate()
	winner.OperationID = fixtureOtherOp
	outer.WithHooks(&Hooks{
		AfterStage: func(path string) error {
			inner, err := Open(root)
			if err != nil {
				return err
			}
			_, _, err = inner.Bind(fixtureSession, fixtureOtherOp, winner)
			return err
		},
	})
	if _, _, err := outer.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err == nil {
		t.Fatal("Bind() changed-op race loser reattached, want idempotency_mismatch")
	} else {
		var backend *terminalbackend.Error
		if !errors.As(err, &backend) || backend.Code != terminalbackend.CodeIdempotencyMismatch {
			t.Fatalf("Bind() changed-op race loser error = %v, want idempotency_mismatch", err)
		}
	}
}

// TestStatusSessionMismatchRefuses pins the binding-to-directory
// check: a receipt read under another session refuses, even when
// the two session IDs share a prefix.
func TestStatusSessionMismatchRefuses(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.bindingPath(fixtureSession))
	if err != nil {
		t.Fatal(err)
	}
	foreignDir := store.sessionDir(fixtureForeign)
	if err := os.MkdirAll(foreignDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreignDir, "binding.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Status(fixtureForeign); err == nil || found {
		t.Fatalf("Status(foreign) = (%v, %v), want the session-mismatch error", found, err)
	}
}

// TestBindingPersistsNoPID pins the §4.C identity rule on the stored
// bytes: the closed receipt carries the AX-allocated UUIDv7 instance
// and the opaque digest, and no PID, handle, socket, path, endpoint,
// or token member exists to mistake for identity.
func TestBindingPersistsNoPID(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	raw, err := os.ReadFile(store.bindingPath(fixtureSession))
	if err != nil {
		t.Fatal(err)
	}
	var members map[string]any
	if err := json.Unmarshal(raw, &members); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"schema": true, "schema_version": true, "session_id": true,
		"bootstrap_operation_id": true, "terminal_instance_id": true,
		"binding_digest": true, "created_at": true,
	}
	if len(members) != len(want) {
		t.Fatalf("receipt members = %v, want exactly %v", members, want)
	}
	for name := range members {
		if !want[name] {
			t.Fatalf("receipt carries unexpected member %q", name)
		}
		lowered := strings.ToLower(name)
		for _, forbidden := range []string{"pid", "handle", "socket", "endpoint", "token", "pipe", "url", "path"} {
			if strings.Contains(lowered, forbidden) {
				t.Fatalf("receipt member %q names a forbidden identity class", name)
			}
		}
	}
}
