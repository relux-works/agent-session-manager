package matjournal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

func TestCreateRefusesMalformedClosure(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*CreateInputs)
	}{
		{"bad_materialization", func(i *CreateInputs) { i.MaterializationID = "not-a-uuid" }},
		{"bad_prepare", func(i *CreateInputs) { i.PrepareOperationID = "not-a-uuid" }},
		{"body_not_json", func(i *CreateInputs) { i.RequestBody = []byte("{") }},
		{"body_not_object", func(i *CreateInputs) { i.RequestBody = []byte(`[1,2]`) }},
		{"body_duplicate_key", func(i *CreateInputs) { i.RequestBody = []byte(`{"a":1,"a":2}`) }},
		{"bad_transfer", func(i *CreateInputs) { i.TransferID = "sha256:zzz" }},
		{"bad_plan", func(i *CreateInputs) { i.PlanID = "not-a-digest" }},
		{"bad_checkpoint", func(i *CreateInputs) { i.SourceCheckpointID = "not-a-digest" }},
		{"bad_replica", func(i *CreateInputs) { i.ManagedReplicaID = "not-a-uuid" }},
		{"bad_host_platform", func(i *CreateInputs) { i.HostPlatform = "plan9" }},
		{"bad_prior", func(i *CreateInputs) { i.ExpectedPriorCheckpointID = "not-a-digest" }},
		{"bad_extension_key", func(i *CreateInputs) { i.Extensions = map[string]string{"pid": "4242"} }},
		{"duplicate_authority", func(i *CreateInputs) {
			i.Plan = append(i.Plan, i.Plan[0])
		}},
		{"bad_authority_id", func(i *CreateInputs) { i.Plan[0].ID = "Relux!" }},
		{"bad_authority_kind", func(i *CreateInputs) { i.Plan[0].Kind = "replica" }},
		{"bad_authority_platform", func(i *CreateInputs) { i.Plan[0].Platform = "plan9" }},
		{"relative_authority_root", func(i *CreateInputs) { i.Plan[0].RootPath = "srv/relux" }},
		{"partial_bridge_bundle_only", func(i *CreateInputs) { i.BundleID = testBundleID }},
		{"partial_bridge_mode_only", func(i *CreateInputs) { i.ActivationMode = ModeFork }},
		{"partial_bridge_ids_only", func(i *CreateInputs) { i.ImportOperationID = testImportOp }},
		{"bad_bridge_bundle", func(i *CreateInputs) {
			*i = testInputsComposite()
			i.BundleID = "not-a-digest"
		}},
		{"bad_bridge_mode", func(i *CreateInputs) {
			*i = testInputsComposite()
			i.ActivationMode = "passive"
		}},
		{"bad_bridge_import", func(i *CreateInputs) {
			*i = testInputsComposite()
			i.ImportOperationID = "not-a-uuid"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := openTestStore(t)
			inputs := testInputs()
			tc.mutate(&inputs)
			_, _, err := store.Create(inputs)
			mustInvalid(t, err, "Create(member)")
		})
	}
	t.Run("too_many_extensions", func(t *testing.T) {
		store := openTestStore(t)
		inputs := testInputs()
		inputs.Extensions = map[string]string{}
		for i := 0; i < 65; i++ {
			inputs.Extensions["works.relux.ax.test."+string(rune('a'+i/26))+string(rune('a'+i%26))] = "x"
		}
		_, _, err := store.Create(inputs)
		mustInvalid(t, err, "Create(65 extensions)")
	})
}

