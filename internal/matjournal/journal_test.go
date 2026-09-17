package matjournal

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Fixed identities shared by every fixture. UUIDv7 grammar carries
// the version nibble; the lease token is UUIDv4.
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
)

// Fixed digests. Well-formed content hashes the journal never
// resolves: resolution belongs to the manifest owners.
const (
	testPlanID       = "sha256:64644a5ad573d36c0c13f44f56ef25ab93cff33001ff2a3371b082603910f2dd"
	testCheckpointID = "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656"
	testBundleID     = "sha256:0af7b44e7063375a0f06e546fd820438c72607f191c14be78d35c8ffa109844f"
	testPriorID      = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	testBlobA        = "sha256:0e442b07e3772e8f5622478242ddf5f9f197bbd6a0402cd71471db4081abb291"
	testBlobB        = "sha256:444e0fffbd825e9610ff5b199485707a0c895339ae80c15cc8a8aee41b106fda"
	testGroupRecord  = "sha256:3b366ca989681c63323c5de6db28198796aa913947ad3cd9456fc6dcee62b743"
	testWsManifest   = "sha256:a98ca90522b4de30e4aaaf9bf50529d09e15a817ffa67f94552fb313d1a1ad2e"
)

// Fixed tokens: the spec provider/open vectors plus a derived import
// token. Each decodes to at least 32 bytes of canonical base64url.
const (
	testProviderToken = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5QUI"
	testOpenToken     = "b3duZXItb25seS1vcGVuLXRva2VuLTAxMjM0NTY3ODlhYmNkZWY"
	testImportToken   = "aW1wb3J0LW9ubHktYnJpZGdlLXRva2VuLTAxMjM0NTY3ODk"
)

const (
	testRootWorkspace = "/srv/relux"
	testRootProvider  = "/home/ivan/.codex/sessions"
	testTxRoot        = "/home/ivan/.local/state/ax/provider-transactions/codex/0198f4c8-f5c0-76dd-9677-1234567890ab"
	testStagingRoot   = "/home/ivan/.local/state/ax/staging/0198f4c8-c290-73aa-9374-1234567890ab"
)

// fixedClock pins store time to one instant.
func fixedClock() time.Time {
	return time.Date(2026, time.August, 19, 4, 12, 0, 0, time.UTC)
}

// openTestStore binds an empty journal store in a temp dir with a
// fixed clock.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.Now = fixedClock
	return store
}

// testRequestBody returns the complete prepare request body carrying
// both caller-stable IDs.
func testRequestBody() []byte {
	return []byte(`{"materialization_id":"` + testMatID + `","operation_id":"` + testPrepareOp + `","plan_id":"` + testPlanID + `"}`)
}

// testInputs returns the workspace prepare closure over one workspace
// authority with no provider or bridge.
func testInputs() CreateInputs {
	return CreateInputs{
		MaterializationID:  testMatID,
		PrepareOperationID: testPrepareOp,
		RequestBody:        testRequestBody(),
		TransferID:         testTransfer,
		PlanID:             testPlanID,
		SourceCheckpointID: testCheckpointID,
		ManagedReplicaID:   testReplicaID,
		Plan: []PlanAuthority{
			{ID: "workspace_relux", Kind: PlanKindWorkspace, Platform: "linux", RootPath: testRootWorkspace},
		},
		HostPlatform:              "linux",
		ExpectedPriorCheckpointID: testPriorID,
		Extensions:                map[string]string{},
	}
}

// testInputsComposite returns the composite prepare closure: a
// workspace authority, a provider-store authority, and a task-board
// bridge in ownership_transfer mode.
func testInputsComposite() CreateInputs {
	inputs := testInputs()
	inputs.Plan = []PlanAuthority{
		{ID: "codex_sessions", Kind: PlanKindProviderStore, Platform: "linux", RootPath: testRootProvider},
		{ID: "workspace_relux", Kind: PlanKindWorkspace, Platform: "linux", RootPath: testRootWorkspace},
	}
	inputs.BundleID = testBundleID
	inputs.ActivationMode = ModeOwnershipTransfer
	inputs.ImportOperationID = testImportOp
	inputs.OpenOperationID = testOpenOp
	inputs.AdoptOperationID = testAdoptOp
	inputs.ResumeOperationID = testResumeOp
	return inputs
}

