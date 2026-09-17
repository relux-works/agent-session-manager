package crashgate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Fixed identities shared by every fixture. UUIDv7 grammar carries
// the version nibble; the lease tokens are UUIDv4.
const (
	testMatID      = "0198f4c8-c290-73aa-9374-1234567890ab"
	testPrepareOp  = "0198f4c8-b180-72cc-9271-1234567890ab"
	testTransfer   = "0198f4c8-d3a0-74bb-9475-1234567890ab"
	testReplicaID  = "0198f4c8-8e50-7f66-8f70-2234567890ab"
	testProviderOp = "0198f4c8-e4b0-75cc-9576-1234567890ab"
	testTxID       = "0198f4c8-f5c0-76dd-9677-1234567890ab"
	testImportOp   = "0198f4c8-0a10-71aa-8111-1234567890ab"
	testOpenOp     = "0198f4c8-0a20-72bb-8222-1234567890ab"
	testAdoptOp    = "0198f4c8-0a30-73cc-8333-1234567890ab"
	testResumeOp   = "0198f4c8-0a40-74dd-8444-1234567890ab"
	testHostID     = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	testSessionID  = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	testGroupID    = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	testLeaseID    = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	testLeaseB     = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"

	testHostA = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testHostB = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
	testOpA   = "0198f4c8-0a10-71aa-8111-1234567890ab"
	testOpB   = "0198f4c8-0a20-72bb-8222-1234567890ab"

	testPlanID       = "sha256:64644a5ad573d36c0c13f44f56ef25ab93cff33001ff2a3371b082603910f2dd"
	testCheckpointID = "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656"
	testBundleID     = "sha256:0af7b44e7063375a0f06e546fd820438c72607f191c14be78d35c8ffa109844f"
	testPriorID      = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	testBlobA        = "sha256:0e442b07e3772e8f5622478242ddf5f9f197bbd6a0402cd71471db4081abb291"
	testBlobB        = "sha256:444e0fffbd825e9610ff5b199485707a0c895339ae80c15cc8a8aee41b106fda"

	testProviderToken = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5QUI"
	testOpenToken     = "b3duZXItb25seS1vcGVuLXRva2VuLTAxMjM0NTY3ODlhYmNkZWY"
	testImportToken   = "aW1wb3J0LW9ubHktYnJpZGdlLXRva2VuLTAxMjM0NTY3ODk"

	testRootWorkspace = "/srv/relux"
	testRootProvider  = "/home/ivan/.codex/sessions"
	testTxRoot        = "/home/ivan/.local/state/ax/provider-transactions/codex/0198f4c8-f5c0-76dd-9677-1234567890ab"
	testStagingRoot   = "/home/ivan/.local/state/ax/staging/0198f4c8-c290-73aa-9374-1234567890ab"

	testWorkspaceManifest = "sha256:a98ca90522b4de30e4aaaf9bf50529d09e15a817ffa67f94552fb313d1a1ad2e"
	testProviderManifest  = "sha256:1e817955dcc529e282ab31f91c99561d03b3c5642282d2e0a0e05b0f60dd0f91"
	testBoardBundle       = "sha256:0af7b44e7063375a0f06e546fd820438c72607f191c14be78d35c8ffa109844f"
	testOtherManifest     = "sha256:2222222222222222222222222222222222222222222222222222222222222222"

	testCreatedAt = "2026-08-19T04:09:30.000Z"
	zeroDigest    = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	testNative = "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab:aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
)

func str(value string) *string { return &value }

// fixedClock pins journal store time to one instant.
func fixedClock() time.Time {
	return time.Date(2026, time.August, 19, 4, 12, 0, 0, time.UTC)
}

// journalDirs binds an empty journal store in a fresh data directory
// and returns both, so the test can reopen the same directory after
// the simulated crash: hooks never survive the restart.
func journalDirs(t *testing.T) (*matjournal.Store, string) {
	t.Helper()
	dataDir := t.TempDir()
	store, err := matjournal.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store.Now = fixedClock
	return store, dataDir
}

