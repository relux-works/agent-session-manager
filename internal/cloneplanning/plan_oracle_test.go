package cloneplanning

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// This file drives PlanItem through the production entry against an
// independent oracle written from the derivation in TRACEABILITY.md.
// The oracle is organized class-group-major with per-group
// specification quotes; production is a strategy-arm-first gate
// chain. Every expectation is a literal retyped here, never a
// production constant (production constants are unexported, so the
// oracle cannot reference them even by accident). Domains iterate
// the owner vocabularies; the census test pins the group partition
// so an owner-table addition breaks loudly.

// oracleCell is one expected PlanItem outcome: either a mapping
// with sorted literal reasons, or a refusal naming the expected
// message token.
type oracleCell struct {
	disposition string
	reasons     []string
	refuseToken string
}

// oraclePlan decides the expected cell from the derivation rules.
// Group R-ARCH (SPEC.v0.7.0.md:10663-10665, 3841, 10629-10631):
// archive strategy or profile refuses; the archive branch carries
// no plan.
func oraclePlan(strategy, profile string, item Item) oracleCell {
	if strategy == "archive_only" || profile == "archive_only" {
		return oracleCell{refuseToken: "archive branch"}
	}
	disposition, reasons := oracleBase(strategy, item)
	return oracleOverlay(profile, item, disposition, reasons)
}

// oracleBase decides the strategy arm per class group. Groups
// quote their forcing specification lines; leaf-pinned arms say
// so. Reasons are returned sorted.
func oracleBase(strategy string, item Item) (string, []string) {
	// R-OPAQUE-R (10392: "foreign encrypted/signed reasoning is
	// opaque-preserved"; 10552-10553: the two foreign reasons).
	if item.Kind == "opaque_reasoning" {
		if item.Protection == "signed" {
			return "opaque_preserved", []string{"foreign_signature_unverifiable"}
		}
		return "opaque_preserved", []string{"foreign_encrypted_payload"}
	}
	// R-OPAQUE-E (10390: "Unknown native records become
	// raw-addressable opaque events"; 10561: unknown_native_event).
	// Opaque blocks inherit the unknown-content cause: blocks
	// carry no protection fact (leaf-pinned).
	if item.Kind == "opaque_event" || item.Block == "opaque" {
		return "opaque_preserved", []string{"unknown_native_event"}
	}
	// R-ABORT (10391: "incomplete calls become aborted history";
	// unsafe_pending_action 10558). Aborted history preserves
	// (summarized, never omitted) without executing.
	if item.Kind == "tool_call" || item.Kind == "tool_result" {
		if item.Resolution == "aborted" {
			return "summarized", []string{"unsafe_pending_action"}
		}
	}
	// R-CONT (10565: "Continuation context is explicitly
	// non-native historical fidelity"): exact is forbidden, history
	// keeps semantic meaning without a native equivalent.
	if strategy == "continuation_context" {
		return "semantic", []string{"target_no_equivalent"}
	}
	// R-IMPORT-TOOL (official_importer_loss 10557; the
	// official_import capability 3888): the vendor importer keeps
	// native tool history lossy.
	if strategy == "target_official_import" {
		if item.Kind == "tool_call" || item.Kind == "tool_result" || item.Kind == "tool_definition_snapshot" {
			return "semantic", []string{"official_importer_loss"}
		}
		// R-IMPORT-GRAPH (graph_flattened 10558): the subagent
		// graph does not survive official import.
		if item.Kind == "subagent_started" || item.Kind == "subagent_completed" {
			return "semantic", []string{"graph_flattened"}
		}
	}
	// R-DEFAULT (leaf-pinned): native strategies reproduce bytes.
	return "exact", nil
}

// oracleOverlay refines the strategy cell under the profile.
func oracleOverlay(profile string, item Item, disposition string, reasons []string) oracleCell {
	switch profile {
	case "strict_exact":
		// R-STRICT (leaf-pinned from the profile name with
		// 10560 "An aggregate score cannot replace item gates"
		// and AC-CLONE-001 item-level fidelity): any non-exact
		// item refuses the plan.
		if disposition != "exact" {
			return oracleCell{refuseToken: "strict_exact"}
		}
		return oracleCell{disposition: disposition, reasons: reasons}
	case "compact":
		// R-COMPACT (leaf-pinned from the profile name with
		// target_context_limit 10555): semantic clamps to
		// summarized, keeping and extending the reason set.
		if disposition == "semantic" {
			return oracleCell{disposition: "summarized", reasons: sortedCopy(append(append([]string(nil), reasons...), "target_context_limit"))}
		}
		return oracleCell{disposition: disposition, reasons: reasons}
	case "messages_only":
		// R-MSG (leaf-pinned from the profile name with
		// operator_policy and unsupported_media_type 10560):
		// non-message classes omit, keeping and extending the
		// reason set; media blocks name their cause too.
		if isOracleMessageClass(item) {
			return oracleCell{disposition: disposition, reasons: reasons}
		}
		omitted := append(append([]string(nil), reasons...), "operator_policy")
		if item.Block == "image" || item.Block == "audio" || item.Block == "document" || item.Block == "resource_link" {
			omitted = append(omitted, "unsupported_media_type")
		}
		return oracleCell{disposition: "omitted", reasons: sortedCopy(omitted)}
	case "maximal_safe":
		return oracleCell{disposition: disposition, reasons: reasons}
	default:
		return oracleCell{refuseToken: "profile"}
	}
}

