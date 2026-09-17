package matjournal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Recovery outcomes (SPEC Section 13.13): exactly one classifies every
// crash boundary. They are mutually exclusive and collectively
// exhaustive; missing, stale, contradictory, unreachable, or ambiguous
// evidence selects recoverable_parked_state, and no fourth outcome
// exists.
type Outcome string

const (
	// OutcomeSafeRetry means the same logical operation can continue
	// or return its recorded result with every caller-stable ID and a
	// byte-identical immutable input, after reconciling any uncertain
	// external effect and allocating no second process, manager,
	// handle, lease, authority, or transaction root.
	OutcomeSafeRetry Outcome = "safe_retry"
	// OutcomeExplicitRollback means recovery executed an allowed
	// abort, rollback, or proven closure before owner activation,
	// restoring every affected predecessor or removing only inert
	// fresh staging, with the terminal result durable and visible.
	OutcomeExplicitRollback Outcome = "explicit_rollback"
	// OutcomeRecoverableParked means recovery cannot yet prove safe
	// replay or legal rollback, so it fails closed with the lease,
	// IDs, checkpoint/native identity, phase, and blocking reason
	// recoverable and input blocked.
	OutcomeRecoverableParked Outcome = "recoverable_parked_state"
)

// Crash boundaries for every materialization (SPEC Section 13.13
// CR-MAT-01..08): before journal creation; after journal/prepare
// receipt; after transfer; after validation; after provider prepare or
// bridge import; after bridge open or host commit enters prepared;
// after owner activation may have occurred; after finalize/cleanup may
// have occurred but before its result is durable.
const (
	BoundaryBeforeCreate    = "CR-MAT-01"
	BoundaryAfterPrepare    = "CR-MAT-02"
	BoundaryAfterTransfer   = "CR-MAT-03"
	BoundaryAfterValidation = "CR-MAT-04"
	BoundaryAfterPrepareOp  = "CR-MAT-05"
	BoundaryAfterOpen       = "CR-MAT-06"
	BoundaryAfterActivation = "CR-MAT-07"
	BoundaryAfterFinalize   = "CR-MAT-08"
)

// validBoundary reports whether the boundary names the registry.
func validBoundary(boundary string) bool {
	switch boundary {
	case BoundaryBeforeCreate, BoundaryAfterPrepare, BoundaryAfterTransfer,
		BoundaryAfterValidation, BoundaryAfterPrepareOp, BoundaryAfterOpen,
		BoundaryAfterActivation, BoundaryAfterFinalize:
		return true
	default:
		return false
	}
}

// Recovery paths the CR-MAT row applies to independently.
const (
	PathGracefulTakeover = "graceful_takeover"
	PathForceTakeover    = "force_takeover"
	PathPassiveSync      = "passive_sync"
	PathOwnerResume      = "owner_resume"
	PathFork             = "fork"
)

func validPath(path string) bool {
	switch path {
	case PathGracefulTakeover, PathForceTakeover, PathPassiveSync, PathOwnerResume, PathFork:
		return true
	default:
		return false
	}
}

// Host conditions (SPEC Section 13.12): destination disk full and
// atomic rename blocked. Either parks with its remediation; neither
// overwrites in place.
const (
	HostOK            = "none"
	HostDiskFull      = "disk_full"
	HostRenameBlocked = "rename_blocked"
)

// Marker probe states: absent, a valid marker matching the journal, a
// mismatching marker, or an invalid marker.
const (
	MarkerAbsent     = "absent"
	MarkerValidMatch = "valid_match"
	MarkerMismatch   = "mismatch"
	MarkerInvalid    = "invalid"
)

// ProviderProbe is the modeled materialize-status reading: the plugin
// state with ID, plan, and token agreement against the journal.
type ProviderProbe struct {
	State        string
	IDsMatch     bool
	PlanMatches  bool
	TokenMatches bool
}

// BridgeProbe is the modeled bridge-status reading: the manager state
// with binding and reference agreement, lease authority for adopt,
// liveness, and token expiry.
type BridgeProbe struct {
	State          string
	BindingMatches bool
	ManagerMatches bool
	LeaseActive    bool
	Live           bool
	TokenExpired   bool
	Failed         bool
}

// LeaseIdentity is the winning-lease observation before or after the
// crash: epoch plus fencing token.
type LeaseIdentity struct {
	Epoch uint64 `json:"epoch"`
	ID    string `json:"lease_id"`
}

// RecoveryInput is the status-first recovery closure: the crash
// boundary and path, one status probe per possibly executed external
// effect, the host condition, the winning lease and native
// identity/binding before and after, and the rollback-closure proofs.
// Durable facts (journal, receipt, plan view) are read first from the
// store; these modeled inputs stand in for the provider plugin,
// bridge process, replica files, and lease arbitration this package
// never drives.
type RecoveryInput struct {
	Boundary    string
	Path        string
	Provider    ProviderProbe
	Bridge      BridgeProbe
	Marker      string
	Host        string
	LeaseKnown  bool
	LeaseBefore LeaseIdentity
	LeaseAfter  LeaseIdentity
	// NativeKnown gates the native identity/binding comparison. It is
	// required only when the journal carries a provider transaction
	// or an adopted bridge binding; otherwise no native identity
	// exists to compare.
	NativeKnown  bool
	NativeBefore string
	NativeAfter  string
	// ExternalEffect describes the possibly executed external effect
	// the probes reconcile (for example "bridge adopt may have
	// adopted the dormant manager").
	ExternalEffect string
	// MarkerBytes carries the probe-attested destination marker for
	// a workspace commit completion; PriorMarkerBytes carries its
	// predecessor link when non-null. Both validate through the
	// closed marker shape before anything binds them.
	MarkerBytes      []byte
	PriorMarkerBytes []byte
	// PredecessorsRestored and StagingRemoved are the probe-verified
	// rollback-closure facts.
	PredecessorsRestored bool
	StagingRemoved       bool
}

