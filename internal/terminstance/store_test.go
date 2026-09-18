package terminstance

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func testStore(t *testing.T) *ReceiptStore {
	t.Helper()
	store, err := OpenReceiptStore(t.TempDir())
	if err != nil {
		t.Fatalf("OpenReceiptStore() error = %v", err)
	}
	return store
}

// TestStoreBindReplayMismatch proves the receipt seam through the
// production entry: the first bind is fresh, the identical retry
// replays, and a changed operation or operation ID in the window is
// idempotency_mismatch with the landed literal.
func TestStoreBindReplayMismatch(t *testing.T) {
	store := testStore(t)
	key := fixtureInstance + "/quiesce/" + fixtureQuiescence

	receipt, replayed, err := store.Bind(key, terminalbackend.OperationQuiesceInput, fixtureOperation)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if replayed {
		t.Error("Bind() replayed a fresh key, want fresh")
	}
	requireLiteral(t, "key", receipt.Key, key)

	again, replayed, err := store.Bind(key, terminalbackend.OperationQuiesceInput, fixtureOperation)
	if err != nil {
		t.Fatalf("Bind() identical retry error = %v", err)
	}
	if !replayed {
		t.Error("Bind() identical retry installs fresh, want replay")
	}
	if again != receipt {
		t.Errorf("Bind() replay = %+v, want %+v", again, receipt)
	}

	_, _, err = store.Bind(key, terminalbackend.OperationWaitSafeBoundary, fixtureOperation)
	requireRefusal(t, err, "idempotency_mismatch", "idempotency key conflict")

	_, _, err = store.Bind(key, terminalbackend.OperationQuiesceInput, fixtureOperationB)
	requireRefusal(t, err, "idempotency_mismatch", "idempotency key conflict")
}

