package gitsnap

import (
	"context"
	"encoding/json"
	"testing"
)

// Capture mutates no durable state, so crash evidence means the call
// either returns a complete snapshot or a refusal, and idempotency means
// a quiesced repository captures byte-identically. The fault-injection
// tests below prove the consistency re-read provably fires inside the
// window it claims to cover: the scripted fault is armed exactly once,
// the test asserts the armed set is empty afterwards, and a retry with a
// stable conversation heals.

// TestIdempotentRepeatCapture proves two sequential captures of one
// quiesced live repository encode byte-for-byte identically.
func TestIdempotentRepeatCapture(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)
	first := captureLive(t, dir)
	second := captureLive(t, dir)
	firstBytes, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Errorf("repeat captures differ:\n%s\n%s", firstBytes, secondBytes)
	}
}

// TestConsistencyRefusesIndexMutationArmedOnce proves GateConsistency
// fires on an index that moves between the capture read and the
// re-read, and proves the fault fired inside the window: the ls-files
// queue is armed with exactly [valid, changed], both are consumed, and
// the remaining armed set is empty.
func TestConsistencyRefusesIndexMutationArmedOnce(t *testing.T) {
	t.Parallel()
	changed := "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md\x00" +
		"  ctime: 9:9\n  mtime: 3:4\n  dev: 5\tino: 6\n  uid: 7\tgid: 8\n  size: 6\tflags: 0\n" +
		"100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md\x00" + stageProbeDebug
	fake := scriptBaseline(newScriptedRunner())
	lsArgs := []string{"ls-files", "--stage", "--debug", "-z"}
	valid := fake.scripts[scriptKey(lsArgs)][0].Stdout
	fake.Rescript(lsArgs, okResult(string(valid)), okResult(changed))
	if fake.remaining(lsArgs...) != 2 {
		t.Fatalf("armed set = %d, want 2 queued reads", fake.remaining(lsArgs...))
	}
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateConsistency)
	if fake.remaining(lsArgs...) != 0 {
		t.Errorf("armed set = %d after capture, want empty: the fault must fire inside the window", fake.remaining(lsArgs...))
	}
	if count := fake.callCount(lsArgs...); count != 2 {
		t.Errorf("ls-files ran %d times, want capture read plus re-read", count)
	}
}

// TestConsistencyRefusesHeadMutation proves a HEAD that advances
// mid-capture is refused even when the index is stable.
func TestConsistencyRefusesHeadMutation(t *testing.T) {
	t.Parallel()
	headArgs := []string{"rev-parse", "--verify", "--quiet", "HEAD"}
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript(headArgs,
		okResult("602548b4fd46332c934667db9992b8bb00318c88\n"),
		okResult("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateConsistency)
	if fake.remaining(headArgs...) != 0 {
		t.Errorf("armed set = %d after capture, want empty", fake.remaining(headArgs...))
	}
}

// TestConsistencyRetryHeals proves the refusal is transient: the same
// conversation, replayed stable, captures.
func TestConsistencyRetryHeals(t *testing.T) {
	t.Parallel()
	lsArgs := []string{"ls-files", "--stage", "--debug", "-z"}
	fake := scriptBaseline(newScriptedRunner())
	valid := fake.scripts[scriptKey(lsArgs)][0].Stdout
	changed := "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md\x00" +
		"  ctime: 9:9\n  mtime: 3:4\n  dev: 5\tino: 6\n  uid: 7\tgid: 8\n  size: 6\tflags: 0\n" +
		"100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md\x00" + stageProbeDebug
	fake.Rescript(lsArgs, okResult(string(valid)), okResult(changed))
	if _, err := Capture(context.Background(), fake, "/tmp/repo"); err == nil {
		t.Fatal("want GateConsistency on the mutated run")
	}
	healed := scriptBaseline(newScriptedRunner())
	snapshot, err := Capture(context.Background(), healed, "/tmp/repo")
	if err != nil {
		t.Fatalf("retry Capture() error = %v", err)
	}
	if len(snapshot.Index.Entries) != 2 {
		t.Errorf("healed entries = %+v", snapshot.Index.Entries)
	}
}
