package axpane

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Rev3 rework tests: one root-cause class (P1-1, the fold's newest
// checkpoint versus the lease record's checkpoint), two P2
// (receipt retention, unknown checkpoint store), and the P3
// hardening rows. Each test drives a production entry; the
// narrowing-mutant row that attacks its gate is named in
// TRACEABILITY.md.

// TestDecideWindowEpochOneOwnerCloses pins P1-1 (window): the
// bootstrap window closes on the fold's newest checkpoint, not on
// the lease record's checkpoint. An epoch-1 winner (null lease
// checkpoint) with a published newest launches a post-window new
// operation. Before the fix (Winner.HasCheckpoint) this refused
// idempotency_mismatch on the creating host.
func TestDecideWindowEpochOneOwnerCloses(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.BootstrapOperationID = fixtureOtherOp
	input.SessionBound = true
	if input.Observation.Winner.HasCheckpoint {
		t.Fatal("fixture winner unexpectedly carries a checkpoint")
	}
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = deps.ckptID
	input.MaterializationRequired = true
	input.JournalOK = true
	input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() = (%q, %q, %v), want launch for the epoch-1 owner past its first checkpoint", decision.Action, decision.Class, decision.Cause)
	}
}

// TestDecideCheckpointNullFoldParks pins P1-1 (admission): with no
// published checkpoint, a required checkpoint parks restore_policy
// — even a self-consistent checkpoint of this session. The cause
// pins the null-fold arm (killer for N-checkpoint-nullfold).
func TestDecideCheckpointNullFoldParks(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Decide() = (%q, %q), want parked restore_policy with no published checkpoint", decision.Action, decision.ParkReason)
	}
	if decision.Cause == nil || !strings.Contains(decision.Cause.Error(), "resume requires a published checkpoint and the chain names none") {
		t.Fatalf("Decide() cause = %v, want the null-fold arm", decision.Cause)
	}
}

// TestDecideCheckpointUnpublishedParks pins P1-1 (admission): with
// the chain's newest at C, a second attested checkpoint C2 of this
// session that the chain never published parks restore_policy,
// never launches. Before the fix (null lease checkpoint admits
// anything) this launched.
func TestDecideCheckpointUnpublishedParks(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	idleID := appendChainEvent(t, deps.repo, "session.idle", 1, fixtureLeaseA, 2, deps.createdID, map[string]any{
		"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true,
	})
	c2ID, c2Doc := captureUnpublished(t, deps.repo, deps.ckpt, fixtureOtherOp, []string{idleID})
	if c2ID == deps.ckptID {
		t.Fatal("fixture produced the published checkpoint, want an unpublished one")
	}
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = deps.ckptID
	input.CheckpointRequired = true
	input.CheckpointDoc = c2Doc
	input.CheckpointID = c2ID
	decision := Decide(input)
	if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Decide() = (%q, %q), want parked restore_policy for the unpublished checkpoint", decision.Action, decision.ParkReason)
	}
	if decision.Cause == nil || !strings.Contains(decision.Cause.Error(), "not the newest published checkpoint") {
		t.Fatalf("Decide() cause = %v, want the newest arm", decision.Cause)
	}
}

// TestDecidePostTakeoverDivergedBaseLaunches pins the corrected
// P1-1 rule through the pure core: the successor lease's handoff
// base (C0) diverged from the fold's newest (C1, published by the
// successor owner afterwards) is the ordinary post-takeover state,
// not a refusal — a restore naming the newest with a journal
// sourced from it launches. The strict lease/fold equality this
// test replaced parked every session that was ever taken over.
func TestDecidePostTakeoverDivergedBaseLaunches(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	input.Observation.Winner.Epoch = 2
	input.Observation.Winner.LeaseID = rev4Successor
	input.Observation.Winner.HasCheckpoint = true
	input.Observation.Winner.Checkpoint = seedDigest(0xC0)
	input.Observation.Grant.Token = sessrepo.FencingToken{Epoch: 2, LeaseID: rev4Successor, HolderHostID: fixtureLocalHost}
	input.HasNewestCheckpoint = true
	input.NewestCheckpointID = deps.ckptID
	input.MaterializationRequired = true
	input.JournalOK = true
	input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
	input.CheckpointRequired = true
	input.CheckpointDoc = deps.ckptDoc
	input.CheckpointID = deps.ckptID
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() = (%q, %q, %v), want launch for the post-takeover restore from the newest", decision.Action, decision.Class, decision.Cause)
	}
	if decision.Profile.Profile != "standard" || decision.Profile.HasSource {
		t.Fatalf("Decide() profile = %+v, want the newest checkpoint's closure pair", decision.Profile)
	}
}

