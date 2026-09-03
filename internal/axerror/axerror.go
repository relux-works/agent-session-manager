// Package axerror implements the AX Structured Error contract of pinned
// specification Sections 15.1 through 15.3: the closed versioned failure
// object, its stable code-to-exit registry, its retryability rule, its typed
// diagnostic details, and the redaction that keeps a local cause off the wire.
//
// The package holds no state, opens no file, and starts no process. It
// advertises no provider, platform, backend, or CLI capability, and mutates no
// durable state, so it has no crash or idempotency surface of its own.
package axerror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// ErrInvalidStructuredError reports a failure object outside the closed Section
// 15.1 shape.
var ErrInvalidStructuredError = errors.New("invalid structured error")

// IDs carries the two optional Section 15.1 identifiers. The pinned document
// says they "are optional and MUST be present when known", which is an
// obligation on the constructing call site rather than a shape a validator can
// check. This type turns the obligation into a decision the call site has to
// make: NoIDs states that neither identifier is known, and WithOperation and
// WithSession state that one is. There is no way to leave the question
// unanswered by forgetting a struct field.
type IDs struct {
	operation *scalar.UUIDv7
	session   *scalar.UUIDv7
}

// NoIDs states that no operation was allocated and the failure is not
// session-scoped.
func NoIDs() IDs { return IDs{} }

// WithOperation records an allocated operation identifier.
func (ids IDs) WithOperation(operation scalar.UUIDv7) IDs {
	ids.operation = &operation
	return ids
}

// WithSession records the session the failure is scoped to.
func (ids IDs) WithSession(session scalar.UUIDv7) IDs {
	ids.session = &session
	return ids
}

// Operation reports the allocated operation identifier, if one is known.
func (ids IDs) Operation() (scalar.UUIDv7, bool) {
	if ids.operation == nil {
		return scalar.UUIDv7{}, false
	}
	return *ids.operation, true
}

// Session reports the session the failure is scoped to, if one is known.
func (ids IDs) Session() (scalar.UUIDv7, bool) {
	if ids.session == nil {
		return scalar.UUIDv7{}, false
	}
	return *ids.session, true
}

// Spec is the complete input to New. It deliberately has no exit-code field:
// the exit status is resolved from the pinned registry for the selected
// version, so a call site cannot mint a mapping the specification does not
// assign, and no reviewer has to check one against the table by hand.
type Spec struct {
	// Version is the Structured Error version the containing contract binds.
	Version Version
	// Code is a stable registry code the selected version registers.
	Code Code
	// Message is human text of 1..4096 UTF-8 characters. Automation never
	// branches on it.
	Message string
	// Retryable is true only when the identical request may safely be retried
	// without new authority or confirmation.
	Retryable bool
	// IDs carries the operation and session identifiers that are known.
	IDs IDs
	// Details is the required diagnostic map. An empty map is valid; nil is
	// not, because the member itself is required.
	Details Details
	// Cause is the local Go error this failure was built from. It is kept for
	// errors.Is and errors.As and is structurally unreachable from the wire
	// object: no encoder in this package can serialize it.
	Cause error
}

// Error is one validated Structured Error. Every field is unexported and the
// only encoder is MarshalJSON, so the closed Section 15.1 object is the only
// shape this type can produce and the local cause cannot escape into it.
//
// The details map is owned outright rather than shared. New deep-copies the
// caller's map on the way in and Detail deep-copies on the way out, so the
// graph ValidateDetails checked is the graph this object encodes. A shallow
// copy would leave every Section 15.1 detail bound - the four forbidden
// classes, the 16 KiB canonical size, the depth-4 nesting limit, the admitted
// value types - violable after construction through a retained nested
// container, which is a bypass path around the gate rather than a gate.
type Error struct {
	version   Version
	code      Code
	message   string
	exitCode  int
	retryable bool
	ids       IDs
	details   Details
	cause     error

	// codeRegistered records whether the pinned registry carries this code for
	// this version. It is false only for an object decoded from a conforming
	// peer that used a code added in a later compatible minor.
	codeRegistered bool
}

