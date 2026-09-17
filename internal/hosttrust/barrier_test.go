package hosttrust

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// stageJointMarker writes one outstanding intent marker exactly as a crash
// between the configuration replacement and the generation bump leaves it.
func stageJointMarker(t *testing.T, store *Store, marker PendingCommit) {
	t.Helper()
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
}

func assertMarkerGone(t *testing.T, store *Store) {
	t.Helper()
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains: %v", err)
	}
}

func assertMarkerKept(t *testing.T, store *Store) {
	t.Helper()
	if _, err := os.Lstat(pendingPath(store.root)); err != nil {
		t.Fatalf("joint marker was removed: %v", err)
	}
}

// A surviving store (one opened before the crash) must converge pending
// intent on read, exactly like a reopened store: the replaced configuration
// is never exposed beside the old generation.
func TestReadSnapshotConvergesInterruptedApply(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	converged, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot error = %v", err)
	}
	if converged.Generation != snapshot.Generation+1 {
		t.Fatalf("ReadSnapshot generation = %d, want %d", converged.Generation, snapshot.Generation+1)
	}
	assertMarkerGone(t, store)
}

// An unreplaced marker aborts on read with no generation change.
func TestReadSnapshotAbortsUnreplacedCommit(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	converged, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot error = %v", err)
	}
	if converged.Generation != snapshot.Generation {
		t.Fatalf("ReadSnapshot generation = %d, want %d", converged.Generation, snapshot.Generation)
	}
	assertMarkerGone(t, store)
}