// TestDecideSuccessorNullFoldParks pins the corrected P1-1 rule:
// the only fact a successor lease adds is the implication "a
// checkpoint-carrying winner means the fold published a newest",
// asserted at one site every launch-class path passes. A
// checkpoint-carrying winner over a null fold parks
// restore_policy in both modes, with or without a required
// checkpoint or journal; the cause pins the null-fold arm.
func TestDecideSuccessorNullFoldParks(t *testing.T) {
	t.Parallel()
	t.Run("launch_no_requirements", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeLaunch
		input.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
		input.Observation.Winner.Epoch = 2
		input.Observation.Winner.LeaseID = rev4Successor
		input.Observation.Winner.HasCheckpoint = true
		input.Observation.Winner.Checkpoint = seedDigest(0xC0)
		input.Observation.Grant.Token = sessrepo.FencingToken{Epoch: 2, LeaseID: rev4Successor, HolderHostID: fixtureLocalHost}
		decision := Decide(input)
		if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
			t.Fatalf("Decide() = (%q, %q), want parked restore_policy for the successor null fold", decision.Action, decision.ParkReason)
		}
		if decision.Cause == nil || !strings.Contains(decision.Cause.Error(), "carries a checkpoint and the chain publishes none") {
			t.Fatalf("Decide() cause = %v, want the successor null-fold arm", decision.Cause)
		}
	})
	t.Run("restore_with_requirements", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
		input.Observation.Winner.Epoch = 2
		input.Observation.Winner.LeaseID = rev4Successor
		input.Observation.Winner.HasCheckpoint = true
		input.Observation.Winner.Checkpoint = seedDigest(0xC0)
		input.Observation.Grant.Token = sessrepo.FencingToken{Epoch: 2, LeaseID: rev4Successor, HolderHostID: fixtureLocalHost}
		input.MaterializationRequired = true
		input.JournalOK = true
		input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
		input.CheckpointRequired = true
		input.CheckpointDoc = deps.ckptDoc
		input.CheckpointID = deps.ckptID
		decision := Decide(input)
		if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
			t.Fatalf("Decide() = (%q, %q), want parked restore_policy for the successor null fold", decision.Action, decision.ParkReason)
		}
		if decision.Cause == nil || !strings.Contains(decision.Cause.Error(), "carries a checkpoint and the chain publishes none") {
			t.Fatalf("Decide() cause = %v, want the successor null-fold arm owning the required path", decision.Cause)
		}
	})
}

// TestDecideJournalNullFoldParks pins P1-1 (journal): with no
// published checkpoint, a committed journal sourced from any
// digest parks restore_policy. The cause pins the null-fold arm
// (killer for N-materialization-nullfold).
func TestDecideJournalNullFoldParks(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Mode = ModeRestore
	input.MaterializationRequired = true
	input.JournalOK = true
	input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: seedDigest(0xD2), MaterializationID: fixtureMat}
	decision := Decide(input)
	if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Decide() = (%q, %q), want parked restore_policy with no published checkpoint", decision.Action, decision.ParkReason)
	}
	if decision.Cause == nil || !strings.Contains(decision.Cause.Error(), "required materialization names no published checkpoint to bind") {
		t.Fatalf("Decide() cause = %v, want the null-fold arm", decision.Cause)
	}
}