// Evidence is the Section 13.13 conformance record for one recovery
// assessment: boundary ID, path, operation IDs, pre/post durable
// facts, external effect and status probe, winning lease and native
// identity before/after, the selected outcome, and the reason and
// remediation satisfying that outcome. Parked and rollback assessments
// persist durably as recovery.json; safe_retry establishes no new
// durable fact beyond the journal and receipt it returns.
type Evidence struct {
	MaterializationID string            `json:"materialization_id"`
	Boundary          string            `json:"boundary"`
	Path              string            `json:"path"`
	OperationIDs      map[string]string `json:"operation_ids"`
	PhaseBefore       string            `json:"phase_before"`
	PhaseAfter        string            `json:"phase_after"`
	ReceiptPresent    bool              `json:"receipt_present"`
	ProviderState     string            `json:"provider_state"`
	BridgeState       string            `json:"bridge_state"`
	ExternalEffect    string            `json:"external_effect"`
	StatusProbe       string            `json:"status_probe"`
	LeaseBefore       LeaseIdentity     `json:"lease_before"`
	LeaseAfter        LeaseIdentity     `json:"lease_after"`
	NativeBefore      string            `json:"native_before"`
	NativeAfter       string            `json:"native_after"`
	Selected          Outcome           `json:"selected"`
	Reason            string            `json:"reason"`
	Remediation       string            `json:"remediation"`
	DecidedAt         string            `json:"decided_at"`
}

// parkedFailure builds the blocking-reason Structured Error for a
// parked assessment: the named code with retryable false (parked
// ambiguity is never retry permission).
func parkedFailure(code axerror.Code, message string, operation string) (*axerror.Error, error) {
	ids := axerror.NoIDs()
	if operation != "" {
		parsed, err := scalar.ParseUUIDv7(operation)
		if err == nil {
			ids = ids.WithOperation(parsed)
		}
	}
	return axerror.New(axerror.Spec{
		Version:   lastErrorVersion,
		Code:      code,
		Message:   message,
		Retryable: false,
		IDs:       ids,
		Details:   axerror.Details{},
	})
}

// rollbackFailure builds the terminal-reason Structured Error for an
// executed rollback.
func rollbackFailure(code axerror.Code, message string, operation string) (*axerror.Error, error) {
	return parkedFailure(code, message, operation)
}

// Recover classifies one crashed materialization after a clean restart
// into exactly one Section 13.13 outcome. It reads durable facts first
// (journal, receipt, plan view), then applies the rejection gates in
// order: unknown or moved authority, substitution, two live
// authorities, unfenced continuation, host conditions, marker
// integrity, unknown decisive status, and contradictory probes all
// park. Rollback-required facts execute the allowed abort and return
// explicit_rollback; provable terminal completions (commit, dormant
// finalize, rollback close) apply and return safe_retry; every other
// consistent assessment resumes or replays with safe_retry.
//
// Parked assessments record the blocking reason as last_error (except
// on a terminal journal, which stays frozen) and persist the evidence
// as recovery.json. Rollback assessments persist the terminal journal
// and the evidence. Safe_retry assessments mutate nothing: the journal
// and receipt they return are the evidence.
func (store *Store) Recover(materializationID string, input RecoveryInput) (Outcome, Evidence, error) {
	id, err := checkUUIDv7(materializationID, "materialization_id")
	if err != nil {
		return "", Evidence{}, err
	}
	if !validBoundary(input.Boundary) {
		return "", Evidence{}, invalid("recovery boundary %q is outside CR-MAT-01..08", input.Boundary)
	}
	if !validPath(input.Path) {
		return "", Evidence{}, invalid("recovery path %q is outside the CR-MAT path set", input.Path)
	}
	if err := checkRecoveryProbes(&input); err != nil {
		return "", Evidence{}, err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	matDir := filepath.Join(store.root, id)
	raw, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		if os.IsNotExist(err) {
			if input.Boundary != BoundaryBeforeCreate {
				return "", Evidence{}, fmt.Errorf("%w: %s", ErrUnknownJournal, id)
			}
			return store.recoverBeforeCreateLocked(id, input)
		}
		return "", Evidence{}, err
	}
	view, err := store.readPlanViewLocked(matDir)
	if err != nil {
		return "", Evidence{}, err
	}
	journal, err := decodeJournal(raw, view)
	if err != nil {
		return store.parkTornLocked(id, matDir, input, err)
	}
	_, receiptPresent, err := store.readReceiptLocked(matDir, journal.PrepareOperationID)
	if err != nil {
		return "", Evidence{}, err
	}
	assessment := recoveryAssessment{
		store:          store,
		matDir:         matDir,
		journal:        &journal,
		view:           view,
		input:          input,
		receiptPresent: receiptPresent,
	}
	return assessment.decide()
}

// checkRecoveryProbes validates the modeled probe enums and the lease
// observations.
func checkRecoveryProbes(input *RecoveryInput) error {
	switch input.Provider.State {
	case ProviderUnknown, ProviderPrepared, ProviderCommitted, ProviderRolledBack:
	default:
		return invalid("provider probe state %q is outside the closed enum", input.Provider.State)
	}
	switch input.Bridge.State {
	case ProviderUnknown, BoardNotStarted, BoardImported, BoardOpened, BoardAdopted,
		BoardResumed, BoardDormantFinalized, BoardRolledBack, BoardFailed,
		"dormant", "quiesced", "stopped", "running", "idle":
	default:
		return invalid("bridge probe state %q is outside the closed vocabulary", input.Bridge.State)
	}
	switch input.Marker {
	case MarkerAbsent, MarkerValidMatch, MarkerMismatch, MarkerInvalid:
	default:
		return invalid("marker probe %q is outside the closed vocabulary", input.Marker)
	}
	switch input.Host {
	case HostOK, HostDiskFull, HostRenameBlocked:
	default:
		return invalid("host condition %q is outside the closed vocabulary", input.Host)
	}
	if input.LeaseKnown {
		if input.LeaseBefore.Epoch < 1 || input.LeaseBefore.Epoch > uint53Max {
			return invalid("recovery winning lease epoch %d is outside 1..2^53-1", input.LeaseBefore.Epoch)
		}
		if _, err := scalar.ParseUUIDv4(input.LeaseBefore.ID); err != nil {
			return invalid("recovery winning lease id: %v", err)
		}
		if input.LeaseAfter.Epoch < 1 || input.LeaseAfter.Epoch > uint53Max {
			return invalid("recovery winning lease epoch %d is outside 1..2^53-1", input.LeaseAfter.Epoch)
		}
		if _, err := scalar.ParseUUIDv4(input.LeaseAfter.ID); err != nil {
			return invalid("recovery winning lease id: %v", err)
		}
	}
	return nil
}

