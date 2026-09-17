package hosttrust

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The hold is unforgeable outside the lock acquisition path: a forged,
// absent or released token refuses before anything is written. These tests
// pin the fail-closed boundary the hold-require-skip narrowing plant
// attacks (it admits exactly the zero token).

func TestCommitDocumentRefusesZeroHold(t *testing.T) {
	store := setupStoreForTest(t)
	target := filepath.Join(store.root, trustFileName)
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	document, err := EncodeTrust(TrustStore{Generation: 99})
	if err != nil {
		t.Fatal(err)
	}
	if err := commitDocument(HeldExclusive{}, store.fs, store.root, target, document); !errors.Is(err, ErrExclusiveHoldRequired) {
		t.Fatalf("commitDocument(zero hold) err = %v, want %v", err, ErrExclusiveHoldRequired)
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("commitDocument(zero hold) changed the committed document")
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry.Name()) >= len(stagePrefix) && entry.Name()[:len(stagePrefix)] == stagePrefix {
			t.Fatalf("commitDocument(zero hold) left staging file %q", entry.Name())
		}
	}
}

func TestCommitDocumentRefusesReleasedHold(t *testing.T) {
	store := setupStoreForTest(t)
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	document, err := EncodeTrust(TrustStore{Generation: 99})
	if err != nil {
		t.Fatal(err)
	}
	if err := commitDocument(hold, store.fs, store.root, filepath.Join(store.root, trustFileName), document); !errors.Is(err, ErrExclusiveHoldRequired) {
		t.Fatalf("commitDocument(released hold) err = %v, want %v", err, ErrExclusiveHoldRequired)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation == 99 {
		t.Fatal("commitDocument(released hold) changed the committed generation")
	}
}

// A generic live hold carries the exact Store resource but no configuration
// pairing. Config writers must reject it rather than treating liveness as a
// wildcard capability for an arbitrary target path.
func TestHeldExclusiveRejectsUnboundConfigPath(t *testing.T) {
	store := setupStoreForTest(t)
	paths := resolvedPathsForTest(t, filepath.Join(storeStateDirForTest(store), "config.toml"), storeStateDirForTest(store))
	var validationErr error
	if err := store.WithExclusiveHold(func(hold HeldExclusive) error {
		validationErr = hold.ValidateConfigPaths(paths)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(validationErr, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("ValidateConfigPath(unbound) err = %v, want %v", validationErr, ErrExclusiveHoldResourceMismatch)
	}
}

func TestHeldExclusiveRejectsForeignConfigStateRoot(t *testing.T) {
	store := setupStoreForTest(t)
	stateDir := storeStateDirForTest(store)
	configPath := filepath.Join(stateDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("configuration\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := resolvedPathsForTest(t, configPath, stateDir)
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatal(err)
	}
	var validationErr error
	foreignState := t.TempDir()
	if err := store.WithExclusiveHold(func(hold HeldExclusive) error {
		validationErr = hold.ValidateConfigPaths(resolvedPathsForTest(t, configPath, foreignState))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(validationErr, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("ValidateConfigPaths(foreign state) err = %v, want %v", validationErr, ErrExclusiveHoldResourceMismatch)
	}
}

// A generic Store must not mint a configuration-bound capability from a
// caller-supplied absolute path. The narrowing hold-resource-skip plant
// removes exactly this Store-owned binding check while retaining liveness and
// the configured-store path comparison.
func TestEnsureConfigBindingRefusesUnconfiguredStorePathThroughGenericHold(t *testing.T) {
	store := setupStoreForTest(t)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("configuration\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := store.WithExclusiveHold(func(hold HeldExclusive) error {
		return hold.ValidateConfigPaths(resolvedPathsForTest(t, configPath, storeStateDirForTest(store)))
	})
	if !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("generic hold(unconfigured store) err = %v, want %v", err, ErrExclusiveHoldResourceMismatch)
	}
}

func TestEnsureConfigBindingRefusesDifferentPathOnConfiguredStore(t *testing.T) {
	directory := t.TempDir()
	stateDir := filepath.Join(directory, "state")
	first := filepath.Join(directory, "first.toml")
	second := filepath.Join(directory, "second.toml")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("configuration\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	paths := resolvedPathsForTest(t, first, stateDir)
	store, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	err = store.WithExclusiveHold(func(hold HeldExclusive) error {
		return hold.ValidateConfigPaths(resolvedPathsForTest(t, second, stateDir))
	})
	if !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("generic hold(different configured path) err = %v, want %v", err, ErrExclusiveHoldResourceMismatch)
	}
}

func TestConfigBindingSurvivesReopenAndRejectsForeignRoot(t *testing.T) {
	directory := t.TempDir()
	stateDir := filepath.Join(directory, "state")
	configPath := filepath.Join(directory, "config.toml")
	foreignPath := filepath.Join(directory, "foreign.toml")
	for _, path := range []string{configPath, foreignPath} {
		if err := os.WriteFile(path, []byte("configuration\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	paths := resolvedPathsForTest(t, configPath, stateDir)
	store, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.WithExclusiveHold(func(hold HeldExclusive) error {
		if err := hold.ValidateConfigPaths(resolvedPathsForTest(t, foreignPath, stateDir)); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
			return errors.Join(errors.New("foreign path admitted"), err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveMarkerLockedRefusesZeroHold(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := removeMarkerLocked(HeldExclusive{}, store.fs, store.root); !errors.Is(err, ErrExclusiveHoldRequired) {
		t.Fatalf("removeMarkerLocked(zero hold) err = %v, want %v", err, ErrExclusiveHoldRequired)
	}
	assertMarkerKept(t, store)
}

// WithExclusiveHold must actually serialize: a transaction racing a held
// hold waits it out instead of landing beside it. The hold-no-lock
// narrowing plant runs the boundary with a genuine token but no lock; it
// is killed here because the racing revocation then completes early.
func TestWithExclusiveHoldSerializesWithTransaction(t *testing.T) {
	store := setupStoreForTest(t)
	credential, err := store.Issue(testHostA, testNow)
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- store.WithExclusiveHold(func(HeldExclusive) error {
			close(entered)
			<-release
			return nil
		})
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("hold boundary did not start")
	}
	revoked := make(chan error, 1)
	go func() { revoked <- store.Revoke(credential) }()
	select {
	case err := <-revoked:
		t.Fatalf("revocation completed while the exclusive hold was live: err=%v", err)
	case <-time.After(250 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-revoked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("revocation did not complete after the hold released")
	}
}

// WithExclusiveHold converges leftover joint intent before running its
// boundary, so a legacy migration coordinated through it never builds
// beside an unresolved marker.
func TestWithExclusiveHoldConvergesLeftover(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	called := false
	if err := store.WithExclusiveHold(func(HeldExclusive) error {
		called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("WithExclusiveHold did not run its boundary")
	}
	assertMarkerGone(t, store)
}

// resolveFailedReplace must not discard intent when the replacement is
// durable despite the reported failure: the marker is kept (the restore
// fails here) and a surviving reader converges forward instead of
// admitting the replacement at the old generation. The
// resolve-source-skip narrowing plant treats durable replacement as
// intact; it is killed here because the marker is then gone and the
// surviving read admits the old generation.
func TestResolveFailedReplaceKeepsMarker(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	restoreErr := errors.New("injected restore failure")
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
				return err
			}
			return errors.New("injected replace failure after durable rename")
		},
		func(HeldExclusive) error { return restoreErr })
	if !errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit err = %v, want %v", err, ErrJointCompensationFailed)
	}
	assertMarkerKept(t, store)
	converged, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("surviving ReadSnapshot err = %v, want forward convergence", err)
	}
	if converged.Generation != snapshot.Generation+1 {
		t.Fatalf("surviving generation = %d, want %d", converged.Generation, snapshot.Generation+1)
	}
	assertMarkerGone(t, store)
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(replacement) {
		t.Fatal("converged pair does not carry the durable replacement")
	}
}
