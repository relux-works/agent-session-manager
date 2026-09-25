package cloneplanning

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file drives PlanSession and PlanTargetEffects through the
// production entries. Sessions fold to predicted counts derived
// from the mappings; every mapping the planner emits folds to zero
// target effects (no callable tools, no pending actions, no target
// token accounting).

// foldOracleCounts folds the independent oracle over a session,
// mirroring the production count derivation from literals.
func foldOracleCounts(t *testing.T, strategy, profile string, items []Item) []Mapping {
	t.Helper()
	mappings := make([]Mapping, 0, len(items))
	for _, item := range items {
		want := oraclePlan(strategy, profile, item)
		if want.refuseToken != "" {
			t.Fatalf("oracle refuses %s/%s/%s", strategy, profile, describeItem(item))
		}
		mappings = append(mappings, Mapping{Disposition: want.disposition, Reasons: want.reasons})
	}
	return mappings
}

func countDispositions(mappings []Mapping) clonefidelity.FidelityCounts {
	var counts clonefidelity.FidelityCounts
	for _, mapping := range mappings {
		switch mapping.Disposition {
		case "exact":
			counts.Exact++
		case "semantic":
			counts.Semantic++
		case "summarized":
			counts.Summarized++
		case "opaque_preserved":
			counts.OpaquePreserved++
		case "synthesized":
			counts.Synthesized++
		case "omitted":
			counts.Omitted++
		case "unrecoverable":
			counts.Unrecoverable++
		}
	}
	return counts
}

