package cloneplan_test

import (
	"fmt"
	"strings"
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// stringEdge pins one string bound at edge and edge+1 through one
// build setter and one decode document path.
type stringEdge struct {
	label   string
	minimum int
	maximum int
	build   func(*cloneplan.ProjectionPlanInput, string)
	decode  func(map[string]any, string)
	literal string
}

func planStringEdges() []stringEdge {
	mapping := func(set func(*cloneplan.ItemMappingInput, string)) func(*cloneplan.ProjectionPlanInput, string) {
		return func(input *cloneplan.ProjectionPlanInput, value string) {
			row := input.ItemMappings[0]
			set(&row, value)
			input.ItemMappings = []cloneplan.ItemMappingInput{row}
		}
	}
	mappingDoc := func(member string) func(map[string]any, string) {
		return func(document map[string]any, value string) {
			document["item_mappings"].([]any)[0].(map[string]any)[member] = value
		}
	}
	resource := func(set func(*cloneplan.ExpectedResourceInput, string)) func(*cloneplan.ProjectionPlanInput, string) {
		return func(input *cloneplan.ProjectionPlanInput, value string) {
			row := input.ExpectedResources[0]
			set(&row, value)
			input.ExpectedResources = []cloneplan.ExpectedResourceInput{row}
		}
	}
	resourceDoc := func(member string) func(map[string]any, string) {
		return func(document map[string]any, value string) {
			document["expected_resources"].([]any)[0].(map[string]any)[member] = value
		}
	}
	return []stringEdge{
		{"native-id", 1, 512,
			func(input *cloneplan.ProjectionPlanInput, value string) { input.ExpectedTargetNativeSession = value },
			func(document map[string]any, value string) { document["expected_target_native_session_id"] = value },
			"expected_target_native_session_id"},
		{"rationale", 1, 4096,
			func(input *cloneplan.ProjectionPlanInput, value string) { input.StrategyRationale = value },
			func(document map[string]any, value string) { document["strategy_rationale"] = value },
			"strategy_rationale"},
		{"source-key", 1, 512,
			mapping(func(row *cloneplan.ItemMappingInput, value string) { row.SourceItemKey = value }),
			mappingDoc("source_item_key"), "source_item_key"},
		{"resource-key", 1, 512,
			resource(func(row *cloneplan.ExpectedResourceInput, value string) { row.ResourceKey = value }),
			resourceDoc("resource_key"), "resource_key"},
		{"contract-id", 1, 256,
			func(input *cloneplan.ProjectionPlanInput, value string) {
				contract := input.RequiredContracts[0]
				contract.ContractID = value
				input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
			},
			func(document map[string]any, value string) {
				document["required_contracts"].([]any)[0].(map[string]any)["contract_id"] = value
			}, "contract_id"},
	}
}

// TestStringBoundsEdges pins every plan string bound at edge and
// edge+1 through both entries, measured in characters.
func TestStringBoundsEdges(t *testing.T) {
	for _, edge := range planStringEdges() {
		t.Run(edge.label, func(t *testing.T) {
			admit := []int{edge.minimum, edge.maximum}
			if edge.minimum+1 < edge.maximum {
				admit = append(admit, edge.minimum+1)
			}
			for _, length := range admit {
				value := strings.Repeat("a", length)
				input := validPlanInput()
				edge.build(&input, value)
				sealed := mustBuildPlan(t, input)
				mustDecodePlan(t, sealed)
			}
			refuse := []int{edge.minimum - 1, edge.maximum + 1}
			for _, length := range refuse {
				if length < 0 {
					continue
				}
				value := strings.Repeat("a", length)
				input := validPlanInput()
				edge.build(&input, value)
				_, err := cloneplan.BuildProjectionPlan(input)
				requireRefusal(t, err, edge.literal)
				sealed := mustBuildPlan(t, validPlanInput())
				document := decodeDocument(t, sealed)
				edge.decode(document, value)
				_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
				requireRefusal(t, err, edge.literal)
			}
		})
	}
}

// TestMultibyteMeasure pins the character (not byte) measure: 512
// two-byte characters admit, 513 refuse, on both entries.
func TestMultibyteMeasure(t *testing.T) {
	admit := strings.Repeat("é", 512)
	input := validPlanInput()
	input.ExpectedTargetNativeSession = admit
	sealed := mustBuildPlan(t, input)
	mustDecodePlan(t, sealed)
	refuse := strings.Repeat("é", 513)
	input = validPlanInput()
	input.ExpectedTargetNativeSession = refuse
	_, err := cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "expected_target_native_session_id", "string[1..512]")
	sealed = mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	document["expected_target_native_session_id"] = refuse
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "expected_target_native_session_id", "string[1..512]")
}

// TestManifestStringEdges pins the manifest native ID bound at edge
// and edge+1 through both entries.
func TestManifestStringEdges(t *testing.T) {
	for _, length := range []int{1, 512} {
		manifest := validManifestInput()
		manifest.ExpectedTargetNativeSession = strings.Repeat("m", length)
		sealed := mustBuildManifest(t, manifest)
		mustDecodeManifest(t, sealed)
	}
	for _, length := range []int{0, 513} {
		manifest := validManifestInput()
		manifest.ExpectedTargetNativeSession = strings.Repeat("m", length)
		_, err := cloneplan.BuildProjectedObjectManifest(manifest)
		requireRefusal(t, err, "expected_target_native_session_id")
		sealed := mustBuildManifest(t, validManifestInput())
		document := decodeDocument(t, sealed)
		document["expected_target_native_session_id"] = strings.Repeat("m", length)
		_, err = cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, document))
		requireRefusal(t, err, "expected_target_native_session_id")
	}
}

