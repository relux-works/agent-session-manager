package sessrepo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// sessionRecord is the validated identity of one Session Record: the exact
// bytes plus the members the repository must bind (SPEC.md Section 5.1).
type sessionRecord struct {
	raw       []byte
	recordID  string
	sessionID string
	name      string
	kind      string
	provider  string
}

// eventView is the validated routing surface of one Session Event: the
// exact bytes plus the members the chain rules must read (SPEC.md 5.2).
type eventView struct {
	raw          []byte
	eventID      string
	eventType    string
	sessionID    string
	leaseEpoch   uint64
	leaseID      string
	leaseSeq     uint64
	predecessors []string
}

// storedChain is the chain index: the record digest plus one entry per
// chained event in authoritative order. The blobs are truth; the index is
// a verified cache rebound to them on every load.
type storedChain struct {
	RecordID string
	Events   []EventSummary
}

// sessionView is one loaded session: record, index, directory, and name facts.
type sessionView struct {
	directory string
	record    sessionRecord
	chain     storedChain
}

// decodeSessionRecord runs the three validation layers in order: strict
// frame (environ owner), member shape, then canonical identity and closed
// shape (canonicaljson owner). Each layer refuses before the next runs,
// so every site below is reachable with a vector that clears the earlier
// layers and fails exactly this one.
func decodeSessionRecord(raw []byte) (sessionRecord, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return sessionRecord{}, refuse(ErrInvalidRecord, "decode session record frame: %v", fault)
	}
	if schema, err := stringMember(members, "schema"); err != nil || schema != sessionRecordSchema {
		return sessionRecord{}, refuse(ErrInvalidRecord, "session record schema %q, want %q", schema, sessionRecordSchema)
	}
	sessionID, ok := environ.CheckUUIDv7(memberRaw(members, "session_id"))
	if !ok {
		return sessionRecord{}, refuse(ErrInvalidRecord, "session record carries no valid session_id member")
	}
	digest, field, err := canonicaljson.VerifyObjectIdentity(raw)
	if err != nil {
		return sessionRecord{}, refuse(ErrInvalidRecord, "verify session record identity: %v", err)
	}
	if field != canonicaljson.SelfRecordID {
		return sessionRecord{}, fmt.Errorf("session record identity field %q, want record_id", string(field))
	}
	name, _ := stringMember(members, "name")
	kind, _ := stringMember(members, "kind")
	provider, _ := stringMember(members, "provider_id")
	return sessionRecord{raw: raw, recordID: digest.String(), sessionID: sessionID.String(), name: name, kind: kind, provider: provider}, nil
}

// decodeSessionEvent runs the same three layers for Session Events: strict
// frame, member shape, then canonical identity and closed payload shape.
func decodeSessionEvent(raw []byte) (eventView, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return eventView{}, refuse(ErrInvalidEvent, "decode session event frame: %v", fault)
	}
	if schema, err := stringMember(members, "schema"); err != nil || schema != sessionEventSchema {
		return eventView{}, refuse(ErrInvalidEvent, "session event schema %q, want %q", schema, sessionEventSchema)
	}
	event, err := extractEventMembers(members)
	if err != nil {
		return eventView{}, refuse(ErrInvalidEvent, "session event member: %v", err)
	}
	digest, field, err := canonicaljson.VerifyObjectIdentity(raw)
	if err != nil {
		return eventView{}, refuse(ErrInvalidEvent, "verify session event identity: %v", err)
	}
	if field != canonicaljson.SelfEventID {
		return eventView{}, fmt.Errorf("session event identity field %q, want event_id", string(field))
	}
	event.raw = raw
	event.eventID = digest.String()
	return event, nil
}

// extractEventMembers reads the routing members after the frame gate. It
// returns plain errors; the caller attributes them through its one member
// site. Grammar and range rules stay with their owners: UUIDv7 and uint53
// with environ, digests and UUIDv4 with scalar.
func extractEventMembers(members map[string]json.RawMessage) (eventView, error) {
	var event eventView
	sessionID, ok := environ.CheckUUIDv7(memberRaw(members, "session_id"))
	if !ok {
		return eventView{}, fmt.Errorf("member %q is not a UUIDv7", "session_id")
	}
	event.sessionID = sessionID.String()
	eventType, err := stringMember(members, "event_type")
	if err != nil {
		return eventView{}, err
	}
	event.eventType = eventType
	epoch, ok := environ.CheckUint53Bounds(memberRaw(members, "lease_epoch"), 1, uint53Max)
	if !ok {
		return eventView{}, fmt.Errorf("member %q is not a uint53 at or above 1", "lease_epoch")
	}
	event.leaseEpoch = epoch
	leaseID, err := stringMember(members, "lease_id")
	if err != nil {
		return eventView{}, err
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return eventView{}, fmt.Errorf("member %q is not a UUIDv4: %v", "lease_id", err)
	}
	event.leaseID = leaseID
	sequence, ok := environ.CheckUint53Bounds(memberRaw(members, "lease_sequence"), 1, uint53Max)
	if !ok {
		return eventView{}, fmt.Errorf("member %q is not a uint53 at or above 1", "lease_sequence")
	}
	event.leaseSeq = sequence
	predecessors, err := digestMembers(members, "predecessors")
	if err != nil {
		return eventView{}, err
	}
	if len(predecessors) == 0 {
		return eventView{}, fmt.Errorf("member %q names no predecessor", "predecessors")
	}
	event.predecessors = predecessors
	return event, nil
}

