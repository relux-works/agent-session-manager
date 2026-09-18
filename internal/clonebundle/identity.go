package clonebundle

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Section 13.14.1 identities: NativeIdentity,
// WorkspaceBinding, and the sanitized native-key admission shared by
// every native key in this package. Sanitized means printable,
// relative, and unambiguously native: never an absolute source path,
// never an AX Session identity, never carrying embedded credentials.

var nativeIdentityMembers = map[string]bool{
	"native_session_id":         true,
	"identity_kind":             true,
	"logical_workspace_id":      true,
	"backend_realm_fingerprint": true,
	"opaque_identity":           true,
	"extensions":                true,
}

var nativeIdentityRequired = []string{
	"native_session_id",
	"identity_kind",
	"logical_workspace_id",
	"backend_realm_fingerprint",
	"opaque_identity",
	"extensions",
}

var nativeIdentityKinds = []string{
	"provider_native",
	"official_import",
	"continuation_context",
}

func validNativeIdentityKind(kind string) bool {
	for _, allowed := range nativeIdentityKinds {
		if kind == allowed {
			return true
		}
	}
	return false
}

// NativeIdentity is one validated sanitized native identity.
type NativeIdentity struct {
	NativeSessionID         string
	IdentityKind            string
	LogicalWorkspaceID      scalar.UUIDv7
	BackendRealmFingerprint *scalar.Digest
	OpaqueIdentity          *string
}

