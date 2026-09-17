package hosttrust

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/relux-works/agent-session-manager/internal/localstore"
)

// resolvedPair extracts the only two paths Host Trust Store may use for a
// configuration/trust operation. ResolvedPaths is minted by localstore with
// private fields, so callers cannot construct a tuple by independently
// supplying a config filename and a state directory.
func resolvedPair(paths localstore.ResolvedPaths) (configPath, stateDir string, err error) {
	config, configOK := paths.Path(localstore.PathConfig)
	state, stateOK := paths.Path(localstore.PathState)
	if !configOK || !stateOK {
		return "", "", trustError(TrustError{Operation: "resolve local path pair", Err: ErrInvalidStoreContext})
	}
	configPath, err = absoluteClean(config.Value.String())
	if err != nil {
		return "", "", trustError(TrustError{Operation: "resolve local path pair", Err: errors.Join(ErrInvalidStoreContext, err)})
	}
	stateDir, err = absoluteClean(state.Value.String())
	if err != nil {
		return "", "", trustError(TrustError{Operation: "resolve local path pair", Err: errors.Join(ErrInvalidStoreContext, err)})
	}
	return configPath, stateDir, nil
}

func (store *Store) validateResolvedPair(paths localstore.ResolvedPaths) (string, string, error) {
	configPath, stateDir, err := resolvedPair(paths)
	if err != nil {
		return "", "", err
	}
	ownedConfig, ownedState, err := resolvedPair(store.paths)
	if err != nil || configPath != ownedConfig || stateDir != ownedState {
		return "", "", trustError(TrustError{Operation: "validate local path pair", Err: ErrExclusiveHoldResourceMismatch})
	}
	return configPath, stateDir, nil
}

func (store *Store) resolvedConfigPath() string {
	configPath, _, err := resolvedPair(store.paths)
	if err != nil {
		return ""
	}
	return configPath
}

func (store *Store) convergeLocked(hold HeldExclusive) error {
	return convergeLocked(hold, store.fs, store.root, store.paths)
}

// canonicalPairTarget identifies the selected configuration path from the
// same immutable pair that selected the store root. It permits a missing
// final file so crash recovery can report the underlying unreadable target;
// the parent must still resolve, and a marker can never retarget a different
// symlink-resolved object.
func canonicalPairTarget(paths localstore.ResolvedPaths) (string, error) {
	configPath, _, err := resolvedPair(paths)
	if err != nil {
		return "", err
	}
	return canonicalPathIdentity(configPath)
}

func canonicalPathIdentity(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", ErrInvalidStoreContext
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return filepath.Clean(resolved), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(parent), filepath.Base(abs)), nil
}

// canonicalResolvedConfigPath resolves the selected target only at a
// mutation/read-association boundary. Open may legitimately precede creation
// of an absent config file; binding and joint operations cannot.
func canonicalResolvedConfigPath(paths localstore.ResolvedPaths) (string, error) {
	configPath, _, err := resolvedPair(paths)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalExistingPath(configPath)
	if err != nil {
		return "", trustError(TrustError{Operation: "resolve selected configuration", Err: errors.Join(ErrExclusiveHoldResourceMismatch, err)})
	}
	return canonical, nil
}

func resolvedStateRoot(paths localstore.ResolvedPaths) (string, error) {
	_, stateDir, err := resolvedPair(paths)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalExistingPath(stateDir)
	if err != nil {
		return "", trustError(TrustError{Operation: "resolve selected state root", Err: errors.Join(ErrExclusiveHoldResourceMismatch, err)})
	}
	return canonical, nil
}

func storeStateDir(root string) string { return filepath.Clean(filepath.Dir(root)) }
