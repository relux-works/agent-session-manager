package cloneplan_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// Closed-constant oracle, retyped from pinned
// internal/specdoc/SPEC.v0.7.0.md §13.14.2: TransactionPlan carries
// materialization_intent=clone,
// target_collision_policy=must_be_absent and
// activation=dormant_validated (lines 10703-10706); ReadBackPlan
// carries modes:[staged,live] with require_identity_match,
// require_workspace_match and require_semantic_marker all true
// (10707-10711); ResumeProjectionPlan carries
// opens_existing_identity=true, allow_blank_fallback=false and an
// open bounded_continuation_turn_required:boolean (10712-10716);
// RollbackPlan carries required=true,
// retain_through=live_validated and
// forbidden_after_provider_commit=true (10716-10718); the plan
// schema row pins schema/schema_version (10654) and the registry
// pins both schema URNs with 1.0.0 (168-169).
var closedStringOracle = map[string]string{
	"materialization_intent":  "clone",
	"target_collision_policy": "must_be_absent",
	"activation":              "dormant_validated",
	"retain_through":          "live_validated",
}

// closedBoolOracle pins the closed boolean polarity per member:
// true means only true admits, false means only false admits.
var closedBoolOracle = map[string]bool{
	"require_identity_match":          true,
	"require_workspace_match":         true,
	"require_semantic_marker":         true,
	"opens_existing_identity":         true,
	"allow_blank_fallback":            false,
	"required":                        true,
	"forbidden_after_provider_commit": true,
}

// openBoolMembers holds boolean members that admit both values:
// the pinned text types bounded_continuation_turn_required as a
// bare boolean (10715).
var openBoolMembers = map[string]bool{
	"bounded_continuation_turn_required": true,
}

// modesOracle pins the closed ReadBack modes pair in order
// (SPEC.v0.7.0.md:10708).
var modesOracle = []string{"staged", "live"}

// schemaOraclePlan and schemaOracleManifest pin the top-level
// schema/version literals per shape (SPEC.v0.7.0.md:10654 for the
// plan row, 168-169 for the registry rows).
var schemaOraclePlan = map[string]string{
	"schema":         "urn:ax:schema:projection-plan",
	"schema_version": "1.0.0",
}

var schemaOracleManifest = map[string]string{
	"schema":         "urn:ax:schema:clone-projected-object-manifest",
	"schema_version": "1.0.0",
}

// nestedPlanGate binds one closed nested plan to its Go input
// field inside ProjectionPlanInput, its sealed-document member,
// and its production error owner phrase.
type nestedPlanGate struct {
	inputField string
	docMember  string
	owner      string
	typ        reflect.Type
}

// nestedPlanGates lists the four closed nested plans. The member
// FIELDS inside each input type derive by reflection below; only
// the plan-level wiring is tabular.
func nestedPlanGates() []nestedPlanGate {
	return []nestedPlanGate{
		{"TransactionPlan", "transaction_plan", "transaction plan", reflect.TypeOf(cloneplan.TransactionPlanInput{})},
		{"ReadBackPlan", "read_back_plan", "read-back plan", reflect.TypeOf(cloneplan.ReadBackPlanInput{})},
		{"ResumePlan", "resume_plan", "resume projection plan", reflect.TypeOf(cloneplan.ResumePlanInput{})},
		{"RollbackPlan", "rollback_plan", "rollback plan", reflect.TypeOf(cloneplan.RollbackPlanInput{})},
	}
}

// constantField is one derived constant-gate field: its Go field
// name, its JSON member (mechanical snake rule plus the documented
// irregular override, shared with the member census), and its
// kind (String, Bool, or Slice for modes).
type constantField struct {
	gate   nestedPlanGate
	field  string
	member string
	kind   reflect.Kind
}

// deriveConstantFields enumerates every gate field of the four
// nested plan input types. Map fields (extensions) are
// owner-gated and skipped by kind, mechanically; any other
// unhandled kind fails loudly so the set grows with the types and
// never shrinks silently.
func deriveConstantFields(t *testing.T) []constantField {
	t.Helper()
	var out []constantField
	for _, gate := range nestedPlanGates() {
		for index := 0; index < gate.typ.NumField(); index++ {
			field := gate.typ.Field(index)
			if field.PkgPath != "" {
				t.Fatalf("unexported field %s.%s: extend the constant census", gate.inputField, field.Name)
			}
			if field.Type.Kind() == reflect.Map {
				continue
			}
			member := snakeMember(field.Name)
			if irregular, ok := irregularMembers[field.Name]; ok {
				member = irregular
			}
			switch field.Type.Kind() {
			case reflect.String, reflect.Bool:
				out = append(out, constantField{gate: gate, field: field.Name, member: member, kind: field.Type.Kind()})
			case reflect.Slice:
				if field.Type.Elem().Kind() != reflect.String {
					t.Fatalf("unhandled slice element kind %s at %s.%s: extend the constant census", field.Type.Elem().Kind(), gate.inputField, field.Name)
				}
				out = append(out, constantField{gate: gate, field: field.Name, member: member, kind: reflect.Slice})
			default:
				t.Fatalf("unhandled kind %s at %s.%s: extend the constant census", field.Type.Kind(), gate.inputField, field.Name)
			}
		}
	}
	return out
}

// buildPlanNestedString drives BuildProjectionPlan with one nested
// string field set to value.
func buildPlanNestedString(t *testing.T, gate nestedPlanGate, field, value string) error {
	t.Helper()
	input := validPlanInput()
	nested := reflect.ValueOf(&input).Elem().FieldByName(gate.inputField)
	nested.FieldByName(field).SetString(value)
	_, err := cloneplan.BuildProjectionPlan(input)
	return err
}

// buildPlanNestedBool drives BuildProjectionPlan with one nested
// boolean field set to value.
func buildPlanNestedBool(t *testing.T, gate nestedPlanGate, field string, value bool) error {
	t.Helper()
	input := validPlanInput()
	nested := reflect.ValueOf(&input).Elem().FieldByName(gate.inputField)
	nested.FieldByName(field).SetBool(value)
	_, err := cloneplan.BuildProjectionPlan(input)
	return err
}

// buildPlanNestedModes drives BuildProjectionPlan with the nested
// modes slice set to modes.
func buildPlanNestedModes(t *testing.T, gate nestedPlanGate, field string, modes []string) error {
	t.Helper()
	input := validPlanInput()
	nested := reflect.ValueOf(&input).Elem().FieldByName(gate.inputField)
	nested.FieldByName(field).Set(reflect.ValueOf(modes))
	_, err := cloneplan.BuildProjectionPlan(input)
	return err
}

// decodePlanNestedMember drives DecodeProjectionPlan with one
// nested member set to value inside the sealed document.
func decodePlanNestedMember(t *testing.T, sealed []byte, gate nestedPlanGate, member string, value any) error {
	t.Helper()
	document := decodeDocument(t, sealed)
	document[gate.docMember].(map[string]any)[member] = value
	_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	return err
}

// decodePlanTopMember drives DecodeProjectionPlan with one
// top-level member set to value.
func decodePlanTopMember(t *testing.T, sealed []byte, member string, value any) error {
	t.Helper()
	document := decodeDocument(t, sealed)
	document[member] = value
	_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	return err
}

