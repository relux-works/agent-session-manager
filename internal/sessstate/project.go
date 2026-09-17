package sessstate

import (
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Projector binds the pure reducer to one repository. It only reads:
// ListSessions locates the session (and its parked channel), GetRecord
// and ListEvents with GetEvent load the authoritative bytes, and
// Decode plus Reduce derive the projection. No entry here mutates
// durable state, so there is no crash window to arm: placing
// secconftest points around reads would measure nothing, and a crashed
// read retries as the same read. Idempotency is the reducer's purity —
// projecting twice yields the identical projection — pinned by
// TestProjectIsIdempotentAcrossReads.
type Projector struct {
	// Repo is the session repository to project from. It is required.
	Repo *sessrepo.Repository
	// LocalHostID selects the local role: the owner host of the
	// newest authoritative event decides owner against replica. Empty
	// leaves the role unknown rather than guessing.
	LocalHostID string
	// Union carries competing off-chain leases observed at mesh union
	// for winner resolution. Nil projects the chain alone.
	Union []LeaseHead
	// Local is the lease the local host acts under for the stale
	// projection. Zero disables the stale override.
	Local LeaseHead
}

// Project derives the session projection for one session ID. A parked
// session returns the parked state with its blocking reason and retry,
// never an error or an omission; an ID with no listing entry refuses
// unknown. The listing therefore tells a parked name apart from a
// missing one with no sessrepo interface change.
func (projector *Projector) Project(sessionID string) (Projection, error) {
	if projector == nil || projector.Repo == nil {
		return Projection{}, refuse(ErrDerivation, "projector carries no repository")
	}
	summaries, err := projector.Repo.ListSessions()
	if err != nil {
		return Projection{}, err
	}
	var found *sessrepo.SessionSummary
	for index := range summaries {
		if summaries[index].SessionID == sessionID {
			found = &summaries[index]
			break
		}
	}
	if found == nil {
		return Projection{}, refuse(ErrUnknownSession, "no session %s", sessionID)
	}
	if found.Parked {
		return Reduce(Input{
			Record: Record{
				SessionID:  found.SessionID,
				RecordID:   found.RecordID,
				Name:       found.Name,
				Kind:       found.Kind,
				ProviderID: found.ProviderID,
			},
			Parked: &ParkedFacts{
				BlockingReason: found.BlockingReason,
				RetryHint:      found.RetryHint,
			},
		})
	}
	recordBytes, err := projector.Repo.GetRecord(sessionID)
	if err != nil {
		return Projection{}, err
	}
	record, err := DecodeRecord(recordBytes)
	if err != nil {
		return Projection{}, err
	}
	indexed, err := projector.Repo.ListEvents(sessionID)
	if err != nil {
		return Projection{}, err
	}
	events := make([]Event, 0, len(indexed))
	for _, summary := range indexed {
		blob, err := projector.Repo.GetEvent(sessionID, summary.EventID)
		if err != nil {
			return Projection{}, err
		}
		event, err := DecodeEvent(blob)
		if err != nil {
			return Projection{}, err
		}
		events = append(events, event)
	}
	return Reduce(Input{
		Record:      record,
		Events:      events,
		Union:       append([]LeaseHead(nil), projector.Union...),
		Local:       projector.Local,
		LocalHostID: projector.LocalHostID,
	})
}
