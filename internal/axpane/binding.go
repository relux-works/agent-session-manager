package axpane

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Binding schema identity. This is an AX-internal host-local durable
// record (the §4.E host-local durable metadata class: the
// idempotency receipt for the bootstrap window), not a replicated
// contract: it is never published to the mesh and never leaves the
// host.
const (
	bindingSchema        = "urn:ax:schema:pane-bootstrap-binding"
	bindingSchemaVersion = "1.0.0"
)

// Binding is the one recorded wrapper/child for a
// (session_id, bootstrap_operation_id) pair. Identity is the
// AX-allocated terminal instance UUIDv7 plus the binding digest; a
// PID, handle, socket, path, or endpoint is never identity and never
// persisted here.
type Binding struct {
	SessionID          string `json:"session_id"`
	OperationID        string `json:"bootstrap_operation_id"`
	TerminalInstanceID string `json:"terminal_instance_id"`
	BindingDigest      string `json:"binding_digest"`
	CreatedAt          string `json:"created_at"`
}

// bindingMembers is the exact closed member set of the stored
// document.
var bindingMembers = []string{
	"schema",
	"schema_version",
	"session_id",
	"bootstrap_operation_id",
	"terminal_instance_id",
	"binding_digest",
	"created_at",
}

// Hooks arms the crash boundaries of the staged installs: after
// the receipt bytes are staged to the temporary file (before the
// atomic install), and after the install commits (before the
// directory sync makes the commit durable). The hooks fire on
// whichever install path runs: Bind's first-anchor install and
// Supersede's pair-receipt install. A nil hook is a no-op; a hook
// error aborts the call with that error and installs nothing
// further on that path.
type Hooks struct {
	AfterStage   func(path string) error
	AfterInstall func(path string) error
}

// Store is the durable bootstrap binding store rooted at one data
// directory. The session's first binding anchors at
// <root>/axpane/bootstrap/<session_id>/binding.json, and every
// recorded (session_id, bootstrap_operation_id) pair keeps its own
// receipt at
// <root>/axpane/bootstrap/<session_id>/receipts/<operation_id>.json.
// Superseded pairs keep their receipts: an identical retry of any
// recorded pair replays its own receipt.
type Store struct {
	root  string
	hooks *Hooks
	now   func() time.Time
}

// Open creates the store, ensuring the root exists. It creates no
// session directories: those appear with the first Bind.
func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("axpane bootstrap store root is empty")
	}
	base := filepath.Join(root, "axpane", "bootstrap")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, fmt.Errorf("create axpane bootstrap root: %w", err)
	}
	return &Store{root: root, now: time.Now}, nil
}

// WithHooks returns the store with crash hooks armed. It mutates the
// receiver and returns it for call chaining.
func (store *Store) WithHooks(hooks *Hooks) *Store {
	store.hooks = hooks
	return store
}

func (store *Store) sessionDir(sessionID string) string {
	return filepath.Join(store.root, "axpane", "bootstrap", sessionID)
}

func (store *Store) bindingPath(sessionID string) string {
	return filepath.Join(store.sessionDir(sessionID), "binding.json")
}

func (store *Store) receiptsDir(sessionID string) string {
	return filepath.Join(store.sessionDir(sessionID), "receipts")
}

func (store *Store) receiptPath(sessionID, operationID string) string {
	return filepath.Join(store.receiptsDir(sessionID), operationID+".json")
}

