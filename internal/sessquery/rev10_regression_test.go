// Fork and resume profile-authority regression tests for
// TASK-260830-21gygk (SPEC 5.2 pair sentence with referenced 2.4 and
// 5.4, CR10 P1-A/P1-B): every fork.created event in a required
// checkpoint closure must carry the newly persisted Session Record
// profile with a null source, and every session.resumed event must
// carry the effective pair of its own referenced checkpoint's
// event-head closure, never the outer-walk prefix. Every case drives
// the four shared production entries (BuildPlan, Revalidate fresh
// and old-plan, AuthoritativeStatus, AuthoritativeList), never the
// validators directly. TestReview9ForkLocalProfile and
// TestReview9ResumeReferencedCheckpoint are ported verbatim from the
// CR9 independent reviewer probes; the TestRev10* suites add
// winner/necessary-ancestor, direct/task_board, old-plan
// substitution, and task_board.launched variants with refusal-class
// pins.
package sessquery

import (
	"errors"
	"fmt"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// rev10ForkPayload builds one fork.created payload carrying the given
// Section 2.4 new-session pair plus fixed provenance members. A nil
// source renders the required null; provenance takes any digest and
// never participates in new-session derivation.
func rev10ForkPayload(recordID, profile string, source, provenance any) map[string]any {
	return map[string]any{"source_session_id": idB, "source_checkpoint_id": zeroDigest, "new_session_record_id": recordID, "provider_fork_mode": "native", "execution_profile": profile, "profile_source_event_id": source, "source_profile_event_id": provenance}
}

// rev10TaskBoardLaunchPayload builds one task_board.launched payload
// carrying the given Section 2.4 pair. The launch repeats the
// winning creation lease and the Session Record provider/mode.
func rev10TaskBoardLaunchPayload(profile string, source any) map[string]any {
	return map[string]any{"operation_id": hostB, "manager_session_ref": "manager-1", "provider_id": "codex", "launch_mode": "tracked_prompt", "lease_epoch": 1, "lease_id": lease, "execution_profile": profile, "profile_source_event_id": source, "board_goal_id": nil, "board_goal_revision": nil, "state": "running"}
}

// rev10ResumePayload builds one session.resumed payload carrying the
// given referenced checkpoint and Section 2.4 pair.
func rev10ResumePayload(checkpointID, profile string, source any) map[string]any {
	return map[string]any{"checkpoint_id": checkpointID, "execution_profile": profile, "profile_source_event_id": source, "terminal_backend": "tmux", "native_session_id": "sess-1"}
}

// rev10CheckUnavailableRefusal drives BuildPlan and both authoritative
// summaries and requires each to refuse with observation_unavailable:
// the referenced authority cannot be established, so it is never
// minted and never reported as contradictory history.
func rev10CheckUnavailableRefusal(t *testing.T, r *Reader) {
	t.Helper()
	metadata := summaryReader(t)
	r.HostMetadata, r.Observations = metadata.HostMetadata, metadata.Observations
	stamp := "2026-08-19T04:09:30.000Z"
	observation := r.Observations[idA]
	observation.NewestCheckpointCreatedAt = &stamp
	r.Observations[idA] = observation
	plan, buildErr := r.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus})
	if buildErr == nil {
		t.Fatalf("BuildPlan admitted missing referenced authority; Revalidate=%v", r.Revalidate(plan))
	}
	if !errors.Is(buildErr, ErrObservationUnavailable) {
		t.Errorf("BuildPlan refusal = %v, want observation_unavailable", buildErr)
	}
	if _, err := r.AuthoritativeStatus("alpha"); err == nil {
		t.Error("AuthoritativeStatus admitted missing referenced authority")
	} else if !errors.Is(err, ErrObservationUnavailable) {
		t.Errorf("AuthoritativeStatus refusal = %v, want observation_unavailable", err)
	}
	if _, err := r.AuthoritativeList(); err == nil {
		t.Error("AuthoritativeList admitted missing referenced authority")
	} else if !errors.Is(err, ErrObservationUnavailable) {
		t.Errorf("AuthoritativeList refusal = %v, want observation_unavailable", err)
	}
}