// decodeManifestTopMember drives DecodeProjectedObjectManifest
// with one top-level member set to value.
func decodeManifestTopMember(t *testing.T, sealed []byte, member string, value any) error {
	t.Helper()
	document := decodeDocument(t, sealed)
	document[member] = value
	_, err := cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, document))
	return err
}

// neighborAlphabet is the generator alphabet for invalid constant
// values: lowercase, digits, and underscore.
const neighborAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789_"

// constantNeighbors returns every string at edit distance 1 from
// the literal over neighborAlphabet (deletion, substitution,
// insertion, transposition), deduplicated, excluding the literal
// itself, in sorted order.
func constantNeighbors(literal string) []string {
	seen := map[string]bool{literal: true}
	var out []string
	add := func(value string) {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	runes := []rune(literal)
	alpha := []rune(neighborAlphabet)
	for index := range runes {
		deleted := append(append([]rune{}, runes[:index]...), runes[index+1:]...)
		add(string(deleted))
	}
	for index := range runes {
		for _, candidate := range alpha {
			if candidate == runes[index] {
				continue
			}
			substituted := append([]rune{}, runes...)
			substituted[index] = candidate
			add(string(substituted))
		}
	}
	for index := 0; index <= len(runes); index++ {
		for _, candidate := range alpha {
			inserted := append(append([]rune{}, runes[:index]...), candidate)
			inserted = append(inserted, runes[index:]...)
			add(string(inserted))
		}
	}
	for index := 0; index+1 < len(runes); index++ {
		if runes[index] == runes[index+1] {
			continue
		}
		transposed := append([]rune{}, runes...)
		transposed[index], transposed[index+1] = transposed[index+1], transposed[index]
		add(string(transposed))
	}
	sort.Strings(out)
	return out
}

// constantRandomSample returns count deterministic pseudo-random
// strings over neighborAlphabet (lengths 1..24) that differ from
// the literal, from a fixed seed.
func constantRandomSample(literal string, count int) []string {
	rng := rand.New(rand.NewSource(260830))
	alpha := []rune(neighborAlphabet)
	seen := map[string]bool{literal: true}
	var out []string
	for len(out) < count {
		length := 1 + rng.Intn(24)
		buf := make([]rune, length)
		for index := range buf {
			buf[index] = alpha[rng.Intn(len(alpha))]
		}
		value := string(buf)
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// sweepStrings returns the full generated invalid set for one
// string literal: every edit-distance-1 neighbor, every case
// variant class, a fixed-seed random sample, the reviewer's
// second-value probe, the empty string, and whitespace paddings.
func sweepStrings(literal string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(value string) {
		if value == literal || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}
	for _, value := range constantNeighbors(literal) {
		add(value)
	}
	for _, variant := range caseVariantsOf(literal) {
		add(variant.value)
	}
	for _, value := range constantRandomSample(literal, 200) {
		add(value)
	}
	add("copy")
	add("")
	add(" " + literal)
	add(literal + " ")
	add("\t" + literal)
	add(literal + "\n")
	return out
}

// derivedProbes returns the fast derived invalid set for one
// string literal: case folds, the first transposition, the
// last-rune deletion, the digit-2 insertion neighbor, the empty
// string, and whitespace padding. Every probe derives from the
// literal; none is hand-picked.
func derivedProbes(literal string) []string {
	seen := map[string]bool{literal: true}
	var out []string
	add := func(value string) {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	runes := []rune(literal)
	add(strings.ToUpper(literal))
	if len(runes) >= 1 {
		add(strings.ToUpper(string(runes[0])) + string(runes[1:]))
		add(string(runes[:len(runes)-1]))
	}
	if len(runes) >= 2 {
		transposed := append([]rune{}, runes...)
		transposed[0], transposed[1] = transposed[1], transposed[0]
		add(string(transposed))
	}
	add(literal + "2")
	add("")
	add(" " + literal + " ")
	return out
}

// nonStringJSONValues is the decode-side non-string grid: null,
// boolean, number, array, and object.
func nonStringJSONValues() []any {
	return []any{nil, true, float64(1), []any{"x"}, map[string]any{"x": 1}}
}

// TestPlanConstantCopyRegression is the named regression for the
// rev3 finding plan-constant-second-value-uncounted: the value
// "copy" refuses at every closed string gate on both production
// entries. Each subtest is the killer for its gate's copy
// narrowing, run alone.
func TestPlanConstantCopyRegression(t *testing.T) {
	for _, field := range deriveConstantFields(t) {
		if field.kind != reflect.String {
			continue
		}
		literal, ok := closedStringOracle[field.member]
		if !ok {
			t.Fatalf("derived string field %s.%s has no oracle literal", field.gate.inputField, field.field)
		}
		field := field
		t.Run(field.member+"-build", func(t *testing.T) {
			// Sibling probes: the literal admits and the first
			// derived neighbor refuses, so a kill lands on the
			// copy probe only when the plant admits exactly copy.
			t.Run("literal-admits", func(t *testing.T) {
				if err := buildPlanNestedString(t, field.gate, field.field, literal); err != nil {
					t.Fatalf("literal %q refused: %v", literal, err)
				}
			})
			t.Run("sibling-refuses", func(t *testing.T) {
				sibling := derivedProbes(literal)[0]
				err := buildPlanNestedString(t, field.gate, field.field, sibling)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			})
			t.Run("copy-refuses", func(t *testing.T) {
				err := buildPlanNestedString(t, field.gate, field.field, "copy")
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			})
		})
		t.Run(field.member+"-decode", func(t *testing.T) {
			sealed := mustBuildPlan(t, validPlanInput())
			t.Run("literal-admits", func(t *testing.T) {
				mustDecodePlan(t, sealed)
			})
			t.Run("sibling-refuses", func(t *testing.T) {
				sibling := derivedProbes(literal)[0]
				err := decodePlanNestedMember(t, sealed, field.gate, field.member, sibling)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			})
			t.Run("copy-refuses", func(t *testing.T) {
				err := decodePlanNestedMember(t, sealed, field.gate, field.member, "copy")
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			})
		})
	}
	modes := modesGate(t)
	t.Run("modes-build", func(t *testing.T) {
		t.Run("literal-admits", func(t *testing.T) {
			if err := buildPlanNestedModes(t, modes.gate, modes.field, []string{"staged", "live"}); err != nil {
				t.Fatalf("literal modes refused: %v", err)
			}
		})
		t.Run("sibling-refuses", func(t *testing.T) {
			err := buildPlanNestedModes(t, modes.gate, modes.field, []string{"live", "staged"})
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		})
		t.Run("copy-refuses", func(t *testing.T) {
			err := buildPlanNestedModes(t, modes.gate, modes.field, []string{"staged", "copy"})
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		})
	})
	t.Run("modes-decode", func(t *testing.T) {
		sealed := mustBuildPlan(t, validPlanInput())
		t.Run("literal-admits", func(t *testing.T) {
			mustDecodePlan(t, sealed)
		})
		t.Run("sibling-refuses", func(t *testing.T) {
			err := decodePlanNestedMember(t, sealed, modes.gate, modes.member, []any{"live", "staged"})
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		})
		t.Run("copy-refuses", func(t *testing.T) {
			err := decodePlanNestedMember(t, sealed, modes.gate, modes.member, []any{"staged", "copy"})
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		})
	})
	sealed := mustBuildPlan(t, validPlanInput())
	for member, literal := range schemaOraclePlan {
		member, literal := member, literal
		t.Run(member+"-plan", func(t *testing.T) {
			t.Run("literal-admits", func(t *testing.T) {
				mustDecodePlan(t, sealed)
			})
			t.Run("sibling-refuses", func(t *testing.T) {
				sibling := derivedProbes(literal)[0]
				err := decodePlanTopMember(t, sealed, member, sibling)
				requireRefusal(t, err, "projection plan", member+" is not "+literal)
			})
			t.Run("copy-refuses", func(t *testing.T) {
				err := decodePlanTopMember(t, sealed, member, "copy")
				requireRefusal(t, err, "projection plan", member+" is not "+literal)
			})
		})
	}
	msealed := mustBuildManifest(t, validManifestInput())
	for member, literal := range schemaOracleManifest {
		member, literal := member, literal
		t.Run(member+"-manifest", func(t *testing.T) {
			t.Run("literal-admits", func(t *testing.T) {
				mustDecodeManifest(t, msealed)
			})
			t.Run("sibling-refuses", func(t *testing.T) {
				sibling := derivedProbes(literal)[0]
				err := decodeManifestTopMember(t, msealed, member, sibling)
				requireRefusal(t, err, "projected object manifest", member+" is not "+literal)
			})
			t.Run("copy-refuses", func(t *testing.T) {
				err := decodeManifestTopMember(t, msealed, member, "copy")
				requireRefusal(t, err, "projected object manifest", member+" is not "+literal)
			})
		})
	}
}

// modesGate returns the derived modes field: exactly one []string
// field exists across the nested plan inputs.
func modesGate(t *testing.T) constantField {
	t.Helper()
	var found []constantField
	for _, field := range deriveConstantFields(t) {
		if field.kind == reflect.Slice {
			found = append(found, field)
		}
	}
	if len(found) != 1 {
		t.Fatalf("derived []string fields = %d, want exactly 1 (modes)", len(found))
	}
	return found[0]
}

// TestPlanConstantDerivedValues refuses the fast derived invalid
// set at every closed string gate on both entries: case folds,
// the first transposition, the last-rune deletion, the digit-2
// insertion neighbor, the empty string, and whitespace padding.
func TestPlanConstantDerivedValues(t *testing.T) {
	for _, field := range deriveConstantFields(t) {
		if field.kind != reflect.String {
			continue
		}
		literal, ok := closedStringOracle[field.member]
		if !ok {
			t.Fatalf("derived string field %s.%s has no oracle literal", field.gate.inputField, field.field)
		}
		field := field
		t.Run(field.member+"-build", func(t *testing.T) {
			for _, probe := range derivedProbes(literal) {
				err := buildPlanNestedString(t, field.gate, field.field, probe)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			}
		})
		t.Run(field.member+"-decode", func(t *testing.T) {
			sealed := mustBuildPlan(t, validPlanInput())
			for _, probe := range derivedProbes(literal) {
				err := decodePlanNestedMember(t, sealed, field.gate, field.member, probe)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			}
		})
	}
	arrays := modesDerivedArrays(t)
	t.Run("modes-build", func(t *testing.T) {
		for _, modes := range arrays {
			err := buildPlanNestedModes(t, modes.gate, modes.field, modes.build)
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		}
	})
	t.Run("modes-decode", func(t *testing.T) {
		sealed := mustBuildPlan(t, validPlanInput())
		for _, modes := range arrays {
			err := decodePlanNestedMember(t, sealed, modes.gate, modes.member, modes.decode)
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		}
	})
	sealed := mustBuildPlan(t, validPlanInput())
	for member, literal := range schemaOraclePlan {
		member, literal := member, literal
		t.Run(member+"-plan", func(t *testing.T) {
			for _, probe := range derivedProbes(literal) {
				err := decodePlanTopMember(t, sealed, member, probe)
				requireRefusal(t, err, "projection plan", member+" is not "+literal)
			}
		})
	}
	msealed := mustBuildManifest(t, validManifestInput())
	for member, literal := range schemaOracleManifest {
		member, literal := member, literal
		t.Run(member+"-manifest", func(t *testing.T) {
			for _, probe := range derivedProbes(literal) {
				err := decodeManifestTopMember(t, msealed, member, probe)
				requireRefusal(t, err, "projected object manifest", member+" is not "+literal)
			}
		})
	}
}

// modesArray is one invalid modes probe in build ([]string) and
// decode ([]any) spelling.
type modesArray struct {
	gate   nestedPlanGate
	field  string
	member string
	build  []string
	decode []any
}

// modesDerivedArrays returns the fast derived invalid modes set:
// the order swap, length variants, the duplication, the digit-2
// insertion neighbor, the first case variant of each element, and
// the first transposition neighbor of each element.
func modesDerivedArrays(t *testing.T) []modesArray {
	t.Helper()
	modes := modesGate(t)
	arrays := [][2]string{
		{"live", "staged"},
		{"staged", "live2"},
	}
	for _, variant := range caseVariantsOf(modesOracle[0]) {
		arrays = append(arrays, [2]string{variant.value, modesOracle[1]})
	}
	for _, variant := range caseVariantsOf(modesOracle[1]) {
		arrays = append(arrays, [2]string{modesOracle[0], variant.value})
	}
	neighbors := func(literal string) string {
		transposed := []rune(literal)
		transposed[0], transposed[1] = transposed[1], transposed[0]
		return string(transposed)
	}
	arrays = append(arrays, [2]string{neighbors(modesOracle[0]), modesOracle[1]})
	arrays = append(arrays, [2]string{modesOracle[0], neighbors(modesOracle[1])})
	var out []modesArray
	for _, pair := range arrays {
		out = append(out, modesArray{
			gate: modes.gate, field: modes.field, member: modes.member,
			build: []string{pair[0], pair[1]}, decode: []any{pair[0], pair[1]},
		})
	}
	for _, short := range [][]string{{}, {"staged"}, {"live"}, {"staged", "live", "live"}, {"staged", "staged"}} {
		decoded := make([]any, 0, len(short))
		for _, element := range short {
			decoded = append(decoded, element)
		}
		out = append(out, modesArray{
			gate: modes.gate, field: modes.field, member: modes.member,
			build: short, decode: decoded,
		})
	}
	return out
}

// TestPlanConstantGeneratedSweep refuses the full generated
// invalid set at every closed string gate: every
// edit-distance-1 neighbor, every case variant class, a
// fixed-seed random sample, the second-value probe, the empty
// string, whitespace paddings, and the non-string JSON grid at
// decode. This is breadth evidence; the per-gate narrowing rows
// name the fast killers above so the battery stays quick.
func TestPlanConstantGeneratedSweep(t *testing.T) {
	for _, field := range deriveConstantFields(t) {
		if field.kind != reflect.String {
			continue
		}
		literal, ok := closedStringOracle[field.member]
		if !ok {
			t.Fatalf("derived string field %s.%s has no oracle literal", field.gate.inputField, field.field)
		}
		field := field
		t.Run(field.member+"-build", func(t *testing.T) {
			for _, probe := range sweepStrings(literal) {
				err := buildPlanNestedString(t, field.gate, field.field, probe)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			}
		})
		t.Run(field.member+"-decode", func(t *testing.T) {
			sealed := mustBuildPlan(t, validPlanInput())
			for _, probe := range sweepStrings(literal) {
				err := decodePlanNestedMember(t, sealed, field.gate, field.member, probe)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			}
			for _, probe := range nonStringJSONValues() {
				err := decodePlanNestedMember(t, sealed, field.gate, field.member, probe)
				requireRefusal(t, err, field.gate.owner, field.member+" is not "+literal)
			}
		})
	}
	t.Run("modes-build", func(t *testing.T) {
		for _, modes := range modesSweepArrays(t) {
			err := buildPlanNestedModes(t, modes.gate, modes.field, modes.build)
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		}
	})
	t.Run("modes-decode", func(t *testing.T) {
		sealed := mustBuildPlan(t, validPlanInput())
		field := modesGate(t)
		for _, modes := range modesSweepArrays(t) {
			err := decodePlanNestedMember(t, sealed, modes.gate, modes.member, modes.decode)
			requireRefusal(t, err, modes.gate.owner, "modes are not [staged,live]")
		}
		for _, probe := range modesNonStringElements() {
			err := decodePlanNestedMember(t, sealed, field.gate, field.member, probe)
			requireRefusal(t, err, field.gate.owner, "modes are not [staged,live]")
		}
		for _, probe := range modesNonArrays() {
			err := decodePlanNestedMember(t, sealed, field.gate, field.member, probe)
			requireRefusal(t, err, field.gate.owner, "modes are not [staged,live]")
		}
	})
	sealed := mustBuildPlan(t, validPlanInput())
	for member, literal := range schemaOraclePlan {
		member, literal := member, literal
		t.Run(member+"-plan", func(t *testing.T) {
			for _, probe := range sweepStrings(literal) {
				err := decodePlanTopMember(t, sealed, member, probe)
				requireRefusal(t, err, "projection plan", member+" is not "+literal)
			}
			for _, probe := range nonStringJSONValues() {
				err := decodePlanTopMember(t, sealed, member, probe)
				requireRefusal(t, err, "projection plan", member+" is not "+literal)
			}
		})
	}
	msealed := mustBuildManifest(t, validManifestInput())
	for member, literal := range schemaOracleManifest {
		member, literal := member, literal
		t.Run(member+"-manifest", func(t *testing.T) {
			for _, probe := range sweepStrings(literal) {
				err := decodeManifestTopMember(t, msealed, member, probe)
				requireRefusal(t, err, "projected object manifest", member+" is not "+literal)
			}
			for _, probe := range nonStringJSONValues() {
				err := decodeManifestTopMember(t, msealed, member, probe)
				requireRefusal(t, err, "projected object manifest", member+" is not "+literal)
			}
		})
	}
}

// modesSweepArrays returns the full modes sweep: every
// edit-distance-1 neighbor of each element in place, every case
// variant of each element in place, the order swap, and length
// variants 0..4.
func modesSweepArrays(t *testing.T) []modesArray {
	t.Helper()
	modes := modesGate(t)
	pair := func(first, second string) modesArray {
		return modesArray{
			gate: modes.gate, field: modes.field, member: modes.member,
			build: []string{first, second}, decode: []any{first, second},
		}
	}
	var out []modesArray
	for _, neighbor := range constantNeighbors(modesOracle[0]) {
		out = append(out, pair(neighbor, modesOracle[1]))
	}
	for _, neighbor := range constantNeighbors(modesOracle[1]) {
		out = append(out, pair(modesOracle[0], neighbor))
	}
	for _, variant := range caseVariantsOf(modesOracle[0]) {
		out = append(out, pair(variant.value, modesOracle[1]))
	}
	for _, variant := range caseVariantsOf(modesOracle[1]) {
		out = append(out, pair(modesOracle[0], variant.value))
	}
	out = append(out, pair("staged", "copy"))
	out = append(out, pair(modesOracle[1], modesOracle[0]))
	lengths := [][]string{{}, {"staged"}, {"live"}, {"staged", "staged"}, {"live", "live"},
		{"staged", "live", "staged"}, {"staged", "live", "live"}, {"staged", "live", "live", "live"}}
	for _, elements := range lengths {
		decoded := make([]any, 0, len(elements))
		for _, element := range elements {
			decoded = append(decoded, element)
		}
		out = append(out, modesArray{
			gate: modes.gate, field: modes.field, member: modes.member,
			build: elements, decode: decoded,
		})
	}
	return out
}

// modesNonStringElements returns modes arrays carrying non-string
// elements for the decode type grid.
func modesNonStringElements() []any {
	return []any{
		[]any{nil, "live"},
		[]any{"staged", nil},
		[]any{true, "live"},
		[]any{"staged", float64(1)},
		[]any{[]any{"staged"}, "live"},
		[]any{"staged", map[string]any{"x": 1}},
	}
}

// modesNonArrays returns non-array JSON values for the decode
// modes member.
func modesNonArrays() []any {
	return []any{nil, "staged", true, float64(1), map[string]any{"x": 1}}
}

// TestPlanConstantBooleans drives every derived boolean gate: a
// closed member refuses the other boolean on both entries, and
// the open bounded flag admits both booleans on both entries.
func TestPlanConstantBooleans(t *testing.T) {
	for _, field := range deriveConstantFields(t) {
		if field.kind != reflect.Bool {
			continue
		}
		field := field
		t.Run(field.member, func(t *testing.T) {
			if openBoolMembers[field.member] {
				for _, value := range []bool{true, false} {
					input := validPlanInput()
					nested := reflect.ValueOf(&input).Elem().FieldByName(field.gate.inputField)
					nested.FieldByName(field.field).SetBool(value)
					sealed := mustBuildPlan(t, input)
					plan := mustDecodePlan(t, sealed)
					validated := reflect.ValueOf(plan).FieldByName(field.gate.inputField)
					if !validated.IsValid() {
						t.Fatalf("validated plan has no %s: extend the open-boolean admission check", field.gate.inputField)
					}
					got := validated.FieldByName(field.field)
					if !got.IsValid() || got.Kind() != reflect.Bool {
						t.Fatalf("validated %s has no boolean %s: extend the open-boolean admission check", field.gate.inputField, field.field)
					}
					if got.Bool() != value {
						t.Fatalf("%s = %v, want %v", field.member, got.Bool(), value)
					}
				}
				return
			}
			polarity, ok := closedBoolOracle[field.member]
			if !ok {
				t.Fatalf("derived boolean field %s.%s has no oracle polarity", field.gate.inputField, field.field)
			}
			word := "true"
			if !polarity {
				word = "false"
			}
			err := buildPlanNestedBool(t, field.gate, field.field, !polarity)
			requireRefusal(t, err, field.gate.owner, field.member+" is not "+word)
			sealed := mustBuildPlan(t, validPlanInput())
			err = decodePlanNestedMember(t, sealed, field.gate, field.member, !polarity)
			requireRefusal(t, err, field.gate.owner, field.member+" is not "+word)
		})
	}
}

// TestPlanConstantNonBooleanJSON refuses every non-boolean JSON
// type at every derived boolean member on decode: string, number,
// array, object, and null last. Each closed-member subtest is the
// killer for its null-admitting narrowing, run alone: the
// other-boolean and non-null probes refuse first, so a kill lands
// on the null probe only when the plant admits exactly null.
func TestPlanConstantNonBooleanJSON(t *testing.T) {
	values := []struct {
		label string
		value any
	}{
		{"string", "true"},
		{"number", float64(1)},
		{"array", []any{true}},
		{"object", map[string]any{}},
		{"null", nil},
	}
	for _, field := range deriveConstantFields(t) {
		if field.kind != reflect.Bool {
			continue
		}
		field := field
		t.Run(field.member, func(t *testing.T) {
			token := field.member + " is not a boolean"
			if polarity, ok := closedBoolOracle[field.member]; ok {
				word := "true"
				if !polarity {
					word = "false"
				}
				token = field.member + " is not " + word
				t.Run("other-boolean", func(t *testing.T) {
					sealed := mustBuildPlan(t, validPlanInput())
					err := decodePlanNestedMember(t, sealed, field.gate, field.member, !polarity)
					requireRefusal(t, err, field.gate.owner, token)
				})
			} else if !openBoolMembers[field.member] {
				t.Fatalf("derived boolean field %s.%s has no oracle entry", field.gate.inputField, field.field)
			}
			sealed := mustBuildPlan(t, validPlanInput())
			for _, probe := range values {
				probe := probe
				t.Run(probe.label, func(t *testing.T) {
					err := decodePlanNestedMember(t, sealed, field.gate, field.member, probe.value)
					requireRefusal(t, err, field.gate.owner, token)
				})
			}
		})
	}
}

// TestClosedConstantGateShape is the structural guard: every
// closed-constant gate compares its value for equality against
// exactly ONE spec literal and refuses otherwise. The gate LIST
// derives mechanically (reflection over the nested plan input
// types plus type-directed function lookup in the AST); only the
// expected VALUES come from the spec oracle above. A widened gate
// (a second admitted value), a two-case switch, or a
// prefix/fold/contains comparison changes the gate shape or the
// comparison-literal set and fails this test. Control plants (the
// reviewer's copy widening, a two-case switch) redden it; see the
// rev4 results.
func TestClosedConstantGateShape(t *testing.T) {
	fields := deriveConstantFields(t)
	var stringFields, boolFields, sliceFields []constantField
	for _, field := range fields {
		switch field.kind {
		case reflect.String:
			stringFields = append(stringFields, field)
		case reflect.Bool:
			boolFields = append(boolFields, field)
		case reflect.Slice:
			sliceFields = append(sliceFields, field)
		}
	}
	// Both-directions cross-check: every derived string member has
	// an oracle literal and every oracle literal is a derived
	// member.
	derivedStrings := map[string]bool{}
	for _, field := range stringFields {
		derivedStrings[field.member] = true
	}
	for member := range closedStringOracle {
		if !derivedStrings[member] {
			t.Fatalf("oracle string member %q is not a derived field", member)
		}
	}
	for member := range derivedStrings {
		if _, ok := closedStringOracle[member]; !ok {
			t.Fatalf("derived string field %q has no oracle literal", member)
		}
	}
	derivedBools := map[string]bool{}
	for _, field := range boolFields {
		derivedBools[field.member] = true
	}
	for member := range closedBoolOracle {
		if !derivedBools[member] {
			t.Fatalf("oracle boolean member %q is not a derived field", member)
		}
	}
	for member := range openBoolMembers {
		if !derivedBools[member] {
			t.Fatalf("open boolean member %q is not a derived field", member)
		}
	}
	for member := range derivedBools {
		_, closed := closedBoolOracle[member]
		_, open := openBoolMembers[member]
		if !closed && !open {
			t.Fatalf("derived boolean field %q has no oracle entry", member)
		}
	}
	if len(sliceFields) != 1 {
		t.Fatalf("derived []string fields = %d, want exactly 1 (modes)", len(sliceFields))
	}

	parsed := parseProductionFiles(t, "plans.go", "plan.go", "manifest.go")
	constValues := resolveProductionConsts(t, parsed)

	// The schema/version const idents resolve to the oracle
	// literals; the idents are Go names, the literals are spec.
	for ident, want := range map[string]string{
		"projectionPlanSchema":     schemaOraclePlan["schema"],
		"projectionPlanVersion":    schemaOraclePlan["schema_version"],
		"projectedManifestSchema":  schemaOracleManifest["schema"],
		"projectedManifestVersion": schemaOracleManifest["schema_version"],
	} {
		got, ok := constValues[ident]
		if !ok {
			t.Fatalf("production const %s not found", ident)
		}
		if got != want {
			t.Fatalf("production const %s = %q, want oracle %q", ident, got, want)
		}
	}

	funcsByFile := map[string]map[string]*ast.FuncDecl{}
	for name, file := range parsed {
		funcs := map[string]*ast.FuncDecl{}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcs[fn.Name.Name] = fn
		}
		funcsByFile[name] = funcs
	}
	plansFuncs := funcsByFile["plans.go"]

	// Per-gate shape checks over the derived fields.
	for _, field := range stringFields {
		literal := closedStringOracle[field.member]
		buildFn := findBuildFunc(t, plansFuncs, field.gate.typ.Name())
		decodeFn := findDecodeFunc(t, plansFuncs, strings.TrimSuffix(field.gate.typ.Name(), "Input"))
		assertSingleStringBuildGate(t, buildFn, field.field, literal)
		assertSingleStringDecodeGate(t, decodeFn, field.member, literal)
	}
	for _, field := range boolFields {
		buildFn := findBuildFunc(t, plansFuncs, field.gate.typ.Name())
		decodeFn := findDecodeFunc(t, plansFuncs, strings.TrimSuffix(field.gate.typ.Name(), "Input"))
		if openBoolMembers[field.member] {
			assertOpenBoolBuildGate(t, buildFn, field.field)
			assertOpenBoolDecodeGate(t, decodeFn, field.member)
			continue
		}
		assertClosedBoolBuildGate(t, buildFn, field.field, closedBoolOracle[field.member])
		assertClosedBoolDecodeGate(t, decodeFn, field.member, closedBoolOracle[field.member])
	}
	modesField := sliceFields[0]
	readBackBuild := findBuildFunc(t, plansFuncs, modesField.gate.typ.Name())
	readBackDecode := findDecodeFunc(t, plansFuncs, strings.TrimSuffix(modesField.gate.typ.Name(), "Input"))
	assertModesGates(t, plansFuncs, readBackBuild, readBackDecode, modesField)

	// Schema/version single-literal gates at the top-level decodes.
	assertTopLevelLiteralGates(t, funcsByFile["plan.go"]["DecodeProjectionPlan"], schemaOraclePlan, constValues)
	assertTopLevelLiteralGates(t, funcsByFile["manifest.go"]["DecodeProjectedObjectManifest"], schemaOracleManifest, constValues)

	// Global shape rules over every walked function: no switch on
	// constant values, no prefix/fold/contains/normalize call, and
	// the exact comparison-literal set.
	walked := []*ast.FuncDecl{}
	for _, gate := range nestedPlanGates() {
		walked = append(walked, findBuildFunc(t, plansFuncs, gate.typ.Name()))
		walked = append(walked, findDecodeFunc(t, plansFuncs, strings.TrimSuffix(gate.typ.Name(), "Input")))
	}
	modesCallee := findModesCallee(t, readBackBuild, modesField.field)
	walked = append(walked, plansFuncs[modesCallee])
	for _, fn := range walked {
		assertNoSwitch(t, fn)
		assertNoFoldCalls(t, fn)
	}
	wantPlansLiterals := map[string]bool{}
	for _, literal := range closedStringOracle {
		wantPlansLiterals[literal] = true
	}
	for _, literal := range modesOracle {
		wantPlansLiterals[literal] = true
	}
	assertComparisonLiteralSet(t, walked, wantPlansLiterals)
	assertComparisonLiteralSet(t, []*ast.FuncDecl{funcsByFile["plan.go"]["DecodeProjectionPlan"]}, map[string]bool{})
	assertComparisonLiteralSet(t, []*ast.FuncDecl{funcsByFile["manifest.go"]["DecodeProjectedObjectManifest"]}, map[string]bool{})
}

// parseProductionFiles parses the named production files from the
// package directory.
func parseProductionFiles(t *testing.T, names ...string) map[string]*ast.File {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	parsed := map[string]*ast.File{}
	for _, name := range names {
		path := filepath.Join(filepath.Dir(file), name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		program, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		parsed[name] = program
	}
	return parsed
}

// resolveProductionConsts collects const ident to string-literal
// bindings from the parsed files.
func resolveProductionConsts(t *testing.T, parsed map[string]*ast.File) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, file := range parsed {
		for _, decl := range file.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, spec := range general.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
					continue
				}
				literal, ok := value.Values[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				out[value.Names[0].Name] = unquoteBasicLit(t, literal)
			}
		}
	}
	return out
}

// unquoteBasicLit unquotes a string literal or fails.
func unquoteBasicLit(t *testing.T, literal *ast.BasicLit) string {
	t.Helper()
	if literal.Kind != token.STRING {
		t.Fatalf("literal kind = %v, want STRING", literal.Kind)
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		t.Fatalf("unquote %s: %v", literal.Value, err)
	}
	return value
}

// findBuildFunc returns the unique plans.go function taking
// exactly one parameter of the named input type.
func findBuildFunc(t *testing.T, funcs map[string]*ast.FuncDecl, inputType string) *ast.FuncDecl {
	t.Helper()
	var found *ast.FuncDecl
	for _, fn := range funcs {
		if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
			continue
		}
		param := fn.Type.Params.List[0]
		if len(param.Names) != 1 {
			continue
		}
		ident, ok := param.Type.(*ast.Ident)
		if !ok || ident.Name != inputType {
			continue
		}
		if found != nil {
			t.Fatalf("two build funcs take %s", inputType)
		}
		found = fn
	}
	if found == nil {
		t.Fatalf("no build func takes %s", inputType)
	}
	return found
}