// testInputsBoardOnly returns the task-board-only closure: one staging
// authority, no workspace bytes, dormant activation.
func testInputsBoardOnly() CreateInputs {
	inputs := CreateInputs{
		MaterializationID:  testMatID,
		PrepareOperationID: testPrepareOp,
		RequestBody:        testRequestBody(),
		PlanID:             testPlanID,
		SourceCheckpointID: testCheckpointID,
		Plan: []PlanAuthority{
			{ID: "board_staging", Kind: PlanKindTaskBoardStaging, Platform: "linux", RootPath: testStagingRoot},
		},
		HostPlatform:      "linux",
		BundleID:          testBundleID,
		ActivationMode:    ModeDormantReplica,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		Extensions:        map[string]string{},
	}
	return inputs
}

// mustCreate creates through the production entry or fails.
func mustCreate(t *testing.T, store *Store, inputs CreateInputs) (JournalRef, Journal) {
	t.Helper()
	ref, raw, err := store.Create(inputs)
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	journal, err := decodeForTest(t, store, raw)
	if err != nil {
		t.Fatalf("decode created journal: %v", err)
	}
	return ref, journal
}

// decodeForTest decodes journal bytes with the store's installed plan
// view, proving the read path agrees with the write path.
func decodeForTest(t *testing.T, store *Store, raw []byte) (Journal, error) {
	t.Helper()
	var journal Journal
	if err := json.Unmarshal(raw, &journal); err != nil {
		return Journal{}, err
	}
	loaded, _, err := store.Get(journal.MaterializationID)
	if err != nil {
		return Journal{}, err
	}
	return loaded, nil
}

// mustInvalid asserts the invalid-journal refusal class.
func mustInvalid(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want invalid journal", what)
	}
	if !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("%s error = %v, want ErrInvalidJournal", what, err)
	}
}

// mustConflict asserts the idempotency-conflict class without the
// invalid-closure class.
func mustConflict(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want journal conflict", what)
	}
	if !errors.Is(err, ErrJournalConflict) {
		t.Fatalf("%s error = %v, want ErrJournalConflict", what, err)
	}
	if errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("%s error = %v, want the conflict class, not the invalid class", what, err)
	}
}

// str returns a string pointer.
func str(value string) *string { return &value }

// testProvider returns the prepared provider transaction bound to the
// test journal.
func testProvider() ProviderTransaction {
	return ProviderTransaction{
		OperationID:   testProviderOp,
		TransactionID: testTxID,
		State:         ProviderPrepared,
		RollbackToken: str(testProviderToken),
		Authority: TransactionAuthority{
			AuthorityID:                        txAuthorityID,
			Kind:                               txAuthorityKind,
			RootPath:                           testTxRoot,
			Layout:                             txAuthorityLayout,
			Access:                             txAuthorityAccess,
			MaterializationID:                  testMatID,
			ProviderID:                         "codex",
			TransactionID:                      testTxID,
			PlanID:                             testPlanID,
			SameFilesystemProviderAuthorityIDs: []string{"codex_sessions"},
		},
		LastStatusAt: "2026-08-19T04:12:10.000Z",
	}
}

// testBoardImported returns the imported bridge transaction.
func testBoardImported() TaskBoardTransaction {
	return TaskBoardTransaction{
		BundleID:          testBundleID,
		ActivationMode:    ModeOwnershipTransfer,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		State:             BoardImported,
		ImportToken:       str(testImportToken),
		StagedManagerRef:  str("tbm:staged:0198f4c8-7a10-7b22-8b3c-2234567890ab"),
		ImportExpiresAt:   str("2026-08-19T04:22:10.000Z"),
		LastBridgeState:   str("stopped"),
		LastStatusAt:      str("2026-08-19T04:12:10.000Z"),
		CleanupState:      "not_started",
	}
}