// reopenJournal loads a fresh store over the same data directory:
// the simulated clean restart after a crash.
func reopenJournal(t *testing.T, dataDir string) *matjournal.Store {
	t.Helper()
	fresh, err := matjournal.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Now = fixedClock
	return fresh
}

// checkpointDirs binds an empty checkpoint store in a fresh data
// directory and returns both.
func checkpointDirs(t *testing.T) (*sessckpt.Store, string) {
	t.Helper()
	dataDir := t.TempDir()
	store, err := sessckpt.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return store, dataDir
}

// reopenCheckpoint loads a fresh checkpoint store over the same data
// directory: the simulated clean restart after a crash.
func reopenCheckpoint(t *testing.T, dataDir string) *sessckpt.Store {
	t.Helper()
	fresh, err := sessckpt.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

// openCheckpointEnv binds the checkpoint store under the data
// directory named by the environment variable.
func openCheckpointEnv(variable string) (*sessckpt.Store, error) {
	return sessckpt.Open(os.Getenv(variable))
}

// openJournalEnv binds the journal store under the data directory
// named by the environment variable.
func openJournalEnv(variable string) (*matjournal.Store, error) {
	return matjournal.Open(os.Getenv(variable))
}

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

// chainFixture persists a session with its bootstrap event and
// returns the repository directory with the created event digest.
func chainFixture(t *testing.T) (string, string) {
	t.Helper()
	repoDir := t.TempDir()
	repo, err := sessrepo.Open(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	record := decodeObject(t, specSessionRecord)
	record["session_id"], record["subject_id"], record["name"] = testSessionID, testSessionID, "alpha"
	ref, err := repo.CreateSession(identify(t, record, "record_id"))
	if err != nil {
		t.Fatal(err)
	}
	created := appendChainEvent(t, repo, "session.created", 1, testLeaseID, 1, ref.RecordID, map[string]any{
		"session_record_id": ref.RecordID, "bootstrap_operation_id": testHostA, "first_checkpoint_operation_id": testHostB,
	})
	return repoDir, created
}

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

func testBoundary() sessckpt.SafeBoundary {
	return sessckpt.SafeBoundary{
		ProviderID:          "codex",
		ProviderVersion:     "0.147.0",
		Evidence:            sessckpt.EvidenceAcceptedTest,
		InputBlocked:        true,
		ForegroundIdle:      true,
		BackgroundIdle:      true,
		OpenProcesses:       0,
		OpenDatabaseHandles: 0,
	}
}

// captureInputs returns the direct-kind checkpoint closure over the
// given heads and operation.
func captureInputs(heads []string, operation string) sessckpt.Inputs {
	return sessckpt.Inputs{
		OperationID:         operation,
		SessionID:           testSessionID,
		SessionKind:         sessckpt.SessionKindDirect,
		LeaseEpoch:          1,
		LeaseID:             testLeaseID,
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

// captureInputsBoard returns the task-board-kind checkpoint closure.
func captureInputsBoard(heads []string, operation string) sessckpt.Inputs {
	inputs := captureInputs(heads, operation)
	inputs.SessionKind = sessckpt.SessionKindTaskBoard
	inputs.ProviderManifestID = ""
	inputs.TaskBoardBundleID = testBoardBundle
	return inputs
}

// countCheckpointFiles counts installed blobs and receipts under a
// checkpoint data directory.
func countCheckpointFiles(t *testing.T, dataDir string) (blobs, receipts int) {
	t.Helper()
	root := filepath.Join(dataDir, "checkpoints")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			blobs++
		}
	}
	receiptEntries, err := os.ReadDir(filepath.Join(root, "operations"))
	if err != nil {
		t.Fatal(err)
	}
	return blobs, len(receiptEntries)
}

// everyCheckpointVerifies fails when any installed blob does not
// attest through the canonical owner: an interrupted capture must
// leave no partial record behind.
func everyCheckpointVerifies(t *testing.T, dataDir string) {
	t.Helper()
	root := filepath.Join(dataDir, "checkpoints")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := sessrepo.AttestCheckpointRecord(raw); err != nil {
			t.Fatalf("installed file %s does not attest: %v", entry.Name(), err)
		}
	}
}

// journalFilesExist reports whether the journal and at least one
// prepare receipt exist for the test materialization.
func journalFilesExist(t *testing.T, dataDir string) (journal, receipt bool) {
	t.Helper()
	matDir := filepath.Join(dataDir, "materializations", testMatID)
	if _, err := os.Stat(filepath.Join(matDir, "journal.json")); err == nil {
		journal = true
	}
	entries, err := os.ReadDir(filepath.Join(matDir, "operations"))
	if err == nil && len(entries) > 0 {
		receipt = true
	}
	return journal, receipt
}

func testRequestBody() []byte {
	return []byte(`{"materialization_id":"` + testMatID + `","operation_id":"` + testPrepareOp + `","plan_id":"` + testPlanID + `"}`)
}

// directInputs returns the workspace-only prepare closure: no
// provider branch, no bridge.
func directInputs() matjournal.CreateInputs {
	return matjournal.CreateInputs{
		MaterializationID:  testMatID,
		PrepareOperationID: testPrepareOp,
		RequestBody:        testRequestBody(),
		TransferID:         testTransfer,
		PlanID:             testPlanID,
		SourceCheckpointID: testCheckpointID,
		ManagedReplicaID:   testReplicaID,
		Plan: []matjournal.PlanAuthority{
			{ID: "workspace_relux", Kind: matjournal.PlanKindWorkspace, Platform: "linux", RootPath: testRootWorkspace},
		},
		HostPlatform:              "linux",
		ExpectedPriorCheckpointID: testPriorID,
		Extensions:                map[string]string{},
	}
}

// compositeInputs returns the composite prepare closure: a workspace
// authority, a provider-store authority, and a task-board bridge in
// ownership_transfer mode.
func compositeInputs() matjournal.CreateInputs {
	inputs := directInputs()
	inputs.Plan = []matjournal.PlanAuthority{
		{ID: "codex_sessions", Kind: matjournal.PlanKindProviderStore, Platform: "linux", RootPath: testRootProvider},
		{ID: "workspace_relux", Kind: matjournal.PlanKindWorkspace, Platform: "linux", RootPath: testRootWorkspace},
	}
	inputs.BundleID = testBundleID
	inputs.ActivationMode = matjournal.ModeOwnershipTransfer
	inputs.ImportOperationID = testImportOp
	inputs.OpenOperationID = testOpenOp
	inputs.AdoptOperationID = testAdoptOp
	inputs.ResumeOperationID = testResumeOp
	return inputs
}

// directProviderInputs returns a direct closure with a provider
// branch but no bridge: workspace plus provider-store authorities.
func directProviderInputs() matjournal.CreateInputs {
	inputs := directInputs()
	inputs.Plan = []matjournal.PlanAuthority{
		{ID: "codex_sessions", Kind: matjournal.PlanKindProviderStore, Platform: "linux", RootPath: testRootProvider},
		{ID: "workspace_relux", Kind: matjournal.PlanKindWorkspace, Platform: "linux", RootPath: testRootWorkspace},
	}
	return inputs
}

// boardDormantInputs returns the task-board-only closure in dormant
// replica mode: one staging authority, no workspace bytes.
func boardDormantInputs() matjournal.CreateInputs {
	return matjournal.CreateInputs{
		MaterializationID:  testMatID,
		PrepareOperationID: testPrepareOp,
		RequestBody:        testRequestBody(),
		PlanID:             testPlanID,
		SourceCheckpointID: testCheckpointID,
		Plan: []matjournal.PlanAuthority{
			{ID: "board_staging", Kind: matjournal.PlanKindTaskBoardStaging, Platform: "linux", RootPath: testStagingRoot},
		},
		HostPlatform:      "linux",
		BundleID:          testBundleID,
		ActivationMode:    matjournal.ModeDormantReplica,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		Extensions:        map[string]string{},
	}
}

// boardOwnershipInputs returns the task-board-only closure in
// ownership_transfer mode: no provider branch, no workspace bytes.
func boardOwnershipInputs() matjournal.CreateInputs {
	inputs := boardDormantInputs()
	inputs.ActivationMode = matjournal.ModeOwnershipTransfer
	return inputs
}

// preparedProvider returns the prepared provider transaction bound
// to the test journal.
func preparedProvider() matjournal.ProviderTransaction {
	return matjournal.ProviderTransaction{
		OperationID:   testProviderOp,
		TransactionID: testTxID,
		State:         matjournal.ProviderPrepared,
		RollbackToken: str(testProviderToken),
		Authority: matjournal.TransactionAuthority{
			AuthorityID:                        "provider_transaction",
			Kind:                               "provider_transaction",
			RootPath:                           testTxRoot,
			Layout:                             "provider_transaction_v1",
			Access:                             "read_write",
			MaterializationID:                  testMatID,
			ProviderID:                         "codex",
			TransactionID:                      testTxID,
			PlanID:                             testPlanID,
			SameFilesystemProviderAuthorityIDs: []string{"codex_sessions"},
		},
		LastStatusAt: "2026-08-19T04:12:10.000Z",
	}
}

// importedBoard returns the imported bridge transaction.
func importedBoard() matjournal.TaskBoardTransaction {
	return matjournal.TaskBoardTransaction{
		BundleID:          testBundleID,
		ActivationMode:    matjournal.ModeOwnershipTransfer,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		State:             matjournal.BoardImported,
		ImportToken:       str(testImportToken),
		StagedManagerRef:  str("tbm:staged:0198f4c8-7a10-7b22-8b3c-2234567890ab"),
		ImportExpiresAt:   str("2026-08-19T04:22:10.000Z"),
		LastBridgeState:   str("stopped"),
		LastStatusAt:      str("2026-08-19T04:12:10.000Z"),
		CleanupState:      "not_started",
	}
}

// openedBoard returns the opened bridge transaction matching the
// MJ-TASK-BOARD-OPENED-POS fixture members.
func openedBoard() matjournal.TaskBoardTransaction {
	return matjournal.TaskBoardTransaction{
		BundleID:          testBundleID,
		ActivationMode:    matjournal.ModeOwnershipTransfer,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		State:             matjournal.BoardOpened,
		OpenToken:         str(testOpenToken),
		DormantManagerRef: str("tbm:dormant:0198f4c8-7a10-7b22-8b3c-2234567890ab"),
		OpenExpiresAt:     str("2026-08-19T04:22:10.000Z"),
		LastBridgeState:   str("dormant"),
		LastStatusAt:      str("2026-08-19T04:12:10.000Z"),
		CleanupState:      "not_started",
	}
}

// adoptedBoard returns the adopted bridge transaction.
func adoptedBoard() matjournal.TaskBoardTransaction {
	return matjournal.TaskBoardTransaction{
		BundleID:          testBundleID,
		ActivationMode:    matjournal.ModeOwnershipTransfer,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		State:             matjournal.BoardAdopted,
		ManagerSessionRef: str("tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab"),
		AxBinding:         &matjournal.AxBinding{AxSessionID: testSessionID, LeaseEpoch: 4, LeaseID: testLeaseID},
		LastBridgeState:   str("stopped"),
		LastStatusAt:      str("2026-08-19T04:12:10.000Z"),
		CleanupState:      "retained_active",
	}
}

// resumedBoard returns the resumed bridge transaction.
func resumedBoard() matjournal.TaskBoardTransaction {
	board := adoptedBoard()
	board.State = matjournal.BoardResumed
	board.LastBridgeState = str("running")
	return board
}

func importedBoardDormant() matjournal.TaskBoardTransaction {
	board := importedBoard()
	board.ActivationMode = matjournal.ModeDormantReplica
	return board
}

func openedBoardDormant() matjournal.TaskBoardTransaction {
	board := openedBoard()
	board.ActivationMode = matjournal.ModeDormantReplica
	return board
}

// mustCreate creates through the production entry or fails.
func mustCreate(t *testing.T, store *matjournal.Store, inputs matjournal.CreateInputs) (matjournal.JournalRef, matjournal.Journal) {
	t.Helper()
	ref, raw, err := store.Create(inputs)
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	loaded, _, err := store.Get(inputs.MaterializationID)
	if err != nil {
		t.Fatal(err)
	}
	_ = raw
	return ref, loaded
}

// convergePrepared drives a composite journal to the prepared phase
// through the production entries: provider prepared, both
// authorities prepared, bridge imported and opened, then staging to
// validating to prepared.
func convergePrepared(t *testing.T, store *matjournal.Store) {
	t.Helper()
	if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
		t.Fatalf("UpdateProvider error = %v", err)
	}
	providerRoot := testTxRoot + "/backups/codex_sessions"
	if _, err := store.UpdateAuthority(testMatID, "codex_sessions", matjournal.AuthorityState{
		RootPath: testRootProvider, CompletedSequences: []uint64{2}, RollbackRoot: &providerRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", matjournal.AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, RollbackRoot: &workspaceRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
		t.Fatalf("UpdateTaskBoard error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, openedBoard()); err != nil {
		t.Fatalf("UpdateTaskBoard error = %v", err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhasePrepared, matjournal.TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
}

// convergePreparedDirectProvider drives a direct closure with a
// provider branch to the prepared phase: provider prepared, both
// authorities prepared, then staging to validating to prepared. No
// bridge participates.
func convergePreparedDirectProvider(t *testing.T, store *matjournal.Store) {
	t.Helper()
	if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
		t.Fatalf("UpdateProvider error = %v", err)
	}
	providerRoot := testTxRoot + "/backups/codex_sessions"
	if _, err := store.UpdateAuthority(testMatID, "codex_sessions", matjournal.AuthorityState{
		RootPath: testRootProvider, CompletedSequences: []uint64{2}, RollbackRoot: &providerRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", matjournal.AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, RollbackRoot: &workspaceRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhasePrepared, matjournal.TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
}

// testLease returns the winning-lease observation.
func testLease() matjournal.LeaseIdentity {
	return matjournal.LeaseIdentity{Epoch: 4, ID: testLeaseID}
}

// baseRecoveryInput returns the consistent probe closure: prepared
// provider status with agreement, no bridge effect, no marker, a
// healthy host, and the unchanged winning lease.
func baseRecoveryInput() matjournal.RecoveryInput {
	lease := testLease()
	return matjournal.RecoveryInput{
		Boundary:    matjournal.BoundaryAfterPrepare,
		Path:        matjournal.PathOwnerResume,
		Provider:    matjournal.ProviderProbe{State: matjournal.ProviderPrepared, IDsMatch: true, PlanMatches: true, TokenMatches: true},
		Bridge:      matjournal.BridgeProbe{State: matjournal.BoardNotStarted},
		Marker:      matjournal.MarkerAbsent,
		Host:        matjournal.HostOK,
		LeaseKnown:  true,
		LeaseBefore: lease,
		LeaseAfter:  lease,
	}
}

// mustOutcome asserts the exact outcome and the never-fourth-outcome
// rule.
func mustOutcome(t *testing.T, got, want matjournal.Outcome, what string) {
	t.Helper()
	switch got {
	case matjournal.OutcomeSafeRetry, matjournal.OutcomeExplicitRollback, matjournal.OutcomeRecoverableParked:
	default:
		t.Fatalf("%s outcome = %q, want one of the three Section 13.13 outcomes", what, got)
	}
	if got != want {
		t.Fatalf("%s outcome = %q, want %q", what, got, want)
	}
}

// mustJournalEvidenceComplete asserts the Section 13.13 evidence
// record names every required fact: boundary ID, path, pre/post
// durable facts, external effect and status probe, winning lease
// before/after, selected outcome, and the satisfying evidence. The
// CR-MAT-01 record carries no operation IDs or phases because no
// journal exists; every other record must name them.
func mustJournalEvidenceComplete(t *testing.T, evidence matjournal.Evidence, what string) {
	t.Helper()
	if evidence.Boundary == "" || evidence.Path == "" {
		t.Fatalf("%s evidence lacks boundary or path: %+v", what, evidence)
	}
	if evidence.Boundary != matjournal.BoundaryBeforeCreate && len(evidence.OperationIDs) == 0 {
		t.Fatalf("%s evidence lacks operation IDs: %+v", what, evidence)
	}
	if evidence.Boundary != matjournal.BoundaryBeforeCreate && (evidence.PhaseBefore == "" || evidence.PhaseAfter == "") {
		t.Fatalf("%s evidence lacks pre/post durable facts: %+v", what, evidence)
	}
	if evidence.Reason == "" || evidence.Remediation == "" || evidence.DecidedAt == "" {
		t.Fatalf("%s evidence lacks reason, remediation, or decided_at: %+v", what, evidence)
	}
	if evidence.LeaseBefore.Epoch == 0 || evidence.LeaseAfter.Epoch == 0 {
		t.Fatalf("%s evidence lacks the winning lease before/after: %+v", what, evidence)
	}
	switch evidence.Selected {
	case matjournal.OutcomeSafeRetry, matjournal.OutcomeExplicitRollback, matjournal.OutcomeRecoverableParked:
	default:
		t.Fatalf("%s evidence selects %q, want one of the three outcomes", what, evidence.Selected)
	}
}

// mustCrashFault fails unless err is the injected fault for the
// armed point.
func mustCrashFault(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want the injected crash fault", what)
	}
}

// mustFailure builds one Structured Error through the axerror owner.
func mustFailure(t *testing.T, code axerror.Code, message string) *axerror.Error {
	t.Helper()
	operation, err := scalar.ParseUUIDv7(testPrepareOp)
	if err != nil {
		t.Fatal(err)
	}
	failure, err := axerror.New(axerror.Spec{
		Version:   axerror.Version120,
		Code:      code,
		Message:   message,
		Retryable: false,
		IDs:       axerror.NoIDs().WithOperation(operation),
		Details:   axerror.Details{},
	})
	if err != nil {
		t.Fatalf("axerror.New(%s) error = %v", code, err)
	}
	return failure
}

// ConformanceRecord is the machine-readable Section 13.13
// conformance record for one executed boundary row: boundary ID,
// path, operation IDs, pre/post durable facts, external effect and
// status probe, winning lease and native identity before/after,
// selected outcome, and the satisfying evidence. Journal rows embed
// the evaluator's evidence; checkpoint rows carry the retry
// convergence proof (capture has no evaluator: recovery is the
// idempotent retry).
type ConformanceRecord struct {
	Boundary       string            `json:"boundary"`
	RegistryPath   string            `json:"path"`
	EvaluatedVia   string            `json:"evaluated_via"`
	RecoverPath    string            `json:"recover_path"`
	Closure        string            `json:"closure"`
	Variant        string            `json:"variant"`
	Driver         string            `json:"driver"`
	OperationIDs   map[string]string `json:"operation_ids"`
	PhaseBefore    string            `json:"phase_before"`
	PhaseAfter     string            `json:"phase_after"`
	ReceiptPresent bool              `json:"receipt_present"`
	ExternalEffect string            `json:"external_effect"`
	StatusProbe    string            `json:"status_probe"`
	LeaseBefore    string            `json:"lease_before"`
	LeaseAfter     string            `json:"lease_after"`
	NativeBefore   string            `json:"native_before"`
	NativeAfter    string            `json:"native_after"`
	Selected       string            `json:"selected"`
	Reason         string            `json:"reason"`
	Remediation    string            `json:"remediation"`
}

// recordFromEvidence builds the conformance record for a journal row
// from the evaluator's evidence.
func recordFromEvidence(boundary, path, evaluatedVia, closure, variant, driver string, evidence matjournal.Evidence) ConformanceRecord {
	operationIDs := map[string]string{}
	for key, value := range evidence.OperationIDs {
		operationIDs[key] = value
	}
	return ConformanceRecord{
		Boundary:       boundary,
		RegistryPath:   path,
		EvaluatedVia:   evaluatedVia,
		RecoverPath:    evidence.Path,
		Closure:        closure,
		Variant:        variant,
		Driver:         driver,
		OperationIDs:   operationIDs,
		PhaseBefore:    evidence.PhaseBefore,
		PhaseAfter:     evidence.PhaseAfter,
		ReceiptPresent: evidence.ReceiptPresent,
		ExternalEffect: evidence.ExternalEffect,
		StatusProbe:    evidence.StatusProbe,
		LeaseBefore:    leaseString(evidence.LeaseBefore),
		LeaseAfter:     leaseString(evidence.LeaseAfter),
		NativeBefore:   evidence.NativeBefore,
		NativeAfter:    evidence.NativeAfter,
		Selected:       string(evidence.Selected),
		Reason:         evidence.Reason,
		Remediation:    evidence.Remediation,
	}
}

func leaseString(lease matjournal.LeaseIdentity) string {
	raw, err := json.Marshal(lease)
	if err != nil {
		return ""
	}
	return string(raw)
}

// conformanceDir returns the directory conformance records are
// written to: CRASHGATE_CONFORMANCE_DIR when set (the evidence
// collection run), otherwise a per-test temp directory. A relative
// directory resolves against the module root (tests run with the
// package directory as the working directory), so
// CRASHGATE_CONFORMANCE_DIR=.temp/... lands under the repository
// .temp tree as the evidence contract requires.
func conformanceDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("CRASHGATE_CONFORMANCE_DIR")
	if dir == "" {
		return t.TempDir()
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(moduleRoot(t), dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// moduleRoot walks up from the test working directory to the
// directory carrying go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("go.mod not found above the test working directory")
		}
		current = parent
	}
}

// writeRecord writes one machine-readable conformance record. The
// file name carries boundary, path, and variant so the evidence
// tarball enumerates rows without parsing bodies.
func writeRecord(t *testing.T, dir, name string, record ConformanceRecord) {
	t.Helper()
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dir, name+".json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustRecordComplete asserts the conformance record names every
// Section 13.13 fact: boundary ID, path, operation IDs (except the
// no-journal row), pre/post durable facts, external effect and
// status probe, winning lease and native identity before/after,
// selected outcome, and the satisfying evidence.
func mustRecordComplete(t *testing.T, record ConformanceRecord, what string) {
	t.Helper()
	if record.Boundary == "" || record.RegistryPath == "" || record.Driver == "" || record.Variant == "" {
		t.Fatalf("%s record lacks boundary, path, driver, or variant: %+v", what, record)
	}
	if record.EvaluatedVia == "" || record.Closure == "" {
		t.Fatalf("%s record lacks evaluated_via or closure: %+v", what, record)
	}
	if record.Selected == "" || record.Reason == "" || record.Remediation == "" {
		t.Fatalf("%s record lacks outcome, reason, or remediation: %+v", what, record)
	}
	switch record.Selected {
	case string(matjournal.OutcomeSafeRetry), string(matjournal.OutcomeExplicitRollback), string(matjournal.OutcomeRecoverableParked):
	default:
		t.Fatalf("%s record selects %q, want one of the three outcomes", what, record.Selected)
	}
	if record.LeaseBefore == "" || record.LeaseAfter == "" {
		t.Fatalf("%s record lacks the winning lease before/after: %+v", what, record)
	}
}
