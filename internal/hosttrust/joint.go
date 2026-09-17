package hosttrust

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/relux-works/agent-session-manager/internal/localstore"
)

// pendingCommitFileName is the durable intent marker for one joint
// config-plus-trust commit. It lives beside trust.json under the same
// owner-only custody and crash-durable write discipline.
const pendingCommitFileName = "pending-commit.json"

// Joint operations coupled to a trust generation bump.
const (
	JointApplyV4    = "apply-v4"
	JointRollbackV4 = "rollback-v4"
)

// PendingCommit is the durable intent record for one joint commit. The
// hashes pin the exact configuration bytes the operator confirmed; the
// source generation pins the trust state they were rendered against. Only
// the confirmed replacement may complete forward; anything else aborts or
// refuses on restart.
type PendingCommit struct {
	Schema            string `json:"schema"`
	SchemaVersion     string `json:"schema_version"`
	Operation         string `json:"operation"`
	ConfigPath        string `json:"config_path"`
	SourceSHA256      string `json:"source_sha256"`
	ReplacementSHA256 string `json:"replacement_sha256"`
	SourceGeneration  uint64 `json:"source_generation"`
	BackupPath        string `json:"backup_path"`
}

// SHA256HexOf returns the lowercase hex SHA-256 of document bytes for
// marker pins. It exposes no key material: markers pin configuration bytes
// only, never credentials.
func SHA256HexOf(document []byte) string {
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:])
}

func encodePendingCommit(marker PendingCommit) ([]byte, error) {
	if err := checkPendingCommit(marker); err != nil {
		return nil, err
	}
	document, err := json.Marshal(marker)
	if err != nil {
		return nil, trustError(TrustError{Operation: "encode joint commit", Err: errors.Join(ErrTrustDurability, err)})
	}
	return document, nil
}

func checkPendingCommit(marker PendingCommit) error {
	const operation = "validate joint commit"
	if marker.Schema != "urn:ax:schema:host-pending-commit" || marker.SchemaVersion != "1.0.0" {
		return trustError(TrustError{Operation: operation, Err: ErrTrustValidation})
	}
	if marker.Operation != JointApplyV4 && marker.Operation != JointRollbackV4 {
		return trustError(TrustError{Operation: operation, Err: ErrTrustValidation})
	}
	if !filepath.IsAbs(marker.ConfigPath) || marker.BackupPath == "" {
		return trustError(TrustError{Operation: operation, Err: ErrTrustValidation})
	}
	for _, digest := range []string{marker.SourceSHA256, marker.ReplacementSHA256} {
		raw, err := hex.DecodeString(digest)
		if err != nil || len(raw) != sha256.Size {
			return trustError(TrustError{Operation: operation, Err: ErrTrustValidation})
		}
	}
	if marker.SourceSHA256 == marker.ReplacementSHA256 {
		return trustError(TrustError{Operation: operation, Err: ErrTrustValidation})
	}
	if marker.SourceGeneration == 0 || marker.SourceGeneration > MaxGeneration {
		return trustError(TrustError{Operation: operation, Err: ErrTrustValidation})
	}
	return nil
}

func decodePendingCommit(document []byte) (PendingCommit, error) {
	var marker PendingCommit
	if err := rejectDuplicateKeys(document); err != nil {
		return PendingCommit{}, err
	}
	// The generation must be a bare JSON number, never a quoted string:
	// validate it through the same uint53 discipline as the trust document
	// before the struct decode, so a quoted number refuses as invalid
	// state rather than as a syntax error.
	raw := map[string]json.RawMessage{}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return PendingCommit{}, trustError(TrustError{Operation: "decode joint commit", Err: errors.Join(ErrTrustDecode, err)})
	}
	generation, err := decodeMarkerGeneration(raw)
	if err != nil {
		return PendingCommit{}, err
	}
	decoder = json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&marker); err != nil {
		return PendingCommit{}, trustError(TrustError{Operation: "decode joint commit", Err: errors.Join(ErrTrustDecode, err)})
	}
	if decoder.More() {
		return PendingCommit{}, trustError(TrustError{Operation: "decode joint commit", Err: ErrTrustDecode})
	}
	marker.SourceGeneration = generation
	if err := checkPendingCommit(marker); err != nil {
		return PendingCommit{}, err
	}
	return marker, nil
}

func parseUint53(text string) (uint64, error) {
	if text == "" {
		return 0, errors.New("empty generation number")
	}
	for _, digit := range text {
		if digit < '0' || digit > '9' {
			return 0, errors.New("non-digit generation number")
		}
	}
	parsed, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed == 0 || parsed > MaxGeneration {
		return 0, errors.New("generation outside uint53")
	}
	return parsed, nil
}

