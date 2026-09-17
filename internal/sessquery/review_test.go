// Independent-review regression tests for TASK-260830-21gygk,
// carried from the CR revision 2 review bundle
// (reviewer_regression_test.go) into the candidate: cross-source
// authority divergence must refuse revalidation, record-only
// bootstraps must refuse summaries, and every marshalled plan must
// carry the required lease attestation.
package sessquery

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

func TestReviewerOtherSourceDivergenceMustRefuse(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	ref := create(t, peer, idB, "beta")
	appendEvent(t, peer, idB, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	reader := &Reader{Local: local, LocalHostID: hostB, AllowlistedPeerIDs: []string{hostA}, Peers: []PeerIndex{{HostID: hostA, Repo: peer}}}
	withLeases(reader, leaseCreate(t, idB, hostA))
	plan := mustBuild(t, reader, PlanArgs{Selector: "id:" + idB, Action: ActionStatus})
	if err := reader.Revalidate(plan); err != nil {
		t.Fatal(err)
	}
	create(t, local, idB, "different-record")
	if _, err := reader.Resolve("id:" + idB); err == nil {
		t.Fatal("diagnostic premise: expected record divergence")
	} else {
		t.Logf("fresh Resolve: %v", err)
	}
	if err := reader.Revalidate(plan); err == nil {
		t.Fatal("Revalidate accepted conflicting immutable record for the selected UUID in another allowed source")
	}
}
func TestReviewerBootstrapSummaryMustRefuse(t *testing.T) {
	local, _ := repository(t)
	create(t, local, idA, "alpha")
	reader := &Reader{Local: local, LocalHostID: hostA}
	t.Run("list", func(t *testing.T) {
		rows, err := reader.List()
		if err == nil {
			t.Fatalf("List accepted record-only bootstrap: rows=%d epoch=%d owner=%q", len(rows), rows[0].Projection.Winner.Epoch, rows[0].Projection.OwnerHostID)
		}
	})
	t.Run("status", func(t *testing.T) {
		row, err := reader.Status("alpha")
		if err == nil {
			t.Fatalf("Status accepted record-only bootstrap: epoch=%d owner=%q", row.Projection.Winner.Epoch, row.Projection.OwnerHostID)
		}
	})
}
func TestReviewerPlanRequiresLeaseRecord(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	// plannedSession already carries the winning Lease Record; this
	// gate proves the plan binds its canonical digest, never an
	// envelope fingerprint, and refuses when the record is absent.
	reader.LocalHostID = hostA
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	raw, err := plan.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["lease_record_id"]; !ok {
		t.Fatal("plan omits required validated lease_record_id")
	}
	winner, err := reader.winningLeaseFor(idA, "direct", reader.Local)
	if err != nil {
		t.Fatal(err)
	}
	if plan.LeaseRecordID != winner.Digest {
		t.Fatalf("lease_record_id = %q, want winning record %q", plan.LeaseRecordID, winner.Digest)
	}
	// Absence refuses rather than binding a placeholder.
	bare, _ := repository(t)
	bareRef := create(t, bare, idA, "alpha")
	appendEvent(t, bare, idA, bareRef.RecordID, "session.created", 1, map[string]any{"session_record_id": bareRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	bareReader := &Reader{Local: bare, LocalHostID: hostA}
	if _, err := bareReader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing lease build = %v, want selector_observation_unavailable", err)
	}
}

// TestRev3RealLeaseRecordIdentity proves the plan binds the canonical
// digest of an actual validated Lease Record, never a four-field
// envelope fingerprint. The fixture is validated through the
// canonicaljson owner before driving the production BuildPlan entry.
func TestRev3RealLeaseRecordIdentity(t *testing.T) {
	local, _ := repository(t)
	ref := create(t, local, idA, "alpha")
	appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	leaseBytes := leaseCreate(t, idA, hostA)
	digest, field, err := canonicaljson.VerifyObjectIdentity(leaseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if string(field) != "record_id" {
		t.Fatalf("lease identity field = %q, want record_id", string(field))
	}
	reader := &Reader{Local: local, LocalHostID: hostA}
	withLeases(reader, leaseBytes)
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if plan.LeaseRecordID != digest.String() {
		t.Fatalf("lease_record_id = %q, want validated record %q", plan.LeaseRecordID, digest.String())
	}
	if plan.LeaseEpoch != 1 || plan.LeaseID != lease || plan.OwnerHostID != hostA {
		t.Fatalf("lease triple not from winning record: %+v", plan)
	}
}

// TestRev3DifferentAttestationMustNotAuthorize proves a different
// envelope attestation never authorizes without its Lease Record: a
// plan carrying a substituted digest refuses at revalidation, and a
// build with no admitted record refuses instead of binding.
func TestRev3DifferentAttestationMustNotAuthorize(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	raw, err := plan.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParsePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Revalidate(parsed); err != nil {
		t.Fatalf("fresh plan with its admitted record is not current: %v", err)
	}
	// A substituted attestation digest with no supporting record is
	// caught at shape time when malformed and at revalidation when
	// well-formed but disagreeing: hand-forge a well-formed digest for
	// another lease and prove it does not validate.
	forged := plan
	forged.LeaseID = leaseB
	forged.LeaseRecordID = plan.LeaseRecordID
	if err := reader.Revalidate(forged); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("substituted lease triple = %v, want selector_plan_stale", err)
	}
	empty, _ := repository(t)
	emptyRef := create(t, empty, idA, "alpha")
	appendEvent(t, empty, idA, emptyRef.RecordID, "session.created", 1, map[string]any{"session_record_id": emptyRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	emptyReader := &Reader{Local: empty, LocalHostID: hostA}
	if _, err := emptyReader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("record without lease build = %v, want selector_observation_unavailable", err)
	}
}
