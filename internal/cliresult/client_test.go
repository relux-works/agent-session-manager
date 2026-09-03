package cliresult

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// mustEmittedFailure renders one failure through the production emitter in JSON
// mode and returns the invocation a machine client would observe, together with
// the stderr the same invocation produced. It is the real path: the bytes under
// test are the bytes Emit wrote, not a literal a test composed.
func mustEmittedFailure(t *testing.T, code axerror.Code) (InvocationOutput, string) {
	t.Helper()
	var streams capture
	emitter := mustEmitter(t, ModeJSON, true, streams.streams(true))
	if err := emitter.Log("resolving owner"); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if _, err := emitter.Progress("transferring 3 of 9"); err != nil {
		t.Fatalf("Progress: %v", err)
	}
	status, err := emitter.Emit(Outcome{Failure: mustFailure(t, code)})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	return InvocationOutput{Stdout: streams.stdout.Bytes(), ExitStatus: status}, streams.stderr.String()
}

func mustEmittedSuccess(t *testing.T, command Command) InvocationOutput {
	t.Helper()
	var streams capture
	emitter := mustEmitter(t, ModeJSON, true, streams.streams(false))
	status, err := emitter.Emit(Outcome{Result: mustResult(t, validSpec(t, command))})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	return InvocationOutput{Stdout: streams.stdout.Bytes(), ExitStatus: status}
}

// implementedSuccessDocuments emits one conforming CLI Result per implemented
// command tag through the production emitter, so a sweep over command pairs
// moves the command tag and nothing else: every document here is exactly what
// its own invocation would have written.
//
// The count is asserted against the production enumeration rather than against
// a number retyped here - TestRefusalInventoryIsMeasuredRatherThanClaimed pins
// the 18-of-44 ratio itself - so a map that silently built fewer documents
// cannot turn a caller's cross product into a smaller sweep.
func implementedSuccessDocuments(t *testing.T) map[Command]InvocationOutput {
	t.Helper()
	implemented := ImplementedCommands()
	documents := make(map[Command]InvocationOutput, len(implemented))
	for _, command := range implemented {
		documents[command] = mustEmittedSuccess(t, command)
	}
	if len(documents) != len(implemented) || len(documents) == 0 {
		t.Fatalf("built %d success documents for %d implemented tags", len(documents), len(implemented))
	}
	return documents
}

// TestMachineReadingCannotSeeStderr is the structural half of "a machine client
// never depends on stderr". Section 14.2 puts the machine-readable answer on
// stdout and keeps logs on stderr, and this package expresses that by giving
// Read no way to receive stderr at all.
//
// The assertion is over the input type rather than over a behaviour, because a
// behavioural test can only show that today's implementation ignores a stderr
// member it has. This one fails the moment the member exists.
func TestMachineReadingCannotSeeStderr(t *testing.T) {
	t.Parallel()

	observed := reflect.TypeOf(InvocationOutput{})
	var members []string
	for index := 0; index < observed.NumField(); index++ {
		members = append(members, observed.Field(index).Name)
	}
	want := []string{"Stdout", "ExitStatus"}
	if !reflect.DeepEqual(members, want) {
		t.Fatalf("InvocationOutput members = %v, want exactly %v", members, want)
	}
	for _, member := range members {
		if strings.Contains(strings.ToLower(member), "err") && member != "ExitStatus" {
			t.Fatalf("InvocationOutput carries %q, which lets a reading depend on a diagnostic stream", member)
		}
	}
}

// TestClassificationSurvivesDiscardedStderr drives the production emitter with
// logs and progress on stderr, then classifies the invocation from stdout and
// the exit status alone. Every machine answer must be complete without a byte
// of stderr, and the stderr the same invocation produced must carry none of
// them.
func TestClassificationSurvivesDiscardedStderr(t *testing.T) {
	t.Parallel()

	output, stderr := mustEmittedFailure(t, "workspace_conflict")
	reading, err := Read(CommandMaterialize, output)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if reading.Succeeded() {
		t.Fatal("a Structured Error on stdout was read as a success")
	}
	code, present := reading.Code()
	if !present || code != "workspace_conflict" {
		t.Fatalf("Code() = %q/%t, want workspace_conflict/true", code, present)
	}
	if reading.ExitStatus() != 5 {
		t.Fatalf("ExitStatus() = %d, want 5", reading.ExitStatus())
	}
	if !reading.CodeRegistered() {
		t.Fatal("a registered code was reported as unregistered")
	}
	if stderr == "" {
		t.Fatal("the emitter wrote no stderr, so this test would pass without proving anything")
	}
	for _, forbidden := range []string{"urn:ax:schema:error", "workspace_conflict", "exit_code"} {
		if strings.Contains(stderr, forbidden) {
			t.Fatalf("stderr %q carries machine-readable fact %q", stderr, forbidden)
		}
	}
}

// TestAMisroutedDocumentLeavesTheMachineClientWithNothing is the mutant behind
// the test above. It reroutes exactly one thing - the stream the document is
// written to - and requires the machine client to end up with nothing rather
// than with a classification recovered from the exit status.
func TestAMisroutedDocumentLeavesTheMachineClientWithNothing(t *testing.T) {
	t.Parallel()

	var streams capture
	// The mutant: stdout and stderr are swapped, so Emit's one JSON document
	// lands on the diagnostic stream.
	misrouted := Streams{Stdout: &streams.stderr, Stderr: &streams.stdout, StderrIsTTY: false}
	emitter := mustEmitter(t, ModeJSON, true, misrouted)
	status, err := emitter.Emit(Outcome{Failure: mustFailure(t, "workspace_conflict")})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if streams.stderr.Len() == 0 {
		t.Fatal("the misrouted emission wrote nothing, so the mutant did not apply")
	}

	_, err = Read(CommandMaterialize, InvocationOutput{Stdout: streams.stdout.Bytes(), ExitStatus: status})
	if !errors.Is(err, ErrAbsentDocument) {
		t.Fatalf("Read(document on stderr) error = %v, want ErrAbsentDocument", err)
	}
	if !strings.Contains(err.Error(), "identifies no failure") {
		t.Fatalf("refusal %q does not state why the exit status alone is not enough", err)
	}
}

