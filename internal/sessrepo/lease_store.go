// Lease lifecycle: creation, compare-and-swap succession, fencing
// revalidation, and the operational grant-expiry policy.
//
// Normative scope: AX v0.6.0 Sections 2.2 (global invariants), 5.3 (Lease
// Record and ownership), and 13.6-13.10 (takeover, fork, stop, resume).
// This file owns the durable Lease Record 1.0.0 lifecycle inside
// sessrepo: minting epoch-1 create leases from the durable bootstrap
// inputs, persisting successor leases through compare-and-swap writes,
// reading the winning lease by the greatest (epoch, lease_id) tuple,
// and revalidating presented fencing tokens.
//
// Division of authority (no second lease model):
//
//   - Closed shape, self identity, and the three Section 5.3 couplings
//     (predecessor presence, epoch-1-create predecessor/checkpoint rules)
//     are owned by internal/canonicaljson. Minting computes record_id
//     through CalculateObjectIdentity and confirms through
//     VerifyObjectIdentity; loading re-verifies every blob the same way.
//     This file never re-decodes a frame, never recomputes a bound, and
//     never selects a self field.
//   - The winning-tuple rule is owned by internal/sessstate (Compare).
//     This package cannot import sessstate (sessstate/project.go imports
//     sessrepo), so CompareLeaseTuple restates the rule for stored
//     summaries and a cross-package agreement test pins identical order.
//     Query-layer admission (internal/sessquery winningLeaseFor) is
//     unchanged; an adoption test proves records minted here admit there.
//   - Checkpoint-record admission stays with sessquery. CAS requires a
//     well-formed checkpoint digest for every successor (Section 5.3
//     "otherwise the validated materialized handoff base"); validating
//     that the digest names an admitted checkpoint for the predecessor
//     lease belongs to the query layer, not to this write path.
//
// Expiry policy (Section 5.3: "There is no time-expiring ownership
// lease"; liveness is not authority). The lease itself never expires:
// only a takeover or fork advances ownership, and created_at is
// diagnostic only. What lapses is the operational fencing grant — the
// process-local authorization derived from a winning lease — after the
// configured lease refresh interval without revalidation. CheckFencingExpiry
// refuses a lapsed grant so the holder revalidates; it never retires the
// lease. Holder identity is the host UUIDv7 bound in holder_host_id; any
// process binding is machine-local and never persisted cross-host
// (Section 2.2 invariants 4-5).
package sessrepo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	// ErrInvalidLease refuses a lease input or fencing token whose grammar
	// or semantics the Section 5.3 record shape cannot carry.
	ErrInvalidLease = errors.New("invalid lease record")
	// ErrLeaseExists refuses a second epoch-1 create carrying bytes that
	// differ from the persisted lease.
	ErrLeaseExists = errors.New("lease already exists")
	// ErrUnknownLease refuses a well-formed lease reference that names no
	// persisted lease, including a fencing token from an epoch beyond the
	// winner and a CAS against a session with no lease yet.
	ErrUnknownLease = errors.New("unknown lease")
	// ErrLeaseConflict is the Section 13.7 lease_conflict refusal: a CAS
	// expectation that does not name the current head, or a fencing token
	// whose lease ID loses the same-epoch tie.
	ErrLeaseConflict = errors.New("lease_conflict")
	// ErrStaleLeaseEpoch refuses a CAS expectation or fencing token from a
	// superseded epoch.
	ErrStaleLeaseEpoch = errors.New("stale lease epoch")
	// ErrLeaseHolderMismatch refuses a fencing token presented by a host
	// other than the winning lease holder.
	ErrLeaseHolderMismatch = errors.New("lease_holder_mismatch")
	// ErrFencingExpired refuses an operational fencing grant that lapsed
	// past the refresh interval. The underlying lease stays authoritative;
	// the holder revalidates and continues.
	ErrFencingExpired = errors.New("fencing_grant_expired")
)

// leaseRecordSchema is the exact Lease Record schema this store mints.
const leaseRecordSchema = "urn:ax:schema:lease"

