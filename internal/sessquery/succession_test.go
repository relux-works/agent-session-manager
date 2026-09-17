// Winning-lease succession tests for TASK-260830-21gygk (SPEC 5.3,
// 14.7.2): the greatest (epoch, lease_id) winner must chain by epoch
// plus one to an epoch-1 root and reference a validated checkpoint
// for its session and predecessor lease. All cases drive the shared
// production entries (BuildPlan/Revalidate/AuthoritativeStatus),
// never the validators directly.
package sessquery

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// leaseCycleB is a third fencing token for cycle fixtures.
const leaseCycleB = "dddddddd-eeee-4fff-8000-222222222222"

// danglingLeaseID names no admitted record.
const danglingLeaseID = "aaaaaaaa-1111-4111-8111-111111111111"

// successorFixture builds a local session with the epoch-1 to
// epoch-2 transfer event chain, ready for successor lease fixtures.
func successorFixture(t *testing.T) *Reader {
	t.Helper()
	reader, _, _, created := plannedSession(t)
	launched := appendEvent(t, reader.Local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	appendEventAs(t, reader.Local, idA, launched, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	return reader
}

// leaseEpochOneWithCheckpoint builds an epoch-1 non-create lease
// with no predecessor and a non-null checkpoint reference.
func leaseEpochOneWithCheckpoint(t *testing.T, sessionID, leaseID, holder, checkpoint string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "lease_id": leaseID, "epoch": json.Number(uint64String(1)),
		"holder_host_id": holder, "predecessor_lease_id": nil, "reason": "recovery",
		"checkpoint_id": checkpoint, "issued_by_host_id": holder, "created_by_host_id": holder,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
	return identity(t, value, "record_id")
}

func TestValidSuccessorBuildsAndRevalidates(t *testing.T) {
	reader := successorFixture(t)
	checkpoint := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), headAtOrBefore(t, reader, idA, 1))
	withCheckpoints(reader, checkpoint)
	successor := leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, checkpoint))
	withLeases(reader, successor)
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if plan.LeaseEpoch != 2 || plan.LeaseID != leaseB || plan.OwnerHostID != hostA {
		t.Fatalf("successor triple: %+v", plan)
	}
	winner, err := reader.winningLeaseFor(idA, "direct", reader.Local)
	if err != nil {
		t.Fatal(err)
	}
	if plan.LeaseRecordID != winner.Digest {
		t.Fatalf("lease_record_id = %q, want successor %q", plan.LeaseRecordID, winner.Digest)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("valid successor plan is not current: %v", err)
	}
}

func TestSuccessorCycleMustRefuse(t *testing.T) {
	reader := successorFixture(t)
	firstCheckpoint := checkpointRecordBytes(t, idA, 1, lease, hostA)
	secondCheckpoint := checkpointRecordBytes(t, idA, 2, leaseCycleB, hostA)
	withCheckpoints(reader, firstCheckpoint, secondCheckpoint)
	// Epoch-3 winner names the epoch-2 record, which names the
	// winner back: epochs cannot decrease around the cycle, so the
	// second link refuses.
	cycle := leaseSuccessorBytes(t, idA, 2, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, firstCheckpoint))
	winner := leaseSuccessorBytes(t, idA, 3, leaseB, hostA, leaseCycleB, checkpointDigestOf(t, secondCheckpoint))
	withLeases(reader, cycle, winner)
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("cyclic succession build = %v, want integrity_failure", err)
	}
}

func TestSkippedEpochMustRefuse(t *testing.T) {
	reader := successorFixture(t)
	checkpoint := checkpointRecordBytes(t, idA, 1, lease, hostA)
	withCheckpoints(reader, checkpoint)
	// Epoch 3 directly names the epoch-1 predecessor, skipping
	// epoch 2: the takeover did not use max_observed_epoch + 1.
	skipped := leaseSuccessorBytes(t, idA, 3, leaseB, hostA, lease, checkpointDigestOf(t, checkpoint))
	withLeases(reader, skipped)
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("skipped-epoch succession build = %v, want integrity_failure", err)
	}
}