// recoverBeforeCreateLocked classifies CR-MAT-01 with no journal:
// nothing durable exists, so the same IDs retry fresh.
func (store *Store) recoverBeforeCreateLocked(id string, input RecoveryInput) (Outcome, Evidence, error) {
	decided, err := checkTimestamp(store.now(), "decided_at")
	if err != nil {
		return "", Evidence{}, err
	}
	evidence := Evidence{
		MaterializationID: id,
		Boundary:          input.Boundary,
		Path:              input.Path,
		OperationIDs:      map[string]string{},
		PhaseBefore:       "",
		PhaseAfter:        "",
		ReceiptPresent:    false,
		ProviderState:     "",
		BridgeState:       "",
		ExternalEffect:    input.ExternalEffect,
		StatusProbe:       statusSummary(input),
		LeaseBefore:       input.LeaseBefore,
		LeaseAfter:        input.LeaseAfter,
		NativeBefore:      input.NativeBefore,
		NativeAfter:       input.NativeAfter,
		Selected:          OutcomeSafeRetry,
		Reason:            "no journal exists: nothing durable mutated, the same prepare IDs retry fresh",
		Remediation:       "retry materialize.prepare with the caller-retained operation and materialization IDs and the identical body",
		DecidedAt:         decided,
	}
	return OutcomeSafeRetry, evidence, nil
}

// parkTornLocked parks a torn journal: the bytes fail validation, so
// recovery quarantines with integrity_failure and persists the
// evidence. The torn bytes are never repaired or trusted.
func (store *Store) parkTornLocked(id, matDir string, input RecoveryInput, cause error) (Outcome, Evidence, error) {
	decided, err := checkTimestamp(store.now(), "decided_at")
	if err != nil {
		return "", Evidence{}, err
	}
	evidence := Evidence{
		MaterializationID: id,
		Boundary:          input.Boundary,
		Path:              input.Path,
		OperationIDs:      map[string]string{},
		PhaseBefore:       "unknown",
		PhaseAfter:        "unknown",
		ReceiptPresent:    false,
		ExternalEffect:    input.ExternalEffect,
		StatusProbe:       statusSummary(input),
		LeaseBefore:       input.LeaseBefore,
		LeaseAfter:        input.LeaseAfter,
		NativeBefore:      input.NativeBefore,
		NativeAfter:       input.NativeAfter,
		Selected:          OutcomeRecoverableParked,
		Reason:            fmt.Sprintf("journal bytes fail validation, quarantined: %v", cause),
		Remediation:       "inspect the torn journal, restore the materialization directory from a verified copy, then re-run recovery",
		DecidedAt:         decided,
	}
	if err := writeRecoveryLocked(matDir, evidence); err != nil {
		return "", Evidence{}, err
	}
	return OutcomeRecoverableParked, evidence, nil
}

// recoveryAssessment carries one recovery decision with its durable
// facts and modeled probes. pendingMarker holds the validated
// destination marker once commitProvable proves the commit.
type recoveryAssessment struct {
	store          *Store
	matDir         string
	journal        *Journal
	view           planView
	input          RecoveryInput
	receiptPresent bool
	pendingMarker  *Marker
}

