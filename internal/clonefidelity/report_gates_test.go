package clonefidelity_test

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// TestScalarGates pins the delegated digest, UUIDv7, tuple, uint53,
// and boolean gates at the production entries: each refusal names
// its member with a literal detail. The value grammars belong to
// the landed owners; these rows pin that the entries reach them.
func TestScalarGates(t *testing.T) {
	row := validRowInput("scalar-row", "exact")
	buildCases := []struct {
		name    string
		mutate  func(*clonefidelity.FidelityReportInput)
		literal string
	}{
		{"operation_bad_uuid", func(input *clonefidelity.FidelityReportInput) { input.OperationID = "not-a-uuid" }, "operation_id is not a UUIDv7"},
		{"operation_uuid4", func(input *clonefidelity.FidelityReportInput) {
			input.OperationID = "0198f4c8-8e50-4f66-8f70-111111111111"
		}, "operation_id is not a UUIDv7"},
		{"bundle_bad_uuid", func(input *clonefidelity.FidelityReportInput) { input.BundleID = "0198f4c8-8e50-7f66-8f70-11111111111" }, "bundle_id is not a UUIDv7"},
		{"snapshot_bad_digest", func(input *clonefidelity.FidelityReportInput) { input.SourceSnapshotDigest = "sha256:xyz" }, "source_snapshot_digest is not a digest"},
		{"capture_bad_digest", func(input *clonefidelity.FidelityReportInput) { input.CaptureManifestID = "not-a-digest" }, "capture_manifest_id is not a digest"},
		{"canonical_bad_digest", func(input *clonefidelity.FidelityReportInput) { input.CanonicalSessionID = "" }, "canonical_session_id is not a digest"},
		{"plan_bad_digest", func(input *clonefidelity.FidelityReportInput) { input.ProjectionPlanID = strptr("sha256:xyz") }, "projection_plan_id is not a digest"},
		{"source_tuple_bad", func(input *clonefidelity.FidelityReportInput) {
			input.SourceEnvironment = []byte(`{"platform":"linux"}`)
		}, "source_environment is not an Environment Tuple"},
		{"source_tuple_not_json", func(input *clonefidelity.FidelityReportInput) { input.SourceEnvironment = []byte(`nope`) }, "source_environment is not an Environment Tuple"},
		{"target_tuple_bad", func(input *clonefidelity.FidelityReportInput) { input.TargetEnvironment = []byte(`[]`) }, "target_environment is not an Environment Tuple"},
		{"staged_bad_digest", func(input *clonefidelity.FidelityReportInput) {
			input.StagedReadBackEvidenceManifestID = strptr("nope")
		}, "staged_read_back_evidence_manifest_id is not a digest"},
		{"live_bad_digest", func(input *clonefidelity.FidelityReportInput) { input.LiveReadBackEvidenceManifestID = strptr("nope") }, "live_read_back_evidence_manifest_id is not a digest"},
		{"attestation_bad_digest", func(input *clonefidelity.FidelityReportInput) { input.AdapterAttestations = []string{"nope"} }, "adapter_attestations[0] is not a digest"},
		{"attestation_unsorted", func(input *clonefidelity.FidelityReportInput) {
			input.AdapterAttestations = []string{
				"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			}
		}, "adapter_attestations are not sorted unique"},
		{"row_canonical_bad_digest", func(input *clonefidelity.FidelityReportInput) {
			input.Rows = []clonefidelity.DispositionRecordInput{validRowInput("scalar-row", "exact")}
			input.Rows[0].CanonicalObjectID = strptr("nope")
		}, "canonical_object_id is not a digest"},
		{"row_evidence_bad_digest", func(input *clonefidelity.FidelityReportInput) {
			input.Rows = []clonefidelity.DispositionRecordInput{validRowInput("scalar-row", "exact")}
			input.Rows[0].SourceEvidenceIDs = []string{"nope"}
			events, blocks, bytes := concentratedBreakdowns(input.Rows)
			input.EventKindCounts, input.ContentBlockCounts, input.ByteCounts = events, blocks, bytes
		}, "source_evidence_ids[0] is not a digest"},
	}
	for _, tc := range buildCases {
		t.Run("build/"+tc.name, func(t *testing.T) {
			input := validTargetInput(row)
			tc.mutate(&input)
			_, err := clonefidelity.BuildFidelityReport(input)
			requireRefusal(t, err, tc.literal)
		})
	}
	sealed := mustBuild(t, validTargetInput(row))
	decodeCases := []struct {
		name    string
		mutate  func(document map[string]any)
		literal string
	}{
		{"operation_bad_uuid", func(document map[string]any) { document["operation_id"] = "not-a-uuid" }, "operation_id is not a UUIDv7"},
		{"bundle_bad_uuid", func(document map[string]any) { document["bundle_id"] = 7 }, "bundle_id is not a UUIDv7"},
		{"snapshot_bad_digest", func(document map[string]any) { document["source_snapshot_digest"] = "sha256:xyz" }, "source_snapshot_digest is not a digest"},
		{"capture_bad_digest", func(document map[string]any) { document["capture_manifest_id"] = nil }, "capture_manifest_id is not a digest"},
		{"plan_bad_digest", func(document map[string]any) { document["projection_plan_id"] = "nope" }, "projection_plan_id is not a digest for target"},
		{"source_tuple_bad", func(document map[string]any) { document["source_environment"] = map[string]any{"platform": "linux"} }, "source_environment is not an Environment Tuple"},
		{"target_tuple_bad_platform", func(document map[string]any) {
			tuple := document["target_environment"].(map[string]any)
			tuple["platform"] = "plan9"
		}, "target_environment is not an Environment Tuple"},
		{"target_tuple_unknown_member", func(document map[string]any) {
			tuple := document["target_environment"].(map[string]any)
			tuple["extensions"] = map[string]any{}
		}, "target_environment is not an Environment Tuple"},
		{"staged_bad_digest", func(document map[string]any) { document["staged_read_back_evidence_manifest_id"] = "nope" }, "staged_read_back_evidence_manifest_id is not a digest for target"},
		{"live_null", func(document map[string]any) { document["live_read_back_evidence_manifest_id"] = nil }, "live_read_back_evidence_manifest_id is not a digest for target"},
		{"attestation_bad_digest", func(document map[string]any) { document["adapter_attestations"] = []any{"nope"} }, "adapter_attestations are not sorted unique digest[0..64]"},
		{"attestation_unsorted", func(document map[string]any) {
			document["adapter_attestations"] = []any{
				"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			}
		}, "adapter_attestations are not sorted unique digest[0..64]"},
		{"row_canonical_bad_digest", func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { row["canonical_object_id"] = "nope" })
		}, "canonical_object_id is not a digest or null"},
		{"row_evidence_bad_digest", func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { row["source_evidence_ids"] = []any{"nope"} })
		}, "source_evidence_ids are not sorted unique digest[1..65536]"},
		{"raw_complete_string", func(document map[string]any) { document["raw_bundle_complete"] = "true" }, "raw_bundle_complete is not a boolean"},
		{"canonical_complete_number", func(document map[string]any) { document["canonical_complete"] = 1 }, "canonical_complete is not a boolean"},
		{"continuable_null", func(document map[string]any) { document["target_semantically_continuable"] = nil }, "target_semantically_continuable is not a boolean"},
		{"resumable_number", func(document map[string]any) { document["target_natively_resumable"] = 0 }, "target_natively_resumable is not a boolean"},
	}
	for _, tc := range decodeCases {
		t.Run("decode/"+tc.name, func(t *testing.T) {
			mutated := tampered(t, sealed, tc.mutate)
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.literal)
		})
	}
}

