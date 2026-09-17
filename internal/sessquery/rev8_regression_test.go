// Checkpoint profile-authority regression tests for
// TASK-260830-21gygk (SPEC 5.4 event-head closure plus referenced
// 2.4 and 5.2, CR8 P1): every required checkpoint in the winning
// ancestry must admit its transitive event-head closure as Section
// 2.4 profile authority. The first launch carries the Session Record
// creation profile with a null source; every later launch, resume,
// or fork event carries the newest authoritative profile.changed
// event at or before it, and no consumer consults a later
// local-only event or falls back to the creation value. Every case
// drives the four shared production entries (BuildPlan, Revalidate
// fresh and old-plan, AuthoritativeStatus, AuthoritativeList),
// never the validators directly. The rev8 fixture is ported from
// the CR8 independent reviewer regression.
package sessquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// leaseLoser is a second epoch-1 fencing token: events under it
// diverge from the chain head lease, so the repository preserves
// their blobs without indexing them (losing-lease branch).
const leaseLoser = "cccccccc-dddd-4eee-8fff-111111111111"

func rev8ProfileFixture(t *testing.T, kind string, ancestor bool, bad bool) (*Reader, string) {
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
	var source any
	if bad {
		source = zeroDigest
	}
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": source, "profile_mapping": "default"})
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
	return r, tail
}

func TestRev8CheckpointProfileAuthority(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, bad := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/ancestor=%t/missing_source=%t", kind, ancestor, bad), func(t *testing.T) {
					r, _ := rev8ProfileFixture(t, kind, ancestor, bad)
					rev7CheckEntries(t, r, bad)
				})
			}
		}
	}
}

// rev8LaunchPayload builds one provider.launched payload carrying
// the given Section 2.4 pair. A nil source renders the required
// null for the creation pair.
func rev8LaunchPayload(profile string, source any) map[string]any {
	return map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": profile, "profile_source_event_id": source, "profile_mapping": "default"}
}

// rev8ChangedPayload builds one authoritative profile.changed
// payload moving from one profile to the other.
func rev8ChangedPayload(from, to string) map[string]any {
	return map[string]any{"from": from, "to": to, "confirmed": true}
}

// rev8ChangeFixture builds a session whose closure carries a real
// authoritative change: created(1) -> launch(2, yolo/null) ->
// E1(3, yolo->standard) -> launch2(4, standard/E1) -> idle(5) with
// the checkpoint over idle, then the epoch-2 transfer. The ancestor
// variant extends the winning ancestry with a second launch under
// the successor lease that still cites E1. Both closures derive
// (standard, E1) with no fallback to the creation value.
func rev8ChangeFixture(t *testing.T, kind string, ancestor bool) *Reader {
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
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
	changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
	launched2 := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("standard", changed))
	idle := appendEvent(t, local, idA, launched2, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cp := checkpointRecordBytes(t, idA, 1, lease, hostA)
	if kind == "task_board" {
		cp = taskBoardCheckpointBytes(t, idA, 1, lease, hostA)
	}
	cp = rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cp)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", 6, map[string]any{"checkpoint_id": checkpointDigestOf(t, cp), "kind": "manual"})
	tail := appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	if ancestor {
		tail = appendEventAs(t, local, idA, tail, "provider.launched", 2, 2, leaseB, rev8LaunchPayload("standard", changed))
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
	_ = tail
	return r
}