// decide applies the gate chain and returns the single outcome.
func (assessment *recoveryAssessment) decide() (Outcome, Evidence, error) {
	journal := assessment.journal
	input := assessment.input
	terminal := terminalPhase(journal.Phase)

	// Rejection gates that park even on a terminal journal: two live
	// authorities, substitution, and unfenced continuation are never a
	// successful recovery.
	if input.LeaseKnown && (input.LeaseAfter.Epoch != input.LeaseBefore.Epoch || input.LeaseAfter.ID != input.LeaseBefore.ID) {
		return assessment.park("lease_conflict", "winning lease moved across the crash; reconcile the deterministic winner before any retry",
			"recompute the winning lease, fence the loser, then re-run recovery with the unchanged winner")
	}
	if assessment.nativeRequired() {
		if !input.NativeKnown {
			return assessment.park("operation_uncertain", "native identity/binding evidence is missing for an adopted transaction",
				"reconcile the exact persisted provider identity or task-board binding, then re-run recovery")
		}
		if input.NativeAfter != input.NativeBefore {
			return assessment.park("lease_conflict", "native identity/binding differs across the crash: substitution is never a successful recovery",
				"restore the exact persisted identity or binding; a fresh handle, blank state, or different realm never recovers this operation")
		}
	}
	if input.Provider.State == ProviderCommitted && input.Provider.PlanMatches && input.Provider.IDsMatch &&
		assessment.bridgeLive() && !input.Bridge.BindingMatches {
		return assessment.park("lease_conflict", "two live authorities: the provider committed while a differently bound manager is live",
			"fence one authority with visible divergent-history evidence, then re-run recovery")
	}
	if assessment.bridgeLive() && !input.Bridge.LeaseActive && assessment.bridgeAdopted() {
		return assessment.park("stale_owner", "unfenced continuation: an adopted manager is live without the winning lease",
			"fence or stop the losing manager with visible evidence, then re-run recovery")
	}
	// Terminal journals return their recorded result once the
	// authority gates pass: probes cannot reopen them.
	if terminal {
		return assessment.completeSafeRetry("journal is terminal in %s: returning the recorded result", journal.Phase)
	}
	// Authority and evidence gates for non-terminal journals.
	if !input.LeaseKnown {
		return assessment.park("operation_uncertain", "winning lease evidence is missing",
			"reconcile the winning lease, then re-run recovery")
	}
	switch input.Host {
	case HostDiskFull:
		return assessment.park("staging_incomplete", "destination disk full: existing destination retained, staging partial",
			"free space, then resume the same operation")
	case HostRenameBlocked:
		return assessment.park("staging_incomplete", "atomic rename blocked: staging retained, never overwritten in place",
			"close the blocking handles, then retry the same operation")
	}
	switch input.Marker {
	case MarkerMismatch:
		return assessment.park("integrity_failure", "destination marker disagrees with the journal (MJ-CRASH-MARKER-MISMATCH): quarantined, nothing further mutates",
			"quarantine the transaction, reconcile the marker against the plan and checkpoint, then re-run recovery")
	case MarkerInvalid:
		return assessment.park("integrity_failure", "destination marker is invalid: quarantined, nothing further mutates",
			"quarantine the transaction, reconcile the marker bytes, then re-run recovery")
	}
	if input.Provider.State == ProviderUnknown && assessment.providerEffectUncertain() {
		return assessment.park("operation_uncertain", "provider status is unknown after a possibly executed effect",
			"call materialize-status with the same IDs until the state is known, then re-run recovery")
	}
	if journal.TaskBoard != nil && input.Bridge.State == ProviderUnknown && assessment.bridgeEffectUncertain() {
		return assessment.park("operation_uncertain", "bridge status is unknown after a possibly executed effect",
			"query bridge status by reference and binding, then re-run recovery")
	}
	if journal.Provider != nil && journal.Provider.State == ProviderPrepared &&
		input.Provider.State == ProviderPrepared && !input.Provider.TokenMatches {
		return assessment.park("integrity_failure", "provider token disagrees between the journal and the status probe",
			"reconcile the prepared token from the plugin root without allocating a second transaction, then re-run recovery")
	}
	if journal.TaskBoard != nil && (journal.TaskBoard.State == BoardAdopted || journal.TaskBoard.State == BoardResumed) &&
		(assessment.bridgeLive() || input.Bridge.State == BoardAdopted || input.Bridge.State == BoardResumed) &&
		(!input.Bridge.BindingMatches || !input.Bridge.ManagerMatches) {
		return assessment.park("lease_conflict", "bridge reports another session, epoch, lease, or manager (TB-TXN-BINDING-MISMATCH): evidence preserved, nothing further mutates",
			"reconcile the exact Ax Binding and manager reference, then re-run recovery")
	}
	if (journal.Phase == PhaseRollingBack || journal.Phase == PhaseRolledBack) &&
		((journal.TaskBoard != nil && (journal.TaskBoard.State == BoardAdopted || journal.TaskBoard.State == BoardResumed)) ||
			(journal.Provider != nil && journal.Provider.State == ProviderCommitted)) {
		return assessment.park("integrity_failure", "journal contradicts itself: rollback phase with an activated sub-state",
			"inspect the torn journal progression, restore a verified copy, then re-run recovery")
	}
	if (journal.Phase == PhaseStaging || journal.Phase == PhaseValidating || journal.Phase == PhasePrepared) &&
		assessment.activatedEarly() {
		return assessment.park("integrity_failure", "journal contradicts itself: an activated sub-state before the committing phase",
			"inspect the torn journal progression, restore a verified copy, then re-run recovery")
	}
	// Rollback-required facts execute the allowed abort.
	if rollback, code, reason, remediation := assessment.rollbackRequired(); rollback {
		return assessment.executeRollback(code, reason, remediation)
	}
	// Provable terminal completions apply and return safe_retry.
	if done, outcome, evidence, err := assessment.tryComplete(); done {
		return outcome, evidence, err
	}
	// Every other consistent assessment resumes or replays.
	return assessment.completeSafeRetry("%s", assessment.resumeReason())
}

// nativeRequired reports whether a native identity/binding comparison
// exists: a committed provider transaction (live native handles) or an
// adopted bridge binding. A merely prepared transaction has no live
// identity to substitute, so no comparison exists yet.
func (assessment *recoveryAssessment) nativeRequired() bool {
	if assessment.journal.Provider != nil && assessment.journal.Provider.State == ProviderCommitted {
		return true
	}
	board := assessment.journal.TaskBoard
	return board != nil && (board.State == BoardAdopted || board.State == BoardResumed)
}

// bridgeLive reports a live manager from the probe.
func (assessment *recoveryAssessment) bridgeLive() bool {
	if assessment.input.Bridge.Live {
		return true
	}
	switch assessment.input.Bridge.State {
	case "running", "idle":
		return true
	default:
		return false
	}
}

// bridgeAdopted reports whether the journal or the probe names an
// adopted manager.
func (assessment *recoveryAssessment) bridgeAdopted() bool {
	board := assessment.journal.TaskBoard
	if board != nil && (board.State == BoardAdopted || board.State == BoardResumed) {
		return true
	}
	switch assessment.input.Bridge.State {
	case BoardAdopted, BoardResumed:
		return true
	default:
		return false
	}
}

// providerEffectUncertain reports whether an unknown provider probe
// leaves a possibly executed effect unreconciled: a commit or rollback
// may have executed once the journal reached prepared, or a prepare
// may have executed unrecorded from CR-MAT-05 on. A recorded prepared
// token at staging or validating stands on its own: the commit is
// impossible there, so the unknown probe is vacuous.
func (assessment *recoveryAssessment) providerEffectUncertain() bool {
	journal := assessment.journal
	if !assessment.providerParticipates() {
		return false
	}
	if journal.Provider != nil {
		return journal.Phase == PhasePrepared || journal.Phase == PhaseCommitting ||
			journal.Phase == PhaseRollingBack
	}
	return boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterPrepareOp)
}

