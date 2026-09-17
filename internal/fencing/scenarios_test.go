package fencing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// buildFenceRecord derives a valid Session Record from the SPEC
// example with the caller's session identity and name.
func buildFenceRecord(t *testing.T, sessionID, name string) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal([]byte(specRecordExample), &object); err != nil {
		t.Fatalf("unmarshal SPEC record example: %v", err)
	}
	object["session_id"], object["subject_id"], object["name"] = sessionID, sessionID, name
	return identifyFenceObject(t, object, "record_id")
}

func createdFencePayload(recordID string) map[string]any {
	return map[string]any{"session_record_id": recordID, "bootstrap_operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab", "first_checkpoint_operation_id": "0198f4c8-7d40-7e55-8e6f-1234567890ac"}
}

func idleFencePayload() map[string]any {
	return map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}
}

func resumedFencePayload() map[string]any {
	return map[string]any{"checkpoint_id": fenceZeroDigest, "execution_profile": "yolo", "profile_source_event_id": nil, "terminal_backend": "tmux", "native_session_id": "pane-1"}
}

// appendFenceEvent appends one valid event at the given chain position
// and returns its reference and bytes.
func appendFenceEvent(t *testing.T, repository *sessrepo.Repository, sessionID string, predecessors []string, epoch uint64, leaseID string, sequence uint64, hostID, eventType string, payload map[string]any) (sessrepo.EventRef, []byte) {
	t.Helper()
	raw := buildFenceEvent(t, sessionID, predecessors, epoch, leaseID, sequence, hostID, eventType, payload)
	reference, err := repository.AppendEvent(sessionID, raw)
	if err != nil {
		t.Fatalf("AppendEvent(seq %d) error = %v", sequence, err)
	}
	return reference, raw
}

// observeFence builds the Observe input for a verified local grant.
func observeFence(localHostID string) ObserveInput {
	return ObserveInput{LocalHostID: localHostID, Verified: true, HasGrant: true, Grant: sessrepo.FencingGrant{SessionID: fenceSessionA, ValidatedAt: fenceValidatedAt}, Policy: fencePolicy, Now: fenceValidatedAt}
}

// plantForceTakeover builds the two-replica force-takeover fixture:
// repoOld holds session A with the epoch-1 lease and its two-event
// chain, repoNew holds the byte-identical base plus the epoch-2 force
// lease and the resumed event under it. It returns both repositories
// with the new replica's data root for blob assertions.
func plantForceTakeover(t *testing.T) (repoOld, repoNew *sessrepo.Repository, rootNew string) {
	t.Helper()
	var rootOld string
	repoOld, rootOld = openFenceRepository(t)
	_ = rootOld
	repoNew, rootNew = openFenceRepository(t)
	record := createFenceSession(t, repoOld)
	if _, err := repoNew.CreateSession([]byte(specRecordExample)); err != nil {
		t.Fatalf("CreateSession(new) error = %v", err)
	}
	base := sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt}
	oldHead := mustCreateFenceLease(t, repoOld, fenceSessionA, base)
	newHead := mustCreateFenceLease(t, repoNew, fenceSessionA, base)
	if oldHead.RecordID != newHead.RecordID {
		t.Fatalf("epoch-1 bases differ: %s vs %s", oldHead.RecordID, newHead.RecordID)
	}
	created, _ := appendFenceEvent(t, repoOld, fenceSessionA, []string{record.RecordID}, 1, fenceLeaseOld, 1, fenceHostA, "session.created", createdFencePayload(record.RecordID))
	idle, _ := appendFenceEvent(t, repoOld, fenceSessionA, []string{created.EventID}, 1, fenceLeaseOld, 2, fenceHostA, "session.idle", idleFencePayload())
	if _, err := repoNew.AppendEvent(fenceSessionA, buildFenceEvent(t, fenceSessionA, []string{record.RecordID}, 1, fenceLeaseOld, 1, fenceHostA, "session.created", createdFencePayload(record.RecordID))); err != nil {
		t.Fatalf("AppendEvent(new created) error = %v", err)
	}
	if _, err := repoNew.AppendEvent(fenceSessionA, buildFenceEvent(t, fenceSessionA, []string{created.EventID}, 1, fenceLeaseOld, 2, fenceHostA, "session.idle", idleFencePayload())); err != nil {
		t.Fatalf("AppendEvent(new idle) error = %v", err)
	}
	mustSwapFenceLease(t, repoNew, fenceSessionA, sessrepo.LeaseExpectation{RecordID: newHead.RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: fenceLeaseA, HolderHostID: fenceHostB, IssuedByHostID: fenceHostB, CreatedAt: fenceLeaseAt},
		Reason:           "force_takeover",
		CheckpointID:     fenceZeroDigest,
	})
	appendFenceEvent(t, repoNew, fenceSessionA, []string{idle.EventID}, 2, fenceLeaseA, 1, fenceHostB, "session.resumed", resumedFencePayload())
	return repoOld, repoNew, rootNew
}