// TestRunEpochOneOwnerAfterCheckpoint pins P1-1 through Run: the
// ordinary same-host story — created, idle, stopped with
// checkpoint C on the epoch-1 lease — restores with a new
// bootstrap operation and launches; an unpublished C2 and a
// journal sourced from an unnamed digest park instead of
// launching.
func TestRunEpochOneOwnerAfterCheckpoint(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	first, err := Run(world.stores(), runRequest(t, world))
	if err != nil || first.Decision.Action != ActionLaunch {
		t.Fatalf("first launch = (%v, %v)", first.Decision.Action, err)
	}
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	has, newest, err := LoadNewestCheckpoint(world.repo, fixtureSession)
	if err != nil || !has || newest != world.ckptID {
		t.Fatalf("LoadNewestCheckpoint() = (%v, %q, %v), want the published checkpoint", has, newest, err)
	}
	observation, err := ObserveOwnership(world.repo, fixtureSession, runObserve(fixtureNow()))
	if err != nil {
		t.Fatal(err)
	}
	if observation.Winner.Epoch != 1 || observation.Winner.HasCheckpoint {
		t.Fatalf("winner = %+v, want the epoch-1 lease with a null checkpoint", observation.Winner)
	}
	world.mat = journalSourced(t, world.ckptID)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.BootstrapOperationID = fixtureOtherOp
	request.InstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	request.MaterializationRequired = true
	request.MaterializationID = fixtureMat
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() action = %q, class %q, cause %v, want launch for the epoch-1 post-checkpoint restore",
			outcome.Decision.Action, outcome.Decision.Class, outcome.Decision.Cause)
	}
	if outcome.Binding == nil || outcome.Binding.OperationID != fixtureOtherOp {
		t.Fatalf("Run() binding = %+v, want the op2 pair receipt", outcome.Binding)
	}

	// The same state with an unpublished C2 required: parks.
	c2ID, _ := captureUnpublished(t, world.repo, world.ckpt, "0198f4c8-7d40-7e55-8e6f-1234567890ad", []string{world.headID})
	op3 := "0198f4c8-7d40-7e55-8e6f-1234567890ad"
	unpublished := runRequest(t, world)
	unpublished.Mode = ModeRestore
	unpublished.BootstrapOperationID = op3
	unpublished.MaterializationRequired = true
	unpublished.MaterializationID = fixtureMat
	unpublished.CheckpointRequired = true
	unpublished.CheckpointID = c2ID
	parked, err := Run(world.stores(), unpublished)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if parked.Decision.Action != ActionParked {
		t.Fatalf("Run() action = %q, want parked for the unpublished checkpoint", parked.Decision.Action)
	}
	if _, found, err := world.pane.Lookup(fixtureSession, op3); err != nil || found {
		t.Fatalf("Lookup(op3) = (%v, %v), want no binding for the parked retry", found, err)
	}

	// The same state with a journal sourced from a digest no lease
	// and no event names: parks.
	world.mat = journalSourced(t, seedDigest(0xD2))
	nameless := runRequest(t, world)
	nameless.Mode = ModeRestore
	nameless.BootstrapOperationID = op3
	nameless.MaterializationRequired = true
	nameless.MaterializationID = fixtureMat
	nameless.CheckpointRequired = true
	nameless.CheckpointID = world.ckptID
	stale, err := Run(world.stores(), nameless)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stale.Decision.Action != ActionParked {
		t.Fatalf("Run() action = %q, want parked for the unnamed-digest journal", stale.Decision.Action)
	}
}

// TestRunNullFoldRestoreParks pins P1-1 through Run: a restore with
// a stored checkpoint and a committed journal parks when the chain
// publishes no checkpoint — captured objects are never relabeled
// as authoritative (§13.1), and no binding installs.
func TestRunNullFoldRestoreParks(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.MaterializationRequired = true
	request.MaterializationID = fixtureMat
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked || outcome.Decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Run() = (%q, %q), want parked restore_policy with no published checkpoint",
			outcome.Decision.Action, outcome.Decision.ParkReason)
	}
	if outcome.Binding != nil {
		t.Fatalf("Run() binding = %+v, want no binding for the parked restore", outcome.Binding)
	}
	if _, found, err := world.pane.Lookup(fixtureSession, fixtureBootstrap); err != nil || found {
		t.Fatalf("Lookup() = (%v, %v), want no binding for the parked restore", found, err)
	}
}