func TestPlanSessionCounts(t *testing.T) {
	sessions := []struct {
		name     string
		strategy string
		profile  string
		items    []Item
	}{
		{"single-exact", "target_native_writer", "maximal_safe", []Item{{Kind: "user_message"}}},
		{"mixed-native", "same_environment_native_rewrite", "maximal_safe", []Item{
			{Kind: "user_message"},
			{Kind: "tool_call", Resolution: "aborted"},
			{Kind: "opaque_reasoning", Protection: "signed"},
			{Block: "image"},
		}},
		{"mixed-continuation", "continuation_context", "compact", []Item{
			{Kind: "user_message"},
			{Kind: "assistant_message"},
			{Kind: "tool_call", Resolution: "completed"},
			{Kind: "usage"},
			{Block: "text"},
		}},
		{"mixed-importer", "target_official_import", "messages_only", []Item{
			{Kind: "user_message"},
			{Kind: "tool_call", Resolution: "completed"},
			{Kind: "subagent_completed"},
			{Kind: "usage"},
			{Block: "image"},
			{Block: "text"},
		}},
		{"strict-exact", "target_native_writer", "strict_exact", []Item{
			{Kind: "user_message"},
			{Block: "text"},
		}},
	}
	for _, session := range sessions {
		want := foldOracleCounts(t, session.strategy, session.profile, session.items)
		wantCounts := countDispositions(want)
		got, counts, err := PlanSession(session.strategy, session.profile, session.items)
		if err != nil {
			t.Fatalf("%s refused: %v", session.name, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s: %d mappings, want %d", session.name, len(got), len(want))
		}
		for i := range want {
			if got[i].Disposition != want[i].Disposition || !equalReasons(got[i].Reasons, want[i].Reasons) {
				t.Fatalf("%s mapping[%d] = %+v, want %+v", session.name, i, got[i], want[i])
			}
		}
		if counts != wantCounts {
			t.Fatalf("%s: counts %+v, want %+v", session.name, counts, wantCounts)
		}
		total := counts.Exact + counts.Semantic + counts.Summarized + counts.OpaquePreserved + counts.Synthesized + counts.Omitted + counts.Unrecoverable
		if total != uint64(len(session.items)) {
			t.Fatalf("%s: counts total %d, want %d", session.name, total, len(session.items))
		}
	}
}

func TestPlanSessionRefusals(t *testing.T) {
	item := Item{Kind: "user_message"}
	// Empty sessions refuse: the plan table requires
	// item_mappings[1..1000000].
	if _, _, err := PlanSession("target_native_writer", "maximal_safe", nil); err == nil {
		t.Fatalf("empty session admitted")
	}
	// Over-bound sessions refuse at exactly max+1 (the narrowing
	// probe N-session-bound admits exactly 1000001).
	many := make([]Item, 1000001)
	for i := range many {
		many[i] = item
	}
	if _, _, err := PlanSession("target_native_writer", "maximal_safe", many); err == nil {
		t.Fatalf("1000001-item session admitted")
	} else if !strings.Contains(err.Error(), "1000001") {
		t.Fatalf("over-bound refusal %q does not name the count", err.Error())
	}
	// The first refusing item refuses the session: strict_exact
	// with a non-exact member, and an invalid member mid-session.
	if _, _, err := PlanSession("target_native_writer", "strict_exact", []Item{item, {Kind: "tool_call", Resolution: "aborted"}}); err == nil {
		t.Fatalf("strict session with aborted tool admitted")
	} else if !strings.Contains(err.Error(), "strict_exact") {
		t.Fatalf("strict session refusal %q does not name strict_exact", err.Error())
	}
	if _, _, err := PlanSession("target_native_writer", "maximal_safe", []Item{item, {Kind: "bogus_kind"}}); err == nil {
		t.Fatalf("session with invalid member admitted")
	}
}

func TestPlanSessionMillionEdge(t *testing.T) {
	// The plan table's item_mappings[1..1000000] upper edge seals
	// at the production entry: one million identical items plan
	// and fold to Exact == 1000000.
	items := make([]Item, 1000000)
	for i := range items {
		items[i] = Item{Kind: "user_message"}
	}
	got, counts, err := PlanSession("target_native_writer", "maximal_safe", items)
	if err != nil {
		t.Fatalf("million-item session refused: %v", err)
	}
	if len(got) != 1000000 || counts.Exact != 1000000 {
		t.Fatalf("million-item session folds to %+v over %d mappings", counts, len(got))
	}
}

func TestPlanTargetEffectsZeroSweep(t *testing.T) {
	// Every mapping the planner emits over the whole domain folds
	// to zero target effects: no callable tools, no pending
	// actions, no target token accounting (R-INERT, R-ACCT).
	mappings := []Mapping{}
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range allValidItems() {
				got, err := PlanItem(strategy, profile, item)
				if err != nil {
					continue
				}
				mappings = append(mappings, got)
			}
		}
	}
	if len(mappings) == 0 {
		t.Fatalf("no admitted mappings to fold")
	}
	effects, err := PlanTargetEffects(mappings)
	if err != nil {
		t.Fatalf("PlanTargetEffects refused planned mappings: %v", err)
	}
	if len(effects.CallableTools) != 0 || len(effects.PendingActions) != 0 ||
		effects.TargetInputTokens != 0 || effects.TargetOutputTokens != 0 {
		t.Fatalf("planned mappings fold to %+v, want zero", effects)
	}
	// Per-disposition: each emitted disposition folds to zero
	// alone, so the N-effects-nonzero narrowing (exact funds one
	// token) fails on its cell.
	seen := map[string]Mapping{}
	for _, mapping := range mappings {
		seen[mapping.Disposition] = mapping
	}
	for disposition, mapping := range seen {
		effects, err := PlanTargetEffects([]Mapping{mapping})
		if err != nil {
			t.Fatalf("PlanTargetEffects(%s) refused: %v", disposition, err)
		}
		if len(effects.CallableTools) != 0 || len(effects.PendingActions) != 0 ||
			effects.TargetInputTokens != 0 || effects.TargetOutputTokens != 0 {
			t.Fatalf("PlanTargetEffects(%s) = %+v, want zero", disposition, effects)
		}
	}
	t.Logf("effects sweep: %d mappings, %d dispositions", len(mappings), len(seen))
}

func TestPlanTargetEffectsRefusals(t *testing.T) {
	good := Mapping{Disposition: "exact"}
	if _, err := PlanTargetEffects([]Mapping{good}); err != nil {
		t.Fatalf("exact mapping refused: %v", err)
	}
	if _, err := PlanTargetEffects(nil); err != nil {
		t.Fatalf("empty mapping list refused: %v", err)
	}
	// "bogus_disposition" and "bogus_reason" are the narrowing
	// probes (N-effects-disposition, N-effects-reason).
	for _, mapping := range []Mapping{
		{Disposition: "bogus_disposition"},
		{Disposition: ""},
		{Disposition: "exact", Reasons: []string{"bogus_reason"}},
		{Disposition: "exact", Reasons: []string{"unsafe_pending_action", "operator_policy"}},
		{Disposition: "exact", Reasons: []string{"operator_policy", "operator_policy"}},
	} {
		if _, err := PlanTargetEffects([]Mapping{good, mapping}); err == nil {
			t.Fatalf("mapping %+v admitted", mapping)
		}
	}
}