// TestReadDistinguishesAbsenceFromAReadFailure narrows the two facts a fallback
// would collapse. Nothing written and something unreadable are different
// observations of an invocation, and neither is resolved from the exit status.
func TestReadDistinguishesAbsenceFromAReadFailure(t *testing.T) {
	t.Parallel()

	valid, _ := mustEmittedFailure(t, "workspace_conflict")
	// The sentinels are addressed BY NAME rather than by value, because the
	// mutual-exclusion check below has to survive the exact mutant it exists
	// for. Aliasing ErrAbsentDocument to ErrUnreadableDocument makes the two
	// values equal, so a check that skipped "the sentinel this row wanted" by
	// comparing values would skip precisely the comparison that would have
	// caught the alias, and the mutant would survive the whole suite.
	observations := map[string]error{
		"ErrAbsentDocument":     ErrAbsentDocument,
		"ErrUnreadableDocument": ErrUnreadableDocument,
		"ErrForeignDocument":    ErrForeignDocument,
	}
	for _, test := range []struct {
		name string
		// want names the fact the row must report, keyed into observations.
		want   string
		stdout []byte
		// says, when set, is text the refusal must carry. It is used only where
		// two different guards both refuse the same bytes and the row exists to
		// pin WHICH of them answered.
		says string
	}{
		{name: "empty", want: "ErrAbsentDocument"},
		{name: "whitespace", want: "ErrAbsentDocument", stdout: []byte("  \n\t ")},
		{name: "truncated", want: "ErrUnreadableDocument", stdout: valid.Stdout[:len(valid.Stdout)/2]},
		{name: "not json", want: "ErrUnreadableDocument", stdout: []byte("workspace conflict\n")},
		{
			// Section 14.2 allows exactly one document, and the common-data-model
			// gate further along refuses the same bytes as trailing data. Both
			// refuse; the row pins which fact a caller is told, so the guard
			// cannot be deleted on the grounds that something else catches it.
			name:   "two documents",
			want:   "ErrUnreadableDocument",
			stdout: append(append([]byte{}, valid.Stdout...), valid.Stdout...),
			says:   "more than one document",
		},
		{name: "no schema member", want: "ErrUnreadableDocument", stdout: []byte(`{"code":"workspace_conflict","exit_code":5}`)},
		{name: "schema is not a string", want: "ErrUnreadableDocument", stdout: []byte(`{"schema":5}`)},
		{name: "foreign schema", want: "ErrForeignDocument", stdout: []byte(`{"schema":"urn:ax:schema:session-record"}`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			want, declared := observations[test.want]
			if !declared {
				t.Fatalf("row names %q, which is not an observation sentinel", test.want)
			}
			_, err := Read(CommandMaterialize, InvocationOutput{Stdout: test.stdout, ExitStatus: 5})
			if !errors.Is(err, want) {
				t.Fatalf("Read(%s) error = %v, want %s", test.name, err, test.want)
			}
			if test.says != "" && !strings.Contains(err.Error(), test.says) {
				t.Fatalf("Read(%s) error = %v, want the refusal to say %q; another guard "+
					"further along refuses the same bytes with a different fact",
					test.name, err, test.says)
			}
			// Membership alone does not prove a distinction. Until this
			// assertion existed, aliasing ErrAbsentDocument to
			// ErrUnreadableDocument - collapsing the two facts the doc comment
			// calls deliberately separate - survived the whole suite, because
			// every row only asked whether the answer was in the set it
			// expected. Each row now also states which facts the answer is NOT.
			for name, other := range observations {
				if name == test.want {
					continue
				}
				if errors.Is(err, other) {
					t.Fatalf("Read(%s) error = %v reports %s as well as %s; "+
						"an absence, a failure to read and a foreign contract are three facts",
						test.name, err, name, test.want)
				}
			}
		})
	}
}

// TestTheObservationSentinelsAreDistinctValues closes the same gap one level
// above the rows: the loop above compares sentinels to each other, so it is only
// meaningful while they are actually different values.
func TestTheObservationSentinelsAreDistinctValues(t *testing.T) {
	t.Parallel()

	sentinels := map[string]error{
		"ErrAbsentDocument":         ErrAbsentDocument,
		"ErrUnreadableDocument":     ErrUnreadableDocument,
		"ErrForeignDocument":        ErrForeignDocument,
		"ErrOutcomeDisagreement":    ErrOutcomeDisagreement,
		"ErrUnregisteredExitStatus": ErrUnregisteredExitStatus,
	}
	for leftName, left := range sentinels {
		for rightName, right := range sentinels {
			if leftName == rightName {
				continue
			}
			if errors.Is(left, right) {
				t.Fatalf("%s and %s are the same fact; the reader claims to report five", leftName, rightName)
			}
			if left.Error() == right.Error() {
				t.Fatalf("%s and %s carry identical text, so a caller reading the message cannot tell them apart",
					leftName, rightName)
			}
		}
	}
}

// interruptedExitStatus is the Section 15.2 row "Interrupted by operator signal
// before a clean response; inspect authority before retry". It is named because
// it is the one failure status at which a success document plausibly does reach
// stdout before the process dies, which makes a guard narrowed to spare it a
// realistic bypass rather than an artificial one.
const interruptedExitStatus = 130

// registeredFailureExitStatuses enumerates the Section 15.2 failure statuses
// through the production predicate rather than from a list retyped in a test,
// so a status added to the pinned table joins every loop that uses this.
//
// The measured count is asserted here and not left to each caller: a predicate
// that answered false for everything would otherwise turn every loop over this
// slice into a green test that drove nothing.
func registeredFailureExitStatuses(t *testing.T) []int {
	t.Helper()
	var statuses []int
	for status := 0; status <= 255; status++ {
		if axerror.IsFailureExitStatus(status) {
			statuses = append(statuses, status)
		}
	}
	if len(statuses) != 17 {
		t.Fatalf("Section 15.2 offers %d failure statuses, want the pinned 17", len(statuses))
	}
	if meaning, known := axerror.ExitStatusMeaning(interruptedExitStatus); !known ||
		!strings.Contains(meaning, "Interrupted by operator signal") {
		t.Fatalf("exit %d meaning = %q/%t, want the Section 15.2 signal-interruption row",
			interruptedExitStatus, meaning, known)
	}
	return statuses
}

// TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry covers the row
// where readFailure is the sole enforcement of Section 14.2's "the process exit
// status MUST equal that error's exit_code".
//
// For a registered code the equality is bound twice. axerror.decodeBody already
// cross-checks the document's exit_code against the pinned registry, so a
// document that misdeclares its class is refused before Read compares anything.
// For a code a later compatible minor added, ExitCodeFor takes the
// ErrUnregisteredCode branch and runs no exit check at all - Section 15.3 keeps
// the envelope's exit class, and the class is the status, not the code - so the
// guard in readFailure is the only thing left binding the document to the
// status the process actually exited with.
//
// That is exactly the row the narrowing
//
//	if failure.CodeRegistered() && failure.ExitCode() != output.ExitStatus
//
// leaves uncovered, and it survived this package's whole suite before this
// test existed. Under that mutant a machine client is handed exit class 5
// ("Workspace/native-store conflict; no silent overwrite") for a frozen
// document declaring exit class 10 (ownership/lease/fencing) - two different
// remediations, chosen from the wrong one.
//
// The row is driven over every registered Section 15.2 failure status, because
// a single sample would leave the same narrowing available one status over.
func TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry(t *testing.T) {
	t.Parallel()

	entry := historicalEntry(t, "error-1.0.0-later-minor-code.json")
	document := readHistorical(t, entry)

	conforming, err := Read(entry.command, InvocationOutput{Stdout: document, ExitStatus: entry.exitStatus})
	if err != nil {
		t.Fatalf("Read(conforming later-minor failure) error = %v", err)
	}
	if conforming.CodeRegistered() {
		t.Fatal("the fixture's code is registered for the bound envelope version, so this row no longer " +
			"covers the class in which readFailure is the only check on the exit status")
	}
	if conforming.ExitStatus() != entry.exitStatus {
		t.Fatalf("ExitStatus() = %d, want %d", conforming.ExitStatus(), entry.exitStatus)
	}

	moved := 0
	for _, status := range registeredFailureExitStatuses(t) {
		if status == entry.exitStatus {
			continue
		}
		reading, err := Read(entry.command, InvocationOutput{Stdout: document, ExitStatus: status})
		if !errors.Is(err, ErrOutcomeDisagreement) {
			classified := "no reading"
			if reading != nil {
				code, _ := reading.Code()
				classified = fmt.Sprintf("code %q at ExitStatus() %d", code, reading.ExitStatus())
			}
			t.Fatalf("Read(unregistered code declaring exit_code %d, process exit %d) error = %v (%s), "+
				"want ErrOutcomeDisagreement", entry.exitStatus, status, err, classified)
		}
		if !strings.Contains(err.Error(), "must equal that error's exit_code") {
			t.Fatalf("refusal at exit %d is %q, which does not name the Section 14.2 equality", status, err)
		}
		moved++
	}
	if moved != 16 {
		t.Fatalf("moved the status on %d rows, want the 16 registered failure statuses other than %d",
			moved, entry.exitStatus)
	}
}

// TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus is the gate that
// makes neither signal sufficient on its own. Each row keeps one signal exactly
// as a conforming invocation would produce it and moves the other.
func TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus(t *testing.T) {
	t.Parallel()

	failure, _ := mustEmittedFailure(t, "workspace_conflict")
	success := mustEmittedSuccess(t, CommandList)

	t.Run("failure object at exit 0", func(t *testing.T) {
		_, err := Read(CommandMaterialize, InvocationOutput{Stdout: failure.Stdout, ExitStatus: 0})
		if !errors.Is(err, ErrOutcomeDisagreement) {
			t.Fatalf("error = %v, want ErrOutcomeDisagreement", err)
		}
	})
	t.Run("success object at a failure status", func(t *testing.T) {
		// Driven at every registered Section 15.2 failure status rather than at
		// one sample. A single sample only shows the guard is reachable: a
		// narrowing that spares any one status - `output.ExitStatus !=
		// SuccessExitStatus && output.ExitStatus != 130` is the realistic one,
		// because 130 is the row "interrupted by operator signal before a clean
		// response" and therefore the one status at which a success document
		// plausibly does reach stdout before the process dies - stays green
		// against it. The count is asserted so the loop cannot go vacuous.
		statuses := registeredFailureExitStatuses(t)
		for _, status := range statuses {
			_, err := Read(CommandList, InvocationOutput{Stdout: success.Stdout, ExitStatus: status})
			if !errors.Is(err, ErrOutcomeDisagreement) {
				t.Fatalf("Read(success object, exit %d) error = %v, want ErrOutcomeDisagreement", status, err)
			}
		}
		if !slices.Contains(statuses, interruptedExitStatus) {
			t.Fatalf("exit %d is not among the %d statuses driven, so the signal-interruption row was not exercised",
				interruptedExitStatus, len(statuses))
		}
	})
	t.Run("failure exit_code differs from the process status", func(t *testing.T) {
		// Both statuses are registered Section 15.2 rows, so the refusal can
		// only come from the equality Section 14.2 requires.
		_, err := Read(CommandMaterialize, InvocationOutput{Stdout: failure.Stdout, ExitStatus: 12})
		if !errors.Is(err, ErrOutcomeDisagreement) {
			t.Fatalf("error = %v, want ErrOutcomeDisagreement", err)
		}
		if !strings.Contains(err.Error(), "must equal that error's exit_code") {
			t.Fatalf("refusal %q does not name the Section 14.2 equality it enforces", err)
		}
		// The same equality over the whole status vocabulary, so a narrowing
		// that spares one status is measured rather than assumed. The emitted
		// document carries exit_code 5, so every other registered failure
		// status is a disagreement.
		const declared = 5
		moved := 0
		for _, status := range registeredFailureExitStatuses(t) {
			if status == declared {
				continue
			}
			_, err := Read(CommandMaterialize, InvocationOutput{Stdout: failure.Stdout, ExitStatus: status})
			if !errors.Is(err, ErrOutcomeDisagreement) {
				t.Fatalf("Read(exit_code %d document, exit %d) error = %v, want ErrOutcomeDisagreement",
					declared, status, err)
			}
			moved++
		}
		if moved != len(registeredFailureExitStatuses(t))-1 {
			t.Fatalf("moved the status on %d rows, want every registered failure status but %d", moved, declared)
		}
	})
	t.Run("document reports another command", func(t *testing.T) {
		// Driven over the whole implemented command vocabulary in both
		// directions rather than at one ordered pair. A single pair - the
		// (invoked doctor, document says list) row this subtest used to be -
		// only shows the guard is reachable. Two narrowings stayed green
		// against it and were measured SURVIVED before this sweep existed:
		// `command != CommandList`, which spares one invoked command, and
		// `result.Command() != CommandTakeover`, which admits any document
		// CLAIMING to be a takeover result - the one body in this contract
		// carrying adoption and authority semantics - for an invocation that
		// never ran it. The cross product closes both directions at once,
		// because every implemented tag appears on each side of every pair.
		//
		// Every document here is emitted through the production emitter for
		// its own tag, so each is a conforming CLI Result that only the
		// invocation disagrees with; nothing earlier in Read has grounds to
		// refuse it. That is asserted rather than assumed: the refusal must
		// carry this guard's own sentence naming both tags, so a pair refused
		// by the version binding or the closed decoder would not count.
		documents := implementedSuccessDocuments(t)
		implemented := ImplementedCommands()
		refused, admitted := 0, 0
		for _, invoked := range implemented {
			for _, claimed := range implemented {
				output := documents[claimed]
				if invoked == claimed {
					// The positive control that stops a guard which refuses
					// everything from passing the sweep above.
					reading, err := Read(invoked, output)
					if err != nil {
						t.Fatalf("Read(%q, its own document) error = %v", invoked, err)
					}
					if reading.Command() != invoked || !reading.Succeeded() {
						t.Fatalf("Read(%q, its own document) = %q/%t, want %q/true",
							invoked, reading.Command(), reading.Succeeded(), invoked)
					}
					admitted++
					continue
				}
				_, err := Read(invoked, output)
				if !errors.Is(err, ErrOutcomeDisagreement) {
					t.Fatalf("Read(%q, document reporting %q) error = %v, want ErrOutcomeDisagreement",
						invoked, claimed, err)
				}
				want := fmt.Sprintf("stdout reports command %q and the invocation was %q", claimed, invoked)
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Read(%q, document reporting %q) refused with %q, which is not the command-agreement guard",
						invoked, claimed, err)
				}
				refused++
			}
		}
		// The counts are asserted against the enumeration so a vocabulary that
		// shrank - or a helper that quietly built fewer documents - cannot turn
		// this into a smaller sweep that still passes.
		if wantRefused := len(implemented) * (len(implemented) - 1); refused != wantRefused {
			t.Fatalf("drove %d disagreeing pairs, want %d over the %d implemented tags",
				refused, wantRefused, len(implemented))
		}
		if admitted != len(implemented) {
			t.Fatalf("admitted %d agreeing pairs, want %d", admitted, len(implemented))
		}
		// The two tags the surviving narrowings named must be inside the swept
		// vocabulary, in both roles. Without this, a vocabulary that dropped
		// them would still satisfy every count above.
		for _, anchor := range []Command{CommandTakeover, CommandList, CommandDoctor} {
			if !slices.Contains(implemented, anchor) {
				t.Fatalf("%q is not among the %d implemented tags, so the narrowing that named it was not driven",
					anchor, len(implemented))
			}
		}
	})
	t.Run("exit status outside the Section 15.2 table", func(t *testing.T) {
		for _, status := range []int{1, 42, -1, 255} {
			_, err := Read(CommandMaterialize, InvocationOutput{Stdout: failure.Stdout, ExitStatus: status})
			if !errors.Is(err, ErrUnregisteredExitStatus) {
				t.Fatalf("Read(exit %d) error = %v, want ErrUnregisteredExitStatus", status, err)
			}
		}
	})
	t.Run("conforming invocation is admitted", func(t *testing.T) {
		if _, err := Read(CommandMaterialize, failure); err != nil {
			t.Fatalf("Read(conforming failure) error = %v", err)
		}
		if _, err := Read(CommandList, success); err != nil {
			t.Fatalf("Read(conforming success) error = %v", err)
		}
	})
}

// TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary states, as a
// measured ratio over the whole registered vocabulary rather than as prose, how
// much of the "the document may not claim a command the invocation did not run"
// class this one guard actually owns.
//
// The sweep above proves the guard over the 18 implemented tags in both
// directions. That is not the whole vocabulary: Section 14.2 registers 44 tags,
// and the other 26 select CLI Result 2.0.0, 3.0.0, or 4.0.0, whose bodies this
// repository does not build. This test drives every one of the 44x44 ordered
// pairs and classifies each answer, so the bound is a number with its
// denominator rather than a sentence.
//
// The stated bound it produces: the command-agreement guard in readSuccess is
// the sole enforcement for the pairs where both tags are implemented, and it is
// never reached for a pair involving an unimplemented tag - not because it was
// bypassed, but because an earlier named refusal already settled that pair.
// Nothing in the grid is admitted except the agreeing pairs. A pair involving an
// unimplemented tag can only be driven with a forged document, because this
// repository builds no body for those tags at all, and the forgery is refused by
// the version binding rather than by this guard.
func TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary(t *testing.T) {
	t.Parallel()

	documents := implementedSuccessDocuments(t)
	registered := Commands()
	implemented := ImplementedCommands()

	// The forged documents: a conforming CLI Result 1.0.0 whose command member
	// has been rewritten to a tag whose body this repository never builds.
	// There is no honest way to produce one, which is the bound.
	forged := make(map[Command]InvocationOutput)
	conforming := mustEmittedSuccess(t, CommandList)
	for _, command := range registered {
		if slices.Contains(implemented, command) {
			continue
		}
		forged[command] = InvocationOutput{
			Stdout:     reversionedResult(t, conforming.Stdout, "1.0.0", string(command)),
			ExitStatus: 0,
		}
	}
	if len(forged)+len(implemented) != len(registered) {
		t.Fatalf("built %d forged and %d emitted documents for %d registered tags",
			len(forged), len(implemented), len(registered))
	}

	const guardPhrase = "stdout reports command"
	admitted, byGuard, byUnimplementedInvocation, byVersionBinding := 0, 0, 0, 0
	for _, invoked := range registered {
		for _, claimed := range registered {
			output, emitted := documents[claimed]
			if !emitted {
				output = forged[claimed]
			}
			_, err := Read(invoked, output)
			switch {
			case err == nil:
				if invoked != claimed {
					t.Fatalf("Read(%q, document reporting %q) was admitted", invoked, claimed)
				}
				admitted++
			case errors.Is(err, ErrOutcomeDisagreement):
				if !strings.Contains(err.Error(), guardPhrase) {
					t.Fatalf("Read(%q, %q) refused with %q, an outcome disagreement that is not the command guard",
						invoked, claimed, err)
				}
				byGuard++
			case errors.Is(err, ErrUnimplementedVersion):
				// The invoked tag selects a version this repository does not
				// build, so Read refuses before it ever reads a command tag out
				// of the document.
				if strings.Contains(err.Error(), guardPhrase) {
					t.Fatalf("Read(%q, %q) refused as unimplemented but quoted the command guard: %v",
						invoked, claimed, err)
				}
				byUnimplementedInvocation++
			case errors.Is(err, ErrInvalidResult):
				// The document claims a tag bound to another CLI Result major,
				// which the closed decoder settles against the document's own
				// declared version before the guard runs.
				if strings.Contains(err.Error(), guardPhrase) {
					t.Fatalf("Read(%q, %q) refused as invalid but quoted the command guard: %v",
						invoked, claimed, err)
				}
				byVersionBinding++
			default:
				t.Fatalf("Read(%q, document reporting %q) refused with an unclassified error: %v",
					invoked, claimed, err)
			}
		}
	}

	// The measured ratio. Every figure is derived from the production
	// enumerations, so a tag that becomes implemented moves all four together
	// instead of leaving one of them quietly wrong.
	pairs := len(registered) * len(registered)
	wantAdmitted := len(implemented)
	wantGuard := len(implemented) * (len(implemented) - 1)
	wantUnimplementedInvocation := (len(registered) - len(implemented)) * len(registered)
	wantVersionBinding := len(implemented) * (len(registered) - len(implemented))
	if admitted != wantAdmitted || byGuard != wantGuard ||
		byUnimplementedInvocation != wantUnimplementedInvocation || byVersionBinding != wantVersionBinding {
		t.Fatalf("over %d ordered pairs: admitted %d/%d, guard %d/%d, unimplemented invocation %d/%d, version binding %d/%d",
			pairs, admitted, wantAdmitted, byGuard, wantGuard,
			byUnimplementedInvocation, wantUnimplementedInvocation, byVersionBinding, wantVersionBinding)
	}
	if admitted+byGuard+byUnimplementedInvocation+byVersionBinding != pairs {
		t.Fatalf("classified %d answers over %d ordered pairs",
			admitted+byGuard+byUnimplementedInvocation+byVersionBinding, pairs)
	}
	if byGuard == 0 || byVersionBinding == 0 || byUnimplementedInvocation == 0 {
		t.Fatalf("a refusal class drove nothing: guard %d, version binding %d, unimplemented invocation %d",
			byGuard, byVersionBinding, byUnimplementedInvocation)
	}
}

