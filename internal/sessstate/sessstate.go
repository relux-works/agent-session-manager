package sessstate

import (
	"errors"
	"fmt"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// SessionState spellings in SPEC.md Section 5.7 declared order. Every
// RPC and CLI field typed SessionState uses exactly this registry; the
// spellings created, starting, and quiesced are not session lifecycle
// states. The const block below is the single production source of the
// registry: no other production literal may name a state spelling, so
// the census derives the state space from these lines.
const (
	StateCreating      State = "creating"
	StateRunning       State = "running"
	StateIdle          State = "idle"
	StateQuiescing     State = "quiescing"
	StateCheckpointing State = "checkpointing"
	StateStopped       State = "stopped"
	StateMaterializing State = "materializing"
	StateParked        State = "parked"
	StateFailed        State = "failed"
	StateStale         State = "stale"
	StateTombstoned    State = "tombstoned"
)

// State is one Section 5.7 lifecycle value.
type State string

// States returns the SessionState registry in Section 5.7 declared
// order. It converts the const block above instead of retyping the
// spellings, so every state literal lives exactly once in production
// and the census can require that single source. Callers must not
// invent spellings beside this list.
func States() []string {
	return []string{
		string(StateCreating), string(StateRunning), string(StateIdle),
		string(StateQuiescing), string(StateCheckpointing),
		string(StateStopped), string(StateMaterializing),
		string(StateParked), string(StateFailed), string(StateStale),
		string(StateTombstoned),
	}
}

// allowedTransitions is the Section 5.7 transition table, the single
// source the step gate enforces. A same-state restatement (a confirming
// event while already in the target state) is not a transition and
// needs no edge; every other move needs exactly the edge below.
var allowedTransitions = map[State]map[State]struct{}{
	StateCreating: {
		StateRunning: {}, StateIdle: {}, StateFailed: {},
	},
	StateRunning: {
		StateIdle: {}, StateQuiescing: {}, StateFailed: {}, StateStale: {},
	},
	StateIdle: {
		StateRunning: {}, StateQuiescing: {}, StateCheckpointing: {},
		StateStopped: {}, StateStale: {},
	},
	StateQuiescing: {
		StateIdle: {}, StateCheckpointing: {}, StateFailed: {}, StateStale: {},
	},
	StateCheckpointing: {
		StateIdle: {}, StateStopped: {}, StateMaterializing: {}, StateFailed: {},
	},
	StateMaterializing: {
		StateStopped: {}, StateRunning: {}, StateParked: {}, StateFailed: {},
	},
	StateStopped: {
		StateMaterializing: {}, StateRunning: {}, StateParked: {}, StateTombstoned: {},
	},
	StateParked: {
		StateMaterializing: {}, StateRunning: {}, StateStopped: {}, StateStale: {},
	},
	StateFailed: {
		StateCreating: {}, StateStopped: {}, StateParked: {},
		StateMaterializing: {}, StateStale: {}, StateTombstoned: {},
	},
	StateStale: {
		StateStopped: {}, StateTombstoned: {},
	},
}

// initialTargets bounds the first event of a chain: a session opens in
// one of these states, and only through its bootstrap event. Any other
// first event is invalid_state_transition, not a silent default.
var initialTargets = map[State]struct{}{
	StateCreating: {}, StateRunning: {}, StateIdle: {}, StateFailed: {},
}

// staleSources bounds the local-stale override the same way the table
// bounds every other move into stale: the override applies only when
// the winner projection stands in a state the table lets go stale,
// otherwise the winner state stands with the lease conflict recorded.
var staleSources = map[State]struct{}{
	StateRunning: {}, StateIdle: {}, StateQuiescing: {},
	StateFailed: {}, StateParked: {},
}

var (
	ErrInvalidRecord     = errors.New("invalid session record")
	ErrInvalidEvent      = errors.New("invalid session event")
	ErrInvalidTransition = errors.New("invalid_state_transition")
	ErrIntegrity         = errors.New("integrity_failure")
	ErrDerivation        = errors.New("derivation input is corrupt")
	ErrUnknownSession    = errors.New("unknown session")
	ErrStaleLease        = errors.New("event lease epoch precedes the chain head")
	ErrDivergentLease    = errors.New("divergent lease branch cannot be authoritative")
)

// refuse is the single refusal funnel every sessstate rejection flows
// through. It is a var so the census TestMain can wrap it and record
// the production file:line behind each exercised refusal. No production
// path wraps a sentinel with fmt.Errorf directly.
var refuse = func(sentinel error, format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, arguments...))
}

// LeaseHead is one observed fencing token: the winning-lease pair
// carried by every owner-authored event and mutation.
type LeaseHead struct {
	Epoch   uint64
	LeaseID string
}

// IsZero reports an unset lease head: no local lease was supplied.
func (head LeaseHead) IsZero() bool {
	return head.Epoch == 0 && head.LeaseID == ""
}

