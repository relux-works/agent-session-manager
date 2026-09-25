package cloneplan

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Clone Projected Object
// Manifest 1.0.0: the closed 10-member manifest Section 13.14.2
// states with its sorted entries partitioned by operation sequence.
// Entries are branch-exact by kind: a blob carries mode, byte
// count, blob ID, and blob descriptor ID; a directory carries only
// mode. Entries partitioning Plan resources needs the plan as an
// input and is a stated bound, as is total_bytes reconciliation:
// the pinned text states no derivation rule for the total, so any
// uint53 total admits.

const (
	projectedManifestSchema  = "urn:ax:schema:clone-projected-object-manifest"
	projectedManifestVersion = "1.0.0"
	projectedManifestSelf    = "projected_object_manifest_id"
)

var projectedManifestMembers = map[string]bool{
	"schema": true, "schema_version": true,
	"projected_object_manifest_id": true, "operation_id": true,
	"projection_plan_id": true, "target_environment": true,
	"expected_target_native_session_id": true,
	"entries":                           true, "total_bytes": true, "extensions": true,
}

var projectedManifestRequired = []string{
	"schema", "schema_version",
	"projected_object_manifest_id", "operation_id",
	"projection_plan_id", "target_environment",
	"expected_target_native_session_id",
	"entries", "total_bytes", "extensions",
}

var blobEntryMembers = map[string]bool{
	"operation_sequence": true,
	"resource_key":       true,
	"kind":               true,
	"mode":               true,
	"byte_count":         true,
	"blob_id":            true,
	"blob_descriptor_id": true,
}

var blobEntryRequired = []string{
	"operation_sequence",
	"resource_key",
	"kind",
	"mode",
	"byte_count",
	"blob_id",
	"blob_descriptor_id",
}

var directoryEntryMembers = map[string]bool{
	"operation_sequence": true,
	"resource_key":       true,
	"kind":               true,
	"mode":               true,
}

var directoryEntryRequired = []string{
	"operation_sequence",
	"resource_key",
	"kind",
	"mode",
}

// ProjectedEntry is one validated projected object entry. Blob
// entries carry byte count and blob IDs; directory entries carry
// only mode.
type ProjectedEntry struct {
	OperationSequence uint64
	ResourceKey       string
	Kind              string
	Mode              *uint64
	ByteCount         *uint64
	BlobID            *scalar.Digest
	BlobDescriptorID  *scalar.Digest
}

// ProjectedEntryInput is the caller-supplied projected entry
// candidate for Build. Nil Mode seals as null on both branches; the
// blob-only members must be non-nil exactly for blob entries.
type ProjectedEntryInput struct {
	OperationSequence uint64
	ResourceKey       string
	Kind              string
	Mode              *uint64
	ByteCount         *uint64
	BlobID            *string
	BlobDescriptorID  *string
}

// ProjectedManifest is one validated projected object manifest.
type ProjectedManifest struct {
	ManifestID                  scalar.Digest
	OperationID                 scalar.UUIDv7
	ProjectionPlanID            scalar.Digest
	TargetEnvironment           sessadapter.Tuple
	ExpectedTargetNativeSession string
	Entries                     []ProjectedEntry
	TotalBytes                  uint64
}

// ProjectedManifestInput is the caller-supplied projected manifest
// candidate for Build.
type ProjectedManifestInput struct {
	OperationID                 string
	ProjectionPlanID            string
	TargetEnvironment           []byte
	ExpectedTargetNativeSession string
	Entries                     []ProjectedEntryInput
	TotalBytes                  uint64
	Extensions                  map[string]any
}

// checkEntryBranch enforces branch-exact entry membership: blob
// entries carry byte count and both blob IDs, directory entries
// carry none of the three. Both entries share this gate.
func checkEntryBranch(owner, kind string, hasCount, hasBlob, hasDescriptor bool) error {
	if kind == "blob" {
		if !hasCount {
			return invalid("%s byte_count is missing for blob", owner)
		}
		if !hasBlob {
			return invalid("%s blob_id is missing for blob", owner)
		}
		if !hasDescriptor {
			return invalid("%s blob_descriptor_id is missing for blob", owner)
		}
		return nil
	}
	if hasCount {
		return invalid("%s byte_count is present for directory", owner)
	}
	if hasBlob {
		return invalid("%s blob_id is present for directory", owner)
	}
	if hasDescriptor {
		return invalid("%s blob_descriptor_id is present for directory", owner)
	}
	return nil
}

