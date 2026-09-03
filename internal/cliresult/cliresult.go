// Package cliresult implements the AX CLI Result contract of pinned
// specification Section 14.2: the independently versioned closed success
// envelope, the static per-command version selection, the closed embedded
// types and tagged command bodies, the reader rules of Section 17.2, the
// stdout/stderr rendering boundary, and the exact Section 15.2 process exit
// status a failure carries.
//
// The package holds no state, opens no file, and starts no process. It writes
// only to the streams a caller hands it, mutates no durable state, and
// advertises no provider, platform, backend, or CLI capability, so it has no
// crash or idempotency surface of its own.
//
// Success and failure are different objects, not two shapes of one object.
// Section 14.2 says "failure output is one Structured Error object from Section
// 15.1, not a CLI Result with ok = false", so this package can only build an
// object whose ok member is the literal true, and every failure path it exposes
// takes an *axerror.Error instead.
package cliresult

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Schema is the exact CLI Result schema identifier of Section 14.2.
const Schema = "urn:ax:schema:cli-result"

// Version is one registered CLI Result version. The registry is the pinned
// Section 1.5 row for urn:ax:schema:cli-result, which carries exactly these
// four; a value outside it is refused rather than coerced to a neighbour.
type Version string

const (
	// Version100 is selected by every legacy command and binds Structured
	// Error 1.0.0.
	Version100 Version = "1.0.0"
	// Version200 is selected by every session.clone.* command and binds
	// Structured Error 1.1.0.
	Version200 Version = "2.0.0"
	// Version300 is the Section 14.5 Session Directory surface. It is
	// registered and not implemented here; see ErrUnimplementedVersion.
	Version300 Version = "3.0.0"
	// Version400 is the Section 14.6 TerminalBackend surface. It is registered
	// and not implemented here; see ErrUnimplementedVersion.
	Version400 Version = "4.0.0"
)

var (
	// ErrInvalidResult reports a success object outside the closed Section 14.2
	// shape, bounds, or cross-member rules.
	ErrInvalidResult = errors.New("invalid cli result")

	// ErrUnsupportedVersion reports a CLI Result version outside the pinned
	// Section 1.5 registry.
	ErrUnsupportedVersion = errors.New("unsupported cli result version")

	// ErrUnimplementedVersion reports a registered CLI Result version this
	// repository does not build yet. It is a separate error from
	// ErrUnsupportedVersion on purpose: "this repository has no builder" and
	// "the specification registers no such version" are different facts, and
	// collapsing them would let a later slice mistake one for the other.
	ErrUnimplementedVersion = errors.New("cli result version is registered but not implemented in this repository")

	// ErrUnsupportedMajor reports a document whose major differs from the major
	// the reader was bound to. Section 17.2 rule 1 requires a reader to reject
	// an unsupported major rather than parse it.
	ErrUnsupportedMajor = errors.New("unsupported cli result major")

	// ErrVersionMismatch reports a registered version that is not the one the
	// reader expects.
	ErrVersionMismatch = errors.New("cli result version differs from the expected version")
)

// registeredVersions is the pinned registry in registry order. Index position
// is the only ordering used; the strings are never compared lexically.
var registeredVersions = []Version{Version100, Version200, Version300, Version400}

// implementedVersions is the subset this repository builds. Section 14.2
// defines CLI Result 1.0.0 and 2.0.0; 3.0.0 and 4.0.0 are defined by Sections
// 14.5 and 14.6, which this slice does not implement.
var implementedVersions = map[Version]struct{}{Version100: {}, Version200: {}}

func failf(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidResult, fmt.Sprintf(format, arguments...))
}

// Versions returns the registered CLI Result versions in registry order.
func Versions() []Version { return append([]Version(nil), registeredVersions...) }

// ImplementedVersions returns the registered versions this repository builds,
// in registry order. It is the measured denominator of this package's version
// coverage; a caller that wants a prose summary has to state the ratio itself.
func ImplementedVersions() []Version {
	var result []Version
	for _, version := range registeredVersions {
		if _, ok := implementedVersions[version]; ok {
			result = append(result, version)
		}
	}
	return result
}

