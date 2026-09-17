package hosttrust

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/relux-works/agent-session-manager/internal/localstore"
)

const (
	// The store-side file is a derived index retained for fast reopen. It is
	// never the authority for a config replacement: the config-side record is.
	configBindingFileName = "config-binding.json"
	configBindingSchema   = "urn:ax:schema:host-config-binding"
	configBindingVersion  = "1.0.0"

	// A configuration-side association is keyed by the canonical target path,
	// so two configurable configuration files in one directory cannot share a
	// binding or its bootstrap lock. Both files are local control metadata and
	// are excluded from every replication/export surface.
	configBindingTargetSuffix = ".ax-config-binding.json"
	configBindingLockSuffix   = ".ax-config-binding.lock"
	configBindingStagePrefix  = ".ax-config-binding-stage-"
)

// ConfigBinding is the machine-local, immutable association between one Host
// Trust Store and one configuration document. It is deliberately a sidecar,
// not a mesh.host_channel member: Configuration 4.0.0's host_channel table is
// an exact closed two-field shape. The config-side record is authoritative;
// the state-root field binds that exact target to one store coordination root.
// Neither sidecar enters the configuration or any replication surface.
type ConfigBinding struct {
	Schema        string `json:"schema"`
	SchemaVersion string `json:"schema_version"`
	ConfigPath    string `json:"config_path"`
	StateRoot     string `json:"state_root"`
}

// bindingPath is the derived store-side index path. It is intentionally kept
// separate from configBindingPath: callers must not be able to turn a
// caller-selected store index into authority for a different configuration.
func bindingPath(root string) string { return filepath.Join(root, configBindingFileName) }

func configBindingPath(configPath string) string { return configPath + configBindingTargetSuffix }

func configBindingLockPath(configPath string) string { return configPath + configBindingLockSuffix }

func encodeConfigBinding(path, stateRoot string) ([]byte, error) {
	if err := validateBindingPath(path); err != nil {
		return nil, err
	}
	if err := validateBindingStateRoot(stateRoot); err != nil {
		return nil, err
	}
	document, err := json.Marshal(ConfigBinding{
		Schema:        configBindingSchema,
		SchemaVersion: configBindingVersion,
		ConfigPath:    path,
		StateRoot:     stateRoot,
	})
	if err != nil {
		return nil, trustError(TrustError{Operation: "encode configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	return document, nil
}

// decodeConfigBinding keeps the historical path-only helper for tests and
// diagnostics. All authority checks use decodeConfigBindingRecord so the
// state-root association cannot be discarded accidentally.
func decodeConfigBinding(document []byte) (string, error) {
	binding, err := decodeConfigBindingRecord(document)
	if err != nil {
		return "", err
	}
	return binding.ConfigPath, nil
}

func decodeConfigBindingRecord(document []byte) (ConfigBinding, error) {
	if err := rejectDuplicateKeys(document); err != nil {
		return ConfigBinding{}, err
	}
	var shape map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(document))
	if err := decoder.Decode(&shape); err != nil {
		return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: errors.Join(ErrTrustDecode, err)})
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: ErrTrustDecode})
		}
		return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: errors.Join(ErrTrustDecode, err)})
	}
	if len(shape) != 4 {
		return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: ErrTrustValidation})
	}
	for key := range shape {
		switch key {
		case "schema", "schema_version", "config_path", "state_root":
		default:
			return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: ErrTrustValidation})
		}
	}
	var binding ConfigBinding
	decoder = json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&binding); err != nil {
		return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: errors.Join(ErrTrustDecode, err)})
	}
	if binding.Schema != configBindingSchema || binding.SchemaVersion != configBindingVersion {
		return ConfigBinding{}, trustError(TrustError{Operation: "decode configuration binding", Err: ErrTrustValidation})
	}
	if err := validateBindingPath(binding.ConfigPath); err != nil {
		return ConfigBinding{}, err
	}
	if err := validateBindingStateRoot(binding.StateRoot); err != nil {
		return ConfigBinding{}, err
	}
	return binding, nil
}

func validateBindingPath(path string) error {
	canonical, err := absoluteClean(path)
	if err != nil || canonical != path {
		return trustError(TrustError{Operation: "validate configuration binding", Err: ErrTrustValidation})
	}
	return nil
}

func validateBindingStateRoot(stateRoot string) error {
	canonical, err := absoluteClean(stateRoot)
	if err != nil || canonical != stateRoot {
		return trustError(TrustError{Operation: "validate configuration binding state root", Err: ErrTrustValidation})
	}
	return nil
}