// buildProjectedEntry validates one caller-supplied entry and
// renders its closed object. The rendered member set is branch-exact:
// seven members for blob, four for directory.
func buildProjectedEntry(input ProjectedEntryInput, index int) (ProjectedEntry, map[string]any, error) {
	owner := fmt.Sprintf("projected object entry[%d]", index)
	if input.OperationSequence == 0 || input.OperationSequence > maxUint53 {
		return ProjectedEntry{}, nil, invalid("%s operation_sequence is not a uint53>0", owner)
	}
	if !validText(input.ResourceKey) {
		return ProjectedEntry{}, nil, invalid("%s resource_key is not valid UTF-8", owner)
	}
	if length := stringLength(input.ResourceKey); length < 1 || length > 512 {
		return ProjectedEntry{}, nil, invalid("%s resource_key is not a string[1..512]", owner)
	}
	if !validText(input.Kind) {
		return ProjectedEntry{}, nil, invalid("%s kind is not valid UTF-8", owner)
	}
	if !ValidResourceKind(input.Kind) {
		return ProjectedEntry{}, nil, invalid("%s kind is outside blob|directory", owner)
	}
	mode, modeValue, err := checkBuildMode(input.Mode, owner)
	if err != nil {
		return ProjectedEntry{}, nil, err
	}
	if err := checkEntryBranch(owner, input.Kind, input.ByteCount != nil, input.BlobID != nil, input.BlobDescriptorID != nil); err != nil {
		return ProjectedEntry{}, nil, err
	}
	object := map[string]any{
		"operation_sequence": input.OperationSequence,
		"resource_key":       input.ResourceKey,
		"kind":               input.Kind,
		"mode":               modeValue,
	}
	entry := ProjectedEntry{
		OperationSequence: input.OperationSequence,
		ResourceKey:       input.ResourceKey,
		Kind:              input.Kind,
		Mode:              mode,
	}
	if input.Kind == "blob" {
		if *input.ByteCount > maxUint53 {
			return ProjectedEntry{}, nil, invalid("%s byte_count exceeds uint53", owner)
		}
		count := *input.ByteCount
		blob, err := scalar.ParseDigest(*input.BlobID)
		if err != nil {
			return ProjectedEntry{}, nil, invalid("%s blob_id is not a digest: %v", owner, err)
		}
		descriptor, err := scalar.ParseDigest(*input.BlobDescriptorID)
		if err != nil {
			return ProjectedEntry{}, nil, invalid("%s blob_descriptor_id is not a digest: %v", owner, err)
		}
		entry.ByteCount = &count
		entry.BlobID = &blob
		entry.BlobDescriptorID = &descriptor
		object["byte_count"] = count
		object["blob_id"] = blob.String()
		object["blob_descriptor_id"] = descriptor.String()
	}
	return entry, object, nil
}