// Owner refusal literals, retyped from the record owner's refusal
// texts: the disposition vocabulary refusal and the reason-set
// coupling in internal/clonefidelity/record.go
// (checkRecordReasonRule), and the reason vocabulary, bounds, and
// sorted-unique refusals in internal/clonefidelity/decode.go
// (checkSortedUniqueReasonStrings). The coupling literals derive
// from SPEC.v0.7.0.md:10612-10614 ("Exact requires an empty reason
// set; every other disposition requires at least one reason").
// Every cell below asserts the owner's rule surfaces through
// PlanTargetEffects verbatim with the owner's sentinel matchable:
// swapping the delegation for a local re-implementation changes
// these literals and reddens here.
var ownerRecordLiterals = map[string]string{
	"disposition-vocabulary": "outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable",
	"exact-empty":            "want an empty reason set",
	"nonexact-reason":        "want at least one reason",
	"reason-vocabulary":      "is outside the core and reverse-DNS reason vocabulary",
	"reason-sorted-unique":   "reason_codes are not sorted unique",
	"reason-bounds":          "is not a string[1..128]",
	"reason-count":           "reason_codes, want [0..128]",
}

// requireOwnerRefusal asserts the delegated refusal: the entry
// refuses, both the package and the owner sentinels match, and the
// message carries the owner's literal rule text.
func requireOwnerRefusal(t *testing.T, label string, err error, literal string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: malformed mapping admitted", label)
	}
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("%s: refusal %q does not match the package sentinel", label, err.Error())
	}
	if !errors.Is(err, clonefidelity.ErrInvalid) {
		t.Fatalf("%s: refusal %q does not match the owner sentinel", label, err.Error())
	}
	if !strings.Contains(err.Error(), literal) {
		t.Fatalf("%s: refusal %q does not carry the owner literal %q", label, err.Error(), literal)
	}
}

func TestPlanTargetEffectsRefusesMalformedReasonSets(t *testing.T) {
	// Named regression for CR rev4 finding
	// effects-reason-set-coupling-missing: PlanTargetEffects
	// admitted a semantic mapping with no reason and an exact
	// mapping with an operator_policy reason. Both cells drive
	// the reviewer's exact inputs through the production entry
	// and assert the owner's coupling rule. Killers for
	// N-effects-exact-reasons and N-effects-nonexact-bare.
	_, err := PlanTargetEffects([]Mapping{{Disposition: "semantic"}})
	requireOwnerRefusal(t, "semantic-without-reasons", err, "want at least one reason")
	_, err = PlanTargetEffects([]Mapping{{Disposition: "exact", Reasons: []string{"operator_policy"}}})
	requireOwnerRefusal(t, "exact-with-operator-policy", err, "want an empty reason set")
}

// ownerRuleTableRow is one record-rule cell: a violating mapping
// through the censused entry (PlanTargetEffects is the only
// exported entry whose parameters carry a disposition or reason
// set; the census test pins that) with the owner's literal rule
// text.
type ownerRuleTableRow struct {
	rule    string
	mapping Mapping
	literal string
}

