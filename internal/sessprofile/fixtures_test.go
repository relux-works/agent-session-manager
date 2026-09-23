package sessprofile

import (
	"encoding/json"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Fixed test identities. Every UUIDv7 keeps the version nibble 7 and
// an RFC 4122 variant; every UUIDv4 keeps nibble 4. The session,
// host, group, and workspace values are the SPEC Section 5.1
// example identities, shared with the sessrepo fixtures.
const (
	testSessionID   = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	testSessionIDB  = "0198f4c8-3e70-7a11-8a2b-1234567890ac"
	testHostID      = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testHostIDB     = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
	testGroupID     = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	testWorkspaceID = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	testLeaseID     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	testLeaseIDB    = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	testLeaseIDC    = "cccccccc-dddd-4eee-8fff-000000000000"
	testCreatedAt   = "2026-08-19T04:00:00.000Z"
	testChangedAt   = "2026-08-19T04:05:00.000Z"
	testCheckpoint  = "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656"
)

const zeroDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// specRecordExample is the SPEC Section 5.1 direct example with a
// yolo creation profile. Builders below re-derive it per fixture.
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

// openTestRepository opens a repository under a test-temporary data root.
func openTestRepository(t *testing.T) *sessrepo.Repository {
	t.Helper()
	repository, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open(test root) error = %v", err)
	}
	return repository
}

func captureProfileHandoff(t *testing.T, repository *sessrepo.Repository, heads []string) (*sessckpt.Store, string) {
	t.Helper()
	store, err := sessckpt.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ref, _, err := store.Capture(repository, sessckpt.Inputs{
		OperationID:         "0198f4c8-7d40-7e55-8e6f-1234567890ad",
		SessionID:           testSessionID,
		SessionKind:         sessckpt.SessionKindDirect,
		LeaseEpoch:          1,
		LeaseID:             testLeaseID,
		CreatorHostID:       testHostID,
		WorkspaceManifestID: testCheckpoint,
		ProviderManifestID:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Boundary: sessckpt.SafeBoundary{
			ProviderID: "codex", ProviderVersion: "0.147.0", Evidence: sessckpt.EvidenceAcceptedTest,
			InputBlocked: true, ForegroundIdle: true, BackgroundIdle: true,
		},
		EventHeads: heads,
		CreatedAt:  testChangedAt,
		Extensions: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Capture(profile handoff) error = %v", err)
	}
	return store, ref.CheckpointID
}

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

// withCreation sets the Session Record creation profile.
func withCreation(profile string) func(map[string]any) {
	return func(object map[string]any) {
		object["execution_profile"] = profile
	}
}

// withSession sets the Session Record session identity.
func withSession(sessionID, name string) func(map[string]any) {
	return func(object map[string]any) {
		object["session_id"] = sessionID
		object["subject_id"] = sessionID
		object["name"] = name
	}
}

// eventOptions selects the chain position and payload of a built event.
type eventOptions struct {
	sessionID    string
	author       string
	createdAt    string
	predecessors []string
	epoch        uint64
	leaseID      string
	sequence     uint64
	eventType    string
	payload      map[string]any
}

// buildEvent derives a valid Session Event for the given chain
// position, computing the canonical digest through the production
// identity entry.
func buildEvent(t *testing.T, options eventOptions) []byte {
	t.Helper()
	author := options.author
	if author == "" {
		author = testHostID
	}
	createdAt := options.createdAt
	if createdAt == "" {
		createdAt = testCreatedAt
	}
	object := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0",
		"event_id": zeroDigest, "subject_id": options.sessionID, "session_id": options.sessionID,
		"event_type": options.eventType, "created_by_host_id": author,
		"lease_epoch": options.epoch, "lease_id": options.leaseID, "lease_sequence": options.sequence,
		"predecessors": options.predecessors, "created_at": createdAt,
		"payload": options.payload, "extensions": map[string]any{},
	}
	staged, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal staged event: %v", err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity(staged event) error = %v", err)
	}
	object["event_id"] = digest.String()
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(out); err != nil {
		t.Fatalf("VerifyObjectIdentity(built event) error = %v", err)
	}
	return out
}

// Payload builders. Each carries exactly the Section 5.2 members
// for its type.

func createdPayload(recordID string) map[string]any {
	return map[string]any{"session_record_id": recordID, "bootstrap_operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab", "first_checkpoint_operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ac"}
}

func changedPayload(from, to string, confirmed bool) map[string]any {
	return map[string]any{"from": from, "to": to, "confirmed": confirmed}
}

func launchedPayload(provider, version, profile, source, mapping string) map[string]any {
	var sourceValue any
	if source != "" {
		sourceValue = source
	}
	return map[string]any{"provider_id": provider, "provider_version": version, "execution_profile": profile, "profile_source_event_id": sourceValue, "profile_mapping": mapping}
}

