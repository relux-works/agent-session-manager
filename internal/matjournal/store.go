package matjournal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Store durably installs Materialization Journals with their plan
// routing views and prepare receipts under one data root. Journals are
// mutable: creation installs no-replace, and every later phase or
// sub-state move replaces atomically, so a crash leaves either nothing
// or complete re-validating bytes.
//
// BeforeWrite fires after validation and before the first durable byte
// of an operation; AfterJournal fires after each durable journal write
// inside the operation; AfterCommit fires after the operation's last
// durable write. All three are nil in normal operation. Tests connect
// them to a secconftest Injector firing the owner's own points (a fault
// before any durable byte carries safe_retry) or to plain faults for
// the interior seams.
type Store struct {
	root         string
	mutex        sync.Mutex
	BeforeWrite  func() error
	AfterJournal func() error
	AfterCommit  func() error
	// Now reports the instant stamped as started_at/updated_at. Nil
	// reads the wall clock in UTC; tests inject a fixed clock.
	Now func() time.Time
}

// Open binds a journal store to the materializations namespace beneath
// dataRoot, creating it owner-only when absent. The root must be
// absolute.
func Open(dataRoot string) (*Store, error) {
	if !filepath.IsAbs(dataRoot) {
		return nil, fmt.Errorf("%w: data root %q is not absolute", ErrInvalidJournal, dataRoot)
	}
	root := filepath.Join(dataRoot, "materializations")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create materializations root %q: %w", root, err)
	}
	return &Store{root: root}, nil
}

// JournalRef names one durable journal: its materialization identity
// with the prepare operation that created it and its current phase.
type JournalRef struct {
	MaterializationID  string
	PrepareOperationID string
	Phase              string
}

// TransitionOpts carries phase-entry evidence: the destination marker
// (with its prior chain link) for a workspace commit, and the
// explaining Structured Error for a failure.
type TransitionOpts struct {
	Marker      []byte
	PriorMarker []byte
	LastError   json.RawMessage
}

// prepareReceipt is the machine-local prepare record: the caller-stable
// prepare identity with the journal it created and the canonical
// request bytes that make a retry comparable. It adopts the sessckpt
// operation-receipt discipline (operation key, input digest, no-replace
// install after the bytes it names) and is a distinct record because
// it binds the materialize.prepare namespace rather than a checkpoint
// capture; see TRACEABILITY.md.
type prepareReceipt struct {
	OperationID       string `json:"operation_id"`
	MaterializationID string `json:"materialization_id"`
	InputDigest       string `json:"input_digest"`
	CanonicalRequest  string `json:"canonical_request"`
	Phase             string `json:"phase"`
}

func (store *Store) now() string {
	now := time.Now().UTC()
	if store.Now != nil {
		now = store.Now().UTC()
	}
	return now.Format("2006-01-02T15:04:05.000Z07:00")
}

