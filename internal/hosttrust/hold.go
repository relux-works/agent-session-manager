package hosttrust

import (
	"errors"
	"path/filepath"
	"sync/atomic"

	"github.com/relux-works/agent-session-manager/internal/localstore"
)

// HeldExclusive is the unforgeable capability proving the holder acquired
// the exclusive authorization lock and that the hold is still live. Only
// the lock acquisition path (exclusiveHold) constructs a genuine value;
// every other package sees an opaque struct it can pass along but never
// forge: the zero value and any use-after-release refuse wherever a hold
// is required.
//
// A hold proves serialization, not protocol: helpers that mutate the
// config-plus-trust pair require a live hold (fail closed without one),
// while the write-path census gate statically restricts which call sites
// may perform pair mutations at all.
type HeldExclusive struct {
	hold              *exclusiveHold
	configPath        string
	stateRoot         string
	configAssociation bool
}

// exclusiveHold is the liveness cell behind one HeldExclusive. The unlock
// wrapper marks it released, so a token stashed past its hold refuses
// instead of authorizing a write outside the exclusion window.
type exclusiveHold struct {
	released atomic.Bool
	root     string
}

// Valid reports a genuine live hold: constructed by the lock acquisition
// path and not yet released.
func (held HeldExclusive) Valid() bool {
	return held.hold != nil && !held.hold.released.Load()
}

// IsZero reports the zero value, which never authorizes a write. It lets
// narrowing tests distinguish a forged/absent token from a released one.
func (held HeldExclusive) IsZero() bool {
	return held.hold == nil
}

// requireHold refuses a pair mutation attempted without a live exclusive
// hold. It is the single enforcement point behind every token-gated
// helper; weakening it admits exactly the ungated writes the narrowing
// plants cover.
func requireHold(hold HeldExclusive) error {
	if !hold.Valid() {
		return trustError(TrustError{Operation: "require exclusive hold", Err: ErrExclusiveHoldRequired})
	}
	return nil
}

// requireHoldForRoot proves both liveness and the exact machine-local trust
// resource that the helper is about to mutate. A live capability from a
// different Store is not interchangeable, even when both stores happen to
// be held by the same goroutine.
func requireHoldForRoot(hold HeldExclusive, root string) error {
	if err := requireHold(hold); err != nil {
		return err
	}
	canonical, err := absoluteClean(root)
	if err != nil || hold.hold.root != canonical {
		return trustError(TrustError{Operation: "require exclusive hold resource", Err: ErrExclusiveHoldResourceMismatch})
	}
	return nil
}

// ValidateConfigPaths proves that a live capability was bound by this store's
// durable immutable configuration association to the exact path it is about
// to replace. Generic stores intentionally mint no configuration binding, so
// a token from an unrelated store cannot authorize a pair write by accident.
func (held HeldExclusive) ValidateConfigPaths(paths localstore.ResolvedPaths) error {
	if err := requireHold(held); err != nil {
		return err
	}
	canonical, err := canonicalResolvedConfigPath(paths)
	stateRoot, stateErr := resolvedStateRoot(paths)
	if err != nil || stateErr != nil || !held.configAssociation || held.configPath == "" || held.stateRoot == "" || held.configPath != canonical || held.stateRoot != stateRoot {
		return trustError(TrustError{Operation: "require configuration hold resource", Err: ErrExclusiveHoldResourceMismatch})
	}
	return nil
}

func absoluteClean(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", ErrInvalidStoreContext
	}
	return filepath.Clean(path), nil
}

// canonicalExistingPath returns the identity used by the pair writer. The
// identity follows symlinks before comparing paths, and a missing/unreadable
// target is a refusal rather than an invitation to mint a new binding.
func canonicalExistingPath(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", ErrInvalidStoreContext
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

// exclusiveHold acquires the exclusive authorization lock and returns the
// capability proving the hold plus its release function. The release marks
// the capability expired before unlocking, so no write can slip outside
// the exclusion window it was authorized under.
func (lock *fileLock) exclusiveHold() (HeldExclusive, func(), error) {
	unlock, err := lock.exclusive()
	if err != nil {
		return HeldExclusive{}, nil, err
	}
	state := &exclusiveHold{root: filepath.Clean(filepath.Dir(lock.path))}
	return HeldExclusive{hold: state}, func() {
		state.released.Store(true)
		unlock()
	}, nil
}

// WithExclusiveHold runs fn under the exclusive authorization lock with
// the capability proving the hold. Pending joint intent converges first,
// so fn never runs beside unresolved intent; fn receives the live hold to
// pass to token-gated pair writers. Legacy configuration migration uses
// this to serialize with joint commits; credential lifecycle and joint
// commits take their holds through their own entries.
func (store *Store) WithExclusiveHold(fn func(HeldExclusive) error) error {
	return store.withExclusiveHold(fn)
}

// WithExclusiveHoldForConfig runs a configuration-side mutation under the
// store's live exclusive hold only when the caller presents the exact
// localstore-resolved pair with which this store was opened and explicitly
// bound. A generic hold can serialize credential-only work but can never lend
// its liveness to a configuration writer.
func (store *Store) WithExclusiveHoldForConfig(paths localstore.ResolvedPaths, fn func(HeldExclusive) error) error {
	if _, _, err := store.validateResolvedPair(paths); err != nil {
		return err
	}
	return store.withExclusiveHold(func(hold HeldExclusive) error {
		if err := hold.ValidateConfigPaths(paths); err != nil {
			return err
		}
		if fn == nil {
			return trustError(TrustError{Operation: "run configuration hold", Err: ErrAuthorizationRefused})
		}
		return fn(hold)
	})
}

func (store *Store) withExclusiveHold(fn func(HeldExclusive) error) error {
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		return trustError(TrustError{Operation: "lock exclusive hold", Err: errors.Join(ErrTrustDurability, err)})
	}
	defer unlock()
	if err := store.convergeLocked(hold); err != nil {
		return err
	}
	if fn == nil {
		return trustError(TrustError{Operation: "run exclusive hold", Err: ErrAuthorizationRefused})
	}
	// Refresh both sides of the immutable association under the hold. The
	// state-side index is only a derived cache; the target-side record is the
	// authority for pair-write capability. A missing target record narrows a
	// live hold to generic trust serialization, while a present malformed or
	// unreadable record refuses closed.
	binding, present, err := loadStoreConfigBinding(store.fs, store.root)
	if err != nil {
		return err
	}
	store.configPath = ""
	if !present {
		return fn(hold)
	}
	store.configPath = binding.ConfigPath
	target, targetPresent, err := loadTargetConfigBinding(store.fs, binding.ConfigPath)
	if err != nil {
		return err
	}
	associated := targetPresent && validateBindingForStore(target, binding.ConfigPath, store.root) == nil
	hold = HeldExclusive{hold: hold.hold, configPath: binding.ConfigPath, stateRoot: binding.StateRoot, configAssociation: associated}
	return fn(hold)
}
