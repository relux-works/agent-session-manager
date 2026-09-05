package provhost

import (
	"encoding/json"
	"regexp"
	"strings"
)

// This file validates the Provider Identity Record (Section 5.5,
// urn:ax:schema:provider-identity 1.0.0) the host reads from
// identify-session results and carries in launch, resume, quiesce,
// and fork bodies, plus the identify-session success wrapper. The
// record_id is required to be a digest but is not recomputed here:
// content addressing belongs to the object-identity story, and this
// package holds no canonical encoder. The opaque_identity absolute
// prefix rule mirrors the decidable half of the Section 5.5
// sentence: a value beginning with an absolute path is refused, and
// an embedded path the prefix cannot see is a stated bound.
//
// CheckIdentity deliberately re-validates the record shape instead of
// delegating to canonicaljson.validateProviderIdentityRecord: this
// package must mint Section 15.1 provider-stdio refusals
// (provider_protocol_error with member attribution), not identity
// errors, and the shared package offers no such entry point. The two
// validators must agree on the Section 5.5 shape rule for rule; the
// decoder half of that agreement is pinned by the derived
// TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON sweep, which
// judges escape and raw-UTF-8 vectors through the shared
// decodeStrictObject every entry here reads through, and any
// shape-rule change must land in both validators in the same change.

// identitySchema is the exact schema identifier the record carries.
const identitySchema = "urn:ax:schema:provider-identity"

// identitySchemaVersion is the only identity version accepted.
const identitySchemaVersion = "1.0.0"

// identityKinds is the closed identity-kind enum in section order.
var identityKinds = []string{
	"session_uuid",
	"session_path_or_id",
	"backend_conversation_uuid",
	"task_board_managed",
	"provider_defined",
}

// identityMembers is the exact required member set of a Provider
// Identity Record.
var identityMembers = map[string]bool{
	"schema":                    true,
	"schema_version":            true,
	"record_id":                 true,
	"subject_id":                true,
	"session_id":                true,
	"provider_id":               true,
	"provider_version":          true,
	"provider_version_range":    true,
	"native_session_id":         true,
	"identity_kind":             true,
	"logical_workspace_id":      true,
	"backend_realm_fingerprint": true,
	"opaque_identity":           true,
	"created_by_host_id":        true,
	"created_at":                true,
	"extensions":                true,
}

// identityRequired lists identityMembers in a fixed order so a record
// missing several members always names the same one.
var identityRequired = []string{
	"schema",
	"schema_version",
	"record_id",
	"subject_id",
	"session_id",
	"provider_id",
	"provider_version",
	"provider_version_range",
	"native_session_id",
	"identity_kind",
	"logical_workspace_id",
	"backend_realm_fingerprint",
	"opaque_identity",
	"created_by_host_id",
	"created_at",
	"extensions",
}

// identityKeyPattern is the provider-identity key grammar Section 5.5
// states: [a-z][a-z0-9_.-]{0,63}.
var identityKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

