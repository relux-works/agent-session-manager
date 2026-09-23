package sessprofile

import (
	"errors"
	"fmt"
)

// Execution profiles Section 2.4 defines. The vocabulary is exactly
// these two spellings; no registry, alias, or third value exists.
const (
	ProfileStandard = "standard"
	ProfileYOLO     = "yolo"
)

var (
	// ErrInvalidRecord reports a Session Record projection without
	// session identity or without a valid creation profile.
	ErrInvalidRecord = errors.New("invalid session record")
	// ErrInvalidEvent reports a Session Event projection without the
	// routing members the derivation must read.
	ErrInvalidEvent = errors.New("invalid session event")
	// ErrInvalidTransition reports chain input the repository could
	// not have authored: a broken predecessor link, a repeated or
	// skipped sequence, or an ambiguous branch. The message is the
	// Section 5.2 code.
	ErrInvalidTransition = errors.New("invalid_state_transition")
	// ErrStaleLease reports an input event under an epoch below the
	// chain head, or a set-profile request acting under one.
	ErrStaleLease = errors.New("event lease epoch precedes the chain head")
	// ErrDivergentLease reports an input event under a second lease
	// at the chain-head epoch, or a set-profile request acting
	// under a lease that is not the chain head.
	ErrDivergentLease = errors.New("divergent lease branch cannot be authoritative")
	// ErrIntegrity reports a pair or closure the admitted history
	// contradicts. The message is the Section 2.4/5.2 code.
	ErrIntegrity = errors.New("integrity_failure")
	// ErrInvalidProfile reports a profile spelling outside the
	// standard|yolo vocabulary.
	ErrInvalidProfile = errors.New("invalid execution profile")
	// ErrUnconfirmedYOLO reports a profile change to yolo without
	// operator confirmation.
	ErrUnconfirmedYOLO = errors.New("profile change to yolo requires operator confirmation")
	// ErrProfileUnchanged reports a profile change whose from and to
	// are equal.
	ErrProfileUnchanged = errors.New("profile.changed from and to must differ")
	// ErrDerivation reports corrupt derivation input: a missing
	// repository, an empty closure, or an unknown activation.
	ErrDerivation = errors.New("derivation input is corrupt")
	// ErrUnknownSession reports a session ID with no listing entry.
	// A parked session is not unknown; see ErrSessionParked.
	ErrUnknownSession = errors.New("unknown session")
	// ErrSessionParked reports a session whose bytes no longer
	// verify. The message carries the blocking reason and retry;
	// the profile of a parked session cannot be derived.
	ErrSessionParked = errors.New("session is parked")
)

// refuse is the single refusal funnel every sessprofile rejection
// flows through. It is a var so a census can wrap it and record the
// production file:line behind each exercised refusal. No production
// path wraps a sentinel with fmt.Errorf directly.
var refuse = func(sentinel error, format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, arguments...))
}

// Pair is one Section 2.4 effective profile with its nullable
// source: the profile spelling plus the digest of the newest
// authoritative profile.changed event it was derived from, or no
// source when the Session Record is still the authority. The zero
// value is the dormant null pair (no profile, no source) used only
// by dormant materialization finalization.
type Pair struct {
	Profile   string
	Source    string
	HasSource bool
}

// Equal reports whole-pair equality: the profile spelling and the
// source presence with its digest must all match. A dormant null
// pair equals only another dormant null pair.
func (pair Pair) Equal(other Pair) bool {
	return pair.Profile == other.Profile && pair.HasSource == other.HasSource && pair.Source == other.Source
}

// Record is the decoded Session Record surface the reducer reads:
// session identity plus the immutable creation profile.
type Record struct {
	SessionID string
	RecordID  string
	Creation  string
}

// Event is one decoded authoritative Session Event: the routing
// members plus the closed payload object admitted by the canonical
// owner at decode time. The author and instant identify a committed
// change for set-profile replay.
type Event struct {
	ID              string
	Type            string
	SchemaVersion   string
	SessionID       string
	CreatedByHostID string
	CreatedAt       string
	LeaseEpoch      uint64
	LeaseID         string
	Sequence        uint64
	Predecessors    []string
	Payload         map[string]any
}

// LeaseHead is one observed fencing token: the winning-lease pair
// carried by every owner-authored event and mutation.
type LeaseHead struct {
	Epoch   uint64
	LeaseID string
}