// providerParticipates reports whether a provider branch exists: a
// recorded transaction or a provider-store plan authority. Plans
// without one leave the provider probe vacuous.
func (assessment *recoveryAssessment) providerParticipates() bool {
	if assessment.journal.Provider != nil {
		return true
	}
	for _, authority := range assessment.view.Authorities {
		if authority.Kind == PlanKindProviderStore {
			return true
		}
	}
	return false
}

// bridgeEffectUncertain reports whether an unknown bridge probe leaves
// a possibly executed effect unreconciled: an adopted manager may have
// acted, a recorded import or open may have advanced once the journal
// reached prepared, or an import may have executed unrecorded from
// CR-MAT-05 on. Recorded dormant, rolled-back, and failed states stand
// on their own.
func (assessment *recoveryAssessment) bridgeEffectUncertain() bool {
	journal := assessment.journal
	state := journal.TaskBoard.State
	switch state {
	case BoardAdopted, BoardResumed:
		return true
	case BoardImported, BoardOpened:
		return journal.Phase == PhasePrepared || journal.Phase == PhaseCommitting ||
			journal.Phase == PhaseRollingBack
	case BoardNotStarted:
		return boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterPrepareOp)
	default:
		return false
	}
}

// boundaryAtOrAfter orders the CR-MAT registry: boundaries sort
// lexicographically from CR-MAT-01 to CR-MAT-08.
func boundaryAtOrAfter(boundary, floor string) bool {
	return boundary >= floor
}

// rollbackRequired decides whether the durable facts plus probes
// require the allowed abort: bridge failure or token expiry before
// activation, a provider that rolled back underneath a prepared
// journal, and the rolling_back retry/close rules.
func (assessment *recoveryAssessment) rollbackRequired() (bool, axerror.Code, string, string) {
	journal := assessment.journal
	input := assessment.input
	preActivation := true
	if journal.TaskBoard != nil && (journal.TaskBoard.State == BoardAdopted || journal.TaskBoard.State == BoardResumed) {
		preActivation = false
	}
	if journal.Provider != nil && journal.Provider.State == ProviderCommitted {
		preActivation = false
	}
	if journal.TaskBoard != nil && preActivation && !input.Bridge.LeaseActive &&
		(input.Bridge.Failed || input.Bridge.TokenExpired) {
		if input.Bridge.TokenExpired {
			return true, "staging_incomplete", "bridge token expired before lease activation: rolling back the dormant import",
				"re-run prepare from the same checkpoint with fresh operation IDs after the rollback"
		}
		return true, "task_board_bridge_unavailable", "bridge import, open, or adopt failed before the new lease: rolling back the dormant import",
			"retry the same operation after the rollback, or inspect the bridge failure"
	}
	if journal.Provider != nil && journal.Phase == PhasePrepared &&
		input.Provider.State == ProviderRolledBack && preActivation {
		return true, "rollback_failed", "provider rolled back underneath a prepared journal: converging the journal",
			"inspect the provider rollback, then retry the same operation from the recorded checkpoint"
	}
	if journal.Phase == PhaseRollingBack && input.Provider.State == ProviderPrepared && journal.Provider != nil {
		return true, "operation_uncertain", "journal is rolling_back with the provider still prepared: retrying the rollback",
			"no operator action: the rollback retries under the same IDs"
	}
	return false, "", "", ""
}

// tryComplete applies provable terminal completions: a committing
// journal with committed provider status plus the exact destination
// marker commits; a dormant finalize closes; a rolling_back journal
// with rolled_back status plus restored predecessors closes. It
// reports done=false when nothing completes.
func (assessment *recoveryAssessment) tryComplete() (bool, Outcome, Evidence, error) {
	journal := assessment.journal
	input := assessment.input
	if journal.Phase == PhaseCommitting {
		provable, contradictory := assessment.commitProvable()
		if contradictory {
			outcome, evidence, err := assessment.park("integrity_failure", "destination marker evidence contradicts the valid-match probe",
				"reconcile the marker bytes against the journal and plan, then re-run recovery")
			return true, outcome, evidence, err
		}
		if provable {
			if journal.TaskBoard != nil && journal.TaskBoard.State == BoardOpened &&
				journal.TaskBoard.ActivationMode == ModeDormantReplica && assessment.dormantProvable() {
				if err := assessment.applyDormantFinalize(); err != nil {
					return true, "", Evidence{}, err
				}
			}
			if err := assessment.applyCommit(); err != nil {
				return true, "", Evidence{}, err
			}
			outcome, evidence, err := assessment.completeSafeRetry("committing journal completed to committed on exact status plus destination evidence")
			return true, outcome, evidence, err
		}
		if assessment.commitConverged() && journal.ManagedReplicaID != nil {
			outcome, evidence, err := assessment.park("operation_uncertain", "committing journal converged but the exact destination marker is unproven",
				"revalidate every plan authority, predecessor, installed manifest byte, and transaction, then re-run recovery with the marker evidence")
			return true, outcome, evidence, err
		}
	}
	if journal.Phase == PhaseRollingBack && input.Provider.State == ProviderRolledBack {
		if journal.Provider == nil {
			return false, "", Evidence{}, nil
		}
		if !input.PredecessorsRestored && !input.StagingRemoved {
			outcome, evidence, err := assessment.park("operation_uncertain", "provider rolled back but predecessor restoration is unproven",
				"verify predecessor restoration or inert staging removal, then re-run recovery")
			return true, outcome, evidence, err
		}
		failure, err := rollbackFailure("operation_uncertain", "rollback response lost; provider rolled back and predecessors verified", journal.PrepareOperationID)
		if err != nil {
			return true, "", Evidence{}, err
		}
		framed, err := failure.MarshalJSON()
		if err != nil {
			return true, "", Evidence{}, err
		}
		rolled, err := assessment.store.rollbackLocked(journal.MaterializationID, framed)
		if err != nil {
			return true, "", Evidence{}, err
		}
		assessment.journal = &rolled
		outcome, evidence, err := assessment.completeRollback("rolling_back journal closed to rolled_back on rolled_back status plus verified restoration",
			"no operator action: the terminal rollback is durable and visible")
		return true, outcome, evidence, err
	}
	return false, "", Evidence{}, nil
}