func TestReview9ForkLocalProfile(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, bad := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/bad=%t", kind, bad), func(t *testing.T) {
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
				profile := "yolo"
				if bad {
					profile = "standard"
				}
				fork := appendEvent(t, local, idA, created, "fork.created", 2, map[string]any{"source_session_id": idB, "source_checkpoint_id": zeroDigest, "new_session_record_id": ref.RecordID, "provider_fork_mode": "native", "execution_profile": profile, "profile_source_event_id": nil, "source_profile_event_id": nil})
				launched := appendEvent(t, local, idA, fork, "provider.launched", 3, rev8LaunchPayload("yolo", nil))
				idle := appendEvent(t, local, idA, launched, "session.idle", 4, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
				cp := checkpointRecordBytes(t, idA, 1, lease, hostA)
				if kind == "task_board" {
					cp = taskBoardCheckpointBytes(t, idA, 1, lease, hostA)
				}
				good := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{created}})
				covered := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{idle}})
				announced := appendEvent(t, local, idA, idle, "checkpoint.created", 5, map[string]any{"checkpoint_id": checkpointDigestOf(t, good), "kind": "manual"})
				appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
				r := &Reader{Local: local, LocalHostID: hostA}
				withCheckpoints(r, good)
				withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, good)))
				rev7CheckEntries(t, r, false)
				old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
				rev7SwapCheckpoint(t, r, covered)
				err = r.Revalidate(old)
				t.Logf("old-plan substitution refusal: %v", err)
				if bad && err == nil {
					t.Error("old-plan substitution admitted")
				}
				rev7CheckEntries(t, r, bad)
			})
		}
	}
}

func TestReview9ResumeReferencedCheckpoint(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprintf("wrong_checkpoint_pair=%t", bad), func(t *testing.T) {
			local, _ := repository(t)
			ref, err := local.CreateSession(record(t, idA, "alpha"))
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			beforeChangeIdle := appendEvent(t, local, idA, launched, "session.idle", 3, map[string]any{"boundary_ref": "before-change", "foreground_idle": true, "background_idle": true})
			changed := appendEvent(t, local, idA, beforeChangeIdle, "profile.changed", 4, rev8ChangedPayload("yolo", "standard"))
			quiesced := appendEvent(t, local, idA, changed, "session.idle", 5, map[string]any{"boundary_ref": "turn-0", "foreground_idle": true, "background_idle": true})
			head := quiesced
			if bad {
				head = beforeChangeIdle
			}
			cpA := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{head}})
			announced := appendEvent(t, local, idA, quiesced, "checkpoint.created", 6, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpA), "kind": "manual"})
			stopped := appendEvent(t, local, idA, announced, "session.stopped", 7, map[string]any{"graceful": true, "checkpoint_id": checkpointDigestOf(t, cpA), "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true})
			resumed := appendEvent(t, local, idA, stopped, "session.resumed", 8, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpA), "execution_profile": "standard", "profile_source_event_id": changed, "terminal_backend": "tmux", "native_session_id": "sess-1"})
			idle := appendEvent(t, local, idA, resumed, "session.idle", 9, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			cpB := rev7Replace(t, checkpointRecordBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			announcedB := appendEvent(t, local, idA, idle, "checkpoint.created", 10, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpB), "kind": "manual"})
			tail := appendEventAs(t, local, idA, announcedB, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, cpB, cpA)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cpB)))
			_ = announced
			_ = tail
			rev7CheckEntries(t, r, bad)
			if bad {
				return
			}
			old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
			if err := r.Revalidate(old); err != nil {
				t.Errorf("fresh Revalidate refused valid resume profile authority: %v", err)
			}
		})
	}
}