// TestSupersedeKeepsReceipts pins P2-1: post-window pairs record
// alongside — never over — their superseded receipts. An
// identical retry of any recorded pair replays its own receipt,
// and Status keeps pointing at the first binding.
func TestSupersedeKeepsReceipts(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	first, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil {
		t.Fatal(err)
	}
	op3 := "0198f4c8-7d40-7e55-8e6f-1234567890ad"
	second := bindCandidate()
	second.OperationID = fixtureOtherOp
	second.TerminalInstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	second.BindingDigest = BindingDigest(second)
	third := bindCandidate()
	third.OperationID = op3
	third.TerminalInstanceID = "0198f4c9-3333-7ccc-8ccc-1234567890ab"
	third.BindingDigest = BindingDigest(third)
	boundSecond, replayedSecond, err := store.Supersede(fixtureSession, fixtureOtherOp, second)
	if err != nil {
		t.Fatalf("Supersede(op2) error = %v", err)
	}
	if replayedSecond {
		t.Fatal("Supersede(op2) fresh install reports replayed")
	}
	boundThird, replayedThird, err := store.Supersede(fixtureSession, op3, third)
	if err != nil {
		t.Fatalf("Supersede(op3) error = %v", err)
	}
	if replayedThird {
		t.Fatal("Supersede(op3) fresh install reports replayed")
	}
	for _, want := range []Binding{first, boundSecond, boundThird} {
		replayed, found, err := store.Lookup(fixtureSession, want.OperationID)
		if err != nil || !found || replayed != want {
			t.Fatalf("Lookup(%q) = (%+v, %v, %v), want the recorded %+v", want.OperationID, replayed, found, err, want)
		}
	}
	anchor, found, err := store.Status(fixtureSession)
	if err != nil || !found || anchor != first {
		t.Fatalf("Status() = (%+v, %v, %v), want the first-binding anchor", anchor, found, err)
	}
	bound, err := store.SessionBound(fixtureSession)
	if err != nil || !bound {
		t.Fatalf("SessionBound() = (%v, %v), want true", bound, err)
	}
}

// TestSupersedeConcurrentDifferentOps pins P2-1: the post-window
// install is no-replace per pair, so two Supersede calls with
// different operations each record their own child.
func TestSupersedeConcurrentDifferentOps(t *testing.T) {
	t.Parallel()
	store := openPaneStore(t)
	if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatal(err)
	}
	op3 := "0198f4c8-7d40-7e55-8e6f-1234567890ad"
	second := bindCandidate()
	second.OperationID = fixtureOtherOp
	second.TerminalInstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	second.BindingDigest = BindingDigest(second)
	third := bindCandidate()
	third.OperationID = op3
	third.TerminalInstanceID = "0198f4c9-3333-7ccc-8ccc-1234567890ab"
	third.BindingDigest = BindingDigest(third)
	boundSecond, _, errSecond := store.Supersede(fixtureSession, fixtureOtherOp, second)
	boundThird, _, errThird := store.Supersede(fixtureSession, op3, third)
	if errSecond != nil || errThird != nil {
		t.Fatalf("Supersede(op2, op3) = (%v, %v), want both recorded", errSecond, errThird)
	}
	if boundSecond.TerminalInstanceID != second.TerminalInstanceID || boundThird.TerminalInstanceID != third.TerminalInstanceID {
		t.Fatalf("Supersede receipts = (%+v, %+v), want one child per pair", boundSecond, boundThird)
	}
}

