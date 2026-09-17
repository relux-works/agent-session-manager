// Referenced-checkpoint semantic-admission regression tests for
// TASK-260830-21gygk (SPEC 5.4 with referenced 5.2/2.4, CR10 P1):
// every session.resumed event consumes its referenced Checkpoint
// Record as authority, so the referenced record must pass the same
// semantic admission the winner and necessary-ancestor checkpoints
// pass (owning lease/fencing token, creator-holder, Session.kind
// persistence variant, event-head closure), not merely canonical
// parsing plus session equality and head membership. Every case
// drives the four shared production entries (BuildPlan, Revalidate
// fresh and old-plan, AuthoritativeStatus, AuthoritativeList),
// never the validators directly. TestReview10ReferencedCheckpointAdmission
// is ported verbatim from the CR10 independent reviewer probe.
package sessquery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func review10ReferenceFixture(t *testing.T, kind string, ancestor bool, mode string) *Reader {
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
	switch mode {
	case "wrong_creator":
		cpA = rev7Replace(t, cpA, "checkpoint_id", map[string]any{"created_by_host_id": hostB})
	case "wrong_lease":
		cpA = rev7Replace(t, cpA, "checkpoint_id", map[string]any{"lease_id": leaseCycleB})
	case "wrong_variant":
		changes := map[string]any{"provider_manifest_id": nil, "task_board_bundle_id": zeroDigest}
		if kind == "task_board" {
			changes = map[string]any{"provider_manifest_id": zeroDigest, "task_board_bundle_id": nil}
		}
		cpA = rev7Replace(t, cpA, "checkpoint_id", changes)
	}
	stoppedRef := checkpointDigestOf(t, cpA)
	reference := stoppedRef
	profile, source := "standard", any(changed)
	var extra []byte
	switch mode {
	case "good", "good_divergent", "wrong_creator", "wrong_variant", "wrong_lease":
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

func TestReview10ReferencedCheckpointAdmission(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, mode := range []string{"good", "wrong_creator", "wrong_variant", "wrong_lease"} {
				t.Run(fmt.Sprintf("%s/ancestor=%t/%s", kind, ancestor, mode), func(t *testing.T) {
					r := review10ReferenceFixture(t, kind, ancestor, mode)
					if mode == "good" {
						rev7CheckEntries(t, r, false)
						return
					}
					// Bind an old plan before the admitting closure reaches the resume.
					originalCP := append([][]byte(nil), r.CheckpointRecords...)
					originalLeases := append([][]byte(nil), r.LeaseRecords...)
					events, err := r.Local.ListEvents(idA)
					if err != nil {
						t.Fatal(err)
					}
					head := ""
					for _, e := range events {
						if e.EventType == "session.stopped" {
							head = e.EventID
							break
						}
					}
					replacements := map[string]string{}
					for i, cp := range r.CheckpointRecords {
						if i == 1 {
							continue
						}
						oldID := checkpointDigestOf(t, cp)
						narrow := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{head}})
						r.CheckpointRecords[i] = narrow
						replacements[oldID] = checkpointDigestOf(t, narrow)
					}
					for i, raw := range r.LeaseRecords {
						var v map[string]any
						if err := json.Unmarshal(raw, &v); err != nil {
							t.Fatal(err)
						}
						if cp, ok := v["checkpoint_id"].(string); ok {
							if next, ok := replacements[cp]; ok {
								r.LeaseRecords[i] = rev7Replace(t, raw, "record_id", map[string]any{"checkpoint_id": next})
							}
						}
					}
					rev7CheckEntries(t, r, false)
					old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
					r.CheckpointRecords = originalCP
					r.LeaseRecords = originalLeases
					t.Logf("old-plan Revalidate: %v", r.Revalidate(old))
					rev7CheckEntries(t, r, true)
				})
			}
		}
	}
}

// TestRev11ReferencedCheckpointAdmissionRefusalClass pins the
// referenced-admission refusal to incomplete authority: a
// wrong-creator, wrong-variant, or wrong-lease referenced record
// keeps a valid pair and closure yet refuses
// observation_unavailable through every shared entry, never a
// minted fact or a contradictory-history claim.
func TestRev11ReferencedCheckpointAdmissionRefusalClass(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, mode := range []string{"wrong_creator", "wrong_variant", "wrong_lease"} {
				t.Run(fmt.Sprintf("%s/ancestor=%t/%s", kind, ancestor, mode), func(t *testing.T) {
					r := review10ReferenceFixture(t, kind, ancestor, mode)
					rev10CheckUnavailableRefusal(t, r)
				})
			}
		}
	}
}

// TestRev11ReferencedCheckpointAdmissionOldPlan proves an old plan
// revalidates referenced authority instead of trusting its bound
// lease: the plan builds over a narrower closure ending at
// session.stopped, then the full referenced record is restored and
// the old plan refuses. The refusal is incomplete authority
// because the current referenced record cannot be admitted;
// a stale lease digest would only surface after admission.
func TestRev11ReferencedCheckpointAdmissionOldPlan(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, mode := range []string{"wrong_creator", "wrong_variant", "wrong_lease"} {
				t.Run(fmt.Sprintf("%s/ancestor=%t/%s", kind, ancestor, mode), func(t *testing.T) {
					r := review10ReferenceFixture(t, kind, ancestor, mode)
					originalCP := append([][]byte(nil), r.CheckpointRecords...)
					originalLeases := append([][]byte(nil), r.LeaseRecords...)
					events, err := r.Local.ListEvents(idA)
					if err != nil {
						t.Fatal(err)
					}
					head := ""
					for _, e := range events {
						if e.EventType == "session.stopped" {
							head = e.EventID
							break
						}
					}
					replacements := map[string]string{}
					for i, cp := range r.CheckpointRecords {
						if i == 1 {
							continue
						}
						oldID := checkpointDigestOf(t, cp)
						narrow := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{head}})
						r.CheckpointRecords[i] = narrow
						replacements[oldID] = checkpointDigestOf(t, narrow)
					}
					for i, raw := range r.LeaseRecords {
						var v map[string]any
						if err := json.Unmarshal(raw, &v); err != nil {
							t.Fatal(err)
						}
						if cp, ok := v["checkpoint_id"].(string); ok {
							if next, ok := replacements[cp]; ok {
								r.LeaseRecords[i] = rev7Replace(t, raw, "record_id", map[string]any{"checkpoint_id": next})
							}
						}
					}
					rev7CheckEntries(t, r, false)
					old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
					r.CheckpointRecords = originalCP
					r.LeaseRecords = originalLeases
					if err := r.Revalidate(old); err == nil {
						t.Fatal("old-plan Revalidate admitted invalid referenced checkpoint authority")
					} else if !errors.Is(err, ErrObservationUnavailable) {
						t.Fatalf("old-plan Revalidate refusal = %v, want observation_unavailable", err)
					}
					rev7CheckEntries(t, r, true)
				})
			}
		}
	}
}

