package cloneplan_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// Shared fixtures for the cloneplan suite. Every literal is
// hand-written and independent of production constants: verdicts
// always come from production; vectors never do. The oracle lists
// are retyped from the pinned SPEC v0.7.0 text in each oracle test,
// never imported from production or from these builders.

const (
	fixtureOperationID = "0198f4c8-8e50-7f66-8f70-444444444444"
	fixtureBundleID    = "0198f4c8-8e50-7f66-8f70-555555555555"
	fixtureWorkspaceID = "0198f4c8-8e50-7f66-8f70-666666666666"
)

func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fixtureTupleJSON() string {
	return `{"environment_id":"test.env","environment_version":"2.1.0","platform":"linux","architecture":"amd64","store_schema_fingerprint":"` +
		fixtureDigest("plan-store-fingerprint") + `","adapter_version":"1.2.3"}`
}

func fixtureWorkspaceJSON() string {
	return `{"logical_workspace_id":"` + fixtureWorkspaceID + `","cwd_relative":"work/target","repository_remote_fingerprints":[],"branch":null,"head_digest":null,"index_digest":null,"working_tree_digest":null,"extensions":{}}`
}

func fixtureLimitsJSON() string {
	return `{"max_objects":100,"max_total_bytes":1000000,"max_single_object_bytes":65536,"max_events":100,"max_target_resources":100}`
}

func strptr(value string) *string { return &value }

func u64ptr(value uint64) *uint64 { return &value }

// validMappingInput returns a valid item mapping candidate.
func validMappingInput(key string) cloneplan.ItemMappingInput {
	return cloneplan.ItemMappingInput{
		SourceItemKey:     key,
		CanonicalObjectID: strptr(fixtureDigest("canonical-" + key)),
		TargetResourceKey: []string{"res/" + key},
		ExpectedDispos:    "exact",
		ReasonCodes:       []string{},
		Extensions:        map[string]any{},
	}
}

// validOperationInput returns a valid target operation candidate.
func validOperationInput(sequence uint64, depends ...uint64) cloneplan.TargetOperationInput {
	return cloneplan.TargetOperationInput{
		Sequence:           sequence,
		Action:             "write_blob",
		ResourceKeys:       []string{"res/op"},
		DependsOnSequences: depends,
		Extensions:         map[string]any{},
	}
}

// validResourceInput returns a valid expected blob resource.
func validResourceInput(sequence uint64, key string) cloneplan.ExpectedResourceInput {
	return cloneplan.ExpectedResourceInput{
		OperationSequence: sequence,
		ResourceKey:       key,
		Kind:              "blob",
		Mode:              nil,
		ExpectedBlobID:    strptr(fixtureDigest("blob-" + key)),
		Extensions:        map[string]any{},
	}
}

// validDirectoryResourceInput returns a valid expected directory
// resource.
func validDirectoryResourceInput(sequence uint64, key string) cloneplan.ExpectedResourceInput {
	return cloneplan.ExpectedResourceInput{
		OperationSequence: sequence,
		ResourceKey:       key,
		Kind:              "directory",
		Mode:              u64ptr(0755),
		ExpectedBlobID:    nil,
		Extensions:        map[string]any{},
	}
}

// validSynthEventInput returns a valid synthesized event candidate.
func validSynthEventInput(seed string) cloneplan.SynthesizedEventInput {
	return cloneplan.SynthesizedEventInput{
		CanonicalEventID:    fixtureDigest("synth-" + seed),
		InsertionAfterEvent: strptr(fixtureDigest("anchor-" + seed)),
		Purpose:             "summary",
		Extensions:          map[string]any{},
	}
}

func validTransactionInput() cloneplan.TransactionPlanInput {
	return cloneplan.TransactionPlanInput{
		MaterializationIntent: "clone",
		TargetCollisionPolicy: "must_be_absent",
		Activation:            "dormant_validated",
		Extensions:            map[string]any{},
	}
}

func validReadBackInput() cloneplan.ReadBackPlanInput {
	return cloneplan.ReadBackPlanInput{
		Modes:                 []string{"staged", "live"},
		RequireIdentityMatch:  true,
		RequireWorkspaceMatch: true,
		RequireSemanticMarker: true,
		Extensions:            map[string]any{},
	}
}

func validResumeInput() cloneplan.ResumePlanInput {
	return cloneplan.ResumePlanInput{
		OpensExistingIdentity:           true,
		AllowBlankFallback:              false,
		BoundedContinuationTurnRequired: true,
		Extensions:                      map[string]any{},
	}
}

func validRollbackInput() cloneplan.RollbackPlanInput {
	return cloneplan.RollbackPlanInput{
		Required:                     true,
		RetainThrough:                "live_validated",
		ForbiddenAfterProviderCommit: true,
		Extensions:                   map[string]any{},
	}
}

func validContractInput(id string) cloneplan.ContractRequirementInput {
	return cloneplan.ContractRequirementInput{ContractID: id, Version: "1.2.3"}
}

