package clonefidelity_test

import (
	"strings"
	"testing"

	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/environ"
)

// Closed-vocabulary oracles. Every list is retyped from the pinned
// SPEC v0.7.0 Section 13.14.2 text (lines cited per test), never
// imported from production. Every gate is driven through the
// production entries: the registry predicates plus Build and Decode
// where the vocabulary is used. Refusals assert the stable code
// plus a literal detail.

// TestDispositionVocabularyOracle sweeps the seven dispositions from
// SPEC v0.7.0 line 10550 ("exact|semantic|summarized|
// opaque_preserved|synthesized|omitted|unrecoverable"): every member
// is admitted by the registry and round-trips through both entries;
// every other token is refused with a literal code.
func TestDispositionVocabularyOracle(t *testing.T) {
	members := []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"}
	refusals := []string{
		"", "Exact", "EXACT", "eXaCt", " exact", "exact ", "exact\n", "\texact",
		"exac", "exactx", "ex-act", "exact_", "xact", "semantics", "synthesize",
		"omited", "unrecoverabl", "opaque-preserved", "opaque",
		"archive_only", "unknown_native_event", "null", "7", "true",
	}
	if len(members) != 7 {
		t.Fatalf("oracle member count = %d, want 7", len(members))
	}
	for _, disposition := range members {
		if !clonefidelity.ValidDisposition(disposition) {
			t.Errorf("ValidDisposition(%q) = false, want true", disposition)
		}
		row := validRowInput("vocab-"+disposition, disposition)
		if disposition == "synthesized" {
			row.CanonicalObjectID = nil
		}
		sealed := mustBuild(t, validTargetInput(row))
		report := mustDecode(t, sealed)
		if report.Rows[0].Disposition != disposition {
			t.Errorf("round-trip disposition = %q, want %q", report.Rows[0].Disposition, disposition)
		}
	}
	literal := "outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable"
	for _, token := range refusals {
		if clonefidelity.ValidDisposition(token) {
			t.Errorf("ValidDisposition(%q) = true, want false", token)
		}
		row := validRowInput("vocab-refuse", "semantic")
		row.Disposition = token
		_, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
		requireRefusal(t, err, literal)
	}
	// Non-string JSON values refuse at the decode gate with the same
	// literal code.
	sealed := mustBuild(t, validTargetInput(validRowInput("vocab-nonstring", "semantic")))
	nonStrings := []any{7, true, nil, []any{"exact"}, map[string]any{"d": "exact"}}
	for _, value := range nonStrings {
		mutated := tampered(t, sealed, func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { row["disposition"] = value })
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, literal)
	}
	// Decode-side string refusals pin the decode gate too (the shared
	// predicate alone would leave the call site unmeasured).
	for _, token := range refusals {
		mutated := tampered(t, sealed, func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { row["disposition"] = token })
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, literal)
	}
}

// TestProfileVocabularyOracle sweeps the five profiles from SPEC
// v0.7.0 line 10562 ("strict_exact|maximal_safe|compact|
// messages_only|archive_only"): every member is admitted by the
// registry; every other token is refused with a literal code at both
// entries. Scope coupling (archive_only exactly for archive) is
// pinned by the branch matrix; here target-scope members
// round-trip and archive_only round-trips in archive scope.
func TestProfileVocabularyOracle(t *testing.T) {
	members := []string{"strict_exact", "maximal_safe", "compact", "messages_only", "archive_only"}
	refusals := []string{
		"", "Maximal_Safe", "MAXIMAL_SAFE", " maximal_safe", "maximal_safe ",
		"maximal-safe", "maximal", "strict", "compactx", "messagesonly",
		"archiveonly", "archive_only ", "Archive_Only", "exact", "null", "5",
	}
	if len(members) != 5 {
		t.Fatalf("oracle member count = %d, want 5", len(members))
	}
	row := validRowInput("vocab-profile", "exact")
	for _, profile := range members {
		if !clonefidelity.ValidProfile(profile) {
			t.Errorf("ValidProfile(%q) = false, want true", profile)
		}
		input := validTargetInput(row)
		if profile == "archive_only" {
			input = validArchiveInput(row)
		} else {
			input.Profile = profile
		}
		sealed := mustBuild(t, input)
		report := mustDecode(t, sealed)
		if report.Profile != profile {
			t.Errorf("round-trip profile = %q, want %q", report.Profile, profile)
		}
	}
	literal := "outside strict_exact|maximal_safe|compact|messages_only|archive_only"
	for _, token := range refusals {
		if clonefidelity.ValidProfile(token) {
			t.Errorf("ValidProfile(%q) = true, want false", token)
		}
		input := validTargetInput(row)
		input.Profile = token
		_, err := clonefidelity.BuildFidelityReport(input)
		requireRefusal(t, err, literal)
	}
	sealed := mustBuild(t, validTargetInput(row))
	for _, token := range refusals {
		mutated := tampered(t, sealed, func(document map[string]any) { document["profile"] = token })
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, literal)
	}
	nonStrings := map[string]any{
		"number": 7, "bool": true, "null": nil,
		"array": []any{"maximal_safe"}, "object": map[string]any{"p": "maximal_safe"},
	}
	for _, value := range nonStrings {
		mutated := tampered(t, sealed, func(document map[string]any) { document["profile"] = value })
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, literal)
	}
}

