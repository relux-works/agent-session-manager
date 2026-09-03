package axerror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	// ErrUnsupportedMajor reports a Structured Error document whose major is not
	// 1. Section 17.2 rule 1 requires a reader to reject an unsupported major
	// rather than parse it, and Section 15.1 forbids trusting a different
	// major's code, retryable bit, details, or authority fields.
	ErrUnsupportedMajor = errors.New("unsupported structured error major")

	// ErrVersionMismatch reports a supported version that is not the one the
	// containing contract statically binds.
	ErrVersionMismatch = errors.New("structured error version differs from the bound version")
)

// semverPattern accepts only a full, unpadded semantic version triple.
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// wireDocument is the closed decode target. Every member is a pointer so that
// an omitted required member is distinguishable from a present zero value, and
// DisallowUnknownFields makes any tenth member a refusal: an authority field
// smuggled alongside the nine declared ones is rejected, not ignored.
type wireDocument struct {
	Schema        *string          `json:"schema"`
	SchemaVersion *string          `json:"schema_version"`
	Code          *string          `json:"code"`
	Message       *string          `json:"message"`
	ExitCode      *json.RawMessage `json:"exit_code"`
	Retryable     *bool            `json:"retryable"`
	OperationID   *json.RawMessage `json:"operation_id"`
	SessionID     *json.RawMessage `json:"session_id"`
	Details       *Details         `json:"details"`
}

// Decode reads one Structured Error document that is expected to carry exactly
// version.
//
// The common logical data model is settled first, before the envelope identity,
// because a document whose members repeat has no unambiguous identity to settle:
// see requireCommonDataModel. Nothing about the document is reported to the
// caller from that pass except that the bytes are outside the model.
//
// The envelope identity is settled before any other member is consulted, the
// closed top-level member set included. A document whose schema is not
// urn:ax:schema:error, whose major is not 1, or whose supported minor is not
// the expected one is refused without this function having read its member set,
// code, retryable bit, details, or identifiers, so no part of a wrong-version
// payload can be trusted on the way to that refusal.
//
// The member set is checked after the identity rather than before it, because
// the pinned document scopes that obligation to the object it governs: Section
// 1.6 requires a reader to "reject an unknown top-level field in a major
// version 1 object", Section 17.1 scopes the same rule to "within any
// negotiated major version", and Section 17.2 lists "rejects an unsupported
// major" as the reader's first rule. Section 15.1 is explicit that "receivers
// MUST NOT parse a different major's payload far enough to trust its error
// code, retryable bit, details, or authority fields", and reporting which
// members a different major's payload carries is exactly that kind of reading.
// Both orders refuse; only one of them tells the caller the version fact it
// needs.
//
// A code the registry does not carry is admitted, because Section 15.3 states
// that "New error codes MAY be added in a compatible minor contract version".
// Such an object keeps its envelope's exit class, is reported by
// CodeRegistered as unrecognized, and is never a success: its exit status must
// still be a registered Section 15.2 failure status, and the retryability
// refusals that apply to that exit class still apply.
func Decode(version Version, data []byte) (*Error, error) {
	if !isRegisteredVersion(version) {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
	if err := requireCommonDataModel(data); err != nil {
		return nil, err
	}
	identity, err := parseEnvelopeIdentity(data)
	if err != nil {
		return nil, err
	}
	if err := verifyEnvelopeIdentity(identity, version); err != nil {
		return nil, err
	}
	document, err := decodeClosedDocument(data)
	if err != nil {
		return nil, err
	}
	return decodeBody(version, document)
}

// DecodeBound reads a Structured Error embedded in a containing contract. The
// version is resolved from the static binding table, never from the document,
// so a peer cannot select its own error version by writing one into the
// payload.
func DecodeBound(contract ContainingContract, data []byte) (*Error, error) {
	version, err := BindingFor(contract)
	if err != nil {
		return nil, err
	}
	return Decode(version, data)
}

// requireCommonDataModel refuses bytes that are outside the Section 1.6 common
// logical data model before any member of the document is read.
//
// It exists because a duplicate member has no single value. Section 1.6 states
// the rule twice and in both directions, quoted verbatim from the pinned
// internal/specdoc/SPEC.md at the lines named:
//
//	SPEC.md:218 "map keys MUST be UTF-8 strings and MUST be unique"
//	SPEC.md:221 "floating-point numbers, NaN, Infinity, non-string map keys, and duplicate keys are forbidden"
//
// encoding/json does not refuse such a document:
// it resolves repeats, and it resolves them differently depending on the decode
// target. A map decode keeps the last occurrence, a struct decode keeps the last
// occurrence, and a decode into a map-typed member such as details merges both
// occurrences into one map. So a Structured Error carrying "retryable": false
// followed by "retryable": true used to be read as retryable - a forged retry
// claim assembled out of two members neither of which a conforming writer could
// have emitted - and a repeated details resolved to the union of both
// occurrences, which is neither one. Two conforming readers of the same bytes
// disagreed, which is precisely the compatibility property this package exists
// to hold.
//
// The gate is the same one the CLI Result reader runs (cliresult.Decode), so
// both branches of a machine reading now enforce the same data model rather than
// one of them relying on the other to do it. It reaches every peer-supplied
// envelope through DecodeBound: provider, bridge, RPC, session-adapter and
// terminal-backend payloads are read here, so the duplicate-member shape is
// remotely reachable and is refused at the same place for all of them.
//
// Canonicalize's strict decoder settles more than duplicates - malformed UTF-8,
// lone or unpaired surrogate escapes, unescaped control characters in strings,
// trailing data after the top-level value, and more than 256 simultaneously open
// containers. Measured against the mutant that deletes this call, it is the only
// refusal for duplicates, malformed UTF-8, lone surrogate escapes and trailing
// content: all four redden. An unescaped control character does NOT redden,
// because validateMessage refuses it independently for the message member, so
// that case is subsumed rather than owned here and is stated as such rather than
// counted as coverage this gate provides.
//
// THE CANONICAL BYTES ARE DELIBERATELY DISCARDED. Canonicalize is used as a
// gate over the caller's bytes, not as a rewrite of them, and Decode keeps
// parsing the original document. RFC 8785 Section 3.2.2.3 serializes every
// number through the ECMAScript Number.prototype.toString algorithm, so the
// transform rewrites 1e1 to 10 and 9.0 to 9. decodeExitStatus reads exit_code
// from its raw bytes precisely so that the exponent of 1e1 and the point of 9.0
// are refused rather than normalized, and adopting the canonical form here would
// hand it a document those literals had already been laundered out of.
// TestTheCanonicalGateDoesNotLaunderTheExitStatusToken pins that, and the
// mutation harness carries the adopt-the-canonical-bytes mutant.
func requireCommonDataModel(data []byte) error {
	if _, err := canonicaljson.Canonicalize(data); err != nil {
		return fmt.Errorf(
			"%w: document is outside the Section 1.6 common logical data model: %v",
			ErrInvalidStructuredError, err)
	}
	return nil
}

// decodeClosedDocument reads the nine declared members and refuses a tenth. It
// no longer checks for trailing content: parseEnvelopeIdentity runs first on the
// same bytes and settles that, and a second unreachable guard would be a branch
// no test could ever redden.
func decodeClosedDocument(data []byte) (*wireDocument, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var document wireDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrInvalidStructuredError, err)
	}
	return &document, nil
}

