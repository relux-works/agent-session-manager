package cliresult

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// InvocationOutput is everything a machine client observes of one completed
// `ax --json` invocation.
//
// It has exactly two members, and the members it does not have are the point.
// Section 14.2 puts the machine-readable answer in one place - "in JSON mode,
// stdout MUST contain exactly one JSON document; logs remain on stderr" - so a
// client that reads stderr is reading diagnostics, not results. There is no
// Stderr member here, so a caller of Read cannot pass one and this package
// cannot branch on one. That is a structural guarantee rather than a promise:
// adding the member is the only way to break it, and
// TestMachineReadingCannotSeeStderr fails when it appears.
type InvocationOutput struct {
	// Stdout is the exact bytes the invocation wrote to its standard output.
	Stdout []byte
	// ExitStatus is the process exit status. Section 14.2 requires it to equal
	// the failure object's exit_code, so it is corroborating evidence for the
	// document on stdout, never a substitute for it.
	ExitStatus int
}

var (
	// ErrAbsentDocument reports stdout that carries no document at all.
	//
	// It is deliberately distinct from ErrUnreadableDocument. An absence and a
	// failure to read are different facts about an invocation: nothing was
	// written, versus something was written that this reader cannot parse. A
	// client that collapsed them would treat a truncated pipe as a silent
	// command.
	ErrAbsentDocument = errors.New("invocation wrote no document to stdout")

	// ErrUnreadableDocument reports stdout that is not exactly one readable
	// JSON document. A partial, malformed, or multi-document stdout is a read
	// failure and is never resolved from the exit status instead.
	//
	// A document outside the Section 1.6 common logical data model is the same
	// fact, not a lesser one. Bytes whose members repeat are readable in more
	// than one way - encoding/json keeps the last occurrence of a scalar and
	// merges both occurrences of an object - so no single reading of them
	// exists to report.
	ErrUnreadableDocument = errors.New("invocation stdout is not one readable JSON document")

	// ErrForeignDocument reports one readable JSON document whose schema is
	// neither the CLI Result schema nor the Structured Error schema. It is a
	// third fact again: the bytes parsed, and they belong to some other
	// contract.
	ErrForeignDocument = errors.New("invocation stdout carries neither a cli result nor a structured error")

	// ErrOutcomeDisagreement reports a document and an exit status that
	// contradict each other: a success object at a failure status, a failure
	// object at exit 0, or a failure object whose exit_code differs from the
	// status the process actually exited with.
	ErrOutcomeDisagreement = errors.New("invocation stdout and exit status disagree")

	// ErrUnregisteredExitStatus reports a process exit status that Section 15.2
	// assigns no meaning. Such an invocation is not classified: the pinned
	// table is the whole vocabulary, and inventing a class for a status outside
	// it would be a guess.
	ErrUnregisteredExitStatus = errors.New("invocation exit status is outside the Section 15.2 table")
)

// Reading is the machine-actionable classification of one completed invocation.
//
// Every answer it exposes is derived from the document on stdout. The human
// message is reachable through HumanMessage for display, and no accessor here
// is computed from it: Section 15.1 says "messages are for humans; automation
// MUST branch on code and exit_code", so a message-derived answer would be the
// defect this type exists to make impossible.
type Reading struct {
	command    Command
	result     *Result
	failure    *axerror.Error
	exitStatus int
}

// Read classifies one completed invocation of a command.
//
// The document on stdout decides the outcome and the exit status corroborates
// it; neither signal is trusted alone. Concretely:
//
//   - the command tag selects the CLI Result version and, through the static
//     Section 15.1 binding table, the Structured Error version - the document
//     never selects its own version;
//   - stdout must carry exactly one readable JSON document of one of those two
//     schemas, and an absence, a read failure, and a foreign schema are three
//     distinct refusals rather than one fallback;
//   - a failure document must carry the exact status the process exited with,
//     which is Section 14.2's "the process exit status MUST equal that error's
//     exit_code" checked from the reading side; and
//   - an exit status Section 15.2 does not assign is refused, because the
//     pinned table is the whole vocabulary.
//
// Nothing is inferred from the exit status by itself. A registered failure
// status maps to many codes - CodesByExitStatus measures how many - so the
// status narrows the class and never identifies the failure.
func Read(command Command, output InvocationOutput) (*Reading, error) {
	registered, err := RegisteredVersionForCommand(command)
	if err != nil {
		return nil, err
	}
	major, err := majorOf(registered)
	if err != nil {
		return nil, err
	}
	errorVersion, err := axerror.BindingFor(axerror.ContainingContract{ID: Schema, Major: major})
	if err != nil {
		return nil, err
	}
	if output.ExitStatus != SuccessExitStatus && !axerror.IsFailureExitStatus(output.ExitStatus) {
		return nil, fmt.Errorf("%w: %d", ErrUnregisteredExitStatus, output.ExitStatus)
	}
	schema, err := documentSchema(output.Stdout, output.ExitStatus, errorVersion)
	if err != nil {
		return nil, err
	}
	switch schema {
	case Schema:
		return readSuccess(command, output)
	case axerror.Schema:
		return readFailure(command, output, errorVersion)
	default:
		return nil, fmt.Errorf("%w: schema %q", ErrForeignDocument, schema)
	}
}