func TestCreateReplayAndConflict(t *testing.T) {
	store := openTestStore(t)
	ref, _ := mustCreate(t, store, testInputsComposite())
	// The byte-identical retry replays the recorded journal without
	// writing: the stored bytes are unchanged.
	again, raw, err := store.Create(testInputsComposite())
	if err != nil {
		t.Fatalf("replay Create error = %v", err)
	}
	if again != ref {
		t.Fatalf("replay ref = %+v, want %+v", again, ref)
	}
	loaded, stored, err := store.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != PhaseStaging || string(stored) != string(raw) {
		t.Fatalf("replay diverged stored bytes")
	}
	// The same IDs with a moved body refuse idempotency_mismatch
	// and write nothing.
	moved := testInputsComposite()
	moved.RequestBody = []byte(`{"materialization_id":"` + testMatID + `","operation_id":"` + testPrepareOp + `","plan_id":"` + testPriorID + `"}`)
	_, _, err = store.Create(moved)
	mustConflict(t, err, "Create(moved body)")
	if !strings.Contains(err.Error(), "idempotency_mismatch") {
		t.Fatalf("error = %v, want idempotency_mismatch named", err)
	}
	// The same materialization under a different prepare operation
	// is a second prepare for one materialization, not a replay.
	// The refusal names the first prepare: a backstop conflict with
	// another message fails here.
	second := testInputsComposite()
	second.PrepareOperationID = testAdoptOp
	_, _, err = store.Create(second)
	mustConflict(t, err, "Create(second prepare)")
	if !strings.Contains(err.Error(), "prepared under operation") {
		t.Fatalf("Create(second prepare) = %v, want the first-prepare refusal", err)
	}
	// The journal still records the first prepare only.
	current, _, err := store.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if current.PrepareOperationID != testPrepareOp {
		t.Fatalf("prepare operation drifted to %q", current.PrepareOperationID)
	}
}

func TestTransitionRefusesIllegalEdges(t *testing.T) {
	// Every phase pair outside the table refuses through the
	// production entry, including self-transitions from terminal
	// phases.
	illegal := map[string][]string{
		PhaseStaging:     {PhaseStaging, PhasePrepared, PhaseCommitting, PhaseRolledBack, PhaseCommitted},
		PhaseValidating:  {PhaseStaging, PhaseValidating, PhaseCommitting, PhaseRolledBack, PhaseCommitted},
		PhasePrepared:    {PhaseStaging, PhaseValidating, PhasePrepared, PhaseRolledBack, PhaseCommitted},
		PhaseCommitting:  {PhaseStaging, PhaseValidating, PhasePrepared, PhaseCommitting, PhaseRolledBack},
		PhaseRollingBack: {PhaseStaging, PhaseValidating, PhasePrepared, PhaseCommitting, PhaseRollingBack, PhaseCommitted},
		PhaseRolledBack:  {PhaseStaging, PhaseValidating, PhasePrepared, PhaseCommitting, PhaseRollingBack, PhaseRolledBack, PhaseCommitted, PhaseFailed},
		PhaseCommitted:   {PhaseStaging, PhaseValidating, PhasePrepared, PhaseCommitting, PhaseRollingBack, PhaseRolledBack, PhaseCommitted, PhaseFailed},
		PhaseFailed:      {PhaseStaging, PhaseValidating, PhasePrepared, PhaseCommitting, PhaseRollingBack, PhaseRolledBack, PhaseCommitted, PhaseFailed},
	}
	for from, targets := range illegal {
		t.Run("from_"+from, func(t *testing.T) {
			for _, to := range targets {
				store := openTestStore(t)
				setupPhase(t, store, from)
				_, err := store.Transition(testMatID, to, TransitionOpts{})
				if err == nil {
					t.Fatalf("Transition(%s to %s) error = nil, want refusal", from, to)
				}
				mustInvalid(t, err, "Transition(illegal)")
				// The refusal names its exact gate: the phase
				// table for live phases, the terminal freeze
				// for terminal ones. A backstop refusal with
				// another message fails here.
				switch from {
				case PhaseCommitted, PhaseRolledBack, PhaseFailed:
					if !strings.Contains(err.Error(), "terminal in") {
						t.Fatalf("Transition(%s to %s) = %v, want the terminal freeze", from, to, err)
					}
				default:
					if !strings.Contains(err.Error(), "outside the phase table") {
						t.Fatalf("Transition(%s to %s) = %v, want the phase-table refusal", from, to, err)
					}
				}
			}
		})
	}
	// Unknown targets refuse as unknown enum members.
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	_, err := store.Transition(testMatID, "archived", TransitionOpts{})
	mustInvalid(t, err, "Transition(unknown)")
}