func ownerRuleTable() []ownerRuleTableRow {
	rows := []ownerRuleTableRow{
		{"disposition-vocabulary/bogus", Mapping{Disposition: "bogus_disposition"}, ownerRecordLiterals["disposition-vocabulary"]},
		{"disposition-vocabulary/empty", Mapping{Disposition: ""}, ownerRecordLiterals["disposition-vocabulary"]},
		{"exact-empty/operator-policy", Mapping{Disposition: "exact", Reasons: []string{"operator_policy"}}, ownerRecordLiterals["exact-empty"]},
		{"exact-empty/two-reasons", Mapping{Disposition: "exact", Reasons: []string{"operator_policy", "unknown_native_event"}}, ownerRecordLiterals["exact-empty"]},
		{"reason-vocabulary", Mapping{Disposition: "semantic", Reasons: []string{"bogus_reason"}}, ownerRecordLiterals["reason-vocabulary"]},
		{"reason-sorted-unique/unsorted", Mapping{Disposition: "semantic", Reasons: []string{"unsafe_pending_action", "operator_policy"}}, ownerRecordLiterals["reason-sorted-unique"]},
		{"reason-sorted-unique/duplicate", Mapping{Disposition: "semantic", Reasons: []string{"operator_policy", "operator_policy"}}, ownerRecordLiterals["reason-sorted-unique"]},
		{"reason-bounds/empty", Mapping{Disposition: "semantic", Reasons: []string{""}}, ownerRecordLiterals["reason-bounds"]},
		{"reason-bounds/long", Mapping{Disposition: "semantic", Reasons: []string{"ab." + strings.Repeat("c", 62) + "." + strings.Repeat("d", 63)}}, ownerRecordLiterals["reason-bounds"]},
	}
	// Every non-exact disposition without a reason: the coupling
	// rule names no exception, so each of the six dispositions
	// drives its own cell.
	for _, disposition := range []string{"semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"} {
		rows = append(rows, ownerRuleTableRow{
			rule:    "nonexact-reason/" + disposition,
			mapping: Mapping{Disposition: disposition},
			literal: ownerRecordLiterals["nonexact-reason"],
		})
	}
	// 129 sorted valid reasons: 19 core plus 110 reverse-DNS
	// extensions, over the owner's [0..128] count bound. The
	// extension labels are zero-padded so lexical order matches
	// numeric order before the final sort.
	many := append([]string(nil), clonefidelity.CoreReasonCodes()...)
	for i := 0; i < 110; i++ {
		many = append(many, fmt.Sprintf("com.example.reason.r%03d", i))
	}
	sort.Strings(many)
	if len(many) != 129 {
		panic("reason-count probe carries the wrong size")
	}
	rows = append(rows, ownerRuleTableRow{"reason-count/129", Mapping{Disposition: "semantic", Reasons: many}, ownerRecordLiterals["reason-count"]})
	return rows
}

func TestPlanTargetEffectsOwnerRecordRules(t *testing.T) {
	// The rule x entry table: every record rule the owner's
	// validator enforces on the disposition/reason members, each
	// driven as a violating input through PlanTargetEffects with
	// the owner's literal code asserted. Two owner rules are not
	// drivable here and are stated bounds, not silent waivers:
	// synthesized-requires-no-canonical and
	// non-synthesized-traces-to-evidence see only fixed
	// scaffolding (null canonical, derived evidence digest)
	// because the Mapping type carries neither member; the
	// scaffolding-shape test pins that construction and the
	// owner suite pins both arms of each rule. Killers for
	// N-effects-reason-bounds and N-effects-reason-count (the
	// other rows share killers with the refusals and regression
	// tests above).
	for _, row := range ownerRuleTable() {
		_, err := PlanTargetEffects([]Mapping{row.mapping})
		requireOwnerRefusal(t, row.rule, err, row.literal)
	}
	// Positive edges: the 128-count bound and the 128-character
	// reason bound admit, synthesized admits on its null
	// canonical, and a reverse-DNS reason admits.
	edge := append([]string(nil), clonefidelity.CoreReasonCodes()...)
	for i := 0; i < 109; i++ {
		edge = append(edge, fmt.Sprintf("com.example.reason.r%03d", i))
	}
	sort.Strings(edge)
	admitted := []Mapping{
		{Disposition: "exact"},
		{Disposition: "semantic", Reasons: []string{"operator_policy"}},
		{Disposition: "synthesized", Reasons: []string{"operator_policy"}},
		{Disposition: "omitted", Reasons: []string{"operator_policy", "unsupported_media_type"}},
		{Disposition: "unrecoverable", Reasons: []string{"com.example.reason.r000"}},
		{Disposition: "semantic", Reasons: edge},
		{Disposition: "semantic", Reasons: []string{"ab." + strings.Repeat("c", 62) + "." + strings.Repeat("d", 62)}},
	}
	if len(edge) != 128 {
		t.Fatalf("128-reason edge carries %d reasons", len(edge))
	}
	for i, mapping := range admitted {
		effects, err := PlanTargetEffects([]Mapping{mapping})
		if err != nil {
			t.Fatalf("admitted[%d] %+v refused: %v", i, mapping, err)
		}
		if len(effects.CallableTools) != 0 || len(effects.PendingActions) != 0 ||
			effects.TargetInputTokens != 0 || effects.TargetOutputTokens != 0 {
			t.Fatalf("admitted[%d] folds to %+v, want zero", i, effects)
		}
	}
}