// commitProvable reports whether the committing journal may complete:
// every participating transaction converged and the destination
// evidence is exact (a validated binding marker for workspace
// transactions). Contradictory marker evidence (a valid-match probe
// with bytes that fail) parks instead of completing.
func (assessment *recoveryAssessment) commitProvable() (provable, contradictory bool) {
	journal := assessment.journal
	input := assessment.input
	if journal.Provider != nil {
		if input.Provider.State != ProviderCommitted || !input.Provider.PlanMatches || !input.Provider.IDsMatch {
			return false, false
		}
	}
	if journal.TaskBoard != nil {
		switch journal.TaskBoard.State {
		case BoardResumed:
			if input.Bridge.State != BoardResumed && input.Bridge.State != "running" && input.Bridge.State != "idle" {
				return false, false
			}
			if !input.Bridge.BindingMatches || !input.Bridge.ManagerMatches {
				return false, false
			}
		case BoardDormantFinalized:
			if input.Bridge.State != BoardDormantFinalized && input.Bridge.State != "dormant" {
				return false, false
			}
		case BoardOpened:
			if journal.TaskBoard.ActivationMode != ModeDormantReplica || !assessment.dormantProvable() {
				return false, false
			}
		default:
			return false, false
		}
	}
	for _, state := range journal.AuthorityStates {
		if state.State != AuthorityCommitted {
			return false, false
		}
	}
	if journal.ManagedReplicaID == nil {
		return true, false
	}
	if input.Marker != MarkerValidMatch {
		return false, false
	}
	marker, err := ValidateMarker(input.MarkerBytes)
	if err != nil {
		return false, true
	}
	if err := checkMarkerBinding(marker, journal, input.PriorMarkerBytes); err != nil {
		return false, true
	}
	assessment.pendingMarker = &marker
	return true, false
}

// commitConverged reports whether every participating transaction and
// authority converged, leaving only the destination marker unproven.
func (assessment *recoveryAssessment) commitConverged() bool {
	journal := assessment.journal
	input := assessment.input
	if journal.Provider != nil {
		if input.Provider.State != ProviderCommitted || !input.Provider.PlanMatches || !input.Provider.IDsMatch {
			return false
		}
	}
	if journal.TaskBoard != nil {
		switch journal.TaskBoard.State {
		case BoardResumed:
			if !input.Bridge.BindingMatches || !input.Bridge.ManagerMatches {
				return false
			}
		case BoardDormantFinalized:
		case BoardOpened:
			if journal.TaskBoard.ActivationMode != ModeDormantReplica || !assessment.dormantProvable() {
				return false
			}
		default:
			return false
		}
	}
	for _, state := range journal.AuthorityStates {
		if state.State != AuthorityCommitted {
			return false
		}
	}
	return true
}

// dormantProvable reports whether the opened dormant bridge may
// finalize: the probe shows the dormant manager with its reference.
func (assessment *recoveryAssessment) dormantProvable() bool {
	input := assessment.input
	if input.Bridge.State != BoardDormantFinalized && input.Bridge.State != "dormant" {
		return false
	}
	return input.Bridge.ManagerMatches
}

// applyDormantFinalize records the dormant finalization: the open
// token copy is destroyed, dormant_finalized records with cleanup
// pending expiry at the consumed open expiry.
func (assessment *recoveryAssessment) applyDormantFinalize() error {
	journal := assessment.journal
	board := *journal.TaskBoard
	if board.OpenExpiresAt == nil {
		return invalid("dormant finalize requires the consumed open expiry")
	}
	dormant := "dormant"
	board.State = BoardDormantFinalized
	board.OpenToken = nil
	board.OpenExpiresAt = nil
	board.LastBridgeState = &dormant
	board.CleanupState = "pending_expiry"
	board.CleanupAfter = journal.TaskBoard.OpenExpiresAt
	updated, err := assessment.store.updateTaskBoardLocked(journal.MaterializationID, board)
	if err != nil {
		return err
	}
	assessment.journal = &updated
	return nil
}

// applyCommit converges the sub-states and commits the journal. The
// provider transaction commits, and the destination marker binds for
// workspace transactions from the already validated marker evidence
// the commitProvable gate required.
func (assessment *recoveryAssessment) applyCommit() error {
	journal := assessment.journal
	if journal.Provider != nil {
		provider := *journal.Provider
		provider.State = ProviderCommitted
		provider.RollbackToken = nil
		updated, err := assessment.store.updateProviderLocked(journal.MaterializationID, provider)
		if err != nil {
			return err
		}
		assessment.journal = &updated
		journal = &updated
	}
	// The marker ID binds from the validated probe evidence: the
	// committing journal records the exact marker commitProvable
	// proved against the closed shape and the journal binding.
	markerID := ""
	if assessment.pendingMarker != nil {
		markerID = assessment.pendingMarker.MarkerID
	}
	committed, err := assessment.store.commitWithMarkerLocked(journal.MaterializationID, markerID)
	if err != nil {
		return err
	}
	assessment.journal = &committed
	return nil
}

// activatedEarly reports whether an activated sub-state (a committed
// provider, an adopted or resumed bridge, a dormant finalization) is
// recorded before the committing phase, where activation happens. The
// coordinator adopts, resumes, commits, and finalizes only during
// finalize, so an earlier activation contradicts the progression.
func (assessment *recoveryAssessment) activatedEarly() bool {
	journal := assessment.journal
	if journal.Provider != nil && journal.Provider.State == ProviderCommitted {
		return true
	}
	if journal.TaskBoard != nil {
		switch journal.TaskBoard.State {
		case BoardAdopted, BoardResumed, BoardDormantFinalized:
			return true
		}
	}
	return false
}