// setupPhase drives one journal to the named phase through the
// production entries.
func setupPhase(t *testing.T, store *Store, phase string) {
	t.Helper()
	mustCreate(t, store, testInputs())
	transition := func(to string, opts TransitionOpts) {
		t.Helper()
		if _, err := store.Transition(testMatID, to, opts); err != nil {
			t.Fatalf("setup Transition(%s) error = %v", to, err)
		}
	}
	switch phase {
	case PhaseStaging:
	case PhaseValidating:
		transition(PhaseValidating, TransitionOpts{})
	case PhasePrepared:
		transition(PhaseValidating, TransitionOpts{})
		transition(PhasePrepared, TransitionOpts{})
	case PhaseCommitting:
		transition(PhaseValidating, TransitionOpts{})
		transition(PhasePrepared, TransitionOpts{})
		transition(PhaseCommitting, TransitionOpts{})
	case PhaseRollingBack:
		transition(PhaseRollingBack, TransitionOpts{})
	case PhaseRolledBack:
		failure := mustFailure(t, "interrupted", "operator abort")
		if _, err := store.Rollback(testMatID, failure); err != nil {
			t.Fatalf("setup Rollback error = %v", err)
		}
	case PhaseCommitted:
		commitWorkspaceForTest(t, store)
	case PhaseFailed:
		transition(PhaseFailed, TransitionOpts{LastError: mustFrame(t, mustFailure(t, "materialization_failed", "failed"))})
	default:
		t.Fatalf("unknown setup phase %q", phase)
	}
}

// commitWorkspaceForTest converges the workspace journal to committed
// through the production entries.
func commitWorkspaceForTest(t *testing.T, store *Store) {
	t.Helper()
	backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
		RollbackRoot: &backup, State: AuthorityPrepared,
	}); err != nil {
		t.Fatalf("setup UpdateAuthority(prepared) error = %v", err)
	}
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
		State: AuthorityCommitted,
	}); err != nil {
		t.Fatalf("setup UpdateAuthority(committed) error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
		t.Fatalf("setup Transition(validating) error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
		t.Fatalf("setup Transition(prepared) error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
		t.Fatalf("setup Transition(committing) error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhaseCommitted, TransitionOpts{Marker: testMarker(t)}); err != nil {
		t.Fatalf("setup Transition(committed) error = %v", err)
	}
}

func TestTransitionGuards(t *testing.T) {
	t.Run("prepared_requires_opened", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// The bridge is still not_started: prepared refuses.
		_, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{})
		mustInvalid(t, err, "Transition(prepared)")
		// Imported is not opened either.
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		_, err = store.Transition(testMatID, PhasePrepared, TransitionOpts{})
		mustInvalid(t, err, "Transition(prepared)")
	})
	t.Run("prepared_requires_provider_prepared", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		provider := testProvider()
		provider.State = ProviderUnknown
		provider.RollbackToken = nil
		if _, err := store.UpdateProvider(testMatID, provider); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		_, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{})
		mustInvalid(t, err, "Transition(prepared)")
	})
	t.Run("committed_requires_convergence", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// Authorities and transactions have not converged.
		_, err := store.Transition(testMatID, PhaseCommitted, TransitionOpts{Marker: testMarker(t)})
		mustInvalid(t, err, "Transition(committed)")
	})
	t.Run("committed_workspace_requires_marker", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &backup, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
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
		_, err := store.Transition(testMatID, PhaseCommitted, TransitionOpts{})
		mustInvalid(t, err, "Transition(committed)")
		if !strings.Contains(err.Error(), "requires the destination marker") {
			t.Fatalf("Transition(committed) = %v, want the marker-evidence refusal", err)
		}
	})
	t.Run("committed_rejects_drifting_marker", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &backup, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
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
		// A marker bound to another plan is never destination
		// authority for this journal.
		var value map[string]any
		if err := json.Unmarshal(testMarker(t), &value); err != nil {
			t.Fatal(err)
		}
		value["plan_id"] = testPriorID
		foreign := identifyMarker(t, value)
		_, err := store.Transition(testMatID, PhaseCommitted, TransitionOpts{Marker: foreign})
		mustInvalid(t, err, "Transition(committed)")
	})
	t.Run("failed_requires_error", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		_, err := store.Transition(testMatID, PhaseFailed, TransitionOpts{})
		mustInvalid(t, err, "Transition(failed)")
		_, err = store.Transition(testMatID, PhaseFailed, TransitionOpts{LastError: json.RawMessage(`{"nope":true}`)})
		mustInvalid(t, err, "Transition(failed)")
		// An explicit null is still no explaining error.
		_, err = store.Transition(testMatID, PhaseFailed, TransitionOpts{LastError: json.RawMessage(`null`)})
		mustInvalid(t, err, "Transition(failed)")
	})
	t.Run("rollback_refused_after_adopt", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardAdopted()); err != nil {
			t.Fatal(err)
		}
		_, err := store.Transition(testMatID, PhaseRollingBack, TransitionOpts{})
		mustInvalid(t, err, "Transition(rolling_back)")
		_, err = store.Rollback(testMatID, mustFailure(t, "interrupted", "operator abort"))
		mustInvalid(t, err, "Rollback(adopted)")
	})
	t.Run("rollback_refused_after_provider_commit", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		provider := testProvider()
		provider.State = ProviderCommitted
		provider.RollbackToken = nil
		if _, err := store.UpdateProvider(testMatID, provider); err != nil {
			t.Fatal(err)
		}
		_, err := store.Transition(testMatID, PhaseRollingBack, TransitionOpts{})
		mustInvalid(t, err, "Transition(rolling_back)")
	})
	t.Run("rolled_back_requires_convergence", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseRollingBack, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// The provider is still prepared: closing refuses.
		_, err := store.Transition(testMatID, PhaseRolledBack, TransitionOpts{})
		mustInvalid(t, err, "Transition(rolled_back)")
	})
}

