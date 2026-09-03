package cliresult

import (
	"errors"
	"testing"
)

// failingWriter reports a write error, which is what a closed pipe looks like
// to a command that has already decided its outcome.
type failingWriter struct{ err error }

func (writer failingWriter) Write([]byte) (int, error) { return 0, writer.err }

// TestExitStatusSurvivesAFailedWrite pins the ordering the Section 14.2 exit
// rule needs: the process status is the outcome's status even when stdout could
// not be written. Reporting exit 0 because a pipe closed would turn a failure
// into a success at exactly the moment nobody can read the document.
func TestExitStatusSurvivesAFailedWrite(t *testing.T) {
	broken := errors.New("broken pipe")
	var sink capture
	emitter := mustEmitter(t, ModeJSON, false, Streams{
		Stdout: failingWriter{err: broken}, Stderr: &sink.stderr,
	})
	failure := mustFailure(t, "workspace_conflict")
	status, err := emitter.Emit(Outcome{Failure: failure})
	if !errors.Is(err, broken) {
		t.Fatalf("Emit = %v, want the write error", err)
	}
	if status != 5 {
		t.Fatalf("status = %d, want the workspace_conflict exit 5 despite the failed write", status)
	}

	emitter = mustEmitter(t, ModeText, false, Streams{
		Stdout: &sink.stdout, Stderr: failingWriter{err: broken},
	})
	if err := emitter.Log("diagnostic"); !errors.Is(err, broken) {
		t.Fatalf("Log = %v, want the write error", err)
	}
	if _, err := emitter.Progress("progress"); err != nil {
		t.Fatalf("Progress on a non-TTY reported a write error: %v", err)
	}
	emitter = mustEmitter(t, ModeText, false, Streams{
		Stdout: &sink.stdout, Stderr: failingWriter{err: broken},
	})
	emitter.streams.StderrIsTTY = true
	if written, err := emitter.Progress("progress"); written || !errors.Is(err, broken) {
		t.Fatalf("Progress = %t/%v, want false and the write error", written, err)
	}
	if err := emitter.Prompt("continue?"); !errors.Is(err, broken) {
		t.Fatalf("Prompt = %v, want the write error", err)
	}
}