// review11TemporalFixture extends the referenced-checkpoint admission fixture
// with a second lease. The future-owner cases make the referenced checkpoint
// semantically belong to a later lease than the session.resumed event that
// consumes it. The checkpoint may still have a canonical identity and an
// otherwise valid owner tuple somewhere in the winning ancestry; that is the
// authority relationship this regression protects. `good_prechange` is a
// distinct admitted historical pair whose checkpoint head predates the
// profile change and whose resume carries yolo/null. `good_divergent` keeps a
// losing-lease profile-change branch out of the authoritative index while
// admitting the standard/changed pair on the winning branch.
func review11TemporalFixture(t *testing.T, kind string, ancestor bool, mode string) *Reader {
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
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", next(), map[string]any{
		"session_record_id":             ref.RecordID,
		"bootstrap_operation_id":        hostA,
		"first_checkpoint_operation_id": hostB,
	})
	launched := appendEvent(t, local, idA, created, "provider.launched", next(), rev8LaunchPayload("yolo", nil))
	beforeChangeIdle := appendEvent(t, local, idA, launched, "session.idle", next(), map[string]any{
		"boundary_ref":    "before-change",
		"foreground_idle": true,
		"background_idle": true,
	})
	changed := appendEvent(t, local, idA, beforeChangeIdle, "profile.changed", next(), rev8ChangedPayload("yolo", "standard"))
	var loser string
	if mode == "good_divergent" {
		loser = rev8PreservedLoser(t, local, idA, changed)
		if loser == "" {
			t.Fatal("good_divergent did not preserve a losing-lease branch")
		}
	}
	quiesced := appendEvent(t, local, idA, changed, "session.idle", next(), map[string]any{
		"boundary_ref":    "turn-0",
		"foreground_idle": true,
		"background_idle": true,
	})
	head := quiesced
	if mode == "good_prechange" {
		head = beforeChangeIdle
	}
	cpA := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{
		"event_heads": []string{head},
	})
	switch mode {
	case "future_owner":
		cpA = rev7Replace(t, cpA, "checkpoint_id", map[string]any{"lease_epoch": 2, "lease_id": leaseB})
	case "epoch_mismatch":
		cpA = rev7Replace(t, cpA, "checkpoint_id", map[string]any{"lease_epoch": 2})
	}
	stoppedRef := checkpointDigestOf(t, cpA)
	reference := stoppedRef
	profile, source := "standard", any(changed)
	var extra []byte
	switch mode {
	case "good", "good_divergent", "future_owner", "epoch_mismatch", "later_resume":
	case "good_prechange":
		profile, source = "yolo", nil
	case "bad_wrong_session":
		extra = checkpointRecordBytes(t, idB, 1, lease, hostA)
		reference = checkpointDigestOf(t, extra)
	case "bad_unknown_head":
		extra = rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{
			"event_heads": []string{zeroDigest},
		})
		reference = checkpointDigestOf(t, extra)
	case "bad_missing":
		reference = zeroDigest
		profile, source = "yolo", nil
	default:
		t.Fatalf("unknown resume mode %q", mode)
	}
	announced := appendEvent(t, local, idA, quiesced, "checkpoint.created", next(), map[string]any{
		"checkpoint_id": stoppedRef,
		"kind":          "manual",
	})
	stopped := appendEvent(t, local, idA, announced, "session.stopped", next(), map[string]any{
		"graceful":       true,
		"checkpoint_id":  stoppedRef,
		"resumable":      true,
		"closure_kind":   "checkpointed",
		"process_closed": true,
		"store_closed":   true,
	})
	resumed := appendEvent(t, local, idA, stopped, "session.resumed", next(), rev10ResumePayload(reference, profile, source))
	idle := appendEvent(t, local, idA, resumed, "session.idle", next(), map[string]any{
		"boundary_ref":    "turn-1",
		"foreground_idle": true,
		"background_idle": true,
	})
	cpB := rev7Replace(t, checkpoint(t, idA, 1, lease, hostA), "checkpoint_id", map[string]any{
		"event_heads": []string{idle},
	})
	announcedB := appendEvent(t, local, idA, idle, "checkpoint.created", next(), map[string]any{
		"checkpoint_id": checkpointDigestOf(t, cpB),
		"kind":          "manual",
	})
	tail := appendEventAs(t, local, idA, announcedB, "lease.transferred", 1, 2, leaseB, map[string]any{
		"operation_id":         hostB,
		"from_host_id":         hostA,
		"to_host_id":           hostA,
		"predecessor_lease_id": lease,
		"new_lease_id":         leaseB,
	})
	if mode == "later_resume" {
		tail = appendEventAs(t, local, idA, tail, "session.resumed", 2, 2, leaseB, rev10ResumePayload(stoppedRef, "standard", changed))
		tail = appendEventAs(t, local, idA, tail, "session.idle", 3, 2, leaseB, map[string]any{
			"boundary_ref":    "later-lease-resume",
			"foreground_idle": true,
			"background_idle": true,
		})
	}
	r := &Reader{Local: local, LocalHostID: hostA}
	switch mode {
	case "bad_missing":
		withCheckpoints(r, cpB)
	case "bad_wrong_session", "bad_unknown_head":
		withCheckpoints(r, cpB, extra)
	default:
		withCheckpoints(r, cpB, cpA)
	}
	withLeases(r, leaseCreate(t, idA, hostA), leaseSuccessorBytes(t, idA, 2, leaseB, hostA, lease, checkpointDigestOf(t, cpB)))
	if ancestor {
		launchSequence, idleSequence := 2, 3
		if mode == "later_resume" {
			launchSequence, idleSequence = 4, 5
		}
		tail = appendEventAs(t, local, idA, tail, "provider.launched", launchSequence, 2, leaseB, rev8LaunchPayload("standard", changed))
		tail = appendEventAs(t, local, idA, tail, "session.idle", idleSequence, 2, leaseB, map[string]any{
			"boundary_ref":    "turn-2",
			"foreground_idle": true,
			"background_idle": true,
		})
		cp2 := rev7Replace(t, checkpoint(t, idA, 2, leaseB, hostA), "checkpoint_id", map[string]any{
			"event_heads": []string{tail},
		})
		withCheckpoints(r, cp2)
		withLeases(r, leaseSuccessorBytes(t, idA, 3, leaseCycleB, hostA, leaseB, checkpointDigestOf(t, cp2)))
		tail = appendEventAs(t, local, idA, tail, "lease.transferred", 1, 3, leaseCycleB, map[string]any{
			"operation_id":         hostB,
			"from_host_id":         hostA,
			"to_host_id":           hostA,
			"predecessor_lease_id": leaseB,
			"new_lease_id":         leaseCycleB,
		})
	}
	_ = tail
	return r
}

// TestRev12ReferencedCheckpointLaterLease proves the allowed temporal case:
// a resume in a later lease may consume a checkpoint owned by an earlier
// lease, provided its event-head closure is on the resume's predecessor path.
// The shared BuildPlan, fresh/old-plan Revalidate, AuthoritativeStatus, and
// AuthoritativeList callers are exercised by rev7CheckEntries.
func TestRev12ReferencedCheckpointLaterLease(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		t.Run(kind, func(t *testing.T) {
			r := review11TemporalFixture(t, kind, true, "later_resume")
			rev7CheckEntries(t, r, false)
		})
	}
}