// mustConvergePrepared drives the composite journal to prepared
// through the production entries.
func mustConvergePrepared(t *testing.T, store *Store) {
	t.Helper()
	if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
		t.Fatalf("UpdateProvider error = %v", err)
	}
	providerRoot := testTxRoot + "/backups/codex_sessions"
	if _, err := store.UpdateAuthority(testMatID, "codex_sessions", AuthorityState{
		RootPath: testRootProvider, CompletedSequences: []uint64{2}, RollbackRoot: &providerRoot, State: AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, RollbackRoot: &workspaceRoot, State: AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
		t.Fatalf("UpdateTaskBoard error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
		t.Fatalf("UpdateTaskBoard error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
	if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
}

func TestProviderTokenAndIDRules(t *testing.T) {
	t.Run("prepared_requires_token", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		provider := testProvider()
		provider.RollbackToken = nil
		_, err := store.UpdateProvider(testMatID, provider)
		mustInvalid(t, err, "UpdateProvider(prepared)")
	})
	t.Run("terminal_forbids_token", func(t *testing.T) {
		for _, state := range []string{ProviderUnknown, ProviderCommitted, ProviderRolledBack} {
			store := openTestStore(t)
			mustCreate(t, store, testInputsComposite())
			provider := testProvider()
			provider.State = state
			_, err := store.UpdateProvider(testMatID, provider)
			mustInvalid(t, err, "UpdateProvider(token)")
		}
	})
	t.Run("short_token_refuses", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		provider := testProvider()
		provider.RollbackToken = str("YWJj")
		_, err := store.UpdateProvider(testMatID, provider)
		mustInvalid(t, err, "UpdateProvider(token)")
	})
	t.Run("padded_token_refuses", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		provider := testProvider()
		provider.RollbackToken = str(testProviderToken + "=")
		_, err := store.UpdateProvider(testMatID, provider)
		mustInvalid(t, err, "UpdateProvider(token)")
	})
	t.Run("authority_ids_bind", func(t *testing.T) {
		cases := []struct {
			name   string
			mutate func(*ProviderTransaction)
		}{
			{"materialization", func(p *ProviderTransaction) { p.Authority.MaterializationID = testTxID }},
			{"transaction", func(p *ProviderTransaction) { p.Authority.TransactionID = testPrepareOp }},
			{"plan", func(p *ProviderTransaction) { p.Authority.PlanID = testPriorID }},
			{"layout", func(p *ProviderTransaction) { p.Authority.Layout = "provider_transaction_v2" }},
			{"access", func(p *ProviderTransaction) { p.Authority.Access = "read_only" }},
			{"provider_id", func(p *ProviderTransaction) { p.Authority.ProviderID = "Codex!" }},
			{"same_filesystem_empty", func(p *ProviderTransaction) { p.Authority.SameFilesystemProviderAuthorityIDs = []string{} }},
			{"same_filesystem_unsorted", func(p *ProviderTransaction) {
				p.Authority.SameFilesystemProviderAuthorityIDs = []string{"workspace_relux", "codex_sessions"}
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				store := openTestStore(t)
				mustCreate(t, store, testInputsComposite())
				provider := testProvider()
				tc.mutate(&provider)
				_, err := store.UpdateProvider(testMatID, provider)
				mustInvalid(t, err, "UpdateProvider(authority)")
			})
		}
	})
	t.Run("transaction_drift_refuses", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		provider := testProvider()
		provider.TransactionID = testPrepareOp
		provider.Authority.TransactionID = testPrepareOp
		_, err := store.UpdateProvider(testMatID, provider)
		mustInvalid(t, err, "UpdateProvider(drift)")
	})
	t.Run("operation_id_may_advance", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		// The per-call operation ID advances with each plugin
		// call; only the transaction binding is stable.
		provider := testProvider()
		provider.OperationID = testAdoptOp
		if _, err := store.UpdateProvider(testMatID, provider); err != nil {
			t.Fatalf("UpdateProvider(operation advance) error = %v", err)
		}
	})
}

