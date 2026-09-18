package axpane

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Rev4 rework tests: the corrected P1-1 rule (the fold is the sole
// newest-checkpoint authority; a successor lease's checkpoint is the
// handoff base at takeover time and never moves), the P2-1
// post-window replay signal, the P3-1/P3-2 negatives, and the P3-7
// identity session binding. Each test drives a production entry;
// the narrowing-mutant row that attacks its gate is named in
// TRACEABILITY.md.

const (
	rev4Successor = "cccccccc-dddd-4eee-8fff-000000000001"
	rev4Op3       = "0198f4c8-7d40-7e55-8e6f-1234567890ad"
	rev4Op4       = "0198f4c8-7d40-7e55-8e6f-1234567890ae"
	rev4Inst2     = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	rev4Inst3     = "0198f4c9-3333-7ccc-8ccc-1234567890ab"
	rev4Inst4     = "0198f4c9-4444-7ddd-8ddd-1234567890ab"
)

// rev4Capture captures a checkpoint of the fixture session under an
// arbitrary owning (epoch, lease) tuple over the given heads.
func rev4Capture(t *testing.T, repository *sessrepo.Repository, store *sessckpt.Store, operation string, epoch uint64, leaseID string, heads []string) (string, []byte) {
	t.Helper()
	reference, raw, err := store.Capture(repository, sessckpt.Inputs{
		OperationID:         operation,
		SessionID:           fixtureSession,
		SessionKind:         sessckpt.SessionKindDirect,
		LeaseEpoch:          epoch,
		LeaseID:             leaseID,
		CreatorHostID:       fixtureLocalHost,
		WorkspaceManifestID: seedDigest(0xE1),
		ProviderManifestID:  seedDigest(0xE2),
		Boundary: sessckpt.SafeBoundary{
			ProviderID: "codex", ProviderVersion: "0.147.0", Evidence: sessckpt.EvidenceAcceptedTest,
			InputBlocked: true, ForegroundIdle: true, BackgroundIdle: true, OpenProcesses: 0, OpenDatabaseHandles: 0,
		},
		EventHeads: heads,
		CreatedAt:  fixtureCreatedAt,
		Extensions: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Capture(epoch %d) error = %v", epoch, err)
	}
	return reference.CheckpointID, raw
}

// rev4PostTakeoverWorld builds the ordinary post-takeover story:
// the epoch-1 owner stops with checkpoint C0 (published on the
// chain), a LOCAL successor lease (2,S) is minted with handoff base
// C0, the successor owner resumes from C0, goes idle, and stops
// with a NEW checkpoint C1 published under (2,S). The lease
// record's checkpoint stays C0 forever (immutable record); the
// chain fold's newest is C1.
func rev4PostTakeoverWorld(t *testing.T) (*runWorld, string, string) {
	t.Helper()
	world := buildRunWorld(t)
	c0 := world.ckptID
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, c0)
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, c0)
	resumedID := appendChainEvent(t, world.repo, "session.resumed", 2, rev4Successor, 1, world.headID, map[string]any{
		"checkpoint_id": c0, "execution_profile": "standard", "profile_source_event_id": nil,
		"terminal_backend": "tmux", "native_session_id": "native-session-alpha",
	})
	idleID := appendChainEvent(t, world.repo, "session.idle", 2, rev4Successor, 2, resumedID, map[string]any{
		"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true,
	})
	c1, _ := rev4Capture(t, world.repo, world.ckpt, rev4Op3, 2, rev4Successor, []string{idleID})
	world.headID = appendChainEvent(t, world.repo, "session.stopped", 2, rev4Successor, 3, idleID, map[string]any{
		"graceful": true, "checkpoint_id": c1, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true,
	})
	return world, c0, c1
}

func rev4RestoreRequest(t *testing.T, world *runWorld, op, instance string, epoch uint64, lease, ckpt string) Request {
	t.Helper()
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.BootstrapOperationID = op
	request.InstanceID = instance
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: epoch, LeaseID: lease}
	request.MaterializationRequired = true
	request.MaterializationID = fixtureMat
	request.CheckpointRequired = true
	request.CheckpointID = ckpt
	return request
}