// testBoardOpened returns the opened bridge transaction matching the
// MJ-TASK-BOARD-OPENED-POS fixture members.
func testBoardOpened() TaskBoardTransaction {
	return TaskBoardTransaction{
		BundleID:          testBundleID,
		ActivationMode:    ModeOwnershipTransfer,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		State:             BoardOpened,
		OpenToken:         str(testOpenToken),
		DormantManagerRef: str("tbm:dormant:0198f4c8-7a10-7b22-8b3c-2234567890ab"),
		OpenExpiresAt:     str("2026-08-19T04:22:10.000Z"),
		LastBridgeState:   str("dormant"),
		LastStatusAt:      str("2026-08-19T04:12:10.000Z"),
		CleanupState:      "not_started",
	}
}

// testBoardAdopted returns the adopted bridge transaction.
func testBoardAdopted() TaskBoardTransaction {
	return TaskBoardTransaction{
		BundleID:          testBundleID,
		ActivationMode:    ModeOwnershipTransfer,
		ImportOperationID: testImportOp,
		OpenOperationID:   testOpenOp,
		AdoptOperationID:  testAdoptOp,
		ResumeOperationID: testResumeOp,
		State:             BoardAdopted,
		ManagerSessionRef: str("tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab"),
		AxBinding:         &AxBinding{AxSessionID: testSessionID, LeaseEpoch: 4, LeaseID: testLeaseID},
		LastBridgeState:   str("stopped"),
		LastStatusAt:      str("2026-08-19T04:12:10.000Z"),
		CleanupState:      "retained_active",
	}
}

// testBoardResumed returns the resumed bridge transaction.
func testBoardResumed() TaskBoardTransaction {
	board := testBoardAdopted()
	board.State = BoardResumed
	board.LastBridgeState = str("running")
	return board
}

