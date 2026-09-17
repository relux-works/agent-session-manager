package provhost

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// This file constructs Provider Identity Records (Section 5.5,
// urn:ax:schema:provider-identity 1.0.0) from host-side inputs: the
// exact native session ID, backend realm, opaque adapter data, and
// probed build facts the adapter established before identifying the
// session. CheckIdentity in identity.go validates records a plugin
// returned; CreateIdentity builds records the host owns, and every
// record it emits must pass CheckIdentity, the owner's shape entry
// canonicaljson.CalculateObjectIdentity, and the binding attestation
// canonicaljson.VerifyObjectIdentity, because record_id is the true
// omit-self digest computed here, never a carried claim.
//
// Creation is a pure function of its params: no clock, no randomness,
// no filesystem, no durable state. Identical params marshal through
// encoding/json's bytewise-sorted map keys and the single textual
// record_id substitution below, so identical inputs emit byte-identical
// records across calls and processes, which is the idempotency this
// leaf promises. A time-varying created_at is the caller's input, not
// this function's decision: two creations at different instants name
// different instants and MUST differ, exactly as their params do.
//
// Every refusal is an invalid_config caller error: the caller, not a
// plugin, supplied the bad input. The member rules mirror CheckIdentity
// arm for arm (bounds, key grammar, the Antigravity backend-kind realm
// requirement, the absolute-prefix half of the opaque rule), and the
// verdict is conjoined with the owner's production identity entry the
// same way: a record this dialect admits but the owner refuses is
// refused at the backstop below rather than emitted through a drifted
// copy. Any shape-rule change must land in all three sites — here,
// CheckIdentity, and validateProviderIdentityRecord — in the same
// change.

// IdentityParams carries the host-side facts one Provider Identity
// Record is built from. SessionID becomes both subject_id and
// session_id, which Section 5.5 requires equal. BackendRealm is a
// content digest, or the empty string for a null fingerprint.
// OpaqueIdentity maps adapter keys to values; a nil map means the
// empty map. Extensions maps reverse-DNS keys to JSON values; a nil
// map means the empty object.
type IdentityParams struct {
	SessionID            string
	ProviderID           string
	ProviderVersion      string
	ProviderVersionRange string
	NativeSessionID      string
	IdentityKind         string
	LogicalWorkspaceID   string
	BackendRealm         string
	OpaqueIdentity       map[string]string
	CreatedByHostID      string
	CreatedAt            string
	Extensions           map[string]any
}

