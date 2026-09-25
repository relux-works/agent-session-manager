package clonereadback

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Clone Validation Report
// 1.0.0: the closed 22-member report Section 13.14.2 states,
// aggregating both read-back manifests, the seven check booleans,
// the Fidelity Report reference, findings, and the valid result.
// The report carries the two validated reads as trusted siblings:
// the staged and live references must name those reads in their
// own slots, so the same digest in both slots, a swapped pair, or
// a twice-named read refuses. Valid is decided, never carried:
// valid=true if and only if every check boolean is true, the
// native Session IDs match, and no finding carries error severity.
// The report tuple cross-match against the read manifests'
// observed tuples belongs to the final reconciliation leaf.

const (
	validationSchema  = "urn:ax:schema:clone-validation-report"
	validationVersion = "1.0.0"
	validationSelf    = "validation_report_id"
	maxReportFindings = 4096
)

var validationMembers = map[string]bool{
	"schema": true, "schema_version": true,
	"validation_report_id": true, "operation_id": true,
	"projection_plan_id":                    true,
	"projected_object_manifest_id":          true,
	"staged_read_back_evidence_manifest_id": true,
	"live_read_back_evidence_manifest_id":   true,
	"target_provider_manifest_id":           true,
	"fidelity_report_id":                    true,
	"expected_target_native_session_id":     true,
	"observed_target_native_session_id":     true,
	"target_environment":                    true,
	"staged_structural_valid":               true,
	"live_structural_valid":                 true,
	"semantic_marker_valid":                 true,
	"identity_valid":                        true,
	"workspace_binding_valid":               true,
	"resume_surface_valid":                  true,
	"source_generation_revalidated":         true,
	"findings":                              true,
	"valid":                                 true, "extensions": true,
}

var validationRequired = []string{
	"schema", "schema_version",
	"validation_report_id", "operation_id",
	"projection_plan_id",
	"projected_object_manifest_id",
	"staged_read_back_evidence_manifest_id",
	"live_read_back_evidence_manifest_id",
	"target_provider_manifest_id",
	"fidelity_report_id",
	"expected_target_native_session_id",
	"observed_target_native_session_id",
	"target_environment",
	"staged_structural_valid",
	"live_structural_valid",
	"semantic_marker_valid",
	"identity_valid",
	"workspace_binding_valid",
	"resume_surface_valid",
	"source_generation_revalidated",
	"findings",
	"valid", "extensions",
}

// ValidationChecks carries the seven check booleans the valid rule
// conjoins. Every check is always applicable in this schema: unlike
// the adapter validate body, the report has no nullable checks, so
// the oracle enumerates outcomes over all seven with applicability
// fixed.
type ValidationChecks struct {
	StagedStructuralValid       bool
	LiveStructuralValid         bool
	SemanticMarkerValid         bool
	IdentityValid               bool
	WorkspaceBindingValid       bool
	ResumeSurfaceValid          bool
	SourceGenerationRevalidated bool
}

// ValidationReport is one validated Clone Validation Report 1.0.0.
type ValidationReport struct {
	ReportID                    scalar.Digest
	OperationID                 scalar.UUIDv7
	ProjectionPlanID            scalar.Digest
	ProjectedObjectManifestID   scalar.Digest
	StagedManifestID            scalar.Digest
	LiveManifestID              scalar.Digest
	ProviderManifestID          scalar.Digest
	FidelityReportID            scalar.Digest
	ExpectedTargetNativeSession string
	ObservedTargetNativeSession string
	TargetEnvironment           sessadapter.Tuple
	Checks                      ValidationChecks
	Findings                    []sessadapter.Finding
	Valid                       bool
}

