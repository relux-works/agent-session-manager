package sessckpt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// ErrInvalidCheckpoint reports a checkpoint closure that must be
// refused before publication. It always wraps
// canonicaljson.ErrInvalidIdentity, so errors.Is proves the SPEC
// Section 5.4 incompatible_schema class for every unpublishable
// closure, including the CP-N1..CP-N4 fixtures.
var ErrInvalidCheckpoint = errors.New("invalid checkpoint closure")

// ErrCheckpointConflict reports the idempotency boundary: the same
// capture operation retried with moved inputs
// (idempotency_mismatch), or a digest path holding bytes that
// disagree with the installing candidate (a torn store, never a
// second version of an immutable record).
var ErrCheckpointConflict = errors.New("checkpoint capture conflict")

// Store durably installs Checkpoint Records and their operation
// receipts under one data root. Checkpoint blobs are
// content-addressed and installed no-replace, so capture is
// append-only by construction and there is no update or delete
// entry to test.
//
// BeforeWrite fires after validation and before the first durable
// byte; AfterBlob fires after the checkpoint blob is durable and
// before the operation receipt; AfterCommit fires after the receipt
// is durable. All three are nil in normal operation. Tests connect
// them to a secconftest Injector firing the owner's own points: a
// fault before any durable byte carries safe_retry, a fault past
// any durable step carries recoverable_parked_state until the
// identical retry replays the recorded result, and a fault past
// the receipt carries safe_retry through the replayed receipt.
type Store struct {
	root        string
	mutex       sync.Mutex
	BeforeWrite func() error
	AfterBlob   func() error
	AfterCommit func() error
}

// Open binds a checkpoint store to the checkpoints namespace beneath
// dataRoot, creating it owner-only when absent. The root must be
// absolute.
func Open(dataRoot string) (*Store, error) {
	if !filepath.IsAbs(dataRoot) {
		return nil, fmt.Errorf("%w: data root %q is not absolute", ErrInvalidCheckpoint, dataRoot)
	}
	root := filepath.Join(dataRoot, "checkpoints")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create checkpoints root %q: %w", root, err)
	}
	// The operations namespace shares the root so one data-root
	// fsync discipline covers blobs and receipts together.
	if err := os.MkdirAll(filepath.Join(root, "operations"), 0o700); err != nil {
		return nil, fmt.Errorf("create checkpoint operations root: %w", err)
	}
	return &Store{root: root}, nil
}

// CheckpointRef names one durably installed Checkpoint Record: its
// canonical digest and the operation receipt that installed it.
type CheckpointRef struct {
	CheckpointID string
	OperationID  string
}

// checkpointFileName maps a validated checkpoint digest to its blob
// file name. The digest passed scalar grammar before install;
// invalid input maps to a sentinel name that can never collide
// with a real blob.
func checkpointFileName(checkpointID string) string {
	digest, err := scalar.ParseDigest(checkpointID)
	if err != nil {
		return "invalid.json"
	}
	return digest.Hex() + ".json"
}

// operationFileName maps a validated operation UUID to its receipt
// file name. UUIDv7 grammar keeps the file name an exact,
// case-stable rendering of the operation identity.
func operationFileName(operationID string) string {
	id, err := scalar.ParseUUIDv7(operationID)
	if err != nil {
		return "invalid.json"
	}
	return id.String() + ".json"
}

// installBlob installs one content-addressed blob idempotently
// through the no-replace open first: a fresh path is created
// atomically, and only an EEXIST loser falls back to comparing
// against the winner's bytes — identical bytes are reused,
// disagreeing bytes report the torn store for refusal. Leading
// with the exclusive create (instead of stat-then-act) keeps the
// no-replace property a filesystem fact under a second process,
// not a check-then-act race the store mutex cannot see: the mutex
// serializes one process, while O_EXCL serializes all of them.
func installBlob(path string, data []byte) error {
	if err := writeExclusive(path, data); err == nil {
		return syncDirectory(filepath.Dir(path))
	} else if !os.IsExist(err) {
		return fmt.Errorf("install checkpoint blob: %w", err)
	}
	existing, readErr := os.ReadFile(path)
	if readErr != nil {
		return fmt.Errorf("read checkpoint blob for compare: %w", readErr)
	}
	if string(existing) != string(data) {
		return fmt.Errorf("%w: digest path holds disagreeing bytes", ErrCheckpointConflict)
	}
	return nil
}

// writeExclusive installs bytes at a path that must not exist. The
// no-replace open makes append-only a filesystem property, not a
// convention. Bytes fsync before close so a crash never leaves a
// torn prefix behind.
func writeExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write exclusive file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("fsync exclusive file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close exclusive file: %w", err)
	}
	return nil
}

// syncDirectory fsyncs a directory so an install inside it survives
// a crash. Opening a directory fails on Windows, where the install
// itself is the durability boundary; there the sync is skipped,
// never faked.
func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory for fsync: %w", err)
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("fsync directory: %w", err)
	}
	return nil
}
