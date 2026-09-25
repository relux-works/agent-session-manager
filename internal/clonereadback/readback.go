package clonereadback

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Clone Read-Back Evidence
// Manifest 1.0.0: the closed 15-member manifest Section 13.14.2
// states, binding operation, staged|live mode, plan and projected
// manifest, equal expected/observed native Session IDs, observed
// tuple, parsed count/heads, Workspace Binding, structural digest,
// and sorted evidence blobs. The mode is derived from the trusted
// read authority the caller threads from its authority chain and
// the manifest claim must agree with it: a resealed opposite mode
// refuses as an authority mismatch even though its self-digest
// recomputes, because the digest alone does not carry the stage.
// Staged and live manifests are distinct documents, never two
// readings of one.

const (
	readBackSchema   = "urn:ax:schema:clone-read-back-evidence-manifest"
	readBackVersion  = "1.0.0"
	readBackSelf     = "read_back_evidence_manifest_id"
	maxEvidenceRows  = 65536
	maxParsedHeadIDs = 1024
)

var readBackMembers = map[string]bool{
	"schema": true, "schema_version": true,
	"read_back_evidence_manifest_id": true, "operation_id": true,
	"mode": true, "projection_plan_id": true,
	"projected_object_manifest_id":      true,
	"expected_target_native_session_id": true,
	"observed_target_native_session_id": true,
	"observed_environment":              true,
	"parsed_event_count":                true,
	"parsed_head_ids":                   true,
	"workspace_binding":                 true,
	"structural_digest":                 true,
	"evidence_objects":                  true, "extensions": true,
}

var readBackRequired = []string{
	"schema", "schema_version",
	"read_back_evidence_manifest_id", "operation_id",
	"mode", "projection_plan_id",
	"projected_object_manifest_id",
	"expected_target_native_session_id",
	"observed_target_native_session_id",
	"observed_environment",
	"parsed_event_count",
	"parsed_head_ids",
	"workspace_binding",
	"structural_digest",
	"evidence_objects", "extensions",
}

var evidenceMembers = map[string]bool{
	"evidence_kind": true, "media_type": true,
	"byte_count": true, "blob_id": true,
	"blob_descriptor_id": true,
}

var evidenceRequired = []string{
	"evidence_kind", "media_type",
	"byte_count", "blob_id",
	"blob_descriptor_id",
}

// EvidenceObject is one validated evidence row: the closed kind, a
// media type, a byte count, and the blob pair.
type EvidenceObject struct {
	EvidenceKind     string
	MediaType        string
	ByteCount        uint64
	BlobID           scalar.Digest
	BlobDescriptorID scalar.Digest
}

// EvidenceObjectInput is the caller-supplied evidence row candidate
// for Build.
type EvidenceObjectInput struct {
	EvidenceKind     string
	MediaType        string
	ByteCount        uint64
	BlobID           string
	BlobDescriptorID string
}

// ReadBackManifest is one validated Clone Read-Back Evidence
// Manifest 1.0.0.
type ReadBackManifest struct {
	ManifestID                  scalar.Digest
	OperationID                 scalar.UUIDv7
	Mode                        string
	ProjectionPlanID            scalar.Digest
	ProjectedObjectManifestID   scalar.Digest
	ExpectedTargetNativeSession string
	ObservedTargetNativeSession string
	ObservedEnvironment         sessadapter.Tuple
	ParsedEventCount            uint64
	ParsedHeadIDs               []string
	WorkspaceBinding            clonebundle.WorkspaceBinding
	StructuralDigest            scalar.Digest
	EvidenceObjects             []EvidenceObject
}

// ReadBackManifestInput is the caller-supplied read-back manifest
// candidate for Build. Nested closed shapes (observed tuple,
// workspace binding) arrive as raw JSON and decode through their
// owners; extensions arrive as a Go map and seal through the
// closed-extensions owner.
type ReadBackManifestInput struct {
	OperationID                 string
	Mode                        string
	ProjectionPlanID            string
	ProjectedObjectManifestID   string
	ExpectedTargetNativeSession string
	ObservedTargetNativeSession string
	ObservedEnvironment         []byte
	ParsedEventCount            uint64
	ParsedHeadIDs               []string
	WorkspaceBinding            []byte
	StructuralDigest            string
	EvidenceObjects             []EvidenceObjectInput
	Extensions                  map[string]any
}

