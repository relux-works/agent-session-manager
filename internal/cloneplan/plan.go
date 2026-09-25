package cloneplan

import (
	"encoding/json"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Projection Plan 1.0.0: the
// closed 36-member plan Section 13.14.2 states with its ordered item
// mappings, sequenced target operations, expected resources,
// synthesized events, closed plan constants, and contract
// requirements. Cross-artifact couplings — canonical_event_ids
// equality with Canonical Session order, mapping coverage of
// captured/canonical items, resource partitioning against plan
// operations, and synthesized anchors — need the sibling artifacts
// as inputs and are stated bounds; what this leaf gates is the
// closed shape, the vocabularies, the orderings, the DAG, the
// branches, the constants, and the self identity.

const (
	projectionPlanSchema  = "urn:ax:schema:projection-plan"
	projectionPlanVersion = "1.0.0"
	projectionPlanSelf    = "projection_plan_id"
)

var projectionPlanMembers = map[string]bool{
	"schema": true, "schema_version": true, "projection_plan_id": true,
	"operation_id": true, "bundle_id": true,
	"request_digest": true, "source_snapshot_digest": true,
	"capture_manifest_id": true, "canonical_session_id": true,
	"canonical_event_ids": true,
	"source_environment":  true, "target_environment": true,
	"expected_target_native_session_id": true, "target_workspace": true,
	"strategy": true, "strategy_rationale": true, "fidelity_profile": true,
	"required_dispositions": true, "forbid_reasons": true,
	"item_mappings": true, "target_operations": true,
	"expected_resources": true, "synthesized_events": true,
	"security_exclusions": true, "resource_limits": true,
	"transaction_plan": true, "read_back_plan": true,
	"resume_plan": true, "rollback_plan": true,
	"required_contracts": true, "required_capabilities": true,
	"fidelity_basis_digest":       true,
	"source_adapter_build_digest": true, "target_adapter_build_digest": true,
	"controller_build_digest": true, "extensions": true,
}

var projectionPlanRequired = []string{
	"schema", "schema_version", "projection_plan_id",
	"operation_id", "bundle_id",
	"request_digest", "source_snapshot_digest",
	"capture_manifest_id", "canonical_session_id",
	"canonical_event_ids",
	"source_environment", "target_environment",
	"expected_target_native_session_id", "target_workspace",
	"strategy", "strategy_rationale", "fidelity_profile",
	"required_dispositions", "forbid_reasons",
	"item_mappings", "target_operations",
	"expected_resources", "synthesized_events",
	"security_exclusions", "resource_limits",
	"transaction_plan", "read_back_plan",
	"resume_plan", "rollback_plan",
	"required_contracts", "required_capabilities",
	"fidelity_basis_digest",
	"source_adapter_build_digest", "target_adapter_build_digest",
	"controller_build_digest", "extensions",
}

// ProjectionPlan is one validated projection plan.
type ProjectionPlan struct {
	PlanID                      scalar.Digest
	OperationID                 scalar.UUIDv7
	BundleID                    scalar.UUIDv7
	RequestDigest               scalar.Digest
	SourceSnapshotDigest        scalar.Digest
	CaptureManifestID           scalar.Digest
	CanonicalSessionID          scalar.Digest
	CanonicalEventIDs           []scalar.Digest
	SourceEnvironment           sessadapter.Tuple
	TargetEnvironment           sessadapter.Tuple
	ExpectedTargetNativeSession string
	TargetWorkspace             clonebundle.WorkspaceBinding
	Strategy                    string
	StrategyRationale           string
	FidelityProfile             string
	RequiredDispositions        map[string][]string
	ForbidReasons               []string
	ItemMappings                []ItemMapping
	TargetOperations            []TargetOperation
	ExpectedResources           []ExpectedResource
	SynthesizedEvents           []SynthesizedEvent
	SecurityExclusions          []string
	ResourceLimits              sessadapter.ResourceLimits
	TransactionPlan             TransactionPlan
	ReadBackPlan                ReadBackPlan
	ResumePlan                  ResumePlan
	RollbackPlan                RollbackPlan
	RequiredContracts           []ContractRequirement
	RequiredCapabilities        []string
	FidelityBasisDigest         scalar.Digest
	SourceAdapterBuildDigest    scalar.Digest
	TargetAdapterBuildDigest    scalar.Digest
	ControllerBuildDigest       scalar.Digest
}

// ProjectionPlanInput is the caller-supplied projection plan
// candidate for Build. Tuples, workspace, and resource limits arrive
// as raw JSON and validate through their landed owners; the sealed
// plan carries them back in canonical form.
type ProjectionPlanInput struct {
	OperationID                 string
	BundleID                    string
	RequestDigest               string
	SourceSnapshotDigest        string
	CaptureManifestID           string
	CanonicalSessionID          string
	CanonicalEventIDs           []string
	SourceEnvironment           []byte
	TargetEnvironment           []byte
	ExpectedTargetNativeSession string
	TargetWorkspace             []byte
	Strategy                    string
	StrategyRationale           string
	FidelityProfile             string
	RequiredDispositions        map[string][]string
	ForbidReasons               []string
	ItemMappings                []ItemMappingInput
	TargetOperations            []TargetOperationInput
	ExpectedResources           []ExpectedResourceInput
	SynthesizedEvents           []SynthesizedEventInput
	SecurityExclusions          []string
	ResourceLimits              []byte
	TransactionPlan             TransactionPlanInput
	ReadBackPlan                ReadBackPlanInput
	ResumePlan                  ResumePlanInput
	RollbackPlan                RollbackPlanInput
	RequiredContracts           []ContractRequirementInput
	RequiredCapabilities        []string
	FidelityBasisDigest         string
	SourceAdapterBuildDigest    string
	TargetAdapterBuildDigest    string
	ControllerBuildDigest       string
	Extensions                  map[string]any
}

// BuildProjectionPlan constructs one closed Projection Plan 1.0.0
// as canonical bytes. Identical inputs produce byte-identical
// outputs. Every closed-shape, vocabulary, branch, order, DAG, and
// constant rule is a refusal, never a repair.
func BuildProjectionPlan(input ProjectionPlanInput) ([]byte, error) {
	_, object, err := buildProjectionPlan(input)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	planID := scalar.SHA256Digest(omitted)
	object[projectionPlanSelf] = planID.String()
	return canonicalizeObject(object)
}

func buildProjectionPlan(input ProjectionPlanInput) (ProjectionPlan, map[string]any, error) {
	owner := "projection plan"
	operation, err := scalar.ParseUUIDv7(input.OperationID)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s operation_id is not a UUIDv7: %v", owner, err)
	}
	bundle, err := scalar.ParseUUIDv7(input.BundleID)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s bundle_id is not a UUIDv7: %v", owner, err)
	}
	request, err := scalar.ParseDigest(input.RequestDigest)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s request_digest is not a digest: %v", owner, err)
	}
	snapshot, err := scalar.ParseDigest(input.SourceSnapshotDigest)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s source_snapshot_digest is not a digest: %v", owner, err)
	}
	captureID, err := scalar.ParseDigest(input.CaptureManifestID)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s capture_manifest_id is not a digest: %v", owner, err)
	}
	canonicalID, err := scalar.ParseDigest(input.CanonicalSessionID)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s canonical_session_id is not a digest: %v", owner, err)
	}
	eventIDs, err := parseOrderedUniqueDigestStrings(input.CanonicalEventIDs, 0, 65536, owner, "canonical_event_ids")
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	sourceTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.SourceEnvironment))
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s source_environment is not an Environment Tuple: %v", owner, err)
	}
	targetTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.TargetEnvironment))
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
	}
	if !validText(input.ExpectedTargetNativeSession) {
		return ProjectionPlan{}, nil, invalid("%s expected_target_native_session_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ExpectedTargetNativeSession); length < 1 || length > 512 {
		return ProjectionPlan{}, nil, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	workspace, err := clonebundle.DecodeWorkspaceBinding(json.RawMessage(input.TargetWorkspace))
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s target_workspace is not a Workspace Binding: %v", owner, err)
	}
	var workspaceValue any
	if err := json.Unmarshal(input.TargetWorkspace, &workspaceValue); err != nil {
		return ProjectionPlan{}, nil, invalid("%s target_workspace is not a Workspace Binding: %v", owner, err)
	}
	if !validText(input.Strategy) {
		return ProjectionPlan{}, nil, invalid("%s strategy is not valid UTF-8", owner)
	}
	if !ValidPlanStrategy(input.Strategy) {
		return ProjectionPlan{}, nil, invalid("%s strategy is outside same_environment_native_rewrite|target_native_writer|target_official_import|continuation_context", owner)
	}
	if !validText(input.StrategyRationale) {
		return ProjectionPlan{}, nil, invalid("%s strategy_rationale is not valid UTF-8", owner)
	}
	if length := stringLength(input.StrategyRationale); length < 1 || length > 4096 {
		return ProjectionPlan{}, nil, invalid("%s strategy_rationale is not a string[1..4096]", owner)
	}
	if !validText(input.FidelityProfile) {
		return ProjectionPlan{}, nil, invalid("%s fidelity_profile is not valid UTF-8", owner)
	}
	if !ValidPlanProfile(input.FidelityProfile) {
		return ProjectionPlan{}, nil, invalid("%s fidelity_profile is outside strict_exact|maximal_safe|compact|messages_only", owner)
	}
	required, err := buildRequiredDispositions(input.RequiredDispositions)
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	forbidden, err := checkSortedUniqueBoundedStrings(input.ForbidReasons, 1, 128, 0, 128, owner, "forbid_reasons")
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	if err := checkMappingCount(len(input.ItemMappings)); err != nil {
		return ProjectionPlan{}, nil, err
	}
	mappings := make([]ItemMapping, 0, len(input.ItemMappings))
	mappingObjects := make([]any, 0, len(input.ItemMappings))
	for index, candidate := range input.ItemMappings {
		mapping, object, err := buildItemMapping(candidate, index)
		if err != nil {
			return ProjectionPlan{}, nil, err
		}
		mappings = append(mappings, mapping)
		mappingObjects = append(mappingObjects, object)
	}
	if err := checkMappingOrder(mappings); err != nil {
		return ProjectionPlan{}, nil, err
	}
	if err := checkOperationCount(len(input.TargetOperations)); err != nil {
		return ProjectionPlan{}, nil, err
	}
	operations := make([]TargetOperation, 0, len(input.TargetOperations))
	operationObjects := make([]any, 0, len(input.TargetOperations))
	for index, candidate := range input.TargetOperations {
		operation, object, err := buildTargetOperation(candidate, index)
		if err != nil {
			return ProjectionPlan{}, nil, err
		}
		operations = append(operations, operation)
		operationObjects = append(operationObjects, object)
	}
	if err := checkOperationOrder(operations); err != nil {
		return ProjectionPlan{}, nil, err
	}
	if err := checkOperationDAG(operations); err != nil {
		return ProjectionPlan{}, nil, err
	}
	if err := checkExpectedResourceCount(len(input.ExpectedResources)); err != nil {
		return ProjectionPlan{}, nil, err
	}
	resources := make([]ExpectedResource, 0, len(input.ExpectedResources))
	resourceObjects := make([]any, 0, len(input.ExpectedResources))
	for index, candidate := range input.ExpectedResources {
		resource, object, err := buildExpectedResource(candidate, index)
		if err != nil {
			return ProjectionPlan{}, nil, err
		}
		resources = append(resources, resource)
		resourceObjects = append(resourceObjects, object)
	}
	if err := checkExpectedResourceOrder(resources); err != nil {
		return ProjectionPlan{}, nil, err
	}
	if err := checkSynthesizedEventCount(len(input.SynthesizedEvents)); err != nil {
		return ProjectionPlan{}, nil, err
	}
	events := make([]SynthesizedEvent, 0, len(input.SynthesizedEvents))
	eventObjects := make([]any, 0, len(input.SynthesizedEvents))
	for index, candidate := range input.SynthesizedEvents {
		event, object, err := buildSynthesizedEvent(candidate, index)
		if err != nil {
			return ProjectionPlan{}, nil, err
		}
		events = append(events, event)
		eventObjects = append(eventObjects, object)
	}
	exclusions, err := checkSecurityExclusions(input.SecurityExclusions, owner)
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	limits, err := sessadapter.DecodeResourceLimits(json.RawMessage(input.ResourceLimits))
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s resource_limits are not Resource Limits: %v", owner, err)
	}
	transaction, transactionObject, err := buildTransactionPlan(input.TransactionPlan)
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	readBack, readBackObject, err := buildReadBackPlan(input.ReadBackPlan)
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	resume, resumeObject, err := buildResumePlan(input.ResumePlan)
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	rollback, rollbackObject, err := buildRollbackPlan(input.RollbackPlan)
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	if err := checkContractCount(len(input.RequiredContracts)); err != nil {
		return ProjectionPlan{}, nil, err
	}
	contracts := make([]ContractRequirement, 0, len(input.RequiredContracts))
	contractObjects := make([]any, 0, len(input.RequiredContracts))
	for index, candidate := range input.RequiredContracts {
		contract, object, err := buildContractRequirement(candidate, index)
		if err != nil {
			return ProjectionPlan{}, nil, err
		}
		contracts = append(contracts, contract)
		contractObjects = append(contractObjects, object)
	}
	if err := checkContractOrder(contracts); err != nil {
		return ProjectionPlan{}, nil, err
	}
	capabilities, err := checkSortedUniqueBoundedStrings(input.RequiredCapabilities, 1, 128, 1, 64, owner, "required_capabilities")
	if err != nil {
		return ProjectionPlan{}, nil, err
	}
	basis, err := scalar.ParseDigest(input.FidelityBasisDigest)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s fidelity_basis_digest is not a digest: %v", owner, err)
	}
	sourceBuild, err := scalar.ParseDigest(input.SourceAdapterBuildDigest)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s source_adapter_build_digest is not a digest: %v", owner, err)
	}
	targetBuild, err := scalar.ParseDigest(input.TargetAdapterBuildDigest)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s target_adapter_build_digest is not a digest: %v", owner, err)
	}
	controllerBuild, err := scalar.ParseDigest(input.ControllerBuildDigest)
	if err != nil {
		return ProjectionPlan{}, nil, invalid("%s controller_build_digest is not a digest: %v", owner, err)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return ProjectionPlan{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	plan := ProjectionPlan{
		OperationID:                 operation,
		BundleID:                    bundle,
		RequestDigest:               request,
		SourceSnapshotDigest:        snapshot,
		CaptureManifestID:           captureID,
		CanonicalSessionID:          canonicalID,
		CanonicalEventIDs:           eventIDs,
		SourceEnvironment:           sourceTuple,
		TargetEnvironment:           targetTuple,
		ExpectedTargetNativeSession: input.ExpectedTargetNativeSession,
		TargetWorkspace:             workspace,
		Strategy:                    input.Strategy,
		StrategyRationale:           input.StrategyRationale,
		FidelityProfile:             input.FidelityProfile,
		RequiredDispositions:        required,
		ForbidReasons:               forbidden,
		ItemMappings:                mappings,
		TargetOperations:            operations,
		ExpectedResources:           resources,
		SynthesizedEvents:           events,
		SecurityExclusions:          exclusions,
		ResourceLimits:              limits,
		TransactionPlan:             transaction,
		ReadBackPlan:                readBack,
		ResumePlan:                  resume,
		RollbackPlan:                rollback,
		RequiredContracts:           contracts,
		RequiredCapabilities:        capabilities,
		FidelityBasisDigest:         basis,
		SourceAdapterBuildDigest:    sourceBuild,
		TargetAdapterBuildDigest:    targetBuild,
		ControllerBuildDigest:       controllerBuild,
	}
	object := map[string]any{
		"schema":                            projectionPlanSchema,
		"schema_version":                    projectionPlanVersion,
		"operation_id":                      operation.String(),
		"bundle_id":                         bundle.String(),
		"request_digest":                    request.String(),
		"source_snapshot_digest":            snapshot.String(),
		"capture_manifest_id":               captureID.String(),
		"canonical_session_id":              canonicalID.String(),
		"canonical_event_ids":               digestStrings(eventIDs),
		"source_environment":                tupleObject(sourceTuple),
		"target_environment":                tupleObject(targetTuple),
		"expected_target_native_session_id": input.ExpectedTargetNativeSession,
		"target_workspace":                  workspaceValue,
		"strategy":                          input.Strategy,
		"strategy_rationale":                input.StrategyRationale,
		"fidelity_profile":                  input.FidelityProfile,
		"required_dispositions":             requiredObject(required),
		"forbid_reasons":                    stringValues(forbidden),
		"item_mappings":                     mappingObjects,
		"target_operations":                 operationObjects,
		"expected_resources":                resourceObjects,
		"synthesized_events":                eventObjects,
		"security_exclusions":               stringValues(exclusions),
		"resource_limits":                   limitsObject(limits),
		"transaction_plan":                  transactionObject,
		"read_back_plan":                    readBackObject,
		"resume_plan":                       resumeObject,
		"rollback_plan":                     rollbackObject,
		"required_contracts":                contractObjects,
		"required_capabilities":             stringValues(capabilities),
		"fidelity_basis_digest":             basis.String(),
		"source_adapter_build_digest":       sourceBuild.String(),
		"target_adapter_build_digest":       targetBuild.String(),
		"controller_build_digest":           controllerBuild.String(),
		"extensions":                        extensions,
	}
	return plan, object, nil
}

// buildRequiredDispositions validates the caller-supplied policy
// map: the exact policy from the projection-plan request — a
// non-empty map of classes to sorted unique non-empty disposition
// sets of at most seven. The key vocabulary
// (event-or-artifact-class) has no enumerable list in the pinned
// text, so keys validate as present strings exactly like the
// request owner (sessadapter checkRequiredDispositions); the key
// closure is that owner's stated bound, mirrored here. Keys
// validate in sorted order so multi-key refusals are deterministic.
func buildRequiredDispositions(input map[string][]string) (map[string][]string, error) {
	owner := "projection plan required_dispositions"
	if len(input) == 0 {
		return nil, invalid("%s carry no class", owner)
	}
	keys := make([]string, 0, len(input))
	for class := range input {
		keys = append(keys, class)
	}
	sort.Strings(keys)
	required := make(map[string][]string, len(input))
	for _, class := range keys {
		if !validText(class) {
			return nil, invalid("%s carry a class that is not valid UTF-8", owner)
		}
		values := input[class]
		if len(values) < 1 || len(values) > 7 {
			return nil, invalid("%s[%q] carry %d dispositions, want [1..7]", owner, class, len(values))
		}
		previous := ""
		for index, disposition := range values {
			if !validText(disposition) {
				return nil, invalid("%s[%q][%d] is not valid UTF-8", owner, class, index)
			}
			if !clonefidelity.ValidDisposition(disposition) {
				return nil, invalid("%s[%q][%d] disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, class, index)
			}
			if index > 0 && disposition <= previous {
				return nil, invalid("%s[%q] are not sorted unique", owner, class)
			}
			previous = disposition
		}
		required[class] = values
	}
	return required, nil
}

// checkSecurityExclusions validates the caller-supplied exclusion
// list: sorted unique capture classes in [0..9]. The class
// vocabulary delegates to the clonebundle owner.
func checkSecurityExclusions(values []string, owner string) ([]string, error) {
	if len(values) > 9 {
		return nil, invalid("%s carries %d security_exclusions, want [0..9]", owner, len(values))
	}
	previous := ""
	for index, class := range values {
		if !validText(class) {
			return nil, invalid("%s security_exclusions[%d] is not valid UTF-8", owner, index)
		}
		if !clonebundle.ValidCaptureClass(class) {
			return nil, invalid("%s security_exclusions[%d] %q is outside the nine-class vocabulary", owner, index, class)
		}
		if index > 0 && class <= previous {
			return nil, invalid("%s security_exclusions are not sorted unique", owner)
		}
		previous = class
	}
	return values, nil
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

func requiredObject(required map[string][]string) map[string]any {
	object := make(map[string]any, len(required))
	for class, values := range required {
		object[class] = stringValues(values)
	}
	return object
}

func limitsObject(limits sessadapter.ResourceLimits) map[string]any {
	return map[string]any{
		"max_objects":             limits.MaxObjects,
		"max_total_bytes":         limits.MaxTotalBytes,
		"max_single_object_bytes": limits.MaxSingleObjectBytes,
		"max_events":              limits.MaxEvents,
		"max_target_resources":    limits.MaxTargetResources,
	}
}

// DecodeProjectionPlan validates one closed Projection Plan 1.0.0:
// exact members, schema/version, scalar bounds, closed
// vocabularies, sorted-unique arrays, ordered mappings and
// operations, the dependency DAG, branch-exact resource nullability,
// the closed plan constants, and self-digest agreement.
func DecodeProjectionPlan(data []byte) (ProjectionPlan, error) {
	owner := "projection plan"
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return ProjectionPlan{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, projectionPlanMembers); unknown {
		return ProjectionPlan{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, projectionPlanRequired); missing {
		return ProjectionPlan{}, invalid("%s misses a required member %q", owner, name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != projectionPlanSchema {
		return ProjectionPlan{}, invalid("%s schema is not urn:ax:schema:projection-plan", owner)
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != projectionPlanVersion {
		return ProjectionPlan{}, invalid("%s schema_version is not 1.0.0", owner)
	}
	claimed, ok := checkDigest(members["projection_plan_id"])
	if !ok {
		return ProjectionPlan{}, invalid("%s projection_plan_id is not a digest", owner)
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return ProjectionPlan{}, invalid("%s operation_id is not a UUIDv7", owner)
	}
	bundle, ok := checkUUIDv7(members["bundle_id"])
	if !ok {
		return ProjectionPlan{}, invalid("%s bundle_id is not a UUIDv7", owner)
	}
	request, ok := checkDigest(members["request_digest"])
	if !ok {
		return ProjectionPlan{}, invalid("%s request_digest is not a digest", owner)
	}
	snapshot, ok := checkDigest(members["source_snapshot_digest"])
	if !ok {
		return ProjectionPlan{}, invalid("%s source_snapshot_digest is not a digest", owner)
	}
	captureID, ok := checkDigest(members["capture_manifest_id"])
	if !ok {
		return ProjectionPlan{}, invalid("%s capture_manifest_id is not a digest", owner)
	}
	canonicalID, ok := checkDigest(members["canonical_session_id"])
	if !ok {
		return ProjectionPlan{}, invalid("%s canonical_session_id is not a digest", owner)
	}
	eventIDs, err := decodeOrderedUniqueDigests(members["canonical_event_ids"], 0, 65536, owner, "canonical_event_ids")
	if err != nil {
		return ProjectionPlan{}, err
	}
	sourceTuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil {
		return ProjectionPlan{}, invalid("%s source_environment is not an Environment Tuple: %v", owner, err)
	}
	targetTuple, err := sessadapter.DecodeTuple(members["target_environment"])
	if err != nil {
		return ProjectionPlan{}, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
	}
	nativeID, ok := checkStringBounds(members["expected_target_native_session_id"], 1, 512)
	if !ok {
		return ProjectionPlan{}, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	workspace, err := clonebundle.DecodeWorkspaceBinding(members["target_workspace"])
	if err != nil {
		return ProjectionPlan{}, invalid("%s target_workspace is not a Workspace Binding: %v", owner, err)
	}
	strategy, ok := rawString(members["strategy"])
	if !ok || !ValidPlanStrategy(strategy) {
		return ProjectionPlan{}, invalid("%s strategy is outside same_environment_native_rewrite|target_native_writer|target_official_import|continuation_context", owner)
	}
	rationale, ok := checkStringBounds(members["strategy_rationale"], 1, 4096)
	if !ok {
		return ProjectionPlan{}, invalid("%s strategy_rationale is not a string[1..4096]", owner)
	}
	profile, ok := rawString(members["fidelity_profile"])
	if !ok || !ValidPlanProfile(profile) {
		return ProjectionPlan{}, invalid("%s fidelity_profile is outside strict_exact|maximal_safe|compact|messages_only", owner)
	}
	required, err := decodeRequiredDispositions(members["required_dispositions"])
	if err != nil {
		return ProjectionPlan{}, err
	}
	forbidden, ok := checkSortedUniqueStrings(members["forbid_reasons"], 1, 128, 0, 128)
	if !ok {
		return ProjectionPlan{}, invalid("%s forbid_reasons are not sorted unique string[1..128][0..128]", owner)
	}
	mappingElements, ok := decodeArray(members["item_mappings"])
	if !ok {
		return ProjectionPlan{}, invalid("%s item_mappings is not an array", owner)
	}
	if err := checkMappingCount(len(mappingElements)); err != nil {
		return ProjectionPlan{}, err
	}
	mappings := make([]ItemMapping, 0, len(mappingElements))
	for index, element := range mappingElements {
		mapping, err := decodeItemMapping(element, index)
		if err != nil {
			return ProjectionPlan{}, err
		}
		mappings = append(mappings, mapping)
	}
	if err := checkMappingOrder(mappings); err != nil {
		return ProjectionPlan{}, err
	}
	operationElements, ok := decodeArray(members["target_operations"])
	if !ok {
		return ProjectionPlan{}, invalid("%s target_operations is not an array", owner)
	}
	if err := checkOperationCount(len(operationElements)); err != nil {
		return ProjectionPlan{}, err
	}
	operations := make([]TargetOperation, 0, len(operationElements))
	for index, element := range operationElements {
		operation, err := decodeTargetOperation(element, index)
		if err != nil {
			return ProjectionPlan{}, err
		}
		operations = append(operations, operation)
	}
	if err := checkOperationOrder(operations); err != nil {
		return ProjectionPlan{}, err
	}
	if err := checkOperationDAG(operations); err != nil {
		return ProjectionPlan{}, err
	}
	resourceElements, ok := decodeArray(members["expected_resources"])
	if !ok {
		return ProjectionPlan{}, invalid("%s expected_resources is not an array", owner)
	}
	if err := checkExpectedResourceCount(len(resourceElements)); err != nil {
		return ProjectionPlan{}, err
	}
	resources := make([]ExpectedResource, 0, len(resourceElements))
	for index, element := range resourceElements {
		resource, err := decodeExpectedResource(element, index)
		if err != nil {
			return ProjectionPlan{}, err
		}
		resources = append(resources, resource)
	}
	if err := checkExpectedResourceOrder(resources); err != nil {
		return ProjectionPlan{}, err
	}
	eventElements, ok := decodeArray(members["synthesized_events"])
	if !ok {
		return ProjectionPlan{}, invalid("%s synthesized_events is not an array", owner)
	}
	if err := checkSynthesizedEventCount(len(eventElements)); err != nil {
		return ProjectionPlan{}, err
	}
	events := make([]SynthesizedEvent, 0, len(eventElements))
	for index, element := range eventElements {
		event, err := decodeSynthesizedEvent(element, index)
		if err != nil {
			return ProjectionPlan{}, err
		}
		events = append(events, event)
	}
	exclusionValues, ok := checkSortedUniqueStrings(members["security_exclusions"], 1, 128, 0, 9)
	if !ok {
		return ProjectionPlan{}, invalid("%s security_exclusions are not sorted unique string[1..128][0..9]", owner)
	}
	for _, class := range exclusionValues {
		if !clonebundle.ValidCaptureClass(class) {
			return ProjectionPlan{}, invalid("%s security_exclusions carry %q outside the nine-class vocabulary", owner, class)
		}
	}
	limits, err := sessadapter.DecodeResourceLimits(members["resource_limits"])
	if err != nil {
		return ProjectionPlan{}, invalid("%s resource_limits are not Resource Limits: %v", owner, err)
	}
	transaction, err := decodeTransactionPlan(members["transaction_plan"])
	if err != nil {
		return ProjectionPlan{}, err
	}
	readBack, err := decodeReadBackPlan(members["read_back_plan"])
	if err != nil {
		return ProjectionPlan{}, err
	}
	resume, err := decodeResumePlan(members["resume_plan"])
	if err != nil {
		return ProjectionPlan{}, err
	}
	rollback, err := decodeRollbackPlan(members["rollback_plan"])
	if err != nil {
		return ProjectionPlan{}, err
	}
	contractElements, ok := decodeArray(members["required_contracts"])
	if !ok {
		return ProjectionPlan{}, invalid("%s required_contracts is not an array", owner)
	}
	if err := checkContractCount(len(contractElements)); err != nil {
		return ProjectionPlan{}, err
	}
	contracts := make([]ContractRequirement, 0, len(contractElements))
	for index, element := range contractElements {
		contract, err := decodeContractRequirement(element, index)
		if err != nil {
			return ProjectionPlan{}, err
		}
		contracts = append(contracts, contract)
	}
	if err := checkContractOrder(contracts); err != nil {
		return ProjectionPlan{}, err
	}
	capabilities, ok := checkSortedUniqueStrings(members["required_capabilities"], 1, 128, 1, 64)
	if !ok {
		return ProjectionPlan{}, invalid("%s required_capabilities are not sorted unique string[1..128][1..64]", owner)
	}
	basis, ok := checkDigest(members["fidelity_basis_digest"])
	if !ok {
		return ProjectionPlan{}, invalid("%s fidelity_basis_digest is not a digest", owner)
	}
	sourceBuild, ok := checkDigest(members["source_adapter_build_digest"])
	if !ok {
		return ProjectionPlan{}, invalid("%s source_adapter_build_digest is not a digest", owner)
	}
	targetBuild, ok := checkDigest(members["target_adapter_build_digest"])
	if !ok {
		return ProjectionPlan{}, invalid("%s target_adapter_build_digest is not a digest", owner)
	}
	controllerBuild, ok := checkDigest(members["controller_build_digest"])
	if !ok {
		return ProjectionPlan{}, invalid("%s controller_build_digest is not a digest", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ProjectionPlan{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	if err := verifySelfDigest(members, projectionPlanSelf, claimed); err != nil {
		return ProjectionPlan{}, err
	}
	return ProjectionPlan{
		PlanID:                      claimed,
		OperationID:                 operation,
		BundleID:                    bundle,
		RequestDigest:               request,
		SourceSnapshotDigest:        snapshot,
		CaptureManifestID:           captureID,
		CanonicalSessionID:          canonicalID,
		CanonicalEventIDs:           eventIDs,
		SourceEnvironment:           sourceTuple,
		TargetEnvironment:           targetTuple,
		ExpectedTargetNativeSession: nativeID,
		TargetWorkspace:             workspace,
		Strategy:                    strategy,
		StrategyRationale:           rationale,
		FidelityProfile:             profile,
		RequiredDispositions:        required,
		ForbidReasons:               forbidden,
		ItemMappings:                mappings,
		TargetOperations:            operations,
		ExpectedResources:           resources,
		SynthesizedEvents:           events,
		SecurityExclusions:          exclusionValues,
		ResourceLimits:              limits,
		TransactionPlan:             transaction,
		ReadBackPlan:                readBack,
		ResumePlan:                  resume,
		RollbackPlan:                rollback,
		RequiredContracts:           contracts,
		RequiredCapabilities:        capabilities,
		FidelityBasisDigest:         basis,
		SourceAdapterBuildDigest:    sourceBuild,
		TargetAdapterBuildDigest:    targetBuild,
		ControllerBuildDigest:       controllerBuild,
	}, nil
}

// decodeOrderedUniqueDigests validates an ordered unique digest
// array in the count bound: every element a digest, no duplicates,
// order preserved. Unlike the sorted-unique gate,
// canonical_event_ids follow Canonical Session order, so an
// unsorted-but-unique array admits.
func decodeOrderedUniqueDigests(raw json.RawMessage, minimum, maximum int, owner, member string) ([]scalar.Digest, error) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, invalid("%s %s is not an array", owner, member)
	}
	if len(elements) < minimum || len(elements) > maximum {
		return nil, invalid("%s carries %d %s, want [%d..%d]", owner, len(elements), member, minimum, maximum)
	}
	ids := make([]scalar.Digest, 0, len(elements))
	seen := make(map[string]bool, len(elements))
	for index, element := range elements {
		id, ok := checkDigest(element)
		if !ok {
			return nil, invalid("%s %s[%d] is not a digest", owner, member, index)
		}
		if seen[id.String()] {
			return nil, invalid("%s %s are not unique", owner, member)
		}
		seen[id.String()] = true
		ids = append(ids, id)
	}
	return ids, nil
}

// decodeRequiredDispositions validates the closed policy map: a
// non-empty map of classes to sorted unique non-empty disposition
// sets of at most seven. Keys validate as present strings exactly
// like the request owner; the key closure is that owner's stated
// bound, mirrored here.
func decodeRequiredDispositions(raw json.RawMessage) (map[string][]string, error) {
	owner := "projection plan required_dispositions"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return nil, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if len(members) == 0 {
		return nil, invalid("%s carry no class", owner)
	}
	required := make(map[string][]string, len(members))
	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, class := range keys {
		elements, ok := decodeArray(members[class])
		if !ok || len(elements) < 1 || len(elements) > 7 {
			return nil, invalid("%s[%q] carry no 1..7 dispositions, want a non-empty set", owner, class)
		}
		values := make([]string, 0, len(elements))
		previous := ""
		for index, element := range elements {
			disposition, ok := rawString(element)
			if !ok || !clonefidelity.ValidDisposition(disposition) {
				return nil, invalid("%s[%q][%d] disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, class, index)
			}
			if index > 0 && disposition <= previous {
				return nil, invalid("%s[%q] are not sorted unique", owner, class)
			}
			previous = disposition
			values = append(values, disposition)
		}
		required[class] = values
	}
	return required, nil
}
