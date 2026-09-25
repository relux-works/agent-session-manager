package cloneplan

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/environ"
)

// This file validates and constructs the four closed plan
// components Section 13.14.2 states (TransactionPlan, ReadBackPlan,
// ResumeProjectionPlan, RollbackPlan) and ContractRequirement. Every
// constant is a refusal: any value but the pinned one refuses, on
// both entries, including non-string and non-boolean JSON types on
// decode. ContractRequirement carries exactly contract_id and
// version — the pinned text states no extensions member for it, so
// an extensions member there is unknown.

var transactionPlanMembers = map[string]bool{
	"materialization_intent":  true,
	"target_collision_policy": true,
	"activation":              true,
	"extensions":              true,
}

var transactionPlanRequired = []string{
	"materialization_intent",
	"target_collision_policy",
	"activation",
	"extensions",
}

var readBackPlanMembers = map[string]bool{
	"modes":                   true,
	"require_identity_match":  true,
	"require_workspace_match": true,
	"require_semantic_marker": true,
	"extensions":              true,
}

var readBackPlanRequired = []string{
	"modes",
	"require_identity_match",
	"require_workspace_match",
	"require_semantic_marker",
	"extensions",
}

var resumePlanMembers = map[string]bool{
	"opens_existing_identity":            true,
	"allow_blank_fallback":               true,
	"bounded_continuation_turn_required": true,
	"extensions":                         true,
}

var resumePlanRequired = []string{
	"opens_existing_identity",
	"allow_blank_fallback",
	"bounded_continuation_turn_required",
	"extensions",
}

var rollbackPlanMembers = map[string]bool{
	"required":                        true,
	"retain_through":                  true,
	"forbidden_after_provider_commit": true,
	"extensions":                      true,
}

var rollbackPlanRequired = []string{
	"required",
	"retain_through",
	"forbidden_after_provider_commit",
	"extensions",
}

var contractRequirementMembers = map[string]bool{
	"contract_id": true,
	"version":     true,
}

var contractRequirementRequired = []string{
	"contract_id",
	"version",
}

// TransactionPlan is one validated closed transaction plan.
type TransactionPlan struct{}

// TransactionPlanInput is the caller-supplied transaction plan
// candidate for Build.
type TransactionPlanInput struct {
	MaterializationIntent string
	TargetCollisionPolicy string
	Activation            string
	Extensions            map[string]any
}

// ReadBackPlan is one validated closed read-back plan.
type ReadBackPlan struct{}

// ReadBackPlanInput is the caller-supplied read-back plan candidate
// for Build.
type ReadBackPlanInput struct {
	Modes                 []string
	RequireIdentityMatch  bool
	RequireWorkspaceMatch bool
	RequireSemanticMarker bool
	Extensions            map[string]any
}

// ResumePlan is one validated closed resume projection plan.
type ResumePlan struct {
	BoundedContinuationTurnRequired bool
}

// ResumePlanInput is the caller-supplied resume plan candidate for
// Build.
type ResumePlanInput struct {
	OpensExistingIdentity           bool
	AllowBlankFallback              bool
	BoundedContinuationTurnRequired bool
	Extensions                      map[string]any
}

// RollbackPlan is one validated closed rollback plan.
type RollbackPlan struct{}

// RollbackPlanInput is the caller-supplied rollback plan candidate
// for Build.
type RollbackPlanInput struct {
	Required                     bool
	RetainThrough                string
	ForbiddenAfterProviderCommit bool
	Extensions                   map[string]any
}

// ContractRequirement is one validated contract requirement.
type ContractRequirement struct {
	ContractID string
	Version    string
}

// ContractRequirementInput is the caller-supplied contract
// requirement candidate for Build.
type ContractRequirementInput struct {
	ContractID string
	Version    string
}

