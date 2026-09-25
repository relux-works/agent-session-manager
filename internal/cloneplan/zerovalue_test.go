package cloneplan_test

import (
	"reflect"
	"strings"
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// censusFieldCount returns the reflected field count of one input
// type: every zero table below asserts its cell count against this,
// so a new input field fails loudly instead of slipping out of the
// build-side member census.
func censusFieldCount(sample any) int {
	return reflect.TypeOf(sample).NumField()
}

// zeroCell is one build-side census cell: zero one input member of
// an otherwise-valid candidate and assert the production verdict.
// Refuse cells pin the member literal; admit cells pin that the
// zero value is inside the bound (empty arrays with min 0, null
// members, either boolean, nil extensions sealing as {}).
type zeroCell struct {
	label string
	zero  func(*cloneplan.ProjectionPlanInput)
	admit bool
	token string
}

func planZeroCells() []zeroCell {
	return []zeroCell{
		{"operation", func(input *cloneplan.ProjectionPlanInput) { input.OperationID = "" }, false, "operation_id is not a UUIDv7"},
		{"bundle", func(input *cloneplan.ProjectionPlanInput) { input.BundleID = "" }, false, "bundle_id is not a UUIDv7"},
		{"request", func(input *cloneplan.ProjectionPlanInput) { input.RequestDigest = "" }, false, "request_digest is not a digest"},
		{"snapshot", func(input *cloneplan.ProjectionPlanInput) { input.SourceSnapshotDigest = "" }, false, "source_snapshot_digest is not a digest"},
		{"capture", func(input *cloneplan.ProjectionPlanInput) { input.CaptureManifestID = "" }, false, "capture_manifest_id is not a digest"},
		{"canonical", func(input *cloneplan.ProjectionPlanInput) { input.CanonicalSessionID = "" }, false, "canonical_session_id is not a digest"},
		{"event-ids", func(input *cloneplan.ProjectionPlanInput) { input.CanonicalEventIDs = nil }, true, ""},
		{"source-tuple", func(input *cloneplan.ProjectionPlanInput) { input.SourceEnvironment = nil }, false, "source_environment is not an Environment Tuple"},
		{"target-tuple", func(input *cloneplan.ProjectionPlanInput) { input.TargetEnvironment = nil }, false, "target_environment is not an Environment Tuple"},
		{"native-id", func(input *cloneplan.ProjectionPlanInput) { input.ExpectedTargetNativeSession = "" }, false, "expected_target_native_session_id is not a string[1..512]"},
		{"workspace", func(input *cloneplan.ProjectionPlanInput) { input.TargetWorkspace = nil }, false, "target_workspace is not a Workspace Binding"},
		{"strategy", func(input *cloneplan.ProjectionPlanInput) { input.Strategy = "" }, false, "strategy is outside same_environment_native_rewrite|target_native_writer|target_official_import|continuation_context"},
		{"rationale", func(input *cloneplan.ProjectionPlanInput) { input.StrategyRationale = "" }, false, "strategy_rationale is not a string[1..4096]"},
		{"profile", func(input *cloneplan.ProjectionPlanInput) { input.FidelityProfile = "" }, false, "fidelity_profile is outside strict_exact|maximal_safe|compact|messages_only"},
		{"required", func(input *cloneplan.ProjectionPlanInput) { input.RequiredDispositions = nil }, false, "required_dispositions carry no class"},
		{"forbid", func(input *cloneplan.ProjectionPlanInput) { input.ForbidReasons = nil }, true, ""},
		{"mappings", func(input *cloneplan.ProjectionPlanInput) { input.ItemMappings = nil }, false, "0 item_mappings"},
		{"operations", func(input *cloneplan.ProjectionPlanInput) { input.TargetOperations = nil }, false, "0 target_operations"},
		{"resources", func(input *cloneplan.ProjectionPlanInput) { input.ExpectedResources = nil }, true, ""},
		{"events", func(input *cloneplan.ProjectionPlanInput) { input.SynthesizedEvents = nil }, true, ""},
		{"exclusions", func(input *cloneplan.ProjectionPlanInput) { input.SecurityExclusions = nil }, true, ""},
		{"limits", func(input *cloneplan.ProjectionPlanInput) { input.ResourceLimits = nil }, false, "resource_limits are not Resource Limits"},
		{"transaction", func(input *cloneplan.ProjectionPlanInput) { input.TransactionPlan = cloneplan.TransactionPlanInput{} }, false, "transaction plan materialization_intent is not clone"},
		{"read-back", func(input *cloneplan.ProjectionPlanInput) { input.ReadBackPlan = cloneplan.ReadBackPlanInput{} }, false, "read-back plan modes are not [staged,live]"},
		{"resume", func(input *cloneplan.ProjectionPlanInput) { input.ResumePlan = cloneplan.ResumePlanInput{} }, false, "resume projection plan opens_existing_identity is not true"},
		{"rollback", func(input *cloneplan.ProjectionPlanInput) { input.RollbackPlan = cloneplan.RollbackPlanInput{} }, false, "rollback plan required is not true"},
		{"contracts", func(input *cloneplan.ProjectionPlanInput) { input.RequiredContracts = nil }, false, "0 required_contracts"},
		{"capabilities", func(input *cloneplan.ProjectionPlanInput) { input.RequiredCapabilities = nil }, false, "0 required_capabilities"},
		{"basis", func(input *cloneplan.ProjectionPlanInput) { input.FidelityBasisDigest = "" }, false, "fidelity_basis_digest is not a digest"},
		{"source-build", func(input *cloneplan.ProjectionPlanInput) { input.SourceAdapterBuildDigest = "" }, false, "source_adapter_build_digest is not a digest"},
		{"target-build", func(input *cloneplan.ProjectionPlanInput) { input.TargetAdapterBuildDigest = "" }, false, "target_adapter_build_digest is not a digest"},
		{"controller-build", func(input *cloneplan.ProjectionPlanInput) { input.ControllerBuildDigest = "" }, false, "controller_build_digest is not a digest"},
		{"extensions", func(input *cloneplan.ProjectionPlanInput) { input.Extensions = nil }, true, ""},
	}
}

// TestBuildZeroValueCensus sweeps every top-level plan input
// member: zero it in an otherwise-valid candidate and assert the
// pinned verdict through the production build entry.
func TestBuildZeroValueCensus(t *testing.T) {
	cells := planZeroCells()
	if want := censusFieldCount(cloneplan.ProjectionPlanInput{}); len(cells) != want {
		t.Fatalf("zero cells = %d, want reflected %d", len(cells), want)
	}
	admitted := 0
	for _, cell := range cells {
		t.Run(cell.label, func(t *testing.T) {
			input := validPlanInput()
			cell.zero(&input)
			sealed, err := cloneplan.BuildProjectionPlan(input)
			if cell.admit {
				admitted++
				if err != nil {
					t.Fatalf("zero %s refused: %v", cell.label, err)
				}
				mustDecodePlan(t, sealed)
				return
			}
			// Nested constant errors name their own owner, not the
			// plan; top-level errors name the plan.
			if strings.Contains(cell.token, " plan ") || strings.HasPrefix(cell.token, "read-back") || strings.HasPrefix(cell.token, "rollback") {
				requireRefusal(t, err, cell.token)
			} else {
				requireRefusal(t, err, "projection plan", cell.token)
			}
		})
	}
	if admitted != 6 {
		t.Fatalf("admitted zero cells = %d, want 6", admitted)
	}
}

// TestBuildZeroValueRows sweeps every row input member through the
// production build entry.
func TestBuildZeroValueRows(t *testing.T) {
	t.Run("mapping", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.ItemMappingInput)
			admit bool
			token string
		}{
			{"key", func(row *cloneplan.ItemMappingInput) { row.SourceItemKey = "" }, false, "source_item_key is not a string[1..512]"},
			{"canonical", func(row *cloneplan.ItemMappingInput) { row.CanonicalObjectID = nil }, true, ""},
			{"keys", func(row *cloneplan.ItemMappingInput) { row.TargetResourceKey = nil }, true, ""},
			{"disposition", func(row *cloneplan.ItemMappingInput) { row.ExpectedDispos = "" }, false, "expected_disposition is outside"},
			{"reasons", func(row *cloneplan.ItemMappingInput) { row.ReasonCodes = nil }, true, ""},
			{"extensions", func(row *cloneplan.ItemMappingInput) { row.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.ItemMappingInput{}); len(cases) != want {
			t.Fatalf("mapping zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			mapping := validMappingInput("zero")
			row.zero(&mapping)
			input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero mapping %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "projection item mapping[0]", row.token)
		}
	})
	t.Run("operation", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.TargetOperationInput)
			admit bool
			token string
		}{
			{"sequence", func(row *cloneplan.TargetOperationInput) { row.Sequence = 0 }, false, "sequence is not a uint53>0"},
			{"action", func(row *cloneplan.TargetOperationInput) { row.Action = "" }, false, "action is outside"},
			{"keys", func(row *cloneplan.TargetOperationInput) { row.ResourceKeys = nil }, false, "0 resource_keys"},
			{"depends", func(row *cloneplan.TargetOperationInput) { row.DependsOnSequences = nil }, true, ""},
			{"extensions", func(row *cloneplan.TargetOperationInput) { row.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.TargetOperationInput{}); len(cases) != want {
			t.Fatalf("operation zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			operation := validOperationInput(1)
			row.zero(&operation)
			input.TargetOperations = []cloneplan.TargetOperationInput{operation}
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero operation %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "projection target operation[0]", row.token)
		}
	})
	t.Run("resource", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.ExpectedResourceInput)
			admit bool
			token string
		}{
			{"sequence", func(row *cloneplan.ExpectedResourceInput) { row.OperationSequence = 0 }, false, "operation_sequence is not a uint53>0"},
			{"key", func(row *cloneplan.ExpectedResourceInput) { row.ResourceKey = "" }, false, "resource_key is not a string[1..512]"},
			{"kind", func(row *cloneplan.ExpectedResourceInput) { row.Kind = "" }, false, "kind is outside blob|directory"},
			{"mode", func(row *cloneplan.ExpectedResourceInput) { row.Mode = nil }, true, ""},
			{"blob", func(row *cloneplan.ExpectedResourceInput) { row.ExpectedBlobID = nil }, false, "expected_blob_id is null for blob"},
			{"extensions", func(row *cloneplan.ExpectedResourceInput) { row.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.ExpectedResourceInput{}); len(cases) != want {
			t.Fatalf("resource zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			resource := validResourceInput(1, "zero")
			row.zero(&resource)
			input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero resource %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "expected target resource[0]", row.token)
		}
	})
	t.Run("event", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.SynthesizedEventInput)
			admit bool
			token string
		}{
			{"id", func(row *cloneplan.SynthesizedEventInput) { row.CanonicalEventID = "" }, false, "canonical_event_id is not a digest"},
			{"anchor", func(row *cloneplan.SynthesizedEventInput) { row.InsertionAfterEvent = nil }, true, ""},
			{"purpose", func(row *cloneplan.SynthesizedEventInput) { row.Purpose = "" }, false, "purpose is outside"},
			{"extensions", func(row *cloneplan.SynthesizedEventInput) { row.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.SynthesizedEventInput{}); len(cases) != want {
			t.Fatalf("event zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			event := validSynthEventInput("zero")
			row.zero(&event)
			input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{event}
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero event %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "synthesized projection event[0]", row.token)
		}
	})
	t.Run("contract", func(t *testing.T) {
		if want := censusFieldCount(cloneplan.ContractRequirementInput{}); want != 2 {
			t.Fatalf("contract fields = %d, want 2: extend the cells below", want)
		}
		input := validPlanInput()
		contract := input.RequiredContracts[0]
		contract.ContractID = ""
		input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "contract requirement[0]", "contract_id is not a string[1..256]")
		input = validPlanInput()
		contract = input.RequiredContracts[0]
		contract.Version = ""
		input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "contract requirement[0]", "version is not a SemVer")
	})
}

