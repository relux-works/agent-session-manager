package localstore

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var errInjectedBlobRead = errors.New("injected blob read failure")

// failingBlobOpener injects the three shapes of an incomplete read that a real
// descriptor exhaustion or media failure produces. None of them establish the
// bytes on disk, so none of them may be treated as a proven mismatch.
func failingBlobOpener(stage string) blobOpener {
	return func(path string) (io.ReadCloser, error) {
		switch stage {
		case "open":
			return nil, errInjectedBlobRead
		case "read":
			return readCloserFunc{read: func([]byte) (int, error) { return 0, errInjectedBlobRead }}, nil
		case "partial":
			delivered := false
			return readCloserFunc{read: func(destination []byte) (int, error) {
				if delivered {
					return 0, errInjectedBlobRead
				}
				delivered = true
				return copy(destination, []byte("imm")), nil
			}}, nil
		case "close":
			return readCloserFunc{
				read:  func([]byte) (int, error) { return 0, io.EOF },
				close: func() error { return errInjectedBlobRead },
			}, nil
		}
		return openBlobFile(path)
	}
}

type readCloserFunc struct {
	read  func([]byte) (int, error)
	close func() error
}

func (reader readCloserFunc) Read(buffer []byte) (int, error) { return reader.read(buffer) }

func (reader readCloserFunc) Close() error {
	if reader.close == nil {
		return nil
	}
	return reader.close()
}

// TestPutBlobReportsIncompleteExistingReadAsDurabilityAndMovesNothing is the
// negative case for the quarantine gate. SPEC.md §3.2 permits quarantine only
// for a hash mismatch or representation disagreement, so a read that never
// completed must leave the immutable object exactly where it is.
//
// Collapsing the classification back — routing any non-matching inspection to
// quarantineExisting, or widening blobVerdict.quarantineWarranted beyond
// blobMismatch — reddens this test: the durability assertion becomes an
// immutable conflict, ExistingQuarantinePath becomes non-empty, and the
// existing object disappears from the digest path.
func TestPutBlobReportsIncompleteExistingReadAsDurabilityAndMovesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("immutable")
	digest := scalar.SHA256Digest(content)
	tests := []struct {
		name  string
		stage string
		raced bool
	}{
		{name: "open fails", stage: "open"},
		{name: "read fails mid copy", stage: "read"},
		{name: "close fails after read", stage: "close"},
		{name: "raced entry appears then open fails", stage: "open", raced: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			target, err := nativeDigestPath(store.objects, digest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}

			operations := defaultStoreOperations()
			operations.openExisting = failingBlobOpener(test.stage)
			if test.raced {
				// The digest path is empty when PutBlob inspects it, so the
				// entry can only be discovered on the post-rename raced branch.
				// Both branches must classify the failure identically.
				install := operations.install
				operations.install = func(source, destination string) error {
					if destination != target {
						return install(source, destination)
					}
					if err := os.WriteFile(target, content, 0o600); err != nil {
						return err
					}
					return os.ErrExist
				}
			} else if err := os.WriteFile(target, content, 0o600); err != nil {
				t.Fatal(err)
			}
			store.operations = operations

			result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if !errors.Is(err, ErrDurability) {
				t.Fatalf("PutBlob(incomplete existing read) error = %v, want %v", err, ErrDurability)
			}
			if errors.Is(err, ErrImmutableConflict) || errors.Is(err, ErrIntegrityMismatch) {
				t.Fatalf("PutBlob(incomplete existing read) error = %v, want a durability failure rather than an integrity finding", err)
			}
			if result.QuarantinePath != "" || result.ExistingQuarantinePath != "" {
				t.Fatalf("PutBlob(incomplete existing read) result = %+v, want nothing quarantined", result)
			}
			assertFileBytes(t, target, content)
			assertNoStagedFiles(t, filepath.Dir(target))
			assertQuarantineNamespaceEmpty(t, store)

			// The object was not sidelined: once the read works again the same
			// digest path verifies and the identical retry reuses the inode.
			store.operations = defaultStoreOperations()
			retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if retryErr != nil {
				t.Fatalf("PutBlob(retry after transient read failure) error = %v", retryErr)
			}
			if retry.Path != target || retry.Installed {
				t.Fatalf("PutBlob(retry after transient read failure) result = %+v, want the existing inode reused at %q", retry, target)
			}
			assertFileBytes(t, target, content)
		})
	}
}

