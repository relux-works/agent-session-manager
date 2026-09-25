package clonefidelity_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// Seven-disposition fixture: two exact rows plus one of every other
// disposition, so every count is non-zero and both +1 and -1
// perturbations stay inside uint53.
func sevenDispositionRows() []clonefidelity.DispositionRecordInput {
	return []clonefidelity.DispositionRecordInput{
		validRowInput("agg-exact-1", "exact"),
		validRowInput("agg-exact-2", "exact"),
		validRowInput("agg-omitted", "omitted", "credential_excluded"),
		validRowInput("agg-opaque", "opaque_preserved", "foreign_encrypted_payload"),
		validRowInput("agg-semantic", "semantic", "target_no_equivalent"),
		validRowInput("agg-summarized", "summarized", "source_truncated", "target_context_limit"),
		validRowInput("agg-unrecoverable", "unrecoverable", "source_corrupt"),
		validSynthInput("agg-synth"),
	}
}

// TestCountsDerived pins the counts derivation with literal values,
// then perturbs every counts cell by +1 and -1: each tampered cell
// refuses with the reconciliation detail naming the disposition.
func TestCountsDerived(t *testing.T) {
	rows := sevenDispositionRows()
	report := mustDecode(t, mustBuild(t, validTargetInput(rows...)))
	want := map[string]uint64{
		"exact": 2, "semantic": 1, "summarized": 1, "opaque_preserved": 1,
		"synthesized": 1, "omitted": 1, "unrecoverable": 1,
	}
	for disposition, count := range want {
		var got uint64
		switch disposition {
		case "exact":
			got = report.Counts.Exact
		case "semantic":
			got = report.Counts.Semantic
		case "summarized":
			got = report.Counts.Summarized
		case "opaque_preserved":
			got = report.Counts.OpaquePreserved
		case "synthesized":
			got = report.Counts.Synthesized
		case "omitted":
			got = report.Counts.Omitted
		case "unrecoverable":
			got = report.Counts.Unrecoverable
		}
		if got != count {
			t.Fatalf("counts.%s = %d, want %d", disposition, got, count)
		}
	}
	dispositions := []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"}
	sealed := mustBuild(t, validTargetInput(rows...))
	perturbed := 0
	for _, disposition := range dispositions {
		for _, delta := range []int64{1, -1} {
			var next uint64
			if delta < 0 {
				next = want[disposition] - 1
			} else {
				next = want[disposition] + 1
			}
			mutated := tampered(t, sealed, func(document map[string]any) {
				document["counts"].(map[string]any)[disposition] = next
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, "counts "+disposition+" reconciles")
			perturbed++
		}
	}
	if perturbed != 14 {
		t.Fatalf("perturbed cells = %d, want 14", perturbed)
	}
	// A -1 perturbation on a zero cell refuses at the uint53 gate
	// instead (documented boundary, still a refusal).
	singleSealed := mustBuild(t, validTargetInput(validRowInput("single", "exact")))
	zeroMutated := tampered(t, singleSealed, func(document map[string]any) {
		document["counts"].(map[string]any)["semantic"] = -1
	})
	_, err := clonefidelity.DecodeFidelityReport(zeroMutated)
	requireRefusal(t, err, "counts semantic is not a uint53")
}

// TestReasonCountsDerived pins the reason derivation with literal
// occurrence counts (including a reason shared across rows), then
// perturbs every reason cell by +1 and -1 and adds extra/missing
// key cases: each refuses with the reconciliation detail.
func TestReasonCountsDerived(t *testing.T) {
	rows := []clonefidelity.DispositionRecordInput{
		validRowInput("rc-1", "semantic", "unknown_native_event"),
		validRowInput("rc-2", "summarized", "target_context_limit", "unknown_native_event"),
		validRowInput("rc-3", "omitted", "credential_excluded"),
		validRowInput("rc-4", "exact"),
	}
	report := mustDecode(t, mustBuild(t, validTargetInput(rows...)))
	want := map[string]uint64{
		"unknown_native_event": 2, "target_context_limit": 1, "credential_excluded": 1,
	}
	if len(report.ReasonCounts) != len(want) {
		t.Fatalf("reason_counts keys = %d, want %d", len(report.ReasonCounts), len(want))
	}
	for reason, count := range want {
		if report.ReasonCounts[reason] != count {
			t.Fatalf("reason_counts[%q] = %d, want %d", reason, report.ReasonCounts[reason], count)
		}
	}
	sealed := mustBuild(t, validTargetInput(rows...))
	perturbed := 0
	for reason, count := range want {
		for _, delta := range []int64{1, -1} {
			var next uint64
			if delta < 0 {
				next = count - 1
			} else {
				next = count + 1
			}
			mutated := tampered(t, sealed, func(document map[string]any) {
				document["reason_counts"].(map[string]any)[reason] = next
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, "reason_counts")
			perturbed++
		}
	}
	// Extra key and missing key refuse.
	mutated := tampered(t, sealed, func(document map[string]any) {
		document["reason_counts"].(map[string]any)["frobnicate"] = 1
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `reason_counts "frobnicate" reconciles`)
	mutated = tampered(t, sealed, func(document map[string]any) {
		delete(document["reason_counts"].(map[string]any), "credential_excluded")
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `reason_counts miss reason "credential_excluded"`)
	// A zero value refuses at the shape gate (uint53 above zero).
	mutated = tampered(t, sealed, func(document map[string]any) {
		document["reason_counts"].(map[string]any)["credential_excluded"] = 0
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `reason_counts["credential_excluded"] is not a uint53 above zero`)
	if perturbed != 6 {
		t.Fatalf("perturbed cells = %d, want 6", perturbed)
	}
}

// TestEventKindBreakdown pins the closed 26-kind map: every kind key
// is required, no other key admits, and every one of the 182 cells
// reconciles — a +1 perturbation in any single cell refuses, as
// does a -1 on every non-zero cell.
func TestEventKindBreakdown(t *testing.T) {
	kinds := []string{
		"session_started", "instruction_snapshot", "user_message",
		"assistant_message", "reasoning_summary", "opaque_reasoning",
		"tool_definition_snapshot", "tool_call", "tool_result",
		"approval_request", "approval_response", "plan_update",
		"progress", "usage", "rate_limit", "file_change",
		"compaction", "turn_started", "turn_completed", "turn_aborted",
		"subagent_started", "subagent_completed", "error",
		"migration_checkpoint", "session_finished", "opaque_event",
	}
	if len(kinds) != 26 {
		t.Fatalf("oracle kind count = %d, want 26", len(kinds))
	}
	rows := sevenDispositionRows()
	sealed := mustBuild(t, validTargetInput(rows...))
	report := mustDecode(t, sealed)
	if len(report.EventKindCounts) != 26 {
		t.Fatalf("event_kind_counts keys = %d, want 26", len(report.EventKindCounts))
	}
	dispositions := []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"}
	// Concentrated fixture: every row sits on the first kind, so all
	// other cells are zero.
	first := kinds[0]
	concentrated := report.EventKindCounts[first]
	if concentrated.Exact != 2 || concentrated.Semantic != 1 || concentrated.Unrecoverable != 1 {
		t.Fatalf("concentrated kind = %+v, want the row totals", concentrated)
	}
	perturbed := 0
	for _, kind := range kinds {
		for _, disposition := range dispositions {
			mutated := tampered(t, sealed, func(document map[string]any) {
				cell := document["event_kind_counts"].(map[string]any)[kind].(map[string]any)
				cell[disposition] = cellValue(cell[disposition]) + 1
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, "event_kind_counts reconciles")
			perturbed++
		}
	}
	if perturbed != 182 {
		t.Fatalf("perturbed cells = %d, want 182", perturbed)
	}
	// -1 on the seven non-zero cells refuses as well.
	for _, disposition := range dispositions {
		mutated := tampered(t, sealed, func(document map[string]any) {
			cell := document["event_kind_counts"].(map[string]any)[first].(map[string]any)
			cell[disposition] = cellValue(cell[disposition]) - 1
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, "event_kind_counts reconciles")
	}
	// Closed keys: a 27th key and a missing key refuse.
	mutated := tampered(t, sealed, func(document map[string]any) {
		document["event_kind_counts"].(map[string]any)["frobnicate"] = map[string]any{}
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `event_kind_counts carry unknown key "frobnicate"`)
	mutated = tampered(t, sealed, func(document map[string]any) {
		delete(document["event_kind_counts"].(map[string]any), "usage")
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `event_kind_counts miss key "usage"`)
	// A malformed cell refuses naming the key.
	mutated = tampered(t, sealed, func(document map[string]any) {
		document["event_kind_counts"].(map[string]any)["usage"].(map[string]any)["exact"] = "seven"
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `event_kind_counts["usage"] exact is not a uint53`)
	// A cell missing a member refuses naming the key and member.
	mutated = tampered(t, sealed, func(document map[string]any) {
		delete(document["event_kind_counts"].(map[string]any)["usage"].(map[string]any), "exact")
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `event_kind_counts["usage"] misses a required member "exact"`)
	// A cell with an unknown member refuses naming the key.
	mutated = tampered(t, sealed, func(document map[string]any) {
		document["event_kind_counts"].(map[string]any)["usage"].(map[string]any)["frobnicate"] = 1
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `event_kind_counts["usage"] carries unknown member "frobnicate"`)
	// Build side: missing key, extra key, and a +1 cell refuse.
	input := validTargetInput(rows...)
	delete(input.EventKindCounts, "usage")
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, `event_kind_counts miss key "usage"`)
	input = validTargetInput(rows...)
	input.EventKindCounts["frobnicate"] = clonefidelity.FidelityCounts{}
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, `event_kind_counts carry unknown key "frobnicate"`)
	input = validTargetInput(rows...)
	bumped := input.EventKindCounts[first]
	bumped.Exact++
	input.EventKindCounts[first] = bumped
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "event_kind_counts reconciles")
	// Build side refuses a breakdown value above uint53.
	input = validTargetInput(rows...)
	big := input.EventKindCounts[first]
	big.Exact = 1 << 53
	input.EventKindCounts[first] = big
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "exceeds uint53")
}

// TestContentBlockBreakdown pins the closed 8-type map the same way:
// exact keys, and a +1 perturbation in any of the 56 cells refuses,
// as does a -1 on every non-zero cell.
func TestContentBlockBreakdown(t *testing.T) {
	types := []string{"text", "json", "image", "audio", "document", "resource_link", "redacted", "opaque"}
	if len(types) != 8 {
		t.Fatalf("oracle type count = %d, want 8", len(types))
	}
	rows := sevenDispositionRows()
	sealed := mustBuild(t, validTargetInput(rows...))
	report := mustDecode(t, sealed)
	if len(report.ContentBlockCounts) != 8 {
		t.Fatalf("content_block_counts keys = %d, want 8", len(report.ContentBlockCounts))
	}
	dispositions := []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"}
	first := types[0]
	perturbed := 0
	for _, block := range types {
		for _, disposition := range dispositions {
			mutated := tampered(t, sealed, func(document map[string]any) {
				cell := document["content_block_counts"].(map[string]any)[block].(map[string]any)
				cell[disposition] = cellValue(cell[disposition]) + 1
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, "content_block_counts reconciles")
			perturbed++
		}
	}
	if perturbed != 56 {
		t.Fatalf("perturbed cells = %d, want 56", perturbed)
	}
	for _, disposition := range dispositions {
		mutated := tampered(t, sealed, func(document map[string]any) {
			cell := document["content_block_counts"].(map[string]any)[first].(map[string]any)
			cell[disposition] = cellValue(cell[disposition]) - 1
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, "content_block_counts reconciles")
	}
	mutated := tampered(t, sealed, func(document map[string]any) {
		document["content_block_counts"].(map[string]any)["frobnicate"] = map[string]any{}
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `content_block_counts carry unknown key "frobnicate"`)
	mutated = tampered(t, sealed, func(document map[string]any) {
		delete(document["content_block_counts"].(map[string]any), "text")
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `content_block_counts miss key "text"`)
	input := validTargetInput(rows...)
	bumped := input.ContentBlockCounts[first]
	bumped.Semantic++
	input.ContentBlockCounts[first] = bumped
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "content_block_counts reconciles")
}

// TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis
// pins the closed seven-disposition byte map shape (exact keys with
// uint53 values) and witnesses the EXPLICIT STATED BOUND that its
// values are not reconciled to the rows: two reports over identical
// rows with different byte values both admit, and this test keeps
// asserting that admission so a future reconciliation change reddens
// it deliberately.
//
// Pinned SPEC v0.7.0 §13.14.2 lists byte_counts as a "Closed map of
// every disposition to uint53" (line 10636) under the rule
// "Aggregate maps reconcile exactly to the rows" (line 10645), but
// FidelityDispositionRecord carries no byte measure. The only byte
// measure in §13.14 is CaptureItem.byte_count in §13.14.1 (line
// 10471), on the Capture Manifest the report names by
// capture_manifest_id. Rows are capture items, canonical events and
// synthesized rows, and the spec defines no byte basis for the latter
// two. Verifying byte reconciliation therefore requires the Capture
// Manifest plus a spec-defined byte basis for canonical-event and
// synthesized rows — a spec gap, not an implementation defect.
// Owner: spec clarification (agent-session-manager-spec) + the
// §13.14.1 capture-manifest leaf.
func TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis(t *testing.T) {
	rows := sevenDispositionRows()
	input := validTargetInput(rows...)
	input.ByteCounts = map[string]uint64{
		"exact": 100, "semantic": 200, "summarized": 300, "opaque_preserved": 400,
		"synthesized": 500, "omitted": 600, "unrecoverable": 9007199254740991,
	}
	report := mustDecode(t, mustBuild(t, input))
	if report.ByteCounts["unrecoverable"] != 9007199254740991 || report.ByteCounts["exact"] != 100 {
		t.Fatalf("byte_counts = %v, want the caller values kept", report.ByteCounts)
	}
	// Same rows, different bytes: still admits (the stated bound).
	input.ByteCounts["exact"] = 101
	mustDecode(t, mustBuild(t, input))
	sealed := mustBuild(t, validTargetInput(rows...))
	mutated := tampered(t, sealed, func(document map[string]any) {
		document["byte_counts"].(map[string]any)["frobnicate"] = 1
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `byte_counts carry unknown disposition "frobnicate"`)
	mutated = tampered(t, sealed, func(document map[string]any) {
		delete(document["byte_counts"].(map[string]any), "exact")
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `byte_counts miss disposition "exact"`)
	badValues := map[string]any{
		"above_uint53": 9007199254740992, "float": 1.5, "negative": -1,
		"string": "100", "bool": true, "null": nil,
	}
	for name, value := range badValues {
		t.Run("decode/"+name, func(t *testing.T) {
			mutated := tampered(t, sealed, func(document map[string]any) {
				document["byte_counts"].(map[string]any)["exact"] = value
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, `byte_counts["exact"] is not a uint53`)
		})
	}
	// Build side: missing key, extra key, and above-uint53 refuse.
	build := validTargetInput(rows...)
	delete(build.ByteCounts, "exact")
	_, err = clonefidelity.BuildFidelityReport(build)
	requireRefusal(t, err, `byte_counts miss disposition "exact"`)
	build = validTargetInput(rows...)
	build.ByteCounts["frobnicate"] = 1
	_, err = clonefidelity.BuildFidelityReport(build)
	requireRefusal(t, err, `byte_counts carry unknown disposition "frobnicate"`)
	build = validTargetInput(rows...)
	build.ByteCounts["exact"] = 1 << 53
	_, err = clonefidelity.BuildFidelityReport(build)
	requireRefusal(t, err, `byte_counts["exact"] exceeds uint53`)
}

// TestSpreadBreakdownAdmits proves the breakdown reconciliation is a
// sum, not a shape: the same totals spread over several keys admit
// at both entries.
func TestSpreadBreakdownAdmits(t *testing.T) {
	rows := sevenDispositionRows()
	input := validTargetInput(rows...)
	spread := map[string]clonefidelity.FidelityCounts{}
	for _, kind := range clonebundle.EventKinds() {
		spread[kind] = clonefidelity.FidelityCounts{}
	}
	spread["user_message"] = clonefidelity.FidelityCounts{Exact: 1, Semantic: 1}
	spread["assistant_message"] = clonefidelity.FidelityCounts{Exact: 1, Summarized: 1}
	spread["usage"] = clonefidelity.FidelityCounts{OpaquePreserved: 1, Synthesized: 1, Omitted: 1, Unrecoverable: 1}
	input.EventKindCounts = spread
	report := mustDecode(t, mustBuild(t, input))
	if report.EventKindCounts["usage"].Omitted != 1 {
		t.Fatalf("spread breakdown did not survive: %+v", report.EventKindCounts["usage"])
	}
}

// cellValue reads a tampered-document number cell as uint64.
// Tampered documents decode with UseNumber, so every cell is a
// json.Number literal.
func cellValue(value any) uint64 {
	number, ok := value.(json.Number)
	if !ok {
		panic(fmt.Sprintf("unexpected cell type %T", value))
	}
	integer, err := number.Int64()
	if err != nil {
		panic(fmt.Sprintf("cell %q is not an integer: %v", number.String(), err))
	}
	return uint64(integer)
}
