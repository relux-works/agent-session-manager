// CR revision 4 independent-review regression probes for
// TASK-260830-21gygk, carried verbatim from the review bundle
// (reviewer_rev4_test.go) into the candidate: predecessor legality,
// checkpoint admission, parked authority, closed capability names,
// plus the two passing live-record/digest controls.
package sessquery

import (
	"encoding/json"
	"errors"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"strings"
	"testing"
)

func rev4Successor(t *testing.T) *Reader {
	r, _, _, created := plannedSession(t)
	launched := appendEvent(t, r.Local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	appendEventAs(t, r.Local, idA, launched, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	return r
}
func TestRev4SelfPredecessorMustRefuse(t *testing.T) {
	r := rev4Successor(t)
	withLeases(r, leaseRecordBytes(t, idA, 2, leaseB, hostA, leaseB, true))
	p, err := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus})
	if err == nil {
		t.Fatalf("BuildPlan accepted epoch-2 self predecessor; Revalidate=%v lease=%s", r.Revalidate(p), p.LeaseRecordID)
	}
}
func TestRev4MissingCheckpointMustRefuse(t *testing.T) {
	r := rev4Successor(t)
	withLeases(r, leaseRecordBytes(t, idA, 2, leaseB, hostA, lease, true))
	p, err := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus})
	if err == nil {
		t.Fatalf("BuildPlan accepted successor with placeholder checkpoint and no validated checkpoint supplied; Revalidate=%v lease=%s", r.Revalidate(p), p.LeaseRecordID)
	}
}
func TestRev4ParkedAuthorityMustRefuse(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	leasedPeerSession(t, peer, idB, "beta")
	r := unionReader(t, local, peer)
	p := mustBuild(t, r, PlanArgs{Selector: "id:" + idB, Action: ActionStatus})
	local.AfterCreateStep = func(step sessrepo.CreateStep) error {
		if step == sessrepo.CreateStepRecord {
			return errors.New("interrupted creation")
		}
		return nil
	}
	if _, err := local.CreateSession(record(t, idB, "conflicting-record")); err == nil {
		t.Fatal("fault did not fire")
	}
	local.AfterCreateStep = nil
	rows, err := r.InspectLocal(idB)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Projection.Parked == nil {
		t.Fatal("premise: copy must be parked")
	}
	t.Logf("required source parked: %+v", rows.Projection.Parked)
	if err := r.Revalidate(p); err == nil {
		t.Fatal("Revalidate accepted while required same-UUID authority is unknown/parked")
	}
}
func TestRev4UnknownCapabilityMustRefuse(t *testing.T) {
	r := summaryReader(t)
	o := r.Observations[idA]
	o.Capabilities = map[string]CapabilitySummary{"invented_capability": {Status: "available", Enabled: true}}
	r.Observations[idA] = o
	for _, list := range []bool{false, true} {
		if list {
			if rows, err := r.AuthoritativeList(); err == nil {
				t.Errorf("AuthoritativeList advertises unsupported name: %+v", rows[0].Capabilities)
			}
		} else {
			if row, err := r.AuthoritativeStatus("alpha"); err == nil {
				t.Errorf("AuthoritativeStatus advertises unsupported name: %+v", row.Capabilities)
			}
		}
	}
}
func TestRev4ChangedLiveRecordRefuses(t *testing.T) {
	r, _, _, _ := plannedSession(t)
	p := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
	replacement, _ := repository(t)
	ref := create(t, replacement, idA, "renamed")
	appendEvent(t, replacement, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	r.Local = replacement
	s, err := r.InspectLocal(idA)
	if err != nil {
		t.Fatal(err)
	}
	if s.Projection.Parked != nil || s.Projection.RecordID == "" || s.Projection.RecordID == p.SessionRecordID {
		t.Fatal("invalid live changed-record witness")
	}
	err = r.Revalidate(p)
	if !errors.Is(err, ErrPlanStale) {
		t.Fatalf("changed live record = %v", err)
	}
	t.Logf("live record change refused: %v", err)
}
func TestRev4DifferentLeaseDigestRefuses(t *testing.T) {
	r, _, _, _ := plannedSession(t)
	p := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
	var value map[string]any
	if err := json.Unmarshal(r.LeaseRecords[0], &value); err != nil {
		t.Fatal(err)
	}
	value["created_at"] = "2026-08-19T04:10:00.000Z"
	r.LeaseRecords = [][]byte{identity(t, value, "record_id")}
	if err := r.Revalidate(p); !errors.Is(err, ErrPlanStale) || !strings.Contains(err.Error(), "winning lease") {
		t.Fatalf("changed canonical lease identity = %v", err)
	}
}