// Derive folds one record with its authoritative events in chain
// order into the session-head effective pair: the creation profile
// while no authoritative profile.changed event exists, otherwise
// the newest change's target with that event's digest as source.
// An empty chain derives the creation pair. It is pure: no I/O, no
// clock, no cache. Chain continuity is re-checked over the input,
// so a chain-forbidden reordering refuses with
// invalid_state_transition instead of silently deriving a different
// pair. This reducer has no lease-store view and does not establish
// durable source authority; repository-backed consumers use
// Derivation.Derive, which applies the winning-lease and handoff-closure
// checks before selecting a profile.changed source.
func Derive(record Record, events []Event) (Pair, error) {
	if err := checkRecord(record); err != nil {
		return Pair{}, err
	}
	if len(events) == 0 {
		return Pair{Profile: record.Creation}, nil
	}
	ordered, err := checkChain(record, events, len(events))
	if err != nil {
		return Pair{}, err
	}
	return foldChanges(record.Creation, ordered, nil)
}

// DeriveForHeads derives the effective pair over the transitive
// predecessor closure of heads: the Section 5.4 event-head closure
// of one checkpoint. Only closure events are consulted — a later
// local-only change outside the heads never affects the pair, and
// the derivation never falls back to the creation value while the
// closure holds a change. Events past the closure are not even
// validated. This pure reducer does not establish durable lease
// authority; repository-backed consumers use Derivation.DeriveForHeads.
// An unknown head or a dangling predecessor contradicts the admitted
// closure (integrity_failure); empty heads carry no closure to derive over.
func DeriveForHeads(record Record, events []Event, heads []string) (Pair, error) {
	if err := checkRecord(record); err != nil {
		return Pair{}, err
	}
	if len(heads) == 0 {
		return Pair{}, refuse(ErrDerivation, "checkpoint closure names no event head")
	}
	closure, limit, err := closurePrefix(record.RecordID, events, heads)
	if err != nil {
		return Pair{}, err
	}
	ordered, err := checkChain(record, events, limit)
	if err != nil {
		return Pair{}, err
	}
	return foldChanges(record.Creation, ordered, closure)
}

// checkRecord validates the record surface the derivation reads:
// session identity plus a creation profile inside the vocabulary.
func checkRecord(record Record) error {
	if record.SessionID == "" || record.RecordID == "" {
		return refuse(ErrInvalidRecord, "reducer input carries no session identity")
	}
	if record.Creation != ProfileStandard && record.Creation != ProfileYOLO {
		return refuse(ErrInvalidRecord, "session record carries creation profile %q, want standard|yolo", record.Creation)
	}
	return nil
}

// closurePrefix resolves the transitive predecessor closure of
// heads against the chain index in input order. The Session Record
// digest is the genesis terminal; every other predecessor must
// name a chained event. It returns the closure set plus the
// one-past-the-end validation limit: the highest closure position
// plus one, so events past the closure are never consulted.
func closurePrefix(recordID string, events []Event, heads []string) (map[string]bool, int, error) {
	byID := make(map[string]Event, len(events))
	position := make(map[string]int, len(events))
	for index := range events {
		byID[events[index].ID] = events[index]
		position[events[index].ID] = index
	}
	for _, head := range heads {
		if _, ok := byID[head]; !ok {
			return nil, 0, refuse(ErrIntegrity, "checkpoint closure names unknown event head %s", head)
		}
	}
	closure := make(map[string]bool, len(events))
	stack := append([]string(nil), heads...)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if closure[id] {
			continue
		}
		event, ok := byID[id]
		if !ok {
			if id == recordID {
				continue
			}
			return nil, 0, refuse(ErrIntegrity, "checkpoint closure names unknown event %s", id)
		}
		closure[id] = true
		stack = append(stack, event.Predecessors...)
	}
	limit := 0
	for id := range closure {
		if position[id]+1 > limit {
			limit = position[id] + 1
		}
	}
	return closure, limit, nil
}

