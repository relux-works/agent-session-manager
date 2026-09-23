package sessckpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Checkpoint schema identity the capture entry binds.
const (
	checkpointSchema  = "urn:ax:schema:checkpoint"
	checkpointVersion = "1.0.0"
)

// Evidence kinds the Safe Boundary Evidence closed shape admits
// (SPEC Section 5.4).
const (
	EvidenceProviderAPI     = "provider_api"
	EvidenceProviderEvent   = "provider_event"
	EvidenceManagedPTY      = "managed_pty"
	EvidenceTaskBoardBridge = "task_board_bridge"
	EvidenceAcceptedTest    = "accepted_test"
)

// Session persistence kinds the referenced Session Record selects
// (SPEC Section 5.4).
const (
	SessionKindDirect    = "direct"
	SessionKindTaskBoard = "task_board"
)

// SafeBoundary is the typed terminal evidence leg of the capture
// closure: provider identity plus the quiescence proof. A published
// checkpoint requires all three idleness facts true and both
// counters zero; an adapter that cannot prove full idle must not
// publish, so Capture refuses a non-quiescent boundary before
// publication with the incompatible_schema class (CP-N1).
type SafeBoundary struct {
	ProviderID          string
	ProviderVersion     string
	Evidence            string
	InputBlocked        bool
	ForegroundIdle      bool
	BackgroundIdle      bool
	OpenProcesses       uint64
	OpenDatabaseHandles uint64
}

// Inputs is the typed checkpoint closure: provider identity,
// workspace manifests, the task-board bundle, terminal evidence,
// and the source head, bound to the owning lease tuple and the
// caller-stable operation that makes capture idempotent.
//
// Exactly one persistence leg must be set: ProviderManifestID for
// SessionKindDirect, TaskBoardBundleID for SessionKindTaskBoard.
// The other leg must stay empty. EventHeads carries the
// authoritative event DAG heads immediately before the checkpoint
// object: 1..64 sorted unique digests, each resolving to a chained
// event for the session at or before the owning lease.
type Inputs struct {
	OperationID         string
	SessionID           string
	SessionKind         string
	LeaseEpoch          uint64
	LeaseID             string
	CreatorHostID       string
	WorkspaceManifestID string
	ProviderManifestID  string
	TaskBoardBundleID   string
	Boundary            SafeBoundary
	EventHeads          []string
	CreatedAt           string
	Extensions          map[string]string
}

// receipt is the machine-local operation record: the operation
// identity with the checkpoint it installed and the input digest
// that makes a retry comparable. It is validated by strict decode
// plus grammar on every read; it carries no closed schema version
// because no peer ever consumes it.
type receipt struct {
	OperationID  string `json:"operation_id"`
	CheckpointID string `json:"checkpoint_id"`
	InputDigest  string `json:"input_digest"`
	SessionID    string `json:"session_id"`
}

// invalid refuses an unpublishable closure with the SPEC Section
// 5.4 incompatible_schema class: every such error wraps both the
// package sentinel and canonicaljson.ErrInvalidIdentity, so
// callers match either layer with errors.Is. It is a function,
// not a method, so every gate site reads identically.
func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %w: %s",
		ErrInvalidCheckpoint, canonicaljson.ErrInvalidIdentity, fmt.Sprintf(format, arguments...))
}

// Capture builds the typed closure into a Checkpoint Record
// 1.0.0, attests it through the sessrepo owner, and installs it
// durably with its operation receipt. The same operation retried
// with byte-identical inputs replays the recorded checkpoint
// without writing; the same operation retried with moved inputs
// refuses ErrCheckpointConflict and writes nothing.
//
// Chain binds the source-head leg: every head must resolve to a
// chained event for the session at or before the owning lease. A
// nil chain refuses closed rather than skipping the gate.
func (store *Store) Capture(chain *sessrepo.Repository, inputs Inputs) (CheckpointRef, []byte, error) {
	candidate, err := buildCandidate(inputs)
	if err != nil {
		return CheckpointRef{}, nil, err
	}
	if err := checkHeadBinding(chain, inputs); err != nil {
		return CheckpointRef{}, nil, err
	}
	raw, digest, err := identifyCandidate(candidate)
	if err != nil {
		return CheckpointRef{}, nil, err
	}
	inputDigest, err := digestInputs(candidate)
	if err != nil {
		return CheckpointRef{}, nil, err
	}
	operation, err := checkOperationID(inputs.OperationID)
	if err != nil {
		return CheckpointRef{}, nil, err
	}
	session, err := checkSessionID(inputs.SessionID)
	if err != nil {
		return CheckpointRef{}, nil, err
	}
	return store.install(operation, session, digest, inputDigest, raw)
}

