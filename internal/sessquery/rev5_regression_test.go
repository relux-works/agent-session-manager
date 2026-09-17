package sessquery

import (
	"encoding/json"
	"testing"
)

func rev5CheckEntries(t *testing.T, r *Reader, refuse bool) {
	t.Helper()
	metadata := summaryReader(t)
	r.HostMetadata, r.Observations = metadata.HostMetadata, metadata.Observations
	p, buildErr := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus})
	if refuse && buildErr == nil {
		t.Errorf("BuildPlan admitted invalid lease/checkpoint authority; Revalidate=%v", r.Revalidate(p))
	}
	if !refuse && buildErr != nil {
		t.Fatalf("positive BuildPlan: %v", buildErr)
	}
	if !refuse {
		if err := r.Revalidate(p); err != nil {
			t.Errorf("positive Revalidate: %v", err)
		}
	}
	_, statusErr := r.AuthoritativeStatus("alpha")
	_, listErr := r.AuthoritativeList()
	if refuse && statusErr == nil {
		t.Error("AuthoritativeStatus admitted invalid lease/checkpoint authority")
	}
	if refuse && listErr == nil {
		t.Error("AuthoritativeList admitted invalid lease/checkpoint authority")
	}
	if !refuse && (statusErr != nil || listErr != nil) {
		t.Errorf("positive summaries: status=%v list=%v", statusErr, listErr)
	}
}

func TestRev5AncestorCheckpointAuthority(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "complete"
		if missing {
			name = "missing_ancestor_checkpoint"
		}
		t.Run(name, func(t *testing.T) {
			r, _, _, created := plannedSession(t)
			launched := appendEvent(t, r.Local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
			cp1 := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), launched)
			cp2 := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 2, leaseB, hostA), launched)
			withCheckpoints(r, cp2)
			if !missing {
				withCheckpoints(r, cp1)
			}
			withLeases(r, leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp1)), leaseSuccessorBytes(t, idA, 3, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, cp2)))
			var forceRecord map[string]any
			if err := json.Unmarshal(r.LeaseRecords[len(r.LeaseRecords)-1], &forceRecord); err != nil {
				t.Fatal(err)
			}
			forceRecord["reason"] = "force_takeover"
			r.LeaseRecords[len(r.LeaseRecords)-1] = identity(t, forceRecord, "record_id")
			appendEventAs(t, r.Local, idA, launched, "lease.forced", 1, 3, leaseCycleB, map[string]any{"operation_id": hostB, "expected_owner_host_id": hostA, "expected_epoch": 2, "new_lease_id": leaseCycleB, "checkpoint_id": checkpointDigestOf(t, cp2)})
			if missing {
				withCheckpoints(r, cp1)
				oldPlan := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
				r.CheckpointRecords = r.CheckpointRecords[:len(r.CheckpointRecords)-1]
				if err := r.Revalidate(oldPlan); err == nil {
					t.Error("Revalidate retained an originally valid plan after ancestor checkpoint authority became unavailable")
				}
			}
			rev5CheckEntries(t, r, missing)
		})
	}
}

func TestRev5CheckpointCreatorMustBeHolder(t *testing.T) {
	for _, wrong := range []bool{false, true} {
		name := "holder"
		creator := hostA
		if wrong {
			name = "non_holder"
			creator = hostB
		}
		t.Run(name, func(t *testing.T) {
			r := successorFixture(t)
			cp := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, creator), headAtOrBefore(t, r, idA, 1))
			withCheckpoints(r, cp)
			withLeases(r, leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			rev5CheckEntries(t, r, wrong)
		})
	}
}

// rev5AncestorFixture builds the three-generation winning ancestry the
// ancestor-authority negatives share: epoch-1/2/3 canonical Lease
// Records with legal predecessor IDs and epochs, an event chain
// observing the epoch-3 force takeover, and a valid epoch-2
// checkpoint. The caller supplies the epoch-1 checkpoint the epoch-2
// ancestor references; only a checkpoint bound to (idA, 1, lease)
// and created by hostA is admissible. Both checkpoints are rebound
// to the real epoch-1 launch head so the event-head closure gate
// does not mask the ancestry/creator/persistence gates under test.
func rev5AncestorFixture(t *testing.T, cp1 []byte) *Reader {
	t.Helper()
	r, _, _, created := plannedSession(t)
	launched := appendEvent(t, r.Local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	cp1 = checkpointWithHeads(t, cp1, launched)
	cp2 := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 2, leaseB, hostA), launched)
	withCheckpoints(r, cp1, cp2)
	withLeases(r, leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp1)), leaseSuccessorBytes(t, idA, 3, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, cp2)))
	var forceRecord map[string]any
	if err := json.Unmarshal(r.LeaseRecords[len(r.LeaseRecords)-1], &forceRecord); err != nil {
		t.Fatal(err)
	}
	forceRecord["reason"] = "force_takeover"
	r.LeaseRecords[len(r.LeaseRecords)-1] = identity(t, forceRecord, "record_id")
	appendEventAs(t, r.Local, idA, launched, "lease.forced", 1, 3, leaseCycleB, map[string]any{"operation_id": hostB, "expected_owner_host_id": hostA, "expected_epoch": 2, "new_lease_id": leaseCycleB, "checkpoint_id": checkpointDigestOf(t, cp2)})
	return r
}

// TestRev5AncestorCheckpointBinding proves the ancestor gate binds
// the required checkpoint relationship instead of admitting any
// canonically identified record: an ancestor checkpoint bound to
// another session, another lease, or another creator refuses
// through BuildPlan, Revalidate, AuthoritativeStatus, and
// AuthoritativeList alike.
func TestRev5AncestorCheckpointBinding(t *testing.T) {
	cases := []struct {
		name string
		cp1  func(t *testing.T) []byte
	}{
		{"wrong_session", func(t *testing.T) []byte { return checkpointRecordBytes(t, idB, 1, lease, hostA) }},
		{"wrong_lease", func(t *testing.T) []byte { return checkpointRecordBytes(t, idA, 1, leaseCycleB, hostA) }},
		{"wrong_creator", func(t *testing.T) []byte { return checkpointRecordBytes(t, idA, 1, lease, hostB) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rev5CheckEntries(t, rev5AncestorFixture(t, tc.cp1(t)), true)
		})
	}
}