func TestReview11ReferencedTemporalAuthority(t *testing.T) {
	for _, kind := range []string{"direct", "task_board"} {
		for _, ancestor := range []bool{false, true} {
			for _, mode := range []string{"good", "good_prechange", "good_divergent", "future_owner", "epoch_mismatch"} {
				t.Run(fmt.Sprintf("%s/ancestor=%t/%s", kind, ancestor, mode), func(t *testing.T) {
					r := review11TemporalFixture(t, kind, ancestor, mode)
					if mode == "good" || mode == "good_prechange" || mode == "good_divergent" {
						rev7CheckEntries(t, r, false)
						return
					}
					originalCP := append([][]byte(nil), r.CheckpointRecords...)
					originalLeases := append([][]byte(nil), r.LeaseRecords...)
					events, err := r.Local.ListEvents(idA)
					if err != nil {
						t.Fatal(err)
					}
					head := ""
					for _, event := range events {
						if event.EventType == "session.stopped" {
							head = event.EventID
							break
						}
					}
					replacements := map[string]string{}
					for i, cp := range r.CheckpointRecords {
						if i == 1 {
							continue
						}
						oldID := checkpointDigestOf(t, cp)
						narrow := rev7Replace(t, cp, "checkpoint_id", map[string]any{"event_heads": []string{head}})
						r.CheckpointRecords[i] = narrow
						replacements[oldID] = checkpointDigestOf(t, narrow)
					}
					for i, raw := range r.LeaseRecords {
						var value map[string]any
						if err := json.Unmarshal(raw, &value); err != nil {
							t.Fatal(err)
						}
						if cp, ok := value["checkpoint_id"].(string); ok {
							if next, ok := replacements[cp]; ok {
								r.LeaseRecords[i] = rev7Replace(t, raw, "record_id", map[string]any{"checkpoint_id": next})
							}
						}
					}
					rev7CheckEntries(t, r, false)
					old := mustBuild(t, r, PlanArgs{Selector: "alpha", Action: ActionStatus})
					r.CheckpointRecords = originalCP
					r.LeaseRecords = originalLeases
					oldErr := r.Revalidate(old)
					t.Logf("old-plan refusal: %v", oldErr)
					if oldErr == nil {
						t.Error("old-plan admitted invalid temporal authority")
					}
					rev7CheckEntries(t, r, true)
				})
			}
		}
	}
}

// TestRev11RecordConsumptionCensus is the type-level gate for the
// record-consumption table. Canonical parsing produces a raw
// validatedCheckpoint, but only the shared admission owners may consume that
// representation. Every profile derivation entry point accepts the distinct
// admittedCheckpoint capability, and the gate checks object identity plus
// expression types across the whole package rather than trusting a call-token
// census in one file.
func TestRev11RecordConsumptionCensus(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	if err := rev12CheckRecordConsumptionSource(filepath.Dir(file)); err != nil {
		t.Fatal(err)
	}
}

// TestRev12RecordConsumptionCensusRejectsAlternatePaths keeps the type-level
// gate honest against the review11 bypass and three alternate shapes: a
// new-file helper, a method receiver, and a closure-returning factory. The
// current package is copied into isolated scratch directories, each plant is
// type-checked by the same gate, and every plant must be rejected.
func TestRev12RecordConsumptionCensusRejectsAlternatePaths(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	packageDir := filepath.Dir(file)
	plants := map[string]string{
		"review11_bypass.go": `package sessquery

import "github.com/relux-works/agent-session-manager/internal/sessrepo"

func review11UseUnchecked(checkpoint validatedCheckpoint, summary sessrepo.EventSummary, repo *sessrepo.Repository, checkpoints map[string]validatedCheckpoint) bool {
	copied, ok := checkpoints[summary.EventID]
	return ok && copied.CreatorHostID == checkpoint.CreatorHostID && repo != nil
}
`,
		"review12_new_file.go": `package sessquery

func review12NewFileHelper(checkpoints map[string]validatedCheckpoint, id string) string {
	return checkpoints[id].Digest
}
`,
		"review12_method_receiver.go": `package sessquery

type review12Receiver struct {
	checkpoints map[string]validatedCheckpoint
}

func (receiver review12Receiver) digest(id string) string {
	return receiver.checkpoints[id].Digest
}
`,
		"review12_closure_factory.go": `package sessquery

func review12ClosureFactory(checkpoints map[string]validatedCheckpoint) func(string) string {
	return func(id string) string {
		return checkpoints[id].Digest
	}
}
`,
		"review12_inferred_loader.go": `package sessquery

func review12InferredLoader(reader *Reader, id string) string {
	checkpoints, _ := reader.validatedCheckpointsByDigest()
	return checkpoints[id].Digest
}
`,
	}
	for name, plant := range plants {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			rev12CopyProductionSources(t, packageDir, dir)
			if err := os.WriteFile(filepath.Join(dir, name), []byte(plant), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := rev12CheckRecordConsumptionSource(dir); err == nil {
				t.Fatalf("type-level record-consumption gate admitted %s", name)
			} else {
				t.Logf("rejected %s: %v", name, err)
			}
		})
	}
}

// TestRev12AdmissionPrecisionCensus retains the small lexical checks as
// precision evidence only. The behavioral temporal and semantic suites are
// the correctness evidence; this test merely pins the intended call edges
// and the fact that admission precedes referenced-closure derivation.
func TestRev12AdmissionPrecisionCensus(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	leasePath := filepath.Join(filepath.Dir(file), "lease.go")
	source, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, leasePath, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	callsOf := func(functionName string) map[string][]token.Pos {
		calls := map[string][]token.Pos{}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != functionName || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch called := call.Fun.(type) {
				case *ast.Ident:
					name = called.Name
				case *ast.SelectorExpr:
					name = called.Sel.Name
				}
				if name != "" {
					calls[name] = append(calls[name], call.Pos())
				}
				return true
			})
		}
		return calls
	}
	requireCall := func(caller, callee string) {
		t.Helper()
		if len(callsOf(caller)[callee]) == 0 {
			t.Errorf("%s must call %s", caller, callee)
		}
	}
	requireCall("checkClosureProfilePairs", "referencedCheckpointProfile")
	requireCall("checkCheckpointProfileAuthority", "checkClosureProfilePairs")
	requireCall("checkWinnerCheckpoint", "admitCheckpoint")
	requireCall("checkCheckpointBinding", "admitCheckpoint")
	requireCall("buildCheckpointAdmission", "checkReferencedCheckpointBinding")
	requireCall("checkReferencedCheckpointBinding", "admitCheckpoint")
	requireCall("admitCheckpoint", "checkCheckpointPersistence")
	requireCall("admitCheckpoint", "checkCheckpointEventHeads")
	requireCall("admitCheckpoint", "checkReferencedCheckpointTemporal")
	profileCalls := callsOf("referencedCheckpointProfile")
	admissionPos := profileCalls["admission"]
	closurePos := profileCalls["profileClosure"]
	if len(admissionPos) != 1 || len(closurePos) != 1 {
		t.Fatalf("referencedCheckpointProfile calls = admission:%d profileClosure:%d, want one of each", len(admissionPos), len(closurePos))
	}
	if fset.Position(admissionPos[0]).Line > fset.Position(closurePos[0]).Line {
		t.Fatalf("referenced checkpoint admission must precede profile closure: admission line %d, closure line %d", fset.Position(admissionPos[0]).Line, fset.Position(closurePos[0]).Line)
	}
}

var rev12RawAdmissionOwners = map[string]bool{
	"parseCheckpointRecord":             true,
	"validatedCheckpointsByDigest":      true,
	"checkWinnerCheckpoint":             true,
	"checkAncestorCheckpoints":          true,
	"checkCheckpointBinding":            true,
	"buildCheckpointAdmission":          true,
	"checkReferencedCheckpointBinding":  true,
	"checkReferencedCheckpointTemporal": true,
	"admitCheckpoint":                   true,
	"checkCheckpointEventHeads":         true,
	"checkCheckpointPersistence":        true,
}

var rev12ProfileDerivationEntries = []string{
	"checkCheckpointProfileAuthority",
	"sessionCreationProfile",
	"profileClosure",
	"eventProfilePair",
	"eventProfileTarget",
	"eventPayloadMembers",
	"checkClosureProfilePairs",
	"referencedCheckpointProfile",
	"checkProfilePairSource",
}