// rev10ForkFixture builds a session whose checkpoint closure carries
// one fork: created -> fork(mode) -> launch(yolo/null) -> idle with
// the checkpoint over idle, then the epoch-2 transfer. The ancestor
// variant extends the winning ancestry so the forked checkpoint is
// validated through the necessary-ancestor path. The good_provenance
// mode carries a real source_profile_event_id digest plus a preserved
// losing-lease change: neither participates in derivation.
func rev10ForkFixture(t *testing.T, kind string, ancestor bool, mode string) *Reader {
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
	profile, source, provenance := "yolo", any(nil), any(nil)
	switch mode {
	case "good":
	case "good_provenance":
		provenance = created
		rev8PreservedLoser(t, local, idA, created)
	case "bad_profile":
		profile = "standard"
	default:
		t.Fatalf("unknown fork mode %q", mode)
	}
	// A fork with a non-null source has no chained fixture: the
	// canonical shape owner pins fork profile_source_event_id to null
	// at append time, so the shared arm's presence check is
	// unreachable for forks and stays covered through launches.
	fork := appendEvent(t, local, idA, created, "fork.created", 2, rev10ForkPayload(ref.RecordID, profile, source, provenance))
	launched := appendEvent(t, local, idA, fork, "provider.launched", 3, rev8LaunchPayload("yolo", nil))
	idle := appendEvent(t, local, idA, launched, "session.idle", 4, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cp := checkpointRecordBytes(t, idA, 1, lease, hostA)
	if kind == "task_board" {
		cp = taskBoardCheckpointBytes(t, idA, 1, lease, hostA)
	}
	cp = rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cp)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", 5, map[string]any{"checkpoint_id": checkpointDigestOf(t, cp), "kind": "manual"})
	tail := appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	if ancestor {
		tail = appendEventAs(t, local, idA, tail, "provider.launched", 2, 2, leaseB, rev8LaunchPayload("yolo", nil))
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

// TestRev10ForkLocalProfile drives the fork.created new-session pair
// across persistence kinds, winner/necessary-ancestor paths, and
// refusal classes: a wrong profile value contradicts the admitted
// Session Record through all four entries. A non-null fork source
// cannot be chained (the canonical owner pins it at append), so it
// has no negative here; the shared presence check stays covered
// through launches.
func TestRev10ForkLocalProfile(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, mode := range []string{"good", "good_provenance", "bad_profile"} {
				t.Run(fmt.Sprintf("%s/ancestor=%t/%s", kind, ancestor, mode), func(t *testing.T) {
					r := rev10ForkFixture(t, kind, ancestor, mode)
					if mode == "good" || mode == "good_provenance" {
						rev7CheckEntries(t, r, false)
						return
					}
					rev8CheckIntegrityRefusal(t, r)
				})
			}
		}
	}
}

// TestRev10ForkOldPlanSubstitution proves an old plan revalidates fork
// authority instead of trusting its bound lease: the plan builds over
// a checkpoint whose heads predate the fork, then the reader
// substitutes a checkpoint whose valid heads cover the corrupt fork
// pair, and the old plan refuses with integrity failure.
func TestRev10ForkOldPlanSubstitution(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, mode := range []string{"bad_profile"} {
			t.Run(fmt.Sprintf("%s/%s", kind, mode), func(t *testing.T) {
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
				profile := "yolo"
				if mode == "bad_profile" {
					profile = "standard"
				}
				var source any
				fork := appendEvent(t, local, idA, created, "fork.created", 2, rev10ForkPayload(ref.RecordID, profile, source, nil))
				launched := appendEvent(t, local, idA, fork, "provider.launched", 3, rev8LaunchPayload("yolo", nil))
				idle := appendEvent(t, local, idA, launched, "session.idle", 4, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
				cp := checkpointRecordBytes(t, idA, 1, lease, hostA)
				if kind == "task_board" {
					cp = taskBoardCheckpointBytes(t, idA, 1, lease, hostA)
				}
				narrow := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{created}})
				covered := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{idle}})
				announced := appendEvent(t, local, idA, idle, "checkpoint.created", 5, map[string]any{"checkpoint_id": checkpointDigestOf(t, narrow), "kind": "manual"})
				_ = announced
				appendEventAs(t, local, idA, announced, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
				r := &Reader{Local: local, LocalHostID: hostA}
				withCheckpoints(r, narrow)
				withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, narrow)))
				rev7CheckEntries(t, r, false)
				old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
				rev7SwapCheckpoint(t, r, covered)
				if err := r.Revalidate(old); err == nil {
					t.Fatal("old-plan Revalidate admitted substituted fork authority")
				} else if !errors.Is(err, sessstate.ErrIntegrity) {
					t.Fatalf("old-plan Revalidate refusal = %v, want integrity_failure", err)
				}
				rev8CheckIntegrityRefusal(t, r)
			})
		}
	}
}

