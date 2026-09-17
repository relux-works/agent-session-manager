package hosttrust

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// AuthRequest carries one dispatch or mutation-boundary authorization check.
// SnapshotGeneration binds the stream to the coherent snapshot it was opened
// under; LocalCredentialID and the remote DER bytes select the exact enrolled
// entries; Allowlisted is the local mesh allowlist; HelloHostID is the
// authenticated hello identity; ExpectedRemoteHostID is the selected
// configured destination on the client.
type AuthRequest struct {
	SnapshotGeneration   uint64
	LocalHostID          string
	LocalCredentialID    string
	RemoteLeafDER        []byte
	RemoteRootDER        []byte
	ExpectedRemoteHostID string
	Allowlisted          []string
	HelloHostID          string
	Now                  time.Time
}

// Formatting an authorization request must never expose DER bytes, UUIDs,
// digests or allowlists.
func (AuthRequest) String() string   { return "host channel authorization request" }
func (AuthRequest) GoString() string { return "host channel authorization request" }
func (AuthRequest) Format(state fmt.State, verb rune) {
	_, _ = state.Write([]byte("host channel authorization request"))
}

// Formatting a snapshot must never expose the entries it binds.
func (Snapshot) String() string   { return "host trust snapshot" }
func (Snapshot) GoString() string { return "host trust snapshot" }
func (Snapshot) Format(state fmt.State, verb rune) {
	_, _ = state.Write([]byte("host trust snapshot"))
}

// AuthorizeDispatch checks current generation, local credential validity,
// allowlist membership and authenticated hello against one committed read
// held under the shared authorization lock. The check observes a coherent
// snapshot, but the dispatch itself runs after the lock releases, so the
// caller must bind the admitted generation to the stream and enforce the
// one-second invalidation on commit: re-check currency before sending, and
// close stale-generation streams on watchdog, never on notifications alone.
// Mutation side effects additionally require WithMutationAuthorization, which
// holds the exclusive lock across the recheck and the boundary. Work prepared
// under an old generation cannot gain new authority by refreshing a cached
// number: a stale snapshot refuses.
func (store *Store) AuthorizeDispatch(snapshot Snapshot, request AuthRequest) error {
	// Pending joint intent converges before the currency check, so a
	// dispatch bound before a crash compares against the recovered
	// generation and refuses stale instead of admitting superseded trust.
	// Convergence failures name their own operation and pass through: a
	// refused recovery is a durability event, not an authorization verdict.
	unlock, err := lockForConvergedRead(store.fs, store.root, store.lock, store.paths)
	if err != nil {
		return err
	}
	defer unlock()
	current, err := readCommitted(store)
	if err != nil {
		return err
	}
	// The stream binding must match the snapshot it was opened under: work
	// prepared under an old generation cannot gain new authority by
	// refreshing a cached number.
	if request.SnapshotGeneration != snapshot.Generation {
		return trustError(TrustError{Operation: "authorize generation", Err: ErrStaleGeneration})
	}
	if snapshot.Generation != current.Generation {
		return trustError(TrustError{Operation: "authorize generation", Err: ErrStaleGeneration})
	}
	return checkAuthorization(snapshot.Trust, request)
}