// TestRunPostTakeoverRestoreLaunchesWithNewestClosure pins the
// corrected P1-1 positive through Run: in the ordinary
// post-takeover world (successor lease base C0, fold newest C1) a
// restore naming C1 with a journal sourced from C1 launches — and
// carries C1's closure profile, not the handoff base's. The
// successor owner changed the profile to yolo before C1, so the
// launch must carry yolo with that change as its source.
func TestRunPostTakeoverRestoreLaunchesWithNewestClosure(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	c0 := world.ckptID
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, c0)
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, c0)
	resumedID := appendChainEvent(t, world.repo, "session.resumed", 2, rev4Successor, 1, world.headID, map[string]any{
		"checkpoint_id": c0, "execution_profile": "standard", "profile_source_event_id": nil,
		"terminal_backend": "tmux", "native_session_id": "native-session-alpha",
	})
	changed := appendChainEvent(t, world.repo, "profile.changed", 2, rev4Successor, 2, resumedID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	idleID := appendChainEvent(t, world.repo, "session.idle", 2, rev4Successor, 3, changed, map[string]any{
		"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true,
	})
	c1, _ := rev4Capture(t, world.repo, world.ckpt, rev4Op3, 2, rev4Successor, []string{idleID})
	world.headID = appendChainEvent(t, world.repo, "session.stopped", 2, rev4Successor, 4, idleID, map[string]any{
		"graceful": true, "checkpoint_id": c1, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true,
	})
	has, newest, err := LoadNewestCheckpoint(world.repo, fixtureSession)
	if err != nil || !has || newest != c1 {
		t.Fatalf("fold = (%v, %s, %v), want the successor's C1", has, newest, err)
	}
	world.mat = journalSourced(t, c1)
	outcome, err := Run(world.stores(), rev4RestoreRequest(t, world, fixtureOtherOp, rev4Inst2, 2, rev4Successor, c1))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() = (%q, %q, %v), want launch for the post-takeover restore from C1",
			outcome.Decision.Action, outcome.Decision.Class, outcome.Decision.Cause)
	}
	if outcome.Decision.Profile.Profile != "yolo" || outcome.Decision.Profile.Source != changed {
		t.Fatalf("Run() profile = %+v, want the resumed checkpoint's closure (yolo/%s)", outcome.Decision.Profile, changed[:20])
	}
}

// TestRunPostTakeoverStaleBaseParks pins the stale-base negative
// through Run: in the post-takeover world, a restore that names
// the handoff base C0 (superseded by C1) parks at the newest arm,
// never launches.
func TestRunPostTakeoverStaleBaseParks(t *testing.T) {
	t.Parallel()
	world, c0, _ := rev4PostTakeoverWorld(t)
	world.mat = journalSourced(t, c0)
	outcome, err := Run(world.stores(), rev4RestoreRequest(t, world, fixtureOtherOp, rev4Inst2, 2, rev4Successor, c0))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked || outcome.Decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Run() = (%q, %q), want parked restore_policy for the stale handoff base", outcome.Decision.Action, outcome.Decision.ParkReason)
	}
	if outcome.Decision.Cause == nil || !strings.Contains(outcome.Decision.Cause.Error(), "superseded checkpoint") {
		t.Fatalf("Run() cause = %v, want the stale-source arm", outcome.Decision.Cause)
	}
}

