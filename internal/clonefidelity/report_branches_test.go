package clonefidelity_test

import (
	"testing"

	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// TestArchiveTargetBranches pins the branch-exact nullability from
// the SPEC v0.7.0 Fidelity Report 1.0.0 table at both entries:
// projection_plan_id, target_environment, the staged/live manifest
// IDs (null exactly for archive), profile (archive_only exactly
// for archive), and the target booleans (both false for archive).
// Each scope admits its valid shape and refuses every single flip
// with a literal code.
func TestArchiveTargetBranches(t *testing.T) {
	row := validRowInput("branch-row", "exact")
	t.Run("archive_admits", func(t *testing.T) {
		mustDecode(t, mustBuild(t, validArchiveInput(row)))
	})
	t.Run("target_admits", func(t *testing.T) {
		mustDecode(t, mustBuild(t, validTargetInput(row)))
	})
	buildCases := []struct {
		name    string
		base    func(...clonefidelity.DispositionRecordInput) clonefidelity.FidelityReportInput
		mutate  func(*clonefidelity.FidelityReportInput)
		literal string
	}{
		{"archive_plan_nonnull", validArchiveInput,
			func(input *clonefidelity.FidelityReportInput) {
				input.ProjectionPlanID = strptr("sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
			},
			"projection_plan_id is not null for archive"},
		{"archive_target_env_nonnull", validArchiveInput,
			func(input *clonefidelity.FidelityReportInput) { input.TargetEnvironment = []byte(fixtureTupleJSON()) },
			"target_environment is not null for archive"},
		{"archive_staged_nonnull", validArchiveInput,
			func(input *clonefidelity.FidelityReportInput) {
				input.StagedReadBackEvidenceManifestID = strptr(fixtureDigest("staged"))
			},
			"staged_read_back_evidence_manifest_id is not null for archive"},
		{"archive_live_nonnull", validArchiveInput,
			func(input *clonefidelity.FidelityReportInput) {
				input.LiveReadBackEvidenceManifestID = strptr(fixtureDigest("live"))
			},
			"live_read_back_evidence_manifest_id is not null for archive"},
		{"archive_continuable_true", validArchiveInput,
			func(input *clonefidelity.FidelityReportInput) { input.TargetSemanticallyContinuable = true },
			"target_semantically_continuable is true for archive"},
		{"archive_resumable_true", validArchiveInput,
			func(input *clonefidelity.FidelityReportInput) { input.TargetNativelyResumable = true },
			"target_natively_resumable is true for archive"},
		{"target_plan_null", validTargetInput,
			func(input *clonefidelity.FidelityReportInput) { input.ProjectionPlanID = nil },
			"projection_plan_id is null for target"},
		{"target_env_null", validTargetInput,
			func(input *clonefidelity.FidelityReportInput) { input.TargetEnvironment = nil },
			"target_environment is not an Environment Tuple"},
		{"target_staged_null", validTargetInput,
			func(input *clonefidelity.FidelityReportInput) { input.StagedReadBackEvidenceManifestID = nil },
			"staged_read_back_evidence_manifest_id is null for target"},
		{"target_live_null", validTargetInput,
			func(input *clonefidelity.FidelityReportInput) { input.LiveReadBackEvidenceManifestID = nil },
			"live_read_back_evidence_manifest_id is null for target"},
	}
	for _, tc := range buildCases {
		t.Run("build/"+tc.name, func(t *testing.T) {
			input := tc.base(row)
			tc.mutate(&input)
			_, err := clonefidelity.BuildFidelityReport(input)
			requireRefusal(t, err, tc.literal)
		})
	}
	decodeCases := []struct {
		name    string
		archive bool
		mutate  func(document map[string]any)
		literal string
	}{
		{"archive_plan_nonnull", true,
			func(document map[string]any) {
				document["projection_plan_id"] = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
			},
			"projection_plan_id is not null for archive"},
		{"archive_target_env_nonnull", true,
			func(document map[string]any) { document["target_environment"] = decodeTupleMap(t) },
			"target_environment is not null for archive"},
		{"archive_staged_nonnull", true,
			func(document map[string]any) {
				document["staged_read_back_evidence_manifest_id"] = fixtureDigest("staged")
			},
			"staged_read_back_evidence_manifest_id is not null for archive"},
		{"archive_live_nonnull", true,
			func(document map[string]any) { document["live_read_back_evidence_manifest_id"] = fixtureDigest("live") },
			"live_read_back_evidence_manifest_id is not null for archive"},
		{"archive_continuable_true", true,
			func(document map[string]any) { document["target_semantically_continuable"] = true },
			"target_semantically_continuable is true for archive"},
		{"archive_resumable_true", true,
			func(document map[string]any) { document["target_natively_resumable"] = true },
			"target_natively_resumable is true for archive"},
		{"target_plan_null", false,
			func(document map[string]any) { document["projection_plan_id"] = nil },
			"projection_plan_id is not a digest for target"},
		{"target_env_null", false,
			func(document map[string]any) { document["target_environment"] = nil },
			"target_environment is not an Environment Tuple"},
		{"target_staged_null", false,
			func(document map[string]any) { document["staged_read_back_evidence_manifest_id"] = nil },
			"staged_read_back_evidence_manifest_id is not a digest for target"},
		{"target_live_null", false,
			func(document map[string]any) { document["live_read_back_evidence_manifest_id"] = nil },
			"live_read_back_evidence_manifest_id is not a digest for target"},
	}
	sealedArchive := mustBuild(t, validArchiveInput(row))
	sealedTarget := mustBuild(t, validTargetInput(row))
	for _, tc := range decodeCases {
		t.Run("decode/"+tc.name, func(t *testing.T) {
			sealed := sealedTarget
			if tc.archive {
				sealed = sealedArchive
			}
			mutated := tampered(t, sealed, tc.mutate)
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.literal)
		})
	}
	// Profile/scope coupling: archive admits only archive_only,
	// target admits every profile except archive_only, at both
	// entries.
	profiles := []string{"strict_exact", "maximal_safe", "compact", "messages_only", "archive_only"}
	for _, profile := range profiles {
		t.Run("build/archive/profile="+profile, func(t *testing.T) {
			input := validArchiveInput(row)
			input.Profile = profile
			if profile == "archive_only" {
				mustDecode(t, mustBuild(t, input))
				return
			}
			_, err := clonefidelity.BuildFidelityReport(input)
			requireRefusal(t, err, "want archive_only")
		})
		t.Run("build/target/profile="+profile, func(t *testing.T) {
			input := validTargetInput(row)
			input.Profile = profile
			if profile == "archive_only" {
				_, err := clonefidelity.BuildFidelityReport(input)
				requireRefusal(t, err, "profile is archive_only for target")
				return
			}
			mustDecode(t, mustBuild(t, input))
		})
		// Decode admission is proven by archive_admits/target_admits
		// (a tamper breaks the self digest); only refusal cells run
		// as tampers here.
		if profile != "archive_only" {
			t.Run("decode/archive/profile="+profile, func(t *testing.T) {
				mutated := tampered(t, sealedArchive, func(document map[string]any) { document["profile"] = profile })
				_, err := clonefidelity.DecodeFidelityReport(mutated)
				requireRefusal(t, err, "want archive_only")
			})
		}
		if profile == "archive_only" {
			t.Run("decode/target/profile="+profile, func(t *testing.T) {
				mutated := tampered(t, sealedTarget, func(document map[string]any) { document["profile"] = profile })
				_, err := clonefidelity.DecodeFidelityReport(mutated)
				requireRefusal(t, err, "profile is archive_only for target")
			})
		}
	}
	// Target booleans admit both values in target scope (only the
	// archive rule is pinned; target semantics are a stated bound).
	for _, continuable := range []bool{true, false} {
		for _, resumable := range []bool{true, false} {
			input := validTargetInput(row)
			input.TargetSemanticallyContinuable = continuable
			input.TargetNativelyResumable = resumable
			report := mustDecode(t, mustBuild(t, input))
			if report.TargetSemanticallyContinuable != continuable || report.TargetNativelyResumable != resumable {
				t.Fatalf("target booleans = %t/%t, want %t/%t",
					report.TargetSemanticallyContinuable, report.TargetNativelyResumable, continuable, resumable)
			}
		}
	}
	// raw_bundle_complete and canonical_complete admit both values in
	// both scopes (core-derived, undefined here: a stated bound).
	for _, scope := range []string{"archive", "target"} {
		for _, raw := range []bool{true, false} {
			for _, canonical := range []bool{true, false} {
				var input clonefidelity.FidelityReportInput
				if scope == "archive" {
					input = validArchiveInput(row)
				} else {
					input = validTargetInput(row)
				}
				input.RawBundleComplete = raw
				input.CanonicalComplete = canonical
				report := mustDecode(t, mustBuild(t, input))
				if report.RawBundleComplete != raw || report.CanonicalComplete != canonical {
					t.Fatalf("%s completeness = %t/%t, want %t/%t",
						scope, report.RawBundleComplete, report.CanonicalComplete, raw, canonical)
				}
			}
		}
	}
}

func decodeTupleMap(t *testing.T) map[string]any {
	t.Helper()
	return decodeDocument(t, []byte(fixtureTupleJSON()))
}

// TestRequiredDispositions pins the policy map at both entries:
// closed capture-class keys (all nine admit) to sorted unique
// non-empty disposition sets.
func TestRequiredDispositions(t *testing.T) {
	classes := []string{
		"durable_payload", "durable_index_required", "durable_sidecar",
		"derived_cache_optional", "credential", "machine_auth",
		"runtime_state", "transient_lock", "unknown",
	}
	row := validRowInput("required-row", "exact")
	// Every class admits as a key with a valid set.
	full := map[string][]string{}
	for _, class := range classes {
		full[class] = []string{"exact", "semantic"}
	}
	input := validTargetInput(row)
	input.RequiredDispositions = full
	report := mustDecode(t, mustBuild(t, input))
	if len(report.RequiredDispositions) != 9 {
		t.Fatalf("required_dispositions keys = %d, want 9", len(report.RequiredDispositions))
	}
	// The empty map admits (a subset map, not a complete one).
	input.RequiredDispositions = map[string][]string{}
	mustDecode(t, mustBuild(t, input))
	buildCases := []struct {
		name    string
		value   map[string][]string
		literal string
	}{
		{"unknown_class", map[string][]string{"frobnicate": {"exact"}}, `carry unknown class "frobnicate"`},
		{"event_kind_key", map[string][]string{"user_message": {"exact"}}, `carry unknown class "user_message"`},
		{"empty_set", map[string][]string{"unknown": {}}, "carry no dispositions, want a non-empty set"},
		{"bad_disposition", map[string][]string{"unknown": {"exactx"}}, "disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable"},
		{"unsorted_set", map[string][]string{"unknown": {"semantic", "exact"}}, "are not sorted unique"},
		{"duplicate_set", map[string][]string{"unknown": {"exact", "exact"}}, "are not sorted unique"},
	}
	for _, tc := range buildCases {
		t.Run("build/"+tc.name, func(t *testing.T) {
			input := validTargetInput(row)
			input.RequiredDispositions = tc.value
			_, err := clonefidelity.BuildFidelityReport(input)
			requireRefusal(t, err, tc.literal)
		})
	}
	sealed := mustBuild(t, validTargetInput(row))
	decodeCases := []struct {
		name    string
		value   any
		literal string
	}{
		{"unknown_class", map[string]any{"frobnicate": []any{"exact"}}, `carry unknown class "frobnicate"`},
		{"empty_set", map[string]any{"unknown": []any{}}, "carry no dispositions, want a non-empty set"},
		{"bad_disposition", map[string]any{"unknown": []any{"exactx"}}, "disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable"},
		{"unsorted_set", map[string]any{"unknown": []any{"semantic", "exact"}}, "are not sorted unique"},
		{"duplicate_set", map[string]any{"unknown": []any{"exact", "exact"}}, "are not sorted unique"},
		{"non_array", map[string]any{"unknown": "exact"}, "carry no dispositions, want a non-empty set"},
		{"non_string_item", map[string]any{"unknown": []any{7}}, "disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable"},
	}
	for _, tc := range decodeCases {
		t.Run("decode/"+tc.name, func(t *testing.T) {
			mutated := tampered(t, sealed, func(document map[string]any) { document["required_dispositions"] = tc.value })
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.literal)
		})
	}
	// A non-object member refuses at the frame gate.
	mutated := tampered(t, sealed, func(document map[string]any) { document["required_dispositions"] = []any{} })
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "required_dispositions")
}

