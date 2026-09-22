package sessrepo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// Fixed test identities. Every UUIDv7 keeps the version nibble 7 and an
// RFC 4122 variant; every UUIDv4 keeps nibble 4. The session, host, group,
// and workspace values are the SPEC.md Section 5.1 example identities.
const (
	testSessionID    = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	testSessionIDB   = "0198f4c8-3e70-7a11-8a2b-1234567890ac"
	testHostID       = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testGroupID      = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	testWorkspaceID  = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	testLeaseID      = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	testLeaseIDB     = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	testLeaseIDLower = "11111111-2222-4333-8444-555555555555"
	testCreatedAt    = "2026-08-19T04:00:00.000Z"
)

const zeroDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// specRecordExample is the verbatim SPEC.md Section 5.1 normative example.
// Its record_id is the computed canonical digest, verified by
// TestCreateSessionPersistsSpecRecordFixture below.
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

// specStoppedExample is the verbatim SPEC.md Section 5.2 normative example.
const specStoppedExample = `{
  "schema": "urn:ax:schema:session-event",
  "schema_version": "1.0.0",
  "event_id": "sha256:46d2745fe7dfce856027be36e34f1cc6a56ffc846063dc4aa6aabf3f5a85bacb",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "event_type": "session.stopped",
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "lease_epoch": 4,
  "lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
  "lease_sequence": 12,
  "predecessors": [
    "sha256:7777777777777777777777777777777777777777777777777777777777777777"
  ],
  "created_at": "2026-08-19T04:08:00.000Z",
  "payload": {
    "graceful": true,
    "checkpoint_id": "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656",
    "resumable": true,
    "closure_kind": "checkpointed",
    "process_closed": true,
    "store_closed": true
  },
  "extensions": {}
}`

// openTestRepository opens a repository under a test-temporary data root.
func openTestRepository(t *testing.T) *Repository {
	t.Helper()
	repository, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open(test root) error = %v", err)
	}
	return repository
}

// buildRecord derives a valid Session Record from the SPEC example with
// the caller's mutations, recomputing the canonical digest through the
// production identity entry. A mutation that breaks the closed shape fails
// the build, so invalid vectors are crafted as raw JSON instead.
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

// eventOptions selects the chain position and payload of a built event.
type eventOptions struct {
	sessionID    string
	predecessors []string
	epoch        uint64
	leaseID      string
	sequence     uint64
	eventType    string
	payload      map[string]any
}

// buildEvent derives a valid Session Event for the given chain position,
// computing the canonical digest through the production identity entry.
func buildEvent(t *testing.T, options eventOptions) []byte {
	t.Helper()
	object := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0",
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

// Payload builders for the event types the chain tests use. Each carries
// exactly the Section 5.2 members for its type.
func createdPayload(recordID string) map[string]any {
	return map[string]any{"session_record_id": recordID, "bootstrap_operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab", "first_checkpoint_operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ac"}
}

func idlePayload() map[string]any {
	return map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}
}

func checkpointPayload() map[string]any {
	return map[string]any{"checkpoint_id": "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656", "kind": "manual"}
}

// createTestSession persists the SPEC record and returns its reference.
func createTestSession(t *testing.T, repository *Repository) SessionRef {
	t.Helper()
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("CreateSession(SPEC record) error = %v", err)
	}
	return reference
}

// appendTestEvent appends one valid event at the given chain position and
// returns its reference and bytes.
func appendTestEvent(t *testing.T, repository *Repository, sessionID string, predecessors []string, epoch uint64, leaseID string, sequence uint64, eventType string, payload map[string]any) (EventRef, []byte) {
	t.Helper()
	raw := buildEvent(t, eventOptions{sessionID: sessionID, predecessors: predecessors, epoch: epoch, leaseID: leaseID, sequence: sequence, eventType: eventType, payload: payload})
	reference, err := repository.AppendEvent(sessionID, raw)
	if err != nil {
		t.Fatalf("AppendEvent(seq %d) error = %v", sequence, err)
	}
	return reference, raw
}

// mustErrorIs fails unless err matches the sentinel with errors.Is. Every
// negative test below names its arm this way, so a weakened gate that
// returns a different error — or none — fails the test.
func mustErrorIs(t *testing.T, err error, sentinel error, what string) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("%s error = %v, want errors.Is %v", what, err, sentinel)
	}
}

// mustParkedEntry fails unless summary is the per-session parked channel
// for the named session: Parked with the session directory name, a
// recoverable_parked_state blocking reason, and the expected retry hint.
// A listing that skips the parked session, returns it healthy, or fails
// the whole repository instead fails here.
func mustParkedEntry(t *testing.T, summary SessionSummary, sessionID, retryHint, what string) {
	t.Helper()
	if !summary.Parked {
		t.Fatalf("%s = %+v, want a parked entry", what, summary)
	}
	if summary.SessionID != sessionID {
		t.Fatalf("%s SessionID = %q, want %q", what, summary.SessionID, sessionID)
	}
	if summary.BlockingReason == "" || !strings.Contains(summary.BlockingReason, "recoverable_parked_state") {
		t.Fatalf("%s BlockingReason = %q, want a recoverable_parked_state reason", what, summary.BlockingReason)
	}
	if summary.RetryHint != retryHint {
		t.Fatalf("%s RetryHint = %q, want %q", what, summary.RetryHint, retryHint)
	}
}

// mustHealthyEntry fails unless summary is a healthy listing entry.
func mustHealthyEntry(t *testing.T, summary SessionSummary, sessionID, what string) {
	t.Helper()
	if summary.Parked {
		t.Fatalf("%s = %+v, want a healthy entry", what, summary)
	}
	if summary.SessionID != sessionID {
		t.Fatalf("%s SessionID = %q, want %q", what, summary.SessionID, sessionID)
	}
	if summary.BlockingReason != "" || summary.RetryHint != "" {
		t.Fatalf("%s carries parked fields %+v, want empty", what, summary)
	}
}

func TestCreateSessionPersistsSpecRecordFixture(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	if reference.SessionID != testSessionID {
		t.Fatalf("SessionID = %q, want %q", reference.SessionID, testSessionID)
	}
	if reference.RecordID != "sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772" {
		t.Fatalf("RecordID = %q, want the SPEC example digest", reference.RecordID)
	}
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord error = %v", err)
	}
	var wantCanonical, gotCanonical any
	if err := json.Unmarshal([]byte(specRecordExample), &wantCanonical); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stored, &gotCanonical); err != nil {
		t.Fatal(err)
	}
	wantBytes, _ := json.Marshal(wantCanonical)
	gotBytes, _ := json.Marshal(gotCanonical)
	if string(wantBytes) != string(gotBytes) {
		t.Fatal("stored record differs from the fixture")
	}
}

func TestSessionRecordReloadsByteIdenticallyAcrossProcessBoundary(t *testing.T) {
	root := t.TempDir()
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	reference := createTestSession(t, first)
	firstEvent, firstBytes := appendTestEvent(t, first, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	// A second handle over the same root is a new process for file-backed
	// state: no memory carries over.
	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := second.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord across boundary error = %v", err)
	}
	if string(stored) != specRecordExample && !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("record bytes differ across the process boundary")
	}
	events, err := second.ListEvents(testSessionID)
	if err != nil {
		t.Fatalf("ListEvents across boundary error = %v", err)
	}
	if len(events) != 1 || events[0].EventID != firstEvent.EventID {
		t.Fatalf("reloaded chain = %+v, want the one appended event", events)
	}
	storedEvent, err := second.GetEvent(testSessionID, firstEvent.EventID)
	if err != nil {
		t.Fatalf("GetEvent across boundary error = %v", err)
	}
	if string(storedEvent) != string(firstBytes) {
		t.Fatal("event bytes differ across the process boundary")
	}
	// The chain continues on the new handle: sequence 2 appends cleanly.
	secondRef, _ := appendTestEvent(t, second, testSessionID, []string{firstEvent.EventID}, 1, testLeaseID, 2, "session.idle", idlePayload())
	if secondRef.Position != 1 || secondRef.LeaseSequence != 2 {
		t.Fatalf("continued ref = %+v, want position 1 sequence 2", secondRef)
	}
}