func isRegisteredVersion(version Version) bool {
	for _, candidate := range registeredVersions {
		if candidate == version {
			return true
		}
	}
	return false
}

// requireImplementedVersion refuses an unregistered version and, separately, a
// registered version with no builder here.
func requireImplementedVersion(version Version) error {
	if !isRegisteredVersion(version) {
		return fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
	if _, ok := implementedVersions[version]; !ok {
		return fmt.Errorf("%w: %q", ErrUnimplementedVersion, version)
	}
	return nil
}

// ContractBound states what this package decides and what it does not, so that
// no caller reads a successful validation as more than it is. It is quoted in
// README.md and asserted by this package's tests, so a bound cannot be widened
// in the code while the disclosure keeps saying otherwise.
//
// Five limits are stated rather than discovered:
//
//   - CLI Result 3.0.0 and 4.0.0 are registered by the pinned Section 1.5 row
//     and are not built here; their command tags are refused with
//     ErrUnimplementedVersion, never emitted with an unchecked body.
//   - The eight session.clone.* tags select CLI Result 2.0.0, which is the
//     Section 14.2 rule this package implements, but their Section 14.1 closed
//     bodies are not built, so the version has a selection and no producer.
//   - The takeover adoption rule needs a session kind the body does not carry.
//     New requires it; Decode cannot have it and reports that rather than
//     skipping the check silently.
//   - An absolute-path member is admitted when it is absolute on any supported
//     platform, because a CLI Result names none. VerifyDestinationPlatform is
//     the narrowing hook a caller that knows the emitting host must use.
//   - The Section 18.1 total order of a logs event array is not checked: it is
//     a property of a durable stream, not of one array, and approximating it
//     with a bytewise sort would invent an ordering Section 18.1 does not
//     define.
//
// Two more limits belong to the reader. Section 17.2's same-or-lower-minor rule
// is implemented and unit-tested as acceptsVersion, but the pinned registry
// carries no two CLI Result versions sharing a major, so Decode reaches it only
// with equal versions. And this package validates shapes: it does not decide
// that a reported session state, capability status, or lease epoch is true of
// any host.
const ContractBound = "CLI Result validation decides shape, closed vocabulary, declared bounds, and the cross-member " +
	"rules Section 14.2 states. It does not build CLI Result 3.0.0 or 4.0.0, does not build the Section 14.1 " +
	"clone bodies of CLI Result 2.0.0, cannot check the takeover adoption rule without a session kind the " +
	"document does not carry, admits an absolute path that is absolute on any supported platform because the " +
	"object names none, does not enforce the Section 18.1 total order of a logs event array, and never decides " +
	"that a reported state, capability, or epoch is true of any host."

// SessionKind is the Section 14.2 session kind a takeover result is scoped to.
// It is a constructor input rather than a body member because the takeover body
// carries no kind of its own, and the adoption rule below cannot be decided
// without it.
type SessionKind string

const (
	// KindDirect is a direct session.
	KindDirect SessionKind = "direct"
	// KindTaskBoard is a task-board session.
	KindTaskBoard SessionKind = "task_board"
)

// IDs carries the two Section 14.2 top-level identifiers. Both are required
// members that may be JSON null, so a call site has to state which of the two
// it knows rather than leave a struct field at its zero value: NoIDs states
// that neither is known, and WithOperation and WithSession state that one is.
type IDs struct {
	operation *scalar.UUIDv7
	session   *scalar.UUIDv7
}

// NoIDs states that no operation was allocated and the result is not
// session-scoped.
func NoIDs() IDs { return IDs{} }

// WithOperation records the allocated operation identifier.
func (ids IDs) WithOperation(operation scalar.UUIDv7) IDs {
	ids.operation = &operation
	return ids
}

// WithSession records the session this result is scoped to.
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

// Session reports the session this result is scoped to, if one is known.
func (ids IDs) Session() (scalar.UUIDv7, bool) {
	if ids.session == nil {
		return scalar.UUIDv7{}, false
	}
	return *ids.session, true
}

// Spec is the complete input to New. It has no schema, schema_version, or ok
// field: the schema identifier is fixed, the version is resolved from the
// command through VersionForCommand, and ok is the literal true, so a call site
// cannot select a version the specification does not assign to its command or
// mint a failure-shaped success object.
type Spec struct {
	// Command is the Section 14.2 command tag that selects the body.
	Command Command
	// IDs carries the operation and session identifiers that are known.
	IDs IDs
	// Body is the tagged body. It is marshalled, validated against the strict
	// common data model, and decoded into the object this result owns, so the
	// graph the validator checked is the graph the result encodes.
	Body any
	// Extensions is the Section 1.6 extensions map. A nil value is the empty
	// map; the member itself is required and is always emitted.
	Extensions any
	// SessionKind is required for the takeover command and refused for every
	// other command, because Section 14.2 fixes the adoption rule per session
	// kind and no other body's validation consults it.
	SessionKind SessionKind
}

// Result is one validated CLI Result. Every field is unexported and the only
// encoder is MarshalJSON, so the closed Section 14.2 object is the only shape
// this type can produce.
//
// The body and extension graphs are owned outright rather than shared. New
// rebuilds both from their canonical bytes and the accessors deep-copy on the
// way out, so no caller retains a container it could write to after validation
// has already run and passed.
type Result struct {
	version    Version
	command    Command
	ids        IDs
	body       map[string]any
	extensions map[string]any
}

// New builds a CLI Result from spec. It refuses, in this order: an unknown or
// unimplemented command tag; a body or extensions map outside the strict common
// data model; a session-kind argument that the command does not take or does
// not have; an operation or session identifier that Section 14.2's nullability
// rules forbid or require; a body outside its command's closed shape; and an
// extensions map outside the Section 1.6 bounds.
//
// The version is never taken from the caller. It is resolved through
// VersionForCommand, so the returned object's schema_version is the exact
// version Section 14.2 assigns to the command.
func New(spec Spec) (*Result, error) {
	version, err := VersionForCommand(spec.Command)
	if err != nil {
		return nil, err
	}
	body, err := adoptObject(spec.Body, "body")
	if err != nil {
		return nil, err
	}
	extensions, err := adoptExtensions(spec.Extensions)
	if err != nil {
		return nil, err
	}
	if err := validateSessionKindArgument(spec.Command, spec.SessionKind); err != nil {
		return nil, err
	}
	result := &Result{
		version:    version,
		command:    spec.Command,
		ids:        spec.IDs,
		body:       body,
		extensions: extensions,
	}
	if err := result.validate(spec.SessionKind); err != nil {
		return nil, err
	}
	return result, nil
}

// validate runs every check that is shared between the writer and the reader.
// Decode calls it with an unknown session kind, because the document does not
// carry one; see VerifyTakeoverAdoption.
func (result *Result) validate(kind SessionKind) error {
	if err := validateIdentifiers(result.command, result.ids); err != nil {
		return err
	}
	if err := validateBody(result.command, result.body); err != nil {
		return err
	}
	if err := validateNestedSessionScope(result.command, result.ids, result.body); err != nil {
		return err
	}
	if kind != "" {
		if err := verifyTakeoverAdoption(result.command, result.body, kind); err != nil {
			return err
		}
	}
	return validateExtensions(result.extensions)
}

// adoptObject rebuilds a caller value as an owned object in the strict common
// data model.
//
// The caller's value is walked first, because encoding/json is permissive in
// exactly the two ways this contract is not: it replaces invalid UTF-8 with
// U+FFFD instead of failing, and it encodes a Go float as a JSON number that
// Section 1.6 forbids. Substituting a replacement character for a byte the
// caller supplied would silently change data the specification requires to be
// valid UTF-8, so the walk refuses it instead.
//
// Marshalling and re-parsing then enforces the rest of the model - the
// safe-integer interval, lone surrogates in escapes - and detaches the result
// from every container the caller still holds.
func adoptObject(value any, where string) (map[string]any, error) {
	if value == nil {
		return nil, failf("%s is required", where)
	}
	walked, err := walkInputValue(where, value, 0)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(walked)
	if err != nil {
		return nil, failf("encode %s: %v", where, err)
	}
	canonical, err := canonicaljson.Canonicalize(encoded)
	if err != nil {
		return nil, failf("%s is outside the common data model: %v", where, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, failf("decode %s: %v", where, err)
	}
	object, ok := decoded.(map[string]any)
	if !ok {
		return nil, failf("%s is not a JSON object", where)
	}
	return object, nil
}

// maxInputDepth bounds the walk below. It is a guard against an input the
// canonicalizer would refuse anyway, placed before the recursion rather than
// after it so a cyclic or pathological value cannot exhaust the stack first.
const maxInputDepth = 256

// walkInputValue admits exactly the Go forms of the Section 1.6 common data
// model and refuses everything else by name. A caller builds a body from
// map[string]any, []any, string, bool, nil, json.Number, and the Go integer
// types; a float, a struct, or any other value is refused rather than encoded
// into something the validators would then have to un-guess.
func walkInputValue(where string, value any, depth int) (any, error) {
	if depth > maxInputDepth {
		return nil, failf("%s exceeds the maximum nesting depth %d", where, maxInputDepth)
	}
	switch typed := value.(type) {
	case nil, bool:
		return value, nil
	case json.Number:
		return typed, nil
	case string:
		if !utf8.ValidString(typed) {
			return nil, failf("%s is not valid UTF-8", where)
		}
		return typed, nil
	case int:
		return json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int32:
		return json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int64:
		return json.Number(strconv.FormatInt(typed, 10)), nil
	case uint:
		return json.Number(strconv.FormatUint(uint64(typed), 10)), nil
	case uint32:
		return json.Number(strconv.FormatUint(uint64(typed), 10)), nil
	case uint64:
		return json.Number(strconv.FormatUint(typed, 10)), nil
	case []any:
		members := make([]any, len(typed))
		for index, member := range typed {
			walked, err := walkInputValue(fmt.Sprintf("%s[%d]", where, index), member, depth+1)
			if err != nil {
				return nil, err
			}
			members[index] = walked
		}
		return members, nil
	case map[string]any:
		object := make(map[string]any, len(typed))
		for key, member := range typed {
			if !utf8.ValidString(key) {
				return nil, failf("%s has a member name that is not valid UTF-8", where)
			}
			walked, err := walkInputValue(fmt.Sprintf("%s[%q]", where, key), member, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = walked
		}
		return object, nil
	default:
		return nil, failf("%s has type %T, which the common data model does not admit", where, value)
	}
}

// adoptExtensions treats a nil extensions input as the empty map. The member is
// required in every CLI Result, so an omitted one is emitted as {} rather than
// left absent or null.
func adoptExtensions(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	return adoptObject(value, "extensions")
}

func validateSessionKindArgument(command Command, kind SessionKind) error {
	takesKind := command == CommandTakeover
	switch {
	case takesKind && kind == "":
		return failf("command %q requires a session kind: Section 14.2 fixes its adoption rule per kind", command)
	case !takesKind && kind != "":
		return failf("command %q takes no session kind", command)
	case kind == "" || kind == KindDirect || kind == KindTaskBoard:
		return nil
	default:
		return failf("session kind %q is not direct or task_board", kind)
	}
}

// Version reports the CLI Result version of this object.
func (result *Result) Version() Version { return result.version }

// Command reports the Section 14.2 command tag that selects the body.
func (result *Result) Command() Command { return result.command }

// IDs reports the two top-level identifiers this object carries.
func (result *Result) IDs() IDs { return result.ids }

// Body returns a deep copy of the tagged body. The copy is what keeps a
// validated object validated: a nested container handed to a caller cannot be
// written back into this result.
func (result *Result) Body() map[string]any {
	return cloneObject(result.body)
}

// Extension returns one extension value as inert data, deep-copied. Section 1.6
// says an extension "MUST NOT shadow, weaken, or be required to interpret a
// core ownership, fencing, path-safety, secret-exclusion, or transaction fact",
// and nothing in this package consults the extensions map: no accessor above
// reads it, so an unknown or adversarial extension cannot change Command,
// Version, either identifier, or any body member.
func (result *Result) Extension(key string) (any, bool) {
	value, ok := result.extensions[key]
	if !ok {
		return nil, false
	}
	return cloneValue(value), true
}

// ExtensionKeys returns the extension keys in sorted order.
func (result *Result) ExtensionKeys() []string { return sortedKeys(result.extensions) }

// VerifyTakeoverAdoption checks the Section 14.2 rule that "for a task-board
// takeover, adopted MUST be true before resumed can be true; for a direct
// takeover it MUST be false".
//
// It is a separate call rather than part of Decode because the takeover body
// carries no session kind, and a reader that does not know the kind cannot
// decide the rule. Reporting that as an explicit call the caller has to make is
// the honest shape: the alternative is a decoder that silently skips a MUST and
// returns an object indistinguishable from a checked one.
func (result *Result) VerifyTakeoverAdoption(kind SessionKind) error {
	if kind != KindDirect && kind != KindTaskBoard {
		return failf("session kind %q is not direct or task_board", kind)
	}
	return verifyTakeoverAdoption(result.command, result.body, kind)
}

// wireResult is the closed Section 14.2 object: exactly the eight members the
// pinned sentence declares. The two identifiers are pointers without omitempty,
// because Section 1.6 requires a T|null member to be present and to carry JSON
// null when it is unknown - omitting it is a different, invalid document.
type wireResult struct {
	Schema        string         `json:"schema"`
	SchemaVersion Version        `json:"schema_version"`
	Command       Command        `json:"command"`
	OK            bool           `json:"ok"`
	OperationID   *string        `json:"operation_id"`
	SessionID     *string        `json:"session_id"`
	Body          map[string]any `json:"body"`
	Extensions    map[string]any `json:"extensions"`
}

// MarshalJSON encodes the closed Section 14.2 object. It is the only encoder in
// this package, so no future unexported field can reach the wire without being
// added here as a declared member. ok is the literal true and is not read from
// the value: a CLI Result with ok = false is not a shape this type can hold.
func (result *Result) MarshalJSON() ([]byte, error) {
	object := wireResult{
		Schema:        Schema,
		SchemaVersion: result.version,
		Command:       result.command,
		OK:            true,
		Body:          result.body,
		Extensions:    result.extensions,
	}
	if operation, known := result.ids.Operation(); known {
		rendered := operation.String()
		object.OperationID = &rendered
	}
	if session, known := result.ids.Session(); known {
		rendered := session.String()
		object.SessionID = &rendered
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(object); err != nil {
		return nil, failf("encode: %v", err)
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

// cloneObject deep-copies a validated object.
func cloneObject(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = cloneValue(value)
	}
	return clone
}

// cloneValue deep-copies the two container kinds a decoded value may open and
// returns every other value unchanged. That is safe rather than partial: the
// adopted value model contains only nil, bool, string, json.Number, []any, and
// map[string]any, and the four scalar forms are immutable values in Go.
func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		nested := make(map[string]any, len(typed))
		for key, member := range typed {
			nested[key] = cloneValue(member)
		}
		return nested
	case []any:
		members := make([]any, len(typed))
		for index, member := range typed {
			members[index] = cloneValue(member)
		}
		return members
	default:
		return value
	}
}

var _ json.Marshaler = (*Result)(nil)