// documentSchema discriminates the one document on stdout without trusting it.
// It reads the schema member only, and it reads it after the whole document has
// been shown to be inside the Section 1.6 common logical data model, because a
// document whose schema member repeats has no single schema to discriminate on.
// encoding/json would resolve that repeat to the last occurrence and route the
// reading to a branch the other occurrence contradicts.
//
// What each branch then validates, stated exactly rather than delegated in the
// abstract. Both closed decoders run the same common-data-model gate on the same
// bytes - cliresult.Decode through canonicaljson.Canonicalize, axerror.Decode
// through requireCommonDataModel - and both then enforce their own closed
// top-level member set after settling the envelope identity. The gate here is
// therefore not the only thing standing between a duplicate member and a
// reading; it is what makes the member this function reads unambiguous, which
// neither closed decoder can do for it because both run after the branch is
// chosen.
//
// That statement is load-bearing and was wrong once. Until this leaf's rework
// axerror.Decode ran no such gate, so the sentence "the closed decoder for the
// selected contract then validates the whole object, including the duplicate
// members" was true of the success branch and false of the failure branch, and a
// Structured Error carrying "retryable": false followed by "retryable": true was
// read as retryable through this very function.
//
// The refusals carry the measured reason the exit status cannot stand in for
// the missing document, so a caller that logs the error is told the fan-in
// rather than left to assume the status was enough.
func documentSchema(stdout []byte, exitStatus int, errorVersion axerror.Version) (string, error) {
	if len(bytes.TrimSpace(stdout)) == 0 {
		return "", fmt.Errorf("%w: %s", ErrAbsentDocument, exitStatusIsNotEnough(exitStatus, errorVersion))
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	decoder.UseNumber()
	var document map[string]json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return "", fmt.Errorf("%w: decode: %v; %s", ErrUnreadableDocument, err, exitStatusIsNotEnough(exitStatus, errorVersion))
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "", fmt.Errorf(
			"%w: stdout carries more than one document, and Section 14.2 allows exactly one; %s",
			ErrUnreadableDocument, exitStatusIsNotEnough(exitStatus, errorVersion))
	}
	if _, err := canonicaljson.Canonicalize(stdout); err != nil {
		return "", fmt.Errorf(
			"%w: stdout is outside the Section 1.6 common logical data model, so it is not one document a machine can read the same way twice: %v; %s",
			ErrUnreadableDocument, err, exitStatusIsNotEnough(exitStatus, errorVersion))
	}
	raw, present := document["schema"]
	if !present {
		return "", fmt.Errorf("%w: document has no schema member; %s",
			ErrUnreadableDocument, exitStatusIsNotEnough(exitStatus, errorVersion))
	}
	// JSON null is checked alongside the unmarshal because encoding/json admits
	// null into a string and yields "". Without it a document whose schema
	// member is null was answered as a foreign contract carrying the schema "" -
	// a claim that some other contract owns the document, when what is true is
	// that this one is not readable. The two facts are reported separately
	// everywhere else in this reader and are reported separately here.
	var schema string
	if err := json.Unmarshal(raw, &schema); err != nil || string(bytes.TrimSpace(raw)) == "null" {
		return "", fmt.Errorf("%w: schema member is not a string; %s",
			ErrUnreadableDocument, exitStatusIsNotEnough(exitStatus, errorVersion))
	}
	return schema, nil
}

