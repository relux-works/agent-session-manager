package termbind

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Attach receipt schema identity. This is an AX-internal host-local durable
// record (the Section 4.E host-local durable metadata class: the durable
// client receipt the Section 4.C attach row requires), not a replicated
// contract: it never leaves the host and never touches the session chain,
// leases, or fencing state.
const (
	attachSchema        = "urn:ax:schema:termbind-attach-receipt"
	attachSchemaVersion = "1.0.0"
)

// Attach transports from Section 4.E. The enum gate admits all three; the
// landed attach policy refuses the relay transport with unauthorized,
// since current AX admits only the first two.
var attachTransports = []string{"local_only", "trusted_private_mesh", "third_party_relay"}

// AttachReceipt is the one recorded presentation client for a
// (terminal_instance_id, client_id) pair: the Section 4.C attach
// idempotency key material. Identity is AX-allocated UUIDv7 throughout;
// the receipt carries no chain, lease, or fencing member by construction.
type AttachReceipt struct {
	SessionID          string `json:"session_id"`
	TerminalInstanceID string `json:"terminal_instance_id"`
	ClientID           string `json:"client_id"`
	Transport          string `json:"transport"`
	InputAuthorized    bool   `json:"input_authorized"`
	CreatedAt          string `json:"created_at"`
}

// attachMembers is the exact closed member set of the stored document.
var attachMembers = []string{
	"schema",
	"schema_version",
	"session_id",
	"terminal_instance_id",
	"client_id",
	"transport",
	"input_authorized",
	"created_at",
}

// AttachHooks arms the crash boundaries of the receipt install: after the
// receipt bytes are staged to the temporary file (before the atomic
// install), and after the install commits (before the directory sync makes
// the commit durable). A nil hook is a no-op; a hook error aborts the call
// with that error.
type AttachHooks struct {
	AfterStage   func(path string) error
	AfterInstall func(path string) error
}

// AttachStore is the durable attach-client receipt store rooted at one
// data directory. Each (terminal_instance_id, client_id) pair keeps its
// own receipt at
// <root>/termbind/attach/<instance_id>/<client_id>.json, installed
// no-replace: an identical retry replays, a conflicting retry refuses
// idempotency_mismatch, and a second client records alongside, never over.
type AttachStore struct {
	root  string
	hooks *AttachHooks
	now   func() time.Time
}

// OpenAttachStore creates the store, ensuring the root exists. It creates
// no instance directories: those appear with the first Attach.
func OpenAttachStore(root string) (*AttachStore, error) {
	if root == "" {
		return nil, errors.New("termbind attach store root is empty")
	}
	base := filepath.Join(root, "termbind", "attach")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, fmt.Errorf("create termbind attach root: %w", err)
	}
	return &AttachStore{root: root, now: time.Now}, nil
}

// WithHooks returns the store with crash hooks armed. It mutates the
// receiver and returns it for call chaining.
func (store *AttachStore) WithHooks(hooks *AttachHooks) *AttachStore {
	store.hooks = hooks
	return store
}

func (store *AttachStore) instanceDir(instanceID string) string {
	return filepath.Join(store.root, "termbind", "attach", instanceID)
}

func (store *AttachStore) receiptPath(instanceID, clientID string) string {
	return filepath.Join(store.instanceDir(instanceID), clientID+".json")
}

// Attach records one presentation client under its Section 4.C idempotency
// key (terminal_instance_id + "/" + client_id) and returns the receipt. An
// identical retry replays the recorded receipt with replayed true; a retry
// that changes transport or input authorization refuses
// idempotency_mismatch; a second client records its own receipt. Every
// call, including a replay, requires a valid unexpired AttachAuthorization
// equal to the request transport and input: timeout creates no authorized
// input unless the durable receipt says so, and an expired policy never
// replays.
//
// Attach takes no repository, lease, or fencing input and emits no event:
// presentation-client activity cannot change Owner/Replica, lease, or
// fencing state by construction. The zero-events property is proven by
// snapshotting a real chain and lease store around the call.
func (store *AttachStore) Attach(sessionID, instanceID, clientID, transport string, inputAuthorized bool, authRaw []byte, now time.Time) (AttachReceipt, bool, error) {
	empty := AttachReceipt{}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return empty, false, &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: "attach session"}
	}
	if err := CheckInstanceIdentity(instanceID); err != nil {
		return empty, false, err
	}
	if err := CheckInstanceIdentity(clientID); err != nil {
		return empty, false, err
	}
	if !isAttachTransport(transport) {
		return empty, false, protocolRefusal("attach transport")
	}
	auth, err := terminalbackend.ParseAttachAuthorization(authRaw)
	if err != nil {
		return empty, false, err
	}
	if err := terminalbackend.CheckAttachRequest(auth, transport, inputAuthorized, now); err != nil {
		return empty, false, err
	}
	stored, found, err := store.Lookup(sessionID, instanceID, clientID)
	if err != nil {
		return empty, false, err
	}
	if found {
		if stored.Transport != transport || stored.InputAuthorized != inputAuthorized {
			return empty, false, &terminalbackend.Error{Code: terminalbackend.CodeIdempotencyMismatch, Detail: "attach client"}
		}
		return stored, true, nil
	}
	receipt, err := encodeAttachReceipt(AttachReceipt{
		SessionID:          sessionID,
		TerminalInstanceID: instanceID,
		ClientID:           clientID,
		Transport:          transport,
		InputAuthorized:    inputAuthorized,
	}, store.timestamp())
	if err != nil {
		return empty, false, err
	}
	return store.install(sessionID, instanceID, clientID, transport, inputAuthorized, receipt)
}

