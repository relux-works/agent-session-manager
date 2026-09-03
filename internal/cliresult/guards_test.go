package cliresult

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// registeredExitStatusesFromTheMeaningTable enumerates every status the Section
// 15.2 table assigns a meaning, success included.
//
// The oracle is ExitStatusMeaning rather than IsFailureExitStatus, which is the
// predicate the gate under test calls. An oracle built from the predicate under
// test moves with a mutant that narrows it, and the sweep would then measure
// nothing. The asserted count catches a mutation that moves both.
func registeredExitStatusesFromTheMeaningTable(t *testing.T) []int {
	t.Helper()

	var statuses []int
	for status := 0; status <= 255; status++ {
		if _, known := axerror.ExitStatusMeaning(status); known {
			statuses = append(statuses, status)
		}
	}
	const registeredStatusCount = 18
	if len(statuses) != registeredStatusCount {
		t.Fatalf("the Section 15.2 table carries %d statuses %v, want the pinned %d "+
			"(success plus the 17 failure rows)", len(statuses), statuses, registeredStatusCount)
	}
	for _, required := range []int{SuccessExitStatus, interruptedExitStatus} {
		if !slices.Contains(statuses, required) {
			t.Fatalf("exit %d is absent from the enumeration, so no sweep drawing on it covers that row", required)
		}
	}
	return statuses
}

// TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain turns the
// ErrUnregisteredExitStatus gate in Read into a measured ratio over its whole
// input domain.
//
// The gate was driven at four sampled values - {1, 42, -1, 255} - so restricting
// it to exactly those four passed all thirteen packages, as did sparing 127 and
// 137, the two statuses a wrapper most plausibly returns when ax never ran at
// all. Both were reproduced as SURVIVED on the reviewed tree with a delete-only
// control KILLED in the same run. The complement of a four-value sample is not a
// bound.
//
// It is not an admission bypass and is not claimed as one: a failure document is
// still refused by readFailure's exit_code equality, a success document by
// readSuccess's status guard, and axerror.decodeExitStatus still refuses an
// unregistered exit_code inside the document. What the gate decides is WHICH
// FACT the machine client is told, which is the deliverable this leaf exists to
// establish. Under the {127, 137} narrowing an invocation that wrote nothing at
// exit 127 answers ErrAbsentDocument and states "exit status 127 is assigned to
// 0 registered Structured Error 1.0.0 codes" - a fabricated fan-in for a status
// the Section 15.2 table does not carry, reported in the sentence README
// publishes as a measured count.
//
// This is the reader-side twin of axerror.decodeExitStatus, swept over the same
// domain by TestTheExitStatusAdmissionIsSweptOverEveryByteValue.
func TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain(t *testing.T) {
	t.Parallel()

	registered := make(map[int]bool)
	for _, status := range registeredExitStatusesFromTheMeaningTable(t) {
		registered[status] = true
	}

	// Empty stdout isolates this gate and nothing else: past it the reader
	// reports an absent document, and the two answers cannot be confused.
	admitted, refused := 0, 0
	for status := 0; status <= 255; status++ {
		_, err := Read(CommandMaterialize, InvocationOutput{ExitStatus: status})
		if registered[status] {
			if errors.Is(err, ErrUnregisteredExitStatus) {
				t.Fatalf("Read(exit %d) was refused as an unregistered status, and the Section 15.2 "+
					"table assigns that status a meaning: %v", status, err)
			}
			if !errors.Is(err, ErrAbsentDocument) {
				t.Fatalf("Read(exit %d, nothing on stdout) error = %v, want ErrAbsentDocument", status, err)
			}
			admitted++
			continue
		}
		if !errors.Is(err, ErrUnregisteredExitStatus) {
			t.Fatalf("Read(exit %d) error = %v, want ErrUnregisteredExitStatus: the Section 15.2 table "+
				"assigns that status no meaning", status, err)
		}
		if want := fmt.Sprintf(": %d", status); !strings.HasSuffix(err.Error(), want) {
			t.Fatalf("Read(exit %d) refused without naming the status it refused: %v", status, err)
		}
		// The refusal must not carry the measured-fan-in sentence, whose count
		// for a status outside the table is a map miss rather than a
		// measurement. See TestTheFanInSentenceCannotReportAMapMissAsAMeasurement.
		if strings.Contains(err.Error(), "is assigned to") {
			t.Fatalf("Read(exit %d) refused with a fan-in measurement for a status the table does "+
				"not carry: %v", status, err)
		}
		refused++
	}
	if admitted != len(registered) {
		t.Fatalf("over 0..255 Read admitted %d exit statuses past the gate, and the Section 15.2 "+
			"table registers %d", admitted, len(registered))
	}
	if want := 256 - len(registered); refused != want {
		t.Fatalf("over 0..255 Read refused %d exit statuses, want %d", refused, want)
	}

	// The gate's two production predicates are cross-checked against the
	// meaning-table oracle here, so a narrowing of either one is a red rather
	// than a quietly different admitted set.
	for status := 0; status <= 255; status++ {
		byPredicates := status == SuccessExitStatus || axerror.IsFailureExitStatus(status)
		if byPredicates != registered[status] {
			t.Fatalf("exit %d: the gate's predicates admit=%t and the Section 15.2 meaning table "+
				"registers=%t", status, byPredicates, registered[status])
		}
	}

	// The domain is int, not a byte. No process exits outside 0..255, but
	// InvocationOutput.ExitStatus is an int a caller fills in, and a wrapper,
	// an RPC hop, or a test harness can put anything there.
	outOfRange := 0
	for _, status := range []int{-1, -130, -2147483648, 256, 1000, 65536, math.MaxInt32, math.MaxInt64} {
		_, err := Read(CommandMaterialize, InvocationOutput{ExitStatus: status})
		if !errors.Is(err, ErrUnregisteredExitStatus) {
			t.Fatalf("Read(exit %d) error = %v, want ErrUnregisteredExitStatus", status, err)
		}
		outOfRange++
	}
	if outOfRange != 8 {
		t.Fatalf("drove %d out-of-range statuses, want 8", outOfRange)
	}
}