// TestSupersedeConcurrentSamePairReplays pins P2-1: a second
// wrapper recording the same pair between Lookup and link wins,
// and the loser replays the winner's receipt instead of replacing
// it. The AfterStage hook plays the concurrent winner
// deterministically (killer for N-supersede-no-replace).
func TestSupersedeConcurrentSamePairReplays(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outer, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := outer.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatal(err)
	}
	winner := bindCandidate()
	winner.OperationID = fixtureOtherOp
	winner.TerminalInstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	winner.BindingDigest = BindingDigest(winner)
	outer.WithHooks(&Hooks{
		AfterStage: func(path string) error {
			inner, err := Open(root)
			if err != nil {
				return err
			}
			if _, _, err := inner.Supersede(fixtureSession, fixtureOtherOp, winner); err != nil {
				return err
			}
			return nil
		},
	})
	loser := bindCandidate()
	loser.OperationID = fixtureOtherOp
	loser.TerminalInstanceID = "0198f4c9-4444-7ddd-8ddd-1234567890ab"
	loser.BindingDigest = BindingDigest(loser)
	bound, replayed, err := outer.Supersede(fixtureSession, fixtureOtherOp, loser)
	if err != nil {
		t.Fatalf("Supersede() race loser error = %v", err)
	}
	if !replayed {
		t.Fatal("Supersede() race loser reports a fresh install, want the replay signal")
	}
	if bound.TerminalInstanceID != winner.TerminalInstanceID {
		t.Fatalf("Supersede() race loser = %+v, want the winner's receipt", bound)
	}
}

// TestSupersedeCrashSeams pins P2-1 crash convergence: an aborted
// stage records nothing and keeps the anchor, while a post-commit
// hook error keeps the recorded pair receipt the retry replays.
func TestSupersedeCrashSeams(t *testing.T) {
	t.Parallel()
	t.Run("stage", func(t *testing.T) {
		t.Parallel()
		store := openPaneStore(t)
		first, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
		if err != nil {
			t.Fatal(err)
		}
		store.WithHooks(&Hooks{
			AfterStage: func(path string) error { return errFixtureConfig },
		})
		second := bindCandidate()
		second.OperationID = fixtureOtherOp
		if _, _, err := store.Supersede(fixtureSession, fixtureOtherOp, second); err == nil {
			t.Fatal("Supersede() over aborted stage succeeded, want the hook error")
		}
		if _, found, err := store.Lookup(fixtureSession, fixtureOtherOp); err != nil || found {
			t.Fatalf("Lookup(op2) = (%v, %v), want absence after the aborted stage", found, err)
		}
		anchor, found, err := store.Status(fixtureSession)
		if err != nil || !found || anchor != first {
			t.Fatalf("Status() = (%+v, %v, %v), want the intact anchor", anchor, found, err)
		}
	})
	t.Run("install", func(t *testing.T) {
		t.Parallel()
		store := openPaneStore(t)
		if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
			t.Fatal(err)
		}
		store.WithHooks(&Hooks{
			AfterInstall: func(path string) error { return errFixtureConfig },
		})
		second := bindCandidate()
		second.OperationID = fixtureOtherOp
		if _, _, err := store.Supersede(fixtureSession, fixtureOtherOp, second); err == nil {
			t.Fatal("Supersede() over post-commit hook succeeded, want the hook error")
		}
		committed, found, err := store.Lookup(fixtureSession, fixtureOtherOp)
		if err != nil || !found {
			t.Fatalf("Lookup(op2) = (%v, %v), want the committed pair receipt", found, err)
		}
		store.WithHooks(nil)
		replayed, replayedFlag, err := store.Supersede(fixtureSession, fixtureOtherOp, second)
		if err != nil || replayed != committed {
			t.Fatalf("Supersede() retry = (%+v, %v), want the recorded %+v", replayed, err, committed)
		}
		if !replayedFlag {
			t.Fatal("Supersede() identical retry reports a fresh install, want the replay signal")
		}
	})
}

