package hosttrust

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/relux-works/agent-session-manager/internal/localstore"
)

const (
	hostChannelDir  = "host-channel"
	trustFileName   = "trust.json"
	lockFileName    = "lock"
	credentialsDir  = "credentials"
	certificateFile = "certificate.pem"
	privateKeyFile  = "private-key.pem"
	rootFile        = "root.pem"
	stagePrefix     = ".trust-stage-"
)

// FileSystem abstracts durable writes so crash and fault paths are testable
// without touching the operator disk. The OS implementation is the only
// production backend.
type FileSystem interface {
	CreateTemp(dir, pattern string, mode fs.FileMode) (StagedFile, error)
	// CreateExclusive atomically creates a file that must not exist: it
	// fails when the path already exists and never replaces a live file.
	CreateExclusive(path string, mode fs.FileMode) error
	ReadFile(name string) ([]byte, error)
	Lstat(name string) (fs.FileInfo, error)
	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
	Rename(oldname, newname string) error
	MkdirAll(path string, mode fs.FileMode) error
	Chmod(name string, mode fs.FileMode) error
	OpenDirectory(name string) (SyncedDirectory, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

// StagedFile is one temp staging file awaiting fsync and atomic rename.
type StagedFile interface {
	io.Writer
	Name() string
	Chmod(fs.FileMode) error
	Sync() error
	Close() error
}

// SyncedDirectory fsyncs a directory entry after renames.
type SyncedDirectory interface {
	Sync() error
	Close() error
}

type osFileSystem struct{}

func (osFileSystem) CreateTemp(dir, pattern string, mode fs.FileMode) (StagedFile, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

func (osFileSystem) CreateExclusive(path string, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	return file.Close()
}

func (osFileSystem) ReadFile(name string) ([]byte, error)   { return os.ReadFile(name) }
func (osFileSystem) Lstat(name string) (fs.FileInfo, error) { return os.Lstat(name) }
func (osFileSystem) Stat(name string) (fs.FileInfo, error)  { return os.Stat(name) }
func (osFileSystem) Remove(name string) error               { return os.Remove(name) }
func (osFileSystem) Rename(oldname, newname string) error   { return os.Rename(oldname, newname) }

func (osFileSystem) MkdirAll(path string, mode fs.FileMode) error { return os.MkdirAll(path, mode) }
func (osFileSystem) Chmod(name string, mode fs.FileMode) error    { return os.Chmod(name, mode) }

func (osFileSystem) OpenDirectory(name string) (SyncedDirectory, error) { return os.Open(name) }
func (osFileSystem) ReadDir(name string) ([]fs.DirEntry, error)         { return os.ReadDir(name) }

// Store is machine-local Host Channel authority rooted at
// STATE_DIR/host-channel. The zero value is unusable; construct with Open.
type Store struct {
	root       string
	configPath string
	paths      localstore.ResolvedPaths
	fs         FileSystem
	lock       *fileLock
}

// Open validates owner-only custody of the resolved StateRoot/host-channel,
// creating the
// directory skeleton with exact 0700/0600 modes when absent. Every open
// re-verifies custody: symlink escapes, foreign ownership and group/world
// access refuse with ErrTrustUnsafeCustody.
func Open(paths localstore.ResolvedPaths) (*Store, error) { return openStore(paths, osFileSystem{}) }

func openStore(paths localstore.ResolvedPaths, filesystem FileSystem) (*Store, error) {
	if filesystem == nil {
		return nil, trustError(TrustError{Operation: "open store", Err: ErrInvalidStoreContext})
	}
	_, stateDir, err := resolvedPair(paths)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(stateDir, hostChannelDir)
	if err := ensureOwnerDir(filesystem, root, 0o700); err != nil {
		return nil, err
	}
	if err := ensureOwnerDir(filesystem, filepath.Join(root, credentialsDir), 0o700); err != nil {
		return nil, err
	}
	lock, err := openLock(filesystem, filepath.Join(root, lockFileName))
	if err != nil {
		return nil, err
	}
	store := &Store{root: root, paths: paths, fs: filesystem, lock: lock}
	binding, present, err := loadStoreConfigBinding(filesystem, root)
	if err != nil {
		return nil, err
	}
	if present {
		canonical, err := canonicalResolvedConfigPath(paths)
		if err != nil {
			return nil, err
		}
		if err := validateBindingForStore(binding, canonical, root); err != nil {
			return nil, err
		}
		targetBinding, targetPresent, err := loadTargetConfigBinding(filesystem, canonical)
		if err != nil {
			return nil, err
		}
		if !targetPresent {
			return nil, trustError(TrustError{Operation: "open store configuration binding", Err: ErrExclusiveHoldResourceMismatch})
		}
		if err := validateBindingForStore(targetBinding, canonical, root); err != nil {
			return nil, err
		}
		store.configPath = binding.ConfigPath
	}
	// Converge one interrupted joint config-plus-trust commit, if any: a
	// crash between the configuration replacement and the generation bump
	// must complete forward or abort on restart, never admit the new
	// configuration with the old generation. A refused recovery fails the
	// open so incoherent state cannot be used.
	if _, err := store.Recover(); err != nil {
		return nil, err
	}
	return store, nil
}

// Root reports the validated host-channel directory. It carries no secrets.
func (store *Store) Root() string { return store.root }

// EnsureConfigBinding establishes the one immutable configuration association
// for this machine-local store. It is an explicit bootstrap operation, not a
// constructor shortcut: the binding is persisted atomically under the same
// exclusive lock used by trust mutations, and a later request for another
// path refuses. The configuration-side StateRoot is checked in the same
// operation, so a foreign store cannot bind itself to a target document even
// when the caller supplies that document's path. Callers that only need
// credential custody may leave a store unbound; pair readers and writers
// require this association.
func (store *Store) EnsureConfigBinding(paths localstore.ResolvedPaths) error {
	configPath, _, err := store.validateResolvedPair(paths)
	if err != nil {
		return err
	}
	if _, err := canonicalExistingPath(configPath); err != nil {
		return trustError(TrustError{Operation: "establish configuration binding", Err: errors.Join(ErrInvalidStoreContext, err)})
	}
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		return trustError(TrustError{Operation: "lock configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	defer unlock()
	if err := store.convergeLocked(hold); err != nil {
		return err
	}
	_, err = store.ensureConfigBindingLocked(hold, paths)
	return err
}

// ValidateBoundConfigPath checks the durable binding and the current target
// identity. It takes a shared lock so a caller cannot validate against a
// deleted or malformed sidecar and then proceed as if the association still
// existed. The in-hold pair writers perform the same check through the opaque
// HeldExclusive capability immediately before mutation.
func (store *Store) ValidateBoundConfigPath(paths localstore.ResolvedPaths) error {
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return err
	}
	unlock, err := store.lock.shared()
	if err != nil {
		return trustError(TrustError{Operation: "read configuration binding", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	defer unlock()
	return store.validateBoundConfigPathLocked(paths)
}

func (store *Store) validateBoundConfigPathLocked(paths localstore.ResolvedPaths) error {
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return err
	}
	canonical, err := canonicalResolvedConfigPath(paths)
	if err != nil {
		return trustError(TrustError{Operation: "validate configuration binding", Err: ErrExclusiveHoldResourceMismatch})
	}
	binding, present, err := loadTargetConfigBinding(store.fs, canonical)
	if err != nil {
		return err
	}
	if !present {
		return trustError(TrustError{Operation: "validate configuration binding", Err: ErrExclusiveHoldResourceMismatch})
	}
	if err := validateBindingForStore(binding, canonical, store.root); err != nil {
		return trustError(TrustError{Operation: "validate configuration binding", Err: ErrExclusiveHoldResourceMismatch})
	}
	return nil
}

// ValidateBoundPaths is the coherent-read association check. The resolved
// pair must be the exact pair with which this Store was opened, and the
// durable target-side binding must still name both that config target and the
// store's state root.
func (store *Store) ValidateBoundPaths(paths localstore.ResolvedPaths) error {
	return store.ValidateBoundConfigPath(paths)
}

// TrustPath reports the trust.json path for operator diagnostics. Reading or
// writing it directly bypasses the authorization lock; use transactions.
func (store *Store) TrustPath() string { return filepath.Join(store.root, trustFileName) }

// PendingCommitPath reports the pending-commit.json path for operator
// diagnostics. A marker left here after a crash converges automatically on
// the next converged read; a marker that refuses convergence needs
// operator resolution and must never be deleted blindly.
func (store *Store) PendingCommitPath() string { return pendingPath(store.root) }

// CredentialDir reports the custody directory for one credential ID hex. It
// carries no key material.
func (store *Store) CredentialDir(credentialHex string) (string, error) {
	if len(credentialHex) != 64 {
		return "", trustError(TrustError{Operation: "resolve credential directory", Err: ErrCredentialCustody})
	}
	for _, unit := range credentialHex {
		if (unit < '0' || unit > '9') && (unit < 'a' || unit > 'f') {
			return "", trustError(TrustError{Operation: "resolve credential directory", Err: ErrCredentialCustody})
		}
	}
	return filepath.Join(store.root, credentialsDir, credentialHex), nil
}

func ensureOwnerDir(filesystem FileSystem, path string, mode fs.FileMode) error {
	info, err := filesystem.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return trustError(TrustError{Operation: "inspect directory", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err == nil {
		// An existing directory is verified, never repaired: silently
		// narrowing a widened directory would mask the custody violation.
		return verifyOwnerDir(path, info, mode)
	}
	if err := filesystem.MkdirAll(path, mode); err != nil {
		return trustError(TrustError{Operation: "create directory", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	// MkdirAll honors umask, so the exact owner-only mode is enforced
	// explicitly after creation.
	if err := filesystem.Chmod(path, mode); err != nil {
		return trustError(TrustError{Operation: "create directory", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	// On platforms where mode bits do not govern access, install the
	// owner-only ACL at creation: verification below would otherwise
	// refuse the directory the store just created.
	if err := secureStaged(path, true); err != nil {
		return err
	}
	info, err = filesystem.Lstat(path)
	if err != nil {
		return trustError(TrustError{Operation: "inspect directory", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	return verifyOwnerDir(path, info, mode)
}

func verifyOwnerDir(path string, info fs.FileInfo, mode fs.FileMode) error {
	if !info.IsDir() {
		return trustError(TrustError{Operation: "inspect directory", Err: ErrTrustUnsafeCustody})
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return trustError(TrustError{Operation: "inspect directory", Err: ErrTrustUnsafeCustody})
	}
	// Permission bits are a Unix signal only: on Windows the DACL governs and
	// Go synthesizes mode bits from the read-only attribute, so an exact
	// comparison would refuse every Windows path. verifyOwner enforces the
	// owner-only DACL there.
	if !platformModeOK(info, mode) {
		return trustError(TrustError{Operation: "inspect directory", Err: ErrTrustUnsafeCustody})
	}
	return verifyOwner(path, info)
}

// lockVerifyBudget bounds verification retries on an existing lock file.
// A concurrent first-opener can observe the lock between its atomic
// creation and the creator's mode/ACL install; the window is one Chmod and
// one ACL install, so a fraction of a second closes it on any supported
// platform. Genuine custody violations still refuse once the budget
// expires.
const lockVerifyBudget = 500 * time.Millisecond

const lockVerifyRetry = 5 * time.Millisecond

// verifyLockFile verifies owner-only custody of an existing lock file. A
// non-regular file refuses immediately: lock creation only ever publishes
// regular files, so a symlink or directory at the path can never be a
// half-installed lock. Other verification failures retry within the
// budget before refusing, so a racing first-open never fails spuriously.
func verifyLockFile(filesystem FileSystem, path string) error {
	deadline := time.Now().Add(lockVerifyBudget)
	for {
		info, err := filesystem.Lstat(path)
		if err != nil {
			return trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustStoreUnreadable, err)})
		}
		if !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
			return verifyOwnerFile(path, info, 0o600)
		}
		if err := verifyOwnerFile(path, info, 0o600); err == nil {
			return nil
		} else if time.Now().After(deadline) {
			return err
		}
		time.Sleep(lockVerifyRetry)
	}
}

func verifyOwnerFile(path string, info fs.FileInfo, mode fs.FileMode) error {
	if !info.Mode().IsRegular() {
		return trustError(TrustError{Operation: "inspect file", Err: ErrTrustUnsafeCustody})
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return trustError(TrustError{Operation: "inspect file", Err: ErrTrustUnsafeCustody})
	}
	if !platformModeOK(info, mode) {
		return trustError(TrustError{Operation: "inspect file", Err: ErrTrustUnsafeCustody})
	}
	return verifyOwner(path, info)
}
