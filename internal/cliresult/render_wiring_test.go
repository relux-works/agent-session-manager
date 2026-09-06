package cliresult

import (
	"errors"
	"strings"
	"testing"
)

// hostileTerminalLine carries one member of every class Section 16.7
// names: an ANSI color sequence, an OSC hyperlink, a C0 control, DEL, a
// bidi override, a zero-width joiner, a secret-shaped assignment, and a
// private-key block opener.
const hostileTerminalLine = "provider pi says \x1b[31mred\x1b[0m and \x1b]8;;https://evil\x07link" +
	" with bell\x07 del\x7f bidi\u202e join\u200d password=hunter2-hunter key -----BEGIN EC PRIVATE KEY-----"

// TestTextEmissionNeutralizesHostileContent drives hostile provider-derived
// text through the production text paths (Log, Prompt, and failure Emit)
// and requires inert output: no ESC byte, no C0 byte except newline, no
// DEL, valid UTF-8, the secret masked, and the key block redacted.
func TestTextEmissionNeutralizesHostileContent(t *testing.T) {
	t.Parallel()
	var sink capture
	emitter := mustEmitter(t, ModeText, false, Streams{Stdout: &sink.stdout, Stderr: &sink.stderr})
	if err := emitter.Log(hostileTerminalLine); err != nil {
		t.Fatal(err)
	}
	assertInertTerminal(t, sink.stderr.String(), "Log")
	sink.stderr.Reset()

	if err := emitter.Prompt(hostileTerminalLine); err != nil {
		t.Fatal(err)
	}
	assertInertTerminal(t, sink.stderr.String(), "Prompt")
	sink.stderr.Reset()

	failure := mustFailure(t, "workspace_conflict")
	emitter = mustEmitter(t, ModeText, false, Streams{Stdout: &sink.stdout, Stderr: &sink.stderr})
	if _, err := emitter.Emit(Outcome{Failure: failure, Rendered: hostileTerminalLine}); err != nil {
		t.Fatal(err)
	}
	assertInertTerminal(t, sink.stderr.String(), "Emit")
	if sink.stdout.Len() != 0 {
		t.Fatalf("failure rendering reached stdout: %q", sink.stdout.String())
	}
}

func assertInertTerminal(t *testing.T, output, path string) {
	t.Helper()
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("%s output %q is not newline-terminated", path, output)
	}
	for _, character := range output {
		switch {
		case character == '\n' || character == '\t':
		case character < 0x20 || character == 0x7f || character == 0x1b:
			t.Fatalf("%s output carries raw control %U in %q", path, character, output)
		}
	}
	if strings.Contains(output, "hunter2-hunter") {
		t.Fatalf("%s output carries the secret value: %q", path, output)
	}
	if strings.Contains(output, "PRIVATE KEY-----") {
		t.Fatalf("%s output carries the key block: %q", path, output)
	}
	if !strings.Contains(output, "[redacted]") {
		t.Fatalf("%s output masked nothing: %q", path, output)
	}
}

// TestTextSuccessEmissionNeutralizes drives hostile text through the
// primary human stdout path: a success Emit in text mode. Without the
// terminalLine wire at that site the raw escape sequence reaches
// stdout, so this test kills the success-Emit narrowing mutant the
// failure-only test cannot see.
func TestTextSuccessEmissionNeutralizes(t *testing.T) {
	t.Parallel()
	var sink capture
	emitter := mustEmitter(t, ModeText, false, Streams{Stdout: &sink.stdout, Stderr: &sink.stderr})
	success := mustResult(t, validSpec(t, CommandList))
	if _, err := emitter.Emit(Outcome{Result: success, Rendered: hostileTerminalLine}); err != nil {
		t.Fatal(err)
	}
	assertInertTerminal(t, sink.stdout.String(), "Emit success")
	if sink.stderr.Len() != 0 {
		t.Fatalf("success rendering reached stderr: %q", sink.stderr.String())
	}
}

// TestProgressEmissionNeutralizes drives hostile text through Progress
// on a TTY stderr. The routing tests pin when progress writes; this
// test pins what it writes: without the terminalLine wire the raw
// controls reach the terminal.
func TestProgressEmissionNeutralizes(t *testing.T) {
	t.Parallel()
	var sink capture
	emitter := mustEmitter(t, ModeText, false, sink.streams(true))
	written, err := emitter.Progress(hostileTerminalLine)
	if err != nil {
		t.Fatal(err)
	}
	if !written {
		t.Fatal("Progress on a TTY reported no write")
	}
	assertInertTerminal(t, sink.stderr.String(), "Progress")
}

// TestPromptRefusalCarriesNeutralizedText pins the non-interactive
// prompt refusal end to end: it refuses with ErrPromptForbidden and
// the refused text is neutralized, not raw. The expected string is
// hand-computed from the escape rule (ESC becomes U+241B, the rest of
// this line is printable), so an unwired site fails on the exact
// comparison rather than hiding behind %q quoting.
func TestPromptRefusalCarriesNeutralizedText(t *testing.T) {
	t.Parallel()
	var sink capture
	emitter := mustEmitter(t, ModeText, true, Streams{Stdout: &sink.stdout, Stderr: &sink.stderr})
	line := "wipe \x1b[31mred\x1b[0m all?"
	err := emitter.Prompt(line)
	if err == nil {
		t.Fatal("Prompt under --non-interactive admitted")
	}
	if !errors.Is(err, ErrPromptForbidden) {
		t.Fatalf("Prompt refusal = %v, want ErrPromptForbidden", err)
	}
	want := `prompt forbidden by --non-interactive: "wipe ␛[31mred␛[0m all?"`
	if err.Error() != want {
		t.Fatalf("Prompt refusal = %q, want %q", err.Error(), want)
	}
	if sink.stderr.Len() != 0 {
		t.Fatalf("refused prompt reached stderr: %q", sink.stderr.String())
	}
}

// TestJSONModeEmissionStaysByteExact proves the escaping wire stops at
// text mode: JSON stdout carries the exact MarshalJSON document with no
// terminal neutralization applied, because machine output must stay
// byte-exact.
func TestJSONModeEmissionStaysByteExact(t *testing.T) {
	t.Parallel()
	var sink capture
	emitter := mustEmitter(t, ModeJSON, true, Streams{Stdout: &sink.stdout, Stderr: &sink.stderr})
	failure := mustFailure(t, "workspace_conflict")
	if _, err := emitter.Emit(Outcome{Failure: failure}); err != nil {
		t.Fatal(err)
	}
	document, err := failure.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if sink.stdout.String() != string(document)+"\n" {
		t.Fatalf("JSON stdout = %q, want the exact document", sink.stdout.String())
	}
}

// TestBenignTextPassesThroughUnchanged proves the neutralization is
// narrow: ordinary operator text with newlines and tabs reaches the
// stream byte-identical.
func TestBenignTextPassesThroughUnchanged(t *testing.T) {
	t.Parallel()
	var sink capture
	emitter := mustEmitter(t, ModeText, false, Streams{Stdout: &sink.stdout, Stderr: &sink.stderr})
	line := "sync converged\t3 sessions\nall replicas fresh"
	if err := emitter.Log(line); err != nil {
		t.Fatal(err)
	}
	if sink.stderr.String() != line+"\n" {
		t.Fatalf("Log(%q) = %q, want byte-identical", line, sink.stderr.String())
	}
}
