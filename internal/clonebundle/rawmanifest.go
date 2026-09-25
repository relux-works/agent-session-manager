package clonebundle

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Clone Raw Object Manifest 1.0.0:
// the closed sorted unique entry list over content-addressed blobs.
// Credential, machine-auth, runtime-state, transient-lock, and
// hosttrust-excluded candidates are forbidden here; they never become
// raw objects.

const (
	rawManifestSchema  = "urn:ax:schema:clone-raw-object-manifest"
	rawManifestVersion = "1.0.0"
	rawManifestSelf    = "raw_object_manifest_id"
)

// rawEntryClasses is the closed RawObjectEntry class subset:
// durable and derived-cache classes plus unknown. The four
// excluded classes (credential, machine_auth, runtime_state,
// transient_lock) are absent by construction, so membership in this
// table is the gate.
var rawEntryClasses = []string{
	"durable_payload",
	"durable_index_required",
	"durable_sidecar",
	"derived_cache_optional",
	"unknown",
}

func validRawEntryClass(class string) bool {
	for _, allowed := range rawEntryClasses {
		if class == allowed {
			return true
		}
	}
	return false
}

var rawManifestMembers = map[string]bool{
	"schema":                   true,
	"schema_version":           true,
	"raw_object_manifest_id":   true,
	"operation_id":             true,
	"source_environment":       true,
	"source_native_session_id": true,
	"source_identity_digest":   true,
	"capture_plan_digest":      true,
	"entries":                  true,
	"total_bytes":              true,
	"extensions":               true,
}

var rawManifestRequired = []string{
	"schema",
	"schema_version",
	"raw_object_manifest_id",
	"operation_id",
	"source_environment",
	"source_native_session_id",
	"source_identity_digest",
	"capture_plan_digest",
	"entries",
	"total_bytes",
	"extensions",
}

var rawEntryMembers = map[string]bool{
	"native_item_key":    true,
	"class":              true,
	"byte_count":         true,
	"blob_id":            true,
	"blob_descriptor_id": true,
}

var rawEntryRequired = []string{
	"native_item_key",
	"class",
	"byte_count",
	"blob_id",
	"blob_descriptor_id",
}

// RawObjectEntry is one validated raw object entry.
type RawObjectEntry struct {
	NativeItemKey    string
	Class            string
	ByteCount        uint64
	BlobID           scalar.Digest
	BlobDescriptorID scalar.Digest
}

// RawObjectManifest is one validated Clone Raw Object Manifest.
type RawObjectManifest struct {
	ManifestID            scalar.Digest
	OperationID           scalar.UUIDv7
	SourceEnvironment     sessadapter.Tuple
	SourceNativeSessionID string
	SourceIdentityDigest  scalar.Digest
	CapturePlanDigest     scalar.Digest
	Entries               []RawObjectEntry
	TotalBytes            uint64
}

// EntryInput is one caller-supplied raw entry candidate for Build.
// Descriptor carries the exact Section 10.2 Blob Descriptor bytes
// the entry references; Build verifies the descriptor's closed
// shape and identity and its blob-ID/byte-count agreement before
// the entry is admitted.
type EntryInput struct {
	NativeItemKey string
	Class         string
	ByteCount     uint64
	BlobID        string
	DescriptorID  string
	Descriptor    []byte
}