func TestEpochOneWithPredecessorMustRefuse(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	reader.LeaseRecords = nil
	// An epoch-1 lease cannot name a predecessor: no epoch 0
	// exists, so the link is malformed authority.
	epochOne := leaseRecordBytes(t, idA, 1, lease, hostA, leaseB, true)
	withLeases(reader, epochOne)
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("epoch-1 with predecessor build = %v, want integrity_failure", err)
	}
}

// TestSelfPredecessorWithCheckpointMustRefuse isolates the chain
// gate from the checkpoint gate: the self-predecessor lease carries
// a fully valid checkpoint bound to its own tuple (including a real
// event head at or before its lease), so only the
// predecessor-legality check can refuse. A mutant admitting the
// self link must fail this test by building successfully.
func TestSelfPredecessorWithCheckpointMustRefuse(t *testing.T) {
	reader := successorFixture(t)
	checkpoint := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 2, leaseB, hostA), headAtOrBefore(t, reader, idA, 2))
	withCheckpoints(reader, checkpoint)
	withLeases(reader, leaseSuccessorBytes(t, idA, 2, leaseB, hostA, leaseB, checkpointDigestOf(t, checkpoint)))
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("self predecessor with valid checkpoint build = %v, want integrity_failure", err)
	}
}

func TestDanglingPredecessorMustRefuse(t *testing.T) {
	reader := successorFixture(t)
	dangling := leaseRecordBytes(t, idA, 2, leaseB, hostA, danglingLeaseID, true)
	withLeases(reader, dangling)
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("dangling predecessor build = %v, want selector_observation_unavailable", err)
	}
}

func TestWrongSessionCheckpointMustRefuse(t *testing.T) {
	reader := successorFixture(t)
	// The referenced checkpoint is validated but carries another
	// session: the winner's reference cannot be established.
	foreign := checkpointRecordBytes(t, idB, 1, lease, hostA)
	withCheckpoints(reader, foreign)
	successor := leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, foreign))
	withLeases(reader, successor)
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("wrong-session checkpoint build = %v, want selector_observation_unavailable", err)
	}
}

func TestWrongPredecessorCheckpointMustRefuse(t *testing.T) {
	reader := successorFixture(t)
	// The referenced checkpoint is validated for this session but
	// bound to another lease, not the winner's predecessor.
	misbound := checkpointRecordBytes(t, idA, 1, leaseCycleB, hostA)
	withCheckpoints(reader, misbound)
	successor := leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, misbound))
	withLeases(reader, successor)
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("wrong-predecessor checkpoint build = %v, want selector_observation_unavailable", err)
	}
}

func TestMalformedCheckpointBytesRefuseConfig(t *testing.T) {
	reader := successorFixture(t)
	checkpoint := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), headAtOrBefore(t, reader, idA, 1))
	withCheckpoints(reader, checkpoint)
	successor := leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, checkpoint))
	withLeases(reader, successor)
	reader.CheckpointRecords = append(reader.CheckpointRecords, []byte(`{broken`))
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("malformed checkpoint build = %v, want invalid_config", err)
	}
}

func TestEpochOneWithSelfBoundCheckpointBuilds(t *testing.T) {
	reader, _, _, created := plannedSession(t)
	reader.LeaseRecords = nil
	// An epoch-1 lease past the first provider boundary references
	// a validated checkpoint taken under itself: the only lease a
	// root can name. The head names the real bootstrap event.
	checkpoint := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), created)
	withCheckpoints(reader, checkpoint)
	withLeases(reader, leaseEpochOneWithCheckpoint(t, idA, lease, hostA, checkpointDigestOf(t, checkpoint)))
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if plan.LeaseEpoch != 1 || plan.LeaseID != lease {
		t.Fatalf("epoch-1 checkpointed triple: %+v", plan)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("epoch-1 checkpointed plan is not current: %v", err)
	}
}