// isOracleMessageClass mirrors the messages_only survivor set from
// literals: user/assistant message events and text, structured,
// and redacted blocks.
func isOracleMessageClass(item Item) bool {
	if item.Kind != "" {
		return item.Kind == "user_message" || item.Kind == "assistant_message"
	}
	return item.Block == "text" || item.Block == "json" || item.Block == "redacted"
}

// checkOracleCell drives one cell through PlanItem and asserts the
// oracle outcome with literal comparisons. It reports whether the
// cell admitted, so callers skip admitted-only assertions on
// refusals.
func checkOracleCell(t *testing.T, strategy, profile string, item Item) (Mapping, bool) {
	t.Helper()
	want := oraclePlan(strategy, profile, item)
	got, err := PlanItem(strategy, profile, item)
	label := strategy + "/" + profile + "/" + describeItem(item) + "/" + item.Resolution + item.Protection
	if want.refuseToken != "" {
		if err == nil {
			t.Fatalf("%s: admitted, want refusal naming %q", label, want.refuseToken)
		}
		if !strings.Contains(err.Error(), want.refuseToken) {
			t.Fatalf("%s: refusal %q does not name %q", label, err.Error(), want.refuseToken)
		}
		return Mapping{}, false
	}
	if err != nil {
		t.Fatalf("%s: refused: %v", label, err)
	}
	if got.Disposition != want.disposition {
		t.Fatalf("%s: disposition %q, want %q", label, got.Disposition, want.disposition)
	}
	if !equalReasons(got.Reasons, want.reasons) {
		t.Fatalf("%s: reasons %q, want %q", label, got.Reasons, want.reasons)
	}
	return got, true
}

func TestPlanItemWholeDomainOracle(t *testing.T) {
	items := allValidItems()
	cells := 0
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range items {
				got, admitted := checkOracleCell(t, strategy, profile, item)
				if !admitted {
					cells++
					continue
				}
				label := strategy + "/" + profile + "/" + describeItem(item)
				// R-NEVER: synthesized rows have no source
				// object (10605) and planned canonical items
				// exist, so neither synthesized nor
				// unrecoverable ever emits.
				if got.Disposition == "synthesized" || got.Disposition == "unrecoverable" {
					t.Fatalf("%s: emitted %q", label, got.Disposition)
				}
				// Exact carries no reason (10605: "Exact
				// requires an empty reason set").
				if got.Disposition == "exact" && len(got.Reasons) != 0 {
					t.Fatalf("%s: exact with reasons %q", label, got.Reasons)
				}
				// Every other disposition names a reason
				// (10605).
				if got.Disposition != "exact" && len(got.Reasons) == 0 {
					t.Fatalf("%s: %q with empty reasons", label, got.Disposition)
				}
				cells++
			}
		}
	}
	t.Logf("whole-domain oracle: %d cells", cells)
}

func TestPlanItemArchiveRule(t *testing.T) {
	items := allValidItems()
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			if strategy != "archive_only" && profile != "archive_only" {
				continue
			}
			for _, item := range items {
				checkOracleCell(t, strategy, profile, item)
			}
		}
	}
}

func TestPlanItemOpaqueRule(t *testing.T) {
	items := []Item{
		{Kind: "opaque_reasoning", Protection: "encrypted"},
		{Kind: "opaque_reasoning", Protection: "signed"},
		{Kind: "opaque_event"},
		{Kind: "opaque_event", Protection: "encrypted"},
		{Kind: "opaque_event", Protection: "signed"},
		{Block: "opaque"},
	}
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range items {
				checkOracleCell(t, strategy, profile, item)
			}
		}
	}
}

func TestPlanItemAbortRule(t *testing.T) {
	items := []Item{
		{Kind: "tool_call", Resolution: "aborted"},
		{Kind: "tool_result", Resolution: "aborted"},
	}
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range items {
				checkOracleCell(t, strategy, profile, item)
			}
		}
	}
}