// TestTheReadLevelExitStatusAdmissionAdmitsARealReadingAtEveryRegisteredStatus
// is the positive half of the sweep above. Reporting a different refusal past
// the gate shows the gate did not fire; it does not show that a conforming
// invocation at that status is classified.
//
// Every registered failure status is driven with a Structured Error declaring
// exactly that exit_code, and the success status with a CLI Result. The document
// carries a code the pinned registry does not register, which takes Section
// 15.3's unknown-code branch in axerror.decodeBody and skips the code-to-exit
// agreement, so the exit status is the only thing the row moves.
func TestTheReadLevelExitStatusAdmissionAdmitsARealReadingAtEveryRegisteredStatus(t *testing.T) {
	t.Parallel()

	entry := historicalEntry(t, "error-1.0.0-later-minor-code.json")
	frozen := readHistorical(t, entry)

	classified := 0
	for _, status := range registeredFailureExitStatuses(t) {
		document := rewriteFailure(t, frozen, map[string]any{"exit_code": status})
		reading, err := Read(entry.command, InvocationOutput{Stdout: document, ExitStatus: status})
		if err != nil {
			t.Fatalf("Read(failure declaring exit_code %d at exit %d) error = %v", status, status, err)
		}
		if reading.Succeeded() {
			t.Fatalf("Read(Structured Error at exit %d) reported success", status)
		}
		if reading.ExitStatus() != status {
			t.Fatalf("Read(exit %d).ExitStatus() = %d", status, reading.ExitStatus())
		}
		if reading.CodeRegistered() {
			t.Fatalf("the fixture's code is registered at exit %d, so this row measures the "+
				"code-to-exit agreement rather than the admission", status)
		}
		classified++
	}
	if want := len(registeredFailureExitStatuses(t)); classified != want {
		t.Fatalf("classified %d failure statuses, want %d", classified, want)
	}

	success := mustEmittedSuccess(t, CommandList)
	reading, err := Read(CommandList, success)
	if err != nil {
		t.Fatalf("Read(conforming success at exit %d) error = %v", SuccessExitStatus, err)
	}
	if !reading.Succeeded() || reading.ExitStatus() != SuccessExitStatus {
		t.Fatalf("Read(conforming success) = %t/%d", reading.Succeeded(), reading.ExitStatus())
	}
}