// TestForbidReasons pins forbid_reasons at both entries: sorted
// unique string[1..128][0..128] of LITERAL strings (the table row
// types them as strings, not reason codes, so a non-reason string
// admits and stays).
func TestForbidReasons(t *testing.T) {
	row := validRowInput("forbid-row", "exact")
	input := validTargetInput(row)
	input.ForbidReasons = []string{"credential_excluded", "frobnicate"}
	report := mustDecode(t, mustBuild(t, input))
	if len(report.ForbidReasons) != 2 || report.ForbidReasons[1] != "frobnicate" {
		t.Fatalf("forbid_reasons = %q, want the literal strings kept", report.ForbidReasons)
	}
	input.ForbidReasons = []string{}
	mustDecode(t, mustBuild(t, input))
	edge := make([]string, 128)
	for i := range edge {
		edge[i] = "reason-" + itoa(1000+i)
	}
	// Sorted: reason-1000 .. reason-1127 sort lexicographically.
	input.ForbidReasons = edge
	mustDecode(t, mustBuild(t, input))
	over := make([]string, 129)
	for i := range over {
		over[i] = "reason-" + itoa(2000+i)
	}
	input.ForbidReasons = over
	_, err := clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "carries 129 forbid_reasons, want [0..128]")
	buildCases := []struct {
		name    string
		value   []string
		literal string
	}{
		{"unsorted", []string{"b-reason", "a-reason"}, "forbid_reasons are not sorted unique"},
		{"duplicate", []string{"a-reason", "a-reason"}, "forbid_reasons are not sorted unique"},
		{"item_too_long", []string{repeat("f", 129)}, "forbid_reasons[0] is not a string[1..128]"},
		{"item_empty", []string{""}, "forbid_reasons[0] is not a string[1..128]"},
	}
	for _, tc := range buildCases {
		t.Run("build/"+tc.name, func(t *testing.T) {
			input := validTargetInput(row)
			input.ForbidReasons = tc.value
			_, err := clonefidelity.BuildFidelityReport(input)
			requireRefusal(t, err, tc.literal)
		})
	}
	sealed := mustBuild(t, validTargetInput(row))
	decodeCases := []struct {
		name    string
		value   any
		literal string
	}{
		{"unsorted", []any{"b-reason", "a-reason"}, "forbid_reasons are not sorted unique string[1..128][0..128]"},
		{"duplicate", []any{"a-reason", "a-reason"}, "forbid_reasons are not sorted unique string[1..128][0..128]"},
		{"item_too_long", []any{repeat("f", 129)}, "forbid_reasons are not sorted unique string[1..128][0..128]"},
		{"item_empty", []any{""}, "forbid_reasons are not sorted unique string[1..128][0..128]"},
		{"non_string", []any{7}, "forbid_reasons are not sorted unique string[1..128][0..128]"},
	}
	for _, tc := range decodeCases {
		t.Run("decode/"+tc.name, func(t *testing.T) {
			mutated := tampered(t, sealed, func(document map[string]any) { document["forbid_reasons"] = tc.value })
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.literal)
		})
	}
	// Decode count edge: 129 items refuse at the shape gate.
	many := make([]any, 129)
	for i := range many {
		many[i] = "reason-" + itoa(3000+i)
	}
	mutated := tampered(t, sealed, func(document map[string]any) { document["forbid_reasons"] = many })
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "forbid_reasons are not sorted unique string[1..128][0..128]")
}