func TestPlanItemContinuationRule(t *testing.T) {
	for _, profile := range clonefidelity.Profiles() {
		for _, item := range allValidItems() {
			got, admitted := checkOracleCell(t, "continuation_context", profile, item)
			if !admitted {
				continue
			}
			// R-CONT pins the clamp structurally: exact
			// never emits under continuation.
			if got.Disposition == "exact" {
				t.Fatalf("continuation_context/%s/%s: emitted exact", profile, describeItem(item))
			}
		}
	}
}

func TestPlanItemImporterRule(t *testing.T) {
	for _, profile := range clonefidelity.Profiles() {
		for _, item := range allValidItems() {
			checkOracleCell(t, "target_official_import", profile, item)
		}
	}
}

func TestPlanItemNativeRule(t *testing.T) {
	for _, strategy := range []string{"same_environment_native_rewrite", "target_native_writer"} {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range allValidItems() {
				checkOracleCell(t, strategy, profile, item)
			}
		}
	}
}

func TestPlanItemStrictRule(t *testing.T) {
	for _, strategy := range clonefidelity.Strategies() {
		for _, item := range allValidItems() {
			checkOracleCell(t, strategy, "strict_exact", item)
		}
	}
}

func TestPlanItemCompactRule(t *testing.T) {
	for _, strategy := range clonefidelity.Strategies() {
		for _, item := range allValidItems() {
			got, admitted := checkOracleCell(t, strategy, "compact", item)
			// R-COMPACT pins the clamp structurally:
			// semantic never survives compact.
			if !admitted {
				continue
			}
			if got.Disposition == "semantic" {
				t.Fatalf("compact/%s/%s: emitted semantic", strategy, describeItem(item))
			}
		}
	}
}

func TestPlanItemMessagesRule(t *testing.T) {
	for _, strategy := range clonefidelity.Strategies() {
		for _, item := range allValidItems() {
			got, admitted := checkOracleCell(t, strategy, "messages_only", item)
			if !admitted {
				continue
			}
			// R-MSG pins the survivor set structurally:
			// non-message classes always omit.
			if !isOracleMessageClass(item) && got.Disposition != "omitted" {
				t.Fatalf("messages_only/%s/%s: disposition %q, want omitted", strategy, describeItem(item), got.Disposition)
			}
		}
	}
}