// TestTheForeignSchemaRefusalIsMeasuredOverTheRegisteredContractVocabulary
// states, as a measured ratio over the whole pinned contract registry rather
// than as one sample, how much of "stdout carries neither a CLI Result nor a
// Structured Error" the default branch of Read actually owns.
//
// Its whole coverage was one row of TestReadDistinguishesAbsenceFromAReadFailure
// driving urn:ax:schema:session-record. Restricting the branch to exactly that
// identifier - routing every other foreign schema to readSuccess instead -
// passed all thirteen packages. The document is still refused there, by
// verifyEnvelopeIdentity, so this is the same class as the exit-status finding:
// which fact the machine client is told, not whether it is admitted.
//
// STATED BOUND: the domain of the schema member is every JSON string, which is
// unbounded. What is measured is every contract identifier the pinned catalog
// registers, plus the near-miss neighbours of the two admitted identifiers -
// case variants, a trailing space, a prefix, and a suffix - because a
// discriminator that fails does so on a neighbour of what it admits, not on an
// arbitrary string.
func TestTheForeignSchemaRefusalIsMeasuredOverTheRegisteredContractVocabulary(t *testing.T) {
	t.Parallel()

	admittedSchemas := []string{Schema, axerror.Schema}
	var foreign []string
	for _, contract := range catalog.Current().Contracts {
		identifier := string(contract.ID)
		if !slices.Contains(admittedSchemas, identifier) {
			foreign = append(foreign, identifier)
		}
	}
	// The denominator is asserted against the catalog rather than merely
	// required to be non-empty. Without this, narrowing the loop above to the
	// first two contracts left the near-miss neighbours below carrying the whole
	// sweep, and the catalog half of the measurement vanished in silence.
	catalogued := len(catalog.Current().Contracts)
	admittedInCatalog := 0
	for _, contract := range catalog.Current().Contracts {
		if slices.Contains(admittedSchemas, string(contract.ID)) {
			admittedInCatalog++
		}
	}
	if admittedInCatalog != len(admittedSchemas) {
		t.Fatalf("the catalog registers %d of the %d admitted schema identifiers; the subtraction "+
			"below has the wrong denominator", admittedInCatalog, len(admittedSchemas))
	}
	if want := catalogued - len(admittedSchemas); len(foreign) != want {
		t.Fatalf("drew %d foreign identifiers from a catalog of %d contracts, want %d",
			len(foreign), catalogued, want)
	}
	if len(foreign) < 40 {
		t.Fatalf("the catalog offers only %d foreign contract identifiers, which is below the "+
			"vocabulary this sweep was published against", len(foreign))
	}
	fromCatalog := len(foreign)

	for _, admitted := range admittedSchemas {
		neighbours := []string{
			admitted + " ",
			" " + admitted,
			admitted + "s",
			strings.ToUpper(admitted),
			strings.Replace(admitted, "urn:ax:", "urn:AX:", 1),
			admitted[:len(admitted)-1],
		}
		for _, neighbour := range neighbours {
			if slices.Contains(admittedSchemas, neighbour) {
				t.Fatalf("the near-miss %q equals an admitted identifier, so this row would "+
					"require the reader to refuse a document it must accept", neighbour)
			}
		}
		foreign = append(foreign, neighbours...)
	}
	foreign = append(foreign, "")

	refused := 0
	for _, identifier := range foreign {
		document, err := json.Marshal(map[string]any{"schema": identifier})
		if err != nil {
			t.Fatalf("encode probe document for %q: %v", identifier, err)
		}
		_, err = Read(CommandMaterialize, InvocationOutput{Stdout: document, ExitStatus: 5})
		if !errors.Is(err, ErrForeignDocument) {
			t.Fatalf("Read(document declaring schema %q) error = %v, want ErrForeignDocument", identifier, err)
		}
		// The refusal must be this guard's own sentence naming the schema it
		// refused, so a document settled by some earlier check is not counted
		// as coverage of the discriminator's default branch.
		if want := fmt.Sprintf("schema %q", identifier); !strings.Contains(err.Error(), want) {
			t.Fatalf("Read(schema %q) refused with %q, which is not the foreign-schema branch", identifier, err)
		}
		refused++
	}
	if refused != len(foreign) {
		t.Fatalf("refused %d of %d foreign identifiers", refused, len(foreign))
	}
	if want := fromCatalog + len(admittedSchemas)*6 + 1; len(foreign) != want {
		t.Fatalf("swept %d identifiers, want %d catalogued plus %d near-miss neighbours plus the "+
			"empty string", len(foreign), fromCatalog, len(admittedSchemas)*6)
	}
	// The positive control, in the same test: both admitted identifiers reach
	// their branch. Without it a default branch that refused everything would
	// pass the sweep above.
	if _, err := Read(CommandList, mustEmittedSuccess(t, CommandList)); err != nil {
		t.Fatalf("Read(conforming CLI Result) error = %v", err)
	}
	failure, _ := mustEmittedFailure(t, "workspace_conflict")
	if _, err := Read(CommandMaterialize, failure); err != nil {
		t.Fatalf("Read(conforming Structured Error) error = %v", err)
	}
}

