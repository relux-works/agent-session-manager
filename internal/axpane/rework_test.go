package axpane

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestDecideCheckpointNotLeaseParks pins P1-4 (probe 6): a valid
// checkpoint that is neither the fold's newest nor the winning
// lease's checkpoint parks restore_policy, never launches. On rev1
// (digest-only admission) this launched.
func TestDecideCheckpointNotLeaseParks(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = seedDigest(0xC9)
	input.Observation.Winner.Checkpoint = seedDigest(0xC9)
	input.Observation.Winner.HasCheckpoint = true
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Decide() = (%q, %q), want parked restore_policy for non-lease checkpoint", decision.Action, decision.ParkReason)
	}
}

// TestDecideForeignCheckpointParks pins P1-4 (probe 6b): a valid
// checkpoint captured for another session parks, never launches.
// On rev1 (no session_id binding) this launched.
func TestDecideForeignCheckpointParks(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	repository, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil {
		t.Fatal(err)
	}
	record["subject_id"] = fixtureForeign
	record["session_id"] = fixtureForeign
	reference, err := repository.CreateSession(identifyObject(t, record, "record_id"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateLease(fixtureForeign, sessrepo.CreateLeaseInput{LeaseID: fixtureLeaseA, HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt}); err != nil {
		t.Fatal(err)
	}
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"subject_id": fixtureForeign, "session_id": fixtureForeign, "event_type": "session.created", "created_by_host_id": fixtureLocalHost,
		"lease_epoch": 1, "lease_id": fixtureLeaseA, "lease_sequence": 1, "predecessors": []string{reference.RecordID},
		"created_at": fixtureCreatedAt, "payload": map[string]any{"session_record_id": reference.RecordID, "bootstrap_operation_id": fixtureBootstrap, "first_checkpoint_operation_id": fixtureOtherOp}, "extensions": map[string]any{},
	}
	created, err := repository.AppendEvent(fixtureForeign, identifyObject(t, value, "event_id"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessckpt.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	foreignID, foreignDoc := checkpointFixtureForRework(t, repository, store, fixtureForeign, []string{created.EventID})
	input := validInput(t, deps)
	input.Mode = ModeRestore
	// Even when the fold's newest and the winning lease's handoff
	// base agree on the foreign digest, the session binding parks.
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = foreignID
	input.Observation.Winner.Checkpoint = foreignID
	input.Observation.Winner.HasCheckpoint = true
	input.CheckpointRequired = true
	input.CheckpointDoc = foreignDoc
	input.CheckpointID = foreignID
	decision := Decide(input)
	if decision.Action != ActionParked {
		t.Fatalf("Decide() action = %q, want parked for foreign checkpoint", decision.Action)
	}
}

func checkpointFixtureForRework(t *testing.T, repository *sessrepo.Repository, store *sessckpt.Store, sessionID string, heads []string) (string, []byte) {
	t.Helper()
	reference, raw, err := store.Capture(repository, sessckpt.Inputs{
		OperationID:         fixtureBootstrap,
		SessionID:           sessionID,
		SessionKind:         sessckpt.SessionKindDirect,
		LeaseEpoch:          1,
		LeaseID:             fixtureLeaseA,
		CreatorHostID:       fixtureLocalHost,
		WorkspaceManifestID: seedDigest(0xE1),
		ProviderManifestID:  seedDigest(0xE2),
		Boundary: sessckpt.SafeBoundary{
			ProviderID:          "codex",
			ProviderVersion:     "0.147.0",
			Evidence:            sessckpt.EvidenceAcceptedTest,
			InputBlocked:        true,
			ForegroundIdle:      true,
			BackgroundIdle:      true,
			OpenProcesses:       0,
			OpenDatabaseHandles: 0,
		},
		EventHeads: heads,
		CreatedAt:  fixtureCreatedAt,
		Extensions: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	return reference.CheckpointID, raw
}

// TestDecideMaterializationStaleParks pins P2-2 (probe 7): a
// committed journal sourced from a superseded checkpoint parks,
// never launches. On rev1 (Phase-only) this launched.
func TestDecideMaterializationStaleParks(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = deps.ckptID
	input.MaterializationRequired = true
	input.JournalOK = true
	input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: seedDigest(0xD2), MaterializationID: fixtureMat}
	input.Observation.Winner.Checkpoint = deps.ckptID
	input.Observation.Winner.HasCheckpoint = true
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Decide() = (%q, %q), want parked restore_policy for stale materialization", decision.Action, decision.ParkReason)
	}
}

// TestDecideResumeProfileUsesClosure pins P2-1 (probe 8): resume
// from checkpoint C1 (closure [created]) with a head-only
// profile.changed E2 carries the closure pair (standard/null), not
// the head pair (yolo/E2). On rev1 (Derive head) this carried
// yolo/E2.
func TestDecideResumeProfileUsesClosure(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	changed := appendChainEvent(t, deps.repo, "profile.changed", 1, fixtureLeaseA, 2, deps.createdID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	_ = changed
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = deps.ckptID
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() action = %q, cause %v, want launch", decision.Action, decision.Cause)
	}
	if decision.Profile.Profile != "standard" || decision.Profile.HasSource {
		t.Fatalf("Decide() profile = %+v, want standard with no source from the closure", decision.Profile)
	}
}

// TestDecideNonInteractiveRequiresHeadless pins P2-3 (probe 4): a
// non-interactive create with headless_creation=false refuses
// capability_unproven even when durable_disconnect confers create.
// On rev1 (no Interactive input) this launched.
func TestDecideNonInteractiveRequiresHeadless(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	claims := deps.universe.probe["capability_claims"].([]any)
	for _, raw := range claims {
		claim := raw.(map[string]any)
		if claim["capability"] == "headless_creation" {
			claim["value"] = false
		}
	}
	kept := deps.universe.evidence[:0]
	for _, object := range deps.universe.evidence {
		if object["capability"].(string) != "headless_creation" {
			kept = append(kept, object)
		}
	}
	deps.universe.evidence = kept
	var ids []any
	for _, object := range deps.universe.evidence {
		ids = append(ids, object["evidence_id"])
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].(string) < ids[j].(string) })
	deps.universe.probe["evidence_ids"] = ids
	deps.universe.probe["probe_id"] = omitSelfIdentity(t, deps.universe.probe, "probe_id")
	input := validInput(t, deps)
	input.Interactive = false
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != terminalbackend.CodeCapabilityUnproven {
		t.Fatalf("Decide() = (%q, %q), want refused capability_unproven for non-interactive without headless", decision.Action, decision.Class)
	}
}

// TestDecidePostWindowNewOperationLaunches pins P1-3: after the
// window closes (the fold names a newest checkpoint) a new
// bootstrap operation launches, never refuses idempotency_mismatch.
// On rev1 (one binding per session forever) this refused.
func TestDecidePostWindowNewOperationLaunches(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.BootstrapOperationID = fixtureOtherOp
	input.SessionBound = true
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = deps.ckptID
	input.Observation.Winner.Checkpoint = deps.ckptID
	input.Observation.Winner.HasCheckpoint = true
	input.MaterializationRequired = true
	input.JournalOK = true
	input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() = (%q, %q, %v), want launch for post-window new operation", decision.Action, decision.Class, decision.Cause)
	}
}