// TestEntryStringEdges pins the manifest entry key bound at edge
// and edge+1 through both entries.
func TestEntryStringEdges(t *testing.T) {
	for _, length := range []int{1, 512} {
		manifest := validManifestInput()
		entry := validBlobEntryInput(1, strings.Repeat("k", length))
		manifest.Entries = []cloneplan.ProjectedEntryInput{entry}
		sealed := mustBuildManifest(t, manifest)
		mustDecodeManifest(t, sealed)
	}
	for _, length := range []int{0, 513} {
		manifest := validManifestInput()
		entry := validBlobEntryInput(1, "k")
		if length == 0 {
			entry.ResourceKey = ""
		} else {
			entry.ResourceKey = strings.Repeat("k", length)
		}
		manifest.Entries = []cloneplan.ProjectedEntryInput{entry}
		_, err := cloneplan.BuildProjectedObjectManifest(manifest)
		requireRefusal(t, err, "resource_key")
	}
}

// TestSortedUniqueSweep pins that every sorted-unique array refuses
// unsorted and duplicated inputs on both entries.
func TestSortedUniqueSweep(t *testing.T) {
	t.Run("target-resource-keys", func(t *testing.T) {
		for _, keys := range [][]string{{"b", "a"}, {"a", "a"}} {
			input := validPlanInput()
			mapping := validMappingInput("sort")
			mapping.TargetResourceKey = keys
			input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "target_resource_keys", "not sorted unique")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			row := document["item_mappings"].([]any)[0].(map[string]any)
			anyKeys := make([]any, 0, len(keys))
			for _, key := range keys {
				anyKeys = append(anyKeys, key)
			}
			row["target_resource_keys"] = anyKeys
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "target_resource_keys")
		}
	})
	t.Run("mapping-reasons", func(t *testing.T) {
		for _, reasons := range [][]string{
			{"unknown_native_event", "derived_index_rebuilt"},
			{"unknown_native_event", "unknown_native_event"},
		} {
			input := validPlanInput()
			mapping := validMappingInput("sort")
			mapping.ReasonCodes = reasons
			input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "reason_codes", "not sorted unique")
		}
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["item_mappings"].([]any)[0].(map[string]any)["reason_codes"] = []any{"unknown_native_event", "derived_index_rebuilt"}
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "reason_codes")
	})
	t.Run("operation-keys", func(t *testing.T) {
		for _, keys := range [][]string{{"b", "a"}, {"a", "a"}} {
			input := validPlanInput()
			operation := validOperationInput(1)
			operation.ResourceKeys = keys
			input.TargetOperations = []cloneplan.TargetOperationInput{operation}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "resource_keys", "not sorted unique")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			row := document["target_operations"].([]any)[0].(map[string]any)
			anyKeys := make([]any, 0, len(keys))
			for _, key := range keys {
				anyKeys = append(anyKeys, key)
			}
			row["resource_keys"] = anyKeys
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "resource_keys")
		}
	})
	t.Run("depends", func(t *testing.T) {
		input := validPlanInput()
		input.TargetOperations = []cloneplan.TargetOperationInput{
			validOperationInput(1), validOperationInput(2), validOperationInput(3),
		}
		input.TargetOperations[2].DependsOnSequences = []uint64{2, 1}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "depends_on_sequences", "not sorted unique")
		input.TargetOperations[2].DependsOnSequences = []uint64{1, 1}
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "depends_on_sequences", "not sorted unique")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		rows := document["target_operations"].([]any)
		rows[0].(map[string]any)["depends_on_sequences"] = []any{float64(1), float64(1)}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "depends_on_sequences")
	})
	t.Run("forbid", func(t *testing.T) {
		for _, reasons := range [][]string{{"b", "a"}, {"a", "a"}} {
			input := validPlanInput()
			input.ForbidReasons = reasons
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "forbid_reasons", "not sorted unique")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			anyReasons := make([]any, 0, len(reasons))
			for _, reason := range reasons {
				anyReasons = append(anyReasons, reason)
			}
			document["forbid_reasons"] = anyReasons
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "forbid_reasons")
		}
	})
	t.Run("exclusions", func(t *testing.T) {
		for _, classes := range [][]string{
			{"unknown", "credential"},
			{"credential", "credential"},
		} {
			input := validPlanInput()
			input.SecurityExclusions = classes
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "security_exclusions", "not sorted unique")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			anyClasses := make([]any, 0, len(classes))
			for _, class := range classes {
				anyClasses = append(anyClasses, class)
			}
			document["security_exclusions"] = anyClasses
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "security_exclusions")
		}
	})
	t.Run("capabilities", func(t *testing.T) {
		for _, caps := range [][]string{{"b.cap", "a.cap"}, {"a.cap", "a.cap"}} {
			input := validPlanInput()
			input.RequiredCapabilities = caps
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "required_capabilities", "not sorted unique")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			anyCaps := make([]any, 0, len(caps))
			for _, cap := range caps {
				anyCaps = append(anyCaps, cap)
			}
			document["required_capabilities"] = anyCaps
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "required_capabilities")
		}
	})
	t.Run("canonical-ordered-not-sorted", func(t *testing.T) {
		// canonical_event_ids are ORDERED unique (Canonical
		// Session order), not sorted: descending digests admit,
		// duplicates refuse.
		first := fixtureDigest("order-b")
		second := fixtureDigest("order-a")
		ordered := []string{first, second}
		if ordered[0] < ordered[1] {
			ordered[0], ordered[1] = ordered[1], ordered[0]
		}
		input := validPlanInput()
		input.CanonicalEventIDs = ordered
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if plan.CanonicalEventIDs[0].String() != ordered[0] {
			t.Fatal("canonical_event_ids order not preserved")
		}
		input = validPlanInput()
		input.CanonicalEventIDs = []string{first, first}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "canonical_event_ids", "not unique")
		sealed = mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["canonical_event_ids"] = []any{first, first}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "canonical_event_ids", "not unique")
	})
}