// loadBindingAt reads one exact closed owner-only binding document. Absence
// is returned explicitly; a present but partial, unreadable, downgraded or
// ambiguous document refuses closed.
func loadBindingAt(filesystem FileSystem, path string) (ConfigBinding, bool, error) {
	info, err := filesystem.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConfigBinding{}, false, nil
		}
		return ConfigBinding{}, false, trustError(TrustError{Operation: "read configuration binding", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return ConfigBinding{}, false, err
	}
	document, err := filesystem.ReadFile(path)
	if err != nil {
		return ConfigBinding{}, false, trustError(TrustError{Operation: "read configuration binding", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	binding, err := decodeConfigBindingRecord(document)
	if err != nil {
		return ConfigBinding{}, false, err
	}
	return binding, true, nil
}

// loadConfigBinding reads the derived store-side index. The returned path is
// intentionally insufficient for authority; callers that authorize a target
// must use loadTargetConfigBinding and compare StateRoot to their store.
func loadConfigBinding(filesystem FileSystem, root string) (string, error) {
	binding, present, err := loadBindingAt(filesystem, bindingPath(root))
	if err != nil {
		return "", err
	}
	if !present {
		return "", nil
	}
	return binding.ConfigPath, nil
}

func loadStoreConfigBinding(filesystem FileSystem, root string) (ConfigBinding, bool, error) {
	return loadBindingAt(filesystem, bindingPath(root))
}

func loadTargetConfigBinding(filesystem FileSystem, configPath string) (ConfigBinding, bool, error) {
	return loadBindingAt(filesystem, configBindingPath(configPath))
}

func storeStateRoot(root string) (string, error) {
	stateRoot, err := canonicalExistingPath(filepath.Dir(root))
	if err != nil {
		return "", trustError(TrustError{Operation: "resolve store state root", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	return stateRoot, nil
}

func validateBindingForStore(binding ConfigBinding, canonical, root string) error {
	stateRoot, err := storeStateRoot(root)
	if err != nil {
		return err
	}
	if binding.ConfigPath != canonical || binding.StateRoot != stateRoot {
		return trustError(TrustError{Operation: "validate configuration binding", Err: ErrExclusiveHoldResourceMismatch})
	}
	return nil
}

// ensureConfigBindingLocked converges an immutable binding while the caller
// holds the store's exclusive authorization lock. The config-side record and
// its bootstrap lock are the authority: a second Store cannot claim the
// same target by supplying a different stateDir. The store-side index is
// written only as a derived cache after the authoritative record is present.
func (store *Store) ensureConfigBindingLocked(hold HeldExclusive, paths localstore.ResolvedPaths) (HeldExclusive, error) {
	if err := requireHoldForRoot(hold, store.root); err != nil {
		return HeldExclusive{}, err
	}
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return HeldExclusive{}, err
	}
	canonical, err := canonicalResolvedConfigPath(paths)
	if err != nil {
		return HeldExclusive{}, err
	}
	if err := validateBindingPath(canonical); err != nil {
		return HeldExclusive{}, err
	}
	stateRoot, err := resolvedStateRoot(paths)
	if err != nil {
		return HeldExclusive{}, err
	}
	bootstrapLock, err := openLock(store.fs, configBindingLockPath(canonical))
	if err != nil {
		return HeldExclusive{}, trustError(TrustError{Operation: "open configuration binding lock", Err: err})
	}
	unlockBootstrap, err := bootstrapLock.exclusive()
	if err != nil {
		return HeldExclusive{}, trustError(TrustError{Operation: "lock configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	defer unlockBootstrap()

	// Inspect the store-side association before publishing anything at the
	// target. A Store can have been opened while unbound, then another handle
	// can bind the shared root before this request reaches the bootstrap lock.
	// The existing store association is authoritative for that root; discovering
	// a mismatch must therefore be side-effect free.
	storeBinding, storePresent, err := loadStoreConfigBinding(store.fs, store.root)
	if err != nil {
		return HeldExclusive{}, err
	}
	if storePresent {
		if err := validateBindingForStore(storeBinding, canonical, store.root); err != nil {
			return HeldExclusive{}, err
		}
	}

	binding, present, err := loadTargetConfigBinding(store.fs, canonical)
	if err != nil {
		return HeldExclusive{}, err
	}
	if present {
		if err := validateBindingForStore(binding, canonical, store.root); err != nil {
			return HeldExclusive{}, err
		}
		if storePresent && (binding.ConfigPath != storeBinding.ConfigPath || binding.StateRoot != storeBinding.StateRoot) {
			return HeldExclusive{}, trustError(TrustError{Operation: "validate configuration binding", Err: ErrExclusiveHoldResourceMismatch})
		}
	} else if storePresent {
		// An existing derived index without its authoritative target record is
		// incoherent. Refuse closed; do not recreate the target from the cache.
		return HeldExclusive{}, trustError(TrustError{Operation: "validate configuration binding", Err: ErrExclusiveHoldResourceMismatch})
	} else {
		document, err := encodeConfigBinding(canonical, stateRoot)
		if err != nil {
			return HeldExclusive{}, err
		}
		if err := store.commitConfigBinding(hold, paths, document); err != nil {
			return HeldExclusive{}, err
		}
		binding = ConfigBinding{Schema: configBindingSchema, SchemaVersion: configBindingVersion, ConfigPath: canonical, StateRoot: stateRoot}
	}

	// Persist the derived state-side index only after the config-side
	// association is durable. A crash between the two leaves a safe partial
	// state: readers cannot use the target without repairing it explicitly.
	if !storePresent {
		document, err := json.Marshal(binding)
		if err != nil {
			return HeldExclusive{}, trustError(TrustError{Operation: "encode configuration binding", Err: errors.Join(ErrTrustDurability, err)})
		}
		if err := commitDocument(hold, store.fs, store.root, bindingPath(store.root), document); err != nil {
			return HeldExclusive{}, err
		}
	}
	store.configPath = canonical
	return HeldExclusive{hold: hold.hold, configPath: canonical, stateRoot: stateRoot, configAssociation: true}, nil
}

// commitConfigBinding publishes the authoritative target-side association
// with the same staged-write, fsync, atomic-rename and directory-fsync
// discipline as trust.json. The store hold proves the owning trust resource;
// the per-target bootstrap lock serializes first association across stores.
func (store *Store) commitConfigBinding(hold HeldExclusive, paths localstore.ResolvedPaths, document []byte) error {
	if err := requireHoldForRoot(hold, store.root); err != nil {
		return err
	}
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return err
	}
	configPath, err := canonicalResolvedConfigPath(paths)
	if err != nil {
		return err
	}
	directory := filepath.Dir(configPath)
	// The selected configuration directory is operator-owned input and may
	// legitimately be shared with other configuration files. The binding
	// record and its bootstrap lock are the owner-only control files; do not
	// silently impose Host Trust Store's 0700 directory custody on that
	// unrelated directory.
	staged, err := store.fs.CreateTemp(directory, configBindingStagePrefix+"*", 0o600)
	if err != nil {
		return trustError(TrustError{Operation: "stage configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	name := staged.Name()
	failed := true
	defer cleanupStagedFile(store.fs, name, &failed)
	if err := writeAll(staged, document); err != nil {
		return trustError(TrustError{Operation: "stage configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return trustError(TrustError{Operation: "sync configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := staged.Close(); err != nil {
		return trustError(TrustError{Operation: "sync configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := store.fs.Chmod(name, 0o600); err != nil {
		return trustError(TrustError{Operation: "stage configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := secureStaged(name, false); err != nil {
		return err
	}
	if err := store.fs.Rename(name, configBindingPath(configPath)); err != nil {
		return trustError(TrustError{Operation: "replace configuration binding", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := syncDir(store.fs, directory); err != nil {
		return trustError(TrustError{Operation: "sync configuration binding directory", Err: errors.Join(ErrTrustDurability, err)})
	}
	failed = false
	return nil
}

// boundConfigHoldLocked returns a configuration-bound capability only when
// the authoritative config-side association already exists and repeats the
// marker's canonical target and this store's coordination root. JointCommit
// never bootstraps a new association from a marker.
func (store *Store) boundConfigHoldLocked(hold HeldExclusive, paths localstore.ResolvedPaths) (HeldExclusive, error) {
	if err := requireHoldForRoot(hold, store.root); err != nil {
		return HeldExclusive{}, err
	}
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return HeldExclusive{}, err
	}
	canonical, err := canonicalResolvedConfigPath(paths)
	if err != nil {
		return HeldExclusive{}, err
	}
	if err := validateBindingPath(canonical); err != nil {
		return HeldExclusive{}, err
	}
	binding, present, err := loadTargetConfigBinding(store.fs, canonical)
	if err != nil {
		return HeldExclusive{}, err
	}
	if !present {
		return HeldExclusive{}, trustError(TrustError{Operation: "validate configuration hold resource", Err: ErrExclusiveHoldResourceMismatch})
	}
	if err := validateBindingForStore(binding, canonical, store.root); err != nil {
		return HeldExclusive{}, err
	}
	store.configPath = canonical
	return HeldExclusive{hold: hold.hold, configPath: canonical, stateRoot: binding.StateRoot, configAssociation: true}, nil
}