// Bind durably binds (session_id, bootstrap_operation_id) to the one
// recorded wrapper/child before any launch side effect. The first
// call for a session anchors the candidate receipt and records its
// pair receipt from the same bytes; an identical retry returns the
// stored pair receipt with reattached true; a changed operation
// refuses idempotency_mismatch while the bootstrap window is open.
// After the window closes a new operation records its own pair
// receipt through Supersede, never through this no-replace entry.
// The candidate's session and operation must equal the call
// arguments; its instance ID and binding digest are validated, and
// the store stamps created_at at install. The stored receipt is
// authoritative on retry: a retry carrying different child facts
// still reattaches to the one recorded child, never to a second
// child. Stale staging files from a pre-commit crash are swept
// before the install.
func (store *Store) Bind(sessionID, operationID string, candidate Binding) (Binding, bool, error) {
	empty := Binding{}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return empty, false, fmt.Errorf("bind session %q is not a UUIDv7: %w", sessionID, err)
	}
	if _, err := scalar.ParseUUIDv7(operationID); err != nil {
		return empty, false, fmt.Errorf("bind operation %q is not a UUIDv7: %w", operationID, err)
	}
	if candidate.SessionID != sessionID || candidate.OperationID != operationID {
		return empty, false, fmt.Errorf("bind candidate names another bootstrap pair")
	}
	if err := checkBindingCandidate(candidate); err != nil {
		return empty, false, err
	}
	pair, pairFound, err := store.Lookup(sessionID, operationID)
	if err != nil {
		return empty, false, err
	}
	if pairFound {
		return pair, true, nil
	}
	existing, found, err := store.Status(sessionID)
	if err != nil {
		return empty, false, err
	}
	if found {
		if existing.OperationID != operationID {
			return empty, false, &terminalbackend.Error{Code: terminalbackend.CodeIdempotencyMismatch, Detail: "bootstrap binding"}
		}
		// The anchor names this operation but the pair
		// receipt is absent: a crash landed between the two
		// installs. Converge the pair receipt from the
		// anchor bytes and reattach to the one child.
		converged, err := store.convergePairFromFirst(sessionID, operationID)
		if err != nil {
			return empty, false, err
		}
		return converged, true, nil
	}
	receipt, err := encodeBinding(candidate, store.timestamp())
	if err != nil {
		return empty, false, err
	}
	if err := store.install(sessionID, operationID, receipt); err != nil {
		var concurrent errConcurrentWinner
		if errors.As(err, &concurrent) {
			return concurrent.bound, true, nil
		}
		return empty, false, err
	}
	stored, found, err := store.Lookup(sessionID, operationID)
	if err != nil {
		return empty, false, err
	}
	if !found {
		return empty, false, errors.New("bootstrap binding vanished after install")
	}
	return stored, false, nil
}

// Status proves absence or identifies the session's first recorded
// wrapper/child: found is true with the anchor receipt, or false
// when no binding exists. A read or validation failure is an
// error, never absence: unknown is not absent. Pair-scoped replay
// reads through Lookup: superseded pairs keep their receipts while
// the anchor keeps pointing at the first.
func (store *Store) Status(sessionID string) (Binding, bool, error) {
	empty := Binding{}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return empty, false, fmt.Errorf("binding status session %q is not a UUIDv7: %w", sessionID, err)
	}
	raw, err := os.ReadFile(store.bindingPath(sessionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return empty, false, nil
		}
		return empty, false, fmt.Errorf("read bootstrap binding: %w", err)
	}
	binding, err := decodeBinding(raw)
	if err != nil {
		return empty, false, err
	}
	if binding.SessionID != sessionID {
		return empty, false, errors.New("bootstrap binding names another session")
	}
	return binding, true, nil
}

// Lookup proves absence or replays the one recorded wrapper/child
// for a (session_id, bootstrap_operation_id) pair: found is true
// with that pair's own receipt, or false when the pair is
// unrecorded. A read or validation failure is an error, never
// absence: unknown is not absent. An identical retry of any
// recorded pair replays its receipt here, including pairs later
// pairs superseded.
func (store *Store) Lookup(sessionID, operationID string) (Binding, bool, error) {
	empty := Binding{}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return empty, false, fmt.Errorf("binding lookup session %q is not a UUIDv7: %w", sessionID, err)
	}
	if _, err := scalar.ParseUUIDv7(operationID); err != nil {
		return empty, false, fmt.Errorf("binding lookup operation %q is not a UUIDv7: %w", operationID, err)
	}
	raw, err := os.ReadFile(store.receiptPath(sessionID, operationID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return empty, false, nil
		}
		return empty, false, fmt.Errorf("read bootstrap pair receipt: %w", err)
	}
	binding, err := decodeBinding(raw)
	if err != nil {
		return empty, false, err
	}
	if binding.SessionID != sessionID {
		return empty, false, errors.New("bootstrap pair receipt names another session")
	}
	if binding.OperationID != operationID {
		return empty, false, errors.New("bootstrap pair receipt names another operation")
	}
	return binding, true, nil
}

