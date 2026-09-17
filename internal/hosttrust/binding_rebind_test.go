package hosttrust

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestReviewerRefusedRebindLeavesTargetUnclaimed drives the production
// enrollment entry point with two Stores opened before either configuration is
// bound. A rejected request must not publish a target-side claim as a side
// effect of discovering that the shared store is already bound.
func TestReviewerRefusedRebindLeavesTargetUnclaimed(t *testing.T) {
	directory := t.TempDir()
	configA := filepath.Join(directory, "a.toml")
	configB := filepath.Join(directory, "b.toml")
	for _, path := range []string{configA, configB} {
		if err := os.WriteFile(path, []byte("source"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	state := filepath.Join(directory, "shared-state")
	pathsA := resolvedPathsForTest(t, configA, state)
	pathsB := resolvedPathsForTest(t, configB, state)
	storeA := openPathsForTest(t, configA, state)
	storeB := openPathsForTest(t, configB, state)
	if err := storeA.EnsureConfigBinding(pathsA); err != nil {
		t.Fatalf("EnsureConfigBinding(A/shared-state) = %v", err)
	}
	if err := storeB.EnsureConfigBinding(pathsB); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("EnsureConfigBinding(B/shared-state) = %v, want resource mismatch", err)
	}

	canonicalB, err := canonicalExistingPath(configB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(configBindingPath(canonicalB)); !os.IsNotExist(err) {
		t.Fatalf("rejected B target sidecar = %v, want absent", err)
	}
	canonicalA, err := canonicalExistingPath(configA)
	if err != nil {
		t.Fatal(err)
	}
	boundA, present, err := loadTargetConfigBinding(osFileSystem{}, canonicalA)
	if err != nil {
		t.Fatal(err)
	}
	if !present || boundA.ConfigPath != canonicalA || boundA.StateRoot != mustCanonicalPathForTest(t, state) {
		t.Fatalf("accepted A binding after rejected B = %#v (present=%v), want the original pair", boundA, present)
	}
	boundStore, present, err := loadStoreConfigBinding(osFileSystem{}, storeA.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !present || boundStore.ConfigPath != canonicalA || boundStore.StateRoot != mustCanonicalPathForTest(t, state) {
		t.Fatalf("store binding after rejected B = %#v (present=%v), want the original pair", boundStore, present)
	}

	// If the rejected request left a sidecar behind, this otherwise independent
	// state root would be incorrectly refused by the next legitimate enrollment.
	foreignState := filepath.Join(directory, "foreign-state")
	foreignPaths := resolvedPathsForTest(t, configB, foreignState)
	foreign := openPathsForTest(t, configB, foreignState)
	if err := foreign.EnsureConfigBinding(foreignPaths); err != nil {
		t.Fatalf("EnsureConfigBinding(B/foreign-state) = %v, want success after refused shared request", err)
	}
}

// TestCompetingFirstBindingsPublishExactlyOneAssociation drives the same
// target through two independently opened stores. The per-target bootstrap
// lock and the state-root check must produce one winner and one side-effect
// free refusal.
func TestCompetingFirstBindingsPublishExactlyOneAssociation(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	stateA := filepath.Join(directory, "state-a")
	stateB := filepath.Join(directory, "state-b")
	pathsA := resolvedPathsForTest(t, configPath, stateA)
	pathsB := resolvedPathsForTest(t, configPath, stateB)
	storeA := openPathsForTest(t, configPath, stateA)
	storeB := openPathsForTest(t, configPath, stateB)

	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		results <- storeA.EnsureConfigBinding(pathsA)
	}()
	go func() {
		defer wait.Done()
		<-start
		results <- storeB.EnsureConfigBinding(pathsB)
	}()
	close(start)
	wait.Wait()
	close(results)

	var successes, refusals int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrExclusiveHoldResourceMismatch):
			refusals++
		default:
			t.Fatalf("competing EnsureConfigBinding error = %v, want success or resource mismatch", err)
		}
	}
	if successes != 1 || refusals != 1 {
		t.Fatalf("competing binding outcomes = %d successes/%d refusals, want 1/1", successes, refusals)
	}

	canonical, err := canonicalExistingPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	binding, present, err := loadTargetConfigBinding(osFileSystem{}, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("competing binding requests left no target association")
	}
	if binding.StateRoot != mustCanonicalPathForTest(t, stateA) && binding.StateRoot != mustCanonicalPathForTest(t, stateB) {
		t.Fatalf("winning state root = %q, want %q or %q", binding.StateRoot, mustCanonicalPathForTest(t, stateA), mustCanonicalPathForTest(t, stateB))
	}
	for name, store := range map[string]*Store{"A": storeA, "B": storeB} {
		storeBinding, present, err := loadStoreConfigBinding(osFileSystem{}, store.Root())
		if err != nil {
			t.Fatal(err)
		}
		wantPresent := storeBinding.StateRoot == binding.StateRoot
		if present != wantPresent {
			t.Fatalf("competing store %s index presence = %v with binding %#v, want %v", name, present, storeBinding, wantPresent)
		}
		if present && (storeBinding.ConfigPath != binding.ConfigPath || storeBinding.StateRoot != binding.StateRoot) {
			t.Fatalf("competing store %s index = %#v, want winner %#v", name, storeBinding, binding)
		}
	}
}

