// Immutable SelectionPlan construction and current-fact revalidation
// for the v0.6.0 selector contract (SPEC Section 14.7.2).
//
// Resolve the key to a session UUID once per invocation or confirmed
// plan, then bind the selection facts into a machine-local immutable
// plan: the literal selector, the selected session and its validated
// immutable record digest, the selected source (host and exact alias),
// the digests of the complete validated index read and the effective
// validated configuration, the validated winning lease facts, the
// validated authority heads, and the caller-declared action,
// destination, and expectations. No plan hash, name, source, or display
// field is a capability. All sixteen contract members bind; a future
// member without a bound row fails its publication gate closed instead
// of falling into a default.
//
// Lease authority comes from validated urn:ax:schema:lease records
// supplied in Reader.LeaseRecords and admitted through the
// canonicaljson owner. The plan binds lease_record_id as the canonical
// digest of the winning Lease Record for the selected session, and the
// lease triple binds from that same record. No envelope observation
// substitutes for the record: a session with no admitted winning
// record refuses selector_observation_unavailable, and malformed bytes
// refuse invalid_config. A complete read proving a record without any
// valid lease refuses selector_bootstrap_incomplete instead of binding
// an empty triple, and a local selection without a known local host
// refuses invalid_config instead of binding an empty source host.
//
// Revalidation rereads the current configuration and the plan's source,
// validates the same immutable record, the current winning lease, and
// the action prerequisites this leaf can check, compares every bound
// plan fact in a fixed order, and then validates the complete
// authority union for the pinned session UUID across the local and
// every allowlisted source: a contradictory immutable record is
// integrity failure, a contradictory winning lease or fresh tombstone
// evidence is stale. A changed mapping, alias, allowlist, transport,
// source index, record, authority head, lease, action, destination, or
// expectation invalidates the plan with selector_plan_stale naming the
// first diverged member; explicit authorization revocation returns
// peer_not_allowlisted; failed reads retain their read-failure
// classification. Revalidation never re-resolves by name, substitutes
// a same-named session, silently replans, transfers a lease, or reuses
// confirmation for new facts.
//
// Effect authorization beyond read projection — fencing, commit,
// retry, recovery, remote dispatch/admission, and transport-resume
// prerequisites, and the composed per-endpoint invocation bindings —
// belongs to the CLI and lifecycle owners, which invoke Revalidate at
// every boundary in BoundariesFor for their action. A plan built here
// authorizes read projection only.
package sessquery