// TestOldOwnerReconnectRejectedAfterForceTakeover drives the
// old-owner-reconnect row through production entries: after a force
// takeover, the old token is rejected from input, mutation, and
// restore, the old wrapper parks remote, the losing events stay
// preserved without application, and the sessstate union reports the
// losing branch preserved while authoritative state stands.
func TestOldOwnerReconnectRejectedAfterForceTakeover(t *testing.T) {
	_, repoNew, rootNew := plantForceTakeover(t)
	old := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}

	t.Run("new_owner_rejects_old_token_as_stale", func(t *testing.T) {
		observation, err := Observe(repoNew, fenceSessionA, observeFence(fenceHostB))
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		_, err = AuthorizeInput(old, observation)
		mustRefuse(t, err, ErrStaleOwner, "reconnect/input")
		_, err = AuthorizeMutation(old, observation)
		mustRefuse(t, err, ErrStaleOwner, "reconnect/mutation")
		_, err = AuthorizeCheckpoint(old, observation)
		mustRefuse(t, err, ErrStaleOwner, "reconnect/checkpoint")
	})

	t.Run("old_wrapper_parks_remote", func(t *testing.T) {
		observation, err := Observe(repoNew, fenceSessionA, observeFence(fenceHostA))
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		_, err = AuthorizeRestore(old, observation)
		mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, "reconnect/restore")
		_, err = AuthorizeActivation(old, observation)
		mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, "reconnect/activation")
	})

	t.Run("losing_events_preserved_without_application", func(t *testing.T) {
		before, err := repoNew.ListEvents(fenceSessionA)
		if err != nil {
			t.Fatalf("ListEvents error = %v", err)
		}
		divergent := buildFenceEvent(t, fenceSessionA, []string{before[len(before)-1].EventID}, 1, fenceLeaseOld, 3, fenceHostA, "session.idle", idleFencePayload())
		_, err = repoNew.AppendEvent(fenceSessionA, divergent)
		if !errors.Is(err, sessrepo.ErrStaleLease) {
			t.Fatalf("AppendEvent(divergent) error = %v, want stale lease", err)
		}
		after, err := repoNew.ListEvents(fenceSessionA)
		if err != nil {
			t.Fatalf("ListEvents error = %v", err)
		}
		if len(after) != len(before) || after[len(after)-1].EventID != before[len(before)-1].EventID {
			t.Fatalf("chain moved under a losing append: %d events, tail %s", len(after), after[len(after)-1].EventID)
		}
		var decoded map[string]any
		if err := json.Unmarshal(divergent, &decoded); err != nil {
			t.Fatalf("unmarshal divergent event: %v", err)
		}
		eventID, _ := decoded["event_id"].(string)
		blobPath := filepath.Join(rootNew, "sessions", fenceSessionA, "events", strings.TrimPrefix(eventID, "sha256:")+".json")
		stored, err := os.ReadFile(blobPath)
		if err != nil {
			t.Fatalf("read preserved divergent blob: %v", err)
		}
		if !bytes.Equal(stored, divergent) {
			t.Fatal("preserved divergent blob differs from the refused bytes")
		}
	})

	t.Run("union_preserves_losing_branch", func(t *testing.T) {
		repoOld, _ := openFenceRepository(t)
		record := createFenceSession(t, repoOld)
		base := sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt}
		mustCreateFenceLease(t, repoOld, fenceSessionA, base)
		created, _ := appendFenceEvent(t, repoOld, fenceSessionA, []string{record.RecordID}, 1, fenceLeaseOld, 1, fenceHostA, "session.created", createdFencePayload(record.RecordID))
		appendFenceEvent(t, repoOld, fenceSessionA, []string{created.EventID}, 1, fenceLeaseOld, 2, fenceHostA, "session.idle", idleFencePayload())

		plain, err := (&sessstate.Projector{Repo: repoOld}).Project(fenceSessionA)
		if err != nil {
			t.Fatalf("Project(chain) error = %v", err)
		}
		if plain.State != sessstate.StateIdle {
			t.Fatalf("chain projection state = %s, want idle", plain.State)
		}
		union := []sessstate.LeaseHead{{Epoch: 2, LeaseID: fenceLeaseA}, {Epoch: 1, LeaseID: fenceLeaseOld}}
		withUnion, err := (&sessstate.Projector{Repo: repoOld, Union: union}).Project(fenceSessionA)
		if err != nil {
			t.Fatalf("Project(union) error = %v", err)
		}
		if withUnion.Winner.Epoch != 2 || withUnion.Winner.LeaseID != fenceLeaseA {
			t.Fatalf("union winner = %+v, want the force lease", withUnion.Winner)
		}
		if withUnion.State != sessstate.StateIdle {
			t.Fatalf("union projection state = %s, want idle: authoritative state must stand", withUnion.State)
		}
		staleView, err := (&sessstate.Projector{Repo: repoOld, Union: union, Local: sessstate.LeaseHead{Epoch: 1, LeaseID: fenceLeaseOld}, LocalHostID: fenceHostA}).Project(fenceSessionA)
		if err != nil {
			t.Fatalf("Project(union+local) error = %v", err)
		}
		if staleView.State != sessstate.StateStale {
			t.Fatalf("old-owner projection state = %s, want stale", staleView.State)
		}
		assertConflict(t, staleView, sessstate.ConflictLosingBranchPreserved)
		assertConflict(t, staleView, sessstate.ConflictUnionSupersedesChain)
	})
}

