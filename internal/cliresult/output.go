package cliresult

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// Mode is the Section 14.2 output mode. --json selects "one version-selected
// CLI Result success object or Structured Error failure object"; its absence
// selects text.
type Mode string

const (
	// ModeText is the default human mode: "data goes to stdout and
	// prompts/diagnostics to stderr".
	ModeText Mode = "text"
	// ModeJSON is structured mode: "stdout MUST contain exactly one JSON
	// document; logs remain on stderr".
	ModeJSON Mode = "json"
)

// SuccessExitStatus is the Section 15.2 status of a successful operation. It is
// the only status a CLI Result can produce.
const SuccessExitStatus = 0

var (
	// ErrStreamDiscipline reports a write that would violate the Section 14.2
	// stdout/stderr boundary, including a second document on stdout in JSON
	// mode.
	ErrStreamDiscipline = errors.New("cli output stream discipline violated")

	// ErrPromptForbidden reports a prompt attempted under --non-interactive,
	// which Section 14.2 says "forbids prompts".
	ErrPromptForbidden = errors.New("prompt forbidden by --non-interactive")
)

// Streams are the two process streams a command writes to, plus whether stderr
// is a terminal. Section 14.2 permits progress on stderr "only when it is a
// TTY", so the answer is an input rather than something this package probes:
// the caller owns the file descriptors and this package opens nothing.
type Streams struct {
	Stdout      io.Writer
	Stderr      io.Writer
	StderrIsTTY bool
}

// Emitter enforces the Section 14.2 rendering boundary for one invocation. It
// is not a renderer: it decides which stream each kind of output belongs on and
// how many documents stdout may carry, and it never invents human text for a
// body the specification does not describe in human form.
type Emitter struct {
	mode           Mode
	streams        Streams
	nonInteractive bool
	emitted        bool
}

// NewEmitter builds the rendering boundary for a parsed invocation.
func NewEmitter(invocation *Invocation, streams Streams) (*Emitter, error) {
	if invocation == nil {
		return nil, fmt.Errorf("%w: invocation is nil", ErrStreamDiscipline)
	}
	return newEmitter(invocation.Mode(), invocation.NonInteractive(), streams)
}

func newEmitter(mode Mode, nonInteractive bool, streams Streams) (*Emitter, error) {
	if mode != ModeText && mode != ModeJSON {
		return nil, fmt.Errorf("%w: %q is not a supported output mode", ErrStreamDiscipline, mode)
	}
	if streams.Stdout == nil || streams.Stderr == nil {
		return nil, fmt.Errorf("%w: both streams are required", ErrStreamDiscipline)
	}
	return &Emitter{mode: mode, streams: streams, nonInteractive: nonInteractive}, nil
}

// Mode reports the output mode this emitter enforces.
func (emitter *Emitter) Mode() Mode { return emitter.mode }

// Log writes one diagnostic line. Section 14.2 puts diagnostics on stderr in
// text mode and says "logs remain on stderr" in JSON mode, so the destination
// is the same in both and there is no mode in which a log can reach stdout.
func (emitter *Emitter) Log(line string) error {
	return writeLine(emitter.streams.Stderr, line)
}

// Progress writes one progress line. Section 14.2 says "progress MAY use stderr
// only when it is a TTY", so a non-TTY stderr drops the line instead of
// choosing another stream for it. The boolean reports whether it was written,
// because silently dropping output without saying so is how a caller comes to
// believe something was displayed.
func (emitter *Emitter) Progress(line string) (bool, error) {
	if !emitter.streams.StderrIsTTY {
		return false, nil
	}
	if err := writeLine(emitter.streams.Stderr, line); err != nil {
		return false, err
	}
	return true, nil
}

// Prompt writes one prompt to stderr. It refuses under --non-interactive, which
// Section 14.2 says "forbids prompts". The refusal is the point: a prompt that
// silently degrades to a default is exactly the confirmation bypass Section
// 14.2 forbids elsewhere.
func (emitter *Emitter) Prompt(line string) error {
	if emitter.nonInteractive {
		return fmt.Errorf("%w: %q", ErrPromptForbidden, line)
	}
	return writeLine(emitter.streams.Stderr, line)
}