// leaseIdentityPlaceholder fills record_id while the canonical owner
// computes the omit-self digest. The value is never persisted: minting
// replaces it with the calculated identity before Verify.
const leaseIdentityPlaceholder = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// leaseStagePrefix marks staged lease blobs. The loader ignores these
// names: a kill between stage and rename leaves only an orphaned temp,
// never a torn final.
const leaseStagePrefix = ".ax-lease-stage-"

// CreateLeaseInput carries the durable bootstrap inputs for an epoch-1
// create lease: the session must already exist (its record is the
// bootstrap fact), and the caller supplies the fencing token, the
// proposed owner, the initiator, and the diagnostic timestamp. Reason is
// fixed to create: creation never mints a takeover lease.
type CreateLeaseInput struct {
	LeaseID        string
	HolderHostID   string
	IssuedByHostID string
	CreatedAt      string
}

// SuccessorLeaseInput extends the bootstrap inputs with the two members
// a successor lease must carry: a reason and the validated materialized
// handoff base. Any of the four Section 5.3 reasons is admitted: the
// specification never declares a create-implies-epoch-1 direction, so
// where it is silent this writer stays permissive like the canonical
// owner, and the epoch-1-create couplings stay enforced by attestation.
type SuccessorLeaseInput struct {
	CreateLeaseInput
	Reason       string
	CheckpointID string
}

// LeaseExpectation is the CAS compare value: the record digest the caller
// believes is the current head.
type LeaseExpectation struct {
	RecordID string
}

// LeaseRef names one persisted lease: its session, record digest, fencing
// token, epoch, and holder.
type LeaseRef struct {
	SessionID    string
	RecordID     string
	LeaseID      string
	Epoch        uint64
	HolderHostID string
}

// LeaseSummary is the read projection of one stored lease.
type LeaseSummary struct {
	RecordID       string
	SessionID      string
	LeaseID        string
	Epoch          uint64
	HolderHostID   string
	Predecessor    string
	HasPredecessor bool
	Checkpoint     string
	HasCheckpoint  bool
	Reason         string
}

// FencingToken is a presented ownership proof: the epoch, lease ID, and
// holder the presenter claims.
type FencingToken struct {
	Epoch        uint64
	LeaseID      string
	HolderHostID string
}

// FencingGrant is the process-local authorization derived from a winning
// lease at ValidatedAt. Grants are machine-local transient state: they are
// never persisted and never replicated.
type FencingGrant struct {
	SessionID   string
	RecordID    string
	Token       FencingToken
	ValidatedAt time.Time
}

// FencingPolicy carries the configured lease refresh interval: the maximum
// age of a fencing grant before the holder must revalidate.
type FencingPolicy struct {
	RefreshInterval time.Duration
}

// CompareLeaseTuple orders two stored lease summaries by the Section 5.3
// rule: the greatest (epoch, lease_id) wins, where lease_id uses bytewise
// UUID order. Lease IDs are canonical lowercase UUIDv4, so string order is
// bytewise order. This restates sessstate.Compare for stored summaries
// (sessstate imports sessrepo, so the rule cannot be shared by import);
// the cross-package agreement test pins identical order.
func CompareLeaseTuple(a, b LeaseSummary) int {
	if a.Epoch != b.Epoch {
		if a.Epoch < b.Epoch {
			return -1
		}
		return 1
	}
	switch {
	case a.LeaseID < b.LeaseID:
		return -1
	case a.LeaseID > b.LeaseID:
		return 1
	}
	return 0
}

// leasesDir joins the per-session leases namespace. It is created on the
// first lease write; a session without it simply has no leases.
func (repository *Repository) leasesDir(sessionID string) string {
	return filepath.Join(repository.sessionDir(sessionID), "leases")
}