func decodeMarkerGeneration(shape map[string]json.RawMessage) (uint64, error) {
	raw, ok := shape["source_generation"]
	if !ok {
		return 0, trustError(TrustError{Operation: "decode joint generation", Err: ErrTrustValidation})
	}
	var value any
	if err := strictJSONDecoder(raw).Decode(&value); err != nil {
		return 0, trustError(TrustError{Operation: "decode joint generation", Err: errors.Join(ErrTrustDecode, err)})
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, trustError(TrustError{Operation: "decode joint generation", Err: ErrTrustValidation})
	}
	parsed, err := parseUint53(number.String())
	if err != nil {
		return 0, trustError(TrustError{Operation: "decode joint generation", Err: ErrTrustValidation})
	}
	return parsed, nil
}

// JointCommit runs one crash-atomic authorization-generation commit coupled
// to an external durable replacement (Configuration 4.0.0 apply or
// rollback). Under a single exclusive hold it converges any leftover
// intent, revalidates the confirmed pins against committed trust, stages
// the durable intent marker, runs replace (the configuration file
// replacement), bumps the generation, and removes the marker.
//
//   - Leftover intent from a crashed commit converges first: a marker the
//     replacement committed completes forward (moving the generation, so
//     stale markers still refuse below); an unreplaced marker aborts. A
//     refused leftover fails the commit with the marker kept.
//   - revalidate runs inside the lock: the preview, source and generation
//     checks it repeats there close the concurrent-change window between
//     the unlocked fast-fail checks and the commit.
//   - A replace failure resolves against the confirmed pins before the
//     marker moves: the configuration still equal to the source hash
//     aborts with no generation change and the original error, while a
//     replacement that landed despite the failure (or state that can no
//     longer be proven) runs the same compensation protocol as a failed
//     bump, restoring the source or keeping the marker for operator
//     resolution. An error alone never proves the configuration is
//     untouched.
//   - A trust-commit failure after replace compensates under the same
//     hold: the marker and the configuration are revalidated against the
//     confirmed pins, restore reinstalls the source bytes, and the marker
//     aborts with no generation change. Compensation never releases the
//     hold, so no surviving reader, writer or held boundary can converge
//     the replacement between the failure and the restore. A compensation
//     that cannot complete keeps the marker for operator resolution and
//     joins ErrJointCompensationFailed. The committed trust itself is never
//     re-read for the decision: the fault that broke the bump (for example
//     failed custody) must not break the abort, and an apparently durable
//     bump is still unproven exactly because its commit reported failure.
//   - Marker-removal failure after a durable bump still returns success:
//     the joint state is already forward-complete, and the next converged
//     read or open removes the marker.
//
// Only process crashes are covered by the restart protocol below; power
// loss additionally depends on the disk honoring fsync, which no test here
// can establish. Crash tests terminate a real child process at the
// replacement seam and re-read through surviving and reopened stores.
//
// The replace and restore closures receive the live hold to pass to
// token-gated pair writers; only closures invoked by this commit ever see
// a genuine hold.
func (store *Store) JointCommit(paths localstore.ResolvedPaths, marker PendingCommit, revalidate func(current TrustStore) error, replace func(HeldExclusive) error, restore func(HeldExclusive) error) (uint64, error) {
	const operation = "joint commit"
	if revalidate == nil || replace == nil || restore == nil {
		return 0, trustError(TrustError{Operation: operation, Err: ErrTrustDurability})
	}
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return 0, err
	}
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		return 0, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustDurability, err)})
	}
	defer unlock()
	if err := store.convergeLocked(hold); err != nil {
		return 0, err
	}
	committed, err := loadCommitted(store.fs, store.root)
	if err != nil {
		return 0, err
	}
	if err := revalidate(committed); err != nil {
		return 0, err
	}
	if marker.SourceGeneration != committed.Generation {
		return 0, trustError(TrustError{Operation: operation, Err: ErrStaleGeneration})
	}
	if err := checkPendingCommit(marker); err != nil {
		return 0, err
	}
	// The marker can never mint its own target capability. The store's
	// persistent binding is the authority; the marker path must repeat that
	// binding (after canonicalization), and an explicitly established binding
	// is required before a joint operation may stage its marker.
	canonicalConfigPath, err := canonicalResolvedConfigPath(paths)
	markerConfigPath, markerPathErr := canonicalExistingPath(marker.ConfigPath)
	if err != nil || markerPathErr != nil || markerConfigPath != canonicalConfigPath {
		return 0, trustError(TrustError{Operation: operation, Err: ErrExclusiveHoldResourceMismatch})
	}
	// Persist the canonical target identity even when the caller supplied a
	// symlink-preserving alias. Recovery and later readers compare the same
	// durable identity, while the localstore pair still controls which target
	// may be selected by this store.
	marker.ConfigPath = canonicalConfigPath
	configHold, err := store.boundConfigHoldLocked(hold, paths)
	if err != nil {
		return 0, err
	}
	if _, err := store.fs.Lstat(pendingPath(store.root)); err == nil {
		return 0, trustError(TrustError{Operation: operation, Err: ErrJointIntervened})
	} else if !os.IsNotExist(err) {
		return 0, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		return 0, err
	}
	if err := commitDocument(hold, store.fs, store.root, pendingPath(store.root), encoded); err != nil {
		return 0, err
	}
	if err := replace(configHold); err != nil {
		return 0, resolveFailedReplace(hold, configHold, store.fs, store.root, paths, marker, err, restore)
	}
	var generation uint64
	if err := transactLocked(hold, store.fs, store.root, func(trust *TrustStore) error {
		generation = trust.Generation + 1
		return nil
	}); err != nil {
		return 0, compensateLocked(hold, configHold, store.fs, store.root, paths, marker, err, restore)
	}
	// The replacement and the bump are both durable: converge the marker
	// removal on next open when this cleanup itself is interrupted.
	_ = removeMarkerLocked(hold, store.fs, store.root)
	return generation, nil
}

