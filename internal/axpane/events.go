package axpane

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Event vocabulary the wrapper authors (§5.2 Terminal Events).
const (
	eventSchema        = "urn:ax:schema:session-event"
	eventSchemaV1      = "1.0.0"
	eventSchemaV4      = "4.0.0"
	eventSessionParked = "session.parked"
	eventTerminalMade  = "terminal.created"
	eventResumed       = "session.resumed"
)

// ParkedPayload builds the exact closed session.parked payload: the
// reason vocabulary member plus the winning lease ID. Any reason
// outside remote_owner|stale_owner|restore_policy|failed_handoff
// refuses: the vocabulary is closed.
func ParkedPayload(reason fencing.ParkReason, winningLeaseID string) (map[string]any, error) {
	switch reason {
	case fencing.ParkRemoteOwner, fencing.ParkStaleOwner, fencing.ParkRestorePolicy, fencing.ParkFailedHandoff:
	default:
		return nil, fmt.Errorf("parked reason %q is not the session.parked vocabulary", string(reason))
	}
	if _, err := scalar.ParseUUIDv4(winningLeaseID); err != nil {
		return nil, fmt.Errorf("parked winning lease is not a UUIDv4: %w", err)
	}
	return map[string]any{
		"reason":           string(reason),
		"winning_lease_id": winningLeaseID,
	}, nil
}

// TerminalBindingFacts carries the v4 terminal binding/evidence
// payload members: the opaque binding digest, the admitted backend
// identity, and the evidence IDs resolving locally to validated
// Manifest, Probe, and Capability Evidence.
type TerminalBindingFacts struct {
	BindingDigest         string
	BackendID             string
	ImplementationVersion string
	ProtocolVersion       string
	ProtocolVersions      []string
	EvidenceIDs           []string
}

func checkBindingFacts(facts TerminalBindingFacts) error {
	if _, err := scalar.ParseDigest(facts.BindingDigest); err != nil {
		return fmt.Errorf("terminal binding digest is not a digest: %w", err)
	}
	if _, err := terminalbackend.ParseID(facts.BackendID); err != nil {
		return err
	}
	if err := terminalbackend.CheckVersionTuple(facts.BackendID, facts.ImplementationVersion, facts.ProtocolVersion, facts.ProtocolVersions); err != nil {
		return err
	}
	if err := checkEvidenceIDs(facts.EvidenceIDs); err != nil {
		return err
	}
	return nil
}

func checkEvidenceIDs(ids []string) error {
	if len(ids) < 1 || len(ids) > 256 {
		return fmt.Errorf("evidence ids number %d, want 1..256", len(ids))
	}
	previous := ""
	for index, id := range ids {
		if _, err := scalar.ParseDigest(id); err != nil {
			return fmt.Errorf("evidence id is not a digest: %w", err)
		}
		if index > 0 && id <= previous {
			return fmt.Errorf("evidence ids are not sorted unique")
		}
		previous = id
	}
	return nil
}

// TerminalCreatedPayload builds the exact closed v4
// terminal.created payload: the opaque binding digest (the binding
// object itself is host-local and is not a member), the backend
// identity, and the evidence IDs.
func TerminalCreatedPayload(facts TerminalBindingFacts) (map[string]any, error) {
	if err := checkBindingFacts(facts); err != nil {
		return nil, err
	}
	return map[string]any{
		"terminal_binding_id":    facts.BindingDigest,
		"terminal_backend_id":    facts.BackendID,
		"implementation_version": facts.ImplementationVersion,
		"protocol_version":       facts.ProtocolVersion,
		"evidence_ids":           facts.EvidenceIDs,
	}, nil
}