// TestRunSupersededPairIdenticalRetry pins P2-1 through Run (§4.1):
// after op2 and op3 each recorded post-window, an identical retry
// of op2 reattaches to op2's own child instead of launching a
// third one.
func TestRunSupersededPairIdenticalRetry(t *testing.T) {
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
	run := func(op, instance string) Outcome {
		request := runRequest(t, world)
		request.Mode = ModeRestore
		request.BootstrapOperationID = op
		request.InstanceID = instance
		request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: successor}
		request.MaterializationRequired = true
		request.MaterializationID = fixtureMat
		request.CheckpointRequired = true
		request.CheckpointID = world.ckptID
		outcome, err := Run(world.stores(), request)
		if err != nil {
			t.Fatalf("Run(%s) error = %v", op, err)
		}
		return outcome
	}
	op3 := "0198f4c8-7d40-7e55-8e6f-1234567890ad"
	second := run(fixtureOtherOp, "0198f4c9-2222-7bbb-8bbb-1234567890ab")
	if second.Decision.Action != ActionLaunch {
		t.Fatalf("Run(op2) action = %q, want launch", second.Decision.Action)
	}
	third := run(op3, "0198f4c9-3333-7ccc-8ccc-1234567890ab")
	if third.Decision.Action != ActionLaunch {
		t.Fatalf("Run(op3) action = %q, want launch", third.Decision.Action)
	}
	retry := run(fixtureOtherOp, "0198f4c9-4444-7ddd-8ddd-1234567890ab")
	if retry.Decision.Action != ActionReattach {
		t.Fatalf("Run(op2 retry) action = %q, want reattach to op2's own child", retry.Decision.Action)
	}
	if retry.Binding == nil || retry.Binding.TerminalInstanceID != second.Binding.TerminalInstanceID {
		t.Fatalf("Run(op2 retry) binding = %+v, want op2's recorded %+v", retry.Binding, second.Binding)
	}
}

// TestLookupSessionMismatchRefuses pins the pair receipt's
// binding-to-directory checks: a receipt read under another
// session, or under another operation, refuses.
func TestLookupSessionMismatchRefuses(t *testing.T) {
	t.Parallel()
	t.Run("session", func(t *testing.T) {
		t.Parallel()
		store := openPaneStore(t)
		if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(store.receiptPath(fixtureSession, fixtureBootstrap))
		if err != nil {
			t.Fatal(err)
		}
		foreignReceipts := store.receiptsDir(fixtureForeign)
		if err := os.MkdirAll(foreignReceipts, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(foreignReceipts, fixtureBootstrap+".json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, found, err := store.Lookup(fixtureForeign, fixtureBootstrap); err == nil || found {
			t.Fatalf("Lookup(foreign) = (%v, %v), want the session-mismatch error", found, err)
		}
	})
	t.Run("operation", func(t *testing.T) {
		t.Parallel()
		store := openPaneStore(t)
		if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(store.receiptPath(fixtureSession, fixtureBootstrap))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store.receiptsDir(fixtureSession), fixtureOtherOp+".json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, found, err := store.Lookup(fixtureSession, fixtureOtherOp); err == nil || found {
			t.Fatalf("Lookup(other op) = (%v, %v), want the operation-mismatch error", found, err)
		}
	})
}