// TestRowOrdering pins the disposition order at both entries: source
// rows sorted unique by item key, then synthesized rows sorted
// unique by item key, with keys unique across all rows.
func TestRowOrdering(t *testing.T) {
	admit := []clonefidelity.DispositionRecordInput{
		validRowInput("k1", "exact"),
		validRowInput("k2", "semantic", "target_no_equivalent"),
		validSynthInput("k3"),
		validSynthInput("k4"),
	}
	mustDecode(t, mustBuild(t, validTargetInput(admit...)))
	single := []clonefidelity.DispositionRecordInput{validSynthInput("only")}
	mustDecode(t, mustBuild(t, validTargetInput(single...)))
	buildCases := []struct {
		name    string
		rows    []clonefidelity.DispositionRecordInput
		literal string
	}{
		{"synth_before_source",
			[]clonefidelity.DispositionRecordInput{validSynthInput("k1"), validRowInput("k2", "exact")},
			"order a source row after synthesized rows"},
		{"source_unsorted",
			[]clonefidelity.DispositionRecordInput{validRowInput("k2", "exact"), validRowInput("k1", "exact")},
			"source dispositions are not sorted unique by source item key"},
		{"synth_unsorted",
			[]clonefidelity.DispositionRecordInput{validSynthInput("k2"), validSynthInput("k1")},
			"synthesized dispositions are not sorted unique by source item key"},
		{"duplicate_source_keys",
			[]clonefidelity.DispositionRecordInput{validRowInput("k1", "exact"), validRowInput("k1", "semantic", "target_no_equivalent")},
			`duplicate source item key "k1"`},
		{"duplicate_across_groups",
			[]clonefidelity.DispositionRecordInput{validRowInput("k1", "exact"), validSynthInput("k1")},
			`duplicate source item key "k1"`},
	}
	for _, tc := range buildCases {
		t.Run("build/"+tc.name, func(t *testing.T) {
			_, err := clonefidelity.BuildFidelityReport(validTargetInput(tc.rows...))
			requireRefusal(t, err, tc.literal)
		})
	}
	// Decode side tampers row keys and dispositions on valid sealed
	// reports. The tamper breaks aggregates too, but the order gate
	// runs before reconciliation.
	sealed := mustBuild(t, validTargetInput(admit...))
	decodeCases := []struct {
		name    string
		mutate  func(document map[string]any)
		literal string
	}{
		{"synth_before_source",
			func(document map[string]any) {
				rows := document["dispositions"].([]any)
				rows[0].(map[string]any)["disposition"] = "synthesized"
				rows[0].(map[string]any)["canonical_object_id"] = nil
				rows[0].(map[string]any)["reason_codes"] = []any{"derived_index_rebuilt"}
			},
			"order a source row after synthesized rows"},
		{"source_unsorted",
			func(document map[string]any) {
				rows := document["dispositions"].([]any)
				rows[0].(map[string]any)["source_item_key"] = "k9"
			},
			"source dispositions are not sorted unique by source item key"},
		{"synth_unsorted",
			func(document map[string]any) {
				rows := document["dispositions"].([]any)
				rows[3].(map[string]any)["source_item_key"] = "k0"
			},
			"synthesized dispositions are not sorted unique by source item key"},
		{"duplicate_source_keys",
			func(document map[string]any) {
				rows := document["dispositions"].([]any)
				rows[1].(map[string]any)["source_item_key"] = "k1"
			},
			`duplicate source item key "k1"`},
		{"duplicate_across_groups",
			func(document map[string]any) {
				rows := document["dispositions"].([]any)
				rows[2].(map[string]any)["source_item_key"] = "k1"
			},
			`duplicate source item key "k1"`},
	}
	for _, tc := range decodeCases {
		t.Run("decode/"+tc.name, func(t *testing.T) {
			mutated := tampered(t, sealed, tc.mutate)
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.literal)
		})
	}
}

// TestRowCountEdges pins the dispositions cardinality: zero rows
// refuse at both entries, one row admits, and 1000001 inputs refuse
// at Build (the count gate runs before per-row work). The exact
// upper-edge admission is pinned by the factored gate test (see
// rowcount_internal_test.go) paired with these entry reachability
// rows.
func TestRowCountEdges(t *testing.T) {
	_, err := clonefidelity.BuildFidelityReport(validTargetInput())
	requireRefusal(t, err, "carries 0 dispositions, want [1..1000000]")
	sealed := mustBuild(t, validTargetInput(validRowInput("count-row", "exact")))
	mutated := tampered(t, sealed, func(document map[string]any) { document["dispositions"] = []any{} })
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "carries 0 dispositions, want [1..1000000]")
	huge := make([]clonefidelity.DispositionRecordInput, 1000001)
	_, err = clonefidelity.BuildFidelityReport(validTargetInput(huge...))
	requireRefusal(t, err, "carries 1000001 dispositions, want [1..1000000]")
}