// failStoreBindingRenameFS simulates a crash/failure after the authoritative
// target sidecar is durable but before its derived store index is published.
// The exact store-index rename is the only injected failure.
type failStoreBindingRenameFS struct {
	FileSystem
}

func (filesystem failStoreBindingRenameFS) Rename(oldname, newname string) error {
	if filepath.Base(newname) == configBindingFileName {
		return errors.New("injected store binding index rename failure")
	}
	return filesystem.FileSystem.Rename(oldname, newname)
}

func (filesystem failStoreBindingRenameFS) CreateTemp(dir, pattern string, mode fs.FileMode) (StagedFile, error) {
	return filesystem.FileSystem.CreateTemp(dir, pattern, mode)
}

// TestInterruptedFirstBindingIsRepairedAfterReopen preserves the target-side
// record across a failed derived-index commit, then reopens the store and
// drives the explicit repair path. The target is never rebound to a different
// state root, and the repair publishes only the missing index.
func TestInterruptedFirstBindingIsRepairedAfterReopen(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(directory, "state")
	paths := resolvedPathsForTest(t, configPath, state)
	store := openPathsForTest(t, configPath, state)
	store.fs = failStoreBindingRenameFS{FileSystem: store.fs}
	if err := store.EnsureConfigBinding(paths); !errors.Is(err, ErrTrustDurability) {
		t.Fatalf("EnsureConfigBinding(injected index failure) = %v, want durability error", err)
	}
	canonical, err := canonicalExistingPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(configBindingPath(canonical)); err != nil {
		t.Fatalf("target sidecar after injected index failure: %v", err)
	}
	if _, err := os.Lstat(bindingPath(store.Root())); !os.IsNotExist(err) {
		t.Fatalf("store index after injected index failure = %v, want absent", err)
	}

	reopened, err := Open(paths)
	if err != nil {
		t.Fatalf("Open after interrupted first binding = %v", err)
	}
	if err := reopened.EnsureConfigBinding(paths); err != nil {
		t.Fatalf("EnsureConfigBinding repair after reopen = %v", err)
	}
	if _, err := os.Lstat(bindingPath(reopened.Root())); err != nil {
		t.Fatalf("store index after repair: %v", err)
	}
	if err := reopened.ValidateBoundConfigPath(paths); err != nil {
		t.Fatalf("ValidateBoundConfigPath after repair = %v", err)
	}
}

func mustCanonicalPathForTest(t *testing.T, path string) string {
	t.Helper()
	canonical, err := canonicalExistingPath(path)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}