// Create binds the prepare closure durably: both caller-stable IDs and
// the canonical request digest install before any staging authority or
// bridge mutation. The same IDs retried with a byte-identical body
// replay the recorded journal without writing; the same IDs retried
// with a moved body refuse ErrJournalConflict and write nothing.
func (store *Store) Create(inputs CreateInputs) (JournalRef, []byte, error) {
	started, err := checkTimestamp(store.now(), "started_at")
	if err != nil {
		return JournalRef{}, nil, err
	}
	journal, view, digest, err := buildCreate(inputs, started)
	if err != nil {
		return JournalRef{}, nil, err
	}
	canonical, err := canonicalRequestBody(inputs.RequestBody)
	if err != nil {
		return JournalRef{}, nil, err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if recorded, raw, ok, err := store.replayCreateLocked(journal.MaterializationID, journal.PrepareOperationID, digest, canonical); err != nil {
		return JournalRef{}, nil, err
	} else if ok {
		return recorded, raw, nil
	}
	if store.BeforeWrite != nil {
		if err := store.BeforeWrite(); err != nil {
			return JournalRef{}, nil, err
		}
	}
	matDir := filepath.Join(store.root, journal.MaterializationID)
	if err := os.MkdirAll(matDir, 0o700); err != nil {
		return JournalRef{}, nil, fmt.Errorf("create materialization directory: %w", err)
	}
	viewBytes, err := json.Marshal(view)
	if err != nil {
		return JournalRef{}, nil, fmt.Errorf("encode plan view: %w", err)
	}
	if err := installOrCompare(filepath.Join(matDir, "plan.json"), viewBytes); err != nil {
		return JournalRef{}, nil, err
	}
	raw, err := encodeJournal(&journal)
	if err != nil {
		return JournalRef{}, nil, err
	}
	if err := installOrCompare(filepath.Join(matDir, "journal.json"), raw); err != nil {
		return JournalRef{}, nil, err
	}
	if err := syncDirectory(matDir); err != nil {
		return JournalRef{}, nil, err
	}
	if store.AfterJournal != nil {
		if err := store.AfterJournal(); err != nil {
			return JournalRef{}, nil, err
		}
	}
	receipt := prepareReceipt{
		OperationID:       journal.PrepareOperationID,
		MaterializationID: journal.MaterializationID,
		InputDigest:       digest,
		CanonicalRequest:  string(canonical),
		Phase:             journal.Phase,
	}
	if err := store.writeReceiptLocked(receipt); err != nil {
		return JournalRef{}, nil, err
	}
	if store.AfterCommit != nil {
		if err := store.AfterCommit(); err != nil {
			return JournalRef{}, nil, err
		}
	}
	stored, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		return JournalRef{}, nil, err
	}
	return JournalRef{
		MaterializationID:  journal.MaterializationID,
		PrepareOperationID: journal.PrepareOperationID,
		Phase:              journal.Phase,
	}, stored, nil
}

// replayCreateLocked replays the recorded journal when the same prepare
// retried byte-identical inputs, and refuses moved bodies. Callers hold
// the store mutex.
func (store *Store) replayCreateLocked(materialization, prepare, digest string, canonical []byte) (JournalRef, []byte, bool, error) {
	matDir := filepath.Join(store.root, materialization)
	raw, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return JournalRef{}, nil, false, nil
		}
		return JournalRef{}, nil, false, err
	}
	view, err := store.readPlanViewLocked(matDir)
	if err != nil {
		return JournalRef{}, nil, false, err
	}
	journal, err := decodeJournal(raw, view)
	if err != nil {
		return JournalRef{}, nil, false, err
	}
	if journal.PrepareOperationID != prepare {
		return JournalRef{}, nil, false, conflict(
			"materialization %s prepared under operation %s, retry names %s (idempotency_mismatch)",
			materialization, journal.PrepareOperationID, prepare)
	}
	if journal.PrepareRequestDigest != digest {
		return JournalRef{}, nil, false, conflict(
			"prepare %s retried with a moved request body (idempotency_mismatch)", prepare)
	}
	receipt, ok, err := store.readReceiptLocked(matDir, prepare)
	if err != nil {
		return JournalRef{}, nil, false, err
	}
	if ok {
		if receipt.InputDigest != digest || receipt.CanonicalRequest != string(canonical) {
			return JournalRef{}, nil, false, conflict(
				"prepare %s retried with a byte-unequal body (idempotency_mismatch)", prepare)
		}
		if receipt.MaterializationID != materialization {
			return JournalRef{}, nil, false, conflict(
				"prepare receipt binds materialization %s, want %s", receipt.MaterializationID, materialization)
		}
	} else {
		// A crash landed between the journal and the receipt: the
		// digest already matches the durable journal, so complete
		// the receipt and replay.
		completed := prepareReceipt{
			OperationID:       prepare,
			MaterializationID: materialization,
			InputDigest:       digest,
			CanonicalRequest:  string(canonical),
			Phase:             journal.Phase,
		}
		if err := store.writeReceiptLocked(completed); err != nil {
			return JournalRef{}, nil, false, err
		}
	}
	return JournalRef{
		MaterializationID:  materialization,
		PrepareOperationID: prepare,
		Phase:              journal.Phase,
	}, raw, true, nil
}

