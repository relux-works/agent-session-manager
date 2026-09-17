package sessstate

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Fixed test identities. UUIDv7 keeps the version nibble 7 with an RFC
// 4122 variant; UUIDv4 keeps nibble 4.
const (
	testSessionID  = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	testSessionIDB = "0198f4c8-3e70-7a11-8a2b-1234567890ac"
	testHostID     = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testHostIDB    = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	testGroupID    = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	testOpID       = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	testOpIDB      = "0198f4c8-7d40-7e55-8e6f-1234567890ac"
	testLeaseID    = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	testLeaseIDB   = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	testLeaseIDC   = "cccccccc-dddd-4eee-8fff-000000000000"
	testLeaseIDR   = "00000000-1111-4000-8000-000000000001"
	testCreatedAt  = "2026-08-19T04:00:00.000Z"
)

const (
	zeroDigest     = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	checkpointOne  = "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656"
	checkpointTwo  = "sha256:f162007a62a46bd93c5def2f41a0e17ea060f15e0d7b8c0a03131c42d53f74a7"
	identityRecord = "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2"
	tombstoneID    = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"
	bindingID      = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	evidenceID     = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

// specRecordExample is the verbatim SPEC.md Section 5.1 normative
// example, reused as the fixture root.
const specRecordExample = `{
  "schema": "urn:ax:schema:session-record",
  "schema_version": "1.0.0",
  "record_id": "sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "name": "payments-api",
  "kind": "direct",
  "created_at": "2026-08-19T04:00:00.000Z",
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "provider_id": "codex",
  "workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
  "execution_profile": "yolo",
  "launch_plan": {
    "argv": ["codex"],
    "cwd_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab",
    "cwd_relative": "src",
    "env_names": ["OPENAI_API_KEY"],
    "env_literals": {},
    "contains_secrets": false,
    "extensions": {}
  },
  "task_board": null,
  "fork_provenance": null,
  "extensions": {}
}`

// buildRecord derives a valid Session Record from the SPEC example
// with the caller's mutations, recomputing the canonical digest
// through the production identity entry.
func buildRecord(t *testing.T, mutate func(object map[string]any)) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal([]byte(specRecordExample), &object); err != nil {
		t.Fatalf("unmarshal SPEC record example: %v", err)
	}
	if mutate != nil {
		mutate(object)
	}
	object["record_id"] = zeroDigest
	staged, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal staged record: %v", err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity(staged record) error = %v", err)
	}
	object["record_id"] = digest.String()
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(out); err != nil {
		t.Fatalf("VerifyObjectIdentity(built record) error = %v", err)
	}
	return out
}

// eventOptions selects the envelope and payload of a built event.
type eventOptions struct {
	sessionID     string
	predecessors  []string
	epoch         uint64
	leaseID       string
	sequence      uint64
	eventType     string
	payload       map[string]any
	schemaVersion string
}

// buildEvent derives a valid Session Event, computing the canonical
// digest through the production identity entry. The payload must carry
// exactly the canonical owner's members for its type, so every vector
// below doubles as a contract fixture.
func buildEvent(t *testing.T, options eventOptions) []byte {
	t.Helper()
	version := options.schemaVersion
	if version == "" {
		version = "1.0.0"
	}
	object := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": version,
		"event_id": zeroDigest, "subject_id": options.sessionID, "session_id": options.sessionID,
		"event_type": options.eventType, "created_by_host_id": testHostID,
		"lease_epoch": options.epoch, "lease_id": options.leaseID, "lease_sequence": options.sequence,
		"predecessors": options.predecessors, "created_at": testCreatedAt,
		"payload": options.payload, "extensions": map[string]any{},
	}
	staged, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal staged event: %v", err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity(staged event %s) error = %v", options.eventType, err)
	}
	object["event_id"] = digest.String()
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(out); err != nil {
		t.Fatalf("VerifyObjectIdentity(built event %s) error = %v", options.eventType, err)
	}
	return out
}

// mustDecodeEvent builds then decodes one event for direct Reduce
// inputs, so the envelope under test is always canonical-exact.
func mustDecodeEvent(t *testing.T, options eventOptions) (Event, []byte) {
	t.Helper()
	raw := buildEvent(t, options)
	event, err := DecodeEvent(raw)
	if err != nil {
		t.Fatalf("DecodeEvent(%s) error = %v", options.eventType, err)
	}
	return event, raw
}