// rev8CheckIntegrityRefusal drives the four shared production
// entries and requires each to refuse with integrity failure: the
// profile gate contradicts corrupt history, it never reports it
// merely unavailable.
func rev8CheckIntegrityRefusal(t *testing.T, r *Reader) {
	t.Helper()
	metadata := summaryReader(t)
	r.HostMetadata, r.Observations = metadata.HostMetadata, metadata.Observations
	stamp := "2026-08-19T04:09:30.000Z"
	observation := r.Observations[idA]
	observation.NewestCheckpointCreatedAt = &stamp
	r.Observations[idA] = observation
	plan, buildErr := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus})
	if buildErr == nil {
		t.Fatalf("BuildPlan admitted corrupt profile authority; Revalidate=%v", r.Revalidate(plan))
	}
	if !errors.Is(buildErr, sessstate.ErrIntegrity) {
		t.Errorf("BuildPlan refusal = %v, want integrity_failure", buildErr)
	}
	if _, err := r.AuthoritativeStatus("alpha"); err == nil {
		t.Error("AuthoritativeStatus admitted corrupt profile authority")
	} else if !errors.Is(err, sessstate.ErrIntegrity) {
		t.Errorf("AuthoritativeStatus refusal = %v, want integrity_failure", err)
	}
	if _, err := r.AuthoritativeList(); err == nil {
		t.Error("AuthoritativeList admitted corrupt profile authority")
	} else if !errors.Is(err, sessstate.ErrIntegrity) {
		t.Errorf("AuthoritativeList refusal = %v, want integrity_failure", err)
	}
}

func TestRev8ProfileChangedPositiveControl(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			name := fmt.Sprintf("%s/ancestor=%t", kind, ancestor)
			t.Run(name, func(t *testing.T) {
				r := rev8ChangeFixture(t, kind, ancestor)
				rev7CheckEntries(t, r, false)
				old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
				if err := r.Revalidate(old); err != nil {
					t.Errorf("fresh Revalidate refused valid profile authority: %v", err)
				}
			})
		}
	}
}

// TestRev8ProfileResumePositiveControl proves a resume carries the
// checkpoint effective pair instead of falling back to the
// creation value: E1 moves the closure to standard, and the resume
// from the E1 checkpoint carries (standard, E1). Both the admitting
// checkpoint and the resume's referenced checkpoint are supplied,
// because the resume expectation resolves through the referenced
// record's own closure.
func TestRev8ProfileResumePositiveControl(t *testing.T) {
	local, _ := repository(t)
	ref, err := local.CreateSession(record(t, idA, "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
	changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
	quiesced := appendEvent(t, local, idA, changed, "session.idle", 4, map[string]any{"boundary_ref": "turn-0", "foreground_idle": true, "background_idle": true})
	cpA := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{quiesced}})
	announced := appendEvent(t, local, idA, quiesced, "checkpoint.created", 5, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpA), "kind": "manual"})
	stopped := appendEvent(t, local, idA, announced, "session.stopped", 6, map[string]any{"graceful": true, "checkpoint_id": checkpointDigestOf(t, cpA), "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true})
	resumed := appendEvent(t, local, idA, stopped, "session.resumed", 7, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpA), "execution_profile": "standard", "profile_source_event_id": changed, "terminal_backend": "tmux", "native_session_id": "sess-1"})
	idle := appendEvent(t, local, idA, resumed, "session.idle", 8, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cpB := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	announcedB := appendEvent(t, local, idA, idle, "checkpoint.created", 9, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpB), "kind": "manual"})
	tail := appendEventAs(t, local, idA, announcedB, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cpB, cpA)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cpB)))
	_ = announced
	_ = tail
	rev7CheckEntries(t, r, false)
	old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if err := r.Revalidate(old); err != nil {
		t.Errorf("fresh Revalidate refused valid resume profile authority: %v", err)
	}
}