// TestNumberModel pins the AX number model at the numeric gates: a
// fraction, an exponent, a string, or a magnitude at or beyond 2^53
// refuses with a literal code; 2^53-1 admits.
func TestNumberModel(t *testing.T) {
	row := validRowInput("number-row", "exact")
	sealed := mustBuild(t, validTargetInput(row))
	cells := []struct {
		name   string
		mutate func(document map[string]any, value any)
	}{
		{"counts", func(document map[string]any, value any) { document["counts"].(map[string]any)["exact"] = value }},
		{"event_cell", func(document map[string]any, value any) {
			document["event_kind_counts"].(map[string]any)["usage"].(map[string]any)["exact"] = value
		}},
		{"block_cell", func(document map[string]any, value any) {
			document["content_block_counts"].(map[string]any)["text"].(map[string]any)["exact"] = value
		}},
		{"bytes", func(document map[string]any, value any) { document["byte_counts"].(map[string]any)["exact"] = value }},
	}
	values := []any{1.5, "7", true, nil, 9007199254740992, -1, json.Number("1e3")}
	for _, cell := range cells {
		for _, value := range values {
			mutated := tampered(t, sealed, func(document map[string]any) { cell.mutate(document, value) })
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			if err == nil {
				t.Fatalf("%s value %v admitted, want refusal", cell.name, value)
			}
			requireRefusal(t, err, "is not a uint53")
		}
	}
	// 2^53-1 admits in a byte cell (row reconciliation cannot apply
	// to bytes, so the max value seals and decodes).
	input := validTargetInput(row)
	input.ByteCounts["exact"] = 9007199254740991
	report := mustDecode(t, mustBuild(t, input))
	if report.ByteCounts["exact"] != 9007199254740991 {
		t.Fatalf("byte max = %d, want 9007199254740991", report.ByteCounts["exact"])
	}
}

