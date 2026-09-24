package tmuxserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// InstanceStates is the durable AX-side state memory for tmux-hosted
// terminal instances: one JSON document per instance under
// <root>/tmuxlifecycle/instances/<instance_id>.json, installed by
// atomic rename. tmux itself reports only presence (sessions, panes,
// attached counts); the AX states quiescing, stopped, and stale_fenced
// exist only on this side, so status reconciles the live tmux probe
// against this memory (see ClassifyStatus). The store is MUTABLE —
// every mutating operation records its after-state — unlike the
// no-replace idempotency receipts, whose keys can never name a state.
//
// The store is rooted at a DURABLE data root, never the runtime root:
// the runtime root is per-boot temporary, while status after reboot
// must still distinguish a stopped instance from an absent one.
type InstanceStates struct {
	root  string
	hooks *StateHooks
}

// StateHooks arms the crash boundaries of the state install: after the
// state bytes are staged to the temporary file (before the atomic
// rename), and after the rename commits (before the directory sync
// makes the commit durable). A nil hook is a no-op; a hook error aborts
// the call with that error. Hooks is nil in production.
type StateHooks struct {
	AfterStage   func(path string) error
	AfterInstall func(path string) error
}

// stateDocument is the stored shape: exactly schema, schema_version,
// terminal_instance_id, and state.
type stateDocument struct {
	Schema             string `json:"schema"`
	SchemaVersion      string `json:"schema_version"`
	TerminalInstanceID string `json:"terminal_instance_id"`
	State              string `json:"state"`
}

// OpenInstanceStates creates the store, ensuring the root exists. It
// creates no instance documents: those appear with the first Record.
func OpenInstanceStates(root string) (*InstanceStates, error) {
	if root == "" {
		return nil, errors.New("tmux instance state root is empty")
	}
	base := filepath.Join(root, "tmuxlifecycle", "instances")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, fmt.Errorf("create tmux instance state root: %w", err)
	}
	return &InstanceStates{root: root}, nil
}

// WithHooks returns the store with crash hooks armed.
func (store *InstanceStates) WithHooks(hooks *StateHooks) *InstanceStates {
	store.hooks = hooks
	return store
}

func (store *InstanceStates) instancePath(instanceID string) string {
	return filepath.Join(store.root, "tmuxlifecycle", "instances", instanceID+".json")
}

// Record stores the after-state of one mutating operation for one
// instance. The install is atomic (stage, fsync, rename, directory
// fsync): a crash leaves the previous state or the new state, never a
// torn document, and a retry converges by re-recording. Only AX
// lifecycle states record; creating never records (it exists only
// between the idempotency receipt and the first side effect, and the
// receipt already proves that window).
func (store *InstanceStates) Record(instanceID string, state terminalbackend.InstanceState) error {
	if store == nil {
		return errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "instance state identity"}
	}
	if _, err := terminalbackend.ParseInstanceState(string(state)); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "instance state value"}
	}
	if state == terminalbackend.StateCreating {
		return &Error{Code: CodeInvalidArguments, Detail: "instance state value"}
	}
	raw, err := json.Marshal(stateDocument{
		Schema:             "urn:ax:schema:tmux-instance-state",
		SchemaVersion:      "1.0.0",
		TerminalInstanceID: instanceID,
		State:              string(state),
	})
	if err != nil {
		return fmt.Errorf("encode tmux instance state: %w", err)
	}
	return store.install(instanceID, raw)
}

// Lookup replays the recorded state for one instance, or reports
// absence when the instance was never recorded. A read or validation
// failure is an error, never absence: unknown is not absent.
func (store *InstanceStates) Lookup(instanceID string) (terminalbackend.InstanceState, bool, error) {
	if store == nil {
		return "", false, errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return "", false, &Error{Code: CodeInvalidArguments, Detail: "instance state identity"}
	}
	raw, err := os.ReadFile(store.instancePath(instanceID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read tmux instance state: %w", err)
	}
	var document stateDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return "", false, fmt.Errorf("decode tmux instance state: %w", err)
	}
	if document.TerminalInstanceID != instanceID {
		return "", false, &Error{Code: CodeInvalidArguments, Detail: "instance state identity"}
	}
	state, err := terminalbackend.ParseInstanceState(document.State)
	if err != nil {
		return "", false, &Error{Code: CodeInvalidArguments, Detail: "instance state value"}
	}
	return state, true, nil
}

