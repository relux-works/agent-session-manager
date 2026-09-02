package localstore

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// The cases in this file drive PutBlob against the filesystem faults a full or
// failing volume actually produces, rather than against the sentinel-shaped
// inputs the entry point validates before it touches the disk. Three properties
// are under test at every one of those boundaries, because a durable write that
// gives up has to give up completely:
//
//   - the immutable digest path stays absent, so no truncated or unverified
//     object is ever reachable as an installed object;
//   - the shard directory keeps no staged residue, so a repeated failure cannot
//     accumulate partial candidates next to live objects;
//   - a retry once the fault clears installs the exact declared bytes, so the
//     refusal is recoverable rather than terminal.
//
// The last property is what separates a refusal from a loss, so every case
// carries it — but two of them carry it in a different shape, and each says so
// at the case itself. A candidate whose quarantine move was denied is not a
// valid object, so its recovery is that the same candidate is preserved as
// evidence once the volume has room. A corrupt object whose move was denied is
// deliberately left at the digest path, so its recovery is what the next run
// does with it still sitting there.
//
// storeOperations.syncFile is the seam every write-side fault is driven
// through. That is not an accident of convenience: fsync is where a volume with
// delayed allocation actually discovers it is full, so a failure raised there,
// and a length that changed by the time the staged file is stat'd, are the two
// shapes a real ENOSPC takes on ext4, XFS and APFS.

// stagedIdentityFault is a durability fault that is invisible to both the copy
// and the hasher: the bytes the source produced are exactly the declared bytes
// and the digest agrees, but what the filesystem kept does not. It exists to
// isolate the staged-length comparison in PutBlob, which no other clause in
// that condition can decide.
type stagedIdentityFault struct {
	name       string
	keptLength func(declared int64) int64
}

// TestPutBlobRefusesShortStagedWriteAsDurabilityNotMismatch pins the staged
// length comparison in PutBlob. Both fixtures leave the declared digest and the
// copied byte count in complete agreement with the caller's declaration, so the
// two identity terms that follow cannot refuse and the length term is the only
// thing standing between a partially written file and the immutable namespace.
//
// Narrowing rather than deleting is what the second case buys: a comparison
// weakened from "the kept length differs" to "the kept length is short" still
// refuses the truncation and admits the over-long file, which is the same class
// of durability fault seen from the other side.
//
// The refusal is a durability failure and not an integrity mismatch, and the
// test asserts that distinction directly. A write that did not complete proves
// nothing about the source bytes, so quarantining the fragment would record it
// as disagreeing evidence about a source that may have been perfectly valid,
// and would consume more of the space whose exhaustion caused the fault.
func TestPutBlobRefusesShortStagedWriteAsDurabilityNotMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("bytes the volume did not keep")
	digest := scalar.SHA256Digest(content)
	faults := []stagedIdentityFault{
		{name: "volume kept less than the copy accepted", keptLength: func(declared int64) int64 { return declared - 1 }},
		{name: "volume kept more than the copy accepted", keptLength: func(declared int64) int64 { return declared + 1 }},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			store := openTestStore(t)
			target := requireDigestPath(t, store.objects, digest)
			operations := defaultStoreOperations()
			operations.syncFile = func(file *os.File) error {
				if err := file.Sync(); err != nil {
					return err
				}
				return file.Truncate(fault.keptLength(int64(len(content))))
			}
			store.operations = operations

			result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if !errors.Is(err, ErrDurability) {
				t.Fatalf("PutBlob(%s) error = %v, want %v", fault.name, err, ErrDurability)
			}
			if errors.Is(err, ErrIntegrityMismatch) || errors.Is(err, ErrImmutableConflict) {
				t.Fatalf("PutBlob(%s) error = %v, want an incomplete write reported as durability, not as proven disagreement", fault.name, err)
			}
			if !strings.Contains(err.Error(), "staged write kept") {
				t.Fatalf("PutBlob(%s) error = %v, want the staged-length clause", fault.name, err)
			}
			if result != (PutResult{}) {
				t.Fatalf("PutBlob(%s) result = %+v, want no reported identity for bytes that never landed", fault.name, result)
			}
			if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("PutBlob(%s) digest path stat error = %v, want absent", fault.name, statErr)
			}
			assertNoStagedFiles(t, filepath.Dir(target))
			assertQuarantineIsEmpty(t, store)

			store.operations = defaultStoreOperations()
			retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if retryErr != nil || !retry.Installed || retry.Path != target {
				t.Fatalf("PutBlob(retry after %s) result/error = %+v/%v, want the declared bytes installed", fault.name, retry, retryErr)
			}
			assertFileBytes(t, target, content)
		})
	}
}

