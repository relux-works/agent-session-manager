package sessckpt

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessquery"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Fixed identities shared by every fixture. UUIDv7 grammar carries
// the version nibble; the lease tokens are UUIDv4.
const (
	testSessionID = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	testHostA     = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testHostB     = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
	testLease     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	testLeaseB    = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	testOpA       = "0198f4c8-0a10-71aa-8111-1234567890ab"
	testOpB       = "0198f4c8-0a20-72bb-8222-1234567890ab"
	testCreatedAt = "2026-08-19T04:09:30.000Z"
)

// Fixed manifest legs. Well-formed digests the capture entry never
// resolves: resolution belongs to the manifest owners, not to the
// checkpoint closure.
const (
	testWorkspaceManifest = "sha256:a98ca90522b4de30e4aaaf9bf50529d09e15a817ffa67f94552fb313d1a1ad2e"
	testProviderManifest  = "sha256:1e817955dcc529e282ab31f91c99561d03b3c5642282d2e0a0e05b0f60dd0f91"
	testBoardBundle       = "sha256:0af7b44e7063375a0f06e546fd820438c72607f191c14be78d35c8ffa109844f"
	testOtherManifest     = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

// specCheckpointExample is the verbatim SPEC Section 5.4 normative
// example.
const specCheckpointExample = `{
  "schema": "urn:ax:schema:checkpoint",
  "schema_version": "1.0.0",
  "checkpoint_id": "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "lease_epoch": 4,
  "lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
  "safe_boundary": {
    "provider_id": "codex",
    "provider_version": "0.147.0",
    "evidence": "accepted_test",
    "input_blocked": true,
    "foreground_idle": true,
    "background_idle": true,
    "open_processes": 0,
    "open_database_handles": 0
  },
  "event_heads": [
    "sha256:7777777777777777777777777777777777777777777777777777777777777777"
  ],
  "workspace_manifest_id": "sha256:a98ca90522b4de30e4aaaf9bf50529d09e15a817ffa67f94552fb313d1a1ad2e",
  "provider_manifest_id": "sha256:1e817955dcc529e282ab31f91c99561d03b3c5642282d2e0a0e05b0f60dd0f91",
  "task_board_bundle_id": null,
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "created_at": "2026-08-19T04:09:30.000Z",
  "status": "validated",
  "extensions": {}
}`

// specSessionRecord is the Section 5.1 fixture shape with the test
// session identity, identified through the canonical owner.
const specSessionRecord = `{
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

// identify stamps the omit-self digest through the canonical owner.
// A fixture failure surfaces here, never as a production admission.
func identify(t *testing.T, value map[string]any, field string) []byte {
	t.Helper()
	value[field] = zeroDigest
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	value[field] = digest.String()
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func decodeObject(t *testing.T, raw string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

// testBoundary returns the all-quiescent terminal evidence leg.
func testBoundary() SafeBoundary {
	return SafeBoundary{
		ProviderID:          "codex",
		ProviderVersion:     "0.147.0",
		Evidence:            EvidenceAcceptedTest,
		InputBlocked:        true,
		ForegroundIdle:      true,
		BackgroundIdle:      true,
		OpenProcesses:       0,
		OpenDatabaseHandles: 0,
	}
}

// testInputs returns the direct-kind closure over the given heads
// and operation.
func testInputs(heads []string, operation string) Inputs {
	return Inputs{
		OperationID:         operation,
		SessionID:           testSessionID,
		SessionKind:         SessionKindDirect,
		LeaseEpoch:          1,
		LeaseID:             testLease,
		CreatorHostID:       testHostA,
		WorkspaceManifestID: testWorkspaceManifest,
		ProviderManifestID:  testProviderManifest,
		TaskBoardBundleID:   "",
		Boundary:            testBoundary(),
		EventHeads:          heads,
		CreatedAt:           testCreatedAt,
		Extensions:          map[string]string{},
	}
}

// openTestStore binds an empty checkpoint store in a temp dir.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// chainFixture persists a session with its bootstrap event and
// returns the repository with the created event digest.
func chainFixture(t *testing.T) (*sessrepo.Repository, string) {
	t.Helper()
	repo, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	record := decodeObject(t, specSessionRecord)
	record["session_id"], record["subject_id"], record["name"] = testSessionID, testSessionID, "alpha"
	ref, err := repo.CreateSession(identify(t, record, "record_id"))
	if err != nil {
		t.Fatal(err)
	}
	created := appendChainEvent(t, repo, "session.created", 1, testLease, 1, ref.RecordID, map[string]any{
		"session_record_id": ref.RecordID, "bootstrap_operation_id": testHostA, "first_checkpoint_operation_id": testHostB,
	})
	return repo, created
}

// appendChainEvent chains one session event under an explicit
// envelope lease and returns its digest.
func appendChainEvent(t *testing.T, repo *sessrepo.Repository, typ string, epoch uint64, leaseID string, sequence int, predecessor string, payload map[string]any) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "event_type": typ, "created_by_host_id": testHostA,
		"lease_epoch": epoch, "lease_id": leaseID, "lease_sequence": sequence, "predecessors": []string{predecessor},
		"created_at": "2026-08-19T04:00:00.000Z", "payload": payload, "extensions": map[string]any{},
	}
	ref, err := repo.AppendEvent(testSessionID, identify(t, value, "event_id"))
	if err != nil {
		t.Fatal(err)
	}
	return ref.EventID
}

// leaseRecordBytes builds one epoch-1 lease naming the given
// checkpoint, identified through the canonical owner.
func leaseRecordBytes(t *testing.T, leaseID, holder, checkpoint string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "lease_id": leaseID, "epoch": float64(1),
		"holder_host_id": holder, "predecessor_lease_id": nil, "reason": "recovery",
		"checkpoint_id": checkpoint, "issued_by_host_id": holder, "created_by_host_id": holder,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
	return identify(t, value, "record_id")
}

// mustCapture captures through the production entry or fails.
func mustCapture(t *testing.T, store *Store, chain *sessrepo.Repository, inputs Inputs) (CheckpointRef, []byte) {
	t.Helper()
	ref, raw, err := store.Capture(chain, inputs)
	if err != nil {
		t.Fatalf("Capture error = %v", err)
	}
	return ref, raw
}

// mustInvalid asserts the incompatible_schema refusal class on
// both layers: the package sentinel and the owner sentinel the
// SPEC CP-N1..CP-N4 language requires.
func mustInvalid(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want invalid checkpoint closure", what)
	}
	if !errors.Is(err, ErrInvalidCheckpoint) {
		t.Fatalf("%s error = %v, want ErrInvalidCheckpoint", what, err)
	}
	if !errors.Is(err, canonicaljson.ErrInvalidIdentity) {
		t.Fatalf("%s error = %v, want incompatible_schema (ErrInvalidIdentity)", what, err)
	}
}

func TestCaptureDirectInstallsAttestedRecord(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	ref, raw := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	if ref.OperationID != testOpA {
		t.Fatalf("OperationID = %q, want %q", ref.OperationID, testOpA)
	}
	if _, err := scalar.ParseDigest(ref.CheckpointID); err != nil {
		t.Fatalf("CheckpointID %q: %v", ref.CheckpointID, err)
	}
	// The installed bytes attest through the sessrepo owner with
	// the checkpoint self field.
	digest, field, err := sessrepo.AttestCheckpointRecord(raw)
	if err != nil {
		t.Fatalf("AttestCheckpointRecord error = %v", err)
	}
	if field != canonicaljson.SelfCheckpointID || digest.String() != ref.CheckpointID {
		t.Fatalf("attested %v %q, want checkpoint_id %q", field, digest, ref.CheckpointID)
	}
	// Get returns the byte-identical record.
	stored, err := store.Get(ref.CheckpointID)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if string(stored) != string(raw) {
		t.Fatalf("Get returned %d bytes, want the %d captured bytes", len(stored), len(raw))
	}
	// The closed member set matches the normative example exactly.
	var example, got map[string]any
	if err := json.Unmarshal([]byte(specCheckpointExample), &example); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for key := range example {
		if _, ok := got[key]; !ok {
			t.Fatalf("captured record misses member %q", key)
		}
	}
	for key := range got {
		if _, ok := example[key]; !ok {
			t.Fatalf("captured record carries extra member %q", key)
		}
	}
}

func TestCaptureTaskBoardVariantInstalls(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	inputs := testInputs([]string{created}, testOpA)
	inputs.SessionKind = SessionKindTaskBoard
	inputs.ProviderManifestID = ""
	inputs.TaskBoardBundleID = testBoardBundle
	ref, raw := mustCapture(t, store, chain, inputs)
	if _, _, err := sessrepo.AttestCheckpointRecord(raw); err != nil {
		t.Fatalf("AttestCheckpointRecord error = %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if value["provider_manifest_id"] != nil {
		t.Fatalf("task_board record provider_manifest_id = %v, want null", value["provider_manifest_id"])
	}
	if value["task_board_bundle_id"] == nil {
		t.Fatalf("task_board record task_board_bundle_id is null, want %q", testBoardBundle)
	}
	if ref.CheckpointID == "" {
		t.Fatalf("empty checkpoint id")
	}
}

func TestSpecExampleAttestsThroughConsumerOwner(t *testing.T) {
	// The verbatim SPEC Section 5.4 normative example attests
	// through the same sessrepo owner the capture entry uses, so
	// the exact contract fixture and the implementation agree on
	// the closed shape.
	digest, field, err := sessrepo.AttestCheckpointRecord([]byte(specCheckpointExample))
	if err != nil {
		t.Fatalf("AttestCheckpointRecord(spec example) error = %v", err)
	}
	if field != canonicaljson.SelfCheckpointID {
		t.Fatalf("example identity field = %q, want checkpoint_id", string(field))
	}
	if digest.String() != "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656" {
		t.Fatalf("example digest = %q", digest)
	}
}

func TestCaptureIdenticalClosuresShareIdentity(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	first, _ := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// A different operation over the identical closure mints no
	// second identity: the checkpoint digest is a pure function
	// of the closure.
	second, _ := mustCapture(t, store, chain, testInputs([]string{created}, testOpB))
	if first.CheckpointID != second.CheckpointID {
		t.Fatalf("identical closures share no identity: %q vs %q", first.CheckpointID, second.CheckpointID)
	}
	if first.OperationID == second.OperationID {
		t.Fatalf("operations collide: %q", first.OperationID)
	}
}

func TestCaptureMovedInputsChangeIdentity(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	first, _ := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// The same checkpoint with different inputs is a different
	// checkpoint: moving one manifest leg changes the digest.
	moved := testInputs([]string{created}, testOpB)
	moved.WorkspaceManifestID = testOtherManifest
	second, _ := mustCapture(t, store, chain, moved)
	if first.CheckpointID == second.CheckpointID {
		t.Fatalf("moved inputs kept identity %q", first.CheckpointID)
	}
}

func TestCaptureReplayIsIdempotent(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	inputs := testInputs([]string{created}, testOpA)
	first, firstRaw := mustCapture(t, store, chain, inputs)
	second, secondRaw, err := store.Capture(chain, inputs)
	if err != nil {
		t.Fatalf("replay Capture error = %v", err)
	}
	if first != second {
		t.Fatalf("replay ref = %+v, want %+v", second, first)
	}
	if string(firstRaw) != string(secondRaw) {
		t.Fatalf("replay returned different bytes")
	}
	// Exactly one blob and one receipt exist: the replay wrote
	// nothing.
	blobs, err := filepath.Glob(filepath.Join(store.root, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 {
		t.Fatalf("blobs = %d, want 1", len(blobs))
	}
	receipts, err := filepath.Glob(filepath.Join(store.root, "operations", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 1 {
		t.Fatalf("receipts = %d, want 1", len(receipts))
	}
}

func TestCaptureAdmitsThroughWinningLeaseConsumer(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	ref, raw := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// The captured record admits through the landed sessquery
	// consumer: an epoch-1 lease naming it builds and revalidates.
	reader := &sessquery.Reader{Local: chain, LocalHostID: testHostA}
	reader.CheckpointRecords = append(reader.CheckpointRecords, raw)
	reader.LeaseRecords = append(reader.LeaseRecords, leaseRecordBytes(t, testLease, testHostA, ref.CheckpointID))
	plan, err := reader.BuildPlan(sessquery.PlanArgs{Selector: "alpha", Action: sessquery.ActionStatus})
	if err != nil {
		t.Fatalf("BuildPlan(captured checkpoint) error = %v", err)
	}
	if plan.LeaseEpoch != 1 || plan.LeaseID != testLease || plan.OwnerHostID != testHostA {
		t.Fatalf("plan triple: %+v", plan)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("captured plan is not current: %v", err)
	}
}

func TestCaptureSuccessorAdmitsThroughConsumer(t *testing.T) {
	chain, created := chainFixture(t)
	launched := appendChainEvent(t, chain, "provider.launched", 1, testLease, 2, created, map[string]any{
		"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo",
		"profile_source_event_id": nil, "profile_mapping": "default",
	})
	// The transfer event restarts the lease sequence under the new
	// epoch, mirroring the landed succession fixtures.
	_ = appendChainEvent(t, chain, "lease.transferred", 2, testLeaseB, 1, launched, map[string]any{
		"operation_id": testHostB, "from_host_id": testHostA, "to_host_id": testHostA,
		"predecessor_lease_id": testLease, "new_lease_id": testLeaseB,
	})
	store := openTestStore(t)
	// The checkpoint is taken under the epoch-1 predecessor and
	// names the latest epoch-1 head; the epoch-2 successor names
	// it for its session and predecessor lease.
	ref, raw := mustCapture(t, store, chain, testInputs([]string{launched}, testOpA))
	reader := &sessquery.Reader{Local: chain, LocalHostID: testHostA}
	reader.CheckpointRecords = append(reader.CheckpointRecords, raw)
	reader.LeaseRecords = append(reader.LeaseRecords,
		leaseCreateBytes(t, testLease, testHostA),
		leaseSuccessorBytes(t, testLeaseB, testLease, ref.CheckpointID),
	)
	plan, err := reader.BuildPlan(sessquery.PlanArgs{Selector: "alpha", Action: sessquery.ActionStatus})
	if err != nil {
		t.Fatalf("BuildPlan(successor over captured checkpoint) error = %v", err)
	}
	if plan.LeaseEpoch != 2 || plan.LeaseID != testLeaseB {
		t.Fatalf("successor plan: %+v", plan)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("successor plan is not current: %v", err)
	}
}

// leaseCreateBytes builds the epoch-1 create lease with no
// checkpoint reference.
func leaseCreateBytes(t *testing.T, leaseID, holder string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "lease_id": leaseID, "epoch": float64(1),
		"holder_host_id": holder, "predecessor_lease_id": nil, "reason": "create",
		"checkpoint_id": nil, "issued_by_host_id": holder, "created_by_host_id": holder,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
	return identify(t, value, "record_id")
}

// leaseSuccessorBytes builds the epoch-2 successor naming its
// predecessor and the captured checkpoint digest.
func leaseSuccessorBytes(t *testing.T, leaseID, predecessor, checkpoint string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "lease_id": leaseID, "epoch": float64(2),
		"holder_host_id": testHostA, "predecessor_lease_id": predecessor, "reason": "graceful_takeover",
		"checkpoint_id": checkpoint, "issued_by_host_id": testHostA, "created_by_host_id": testHostA,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
	return identify(t, value, "record_id")
}

func TestCaptureWrongCreatorRefusedByConsumer(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	// Capture records exactly what the caller declares: a creator
	// that is not the owning lease holder stays publishable here
	// and is refused at admission by the consumer's
	// creator-holder gate.
	inputs := testInputs([]string{created}, testOpA)
	inputs.CreatorHostID = testHostB
	_, raw := mustCapture(t, store, chain, inputs)
	reader := &sessquery.Reader{Local: chain, LocalHostID: testHostA}
	reader.CheckpointRecords = append(reader.CheckpointRecords, raw)
	checkpointID := checkpointIDOf(t, raw)
	reader.LeaseRecords = append(reader.LeaseRecords, leaseRecordBytes(t, testLease, testHostA, checkpointID))
	if _, err := reader.BuildPlan(sessquery.PlanArgs{Selector: "alpha", Action: sessquery.ActionStatus}); !errors.Is(err, sessquery.ErrObservationUnavailable) {
		t.Fatalf("wrong-creator build = %v, want selector_observation_unavailable", err)
	}
}

func checkpointIDOf(t *testing.T, raw []byte) string {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	id, _ := value["checkpoint_id"].(string)
	if id == "" {
		t.Fatalf("captured record carries no checkpoint_id")
	}
	return id
}

func TestAdmitRawRecordRoundTrip(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	_, raw := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// The same bytes admitted under a fresh operation install
	// through the same durable path.
	admitted, err := store.Admit(chain, raw, testOpB, SessionKindDirect)
	if err != nil {
		t.Fatalf("Admit error = %v", err)
	}
	if admitted.CheckpointID != checkpointIDOf(t, raw) {
		t.Fatalf("admitted %q, want %q", admitted.CheckpointID, checkpointIDOf(t, raw))
	}
	stored, err := store.Get(admitted.CheckpointID)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if string(stored) != string(raw) {
		t.Fatalf("admitted bytes differ from captured bytes")
	}
	// A byte-identical re-admit replays the recorded result.
	again, err := store.Admit(chain, raw, testOpB, SessionKindDirect)
	if err != nil {
		t.Fatalf("re-admit error = %v", err)
	}
	if again != admitted {
		t.Fatalf("re-admit = %+v, want %+v", again, admitted)
	}
}
