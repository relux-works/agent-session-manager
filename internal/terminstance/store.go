package terminstance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Receipt is one durable idempotency record: the canonical key, the
// operation it was bound for, and the operation ID it was bound with.
// The shape extends the landed terminalbackend.Receipt triple
// (Key/Operation/ResultID); ResultID is the MutationContext operation
// ID, so an identical retry (same key, same operation, same operation
// ID) replays and a changed operation in the window is
// idempotency_mismatch.
type Receipt struct {
	Key         string
	Operation   terminalbackend.Operation
	OperationID string
}

// StoreHooks arms the crash seams of the receipt store. AfterCommit runs
// after the no-replace link lands and before the directory sync, exactly
// like the predecessor leaf's AfterInstall: a hook error propagates to
// the caller but the receipt stands, so the identical retry reconciles
// through the production status entry — it replays when a completion
// exists, resumes under the same receipt when status proves the source,
// and refuses uncertain otherwise — instead of binding a second receipt.
type StoreHooks struct {
	AfterCommit func(path string) error
}

// ReceiptStore is the durable idempotency receipt table backing the §4.C
// recovery rules: persist the receipt before the first side effect,
// identical retry replays it, a changed operation in the window is
// idempotency_mismatch, and uncertainty requires status. Each receipt is
// one file committed by temp-stage, fsync, no-replace link and directory
// sync; readers see absence or the complete receipt, never a torn
// prefix. Completion (the recorded result) is a second no-replace file,
// because updating the receipt in place would break no-replace: a
// pending receipt without its completion proves an interrupted attempt,
// never a result, and the engine reconciles its retry through the
// production status entry: the same operation resumes under the same
// receipt when status proves the source, and refuses uncertain
// otherwise. No path binds a second receipt for one key.
type ReceiptStore struct {
	mutex sync.Mutex
	base  string
	hooks *StoreHooks
}

// OpenReceiptStore opens (creating) the receipt store rooted at base.
func OpenReceiptStore(base string) (*ReceiptStore, error) {
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, fmt.Errorf("create receipt store: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "pending"), 0o755); err != nil {
		return nil, fmt.Errorf("create receipt pending directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "done"), 0o755); err != nil {
		return nil, fmt.Errorf("create receipt done directory: %w", err)
	}
	return &ReceiptStore{base: base}, nil
}

// WithHooks arms the crash seams. A nil hooks value disarms them.
func (store *ReceiptStore) WithHooks(hooks *StoreHooks) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.hooks = hooks
}

// receiptName maps a key to its file name: the hex sha256 of the key.
// The key itself is embedded in the file content, so the name is a
// lookup optimization the content re-validates, never trusted identity.
func receiptName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// encodeReceipt renders one receipt as the landed per-receipt triple:
// key, operation and operation ID on three lines. The encoding matches
// terminalbackend.Ledger.Export line for line, so Export below produces
// bytes ImportLedger admits.
func encodeReceipt(receipt Receipt) []byte {
	return []byte(receipt.Key + "\n" + string(receipt.Operation) + "\n" + receipt.OperationID + "\n")
}

// decodeReceipt parses one receipt file. A torn or malformed file is an
// integrity failure, never absence and never a partial receipt.
func decodeReceipt(raw []byte) (Receipt, error) {
	lines := strings.Split(string(raw), "\n")
	if len(lines) != 4 || lines[3] != "" || lines[0] == "" || lines[2] == "" {
		return Receipt{}, refuse(CodeIntegrityFailure, "idempotency receipt image")
	}
	operation, err := terminalbackend.ParseOperation(lines[1])
	if err != nil {
		return Receipt{}, refuse(CodeIntegrityFailure, "idempotency receipt image")
	}
	return Receipt{Key: lines[0], Operation: operation, OperationID: lines[2]}, nil
}