// TestPutBlobRefusesFullVolumeFaultsOnEverySideOfTheStagedWrite drives the two
// error-raising directions of the staged copy — the source stops producing and
// the volume stops accepting — plus the writeback failure a filesystem with
// delayed allocation raises at fsync rather than at the write that filled it.
//
// syscall.ENOSPC is used as the injected cause rather than an anonymous error
// so the cases read as the condition they model. PutBlob deliberately does not
// republish that cause: ErrDurability is the contract, and the assertions below
// only require the sentinel and a recoverable retry.
func TestPutBlobRefusesFullVolumeFaultsOnEverySideOfTheStagedWrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("payload that needs a volume with room")
	digest := scalar.SHA256Digest(content)
	tests := []struct {
		name    string
		source  func() io.Reader
		prepare func(*storeOperations)
	}{
		{
			name: "source stops producing part way through the copy",
			source: func() io.Reader {
				return io.MultiReader(bytes.NewReader(content[:4]), failingReader{err: syscall.ENOSPC})
			},
		},
		{
			name:   "volume refuses the accepting write",
			source: func() io.Reader { return bytes.NewReader(content) },
			prepare: func(operations *storeOperations) {
				operations.syncFile = func(*os.File) error { return syscall.ENOSPC }
			},
		},
		{
			name:   "volume discovers exhaustion at writeback",
			source: func() io.Reader { return bytes.NewReader(content) },
			prepare: func(operations *storeOperations) {
				operations.syncFile = func(file *os.File) error {
					_ = file.Sync()
					return &os.PathError{Op: "fsync", Path: file.Name(), Err: syscall.ENOSPC}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			target := requireDigestPath(t, store.objects, digest)
			operations := defaultStoreOperations()
			if test.prepare != nil {
				test.prepare(&operations)
			}
			store.operations = operations

			result, err := store.PutBlob(digest, uint64(len(content)), test.source())
			if !errors.Is(err, ErrDurability) {
				t.Fatalf("PutBlob(%s) error = %v, want %v", test.name, err, ErrDurability)
			}
			if result != (PutResult{}) {
				t.Fatalf("PutBlob(%s) result = %+v, want no mutation reported", test.name, result)
			}
			if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("PutBlob(%s) digest path stat error = %v, want absent", test.name, statErr)
			}
			assertNoStagedFiles(t, filepath.Dir(target))
			assertQuarantineIsEmpty(t, store)

			store.operations = defaultStoreOperations()
			retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if retryErr != nil || !retry.Installed {
				t.Fatalf("PutBlob(retry after %s) result/error = %+v/%v, want a recoverable install", test.name, retry, retryErr)
			}
			assertFileBytes(t, target, content)
		})
	}
}

// TestPutBlobRefusesStagedModeDriftBeforeInstallingIt pins the owner-only check
// PutBlob applies to the file it staged itself. The fixture changes the mode
// after the bytes are written and flushed, which is the window an out-of-band
// writer or a permissive inherited ACL has, and it changes nothing else: the
// staged file still holds exactly the declared bytes, so no length or identity
// clause can refuse and the owner-only gate is the only thing that can.
func TestPutBlobRefusesStagedModeDriftBeforeInstallingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	content := []byte("correct bytes at the wrong mode")
	digest := scalar.SHA256Digest(content)
	target := requireDigestPath(t, store.objects, digest)
	operations := defaultStoreOperations()
	operations.syncFile = func(file *os.File) error {
		if err := file.Sync(); err != nil {
			return err
		}
		return os.Chmod(file.Name(), 0o644)
	}
	store.operations = operations

	result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if !errors.Is(err, ErrDurability) {
		t.Fatalf("PutBlob(group-readable staged file) error = %v, want %v", err, ErrDurability)
	}
	if !strings.Contains(err.Error(), "staged blob") || !strings.Contains(err.Error(), "mode is 0644") {
		t.Fatalf("PutBlob(group-readable staged file) error = %v, want the staged owner-only clause", err)
	}
	if result != (PutResult{}) {
		t.Fatalf("PutBlob(group-readable staged file) result = %+v, want no mutation reported", result)
	}
	if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("world-readable candidate reached the immutable namespace: stat error = %v", statErr)
	}
	assertNoStagedFiles(t, filepath.Dir(target))
	assertQuarantineIsEmpty(t, store)

	// The out-of-band writer stops interfering. Nothing the refusal did may
	// have left the digest path unusable for the write it just declined.
	store.operations = defaultStoreOperations()
	retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if retryErr != nil || !retry.Installed || retry.Path != target {
		t.Fatalf("PutBlob(retry after staged mode drift cleared) result/error = %+v/%v, want the declared bytes installed", retry, retryErr)
	}
	assertFileBytes(t, target, content)
}

