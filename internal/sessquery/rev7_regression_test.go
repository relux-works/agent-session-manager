// Checkpoint event-head authority regression tests for
// TASK-260830-21gygk (SPEC 5.4, CR7 P1): every required checkpoint
// in the winning ancestry must resolve each event head to a chained
// event for its session at or before its bound lease in the winning
// source chain. A head naming an absent event or an event under a
// later lease cannot establish current complete authority. Every
// case drives the four shared production entries (BuildPlan,
// Revalidate fresh and old-plan, AuthoritativeStatus,
// AuthoritativeList), never the validators directly. Ported from the
// CR7 independent reviewer regression and retained as the candidate
// positive/negative control for the shared admission path.
package sessquery

import (
	"encoding/json"
	"errors"
	"testing"
)

func rev7Replace(t *testing.T, raw []byte, self string, values map[string]any) []byte {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	for k, value := range values {
		v[k] = value
	}
	return identity(t, v, self)
}

// Complete local event histories give the checkpoint a real idle head under
// its owning predecessor, rather than the shipped fixture's zero digest.
// Manifest publication/storage is the accepted external fixture boundary.
func rev7Fixture(t *testing.T, kind string, ancestor bool) (*Reader, string) {
	t.Helper()
	local, _ := repository(t)
	raw := record(t, idA, "alpha")
	if kind == "task_board" {
		raw = taskBoardRecordBytes(t, idA, "alpha")
	}
	ref, err := local.CreateSession(raw)
	if err != nil {
		t.Fatal(err)
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	idle := appendEvent(t, local, idA, launched, "session.idle", 3, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cp := checkpointRecordBytes(t, idA, 1, lease, hostA)
	if kind == "task_board" {
		cp = taskBoardCheckpointBytes(t, idA, 1, lease, hostA)
	}
	cp = rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cp)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", 4, map[string]any{"checkpoint_id": checkpointDigestOf(t, cp), "kind": "manual"})
	tail := appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	if ancestor {
		tail = appendEventAs(t, local, idA, tail, "provider.launched", 2, 2, leaseB, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
		tail = appendEventAs(t, local, idA, tail, "session.idle", 3, 2, leaseB, map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
		cp2 := checkpointRecordBytes(t, idA, 2, leaseB, hostA)
		if kind == "task_board" {
			cp2 = taskBoardCheckpointBytes(t, idA, 2, leaseB, hostA)
		}
		cp2 = rev7Replace(t, cp2, "checkpoint_id", map[string]any{"event_heads": []string{tail}})
		withCheckpoints(r, cp2)
		withLeases(r, leaseSuccessorBytes(t, idA, 3, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, cp2)))
		tail = appendEventAs(t, local, idA, tail, "lease.transferred", 1, 3, leaseCycleB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": leaseB, "new_lease_id": leaseCycleB})
	}
	rev7CheckEntries(t, r, false)
	return r, tail
}

func rev7SwapCheckpoint(t *testing.T, r *Reader, cp []byte) {
	t.Helper()
	r.CheckpointRecords[0] = cp
	for i, raw := range r.LeaseRecords {
		var v map[string]any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		if v["lease_id"] == leaseB {
			r.LeaseRecords[i] = rev7Replace(t, raw, "record_id", map[string]any{"checkpoint_id": checkpointDigestOf(t, cp)})
		}
	}
}

func TestRev7IndependentPersistence(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			path := "winner"
			if ancestor {
				path = "ancestor"
			}
			for _, wrong := range []bool{false, true} {
				mode := "control"
				if wrong {
					mode = "wrong_variant"
				}
				t.Run(kind+"/"+path+"/"+mode, func(t *testing.T) {
					r, _ := rev7Fixture(t, kind, ancestor)
					old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
					if wrong {
						change := map[string]any{"provider_manifest_id": nil, "task_board_bundle_id": zeroDigest}
						if kind == "task_board" {
							change = map[string]any{"provider_manifest_id": zeroDigest, "task_board_bundle_id": nil}
						}
						rev7SwapCheckpoint(t, r, rev7Replace(t, r.CheckpointRecords[0], "checkpoint_id", change))
						if err := r.Revalidate(old); err == nil {
							t.Error("old-plan Revalidate admitted wrong persistence variant")
						}
					}
					rev7CheckEntries(t, r, wrong)
				})
			}
		}
	}
}

func TestRev7CheckpointHeadAuthority(t *testing.T) {
	for _, ancestor := range []bool{false, true} {
		path := "winner"
		if ancestor {
			path = "ancestor"
		}
		for _, mode := range []string{"control", "missing_head", "later_lease_head"} {
			t.Run(path+"/"+mode, func(t *testing.T) {
				r, future := rev7Fixture(t, "direct", ancestor)
				old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
				if mode != "control" {
					head := zeroDigest
					if mode == "later_lease_head" {
						head = future
					}
					rev7SwapCheckpoint(t, r, rev7Replace(t, r.CheckpointRecords[0], "checkpoint_id", map[string]any{"event_heads": []string{head}}))
					if err := r.Revalidate(old); err == nil {
						t.Error("old-plan Revalidate admitted invalid checkpoint event-head authority")
					} else if !errors.Is(err, ErrObservationUnavailable) {
						t.Errorf("old-plan Revalidate refusal = %v, want observation_unavailable", err)
					}
				}
				rev7CheckEntries(t, r, mode != "control")
				if mode != "control" {
					rev7CheckHeadRefusalClass(t, r)
				}
			})
		}
	}
}

// rev7CheckHeadRefusalClass pins the head-authority refusal to the
// incomplete-authority class: resolvability is admitted before
// profile derivation, so a missing or later-lease head refuses
// observation_unavailable even though the Section 2.4 profile
// closure would independently contradict it. The pin keeps the
// CR8 profile gate from masking the CR7 heads gate in mutant
// attribution: weakening the heads gate surfaces here as the wrong
// class, never as silent admission.
func rev7CheckHeadRefusalClass(t *testing.T, r *Reader) {
	t.Helper()
	metadata := summaryReader(t)
	r.HostMetadata, r.Observations = metadata.HostMetadata, metadata.Observations
	if _, err := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); err == nil {
		t.Fatal("BuildPlan admitted invalid checkpoint event-head authority")
	} else if !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("BuildPlan refusal = %v, want observation_unavailable", err)
	}
	if _, err := r.AuthoritativeStatus("alpha"); err == nil {
		t.Fatal("AuthoritativeStatus admitted invalid checkpoint event-head authority")
	} else if !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("AuthoritativeStatus refusal = %v, want observation_unavailable", err)
	}
	if _, err := r.AuthoritativeList(); err == nil {
		t.Fatal("AuthoritativeList admitted invalid checkpoint event-head authority")
	} else if !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("AuthoritativeList refusal = %v, want observation_unavailable", err)
	}
}

func rev7CheckEntries(t *testing.T, r *Reader, refuse bool) {
	t.Helper()
	metadata := summaryReader(t)
	r.HostMetadata, r.Observations = metadata.HostMetadata, metadata.Observations
	stamp := "2026-08-19T04:09:30.000Z"
	observation := r.Observations[idA]
	observation.NewestCheckpointCreatedAt = &stamp
	r.Observations[idA] = observation
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