func assertConflict(t *testing.T, projection sessstate.Projection, kind string) {
	t.Helper()
	for _, conflict := range projection.Conflicts {
		if conflict.Kind == kind {
			return
		}
	}
	t.Fatalf("projection conflicts %v lack %s", projection.Conflicts, kind)
}

// TestConcurrentForceTakeoversDeterministicWinner drives the
// concurrent-takeover row: two force leases at the same epoch from
// one base resolve to the bytewise-greater lease ID in either union
// order under both tuple rules, the winner authorizes, and the loser
// stops accepting input.
func TestConcurrentForceTakeoversDeterministicWinner(t *testing.T) {
	build := func(t *testing.T, leaseID, holder string) *sessrepo.Repository {
		t.Helper()
		repository, _ := openFenceRepository(t)
		createFenceSession(t, repository)
		base := mustCreateFenceLease(t, repository, fenceSessionA, sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt})
		mustSwapFenceLease(t, repository, fenceSessionA, sessrepo.LeaseExpectation{RecordID: base.RecordID}, sessrepo.SuccessorLeaseInput{
			CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: leaseID, HolderHostID: holder, IssuedByHostID: holder, CreatedAt: fenceLeaseAt},
			Reason:           "force_takeover",
			CheckpointID:     fenceZeroDigest,
		})
		return repository
	}
	repoX := build(t, fenceLeaseA, fenceHostB)
	repoY := build(t, fenceLeaseC2, fenceHostC)
	headX, err := repoX.WinningLease(fenceSessionA)
	if err != nil {
		t.Fatalf("WinningLease(X) error = %v", err)
	}
	headY, err := repoY.WinningLease(fenceSessionA)
	if err != nil {
		t.Fatalf("WinningLease(Y) error = %v", err)
	}
	for _, order := range [][2]sessrepo.LeaseSummary{{headX, headY}, {headY, headX}} {
		winner := order[0]
		if sessrepo.CompareLeaseTuple(order[1], winner) > 0 {
			winner = order[1]
		}
		if winner.LeaseID != fenceLeaseC2 {
			t.Fatalf("union winner = %s, want the greater lease %s", winner.LeaseID, fenceLeaseC2)
		}
		stateWinner := sessstate.LeaseHead{Epoch: order[0].Epoch, LeaseID: order[0].LeaseID}
		other := sessstate.LeaseHead{Epoch: order[1].Epoch, LeaseID: order[1].LeaseID}
		if sessstate.Compare(other, stateWinner) > 0 {
			stateWinner = other
		}
		if stateWinner.LeaseID != winner.LeaseID {
			t.Fatalf("sessstate winner %s disagrees with store winner %s", stateWinner.LeaseID, winner.LeaseID)
		}
	}
	// The loser stops accepting input once it learns the union
	// winner: its same-epoch token loses the tie on the winning host.
	loser := PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseA}
	union := ObserveInput{LocalHostID: fenceHostC, Verified: true, HasGrant: true, Grant: sessrepo.FencingGrant{SessionID: fenceSessionA, ValidatedAt: fenceValidatedAt}, Policy: fencePolicy, Now: fenceValidatedAt}
	observation, err := Observe(repoY, fenceSessionA, union)
	if err != nil {
		t.Fatalf("Observe error = %v", err)
	}
	_, err = AuthorizeInput(loser, observation)
	mustRefuse(t, err, ErrLeaseConflict, "loser/input")
	_, err = AuthorizeMutation(loser, observation)
	mustRefuse(t, err, ErrLeaseConflict, "loser/mutation")
	champion := PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseC2}
	token, err := AuthorizeActivation(champion, observation)
	mustMintUnion(t, token, err, fenceLeaseC2)
}