// Outcome is the single result of one command invocation. Exactly one of
// Result and Failure is set: Section 14.2 makes success and failure different
// objects, so an outcome carrying both, or neither, is not representable as a
// conforming emission and is refused.
type Outcome struct {
	// Result is the success object, or nil for a failure.
	Result *Result
	// Failure is the Structured Error, or nil for a success.
	Failure *axerror.Error
	// Rendered is the human text for text mode. It is required in text mode
	// and refused in JSON mode, where stdout carries exactly one JSON document
	// and nothing else.
	Rendered string
}

// Emit writes the outcome and returns the exact process exit status for it.
//
// The status is the Section 14.2 rule verbatim: "the process exit status MUST
// equal that error's exit_code". A success is SuccessExitStatus, and a failure
// is the status the Structured Error registry assigned to its code - this
// function does not compute, adjust, or clamp it.
//
// In JSON mode stdout receives exactly one JSON document and nothing else, and
// a second call is refused. In text mode the success rendering goes to stdout
// and the failure rendering goes to stderr, because Section 14.2 puts data on
// stdout and diagnostics on stderr and a failure is a diagnostic.
//
// The returned status is valid even when the write fails: the operation's
// outcome is already decided, and reporting exit 0 because a pipe closed would
// misreport a failure as a success.
func (emitter *Emitter) Emit(outcome Outcome) (int, error) {
	status, err := outcome.exitStatus()
	if err != nil {
		return status, err
	}
	if emitter.emitted {
		return status, fmt.Errorf("%w: stdout already carries this command's one document", ErrStreamDiscipline)
	}
	emitter.emitted = true
	if emitter.mode == ModeJSON {
		if outcome.Rendered != "" {
			return status, fmt.Errorf(
				"%w: JSON mode carries exactly one JSON document, so human text is not emitted",
				ErrStreamDiscipline)
		}
		document, err := outcome.document()
		if err != nil {
			return status, err
		}
		return status, writeLine(emitter.streams.Stdout, string(document))
	}
	if outcome.Rendered == "" {
		return status, fmt.Errorf("%w: text mode requires a human rendering", ErrStreamDiscipline)
	}
	if outcome.Failure != nil {
		return status, writeLine(emitter.streams.Stderr, outcome.Rendered)
	}
	return status, writeLine(emitter.streams.Stdout, outcome.Rendered)
}

// exitStatus resolves the Section 14.2 process exit status of an outcome.
func (outcome Outcome) exitStatus() (int, error) {
	switch {
	case outcome.Result != nil && outcome.Failure != nil:
		return 0, fmt.Errorf("%w: an outcome is a success or a failure, never both", ErrStreamDiscipline)
	case outcome.Result != nil:
		return SuccessExitStatus, nil
	case outcome.Failure != nil:
		return outcome.Failure.ExitCode(), nil
	default:
		return 0, fmt.Errorf("%w: an outcome carries neither a result nor a failure", ErrStreamDiscipline)
	}
}

// ExitStatus reports the process exit status of an outcome without emitting it.
// It exists so that a caller can settle the status and the emission separately
// and still get one answer from one place.
func ExitStatus(outcome Outcome) (int, error) { return outcome.exitStatus() }

func (outcome Outcome) document() ([]byte, error) {
	if outcome.Failure != nil {
		return outcome.Failure.MarshalJSON()
	}
	return outcome.Result.MarshalJSON()
}

// writeLine terminates every write. A stream that must carry exactly one
// document cannot afford an unterminated write merging with whatever follows
// it, and a terminator is what makes "one document" observable to a reader.
func writeLine(writer io.Writer, line string) error {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	_, err := io.WriteString(writer, line)
	return err
}

// DestructiveOperation names one Section 14.2 "destructive or split-brain-risk"
// operation together with the expectation flags Section 14.1 documents for it.
type DestructiveOperation string