// TestTheSchemaMemberTypeGuardIsMeasuredOverEveryJSONValueForm closes the same
// class on the neighbouring guard: "the schema member is there and is not a
// string".
//
// Its whole coverage was one row driving the number 5, so narrowing the guard to
// refuse only JSON numbers passed all thirteen packages, and a document whose
// schema member was true, null, an array, or an object was routed to
// readSuccess. The domain here is closed and small - the six JSON value forms -
// so it is swept rather than sampled.
func TestTheSchemaMemberTypeGuardIsMeasuredOverEveryJSONValueForm(t *testing.T) {
	t.Parallel()

	nonStrings := map[string]string{
		"number":         `5`,
		"negative":       `-1`,
		"float":          `1.5`,
		"true":           `true`,
		"false":          `false`,
		"null":           `null`,
		"array":          `["urn:ax:schema:cli-result"]`,
		"object":         `{"id":"urn:ax:schema:cli-result"}`,
		"nested array":   `[["urn:ax:schema:error"]]`,
		"array of one":   `[1]`,
		"empty array":    `[]`,
		"empty object":   `{}`,
		"number as text": `"5"`,
	}
	measured := 0
	for name, raw := range nonStrings {
		document := []byte(fmt.Sprintf(`{"schema":%s}`, raw))
		_, err := Read(CommandMaterialize, InvocationOutput{Stdout: document, ExitStatus: 5})
		if name == "number as text" {
			// The control: a JSON string reaches the discriminator and is
			// answered as a foreign contract, not as an unreadable document.
			// Without it a guard that refused every schema member would pass.
			if !errors.Is(err, ErrForeignDocument) {
				t.Fatalf("Read(schema %s) error = %v, want ErrForeignDocument", raw, err)
			}
			measured++
			continue
		}
		if !errors.Is(err, ErrUnreadableDocument) {
			t.Fatalf("Read(schema %s, %s) error = %v, want ErrUnreadableDocument", raw, name, err)
		}
		if !strings.Contains(err.Error(), "schema member is not a string") {
			t.Fatalf("Read(schema %s) refused with %q, which is not the schema-member type guard", raw, err)
		}
		measured++
	}
	if measured != len(nonStrings) {
		t.Fatalf("measured %d of %d schema member forms", measured, len(nonStrings))
	}
	// Asserted rather than left implicit: the six JSON value forms other than
	// string, plus the string control, must all be present, so a shrunken table
	// is a red instead of a quietly smaller sweep.
	for _, required := range []string{"number", "float", "true", "false", "null", "array", "object", "number as text"} {
		if _, present := nonStrings[required]; !present {
			t.Fatalf("the %q form is absent from the swept table", required)
		}
	}
}

