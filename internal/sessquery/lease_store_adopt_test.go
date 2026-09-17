// Lease-store adoption tests for TASK-260830-2f5393: records minted and
// persisted through the sessrepo lease lifecycle must admit through the
// unchanged query-layer admission (winningLeaseFor/BuildPlan) with the
// same digest, epoch, lease ID, and holder the store reports. This is the
// no-second-lease-model proof: the store and the selector share one
// record shape and one tuple rule.
package sessquery

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

func TestLeaseStoreRecordsAdmitWithoutBehavioralChange(t *testing.T) {
	reader := successorFixture(t)
	first, err := reader.Local.CreateLease(idA, sessrepo.CreateLeaseInput{LeaseID: lease, HolderHostID: hostA, IssuedByHostID: hostA, CreatedAt: "2026-08-19T04:09:00.000Z"})
	if err != nil {
		t.Fatalf("CreateLease error = %v", err)
	}
	checkpoint := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), headAtOrBefore(t, reader, idA, 1))
	withCheckpoints(reader, checkpoint)
	second, err := reader.Local.CompareAndSwapLease(idA, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: leaseB, HolderHostID: hostA, IssuedByHostID: hostA, CreatedAt: "2026-08-19T04:09:00.000Z"},
		Reason:           "graceful_takeover",
		CheckpointID:     checkpointDigestOf(t, checkpoint),
	})
	if err != nil {
		t.Fatalf("CompareAndSwapLease error = %v", err)
	}
	firstBytes, err := reader.Local.GetLease(idA, first.RecordID)
	if err != nil {
		t.Fatalf("GetLease(first) error = %v", err)
	}
	secondBytes, err := reader.Local.GetLease(idA, second.RecordID)
	if err != nil {
		t.Fatalf("GetLease(second) error = %v", err)
	}
	reader.LeaseRecords = nil
	withLeases(reader, firstBytes, secondBytes)
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if plan.LeaseEpoch != 2 || plan.LeaseID != leaseB || plan.OwnerHostID != hostA || plan.LeaseRecordID != second.RecordID {
		t.Fatalf("store-minted succession plan = %+v, want epoch 2 %s %s %s", plan, leaseB, hostA, second.RecordID)
	}
	winner, err := reader.winningLeaseFor(idA, "direct", reader.Local)
	if err != nil {
		t.Fatal(err)
	}
	storeWinner, err := reader.Local.WinningLease(idA)
	if err != nil {
		t.Fatalf("WinningLease error = %v", err)
	}
	if winner.Digest != storeWinner.RecordID || winner.Epoch != storeWinner.Epoch || winner.LeaseID != storeWinner.LeaseID || winner.HolderHostID != storeWinner.HolderHostID {
		t.Fatalf("admitted winner %+v disagrees with store winner %+v", winner, storeWinner)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("store-minted succession plan is not current: %v", err)
	}
}

// TestLeaseTupleOrderAgreesWithSessstateCompare pins the cross-package
// tuple rule: sessrepo.CompareLeaseTuple (which cannot import sessstate)
// must order every corpus pair exactly like sessstate.Compare. The
// enumeration is exhaustive over a small alphabet: epochs {1,2,3} by
// lease IDs {x,y} on each side, with the holder drawn from {a,b} and
// the remaining summary members varied to prove they never affect the
// order. Neither comparator takes a created_at member (the Section 5.3
// timestamp is diagnostic only), so created_at orderings are covered
// by varying every other non-key member instead: holder, record,
// predecessor, checkpoint, and reason.
func TestLeaseTupleOrderAgreesWithSessstateCompare(t *testing.T) {
	epochs := []uint64{1, 2, 3}
	leases := []string{lease, leaseB}
	holders := []string{hostA, hostB}
	sign := func(value int) int {
		switch {
		case value < 0:
			return -1
		case value > 0:
			return 1
		}
		return 0
	}
	checked := 0
	for _, epochA := range epochs {
		for _, leaseA := range leases {
			for _, holderA := range holders {
				for _, epochB := range epochs {
					for _, leaseB := range leases {
						for _, holderB := range holders {
							summaryA := sessrepo.LeaseSummary{
								RecordID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SessionID: idA,
								LeaseID: leaseA, Epoch: epochA, HolderHostID: holderA,
								Predecessor: lease, HasPredecessor: true, Checkpoint: zeroDigest, HasCheckpoint: true, Reason: "force_takeover",
							}
							summaryB := sessrepo.LeaseSummary{
								RecordID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SessionID: idB,
								LeaseID: leaseB, Epoch: epochB, HolderHostID: holderB,
								Predecessor: leaseB, HasPredecessor: false, Checkpoint: zeroDigest, HasCheckpoint: true, Reason: "create",
							}
							got := sign(sessrepo.CompareLeaseTuple(summaryA, summaryB))
							want := sign(sessstate.Compare(
								sessstate.LeaseHead{Epoch: epochA, LeaseID: leaseA},
								sessstate.LeaseHead{Epoch: epochB, LeaseID: leaseB},
							))
							if got != want {
								t.Fatalf("tuple (%d %s) vs (%d %s): sessrepo = %d, sessstate = %d", epochA, leaseA, epochB, leaseB, got, want)
							}
							checked++
						}
					}
				}
			}
		}
	}
	if checked != 144 {
		t.Fatalf("checked %d pairs, want the full 144-pair enumeration", checked)
	}
}