// Compare orders two lease heads by the Section 5.3 tuple rule: the
// greatest (epoch, lease_id) wins, where lease_id uses bytewise UUID
// order. Lease IDs are canonical lowercase UUIDv4, so string order is
// bytewise order.
func Compare(a, b LeaseHead) int {
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

// Conflict kinds. A conflict is reported data, never an error: both
// sides were individually admittable, so the projection names the rule
// that resolved them.
const (
	// ConflictSameEpochTie ResolveRuleGreatestLeaseIDWins: two distinct
	// leases share the winning epoch (concurrent force takeovers under
	// partition); the bytewise-greater lease ID wins.
	ConflictSameEpochTie = "same_epoch_tie"
	// ConflictEpochGap ResolveRuleGreatestTupleWins: a successor epoch
	// skips its predecessor (epoch N to N+2 or more); the greatest
	// tuple still wins, and only an explicit takeover checkpoint heals
	// the gap.
	ConflictEpochGap = "epoch_gap"
	// ConflictLosingBranchPreserved ResolveRulePreservedNeverApplied:
	// an observed lease loses the tuple comparison; its history stays
	// preserved but never affects authoritative state.
	ConflictLosingBranchPreserved = "losing_branch_preserved"
	// ConflictProviderMismatch ResolveRuleRecordPinnedProviderWins:
	// a launch event names a provider other than the immutable Session
	// Record provider; the record value stays authoritative.
	ConflictProviderMismatch = "provider_mismatch"
	// ConflictUnionSupersedesChain
	// ResolveRuleAuthoritativeStateUnchanged: the union winner is not
	// among the authoritative chain leases; the reported winner moves
	// but derived state does not until an explicit takeover lands.
	ConflictUnionSupersedesChain = "union_supersedes_chain"
)

// Resolution rules named by conflicts.
const (
	ResolveRuleGreatestLeaseIDWins         = "greatest-lease-id-wins"
	ResolveRuleGreatestTupleWins           = "greatest-tuple-wins"
	ResolveRulePreservedNeverApplied       = "preserved-never-applied"
	ResolveRuleRecordPinnedProviderWins    = "record-pinned-provider-wins"
	ResolveRuleAuthoritativeStateUnchanged = "authoritative-state-unchanged"
)

// Conflict is one resolved lease or identity conflict.
type Conflict struct {
	Kind   string
	Rule   string
	Detail string
}

// Checkpoint is the newest validated checkpoint the projection knows.
type Checkpoint struct {
	ID      string
	Kind    string
	EventID string
}

// ProviderIdentity is the derived provider surface: the immutable
// record provider, the newest launched version, and the newest
// identified record with its confidence.
type ProviderIdentity struct {
	ID               string
	Version          string
	IdentityRecordID string
	Confidence       string
}

// TerminalBinding is the current terminal surface: the newest
// terminal.created or session.resumed binding. V1 bindings carry the
// backend with the terminal or native session handle; v4 bindings
// carry the binding digest with the backend and versions.
type TerminalBinding struct {
	Backend         string
	BindingID       string
	TerminalID      string
	NativeSessionID string
	EventID         string
}

// ParkedFacts carries the per-session parked channel from sessrepo
// into the projection: the recoverable_parked_state blocking reason
// and the same-operation retry.
type ParkedFacts struct {
	BlockingReason string
	RetryHint      string
}

// Record is the decoded Session Record surface the reducer reads.
type Record struct {
	SessionID  string
	RecordID   string
	Name       string
	Kind       string
	ProviderID string
}

// Event is one decoded authoritative Session Event: the routing
// members plus the closed payload object admitted by the canonical
// owner at decode time.
type Event struct {
	ID              string
	Type            string
	SchemaVersion   string
	SessionID       string
	CreatedByHostID string
	LeaseEpoch      uint64
	LeaseID         string
	Sequence        uint64
	Predecessors    []string
	Payload         map[string]any
}

// Input is the complete pure-reducer input: the record, the
// authoritative events in chain order, the competing off-chain leases
// observed at union, the lease the local host acts under (zero when
// unknown), the local host ID (empty when unknown), and the parked
// channel (non-nil when sessrepo reports the session parked).
type Input struct {
	Record      Record
	Events      []Event
	Union       []LeaseHead
	Local       LeaseHead
	LocalHostID string
	Parked      *ParkedFacts
}

// Projection is the derived session surface: lifecycle state, winning
// lease, conflicts, current checkpoint, provider identity, terminal
// binding, owner and role, the parked channel when set, and warnings.
type Projection struct {
	SessionID     string
	RecordID      string
	Name          string
	Kind          string
	State         State
	Winner        LeaseHead
	Conflicts     []Conflict
	HasCheckpoint bool
	Newest        Checkpoint
	Provider      ProviderIdentity
	HasTerminal   bool
	Terminal      TerminalBinding
	OwnerHostID   string
	LocalRole     string
	Parked        *ParkedFacts
	Warnings      []string
}

// Reduce folds one record with its authoritative events into a
// Projection. It is pure: no I/O, no clock, no cache. Chain continuity
// is re-checked over the input, so a chain-forbidden reordering
// refuses with invalid_state_transition instead of silently deriving a
// different state.
func Reduce(input Input) (Projection, error) {
	projection := Projection{
		SessionID: input.Record.SessionID,
		RecordID:  input.Record.RecordID,
		Name:      input.Record.Name,
		Kind:      input.Record.Kind,
		Provider:  ProviderIdentity{ID: input.Record.ProviderID},
	}
	if input.Parked != nil {
		projection.State = StateParked
		projection.Parked = &ParkedFacts{
			BlockingReason: input.Parked.BlockingReason,
			RetryHint:      input.Parked.RetryHint,
		}
		projection.Warnings = []string{"parked: " + input.Parked.BlockingReason}
		return projection, nil
	}
	if input.Record.SessionID == "" || input.Record.RecordID == "" {
		return Projection{}, refuse(ErrInvalidRecord, "reducer input carries no session identity")
	}
	fold := &chainFold{
		recordID:       input.Record.RecordID,
		recordProvider: input.Record.ProviderID,
		projection:     &projection,
	}
	for index := range input.Events {
		if err := fold.apply(input.Events[index]); err != nil {
			return Projection{}, err
		}
	}
	if len(input.Events) == 0 {
		// An empty chain derives creating, but the union still
		// passes through resolveWinner below: a malformed union
		// refuses here exactly as on a non-empty chain, and a
		// well-formed union moves only the reported winner while
		// authoritative state stays creating.
		fold.state = StateCreating
	}
	if err := fold.resolveWinner(input.Union); err != nil {
		return Projection{}, err
	}
	fold.applyLocalLease(input.Local, input.LocalHostID)
	fold.emitWarnings()
	return projection, nil
}

// chainFold carries the mutable fold state across one Reduce call. It
// never escapes Reduce.
type chainFold struct {
	recordID       string
	recordProvider string
	projection     *Projection
	state          State
	current        LeaseHead
	createLease    LeaseHead
	haveLease      bool
	seenCheckpoint bool
	seenAbort      bool
	onChain        map[string]struct{}
	tailID         string
	tailSeq        uint64
	leases         []LeaseHead
	previous       LeaseHead
}

// apply folds one event: continuity first, then the type effect.
func (fold *chainFold) apply(event Event) error {
	if err := fold.checkContinuity(event); err != nil {
		return err
	}
	if event.SchemaVersion != "1.0.0" && event.SchemaVersion != "2.0.0" &&
		event.SchemaVersion != "3.0.0" && event.SchemaVersion != "4.0.0" {
		return refuse(ErrInvalidEvent, "event %s carries unregistered schema version %q", event.ID, event.SchemaVersion)
	}
	if event.SessionID != fold.projection.SessionID {
		return refuse(ErrInvalidEvent, "event %s session %s does not belong to session %s", event.ID, event.SessionID, fold.projection.SessionID)
	}
	lease := LeaseHead{Epoch: event.LeaseEpoch, LeaseID: event.LeaseID}
	succession, err := fold.trackLease(event, lease)
	if err != nil {
		return err
	}
	if succession {
		if err := fold.successionMove(what(event)); err != nil {
			return err
		}
	}
	fold.projection.OwnerHostID = event.CreatedByHostID
	return fold.effect(event, lease, succession)
}

// what names one event for refusal evidence.
func what(event Event) string {
	return "event " + event.Type + " " + event.ID
}

// successionMove folds a successor lease arrival before the event's own
// effect: a takeover while the winner is active stales the old host's
// projection under the force-takeover rule, and from a checkpoint, a
// stop, a failure, or a park the destination stages into
// materializing. The move composes through step, so the arriving
// event's own effect still applies after it.
func (fold *chainFold) successionMove(name string) error {
	switch fold.state {
	case StateRunning, StateIdle, StateQuiescing:
		return fold.step(StateStale, name)
	default:
		return fold.step(StateMaterializing, name)
	}
}

// checkContinuity re-enforces the Section 5.2 chain rules over the
// reducer input: the first event links exactly the Session Record at
// sequence 1, within one lease the sequence increases by exactly one
// and each event references the prior authoritative event, and a
// successor lease restarts at 1 referencing the prior head. A
// violation is invalid_state_transition: the input is not a chain the
// repository could have authored.
func (fold *chainFold) checkContinuity(event Event) error {
	if event.Sequence == 0 {
		return refuse(ErrInvalidTransition, "event %s carries no lease sequence", event.ID)
	}
	if len(event.Predecessors) == 0 {
		return refuse(ErrInvalidTransition, "event %s names no predecessor", event.ID)
	}
	if fold.onChain == nil {
		if len(event.Predecessors) != 1 || event.Predecessors[0] != fold.recordID {
			return refuse(ErrInvalidTransition, "first event %s must link exactly the session record", event.ID)
		}
		if event.Sequence != 1 {
			return refuse(ErrInvalidTransition, "first event %s must open its lease at sequence 1", event.ID)
		}
		fold.onChain = map[string]struct{}{event.ID: {}}
		return nil
	}
	tail := fold.tailEventID()
	if event.LeaseID == fold.current.LeaseID && event.LeaseEpoch == fold.current.Epoch {
		if event.Sequence != fold.tailSequence()+1 {
			if event.Sequence <= fold.tailSequence() {
				return refuse(ErrInvalidTransition, "event %s repeats chained sequence through %d", event.ID, fold.tailSequence())
			}
			return refuse(ErrInvalidTransition, "event %s skips chained sequence %d", event.ID, fold.tailSequence()+1)
		}
	} else {
		if event.Sequence != 1 {
			return refuse(ErrInvalidTransition, "successor-lease event %s must restart at sequence 1", event.ID)
		}
	}
	if !containsString(event.Predecessors, tail) {
		return refuse(ErrInvalidTransition, "event %s predecessors omit prior authoritative event", event.ID)
	}
	fold.onChain[event.ID] = struct{}{}
	return nil
}

// tail tracking: the fold remembers the last applied event ID and its
// sequence alongside the current lease.
func (fold *chainFold) trackTail(event Event) {
	fold.tailID = event.ID
	fold.tailSeq = event.Sequence
}

func (fold *chainFold) tailEventID() string { return fold.tailID }

func (fold *chainFold) tailSequence() uint64 { return fold.tailSeq }

// trackLease observes the envelope lease: same-lease events continue,
// a greater tuple succeeds (recording an epoch gap past plus one), a
// same-epoch second lease diverges, and a lower epoch is stale. The
// chain cannot hold the losing cases, so they refuse here for direct
// Reduce inputs; the union path reports them as conflicts instead.
func (fold *chainFold) trackLease(event Event, lease LeaseHead) (bool, error) {
	if !fold.haveLease {
		fold.current = lease
		fold.createLease = lease
		fold.haveLease = true
		fold.leases = append(fold.leases, lease)
		fold.trackTail(event)
		return false, nil
	}
	switch Compare(lease, fold.current) {
	case 0:
		fold.trackTail(event)
		return false, nil
	case 1:
		if lease.Epoch == fold.current.Epoch {
			return false, refuse(ErrDivergentLease, "event %s under lease %s diverges from chain head lease %s at epoch %d", event.ID, lease.LeaseID, fold.current.LeaseID, fold.current.Epoch)
		}
		if lease.Epoch > fold.current.Epoch+1 {
			fold.conflict(ConflictEpochGap, ResolveRuleGreatestTupleWins,
				"lease epoch jumps from %d to %d at event %s; only an explicit takeover checkpoint heals the gap",
				fold.current.Epoch, lease.Epoch, event.ID)
		}
		fold.previous = fold.current
		fold.current = lease
		fold.leases = append(fold.leases, lease)
		fold.trackTail(event)
		return true, nil
	default:
		return false, refuse(ErrStaleLease, "event %s lease epoch %d precedes chain head epoch %d", event.ID, lease.Epoch, fold.current.Epoch)
	}
}

// tailID and tailSeq live beside the fold; declared here to keep the
// struct literal above readable.
func (fold *chainFold) conflict(kind, rule, format string, arguments ...any) {
	fold.projection.Conflicts = append(fold.projection.Conflicts, Conflict{
		Kind:   kind,
		Rule:   rule,
		Detail: fmt.Sprintf(format, arguments...),
	})
}

// step moves the derivation through the Section 5.7 table. An empty
// target keeps the state (a fact or inert event performs no
// transition); an equal target restates it (a confirming event while
// already there is not a transition); the first event may only open a
// bootstrap state; every other move needs its table edge, else
// invalid_state_transition.
func (fold *chainFold) step(target State, what string) error {
	if target == "" || target == fold.state {
		return nil
	}
	if fold.state == "" {
		if _, ok := initialTargets[target]; !ok {
			return refuse(ErrInvalidTransition, "%s opens unbootstrapped state %q", what, string(target))
		}
		fold.state = target
		return nil
	}
	if _, ok := allowedTransitions[fold.state][target]; !ok {
		return refuse(ErrInvalidTransition, "%s moves %q to %q without a Section 5.7 edge", what, string(fold.state), string(target))
	}
	fold.state = target
	return nil
}

// effect applies one event's lifecycle meaning after continuity and
// lease tracking. Fact events return an empty target and keep the
// state; unknown v1 types and the v2/v3/v4 non-lifecycle types are
// inert by Section 5.2 (retained, never applied).
func (fold *chainFold) effect(event Event, lease LeaseHead, succession bool) error {
	name := "event " + event.Type + " " + event.ID
	switch event.Type {
	case "session.created":
		return fold.effectCreated(event, name)
	case "terminal.created":
		return fold.effectTerminalCreated(event, name)
	case "provider.launched":
		return fold.effectProviderLaunched(event, name)
	case "provider.identified":
		return fold.effectProviderIdentified(event, name)
	case "session.idle":
		return fold.step(StateIdle, name)
	case "session.quiescing":
		return fold.step(StateQuiescing, name)
	case "checkpoint.created":
		return fold.effectCheckpointCreated(event, name)
	case "sync.completed":
		return nil
	case "session.stopped":
		return fold.effectStopped(event, name)
	case "session.resumed":
		return fold.effectResumed(event, name)
	case "session.bootstrap_aborted":
		return fold.effectBootstrapAborted(event, name)
	case "lease.transferred", "lease.forced":
		return fold.effectLease(event, lease, name, succession)
	case "session.parked":
		return fold.step(StateParked, name)
	case "session.failed":
		return fold.step(StateFailed, name)
	case "fork.created", "profile.changed", "takeover.force_confirmed",
		"replica.replace_confirmed", "task_board.adopted",
		"tombstone.issued", "tombstone.resolved":
		return nil
	case "session.tombstoned":
		return fold.step(StateTombstoned, name)
	case "task_board.launched":
		return fold.effectTaskBoardLaunched(event, name)
	default:
		return nil
	}
}

// effectCreated opens the bootstrap, including the Section 5.7
// failed-to-creating retry: epoch 1 under the same winning create
// lease, no validated checkpoint yet, and no authoritative bootstrap
// abort on record. Anything else that reaches for creating refuses.
func (fold *chainFold) effectCreated(event Event, what string) error {
	if fold.state == "" {
		return fold.step(StateCreating, what)
	}
	if fold.state != StateFailed {
		return fold.step(StateCreating, what)
	}
	if event.LeaseEpoch != 1 || event.LeaseID != fold.createLease.LeaseID ||
		event.LeaseEpoch != fold.createLease.Epoch {
		return refuse(ErrInvalidTransition, "%s retries bootstrap under a new lease", what)
	}
	if fold.seenCheckpoint {
		return refuse(ErrInvalidTransition, "%s retries bootstrap past a validated checkpoint", what)
	}
	if fold.seenAbort {
		return refuse(ErrInvalidTransition, "%s retries bootstrap past an authoritative abort", what)
	}
	return fold.step(StateCreating, what)
}

// effectTerminalCreated records the terminal binding; as the first
// event it also opens the bootstrap in creating.
func (fold *chainFold) effectTerminalCreated(event Event, what string) error {
	if event.SchemaVersion == "4.0.0" {
		binding, err := payloadString(event.Payload, "terminal_binding_id")
		if err != nil {
			return err
		}
		if _, err := scalar.ParseDigest(binding); err != nil {
			return refuse(ErrDerivation, "%s carries malformed terminal_binding_id: %v", what, err)
		}
		backend, err := payloadString(event.Payload, "terminal_backend_id")
		if err != nil {
			return err
		}
		if _, err := terminalbackend.ParseID(backend); err != nil {
			return refuse(ErrDerivation, "%s carries refused terminal backend: %v", what, err)
		}
		fold.projection.Terminal = TerminalBinding{Backend: backend, BindingID: binding, EventID: event.ID}
	} else {
		backend, err := payloadString(event.Payload, "backend")
		if err != nil {
			return err
		}
		if backend != "tmux" && backend != "conpty" {
			return refuse(ErrDerivation, "%s carries unknown terminal backend %q", what, backend)
		}
		terminalID, err := payloadString(event.Payload, "terminal_id")
		if err != nil {
			return err
		}
		fold.projection.Terminal = TerminalBinding{Backend: backend, TerminalID: terminalID, EventID: event.ID}
	}
	fold.projection.HasTerminal = true
	if fold.state == "" {
		return fold.step(StateCreating, what)
	}
	return nil
}

// effectProviderLaunched moves to running and records the launched
// version. The record provider stays authoritative: a launched event
// naming another provider records a provider_mismatch conflict under
// the record-pinned-provider-wins rule instead of rewriting identity.
func (fold *chainFold) effectProviderLaunched(event Event, what string) error {
	providerID, err := payloadString(event.Payload, "provider_id")
	if err != nil {
		return err
	}
	version, err := payloadString(event.Payload, "provider_version")
	if err != nil {
		return err
	}
	if providerID != fold.recordProvider {
		fold.conflict(ConflictProviderMismatch, ResolveRuleRecordPinnedProviderWins,
			"event %s launches provider %q under record provider %q; the record value stays authoritative",
			event.ID, providerID, fold.recordProvider)
	} else {
		fold.projection.Provider.Version = version
	}
	return fold.step(StateRunning, what)
}

// effectProviderIdentified records the newest identity record with
// its confidence.
func (fold *chainFold) effectProviderIdentified(event Event, what string) error {
	recordID, err := payloadString(event.Payload, "provider_identity_record_id")
	if err != nil {
		return err
	}
	if _, err := scalar.ParseDigest(recordID); err != nil {
		return refuse(ErrDerivation, "%s carries malformed provider_identity_record_id: %v", what, err)
	}
	confidence, err := payloadString(event.Payload, "confidence")
	if err != nil {
		return err
	}
	fold.projection.Provider.IdentityRecordID = recordID
	fold.projection.Provider.Confidence = confidence
	return nil
}

// effectCheckpointCreated records the newest validated checkpoint and
// moves idle or quiescing into checkpointing, or checkpointing back to
// idle at the validated boundary. Any other lifecycle position refuses:
// a checkpoint cannot land while running, creating, stopped, parked,
// failed, stale, materializing, or tombstoned without the quiesce or
// resume the table requires.
func (fold *chainFold) effectCheckpointCreated(event Event, what string) error {
	checkpointID, err := payloadString(event.Payload, "checkpoint_id")
	if err != nil {
		return err
	}
	if _, err := scalar.ParseDigest(checkpointID); err != nil {
		return refuse(ErrDerivation, "%s carries malformed checkpoint_id: %v", what, err)
	}
	kind, err := payloadString(event.Payload, "kind")
	if err != nil {
		return err
	}
	fold.noteCheckpoint(checkpointID, kind, event.ID)
	switch fold.state {
	case "", StateIdle, StateQuiescing:
		if fold.state == "" {
			return refuse(ErrInvalidTransition, "%s publishes a checkpoint before any bootstrap", what)
		}
		return fold.step(StateCheckpointing, what)
	case StateCheckpointing:
		return fold.step(StateIdle, what)
	default:
		return fold.step(StateCheckpointing, what)
	}
}

// effectStopped derives stopped for a checkpointed closure and failed
// for a bootstrap abort. No event may derive stopped while the newest
// checkpoint is null: the event's own checkpoint counts, so a
// checkpointed stop carries its resumable base with it. A bootstrap
// abort requires the null checkpoint, false resumable and graceful
// values the canonical owner admitted, plus a preceding authoritative
// abort event; an ambiguous live process stays failed with recovery
// diagnostics instead.
func (fold *chainFold) effectStopped(event Event, what string) error {
	checkpointID, hasCheckpoint, err := payloadNullableDigest(event.Payload, "checkpoint_id")
	if err != nil {
		return err
	}
	resumable, err := payloadBool(event.Payload, "resumable")
	if err != nil {
		return err
	}
	closure, err := payloadString(event.Payload, "closure_kind")
	if err != nil {
		return err
	}
	if closure == "checkpointed" {
		if !hasCheckpoint || !resumable {
			return refuse(ErrIntegrity, "%s closes checkpointed without a resumable checkpoint", what)
		}
		fold.noteCheckpoint(checkpointID, "", event.ID)
		return fold.step(StateStopped, what)
	}
	if hasCheckpoint || resumable {
		return refuse(ErrIntegrity, "%s closes bootstrap_abort with a checkpoint or resumable set", what)
	}
	if !fold.seenAbort {
		return refuse(ErrInvalidTransition, "%s closes bootstrap_abort with no authoritative abort event", what)
	}
	return fold.step(StateFailed, what)
}

// effectResumed moves to running under the resumed checkpoint and
// records the terminal binding the resume selected.
func (fold *chainFold) effectResumed(event Event, what string) error {
	checkpointID, err := payloadString(event.Payload, "checkpoint_id")
	if err != nil {
		return err
	}
	if _, err := scalar.ParseDigest(checkpointID); err != nil {
		return refuse(ErrDerivation, "%s carries malformed checkpoint_id: %v", what, err)
	}
	fold.noteCheckpoint(checkpointID, "", event.ID)
	if event.SchemaVersion == "4.0.0" {
		binding, err := payloadString(event.Payload, "terminal_binding_id")
		if err != nil {
			return err
		}
		if _, err := scalar.ParseDigest(binding); err != nil {
			return refuse(ErrDerivation, "%s carries malformed terminal_binding_id: %v", what, err)
		}
		backend, err := payloadString(event.Payload, "terminal_backend_id")
		if err != nil {
			return err
		}
		if _, err := terminalbackend.ParseID(backend); err != nil {
			return refuse(ErrDerivation, "%s carries refused terminal backend: %v", what, err)
		}
		fold.projection.Terminal = TerminalBinding{Backend: backend, BindingID: binding, EventID: event.ID}
	} else {
		backend, err := payloadString(event.Payload, "terminal_backend")
		if err != nil {
			return err
		}
		if backend != "tmux" && backend != "conpty" {
			return refuse(ErrDerivation, "%s carries unknown terminal backend %q", what, backend)
		}
		nativeID, err := payloadString(event.Payload, "native_session_id")
		if err != nil {
			return err
		}
		fold.projection.Terminal = TerminalBinding{Backend: backend, NativeSessionID: nativeID, EventID: event.ID}
	}
	fold.projection.HasTerminal = true
	return fold.step(StateRunning, what)
}

// effectBootstrapAborted records the authoritative abort and fails the
// bootstrap. Only the creating session can abort its bootstrap; any
// later abort is invalid_state_transition.
func (fold *chainFold) effectBootstrapAborted(event Event, what string) error {
	fold.seenAbort = true
	if fold.state != "" && fold.state != StateCreating {
		return refuse(ErrInvalidTransition, "%s aborts a bootstrap past creating", what)
	}
	return fold.step(StateFailed, what)
}

// effectLease checks the successor-lease payload linkage: the payload
// must name the predecessor the chain stands on and the envelope it
// arrives under, else integrity_failure. The lifecycle move already
// happened in successionMove; a lease event repeating the current
// lease is a fact, not a transition.
func (fold *chainFold) effectLease(event Event, lease LeaseHead, what string, succession bool) error {
	if !succession {
		return nil
	}
	if event.Type == "lease.transferred" {
		predecessor, err := payloadString(event.Payload, "predecessor_lease_id")
		if err != nil {
			return err
		}
		if predecessor != fold.previous.LeaseID {
			return refuse(ErrIntegrity, "%s transfers from lease %s past chain head lease %s", what, predecessor, fold.previous.LeaseID)
		}
		successor, err := payloadString(event.Payload, "new_lease_id")
		if err != nil {
			return err
		}
		if successor != lease.LeaseID {
			return refuse(ErrIntegrity, "%s transfers to lease %s outside its envelope lease %s", what, successor, lease.LeaseID)
		}
	} else {
		successor, err := payloadString(event.Payload, "new_lease_id")
		if err != nil {
			return err
		}
		if successor != lease.LeaseID {
			return refuse(ErrIntegrity, "%s forces lease %s outside its envelope lease %s", what, successor, lease.LeaseID)
		}
	}
	return nil
}

// effectTaskBoardLaunched enforces the Section 5.2 repeat rule — the
// event must repeat the winning creation lease and the Session
// Record's provider and launch mode — then derives the payload's
// running or idle state through the table.
func (fold *chainFold) effectTaskBoardLaunched(event Event, what string) error {
	providerID, err := payloadString(event.Payload, "provider_id")
	if err != nil {
		return err
	}
	if providerID != fold.recordProvider {
		return refuse(ErrIntegrity, "%s repeats provider %q past record provider %q", what, providerID, fold.recordProvider)
	}
	payloadEpoch, err := payloadUint(event.Payload, "lease_epoch")
	if err != nil {
		return err
	}
	payloadLease, err := payloadString(event.Payload, "lease_id")
	if err != nil {
		return err
	}
	if payloadEpoch != fold.createLease.Epoch || payloadLease != fold.createLease.LeaseID {
		return refuse(ErrIntegrity, "%s repeats creation lease (%d, %s) past create lease (%d, %s)", what, payloadEpoch, payloadLease, fold.createLease.Epoch, fold.createLease.LeaseID)
	}
	reported, err := payloadString(event.Payload, "state")
	if err != nil {
		return err
	}
	if reported == string(StateRunning) {
		return fold.step(StateRunning, what)
	}
	return fold.step(StateIdle, what)
}

// noteCheckpoint records the newest validated checkpoint.
func (fold *chainFold) noteCheckpoint(id, kind, eventID string) {
	fold.seenCheckpoint = true
	fold.projection.HasCheckpoint = true
	fold.projection.Newest = Checkpoint{ID: id, Kind: kind, EventID: eventID}
}

// resolveWinner selects the winning lease over the authoritative chain
// leases and the union leases by the greatest (epoch, lease_id)
// tuple. A same-epoch tie resolves to the bytewise-greater lease ID;
// an off-chain union winner moves the reported winner but never
// rewrites authoritative state; union leases that lose are reported
// preserved-never-applied. With no union the chain head wins alone
// and no conflict is recorded. The reported projection is a pure
// function of the union multiset: every permutation and every
// partition into successive unions yields the identical winner,
// conflicts, and warnings.
func (fold *chainFold) resolveWinner(union []LeaseHead) error {
	winner := fold.current
	onChain := map[string]struct{}{}
	if fold.haveLease {
		onChain[fold.current.LeaseID] = struct{}{}
	}
	for _, lease := range fold.chainLeases() {
		onChain[lease.LeaseID] = struct{}{}
	}
	byEpoch := map[uint64]map[string]struct{}{}
	note := func(lease LeaseHead) {
		if byEpoch[lease.Epoch] == nil {
			byEpoch[lease.Epoch] = map[string]struct{}{}
		}
		byEpoch[lease.Epoch][lease.LeaseID] = struct{}{}
	}
	for _, lease := range fold.chainLeases() {
		note(lease)
	}
	// First pass: validate every union entry and select the greatest
	// tuple. The winner depends only on the multiset, never on
	// arrival order.
	for _, lease := range union {
		if lease.IsZero() {
			return refuse(ErrInvalidEvent, "union carries an empty lease head")
		}
		if _, err := scalar.ParseUUIDv4(lease.LeaseID); err != nil {
			return refuse(ErrInvalidEvent, "union lease %q is not a UUIDv4: %v", lease.LeaseID, err)
		}
		note(lease)
		if Compare(lease, winner) > 0 {
			winner = lease
		}
	}
	// Second pass: every union entry strictly below the final winner
	// is reported preserved-never-applied in ascending tuple order,
	// naming the final winner. Reporting each loss against the
	// transient arrival-order winner instead would make the conflict
	// evidence — even its presence — depend on union order. A
	// duplicate of the winning tuple is neither greater nor lesser,
	// so it reports nothing and the off-chain flag below stands.
	losing := make([]LeaseHead, 0, len(union))
	for _, lease := range union {
		if Compare(lease, winner) < 0 {
			losing = append(losing, lease)
		}
	}
	sort.Slice(losing, func(i, j int) bool { return Compare(losing[i], losing[j]) < 0 })
	for _, lease := range losing {
		fold.conflict(ConflictLosingBranchPreserved, ResolveRulePreservedNeverApplied,
			"union lease (%d, %s) loses to (%d, %s); its history stays preserved and never applies",
			lease.Epoch, lease.LeaseID, winner.Epoch, winner.LeaseID)
	}
	epochs := make([]uint64, 0, len(byEpoch))
	for epoch := range byEpoch {
		epochs = append(epochs, epoch)
	}
	sort.Slice(epochs, func(i, j int) bool { return epochs[i] < epochs[j] })
	for _, epoch := range epochs {
		rivals := byEpoch[epoch]
		if len(rivals) < 2 {
			continue
		}
		ordered := make([]string, 0, len(rivals))
		for id := range rivals {
			ordered = append(ordered, id)
		}
		sort.Strings(ordered)
		fold.conflict(ConflictSameEpochTie, ResolveRuleGreatestLeaseIDWins,
			"epoch %d is shared by leases %v; the bytewise-greater lease ID %s wins",
			epoch, ordered, ordered[len(ordered)-1])
	}
	// The off-chain flag is recomputed from the final winner rather
	// than tracked through the arrival order: it is set exactly when
	// the winning tuple stands outside the authoritative chain and
	// arrived in the union. A repeated winning entry keeps the
	// conflict recorded (and its divergent_history warning raised).
	offChainWinner := false
	if _, ok := onChain[winner.LeaseID]; !ok {
		for _, lease := range union {
			if lease == winner {
				offChainWinner = true
				break
			}
		}
	}
	if offChainWinner {
		fold.conflict(ConflictUnionSupersedesChain, ResolveRuleAuthoritativeStateUnchanged,
			"union winner (%d, %s) stands outside the authoritative chain; derived state is unchanged until an explicit takeover lands",
			winner.Epoch, winner.LeaseID)
	}
	fold.projection.Winner = winner
	return nil
}

// chainLeases replays the distinct envelope leases in first-appearance
// order. Every succession appends, so a tie planted at any past epoch
// still resolves against the lease the chain actually stood on. Gaps
// were already recorded at succession time.
func (fold *chainFold) chainLeases() []LeaseHead {
	return append([]LeaseHead(nil), fold.leases...)
}

// applyLocalLease derives the local projection: the owner host of the
// newest authoritative event decides the local role against the
// caller-supplied host, and a local lease that lost the tuple rule
// stales the projection exactly where the table lets it go stale.
func (fold *chainFold) applyLocalLease(local LeaseHead, localHostID string) {
	if localHostID != "" && fold.projection.OwnerHostID != "" {
		if localHostID == fold.projection.OwnerHostID {
			fold.projection.LocalRole = "owner"
		} else {
			fold.projection.LocalRole = "replica"
		}
	}
	if local.IsZero() || !fold.haveLease {
		return
	}
	// Only a lease that lost the tuple rule stales the projection:
	// an equal or strictly greater local lease stands with the
	// winner instead of reporting stale.
	if Compare(local, fold.projection.Winner) >= 0 {
		return
	}
	if _, ok := staleSources[fold.state]; !ok {
		return
	}
	fold.state = StateStale
	fold.projection.State = StateStale
}

// emitWarnings publishes the sorted unique warnings this leaf derives:
// divergent history behind any lease or provider conflict, a stale
// local process, and the parked blocking reason. Capability,
// authentication, and sync-age warnings belong to other leaves.
func (fold *chainFold) emitWarnings() {
	fold.projection.State = fold.state
	warnings := map[string]struct{}{}
	for _, conflict := range fold.projection.Conflicts {
		switch conflict.Kind {
		case ConflictSameEpochTie, ConflictEpochGap, ConflictLosingBranchPreserved,
			ConflictUnionSupersedesChain, ConflictProviderMismatch:
			warnings["divergent_history"] = struct{}{}
		}
	}
	if fold.state == StateStale {
		warnings["stale_process"] = struct{}{}
	}
	if fold.projection.Parked != nil {
		warnings["parked: "+fold.projection.Parked.BlockingReason] = struct{}{}
	}
	if len(warnings) == 0 {
		return
	}
	ordered := make([]string, 0, len(warnings))
	for warning := range warnings {
		ordered = append(ordered, warning)
	}
	sort.Strings(ordered)
	fold.projection.Warnings = ordered
}

func containsString(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}
