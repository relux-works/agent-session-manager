package cliresult

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// duplicateMember returns document with member repeated as its LAST top-level
// member, the second occurrence carrying replacement.
//
// The second occurrence is appended rather than prepended because which one
// encoding/json keeps is the whole point: a scalar member resolves to the last
// occurrence, so the appended value is what a resolving reader would answer
// with. A prepended copy would leave the original value in place and the row
// would pass for the wrong reason.
//
// The rewrite is textual because no builder in this repository can emit the
// shape - a Go map has one value per key - which is also why the shape only ever
// arrives from a writer that composed the bytes itself.
func duplicateMember(t *testing.T, document []byte, member, replacement string) []byte {
	t.Helper()
	text := string(document)
	anchor := `"` + member + `":`
	if !strings.Contains(text, anchor) && !strings.Contains(text, `"`+member+`" :`) {
		t.Fatalf("document carries no %q member to duplicate", member)
	}
	closing := strings.LastIndex(text, "}")
	if closing < 0 {
		t.Fatal("document has no closing brace")
	}
	injected := text[:closing] + `, "` + member + `": ` + replacement + text[closing:]
	var lenient map[string]json.RawMessage
	if err := json.Unmarshal([]byte(injected), &lenient); err != nil {
		t.Fatalf("the rewrite produced bytes encoding/json itself refuses: %v", err)
	}
	return []byte(injected)
}

// TestReadRefusesADuplicateKeyedDocumentOnEitherBranch is the reading-side half
// of the Section 1.6 common-data-model gate.
//
// Read discriminates the two contracts by the schema member and then hands the
// document to the closed decoder for the branch it chose. Until this leaf's
// rework only ONE of those two decoders enforced the common data model:
// cliresult.Decode canonicalized, axerror.Decode did not. So the same reader
// refused a duplicate-keyed CLI Result and admitted a duplicate-keyed Structured
// Error, and the exit-status corroboration this leaf added did not catch it
// because both occurrences of a repeated code can share one exit class.
//
// Every row therefore drives Read, the production entry point, not either
// decoder directly, and every row is run on BOTH branches so a gate restored to
// one of them cannot pass here.
func TestReadRefusesADuplicateKeyedDocumentOnEitherBranch(t *testing.T) {
	t.Parallel()

	failure, _ := mustEmittedFailure(t, "workspace_conflict")
	success := mustEmittedSuccess(t, CommandTakeover)

	for _, branch := range []struct {
		name    string
		output  InvocationOutput
		command Command
		members map[string]string
	}{
		{
			name:    "structured error",
			output:  failure,
			command: CommandMaterialize,
			members: map[string]string{
				// A second code inside the SAME exit class, which is the case
				// the exit_code equality check cannot see.
				"code":      `"destination_not_empty"`,
				"retryable": `true`,
				"details":   `{"expected_checkpoint": "sha256:1111111111111111111111111111111111111111111111111111111111111111"}`,
				"schema":    `"urn:ax:schema:error"`,
				"exit_code": `5`,
				"message":   `"a second message"`,
			},
		},
		{
			name:    "cli result",
			output:  success,
			command: CommandTakeover,
			members: map[string]string{
				"command":    `"takeover"`,
				"ok":         `true`,
				"schema":     `"urn:ax:schema:cli-result"`,
				"body":       `{}`,
				"extensions": `{}`,
				"session_id": `null`,
			},
		},
	} {
		t.Run(branch.name, func(t *testing.T) {
			if _, err := Read(branch.command, branch.output); err != nil {
				t.Fatalf("the undoubled invocation was refused, so these rows prove nothing: %v", err)
			}
			for member, replacement := range branch.members {
				t.Run(member, func(t *testing.T) {
					stdout := duplicateMember(t, branch.output.Stdout, member, replacement)
					reading, err := Read(branch.command, InvocationOutput{
						Stdout: stdout, ExitStatus: branch.output.ExitStatus})
					if err == nil {
						code, _ := reading.Code()
						t.Fatalf("Read admitted a duplicate %q: succeeded=%v code=%q retryable=%v",
							member, reading.Succeeded(), code, reading.Retryable())
					}
					if !errors.Is(err, ErrUnreadableDocument) {
						t.Fatalf("Read(duplicate %s) error = %v, want ErrUnreadableDocument", member, err)
					}
					if !strings.Contains(err.Error(), `duplicate object member "`+member+`"`) {
						t.Fatalf("Read(duplicate %s) error = %v, want the repeated member named; "+
							"a gate narrowed to another member set would pass without this",
							member, err)
					}
				})
			}
		})
	}
}