// TestDecideRestoreChangedOperationRefuses pins P1-3/RM4: restore
// mode with a changed bootstrap operation inside the window refuses
// idempotency_mismatch. On rev1 no restore-mode idempotency test
// existed and RM4 survived.
func TestDecideRestoreChangedOperationRefuses(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.SessionBound = true
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != terminalbackend.CodeIdempotencyMismatch {
		t.Fatalf("Decide() = (%q, %q), want refused idempotency_mismatch for restore changed op", decision.Action, decision.Class)
	}
}

// TestRunPostWindowSupersedes pins P1-3 (probe 5) through Run: create
// binds op1, the chain publishes the checkpoint and a successor
// lease carries it as the handoff base, and restore with op2
// launches with its own pair receipt while op1's receipt is kept.
// On rev1 this refused idempotency_mismatch.
func TestRunPostWindowSupersedes(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	first, err := Run(world.stores(), runRequest(t, world))
	if err != nil || first.Decision.Action != ActionLaunch {
		t.Fatalf("first launch = (%v, %v)", first.Decision.Action, err)
	}
	successor := "cccccccc-dddd-4eee-8fff-000000000001"
	successorLease(t, world.repo, successor, fixtureLocalHost, world.ckptID)
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	world.mat = journalSourced(t, world.ckptID)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.BootstrapOperationID = fixtureOtherOp
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: successor}
	request.MaterializationRequired = true
	request.MaterializationID = fixtureMat
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() action = %q, class %q, cause %v, want launch", outcome.Decision.Action, outcome.Decision.Class, outcome.Decision.Cause)
	}
	if outcome.Binding == nil || outcome.Binding.OperationID != fixtureOtherOp {
		t.Fatalf("Run() binding = %+v, want the op2 pair receipt", outcome.Binding)
	}
	second, found, err := world.pane.Lookup(fixtureSession, fixtureOtherOp)
	if err != nil || !found || second != *outcome.Binding {
		t.Fatalf("Lookup(op2) = (%+v, %v, %v), want the recorded op2 receipt", second, found, err)
	}
	kept, found, err := world.pane.Lookup(fixtureSession, fixtureBootstrap)
	if err != nil || !found || kept != *first.Binding {
		t.Fatalf("Lookup(op1) = (%+v, %v, %v), want the kept op1 receipt", kept, found, err)
	}
	anchor, found, err := world.pane.Status(fixtureSession)
	if err != nil || !found || anchor != *first.Binding {
		t.Fatalf("Status() = (%+v, %v, %v), want the first-binding anchor", anchor, found, err)
	}
}

