package cliresult

import (
	"testing"
	"time"
)

// TestEveryUserCommandSupportsTheTenCommonFlags drives the production parser
// with each of the Section 14.2 flags for each user command. It is the
// inventory the sentence "all user commands MUST support" names, measured
// rather than asserted in prose: 29 user surfaces times ten flags.
func TestEveryUserCommandSupportsTheTenCommonFlags(t *testing.T) {
	flags := CommonFlags()
	if len(flags) != 10 {
		t.Fatalf("common flags = %v, want the ten Section 14.2 flags", flags)
	}
	reviewed := []CommonFlag{
		"--config", "--data-dir", "--state-dir", "--cache-dir", "--runtime-dir",
		"--json", "--no-color", "--non-interactive", "--timeout", "--verbose",
	}
	for index, flag := range reviewed {
		if flags[index] != flag {
			t.Fatalf("flag %d = %q, want %q", index, flags[index], flag)
		}
	}
	users := UserSurfaces()
	if len(users) != len(Surfaces())-2 {
		t.Fatalf("user surfaces = %d of %d; Section 14.1 names exactly two internal commands",
			len(users), len(Surfaces()))
	}
	for _, surface := range users {
		for _, flag := range flags {
			argv := []string{string(flag)}
			if _, valued := valuedFlags[flag]; valued {
				value := "/tmp/value"
				if flag == FlagTimeout {
					value = "1500"
				}
				argv = append(argv, value)
			}
			invocation, failure := ParseCommonFlags(surface, argv)
			if failure != nil {
				t.Fatalf("%q rejected %s: %s", surface, flag, failure.Message())
			}
			if len(invocation.Operands()) != 0 {
				t.Fatalf("%q treated %s as an operand", surface, flag)
			}
		}
	}
}

// TestYesIsAcceptedOnlyWhereAConfirmationIsDocumented narrows the Section 14.2
// sentence "commands with a documented confirmation additionally accept --yes;
// commands without such a confirmation MUST reject it". Each surface is checked
// on its own row, so a gate widened to admit one extra command fails here.
func TestYesIsAcceptedOnlyWhereAConfirmationIsDocumented(t *testing.T) {
	accepting := map[SurfaceCommand]struct{}{
		SurfaceTakeover: {}, SurfaceStop: {}, SurfaceMaterialize: {},
	}
	for _, surface := range Surfaces() {
		_, expected := accepting[surface]
		got, err := AcceptsYes(surface)
		if err != nil {
			t.Fatalf("AcceptsYes(%q): %v", surface, err)
		}
		if got != expected {
			t.Fatalf("AcceptsYes(%q) = %t, want %t", surface, got, expected)
		}
		invocation, failure := ParseCommonFlags(surface, []string{"--yes"})
		if expected {
			if failure != nil {
				t.Fatalf("%q rejected --yes: %s", surface, failure.Message())
			}
			if !invocation.Yes() {
				t.Fatalf("%q parsed --yes but did not record it", surface)
			}
			continue
		}
		if failure == nil {
			t.Fatalf("%q accepted --yes without a documented confirmation", surface)
		}
		if failure.Code() != "invalid_arguments" || failure.ExitCode() != 2 {
			t.Fatalf("%q refused --yes with %q/%d, want invalid_arguments/2",
				surface, failure.Code(), failure.ExitCode())
		}
	}
}

// TestRPCServeRejectsJSON narrows the Section 14.2 sentence "the internal
// streaming command ax rpc serve --stdio is an RPC protocol endpoint, not a CLI
// Result producer, and MUST reject --json", and proves the refusal is scoped to
// that one surface rather than applied to a class.
func TestRPCServeRejectsJSON(t *testing.T) {
	invocation, failure := ParseCommonFlags(SurfaceRPCServe, []string{"--stdio", "--json"})
	if failure == nil {
		t.Fatalf("rpc serve accepted --json")
	}
	if failure.Code() != "invalid_arguments" || failure.ExitCode() != 2 {
		t.Fatalf("refusal is %q/%d, want invalid_arguments/2", failure.Code(), failure.ExitCode())
	}
	if invocation != nil {
		t.Fatalf("a refused invocation was returned")
	}
	// Without --json it parses, and --stdio is handed back as an operand for
	// the surface that owns it.
	invocation, failure = ParseCommonFlags(SurfaceRPCServe, []string{"--stdio"})
	if failure != nil {
		t.Fatalf("rpc serve rejected --stdio: %s", failure.Message())
	}
	if operands := invocation.Operands(); len(operands) != 1 || operands[0] != "--stdio" {
		t.Fatalf("operands = %v, want [--stdio]", operands)
	}
	for _, surface := range Surfaces() {
		accepts, err := AcceptsJSON(surface)
		if err != nil {
			t.Fatalf("AcceptsJSON(%q): %v", surface, err)
		}
		if accepts == (surface == SurfaceRPCServe) {
			t.Fatalf("AcceptsJSON(%q) = %t; only rpc serve rejects --json", surface, accepts)
		}
	}
	// pane is internal but still produces a CLI Result, so it accepts --json.
	if _, failure := ParseCommonFlags(SurfacePane, []string{"--json"}); failure != nil {
		t.Fatalf("pane rejected --json: %s", failure.Message())
	}
}