func TestBoardNullabilityRefusals(t *testing.T) {
	setup := func(t *testing.T) *Store {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		return store
	}
	t.Run("imported_triple_incomplete", func(t *testing.T) {
		store := setup(t)
		board := testBoardImported()
		board.ImportExpiresAt = nil
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(imported)")
	})
	t.Run("imported_keeps_open", func(t *testing.T) {
		store := setup(t)
		board := testBoardImported()
		board.OpenToken = str(testOpenToken)
		board.DormantManagerRef = str("tbm:dormant:x")
		board.OpenExpiresAt = str("2026-08-19T04:22:10.000Z")
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(imported)")
	})
	t.Run("opened_keeps_import", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		board := testBoardOpened()
		board.ImportToken = str(testImportToken)
		board.StagedManagerRef = str("tbm:staged:x")
		board.ImportExpiresAt = str("2026-08-19T04:22:10.000Z")
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(opened)")
	})
	t.Run("adopted_requires_binding", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		board := testBoardAdopted()
		board.AxBinding = nil
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(adopted)")
	})
	t.Run("adopted_requires_retained", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		board := testBoardAdopted()
		board.CleanupState = "not_started"
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(adopted)")
	})
	t.Run("adopted_keeps_token", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		board := testBoardAdopted()
		board.OpenToken = str(testOpenToken)
		board.DormantManagerRef = str("tbm:dormant:x")
		board.OpenExpiresAt = str("2026-08-19T04:22:10.000Z")
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(adopted)")
	})
	t.Run("dormant_requires_dormant_bridge", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsBoardOnly())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImportedDormant()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpenedDormant()); err != nil {
			t.Fatal(err)
		}
		board := testBoardOpenedDormant()
		running := "running"
		finalized := TaskBoardTransaction{
			BundleID: board.BundleID, ActivationMode: board.ActivationMode,
			ImportOperationID: board.ImportOperationID, OpenOperationID: board.OpenOperationID,
			AdoptOperationID: board.AdoptOperationID, ResumeOperationID: board.ResumeOperationID,
			State: BoardDormantFinalized, DormantManagerRef: board.DormantManagerRef,
			LastBridgeState: &running, LastStatusAt: board.LastStatusAt,
			CleanupState: "pending_expiry", CleanupAfter: board.OpenExpiresAt,
		}
		_, err := store.UpdateTaskBoard(testMatID, finalized)
		mustInvalid(t, err, "UpdateTaskBoard(dormant)")
	})
	t.Run("dormant_expiry_binds_consumed", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsBoardOnly())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImportedDormant()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpenedDormant()); err != nil {
			t.Fatal(err)
		}
		board := testBoardOpenedDormant()
		dormant := "dormant"
		finalized := TaskBoardTransaction{
			BundleID: board.BundleID, ActivationMode: board.ActivationMode,
			ImportOperationID: board.ImportOperationID, OpenOperationID: board.OpenOperationID,
			AdoptOperationID: board.AdoptOperationID, ResumeOperationID: board.ResumeOperationID,
			State: BoardDormantFinalized, DormantManagerRef: board.DormantManagerRef,
			LastBridgeState: &dormant, LastStatusAt: board.LastStatusAt,
			CleanupState: "pending_expiry", CleanupAfter: str("2026-08-19T05:22:10.000Z"),
		}
		_, err := store.UpdateTaskBoard(testMatID, finalized)
		mustInvalid(t, err, "UpdateTaskBoard(dormant)")
	})
	t.Run("failed_partial_triple", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		board := testBoardImported()
		board.State = BoardFailed
		board.StagedManagerRef = nil
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(failed)")
	})
	t.Run("failed_two_pairs", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		board := testBoardImported()
		board.State = BoardFailed
		board.OpenToken = str(testOpenToken)
		board.DormantManagerRef = str("tbm:dormant:x")
		board.OpenExpiresAt = str("2026-08-19T04:22:10.000Z")
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(failed)")
	})
	t.Run("failed_single_pair_admits", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		board := testBoardImported()
		board.State = BoardFailed
		if _, err := store.UpdateTaskBoard(testMatID, board); err != nil {
			t.Fatalf("UpdateTaskBoard(failed) error = %v", err)
		}
	})
	t.Run("cleanup_after_only_dormant", func(t *testing.T) {
		store := setup(t)
		board := testBoardImported()
		board.CleanupAfter = str("2026-08-19T05:22:10.000Z")
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(cleanup)")
	})
	t.Run("operation_drift_refuses", func(t *testing.T) {
		store := setup(t)
		board := testBoardImported()
		board.AdoptOperationID = testPrepareOp
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(drift)")
	})
	t.Run("bundle_and_mode_bind", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		board := testBoardOpened()
		board.BundleID = testPriorID
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(bundle)")
		board = testBoardOpened()
		board.ActivationMode = ModeFork
		_, err = store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(mode)")
	})
	t.Run("subtable_refuses_skip", func(t *testing.T) {
		store := setup(t)
		// not_started reaches imported, never opened directly.
		_, err := store.UpdateTaskBoard(testMatID, testBoardOpened())
		mustInvalid(t, err, "UpdateTaskBoard(skip)")
	})
	t.Run("subtable_refuses_adopted_rollback", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardAdopted()); err != nil {
			t.Fatal(err)
		}
		board := testBoardAdopted()
		board.State = BoardRolledBack
		board.ManagerSessionRef, board.AxBinding = nil, nil
		board.CleanupState = "removed"
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(adopted rollback)")
	})
	t.Run("binding_grammar", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		board := testBoardAdopted()
		board.AxBinding.LeaseEpoch = 0
		_, err := store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(binding)")
		board = testBoardAdopted()
		board.AxBinding.LeaseID = "not-a-uuid"
		_, err = store.UpdateTaskBoard(testMatID, board)
		mustInvalid(t, err, "UpdateTaskBoard(binding)")
	})
}