// TestTheUnsupportedMajorGuardIsSweptOverAMeasuredMajorRange answers reviewer
// observation O3 by measurement rather than by prose.
//
// verifyEnvelopeIdentity's major comparison was proven at major 2 only, so
// `candidateMajor != expectedMajor && candidateMajor != 1 && candidateMajor != 3`
// survived. The domain is the unbounded set of semantic-version majors.
//
// STATED BOUND: majors 0..64 are swept; the tail above 64 is not, because the
// comparison is a single integer inequality with no table behind it and no
// value in that tail is reachable through any registered contract. A per-major
// special case anywhere inside the swept range is a red.
func TestTheUnsupportedMajorGuardIsSweptOverAMeasuredMajorRange(t *testing.T) {
	t.Parallel()

	conforming := mustEmittedSuccess(t, CommandList)
	bound, err := VersionForCommand(CommandList)
	if err != nil {
		t.Fatalf("VersionForCommand(%s): %v", CommandList, err)
	}
	boundMajor, err := majorOf(bound)
	if err != nil {
		t.Fatalf("majorOf(%s): %v", bound, err)
	}

	refused, admitted := 0, 0
	for major := 0; major <= 64; major++ {
		document := reversionedResult(t, conforming.Stdout, fmt.Sprintf("%d.0.0", major), string(CommandList))
		_, err := Read(CommandList, InvocationOutput{Stdout: document, ExitStatus: SuccessExitStatus})
		if major == boundMajor {
			if err != nil {
				t.Fatalf("Read(CLI Result major %d, the bound major) error = %v", major, err)
			}
			admitted++
			continue
		}
		if !errors.Is(err, ErrUnsupportedMajor) {
			t.Fatalf("Read(CLI Result major %d) error = %v, want ErrUnsupportedMajor", major, err)
		}
		if want := fmt.Sprintf("document is %d.0.0", major); !strings.Contains(err.Error(), want) {
			t.Fatalf("Read(major %d) refused with %q, which does not name the major it refused", major, err)
		}
		refused++
	}
	if admitted != 1 {
		t.Fatalf("admitted %d majors, want exactly the bound major %d", admitted, boundMajor)
	}
	if refused != 64 {
		t.Fatalf("refused %d majors over 0..64, want 64", refused)
	}
}

// TestTheCommonDataModelGateDoesNotCoverTheSection16NumberRule pins a DISCLOSED
// BOUND rather than a guarantee, and exists so the disclosure cannot rot.
//
// documentSchema's doc comment and README both say the document is shown to be
// "inside the Section 1.6 common logical data model" before its schema member is
// read. That is true of the part canonicaljson.Canonicalize enforces - duplicate
// members, encoding, ordering - and it is NOT true of the number half of Section
// 1.6, which SPEC.md states as "integers only, inside the IEEE 754 double
// safe-integer range". Read admits a Structured Error whose details carry 1.5.
//
// This test asserts the admission, so the bound is measured rather than
// asserted in prose. Closing the gap - Section 1.6's number rule is bound to
// internal/scalar and is unevidenced there - turns this test red, which is the
// intended signal to update the disclosure alongside it.
func TestTheCommonDataModelGateDoesNotCoverTheSection16NumberRule(t *testing.T) {
	t.Parallel()

	document := []byte(`{"schema":"urn:ax:schema:error","schema_version":"1.0.0",` +
		`"code":"workspace_conflict","message":"disclosed bound probe","exit_code":5,` +
		`"retryable":false,"details":{"measured":1.5}}`)
	reading, err := Read(CommandMaterialize, InvocationOutput{Stdout: document, ExitStatus: 5})
	if err != nil {
		t.Fatalf("Read(details carrying 1.5) error = %v; if the Section 1.6 number rule is now "+
			"enforced on this path, update the disclosure in README and in documentSchema's "+
			"doc comment in the same change", err)
	}
	code, present := reading.Code()
	if !present || code != "workspace_conflict" {
		t.Fatalf("Read(details carrying 1.5) = %q/%t, want the admission this bound discloses", code, present)
	}
	// The gate does enforce the half it claims, on the same path, so the bound
	// is a gap in coverage and not a gate that does nothing.
	duplicate := []byte(`{"schema":"urn:ax:schema:error","schema":"urn:ax:schema:error",` +
		`"schema_version":"1.0.0","code":"workspace_conflict","message":"x","exit_code":5,` +
		`"retryable":false,"details":{}}`)
	if _, err := Read(CommandMaterialize, InvocationOutput{Stdout: duplicate, ExitStatus: 5}); !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Read(duplicate schema member) error = %v, want ErrUnreadableDocument", err)
	}
}