// decodeProjectedEntry validates one closed ProjectedObjectEntry,
// dispatching on kind to the branch-exact member set.
func decodeProjectedEntry(raw json.RawMessage, index int) (ProjectedEntry, error) {
	owner := "projected object entry"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return ProjectedEntry{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	kindMember, present := members["kind"]
	if !present {
		return ProjectedEntry{}, invalid("%s[%d] misses a required member %q", owner, index, "kind")
	}
	kind, ok := rawString(kindMember)
	if !ok || !ValidResourceKind(kind) {
		return ProjectedEntry{}, invalid("%s[%d] kind is outside blob|directory", owner, index)
	}
	allowed, required := blobEntryMembers, blobEntryRequired
	if kind == "directory" {
		allowed, required = directoryEntryMembers, directoryEntryRequired
	}
	if name, unknown := unknownMember(members, allowed); unknown {
		return ProjectedEntry{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, required); missing {
		return ProjectedEntry{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	sequence, ok := checkUint53Bounds(members["operation_sequence"], 1, maxUint53)
	if !ok {
		return ProjectedEntry{}, invalid("%s[%d] operation_sequence is not a uint53>0", owner, index)
	}
	key, ok := checkStringBounds(members["resource_key"], 1, 512)
	if !ok {
		return ProjectedEntry{}, invalid("%s[%d] resource_key is not a string[1..512]", owner, index)
	}
	var mode *uint64
	if !isNull(members["mode"]) {
		value, ok := checkUint53Bounds(members["mode"], 0, maxMode)
		if !ok {
			return ProjectedEntry{}, invalid("%s[%d] mode is not a uint32[0..4095] or null", owner, index)
		}
		mode = &value
	}
	entry := ProjectedEntry{
		OperationSequence: sequence,
		ResourceKey:       key,
		Kind:              kind,
		Mode:              mode,
	}
	if kind == "blob" {
		count, ok := checkUint53Bounds(members["byte_count"], 0, maxUint53)
		if !ok {
			return ProjectedEntry{}, invalid("%s[%d] byte_count is not a uint53", owner, index)
		}
		blob, ok := checkDigest(members["blob_id"])
		if !ok {
			return ProjectedEntry{}, invalid("%s[%d] blob_id is not a digest", owner, index)
		}
		descriptor, ok := checkDigest(members["blob_descriptor_id"])
		if !ok {
			return ProjectedEntry{}, invalid("%s[%d] blob_descriptor_id is not a digest", owner, index)
		}
		entry.ByteCount = &count
		entry.BlobID = &blob
		entry.BlobDescriptorID = &descriptor
	}
	return entry, nil
}

// checkProjectedEntryOrder enforces "sorted unique by operation
// sequence/resource key": the (sequence, key) pairs are strictly
// increasing. Both entries share this gate over the sealed entry
// set.
func checkProjectedEntryOrder(entries []ProjectedEntry) error {
	for index := 1; index < len(entries); index++ {
		previous := entries[index-1]
		current := entries[index]
		if current.OperationSequence < previous.OperationSequence ||
			(current.OperationSequence == previous.OperationSequence && current.ResourceKey <= previous.ResourceKey) {
			return invalid("projected object manifest entries are not sorted unique by operation sequence/resource key")
		}
	}
	return nil
}

// checkProjectedEntryCount enforces the entries cardinality bound
// ProjectedObjectEntry[0..65536]. The gate is factored so the exact
// upper edge pins at the gate while entry reachability pins at both
// entries.
func checkProjectedEntryCount(count int) error {
	if count < 0 || count > 65536 {
		return invalid("projected object manifest carries %d entries, want [0..65536]", count)
	}
	return nil
}

// BuildProjectedObjectManifest constructs one closed Clone
// Projected Object Manifest 1.0.0 as canonical bytes. Identical
// inputs produce byte-identical outputs. Every closed-shape,
// branch, order, and vocabulary rule is a refusal, never a repair.
func BuildProjectedObjectManifest(input ProjectedManifestInput) ([]byte, error) {
	_, object, err := buildProjectedManifest(input)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	manifestID := scalar.SHA256Digest(omitted)
	object[projectedManifestSelf] = manifestID.String()
	return canonicalizeObject(object)
}

func buildProjectedManifest(input ProjectedManifestInput) (ProjectedManifest, map[string]any, error) {
	owner := "projected object manifest"
	operation, err := scalar.ParseUUIDv7(input.OperationID)
	if err != nil {
		return ProjectedManifest{}, nil, invalid("%s operation_id is not a UUIDv7: %v", owner, err)
	}
	planID, err := scalar.ParseDigest(input.ProjectionPlanID)
	if err != nil {
		return ProjectedManifest{}, nil, invalid("%s projection_plan_id is not a digest: %v", owner, err)
	}
	targetTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.TargetEnvironment))
	if err != nil {
		return ProjectedManifest{}, nil, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
	}
	if !validText(input.ExpectedTargetNativeSession) {
		return ProjectedManifest{}, nil, invalid("%s expected_target_native_session_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ExpectedTargetNativeSession); length < 1 || length > 512 {
		return ProjectedManifest{}, nil, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	if err := checkProjectedEntryCount(len(input.Entries)); err != nil {
		return ProjectedManifest{}, nil, err
	}
	entries := make([]ProjectedEntry, 0, len(input.Entries))
	objects := make([]any, 0, len(input.Entries))
	for index, candidate := range input.Entries {
		entry, object, err := buildProjectedEntry(candidate, index)
		if err != nil {
			return ProjectedManifest{}, nil, err
		}
		entries = append(entries, entry)
		objects = append(objects, object)
	}
	if err := checkProjectedEntryOrder(entries); err != nil {
		return ProjectedManifest{}, nil, err
	}
	if input.TotalBytes > maxUint53 {
		return ProjectedManifest{}, nil, invalid("%s total_bytes exceeds uint53", owner)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return ProjectedManifest{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	manifest := ProjectedManifest{
		OperationID:                 operation,
		ProjectionPlanID:            planID,
		TargetEnvironment:           targetTuple,
		ExpectedTargetNativeSession: input.ExpectedTargetNativeSession,
		Entries:                     entries,
		TotalBytes:                  input.TotalBytes,
	}
	object := map[string]any{
		"schema":                            projectedManifestSchema,
		"schema_version":                    projectedManifestVersion,
		"operation_id":                      operation.String(),
		"projection_plan_id":                planID.String(),
		"target_environment":                tupleObject(targetTuple),
		"expected_target_native_session_id": input.ExpectedTargetNativeSession,
		"entries":                           objects,
		"total_bytes":                       input.TotalBytes,
		"extensions":                        extensions,
	}
	return manifest, object, nil
}

// DecodeProjectedObjectManifest validates one closed Clone
// Projected Object Manifest 1.0.0: exact members, schema/version,
// scalar bounds, branch-exact entries, entry order, and self-digest
// agreement.
func DecodeProjectedObjectManifest(data []byte) (ProjectedManifest, error) {
	owner := "projected object manifest"
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return ProjectedManifest{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, projectedManifestMembers); unknown {
		return ProjectedManifest{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, projectedManifestRequired); missing {
		return ProjectedManifest{}, invalid("%s misses a required member %q", owner, name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != projectedManifestSchema {
		return ProjectedManifest{}, invalid("%s schema is not urn:ax:schema:clone-projected-object-manifest", owner)
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != projectedManifestVersion {
		return ProjectedManifest{}, invalid("%s schema_version is not 1.0.0", owner)
	}
	claimed, ok := checkDigest(members["projected_object_manifest_id"])
	if !ok {
		return ProjectedManifest{}, invalid("%s projected_object_manifest_id is not a digest", owner)
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return ProjectedManifest{}, invalid("%s operation_id is not a UUIDv7", owner)
	}
	planID, ok := checkDigest(members["projection_plan_id"])
	if !ok {
		return ProjectedManifest{}, invalid("%s projection_plan_id is not a digest", owner)
	}
	targetTuple, err := sessadapter.DecodeTuple(members["target_environment"])
	if err != nil {
		return ProjectedManifest{}, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
	}
	nativeID, ok := checkStringBounds(members["expected_target_native_session_id"], 1, 512)
	if !ok {
		return ProjectedManifest{}, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	elements, ok := decodeArray(members["entries"])
	if !ok {
		return ProjectedManifest{}, invalid("%s entries is not an array", owner)
	}
	if err := checkProjectedEntryCount(len(elements)); err != nil {
		return ProjectedManifest{}, err
	}
	entries := make([]ProjectedEntry, 0, len(elements))
	for index, element := range elements {
		entry, err := decodeProjectedEntry(element, index)
		if err != nil {
			return ProjectedManifest{}, err
		}
		entries = append(entries, entry)
	}
	if err := checkProjectedEntryOrder(entries); err != nil {
		return ProjectedManifest{}, err
	}
	total, ok := checkUint53Bounds(members["total_bytes"], 0, maxUint53)
	if !ok {
		return ProjectedManifest{}, invalid("%s total_bytes is not a uint53", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ProjectedManifest{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	if err := verifySelfDigest(members, projectedManifestSelf, claimed); err != nil {
		return ProjectedManifest{}, err
	}
	return ProjectedManifest{
		ManifestID:                  claimed,
		OperationID:                 operation,
		ProjectionPlanID:            planID,
		TargetEnvironment:           targetTuple,
		ExpectedTargetNativeSession: nativeID,
		Entries:                     entries,
		TotalBytes:                  total,
	}, nil
}