// resolveFailedReplace resolves a joint commit whose replace closure
// reported failure. A replace error does not prove the configuration is
// unchanged: the replacement rename may have landed with a later sync or
// restoration step failing. Only configuration still equal to the source
// pin aborts with the marker removed and the original error; anything
// else (durable replacement, diverged bytes, unreadable state) runs the
// common compensation protocol, which restores the source when it can be
// proven and otherwise keeps the marker so readers refuse or converge
// instead of admitting a changed configuration at the old generation.
// The caller must hold the exclusive lock: no writer can move the
// configuration between this resolution and the compensation it runs.
func resolveFailedReplace(hold, configHold HeldExclusive, filesystem FileSystem, root string, paths localstore.ResolvedPaths, intent PendingCommit, replaceErr error, restore func(HeldExclusive) error) error {
	if err := requireHoldForRoot(hold, root); err != nil {
		return err
	}
	configPath, err := pairedIntentTarget(paths, intent)
	if err != nil {
		return err
	}
	configuration, err := filesystem.ReadFile(configPath)
	if err == nil && SHA256HexOf(configuration) == intent.SourceSHA256 {
		_ = removeMarkerLocked(hold, filesystem, root)
		return replaceErr
	}
	return compensateLocked(hold, configHold, filesystem, root, paths, intent, replaceErr, restore)
}

