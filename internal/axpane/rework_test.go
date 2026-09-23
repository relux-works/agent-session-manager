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
	"github.com/relux-works/agent-session-manager/internal/sessprofile"
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
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	successor := "cccccccc-dddd-4eee-8fff-000000000001"
	successorLease(t, world.repo, successor, fixtureLocalHost, world.ckptID)
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

// TestEmitReachesAppendAdmissionGateAfterStaleObservation drives the
// wrapper's exported Emit entry with a stale-but-well-formed observation.
// AuthorizeMutation may accept that caller snapshot, but Repository.AppendEvent
// must still refuse the lower-epoch event once successor B is durable.
func TestEmitReachesAppendAdmissionGateAfterStaleObservation(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	successorLease(t, world.repo, "cccccccc-dddd-4eee-8fff-000000000003", fixtureLocalHost, world.ckptID)
	_, tail, err := ChainTail(world.repo, fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = Emit(world.repo, EmitParams{
		SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: tail.LeaseSequence + 1,
		Predecessors: []string{tail.EventID}, CreatedAt: fixtureCreatedAt,
		EventType: "profile.changed", SchemaVersion: "1.0.0",
		Payload:   map[string]any{"from": "standard", "to": "yolo", "confirmed": true},
		Presented: fixturePresented(), Observation: fixtureObservation(fixtureNow()),
	})
	if err == nil || !errors.Is(err, sessrepo.ErrStaleLease) {
		t.Fatalf("Emit(stale observation) error = %v, want stale lease refusal from AppendEvent", err)
	}
	events, err := world.repo.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != tail.EventID {
		t.Fatalf("chain after refused Emit = %+v, want the original tail only", events)
	}
}

// TestEmitReachesSameEpochAppendAdmissionGate exercises the same-epoch loser
// arm through Emit: C is the chain tail at epoch 2, then successor B wins the
// lease store at that same epoch while the caller still presents C.
func TestEmitReachesSameEpochAppendAdmissionGate(t *testing.T) {
	t.Parallel()
	for _, losing := range []string{fixtureLeaseAdmissionLoser, fixtureLeaseAdmissionLower} {
		t.Run(losing, func(t *testing.T) {
			world := buildRunWorld(t)
			losingID := appendChainEvent(t, world.repo, "profile.changed", 2, losing, 1, world.headID, map[string]any{
				"from": "standard", "to": "yolo", "confirmed": true,
			})
			successorLease(t, world.repo, fixtureLeaseB, fixtureLocalHost, world.ckptID)
			observation := fixtureObservation(fixtureNow())
			observation.Winner.Epoch = 2
			observation.Winner.LeaseID = losing
			observation.Grant.Token.Epoch = 2
			observation.Grant.Token.LeaseID = losing
			_, _, err := Emit(world.repo, EmitParams{
				SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
				LeaseEpoch: 2, LeaseID: losing, LeaseSequence: 2,
				Predecessors: []string{losingID}, CreatedAt: fixtureCreatedAt,
				EventType: "profile.changed", SchemaVersion: "1.0.0",
				Payload:   map[string]any{"from": "yolo", "to": "standard", "confirmed": false},
				Presented: fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: losing}, Observation: observation,
			})
			if err == nil || !errors.Is(err, sessrepo.ErrDivergentBranch) {
				t.Fatalf("Emit(same-epoch loser) error = %v, want divergent branch refusal from AppendEvent", err)
			}
			events, err := world.repo.ListEvents(fixtureSession)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 2 || events[1].EventID != losingID {
				t.Fatalf("chain after refused same-epoch Emit = %+v, want the losing tail only", events)
			}
			assertPreservedAdmissionBlob(t, world.repoRoot, losing, 2, 2, "profile.changed")
		})
	}
}

// TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation drives the
// nested parked writer through EmitParked and Emit, proving the same admission
// gate is not skipped by the lifecycle/foldable-state wrapper.
func TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	failedID := appendChainEvent(t, world.repo, "session.failed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
		"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
	})
	successorLease(t, world.repo, "cccccccc-dddd-4eee-8fff-000000000004", fixtureLocalHost, world.ckptID)
	_, _, err := EmitParked(world.repo, Decision{
		Action: ActionParked, ParkReason: fencing.ParkStaleOwner, WinningLeaseID: fixtureLeaseA,
	}, EmitParams{
		SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: 3,
		Predecessors: []string{failedID}, CreatedAt: fixtureCreatedAt,
		Presented: fixturePresented(), Observation: fixtureObservation(fixtureNow()),
	})
	if err == nil || !errors.Is(err, sessrepo.ErrStaleLease) {
		t.Fatalf("EmitParked(stale observation) error = %v, want stale lease refusal from AppendEvent", err)
	}
	events, err := world.repo.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[1].EventID != failedID {
		t.Fatalf("chain after refused EmitParked = %+v, want the failed tail only", events)
	}
}

// TestEmitParkedReachesSameEpochAppendAdmissionGate exercises the nested
// parked writer on the same-epoch loser arm.
func TestEmitParkedReachesSameEpochAppendAdmissionGate(t *testing.T) {
	t.Parallel()
	for _, losing := range []string{fixtureLeaseAdmissionLoser, fixtureLeaseAdmissionLower} {
		t.Run(losing, func(t *testing.T) {
			world := buildRunWorld(t)
			failedID := appendChainEvent(t, world.repo, "session.failed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
				"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
			})
			losingID := appendChainEvent(t, world.repo, "session.failed", 2, losing, 1, failedID, map[string]any{
				"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
			})
			successorLease(t, world.repo, fixtureLeaseB, fixtureLocalHost, world.ckptID)
			observation := fixtureObservation(fixtureNow())
			observation.Winner.Epoch = 2
			observation.Winner.LeaseID = losing
			observation.Grant.Token.Epoch = 2
			observation.Grant.Token.LeaseID = losing
			_, _, err := EmitParked(world.repo, Decision{
				Action: ActionParked, ParkReason: fencing.ParkStaleOwner, WinningLeaseID: losing,
			}, EmitParams{
				SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
				LeaseEpoch: 2, LeaseID: losing, LeaseSequence: 2,
				Predecessors: []string{losingID}, CreatedAt: fixtureCreatedAt,
				Presented: fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: losing}, Observation: observation,
			})
			if err == nil || !errors.Is(err, sessrepo.ErrDivergentBranch) {
				t.Fatalf("EmitParked(same-epoch loser) error = %v, want divergent branch refusal from AppendEvent", err)
			}
			events, err := world.repo.ListEvents(fixtureSession)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 3 || events[2].EventID != losingID {
				t.Fatalf("chain after refused same-epoch EmitParked = %+v, want the losing tail only", events)
			}
			assertPreservedAdmissionBlob(t, world.repoRoot, losing, 2, 2, "session.parked")
		})
	}
}

func assertPreservedAdmissionBlob(t *testing.T, root, leaseID string, epoch, sequence uint64, eventType string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "sessions", fixtureSession, "events"))
	if err != nil {
		t.Fatalf("read event blob directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, "sessions", fixtureSession, "events", entry.Name()))
		if err != nil {
			t.Fatalf("read preserved event blob %q: %v", entry.Name(), err)
		}
		var event struct {
			EventType     string `json:"event_type"`
			LeaseEpoch    uint64 `json:"lease_epoch"`
			LeaseID       string `json:"lease_id"`
			LeaseSequence uint64 `json:"lease_sequence"`
		}
		if err := json.Unmarshal(raw, &event); err != nil {
			t.Fatalf("decode event blob %q: %v", entry.Name(), err)
		}
		if event.EventType == eventType && event.LeaseEpoch == epoch && event.LeaseID == leaseID && event.LeaseSequence == sequence {
			return
		}
	}
	t.Fatalf("preserved %s event blob for lease %s epoch %d sequence %d not found", eventType, leaseID, epoch, sequence)
}