// TestPutBlobRefusesGroupAccessibleObjectShardBeforeStagingAnything drives the
// owner-only directory gate that runs before os.CreateTemp. A shard whose mode
// drifted must stop the write while it is still a no-op, because a staged file
// created inside a group-accessible directory is readable by another user for
// as long as it exists there.
func TestPutBlobRefusesGroupAccessibleObjectShardBeforeStagingAnything(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	content := []byte("candidate for a shard that drifted")
	digest := scalar.SHA256Digest(content)
	target := requireDigestPath(t, store.objects, digest)
	shardRoot := filepath.Join(store.objects, "sha256")
	if err := os.Chmod(shardRoot, 0o750); err != nil {
		t.Fatal(err)
	}

	result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if !errors.Is(err, ErrDurability) {
		t.Fatalf("PutBlob(group-accessible shard root) error = %v, want %v", err, ErrDurability)
	}
	if !strings.Contains(err.Error(), "create object shard") {
		t.Fatalf("PutBlob(group-accessible shard root) error = %v, want the shard creation clause", err)
	}
	if result != (PutResult{}) {
		t.Fatalf("PutBlob(group-accessible shard root) result = %+v, want no mutation reported", result)
	}
	if entries, readErr := os.ReadDir(shardRoot); readErr != nil || len(entries) != 0 {
		t.Fatalf("shard root entries/error = %v/%v, want nothing staged beneath an unsafe directory", entries, readErr)
	}
	assertQuarantineIsEmpty(t, store)

	if err := os.Chmod(shardRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if retryErr != nil || !retry.Installed || retry.Path != target {
		t.Fatalf("PutBlob(retry after shard repair) result/error = %+v/%v, want a recoverable install", retry, retryErr)
	}
	assertFileBytes(t, target, content)
}

// TestPutBlobRefusesSymlinkedObjectShardWithoutFollowingIt closes the symlink
// dimension one level above the digest path. The existing symlink case in
// object_store_test.go replaces the leaf, which inspectExisting classifies; this
// one replaces the directory the staged file would be created in, which is
// decided much earlier by the owner-only child-tree walk. A store that followed
// it would create the staged file, and then the immutable object, outside the
// AX-owned data root entirely.
//
// The two shards are driven separately because they are refused at different
// steps of the same walk, and a fixture at one says nothing about the other.
func TestPutBlobRefusesSymlinkedObjectShardWithoutFollowingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("candidate for a shard that points elsewhere")
	digest := scalar.SHA256Digest(content)
	tests := []struct {
		name  string
		shard func(*ObjectStore) string
	}{
		{"digest root", func(store *ObjectStore) string { return filepath.Join(store.objects, "sha256") }},
		{"digest shard", func(store *ObjectStore) string {
			return filepath.Join(store.objects, "sha256", digest.Hex()[:2])
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			shard := test.shard(store)
			external := filepath.Join(t.TempDir(), "external")
			if err := os.Mkdir(external, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(shard); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(shard), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, shard); err != nil {
				t.Fatal(err)
			}

			result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if !errors.Is(err, ErrDurability) {
				t.Fatalf("PutBlob(symlinked %s) error = %v, want %v", test.name, err, ErrDurability)
			}
			if !strings.Contains(err.Error(), "create object shard") {
				t.Fatalf("PutBlob(symlinked %s) error = %v, want the shard creation clause", test.name, err)
			}
			if result != (PutResult{}) {
				t.Fatalf("PutBlob(symlinked %s) result = %+v, want no mutation reported", test.name, result)
			}
			info, statErr := os.Lstat(shard)
			if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("symlinked %s was replaced or followed: info/error = %v/%v", test.name, info, statErr)
			}
			entries, readErr := os.ReadDir(external)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("symlink target entries/error = %v/%v, want nothing written outside the data root", entries, readErr)
			}
			assertQuarantineIsEmpty(t, store)

			if err := os.Remove(shard); err != nil {
				t.Fatal(err)
			}
			target := requireDigestPath(t, store.objects, digest)
			retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if retryErr != nil || !retry.Installed || retry.Path != target {
				t.Fatalf("PutBlob(retry after symlinked %s was removed) result/error = %+v/%v, want the declared bytes installed",
					test.name, retry, retryErr)
			}
			assertFileBytes(t, target, content)
		})
	}
}