// Lookup replays the recorded receipt for a (terminal_instance_id,
// client_id) pair, or reports absence when the pair is unrecorded. A read
// or validation failure is an error, never absence: unknown is not absent.
func (store *AttachStore) Lookup(sessionID, instanceID, clientID string) (AttachReceipt, bool, error) {
	empty := AttachReceipt{}
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return empty, false, fmt.Errorf("attach lookup session %q is not a UUIDv7: %w", sessionID, err)
	}
	if err := CheckInstanceIdentity(instanceID); err != nil {
		return empty, false, err
	}
	if err := CheckInstanceIdentity(clientID); err != nil {
		return empty, false, err
	}
	raw, err := os.ReadFile(store.receiptPath(instanceID, clientID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return empty, false, nil
		}
		return empty, false, fmt.Errorf("read attach receipt: %w", err)
	}
	receipt, err := decodeAttachReceipt(raw)
	if err != nil {
		return empty, false, err
	}
	if receipt.SessionID != sessionID {
		return empty, false, errors.New("attach receipt names another session")
	}
	if receipt.TerminalInstanceID != instanceID || receipt.ClientID != clientID {
		return empty, false, errors.New("attach receipt names another client pair")
	}
	return receipt, true, nil
}

// install stages the receipt to a same-directory temporary file, fsyncs
// it, atomically links it to the receipt path (which must not exist: the
// link is the no-replace commit), removes the staging name, and fsyncs the
// directory. A concurrent same-pair commit wins and the loser converges to
// the winner: an identical retry replays it, a conflicting retry refuses
// idempotency_mismatch. Stale staging files from a pre-commit crash are
// swept before the install.
func (store *AttachStore) install(sessionID, instanceID, clientID, transport string, inputAuthorized bool, receipt []byte) (AttachReceipt, bool, error) {
	empty := AttachReceipt{}
	dir := store.instanceDir(instanceID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return empty, false, fmt.Errorf("create attach instance directory: %w", err)
	}
	sweepAttachStaging(dir)
	target := store.receiptPath(instanceID, clientID)
	staged, err := os.CreateTemp(dir, "attach-*.tmp")
	if err != nil {
		return empty, false, fmt.Errorf("stage attach receipt: %w", err)
	}
	stagedPath := staged.Name()
	defer func() { _ = os.Remove(stagedPath) }()
	if err := staged.Chmod(0o600); err != nil {
		_ = staged.Close()
		return empty, false, fmt.Errorf("protect attach receipt: %w", err)
	}
	if _, err := staged.Write(receipt); err != nil {
		_ = staged.Close()
		return empty, false, fmt.Errorf("write attach receipt: %w", err)
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return empty, false, fmt.Errorf("sync attach receipt: %w", err)
	}
	if err := staged.Close(); err != nil {
		return empty, false, fmt.Errorf("close attach receipt: %w", err)
	}
	if store.hooks != nil && store.hooks.AfterStage != nil {
		if err := store.hooks.AfterStage(stagedPath); err != nil {
			return empty, false, err
		}
	}
	if err := os.Link(stagedPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			winner, found, readErr := store.Lookup(sessionID, instanceID, clientID)
			if readErr != nil {
				return empty, false, readErr
			}
			if !found {
				return empty, false, errors.New("attach receipt vanished after concurrent commit")
			}
			if winner.Transport != transport || winner.InputAuthorized != inputAuthorized {
				return empty, false, &terminalbackend.Error{Code: terminalbackend.CodeIdempotencyMismatch, Detail: "attach client"}
			}
			return winner, true, nil
		}
		return empty, false, fmt.Errorf("install attach receipt: %w", err)
	}
	_ = os.Remove(stagedPath)
	if store.hooks != nil && store.hooks.AfterInstall != nil {
		if err := store.hooks.AfterInstall(target); err != nil {
			return empty, false, err
		}
	}
	if err := syncAttachDirectory(dir); err != nil {
		return empty, false, err
	}
	stored, found, err := store.Lookup(sessionID, instanceID, clientID)
	if err != nil {
		return empty, false, err
	}
	if !found {
		return empty, false, errors.New("attach receipt vanished after install")
	}
	return stored, false, nil
}