// TestRunSuccessorNullFoldParks pins the successor-with-null-fold
// negative through Run: a checkpoint-carrying winner over a chain
// that publishes no checkpoint parks restore_policy with the
// null-fold cause — in launch mode with no requirements at all —
// and installs no binding. A null fold needs no checkpoint store,
// so the same park holds with the store unbound.
func TestRunSuccessorNullFoldParks(t *testing.T) {
	t.Parallel()
	// The session stays unbound: on a bound session with a
	// changed operation the in-window idempotency arm refuses
	// first (pinned by TestRunWindowLeaseCheckpointNullFoldRefuses).
	world := buildRunWorld(t)
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, world.ckptID)
	request := runRequest(t, world)
	request.Mode = ModeLaunch
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked || outcome.Decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Run() = (%q, %q), want parked restore_policy for the successor null fold", outcome.Decision.Action, outcome.Decision.ParkReason)
	}
	if outcome.Decision.Cause == nil || !strings.Contains(outcome.Decision.Cause.Error(), "carries a checkpoint and the chain publishes none") {
		t.Fatalf("Run() cause = %v, want the successor null-fold arm", outcome.Decision.Cause)
	}
	if _, found, err := world.pane.Lookup(fixtureSession, fixtureBootstrap); err != nil || found {
		t.Fatalf("Lookup(op1) = (%v, %v), want no binding for the parked launch", found, err)
	}
	// The same park holds with no checkpoint store bound: a null
	// fold needs no store, so the unknown-store guard stays
	// silent and the parked decision stands.
	without := world.stores()
	without.Ckpt = nil
	parked, err := Run(without, request)
	if err != nil {
		t.Fatalf("Run() without a store error = %v", err)
	}
	if parked.Decision.Action != ActionParked {
		t.Fatalf("Run() without a store = %q, want the same parked decision", parked.Decision.Action)
	}
}

// TestRunPostTakeoverLaunchCarriesNewestClosureStandardToYolo pins
// the standard→yolo wrong-profile plant through Run in launch
// mode (create from stopped, no checkpoint required): the
// successor owner changed the profile to yolo (authoritative,
// under (2,S)) and stopped with C1 whose closure carries that
// change, so the launch must carry yolo with that source — never
// the handoff base C0's closure (standard, no source).
func TestRunPostTakeoverLaunchCarriesNewestClosureStandardToYolo(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	c0 := world.ckptID
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, c0)
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, c0)
	resumedID := appendChainEvent(t, world.repo, "session.resumed", 2, rev4Successor, 1, world.headID, map[string]any{
		"checkpoint_id": c0, "execution_profile": "standard", "profile_source_event_id": nil,
		"terminal_backend": "tmux", "native_session_id": "native-session-alpha",
	})
	changed := appendChainEvent(t, world.repo, "profile.changed", 2, rev4Successor, 2, resumedID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	idleID := appendChainEvent(t, world.repo, "session.idle", 2, rev4Successor, 3, changed, map[string]any{
		"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true,
	})
	c1, _ := rev4Capture(t, world.repo, world.ckpt, rev4Op3, 2, rev4Successor, []string{idleID})
	world.headID = appendChainEvent(t, world.repo, "session.stopped", 2, rev4Successor, 4, idleID, map[string]any{
		"graceful": true, "checkpoint_id": c1, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true,
	})
	has, newest, err := LoadNewestCheckpoint(world.repo, fixtureSession)
	if err != nil || !has || newest != c1 {
		t.Fatalf("fold = (%v, %s, %v), want the successor's C1", has, newest, err)
	}
	request := runRequest(t, world)
	request.Mode = ModeLaunch
	request.BootstrapOperationID = fixtureOtherOp
	request.InstanceID = rev4Inst2
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() = (%q, %q, %v), want launch for create-from-stopped", outcome.Decision.Action, outcome.Decision.Class, outcome.Decision.Cause)
	}
	if outcome.Decision.Profile.Profile != "yolo" || outcome.Decision.Profile.Source != changed {
		t.Fatalf("Run() profile = %+v, want the newest checkpoint's closure (yolo/%s)", outcome.Decision.Profile, changed[:20])
	}
}

// TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard pins
// the dangerous yolo→standard direction through Run in launch
// mode: the epoch-1 owner set yolo before C0, the successor owner
// set the profile BACK to standard (authoritative, under (2,S))
// and stopped with C1. The launch must carry standard with an
// empty mapping — deriving from the handoff base C0 would launch
// yolo with the bypass-approvals mapping on a session whose
// authoritative profile is standard.
func TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	toYolo := appendChainEvent(t, world.repo, "profile.changed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	c0, _ := rev4Capture(t, world.repo, world.ckpt, rev4Op4, 1, fixtureLeaseA, []string{toYolo})
	world.headID = publishCheckpoint(t, world.repo, toYolo, 3, c0)
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, c0)
	resumedID := appendChainEvent(t, world.repo, "session.resumed", 2, rev4Successor, 1, world.headID, map[string]any{
		"checkpoint_id": c0, "execution_profile": "yolo", "profile_source_event_id": toYolo,
		"terminal_backend": "tmux", "native_session_id": "native-session-alpha",
	})
	toStandard := appendChainEvent(t, world.repo, "profile.changed", 2, rev4Successor, 2, resumedID, map[string]any{
		"from": "yolo", "to": "standard", "confirmed": true,
	})
	idleID := appendChainEvent(t, world.repo, "session.idle", 2, rev4Successor, 3, toStandard, map[string]any{
		"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true,
	})
	c1, _ := rev4Capture(t, world.repo, world.ckpt, rev4Op3, 2, rev4Successor, []string{idleID})
	world.headID = appendChainEvent(t, world.repo, "session.stopped", 2, rev4Successor, 4, idleID, map[string]any{
		"graceful": true, "checkpoint_id": c1, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true,
	})
	has, newest, err := LoadNewestCheckpoint(world.repo, fixtureSession)
	if err != nil || !has || newest != c1 {
		t.Fatalf("fold = (%v, %s, %v), want the successor's C1", has, newest, err)
	}
	request := runRequest(t, world)
	request.Mode = ModeLaunch
	request.BootstrapOperationID = fixtureOtherOp
	request.InstanceID = rev4Inst2
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() = (%q, %q, %v), want launch for create-from-stopped", outcome.Decision.Action, outcome.Decision.Class, outcome.Decision.Cause)
	}
	if outcome.Decision.Profile.Profile != "standard" || outcome.Decision.Profile.Source != toStandard {
		t.Fatalf("Run() profile = %+v, want the newest checkpoint's closure (standard/%s)", outcome.Decision.Profile, toStandard[:20])
	}
	if outcome.Decision.Mapping != "" {
		t.Fatalf("Run() mapping = %q, want empty for the authoritative standard profile", outcome.Decision.Mapping)
	}
}

// TestRunPostWindowSamePairRaceReattaches pins P2-1 through Run:
// post-window, a concurrent wrapper records the same new pair
// between stage and link (played through Hooks.AfterStage); the
// loser reattaches to the winner's child (§4.1: an identical retry
// "returns or reattaches to the one recorded wrapper/child"),
// never reports a launch of a child it did not create.
func TestRunPostWindowSamePairRaceReattaches(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	if _, err := Run(world.stores(), runRequest(t, world)); err != nil {
		t.Fatal(err)
	}
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	world.mat = journalSourced(t, world.ckptID)
	winner := bindCandidate()
	winner.OperationID = fixtureOtherOp
	winner.TerminalInstanceID = rev4Inst3
	winner.BindingDigest = BindingDigest(winner)
	request := rev4RestoreRequest(t, world, fixtureOtherOp, rev4Inst2, 1, fixtureLeaseA, world.ckptID)
	request.Hooks = &Hooks{
		AfterStage: func(path string) error {
			inner, err := Open(world.paneRoot)
			if err != nil {
				return err
			}
			_, _, err = inner.Supersede(fixtureSession, fixtureOtherOp, winner)
			return err
		},
	}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Binding == nil || outcome.Binding.TerminalInstanceID != winner.TerminalInstanceID {
		t.Fatalf("Run() binding = %+v, want the winner's recorded receipt", outcome.Binding)
	}
	if outcome.Decision.Action != ActionReattach {
		t.Fatalf("Run() race loser = %q, want reattach to the winner's child", outcome.Decision.Action)
	}
	if outcome.Decision.Detail != "reattach to the one recorded wrapper/child" {
		t.Fatalf("Run() detail = %q, want the reattach detail", outcome.Decision.Detail)
	}
}