// chainBuilder folds raw events with their decoded forms in lockstep,
// tracking predecessors, and appends them to a repository on demand.
type chainBuilder struct {
	t         *testing.T
	repo      *sessrepo.Repository
	sessionID string
	recordID  string
	raws      [][]byte
	events    []Event
}

func openRepo(t *testing.T) (*sessrepo.Repository, error) {
	t.Helper()
	return sessrepo.Open(t.TempDir())
}

func createSession(t *testing.T, repository *sessrepo.Repository, record []byte) sessrepo.SessionRef {
	t.Helper()
	reference, err := repository.CreateSession(record)
	if err != nil {
		t.Fatalf("CreateSession error = %v", err)
	}
	return reference
}

func openChain(t *testing.T, record []byte) *chainBuilder {
	t.Helper()
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("sessrepo.Open error = %v", err)
	}
	reference := createSession(t, repository, record)
	return &chainBuilder{t: t, repo: repository, sessionID: reference.SessionID, recordID: reference.RecordID}
}

func (builder *chainBuilder) append(options eventOptions) Event {
	builder.t.Helper()
	options.sessionID = builder.sessionID
	if options.predecessors == nil {
		if len(builder.events) == 0 {
			options.predecessors = []string{builder.recordID}
		} else {
			options.predecessors = []string{builder.events[len(builder.events)-1].ID}
		}
	}
	event, raw := mustDecodeEvent(builder.t, options)
	if _, err := builder.repo.AppendEvent(builder.sessionID, raw); err != nil {
		builder.t.Fatalf("AppendEvent(%s seq %d) error = %v", options.eventType, options.sequence, err)
	}
	builder.raws = append(builder.raws, raw)
	builder.events = append(builder.events, event)
	return event
}

// Payload builders. Each carries exactly the canonical owner's members
// for its type and version.

func createdPayload(recordID string) map[string]any {
	return map[string]any{"session_record_id": recordID, "bootstrap_operation_id": testOpID, "first_checkpoint_operation_id": testOpIDB}
}

func terminalPayload() map[string]any {
	return map[string]any{"backend": "tmux", "terminal_id": "tmux-session-1"}
}

func terminalV4Payload() map[string]any {
	return map[string]any{
		"terminal_binding_id": bindingID, "terminal_backend_id": "ax.tmux",
		"implementation_version": "1.2.3", "protocol_version": "0.9.0",
		"evidence_ids": []any{evidenceID},
	}
}

func launchedPayload(providerID string) map[string]any {
	return map[string]any{
		"provider_id": providerID, "provider_version": "0.147.0",
		"execution_profile": "yolo", "profile_source_event_id": nil,
		"profile_mapping": "default",
	}
}

func identifiedPayload() map[string]any {
	return map[string]any{"provider_identity_record_id": identityRecord, "confidence": "exact"}
}

func idlePayload() map[string]any {
	return map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}
}

func quiescingPayload() map[string]any {
	return map[string]any{"operation_id": testOpID, "reason": "checkpoint", "input_blocked": true}
}

func checkpointPayload() map[string]any {
	return map[string]any{"checkpoint_id": checkpointOne, "kind": "manual"}
}

func syncPayload() map[string]any {
	return map[string]any{
		"peer_host_id": testHostIDB, "checkpoint_id": checkpointOne,
		"manifest_ids": []any{checkpointOne}, "materialized": true,
	}
}

func stoppedPayload() map[string]any {
	return stoppedPayloadWith(checkpointOne)
}

func stoppedPayloadWith(checkpoint string) map[string]any {
	return map[string]any{
		"graceful": true, "checkpoint_id": checkpoint, "resumable": true,
		"closure_kind": "checkpointed", "process_closed": true, "store_closed": true,
	}
}

func stoppedAbortPayload() map[string]any {
	return map[string]any{
		"graceful": false, "checkpoint_id": nil, "resumable": false,
		"closure_kind": "bootstrap_abort", "process_closed": true, "store_closed": true,
	}
}

func resumedPayload() map[string]any {
	return resumedPayloadWith(checkpointOne)
}

func resumedPayloadWith(checkpoint string) map[string]any {
	return map[string]any{
		"checkpoint_id": checkpoint, "execution_profile": "yolo",
		"profile_source_event_id": nil, "terminal_backend": "tmux",
		"native_session_id": "native-1",
	}
}

