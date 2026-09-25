package cloneplanning

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// This file is the regression answer to review finding
// escape-invalid-utf8-value-change (CR rev3): EscapeVisibleText
// passed its input to encoding/json.Marshal, which silently
// rewrites invalid UTF-8 as U+FFFD and succeeds, so only the
// composed ProjectVisibleText entry refused malformed text. Every
// exported entry that accepts text now refuses invalid UTF-8
// through the shared validText gate before any encoding step, and
// no entry returns a value whose text differs from its input.
//
// The refusal table below drives every censused entry with every
// invalid-UTF-8 class; the census test pins the entry set from
// the production AST, never a hand list.

// utf8InvalidClass is one invalid-UTF-8 class with its probe. The
// five classes are the rework brief's list; the ff-fe probe is
// exactly the narrowing probe the escape/profile mutants admit.
type utf8InvalidClass struct {
	name  string
	probe string
}

func utf8InvalidClasses() []utf8InvalidClass {
	return []utf8InvalidClass{
		{"lone-continuation", "\x80"},
		{"ff-fe", "\xff\xfe"},
		{"overlong", "\xc0\xaf"},
		{"lone-surrogate", "\xed\xa0\x80"},
		{"truncated", "abc\xe2\x82"},
	}
}

// requireUTF8Refusal asserts the UTF-8 refusal: the entry refuses,
// and the refusal names the literal "valid UTF-8" from
// SPEC.v0.7.0.md:256 ("text MUST be valid UTF-8"), so a fallback
// gate refusing with another message does not satisfy the cell.
func requireUTF8Refusal(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: invalid UTF-8 admitted", label)
	}
	if !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("%s: refusal %q does not name valid UTF-8", label, err.Error())
	}
}

func driveEscapeVisibleText(t *testing.T, class, probe string) {
	t.Helper()
	got, err := EscapeVisibleText(probe)
	label := "EscapeVisibleText/" + class
	requireUTF8Refusal(t, label, err)
	if got != "" {
		t.Fatalf("%s: refused but returned %q, want no value", label, got)
	}
	if strings.Contains(got, "\uFFFD") {
		t.Fatalf("%s: returned a U+FFFD-substituted value", label)
	}
}

func driveSelectProfile(t *testing.T, class, probe string) {
	t.Helper()
	_, _, err := SelectProfile(probe)
	requireUTF8Refusal(t, "SelectProfile/"+class, err)
}

func driveClassifyItem(t *testing.T, class, probe string) {
	t.Helper()
	_, err := ClassifyItem(clonebundle.CanonicalEvent{Kind: probe})
	requireUTF8Refusal(t, "ClassifyItem/kind/"+class, err)
	_, err = ClassifyItem(clonebundle.CanonicalEvent{
		Kind:    "tool_call",
		Payload: map[string]any{"resolution": probe},
	})
	requireUTF8Refusal(t, "ClassifyItem/resolution/"+class, err)
	_, err = ClassifyItem(clonebundle.CanonicalEvent{
		Kind:    "opaque_reasoning",
		Payload: map[string]any{"protection": probe},
	})
	requireUTF8Refusal(t, "ClassifyItem/protection/"+class, err)
}

func drivePlanItem(t *testing.T, class, probe string) {
	t.Helper()
	validItem := Item{Kind: "user_message"}
	_, err := PlanItem(probe, "maximal_safe", validItem)
	requireUTF8Refusal(t, "PlanItem/strategy/"+class, err)
	_, err = PlanItem("target_native_writer", probe, validItem)
	requireUTF8Refusal(t, "PlanItem/profile/"+class, err)
	_, err = PlanItem("target_native_writer", "maximal_safe", Item{Kind: probe})
	requireUTF8Refusal(t, "PlanItem/kind/"+class, err)
	_, err = PlanItem("target_native_writer", "maximal_safe", Item{Block: probe})
	requireUTF8Refusal(t, "PlanItem/block/"+class, err)
	_, err = PlanItem("target_native_writer", "maximal_safe", Item{Kind: "tool_call", Resolution: probe})
	requireUTF8Refusal(t, "PlanItem/resolution/"+class, err)
	_, err = PlanItem("target_native_writer", "maximal_safe", Item{Kind: "opaque_reasoning", Protection: probe})
	requireUTF8Refusal(t, "PlanItem/protection/"+class, err)
}

func drivePlanSession(t *testing.T, class, probe string) {
	t.Helper()
	items := []Item{{Kind: "user_message"}}
	_, _, err := PlanSession(probe, "maximal_safe", items)
	requireUTF8Refusal(t, "PlanSession/strategy/"+class, err)
	_, _, err = PlanSession("target_native_writer", probe, items)
	requireUTF8Refusal(t, "PlanSession/profile/"+class, err)
	_, _, err = PlanSession("target_native_writer", "maximal_safe", []Item{{Kind: probe}})
	requireUTF8Refusal(t, "PlanSession/item/"+class, err)
}

func drivePlanTargetEffects(t *testing.T, class, probe string) {
	t.Helper()
	_, err := PlanTargetEffects([]Mapping{{Disposition: probe}})
	requireUTF8Refusal(t, "PlanTargetEffects/disposition/"+class, err)
	_, err = PlanTargetEffects([]Mapping{{Disposition: "exact", Reasons: []string{probe}}})
	requireUTF8Refusal(t, "PlanTargetEffects/reason/"+class, err)
}

