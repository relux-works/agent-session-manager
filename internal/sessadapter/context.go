package sessadapter

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the reusable closed body types Section 7.8
// defines: AdapterCallContext with its request-digest binding,
// ReadAuthority, ObjectAuthority with the fresh-sink rule,
// SourceSelector with its exactly-one rule, AdapterFinding,
// ResourceLimits, and CapturePlanItem.

// contextMembers is the exact AdapterCallContext member set: the
// operation UUID, provider, environment, manifest and executable
// digests, request digest, and extensions.
var contextMembers = map[string]bool{
	"operation_id":                    true,
	"provider_id":                     true,
	"environment":                     true,
	"session_adapter_manifest_digest": true,
	"executable_sha256":               true,
	"request_digest":                  true,
	"extensions":                      true,
}

// contextRequired lists contextMembers in a fixed order.
var contextRequired = []string{
	"operation_id",
	"provider_id",
	"environment",
	"session_adapter_manifest_digest",
	"executable_sha256",
	"request_digest",
	"extensions",
}

// CallContext is one validated AdapterCallContext. The raw member
// is retained so the success echo can be compared byte-for-byte:
// the adapter must echo the context exactly, not rebuild it.
type CallContext struct {
	OperationID      string
	ProviderID       string
	Environment      Tuple
	ManifestDigest   string
	ExecutableSHA256 string
	RequestDigest    string
	raw              []byte
}

// Raw returns the canonical context bytes for echo comparison.
func (context CallContext) Raw() []byte {
	return append([]byte(nil), context.raw...)
}