// Get returns the byte-identical journal for its materialization,
// re-validated through the closed shape on every read.
func (store *Store) Get(materializationID string) (Journal, []byte, error) {
	id, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, nil, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	matDir := filepath.Join(store.root, id.String())
	raw, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Journal{}, nil, fmt.Errorf("%w: %s", ErrUnknownJournal, id.String())
		}
		return Journal{}, nil, err
	}
	view, err := store.readPlanViewLocked(matDir)
	if err != nil {
		return Journal{}, nil, err
	}
	journal, err := decodeJournal(raw, view)
	if err != nil {
		return Journal{}, nil, err
	}
	return journal, raw, nil
}

// update loads, mutates through apply, re-validates, and atomically
// replaces one journal. The mutation stamps updated_at; apply carries
// the phase and sub-state rules.
func (store *Store) update(materializationID string, apply func(*Journal, planView) error) (Journal, error) {
	id, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.updateLocked(id.String(), apply)
}

// updateLocked is the mutex-held core of update. Callers hold the
// store mutex (public entries and Recover).
func (store *Store) updateLocked(materializationID string, apply func(*Journal, planView) error) (Journal, error) {
	matDir := filepath.Join(store.root, materializationID)
	raw, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Journal{}, fmt.Errorf("%w: %s", ErrUnknownJournal, materializationID)
		}
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
	if terminalPhase(journal.Phase) {
		return Journal{}, invalid("journal is terminal in %s", journal.Phase)
	}
	if store.BeforeWrite != nil {
		if err := store.BeforeWrite(); err != nil {
			return Journal{}, err
		}
	}
	if err := apply(&journal, view); err != nil {
		return Journal{}, err
	}
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

// Transition moves one journal along the phase table, enforcing the
// phase-entry guards. A workspace commit carries its destination
// marker evidence; a failure carries its explaining Structured Error.
func (store *Store) Transition(materializationID, to string, opts TransitionOpts) (Journal, error) {
	if !validPhase(to) {
		return Journal{}, invalid("transition target %q is outside the closed phase enum", to)
	}
	id, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.transitionLocked(id.String(), to, opts)
}

// transitionLocked is the mutex-held core of Transition.
func (store *Store) transitionLocked(materializationID, to string, opts TransitionOpts) (Journal, error) {
	return store.updateLocked(materializationID, func(journal *Journal, _ planView) error {
		if !legalTransition(journal.Phase, to) {
			return invalid("transition %s to %s is outside the phase table", journal.Phase, to)
		}
		if err := checkTransitionGuards(journal, to, opts); err != nil {
			return err
		}
		if to == PhaseFailed {
			journal.LastError = append(json.RawMessage(nil), opts.LastError...)
		}
		if to == PhaseCommitted && journal.ManagedReplicaID != nil {
			marker, err := ValidateMarker(opts.Marker)
			if err != nil {
				return err
			}
			journal.DestinationMarkerID = &marker.MarkerID
		}
		journal.Phase = to
		return nil
	})
}

// UpdateProvider records one provider transaction state. The
// transaction, materialization, and plan IDs must equal the journal;
// the per-call operation ID carries grammar only.
func (store *Store) UpdateProvider(materializationID string, provider ProviderTransaction) (Journal, error) {
	id, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.updateProviderLocked(id.String(), provider)
}