// SessionBound reports whether the session has any recorded
// binding: the first-binding anchor or any pair receipt. A torn
// anchor fails the read (unknown, never unbound); pair receipts
// count by presence, since each pair validates on its own Lookup.
func (store *Store) SessionBound(sessionID string) (bool, error) {
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return false, fmt.Errorf("binding session %q is not a UUIDv7: %w", sessionID, err)
	}
	_, found, err := store.Status(sessionID)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}
	matches, err := filepath.Glob(filepath.Join(store.receiptsDir(sessionID), "*.json"))
	if err != nil {
		return false, err
	}
	return len(matches) > 0, nil
}

func (store *Store) timestamp() string {
	now := time.Now().UTC()
	if store.now != nil {
		now = store.now().UTC()
	}
	return now.Format("2006-01-02T15:04:05.000Z07:00")
}

// install stages the receipt to a same-directory temporary file,
// fsyncs it, atomically links it to the binding path (which must
// not exist: the link is the no-replace commit), removes the
// staging name, and fsyncs the directory. Readers see absence or
// the complete receipt, never a torn prefix: a crash before the
// link leaves only ignored staging garbage, and a crash after it
// leaves the complete binding the identical retry reattaches to.
// Any pre-existing binding refuses without replacement.
// errConcurrentWinner carries the winning receipt when a concurrent
// wrapper commits first: the loser reattaches instead of failing.
type errConcurrentWinner struct {
	bound Binding
}

func (err errConcurrentWinner) Error() string {
	return "bootstrap binding concurrently committed by another wrapper"
}

// Supersede durably records a post-window bootstrap pair: the
// caller proved (through the fold's newest checkpoint) that a new
// bootstrap operation is legitimate, and the store installs that
// pair's own receipt no-replace. Superseded pairs keep their
// receipts — nothing is renamed over — so an identical retry of
// any recorded pair replays its own receipt, and two post-window
// Supersede calls with different operations each record their own
// child. A recorded pair replays idempotently: Supersede never
// replaces a pair receipt, and reports the replay so the caller
// reattaches to the recorded child instead of reporting a launch
// of a child it did not create. The session's first-binding
// anchor is claimed when absent so the anchor always names the
// first recorded pair. Supersede refuses malformed candidates
// exactly like Bind.
func (store *Store) Supersede(sessionID, operationID string, candidate Binding) (Binding, bool, error) {
	empty := Binding{}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return empty, false, fmt.Errorf("bind session %q is not a UUIDv7: %w", sessionID, err)
	}
	if _, err := scalar.ParseUUIDv7(operationID); err != nil {
		return empty, false, fmt.Errorf("bind operation %q is not a UUIDv7: %w", operationID, err)
	}
	if candidate.SessionID != sessionID || candidate.OperationID != operationID {
		return empty, false, fmt.Errorf("bind candidate names another bootstrap pair")
	}
	if err := checkBindingCandidate(candidate); err != nil {
		return empty, false, err
	}
	stored, found, err := store.Lookup(sessionID, operationID)
	if err != nil {
		return empty, false, err
	}
	replayed := found
	if !found {
		receipt, err := encodeBinding(candidate, store.timestamp())
		if err != nil {
			return empty, false, err
		}
		stored, replayed, err = store.installPair(sessionID, operationID, receipt)
		if err != nil {
			return empty, false, err
		}
	}
	// The anchor claim runs on every return path, including the
	// idempotent replay: a crash between the pair install and
	// the anchor claim converges on retry.
	if err := store.ensureFirstFromPair(sessionID, operationID); err != nil {
		return empty, false, err
	}
	return stored, replayed, nil
}

