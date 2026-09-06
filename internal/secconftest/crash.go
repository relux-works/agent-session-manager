package secconftest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Outcome is the Section 13.13 crash/restart vocabulary every crash
// point classifies into. AC-CRASH-001 requires exactly one of these per
// injection with the section's evidence record; no run may produce
// duplicate live owners, treat an unfenced external continuation as
// safe, or substitute a fresh identity for the persisted one.
type Outcome string

const (
	// OutcomeSafeRetry means the operation never mutated durable state
	// and the identical retry is safe.
	OutcomeSafeRetry Outcome = "safe_retry"
	// OutcomeExplicitRollback means durable state mutated and the caller
	// must roll back before any retry.
	OutcomeExplicitRollback Outcome = "explicit_rollback"
	// OutcomeRecoverableParked means the operation stopped in a named
	// parked state from which an operator or a later run recovers; it
	// must never silently resume as a fresh identity.
	OutcomeRecoverableParked Outcome = "recoverable_parked_state"
)

// Crash points name the phase boundaries the injector can fire on. The
// families follow the AC-CRASH-001 class list (CR-LAUNCH-*, CR-SYNC-*,
// CR-MAT-*, CR-GRACE-*, CR-FORCE-*, CR-FORK-*, CR-STOP-*, CR-RESUME-*,
// CR-RESTORE-*, CR-CLONE-01..16); the suffix selects the boundary
// inside the modeled operation.
const (
	PointPrepareEnter  = "CR-MAT-prepare-enter"
	PointPrepareCommit = "CR-MAT-prepare-commit"
	PointCommitEnter   = "CR-MAT-commit-enter"
	PointCommitApply   = "CR-MAT-commit-apply"
	PointRollbackEnter = "CR-MAT-rollback-enter"
)

// ErrInjected reports a fired crash point. It carries the point and the
// classified outcome so the caller cannot observe the fault without
// observing what it requires.
var ErrInjected = errors.New("secconftest injected crash")

// Fault is the fired crash-point evidence.
type Fault struct {
	point   string
	outcome Outcome
}

func (fault *Fault) Error() string {
	return fmt.Sprintf("%v: %s requires %s", ErrInjected, fault.point, fault.outcome)
}

// Unwrap matches ErrInjected so callers match the fault kind without
// parsing human text.
func (fault *Fault) Unwrap() error {
	return ErrInjected
}

// Point names the fired crash point.
func (fault *Fault) Point() string { return fault.point }

// Outcome classifies the fired crash point into the Section 13.13
// vocabulary.
func (fault *Fault) Outcome() Outcome { return fault.outcome }

// Classify maps a crash point to its required outcome. Points before
// any durable mutation are safe_retry; points after the journal takes
// effect require explicit_rollback; points past the commit marker park
// for recovery. Weakening any row to a different outcome must fail the
// table test that drives it.
func Classify(point string) (Outcome, error) {
	switch point {
	case PointPrepareEnter:
		return OutcomeSafeRetry, nil
	case PointPrepareCommit, PointCommitEnter, PointRollbackEnter:
		return OutcomeExplicitRollback, nil
	case PointCommitApply:
		return OutcomeRecoverableParked, nil
	default:
		return "", fmt.Errorf("secconftest: unknown crash point %q", point)
	}
}

// Injector arms named crash points. The zero value is usable and fires
// nothing. MaybeFail is the only firing entry: production-shaped code
// under test calls it at each phase boundary.
type Injector struct {
	mutex sync.Mutex
	armed map[string]bool
}

// Arm fires point on its next MaybeFail call.
func (injector *Injector) Arm(point string) {
	injector.mutex.Lock()
	defer injector.mutex.Unlock()
	if injector.armed == nil {
		injector.armed = map[string]bool{}
	}
	injector.armed[point] = true
}

// Disarm clears every armed point.
func (injector *Injector) Disarm() {
	injector.mutex.Lock()
	defer injector.mutex.Unlock()
	injector.armed = map[string]bool{}
}

// Armed reports whether point is currently armed.
func (injector *Injector) Armed(point string) bool {
	injector.mutex.Lock()
	defer injector.mutex.Unlock()
	return injector.armed[point]
}

// RemainingArmed lists the points still armed, sorted. A test that arms
// a point and drives the operation must find this empty afterwards: an
// armed point that never fires is otherwise noticed only incidentally,
// and a new crash site added without an arm is outside this model (see
// the package bounds). Weakening this to a constant empty slice must
// fail the consumption test that drives it.
func (injector *Injector) RemainingArmed() []string {
	injector.mutex.Lock()
	defer injector.mutex.Unlock()
	remaining := make([]string, 0, len(injector.armed))
	for point := range injector.armed {
		remaining = append(remaining, point)
	}
	sort.Strings(remaining)
	return remaining
}