// findDecodeFunc returns the unique plans.go function taking
// exactly json.RawMessage and returning the named validated type
// plus an error.
func findDecodeFunc(t *testing.T, funcs map[string]*ast.FuncDecl, validatedType string) *ast.FuncDecl {
	t.Helper()
	var found *ast.FuncDecl
	for _, fn := range funcs {
		if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
			continue
		}
		param := fn.Type.Params.List[0]
		selector, ok := param.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok || pkg.Name != "json" || selector.Sel.Name != "RawMessage" {
			continue
		}
		if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
			continue
		}
		first, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
		if !ok || first.Name != validatedType {
			continue
		}
		if found != nil {
			t.Fatalf("two decode funcs return %s", validatedType)
		}
		found = fn
	}
	if found == nil {
		t.Fatalf("no decode func returns %s", validatedType)
	}
	return found
}

// ifConditions collects every if-statement condition in the
// function body.
func ifConditions(fn *ast.FuncDecl) []ast.Expr {
	var out []ast.Expr
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		branch, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		out = append(out, branch.Cond)
		return true
	})
	return out
}

// ifBodiesWithCond returns the bodies of if statements whose
// condition matches.
func ifBodiesWithCond(fn *ast.FuncDecl, match func(ast.Expr) bool) []*ast.BlockStmt {
	var out []*ast.BlockStmt
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		branch, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		if match(branch.Cond) {
			out = append(out, branch.Body)
		}
		return true
	})
	return out
}