func resumedV4Payload() map[string]any {
	return map[string]any{
		"checkpoint_id": checkpointOne, "execution_profile": "yolo",
		"profile_source_event_id": nil, "terminal_binding_id": bindingID,
		"terminal_backend_id": "ax.tmux", "implementation_version": "1.2.3",
		"protocol_version": "0.9.0", "evidence_ids": []any{evidenceID},
	}
}

func abortPayload() map[string]any {
	return map[string]any{
		"operation_id": testOpID, "failure_phase": "before_terminal",
		"provider_identity_record_id": nil, "manager_session_ref": nil,
		"process_closed": true, "store_closed": true, "resume_allowed": false,
	}
}

func transferredPayload(fromLease, toLease string) map[string]any {
	return map[string]any{
		"operation_id": testOpID, "from_host_id": testHostID, "to_host_id": testHostIDB,
		"predecessor_lease_id": fromLease, "new_lease_id": toLease,
	}
}

func forcedPayload(newLease string, expectedEpoch uint64) map[string]any {
	return map[string]any{
		"operation_id": testOpID, "expected_owner_host_id": testHostID,
		"expected_epoch": expectedEpoch, "new_lease_id": newLease,
		"checkpoint_id": checkpointOne,
	}
}

func parkedPayload(winnerLease string) map[string]any {
	return map[string]any{"reason": "remote_owner", "winning_lease_id": winnerLease}
}

func failedPayload() map[string]any {
	return map[string]any{"error_code": "E_LAUNCH", "retryable": true, "operation_id": nil}
}

func forkPayload() map[string]any {
	return map[string]any{
		"source_session_id": testSessionID, "source_checkpoint_id": checkpointOne,
		"new_session_record_id": zeroDigest, "provider_fork_mode": "native",
		"execution_profile": "yolo", "profile_source_event_id": nil,
		"source_profile_event_id": nil,
	}
}

func profilePayload() map[string]any {
	return map[string]any{"from": "standard", "to": "yolo", "confirmed": true}
}

func tombstonedPayload() map[string]any {
	return map[string]any{"tombstone_id": tombstoneID}
}

func forceConfirmedPayload() map[string]any {
	return map[string]any{
		"operation_id": testOpID, "expected_owner_host_id": testHostID,
		"expected_epoch": 1, "checkpoint_id": checkpointOne,
		"accepted_risks":    []any{"divergent_history", "split_brain", "stale_process"},
		"confirmation_mode": "non_interactive",
	}
}

func replaceConfirmedPayload() map[string]any {
	return map[string]any{
		"operation_id": testOpID, "workspace_group_id": testGroupID,
		"target_host_id": testHostIDB, "managed_replica_id": testOpIDB,
		"expected_marker_id": zeroDigest, "expected_checkpoint_id": checkpointOne,
		"replacement_checkpoint_id": checkpointTwo, "confirmation_mode": "non_interactive",
	}
}

func taskBoardLaunchedPayload(state string, epoch uint64, leaseID string) map[string]any {
	return map[string]any{
		"operation_id": testOpID, "manager_session_ref": "mgr-1",
		"provider_id": "codex", "launch_mode": "primary_owner",
		"lease_epoch": epoch, "lease_id": leaseID,
		"execution_profile": "yolo", "profile_source_event_id": nil,
		"board_goal_id": nil, "board_goal_revision": nil, "state": state,
	}
}

func taskBoardAdoptedPayload() map[string]any {
	return map[string]any{
		"operation_id": testOpID, "bundle_id": zeroDigest,
		"manager_session_ref": "mgr-1", "board_goal_id": nil,
		"board_goal_revision": nil,
	}
}

func tombstoneIssuedPayload() map[string]any {
	return map[string]any{
		"tombstone_id": tombstoneID, "scope": "session",
		"subject_id": testSessionID, "target_ref": "ref-1",
	}
}

func tombstoneResolvedPayload() map[string]any {
	return map[string]any{
		"tombstone_id": tombstoneID, "resolution": "deleted",
		"target_ref": "ref-1", "resulting_entry_digest": nil,
	}
}

// mustErrorIs fails unless err matches the sentinel with errors.Is,
// so a weakened gate that returns a different error — or none — fails
// the test.
func mustErrorIs(t *testing.T, err error, sentinel error, what string) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("%s error = %v, want errors.Is %v", what, err, sentinel)
	}
}