// buildTransactionPlan validates one caller-supplied transaction
// plan and renders its closed object: materialization_intent=clone,
// target_collision_policy=must_be_absent,
// activation=dormant_validated.
func buildTransactionPlan(input TransactionPlanInput) (TransactionPlan, map[string]any, error) {
	owner := "transaction plan"
	if !validText(input.MaterializationIntent) {
		return TransactionPlan{}, nil, invalid("%s materialization_intent is not valid UTF-8", owner)
	}
	if input.MaterializationIntent != "clone" {
		return TransactionPlan{}, nil, invalid("%s materialization_intent is not clone", owner)
	}
	if !validText(input.TargetCollisionPolicy) {
		return TransactionPlan{}, nil, invalid("%s target_collision_policy is not valid UTF-8", owner)
	}
	if input.TargetCollisionPolicy != "must_be_absent" {
		return TransactionPlan{}, nil, invalid("%s target_collision_policy is not must_be_absent", owner)
	}
	if !validText(input.Activation) {
		return TransactionPlan{}, nil, invalid("%s activation is not valid UTF-8", owner)
	}
	if input.Activation != "dormant_validated" {
		return TransactionPlan{}, nil, invalid("%s activation is not dormant_validated", owner)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return TransactionPlan{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return TransactionPlan{}, map[string]any{
		"materialization_intent":  "clone",
		"target_collision_policy": "must_be_absent",
		"activation":              "dormant_validated",
		"extensions":              extensions,
	}, nil
}

// decodeTransactionPlan validates one closed TransactionPlan.
func decodeTransactionPlan(raw json.RawMessage) (TransactionPlan, error) {
	owner := "transaction plan"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return TransactionPlan{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, transactionPlanMembers); unknown {
		return TransactionPlan{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, transactionPlanRequired); missing {
		return TransactionPlan{}, invalid("%s misses a required member %q", owner, name)
	}
	intent, ok := rawString(members["materialization_intent"])
	if !ok || intent != "clone" {
		return TransactionPlan{}, invalid("%s materialization_intent is not clone", owner)
	}
	policy, ok := rawString(members["target_collision_policy"])
	if !ok || policy != "must_be_absent" {
		return TransactionPlan{}, invalid("%s target_collision_policy is not must_be_absent", owner)
	}
	activation, ok := rawString(members["activation"])
	if !ok || activation != "dormant_validated" {
		return TransactionPlan{}, invalid("%s activation is not dormant_validated", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return TransactionPlan{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	return TransactionPlan{}, nil
}

// checkReadBackModes validates the closed modes constant
// [staged,live]: exactly two elements in exactly that order.
func checkReadBackModes(modes []string) error {
	if len(modes) != 2 || modes[0] != "staged" || modes[1] != "live" {
		return invalid("read-back plan modes are not [staged,live]")
	}
	return nil
}

// buildReadBackPlan validates one caller-supplied read-back plan
// and renders its closed object: modes=[staged,live] with all three
// require flags true.
func buildReadBackPlan(input ReadBackPlanInput) (ReadBackPlan, map[string]any, error) {
	owner := "read-back plan"
	for _, mode := range input.Modes {
		if !validText(mode) {
			return ReadBackPlan{}, nil, invalid("%s modes carry a member that is not valid UTF-8", owner)
		}
	}
	if err := checkReadBackModes(input.Modes); err != nil {
		return ReadBackPlan{}, nil, err
	}
	if !input.RequireIdentityMatch {
		return ReadBackPlan{}, nil, invalid("%s require_identity_match is not true", owner)
	}
	if !input.RequireWorkspaceMatch {
		return ReadBackPlan{}, nil, invalid("%s require_workspace_match is not true", owner)
	}
	if !input.RequireSemanticMarker {
		return ReadBackPlan{}, nil, invalid("%s require_semantic_marker is not true", owner)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return ReadBackPlan{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return ReadBackPlan{}, map[string]any{
		"modes":                   []any{"staged", "live"},
		"require_identity_match":  true,
		"require_workspace_match": true,
		"require_semantic_marker": true,
		"extensions":              extensions,
	}, nil
}

// decodeReadBackPlan validates one closed ReadBackPlan.
func decodeReadBackPlan(raw json.RawMessage) (ReadBackPlan, error) {
	owner := "read-back plan"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return ReadBackPlan{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, readBackPlanMembers); unknown {
		return ReadBackPlan{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, readBackPlanRequired); missing {
		return ReadBackPlan{}, invalid("%s misses a required member %q", owner, name)
	}
	elements, ok := decodeArray(members["modes"])
	if !ok {
		return ReadBackPlan{}, invalid("%s modes are not [staged,live]", owner)
	}
	modes := make([]string, 0, len(elements))
	for _, element := range elements {
		mode, ok := rawString(element)
		if !ok {
			return ReadBackPlan{}, invalid("%s modes are not [staged,live]", owner)
		}
		modes = append(modes, mode)
	}
	if err := checkReadBackModes(modes); err != nil {
		return ReadBackPlan{}, err
	}
	identity, ok := rawBool(members["require_identity_match"])
	if !ok || !identity {
		return ReadBackPlan{}, invalid("%s require_identity_match is not true", owner)
	}
	workspace, ok := rawBool(members["require_workspace_match"])
	if !ok || !workspace {
		return ReadBackPlan{}, invalid("%s require_workspace_match is not true", owner)
	}
	semantic, ok := rawBool(members["require_semantic_marker"])
	if !ok || !semantic {
		return ReadBackPlan{}, invalid("%s require_semantic_marker is not true", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ReadBackPlan{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	return ReadBackPlan{}, nil
}

// buildResumePlan validates one caller-supplied resume plan and
// renders its closed object: opens_existing_identity=true,
// allow_blank_fallback=false, and a boolean
// bounded_continuation_turn_required.
func buildResumePlan(input ResumePlanInput) (ResumePlan, map[string]any, error) {
	owner := "resume projection plan"
	if !input.OpensExistingIdentity {
		return ResumePlan{}, nil, invalid("%s opens_existing_identity is not true", owner)
	}
	if input.AllowBlankFallback {
		return ResumePlan{}, nil, invalid("%s allow_blank_fallback is not false", owner)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return ResumePlan{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return ResumePlan{BoundedContinuationTurnRequired: input.BoundedContinuationTurnRequired}, map[string]any{
		"opens_existing_identity":            true,
		"allow_blank_fallback":               false,
		"bounded_continuation_turn_required": input.BoundedContinuationTurnRequired,
		"extensions":                         extensions,
	}, nil
}

// decodeResumePlan validates one closed ResumeProjectionPlan.
func decodeResumePlan(raw json.RawMessage) (ResumePlan, error) {
	owner := "resume projection plan"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return ResumePlan{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, resumePlanMembers); unknown {
		return ResumePlan{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, resumePlanRequired); missing {
		return ResumePlan{}, invalid("%s misses a required member %q", owner, name)
	}
	opens, ok := rawBool(members["opens_existing_identity"])
	if !ok || !opens {
		return ResumePlan{}, invalid("%s opens_existing_identity is not true", owner)
	}
	blank, ok := rawBool(members["allow_blank_fallback"])
	if !ok || blank {
		return ResumePlan{}, invalid("%s allow_blank_fallback is not false", owner)
	}
	bounded, ok := rawBool(members["bounded_continuation_turn_required"])
	if !ok {
		return ResumePlan{}, invalid("%s bounded_continuation_turn_required is not a boolean", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ResumePlan{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	return ResumePlan{BoundedContinuationTurnRequired: bounded}, nil
}

// buildRollbackPlan validates one caller-supplied rollback plan and
// renders its closed object: required=true,
// retain_through=live_validated,
// forbidden_after_provider_commit=true.
func buildRollbackPlan(input RollbackPlanInput) (RollbackPlan, map[string]any, error) {
	owner := "rollback plan"
	if !input.Required {
		return RollbackPlan{}, nil, invalid("%s required is not true", owner)
	}
	if !validText(input.RetainThrough) {
		return RollbackPlan{}, nil, invalid("%s retain_through is not valid UTF-8", owner)
	}
	if input.RetainThrough != "live_validated" {
		return RollbackPlan{}, nil, invalid("%s retain_through is not live_validated", owner)
	}
	if !input.ForbiddenAfterProviderCommit {
		return RollbackPlan{}, nil, invalid("%s forbidden_after_provider_commit is not true", owner)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return RollbackPlan{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return RollbackPlan{}, map[string]any{
		"required":                        true,
		"retain_through":                  "live_validated",
		"forbidden_after_provider_commit": true,
		"extensions":                      extensions,
	}, nil
}

// decodeRollbackPlan validates one closed RollbackPlan.
func decodeRollbackPlan(raw json.RawMessage) (RollbackPlan, error) {
	owner := "rollback plan"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return RollbackPlan{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, rollbackPlanMembers); unknown {
		return RollbackPlan{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, rollbackPlanRequired); missing {
		return RollbackPlan{}, invalid("%s misses a required member %q", owner, name)
	}
	required, ok := rawBool(members["required"])
	if !ok || !required {
		return RollbackPlan{}, invalid("%s required is not true", owner)
	}
	retain, ok := rawString(members["retain_through"])
	if !ok || retain != "live_validated" {
		return RollbackPlan{}, invalid("%s retain_through is not live_validated", owner)
	}
	forbidden, ok := rawBool(members["forbidden_after_provider_commit"])
	if !ok || !forbidden {
		return RollbackPlan{}, invalid("%s forbidden_after_provider_commit is not true", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return RollbackPlan{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	return RollbackPlan{}, nil
}

// buildContractRequirement validates one caller-supplied contract
// requirement: contract_id string[1..256] and a SemVer version. The
// SemVer grammar delegates to the environ owner.
func buildContractRequirement(input ContractRequirementInput, index int) (ContractRequirement, map[string]any, error) {
	owner := fmt.Sprintf("contract requirement[%d]", index)
	if !validText(input.ContractID) {
		return ContractRequirement{}, nil, invalid("%s contract_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ContractID); length < 1 || length > 256 {
		return ContractRequirement{}, nil, invalid("%s contract_id is not a string[1..256]", owner)
	}
	if !validText(input.Version) {
		return ContractRequirement{}, nil, invalid("%s version is not valid UTF-8", owner)
	}
	if !environ.CheckSemver(input.Version) {
		return ContractRequirement{}, nil, invalid("%s version is not a SemVer", owner)
	}
	return ContractRequirement{ContractID: input.ContractID, Version: input.Version}, map[string]any{
		"contract_id": input.ContractID,
		"version":     input.Version,
	}, nil
}

// decodeContractRequirement validates one closed
// ContractRequirement.
func decodeContractRequirement(raw json.RawMessage, index int) (ContractRequirement, error) {
	owner := "contract requirement"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return ContractRequirement{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, contractRequirementMembers); unknown {
		return ContractRequirement{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, contractRequirementRequired); missing {
		return ContractRequirement{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	id, ok := checkStringBounds(members["contract_id"], 1, 256)
	if !ok {
		return ContractRequirement{}, invalid("%s[%d] contract_id is not a string[1..256]", owner, index)
	}
	version, ok := rawString(members["version"])
	if !ok || !environ.CheckSemver(version) {
		return ContractRequirement{}, invalid("%s[%d] version is not a SemVer", owner, index)
	}
	return ContractRequirement{ContractID: id, Version: version}, nil
}

// checkContractOrder enforces "Sorted unique
// ContractRequirement": contract IDs are strictly increasing. Both
// entries share this gate over the sealed requirement set.
func checkContractOrder(requirements []ContractRequirement) error {
	for index := 1; index < len(requirements); index++ {
		if requirements[index].ContractID <= requirements[index-1].ContractID {
			return invalid("projection plan required_contracts are not sorted unique by contract_id")
		}
	}
	return nil
}

// checkContractCount enforces the required_contracts cardinality
// bound ContractRequirement[1..64].
func checkContractCount(count int) error {
	if count < 1 || count > 64 {
		return invalid("projection plan carries %d required_contracts, want [1..64]", count)
	}
	return nil
}