// Admit installs an externally built Checkpoint Record (for
// example one received through sync) through the same attestation
// and durability path as Capture. Raw bytes are attested through
// the sessrepo owner first; the kind-selected persistence variant
// and the head binding are then decided exactly as in Capture, so
// an unpublishable record is refused before publication with the
// same classes. The operation receipt binds the raw digest as the
// input digest, so a byte-identical re-admit replays and moved
// bytes under the same operation refuse.
func (store *Store) Admit(chain *sessrepo.Repository, raw []byte, operationID, sessionKind string) (CheckpointRef, error) {
	operation, err := checkOperationID(operationID)
	if err != nil {
		return CheckpointRef{}, err
	}
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return CheckpointRef{}, invalid("decode checkpoint record frame: %v", fault)
	}
	schema, err := stringMember(members, "schema")
	if err != nil || schema != checkpointSchema {
		return CheckpointRef{}, invalid("checkpoint record schema %q, want %q", schema, checkpointSchema)
	}
	digest, field, err := sessrepo.AttestCheckpointRecord(raw)
	if err != nil {
		return CheckpointRef{}, invalid("verify checkpoint record identity: %v", err)
	}
	if field != canonicaljson.SelfCheckpointID {
		return CheckpointRef{}, invalid("checkpoint record identity field %q, want checkpoint_id", string(field))
	}
	admitted, err := extractAdmitted(members)
	if err != nil {
		return CheckpointRef{}, err
	}
	kind, err := checkSessionKind(sessionKind)
	if err != nil {
		return CheckpointRef{}, err
	}
	if err := checkPersistenceVariant(kind, admitted.hasProvider, admitted.hasBoard); err != nil {
		return CheckpointRef{}, err
	}
	if err := checkRawHeadBinding(chain, admitted); err != nil {
		return CheckpointRef{}, err
	}
	inputDigest := scalar.SHA256Digest(raw).String()
	installed, _, err := store.install(operation, admitted.sessionID, digest.String(), inputDigest, append([]byte(nil), raw...))
	return installed, err
}

