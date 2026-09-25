package cloneplan_test

import (
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// The oracle lists below are retyped from the pinned SPEC v0.7.0
// Section 13.14.2 text, never imported from production. Every
// vocabulary sweeps admission and refusal through the production
// entries with literal expectations.

// TestPlanStrategyOracle pins the exact four-member plan strategy
// vocabulary: same_environment_native_rewrite|target_native_writer|
// target_official_import|continuation_context. The fifth Section
// 13.14.2 strategy, archive_only, refuses on a plan through both
// entries.
func TestPlanStrategyOracle(t *testing.T) {
	admit := []string{
		"same_environment_native_rewrite",
		"target_native_writer",
		"target_official_import",
		"continuation_context",
	}
	if len(cloneplan.PlanStrategies()) != 4 {
		t.Fatalf("PlanStrategies count = %d, want 4", len(cloneplan.PlanStrategies()))
	}
	for _, strategy := range admit {
		if !cloneplan.ValidPlanStrategy(strategy) {
			t.Errorf("ValidPlanStrategy(%q) = false, want true", strategy)
		}
		input := validPlanInput()
		input.Strategy = strategy
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if plan.Strategy != strategy {
			t.Errorf("strategy = %q, want %q", plan.Strategy, strategy)
		}
	}
	refuse := []string{
		"archive_only",
		"",
		"target_native_writer ",
		"Target_Native_Writer",
		"continuation-context",
		"same_environment_native_rewrite_v2",
		"target_official_importer",
		"native_rewrite",
		"strict_exact",
		"clone",
	}
	for _, strategy := range refuse {
		if cloneplan.ValidPlanStrategy(strategy) {
			t.Errorf("ValidPlanStrategy(%q) = true, want false", strategy)
		}
		input := validPlanInput()
		input.Strategy = strategy
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "projection plan", "strategy", "same_environment_native_rewrite|target_native_writer|target_official_import|continuation_context")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["strategy"] = strategy
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "projection plan", "strategy", "same_environment_native_rewrite|target_native_writer|target_official_import|continuation_context")
	}
}

// TestPlanProfileOracle pins the exact four-member plan profile
// vocabulary: strict_exact|maximal_safe|compact|messages_only. The
// fifth Section 13.14.2 profile, archive_only, refuses on a plan
// through both entries.
func TestPlanProfileOracle(t *testing.T) {
	admit := []string{
		"strict_exact",
		"maximal_safe",
		"compact",
		"messages_only",
	}
	if len(cloneplan.PlanProfiles()) != 4 {
		t.Fatalf("PlanProfiles count = %d, want 4", len(cloneplan.PlanProfiles()))
	}
	for _, profile := range admit {
		if !cloneplan.ValidPlanProfile(profile) {
			t.Errorf("ValidPlanProfile(%q) = false, want true", profile)
		}
		input := validPlanInput()
		input.FidelityProfile = profile
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if plan.FidelityProfile != profile {
			t.Errorf("profile = %q, want %q", plan.FidelityProfile, profile)
		}
	}
	refuse := []string{
		"archive_only",
		"",
		"maximal-safe",
		"Maximal_Safe",
		"compact ",
		"messages-only",
		"strict",
		"exact",
		"target_native_writer",
	}
	for _, profile := range refuse {
		if cloneplan.ValidPlanProfile(profile) {
			t.Errorf("ValidPlanProfile(%q) = true, want false", profile)
		}
		input := validPlanInput()
		input.FidelityProfile = profile
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "projection plan", "fidelity_profile", "strict_exact|maximal_safe|compact|messages_only")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		document["fidelity_profile"] = profile
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "projection plan", "fidelity_profile", "strict_exact|maximal_safe|compact|messages_only")
	}
}