func TestOwnerRuleTableProbeHygiene(t *testing.T) {
	// Fixture guard: every rule-table probe violates exactly the
	// rule its cell names, and every probe string is valid UTF-8
	// so the package pre-gate never fires ahead of the owner.
	// Without this, a cell could refuse for the wrong reason and
	// still satisfy its literal by check order alone.
	if len(ownerRecordLiterals) != 7 {
		t.Fatalf("owner literal table carries %d rules, want 7", len(ownerRecordLiterals))
	}
	rows := ownerRuleTable()
	if len(rows) != 16 {
		t.Fatalf("rule table carries %d rows, want 16", len(rows))
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.rule] {
			t.Fatalf("duplicate rule-table row %q", row.rule)
		}
		seen[row.rule] = true
		if !validText(row.mapping.Disposition) {
			t.Fatalf("%s: disposition probe is not valid UTF-8", row.rule)
		}
		for _, reason := range row.mapping.Reasons {
			if !validText(reason) {
				t.Fatalf("%s: reason probe is not valid UTF-8", row.rule)
			}
		}
		switch {
		case strings.HasPrefix(row.rule, "disposition-vocabulary/"):
			if clonefidelity.ValidDisposition(row.mapping.Disposition) {
				t.Fatalf("%s: disposition %q is valid", row.rule, row.mapping.Disposition)
			}
		case strings.HasPrefix(row.rule, "nonexact-reason/"):
			if !clonefidelity.ValidDisposition(row.mapping.Disposition) {
				t.Fatalf("%s: disposition %q is outside the vocabulary", row.rule, row.mapping.Disposition)
			}
			if len(row.mapping.Reasons) != 0 {
				t.Fatalf("%s: carries reasons", row.rule)
			}
		case strings.HasPrefix(row.rule, "exact-empty/"):
			for _, reason := range row.mapping.Reasons {
				if !clonefidelity.ValidReasonCode(reason) {
					t.Fatalf("%s: reason %q is outside the vocabulary", row.rule, reason)
				}
			}
			if !sort.StringsAreSorted(row.mapping.Reasons) {
				t.Fatalf("%s: reasons are not sorted", row.rule)
			}
		case row.rule == "reason-vocabulary":
			if clonefidelity.ValidReasonCode(row.mapping.Reasons[0]) {
				t.Fatalf("%s: reason %q is valid", row.rule, row.mapping.Reasons[0])
			}
		case row.rule == "reason-sorted-unique/unsorted":
			if row.mapping.Reasons[0] <= row.mapping.Reasons[1] {
				t.Fatalf("%s: pair is sorted", row.rule)
			}
			for _, reason := range row.mapping.Reasons {
				if !clonefidelity.ValidReasonCode(reason) {
					t.Fatalf("%s: reason %q is outside the vocabulary", row.rule, reason)
				}
			}
		case row.rule == "reason-sorted-unique/duplicate":
			if row.mapping.Reasons[0] != row.mapping.Reasons[1] {
				t.Fatalf("%s: pair is not a duplicate", row.rule)
			}
		case row.rule == "reason-bounds/long":
			probe := row.mapping.Reasons[0]
			if !clonefidelity.ValidReasonCode(probe) {
				t.Fatalf("%s: long probe is outside the vocabulary, bounds is not the only violation", row.rule)
			}
			if len([]rune(probe)) != 129 {
				t.Fatalf("%s: long probe carries %d characters, want 129", row.rule, len([]rune(probe)))
			}
		case row.rule == "reason-count/129":
			if !sort.StringsAreSorted(row.mapping.Reasons) {
				t.Fatalf("%s: reasons are not sorted", row.rule)
			}
			for _, reason := range row.mapping.Reasons {
				if !clonefidelity.ValidReasonCode(reason) {
					t.Fatalf("%s: reason %q is outside the vocabulary", row.rule, reason)
				}
			}
		case row.rule == "reason-bounds/empty":
			if row.mapping.Reasons[0] != "" {
				t.Fatalf("%s: probe is not empty", row.rule)
			}
		default:
			t.Fatalf("unhygiened rule-table row %q", row.rule)
		}
	}
}