// TestPutBlobReportsQuarantineFailuresWithoutLosingEitherArtifact covers the
// second-order fault: the volume is too full to accept the quarantine copy of
// the evidence a refusal just produced. Both quarantine movers are driven, and
// each is isolated from the other by the source path the injected installer
// refuses, because the candidate is always moved before the existing entry and
// a failure on the first would otherwise mask the second.
//
// The property being pinned is that a failed quarantine never becomes a silent
// deletion. Neither the immutable namespace nor the caller may end up believing
// an artifact was preserved when it was not.
//
// Each subtest then establishes that the refusal was a deferral and not a
// stall, which for a full volume is the half that matters: anything can fail,
// and what a durability suite has to show is what a later run can still do.
// The recovery differs in shape between the two because the artifact left
// behind differs. A candidate that was never a valid object cannot install on
// retry, so its recovery is that it is preserved as evidence exactly as it
// would have been without the fault. A corrupt existing object is deliberately
// left at the digest path, so its recovery is what the next run does with it
// still there.
func TestPutBlobReportsQuarantineFailuresWithoutLosingEitherArtifact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("valid")
	corrupt := []byte("corrupt existing object")

	t.Run("candidate quarantine cannot be written", func(t *testing.T) {
		store := openTestStore(t)
		declared := scalar.SHA256Digest([]byte("something else entirely"))
		target := requireDigestPath(t, store.objects, declared)
		store.operations = quarantineDenyingOperations(store, func(string) bool { return true })

		result, err := store.PutBlob(declared, uint64(len(content)), bytes.NewReader(content))
		if !errors.Is(err, ErrDurability) {
			t.Fatalf("PutBlob(mismatch on a full volume) error = %v, want %v", err, ErrDurability)
		}
		if !strings.Contains(err.Error(), "mismatch and quarantine failed") {
			t.Fatalf("PutBlob(mismatch on a full volume) error = %v, want the quarantine failure clause", err)
		}
		if result != (PutResult{}) {
			t.Fatalf("PutBlob(mismatch on a full volume) result = %+v, want no quarantine path claimed", result)
		}
		if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
			t.Fatalf("unverified candidate reached the immutable namespace: stat error = %v", statErr)
		}
		assertNoStagedFiles(t, filepath.Dir(target))
		assertQuarantineIsEmpty(t, store)

		// Room returns. The same mismatched candidate must now be preserved as
		// evidence exactly as it would have been had the volume never filled,
		// and it still must not reach the immutable namespace: recovering from
		// the fault may not also relax the verdict that produced it.
		store.operations = defaultStoreOperations()
		retry, retryErr := store.PutBlob(declared, uint64(len(content)), bytes.NewReader(content))
		if !errors.Is(retryErr, ErrIntegrityMismatch) {
			t.Fatalf("PutBlob(retry once quarantine has room) error = %v, want %v", retryErr, ErrIntegrityMismatch)
		}
		if retry.QuarantinePath == "" {
			t.Fatalf("PutBlob(retry once quarantine has room) result = %+v, want the mismatched candidate preserved", retry)
		}
		assertFileBytes(t, retry.QuarantinePath, content)
		if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
			t.Fatalf("retry installed an unverified candidate: stat error = %v", statErr)
		}
		assertNoStagedFiles(t, filepath.Dir(target))
	})

	t.Run("existing quarantine cannot be written", func(t *testing.T) {
		store := openTestStore(t)
		digest := scalar.SHA256Digest(content)
		target := requireDigestPath(t, store.objects, digest)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
		// Only the existing object's move is denied. The candidate is moved
		// first and must still succeed, so the failure observed below can only
		// be the second mover's.
		store.operations = quarantineDenyingOperations(store, func(source string) bool { return source == target })

		result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
		if !errors.Is(err, ErrImmutableConflict) {
			t.Fatalf("PutBlob(corrupt existing on a full volume) error = %v, want %v", err, ErrImmutableConflict)
		}
		if !strings.Contains(err.Error(), "existing quarantine failed") {
			t.Fatalf("PutBlob(corrupt existing on a full volume) error = %v, want the existing quarantine failure clause", err)
		}
		if result.QuarantinePath == "" {
			t.Fatalf("PutBlob(corrupt existing on a full volume) result = %+v, want the candidate reported as preserved", result)
		}
		assertFileBytes(t, result.QuarantinePath, content)
		if result.ExistingQuarantinePath != "" {
			t.Fatalf("PutBlob(corrupt existing on a full volume) claimed existing quarantine at %q that was never written", result.ExistingQuarantinePath)
		}
		assertFileBytes(t, target, corrupt)

		// The corrupt object was deliberately left at the digest path, so the
		// recovery here is not a retry of the same call succeeding but what a
		// later run does with it still in place. It must read it, prove the
		// disagreement, and preserve both artifacts. That path is what keeps a
		// denied move from stranding the digest path permanently, and nothing
		// else in the suite pins it.
		firstCandidate := result.QuarantinePath
		store.operations = defaultStoreOperations()
		second, secondErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
		if !errors.Is(secondErr, ErrImmutableConflict) {
			t.Fatalf("PutBlob(later run over the corrupt object) error = %v, want %v", secondErr, ErrImmutableConflict)
		}
		if !strings.Contains(secondErr.Error(), "existing digest path failed verification") {
			t.Fatalf("PutBlob(later run over the corrupt object) error = %v, want the completed-disagreement clause", secondErr)
		}
		if second.QuarantinePath == "" || second.ExistingQuarantinePath == "" {
			t.Fatalf("PutBlob(later run over the corrupt object) result = %+v, want both artifacts preserved", second)
		}
		assertFileBytes(t, second.QuarantinePath, content)
		assertFileBytes(t, second.ExistingQuarantinePath, corrupt)
		// The artifact the first run did preserve is untouched by the second.
		assertFileBytes(t, firstCandidate, content)
		if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
			t.Fatalf("corrupt object remains at the digest path after its quarantine succeeded: stat error = %v", statErr)
		}
		assertNoStagedFiles(t, filepath.Dir(target))

		// With the conflict cleared by that quarantine, the declared bytes
		// finally install. The store returned to a usable state on its own.
		third, thirdErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
		if thirdErr != nil || !third.Installed || third.Path != target {
			t.Fatalf("PutBlob(run after the conflict cleared) result/error = %+v/%v, want the declared bytes installed", third, thirdErr)
		}
		assertFileBytes(t, target, content)
	})
}