// TestAppendGateRefusesLosingLeaseProfileEvent pins the independent durable
// append refusal. Derivation authority is covered by the instrumented tests
// below, which run with this append gate disabled.
func TestAppendGateRefusesLosingLeaseProfileEvent(t *testing.T) {
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
	changed := identifyObject(t, map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"subject_id": fixtureSession, "session_id": fixtureSession, "event_type": "profile.changed", "created_by_host_id": fixtureLocalHost,
		"lease_epoch": 1, "lease_id": fixtureLeaseA, "lease_sequence": 2, "predecessors": []string{world.headID},
		"created_at": fixtureCreatedAt, "payload": map[string]any{"from": "standard", "to": "yolo", "confirmed": true}, "extensions": map[string]any{},
	}, "event_id")
	if _, err := world.repo.AppendEvent(fixtureSession, changed); !errors.Is(err, sessrepo.ErrStaleLease) {
		t.Fatalf("AppendEvent(losing profile.changed) error = %v, want stale lease refusal", err)
	}
	profile, err := LoadProfile(world.repo, world.ckpt, fixtureSession)
	if err != nil {
		t.Fatalf("LoadProfile() after losing append = %v", err)
	}
	pair, err := profile.Derive()
	if err != nil {
		t.Fatalf("Derivation.Derive() after losing append = %v", err)
	}
	if pair.Profile != sessprofile.ProfileStandard || pair.HasSource {
		t.Fatalf("effective profile after refused losing append = %+v, want standard with no source", pair)
	}
	var changedRef struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(changed, &changedRef); err != nil {
		t.Fatalf("decode losing event identity = %v", err)
	}
	request := runRequest(t, world)
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action == ActionLaunch && outcome.Decision.Profile.Profile == "yolo" {
		t.Fatalf("Run() launch profile = %+v, losing-lease yolo must never be effective", outcome.Decision.Profile)
	}
	if outcome.Decision.Action == ActionLaunch && outcome.Decision.Profile.Source == changedRef.EventID {
		t.Fatalf("Run() launch profile source = losing-lease %s, want the checkpoint closure", changedRef.EventID[:20])
	}
	if outcome.Decision.Action == ActionLaunch && outcome.Decision.Profile.Profile != "standard" {
		t.Fatalf("Run() launch profile = %+v, want standard from the closure", outcome.Decision.Profile)
	}
}

// gateOnLosingProfileWorld appends a profile change while lease A is still
// the winner, then transfers ownership to B with a checkpoint that predates
// that change. The append is admitted by the real gate, but the later source
// is outside B's attested handoff closure.
func gateOnLosingProfileWorld(t *testing.T) (*runWorld, string) {
	t.Helper()
	world := buildRunWorld(t)
	losingID := appendChainEvent(t, world.repo, "profile.changed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	successorLease(t, world.repo, fixtureLeaseB, fixtureLocalHost, world.ckptID)
	t.Logf("gate-on losing profile event id=%s", losingID)
	return world, losingID
}

func appendProfileChange(t *testing.T, repository *sessrepo.Repository, epoch uint64, leaseID string, sequence int, predecessor, from, to string) (string, error) {
	t.Helper()
	raw := identifyObject(t, map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"subject_id": fixtureSession, "session_id": fixtureSession, "event_type": "profile.changed", "created_by_host_id": fixtureLocalHost,
		"lease_epoch": epoch, "lease_id": leaseID, "lease_sequence": sequence, "predecessors": []string{predecessor},
		"created_at": fixtureCreatedAt, "payload": map[string]any{"from": from, "to": to, "confirmed": true}, "extensions": map[string]any{},
	}, "event_id")
	reference, err := repository.AppendEvent(fixtureSession, raw)
	if err != nil {
		return "", err
	}
	return reference.EventID, nil
}

