package clonebundle

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the CaptureItem, Capture Source Basis, and
// Capture Boundary closed shapes of Clone Capture Manifest 1.0.0.
// The manifest envelope itself lives in capturebuild.go.

// captureClasses is the closed nine-class vocabulary Section 13.14.1
// states for CaptureItem. It mirrors the sessadapter capture-plan
// vocabulary over the same pinned section; membership here is the
// manifest-side gate.
var captureClasses = []string{
	"durable_payload",
	"durable_index_required",
	"durable_sidecar",
	"derived_cache_optional",
	"credential",
	"machine_auth",
	"runtime_state",
	"transient_lock",
	"unknown",
}

// ValidCaptureClass reports whether the name is a capture class.
func ValidCaptureClass(name string) bool {
	for _, class := range captureClasses {
		if class == name {
			return true
		}
	}
	return false
}

// alwaysExcludedClass reports whether the class is always excluded:
// credential, auth, runtime, and lock classes are never included.
func alwaysExcludedClass(class string) bool {
	switch class {
	case "credential", "machine_auth", "runtime_state", "transient_lock":
		return true
	default:
		return false
	}
}

var captureItemMembers = map[string]bool{
	"native_item_key":    true,
	"class":              true,
	"disposition":        true,
	"blob_descriptor_id": true,
	"byte_count":         true,
	"exclusion_reason":   true,
	"extensions":         true,
}

var captureItemRequired = []string{
	"native_item_key",
	"class",
	"disposition",
	"blob_descriptor_id",
	"byte_count",
	"exclusion_reason",
	"extensions",
}

// CaptureItem is one validated capture item.
type CaptureItem struct {
	NativeItemKey    string
	Class            string
	Disposition      string
	BlobDescriptorID *scalar.Digest
	ByteCount        *uint64
	ExclusionReason  *string
	Extensions       map[string]any
}

// Included reports whether the item is an included row.
func (item CaptureItem) Included() bool { return item.Disposition == "included" }