// TestOperationActionOracle pins the exact four-member action
// vocabulary: create_directory|write_blob|write_native_record|
// rebuild_index.
func TestOperationActionOracle(t *testing.T) {
	admit := []string{
		"create_directory",
		"write_blob",
		"write_native_record",
		"rebuild_index",
	}
	if len(cloneplan.OperationActions()) != 4 {
		t.Fatalf("OperationActions count = %d, want 4", len(cloneplan.OperationActions()))
	}
	for _, action := range admit {
		if !cloneplan.ValidOperationAction(action) {
			t.Errorf("ValidOperationAction(%q) = false, want true", action)
		}
		input := validPlanInput()
		input.TargetOperations = []cloneplan.TargetOperationInput{validOperationInput(1)}
		input.TargetOperations[0].Action = action
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if plan.TargetOperations[0].Action != action {
			t.Errorf("action = %q, want %q", plan.TargetOperations[0].Action, action)
		}
	}
	refuse := []string{
		"",
		"create-directory",
		"write_blob ",
		"Write_Blob",
		"write_native_records",
		"rebuild-index",
		"delete_blob",
		"clone",
		"blob",
	}
	for _, action := range refuse {
		if cloneplan.ValidOperationAction(action) {
			t.Errorf("ValidOperationAction(%q) = true, want false", action)
		}
		input := validPlanInput()
		input.TargetOperations = []cloneplan.TargetOperationInput{validOperationInput(1)}
		input.TargetOperations[0].Action = action
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "projection target operation[0]", "action", "create_directory|write_blob|write_native_record|rebuild_index")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		operations := document["target_operations"].([]any)
		operations[0].(map[string]any)["action"] = action
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "projection target operation[0]", "action", "create_directory|write_blob|write_native_record|rebuild_index")
	}
}

// TestSynthesizedPurposeOracle pins the exact three-member purpose
// vocabulary: migration_checkpoint|summary|delimiter.
func TestSynthesizedPurposeOracle(t *testing.T) {
	admit := []string{
		"migration_checkpoint",
		"summary",
		"delimiter",
	}
	if len(cloneplan.SynthesizedPurposes()) != 3 {
		t.Fatalf("SynthesizedPurposes count = %d, want 3", len(cloneplan.SynthesizedPurposes()))
	}
	for _, purpose := range admit {
		if !cloneplan.ValidSynthesizedPurpose(purpose) {
			t.Errorf("ValidSynthesizedPurpose(%q) = false, want true", purpose)
		}
		input := validPlanInput()
		event := validSynthEventInput("purpose")
		event.Purpose = purpose
		input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{event}
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if plan.SynthesizedEvents[0].Purpose != purpose {
			t.Errorf("purpose = %q, want %q", plan.SynthesizedEvents[0].Purpose, purpose)
		}
	}
	refuse := []string{
		"",
		"migration-checkpoint",
		"Summary",
		" summary",
		"delimiters",
		"checkpoint",
		"blob",
		"exact",
	}
	for _, purpose := range refuse {
		if cloneplan.ValidSynthesizedPurpose(purpose) {
			t.Errorf("ValidSynthesizedPurpose(%q) = true, want false", purpose)
		}
		input := validPlanInput()
		event := validSynthEventInput("purpose")
		event.Purpose = purpose
		input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{event}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "synthesized projection event[0]", "purpose", "migration_checkpoint|summary|delimiter")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		events := document["synthesized_events"].([]any)
		_ = events
		document["synthesized_events"] = []any{map[string]any{
			"canonical_event_id":       fixtureDigest("synth-purpose"),
			"insertion_after_event_id": nil,
			"purpose":                  purpose,
			"extensions":               map[string]any{},
		}}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "synthesized projection event[0]", "purpose", "migration_checkpoint|summary|delimiter")
	}
}

// TestResourceKindOracle pins the exact two-member kind vocabulary
// blob|directory on both the expected resource and the manifest
// entry shapes.
func TestResourceKindOracle(t *testing.T) {
	admit := []string{"blob", "directory"}
	if len(cloneplan.ResourceKinds()) != 2 {
		t.Fatalf("ResourceKinds count = %d, want 2", len(cloneplan.ResourceKinds()))
	}
	for _, kind := range admit {
		if !cloneplan.ValidResourceKind(kind) {
			t.Errorf("ValidResourceKind(%q) = false, want true", kind)
		}
	}
	refuse := []string{
		"",
		"Blob",
		" blob",
		"file",
		"dir",
		"symlink",
		"write_blob",
	}
	for _, kind := range refuse {
		if cloneplan.ValidResourceKind(kind) {
			t.Errorf("ValidResourceKind(%q) = true, want false", kind)
		}
		input := validPlanInput()
		resource := validResourceInput(1, "kind-refuse")
		resource.Kind = kind
		input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "expected target resource[0]", "kind", "blob|directory")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		resources := document["expected_resources"].([]any)
		resources[0].(map[string]any)["kind"] = kind
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "expected target resource[0]", "kind", "blob|directory")
		entry := validBlobEntryInput(1, "kind-refuse")
		entry.Kind = kind
		manifest := validManifestInput()
		manifest.Entries = []cloneplan.ProjectedEntryInput{entry}
		_, err = cloneplan.BuildProjectedObjectManifest(manifest)
		requireRefusal(t, err, "projected object entry[0]", "kind", "blob|directory")
		msealed := mustBuildManifest(t, validManifestInput())
		mdocument := decodeDocument(t, msealed)
		entries := mdocument["entries"].([]any)
		entries[0].(map[string]any)["kind"] = kind
		_, err = cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, mdocument))
		requireRefusal(t, err, "projected object entry[0]", "kind", "blob|directory")
	}
}