// updateProviderLocked is the mutex-held core of UpdateProvider.
func (store *Store) updateProviderLocked(materializationID string, provider ProviderTransaction) (Journal, error) {
	return store.updateLocked(materializationID, func(journal *Journal, _ planView) error {
		if err := checkProviderTransaction(&provider, journal); err != nil {
			return err
		}
		if journal.Provider != nil && provider.TransactionID != journal.Provider.TransactionID {
			return invalid("provider transaction_id drifts from the recorded transaction")
		}
		journal.Provider = &provider
		return nil
	})
}

// UpdateTaskBoard records one task-board transaction state along the
// sub-state table. The four operation IDs must equal the
// prepare-bound keys; drift refuses. Moving opened to dormant_finalized
// binds cleanup_after to the consumed open expiry.
func (store *Store) UpdateTaskBoard(materializationID string, board TaskBoardTransaction) (Journal, error) {
	id, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.updateTaskBoardLocked(id.String(), board)
}

// updateTaskBoardLocked is the mutex-held core of UpdateTaskBoard.
func (store *Store) updateTaskBoardLocked(materializationID string, board TaskBoardTransaction) (Journal, error) {
	return store.updateLocked(materializationID, func(journal *Journal, view planView) error {
		if journal.TaskBoard == nil {
			return invalid("journal carries no task-board transaction to update")
		}
		if view.Bridge == nil {
			return invalid("plan view carries no bound bridge keys")
		}
		if err := checkTaskBoardTransaction(&board, view.Bridge); err != nil {
			return err
		}
		from := journal.TaskBoard.State
		if from != board.State && !legalBoardTransition(from, board.State) {
			return invalid("task-board transition %s to %s is outside the sub-state table", from, board.State)
		}
		if board.BundleID != journal.TaskBoard.BundleID {
			return invalid("task-board bundle_id drifts from the recorded bundle")
		}
		if board.ActivationMode != journal.TaskBoard.ActivationMode {
			return invalid("task-board activation_mode drifts from the recorded mode")
		}
		if from == BoardOpened && board.State == BoardDormantFinalized {
			if journal.TaskBoard.OpenExpiresAt == nil || board.CleanupAfter == nil ||
				*board.CleanupAfter != *journal.TaskBoard.OpenExpiresAt {
				return invalid("task-board dormant_finalized cleanup_after must equal the consumed open expiry")
			}
		}
		journal.TaskBoard = &board
		return nil
	})
}

// UpdateAuthority records one authority state along the sub-state
// table. The root path must still equal its plan authority; the
// rollback-root and provider-backup rules re-check on every write.
func (store *Store) UpdateAuthority(materializationID, id string, state AuthorityState) (Journal, error) {
	parsed, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.updateAuthorityLocked(parsed.String(), id, state)
}

// updateAuthorityLocked is the mutex-held core of UpdateAuthority.
func (store *Store) updateAuthorityLocked(materializationID, id string, state AuthorityState) (Journal, error) {
	return store.updateLocked(materializationID, func(journal *Journal, view planView) error {
		authority, ok := view.Authorities[id]
		if !ok {
			return invalid("plan authority %q is unknown", id)
		}
		previous, ok := journal.AuthorityStates[id]
		if !ok {
			return invalid("journal authority states miss plan authority %q", id)
		}
		if state.RootPath != authority.RootPath {
			return invalid("journal authority %q root drifts from plan authority", id)
		}
		if !legalAuthorityTransition(previous.State, state.State) {
			return invalid("authority %q transition %s to %s is outside the sub-state table",
				id, previous.State, state.State)
		}
		if err := checkAuthorityState(id, &state); err != nil {
			return err
		}
		if authority.Kind == PlanKindProviderStore {
			if err := checkProviderRollbackRoot(id, &state, journal.Provider); err != nil {
				return err
			}
		}
		journal.AuthorityStates[id] = state
		return nil
	})
}