// TestRowOrderSweep pins the cross-row order gates: mappings sorted
// by key, resources and entries sorted by (sequence, key),
// contracts sorted by ID.
func TestRowOrderSweep(t *testing.T) {
	t.Run("mappings", func(t *testing.T) {
		for _, keys := range [][]string{{"b", "a"}, {"a", "a"}} {
			input := validPlanInput()
			var mappings []cloneplan.ItemMappingInput
			for _, key := range keys {
				mappings = append(mappings, validMappingInput(key))
			}
			input.ItemMappings = mappings
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "item_mappings", "not sorted unique by source item key")
		}
	})
	t.Run("resources", func(t *testing.T) {
		bad := [][]cloneplan.ExpectedResourceInput{
			{validResourceInput(2, "a"), validResourceInput(1, "b")},
			{validResourceInput(1, "b"), validResourceInput(1, "a")},
			{validResourceInput(1, "a"), validResourceInput(1, "a")},
		}
		for _, resources := range bad {
			input := validPlanInput()
			input.ExpectedResources = resources
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "expected_resources", "not sorted by operation sequence/key")
		}
		good := [][]cloneplan.ExpectedResourceInput{
			{validResourceInput(1, "a"), validResourceInput(1, "b")},
			{validResourceInput(1, "b"), validResourceInput(2, "a")},
		}
		for _, resources := range good {
			input := validPlanInput()
			input.ExpectedResources = resources
			sealed := mustBuildPlan(t, input)
			mustDecodePlan(t, sealed)
		}
	})
	t.Run("entries", func(t *testing.T) {
		bad := [][]cloneplan.ProjectedEntryInput{
			{validBlobEntryInput(2, "a"), validBlobEntryInput(1, "b")},
			{validBlobEntryInput(1, "b"), validBlobEntryInput(1, "a")},
			{validBlobEntryInput(1, "a"), validBlobEntryInput(1, "a")},
		}
		for _, entries := range bad {
			manifest := validManifestInput()
			manifest.Entries = entries
			_, err := cloneplan.BuildProjectedObjectManifest(manifest)
			requireRefusal(t, err, "entries", "not sorted unique by operation sequence/resource key")
		}
	})
	t.Run("contracts", func(t *testing.T) {
		for _, ids := range [][]string{{"b.c", "a.c"}, {"a.c", "a.c"}} {
			input := validPlanInput()
			var contracts []cloneplan.ContractRequirementInput
			for _, id := range ids {
				contracts = append(contracts, validContractInput(id))
			}
			input.RequiredContracts = contracts
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "required_contracts", "not sorted unique by contract_id")
		}
	})
	t.Run("decode-orders", func(t *testing.T) {
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		mappings := document["item_mappings"].([]any)
		first := mappings[0].(map[string]any)
		second := map[string]any{}
		for key, value := range first {
			second[key] = value
		}
		second["source_item_key"] = "a-item"
		document["item_mappings"] = []any{first, second}
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "item_mappings", "not sorted unique by source item key")

		document = decodeDocument(t, sealed)
		resources := document["expected_resources"].([]any)
		resources[0].(map[string]any)["resource_key"] = "z-res"
		extra := map[string]any{}
		for key, value := range resources[0].(map[string]any) {
			extra[key] = value
		}
		extra["resource_key"] = "a-res"
		document["expected_resources"] = []any{resources[0], extra}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "expected_resources", "not sorted by operation sequence/key")

		msealed := mustBuildManifest(t, validManifestInput())
		mdocument := decodeDocument(t, msealed)
		entries := mdocument["entries"].([]any)
		mdocument["entries"] = []any{entries[1], entries[0]}
		_, err = cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, mdocument))
		requireRefusal(t, err, "entries", "not sorted unique by operation sequence/resource key")
	})
}