// OperationOutcome is the persisted operation-specific report for one
// idempotency key: the closure timestamps the wire result carries
// outside the mutation receipt (InputClosedAt for quiesce-input,
// BoundaryObservedAt for wait-safe-boundary). The receipt proves the
// effect; this document proves the reported time. First success
// records; every retry replays the recorded values, so the same
// receipt never changes its closure time. Incarnation binds the
// report to the instance incarnation that was current when the
// closure recorded: a create or restore success rotates the
// incarnation, and the barrier scan ignores reports from a
// superseded incarnation, so a new incarnation's input starts open
// by construction instead of inheriting a stale closure.
type OperationOutcome struct {
	Operation          string
	InputClosedAt      string
	BoundaryObservedAt string
	Incarnation        string
}

// outcomeDocument is the stored outcome shape.
type outcomeDocument struct {
	Schema             string `json:"schema"`
	SchemaVersion      string `json:"schema_version"`
	IdempotencyKey     string `json:"idempotency_key"`
	Operation          string `json:"operation"`
	InputClosedAt      string `json:"input_closed_at"`
	BoundaryObservedAt string `json:"boundary_observed_at"`
	Incarnation        string `json:"incarnation"`
}

// RecordOutcome stores the operation-specific report for one
// idempotency key. Keys carry slashes, so the document name is the
// hex SHA-256 of the key; the key itself is echoed inside for the
// lookup to verify. The install is atomic like Record. A concurrent
// duplicate first-completion may record either report; sequential
// retries are stable, which is the tested property.
func (store *InstanceStates) RecordOutcome(key string, outcome OperationOutcome) error {
	if store == nil {
		return errors.New("tmux instance state store is nil")
	}
	if key == "" {
		return &Error{Code: CodeInvalidArguments, Detail: "instance outcome key"}
	}
	raw, err := json.Marshal(outcomeDocument{
		Schema:             "urn:ax:schema:tmux-operation-outcome",
		SchemaVersion:      "1.0.0",
		IdempotencyKey:     key,
		Operation:          outcome.Operation,
		InputClosedAt:      outcome.InputClosedAt,
		BoundaryObservedAt: outcome.BoundaryObservedAt,
		Incarnation:        outcome.Incarnation,
	})
	if err != nil {
		return fmt.Errorf("encode tmux operation outcome: %w", err)
	}
	sum := sha256.Sum256([]byte(key))
	return store.installDocument(filepath.Join("tmuxlifecycle", "outcomes"), hex.EncodeToString(sum[:])+".json", raw)
}

// LookupOutcome replays the recorded report for one idempotency key,
// or reports absence when the key never completed an outcome record.
// A read or validation failure is an error, never absence.
func (store *InstanceStates) LookupOutcome(key string) (OperationOutcome, bool, error) {
	if store == nil {
		return OperationOutcome{}, false, errors.New("tmux instance state store is nil")
	}
	if key == "" {
		return OperationOutcome{}, false, &Error{Code: CodeInvalidArguments, Detail: "instance outcome key"}
	}
	sum := sha256.Sum256([]byte(key))
	raw, err := os.ReadFile(filepath.Join(store.root, "tmuxlifecycle", "outcomes", hex.EncodeToString(sum[:])+".json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return OperationOutcome{}, false, nil
		}
		return OperationOutcome{}, false, fmt.Errorf("read tmux operation outcome: %w", err)
	}
	var document outcomeDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return OperationOutcome{}, false, fmt.Errorf("decode tmux operation outcome: %w", err)
	}
	if document.IdempotencyKey != key {
		return OperationOutcome{}, false, &Error{Code: CodeInvalidArguments, Detail: "instance outcome key"}
	}
	return OperationOutcome{
		Operation:          document.Operation,
		InputClosedAt:      document.InputClosedAt,
		BoundaryObservedAt: document.BoundaryObservedAt,
		Incarnation:        document.Incarnation,
	}, true, nil
}

// incarnationDocument is the stored incarnation shape: exactly
// schema, schema_version, terminal_instance_id, and key.
type incarnationDocument struct {
	Schema             string `json:"schema"`
	SchemaVersion      string `json:"schema_version"`
	TerminalInstanceID string `json:"terminal_instance_id"`
	Key                string `json:"key"`
}