func jsonEqual(t *testing.T, left, right []byte) bool {
	t.Helper()
	var l, r any
	if err := json.Unmarshal(left, &l); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &r); err != nil {
		return false
	}
	lb, _ := json.Marshal(l)
	rb, _ := json.Marshal(r)
	return string(lb) == string(rb)
}

func TestAppendEventChainInOrder(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	second, _ := appendTestEvent(t, repository, testSessionID, []string{first.EventID}, 1, testLeaseID, 2, "session.idle", idlePayload())
	third, _ := appendTestEvent(t, repository, testSessionID, []string{second.EventID}, 1, testLeaseID, 3, "checkpoint.created", checkpointPayload())
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("chain length = %d, want 3", len(events))
	}
	want := []EventRef{first, second, third}
	for index, ref := range want {
		if events[index].EventID != ref.EventID || events[index].LeaseSequence != uint64(index+1) {
			t.Fatalf("chain[%d] = %+v, want %+v", index, events[index], ref)
		}
	}
	if events[0].EventType != "session.created" || events[2].EventType != "checkpoint.created" {
		t.Fatalf("event types = %q %q %q", events[0].EventType, events[1].EventType, events[2].EventType)
	}
}

func TestAppendEventIdempotentRetry(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(reference.RecordID)})
	first, err := repository.AppendEvent(testSessionID, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.AppendEvent(testSessionID, raw)
	if err != nil {
		t.Fatalf("idempotent retry error = %v", err)
	}
	if first != second {
		t.Fatalf("retry ref = %+v, want %+v", second, first)
	}
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain length after retry = %d, want 1", len(events))
	}
}

func TestListSessionsExposesStoredIdentityAndChainHead(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	other := buildRecord(t, func(object map[string]any) {
		object["session_id"] = testSessionIDB
		object["subject_id"] = testSessionIDB
		object["name"] = "worker"
	})
	otherRef, err := repository.CreateSession(other)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	summaries, err := repository.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("sessions = %d, want 2", len(summaries))
	}
	if summaries[0].SessionID != testSessionID || summaries[1].SessionID != testSessionIDB {
		t.Fatalf("session order = %q %q, want ID order", summaries[0].SessionID, summaries[1].SessionID)
	}
	if summaries[0].Name != "payments-api" || summaries[0].Kind != "direct" || summaries[0].ProviderID != "codex" {
		t.Fatalf("first summary = %+v, want stored identity", summaries[0])
	}
	if summaries[0].EventCount != 1 || summaries[0].TailEvent != first.EventID {
		t.Fatalf("first head = %+v, want count 1 tail %s", summaries[0], first.EventID)
	}
	if summaries[1].RecordID != otherRef.RecordID || summaries[1].EventCount != 0 || summaries[1].TailEvent != "" {
		t.Fatalf("second summary = %+v, want empty chain", summaries[1])
	}
	// A stray non-session file in the namespace is ignored, never listed.
	if err := os.WriteFile(filepath.Join(repository.root, "stray.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	summaries, err = repository.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("sessions with stray file = %d, want 2", len(summaries))
	}
}

func TestOpenRefusesNonAbsoluteRoot(t *testing.T) {
	_, err := Open("relative/data")
	mustErrorIs(t, err, ErrRepositoryPath, "Open(relative)")
}

func TestOpenRefusesFileAsRoot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Open(file)
	mustErrorIs(t, err, ErrRepositoryPath, "Open(file)")
}

func TestCreateSessionRefusesMalformedFrame(t *testing.T) {
	repository := openTestRepository(t)
	_, err := repository.CreateSession([]byte(`{"schema": "urn:ax:schema:session-record",`))
	mustErrorIs(t, err, ErrInvalidRecord, "CreateSession(truncated)")
}

func TestCreateSessionRefusesForeignSchema(t *testing.T) {
	repository := openTestRepository(t)
	// A valid session EVENT is not a session record: the schema gate fires
	// before identity is ever consulted.
	event := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{"sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772"}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.CreateSession(event)
	mustErrorIs(t, err, ErrInvalidRecord, "CreateSession(event)")
}

// TestCreateSessionRefusesMissingSessionID pins the ErrInvalidRecord
// sentinel for a record without a session_id. Attribution bound (N2):
// the object is mutated after the digest is computed, so the canonical
// owner's verification refuses first with the same sentinel; the sessrepo
// UUIDv7 member arm is subsumed-by-construction for this class.
func TestCreateSessionRefusesMissingSessionID(t *testing.T) {
	repository := openTestRepository(t)
	var object map[string]any
	if err := json.Unmarshal([]byte(specRecordExample), &object); err != nil {
		t.Fatal(err)
	}
	delete(object, "session_id")
	raw, _ := json.Marshal(object)
	_, err := repository.CreateSession(raw)
	mustErrorIs(t, err, ErrInvalidRecord, "CreateSession(no session_id)")
}

func TestCreateSessionRefusesDigestMismatch(t *testing.T) {
	repository := openTestRepository(t)
	tampered := strings.Replace(specRecordExample, `"name": "payments-api"`, `"name": "payments-apj"`, 1)
	if tampered == specRecordExample {
		t.Fatal("tamper did not apply")
	}
	_, err := repository.CreateSession([]byte(tampered))
	mustErrorIs(t, err, ErrInvalidRecord, "CreateSession(tampered)")
}

func TestCreateSessionRefusesDuplicateAndRewrite(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	// Exact duplicate create is refused: records are immutable and the
	// repository offers no rewrite path.
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(duplicate)")
	// A different record under the same session ID is also refused: history
	// cannot be replaced, only extended by events.
	collision := buildRecord(t, func(object map[string]any) {
		object["name"] = "payments-api-v2"
	})
	// Force the colliding session ID while keeping a valid digest.
	var object map[string]any
	if err := json.Unmarshal(collision, &object); err != nil {
		t.Fatal(err)
	}
	object["session_id"] = testSessionID
	object["subject_id"] = testSessionID
	collision = reidentifyRecord(t, object)
	_, err = repository.CreateSession(collision)
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(collision)")
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("stored record changed after refused rewrite")
	}
}

// reidentifyRecord recomputes the canonical digest after test-side surgery.
func reidentifyRecord(t *testing.T, object map[string]any) []byte {
	t.Helper()
	object["record_id"] = zeroDigest
	staged, _ := json.Marshal(object)
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		t.Fatalf("reidentify record: %v", err)
	}
	object["record_id"] = digest.String()
	out, _ := json.Marshal(object)
	return out
}

func TestAppendEventRefusesUnknownSession(t *testing.T) {
	repository := openTestRepository(t)
	event := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{"sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772"}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.AppendEvent(testSessionID, event)
	mustErrorIs(t, err, ErrUnknownSession, "AppendEvent(unknown session)")
}

func TestAppendEventRefusesMalformedFrame(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	_, err := repository.AppendEvent(testSessionID, []byte(`{"schema": "urn:ax:schema:session-event",`))
	mustErrorIs(t, err, ErrInvalidEvent, "AppendEvent(truncated)")
}