// ValidationReportInput is the caller-supplied validation report
// candidate for Build. Nested closed shapes (target tuple,
// findings) arrive as raw JSON and decode through their owners;
// extensions arrive as a Go map and seal through the
// closed-extensions owner.
type ValidationReportInput struct {
	OperationID                      string
	ProjectionPlanID                 string
	ProjectedObjectManifestID        string
	StagedReadBackEvidenceManifestID string
	LiveReadBackEvidenceManifestID   string
	TargetProviderManifestID         string
	FidelityReportID                 string
	ExpectedTargetNativeSession      string
	ObservedTargetNativeSession      string
	TargetEnvironment                []byte
	StagedStructuralValid            bool
	LiveStructuralValid              bool
	SemanticMarkerValid              bool
	IdentityValid                    bool
	WorkspaceBindingValid            bool
	ResumeSurfaceValid               bool
	SourceGenerationRevalidated      bool
	Findings                         []byte
	Valid                            bool
	Extensions                       map[string]any
}

// decideValid computes the pinned valid rule: every check boolean
// true, matching native IDs, and no error-severity finding. Both
// entries share this single decider, so the rule cannot drift
// between Build and Decode.
func decideValid(checks ValidationChecks, expected, observed string, findings []sessadapter.Finding) bool {
	if !checks.StagedStructuralValid {
		return false
	}
	if !checks.LiveStructuralValid {
		return false
	}
	if !checks.SemanticMarkerValid {
		return false
	}
	if !checks.IdentityValid {
		return false
	}
	if !checks.WorkspaceBindingValid {
		return false
	}
	if !checks.ResumeSurfaceValid {
		return false
	}
	if !checks.SourceGenerationRevalidated {
		return false
	}
	if expected != observed {
		return false
	}
	for _, finding := range findings {
		if finding.Severity == "error" {
			return false
		}
	}
	return true
}

// checkValidAgreement enforces that the carried valid bit equals the
// decided rule: a report claiming valid=true with a failed check is
// refused, and a report claiming valid=false with every check
// passing is refused. Both entries share this gate.
func checkValidAgreement(owner string, claimed bool, checks ValidationChecks, expected, observed string, findings []sessadapter.Finding) error {
	if claimed != decideValid(checks, expected, observed, findings) {
		return invalid("%s valid disagrees with the check outcome", owner)
	}
	return nil
}

// checkSiblingsSealed enforces that both aggregated reads are
// sealed: each sibling must come from the read-back validator,
// never from a caller-minted value. The zero value seals nothing
// and refuses here, before any claim comparison reads its
// fields. Both report entries call this gate first.
func checkSiblingsSealed(owner string, staged, live ValidatedReadBack) error {
	if !staged.sealed || !live.sealed {
		return invalid("%s read-back sibling is not a validated read", owner)
	}
	return nil
}

// checkReadRefs enforces that the report aggregates TWO distinct
// sealed reads: the staged sibling is a staged manifest, the live
// sibling is a live manifest, the two reads are distinct
// documents, and the carried references name those reads in their
// own slots. An equal pair, a swapped pair, or a mismatched
// reference refuses. Both report entries share this gate; the
// seal gate above runs first at each entry.
func checkReadRefs(owner string, stagedClaim, liveClaim scalar.Digest, staged, live ValidatedReadBack) error {
	if staged.read.ManifestID == live.read.ManifestID {
		return invalid("%s staged and live read-back references are not distinct", owner)
	}
	if staged.read.Mode != "staged" {
		return invalid("%s staged read is not a staged manifest", owner)
	}
	if live.read.Mode != "live" {
		return invalid("%s live read is not a live manifest", owner)
	}
	if stagedClaim != staged.read.ManifestID {
		return invalid("%s staged_read_back_evidence_manifest_id does not match the staged read", owner)
	}
	if liveClaim != live.read.ManifestID {
		return invalid("%s live_read_back_evidence_manifest_id does not match the live read", owner)
	}
	return nil
}

// BuildValidationReport constructs one closed Clone Validation
// Report 1.0.0 as canonical bytes. Identical inputs produce
// byte-identical outputs. The staged and live siblings are the
// sealed reads the report aggregates: each must come from the
// read-back validator, and the carried references must name them
// in their own slots. Every closed-shape, scalar, finding, and
// validity rule is a refusal, never a repair.
func BuildValidationReport(input ValidationReportInput, staged ValidatedReadBack, live ValidatedReadBack) ([]byte, error) {
	_, object, err := buildValidationReport(input, staged, live)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	reportID := scalar.SHA256Digest(omitted)
	object[validationSelf] = reportID.String()
	return canonicalizeObject(object)
}

