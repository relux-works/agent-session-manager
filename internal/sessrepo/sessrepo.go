package sessrepo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// uint53Max is the AX safe-integer ceiling from SPEC.md Section 1.6. It is
// passed to the environ bound checker, never re-enforced here.
const uint53Max = uint64(1<<53 - 1)

const (
	sessionRecordSchema = "urn:ax:schema:session-record"
	sessionEventSchema  = "urn:ax:schema:session-event"
)

var (
	ErrInvalidRecord   = errors.New("invalid session record")
	ErrSessionExists   = errors.New("session already exists")
	ErrUnknownSession  = errors.New("unknown session")
	ErrInvalidEvent    = errors.New("invalid session event")
	ErrSequenceGap     = errors.New("event source sequence gap")
	ErrSequenceRepeat  = errors.New("event source sequence repeat")
	ErrPredecessorLink = errors.New("event predecessor link mismatch")
	ErrDivergentBranch = errors.New("divergent lease branch preserved without application")
	ErrStaleLease      = errors.New("event lease epoch precedes the chain head")
	ErrChainCorrupt    = errors.New("session event chain is corrupt")
	ErrUnknownEvent    = errors.New("unknown session event")
	ErrNameAmbiguous   = errors.New("name_ambiguous")
	ErrNameNotFound    = errors.New("session name not found")
	ErrRepositoryPath  = errors.New("invalid session repository path")
)

// refuse is the single refusal funnel every sessrepo rejection flows
// through. It is a var so the census TestMain can wrap it and record the
// production file:line behind each exercised refusal. No production path
// wraps a sentinel with fmt.Errorf directly.
var refuse = func(sentinel error, format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, arguments...))
}

// SessionRef names one persisted session: its Session Record digest and ID.
type SessionRef struct {
	SessionID string
	RecordID  string
}

// EventRef names one chained event: its digest, lease sequence, and position.
type EventRef struct {
	SessionID     string
	EventID       string
	LeaseSequence uint64
	Position      int
}

// EventSummary is the chain-index entry for one event, derived from the
// validated event bytes at append time and re-verified against them on load.
type EventSummary struct {
	EventID       string   `json:"event_id"`
	EventType     string   `json:"event_type"`
	LeaseEpoch    uint64   `json:"lease_epoch"`
	LeaseID       string   `json:"lease_id"`
	LeaseSequence uint64   `json:"lease_sequence"`
	Predecessors  []string `json:"predecessors"`
}

// SessionSummary is the Section 14.4 list projection this leaf owns: stored
// identity plus chain-head facts for a healthy session, or the per-session
// parked channel for an interrupted create or a torn store. Owner host,
// checkpoint age, capability status, and rendered warnings are derived
// downstream, never here; the parked fields below are the facts the
// Section 5.7 reducer and the Section 14.4 rendering leaves project from.
//
// A healthy entry carries Parked=false with empty BlockingReason and
// RetryHint. A parked entry carries Parked=true, SessionID set to the
// session directory name, BlockingReason naming the load failure (a
// recoverable_parked_state block with its per-session cause), and RetryHint
// naming the same-operation retry. Identity members (RecordID, Name, Kind,
// ProviderID) are filled from the parked record when it still verifies;
// a bare directory with no verifiable record leaves them empty.
type SessionSummary struct {
	SessionID      string
	RecordID       string
	Name           string
	Kind           string
	ProviderID     string
	EventCount     int
	TailEvent      string
	Parked         bool
	BlockingReason string
	RetryHint      string
}

// retryHintForParkedRecord names the same-operation retry for a parked
// session: a record is already parked, so only the byte-identical record
// resumes through CreateSession.
const retryHintForParkedRecord = "retry CreateSession with the byte-identical session record"

// retryHintForBareDirectory names the same-operation retry for a bare
// session directory: no record is parked yet, so CreateSession with the
// session record for this session ID resumes.
const retryHintForBareDirectory = "retry CreateSession with the session record for this session ID"

// CreateStep names one durable boundary inside CreateSession's four-step
// span: the session directory, the record install, the events directory,
// and the chain index install, in that order.
type CreateStep string

const (
	CreateStepSessionDir CreateStep = "session-dir"
	CreateStepRecord     CreateStep = "record"
	CreateStepEventsDir  CreateStep = "events-dir"
	CreateStepChain      CreateStep = "chain"
)

