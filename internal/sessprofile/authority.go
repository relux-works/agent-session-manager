package sessprofile

import (
	"errors"

	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// SourceAuthority carries the durable lease facts used when selecting a
// profile.changed source. A previous lease contributes only through the
// winning lease's captured handoff closure; the current winner contributes
// through its own events. Lease summaries and checkpoint heads come from
// their canonical owners.
type SourceAuthority struct {
	Winner            sessrepo.LeaseSummary
	HasWinner         bool
	Leases            []sessrepo.LeaseSummary
	HandoffHeads      []string
	HasHandoffClosure bool
	HandoffErr        error
}

// Derivation bundles the stored profile surface with the durable facts that
// authorize profile.changed sources. Use Derive or DeriveForHeads instead of
// the pure reducer when the input came from a repository.
type Derivation struct {
	Record    Record
	Events    []Event
	Authority SourceAuthority
}

// Derive projects the session-head pair using the captured lease authority.
func (derivation Derivation) Derive() (Pair, error) {
	return deriveWithAuthority(derivation.Record, derivation.Events, derivation.Authority, nil)
}

// DeriveForHeads projects only the transitive closure named by a checkpoint.
// A previous-lease source must also belong to the current winner's attested
// handoff closure; naming an event directly cannot grant it authority.
func (derivation Derivation) DeriveForHeads(heads []string) (Pair, error) {
	if len(heads) == 0 {
		return Pair{}, refuse(ErrDerivation, "checkpoint closure names no event head")
	}
	return deriveWithAuthority(derivation.Record, derivation.Events, derivation.Authority, heads)
}

// LoadSourceAuthority reads the lease store and, when the winner names a
// handoff checkpoint, asks sessckpt to re-attest its closure against the
// session chain. An empty lease store is represented explicitly: the record
// remains the profile authority until a winning lease exists.
func LoadSourceAuthority(repository *sessrepo.Repository, checkpoints *sessckpt.Store, sessionID string) (SourceAuthority, error) {
	if repository == nil {
		return SourceAuthority{}, refuse(ErrDerivation, "source authority carries no repository")
	}
	leases, err := repository.ListLeases(sessionID)
	if err != nil {
		return SourceAuthority{}, err
	}
	if len(leases) == 0 {
		return SourceAuthority{}, nil
	}
	// ListLeases returns the validated store in the tuple order owned by
	// sessrepo; the final row is the winner from the same snapshot as Leases.
	winner := leases[len(leases)-1]
	authority := SourceAuthority{Winner: winner, HasWinner: true, Leases: leases}
	if winner.HasCheckpoint {
		if checkpoints == nil {
			authority.HandoffErr = errors.New("checkpoint store is not bound")
		} else {
			heads, err := checkpoints.EventHeads(repository, winner.Checkpoint, sessionID)
			if err != nil {
				authority.HandoffErr = err
			} else {
				authority.HandoffHeads = heads
				authority.HasHandoffClosure = true
			}
		}
	}
	return authority, nil
}

// deriveWithAuthority keeps chain validation with the existing reducer, then
// folds only changes supported by the durable winning lease and its handoff
// closure. A checkpoint closure supplied by a resume caller narrows that set
// further. An unreadable handoff closure never promotes an ancestor event.
func deriveWithAuthority(record Record, events []Event, authority SourceAuthority, heads []string) (Pair, error) {
	if err := checkRecord(record); err != nil {
		return Pair{}, err
	}
	limit := len(events)
	var requestedClosure map[string]bool
	var handoffClosure map[string]bool
	if authority.HasWinner && authority.Winner.HasCheckpoint && authority.HasHandoffClosure {
		var err error
		handoffClosure, _, err = closurePrefix(record.RecordID, events, authority.HandoffHeads)
		if err != nil {
			return Pair{}, err
		}
	}
	if len(heads) > 0 {
		var err error
		requestedClosure, limit, err = closurePrefix(record.RecordID, events, heads)
		if err != nil {
			return Pair{}, err
		}
	}
	if authority.HasWinner && authority.Winner.HasCheckpoint && !authority.HasHandoffClosure {
		for _, event := range events {
			if len(heads) > 0 && !requestedClosure[event.ID] {
				continue
			}
			if event.Type == "profile.changed" && knownLease(authority, event) && !isWinningLease(authority, event) {
				return Pair{}, refuse(ErrDerivation, "cannot establish authority for prior profile.changed event %s without the winning handoff closure: %v", event.ID, authority.HandoffErr)
			}
		}
	}
	ordered, err := checkChain(record, events, limit)
	if err != nil {
		return Pair{}, err
	}
	pair := Pair{Profile: record.Creation}
	for _, event := range ordered {
		if event.Type != "profile.changed" {
			continue
		}
		if len(heads) > 0 && !requestedClosure[event.ID] {
			continue
		}
		if !authorizedSource(authority, event, requestedClosure, handoffClosure, len(heads) > 0) {
			continue
		}
		target, err := changeTarget(event)
		if err != nil {
			return Pair{}, err
		}
		pair = Pair{Profile: target, Source: event.ID, HasSource: true}
	}
	return pair, nil
}

// authorizedSource accepts a current-winner event in the requested closure,
// or a previous-lease event in both the requested closure and the winner's
// captured handoff closure. Exact lease tuples must exist in the store.
func authorizedSource(authority SourceAuthority, event Event, requestedClosure map[string]bool, handoffClosure map[string]bool, explicitClosure bool) bool {
	if !authority.HasWinner || !knownLease(authority, event) {
		return false
	}
	if explicitClosure && !requestedClosure[event.ID] {
		return false
	}
	if isWinningLease(authority, event) {
		return true
	}
	return handoffClosure[event.ID]
}

func knownLease(authority SourceAuthority, event Event) bool {
	for _, lease := range authority.Leases {
		if event.LeaseEpoch == lease.Epoch && event.LeaseID == lease.LeaseID {
			return true
		}
	}
	return false
}

func isWinningLease(authority SourceAuthority, event Event) bool {
	return authority.HasWinner && event.LeaseEpoch == authority.Winner.Epoch && event.LeaseID == authority.Winner.LeaseID
}