const (
	// OperationForceTakeover is ax takeover --force, whose Section 14.1 grammar
	// is "[--force --expect-owner ID --expect-epoch N] [--yes]".
	OperationForceTakeover DestructiveOperation = "takeover --force"
	// OperationReplaceManagedReplica is ax materialize
	// --replace-managed-replica, which Section 14.1 says "requires
	// --expect-checkpoint equal to its managed marker" and whose
	// "non-interactive replacement also requires --yes".
	OperationReplaceManagedReplica DestructiveOperation = "materialize --replace-managed-replica"
)

// destructiveOperations is the reviewed table of operations that Section 14.2's
// confirmation rule governs, each with the expectation flags Section 14.1
// documents for it. An operation is listed only where the pinned document
// states its expectation flags; nothing is added here because it merely looks
// risky.
var destructiveOperations = map[DestructiveOperation][]string{
	OperationForceTakeover:         {"--expect-epoch", "--expect-owner"},
	OperationReplaceManagedReplica: {"--expect-checkpoint"},
}

// DestructiveOperations returns the governed operations in sorted order.
func DestructiveOperations() []DestructiveOperation {
	result := make([]DestructiveOperation, 0, len(destructiveOperations))
	for operation := range destructiveOperations {
		result = append(result, operation)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

// ExpectationFlags returns the documented expectation flags of an operation, in
// sorted order.
func ExpectationFlags(operation DestructiveOperation) ([]string, error) {
	flags, known := destructiveOperations[operation]
	if !known {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSurface, operation)
	}
	return append([]string(nil), flags...), nil
}

// Confirmation is the decision the Section 14.2 confirmation rule produces.
type Confirmation struct {
	// PromptRequired is true when the caller must prompt before mutating.
	// Section 14.2: "destructive or split-brain-risk operations MUST prompt in
	// interactive mode".
	PromptRequired bool
}

// RequireConfirmation decides whether a destructive or split-brain-risk
// operation may proceed.
//
// Section 14.2 states the rule in two sentences, and both are enforced here:
// such an operation "MUST prompt in interactive mode and require --yes plus
// every documented expectation flag in non-interactive mode", and "--yes alone
// MUST NOT bypass an expected owner/epoch/checkpoint check".
//
// The second sentence is why the expectation flags are checked before --yes is
// consulted and independently of it: a missing expectation flag is refused
// whether or not --yes is present, in interactive and non-interactive mode
// alike, so confirming can never stand in for stating the expected owner,
// epoch, or checkpoint.
func RequireConfirmation(
	operation DestructiveOperation,
	invocation *Invocation,
	supplied []string,
) (Confirmation, *axerror.Error) {
	expected, err := ExpectationFlags(operation)
	if err != nil {
		return Confirmation{}, mustInvalidArguments(err.Error())
	}
	if invocation == nil {
		return Confirmation{}, mustInvalidArguments("no parsed invocation was supplied")
	}
	present := make(map[string]struct{}, len(supplied))
	for _, flag := range supplied {
		present[flag] = struct{}{}
	}
	var missing []string
	for _, flag := range expected {
		if _, ok := present[flag]; !ok {
			missing = append(missing, flag)
		}
	}
	if len(missing) > 0 {
		return Confirmation{}, mustInvalidArguments(fmt.Sprintf(
			"%s requires %s; --yes does not bypass an expected owner, epoch, or checkpoint check",
			operation, strings.Join(missing, " and ")))
	}
	if !invocation.NonInteractive() {
		return Confirmation{PromptRequired: true}, nil
	}
	if !invocation.Yes() {
		return Confirmation{}, mustConfirmationRequired(fmt.Sprintf(
			"%s requires --yes in non-interactive mode", operation))
	}
	return Confirmation{}, nil
}

// mustConfirmationRequired builds the Section 15.3 exit-16 refusal for a
// missing destructive confirmation.
func mustConfirmationRequired(message string) *axerror.Error {
	failure, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    "confirmation_required",
		Message: message,
		IDs:     axerror.NoIDs(),
		Details: axerror.Details{},
	})
	if err != nil {
		panic(fmt.Sprintf("cli result confirmation refusal is unconstructible: %v", err))
	}
	return failure
}