// Repository persists Session Records and per-session event chains under
// one data root. All mutating entries serialize on one mutex; files are
// content-addressed and installed no-replace, so the chain is append-only
// by construction and there is no update or delete entry to test.
//
// BeforeWrite fires after validation and before the first durable byte;
// AfterCommit fires after the chain index is durable. AfterCreateStep
// fires after each CreateStep inside CreateSession's four-step span, so
// an interrupted create — not only a fault outside it — is drivable.
// All three are nil in normal operation. Tests connect them to a
// secconftest Injector firing the owner's own points: a fault before
// any durable byte carries safe_retry, and a fault past any durable
// step — including every interior boundary, whose parked state the
// identical retry resumes — carries recoverable_parked_state.
type Repository struct {
	root            string
	mutex           sync.Mutex
	BeforeWrite     func() error
	AfterCommit     func() error
	AfterCreateStep func(CreateStep) error
	// AfterLeaseStage fires after a lease blob stages and fsyncs and
	// before the install rename. It is nil in normal operation; crash
	// tests drive the rename seam through it.
	AfterLeaseStage func() error
}

// Open binds a repository to the sessions namespace beneath dataRoot,
// creating it owner-only when absent. The root must be absolute.
func Open(dataRoot string) (*Repository, error) {
	if !filepath.IsAbs(dataRoot) {
		return nil, refuse(ErrRepositoryPath, "data root %q is not absolute", dataRoot)
	}
	root := filepath.Join(dataRoot, "sessions")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, refuse(ErrRepositoryPath, "create sessions root %q: %v", root, err)
	}
	return &Repository{root: root}, nil
}

// CreateSession validates recordJSON through the canonical owner and
// persists it as a new session. The bytes are stored verbatim.
func (repository *Repository) CreateSession(recordJSON []byte) (SessionRef, error) {
	record, err := decodeSessionRecord(recordJSON)
	if err != nil {
		return SessionRef{}, err
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if repository.BeforeWrite != nil {
		if err := repository.BeforeWrite(); err != nil {
			return SessionRef{}, err
		}
	}
	directory := repository.sessionDir(record.sessionID)
	if _, err := os.Stat(directory); err == nil {
		resumed, resumeErr := repository.resumeCreateLocked(directory, record, recordJSON)
		if resumeErr != nil {
			return SessionRef{}, resumeErr
		}
		if !resumed {
			return SessionRef{}, refuse(ErrSessionExists, "session %s already exists", record.sessionID)
		}
	} else {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return SessionRef{}, fmt.Errorf("create session directory: %w", err)
		}
		if err := repository.crashCreateStep(CreateStepSessionDir); err != nil {
			return SessionRef{}, err
		}
		if err := writeExclusive(filepath.Join(directory, "record.json"), recordJSON); err != nil {
			return SessionRef{}, fmt.Errorf("install session record: %w", err)
		}
		if err := repository.crashCreateStep(CreateStepRecord); err != nil {
			return SessionRef{}, err
		}
		if err := os.MkdirAll(filepath.Join(directory, "events"), 0o700); err != nil {
			return SessionRef{}, fmt.Errorf("create session events directory: %w", err)
		}
		if err := repository.crashCreateStep(CreateStepEventsDir); err != nil {
			return SessionRef{}, err
		}
		if err := repository.writeChain(directory, storedChain{RecordID: record.recordID}); err != nil {
			return SessionRef{}, err
		}
		if err := repository.crashCreateStep(CreateStepChain); err != nil {
			return SessionRef{}, err
		}
	}
	if repository.AfterCommit != nil {
		if err := repository.AfterCommit(); err != nil {
			return SessionRef{}, err
		}
	}
	return SessionRef{SessionID: record.sessionID, RecordID: record.recordID}, nil
}

// crashCreateStep fires the AfterCreateStep hook after one durable create
// step. It is nil-safe: unhooked production never faults. A hook error —
// the injected secconftest fault in tests — propagates unwrapped so the
// caller observes the owner's point and outcome.
func (repository *Repository) crashCreateStep(step CreateStep) error {
	if repository.AfterCreateStep == nil {
		return nil
	}
	return repository.AfterCreateStep(step)
}