// TestStoreLookupAbsenceIsNotFailure proves a missing receipt reports
// absence without an error, while a torn receipt file is an integrity
// failure, never absence.
func TestStoreLookupAbsenceIsNotFailure(t *testing.T) {
	store := testStore(t)
	if _, found, err := store.Lookup("no/such/key"); err != nil || found {
		t.Errorf("Lookup(absent) = (%v, %v), want (false, nil)", found, err)
	}

	key := "torn/key"
	if _, _, err := store.Bind(key, terminalbackend.OperationStatus, fixtureOperation); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	// Corrupt the committed file in place: the link is immutable but
	// the bytes are host-local, so corruption must read corrupt.
	entries, err := os.ReadDir(filepath.Join(store.base, "pending"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if len(entry.Name()) == 64 {
			if err := os.WriteFile(filepath.Join(store.base, "pending", entry.Name()), []byte("torn"), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
		}
	}
	_, _, err = store.Lookup(key)
	requireRefusal(t, err, "terminal_backend_integrity_failure", "idempotency receipt image")
}

// TestStoreCompleteRequiresReceipt proves completion binds only a bound
// key: identical completion replays, divergent completion is an
// integrity failure, and completing an unbound key is a failed
// precondition.
func TestStoreCompleteRequiresReceipt(t *testing.T) {
	store := testStore(t)
	key := fixtureInstance + "/quiesce/" + fixtureQuiescence

	requireRefusal(t, store.Complete(key, []byte(`{}`)), "local_precondition_failed", "idempotency completion without receipt")

	if _, _, err := store.Bind(key, terminalbackend.OperationQuiesceInput, fixtureOperation); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if _, found, err := store.Completed(key); err != nil || found {
		t.Fatalf("Completed() = (%v, %v), want (false, nil)", found, err)
	}
	if err := store.Complete(key, []byte(`{"after":"quiescing"}`)); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	stored, found, err := store.Completed(key)
	if err != nil || !found {
		t.Fatalf("Completed() = (%v, %v), want the result", found, err)
	}
	requireLiteral(t, "result", string(stored), `{"after":"quiescing"}`)
	if err := store.Complete(key, []byte(`{"after":"quiescing"}`)); err != nil {
		t.Fatalf("Complete() identical error = %v", err)
	}
	requireRefusal(t, store.Complete(key, []byte(`{"after":"stopped"}`)), "terminal_backend_integrity_failure", "idempotency result conflict")
}

// TestStoreExportAgreesWithLedger proves the store encoding is the
// landed ledger encoding: the exported bytes import through the landed
// ImportLedger and replay the same receipts, and staging garbage is
// never exported.
func TestStoreExportAgreesWithLedger(t *testing.T) {
	store := testStore(t)
	keys := []string{"b/key", "a/key"}
	for _, key := range keys {
		if _, _, err := store.Bind(key, terminalbackend.OperationStatus, fixtureOperation); err != nil {
			t.Fatalf("Bind(%q) error = %v", key, err)
		}
	}
	if err := os.WriteFile(filepath.Join(store.base, "pending", "receipt-stale.tmp"), []byte("garbage"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	image, err := store.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	requireLiteral(t, "export", string(image),
		"a/key\nstatus\n"+fixtureOperation+"\n"+"b/key\nstatus\n"+fixtureOperation+"\n")

	ledger, err := terminalbackend.ImportLedger(image)
	if err != nil {
		t.Fatalf("ImportLedger(store export) error = %v", err)
	}
	for _, key := range keys {
		receipt, known := ledger.Replay(key)
		if !known {
			t.Errorf("Replay(%q) unknown, want the receipt", key)
			continue
		}
		if receipt.Operation != terminalbackend.OperationStatus || receipt.ResultID != fixtureOperation {
			t.Errorf("Replay(%q) = %+v, want the bound triple", key, receipt)
		}
	}
	fresh := terminalbackend.NewLedger()
	for _, key := range keys {
		if _, err := fresh.Bind(key, terminalbackend.OperationStatus, fixtureOperation); err != nil {
			t.Fatalf("Ledger.Bind(%q) error = %v", key, err)
		}
	}
	if string(fresh.Export()) != string(image) {
		t.Errorf("ledger export = %q, store export = %q, want byte equality", string(fresh.Export()), string(image))
	}
}

// TestStoreSweepKeepsLiveStaging proves the staging sweep is age-gated:
// a fresh *.tmp file — a concurrent Bind's live stage — survives a
// Bind on the same store instead of being deleted mid-stage.
func TestStoreSweepKeepsLiveStaging(t *testing.T) {
	store := testStore(t)
	live := filepath.Join(store.base, "pending", "receipt-live.tmp")
	if err := os.WriteFile(live, []byte("live stage"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, _, err := store.Bind("fresh/key", terminalbackend.OperationStatus, fixtureOperation); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("live staging after Bind: %v, want the file kept", err)
	}
}

// TestStoreSweepRemovesAgedStaging proves crash garbage still ages out:
// a *.tmp file older than the sweep bound is removed by the next Bind.
func TestStoreSweepRemovesAgedStaging(t *testing.T) {
	store := testStore(t)
	stale := filepath.Join(store.base, "pending", "receipt-stale.tmp")
	if err := os.WriteFile(stale, []byte("crash garbage"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	aged := time.Now().Add(-2 * stagingSweepAge)
	if err := os.Chtimes(stale, aged, aged); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
	if _, _, err := store.Bind("fresh/key", terminalbackend.OperationStatus, fixtureOperation); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("aged staging after Bind: %v, want removal", err)
	}
}

// TestStoreHookErrorKeepsReceipt proves the AfterCommit seam: a hook
// error propagates to the caller but the committed receipt stands, so
// the identical retry replays it.
func TestStoreHookErrorKeepsReceipt(t *testing.T) {
	store := testStore(t)
	key := fixtureInstance + "/quiesce/" + fixtureQuiescence
	hookErr := errTestHook
	store.WithHooks(&StoreHooks{AfterCommit: func(string) error { return hookErr }})
	if _, _, err := store.Bind(key, terminalbackend.OperationQuiesceInput, fixtureOperation); err != hookErr {
		t.Fatalf("Bind() error = %v, want the hook error", err)
	}
	store.WithHooks(nil)
	if _, replayed, err := store.Bind(key, terminalbackend.OperationQuiesceInput, fixtureOperation); err != nil || !replayed {
		t.Fatalf("Bind() retry = (%v, %v), want (replay, nil)", replayed, err)
	}
}

type hookError string

func (err hookError) Error() string { return string(err) }

const errTestHook = hookError("test hook failure")
