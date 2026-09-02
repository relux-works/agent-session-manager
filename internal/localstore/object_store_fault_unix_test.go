//go:build darwin || linux

package localstore

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// The regularity clause in inspectExisting is already reached by the directory
// fixture in object_store_test.go, but a directory does not isolate it: with
// the clause removed, os.Open succeeds and the read fails with EISDIR, so the
// unreadable verdict decides the outcome and the object store still refuses.
// That fixture therefore proves the clause exists, and proves nothing about the
// class of entries it has to cover.
//
// A FIFO at exactly mode 0600 is the fixture that does. It is not a symlink, it
// belongs to the effective user and it already carries owner-only bits, so
// every preceding check passes and the shape clause is the only thing that can
// refuse. It also bounds availability rather than only safety: with the clause
// removed, openBlobFile blocks indefinitely on a FIFO that has no writer, so a
// digest path an attacker or a crashed tool replaced with one would hang the
// writer instead of refusing it. The case therefore asserts a bounded return.

const putBlobOpenBound = 20 * time.Second

// putBlobResult carries a bounded PutBlob outcome back to the test goroutine.
type putBlobResult struct {
	result PutResult
	err    error
}

// putBlobBounded drives the real PutBlob entry point and fails instead of
// hanging when it does not return. fifos names any FIFO fixture whose blocked
// reader must be released so the leaked goroutine cannot outlive the failure.
func putBlobBounded(
	t *testing.T, store *ObjectStore, digest scalar.Digest, size uint64, content []byte, fifos ...string,
) putBlobResult {
	t.Helper()
	done := make(chan putBlobResult, 1)
	go func() {
		result, err := store.PutBlob(digest, size, bytes.NewReader(content))
		done <- putBlobResult{result: result, err: err}
	}()
	select {
	case outcome := <-done:
		return outcome
	case <-time.After(putBlobOpenBound):
		for _, fifo := range fifos {
			if file, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
				_ = file.Close()
			}
		}
		t.Fatalf("PutBlob did not return within %s: the shape clause under test is the only bound on this write", putBlobOpenBound)
		return putBlobResult{}
	}
}

// TestPutBlobRefusesSpecialFileAtDigestPathWithoutOpeningOrMovingIt pins the
// regularity clause of inspectExisting with an isolated FIFO fixture, and pins
// the consequence that clause is allowed to have. A special file is an unsafe
// entry, not a proven mismatch, so it may not be moved out of the immutable
// namespace: only a completed, disagreeing read authorizes that.
func TestPutBlobRefusesSpecialFileAtDigestPathWithoutOpeningOrMovingIt(t *testing.T) {
	store := openTestStore(t)
	content := []byte("candidate for a digest path that is not a file")
	digest := scalar.SHA256Digest(content)
	target := requireDigestPath(t, store.objects, digest)
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	mkfifoFixture(t, target)

	outcome := putBlobBounded(t, store, digest, uint64(len(content)), content, target)
	if !errors.Is(outcome.err, ErrImmutableConflict) {
		t.Fatalf("PutBlob(fifo digest path) error = %v, want %v", outcome.err, ErrImmutableConflict)
	}
	if !strings.Contains(outcome.err.Error(), "target is not a regular file") {
		t.Fatalf("PutBlob(fifo digest path) error = %v, want the digest-path regularity clause", outcome.err)
	}
	if outcome.result.QuarantinePath == "" {
		t.Fatalf("PutBlob(fifo digest path) result = %+v, want the refused candidate preserved", outcome.result)
	}
	assertFileBytes(t, outcome.result.QuarantinePath, content)
	if outcome.result.ExistingQuarantinePath != "" {
		t.Fatalf("unsafe digest-path entry was moved to %q on a verdict that proved nothing", outcome.result.ExistingQuarantinePath)
	}
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("digest path after refusal: info/error = %v/%v, want the FIFO left exactly in place", info, err)
	}
	assertNoStagedFiles(t, filepath.Dir(target))

	// An operator removes the entry the store refused to touch. The refusal
	// must have left the digest path usable, not merely left it alone.
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if retryErr != nil || !retry.Installed || retry.Path != target {
		t.Fatalf("PutBlob(retry after the FIFO digest path was cleared) result/error = %+v/%v, want the declared bytes installed",
			retry, retryErr)
	}
	assertFileBytes(t, target, content)
}

// TestPutBlobRefusesSpecialFileAtObjectShardBeforeStagingAnything drives the
// same class one level up the tree. A shard replaced by a FIFO must stop the
// write before os.CreateTemp is reached, because there is no directory to stage
// into and a writer that followed it would block on a pipe instead.
func TestPutBlobRefusesSpecialFileAtObjectShardBeforeStagingAnything(t *testing.T) {
	store := openTestStore(t)
	content := []byte("candidate for a shard that is not a directory")
	digest := scalar.SHA256Digest(content)
	target := requireDigestPath(t, store.objects, digest)
	shard := mkfifoFixture(t, filepath.Dir(target))

	outcome := putBlobBounded(t, store, digest, uint64(len(content)), content, shard)
	if !errors.Is(outcome.err, ErrDurability) {
		t.Fatalf("PutBlob(fifo object shard) error = %v, want %v", outcome.err, ErrDurability)
	}
	if !strings.Contains(outcome.err.Error(), "create object shard") {
		t.Fatalf("PutBlob(fifo object shard) error = %v, want the shard creation clause", outcome.err)
	}
	if outcome.result != (PutResult{}) {
		t.Fatalf("PutBlob(fifo object shard) result = %+v, want no mutation reported", outcome.result)
	}
	info, err := os.Lstat(shard)
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("object shard after refusal: info/error = %v/%v, want the FIFO left exactly in place", info, err)
	}
	assertQuarantineIsEmpty(t, store)

	if err := os.Remove(shard); err != nil {
		t.Fatal(err)
	}
	retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if retryErr != nil || !retry.Installed || retry.Path != target {
		t.Fatalf("PutBlob(retry after the FIFO shard was cleared) result/error = %+v/%v, want the declared bytes installed",
			retry, retryErr)
	}
	assertFileBytes(t, target, content)
}