// RecordProgress merges per-blob chunk completions and whole-blob
// verifications into a staging or validating journal. Chunks merge by
// union per blob (MJ-MULTIBLOB-POS: each blob's set is independent);
// verifications merge by union. Bounds re-check on every write.
func (store *Store) RecordProgress(materializationID string, chunks map[string][]uint32, verified []string) (Journal, error) {
	parsed, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.updateLocked(parsed.String(), func(journal *Journal, _ planView) error {
		if journal.Phase != PhaseStaging && journal.Phase != PhaseValidating {
			return invalid("progress records in staging or validating, not %s", journal.Phase)
		}
		merged := make(map[string][]uint32, len(journal.CompletedBlobChunks)+len(chunks))
		for id, indexes := range journal.CompletedBlobChunks {
			merged[id] = append([]uint32(nil), indexes...)
		}
		for id, indexes := range chunks {
			if _, err := scalar.ParseDigest(id); err != nil {
				return invalid("completed_blob_chunks key %q: %v", id, err)
			}
			merged[id] = unionUint32(merged[id], indexes)
		}
		if err := checkChunkMap(merged); err != nil {
			return err
		}
		union := append([]string{}, journal.VerifiedBlobIDs...)
		union = append(union, verified...)
		sort.Strings(union)
		deduped := []string{}
		for _, value := range union {
			if len(deduped) == 0 || deduped[len(deduped)-1] != value {
				deduped = append(deduped, value)
			}
		}
		checked, err := checkSortedUniqueDigests(deduped, maxVerifiedBlobs, "verified_blob_ids")
		if err != nil {
			return err
		}
		journal.CompletedBlobChunks = merged
		journal.VerifiedBlobIDs = checked
		return nil
	})
}

