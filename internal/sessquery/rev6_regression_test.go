// Checkpoint persistence-variant regression tests for TASK-260830-21gygk
// (SPEC 5.4, CR6 P1): the referenced Session Record selects the
// persistence variant. A direct session admits only a provider-manifest
// checkpoint; a task_board session admits only a task-board-bundle
// checkpoint. Canonical identity, session, lease, epoch, and creator
// agreement alone never admit a swapped variant. Every case drives
// the four shared production entries (BuildPlan, Revalidate fresh and
// old-plan, AuthoritativeStatus, AuthoritativeList), never the
// validators directly.
package sessquery

import (
	"encoding/json"
	"errors"
	"testing"
)

// TestRev6CheckpointPersistenceMustMatchSession ports the CR6 reviewer
// regression: a task_board-bundle checkpoint for a direct session
// refuses on the winner path and on a necessary-ancestor path,
// including old-plan revalidation after only the ancestor variant
// changes. Both direct controls pass. Variant swaps preserve the
// real event-head closure so the persistence gate is isolated from
// the head-authority gate.
func TestRev6CheckpointPersistenceMustMatchSession(t *testing.T) {
	swapToTaskBoard := func(t *testing.T, good []byte) []byte {
		t.Helper()
		var value map[string]any
		if err := json.Unmarshal(good, &value); err != nil {
			t.Fatal(err)
		}
		value["task_board_bundle_id"] = value["provider_manifest_id"]
		value["provider_manifest_id"] = nil
		return identity(t, value, "checkpoint_id")
	}
	for _, ancestor := range []bool{false, true} {
		path := "winner"
		if ancestor {
			path = "ancestor"
		}
		for _, wrong := range []bool{false, true} {
			mode := "direct_control"
			if wrong {
				mode = "task_board_checkpoint_for_direct_session"
			}
			t.Run(path+"/"+mode, func(t *testing.T) {
				var r *Reader
				if ancestor {
					good := checkpointRecordBytes(t, idA, 1, lease, hostA)
					r = rev5AncestorFixture(t, good)
					old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
					if wrong {
						// Preserve the winner digest, event bytes, and real
						// head closure; alter only the ancestor's
						// checkpoint variant, retaining canonical identities.
						cp := swapToTaskBoard(t, r.CheckpointRecords[0])
						r.CheckpointRecords[0] = cp
						for i, raw := range r.LeaseRecords {
							var value map[string]any
							if err := json.Unmarshal(raw, &value); err != nil {
								t.Fatal(err)
							}
							if value["lease_id"] == leaseB {
								value["checkpoint_id"] = checkpointDigestOf(t, cp)
								r.LeaseRecords[i] = identity(t, value, "record_id")
							}
						}
						if err := r.Revalidate(old); err == nil {
							t.Error("old-plan Revalidate admits wrong persistence variant in necessary ancestor")
						}
					}
				} else {
					r = successorFixture(t)
					goodCp := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), headAtOrBefore(t, r, idA, 1))
					withCheckpoints(r, goodCp)
					withLeases(r, leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, goodCp)))
					old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
					if wrong {
						// Swap only the winner checkpoint variant, keeping the
						// lease tuple, event bytes, real head closure, and
						// canonical identities.
						cp := swapToTaskBoard(t, r.CheckpointRecords[0])
						r.CheckpointRecords[0] = cp
						for i, raw := range r.LeaseRecords {
							var value map[string]any
							if err := json.Unmarshal(raw, &value); err != nil {
								t.Fatal(err)
							}
							if value["lease_id"] == leaseB {
								value["checkpoint_id"] = checkpointDigestOf(t, cp)
								r.LeaseRecords[i] = identity(t, value, "record_id")
							}
						}
						if err := r.Revalidate(old); err == nil {
							t.Error("old-plan Revalidate admits wrong persistence variant in winner checkpoint")
						}
					}
				}
				rev5CheckEntries(t, r, wrong)
			})
		}
	}
}

// taskBoardRecordBytes builds one validated task_board Session Record
// for tests, following the SPEC 5.1 task_board example with a
// tracked_prompt reference and no board goal.
func taskBoardRecordBytes(t *testing.T, id, name string) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(specRecordExample), &value); err != nil {
		t.Fatal(err)
	}
	value["session_id"], value["subject_id"], value["name"] = id, id, name
	value["kind"] = "task_board"
	value["task_board"] = map[string]any{
		"bridge_protocol_version": "1.0.0",
		"board": map[string]any{
			"kind": "local", "logical_id": "agent-session-manager-spec",
			"remote_url": nil, "extensions": map[string]any{},
		},
		"task_element_id": "TASK-260819-example", "launch_mode": "tracked_prompt",
		"manager_session_ref": nil, "board_goal": nil,
		"native_goal_binding": "prompt", "extensions": map[string]any{},
	}
	return identity(t, value, "record_id")
}