// TestStrategyVocabularyOracle sweeps the five strategies from SPEC
// v0.7.0 lines 10563-10564 ("same_environment_native_rewrite|
// target_native_writer|target_official_import|continuation_context|
// archive_only"). No report field carries a strategy (Projection
// Plan is sibling scope), so the registry predicate is the
// production entry and the oracle drives it directly.
func TestStrategyVocabularyOracle(t *testing.T) {
	members := []string{
		"same_environment_native_rewrite", "target_native_writer",
		"target_official_import", "continuation_context", "archive_only",
	}
	refusals := []string{
		"", "Archive_Only", "ARCHIVE_ONLY", " archive_only", "archive_only ",
		"archive-only", "continuation-context", "continuation", "target_native_write",
		"target_native_writer ", "same-environment-native-rewrite", "same_environment",
		"official_import", "maximal_safe", "null",
	}
	if len(members) != 5 {
		t.Fatalf("oracle member count = %d, want 5", len(members))
	}
	for _, strategy := range members {
		if !clonefidelity.ValidStrategy(strategy) {
			t.Errorf("ValidStrategy(%q) = false, want true", strategy)
		}
	}
	for _, token := range refusals {
		if clonefidelity.ValidStrategy(token) {
			t.Errorf("ValidStrategy(%q) = true, want false", token)
		}
	}
	if got := clonefidelity.Strategies(); len(got) != 5 {
		t.Fatalf("Strategies() count = %d, want 5", len(got))
	}
}