func resumedPayload(checkpoint, profile, source string) map[string]any {
	var sourceValue any
	if source != "" {
		sourceValue = source
	}
	return map[string]any{"checkpoint_id": checkpoint, "execution_profile": profile, "profile_source_event_id": sourceValue, "terminal_backend": "tmux", "native_session_id": "11111111-2222-4333-8444-555555555555"}
}

func forkPayload(sourceSession, checkpoint, newRecord, mode, profile, source, provenance string) map[string]any {
	var sourceValue any
	if source != "" {
		sourceValue = source
	}
	var provenanceValue any
	if provenance != "" {
		provenanceValue = provenance
	}
	return map[string]any{"source_session_id": sourceSession, "source_checkpoint_id": checkpoint, "new_session_record_id": newRecord, "provider_fork_mode": mode, "execution_profile": profile, "profile_source_event_id": sourceValue, "source_profile_event_id": provenanceValue}
}

func taskBoardLaunchedPayload(profile, source string) map[string]any {
	var sourceValue any
	if source != "" {
		sourceValue = source
	}
	return map[string]any{"operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab", "manager_session_ref": "tb-session-1", "provider_id": "codex", "launch_mode": "primary_owner", "lease_epoch": uint64(1), "lease_id": testLeaseID, "execution_profile": profile, "profile_source_event_id": sourceValue, "board_goal_id": "PRIMARY-GOAL-1", "board_goal_revision": uint64(3), "state": "running"}
}

func checkpointAnnouncedPayload() map[string]any {
	return map[string]any{"checkpoint_id": testCheckpoint, "kind": "manual"}
}

// createTestSession persists a record with the chosen creation
// profile and returns its reference with the record digest.
func createTestSession(t *testing.T, repository *sessrepo.Repository, sessionID, name, creation string) sessrepo.SessionRef {
	t.Helper()
	return createTestSessionWithLease(t, repository, sessionID, name, creation, true)
}

// createTestSessionWithoutLease creates a session before its owner lease is
// established. It is reserved for tests that exercise the empty-store or
// lease-creation transition itself.
func createTestSessionWithoutLease(t *testing.T, repository *sessrepo.Repository, sessionID, name, creation string) sessrepo.SessionRef {
	t.Helper()
	return createTestSessionWithLease(t, repository, sessionID, name, creation, false)
}

func createTestSessionWithLease(t *testing.T, repository *sessrepo.Repository, sessionID, name, creation string, withLease bool) sessrepo.SessionRef {
	t.Helper()
	record := buildRecord(t, func(object map[string]any) {
		withSession(sessionID, name)(object)
		withCreation(creation)(object)
	})
	reference, err := repository.CreateSession(record)
	if err != nil {
		t.Fatalf("CreateSession error = %v", err)
	}
	if withLease {
		if _, err := repository.CreateLease(sessionID, sessrepo.CreateLeaseInput{
			LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt,
		}); err != nil {
			t.Fatalf("CreateLease(test owner) error = %v", err)
		}
	}
	return reference
}

// appendTestEvent builds and chains one event, returning its digest.
func appendTestEvent(t *testing.T, repository *sessrepo.Repository, sessionID string, predecessors []string, epoch uint64, leaseID string, sequence uint64, eventType string, payload map[string]any) string {
	t.Helper()
	raw := buildEvent(t, eventOptions{sessionID: sessionID, predecessors: predecessors, epoch: epoch, leaseID: leaseID, sequence: sequence, eventType: eventType, payload: payload})
	reference, err := repository.AppendEvent(sessionID, raw)
	if err != nil {
		t.Fatalf("AppendEvent(%s) error = %v", eventType, err)
	}
	return reference.EventID
}

// decodeTestChain decodes the stored record with the full chain in
// index order through the production Decode entries.
func decodeTestChain(t *testing.T, repository *sessrepo.Repository, sessionID string) (Record, []Event) {
	t.Helper()
	recordBytes, err := repository.GetRecord(sessionID)
	if err != nil {
		t.Fatalf("GetRecord error = %v", err)
	}
	record, err := DecodeRecord(recordBytes)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	indexed, err := repository.ListEvents(sessionID)
	if err != nil {
		t.Fatalf("ListEvents error = %v", err)
	}
	events := make([]Event, 0, len(indexed))
	for _, summary := range indexed {
		blob, err := repository.GetEvent(sessionID, summary.EventID)
		if err != nil {
			t.Fatalf("GetEvent error = %v", err)
		}
		event, err := DecodeEvent(blob)
		if err != nil {
			t.Fatalf("DecodeEvent error = %v", err)
		}
		events = append(events, event)
	}
	return record, events
}

// mustPairEqual fails unless the derived pair equals the want pair.
func mustPairEqual(t *testing.T, got, want Pair, what string) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("%s = %+v, want %+v", what, got, want)
	}
}
