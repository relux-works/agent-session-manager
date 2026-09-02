//go:build darwin || linux

package localstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// The clauses pinned here guard the process lock, the derived-cache state root,
// the quarantine mover and the post-rebuild file-mode gate. Each one was
// reachable but unpinned: disabling it individually left the package suite
// green, so nothing would have noticed the refusal disappearing.

// TestOpenProjectionRefusesUnsafeProcessLockAtProductionEntry drives the real
// OpenProjection entry point against a lock file that is owner-readable by the
// group and against one replaced by a FIFO. Both are refusals the shipped
// implementation makes, and both are reachable with plain on-disk fixtures, so
// neither needs a hook, a race or a direct helper call to prove.
func TestOpenProjectionRefusesUnsafeProcessLockAtProductionEntry(t *testing.T) {
	openedLockPath := func(t *testing.T) (ResolvedPaths, string) {
		t.Helper()
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		projection, _, err := OpenProjection(context.Background(), paths)
		if err != nil {
			t.Fatalf("OpenProjection() error = %v", err)
		}
		lockPath := projection.Path() + ".lock"
		if err := projection.Close(); err != nil {
			t.Fatal(err)
		}
		return paths, lockPath
	}

	t.Run("group-readable lock", func(t *testing.T) {
		paths, lockPath := openedLockPath(t)
		if err := os.Chmod(lockPath, 0o640); err != nil {
			t.Fatal(err)
		}

		_, _, err := OpenProjection(context.Background(), paths)
		if !errors.Is(err, ErrUnsafeOwnership) {
			t.Fatalf("OpenProjection(group-readable lock) error = %v, want %v", err, ErrUnsafeOwnership)
		}
		if !strings.Contains(err.Error(), "projection lock") {
			t.Fatalf("OpenProjection(group-readable lock) error = %v, want the lock ownership refusal", err)
		}
		if !strings.Contains(err.Error(), "mode is 0640, want 0600") {
			t.Fatalf("OpenProjection(group-readable lock) error = %v, want the underlying owner-only cause preserved", err)
		}
		info, err := os.Lstat(lockPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o640 {
			t.Fatalf("lock mode after refusal = %o, want the unsafe fixture left untouched", info.Mode().Perm())
		}
	})

	t.Run("non-regular lock", func(t *testing.T) {
		paths, lockPath := openedLockPath(t)
		if err := os.Remove(lockPath); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(lockPath, 0o600); err != nil {
			t.Fatal(err)
		}

		_, _, err := OpenProjection(context.Background(), paths)
		if !errors.Is(err, ErrUnsafeOwnership) {
			t.Fatalf("OpenProjection(fifo lock) error = %v, want %v", err, ErrUnsafeOwnership)
		}
		if !strings.Contains(err.Error(), "projection lock is not a regular file") {
			t.Fatalf("OpenProjection(fifo lock) error = %v, want the lock regularity refusal", err)
		}
		info, err := os.Lstat(lockPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeNamedPipe == 0 {
			t.Fatalf("lock mode after refusal = %s, want the FIFO fixture left in place", info.Mode())
		}
	})
}

// TestOpenProjectionRefusesGroupAccessibleBlobDigestRoot pins the digest-root
// owner-only clause. The existing suite covered the objects root and one shard
// directory; the digest root between them was verified by production and by
// nothing else.
func TestOpenProjectionRefusesGroupAccessibleBlobDigestRoot(t *testing.T) {
	store := openTestStore(t)
	content := []byte("digest root ownership")
	if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
	if err := os.Chmod(digestRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)

	_, _, err := OpenProjection(context.Background(), paths)
	if !errors.Is(err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(group-accessible digest root) error = %v, want %v", err, ErrProjectionSourceIntegrity)
	}
	if !errors.Is(err, ErrUnsafeOwnership) {
		t.Fatalf("OpenProjection(group-accessible digest root) error = %v, want the owner-only cause preserved", err)
	}
	state, _ := paths.Path(PathState)
	if _, err := os.Lstat(filepath.Join(state.Value.String(), projectionFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("index stat after source refusal = %v, want no index created", err)
	}
}

// TestOpenProjectionRefusesNonRegularSidecarLeftAfterRebuild pins the
// non-regular branch of the post-rebuild file-mode gate. Its owner-mode sibling
// was already covered; replacing a sidecar with a FIFO after the rebuild commit
// is the state that reaches the regularity clause through openProjection.
func TestOpenProjectionRefusesNonRegularSidecarLeftAfterRebuild(t *testing.T) {
	for _, suffix := range []string{"-wal", "-journal"} {
		t.Run("fifo sidecar "+suffix, func(t *testing.T) {
			store := openTestStore(t)
			content := []byte("fifo sidecar " + suffix)
			if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
				t.Fatal(err)
			}
			projection, _, err := openProjection(context.Background(), store.resolvedPathsForTest(t), projectionHooks{
				afterRebuildCommit: func(path string) error {
					target := path + suffix
					if err := os.RemoveAll(target); err != nil {
						return err
					}
					return syscall.Mkfifo(target, 0o600)
				},
			})
			if projection != nil {
				_ = projection.Close()
			}
			if !errors.Is(err, ErrProjection) {
				t.Fatalf("openProjection(fifo %s) error = %v, want %v", suffix, err, ErrProjection)
			}
			if !strings.Contains(err.Error(), "verify index files") {
				t.Fatalf("openProjection(fifo %s) error = %v, want the post-rebuild file gate", suffix, err)
			}
			if !strings.Contains(err.Error(), "is not a regular file") {
				t.Fatalf("openProjection(fifo %s) error = %v, want the regularity clause, not the owner-mode clause", suffix, err)
			}
			if strings.Contains(err.Error(), "after rebuild commit") {
				t.Fatalf("openProjection(fifo %s) failed inside the hook, not at the gate: %v", suffix, err)
			}
		})
	}
}

// TestQuarantineProjectionFilesRefusesUnsafeRecoverySources drives the
// quarantine mover directly. Its two clauses re-check regularity and ownership
// of files that prepareProjectionFile, verifyProjectionSidecars or
// ensureProjectionFileModes proved safe moments earlier under the held index
// lock, so no production fixture can reach them; the clauses are annotated
// projection-refusal-direct and pinned here instead.
func TestQuarantineProjectionFilesRefusesUnsafeRecoverySources(t *testing.T) {
	prepared := func(t *testing.T) (string, string) {
		t.Helper()
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		projection, _, err := OpenProjection(context.Background(), paths)
		if err != nil {
			t.Fatal(err)
		}
		if err := projection.Close(); err != nil {
			t.Fatal(err)
		}
		state, _ := paths.Path(PathState)
		return state.Value.String(), projection.Path()
	}

	t.Run("non-regular recovery source", func(t *testing.T) {
		stateRoot, indexPath := prepared(t)
		fifo := indexPath + "-journal"
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Fatal(err)
		}

		directory, err := quarantineProjectionFiles(stateRoot, indexPath)
		if !errors.Is(err, ErrUnsafeOwnership) {
			t.Fatalf("quarantineProjectionFiles(fifo journal) error = %v, want %v", err, ErrUnsafeOwnership)
		}
		if !strings.Contains(err.Error(), "index recovery source") {
			t.Fatalf("quarantineProjectionFiles(fifo journal) error = %v, want the recovery-source refusal", err)
		}
		if directory != "" {
			t.Fatalf("quarantineProjectionFiles(fifo journal) directory = %q, want no reported recovery", directory)
		}
		if _, err := os.Lstat(fifo); err != nil {
			t.Fatalf("fifo after refusal = %v, want the unsafe source left in place", err)
		}
	})

	t.Run("group-readable recovery source", func(t *testing.T) {
		stateRoot, indexPath := prepared(t)
		sidecar := indexPath + "-journal"
		if err := os.WriteFile(sidecar, []byte("stale rollback journal"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(sidecar, 0o640); err != nil {
			t.Fatal(err)
		}

		directory, err := quarantineProjectionFiles(stateRoot, indexPath)
		if !errors.Is(err, ErrUnsafeOwnership) {
			t.Fatalf("quarantineProjectionFiles(group-readable journal) error = %v, want %v", err, ErrUnsafeOwnership)
		}
		if !strings.Contains(err.Error(), "mode is 0640, want 0600") {
			t.Fatalf("quarantineProjectionFiles(group-readable journal) error = %v, want the owner-only cause preserved", err)
		}
		if directory != "" {
			t.Fatalf("quarantineProjectionFiles(group-readable journal) directory = %q, want no reported recovery", directory)
		}
	})
}

// TestPrepareProjectionFileRefusesUnsafeStateRootDirectly pins the state-root
// re-check inside prepareProjectionFile. InitializeLayout verifies the same
// directory at the top of openProjection and refuses first, so this clause is
// annotated projection-refusal-direct and driven here.
func TestPrepareProjectionFileRefusesUnsafeStateRootDirectly(t *testing.T) {
	root := t.TempDir()
	stateRoot := filepath.Join(root, "state")
	if err := os.Mkdir(stateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(stateRoot, projectionFilename)
	if err := prepareProjectionFile(indexPath); err != nil {
		t.Fatalf("prepareProjectionFile(owner-only state root) error = %v, want acceptance at the limit", err)
	}
	if err := os.Chmod(stateRoot, 0o750); err != nil {
		t.Fatal(err)
	}

	err := prepareProjectionFile(indexPath)
	if !errors.Is(err, ErrUnsafeOwnership) {
		t.Fatalf("prepareProjectionFile(group-accessible state root) error = %v, want %v", err, ErrUnsafeOwnership)
	}
	if !strings.Contains(err.Error(), "index state root") {
		t.Fatalf("prepareProjectionFile(group-accessible state root) error = %v, want the state-root refusal", err)
	}
}

// The clauses pinned below are shape clauses: they refuse a path that is not the
// kind of file it must be. Every fixture that existed for them was a directory
// at mode 0700 or a symlink at lstat permission 0777, and both are already
// refused by the verifyOwnerFileInfo(info, 0o600) call that follows, so the
// shape clause never decided anything the suite could observe. A FIFO created at
// exactly mode 0600 passes every preceding check — it is not a symlink, its
// owner is the effective user and its permission bits are already owner-only —
// so the shape clause is the only thing that can refuse it. Each test asserts
// that isolation explicitly with verifyOwnerFileInfo before driving production.
//
// Removing one of these clauses is not merely a wrong answer: os.Open and SQLite
// both block indefinitely on a FIFO with no writer, so the unpinned failure mode
// is an unbounded hang inside the production entry point. Every case therefore
// drives OpenProjection under a bound instead of calling it inline.

// projectionOpenResult carries a bounded OpenProjection outcome back to the test
// goroutine.
type projectionOpenResult struct {
	projection *Projection
	recovery   ProjectionRecovery
	err        error
}

const projectionOpenBound = 20 * time.Second

// openProjectionBounded drives the real OpenProjection entry point and fails
// instead of hanging when it does not return. fifos names any FIFO fixture whose
// blocked reader must be released so the leaked goroutine cannot outlive the
// failure.
func openProjectionBounded(t *testing.T, paths ResolvedPaths, fifos ...string) projectionOpenResult {
	t.Helper()
	done := make(chan projectionOpenResult, 1)
	go func() {
		projection, recovery, err := OpenProjection(context.Background(), paths)
		done <- projectionOpenResult{projection: projection, recovery: recovery, err: err}
	}()
	select {
	case result := <-done:
		if result.projection != nil {
			t.Cleanup(func() { _ = result.projection.Close() })
		}
		return result
	case <-time.After(projectionOpenBound):
		for _, fifo := range fifos {
			if file, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
				_ = file.Close()
			}
		}
		t.Fatalf("OpenProjection did not return within %s: the shape clause under test is the only bound on this open", projectionOpenBound)
		return projectionOpenResult{}
	}
}

// requireIsolatedShapeFixture proves the fixture reaches the shape clause under
// test rather than tripping the owner-only clause that follows it. Without this
// assertion a fixture can look like a negative case while the refusal it
// observes is decided somewhere else entirely.
func requireIsolatedShapeFixture(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().IsRegular() {
		t.Fatalf("fixture %q is a regular file, so no shape clause can refuse it", path)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("fixture %q is a symlink, whose 0777 lstat permission is refused by the owner-only clause instead", path)
	}
	if err := verifyOwnerFileInfo(info, 0o600); err != nil {
		t.Fatalf("fixture %q does not isolate the shape clause: the owner-only check refuses it first: %v", path, err)
	}
}

func mkfifoFixture(t *testing.T, path string) string {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	requireIsolatedShapeFixture(t, path)
	return path
}

func projectionIndexPathForTest(t *testing.T, paths ResolvedPaths) string {
	t.Helper()
	if err := InitializeLayout(paths); err != nil {
		t.Fatal(err)
	}
	state, ok := paths.Path(PathState)
	if !ok {
		t.Fatal("resolved paths carry no state root")
	}
	return filepath.Join(state.Value.String(), projectionFilename)
}

// TestOpenProjectionRefusesNonRegularIndexAtProductionEntry pins the
// index-is-not-a-regular-file clause in prepareProjectionFile. With the clause
// disabled the FIFO passes the owner-only check, its zero size skips the header
// verifier, and the unsafe file is handed straight to SQLite.
func TestOpenProjectionRefusesNonRegularIndexAtProductionEntry(t *testing.T) {
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	indexPath := projectionIndexPathForTest(t, paths)
	mkfifoFixture(t, indexPath)

	result := openProjectionBounded(t, paths, indexPath)
	if !errors.Is(result.err, ErrUnsafeOwnership) {
		t.Fatalf("OpenProjection(fifo index) error = %v, want %v", result.err, ErrUnsafeOwnership)
	}
	if !strings.Contains(result.err.Error(), "index is not a regular file") {
		t.Fatalf("OpenProjection(fifo index) error = %v, want the index regularity refusal", result.err)
	}
	info, err := os.Lstat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("index after refusal = %s, want the FIFO fixture left in place", info.Mode())
	}
}

// TestOpenProjectionRefusesNonRegularSidecarBeforeOpeningTheIndex pins the
// sidecar regularity clause in verifyProjectionSidecars. The existing symlink
// fixture is decided by the owner-only clause on the following line; a 0600 WAL
// FIFO reaches the shape clause and, with it disabled, is handed to SQLite as
// the write-ahead log of a freshly created index.
func TestOpenProjectionRefusesNonRegularSidecarBeforeOpeningTheIndex(t *testing.T) {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		t.Run("fifo sidecar "+suffix, func(t *testing.T) {
			store := openTestStore(t)
			paths := store.resolvedPathsForTest(t)
			indexPath := projectionIndexPathForTest(t, paths)
			sidecar := mkfifoFixture(t, indexPath+suffix)

			result := openProjectionBounded(t, paths, sidecar)
			if !errors.Is(result.err, ErrUnsafeOwnership) {
				t.Fatalf("OpenProjection(fifo %s) error = %v, want %v", suffix, result.err, ErrUnsafeOwnership)
			}
			if !strings.Contains(result.err.Error(), "projection sidecar") {
				t.Fatalf("OpenProjection(fifo %s) error = %v, want the sidecar refusal", suffix, result.err)
			}
			if !strings.Contains(result.err.Error(), "is not a regular file") {
				t.Fatalf("OpenProjection(fifo %s) error = %v, want the regularity clause, not the owner-only clause", suffix, result.err)
			}
			if _, err := os.Lstat(indexPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("index stat after sidecar refusal = %v, want no index created", err)
			}
		})
	}
}

// TestOpenProjectionRefusesNonRegularAuthoritativeBlobLeaf pins the blob
// regularity clause in scanAuthoritativeBlobs. This is the clause the reviewer
// probed as an availability defect: with it disabled, scalar.ParseDigest accepts
// the FIFO's name, validateProjectionBlobSize accepts its zero size, and
// os.Open blocks forever waiting for a writer.
func TestOpenProjectionRefusesNonRegularAuthoritativeBlobLeaf(t *testing.T) {
	store := openTestStore(t)
	digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
	shard := filepath.Join(digestRoot, "aa")
	if err := os.Mkdir(shard, 0o700); err != nil {
		t.Fatal(err)
	}
	leaf := mkfifoFixture(t, filepath.Join(shard, strings.Repeat("a", 62)))
	paths := store.resolvedPathsForTest(t)

	result := openProjectionBounded(t, paths, leaf)
	if !errors.Is(result.err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(fifo blob leaf) error = %v, want %v", result.err, ErrProjectionSourceIntegrity)
	}
	if !strings.Contains(result.err.Error(), "is not a regular file") {
		t.Fatalf("OpenProjection(fifo blob leaf) error = %v, want the blob regularity refusal", result.err)
	}
	if strings.Contains(result.err.Error(), "staged blob") {
		t.Fatalf("OpenProjection(fifo blob leaf) error = %v, want the authoritative-leaf clause, not the staged-blob clause", result.err)
	}
	state, _ := paths.Path(PathState)
	if _, err := os.Lstat(filepath.Join(state.Value.String(), projectionFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("index stat after source refusal = %v, want no index created", err)
	}
}

// TestOpenProjectionRefusesDirectoryAuthoritativeBlobLeafByItsOwnClause pins the
// directory clause that precedes the regularity clause. The existing 0700
// directory fixture is refused identically by either clause, so the refusal is
// asserted by its message: with the directory clause disabled the regularity
// clause answers instead, and the assertion reddens.
func TestOpenProjectionRefusesDirectoryAuthoritativeBlobLeafByItsOwnClause(t *testing.T) {
	store := openTestStore(t)
	digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
	shard := filepath.Join(digestRoot, "aa")
	if err := os.Mkdir(shard, 0o700); err != nil {
		t.Fatal(err)
	}
	leafName := strings.Repeat("b", 62)
	leaf := filepath.Join(shard, leafName)
	// Mode 0600 keeps the owner-only clause from deciding this case, exactly as
	// it would for a FIFO; only the leaf kind differs.
	if err := os.Mkdir(leaf, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(leaf, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(leaf)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyOwnerFileInfo(info, 0o600); err != nil {
		t.Fatalf("fixture does not isolate the directory clause: %v", err)
	}

	result := openProjectionBounded(t, store.resolvedPathsForTest(t), leaf)
	if !errors.Is(result.err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(directory blob leaf) error = %v, want %v", result.err, ErrProjectionSourceIntegrity)
	}
	if !strings.Contains(result.err.Error(), "blob leaf") || !strings.Contains(result.err.Error(), "is a directory") {
		t.Fatalf("OpenProjection(directory blob leaf) error = %v, want the directory-leaf clause rather than the regularity clause behind it", result.err)
	}
	if !strings.Contains(result.err.Error(), leafName) {
		t.Fatalf("OpenProjection(directory blob leaf) error = %v, want the refused leaf named", result.err)
	}
}

// TestOpenProjectionRefusesNonRegularStagedBlob pins the staged-blob regularity
// clause. With it disabled a 0600 FIFO passes the owner-only check and the
// unsafe stage is silently skipped, so the open succeeds and reports a clean
// scan of a source tree it never inspected.
func TestOpenProjectionRefusesNonRegularStagedBlob(t *testing.T) {
	store := openTestStore(t)
	digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
	shard := filepath.Join(digestRoot, "aa")
	if err := os.Mkdir(shard, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := mkfifoFixture(t, filepath.Join(shard, stagedFilePrefix+"fifo"))

	result := openProjectionBounded(t, store.resolvedPathsForTest(t), stage)
	if !errors.Is(result.err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(fifo staged blob) error = %v, want %v", result.err, ErrProjectionSourceIntegrity)
	}
	if !strings.Contains(result.err.Error(), "staged blob") || !strings.Contains(result.err.Error(), "is not a regular file") {
		t.Fatalf("OpenProjection(fifo staged blob) error = %v, want the staged-blob regularity refusal", result.err)
	}
}

// TestOpenProjectionRefusesNonDirectoryBlobShardByItsOwnClause pins the shard
// kind clause. A FIFO named with a valid lowercase-hex shard name at mode 0600
// passes the name check and the owner-only check that follow it, so disabling
// the clause moves the refusal to verifyOwnerDirectory and changes the message.
func TestOpenProjectionRefusesNonDirectoryBlobShardByItsOwnClause(t *testing.T) {
	store := openTestStore(t)
	digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
	shard := mkfifoFixture(t, filepath.Join(digestRoot, "aa"))

	result := openProjectionBounded(t, store.resolvedPathsForTest(t), shard)
	if !errors.Is(result.err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(fifo blob shard) error = %v, want %v", result.err, ErrProjectionSourceIntegrity)
	}
	if !strings.Contains(result.err.Error(), `blob shard "aa" is not a directory`) {
		t.Fatalf("OpenProjection(fifo blob shard) error = %v, want the shard kind clause rather than the ownership check behind it", result.err)
	}
}

// TestOpenProjectionSerializesProjectionWorkThroughTheIndexLock pins the mutual
// exclusion the README advertises. Owner-only mode, regularity and O_NOFOLLOW on
// the lock file were each pinned already; the flock itself was not, and the
// concurrency tests pass on SQLite's busy timeout alone. Two independent open
// file descriptions are what flock arbitrates, so holding one here reproduces a
// second process exactly.
func TestOpenProjectionSerializesProjectionWorkThroughTheIndexLock(t *testing.T) {
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	content := []byte("index lock serialization")
	if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	warm, _, err := OpenProjection(context.Background(), paths)
	if err != nil {
		t.Fatalf("OpenProjection() error = %v", err)
	}
	indexPath := warm.Path()
	if err := warm.Close(); err != nil {
		t.Fatal(err)
	}

	held, err := acquireProjectionLock(indexPath + ".lock")
	if err != nil {
		t.Fatalf("acquireProjectionLock() error = %v", err)
	}
	released := false
	defer func() {
		if !released {
			_ = held.release()
		}
	}()

	reached := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		projection, _, err := openProjection(context.Background(), paths, projectionHooks{
			beforeRebuildCommit: func() error {
				close(reached)
				return nil
			},
		})
		if projection != nil {
			_ = projection.Close()
		}
		done <- err
	}()

	// Without the flock the second open reaches its rebuild transaction in
	// milliseconds: the index already exists, its schema is current and no
	// SQLite connection is open, so nothing else would make it wait.
	select {
	case <-reached:
		t.Fatal("a second open reached the rebuild transaction while the index lock was held: the lock does not serialize projection work across open file descriptions")
	case err := <-done:
		t.Fatalf("a second open completed while the index lock was held (error = %v)", err)
	case <-time.After(2 * time.Second):
	}

	released = true
	if err := held.release(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-reached:
	case <-time.After(projectionOpenBound):
		t.Fatal("second open did not proceed after the index lock was released")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second open after release error = %v", err)
		}
	case <-time.After(projectionOpenBound):
		t.Fatal("second open did not complete after the index lock was released")
	}
}

// TestScanAuthoritativeBlobsRefusesUnreadableObjectsRootInsteadOfReportingAbsence
// pins the read-failure branch beside the absence branch. An lstat that fails
// for any reason other than ErrNotExist is not an empty object store, and
// treating it as one would launder an unreadable source into a clean, empty
// projection. InitializeLayout requires the data root to be exactly 0700 and
// refuses first, so no fixture reaches this branch through OpenProjection; it is
// driven at scanAuthoritativeBlobs, the function openProjection calls.
func TestScanAuthoritativeBlobsRefusesUnreadableObjectsRootInsteadOfReportingAbsence(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the superuser is not subject to directory search permissions")
	}
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	if err := os.Mkdir(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dataRoot, "objects"), 0o700); err != nil {
		t.Fatal(err)
	}
	objects, err := scanAuthoritativeBlobs(dataRoot, projectionHooks{})
	if err != nil {
		t.Fatalf("scanAuthoritativeBlobs(readable empty source) error = %v, want acceptance", err)
	}
	if len(objects) != 0 {
		t.Fatalf("scanAuthoritativeBlobs(readable empty source) objects = %d, want 0", len(objects))
	}

	if err := os.Chmod(dataRoot, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dataRoot, 0o700) })

	objects, err = scanAuthoritativeBlobs(dataRoot, projectionHooks{})
	if err == nil {
		t.Fatalf("scanAuthoritativeBlobs(unreadable objects root) objects = %d, err = nil, want a read failure rather than a reported absence", len(objects))
	}
	if objects != nil {
		t.Fatalf("scanAuthoritativeBlobs(unreadable objects root) objects = %v, want no scan result", objects)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scanAuthoritativeBlobs(unreadable objects root) error = %v, want a read failure distinguished from absence", err)
	}
	if !errors.Is(err, ErrProjectionSourceIntegrity) {
		t.Fatalf("scanAuthoritativeBlobs(unreadable objects root) error = %v, want %v", err, ErrProjectionSourceIntegrity)
	}
	if !strings.Contains(err.Error(), "lstat objects root") {
		t.Fatalf("scanAuthoritativeBlobs(unreadable objects root) error = %v, want the objects-root read failure named", err)
	}
}