// rev12CheckRecordConsumptionSource parses and type-checks all production
// files in one package, then enforces the raw-to-admitted boundary using
// exact go/types object identity and expression types. The historical helper
// name is retained for the earlier regression tests; the current census is
// the rev14 sealed-capability gate. A new file, receiver method, closure, or
// inferred raw loader result is still inside the checked package and is
// rejected.
func rev12CheckRecordConsumptionSource(packageDir string) error {
	archives, err := rev12LoadImportArchives()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return fmt.Errorf("read sessquery sources: %w", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(packageDir, name)
		parsed, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", name, parseErr)
		}
		files = append(files, parsed)
	}
	if len(files) == 0 {
		return fmt.Errorf("no production Go sources in %s", packageDir)
	}
	errs := make([]error, 0)
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	conf := types.Config{
		Importer: importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
			archive := archives[path]
			if archive == "" {
				return nil, fmt.Errorf("no export archive for %s", path)
			}
			return os.Open(archive)
		}),
		Error: func(typeErr error) {
			errs = append(errs, typeErr)
		},
	}
	pkg, checkErr := conf.Check("github.com/relux-works/agent-session-manager/internal/sessquery", fset, files, info)
	if checkErr != nil {
		errs = append(errs, checkErr)
	}
	if len(errs) > 0 {
		return fmt.Errorf("type-check record-consumption package: %v", errs[0])
	}
	rawObject := pkg.Scope().Lookup("validatedCheckpoint")
	rawType, ok := rawObject.(*types.TypeName)
	if !ok {
		return fmt.Errorf("validatedCheckpoint type object is missing")
	}
	admittedObject := pkg.Scope().Lookup("admittedCheckpoint")
	admittedType, ok := admittedObject.(*types.TypeName)
	if !ok {
		return fmt.Errorf("admittedCheckpoint type object is missing")
	}
	sealObject, ok := pkg.Scope().Lookup("checkpointSeal").(*types.TypeName)
	if !ok {
		return fmt.Errorf("checkpointSeal type object is missing")
	}
	tokenObject, ok := pkg.Scope().Lookup("checkpointSealToken").(*types.TypeName)
	if !ok {
		return fmt.Errorf("checkpointSealToken type object is missing")
	}
	functions := rev14CollectFunctionContexts(files, info)
	producer := rev14PackageFunction(pkg, "admitCheckpoint")
	if producer == nil {
		return fmt.Errorf("free admitCheckpoint constructor is missing")
	}
	rawOwners := rev14FunctionSet(pkg, rev12RawAdmissionOwners, functions)
	if len(rawOwners) != len(rev12RawAdmissionOwners) {
		return fmt.Errorf("one or more raw admission owners are missing")
	}
	profileNames := make(map[string]bool, len(rev12ProfileDerivationEntries))
	for _, name := range rev12ProfileDerivationEntries {
		profileNames[name] = true
	}
	profileEntries := rev14FunctionSet(pkg, profileNames, functions)
	if len(profileEntries) != len(rev12ProfileDerivationEntries) {
		return fmt.Errorf("one or more profile derivation entries are missing")
	}
	buildAdmission := rev14PackageFunction(pkg, "buildCheckpointAdmission")
	referencedBinding := rev14PackageFunction(pkg, "checkReferencedCheckpointBinding")
	authorityMethod := rev14MethodFunction(functions, "authority")
	allowedSignatures := make(map[*types.Func]bool, len(profileEntries)+3)
	for function := range profileEntries {
		allowedSignatures[function] = true
	}
	allowedSignatures[producer] = true
	if buildAdmission != nil {
		allowedSignatures[buildAdmission] = true
	}
	if referencedBinding != nil {
		allowedSignatures[referencedBinding] = true
	}
	if authorityMethod != nil {
		allowedSignatures[authorityMethod] = true
	}

	checkpointAdmissionType, _ := pkg.Scope().Lookup("checkpointAdmission").(*types.TypeName)
	allowedTypeSpans := rev14CheckpointAdmissionTypeSpans(files, info, checkpointAdmissionType)
	callbackParameters := rev14CheckpointAdmissionParameters(profileEntries, checkpointAdmissionType)
	violations := make([]string, 0)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			context := rev14EnclosingFunction(node.Pos(), functions)
			if ident, isIdent := node.(*ast.Ident); isIdent {
				object := info.Uses[ident]
				if object == nil {
					object = info.Defs[ident]
				}
				if object == rawType && ident.Pos() != rawType.Pos() && !rev14RawOwner(context, rawOwners, functions, buildAdmission) {
					violations = append(violations, fmt.Sprintf("raw validatedCheckpoint use at %s outside an exact admission owner", fset.Position(ident.Pos())))
				}
				if object == admittedType && ident.Pos() != admittedType.Pos() && !rev14AdmittedSignatureUse(ident.Pos(), context, functions, allowedSignatures, buildAdmission, allowedTypeSpans) {
					violations = append(violations, fmt.Sprintf("admittedCheckpoint type use at %s outside an approved signature", fset.Position(ident.Pos())))
				}
			}
			if valueSpec, isValueSpec := node.(*ast.ValueSpec); isValueSpec && valueSpec.Type != nil && rev14SealedTypeExpr(valueSpec.Type, info, sealObject, tokenObject) && !rev14SealSignatureAllowed(context, producer, authorityMethod) {
				violations = append(violations, fmt.Sprintf("checkpoint seal/token variable at %s is outside the constructor and verifier", fset.Position(valueSpec.Pos())))
			}
			if functionType, isFunctionType := node.(*ast.FuncType); isFunctionType && rev14SealedFuncType(functionType, info, sealObject, tokenObject) && !rev14SealSignatureAllowed(context, producer, authorityMethod) {
				violations = append(violations, fmt.Sprintf("checkpoint seal/token signature at %s is outside the constructor and verifier", fset.Position(functionType.Pos())))
			}
			if literal, isLiteral := node.(*ast.CompositeLit); isLiteral {
				literalType := info.Types[literal].Type
				if rev12IsNamedType(literalType, admittedType) && !rev14ExactFunction(context, producer) {
					violations = append(violations, fmt.Sprintf("admitted checkpoint construction at %s outside the free admitCheckpoint constructor", fset.Position(literal.Pos())))
				}
				if rev14NamedTypeOrPointer(literalType, sealObject) && !rev14ExactFunction(context, producer) {
					violations = append(violations, fmt.Sprintf("checkpoint seal construction at %s outside the free admitCheckpoint constructor", fset.Position(literal.Pos())))
				}
				if rev14NamedTypeOrPointer(literalType, tokenObject) && !rev14ExactFunction(context, producer) {
					violations = append(violations, fmt.Sprintf("checkpoint seal token construction at %s outside the free admitCheckpoint constructor", fset.Position(literal.Pos())))
				}
			}
			if expression, isExpr := node.(ast.Expr); isExpr && !rev14TypeSyntax(expression, info) && rev14SealedValueContains(info.TypeOf(expression), sealObject, tokenObject) && !rev14SealValueAllowed(context, producer, authorityMethod, profileEntries, rawOwners, functions, buildAdmission) {
				violations = append(violations, fmt.Sprintf("checkpoint seal/token value at %s escapes the constructor/verifier flow", fset.Position(expression.Pos())))
			}
			if call, isCall := node.(*ast.CallExpr); isCall && rev14SealedTypeInCallTarget(call, info, sealObject, tokenObject) && !rev14ExactFunction(context, producer) {
				violations = append(violations, fmt.Sprintf("checkpoint seal/token construction target at %s is outside the free admitCheckpoint constructor", fset.Position(call.Pos())))
			}
			if expression, isExpr := node.(ast.Expr); isExpr && !rev14TypeSyntax(expression, info) && rev14TypeContains(info.TypeOf(expression), rawType) && !rev14RawOwner(context, rawOwners, functions, buildAdmission) {
				violations = append(violations, fmt.Sprintf("raw checkpoint value at %s outside an exact admission owner", fset.Position(expression.Pos())))
			}
			if selector, isSelector := node.(*ast.SelectorExpr); isSelector && selector.Sel.Name == "seal" {
				if rev12IsNamedType(info.TypeOf(selector.X), admittedType) && !rev14ExactMethod(context, authorityMethod) && !rev14ExactFunction(context, producer) {
					violations = append(violations, fmt.Sprintf("admitted checkpoint seal access at %s outside the seal verifier", fset.Position(selector.Pos())))
				}
			}
			if call, isCall := node.(*ast.CallExpr); isCall {
				callee := rev14Callee(call.Fun, info)
				if rev14AdmittedValueContains(info.TypeOf(call), admittedType, checkpointAdmissionType) && !rev14AdmittedResultCallAllowed(callee, context, rawOwners, producer, referencedBinding, profileEntries, callbackParameters, buildAdmission, functions) {
					violations = append(violations, fmt.Sprintf("admitted checkpoint result at %s escapes an approved flow", fset.Position(call.Pos())))
				}
				for _, argument := range call.Args {
					if rev14AdmittedValueContains(info.TypeOf(argument), admittedType, checkpointAdmissionType) && !rev14AdmittedArgumentAllowed(callee, context, rawOwners, profileEntries, authorityMethod, buildAdmission, functions) {
						violations = append(violations, fmt.Sprintf("admitted checkpoint value at %s enters an unapproved call", fset.Position(argument.Pos())))
					}
				}
			}
			if assignment, isAssignment := node.(*ast.AssignStmt); isAssignment {
				for _, left := range assignment.Lhs {
					if selector, ok := left.(*ast.SelectorExpr); ok && rev14SealedReceiver(info.TypeOf(selector.X), sealObject, tokenObject) && !rev14SealSignatureAllowed(context, producer, authorityMethod) {
						violations = append(violations, fmt.Sprintf("checkpoint seal/token field mutation at %s is outside the constructor and verifier", fset.Position(selector.Pos())))
					}
				}
				for _, rhs := range assignment.Rhs {
					if rev14AdmittedValueContains(info.TypeOf(rhs), admittedType, checkpointAdmissionType) && !rev14DirectAdmittedCall(rhs, info, context, rawOwners, producer, referencedBinding, profileEntries, callbackParameters, buildAdmission, functions) {
						violations = append(violations, fmt.Sprintf("admitted checkpoint copy at %s is outside an approved flow", fset.Position(rhs.Pos())))
					}
				}
			}
			if incDec, isIncDec := node.(*ast.IncDecStmt); isIncDec {
				if selector, ok := incDec.X.(*ast.SelectorExpr); ok && rev14SealedReceiver(info.TypeOf(selector.X), sealObject, tokenObject) && !rev14SealSignatureAllowed(context, producer, authorityMethod) {
					violations = append(violations, fmt.Sprintf("checkpoint seal/token field mutation at %s is outside the constructor and verifier", fset.Position(selector.Pos())))
				}
			}
			if returned, isReturn := node.(*ast.ReturnStmt); isReturn {
				for _, result := range returned.Results {
					if rev14AdmittedValueContains(info.TypeOf(result), admittedType, checkpointAdmissionType) && !rev14ExactFunction(context, producer) && !rev14DirectAdmittedCall(result, info, context, rawOwners, producer, referencedBinding, profileEntries, callbackParameters, buildAdmission, functions) {
						violations = append(violations, fmt.Sprintf("admitted checkpoint return at %s is outside an approved flow", fset.Position(result.Pos())))
					}
				}
			}
			return true
		})
	}
	for _, functionName := range rev12ProfileDerivationEntries {
		object := pkg.Scope().Lookup(functionName)
		function, ok := object.(*types.Func)
		if !ok {
			violations = append(violations, fmt.Sprintf("profile derivation entry %s is missing", functionName))
			continue
		}
		signature, ok := function.Type().(*types.Signature)
		if !ok || signature.Params().Len() == 0 || !rev12IsNamedType(signature.Params().At(0).Type(), admittedType) {
			violations = append(violations, fmt.Sprintf("profile derivation entry %s does not accept admittedCheckpoint", functionName))
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("record-consumption gate violations: %s", strings.Join(violations, "; "))
	}
	return nil
}