// ResumedPayload builds the exact closed v4 session.resumed payload:
// the checkpoint, the effective profile pair, and the terminal
// binding/evidence members. The profile and source must equal the
// Section 2.4 effective pair the decision carries; a non-empty
// source must be a digest.
func ResumedPayload(checkpointID, profile, source string, hasSource bool, facts TerminalBindingFacts) (map[string]any, error) {
	if _, err := scalar.ParseDigest(checkpointID); err != nil {
		return nil, fmt.Errorf("resumed checkpoint is not a digest: %w", err)
	}
	if profile != "standard" && profile != "yolo" {
		return nil, fmt.Errorf("resumed profile %q is not standard or yolo", profile)
	}
	var sourceValue any
	if hasSource {
		if _, err := scalar.ParseDigest(source); err != nil {
			return nil, fmt.Errorf("resumed profile source is not a digest: %w", err)
		}
		sourceValue = source
	}
	if err := checkBindingFacts(facts); err != nil {
		return nil, err
	}
	return map[string]any{
		"checkpoint_id":           checkpointID,
		"execution_profile":       profile,
		"profile_source_event_id": sourceValue,
		"terminal_binding_id":     facts.BindingDigest,
		"terminal_backend_id":     facts.BackendID,
		"implementation_version":  facts.ImplementationVersion,
		"protocol_version":        facts.ProtocolVersion,
		"evidence_ids":            facts.EvidenceIDs,
	}, nil
}

// EmitParams carries one wrapper-authored event: the chain position
// under the authoring lease plus the closed payload and the fencing
// authority it is authored under. Presented must name the authoring
// (epoch, lease_id) tuple exactly; Observation is the verified
// lease view the mutation gate decides over. Emit authorizes only
// under a lease the local host holds: a remote, unverified,
// ambiguous, or losing lease refuses, never appends.
type EmitParams struct {
	SessionID     string
	CreatedByHost string
	LeaseEpoch    uint64
	LeaseID       string
	LeaseSequence uint64
	Predecessors  []string
	CreatedAt     string
	EventType     string
	SchemaVersion string
	Payload       map[string]any
	Presented     fencing.PresentedToken
	Observation   fencing.Observation
}