// New builds a Structured Error from spec. It refuses, in this order: an
// unregistered version; a code the version does not register; a message outside
// 1..4096 UTF-8 characters; a details map outside the Section 15.1 bounds or
// carrying a Section 16.2 excluded class; a code whose typed details the pinned
// document names and spec omits; a retryable claim the pinned document
// forbids; and human text or a detail value that reproduces the local cause
// verbatim.
//
// The exit status is never taken from the caller. It is resolved through
// ExitCodeFor, so the returned object's exit_code is the exact Section 15.2
// status the registry assigns to the code.
//
// The typed-detail requirement below is asymmetric on purpose.
// target_auth_missing has exactly one declared use in Section 15.3 and that
// clause names its five typed details, so the generic constructor may not emit
// it with a weaker detail set than NewTargetAuthMissing produces.
// capability_unavailable is a general exit-6 code whose typed details Section
// 15.3 names only for the realm/broker-evidence case, so requiring them from
// every capability_unavailable call site would invent a constraint the pinned
// document does not state; NewRealmEvidenceUnavailable carries that
// requirement for the case that does declare it.
func New(spec Spec) (*Error, error) {
	exitCode, err := ExitCodeFor(spec.Version, spec.Code)
	if err != nil {
		return nil, err
	}
	if err := validateMessage(spec.Message); err != nil {
		return nil, err
	}
	if err := ValidateDetails(spec.Details); err != nil {
		return nil, err
	}
	if spec.Code == "target_auth_missing" {
		if err := requireDetailKeys(spec.Code, spec.Details, targetAuthMissingKeys); err != nil {
			return nil, err
		}
	}
	if spec.Retryable {
		if reason, forbidden := RetryabilityRefusal(spec.Code, exitCode); forbidden {
			return nil, fmt.Errorf(
				"%w: %q may not claim retryable: %s", ErrInvalidStructuredError, spec.Code, reason)
		}
	}
	if err := refuseCausalLeak(spec.Message, spec.Details, spec.Cause); err != nil {
		return nil, err
	}
	return &Error{
		version:        spec.Version,
		code:           spec.Code,
		message:        spec.Message,
		exitCode:       exitCode,
		retryable:      spec.Retryable,
		ids:            spec.IDs,
		details:        cloneDetails(spec.Details),
		cause:          spec.Cause,
		codeRegistered: true,
	}, nil
}

// NewRealmEvidenceUnavailable builds the Section 15.3 capability_unavailable
// failure for missing or unsafe broker, tmux server generation, or
// functional-sentinel evidence, with the exact typed details that clause names.
// It exists so that the realm case cannot be reported with an untyped detail
// map, and so that no realm-specific code is minted while capability_unavailable
// remains truthful.
func NewRealmEvidenceUnavailable(version Version, message string, ids IDs, evidence RealmEvidence, cause error) (*Error, error) {
	details, err := evidence.Details()
	if err != nil {
		return nil, err
	}
	return New(Spec{
		Version: version,
		Code:    "capability_unavailable",
		Message: message,
		IDs:     ids,
		Details: details,
		Cause:   cause,
	})
}

// NewTargetAuthMissing builds the Section 15.3 target_auth_missing failure for
// a missing or failed provider-auth smoke, with the exact typed details that
// clause names.
func NewTargetAuthMissing(version Version, message string, ids IDs, target TargetAuth, cause error) (*Error, error) {
	details, err := target.Details()
	if err != nil {
		return nil, err
	}
	return New(Spec{
		Version: version,
		Code:    "target_auth_missing",
		Message: message,
		IDs:     ids,
		Details: details,
		Cause:   cause,
	})
}

func validateMessage(message string) error {
	if !utf8.ValidString(message) {
		return fmt.Errorf("%w: message is not valid UTF-8", ErrInvalidStructuredError)
	}
	count := utf8.RuneCountInString(message)
	if count < minMessageRunes || count > maxMessageRunes {
		return fmt.Errorf(
			"%w: message is %d UTF-8 characters, the bound is %d..%d",
			ErrInvalidStructuredError, count, minMessageRunes, maxMessageRunes)
	}
	return nil
}

// Version reports the Structured Error version of this object.
func (failure *Error) Version() Version { return failure.version }

// Code reports the stable registry code. Automation branches on this and on
// ExitCode, never on Message.
func (failure *Error) Code() Code { return failure.code }

// Message reports the human text. It carries no machine meaning.
func (failure *Error) Message() string { return failure.message }

// ExitCode reports the exact Section 15.2 status carried by this object. It is
// never the success status.
func (failure *Error) ExitCode() int { return failure.exitCode }

// Retryable reports whether the identical request may safely be retried without
// new authority or confirmation. It is false for every code and exit class the
// pinned document disqualifies, on both the writing and the reading side.
func (failure *Error) Retryable() bool { return failure.retryable }

// IDs reports the optional identifiers this object carries.
func (failure *Error) IDs() IDs { return failure.ids }