// TestRev8ProfileTwoGenerationControl proves the derivation tracks
// the newest change across generations: E2 supersedes E1, and the
// later launch cites (yolo, E2).
func TestRev8ProfileTwoGenerationControl(t *testing.T) {
	local, _ := repository(t)
	ref, err := local.CreateSession(record(t, idA, "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
	changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
	changed2 := appendEvent(t, local, idA, changed, "profile.changed", 4, rev8ChangedPayload("standard", "yolo"))
	launched2 := appendEvent(t, local, idA, changed2, "provider.launched", 5, rev8LaunchPayload("yolo", changed2))
	idle := appendEvent(t, local, idA, launched2, "session.idle", 6, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", 7, map[string]any{"checkpoint_id": checkpointDigestOf(t, cp), "kind": "manual"})
	_ = announced
	appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cp)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
	rev7CheckEntries(t, r, false)
}

// TestRev8ProfileSourceRefusals drives one corrupt history per
// rejected source class through all four shared entries: a source
// naming a non-change event, a superseded change, a stale creation
// value, a wrong first-launch profile, a dangling resume source, a
// cross-session change, a losing-lease preserved change, and a
// dangling closure predecessor. A citation of a well-formed later
// local-only digest from inside the closure has no fixture: chain
// continuity forces every prior event into the closure, so a
// writer cannot cite its future; the historical control below
// proves later events are never consulted instead.
func TestRev8ProfileSourceRefusals(t *testing.T) {
	builders := map[string]func(t *testing.T) *Reader{
		"wrong_type": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			launched2 := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("standard", created))
			idle := appendEvent(t, local, idA, launched2, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
		"non_newest": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			changed2 := appendEvent(t, local, idA, changed, "profile.changed", 4, rev8ChangedPayload("standard", "yolo"))
			launched2 := appendEvent(t, local, idA, changed2, "provider.launched", 5, rev8LaunchPayload("standard", changed))
			idle := appendEvent(t, local, idA, launched2, "session.idle", 6, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
		"stale_value": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			launched2 := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("yolo", changed))
			idle := appendEvent(t, local, idA, launched2, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
		"first_wrong_profile": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("standard", nil))
			idle := appendEvent(t, local, idA, launched, "session.idle", 3, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
		"dangling_resume_source": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			quiesced := appendEvent(t, local, idA, changed, "session.idle", 4, map[string]any{"boundary_ref": "turn-0", "foreground_idle": true, "background_idle": true})
			cpA := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{quiesced}})
			resumed := appendEvent(t, local, idA, quiesced, "session.resumed", 5, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpA), "execution_profile": "standard", "profile_source_event_id": zeroDigest, "terminal_backend": "tmux", "native_session_id": "sess-1"})
			idle := appendEvent(t, local, idA, resumed, "session.idle", 6, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp, cpA)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
		"cross_session": func(t *testing.T) *Reader {
			local, _ := repository(t)
			peer := create(t, local, idB, "beta")
			peerCreated := appendEvent(t, local, idB, peer.RecordID, "session.created", 1, map[string]any{"session_record_id": peer.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			peerLaunched := appendEvent(t, local, idB, peerCreated, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			peerChanged := appendEvent(t, local, idB, peerLaunched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			launched2 := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("standard", peerChanged))
			idle := appendEvent(t, local, idA, launched2, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)), leaseCreate(t, idB, hostA))
			return r
		},
		"losing_branch": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
			loser := rev8PreservedLoser(t, local, idA, changed)
			launched2 := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("standard", loser))
			idle := appendEvent(t, local, idA, launched2, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
		"dangling_predecessor": func(t *testing.T) *Reader {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			idle := appendEvent(t, local, idA, launched, "session.idle", 3, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			merged := rev8AppendPreds(t, local, idA, []string{zeroDigest, idle}, "session.idle", 4, map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
			cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{merged}})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cp)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
			return r
		},
	}
	order := []string{"wrong_type", "non_newest", "stale_value", "first_wrong_profile", "dangling_resume_source", "cross_session", "losing_branch", "dangling_predecessor"}
	for _, name := range order {
		t.Run(name, func(t *testing.T) {
			rev8CheckIntegrityRefusal(t, builders[name](t))
		})
	}
}