// failingReader is a source that stops producing bytes with a chosen cause.
type failingReader struct{ err error }

func (reader failingReader) Read([]byte) (int, error) { return 0, reader.err }

// quarantineDenyingOperations returns production operations whose installer
// refuses exactly the quarantine moves the predicate selects, with the error a
// full volume raises. Every other install, including the atomic no-replace
// rename into the immutable namespace, remains the production implementation.
func quarantineDenyingOperations(store *ObjectStore, deny func(source string) bool) storeOperations {
	operations := defaultStoreOperations()
	install := operations.install
	operations.install = func(source, destination string) error {
		if strings.HasPrefix(destination, store.quarantine+string(os.PathSeparator)) && deny(source) {
			return &os.LinkError{Op: "rename", Old: source, New: destination, Err: syscall.ENOSPC}
		}
		return install(source, destination)
	}
	return operations
}

func requireDigestPath(t *testing.T, root string, digest scalar.Digest) string {
	t.Helper()
	path, err := nativeDigestPath(root, digest)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// assertQuarantineIsEmpty proves a refusal moved nothing into quarantine. It is
// the counterpart of the quarantine assertions in object_store_test.go: those
// prove a proven mismatch is preserved, and this proves an unproven one is not
// manufactured into evidence.
func assertQuarantineIsEmpty(t *testing.T, store *ObjectStore) {
	t.Helper()
	err := filepath.WalkDir(store.quarantine, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			t.Errorf("quarantine holds %q after a refusal that proved nothing about the source bytes", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