func TestAppendEventRefusesForeignSchema(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	_, err := repository.AppendEvent(testSessionID, []byte(specRecordExample))
	mustErrorIs(t, err, ErrInvalidEvent, "AppendEvent(record)")
}

// TestAppendEventRefusesBadMember exercises the ErrInvalidEvent sentinel
// across malformed member classes. Attribution bound (reviewer N2,
// probe P4): these vectors mutate the object after buildEvent computed a
// valid digest, so the canonical owner's identity verification refuses
// them one arm later with the same sentinel — decodeSessionEvent runs the
// sessrepo member arms (sequence bounds, lease UUIDv4, non-empty
// predecessors, event type presence) before Verify, and the owner rejects
// every one of these classes on any input, so the member arms are
// subsumed-by-construction here, exactly like the load verify-error arm
// stated in the outcome. The tests below pin
// the sentinel at the production AppendEvent entry; they do not attribute
// to the member clause.
func TestAppendEventRefusesBadMember(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	vectors := []struct {
		name   string
		mutate func(object map[string]any)
	}{
		{"zero sequence", func(object map[string]any) { object["lease_sequence"] = 0 }},
		{"unsafe sequence", func(object map[string]any) { object["lease_sequence"] = uint64(1 << 53) }},
		{"non-v4 lease", func(object map[string]any) { object["lease_id"] = testSessionID }},
		{"missing predecessors", func(object map[string]any) { delete(object, "predecessors") }},
		{"malformed predecessor", func(object map[string]any) { object["predecessors"] = []string{"not-a-digest"} }},
		{"missing event type", func(object map[string]any) { delete(object, "event_type") }},
	}
	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
			var object map[string]any
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			vector.mutate(object)
			mutated, _ := json.Marshal(object)
			_, err := repository.AppendEvent(testSessionID, mutated)
			mustErrorIs(t, err, ErrInvalidEvent, "AppendEvent("+vector.name+")")
		})
	}
}

func TestAppendEventRefusesDigestMismatch(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	tampered := strings.Replace(string(raw), `"foreground_idle":true`, `"foreground_idle":false`, 1)
	if tampered == string(raw) {
		t.Fatal("tamper did not apply")
	}
	_, err := repository.AppendEvent(testSessionID, []byte(tampered))
	mustErrorIs(t, err, ErrInvalidEvent, "AppendEvent(tampered)")
}

func TestAppendEventRefusesSessionMismatch(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	foreign := buildEvent(t, eventOptions{sessionID: testSessionIDB, predecessors: []string{"sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772"}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.AppendEvent(testSessionID, foreign)
	mustErrorIs(t, err, ErrInvalidEvent, "AppendEvent(session mismatch)")
}

func TestAppendEventRefusesSequenceGap(t *testing.T) {
	t.Run("skipped sequence", func(t *testing.T) {
		repository := openTestRepository(t)
		reference := createTestSession(t, repository)
		first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
		skipped := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{first.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 3, eventType: "session.idle", payload: idlePayload()})
		_, err := repository.AppendEvent(testSessionID, skipped)
		mustErrorIs(t, err, ErrSequenceGap, "AppendEvent(skipped)")
	})
	t.Run("first sequence past one", func(t *testing.T) {
		repository := openTestRepository(t)
		reference := createTestSession(t, repository)
		raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "session.created", payload: createdPayload(reference.RecordID)})
		_, err := repository.AppendEvent(testSessionID, raw)
		mustErrorIs(t, err, ErrSequenceGap, "AppendEvent(first past one)")
	})
	t.Run("successor lease past one", func(t *testing.T) {
		repository := openTestRepository(t)
		reference := createTestSession(t, repository)
		first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
		next := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{first.EventID}, epoch: 2, leaseID: testLeaseIDB, sequence: 2, eventType: "session.idle", payload: idlePayload()})
		_, err := repository.AppendEvent(testSessionID, next)
		mustErrorIs(t, err, ErrSequenceGap, "AppendEvent(successor past one)")
	})
	t.Run("spec stopped fixture is a gap, not an invalid event", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		if _, _, err := canonicaljson.VerifyObjectIdentity([]byte(specStoppedExample)); err != nil {
			t.Fatalf("SPEC event fixture must verify: %v", err)
		}
		_, err := repository.AppendEvent(testSessionID, []byte(specStoppedExample))
		mustErrorIs(t, err, ErrSequenceGap, "AppendEvent(spec stopped)")
	})
}

func TestAppendEventRefusesSequenceRepeat(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	// A different event at an already chained sequence is a repeat, and the
	// two-events-at-one-sequence fork the spec forbids.
	repeat := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.AppendEvent(testSessionID, repeat)
	mustErrorIs(t, err, ErrSequenceRepeat, "AppendEvent(repeat)")
	second, _ := appendTestEvent(t, repository, testSessionID, []string{first.EventID}, 1, testLeaseID, 2, "session.idle", idlePayload())
	stale := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{first.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	_, err = repository.AppendEvent(testSessionID, stale)
	mustErrorIs(t, err, ErrSequenceRepeat, "AppendEvent(old sequence)")
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[1].EventID != second.EventID {
		t.Fatalf("chain after refused repeats = %+v", events)
	}
}

func TestAppendEventRefusesFirstPredecessorMismatch(t *testing.T) {
	t.Run("unrelated predecessor", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{zeroDigest}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload("sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772")})
		_, err := repository.AppendEvent(testSessionID, raw)
		mustErrorIs(t, err, ErrPredecessorLink, "AppendEvent(unrelated predecessor)")
	})
	t.Run("record plus extra predecessor", func(t *testing.T) {
		repository := openTestRepository(t)
		reference := createTestSession(t, repository)
		extra := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID, extra}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(reference.RecordID)})
		_, err := repository.AppendEvent(testSessionID, raw)
		mustErrorIs(t, err, ErrPredecessorLink, "AppendEvent(extra predecessor)")
	})
}

func TestAppendEventRefusesLaterPredecessorMismatch(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	_, _ = appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	// The second event must reference the first, not skip back to the record.
	raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.AppendEvent(testSessionID, raw)
	mustErrorIs(t, err, ErrPredecessorLink, "AppendEvent(skipped head)")
}

func TestAppendEventPreservesDivergentBranchWithoutApplying(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	_, _ = appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	// A same-epoch second lease is the concurrent-takeover fork: its bytes
	// are preserved in the immutable namespace but never linked into the
	// authoritative chain.
	fork := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseIDB, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.AppendEvent(testSessionID, fork)
	mustErrorIs(t, err, ErrDivergentBranch, "AppendEvent(divergent)")
	var forkID struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(fork, &forkID); err != nil {
		t.Fatal(err)
	}
	preserved, err := os.ReadFile(blobPathFor(t, repository, testSessionID, forkID.EventID))
	if err != nil {
		t.Fatalf("divergent blob was not preserved: %v", err)
	}
	if string(preserved) != string(fork) {
		t.Fatal("preserved divergent bytes differ")
	}
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain after divergent fork = %d events, want 1", len(events))
	}
}

func TestAppendEventRefusesStaleLeaseEpoch(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	_, _ = appendTestEvent(t, repository, testSessionID, []string{first.EventID}, 2, testLeaseIDB, 1, "session.idle", idlePayload())
	// An event from the superseded epoch cannot rejoin the chain.
	late := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{first.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "session.idle", payload: idlePayload()})
	_, err := repository.AppendEvent(testSessionID, late)
	mustErrorIs(t, err, ErrStaleLease, "AppendEvent(stale epoch)")
}