// TestExpectedDispositionOracle pins that expected_disposition
// admits exactly the seven fidelity dispositions on both entries.
func TestExpectedDispositionOracle(t *testing.T) {
	admit := []string{
		"exact",
		"semantic",
		"summarized",
		"opaque_preserved",
		"synthesized",
		"omitted",
		"unrecoverable",
	}
	for _, disposition := range admit {
		input := validPlanInput()
		mapping := validMappingInput("disposition-" + disposition)
		mapping.ExpectedDispos = disposition
		if disposition != "exact" {
			mapping.ReasonCodes = []string{"unknown_native_event"}
		}
		input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if plan.ItemMappings[0].ExpectedDispos != disposition {
			t.Errorf("expected_disposition = %q, want %q", plan.ItemMappings[0].ExpectedDispos, disposition)
		}
	}
	refuse := []string{
		"",
		"Exact",
		"exact ",
		"partial",
		"synthesized_event",
		"maximal_safe",
		"blob",
	}
	for _, disposition := range refuse {
		input := validPlanInput()
		mapping := validMappingInput("disposition-refuse")
		mapping.ExpectedDispos = disposition
		input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "projection item mapping[0]", "expected_disposition", "exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		mappings := document["item_mappings"].([]any)
		mappings[0].(map[string]any)["expected_disposition"] = disposition
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "projection item mapping[0]", "expected_disposition", "exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable")
	}
}

// TestMappingReasonVocabulary pins that mapping reason_codes admit
// core reasons and reverse-DNS extensions and refuse everything
// else on both entries.
func TestMappingReasonVocabulary(t *testing.T) {
	admit := []string{
		"target_no_equivalent",
		"source_not_persisted",
		"source_truncated",
		"source_corrupt",
		"foreign_encrypted_payload",
		"foreign_signature_unverifiable",
		"target_schema_constraint",
		"target_context_limit",
		"target_size_limit",
		"target_version_gate",
		"official_importer_loss",
		"graph_flattened",
		"unsafe_pending_action",
		"credential_excluded",
		"secret_policy",
		"operator_policy",
		"unsupported_media_type",
		"unknown_native_event",
		"derived_index_rebuilt",
		"com.example.reason",
	}
	for _, reason := range admit {
		input := validPlanInput()
		mapping := validMappingInput("reason-" + reason)
		mapping.ReasonCodes = []string{reason}
		input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
		sealed := mustBuildPlan(t, input)
		plan := mustDecodePlan(t, sealed)
		if len(plan.ItemMappings[0].ReasonCodes) != 1 || plan.ItemMappings[0].ReasonCodes[0] != reason {
			t.Errorf("reason_codes = %v, want [%q]", plan.ItemMappings[0].ReasonCodes, reason)
		}
	}
	refuse := []string{
		"",
		"frobnicate",
		"exact",
		"not a reason",
		"com.example.",
		".com.example",
		"target_no_equivalent ",
	}
	for _, reason := range refuse {
		input := validPlanInput()
		mapping := validMappingInput("reason-refuse")
		mapping.ReasonCodes = []string{reason}
		input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "projection item mapping[0]", "reason_codes")
		sealed := mustBuildPlan(t, validPlanInput())
		document := decodeDocument(t, sealed)
		mappings := document["item_mappings"].([]any)
		mappings[0].(map[string]any)["reason_codes"] = []any{reason}
		_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "projection item mapping[0]", "reason_codes")
	}
}
