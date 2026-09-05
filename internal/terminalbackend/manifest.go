// Manifest, Probe, and Capability Evidence admission for the
// TerminalBackend registry.
//
// Normative scope is relux-works/agent-session-manager-spec@v0.5.0 §4.B
// (Manifest/Probe closed schemas, generation digest, reconciliation,
// identity rule), §4.D (closed capability registry, claim rows, evidence
// schema, fact mapping), §6.5 (trust established before any probe;
// configuration admission), and §7.A (generation-bound descriptor binding).
// Lifecycle operations other than admission (create, attach, status, and
// friends) belong to the sibling lifecycle task; this file gates only which
// capabilities an operation may rely on, never the operations themselves.
//
// Fail-closed conventions shared with the registry half of this package:
// every refusal is an *Error with a wire code from the pinned catalog error
// vocabulary and a static Detail clause that never echoes paths, digests,
// generations, or other local data. A partial, malformed, stale, or failed
// Manifest, Probe, or evidence read is an error, never absence or
// availability, so callers must not fall back to a default or to PATH
// discovery.
package terminalbackend

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gowebpki/jcs"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Wire error codes this file may report. They extend the registry-half codes
// with the exact Section 15.3 codes for manifest/probe disagreement,
// unevidenced capability use, and integrity failure; every one is a member of
// the pinned catalog error vocabulary.
const (
	// CodeMismatch reports any failed Manifest/Probe equality, membership,
	// keyed claim relation, evidence binding, or evidence coverage rule
	// (§4.B). Malformed Manifest, Probe, or evidence documents also report
	// this code: a malformed read is an error, never absence.
	CodeMismatch = "terminal_backend_manifest_probe_mismatch"
	// CodeCapabilityUnproven reports an operation gate evaluated against an
	// admitted capability set that confers no true capability for the
	// requested operation (§4.C capability dependencies).
	CodeCapabilityUnproven = "terminal_backend_capability_unproven"
	// CodeIntegrityFailure reports a missing, malformed, unknown-key, or
	// invalid capability-evidence signature, or a signature verification
	// subsystem that is unavailable (§4.D). It fails before capability
	// admission.
	CodeIntegrityFailure = "terminal_backend_integrity_failure"
)

// IsMismatch reports CodeMismatch refusals.
func IsMismatch(err error) bool { return errorCode(err, CodeMismatch) }

// IsCapabilityUnproven reports CodeCapabilityUnproven refusals.
func IsCapabilityUnproven(err error) bool { return errorCode(err, CodeCapabilityUnproven) }

// IsIntegrityFailure reports CodeIntegrityFailure refusals.
func IsIntegrityFailure(err error) bool { return errorCode(err, CodeIntegrityFailure) }

// Closed schema literals (§4.B, §4.D). No other schema or version is
// admitted by this file.
const (
	// SchemaManifest is the Terminal Backend Manifest 1.0.0 schema URN.
	SchemaManifest = "urn:ax:schema:terminal-backend-manifest"
	// SchemaProbe is the Terminal Backend Probe 1.0.0 schema URN.
	SchemaProbe = "urn:ax:schema:terminal-backend-probe"
	// SchemaCapabilityEvidence is the Capability Evidence 1.0.0 schema URN.
	SchemaCapabilityEvidence = "urn:ax:schema:terminal-capability-evidence"
	// SchemaVersion100 is the only contract version this file implements.
	SchemaVersion100 = "1.0.0"
)

// Closed small vocabularies (§4.B, §4.D).
const (
	// OriginStatic marks a static manifest echo claim.
	OriginStatic = "static"
	// OriginProbed marks a live probe observation claim.
	OriginProbed = "probed"

	// AvailabilityAvailable marks a ready backend.
	AvailabilityAvailable = "available"
	// AvailabilityConditional marks a backend that needs operator action.
	AvailabilityConditional = "conditional"
	// AvailabilityUnavailable marks a backend that cannot serve.
	AvailabilityUnavailable = "unavailable"
	// AvailabilityUnknown marks a backend whose readiness is not established.
	AvailabilityUnknown = "unknown"

	// IssuerRelease marks release-signed capability evidence.
	IssuerRelease = "ax_release"
	// IssuerLocalProbe marks host-incarnation-signed capability evidence.
	IssuerLocalProbe = "ax_local_probe"
)

// signatureSchemePrefix is the literal attestation_signature scheme (§4.D).
const signatureSchemePrefix = "rsa-sha256:"

// generationDigestDomain is the UTF-8 domain separator prepended to the
// backend's host-local raw generation bytes before SHA-256 (§4.B). It
// neither reveals nor reconstructs that generation.
const generationDigestDomain = "ax-terminal-backend-generation-v1\x00"

// evidenceSignatureDomain is the ASCII domain separator prepended to the
// canonical evidence bytes before signing (§4.D).
const evidenceSignatureDomain = "ax-terminal-capability-evidence-v1"

// Closed TerminalBackendCapability registry (§4.D). The map key is the
// capability name; the row carries the exact registry members a Manifest or
// Probe claim must repeat verbatim.
type capabilityRow struct {
	generationVariable   bool
	dependentOperations  []string
	evidenceRequirements []string
}

