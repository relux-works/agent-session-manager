package hosttrust

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/relux-works/agent-session-manager/internal/localstore"
)

// Snapshot is one coherent committed read: trust bytes plus the generation
// that authorized them. Streams and dispatches bind this generation; any use
// must re-check currency before crossing a side-effect boundary.
type Snapshot struct {
	Generation uint64
	Trust      TrustStore
}

// TransactionFunc mutates a copy of the committed trust. Returning nil
// commits with generation+1; returning an error aborts with no durable effect.
type TransactionFunc func(trust *TrustStore) error

// ReadSnapshot obtains one coherent committed snapshot under a shared lock.
// A missing store is ErrTrustStoreMissing, never an empty store. Unreadable,
// malformed or partially read state refuses; leftover staging files from
// crashed commits are ignored and never selected.
func (store *Store) ReadSnapshot() (Snapshot, error) {
	return readSnapshot(store.fs, store.root, store.lock, store.paths)
}

// WithSharedSnapshot runs fn under the authorization lock with the
// committed trust, so a config-plus-trust pair observed inside fn is
// coherent: no trust or config commit can land between the trust read and
// fn's own reads. Pending joint intent converges before fn runs, so a
// surviving process observes the same converged pair as a reopened store.
// Callers must not acquire the lock again inside fn; internal reads there
// use the supplied committed value directly.
func (store *Store) WithSharedSnapshot(fn func(current TrustStore) error) error {
	unlock, err := lockForConvergedRead(store.fs, store.root, store.lock, store.paths)
	if err != nil {
		return err
	}
	defer unlock()
	snapshot, err := readSnapshotLocked(store.fs, store.root)
	if err != nil {
		return err
	}
	committed := snapshot.Trust
	if fn == nil {
		return trustError(TrustError{Operation: "read snapshot", Err: ErrAuthorizationRefused})
	}
	return fn(committed)
}

// WithSharedSnapshotForConfig is the paired-reader boundary. It holds the
// converged authorization lock while checking the exact localstore-resolved
// pair and its durable target-side binding, then invokes the reader with the
// committed trust snapshot. The check occurs under the same lock as the read,
// so an out-of-band binding change cannot be separated from the snapshot.
func (store *Store) WithSharedSnapshotForConfig(paths localstore.ResolvedPaths, fn func(current TrustStore) error) error {
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return err
	}
	unlock, err := lockForConvergedRead(store.fs, store.root, store.lock, paths)
	if err != nil {
		return err
	}
	defer unlock()
	if err := store.validateBoundConfigPathLocked(paths); err != nil {
		return err
	}
	snapshot, err := readSnapshotLocked(store.fs, store.root)
	if err != nil {
		return err
	}
	if fn == nil {
		return trustError(TrustError{Operation: "read configuration snapshot", Err: ErrAuthorizationRefused})
	}
	return fn(snapshot.Trust)
}

func readSnapshot(filesystem FileSystem, root string, lock *fileLock, paths localstore.ResolvedPaths) (Snapshot, error) {
	unlock, err := lockForConvergedRead(filesystem, root, lock, paths)
	if err != nil {
		return Snapshot{}, err
	}
	defer unlock()
	return readSnapshotLocked(filesystem, root)
}