func (store *AttachStore) timestamp() string {
	now := time.Now().UTC()
	if store.now != nil {
		now = store.now().UTC()
	}
	return now.Format("2006-01-02T15:04:05.000Z07:00")
}

// isAttachTransport reports whether transport is one of the three Section
// 4.E presentation transports. The literals come from the specification
// table, never from the implementation.
func isAttachTransport(transport string) bool {
	for _, member := range attachTransports {
		if transport == member {
			return true
		}
	}
	return false
}

// sweepAttachStaging removes stale attach-*.tmp files left by a pre-commit
// crash. Only files older than a minute are removed: a concurrent live
// attach stages under the same pattern, and an age gate keeps the sweep
// from deleting its in-flight receipt. It is best-effort and removes only
// the staging pattern.
func sweepAttachStaging(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, "attach-*.tmp"))
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

// syncAttachDirectory fsyncs a directory so an install inside it survives
// a crash. Opening a directory fails on Windows, where the install itself
// is the durability boundary; there the sync is skipped, never faked. This
// mirrors the landed matjournal discipline the axpane owner follows.
func syncAttachDirectory(path string) error {
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

func encodeAttachReceipt(receipt AttachReceipt, createdAt string) ([]byte, error) {
	document := map[string]any{
		"schema":               attachSchema,
		"schema_version":       attachSchemaVersion,
		"session_id":           receipt.SessionID,
		"terminal_instance_id": receipt.TerminalInstanceID,
		"client_id":            receipt.ClientID,
		"transport":            receipt.Transport,
		"input_authorized":     receipt.InputAuthorized,
		"created_at":           createdAt,
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode attach receipt: %w", err)
	}
	return raw, nil
}

func decodeAttachReceipt(raw []byte) (AttachReceipt, error) {
	empty := AttachReceipt{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document struct {
		Schema             string `json:"schema"`
		SchemaVersion      string `json:"schema_version"`
		SessionID          string `json:"session_id"`
		TerminalInstanceID string `json:"terminal_instance_id"`
		ClientID           string `json:"client_id"`
		Transport          string `json:"transport"`
		InputAuthorized    bool   `json:"input_authorized"`
		CreatedAt          string `json:"created_at"`
	}
	if err := decoder.Decode(&document); err != nil {
		return empty, fmt.Errorf("attach receipt is not the closed shape: %w", err)
	}
	if decoder.More() {
		return empty, fmt.Errorf("attach receipt carries trailing data")
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return empty, fmt.Errorf("attach receipt is not an object: %w", err)
	}
	for _, name := range attachMembers {
		if _, ok := members[name]; !ok {
			return empty, fmt.Errorf("attach receipt misses required member %q", name)
		}
	}
	if document.Schema != attachSchema || document.SchemaVersion != attachSchemaVersion {
		return empty, fmt.Errorf("attach receipt schema is not %s %s", attachSchema, attachSchemaVersion)
	}
	receipt := AttachReceipt{
		SessionID:          document.SessionID,
		TerminalInstanceID: document.TerminalInstanceID,
		ClientID:           document.ClientID,
		Transport:          document.Transport,
		InputAuthorized:    document.InputAuthorized,
		CreatedAt:          document.CreatedAt,
	}
	if _, err := scalar.ParseUUIDv7(receipt.SessionID); err != nil {
		return empty, fmt.Errorf("attach receipt session is not a UUIDv7: %w", err)
	}
	if _, err := scalar.ParseUUIDv7(receipt.TerminalInstanceID); err != nil {
		return empty, fmt.Errorf("attach receipt instance is not a UUIDv7: %w", err)
	}
	if _, err := scalar.ParseUUIDv7(receipt.ClientID); err != nil {
		return empty, fmt.Errorf("attach receipt client is not a UUIDv7: %w", err)
	}
	if _, err := scalar.ParseTimestamp(receipt.CreatedAt); err != nil {
		return empty, fmt.Errorf("attach receipt timestamp is not a timestamp: %w", err)
	}
	return receipt, nil
}