// TestUnknownSurfaceIsRefusedRatherThanDefaulted proves the parser does not
// invent a permissive default for a command it does not know.
func TestUnknownSurfaceIsRefusedRatherThanDefaulted(t *testing.T) {
	for _, surface := range []SurfaceCommand{"", "clone", "Takeover", "session clone"} {
		if _, failure := ParseCommonFlags(surface, []string{"--json"}); failure == nil {
			t.Fatalf("ParseCommonFlags admitted the unknown surface %q", surface)
		}
		if _, err := AcceptsYes(surface); err == nil {
			t.Fatalf("AcceptsYes admitted the unknown surface %q", surface)
		}
		if _, err := AcceptsJSON(surface); err == nil {
			t.Fatalf("AcceptsJSON admitted the unknown surface %q", surface)
		}
	}
}

// TestCommonFlagValuesAndTimeoutGrammar narrows the value-taking flags and the
// Section 1.6 rule that "durations MUST be integer milliseconds".
func TestCommonFlagValuesAndTimeoutGrammar(t *testing.T) {
	invocation, failure := ParseCommonFlags(SurfaceList, []string{
		"--config", "/etc/ax.toml", "--data-dir=/var/ax", "--timeout", "1500",
		"--no-color", "--non-interactive", "--verbose", "--json", "payments-api",
	})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	if path, ok := invocation.Path(FlagConfig); !ok || path != "/etc/ax.toml" {
		t.Fatalf("--config = %q/%t", path, ok)
	}
	if path, ok := invocation.Path(FlagDataDir); !ok || path != "/var/ax" {
		t.Fatalf("--data-dir = %q/%t", path, ok)
	}
	timeout, set := invocation.Timeout()
	if !set || timeout != 1500*time.Millisecond {
		t.Fatalf("--timeout = %v/%t", timeout, set)
	}
	if !invocation.NoColor() || !invocation.NonInteractive() || !invocation.Verbose() || !invocation.JSON() {
		t.Fatalf("a boolean common flag was dropped")
	}
	if invocation.Mode() != ModeJSON {
		t.Fatalf("mode = %q, want json", invocation.Mode())
	}
	if operands := invocation.Operands(); len(operands) != 1 || operands[0] != "payments-api" {
		t.Fatalf("operands = %v", operands)
	}
	if _, set := ParseCommonFlags(SurfaceList, nil); set != nil {
		t.Fatalf("an empty argv was refused")
	}

	refused := [][]string{
		{"--timeout"},
		{"--timeout", "soon"},
		{"--timeout", "-5"},
		{"--timeout", "1500us"},
		{"--config"},
		{"--config", ""},
		{"--json=true"},
		{"--verbose=1"},
	}
	for _, argv := range refused {
		if _, failure := ParseCommonFlags(SurfaceList, argv); failure == nil {
			t.Fatalf("ParseCommonFlags admitted %v", argv)
		}
	}
	// A Go duration that resolves to whole milliseconds is accepted.
	invocation, failure = ParseCommonFlags(SurfaceList, []string{"--timeout", "2s"})
	if failure != nil {
		t.Fatalf("--timeout 2s: %s", failure.Message())
	}
	if timeout, _ := invocation.Timeout(); timeout != 2*time.Second {
		t.Fatalf("--timeout 2s = %v", timeout)
	}
}

// TestDefaultModeIsTextAndYesDefaultsFalse pins the absence of a flag.
func TestDefaultModeIsTextAndYesDefaultsFalse(t *testing.T) {
	invocation, failure := ParseCommonFlags(SurfaceTakeover, []string{"payments-api"})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	if invocation.Mode() != ModeText || invocation.JSON() {
		t.Fatalf("default mode = %q", invocation.Mode())
	}
	if invocation.Yes() || invocation.NonInteractive() || invocation.NoColor() || invocation.Verbose() {
		t.Fatalf("a boolean flag defaulted to true")
	}
	if _, set := invocation.Timeout(); set {
		t.Fatalf("--timeout reported as set without the flag")
	}
	if invocation.Surface() != SurfaceTakeover {
		t.Fatalf("surface = %q", invocation.Surface())
	}
}