// lockForConvergedRead holds the authorization lock for one coherent read,
// converging pending joint intent first. The fast path keeps the shared
// lock when no marker is present: staging a marker needs the exclusive
// lock, so absence observed under shared hold still holds at the read.
// When a marker is present the shared lock is released before the
// exclusive lock is acquired (a held shared lock can never upgrade in
// place without risking a cross-process upgrade deadlock), the leftover
// converges or refuses, and the read proceeds under the exclusive hold.
// Convergence may move the generation forward; callers compare request
// bindings against the converged read, never by refreshing a stale number.
func lockForConvergedRead(filesystem FileSystem, root string, lock *fileLock, paths localstore.ResolvedPaths) (func(), error) {
	unlock, err := lock.shared()
	if err != nil {
		return nil, trustError(TrustError{Operation: "lock snapshot", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	present, err := markerPresent(filesystem, root)
	if err != nil {
		unlock()
		return nil, err
	}
	if !present {
		return unlock, nil
	}
	unlock()
	hold, exclusiveUnlock, err := lock.exclusiveHold()
	if err != nil {
		return nil, trustError(TrustError{Operation: "lock snapshot", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	unlock = exclusiveUnlock
	if err := convergeLocked(hold, filesystem, root, paths); err != nil {
		unlock()
		return nil, err
	}
	return unlock, nil
}

// readSnapshotLocked reads one coherent committed snapshot assuming the
// caller already holds the authorization lock and has converged pending
// joint intent (see lockForConvergedRead): reading beside an unresolved
// marker would admit a superseded generation with replaced configuration.
func readSnapshotLocked(filesystem FileSystem, root string) (Snapshot, error) {
	path := filepath.Join(root, trustFileName)
	info, err := filesystem.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, trustError(TrustError{Operation: "read snapshot", Err: ErrTrustStoreMissing})
		}
		return Snapshot{}, trustError(TrustError{Operation: "read snapshot", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return Snapshot{}, err
	}
	document, err := filesystem.ReadFile(path)
	if err != nil {
		return Snapshot{}, trustError(TrustError{Operation: "read snapshot", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	store, err := DecodeTrust(document)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Generation: store.Generation, Trust: store}, nil
}

// Initialize creates the explicit empty generation-1 store. It refuses when
// trust.json already exists: setup never overwrites committed authority.
// Pending joint intent converges first, so setup never builds beside an
// unresolved marker.
func (store *Store) Initialize() (Snapshot, error) {
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		return Snapshot{}, trustError(TrustError{Operation: "lock initialize", Err: errors.Join(ErrTrustDurability, err)})
	}
	defer unlock()
	if err := store.convergeLocked(hold); err != nil {
		return Snapshot{}, err
	}
	path := filepath.Join(store.root, trustFileName)
	if _, err := store.fs.Lstat(path); err == nil {
		return Snapshot{}, trustError(TrustError{Operation: "initialize store", Err: ErrTrustDurability})
	} else if !os.IsNotExist(err) {
		return Snapshot{}, trustError(TrustError{Operation: "initialize store", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	document, err := EncodeTrust(TrustStore{Generation: 1})
	if err != nil {
		return Snapshot{}, err
	}
	if err := commitDocument(hold, store.fs, store.root, path, document); err != nil {
		return Snapshot{}, err
	}
	// The exclusive lock is still held: read without re-acquiring, since a
	// second acquisition in this goroutine would block on the held lock.
	return readSnapshotLocked(store.fs, store.root)
}

// transact runs one crash-durable generation transaction under the exclusive
// cross-process lock: load committed state, apply fn, encode, stage with
// fsync, atomic rename, directory fsync. A missing store refuses with
// ErrTrustStoreMissing: only explicit Initialize commits generation 1. Any
// other malformed state refuses before fn runs.
func (store *Store) transact(fn TransactionFunc) error {
	return transact(store.fs, store.root, store.lock, store.paths, fn)
}

func transact(filesystem FileSystem, root string, lock *fileLock, paths localstore.ResolvedPaths, fn TransactionFunc) error {
	hold, unlock, err := lock.exclusiveHold()
	if err != nil {
		return trustError(TrustError{Operation: "lock transaction", Err: errors.Join(ErrTrustDurability, err)})
	}
	defer unlock()
	if err := convergeLocked(hold, filesystem, root, paths); err != nil {
		return err
	}
	return transactLocked(hold, filesystem, root, fn)
}

// transactLocked runs one crash-durable generation transaction assuming the
// caller already holds the exclusive authorization lock. Joint commits and
// recovery use it to sequence the marker, the external replacement and the
// bump under a single hold. The hold proves the exclusion window the bump
// commits under; without it the bump refuses.
func transactLocked(hold HeldExclusive, filesystem FileSystem, root string, fn TransactionFunc) error {
	if err := requireHoldForRoot(hold, root); err != nil {
		return err
	}
	committed, err := loadCommitted(filesystem, root)
	if err != nil {
		return err
	}
	next := committed
	next.Entries = append([]CredentialEntry(nil), committed.Entries...)
	if err := fn(&next); err != nil {
		return err
	}
	if next.Generation != committed.Generation {
		return trustError(TrustError{Operation: "commit generation", Err: ErrTrustDurability})
	}
	if committed.Generation >= MaxGeneration {
		return trustError(TrustError{Operation: "commit generation", Err: ErrGenerationExhausted})
	}
	next.Generation = committed.Generation + 1
	encoded, err := EncodeTrust(next)
	if err != nil {
		return err
	}
	if err := commitDocument(hold, filesystem, root, filepath.Join(root, trustFileName), encoded); err != nil {
		return err
	}
	return nil
}

// loadCommitted reads the committed trust without acquiring the lock: the
// caller must already hold the authorization lock, and (outside recovery
// itself) must have converged pending joint intent first.
func loadCommitted(filesystem FileSystem, root string) (TrustStore, error) {
	path := filepath.Join(root, trustFileName)
	info, err := filesystem.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TrustStore{}, trustError(TrustError{Operation: "read committed trust", Err: ErrTrustStoreMissing})
		}
		return TrustStore{}, trustError(TrustError{Operation: "read committed trust", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return TrustStore{}, err
	}
	document, err := filesystem.ReadFile(path)
	if err != nil {
		return TrustStore{}, trustError(TrustError{Operation: "read committed trust", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	decoded, err := DecodeTrust(document)
	if err != nil {
		return TrustStore{}, err
	}
	return decoded, nil
}

// commitDocument stages bytes in a temp file with fsync, atomically renames
// over the target, and fsyncs the directory. On restart only the last fully
// committed file is selected; staged files are ignored by every reader.
// The hold proves the live exclusion window the commit lands under; a
// forged, absent or released hold refuses before anything is staged.
func commitDocument(hold HeldExclusive, filesystem FileSystem, root, path string, document []byte) error {
	if err := requireHoldForRoot(hold, root); err != nil {
		return err
	}
	staged, err := filesystem.CreateTemp(root, stagePrefix+"*", 0o600)
	if err != nil {
		return trustError(TrustError{Operation: "stage trust", Err: errors.Join(ErrTrustDurability, err)})
	}
	name := staged.Name()
	failed := true
	defer cleanupStagedFile(filesystem, name, &failed)
	if err := writeAll(staged, document); err != nil {
		return trustError(TrustError{Operation: "stage trust", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return trustError(TrustError{Operation: "sync trust", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := staged.Close(); err != nil {
		return trustError(TrustError{Operation: "sync trust", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := filesystem.Chmod(name, 0o600); err != nil {
		return trustError(TrustError{Operation: "stage trust", Err: errors.Join(ErrTrustDurability, err)})
	}
	// On platforms where mode bits do not govern access, install the
	// owner-only ACL before the staging file becomes visible under load.
	if err := secureStaged(name, false); err != nil {
		return err
	}
	if err := filesystem.Rename(name, path); err != nil {
		return trustError(TrustError{Operation: "replace trust", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := syncDir(filesystem, root); err != nil {
		return trustError(TrustError{Operation: "sync trust directory", Err: errors.Join(ErrTrustDurability, err)})
	}
	failed = false
	return nil
}

func cleanupStagedFile(filesystem FileSystem, name string, failed *bool) {
	if *failed {
		_ = filesystem.Remove(name)
	}
}

func writeAll(writer interface{ Write([]byte) (int, error) }, document []byte) error {
	for len(document) > 0 {
		written, err := writer.Write(document)
		if err != nil {
			return err
		}
		if written == 0 {
			return errors.New("short trust write")
		}
		document = document[written:]
	}
	return nil
}

func syncDir(filesystem FileSystem, root string) error {
	handle, err := filesystem.OpenDirectory(root)
	if err != nil {
		return err
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		return err
	}
	return handle.Close()
}