// TestADuplicateMemberCannotForgeARetryClaimThroughRead states the finding as
// the machine answer it changed rather than as a parse fact, at the entry point
// a client actually calls.
//
// The document declares retryable = false and repeats the member as true. Before
// the fix Read returned a Reading whose Retryable() was true: a retry claim
// assembled from two members, neither of which a conforming writer could emit,
// on a code whose retryability refusal table was never consulted for it. The
// pre-fix resolution is measured here rather than described, so the row cannot
// decay into prose about a defect nobody re-derives.
func TestADuplicateMemberCannotForgeARetryClaimThroughRead(t *testing.T) {
	t.Parallel()

	failure, _ := mustEmittedFailure(t, "workspace_conflict")
	baseline, err := Read(CommandMaterialize, failure)
	if err != nil {
		t.Fatalf("Read(conforming failure): %v", err)
	}
	if baseline.Retryable() {
		t.Fatal("the fixture already claims retryable, so the forged row would prove nothing")
	}

	stdout := duplicateMember(t, failure.Stdout, "retryable", "true")
	var resolved struct {
		Retryable *bool `json:"retryable"`
	}
	if err := json.Unmarshal(stdout, &resolved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resolved.Retryable == nil || !*resolved.Retryable {
		t.Fatalf("encoding/json resolved the repeated retryable to %v, want the appended true; "+
			"if this changed, the forged-claim shape this gate refuses changed with it", resolved.Retryable)
	}

	reading, err := Read(CommandMaterialize, InvocationOutput{Stdout: stdout, ExitStatus: failure.ExitStatus})
	if err == nil {
		t.Fatalf("Read admitted a forged retry claim: Retryable() = %v", reading.Retryable())
	}
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Read(forged retry) error = %v, want ErrUnreadableDocument", err)
	}
}

// TestADuplicateSchemaMemberCannotSelectTheBranch pins the one duplicate the
// discriminator itself has to settle, because neither closed decoder can settle
// it for the discriminator: both run after the branch is chosen.
//
// A repeated schema member resolves to the last occurrence in a map decode, so a
// document declaring the Structured Error schema and repeating it as the CLI
// Result schema routed the reading to readSuccess, which then answered with a
// disagreement between stdout and the exit status - a fact about the exit status
// that has nothing to do with what is wrong with the bytes.
func TestADuplicateSchemaMemberCannotSelectTheBranch(t *testing.T) {
	t.Parallel()

	failure, _ := mustEmittedFailure(t, "workspace_conflict")
	stdout := duplicateMember(t, failure.Stdout, "schema", `"`+Schema+`"`)

	var resolved struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(stdout, &resolved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resolved.Schema != Schema {
		t.Fatalf("encoding/json resolved the repeated schema to %q, want the appended %q; "+
			"the branch-selection shape this row exists for is gone", resolved.Schema, Schema)
	}

	_, err := Read(CommandMaterialize, InvocationOutput{Stdout: stdout, ExitStatus: failure.ExitStatus})
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Read(duplicate schema) error = %v, want ErrUnreadableDocument", err)
	}
	if errors.Is(err, ErrOutcomeDisagreement) {
		t.Fatalf("Read(duplicate schema) error = %v; the reading reported an exit-status "+
			"disagreement for a document whose schema it never settled", err)
	}
}

