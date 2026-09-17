package sessrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// syncDirectory fsyncs a directory so a rename inside it survives a crash.
// Opening a directory fails on Windows, where the rename itself is the
// durability boundary; there the sync is skipped, never faked.
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

// GetRecord returns the byte-identical Session Record for a session,
// re-verified through the canonical owner on every read.
func (repository *Repository) GetRecord(sessionID string) ([]byte, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), view.record.raw...), nil
}

// GetEvent returns the byte-identical chained event for its digest.
func (repository *Repository) GetEvent(sessionID, eventID string) ([]byte, error) {
	if _, err := scalar.ParseDigest(eventID); err != nil {
		return nil, refuse(ErrUnknownEvent, "malformed event id %q", eventID)
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return nil, err
	}
	for _, indexed := range view.chain.Events {
		if indexed.EventID == eventID {
			blob, err := os.ReadFile(filepath.Join(view.directory, "events", blobFileName(eventID)))
			if err != nil {
				return nil, fmt.Errorf("read chained event for session %s: %w", sessionID, err)
			}
			return blob, nil
		}
	}
	return nil, refuse(ErrUnknownEvent, "unknown event %q in session %s", eventID, sessionID)
}

// ListEvents returns the chain summaries in authoritative order.
func (repository *Repository) ListEvents(sessionID string) ([]EventSummary, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	view, err := repository.loadSessionLocked(sessionID)
	if err != nil {
		return nil, err
	}
	return append([]EventSummary(nil), view.chain.Events...), nil
}

// ListSessions returns one summary per stored session in session-ID order,
// healthy sessions first-class and parked sessions on the per-session
// parked channel. A session whose bytes no longer verify never fails the
// listing for its siblings: it is returned as a parked entry (Parked=true)
// carrying the blocking reason and the same-operation retry the Section
// 5.7 reducer and the Section 14.4 rendering leaves project into the
// parked lifecycle state and its warnings. Only a failure to read the
// sessions root itself is a listing error.
//
// Decision record (SPEC.md:9082, Section 13.13 recoverable_parked_state;
// Section 14.4 per-session warnings): the prior closed-listing rule failed
// the whole repository — one parked session removed every healthy sibling
// from listing and name resolution, and only the byte-identical original
// record healed it, so lost bytes bricked the repository permanently with
// no per-session channel for downstream leaves to render. Per-session
// parked reporting keeps healthy sessions listable and resolvable while
// the parked session stays visible with its reason and retry. The tradeoff
// is that callers must check Parked instead of relying on an error: a
// listing with parked entries returns nil error by design. See doc.go for
// the retry table and the operator remedy when no retry heals.
func (repository *Repository) ListSessions() ([]SessionSummary, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	return repository.listSessionsLocked()
}

// Resolve implements the local steps of SPEC.md Section 2.3: an exact live
// session name in the local index wins first; a UUID query routes to its
// session; anything else is not found. Names display with original case
// but collide under ASCII case folding because the allowed alphabet is
// ASCII, and a colliding query fails with ErrNameAmbiguous. Parked
// sessions never participate: listing reports them per session, but name
// resolution routes only healthy sessions, so one parked sibling cannot
// deny service to the rest. A query naming only a parked session is not
// found. Peer-learned names and the interactive choice surface belong to
// the name-resolution leaf, which builds on this entry.
func (repository *Repository) Resolve(query string) (SessionSummary, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	summaries, err := repository.listSessionsLocked()
	if err != nil {
		return SessionSummary{}, err
	}
	var live []SessionSummary
	for _, summary := range summaries {
		if !summary.Parked {
			live = append(live, summary)
		}
	}
	var folded []SessionSummary
	for _, summary := range live {
		if asciiFold(summary.Name) == asciiFold(query) {
			folded = append(folded, summary)
		}
	}
	if len(folded) > 1 {
		return SessionSummary{}, refuse(ErrNameAmbiguous, "name %q collides across %d sessions", query, len(folded))
	}
	if len(folded) == 1 && folded[0].Name == query {
		return folded[0], nil
	}
	if id, err := scalar.ParseUUIDv7(query); err == nil {
		for _, summary := range live {
			if summary.SessionID == id.String() {
				return summary, nil
			}
		}
	}
	return SessionSummary{}, refuse(ErrNameNotFound, "no live session named %q", query)
}

// listSessionsLocked shares the listing scan with Resolve without
// re-locking. Per-session load failures become parked entries; only the
// root read is an error. Callers hold the repository mutex.
func (repository *Repository) listSessionsLocked() ([]SessionSummary, error) {
	entries, err := os.ReadDir(repository.root)
	if err != nil {
		return nil, refuse(ErrRepositoryPath, "read sessions root: %v", err)
	}
	var summaries []SessionSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		view, err := repository.loadSessionLocked(entry.Name())
		if err != nil {
			summaries = append(summaries, parkedSummary(repository.sessionDir(entry.Name()), entry.Name(), err))
			continue
		}
		summary := SessionSummary{SessionID: view.record.sessionID, RecordID: view.record.recordID, Name: view.record.name, Kind: view.record.kind, ProviderID: view.record.provider, EventCount: len(view.chain.Events)}
		if len(view.chain.Events) > 0 {
			summary.TailEvent = view.chain.Events[len(view.chain.Events)-1].EventID
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].SessionID < summaries[j].SessionID })
	return summaries, nil
}

// retryHintForTornStore names the remedy when no same-operation retry can
// heal: the parked bytes are not an interrupted create (the chain index is
// present but the store no longer verifies), so CreateSession cannot
// resume it. The operator remedy is documented in doc.go.
const retryHintForTornStore = "no CreateSession retry heals this state; see the operator remedy for parked sessions"

// parkedSummary builds the per-session parked channel for one load
// failure: the session directory name is always reported, the blocking
// reason names the recoverable_parked_state block with its per-session
// cause, and the retry hint names the same-operation retry when one
// exists. Identity members are filled from the parked record when it
// still verifies; otherwise they stay empty. It reports parked state as
// data and refuses nothing, so the refusal census is unchanged.
func parkedSummary(directory, sessionID string, loadErr error) SessionSummary {
	summary := SessionSummary{SessionID: sessionID, Parked: true, BlockingReason: "recoverable_parked_state: " + loadErr.Error()}
	if _, err := os.Stat(filepath.Join(directory, "chain.json")); os.IsNotExist(err) {
		recordBytes, readErr := os.ReadFile(filepath.Join(directory, "record.json"))
		if readErr != nil {
			if !os.IsNotExist(readErr) {
				summary.RetryHint = retryHintForTornStore
				return summary
			}
			summary.RetryHint = retryHintForBareDirectory
			return summary
		}
		if record, decodeErr := decodeSessionRecord(recordBytes); decodeErr == nil {
			summary.RecordID = record.recordID
			summary.Name = record.name
			summary.Kind = record.kind
			summary.ProviderID = record.provider
		}
		summary.RetryHint = retryHintForParkedRecord
		return summary
	}
	summary.RetryHint = retryHintForTornStore
	return summary
}

// asciiFold folds ASCII uppercase to lowercase and passes every other byte
// through. Session names match [A-Za-z0-9][A-Za-z0-9._-]{0,63} under the
// shape owner, so the fold domain is ASCII by construction and a Unicode
// fold table would be an unowned duplicate, not a reuse.
func asciiFold(value string) string {
	folded := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		folded = append(folded, character)
	}
	return string(folded)
}