// Get returns the byte-identical Checkpoint Record for its digest,
// re-verified through the canonical owner on every read.
func (store *Store) Get(checkpointID string) ([]byte, error) {
	digest, err := scalar.ParseDigest(checkpointID)
	if err != nil {
		return nil, invalid("checkpoint id %q: %v", checkpointID, err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	raw, err := readBlob(store.blobPath(digest.String()))
	if err != nil {
		return nil, err
	}
	if _, _, err := sessrepo.AttestCheckpointRecord(raw); err != nil {
		return nil, invalid("verify stored checkpoint record identity: %v", err)
	}
	return raw, nil
}

// EventHeads returns the checkpoint's attested event-head closure for the
// named session. It reuses the same strict shape extraction and source-chain
// binding as Capture and Admit; callers never need to decode checkpoint bytes
// or treat a readable blob as an admitted closure.
func (store *Store) EventHeads(chain *sessrepo.Repository, checkpointID, sessionID string) ([]string, error) {
	raw, err := store.Get(checkpointID)
	if err != nil {
		return nil, err
	}
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return nil, invalid("decode stored checkpoint closure: %v", fault)
	}
	admitted, err := extractAdmitted(members)
	if err != nil {
		return nil, err
	}
	if admitted.sessionID != sessionID {
		return nil, invalid("checkpoint session %s does not match requested session %s", admitted.sessionID, sessionID)
	}
	if err := checkRawHeadBinding(chain, admitted); err != nil {
		return nil, err
	}
	return append([]string(nil), admitted.heads...), nil
}

// admittedMembers carries the closure legs extracted from attested
// raw bytes for the variant and head gates.
type admittedMembers struct {
	sessionID   string
	leaseEpoch  uint64
	leaseID     string
	hasProvider bool
	hasBoard    bool
	heads       []string
}

// extractAdmitted reads the bound members after the owner
// attested the closed shape. Grammar failures here cannot fire
// on owner-admitted bytes; they stay invalid-class refusals so a
// future shape drift fails closed into the same publication bar.
func extractAdmitted(members map[string]json.RawMessage) (admittedMembers, error) {
	var admitted admittedMembers
	session, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return admitted, invalid("checkpoint record carries no valid session_id member")
	}
	admitted.sessionID = session.String()
	epoch, ok := environ.CheckUint53Bounds(members["lease_epoch"], 1, uint53Max)
	if !ok {
		return admitted, invalid("checkpoint record carries no lease_epoch at or above 1")
	}
	admitted.leaseEpoch = epoch
	leaseID, err := stringMember(members, "lease_id")
	if err != nil {
		return admitted, invalid("checkpoint record carries no lease_id member: %v", err)
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return admitted, invalid("checkpoint record lease_id: %v", err)
	}
	admitted.leaseID = leaseID
	for _, leg := range []struct {
		name    string
		present *bool
	}{
		{"provider_manifest_id", &admitted.hasProvider},
		{"task_board_bundle_id", &admitted.hasBoard},
	} {
		raw, ok := members[leg.name]
		if !ok {
			return admitted, invalid("checkpoint record member %q is absent", leg.name)
		}
		if string(raw) == "null" {
			continue
		}
		if _, ok := environ.CheckDigest(raw); !ok {
			return admitted, invalid("checkpoint record %s is neither a digest nor null", leg.name)
		}
		*leg.present = true
	}
	raw, ok := members["event_heads"]
	if !ok {
		return admitted, invalid("checkpoint record member %q is absent", "event_heads")
	}
	var heads []string
	if err := json.Unmarshal(raw, &heads); err != nil {
		return admitted, invalid("checkpoint record event_heads is not a string array: %v", err)
	}
	for _, head := range heads {
		if _, err := scalar.ParseDigest(head); err != nil {
			return admitted, invalid("checkpoint record event head %q: %v", head, err)
		}
		admitted.heads = append(admitted.heads, head)
	}
	return admitted, nil
}

// uint53Max is the AX safe-integer ceiling from SPEC Section 1.6.
// Bounds stay with the environ owner; the ceiling is passed, never
// re-enforced here.
const uint53Max = uint64(1<<53 - 1)

// buildCandidate validates every typed closure leg and assembles
// the candidate object with a zero placeholder identity. The
// placeholder must parse as a digest because the identity owner
// validates shape before omitting the self field; the real digest
// is computed by identifyCandidate.
func buildCandidate(inputs Inputs) (map[string]any, error) {
	session, err := checkSessionID(inputs.SessionID)
	if err != nil {
		return nil, err
	}
	kind, err := checkSessionKind(inputs.SessionKind)
	if err != nil {
		return nil, err
	}
	epoch, err := checkLeaseEpoch(inputs.LeaseEpoch)
	if err != nil {
		return nil, err
	}
	lease, err := checkLeaseID(inputs.LeaseID)
	if err != nil {
		return nil, err
	}
	creator, err := checkCreator(inputs.CreatorHostID)
	if err != nil {
		return nil, err
	}
	workspace, err := checkDigestMember(inputs.WorkspaceManifestID, "workspace_manifest_id")
	if err != nil {
		return nil, err
	}
	provider, hasProvider, err := checkNullableDigest(inputs.ProviderManifestID, "provider_manifest_id")
	if err != nil {
		return nil, err
	}
	bundle, hasBoard, err := checkNullableDigest(inputs.TaskBoardBundleID, "task_board_bundle_id")
	if err != nil {
		return nil, err
	}
	if err := checkPersistenceVariant(kind, hasProvider, hasBoard); err != nil {
		return nil, err
	}
	boundary, err := checkBoundary(inputs.Boundary)
	if err != nil {
		return nil, err
	}
	heads, err := checkHeads(inputs.EventHeads)
	if err != nil {
		return nil, err
	}
	created, err := checkCreatedAt(inputs.CreatedAt)
	if err != nil {
		return nil, err
	}
	extensions, err := checkExtensions(inputs.Extensions)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"schema":                checkpointSchema,
		"schema_version":        checkpointVersion,
		"checkpoint_id":         zeroDigest,
		"subject_id":            session,
		"session_id":            session,
		"lease_epoch":           epoch,
		"lease_id":              lease,
		"safe_boundary":         boundary,
		"event_heads":           heads,
		"workspace_manifest_id": workspace,
		"provider_manifest_id":  provider,
		"task_board_bundle_id":  bundle,
		"created_by_host_id":    creator,
		"created_at":            created,
		"status":                "validated",
		"extensions":            extensions,
	}, nil
}