// DecodeNativeIdentity validates one closed NativeIdentity: exact
// members, the closed identity-kind vocabulary, a sanitized native
// session ID, a UUIDv7 logical workspace, an optional digest realm
// fingerprint, an optional sanitized opaque identity, and
// reverse-DNS extensions. Credential, absolute-path, PID, socket,
// token, and secret-environment material is refused by the shared
// sanitizer, never laundered into an identity.
func DecodeNativeIdentity(raw json.RawMessage) (NativeIdentity, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return NativeIdentity{}, invalid("native identity %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, nativeIdentityMembers); unknown {
		return NativeIdentity{}, invalid("native identity carries unknown member %q", name)
	}
	if name, missing := missingMember(members, nativeIdentityRequired); missing {
		return NativeIdentity{}, invalid("native identity misses a required member %q", name)
	}
	nativeID, ok := checkStringBounds(members["native_session_id"], 1, 512)
	if !ok {
		return NativeIdentity{}, invalid("native identity native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(nativeID); err != nil {
		return NativeIdentity{}, err
	}
	kind, ok := rawString(members["identity_kind"])
	if !ok || !validNativeIdentityKind(kind) {
		return NativeIdentity{}, invalid("native identity kind is outside provider_native|official_import|continuation_context")
	}
	workspaceID, ok := checkUUIDv7(members["logical_workspace_id"])
	if !ok {
		return NativeIdentity{}, invalid("native identity logical_workspace_id is not a UUIDv7")
	}
	var fingerprint *scalar.Digest
	if !isNull(members["backend_realm_fingerprint"]) {
		digest, ok := checkDigest(members["backend_realm_fingerprint"])
		if !ok {
			return NativeIdentity{}, invalid("native identity backend_realm_fingerprint is not a digest")
		}
		fingerprint = &digest
	}
	var opaque *string
	if !isNull(members["opaque_identity"]) {
		value, ok := checkStringBounds(members["opaque_identity"], 1, 512)
		if !ok {
			return NativeIdentity{}, invalid("native identity opaque_identity is not a string[1..512]")
		}
		if err := SanitizeNativeKey(value); err != nil {
			return NativeIdentity{}, err
		}
		opaque = &value
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return NativeIdentity{}, invalid("native identity extensions %s", extensionsFault)
	}
	return NativeIdentity{
		NativeSessionID:         nativeID,
		IdentityKind:            kind,
		LogicalWorkspaceID:      workspaceID,
		BackendRealmFingerprint: fingerprint,
		OpaqueIdentity:          opaque,
	}, nil
}

// EncodeNativeIdentity renders one validated identity as canonical
// bytes under the given extensions. The native session ID and the
// opaque identity are sanitized here, before marshaling —
// encoding/json would otherwise rewrite invalid UTF-8 to U+FFFD and
// the caller's re-decode would only ever see the rewritten bytes,
// laundering a native key past the sanitizer's own refusal and
// sealing two distinct inputs to byte-identical objects. The kind
// refuses invalid UTF-8 here too, so no member the encoder seals is
// ever rewritten. The extensions are validated here as well, so
// encoding never admits what decoding refused. A nil extensions map
// encodes as the empty object.
func EncodeNativeIdentity(identity NativeIdentity, extensions map[string]any) ([]byte, error) {
	if !validText(identity.IdentityKind) {
		return nil, invalid("native identity kind is not valid UTF-8")
	}
	if err := SanitizeNativeKey(identity.NativeSessionID); err != nil {
		return nil, err
	}
	if identity.OpaqueIdentity != nil {
		if err := SanitizeNativeKey(*identity.OpaqueIdentity); err != nil {
			return nil, err
		}
	}
	if _, err := encodeExtensions(extensions); err != nil {
		return nil, err
	}
	var fingerprint any
	if identity.BackendRealmFingerprint != nil {
		fingerprint = identity.BackendRealmFingerprint.String()
	}
	var opaque any
	if identity.OpaqueIdentity != nil {
		opaque = *identity.OpaqueIdentity
	}
	if extensions == nil {
		extensions = map[string]any{}
	}
	return canonicalizeObject(map[string]any{
		"native_session_id":         identity.NativeSessionID,
		"identity_kind":             identity.IdentityKind,
		"logical_workspace_id":      identity.LogicalWorkspaceID.String(),
		"backend_realm_fingerprint": fingerprint,
		"opaque_identity":           opaque,
		"extensions":                extensions,
	})
}

var workspaceBindingMembers = map[string]bool{
	"logical_workspace_id":           true,
	"cwd_relative":                   true,
	"repository_remote_fingerprints": true,
	"branch":                         true,
	"head_digest":                    true,
	"index_digest":                   true,
	"working_tree_digest":            true,
	"extensions":                     true,
}

var workspaceBindingRequired = []string{
	"logical_workspace_id",
	"cwd_relative",
	"repository_remote_fingerprints",
	"branch",
	"head_digest",
	"index_digest",
	"working_tree_digest",
	"extensions",
}

// WorkspaceBinding is one validated closed workspace binding. No
// member grants filesystem authority: the cwd is a normalized
// relative path beneath the Workspace Group root, never an absolute
// path or an escape.
type WorkspaceBinding struct {
	LogicalWorkspaceID scalar.UUIDv7
	CwdRelative        string
	RemoteFingerprints []scalar.Digest
	Branch             *string
	HeadDigest         *scalar.Digest
	IndexDigest        *scalar.Digest
	WorkingTreeDigest  *scalar.Digest
}

// DecodeWorkspaceBinding validates one closed WorkspaceBinding.
func DecodeWorkspaceBinding(raw json.RawMessage) (WorkspaceBinding, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return WorkspaceBinding{}, invalid("workspace binding %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, workspaceBindingMembers); unknown {
		return WorkspaceBinding{}, invalid("workspace binding carries unknown member %q", name)
	}
	if name, missing := missingMember(members, workspaceBindingRequired); missing {
		return WorkspaceBinding{}, invalid("workspace binding misses a required member %q", name)
	}
	workspaceID, ok := checkUUIDv7(members["logical_workspace_id"])
	if !ok {
		return WorkspaceBinding{}, invalid("workspace binding logical_workspace_id is not a UUIDv7")
	}
	cwd, ok := checkStringBounds(members["cwd_relative"], 1, 4096)
	if !ok {
		return WorkspaceBinding{}, invalid("workspace binding cwd_relative is not a string[1..4096]")
	}
	if err := checkCwdRelative(cwd); err != nil {
		return WorkspaceBinding{}, err
	}
	fingerprints, ok := checkSortedUniqueDigests(members["repository_remote_fingerprints"], 0, 128)
	if !ok {
		return WorkspaceBinding{}, invalid("workspace binding repository_remote_fingerprints are not sorted unique digest[0..128]")
	}
	var branch *string
	if !isNull(members["branch"]) {
		value, ok := checkStringBounds(members["branch"], 1, 1024)
		if !ok {
			return WorkspaceBinding{}, invalid("workspace binding branch is not a string[1..1024]")
		}
		branch = &value
	}
	head, err := checkNullableDigest(members, "head_digest", "workspace binding")
	if err != nil {
		return WorkspaceBinding{}, err
	}
	index, err := checkNullableDigest(members, "index_digest", "workspace binding")
	if err != nil {
		return WorkspaceBinding{}, err
	}
	tree, err := checkNullableDigest(members, "working_tree_digest", "workspace binding")
	if err != nil {
		return WorkspaceBinding{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return WorkspaceBinding{}, invalid("workspace binding extensions %s", extensionsFault)
	}
	return WorkspaceBinding{
		LogicalWorkspaceID: workspaceID,
		CwdRelative:        cwd,
		RemoteFingerprints: fingerprints,
		Branch:             branch,
		HeadDigest:         head,
		IndexDigest:        index,
		WorkingTreeDigest:  tree,
	}, nil
}

// checkCwdRelative admits exactly the normalized relative paths
// beneath (or equal to) the Workspace Group root: scalar relative
// paths plus the root itself ("."). Anything else — absolute,
// escaped, backslashed, encoded-separator, or dot-segment forms —
// would grant or smuggle filesystem authority and is refused.
func checkCwdRelative(cwd string) error {
	if cwd == "." {
		return nil
	}
	if _, err := scalar.ParseRelativePath(cwd); err != nil {
		return invalid("workspace binding cwd_relative is not normalized beneath the workspace root: %v", err)
	}
	return nil
}

func checkNullableDigest(members map[string]json.RawMessage, name, owner string) (*scalar.Digest, error) {
	raw, present := members[name]
	if !present {
		return nil, invalid("%s misses a required member %q", owner, name)
	}
	if isNull(raw) {
		return nil, nil
	}
	digest, ok := checkDigest(raw)
	if !ok {
		return nil, invalid("%s %s is not a digest", owner, name)
	}
	return &digest, nil
}

// SanitizeNativeKey refuses every native key that is not sanitized:
// empty values are refused here (over-long values by the caller's
// bound check), and this gate refuses control characters (the full
// Cc class, C0 and C1), invisible format characters (Cf, Zl, Zp:
// zero-width and separator marks that render invisibly),
// absolute-path forms, a UUIDv7 form (indistinguishable from a
// fabricated AX Session identity), embedded-credential forms (a
// password-style colon before the first @, with or without a
// scheme), and AX-schema markers in any case. The pinned text does
// not define the sanitizer's exact scope, so the implemented bound
// is exactly this list, pinned by the sanitizer corpus and stated
// in TRACEABILITY.md: ordinary spaces, @ without a password
// colon, and colons without an @ all admit. A native key that
// survives is printable, relative, and unambiguously native.
func SanitizeNativeKey(key string) error {
	if key == "" {
		return invalid("native key is empty")
	}
	if !utf8.ValidString(key) {
		return invalid("native key is not valid UTF-8")
	}
	for _, unit := range key {
		if unicode.IsControl(unit) {
			return invalid("native key carries control characters")
		}
		if unicode.Is(unicode.Cf, unit) || unicode.Is(unicode.Zl, unit) || unicode.Is(unicode.Zp, unit) {
			return invalid("native key carries invisible format characters")
		}
	}
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, `\\`) || hasDrivePrefix(key) {
		return invalid("native key is an absolute source path, never a cross-host identity")
	}
	if _, err := scalar.ParseUUIDv7(key); err == nil {
		return invalid("native key %q is a UUIDv7, indistinguishable from a fabricated AX Session identity", key)
	}
	if strings.Contains(strings.ToLower(key), "urn:ax:") {
		return invalid("native key carries an AX identity marker")
	}
	if at := strings.Index(key, "@"); at >= 0 && strings.Contains(key[:at], ":") {
		return invalid("native key carries embedded credentials")
	}
	return nil
}

func hasDrivePrefix(value string) bool {
	return len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
}

func memberField(member string) string {
	if member == "" {
		return "frame"
	}
	return "member " + member
}