// TestADuplicateSchemaMemberCannotSelectTheBranchInALargeDocument is the same
// gate at a payload size no hand-built fixture reaches.
//
// The row above builds a short document, as does every duplicate-member row in
// this package and in axerror. A gate narrowed by payload SIZE - `if
// len(stdout) < 4096 { ... }` - therefore refuses all of them and admits a large
// one, and that narrowing survived the whole repository suite when this leaf was
// reviewed. Byte length is not a dimension of the Section 1.6 contract, so this
// is not a rule being asserted. It is the one predicate a peer chooses: stdout
// is whatever the invoked process wrote, and its length is not bounded by
// anything this reader controls.
//
// The document is a real CLI Result carrying twelve Session Summaries, emitted
// through the production emitter, so the only thing separating it from a
// conforming invocation is the repeated schema member.
func TestADuplicateSchemaMemberCannotSelectTheBranchInALargeDocument(t *testing.T) {
	t.Parallel()

	spec := validSpec(t, CommandList)
	sessions := make([]any, 0, 12)
	for index := 0; index < 12; index++ {
		sessions = append(sessions, sessionSummary(fmt.Sprintf("0198f4c8-3e70-7a11-8a2b-12345678900%x", index)))
	}
	spec.Body = mutateBody(spec.Body.(map[string]any), "sessions", sessions)
	var streams capture
	emitter := mustEmitter(t, ModeJSON, true, streams.streams(false))
	if _, err := emitter.Emit(Outcome{Result: mustResult(t, spec)}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	conforming := InvocationOutput{Stdout: streams.stdout.Bytes(), ExitStatus: 0}
	if len(conforming.Stdout) <= 4096 {
		t.Fatalf("the padded result is %d bytes, which does not exceed the short fixtures this row exists to leave behind",
			len(conforming.Stdout))
	}
	if _, err := Read(CommandList, conforming); err != nil {
		t.Fatalf("the padded result without a duplicate was refused, so this row proves nothing: %v", err)
	}

	stdout := duplicateMember(t, conforming.Stdout, "schema", `"`+axerror.Schema+`"`)
	if len(stdout) <= 4096 {
		t.Fatalf("the duplicated document is %d bytes", len(stdout))
	}
	_, err := Read(CommandList, InvocationOutput{Stdout: stdout, ExitStatus: 0})
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Read(%d-byte document, duplicate schema) error = %v, want ErrUnreadableDocument", len(stdout), err)
	}
	if errors.Is(err, ErrOutcomeDisagreement) || errors.Is(err, ErrForeignDocument) {
		t.Fatalf("Read(duplicate schema) error = %v; the reading answered from a schema it never settled", err)
	}
}

// TestBothClosedDecodersEnforceTheSameDataModel drives each contract's own
// reader directly, so the claim documentSchema makes about its delegation is
// checked against both branches rather than assumed for one.
//
// That sentence was false once. It said "the closed decoder for the selected
// contract then validates the whole object, including the duplicate members
// this discriminator deliberately does not settle on its own", which was true of
// Decode and false of axerror.Decode.
func TestBothClosedDecodersEnforceTheSameDataModel(t *testing.T) {
	t.Parallel()

	t.Run("cliresult.Decode", func(t *testing.T) {
		document := duplicateMember(t, []byte(normativeCLISuccess), "ok", "true")
		if _, err := Decode(Version100, document); err == nil {
			t.Fatal("Decode admitted a duplicate-keyed CLI Result")
		}
	})

	t.Run("axerror.Decode", func(t *testing.T) {
		failure, _ := mustEmittedFailure(t, "workspace_conflict")
		document := duplicateMember(t, failure.Stdout, "retryable", "true")
		if _, err := axerror.Decode(axerror.Version100, document); err == nil {
			t.Fatal("axerror.Decode admitted a duplicate-keyed Structured Error")
		}
	})
}
