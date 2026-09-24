// Complete-authority union tests for TASK-260830-21gygk: revalidation
// and construction validate the pinned session UUID across the local
// and every allowlisted source without re-resolving names or
// substituting selections.
package sessquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

func TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation(t *testing.T) {
	leaseA := leaseRecordWithCreatedAt(t, lease, hostA, "2026-08-19T04:09:00.000Z")
	leaseBBytes := leaseRecordWithCreatedAt(t, leaseB, hostB, "2026-08-20T04:09:00.000Z")
	reader := &Reader{LeaseRecords: [][]byte{leaseBBytes, leaseA}}
	got, err := reader.LeaseHeadsForSession(idA)
	if err != nil {
		t.Fatalf("LeaseHeadsForSession(first timestamp assignment) = %v", err)
	}
	want := []sessstate.LeaseHead{
		{Epoch: 1, LeaseID: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"},
		{Epoch: 1, LeaseID: "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LeaseHeadsForSession() = %#v, want both literal §5.3 tuples %#v", got, want)
	}

	// Reverse the timestamps and input order. The same complete tuple set must
	// reach Reduce; created_at is diagnostic only under pinned §5.3.
	leaseA = leaseRecordWithCreatedAt(t, lease, hostA, "2099-12-31T23:59:59.999Z")
	leaseBBytes = leaseRecordWithCreatedAt(t, leaseB, hostB, "1900-01-01T00:00:00.000Z")
	reader = &Reader{LeaseRecords: [][]byte{leaseA, leaseBBytes}}
	got, err = reader.LeaseHeadsForSession(idA)
	if err != nil {
		t.Fatalf("LeaseHeadsForSession(perturbed timestamps) = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LeaseHeadsForSession(perturbed timestamps) = %#v, want unchanged tuples %#v", got, want)
	}
}

func TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID(t *testing.T) {
	first := leaseRecordWithCreatedAt(t, lease, hostA, "2026-08-19T04:09:00.000Z")
	second := leaseRecordWithCreatedAt(t, lease, hostA, "2026-08-20T04:09:00.000Z")
	reader := &Reader{LeaseRecords: [][]byte{first, second}}
	got, err := reader.LeaseHeadsForSession(idA)
	if len(got) != 0 || !errors.Is(err, sessstate.ErrIntegrity) || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("LeaseHeadsForSession(conflicting bytes for one lease ID) = (%#v, %v), want literal integrity_failure", got, err)
	}
}

func TestLeaseHeadsForSessionCoversGeneratedCardinalityRange(t *testing.T) {
	for size := 0; size <= 32; size++ {
		t.Run(fmt.Sprintf("size_%02d", size), func(t *testing.T) {
			firstOrder := make([][]byte, size)
			secondOrder := make([][]byte, size)
			wantIDs := make([]string, size)
			for index := 0; index < size; index++ {
				leaseID := fmt.Sprintf("00000000-0000-4000-8000-%012x", index+1)
				wantIDs[index] = leaseID
				firstTime, secondTime := "1900-01-01T00:00:00.000Z", "2099-12-31T23:59:59.999Z"
				if index%2 == 1 {
					firstTime, secondTime = secondTime, firstTime
				}
				firstOrder[index] = leaseRecordWithCreatedAt(t, leaseID, hostA, firstTime)
				secondOrder[size-index-1] = leaseRecordWithCreatedAt(t, leaseID, hostA, secondTime)
			}
			sort.Strings(wantIDs)
			want := make([]sessstate.LeaseHead, size)
			for index, leaseID := range wantIDs {
				want[index] = sessstate.LeaseHead{Epoch: 1, LeaseID: leaseID}
			}
			for _, records := range [][][]byte{firstOrder, secondOrder} {
				reader := &Reader{LeaseRecords: records}
				got, err := reader.LeaseHeadsForSession(idA)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("LeaseHeadsForSession(%d generated records) = %#v/%v, want independently sorted tuples %#v", size, got, err, want)
				}
			}
		})
	}
}