// CodeRegistered reports whether the pinned registry carries this object's code
// for this object's version. A false result is not an error: Section 15.3
// admits a code added in a compatible minor, and such an object still carries
// its envelope's exit class. It is reported rather than inferred so that a
// caller never mistakes an unrecognized code for a recognized one.
func (failure *Error) CodeRegistered() bool { return failure.codeRegistered }

// Detail returns one diagnostic value as inert data. Section 15.1 says readers
// "MUST never infer success, authority, or a remediation action from them", and
// nothing in this package does: no accessor above consults the details map, so
// an unknown or adversarial detail key cannot change Code, ExitCode, Retryable,
// CodeRegistered, or any identifier.
//
// The returned value is a deep copy: a nested object or array handed to a
// caller cannot be written back into this object, so a validated failure stays
// exactly the object ValidateDetails admitted.
func (failure *Error) Detail(key string) (any, bool) {
	value, ok := failure.details[key]
	if !ok {
		return nil, false
	}
	return cloneDetailValue(value), true
}

// DetailKeys returns the diagnostic keys in sorted order.
func (failure *Error) DetailKeys() []string { return sortedKeys(failure.details) }

// Error implements the error interface. It renders the code and the human text
// and never the cause, so wrapping this value with %v cannot copy untrusted or
// sensitive cause text into a new message.
func (failure *Error) Error() string {
	return string(failure.code) + ": " + failure.message
}

// Unwrap exposes the local cause for errors.Is and errors.As. The cause is a
// local Go value: it is never encoded, and MarshalJSON has no access to it.
func (failure *Error) Unwrap() error { return failure.cause }

// wireError is the closed Section 15.1 object. Its field set is exactly the
// nine members the pinned table declares, and the two optional identifiers are
// omitted when unknown rather than emitted as null.
type wireError struct {
	Schema        string  `json:"schema"`
	SchemaVersion Version `json:"schema_version"`
	Code          Code    `json:"code"`
	Message       string  `json:"message"`
	ExitCode      int     `json:"exit_code"`
	Retryable     bool    `json:"retryable"`
	OperationID   *string `json:"operation_id,omitempty"`
	SessionID     *string `json:"session_id,omitempty"`
	Details       Details `json:"details"`
}

// MarshalJSON encodes the closed Section 15.1 object. It is the only encoder in
// this package, and it can reach neither the cause nor any other local field,
// so no future field added to Error can leak onto the wire without being added
// here as a declared member.
func (failure *Error) MarshalJSON() ([]byte, error) {
	object := wireError{
		Schema:        Schema,
		SchemaVersion: failure.version,
		Code:          failure.code,
		Message:       failure.message,
		ExitCode:      failure.exitCode,
		Retryable:     failure.retryable,
		Details:       failure.details,
	}
	if operation, known := failure.ids.Operation(); known {
		rendered := operation.String()
		object.OperationID = &rendered
	}
	if session, known := failure.ids.Session(); known {
		rendered := session.String()
		object.SessionID = &rendered
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	// The encode failure below is unreachable for a validated object: every
	// member is a string, bool, int, or a details map whose values already
	// passed ValidateDetails. It is reported rather than ignored so that a
	// future member type cannot fail silently.
	if err := encoder.Encode(object); err != nil {
		return nil, fmt.Errorf("%w: encode: %v", ErrInvalidStructuredError, err)
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

// cloneDetails deep-copies a validated diagnostic map. It always returns a
// non-nil map, so the required details member is present in every encoded
// object even when the constructing call site passed an empty one, and
// MarshalJSON therefore needs no nil guard of its own.
//
// The copy is deep because a shallow one only isolates the top level. The
// caller of New keeps whatever nested containers it passed, and a nested
// container handed back by Detail is the same allocation; either one can be
// written to after ValidateDetails has already run and passed.
func cloneDetails(source Details) Details {
	clone := make(Details, len(source))
	for key, value := range source {
		clone[key] = cloneDetailValue(value)
	}
	return clone
}

// cloneDetailValue deep-copies the two container kinds a detail value may open
// and returns every other admitted value unchanged. That is safe rather than
// partial: validateDetailValue admits only nil, bool, string, json.Number,
// []any, and map[string]any, and the four scalar forms are immutable values in
// Go. Any other type is refused before a value reaches this function, so no
// mutable value can fall through the default arm into a constructed object.
func cloneDetailValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		nested := make(map[string]any, len(typed))
		for key, member := range typed {
			nested[key] = cloneDetailValue(member)
		}
		return nested
	case []any:
		members := make([]any, len(typed))
		for index, member := range typed {
			members[index] = cloneDetailValue(member)
		}
		return members
	default:
		return value
	}
}

var _ json.Marshaler = (*Error)(nil)
var _ error = (*Error)(nil)
