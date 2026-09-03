package cliresult

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

type capture struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (streams *capture) streams(tty bool) Streams {
	return Streams{Stdout: &streams.stdout, Stderr: &streams.stderr, StderrIsTTY: tty}
}

func mustFailure(t *testing.T, code axerror.Code) *axerror.Error {
	t.Helper()
	failure, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    code,
		Message: "the operation did not complete",
		IDs:     axerror.NoIDs(),
		Details: axerror.Details{},
	})
	if err != nil {
		t.Fatalf("axerror.New(%q): %v", code, err)
	}
	return failure
}

func mustEmitter(t *testing.T, mode Mode, nonInteractive bool, streams Streams) *Emitter {
	t.Helper()
	emitter, err := newEmitter(mode, nonInteractive, streams)
	if err != nil {
		t.Fatalf("newEmitter: %v", err)
	}
	return emitter
}

// TestJSONModeStdoutCarriesExactlyOneDocument narrows the Section 14.2 sentence
// "in JSON mode, stdout MUST contain exactly one JSON document; logs remain on
// stderr". Logs and progress written before and after the emission must not
// reach stdout, and a second emission is refused.
func TestJSONModeStdoutCarriesExactlyOneDocument(t *testing.T) {
	var streams capture
	emitter := mustEmitter(t, ModeJSON, false, streams.streams(true))
	if err := emitter.Log("resolving owner"); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if written, err := emitter.Progress("transferring 3 of 9"); err != nil || !written {
		t.Fatalf("Progress = %t/%v", written, err)
	}
	result := mustResult(t, validSpec(t, CommandList))
	status, err := emitter.Emit(Outcome{Result: result})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if status != SuccessExitStatus {
		t.Fatalf("status = %d, want 0", status)
	}
	if err := emitter.Log("done"); err != nil {
		t.Fatalf("Log after Emit: %v", err)
	}

	var decoder = json.NewDecoder(bytes.NewReader(streams.stdout.Bytes()))
	var first any
	if err := decoder.Decode(&first); err != nil {
		t.Fatalf("stdout does not carry a JSON document: %v\n%s", err, streams.stdout.String())
	}
	var second any
	if err := decoder.Decode(&second); err == nil {
		t.Fatalf("stdout carries more than one JSON document:\n%s", streams.stdout.String())
	}
	if !strings.Contains(streams.stderr.String(), "resolving owner") ||
		!strings.Contains(streams.stderr.String(), "transferring 3 of 9") ||
		!strings.Contains(streams.stderr.String(), "done") {
		t.Fatalf("logs did not stay on stderr:\n%s", streams.stderr.String())
	}
	if _, err := emitter.Emit(Outcome{Result: result}); !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("second Emit = %v, want ErrStreamDiscipline", err)
	}
}

// TestJSONModeRefusesHumanTextAlongsideTheDocument proves the one-document rule
// is enforced against the caller's own rendering, not only against a second
// Emit. A human line written next to the document would make stdout carry
// something that is not the document.
func TestJSONModeRefusesHumanTextAlongsideTheDocument(t *testing.T) {
	var streams capture
	emitter := mustEmitter(t, ModeJSON, false, streams.streams(false))
	result := mustResult(t, validSpec(t, CommandList))
	status, err := emitter.Emit(Outcome{Result: result, Rendered: "3 sessions"})
	if !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("Emit = %v, want ErrStreamDiscipline", err)
	}
	if status != SuccessExitStatus {
		t.Fatalf("status = %d, want the outcome's status even on a refused write", status)
	}
	if streams.stdout.Len() != 0 {
		t.Fatalf("stdout was written despite the refusal: %s", streams.stdout.String())
	}
}