// installPair stages the pair receipt and atomically links it to
// the pair path, which must not exist: the link is the no-replace
// commit. Readers see absence or the complete receipt, never a
// torn prefix: a crash before the link leaves only ignored staging
// garbage, and a crash after it leaves the complete receipt the
// identical retry replays. A concurrent same-pair commit wins and
// the loser replays the winner's receipt, reported so the caller
// reattaches. Staging garbage from a pre-commit crash is swept
// first.
func (store *Store) installPair(sessionID, operationID string, receipt []byte) (Binding, bool, error) {
	empty := Binding{}
	dir := store.receiptsDir(sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return empty, false, fmt.Errorf("create bootstrap receipts directory: %w", err)
	}
	sweepReceiptsStaging(dir)
	target := store.receiptPath(sessionID, operationID)
	staged, err := os.CreateTemp(dir, "receipt-*.tmp")
	if err != nil {
		return empty, false, fmt.Errorf("stage bootstrap pair receipt: %w", err)
	}
	stagedPath := staged.Name()
	defer func() { _ = os.Remove(stagedPath) }()
	if err := staged.Chmod(0o600); err != nil {
		_ = staged.Close()
		return empty, false, fmt.Errorf("protect bootstrap pair receipt: %w", err)
	}
	if _, err := staged.Write(receipt); err != nil {
		_ = staged.Close()
		return empty, false, fmt.Errorf("write bootstrap pair receipt: %w", err)
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return empty, false, fmt.Errorf("sync bootstrap pair receipt: %w", err)
	}
	if err := staged.Close(); err != nil {
		return empty, false, fmt.Errorf("close bootstrap pair receipt: %w", err)
	}
	if store.hooks != nil && store.hooks.AfterStage != nil {
		if err := store.hooks.AfterStage(stagedPath); err != nil {
			return empty, false, err
		}
	}
	if err := os.Link(stagedPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			// A concurrent wrapper recorded this pair
			// between our Lookup and our link: replay the
			// winner's receipt.
			winner, found, readErr := store.Lookup(sessionID, operationID)
			if readErr != nil {
				return empty, false, readErr
			}
			if !found {
				return empty, false, errors.New("bootstrap pair receipt vanished after concurrent commit")
			}
			return winner, true, nil
		}
		return empty, false, fmt.Errorf("install bootstrap pair receipt: %w", err)
	}
	_ = os.Remove(stagedPath)
	if store.hooks != nil && store.hooks.AfterInstall != nil {
		if err := store.hooks.AfterInstall(target); err != nil {
			return empty, false, err
		}
	}
	if err := syncDirectory(dir); err != nil {
		return empty, false, err
	}
	if err := syncDirectory(store.sessionDir(sessionID)); err != nil {
		return empty, false, err
	}
	stored, found, err := store.Lookup(sessionID, operationID)
	if err != nil {
		return empty, false, err
	}
	if !found {
		return empty, false, errors.New("bootstrap pair receipt vanished after install")
	}
	return stored, false, nil
}

// convergePairFromFirst records the pair receipt from the anchor
// bytes after a crash landed between the anchor install and the
// pair install. The anchor names this operation (checked by the
// caller): the pair receipt carries the identical bytes, so the
// anchor and the pair agree byte for byte.
func (store *Store) convergePairFromFirst(sessionID, operationID string) (Binding, error) {
	empty := Binding{}
	raw, err := os.ReadFile(store.bindingPath(sessionID))
	if err != nil {
		return empty, fmt.Errorf("read bootstrap binding for pair convergence: %w", err)
	}
	binding, err := decodeBinding(raw)
	if err != nil {
		return empty, err
	}
	if binding.SessionID != sessionID || binding.OperationID != operationID {
		return empty, fmt.Errorf("bind anchor names another bootstrap pair")
	}
	converged, _, err := store.installPair(sessionID, operationID, raw)
	if err != nil {
		return empty, err
	}
	return converged, nil
}

