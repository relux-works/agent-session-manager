package axpane

import (
	"errors"
	"fmt"
	"os"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessprofile"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// Stores binds the durable owners the adapters read: sessions,
// leases, and events; materialization journals; checkpoints; and
// the wrapper's own bootstrap bindings.
type Stores struct {
	Repo *sessrepo.Repository
	Mat  *matjournal.Store
	Ckpt *sessckpt.Store
	Pane *Store
}

// ValidateConfig validates the configuration through the config
// owner. A nil error is the valid outcome Decide gates on; the
// loaded snapshot itself is the caller's to keep.
func ValidateConfig(inputs config.Inputs, overrides config.Overrides) error {
	_, err := config.Load(inputs, overrides)
	return err
}

// LoadSession loads the logical session through the sessrepo owner.
// known is false only when the session is absent; any read or
// verification failure propagates as an error, never as absence.
func LoadSession(repository *sessrepo.Repository, sessionID string) (known bool, record []byte, err error) {
	if repository == nil {
		return false, nil, errors.New("axpane session load carries no repository")
	}
	raw, err := repository.GetRecord(sessionID)
	if err != nil {
		if errors.Is(err, sessrepo.ErrUnknownSession) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, raw, nil
}

// ObserveOwnership loads the winning lease and combines it with the
// caller-reported sync, handoff, grant, and clock facts through the
// fencing owner. A session with no leases yet yields HasWinner
// false without an error: absence is data the gate parks on.
func ObserveOwnership(repository *sessrepo.Repository, sessionID string, input fencing.ObserveInput) (fencing.Observation, error) {
	if repository == nil {
		return fencing.Observation{}, errors.New("axpane ownership observation carries no repository")
	}
	return fencing.Observe(repository, sessionID, input)
}

// LoadProfile loads the decoded session surface the sessprofile
// reducer derives the effective pair from: the Session Record plus
// the authoritative chain in index order.
func LoadProfile(repository *sessrepo.Repository, sessionID string) (sessprofile.Record, []sessprofile.Event, error) {
	empty := sessprofile.Record{}
	if repository == nil {
		return empty, nil, errors.New("axpane profile load carries no repository")
	}
	recordJSON, err := repository.GetRecord(sessionID)
	if err != nil {
		return empty, nil, err
	}
	record, err := sessprofile.DecodeRecord(recordJSON)
	if err != nil {
		return empty, nil, err
	}
	summaries, err := repository.ListEvents(sessionID)
	if err != nil {
		return empty, nil, err
	}
	events := make([]sessprofile.Event, 0, len(summaries))
	for _, summary := range summaries {
		eventJSON, err := repository.GetEvent(sessionID, summary.EventID)
		if err != nil {
			return empty, nil, err
		}
		event, err := sessprofile.DecodeEvent(eventJSON)
		if err != nil {
			return empty, nil, err
		}
		events = append(events, event)
	}
	return record, events, nil
}

// LoadMaterialization loads the required materialization journal
// through the matjournal owner: grammar and cross-reference
// validation run on every read. Admitted is false only when the
// journal is absent; any read or validation failure propagates as
// an error, never as absence.
func LoadMaterialization(store *matjournal.Store, materializationID string) (matjournal.Journal, bool, error) {
	if store == nil {
		return matjournal.Journal{}, false, errors.New("axpane materialization load carries no store")
	}
	journal, _, err := store.Get(materializationID)
	if err != nil {
		if errors.Is(err, matjournal.ErrUnknownJournal) {
			return matjournal.Journal{}, false, nil
		}
		return matjournal.Journal{}, false, err
	}
	return journal, true, nil
}

// LoadCheckpoint loads the resume checkpoint through the sessckpt
// owner, which re-attests the closed shape and identity on every
// read. A nil document with a nil error is absence; any read or
// attestation failure propagates as an error, never as absence.
func LoadCheckpoint(store *sessckpt.Store, checkpointID string) ([]byte, error) {
	if store == nil {
		return nil, errors.New("axpane checkpoint load carries no store")
	}
	raw, err := store.Get(checkpointID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return raw, nil
}

// loadFoldInput loads the session record with its authoritative
// chain in index order as one reducer input. It is the one
// chain-fold reader: the parked-emission fold gate and the
// newest-checkpoint adapter both compose it, never a second fold.
// A read or decode failure propagates as an error, never as an
// empty chain.
func loadFoldInput(repository *sessrepo.Repository, sessionID string) (sessstate.Input, error) {
	empty := sessstate.Input{}
	if repository == nil {
		return empty, errors.New("axpane chain fold carries no repository")
	}
	recordJSON, err := repository.GetRecord(sessionID)
	if err != nil {
		return empty, err
	}
	record, err := sessstate.DecodeRecord(recordJSON)
	if err != nil {
		return empty, err
	}
	summaries, err := repository.ListEvents(sessionID)
	if err != nil {
		return empty, err
	}
	events := make([]sessstate.Event, 0, len(summaries))
	for _, summary := range summaries {
		raw, err := repository.GetEvent(sessionID, summary.EventID)
		if err != nil {
			return empty, err
		}
		event, err := sessstate.DecodeEvent(raw)
		if err != nil {
			return empty, err
		}
		events = append(events, event)
	}
	return sessstate.Input{Record: record, Events: events}, nil
}

// foldChain loads the session record with its authoritative chain
// and folds it through the landed sessstate reducer. A read,
// decode, or reducer failure propagates as an error: the caller
// decides whether an unfolderable chain is untrusted (the newest
// adapter) or simply not emittable (the parked fold gate).
func foldChain(repository *sessrepo.Repository, sessionID string) (sessstate.Projection, error) {
	empty := sessstate.Projection{}
	foldInput, err := loadFoldInput(repository, sessionID)
	if err != nil {
		return empty, err
	}
	return sessstate.Reduce(foldInput)
}

// LoadNewestCheckpoint derives the authoritative newest checkpoint
// through the landed chain fold (sessstate.Reduce over the
// record with its authoritative chain: Projection.HasCheckpoint
// and Newest.ID from checkpoint.created, checkpointed
// session.stopped, and session.resumed). It returns has false with
// an empty ID when the chain publishes no checkpoint. A read,
// decode, or reducer failure propagates as an error, never as
// absence: an unfolderable chain is untrusted, not checkpoint-free.
func LoadNewestCheckpoint(repository *sessrepo.Repository, sessionID string) (has bool, newest string, err error) {
	projection, err := foldChain(repository, sessionID)
	if err != nil {
		return false, "", err
	}
	if !projection.HasCheckpoint {
		return false, "", nil
	}
	return true, projection.Newest.ID, nil
}

// ChainTail returns the record digest and the current tail of the
// authoritative chain for parked-event linkage: the next sequence
// and the predecessor set the emission must reference.
func ChainTail(repository *sessrepo.Repository, sessionID string) (recordID string, tail *sessrepo.EventSummary, err error) {
	if repository == nil {
		return "", nil, errors.New("axpane chain tail carries no repository")
	}
	summaries, err := repository.ListSessions()
	if err != nil {
		return "", nil, err
	}
	found := false
	for index := range summaries {
		if summaries[index].SessionID == sessionID {
			recordID = summaries[index].RecordID
			found = true
			break
		}
	}
	if !found {
		return "", nil, fmt.Errorf("chain tail for unknown session %s", sessionID)
	}
	events, err := repository.ListEvents(sessionID)
	if err != nil {
		return "", nil, err
	}
	if len(events) == 0 {
		return recordID, nil, nil
	}
	previous := events[len(events)-1]
	return recordID, &previous, nil
}