import (
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// SelectorVersion is the exact selector contract version every plan binds.
const SelectorVersion = "1.0.0"

// Plan actions admitted by the Section 14.7.2 boundary matrix. Each is
// an existing command/action tag; binding one selects it, never an
// umbrella bypass.
const (
	ActionStatus      = "status"
	ActionAttach      = "attach"
	ActionTakeover    = "takeover"
	ActionFork        = "fork"
	ActionStop        = "stop"
	ActionResume      = "resume"
	ActionSync        = "sync"
	ActionDiff        = "diff"
	ActionMaterialize = "materialize"
	ActionSetProfile  = "session.set-profile"
	ActionLogs        = "logs"
	ActionCancel      = "cancel"
)

// Plan boundaries from the normative action/boundary matrix. A boundary
// is reached only when the existing operation reaches that phase; these
// rows introduce no new effects.
const (
	BoundaryProjection      = "projection"
	BoundaryPreEffect       = "pre-effect"
	BoundaryFencing         = "fencing"
	BoundaryCommit          = "commit"
	BoundaryRetry           = "retry"
	BoundaryRecovery        = "recovery"
	BoundaryRemoteDispatch  = "remote-dispatch"
	BoundaryRemoteAdmission = "remote-admission"
	BoundaryTransportResume = "transport-resume"
)

// planBoundaries is the closed action/boundary matrix: every action has
// exactly its normative row, and an unknown action has none.
var planBoundaries = map[string][]string{
	ActionStatus:      {BoundaryProjection, BoundaryRetry},
	ActionAttach:      {BoundaryPreEffect, BoundaryFencing, BoundaryRetry, BoundaryRemoteDispatch, BoundaryRemoteAdmission, BoundaryTransportResume},
	ActionTakeover:    {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
	ActionFork:        {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
	ActionStop:        {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
	ActionResume:      {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
	ActionSync:        {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
	ActionDiff:        {BoundaryProjection, BoundaryRetry},
	ActionMaterialize: {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery},
	ActionSetProfile:  {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery},
	ActionLogs:        {BoundaryProjection, BoundaryRetry, BoundaryRemoteDispatch, BoundaryRemoteAdmission, BoundaryTransportResume},
	ActionCancel:      {BoundaryProjection},
}

// BoundariesFor returns the required boundary classes for a plan
// action. The boolean reports whether the action is a known tag; an
// unknown action binds no boundary and builds no plan.
func BoundariesFor(action string) ([]string, bool) {
	boundaries, ok := planBoundaries[action]
	if !ok {
		return nil, false
	}
	return append([]string(nil), boundaries...), true
}

// SelectionPlan is the machine-local immutable binding of one selector
// resolution. All members are values, never references: there is no
// mutator, and Marshal carries the locally verified canonical digest
// so a persisted plan is tamper-evident. Caller-supplied digest
// equality alone is never evidence of authority; only Revalidate
// against current facts decides currency.
type SelectionPlan struct {
	// SelectorVersion is exactly SelectorVersion.
	SelectorVersion string
	// Selector is the original single argument, at most
	// MaxSelectorRunes characters.
	Selector string
	// SessionID is the selected UUIDv7, never resolved again by name.
	SessionID string
	// SessionRecordID is the validated immutable Session Record digest.
	SessionRecordID string
	// SourceHostID is the UUIDv7 of the selected source, always bound:
	// a local selection without a known local host refuses
	// invalid_config at build time instead of binding empty.
	SourceHostID string
	// SourceAlias is the exact configured alias, or nil for local. It
	// is a diagnostic label, never identity.
	SourceAlias *string
	// SourceIndexDigest is the digest of the complete validated index
	// read used for selection.
	SourceIndexDigest string
	// ConfigurationDigest is the digest of the effective validated
	// configuration, including peer mapping, allowlist, and transport.
	ConfigurationDigest string
	// LeaseRecordID is the canonical digest of the validated winning
	// Lease Record for the selected session, admitted through the
	// canonicaljson owner from Reader.LeaseRecords.
	LeaseRecordID string
	// LeaseEpoch, LeaseID, and OwnerHostID come from that same winning
	// Lease Record: positive uint53, UUIDv4, and UUIDv7 holder. A plan
	// with no admitted winning record is never built; the builder
	// refuses selector_observation_unavailable, and a complete read
	// proving a record without any valid lease refuses
	// selector_bootstrap_incomplete.
	LeaseEpoch  uint64
	LeaseID     string
	OwnerHostID string
	// AuthorityHeads is the sorted unique digest array of the validated
	// lease, event, and tombstone heads used for the decision: the
	// winning Lease Record digest plus the tail event digest of every
	// allowed source holding a non-parked copy of the pinned session.
	AuthorityHeads []string
	// Action is the explicit command/action tag bound at build time.
	Action string
	// DestinationHostID is the explicit destination UUIDv7, or nil
	// when the action takes no destination.
	DestinationHostID *string
	// ExpectationDigest binds the caller-declared action expectations
	// (canonically encoded JSON) for change detection. It does not
	// replace the checkpoint, workspace, cohort, provider, terminal,
	// or confirmation contracts, which have no owners in this leaf.
	ExpectationDigest string
}

// PlanArgs declares the invocation facts BuildPlan binds: the literal
// selector, the explicit action tag, the explicit destination (empty
// for none), and the opaque caller expectations (nil for none).
type PlanArgs struct {
	Selector     string
	Action       string
	Destination  string
	Expectations []byte
}

// BuildPlan resolves the selector once through the shared path and
// binds the current facts into an immutable plan. The destination is
// independently validated as a UUIDv7; expectations must be canonically
// encodable JSON and bind by digest for change detection only.
func (reader *Reader) BuildPlan(args PlanArgs) (SelectionPlan, error) {
	parsed, err := ParseSelector(args.Selector)
	if err != nil {
		return SelectionPlan{}, err
	}
	if reader == nil || reader.Local == nil {
		return SelectionPlan{}, fmt.Errorf("%w: local repository is required", ErrUnavailable)
	}
	config, err := reader.validatedConfiguration()
	if err != nil {
		return SelectionPlan{}, err
	}
	if _, ok := BoundariesFor(args.Action); !ok {
		return SelectionPlan{}, fmt.Errorf("%w: unknown plan action %q", ErrInvalidArgument, args.Action)
	}
	var destination *string
	if args.Destination != "" {
		id, err := scalar.ParseUUIDv7(args.Destination)
		if err != nil {
			return SelectionPlan{}, fmt.Errorf("%w: malformed destination host ID: %v", ErrInvalidArgument, err)
		}
		value := id.String()
		destination = &value
	}
	expectations, err := fingerprintExpectations(args.Expectations)
	if err != nil {
		return SelectionPlan{}, err
	}
	var selected Selection
	if parsed.Source != SourceBare {
		selected, err = reader.resolveExplicit(parsed, config)
	} else {
		selected, err = reader.resolveBare(parsed, config)
	}
	if err != nil {
		return SelectionPlan{}, err
	}
	return reader.bindPlan(parsed, config, selected, args.Action, destination, expectations)
}

// bindPlan reads the winning source completely and binds every plan
// member from validated current facts. The session is located by its
// UUID in the winning source only, never re-resolved by name. Lease
// authority comes from the admitted winning Lease Record for that
// UUID; the envelope winner on the selected chain must agree with it,
// and the union winner across all allowed sources must not exceed it.
func (reader *Reader) bindPlan(parsed Selector, config validatedConfig, selected Selection, action string, destination *string, expectations string) (SelectionPlan, error) {
	rows, err := reader.readSource(selected.repo, selected.PeerHostID)
	if err != nil {
		return SelectionPlan{}, err
	}
	index, err := fingerprintIndex(reader, selected.repo, rows, selected.PeerHostID)
	if err != nil {
		return SelectionPlan{}, err
	}
	configuration, err := fingerprintConfiguration(reader, config)
	if err != nil {
		return SelectionPlan{}, err
	}
	var recordID string
	found := false
	for _, row := range rows {
		if row.Projection.SessionID == selected.SessionID {
			recordID, found = row.Projection.RecordID, true
			break
		}
	}
	if !found {
		return SelectionPlan{}, fmt.Errorf("%w: selection absent from source index", ErrPlanStale)
	}
	projection, err := (&sessstate.Projector{Repo: selected.repo, LocalHostID: reader.LocalHostID}).Project(selected.SessionID)
	if err != nil {
		return SelectionPlan{}, err
	}
	if projection.Winner.IsZero() {
		return SelectionPlan{}, fmt.Errorf("%w: complete authority read proves record %s without any valid lease", ErrBootstrapIncomplete, selected.SessionID)
	}
	lease, err := reader.winningLeaseFor(selected.SessionID, projection.Kind, selected.repo)
	if err != nil {
		return SelectionPlan{}, err
	}
	if projection.Winner.Epoch != lease.Epoch || projection.Winner.LeaseID != lease.LeaseID {
		return SelectionPlan{}, fmt.Errorf("%w: winning lease changed", ErrPlanStale)
	}
	if projection.OwnerHostID != lease.HolderHostID {
		return SelectionPlan{}, fmt.Errorf("%w: session %s carries disagreeing owner facts across lease and chain", sessstate.ErrIntegrity, selected.SessionID)
	}
	plan := SelectionPlan{
		SelectorVersion:     SelectorVersion,
		Selector:            parsed.Raw,
		SessionID:           selected.SessionID,
		SessionRecordID:     recordID,
		SourceIndexDigest:   index,
		ConfigurationDigest: configuration,
		LeaseRecordID:       lease.Digest,
		LeaseEpoch:          lease.Epoch,
		LeaseID:             lease.LeaseID,
		OwnerHostID:         lease.HolderHostID,
		Action:              action,
		DestinationHostID:   destination,
		ExpectationDigest:   expectations,
	}
	if selected.PeerHostID == "" {
		if config.localHost == "" {
			return SelectionPlan{}, fmt.Errorf("%w: local host identity is required to bind a local plan source", ErrInvalidConfig)
		}
		plan.SourceHostID = config.localHost
	} else {
		plan.SourceHostID = selected.PeerHostID
		// Null marks the local source only. An alias-less peer binds
		// empty: the host still distinguishes it from local.
		alias := config.aliasFor(selected.PeerHostID)
		aliasCopy := alias
		plan.SourceAlias = &aliasCopy
	}
	// The union check consults record, tombstone, and winner facts
	// only, never heads; heads are bound after the union validates.
	plan.AuthorityHeads = []string{}
	if err := reader.validateAuthorityUnion(plan, config, selected.PeerHostID); err != nil {
		return SelectionPlan{}, err
	}
	heads, err := reader.collectAuthorityHeads(config, selected.SessionID, lease.Digest, selected.PeerHostID)
	if err != nil {
		return SelectionPlan{}, err
	}
	plan.AuthorityHeads = heads
	if err := checkPlanShapes(plan); err != nil {
		return SelectionPlan{}, err
	}
	return plan, nil
}

// chainTail returns the tail event digest of a validated session chain,
// or empty when the chain carries no events yet.
func chainTail(repo *sessrepo.Repository, sessionID string) (string, error) {
	events, err := repo.ListEvents(sessionID)
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return "", nil
	}
	return events[len(events)-1].EventID, nil
}