// rev10ResumeFixture builds a stop/resume session: created -> launch
// -> beforeChangeIdle -> E1 -> [E2] -> quiesced -> cpA(heads vary) ->
// announced -> stopped -> resumed(mode) -> idle -> cpB(heads idle) ->
// announcedB -> transfer. The ancestor variant extends the winning
// ancestry so the resumed checkpoint is validated through the
// necessary-ancestor path. Modes select the referenced checkpoint
// and the resumed pair:
//
//	good, good_divergent: cpA covers E1, resume cites (standard, E1);
//	  good_divergent also preserves a losing-lease change.
//	good_prechange: cpA predates E1, resume cites (yolo, null); the
//	  later outer change is never consulted.
//	bad_stale: cpA predates E1, resume cites (standard, E1).
//	bad_missing: resume names the zero digest, which no record binds,
//	  while carrying the creation pair.
//	bad_missing_stale: resume names the zero digest while carrying a
//	  non-creation pair; both missings refuse unavailable.
//	bad_wrong_session: resume names an idB checkpoint record.
//	bad_unknown_head: resume names a record whose head is unchained.
//	bad_non_newest: cpA covers E1 and E2, resume cites E1.
//	bad_losing: resume cites a preserved losing-lease change.
//	bad_creation_fallback: cpA covers E1, resume cites (yolo, null).
func rev10ResumeFixture(t *testing.T, kind string, ancestor bool, mode string) *Reader {
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
	checkpoint := checkpointRecordBytes
	if kind == "task_board" {
		checkpoint = taskBoardCheckpointBytes
	}
	seq := 0
	next := func() int {
		seq++
		return seq
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", next(), map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	launched := appendEvent(t, local, idA, created, "provider.launched", next(), rev8LaunchPayload("yolo", nil))
	beforeChangeIdle := appendEvent(t, local, idA, launched, "session.idle", next(), map[string]any{"boundary_ref": "before-change", "foreground_idle": true, "background_idle": true})
	changed := appendEvent(t, local, idA, beforeChangeIdle, "profile.changed", next(), rev8ChangedPayload("yolo", "standard"))
	var changed2 string
	if mode == "bad_non_newest" {
		changed2 = appendEvent(t, local, idA, changed, "profile.changed", next(), rev8ChangedPayload("standard", "yolo"))
	}
	var loser string
	if mode == "bad_losing" || mode == "good_divergent" {
		loser = rev8PreservedLoser(t, local, idA, changed)
	}
	quiescePred := changed
	if mode == "bad_non_newest" {
		quiescePred = changed2
	}
	quiesced := appendEvent(t, local, idA, quiescePred, "session.idle", next(), map[string]any{"boundary_ref": "turn-0", "foreground_idle": true, "background_idle": true})
	head := quiesced
	if mode == "good_prechange" || mode == "bad_stale" {
		head = beforeChangeIdle
	}
	cpA := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{head}})
	stoppedRef := checkpointDigestOf(t, cpA)
	reference := stoppedRef
	profile, source := "standard", any(changed)
	var extra []byte
	switch mode {
	case "good", "good_divergent":
	case "good_prechange":
		profile, source = "yolo", nil
	case "bad_stale":
	case "bad_missing":
		reference = zeroDigest
		profile, source = "yolo", nil
	case "bad_missing_stale":
		reference = zeroDigest
	case "bad_wrong_session":
		extra = checkpointRecordBytes(t, idB, 1, lease, hostA)
		reference = checkpointDigestOf(t, extra)
	case "bad_unknown_head":
		extra = rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{zeroDigest}})
		reference = checkpointDigestOf(t, extra)
	case "bad_non_newest":
	case "bad_losing":
		source = loser
	case "bad_creation_fallback":
		profile, source = "yolo", nil
	default:
		t.Fatalf("unknown resume mode %q", mode)
	}
	announced := appendEvent(t, local, idA, quiesced, "checkpoint.created", next(), map[string]any{"checkpoint_id": stoppedRef, "kind": "manual"})
	stopped := appendEvent(t, local, idA, announced, "session.stopped", next(), map[string]any{"graceful": true, "checkpoint_id": stoppedRef, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true})
	resumed := appendEvent(t, local, idA, stopped, "session.resumed", next(), rev10ResumePayload(reference, profile, source))
	idle := appendEvent(t, local, idA, resumed, "session.idle", next(), map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cpB := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	announcedB := appendEvent(t, local, idA, idle, "checkpoint.created", next(), map[string]any{"checkpoint_id": checkpointDigestOf(t, cpB), "kind": "manual"})
	tail := appendEventAs(t, local, idA, announcedB, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	r := &Reader{Local: local, LocalHostID: hostA}
	switch mode {
	case "bad_missing", "bad_missing_stale":
		withCheckpoints(r, cpB)
	case "bad_wrong_session", "bad_unknown_head":
		withCheckpoints(r, cpB, extra)
	default:
		withCheckpoints(r, cpB, cpA)
	}
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cpB)))
	if ancestor {
		launchProfile, launchSource := "standard", any(changed)
		if mode == "bad_non_newest" {
			launchProfile, launchSource = "yolo", changed2
		}
		tail = appendEventAs(t, local, idA, tail, "provider.launched", 2, 2, leaseB, rev8LaunchPayload(launchProfile, launchSource))
		tail = appendEventAs(t, local, idA, tail, "session.idle", 3, 2, leaseB, map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
		cp2 := rev7Replace(t, checkpoint(t, idA, 2, leaseB, hostA), "checkpoint_id", map[string]any{"event_heads": []string{tail}})
		withCheckpoints(r, cp2)
		withLeases(r, leaseSuccessorBytes(t, idA, 3, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, cp2)))
		tail = appendEventAs(t, local, idA, tail, "lease.transferred", 1, 3, leaseCycleB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": leaseB, "new_lease_id": leaseCycleB})
	}
	_ = tail
	return r
}

