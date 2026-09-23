package sessprofile

import (
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Transactor appends profile.changed events under the current lease
// through the sessrepo append/attest path. It mints deterministically
// from the request, so a retry after a lost response replays the
// committed event instead of duplicating it.
type Transactor struct {
	// Repo is the session repository the transaction appends to. It
	// is required.
	Repo *sessrepo.Repository
	// Ckpt supplies the winner's captured handoff closure when the
	// effective source was authored by an earlier lease.
	Ckpt *sessckpt.Store
}

// SetProfileRequest is one ax session set-profile operation: the
// session, the requested profile, the operator confirmation (which
// Section 2.4 requires exactly when changing to yolo), the lease
// the operator acts under, and the author with the creation
// instant. The instant is caller input: a retry repeats it
// byte-identically, which is what makes the retry idempotent.
type SetProfileRequest struct {
	SessionID       string
	To              string
	Confirmed       bool
	LeaseEpoch      uint64
	LeaseID         string
	CreatedByHostID string
	CreatedAt       string
}

// SetProfileResult is the committed (or replayed) change: the
// previous and new profiles with the authoritative event carrying
// the change. PreviousProfile is the effective profile before the
// transaction; NewProfile equals the request target.
type SetProfileResult struct {
	SessionID       string
	PreviousProfile string
	NewProfile      string
	EventID         string
	Event           []byte
	Ref             sessrepo.EventRef
}

// SetProfile validates one profile change and appends it as a new
// event under the current lease. The from end is the session-head
// effective profile, never the creation value alone: a second
// change starts where the first landed. Refusals, in order: an
// unknown session or a parked one; a target outside the vocabulary;
// an acting lease that is not the chain-head lease (a stale epoch
// or a divergent token — there is no time-expiring lease in the
// pinned Section 5.3, so a superseded lease is the expired lease);
// an unconfirmed change to yolo; and a change from a profile to
// itself. A retry of an already-committed change replays the
// committed event when the newest authoritative change carries the
// request envelope; anything else from-equal-to refuses
// unchanged. Only the append mutates durable state, so the
// landed sessrepo crash semantics apply unchanged: a fault before
// the durable write admits nothing and the identical retry
// appends, while a fault after it counts as committed and the
// identical retry replays.
func (transactor *Transactor) SetProfile(request SetProfileRequest) (SetProfileResult, error) {
	if transactor == nil || transactor.Repo == nil {
		return SetProfileResult{}, refuse(ErrDerivation, "transactor carries no repository")
	}
	if request.To != ProfileStandard && request.To != ProfileYOLO {
		return SetProfileResult{}, refuse(ErrInvalidProfile, "set-profile target %q is not standard|yolo", request.To)
	}
	derivation, err := transactor.load(request.SessionID)
	if err != nil {
		return SetProfileResult{}, err
	}
	winner, err := transactor.Repo.WinningLease(request.SessionID)
	if err != nil {
		return SetProfileResult{}, err
	}
	if len(derivation.Events) == 0 {
		// With no chain tail there is no link check to reject an
		// unminted greater-epoch request. Revalidate through sessrepo
		// so the empty-chain path still requires the current winner.
		if err := transactor.Repo.VerifyFencingToken(request.SessionID, sessrepo.FencingToken{
			Epoch: request.LeaseEpoch, LeaseID: request.LeaseID, HolderHostID: winner.HolderHostID,
		}); err != nil {
			return SetProfileResult{}, err
		}
	}
	current, err := derivation.Derive()
	if err != nil {
		return SetProfileResult{}, err
	}
	sequence, predecessors, err := currentLeaseLink(derivation.Events, derivation.Record.RecordID, request.LeaseEpoch, request.LeaseID)
	if err != nil {
		return SetProfileResult{}, err
	}
	if current.Profile == request.To {
		return transactor.replayOrRefuse(request, derivation.Record, derivation.Events, current)
	}
	minted, err := MintChangeEvent(ChangeParams{
		SessionID:       request.SessionID,
		CreatedByHostID: request.CreatedByHostID,
		LeaseEpoch:      request.LeaseEpoch,
		LeaseID:         request.LeaseID,
		Sequence:        sequence,
		Predecessors:    predecessors,
		From:            current.Profile,
		To:              request.To,
		Confirmed:       request.Confirmed,
		CreatedAt:       request.CreatedAt,
	})
	if err != nil {
		return SetProfileResult{}, err
	}
	reference, err := transactor.Repo.AppendEvent(request.SessionID, minted)
	if err != nil {
		return SetProfileResult{}, err
	}
	return SetProfileResult{
		SessionID:       request.SessionID,
		PreviousProfile: current.Profile,
		NewProfile:      request.To,
		EventID:         reference.EventID,
		Event:           minted,
		Ref:             reference,
	}, nil
}

// load locates the session through the listing (distinguishing a
// parked session from an unknown one) and decodes the stored record
// with the authoritative chain in index order.
func (transactor *Transactor) load(sessionID string) (Derivation, error) {
	summaries, err := transactor.Repo.ListSessions()
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
	recordBytes, err := transactor.Repo.GetRecord(sessionID)
	if err != nil {
		return Derivation{}, err
	}
	record, err := DecodeRecord(recordBytes)
	if err != nil {
		return Derivation{}, err
	}
	indexed, err := transactor.Repo.ListEvents(sessionID)
	if err != nil {
		return Derivation{}, err
	}
	events := make([]Event, 0, len(indexed))
	for _, summary := range indexed {
		blob, err := transactor.Repo.GetEvent(sessionID, summary.EventID)
		if err != nil {
			return Derivation{}, err
		}
		event, err := DecodeEvent(blob)
		if err != nil {
			return Derivation{}, err
		}
		events = append(events, event)
	}
	authority, err := LoadSourceAuthority(transactor.Repo, transactor.Ckpt, sessionID)
	if err != nil {
		return Derivation{}, err
	}
	return Derivation{Record: record, Events: events, Authority: authority}, nil
}

// currentLeaseLink binds the acting lease to the chain head and
// returns the sequence with the predecessors the new event must
// carry. On an eventless chain the acting lease opens sequence 1
// (the Lease Record binding of that opening lease belongs to the
// ownership-leases story; the caller passes the winning lease).
// On a non-empty chain the acting lease must equal the head
// exactly: set-profile authors under the current lease only, so a
// greater-epoch acting lease refuses alongside the stale and
// divergent ones — lease succession arrives through takeover
// flows, never through a profile event.
func currentLeaseLink(events []Event, recordID string, epoch uint64, leaseID string) (uint64, []string, error) {
	if len(events) == 0 {
		return 1, []string{recordID}, nil
	}
	tail := events[len(events)-1]
	acting := LeaseHead{Epoch: epoch, LeaseID: leaseID}
	head := LeaseHead{Epoch: tail.LeaseEpoch, LeaseID: tail.LeaseID}
	switch compareLease(acting, head) {
	case 0:
		return tail.Sequence + 1, []string{tail.ID}, nil
	case -1:
		if acting.Epoch < head.Epoch {
			return 0, nil, refuse(ErrStaleLease, "set-profile acts under epoch %d past chain head epoch %d", acting.Epoch, head.Epoch)
		}
		return 0, nil, refuse(ErrDivergentLease, "set-profile acts under lease %s past chain head lease %s at epoch %d", acting.LeaseID, head.LeaseID, head.Epoch)
	default:
		return 0, nil, refuse(ErrDivergentLease, "set-profile acts under lease (%d, %s) past current lease (%d, %s)", acting.Epoch, acting.LeaseID, head.Epoch, head.LeaseID)
	}
}

// replayOrRefuse handles a request whose target already equals the
// effective profile. When the newest authoritative change carries
// the request envelope — the same from and to under the same
// lease, author, and instant — the change already committed and
// the retry replays it with the committed bytes; anything else is
// a genuine no-op change and refuses unchanged. The chained bytes
// are returned verbatim, so a replayed result equals the original
// commit result.
func (transactor *Transactor) replayOrRefuse(request SetProfileRequest, record Record, events []Event, current Pair) (SetProfileResult, error) {
	if !current.HasSource {
		return SetProfileResult{}, refuse(ErrProfileUnchanged, "session %s already carries effective profile %q", request.SessionID, request.To)
	}
	blob, err := transactor.Repo.GetEvent(request.SessionID, current.Source)
	if err != nil {
		return SetProfileResult{}, err
	}
	committed, err := DecodeEvent(blob)
	if err != nil {
		return SetProfileResult{}, err
	}
	if !sameEnvelope(committed, request) {
		return SetProfileResult{}, refuse(ErrProfileUnchanged, "session %s already carries effective profile %q", request.SessionID, request.To)
	}
	// The position scan always succeeds: Derive named the
	// source from this same slice, so the committed change is
	// chained by construction.
	position := 0
	for index := range events {
		if events[index].ID == committed.ID {
			position = index
			break
		}
	}
	return SetProfileResult{
		SessionID:       request.SessionID,
		PreviousProfile: current.Profile,
		NewProfile:      request.To,
		EventID:         committed.ID,
		Event:           append([]byte(nil), blob...),
		Ref:             sessrepo.EventRef{SessionID: request.SessionID, EventID: committed.ID, LeaseSequence: committed.Sequence, Position: position},
	}, nil
}

// sameEnvelope reports whether the committed change is the request
// already applied: the same target under the same lease, author,
// and instant. The from end is deliberately not compared: from is
// derived, not requested, and the committed from necessarily
// differs from the current profile (a change never targets its
// own source). Confirmation is likewise not part of the envelope:
// a replay commits nothing new, and the original commit already
// carried the confirmation its mint required.
func sameEnvelope(committed Event, request SetProfileRequest) bool {
	if committed.Type != "profile.changed" {
		return false
	}
	payloadTo, ok := payloadString(committed.Payload, "to")
	if !ok || payloadTo != request.To {
		return false
	}
	if committed.LeaseEpoch != request.LeaseEpoch || committed.LeaseID != request.LeaseID {
		return false
	}
	if committed.CreatedByHostID != request.CreatedByHostID || committed.CreatedAt != request.CreatedAt {
		return false
	}
	return true
}