// RecordIncarnation stores the current incarnation key for one
// instance: the idempotency key of the create or restore success
// that opened the incarnation. The install is atomic like Record.
// Only fresh create and restore successes rotate, never replays
// (replaying an old success must not supersede a newer closure)
// and never failures (a failure proves no new incarnation).
func (store *InstanceStates) RecordIncarnation(instanceID, key string) error {
	if store == nil {
		return errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "instance incarnation identity"}
	}
	if key == "" {
		return &Error{Code: CodeInvalidArguments, Detail: "instance incarnation key"}
	}
	raw, err := json.Marshal(incarnationDocument{
		Schema:             "urn:ax:schema:tmux-instance-incarnation",
		SchemaVersion:      "1.0.0",
		TerminalInstanceID: instanceID,
		Key:                key,
	})
	if err != nil {
		return fmt.Errorf("encode tmux instance incarnation: %w", err)
	}
	return store.installDocument(filepath.Join("tmuxlifecycle", "incarnation"), instanceID+".json", raw)
}

// LookupIncarnation replays the current incarnation key for one
// instance, or reports absence when no create or restore success
// ever rotated it. Absence reads as the zero incarnation (""):
// pre-rotation closure reports bind "" and prove the barrier until
// the first rotation supersedes them. A read or validation failure
// is an error, never absence: unknown is not a verdict.
func (store *InstanceStates) LookupIncarnation(instanceID string) (string, bool, error) {
	if store == nil {
		return "", false, errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return "", false, &Error{Code: CodeInvalidArguments, Detail: "instance incarnation identity"}
	}
	raw, err := os.ReadFile(filepath.Join(store.root, "tmuxlifecycle", "incarnation", instanceID+".json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read tmux instance incarnation: %w", err)
	}
	var document incarnationDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return "", false, fmt.Errorf("decode tmux instance incarnation: %w", err)
	}
	if document.TerminalInstanceID != instanceID {
		return "", false, &Error{Code: CodeInvalidArguments, Detail: "instance incarnation identity"}
	}
	if document.Key == "" {
		return "", false, &Error{Code: CodeInvalidArguments, Detail: "instance incarnation key"}
	}
	return document.Key, true, nil
}

// QuiesceBarrierProven reports whether a completed input-closure
// outcome exists for one instance in its current incarnation: a
// recorded quiesce-input or wait-safe-boundary report whose
// idempotency key names this instance and whose incarnation equals
// the current one. The attach entry consults it as the sole
// admission verdict: advisory memory, caller-carried source state,
// and failure reporting are never consulted, so none of them can
// reopen a closed barrier. A report from a superseded incarnation
// is skipped, never an error: the rotation that superseded it is
// itself the authoritative reopen proof. A missing outcomes
// directory reports no barrier; any read or decode failure is an
// error, never absence — unknown is not a verdict. A matching
// current-incarnation report with an empty closure time is corrupt
// (the exact read refuses it the same way) and errors instead of
// proving the barrier.
func (store *InstanceStates) QuiesceBarrierProven(instanceID string) (bool, error) {
	if store == nil {
		return false, errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return false, &Error{Code: CodeInvalidArguments, Detail: "instance state identity"}
	}
	current, _, err := store.LookupIncarnation(instanceID)
	if err != nil {
		return false, err
	}
	dir := filepath.Join(store.root, "tmuxlifecycle", "outcomes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read tmux operation outcomes: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return false, fmt.Errorf("read tmux operation outcome: %w", err)
		}
		var document outcomeDocument
		if err := json.Unmarshal(raw, &document); err != nil {
			return false, fmt.Errorf("decode tmux barrier outcome: %w", err)
		}
		if document.IdempotencyKey == "" {
			return false, &Error{Code: CodeInvalidArguments, Detail: "instance outcome key"}
		}
		// The key is authoritative for the family, exactly as on
		// the exact read (which never consults the echoed
		// operation): a closure key for this instance proves the
		// barrier when its time is present and its incarnation is
		// current. A superseded incarnation is skipped before the
		// time arms: the rotation already reopened the input, so
		// the stale report proves nothing either way.
		quiesce := strings.HasPrefix(document.IdempotencyKey, instanceID+"/quiesce/")
		boundary := !quiesce && strings.HasPrefix(document.IdempotencyKey, instanceID+"/boundary/")
		if !quiesce && !boundary {
			continue
		}
		if document.Incarnation != current {
			continue
		}
		if quiesce && document.InputClosedAt == "" {
			return false, &Error{Code: CodeInvalidArguments, Detail: "instance outcome key"}
		}
		if boundary && document.BoundaryObservedAt == "" {
			return false, &Error{Code: CodeInvalidArguments, Detail: "instance outcome key"}
		}
		return true, nil
	}
	return false, nil
}