func TestPlanItemEmittedMappingsOwnerValid(t *testing.T) {
	// Planner/owner agreement over the whole domain: every mapping
	// PlanItem admits validates through the owner's seam with
	// independently built scaffolding (different key, class,
	// evidence, and explanation from production's), so the
	// PlanItem self-check arm is proven silent by measurement,
	// not by reading. The sweep also proves the planner never
	// emits a row the owner would reject.
	checked := 0
	for _, strategy := range clonefidelity.Strategies() {
		for _, profile := range clonefidelity.Profiles() {
			for _, item := range allValidItems() {
				mapping, err := PlanItem(strategy, profile, item)
				if err != nil {
					continue
				}
				row := clonefidelity.DispositionRecordInput{
					SourceItemKey:     "oracle probe",
					SourceClass:       "oracle_probe",
					SourceEvidenceIDs: []string{orderedDigest(7)},
					Disposition:       mapping.Disposition,
					ReasonCodes:       mapping.Reasons,
					Explanation:       "oracle emitted row",
				}
				if err := clonefidelity.ValidateDispositionRow(row); err != nil {
					t.Fatalf("%s/%s/%s emitted %+v, the owner refuses: %v", strategy, profile, describeItem(item), mapping, err)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no admitted mappings to validate")
	}
	t.Logf("emitted owner-valid: %d mappings", checked)
}

func TestOwnerRowScaffoldingShape(t *testing.T) {
	// Construction pin for the delegation seam: the disposition
	// and reasons pass through verbatim, canonical object and
	// target locator render null (the Mapping type carries
	// neither, so the owner's synthesized and evidence rules see
	// only the passing arm through this entry), and the evidence
	// is one valid digest that differs per distinct mapping.
	probes := []Mapping{
		{Disposition: "exact"},
		{Disposition: "semantic", Reasons: []string{"operator_policy"}},
		{Disposition: "omitted", Reasons: []string{"operator_policy", "unsupported_media_type"}},
	}
	seen := map[string]bool{}
	for _, mapping := range probes {
		row := ownerRowForMapping(mapping)
		if row.Disposition != mapping.Disposition || !equalReasons(row.ReasonCodes, mapping.Reasons) {
			t.Fatalf("seam rewrote %+v to %q %q", mapping, row.Disposition, row.ReasonCodes)
		}
		if row.CanonicalObjectID != nil || row.TargetLocator != nil {
			t.Fatalf("seam rendered a canonical object or locator for %+v", mapping)
		}
		if row.SourceItemKey != effectsSourceKey || row.SourceClass != effectsSourceClass || row.Explanation != effectsExplanation {
			t.Fatalf("seam scaffolding %q/%q/%q drifted", row.SourceItemKey, row.SourceClass, row.Explanation)
		}
		if len(row.SourceEvidenceIDs) != 1 {
			t.Fatalf("seam rendered %d evidence IDs, want 1", len(row.SourceEvidenceIDs))
		}
		if _, err := scalar.ParseDigest(row.SourceEvidenceIDs[0]); err != nil {
			t.Fatalf("seam evidence %q is not a digest: %v", row.SourceEvidenceIDs[0], err)
		}
		seen[row.SourceEvidenceIDs[0]] = true
	}
	if len(seen) != len(probes) {
		t.Fatalf("seam evidence repeats across distinct mappings")
	}
}

func TestPlanTargetEffectsChained(t *testing.T) {
	// The real chain: PlanSession output folds through
	// PlanTargetEffects to zero.
	items := []Item{
		{Kind: "user_message"},
		{Kind: "tool_call", Resolution: "aborted"},
		{Kind: "usage"},
		{Kind: "subagent_completed"},
		{Block: "opaque"},
	}
	mappings, _, err := PlanSession("target_official_import", "compact", items)
	if err != nil {
		t.Fatalf("PlanSession refused: %v", err)
	}
	effects, err := PlanTargetEffects(mappings)
	if err != nil {
		t.Fatalf("PlanTargetEffects refused: %v", err)
	}
	if len(effects.CallableTools) != 0 || len(effects.PendingActions) != 0 ||
		effects.TargetInputTokens != 0 || effects.TargetOutputTokens != 0 {
		t.Fatalf("chained effects = %+v, want zero", effects)
	}
}
