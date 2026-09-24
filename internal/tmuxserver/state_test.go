package tmuxserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func TestInstanceStatesRoundTrip(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []terminalbackend.InstanceState{
		terminalbackend.StateAbsent,
		terminalbackend.StateParked,
		terminalbackend.StateActive,
		terminalbackend.StateQuiescing,
		terminalbackend.StateStopped,
		terminalbackend.StateStaleFenced,
		terminalbackend.StateUnavailable,
	} {
		if err := store.Record(lxInstance, state); err != nil {
			t.Fatalf("Record(%s): %v", state, err)
		}
		got, found, err := store.Lookup(lxInstance)
		if err != nil || !found {
			t.Fatalf("Lookup after Record(%s): %v, %v", state, found, err)
		}
		if got != state {
			t.Fatalf("Lookup = %s, want %s", got, state)
		}
	}
}

func TestInstanceStatesMissingReadsAbsent(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, found, err := store.Lookup(lxInstance)
	if err != nil || found {
		t.Fatalf("Lookup unrecorded = %v, %v", found, err)
	}
}

func TestInstanceStatesRefusesCreating(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Record(lxInstance, terminalbackend.StateCreating); err == nil {
		t.Fatal("creating recorded")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance state value")
	}
}

func TestInstanceStatesRefusesBadIdentity(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Record("not-a-uuid", terminalbackend.StateActive); err == nil {
		t.Fatal("bad identity recorded")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance state identity")
	}
	if _, _, err := store.Lookup("not-a-uuid"); err == nil {
		t.Fatal("bad identity lookup admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance state identity")
	}
}

func TestInstanceStatesRefusesUnknownState(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Record(lxInstance, "launched"); err == nil {
		t.Fatal("unknown state recorded")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance state value")
	}
}

func TestInstanceStatesRefusesForeignDocument(t *testing.T) {
	root := t.TempDir()
	store, err := OpenInstanceStates(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Record(lxInstance, terminalbackend.StateActive); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tmuxlifecycle", "instances", lxInstance+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	forged := make([]byte, len(raw))
	copy(forged, raw)
	// Rewrite the instance member to another UUID: same length, valid
	// JSON, wrong identity.
	idx := -1
	for i := 0; i+len(lxInstance) <= len(forged); i++ {
		if string(forged[i:i+len(lxInstance)]) == lxInstance {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("fixture instance not found in document")
	}
	copy(forged[idx:], lxSession)
	if err := os.WriteFile(path, forged, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Lookup(lxInstance); err == nil {
		t.Fatal("foreign document adopted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "instance state identity")
	}
}

func TestInstanceStatesPreCommitCrashKeepsOldState(t *testing.T) {
	root := t.TempDir()
	store, err := OpenInstanceStates(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Record(lxInstance, terminalbackend.StateParked); err != nil {
		t.Fatal(err)
	}
	crashed := errors.New("crash between stage and install")
	store.WithHooks(&StateHooks{AfterStage: func(string) error { return crashed }})
	if err := store.Record(lxInstance, terminalbackend.StateActive); err != crashed {
		t.Fatalf("staged record = %v, want the crash", err)
	}
	store.WithHooks(nil)
	got, found, err := store.Lookup(lxInstance)
	if err != nil || !found || got != terminalbackend.StateParked {
		t.Fatalf("after pre-commit crash: %v, %v, %v", got, found, err)
	}
	// Retry converges to the new state and sweeps the staging file.
	if err := store.Record(lxInstance, terminalbackend.StateActive); err != nil {
		t.Fatal(err)
	}
	got, found, err = store.Lookup(lxInstance)
	if err != nil || !found || got != terminalbackend.StateActive {
		t.Fatalf("after retry: %v, %v, %v", got, found, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "tmuxlifecycle", "instances"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-4:] == ".tmp" {
			t.Fatalf("staging file survived: %s", entry.Name())
		}
	}
}

func TestInstanceStatesPostCommitCrashStandsCommitted(t *testing.T) {
	store, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	crashed := errors.New("crash between install and sync")
	store.WithHooks(&StateHooks{AfterInstall: func(string) error { return crashed }})
	if err := store.Record(lxInstance, terminalbackend.StateStopped); err != crashed {
		t.Fatalf("committed record = %v, want the crash", err)
	}
	store.WithHooks(nil)
	got, found, err := store.Lookup(lxInstance)
	if err != nil || !found || got != terminalbackend.StateStopped {
		t.Fatalf("after post-commit crash: %v, %v, %v", got, found, err)
	}
}

func TestOpenInstanceStatesRefusesEmptyRoot(t *testing.T) {
	if _, err := OpenInstanceStates(""); err == nil {
		t.Fatal("empty root admitted")
	}
}