// resumeCreateLocked completes a create that crashed inside the four-step
// span. Two parked states resume:
//
//   - record absent: the crash landed after the session directory and
//     before the record install, so nothing committed. The chain index
//     must also be absent — a chain without a record is torn foreign
//     state, never a resume — and the retry's bytes become the record.
//   - record present with the retried bytes and the chain still absent:
//     the crash landed between the record write and the chain write, and
//     the retry completes the index.
//
// Anything else is not a resume — a completed create, or a colliding
// record under a live session ID — and the caller refuses it as an
// existing session. It returns plain operational errors; refusal
// attribution stays at the caller.
func (repository *Repository) resumeCreateLocked(directory string, record sessionRecord, recordJSON []byte) (bool, error) {
	existing, err := os.ReadFile(filepath.Join(directory, "record.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("read session record for resume: %w", err)
		}
		if _, statErr := os.Stat(filepath.Join(directory, "chain.json")); statErr == nil {
			return false, nil
		}
		if err := writeExclusive(filepath.Join(directory, "record.json"), recordJSON); err != nil {
			return false, fmt.Errorf("install session record on resume: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(directory, "events"), 0o700); err != nil {
			return false, fmt.Errorf("recreate session events directory: %w", err)
		}
		if err := repository.writeChain(directory, storedChain{RecordID: record.recordID}); err != nil {
			return false, err
		}
		return true, nil
	}
	if !bytes.Equal(existing, recordJSON) {
		return false, nil
	}
	if _, err := os.Stat(filepath.Join(directory, "chain.json")); err == nil {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Join(directory, "events"), 0o700); err != nil {
		return false, fmt.Errorf("recreate session events directory: %w", err)
	}
	if err := repository.writeChain(directory, storedChain{RecordID: record.recordID}); err != nil {
		return false, err
	}
	return true, nil
}

// AppendEvent validates eventJSON through the canonical owner and links it
// onto the session chain after the continuity checks. A byte-identical
// retry of an already chained event returns its existing reference without
// duplicating the chain, so a retry after a crashed commit is safe.
func (repository *Repository) AppendEvent(sessionID string, eventJSON []byte) (EventRef, error) {
	event, err := decodeSessionEvent(eventJSON)
	if err != nil {
		return EventRef{}, err
	}
	if event.sessionID != sessionID {
		return EventRef{}, refuse(ErrInvalidEvent, "event session %s does not match target session %s", event.sessionID, sessionID)
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return EventRef{}, err
	}
	for position, existing := range view.chain.Events {
		if existing.EventID == event.eventID {
			return EventRef{SessionID: sessionID, EventID: event.eventID, LeaseSequence: existing.LeaseSequence, Position: position}, nil
		}
	}
	if repository.BeforeWrite != nil {
		if err := repository.BeforeWrite(); err != nil {
			return EventRef{}, err
		}
	}
	var tail *EventSummary
	if len(view.chain.Events) > 0 {
		previous := view.chain.Events[len(view.chain.Events)-1]
		tail = &previous
	}
	if err := checkAppend(view.chain.RecordID, tail, event); err != nil {
		// Losing-lease branches are preserved as immutable blobs but never
		// applied: the bytes stay addressable while the chain is untouched.
		if errors.Is(err, ErrDivergentBranch) || errors.Is(err, ErrStaleLease) {
			if preserveErr := installEventBlob(filepath.Join(view.directory, "events", blobFileName(event.eventID)), eventJSON); preserveErr != nil {
				return EventRef{}, fmt.Errorf("preserve unapplied branch: %v: %w", preserveErr, err)
			}
		}
		return EventRef{}, err
	}
	blobPath := filepath.Join(view.directory, "events", blobFileName(event.eventID))
	if err := installEventBlob(blobPath, eventJSON); err != nil {
		if errors.Is(err, errDigestDisagreement) {
			return EventRef{}, refuse(ErrChainCorrupt, "digest path holds disagreeing bytes for event")
		}
		return EventRef{}, err
	}
	next := append(append([]EventSummary(nil), view.chain.Events...), eventSummary(event))
	if err := repository.writeChain(view.directory, storedChain{RecordID: view.chain.RecordID, Events: next}); err != nil {
		return EventRef{}, err
	}
	if repository.AfterCommit != nil {
		if err := repository.AfterCommit(); err != nil {
			return EventRef{}, err
		}
	}
	return EventRef{SessionID: sessionID, EventID: event.eventID, LeaseSequence: event.leaseSeq, Position: len(next) - 1}, nil
}

// eventSummary projects the validated routing members into the chain index.
func eventSummary(event eventView) EventSummary {
	return EventSummary{EventID: event.eventID, EventType: event.eventType, LeaseEpoch: event.leaseEpoch, LeaseID: event.leaseID, LeaseSequence: event.leaseSeq, Predecessors: append([]string(nil), event.predecessors...)}
}