// TestPlanConstants pins every closed plan constant: any other
// value refuses on both entries.
func TestPlanConstants(t *testing.T) {
	t.Run("transaction", func(t *testing.T) {
		cases := []struct {
			label  string
			mutate func(*cloneplan.TransactionPlanInput)
			doc    func(map[string]any)
			token  string
		}{
			{"intent", func(input *cloneplan.TransactionPlanInput) { input.MaterializationIntent = "prepare" },
				func(nested map[string]any) { nested["materialization_intent"] = "prepare" }, "materialization_intent is not clone"},
			{"intent-near", func(input *cloneplan.TransactionPlanInput) { input.MaterializationIntent = "clone2" },
				func(nested map[string]any) { nested["materialization_intent"] = "clone2" }, "materialization_intent is not clone"},
			{"intent-case", func(input *cloneplan.TransactionPlanInput) { input.MaterializationIntent = "Clone" },
				func(nested map[string]any) { nested["materialization_intent"] = "Clone" }, "materialization_intent is not clone"},
			{"policy", func(input *cloneplan.TransactionPlanInput) { input.TargetCollisionPolicy = "must_be_present" },
				func(nested map[string]any) { nested["target_collision_policy"] = "must_be_present" }, "target_collision_policy is not must_be_absent"},
			{"policy-near", func(input *cloneplan.TransactionPlanInput) { input.TargetCollisionPolicy = "must_be_absent2" },
				func(nested map[string]any) { nested["target_collision_policy"] = "must_be_absent2" }, "target_collision_policy is not must_be_absent"},
			{"activation", func(input *cloneplan.TransactionPlanInput) { input.Activation = "active" },
				func(nested map[string]any) { nested["activation"] = "active" }, "activation is not dormant_validated"},
			{"activation-near", func(input *cloneplan.TransactionPlanInput) { input.Activation = "dormant_validated2" },
				func(nested map[string]any) { nested["activation"] = "dormant_validated2" }, "activation is not dormant_validated"},
		}
		for _, row := range cases {
			input := validPlanInput()
			transaction := input.TransactionPlan
			row.mutate(&transaction)
			input.TransactionPlan = transaction
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "transaction plan", row.token)
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			row.doc(document["transaction_plan"].(map[string]any))
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "transaction plan", row.token)
		}
	})
	t.Run("read-back", func(t *testing.T) {
		for _, modes := range [][]string{
			{}, {"staged"}, {"live"}, {"live", "staged"},
			{"staged", "live", "live"}, {"staged", "staged"}, {"x", "y"},
			{"staged", "live2"},
		} {
			input := validPlanInput()
			readBack := input.ReadBackPlan
			readBack.Modes = modes
			input.ReadBackPlan = readBack
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "read-back plan", "modes are not [staged,live]")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			anyModes := make([]any, 0, len(modes))
			for _, mode := range modes {
				anyModes = append(anyModes, mode)
			}
			document["read_back_plan"].(map[string]any)["modes"] = anyModes
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "read-back plan", "modes are not [staged,live]")
		}
		flags := []struct {
			label  string
			mutate func(*cloneplan.ReadBackPlanInput)
			member string
		}{
			{"identity", func(input *cloneplan.ReadBackPlanInput) { input.RequireIdentityMatch = false }, "require_identity_match"},
			{"workspace", func(input *cloneplan.ReadBackPlanInput) { input.RequireWorkspaceMatch = false }, "require_workspace_match"},
			{"semantic", func(input *cloneplan.ReadBackPlanInput) { input.RequireSemanticMarker = false }, "require_semantic_marker"},
		}
		for _, flag := range flags {
			input := validPlanInput()
			readBack := input.ReadBackPlan
			flag.mutate(&readBack)
			input.ReadBackPlan = readBack
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusal(t, err, "read-back plan", flag.member+" is not true")
			sealed := mustBuildPlan(t, validPlanInput())
			document := decodeDocument(t, sealed)
			document["read_back_plan"].(map[string]any)[flag.member] = false
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "read-back plan", flag.member+" is not true")
		}
	})
	t.Run("resume", func(t *testing.T) {
		input := validPlanInput()
		resume := input.ResumePlan
		resume.OpensExistingIdentity = false
		input.ResumePlan = resume
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "resume projection plan", "opens_existing_identity is not true")
		input = validPlanInput()
		resume = input.ResumePlan
		resume.AllowBlankFallback = true
		input.ResumePlan = resume
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "resume projection plan", "allow_blank_fallback is not false")
		// The bounded flag admits both booleans on both entries.
		for _, bounded := range []bool{true, false} {
			input = validPlanInput()
			resume = input.ResumePlan
			resume.BoundedContinuationTurnRequired = bounded
			input.ResumePlan = resume
			sealed := mustBuildPlan(t, input)
			plan := mustDecodePlan(t, sealed)
			if plan.ResumePlan.BoundedContinuationTurnRequired != bounded {
				t.Fatalf("bounded flag = %v, want %v", plan.ResumePlan.BoundedContinuationTurnRequired, bounded)
			}
		}
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["resume_plan"].(map[string]any)["opens_existing_identity"] = false
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "resume projection plan", "opens_existing_identity is not true")
		document = decodeDocument(t, sealed)
		document["resume_plan"].(map[string]any)["allow_blank_fallback"] = true
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "resume projection plan", "allow_blank_fallback is not false")
		for _, bad := range []any{nil, "true", float64(1), []any{true}, map[string]any{}} {
			document = decodeDocument(t, sealed)
			document["resume_plan"].(map[string]any)["bounded_continuation_turn_required"] = bad
			_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
			requireRefusal(t, err, "resume projection plan", "bounded_continuation_turn_required is not a boolean")
		}
	})
	t.Run("rollback", func(t *testing.T) {
		input := validPlanInput()
		rollback := input.RollbackPlan
		rollback.Required = false
		input.RollbackPlan = rollback
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "rollback plan", "required is not true")
		input = validPlanInput()
		rollback = input.RollbackPlan
		rollback.RetainThrough = "prepared"
		input.RollbackPlan = rollback
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "rollback plan", "retain_through is not live_validated")
		input = validPlanInput()
		rollback = input.RollbackPlan
		rollback.RetainThrough = "live_validated2"
		input.RollbackPlan = rollback
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "rollback plan", "retain_through is not live_validated")
		input = validPlanInput()
		rollback = input.RollbackPlan
		rollback.ForbiddenAfterProviderCommit = false
		input.RollbackPlan = rollback
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "rollback plan", "forbidden_after_provider_commit is not true")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["rollback_plan"].(map[string]any)["required"] = false
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "rollback plan", "required is not true")
		document = decodeDocument(t, sealed)
		document["rollback_plan"].(map[string]any)["retain_through"] = "prepared"
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "rollback plan", "retain_through is not live_validated")
		document = decodeDocument(t, sealed)
		document["rollback_plan"].(map[string]any)["retain_through"] = "live_validated2"
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "rollback plan", "retain_through is not live_validated")
		document = decodeDocument(t, sealed)
		document["rollback_plan"].(map[string]any)["forbidden_after_provider_commit"] = false
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "rollback plan", "forbidden_after_provider_commit is not true")
	})
}

// TestContractSemver pins the SemVer version gate on both entries
// with literal expectations independent of the environ grammar.
func TestContractSemver(t *testing.T) {
	for _, version := range []string{"1.2.3", "0.0.0", "10.20.30", "1.0.0-alpha", "1.0.0+build.1"} {
		input := validPlanInput()
		contract := input.RequiredContracts[0]
		contract.Version = version
		input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
		sealed := mustBuildPlan(t, input)
		mustDecodePlan(t, sealed)
	}
	for _, version := range []string{"", "1.2", "1.2.3.4", "v1.2.3", "1.2.x", "latest", "1.02.3"} {
		input := validPlanInput()
		contract := input.RequiredContracts[0]
		contract.Version = version
		input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "contract requirement[0]", "version is not a SemVer")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["required_contracts"].([]any)[0].(map[string]any)["version"] = version
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "contract requirement[0]", "version is not a SemVer")
	}
}

// TestContractNoExtensions pins that ContractRequirement carries no
// extensions member: the pinned text states exactly contract_id and
// version.
func TestContractNoExtensions(t *testing.T) {
	sealed := mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	document["required_contracts"].([]any)[0].(map[string]any)["extensions"] = map[string]any{}
	_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "contract requirement[0]", "unknown member", "extensions")
}