// checkLeaseIdentityInput validates the grammar shared by creation and
// succession: fencing-token, holder, issuer, and diagnostic timestamp.
// Attestation re-checks the token and host members at mint time, so each
// gate below is the last line of defense only for its refusal class, not
// for the bytes: weakening one still fails closed at mint with a plain
// (non-refusal) error, which is what kills that gate's narrowing mutant.
func checkLeaseIdentityInput(input CreateLeaseInput) error {
	if _, err := scalar.ParseUUIDv4(input.LeaseID); err != nil {
		return refuse(ErrInvalidLease, "lease fencing token %q is not a UUIDv4: %v", input.LeaseID, err)
	}
	if _, err := scalar.ParseUUIDv7(input.HolderHostID); err != nil {
		return refuse(ErrInvalidLease, "lease holder %q is not a UUIDv7: %v", input.HolderHostID, err)
	}
	if _, err := scalar.ParseUUIDv7(input.IssuedByHostID); err != nil {
		return refuse(ErrInvalidLease, "lease issuer %q is not a UUIDv7: %v", input.IssuedByHostID, err)
	}
	if _, err := scalar.ParseTimestamp(input.CreatedAt); err != nil {
		return refuse(ErrInvalidLease, "lease created_at %q is not a timestamp: %v", input.CreatedAt, err)
	}
	return nil
}

// checkSuccessorInput validates the two members only a successor carries.
// Absent and malformed checkpoints refuse identically: both fail digest
// grammar, and a successor without a handoff base is not expressible.
func checkSuccessorInput(input SuccessorLeaseInput) error {
	switch input.Reason {
	case "create", "graceful_takeover", "force_takeover", "recovery":
	default:
		return refuse(ErrInvalidLease, "successor lease reason %q is outside the Section 5.3 enum", input.Reason)
	}
	if _, err := scalar.ParseDigest(input.CheckpointID); err != nil {
		return refuse(ErrInvalidLease, "successor lease checkpoint %q is not a digest: %v", input.CheckpointID, err)
	}
	return nil
}

// CreateLease mints and persists the epoch-1 create lease for a session
// whose record already exists. A byte-identical retry returns the existing
// reference without duplicating the lease, so a retry after a crashed
// commit is safe; a differing second create is refused and the persisted
// lease survives.
func (repository *Repository) CreateLease(sessionID string, input CreateLeaseInput) (LeaseRef, error) {
	if err := checkLeaseIdentityInput(input); err != nil {
		return LeaseRef{}, err
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return LeaseRef{}, err
	}
	stored, err := repository.loadLeasesLocked(view.directory, view.record.sessionID)
	if err != nil {
		return LeaseRef{}, err
	}
	candidate, digest, err := mintLeaseRecord(view.record.sessionID, 1, input.LeaseID, input.HolderHostID, input.IssuedByHostID, "", false, "create", "", false, input.CreatedAt)
	if err != nil {
		return LeaseRef{}, err
	}
	if identical, ref := findIdenticalLease(stored, digest); identical {
		return ref, nil
	}
	if len(stored) > 0 {
		return LeaseRef{}, refuse(ErrLeaseExists, "session %s already holds lease %s at epoch %d", sessionID, stored[len(stored)-1].RecordID, stored[len(stored)-1].Epoch)
	}
	if repository.BeforeWrite != nil {
		if err := repository.BeforeWrite(); err != nil {
			return LeaseRef{}, err
		}
	}
	if err := repository.installLeaseBlobLocked(view.directory, digest, candidate); err != nil {
		return LeaseRef{}, err
	}
	if repository.AfterCommit != nil {
		if err := repository.AfterCommit(); err != nil {
			return LeaseRef{}, err
		}
	}
	return LeaseRef{SessionID: view.record.sessionID, RecordID: digest, LeaseID: input.LeaseID, Epoch: 1, HolderHostID: input.HolderHostID}, nil
}