// A marker the configuration no longer matches refuses the read and keeps
// the marker for operator resolution.
func TestReadSnapshotRefusesIntervention(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if refused, err := store.ReadSnapshot(); err == nil {
		t.Fatalf("ReadSnapshot(intervened) admitted generation %d, want refusal", refused.Generation)
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("ReadSnapshot error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, store)
}

// WithSharedSnapshot converges before running fn, so the trust fn observes
// is the converged generation, never the superseded one.
func TestSharedSnapshotConvergesInterruptedApply(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	var observed uint64
	if err := store.WithSharedSnapshot(func(current TrustStore) error {
		observed = current.Generation
		return nil
	}); err != nil {
		t.Fatalf("WithSharedSnapshot error = %v", err)
	}
	if observed != snapshot.Generation+1 {
		t.Fatalf("WithSharedSnapshot generation = %d, want %d", observed, snapshot.Generation+1)
	}
	assertMarkerGone(t, store)
}

// Ordinary trust transactions converge pending intent before applying: the
// interrupted apply completes first (source+1), then the transaction
// commits on top (source+2).
func TestTransactionConvergesPendingMarker(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.transact(func(*TrustStore) error { return nil }); err != nil {
		t.Fatalf("transact error = %v", err)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != snapshot.Generation+2 {
		t.Fatalf("transacted generation = %d, want %d", fresh.Generation, snapshot.Generation+2)
	}
	assertMarkerGone(t, store)
}

// A joint commit converging a completed leftover refuses its own stale
// pins: convergence moved the generation, so the backstop fires before
// anything is staged.
func TestJointCommitConvergesLeftoverThenRefusesStale(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	if _, err := store.JointCommit(store.paths, jointMarker(configPath, source, replacement, snapshot.Generation),
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			called = true
			return nil
		},
		unexpectedRestore); err == nil || called {
		t.Fatalf("JointCommit(stale pins) admitted: replace=%v", called)
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("JointCommit error = %v, want ErrStaleGeneration", err)
	}
	assertMarkerGone(t, store)
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != snapshot.Generation+1 {
		t.Fatalf("converged generation = %d, want %d", fresh.Generation, snapshot.Generation+1)
	}
}

// A joint commit converging an aborted leftover proceeds: the unreplaced
// marker aborts with no generation change, then the new commit stages and
// lands normally.
func TestJointCommitConvergesAbortedLeftoverThenCommits(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	generation, err := store.JointCommit(store.paths, jointMarker(configPath, source, replacement, snapshot.Generation),
		func(current TrustStore) error {
			if current.Generation != snapshot.Generation {
				return errors.New("generation moved")
			}
			return nil
		},
		func(HeldExclusive) error { return os.WriteFile(configPath, replacement, 0o600) },
		unexpectedRestore)
	if err != nil {
		t.Fatalf("JointCommit error = %v", err)
	}
	if generation != snapshot.Generation+1 {
		t.Fatalf("JointCommit generation = %d, want %d", generation, snapshot.Generation+1)
	}
	assertMarkerGone(t, store)
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(replacement) {
		t.Fatal("JointCommit did not install the replacement")
	}
}

// A mutation request bound before a crash refuses stale after convergence:
// the binding compares against the recovered generation, and the boundary
// never runs beside unresolved intent. A request rebound after convergence
// is admitted.
func TestMutationAuthorizationRefusesStaleAfterConvergence(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leaf, root := peerMaterialForTest(t, peer, peerCredential)
	before := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leaf, root, []string{testHostA, testHostB}, testNow)
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	configPath := local.resolvedConfigPath()
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stageJointMarker(t, local, jointMarker(configPath, source, replacement, localSnap.Generation))
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := local.WithMutationAuthorization(before, func() error {
		called = true
		return nil
	}); err == nil || called {
		t.Fatalf("WithMutationAuthorization(pre-crash) admitted: err=nil boundary=%v", called)
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("WithMutationAuthorization error = %v, want ErrStaleGeneration", err)
	}
	assertMarkerGone(t, local)
	fresh, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != localSnap.Generation+1 {
		t.Fatalf("converged generation = %d, want %d", fresh.Generation, localSnap.Generation+1)
	}
	rebound := dispatchRequest(fresh, localCredential, peerCredential, testHostB, leaf, root, []string{testHostA, testHostB}, testNow)
	admitted := false
	if err := local.WithMutationAuthorization(rebound, func() error {
		admitted = true
		return nil
	}); err != nil {
		t.Fatalf("WithMutationAuthorization(rebound) error = %v", err)
	}
	if !admitted {
		t.Fatal("WithMutationAuthorization(rebound) did not run the boundary")
	}
}

// An unreadable marker path refuses the read: absence must be established,
// never assumed from a failed inspection.
func TestReadSnapshotRefusesUnreadableMarker(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, _, _ := jointFixture(t, source, replacement)
	shadow := &Store{root: store.root, fs: unreadableMarkerFS{FileSystem: store.fs}, lock: store.lock}
	if refused, err := shadow.ReadSnapshot(); err == nil {
		t.Fatalf("ReadSnapshot(unreadable marker) admitted generation %d, want refusal", refused.Generation)
	} else if !errors.Is(err, ErrTrustStoreUnreadable) {
		t.Fatalf("ReadSnapshot error = %v, want ErrTrustStoreUnreadable", err)
	}
}

type unreadableMarkerFS struct {
	FileSystem
}

func (filesystem unreadableMarkerFS) Lstat(path string) (fs.FileInfo, error) {
	if filepath.Base(path) == pendingCommitFileName {
		return nil, errors.New("joint marker inspection unavailable")
	}
	return filesystem.FileSystem.Lstat(path)
}

// A transaction beside an intervened marker refuses with the marker kept:
// no trust commit may land while joint intent is unresolvable.
func TestTransactionRefusesIntervention(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.transact(func(*TrustStore) error { return nil }); err == nil {
		t.Fatal("transact(intervened) succeeded, want refusal")
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("transact error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, store)
}

// A joint commit beside an intervened leftover refuses before staging: the
// new commit never runs while prior intent is unresolvable.
func TestJointCommitRefusesIntervenedLeftover(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, snapshot.Generation))
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	if _, err := store.JointCommit(store.paths, jointMarker(configPath, source, replacement, snapshot.Generation),
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			called = true
			return nil
		},
		unexpectedRestore); err == nil || called {
		t.Fatalf("JointCommit(intervened leftover) admitted: replace=%v", called)
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("JointCommit error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, store)
}

// Setup beside an intervened marker refuses: initialization never builds
// generation 1 while joint intent is unresolvable.
func TestInitializeRefusesIntervention(t *testing.T) {
	store := openForTest(t, t.TempDir())
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	configPath := store.resolvedConfigPath()
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stageJointMarker(t, store, jointMarker(configPath, source, replacement, 1))
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Initialize(); err == nil {
		t.Fatal("Initialize(intervened) succeeded, want refusal")
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("Initialize error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, store)
}

// A mutation boundary beside an intervened marker refuses without running:
// authorization never proceeds while joint intent is unresolvable.
func TestMutationAuthorizationRefusesIntervention(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leaf, root := peerMaterialForTest(t, peer, peerCredential)
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leaf, root, []string{testHostA, testHostB}, testNow)
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	configPath := local.resolvedConfigPath()
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stageJointMarker(t, local, jointMarker(configPath, source, replacement, localSnap.Generation))
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := local.WithMutationAuthorization(request, func() error {
		called = true
		return nil
	}); err == nil || called {
		t.Fatalf("WithMutationAuthorization(intervened) admitted: boundary=%v", called)
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("WithMutationAuthorization error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, local)
}

// A dispatch beside an intervened marker refuses: the currency check never
// runs against unresolvable trust.
func TestAuthorizeDispatchRefusesIntervention(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leaf, root := peerMaterialForTest(t, peer, peerCredential)
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leaf, root, []string{testHostA, testHostB}, testNow)
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	configPath := local.resolvedConfigPath()
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stageJointMarker(t, local, jointMarker(configPath, source, replacement, localSnap.Generation))
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := local.AuthorizeDispatch(localSnap, request); err == nil {
		t.Fatal("AuthorizeDispatch(intervened) succeeded, want refusal")
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("AuthorizeDispatch error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, local)
}

// A dispatch bound before a crash refuses stale after convergence instead
// of admitting superseded trust.
func TestAuthorizeDispatchStaleAfterConvergence(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leaf, root := peerMaterialForTest(t, peer, peerCredential)
	before := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leaf, root, []string{testHostA, testHostB}, testNow)
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	configPath := local.resolvedConfigPath()
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stageJointMarker(t, local, jointMarker(configPath, source, replacement, localSnap.Generation))
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := local.AuthorizeDispatch(localSnap, before); err == nil {
		t.Fatal("AuthorizeDispatch(pre-crash) admitted, want stale refusal")
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("AuthorizeDispatch error = %v, want ErrStaleGeneration", err)
	}
	assertMarkerGone(t, local)
}