func TestPlanItemGroupCensus(t *testing.T) {
	// The oracle partition over the owner vocabularies, retyped
	// from SPEC.v0.7.0.md:10385 (26 event kinds), 10386-10388 (8
	// content blocks), 10562-10565 (5 profiles, 5 strategies). An
	// owner-table addition breaks here instead of sliding into a
	// default arm.
	wantKinds := []string{
		"session_started", "instruction_snapshot", "user_message", "assistant_message",
		"reasoning_summary", "opaque_reasoning", "tool_definition_snapshot", "tool_call",
		"tool_result", "approval_request", "approval_response", "plan_update", "progress",
		"usage", "rate_limit", "file_change", "compaction", "turn_started", "turn_completed",
		"turn_aborted", "subagent_started", "subagent_completed", "error", "migration_checkpoint",
		"session_finished", "opaque_event",
	}
	if got := clonebundle.EventKinds(); !equalStrings(got, wantKinds) {
		t.Fatalf("event kinds %q, want %q", got, wantKinds)
	}
	wantBlocks := []string{"text", "json", "image", "audio", "document", "resource_link", "redacted", "opaque"}
	if got := clonebundle.ContentBlockTypes(); !equalStrings(got, wantBlocks) {
		t.Fatalf("content blocks %q, want %q", got, wantBlocks)
	}
	wantStrategies := []string{"same_environment_native_rewrite", "target_native_writer", "target_official_import", "continuation_context", "archive_only"}
	if got := clonefidelity.Strategies(); !equalStrings(got, wantStrategies) {
		t.Fatalf("strategies %q, want %q", got, wantStrategies)
	}
	wantProfiles := []string{"strict_exact", "maximal_safe", "compact", "messages_only", "archive_only"}
	if got := clonefidelity.Profiles(); !equalStrings(got, wantProfiles) {
		t.Fatalf("profiles %q, want %q", got, wantProfiles)
	}
	if got := len(allValidItems()); got != 39 {
		t.Fatalf("valid items %d, want 39 (31 event + 8 block)", got)
	}
	// Every kind and every block type occurs in the item domain.
	seenKinds := map[string]bool{}
	seenBlocks := map[string]bool{}
	for _, item := range allValidItems() {
		seenKinds[item.Kind] = true
		seenBlocks[item.Block] = true
	}
	for _, kind := range wantKinds {
		if !seenKinds[kind] {
			t.Fatalf("kind %q missing from the item domain", kind)
		}
	}
	for _, block := range wantBlocks {
		if !seenBlocks[block] {
			t.Fatalf("block %q missing from the item domain", block)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPlanItemInvalidArms(t *testing.T) {
	validItem := Item{Kind: "user_message"}
	// Invalid strategies refuse through the vocabulary gate at a
	// fixed valid context; "bogus_strategy" is the narrowing
	// probe (N-strategy-vocab admits exactly it).
	for _, strategy := range []string{"", "bogus_strategy", "ARCHIVE_ONLY", "official_import", "continuation", "same_environment", "archive_only\xff"} {
		if _, err := PlanItem(strategy, "maximal_safe", validItem); err == nil {
			t.Fatalf("strategy %q admitted", strategy)
		} else if !strings.Contains(err.Error(), "strategy") {
			t.Fatalf("strategy %q refusal %q does not name the strategy", strategy, err.Error())
		}
	}
	// Invalid profiles refuse likewise; "bogus_profile" is the
	// narrowing probe (N-profile-vocab).
	for _, profile := range []string{"", "bogus_profile", "MAXIMAL_SAFE", "maximal", "archive", "strict", "messages_only\xff"} {
		if _, err := PlanItem("target_native_writer", profile, validItem); err == nil {
			t.Fatalf("profile %q admitted", profile)
		} else if !strings.Contains(err.Error(), "profile") {
			t.Fatalf("profile %q refusal %q does not name the profile", profile, err.Error())
		}
	}
	// Invalid items refuse at every strategy/profile context. The
	// archive contexts refuse with the archive token first: this
	// pins the gate order (archive before item check). Probes
	// "bogus_kind", "bogus_block", "bogus_resolution", and
	// "bogus_protection" are the narrowing probes
	// (N-kind-vocab, N-block-vocab, N-resolution-vocab,
	// N-protection-vocab).
	invalidItems := []Item{
		{Kind: "user_message", Block: "text"},
		{},
		{Kind: "bogus_kind"},
		{Kind: "USER_MESSAGE"},
		{Kind: "user_message\xff"},
		{Block: "bogus_block"},
		{Block: "TEXT"},
		{Kind: "user_message", Resolution: "completed"},
		{Kind: "user_message", Protection: "encrypted"},
		{Kind: "tool_call"},
		{Kind: "tool_call", Resolution: "bogus_resolution"},
		{Kind: "opaque_reasoning"},
		{Kind: "opaque_reasoning", Protection: "bogus_protection"},
		{Block: "text", Resolution: "completed"},
		{Block: "opaque", Protection: "encrypted"},
	}
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range invalidItems {
				_, err := PlanItem(strategy, profile, item)
				if err == nil {
					t.Fatalf("%s/%s/%+v admitted", strategy, profile, item)
				}
				if strategy == "archive_only" || profile == "archive_only" {
					if !strings.Contains(err.Error(), "archive branch") {
						t.Fatalf("%s/%s/%+v refusal %q does not name the archive branch first", strategy, profile, item, err.Error())
					}
					continue
				}
				if !strings.Contains(err.Error(), "projection item") {
					t.Fatalf("%s/%s/%+v refusal %q does not name the item", strategy, profile, item, err.Error())
				}
			}
		}
	}
}

func TestPlanItemAgreement(t *testing.T) {
	// Supplement to the literal oracle above: every emitted
	// disposition and reason stays inside the owner vocabularies.
	// This test alone would be circular (production values against
	// production predicates); the literal oracle is the pin.
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range allValidItems() {
				got, err := PlanItem(strategy, profile, item)
				if err != nil {
					continue
				}
				if !clonefidelity.ValidDisposition(got.Disposition) {
					t.Fatalf("%s/%s/%s: disposition %q outside the owner vocabulary", strategy, profile, describeItem(item), got.Disposition)
				}
				for _, reason := range got.Reasons {
					if !clonefidelity.IsCoreReasonCode(reason) {
						t.Fatalf("%s/%s/%s: reason %q outside the core codes", strategy, profile, describeItem(item), reason)
					}
				}
				for i := 1; i < len(got.Reasons); i++ {
					if got.Reasons[i-1] >= got.Reasons[i] {
						t.Fatalf("%s/%s/%s: reasons %q not sorted unique", strategy, profile, describeItem(item), got.Reasons)
					}
				}
			}
		}
	}
}