// resumeReason describes the safe_retry continuation for the journal
// phase, the bridge position, and the crash boundary. Bridge lost
// responses name their exact same-ID retry once their boundary may
// have executed the effect; otherwise the phase-generic continuation
// applies.
func (assessment *recoveryAssessment) resumeReason() string {
	journal := assessment.journal
	input := assessment.input
	// An adopted bridge names its resume retry first: the adopted
	// manager is the finest pending truth at any phase.
	if journal.TaskBoard != nil && journal.TaskBoard.State == BoardAdopted &&
		boundaryAtOrAfter(input.Boundary, BoundaryAfterFinalize) {
		return "bridge resume may have started provider work: retrying resume with the same ID and profile after status"
	}
	// Prepared and committing continuations precede the earlier
	// bridge positions: the phase already passed them.
	if journal.Phase == PhasePrepared && input.Provider.State == ProviderPrepared {
		return "prepared journal with the prepared token durable continues to commit under the same IDs"
	}
	if journal.Phase == PhaseCommitting && input.Provider.State == ProviderPrepared {
		return "committing journal with the provider still prepared retries the commit under the same IDs"
	}
	if journal.TaskBoard != nil {
		switch journal.TaskBoard.State {
		case BoardNotStarted:
			if boundaryAtOrAfter(input.Boundary, BoundaryAfterPrepareOp) {
				return "bridge import may have executed: retrying import with the identical operation ID and body"
			}
		case BoardImported:
			if boundaryAtOrAfter(input.Boundary, BoundaryAfterOpen) {
				return "bridge open may have executed: retrying open with its stable ID and persisting before commit"
			}
		case BoardOpened:
			if input.Bridge.LeaseActive && boundaryAtOrAfter(input.Boundary, BoundaryAfterActivation) {
				return "bridge adopt may proceed under the authoritative lease: retrying adopt with the same ID and exact binding"
			}
			if !input.Bridge.LeaseActive {
				return "bridge opened dormant; adoption waits for the authoritative lease"
			}
		}
	}
	if journal.Phase == PhaseStaging || journal.Phase == PhaseValidating {
		return "valid staging transaction resumes: requesting only absent chunk indexes, never marking another blob's equal index present"
	}
	return "durable facts and status reconcile: the same operation continues under its caller-stable IDs"
}

// completeSafeRetry returns the safe_retry outcome with its evidence.
// It mutates nothing: the assessment only classified.
func (assessment *recoveryAssessment) completeSafeRetry(format string, arguments ...any) (Outcome, Evidence, error) {
	decided, err := checkTimestamp(assessment.store.now(), "decided_at")
	if err != nil {
		return "", Evidence{}, err
	}
	evidence := assessment.baseEvidence()
	evidence.PhaseAfter = assessment.journal.Phase
	evidence.Selected = OutcomeSafeRetry
	evidence.Reason = fmt.Sprintf(format, arguments...)
	evidence.Remediation = "continue the same operation with every caller-stable ID and a byte-identical body"
	evidence.DecidedAt = decided
	return OutcomeSafeRetry, evidence, nil
}

// completeRollback returns the explicit_rollback outcome with its
// evidence and persists the evidence durably beside the terminal
// journal.
func (assessment *recoveryAssessment) completeRollback(reason, remediation string) (Outcome, Evidence, error) {
	decided, err := checkTimestamp(assessment.store.now(), "decided_at")
	if err != nil {
		return "", Evidence{}, err
	}
	evidence := assessment.baseEvidence()
	evidence.PhaseAfter = assessment.journal.Phase
	evidence.Selected = OutcomeExplicitRollback
	evidence.Reason = reason
	evidence.Remediation = remediation
	evidence.DecidedAt = decided
	if err := writeRecoveryLocked(assessment.matDir, evidence); err != nil {
		return "", Evidence{}, err
	}
	return OutcomeExplicitRollback, evidence, nil
}

// park records the recoverable_parked_state outcome: the blocking
// reason persists as last_error (except on a terminal journal, which
// stays frozen) and the evidence persists as recovery.json with the
// retained lease, exact IDs, checkpoint/native identity, and error.
func (assessment *recoveryAssessment) park(code axerror.Code, reason, remediation string) (Outcome, Evidence, error) {
	journal := assessment.journal
	failure, err := parkedFailure(code, reason, journal.PrepareOperationID)
	if err != nil {
		return "", Evidence{}, err
	}
	if !terminalPhase(journal.Phase) {
		framed, err := failure.MarshalJSON()
		if err != nil {
			return "", Evidence{}, err
		}
		updated, err := assessment.store.recordErrorLocked(journal.MaterializationID, framed)
		if err != nil {
			return "", Evidence{}, err
		}
		assessment.journal = &updated
	}
	decided, err := checkTimestamp(assessment.store.now(), "decided_at")
	if err != nil {
		return "", Evidence{}, err
	}
	evidence := assessment.baseEvidence()
	evidence.PhaseAfter = assessment.journal.Phase
	evidence.Selected = OutcomeRecoverableParked
	evidence.Reason = reason
	evidence.Remediation = remediation
	evidence.DecidedAt = decided
	if err := writeRecoveryLocked(assessment.matDir, evidence); err != nil {
		return "", Evidence{}, err
	}
	return OutcomeRecoverableParked, evidence, nil
}

// executeRollback runs the allowed abort and returns
// explicit_rollback with the persisted evidence.
func (assessment *recoveryAssessment) executeRollback(code axerror.Code, reason, remediation string) (Outcome, Evidence, error) {
	journal := assessment.journal
	failure, err := rollbackFailure(code, reason, journal.PrepareOperationID)
	if err != nil {
		return "", Evidence{}, err
	}
	framed, err := failure.MarshalJSON()
	if err != nil {
		return "", Evidence{}, err
	}
	rolled, err := assessment.store.rollbackLocked(journal.MaterializationID, framed)
	if err != nil {
		return "", Evidence{}, err
	}
	assessment.journal = &rolled
	return assessment.completeRollback(reason, remediation)
}

