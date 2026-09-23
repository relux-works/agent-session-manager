package sessprofile

import (
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
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
	// Ckpt supplies the winning lease's captured handoff closure when the
	// winner has one. Without it, prior-lease profile sources cannot be
	// established and derivation refuses those sources closed.
	Ckpt *sessckpt.Store
}

// Project derives the session-head effective pair for one session
// ID. A parked session refuses with the blocking reason and retry,
// never with an unknown; an ID with no listing entry refuses
// unknown.
func (projector *Projector) Project(sessionID string) (Pair, error) {
	derivation, err := projector.load(sessionID)
	if err != nil {
		return Pair{}, err
	}
	return derivation.Derive()
}

// ProjectForHeads derives the effective pair over the transitive
// predecessor closure of heads: the Section 5.4 event-head closure
// of one checkpoint. Unknown heads and dangling predecessors
// refuse integrity_failure; empty heads carry no closure.
func (projector *Projector) ProjectForHeads(sessionID string, heads []string) (Pair, error) {
	derivation, err := projector.load(sessionID)
	if err != nil {
		return Pair{}, err
	}
	return derivation.DeriveForHeads(heads)
}

// load locates the session through the listing (distinguishing a
// parked session from an unknown one with no sessrepo interface
// change) and decodes the stored record with the authoritative
// chain in index order.
func (projector *Projector) load(sessionID string) (Derivation, error) {
	if projector == nil || projector.Repo == nil {
		return Derivation{}, refuse(ErrDerivation, "projector carries no repository")
	}
	summaries, err := projector.Repo.ListSessions()
	if err != nil {
		return Derivation{}, err
	}
	var found *sessrepo.SessionSummary
	for index := range summaries {
		if summaries[index].SessionID == sessionID {
			found = &summaries[index]
			break
		}
	}
	if found == nil {
		return Derivation{}, refuse(ErrUnknownSession, "no session %s", sessionID)
	}
	if found.Parked {
		return Derivation{}, refuse(ErrSessionParked, "session %s: %s (%s)", sessionID, found.BlockingReason, found.RetryHint)
	}
	recordBytes, err := projector.Repo.GetRecord(sessionID)
	if err != nil {
		return Derivation{}, err
	}
	record, err := DecodeRecord(recordBytes)
	if err != nil {
		return Derivation{}, err
	}
	indexed, err := projector.Repo.ListEvents(sessionID)
	if err != nil {
		return Derivation{}, err
	}
	events := make([]Event, 0, len(indexed))
	for _, summary := range indexed {
		blob, err := projector.Repo.GetEvent(sessionID, summary.EventID)
		if err != nil {
			return Derivation{}, err
		}
		event, err := DecodeEvent(blob)
		if err != nil {
			return Derivation{}, err
		}
		events = append(events, event)
	}
	authority, err := LoadSourceAuthority(projector.Repo, projector.Ckpt, sessionID)
	if err != nil {
		return Derivation{}, err
	}
	return Derivation{Record: record, Events: events, Authority: authority}, nil
}