// MaybeFail returns a *Fault when point is armed and consumes the arm,
// so each armed point fires exactly once. It returns nil when
// disarmed. An unknown point name is a caller error, never silent.
func (injector *Injector) MaybeFail(point string) error {
	outcome, err := Classify(point)
	if err != nil {
		return err
	}
	injector.mutex.Lock()
	fire := injector.armed[point]
	if fire {
		delete(injector.armed, point)
	}
	injector.mutex.Unlock()
	if !fire {
		return nil
	}
	return &Fault{point: point, outcome: outcome}
}

// ErrIdempotencyMismatch reports a lost-response retry that changed the
// operation body: AC-MAT-004 requires the byte-identical receipt for
// the identical retry and a failure that creates no second root for a
// changed one.
var ErrIdempotencyMismatch = errors.New("secconftest idempotency_mismatch")

// Receipt is the durable proof of one transacted operation.
type Receipt struct {
	OperationID string
	BodyDigest  string
	Outcome     Outcome
	Bytes       []byte
}

// Transactor is a three-phase durable-commit model (prepare, commit,
// rollback) over a directory, driven through MaybeFail boundaries. It
// exists to prove the injector, the outcome vocabulary, and the
// idempotency rule against real filesystem state; it is not a product
// journal and no product path may import it for recovery.
type Transactor struct {
	injector *Injector
	root     string
	mutex    sync.Mutex
	receipts map[string]Receipt
	// finalized marks operations whose Commit finished the durable
	// write, cleanly or via a commit-apply fault. The operation is
	// then terminal and Rollback must refuse it (AC-CLONE-005:
	// rollback is forbidden after Provider commit).
	finalized map[string]bool
}

// NewTransactor binds a Transactor to root, which must exist.
func NewTransactor(injector *Injector, root string) *Transactor {
	return &Transactor{injector: injector, root: root, receipts: map[string]Receipt{}, finalized: map[string]bool{}}
}

func digestBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// ErrPreparePastFinalize reports a Prepare refused after the operation
// finalized (clean or crashed Commit): finalization is terminal and a
// new prepare phase cannot resurrect it. The idempotent Prepare retry
// rule (AC-MAT-004) covers retries before Commit only; after Commit the
// operation already has a durable outcome, and returning the old
// explicit_rollback receipt would misdescribe a finalized operation.
var ErrPreparePastFinalize = errors.New("secconftest prepare forbidden after finalize")

// ErrCommitPastFinalize reports a Commit refused after the operation
// already finalized: the second commit has no staged bytes to promote
// and must name the terminal state, not inherit a filesystem ENOENT.
var ErrCommitPastFinalize = errors.New("secconftest commit forbidden after finalize")

// Prepare stages body under operationID and fsyncs the journal entry.
// A retry with the same operation ID and byte-identical body returns
// the identical receipt without staging twice. A retry with the same
// ID and a different body fails with ErrIdempotencyMismatch and
// stages nothing new. Prepare after the operation finalized (clean or
// crashed Commit) is refused with ErrPreparePastFinalize: terminal
// means terminal.
func (transactor *Transactor) Prepare(operationID string, body []byte) (Receipt, error) {
	if operationID == "" {
		return Receipt{}, errors.New("secconftest: empty operation ID")
	}
	if err := transactor.injector.MaybeFail(PointPrepareEnter); err != nil {
		return Receipt{}, err
	}
	digest := digestBody(body)
	transactor.mutex.Lock()
	if transactor.finalized[operationID] {
		transactor.mutex.Unlock()
		return Receipt{}, fmt.Errorf("%w: operation %q", ErrPreparePastFinalize, operationID)
	}
	if prior, ok := transactor.receipts[operationID]; ok {
		transactor.mutex.Unlock()
		if prior.BodyDigest != digest {
			return Receipt{}, fmt.Errorf("%w: operation %q body changed", ErrIdempotencyMismatch, operationID)
		}
		return prior, nil
	}
	transactor.mutex.Unlock()
	staged := filepath.Join(transactor.root, operationID+".staged")
	if err := os.WriteFile(staged, body, 0o600); err != nil {
		return Receipt{}, err
	}
	if err := transactor.injector.MaybeFail(PointPrepareCommit); err != nil {
		_ = os.Remove(staged)
		return Receipt{}, err
	}
	receipt := Receipt{OperationID: operationID, BodyDigest: digest, Outcome: OutcomeSafeRetry, Bytes: append([]byte(nil), body...)}
	transactor.mutex.Lock()
	transactor.receipts[operationID] = receipt
	transactor.mutex.Unlock()
	return receipt, nil
}