// TestRunNoCkptStorePublishedNewestFails pins the corrected
// unknown-store guard: the fold's newest checkpoint drives the
// resume profile closure, so a published newest with no checkpoint
// store bound fails the run — an absent store is unknown, not "no
// closure". With the store bound, the same launch carries the
// newest closure pair, never the head pair. (A null fold needs no
// store at all: the successor-null-fold negative pins that Run
// parks there instead of failing.)
func TestRunNoCkptStorePublishedNewestFails(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	successor := "cccccccc-dddd-4eee-8fff-000000000001"
	successorLease(t, world.repo, successor, fixtureLocalHost, world.ckptID)
	// A head-only authoritative profile change under the
	// successor, outside the newest checkpoint's closure.
	resumedID := appendChainEvent(t, world.repo, "session.resumed", 2, successor, 1, world.headID, map[string]any{
		"checkpoint_id": world.ckptID, "execution_profile": "standard", "profile_source_event_id": nil,
		"terminal_backend": "tmux", "native_session_id": "native-session-alpha",
	})
	changed := appendChainEvent(t, world.repo, "profile.changed", 2, successor, 2, resumedID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	if _, _, err := LoadNewestCheckpoint(world.repo, fixtureSession); err != nil {
		t.Fatalf("chain must fold: %v", err)
	}
	request := runRequest(t, world)
	request.Mode = ModeLaunch
	request.BootstrapOperationID = fixtureOtherOp
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: successor}
	stores := world.stores()
	stores.Ckpt = nil
	outcome, err := Run(stores, request)
	if err == nil {
		t.Fatalf("Run() without a checkpoint store succeeded with %+v, want the store error", outcome.Decision)
	}
	if outcome.Decision.Action != "" {
		t.Fatalf("Run() without a checkpoint store decides %q, want no decision", outcome.Decision.Action)
	}
	// Control: with the store bound the launch carries the
	// newest checkpoint's closure, never the head pair.
	control, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() control error = %v", err)
	}
	if control.Decision.Action != ActionLaunch {
		t.Fatalf("Run() control action = %q, want launch", control.Decision.Action)
	}
	if control.Decision.Profile.Profile != "standard" || control.Decision.Profile.HasSource {
		t.Fatalf("Run() control profile = %+v, want standard with no source from the closure", control.Decision.Profile)
	}
	if control.Decision.Profile.Source == changed {
		t.Fatalf("Run() control profile source = head %s, want the checkpoint closure", changed[:20])
	}
	// Bound: a null fold needs no store — an epoch-1 winner on a
	// checkpoint-free launch path still launches without one.
	fresh := buildRunWorld(t)
	freshStores := fresh.stores()
	freshStores.Ckpt = nil
	plain, err := Run(freshStores, runRequest(t, fresh))
	if err != nil || plain.Decision.Action != ActionLaunch {
		t.Fatalf("Run() epoch-1 without a store = (%q, %v), want launch", plain.Decision.Action, err)
	}
}

// TestRunCreateFromStoppedPostWindow pins P3-2 through Run:
// launch-mode create from a stopped, checkpoint-published session
// with a changed bootstrap operation launches with its own pair
// receipt.
func TestRunCreateFromStoppedPostWindow(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	if _, err := Run(world.stores(), runRequest(t, world)); err != nil {
		t.Fatal(err)
	}
	successor := "cccccccc-dddd-4eee-8fff-000000000001"
	successorLease(t, world.repo, successor, fixtureLocalHost, world.ckptID)
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	request := runRequest(t, world)
	request.Mode = ModeLaunch
	request.BootstrapOperationID = fixtureOtherOp
	request.InstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: successor}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() action = %q, class %q, cause %v, want launch for create-from-stopped",
			outcome.Decision.Action, outcome.Decision.Class, outcome.Decision.Cause)
	}
	if outcome.Binding == nil || outcome.Binding.OperationID != fixtureOtherOp {
		t.Fatalf("Run() binding = %+v, want the op2 pair receipt", outcome.Binding)
	}
}

// TestDecideForegroundCredentialRequiresRealm pins P3-4: the §4.C
// credential conditional applies to every caller, so a foreground
// caller on a credential-requiring path authorizes only through
// the admitted realm row — a passing smoke record plus
// caller-supplied target claims never suffice alone.
func TestDecideForegroundCredentialRequiresRealm(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Realm.Caller = CallerForeground
	input.Smoke.Required = true
	input.Smoke.Record = smokeRecord(t, "pass", "A", passingSmokeChecks(), smokeTuple())
	input.Smoke.Target = smokeTarget()
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
		t.Fatalf("Decide() = (%q, %q), want refused capability_unavailable for foreground without the realm row",
			decision.Action, decision.Class)
	}
	if decision.RealmFailure == nil {
		t.Fatal("Decide() capability_unavailable carries no typed failure")
	}
}