// zeroDigest is the identity placeholder the owner omits before
// the omit-self digest. It must parse as a digest because shape
// validation runs before omission.
const zeroDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// identifyCandidate computes the omit-self digest through the
// canonical owner, stamps it as checkpoint_id, and re-attests the
// final bytes through the sessrepo owner. The stamped digest must
// equal the attested one; a drift between calculation and
// attestation fails closed instead of publishing.
func identifyCandidate(candidate map[string]any) ([]byte, string, error) {
	framed, err := json.Marshal(candidate)
	if err != nil {
		return nil, "", invalid("encode checkpoint candidate: %v", err)
	}
	calculated, field, err := canonicaljson.CalculateObjectIdentity(framed)
	if err != nil {
		return nil, "", invalid("calculate checkpoint identity: %v", err)
	}
	if field != canonicaljson.SelfCheckpointID {
		return nil, "", invalid("checkpoint identity field %q, want checkpoint_id", string(field))
	}
	candidate["checkpoint_id"] = calculated.String()
	raw, err := json.Marshal(candidate)
	if err != nil {
		return nil, "", invalid("encode identified checkpoint: %v", err)
	}
	attested, _, err := sessrepo.AttestCheckpointRecord(raw)
	if err != nil {
		return nil, "", invalid("verify checkpoint record identity: %v", err)
	}
	if attested != calculated {
		return nil, "", invalid("checkpoint identity drifted between calculation and attestation")
	}
	return raw, calculated.String(), nil
}

// digestInputs binds the idempotency receipt to the exact closure:
// the canonical bytes of the candidate with the placeholder
// identity, so equal closures share one input digest and any moved
// leg changes it.
func digestInputs(candidate map[string]any) (string, error) {
	framed, err := json.Marshal(candidate)
	if err != nil {
		return "", invalid("encode checkpoint inputs: %v", err)
	}
	canonical, err := canonicaljson.Canonicalize(framed)
	if err != nil {
		return "", invalid("canonicalize checkpoint inputs: %v", err)
	}
	return scalar.SHA256Digest(canonical).String(), nil
}

// checkSessionKind admits only the two persistence-selecting
// session kinds a Session Record carries.
func checkSessionKind(kind string) (string, error) {
	switch kind {
	case SessionKindDirect, SessionKindTaskBoard:
		return kind, nil
	default:
		return "", invalid("session kind %q selects no checkpoint persistence variant", kind)
	}
}

// checkPersistenceVariant binds the record to the persistence
// variant the referenced Session Record selects (SPEC Section
// 5.4): a direct session requires a non-null provider manifest
// with a null task-board bundle, while a task_board session
// requires the reverse. Canonical shape proves exactly one leg
// is present but cannot select which one is valid, so the
// kind-selected half is decided here and refused before
// publication with the incompatible_schema class. CP-N2 (direct
// with null provider manifest) and CP-N3 (both legs non-null)
// refuse at this gate.
func checkPersistenceVariant(kind string, hasProvider, hasBoard bool) error {
	if hasProvider == hasBoard {
		return invalid("checkpoint requires exactly one of provider_manifest_id and task_board_bundle_id")
	}
	switch kind {
	case SessionKindDirect:
		if !hasProvider || hasBoard {
			return invalid("checkpoint carries a task-board persistence variant for direct session")
		}
	case SessionKindTaskBoard:
		if hasProvider || !hasBoard {
			return invalid("checkpoint carries a provider persistence variant for task_board session")
		}
	default:
		return invalid("session kind %q selects no checkpoint persistence variant", kind)
	}
	return nil
}

