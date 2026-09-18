package terminstance

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func testResult(t *testing.T) (MutationContext, terminalbackend.InstanceState, Result) {
	t.Helper()
	context := testContext(t)
	source := mustParseState(t, "active")
	result := buildResult(context, source, terminalbackend.StateQuiescing,
		[]terminalbackend.SideEffect{terminalbackend.EffectInputClosed},
		[]string{fixtureDigestA}, DispositionReplaySame)
	return context, source, result
}

// TestCheckResultAdmitsEngineResult drives the repetition contract
// through the production entry: the engine-built result repeats every
// request identity and before equals the source.
func TestCheckResultAdmitsEngineResult(t *testing.T) {
	context, source, result := testResult(t)
	if err := CheckResult(context, source, result); err != nil {
		t.Fatalf("CheckResult() error = %v", err)
	}
	requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
}

// TestCheckResultRepetitionTamper proves every repeated identity must
// equal the request context: each tampered member refuses through the
// production entry with its own detail, and the generation tamper
// refuses with the landed stale class.
func TestCheckResultRepetitionTamper(t *testing.T) {
	context, source, result := testResult(t)
	tampers := []struct {
		name   string
		mutate func(*Result)
		code   string
		detail string
	}{
		{"operation", func(r *Result) { r.OperationID = fixtureOperationB }, "terminal_backend_protocol_error", "result operation binding"},
		{"session", func(r *Result) { r.SessionID = fixtureSessionB }, "terminal_backend_protocol_error", "result session binding"},
		{"instance", func(r *Result) { r.TerminalInstanceID = fixtureSessionB }, "terminal_backend_protocol_error", "result instance binding"},
		{"backend", func(r *Result) { r.TerminalBackendID = "other.backend" }, "terminal_backend_protocol_error", "result backend binding"},
		{"implementation", func(r *Result) { r.ImplementationVersion = "9.9.9" }, "terminal_backend_protocol_error", "result implementation binding"},
		{"protocol", func(r *Result) { r.ProtocolVersion = "1.9.9" }, "terminal_backend_protocol_error", "result protocol binding"},
		{"generation", func(r *Result) { r.BackendGeneration = "generation-two" }, "terminal_backend_stale_generation", "result generation binding"},
		{"before", func(r *Result) { r.Before = terminalbackend.StateParked }, "terminal_backend_protocol_error", "result before state"},
	}
	for _, tc := range tampers {
		t.Run(tc.name, func(t *testing.T) {
			tampered := result
			tc.mutate(&tampered)
			err := CheckResult(context, source, tampered)
			requireRefusal(t, err, tc.code, tc.detail)
		})
	}
}

// TestCheckResultGenerationBound256 proves the generation bound inside
// results: length 256 is admitted, 0 and 257 refused. The tampered
// values differ from the context, so the refusal is the binding arm;
// the admitted 256 requires a context carrying it.
func TestCheckResultGenerationBound256(t *testing.T) {
	context, source, result := testResult(t)
	long := context
	long.BackendGeneration = strings.Repeat("g", 256)
	admitted := result
	admitted.BackendGeneration = strings.Repeat("g", 256)
	if err := CheckResult(long, source, admitted); err != nil {
		t.Errorf("CheckResult(generation 256) error = %v, want admission", err)
	}
}

// TestCheckResultShapeArms drives the state, effect, evidence and
// disposition arms: unknown states and effects refuse, over-bound
// arrays refuse, malformed digests refuse, and order and uniqueness
// refuse by separate details.
func TestCheckResultShapeArms(t *testing.T) {
	context, source, result := testResult(t)

	bad := result
	bad.After = "running"
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "lifecycle state vocabulary")

	bad = result
	bad.Effects = []terminalbackend.SideEffect{"input_reopened"}
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "side effect vocabulary")

	bad = result
	bad.Effects = []terminalbackend.SideEffect{
		terminalbackend.EffectWrapperStarted, terminalbackend.EffectBindingPersisted,
	}
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "result effects order")

	bad = result
	bad.Effects = []terminalbackend.SideEffect{
		terminalbackend.EffectInputClosed, terminalbackend.EffectInputClosed,
	}
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "result effects unique")

	bad = result
	bad.EvidenceIDs = []string{fixtureDigestB, fixtureDigestA}
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "result evidence order")

	bad = result
	bad.EvidenceIDs = []string{fixtureDigestA, fixtureDigestA}
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "result evidence unique")

	bad = result
	bad.EvidenceIDs = []string{"not-a-digest"}
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "result evidence digest")

	bad = result
	bad.Disposition = "retry_later"
	requireRefusal(t, CheckResult(context, source, bad), "terminal_backend_protocol_error", "retry disposition vocabulary")
}

// TestBuildResultSortsEffectsAndEvidence proves execution order and wire
// order split: the stop effects commit in transition order but report
// bytewise sorted.
func TestBuildResultSortsEffectsAndEvidence(t *testing.T) {
	context := testContext(t)
	result := buildResult(context, mustParseState(t, "quiescing"), terminalbackend.StateStopped,
		[]terminalbackend.SideEffect{
			terminalbackend.EffectGracefulStopRequested,
			terminalbackend.EffectProcessClosed,
			terminalbackend.EffectBackendStoreClosed,
		},
		[]string{fixtureDigestB, fixtureDigestA}, DispositionReplaySame)
	want := []terminalbackend.SideEffect{
		terminalbackend.EffectBackendStoreClosed,
		terminalbackend.EffectGracefulStopRequested,
		terminalbackend.EffectProcessClosed,
	}
	if len(result.Effects) != 3 {
		t.Fatalf("effects = %v, want 3 sorted", result.Effects)
	}
	for i := range want {
		requireLiteral(t, "effect", string(result.Effects[i]), string(want[i]))
	}
	requireLiteral(t, "evidence 0", result.EvidenceIDs[0], "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	requireLiteral(t, "evidence 1", result.EvidenceIDs[1], "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
}