// CompareAndSwapLease persists the successor of the current head when the
// expectation names it. The new lease always extends the head: epoch is
// head epoch plus one (never derived from input, so it can neither skip
// nor decrease) and the predecessor is the head fencing token. An
// expectation naming a superseded lease is stale; one naming nothing
// known — malformed, empty, or foreign — is a conflict. A byte-identical
// retry of an already persisted successor returns its existing reference,
// so a retry after a crashed commit is safe without re-reading the head.
func (repository *Repository) CompareAndSwapLease(sessionID string, expected LeaseExpectation, input SuccessorLeaseInput) (LeaseRef, error) {
	if err := checkLeaseIdentityInput(input.CreateLeaseInput); err != nil {
		return LeaseRef{}, err
	}
	if err := checkSuccessorInput(input); err != nil {
		return LeaseRef{}, err
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return LeaseRef{}, err
	}
	stored, err := repository.loadLeasesLocked(view.directory, view.record.sessionID)
	if err != nil {
		return LeaseRef{}, err
	}
	if len(stored) == 0 {
		return LeaseRef{}, refuse(ErrUnknownLease, "no lease is persisted for session %s; create the epoch-1 lease first", sessionID)
	}
	head := stored[len(stored)-1]
	basis, known := findExpectedLease(stored, expected.RecordID)
	if !known {
		return LeaseRef{}, refuse(ErrLeaseConflict, "expected lease %q is not the current head %s for session %s", expected.RecordID, head.RecordID, sessionID)
	}
	if basis.RecordID != head.RecordID {
		// A superseded expectation is either a crashed-commit retry —
		// the input already applied as this basis's successor — or a
		// genuinely stale caller. Re-minting against the basis (not the
		// head) reproduces the retry's original bytes: a digest hit
		// answers the retry, a miss refuses stale. The stale class wins
		// over token reuse here: the caller re-reads the head first.
		_, replayed, err := mintLeaseRecord(view.record.sessionID, basis.Epoch+1, input.LeaseID, input.HolderHostID, input.IssuedByHostID, basis.LeaseID, true, input.Reason, input.CheckpointID, true, input.CreatedAt)
		if err != nil {
			return LeaseRef{}, err
		}
		if identical, ref := findIdenticalLease(stored, replayed); identical {
			return ref, nil
		}
		return LeaseRef{}, refuse(ErrStaleLeaseEpoch, "expected lease %s at epoch %d is superseded by head %s at epoch %d", expected.RecordID, basis.Epoch, head.RecordID, head.Epoch)
	}
	candidate, digest, err := mintLeaseRecord(view.record.sessionID, head.Epoch+1, input.LeaseID, input.HolderHostID, input.IssuedByHostID, head.LeaseID, true, input.Reason, input.CheckpointID, true, input.CreatedAt)
	if err != nil {
		return LeaseRef{}, err
	}
	for _, known := range stored {
		if known.LeaseID == input.LeaseID {
			return LeaseRef{}, refuse(ErrInvalidLease, "fencing token %s is already bound to lease %s for session %s", input.LeaseID, known.RecordID, sessionID)
		}
	}
	if repository.BeforeWrite != nil {
		if err := repository.BeforeWrite(); err != nil {
			return LeaseRef{}, err
		}
	}
	if err := repository.installLeaseBlobLocked(view.directory, digest, candidate); err != nil {
		return LeaseRef{}, err
	}
	if repository.AfterCommit != nil {
		if err := repository.AfterCommit(); err != nil {
			return LeaseRef{}, err
		}
	}
	return LeaseRef{SessionID: view.record.sessionID, RecordID: digest, LeaseID: input.LeaseID, Epoch: head.Epoch + 1, HolderHostID: input.HolderHostID}, nil
}

// GetLease returns the byte-identical stored Lease Record for its digest,
// re-verified through the canonical owner on every read.
func (repository *Repository) GetLease(sessionID, recordID string) ([]byte, error) {
	if _, err := scalar.ParseDigest(recordID); err != nil {
		return nil, refuse(ErrInvalidLease, "malformed lease record id %q", recordID)
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return nil, err
	}
	blob, err := os.ReadFile(filepath.Join(view.directory, "leases", blobFileName(recordID)))
	if err != nil {
		return nil, refuse(ErrUnknownLease, "unknown lease %q in session %s", recordID, sessionID)
	}
	if err := verifyLeaseBlob(view.record.sessionID, recordID, blob); err != nil {
		return nil, refuse(ErrChainCorrupt, "stored lease %s for session %s: %v", blobFileName(recordID), sessionID, err)
	}
	return blob, nil
}