// checkBoundary validates the terminal evidence leg and enforces
// the publication quiescence rule: all three idleness facts true
// and both counters zero (CP-N1). An adapter that cannot prove
// full idle must not publish, so a non-quiescent boundary is
// refused before publication with the incompatible_schema class.
func checkBoundary(boundary SafeBoundary) (map[string]any, error) {
	if _, err := scalar.ParseProviderID(boundary.ProviderID); err != nil {
		return nil, invalid("safe_boundary provider_id: %v", err)
	}
	if length := runeCount(boundary.ProviderVersion); length < 1 || length > 128 {
		return nil, invalid("safe_boundary provider_version must contain 1..128 characters")
	}
	switch boundary.Evidence {
	case EvidenceProviderAPI, EvidenceProviderEvent, EvidenceManagedPTY, EvidenceTaskBoardBridge, EvidenceAcceptedTest:
	default:
		return nil, invalid("safe_boundary evidence %q is outside the closed evidence enum", boundary.Evidence)
	}
	if !boundary.InputBlocked {
		return nil, invalid("safe_boundary input_blocked must be true for publication")
	}
	if !boundary.ForegroundIdle {
		return nil, invalid("safe_boundary foreground_idle must be true for publication")
	}
	if !boundary.BackgroundIdle {
		return nil, invalid("safe_boundary background_idle must be true for publication")
	}
	if boundary.OpenProcesses != 0 {
		return nil, invalid("safe_boundary open_processes must be zero for publication")
	}
	if boundary.OpenDatabaseHandles != 0 {
		return nil, invalid("safe_boundary open_database_handles must be zero for publication")
	}
	return map[string]any{
		"provider_id":           boundary.ProviderID,
		"provider_version":      boundary.ProviderVersion,
		"evidence":              boundary.Evidence,
		"input_blocked":         true,
		"foreground_idle":       true,
		"background_idle":       true,
		"open_processes":        boundary.OpenProcesses,
		"open_database_handles": boundary.OpenDatabaseHandles,
	}, nil
}

// checkHeads validates the source-head shape leg: 1..64 sorted
// unique digests. Resolution against the session chain belongs to
// checkHeadBinding; grammar alone never admits authority.
func checkHeads(heads []string) ([]string, error) {
	if len(heads) < 1 || len(heads) > 64 {
		return nil, invalid("event_heads requires 1..64 entries, got %d", len(heads))
	}
	previous := ""
	out := make([]string, 0, len(heads))
	for _, head := range heads {
		digest, err := scalar.ParseDigest(head)
		if err != nil {
			return nil, invalid("event head %q: %v", head, err)
		}
		name := digest.String()
		if len(out) > 0 && name <= previous {
			return nil, invalid("event_heads must be sorted unique")
		}
		previous = name
		out = append(out, name)
	}
	return out, nil
}

// checkHeadBinding resolves every head of the typed closure
// against the winning source chain for its session (SPEC Section
// 5.4 event-head closure). Each head must name a chained event
// for the same session at or before the owning lease: a head
// naming an absent event propagates the owner's unknown-event
// refusal, and a head under a later lease (a greater epoch, or
// the same epoch under a different fencing token) refuses as an
// unpublishable closure, because the checkpoint cannot fix
// authority over events that postdate it. Historical heads stay
// admissible and are never required to equal the current tail. A
// nil chain refuses closed rather than skipping the gate.
func checkHeadBinding(chain *sessrepo.Repository, inputs Inputs) error {
	if chain == nil {
		return invalid("no winning source chain binds checkpoint event heads")
	}
	events, err := chain.ListEvents(inputs.SessionID)
	if err != nil {
		return fmt.Errorf("bind checkpoint event heads: %w", err)
	}
	byID := make(map[string]sessrepo.EventSummary, len(events))
	for _, summary := range events {
		byID[summary.EventID] = summary
	}
	// The owning tuple is compared in canonical form, the same
	// rendering buildCandidate stamps into the record, so a
	// grammar-valid non-canonical spelling cannot split the
	// binding check from the published bytes.
	lease, err := scalar.ParseUUIDv4(inputs.LeaseID)
	if err != nil {
		return invalid("lease id %q: %v", inputs.LeaseID, err)
	}
	for _, head := range inputs.EventHeads {
		// Grammar already passed in checkHeads (buildCandidate runs
		// first); the canonical rendering keys the chain lookup.
		digest, err := scalar.ParseDigest(head)
		if err != nil {
			return invalid("event head %q: %v", head, err)
		}
		summary, ok := byID[digest.String()]
		if !ok {
			return fmt.Errorf("bind checkpoint event head %s: %w", digest.String(), sessrepo.ErrUnknownEvent)
		}
		if summary.LeaseEpoch > inputs.LeaseEpoch {
			return invalid("checkpoint event head %s sits under later lease epoch %d past owning epoch %d",
				digest.String(), summary.LeaseEpoch, inputs.LeaseEpoch)
		}
		if summary.LeaseEpoch == inputs.LeaseEpoch && summary.LeaseID != lease.String() {
			return invalid("checkpoint event head %s sits under lease %s past owning lease %s at epoch %d",
				digest.String(), summary.LeaseID, lease.String(), inputs.LeaseEpoch)
		}
	}
	return nil
}