// DecodeCallContext validates one closed AdapterCallContext.
func DecodeCallContext(raw json.RawMessage) (CallContext, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("call context "+fault.detail, fault.member)
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	if name, unknown := unknownMember(members, contextMembers); unknown {
		failure, err := failProtocol("call context carries unknown member", name)
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	if name, missing := missingMember(members, contextRequired); missing {
		failure, err := failProtocol("call context misses a required member", name)
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	operationID, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		failure, err := failProtocol("call context operation identifier is not a UUIDv7", "operation_id")
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	providerID, ok := rawString(members["provider_id"])
	if !ok || !providerIDPattern.MatchString(providerID) {
		failure, err := failProtocol("call context provider identifier is not a provider-id", "provider_id")
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	environment, err := DecodeTuple(members["environment"])
	if err != nil {
		return CallContext{}, err
	}
	manifestDigest, ok := checkDigest(members["session_adapter_manifest_digest"])
	if !ok {
		failure, faultErr := failProtocol("call context manifest digest is not a digest", "session_adapter_manifest_digest")
		if faultErr != nil {
			return CallContext{}, faultErr
		}
		return CallContext{}, failure
	}
	executable, ok := checkDigest(members["executable_sha256"])
	if !ok {
		failure, faultErr := failProtocol("call context executable digest is not a digest", "executable_sha256")
		if faultErr != nil {
			return CallContext{}, faultErr
		}
		return CallContext{}, failure
	}
	requestDigest, ok := checkDigest(members["request_digest"])
	if !ok {
		failure, faultErr := failProtocol("call context request digest is not a digest", "request_digest")
		if faultErr != nil {
			return CallContext{}, faultErr
		}
		return CallContext{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, faultErr := failProtocol("call context extensions are not reverse-DNS keyed", "extensions")
		if faultErr != nil {
			return CallContext{}, faultErr
		}
		return CallContext{}, failure
	}
	canonical, err := canonicaljson.Canonicalize(bytesTrimSpace(raw))
	if err != nil {
		failure, faultErr := failProtocol("call context is not canonical JSON", "")
		if faultErr != nil {
			return CallContext{}, faultErr
		}
		return CallContext{}, failure
	}
	return CallContext{
		OperationID:      operationID.String(),
		ProviderID:       providerID,
		Environment:      environment,
		ManifestDigest:   manifestDigest.String(),
		ExecutableSHA256: executable.String(),
		RequestDigest:    requestDigest.String(),
		raw:              canonical,
	}, nil
}

// RequestDigestFor computes the request digest the host binds into
// the call context: the SHA-256 of the JCS request body with only
// the context's request_digest member omitted. The omission is
// what makes the digest computable: the host computes it over the
// body it is about to send, writes it into the context, and sends;
// reinsertion cannot change the digest because the digested form
// never contained the member. A body with no context member
// digests whole.
func RequestDigestFor(requestBody []byte) (scalar.Digest, error) {
	canonical, err := canonicaljson.Canonicalize(requestBody)
	if err != nil {
		return scalar.Digest{}, err
	}
	var body map[string]any
	if err := json.Unmarshal(canonical, &body); err != nil {
		return scalar.Digest{}, err
	}
	if context, ok := body["context"].(map[string]any); ok {
		delete(context, "request_digest")
		redacted, err := json.Marshal(body)
		if err != nil {
			return scalar.Digest{}, err
		}
		canonical, err = canonicaljson.Canonicalize(redacted)
		if err != nil {
			return scalar.Digest{}, err
		}
	}
	return scalar.SHA256Digest(canonical), nil
}

// VerifyRequestDigest recomputes the digest of the canonical request
// body the host sent and requires it to equal the context's
// request_digest: the adapter answers for exactly this body, not
// for a neighbouring call. A changed body under a replayed context
// is a protocol error, never a retried mutation.
func VerifyRequestDigest(context CallContext, requestBody []byte) error {
	digest, err := RequestDigestFor(requestBody)
	if err != nil {
		failure, faultErr := failProtocol("call request body is not canonical JSON", "body")
		if faultErr != nil {
			return faultErr
		}
		return failure
	}
	if digest.String() != context.RequestDigest {
		failure, err := failProtocol("call context request digest does not match the request body", "request_digest")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// CheckContextEcho requires the success body to carry the request
// context byte-for-byte: Section 7.8 states every success echoes
// the context exactly. A rebuilt context — even a semantically
// equal one — is a protocol error, because only byte identity
// proves the adapter answered this call. DecodeCallContext
// already returns the canonical bytes, so the comparison reuses
// them: a second canonicalization of the same bytes could only
// repeat its verdict, and an arm that cannot fire is not
// carried.
func CheckContextEcho(sent CallContext, echoed json.RawMessage) (CallContext, error) {
	received, err := DecodeCallContext(echoed)
	if err != nil {
		return CallContext{}, err
	}
	if !bytes.Equal(received.raw, sent.raw) {
		failure, err := failProtocol("success context does not echo the request context byte-for-byte", "context")
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	return received, nil
}

// readAuthorityMembers is the exact ReadAuthority member set:
// authority_id, purpose, root_handle_names, expires_at, and
// extensions. Handle names identify host-opened descriptors
// delivered out of band; they are never paths.
var readAuthorityMembers = map[string]bool{
	"authority_id":      true,
	"purpose":           true,
	"root_handle_names": true,
	"expires_at":        true,
	"extensions":        true,
}

// readAuthorityRequired lists readAuthorityMembers in order.
var readAuthorityRequired = []string{
	"authority_id",
	"purpose",
	"root_handle_names",
	"expires_at",
	"extensions",
}

// ReadAuthority is one validated read authority.
type ReadAuthority struct {
	AuthorityID string
	Purpose     string
	HandleNames []string
	ExpiresAt   string
}

// DecodeReadAuthority validates one closed ReadAuthority: the
// purpose in source_native|target_staged|target_live and 1..128
// sorted unique handle names of 1..128 characters each.
func DecodeReadAuthority(raw json.RawMessage) (ReadAuthority, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("read authority "+fault.detail, fault.member)
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	if name, unknown := unknownMember(members, readAuthorityMembers); unknown {
		failure, err := failProtocol("read authority carries unknown member", name)
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	if name, missing := missingMember(members, readAuthorityRequired); missing {
		failure, err := failProtocol("read authority misses a required member", name)
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	authorityID, ok := checkUUIDv7(members["authority_id"])
	if !ok {
		failure, err := failProtocol("read authority identifier is not a UUIDv7", "authority_id")
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	purpose, ok := rawString(members["purpose"])
	if !ok || !validReadPurpose(purpose) {
		failure, err := failProtocol("read authority purpose is outside source_native|target_staged|target_live", "purpose")
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	handles, ok := checkSortedUniqueStrings(members["root_handle_names"], 1, 128, 1, 128)
	if !ok {
		failure, err := failProtocol("read authority handle names are not sorted unique string[1..128][1..128]", "root_handle_names")
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	expiresAt, ok := checkTimestamp(members["expires_at"])
	if !ok {
		failure, err := failProtocol("read authority expiry is not a timestamp", "expires_at")
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("read authority extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return ReadAuthority{}, err
		}
		return ReadAuthority{}, failure
	}
	return ReadAuthority{
		AuthorityID: authorityID.String(),
		Purpose:     purpose,
		HandleNames: handles,
		ExpiresAt:   expiresAt.String(),
	}, nil
}

// objectAuthorityMembers is the exact ObjectAuthority member set:
// authority_id, purpose, mode, max_objects, max_total_bytes, and
// extensions.
var objectAuthorityMembers = map[string]bool{
	"authority_id":    true,
	"purpose":         true,
	"mode":            true,
	"max_objects":     true,
	"max_total_bytes": true,
	"extensions":      true,
}

// objectAuthorityRequired lists objectAuthorityMembers in order.
var objectAuthorityRequired = []string{
	"authority_id",
	"purpose",
	"mode",
	"max_objects",
	"max_total_bytes",
	"extensions",
}

// ObjectAuthority is one validated object authority.
type ObjectAuthority struct {
	AuthorityID   string
	Purpose       string
	Mode          string
	MaxObjects    uint64
	MaxTotalBytes uint64
}

// DecodeObjectAuthority validates one closed ObjectAuthority: the
// purpose in the six-purpose vocabulary, the mode in
// read|fresh_sink, and the two limits as uint53. fresh_sink with a
// zero limit is refused here; whether the sink is empty is host
// state the caller asserts through CheckFreshSink.
func DecodeObjectAuthority(raw json.RawMessage) (ObjectAuthority, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("object authority "+fault.detail, fault.member)
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	if name, unknown := unknownMember(members, objectAuthorityMembers); unknown {
		failure, err := failProtocol("object authority carries unknown member", name)
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	if name, missing := missingMember(members, objectAuthorityRequired); missing {
		failure, err := failProtocol("object authority misses a required member", name)
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	authorityID, ok := checkUUIDv7(members["authority_id"])
	if !ok {
		failure, err := failProtocol("object authority identifier is not a UUIDv7", "authority_id")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	purpose, ok := rawString(members["purpose"])
	if !ok || !validObjectPurpose(purpose) {
		failure, err := failProtocol("object authority purpose is outside the six-purpose vocabulary", "purpose")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	mode, ok := rawString(members["mode"])
	if !ok || !validObjectMode(mode) {
		failure, err := failProtocol("object authority mode is outside read|fresh_sink", "mode")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	maxObjects, ok := rawUint53(members["max_objects"])
	if !ok {
		failure, err := failProtocol("object authority object limit is not a uint53", "max_objects")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	maxTotalBytes, ok := rawUint53(members["max_total_bytes"])
	if !ok {
		failure, err := failProtocol("object authority byte limit is not a uint53", "max_total_bytes")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	if mode == "fresh_sink" && (maxObjects == 0 || maxTotalBytes == 0) {
		failure, err := failProtocol("object authority fresh sink requires both limits above zero", "mode")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("object authority extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return ObjectAuthority{}, err
		}
		return ObjectAuthority{}, failure
	}
	return ObjectAuthority{
		AuthorityID:   authorityID.String(),
		Purpose:       purpose,
		Mode:          mode,
		MaxObjects:    maxObjects,
		MaxTotalBytes: maxTotalBytes,
	}, nil
}

// readPurposes is the three-purpose read-authority vocabulary
// Section 7.8 states. Membership is a table loop so the census
// derives it like every other closed vocabulary.
var readPurposes = []string{
	"source_native",
	"target_staged",
	"target_live",
}

// validReadPurpose reports whether the name is one of the three
// read-authority purposes Section 7.8 states.
func validReadPurpose(purpose string) bool {
	for _, allowed := range readPurposes {
		if purpose == allowed {
			return true
		}
	}
	return false
}

// findingSeverities is the three-severity finding vocabulary
// Section 7.8 states. Membership is a table loop so the census
// derives it like every other closed vocabulary.
var findingSeverities = []string{
	"info",
	"warning",
	"error",
}

// validFindingSeverity reports whether the name is one of the three
// finding severities Section 7.8 states.
func validFindingSeverity(severity string) bool {
	for _, allowed := range findingSeverities {
		if severity == allowed {
			return true
		}
	}
	return false
}

// objectPurposes is the six-purpose object-authority vocabulary
// Section 7.8 states. Membership is a table loop so the census
// derives it like every other closed vocabulary.
var objectPurposes = []string{
	"capture_plan",
	"raw_source",
	"canonical_source",
	"projection_plan",
	"projected_target",
	"read_back_evidence",
}

// validObjectPurpose reports whether the name is one of the six
// object-authority purposes Section 7.8 states.
func validObjectPurpose(purpose string) bool {
	for _, allowed := range objectPurposes {
		if purpose == allowed {
			return true
		}
	}
	return false
}

// objectModes is the two-mode object-authority vocabulary Section
// 7.8 states: read plus fresh_sink. Membership is a table loop so
// the census derives it like every other closed vocabulary: an
// inline comparison chain here would be invisible to the census,
// and a third mode would silently disable both CheckFreshSink and
// the fresh-sink limits rule below.
var objectModes = []string{
	"read",
	"fresh_sink",
}

// validObjectMode reports whether the mode is one of the two
// Section 7.8 object-authority modes.
func validObjectMode(mode string) bool {
	for _, allowed := range objectModes {
		if mode == allowed {
			return true
		}
	}
	return false
}

// CheckFreshSink enforces the fresh_sink half of the object
// authority: a fresh sink requires an empty host-created sink, so
// the caller states what it observed and the gate refuses a
// non-empty sink outright. A reused sink under a fresh_sink
// authority would mix two calls' writes; read mode exposes only
// request-named objects and takes no sink claim.
func CheckFreshSink(authority ObjectAuthority, sinkEmpty bool) error {
	if authority.Mode != "fresh_sink" {
		return nil
	}
	if !sinkEmpty {
		failure, err := failProtocol("object authority fresh sink is not empty", "mode")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// sourceSelectorMembers is the exact SourceSelector member set:
// native_session_id, logical_workspace_id, and opaque_source_ref,
// each nullable, with exactly one non-null.
var sourceSelectorMembers = map[string]bool{
	"native_session_id":    true,
	"logical_workspace_id": true,
	"opaque_source_ref":    true,
}

// sourceSelectorRequired lists sourceSelectorMembers in order.
var sourceSelectorRequired = []string{
	"native_session_id",
	"logical_workspace_id",
	"opaque_source_ref",
}

// SourceSelector is one validated source selector.
type SourceSelector struct {
	NativeSessionID    *string
	LogicalWorkspaceID *string
	OpaqueSourceRef    *string
}

// DecodeSourceSelector validates one closed SourceSelector with
// exactly one non-null member: zero names no source, two name an
// ambiguous one, and ambiguity is refused here rather than
// resolved by preference order.
func DecodeSourceSelector(raw json.RawMessage) (SourceSelector, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("source selector "+fault.detail, fault.member)
		if err != nil {
			return SourceSelector{}, err
		}
		return SourceSelector{}, failure
	}
	if name, unknown := unknownMember(members, sourceSelectorMembers); unknown {
		failure, err := failProtocol("source selector carries unknown member", name)
		if err != nil {
			return SourceSelector{}, err
		}
		return SourceSelector{}, failure
	}
	if name, missing := missingMember(members, sourceSelectorRequired); missing {
		failure, err := failProtocol("source selector misses a required member", name)
		if err != nil {
			return SourceSelector{}, err
		}
		return SourceSelector{}, failure
	}
	var selector SourceSelector
	if !isNull(members["native_session_id"]) {
		value, ok := checkStringBounds(members["native_session_id"], 1, 512)
		if !ok {
			failure, err := failProtocol("source selector native session identifier is not a string[1..512]", "native_session_id")
			if err != nil {
				return SourceSelector{}, err
			}
			return SourceSelector{}, failure
		}
		selector.NativeSessionID = &value
	}
	if !isNull(members["logical_workspace_id"]) {
		value, ok := checkUUIDv7(members["logical_workspace_id"])
		if !ok {
			failure, err := failProtocol("source selector logical workspace identifier is not a UUIDv7", "logical_workspace_id")
			if err != nil {
				return SourceSelector{}, err
			}
			return SourceSelector{}, failure
		}
		text := value.String()
		selector.LogicalWorkspaceID = &text
	}
	if !isNull(members["opaque_source_ref"]) {
		value, ok := checkStringBounds(members["opaque_source_ref"], 1, 512)
		if !ok {
			failure, err := failProtocol("source selector opaque reference is not a string[1..512]", "opaque_source_ref")
			if err != nil {
				return SourceSelector{}, err
			}
			return SourceSelector{}, failure
		}
		selector.OpaqueSourceRef = &value
	}
	count := 0
	if selector.NativeSessionID != nil {
		count++
	}
	if selector.LogicalWorkspaceID != nil {
		count++
	}
	if selector.OpaqueSourceRef != nil {
		count++
	}
	if count != 1 {
		failure, err := failProtocol("source selector does not name exactly one source", "native_session_id")
		if err != nil {
			return SourceSelector{}, err
		}
		return SourceSelector{}, failure
	}
	return selector, nil
}

// resourceLimitsMembers is the exact ResourceLimits member set.
var resourceLimitsMembers = map[string]bool{
	"max_objects":             true,
	"max_total_bytes":         true,
	"max_single_object_bytes": true,
	"max_events":              true,
	"max_target_resources":    true,
}

// resourceLimitsRequired lists resourceLimitsMembers in order.
var resourceLimitsRequired = []string{
	"max_objects",
	"max_total_bytes",
	"max_single_object_bytes",
	"max_events",
	"max_target_resources",
}

// ResourceLimits is one validated projection resource bound.
type ResourceLimits struct {
	MaxObjects           uint64
	MaxTotalBytes        uint64
	MaxSingleObjectBytes uint64
	MaxEvents            uint64
	MaxTargetResources   uint64
}

// DecodeResourceLimits validates one closed ResourceLimits: every
// per-item maximum above zero, the event and resource counts in
// [0..65536], and each per-item maximum no greater than the total.
// The last rule is checked in both directions — a per-item maximum
// above the total would license an object the total cannot hold,
// and a total of zero would license nothing at all.
func DecodeResourceLimits(raw json.RawMessage) (ResourceLimits, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("resource limits "+fault.detail, fault.member)
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	if name, unknown := unknownMember(members, resourceLimitsMembers); unknown {
		failure, err := failProtocol("resource limits carry unknown member", name)
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	if name, missing := missingMember(members, resourceLimitsRequired); missing {
		failure, err := failProtocol("resource limits miss a required member", name)
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	maxObjects, ok := checkUint53Bounds(members["max_objects"], 1, maxUint53)
	if !ok {
		failure, err := failProtocol("resource object limit is not a uint53 above zero", "max_objects")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	maxTotalBytes, ok := checkUint53Bounds(members["max_total_bytes"], 1, maxUint53)
	if !ok {
		failure, err := failProtocol("resource total byte limit is not a uint53 above zero", "max_total_bytes")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	maxSingle, ok := checkUint53Bounds(members["max_single_object_bytes"], 1, maxUint53)
	if !ok {
		failure, err := failProtocol("resource single-object byte limit is not a uint53 above zero", "max_single_object_bytes")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	maxEvents, ok := checkUint53Bounds(members["max_events"], 0, 65536)
	if !ok {
		failure, err := failProtocol("resource event limit is not a uint53 in [0..65536]", "max_events")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	maxTargetResources, ok := checkUint53Bounds(members["max_target_resources"], 0, 65536)
	if !ok {
		failure, err := failProtocol("resource target limit is not a uint53 in [0..65536]", "max_target_resources")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	if maxObjects > maxTotalBytes {
		failure, err := failProtocol("resource object limit exceeds the total byte limit", "max_objects")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	if maxSingle > maxTotalBytes {
		failure, err := failProtocol("resource single-object byte limit exceeds the total byte limit", "max_single_object_bytes")
		if err != nil {
			return ResourceLimits{}, err
		}
		return ResourceLimits{}, failure
	}
	return ResourceLimits{
		MaxObjects:           maxObjects,
		MaxTotalBytes:        maxTotalBytes,
		MaxSingleObjectBytes: maxSingle,
		MaxEvents:            maxEvents,
		MaxTargetResources:   maxTargetResources,
	}, nil
}

// findingMembers is the exact AdapterFinding member set: severity,
// code, message, remediation, and extensions.
var findingMembers = map[string]bool{
	"severity":    true,
	"code":        true,
	"message":     true,
	"remediation": true,
	"extensions":  true,
}

// findingRequired lists findingMembers in order.
var findingRequired = []string{
	"severity",
	"code",
	"message",
	"remediation",
	"extensions",
}

// Finding is one validated adapter finding.
type Finding struct {
	Severity    string
	Code        string
	Message     string
	Remediation *string
}

// DecodeFinding validates one closed AdapterFinding: the severity
// in info|warning|error, the code as string[1..128], the message as
// string[1..4096], and the remediation as null or
// string[1..4096].
func DecodeFinding(raw json.RawMessage) (Finding, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("adapter finding "+fault.detail, fault.member)
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	if name, unknown := unknownMember(members, findingMembers); unknown {
		failure, err := failProtocol("adapter finding carries unknown member", name)
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	if name, missing := missingMember(members, findingRequired); missing {
		failure, err := failProtocol("adapter finding misses a required member", name)
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	severity, ok := rawString(members["severity"])
	if !ok || !validFindingSeverity(severity) {
		failure, err := failProtocol("adapter finding severity is outside info|warning|error", "severity")
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	code, ok := checkStringBounds(members["code"], 1, 128)
	if !ok {
		failure, err := failProtocol("adapter finding code is not a string[1..128]", "code")
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	message, ok := checkStringBounds(members["message"], 1, 4096)
	if !ok {
		failure, err := failProtocol("adapter finding message is not a string[1..4096]", "message")
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	var remediation *string
	if !isNull(members["remediation"]) {
		text, ok := checkStringBounds(members["remediation"], 1, 4096)
		if !ok {
			failure, err := failProtocol("adapter finding remediation is not a string[1..4096]", "remediation")
			if err != nil {
				return Finding{}, err
			}
			return Finding{}, failure
		}
		remediation = &text
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("adapter finding extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return Finding{}, err
		}
		return Finding{}, failure
	}
	return Finding{
		Severity:    severity,
		Code:        code,
		Message:     message,
		Remediation: remediation,
	}, nil
}

// DecodeFindings validates a findings array in the count bound:
// every element a closed AdapterFinding.
func DecodeFindings(raw json.RawMessage, maximum uint64) ([]Finding, error) {
	elements, ok := decodeArray(bytesTrimSpace(raw))
	if !ok {
		failure, err := failProtocol("adapter findings are not an array", "findings")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if uint64(len(elements)) > maximum {
		failure, err := failProtocol("adapter findings exceed the count bound", "findings")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	findings := make([]Finding, 0, len(elements))
	for _, element := range elements {
		finding, err := DecodeFinding(element)
		if err != nil {
			return nil, err
		}
		findings = append(findings, finding)
	}
	return findings, nil
}

// capturePlanItemMembers is the exact CapturePlanItem member set.
var capturePlanItemMembers = map[string]bool{
	"native_item_key": true,
	"class":           true,
	"byte_count":      true,
	"required":        true,
	"extensions":      true,
}

// capturePlanItemRequired lists capturePlanItemMembers in order.
var capturePlanItemRequired = []string{
	"native_item_key",
	"class",
	"byte_count",
	"required",
	"extensions",
}

// CapturePlanItem is one validated capture-plan candidate row.
type CapturePlanItem struct {
	NativeItemKey string
	Class         string
	ByteCount     *uint64
	Required      bool
}

// DecodeCapturePlanItem validates one closed CapturePlanItem: the
// key as string[1..512], the class in the nine-class vocabulary,
// and the byte count as null or uint53.
func DecodeCapturePlanItem(raw json.RawMessage) (CapturePlanItem, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("capture plan item "+fault.detail, fault.member)
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	if name, unknown := unknownMember(members, capturePlanItemMembers); unknown {
		failure, err := failProtocol("capture plan item carries unknown member", name)
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	if name, missing := missingMember(members, capturePlanItemRequired); missing {
		failure, err := failProtocol("capture plan item misses a required member", name)
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	key, ok := checkStringBounds(members["native_item_key"], 1, 512)
	if !ok {
		failure, err := failProtocol("capture plan item key is not a string[1..512]", "native_item_key")
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	class, ok := rawString(members["class"])
	if !ok || !validCaptureClass(class) {
		failure, err := failProtocol("capture plan item class is outside the nine-class vocabulary", "class")
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	var byteCount *uint64
	if !isNull(members["byte_count"]) {
		count, ok := rawUint53(members["byte_count"])
		if !ok {
			failure, err := failProtocol("capture plan item byte count is not a uint53", "byte_count")
			if err != nil {
				return CapturePlanItem{}, err
			}
			return CapturePlanItem{}, failure
		}
		byteCount = &count
	}
	required, ok := rawBool(members["required"])
	if !ok {
		failure, err := failProtocol("capture plan item required flag is not a boolean", "required")
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("capture plan item extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return CapturePlanItem{}, err
		}
		return CapturePlanItem{}, failure
	}
	return CapturePlanItem{
		NativeItemKey: key,
		Class:         class,
		ByteCount:     byteCount,
		Required:      required,
	}, nil
}