// TestReadSelectsTheErrorVersionFromTheCommandNotTheDocument drives the two
// bindings Section 14.2 fixes for the CLI: legacy commands select Structured
// Error 1.0.0 and every session.clone.* command selects 1.1.0. A document that
// declares the other version is refused rather than adopted.
func TestReadSelectsTheErrorVersionFromTheCommandNotTheDocument(t *testing.T) {
	t.Parallel()

	legacy, _ := mustEmittedFailure(t, "not_found")
	clone := reversionedFailure(t, legacy.Stdout, "1.1.0")

	if _, err := Read(CommandCloneRun, InvocationOutput{Stdout: clone, ExitStatus: 4}); err != nil {
		t.Fatalf("Read(session.clone.run, error 1.1.0) error = %v", err)
	}
	if _, err := Read(CommandCloneRun, legacy); !errors.Is(err, axerror.ErrVersionMismatch) {
		t.Fatalf("Read(session.clone.run, error 1.0.0) error = %v, want ErrVersionMismatch", err)
	}
	if _, err := Read(CommandStatus, InvocationOutput{Stdout: clone, ExitStatus: 4}); !errors.Is(err, axerror.ErrVersionMismatch) {
		t.Fatalf("Read(status, error 1.1.0) error = %v, want ErrVersionMismatch", err)
	}
}

// TestReadClassifiesAFailureOfACommandWhoseSuccessBodyIsNotBuilt is the
// compatibility fact the two bindings produce. This repository builds no
// session.clone.* body, so its success object cannot be read here - but its
// failure is a Structured Error 1.1.0 object, and a machine client classifies
// it completely. "This build cannot construct that success" and "this failure
// is unreadable" are different facts and stay different.
func TestReadClassifiesAFailureOfACommandWhoseSuccessBodyIsNotBuilt(t *testing.T) {
	t.Parallel()

	legacy, _ := mustEmittedFailure(t, "not_found")
	clone := reversionedFailure(t, legacy.Stdout, "1.1.0")

	reading, err := Read(CommandCloneRun, InvocationOutput{Stdout: clone, ExitStatus: 4})
	if err != nil {
		t.Fatalf("Read(session.clone.run failure) error = %v", err)
	}
	if code, _ := reading.Code(); code != "not_found" {
		t.Fatalf("Code() = %q, want not_found", code)
	}

	success := mustEmittedSuccess(t, CommandList)
	rewritten := reversionedResult(t, success.Stdout, "2.0.0", string(CommandCloneRun))
	_, err = Read(CommandCloneRun, InvocationOutput{Stdout: rewritten, ExitStatus: 0})
	if !errors.Is(err, ErrUnimplementedVersion) {
		t.Fatalf("Read(session.clone.run success) error = %v, want ErrUnimplementedVersion", err)
	}
}

// TestExitStatusAloneIdentifiesNoFailure states the fan-in as the measured
// ratio it is, for both Structured Error versions the CLI binds. A client that
// branches on the exit status is choosing between the codes of one group
// without evidence, and these are the group sizes.
func TestExitStatusAloneIdentifiesNoFailure(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		version    axerror.Version
		statuses   int
		ambiguous  int
		codes      int
		largest    int
		largestFor int
	}{
		{version: axerror.Version100, statuses: 17, ambiguous: 14, codes: 47, largest: 6, largestFor: 6},
		{version: axerror.Version110, statuses: 17, ambiguous: 14, codes: 66, largest: 9, largestFor: 6},
		// 1.2.0 and 1.3.0 are bound by the CLI Result 3 and 4 majors. They were
		// missing here, and the two README rows that quoted them had both
		// drifted - 1.2.0 was published as 12 codes at exit 6 and is 14 at exit
		// 16, 1.3.0 as 15 at exit 6 and is 17 - because nothing measured the
		// largest class for them.
		{version: axerror.Version120, statuses: 17, ambiguous: 15, codes: 94, largest: 14, largestFor: 16},
		{version: axerror.Version130, statuses: 17, ambiguous: 15, codes: 109, largest: 17, largestFor: 6},
	} {
		t.Run(string(test.version), func(t *testing.T) {
			groups, err := axerror.CodesByExitStatus(test.version)
			if err != nil {
				t.Fatalf("CodesByExitStatus(%s): %v", test.version, err)
			}
			ambiguous, codes, largest, largestFor := 0, 0, 0, 0
			for status, group := range groups {
				codes += len(group)
				if len(group) > 1 {
					ambiguous++
				}
				if len(group) > largest {
					largest, largestFor = len(group), status
				}
			}
			if len(groups) != test.statuses || codes != test.codes {
				t.Fatalf("%s groups %d statuses over %d codes, want %d over %d",
					test.version, len(groups), codes, test.statuses, test.codes)
			}
			if ambiguous != test.ambiguous || largest != test.largest || largestFor != test.largestFor {
				t.Fatalf("%s has %d ambiguous statuses, largest %d at exit %d; want %d, %d at exit %d",
					test.version, ambiguous, largest, largestFor,
					test.ambiguous, test.largest, test.largestFor)
			}
			if _, present := groups[15]; !present {
				t.Fatalf("%s registers no code at exit 15", test.version)
			}
			// A tie at the maximum would make "the largest class" prose
			// ambiguous, so the published row is only well defined while there
			// is exactly one such status. Measured, not assumed.
			ties := 0
			for _, group := range groups {
				if len(group) == largest {
					ties++
				}
			}
			if ties != 1 {
				t.Fatalf("%s has %d statuses tied at %d codes; the published largest-class row "+
					"names one status and would be ambiguous", test.version, ties, largest)
			}
		})
	}
}

// TestExitStatusAloneDecidesNoRetry narrows the fan-in above to the decision a
// client actually makes with it. Structured Error 1.1.0 assigns exit 12 to both
// staging_incomplete, which may carry retryable = true, and transaction_unknown,
// which the pinned document calls "a parked ambiguous effect, never success or
// absence" and which may not. A client keying on exit 12 alone would retry the
// parked one.
func TestExitStatusAloneDecidesNoRetry(t *testing.T) {
	t.Parallel()

	groups, err := axerror.CodesByExitStatus(axerror.Version110)
	if err != nil {
		t.Fatalf("CodesByExitStatus: %v", err)
	}
	permitted, forbidden := 0, 0
	for _, code := range groups[12] {
		if _, refused := axerror.RetryabilityRefusal(code, 12); refused {
			forbidden++
			continue
		}
		permitted++
	}
	if permitted == 0 || forbidden == 0 {
		t.Fatalf("exit 12 codes %v split %d permitted / %d forbidden; the class must contain both "+
			"for the exit status to be insufficient", groups[12], permitted, forbidden)
	}

	legacy, _ := mustEmittedFailure(t, "not_found")
	for _, test := range []struct {
		code      axerror.Code
		retryable bool
		admitted  bool
	}{
		{code: "staging_incomplete", retryable: true, admitted: true},
		{code: "transaction_unknown", retryable: true, admitted: false},
		{code: "transaction_unknown", retryable: false, admitted: true},
	} {
		t.Run(fmt.Sprintf("%s/%t", test.code, test.retryable), func(t *testing.T) {
			document := rewriteFailure(t, legacy.Stdout, map[string]any{
				"schema_version": "1.1.0",
				"code":           string(test.code),
				"exit_code":      json.Number("12"),
				"retryable":      test.retryable,
			})
			reading, err := Read(CommandCloneRun, InvocationOutput{Stdout: document, ExitStatus: 12})
			switch {
			case test.admitted && err != nil:
				t.Fatalf("Read(%s retryable=%t) error = %v, want admission", test.code, test.retryable, err)
			case !test.admitted:
				if err == nil {
					t.Fatalf("Read(%s retryable=true) admitted a forged retry claim", test.code)
				}
				return
			}
			if reading.Retryable() != test.retryable {
				t.Fatalf("Retryable() = %t, want %t", reading.Retryable(), test.retryable)
			}
		})
	}
}