// compensateLocked converges a joint commit whose replacement is durable
// but whose generation bump failed with commitErr. The caller must hold
// the exclusive authorization lock across the whole compensation: the
// marker and the configuration are revalidated against the confirmed
// intent, restore reinstalls the source bytes, and the marker aborts with
// no generation change. Because the hold never lapses, no surviving
// reader, writer or held boundary can converge the replacement between
// the failure and the restore; the only way out is the clean abort or a
// refused compensation with the marker kept.
//
//   - The marker is re-read and compared pin-for-pin with the staged
//     intent: no in-protocol path changes it under the hold, so a
//     mismatch, an undecodable marker or an unreadable marker path is an
//     intervention that refuses with the marker kept. A vanished marker
//     is the operator/sabotage remainder: the in-memory intent still pins
//     the decision, and the marker is re-staged when the configuration
//     matches neither pin so restart recovery resolves the pair instead
//     of admitting it silently.
//   - Configuration equal to the replacement hash runs restore, whose
//     result is verified against the source hash before the marker
//     aborts: a restore that reports success without reinstalling the
//     source refuses as an intervention. Configuration already equal to
//     the source hash aborts without running the restore.
//   - A restore failure, an unreadable configuration, or configuration
//     matching neither pin keeps the marker for operator resolution and
//     joins ErrJointCompensationFailed. The configuration then stays at
//     the replacement, which restart recovery completes forward once the
//     underlying fault heals.
func compensateLocked(hold, configHold HeldExclusive, filesystem FileSystem, root string, paths localstore.ResolvedPaths, intent PendingCommit, commitErr error, restore func(HeldExclusive) error) error {
	const operation = "compensate joint commit"
	if err := requireHoldForRoot(hold, root); err != nil {
		return err
	}
	configPath, err := pairedIntentTarget(paths, intent)
	if err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrExclusiveHoldResourceMismatch, err)})
	}
	failed := func(cause error) error {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrJointReplacementDurable, ErrJointCompensationFailed, commitErr, cause)})
	}
	reread, present, err := readCompensationMarker(filesystem, root)
	if err != nil {
		return failed(err)
	}
	if present && reread != intent {
		return failed(trustError(TrustError{Operation: "validate joint marker", Err: ErrJointIntervened}))
	}
	configuration, err := filesystem.ReadFile(configPath)
	if err != nil {
		return failed(trustError(TrustError{Operation: "read joint configuration", Err: errors.Join(ErrTrustStoreUnreadable, err)}))
	}
	switch SHA256HexOf(configuration) {
	case intent.ReplacementSHA256:
		if err := restore(configHold); err != nil {
			return failed(err)
		}
		restored, err := filesystem.ReadFile(configPath)
		if err != nil {
			return failed(trustError(TrustError{Operation: "read joint configuration", Err: errors.Join(ErrTrustStoreUnreadable, err)}))
		}
		if SHA256HexOf(restored) != intent.SourceSHA256 {
			return failed(trustError(TrustError{Operation: "validate joint restore", Err: ErrJointIntervened}))
		}
	case intent.SourceSHA256:
		// The replacement never landed: abort without running the
		// restore, which would only rewrite identical bytes.
	default:
		if !present {
			// Re-stage the intent so restart recovery resolves the
			// diverged pair instead of admitting it silently.
			if encoded, err := encodePendingCommit(intent); err == nil {
				_ = commitDocument(hold, filesystem, root, pendingPath(root), encoded)
			}
		}
		return failed(trustError(TrustError{Operation: "validate joint configuration", Err: ErrJointIntervened}))
	}
	if present {
		if err := removeMarkerLocked(hold, filesystem, root); err != nil {
			return failed(err)
		}
	}
	return trustError(TrustError{Operation: "commit joint generation", Err: errors.Join(ErrJointReplacementDurable, commitErr)})
}

// readCompensationMarker re-reads the outstanding intent marker for
// in-hold compensation. Absence is reported, never assumed from a failed
// inspection; an existing marker is custody-verified and decoded exactly
// like recovery verifies it. The present flag is meaningful only when err
// is nil.
func readCompensationMarker(filesystem FileSystem, root string) (marker PendingCommit, present bool, err error) {
	const operation = "read joint marker"
	path := pendingPath(root)
	info, statErr := filesystem.Lstat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return PendingCommit{}, false, nil
		}
		return PendingCommit{}, false, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, statErr)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return PendingCommit{}, false, err
	}
	document, err := filesystem.ReadFile(path)
	if err != nil {
		return PendingCommit{}, false, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	marker, err = decodePendingCommit(document)
	if err != nil {
		return PendingCommit{}, false, err
	}
	return marker, true, nil
}

// RecoverResult reports what one recovery pass found.
type RecoverResult struct {
	// Found reports a marker was present at all.
	Found bool
	// Completed reports the replacement had committed and the generation
	// converged forward to source plus one (or already had).
	Completed bool
	// Aborted reports the replacement had not committed: the marker was
	// removed with no generation change.
	Aborted bool
}