// TestRequiredDispositions pins the exact-policy map: non-empty,
// values sorted unique 1..7 dispositions, on both entries.
func TestRequiredDispositions(t *testing.T) {
	input := validPlanInput()
	input.RequiredDispositions = map[string][]string{}
	_, err := cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "required_dispositions", "carry no class")
	sealed := mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	document["required_dispositions"] = map[string]any{}
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "required_dispositions", "carry no class")

	bad := []struct {
		label       string
		values      []string
		buildToken  string
		decodeToken string
	}{
		{"empty", []string{}, "[1..7]", "1..7"},
		{"eight", []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable", "exact"}, "[1..7]", "1..7"},
		{"unknown", []string{"bogus"}, "outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", "outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable"},
		{"unsorted", []string{"semantic", "exact"}, "not sorted unique", "not sorted unique"},
		{"duplicate", []string{"exact", "exact"}, "not sorted unique", "not sorted unique"},
	}
	for _, row := range bad {
		input := validPlanInput()
		input.RequiredDispositions = map[string][]string{"durable_payload": row.values}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "required_dispositions", row.buildToken)
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		anyValues := make([]any, 0, len(row.values))
		for _, value := range row.values {
			anyValues = append(anyValues, value)
		}
		document["required_dispositions"] = map[string]any{"durable_payload": anyValues}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "required_dispositions", row.decodeToken)
	}
	// Seven sorted dispositions admit: the full set boundary.
	input = validPlanInput()
	input.RequiredDispositions = map[string][]string{
		"durable_payload": {"exact", "omitted", "opaque_preserved", "semantic", "summarized", "synthesized", "unrecoverable"},
	}
	sealed = mustBuildPlan(t, input)
	mustDecodePlan(t, sealed)
}

// TestSecurityExclusions pins the sorted unique capture-class
// [0..9] gate on both entries.
func TestSecurityExclusions(t *testing.T) {
	all := []string{"credential", "derived_cache_optional", "durable_index_required", "durable_payload", "durable_sidecar", "machine_auth", "runtime_state", "transient_lock", "unknown"}
	input := validPlanInput()
	input.SecurityExclusions = all
	sealed := mustBuildPlan(t, input)
	plan := mustDecodePlan(t, sealed)
	if len(plan.SecurityExclusions) != 9 {
		t.Fatalf("exclusions = %d, want 9", len(plan.SecurityExclusions))
	}
	input = validPlanInput()
	input.SecurityExclusions = append(append([]string{}, all...), "zzz")
	_, err := cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "security_exclusions")
	// Ten entries with a duplicate: the count gate fires before
	// vocabulary and order.
	input = validPlanInput()
	input.SecurityExclusions = append(append([]string{}, all...), "credential")
	_, err = cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "10 security_exclusions", "[0..9]")
	sealed = mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	// Ten sorted unique entries with one vocab-invalid head: the
	// count gate fires before vocabulary and order.
	anyExclusions := make([]any, 0, 10)
	for _, class := range append([]string{"aaa"}, all...) {
		anyExclusions = append(anyExclusions, class)
	}
	document["security_exclusions"] = anyExclusions
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "security_exclusions", "[0..9]")
	input = validPlanInput()
	input.SecurityExclusions = []string{"user_message"}
	_, err = cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "security_exclusions", "outside the nine-class vocabulary")
	sealed = mustBuildPlan(t, validPlanInput())
	document = decodeDocument(t, sealed)
	document["security_exclusions"] = []any{"user_message"}
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "security_exclusions", "outside the nine-class vocabulary")
}

// TestScalarGates pins digest, UUIDv7, tuple, workspace, and limits
// refusals on both entries with member literals.
func TestScalarGates(t *testing.T) {
	sealed := mustBuildPlan(t, validPlanInput())
	cases := []struct {
		label  string
		mutate func(map[string]any)
		token  string
	}{
		{"operation", func(document map[string]any) { document["operation_id"] = "not-a-uuid" }, "operation_id is not a UUIDv7"},
		{"bundle", func(document map[string]any) { document["bundle_id"] = "0198f4c8-8e50-4f66-8f70-555555555555" }, "bundle_id is not a UUIDv7"},
		{"request", func(document map[string]any) { document["request_digest"] = "sha256:zzz" }, "request_digest is not a digest"},
		{"snapshot", func(document map[string]any) { document["source_snapshot_digest"] = "md5:abc" }, "source_snapshot_digest is not a digest"},
		{"capture", func(document map[string]any) { document["capture_manifest_id"] = "" }, "capture_manifest_id is not a digest"},
		{"canonical", func(document map[string]any) { document["canonical_session_id"] = "sha256:" }, "canonical_session_id is not a digest"},
		{"basis", func(document map[string]any) { document["fidelity_basis_digest"] = "x" }, "fidelity_basis_digest is not a digest"},
		{"source-build", func(document map[string]any) { document["source_adapter_build_digest"] = "x" }, "source_adapter_build_digest is not a digest"},
		{"target-build", func(document map[string]any) { document["target_adapter_build_digest"] = "x" }, "target_adapter_build_digest is not a digest"},
		{"controller-build", func(document map[string]any) { document["controller_build_digest"] = "x" }, "controller_build_digest is not a digest"},
		{"source-tuple", func(document map[string]any) { document["source_environment"].(map[string]any)["platform"] = "plan9" }, "source_environment is not an Environment Tuple"},
		{"target-tuple", func(document map[string]any) { delete(document["target_environment"].(map[string]any), "platform") }, "target_environment is not an Environment Tuple"},
		{"workspace", func(document map[string]any) {
			document["target_workspace"].(map[string]any)["cwd_relative"] = "/absolute"
		}, "target_workspace is not a Workspace Binding"},
		{"limits", func(document map[string]any) {
			document["resource_limits"].(map[string]any)["max_objects"] = float64(0)
		}, "resource_limits are not Resource Limits"},
	}
	for _, row := range cases {
		document := decodeDocument(t, sealed)
		row.mutate(document)
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "projection plan", row.token)
	}
	buildCases := []struct {
		label  string
		mutate func(*cloneplan.ProjectionPlanInput)
		token  string
	}{
		{"operation", func(input *cloneplan.ProjectionPlanInput) { input.OperationID = "x" }, "operation_id is not a UUIDv7"},
		{"bundle", func(input *cloneplan.ProjectionPlanInput) { input.BundleID = "x" }, "bundle_id is not a UUIDv7"},
		{"request", func(input *cloneplan.ProjectionPlanInput) { input.RequestDigest = "x" }, "request_digest is not a digest"},
		{"snapshot", func(input *cloneplan.ProjectionPlanInput) { input.SourceSnapshotDigest = "x" }, "source_snapshot_digest is not a digest"},
		{"capture", func(input *cloneplan.ProjectionPlanInput) { input.CaptureManifestID = "x" }, "capture_manifest_id is not a digest"},
		{"canonical", func(input *cloneplan.ProjectionPlanInput) { input.CanonicalSessionID = "x" }, "canonical_session_id is not a digest"},
		{"basis", func(input *cloneplan.ProjectionPlanInput) { input.FidelityBasisDigest = "x" }, "fidelity_basis_digest is not a digest"},
		{"source-tuple", func(input *cloneplan.ProjectionPlanInput) { input.SourceEnvironment = []byte(`{}`) }, "source_environment is not an Environment Tuple"},
		{"target-tuple", func(input *cloneplan.ProjectionPlanInput) { input.TargetEnvironment = []byte(`[]`) }, "target_environment is not an Environment Tuple"},
		{"workspace", func(input *cloneplan.ProjectionPlanInput) { input.TargetWorkspace = []byte(`{}`) }, "target_workspace is not a Workspace Binding"},
		{"limits", func(input *cloneplan.ProjectionPlanInput) { input.ResourceLimits = []byte(`{}`) }, "resource_limits are not Resource Limits"},
	}
	for _, row := range buildCases {
		input := validPlanInput()
		row.mutate(&input)
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "projection plan", row.token)
	}
}