func TestAuthorityGuards(t *testing.T) {
	setup := func(t *testing.T) *Store {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		return store
	}
	t.Run("prepared_requires_root", func(t *testing.T) {
		store := setup(t)
		_, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityPrepared,
		})
		mustInvalid(t, err, "UpdateAuthority(prepared)")
	})
	t.Run("committed_requires_null_root", func(t *testing.T) {
		store := setup(t)
		backup := "/tmp/backup"
		_, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &backup, State: AuthorityCommitted,
		})
		mustInvalid(t, err, "UpdateAuthority(committed)")
	})
	t.Run("staging_completed_requires_root", func(t *testing.T) {
		store := setup(t)
		_, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityStaging,
		})
		mustInvalid(t, err, "UpdateAuthority(staging)")
	})
	t.Run("provider_backup_is_single", func(t *testing.T) {
		store := setup(t)
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		second := "/tmp/second-backup"
		_, err := store.UpdateAuthority(testMatID, "codex_sessions", AuthorityState{
			RootPath: testRootProvider, CompletedSequences: []uint64{2},
			RollbackRoot: &second, State: AuthorityPrepared,
		})
		mustInvalid(t, err, "UpdateAuthority(backup)")
	})
	t.Run("provider_backup_needs_transaction", func(t *testing.T) {
		store := setup(t)
		orphan := testTxRoot + "/backups/codex_sessions"
		_, err := store.UpdateAuthority(testMatID, "codex_sessions", AuthorityState{
			RootPath: testRootProvider, CompletedSequences: []uint64{2},
			RollbackRoot: &orphan, State: AuthorityPrepared,
		})
		mustInvalid(t, err, "UpdateAuthority(backup)")
	})
	t.Run("root_binds_plan", func(t *testing.T) {
		store := setup(t)
		_, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: "/srv/other", CompletedSequences: []uint64{}, State: AuthorityStaging,
		})
		mustInvalid(t, err, "UpdateAuthority(root)")
	})
	t.Run("unknown_authority", func(t *testing.T) {
		store := setup(t)
		_, err := store.UpdateAuthority(testMatID, "nope", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{}, State: AuthorityStaging,
		})
		mustInvalid(t, err, "UpdateAuthority(unknown)")
	})
	t.Run("subtable_refuses_skip", func(t *testing.T) {
		store := setup(t)
		_, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{}, State: AuthorityCommitted,
		})
		mustInvalid(t, err, "UpdateAuthority(skip)")
	})
	t.Run("sequences_sorted_unique", func(t *testing.T) {
		store := setup(t)
		backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		_, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{2, 1},
			RollbackRoot: &backup, State: AuthorityPrepared,
		})
		mustInvalid(t, err, "UpdateAuthority(sequences)")
	})
}