// Commit promotes the staged body to committed bytes. It fails armed
// crash points before (commit-enter: rollback required) and after
// (commit-apply: parked) the durable rename.
//
// A commit-apply fault fires AFTER the durable write, so the operation
// is already committed in every observable way: CommittedBytes returns
// the body and the package's own parked-bytes assertion requires it to
// survive. The fault path therefore finalizes the operation with a
// recoverable_parked_state receipt and Rollback afterwards is
// forbidden (AC-CLONE-005: rollback is forbidden after Provider
// commit; all bundle evidence survives). The alternative — letting
// Rollback delete the parked bytes — would walk through both halves
// of that clause at once, which is the one unacceptable answer; the
// leaf states this rule here, in doc.go, and in the README.
//
// Terminal neighbours name their state instead of inheriting ENOENT:
// a second Commit is ErrCommitPastFinalize, and a Commit whose staged
// bytes are gone without finalization names the missing staged bytes.
func (transactor *Transactor) Commit(operationID string) (Receipt, error) {
	if err := transactor.injector.MaybeFail(PointCommitEnter); err != nil {
		return Receipt{}, err
	}
	transactor.mutex.Lock()
	receipt, ok := transactor.receipts[operationID]
	finalized := transactor.finalized[operationID]
	transactor.mutex.Unlock()
	if !ok {
		return Receipt{}, fmt.Errorf("secconftest: commit without prepare for operation %q", operationID)
	}
	if finalized {
		return Receipt{}, fmt.Errorf("%w: operation %q", ErrCommitPastFinalize, operationID)
	}
	staged := filepath.Join(transactor.root, operationID+".staged")
	committed := filepath.Join(transactor.root, operationID+".committed")
	contents, err := os.ReadFile(staged)
	if err != nil {
		if os.IsNotExist(err) {
			return Receipt{}, fmt.Errorf("secconftest: commit without staged bytes for operation %q: %w", operationID, err)
		}
		return Receipt{}, err
	}
	if err := os.WriteFile(committed, contents, 0o600); err != nil {
		return Receipt{}, err
	}
	_ = os.Remove(staged)
	if err := transactor.injector.MaybeFail(PointCommitApply); err != nil {
		receipt.Outcome = OutcomeRecoverableParked
		transactor.mutex.Lock()
		transactor.receipts[operationID] = receipt
		transactor.finalized[operationID] = true
		transactor.mutex.Unlock()
		return Receipt{}, err
	}
	receipt.Outcome = OutcomeExplicitRollback
	transactor.mutex.Lock()
	transactor.receipts[operationID] = receipt
	transactor.finalized[operationID] = true
	transactor.mutex.Unlock()
	return receipt, nil
}

// ErrRollbackPastFinalize reports a rollback refused after finalization
// (clean or crashed Commit): the operation is terminal and rollback is
// forbidden (AC-CLONE-005 shapes this rule; the model enforces it for
// both Commit outcomes because the crashed Commit already performed
// the durable write).
var ErrRollbackPastFinalize = errors.New("secconftest rollback forbidden after commit")

// Rollback removes staged bytes for operationID before finalization
// and forgets the operation, so a second Rollback names the missing
// prepare instead of succeeding silently and a later Commit names it
// too; a fresh Prepare for the same ID restages from scratch (the
// recovery retry). It refuses to run past finalization: once Commit
// returns cleanly — or faults at commit-apply after the durable
// write — the operation is terminal, rollback is forbidden, and the
// committed bytes must survive the refused call untouched. Deleting
// the finalize gate must fail the tests that commit (clean or
// crashed) and then roll back.
func (transactor *Transactor) Rollback(operationID string) error {
	if err := transactor.injector.MaybeFail(PointRollbackEnter); err != nil {
		return err
	}
	transactor.mutex.Lock()
	_, ok := transactor.receipts[operationID]
	finalized := transactor.finalized[operationID]
	transactor.mutex.Unlock()
	if !ok {
		return fmt.Errorf("secconftest: rollback without prepare for operation %q", operationID)
	}
	if finalized {
		return fmt.Errorf("%w: operation %q", ErrRollbackPastFinalize, operationID)
	}
	_ = os.Remove(filepath.Join(transactor.root, operationID+".staged"))
	_ = os.Remove(filepath.Join(transactor.root, operationID+".committed"))
	transactor.mutex.Lock()
	delete(transactor.receipts, operationID)
	transactor.mutex.Unlock()
	return nil
}

// CommittedBytes reads back the committed bytes for operationID.
func (transactor *Transactor) CommittedBytes(operationID string) ([]byte, error) {
	return os.ReadFile(filepath.Join(transactor.root, operationID+".committed"))
}