func buildValidationReport(input ValidationReportInput, staged ValidatedReadBack, live ValidatedReadBack) (ValidationReport, map[string]any, error) {
	owner := "clone validation report"
	if err := checkSiblingsSealed(owner, staged, live); err != nil {
		return ValidationReport{}, nil, err
	}
	operation, err := scalar.ParseUUIDv7(input.OperationID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s operation_id is not a UUIDv7: %v", owner, err)
	}
	planID, err := scalar.ParseDigest(input.ProjectionPlanID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s projection_plan_id is not a digest: %v", owner, err)
	}
	projectedID, err := scalar.ParseDigest(input.ProjectedObjectManifestID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s projected_object_manifest_id is not a digest: %v", owner, err)
	}
	stagedID, err := scalar.ParseDigest(input.StagedReadBackEvidenceManifestID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s staged_read_back_evidence_manifest_id is not a digest: %v", owner, err)
	}
	liveID, err := scalar.ParseDigest(input.LiveReadBackEvidenceManifestID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s live_read_back_evidence_manifest_id is not a digest: %v", owner, err)
	}
	if err := checkReadRefs(owner, stagedID, liveID, staged, live); err != nil {
		return ValidationReport{}, nil, err
	}
	providerID, err := scalar.ParseDigest(input.TargetProviderManifestID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s target_provider_manifest_id is not a digest: %v", owner, err)
	}
	fidelityID, err := scalar.ParseDigest(input.FidelityReportID)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s fidelity_report_id is not a digest: %v", owner, err)
	}
	if !validText(input.ExpectedTargetNativeSession) {
		return ValidationReport{}, nil, invalid("%s expected_target_native_session_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ExpectedTargetNativeSession); length < 1 || length > 512 {
		return ValidationReport{}, nil, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	if !validText(input.ObservedTargetNativeSession) {
		return ValidationReport{}, nil, invalid("%s observed_target_native_session_id is not valid UTF-8", owner)
	}
	if length := stringLength(input.ObservedTargetNativeSession); length < 1 || length > 512 {
		return ValidationReport{}, nil, invalid("%s observed_target_native_session_id is not a string[1..512]", owner)
	}
	targetTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.TargetEnvironment))
	if err != nil {
		return ValidationReport{}, nil, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
	}
	checks := ValidationChecks{
		StagedStructuralValid:       input.StagedStructuralValid,
		LiveStructuralValid:         input.LiveStructuralValid,
		SemanticMarkerValid:         input.SemanticMarkerValid,
		IdentityValid:               input.IdentityValid,
		WorkspaceBindingValid:       input.WorkspaceBindingValid,
		ResumeSurfaceValid:          input.ResumeSurfaceValid,
		SourceGenerationRevalidated: input.SourceGenerationRevalidated,
	}
	findings, err := sessadapter.DecodeFindings(json.RawMessage(input.Findings), maxReportFindings)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s findings are not AdapterFinding[0..4096]: %v", owner, err)
	}
	if err := checkValidAgreement(owner, input.Valid, checks, input.ExpectedTargetNativeSession, input.ObservedTargetNativeSession, findings); err != nil {
		return ValidationReport{}, nil, err
	}
	extensions, err := clonebundle.EncodeExtensions(input.Extensions)
	if err != nil {
		return ValidationReport{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	var extensionsValue any
	if err := json.Unmarshal(extensions, &extensionsValue); err != nil {
		return ValidationReport{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	report := ValidationReport{
		OperationID:                 operation,
		ProjectionPlanID:            planID,
		ProjectedObjectManifestID:   projectedID,
		StagedManifestID:            stagedID,
		LiveManifestID:              liveID,
		ProviderManifestID:          providerID,
		FidelityReportID:            fidelityID,
		ExpectedTargetNativeSession: input.ExpectedTargetNativeSession,
		ObservedTargetNativeSession: input.ObservedTargetNativeSession,
		TargetEnvironment:           targetTuple,
		Checks:                      checks,
		Findings:                    findings,
		Valid:                       input.Valid,
	}
	object := map[string]any{
		"schema":                                validationSchema,
		"schema_version":                        validationVersion,
		"operation_id":                          operation.String(),
		"projection_plan_id":                    planID.String(),
		"projected_object_manifest_id":          projectedID.String(),
		"staged_read_back_evidence_manifest_id": stagedID.String(),
		"live_read_back_evidence_manifest_id":   liveID.String(),
		"target_provider_manifest_id":           providerID.String(),
		"fidelity_report_id":                    fidelityID.String(),
		"expected_target_native_session_id":     input.ExpectedTargetNativeSession,
		"observed_target_native_session_id":     input.ObservedTargetNativeSession,
		"staged_structural_valid":               input.StagedStructuralValid,
		"live_structural_valid":                 input.LiveStructuralValid,
		"semantic_marker_valid":                 input.SemanticMarkerValid,
		"identity_valid":                        input.IdentityValid,
		"workspace_binding_valid":               input.WorkspaceBindingValid,
		"resume_surface_valid":                  input.ResumeSurfaceValid,
		"source_generation_revalidated":         input.SourceGenerationRevalidated,
		"valid":                                 input.Valid,
		"extensions":                            extensionsValue,
	}
	var tupleValue any
	if err := json.Unmarshal(input.TargetEnvironment, &tupleValue); err != nil {
		return ValidationReport{}, nil, invalid("%s target_environment invalid: %v", owner, err)
	}
	object["target_environment"] = tupleValue
	var findingsValue any
	if err := json.Unmarshal(input.Findings, &findingsValue); err != nil {
		return ValidationReport{}, nil, invalid("%s findings invalid: %v", owner, err)
	}
	object["findings"] = findingsValue
	return report, object, nil
}

// DecodeValidationReport validates one closed Clone Validation
// Report 1.0.0: sealed staged and live siblings, exact members,
// literal schema/version, read references bound to the two
// sibling reads, owner-decoded tuple and findings, the decided
// valid bit, and the omit-self identity.
func DecodeValidationReport(data []byte, staged ValidatedReadBack, live ValidatedReadBack) (ValidationReport, error) {
	owner := "clone validation report"
	if err := checkSiblingsSealed(owner, staged, live); err != nil {
		return ValidationReport{}, err
	}
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return ValidationReport{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, validationMembers); unknown {
		return ValidationReport{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, validationRequired); missing {
		return ValidationReport{}, invalid("%s misses a required member %q", owner, name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != validationSchema {
		return ValidationReport{}, invalid("%s schema is not urn:ax:schema:clone-validation-report", owner)
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != validationVersion {
		return ValidationReport{}, invalid("%s schema_version is not 1.0.0", owner)
	}
	claimed, ok := checkDigest(members[validationSelf])
	if !ok {
		return ValidationReport{}, invalid("%s validation_report_id is not a digest", owner)
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return ValidationReport{}, invalid("%s operation_id is not a UUIDv7", owner)
	}
	planID, ok := checkDigest(members["projection_plan_id"])
	if !ok {
		return ValidationReport{}, invalid("%s projection_plan_id is not a digest", owner)
	}
	projectedID, ok := checkDigest(members["projected_object_manifest_id"])
	if !ok {
		return ValidationReport{}, invalid("%s projected_object_manifest_id is not a digest", owner)
	}
	stagedID, ok := checkDigest(members["staged_read_back_evidence_manifest_id"])
	if !ok {
		return ValidationReport{}, invalid("%s staged_read_back_evidence_manifest_id is not a digest", owner)
	}
	liveID, ok := checkDigest(members["live_read_back_evidence_manifest_id"])
	if !ok {
		return ValidationReport{}, invalid("%s live_read_back_evidence_manifest_id is not a digest", owner)
	}
	if err := checkReadRefs(owner, stagedID, liveID, staged, live); err != nil {
		return ValidationReport{}, err
	}
	providerID, ok := checkDigest(members["target_provider_manifest_id"])
	if !ok {
		return ValidationReport{}, invalid("%s target_provider_manifest_id is not a digest", owner)
	}
	fidelityID, ok := checkDigest(members["fidelity_report_id"])
	if !ok {
		return ValidationReport{}, invalid("%s fidelity_report_id is not a digest", owner)
	}
	expected, ok := checkStringBounds(members["expected_target_native_session_id"], 1, 512)
	if !ok {
		return ValidationReport{}, invalid("%s expected_target_native_session_id is not a string[1..512]", owner)
	}
	observed, ok := checkStringBounds(members["observed_target_native_session_id"], 1, 512)
	if !ok {
		return ValidationReport{}, invalid("%s observed_target_native_session_id is not a string[1..512]", owner)
	}
	targetTuple, err := sessadapter.DecodeTuple(members["target_environment"])
	if err != nil {
		return ValidationReport{}, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
	}
	checks, err := decodeValidationChecks(members)
	if err != nil {
		return ValidationReport{}, err
	}
	findings, err := sessadapter.DecodeFindings(members["findings"], maxReportFindings)
	if err != nil {
		return ValidationReport{}, invalid("%s findings are not AdapterFinding[0..4096]: %v", owner, err)
	}
	valid, ok := rawBool(members["valid"])
	if !ok {
		return ValidationReport{}, invalid("%s valid is not a boolean", owner)
	}
	if err := checkValidAgreement(owner, valid, checks, expected, observed, findings); err != nil {
		return ValidationReport{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return ValidationReport{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	if err := verifySelfDigest(members, validationSelf, claimed); err != nil {
		return ValidationReport{}, err
	}
	return ValidationReport{
		ReportID:                    claimed,
		OperationID:                 operation,
		ProjectionPlanID:            planID,
		ProjectedObjectManifestID:   projectedID,
		StagedManifestID:            stagedID,
		LiveManifestID:              liveID,
		ProviderManifestID:          providerID,
		FidelityReportID:            fidelityID,
		ExpectedTargetNativeSession: expected,
		ObservedTargetNativeSession: observed,
		TargetEnvironment:           targetTuple,
		Checks:                      checks,
		Findings:                    findings,
		Valid:                       valid,
	}, nil
}

// decodeValidationChecks reads the seven check booleans. Each member
// is required non-null; null is a wrong type, never false.
func decodeValidationChecks(members map[string]json.RawMessage) (ValidationChecks, error) {
	owner := "clone validation report"
	read := func(member string) (bool, error) {
		value, ok := rawBool(members[member])
		if !ok {
			return false, invalid("%s %s is not a boolean", owner, member)
		}
		return value, nil
	}
	staged, err := read("staged_structural_valid")
	if err != nil {
		return ValidationChecks{}, err
	}
	live, err := read("live_structural_valid")
	if err != nil {
		return ValidationChecks{}, err
	}
	marker, err := read("semantic_marker_valid")
	if err != nil {
		return ValidationChecks{}, err
	}
	identity, err := read("identity_valid")
	if err != nil {
		return ValidationChecks{}, err
	}
	binding, err := read("workspace_binding_valid")
	if err != nil {
		return ValidationChecks{}, err
	}
	resume, err := read("resume_surface_valid")
	if err != nil {
		return ValidationChecks{}, err
	}
	generation, err := read("source_generation_revalidated")
	if err != nil {
		return ValidationChecks{}, err
	}
	return ValidationChecks{
		StagedStructuralValid:       staged,
		LiveStructuralValid:         live,
		SemanticMarkerValid:         marker,
		IdentityValid:               identity,
		WorkspaceBindingValid:       binding,
		ResumeSurfaceValid:          resume,
		SourceGenerationRevalidated: generation,
	}, nil
}
