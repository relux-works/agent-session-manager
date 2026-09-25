package cloneplan

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// This file validates and constructs ProjectionTargetOperation: the
// closed sequenced target step Section 13.14.2 states. Operations
// are ordered by sequence, and every dependency names a strictly
// lower existing sequence, which makes the dependency graph a DAG
// by construction: a cycle would need an edge that is not strictly
// decreasing.

var targetOperationMembers = map[string]bool{
	"sequence":             true,
	"action":               true,
	"resource_keys":        true,
	"depends_on_sequences": true,
	"extensions":           true,
}

var targetOperationRequired = []string{
	"sequence",
	"action",
	"resource_keys",
	"depends_on_sequences",
	"extensions",
}

// TargetOperation is one validated projection target operation.
type TargetOperation struct {
	Sequence           uint64
	Action             string
	ResourceKeys       []string
	DependsOnSequences []uint64
}

// TargetOperationInput is the caller-supplied target operation
// candidate for Build.
type TargetOperationInput struct {
	Sequence           uint64
	Action             string
	ResourceKeys       []string
	DependsOnSequences []uint64
	Extensions         map[string]any
}

// buildTargetOperation validates one caller-supplied operation and
// renders its closed object. Sequence positivity, the action
// vocabulary, and the sorted-unique arrays are refusals here; the
// cross-operation order and DAG rules run over the sealed set in
// checkOperationOrder and checkOperationDAG.
func buildTargetOperation(input TargetOperationInput, index int) (TargetOperation, map[string]any, error) {
	owner := fmt.Sprintf("projection target operation[%d]", index)
	if input.Sequence == 0 || input.Sequence > maxUint53 {
		return TargetOperation{}, nil, invalid("%s sequence is not a uint53>0", owner)
	}
	if !validText(input.Action) {
		return TargetOperation{}, nil, invalid("%s action is not valid UTF-8", owner)
	}
	if !ValidOperationAction(input.Action) {
		return TargetOperation{}, nil, invalid("%s action is outside create_directory|write_blob|write_native_record|rebuild_index", owner)
	}
	keys, err := checkSortedUniqueBoundedStrings(input.ResourceKeys, 1, 512, 1, 65536, owner, "resource_keys")
	if err != nil {
		return TargetOperation{}, nil, err
	}
	depends, err := checkSortedUniqueUint53Inputs(input.DependsOnSequences, 0, maxUint53, 0, 65536, owner, "depends_on_sequences")
	if err != nil {
		return TargetOperation{}, nil, err
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return TargetOperation{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return TargetOperation{
			Sequence:           input.Sequence,
			Action:             input.Action,
			ResourceKeys:       keys,
			DependsOnSequences: depends,
		}, map[string]any{
			"sequence":             input.Sequence,
			"action":               input.Action,
			"resource_keys":        stringValues(keys),
			"depends_on_sequences": uintValues(depends),
			"extensions":           extensions,
		}, nil
}

// decodeTargetOperation validates one closed
// ProjectionTargetOperation.
func decodeTargetOperation(raw json.RawMessage, index int) (TargetOperation, error) {
	owner := "projection target operation"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return TargetOperation{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, targetOperationMembers); unknown {
		return TargetOperation{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, targetOperationRequired); missing {
		return TargetOperation{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	sequence, ok := checkUint53Bounds(members["sequence"], 1, maxUint53)
	if !ok {
		return TargetOperation{}, invalid("%s[%d] sequence is not a uint53>0", owner, index)
	}
	action, ok := rawString(members["action"])
	if !ok || !ValidOperationAction(action) {
		return TargetOperation{}, invalid("%s[%d] action is outside create_directory|write_blob|write_native_record|rebuild_index", owner, index)
	}
	keys, ok := checkSortedUniqueStrings(members["resource_keys"], 1, 512, 1, 65536)
	if !ok {
		return TargetOperation{}, invalid("%s[%d] resource_keys are not sorted unique string[1..512][1..65536]", owner, index)
	}
	rowOwner := fmt.Sprintf("%s[%d]", owner, index)
	depends, err := checkSortedUniqueUint53s(members["depends_on_sequences"], 0, maxUint53, 0, 65536, rowOwner, "depends_on_sequences")
	if err != nil {
		return TargetOperation{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return TargetOperation{}, invalid("%s[%d] extensions %s", owner, index, extensionsFault)
	}
	return TargetOperation{
		Sequence:           sequence,
		Action:             action,
		ResourceKeys:       keys,
		DependsOnSequences: depends,
	}, nil
}

// checkOperationOrder enforces "ordered by sequence": sequences are
// strictly increasing, so no two operations share a sequence. Both
// entries share this gate over the sealed operation set.
func checkOperationOrder(operations []TargetOperation) error {
	for index := 1; index < len(operations); index++ {
		if operations[index].Sequence <= operations[index-1].Sequence {
			return invalid("projection plan target_operations are not ordered by sequence")
		}
	}
	return nil
}

// checkOperationDAG enforces "Dependencies are lower sequences and
// form a DAG": every dependency names a strictly lower sequence
// that an operation in this plan carries. Strict decrease along
// every edge admits no cycle and no self-edge, so a set passing
// this gate is a DAG over the plan operations. Both entries share
// this gate over the sealed operation set.
func checkOperationDAG(operations []TargetOperation) error {
	present := make(map[uint64]bool, len(operations))
	for _, operation := range operations {
		present[operation.Sequence] = true
	}
	for _, operation := range operations {
		for _, dependency := range operation.DependsOnSequences {
			if dependency >= operation.Sequence {
				return invalid("projection plan target operation %d depends on sequence %d, want a lower sequence", operation.Sequence, dependency)
			}
			if !present[dependency] {
				return invalid("projection plan target operation %d depends on absent sequence %d", operation.Sequence, dependency)
			}
		}
	}
	return nil
}

// checkOperationCount enforces the target_operations cardinality
// bound ProjectionTargetOperation[1..65536]. The gate is factored
// so the exact upper edge pins at the gate while entry reachability
// pins at both entries.
func checkOperationCount(count int) error {
	if count < 1 || count > 65536 {
		return invalid("projection plan carries %d target_operations, want [1..65536]", count)
	}
	return nil
}