// checkRawHeadBinding is the raw-admission half of the head gate:
// the same at-or-before-owning-lease rule over the members the
// owner already attested. A nil chain refuses closed.
func checkRawHeadBinding(chain *sessrepo.Repository, admitted admittedMembers) error {
	if chain == nil {
		return invalid("no winning source chain binds checkpoint event heads")
	}
	events, err := chain.ListEvents(admitted.sessionID)
	if err != nil {
		return fmt.Errorf("bind checkpoint event heads: %w", err)
	}
	byID := make(map[string]sessrepo.EventSummary, len(events))
	for _, summary := range events {
		byID[summary.EventID] = summary
	}
	for _, head := range admitted.heads {
		summary, ok := byID[head]
		if !ok {
			return fmt.Errorf("bind checkpoint event head %s: %w", head, sessrepo.ErrUnknownEvent)
		}
		if summary.LeaseEpoch > admitted.leaseEpoch {
			return invalid("checkpoint event head %s sits under later lease epoch %d past owning epoch %d",
				head, summary.LeaseEpoch, admitted.leaseEpoch)
		}
		if summary.LeaseEpoch == admitted.leaseEpoch && summary.LeaseID != admitted.leaseID {
			return invalid("checkpoint event head %s sits under lease %s past owning lease %s at epoch %d",
				head, summary.LeaseID, admitted.leaseID, admitted.leaseEpoch)
		}
	}
	return nil
}

// checkOperationID admits the caller-stable idempotency key.
func checkOperationID(operationID string) (string, error) {
	id, err := scalar.ParseUUIDv7(operationID)
	if err != nil {
		return "", invalid("operation id %q: %v", operationID, err)
	}
	return id.String(), nil
}

// checkSessionID admits the checkpoint subject session.
func checkSessionID(sessionID string) (string, error) {
	id, err := scalar.ParseUUIDv7(sessionID)
	if err != nil {
		return "", invalid("session id %q: %v", sessionID, err)
	}
	return id.String(), nil
}

// checkLeaseEpoch admits the owning lease epoch: greater than
// zero, inside the AX safe-integer ceiling.
func checkLeaseEpoch(epoch uint64) (uint64, error) {
	if epoch < 1 || epoch > uint53Max {
		return 0, invalid("lease epoch %d is outside 1..2^53-1", epoch)
	}
	return epoch, nil
}

// checkLeaseID admits the owning lease fencing token.
func checkLeaseID(leaseID string) (string, error) {
	id, err := scalar.ParseUUIDv4(leaseID)
	if err != nil {
		return "", invalid("lease id %q: %v", leaseID, err)
	}
	return id.String(), nil
}

// checkCreator admits the creating host: the consumer binds it to
// the owning lease holder at admission, so capture records exactly
// what the caller declares and never substitutes another host.
func checkCreator(creator string) (string, error) {
	id, err := scalar.ParseUUIDv7(creator)
	if err != nil {
		return "", invalid("created_by_host id %q: %v", creator, err)
	}
	return id.String(), nil
}

// checkDigestMember admits a required manifest digest leg.
func checkDigestMember(value, name string) (string, error) {
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return "", invalid("%s %q: %v", name, value, err)
	}
	return digest.String(), nil
}