// Bind records the receipt for a key before the first side effect. A key
// bound for the first time is durably stored and reported fresh. A key
// bound again with the identical operation and operation ID replays the
// stored receipt. A key bound again with a different operation or
// operation ID is idempotency_mismatch: only a successful status read
// proves absence, never a second binding.
func (store *ReceiptStore) Bind(key string, operation terminalbackend.Operation, operationID string) (Receipt, bool, error) {
	if store == nil {
		return Receipt{}, false, refuse(CodeProtocolError, "idempotency store unavailable")
	}
	if key == "" || operationID == "" {
		return Receipt{}, false, refuse(CodeProtocolError, "idempotency key shape")
	}
	if _, err := terminalbackend.ParseOperation(string(operation)); err != nil {
		return Receipt{}, false, wrapLanded(err)
	}
	dir := filepath.Join(store.base, "pending")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Receipt{}, false, fmt.Errorf("create receipt pending directory: %w", err)
	}
	sweepStaging(dir)
	target := filepath.Join(dir, receiptName(key))
	receipt := Receipt{Key: key, Operation: operation, OperationID: operationID}
	staged, err := os.CreateTemp(dir, "receipt-*.tmp")
	if err != nil {
		return Receipt{}, false, fmt.Errorf("stage idempotency receipt: %w", err)
	}
	stagedPath := staged.Name()
	defer func() { _ = os.Remove(stagedPath) }()
	if err := staged.Chmod(0o600); err != nil {
		_ = staged.Close()
		return Receipt{}, false, fmt.Errorf("protect idempotency receipt: %w", err)
	}
	if _, err := staged.Write(encodeReceipt(receipt)); err != nil {
		_ = staged.Close()
		return Receipt{}, false, fmt.Errorf("write idempotency receipt: %w", err)
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return Receipt{}, false, fmt.Errorf("sync idempotency receipt: %w", err)
	}
	if err := staged.Close(); err != nil {
		return Receipt{}, false, fmt.Errorf("close idempotency receipt: %w", err)
	}
	if err := os.Link(stagedPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			winner, found, readErr := store.Lookup(key)
			if readErr != nil {
				return Receipt{}, false, readErr
			}
			if !found {
				return Receipt{}, false, refuse(CodeIntegrityFailure, "idempotency receipt image")
			}
			if winner.Operation != operation || winner.OperationID != operationID {
				return Receipt{}, false, refuse(CodeIdempotencyMismatch, "idempotency key conflict")
			}
			return winner, true, nil
		}
		return Receipt{}, false, fmt.Errorf("install idempotency receipt: %w", err)
	}
	_ = os.Remove(stagedPath)
	store.mutex.Lock()
	hooks := store.hooks
	store.mutex.Unlock()
	if hooks != nil && hooks.AfterCommit != nil {
		if err := hooks.AfterCommit(target); err != nil {
			return Receipt{}, false, err
		}
	}
	if err := syncDirectory(dir); err != nil {
		return Receipt{}, false, err
	}
	return receipt, false, nil
}

// Lookup returns the stored receipt for a key. Absence of a receipt
// proves nothing and reports false; a torn receipt file is an integrity
// failure, never absence.
func (store *ReceiptStore) Lookup(key string) (Receipt, bool, error) {
	if store == nil || key == "" {
		return Receipt{}, false, nil
	}
	target := filepath.Join(store.base, "pending", receiptName(key))
	raw, err := os.ReadFile(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Receipt{}, false, nil
		}
		return Receipt{}, false, fmt.Errorf("read idempotency receipt: %w", err)
	}
	receipt, err := decodeReceipt(raw)
	if err != nil {
		return Receipt{}, false, err
	}
	if receipt.Key != key {
		return Receipt{}, false, refuse(CodeIntegrityFailure, "idempotency receipt image")
	}
	return receipt, true, nil
}