type rev14FunctionContext struct {
	object     *types.Func
	name       string
	start, end token.Pos
	isClosure  bool
}

func rev14CollectFunctionContexts(files []*ast.File, info *types.Info) []rev14FunctionContext {
	var functions []rev14FunctionContext
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch function := node.(type) {
			case *ast.FuncDecl:
				object, _ := info.Defs[function.Name].(*types.Func)
				functions = append(functions, rev14FunctionContext{
					object: object,
					name:   function.Name.Name,
					start:  function.Pos(),
					end:    function.End(),
				})
			case *ast.FuncLit:
				functions = append(functions, rev14FunctionContext{
					name:      "<closure>",
					start:     function.Pos(),
					end:       function.End(),
					isClosure: true,
				})
			}
			return true
		})
	}
	return functions
}

func rev14EnclosingFunction(position token.Pos, functions []rev14FunctionContext) *rev14FunctionContext {
	var match *rev14FunctionContext
	for index := range functions {
		candidate := &functions[index]
		if position < candidate.start || position > candidate.end {
			continue
		}
		if match == nil || candidate.end-candidate.start < match.end-match.start {
			match = candidate
		}
	}
	return match
}

func rev14ParentFunction(position token.Pos, child *rev14FunctionContext, functions []rev14FunctionContext) *rev14FunctionContext {
	var match *rev14FunctionContext
	for index := range functions {
		candidate := &functions[index]
		if candidate == child || position < candidate.start || position > candidate.end {
			continue
		}
		if match == nil || candidate.end-candidate.start < match.end-match.start {
			match = candidate
		}
	}
	return match
}

func rev14PackageFunction(pkg *types.Package, name string) *types.Func {
	function, _ := pkg.Scope().Lookup(name).(*types.Func)
	if function == nil || function.Pkg() != pkg {
		return nil
	}
	if signature, _ := function.Type().(*types.Signature); signature != nil && signature.Recv() != nil {
		return nil
	}
	return function
}

func rev14FunctionSet(pkg *types.Package, names map[string]bool, contexts []rev14FunctionContext) map[*types.Func]bool {
	functions := make(map[*types.Func]bool, len(names))
	for name := range names {
		if function := rev14PackageFunction(pkg, name); function != nil {
			functions[function] = true
			continue
		}
		for _, context := range contexts {
			if context.name == name && context.object != nil {
				signature, _ := context.object.Type().(*types.Signature)
				if signature != nil && signature.Recv() != nil {
					functions[context.object] = true
					break
				}
			}
		}
	}
	return functions
}

func rev14MethodFunction(functions []rev14FunctionContext, name string) *types.Func {
	for _, function := range functions {
		if function.name != name || function.object == nil {
			continue
		}
		signature, _ := function.object.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			return function.object
		}
	}
	return nil
}

func rev14ExactFunction(context *rev14FunctionContext, want *types.Func) bool {
	return context != nil && want != nil && context.object == want
}

func rev14ExactMethod(context *rev14FunctionContext, want *types.Func) bool {
	return rev14ExactFunction(context, want)
}

