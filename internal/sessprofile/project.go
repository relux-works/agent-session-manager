package sessprofile

import (
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Projector binds the pure reducer to one repository. It only reads:
// ListSessions locates the session (and its parked channel),
// GetRecord and ListEvents with GetEvent load the authoritative
// bytes, and Decode plus Derive derive the pair. No entry here
// mutates durable state, so there is no crash window to arm: a
// crashed read retries as the same read. Idempotency is the
// reducer's purity — projecting twice yields the identical pair.
type Projector struct {
	// Repo is the session repository to project from. It is required.
	Repo *sessrepo.Repository
}

// Project derives the session-head effective pair for one session
// ID. A parked session refuses with the blocking reason and retry,
// never with an unknown; an ID with no listing entry refuses
// unknown.
func (projector *Projector) Project(sessionID string) (Pair, error) {
	record, events, err := projector.load(sessionID)
	if err != nil {
		return Pair{}, err
	}
	return Derive(record, events)
}

// ProjectForHeads derives the effective pair over the transitive
// predecessor closure of heads: the Section 5.4 event-head closure
// of one checkpoint. Unknown heads and dangling predecessors
// refuse integrity_failure; empty heads carry no closure.
func (projector *Projector) ProjectForHeads(sessionID string, heads []string) (Pair, error) {
	record, events, err := projector.load(sessionID)
	if err != nil {
		return Pair{}, err
	}
	return DeriveForHeads(record, events, heads)
}

// load locates the session through the listing (distinguishing a
// parked session from an unknown one with no sessrepo interface
// change) and decodes the stored record with the authoritative
// chain in index order.
func (projector *Projector) load(sessionID string) (Record, []Event, error) {
	if projector == nil || projector.Repo == nil {
		return Record{}, nil, refuse(ErrDerivation, "projector carries no repository")
	}
	summaries, err := projector.Repo.ListSessions()
	if err != nil {
		return Record{}, nil, err
	}
	var found *sessrepo.SessionSummary
	for index := range summaries {
		if summaries[index].SessionID == sessionID {
			found = &summaries[index]
			break
		}
	}
	if found == nil {
		return Record{}, nil, refuse(ErrUnknownSession, "no session %s", sessionID)
	}
	if found.Parked {
		return Record{}, nil, refuse(ErrSessionParked, "session %s: %s (%s)", sessionID, found.BlockingReason, found.RetryHint)
	}
	recordBytes, err := projector.Repo.GetRecord(sessionID)
	if err != nil {
		return Record{}, nil, err
	}
	record, err := DecodeRecord(recordBytes)
	if err != nil {
		return Record{}, nil, err
	}
	indexed, err := projector.Repo.ListEvents(sessionID)
	if err != nil {
		return Record{}, nil, err
	}
	events := make([]Event, 0, len(indexed))
	for _, summary := range indexed {
		blob, err := projector.Repo.GetEvent(sessionID, summary.EventID)
		if err != nil {
			return Record{}, nil, err
		}
		event, err := DecodeEvent(blob)
		if err != nil {
			return Record{}, nil, err
		}
		events = append(events, event)
	}
	return record, events, nil
}