// envelopeIdentity is the two members that decide which contract a document
// belongs to. It is read from the raw bytes rather than from the closed decode
// target, because the closed member set is an obligation on a document of this
// schema's major and that major is what these two members settle.
type envelopeIdentity struct {
	schema        *string
	schemaVersion *string
}

// parseEnvelopeIdentity reads the schema and schema_version members without
// deciding anything else about the document. It refuses bytes that are not one
// JSON object, and it refuses a schema or schema_version that is present and is
// not a string, because neither can be compared without being read.
//
// It no longer checks for trailing content either. requireCommonDataModel runs
// first on the same bytes and its strict decoder refuses trailing data, so the
// guard that used to live here would be a branch no test could redden. The
// coverage did not move with it: TestReaderRefusalTable's trailing-content row
// still drives Decode, and the harness mutant that removes the gate reddens that
// row rather than only the duplicate-member ones.
func parseEnvelopeIdentity(data []byte) (envelopeIdentity, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var members map[string]json.RawMessage
	if err := decoder.Decode(&members); err != nil {
		return envelopeIdentity{}, fmt.Errorf("%w: decode: %v", ErrInvalidStructuredError, err)
	}
	identity := envelopeIdentity{}
	for name, target := range map[string]**string{
		"schema":         &identity.schema,
		"schema_version": &identity.schemaVersion,
	} {
		raw, present := members[name]
		if !present {
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return envelopeIdentity{}, fmt.Errorf(
				"%w: %s is not a JSON string", ErrInvalidStructuredError, name)
		}
		*target = &value
	}
	return identity, nil
}