// TestRunWindowLeaseCheckpointNullFoldRefuses pins P3-1 through
// Run: the bootstrap window closes on the fold's newest
// checkpoint, never on the lease record's checkpoint. A successor
// lease carrying an unpublished checkpoint over a null fold with
// a bound session and a changed operation refuses
// idempotency_mismatch — the lease's checkpoint cannot close the
// window the fold leaves open.
func TestRunWindowLeaseCheckpointNullFoldRefuses(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	if _, err := Run(world.stores(), runRequest(t, world)); err != nil {
		t.Fatal(err)
	}
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, world.ckptID)
	request := runRequest(t, world)
	request.BootstrapOperationID = fixtureOtherOp
	request.InstanceID = rev4Inst2
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionRefused || outcome.Decision.Class != terminalbackend.CodeIdempotencyMismatch {
		t.Fatalf("Run() = (%q, %q), want refused idempotency_mismatch for the null-fold changed op",
			outcome.Decision.Action, outcome.Decision.Class)
	}
}

// TestRunUnfoldableChainFails pins P3-2 through Run: absence and a
// failure to read are different facts. A chain the landed reducer
// refuses (profile.changed under a successor lease while the
// session is still creating) fails the run with the fold error
// and no decision — never a launch, never a null-fold park.
func TestRunUnfoldableChainFails(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	successorLease(t, world.repo, rev4Successor, fixtureLocalHost, world.ckptID)
	appendChainEvent(t, world.repo, "profile.changed", 2, rev4Successor, 1, world.headID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	if _, _, err := LoadNewestCheckpoint(world.repo, fixtureSession); err == nil {
		t.Fatal("fixture chain unexpectedly folds")
	}
	request := runRequest(t, world)
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	outcome, err := Run(world.stores(), request)
	if err == nil {
		t.Fatalf("Run() over an unfoldable chain decided %q, want the fold error", outcome.Decision.Action)
	}
	if outcome.Decision.Action != "" {
		t.Fatalf("Run() over an unfoldable chain decides %q, want no decision", outcome.Decision.Action)
	}
}

// TestDecideIdentityForeignSessionRefuses pins P3-7: a Provider
// Identity Record minted for another session with the same
// provider and version refuses invalid_config — the wrapper binds
// the record's own session_id to the session under decision,
// exactly as the checkpoint arm binds the checkpoint's.
func TestDecideIdentityForeignSessionRefuses(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	record, err := provhost.CreateIdentity(provhost.IdentityParams{
		SessionID: fixtureForeign, ProviderID: "codex", ProviderVersion: "0.147.0",
		ProviderVersionRange: "opaque-range", NativeSessionID: "native-session-other",
		IdentityKind: "session_uuid", LogicalWorkspaceID: fixtureLocalHost,
		CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Provider.Identity = record
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != ClassInvalidConfig {
		t.Fatalf("Decide() = (%q, %q), want refused invalid_config for the foreign identity record",
			decision.Action, decision.Class)
	}
}

// TestRunNewestAbsentFromStoreFails pins the newest-closure load:
// the chain fold publishes a newest the checkpoint store does not
// carry. The closure is unknown — not "no closure" — so the run
// fails with no decision in both modes instead of deriving the
// session head.
func TestRunNewestAbsentFromStoreFails(t *testing.T) {
	t.Parallel()
	for _, mode := range []Mode{ModeLaunch, ModeRestore} {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()
			world := buildRunWorld(t)
			world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
			stores := world.stores()
			empty, err := sessckpt.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			stores.Ckpt = empty
			request := runRequest(t, world)
			request.Mode = mode
			outcome, err := Run(stores, request)
			if err == nil {
				t.Fatalf("Run(%s) with the newest absent from the store decided %q, want the load error", mode, outcome.Decision.Action)
			}
			if outcome.Decision.Action != "" {
				t.Fatalf("Run(%s) with the newest absent from the store decides %q, want no decision", mode, outcome.Decision.Action)
			}
		})
	}
}