func rev14RawOwner(context *rev14FunctionContext, owners map[*types.Func]bool, functions []rev14FunctionContext, buildAdmission *types.Func) bool {
	if context == nil {
		return false
	}
	if context.object != nil && owners[context.object] {
		return true
	}
	return context.isClosure && rev14ContextInFunction(context, buildAdmission, functions)
}

func rev14ContextInFunction(context *rev14FunctionContext, function *types.Func, functions []rev14FunctionContext) bool {
	if context == nil || function == nil {
		return false
	}
	if context.object == function {
		return true
	}
	if !context.isClosure {
		return false
	}
	parent := rev14ParentFunction(context.start, context, functions)
	return parent != nil && parent.object == function
}

func rev14AdmittedSignatureUse(position token.Pos, context *rev14FunctionContext, functions []rev14FunctionContext, allowed map[*types.Func]bool, buildAdmission *types.Func, typeSpans [][2]token.Pos) bool {
	for _, span := range typeSpans {
		if position >= span[0] && position <= span[1] {
			return true
		}
	}
	if context == nil {
		return false
	}
	if allowed[context.object] {
		return true
	}
	return context.isClosure && rev14ContextInFunction(context, buildAdmission, functions)
}

func rev14CheckpointAdmissionTypeSpans(files []*ast.File, info *types.Info, want types.Object) [][2]token.Pos {
	if want == nil {
		return nil
	}
	var spans [][2]token.Pos
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, ok := node.(*ast.TypeSpec)
			if !ok || info.Defs[typeSpec.Name] != want {
				return true
			}
			if functionType, ok := typeSpec.Type.(*ast.FuncType); ok {
				spans = append(spans, [2]token.Pos{functionType.Pos(), functionType.End()})
			}
			return true
		})
	}
	return spans
}

func rev14NamedTypeOrPointer(typ types.Type, want types.Object) bool {
	if want == nil || typ == nil {
		return false
	}
	typ = types.Unalias(typ)
	if named, ok := typ.(*types.Named); ok {
		return named.Obj() == want
	}
	if pointer, ok := typ.(*types.Pointer); ok {
		return rev14NamedTypeOrPointer(pointer.Elem(), want)
	}
	return false
}

func rev14TypeContains(typ types.Type, want *types.TypeName) bool {
	return rawTypeInType(typ, want, map[types.Type]bool{})
}

func rev14AdmittedValueContains(typ types.Type, want, excluded *types.TypeName) bool {
	if typ == nil {
		return false
	}
	if excluded != nil && types.Identical(typ, excluded.Type().Underlying()) {
		return false
	}
	if named, ok := typ.(*types.Named); ok {
		if named.Obj() == excluded {
			return false
		}
		if named.Obj() == want {
			return true
		}
		return rev14AdmittedValueContains(named.Underlying(), want, excluded)
	}
	switch current := typ.(type) {
	case *types.Pointer:
		return rev14AdmittedValueContains(current.Elem(), want, excluded)
	case *types.Map:
		return rev14AdmittedValueContains(current.Key(), want, excluded) || rev14AdmittedValueContains(current.Elem(), want, excluded)
	case *types.Slice:
		return rev14AdmittedValueContains(current.Elem(), want, excluded)
	case *types.Array:
		return rev14AdmittedValueContains(current.Elem(), want, excluded)
	case *types.Chan:
		return rev14AdmittedValueContains(current.Elem(), want, excluded)
	case *types.Signature:
		return rev14AdmittedValueContains(current.Params(), want, excluded) || rev14AdmittedValueContains(current.Results(), want, excluded)
	case *types.Tuple:
		for index := 0; index < current.Len(); index++ {
			if rev14AdmittedValueContains(current.At(index).Type(), want, excluded) {
				return true
			}
		}
	case *types.Struct:
		for index := 0; index < current.NumFields(); index++ {
			if rev14AdmittedValueContains(current.Field(index).Type(), want, excluded) {
				return true
			}
		}
	}
	return false
}

func rev14TypeSyntax(expression ast.Expr, info *types.Info) bool {
	if ident, ok := expression.(*ast.Ident); ok {
		if _, isType := info.Uses[ident].(*types.TypeName); isType {
			return true
		}
		if _, isType := info.Defs[ident].(*types.TypeName); isType {
			return true
		}
		if variable, isField := info.Defs[ident].(*types.Var); isField && variable.IsField() {
			return true
		}
	}
	switch expression.(type) {
	case *ast.ArrayType, *ast.ChanType, *ast.Ellipsis, *ast.FuncType,
		*ast.InterfaceType, *ast.MapType, *ast.StructType, *ast.StarExpr:
		return true
	default:
		return false
	}
}

func rev14Callee(expression ast.Expr, info *types.Info) types.Object {
	switch called := expression.(type) {
	case *ast.Ident:
		if object := info.Uses[called]; object != nil {
			return object
		}
		if object := info.Defs[called]; object != nil {
			return object
		}
	case *ast.SelectorExpr:
		if selection := info.Selections[called]; selection != nil {
			return selection.Obj()
		}
		if object := info.Uses[called.Sel]; object != nil {
			return object
		}
	}
	return nil
}

func rev14AdmittedResultCallAllowed(callee types.Object, context *rev14FunctionContext, rawOwners map[*types.Func]bool, producer, referencedBinding *types.Func, profileEntries map[*types.Func]bool, callbackParameters map[*types.Func]map[*types.Var]bool, buildAdmission *types.Func, functions []rev14FunctionContext) bool {
	if callee == producer {
		return rev14RawOwner(context, rawOwners, functions, buildAdmission)
	}
	if callee == referencedBinding {
		return rev14RawOwner(context, rawOwners, functions, buildAdmission) || rev14ContextInFunction(context, buildAdmission, functions)
	}
	if context == nil || !profileEntries[context.object] {
		return false
	}
	callback, ok := callee.(*types.Var)
	return ok && callbackParameters[context.object][callback]
}

func rev14AdmittedArgumentAllowed(callee types.Object, context *rev14FunctionContext, rawOwners, profileEntries map[*types.Func]bool, authorityMethod, buildAdmission *types.Func, functions []rev14FunctionContext) bool {
	callerAllowed := context != nil && (profileEntries[context.object] || rev14RawOwner(context, rawOwners, functions, buildAdmission))
	function, ok := callee.(*types.Func)
	return callerAllowed && ok && (profileEntries[function] || function == authorityMethod)
}

func rev14DirectAdmittedCall(expression ast.Expr, info *types.Info, context *rev14FunctionContext, rawOwners map[*types.Func]bool, producer, referencedBinding *types.Func, profileEntries map[*types.Func]bool, callbackParameters map[*types.Func]map[*types.Var]bool, buildAdmission *types.Func, functions []rev14FunctionContext) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	callee := rev14Callee(call.Fun, info)
	return rev14AdmittedResultCallAllowed(callee, context, rawOwners, producer, referencedBinding, profileEntries, callbackParameters, buildAdmission, functions)
}

func rev14CheckpointAdmissionParameters(profileEntries map[*types.Func]bool, want *types.TypeName) map[*types.Func]map[*types.Var]bool {
	parameters := make(map[*types.Func]map[*types.Var]bool, len(profileEntries))
	if want == nil {
		return parameters
	}
	for function := range profileEntries {
		parameters[function] = make(map[*types.Var]bool)
		signature, _ := function.Type().(*types.Signature)
		if signature == nil {
			continue
		}
		for index := 0; index < signature.Params().Len(); index++ {
			parameter := signature.Params().At(index)
			if types.Identical(parameter.Type(), want.Type()) {
				parameters[function][parameter] = true
			}
		}
	}
	return parameters
}