// TestReasonVocabularyOracle sweeps the nineteen closed core reasons
// from SPEC v0.7.0 lines 10551-10553 plus reverse-DNS extensions:
// every core code and every well-formed extension is admitted at
// both entries; case variants, whitespace, near-misses, empty,
// non-strings, and malformed reverse-DNS are refused with a literal
// code. Core/extension routing and the no-shadow rule are pinned
// explicitly.
func TestReasonVocabularyOracle(t *testing.T) {
	core := []string{
		"target_no_equivalent", "source_not_persisted", "source_truncated",
		"source_corrupt", "foreign_encrypted_payload", "foreign_signature_unverifiable",
		"target_schema_constraint", "target_context_limit", "target_size_limit",
		"target_version_gate", "official_importer_loss", "graph_flattened",
		"unsafe_pending_action", "credential_excluded", "secret_policy",
		"operator_policy", "unsupported_media_type", "unknown_native_event",
		"derived_index_rebuilt",
	}
	extensions := []string{
		"a.b", "com.example.reason", "x1.y-2.z3",
		"com.example.target-no-equivalent", "org.ax.custom-reason.v2",
	}
	// Note: "a.b-" is ADMITTED (the owner grammar allows a trailing
	// hyphen inside a label; only the first character is
	// constrained), so it is absent here; the environ suite pins
	// that grammar. Underscores never admit: no DNS label carries
	// one, which is exactly why no core code shadows.
	refusals := []string{
		"Unknown_Native_Event", "UNKNOWN_NATIVE_EVENT",
		" unknown_native_event", "unknown_native_event ", "unknown_native_event\n",
		"unknown_native_even", "unknown_native_eventx", "unknown-native-event",
		"unknown", "exact", "nodots", "a", "ab", "A.b", ".a.b", "a.", "a..b",
		"-a.b", "1a.b", "a.1b", "a b.c", "a.b ", " a.b",
		"com.example." + strings.Repeat("l", 64), "null", "7",
		"com.example.target_no_equivalent",
	}
	if len(core) != 19 {
		t.Fatalf("oracle core count = %d, want 19", len(core))
	}
	// Registry routing: core codes route core, extensions route
	// extension, and no core code is reverse-DNS-shaped, so an
	// extension can never equal or shadow a core code.
	for _, code := range core {
		if !clonefidelity.IsCoreReasonCode(code) {
			t.Errorf("IsCoreReasonCode(%q) = false, want true", code)
		}
		if !clonefidelity.ValidReasonCode(code) {
			t.Errorf("ValidReasonCode(%q) = false, want true", code)
		}
		if environ.CheckReverseDNS(code) {
			t.Errorf("core code %q is reverse-DNS-shaped: extensions could shadow it", code)
		}
	}
	if got := clonefidelity.CoreReasonCodes(); len(got) != 19 {
		t.Fatalf("CoreReasonCodes() count = %d, want 19", len(got))
	}
	for _, code := range extensions {
		if clonefidelity.IsCoreReasonCode(code) {
			t.Errorf("IsCoreReasonCode(%q) = true, want false", code)
		}
		if !clonefidelity.ValidReasonCode(code) {
			t.Errorf("ValidReasonCode(%q) = false, want true", code)
		}
	}
	for _, token := range refusals {
		if clonefidelity.ValidReasonCode(token) {
			t.Errorf("ValidReasonCode(%q) = true, want false", token)
		}
	}
	// The empty string refuses everywhere: by shape at the registry
	// (no core code is empty and reverse-DNS needs three characters)
	// and by the item bound at both JSON entries.
	if clonefidelity.ValidReasonCode("") {
		t.Error(`ValidReasonCode("") = true, want false`)
	}
	emptyRow := validRowInput("vocab-reason-empty", "semantic", "")
	_, err := clonefidelity.BuildFidelityReport(validTargetInput(emptyRow))
	requireRefusal(t, err, "reason_codes[0] is not a string[1..128]")
	// Entry admission: every core code and every extension seals and
	// decodes on a semantic row at both entries.
	for _, code := range append(append([]string(nil), core...), extensions...) {
		row := validRowInput("vocab-reason", "semantic", code)
		sealed := mustBuild(t, validTargetInput(row))
		report := mustDecode(t, sealed)
		if len(report.Rows[0].ReasonCodes) != 1 || report.Rows[0].ReasonCodes[0] != code {
			t.Errorf("round-trip reasons = %q, want [%q]", report.Rows[0].ReasonCodes, code)
		}
		if report.ReasonCounts[code] != 1 {
			t.Errorf("round-trip reason_counts[%q] = %d, want 1", code, report.ReasonCounts[code])
		}
	}
	// Entry refusal with literal codes at both entries.
	buildLiteral := "outside the core and reverse-DNS reason vocabulary"
	for _, token := range refusals {
		row := validRowInput("vocab-reason-refuse", "semantic", token)
		_, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
		requireRefusal(t, err, buildLiteral)
	}
	sealed := mustBuild(t, validTargetInput(validRowInput("vocab-reason-decode", "semantic")))
	emptyMutated := tampered(t, sealed, func(document map[string]any) {
		tamperRow(document, 0, func(row map[string]any) { row["reason_codes"] = []any{""} })
	})
	_, err = clonefidelity.DecodeFidelityReport(emptyMutated)
	requireRefusal(t, err, "reason_codes are not sorted unique string[1..128][0..128]")
	decodeLiteral := "outside the core and reverse-DNS reason vocabulary"
	for _, token := range refusals {
		mutated := tampered(t, sealed, func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { row["reason_codes"] = []any{token} })
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, decodeLiteral)
	}
	// Non-string reason items refuse at the decode shape gate.
	nonStrings := []any{7, true, nil, []any{"exact"}, map[string]any{"r": "x"}}
	for _, value := range nonStrings {
		mutated := tampered(t, sealed, func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { row["reason_codes"] = []any{value} })
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, "reason_codes are not sorted unique string[1..128][0..128]")
	}
}

// TestReasonShadowImpossible pins the no-redefine rule from the
// extension side: a reverse-DNS string containing core text is a
// distinct extension (admitted, never core), and every core code is
// admitted through the core arm at both entries even though no
// extension string can spell it.
func TestReasonShadowImpossible(t *testing.T) {
	shadow := "com.example.target-no-equivalent"
	if !clonefidelity.ValidReasonCode(shadow) {
		t.Fatalf("ValidReasonCode(%q) = false, want true", shadow)
	}
	if clonefidelity.IsCoreReasonCode(shadow) {
		t.Fatalf("IsCoreReasonCode(%q) = true, want false", shadow)
	}
	row := validRowInput("shadow-row", "semantic", shadow)
	report := mustDecode(t, mustBuild(t, validTargetInput(row)))
	if report.Rows[0].ReasonCodes[0] != shadow {
		t.Fatalf("round-trip reasons = %q, want [%q]", report.Rows[0].ReasonCodes, shadow)
	}
	core := "target_no_equivalent"
	if !clonefidelity.IsCoreReasonCode(core) || !clonefidelity.ValidReasonCode(core) {
		t.Fatalf("core code %q lost its core routing", core)
	}
}