// readCommitted reads the committed trust without acquiring the lock: the
// caller must already hold the authorization lock and have converged
// pending joint intent, so the read nests safely inside a holder instead
// of blocking on it.
func readCommitted(store *Store) (TrustStore, error) {
	path := store.TrustPath()
	info, err := store.fs.Lstat(path)
	if err != nil {
		return TrustStore{}, trustError(TrustError{Operation: "authorize generation", Err: errors.Join(ErrAuthorizationRefused, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return TrustStore{}, err
	}
	document, err := store.fs.ReadFile(path)
	if err != nil {
		return TrustStore{}, trustError(TrustError{Operation: "authorize generation", Err: errors.Join(ErrAuthorizationRefused, err)})
	}
	committed, err := DecodeTrust(document)
	if err != nil {
		return TrustStore{}, err
	}
	return committed, nil
}

func checkAuthorization(trust TrustStore, request AuthRequest) error {
	const operation = "authorize dispatch"
	local, found := findEntry(trust, request.LocalCredentialID)
	if !found {
		return trustError(TrustError{Operation: operation, Err: ErrAuthorizationRefused})
	}
	if local.HostID.String() != request.LocalHostID || local.State != EntryActive {
		return trustError(TrustError{Operation: operation, Err: ErrAuthorizationRefused})
	}
	if err := VerifyProfile(local.LeafDER, local.RootDER, local.HostID.String(), request.Now); err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrAuthorizationRefused, err)})
	}
	remote, found := matchRemote(trust, request.RemoteLeafDER, request.RemoteRootDER)
	if !found {
		return trustError(TrustError{Operation: operation, Err: ErrAuthorizationRefused})
	}
	if err := VerifyProfile(remote.LeafDER, remote.RootDER, remote.HostID.String(), request.Now); err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrAuthorizationRefused, err)})
	}
	if remote.State == EntryRevoked {
		return trustError(TrustError{Operation: operation, Err: ErrAuthorizationRefused})
	}
	if remote.State == EntryRetiring {
		if remote.RetireAt == nil {
			return trustError(TrustError{Operation: operation, Err: ErrAuthorizationRefused})
		}
		retireAt, err := remote.RetireAt.Time()
		if err != nil || !request.Now.Before(retireAt) {
			return trustError(TrustError{Operation: operation, Err: ErrAuthorizationRefused})
		}
	}
	allowlisted := false
	for _, hostID := range request.Allowlisted {
		if hostID == remote.HostID.String() {
			allowlisted = true
			break
		}
	}
	if !allowlisted {
		return trustError(TrustError{Operation: operation, Err: ErrNotAllowlisted})
	}
	if request.HelloHostID != remote.HostID.String() || remote.HostID.String() != request.ExpectedRemoteHostID {
		return trustError(TrustError{Operation: operation, Err: ErrHostIdentityMismatch})
	}
	return nil
}

func findEntry(trust TrustStore, credentialID string) (CredentialEntry, bool) {
	for _, entry := range trust.Entries {
		if entry.CredentialID.String() == credentialID {
			return entry, true
		}
	}
	return CredentialEntry{}, false
}

// matchRemote yields the verified UUID only on an exact leaf/root/SPKI match
// against one currently admitted entry. Zero or multiple matches refuse; SAN
// alone and CA membership alone are insufficient, which the byte-exact triple
// comparison enforces structurally.
func matchRemote(trust TrustStore, leafDER, rootDER []byte) (CredentialEntry, bool) {
	spki := spkiOf(leafDER)
	var matched []CredentialEntry
	for _, entry := range trust.Entries {
		if !bytes.Equal(entry.LeafDER, leafDER) || !bytes.Equal(entry.RootDER, rootDER) {
			continue
		}
		if spki == "" || entry.SPKIID.String() != spki {
			continue
		}
		matched = append(matched, entry)
	}
	if len(matched) != 1 {
		return CredentialEntry{}, false
	}
	return matched[0], true
}

func spkiOf(leafDER []byte) string {
	der := mustSPKI(leafDER)
	if len(der) == 0 {
		return ""
	}
	return scalar.SHA256Digest(der).String()
}

// WithMutationAuthorization rechecks generation, credential validity,
// allowlist membership and authenticated hello under the exclusive
// authorization lock immediately before the externally visible side-effect
// boundary, then runs the boundary while still holding the lock. The
// generation check and the boundary serialize with revocation and config
// commits. The request must bind the current generation: work prepared under
// an old generation cannot gain new authority by refreshing a cached number,
// so a stale binding refuses and the caller must discard its authorization
// and replan through a fresh mutually authenticated connection. Boundaries
// must be bounded: long provider calls must not hold the authorization lock
// across unbounded work.
func (store *Store) WithMutationAuthorization(request AuthRequest, boundary func() error) error {
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		return trustError(TrustError{Operation: "authorize mutation", Err: errors.Join(ErrAuthorizationRefused, err)})
	}
	defer unlock()
	// Pending joint intent converges before the binding check: a request
	// bound before a crash compares against the recovered generation and
	// refuses stale, and the boundary never runs beside unresolved intent.
	if err := store.convergeLocked(hold); err != nil {
		return err
	}
	current, err := readCommitted(store)
	if err != nil {
		return err
	}
	if request.SnapshotGeneration != current.Generation {
		return trustError(TrustError{Operation: "authorize mutation", Err: ErrStaleGeneration})
	}
	if err := checkAuthorization(current, request); err != nil {
		return err
	}
	if boundary == nil {
		return trustError(TrustError{Operation: "authorize mutation", Err: ErrAuthorizationRefused})
	}
	return boundary()
}
