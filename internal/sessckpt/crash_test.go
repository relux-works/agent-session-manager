package sessckpt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// wireCrashPoints connects a store's write boundaries to the
// owner's crash injector: BeforeWrite fires PointPrepareEnter
// (before any durable byte) with the safe_retry outcome. The
// interior and post-commit boundaries return plain test faults:
// their recovery is proved by the store's own replay semantics
// below, not by borrowing a mismatched injector label.
func wireCrashPoints(store *Store, injector *secconftest.Injector) {
	store.BeforeWrite = func() error {
		return injector.MaybeFail(secconftest.PointPrepareEnter)
	}
}

// mustCrashFault fails unless err is the injected fault for the
// armed point carrying the expected outcome.
func mustCrashFault(t *testing.T, err error, point string, outcome secconftest.Outcome, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want injected fault at %s", what, point)
	}
	var fault *secconftest.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("%s error = %v, want *secconftest.Fault", what, err)
	}
	if fault.Point() != point {
		t.Fatalf("%s point = %q, want %q", what, fault.Point(), point)
	}
	if fault.Outcome() != outcome {
		t.Fatalf("%s outcome = %q, want %q", what, fault.Outcome(), outcome)
	}
}

// countFiles counts installed blobs and receipts.
func countFiles(t *testing.T, store *Store) (blobs, receipts int) {
	t.Helper()
	entries, err := os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			blobs++
		}
	}
	receiptsEntries, err := os.ReadDir(filepath.Join(store.root, "operations"))
	if err != nil {
		t.Fatal(err)
	}
	return blobs, len(receiptsEntries)
}

// everyFileVerifies fails when any installed blob does not attest
// through the canonical owner: an interrupted capture must leave
// no partial record behind.
func everyFileVerifies(t *testing.T, store *Store) {
	t.Helper()
	entries, err := os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(store.root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := sessrepo.AttestCheckpointRecord(raw); err != nil {
			t.Fatalf("installed file %s does not attest: %v", entry.Name(), err)
		}
	}
}

func TestCrashBeforeDurableWriteIsSafeRetry(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	injector := &secconftest.Injector{}
	wireCrashPoints(store, injector)
	injector.Arm(secconftest.PointPrepareEnter)
	_, _, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	mustCrashFault(t, err, secconftest.PointPrepareEnter, secconftest.OutcomeSafeRetry, "Capture(prepare fault)")
	// Nothing mutated: no blob and no receipt exist.
	if blobs, receipts := countFiles(t, store); blobs != 0 || receipts != 0 {
		t.Fatalf("after prepare fault: blobs = %d, receipts = %d, want 0, 0", blobs, receipts)
	}
	// The identical retry is safe and succeeds.
	injector.Disarm()
	ref, raw, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if err != nil {
		t.Fatalf("retry Capture error = %v", err)
	}
	if _, _, err := sessrepo.AttestCheckpointRecord(raw); err != nil {
		t.Fatalf("retry bytes do not attest: %v", err)
	}
	if ref.OperationID != testOpA {
		t.Fatalf("retry OperationID = %q", ref.OperationID)
	}
}

func TestCrashBetweenBlobAndReceiptResumes(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	interrupted := errors.New("crash after blob before receipt")
	store.AfterBlob = func() error { return interrupted }
	_, _, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if !errors.Is(err, interrupted) {
		t.Fatalf("Capture(blob fault) = %v, want the interior fault", err)
	}
	// The blob is complete and verifies (never a torn prefix),
	// but no receipt exists, so the operation never committed.
	if blobs, receipts := countFiles(t, store); blobs != 1 || receipts != 0 {
		t.Fatalf("after blob fault: blobs = %d, receipts = %d, want 1, 0", blobs, receipts)
	}
	everyFileVerifies(t, store)
	// The identical retry reuses the installed bytes, completes
	// the receipt, and mints no second checkpoint identity.
	store.AfterBlob = nil
	ref, _, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if err != nil {
		t.Fatalf("retry Capture error = %v", err)
	}
	if blobs, receipts := countFiles(t, store); blobs != 1 || receipts != 1 {
		t.Fatalf("after retry: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
	}
	again, _, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if err != nil {
		t.Fatalf("replay Capture error = %v", err)
	}
	if again != ref {
		t.Fatalf("replay ref = %+v, want %+v", again, ref)
	}
}

func TestCrashAfterReceiptReplaysRecordedResult(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	committed := errors.New("crash after receipt before return")
	store.AfterCommit = func() error { return committed }
	_, _, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if !errors.Is(err, committed) {
		t.Fatalf("Capture(commit fault) = %v, want the post-commit fault", err)
	}
	// Blob and receipt are both durable: the operation committed
	// even though the caller never saw the result.
	if blobs, receipts := countFiles(t, store); blobs != 1 || receipts != 1 {
		t.Fatalf("after commit fault: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
	}
	everyFileVerifies(t, store)
	// The identical retry returns the recorded result without
	// writing: safe_retry through the receipt replay.
	store.AfterCommit = nil
	first, firstRaw, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if err != nil {
		t.Fatalf("retry Capture error = %v", err)
	}
	second, secondRaw, err := store.Capture(chain, testInputs([]string{created}, testOpA))
	if err != nil {
		t.Fatalf("replay Capture error = %v", err)
	}
	if first != second || string(firstRaw) != string(secondRaw) {
		t.Fatalf("receipt replay diverged: %+v vs %+v", first, second)
	}
	if blobs, receipts := countFiles(t, store); blobs != 1 || receipts != 1 {
		t.Fatalf("after replay: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
	}
}

func TestCaptureWithMovedInputsRefuses(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	if _, _, err := store.Capture(chain, testInputs([]string{created}, testOpA)); err != nil {
		t.Fatal(err)
	}
	// The same operation retried with moved inputs refuses
	// idempotency_mismatch and writes nothing: no second root is
	// created for a changed body.
	moved := testInputs([]string{created}, testOpA)
	moved.WorkspaceManifestID = testOtherManifest
	_, _, err := store.Capture(chain, moved)
	if !errors.Is(err, ErrCheckpointConflict) {
		t.Fatalf("Capture(moved inputs) = %v, want ErrCheckpointConflict", err)
	}
	if errors.Is(err, ErrInvalidCheckpoint) {
		t.Fatalf("Capture(moved inputs) = %v, want the conflict class, not the invalid-closure class", err)
	}
	if blobs, receipts := countFiles(t, store); blobs != 1 || receipts != 1 {
		t.Fatalf("after refused retry: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
	}
	// The same moved closure under a fresh operation is a new
	// capture, not a conflict.
	moved.OperationID = testOpB
	if _, _, err := store.Capture(chain, moved); err != nil {
		t.Fatalf("Capture(moved inputs, fresh operation) error = %v", err)
	}
	if blobs, receipts := countFiles(t, store); blobs != 2 || receipts != 2 {
		t.Fatalf("after fresh capture: blobs = %d, receipts = %d, want 2, 2", blobs, receipts)
	}
}

func TestCaptureRefusesDisagreeingDigestPath(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	ref, _ := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// Foreign bytes at the digest path: the store is torn, so the
	// identical-bytes install must refuse instead of versioning
	// an immutable record.
	if err := os.WriteFile(store.blobPath(ref.CheckpointID), []byte(`"foreign"`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := store.Capture(chain, testInputs([]string{created}, testOpB))
	if !errors.Is(err, ErrCheckpointConflict) {
		t.Fatalf("Capture(disagreeing path) = %v, want ErrCheckpointConflict", err)
	}
}