// rev8AppendPreds chains one event with an explicit predecessor
// list, unlike the single-predecessor fixture helpers. The chain
// owner still requires the prior tail inside the list, so a merge
// naming a dangling digest appends and reaches the profile gate.
func rev8AppendPreds(t *testing.T, repo *sessrepo.Repository, sessionID string, predecessors []string, typ string, sequence int, payload map[string]any) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "event_type": typ, "created_by_host_id": hostA,
		"lease_epoch": 1, "lease_id": lease, "lease_sequence": sequence, "predecessors": predecessors,
		"created_at": "2026-08-19T04:00:00.000Z", "payload": payload, "extensions": map[string]any{},
	}
	ref, err := repo.AppendEvent(sessionID, identity(t, value, "event_id"))
	if err != nil {
		t.Fatal(err)
	}
	return ref.EventID
}

// rev8PreservedLoser appends one profile.changed event under a
// losing same-epoch lease and returns its digest. The repository
// preserves the blob without indexing it, so the digest is
// well-formed and provider-signed yet outside the authoritative
// chain: exactly the losing-lease member the profile gate must
// reject.
func rev8PreservedLoser(t *testing.T, repo *sessrepo.Repository, sessionID, predecessor string) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "event_type": "profile.changed", "created_by_host_id": hostA,
		"lease_epoch": 1, "lease_id": leaseLoser, "lease_sequence": 1, "predecessors": []string{predecessor},
		"created_at": "2026-08-19T04:00:00.000Z", "payload": rev8ChangedPayload("yolo", "standard"), "extensions": map[string]any{},
	}
	raw := identity(t, value, "event_id")
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	digest, _ := decoded["event_id"].(string)
	if _, err := repo.AppendEvent(sessionID, raw); !errors.Is(err, sessrepo.ErrDivergentBranch) {
		t.Fatalf("losing-lease change append = %v, want divergent branch preservation", err)
	}
	indexed, err := repo.ListEvents(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range indexed {
		if summary.EventID == digest {
			t.Fatalf("losing-lease change %s entered the authoritative index", digest)
		}
	}
	return digest
}

// TestRev8ProfileHistoricalClosureAdmits proves the closure fixes
// the effective pair: a later local-only change past the heads is
// never consulted, so the checkpoint still admits with (standard,
// E1) while the tail already moved on.
func TestRev8ProfileHistoricalClosureAdmits(t *testing.T) {
	local, _ := repository(t)
	ref, err := local.CreateSession(record(t, idA, "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
	changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
	launched2 := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("standard", changed))
	idle := appendEvent(t, local, idA, launched2, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cp := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", 6, map[string]any{"checkpoint_id": checkpointDigestOf(t, cp), "kind": "manual"})
	transferred := appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	later := appendEventAs(t, local, idA, transferred, "profile.changed", 2, 2, leaseB, rev8ChangedPayload("standard", "yolo"))
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cp)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
	_ = later
	rev7CheckEntries(t, r, false)
}

// TestRev8ProfileOldPlanSubstitution proves an old plan revalidates
// profile authority instead of trusting its bound lease: the plan
// builds over the E1 checkpoint, then the reader substitutes a
// checkpoint whose valid heads cover a stale creation pair, and
// the old plan refuses with integrity failure while heads alone
// would still admit it.
func TestRev8ProfileOldPlanSubstitution(t *testing.T) {
	local, _ := repository(t)
	ref, err := local.CreateSession(record(t, idA, "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
	changed := appendEvent(t, local, idA, launched, "profile.changed", 3, rev8ChangedPayload("yolo", "standard"))
	stale := appendEvent(t, local, idA, changed, "provider.launched", 4, rev8LaunchPayload("yolo", nil))
	idle := appendEvent(t, local, idA, stale, "session.idle", 5, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	good := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{changed}})
	bad := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", 6, map[string]any{"checkpoint_id": checkpointDigestOf(t, good), "kind": "manual"})
	_ = announced
	appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, good)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, good)))
	rev7CheckEntries(t, r, false)
	old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
	rev7SwapCheckpoint(t, r, bad)
	if err := r.Revalidate(old); err == nil {
		t.Fatal("old-plan Revalidate admitted substituted profile authority")
	} else if !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("old-plan Revalidate refusal = %v, want integrity_failure", err)
	}
	rev8CheckIntegrityRefusal(t, r)
}