// TestDecideRealmEvidenceOrderIndependent pins P3-5: the realm
// cross-bind accepts when any admitted realm object binds this
// host and build, so the outcome never depends on evidence order.
func TestDecideRealmEvidenceOrderIndependent(t *testing.T) {
	t.Parallel()
	launch := func(t *testing.T, swap bool) Decision {
		t.Helper()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB2), "codex", "0.147.0")
		addSecondRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
		if swap {
			last := len(deps.universe.evidence) - 1
			deps.universe.evidence[last-1], deps.universe.evidence[last] = deps.universe.evidence[last], deps.universe.evidence[last-1]
		}
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		return Decide(input)
	}
	t.Run("foreign_first", func(t *testing.T) {
		t.Parallel()
		decision := launch(t, false)
		if decision.Action != ActionLaunch {
			t.Fatalf("Decide() = (%q, %q, %v), want launch with the binding object second",
				decision.Action, decision.Class, decision.Cause)
		}
	})
	t.Run("binding_first", func(t *testing.T) {
		t.Parallel()
		decision := launch(t, true)
		if decision.Action != ActionLaunch {
			t.Fatalf("Decide() = (%q, %q, %v), want launch with the binding object first",
				decision.Action, decision.Class, decision.Cause)
		}
	})
	t.Run("none_binds", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB2), "codex", "0.147.0")
		addSecondRealmEvidence(t, deps.universe, seedDigest(0xB3), "codex", "0.147.0")
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
			t.Fatalf("Decide() = (%q, %q), want refused when no realm object binds", decision.Action, decision.Class)
		}
	})
}

// TestDecideExpiredRealmRowRefusesUnavailable pins P3-6: a
// background caller whose only realm evidence expired at the
// admission instant refuses the §4.2 capability_unavailable with
// typed realm/readiness details — not the reconcile class — with
// the liveness failure as the cause.
func TestDecideExpiredRealmRowRefusesUnavailable(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
	deps.universe.now = deps.universe.now.AddDate(2, 0, 0)
	input := validInput(t, deps)
	input.Realm.Caller = CallerBackground
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
		t.Fatalf("Decide() = (%q, %q, %v), want refused capability_unavailable for the expired realm row",
			decision.Action, decision.Class, decision.Cause)
	}
	if decision.RealmFailure == nil {
		t.Fatal("Decide() capability_unavailable carries no typed failure")
	}
	// The admission-branch detail pins the dead-realm mapping
	// (not the no-row arm): the reconcile liveness failure is
	// the typed refusal's cause.
	if decision.Detail != "caller has no usable realm evidence" {
		t.Fatalf("Decide() detail = %q, want the dead-realm mapping", decision.Detail)
	}
	inner := errors.Unwrap(decision.Cause)
	if inner == nil || !strings.Contains(inner.Error(), "evidence liveness") {
		t.Fatalf("Decide() cause chain = %v, want the reconcile liveness failure", decision.Cause)
	}
}

// TestDecidePreRebootRealmRowRefusesUnavailable pins P3-6: a
// background caller whose realm evidence binds a superseded
// generation (the raw generation moved) refuses the §4.2
// capability_unavailable with typed details, not
// terminal_backend_stale_generation.
func TestDecidePreRebootRealmRowRefusesUnavailable(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
	input := validInput(t, deps)
	input.Realm.Caller = CallerBackground
	input.Realm.ServerGeneration = "generation-beta"
	input.Backend.RawGeneration = "generation-beta"
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
		t.Fatalf("Decide() = (%q, %q, %v), want refused capability_unavailable for the pre-reboot realm row",
			decision.Action, decision.Class, decision.Cause)
	}
	if decision.RealmFailure == nil {
		t.Fatal("Decide() capability_unavailable carries no typed failure")
	}
}

// TestDecideLiveRealmRowBackendFaultStands bounds P3-6: with a
// usable realm row submitted, an unrelated backend fault keeps
// its own class instead of wearing capability_unavailable.
func TestDecideLiveRealmRowBackendFaultStands(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
	input := validInput(t, deps)
	input.Realm.Caller = CallerBackground
	// Protocol drift is unrelated to the realm evidence: the
	// backend mismatch stands.
	deps.universe.probe["protocol_version"] = "2.0.0"
	deps.universe.probe["probe_id"] = omitSelfIdentity(t, deps.universe.probe, "probe_id")
	input.Backend = deps.universe.facts(t)
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != terminalbackend.CodeMismatch {
		t.Fatalf("Decide() = (%q, %q, %v), want refused manifest_probe_mismatch with a usable realm row",
			decision.Action, decision.Class, decision.Cause)
	}
}