// checkNullableDigest admits one persistence leg: empty binds
// null, anything else must parse as a canonical digest.
func checkNullableDigest(value, name string) (any, bool, error) {
	if value == "" {
		return nil, false, nil
	}
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return nil, false, invalid("%s %q: %v", name, value, err)
	}
	return digest.String(), true, nil
}

// checkCreatedAt admits the diagnostic creation instant. The
// instant is diagnostic only and never confers authority, but its
// grammar is still closed: a malformed timestamp is refused
// before publication.
func checkCreatedAt(createdAt string) (string, error) {
	instant, err := scalar.ParseTimestamp(createdAt)
	if err != nil {
		return "", invalid("created_at %q: %v", createdAt, err)
	}
	return instant.String(), nil
}

// checkExtensions admits the extension object: reverse-DNS keys
// only, at most the owner's 64 members. A nil map binds the empty
// object, never null. Extensions never add operations,
// capabilities, or trust facts.
func checkExtensions(extensions map[string]string) (map[string]any, error) {
	if extensions == nil {
		extensions = map[string]string{}
	}
	out := make(map[string]any, len(extensions))
	for key, value := range extensions {
		out[key] = value
	}
	if len(out) > 64 {
		return nil, invalid("extensions contains %d members, maximum is 64", len(out))
	}
	framed, err := json.Marshal(out)
	if err != nil {
		return nil, invalid("encode checkpoint extensions: %v", err)
	}
	if !environ.CheckExtensions(framed) {
		return nil, invalid("checkpoint extensions keys must be 3..253 character lowercase reverse-DNS names")
	}
	return out, nil
}

// stringMember reads a required string member from a strictly
// decoded frame.
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

// runeCount measures provider_version in Unicode characters, the
// unit the closed shape declares for its 1..128 bound.
func runeCount(value string) int {
	return len([]rune(value))
}

// install is the shared durable tail of Capture and Admit: replay
// the recorded checkpoint when the same operation already
// committed identical inputs, refuse the moved-input retry, and
// otherwise install the blob before the receipt it names. The
// blob-first order keeps an interrupted capture resuming: the
// identical retry finds its bytes already installed and completes
// the receipt, so no second checkpoint identity is ever minted
// for one operation.
func (store *Store) install(operation, session, digest, inputDigest string, raw []byte) (CheckpointRef, []byte, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if recorded, ok, err := store.readReceiptLocked(operation); err != nil {
		return CheckpointRef{}, nil, err
	} else if ok {
		if recorded.InputDigest != inputDigest {
			return CheckpointRef{}, nil, fmt.Errorf("%w: operation %s captured different inputs (idempotency_mismatch)",
				ErrCheckpointConflict, operation)
		}
		replayed, err := readBlob(store.blobPath(recorded.CheckpointID))
		if err != nil {
			return CheckpointRef{}, nil, err
		}
		return CheckpointRef{CheckpointID: recorded.CheckpointID, OperationID: operation}, replayed, nil
	}
	if store.BeforeWrite != nil {
		if err := store.BeforeWrite(); err != nil {
			return CheckpointRef{}, nil, err
		}
	}
	if err := installBlob(store.blobPath(digest), raw); err != nil {
		return CheckpointRef{}, nil, err
	}
	if store.AfterBlob != nil {
		if err := store.AfterBlob(); err != nil {
			return CheckpointRef{}, nil, err
		}
	}
	committed := receipt{OperationID: operation, CheckpointID: digest, InputDigest: inputDigest, SessionID: session}
	if err := store.writeReceiptLocked(committed); err != nil {
		if conflict, ok, readErr := store.rereadReceiptLocked(operation, inputDigest); readErr != nil {
			return CheckpointRef{}, nil, readErr
		} else if ok {
			replayed, err := readBlob(store.blobPath(conflict.CheckpointID))
			if err != nil {
				return CheckpointRef{}, nil, err
			}
			return conflict, replayed, nil
		}
		return CheckpointRef{}, nil, err
	}
	if store.AfterCommit != nil {
		if err := store.AfterCommit(); err != nil {
			return CheckpointRef{}, nil, err
		}
	}
	return CheckpointRef{CheckpointID: digest, OperationID: operation}, append([]byte(nil), raw...), nil
}