// TestAppendEventRefusesSupersededLeaseWhileTailStillMatches pins the
// production append admission gate behind the exact stale-lease shape from
// review probe 9: successor B is already the stored winner, but the
// authoritative tail still belongs to lease A. The losing profile.changed
// event must be refused and preserved without changing the chain.
func TestAppendEventRefusesSupersededLeaseWhileTailStillMatches(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	firstLease := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, firstLease.Epoch, firstLease.LeaseID, 1, "session.created", createdPayload(reference.RecordID))
	secondLease := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: firstLease.RecordID}, successorLeaseFixture())
	losing := buildEvent(t, eventOptions{
		sessionID:    testSessionID,
		predecessors: []string{first.EventID},
		epoch:        firstLease.Epoch,
		leaseID:      firstLease.LeaseID,
		sequence:     2,
		eventType:    "profile.changed",
		payload:      map[string]any{"from": "standard", "to": "yolo", "confirmed": true},
	})
	_, err := repository.AppendEvent(testSessionID, losing)
	mustErrorIs(t, err, ErrStaleLease, "AppendEvent(profile.changed under superseded lease)")
	if secondLease.Epoch != 2 || secondLease.LeaseID != testLeaseIDB {
		t.Fatalf("successor lease = %+v, want epoch 2 lease %s", secondLease, testLeaseIDB)
	}
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].LeaseID != firstLease.LeaseID {
		t.Fatalf("chain after stale append = %+v, want the original lease-A tail only", events)
	}
	var eventID struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(losing, &eventID); err != nil {
		t.Fatal(err)
	}
	preserved, err := os.ReadFile(blobPathFor(t, repository, testSessionID, eventID.EventID))
	if err != nil {
		t.Fatalf("losing event was not preserved: %v", err)
	}
	if string(preserved) != string(losing) {
		t.Fatal("preserved losing profile.changed bytes differ")
	}

	t.Run("same-epoch losing lease", func(t *testing.T) {
		repository := openTestRepository(t)
		reference := createTestSession(t, repository)
		firstLease := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
		first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, firstLease.Epoch, firstLease.LeaseID, 1, "session.created", createdPayload(reference.RecordID))
		secondLease := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: firstLease.RecordID}, successorLeaseFixture())
		second, _ := appendTestEvent(t, repository, testSessionID, []string{first.EventID}, secondLease.Epoch, secondLease.LeaseID, 1, "session.idle", idlePayload())
		losing := buildEvent(t, eventOptions{
			sessionID:    testSessionID,
			predecessors: []string{second.EventID},
			epoch:        secondLease.Epoch,
			leaseID:      testLeaseIDC,
			sequence:     2,
			eventType:    "session.idle",
			payload:      idlePayload(),
		})
		_, err := repository.AppendEvent(testSessionID, losing)
		mustErrorIs(t, err, ErrDivergentBranch, "AppendEvent(same-epoch losing lease)")
	})
}

// TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches exercises
// the other winner-gate arm with a tail that already belongs to the losing
// same-epoch lease. The chain-only bootstrap is built before lease records
// exist; once winner B is installed, a new C event must not extend that old
// tail even though checkAppend alone would accept its sequence and link.
func TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches(t *testing.T) {
	for _, losingLeaseID := range []string{testLeaseIDC, testLeaseIDLower} {
		t.Run(losingLeaseID, func(t *testing.T) {
			repository := openTestRepository(t)
			reference := createTestSession(t, repository)
			tail, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 2, losingLeaseID, 1, "session.created", createdPayload(reference.RecordID))
			firstLease := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
			secondLease := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: firstLease.RecordID}, successorLeaseFixture())
			if secondLease.Epoch != 2 || secondLease.LeaseID != testLeaseIDB {
				t.Fatalf("successor lease = %+v, want epoch 2 lease %s", secondLease, testLeaseIDB)
			}
			losing := buildEvent(t, eventOptions{
				sessionID:    testSessionID,
				predecessors: []string{tail.EventID},
				epoch:        2,
				leaseID:      losingLeaseID,
				sequence:     2,
				eventType:    "session.idle",
				payload:      idlePayload(),
			})
			_, err := repository.AppendEvent(testSessionID, losing)
			mustErrorIs(t, err, ErrDivergentBranch, "AppendEvent(same-epoch losing tail)")
			events, err := repository.ListEvents(testSessionID)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].LeaseID != losingLeaseID {
				t.Fatalf("chain after same-epoch losing append = %+v, want the original losing tail only", events)
			}
			var eventID struct {
				EventID string `json:"event_id"`
			}
			if err := json.Unmarshal(losing, &eventID); err != nil {
				t.Fatal(err)
			}
			preserved, err := os.ReadFile(blobPathFor(t, repository, testSessionID, eventID.EventID))
			if err != nil {
				t.Fatalf("same-epoch losing event was not preserved: %v", err)
			}
			if string(preserved) != string(losing) {
				t.Fatal("preserved same-epoch losing bytes differ")
			}
		})
	}
}