// Recover converges one interrupted joint commit found on disk. It runs
// automatically on Open, and is safe to call directly. Every other read,
// admission and transaction entry point converges through the same logic
// before exposing a snapshot (see convergeLocked), so surviving processes
// converge exactly like reopened ones. In-hold error compensation (see
// compensateLocked) aborts without a separate recovery pass; Recover
// converges only markers a crash left behind:
//
//   - configuration equal to the replacement hash completes forward: a
//     trust at the source generation bumps to source plus one; a trust
//     already there only removes the marker.
//   - configuration equal to the source hash aborts: the marker is removed
//     with no generation change.
//   - anything else (operator edit mid-flight, unreadable state) refuses
//     and keeps the marker for operator resolution.
//
// A refused recovery fails the caller, so no reader can silently admit new
// configuration with an old generation.
func (store *Store) Recover() (RecoverResult, error) {
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		return RecoverResult{}, trustError(TrustError{Operation: "recover joint commit", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	defer unlock()
	return recoverLocked(hold, store.fs, store.root, store.paths)
}

func recoverLocked(hold HeldExclusive, filesystem FileSystem, root string, paths localstore.ResolvedPaths) (RecoverResult, error) {
	const operation = "recover joint commit"
	if err := requireHoldForRoot(hold, root); err != nil {
		return RecoverResult{}, err
	}
	path := pendingPath(root)
	if _, err := filesystem.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return RecoverResult{}, nil
		}
		return RecoverResult{}, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	info, err := filesystem.Lstat(path)
	if err != nil {
		return RecoverResult{}, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return RecoverResult{}, err
	}
	document, err := filesystem.ReadFile(path)
	if err != nil {
		return RecoverResult{}, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	marker, err := decodePendingCommit(document)
	if err != nil {
		return RecoverResult{Found: true}, err
	}
	expected, expectedErr := canonicalPairTarget(paths)
	markerPath, markerErr := canonicalPathIdentity(marker.ConfigPath)
	if expectedErr != nil || markerErr != nil || expected != markerPath {
		return RecoverResult{Found: true}, trustError(TrustError{Operation: operation, Err: ErrExclusiveHoldResourceMismatch})
	}
	configuration, err := filesystem.ReadFile(expected)
	if err != nil {
		return RecoverResult{Found: true}, trustError(TrustError{Operation: operation, Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	switch SHA256HexOf(configuration) {
	case marker.ReplacementSHA256:
		committed, err := loadCommitted(filesystem, root)
		if err != nil {
			return RecoverResult{Found: true}, err
		}
		switch committed.Generation {
		case marker.SourceGeneration:
			if err := transactLocked(hold, filesystem, root, func(*TrustStore) error { return nil }); err != nil {
				return RecoverResult{Found: true}, err
			}
		case marker.SourceGeneration + 1:
		default:
			return RecoverResult{Found: true}, trustError(TrustError{Operation: operation, Err: ErrJointIntervened})
		}
		if err := removeMarkerLocked(hold, filesystem, root); err != nil {
			return RecoverResult{Found: true}, err
		}
		return RecoverResult{Found: true, Completed: true}, nil
	case marker.SourceSHA256:
		if err := removeMarkerLocked(hold, filesystem, root); err != nil {
			return RecoverResult{Found: true}, err
		}
		return RecoverResult{Found: true, Aborted: true}, nil
	default:
		return RecoverResult{Found: true}, trustError(TrustError{Operation: operation, Err: ErrJointIntervened})
	}
}

func pendingPath(root string) string { return filepath.Join(root, pendingCommitFileName) }

// markerPresent reports whether a joint-commit intent marker is
// outstanding. Absence must be established, never assumed: an unreadable
// marker path refuses instead of reporting no marker.
func markerPresent(filesystem FileSystem, root string) (bool, error) {
	_, err := filesystem.Lstat(pendingPath(root))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, trustError(TrustError{Operation: "inspect joint commit", Err: errors.Join(ErrTrustStoreUnreadable, err)})
}

// convergeLocked converges one interrupted joint commit, if any, before
// the caller exposes a snapshot or runs a boundary. It is the general
// admission barrier for pending intent: every read, admission and
// transaction entry point converges here under the authorization lock,
// while recovery itself stays the single convergence implementation. The
// caller must hold the exclusive lock: convergence can bump the generation
// and remove the marker. Shared-lock readers upgrade to exclusive when a
// marker is present (see readSnapshot); when no marker is present the
// shared fast path needs no write. A refused recovery fails the caller
// with the marker kept for operator resolution.
func convergeLocked(hold HeldExclusive, filesystem FileSystem, root string, paths localstore.ResolvedPaths) error {
	if err := requireHoldForRoot(hold, root); err != nil {
		return err
	}
	present, err := markerPresent(filesystem, root)
	if err != nil {
		return err
	}
	if !present {
		return nil
	}
	_, err = recoverLocked(hold, filesystem, root, paths)
	return err
}

func pairedIntentTarget(paths localstore.ResolvedPaths, intent PendingCommit) (string, error) {
	expected, expectedErr := canonicalPairTarget(paths)
	markerPath, markerErr := canonicalPathIdentity(intent.ConfigPath)
	if expectedErr != nil || markerErr != nil || expected != markerPath {
		return "", trustError(TrustError{Operation: "validate joint configuration target", Err: ErrExclusiveHoldResourceMismatch})
	}
	return expected, nil
}

func removeMarkerLocked(hold HeldExclusive, filesystem FileSystem, root string) error {
	if err := requireHoldForRoot(hold, root); err != nil {
		return err
	}
	if err := filesystem.Remove(pendingPath(root)); err != nil && !os.IsNotExist(err) {
		return trustError(TrustError{Operation: "remove joint marker", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := syncDir(filesystem, root); err != nil {
		return trustError(TrustError{Operation: "sync joint marker directory", Err: errors.Join(ErrTrustDurability, err)})
	}
	return nil
}