func driveProjectVisibleText(t *testing.T, class, probe string) {
	t.Helper()
	ids := []string{orderedDigest(1)}
	_, err := ProjectVisibleText(VisibleInput{Kind: "user_message", Texts: []string{probe}, EventIDs: ids})
	requireUTF8Refusal(t, "ProjectVisibleText/texts/"+class, err)
	_, err = ProjectVisibleText(VisibleInput{Kind: probe, Texts: []string{"a"}, EventIDs: ids})
	requireUTF8Refusal(t, "ProjectVisibleText/kind/"+class, err)
	_, err = ProjectVisibleText(VisibleInput{Kind: "user_message", Texts: []string{"a"}, EventIDs: []string{probe}})
	requireUTF8Refusal(t, "ProjectVisibleText/eventIDs/"+class, err)
}

// utf8RefusalTable is the refusal table: every censused entry
// with the driver injecting invalid UTF-8 into each consumed text
// position. The census test asserts this table's key set equals
// the AST-derived text-entry set exactly: a censused entry
// missing here fails, and an unknown row fails.
var utf8RefusalTable = []struct {
	entry string
	drive func(t *testing.T, class, probe string)
}{
	{"EscapeVisibleText", driveEscapeVisibleText},
	{"SelectProfile", driveSelectProfile},
	{"ClassifyItem", driveClassifyItem},
	{"PlanItem", drivePlanItem},
	{"PlanSession", drivePlanSession},
	{"PlanTargetEffects", drivePlanTargetEffects},
	{"ProjectVisibleText", driveProjectVisibleText},
}

func TestUTF8ProbesAreInvalid(t *testing.T) {
	// Fixture guard: every refusal-table probe is actually
	// invalid UTF-8, and the ff-fe probe is exactly the member
	// the escape/profile narrowings admit.
	classes := utf8InvalidClasses()
	if len(classes) != 5 {
		t.Fatalf("refusal table carries %d invalid classes, want 5", len(classes))
	}
	seen := map[string]bool{}
	for _, class := range classes {
		if seen[class.name] {
			t.Fatalf("duplicate invalid class %q", class.name)
		}
		seen[class.name] = true
		if utf8.ValidString(class.probe) {
			t.Fatalf("class %s probe %q is valid UTF-8", class.name, class.probe)
		}
	}
	for _, class := range classes {
		if class.name == "ff-fe" && class.probe != "\xff\xfe" {
			t.Fatalf("ff-fe probe is %q, want exactly the narrowing member", class.probe)
		}
	}
}

func TestEscapeVisibleTextRefusesInvalidUTF8(t *testing.T) {
	// Killer for N-escape-utf8-fffe (the rev3 reviewer's
	// narrowing: skip validation for the ff-fe class at the
	// direct escaping entry).
	for _, class := range utf8InvalidClasses() {
		driveEscapeVisibleText(t, class.name, class.probe)
	}
	// The rev3 reviewer's exact probe (byte FF,
	// reviewer_invalid_escape_test.go): embedded so the next
	// cycle inherits it.
	driveEscapeVisibleText(t, "reviewer-exact-ff", "\xff")
}

func TestSelectProfileRefusesInvalidUTF8(t *testing.T) {
	// Killer for N-select-profile-utf8 (the second censused
	// entry: skip validation for the ff-fe class at profile
	// selection; the vocabulary fallback refuses with another
	// message, which fails the literal cell).
	for _, class := range utf8InvalidClasses() {
		driveSelectProfile(t, class.name, class.probe)
	}
}

func TestClassifyItemRefusesInvalidUTF8(t *testing.T) {
	for _, class := range utf8InvalidClasses() {
		driveClassifyItem(t, class.name, class.probe)
	}
}

func TestPlanItemRefusesInvalidUTF8(t *testing.T) {
	for _, class := range utf8InvalidClasses() {
		drivePlanItem(t, class.name, class.probe)
	}
}

func TestPlanSessionRefusesInvalidUTF8(t *testing.T) {
	for _, class := range utf8InvalidClasses() {
		drivePlanSession(t, class.name, class.probe)
	}
}

func TestPlanTargetEffectsRefusesInvalidUTF8(t *testing.T) {
	for _, class := range utf8InvalidClasses() {
		drivePlanTargetEffects(t, class.name, class.probe)
	}
}

func TestProjectVisibleTextRefusesInvalidUTF8(t *testing.T) {
	for _, class := range utf8InvalidClasses() {
		driveProjectVisibleText(t, class.name, class.probe)
	}
}

func TestClassifyItemIgnoresInvalidPayloadBytes(t *testing.T) {
	// Ignored members stay ignored even when their bytes are not
	// valid UTF-8: invalid bytes in any non-fact payload member
	// cannot change the item, because the item carries no text
	// field by type and the entry emits nothing derived from
	// them. Consumed text (kind, closed facts) refuses instead;
	// the refusal table pins that direction. Hand-built events:
	// the sealed builder cannot carry invalid UTF-8 at all
	// (encoding/json rewrites it), so only the direct struct
	// path exercises these bytes.
	items := eventSweepItems()
	cells := 0
	for _, want := range items {
		for _, field := range sweepProbeFields {
			for _, class := range utf8InvalidClasses() {
				payload := map[string]any{}
				if want.Resolution != "" {
					payload["resolution"] = want.Resolution
				}
				if want.Protection != "" {
					payload["protection"] = want.Protection
				}
				payload[field] = class.probe
				got, err := ClassifyItem(clonebundle.CanonicalEvent{Kind: want.Kind, Payload: payload})
				label := want.Kind + "/" + field + "/" + class.name
				if err != nil {
					t.Fatalf("ignored-bytes %s refused: %v", label, err)
				}
				if got != want {
					t.Fatalf("ignored-bytes %s = %+v, want %+v", label, got, want)
				}
				cells++
			}
		}
	}
	t.Logf("classify ignored-bytes: %d cells", cells)
}