// mustMintUnion is mustMint for a non-default winning lease: the
// minted token binds the exact expected triple.
func mustMintUnion(t *testing.T, token LeaseToken, err error, leaseID string) {
	t.Helper()
	if err != nil {
		t.Fatalf("union authorize error = %v, want a minted token", err)
	}
	bound, err := token.Bind(ProviderCapture)
	if err != nil {
		t.Fatalf("Bind(capture) error = %v", err)
	}
	if bound.SessionID != fenceSessionA || bound.Epoch != 2 || bound.LeaseID != leaseID {
		t.Fatalf("Bind(capture) = %+v, want the union winner", bound)
	}
}

// TestLifecycleCallersPassTheGate models graceful takeover, fork,
// stop, and resume as gate callers: each authorizes its entry point
// with the winning lease and the exact epoch, and the replica resume
// parks remote instead of starting a runtime.
func TestLifecycleCallersPassTheGate(t *testing.T) {
	t.Run("graceful_takeover_commit", func(t *testing.T) {
		repository, _ := openFenceRepository(t)
		createFenceSession(t, repository)
		first := mustCreateFenceLease(t, repository, fenceSessionA, sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt})
		mustSwapFenceLease(t, repository, fenceSessionA, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
			CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: fenceLeaseA, HolderHostID: fenceHostB, IssuedByHostID: fenceHostB, CreatedAt: fenceLeaseAt},
			Reason:           "graceful_takeover",
			CheckpointID:     fenceZeroDigest,
		})
		destination, err := Observe(repository, fenceSessionA, observeFence(fenceHostB))
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		token, err := AuthorizeMutation(PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseA}, destination)
		mustMint(t, token, err, "takeover/commit")
		source, err := Observe(repository, fenceSessionA, observeFence(fenceHostA))
		if err != nil {
			t.Fatalf("Observe(source) error = %v", err)
		}
		_, err = AuthorizeMutation(PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}, source)
		mustRefuse(t, err, ErrNotOwner, "takeover/source")
	})

	t.Run("fork_epoch_one_activation", func(t *testing.T) {
		repository, _ := openFenceRepository(t)
		if _, err := repository.CreateSession(buildFenceRecord(t, fenceSessionB, "payments-api-experiment")); err != nil {
			t.Fatalf("CreateSession(fork) error = %v", err)
		}
		mustCreateFenceLease(t, repository, fenceSessionB, sessrepo.CreateLeaseInput{LeaseID: fenceLeaseC, HolderHostID: fenceHostB, IssuedByHostID: fenceHostB, CreatedAt: fenceLeaseAt})
		input := observeFence(fenceHostB)
		input.Grant.SessionID = fenceSessionB
		observation, err := Observe(repository, fenceSessionB, input)
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		token, err := AuthorizeActivation(PresentedToken{SessionID: fenceSessionB, Epoch: 1, LeaseID: fenceLeaseC}, observation)
		if err != nil {
			t.Fatalf("fork activation error = %v", err)
		}
		bound, err := token.Bind(ProviderCapture)
		if err != nil {
			t.Fatalf("Bind(capture) error = %v", err)
		}
		if bound.SessionID != fenceSessionB || bound.Epoch != 1 || bound.LeaseID != fenceLeaseC {
			t.Fatalf("Bind(capture) = %+v, want the fork epoch-1 triple", bound)
		}
		_, err = AuthorizeActivation(PresentedToken{SessionID: fenceSessionB, Epoch: 2, LeaseID: fenceLeaseC}, observation)
		mustPark(t, err, ParkRestorePolicy, fenceLeaseC, ErrLeaseConflict, "fork/future_epoch")
	})

	t.Run("stop_checkpoint", func(t *testing.T) {
		observation := fenceObservation()
		token, err := AuthorizeCheckpoint(fencePresented(), observation)
		mustMint(t, token, err, "stop/checkpoint")
		_, err = AuthorizeCheckpoint(PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}, observation)
		mustRefuse(t, err, ErrStaleOwner, "stop/stale")
	})

	t.Run("owner_resume_and_replica_park", func(t *testing.T) {
		observation := fenceObservation()
		token, err := AuthorizeRestore(fencePresented(), observation)
		mustMint(t, token, err, "resume/owner")
		replica := fenceObservation()
		replica.LocalHostID = fenceHostC
		_, err = AuthorizeRestore(fencePresented(), replica)
		mustPark(t, err, ParkRemoteOwner, fenceLeaseA, ErrNotOwner, "resume/replica")
	})
}