// windowsDrivePattern matches a Windows drive-qualified absolute
// prefix: a letter, a colon, and a separator.
var windowsDrivePattern = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// CheckIdentity validates one Provider Identity Record naming the
// expected provider. Shape violations are provider_protocol_errors;
// a well-formed record for another provider is an invalid_config
// caller error: the caller correlated the wrong identity to this
// session.
func CheckIdentity(body []byte, wantProviderID string) error {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if name, unknown := unknownMember(members, identityMembers); unknown {
		failure, err := failProtocol("identity carries unknown member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if name, missing := missingMember(members, identityRequired); missing {
		failure, err := failProtocol("identity misses a required member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != identitySchema {
		failure, err := failProtocol("identity schema is not the provider identity", "schema")
		if err != nil {
			return err
		}
		return failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != identitySchemaVersion {
		failure, err := failProtocol("identity schema_version is not 1.0.0", "schema_version")
		if err != nil {
			return err
		}
		return failure
	}
	if digest, ok := rawString(members["record_id"]); !ok || !isDigest(digest) {
		failure, err := failProtocol("identity record_id is not a digest", "record_id")
		if err != nil {
			return err
		}
		return failure
	}
	subject, ok := rawString(members["subject_id"])
	if !ok || !isUUIDv7(subject) {
		failure, err := failProtocol("identity subject_id is not a UUIDv7", "subject_id")
		if err != nil {
			return err
		}
		return failure
	}
	session, ok := rawString(members["session_id"])
	if !ok || !isUUIDv7(session) {
		failure, err := failProtocol("identity session_id is not a UUIDv7", "session_id")
		if err != nil {
			return err
		}
		return failure
	}
	if subject != session {
		failure, err := failProtocol("identity subject_id does not equal session_id", "subject_id")
		if err != nil {
			return err
		}
		return failure
	}
	provider, ok := rawString(members["provider_id"])
	if !ok || !validProviderID(provider) {
		failure, err := failProtocol("identity provider_id is not a provider id", "provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if provider != wantProviderID {
		failure, err := failInvalid("identity names another provider")
		if err != nil {
			return err
		}
		return failure
	}
	if !isBoundedString(members["provider_version"], 1, 128) {
		failure, err := failProtocol("identity provider_version is not 1..128 characters", "provider_version")
		if err != nil {
			return err
		}
		return failure
	}
	if !isBoundedString(members["provider_version_range"], 1, 256) {
		failure, err := failProtocol("identity provider_version_range is not 1..256 characters", "provider_version_range")
		if err != nil {
			return err
		}
		return failure
	}
	if !isBoundedString(members["native_session_id"], 1, 512) {
		failure, err := failProtocol("identity native_session_id is not 1..512 characters", "native_session_id")
		if err != nil {
			return err
		}
		return failure
	}
	kind, ok := rawString(members["identity_kind"])
	if !ok || !isIdentityKind(kind) {
		failure, err := failProtocol("identity kind is not a registry member", "identity_kind")
		if err != nil {
			return err
		}
		return failure
	}
	if workspace, ok := rawString(members["logical_workspace_id"]); !ok || !isUUIDv7(workspace) {
		failure, err := failProtocol("identity workspace is not a UUIDv7", "logical_workspace_id")
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkIdentityRealm(members["backend_realm_fingerprint"], provider, kind); err != nil {
		return err
	}
	if err := checkIdentityOpaque(members["opaque_identity"]); err != nil {
		return err
	}
	if host, ok := rawString(members["created_by_host_id"]); !ok || !isUUIDv7(host) {
		failure, err := failProtocol("identity host is not a UUIDv7", "created_by_host_id")
		if err != nil {
			return err
		}
		return failure
	}
	if created, ok := rawString(members["created_at"]); !ok || !isTimestamp(created) {
		failure, err := failProtocol("identity timestamp is not a timestamp", "created_at")
		if err != nil {
			return err
		}
		return failure
	}
	if !isJSONObject(members["extensions"]) {
		failure, err := failProtocol("identity extensions is not an object", "extensions")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// checkIdentityRealm enforces the digest-or-null shape and the one
// conditional rule Section 5.5 states: an Antigravity
// backend_conversation_uuid MUST carry a non-null realm fingerprint,
// because the backend realm is a resume precondition there.
func checkIdentityRealm(raw json.RawMessage, provider, kind string) error {
	realm, isNull, ok := rawNullableString(raw)
	if !ok || (!isNull && !isDigest(realm)) {
		failure, err := failProtocol("identity realm is not a digest or null", "backend_realm_fingerprint")
		if err != nil {
			return err
		}
		return failure
	}
	if provider == "antigravity" && kind == "backend_conversation_uuid" && isNull {
		failure, err := failProtocol("identity realm is required for this backend kind", "backend_realm_fingerprint")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// checkIdentityOpaque enforces the explicit adapter-data surface: an
// object of at most 32 entries, keyed by the provider-identity key
// grammar, with non-secret string values of 1..1,024 characters that
// MUST NOT begin with an absolute path.
func checkIdentityOpaque(raw json.RawMessage) error {
	opaque, fault := decodeStrictObject(raw)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if len(opaque) > 32 {
		failure, err := failProtocol("identity opaque_identity exceeds 32 entries", "opaque_identity")
		if err != nil {
			return err
		}
		return failure
	}
	keys := make([]string, 0, len(opaque))
	for key := range opaque {
		keys = append(keys, key)
	}
	for _, key := range keys {
		if !identityKeyPattern.MatchString(key) {
			failure, err := failProtocol("identity opaque key is not a provider key", "opaque_identity")
			if err != nil {
				return err
			}
			return failure
		}
		value, ok := rawString(opaque[key])
		if !ok || runeLength(value) < 1 || runeLength(value) > 1024 {
			failure, err := failProtocol("identity opaque value is not 1..1024 characters", "opaque_identity")
			if err != nil {
				return err
			}
			return failure
		}
		if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || windowsDrivePattern.MatchString(value) {
			failure, err := failProtocol("identity opaque value begins with an absolute path", "opaque_identity")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

func isIdentityKind(value string) bool {
	for _, kind := range identityKinds {
		if value == kind {
			return true
		}
	}
	return false
}

// identifyMembers is the exact required member set of an
// identify-session success body.
var identifyMembers = map[string]bool{
	"identity":         true,
	"confidence":       true,
	"matched_evidence": true,
}

// identifyRequired lists identifyMembers in a fixed order.
var identifyRequired = []string{"identity", "confidence", "matched_evidence"}

// identifyConfidences is the closed confidence vocabulary.
var identifyConfidences = []string{"exact", "strong", "weak"}

// identifyEvidence is the closed matched-evidence vocabulary.
var identifyEvidence = []string{"native_id", "store_path", "provider_event", "backend_lookup"}

// DecodeIdentifyResult validates one identify-session success body:
// the Provider Identity Record for the expected provider, a closed
// confidence, and 1..4 sorted unique matched-evidence members.
func DecodeIdentifyResult(body []byte, wantProviderID string) error {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if name, unknown := unknownMember(members, identifyMembers); unknown {
		failure, err := failProtocol("identify result carries unknown member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if name, missing := missingMember(members, identifyRequired); missing {
		failure, err := failProtocol("identify result misses a required member", name)
		if err != nil {
			return err
		}
		return failure
	}
	confidence, ok := rawString(members["confidence"])
	if !ok || !isIdentifyConfidence(confidence) {
		failure, err := failProtocol("identify confidence is not exact strong or weak", "confidence")
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkIdentifyEvidence(members["matched_evidence"]); err != nil {
		return err
	}
	if !isJSONObject(members["identity"]) {
		failure, err := failProtocol("identify identity is not an object", "identity")
		if err != nil {
			return err
		}
		return failure
	}
	if err := CheckIdentity(members["identity"], wantProviderID); err != nil {
		return err
	}
	return nil
}

func isIdentifyConfidence(value string) bool {
	for _, confidence := range identifyConfidences {
		if value == confidence {
			return true
		}
	}
	return false
}

// checkIdentifyEvidence requires 1..4 sorted unique closed-vocabulary
// evidence members: the observation must cite what matched, exactly
// once each.
func checkIdentifyEvidence(raw json.RawMessage) error {
	evidence, ok := rawStringArray(raw)
	if !ok || len(evidence) < 1 || len(evidence) > 4 {
		failure, err := failProtocol("identify evidence is not 1..4 members", "matched_evidence")
		if err != nil {
			return err
		}
		return failure
	}
	allowed := map[string]bool{}
	for _, member := range identifyEvidence {
		allowed[member] = true
	}
	for _, member := range evidence {
		if !allowed[member] {
			failure, err := failProtocol("identify evidence names an unknown member", "matched_evidence")
			if err != nil {
				return err
			}
			return failure
		}
	}
	if !sortedUniqueStrings(evidence) {
		failure, err := failProtocol("identify evidence is not sorted unique", "matched_evidence")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
