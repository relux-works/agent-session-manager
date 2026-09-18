package axpane

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
)

// Story-close witnesses for the independent review's residual P3
// findings (rev4 verdict): the journal-only restore closure source, the
// parked payload winner binding, and the corrupt checkpoint blob. Each
// drives the pinned behavior through the production entry the finding
// names; the mutant harness carries one narrowing row per gate.

// TestRunJournalOnlyRestoreClosureFromNewest closes P3-1: the yolo to
// standard post-takeover world (C0 closure yolo, successor set standard
// before C1), then a restore with the journal required (sourced from C1)
// and NO required checkpoint. The launch must carry the fold newest's
// closure (standard), never the lease handoff base (yolo with the bypass
// mapping).
func TestRunJournalOnlyRestoreClosureFromNewest(t *testing.T) {
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
	world.mat = journalSourced(t, c1)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.BootstrapOperationID = fixtureOtherOp
	request.InstanceID = rev4Inst2
	request.Presented = fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: rev4Successor}
	request.MaterializationRequired = true
	request.MaterializationID = fixtureMat
	request.CheckpointRequired = false
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("action = %q, want launch: %v", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Decision.Profile.Profile != "standard" || outcome.Decision.Profile.Source != toStandard {
		t.Fatalf("journal-only restore carries %+v mapping %q instead of the newest checkpoint's closure (standard)", outcome.Decision.Profile, outcome.Decision.Mapping)
	}
}

// TestEmitParkedWinningLeaseMismatchRefuses closes P3-4: a parked
// decision whose winning lease differs from the authoring lease refuses
// instead of appending a payload that names another winner.
func TestEmitParkedWinningLeaseMismatchRefuses(t *testing.T) {
	repository, _ := chainFixture(t)
	events, err := repository.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	head := events[len(events)-1].EventID
	head = appendChainEvent(t, repository, "session.failed", 1, fixtureLeaseA, 2, head, map[string]any{
		"error_code": "provider_process_failed", "retryable": true, "operation_id": nil,
	})
	observation, err := ObserveOwnership(repository, fixtureSession, runObserve(fixtureNow()))
	if err != nil {
		t.Fatal(err)
	}
	before := eventCount(t, repository)
	decision := parked(fencing.ParkStaleOwner, fixtureLeaseB, errors.New("probe"))
	_, _, err = EmitParked(repository, decision, EmitParams{
		SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: 3,
		Predecessors: []string{head}, CreatedAt: fixtureCreatedAt,
		Presented:   fencing.PresentedToken{SessionID: fixtureSession, Epoch: 1, LeaseID: fixtureLeaseA},
		Observation: observation,
	})
	if err == nil {
		t.Fatal("EmitParked(payload winner B, authoring lease A) succeeded, want refusal")
	}
	if eventCount(t, repository) != before {
		t.Fatal("mismatched parked emission appended an event, want none")
	}
}

// TestRunCorruptCheckpointBlobIsNotAbsence closes P3-5: a stored
// checkpoint blob corrupted in place (attestation failure, not ENOENT)
// fails the run with the load error in both modes, never a decision and
// never a report of absence.
func TestRunCorruptCheckpointBlobIsNotAbsence(t *testing.T) {
	world := buildRunWorld(t)
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	world.mat = journalSourced(t, world.ckptID)
	ckptRoot := t.TempDir()
	corrupt, err := sessckpt.Open(ckptRoot)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := world.ckpt.Get(world.ckptID)
	if err != nil {
		t.Fatal(err)
	}
	hex := strings.TrimPrefix(world.ckptID, "sha256:")
	if err := os.WriteFile(filepath.Join(ckptRoot, "checkpoints", hex+".json"), append([]byte("{\"corrupt\":1,"), raw[1:]...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := corrupt.Get(world.ckptID); err == nil || os.IsNotExist(err) {
		t.Fatalf("corrupt blob Get() = %v, want a non-ENOENT failure", err)
	}
	stores := world.stores()
	stores.Ckpt = corrupt
	for _, mode := range []Mode{ModeRestore, ModeLaunch} {
		request := runRequest(t, world)
		request.Mode = mode
		if mode == ModeRestore {
			request.MaterializationRequired = true
			request.MaterializationID = fixtureMat
			request.CheckpointRequired = true
			request.CheckpointID = world.ckptID
		}
		outcome, err := Run(stores, request)
		if err == nil {
			t.Fatalf("%s over a corrupt blob decided %q, want the load error", mode, outcome.Decision.Action)
		}
		if strings.Contains(err.Error(), "absent") {
			t.Fatalf("%s over a corrupt blob reported absence: %v", mode, err)
		}
	}
}