// TestTextModeSeparatesDataFromDiagnostics narrows the Section 14.2 sentence
// "in text mode, data goes to stdout and prompts/diagnostics to stderr".
func TestTextModeSeparatesDataFromDiagnostics(t *testing.T) {
	var success capture
	emitter := mustEmitter(t, ModeText, false, success.streams(false))
	status, err := emitter.Emit(Outcome{
		Result: mustResult(t, validSpec(t, CommandList)), Rendered: "payments-api  running",
	})
	if err != nil || status != SuccessExitStatus {
		t.Fatalf("Emit = %d/%v", status, err)
	}
	if !strings.Contains(success.stdout.String(), "payments-api") {
		t.Fatalf("success data did not reach stdout: %q", success.stdout.String())
	}
	if success.stderr.Len() != 0 {
		t.Fatalf("success data reached stderr: %q", success.stderr.String())
	}
	if strings.Contains(success.stdout.String(), Schema) {
		t.Fatalf("text mode emitted the JSON document")
	}

	var failed capture
	emitter = mustEmitter(t, ModeText, false, failed.streams(false))
	status, err = emitter.Emit(Outcome{
		Failure: mustFailure(t, "workspace_conflict"), Rendered: "destination differs",
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if status != 5 {
		t.Fatalf("status = %d, want the workspace_conflict exit 5", status)
	}
	if failed.stdout.Len() != 0 {
		t.Fatalf("a diagnostic reached stdout: %q", failed.stdout.String())
	}
	if !strings.Contains(failed.stderr.String(), "destination differs") {
		t.Fatalf("the diagnostic did not reach stderr: %q", failed.stderr.String())
	}

	var missing capture
	emitter = mustEmitter(t, ModeText, false, missing.streams(false))
	if _, err := emitter.Emit(Outcome{Result: mustResult(t, validSpec(t, CommandList))}); !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("text mode admitted an outcome with no human rendering: %v", err)
	}
}

// TestProgressUsesStderrOnlyWhenItIsATTY narrows the Section 14.2 permission
// "progress MAY use stderr only when it is a TTY". A non-TTY stderr must not
// receive the line, and the line must not be redirected to stdout instead.
func TestProgressUsesStderrOnlyWhenItIsATTY(t *testing.T) {
	var tty capture
	emitter := mustEmitter(t, ModeJSON, false, tty.streams(true))
	written, err := emitter.Progress("staging")
	if err != nil || !written {
		t.Fatalf("Progress on a TTY = %t/%v", written, err)
	}
	if !strings.Contains(tty.stderr.String(), "staging") {
		t.Fatalf("progress did not reach a TTY stderr")
	}

	var pipe capture
	emitter = mustEmitter(t, ModeJSON, false, pipe.streams(false))
	written, err = emitter.Progress("staging")
	if err != nil {
		t.Fatalf("Progress on a pipe: %v", err)
	}
	if written {
		t.Fatalf("Progress reported a write to a non-TTY stderr")
	}
	if pipe.stderr.Len() != 0 || pipe.stdout.Len() != 0 {
		t.Fatalf("progress reached a stream: out=%q err=%q", pipe.stdout.String(), pipe.stderr.String())
	}
}

// TestNonInteractiveForbidsPrompts narrows the Section 14.2 sentence
// "--non-interactive, which forbids prompts". A prompt that silently degraded
// to a default would be the confirmation bypass the same section forbids.
func TestNonInteractiveForbidsPrompts(t *testing.T) {
	var interactive capture
	emitter := mustEmitter(t, ModeText, false, interactive.streams(true))
	if err := emitter.Prompt("replace the managed replica? [y/N]"); err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if !strings.Contains(interactive.stderr.String(), "replace the managed replica?") {
		t.Fatalf("prompt did not reach stderr")
	}
	if interactive.stdout.Len() != 0 {
		t.Fatalf("prompt reached stdout")
	}

	var headless capture
	emitter = mustEmitter(t, ModeText, true, headless.streams(false))
	if err := emitter.Prompt("replace the managed replica? [y/N]"); !errors.Is(err, ErrPromptForbidden) {
		t.Fatalf("Prompt under --non-interactive = %v, want ErrPromptForbidden", err)
	}
	if headless.stderr.Len() != 0 || headless.stdout.Len() != 0 {
		t.Fatalf("a forbidden prompt was still written")
	}
}

// TestProcessExitStatusEqualsTheStructuredErrorExitCode is the Section 14.2
// sentence "the process exit status MUST equal that error's exit_code",
// measured over every registered Section 15.2 failure class rather than
// asserted for one code.
func TestProcessExitStatusEqualsTheStructuredErrorExitCode(t *testing.T) {
	// One code per Section 15.3 exit class, written out rather than derived.
	reviewed := map[axerror.Code]int{
		"invalid_arguments":             2,
		"invalid_config":                3,
		"not_found":                     4,
		"workspace_conflict":            5,
		"capability_unavailable":        6,
		"authentication_failed":         7,
		"transport_failure":             8,
		"integrity_failure":             9,
		"not_owner":                     10,
		"quiesce_timeout":               11,
		"materialization_failed":        12,
		"provider_process_failed":       13,
		"task_board_bridge_unavailable": 14,
		"partial_sync":                  15,
		"policy_refused":                16,
		"migration_required":            17,
		"interrupted":                   130,
	}
	if len(reviewed) != 17 {
		t.Fatalf("reviewed table has %d classes, Section 15.2 registers 17 failure classes", len(reviewed))
	}
	for code, want := range reviewed {
		t.Run(string(code), func(t *testing.T) {
			failure := mustFailure(t, code)
			if failure.ExitCode() != want {
				t.Fatalf("%q maps to exit %d, want %d", code, failure.ExitCode(), want)
			}
			var streams capture
			emitter := mustEmitter(t, ModeJSON, true, streams.streams(false))
			status, err := emitter.Emit(Outcome{Failure: failure})
			if err != nil {
				t.Fatalf("Emit: %v", err)
			}
			if status != want {
				t.Fatalf("process exit status = %d, exit_code = %d", status, want)
			}
			// The emitted document is the Structured Error itself, and its
			// exit_code member equals the status the caller must exit with.
			var document map[string]any
			decoder := json.NewDecoder(bytes.NewReader(streams.stdout.Bytes()))
			decoder.UseNumber()
			if err := decoder.Decode(&document); err != nil {
				t.Fatalf("decode emitted failure: %v", err)
			}
			if document["schema"] != "urn:ax:schema:error" {
				t.Fatalf("emitted schema = %v, want the Structured Error schema", document["schema"])
			}
			if document["exit_code"].(json.Number).String() != json.Number(itoa(want)).String() {
				t.Fatalf("emitted exit_code = %v, want %d", document["exit_code"], want)
			}
			if _, present := document["ok"]; present {
				t.Fatalf("a failure was emitted as a CLI Result with ok")
			}
			if standalone, err := ExitStatus(Outcome{Failure: failure}); err != nil || standalone != want {
				t.Fatalf("ExitStatus = %d/%v, want %d", standalone, err, want)
			}
		})
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

// TestOutcomeIsASuccessOrAFailureNeverBoth proves an ambiguous outcome cannot
// be emitted at all, so no exit status is ever derived from a value that could
// be read either way.
func TestOutcomeIsASuccessOrAFailureNeverBoth(t *testing.T) {
	result := mustResult(t, validSpec(t, CommandList))
	failure := mustFailure(t, "not_found")
	for name, outcome := range map[string]Outcome{
		"both":    {Result: result, Failure: failure},
		"neither": {},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ExitStatus(outcome); !errors.Is(err, ErrStreamDiscipline) {
				t.Fatalf("ExitStatus(%s) = %v, want ErrStreamDiscipline", name, err)
			}
			var streams capture
			emitter := mustEmitter(t, ModeJSON, false, streams.streams(false))
			if _, err := emitter.Emit(outcome); !errors.Is(err, ErrStreamDiscipline) {
				t.Fatalf("Emit(%s) = %v, want ErrStreamDiscipline", name, err)
			}
			if streams.stdout.Len() != 0 {
				t.Fatalf("an ambiguous outcome reached stdout")
			}
		})
	}
}

// TestEmitterRefusesAnIncompleteConfiguration proves the boundary does not
// substitute a default stream or mode for a missing one.
func TestEmitterRefusesAnIncompleteConfiguration(t *testing.T) {
	var streams capture
	if _, err := newEmitter("structured", false, streams.streams(false)); !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("an unknown mode was admitted")
	}
	if _, err := newEmitter(ModeJSON, false, Streams{Stderr: &streams.stderr}); !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("a missing stdout was admitted")
	}
	if _, err := newEmitter(ModeJSON, false, Streams{Stdout: &streams.stdout}); !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("a missing stderr was admitted")
	}
	if _, err := NewEmitter(nil, streams.streams(false)); !errors.Is(err, ErrStreamDiscipline) {
		t.Fatalf("a nil invocation was admitted")
	}
}

// TestEmitterModeFollowsTheParsedInvocation wires the production parser to the
// production boundary, so the mode a command runs in is the mode --json
// selected and not a second, parallel decision.
func TestEmitterModeFollowsTheParsedInvocation(t *testing.T) {
	var streams capture
	invocation, failure := ParseCommonFlags(SurfaceList, []string{"--json", "--non-interactive"})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	emitter, err := NewEmitter(invocation, streams.streams(false))
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if emitter.Mode() != ModeJSON {
		t.Fatalf("mode = %q, want json", emitter.Mode())
	}
	if err := emitter.Prompt("continue?"); !errors.Is(err, ErrPromptForbidden) {
		t.Fatalf("the parsed --non-interactive did not reach the emitter: %v", err)
	}
}