// validPlanInput returns a valid projection plan candidate: one
// mapping, one operation, one blob resource, no synthesized events,
// one contract, one capability.
func validPlanInput() cloneplan.ProjectionPlanInput {
	return cloneplan.ProjectionPlanInput{
		OperationID:                 fixtureOperationID,
		BundleID:                    fixtureBundleID,
		RequestDigest:               fixtureDigest("request"),
		SourceSnapshotDigest:        fixtureDigest("snapshot"),
		CaptureManifestID:           fixtureDigest("capture"),
		CanonicalSessionID:          fixtureDigest("canonical"),
		CanonicalEventIDs:           []string{},
		SourceEnvironment:           []byte(fixtureTupleJSON()),
		TargetEnvironment:           []byte(fixtureTupleJSON()),
		ExpectedTargetNativeSession: "target-native-session",
		TargetWorkspace:             []byte(fixtureWorkspaceJSON()),
		Strategy:                    "target_native_writer",
		StrategyRationale:           "native writer covers the target tuple",
		FidelityProfile:             "maximal_safe",
		RequiredDispositions:        map[string][]string{"durable_payload": {"exact"}},
		ForbidReasons:               []string{},
		ItemMappings:                []cloneplan.ItemMappingInput{validMappingInput("a-item")},
		TargetOperations:            []cloneplan.TargetOperationInput{validOperationInput(1)},
		ExpectedResources:           []cloneplan.ExpectedResourceInput{validResourceInput(1, "a-res")},
		SynthesizedEvents:           []cloneplan.SynthesizedEventInput{},
		SecurityExclusions:          []string{},
		ResourceLimits:              []byte(fixtureLimitsJSON()),
		TransactionPlan:             validTransactionInput(),
		ReadBackPlan:                validReadBackInput(),
		ResumePlan:                  validResumeInput(),
		RollbackPlan:                validRollbackInput(),
		RequiredContracts:           []cloneplan.ContractRequirementInput{validContractInput("ax.contract.plan")},
		RequiredCapabilities:        []string{"clone.target.write"},
		FidelityBasisDigest:         fixtureDigest("basis"),
		SourceAdapterBuildDigest:    fixtureDigest("source-build"),
		TargetAdapterBuildDigest:    fixtureDigest("target-build"),
		ControllerBuildDigest:       fixtureDigest("controller-build"),
		Extensions:                  map[string]any{},
	}
}

// validBlobEntryInput returns a valid blob manifest entry.
func validBlobEntryInput(sequence uint64, key string) cloneplan.ProjectedEntryInput {
	return cloneplan.ProjectedEntryInput{
		OperationSequence: sequence,
		ResourceKey:       key,
		Kind:              "blob",
		Mode:              nil,
		ByteCount:         u64ptr(128),
		BlobID:            strptr(fixtureDigest("blob-" + key)),
		BlobDescriptorID:  strptr(fixtureDigest("descriptor-" + key)),
	}
}

// validDirectoryEntryInput returns a valid directory manifest
// entry.
func validDirectoryEntryInput(sequence uint64, key string) cloneplan.ProjectedEntryInput {
	return cloneplan.ProjectedEntryInput{
		OperationSequence: sequence,
		ResourceKey:       key,
		Kind:              "directory",
		Mode:              u64ptr(0755),
	}
}

// validManifestInput returns a valid projected manifest candidate.
func validManifestInput() cloneplan.ProjectedManifestInput {
	return cloneplan.ProjectedManifestInput{
		OperationID:                 fixtureOperationID,
		ProjectionPlanID:            fixtureDigest("plan-id"),
		TargetEnvironment:           []byte(fixtureTupleJSON()),
		ExpectedTargetNativeSession: "target-native-session",
		Entries: []cloneplan.ProjectedEntryInput{
			validDirectoryEntryInput(1, "a-dir"),
			validBlobEntryInput(1, "b-blob"),
		},
		TotalBytes: 128,
		Extensions: map[string]any{},
	}
}

func mustBuildPlan(t *testing.T, input cloneplan.ProjectionPlanInput) []byte {
	t.Helper()
	sealed, err := cloneplan.BuildProjectionPlan(input)
	if err != nil {
		t.Fatalf("BuildProjectionPlan: %v", err)
	}
	return sealed
}

func mustDecodePlan(t *testing.T, sealed []byte) cloneplan.ProjectionPlan {
	t.Helper()
	plan, err := cloneplan.DecodeProjectionPlan(sealed)
	if err != nil {
		t.Fatalf("DecodeProjectionPlan: %v", err)
	}
	return plan
}

func mustBuildManifest(t *testing.T, input cloneplan.ProjectedManifestInput) []byte {
	t.Helper()
	sealed, err := cloneplan.BuildProjectedObjectManifest(input)
	if err != nil {
		t.Fatalf("BuildProjectedObjectManifest: %v", err)
	}
	return sealed
}

func mustDecodeManifest(t *testing.T, sealed []byte) cloneplan.ProjectedManifest {
	t.Helper()
	manifest, err := cloneplan.DecodeProjectedObjectManifest(sealed)
	if err != nil {
		t.Fatalf("DecodeProjectedObjectManifest: %v", err)
	}
	return manifest
}

// requireRefusal asserts the error is an ErrInvalid refusal whose
// detail carries every literal token.
func requireRefusal(t *testing.T, err error, tokens ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want refusal containing %q", tokens)
	}
	if !errors.Is(err, cloneplan.ErrInvalid) {
		t.Fatalf("error = %v, want cloneplan.ErrInvalid", err)
	}
	for _, token := range tokens {
		if !strings.Contains(err.Error(), token) {
			t.Fatalf("error = %q, want token %q", err.Error(), token)
		}
	}
}

// decodeDocument unmarshals sealed bytes into a generic map for
// member surgery.
func decodeDocument(t *testing.T, sealed []byte) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(sealed, &document); err != nil {
		t.Fatalf("unmarshal sealed: %v", err)
	}
	return document
}

// marshalDocument renders a mutated document back to bytes.
func marshalDocument(t *testing.T, document map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal mutated: %v", err)
	}
	return raw
}