// ensureFirstFromPair claims the session's first-binding anchor
// from an already-recorded pair receipt when the anchor is absent.
// The link carries the no-replace guard: a concurrent claim wins
// and this call converges to it. It is a no-op when the anchor
// exists.
func (store *Store) ensureFirstFromPair(sessionID, operationID string) error {
	if _, err := os.Lstat(store.bindingPath(sessionID)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat bootstrap binding: %w", err)
	}
	if err := os.Link(store.receiptPath(sessionID, operationID), store.bindingPath(sessionID)); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("claim first bootstrap binding: %w", err)
	}
	return syncDirectory(store.sessionDir(sessionID))
}

// sweepStaging removes stale binding-*.tmp files left by a
// pre-commit crash. Only files older than a minute are removed: a
// concurrent live wrapper stages under the same pattern, and an
// age gate keeps the sweep from deleting its in-flight receipt. It
// is best-effort: a sweep failure never fails the binding, and
// only the staging pattern is removed.
func sweepStaging(dir string) {
	sweepStagingPattern(dir, "binding-*.tmp")
}

// sweepReceiptsStaging removes stale receipt-*.tmp files left by a
// pre-commit pair-receipt crash, under the same age gate.
func sweepReceiptsStaging(dir string) {
	sweepStagingPattern(dir, "receipt-*.tmp")
}

func sweepStagingPattern(dir, pattern string) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-time.Minute)
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(match)
	}
}

func (store *Store) install(sessionID, operationID string, receipt []byte) error {
	dir := store.sessionDir(sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create bootstrap session directory: %w", err)
	}
	sweepStaging(dir)
	target := store.bindingPath(sessionID)
	staged, err := os.CreateTemp(dir, "binding-*.tmp")
	if err != nil {
		return fmt.Errorf("stage bootstrap binding: %w", err)
	}
	stagedPath := staged.Name()
	defer func() { _ = os.Remove(stagedPath) }()
	if err := staged.Chmod(0o600); err != nil {
		_ = staged.Close()
		return fmt.Errorf("protect bootstrap binding: %w", err)
	}
	if _, err := staged.Write(receipt); err != nil {
		_ = staged.Close()
		return fmt.Errorf("write bootstrap binding: %w", err)
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return fmt.Errorf("sync bootstrap binding: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close bootstrap binding: %w", err)
	}
	if store.hooks != nil && store.hooks.AfterStage != nil {
		if err := store.hooks.AfterStage(stagedPath); err != nil {
			return err
		}
	}
	if err := os.Link(stagedPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			// A concurrent wrapper won the commit between our
			// Status and our link: re-read the winner. The
			// same operation converges its pair receipt from
			// the anchor bytes and reattaches to it; a changed
			// operation is the bootstrap-window mismatch.
			winner, found, readErr := store.Status(sessionID)
			if readErr != nil {
				return readErr
			}
			if !found {
				return &terminalbackend.Error{Code: terminalbackend.CodeIdempotencyMismatch, Detail: "bootstrap binding"}
			}
			if winner.OperationID != operationID {
				return &terminalbackend.Error{Code: terminalbackend.CodeIdempotencyMismatch, Detail: "bootstrap binding"}
			}
			converged, err := store.convergePairFromFirst(sessionID, operationID)
			if err != nil {
				return err
			}
			return errConcurrentWinner{bound: converged}
		}
		return fmt.Errorf("install bootstrap binding: %w", err)
	}
	_ = os.Remove(stagedPath)
	if store.hooks != nil && store.hooks.AfterInstall != nil {
		if err := store.hooks.AfterInstall(target); err != nil {
			return err
		}
	}
	if err := syncDirectory(dir); err != nil {
		return err
	}
	// The anchor committed: record the pair receipt from the same
	// bytes. A failure here still leaves the committed anchor;
	// the retry converges the pair from it.
	if _, _, err := store.installPair(sessionID, operationID, receipt); err != nil {
		return err
	}
	return nil
}

