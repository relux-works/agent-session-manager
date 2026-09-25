package cloneplan

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates and constructs ExpectedTargetResource: the
// closed expected target row Section 13.14.2 states. Blob and
// directory nullability is branch-exact: a blob names its expected
// blob, a directory names none. The mode member keeps its nullable
// uint32[0..4095] type on both branches — the pinned text types it
// nullable without a per-branch distinction, so no mode gate varies
// by kind.

var expectedResourceMembers = map[string]bool{
	"operation_sequence": true,
	"resource_key":       true,
	"kind":               true,
	"mode":               true,
	"expected_blob_id":   true,
	"extensions":         true,
}

var expectedResourceRequired = []string{
	"operation_sequence",
	"resource_key",
	"kind",
	"mode",
	"expected_blob_id",
	"extensions",
}

// maxMode is the uint32[0..4095] permission-bits bound Section
// 13.14.2 pins on the mode member.
const maxMode = uint64(4095)

// ExpectedResource is one validated expected target resource.
type ExpectedResource struct {
	OperationSequence uint64
	ResourceKey       string
	Kind              string
	Mode              *uint64
	ExpectedBlobID    *scalar.Digest
}

// ExpectedResourceInput is the caller-supplied expected resource
// candidate for Build. Nil Mode and ExpectedBlobID seal as null.
type ExpectedResourceInput struct {
	OperationSequence uint64
	ResourceKey       string
	Kind              string
	Mode              *uint64
	ExpectedBlobID    *string
	Extensions        map[string]any
}

// checkResourceBranch enforces branch-exact nullability: blob rows
// carry a non-null expected_blob_id, directory rows carry null.
// Both entries share this gate.
func checkResourceBranch(owner, kind string, hasBlob bool) error {
	if kind == "blob" && !hasBlob {
		return invalid("%s expected_blob_id is null for blob", owner)
	}
	if kind == "directory" && hasBlob {
		return invalid("%s expected_blob_id is not null for directory", owner)
	}
	return nil
}

// checkBuildMode validates one caller-supplied mode: null or a
// uint32 in [0..4095].
func checkBuildMode(mode *uint64, owner string) (*uint64, any, error) {
	if mode == nil {
		return nil, nil, nil
	}
	if *mode > maxMode {
		return nil, nil, invalid("%s mode is not a uint32[0..4095] or null", owner)
	}
	value := *mode
	return &value, value, nil
}