func rev14SealedTypeExpr(expression ast.Expr, info *types.Info, seal, token *types.TypeName) bool {
	switch current := expression.(type) {
	case *ast.Ident:
		object := info.Uses[current]
		if object == nil {
			object = info.Defs[current]
		}
		return rev14SealedTypeObject(object, seal, token)
	case *ast.ParenExpr:
		return rev14SealedTypeExpr(current.X, info, seal, token)
	case *ast.StarExpr:
		return rev14SealedTypeExpr(current.X, info, seal, token)
	case *ast.ArrayType:
		return rev14SealedTypeExpr(current.Elt, info, seal, token)
	case *ast.ChanType:
		return rev14SealedTypeExpr(current.Value, info, seal, token)
	case *ast.Ellipsis:
		return rev14SealedTypeExpr(current.Elt, info, seal, token)
	case *ast.InterfaceType:
		return false
	case *ast.MapType:
		return rev14SealedTypeExpr(current.Key, info, seal, token) || rev14SealedTypeExpr(current.Value, info, seal, token)
	case *ast.IndexExpr:
		return rev14SealedTypeExpr(current.X, info, seal, token) || rev14SealedTypeExpr(current.Index, info, seal, token)
	case *ast.IndexListExpr:
		if rev14SealedTypeExpr(current.X, info, seal, token) {
			return true
		}
		for _, index := range current.Indices {
			if rev14SealedTypeExpr(index, info, seal, token) {
				return true
			}
		}
	}
	return false
}

func rev14SealedTypeObject(object types.Object, seal, token *types.TypeName) bool {
	if object == nil {
		return false
	}
	typeName, ok := object.(*types.TypeName)
	if !ok {
		return object == seal || object == token
	}
	return rev14SealedTypeIdentityContains(types.Unalias(typeName.Type()), seal, token, make(map[types.Type]bool))
}

// rev14SealedTypeIdentityContains follows the type identity denoted by a type
// expression. It deliberately does not descend through the underlying type
// of an unrelated named type: an admittedCheckpoint declaration contains a
// seal internally, but its public capability type is not itself a seal type.
// Alias normalization happens before each structural descent, so aliases of
// pointers, containers, signatures, and instantiated generic types retain the
// canonical seal/token identity.
func rev14SealedTypeIdentityContains(typ types.Type, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	typ = types.Unalias(typ)
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	if named, ok := typ.(*types.Named); ok {
		if named.Obj() == seal || named.Obj() == token {
			return true
		}
		return named.TypeArgs() != nil && rev14SealedTypeListContainsIdentity(named.TypeArgs(), seal, token, seen)
	}
	switch current := typ.(type) {
	case *types.Pointer:
		return rev14SealedTypeIdentityContains(current.Elem(), seal, token, seen)
	case *types.Array:
		return rev14SealedTypeIdentityContains(current.Elem(), seal, token, seen)
	case *types.Slice:
		return rev14SealedTypeIdentityContains(current.Elem(), seal, token, seen)
	case *types.Map:
		return rev14SealedTypeIdentityContains(current.Key(), seal, token, seen) || rev14SealedTypeIdentityContains(current.Elem(), seal, token, seen)
	case *types.Chan:
		return rev14SealedTypeIdentityContains(current.Elem(), seal, token, seen)
	case *types.Signature:
		if receiver := current.Recv(); receiver != nil && rev14SealedTypeIdentityContains(receiver.Type(), seal, token, seen) {
			return true
		}
		if typeParams := current.RecvTypeParams(); typeParams != nil && rev14SealedTypeParamListContainsIdentity(typeParams, seal, token, seen) {
			return true
		}
		if typeParams := current.TypeParams(); typeParams != nil && rev14SealedTypeParamListContainsIdentity(typeParams, seal, token, seen) {
			return true
		}
		return rev14SealedTypeTupleContainsIdentity(current.Params(), seal, token, seen) || rev14SealedTypeTupleContainsIdentity(current.Results(), seal, token, seen)
	case *types.Tuple:
		return rev14SealedTypeTupleContainsIdentity(current, seal, token, seen)
	case *types.Struct:
		for index := 0; index < current.NumFields(); index++ {
			if rev14SealedTypeIdentityContains(current.Field(index).Type(), seal, token, seen) {
				return true
			}
		}
	case *types.Interface:
		for index := 0; index < current.NumEmbeddeds(); index++ {
			if rev14SealedTypeIdentityContains(current.EmbeddedType(index), seal, token, seen) {
				return true
			}
		}
		for index := 0; index < current.NumMethods(); index++ {
			if rev14SealedTypeIdentityContains(current.Method(index).Type(), seal, token, seen) {
				return true
			}
		}
	case *types.TypeParam:
		return rev14SealedTypeIdentityContains(current.Constraint(), seal, token, seen)
	case *types.Union:
		for index := 0; index < current.Len(); index++ {
			if rev14SealedTypeIdentityContains(current.Term(index).Type(), seal, token, seen) {
				return true
			}
		}
	}
	return false
}

func rev14SealedTypeListContainsIdentity(list *types.TypeList, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < list.Len(); index++ {
		if rev14SealedTypeIdentityContains(list.At(index), seal, token, seen) {
			return true
		}
	}
	return false
}

func rev14SealedTypeParamListContainsIdentity(list *types.TypeParamList, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < list.Len(); index++ {
		if rev14SealedTypeIdentityContains(list.At(index), seal, token, seen) {
			return true
		}
	}
	return false
}

func rev14SealedTypeTupleContainsIdentity(tuple *types.Tuple, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < tuple.Len(); index++ {
		if rev14SealedTypeIdentityContains(tuple.At(index).Type(), seal, token, seen) {
			return true
		}
	}
	return false
}

func rev14SealedFuncType(function *ast.FuncType, info *types.Info, seal, token *types.TypeName) bool {
	if function.Params != nil {
		for _, field := range function.Params.List {
			if rev14SealedTypeExpr(field.Type, info, seal, token) {
				return true
			}
		}
	}
	if function.Results != nil {
		for _, field := range function.Results.List {
			if rev14SealedTypeExpr(field.Type, info, seal, token) {
				return true
			}
		}
	}
	return false
}

func rev14SealedTypeInCallTarget(call *ast.CallExpr, info *types.Info, seal, token *types.TypeName) bool {
	if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "new" && len(call.Args) == 1 {
		if _, builtin := info.Uses[identifier].(*types.Builtin); builtin {
			return rev14SealedTypeExpr(call.Args[0], info, seal, token)
		}
	}
	switch target := call.Fun.(type) {
	case *ast.IndexExpr:
		return rev14SealedTypeExpr(target.Index, info, seal, token)
	case *ast.IndexListExpr:
		for _, index := range target.Indices {
			if rev14SealedTypeExpr(index, info, seal, token) {
				return true
			}
		}
	}
	return false
}

func rev14SealedValueContains(typ types.Type, seal, token *types.TypeName) bool {
	return rev14SealedValueContainsSeen(typ, seal, token, make(map[types.Type]bool))
}

func rev14UnaliasSealedValueType(typ types.Type) types.Type {
	return types.Unalias(typ)
}