// buildEvidenceObject validates one caller-supplied evidence row
// and renders its closed object.
func buildEvidenceObject(input EvidenceObjectInput, index int) (EvidenceObject, map[string]any, error) {
	owner := fmt.Sprintf("evidence object[%d]", index)
	if !validText(input.EvidenceKind) {
		return EvidenceObject{}, nil, invalid("%s evidence_kind is not valid UTF-8", owner)
	}
	if !ValidEvidenceKind(input.EvidenceKind) {
		return EvidenceObject{}, nil, invalid("%s evidence_kind is outside native_sample|parser_trace|marker_observation", owner)
	}
	if !validText(input.MediaType) {
		return EvidenceObject{}, nil, invalid("%s media_type is not valid UTF-8", owner)
	}
	if length := stringLength(input.MediaType); length < 1 || length > 128 {
		return EvidenceObject{}, nil, invalid("%s media_type is not a string[1..128]", owner)
	}
	if input.ByteCount > maxUint53 {
		return EvidenceObject{}, nil, invalid("%s byte_count exceeds uint53", owner)
	}
	blob, err := scalar.ParseDigest(input.BlobID)
	if err != nil {
		return EvidenceObject{}, nil, invalid("%s blob_id is not a digest: %v", owner, err)
	}
	descriptor, err := scalar.ParseDigest(input.BlobDescriptorID)
	if err != nil {
		return EvidenceObject{}, nil, invalid("%s blob_descriptor_id is not a digest: %v", owner, err)
	}
	row := EvidenceObject{
		EvidenceKind:     input.EvidenceKind,
		MediaType:        input.MediaType,
		ByteCount:        input.ByteCount,
		BlobID:           blob,
		BlobDescriptorID: descriptor,
	}
	object := map[string]any{
		"evidence_kind":      input.EvidenceKind,
		"media_type":         input.MediaType,
		"byte_count":         input.ByteCount,
		"blob_id":            blob.String(),
		"blob_descriptor_id": descriptor.String(),
	}
	return row, object, nil
}