// TestRev10ResumeReferencedCheckpoint drives the session.resumed
// referenced-closure pair across persistence kinds,
// winner/necessary-ancestor paths, and refusal classes: a pair that
// contradicts its referenced closure refuses integrity failure,
// while a missing, wrong-session, or unresolvable reference refuses
// observation_unavailable. The good_prechange mode is the historical
// control: the resume cites its pre-change checkpoint pair while a
// later outer change exists and is never consulted.
func TestRev10ResumeReferencedCheckpoint(t *testing.T) {
	integrity := map[string]bool{"bad_stale": true, "bad_non_newest": true, "bad_losing": true, "bad_creation_fallback": true}
	unavailable := map[string]bool{"bad_missing": true, "bad_missing_stale": true, "bad_wrong_session": true, "bad_unknown_head": true}
	modes := []string{"good", "good_prechange", "good_divergent", "bad_stale", "bad_missing", "bad_missing_stale", "bad_wrong_session", "bad_unknown_head", "bad_non_newest", "bad_losing", "bad_creation_fallback"}
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, mode := range modes {
				t.Run(fmt.Sprintf("%s/ancestor=%t/%s", kind, ancestor, mode), func(t *testing.T) {
					r := rev10ResumeFixture(t, kind, ancestor, mode)
					switch {
					case integrity[mode]:
						rev8CheckIntegrityRefusal(t, r)
					case unavailable[mode]:
						rev10CheckUnavailableRefusal(t, r)
					default:
						rev7CheckEntries(t, r, false)
					}
				})
			}
		}
	}
}