// TestObjectStoreAndProjectionAgreeOnIncompleteReadVersusProvenMismatch drives
// both production entry points over the same two on-disk conditions and asserts
// they reach the same classification. The agreement is the point: the store is
// the path that moves data, so a disagreement there is what turns a flaky read
// into a permanently sidelined object.
func TestObjectStoreAndProjectionAgreeOnIncompleteReadVersusProvenMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("immutable")
	digest := scalar.SHA256Digest(content)
	tests := []struct {
		name string
		// corrupt replaces the installed bytes to create a completed,
		// disagreeing read; empty leaves them intact.
		corrupt []byte
		// failRead injects an incomplete read on both paths.
		failRead bool
		// wantProvenMismatch is the shared classification both paths must reach.
		wantProvenMismatch bool
	}{
		{name: "completed read disagrees", corrupt: []byte("corrupted"), wantProvenMismatch: true},
		{name: "read does not complete", failRead: true, wantProvenMismatch: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storeMismatch, storeMoved := classifyThroughObjectStore(t, content, digest, test.corrupt, test.failRead)
			projectionMismatch, projectionMoved := classifyThroughProjection(t, content, digest, test.corrupt, test.failRead)

			if storeMismatch != projectionMismatch {
				t.Fatalf("object store proven-mismatch = %t, projection proven-mismatch = %t, want the two paths to agree",
					storeMismatch, projectionMismatch)
			}
			if storeMismatch != test.wantProvenMismatch {
				t.Fatalf("shared classification = %t, want proven mismatch %t", storeMismatch, test.wantProvenMismatch)
			}
			if projectionMoved {
				t.Fatal("projection moved a durable object; it may only refuse")
			}
			if storeMoved != test.wantProvenMismatch {
				t.Fatalf("object store moved the existing object = %t, want %t; only a proven mismatch may quarantine",
					storeMoved, test.wantProvenMismatch)
			}
		})
	}
}