func TestTerminalFreezes(t *testing.T) {
	for _, phase := range []string{PhaseCommitted, PhaseRolledBack, PhaseFailed} {
		t.Run(phase, func(t *testing.T) {
			store := openTestStore(t)
			setupPhase(t, store, phase)
			if _, err := store.UpdateProvider(testMatID, testProvider()); err == nil {
				t.Fatalf("UpdateProvider(%s) error = nil", phase)
			} else {
				mustInvalid(t, err, "UpdateProvider(terminal)")
			}
			if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {1}}, nil); err == nil {
				t.Fatalf("RecordProgress(%s) error = nil", phase)
			} else {
				mustInvalid(t, err, "RecordProgress(terminal)")
			}
			if _, err := store.RecordError(testMatID, mustFailure(t, "interrupted", "late")); err == nil {
				t.Fatalf("RecordError(%s) error = nil", phase)
			} else {
				mustInvalid(t, err, "RecordError(terminal)")
			}
			if _, err := store.Rollback(testMatID, mustFailure(t, "interrupted", "late")); err == nil {
				t.Fatalf("Rollback(%s) error = nil", phase)
			} else {
				mustInvalid(t, err, "Rollback(terminal)")
			}
		})
	}
}

func TestGetRefusals(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	// Malformed identity refuses the invalid class.
	_, _, err := store.Get("not-a-uuid")
	mustInvalid(t, err, "Get(malformed)")
	// Absent materialization is absence, never torn state.
	_, _, err = store.Get(testTxID)
	if !errors.Is(err, ErrUnknownJournal) {
		t.Fatalf("Get(absent) = %v, want ErrUnknownJournal", err)
	}
	if errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("Get(absent) = %v, want absence, not the invalid class", err)
	}
	// Torn journal bytes refuse attestation on read.
	matDir := filepath.Join(store.root, testMatID)
	if err := os.WriteFile(filepath.Join(matDir, "journal.json"), []byte(`{"schema":"urn:ax:schema:materialization-journal"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = store.Get(testMatID)
	mustInvalid(t, err, "Get(torn)")
}

func TestRollbackConverges(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	mustConvergePrepared(t, store)
	failure := mustFailure(t, "interrupted", "operator abort before activation")
	rolled, err := store.Rollback(testMatID, failure)
	if err != nil {
		t.Fatalf("Rollback error = %v", err)
	}
	if rolled.Phase != PhaseRolledBack {
		t.Fatalf("phase = %q", rolled.Phase)
	}
	// Tokens are dropped from the live document; the terminal
	// failure is visible; every sub-state converged.
	if rolled.Provider.State != ProviderRolledBack || rolled.Provider.RollbackToken != nil {
		t.Fatalf("provider = %+v", rolled.Provider)
	}
	if rolled.TaskBoard.State != BoardRolledBack || rolled.TaskBoard.CleanupState != "removed" {
		t.Fatalf("task-board = %+v", rolled.TaskBoard)
	}
	for id, state := range rolled.AuthorityStates {
		if state.State != AuthorityRolledBack || state.RollbackRoot != nil {
			t.Fatalf("authority %q = %+v", id, state)
		}
	}
	decoded, err := axerror.Decode(lastErrorVersion, rolled.LastError)
	if err != nil {
		t.Fatalf("last_error decodes: %v", err)
	}
	if string(decoded.Code()) != "interrupted" {
		t.Fatalf("code = %q", decoded.Code())
	}
	// Rollback without the explaining failure refuses.
	store2 := openTestStore(t)
	mustCreate(t, store2, testInputs())
	_, err = store2.Rollback(testMatID, nil)
	mustInvalid(t, err, "Rollback(nil)")
}