// TestRev10ResumeOldPlanSubstitution proves an old plan revalidates
// resume authority instead of trusting its bound lease: the plan
// builds over a checkpoint whose closure covers only a valid resume,
// then the reader substitutes a checkpoint whose valid heads cover a
// second resume with a stale referenced pair, and the old plan
// refuses with integrity failure.
func TestRev10ResumeOldPlanSubstitution(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		t.Run(kind, func(t *testing.T) {
			local, _ := repository(t)
			raw := record(t, idA, "alpha")
			if kind == "task_board" {
				raw = taskBoardRecordBytes(t, idA, "alpha")
			}
			checkpoint := checkpointRecordBytes
			if kind == "task_board" {
				checkpoint = taskBoardCheckpointBytes
			}
			ref, err := local.CreateSession(raw)
			if err != nil {
				t.Fatal(err)
			}
			created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
			launched := appendEvent(t, local, idA, created, "provider.launched", 2, rev8LaunchPayload("yolo", nil))
			beforeChangeIdle := appendEvent(t, local, idA, launched, "session.idle", 3, map[string]any{"boundary_ref": "before-change", "foreground_idle": true, "background_idle": true})
			changed := appendEvent(t, local, idA, beforeChangeIdle, "profile.changed", 4, rev8ChangedPayload("yolo", "standard"))
			quiesced := appendEvent(t, local, idA, changed, "session.idle", 5, map[string]any{"boundary_ref": "turn-0", "foreground_idle": true, "background_idle": true})
			cpA := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{quiesced}})
			cpA0 := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{beforeChangeIdle}})
			announced := appendEvent(t, local, idA, quiesced, "checkpoint.created", 6, map[string]any{"checkpoint_id": checkpointDigestOf(t, cpA), "kind": "manual"})
			stopped := appendEvent(t, local, idA, announced, "session.stopped", 7, map[string]any{"graceful": true, "checkpoint_id": checkpointDigestOf(t, cpA), "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true})
			resumed := appendEvent(t, local, idA, stopped, "session.resumed", 8, rev10ResumePayload(checkpointDigestOf(t, cpA), "standard", changed))
			idle := appendEvent(t, local, idA, resumed, "session.idle", 9, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
			stopped2 := appendEvent(t, local, idA, idle, "session.stopped", 10, map[string]any{"graceful": true, "checkpoint_id": checkpointDigestOf(t, cpA0), "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true})
			resumed2 := appendEvent(t, local, idA, stopped2, "session.resumed", 11, rev10ResumePayload(checkpointDigestOf(t, cpA0), "standard", changed))
			idle2 := appendEvent(t, local, idA, resumed2, "session.idle", 12, map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
			narrow := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
			covered := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle2}})
			announcedB := appendEvent(t, local, idA, idle2, "checkpoint.created", 13, map[string]any{"checkpoint_id": checkpointDigestOf(t, narrow), "kind": "manual"})
			_ = announcedB
			appendEventAs(t, local, idA, announcedB, "lease.transferred", 1, 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
			r := &Reader{Local: local, LocalHostID: hostA}
			withCheckpoints(r, narrow, cpA, cpA0)
			withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, narrow)))
			rev7CheckEntries(t, r, false)
			old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
			rev7SwapCheckpoint(t, r, covered)
			if err := r.Revalidate(old); err == nil {
				t.Fatal("old-plan Revalidate admitted substituted resume authority")
			} else if !errors.Is(err, sessstate.ErrIntegrity) {
				t.Fatalf("old-plan Revalidate refusal = %v, want integrity_failure", err)
			}
			rev8CheckIntegrityRefusal(t, r)
		})
	}
}

// TestRev10ResumeReferencedWithdrawal proves an old plan re-resolves
// referenced resume authority instead of trusting its bound facts:
// withdrawing the referenced checkpoint record after the build makes
// the old plan refuse with observation_unavailable.
func TestRev10ResumeReferencedWithdrawal(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		t.Run(kind, func(t *testing.T) {
			r := rev10ResumeFixture(t, kind, false, "good")
			rev7CheckEntries(t, r, false)
			old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
			r.CheckpointRecords = append([][]byte(nil), r.CheckpointRecords[0])
			if err := r.Revalidate(old); err == nil {
				t.Fatal("old-plan Revalidate admitted withdrawn referenced authority")
			} else if !errors.Is(err, ErrObservationUnavailable) {
				t.Fatalf("old-plan Revalidate refusal = %v, want observation_unavailable", err)
			}
			rev10CheckUnavailableRefusal(t, r)
		})
	}
}