// Emit appends one wrapper-authored event through the sessrepo owner
// under the authoring lease. Every call gates through the landed
// fencing.AuthorizeMutation entry: the presented token must equal
// the winning tuple, ownership must be verified, unambiguous, and
// local, and the grant must be live. The event identity is computed
// through the canonicaljson owner; chain linkage (sequence,
// predecessors) is enforced by AppendEvent. Emit mints no lease
// and changes no ownership: the parked event records the wrapper
// state, and the remote-offer outcomes authorize no lease mutation.
func Emit(repository *sessrepo.Repository, params EmitParams) (sessrepo.EventRef, []byte, error) {
	empty := sessrepo.EventRef{}
	if repository == nil {
		return empty, nil, fmt.Errorf("emit carries no repository")
	}
	if _, err := scalar.ParseUUIDv7(params.SessionID); err != nil {
		return empty, nil, fmt.Errorf("emit session is not a UUIDv7: %w", err)
	}
	if _, err := scalar.ParseUUIDv7(params.CreatedByHost); err != nil {
		return empty, nil, fmt.Errorf("emit author is not a UUIDv7: %w", err)
	}
	if _, err := scalar.ParseUUIDv4(params.LeaseID); err != nil {
		return empty, nil, fmt.Errorf("emit lease is not a UUIDv4: %w", err)
	}
	if _, err := scalar.ParseTimestamp(params.CreatedAt); err != nil {
		return empty, nil, fmt.Errorf("emit instant is not a timestamp: %w", err)
	}
	if params.LeaseEpoch != params.Presented.Epoch || params.LeaseID != params.Presented.LeaseID || params.SessionID != params.Presented.SessionID {
		return empty, nil, fmt.Errorf("emit lease does not equal the presented fencing tuple")
	}
	if _, err := fencing.AuthorizeMutation(params.Presented, params.Observation); err != nil {
		return empty, nil, err
	}
	object := map[string]any{
		"schema":             eventSchema,
		"schema_version":     params.SchemaVersion,
		"event_id":           "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"subject_id":         params.SessionID,
		"session_id":         params.SessionID,
		"event_type":         params.EventType,
		"created_by_host_id": params.CreatedByHost,
		"lease_epoch":        params.LeaseEpoch,
		"lease_id":           params.LeaseID,
		"lease_sequence":     params.LeaseSequence,
		"predecessors":       params.Predecessors,
		"created_at":         params.CreatedAt,
		"payload":            params.Payload,
		"extensions":         map[string]any{},
	}
	staged, err := json.Marshal(object)
	if err != nil {
		return empty, nil, fmt.Errorf("marshal staged event: %w", err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		return empty, nil, fmt.Errorf("identify staged event: %w", err)
	}
	object["event_id"] = digest.String()
	raw, err := json.Marshal(object)
	if err != nil {
		return empty, nil, fmt.Errorf("marshal event: %w", err)
	}
	reference, err := repository.AppendEvent(params.SessionID, raw)
	if err != nil {
		return empty, nil, err
	}
	return reference, raw, nil
}

// EmitParked authors the session.parked event for a parked decision
// under a lease the local host holds: the parked outcome and both
// remote-offer outcomes, which park the local wrapper while offering
// attach or takeover. The reason must be the decision's parked
// reason and the winning lease the decision's winning lease. The
// parked event is a lifecycle event: it is authored only when the
// §5.7 transition table folds it from the derived state (only
// materializing, stopped, failed, or an already-parked chain park);
// any other state refuses without appending.
func EmitParked(repository *sessrepo.Repository, decision Decision, params EmitParams) (sessrepo.EventRef, []byte, error) {
	switch decision.Action {
	case ActionParked, ActionAttachRemote, ActionTakeoverOffer:
	default:
		return sessrepo.EventRef{}, nil, fmt.Errorf("parked emission requires a parked or offer decision, got %q", string(decision.Action))
	}
	// AuthorizeMutation makes the authoring lease the winner, so a
	// payload naming another winner is a lie: the decision's winning
	// lease must equal the authoring lease. Through Run both come from
	// observation.Winner; a direct caller with a mismatched pair refuses
	// here instead of appending the contradiction.
	if decision.WinningLeaseID != params.LeaseID {
		return sessrepo.EventRef{}, nil, fmt.Errorf("parked payload winner %q does not equal the authoring lease %q", decision.WinningLeaseID, params.LeaseID)
	}
	payload, err := ParkedPayload(decision.ParkReason, decision.WinningLeaseID)
	if err != nil {
		return sessrepo.EventRef{}, nil, err
	}
	foldable, err := canAuthorParked(repository, params.SessionID)
	if err != nil {
		return sessrepo.EventRef{}, nil, err
	}
	if !foldable {
		return sessrepo.EventRef{}, nil, fmt.Errorf("session.parked does not fold from the derived lifecycle state")
	}
	params.EventType = eventSessionParked
	params.SchemaVersion = eventSchemaV1
	params.Payload = payload
	return Emit(repository, params)
}

// canAuthorParked reports whether a session.parked event folds from
// the session's derived lifecycle state through the landed
// sessstate reducer. Only materializing, stopped, failed, and
// parked fold to parked (§5.7); every other state, an empty chain,
// and an already-invalid chain refuse. A repository failure
// propagates as an error, never as foldable.
func canAuthorParked(repository *sessrepo.Repository, sessionID string) (bool, error) {
	if repository == nil {
		return false, fmt.Errorf("parked fold check carries no repository")
	}
	foldInput, err := loadFoldInput(repository, sessionID)
	if err != nil {
		return false, err
	}
	projection, err := sessstate.Reduce(foldInput)
	if err != nil {
		return false, nil
	}
	switch projection.State {
	case sessstate.StateMaterializing, sessstate.StateStopped, sessstate.StateFailed, sessstate.StateParked:
		return true, nil
	default:
		return false, nil
	}
}
