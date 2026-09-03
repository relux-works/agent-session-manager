package cliresult

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// semverPattern accepts only a full, unpadded semantic version triple.
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// reverseDNSPattern is the Section 1.6 extension key grammar: "dot-separated
// labels matching [a-z][a-z0-9-]{0,62}" with at least one dot.
var reverseDNSPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$`)

const (
	maxExtensionKeys      = 64
	maxExtensionDepth     = 4
	maxExtensionCanonical = 65_536
)

// topLevelMembers is the closed Section 14.2 top-level member set: "both CLI
// Result versions contain exactly schema, schema_version, command, ok = true,
// operation_id (UUIDv7 or null), session_id (UUIDv7 or null), the tagged body,
// and extensions".
var topLevelMembers = []string{
	"schema", "schema_version", "command", "ok", "operation_id", "session_id", "body", "extensions",
}

// wireDocument is the closed decode target. Each member is kept as raw bytes so
// that an omitted member and an explicit JSON null stay distinguishable:
// Section 1.6 requires a T|null member to be present, and a decoder that
// collapsed the two would silently admit a document missing an identifier it
// must carry.
type wireDocument map[string]json.RawMessage

// Decode reads one CLI Result document that is expected to carry exactly
// version.
//
// The envelope identity is settled before any other member is consulted, the
// closed top-level member set included. A document whose schema is not
// urn:ax:schema:cli-result, whose major differs from the expected major, or
// whose version is unregistered is refused without this function having read
// its member set, command tag, identifiers, body, or extensions, so no part of
// a wrong-version payload is trusted on the way to that refusal.
//
// That ordering is a decision, not an accident, and the pinned document states
// it three times. Section 1.6's fail-closed rule is scoped to the object it
// governs - "a reader MUST reject an unknown top-level field in a major version
// 1 object" - and whether a document is a major version 1 object is exactly
// what the identity check settles. Section 17.1 scopes the same rule the same
// way: "within any negotiated major version, new object data MUST live under a
// namespaced extensions entry ... unknown top-level fields remain an error".
// And Section 17.2 lists "rejects an unsupported major" as the reader's first
// rule. Checking the member set first inverts all three: it answers a
// compatibility question - is this reader too old for this document - with a
// structural claim about a payload whose major was never established, and
// Section 15.1 is explicit that "receivers MUST NOT parse a different major's
// payload far enough to trust its error code, retryable bit, details, or
// authority fields".
//
// Both orders refuse, so this is not a bypass either way; what changes is which
// fact the caller is told. TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule
// pins it so a later change cannot reorder these two checks silently.
//
// ok = false is refused outright rather than returned as a failed result.
// Section 14.2 states that "failure output is one Structured Error object from
// Section 15.1, not a CLI Result with ok = false", so a document carrying that
// member is not a CLI Result at all.
//
// The takeover adoption rule is not checked here. It depends on the session
// kind, which the document does not carry; see Result.VerifyTakeoverAdoption.
func Decode(version Version, data []byte) (*Result, error) {
	if err := requireImplementedVersion(version); err != nil {
		return nil, err
	}
	canonical, err := canonicaljson.Canonicalize(data)
	if err != nil {
		return nil, failf("document is outside the common data model: %v", err)
	}
	document, err := parseDocument(canonical)
	if err != nil {
		return nil, err
	}
	if err := verifyEnvelopeIdentity(document, version); err != nil {
		return nil, err
	}
	if err := verifyClosedMembers(document); err != nil {
		return nil, err
	}
	return decodeBody(version, document)
}

// parseDocument reads the one JSON object on the wire without deciding anything
// about its members. It is deliberately separate from verifyClosedMembers: the
// closed member set is an obligation on a document of this contract's major,
// and this function runs before the major is known.
func parseDocument(data []byte) (wireDocument, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document wireDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, failf("decode: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, failf("document carries trailing content")
	}
	return document, nil
}

// verifyClosedMembers applies the Section 1.6 fail-closed rule to a document
// whose schema and major the caller has already settled: "a reader MUST reject
// an unknown top-level field in a major version 1 object. This fail-closed rule
// prevents silently ignoring a new ownership or security control."
func verifyClosedMembers(document wireDocument) error {
	declared := make(map[string]struct{}, len(topLevelMembers))
	for _, member := range topLevelMembers {
		declared[member] = struct{}{}
		if _, present := document[member]; !present {
			return failf("document is missing required member %q", member)
		}
	}
	names := make([]string, 0, len(document))
	for key := range document {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		if _, ok := declared[key]; !ok {
			return failf("document carries unknown top-level member %q", key)
		}
	}
	return nil
}

// rawString reads a required JSON string member from the raw document.
//
// Absence and a wrong type are reported separately. The identity check now runs
// before the closed member set is verified, so this is where a document missing
// schema or schema_version is first observed, and "the member is not there" and
// "the member is there and is not a string" are different facts about it.
func (document wireDocument) rawString(name string) (string, error) {
	raw, present := document[name]
	if !present {
		return "", failf("document is missing required member %q", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", failf("%s is not a JSON string", name)
	}
	return value, nil
}

func verifyEnvelopeIdentity(document wireDocument, expected Version) error {
	schema, err := document.rawString("schema")
	if err != nil {
		return err
	}
	if schema != Schema {
		return failf("schema is not %s", Schema)
	}
	raw, err := document.rawString("schema_version")
	if err != nil {
		return err
	}
	if !semverPattern.MatchString(raw) {
		return failf("schema_version %q is not a semantic version", raw)
	}
	candidate := Version(raw)
	expectedMajor, err := majorOf(expected)
	if err != nil {
		return err
	}
	candidateMajor, err := majorOf(candidate)
	if err != nil {
		return err
	}
	if candidateMajor != expectedMajor {
		return fmt.Errorf("%w: document is %s, this reader is bound to major %d", ErrUnsupportedMajor, raw, expectedMajor)
	}
	if !isRegisteredVersion(candidate) {
		return fmt.Errorf("%w: %q", ErrUnsupportedVersion, raw)
	}
	if !acceptsVersion(expected, candidate) {
		return fmt.Errorf("%w: document is %s, this reader accepts at most %s", ErrVersionMismatch, raw, expected)
	}
	return requireImplementedVersion(candidate)
}

// acceptsVersion implements Section 17.2 rule 2 for a reader that supports
// expected: it "accepts the same/lower supported minor" within the same major
// and refuses a higher one, because a higher minor may carry semantics this
// reader does not have.
//
// The pinned CLI Result registry carries no two versions sharing a major, so
// Decode reaches this function only with equal versions today. The comparison
// is therefore stated and tested as a function of its own rather than claimed
// through a code path the registry cannot currently exercise.
func acceptsVersion(expected, candidate Version) bool {
	expectedParts := semverPattern.FindStringSubmatch(string(expected))
	candidateParts := semverPattern.FindStringSubmatch(string(candidate))
	if expectedParts == nil || candidateParts == nil {
		return false
	}
	for index := 1; index <= 3; index++ {
		left, leftErr := strconv.Atoi(candidateParts[index])
		right, rightErr := strconv.Atoi(expectedParts[index])
		if leftErr != nil || rightErr != nil {
			return false
		}
		if left != right {
			return index != 1 && left < right
		}
	}
	return true
}

func majorOf(version Version) (int, error) {
	parts := semverPattern.FindStringSubmatch(string(version))
	if parts == nil {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
	major, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
	return major, nil
}

func decodeBody(version Version, document wireDocument) (*Result, error) {
	name, err := document.rawString("command")
	if err != nil {
		return nil, err
	}
	command := Command(name)
	selected, err := RegisteredVersionForCommand(command)
	if err != nil {
		return nil, err
	}
	// Section 14.2: "no command may emit another registered version or retry a
	// different major after parsing begins". A document whose tag selects a
	// different version is refused rather than reinterpreted under the tag's
	// own version, which would be exactly the retry the sentence forbids.
	if selected != version {
		return nil, failf("command %q selects CLI Result %s, the document is %s", command, selected, version)
	}
	var ok bool
	if err := json.Unmarshal(document["ok"], &ok); err != nil {
		return nil, failf("ok is not a JSON boolean")
	}
	if !ok {
		return nil, failf("ok is false; a failure is one Structured Error object, not a CLI Result")
	}
	body, err := decodeObjectMember(document, "body")
	if err != nil {
		return nil, err
	}
	extensions, err := decodeObjectMember(document, "extensions")
	if err != nil {
		return nil, err
	}
	ids, err := decodeIDs(document)
	if err != nil {
		return nil, err
	}
	// These maps were allocated by the decoder from the document bytes and no
	// caller holds a reference to them, so the result owns them already. New
	// rebuilds instead, because its input comes from a call site that keeps one.
	result := &Result{
		version:    version,
		command:    command,
		ids:        ids,
		body:       body,
		extensions: extensions,
	}
	if err := result.validate(""); err != nil {
		return nil, err
	}
	return result, nil
}

// decodeObjectMember reads a required object member with UseNumber, so every
// number in the body and the extensions map reaches the validators as the exact
// literal the document carried rather than as a host float.
func decodeObjectMember(document wireDocument, name string) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(document[name]))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, failf("decode %s: %v", name, err)
	}
	object, valid := decoded.(map[string]any)
	if !valid {
		return nil, failf("%s is not a JSON object", name)
	}
	return object, nil
}

func decodeIDs(document wireDocument) (IDs, error) {
	ids := NoIDs()
	operation, present, err := decodeRequiredUUIDv7OrNull(document["operation_id"], "operation_id")
	if err != nil {
		return IDs{}, err
	}
	if present {
		ids = ids.WithOperation(operation)
	}
	session, present, err := decodeRequiredUUIDv7OrNull(document["session_id"], "session_id")
	if err != nil {
		return IDs{}, err
	}
	if present {
		ids = ids.WithSession(session)
	}
	return ids, nil
}

// decodeRequiredUUIDv7OrNull reads a required UUIDv7|null member. Section 1.6
// says "a required T|null field MUST be present and MAY contain JSON null", so
// an omitted member is a refusal and is not treated as the null it resembles.
func decodeRequiredUUIDv7OrNull(raw json.RawMessage, name string) (scalar.UUIDv7, bool, error) {
	// This presence guard is subsumed by decodeClosedDocument, which already
	// refuses a document missing any of the eight declared top-level members,
	// so an absent raw member cannot reach here.
	// TestRequiredNullableIdentifiersMustBePresent pins that earlier refusal.
	if len(raw) == 0 {
		return scalar.UUIDv7{}, false, failf("%s is required and must be present as a UUIDv7 or null", name)
	}
	trimmed := bytes.TrimSpace(raw)
	if string(trimmed) == "null" {
		return scalar.UUIDv7{}, false, nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return scalar.UUIDv7{}, false, failf("%s is neither null nor a JSON string", name)
	}
	parsed, err := scalar.ParseUUIDv7(text)
	if err != nil {
		return scalar.UUIDv7{}, false, failf("%s: %v", name, err)
	}
	return parsed, true, nil
}

// validateExtensions checks the Section 1.6 extensions object: at most 64
// reverse-DNS keys of 3..253 lowercase ASCII characters, values inside the
// common data model at a maximum nesting depth of 4, and a complete canonical
// object of at most 65,536 bytes.
func validateExtensions(extensions map[string]any) error {
	if extensions == nil {
		return failf("extensions is required and must not be null")
	}
	if len(extensions) > maxExtensionKeys {
		return failf("extensions has %d members, the maximum is %d", len(extensions), maxExtensionKeys)
	}
	for _, key := range sortedKeys(extensions) {
		if len(key) < 3 || len(key) > 253 || !reverseDNSPattern.MatchString(key) {
			return failf("extensions key %q is not a 3..253 character lowercase reverse-DNS name", key)
		}
		if err := validateExtensionValue(fmt.Sprintf("extensions[%q]", key), extensions[key], 0); err != nil {
			return err
		}
	}
	measured, err := canonicaljson.CanonicalByteLength(extensions)
	if err != nil {
		return failf("measure canonical extensions: %v", err)
	}
	if measured > maxExtensionCanonical {
		return failf("canonical extensions object is %d bytes, the maximum is %d", measured, maxExtensionCanonical)
	}
	return nil
}

// validateExtensionValue admits exactly the ExtensionValue forms Section 1.6
// declares: "JSON null, boolean, a common-model integer, string, array, or
// string-keyed object with maximum nesting depth 4". A value may open four
// containers and the fifth is refused.
func validateExtensionValue(where string, value any, depth int) error {
	switch typed := value.(type) {
	case nil, bool, json.Number:
		return nil
	case string:
		if !utf8.ValidString(typed) {
			return failf("%s is not valid UTF-8", where)
		}
		return nil
	case []any:
		if depth == maxExtensionDepth {
			return failf("%s exceeds the maximum nesting depth %d", where, maxExtensionDepth)
		}
		for index, member := range typed {
			if err := validateExtensionValue(fmt.Sprintf("%s[%d]", where, index), member, depth+1); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		if depth == maxExtensionDepth {
			return failf("%s exceeds the maximum nesting depth %d", where, maxExtensionDepth)
		}
		for _, key := range sortedKeys(typed) {
			if err := validateExtensionValue(fmt.Sprintf("%s[%q]", where, key), typed[key], depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return failf("%s has type %T, which the common data model does not admit", where, value)
	}
}

func parsePlatform(value string) (scalar.Platform, error) {
	platform, err := scalar.ParsePlatform(value)
	if err != nil {
		return "", failf("platform %q is not macos, linux, wsl2, or windows", value)
	}
	return platform, nil
}

func parseAbsolutePath(platform scalar.Platform, value string) (scalar.AbsolutePath, error) {
	return scalar.ParseAbsolutePath(platform, value)
}