// RecordError records the redacted Structured Error without moving the
// phase. Tokens never reach this member: callers build the failure
// from fixed templates and IDs.
func (store *Store) RecordError(materializationID string, failure *axerror.Error) (Journal, error) {
	if failure == nil {
		return Journal{}, invalid("recorded last_error is nil")
	}
	framed, err := failure.MarshalJSON()
	if err != nil {
		return Journal{}, invalid("encode last_error: %v", err)
	}
	parsed, err := scalar.ParseUUIDv7(materializationID)
	if err != nil {
		return Journal{}, invalid("materialization id %q: %v", materializationID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.recordErrorLocked(parsed.String(), framed)
}

// recordErrorLocked is the mutex-held core of RecordError: it stores
// already framed Structured Error bytes.
func (store *Store) recordErrorLocked(materializationID string, framed json.RawMessage) (Journal, error) {
	return store.updateLocked(materializationID, func(journal *Journal, _ planView) error {
		journal.LastError = framed
		return nil
	})
}

// Rollback executes the allowed abort: the journal moves to rolling_back
// durably first (the MJ-CRASH-ROLLBACK-LOST seam), then converges the
// provider, bridge, and authority states and closes in rolled_back. It
// refuses past activation and without the explaining failure.
func (store *Store) Rollback(materializationID string, failure *axerror.Error) (Journal, error) {
	if failure == nil {
		return Journal{}, invalid("rollback requires the explaining failure")
	}
	framed, err := failure.MarshalJSON()
	if err != nil {
		return Journal{}, invalid("encode rollback failure: %v", err)
	}
	id, err := checkUUIDv7(materializationID, "materialization_id")
	if err != nil {
		return Journal{}, err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.rollbackLocked(id, framed)
}

// rollbackLocked is the mutex-held core of Rollback: the journal moves
// to rolling_back durably first, then converges and closes in
// rolled_back with the framed terminal failure. Starting from an
// already rolling_back journal retries the rollback instead of
// refusing the self-edge.
func (store *Store) rollbackLocked(materializationID string, framed json.RawMessage) (Journal, error) {
	current, _, err := store.readLocked(materializationID)
	if err != nil {
		return Journal{}, err
	}
	if current.Phase != PhaseRollingBack {
		if _, err := store.transitionLocked(materializationID, PhaseRollingBack, TransitionOpts{}); err != nil {
			return Journal{}, err
		}
	}
	return store.updateLocked(materializationID, func(journal *Journal, _ planView) error {
		if journal.Phase != PhaseRollingBack {
			return invalid("rollback converges from rolling_back, not %s", journal.Phase)
		}
		if err := checkPreActivation(journal); err != nil {
			return err
		}
		if journal.Provider != nil {
			provider := *journal.Provider
			provider.State = ProviderRolledBack
			provider.RollbackToken = nil
			status, err := checkTimestamp(store.now(), "provider last_status_at")
			if err != nil {
				return err
			}
			provider.LastStatusAt = status
			if err := checkProviderTransaction(&provider, journal); err != nil {
				return err
			}
			journal.Provider = &provider
		}
		if journal.TaskBoard != nil {
			board := *journal.TaskBoard
			board.State = BoardRolledBack
			board.ImportToken, board.StagedManagerRef, board.ImportExpiresAt = nil, nil, nil
			board.OpenToken, board.DormantManagerRef, board.OpenExpiresAt = nil, nil, nil
			board.ManagerSessionRef, board.AxBinding = nil, nil
			board.CleanupState = "removed"
			board.CleanupAfter = nil
			if err := checkTaskBoardTransaction(&board, nil); err != nil {
				return err
			}
			journal.TaskBoard = &board
		}
		for name, state := range journal.AuthorityStates {
			state.State = AuthorityRolledBack
			state.RollbackRoot = nil
			if err := checkAuthorityState(name, &state); err != nil {
				return err
			}
			journal.AuthorityStates[name] = state
		}
		journal.LastError = append(json.RawMessage(nil), framed...)
		journal.Phase = PhaseRolledBack
		return nil
	})
}

// unionUint32 merges two index sets into a sorted-unique union.
func unionUint32(into, add []uint32) []uint32 {
	seen := make(map[uint32]struct{}, len(into)+len(add))
	for _, value := range into {
		seen[value] = struct{}{}
	}
	for _, value := range add {
		seen[value] = struct{}{}
	}
	out := make([]uint32, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// canonicalRequestBody renders the canonical prepare request bytes the
// receipt compares for byte equality.
func canonicalRequestBody(body []byte) ([]byte, error) {
	return canonicaljson.Canonicalize(body)
}

// readLocked loads and validates one journal with its plan view.
// Callers hold the store mutex.
func (store *Store) readLocked(materializationID string) (Journal, planView, error) {
	matDir := filepath.Join(store.root, materializationID)
	raw, err := readJournalFile(filepath.Join(matDir, "journal.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Journal{}, planView{}, fmt.Errorf("%w: %s", ErrUnknownJournal, materializationID)
		}
		return Journal{}, planView{}, err
	}
	view, err := store.readPlanViewLocked(matDir)
	if err != nil {
		return Journal{}, planView{}, err
	}
	journal, err := decodeJournal(raw, view)
	if err != nil {
		return Journal{}, planView{}, err
	}
	return journal, view, nil
}

// readPlanViewLocked reads the installed plan routing view. Callers hold
// the store mutex. A missing or malformed view is torn machine-local
// state: operational, never an invalid journal shape.
func (store *Store) readPlanViewLocked(matDir string) (planView, error) {
	raw, err := os.ReadFile(filepath.Join(matDir, "plan.json"))
	if err != nil {
		return planView{}, fmt.Errorf("read plan view: %w", err)
	}
	var view planView
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&view); err != nil {
		return planView{}, fmt.Errorf("decode plan view: %w", err)
	}
	return view, nil
}

// writeReceiptLocked installs one prepare receipt no-replace: the first
// committer wins and every later retry replays or refuses against it.
// Callers hold the store mutex.
func (store *Store) writeReceiptLocked(receipt prepareReceipt) error {
	matDir := filepath.Join(store.root, receipt.MaterializationID)
	operations := filepath.Join(matDir, "operations")
	if err := os.MkdirAll(operations, 0o700); err != nil {
		return fmt.Errorf("create receipt directory: %w", err)
	}
	framed, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode prepare receipt: %w", err)
	}
	if err := writeExclusive(filepath.Join(operations, receipt.OperationID+".json"), framed); err != nil {
		return err
	}
	return syncDirectory(operations)
}

// readReceiptLocked returns the committed prepare receipt for one
// operation, validated by strict decode plus grammar on every read.
// Callers hold the store mutex.
func (store *Store) readReceiptLocked(matDir, operation string) (prepareReceipt, bool, error) {
	framed, err := os.ReadFile(filepath.Join(matDir, "operations", operation+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return prepareReceipt{}, false, nil
		}
		return prepareReceipt{}, false, err
	}
	var receipt prepareReceipt
	decoder := json.NewDecoder(bytes.NewReader(framed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return prepareReceipt{}, false, fmt.Errorf("decode prepare receipt: %w", err)
	}
	if _, err := scalar.ParseUUIDv7(receipt.OperationID); err != nil {
		return prepareReceipt{}, false, fmt.Errorf("decode prepare receipt operation_id: %v", err)
	}
	if _, err := scalar.ParseUUIDv7(receipt.MaterializationID); err != nil {
		return prepareReceipt{}, false, fmt.Errorf("decode prepare receipt materialization_id: %v", err)
	}
	if _, err := scalar.ParseDigest(receipt.InputDigest); err != nil {
		return prepareReceipt{}, false, fmt.Errorf("decode prepare receipt input_digest: %v", err)
	}
	if _, fault := environ.DecodeStrictObject([]byte(receipt.CanonicalRequest)); fault != nil {
		return prepareReceipt{}, false, fmt.Errorf("decode prepare receipt canonical_request: %v", fault)
	}
	if receipt.OperationID != operation {
		return prepareReceipt{}, false, fmt.Errorf("prepare receipt names %s, want %s", receipt.OperationID, operation)
	}
	return receipt, true, nil
}

// readJournalFile reads one installed file verbatim.
func readJournalFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes.Clone(raw), nil
}

// installOrCompare installs immutable sidecar bytes no-replace: a fresh
// path is created atomically, and only an EEXIST loser falls back to
// comparing against the winner's bytes — identical bytes are reused,
// disagreeing bytes report the torn store for refusal.
func installOrCompare(path string, data []byte) error {
	if err := writeExclusive(path, data); err == nil {
		return nil
	} else if !os.IsExist(err) {
		return fmt.Errorf("install journal sidecar: %w", err)
	}
	existing, readErr := os.ReadFile(path)
	if readErr != nil {
		return fmt.Errorf("read journal sidecar for compare: %w", readErr)
	}
	if string(existing) != string(data) {
		return fmt.Errorf("%w: sidecar path holds disagreeing bytes", ErrJournalConflict)
	}
	return nil
}

// writeExclusive installs bytes at a path that must not exist. Bytes
// fsync before close so a crash never leaves a torn prefix behind.
func writeExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write exclusive file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("fsync exclusive file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close exclusive file: %w", err)
	}
	return nil
}

// replaceAtomic replaces the mutable journal through a same-directory
// temporary file, fsync, atomic rename, and directory sync: readers
// see the old or the new document, never a mix.
func replaceAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".journal-*")
	if err != nil {
		return fmt.Errorf("create journal temporary file: %w", err)
	}
	name := temporary.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write journal temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("fsync journal temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close journal temporary file: %w", err)
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return fmt.Errorf("protect journal temporary file: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace journal: %w", err)
	}
	return nil
}

// syncDirectory fsyncs a directory so an install inside it survives a
// crash. Opening a directory fails on Windows, where the install
// itself is the durability boundary; there the sync is skipped, never
// faked.
func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory for fsync: %w", err)
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("fsync directory: %w", err)
	}
	return nil
}
