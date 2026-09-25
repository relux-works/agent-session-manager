package cloneplan

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates and constructs ProjectionItemMapping: the
// closed per-item row Section 13.14.2 states. Unlike the fidelity
// disposition row, the spec pins no coupling between
// expected_disposition and reason_codes or canonical_object_id here,
// so every combination of the shaped members admits.

var itemMappingMembers = map[string]bool{
	"source_item_key":      true,
	"canonical_object_id":  true,
	"target_resource_keys": true,
	"expected_disposition": true,
	"reason_codes":         true,
	"extensions":           true,
}

var itemMappingRequired = []string{
	"source_item_key",
	"canonical_object_id",
	"target_resource_keys",
	"expected_disposition",
	"reason_codes",
	"extensions",
}

// ItemMapping is one validated projection item mapping.
type ItemMapping struct {
	SourceItemKey     string
	CanonicalObjectID *scalar.Digest
	TargetResourceKey []string
	ExpectedDispos    string
	ReasonCodes       []string
}

// ItemMappingInput is the caller-supplied item mapping candidate
// for Build. A nil CanonicalObjectID seals as null.
type ItemMappingInput struct {
	SourceItemKey     string
	CanonicalObjectID *string
	TargetResourceKey []string
	ExpectedDispos    string
	ReasonCodes       []string
	Extensions        map[string]any
}

// buildItemMapping validates one caller-supplied mapping and
// renders its closed object. Every rule is a refusal: broken
// bounds, unsorted or duplicate arrays, unknown dispositions, and
// out-of-vocabulary reasons are refused, never repaired.
func buildItemMapping(input ItemMappingInput, index int) (ItemMapping, map[string]any, error) {
	owner := fmt.Sprintf("projection item mapping[%d]", index)
	if !validText(input.SourceItemKey) {
		return ItemMapping{}, nil, invalid("%s source_item_key is not valid UTF-8", owner)
	}
	if length := stringLength(input.SourceItemKey); length < 1 || length > 512 {
		return ItemMapping{}, nil, invalid("%s source_item_key is not a string[1..512]", owner)
	}
	var canonical *scalar.Digest
	var canonicalValue any
	if input.CanonicalObjectID != nil {
		id, err := scalar.ParseDigest(*input.CanonicalObjectID)
		if err != nil {
			return ItemMapping{}, nil, invalid("%s canonical_object_id is not a digest: %v", owner, err)
		}
		canonical = &id
		canonicalValue = id.String()
	}
	keys, err := checkSortedUniqueBoundedStrings(input.TargetResourceKey, 1, 512, 0, 65536, owner, "target_resource_keys")
	if err != nil {
		return ItemMapping{}, nil, err
	}
	if !validText(input.ExpectedDispos) {
		return ItemMapping{}, nil, invalid("%s expected_disposition is not valid UTF-8", owner)
	}
	if !clonefidelity.ValidDisposition(input.ExpectedDispos) {
		return ItemMapping{}, nil, invalid("%s expected_disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner)
	}
	reasons, err := checkSortedUniqueReasonStrings(input.ReasonCodes, 0, 128, owner)
	if err != nil {
		return ItemMapping{}, nil, err
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return ItemMapping{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return ItemMapping{
			SourceItemKey:     input.SourceItemKey,
			CanonicalObjectID: canonical,
			TargetResourceKey: keys,
			ExpectedDispos:    input.ExpectedDispos,
			ReasonCodes:       reasons,
		}, map[string]any{
			"source_item_key":      input.SourceItemKey,
			"canonical_object_id":  canonicalValue,
			"target_resource_keys": stringValues(keys),
			"expected_disposition": input.ExpectedDispos,
			"reason_codes":         stringValues(reasons),
			"extensions":           extensions,
		}, nil
}

// decodeItemMapping validates one closed ProjectionItemMapping.
func decodeItemMapping(raw json.RawMessage, index int) (ItemMapping, error) {
	owner := "projection item mapping"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return ItemMapping{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, itemMappingMembers); unknown {
		return ItemMapping{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, itemMappingRequired); missing {
		return ItemMapping{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	key, ok := checkStringBounds(members["source_item_key"], 1, 512)
	if !ok {
		return ItemMapping{}, invalid("%s[%d] source_item_key is not a string[1..512]", owner, index)
	}
	var canonical *scalar.Digest
	if !isNull(members["canonical_object_id"]) {
		id, ok := checkDigest(members["canonical_object_id"])
		if !ok {
			return ItemMapping{}, invalid("%s[%d] canonical_object_id is not a digest or null", owner, index)
		}
		canonical = &id
	}
	keys, ok := checkSortedUniqueStrings(members["target_resource_keys"], 1, 512, 0, 65536)
	if !ok {
		return ItemMapping{}, invalid("%s[%d] target_resource_keys are not sorted unique string[1..512][0..65536]", owner, index)
	}
	disposition, ok := rawString(members["expected_disposition"])
	if !ok || !clonefidelity.ValidDisposition(disposition) {
		return ItemMapping{}, invalid("%s[%d] expected_disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, index)
	}
	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)
	if !ok {
		return ItemMapping{}, invalid("%s[%d] reason_codes are not sorted unique string[1..128][0..128]", owner, index)
	}
	for _, reason := range reasons {
		if !clonefidelity.ValidReasonCode(reason) {
			return ItemMapping{}, invalid("%s[%d] reason_codes carry %q outside the core and reverse-DNS reason vocabulary", owner, index, reason)
		}
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ItemMapping{}, invalid("%s[%d] extensions %s", owner, index, extensionsFault)
	}
	return ItemMapping{
		SourceItemKey:     key,
		CanonicalObjectID: canonical,
		TargetResourceKey: keys,
		ExpectedDispos:    disposition,
		ReasonCodes:       reasons,
	}, nil
}

// checkMappingOrder enforces the plan prose "ordered item mappings":
// mappings are sorted unique by source item key. Both entries share
// this gate over the sealed row set.
func checkMappingOrder(mappings []ItemMapping) error {
	for index := 1; index < len(mappings); index++ {
		if mappings[index].SourceItemKey <= mappings[index-1].SourceItemKey {
			return invalid("projection plan item_mappings are not sorted unique by source item key")
		}
	}
	return nil
}

// checkMappingCount enforces the item_mappings cardinality bound
// ProjectionItemMapping[1..1000000]. The gate is factored so the
// exact upper edge pins at the gate while entry reachability pins
// at both entries.
func checkMappingCount(count int) error {
	if count < 1 || count > 1000000 {
		return invalid("projection plan carries %d item_mappings, want [1..1000000]", count)
	}
	return nil
}