// checkAppend enforces the SPEC.md Section 5.2 chain rules for one event
// against the record and the current tail: the first event links exactly
// the Session Record at sequence 1 with no gap; within one lease the
// sequence increases by exactly one with no repeat and each event
// references the immediately prior authoritative event; a successor lease
// restarts at 1 referencing the predecessor head; a same-epoch second
// lease diverges and a lower epoch is stale. The reload audit folds this
// same check over the stored index, so weakening any arm breaks both the
// append and the reload evidence.
func checkAppend(recordID string, tail *EventSummary, event eventView) error {
	if tail != nil && event.leaseID != tail.LeaseID {
		if event.leaseEpoch < tail.LeaseEpoch {
			return refuse(ErrStaleLease, "event lease epoch %d precedes chain head epoch %d", event.leaseEpoch, tail.LeaseEpoch)
		}
		if event.leaseEpoch == tail.LeaseEpoch {
			return refuse(ErrDivergentBranch, "event under lease %s diverges from chain head lease %s at epoch %d", event.leaseID, tail.LeaseID, tail.LeaseEpoch)
		}
	}
	want := uint64(1)
	if tail != nil && event.leaseID == tail.LeaseID {
		want = tail.LeaseSequence + 1
	}
	if event.leaseSeq < want {
		return refuse(ErrSequenceRepeat, "event lease sequence %d repeats chained sequence through %d", event.leaseSeq, want-1)
	}
	if event.leaseSeq > want {
		return refuse(ErrSequenceGap, "event lease sequence %d skips chained sequence %d", event.leaseSeq, want)
	}
	if tail == nil {
		if len(event.predecessors) != 1 || event.predecessors[0] != recordID {
			return refuse(ErrPredecessorLink, "first event must link exactly the session record")
		}
		return nil
	}
	if !containsDigest(event.predecessors, tail.EventID) {
		return refuse(ErrPredecessorLink, "event predecessors omit prior authoritative event")
	}
	return nil
}

// checkWinningLease enforces the owner-side admission rule for a new event
// once a session has a lease store. Historical chain replay intentionally
// does not call this helper: an event authored by an earlier winner remains
// valid history after a successor wins. A new lower-epoch event, or a new
// same-epoch event from a losing lease, is an unapplied branch even when the
// chain tail still names the old lease.
func checkWinningLease(winner LeaseSummary, event eventView) error {
	if event.leaseEpoch < winner.Epoch {
		return refuse(ErrStaleLease, "event lease epoch %d precedes winning lease epoch %d", event.leaseEpoch, winner.Epoch)
	}
	if event.leaseEpoch == winner.Epoch && event.leaseID != winner.LeaseID {
		return refuse(ErrDivergentBranch, "event lease %s loses to winning lease %s at epoch %d", event.leaseID, winner.LeaseID, winner.Epoch)
	}
	return nil
}

func containsDigest(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}

// memberRaw reports the raw member for the environ bound checkers. A
// missing member yields nil, which every checker refuses.
func memberRaw(members map[string]json.RawMessage, name string) json.RawMessage {
	return members[name]
}