// baseEvidence assembles the shared evidence record: boundary, path,
// operation IDs, pre/post durable facts, external effect and status
// probe, winning lease and native identity before/after.
func (assessment *recoveryAssessment) baseEvidence() Evidence {
	journal := assessment.journal
	input := assessment.input
	operations := map[string]string{
		"materialization_id":   journal.MaterializationID,
		"prepare_operation_id": journal.PrepareOperationID,
	}
	if journal.Provider != nil {
		operations["provider_operation_id"] = journal.Provider.OperationID
		operations["provider_transaction_id"] = journal.Provider.TransactionID
	}
	if journal.TaskBoard != nil {
		operations["import_operation_id"] = journal.TaskBoard.ImportOperationID
		operations["open_operation_id"] = journal.TaskBoard.OpenOperationID
		operations["adopt_operation_id"] = journal.TaskBoard.AdoptOperationID
		operations["resume_operation_id"] = journal.TaskBoard.ResumeOperationID
	}
	providerState := ""
	if journal.Provider != nil {
		providerState = journal.Provider.State
	}
	bridgeState := ""
	if journal.TaskBoard != nil {
		bridgeState = journal.TaskBoard.State
	}
	return Evidence{
		MaterializationID: journal.MaterializationID,
		Boundary:          input.Boundary,
		Path:              input.Path,
		OperationIDs:      operations,
		PhaseBefore:       journal.Phase,
		ReceiptPresent:    assessment.receiptPresent,
		ProviderState:     providerState,
		BridgeState:       bridgeState,
		ExternalEffect:    input.ExternalEffect,
		StatusProbe:       statusSummary(input),
		LeaseBefore:       input.LeaseBefore,
		LeaseAfter:        input.LeaseAfter,
		NativeBefore:      input.NativeBefore,
		NativeAfter:       input.NativeAfter,
	}
}

// statusSummary renders the probe reading for the evidence record.
func statusSummary(input RecoveryInput) string {
	return fmt.Sprintf("provider=%s bridge=%s marker=%s host=%s",
		input.Provider.State, input.Bridge.State, input.Marker, input.Host)
}

// writeRecoveryLocked persists one recovery assessment as
// recovery.json through atomic replace: each record is self-contained
// (boundary, facts, outcome, reason, remediation), so the latest
// assessment supersedes without losing resumable facts.
func writeRecoveryLocked(matDir string, evidence Evidence) error {
	framed, err := json.Marshal(evidence)
	if err != nil {
		return fmt.Errorf("encode recovery evidence: %w", err)
	}
	if err := replaceAtomic(filepath.Join(matDir, "recovery.json"), framed); err != nil {
		return err
	}
	return syncDirectory(matDir)
}

// commitWithMarkerLocked commits a committing journal whose destination
// evidence the recovery gate already proved. Workspace transactions
// record the validated marker ID; the commit entry guards re-check
// every converged sub-state. Callers hold the store mutex.
func (store *Store) commitWithMarkerLocked(materializationID, markerID string) (Journal, error) {
	matDir := filepath.Join(store.root, materializationID)
	raw, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		return Journal{}, err
	}
	view, err := store.readPlanViewLocked(matDir)
	if err != nil {
		return Journal{}, err
	}
	journal, err := decodeJournal(raw, view)
	if err != nil {
		return Journal{}, err
	}
	if journal.Phase != PhaseCommitting {
		return Journal{}, invalid("commit completes from committing, not %s", journal.Phase)
	}
	if journal.Provider != nil && journal.Provider.State != ProviderCommitted {
		return Journal{}, invalid("committed requires the provider transaction committed, got %s", journal.Provider.State)
	}
	if journal.TaskBoard != nil {
		switch journal.TaskBoard.State {
		case BoardResumed, BoardDormantFinalized:
		default:
			return Journal{}, invalid("committed requires the bridge resumed or dormant finalized, got %s", journal.TaskBoard.State)
		}
	}
	for name, state := range journal.AuthorityStates {
		if state.State != AuthorityCommitted {
			return Journal{}, invalid("committed requires authority %q committed, got %s", name, state.State)
		}
	}
	// The marker ID binds from the validated probe: recovery proved
	// the exact destination marker before reaching this write.
	if journal.ManagedReplicaID != nil {
		if markerID == "" {
			return Journal{}, invalid("committing workspace journal has no destination marker to complete with")
		}
		digest, err := scalar.ParseDigest(markerID)
		if err != nil {
			return Journal{}, invalid("destination marker id: %v", err)
		}
		recorded := digest.String()
		journal.DestinationMarkerID = &recorded
	} else if markerID != "" {
		return Journal{}, invalid("committing non-workspace journal carries marker evidence")
	}
	journal.Phase = PhaseCommitted
	updated, err := checkTimestamp(store.now(), "updated_at")
	if err != nil {
		return Journal{}, err
	}
	journal.UpdatedAt = updated
	if err := checkJournalGrammar(&journal); err != nil {
		return Journal{}, err
	}
	if err := checkJournalCrossReferences(&journal, view); err != nil {
		return Journal{}, err
	}
	next, err := encodeJournal(&journal)
	if err != nil {
		return Journal{}, err
	}
	if store.BeforeWrite != nil {
		if err := store.BeforeWrite(); err != nil {
			return Journal{}, err
		}
	}
	if err := replaceAtomic(filepath.Join(matDir, "journal.json"), next); err != nil {
		return Journal{}, err
	}
	if err := syncDirectory(matDir); err != nil {
		return Journal{}, err
	}
	if store.AfterJournal != nil {
		if err := store.AfterJournal(); err != nil {
			return Journal{}, err
		}
	}
	if store.AfterCommit != nil {
		if err := store.AfterCommit(); err != nil {
			return Journal{}, err
		}
	}
	return journal, nil
}
