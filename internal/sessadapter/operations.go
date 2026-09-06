package sessadapter

import (
	"encoding/json"
)

// This file validates the fourteen Section 7.8 operation bodies:
// the closed request member set per operation with the inline
// scalar bounds, and the closed success member set with the
// semantic gates the table states (discover partial/cursor,
// capture-plan digest equality, validate mode coherence,
// resume-plan identity occurrence, projection-plan disposition
// rules).
//
// Nested objects owned by Sections 13.14 and 10.8
// (CaptureBoundary, NativeIdentity, StableSnapshotProof,
// CloneSourceSummary, FidelityCounts, WorkspaceBinding) are
// required to be present JSON objects and are never inspected
// further here: their content owners are the directory-node and
// boundary leaves of this Story. requireNestedObject is the one
// place that rule is enforced, so the bound is visible, not
// scattered.

// RefuseUnknownOperation refuses a dispatch name outside the
// closed registry with operation_unknown. Dispatch refuses
// locally, before any adapter surface is touched.
func RefuseUnknownOperation(name string) error {
	failure, err := failUnknownOperation("dispatch names an operation outside the closed registry", name)
	if err != nil {
		return err
	}
	return failure
}

// contextFreeOperations is the set of operations whose request
// bodies carry no context: manifest takes the empty object and
// probe takes only the expected provider facts. Membership is a
// table loop so the census derives it like every other closed
// vocabulary: an inline `operation != X && operation != Y` chain
// here would be invisible to the census, and a widened chain
// would silently skip context decoding for a third operation.
var contextFreeOperations = []Operation{
	OpManifest,
	OpProbe,
}

// operationSkipsRequestContext reports whether the operation's
// request body carries no context member.
func operationSkipsRequestContext(operation Operation) bool {
	for _, free := range contextFreeOperations {
		if operation == free {
			return true
		}
	}
	return false
}

// RefuseUnavailableOperation reports an operation the adapter does
// not implement with capability_unavailable. Every adapter
// implements every name; an unavailable one is refused, never
// emulated.
func RefuseUnavailableOperation(operation Operation) error {
	failure, err := failUnavailable("adapter does not implement the registry operation", string(operation))
	if err != nil {
		return err
	}
	return failure
}

// requestBodyMembers maps each operation to its exact request-body
// member set in Section 7.8 table order. The manifest body is the
// empty object.
var requestBodyMembers = map[Operation][]string{
	OpManifest:       {},
	OpProbe:          {"expected_provider_id", "expected_candidate_kind", "extensions"},
	OpDiscover:       {"context", "authority", "workspace_filter", "limit", "cursor", "extensions"},
	OpInspect:        {"context", "authority", "source", "extensions"},
	OpSnapshotProof:  {"context", "authority", "source", "expected_source_store_generation", "allow_provider_quiescence", "extensions"},
	OpCapturePlan:    {"context", "authority", "source", "capture_boundary", "plan_sink", "max_items", "max_total_bytes", "extensions"},
	OpCapture:        {"context", "source_authority", "sink", "source", "capture_boundary", "capture_plan_digest", "extensions"},
	OpNormalize:      {"context", "capture_manifest_id", "raw_objects", "canonical_sink", "extensions"},
	OpProjectionPlan: {"context", "capture_manifest_id", "canonical_session_id", "canonical_event_ids", "source_objects", "plan_sink", "target_environment", "expected_target_native_session_id", "fidelity_profile", "required_dispositions", "forbid_reasons", "resource_limits", "extensions"},
	OpProject:        {"context", "projection_plan_id", "capture_manifest_id", "canonical_session_id", "source_objects", "target_sink", "extensions"},
	OpReadBack:       {"context", "authority", "expected_target_native_session_id", "projection_plan_id", "projected_object_manifest_id", "evidence_sink", "extensions"},
	OpValidate:       {"context", "mode", "capture_manifest_id", "canonical_session_id", "projection_plan_id", "projected_object_manifest_id", "read_back_evidence_manifest_id", "expected_target_native_session_id", "extensions"},
	OpResumePlan:     {"context", "authority", "expected_target_native_session_id", "projection_plan_id", "target_checkpoint_id", "extensions"},
	OpDoctor:         {"context", "direction", "tuple_registry_digest", "refresh_requested", "extensions"},
}