func checkBindingCandidate(candidate Binding) error {
	if _, err := scalar.ParseUUIDv7(candidate.SessionID); err != nil {
		return fmt.Errorf("binding session %q is not a UUIDv7: %w", candidate.SessionID, err)
	}
	if _, err := scalar.ParseUUIDv7(candidate.OperationID); err != nil {
		return fmt.Errorf("binding operation %q is not a UUIDv7: %w", candidate.OperationID, err)
	}
	if _, err := scalar.ParseUUIDv7(candidate.TerminalInstanceID); err != nil {
		return fmt.Errorf("binding terminal instance %q is not a UUIDv7: %w", candidate.TerminalInstanceID, err)
	}
	if _, err := scalar.ParseDigest(candidate.BindingDigest); err != nil {
		return fmt.Errorf("binding digest is not a digest: %w", err)
	}
	// The candidate carries no timestamp: the store stamps
	// created_at at install, so a caller clock never enters the
	// receipt. Stored receipts validate their stamp at decode.
	return nil
}

func encodeBinding(candidate Binding, createdAt string) ([]byte, error) {
	stored := candidate
	stored.CreatedAt = createdAt
	document := map[string]string{
		"schema":                 bindingSchema,
		"schema_version":         bindingSchemaVersion,
		"session_id":             stored.SessionID,
		"bootstrap_operation_id": stored.OperationID,
		"terminal_instance_id":   stored.TerminalInstanceID,
		"binding_digest":         stored.BindingDigest,
		"created_at":             stored.CreatedAt,
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode bootstrap binding: %w", err)
	}
	return raw, nil
}

func decodeBinding(raw []byte) (Binding, error) {
	empty := Binding{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document struct {
		Schema             string `json:"schema"`
		SchemaVersion      string `json:"schema_version"`
		SessionID          string `json:"session_id"`
		OperationID        string `json:"bootstrap_operation_id"`
		TerminalInstanceID string `json:"terminal_instance_id"`
		BindingDigest      string `json:"binding_digest"`
		CreatedAt          string `json:"created_at"`
	}
	if err := decoder.Decode(&document); err != nil {
		return empty, fmt.Errorf("bootstrap binding is not the closed shape: %w", err)
	}
	if decoder.More() {
		return empty, fmt.Errorf("bootstrap binding carries trailing data")
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return empty, fmt.Errorf("bootstrap binding is not an object: %w", err)
	}
	for _, name := range bindingMembers {
		if _, ok := members[name]; !ok {
			return empty, fmt.Errorf("bootstrap binding misses required member %q", name)
		}
	}
	if document.Schema != bindingSchema || document.SchemaVersion != bindingSchemaVersion {
		return empty, fmt.Errorf("bootstrap binding schema is not %s %s", bindingSchema, bindingSchemaVersion)
	}
	binding := Binding{
		SessionID:          document.SessionID,
		OperationID:        document.OperationID,
		TerminalInstanceID: document.TerminalInstanceID,
		BindingDigest:      document.BindingDigest,
		CreatedAt:          document.CreatedAt,
	}
	if err := checkBindingCandidate(binding); err != nil {
		return empty, err
	}
	if _, err := scalar.ParseTimestamp(binding.CreatedAt); err != nil {
		return empty, fmt.Errorf("binding timestamp is not a timestamp: %w", err)
	}
	return binding, nil
}

// BindingDigest computes the opaque binding digest over the canonical
// receipt bytes: the value a §7.A descriptor carries as
// terminal_binding_id. It reveals no generation, PID, or native
// reference.
func BindingDigest(binding Binding) string {
	sum := sha256.Sum256([]byte(binding.SessionID + "\x00" + binding.OperationID + "\x00" + binding.TerminalInstanceID))
	return "sha256:" + hex.EncodeToString(sum[:])
}