// rev10TaskBoardLaunchFixture builds a task_board session whose
// closure carries task_board.launched events: created -> tblaunch1
// -> E1 -> [E2] -> tblaunch2 -> idle with the checkpoint over idle,
// then the epoch-2 transfer. The good_historical mode appends a
// post-head change that the closure never consults. The ancestor
// variant extends the winning ancestry with a third launch under
// the successor lease.
func rev10TaskBoardLaunchFixture(t *testing.T, ancestor bool, mode string) *Reader {
	t.Helper()
	local, _ := repository(t)
	ref, err := local.CreateSession(taskBoardRecordBytes(t, idA, "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	seq, seqB := 0, 0
	next := func() int {
		seq++
		return seq
	}
	nextB := func() int {
		seqB++
		return seqB
	}
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", next(), map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	firstProfile, firstSource := "yolo", any(nil)
	switch mode {
	case "bad_first_profile":
		firstProfile = "standard"
	case "bad_first_source":
		firstSource = created
	}
	launched := appendEvent(t, local, idA, created, "task_board.launched", next(), rev10TaskBoardLaunchPayload(firstProfile, firstSource))
	changed := appendEvent(t, local, idA, launched, "profile.changed", next(), rev8ChangedPayload("yolo", "standard"))
	var changed2 string
	if mode == "bad_later_non_newest" {
		changed2 = appendEvent(t, local, idA, changed, "profile.changed", next(), rev8ChangedPayload("standard", "yolo"))
	}
	laterPred, laterProfile, laterSource := changed, "standard", any(changed)
	if mode == "bad_later_stale" {
		laterProfile, laterSource = "yolo", nil
	}
	if mode == "bad_later_non_newest" {
		laterPred = changed2
	}
	launched2 := appendEvent(t, local, idA, laterPred, "task_board.launched", next(), rev10TaskBoardLaunchPayload(laterProfile, laterSource))
	idle := appendEvent(t, local, idA, launched2, "session.idle", next(), map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	cp := rev7Replace(t, taskBoardCheckpointBytes(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{"event_heads": []string{idle}})
	r := &Reader{Local: local, LocalHostID: hostA}
	withCheckpoints(r, cp)
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cp)))
	announced := appendEvent(t, local, idA, idle, "checkpoint.created", next(), map[string]any{"checkpoint_id": checkpointDigestOf(t, cp), "kind": "manual"})
	tail := appendEventAs(t, local, idA, announced, "lease.transferred", nextB(), 2, leaseB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": lease, "new_lease_id": leaseB})
	var historical string
	if mode == "good_historical" {
		historical = appendEventAs(t, local, idA, tail, "profile.changed", nextB(), 2, leaseB, rev8ChangedPayload("standard", "yolo"))
		tail = historical
	}
	if ancestor {
		thirdProfile, thirdSource := "standard", any(changed)
		if mode == "bad_later_non_newest" || mode == "good_historical" {
			thirdProfile = "yolo"
			thirdSource = changed2
			if mode == "good_historical" {
				thirdSource = historical
			}
		}
		tail = appendEventAs(t, local, idA, tail, "task_board.launched", nextB(), 2, leaseB, rev10TaskBoardLaunchPayload(thirdProfile, thirdSource))
		tail = appendEventAs(t, local, idA, tail, "session.idle", nextB(), 2, leaseB, map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
		cp2 := rev7Replace(t, taskBoardCheckpointBytes(t, idA, 2, leaseB, hostA), "checkpoint_id", map[string]any{"event_heads": []string{tail}})
		withCheckpoints(r, cp2)
		withLeases(r, leaseSuccessorBytes(t, idA, 3, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, cp2)))
		tail = appendEventAs(t, local, idA, tail, "lease.transferred", 1, 3, leaseCycleB, map[string]any{"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA, "predecessor_lease_id": leaseB, "new_lease_id": leaseCycleB})
	}
	_ = tail
	return r
}

// TestRev10TaskBoardLaunchProfileAuthority drives the
// task_board.launched first/later pair across winner and
// necessary-ancestor paths: the first launch carries the creation
// pair, later launches carry the newest authoritative change, and
// every corrupt pair refuses integrity failure through all four
// entries. The good_historical mode is the later-local-only
// control: a post-head change is never consulted.
func TestRev10TaskBoardLaunchProfileAuthority(t *testing.T) {
	for _, ancestor := range []bool{false, true} {
		for _, mode := range []string{"good_first", "good_historical", "bad_first_profile", "bad_first_source", "bad_later_stale", "bad_later_non_newest"} {
			t.Run(fmt.Sprintf("ancestor=%t/%s", ancestor, mode), func(t *testing.T) {
				r := rev10TaskBoardLaunchFixture(t, ancestor, mode)
				if mode == "good_first" || mode == "good_historical" {
					rev7CheckEntries(t, r, false)
					return
				}
				rev8CheckIntegrityRefusal(t, r)
			})
		}
	}
}