// buildExpectedResource validates one caller-supplied resource and
// renders its closed object. Every rule is a refusal: broken
// bounds, unknown kinds, out-of-range modes, and branch violations
// are refused, never repaired.
func buildExpectedResource(input ExpectedResourceInput, index int) (ExpectedResource, map[string]any, error) {
	owner := fmt.Sprintf("expected target resource[%d]", index)
	if input.OperationSequence == 0 || input.OperationSequence > maxUint53 {
		return ExpectedResource{}, nil, invalid("%s operation_sequence is not a uint53>0", owner)
	}
	if !validText(input.ResourceKey) {
		return ExpectedResource{}, nil, invalid("%s resource_key is not valid UTF-8", owner)
	}
	if length := stringLength(input.ResourceKey); length < 1 || length > 512 {
		return ExpectedResource{}, nil, invalid("%s resource_key is not a string[1..512]", owner)
	}
	if !validText(input.Kind) {
		return ExpectedResource{}, nil, invalid("%s kind is not valid UTF-8", owner)
	}
	if !ValidResourceKind(input.Kind) {
		return ExpectedResource{}, nil, invalid("%s kind is outside blob|directory", owner)
	}
	mode, modeValue, err := checkBuildMode(input.Mode, owner)
	if err != nil {
		return ExpectedResource{}, nil, err
	}
	var blob *scalar.Digest
	var blobValue any
	if input.ExpectedBlobID != nil {
		id, err := scalar.ParseDigest(*input.ExpectedBlobID)
		if err != nil {
			return ExpectedResource{}, nil, invalid("%s expected_blob_id is not a digest: %v", owner, err)
		}
		blob = &id
		blobValue = id.String()
	}
	if err := checkResourceBranch(owner, input.Kind, blob != nil); err != nil {
		return ExpectedResource{}, nil, err
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return ExpectedResource{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return ExpectedResource{
			OperationSequence: input.OperationSequence,
			ResourceKey:       input.ResourceKey,
			Kind:              input.Kind,
			Mode:              mode,
			ExpectedBlobID:    blob,
		}, map[string]any{
			"operation_sequence": input.OperationSequence,
			"resource_key":       input.ResourceKey,
			"kind":               input.Kind,
			"mode":               modeValue,
			"expected_blob_id":   blobValue,
			"extensions":         extensions,
		}, nil
}

// decodeExpectedResource validates one closed
// ExpectedTargetResource.
func decodeExpectedResource(raw json.RawMessage, index int) (ExpectedResource, error) {
	owner := "expected target resource"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return ExpectedResource{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, expectedResourceMembers); unknown {
		return ExpectedResource{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, expectedResourceRequired); missing {
		return ExpectedResource{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	sequence, ok := checkUint53Bounds(members["operation_sequence"], 1, maxUint53)
	if !ok {
		return ExpectedResource{}, invalid("%s[%d] operation_sequence is not a uint53>0", owner, index)
	}
	key, ok := checkStringBounds(members["resource_key"], 1, 512)
	if !ok {
		return ExpectedResource{}, invalid("%s[%d] resource_key is not a string[1..512]", owner, index)
	}
	kind, ok := rawString(members["kind"])
	if !ok || !ValidResourceKind(kind) {
		return ExpectedResource{}, invalid("%s[%d] kind is outside blob|directory", owner, index)
	}
	var mode *uint64
	if !isNull(members["mode"]) {
		value, ok := checkUint53Bounds(members["mode"], 0, maxMode)
		if !ok {
			return ExpectedResource{}, invalid("%s[%d] mode is not a uint32[0..4095] or null", owner, index)
		}
		mode = &value
	}
	var blob *scalar.Digest
	if !isNull(members["expected_blob_id"]) {
		id, ok := checkDigest(members["expected_blob_id"])
		if !ok {
			return ExpectedResource{}, invalid("%s[%d] expected_blob_id is not a digest or null", owner, index)
		}
		blob = &id
	}
	rowOwner := fmt.Sprintf("%s[%d]", owner, index)
	if err := checkResourceBranch(rowOwner, kind, blob != nil); err != nil {
		return ExpectedResource{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ExpectedResource{}, invalid("%s[%d] extensions %s", owner, index, extensionsFault)
	}
	return ExpectedResource{
		OperationSequence: sequence,
		ResourceKey:       key,
		Kind:              kind,
		Mode:              mode,
		ExpectedBlobID:    blob,
	}, nil
}

// checkExpectedResourceOrder enforces "sorted by operation
// sequence/key": the (sequence, key) pairs are strictly increasing.
// Both entries share this gate over the sealed resource set.
func checkExpectedResourceOrder(resources []ExpectedResource) error {
	for index := 1; index < len(resources); index++ {
		previous := resources[index-1]
		current := resources[index]
		if current.OperationSequence < previous.OperationSequence ||
			(current.OperationSequence == previous.OperationSequence && current.ResourceKey <= previous.ResourceKey) {
			return invalid("projection plan expected_resources are not sorted by operation sequence/key")
		}
	}
	return nil
}

// checkExpectedResourceCount enforces the expected_resources
// cardinality bound ExpectedTargetResource[0..65536]. The gate is
// factored so the exact upper edge pins at the gate while entry
// reachability pins at both entries.
func checkExpectedResourceCount(count int) error {
	if count < 0 || count > 65536 {
		return invalid("projection plan carries %d expected_resources, want [0..65536]", count)
	}
	return nil
}