// capabilityRegistry is the exact 16-value Section 4.D table. Dependent
// operations and evidence requirements are stored sorted as the wire
// requires, so row comparison is element-wise.
var capabilityRegistry = map[string]capabilityRow{
	"durable_disconnect": {
		generationVariable:   false,
		dependentOperations:  []string{"create", "status"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"local_attach": {
		generationVariable:   true,
		dependentOperations:  []string{"attach"},
		evidenceRequirements: []string{"conformance_fixture", "policy_authorization", "runtime_probe"},
	},
	"remote_attach": {
		generationVariable:   true,
		dependentOperations:  []string{"attach"},
		evidenceRequirements: []string{"conformance_fixture", "policy_authorization", "runtime_probe"},
	},
	"web_attach": {
		generationVariable:   true,
		dependentOperations:  []string{"attach"},
		evidenceRequirements: []string{"conformance_fixture", "policy_authorization", "runtime_probe"},
	},
	"multi_attach": {
		generationVariable:   true,
		dependentOperations:  []string{"attach"},
		evidenceRequirements: []string{"conformance_fixture", "policy_authorization", "runtime_probe"},
	},
	"headless_creation": {
		generationVariable:   true,
		dependentOperations:  []string{"create"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"reboot_restoration": {
		generationVariable:   true,
		dependentOperations:  []string{"restore"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"input_quiescence": {
		generationVariable:   true,
		dependentOperations:  []string{"quiesce-input"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"safe_boundary_observation": {
		generationVariable:   true,
		dependentOperations:  []string{"wait-safe-boundary"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"provider_process_observation": {
		generationVariable:   true,
		dependentOperations:  []string{"status", "terminate-stale", "wait-safe-boundary"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"graceful_stop": {
		generationVariable:   true,
		dependentOperations:  []string{"request-stop"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"stale_process_termination": {
		generationVariable:   true,
		dependentOperations:  []string{"terminate-stale"},
		evidenceRequirements: []string{"conformance_fixture", "policy_authorization", "runtime_probe"},
	},
	"terminal_state_retention": {
		generationVariable:   true,
		dependentOperations:  []string{"create", "restore", "status"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"scrollback_retention": {
		generationVariable:   true,
		dependentOperations:  []string{"create", "restore", "status"},
		evidenceRequirements: []string{"conformance_fixture", "runtime_probe"},
	},
	"credential_capable_execution_realm": {
		generationVariable:  true,
		dependentOperations: []string{"create", "restore"},
		evidenceRequirements: []string{
			"conformance_fixture", "credential_sentinel", "provider_auth_smoke", "runtime_probe",
		},
	},
	"multiple_input_clients": {
		generationVariable:   true,
		dependentOperations:  []string{"attach"},
		evidenceRequirements: []string{"conformance_fixture", "policy_authorization", "runtime_probe"},
	},
}

// credentialRealmCapability is the one capability whose evidence carries the
// terminal-binding, provider, sentinel, and smoke members (§4.D). Every other
// capability's evidence must leave those members null.
const credentialRealmCapability = "credential_capable_execution_realm"

// requirementFact maps every CapabilityEvidenceRequirement to the
// CapabilityEvidenceFact that satisfies it (§4.D). ui_absent and
// prompt_absent are supplementary facts with no requirement mapping: they
// may accompany evidence but never satisfy a requirement alone.
var requirementFact = map[string]string{
	"conformance_fixture":  "fixture_passed",
	"runtime_probe":        "runtime_probe_passed",
	"credential_sentinel":  "sentinel_passed",
	"provider_auth_smoke":  "provider_auth_passed",
	"policy_authorization": "policy_checked",
}

// Closed TerminalBackendOperation vocabulary (§4.D).
var operationVocabulary = map[string]bool{
	"manifest": true, "probe": true, "create": true, "attach": true,
	"status": true, "quiesce-input": true, "wait-safe-boundary": true,
	"request-stop": true, "terminate-stale": true, "restore": true,
}

// Closed CapabilityEvidenceRequirement vocabulary (§4.D).
var evidenceRequirementVocabulary = map[string]bool{
	"conformance_fixture": true, "runtime_probe": true,
	"credential_sentinel": true, "provider_auth_smoke": true,
	"policy_authorization": true,
}

// Closed CapabilityEvidenceFact vocabulary (§4.D).
var evidenceFactVocabulary = map[string]bool{
	"fixture_passed": true, "runtime_probe_passed": true,
	"sentinel_passed": true, "provider_auth_passed": true,
	"policy_checked": true, "ui_absent": true, "prompt_absent": true,
}

// Document bounds shared by the three closed schemas.
const (
	maxClaims          = 16
	maxEvidenceIDs     = 256
	maxOperations      = 10
	maxRequirements    = 5
	maxFacts           = 7
	maxOpaqueString    = 256
	maxJSONDepth       = 32
	maxNativeReference = 512
)

// mismatchf refuses with CodeMismatch and a static detail clause. The clause
// must never interpolate local data: paths, digests, generations, and
// document bytes stay out of errors.
func mismatchf(format string, arguments ...any) *Error {
	return &Error{Code: CodeMismatch, Detail: fmt.Sprintf(format, arguments...)}
}

// hasLoneSurrogateEscape reports whether raw JSON carries a lone surrogate
// code point inside any string literal, either as a \uXXXX escape or as raw
// WTF-8 (CESU-8) bytes ED A0..BF 80..BF, which decode to U+D800..U+DFFF.
// It answers only the surrogate question: malformed escapes, other invalid
// UTF-8, unescaped control characters, and unterminated strings are left
// for the decoder's syntax and encoding arms, which refuse them with their
// own details. The scan mirrors internal/canonicaljson
// validateSurrogateEscapes without importing it: canonicaljson is
// byte-identical across stories and must not be edited at integration, so
// the two copies are pinned equal by
// TestSurrogateGateAgreesWithCanonicalJSON instead of shared.
func hasLoneSurrogateEscape(raw []byte) bool {
	for index := 0; index < len(raw); index++ {
		if raw[index] != '"' {
			continue
		}
		for index++; index < len(raw); index++ {
			if isWTF8SurrogateAt(raw, index) {
				return true
			}
			switch raw[index] {
			case '"':
				goto stringComplete
			case '\\':
				index++
				if index >= len(raw) {
					return false
				}
				// The escaped byte is skipped by the switch below without
				// passing the loop head, so test it here: a WTF-8 head
				// immediately after a backslash (for example "...\n<ED A0
				// 80>...") is still raw bytes on the wire, not an escape.
				if isWTF8SurrogateAt(raw, index) {
					return true
				}
				if raw[index] != 'u' {
					continue
				}
				unit, end, ok := readUTF16EscapeUnit(raw, index-1)
				if !ok {
					continue
				}
				index = end - 1
				switch {
				case unit >= 0xd800 && unit <= 0xdbff:
					second, secondEnd, ok := readUTF16EscapeUnit(raw, end)
					if !ok || second < 0xdc00 || second > 0xdfff {
						return true
					}
					index = secondEnd - 1
				case unit >= 0xdc00 && unit <= 0xdfff:
					return true
				}
			}
		}
	stringComplete:
	}
	return false
}

// isWTF8SurrogateAt reports whether raw[index] begins a three-byte WTF-8
// (CESU-8) encoding of a surrogate code point: ED A0..BF 80..BF decodes to
// U+D800..U+DFFF, which utf8.Valid refuses. Only this ED range is
// surrogate-shaped: ED 80..9F heads U+D000..U+D7FF, which is valid UTF-8
// and must not match. A truncated tail cannot be classified here and is
// left for the encoding arm.
func isWTF8SurrogateAt(raw []byte, index int) bool {
	return index+2 < len(raw) && raw[index] == 0xed &&
		raw[index+1] >= 0xa0 && raw[index+1] <= 0xbf &&
		raw[index+2] >= 0x80 && raw[index+2] <= 0xbf
}

// readUTF16EscapeUnit reads one \uXXXX escape at raw[start] (which must be
// the backslash). It reports false when the escape is absent or malformed;
// malformed escapes are refused by the decoder's syntax arm, so this gate
// only needs to stay silent on them rather than diagnose them.
func readUTF16EscapeUnit(raw []byte, start int) (uint16, int, bool) {
	if start < 0 || start+6 > len(raw) || raw[start] != '\\' || raw[start+1] != 'u' {
		return 0, start, false
	}
	var unit uint16
	for _, digit := range raw[start+2 : start+6] {
		unit <<= 4
		switch {
		case digit >= '0' && digit <= '9':
			unit |= uint16(digit - '0')
		case digit >= 'a' && digit <= 'f':
			unit |= uint16(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			unit |= uint16(digit-'A') + 10
		default:
			return 0, start, false
		}
	}
	return unit, start + 6, true
}

// decodeStrictObject decodes one JSON object with duplicate-key refusal,
// invalid-UTF-8 refusal, lone-surrogate-escape refusal, a nesting cap, and
// trailing-data refusal. The surrogate scan runs on the raw bytes before
// any canonicalization (SPEC.md:289): Go's encoding/json would silently
// replace a lone \ud800 with U+FFFD, hiding the violation from every later
// member check. Numbers
// decode as json.Number so member validators can refuse them explicitly:
// none of the three closed schemas has a numeric member.
func decodeStrictObject(raw []byte) (map[string]any, error) {
	if !utf8.Valid(raw) {
		return nil, mismatchf("document encoding")
	}
	if hasLoneSurrogateEscape(raw) {
		return nil, mismatchf("document surrogate escape")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeCappedValue(decoder, 0)
	if err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, mismatchf("document trailing data")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, mismatchf("document shape")
	}
	return object, nil
}

// decodeCappedValue decodes one JSON value with an explicit depth bound. The
// bound is far above the deepest legitimate nesting (object, claims array,
// claim object, member array) and fails closed below any decoder-stack risk.
func decodeCappedValue(decoder *json.Decoder, depth int) (any, error) {
	if depth > maxJSONDepth {
		return nil, mismatchf("document nesting")
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, mismatchf("document syntax")
	}
	if delimiter, ok := token.(json.Delim); ok {
		switch delimiter {
		case '{':
			object := make(map[string]any)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, mismatchf("document syntax")
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, mismatchf("document syntax")
				}
				if _, duplicate := object[key]; duplicate {
					return nil, mismatchf("document duplicate member")
				}
				member, err := decodeCappedValue(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				object[key] = member
			}
			if _, err := decoder.Token(); err != nil {
				return nil, mismatchf("document syntax")
			}
			return object, nil
		case '[':
			// Non-nil initialization: an empty wire array must round-trip
			// as [] rather than null, or identity recomputation over an
			// empty claim or evidence list diverges from the signer.
			array := []any{}
			for decoder.More() {
				member, err := decodeCappedValue(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				array = append(array, member)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, mismatchf("document syntax")
			}
			return array, nil
		default:
			return nil, mismatchf("document syntax")
		}
	}
	return token, nil
}

// checkExactMembers refuses any deviation from the closed member set: an
// unknown member and a missing member are both malformed.
func checkExactMembers(object map[string]any, members []string) error {
	if len(object) != len(members) {
		return mismatchf("document members")
	}
	for _, member := range members {
		if _, known := object[member]; !known {
			return mismatchf("document members")
		}
	}
	return nil
}

// stringMember extracts a required string member. Numbers, booleans, nulls,
// arrays, and objects are refused: no member of these schemas takes them.
func stringMember(object map[string]any, name string) (string, error) {
	value, ok := object[name].(string)
	if !ok {
		return "", mismatchf("document member type")
	}
	return value, nil
}

// boundedStringMember extracts a required string[1..256] measured in UTF-8
// characters over valid UTF-8 (SPEC.md:321), the canonical bound in every
// containing contract.
func boundedStringMember(object map[string]any, name string) (string, error) {
	value, err := stringMember(object, name)
	if err != nil {
		return "", err
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > maxOpaqueString {
		return "", mismatchf("document string bound")
	}
	return value, nil
}

// digestMember extracts a required sha256: digest member.
func digestMember(object map[string]any, name string) (string, error) {
	value, err := stringMember(object, name)
	if err != nil {
		return "", err
	}
	if _, err := scalar.ParseDigest(value); err != nil {
		return "", mismatchf("document digest")
	}
	return value, nil
}

// digestOrNullMember extracts a digest|null member: present and either a
// valid digest or JSON null. Absence is malformed, not null.
func digestOrNullMember(object map[string]any, name string) (string, error) {
	raw, known := object[name]
	if !known {
		return "", mismatchf("document members")
	}
	if raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", mismatchf("document member type")
	}
	if _, err := scalar.ParseDigest(value); err != nil {
		return "", mismatchf("document digest")
	}
	return value, nil
}

// semverMember extracts a required semver member.
func semverMember(object map[string]any, name string) (string, error) {
	value, err := stringMember(object, name)
	if err != nil {
		return "", err
	}
	if !semverPattern.MatchString(value) {
		return "", mismatchf("document semver")
	}
	return value, nil
}

// timestampMember extracts a required timestamp member.
func timestampMember(object map[string]any, name string) (scalar.Timestamp, error) {
	value, err := stringMember(object, name)
	if err != nil {
		return scalar.Timestamp{}, err
	}
	parsed, err := scalar.ParseTimestamp(value)
	if err != nil {
		return scalar.Timestamp{}, mismatchf("document timestamp")
	}
	return parsed, nil
}

// checkExtensions admits exactly the empty object {}. A non-empty extensions
// object requires a later contract version, and absence is malformed.
func checkExtensions(object map[string]any) error {
	raw, known := object["extensions"]
	if !known {
		return mismatchf("document members")
	}
	extensions, ok := raw.(map[string]any)
	if !ok || len(extensions) != 0 {
		return mismatchf("document extensions")
	}
	return nil
}

// stringArrayMember extracts a required array of strings, refusing numbers,
// booleans, nulls, and nested values element-wise.
func stringArrayMember(object map[string]any, name string) ([]string, error) {
	raw, known := object[name]
	if !known {
		return nil, mismatchf("document members")
	}
	array, ok := raw.([]any)
	if !ok {
		return nil, mismatchf("document member type")
	}
	values := make([]string, 0, len(array))
	for _, member := range array {
		value, ok := member.(string)
		if !ok {
			return nil, mismatchf("document member type")
		}
		values = append(values, value)
	}
	return values, nil
}

// checkSortedUnique refuses an unsorted or duplicated string list. Bytewise
// order matches the wire's sorted-unique rule over lowercase ASCII members.
func checkSortedUnique(values []string) error {
	for index := range values {
		if index > 0 && values[index-1] >= values[index] {
			return mismatchf("document ordering")
		}
	}
	return nil
}

// checkClosedList enforces the sorted-unique closed-vocabulary list shape:
// every member in vocabulary, sorted unique, within the inclusive item
// bound. The bound admits zero so [0..N] lists stay usable; [1..N] lists
// are enforced by requiring at least one member.
func checkClosedList(values []string, vocabulary map[string]bool, max int, requireNonEmpty bool) error {
	if len(values) > max || (requireNonEmpty && len(values) == 0) {
		return mismatchf("document list bound")
	}
	for _, value := range values {
		if !vocabulary[value] {
			return mismatchf("document vocabulary")
		}
	}
	return checkSortedUnique(values)
}

// objectIdentity computes the lowercase sha256: digest of the RFC 8785 JCS
// bytes of object with exactly selfField omitted (§4.B identity rule). It
// mirrors the repository's canonical pipeline (logical value, JSON
// re-encode, JCS transform, SHA-256) without routing through the schema
// validators owned by another task.
func objectIdentity(object map[string]any, selfField string) (string, error) {
	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != selfField {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		return "", mismatchf("document identity")
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		return "", mismatchf("document identity")
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// checkIdentity recomputes the omit-self digest and refuses a mismatch. A
// reader must recompute the ID before use; trusting the claimed member is a
// substitution vector.
func checkIdentity(object map[string]any, selfField, claimed string) error {
	if _, err := scalar.ParseDigest(claimed); err != nil {
		return mismatchf("document digest")
	}
	computed, err := objectIdentity(object, selfField)
	if err != nil {
		return err
	}
	if computed != claimed {
		return mismatchf("document identity binding")
	}
	return nil
}

// Claim is one TerminalBackendCapabilityClaim: a closed object whose
// generation_variable, dependent_operations, and evidence_requirements
// members must exactly equal the capability registry row (§4.D). A false
// claim makes a probed refusal explicit but confers no operation.
type Claim struct {
	// Capability is one of the 16 closed registry names.
	Capability string
	// Origin is static (manifest echo) or probed (live observation).
	Origin string
	// Value is the claimed capability state.
	Value bool
	// GenerationVariable repeats the registry row: whether the value may
	// change across backend generations.
	GenerationVariable bool
	// DependentOperations repeats the registry row: sorted unique
	// operations this capability confers.
	DependentOperations []string
	// EvidenceRequirements repeats the registry row: sorted unique
	// evidence requirements that prove a true claim.
	EvidenceRequirements []string
}

// parseClaim validates one claim object. allowProbed selects the Probe shape;
// the Manifest shape admits only static origin.
func parseClaim(raw any, allowProbed bool) (Claim, error) {
	object, ok := raw.(map[string]any)
	if !ok {
		return Claim{}, mismatchf("claim shape")
	}
	if err := checkExactMembers(object, []string{
		"capability", "origin", "value",
		"generation_variable", "dependent_operations", "evidence_requirements",
	}); err != nil {
		return Claim{}, err
	}
	capability, err := stringMember(object, "capability")
	if err != nil {
		return Claim{}, err
	}
	row, known := capabilityRegistry[capability]
	if !known {
		return Claim{}, mismatchf("capability vocabulary")
	}
	origin, err := stringMember(object, "origin")
	if err != nil {
		return Claim{}, err
	}
	if origin != OriginStatic && (origin != OriginProbed || !allowProbed) {
		return Claim{}, mismatchf("claim origin")
	}
	boolean, ok := object["value"].(bool)
	if !ok {
		return Claim{}, mismatchf("document member type")
	}
	generationVariable, ok := object["generation_variable"].(bool)
	if !ok {
		return Claim{}, mismatchf("document member type")
	}
	operations, err := stringArrayMember(object, "dependent_operations")
	if err != nil {
		return Claim{}, err
	}
	if err := checkClosedList(operations, operationVocabulary, maxOperations, true); err != nil {
		return Claim{}, err
	}
	requirements, err := stringArrayMember(object, "evidence_requirements")
	if err != nil {
		return Claim{}, err
	}
	if err := checkClosedList(requirements, evidenceRequirementVocabulary, maxRequirements, true); err != nil {
		return Claim{}, err
	}
	// The three registry-derived members must equal the row: a Manifest or
	// Probe cannot redefine what a capability means or what proves it.
	if generationVariable != row.generationVariable ||
		!equalStrings(operations, row.dependentOperations) ||
		!equalStrings(requirements, row.evidenceRequirements) {
		return Claim{}, mismatchf("capability registry binding")
	}
	return Claim{
		Capability:           capability,
		Origin:               origin,
		Value:                boolean,
		GenerationVariable:   generationVariable,
		DependentOperations:  operations,
		EvidenceRequirements: requirements,
	}, nil
}

// parseClaimList validates a sorted-by-capability unique claim array within
// the [0..16] bound. Sorting is checked here so reconciliation can treat
// the key order as established; duplicates are invalid before keyed
// reconciliation runs.
func parseClaimList(raw any, allowProbed bool) ([]Claim, error) {
	array, ok := raw.([]any)
	if !ok {
		return nil, mismatchf("document member type")
	}
	if len(array) > maxClaims {
		return nil, mismatchf("claim list bound")
	}
	claims := make([]Claim, 0, len(array))
	previous := ""
	for _, member := range array {
		claim, err := parseClaim(member, allowProbed)
		if err != nil {
			return nil, err
		}
		if previous >= claim.Capability {
			return nil, mismatchf("claim ordering")
		}
		previous = claim.Capability
		claims = append(claims, claim)
	}
	return claims, nil
}

// equalClaim reports member-for-member equality of two claims.
func equalClaim(left, right Claim) bool {
	return left.Capability == right.Capability &&
		left.Origin == right.Origin &&
		left.Value == right.Value &&
		left.GenerationVariable == right.GenerationVariable &&
		equalStrings(left.DependentOperations, right.DependentOperations) &&
		equalStrings(left.EvidenceRequirements, right.EvidenceRequirements)
}

// equalStrings reports element-wise equality of two string lists.
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// Manifest is one validated Terminal Backend Manifest 1.0.0: the closed
// immutable backend description. It is not availability proof, and no
// unsupported capability may be advertised through it.
type Manifest struct {
	// ManifestID is the recomputed omit-self digest.
	ManifestID string
	// TerminalBackendID is the canonical backend identity.
	TerminalBackendID string
	// ImplementationVersion is the exact admitted semver.
	ImplementationVersion string
	// ProtocolVersions are the sorted unique semver members in Terminal
	// Backend Protocol major 1.
	ProtocolVersions []string
	// Platforms is the sorted unique non-empty platform subset.
	Platforms []scalar.Platform
	// ImplementationKind is the closed implementation_kind.
	ImplementationKind Kind
	// ExecutableDigest is the sha256: digest for external kinds and empty
	// (null) for built-in kinds.
	ExecutableDigest string
	// StaticCapabilityClaims are the sorted-by-capability unique static
	// claims, each exactly repeating its registry row.
	StaticCapabilityClaims []Claim
	// ConformanceFixtureID names the fixture true claims are proven with.
	ConformanceFixtureID string
}

// Probe is one validated Terminal Backend Probe 1.0.0: the closed immutable
// observation of a backend on one platform, generation, and instant.
type Probe struct {
	// ProbeID is the recomputed omit-self digest.
	ProbeID string
	// TerminalBackendID equals the admitted Manifest member.
	TerminalBackendID string
	// ImplementationVersion equals the admitted Manifest member.
	ImplementationVersion string
	// ProtocolVersion is exactly one member of the Manifest list.
	ProtocolVersion string
	// ImplementationKind equals the Manifest member.
	ImplementationKind Kind
	// ExecutableDigest exactly equals the Manifest member.
	ExecutableDigest string
	// Platform is exactly one member of the Manifest platforms.
	Platform scalar.Platform
	// OSVersion is the string[1..256] host release.
	OSVersion string
	// Availability is the closed readiness vocabulary.
	Availability string
	// BackendGenerationDigest is the digest of the domain-separated raw
	// generation. It neither reveals nor reconstructs that generation.
	BackendGenerationDigest string
	// CapabilityClaims are the sorted-by-capability unique reconciled
	// claims.
	CapabilityClaims []Claim
	// EvidenceIDs are the sorted unique IDs of the evidence objects used
	// by true claims.
	EvidenceIDs []string
	// ProbedAt is the observation instant.
	ProbedAt scalar.Timestamp
}

// Evidence is one validated Capability Evidence 1.0.0 object: the signed,
// expiring, generation-bound proof of one true capability claim. Value is
// always true by construction: a false claim has no matching evidence.
type Evidence struct {
	// EvidenceID is the recomputed omit-self digest.
	EvidenceID string
	// TerminalBackendID equals the Manifest and Probe members.
	TerminalBackendID string
	// ImplementationVersion equals the Manifest and Probe members.
	ImplementationVersion string
	// ProtocolVersion equals the Probe member.
	ProtocolVersion string
	// BackendGenerationDigest equals the Probe member.
	BackendGenerationDigest string
	// Capability names the one true claim this object proves.
	Capability string
	// Platform equals the Probe member.
	Platform scalar.Platform
	// OSVersion equals the Probe member.
	OSVersion string
	// ConformanceFixtureID equals the Manifest member.
	ConformanceFixtureID string
	// ObservedAt opens the validity interval.
	ObservedAt scalar.Timestamp
	// ExpiresAt closes the validity interval, strictly after ObservedAt.
	ExpiresAt scalar.Timestamp
	// Issuer is ax_release or ax_local_probe.
	Issuer string
	// IssuerID is the digest of the trusted signing identity. A
	// caller-supplied issuer ID is not authentication: the signature must
	// verify through the trusted key registry.
	IssuerID string
	// AttestationSignature is the raw signature bytes (scheme stripped).
	AttestationSignature []byte
	// Facts are the sorted unique evidence facts covering the claim's
	// requirements through the requirement-to-fact mapping.
	Facts []string
	// TerminalBindingID is the opaque binding digest, required only for
	// credential_capable_execution_realm and empty otherwise.
	TerminalBindingID string
	// ProviderID is the provider-id, required only for
	// credential_capable_execution_realm and empty otherwise.
	ProviderID string
	// ProviderBuild is the string[1..256] provider build, required only
	// for credential_capable_execution_realm and empty otherwise.
	ProviderBuild string
	// SentinelResult is literal passed, required only for
	// credential_capable_execution_realm and empty otherwise.
	SentinelResult string
	// ProviderAuthSmokeResult is literal passed, required only for
	// credential_capable_execution_realm and empty otherwise.
	ProviderAuthSmokeResult string
}

// manifestMembers is the exact closed Manifest member set.
var manifestMembers = []string{
	"schema", "schema_version", "manifest_id", "terminal_backend_id",
	"implementation_version", "protocol_versions", "platforms",
	"implementation_kind", "executable_digest", "static_capability_claims",
	"conformance_fixture_id", "extensions",
}

// ParseManifest validates one Terminal Backend Manifest 1.0.0 document and
// returns the typed observation. Any deviation from the closed shape,
// vocabulary, bound, ordering, registry-row, or identity rule is
// CodeMismatch: a malformed read is an error, never absence.
func ParseManifest(raw []byte) (Manifest, error) {
	object, err := decodeStrictObject(raw)
	if err != nil {
		return Manifest{}, err
	}
	return parseManifestObject(object)
}

// parseManifestObject validates a decoded Manifest member map. It is split
// from ParseManifest so admission paths that already hold a decoded object
// share the exact gate.
func parseManifestObject(object map[string]any) (Manifest, error) {
	if err := checkExactMembers(object, manifestMembers); err != nil {
		return Manifest{}, err
	}
	if schema, err := stringMember(object, "schema"); err != nil || schema != SchemaManifest {
		return Manifest{}, mismatchf("manifest schema")
	}
	if version, err := stringMember(object, "schema_version"); err != nil || version != SchemaVersion100 {
		return Manifest{}, mismatchf("manifest schema version")
	}
	manifestID, err := stringMember(object, "manifest_id")
	if err != nil {
		return Manifest{}, err
	}
	if err := checkIdentity(object, "manifest_id", manifestID); err != nil {
		return Manifest{}, err
	}
	backendID, err := stringMember(object, "terminal_backend_id")
	if err != nil {
		return Manifest{}, err
	}
	if _, err := ParseID(backendID); err != nil {
		return Manifest{}, mismatchf("manifest backend identity")
	}
	implementationVersion, err := semverMember(object, "implementation_version")
	if err != nil {
		return Manifest{}, err
	}
	protocols, err := stringArrayMember(object, "protocol_versions")
	if err != nil {
		return Manifest{}, err
	}
	if err := validateProtocolList(protocols); err != nil {
		return Manifest{}, err
	}
	platforms, err := parsePlatformList(object)
	if err != nil {
		return Manifest{}, err
	}
	kindName, err := stringMember(object, "implementation_kind")
	if err != nil {
		return Manifest{}, err
	}
	kind, err := parseKind(kindName)
	if err != nil {
		return Manifest{}, mismatchf("manifest implementation kind")
	}
	digest, err := digestOrNullMember(object, "executable_digest")
	if err != nil {
		return Manifest{}, err
	}
	if needsDigest(kind) == (digest == "") {
		return Manifest{}, mismatchf("manifest executable digest")
	}
	claimsRaw, known := object["static_capability_claims"]
	if !known {
		return Manifest{}, mismatchf("document members")
	}
	claims, err := parseClaimList(claimsRaw, false)
	if err != nil {
		return Manifest{}, err
	}
	fixtureID, err := digestMember(object, "conformance_fixture_id")
	if err != nil {
		return Manifest{}, err
	}
	if err := checkExtensions(object); err != nil {
		return Manifest{}, err
	}
	return Manifest{
		ManifestID:             manifestID,
		TerminalBackendID:      backendID,
		ImplementationVersion:  implementationVersion,
		ProtocolVersions:       protocols,
		Platforms:              platforms,
		ImplementationKind:     kind,
		ExecutableDigest:       digest,
		StaticCapabilityClaims: claims,
		ConformanceFixtureID:   fixtureID,
	}, nil
}

// validateProtocolList enforces sorted unique semver[1..32], each in
// Terminal Backend Protocol major 1, without naming a backend.
func validateProtocolList(versions []string) error {
	if len(versions) < 1 || len(versions) > maxProtocolVersions {
		return mismatchf("protocol versions bound")
	}
	for index, version := range versions {
		if !semverPattern.MatchString(version) || semverMajor(version) != 1 {
			return mismatchf("protocol versions major 1")
		}
		if index > 0 && versions[index-1] >= version {
			return mismatchf("protocol versions ordering")
		}
	}
	return nil
}

// parsePlatformList validates the sorted unique non-empty platform subset.
func parsePlatformList(object map[string]any) ([]scalar.Platform, error) {
	names, err := stringArrayMember(object, "platforms")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 || len(names) > 4 {
		return nil, mismatchf("platforms bound")
	}
	platforms := make([]scalar.Platform, 0, len(names))
	for _, name := range names {
		platform, err := scalar.ParsePlatform(name)
		if err != nil {
			return nil, mismatchf("platforms vocabulary")
		}
		platforms = append(platforms, platform)
	}
	for index := range platforms {
		// Bytewise >= refuses duplicates and misordering together: equal
		// neighbours are not strictly ordered.
		if index > 0 && platforms[index-1].String() >= platforms[index].String() {
			return nil, mismatchf("platforms ordering")
		}
	}
	return platforms, nil
}

// probeMembers is the exact closed Probe member set.
var probeMembers = []string{
	"schema", "schema_version", "probe_id", "terminal_backend_id",
	"implementation_version", "protocol_version", "implementation_kind",
	"executable_digest", "platform", "os_version", "availability",
	"backend_generation_digest", "capability_claims", "evidence_ids",
	"probed_at", "extensions",
}

// ParseProbe validates one Terminal Backend Probe 1.0.0 document. Shape,
// vocabulary, bound, ordering, registry-row, and identity failures are all
// CodeMismatch; cross-document reconciliation against the Manifest happens
// in Reconcile, not here.
func ParseProbe(raw []byte) (Probe, error) {
	object, err := decodeStrictObject(raw)
	if err != nil {
		return Probe{}, err
	}
	if err := checkExactMembers(object, probeMembers); err != nil {
		return Probe{}, err
	}
	if schema, err := stringMember(object, "schema"); err != nil || schema != SchemaProbe {
		return Probe{}, mismatchf("probe schema")
	}
	if version, err := stringMember(object, "schema_version"); err != nil || version != SchemaVersion100 {
		return Probe{}, mismatchf("probe schema version")
	}
	probeID, err := stringMember(object, "probe_id")
	if err != nil {
		return Probe{}, err
	}
	if err := checkIdentity(object, "probe_id", probeID); err != nil {
		return Probe{}, err
	}
	backendID, err := stringMember(object, "terminal_backend_id")
	if err != nil {
		return Probe{}, err
	}
	if _, err := ParseID(backendID); err != nil {
		return Probe{}, mismatchf("probe backend identity")
	}
	implementationVersion, err := semverMember(object, "implementation_version")
	if err != nil {
		return Probe{}, err
	}
	protocolVersion, err := semverMember(object, "protocol_version")
	if err != nil {
		return Probe{}, err
	}
	if semverMajor(protocolVersion) != 1 {
		return Probe{}, mismatchf("probe protocol major 1")
	}
	kindName, err := stringMember(object, "implementation_kind")
	if err != nil {
		return Probe{}, err
	}
	kind, err := parseKind(kindName)
	if err != nil {
		return Probe{}, mismatchf("probe implementation kind")
	}
	digest, err := digestOrNullMember(object, "executable_digest")
	if err != nil {
		return Probe{}, err
	}
	if needsDigest(kind) == (digest == "") {
		return Probe{}, mismatchf("probe executable digest")
	}
	platformName, err := stringMember(object, "platform")
	if err != nil {
		return Probe{}, err
	}
	platform, err := scalar.ParsePlatform(platformName)
	if err != nil {
		return Probe{}, mismatchf("probe platform")
	}
	osVersion, err := boundedStringMember(object, "os_version")
	if err != nil {
		return Probe{}, err
	}
	availability, err := stringMember(object, "availability")
	if err != nil {
		return Probe{}, err
	}
	switch availability {
	case AvailabilityAvailable, AvailabilityConditional, AvailabilityUnavailable, AvailabilityUnknown:
	default:
		return Probe{}, mismatchf("probe availability")
	}
	generationDigest, err := digestMember(object, "backend_generation_digest")
	if err != nil {
		return Probe{}, err
	}
	claimsRaw, known := object["capability_claims"]
	if !known {
		return Probe{}, mismatchf("document members")
	}
	claims, err := parseClaimList(claimsRaw, true)
	if err != nil {
		return Probe{}, err
	}
	evidenceIDs, err := stringArrayMember(object, "evidence_ids")
	if err != nil {
		return Probe{}, err
	}
	if len(evidenceIDs) > maxEvidenceIDs {
		return Probe{}, mismatchf("evidence list bound")
	}
	for _, id := range evidenceIDs {
		if _, err := scalar.ParseDigest(id); err != nil {
			return Probe{}, mismatchf("document digest")
		}
	}
	if err := checkSortedUnique(evidenceIDs); err != nil {
		return Probe{}, err
	}
	probedAt, err := timestampMember(object, "probed_at")
	if err != nil {
		return Probe{}, err
	}
	if err := checkExtensions(object); err != nil {
		return Probe{}, err
	}
	return Probe{
		ProbeID:                 probeID,
		TerminalBackendID:       backendID,
		ImplementationVersion:   implementationVersion,
		ProtocolVersion:         protocolVersion,
		ImplementationKind:      kind,
		ExecutableDigest:        digest,
		Platform:                platform,
		OSVersion:               osVersion,
		Availability:            availability,
		BackendGenerationDigest: generationDigest,
		CapabilityClaims:        claims,
		EvidenceIDs:             evidenceIDs,
		ProbedAt:                probedAt,
	}, nil
}

// evidenceMembers is the exact closed Capability Evidence member set.
var evidenceMembers = []string{
	"schema", "schema_version", "evidence_id", "terminal_backend_id",
	"implementation_version", "protocol_version", "backend_generation_digest",
	"capability", "value", "platform", "os_version", "conformance_fixture_id",
	"observed_at", "expires_at", "issuer", "issuer_id",
	"attestation_signature", "facts", "terminal_binding_id", "provider_id",
	"provider_build", "sentinel_result", "provider_auth_smoke_result",
	"extensions",
}

// ParseEvidence validates one Capability Evidence 1.0.0 document. The
// signature is structurally decoded here; cryptographic verification happens
// in Reconcile through the caller's SignatureVerifier, because a
// caller-supplied issuer ID is not authentication. Expiry (expires_at
// strictly after observed_at) is enforced here; liveness at admission time
// is enforced in Reconcile.
func ParseEvidence(raw []byte) (Evidence, error) {
	object, err := decodeStrictObject(raw)
	if err != nil {
		return Evidence{}, err
	}
	if err := checkExactMembers(object, evidenceMembers); err != nil {
		return Evidence{}, err
	}
	if schema, err := stringMember(object, "schema"); err != nil || schema != SchemaCapabilityEvidence {
		return Evidence{}, mismatchf("evidence schema")
	}
	if version, err := stringMember(object, "schema_version"); err != nil || version != SchemaVersion100 {
		return Evidence{}, mismatchf("evidence schema version")
	}
	evidenceID, err := stringMember(object, "evidence_id")
	if err != nil {
		return Evidence{}, err
	}
	if err := checkIdentity(object, "evidence_id", evidenceID); err != nil {
		return Evidence{}, err
	}
	backendID, err := stringMember(object, "terminal_backend_id")
	if err != nil {
		return Evidence{}, err
	}
	if _, err := ParseID(backendID); err != nil {
		return Evidence{}, mismatchf("evidence backend identity")
	}
	implementationVersion, err := semverMember(object, "implementation_version")
	if err != nil {
		return Evidence{}, err
	}
	protocolVersion, err := semverMember(object, "protocol_version")
	if err != nil {
		return Evidence{}, err
	}
	if semverMajor(protocolVersion) != 1 {
		return Evidence{}, mismatchf("evidence protocol major 1")
	}
	generationDigest, err := digestMember(object, "backend_generation_digest")
	if err != nil {
		return Evidence{}, err
	}
	capability, err := stringMember(object, "capability")
	if err != nil {
		return Evidence{}, err
	}
	if _, known := capabilityRegistry[capability]; !known {
		return Evidence{}, mismatchf("capability vocabulary")
	}
	// Evidence value is the literal true: a false claim has no matching
	// evidence, so a false-valued object is malformed.
	if value, ok := object["value"].(bool); !ok || !value {
		return Evidence{}, mismatchf("evidence value")
	}
	platformName, err := stringMember(object, "platform")
	if err != nil {
		return Evidence{}, err
	}
	platform, err := scalar.ParsePlatform(platformName)
	if err != nil {
		return Evidence{}, mismatchf("evidence platform")
	}
	osVersion, err := boundedStringMember(object, "os_version")
	if err != nil {
		return Evidence{}, err
	}
	fixtureID, err := digestMember(object, "conformance_fixture_id")
	if err != nil {
		return Evidence{}, err
	}
	observedAt, err := timestampMember(object, "observed_at")
	if err != nil {
		return Evidence{}, err
	}
	expiresAt, err := timestampMember(object, "expires_at")
	if err != nil {
		return Evidence{}, err
	}
	observed, err := observedAt.Time()
	if err != nil {
		return Evidence{}, mismatchf("document timestamp")
	}
	expires, err := expiresAt.Time()
	if err != nil {
		return Evidence{}, mismatchf("document timestamp")
	}
	if !expires.After(observed) {
		return Evidence{}, mismatchf("evidence expiry")
	}
	issuer, err := stringMember(object, "issuer")
	if err != nil {
		return Evidence{}, err
	}
	if issuer != IssuerRelease && issuer != IssuerLocalProbe {
		return Evidence{}, mismatchf("evidence issuer")
	}
	issuerID, err := digestMember(object, "issuer_id")
	if err != nil {
		return Evidence{}, err
	}
	signature, err := parseAttestationSignature(object)
	if err != nil {
		return Evidence{}, err
	}
	facts, err := stringArrayMember(object, "facts")
	if err != nil {
		return Evidence{}, err
	}
	if err := checkClosedList(facts, evidenceFactVocabulary, maxFacts, true); err != nil {
		return Evidence{}, err
	}
	realm, err := parseRealmMembers(object, capability)
	if err != nil {
		return Evidence{}, err
	}
	if err := checkExtensions(object); err != nil {
		return Evidence{}, err
	}
	return Evidence{
		EvidenceID:              evidenceID,
		TerminalBackendID:       backendID,
		ImplementationVersion:   implementationVersion,
		ProtocolVersion:         protocolVersion,
		BackendGenerationDigest: generationDigest,
		Capability:              capability,
		Platform:                platform,
		OSVersion:               osVersion,
		ConformanceFixtureID:    fixtureID,
		ObservedAt:              observedAt,
		ExpiresAt:               expiresAt,
		Issuer:                  issuer,
		IssuerID:                issuerID,
		AttestationSignature:    signature,
		Facts:                   facts,
		TerminalBindingID:       realm.bindingID,
		ProviderID:              realm.providerID,
		ProviderBuild:           realm.providerBuild,
		SentinelResult:          realm.sentinelResult,
		ProviderAuthSmokeResult: realm.smokeResult,
	}, nil
}

// parseAttestationSignature structurally decodes the rsa-sha256: plus Base64
// RFC 4648 attestation signature. Decoding success is not validity.
func parseAttestationSignature(object map[string]any) ([]byte, error) {
	encoded, err := stringMember(object, "attestation_signature")
	if err != nil {
		return nil, err
	}
	raw, found := strings.CutPrefix(encoded, signatureSchemePrefix)
	if !found || raw == "" {
		return nil, mismatchf("evidence signature scheme")
	}
	signature, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(signature) == 0 {
		return nil, mismatchf("evidence signature encoding")
	}
	return signature, nil
}

// realmMembers carries the credential-realm conditional members.
type realmMembers struct {
	bindingID      string
	providerID     string
	providerBuild  string
	sentinelResult string
	smokeResult    string
}

// parseRealmMembers enforces the credential-realm conditional shape: the
// binding digest, provider ID/build, sentinel, and smoke members are
// required exactly for credential_capable_execution_realm and null
// otherwise. A realm claim proved by realm-less evidence (or realm evidence
// attached to any other claim) is malformed.
func parseRealmMembers(object map[string]any, capability string) (realmMembers, error) {
	isRealm := capability == credentialRealmCapability
	bindingID, err := digestOrNullMember(object, "terminal_binding_id")
	if err != nil {
		return realmMembers{}, err
	}
	providerID, providerBuild, err := parseRealmProvider(object)
	if err != nil {
		return realmMembers{}, err
	}
	sentinelResult, err := parseRealmLiteral(object, "sentinel_result")
	if err != nil {
		return realmMembers{}, err
	}
	smokeResult, err := parseRealmLiteral(object, "provider_auth_smoke_result")
	if err != nil {
		return realmMembers{}, err
	}
	if isRealm {
		if bindingID == "" || providerID == "" || providerBuild == "" ||
			sentinelResult == "" || smokeResult == "" {
			return realmMembers{}, mismatchf("evidence realm binding")
		}
	} else if bindingID != "" || providerID != "" || providerBuild != "" ||
		sentinelResult != "" || smokeResult != "" {
		return realmMembers{}, mismatchf("evidence realm binding")
	}
	return realmMembers{
		bindingID:      bindingID,
		providerID:     providerID,
		providerBuild:  providerBuild,
		sentinelResult: sentinelResult,
		smokeResult:    smokeResult,
	}, nil
}

// GenerationDigest binds a backend's host-local raw generation to its
// advertisable digest: SHA-256 over the UTF-8 domain separator followed by
// the raw generation bytes (§4.B). The raw generation is a string[1..256]
// in UTF-8 characters over valid UTF-8 (SPEC.md:321); anything else
// (including empty, which would make every backend share one digest) is
// CodeStaleGeneration, never a derived value.
func GenerationDigest(rawGeneration string) (string, error) {
	if !utf8.ValidString(rawGeneration) || utf8.RuneCountInString(rawGeneration) < 1 || utf8.RuneCountInString(rawGeneration) > maxGenerationRunes {
		return "", &Error{Code: CodeStaleGeneration, Detail: "backend_generation bound"}
	}
	material := append([]byte(generationDigestDomain), rawGeneration...)
	sum := sha256.Sum256(material)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// SignatureVerifier verifies one evidence attestation signature. issuerID is
// the evidence issuer_id, message the domain-separated canonical bytes from
// UnsignedEvidenceBytes, and signature the decoded attestation bytes. A nil
// return authenticates through the caller's trusted release or
// host-incarnation key registry; any error (unknown key, invalid signature,
// registry failure) fails closed. A nil verifier admits nothing.
type SignatureVerifier func(issuerID string, message, signature []byte) error

// integrityFailure refuses with CodeIntegrityFailure and a static detail.
func integrityFailure(detail string) *Error {
	return &Error{Code: CodeIntegrityFailure, Detail: detail}
}

// UnsignedEvidenceBytes rebuilds the exact bytes an attestation signs: ASCII
// evidenceSignatureDomain, one zero byte, and the RFC 8785 JCS bytes of the
// evidence object with exactly evidence_id and attestation_signature
// omitted (§4.D). Timestamps reuse their validated source spellings so the
// rebuilt bytes match what the issuer signed.
func UnsignedEvidenceBytes(evidence Evidence) ([]byte, error) {
	object := map[string]any{
		"schema":                     SchemaCapabilityEvidence,
		"schema_version":             SchemaVersion100,
		"terminal_backend_id":        evidence.TerminalBackendID,
		"implementation_version":     evidence.ImplementationVersion,
		"protocol_version":           evidence.ProtocolVersion,
		"backend_generation_digest":  evidence.BackendGenerationDigest,
		"capability":                 evidence.Capability,
		"value":                      true,
		"platform":                   evidence.Platform.String(),
		"os_version":                 evidence.OSVersion,
		"conformance_fixture_id":     evidence.ConformanceFixtureID,
		"observed_at":                evidence.ObservedAt.String(),
		"expires_at":                 evidence.ExpiresAt.String(),
		"issuer":                     evidence.Issuer,
		"issuer_id":                  evidence.IssuerID,
		"facts":                      stringsToValues(evidence.Facts),
		"terminal_binding_id":        nullableDigest(evidence.TerminalBindingID),
		"provider_id":                nullableString(evidence.ProviderID),
		"provider_build":             nullableString(evidence.ProviderBuild),
		"sentinel_result":            nullableString(evidence.SentinelResult),
		"provider_auth_smoke_result": nullableString(evidence.ProviderAuthSmokeResult),
		"extensions":                 map[string]any{},
	}
	serialized, err := json.Marshal(object)
	if err != nil {
		return nil, integrityFailure("evidence canonical bytes")
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		return nil, integrityFailure("evidence canonical bytes")
	}
	message := append([]byte(evidenceSignatureDomain), 0x00)
	return append(message, canonical...), nil
}

// stringsToValues converts a string list to JSON values.
func stringsToValues(values []string) []any {
	members := make([]any, 0, len(values))
	for _, value := range values {
		members = append(members, value)
	}
	return members
}

// nullableDigest renders an empty digest as JSON null.
func nullableDigest(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// nullableString renders an empty string as JSON null.
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// Admitted is the reconciled capability set for one Manifest/Probe tuple:
// every true probe claim with generation-bound, unexpired, signature-valid
// evidence covering all its requirements. Capabilities holds the sorted
// admitted true-claim names; EvidenceIDs holds the sorted IDs of the
// evidence objects that proved them.
type Admitted struct {
	Capabilities []string
	EvidenceIDs  []string
}

// Has reports whether capability is an admitted true claim.
func (admitted Admitted) Has(capability string) bool {
	for _, member := range admitted.Capabilities {
		if member == capability {
			return true
		}
	}
	return false
}

// HasOperation reports whether any admitted true capability confers the
// operation through its registry dependent_operations.
func (admitted Admitted) HasOperation(operation string) bool {
	for _, capability := range admitted.Capabilities {
		row, known := capabilityRegistry[capability]
		if !known {
			continue
		}
		for _, conferred := range row.dependentOperations {
			if conferred == operation {
				return true
			}
		}
	}
	return false
}

// CapabilitiesForOperation returns the sorted registry capabilities whose
// dependent_operations confer the operation. manifest and probe confer
// through no capability and return an empty set: they carry no capability
// dependency. An unknown operation is refused: the operation vocabulary is
// closed.
func CapabilitiesForOperation(operation string) ([]string, error) {
	if !operationVocabulary[operation] {
		return nil, mismatchf("operation vocabulary")
	}
	var capabilities []string
	for capability, row := range capabilityRegistry {
		for _, conferred := range row.dependentOperations {
			if conferred == operation {
				capabilities = append(capabilities, capability)
				break
			}
		}
	}
	return capabilities, nil
}

// CheckOperation gates one operation against the admitted set. manifest and
// probe carry no capability dependency (§4.D): they are admitted for any
// admitted set, including the empty one. Every other operation must be in
// the closed vocabulary and conferred by at least one admitted true
// capability, or it is refused with CodeCapabilityUnproven. Conditional §4.C
// dependencies (interactivity, credential need, attach overlap, provider
// observation) are evaluated by the lifecycle owner at the operation call
// site; this gate proves only the capability-confers-operation half, which
// is the bound this file owns.
func CheckOperation(operation string, admitted Admitted) error {
	if !operationVocabulary[operation] {
		return mismatchf("operation vocabulary")
	}
	if operation == "manifest" || operation == "probe" {
		return nil
	}
	if !admitted.HasOperation(operation) {
		return &Error{Code: CodeCapabilityUnproven, Detail: "operation capability dependency"}
	}
	return nil
}

// Reconcile admits one Probe against its Manifest with the returned
// evidence set, binding everything to the AX-known raw generation and the
// admission instant (§4.B keyed relation, §4.D evidence binding). The check
// order is deterministic: identity equality, protocol/platform membership,
// generation binding, claim relation, per-object evidence usability
// (tuple binding, liveness, signature), per-claim requirement coverage,
// then the exact evidence_ids set rule. The first failure wins; every
// failure fails closed before activation.
//
// Each supplied evidence object must independently prove its named claim:
// split-facts evidence (requirement coverage spread across several objects,
// none sufficient alone) is refused. That reading follows the singular
// rule-5 phrasing ("Capability Evidence ... satisfies every listed evidence
// requirement") and is stated here as the bound.
func Reconcile(manifest Manifest, probe Probe, evidence []Evidence, rawGeneration string, now time.Time, verify SignatureVerifier) (Admitted, error) {
	if verify == nil {
		return Admitted{}, integrityFailure("evidence signature verifier")
	}
	if err := checkProbeIdentity(manifest, probe); err != nil {
		return Admitted{}, err
	}
	if err := checkProbeMembership(manifest, probe); err != nil {
		return Admitted{}, err
	}
	if err := checkProbeGeneration(probe, rawGeneration); err != nil {
		return Admitted{}, err
	}
	manifestClaims := indexClaims(manifest.StaticCapabilityClaims)
	if err := checkClaimRelation(manifestClaims, probe.CapabilityClaims); err != nil {
		return Admitted{}, err
	}
	probeClaims := indexClaims(probe.CapabilityClaims)
	used, err := checkEvidenceSet(manifest, probe, probeClaims, evidence, now, verify)
	if err != nil {
		return Admitted{}, err
	}
	if err := checkEvidenceCoverage(probeClaims, used); err != nil {
		return Admitted{}, err
	}
	if err := checkEvidenceIDs(probe, used); err != nil {
		return Admitted{}, err
	}
	return admitCapabilities(probeClaims, used), nil
}

// checkProbeIdentity enforces the §4.B identity equalities. Executable
// substitution additionally maps to the more specific untrusted code.
func checkProbeIdentity(manifest Manifest, probe Probe) error {
	if probe.TerminalBackendID != manifest.TerminalBackendID ||
		probe.ImplementationVersion != manifest.ImplementationVersion ||
		probe.ImplementationKind != manifest.ImplementationKind {
		return mismatchf("probe manifest binding")
	}
	// The backend IDs are already equal here (refused above), so a digest
	// difference is always a substitution of the admitted executable: it
	// is untrusted, never a plain mismatch.
	if probe.ExecutableDigest != manifest.ExecutableDigest {
		return &Error{Code: CodeUntrusted, BackendID: probe.TerminalBackendID, Detail: "executable substitution"}
	}
	return nil
}

// checkProbeMembership enforces array-to-scalar selection as membership:
// the probe protocol version must be exactly one Manifest member and the
// probe platform exactly one Manifest platform member.
func checkProbeMembership(manifest Manifest, probe Probe) error {
	member := false
	for _, version := range manifest.ProtocolVersions {
		if probe.ProtocolVersion == version {
			member = true
			break
		}
	}
	if !member {
		return mismatchf("probe protocol membership")
	}
	for _, platform := range manifest.Platforms {
		if probe.Platform == platform {
			return nil
		}
	}
	return mismatchf("probe platform membership")
}

// checkProbeGeneration binds the probe to the AX-known raw generation. A
// digest mismatch (or an underivable raw generation, including empty, which
// would make every backend share one digest) is stale, never a retry hint.
func checkProbeGeneration(probe Probe, rawGeneration string) error {
	digest, err := GenerationDigest(rawGeneration)
	if err != nil {
		return err
	}
	if digest != probe.BackendGenerationDigest {
		return &Error{Code: CodeStaleGeneration, BackendID: probe.TerminalBackendID, Detail: "probe generation binding"}
	}
	return nil
}

// indexClaims keys validated claims by capability. The validated lists are
// duplicate-free, so the index cannot collide.
func indexClaims(claims []Claim) map[string]Claim {
	indexed := make(map[string]Claim, len(claims))
	for _, claim := range claims {
		indexed[claim.Capability] = claim
	}
	return indexed
}

// checkClaimRelation enforces the complete §4.B keyed relation between the
// Manifest static claims M and the Probe claims P.
func checkClaimRelation(manifest map[string]Claim, probe []Claim) error {
	for _, proved := range probe {
		capability := proved.Capability
		static, exists := manifest[capability]
		switch {
		case proved.Origin == OriginStatic && !exists:
			return mismatchf("probe static claim without manifest")
		case proved.Origin == OriginStatic:
			if !equalClaim(proved, static) {
				return mismatchf("probe static claim echo")
			}
		case exists && !static.GenerationVariable:
			return mismatchf("probe override of stable claim")
		case exists:
			if proved.GenerationVariable != static.GenerationVariable ||
				!equalStrings(proved.DependentOperations, static.DependentOperations) ||
				!equalStrings(proved.EvidenceRequirements, static.EvidenceRequirements) {
				return mismatchf("probe override registry binding")
			}
		default:
			row := capabilityRegistry[proved.Capability]
			if proved.GenerationVariable != row.generationVariable ||
				!equalStrings(proved.DependentOperations, row.dependentOperations) ||
				!equalStrings(proved.EvidenceRequirements, row.evidenceRequirements) {
				return mismatchf("probe addition registry binding")
			}
		}
	}
	for capability := range manifest {
		if _, present := indexProbeCapability(probe, capability); !present {
			return mismatchf("probe omission of manifest claim")
		}
	}
	return nil
}

// indexProbeCapability reports whether the probe lists the capability. The
// probe list is validated duplicate-free, so presence is unambiguous.
func indexProbeCapability(probe []Claim, capability string) (Claim, bool) {
	for _, claim := range probe {
		if claim.Capability == capability {
			return claim, true
		}
	}
	return Claim{}, false
}

// checkEvidenceSet validates every supplied evidence object: it must name a
// present true probe claim, bind to this exact Manifest/Probe tuple, be
// live at now, and carry a valid attestation signature. Expired,
// wrong-generation, dangling, or otherwise unusable evidence refuses the
// whole reconciliation (§4.B); a forged or unverifiable signature is an
// integrity failure. The returned objects are the usable set that coverage
// and ID-set checks consume.
func checkEvidenceSet(manifest Manifest, probe Probe, probeClaims map[string]Claim, evidence []Evidence, now time.Time, verify SignatureVerifier) ([]Evidence, error) {
	used := make([]Evidence, 0, len(evidence))
	seen := make(map[string]Evidence, len(evidence))
	for _, object := range evidence {
		claim, present := probeClaims[object.Capability]
		if !present || !claim.Value {
			return nil, mismatchf("evidence claim binding")
		}
		if err := checkEvidenceTuple(manifest, probe, object); err != nil {
			return nil, err
		}
		if err := checkEvidenceLiveness(object, now); err != nil {
			return nil, err
		}
		if previous, duplicate := seen[object.EvidenceID]; duplicate {
			if !equalEvidence(previous, object) {
				return nil, mismatchf("conflicting evidence")
			}
			continue
		}
		seen[object.EvidenceID] = object
		if err := checkEvidenceSignature(object, verify); err != nil {
			return nil, err
		}
		used = append(used, object)
	}
	return used, nil
}

// equalEvidence reports whether two evidence objects with the same ID carry
// identical members. Same-ID differing-bytes objects are conflicting
// evidence and invalidate the whole Probe.
func equalEvidence(left, right Evidence) bool {
	if left.TerminalBackendID != right.TerminalBackendID ||
		left.ImplementationVersion != right.ImplementationVersion ||
		left.ProtocolVersion != right.ProtocolVersion ||
		left.BackendGenerationDigest != right.BackendGenerationDigest ||
		left.Capability != right.Capability ||
		left.Platform != right.Platform ||
		left.OSVersion != right.OSVersion ||
		left.ConformanceFixtureID != right.ConformanceFixtureID ||
		left.ObservedAt.String() != right.ObservedAt.String() ||
		left.ExpiresAt.String() != right.ExpiresAt.String() ||
		left.Issuer != right.Issuer ||
		left.IssuerID != right.IssuerID ||
		!equalStrings(left.Facts, right.Facts) ||
		left.TerminalBindingID != right.TerminalBindingID ||
		left.ProviderID != right.ProviderID ||
		left.ProviderBuild != right.ProviderBuild ||
		left.SentinelResult != right.SentinelResult ||
		left.ProviderAuthSmokeResult != right.ProviderAuthSmokeResult {
		return false
	}
	return bytes.Equal(left.AttestationSignature, right.AttestationSignature)
}

// checkEvidenceTuple binds one evidence object to this exact Manifest/Probe
// tuple: backend, versions, generation digest, platform, os_version, and
// the Manifest conformance fixture must all match.
func checkEvidenceTuple(manifest Manifest, probe Probe, object Evidence) error {
	if object.TerminalBackendID != probe.TerminalBackendID ||
		object.ImplementationVersion != probe.ImplementationVersion ||
		object.ProtocolVersion != probe.ProtocolVersion ||
		object.BackendGenerationDigest != probe.BackendGenerationDigest ||
		object.Platform != probe.Platform ||
		object.OSVersion != probe.OSVersion ||
		object.ConformanceFixtureID != manifest.ConformanceFixtureID {
		return mismatchf("evidence tuple binding")
	}
	return nil
}

// checkEvidenceLiveness requires observed_at <= now < expires_at. Future
// observed instants and expired objects are both unusable: the former is
// not yet proof, the latter no longer is.
func checkEvidenceLiveness(object Evidence, now time.Time) error {
	observed, err := object.ObservedAt.Time()
	if err != nil {
		return mismatchf("document timestamp")
	}
	expires, err := object.ExpiresAt.Time()
	if err != nil {
		return mismatchf("document timestamp")
	}
	if observed.After(now) || !now.Before(expires) {
		return mismatchf("evidence liveness")
	}
	return nil
}

// checkEvidenceSignature verifies the attestation through the caller's
// trusted key registry. Missing, unknown-key, or invalid signatures fail
// closed before capability admission; a caller-supplied issuer ID alone is
// never authentication.
func checkEvidenceSignature(object Evidence, verify SignatureVerifier) error {
	message, err := UnsignedEvidenceBytes(object)
	if err != nil {
		return err
	}
	if err := verify(object.IssuerID, message, object.AttestationSignature); err != nil {
		return integrityFailure("evidence attestation")
	}
	return nil
}

// checkEvidenceCoverage requires every true probe claim to hold at least
// one usable evidence object whose facts cover every listed requirement
// through the requirement-to-fact mapping. False claims need no evidence
// and take none: evidence for a false claim was already refused as
// unbound in checkEvidenceSet.
func checkEvidenceCoverage(probeClaims map[string]Claim, used []Evidence) error {
	byCapability := make(map[string][]Evidence, len(used))
	for _, object := range used {
		byCapability[object.Capability] = append(byCapability[object.Capability], object)
	}
	for capability, claim := range probeClaims {
		if !claim.Value {
			continue
		}
		covered := false
		for _, object := range byCapability[capability] {
			if factsCover(object.Facts, claim.EvidenceRequirements) {
				covered = true
				break
			}
		}
		if !covered {
			return mismatchf("evidence requirement coverage")
		}
	}
	return nil
}

// factsCover reports whether the fact set satisfies every listed
// requirement through the requirement-to-fact mapping. Supplementary facts
// (ui_absent, prompt_absent) never satisfy a requirement alone.
func factsCover(facts []string, requirements []string) bool {
	held := make(map[string]bool, len(facts))
	for _, fact := range facts {
		held[fact] = true
	}
	for _, requirement := range requirements {
		if !held[requirementFact[requirement]] {
			return false
		}
	}
	return true
}

// checkEvidenceIDs enforces the exact rule-6 set equality: evidence_ids
// must be exactly the sorted unique IDs of the usable evidence objects.
// Dangling IDs, unlisted usable objects, and ordering deviations all fail.
func checkEvidenceIDs(probe Probe, used []Evidence) error {
	want := make([]string, 0, len(used))
	for _, object := range used {
		want = append(want, object.EvidenceID)
	}
	for index := range want {
		for next := index + 1; next < len(want); next++ {
			if want[next] < want[index] {
				want[index], want[next] = want[next], want[index]
			}
		}
	}
	if !equalStrings(probe.EvidenceIDs, want) {
		return mismatchf("evidence id set binding")
	}
	return nil
}

// admitCapabilities collects the sorted admitted true-claim names and the
// sorted evidence IDs that proved them.
func admitCapabilities(probeClaims map[string]Claim, used []Evidence) Admitted {
	var capabilities []string
	for capability, claim := range probeClaims {
		if claim.Value {
			capabilities = append(capabilities, capability)
		}
	}
	for index := range capabilities {
		for next := index + 1; next < len(capabilities); next++ {
			if capabilities[next] < capabilities[index] {
				capabilities[index], capabilities[next] = capabilities[next], capabilities[index]
			}
		}
	}
	evidenceIDs := make([]string, 0, len(used))
	for _, object := range used {
		evidenceIDs = append(evidenceIDs, object.EvidenceID)
	}
	for index := range evidenceIDs {
		for next := index + 1; next < len(evidenceIDs); next++ {
			if evidenceIDs[next] < evidenceIDs[index] {
				evidenceIDs[index], evidenceIDs[next] = evidenceIDs[next], evidenceIDs[index]
			}
		}
	}
	return Admitted{Capabilities: capabilities, EvidenceIDs: evidenceIDs}
}

// AdmitProbe is the registry-bound production admission: it parses the
// Manifest, resolves the backend through the admitted registry (unknown
// identities fail closed, never default), binds the Manifest identity tuple
// to the admitted record, then reconciles the Probe with its evidence set.
// Trust is established before any probe: registration-time trust admission
// owns executables, and this gate additionally refuses executable
// substitution as untrusted and any other identity drift as drift.
//
// rawGeneration is the AX-known current raw generation for the backend;
// now is the admission instant; verify authenticates evidence through the
// trusted key registry and must be non-nil.
func (registry *Registry) AdmitProbe(manifestRaw, probeRaw []byte, evidenceRaws [][]byte, rawGeneration string, now time.Time, verify SignatureVerifier) (Admitted, error) {
	if registry == nil {
		return Admitted{}, &Error{Code: CodeNotFound, Detail: "registry unavailable"}
	}
	manifest, err := ParseManifest(manifestRaw)
	if err != nil {
		return Admitted{}, err
	}
	record, err := registry.Resolve(manifest.TerminalBackendID)
	if err != nil {
		return Admitted{}, err
	}
	if err := checkManifestRecordBinding(manifest, record); err != nil {
		return Admitted{}, err
	}
	probe, err := ParseProbe(probeRaw)
	if err != nil {
		return Admitted{}, err
	}
	evidence := make([]Evidence, 0, len(evidenceRaws))
	for _, raw := range evidenceRaws {
		object, err := ParseEvidence(raw)
		if err != nil {
			return Admitted{}, err
		}
		evidence = append(evidence, object)
	}
	return Reconcile(manifest, probe, evidence, rawGeneration, now, verify)
}

// checkManifestRecordBinding binds the Manifest identity tuple to the
// admitted registry record: implementation version, protocol set, platform
// set, and kind must match member-for-member, and the executable digest
// must match exactly. Any version/kind/platform drift is drift; an
// executable substitution is untrusted.
func checkManifestRecordBinding(manifest Manifest, record Registration) error {
	if manifest.ImplementationVersion != record.ImplementationVersion ||
		manifest.ImplementationKind != record.Kind ||
		!equalStrings(manifest.ProtocolVersions, record.ProtocolVersions) ||
		!equalPlatforms(manifest.Platforms, record.Platforms) {
		return &Error{Code: CodeDrift, BackendID: manifest.TerminalBackendID, Detail: "manifest implementation drift"}
	}
	if manifest.ExecutableDigest != record.ExecutableDigest {
		return &Error{Code: CodeUntrusted, BackendID: manifest.TerminalBackendID, Detail: "executable substitution"}
	}
	return nil
}

// equalPlatforms reports element-wise equality of two platform lists. Both
// lists are validated sorted unique, so order is significant.
func equalPlatforms(left, right []scalar.Platform) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// parseRealmProvider validates the provider_id/provider_build pair: null or
// a provider-id grammar value plus a string[1..256] build.
func parseRealmProvider(object map[string]any) (string, string, error) {
	rawID, known := object["provider_id"]
	if !known {
		return "", "", mismatchf("document members")
	}
	providerID := ""
	if rawID != nil {
		name, ok := rawID.(string)
		if !ok {
			return "", "", mismatchf("document member type")
		}
		if _, err := scalar.ParseProviderID(name); err != nil {
			return "", "", mismatchf("evidence provider identity")
		}
		providerID = name
	}
	rawBuild, known := object["provider_build"]
	if !known {
		return "", "", mismatchf("document members")
	}
	providerBuild := ""
	if rawBuild != nil {
		build, ok := rawBuild.(string)
		if !ok {
			return "", "", mismatchf("document member type")
		}
		if !utf8.ValidString(build) || utf8.RuneCountInString(build) < 1 || utf8.RuneCountInString(build) > maxOpaqueString {
			return "", "", mismatchf("document string bound")
		}
		providerBuild = build
	}
	if (providerID == "") != (providerBuild == "") {
		return "", "", mismatchf("evidence realm binding")
	}
	return providerID, providerBuild, nil
}

// parseRealmLiteral validates a literal passed|null realm member.
func parseRealmLiteral(object map[string]any, name string) (string, error) {
	raw, known := object[name]
	if !known {
		return "", mismatchf("document members")
	}
	if raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok || value != "passed" {
		return "", mismatchf("evidence realm result")
	}
	return value, nil
}