// mustFailure builds one Structured Error through the axerror owner.
func mustFailure(t *testing.T, code axerror.Code, message string) *axerror.Error {
	t.Helper()
	operation, err := scalar.ParseUUIDv7(testPrepareOp)
	if err != nil {
		t.Fatal(err)
	}
	failure, err := axerror.New(axerror.Spec{
		Version:   lastErrorVersion,
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

// mustFrame marshals one Structured Error to its wire bytes.
func mustFrame(t *testing.T, failure *axerror.Error) json.RawMessage {
	t.Helper()
	framed, err := failure.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return framed
}

func TestTokenFixturesAreWellFormed(t *testing.T) {
	for name, token := range map[string]string{
		"provider": testProviderToken,
		"open":     testOpenToken,
		"import":   testImportToken,
	} {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			t.Fatalf("%s token decodes: %v", name, err)
		}
		if len(raw) < 32 || len(raw) > 512 {
			t.Fatalf("%s token decodes to %d bytes", name, len(raw))
		}
		if base64.RawURLEncoding.EncodeToString(raw) != token {
			t.Fatalf("%s token is not a canonical fixpoint", name)
		}
	}
}

func TestCreatePersistsJournalAndReceipt(t *testing.T) {
	store := openTestStore(t)
	ref, journal := mustCreate(t, store, testInputsComposite())
	if ref.MaterializationID != testMatID || ref.PrepareOperationID != testPrepareOp || ref.Phase != PhaseStaging {
		t.Fatalf("ref = %+v", ref)
	}
	if journal.Phase != PhaseStaging {
		t.Fatalf("phase = %q", journal.Phase)
	}
	// The closed member set equals the normative 22-member shape.
	_, raw, err := store.Create(testInputsComposite())
	if err != nil {
		t.Fatalf("replay Create error = %v", err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		t.Fatal(err)
	}
	if len(members) != len(journalMembers) {
		t.Fatalf("journal members = %d, want %d", len(members), len(journalMembers))
	}
	for _, name := range journalMembers {
		if _, ok := members[name]; !ok {
			t.Fatalf("journal member %q is absent", name)
		}
	}
	// One staging entry per plan authority; the bridge key set is
	// bound in not_started with null capabilities.
	if len(journal.AuthorityStates) != 2 {
		t.Fatalf("authority states = %d, want 2", len(journal.AuthorityStates))
	}
	for id, state := range journal.AuthorityStates {
		if state.State != AuthorityStaging {
			t.Fatalf("authority %q state = %q", id, state.State)
		}
	}
	if journal.TaskBoard == nil || journal.TaskBoard.State != BoardNotStarted {
		t.Fatalf("task-board state = %+v", journal.TaskBoard)
	}
	if journal.TaskBoard.ImportOperationID != testImportOp || journal.TaskBoard.ResumeOperationID != testResumeOp {
		t.Fatalf("bridge keys = %+v", journal.TaskBoard)
	}
	if journal.Provider != nil {
		t.Fatalf("provider = %+v, want null before participation", journal.Provider)
	}
	// The receipt binds the operation to the request digest, and Get
	// returns byte-identical bytes.
	if journal.PrepareRequestDigest == "" {
		t.Fatalf("prepare_request_digest is empty")
	}
	loaded, stored, err := store.Get(testMatID)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if loaded.Phase != PhaseStaging || string(stored) != string(raw) {
		t.Fatalf("Get bytes differ from created bytes")
	}
}

func TestSpecExampleShapeDrivesThroughProduction(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	// The provider prepares with its rollback token durable.
	provider := testProvider()
	if _, err := store.UpdateProvider(testMatID, provider); err != nil {
		t.Fatalf("UpdateProvider error = %v", err)
	}
	// Both authorities prepare with their rollback roots: the
	// provider authority references the plugin backup, the workspace
	// authority its host staging root.
	providerRoot := testTxRoot + "/backups/codex_sessions"
	if _, err := store.UpdateAuthority(testMatID, "codex_sessions", AuthorityState{
		RootPath:            testRootProvider,
		CompletedSequences:  []uint64{2},
		ObservedPriorDigest: nil,
		RollbackRoot:        &providerRoot,
		State:               AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority(codex_sessions) error = %v", err)
	}
	workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
		RootPath:            testRootWorkspace,
		CompletedSequences:  []uint64{1},
		ObservedPriorDigest: str("sha256:3333333333333333333333333333333333333333333333333333333333333333"),
		RollbackRoot:        &workspaceRoot,
		State:               AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority(workspace_relux) error = %v", err)
	}
	// The bridge imports and opens; commit then enters prepared.
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
		t.Fatalf("UpdateTaskBoard(imported) error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
		t.Fatalf("UpdateTaskBoard(opened) error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
		t.Fatalf("Transition(validating) error = %v", err)
	}
	prepared, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{})
	if err != nil {
		t.Fatalf("Transition(prepared) error = %v", err)
	}
	// The prepared journal matches the normative example members:
	// phase, provider prepared token, authority states, opened
	// bridge with its open capability.
	if prepared.Phase != PhasePrepared {
		t.Fatalf("phase = %q", prepared.Phase)
	}
	if prepared.Provider.State != ProviderPrepared || prepared.Provider.RollbackToken == nil {
		t.Fatalf("provider = %+v", prepared.Provider)
	}
	if prepared.TaskBoard.State != BoardOpened || prepared.TaskBoard.OpenToken == nil {
		t.Fatalf("task-board = %+v", prepared.TaskBoard)
	}
	if prepared.AuthorityStates["codex_sessions"].State != AuthorityPrepared ||
		prepared.AuthorityStates["workspace_relux"].State != AuthorityPrepared {
		t.Fatalf("authorities = %+v", prepared.AuthorityStates)
	}
}

// specOpenedFixture is the verbatim MJ-TASK-BOARD-OPENED-POS fixture.
const specOpenedFixture = `{
  "bundle_id": "sha256:0af7b44e7063375a0f06e546fd820438c72607f191c14be78d35c8ffa109844f",
  "activation_mode": "ownership_transfer",
  "import_operation_id": "0198f4c8-0a10-71aa-8111-1234567890ab",
  "open_operation_id": "0198f4c8-0a20-72bb-8222-1234567890ab",
  "adopt_operation_id": "0198f4c8-0a30-73cc-8333-1234567890ab",
  "resume_operation_id": "0198f4c8-0a40-74dd-8444-1234567890ab",
  "state": "opened",
  "import_token": null,
  "staged_manager_ref": null,
  "import_expires_at": null,
  "open_token": "b3duZXItb25seS1vcGVuLXRva2VuLTAxMjM0NTY3ODlhYmNkZWY",
  "dormant_manager_ref": "tbm:dormant:0198f4c8-7a10-7b22-8b3c-2234567890ab",
  "open_expires_at": "2026-08-19T04:22:10.000Z",
  "manager_session_ref": null,
  "ax_binding": null,
  "last_bridge_state": "dormant",
  "last_status_at": "2026-08-19T04:12:10.000Z",
  "cleanup_state": "not_started",
  "cleanup_after": null
}`

func TestOpenedFixtureMatchesProduction(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
		t.Fatalf("UpdateTaskBoard(imported) error = %v", err)
	}
	updated, err := store.UpdateTaskBoard(testMatID, testBoardOpened())
	if err != nil {
		t.Fatalf("UpdateTaskBoard(opened) error = %v", err)
	}
	framed, err := json.Marshal(updated.TaskBoard)
	if err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(framed, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(specOpenedFixture), &want); err != nil {
		t.Fatal(err)
	}
	wantBytes, _ := json.Marshal(want)
	gotBytes, _ := json.Marshal(got)
	if string(gotBytes) != string(wantBytes) {
		t.Fatalf("opened board = %s, want %s", gotBytes, wantBytes)
	}
}

func TestRecordProgressMergesPerBlob(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	// MJ-MULTIBLOB-POS: chunk zero for two blobs is two independent
	// facts, recorded separately and merged by union.
	if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, nil); err != nil {
		t.Fatalf("RecordProgress(A0) error = %v", err)
	}
	updated, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobB: {0}}, []string{testBlobA})
	if err != nil {
		t.Fatalf("RecordProgress(B0) error = %v", err)
	}
	if len(updated.CompletedBlobChunks[testBlobA]) != 1 || len(updated.CompletedBlobChunks[testBlobB]) != 1 {
		t.Fatalf("chunks = %+v", updated.CompletedBlobChunks)
	}
	if len(updated.VerifiedBlobIDs) != 1 || updated.VerifiedBlobIDs[0] != testBlobA {
		t.Fatalf("verified = %+v", updated.VerifiedBlobIDs)
	}
	// A union merge keeps both sets sorted and unique.
	updated, err = store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {2, 1, 1}}, []string{testBlobB, testBlobA})
	if err != nil {
		t.Fatalf("RecordProgress(merge) error = %v", err)
	}
	got := updated.CompletedBlobChunks[testBlobA]
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("blob A chunks = %+v", got)
	}
	if len(updated.VerifiedBlobIDs) != 2 {
		t.Fatalf("verified = %+v", updated.VerifiedBlobIDs)
	}
	// Progress closes once the journal leaves validation.
	if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {3}}, nil); err == nil {
		t.Fatalf("RecordProgress(prepared) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "RecordProgress(prepared)")
	}
}