// exitStatusIsNotEnough states, as a measured count rather than as advice, why
// the reader refuses instead of classifying the invocation from its status. The
// count comes from the pinned registry of the bound error version.
func exitStatusIsNotEnough(exitStatus int, errorVersion axerror.Version) string {
	if exitStatus == SuccessExitStatus {
		return "exit 0 is the success status, and Section 14.2 requires the success object itself on stdout"
	}
	fanIn, err := axerror.CodesByExitStatus(errorVersion)
	if err != nil {
		return fmt.Sprintf("exit status %d cannot be resolved to a code without the document", exitStatus)
	}
	return fmt.Sprintf(
		"exit status %d is assigned to %d registered Structured Error %s codes, so the status alone identifies no failure",
		exitStatus, len(fanIn[exitStatus]), errorVersion)
}

func readSuccess(command Command, output InvocationOutput) (*Reading, error) {
	if output.ExitStatus != SuccessExitStatus {
		return nil, fmt.Errorf(
			"%w: stdout carries a CLI Result, which reports success, and the process exited %d",
			ErrOutcomeDisagreement, output.ExitStatus)
	}
	version, err := VersionForCommand(command)
	if err != nil {
		return nil, err
	}
	result, err := Decode(version, output.Stdout)
	if err != nil {
		return nil, err
	}
	if result.Command() != command {
		return nil, fmt.Errorf(
			"%w: stdout reports command %q and the invocation was %q",
			ErrOutcomeDisagreement, result.Command(), command)
	}
	return &Reading{command: command, result: result, exitStatus: output.ExitStatus}, nil
}

func readFailure(command Command, output InvocationOutput, errorVersion axerror.Version) (*Reading, error) {
	if output.ExitStatus == SuccessExitStatus {
		return nil, fmt.Errorf(
			"%w: stdout carries a Structured Error, which reports failure, and the process exited 0",
			ErrOutcomeDisagreement)
	}
	failure, err := axerror.Decode(errorVersion, output.Stdout)
	if err != nil {
		return nil, err
	}
	if failure.ExitCode() != output.ExitStatus {
		return nil, fmt.Errorf(
			"%w: the failure object carries exit_code %d and the process exited %d, "+
				"which Section 14.2 forbids: the process exit status must equal that error's exit_code",
			ErrOutcomeDisagreement, failure.ExitCode(), output.ExitStatus)
	}
	return &Reading{command: command, failure: failure, exitStatus: output.ExitStatus}, nil
}

// Command reports the command tag the reading was classified for.
func (reading *Reading) Command() Command { return reading.command }

// Succeeded reports whether the invocation succeeded. It is decided by the
// document on stdout, which the exit status had to agree with.
func (reading *Reading) Succeeded() bool { return reading.result != nil }

// Result returns the success object, or nil for a failure.
func (reading *Reading) Result() *Result { return reading.result }

// Failure returns the Structured Error, or nil for a success.
func (reading *Reading) Failure() *axerror.Error { return reading.failure }

// ExitStatus reports the status the process exited with, which equals the
// failure object's exit_code for a failure and SuccessExitStatus for a success.
func (reading *Reading) ExitStatus() int { return reading.exitStatus }

// Code reports the stable Section 15.3 code a failure carries. The second
// result is false for a success, which carries no code; it is never false for a
// failure, because a Structured Error cannot exist without one.
func (reading *Reading) Code() (axerror.Code, bool) {
	if reading.failure == nil {
		return "", false
	}
	return reading.failure.Code(), true
}

// CodeRegistered reports whether the pinned registry carries this failure's
// code for the bound version. A false answer is a code added by a later
// compatible minor: Section 15.3 says such a code "retains the envelope's exit
// class and MUST NOT be interpreted as success", which is exactly what this
// reading does with it.
func (reading *Reading) CodeRegistered() bool {
	return reading.failure != nil && reading.failure.CodeRegistered()
}

// Retryable reports the failure's retryable bit, and false for a success.
// Section 15.1 defines it as "true only when the identical request may safely
// be retried without new authority or confirmation".
func (reading *Reading) Retryable() bool {
	return reading.failure != nil && reading.failure.Retryable()
}

// HumanMessage returns the failure's human text, or "" for a success.
//
// It is here so that a client can display the message, and it is the only
// message-derived value this type exposes. Nothing else on Reading is computed
// from it.
func (reading *Reading) HumanMessage() string {
	if reading.failure == nil {
		return ""
	}
	return reading.failure.Message()
}