// TestDecideRemoteOwnerPrecedesMaterialization pins P2-5/RM2: a
// remote interactive owner with a staging materialization offers
// attach_remote (authorize before materialization), never parks
// restore_policy. On the RM2 mutant (steps 3/4 swapped) this parks.
func TestDecideRemoteOwnerPrecedesMaterialization(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.Observation.Winner.HolderHostID = fixtureRemoteHost
	input.RemoteInteractive = true
	input.MaterializationRequired = true
	input.JournalOK = true
	input.Journal = matjournal.Journal{Phase: matjournal.PhaseStaging}
	decision := Decide(input)
	if decision.Action != ActionAttachRemote {
		t.Fatalf("Decide() = (%q, %q), want attach_remote (authorize precedes materialization)", decision.Action, decision.ParkReason)
	}
}

// TestRunTornLeasePropagates pins P2-5/RM1: a torn lease blob fails
// Run with the read error and no decision, never parks. On the RM1
// mutant (ObserveOwnership error swallowed) this parked.
func TestRunTornLeasePropagates(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	root := t.TempDir()
	repository, err := sessrepo.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil {
		t.Fatal(err)
	}
	reference, err := repository.CreateSession(identifyObject(t, record, "record_id"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateLease(fixtureSession, sessrepo.CreateLeaseInput{LeaseID: fixtureLeaseA, HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt}); err != nil {
		t.Fatal(err)
	}
	appendChainEvent(t, repository, "session.created", 1, fixtureLeaseA, 1, reference.RecordID, map[string]any{
		"session_record_id": reference.RecordID, "bootstrap_operation_id": fixtureBootstrap, "first_checkpoint_operation_id": fixtureOtherOp,
	})
	world.repo = repository
	matches, _ := filepath.Glob(filepath.Join(root, "sessions", fixtureSession, "leases", "*"))
	if len(matches) == 0 {
		t.Fatal("no lease blobs to tear")
	}
	for _, match := range matches {
		if err := os.WriteFile(match, []byte("{torn"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outcome, err := Run(world.stores(), runRequest(t, world))
	if err == nil {
		t.Fatalf("Run() over torn lease succeeded with %+v, want the read error", outcome.Decision)
	}
	if outcome.Decision.Action != "" {
		t.Fatalf("Run() over torn lease decides %q, want no decision", outcome.Decision.Action)
	}
}

// TestRunParkedUnfoldableSkipsEmission pins P1-1 (probe 2): a verified
// local park from a creating chain parks without authoring the
// unfolderable session.parked event. On rev1 this authored creating
// -> parked, which sessstate refuses.
func TestRunParkedUnfoldableSkipsEmission(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.MaterializationRequired = true
	request.MaterializationID = "0198f4c8-9a10-7b22-8b3c-1234567890ff"
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	before := eventCount(t, world.repo)
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked {
		t.Fatalf("Run() action = %q, want parked", outcome.Decision.Action)
	}
	if outcome.Emitted {
		t.Fatal("Run() authored session.parked from creating, want no emission")
	}
	if after := eventCount(t, world.repo); after != before {
		t.Fatalf("event count = %d, want %d (no unfolderable emission)", after, before)
	}
	// The chain without the parked event still folds; with it, the
	// landed reducer refuses (the rev1 collision).
	recordJSON, err := world.repo.GetRecord(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	record, err := sessstate.DecodeRecord(recordJSON)
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := world.repo.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	var events []sessstate.Event
	for _, summary := range summaries {
		raw, err := world.repo.GetEvent(fixtureSession, summary.EventID)
		if err != nil {
			t.Fatal(err)
		}
		event, err := sessstate.DecodeEvent(raw)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if _, err := sessstate.Reduce(sessstate.Input{Record: record, Events: events, LocalHostID: fixtureLocalHost}); err != nil {
		t.Fatalf("sessstate.Reduce over parked-without-emission chain error = %v", err)
	}
}

// TestEmitUnderLosingLeaseRefuses pins P1-1 (probe 9): Emit under a
// losing lease refuses at the AuthorizeMutation gate, never
// appends. The chain is foldable (failed), so the refusal comes
// from the mutation gate itself — not_owner under a remote winner,
// stale_owner under a local successor — and never from the fold
// pre-check. On rev1 (no mutation gate) this appended.
func TestEmitUnderLosingLeaseRefuses(t *testing.T) {
	t.Parallel()
	t.Run("remote", func(t *testing.T) {
		t.Parallel()
		world := buildRunWorld(t)
		failedID := appendChainEvent(t, world.repo, "session.failed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
			"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
		})
		successorLease(t, world.repo, fixtureLeaseB, fixtureRemoteHost, seedDigest(0xC9))
		observation, err := ObserveOwnership(world.repo, fixtureSession, runObserve(fixtureNow()))
		if err != nil {
			t.Fatal(err)
		}
		decision := Decision{Action: ActionParked, ParkReason: fencing.ParkStaleOwner, WinningLeaseID: fixtureLeaseA}
		_, _, err = EmitParked(world.repo, decision, EmitParams{
			SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
			LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: 3,
			Predecessors: []string{failedID}, CreatedAt: fixtureCreatedAt,
			Presented:   fencing.PresentedToken{SessionID: fixtureSession, Epoch: 1, LeaseID: fixtureLeaseA},
			Observation: observation,
		})
		if err == nil {
			t.Fatal("EmitParked under losing lease succeeded, want refusal")
		}
		if !errors.Is(err, fencing.ErrNotOwner) {
			t.Fatalf("EmitParked under losing lease error = %v, want the not_owner mutation refusal", err)
		}
		if strings.Contains(err.Error(), "does not fold") {
			t.Fatalf("EmitParked refused at the fold pre-check (%v), want the mutation gate", err)
		}
	})
	t.Run("local", func(t *testing.T) {
		t.Parallel()
		world := buildRunWorld(t)
		failedID := appendChainEvent(t, world.repo, "session.failed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
			"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
		})
		successor := "cccccccc-dddd-4eee-8fff-000000000001"
		successorLease(t, world.repo, successor, fixtureLocalHost, seedDigest(0xC9))
		observation, err := ObserveOwnership(world.repo, fixtureSession, runObserve(fixtureNow()))
		if err != nil {
			t.Fatal(err)
		}
		decision := Decision{Action: ActionParked, ParkReason: fencing.ParkStaleOwner, WinningLeaseID: fixtureLeaseA}
		_, _, err = EmitParked(world.repo, decision, EmitParams{
			SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
			LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: 3,
			Predecessors: []string{failedID}, CreatedAt: fixtureCreatedAt,
			Presented:   fencing.PresentedToken{SessionID: fixtureSession, Epoch: 1, LeaseID: fixtureLeaseA},
			Observation: observation,
		})
		if err == nil {
			t.Fatal("EmitParked under losing lease succeeded, want refusal")
		}
		if !errors.Is(err, fencing.ErrStaleOwner) {
			t.Fatalf("EmitParked under losing lease error = %v, want the stale_owner mutation refusal", err)
		}
		if strings.Contains(err.Error(), "does not fold") {
			t.Fatalf("EmitParked refused at the fold pre-check (%v), want the mutation gate", err)
		}
	})
}

// TestLosingLeaseProfileEventIgnored pins P1-1 (probe 15, §2.4): a
// profile.changed authored under the superseded lease after a local
// successor won never becomes the launch profile. On rev1 this
// drove yolo/E2.
func TestLosingLeaseProfileEventIgnored(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	leases, err := world.repo.ListLeases(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	successor := "cccccccc-dddd-4eee-8fff-000000000002"
	if _, err := world.repo.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: successor, HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt},
		Reason:           "graceful_takeover",
		CheckpointID:     world.ckptID,
	}); err != nil {
		t.Fatalf("CompareAndSwapLease() error = %v", err)
	}
	changed := appendChainEvent(t, world.repo, "profile.changed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	request := runRequest(t, world)
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action == ActionLaunch && outcome.Decision.Profile.Source == changed {
		t.Fatalf("Run() launch profile source = losing-lease %s, want the checkpoint closure", changed[:20])
	}
	if outcome.Decision.Action == ActionLaunch && outcome.Decision.Profile.Profile != "standard" {
		t.Fatalf("Run() launch profile = %+v, want standard from the closure", outcome.Decision.Profile)
	}
}

// TestDecideRealmBindingMismatchRefuses pins P1-2: realm evidence
// bound to another host binding or provider build refuses
// capability_unavailable, never launches.
func TestDecideRealmBindingMismatchRefuses(t *testing.T) {
	t.Parallel()
	t.Run("binding", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB2), "codex", "0.147.0")
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
			t.Fatalf("Decide() = (%q, %q), want refused for binding mismatch", decision.Action, decision.Class)
		}
	})
	t.Run("build", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "9.9.9")
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
			t.Fatalf("Decide() = (%q, %q), want refused for build mismatch", decision.Action, decision.Class)
		}
	})
	t.Run("generation", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		input.Realm.ServerGeneration = "generation-stale"
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
			t.Fatalf("Decide() = (%q, %q), want refused for stale generation", decision.Action, decision.Class)
		}
	})
}