// Complete records the result of a bound key. The result bytes commit
// through the same no-replace link as the receipt: a second completion
// with identical bytes replays, a second completion with different bytes
// is an integrity failure, because one attempt has exactly one result.
// Completing an unbound key is a failed local precondition.
func (store *ReceiptStore) Complete(key string, result []byte) error {
	if store == nil {
		return refuse(CodeProtocolError, "idempotency store unavailable")
	}
	if _, found, err := store.Lookup(key); err != nil {
		return err
	} else if !found {
		return refuse(CodePreconditionFailed, "idempotency completion without receipt")
	}
	dir := filepath.Join(store.base, "done")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create receipt done directory: %w", err)
	}
	sweepStaging(dir)
	target := filepath.Join(dir, receiptName(key))
	staged, err := os.CreateTemp(dir, "result-*.tmp")
	if err != nil {
		return fmt.Errorf("stage idempotency result: %w", err)
	}
	stagedPath := staged.Name()
	defer func() { _ = os.Remove(stagedPath) }()
	if err := staged.Chmod(0o600); err != nil {
		_ = staged.Close()
		return fmt.Errorf("protect idempotency result: %w", err)
	}
	if _, err := staged.Write(result); err != nil {
		_ = staged.Close()
		return fmt.Errorf("write idempotency result: %w", err)
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return fmt.Errorf("sync idempotency result: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close idempotency result: %w", err)
	}
	if err := os.Link(stagedPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			winner, readErr := os.ReadFile(target)
			if readErr != nil {
				return fmt.Errorf("read idempotency result: %w", readErr)
			}
			if string(winner) != string(result) {
				return refuse(CodeIntegrityFailure, "idempotency result conflict")
			}
			return nil
		}
		return fmt.Errorf("install idempotency result: %w", err)
	}
	_ = os.Remove(stagedPath)
	if err := syncDirectory(dir); err != nil {
		return err
	}
	return nil
}

// Completed returns the recorded result for a key. A bound key without
// its completion reports found false: the attempt was interrupted and
// only status recovers the observation, never a second binding.
func (store *ReceiptStore) Completed(key string) ([]byte, bool, error) {
	if store == nil || key == "" {
		return nil, false, nil
	}
	target := filepath.Join(store.base, "done", receiptName(key))
	raw, err := os.ReadFile(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read idempotency result: %w", err)
	}
	return raw, true, nil
}

// Export renders the pending table as stable bytes: receipts sorted by
// key, one triple per receipt. The bytes equal what a
// terminalbackend.Ledger holding the same receipts exports, so they feed
// ImportLedger directly; the agreement is pinned by
// TestStoreExportAgreesWithLedger. Staging garbage is never exported.
func (store *ReceiptStore) Export() ([]byte, error) {
	if store == nil {
		return nil, refuse(CodeProtocolError, "idempotency store unavailable")
	}
	dir := filepath.Join(store.base, "pending")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read receipt table: %w", err)
	}
	receipts := make([]Receipt, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isReceiptName(entry.Name()) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read idempotency receipt: %w", err)
		}
		receipt, err := decodeReceipt(raw)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].Key < receipts[j].Key })
	var image strings.Builder
	for _, receipt := range receipts {
		image.WriteString(receipt.Key + "\n" + string(receipt.Operation) + "\n" + receipt.OperationID + "\n")
	}
	return []byte(image.String()), nil
}

// isReceiptName reports whether a directory entry is a committed receipt
// file: exactly 64 lowercase hex characters. Staging files
// (receipt-*.tmp, result-*.tmp) and anything else are ignored.
func isReceiptName(name string) bool {
	if len(name) != 64 {
		return false
	}
	for i := 0; i < len(name); i++ {
		digit := name[i]
		if (digit < '0' || digit > '9') && (digit < 'a' || digit > 'f') {
			return false
		}
	}
	return true
}

// stagingSweepAge is the minimum age of a staging file the sweep may
// remove. A concurrent Bind stages its own *.tmp between our stage and
// our link; sweeping only files older than this bound keeps a live
// stage alive, while crash garbage (whose mtime stops at the crash)
// ages out on a later call. Readers and Export ignore staging names,
// so a young leftover is invisible until it ages.
const stagingSweepAge = time.Minute

// sweepStaging removes staging garbage a pre-commit crash left behind.
// Staging files carry the *.tmp suffix; committed receipts never do, so
// the sweep cannot remove a commit. Only files older than
// stagingSweepAge are removed, so a concurrent live stage is never
// deleted: without the age gate a racing Bind would lose its staged
// file and fail its no-replace link with a spurious install error.
func sweepStaging(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) < stagingSweepAge {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

// syncDirectory fsyncs a directory so committed links survive a crash.
func syncDirectory(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open receipt directory: %w", err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync receipt directory: %w", err)
	}
	return nil
}