// rereadReceiptLocked resolves a receipt-install race: a
// concurrent capture committed the same operation first. An
// identical-inputs winner replays; moved inputs refuse.
func (store *Store) rereadReceiptLocked(operation, inputDigest string) (CheckpointRef, bool, error) {
	recorded, ok, err := store.readReceiptLocked(operation)
	if err != nil {
		return CheckpointRef{}, false, err
	}
	if !ok {
		return CheckpointRef{}, false, nil
	}
	if recorded.InputDigest != inputDigest {
		return CheckpointRef{}, false, fmt.Errorf("%w: operation %s captured different inputs (idempotency_mismatch)",
			ErrCheckpointConflict, operation)
	}
	return CheckpointRef{CheckpointID: recorded.CheckpointID, OperationID: operation}, true, nil
}

// blobPath names the content-addressed blob for one checkpoint
// digest. Callers hold the store mutex.
func (store *Store) blobPath(digest string) string {
	return store.root + "/" + checkpointFileName(digest)
}

// receiptPath names the operation receipt for one operation UUID.
// Callers hold the store mutex.
func (store *Store) receiptPath(operation string) string {
	return store.root + "/operations/" + operationFileName(operation)
}

// readReceiptLocked returns the committed receipt for one
// operation, validated by strict decode plus grammar on every
// read. Callers hold the store mutex.
func (store *Store) readReceiptLocked(operation string) (receipt, bool, error) {
	framed, err := readBlob(store.receiptPath(operation))
	if err != nil {
		if os.IsNotExist(err) {
			return receipt{}, false, nil
		}
		return receipt{}, false, err
	}
	members, fault := environ.DecodeStrictObject(framed)
	if fault != nil {
		return receipt{}, false, fmt.Errorf("decode checkpoint operation receipt: %v", fault)
	}
	// Receipt grammar is checked with the scalar owner directly and
	// reported as plain operational errors: a damaged
	// machine-local receipt is torn recovery state, never an
	// unpublishable closure, so the invalid-closure sentinel must
	// not classify it.
	var recorded receipt
	for _, leg := range []struct {
		name   string
		target *string
	}{
		{"operation_id", &recorded.OperationID},
		{"checkpoint_id", &recorded.CheckpointID},
		{"input_digest", &recorded.InputDigest},
		{"session_id", &recorded.SessionID},
	} {
		value, err := stringMember(members, leg.name)
		if err != nil {
			return receipt{}, false, fmt.Errorf("decode checkpoint operation receipt: %v", err)
		}
		*leg.target = value
	}
	if _, err := scalar.ParseUUIDv7(recorded.OperationID); err != nil {
		return receipt{}, false, fmt.Errorf("decode checkpoint operation receipt operation_id: %v", err)
	}
	for _, leg := range []struct {
		name  string
		value string
	}{
		{"checkpoint_id", recorded.CheckpointID},
		{"input_digest", recorded.InputDigest},
	} {
		if _, err := scalar.ParseDigest(leg.value); err != nil {
			return receipt{}, false, fmt.Errorf("decode checkpoint operation receipt %s: %v", leg.name, err)
		}
	}
	if _, err := scalar.ParseUUIDv7(recorded.SessionID); err != nil {
		return receipt{}, false, fmt.Errorf("decode checkpoint operation receipt session_id: %v", err)
	}
	if recorded.OperationID != operation {
		return receipt{}, false, fmt.Errorf("checkpoint operation receipt names %s, want %s", recorded.OperationID, operation)
	}
	return recorded, true, nil
}

// writeReceiptLocked installs one operation receipt no-replace:
// the first committer wins and every later retry replays or
// refuses against it. Callers hold the store mutex.
func (store *Store) writeReceiptLocked(committed receipt) error {
	framed, err := json.Marshal(committed)
	if err != nil {
		return fmt.Errorf("encode checkpoint operation receipt: %w", err)
	}
	// An EEXIST loser is not a failure: install() rereads the
	// winner's receipt and replays or refuses against it. The raw
	// error is returned unwrapped so os.IsExist keeps working for
	// any future caller that branches on it.
	if err := writeExclusive(store.receiptPath(committed.OperationID), framed); err != nil {
		return err
	}
	return syncDirectory(store.root + "/operations")
}

// readBlob reads one installed file verbatim.
func readBlob(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes.Clone(raw), nil
}