// TestEmitParkedUnfoldableRefuses pins P1-1 through the direct
// EmitParked entry: a restore_policy park from a creating chain
// refuses, never appends. On rev1 this appended creating -> parked.
func TestEmitParkedUnfoldableRefuses(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.MaterializationRequired = true
	input.JournalOK = true
	decision := Decide(input)
	if decision.Action != ActionParked {
		t.Fatalf("Decide() action = %q, want parked", decision.Action)
	}
	_, tail, err := ChainTail(deps.repo, fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	observation := fixtureObservation(fixtureNow())
	if _, _, err := EmitParked(deps.repo, decision, EmitParams{
		SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: tail.LeaseSequence + 1,
		Predecessors: []string{tail.EventID}, CreatedAt: fixtureCreatedAt,
		Presented:   fixturePresented(),
		Observation: observation,
	}); err == nil {
		t.Fatal("EmitParked from creating succeeded, want refusal")
	}
}

// TestDecideSmokeTargetBindingRefuses pins P1-2: a required smoke
// with a typed target bound to another build or OS version refuses
// target_auth_missing, never launches.
func TestDecideSmokeTargetBindingRefuses(t *testing.T) {
	t.Parallel()
	t.Run("build", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Smoke.Required = true
		input.Smoke.Record = smokeRecord(t, "pass", "A", passingSmokeChecks(), smokeTuple())
		target := smokeTarget()
		target.ProviderBuild = "9.9.9"
		input.Smoke.Target = target
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "target_auth_missing" {
			t.Fatalf("Decide() = (%q, %q), want refused target_auth_missing for target build drift", decision.Action, decision.Class)
		}
	})
	t.Run("os", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Smoke.Required = true
		input.Smoke.Record = smokeRecord(t, "pass", "A", passingSmokeChecks(), smokeTuple())
		target := smokeTarget()
		target.MacOSVersion = "15.0"
		input.Smoke.Target = target
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "target_auth_missing" {
			t.Fatalf("Decide() = (%q, %q), want refused target_auth_missing for target OS drift", decision.Action, decision.Class)
		}
	})
}

// TestIsFencingRefusal covers the Run emission error classifier: a
// fencing gate refusal parks without evidence, while a repository
// failure propagates.
func TestIsFencingRefusal(t *testing.T) {
	t.Parallel()
	if isFencingRefusal(nil) {
		t.Fatal("isFencingRefusal(nil) = true, want false")
	}
	if !isFencingRefusal(fencing.ErrNotOwner) {
		t.Fatal("isFencingRefusal(ErrNotOwner) = false, want true")
	}
	if isFencingRefusal(errCheckpointMembers()) {
		t.Fatal("isFencingRefusal(checkpoint) = true, want false")
	}
}

// TestBindSweepsStaleStaging pins P3-1: a pre-commit crash staging
// file older than a minute is swept by the next Bind, while fresh
// concurrent staging is left alone.
func TestBindSweepsStaleStaging(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	dir := store.sessionDir(fixtureSession)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "binding-9999999999.tmp")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale staging file survives Bind: %v", err)
	}
}