// ListLeases returns every stored lease in tuple order (ascending epoch,
// then bytewise lease ID), each re-verified through the canonical owner.
// A session with no leases yields an empty slice, never an error.
func (repository *Repository) ListLeases(sessionID string) ([]LeaseSummary, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return nil, err
	}
	stored, err := repository.loadLeasesLocked(view.directory, view.record.sessionID)
	if err != nil {
		return nil, err
	}
	return stored, nil
}

// WinningLease returns the current head: the greatest (epoch, lease_id)
// tuple. The head is derived from the blobs on every read, never cached,
// so a second process installing a greater lease is visible immediately.
func (repository *Repository) WinningLease(sessionID string) (LeaseSummary, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return LeaseSummary{}, err
	}
	stored, err := repository.loadLeasesLocked(view.directory, view.record.sessionID)
	if err != nil {
		return LeaseSummary{}, err
	}
	if len(stored) == 0 {
		return LeaseSummary{}, refuse(ErrUnknownLease, "no winning lease is established for session %s", sessionID)
	}
	return stored[len(stored)-1], nil
}

// VerifyFencingToken revalidates a presented fencing token against the
// current head: the renewal step an owner repeats before accepting input,
// starting a provider turn, publishing a checkpoint, pushing records, and
// resuming after an interruption past the refresh interval (Section 5.3).
// A token below the winning epoch is stale, one beyond it names no known
// lease, a same-epoch token under another fencing ID loses the tie, and a
// token from another holder is not the owner. A nil error renews the
// authorization the token carries; it mints no new lease and mutates no
// durable state.
func (repository *Repository) VerifyFencingToken(sessionID string, token FencingToken) error {
	if token.Epoch == 0 {
		return refuse(ErrInvalidLease, "fencing token carries epoch 0, want an epoch at or above 1")
	}
	if _, err := scalar.ParseUUIDv4(token.LeaseID); err != nil {
		return refuse(ErrInvalidLease, "fencing token lease %q is not a UUIDv4: %v", token.LeaseID, err)
	}
	if _, err := scalar.ParseUUIDv7(token.HolderHostID); err != nil {
		return refuse(ErrInvalidLease, "fencing token holder %q is not a UUIDv7: %v", token.HolderHostID, err)
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	stored, err := repository.loadLeasesLocked(view.directory, view.record.sessionID)
	if err != nil {
		return err
	}
	winner := LeaseSummary{}
	if len(stored) > 0 {
		winner = stored[len(stored)-1]
	}
	if token.Epoch < winner.Epoch {
		return refuse(ErrStaleLeaseEpoch, "fencing token epoch %d precedes winning epoch %d for session %s", token.Epoch, winner.Epoch, sessionID)
	}
	if token.Epoch > winner.Epoch {
		return refuse(ErrUnknownLease, "fencing token epoch %d names no known lease for session %s", token.Epoch, sessionID)
	}
	if token.LeaseID != winner.LeaseID {
		return refuse(ErrLeaseConflict, "fencing token lease %s loses to winning lease %s at epoch %d", token.LeaseID, winner.LeaseID, winner.Epoch)
	}
	if token.HolderHostID != winner.HolderHostID {
		return refuse(ErrLeaseHolderMismatch, "fencing token holder %s is not winning holder %s for session %s", token.HolderHostID, winner.HolderHostID, sessionID)
	}
	return nil
}

// CheckFencingExpiry enforces the operational grant-expiry policy: a grant
// older than the refresh interval lapses and its holder must revalidate
// through VerifyFencingToken. The lease named by a lapsed grant stays
// authoritative — expiry retires the cached authorization, never the
// ownership — and a grant validated exactly at the interval boundary is
// still current. A non-positive interval is not a policy: it refuses.
func CheckFencingExpiry(grant FencingGrant, policy FencingPolicy, now time.Time) error {
	if policy.RefreshInterval <= 0 {
		return refuse(ErrInvalidLease, "fencing policy refresh interval must be positive")
	}
	if now.Sub(grant.ValidatedAt) > policy.RefreshInterval {
		return refuse(ErrFencingExpired, "fencing grant for session %s lapsed; revalidate the fencing token", grant.SessionID)
	}
	return nil
}

// loadLeasesLocked reads every stored lease for one session directory in
// tuple order. The blobs are truth: there is no lease index to skew, so a
// crash can only leave a blob stored or not. Staged temps are ignored; any
// other entry that fails to read, decode, attest, bind to the session, or
// agree with its digest name is torn store, refused through the one funnel
// site with the cause carried in the message. Callers hold the mutex.
func (repository *Repository) loadLeasesLocked(directory, sessionID string) ([]LeaseSummary, error) {
	stored, err := scanLeaseBlobs(directory, sessionID)
	if err != nil {
		return nil, refuse(ErrChainCorrupt, "stored leases for session %s: %v", sessionID, err)
	}
	return stored, nil
}

// scanLeaseBlobs collects and orders the verified lease summaries. Every
// failure is a plain error; loadLeasesLocked attributes it through its one
// refusal site.
func scanLeaseBlobs(directory, sessionID string) ([]LeaseSummary, error) {
	entries, err := os.ReadDir(filepath.Join(directory, "leases"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read leases namespace: %w", err)
	}
	var stored []LeaseSummary
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		expected, err := leaseDigestForName(name)
		if err != nil {
			return nil, err
		}
		blob, err := os.ReadFile(filepath.Join(directory, "leases", name))
		if err != nil {
			return nil, fmt.Errorf("read lease %s: %w", name, err)
		}
		summary, err := decodeStoredLease(sessionID, blob)
		if err != nil {
			return nil, fmt.Errorf("lease %s: %v", name, err)
		}
		if summary.RecordID != expected {
			return nil, fmt.Errorf("lease %s attests %s", name, summary.RecordID)
		}
		stored = append(stored, summary)
	}
	sort.Slice(stored, func(i, j int) bool { return CompareLeaseTuple(stored[i], stored[j]) < 0 })
	return stored, nil
}

// verifyLeaseBlob decodes one blob and binds it to its expected session
// and digest name. Failures are plain errors; the caller attributes them
// through its own refusal site.
func verifyLeaseBlob(sessionID, expected string, blob []byte) error {
	summary, err := decodeStoredLease(sessionID, blob)
	if err != nil {
		return err
	}
	if summary.RecordID != expected {
		return fmt.Errorf("blob attests %s, want %s", summary.RecordID, expected)
	}
	return nil
}

// leaseDigestForName renders a leases-namespace file name back to the
// digest it must hold. Anything outside hex.json is torn store, never a
// skipped entry: only staged temps are ignorable.
func leaseDigestForName(name string) (string, error) {
	trimmed := strings.TrimSuffix(name, ".json")
	digest, err := scalar.ParseDigest("sha256:" + trimmed)
	if err != nil {
		return "", fmt.Errorf("lease file %q is not a digest path: %v", name, err)
	}
	return digest.String(), nil
}

// installLeaseBlobLocked installs one minted lease blob atomically: staged
// bytes fsync before the rename, the directory fsyncs after. A second
// install of identical bytes is reused; disagreeing bytes at a digest path
// are torn store. AfterLeaseStage fires after the staged fsync and before
// the rename — the kill seam the crash tests drive — and is nil in normal
// operation. Callers hold the repository mutex.
func (repository *Repository) installLeaseBlobLocked(directory, recordID string, data []byte) error {
	leases := filepath.Join(directory, "leases")
	if err := os.MkdirAll(leases, 0o700); err != nil {
		return fmt.Errorf("create leases namespace: %w", err)
	}
	path := filepath.Join(leases, blobFileName(recordID))
	if existing, err := os.ReadFile(path); err == nil {
		if leaseBytesEqual(existing, data) {
			return nil
		}
		return refuse(ErrChainCorrupt, "digest path holds disagreeing bytes for lease")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read installed lease blob: %w", err)
	}
	staged, err := os.CreateTemp(leases, leaseStagePrefix)
	if err != nil {
		return fmt.Errorf("stage lease blob: %w", err)
	}
	stagedPath := staged.Name()
	finished := false
	defer func() {
		if !finished {
			_ = staged.Close()
			_ = os.Remove(stagedPath)
		}
	}()
	if err := staged.Chmod(0o600); err != nil {
		return fmt.Errorf("mode staged lease blob: %w", err)
	}
	if _, err := staged.Write(data); err != nil {
		return fmt.Errorf("write staged lease blob: %w", err)
	}
	if err := staged.Sync(); err != nil {
		return fmt.Errorf("fsync staged lease blob: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close staged lease blob: %w", err)
	}
	if repository.AfterLeaseStage != nil {
		if err := repository.AfterLeaseStage(); err != nil {
			return err
		}
	}
	finished = true
	if err := os.Rename(stagedPath, path); err != nil {
		return fmt.Errorf("install lease blob: %w", err)
	}
	return syncDirectory(leases)
}

// findExpectedLease resolves a CAS expectation to the stored lease it names.
// An empty or unknown digest resolves nothing: only a persisted record_id
// can be a compare basis.
func findExpectedLease(stored []LeaseSummary, recordID string) (LeaseSummary, bool) {
	for _, known := range stored {
		if known.RecordID == recordID {
			return known, true
		}
	}
	return LeaseSummary{}, false
}

// findIdenticalLease reports the stored lease whose digest equals the minted
// candidate's, so a byte-identical retry returns its reference instead of
// minting a duplicate. The digest is the canonical content identity, so
// digest equality is full-content equality under the same collision
// resistance the content-addressed layout already assumes; the load path
// re-verified every stored blob before this comparison runs.
func findIdenticalLease(stored []LeaseSummary, digest string) (bool, LeaseRef) {
	for _, known := range stored {
		if known.RecordID == digest {
			return true, LeaseRef{SessionID: known.SessionID, RecordID: known.RecordID, LeaseID: known.LeaseID, Epoch: known.Epoch, HolderHostID: known.HolderHostID}
		}
	}
	return false, LeaseRef{}
}

// leaseBytesEqual is the one content-equality site of the lease store,
// registered in the census ledger with its same-length vector.
func leaseBytesEqual(left, right []byte) bool {
	return bytes.Equal(left, right)
}

// mintLeaseRecord builds one canonical Lease Record and returns its bytes
// with the attested record_id. Identity is computed by the canonicaljson
// owner over the placeholder, then confirmed by Verify after the digest is
// bound, so minted bytes always attest. Failures are plain operational
// errors, never refusals: entry validation already admitted the inputs,
// so a mint failure is an internal inconsistency, not a rejected input.
func mintLeaseRecord(sessionID string, epoch uint64, leaseID, holder, issuer, predecessor string, hasPredecessor bool, reason, checkpoint string, hasCheckpoint bool, createdAt string) ([]byte, string, error) {
	var predecessorValue any
	if hasPredecessor {
		predecessorValue = predecessor
	}
	var checkpointValue any
	if hasCheckpoint {
		checkpointValue = checkpoint
	}
	object := map[string]any{
		"schema":               leaseRecordSchema,
		"schema_version":       "1.0.0",
		"record_id":            leaseIdentityPlaceholder,
		"subject_id":           sessionID,
		"session_id":           sessionID,
		"lease_id":             leaseID,
		"epoch":                epoch,
		"holder_host_id":       holder,
		"predecessor_lease_id": predecessorValue,
		"reason":               reason,
		"checkpoint_id":        checkpointValue,
		"issued_by_host_id":    issuer,
		"created_by_host_id":   issuer,
		"created_at":           createdAt,
		"extensions":           map[string]any{},
	}
	staged, err := json.Marshal(object)
	if err != nil {
		return nil, "", fmt.Errorf("encode lease record: %w", err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		return nil, "", fmt.Errorf("identify lease record: %w", err)
	}
	object["record_id"] = digest.String()
	raw, err := json.Marshal(object)
	if err != nil {
		return nil, "", fmt.Errorf("encode identified lease record: %w", err)
	}
	if _, _, err := AttestLeaseRecord(raw); err != nil {
		return nil, "", fmt.Errorf("attest minted lease record: %w", err)
	}
	return raw, digest.String(), nil
}

// decodeStoredLease validates one stored Lease Record through the
// canonicaljson owner and extracts the bound members. Closed-shape,
// coupling, and identity failures are plain errors for the caller to
// attribute; member extraction re-checks grammar through the environ and
// scalar owners, matching the query-layer admission without forking it.
func decodeStoredLease(sessionID string, raw []byte) (LeaseSummary, error) {
	var summary LeaseSummary
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return summary, fmt.Errorf("decode lease record frame: %v", fault)
	}
	schema, err := stringMember(members, "schema")
	if err != nil || schema != leaseRecordSchema {
		return summary, fmt.Errorf("lease record schema %q, want %q", schema, leaseRecordSchema)
	}
	digest, field, err := AttestLeaseRecord(raw)
	if err != nil {
		return summary, fmt.Errorf("verify lease record identity: %v", err)
	}
	if field != canonicaljson.SelfRecordID {
		return summary, fmt.Errorf("lease record identity field %q, want record_id", string(field))
	}
	recordSession, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return summary, fmt.Errorf("lease record carries no valid session_id member")
	}
	if recordSession.String() != sessionID {
		return summary, fmt.Errorf("lease record session %s does not match stored session %s", recordSession.String(), sessionID)
	}
	leaseID, err := stringMember(members, "lease_id")
	if err != nil {
		return summary, fmt.Errorf("lease record carries no lease_id member: %v", err)
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return summary, fmt.Errorf("lease record lease_id: %v", err)
	}
	epoch, ok := environ.CheckUint53Bounds(members["epoch"], 1, uint53Max)
	if !ok {
		return summary, fmt.Errorf("lease record carries no epoch at or above 1")
	}
	holder, ok := environ.CheckUUIDv7(members["holder_host_id"])
	if !ok {
		return summary, fmt.Errorf("lease record carries no valid holder_host_id member")
	}
	predecessor, hasPredecessor, err := nullableUUIDv4Member(members, "predecessor_lease_id")
	if err != nil {
		return summary, fmt.Errorf("lease record predecessor_lease_id: %v", err)
	}
	checkpoint, hasCheckpoint, err := nullableDigestMember(members, "checkpoint_id")
	if err != nil {
		return summary, fmt.Errorf("lease record checkpoint_id: %v", err)
	}
	reason, err := stringMember(members, "reason")
	if err != nil {
		return summary, fmt.Errorf("lease record carries no reason member: %v", err)
	}
	summary = LeaseSummary{RecordID: digest.String(), SessionID: recordSession.String(), LeaseID: leaseID, Epoch: epoch, HolderHostID: holder.String(), Predecessor: predecessor, HasPredecessor: hasPredecessor, Checkpoint: checkpoint, HasCheckpoint: hasCheckpoint, Reason: reason}
	return summary, nil
}

// nullableUUIDv4Member reads a member that is null or a UUIDv4 string. A
// nil pointer binds JSON null without any text comparison.
func nullableUUIDv4Member(members map[string]json.RawMessage, name string) (string, bool, error) {
	raw, ok := members[name]
	if !ok {
		return "", false, fmt.Errorf("member %q is absent", name)
	}
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false, fmt.Errorf("member %q is neither a UUID nor null", name)
	}
	if value == nil {
		return "", false, nil
	}
	if _, err := scalar.ParseUUIDv4(*value); err != nil {
		return "", false, err
	}
	return *value, true, nil
}

// nullableDigestMember reads a member that is null or a digest string. A
// nil pointer binds JSON null without any text comparison.
func nullableDigestMember(members map[string]json.RawMessage, name string) (string, bool, error) {
	raw, ok := members[name]
	if !ok {
		return "", false, fmt.Errorf("member %q is absent", name)
	}
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false, fmt.Errorf("member %q is neither a digest nor null", name)
	}
	if value == nil {
		return "", false, nil
	}
	digest, err := scalar.ParseDigest(*value)
	if err != nil {
		return "", false, err
	}
	return digest.String(), true, nil
}