// decodeEvidenceObject validates one closed EvidenceObject.
func decodeEvidenceObject(raw json.RawMessage, index int) (EvidenceObject, error) {
	owner := "evidence object"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return EvidenceObject{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, evidenceMembers); unknown {
		return EvidenceObject{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, evidenceRequired); missing {
		return EvidenceObject{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	kind, ok := rawString(members["evidence_kind"])
	if !ok || !ValidEvidenceKind(kind) {
		return EvidenceObject{}, invalid("%s[%d] evidence_kind is outside native_sample|parser_trace|marker_observation", owner, index)
	}
	media, ok := checkStringBounds(members["media_type"], 1, 128)
	if !ok {
		return EvidenceObject{}, invalid("%s[%d] media_type is not a string[1..128]", owner, index)
	}
	count, ok := checkUint53Bounds(members["byte_count"], 0, maxUint53)
	if !ok {
		return EvidenceObject{}, invalid("%s[%d] byte_count is not a uint53", owner, index)
	}
	blob, ok := checkDigest(members["blob_id"])
	if !ok {
		return EvidenceObject{}, invalid("%s[%d] blob_id is not a digest", owner, index)
	}
	descriptor, ok := checkDigest(members["blob_descriptor_id"])
	if !ok {
		return EvidenceObject{}, invalid("%s[%d] blob_descriptor_id is not a digest", owner, index)
	}
	return EvidenceObject{
		EvidenceKind:     kind,
		MediaType:        media,
		ByteCount:        count,
		BlobID:           blob,
		BlobDescriptorID: descriptor,
	}, nil
}

// checkEvidenceOrder enforces "sorted unique by evidence kind/blob
// ID": the (kind, blob ID) pairs are strictly increasing in
// bytewise string order. Both entries share this gate over the
// sealed row set.
func checkEvidenceOrder(rows []EvidenceObject) error {
	for index := 1; index < len(rows); index++ {
		previous := rows[index-1]
		current := rows[index]
		if current.EvidenceKind < previous.EvidenceKind ||
			(current.EvidenceKind == previous.EvidenceKind && current.BlobID.String() <= previous.BlobID.String()) {
			return invalid("read-back evidence manifest evidence_objects are not sorted unique by evidence kind/blob ID")
		}
	}
	return nil
}

// checkEvidenceCount enforces the evidence cardinality bound
// EvidenceObject[0..65536]. The gate is factored so the exact upper
// edge pins at the gate while row reachability pins at both
// entries.
func checkEvidenceCount(count int) error {
	if count < 0 || count > maxEvidenceRows {
		return invalid("read-back evidence manifest carries %d evidence_objects, want [0..65536]", count)
	}
	return nil
}

// checkNativeIdentityMatch enforces the pinned equality of the two
// native Session IDs: the observed target session is the expected
// one, never a neighbor. Both entries share this gate.
func checkNativeIdentityMatch(expected, observed string) error {
	if expected != observed {
		return invalid("read-back evidence manifest expected and observed native Session IDs differ")
	}
	return nil
}

// BuildReadBackEvidenceManifest constructs one closed Clone
// Read-Back Evidence Manifest 1.0.0 as canonical bytes. Identical
// inputs produce byte-identical outputs. The authority carries the
// trusted read purpose from the caller chain; the claimed mode
// must agree with the mode that purpose derives, or the build
// refuses as a relabel. Every closed-shape, mode, equality,
// order, and vocabulary rule is a refusal, never a repair. On
// success the entry also mints the sealed read the report
// entries aggregate: the sealed value carries this seal's
// omit-self digest and authority-derived mode.
func BuildReadBackEvidenceManifest(input ReadBackManifestInput, authority sessadapter.ReadAuthority) (ValidatedReadBack, []byte, error) {
	manifest, object, err := buildReadBackManifest(input, authority)
	if err != nil {
		return ValidatedReadBack{}, nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return ValidatedReadBack{}, nil, err
	}
	manifestID := scalar.SHA256Digest(omitted)
	object[readBackSelf] = manifestID.String()
	sealed, err := canonicalizeObject(object)
	if err != nil {
		return ValidatedReadBack{}, nil, err
	}
	manifest.ManifestID = manifestID
	return sealReadBack(manifest), sealed, nil
}

func buildReadBackManifest(input ReadBackManifestInput, authority sessadapter.ReadAuthority) (ReadBackManifest, map[string]any, error) {
	owner := "read-back evidence manifest"
	operation, err := scalar.ParseUUIDv7(input.OperationID)
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s operation_id is not a UUIDv7: %v", owner, err)
	}
	if !validText(input.Mode) {
		return ReadBackManifest{}, nil, invalid("%s mode is not valid UTF-8", owner)
	}
	if !ValidReadBackMode(input.Mode) {
		return ReadBackManifest{}, nil, invalid("%s mode is outside staged|live", owner)
	}
	expectedMode, err := modeForAuthorityPurpose(authority.Purpose)
	if err != nil {
		return ReadBackManifest{}, nil, err
	}
	if err := checkModeAuthority(owner, input.Mode, expectedMode, authority.Purpose); err != nil {
		return ReadBackManifest{}, nil, err
	}
	planID, err := scalar.ParseDigest(input.ProjectionPlanID)
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s projection_plan_id is not a digest: %v", owner, err)
	}
	manifestID, err := scalar.ParseDigest(input.ProjectedObjectManifestID)
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s projected_object_manifest_id is not a digest: %v", owner, err)
	}
	if !validText(input.ExpectedTargetNativeSession) {
		return ReadBackManifest{}, nil, invalid("%s expected_target_native_session_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ExpectedTargetNativeSession); length < 1 || length > 512 {
		return ReadBackManifest{}, nil, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	if !validText(input.ObservedTargetNativeSession) {
		return ReadBackManifest{}, nil, invalid("%s observed_target_native_session_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ObservedTargetNativeSession); length < 1 || length > 512 {
		return ReadBackManifest{}, nil, invalid("%s observed_target_native_session_id is not a string[1..512]", owner)
	}
	if err := checkNativeIdentityMatch(input.ExpectedTargetNativeSession, input.ObservedTargetNativeSession); err != nil {
		return ReadBackManifest{}, nil, err
	}
	observedTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.ObservedEnvironment))
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s observed_environment is not an Environment Tuple: %v", owner, err)
	}
	if input.ParsedEventCount > maxUint53 {
		return ReadBackManifest{}, nil, invalid("%s parsed_event_count exceeds uint53", owner)
	}
	heads, err := checkSortedUniqueBoundedStrings(input.ParsedHeadIDs, 1, 512, 0, maxParsedHeadIDs, owner, "parsed_head_ids")
	if err != nil {
		return ReadBackManifest{}, nil, err
	}
	binding, err := clonebundle.DecodeWorkspaceBinding(json.RawMessage(input.WorkspaceBinding))
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s workspace_binding is not a Workspace Binding: %v", owner, err)
	}
	structural, err := scalar.ParseDigest(input.StructuralDigest)
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s structural_digest is not a digest: %v", owner, err)
	}
	if err := checkEvidenceCount(len(input.EvidenceObjects)); err != nil {
		return ReadBackManifest{}, nil, err
	}
	rows := make([]EvidenceObject, 0, len(input.EvidenceObjects))
	rowObjects := make([]any, 0, len(input.EvidenceObjects))
	for index, candidate := range input.EvidenceObjects {
		row, object, err := buildEvidenceObject(candidate, index)
		if err != nil {
			return ReadBackManifest{}, nil, err
		}
		rows = append(rows, row)
		rowObjects = append(rowObjects, object)
	}
	if err := checkEvidenceOrder(rows); err != nil {
		return ReadBackManifest{}, nil, err
	}
	extensions, err := clonebundle.EncodeExtensions(input.Extensions)
	if err != nil {
		return ReadBackManifest{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	var extensionsValue any
	if err := json.Unmarshal(extensions, &extensionsValue); err != nil {
		return ReadBackManifest{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	manifest := ReadBackManifest{
		OperationID:                 operation,
		Mode:                        input.Mode,
		ProjectionPlanID:            planID,
		ProjectedObjectManifestID:   manifestID,
		ExpectedTargetNativeSession: input.ExpectedTargetNativeSession,
		ObservedTargetNativeSession: input.ObservedTargetNativeSession,
		ObservedEnvironment:         observedTuple,
		ParsedEventCount:            input.ParsedEventCount,
		ParsedHeadIDs:               heads,
		WorkspaceBinding:            binding,
		StructuralDigest:            structural,
		EvidenceObjects:             rows,
	}
	object := map[string]any{
		"schema":                            readBackSchema,
		"schema_version":                    readBackVersion,
		"operation_id":                      operation.String(),
		"mode":                              input.Mode,
		"projection_plan_id":                planID.String(),
		"projected_object_manifest_id":      manifestID.String(),
		"expected_target_native_session_id": input.ExpectedTargetNativeSession,
		"observed_target_native_session_id": input.ObservedTargetNativeSession,
		"parsed_event_count":                input.ParsedEventCount,
		"parsed_head_ids":                   stringValues(heads),
		"structural_digest":                 structural.String(),
		"evidence_objects":                  rowObjects,
		"extensions":                        extensionsValue,
	}
	var tupleValue any
	if err := json.Unmarshal(input.ObservedEnvironment, &tupleValue); err != nil {
		return ReadBackManifest{}, nil, invalid("%s observed_environment invalid: %v", owner, err)
	}
	object["observed_environment"] = tupleValue
	var bindingValue any
	if err := json.Unmarshal(input.WorkspaceBinding, &bindingValue); err != nil {
		return ReadBackManifest{}, nil, invalid("%s workspace_binding invalid: %v", owner, err)
	}
	object["workspace_binding"] = bindingValue
	return manifest, object, nil
}

// DecodeReadBackEvidenceManifest validates one closed Clone
// Read-Back Evidence Manifest 1.0.0: exact members, literal
// schema/version, the closed mode bound to the trusted read
// authority, equal native IDs, owner-decoded tuple and binding,
// sorted heads and rows, and the omit-self identity. A relabeled
// mode refuses as an authority mismatch even when the self-digest
// recomputes. On success the entry mints the sealed read the
// report entries aggregate.
func DecodeReadBackEvidenceManifest(data []byte, authority sessadapter.ReadAuthority) (ValidatedReadBack, error) {
	owner := "read-back evidence manifest"
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return ValidatedReadBack{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, readBackMembers); unknown {
		return ValidatedReadBack{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, readBackRequired); missing {
		return ValidatedReadBack{}, invalid("%s misses a required member %q", owner, name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != readBackSchema {
		return ValidatedReadBack{}, invalid("%s schema is not urn:ax:schema:clone-read-back-evidence-manifest", owner)
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != readBackVersion {
		return ValidatedReadBack{}, invalid("%s schema_version is not 1.0.0", owner)
	}
	claimed, ok := checkDigest(members[readBackSelf])
	if !ok {
		return ValidatedReadBack{}, invalid("%s read_back_evidence_manifest_id is not a digest", owner)
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return ValidatedReadBack{}, invalid("%s operation_id is not a UUIDv7", owner)
	}
	mode, ok := rawString(members["mode"])
	if !ok || !ValidReadBackMode(mode) {
		return ValidatedReadBack{}, invalid("%s mode is outside staged|live", owner)
	}
	expectedMode, err := modeForAuthorityPurpose(authority.Purpose)
	if err != nil {
		return ValidatedReadBack{}, err
	}
	if err := checkModeAuthority(owner, mode, expectedMode, authority.Purpose); err != nil {
		return ValidatedReadBack{}, err
	}
	planID, ok := checkDigest(members["projection_plan_id"])
	if !ok {
		return ValidatedReadBack{}, invalid("%s projection_plan_id is not a digest", owner)
	}
	projectedID, ok := checkDigest(members["projected_object_manifest_id"])
	if !ok {
		return ValidatedReadBack{}, invalid("%s projected_object_manifest_id is not a digest", owner)
	}
	expected, ok := checkStringBounds(members["expected_target_native_session_id"], 1, 512)
	if !ok {
		return ValidatedReadBack{}, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	observed, ok := checkStringBounds(members["observed_target_native_session_id"], 1, 512)
	if !ok {
		return ValidatedReadBack{}, invalid("%s observed_target_native_session_id is not a string[1..512]", owner)
	}
	if err := checkNativeIdentityMatch(expected, observed); err != nil {
		return ValidatedReadBack{}, err
	}
	observedTuple, err := sessadapter.DecodeTuple(members["observed_environment"])
	if err != nil {
		return ValidatedReadBack{}, invalid("%s observed_environment is not an Environment Tuple: %v", owner, err)
	}
	parsedCount, ok := checkUint53Bounds(members["parsed_event_count"], 0, maxUint53)
	if !ok {
		return ValidatedReadBack{}, invalid("%s parsed_event_count is not a uint53", owner)
	}
	heads, ok := checkSortedUniqueStrings(members["parsed_head_ids"], 1, 512, 0, maxParsedHeadIDs)
	if !ok {
		return ValidatedReadBack{}, invalid("%s parsed_head_ids are not sorted unique string[1..512][0..1024]", owner)
	}
	binding, err := clonebundle.DecodeWorkspaceBinding(members["workspace_binding"])
	if err != nil {
		return ValidatedReadBack{}, invalid("%s workspace_binding is not a Workspace Binding: %v", owner, err)
	}
	structural, ok := checkDigest(members["structural_digest"])
	if !ok {
		return ValidatedReadBack{}, invalid("%s structural_digest is not a digest", owner)
	}
	rawRows, ok := decodeArray(members["evidence_objects"])
	if !ok {
		return ValidatedReadBack{}, invalid("%s evidence_objects are not an array", owner)
	}
	if err := checkEvidenceCount(len(rawRows)); err != nil {
		return ValidatedReadBack{}, err
	}
	rows := make([]EvidenceObject, 0, len(rawRows))
	for index, raw := range rawRows {
		row, err := decodeEvidenceObject(raw, index)
		if err != nil {
			return ValidatedReadBack{}, err
		}
		rows = append(rows, row)
	}
	if err := checkEvidenceOrder(rows); err != nil {
		return ValidatedReadBack{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ValidatedReadBack{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	if err := verifySelfDigest(members, readBackSelf, claimed); err != nil {
		return ValidatedReadBack{}, err
	}
	return sealReadBack(ReadBackManifest{
		ManifestID:                  claimed,
		OperationID:                 operation,
		Mode:                        mode,
		ProjectionPlanID:            planID,
		ProjectedObjectManifestID:   projectedID,
		ExpectedTargetNativeSession: expected,
		ObservedTargetNativeSession: observed,
		ObservedEnvironment:         observedTuple,
		ParsedEventCount:            parsedCount,
		ParsedHeadIDs:               heads,
		WorkspaceBinding:            binding,
		StructuralDigest:            structural,
		EvidenceObjects:             rows,
	}), nil
}