// TestBuildZeroValueManifest sweeps every manifest and entry input
// member through the production build entry.
func TestBuildZeroValueManifest(t *testing.T) {
	t.Run("top", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.ProjectedManifestInput)
			admit bool
			token string
		}{
			{"operation", func(input *cloneplan.ProjectedManifestInput) { input.OperationID = "" }, false, "operation_id is not a UUIDv7"},
			{"plan", func(input *cloneplan.ProjectedManifestInput) { input.ProjectionPlanID = "" }, false, "projection_plan_id is not a digest"},
			{"tuple", func(input *cloneplan.ProjectedManifestInput) { input.TargetEnvironment = nil }, false, "target_environment is not an Environment Tuple"},
			{"native-id", func(input *cloneplan.ProjectedManifestInput) { input.ExpectedTargetNativeSession = "" }, false, "expected_target_native_session_id is not a string[1..512]"},
			{"entries", func(input *cloneplan.ProjectedManifestInput) { input.Entries = nil }, true, ""},
			{"total", func(input *cloneplan.ProjectedManifestInput) { input.TotalBytes = 0 }, true, ""},
			{"extensions", func(input *cloneplan.ProjectedManifestInput) { input.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.ProjectedManifestInput{}); len(cases) != want {
			t.Fatalf("manifest zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			manifest := validManifestInput()
			row.zero(&manifest)
			sealed, err := cloneplan.BuildProjectedObjectManifest(manifest)
			if row.admit {
				if err != nil {
					t.Errorf("zero manifest %s refused: %v", row.label, err)
				} else {
					mustDecodeManifest(t, sealed)
				}
				continue
			}
			requireRefusal(t, err, "projected object manifest", row.token)
		}
	})
	t.Run("entry", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.ProjectedEntryInput)
			admit bool
			token string
		}{
			{"sequence", func(row *cloneplan.ProjectedEntryInput) { row.OperationSequence = 0 }, false, "operation_sequence is not a uint53>0"},
			{"key", func(row *cloneplan.ProjectedEntryInput) { row.ResourceKey = "" }, false, "resource_key is not a string[1..512]"},
			{"kind", func(row *cloneplan.ProjectedEntryInput) { row.Kind = "" }, false, "kind is outside blob|directory"},
			{"mode", func(row *cloneplan.ProjectedEntryInput) { row.Mode = nil }, true, ""},
			{"count", func(row *cloneplan.ProjectedEntryInput) { row.ByteCount = nil }, false, "byte_count is missing for blob"},
			{"blob", func(row *cloneplan.ProjectedEntryInput) { row.BlobID = nil }, false, "blob_id is missing for blob"},
			{"descriptor", func(row *cloneplan.ProjectedEntryInput) { row.BlobDescriptorID = nil }, false, "blob_descriptor_id is missing for blob"},
		}
		if want := censusFieldCount(cloneplan.ProjectedEntryInput{}); len(cases) != want {
			t.Fatalf("entry zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			manifest := validManifestInput()
			entry := validBlobEntryInput(1, "zero")
			row.zero(&entry)
			manifest.Entries = []cloneplan.ProjectedEntryInput{entry}
			_, err := cloneplan.BuildProjectedObjectManifest(manifest)
			if row.admit {
				if err != nil {
					t.Errorf("zero entry %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "projected object entry[0]", row.token)
		}
	})
}

// TestBuildZeroValueNestedPlans sweeps every nested plan-component
// input member through the production build entry: the top-level
// census zeroes whole components, this one zeroes each member. Cell
// counts assert against the reflected field counts.
func TestBuildZeroValueNestedPlans(t *testing.T) {
	t.Run("transaction", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.TransactionPlanInput)
			admit bool
			token string
		}{
			{"intent", func(plan *cloneplan.TransactionPlanInput) { plan.MaterializationIntent = "" }, false, "materialization_intent is not clone"},
			{"policy", func(plan *cloneplan.TransactionPlanInput) { plan.TargetCollisionPolicy = "" }, false, "target_collision_policy is not must_be_absent"},
			{"activation", func(plan *cloneplan.TransactionPlanInput) { plan.Activation = "" }, false, "activation is not dormant_validated"},
			{"extensions", func(plan *cloneplan.TransactionPlanInput) { plan.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.TransactionPlanInput{}); len(cases) != want {
			t.Fatalf("transaction zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			plan := validTransactionInput()
			row.zero(&plan)
			input.TransactionPlan = plan
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero transaction %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "transaction plan", row.token)
		}
	})
	t.Run("read-back", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.ReadBackPlanInput)
			admit bool
			token string
		}{
			{"modes", func(plan *cloneplan.ReadBackPlanInput) { plan.Modes = nil }, false, "modes are not [staged,live]"},
			{"identity", func(plan *cloneplan.ReadBackPlanInput) { plan.RequireIdentityMatch = false }, false, "require_identity_match is not true"},
			{"workspace", func(plan *cloneplan.ReadBackPlanInput) { plan.RequireWorkspaceMatch = false }, false, "require_workspace_match is not true"},
			{"semantic", func(plan *cloneplan.ReadBackPlanInput) { plan.RequireSemanticMarker = false }, false, "require_semantic_marker is not true"},
			{"extensions", func(plan *cloneplan.ReadBackPlanInput) { plan.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.ReadBackPlanInput{}); len(cases) != want {
			t.Fatalf("read-back zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			plan := validReadBackInput()
			row.zero(&plan)
			input.ReadBackPlan = plan
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero read-back %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "read-back plan", row.token)
		}
	})
	t.Run("resume", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.ResumePlanInput)
			admit bool
			token string
		}{
			{"opens", func(plan *cloneplan.ResumePlanInput) { plan.OpensExistingIdentity = false }, false, "opens_existing_identity is not true"},
			{"blank", func(plan *cloneplan.ResumePlanInput) { plan.AllowBlankFallback = false }, true, ""},
			{"bounded", func(plan *cloneplan.ResumePlanInput) { plan.BoundedContinuationTurnRequired = false }, true, ""},
			{"extensions", func(plan *cloneplan.ResumePlanInput) { plan.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.ResumePlanInput{}); len(cases) != want {
			t.Fatalf("resume zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			plan := validResumeInput()
			row.zero(&plan)
			input.ResumePlan = plan
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero resume %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "resume projection plan", row.token)
		}
	})
	t.Run("rollback", func(t *testing.T) {
		cases := []struct {
			label string
			zero  func(*cloneplan.RollbackPlanInput)
			admit bool
			token string
		}{
			{"required", func(plan *cloneplan.RollbackPlanInput) { plan.Required = false }, false, "required is not true"},
			{"retain", func(plan *cloneplan.RollbackPlanInput) { plan.RetainThrough = "" }, false, "retain_through is not live_validated"},
			{"forbidden", func(plan *cloneplan.RollbackPlanInput) { plan.ForbiddenAfterProviderCommit = false }, false, "forbidden_after_provider_commit is not true"},
			{"extensions", func(plan *cloneplan.RollbackPlanInput) { plan.Extensions = nil }, true, ""},
		}
		if want := censusFieldCount(cloneplan.RollbackPlanInput{}); len(cases) != want {
			t.Fatalf("rollback zero cells = %d, want reflected %d", len(cases), want)
		}
		for _, row := range cases {
			input := validPlanInput()
			plan := validRollbackInput()
			row.zero(&plan)
			input.RollbackPlan = plan
			_, err := cloneplan.BuildProjectionPlan(input)
			if row.admit {
				if err != nil {
					t.Errorf("zero rollback %s refused: %v", row.label, err)
				}
				continue
			}
			requireRefusal(t, err, "rollback plan", row.token)
		}
	})
}