// TestGatesPerformNoDurableWrites proves the crash/idempotency bound:
// gates are pure over durable inputs, so authorizing (passing and
// failing) through every entry changes no repository byte and decides
// identically on every run.
func TestGatesPerformNoDurableWrites(t *testing.T) {
	repository, root := openFenceRepository(t)
	createFenceSession(t, repository)
	first := mustCreateFenceLease(t, repository, fenceSessionA, sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt})
	mustSwapFenceLease(t, repository, fenceSessionA, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: fenceLeaseA, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt},
		Reason:           "graceful_takeover",
		CheckpointID:     fenceZeroDigest,
	})
	snapshot := snapshotTree(t, root)
	decide := func() []string {
		observation, err := Observe(repository, fenceSessionA, observeFence(fenceHostA))
		if err != nil {
			t.Fatalf("Observe error = %v", err)
		}
		var verdicts []string
		record := func(name string, err error) {
			if err == nil {
				verdicts = append(verdicts, name+":allow")
			} else {
				verdicts = append(verdicts, name+":"+classOf(err))
			}
		}
		entries := fenceEntries()
		var names []string
		for name := range entries {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			entry := entries[name]
			token, err := entry.authorize(fencePresented(), observation)
			record(name+"/exact", err)
			if err == nil {
				if _, err := token.Bind(ProviderQuiesce); err != nil {
					t.Fatalf("%s Bind error = %v", name, err)
				}
			}
			_, err = entry.authorize(PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}, observation)
			record(name+"/stale", err)
		}
		_, err = LeaseToken{}.Bind(ProviderCapture)
		record("bind/forged", err)
		record("terminate/stale", AuthorizeTerminateStale(PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}, observation.Winner, true, true, true))
		record("terminate/live", AuthorizeTerminateStale(fencePresented(), observation.Winner, true, true, true))
		return verdicts
	}
	firstRun := decide()
	if !equalSnapshot(t, snapshotTree(t, root), snapshot) {
		t.Fatal("repository bytes moved under gate evaluation")
	}
	secondRun := decide()
	if strings.Join(firstRun, "\n") != strings.Join(secondRun, "\n") {
		t.Fatalf("verdicts differ across runs:\n%s\nvs\n%s", strings.Join(firstRun, "\n"), strings.Join(secondRun, "\n"))
	}
	if !equalSnapshot(t, snapshotTree(t, root), snapshot) {
		t.Fatal("repository bytes moved under repeated gate evaluation")
	}
}

// classOf renders the Section 15 class of a gate refusal for the
// determinism comparison: park decisions render with their reason.
func classOf(err error) string {
	if reason, _, ok := ParkDetails(err); ok {
		return "park:" + string(reason)
	}
	for _, sentinel := range []error{ErrLeaseConflict, ErrNotOwner, ErrStaleOwner, ErrInvalidArguments, ErrLocalPreconditionFailed} {
		if errors.Is(err, sentinel) {
			return sentinel.Error()
		}
	}
	return "unknown:" + err.Error()
}

// snapshotTree hashes every file under the data root the repository
// was opened on.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = hex.EncodeToString(digest[:])
		return nil
	})
	if err != nil {
		t.Fatalf("walk sessions root: %v", err)
	}
	return out
}

func equalSnapshot(t *testing.T, got, want map[string]string) bool {
	t.Helper()
	if len(got) != len(want) {
		t.Logf("snapshot sizes differ: %d vs %d", len(got), len(want))
		return false
	}
	var keys []string
	for key := range want {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if got[key] != want[key] {
			t.Logf("snapshot differs at %s", key)
			return false
		}
	}
	return true
}