// stringMember reads one string member with the language decoder. Grammar
// rules for the value stay with the shape owner; this only accesses it.
func stringMember(members map[string]json.RawMessage, name string) (string, error) {
	raw, ok := members[name]
	if !ok {
		return "", fmt.Errorf("member %q is absent", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("member %q is not a string: %v", name, err)
	}
	return value, nil
}

// digestMembers reads one array of digest strings. Digest grammar stays
// with the scalar owner.
func digestMembers(members map[string]json.RawMessage, name string) ([]string, error) {
	raw, ok := members[name]
	if !ok {
		return nil, fmt.Errorf("member %q is absent", name)
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("member %q is not a string array: %v", name, err)
	}
	for _, value := range values {
		if _, err := scalar.ParseDigest(value); err != nil {
			return nil, fmt.Errorf("member %q holds malformed digest %q: %v", name, value, err)
		}
	}
	return values, nil
}

// sessionDir joins a caller-supplied session ID with no grammar check: a
// traversal ID escapes the sessions root, and the escape refuses at load
// time through the record/session binding, which it can never satisfy.
func (repository *Repository) sessionDir(sessionID string) string {
	return filepath.Join(repository.root, sessionID)
}

// loadSessionLocked loads and fully re-verifies one session: the chain
// index decodes, the record bytes re-verify against the indexed digest,
// every indexed event blob re-verifies, and the index folds clean through
// the append rules. Callers hold the repository mutex.
func (repository *Repository) loadSessionLocked(sessionID string) (*sessionView, error) {
	directory := repository.sessionDir(sessionID)
	if _, err := os.Stat(directory); err != nil {
		return nil, refuse(ErrUnknownSession, "unknown session %s", sessionID)
	}
	chainBytes, err := os.ReadFile(filepath.Join(directory, "chain.json"))
	if err != nil {
		return nil, refuse(ErrChainCorrupt, "read chain index for session %s: %v", sessionID, err)
	}
	chain, err := decodeChainIndex(chainBytes)
	if err != nil {
		return nil, refuse(ErrChainCorrupt, "decode chain index for session %s: %v", sessionID, err)
	}
	recordBytes, err := os.ReadFile(filepath.Join(directory, "record.json"))
	if err != nil {
		return nil, refuse(ErrChainCorrupt, "read session record for session %s: %v", sessionID, err)
	}
	record, err := decodeSessionRecord(recordBytes)
	if err != nil {
		return nil, refuse(ErrChainCorrupt, "re-verify session record for session %s: %v", sessionID, err)
	}
	if record.recordID != chain.RecordID || record.sessionID != sessionID {
		return nil, refuse(ErrChainCorrupt, "chain index does not describe stored session record for session %s", sessionID)
	}
	events := filepath.Join(directory, "events")
	for position, indexed := range chain.Events {
		blob, err := os.ReadFile(filepath.Join(events, blobFileName(indexed.EventID)))
		if err != nil {
			return nil, refuse(ErrChainCorrupt, "read chained event %d for session %s: %v", position, sessionID, err)
		}
		digest, field, err := canonicaljson.VerifyObjectIdentity(blob)
		if err != nil {
			return nil, refuse(ErrChainCorrupt, "re-verify chained event %d for session %s: %v", position, sessionID, err)
		}
		if field != canonicaljson.SelfEventID {
			return nil, refuse(ErrChainCorrupt, "chained event %d for session %s carries self field %q, want event_id", position, sessionID, string(field))
		}
		if digest.String() != indexed.EventID {
			return nil, refuse(ErrChainCorrupt, "chained event %d for session %s digest %s disagrees with indexed event %s", position, sessionID, digest.String(), indexed.EventID)
		}
		var tail *EventSummary
		if position > 0 {
			tail = &chain.Events[position-1]
		}
		replay := eventView{eventID: indexed.EventID, eventType: indexed.EventType, sessionID: sessionID, leaseEpoch: indexed.LeaseEpoch, leaseID: indexed.LeaseID, leaseSeq: indexed.LeaseSequence, predecessors: indexed.Predecessors}
		if err := checkAppend(chain.RecordID, tail, replay); err != nil {
			return nil, refuse(ErrChainCorrupt, "chain index order breaks continuity at event %d for session %s: %v", position, sessionID, err)
		}
	}
	return &sessionView{directory: directory, record: record, chain: chain}, nil
}

// decodeChainIndex decodes the internal chain index. It is not an AX wire
// object, so the canonical owner does not validate it; member grammars
// still stay with environ and scalar.
func decodeChainIndex(raw []byte) (storedChain, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return storedChain{}, fmt.Errorf("decode chain frame: %v", fault)
	}
	recordID, err := stringMember(members, "record_id")
	if err != nil {
		return storedChain{}, err
	}
	if _, err := scalar.ParseDigest(recordID); err != nil {
		return storedChain{}, fmt.Errorf("chain record_id: %v", err)
	}
	rawEvents, ok := members["events"]
	if !ok {
		return storedChain{}, fmt.Errorf("member %q is absent", "events")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(rawEvents, &entries); err != nil {
		return storedChain{}, fmt.Errorf("member %q is not an object array: %v", "events", err)
	}
	chain := storedChain{RecordID: recordID}
	for _, entry := range entries {
		summary, err := decodeChainEntry(entry)
		if err != nil {
			return storedChain{}, err
		}
		chain.Events = append(chain.Events, summary)
	}
	return chain, nil
}

// decodeChainEntry validates one index entry. Failures are plain errors;
// loadSessionLocked attributes them through its one index site.
func decodeChainEntry(entry map[string]json.RawMessage) (EventSummary, error) {
	var summary EventSummary
	eventID, err := stringMember(entry, "event_id")
	if err != nil {
		return EventSummary{}, err
	}
	if _, err := scalar.ParseDigest(eventID); err != nil {
		return EventSummary{}, fmt.Errorf("chain event_id: %v", err)
	}
	summary.EventID = eventID
	eventType, err := stringMember(entry, "event_type")
	if err != nil {
		return EventSummary{}, err
	}
	summary.EventType = eventType
	epoch, ok := environ.CheckUint53Bounds(entry["lease_epoch"], 1, uint53Max)
	if !ok {
		return EventSummary{}, fmt.Errorf("chain member %q is not a uint53 at or above 1", "lease_epoch")
	}
	summary.LeaseEpoch = epoch
	leaseID, err := stringMember(entry, "lease_id")
	if err != nil {
		return EventSummary{}, err
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return EventSummary{}, fmt.Errorf("chain lease_id: %v", err)
	}
	summary.LeaseID = leaseID
	sequence, ok := environ.CheckUint53Bounds(entry["lease_sequence"], 1, uint53Max)
	if !ok {
		return EventSummary{}, fmt.Errorf("chain member %q is not a uint53 at or above 1", "lease_sequence")
	}
	summary.LeaseSequence = sequence
	predecessors, err := digestMembers(entry, "predecessors")
	if err != nil {
		return EventSummary{}, err
	}
	summary.Predecessors = predecessors
	return summary, nil
}

// writeChain stores the index atomically: temp file, fsync, rename, fsync
// the directory. A crash either leaves the previous index or the new one;
// the blobs it names are verified on every load, so neither outcome can
// invent history.
func (repository *Repository) writeChain(directory string, chain storedChain) error {
	payload, err := json.Marshal(chainPayload{RecordID: chain.RecordID, Events: chain.Events})
	if err != nil {
		return fmt.Errorf("encode chain index: %w", err)
	}
	return writeAtomic(filepath.Join(directory, "chain.json"), payload)
}

// chainPayload is the on-disk shape of the index. EventSummary carries the
// JSON tags so the stored form stays exactly this struct.
type chainPayload struct {
	RecordID string         `json:"record_id"`
	Events   []EventSummary `json:"events"`
}

// errDigestDisagreement reports bytes at a digest path that a completed
// read proved different from the installing candidate. It is an internal
// marker, never a refusal: callers attribute it through their own site.
var errDigestDisagreement = errors.New("digest path holds disagreeing bytes")

// installEventBlob installs one event blob idempotently through the
// no-replace open first: a fresh path is created atomically, and only an
// EEXIST loser falls back to comparing against the winner's bytes —
// identical bytes are reused, disagreeing bytes are reported for
// refusal. Leading with the exclusive create (instead of stat-then-act)
// keeps the no-replace property a filesystem fact under a second
// process, not a check-then-act race the repository mutex cannot see:
// the mutex serializes one process, while O_EXCL serializes all of
// them. A path that exists but is not a readable file still fails the
// install, as an operational error rather than a digest disagreement.
func installEventBlob(path string, data []byte) error {
	if err := writeExclusive(path, data); err == nil {
		return nil
	} else if !os.IsExist(err) {
		return fmt.Errorf("install event blob: %w", err)
	}
	existing, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(existing, data) {
		return errDigestDisagreement
	}
	return nil
}

// writeExclusive installs bytes at a path that must not exist. The
// no-replace open makes append-only a filesystem property, not a convention.
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

// writeAtomic replaces a path through a temp file in the same directory:
// staged bytes fsync before the rename, the directory fsyncs after.
func writeAtomic(path string, data []byte) error {
	staged, err := os.CreateTemp(filepath.Dir(path), ".ax-chain-stage-")
	if err != nil {
		return fmt.Errorf("stage chain index: %w", err)
	}
	stagedPath := staged.Name()
	finished := false
	defer func() {
		if !finished {
			_ = staged.Close()
			_ = os.Remove(stagedPath)
		}
	}()
	if err := staged.Chmod(0o600); err != nil {
		return fmt.Errorf("mode staged chain index: %w", err)
	}
	if _, err := io.Copy(staged, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("write staged chain index: %w", err)
	}
	if err := staged.Sync(); err != nil {
		return fmt.Errorf("fsync staged chain index: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close staged chain index: %w", err)
	}
	finished = true
	if err := os.Rename(stagedPath, path); err != nil {
		return fmt.Errorf("install chain index: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

// blobFileName maps a validated event digest to its file name. The digest
// passed scalar grammar at decode time; Hex is the owner's rendering.
func blobFileName(eventID string) string {
	digest, err := scalar.ParseDigest(eventID)
	if err != nil {
		return "invalid.json"
	}
	return digest.Hex() + ".json"
}