// reverseDNSPattern is the shared extensions-key grammar: two or more
// dot-separated lowercase labels. The symbol name and literal are
// pinned by the environ grammar census, which requires every copy in
// scope to spell this class identically; length 3..253 is checked
// beside the pattern, as the owner does.
var reverseDNSPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$`)

// maxExtensionMembers is the owner's 64-member extensions cap.
const maxExtensionMembers = 64

// placeholderDigestId stands in for record_id while the omit-self
// digest is computed. The owner requires the member present, so the
// staged bytes carry this well-formed digest and the final bytes carry
// the computed one; the substitution below matches the full
// `"record_id":"<placeholder>"` frame, never the bare digest, so a
// param value that repeats the digest text cannot misdirect it: values
// marshal with their quotes escaped while the frame's quotes are bare.
const placeholderDigestID = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// CreateIdentity builds one canonical Provider Identity Record 1.0.0
// from host-side params. It returns the record bytes on success and
// an invalid_config caller error naming the violated rule otherwise.
func CreateIdentity(params IdentityParams) ([]byte, error) {
	if !isUUIDv7(params.SessionID) {
		failure, err := failInvalid("create session_id is not a UUIDv7")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if !validProviderID(params.ProviderID) {
		failure, err := failInvalid("create provider_id is not a provider id")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if runeLength(params.ProviderVersion) < 1 || runeLength(params.ProviderVersion) > 128 {
		failure, err := failInvalid("create provider_version is not 1..128 characters")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if runeLength(params.ProviderVersionRange) < 1 || runeLength(params.ProviderVersionRange) > 256 {
		failure, err := failInvalid("create provider_version_range is not 1..256 characters")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if runeLength(params.NativeSessionID) < 1 || runeLength(params.NativeSessionID) > 512 {
		failure, err := failInvalid("create native_session_id is not 1..512 characters")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if !isIdentityKind(params.IdentityKind) {
		failure, err := failInvalid("create identity_kind is not a registry member")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if !isUUIDv7(params.LogicalWorkspaceID) {
		failure, err := failInvalid("create logical_workspace_id is not a UUIDv7")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if params.BackendRealm != "" && !isDigest(params.BackendRealm) {
		failure, err := failInvalid("create backend_realm_fingerprint is not a digest or empty")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if params.ProviderID == "antigravity" && params.IdentityKind == "backend_conversation_uuid" && params.BackendRealm == "" {
		failure, err := failInvalid("create backend_realm_fingerprint is required for this backend kind")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if err := checkCreateOpaque(params.OpaqueIdentity); err != nil {
		return nil, err
	}
	if !isUUIDv7(params.CreatedByHostID) {
		failure, err := failInvalid("create created_by_host_id is not a UUIDv7")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if !isTimestamp(params.CreatedAt) {
		failure, err := failInvalid("create created_at is not a timestamp")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if err := checkCreateExtensions(params.Extensions); err != nil {
		return nil, err
	}
	object := map[string]any{
		"schema":                    identitySchema,
		"schema_version":            identitySchemaVersion,
		"record_id":                 placeholderDigestID,
		"subject_id":                params.SessionID,
		"session_id":                params.SessionID,
		"provider_id":               params.ProviderID,
		"provider_version":          params.ProviderVersion,
		"provider_version_range":    params.ProviderVersionRange,
		"native_session_id":         params.NativeSessionID,
		"identity_kind":             params.IdentityKind,
		"logical_workspace_id":      params.LogicalWorkspaceID,
		"backend_realm_fingerprint": nil,
		"opaque_identity":           opaqueJSON(params.OpaqueIdentity),
		"created_by_host_id":        params.CreatedByHostID,
		"created_at":                params.CreatedAt,
		"extensions":                extensionsJSON(params.Extensions),
	}
	if params.BackendRealm != "" {
		object["backend_realm_fingerprint"] = params.BackendRealm
	}
	// Only extension values can fail this marshal: every other member
	// is a string, a nil, or a string map, all infallible. A channel,
	// function, or other non-JSON Go value in extensions refuses here
	// rather than emitting a partial record.
	staged, err := json.Marshal(object)
	if err != nil {
		failure, ferr := failInvalid("create extensions carry a non-JSON value")
		if ferr != nil {
			return nil, ferr
		}
		return nil, failure
	}
	// The backstop conjoins the owner's production identity entry:
	// extension nesting beyond depth 4, an extensions object past its
	// byte bound, or any other owner rule this dialect never reads is
	// refused here rather than emitted. The digest return is the true
	// omit-self identity the final record claims.
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		failure, ferr := failInvalid("create params are not a valid provider identity")
		if ferr != nil {
			return nil, ferr
		}
		return nil, failure
	}
	framed := []byte(`"record_id":"` + placeholderDigestID + `"`)
	claimed := []byte(`"record_id":"` + digest.String() + `"`)
	return bytes.Replace(staged, framed, claimed, 1), nil
}

// checkCreateOpaque enforces the explicit adapter-data surface on
// creation params: an object of at most 32 entries, keyed by the
// provider-identity key grammar, with non-secret string values of
// 1..1,024 characters that MUST NOT begin with an absolute path.
// Keys iterate sorted so a params value violating several entries
// always names the same rule.
func checkCreateOpaque(opaque map[string]string) error {
	if len(opaque) > 32 {
		failure, err := failInvalid("create opaque_identity exceeds 32 entries")
		if err != nil {
			return err
		}
		return failure
	}
	keys := make([]string, 0, len(opaque))
	for key := range opaque {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !identityKeyPattern.MatchString(key) {
			failure, err := failInvalid("create opaque key is not a provider key")
			if err != nil {
				return err
			}
			return failure
		}
		value := opaque[key]
		if runeLength(value) < 1 || runeLength(value) > 1024 {
			failure, err := failInvalid("create opaque value is not 1..1024 characters")
			if err != nil {
				return err
			}
			return failure
		}
		if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || windowsDrivePattern.MatchString(value) {
			failure, err := failInvalid("create opaque value begins with an absolute path")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// checkCreateExtensions enforces the extensions count and key rules on
// creation params: at most 64 entries keyed by 3..253-character
// lowercase reverse-DNS names. Value depth and the object byte bound
// belong to the owner's backstop in CreateIdentity, which refuses what
// these arms cannot see. Keys iterate sorted so a params value
// violating several entries always names the same rule.
func checkCreateExtensions(extensions map[string]any) error {
	if len(extensions) > maxExtensionMembers {
		failure, err := failInvalid("create extensions exceed 64 entries")
		if err != nil {
			return err
		}
		return failure
	}
	keys := make([]string, 0, len(extensions))
	for key := range extensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(key) < 3 || len(key) > 253 || !reverseDNSPattern.MatchString(key) {
			failure, err := failInvalid("create extension key is not reverse-DNS")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// opaqueJSON renders the opaque map for the staged record: a nil map
// means the empty object, never null.
func opaqueJSON(opaque map[string]string) map[string]any {
	rendered := make(map[string]any, len(opaque))
	for key, value := range opaque {
		rendered[key] = value
	}
	return rendered
}

// extensionsJSON renders the extensions map for the staged record: a
// nil map means the empty object, never null.
func extensionsJSON(extensions map[string]any) map[string]any {
	if extensions == nil {
		return map[string]any{}
	}
	return extensions
}