// bodyReturns reports whether the block holds a return statement.
func bodyReturns(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(node ast.Node) bool {
		if _, ok := node.(*ast.ReturnStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

// isInputSelector reports whether the expression is input.Field.
func isInputSelector(expr ast.Expr, field string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	return ok && receiver.Name == "input" && selector.Sel.Name == field
}

// stringComparisonLiteral returns the literal a != comparison pins
// against target, or "".
func stringComparisonLiteral(t *testing.T, cond ast.Expr, target func(ast.Expr) bool) (string, bool) {
	t.Helper()
	comparison, ok := cond.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.NEQ {
		return "", false
	}
	var literal *ast.BasicLit
	switch {
	case target(comparison.X):
		literal, ok = comparison.Y.(*ast.BasicLit)
	case target(comparison.Y):
		literal, ok = comparison.X.(*ast.BasicLit)
	default:
		return "", false
	}
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	return unquoteBasicLit(t, literal), true
}

// assertSingleStringBuildGate pins the build side to exactly one
// `input.Field != "literal"` refusal.
func assertSingleStringBuildGate(t *testing.T, fn *ast.FuncDecl, field, literal string) {
	t.Helper()
	var matched []ast.Expr
	for _, cond := range ifConditions(fn) {
		got, ok := stringComparisonLiteral(t, cond, func(expr ast.Expr) bool {
			return isInputSelector(expr, field)
		})
		if !ok {
			continue
		}
		if got != literal {
			t.Fatalf("%s build gate for %s compares %q, want oracle %q", fn.Name.Name, field, got, literal)
		}
		matched = append(matched, cond)
	}
	if len(matched) != 1 {
		t.Fatalf("%s build gate for %s matches %d ifs, want exactly 1", fn.Name.Name, field, len(matched))
	}
	bodies := ifBodiesWithCond(fn, func(other ast.Expr) bool { return other == matched[0] })
	if len(bodies) != 1 || !bodyReturns(bodies[0]) {
		t.Fatalf("%s build gate for %s does not return", fn.Name.Name, field)
	}
}

// rawCallMember reports whether the expression calls helper on
// members["member"], returning the call.
func rawCallMember(expr ast.Expr, helper, member string) *ast.CallExpr {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil
	}
	fn, ok := call.Fun.(*ast.Ident)
	if !ok || fn.Name != helper {
		return nil
	}
	index, ok := call.Args[0].(*ast.IndexExpr)
	if !ok {
		return nil
	}
	receiver, ok := index.X.(*ast.Ident)
	if !ok || receiver.Name != "members" {
		return nil
	}
	key, ok := index.Index.(*ast.BasicLit)
	if !ok || key.Kind != token.STRING {
		return nil
	}
	value, err := strconv.Unquote(key.Value)
	if err != nil || value != member {
		return nil
	}
	return call
}

// decodeValueName returns the value variable a `v, ok :=
// helper(members["member"])` assignment binds, or "".
func decodeValueName(fn *ast.FuncDecl, helper, member string) (string, bool) {
	var name string
	count := 0
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
			return true
		}
		if rawCallMember(assign.Rhs[0], helper, member) == nil {
			return true
		}
		value, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		name = value.Name
		count++
		return true
	})
	return name, count == 1
}