// TestManifestScalarGates pins the manifest scalar refusals on both
// entries.
func TestManifestScalarGates(t *testing.T) {
	sealed := mustBuildManifest(t, validManifestInput())
	cases := []struct {
		label  string
		mutate func(map[string]any)
		token  string
	}{
		{"operation", func(document map[string]any) { document["operation_id"] = "x" }, "operation_id is not a UUIDv7"},
		{"plan", func(document map[string]any) { document["projection_plan_id"] = "x" }, "projection_plan_id is not a digest"},
		{"tuple", func(document map[string]any) { document["target_environment"].(map[string]any)["platform"] = "plan9" }, "target_environment is not an Environment Tuple"},
	}
	for _, row := range cases {
		document := decodeDocument(t, sealed)
		row.mutate(document)
		_, err := cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, document))
		requireRefusal(t, err, "projected object manifest", row.token)
	}
	manifest := validManifestInput()
	manifest.OperationID = "x"
	_, err := cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "operation_id is not a UUIDv7")
	manifest = validManifestInput()
	manifest.ProjectionPlanID = "x"
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projection_plan_id is not a digest")
	manifest = validManifestInput()
	manifest.TargetEnvironment = []byte(`{}`)
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "target_environment is not an Environment Tuple")
}

// TestNumberModel pins the AX number model on every numeric member:
// fractions, exponents, strings, and values at or beyond 2^53
// refuse; the uint53 maximum admits.
func TestNumberModel(t *testing.T) {
	const maxUint53 = uint64(1<<53 - 1)
	bads := []string{`1.5`, `1e3`, `"7"`, `9007199254740992`, `18446744073709551615`, `-1`}
	t.Run("sequence", func(t *testing.T) {
		for _, bad := range bads {
			sealed := mustBuildPlan(t, validPlanInput())
			raw := strings.Replace(string(sealed), `"sequence":1`, `"sequence":`+bad, 1)
			_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
			requireRefusal(t, err, "projection target operation[0]", "sequence is not a uint53>0")
		}
		sealed := mustBuildPlan(t, validPlanInput())
		raw := strings.Replace(string(sealed), `"sequence":1`, `"sequence":0`, 1)
		_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
		requireRefusal(t, err, "projection target operation[0]", "sequence is not a uint53>0")
	})
	t.Run("operation-sequence", func(t *testing.T) {
		for _, bad := range bads {
			sealed := mustBuildPlan(t, validPlanInput())
			raw := strings.Replace(string(sealed), `"operation_sequence":1`, `"operation_sequence":`+bad, 1)
			_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
			requireRefusal(t, err, "expected target resource[0]", "operation_sequence is not a uint53>0")
		}
	})
	t.Run("mode", func(t *testing.T) {
		input := validPlanInput()
		resource := validDirectoryResourceInput(1, "mode-number")
		input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
		sealed := mustBuildPlan(t, input)
		for _, bad := range bads {
			raw := strings.Replace(string(sealed), `"mode":493`, `"mode":`+bad, 1)
			_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
			requireRefusal(t, err, "expected target resource[0]", "mode is not a uint32[0..4095] or null")
		}
		raw := strings.Replace(string(sealed), `"mode":493`, `"mode":4096`, 1)
		_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
		requireRefusal(t, err, "expected target resource[0]", "mode is not a uint32[0..4095] or null")
	})
	t.Run("depends", func(t *testing.T) {
		input := validPlanInput()
		input.TargetOperations = []cloneplan.TargetOperationInput{validOperationInput(1), validOperationInput(2, 1)}
		sealed := mustBuildPlan(t, input)
		raw := strings.Replace(string(sealed), `"depends_on_sequences":[1]`, `"depends_on_sequences":[1.5]`, 1)
		_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
		requireRefusal(t, err, "depends_on_sequences[0]", "not a uint53")
	})
	t.Run("total-bytes", func(t *testing.T) {
		for _, bad := range bads {
			sealed := mustBuildManifest(t, validManifestInput())
			raw := strings.Replace(string(sealed), `"total_bytes":128`, `"total_bytes":`+bad, 1)
			_, err := cloneplan.DecodeProjectedObjectManifest([]byte(raw))
			requireRefusal(t, err, "projected object manifest", "total_bytes is not a uint53")
		}
	})
	t.Run("byte-count", func(t *testing.T) {
		for _, bad := range bads {
			sealed := mustBuildManifest(t, validManifestInput())
			raw := strings.Replace(string(sealed), `"byte_count":128`, `"byte_count":`+bad, 1)
			_, err := cloneplan.DecodeProjectedObjectManifest([]byte(raw))
			requireRefusal(t, err, "projected object entry[1]", "byte_count is not a uint53")
		}
	})
	t.Run("max-admits", func(t *testing.T) {
		manifest := validManifestInput()
		manifest.TotalBytes = maxUint53
		sealed := mustBuildManifest(t, manifest)
		decoded := mustDecodeManifest(t, sealed)
		if decoded.TotalBytes != maxUint53 {
			t.Fatalf("total_bytes = %d, want maxUint53", decoded.TotalBytes)
		}
	})
	t.Run("build-overflow", func(t *testing.T) {
		manifest := validManifestInput()
		manifest.TotalBytes = maxUint53 + 1
		_, err := cloneplan.BuildProjectedObjectManifest(manifest)
		requireRefusal(t, err, "total_bytes exceeds uint53")
		manifest = validManifestInput()
		blob := validBlobEntryInput(1, "overflow")
		blob.ByteCount = u64ptr(maxUint53 + 1)
		manifest.Entries = []cloneplan.ProjectedEntryInput{blob}
		_, err = cloneplan.BuildProjectedObjectManifest(manifest)
		requireRefusal(t, err, "byte_count exceeds uint53")
		input := validPlanInput()
		operation := validOperationInput(0)
		input.TargetOperations = []cloneplan.TargetOperationInput{operation}
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "sequence is not a uint53>0")
		input = validPlanInput()
		resource := validResourceInput(1, "zero")
		resource.OperationSequence = 0
		input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "operation_sequence is not a uint53>0")
		input = validPlanInput()
		resource = validResourceInput(1, "mode-hi")
		resource.Mode = u64ptr(4096)
		input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "mode is not a uint32[0..4095] or null")
		entryManifest := validManifestInput()
		entry := validBlobEntryInput(1, "entry-mode-hi")
		entry.Mode = u64ptr(4096)
		entryManifest.Entries = []cloneplan.ProjectedEntryInput{entry}
		_, err = cloneplan.BuildProjectedObjectManifest(entryManifest)
		requireRefusal(t, err, "mode is not a uint32[0..4095] or null")
	})
}