// verifyEnvelopeIdentity settles schema and version before anything else is
// read. It is deliberately separate from decodeClosedDocument and decodeBody so
// that the ordering is visible rather than incidental.
func verifyEnvelopeIdentity(document envelopeIdentity, expected Version) error {
	if document.schema == nil || *document.schema != Schema {
		return fmt.Errorf("%w: schema is not %s", ErrInvalidStructuredError, Schema)
	}
	if document.schemaVersion == nil {
		return fmt.Errorf("%w: schema_version is required", ErrInvalidStructuredError)
	}
	raw := *document.schemaVersion
	matches := semverPattern.FindStringSubmatch(raw)
	if matches == nil {
		return fmt.Errorf("%w: schema_version %q is not a semantic version", ErrInvalidStructuredError, raw)
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil || major != 1 {
		return fmt.Errorf("%w: %q", ErrUnsupportedMajor, raw)
	}
	candidate := Version(raw)
	if !isRegisteredVersion(candidate) {
		return fmt.Errorf("%w: %q", ErrUnsupportedVersion, raw)
	}
	if candidate != expected {
		return fmt.Errorf("%w: document is %s, bound version is %s", ErrVersionMismatch, raw, expected)
	}
	return nil
}

func decodeBody(version Version, document *wireDocument) (*Error, error) {
	if document.Code == nil {
		return nil, fmt.Errorf("%w: code is required", ErrInvalidStructuredError)
	}
	code := Code(*document.Code)
	if err := validateCodeGrammar(code); err != nil {
		return nil, err
	}
	if document.Message == nil {
		return nil, fmt.Errorf("%w: message is required", ErrInvalidStructuredError)
	}
	if err := validateMessage(*document.Message); err != nil {
		return nil, err
	}
	if document.ExitCode == nil {
		return nil, fmt.Errorf("%w: exit_code is required", ErrInvalidStructuredError)
	}
	exitCode, err := decodeExitStatus(*document.ExitCode)
	if err != nil {
		return nil, err
	}
	registered := true
	expectedExit, lookupErr := ExitCodeFor(version, code)
	switch {
	case lookupErr == nil:
		if exitCode != expectedExit {
			return nil, fmt.Errorf(
				"%w: %q maps to exit %d, document carries %d",
				ErrInvalidStructuredError, code, expectedExit, exitCode)
		}
	case errors.Is(lookupErr, ErrUnregisteredCode):
		// Section 15.3: an unknown code "retains the envelope's exit class and
		// MUST NOT be interpreted as success". The class is retained by the
		// registered-failure-status check above; success is impossible because
		// decodeExitStatus refuses status 0.
		registered = false
	default:
		return nil, lookupErr
	}
	if document.Retryable == nil {
		return nil, fmt.Errorf("%w: retryable is required", ErrInvalidStructuredError)
	}
	if *document.Retryable {
		if reason, forbidden := RetryabilityRefusal(code, exitCode); forbidden {
			return nil, fmt.Errorf(
				"%w: %q may not claim retryable: %s", ErrInvalidStructuredError, code, reason)
		}
	}
	if document.Details == nil {
		return nil, fmt.Errorf("%w: details is required", ErrInvalidStructuredError)
	}
	// This map was allocated by the decoder from the document bytes and no
	// caller holds a reference to it, so the object below owns it already. New
	// clones instead, because its map comes from a call site that keeps one.
	details := *document.Details
	if err := ValidateDetails(details); err != nil {
		return nil, err
	}
	if code == "target_auth_missing" {
		if err := requireDetailKeys(code, details, targetAuthMissingKeys); err != nil {
			return nil, err
		}
	}
	ids, err := decodeIDs(document)
	if err != nil {
		return nil, err
	}
	return &Error{
		version:        version,
		code:           code,
		message:        *document.Message,
		exitCode:       exitCode,
		retryable:      *document.Retryable,
		ids:            ids,
		details:        details,
		codeRegistered: registered,
	}, nil
}

// decodeExitStatus reads exit_code from its raw bytes rather than through a
// json.Number field. encoding/json will unmarshal the JSON string "9" into a
// json.Number, which would let a peer write its exit status as text.
//
// Reading the raw bytes makes the JSON type part of the same parse that reads
// the value, and there is deliberately no separate type guard in front of it:
// the quotes of "9", the point of 9.0, the exponent of 1e1, and the letters of
// true, null and [] are all bytes strconv.ParseInt refuses. A leading +, a
// leading zero, or an underscore separator never reaches here, because the
// document scanner rejects those before json.RawMessage captures a value.
func decodeExitStatus(raw json.RawMessage) (int, error) {
	text := string(bytes.TrimSpace(raw))
	value, err := strconv.ParseInt(text, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: exit_code %s is not a JSON int32", ErrInvalidStructuredError, text)
	}
	status := int(value)
	if !IsFailureExitStatus(status) {
		return 0, fmt.Errorf("%w: exit_code %d", ErrUnregisteredExit, status)
	}
	return status, nil
}

func decodeOptionalUUIDv7(raw *json.RawMessage, name string) (scalar.UUIDv7, bool, error) {
	if raw == nil {
		return scalar.UUIDv7{}, false, nil
	}
	// Section 15.1 declares both identifiers optional and present when known. An
	// explicit JSON null states the same absence the omitted member states, and
	// encoding/json leaves the pointer above nil for both, so the two forms
	// converge on the absent answer before this point.
	trimmed := bytes.TrimSpace(*raw)
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return scalar.UUIDv7{}, false, fmt.Errorf("%w: %s is not a string", ErrInvalidStructuredError, name)
	}
	parsed, err := scalar.ParseUUIDv7(text)
	if err != nil {
		return scalar.UUIDv7{}, false, fmt.Errorf("%w: %s: %v", ErrInvalidStructuredError, name, err)
	}
	return parsed, true, nil
}

func decodeIDs(document *wireDocument) (IDs, error) {
	ids := NoIDs()
	operation, present, err := decodeOptionalUUIDv7(document.OperationID, "operation_id")
	if err != nil {
		return IDs{}, err
	}
	if present {
		ids = ids.WithOperation(operation)
	}
	session, present, err := decodeOptionalUUIDv7(document.SessionID, "session_id")
	if err != nil {
		return IDs{}, err
	}
	if present {
		ids = ids.WithSession(session)
	}
	return ids, nil
}