// isNotIdent reports whether the expression is !name.
func isNotIdent(expr ast.Expr, name string) bool {
	negation, ok := expr.(*ast.UnaryExpr)
	if !ok || negation.Op != token.NOT {
		return false
	}
	ident, ok := negation.X.(*ast.Ident)
	return ok && ident.Name == name
}

// isIdent reports whether the expression is the bare name.
func isIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// orOperands flattens an || chain into operands, or returns nil
// for any other shape.
func orOperands(cond ast.Expr) []ast.Expr {
	comparison, ok := cond.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.LOR {
		return nil
	}
	var out []ast.Expr
	var flatten func(expr ast.Expr)
	flatten = func(expr ast.Expr) {
		nested, ok := expr.(*ast.BinaryExpr)
		if ok && nested.Op == token.LOR {
			flatten(nested.X)
			flatten(nested.Y)
			return
		}
		out = append(out, expr)
	}
	flatten(comparison)
	return out
}

// assertSingleStringDecodeGate pins the decode side to exactly one
// `!ok || v != "literal"` refusal fed by
// rawString(members["member"]).
func assertSingleStringDecodeGate(t *testing.T, fn *ast.FuncDecl, member, literal string) {
	t.Helper()
	value, ok := decodeValueName(fn, "rawString", member)
	if !ok {
		t.Fatalf("%s has no unique rawString(members[%q]) binding", fn.Name.Name, member)
	}
	count := 0
	for _, cond := range ifConditions(fn) {
		operands := orOperands(cond)
		if len(operands) != 2 || !isNotIdent(operands[0], "ok") {
			continue
		}
		got, ok := stringComparisonLiteral(t, operands[1], func(expr ast.Expr) bool {
			return isIdent(expr, value)
		})
		if !ok {
			continue
		}
		if got != literal {
			t.Fatalf("%s decode gate for %s compares %q, want oracle %q", fn.Name.Name, member, got, literal)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("%s decode gate for %s matches %d ifs, want exactly 1", fn.Name.Name, member, count)
	}
}

// assertClosedBoolBuildGate pins a closed boolean build gate to
// exactly one `!input.Field` (polarity true) or `input.Field`
// (polarity false) refusal.
func assertClosedBoolBuildGate(t *testing.T, fn *ast.FuncDecl, field string, polarity bool) {
	t.Helper()
	count := 0
	for _, cond := range ifConditions(fn) {
		if polarity {
			if !isNotSelector(cond, field) {
				continue
			}
		} else if !isInputSelector(cond, field) {
			continue
		}
		count++
	}
	if count != 1 {
		t.Fatalf("%s boolean build gate for %s matches %d ifs, want exactly 1", fn.Name.Name, field, count)
	}
}

// isNotSelector reports whether the expression is !input.Field.
func isNotSelector(expr ast.Expr, field string) bool {
	negation, ok := expr.(*ast.UnaryExpr)
	return ok && negation.Op == token.NOT && isInputSelector(negation.X, field)
}

// assertClosedBoolDecodeGate pins a closed boolean decode gate to
// exactly one `!ok || !v` (polarity true) or `!ok || v`
// (polarity false) refusal fed by rawBool(members["member"]).
func assertClosedBoolDecodeGate(t *testing.T, fn *ast.FuncDecl, member string, polarity bool) {
	t.Helper()
	value, ok := decodeValueName(fn, "rawBool", member)
	if !ok {
		t.Fatalf("%s has no unique rawBool(members[%q]) binding", fn.Name.Name, member)
	}
	count := 0
	for _, cond := range ifConditions(fn) {
		operands := orOperands(cond)
		if len(operands) != 2 || !isNotIdent(operands[0], "ok") {
			continue
		}
		if polarity {
			if !isNotIdent(operands[1], value) {
				continue
			}
		} else if !isIdent(operands[1], value) {
			continue
		}
		count++
	}
	if count != 1 {
		t.Fatalf("%s boolean decode gate for %s matches %d ifs, want exactly 1", fn.Name.Name, member, count)
	}
}

// assertOpenBoolBuildGate pins an open boolean build side to no
// value gate at all: no if condition may reference the field.
func assertOpenBoolBuildGate(t *testing.T, fn *ast.FuncDecl, field string) {
	t.Helper()
	for _, cond := range ifConditions(fn) {
		ast.Inspect(cond, func(node ast.Node) bool {
			expr, ok := node.(ast.Expr)
			if ok && isInputSelector(expr, field) {
				t.Fatalf("%s build side gates open field %s", fn.Name.Name, field)
			}
			return true
		})
	}
}

// assertOpenBoolDecodeGate pins an open boolean decode side to a
// type-only `!ok` refusal: no value operand may appear.
func assertOpenBoolDecodeGate(t *testing.T, fn *ast.FuncDecl, member string) {
	t.Helper()
	value, ok := decodeValueName(fn, "rawBool", member)
	if !ok {
		t.Fatalf("%s has no unique rawBool(members[%q]) binding", fn.Name.Name, member)
	}
	_ = value
	count := 0
	for _, cond := range ifConditions(fn) {
		if !isNotIdent(cond, "ok") {
			continue
		}
		count++
	}
	if count != 1 {
		t.Fatalf("%s open decode gate for %s matches %d type-only ifs, want exactly 1", fn.Name.Name, member, count)
	}
}

// findModesCallee returns the derived name of the shared modes
// checker: the unique function called with input.Modes in the
// read-back build function.
func findModesCallee(t *testing.T, buildFn *ast.FuncDecl, modesField string) string {
	t.Helper()
	var callee string
	count := 0
	ast.Inspect(buildFn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		if !isInputSelector(call.Args[0], modesField) {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		callee = fn.Name
		count++
		return true
	})
	if count != 1 {
		t.Fatalf("%s calls a modes checker %d times, want exactly 1", buildFn.Name.Name, count)
	}
	return callee
}

// assertModesGates pins the shared modes constant: the build side
// calls the derived checker with input.Modes, the decode side
// calls the same checker once with its decoded slice, and the
// checker refuses anything but length 2 with the oracle elements
// in order.
func assertModesGates(t *testing.T, funcs map[string]*ast.FuncDecl, buildFn, decodeFn *ast.FuncDecl, modes constantField) {
	t.Helper()
	callee := findModesCallee(t, buildFn, modes.field)
	calls := 0
	ast.Inspect(decodeFn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != callee {
			return true
		}
		if _, ok := call.Args[0].(*ast.Ident); !ok {
			t.Fatalf("%s calls %s with a non-identifier modes slice", decodeFn.Name.Name, callee)
		}
		calls++
		return true
	})
	if calls != 1 {
		t.Fatalf("%s calls %s %d times, want exactly 1", decodeFn.Name.Name, callee, calls)
	}
	checker, ok := funcs[callee]
	if !ok {
		t.Fatalf("modes checker %s not found", callee)
	}
	if checker.Type.Params == nil || len(checker.Type.Params.List) != 1 {
		t.Fatalf("modes checker %s has no single slice parameter", callee)
	}
	param := checker.Type.Params.List[0]
	if len(param.Names) != 1 {
		t.Fatalf("modes checker %s parameter is unnamed", callee)
	}
	slice := param.Names[0].Name
	conditions := ifConditions(checker)
	if len(conditions) != 1 {
		t.Fatalf("modes checker %s has %d ifs, want exactly 1", callee, len(conditions))
	}
	operands := orOperands(conditions[0])
	if len(operands) != 3 {
		t.Fatalf("modes checker %s gate has %d || operands, want 3", callee, len(operands))
	}
	length, ok := operands[0].(*ast.BinaryExpr)
	if !ok || length.Op != token.NEQ {
		t.Fatalf("modes checker %s first operand is not a length refusal", callee)
	}
	call, ok := length.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		t.Fatalf("modes checker %s first operand is not len(slice)", callee)
	}
	if fn, ok := call.Fun.(*ast.Ident); !ok || fn.Name != "len" || !isIdent(call.Args[0], slice) {
		t.Fatalf("modes checker %s first operand is not len(%s)", callee, slice)
	}
	two, ok := length.Y.(*ast.BasicLit)
	if !ok || two.Kind != token.INT || two.Value != "2" {
		t.Fatalf("modes checker %s length bound is not 2", callee)
	}
	for position, operand := range operands[1:] {
		comparison, ok := operand.(*ast.BinaryExpr)
		if !ok || comparison.Op != token.NEQ {
			t.Fatalf("modes checker %s element %d is not a != refusal", callee, position)
		}
		index, ok := comparison.X.(*ast.IndexExpr)
		if !ok || !isIdent(index.X, slice) {
			t.Fatalf("modes checker %s element %d does not index the slice", callee, position)
		}
		at, ok := index.Index.(*ast.BasicLit)
		if !ok || at.Kind != token.INT || at.Value != strconv.Itoa(position) {
			t.Fatalf("modes checker %s element %d has the wrong index", callee, position)
		}
		literal, ok := comparison.Y.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			t.Fatalf("modes checker %s element %d compares no literal", callee, position)
		}
		if got := unquoteBasicLit(t, literal); got != modesOracle[position] {
			t.Fatalf("modes checker %s element %d compares %q, want oracle %q", callee, position, got, modesOracle[position])
		}
	}
}