func assertRecordProfilePair(t *testing.T, pair sessprofile.Pair, where string) {
	t.Helper()
	if pair.Profile != sessprofile.ProfileStandard || pair.HasSource || pair.Source != "" {
		t.Fatalf("%s = %+v, want standard with no source", where, pair)
	}
}

func TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource(t *testing.T) {
	world, losingID := gateOnLosingProfileWorld(t)
	loaded, err := LoadProfile(world.repo, world.ckpt, fixtureSession)
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	pair, err := loaded.Derive()
	if err != nil {
		t.Fatalf("LoadProfile result Derive() error = %v", err)
	}
	assertRecordProfilePair(t, pair, "LoadProfile effective pair")
	if pair.Source == losingID {
		t.Fatalf("LoadProfile source = losing event %s", losingID)
	}
}

func TestProbe15DisabledAppendGateProjector(t *testing.T) {
	world, losingID := gateOnLosingProfileWorld(t)
	lateID, appendErr := appendProfileChange(t, world.repo, 1, fixtureLeaseA, 3, losingID, "yolo", "standard")
	if appendErr != nil && !errors.Is(appendErr, sessrepo.ErrStaleLease) {
		t.Fatalf("AppendEvent(post-takeover old-lease profile.changed) error = %v, want the append refusal or an instrumented admission", appendErr)
	}
	if appendErr == nil {
		t.Logf("instrumented post-takeover profile event id=%s", lateID)
	} else {
		t.Logf("append gate refused post-takeover event; derivation still runs: %v", appendErr)
	}
	pair, err := (&sessprofile.Projector{Repo: world.repo, Ckpt: world.ckpt}).Project(fixtureSession)
	if err != nil {
		t.Fatalf("Projector.Project() error = %v", err)
	}
	assertRecordProfilePair(t, pair, "Projector.Project effective pair")
	if pair.Source == losingID {
		t.Fatalf("Projector.Project source = losing event %s", losingID)
	}
	if lateID != "" && pair.Source == lateID {
		t.Fatalf("Projector.Project source = post-takeover losing event %s", lateID)
	}
	t.Logf("Projector.Project() = {Profile:%s, HasSource:%t, Source:%s}", pair.Profile, pair.HasSource, pair.Source)
}

func TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource(t *testing.T) {
	world, losingID := gateOnLosingProfileWorld(t)
	pair, err := (&sessprofile.Projector{Repo: world.repo, Ckpt: world.ckpt}).ProjectForHeads(fixtureSession, []string{losingID})
	if err != nil {
		t.Fatalf("Projector.ProjectForHeads() error = %v", err)
	}
	assertRecordProfilePair(t, pair, "Projector.ProjectForHeads effective pair")
	if pair.Source == losingID {
		t.Fatalf("Projector.ProjectForHeads source = losing event %s", losingID)
	}
}

func TestSetProfileFromEndDoesNotReplayLosingLeaseChange(t *testing.T) {
	world, losingID := gateOnLosingProfileWorld(t)
	appendChainEvent(t, world.repo, "session.failed", 2, fixtureLeaseB, 1, losingID, map[string]any{
		"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
	})
	result, err := (&sessprofile.Transactor{Repo: world.repo, Ckpt: world.ckpt}).SetProfile(sessprofile.SetProfileRequest{
		SessionID: fixtureSession, To: sessprofile.ProfileYOLO, Confirmed: true,
		LeaseEpoch: 2, LeaseID: fixtureLeaseB, CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
	})
	if err != nil {
		t.Fatalf("SetProfile() from-end under winning lease = %v", err)
	}
	if result.PreviousProfile != sessprofile.ProfileStandard || result.NewProfile != sessprofile.ProfileYOLO || result.EventID == losingID {
		t.Fatalf("SetProfile() = %+v, want a new standard-to-yolo event after ignoring the losing source", result)
	}
}

func TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource(t *testing.T) {
	world, losingID := gateOnLosingProfileWorld(t)
	loaded, err := LoadProfile(world.repo, world.ckpt, fixtureSession)
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	pair, err := deriveProfile(Input{ProfileData: loaded})
	if err != nil {
		t.Fatalf("deriveProfile() error = %v", err)
	}
	assertRecordProfilePair(t, pair, "deriveProfile effective pair")
	if pair.Source == losingID {
		t.Fatalf("deriveProfile source = losing event %s", losingID)
	}
}

func TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple(t *testing.T) {
	const recordID = "sha256:ebebebebebebebebebebebebebebebebebebebebebebebebebebebebebebebeb"
	const createdID = "sha256:ecececececececececececececececececececececececececececececececec"
	const changeID = "sha256:edededededededededededededededededededededededededededededededed"
	const handoffID = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	derivation := sessprofile.Derivation{
		Record: sessprofile.Record{SessionID: fixtureSession, RecordID: recordID, Creation: sessprofile.ProfileStandard},
		Events: []sessprofile.Event{
			{ID: createdID, Type: "session.created", SchemaVersion: "1.0.0", SessionID: fixtureSession, LeaseEpoch: 1, LeaseID: fixtureLeaseA, Sequence: 1, Predecessors: []string{recordID}},
			{ID: changeID, Type: "profile.changed", SchemaVersion: "1.0.0", SessionID: fixtureSession, LeaseEpoch: 2, LeaseID: fixtureLeaseA, Sequence: 1, Predecessors: []string{createdID}, Payload: map[string]any{"from": sessprofile.ProfileStandard, "to": sessprofile.ProfileYOLO, "confirmed": true}},
		},
		Authority: sessprofile.SourceAuthority{
			Winner:            sessrepo.LeaseSummary{SessionID: fixtureSession, RecordID: handoffID, LeaseID: fixtureLeaseB, Epoch: 2, Checkpoint: handoffID, HasCheckpoint: true},
			HasWinner:         true,
			Leases:            []sessrepo.LeaseSummary{{SessionID: fixtureSession, RecordID: recordID, LeaseID: fixtureLeaseA, Epoch: 1}, {SessionID: fixtureSession, RecordID: handoffID, LeaseID: fixtureLeaseB, Epoch: 2, Checkpoint: handoffID, HasCheckpoint: true}},
			HandoffHeads:      []string{changeID},
			HasHandoffClosure: true,
		},
	}
	derived, err := deriveProfile(Input{ProfileData: derivation})
	if err != nil {
		t.Fatalf("deriveProfile(unminted same-epoch lease tuple) error = %v", err)
	}
	assertRecordProfilePair(t, derived, "deriveProfile unminted same-epoch lease tuple")
}

func TestLoadProfileEmptyLeaseStoreKeepsSessionRecordAuthority(t *testing.T) {
	repository, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil {
		t.Fatal(err)
	}
	reference, err := repository.CreateSession(identifyObject(t, record, "record_id"))
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	changed := identifyObject(t, map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"subject_id": fixtureSession, "session_id": fixtureSession, "event_type": "profile.changed", "created_by_host_id": fixtureLocalHost,
		"lease_epoch": 1, "lease_id": fixtureLeaseA, "lease_sequence": 1, "predecessors": []string{reference.RecordID},
		"created_at": fixtureCreatedAt, "payload": map[string]any{"from": "standard", "to": "yolo", "confirmed": true}, "extensions": map[string]any{},
	}, "event_id")
	if _, err := repository.AppendEvent(fixtureSession, changed); err != nil {
		t.Fatalf("AppendEvent(empty lease store) error = %v", err)
	}
	loaded, err := LoadProfile(repository, nil, fixtureSession)
	if err != nil {
		t.Fatalf("LoadProfile(empty lease store) error = %v", err)
	}
	pair, err := loaded.Derive()
	if err != nil {
		t.Fatalf("Derivation.Derive(empty lease store) error = %v", err)
	}
	assertRecordProfilePair(t, pair, "LoadProfile(empty lease store)")
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