func TestFullDormantWalkCommits(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsBoardOnly())
	// TB-TXN-PASSIVE: a passive replica stops at opened without
	// adopt or resume; the managed replica stays non-executable.
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImportedDormant()); err != nil {
		t.Fatalf("UpdateTaskBoard(imported) error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardOpenedDormant()); err != nil {
		t.Fatalf("UpdateTaskBoard(opened) error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	// Finalize destroys the open token copy and records
	// dormant_finalized with cleanup pending the consumed expiry.
	board := testBoardOpenedDormant()
	dormant := "dormant"
	finalized := TaskBoardTransaction{
		BundleID:          board.BundleID,
		ActivationMode:    board.ActivationMode,
		ImportOperationID: board.ImportOperationID,
		OpenOperationID:   board.OpenOperationID,
		AdoptOperationID:  board.AdoptOperationID,
		ResumeOperationID: board.ResumeOperationID,
		State:             BoardDormantFinalized,
		DormantManagerRef: board.DormantManagerRef,
		LastBridgeState:   &dormant,
		LastStatusAt:      board.LastStatusAt,
		CleanupState:      "pending_expiry",
		CleanupAfter:      board.OpenExpiresAt,
	}
	if _, err := store.UpdateTaskBoard(testMatID, finalized); err != nil {
		t.Fatalf("UpdateTaskBoard(dormant_finalized) error = %v", err)
	}
	stagingBackup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/board_staging"
	if _, err := store.UpdateAuthority(testMatID, "board_staging", AuthorityState{
		RootPath:            testStagingRoot,
		CompletedSequences:  []uint64{1},
		ObservedPriorDigest: nil,
		RollbackRoot:        &stagingBackup,
		State:               AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority(prepared) error = %v", err)
	}
	// Cleanup consumes the rollback root before the committed
	// terminal state.
	if _, err := store.UpdateAuthority(testMatID, "board_staging", AuthorityState{
		RootPath:            testStagingRoot,
		CompletedSequences:  []uint64{1},
		ObservedPriorDigest: nil,
		RollbackRoot:        nil,
		State:               AuthorityCommitted,
	}); err != nil {
		t.Fatalf("UpdateAuthority(committed) error = %v", err)
	}
	committed, err := store.Transition(testMatID, PhaseCommitted, TransitionOpts{})
	if err != nil {
		t.Fatalf("Transition(committed) error = %v", err)
	}
	if committed.Phase != PhaseCommitted || committed.DestinationMarkerID != nil {
		t.Fatalf("committed = %+v", committed.Phase)
	}
	if committed.TaskBoard.State != BoardDormantFinalized || committed.TaskBoard.OpenToken != nil {
		t.Fatalf("dormant token survives commit: %+v", committed.TaskBoard)
	}
}

// testBoardImportedDormant returns the imported bridge in dormant mode.
func testBoardImportedDormant() TaskBoardTransaction {
	board := testBoardImported()
	board.ActivationMode = ModeDormantReplica
	return board
}

// testBoardOpenedDormant returns the opened bridge in dormant mode.
func testBoardOpenedDormant() TaskBoardTransaction {
	board := testBoardOpened()
	board.ActivationMode = ModeDormantReplica
	return board
}

func TestLastErrorIsRedactedStructuredError(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
		t.Fatal(err)
	}
	updated, err := store.RecordError(testMatID, mustFailure(t, "staging_incomplete", "destination disk full: staging partial"))
	if err != nil {
		t.Fatalf("RecordError error = %v", err)
	}
	// The member decodes through the axerror owner at the bound
	// version, and carries no token material anywhere in its bytes.
	decoded, err := axerror.Decode(lastErrorVersion, updated.LastError)
	if err != nil {
		t.Fatalf("last_error decodes: %v", err)
	}
	if string(decoded.Code()) != "staging_incomplete" {
		t.Fatalf("code = %q", decoded.Code())
	}
	for _, token := range []string{testProviderToken, testOpenToken, testImportToken} {
		if strings.Contains(string(updated.LastError), token) {
			t.Fatalf("last_error leaks token material")
		}
	}
	// The failed phase requires and carries the explaining error.
	failed, err := store.Transition(testMatID, PhaseFailed, TransitionOpts{LastError: mustFrame(t, mustFailure(t, "materialization_failed", "validation failed terminally"))})
	if err != nil {
		t.Fatalf("Transition(failed) error = %v", err)
	}
	if len(failed.LastError) == 0 {
		t.Fatalf("failed journal carries no last_error")
	}
}

func TestMarkerExampleValidates(t *testing.T) {
	marker, err := ValidateMarker([]byte(specMarkerExample))
	if err != nil {
		t.Fatalf("ValidateMarker(spec example) error = %v", err)
	}
	// The pinned marker_id recomputes through the omit-self digest.
	if marker.MarkerID != "sha256:385c71c7a29a43615c9d35ffb7c93ae20cd9419bbca461627048de575cade94c" {
		t.Fatalf("marker_id = %q", marker.MarkerID)
	}
	if marker.Destination.LogicalRoot != "relux" || marker.Platform != "linux" {
		t.Fatalf("marker = %+v", marker)
	}
}

// specMarkerExample is the verbatim Section 10.6 normative marker
// fixture.
const specMarkerExample = `{
  "schema": "urn:ax:schema:materialization-journal",
  "schema_version": "2.0.0",
  "document_kind": "managed_replica_marker",
  "marker_id": "sha256:385c71c7a29a43615c9d35ffb7c93ae20cd9419bbca461627048de575cade94c",
  "managed_replica_id": "0198f4c8-8e50-7f66-8f70-2234567890ab",
  "host_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab",
  "platform": "linux",
  "workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
  "primary_session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "source_checkpoint_id": "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656",
  "workspace_group_record_id": "sha256:3b366ca989681c63323c5de6db28198796aa913947ad3cd9456fc6dcee62b743",
  "workspace_manifest_id": "sha256:a98ca90522b4de30e4aaaf9bf50529d09e15a817ffa67f94552fb313d1a1ad2e",
  "plan_id": "sha256:64644a5ad573d36c0c13f44f56ef25ab93cff33001ff2a3371b082603910f2dd",
  "materialization_id": "0198f4c8-c290-73aa-9374-1234567890ab",
  "destination": {
    "logical_root": "relux",
    "workspace_relative_path": "payments",
    "resolved_path_fingerprint": "sha256:244d99a3e794e9ea89b4e30429adfd6bc142c069cf0635576e010031f5b85ded"
  },
  "predecessor_marker_id": null,
  "committed_at": "2026-08-19T04:12:30.000Z",
  "extensions": {}
}`

// identifyMarker stamps the omit-self digest over a marker candidate.
func identifyMarker(t *testing.T, value map[string]any) []byte {
	t.Helper()
	value["marker_id"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	framed, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var logical map[string]any
	if err := json.Unmarshal(framed, &logical); err != nil {
		t.Fatal(err)
	}
	delete(logical, "marker_id")
	serialized, err := json.Marshal(logical)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicaljson.Canonicalize(serialized)
	if err != nil {
		t.Fatal(err)
	}
	value["marker_id"] = scalar.SHA256Digest(canonical).String()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// testMarker returns valid marker bytes bound to the test journal.
func testMarker(t *testing.T) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(specMarkerExample), &value); err != nil {
		t.Fatal(err)
	}
	value["managed_replica_id"] = testReplicaID
	value["materialization_id"] = testMatID
	value["plan_id"] = testPlanID
	value["source_checkpoint_id"] = testCheckpointID
	return identifyMarker(t, value)
}

func TestClassifyDestination(t *testing.T) {
	marker := testMarker(t)
	cases := []struct {
		name  string
		facts DestinationFacts
		want  string
	}{
		{"absent", DestinationFacts{TargetAbsent: true}, DestinationAbsent},
		{"empty", DestinationFacts{TargetEmpty: true}, DestinationEmpty},
		{"unmanaged", DestinationFacts{}, DestinationUnmanagedNonempty},
		{"unchanged", DestinationFacts{Marker: marker, MarkerPresent: true, FingerprintMatches: true, ManifestMatches: true}, DestinationManagedUnchanged},
		{"divergent", DestinationFacts{Marker: marker, MarkerPresent: true, FingerprintMatches: true}, DestinationManagedDivergent},
		{"path_mismatch", DestinationFacts{Marker: marker, MarkerPresent: true, ManifestMatches: true}, DestinationIntegrityFailure},
		{"torn_marker", DestinationFacts{Marker: []byte(`{"schema":"x"}`), MarkerPresent: true, FingerprintMatches: true, ManifestMatches: true}, DestinationIntegrityFailure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ClassifyDestination(tc.facts)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("class = %q, want %q", got, tc.want)
			}
		})
	}
}