// TestEvidenceBounds pins the evidence array bounds at the edge and
// edge+1 through both entries: source[1..65536],
// staged/live[0..65536], attestations[0..64].
func TestEvidenceBounds(t *testing.T) {
	many := func(n int, seed string) []string {
		out := make([]string, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, fixtureDigest(seed+itoa(i)))
		}
		sort.Strings(out)
		return out
	}
	// Admit edges: 65536 source IDs seal and decode.
	edgeRow := validRowInput("edge-evidence", "exact")
	edgeRow.SourceEvidenceIDs = many(65536, "edge-evidence-")
	edgeRow.StagedEvidenceObjectIDs = many(65536, "edge-staged-")
	edgeRow.LiveEvidenceObjectIDs = many(65536, "edge-live-")
	report := mustDecode(t, mustBuild(t, validTargetInput(edgeRow)))
	if len(report.Rows[0].SourceEvidenceIDs) != 65536 {
		t.Fatalf("source evidence = %d, want 65536", len(report.Rows[0].SourceEvidenceIDs))
	}
	input := validTargetInput(validRowInput("edge-att", "exact"))
	input.AdapterAttestations = many(64, "edge-att-")
	mustDecode(t, mustBuild(t, input))
	// Edge+1 refuses at Build (count first: fast).
	overRow := validRowInput("over-evidence", "exact")
	overRow.SourceEvidenceIDs = many(65537, "over-evidence-")
	_, err := clonefidelity.BuildFidelityReport(validTargetInput(overRow))
	requireRefusal(t, err, "carries 65537 source_evidence_ids, want [1..65536]")
	overRow = validRowInput("over-evidence", "exact")
	overRow.StagedEvidenceObjectIDs = many(65537, "over-staged-")
	_, err = clonefidelity.BuildFidelityReport(validTargetInput(overRow))
	requireRefusal(t, err, "carries 65537 staged_evidence_object_ids, want [0..65536]")
	input = validTargetInput(validRowInput("over-att", "exact"))
	input.AdapterAttestations = many(65, "over-att-")
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "carries 65 adapter_attestations, want [0..64]")
	// Edge+1 refuses at Decode (shape gate).
	sealed := mustBuild(t, validTargetInput(validRowInput("over-decode", "exact")))
	toAny := func(ids []string) []any {
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, id)
		}
		return out
	}
	mutated := tampered(t, sealed, func(document map[string]any) {
		tamperRow(document, 0, func(row map[string]any) {
			row["source_evidence_ids"] = toAny(many(65537, "over-decode-evidence-"))
		})
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "source_evidence_ids are not sorted unique digest[1..65536]")
	mutated = tampered(t, sealed, func(document map[string]any) {
		document["adapter_attestations"] = toAny(many(65, "over-decode-att-"))
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "adapter_attestations are not sorted unique digest[0..64]")
}

// TestExtensionsClosed pins the closed extensions rule at both
// entries and both layers: reverse-DNS keys only, values inside the
// AX number model, nested objects duplicate-free.
func TestExtensionsClosed(t *testing.T) {
	row := validRowInput("ext-row", "exact")
	input := validTargetInput(row)
	input.Extensions = map[string]any{"com.example.note": "kept", "com.example.nested": map[string]any{"n": 7}}
	report := mustDecode(t, mustBuild(t, input))
	if report.Scope != "target" {
		t.Fatal("valid extensions refused")
	}
	buildCases := []struct {
		name       string
		extensions map[string]any
		literal    string
	}{
		{"bad_key", map[string]any{"frobnicate": 1}, "extensions are not reverse-DNS keyed"},
		{"float_value", map[string]any{"com.example.f": 1.5}, "extensions carry a number outside the AX safe-integer model"},
		{"big_value", map[string]any{"com.example.f": uint64(1) << 53}, "extensions carry a number outside the AX safe-integer model"},
	}
	for _, tc := range buildCases {
		t.Run("build/"+tc.name, func(t *testing.T) {
			input := validTargetInput(row)
			input.Extensions = tc.extensions
			_, err := clonefidelity.BuildFidelityReport(input)
			requireRefusal(t, err, tc.literal)
		})
		t.Run("build/row_"+tc.name, func(t *testing.T) {
			badRow := validRowInput("ext-row", "exact")
			badRow.Extensions = tc.extensions
			_, err := clonefidelity.BuildFidelityReport(validTargetInput(badRow))
			requireRefusal(t, err, tc.literal)
		})
	}
	sealed := mustBuild(t, validTargetInput(row))
	mutated := tampered(t, sealed, func(document map[string]any) {
		document["extensions"] = map[string]any{"frobnicate": 1}
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "extensions are not reverse-DNS keyed")
	mutated = tampered(t, sealed, func(document map[string]any) {
		document["extensions"] = map[string]any{"com.example.f": 1.5}
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "extensions carry a number outside the AX safe-integer model")
	mutated = tampered(t, sealed, func(document map[string]any) {
		tamperRow(document, 0, func(row map[string]any) {
			row["extensions"] = map[string]any{"frobnicate": 1}
		})
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "extensions are not reverse-DNS keyed")
	// Nested duplicate member refuses (raw-string splice, since a map
	// cannot hold the duplicate).
	text := string(sealed)
	marker := `"extensions":{}`
	if !strings.Contains(text, marker) {
		t.Fatalf("sealed report lacks %q", marker)
	}
	dup := strings.Replace(text, marker, `"extensions":{"com.example.a":{"x":1,"x":2}}`, 1)
	_, err = clonefidelity.DecodeFidelityReport([]byte(dup))
	requireRefusal(t, err, "extensions carry a duplicate nested member")
}

// TestBuildUTF8 pins the pre-marshal UTF-8 gate: every Build string
// admission refuses invalid bytes instead of sealing a rewritten
// U+FFFD twin.
func TestBuildUTF8(t *testing.T) {
	bad := string([]byte{0xff, 0xfe})
	row := validRowInput("utf8-row", "exact")
	row.Explanation = bad
	_, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
	requireRefusal(t, err, "explanation is not valid UTF-8")
	row = validRowInput("utf8-row", "semantic", "unknown_native_event")
	row.SourceItemKey = bad
	_, err = clonefidelity.BuildFidelityReport(validTargetInput(row))
	requireRefusal(t, err, "source_item_key is not valid UTF-8")
	row = validRowInput("utf8-row", "semantic", "unknown_native_event")
	row.ReasonCodes = []string{bad}
	_, err = clonefidelity.BuildFidelityReport(validTargetInput(row))
	requireRefusal(t, err, "reason_codes[0] is not valid UTF-8")
	input := validTargetInput(validRowInput("utf8-row", "exact"))
	input.Extensions = map[string]any{"com.example.bad": bad}
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "extensions")
}