func TestAppendEventRefusesDisagreeingDigestPathBytes(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	next := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{first.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "session.idle", payload: idlePayload()})
	// Plant disagreeing bytes at the path the event's digest names, then
	// append the genuine event: the store must refuse instead of trusting
	// either copy.
	var parsed struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(next, &parsed); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blobPathFor(t, repository, testSessionID, parsed.EventID), []byte(`{"forged":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := repository.AppendEvent(testSessionID, next)
	mustErrorIs(t, err, ErrChainCorrupt, "AppendEvent(disagreeing bytes)")
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain after disagreeing plant = %d events, want 1", len(events))
	}
}

// blobPathFor resolves the content-addressed path of an event blob.
func blobPathFor(t *testing.T, repository *Repository, sessionID, eventID string) string {
	t.Helper()
	hex, ok := strings.CutPrefix(eventID, "sha256:")
	if !ok {
		t.Fatalf("event id %q has no digest prefix", eventID)
	}
	return filepath.Join(repository.sessionDir(sessionID), "events", hex+".json")
}

// TestAppendEventRefusesSameLengthDisagreeingBytes is the same-length
// vector for the installEventBlob content-equality gate (chain.go:442):
// the planted bytes differ from the genuine event in exactly one bit but
// carry the identical byte length, so a narrowed equality that compares
// lengths only would admit them as already installed, write the chain
// entry, and return success over a session that is unreadable on the
// next load. Production must refuse with ErrChainCorrupt, leave the
// chain at one event, and preserve the planted bytes.
func TestAppendEventRefusesSameLengthDisagreeingBytes(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	next := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{first.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "session.idle", payload: idlePayload()})
	var parsed struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(next, &parsed); err != nil {
		t.Fatal(err)
	}
	plant := append([]byte(nil), next...)
	plant[len(plant)/2] ^= 0x01
	if len(plant) != len(next) {
		t.Fatalf("plant lengths differ (%d vs %d); the narrowing mutant would not be admitted", len(plant), len(next))
	}
	if string(plant) == string(next) {
		t.Fatal("plant equals the genuine event; the vector admits nothing")
	}
	if err := os.WriteFile(blobPathFor(t, repository, testSessionID, parsed.EventID), plant, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := repository.AppendEvent(testSessionID, next)
	mustErrorIs(t, err, ErrChainCorrupt, "AppendEvent(same-length disagreeing bytes)")
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain after same-length plant = %d events, want 1", len(events))
	}
	preserved, err := os.ReadFile(blobPathFor(t, repository, testSessionID, parsed.EventID))
	if err != nil {
		t.Fatal(err)
	}
	if string(preserved) != string(plant) {
		t.Fatal("planted bytes were replaced instead of refused")
	}
	_, err = repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord after refused same-length install error = %v", err)
	}
}

// The content-equality class census lives in census_test.go
// (auditSessrepoEquality): it derives every bytes.Equal site and every
// string-conversion comparison over the whole package through invcore and
// diffs the derived set against the registered ledger in both directions.
// The same-length vectors for the two registered sites are
// TestCreateSessionRefusesResumeWithDifferingBytes (resumeCreateLocked)
// and TestAppendEventRefusesSameLengthDisagreeingBytes (installEventBlob);
// their narrowing rows are N10 and N13 in the mutation battery.

func TestGetEventLookup(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, firstBytes := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	stored, err := repository.GetEvent(testSessionID, first.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(firstBytes) {
		t.Fatal("GetEvent bytes differ from appended bytes")
	}
	_, err = repository.GetEvent(testSessionID, "not-a-digest")
	mustErrorIs(t, err, ErrUnknownEvent, "GetEvent(malformed)")
	_, err = repository.GetEvent(testSessionID, zeroDigest)
	mustErrorIs(t, err, ErrUnknownEvent, "GetEvent(absent)")
}

func TestResolveLocalNameAndUUID(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	summary, err := repository.Resolve("payments-api")
	if err != nil {
		t.Fatalf("Resolve(exact) error = %v", err)
	}
	if summary.SessionID != testSessionID {
		t.Fatalf("resolved = %q, want %q", summary.SessionID, testSessionID)
	}
	// Section 2.3 uses folding for uniqueness, but requires exact names.
	_, err = repository.Resolve("Payments-API")
	mustErrorIs(t, err, ErrNameNotFound, "Resolve(non-exact case variant)")
	byUUID, err := repository.Resolve(testSessionID)
	if err != nil {
		t.Fatalf("Resolve(uuid) error = %v", err)
	}
	if byUUID.RecordID != summary.RecordID {
		t.Fatalf("uuid resolve = %+v, want %+v", byUUID, summary)
	}
	_, err = repository.Resolve("no-such-session")
	mustErrorIs(t, err, ErrNameNotFound, "Resolve(absent)")
	// A UUID-shaped query with no session is not found, not ambiguous.
	_, err = repository.Resolve(testSessionIDB)
	mustErrorIs(t, err, ErrNameNotFound, "Resolve(absent uuid)")
}

func TestResolveRefusesCaseFoldCollision(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	// The colliding name carries Y, past T: a narrowed fold that stops at
	// T would miss this collision, so the vector witnesses the full A-Z
	// range rather than only its head.
	other := buildRecord(t, func(object map[string]any) {
		object["session_id"] = testSessionIDB
		object["subject_id"] = testSessionIDB
		object["name"] = "PAYMENTS-API"
	})
	if _, err := repository.CreateSession(other); err != nil {
		t.Fatal(err)
	}
	_, err := repository.Resolve("payments-api")
	mustErrorIs(t, err, ErrNameAmbiguous, "Resolve(collision)")
	_, err = repository.Resolve("PAYMENTS-API")
	mustErrorIs(t, err, ErrNameAmbiguous, "Resolve(collision upper)")
}

// plantCorruptedSession creates a session with two chained events and
// applies one corruption, returning the repository for a load attempt.
func plantCorruptedSession(t *testing.T, corrupt func(t *testing.T, repository *Repository)) *Repository {
	t.Helper()
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	_, _ = appendTestEvent(t, repository, testSessionID, []string{first.EventID}, 1, testLeaseID, 2, "session.idle", idlePayload())
	corrupt(t, repository)
	return repository
}

func sessionPaths(t *testing.T, repository *Repository) (directory, chain, record string) {
	t.Helper()
	directory = repository.sessionDir(testSessionID)
	return directory, filepath.Join(directory, "chain.json"), filepath.Join(directory, "record.json")
}

func TestLoadRefusesMissingChainIndex(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, chain, _ := sessionPaths(t, repository)
		if err := os.Remove(chain); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(missing chain)")
}

func TestLoadRefusesMalformedChainIndex(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, chain, _ := sessionPaths(t, repository)
		if err := os.WriteFile(chain, []byte(`{"record_id":`), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(malformed chain)")
}

func TestLoadRefusesMissingRecord(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, _, record := sessionPaths(t, repository)
		if err := os.Remove(record); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(missing record)")
}

func TestLoadRefusesTamperedRecord(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, _, record := sessionPaths(t, repository)
		raw, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		tampered := strings.Replace(string(raw), `"name": "payments-api"`, `"name": "payments-apj"`, 1)
		if tampered == string(raw) {
			t.Fatal("tamper did not apply")
		}
		if err := os.WriteFile(record, []byte(tampered), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(tampered record)")
}

func TestLoadRefusesDetachedChainIndex(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, chain, _ := sessionPaths(t, repository)
		raw, err := os.ReadFile(chain)
		if err != nil {
			t.Fatal(err)
		}
		var index map[string]any
		if err := json.Unmarshal(raw, &index); err != nil {
			t.Fatal(err)
		}
		index["record_id"] = zeroDigest
		out, _ := json.Marshal(index)
		if err := os.WriteFile(chain, out, 0o600); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(detached index)")
}

func TestLoadRefusesMissingEventBlob(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		events, err := repository.ListEvents(testSessionID)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(blobPathFor(t, repository, testSessionID, events[0].EventID)); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(missing blob)")
}

func TestLoadRefusesTamperedEventBlob(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		events, err := repository.ListEvents(testSessionID)
		if err != nil {
			t.Fatal(err)
		}
		path := blobPathFor(t, repository, testSessionID, events[1].EventID)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tampered := strings.Replace(string(raw), `"foreground_idle":true`, `"foreground_idle":false`, 1)
		if tampered == string(raw) {
			t.Fatal("tamper did not apply")
		}
		if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(tampered blob)")
}

func TestLoadRefusesReorderedChainIndex(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, chain, _ := sessionPaths(t, repository)
		raw, err := os.ReadFile(chain)
		if err != nil {
			t.Fatal(err)
		}
		var index map[string]any
		if err := json.Unmarshal(raw, &index); err != nil {
			t.Fatal(err)
		}
		events := index["events"].([]any)
		if len(events) != 2 {
			t.Fatalf("planted chain has %d events", len(events))
		}
		index["events"] = []any{events[1], events[0]}
		out, _ := json.Marshal(index)
		if err := os.WriteFile(chain, out, 0o600); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(reordered index)")
}

func TestLoadRefusesUnknownSession(t *testing.T) {
	repository := openTestRepository(t)
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrUnknownSession, "GetRecord(absent)")
	_, err = repository.ListEvents(testSessionID)
	mustErrorIs(t, err, ErrUnknownSession, "ListEvents(absent)")
}

// TestReadEntriesRefuseTraversalSessionID drives an unvalidated session ID
// through the three read entries. sessionDir joins the caller string with
// no grammar check, so "../outside" escapes the sessions root — and the
// escape still refuses, because the stored record's UUIDv7 session ID can
// never equal the traversal string at the record/session binding
// (loadSessionLocked). A plant outside the root carrying a full valid
// session reaches exactly that binding, so the refusal names
// ErrChainCorrupt there rather than ErrUnknownSession at a missing
// directory. Dropping the binding half of that check admits the escape.
func TestReadEntriesRefuseTraversalSessionID(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	escaped := repository.sessionDir("../outside")
	relative, err := filepath.Rel(repository.root, escaped)
	if err != nil {
		t.Fatal(err)
	}
	if relative == "." || !strings.HasPrefix(relative, "..") {
		t.Fatalf("sessionDir(../outside) = %q (rel %q), want an escape from %q", escaped, relative, repository.root)
	}
	plantSessionTree(t, repository.sessionDir(testSessionID), escaped)
	_, err = repository.GetRecord("../outside")
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(traversal)")
	_, err = repository.ListEvents("../outside")
	mustErrorIs(t, err, ErrChainCorrupt, "ListEvents(traversal)")
	_, err = repository.GetEvent("../outside", first.EventID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetEvent(traversal)")
	if _, err := repository.GetRecord(testSessionID); err != nil {
		t.Fatalf("GetRecord(healthy) after traversal error = %v", err)
	}
}

// plantSessionTree copies one session directory tree to another path. The
// traversal test uses it to stage a valid session outside the sessions
// root, so the load reaches the record/session binding.
func plantSessionTree(t *testing.T, source, target string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(target, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"record.json", "chain.json"} {
		raw, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(source, "events"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(source, "events", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "events", entry.Name()), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// TestListSessionsReportsTornHintOnUnreadableRecord pins the
// absence-vs-read-failure split in parkedSummary: a session whose
// record.json exists but cannot be read (here a directory, so the read
// fails deterministically with a non-IsNotExist error) is not a bare
// directory — no CreateSession record retry can heal it — and the entry
// carries the torn-store remedy instead of the bare-directory retry.
func TestListSessionsReportsTornHintOnUnreadableRecord(t *testing.T) {
	repository := openTestRepository(t)
	directory := repository.sessionDir(testSessionID)
	if err := os.MkdirAll(filepath.Join(directory, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "record.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	summaries, err := repository.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions(unreadable record) error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("ListSessions(unreadable record) = %d entries, want 1 parked", len(summaries))
	}
	mustParkedEntry(t, summaries[0], testSessionID, retryHintForTornStore, "ListSessions(unreadable record)")
}

// TestListSessionsReportsParkedSessionPerSession replaces the prior
// closed-listing rule: a torn store (chain.json present but unverifiable)
// is returned as a parked entry with the torn-store remedy hint — never
// as a whole-repository error and never skipped. Per-session reads still
// refuse; name resolution finds no live session.
func TestListSessionsReportsParkedSessionPerSession(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, chain, _ := sessionPaths(t, repository)
		if err := os.WriteFile(chain, []byte(`{"record_id":`), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	summaries, err := repository.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions(torn) error = %v, want a per-session parked entry", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("ListSessions(torn) = %d entries, want 1 parked", len(summaries))
	}
	mustParkedEntry(t, summaries[0], testSessionID, retryHintForTornStore, "ListSessions(torn)")
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(torn)")
	_, err = repository.Resolve("payments-api")
	mustErrorIs(t, err, ErrNameNotFound, "Resolve(torn parked name)")
	_, err = repository.Resolve(testSessionID)
	mustErrorIs(t, err, ErrNameNotFound, "Resolve(torn parked id)")
}

// TestListSessionsKeepsHealthySessionsBesideParked is the reviewer probe
// P5 as a committed test: one session parked at CreateStepRecord (record
// durable, chain absent) beside a healthy sibling. The listing returns
// both — the healthy session first-class and the parked one with its
// blocking reason and identical-retry hint — and Resolve routes the
// healthy sibling. A differing record for the parked ID does not heal
// (ErrSessionExists); the byte-identical original does.
func TestListSessionsKeepsHealthySessionsBesideParked(t *testing.T) {
	repository := openTestRepository(t)
	healthyRecord := buildRecord(t, func(object map[string]any) {
		object["session_id"] = testSessionIDB
		object["subject_id"] = testSessionIDB
		object["name"] = "healthy-one"
	})
	if _, err := repository.CreateSession(healthyRecord); err != nil {
		t.Fatal(err)
	}
	parkedDirectory := repository.sessionDir(testSessionID)
	if err := os.MkdirAll(filepath.Join(parkedDirectory, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parkedDirectory, "record.json"), []byte(specRecordExample), 0o600); err != nil {
		t.Fatal(err)
	}
	summaries, err := repository.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions(parked sibling) error = %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("ListSessions(parked sibling) = %d entries, want healthy + parked", len(summaries))
	}
	byID := map[string]SessionSummary{}
	for _, summary := range summaries {
		byID[summary.SessionID] = summary
	}
	mustHealthyEntry(t, byID[testSessionIDB], testSessionIDB, "ListSessions(healthy sibling)")
	mustParkedEntry(t, byID[testSessionID], testSessionID, retryHintForParkedRecord, "ListSessions(parked sibling)")
	if byID[testSessionID].Name != "payments-api" || byID[testSessionID].RecordID == "" {
		t.Fatalf("parked identity = %+v, want the parked record members", byID[testSessionID])
	}
	resolved, err := repository.Resolve("healthy-one")
	if err != nil {
		t.Fatalf("Resolve(healthy sibling beside parked) error = %v", err)
	}
	if resolved.SessionID != testSessionIDB || resolved.Parked {
		t.Fatalf("Resolve(healthy sibling beside parked) = %+v", resolved)
	}
	collision := buildRecord(t, func(object map[string]any) {
		object["name"] = "payments-api-v2"
	})
	var object map[string]any
	if err := json.Unmarshal(collision, &object); err != nil {
		t.Fatal(err)
	}
	object["session_id"] = testSessionID
	object["subject_id"] = testSessionID
	collision = reidentifyRecord(t, object)
	_, err = repository.CreateSession(collision)
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(differing retry does not heal)")
	summaries, err = repository.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("ListSessions(after refused heal) = %d entries, want 2", len(summaries))
	}
	if _, err := repository.CreateSession([]byte(specRecordExample)); err != nil {
		t.Fatalf("identical retry error = %v", err)
	}
	summaries, err = repository.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("ListSessions(after identical retry) = %d entries, want 2", len(summaries))
	}
	for _, summary := range summaries {
		mustHealthyEntry(t, summary, summary.SessionID, "ListSessions(healed)")
	}
}

// TestResolveSkipsParkedSessions pins that parked entries never
// participate in name resolution: a parked record carrying the same name
// as a healthy session does not collide, and a query naming only the
// parked session is not found.
func TestResolveSkipsParkedSessions(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	parkedDirectory := repository.sessionDir(testSessionIDB)
	if err := os.MkdirAll(filepath.Join(parkedDirectory, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	parkedRecord := buildRecord(t, func(object map[string]any) {
		object["session_id"] = testSessionIDB
		object["subject_id"] = testSessionIDB
	})
	if err := os.WriteFile(filepath.Join(parkedDirectory, "record.json"), parkedRecord, 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := repository.Resolve("payments-api")
	if err != nil {
		t.Fatalf("Resolve(healthy name beside same-name parked) error = %v", err)
	}
	if resolved.SessionID != testSessionID {
		t.Fatalf("Resolve(healthy name beside same-name parked) = %q, want %q", resolved.SessionID, testSessionID)
	}
	_, err = repository.Resolve(testSessionIDB)
	mustErrorIs(t, err, ErrNameNotFound, "Resolve(parked id only)")
}

func TestListAndResolveRefuseMissingRoot(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	if err := os.RemoveAll(repository.root); err != nil {
		t.Fatal(err)
	}
	_, err := repository.ListSessions()
	mustErrorIs(t, err, ErrRepositoryPath, "ListSessions(missing root)")
	_, err = repository.Resolve("payments-api")
	mustErrorIs(t, err, ErrRepositoryPath, "Resolve(missing root)")
}

// specTaskBoardRecordExample is the verbatim SPEC.md Section 5.1
// task-board normative example: the same Launch Plan shape with the tagged
// task_board variant instead of null.
const specTaskBoardRecordExample = `{
  "schema": "urn:ax:schema:session-record",
  "schema_version": "1.0.0",
  "record_id": "sha256:0acd3e31635372e176f8f37b1b74aa0ebdcf2d1e4ac40d43adb8e462079b34a2",
  "subject_id": "0198f4c8-9f60-7077-8071-1234567890ab",
  "session_id": "0198f4c8-9f60-7077-8071-1234567890ab",
  "name": "qwen-investigation",
  "kind": "task_board",
  "created_at": "2026-08-19T04:01:00.000Z",
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "provider_id": "qwen",
  "workspace_group_id": "0198f4c8-af70-7188-8172-1234567890ab",
  "execution_profile": "standard",
  "launch_plan": {
    "argv": ["task-board", "qwen", "TASK-260819-example"],
    "cwd_workspace_id": "0198f4c8-b080-7299-8273-1234567890ab",
    "cwd_relative": ".",
    "env_names": [],
    "env_literals": {},
    "contains_secrets": false,
    "extensions": {}
  },
  "task_board": {
    "bridge_protocol_version": "1.0.0",
    "board": {
      "kind": "local",
      "logical_id": "agent-session-manager-spec",
      "remote_url": null,
      "extensions": {}
    },
    "task_element_id": "TASK-260819-example",
    "launch_mode": "tracked_prompt",
    "manager_session_ref": null,
    "board_goal": null,
    "native_goal_binding": "prompt",
    "extensions": {}
  },
  "fork_provenance": null,
  "extensions": {}
}`

func TestBothRecordKindsCoexist(t *testing.T) {
	repository := openTestRepository(t)
	direct := createTestSession(t, repository)
	board, err := repository.CreateSession([]byte(specTaskBoardRecordExample))
	if err != nil {
		t.Fatalf("CreateSession(task_board SPEC record) error = %v", err)
	}
	if board.RecordID != "sha256:0acd3e31635372e176f8f37b1b74aa0ebdcf2d1e4ac40d43adb8e462079b34a2" {
		t.Fatalf("task_board RecordID = %q, want the SPEC example digest", board.RecordID)
	}
	const boardSession = "0198f4c8-9f60-7077-8071-1234567890ab"
	first, _ := appendTestEvent(t, repository, boardSession, []string{board.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(board.RecordID))
	if first.Position != 0 {
		t.Fatalf("task_board chain ref = %+v", first)
	}
	summaries, err := repository.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("sessions = %d, want 2", len(summaries))
	}
	byKind := map[string]SessionSummary{}
	for _, summary := range summaries {
		byKind[summary.Kind] = summary
	}
	if byKind["direct"].SessionID != direct.SessionID || byKind["task_board"].SessionID != boardSession {
		t.Fatalf("kinds = %+v", summaries)
	}
	if byKind["task_board"].ProviderID != "qwen" || byKind["task_board"].EventCount != 1 {
		t.Fatalf("task_board summary = %+v", byKind["task_board"])
	}
	resolved, err := repository.Resolve("qwen-investigation")
	if err != nil {
		t.Fatalf("Resolve(task_board name) error = %v", err)
	}
	if resolved.SessionID != boardSession {
		t.Fatalf("resolved = %q, want %q", resolved.SessionID, boardSession)
	}
}

func TestCreateSessionResumesInterruptedCreate(t *testing.T) {
	repository := openTestRepository(t)
	// Simulate a crash between the record write and the chain write: the
	// record is durable, the index is absent.
	directory := repository.sessionDir(testSessionID)
	if err := os.MkdirAll(filepath.Join(directory, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "record.json"), []byte(specRecordExample), 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("resumed CreateSession error = %v", err)
	}
	if reference.SessionID != testSessionID {
		t.Fatalf("resumed SessionID = %q", reference.SessionID)
	}
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("resumed record differs")
	}
	// A resumed-then-completed create is terminal: the next create refuses.
	_, err = repository.CreateSession([]byte(specRecordExample))
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(after resume)")
}

// specProviderIdentityExample is the verbatim SPEC.md Section 5.5
// normative example (Antigravity backend_conversation_uuid). It is
// duplicated here rather than imported from the provhost test package so
// this leaf's load-path bound stays pinned even if that package moves;
// TestProviderIdentityPlantVerifiesUnderRecordID fails loudly on any
// drift from the owner's example.
const specProviderIdentityExample = `{
  "schema": "urn:ax:schema:provider-identity",
  "schema_version": "1.0.0",
  "record_id": "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "provider_id": "antigravity",
  "provider_version": "1.1.14",
  "provider_version_range": ">=1.1.14 <1.2.0",
  "native_session_id": "11111111-2222-4333-8444-555555555555",
  "identity_kind": "backend_conversation_uuid",
  "logical_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab",
  "backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad",
  "opaque_identity": {},
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "created_at": "2026-08-19T04:09:45.000Z",
  "extensions": {}
}`

const specProviderIdentityDigest = "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2"

// TestProviderIdentityPlantVerifiesUnderRecordID pins what the plant
// below is: the SPEC Section 5.5 example verifies with a nil error under
// the record_id self field, so a load path that recomputes its digest
// matches it, and only the self-field arm can refuse it.
func TestProviderIdentityPlantVerifiesUnderRecordID(t *testing.T) {
	verified, field, err := canonicaljson.VerifyObjectIdentity([]byte(specProviderIdentityExample))
	if err != nil {
		t.Fatalf("VerifyObjectIdentity(provider-identity plant) error = %v", err)
	}
	if field != canonicaljson.SelfRecordID {
		t.Fatalf("VerifyObjectIdentity(provider-identity plant) field = %q, want record_id", string(field))
	}
	if verified.String() != specProviderIdentityDigest {
		t.Fatalf("VerifyObjectIdentity(provider-identity plant) = %q, want %q", verified.String(), specProviderIdentityDigest)
	}
}

// TestLoadRefusesProviderIdentityBlobAtEventPath pins the rewritten
// attestation bound at the load site: entry decodes gate schema before
// Verify, but the load path re-verifies stored blobs through Verify and
// refuses a foreign-schema record at the self-field arm. The planted
// blob verifies cleanly (see the pin above), carries the indexed digest,
// and replays valid routing members — the field arm is the only clause
// that can refuse it, so weakening that arm admits the plant.
func TestLoadRefusesProviderIdentityBlobAtEventPath(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	_, chainPath, _ := sessionPaths(t, repository)
	index := map[string]any{
		"record_id": reference.RecordID,
		"events": []any{map[string]any{
			"event_id": specProviderIdentityDigest, "event_type": "session.created",
			"lease_epoch": 1, "lease_id": testLeaseID, "lease_sequence": 1,
			"predecessors": []any{reference.RecordID},
		}},
	}
	out, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chainPath, out, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blobPathFor(t, repository, testSessionID, specProviderIdentityDigest), []byte(specProviderIdentityExample), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(provider-identity blob)")
	_, err = repository.ListEvents(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "ListEvents(provider-identity blob)")
}

// TestLoadRefusesSwappedEventBlobs drives the content-addressed
// substitution the digest arm exists for: two chained events whose blob
// files trade places. Both blobs verify individually and both carry the
// event_id self field, so the digest comparison is the only clause that
// can refuse. Weakening it admits the substitution with a nil error.
func TestLoadRefusesSwappedEventBlobs(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, firstBytes := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	second, secondBytes := appendTestEvent(t, repository, testSessionID, []string{first.EventID}, 1, testLeaseID, 2, "session.idle", idlePayload())
	if err := os.WriteFile(blobPathFor(t, repository, testSessionID, first.EventID), secondBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blobPathFor(t, repository, testSessionID, second.EventID), firstBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := repository.ListEvents(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "ListEvents(swapped blobs)")
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(swapped blobs)")
}

// TestLoadRefusesSubstitutedFirstEventBlob covers the digest arm at
// chain position 0: a single-event chain whose one blob is replaced by a
// different individually-valid event linking the same record. Both blobs
// verify with the event_id self field, so only the digest comparison can
// refuse; a narrowing that exempts position 0 or sequence 1 admits it.
func TestLoadRefusesSubstitutedFirstEventBlob(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	first, firstBytes := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	substitute := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()})
	if string(substitute) == string(firstBytes) {
		t.Fatal("substitute equals the chained event; the vector admits nothing")
	}
	var subID struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(substitute, &subID); err != nil {
		t.Fatal(err)
	}
	if subID.EventID == first.EventID {
		t.Fatal("substitute digest equals the indexed digest; the vector admits nothing")
	}
	if err := os.WriteFile(blobPathFor(t, repository, testSessionID, first.EventID), substitute, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := repository.ListEvents(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "ListEvents(substituted first blob)")
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(substituted first blob)")
}

// TestCreateSessionRefusesResumeWithDifferingBytes proves the resume
// byte-equality is load-bearing: an interrupted create (record durable,
// chain absent) retried with a different record for the same session ID
// is refused as an existing session, the parked record survives
// byte-identically, and no chain is installed. The retry carries the
// same byte length as the parked record, so a narrowed equality that
// compares lengths only would admit it.
func TestCreateSessionRefusesResumeWithDifferingBytes(t *testing.T) {
	repository := openTestRepository(t)
	parked := buildRecord(t, nil)
	sameLength := buildRecord(t, func(object map[string]any) {
		object["name"] = "payments-apx"
	})
	if len(sameLength) != len(parked) {
		t.Fatalf("plant lengths differ (%d vs %d); the narrowing mutant would not be admitted", len(parked), len(sameLength))
	}
	directory := repository.sessionDir(testSessionID)
	if err := os.MkdirAll(filepath.Join(directory, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "record.json"), parked, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := repository.CreateSession(sameLength)
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(resume collision, same length)")
	longer := buildRecord(t, func(object map[string]any) {
		object["name"] = "payments-api-v2"
	})
	_, err = repository.CreateSession(longer)
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(resume collision, longer)")
	stored, err := os.ReadFile(filepath.Join(directory, "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(parked) {
		t.Fatal("parked record changed after refused resume")
	}
	if _, err := os.Stat(filepath.Join(directory, "chain.json")); !os.IsNotExist(err) {
		t.Fatalf("chain.json state after refused resume = %v, want absent", err)
	}
}

// TestCreateSessionRefusesResumeOfChainWithoutRecord pins the other
// resume guard: a chain index with no record beneath it is torn foreign
// state — no crash ordering in CreateSession produces it — and a retry
// must refuse it as an existing session rather than installing a record
// beside it and overwriting the index. The refusal funnels through the
// existing session-exists site; reads of the same state fail closed as
// corrupt through the load path.
func TestCreateSessionRefusesResumeOfChainWithoutRecord(t *testing.T) {
	repository := openTestRepository(t)
	directory := repository.sessionDir(testSessionID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	foreign, err := json.Marshal(map[string]any{"record_id": zeroDigest, "events": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "chain.json"), foreign, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = repository.CreateSession([]byte(specRecordExample))
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(chain without record)")
	if _, err := os.Stat(filepath.Join(directory, "record.json")); !os.IsNotExist(err) {
		t.Fatalf("record.json state after refused resume = %v, want absent", err)
	}
	restored, err := os.ReadFile(filepath.Join(directory, "chain.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(foreign) {
		t.Fatal("foreign chain index changed after refused resume")
	}
}

// TestLoadRefusesMismatchedSessionBinding isolates the session-binding
// half of the record/chain binding check: a valid record for another
// session stored under this session's directory, with the index rebound
// to its digest so the record-digest half stays silent. Only the
// session-ID comparison can refuse it.
func TestLoadRefusesMismatchedSessionBinding(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	other := buildRecord(t, func(object map[string]any) {
		object["session_id"] = testSessionIDB
		object["subject_id"] = testSessionIDB
	})
	var parsed struct {
		RecordID string `json:"record_id"`
	}
	if err := json.Unmarshal(other, &parsed); err != nil {
		t.Fatal(err)
	}
	_, chainPath, recordPath := sessionPaths(t, repository)
	if err := os.WriteFile(recordPath, other, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]any
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	index["record_id"] = parsed.RecordID
	out, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chainPath, out, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(session mismatch)")
}

// TestLoadRefusesDetachedChainIndexWithReboundPredecessors isolates the
// record-digest half of the binding check: the index names an unrelated
// record digest, and the first indexed predecessors are rebound to that
// same digest so the continuity fold stays silent. Only the
// record-digest comparison can refuse it; the older detached-index test
// passes through the continuity arm instead and does not witness this
// clause.
func TestLoadRefusesDetachedChainIndexWithReboundPredecessors(t *testing.T) {
	repository := plantCorruptedSession(t, func(t *testing.T, repository *Repository) {
		_, chain, _ := sessionPaths(t, repository)
		raw, err := os.ReadFile(chain)
		if err != nil {
			t.Fatal(err)
		}
		var index map[string]any
		if err := json.Unmarshal(raw, &index); err != nil {
			t.Fatal(err)
		}
		index["record_id"] = zeroDigest
		events, ok := index["events"].([]any)
		if !ok || len(events) != 2 {
			t.Fatalf("planted chain has %v events, want 2", index["events"])
		}
		first, ok := events[0].(map[string]any)
		if !ok {
			t.Fatal("planted first event is not an object")
		}
		first["predecessors"] = []any{zeroDigest}
		out, err := json.Marshal(index)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(chain, out, 0o600); err != nil {
			t.Fatal(err)
		}
	})
	_, err := repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "load(detached index, rebound predecessors)")
}

// TestAppendEventRefusesDisagreeingBytesAcrossHandles drives the
// no-replace install through a fresh handle: the session is created and
// the disagreeing bytes are planted through one repository, and the
// adversarial append arrives through a second Open over the same root
// that shares no mutex and no memory with the first. The refusal below
// comes from the filesystem exclusive create alone, which is the whole
// cross-process claim — the in-process mutex cannot see a second
// process. The planted bytes must survive: a truncate would overwrite
// them and extend the chain instead.
func TestAppendEventRefusesDisagreeingBytesAcrossHandles(t *testing.T) {
	root := t.TempDir()
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	reference := createTestSession(t, first)
	firstEvent, _ := appendTestEvent(t, first, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	next := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{firstEvent.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "session.idle", payload: idlePayload()})
	var parsed struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(next, &parsed); err != nil {
		t.Fatal(err)
	}
	plantPath := blobPathFor(t, first, testSessionID, parsed.EventID)
	if err := os.WriteFile(plantPath, []byte(`{"forged":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = second.AppendEvent(testSessionID, next)
	mustErrorIs(t, err, ErrChainCorrupt, "AppendEvent(disagreeing bytes, fresh handle)")
	preserved, err := os.ReadFile(plantPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(preserved) != `{"forged":true}` {
		t.Fatal("planted bytes were replaced instead of refused")
	}
	events, err := second.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain after refused install = %d events, want 1", len(events))
	}
}