// RecordBindingDoc stores the create-time Terminal Instance Binding
// document for one (session, bootstrap) pair, so restore can return
// the required binding without minting (the JCS identity rule stays
// termbind-private: this store never computes digests, it only keeps
// the bytes create carried). The document must be a JSON object; its
// binding identity is re-admitted by termbind on the restore read,
// never here.
func (store *InstanceStates) RecordBindingDoc(sessionID, bootstrapID string, doc []byte) error {
	if store == nil {
		return errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "instance binding identity"}
	}
	if _, err := scalar.ParseUUIDv7(bootstrapID); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "instance binding identity"}
	}
	var probe map[string]any
	if err := json.Unmarshal(doc, &probe); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "instance binding document"}
	}
	return store.installDocument(filepath.Join("tmuxlifecycle", "bindings", sessionID), bootstrapID+".json", doc)
}

// LookupBindingDoc replays the stored binding document for one
// (session, bootstrap) pair, or reports absence when the pair never
// recorded one. A read failure is an error, never absence.
func (store *InstanceStates) LookupBindingDoc(sessionID, bootstrapID string) ([]byte, bool, error) {
	if store == nil {
		return nil, false, errors.New("tmux instance state store is nil")
	}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return nil, false, &Error{Code: CodeInvalidArguments, Detail: "instance binding identity"}
	}
	if _, err := scalar.ParseUUIDv7(bootstrapID); err != nil {
		return nil, false, &Error{Code: CodeInvalidArguments, Detail: "instance binding identity"}
	}
	raw, err := os.ReadFile(filepath.Join(store.root, "tmuxlifecycle", "bindings", sessionID, bootstrapID+".json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read tmux binding document: %w", err)
	}
	return raw, true, nil
}

// install stages the state bytes to a same-directory temporary file,
// fsyncs it, atomically renames it over the instance document (rename
// is the mutable commit: the new state supersedes), removes no prior
// name, and fsyncs the directory. Stale staging files from a
// pre-commit crash are swept before the install.
func (store *InstanceStates) install(instanceID string, raw []byte) error {
	return store.installDocument(filepath.Join("tmuxlifecycle", "instances"), instanceID+".json", raw)
}

// installDocument stages raw bytes to a same-directory temporary file
// and atomically renames them over the named document under the
// store root, sweeping stale staging files first and fsyncing the
// directory after. Crash hooks fire at the same two boundaries as the
// state install. rel is relative to the store root.
func (store *InstanceStates) installDocument(rel, name string, raw []byte) error {
	dir := filepath.Join(store.root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create tmux instance state directory: %w", err)
	}
	sweepStateStaging(dir)
	target := filepath.Join(dir, name)
	staged, err := os.CreateTemp(dir, "state-*.tmp")
	if err != nil {
		return fmt.Errorf("stage tmux instance state: %w", err)
	}
	stagedPath := staged.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(stagedPath)
		}
	}()
	if err := staged.Chmod(0o600); err != nil {
		_ = staged.Close()
		return fmt.Errorf("protect tmux instance state: %w", err)
	}
	if _, err := staged.Write(raw); err != nil {
		_ = staged.Close()
		return fmt.Errorf("write tmux instance state: %w", err)
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return fmt.Errorf("sync tmux instance state: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close tmux instance state: %w", err)
	}
	if store.hooks != nil && store.hooks.AfterStage != nil {
		if err := store.hooks.AfterStage(stagedPath); err != nil {
			return err
		}
	}
	if err := os.Rename(stagedPath, target); err != nil {
		return fmt.Errorf("install tmux instance state: %w", err)
	}
	committed = true
	if store.hooks != nil && store.hooks.AfterInstall != nil {
		if err := store.hooks.AfterInstall(target); err != nil {
			return err
		}
	}
	if err := syncStateDirectory(dir); err != nil {
		return err
	}
	return nil
}

// sweepStateStaging removes stale staging files from pre-commit
// crashes. Only the state-*.tmp pattern is swept; instance documents
// are never touched.
func sweepStateStaging(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 10 && name[:6] == "state-" && filepath.Ext(name) == ".tmp" {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

// syncStateDirectory fsyncs the directory so the rename commit is
// durable before Record returns.
func syncStateDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return dir.Sync()
}