// assertTopLevelLiteralGates pins the schema/version gates of one
// top-level decode function: every `!ok || v != X` gate over a
// rawString(members[M]) binding is collected, and the collected
// set must equal the oracle exactly. The comparison target must
// be a const ident resolving to the oracle literal, and the
// condition must carry no string literal of its own, so a
// second admitted value fails the gate.
func assertTopLevelLiteralGates(t *testing.T, fn *ast.FuncDecl, oracle map[string]string, constValues map[string]string) {
	t.Helper()
	if fn == nil {
		t.Fatal("top-level decode function not found")
	}
	bindings := map[string]string{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		helper, ok := call.Fun.(*ast.Ident)
		if !ok || helper.Name != "rawString" {
			return true
		}
		index, ok := call.Args[0].(*ast.IndexExpr)
		if !ok {
			return true
		}
		receiver, ok := index.X.(*ast.Ident)
		if !ok || receiver.Name != "members" {
			return true
		}
		key, ok := index.Index.(*ast.BasicLit)
		if !ok || key.Kind != token.STRING {
			return true
		}
		value, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		bindings[value.Name] = unquoteBasicLit(t, key)
		return true
	})
	found := map[string]string{}
	for _, cond := range ifConditions(fn) {
		operands := orOperands(cond)
		if len(operands) != 2 || !isNotIdent(operands[0], "ok") {
			continue
		}
		comparison, ok := operands[1].(*ast.BinaryExpr)
		if !ok || comparison.Op != token.NEQ {
			continue
		}
		variable, ok := comparison.X.(*ast.Ident)
		if !ok {
			continue
		}
		member, ok := bindings[variable.Name]
		if !ok {
			continue
		}
		target, ok := comparison.Y.(*ast.Ident)
		if !ok {
			t.Fatalf("%s gate for %q compares a non-const target", fn.Name.Name, member)
		}
		resolved, ok := constValues[target.Name]
		if !ok {
			t.Fatalf("%s gate for %q references unknown const %s", fn.Name.Name, member, target.Name)
		}
		ast.Inspect(cond, func(node ast.Node) bool {
			if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				t.Fatalf("%s gate for %q carries string literal %s beside the const", fn.Name.Name, member, literal.Value)
			}
			return true
		})
		if _, dup := found[member]; dup {
			t.Fatalf("%s gates %q twice", fn.Name.Name, member)
		}
		found[member] = resolved
	}
	for member, literal := range oracle {
		got, ok := found[member]
		if !ok {
			t.Fatalf("%s has no single-literal gate for %q", fn.Name.Name, member)
		}
		if got != literal {
			t.Fatalf("%s gate for %q resolves to %q, want oracle %q", fn.Name.Name, member, got, literal)
		}
	}
	for member := range found {
		if _, ok := oracle[member]; !ok {
			t.Fatalf("%s gates unexpected member %q", fn.Name.Name, member)
		}
	}
}