// successBodyMembers maps each operation to its exact success-body
// member set. Manifest and probe successes decode through their
// dedicated decoders; the rest validate here.
var successBodyMembers = map[Operation][]string{
	OpManifest:       nil,
	OpProbe:          nil,
	OpDiscover:       {"context", "sources", "next_cursor", "partial", "extensions"},
	OpInspect:        {"context", "source", "source_identity", "source_store_generation", "environment", "ambiguities", "extensions"},
	OpSnapshotProof:  {"context", "proof", "provider_quiescence_requested", "provider_quiescence_observed", "extensions"},
	OpCapturePlan:    {"context", "source_identity", "source_store_generation", "capture_plan_candidate_id", "candidate_count", "excluded_classes", "capture_plan_digest", "extensions"},
	OpCapture:        {"context", "capture_plan_digest", "source_store_generation", "capture_result_candidate_id", "source_raw_object_manifest_candidate_id", "item_count", "pre_capture_digest", "post_capture_digest", "extensions"},
	OpNormalize:      {"context", "capture_manifest_id", "canonical_session_candidate_id", "canonical_event_candidate_ids", "raw_reference_ids", "extensions"},
	OpProjectionPlan: {"context", "projection_plan_candidate_id", "required_source_object_ids", "predicted_counts", "findings", "extensions"},
	OpProject:        {"context", "projection_plan_id", "projected_object_manifest_candidate_id", "created_resource_keys", "actual_counts", "extensions"},
	OpReadBack:       {"context", "observed_target_native_session_id", "projection_plan_id", "observed_environment", "parsed_event_count", "parsed_head_ids", "workspace_binding", "structural_digest", "read_back_evidence_manifest_candidate_id", "extensions"},
	OpValidate:       {"context", "mode", "valid", "structural_valid", "semantic_marker_valid", "identity_valid", "workspace_binding_valid", "resume_surface_valid", "findings", "evidence_digest", "extensions"},
	OpResumePlan:     {"context", "target_native_session_id", "projection_plan_id", "argv", "cwd_relative", "environment_names", "opens_existing_identity", "extensions"},
	OpDoctor:         nil,
}

// requestMemberSet builds the closed set for one operation's
// request body.
func requestMemberSet(operation Operation) map[string]bool {
	set := map[string]bool{}
	for _, name := range requestBodyMembers[operation] {
		set[name] = true
	}
	return set
}

// successMemberSet builds the closed set for one operation's
// success body.
func successMemberSet(operation Operation) map[string]bool {
	set := map[string]bool{}
	for _, name := range successBodyMembers[operation] {
		set[name] = true
	}
	return set
}