// TestCountEdges pins every cardinality bound at edge and edge+1
// through both production entries where feasible, plus factored
// gates for the wide bounds.
func TestCountEdges(t *testing.T) {
	t.Run("mappings-entry", func(t *testing.T) {
		input := validPlanInput()
		input.ItemMappings = nil
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "0 item_mappings", "[1..1000000]")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["item_mappings"] = []any{}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "0 item_mappings", "[1..1000000]")
	})
	t.Run("operations-entry", func(t *testing.T) {
		input := validPlanInput()
		input.TargetOperations = nil
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "0 target_operations", "[1..65536]")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["target_operations"] = []any{}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "0 target_operations", "[1..65536]")
	})
	t.Run("contracts", func(t *testing.T) {
		input := validPlanInput()
		input.RequiredContracts = nil
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "0 required_contracts", "[1..64]")
		var many []cloneplan.ContractRequirementInput
		for index := 0; index < 65; index++ {
			many = append(many, validContractInput(fmt.Sprintf("ax.contract.%03d", index)))
		}
		input = validPlanInput()
		input.RequiredContracts = many
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "65 required_contracts", "[1..64]")
		input = validPlanInput()
		input.RequiredContracts = many[:64]
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if len(plan.RequiredContracts) != 64 {
			t.Fatalf("contracts = %d, want 64", len(plan.RequiredContracts))
		}
	})
	t.Run("capabilities", func(t *testing.T) {
		input := validPlanInput()
		input.RequiredCapabilities = nil
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "0 required_capabilities", "[1..64]")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["required_capabilities"] = []any{}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "required_capabilities")
		var many []string
		for index := 0; index < 65; index++ {
			many = append(many, fmt.Sprintf("cap.%03d", index))
		}
		input = validPlanInput()
		input.RequiredCapabilities = many
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "65 required_capabilities", "[1..64]")
		sealed = mustBuildPlan(t, validPlanInput())
		document = decodeDocument(t, sealed)
		anyCaps := make([]any, 0, 65)
		for _, cap := range many {
			anyCaps = append(anyCaps, cap)
		}
		document["required_capabilities"] = anyCaps
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "required_capabilities")
	})
	t.Run("reasons-129", func(t *testing.T) {
		var many []string
		for index := 0; index < 129; index++ {
			many = append(many, fmt.Sprintf("com.example.r%03d", index))
		}
		input := validPlanInput()
		mapping := validMappingInput("reasons")
		mapping.ReasonCodes = many
		input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "129 reason_codes", "[0..128]")
		input = validPlanInput()
		input.ForbidReasons = many
		_, err = cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "129 forbid_reasons", "[0..128]")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		anyReasons := make([]any, 0, len(many))
		for _, reason := range many {
			anyReasons = append(anyReasons, reason)
		}
		document["forbid_reasons"] = anyReasons
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "forbid_reasons")
		document = decodeDocument(t, sealed)
		document["item_mappings"].([]any)[0].(map[string]any)["reason_codes"] = anyReasons
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "reason_codes")
	})
	t.Run("wide-admit-65536", func(t *testing.T) {
		if testing.Short() {
			t.Skip("wide admission needs the full run")
		}
		// canonical_event_ids at exactly 65536 admit on both
		// entries; 65537 refuses.
		var ids []string
		for index := 0; index < 65536; index++ {
			ids = append(ids, fixtureDigest(fmt.Sprintf("wide-ev-%d", index)))
		}
		input := validPlanInput()
		input.CanonicalEventIDs = ids
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if len(plan.CanonicalEventIDs) != 65536 {
			t.Fatalf("canonical_event_ids = %d, want 65536", len(plan.CanonicalEventIDs))
		}
		input = validPlanInput()
		input.CanonicalEventIDs = append(ids, fixtureDigest("wide-ev-extra"))
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "65537 canonical_event_ids", "[0..65536]")
	})
}