// BuildRawObjectManifest constructs one closed Clone Raw Object
// Manifest as canonical bytes. Identical inputs produce
// byte-identical outputs. Every closed-shape rule is a refusal:
// unknown classes, unsanitized or excluded native keys, unsorted or
// duplicate entries, descriptor disagreement, and total-byte
// mismatch are all refused, never repaired.
func BuildRawObjectManifest(
	operationID string,
	sourceEnvironment []byte,
	sourceNativeSessionID string,
	sourceIdentityDigest string,
	capturePlanDigest string,
	inputs []EntryInput,
	extensions map[string]any,
) ([]byte, error) {
	operation, err := scalar.ParseUUIDv7(operationID)
	if err != nil {
		return nil, invalid("raw object manifest operation_id is not a UUIDv7: %v", err)
	}
	tuple, err := sessadapter.DecodeTuple(json.RawMessage(sourceEnvironment))
	if err != nil {
		return nil, invalid("raw object manifest source_environment is not an Environment Tuple: %v", err)
	}
	if stringLength(sourceNativeSessionID) < 1 || stringLength(sourceNativeSessionID) > 512 {
		return nil, invalid("raw object manifest source_native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(sourceNativeSessionID); err != nil {
		return nil, err
	}
	identityDigest, err := scalar.ParseDigest(sourceIdentityDigest)
	if err != nil {
		return nil, invalid("raw object manifest source_identity_digest is not a digest: %v", err)
	}
	planDigest, err := scalar.ParseDigest(capturePlanDigest)
	if err != nil {
		return nil, invalid("raw object manifest capture_plan_digest is not a digest: %v", err)
	}
	if len(inputs) > 65536 {
		return nil, invalid("raw object manifest carries %d entries, maximum is 65536", len(inputs))
	}
	entries, total, err := buildRawEntries(inputs)
	if err != nil {
		return nil, err
	}
	if _, err := encodeExtensions(extensions); err != nil {
		return nil, err
	}
	object := map[string]any{
		"schema":                   rawManifestSchema,
		"schema_version":           rawManifestVersion,
		"operation_id":             operation.String(),
		"source_environment":       tupleObject(tuple),
		"source_native_session_id": sourceNativeSessionID,
		"source_identity_digest":   identityDigest.String(),
		"capture_plan_digest":      planDigest.String(),
		"entries":                  rawEntryObjects(entries),
		"total_bytes":              total,
		"extensions":               extensionValue(extensions),
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	manifestID := scalar.SHA256Digest(omitted)
	object[rawManifestSelf] = manifestID.String()
	return canonicalizeObject(object)
}

func buildRawEntries(inputs []EntryInput) ([]RawObjectEntry, uint64, error) {
	entries := make([]RawObjectEntry, 0, len(inputs))
	previous := ""
	var total uint64
	for index, input := range inputs {
		if stringLength(input.NativeItemKey) < 1 || stringLength(input.NativeItemKey) > 512 {
			return nil, 0, invalid("raw object entry[%d] native_item_key is not a string[1..512]", index)
		}
		if err := SanitizeNativeKey(input.NativeItemKey); err != nil {
			return nil, 0, err
		}
		if err := refuseExcludedMember("raw object entry", input.NativeItemKey, index); err != nil {
			return nil, 0, err
		}
		if !validRawEntryClass(input.Class) {
			return nil, 0, invalid("raw object entry[%d] class %q is outside the raw subset", index, input.Class)
		}
		if input.ByteCount > maxUint53 {
			return nil, 0, invalid("raw object entry[%d] byte_count exceeds uint53", index)
		}
		blobID, err := scalar.ParseDigest(input.BlobID)
		if err != nil {
			return nil, 0, invalid("raw object entry[%d] blob_id is not a digest: %v", index, err)
		}
		descriptorID, err := scalar.ParseDigest(input.DescriptorID)
		if err != nil {
			return nil, 0, invalid("raw object entry[%d] blob_descriptor_id is not a digest: %v", index, err)
		}
		if err := VerifyDescriptorAgreement(input.Descriptor, descriptorID, blobID, input.ByteCount); err != nil {
			return nil, 0, invalid("raw object entry[%d]: %v", index, err)
		}
		if total > maxUint53-input.ByteCount {
			return nil, 0, invalid("raw object manifest total_bytes exceeds uint53")
		}
		total += input.ByteCount
		if index > 0 && input.NativeItemKey <= previous {
			return nil, 0, invalid("raw object manifest entries are not sorted unique by native item key")
		}
		previous = input.NativeItemKey
		entries = append(entries, RawObjectEntry{
			NativeItemKey:    input.NativeItemKey,
			Class:            input.Class,
			ByteCount:        input.ByteCount,
			BlobID:           blobID,
			BlobDescriptorID: descriptorID,
		})
	}
	return entries, total, nil
}

// refuseExcludedMember is the row-21 exclusion allowlist: a native
// key matching hosttrust.MatchExcludedFromReplication or
// hosttrust.ExcludedConfigDirName is trust-adjacent material that
// must never enter a clone bundle, so construction refuses it. The
// owner names the calling shape (raw entry, capture item) so the
// refusal identifies the exact member.
func refuseExcludedMember(owner, key string, index int) error {
	if hosttrust.MatchExcludedFromReplication(key) {
		return invalid("%s[%d] native key %q is excluded from replication", owner, index, key)
	}
	if hosttrust.ExcludedConfigDirName(key) {
		return invalid("%s[%d] native key %q is trust-adjacent config material", owner, index, key)
	}
	return nil
}

// DecodeRawObjectManifest validates one closed Clone Raw Object
// Manifest: exact members, schema/version, self-digest agreement,
// tuple, sanitized source identity, sorted unique entries, and a
// total equal to the sum of entry byte counts.
func DecodeRawObjectManifest(data []byte) (RawObjectManifest, error) {
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return RawObjectManifest{}, invalid("raw object manifest %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, rawManifestMembers); unknown {
		return RawObjectManifest{}, invalid("raw object manifest carries unknown member %q", name)
	}
	if name, missing := missingMember(members, rawManifestRequired); missing {
		return RawObjectManifest{}, invalid("raw object manifest misses a required member %q", name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != rawManifestSchema {
		return RawObjectManifest{}, invalid("raw object manifest schema is not the clone raw object manifest")
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != rawManifestVersion {
		return RawObjectManifest{}, invalid("raw object manifest version is not 1.0.0")
	}
	manifestID, ok := checkDigest(members[rawManifestSelf])
	if !ok {
		return RawObjectManifest{}, invalid("raw object manifest raw_object_manifest_id is not a digest")
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return RawObjectManifest{}, invalid("raw object manifest operation_id is not a UUIDv7")
	}
	tuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil {
		return RawObjectManifest{}, invalid("raw object manifest source_environment is not an Environment Tuple: %v", err)
	}
	nativeID, ok := checkStringBounds(members["source_native_session_id"], 1, 512)
	if !ok {
		return RawObjectManifest{}, invalid("raw object manifest source_native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(nativeID); err != nil {
		return RawObjectManifest{}, err
	}
	identityDigest, ok := checkDigest(members["source_identity_digest"])
	if !ok {
		return RawObjectManifest{}, invalid("raw object manifest source_identity_digest is not a digest")
	}
	planDigest, ok := checkDigest(members["capture_plan_digest"])
	if !ok {
		return RawObjectManifest{}, invalid("raw object manifest capture_plan_digest is not a digest")
	}
	entries, total, err := decodeRawEntries(members["entries"])
	if err != nil {
		return RawObjectManifest{}, err
	}
	claimed, ok := checkUint53Bounds(members["total_bytes"], 0, maxUint53)
	if !ok {
		return RawObjectManifest{}, invalid("raw object manifest total_bytes is not a uint53")
	}
	if claimed != total {
		return RawObjectManifest{}, invalid("raw object manifest total_bytes is %d, entries sum to %d", claimed, total)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return RawObjectManifest{}, invalid("raw object manifest extensions %s", extensionsFault)
	}
	if err := verifySelfDigest(members, rawManifestSelf, manifestID); err != nil {
		return RawObjectManifest{}, err
	}
	return RawObjectManifest{
		ManifestID:            manifestID,
		OperationID:           operation,
		SourceEnvironment:     tuple,
		SourceNativeSessionID: nativeID,
		SourceIdentityDigest:  identityDigest,
		CapturePlanDigest:     planDigest,
		Entries:               entries,
		TotalBytes:            total,
	}, nil
}

func decodeRawEntries(raw json.RawMessage) ([]RawObjectEntry, uint64, error) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, 0, invalid("raw object manifest entries are not an array")
	}
	if len(elements) > 65536 {
		return nil, 0, invalid("raw object manifest carries %d entries, maximum is 65536", len(elements))
	}
	entries := make([]RawObjectEntry, 0, len(elements))
	previous := ""
	var total uint64
	for index, element := range elements {
		members, fault := decodeStrictObject(bytesTrimSpace(element))
		if fault != nil {
			return nil, 0, invalid("raw object entry[%d] %s (%s)", index, fault.detail, memberField(fault.member))
		}
		if name, unknown := unknownMember(members, rawEntryMembers); unknown {
			return nil, 0, invalid("raw object entry[%d] carries unknown member %q", index, name)
		}
		if name, missing := missingMember(members, rawEntryRequired); missing {
			return nil, 0, invalid("raw object entry[%d] misses a required member %q", index, name)
		}
		key, ok := checkStringBounds(members["native_item_key"], 1, 512)
		if !ok {
			return nil, 0, invalid("raw object entry[%d] native_item_key is not a string[1..512]", index)
		}
		if err := SanitizeNativeKey(key); err != nil {
			return nil, 0, err
		}
		if err := refuseExcludedMember("raw object entry", key, index); err != nil {
			return nil, 0, err
		}
		class, ok := rawString(members["class"])
		if !ok || !validRawEntryClass(class) {
			return nil, 0, invalid("raw object entry[%d] class %q is outside the raw subset", index, class)
		}
		count, ok := checkUint53Bounds(members["byte_count"], 0, maxUint53)
		if !ok {
			return nil, 0, invalid("raw object entry[%d] byte_count is not a uint53", index)
		}
		blobID, ok := checkDigest(members["blob_id"])
		if !ok {
			return nil, 0, invalid("raw object entry[%d] blob_id is not a digest", index)
		}
		descriptorID, ok := checkDigest(members["blob_descriptor_id"])
		if !ok {
			return nil, 0, invalid("raw object entry[%d] blob_descriptor_id is not a digest", index)
		}
		if total > maxUint53-count {
			return nil, 0, invalid("raw object manifest total_bytes exceeds uint53")
		}
		total += count
		if index > 0 && key <= previous {
			return nil, 0, invalid("raw object manifest entries are not sorted unique by native item key")
		}
		previous = key
		entries = append(entries, RawObjectEntry{
			NativeItemKey:    key,
			Class:            class,
			ByteCount:        count,
			BlobID:           blobID,
			BlobDescriptorID: descriptorID,
		})
	}
	return entries, total, nil
}

// VerifyRawManifestDescriptors re-checks descriptor agreement for a
// decoded manifest: every entry's blob ID and byte count must agree
// with the fetched Section 10.2 Blob Descriptor bytes, and the
// fetched bytes must be the exact descriptor the entry's
// blob_descriptor_id names. Decode validates structure; this entry
// validates evidence.
func VerifyRawManifestDescriptors(manifest RawObjectManifest, fetch func(descriptorID string) ([]byte, error)) error {
	for index, entry := range manifest.Entries {
		descriptor, err := fetch(entry.BlobDescriptorID.String())
		if err != nil {
			return invalid("raw object entry[%d]: descriptor fetch failed: %v", index, err)
		}
		if len(descriptor) == 0 {
			return invalid("raw object entry[%d]: descriptor is absent", index)
		}
		if err := VerifyDescriptorAgreement(descriptor, entry.BlobDescriptorID, entry.BlobID, entry.ByteCount); err != nil {
			return invalid("raw object entry[%d]: %v", index, err)
		}
	}
	return nil
}

// IdentityDigest computes the terminal digest of canonical sanitized
// NativeIdentity bytes: the source_identity_digest production entry.
// The encoded bytes re-decode before hashing, exactly as
// buildCaptureManifest re-decodes the source identity, so no digest
// is ever sealed over an identity decoding would refuse.
func IdentityDigest(identity NativeIdentity, extensions map[string]any) (scalar.Digest, error) {
	if err := SanitizeNativeKey(identity.NativeSessionID); err != nil {
		return scalar.Digest{}, err
	}
	encoded, err := EncodeNativeIdentity(identity, extensions)
	if err != nil {
		return scalar.Digest{}, err
	}
	if _, err := DecodeNativeIdentity(json.RawMessage(encoded)); err != nil {
		return scalar.Digest{}, invalid("native identity digest seals no undecodable identity: %v", err)
	}
	return scalar.SHA256Digest(encoded), nil
}

func tupleObject(tuple sessadapter.Tuple) map[string]any {
	return map[string]any{
		"environment_id":           tuple.EnvironmentID,
		"environment_version":      tuple.Version,
		"platform":                 tuple.Platform,
		"architecture":             tuple.Architecture,
		"store_schema_fingerprint": tuple.StoreFingerprint,
		"adapter_version":          tuple.AdapterVersion,
	}
}

func rawEntryObjects(entries []RawObjectEntry) []any {
	objects := make([]any, 0, len(entries))
	for _, entry := range entries {
		objects = append(objects, map[string]any{
			"native_item_key":    entry.NativeItemKey,
			"class":              entry.Class,
			"byte_count":         entry.ByteCount,
			"blob_id":            entry.BlobID.String(),
			"blob_descriptor_id": entry.BlobDescriptorID.String(),
		})
	}
	return objects
}

func extensionValue(extensions map[string]any) map[string]any {
	if extensions == nil {
		return map[string]any{}
	}
	return extensions
}

// EncodeExtensions validates caller-supplied extensions for
// sibling Build entries: valid UTF-8 text at every depth (before
// marshaling, so nothing rewrites to U+FFFD) plus the full closed
// extensions rule. It delegates to the single implementation below
// so the rule keeps one owner.
func EncodeExtensions(extensions map[string]any) ([]byte, error) {
	return encodeExtensions(extensions)
}

func encodeExtensions(extensions map[string]any) ([]byte, error) {
	value := extensionValue(extensions)
	if !validExtensionText(value) {
		return nil, invalid("extensions carry text that is not valid UTF-8")
	}
	plain, err := json.Marshal(value)
	if err != nil {
		return nil, invalid("extensions are not JSON: %v", err)
	}
	if fault := checkExtensionsClosed(json.RawMessage(plain)); fault != nil {
		return nil, invalid("extensions %s", fault)
	}
	return plain, nil
}