// assertNoSwitch fails when the function holds any switch
// statement: a closed constant admits no second accepting case.
func assertNoSwitch(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			t.Fatalf("%s holds a switch statement", fn.Name.Name)
		}
		return true
	})
}

// foldCallNames holds selector names that widen a gate past one
// literal: prefix, substring, folding, and normalizing calls.
var foldCallNames = map[string]bool{
	"Contains": true, "ContainsAny": true, "HasPrefix": true, "HasSuffix": true,
	"EqualFold": true, "ToLower": true, "ToUpper": true, "ToTitle": true,
	"Trim": true, "TrimSpace": true, "TrimPrefix": true, "TrimSuffix": true,
	"Fold": true,
}

// assertNoFoldCalls fails when the function calls any widening
// selector: the gate must be a plain equality.
func assertNoFoldCalls(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if foldCallNames[selector.Sel.Name] {
			t.Fatalf("%s calls %s", fn.Name.Name, selector.Sel.Name)
		}
		return true
	})
}

// assertComparisonLiteralSet collects every string literal that
// appears as an == or != operand inside if conditions of the
// walked functions and pins the set exactly: any second admitted
// literal anywhere in these gates fails.
func assertComparisonLiteralSet(t *testing.T, funcs []*ast.FuncDecl, want map[string]bool) {
	t.Helper()
	found := map[string]bool{}
	for _, fn := range funcs {
		for _, cond := range ifConditions(fn) {
			ast.Inspect(cond, func(node ast.Node) bool {
				comparison, ok := node.(*ast.BinaryExpr)
				if !ok || (comparison.Op != token.EQL && comparison.Op != token.NEQ) {
					return true
				}
				for _, operand := range []ast.Expr{comparison.X, comparison.Y} {
					literal, ok := operand.(*ast.BasicLit)
					if ok && literal.Kind == token.STRING {
						found[unquoteBasicLit(t, literal)] = true
					}
				}
				return true
			})
		}
	}
	for literal := range want {
		if !found[literal] {
			t.Fatalf("comparison-literal set misses %q", literal)
		}
	}
	for literal := range found {
		if !want[literal] {
			t.Fatalf("comparison-literal set carries unexpected %q", literal)
		}
	}
}