func leaseRecordWithCreatedAt(t *testing.T, leaseID, holder, createdAt string) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(leaseCreate(t, idA, holder), &object); err != nil {
		t.Fatal(err)
	}
	object["record_id"] = zeroDigest
	object["lease_id"] = leaseID
	object["created_at"] = createdAt
	return identity(t, object, "record_id")
}

// leasedPeerSession creates a leased session on a peer repository for
// union fixtures.
func leasedPeerSession(t *testing.T, peer *sessrepo.Repository, id, name string) sessrepo.SessionRef {
	t.Helper()
	ref := create(t, peer, id, name)
	appendEvent(t, peer, id, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	return ref
}

// unionReader builds a local/peer reader whose peer holds a leased
// session and whose local side starts empty. The reader carries the
// winning Lease Record for the peer session.
func unionReader(t *testing.T, local, peer *sessrepo.Repository) *Reader {
	t.Helper()
	reader := &Reader{
		Local:              local,
		LocalHostID:        hostB,
		AllowlistedPeerIDs: []string{hostA},
		Peers:              []PeerIndex{{hostA, peer}},
		Aliases:            map[string]string{hostA: "workstation"},
	}
	withLeases(reader, leaseCreate(t, idB, hostA))
	return reader
}

func TestRevalidateUnionLeaseDivergenceRefusesStale(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	leasedPeerSession(t, peer, idB, "beta")
	reader := unionReader(t, local, peer)
	plan := mustBuild(t, reader, PlanArgs{Selector: "id:" + idB, Action: ActionStatus})
	if err := reader.Revalidate(plan); err != nil {
		t.Fatal(err)
	}
	// The local source learns the same immutable record, then observes
	// a successor winning lease the plan source has not seen.
	ref := create(t, local, idB, "beta")
	if ref.RecordID != plan.SessionRecordID {
		t.Fatalf("fixture records diverged: %q vs %q", ref.RecordID, plan.SessionRecordID)
	}
	created := appendEvent(t, local, idB, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idB, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	appendEventAs(t, local, idB, launched, "lease.transferred", 1, 2, leaseB, map[string]any{
		"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA,
		"predecessor_lease_id": lease, "new_lease_id": leaseB,
	})
	mustStale(t, reader.Revalidate(plan), "winning lease changed")
}

func TestRevalidateUnionTombstoneRefusesStale(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	leasedPeerSession(t, peer, idB, "beta")
	reader := unionReader(t, local, peer)
	plan := mustBuild(t, reader, PlanArgs{Selector: "id:" + idB, Action: ActionStatus})
	ref := create(t, local, idB, "beta")
	if ref.RecordID != plan.SessionRecordID {
		t.Fatalf("fixture records diverged: %q vs %q", ref.RecordID, plan.SessionRecordID)
	}
	event := appendEvent(t, local, idB, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	event = appendEvent(t, local, idB, event, "session.failed", 2, map[string]any{"error_code": "E_LAUNCH", "retryable": true, "operation_id": nil})
	appendEvent(t, local, idB, event, "session.tombstoned", 3, map[string]any{"tombstone_id": zeroDigest})
	mustStale(t, reader.Revalidate(plan), "tombstone evidence changed")
}

func TestRevalidateUnionAgreeingReplicaStaysCurrent(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	ref := leasedPeerSession(t, peer, idB, "beta")
	reader := unionReader(t, local, peer)
	plan := mustBuild(t, reader, PlanArgs{Selector: "id:" + idB, Action: ActionStatus})
	// A lagging source replicating the same record and the identical
	// bootstrap event observes agreeing authority: no refusal.
	localRef := create(t, local, idB, "beta")
	if localRef.RecordID != ref.RecordID {
		t.Fatalf("fixture records diverged: %q vs %q", localRef.RecordID, ref.RecordID)
	}
	appendEvent(t, local, idB, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("agreeing replica invalidated plan: %v", err)
	}
}

// TestRevalidateUnionParkedCopyRefuses replaces the earlier
// parked-copy-ignored expectation: a parked copy of the pinned
// session in a required source is incomplete authority (unknown),
// never evidence and never absence. Being unable to prove a
// contradiction does not prove currency, so both build and
// revalidation fail closed instead of skipping the parked row.
func TestRevalidateUnionParkedCopyRefuses(t *testing.T) {
	parkedReader := func(t *testing.T, localName string) (*Reader, SelectionPlan) {
		t.Helper()
		local, _ := repository(t)
		peer, _ := repository(t)
		leasedPeerSession(t, peer, idB, "beta")
		reader := unionReader(t, local, peer)
		plan := mustBuild(t, reader, PlanArgs{Selector: "id:" + idB, Action: ActionStatus})
		local.AfterCreateStep = func(step sessrepo.CreateStep) error {
			if step == sessrepo.CreateStepRecord {
				return errors.New("crash")
			}
			return nil
		}
		if _, err := local.CreateSession(record(t, idB, localName)); err == nil {
			t.Fatal("fault did not fire")
		}
		local.AfterCreateStep = nil
		rows, err := reader.InspectLocal(idB)
		if err != nil {
			t.Fatal(err)
		}
		if rows.Projection.Parked == nil {
			t.Fatal("premise: required copy must be parked")
		}
		return reader, plan
	}
	t.Run("revalidate divergent", func(t *testing.T) {
		reader, plan := parkedReader(t, "different-record")
		if err := reader.Revalidate(plan); !errors.Is(err, ErrObservationUnavailable) {
			t.Fatalf("Revalidate over parked authority = %v, want selector_observation_unavailable", err)
		}
	})
	t.Run("build divergent", func(t *testing.T) {
		reader, _ := parkedReader(t, "different-record")
		if _, err := reader.BuildPlan(PlanArgs{Selector: "id:" + idB, Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
			t.Fatalf("BuildPlan over parked authority = %v, want selector_observation_unavailable", err)
		}
	})
	// A parked copy with an agreeing record isolates the parked
	// gate from the record-agreement gate: no other union check
	// can refuse it, so only the parked refusal keeps the plan
	// from validating over unknown authority.
	t.Run("revalidate agreeing", func(t *testing.T) {
		reader, plan := parkedReader(t, "beta")
		inspected, err := reader.InspectLocal(idB)
		if err != nil {
			t.Fatal(err)
		}
		if inspected.Projection.RecordID != plan.SessionRecordID {
			t.Fatalf("premise: parked record %q must agree with plan %q", inspected.Projection.RecordID, plan.SessionRecordID)
		}
		if err := reader.Revalidate(plan); !errors.Is(err, ErrObservationUnavailable) {
			t.Fatalf("Revalidate over agreeing parked authority = %v, want selector_observation_unavailable", err)
		}
	})
	t.Run("build agreeing", func(t *testing.T) {
		reader, _ := parkedReader(t, "beta")
		if _, err := reader.BuildPlan(PlanArgs{Selector: "id:" + idB, Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
			t.Fatalf("BuildPlan over agreeing parked authority = %v, want selector_observation_unavailable", err)
		}
	})
}

func TestRevalidateUnionUnreadablePeerRefuses(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	peer, peerRoot := repository(t)
	reader.AllowlistedPeerIDs = []string{hostC}
	reader.Peers = []PeerIndex{{hostC, peer}}
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if err := os.Rename(filepath.Join(peerRoot, "sessions"), filepath.Join(peerRoot, "saved")); err != nil {
		t.Fatal(err)
	}
	// The complete authority union cannot be validated without the
	// allowlisted peer read: the failure keeps its class.
	err := reader.Revalidate(plan)
	if !errors.Is(err, ErrSourceReadFailed) || !errors.Is(err, sessrepo.ErrRepositoryPath) {
		t.Fatalf("unreadable union peer = %v", err)
	}
}

func TestBuildPlanValidatesUnionAtBuild(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	leasedPeerSession(t, peer, idB, "beta")
	create(t, local, idB, "different-record")
	reader := unionReader(t, local, peer)
	// The explicit peer selection never falls back, and the build
	// still validates the pinned UUID union: the contradictory local
	// record is integrity failure at construction.
	if _, err := reader.BuildPlan(PlanArgs{Selector: "beta@peer:workstation", Action: ActionStatus}); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("explicit build over divergent union = %v", err)
	}
	// The mirror direction behaves the same: a leased local selection
	// against a divergent peer copy refuses without substituting it.
	localOnly, _ := repository(t)
	localRef := create(t, localOnly, idB, "beta")
	appendEvent(t, localOnly, idB, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	divergent, _ := repository(t)
	create(t, divergent, idB, "different-record")
	mirror := &Reader{
		Local:              localOnly,
		LocalHostID:        hostB,
		AllowlistedPeerIDs: []string{hostA},
		Peers:              []PeerIndex{{hostA, divergent}},
	}
	withLeases(mirror, leaseCreate(t, idB, hostA))
	if _, err := mirror.BuildPlan(PlanArgs{Selector: "beta@local", Action: ActionStatus}); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("local build over divergent union = %v", err)
	}
}

func TestRevalidateUnionNameDriftStaysCurrent(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	leasedPeerSession(t, peer, idB, "beta")
	reader := unionReader(t, local, peer)
	plan := mustBuild(t, reader, PlanArgs{Selector: "beta", Action: ActionStatus})
	if plan.SourceHostID != hostA {
		t.Fatalf("source: %+v", plan)
	}
	// A different UUID gaining the same live name locally moves fresh
	// name resolution but never retargets the pinned plan: names are
	// not re-resolved and the drift alone is not a defect.
	drift := create(t, local, idC, "beta")
	appendEvent(t, local, idC, drift.RecordID, "session.created", 1, map[string]any{"session_record_id": drift.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("same-name gain invalidated pinned plan: %v", err)
	}
}

// TestBuildWithLaggingLeaseCopy proves the union winner is the
// greatest (epoch, lease_id) tuple, not copy equality: a lagging
// epoch-1 replica beside an epoch-2 local is valid and a fresh build
// on the local winner succeeds, retaining the losing history.
func TestBuildWithLaggingLeaseCopy(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	// Peer holds the epoch-1 replica with the same immutable record.
	peerRef := create(t, peer, idB, "beta")
	appendEvent(t, peer, idB, peerRef.RecordID, "session.created", 1, map[string]any{"session_record_id": peerRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	// Local holds the same record at epoch 2 with a successor lease.
	localRef := create(t, local, idB, "beta")
	if localRef.RecordID != peerRef.RecordID {
		// Records with the same name and session ID share the digest;
		// a divergence here is a fixture error, not a union case.
		t.Fatalf("fixture records diverged: %q vs %q", localRef.RecordID, peerRef.RecordID)
	}
	created := appendEvent(t, local, idB, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idB, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	appendEventAs(t, local, idB, launched, "lease.transferred", 1, 2, leaseB, map[string]any{
		"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA,
		"predecessor_lease_id": lease, "new_lease_id": leaseB,
	})
	reader := &Reader{
		Local:              local,
		LocalHostID:        hostB,
		AllowlistedPeerIDs: []string{hostA},
		Peers:              []PeerIndex{{hostA, peer}},
		Aliases:            map[string]string{hostA: "workstation"},
	}
	// The epoch-2 winner carries complete validated succession: the
	// epoch-1 predecessor record plus the Checkpoint Record taken
	// under that predecessor which the successor references.
	withSuccessor(t, reader, idB, hostA, leaseB)
	plan := mustBuild(t, reader, PlanArgs{Selector: "beta@local", Action: ActionStatus})
	if plan.LeaseEpoch != 2 || plan.LeaseID != leaseB {
		t.Fatalf("lagging union winner: %+v", plan)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("lagging replica invalidated winner plan: %v", err)
	}
	// The union still refuses a greater lease appearing elsewhere: a
	// same-epoch successor on the peer with a greater lease ID is stale.
	peerCreated, err := chainTail(peer, idB)
	if err != nil {
		t.Fatal(err)
	}
	peerLaunched := appendEvent(t, peer, idB, peerCreated, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	leaseC := "cccccccc-dddd-4eee-8fff-111111111111"
	appendEventAs(t, peer, idB, peerLaunched, "lease.transferred", 1, 2, leaseC, map[string]any{
		"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA,
		"predecessor_lease_id": lease, "new_lease_id": leaseC,
	})
	mustStale(t, reader.Revalidate(plan), "winning lease changed")
}