// TestMessageTextChangesNoMachineAnswer is the message-independence proof at the
// reading entry point. Every machine answer is recomputed from documents that
// differ in exactly one member, including messages that name a different code
// and a message that is itself JSON.
func TestMessageTextChangesNoMachineAnswer(t *testing.T) {
	t.Parallel()

	original, _ := mustEmittedFailure(t, "workspace_conflict")
	baseline, err := Read(CommandMaterialize, original)
	if err != nil {
		t.Fatalf("Read(baseline): %v", err)
	}
	for _, message := range []string{
		"x",
		"not_found",
		`{"code":"not_found","exit_code":4,"retryable":true}`,
		"РАБОЧАЯ ОБЛАСТЬ ИЗМЕНИЛАСЬ",
		strings.Repeat("m", 4096),
	} {
		t.Run(fmt.Sprintf("%.16q", message), func(t *testing.T) {
			document := rewriteFailure(t, original.Stdout, map[string]any{"message": message})
			if string(document) == string(original.Stdout) {
				t.Fatal("the rewritten document is byte-identical, so this row proves nothing")
			}
			reading, err := Read(CommandMaterialize, InvocationOutput{Stdout: document, ExitStatus: original.ExitStatus})
			if err != nil {
				t.Fatalf("Read(rewritten message): %v", err)
			}
			assertSameMachineFacts(t, baseline, reading)
			if reading.HumanMessage() != message {
				t.Fatalf("HumanMessage() = %q, want the rewritten text", reading.HumanMessage())
			}
		})
	}
}

// assertSameMachineFacts compares every machine-actionable answer of two
// readings. The human message is deliberately excluded: it is the member the
// caller changed.
func assertSameMachineFacts(t *testing.T, want, got *Reading) {
	t.Helper()
	if want.Succeeded() != got.Succeeded() {
		t.Fatalf("Succeeded() = %t, want %t", got.Succeeded(), want.Succeeded())
	}
	if want.ExitStatus() != got.ExitStatus() {
		t.Fatalf("ExitStatus() = %d, want %d", got.ExitStatus(), want.ExitStatus())
	}
	wantCode, wantPresent := want.Code()
	gotCode, gotPresent := got.Code()
	if wantCode != gotCode || wantPresent != gotPresent {
		t.Fatalf("Code() = %q/%t, want %q/%t", gotCode, gotPresent, wantCode, wantPresent)
	}
	if want.CodeRegistered() != got.CodeRegistered() {
		t.Fatalf("CodeRegistered() = %t, want %t", got.CodeRegistered(), want.CodeRegistered())
	}
	if want.Retryable() != got.Retryable() {
		t.Fatalf("Retryable() = %t, want %t", got.Retryable(), want.Retryable())
	}
	if want.Command() != got.Command() {
		t.Fatalf("Command() = %q, want %q", got.Command(), want.Command())
	}
	if (want.Failure() == nil) != (got.Failure() == nil) || (want.Result() == nil) != (got.Result() == nil) {
		t.Fatal("one reading carries a failure or result the other does not")
	}
	if want.Failure() != nil {
		if !reflect.DeepEqual(want.Failure().DetailKeys(), got.Failure().DetailKeys()) {
			t.Fatalf("detail keys = %v, want %v", got.Failure().DetailKeys(), want.Failure().DetailKeys())
		}
	}
}

// TestReadRefusesAnUnregisteredCommandTag keeps the two absences apart at the
// reading entry point too: a tag AX registers for no version is not the same
// fact as a tag this repository does not build.
func TestReadRefusesAnUnregisteredCommandTag(t *testing.T) {
	t.Parallel()

	failure, _ := mustEmittedFailure(t, "not_found")
	if _, err := Read("resolve", failure); !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("Read(unregistered tag) error = %v, want ErrUnknownCommand", err)
	}
	if _, err := Read(CommandSessionsList, failure); !errors.Is(err, axerror.ErrVersionMismatch) {
		t.Fatalf("Read(sessions.list, error 1.0.0) error = %v, want the CLI Result 3 binding refusal", err)
	}
}

// rewriteFailure replaces named members of a Structured Error document and
// re-encodes it. It is how a negative row moves exactly one member while
// leaving every other byte the emitter produced.
func rewriteFailure(t *testing.T, document []byte, members map[string]any) []byte {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(document, &decoded); err != nil {
		t.Fatalf("decode failure document: %v", err)
	}
	for name, value := range members {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encode member %q: %v", name, err)
		}
		decoded[name] = encoded
	}
	rewritten, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("encode rewritten document: %v", err)
	}
	return rewritten
}

func reversionedFailure(t *testing.T, document []byte, version string) []byte {
	t.Helper()
	return rewriteFailure(t, document, map[string]any{"schema_version": version})
}

func reversionedResult(t *testing.T, document []byte, version, command string) []byte {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(document, &decoded); err != nil {
		t.Fatalf("decode result document: %v", err)
	}
	for name, value := range map[string]string{"schema_version": version, "command": command} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encode member %q: %v", name, err)
		}
		decoded[name] = encoded
	}
	rewritten, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("encode rewritten document: %v", err)
	}
	return rewritten
}

// TestSuccessReadingCarriesNoFailureAnswers pins what a success reading reports
// for the members that only a failure has. A success has no code, no retry bit,
// and no human message, and each is reported as absent rather than as a zero
// value that could be mistaken for an answer.
func TestSuccessReadingCarriesNoFailureAnswers(t *testing.T) {
	t.Parallel()

	reading, err := Read(CommandList, mustEmittedSuccess(t, CommandList))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reading.Succeeded() || reading.Failure() != nil || reading.Result() == nil {
		t.Fatal("a success reading did not report a success")
	}
	if code, present := reading.Code(); present || code != "" {
		t.Fatalf("Code() = %q/%t, want \"\"/false", code, present)
	}
	if reading.CodeRegistered() || reading.Retryable() {
		t.Fatal("a success reading answered a failure-only question")
	}
	if reading.HumanMessage() != "" {
		t.Fatalf("HumanMessage() = %q, want empty", reading.HumanMessage())
	}
	if reading.ExitStatus() != SuccessExitStatus {
		t.Fatalf("ExitStatus() = %d, want 0", reading.ExitStatus())
	}
}