// classifyThroughObjectStore drives PutBlob and reports whether it called the
// condition a proven integrity mismatch and whether it moved the existing
// object out of the immutable namespace.
func classifyThroughObjectStore(t *testing.T, content []byte, digest scalar.Digest, corrupt []byte, failRead bool) (bool, bool) {
	t.Helper()
	store := openTestStore(t)
	if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatalf("PutBlob(install fixture) error = %v", err)
	}
	target, err := nativeDigestPath(store.objects, digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(corrupt) != 0 {
		if err := os.WriteFile(target, corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if failRead {
		operations := defaultStoreOperations()
		operations.openExisting = failingBlobOpener("read")
		store.operations = operations
	}

	result, putErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if putErr == nil {
		t.Fatalf("PutBlob(%s) succeeded; the fixture must not verify", t.Name())
	}
	_, statErr := os.Lstat(target)
	moved := errors.Is(statErr, fs.ErrNotExist)
	if !moved && statErr != nil {
		t.Fatalf("existing object stat error = %v", statErr)
	}
	if moved != (result.ExistingQuarantinePath != "") {
		t.Fatalf("existing object removed = %t but ExistingQuarantinePath = %q; a moved object must be reported",
			moved, result.ExistingQuarantinePath)
	}
	return errors.Is(putErr, ErrImmutableConflict), moved
}

// classifyThroughProjection drives openProjection over the identical condition
// and reports whether it called it a proven integrity mismatch and whether the
// source blob moved.
func classifyThroughProjection(t *testing.T, content []byte, digest scalar.Digest, corrupt []byte, failRead bool) (bool, bool) {
	t.Helper()
	store := openTestStore(t)
	if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatalf("PutBlob(install fixture) error = %v", err)
	}
	target, err := nativeDigestPath(store.objects, digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(corrupt) != 0 {
		if err := os.WriteFile(target, corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	hooks := projectionHooks{}
	if failRead {
		hooks.openBlob = failingBlobOpener("read")
	}

	_, _, openErr := openProjection(context.Background(), store.resolvedPathsForTest(t), hooks)
	if openErr == nil {
		t.Fatalf("openProjection(%s) succeeded; the fixture must not verify", t.Name())
	}
	if !errors.Is(openErr, ErrProjectionSourceIntegrity) {
		t.Fatalf("openProjection() error = %v, want %v", openErr, ErrProjectionSourceIntegrity)
	}
	_, statErr := os.Lstat(target)
	moved := errors.Is(statErr, fs.ErrNotExist)
	if !moved && statErr != nil {
		t.Fatalf("source blob stat error = %v", statErr)
	}
	// The projection reports an incomplete read as a source failure and a
	// completed disagreement as a refusal about the bytes. Only the second is a
	// proven mismatch.
	return strings.Contains(openErr.Error(), "contains"), moved
}

// TestBlobVerdictAuthorizesQuarantineOnlyForACompletedDisagreeingRead pins the
// single predicate both paths consult. Widening it to any non-matching verdict,
// or narrowing it so a same-size digest disagreement no longer qualifies,
// reddens the production tests above and this one together.
func TestBlobVerdictAuthorizesQuarantineOnlyForACompletedDisagreeingRead(t *testing.T) {
	tests := []struct {
		verdict blobVerdict
		want    bool
	}{
		{verdict: blobUnproven, want: false},
		{verdict: blobAbsent, want: false},
		{verdict: blobMatches, want: false},
		{verdict: blobMismatch, want: true},
		{verdict: blobUnreadable, want: false},
		{verdict: blobUnsafe, want: false},
	}
	for _, test := range tests {
		if got := test.verdict.quarantineWarranted(); got != test.want {
			t.Fatalf("blobVerdict(%d).quarantineWarranted() = %t, want %t", test.verdict, got, test.want)
		}
	}
}

// TestVerifyBlobContentSeparatesAnIncompleteReadFromADisagreeingOne pins the
// shared classifier itself, including the same-size digest disagreement that a
// narrowed mismatch check would drop.
func TestVerifyBlobContentSeparatesAnIncompleteReadFromADisagreeingOne(t *testing.T) {
	content := []byte("immutable")
	digest := scalar.SHA256Digest(content)
	directory := t.TempDir()
	path := filepath.Join(directory, "blob")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	if inspection := verifyBlobContent(openBlobFile, path, digest, uint64(len(content))); inspection.verdict != blobMatches {
		t.Fatalf("verifyBlobContent(matching bytes) verdict = %d, want blobMatches", inspection.verdict)
	}
	for _, stage := range []string{"open", "read", "close"} {
		inspection := verifyBlobContent(failingBlobOpener(stage), path, digest, uint64(len(content)))
		if inspection.verdict != blobUnreadable {
			t.Fatalf("verifyBlobContent(%s failure) verdict = %d, want blobUnreadable", stage, inspection.verdict)
		}
		if !errors.Is(inspection.err, errInjectedBlobRead) {
			t.Fatalf("verifyBlobContent(%s failure) error = %v, want the read cause preserved", stage, inspection.err)
		}
	}
	if inspection := verifyBlobContent(nil, path, digest, uint64(len(content))); inspection.verdict != blobUnreadable {
		t.Fatalf("verifyBlobContent(no reader) verdict = %d, want blobUnreadable", inspection.verdict)
	}

	sameSize := []byte("immutabl3")
	if len(sameSize) != len(content) {
		t.Fatalf("fixture sizes differ: %d and %d", len(sameSize), len(content))
	}
	if err := os.WriteFile(path, sameSize, 0o600); err != nil {
		t.Fatal(err)
	}
	inspection := verifyBlobContent(openBlobFile, path, digest, uint64(len(content)))
	if inspection.verdict != blobMismatch {
		t.Fatalf("verifyBlobContent(same size, different digest) verdict = %d, want blobMismatch", inspection.verdict)
	}
	if inspection.digest != scalar.SHA256Digest(sameSize) || inspection.size != uint64(len(sameSize)) {
		t.Fatalf("verifyBlobContent(same size, different digest) = %+v, want the read identity reported", inspection)
	}
}

// TestInspectExistingClassifiesEveryFaultItCanReach pins the classification for
// every condition the digest path can present. inspectExisting is driven
// directly because two of these faults have no on-disk fixture: an Lstat that
// fails for a reason other than absence, and a read that stops part way
// through. Its production call site is PutBlob, which routes every verdict
// through resolveExistingEntry; the consequences of the moving verdicts are
// proven at that entry point by the tests above.
func TestInspectExistingClassifiesEveryFaultItCanReach(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("immutable")
	digest := scalar.SHA256Digest(content)
	tests := []struct {
		name     string
		occupant func(t *testing.T, store *ObjectStore, target string)
		want     blobVerdict
	}{
		{
			name:     "absent",
			occupant: func(*testing.T, *ObjectStore, string) {},
			want:     blobAbsent,
		},
		{
			name: "identical bytes",
			occupant: func(t *testing.T, _ *ObjectStore, target string) {
				writeDigestPathOccupant(t, target, content, 0o600)
			},
			want: blobMatches,
		},
		{
			name: "different bytes at the same size",
			occupant: func(t *testing.T, _ *ObjectStore, target string) {
				writeDigestPathOccupant(t, target, []byte("immutabl3"), 0o600)
			},
			want: blobMismatch,
		},
		{
			name: "group readable mode",
			occupant: func(t *testing.T, _ *ObjectStore, target string) {
				writeDigestPathOccupant(t, target, content, 0o640)
			},
			want: blobUnsafe,
		},
		{
			name: "directory in the digest path",
			occupant: func(t *testing.T, _ *ObjectStore, target string) {
				if err := os.MkdirAll(target, 0o700); err != nil {
					t.Fatal(err)
				}
			},
			want: blobUnsafe,
		},
		{
			name: "symlink in the digest path",
			occupant: func(t *testing.T, _ *ObjectStore, target string) {
				if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(filepath.Dir(target), "elsewhere"), target); err != nil {
					t.Fatal(err)
				}
			},
			want: blobUnsafe,
		},
		{
			name: "lstat fails for a reason other than absence",
			occupant: func(t *testing.T, _ *ObjectStore, target string) {
				// A non-directory shard makes the Lstat of the leaf fail with
				// ENOTDIR rather than ENOENT: present-but-unreadable, not absent.
				shard := filepath.Dir(target)
				if err := os.MkdirAll(filepath.Dir(shard), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(shard, []byte("not a directory"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: blobUnreadable,
		},
		{
			name: "open fails",
			occupant: func(t *testing.T, store *ObjectStore, target string) {
				writeDigestPathOccupant(t, target, content, 0o600)
				store.operations.openExisting = failingBlobOpener("open")
			},
			want: blobUnreadable,
		},
		{
			name: "read stops part way through",
			occupant: func(t *testing.T, store *ObjectStore, target string) {
				writeDigestPathOccupant(t, target, content, 0o600)
				store.operations.openExisting = failingBlobOpener("partial")
			},
			want: blobUnreadable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			target, err := nativeDigestPath(store.objects, digest)
			if err != nil {
				t.Fatal(err)
			}
			test.occupant(t, store, target)

			inspection := store.inspectExisting(target, digest, uint64(len(content)))
			if inspection.verdict != test.want {
				t.Fatalf("inspectExisting(%s) verdict = %d, want %d (error = %v)",
					test.name, inspection.verdict, test.want, inspection.err)
			}
			if inspection.verdict.quarantineWarranted() != (test.want == blobMismatch) {
				t.Fatalf("inspectExisting(%s) authorized a move on verdict %d", test.name, inspection.verdict)
			}
		})
	}
}

func writeDigestPathOccupant(t *testing.T, target string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, mode); err != nil {
		t.Fatal(err)
	}
}

func assertQuarantineNamespaceEmpty(t *testing.T, store *ObjectStore) {
	t.Helper()
	var found []string
	err := filepath.WalkDir(store.quarantine, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk quarantine namespace error = %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("quarantine namespace contains %v, want nothing moved", found)
	}
}