// checkChain validates the chain prefix below limit: session
// binding, registered schema versions, the Section 5.2 continuity
// rules (first event links exactly the record at sequence 1,
// same-lease sequence plus one with the prior head referenced, a
// successor lease restarts at 1), and the lease ordering (a lower
// epoch is stale, a same-epoch second lease diverges). It returns
// the validated prefix for the fold. Epoch gaps are a lease
// concern, not profile authority, and pass through.
func checkChain(record Record, events []Event, limit int) ([]Event, error) {
	if limit > len(events) {
		limit = len(events)
	}
	ordered := events[:limit]
	var current LeaseHead
	haveLease := false
	var tailID string
	var tailSeq uint64
	for index := range ordered {
		event := ordered[index]
		if event.SessionID != record.SessionID {
			return nil, refuse(ErrInvalidEvent, "event %s session %s does not belong to session %s", event.ID, event.SessionID, record.SessionID)
		}
		switch event.SchemaVersion {
		case "1.0.0", "2.0.0", "3.0.0", "4.0.0":
		default:
			return nil, refuse(ErrInvalidEvent, "event %s carries unregistered schema version %q", event.ID, event.SchemaVersion)
		}
		if event.Sequence == 0 {
			return nil, refuse(ErrInvalidTransition, "event %s carries no lease sequence", event.ID)
		}
		if len(event.Predecessors) == 0 {
			return nil, refuse(ErrInvalidTransition, "event %s names no predecessor", event.ID)
		}
		lease := LeaseHead{Epoch: event.LeaseEpoch, LeaseID: event.LeaseID}
		if !haveLease {
			if len(event.Predecessors) != 1 || event.Predecessors[0] != record.RecordID {
				return nil, refuse(ErrInvalidTransition, "first event %s must link exactly the session record", event.ID)
			}
			if event.Sequence != 1 {
				return nil, refuse(ErrInvalidTransition, "first event %s must open its lease at sequence 1", event.ID)
			}
			current, haveLease = lease, true
			tailID, tailSeq = event.ID, event.Sequence
			continue
		}
		switch compareLease(lease, current) {
		case 0:
			if event.Sequence != tailSeq+1 {
				if event.Sequence <= tailSeq {
					return nil, refuse(ErrInvalidTransition, "event %s repeats chained sequence through %d", event.ID, tailSeq)
				}
				return nil, refuse(ErrInvalidTransition, "event %s skips chained sequence %d", event.ID, tailSeq+1)
			}
		case 1:
			if lease.Epoch == current.Epoch {
				return nil, refuse(ErrDivergentLease, "event %s under lease %s diverges from chain head lease %s at epoch %d", event.ID, lease.LeaseID, current.LeaseID, current.Epoch)
			}
			if event.Sequence != 1 {
				return nil, refuse(ErrInvalidTransition, "successor-lease event %s must restart at sequence 1", event.ID)
			}
			current = lease
		default:
			return nil, refuse(ErrStaleLease, "event %s lease epoch %d precedes chain head epoch %d", event.ID, lease.Epoch, current.Epoch)
		}
		if !containsString(event.Predecessors, tailID) {
			return nil, refuse(ErrInvalidTransition, "event %s predecessors omit prior authoritative event", event.ID)
		}
		tailID, tailSeq = event.ID, event.Sequence
	}
	return ordered, nil
}

// foldChanges tracks the newest profile.changed event over the
// validated prefix, restricted to the closure when non-nil. Every
// other event type is inert. A change payload that no longer
// decodes refuses integrity_failure: chained bytes passed the
// closed shape at append time, so an undecodable member contradicts
// the attested chain. Only the target feeds the derivation; the
// confirmation history belongs to the publication flow that
// admitted the event.
func foldChanges(creation string, ordered []Event, closure map[string]bool) (Pair, error) {
	pair := Pair{Profile: creation}
	for index := range ordered {
		event := ordered[index]
		if closure != nil && !closure[event.ID] {
			continue
		}
		if event.Type != "profile.changed" {
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

// changeTarget decodes the target profile of one validated
// profile.changed event. The from/to/confirmed members mirror the
// closed shape the canonical owner admitted: both ends inside the
// vocabulary, from and to distinct, confirmed a boolean. The value
// of confirmed never affects authority here.
func changeTarget(event Event) (string, error) {
	from, ok := payloadString(event.Payload, "from")
	if !ok || (from != ProfileStandard && from != ProfileYOLO) {
		return "", refuse(ErrIntegrity, "event %s carries no valid change source for profile authority", event.ID)
	}
	target, ok := payloadString(event.Payload, "to")
	if !ok || (target != ProfileStandard && target != ProfileYOLO) {
		return "", refuse(ErrIntegrity, "event %s carries no valid change target for profile authority", event.ID)
	}
	if from == target {
		return "", refuse(ErrIntegrity, "event %s changes profile %q to itself for profile authority", event.ID, from)
	}
	if _, ok := payloadBool(event.Payload, "confirmed"); !ok {
		return "", refuse(ErrIntegrity, "event %s carries no confirmation for profile authority", event.ID)
	}
	return target, nil
}

// payloadString reads one string member from an admitted payload.
// The second result reports presence and type only; the caller
// attributes the refusal.
func payloadString(payload map[string]any, name string) (string, bool) {
	value, ok := payload[name]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

// payloadBool reads one boolean member from an admitted payload.
func payloadBool(payload map[string]any, name string) (bool, bool) {
	value, ok := payload[name]
	if !ok {
		return false, false
	}
	flag, ok := value.(bool)
	return flag, ok
}

// compareLease orders two lease heads by the Section 5.3 tuple
// rule: the greatest (epoch, lease_id) wins, where lease_id uses
// bytewise UUID order over canonical lowercase UUIDv4.
func compareLease(a, b LeaseHead) int {
	if a.Epoch != b.Epoch {
		if a.Epoch < b.Epoch {
			return -1
		}
		return 1
	}
	switch {
	case a.LeaseID < b.LeaseID:
		return -1
	case a.LeaseID > b.LeaseID:
		return 1
	}
	return 0
}

func containsString(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}