// TestAnAbsentSuccessObjectIsRefusedAtExitZero is the exit-0 arm of the absence
// rule. A command that exits 0 without writing its success object has not
// reported success in the way Section 14.2 requires, and the reading says so
// instead of inventing one.
func TestAnAbsentSuccessObjectIsRefusedAtExitZero(t *testing.T) {
	t.Parallel()

	_, err := Read(CommandList, InvocationOutput{Stdout: nil, ExitStatus: 0})
	if !errors.Is(err, ErrAbsentDocument) {
		t.Fatalf("Read(exit 0, no document) error = %v, want ErrAbsentDocument", err)
	}
	if !strings.Contains(err.Error(), "requires the success object itself on stdout") {
		t.Fatalf("refusal %q does not say why exit 0 alone is not a success report", err)
	}
}

// TestEveryRegisteredCommandMajorBindsAnErrorVersion is why Read has no
// fallback for an unbound major: there is none to reach. Every command tag the
// pinned registry carries selects a CLI Result version whose major the static
// Section 15.1 table binds, so the refusal path in Read is unreachable by
// construction rather than untested.
func TestEveryRegisteredCommandMajorBindsAnErrorVersion(t *testing.T) {
	t.Parallel()

	commands := Commands()
	if len(commands) != 44 {
		t.Fatalf("registry carries %d command tags, want the reviewed 44", len(commands))
	}
	for _, command := range commands {
		version, err := RegisteredVersionForCommand(command)
		if err != nil {
			t.Fatalf("RegisteredVersionForCommand(%q): %v", command, err)
		}
		major, err := majorOf(version)
		if err != nil {
			t.Fatalf("majorOf(%q): %v", version, err)
		}
		if _, err := axerror.BindingFor(axerror.ContainingContract{ID: Schema, Major: major}); err != nil {
			t.Fatalf("CLI Result major %d binds no Structured Error version: %v", major, err)
		}
	}
}

// TestAMalformedSuccessBodyIsRefusedRatherThanReported drives the reading path
// through the closed body validator. A document that is a CLI Result envelope
// and carries a body the tagged validator refuses is not a success a machine
// client may act on, and exit 0 does not make it one.
func TestAMalformedSuccessBodyIsRefusedRatherThanReported(t *testing.T) {
	t.Parallel()

	success := mustEmittedSuccess(t, CommandList)
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(success.Stdout, &decoded); err != nil {
		t.Fatalf("decode success document: %v", err)
	}
	decoded["body"] = json.RawMessage(`{"sessions":[],"partial":"no","unreachable_peer_ids":[]}`)
	corrupt, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("encode corrupt document: %v", err)
	}
	if _, err := Read(CommandList, InvocationOutput{Stdout: corrupt, ExitStatus: 0}); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("Read(corrupt body) error = %v, want ErrInvalidResult", err)
	}
}

// TestReadingAccessorsDoNotHandOutLiveInteriorState probes the guarantee this
// package inherits rather than trusting it. A validated object whose accessor
// returns live interior state is not validated: every declared bound can be
// violated after construction through the container the caller was handed. Both
// inherited packages were checked this way before their guarantees were used in
// the compatibility claims above, and a reading is checked here because it
// hands out both of them.
func TestReadingAccessorsDoNotHandOutLiveInteriorState(t *testing.T) {
	t.Parallel()

	success := readHistorical(t, historicalEntry(t, "cli-result-1.0.0-doctor-extensions.json"))
	reading, err := Read(CommandDoctor, InvocationOutput{Stdout: success, ExitStatus: 0})
	if err != nil {
		t.Fatalf("Read(doctor): %v", err)
	}
	body := reading.Result().Body()
	findings, ok := body["findings"].([]any)
	if !ok || len(findings) == 0 {
		t.Fatalf("body findings = %v, want a non-empty array to mutate", body["findings"])
	}
	findings[0].(map[string]any)["severity"] = "error"
	body["healthy"] = true
	extension, known := reading.Result().Extension("works.relux.ax.future-reader-hint")
	if !known {
		t.Fatal("the fixture extension is absent, so this probe would prove nothing")
	}
	extension.(map[string]any)["kind"] = "authoritative"

	fresh := reading.Result().Body()
	if fresh["healthy"] != false {
		t.Fatalf("a mutation through Body() reached the result: healthy = %v", fresh["healthy"])
	}
	if severity := fresh["findings"].([]any)[0].(map[string]any)["severity"]; severity != "warning" {
		t.Fatalf("a mutation two containers deep reached the result: severity = %v", severity)
	}
	again, _ := reading.Result().Extension("works.relux.ax.future-reader-hint")
	if again.(map[string]any)["kind"] != "inert" {
		t.Fatalf("a mutation through Extension() reached the result: %v", again)
	}

	failure := readHistorical(t, historicalEntry(t, "error-1.0.0-partial-sync-unknown-details.json"))
	failureReading, err := Read(CommandSync, InvocationOutput{Stdout: failure, ExitStatus: 15})
	if err != nil {
		t.Fatalf("Read(sync failure): %v", err)
	}
	hint, present := failureReading.Failure().Detail("future_reader_hint")
	if !present {
		t.Fatal("the fixture detail is absent, so this probe would prove nothing")
	}
	hint.(map[string]any)["kind"] = "authoritative"
	fresher, _ := failureReading.Failure().Detail("future_reader_hint")
	if fresher.(map[string]any)["kind"] != "inert" {
		t.Fatalf("a mutation through Detail() reached the failure: %v", fresher)
	}
}

// TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice makes the count in the
// refusal a measurement instead of prose beside one.
//
// exitStatusIsNotEnough documents itself as stating "a measured count rather
// than as advice", and README.md repeats that claim, but nothing read the
// number: every assertion on these refusals matched the trailing clause
// "identifies no failure". Replacing len(fanIn[exitStatus]) with the constant 1,
// or the whole "%d registered" clause with the word "many", left the suite
// green. Both mutants are killed here.
//
// The sweep is the point. A single row would leave the count proven at one
// point of its domain, which is precisely how the two round-2 findings on this
// leaf survived, so every registered command is driven at every registered
// Section 15.2 failure status and each answer is re-derived from
// axerror.CodesByExitStatus for the version that command binds.
func TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice(t *testing.T) {
	t.Parallel()

	statuses := registeredFailureExitStatuses(t)
	commands := Commands()
	if len(commands) == 0 {
		t.Fatal("no registered commands, so this sweep drives nothing")
	}
	measured := 0
	for _, command := range commands {
		resultVersion, err := RegisteredVersionForCommand(command)
		if err != nil {
			t.Fatalf("RegisteredVersionForCommand(%s): %v", command, err)
		}
		major, err := majorOf(resultVersion)
		if err != nil {
			t.Fatalf("majorOf(%s): %v", resultVersion, err)
		}
		errorVersion, err := axerror.BindingFor(axerror.ContainingContract{ID: Schema, Major: major})
		if err != nil {
			t.Fatalf("BindingFor(%s major %d): %v", Schema, major, err)
		}
		fanIn, err := axerror.CodesByExitStatus(errorVersion)
		if err != nil {
			t.Fatalf("CodesByExitStatus(%s): %v", errorVersion, err)
		}
		for _, status := range statuses {
			// The absent-document arm reaches exitStatusIsNotEnough from the
			// production entry point, which is where a machine client meets it.
			_, err := Read(command, InvocationOutput{ExitStatus: status})
			if !errors.Is(err, ErrAbsentDocument) {
				t.Fatalf("Read(%s, nothing on stdout at exit %d) error = %v, want ErrAbsentDocument",
					command, status, err)
			}
			var statedStatus, statedCount int
			var statedVersion string
			read, scanErr := fmt.Sscanf(
				refusalTail(t, err.Error()),
				"exit status %d is assigned to %d registered Structured Error %s codes,",
				&statedStatus, &statedCount, &statedVersion)
			if scanErr != nil || read != 3 {
				t.Fatalf("refusal for %s at exit %d does not state a measured fan-in: %q (%v)",
					command, status, err, scanErr)
			}
			if statedStatus != status {
				t.Fatalf("refusal for %s at exit %d names exit status %d", command, status, statedStatus)
			}
			if statedVersion != string(errorVersion) {
				t.Fatalf("refusal for %s at exit %d names Structured Error %s, want %s",
					command, status, statedVersion, errorVersion)
			}
			if want := len(fanIn[status]); statedCount != want {
				t.Fatalf("refusal for %s at exit %d states %d registered codes; CodesByExitStatus(%s) measures %d",
					command, status, statedCount, errorVersion, want)
			}
			if statedCount == 0 {
				t.Fatalf("refusal for %s at exit %d states no registered codes, so the sweep proves nothing there",
					command, status)
			}
			measured++
		}
	}
	// Asserted rather than left implicit: a predicate or registry that answered
	// with nothing would otherwise make this a green test that read no refusal.
	if want := len(commands) * len(statuses); measured != want {
		t.Fatalf("measured %d refusals, want %d", measured, want)
	}
}

// refusalTail returns the part of a refusal after the last "; " separator, which
// is where exitStatusIsNotEnough is appended. The absent-document arm formats it
// after a ": " instead, so both separators are tried and an unparseable refusal
// is reported rather than silently scanned from its beginning.
func refusalTail(t *testing.T, refusal string) string {
	t.Helper()
	if index := strings.LastIndex(refusal, "; "); index >= 0 {
		return refusal[index+2:]
	}
	if index := strings.LastIndex(refusal, ": "); index >= 0 {
		return refusal[index+2:]
	}
	t.Fatalf("refusal %q carries no appended clause", refusal)
	return ""
}

// TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus generalizes the
// retry proof above from one exit status to the whole Section 15.2 failure
// domain.
//
// TestExitStatusAloneDecidesNoRetry drives exit 12 only, and the one exit-15
// envelope any test reads already declares retryable = true, so a fabricated
// "if the status is 15, retry" branch returned the answer the document would
// have given and survived the suite. The fix is not another single row: it is to
// leave no registered failure status unexercised on BOTH polarities, so a
// special case keyed on any status is a red at that status.
func TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus(t *testing.T) {
	t.Parallel()

	groups, err := axerror.CodesByExitStatus(axerror.Version110)
	if err != nil {
		t.Fatalf("CodesByExitStatus: %v", err)
	}
	legacy, _ := mustEmittedFailure(t, "not_found")
	statuses := registeredFailureExitStatuses(t)
	measuredFalse, measuredTrue := 0, 0
	for _, status := range statuses {
		codes := groups[status]
		if len(codes) == 0 {
			t.Fatalf("Structured Error 1.1.0 registers no code at exit %d, so this status is unexercised", status)
		}
		// A retryable = false claim is admissible for every code, so the false
		// arm covers every status without exception. Any branch that answers
		// true from the status alone dies here at that status.
		t.Run(fmt.Sprintf("exit%d/false", status), func(t *testing.T) {
			document := rewriteFailure(t, legacy.Stdout, map[string]any{
				"schema_version": "1.1.0",
				"code":           string(codes[0]),
				"exit_code":      json.Number(fmt.Sprint(status)),
				"retryable":      false,
			})
			reading, readErr := Read(CommandCloneRun, InvocationOutput{Stdout: document, ExitStatus: status})
			if readErr != nil {
				t.Fatalf("Read(%s retryable=false at exit %d): %v", codes[0], status, readErr)
			}
			if reading.Retryable() {
				t.Fatalf("Retryable() = true at exit %d for a document declaring false", status)
			}
		})
		measuredFalse++

		// The true arm is only defined where the pinned document permits the
		// claim, so it is driven for the first permitted code of the status and
		// the count of covered statuses is asserted below rather than assumed.
		var permitted axerror.Code
		for _, code := range codes {
			if _, refused := axerror.RetryabilityRefusal(code, status); !refused {
				permitted = code
				break
			}
		}
		if permitted == "" {
			continue
		}
		t.Run(fmt.Sprintf("exit%d/true", status), func(t *testing.T) {
			document := rewriteFailure(t, legacy.Stdout, map[string]any{
				"schema_version": "1.1.0",
				"code":           string(permitted),
				"exit_code":      json.Number(fmt.Sprint(status)),
				"retryable":      true,
			})
			reading, readErr := Read(CommandCloneRun, InvocationOutput{Stdout: document, ExitStatus: status})
			if readErr != nil {
				t.Fatalf("Read(%s retryable=true at exit %d): %v", permitted, status, readErr)
			}
			if !reading.Retryable() {
				t.Fatalf("Retryable() = false at exit %d for a document declaring true", status)
			}
		})
		measuredTrue++
	}
	// Measured, not assumed. A registry that answered with empty groups, or a
	// refusal predicate that forbade every claim, would otherwise leave one or
	// both arms driving nothing while the test stayed green.
	if measuredFalse != len(statuses) {
		t.Fatalf("the retryable=false arm covered %d of %d failure statuses", measuredFalse, len(statuses))
	}
	if measuredTrue != retryPermittedFailureStatuses {
		t.Fatalf("the retryable=true arm covered %d failure statuses, want %d",
			measuredTrue, retryPermittedFailureStatuses)
	}
}

// retryPermittedFailureStatuses is the measured number of Section 15.2 failure
// statuses that carry at least one Structured Error 1.1.0 code permitted to
// claim retryable = true. Fourteen of the seventeen do; at the other three the
// pinned document forbids the claim for every code of the class, which is why
// the true arm is asserted against this figure rather than against the status
// count. It is pinned so that a shrinking true arm is a red rather than a
// quietly smaller sweep.
const retryPermittedFailureStatuses = 14