// taskBoardCheckpointBytes builds one validated Checkpoint Record
// carrying the task_board persistence variant: null provider manifest
// with a non-null task-board bundle.
func taskBoardCheckpointBytes(t *testing.T, sessionID string, epoch uint64, leaseID, creator string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:checkpoint", "schema_version": "1.0.0", "checkpoint_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID,
		"lease_epoch": json.Number(uint64String(epoch)), "lease_id": leaseID,
		"safe_boundary": map[string]any{
			"provider_id": "codex", "provider_version": "0.147.0", "evidence": "accepted_test",
			"input_blocked": true, "foreground_idle": true, "background_idle": true,
			"open_processes": json.Number("0"), "open_database_handles": json.Number("0"),
		},
		"event_heads":           []string{zeroDigest},
		"workspace_manifest_id": zeroDigest,
		"provider_manifest_id":  nil,
		"task_board_bundle_id":  zeroDigest,
		"created_by_host_id":    creator,
		"created_at":            "2026-08-19T04:09:30.000Z",
		"status":                "validated",
		"extensions":            map[string]any{},
	}
	return identity(t, value, "checkpoint_id")
}

// taskBoardSuccessorFixture builds a task_board session with the
// epoch-1 to epoch-2 transfer chain and a task_board-variant
// checkpoint bound to the predecessor lease. The caller supplies no
// variant: the only admissible checkpoint is the task_board one. The
// checkpoint names the real launch head so the head-authority gate
// does not mask the persistence gate.
func taskBoardSuccessorFixture(t *testing.T) *Reader {
	t.Helper()
	local, _ := repository(t)
	raw := taskBoardRecordBytes(t, idA, "alpha")
	ref, err := local.CreateSession(raw)
	if err != nil {
		t.Fatal(err)
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
	appendEventAs(t, local, idA, launched, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	reader := &Reader{Local: local, LocalHostID: hostA}
	cp := checkpointWithHeads(t, taskBoardCheckpointBytes(t, idA, 1, lease, hostA), launched)
	withCheckpoints(reader, cp)
	withLeases(reader,
		leaseCreate(t, idA, hostA),
		leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)),
	)
	return reader
}

// TestRev6TaskBoardPersistenceControl proves the variant gate selects
// by session kind instead of admitting one hard-coded variant: the
// task_board session with its task_board checkpoint passes all four
// entries, while a provider-manifest checkpoint for that same session
// refuses through all four entries including old-plan revalidation.
func TestRev6TaskBoardPersistenceControl(t *testing.T) {
	t.Run("task_board_control", func(t *testing.T) {
		rev5CheckEntries(t, taskBoardSuccessorFixture(t), false)
	})
	t.Run("provider_checkpoint_for_task_board_session", func(t *testing.T) {
		r := taskBoardSuccessorFixture(t)
		old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
		provider := checkpointWithHeads(t, checkpointRecordBytes(t, idA, 1, lease, hostA), headAtOrBefore(t, r, idA, 1))
		r.CheckpointRecords[0] = provider
		for i, raw := range r.LeaseRecords {
			var value map[string]any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			if value["lease_id"] == leaseB {
				value["checkpoint_id"] = checkpointDigestOf(t, provider)
				r.LeaseRecords[i] = identity(t, value, "record_id")
			}
		}
		if err := r.Revalidate(old); err == nil {
			t.Error("old-plan Revalidate admits provider persistence variant for task_board session")
		}
		rev5CheckEntries(t, r, true)
	})
}

// TestRev6PersistenceMismatchIsObservationUnavailable pins the refusal
// class: a swapped variant is incomplete authority, never a missing
// lease, a bootstrap prefix, or a minted fact. The swapped record
// keeps a real head so the class proves the persistence gate.
func TestRev6PersistenceMismatchIsObservationUnavailable(t *testing.T) {
	r := successorFixture(t)
	wrong := checkpointWithHeads(t, taskBoardCheckpointBytes(t, idA, 1, lease, hostA), headAtOrBefore(t, r, idA, 1))
	withCheckpoints(r, wrong)
	withLeases(r, leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, wrong)))
	if _, err := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("BuildPlan variant mismatch = %v, want selector_observation_unavailable", err)
	}
}