// requireNestedObject requires the member to be a present JSON
// object. Content owned by Sections 13.14 and 10.8 is never
// inspected here; presence plus object shape is the whole rule,
// and anything else — null, array, scalar — is a protocol error,
// never an absent result.
func requireNestedObject(members map[string]json.RawMessage, name string) error {
	if _, fault := decodeStrictObject(bytesTrimSpace(members[name])); fault != nil {
		failure, err := failProtocol("operation body nested object "+fault.detail, name)
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// requireDigest requires the member to be a digest.
func requireDigest(members map[string]json.RawMessage, name string) error {
	if _, ok := checkDigest(members[name]); !ok {
		failure, err := failProtocol("operation body digest member is not a digest", name)
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// requireStringBounds requires the member to be a string in the
// inclusive character bound.
func requireStringBounds(members map[string]json.RawMessage, name string, minimum, maximum int) error {
	if _, ok := checkStringBounds(members[name], minimum, maximum); !ok {
		failure, err := failProtocol("operation body string member is outside its bound", name)
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// requireUint53Bounds requires the member to be a uint53 in the
// inclusive bound. Both edges are enforced.
func requireUint53Bounds(members map[string]json.RawMessage, name string, minimum, maximum uint64) error {
	if _, ok := checkUint53Bounds(members[name], minimum, maximum); !ok {
		failure, err := failProtocol("operation body count member is outside its bound", name)
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// requireBool requires the member to be a boolean.
func requireBool(members map[string]json.RawMessage, name string) (bool, error) {
	value, ok := rawBool(members[name])
	if !ok {
		failure, err := failProtocol("operation body flag member is not a boolean", name)
		if err != nil {
			return false, err
		}
		return false, failure
	}
	return value, nil
}

// requireNullableDigest requires the member to be null or a
// digest.
func requireNullableDigest(members map[string]json.RawMessage, name string) error {
	if isNull(members[name]) {
		return nil
	}
	return requireDigest(members, name)
}

// requireNullableBool requires the member to be null or a
// boolean.
func requireNullableBool(members map[string]json.RawMessage, name string) error {
	if isNull(members[name]) {
		return nil
	}
	_, err := requireBool(members, name)
	return err
}

// CheckRequestBody validates one outbound request body for its
// operation: the exact member set, the call context where the
// operation carries one, and the inline scalar bounds Section 7.8
// states. The manifest body must be the empty object; an unknown
// operation is operation_unknown. The decoded context is returned
// for the digest binding and the call checks; manifest and probe
// carry no context and return the zero value.
func CheckRequestBody(operation Operation, body []byte) (CallContext, error) {
	if !validOperation(string(operation)) {
		failure, err := failUnknownOperation("request body names an operation outside the closed registry", string(operation))
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol("request body "+fault.detail, fault.member)
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	allowed := requestMemberSet(operation)
	if name, unknown := unknownMember(members, allowed); unknown {
		failure, err := failProtocol("request body carries unknown member", name)
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	if name, missing := missingMember(members, requestBodyMembers[operation]); missing {
		failure, err := failProtocol("request body misses a required member", name)
		if err != nil {
			return CallContext{}, err
		}
		return CallContext{}, failure
	}
	var context CallContext
	if !operationSkipsRequestContext(operation) {
		decoded, err := DecodeCallContext(members["context"])
		if err != nil {
			return CallContext{}, err
		}
		context = decoded
	}
	if err := checkRequestScalars(operation, members); err != nil {
		return CallContext{}, err
	}
	return context, nil
}

// checkRequestScalars validates the inline scalar members of one
// request body: authorities, selectors, limits, enums, and bounds.
// Nested Section 13.14/10.8 objects validate for presence only.
func checkRequestScalars(operation Operation, members map[string]json.RawMessage) error {
	switch operation {
	case OpManifest:
		return nil
	case OpProbe:
		if provider, ok := rawString(members["expected_provider_id"]); !ok || !providerIDPattern.MatchString(provider) {
			failure, err := failProtocol("probe request provider identifier is not a provider-id", "expected_provider_id")
			if err != nil {
				return err
			}
			return failure
		}
		if kind, ok := rawString(members["expected_candidate_kind"]); !ok || !validCandidateKind(CandidateKind(kind)) {
			failure, err := failProtocol("probe request candidate kind is outside builtin|external", "expected_candidate_kind")
			if err != nil {
				return err
			}
			return failure
		}
		if !checkExtensions(members["extensions"]) {
			failure, err := failProtocol("probe request extensions are not reverse-DNS keyed", "extensions")
			if err != nil {
				return err
			}
			return failure
		}
		return nil
	case OpDiscover:
		if _, err := DecodeReadAuthority(members["authority"]); err != nil {
			return err
		}
		if !isNull(members["workspace_filter"]) {
			if _, ok := checkUUIDv7(members["workspace_filter"]); !ok {
				failure, err := failProtocol("discover workspace filter is not a UUIDv7", "workspace_filter")
				if err != nil {
					return err
				}
				return failure
			}
		}
		if err := requireUint53Bounds(members, "limit", 1, 65536); err != nil {
			return err
		}
		if !isNull(members["cursor"]) {
			if err := requireStringBounds(members, "cursor", 1, 1024); err != nil {
				return err
			}
		}
	case OpInspect:
		if _, err := DecodeReadAuthority(members["authority"]); err != nil {
			return err
		}
		if _, err := DecodeSourceSelector(members["source"]); err != nil {
			return err
		}
	case OpSnapshotProof:
		if _, err := DecodeReadAuthority(members["authority"]); err != nil {
			return err
		}
		if _, err := DecodeSourceSelector(members["source"]); err != nil {
			return err
		}
		if err := requireStringBounds(members, "expected_source_store_generation", 1, 512); err != nil {
			return err
		}
		if _, err := requireBool(members, "allow_provider_quiescence"); err != nil {
			return err
		}
	case OpCapturePlan:
		if _, err := DecodeReadAuthority(members["authority"]); err != nil {
			return err
		}
		if _, err := DecodeSourceSelector(members["source"]); err != nil {
			return err
		}
		if err := requireNestedObject(members, "capture_boundary"); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["plan_sink"]); err != nil {
			return err
		}
		if err := requireUint53Bounds(members, "max_items", 1, 65536); err != nil {
			return err
		}
		if err := requireUint53Bounds(members, "max_total_bytes", 1, maxUint53); err != nil {
			return err
		}
	case OpCapture:
		if _, err := DecodeReadAuthority(members["source_authority"]); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["sink"]); err != nil {
			return err
		}
		if _, err := DecodeSourceSelector(members["source"]); err != nil {
			return err
		}
		if err := requireNestedObject(members, "capture_boundary"); err != nil {
			return err
		}
		if err := requireDigest(members, "capture_plan_digest"); err != nil {
			return err
		}
	case OpNormalize:
		if err := requireDigest(members, "capture_manifest_id"); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["raw_objects"]); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["canonical_sink"]); err != nil {
			return err
		}
	case OpProjectionPlan:
		if err := requireDigest(members, "capture_manifest_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "canonical_session_id"); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueDigests(members["canonical_event_ids"], 0, 65536); !ok {
			failure, faultErr := failProtocol("projection request canonical event identifiers are not sorted unique digest[0..65536]", "canonical_event_ids")
			if faultErr != nil {
				return faultErr
			}
			return failure
		}
		if _, err := DecodeObjectAuthority(members["source_objects"]); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["plan_sink"]); err != nil {
			return err
		}
		if _, err := DecodeTuple(members["target_environment"]); err != nil {
			return err
		}
		if err := requireStringBounds(members, "expected_target_native_session_id", 1, 512); err != nil {
			return err
		}
		if profile, ok := rawString(members["fidelity_profile"]); !ok || !validFidelityProfile(profile) {
			failure, err := failProtocol("projection request fidelity profile is outside the four-profile vocabulary", "fidelity_profile")
			if err != nil {
				return err
			}
			return failure
		}
		if err := checkRequiredDispositions(members["required_dispositions"]); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueStrings(members["forbid_reasons"], 1, 128, 0, 128); !ok {
			failure, err := failProtocol("projection request forbid reasons are not sorted-unique-string[1..128][0..128]", "forbid_reasons")
			if err != nil {
				return err
			}
			return failure
		}
		if _, err := DecodeResourceLimits(members["resource_limits"]); err != nil {
			return err
		}
	case OpProject:
		if err := requireDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "capture_manifest_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "canonical_session_id"); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["source_objects"]); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["target_sink"]); err != nil {
			return err
		}
	case OpReadBack:
		if _, err := DecodeReadAuthority(members["authority"]); err != nil {
			return err
		}
		if err := requireStringBounds(members, "expected_target_native_session_id", 1, 512); err != nil {
			return err
		}
		if err := requireDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "projected_object_manifest_id"); err != nil {
			return err
		}
		if _, err := DecodeObjectAuthority(members["evidence_sink"]); err != nil {
			return err
		}
	case OpValidate:
		if mode, ok := rawString(members["mode"]); !ok || !validValidateMode(mode) {
			failure, err := failProtocol("validate request mode is outside staged|live|archive", "mode")
			if err != nil {
				return err
			}
			return failure
		}
		if err := requireDigest(members, "capture_manifest_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "canonical_session_id"); err != nil {
			return err
		}
		if err := requireNullableDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		if err := requireNullableDigest(members, "projected_object_manifest_id"); err != nil {
			return err
		}
		if err := requireNullableDigest(members, "read_back_evidence_manifest_id"); err != nil {
			return err
		}
		if !isNull(members["expected_target_native_session_id"]) {
			if err := requireStringBounds(members, "expected_target_native_session_id", 1, 512); err != nil {
				return err
			}
		}
		if err := checkValidateNullability(members); err != nil {
			return err
		}
	case OpResumePlan:
		if _, err := DecodeReadAuthority(members["authority"]); err != nil {
			return err
		}
		if err := requireStringBounds(members, "expected_target_native_session_id", 1, 512); err != nil {
			return err
		}
		if err := requireDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		if err := requireNullableDigest(members, "target_checkpoint_id"); err != nil {
			return err
		}
	case OpDoctor:
		if direction, ok := rawString(members["direction"]); !ok || !validDirection(direction) {
			failure, err := failProtocol("doctor request direction is outside source_read|target_write", "direction")
			if err != nil {
				return err
			}
			return failure
		}
		if err := requireDigest(members, "tuple_registry_digest"); err != nil {
			return err
		}
		if _, err := requireBool(members, "refresh_requested"); err != nil {
			return err
		}
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("request body extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// fidelityProfiles is the four-profile vocabulary Section 7.8
// states. archive_only is a strategy, never a profile. Membership
// is a table loop so the census derives it like every other
// closed vocabulary.
var fidelityProfiles = []string{
	"strict_exact",
	"maximal_safe",
	"compact",
	"messages_only",
}

// validFidelityProfile reports whether the profile is one of the
// four Section 7.8 fidelity profiles. archive_only is a strategy,
// never a profile.
func validFidelityProfile(profile string) bool {
	for _, allowed := range fidelityProfiles {
		if profile == allowed {
			return true
		}
	}
	return false
}

// validateModes is the three-mode validate vocabulary.
var validateModes = []string{"staged", "live", "archive"}

// validValidateMode reports whether the mode is one of the three
// validate modes.
func validValidateMode(mode string) bool {
	for _, allowed := range validateModes {
		if mode == allowed {
			return true
		}
	}
	return false
}

// checkRequiredDispositions validates the closed
// required_dispositions map: an object whose every value is 1..7
// sorted unique dispositions from the seven-vocabulary with
// archive_only forbidden. The key vocabulary
// (event-or-artifact-class) has no enumerable list in v0.5.0 —
// the term occurs only in this table row — so keys validate as
// present strings and the key closure is a stated bound.
func checkRequiredDispositions(raw json.RawMessage) error {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("projection required dispositions "+fault.detail, "required_dispositions")
		if err != nil {
			return err
		}
		return failure
	}
	if len(members) == 0 {
		failure, err := failProtocol("projection required dispositions carry no class", "required_dispositions")
		if err != nil {
			return err
		}
		return failure
	}
	for class, value := range members {
		elements, ok := decodeArray(bytesTrimSpace(value))
		if !ok || len(elements) == 0 || len(elements) > 7 {
			failure, err := failProtocol("projection disposition list is not 1..7 entries", class)
			if err != nil {
				return err
			}
			return failure
		}
		previous := ""
		for _, element := range elements {
			disposition, ok := rawString(element)
			if !ok || !validDisposition(disposition) {
				failure, err := failProtocol("projection disposition is outside the seven-disposition vocabulary", class)
				if err != nil {
					return err
				}
				return failure
			}
			if previous >= disposition && previous != "" {
				failure, err := failProtocol("projection dispositions are not sorted unique", class)
				if err != nil {
					return err
				}
				return failure
			}
			previous = disposition
		}
	}
	return nil
}

// checkValidateNullability enforces the validate mode rule on
// request bodies: archive requires all four target members null,
// staged and live require them non-null. Success bodies carry no
// target members at all (the pinned Section 7.8 validate success
// list names none), so the rule never applies there; checkValidateResult
// does not call it.
func checkValidateNullability(members map[string]json.RawMessage) error {
	mode, _ := rawString(members["mode"])
	targets := []string{"projection_plan_id", "projected_object_manifest_id", "read_back_evidence_manifest_id", "expected_target_native_session_id"}
	if mode == "archive" {
		for _, name := range targets {
			if !isNull(members[name]) {
				failure, err := failProtocol("validate archive mode carries a target member", name)
				if err != nil {
					return err
				}
				return failure
			}
		}
		return nil
	}
	for _, name := range targets {
		if isNull(members[name]) {
			failure, err := failProtocol("validate staged/live mode misses a target member", name)
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// SuccessFacts are the request-side facts one success body is
// checked against: the sent context for the byte-for-byte echo,
// the validate request mode for the mode pairing, the resume-plan
// expected native identifier for the identity occurrence, and the
// doctor request direction for the direction pairing. Unused
// members stay zero for other operations.
type SuccessFacts struct {
	Context         CallContext
	ValidateMode    string
	ResumeTargetID  string
	DoctorDirection Direction
}

// CheckSuccessBody validates one inbound success body for its
// operation: the exact member set, the byte-for-byte context
// echo, and the semantic gates Section 7.8 states per operation.
// Manifest, probe, and doctor successes decode through their
// dedicated decoders. An unknown operation is operation_unknown.
func CheckSuccessBody(operation Operation, body []byte, facts SuccessFacts) error {
	if !validOperation(string(operation)) {
		failure, err := failUnknownOperation("success body names an operation outside the closed registry", string(operation))
		if err != nil {
			return err
		}
		return failure
	}
	switch operation {
	case OpManifest:
		_, err := DecodeManifest(body)
		return err
	case OpProbe:
		_, err := DecodeProbe(body)
		return err
	case OpDoctor:
		_, err := DecodeDoctorResult(body, facts.Context, facts.DoctorDirection)
		return err
	}
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol("success body "+fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	allowed := successMemberSet(operation)
	if name, unknown := unknownMember(members, allowed); unknown {
		failure, err := failProtocol("success body carries unknown member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if name, missing := missingMember(members, successBodyMembers[operation]); missing {
		failure, err := failProtocol("success body misses a required member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if _, err := CheckContextEcho(facts.Context, members["context"]); err != nil {
		return err
	}
	if err := checkSuccessScalars(operation, members, facts); err != nil {
		return err
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("success body extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// checkSuccessScalars validates the inline members of one success
// body plus the semantic gates.
func checkSuccessScalars(operation Operation, members map[string]json.RawMessage, facts SuccessFacts) error {
	switch operation {
	case OpDiscover:
		if _, err := checkDiscoverSources(members["sources"]); err != nil {
			return err
		}
		partial, err := requireBool(members, "partial")
		if err != nil {
			return err
		}
		cursorNull := isNull(members["next_cursor"])
		if !cursorNull {
			if err := requireStringBounds(members, "next_cursor", 1, 1024); err != nil {
				return err
			}
		}
		// partial=false iff next_cursor is null: a complete page
		// carries no cursor, and a cursor always means more.
		if partial == cursorNull {
			failure, err := failProtocol("discover partial flag disagrees with the cursor", "partial")
			if err != nil {
				return err
			}
			return failure
		}
	case OpInspect:
		if err := requireNestedObject(members, "source"); err != nil {
			return err
		}
		if err := requireNestedObject(members, "source_identity"); err != nil {
			return err
		}
		if err := requireStringBounds(members, "source_store_generation", 1, 512); err != nil {
			return err
		}
		if _, err := DecodeTuple(members["environment"]); err != nil {
			return err
		}
		if _, err := DecodeFindings(members["ambiguities"], 1024); err != nil {
			return err
		}
	case OpSnapshotProof:
		if err := requireNestedObject(members, "proof"); err != nil {
			return err
		}
		if _, err := requireBool(members, "provider_quiescence_requested"); err != nil {
			return err
		}
		if _, err := requireBool(members, "provider_quiescence_observed"); err != nil {
			return err
		}
	case OpCapturePlan:
		if err := requireNestedObject(members, "source_identity"); err != nil {
			return err
		}
		if err := requireStringBounds(members, "source_store_generation", 1, 512); err != nil {
			return err
		}
		candidate, err := requireDigestValue(members, "capture_plan_candidate_id")
		if err != nil {
			return err
		}
		if err := requireUint53Bounds(members, "candidate_count", 0, 65536); err != nil {
			return err
		}
		if err := checkExcludedClasses(members["excluded_classes"]); err != nil {
			return err
		}
		digest, err := requireDigestValue(members, "capture_plan_digest")
		if err != nil {
			return err
		}
		// The two candidate digests are equal after core rehash:
		// the candidate identifier addresses the plan object the
		// digest seals, so a success carrying two different
		// digests answers for an object it cannot name.
		if candidate != digest {
			failure, err := failProtocol("capture plan candidate identifier does not equal the plan digest", "capture_plan_digest")
			if err != nil {
				return err
			}
			return failure
		}
	case OpCapture:
		if err := requireDigest(members, "capture_plan_digest"); err != nil {
			return err
		}
		if err := requireStringBounds(members, "source_store_generation", 1, 512); err != nil {
			return err
		}
		if err := requireDigest(members, "capture_result_candidate_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "source_raw_object_manifest_candidate_id"); err != nil {
			return err
		}
		if err := requireUint53Bounds(members, "item_count", 0, 65536); err != nil {
			return err
		}
		if err := requireDigest(members, "pre_capture_digest"); err != nil {
			return err
		}
		if err := requireDigest(members, "post_capture_digest"); err != nil {
			return err
		}
	case OpNormalize:
		if err := requireDigest(members, "capture_manifest_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "canonical_session_candidate_id"); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueDigests(members["canonical_event_candidate_ids"], 0, 65536); !ok {
			failure, faultErr := failProtocol("normalize canonical event identifiers are not sorted unique digest[0..65536]", "canonical_event_candidate_ids")
			if faultErr != nil {
				return faultErr
			}
			return failure
		}
		if _, ok := checkSortedUniqueDigests(members["raw_reference_ids"], 0, 65536); !ok {
			failure, faultErr := failProtocol("normalize raw references are not sorted unique digest[0..65536]", "raw_reference_ids")
			if faultErr != nil {
				return faultErr
			}
			return failure
		}
	case OpProjectionPlan:
		if err := requireDigest(members, "projection_plan_candidate_id"); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueDigests(members["required_source_object_ids"], 0, 65536); !ok {
			failure, faultErr := failProtocol("projection required source objects are not sorted unique digest[0..65536]", "required_source_object_ids")
			if faultErr != nil {
				return faultErr
			}
			return failure
		}
		if err := requireNestedObject(members, "predicted_counts"); err != nil {
			return err
		}
		if _, err := DecodeFindings(members["findings"], 4096); err != nil {
			return err
		}
	case OpProject:
		if err := requireDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		if err := requireDigest(members, "projected_object_manifest_candidate_id"); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueStrings(members["created_resource_keys"], 1, 512, 0, 65536); !ok {
			failure, err := failProtocol("project created resource keys are not sorted unique string[1..512][0..65536]", "created_resource_keys")
			if err != nil {
				return err
			}
			return failure
		}
		if err := requireNestedObject(members, "actual_counts"); err != nil {
			return err
		}
	case OpReadBack:
		if err := requireStringBounds(members, "observed_target_native_session_id", 1, 512); err != nil {
			return err
		}
		if err := requireDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		if _, err := DecodeTuple(members["observed_environment"]); err != nil {
			return err
		}
		if err := requireUint53Bounds(members, "parsed_event_count", 0, maxUint53); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueStrings(members["parsed_head_ids"], 1, 512, 0, 1024); !ok {
			failure, err := failProtocol("read-back head identifiers are not sorted unique string[1..512][0..1024]", "parsed_head_ids")
			if err != nil {
				return err
			}
			return failure
		}
		if err := requireNestedObject(members, "workspace_binding"); err != nil {
			return err
		}
		if err := requireDigest(members, "structural_digest"); err != nil {
			return err
		}
		if err := requireDigest(members, "read_back_evidence_manifest_candidate_id"); err != nil {
			return err
		}
	case OpValidate:
		return checkValidateResult(members, facts)
	case OpResumePlan:
		if err := requireStringBounds(members, "target_native_session_id", 1, 512); err != nil {
			return err
		}
		if err := requireDigest(members, "projection_plan_id"); err != nil {
			return err
		}
		argv, ok := decodeArray(bytesTrimSpace(members["argv"]))
		if !ok || len(argv) == 0 || len(argv) > 128 {
			failure, err := failProtocol("resume-plan argv is not 1..128 entries", "argv")
			if err != nil {
				return err
			}
			return failure
		}
		occurrences := 0
		for _, element := range argv {
			word, ok := checkStringBounds(element, 1, 4096)
			if !ok {
				failure, err := failProtocol("resume-plan argv word is not a string[1..4096]", "argv")
				if err != nil {
					return err
				}
				return failure
			}
			if word == facts.ResumeTargetID {
				occurrences++
			}
		}
		// The explicit identity occurs exactly once: zero leaves
		// the resume unbound, two or more leave it ambiguous.
		if occurrences != 1 {
			failure, err := failProtocol("resume-plan argv does not carry the explicit identity exactly once", "argv")
			if err != nil {
				return err
			}
			return failure
		}
		if err := requireStringBounds(members, "cwd_relative", 1, 4096); err != nil {
			return err
		}
		if _, ok := checkSortedUniqueStrings(members["environment_names"], 1, 256, 0, 128); !ok {
			failure, err := failProtocol("resume-plan environment names are not sorted unique string[1..256][0..128]", "environment_names")
			if err != nil {
				return err
			}
			return failure
		}
		opens, err := requireBool(members, "opens_existing_identity")
		if err != nil {
			return err
		}
		if !opens {
			failure, err := failProtocol("resume-plan does not open the existing identity", "opens_existing_identity")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// requireDigestValue requires the member to be a digest and
// returns its canonical text. Its detail is distinct from
// requireDigest's: the two helpers are separate refusal
// obligations, so the inventory sees each.
func requireDigestValue(members map[string]json.RawMessage, name string) (string, error) {
	digest, ok := checkDigest(members[name])
	if !ok {
		failure, err := failProtocol("operation body digest value is not a digest", name)
		if err != nil {
			return "", err
		}
		return "", failure
	}
	return digest.String(), nil
}

// checkDiscoverSources validates the discover sources array:
// 0..65536 present objects. Content owned by Section 13.14
// (CloneSourceSummary) validates for presence only.
func checkDiscoverSources(raw json.RawMessage) ([]json.RawMessage, error) {
	elements, ok := decodeArray(bytesTrimSpace(raw))
	if !ok {
		failure, err := failProtocol("discover sources are not an array", "sources")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if uint64(len(elements)) > 65536 {
		failure, err := failProtocol("discover sources exceed 65536 entries", "sources")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	for _, element := range elements {
		if _, fault := decodeStrictObject(bytesTrimSpace(element)); fault != nil {
			failure, err := failProtocol("discover source entry "+fault.detail, "sources")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
	}
	return elements, nil
}

// checkExcludedClasses validates the sorted unique capture-class
// array in the [0..9] count bound.
func checkExcludedClasses(raw json.RawMessage) error {
	elements, ok := decodeArray(bytesTrimSpace(raw))
	if !ok {
		failure, err := failProtocol("capture excluded classes are not an array", "excluded_classes")
		if err != nil {
			return err
		}
		return failure
	}
	if uint64(len(elements)) > 9 {
		failure, err := failProtocol("capture excluded classes exceed 9 entries", "excluded_classes")
		if err != nil {
			return err
		}
		return failure
	}
	previous := ""
	for _, element := range elements {
		class, ok := rawString(element)
		if !ok || !validCaptureClass(class) {
			failure, err := failProtocol("capture excluded class is outside the nine-class vocabulary", "excluded_classes")
			if err != nil {
				return err
			}
			return failure
		}
		if previous >= class && previous != "" {
			failure, err := failProtocol("capture excluded classes are not sorted unique", "excluded_classes")
			if err != nil {
				return err
			}
			return failure
		}
		previous = class
	}
	return nil
}

// checkValidateResult validates one validate success body: the
// mode echoes the request mode, and valid=true requires every
// applicable check true with no error finding. A null applicable
// check is not applicable and is skipped; a false one fails the
// conjunction. Target nullability is a request-body rule only:
// the success body carries no target members, so there is nothing
// to check here.
func checkValidateResult(members map[string]json.RawMessage, facts SuccessFacts) error {
	mode, ok := rawString(members["mode"])
	if !ok || !validValidateMode(mode) {
		failure, err := failProtocol("validate result mode is outside staged|live|archive", "mode")
		if err != nil {
			return err
		}
		return failure
	}
	if mode != facts.ValidateMode {
		failure, err := failProtocol("validate result mode does not echo the request mode", "mode")
		if err != nil {
			return err
		}
		return failure
	}
	structural, err := requireBool(members, "structural_valid")
	if err != nil {
		return err
	}
	semantic, err := requireBool(members, "semantic_marker_valid")
	if err != nil {
		return err
	}
	for _, name := range []string{"identity_valid", "workspace_binding_valid", "resume_surface_valid"} {
		if err := requireNullableBool(members, name); err != nil {
			return err
		}
	}
	findings, err := DecodeFindings(members["findings"], 4096)
	if err != nil {
		return err
	}
	if err := requireDigest(members, "evidence_digest"); err != nil {
		return err
	}
	valid, err := requireBool(members, "valid")
	if err != nil {
		return err
	}
	if !valid {
		return nil
	}
	if !structural || !semantic {
		failure, err := failProtocol("validate reports valid with a failed structural or semantic check", "valid")
		if err != nil {
			return err
		}
		return failure
	}
	for _, name := range []string{"identity_valid", "workspace_binding_valid", "resume_surface_valid"} {
		if !isNull(members[name]) {
			applicable, _ := rawBool(members[name])
			if !applicable {
				failure, err := failProtocol("validate reports valid with a failed applicable check", name)
				if err != nil {
					return err
				}
				return failure
			}
		}
	}
	for _, finding := range findings {
		if finding.Severity == "error" {
			failure, err := failProtocol("validate reports valid with an error finding", "findings")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}