func rev14SealedValueContainsSeen(typ types.Type, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	typ = rev14UnaliasSealedValueType(typ)
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	if named, ok := typ.(*types.Named); ok {
		if named.Obj() == seal || named.Obj() == token {
			return true
		}
		if args := named.TypeArgs(); args != nil && rev14SealedTypeListContains(args, seal, token, seen) {
			return true
		}
		return rev14SealedValueContainsSeen(named.Underlying(), seal, token, seen)
	}
	switch current := typ.(type) {
	case *types.Pointer:
		return rev14SealedValueContainsSeen(current.Elem(), seal, token, seen)
	case *types.Array:
		return rev14SealedValueContainsSeen(current.Elem(), seal, token, seen)
	case *types.Slice:
		return rev14SealedValueContainsSeen(current.Elem(), seal, token, seen)
	case *types.Map:
		return rev14SealedValueContainsSeen(current.Key(), seal, token, seen) || rev14SealedValueContainsSeen(current.Elem(), seal, token, seen)
	case *types.Chan:
		return rev14SealedValueContainsSeen(current.Elem(), seal, token, seen)
	case *types.Signature:
		if receiver := current.Recv(); receiver != nil && rev14SealedValueContainsSeen(receiver.Type(), seal, token, seen) {
			return true
		}
		if typeParams := current.RecvTypeParams(); typeParams != nil && rev14SealedTypeParamListContains(typeParams, seal, token, seen) {
			return true
		}
		if typeParams := current.TypeParams(); typeParams != nil && rev14SealedTypeParamListContains(typeParams, seal, token, seen) {
			return true
		}
		return rev14SealedTypeTupleContains(current.Params(), seal, token, seen) || rev14SealedTypeTupleContains(current.Results(), seal, token, seen)
	case *types.Tuple:
		return rev14SealedTypeTupleContains(current, seal, token, seen)
	case *types.Struct:
		for index := 0; index < current.NumFields(); index++ {
			if rev14SealedValueContainsSeen(current.Field(index).Type(), seal, token, seen) {
				return true
			}
		}
	case *types.Interface:
		for index := 0; index < current.NumEmbeddeds(); index++ {
			if rev14SealedValueContainsSeen(current.EmbeddedType(index), seal, token, seen) {
				return true
			}
		}
		for index := 0; index < current.NumMethods(); index++ {
			if rev14SealedValueContainsSeen(current.Method(index).Type(), seal, token, seen) {
				return true
			}
		}
	case *types.TypeParam:
		return rev14SealedValueContainsSeen(current.Constraint(), seal, token, seen)
	case *types.Union:
		for index := 0; index < current.Len(); index++ {
			if rev14SealedValueContainsSeen(current.Term(index).Type(), seal, token, seen) {
				return true
			}
		}
	}
	return false
}

func rev14SealedTypeListContains(list *types.TypeList, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < list.Len(); index++ {
		if rev14SealedValueContainsSeen(list.At(index), seal, token, seen) {
			return true
		}
	}
	return false
}

func rev14SealedTypeParamListContains(list *types.TypeParamList, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < list.Len(); index++ {
		if rev14SealedValueContainsSeen(list.At(index), seal, token, seen) {
			return true
		}
	}
	return false
}

func rev14SealedTypeTupleContains(tuple *types.Tuple, seal, token *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < tuple.Len(); index++ {
		if rev14SealedValueContainsSeen(tuple.At(index).Type(), seal, token, seen) {
			return true
		}
	}
	return false
}

func rev14SealedReceiver(typ types.Type, seal, token *types.TypeName) bool {
	return rev14NamedTypeOrPointer(typ, seal) || rev14NamedTypeOrPointer(typ, token)
}

func rev14SealSignatureAllowed(context *rev14FunctionContext, producer, authorityMethod *types.Func) bool {
	return rev14ExactFunction(context, producer) || rev14ExactMethod(context, authorityMethod)
}

func rev14SealValueAllowed(context *rev14FunctionContext, producer, authorityMethod *types.Func, profileEntries, rawOwners map[*types.Func]bool, functions []rev14FunctionContext, buildAdmission *types.Func) bool {
	if rev14SealSignatureAllowed(context, producer, authorityMethod) {
		return true
	}
	if context == nil {
		return false
	}
	if profileEntries[context.object] || rev14RawOwner(context, rawOwners, functions, buildAdmission) {
		return true
	}
	return context.isClosure && rev14ContextInFunction(context, buildAdmission, functions)
}

// rev12LoadImportArchives asks the module-aware Go tool for export data, then
// gives go/types an explicit lookup table. importer.Default alone searches
// GOPATH and cannot resolve this module's internal packages from a temporary
// hostile-plant directory.
func rev12LoadImportArchives() (map[string]string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("cannot locate test file for module root")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	command := exec.Command("go", "list", "-deps", "-export", "-json", "./internal/sessquery")
	command.Dir = moduleRoot
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("load Go export archives: %w", err)
	}
	archives := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var item struct {
			ImportPath string
			Export     string
		}
		if err := decoder.Decode(&item); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode Go export archive list: %w", err)
		}
		if item.ImportPath != "" && item.Export != "" {
			archives[item.ImportPath] = item.Export
		}
	}
	return archives, nil
}

type rev12FunctionSpan struct {
	name       string
	start, end token.Pos
}

func rev12CollectFunctions(files []*ast.File, fset *token.FileSet) []rev12FunctionSpan {
	_ = fset
	var functions []rev12FunctionSpan
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			function, ok := node.(*ast.FuncDecl)
			if ok {
				functions = append(functions, rev12FunctionSpan{name: function.Name.Name, start: function.Pos(), end: function.End()})
			}
			return true
		})
	}
	return functions
}

func rev12EnclosingFunction(position token.Pos, functions []rev12FunctionSpan) string {
	var match rev12FunctionSpan
	for _, function := range functions {
		if position < function.start || position > function.end {
			continue
		}
		if match.name == "" || function.end-function.start < match.end-match.start {
			match = function
		}
	}
	return match.name
}

func rev12IsNamedType(typ types.Type, want *types.TypeName) bool {
	named, ok := typ.(*types.Named)
	return ok && named.Obj() == want
}

func rawTypeInType(typ types.Type, raw *types.TypeName, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	switch current := typ.(type) {
	case *types.Named:
		if current.Obj() == raw {
			return true
		}
		return rawTypeInType(current.Underlying(), raw, seen)
	case *types.Map:
		return rawTypeInType(current.Key(), raw, seen) || rawTypeInType(current.Elem(), raw, seen)
	case *types.Slice:
		return rawTypeInType(current.Elem(), raw, seen)
	case *types.Array:
		return rawTypeInType(current.Elem(), raw, seen)
	case *types.Pointer:
		return rawTypeInType(current.Elem(), raw, seen)
	case *types.Chan:
		return rawTypeInType(current.Elem(), raw, seen)
	case *types.Signature:
		if current.Recv() != nil && rawTypeInType(current.Recv().Type(), raw, seen) {
			return true
		}
		return rawTypeInTuple(current.Params(), raw, seen) || rawTypeInTuple(current.Results(), raw, seen)
	case *types.Tuple:
		return rawTypeInTuple(current, raw, seen)
	case *types.Struct:
		for index := 0; index < current.NumFields(); index++ {
			if rawTypeInType(current.Field(index).Type(), raw, seen) {
				return true
			}
		}
	}
	return false
}

func rawTypeInTuple(tuple *types.Tuple, raw *types.TypeName, seen map[types.Type]bool) bool {
	for index := 0; index < tuple.Len(); index++ {
		if rawTypeInType(tuple.At(index).Type(), raw, seen) {
			return true
		}
	}
	return false
}

func rev12CopyProductionSources(t *testing.T, sourceDir, destinationDir string) {
	t.Helper()
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destinationDir, name), source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