// decodeCaptureItem validates one closed CaptureItem: the
// included/excluded content rule, the always-excluded classes, and
// the row-21 exclusion allowlist on the native key.
func decodeCaptureItem(raw json.RawMessage, index int) (CaptureItem, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return CaptureItem{}, invalid("capture item[%d] %s (%s)", index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, captureItemMembers); unknown {
		return CaptureItem{}, invalid("capture item[%d] carries unknown member %q", index, name)
	}
	if name, missing := missingMember(members, captureItemRequired); missing {
		return CaptureItem{}, invalid("capture item[%d] misses a required member %q", index, name)
	}
	key, ok := checkStringBounds(members["native_item_key"], 1, 512)
	if !ok {
		return CaptureItem{}, invalid("capture item[%d] native_item_key is not a string[1..512]", index)
	}
	if err := SanitizeNativeKey(key); err != nil {
		return CaptureItem{}, err
	}
	if err := refuseExcludedMember("capture item", key, index); err != nil {
		return CaptureItem{}, err
	}
	class, ok := rawString(members["class"])
	if !ok || !ValidCaptureClass(class) {
		return CaptureItem{}, invalid("capture item[%d] class %q is outside the nine-class vocabulary", index, class)
	}
	disposition, ok := rawString(members["disposition"])
	if !ok || (disposition != "included" && disposition != "excluded") {
		return CaptureItem{}, invalid("capture item[%d] disposition is outside included|excluded", index)
	}
	var descriptor *scalar.Digest
	if !isNull(members["blob_descriptor_id"]) {
		value, ok := checkDigest(members["blob_descriptor_id"])
		if !ok {
			return CaptureItem{}, invalid("capture item[%d] blob_descriptor_id is not a digest", index)
		}
		descriptor = &value
	}
	var count *uint64
	if !isNull(members["byte_count"]) {
		value, ok := checkUint53Bounds(members["byte_count"], 0, maxUint53)
		if !ok {
			return CaptureItem{}, invalid("capture item[%d] byte_count is not a uint53", index)
		}
		count = &value
	}
	var reason *string
	if !isNull(members["exclusion_reason"]) {
		value, ok := checkStringBounds(members["exclusion_reason"], 1, 128)
		if !ok {
			return CaptureItem{}, invalid("capture item[%d] exclusion_reason is not a string[1..128]", index)
		}
		reason = &value
	}
	if err := checkItemDisposition(index, class, disposition, descriptor != nil, count != nil, reason != nil); err != nil {
		return CaptureItem{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return CaptureItem{}, invalid("capture item[%d] extensions %s", index, extensionsFault)
	}
	var decodedExtensions map[string]any
	if err := json.Unmarshal(bytesTrimSpace(members["extensions"]), &decodedExtensions); err != nil {
		return CaptureItem{}, invalid("capture item[%d] extensions are not an object", index)
	}
	if decodedExtensions == nil {
		decodedExtensions = map[string]any{}
	}
	return CaptureItem{
		NativeItemKey:    key,
		Class:            class,
		Disposition:      disposition,
		BlobDescriptorID: descriptor,
		ByteCount:        count,
		ExclusionReason:  reason,
		Extensions:       decodedExtensions,
	}, nil
}

// checkItemDisposition enforces the included/excluded rule: included
// requires both content members non-null and a null reason; excluded
// requires null content members and a non-null reason; the four
// always-excluded classes are never included.
func checkItemDisposition(index int, class, disposition string, hasDescriptor, hasCount, hasReason bool) error {
	if disposition == "included" {
		if alwaysExcludedClass(class) {
			return invalid("capture item[%d] class %q is always excluded, never included", index, class)
		}
		if !hasDescriptor || !hasCount {
			return invalid("capture item[%d] included requires non-null descriptor and byte count", index)
		}
		if hasReason {
			return invalid("capture item[%d] included carries an exclusion reason", index)
		}
		return nil
	}
	if hasDescriptor || hasCount {
		return invalid("capture item[%d] excluded carries content members", index)
	}
	if !hasReason {
		return invalid("capture item[%d] excluded requires a non-null exclusion reason", index)
	}
	return nil
}

// CaptureItemInput is one caller-supplied capture item candidate for
// Build. Null members are nil pointers: BlobDescriptorID and
// ByteCount for excluded rows, ExclusionReason for included rows.
type CaptureItemInput struct {
	NativeItemKey    string
	Class            string
	Disposition      string
	BlobDescriptorID *string
	ByteCount        *uint64
	ExclusionReason  *string
	Extensions       map[string]any
}

func buildCaptureItem(input CaptureItemInput, index int) (CaptureItem, error) {
	if stringLength(input.NativeItemKey) < 1 || stringLength(input.NativeItemKey) > 512 {
		return CaptureItem{}, invalid("capture item[%d] native_item_key is not a string[1..512]", index)
	}
	if err := SanitizeNativeKey(input.NativeItemKey); err != nil {
		return CaptureItem{}, err
	}
	if err := refuseExcludedMember("capture item", input.NativeItemKey, index); err != nil {
		return CaptureItem{}, err
	}
	if !ValidCaptureClass(input.Class) {
		return CaptureItem{}, invalid("capture item[%d] class %q is outside the nine-class vocabulary", index, input.Class)
	}
	if input.Disposition != "included" && input.Disposition != "excluded" {
		return CaptureItem{}, invalid("capture item[%d] disposition is outside included|excluded", index)
	}
	var descriptor *scalar.Digest
	if input.BlobDescriptorID != nil {
		value, err := scalar.ParseDigest(*input.BlobDescriptorID)
		if err != nil {
			return CaptureItem{}, invalid("capture item[%d] blob_descriptor_id is not a digest: %v", index, err)
		}
		descriptor = &value
	}
	var count *uint64
	if input.ByteCount != nil {
		if *input.ByteCount > maxUint53 {
			return CaptureItem{}, invalid("capture item[%d] byte_count exceeds uint53", index)
		}
		kept := *input.ByteCount
		count = &kept
	}
	var reason *string
	if input.ExclusionReason != nil {
		if !validText(*input.ExclusionReason) {
			return CaptureItem{}, invalid("capture item[%d] exclusion_reason is not valid UTF-8", index)
		}
		if stringLength(*input.ExclusionReason) < 1 || stringLength(*input.ExclusionReason) > 128 {
			return CaptureItem{}, invalid("capture item[%d] exclusion_reason is not a string[1..128]", index)
		}
		kept := *input.ExclusionReason
		reason = &kept
	}
	if err := checkItemDisposition(index, input.Class, input.Disposition, descriptor != nil, count != nil, reason != nil); err != nil {
		return CaptureItem{}, err
	}
	if _, err := encodeExtensions(input.Extensions); err != nil {
		return CaptureItem{}, invalid("capture item[%d]: %v", index, err)
	}
	return CaptureItem{
		NativeItemKey:    input.NativeItemKey,
		Class:            input.Class,
		Disposition:      input.Disposition,
		BlobDescriptorID: descriptor,
		ByteCount:        count,
		ExclusionReason:  reason,
		Extensions:       extensionValue(input.Extensions),
	}, nil
}

func captureItemObject(item CaptureItem) map[string]any {
	var descriptor any
	if item.BlobDescriptorID != nil {
		descriptor = item.BlobDescriptorID.String()
	}
	var count any
	if item.ByteCount != nil {
		count = *item.ByteCount
	}
	var reason any
	if item.ExclusionReason != nil {
		reason = *item.ExclusionReason
	}
	return map[string]any{
		"native_item_key":    item.NativeItemKey,
		"class":              item.Class,
		"disposition":        item.Disposition,
		"blob_descriptor_id": descriptor,
		"byte_count":         count,
		"exclusion_reason":   reason,
		"extensions":         extensionValue(item.Extensions),
	}
}