// TestMappingCountMillionEntryLevel pins the item_mappings upper
// edge at the production entries: exactly 1,000,000 mappings seal
// and decode, and 1,000,001 refuses with the count literal on both
// entries. Rows are minimal (null canonical id, empty keys/reasons,
// nil extensions) with zero-padded sorted keys. The factored gate
// pins the same edge in TestFactoredCountGates; this test pins that
// both entries reach it.
func TestMappingCountMillionEntryLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("million-row admission needs the full run")
	}
	const million = 1000000
	many := make([]cloneplan.ItemMappingInput, 0, million)
	for index := 0; index < million; index++ {
		many = append(many, cloneplan.ItemMappingInput{
			SourceItemKey:  fmt.Sprintf("m-%07d", index),
			ExpectedDispos: "exact",
		})
	}
	input := validPlanInput()
	input.ItemMappings = many
	sealed := mustBuildPlan(t, input)
	plan := mustDecodePlan(t, sealed)
	if len(plan.ItemMappings) != million {
		t.Fatalf("item_mappings = %d, want %d", len(plan.ItemMappings), million)
	}
	overflow := append(many, cloneplan.ItemMappingInput{
		SourceItemKey:  "m-1000000",
		ExpectedDispos: "exact",
	})
	input = validPlanInput()
	input.ItemMappings = overflow
	_, err := cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "1000001 item_mappings", "[1..1000000]")
	document := decodeDocument(t, sealed)
	rows := document["item_mappings"].([]any)
	rows = append(rows, map[string]any{
		"source_item_key":      "m-1000000",
		"canonical_object_id":  nil,
		"target_resource_keys": []any{},
		"expected_disposition": "exact",
		"reason_codes":         []any{},
		"extensions":           map[string]any{},
	})
	document["item_mappings"] = rows
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "1000001 item_mappings", "[1..1000000]")
}

// TestExtensionsClosed pins the closed extensions rule on both
// entries: reverse-DNS keys, AX values, no nested duplicates.
func TestExtensionsClosed(t *testing.T) {
	input := validPlanInput()
	input.Extensions = map[string]any{"com.example.plan": "ok"}
	mapping := input.ItemMappings[0]
	mapping.Extensions = map[string]any{"com.example.mapping": []any{"a", "b"}}
	input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
	sealed := mustBuildPlan(t, input)
	mustDecodePlan(t, sealed)

	input = validPlanInput()
	input.Extensions = map[string]any{"not-reverse-dns": "x"}
	_, err := cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "projection plan", "extensions invalid")

	sealed = mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	document["extensions"] = map[string]any{"not-reverse-dns": "x"}
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "projection plan", "extensions")

	sealed = mustBuildPlan(t, validPlanInput())
	raw := strings.Replace(string(sealed), `"extensions":{}`, `"extensions":{"a":{},"a":{}}`, 1)
	_, err = cloneplan.DecodeProjectionPlan([]byte(raw))
	if err == nil {
		t.Fatal("nested duplicate extensions admit, want refusal")
	}
	// The first empty extensions object in JCS key order sits in
	// the expected resource row; the refusal names that row.
	requireRefusal(t, err, "expected target resource[0]", "extensions")
}

// TestTotalBytesIsAStatedBoundUntilSpecDefinesDerivation pins the
// explicit bound: total_bytes admits any uint53 because the pinned
// text states no derivation rule. Differing totals over identical
// entries both seal and decode.
func TestTotalBytesIsAStatedBoundUntilSpecDefinesDerivation(t *testing.T) {
	entries := []cloneplan.ProjectedEntryInput{validBlobEntryInput(1, "bound-blob")}
	for _, total := range []uint64{0, 128, 999999} {
		manifest := validManifestInput()
		manifest.Entries = entries
		manifest.TotalBytes = total
		sealed := mustBuildManifest(t, manifest)
		decoded := mustDecodeManifest(t, sealed)
		if decoded.TotalBytes != total {
			t.Fatalf("total_bytes = %d, want %d", decoded.TotalBytes, total)
		}
	}
}

// TestForbidReasonsAdmitLiteralStrings pins that forbid_reasons
// admits literal bounded strings (the table row types them as
// strings, not reason codes).
func TestForbidReasonsAdmitLiteralStrings(t *testing.T) {
	input := validPlanInput()
	input.ForbidReasons = []string{"frobnicate"}
	sealed := mustBuildPlan(t, input)
	plan := mustDecodePlan(t, sealed)
	if len(plan.ForbidReasons) != 1 || plan.ForbidReasons[0] != "frobnicate" {
		t.Fatalf("forbid_reasons = %v, want [frobnicate]", plan.ForbidReasons)
	}
}

// TestNoDispositionCouplingOnMappings pins that the mapping shape
// carries no exact/reasons or synthesized/canonical coupling: the
// pinned mapping text states none, so every combination admits.
func TestNoDispositionCouplingOnMappings(t *testing.T) {
	for _, disposition := range []string{"exact", "synthesized", "omitted"} {
		for _, withReasons := range []bool{false, true} {
			for _, withCanonical := range []bool{false, true} {
				input := validPlanInput()
				mapping := validMappingInput("coupling")
				mapping.ExpectedDispos = disposition
				if withReasons {
					mapping.ReasonCodes = []string{"unknown_native_event"}
				} else {
					mapping.ReasonCodes = []string{}
				}
				if !withCanonical {
					mapping.CanonicalObjectID = nil
				}
				input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
				sealed := mustBuildPlan(t, input)
				mustDecodePlan(t, sealed)
			}
		}
	}
}
