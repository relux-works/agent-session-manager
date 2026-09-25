package cloneplan_test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// TestPlanRoundTrip seals a full plan and a full manifest, decodes
// each, and asserts every field survived with literal schema,
// strategy, and profile values.
func TestPlanRoundTrip(t *testing.T) {
	input := validPlanInput()
	event := validSynthEventInput("roundtrip")
	event.InsertionAfterEvent = nil
	input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{event}
	input.CanonicalEventIDs = []string{fixtureDigest("ev-2"), fixtureDigest("ev-1")}
	sealed := mustBuildPlan(t, input)
	plan := mustDecodePlan(t, sealed)
	if plan.Strategy != "target_native_writer" {
		t.Fatalf("strategy = %q, want target_native_writer", plan.Strategy)
	}
	if plan.FidelityProfile != "maximal_safe" {
		t.Fatalf("profile = %q, want maximal_safe", plan.FidelityProfile)
	}
	if len(plan.CanonicalEventIDs) != 2 {
		t.Fatalf("canonical_event_ids = %d, want 2", len(plan.CanonicalEventIDs))
	}
	if plan.CanonicalEventIDs[0].String() != fixtureDigest("ev-2") {
		t.Fatal("canonical_event_ids order did not survive: session order is preserved, not sorted")
	}
	if plan.SynthesizedEvents[0].InsertionAfterEvent != nil {
		t.Fatal("null insertion anchor did not survive")
	}
	if !plan.ResumePlan.BoundedContinuationTurnRequired {
		t.Fatal("bounded_continuation_turn_required did not survive")
	}
	document := decodeDocument(t, sealed)
	if document["schema"] != "urn:ax:schema:projection-plan" {
		t.Fatalf("schema = %v, want urn:ax:schema:projection-plan", document["schema"])
	}
	if document["schema_version"] != "1.0.0" {
		t.Fatalf("schema_version = %v, want 1.0.0", document["schema_version"])
	}
	msealed := mustBuildManifest(t, validManifestInput())
	manifest := mustDecodeManifest(t, msealed)
	if len(manifest.Entries) != 2 || manifest.TotalBytes != 128 {
		t.Fatal("manifest entries/total did not survive")
	}
	if manifest.Entries[0].Kind != "directory" || manifest.Entries[1].Kind != "blob" {
		t.Fatal("manifest entry branches did not survive")
	}
	mdocument := decodeDocument(t, msealed)
	if mdocument["schema"] != "urn:ax:schema:clone-projected-object-manifest" {
		t.Fatalf("schema = %v, want urn:ax:schema:clone-projected-object-manifest", mdocument["schema"])
	}
	if mdocument["schema_version"] != "1.0.0" {
		t.Fatalf("schema_version = %v, want 1.0.0", mdocument["schema_version"])
	}
}

// memberProbe describes one shape member for the census grid: its
// admitted JSON type and whether null admits.
type memberProbe struct {
	name     string
	typ      string // string, uint, boolean, array, object
	nullable bool
}

// censusShape binds one closed shape to its DERIVED member list, a
// baseline sealer, a navigator to the shape object inside the sealed
// document, and the production decode entry.
type censusShape struct {
	label    string
	owner    string // error owner phrase
	members  []memberProbe
	seal     func(t *testing.T) []byte
	navigate func(t *testing.T, document map[string]any) map[string]any
	path     []dupStep // byte path for the duplicate splicer
	decode   func([]byte) error
}

// irregularMembers maps the four Go input fields whose JSON member
// names do not follow the mechanical snake_case rule, each quoted
// from pinned SPEC v0.7.0 §13.14.2: both top shapes carry
// expected_target_native_session_id (with the _id suffix the Go
// field drops), the mapping row carries target_resource_keys
// (plural) and expected_disposition (full word), and the synthesized
// event carries insertion_after_event_id (with the _id suffix).
// Every other field maps mechanically. The exact-set cross-check
// against the sealed bytes fails if this table or the snake rule
// drifts from production.
var irregularMembers = map[string]string{
	"ExpectedTargetNativeSession": "expected_target_native_session_id",
	"TargetResourceKey":           "target_resource_keys",
	"ExpectedDispos":              "expected_disposition",
	"InsertionAfterEvent":         "insertion_after_event_id",
}

// snakeMember renders the mechanical member name for a Go input
// field: a word boundary opens before every uppercase rune that
// follows a lowercase rune or a digit, and the words join with
// underscores in lowercase. Trailing acronym runs stay one word, so
// OperationID becomes operation_id and CanonicalEventIDs becomes
// canonical_event_ids.
func snakeMember(field string) string {
	var words []string
	start := 0
	runes := []rune(field)
	for index := 1; index < len(runes); index++ {
		if unicode.IsUpper(runes[index]) && (unicode.IsLower(runes[index-1]) || unicode.IsDigit(runes[index-1])) {
			words = append(words, string(runes[start:index]))
			start = index
		}
	}
	words = append(words, string(runes[start:]))
	return strings.ToLower(strings.Join(words, "_"))
}

// rowShapeOf maps a container field to the shape label its element
// or nested struct decodes into.
var rowShapeOf = map[string]string{
	"TransactionPlan":   "transaction",
	"ReadBackPlan":      "read-back",
	"ResumePlan":        "resume",
	"RollbackPlan":      "rollback",
	"ItemMappings":      "mapping",
	"TargetOperations":  "operation",
	"ExpectedResources": "resource",
	"SynthesizedEvents": "synth-event",
	"RequiredContracts": "contract",
	"Entries":           "blob-entry",
}

// deriveReflectedCensus walks the two production input types by
// reflection and returns every shape member with its JSON type and
// nullability. Nested structs, slices of structs, and maps recurse;
// any field kind the walker cannot classify fails loudly, so the set
// grows with the types and never shrinks silently. Directory rows
// share the entry input type and are derived from it in
// planCensusShapes.
func deriveReflectedCensus(t *testing.T) map[string][]memberProbe {
	t.Helper()
	census := map[string][]memberProbe{}
	walkCensusFields(t, reflect.TypeOf(cloneplan.ProjectionPlanInput{}), "plan", census)
	walkCensusFields(t, reflect.TypeOf(cloneplan.ProjectedManifestInput{}), "manifest", census)
	return census
}

func walkCensusFields(t *testing.T, typ reflect.Type, shape string, census map[string][]memberProbe) {
	t.Helper()
	for index := 0; index < typ.NumField(); index++ {
		walkCensusField(t, typ.Field(index), shape, census)
	}
}

func walkCensusField(t *testing.T, field reflect.StructField, shape string, census map[string][]memberProbe) {
	t.Helper()
	if field.PkgPath != "" {
		t.Fatalf("unexported field %s.%s: extend the walker before probing it", shape, field.Name)
	}
	member := snakeMember(field.Name)
	if irregular, ok := irregularMembers[field.Name]; ok {
		member = irregular
	}
	emit := func(typ string, nullable bool) {
		census[shape] = append(census[shape], memberProbe{name: member, typ: typ, nullable: nullable})
	}
	fieldType := field.Type
	switch fieldType.Kind() {
	case reflect.String:
		emit("string", false)
	case reflect.Bool:
		emit("boolean", false)
	case reflect.Uint64:
		emit("uint", false)
	case reflect.Pointer:
		switch fieldType.Elem().Kind() {
		case reflect.String:
			emit("string", true)
		case reflect.Uint64:
			emit("uint", true)
		default:
			t.Fatalf("unhandled pointer %s.%s: extend the walker", shape, field.Name)
		}
	case reflect.Slice, reflect.Array:
		element := fieldType.Elem()
		switch element.Kind() {
		case reflect.Uint8:
			// Raw JSON object members: tuples, workspace, limits.
			emit("object", false)
		case reflect.String, reflect.Uint64:
			emit("array", false)
		case reflect.Struct:
			emit("array", false)
			rowShape, ok := rowShapeOf[field.Name]
			if !ok {
				t.Fatalf("slice of structs %s.%s has no row shape: extend the walker", shape, field.Name)
			}
			walkCensusFields(t, element, rowShape, census)
		default:
			t.Fatalf("unhandled slice element kind %s at %s.%s: extend the walker", element.Kind(), shape, field.Name)
		}
	case reflect.Map:
		if fieldType.Key().Kind() != reflect.String {
			t.Fatalf("unhandled map key kind %s at %s.%s: extend the walker", fieldType.Key().Kind(), shape, field.Name)
		}
		switch value := fieldType.Elem(); value.Kind() {
		case reflect.Slice:
			if value.Elem().Kind() != reflect.String {
				t.Fatalf("unhandled map slice element at %s.%s: extend the walker", shape, field.Name)
			}
			emit("object", false)
		case reflect.Interface:
			if value.NumMethod() != 0 {
				t.Fatalf("unhandled map interface at %s.%s: extend the walker", shape, field.Name)
			}
			emit("object", false)
		default:
			t.Fatalf("unhandled map value kind %s at %s.%s: extend the walker", value.Kind(), shape, field.Name)
		}
	case reflect.Struct:
		emit("object", false)
		rowShape, ok := rowShapeOf[field.Name]
		if !ok {
			t.Fatalf("nested struct %s.%s has no row shape: extend the walker", shape, field.Name)
		}
		walkCensusFields(t, fieldType, rowShape, census)
	default:
		t.Fatalf("unhandled kind %s at %s.%s: extend the walker", fieldType.Kind(), shape, field.Name)
	}
}

// sealedShape is the member set observed in the sealed baseline for
// one shape: names with their JSON value types.
type sealedShape struct {
	types map[string]string // string, uint, boolean, array, object, null
}

func sealedTypeOf(value any) string {
	switch typed := value.(type) {
	case string:
		return "string"
	case float64:
		if typed != float64(uint64(typed)) {
			return "non-integral-number"
		}
		return "uint"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("unexpected-%T", value)
	}
}

func planCensusShapes(t *testing.T) []censusShape {
	t.Helper()
	derived := deriveReflectedCensus(t)

	sealPlan := func(t *testing.T) []byte {
		t.Helper()
		input := validPlanInput()
		input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{validSynthEventInput("census")}
		return mustBuildPlan(t, input)
	}
	sealManifest := func(t *testing.T) []byte { t.Helper(); return mustBuildManifest(t, validManifestInput()) }
	top := func(t *testing.T, document map[string]any) map[string]any { return document }
	at := func(member string) func(t *testing.T, document map[string]any) map[string]any {
		return func(t *testing.T, document map[string]any) map[string]any {
			t.Helper()
			nested, ok := document[member].(map[string]any)
			if !ok {
				t.Fatalf("member %q is not an object in the sealed baseline", member)
			}
			return nested
		}
	}
	atIndex := func(member string, index int) func(t *testing.T, document map[string]any) map[string]any {
		return func(t *testing.T, document map[string]any) map[string]any {
			t.Helper()
			array, ok := document[member].([]any)
			if !ok || index >= len(array) {
				t.Fatalf("member %q has no row %d in the sealed baseline", member, index)
			}
			row, ok := array[index].(map[string]any)
			if !ok {
				t.Fatalf("member %q row %d is not an object", member, index)
			}
			return row
		}
	}
	decodePlan := func(raw []byte) error { _, err := cloneplan.DecodeProjectionPlan(raw); return err }
	decodeManifest := func(raw []byte) error { _, err := cloneplan.DecodeProjectedObjectManifest(raw); return err }

	// Synthesized top-level members (schema, version, self id) have
	// no input field: derive them as the sealed-minus-reflected set
	// difference, with JSON types read off the sealed values. They
	// are non-nullable: the wrong-type grid probes null against each
	// one, so a null-admitting synthesized member fails there.
	synthesized := func(t *testing.T, label string, seal func(t *testing.T) []byte, navigate func(t *testing.T, document map[string]any) map[string]any) []memberProbe {
		t.Helper()
		target := navigate(t, decodeDocument(t, seal(t)))
		known := map[string]bool{}
		for _, probe := range derived[label] {
			known[probe.name] = true
		}
		var extra []memberProbe
		for name, value := range target {
			if known[name] {
				continue
			}
			extra = append(extra, memberProbe{name: name, typ: sealedTypeOf(value)})
		}
		sort.Slice(extra, func(i, j int) bool { return extra[i].name < extra[j].name })
		return extra
	}
	planMembers := append(append([]memberProbe{}, derived["plan"]...), synthesized(t, "plan", sealPlan, top)...)
	manifestMembers := append(append([]memberProbe{}, derived["manifest"]...), synthesized(t, "manifest", sealManifest, top)...)

	// Directory rows share the entry input type: derive them as the
	// reflected entry members present in the sealed directory row,
	// keeping reflected types and nullability.
	directoryRow := atIndex("entries", 0)(t, decodeDocument(t, sealManifest(t)))
	byName := map[string]memberProbe{}
	for _, probe := range derived["blob-entry"] {
		byName[probe.name] = probe
	}
	var directoryMembers []memberProbe
	for name := range directoryRow {
		probe, ok := byName[name]
		if !ok {
			t.Fatalf("sealed directory row carries %q outside the reflected entry census", name)
		}
		directoryMembers = append(directoryMembers, probe)
	}
	sort.Slice(directoryMembers, func(i, j int) bool { return directoryMembers[i].name < directoryMembers[j].name })

	return []censusShape{
		{"plan", "projection plan", planMembers, sealPlan, top, nil, decodePlan},
		{"manifest", "projected object manifest", manifestMembers, sealManifest, top, nil, decodeManifest},
		{"mapping", "projection item mapping[0]", derived["mapping"], sealPlan, atIndex("item_mappings", 0), []dupStep{{"item_mappings", 0}}, decodePlan},
		{"operation", "projection target operation[0]", derived["operation"], sealPlan, atIndex("target_operations", 0), []dupStep{{"target_operations", 0}}, decodePlan},
		{"resource", "expected target resource[0]", derived["resource"], sealPlan, atIndex("expected_resources", 0), []dupStep{{"expected_resources", 0}}, decodePlan},
		{"synth-event", "synthesized projection event[0]", derived["synth-event"], sealPlan, atIndex("synthesized_events", 0), []dupStep{{"synthesized_events", 0}}, decodePlan},
		{"transaction", "transaction plan", derived["transaction"], sealPlan, at("transaction_plan"), []dupStep{{"transaction_plan", -1}}, decodePlan},
		{"read-back", "read-back plan", derived["read-back"], sealPlan, at("read_back_plan"), []dupStep{{"read_back_plan", -1}}, decodePlan},
		{"resume", "resume projection plan", derived["resume"], sealPlan, at("resume_plan"), []dupStep{{"resume_plan", -1}}, decodePlan},
		{"rollback", "rollback plan", derived["rollback"], sealPlan, at("rollback_plan"), []dupStep{{"rollback_plan", -1}}, decodePlan},
		{"contract", "contract requirement[0]", derived["contract"], sealPlan, atIndex("required_contracts", 0), []dupStep{{"required_contracts", 0}}, decodePlan},
		// Manifest entries carry one directory row then one blob row in the baseline.
		{"blob-entry", "projected object entry[1]", derived["blob-entry"], sealManifest, atIndex("entries", 1), []dupStep{{"entries", 1}}, decodeManifest},
		{"directory-entry", "projected object entry[0]", directoryMembers, sealManifest, atIndex("entries", 0), []dupStep{{"entries", 0}}, decodeManifest},
	}
}

// TestMemberCensusExactSets pins that the reflection-derived census
// equals the sealed member set on every shape in both directions:
// every derived member is sealed with a matching JSON type, and the
// sealed shape carries nothing else. The synthesized top-level count
// (schema, version, self id) and the directory/blob partition fall
// out of the same comparison.
func TestMemberCensusExactSets(t *testing.T) {
	shapes := planCensusShapes(t)
	for _, shape := range shapes {
		t.Run(shape.label, func(t *testing.T) {
			target := shape.navigate(t, decodeDocument(t, shape.seal(t)))
			derived := map[string]memberProbe{}
			for _, probe := range shape.members {
				if _, dup := derived[probe.name]; dup {
					t.Fatalf("derived member %q twice in shape %s", probe.name, shape.label)
				}
				derived[probe.name] = probe
			}
			if len(target) != len(derived) {
				t.Fatalf("sealed %s carries %d members, derived %d", shape.label, len(target), len(derived))
			}
			for name, probe := range derived {
				value, present := target[name]
				if !present {
					t.Errorf("derived member %q of %s is not sealed", name, shape.label)
					continue
				}
				sealed := sealedTypeOf(value)
				if sealed == "null" {
					if !probe.nullable {
						t.Errorf("sealed %s member %q is null but the reflected kind is non-nullable", shape.label, name)
					}
					continue
				}
				if sealed != probe.typ {
					t.Errorf("sealed %s member %q has JSON type %s, reflected %s", shape.label, name, sealed, probe.typ)
				}
			}
			for name := range target {
				if _, ok := derived[name]; !ok {
					t.Errorf("sealed %s carries extra member %q outside the derived census", shape.label, name)
				}
			}
		})
	}
}

// TestMemberCensusUnknownMissing pins that every shape refuses one
// unknown member and refuses each derived member removed, naming the
// member.
func TestMemberCensusUnknownMissing(t *testing.T) {
	for _, shape := range planCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			sealed := shape.seal(t)
			document := decodeDocument(t, sealed)
			shape.navigate(t, document)["zz_unknown_member"] = "x"
			err := shape.decode(marshalDocument(t, document))
			requireRefusal(t, err, shape.owner, "zz_unknown_member")
			for _, member := range shape.members {
				document := decodeDocument(t, sealed)
				delete(shape.navigate(t, document), member.name)
				err := shape.decode(marshalDocument(t, document))
				requireRefusal(t, err, shape.owner, member.name, "misses a required member")
			}
		})
	}
}

// caseVariant is one renamed-member probe: the class label and the
// renamed member.
type caseVariant struct {
	label string
	value string
}

// caseVariantsOf returns every case variant class for one member
// name: upper, lower, first-rune title, every-segment title, and a
// single toggled rune at the first and last cased positions.
// Variants identical to the original (lowercase names have no lower
// variant) and duplicates collapse; at least upper, first-rune
// title, and last-rune toggle always differ for these names.
func caseVariantsOf(member string) []caseVariant {
	upperFirst := func(value string) string {
		runes := []rune(value)
		for index, rune := range runes {
			if unicode.IsLower(rune) {
				runes[index] = unicode.ToUpper(rune)
				break
			}
		}
		return string(runes)
	}
	titleSegments := func(value string) string {
		segments := strings.Split(value, "_")
		for index, segment := range segments {
			segments[index] = upperFirst(segment)
		}
		return strings.Join(segments, "_")
	}
	toggle := func(value string, fromEnd bool) string {
		runes := []rune(value)
		order := make([]int, len(runes))
		for index := range runes {
			order[index] = index
		}
		if fromEnd {
			for left, right := 0, len(order)-1; left < right; left, right = left+1, right-1 {
				order[left], order[right] = order[right], order[left]
			}
		}
		for _, index := range order {
			rune := runes[index]
			if unicode.IsLower(rune) {
				runes[index] = unicode.ToUpper(rune)
				return string(runes)
			}
			if unicode.IsUpper(rune) {
				runes[index] = unicode.ToLower(rune)
				return string(runes)
			}
		}
		return value
	}
	candidates := []caseVariant{
		{"upper", strings.ToUpper(member)},
		{"lower", strings.ToLower(member)},
		{"title-first", upperFirst(member)},
		{"title-segments", titleSegments(member)},
		{"toggle-first", toggle(member, false)},
		{"toggle-last", toggle(member, true)},
	}
	seen := map[string]bool{member: true}
	var variants []caseVariant
	for _, candidate := range candidates {
		if seen[candidate.value] {
			continue
		}
		seen[candidate.value] = true
		variants = append(variants, candidate)
	}
	return variants
}

// TestMemberCensusMiscased pins that every derived member refuses in
// every case variant class: member names are case-exact at the
// decode entry. The first-rune title class is the reviewer's
// Titlecase plant (Strategy for strategy).
func TestMemberCensusMiscased(t *testing.T) {
	for _, shape := range planCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			sealed := shape.seal(t)
			for _, member := range shape.members {
				variants := caseVariantsOf(member.name)
				if len(variants) < 3 {
					t.Fatalf("member %q yields %d case variants, want at least 3: the probe is wrong", member.name, len(variants))
				}
				for _, variant := range variants {
					document := decodeDocument(t, sealed)
					target := shape.navigate(t, document)
					target[variant.value] = target[member.name]
					delete(target, member.name)
					err := shape.decode(marshalDocument(t, document))
					if err == nil {
						t.Errorf("%s member %q admits %s variant %q, want refusal", shape.label, member.name, variant.label, variant.value)
						continue
					}
					// Entry rows dispatch on kind before the
					// unknown check runs, so a renamed kind
					// reports missing instead of unknown; both
					// are structural refusals naming the shape.
					if (shape.label == "blob-entry" || shape.label == "directory-entry") && member.name == "kind" {
						requireRefusal(t, err, shape.owner, "misses a required member")
						continue
					}
					requireRefusal(t, err, shape.owner, "carries unknown member")
				}
			}
		})
	}
}

// TestMemberCensusWrongTypes pins that every derived member refuses
// every wrong JSON type with the member (or its shape owner) named.
// Cells whose probe type equals the admitted type are skipped:
// same-type values are covered by the vocabulary, bound, and branch
// tests.
func TestMemberCensusWrongTypes(t *testing.T) {
	probes := []struct {
		label string
		typ   string
		value any
	}{
		{"object", "object", map[string]any{"zz": 1}},
		{"array", "array", []any{7}},
		{"string", "string", "zz-top"},
		{"number", "uint", float64(7)},
		{"boolean", "boolean", true},
		{"null", "null", nil},
	}
	// Nested plan components report their owner phrase instead of
	// the member name when the whole member is the wrong type.
	alts := map[string]string{
		"transaction_plan": "transaction plan",
		"read_back_plan":   "read-back plan",
		"resume_plan":      "resume projection plan",
		"rollback_plan":    "rollback plan",
	}
	for _, shape := range planCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			sealed := shape.seal(t)
			for _, member := range shape.members {
				for _, probe := range probes {
					if probe.typ == member.typ {
						continue
					}
					if probe.typ == "null" && member.nullable {
						continue
					}
					document := decodeDocument(t, sealed)
					shape.navigate(t, document)[member.name] = probe.value
					err := shape.decode(marshalDocument(t, document))
					if err == nil {
						t.Errorf("%s member %q admits %s probe, want refusal", shape.label, member.name, probe.label)
						continue
					}
					detail := err.Error()
					alt, hasAlt := alts[member.name]
					if !strings.Contains(detail, member.name) && !strings.Contains(detail, shape.owner) && !(hasAlt && strings.Contains(detail, alt)) {
						t.Errorf("%s member %q %s probe: error %q names neither the member nor %q", shape.label, member.name, probe.label, detail, shape.owner)
					}
				}
			}
		})
	}
}

// dupStep is one descent step for the duplicate splicer: the member
// whose value to enter, and the array index to enter (-1 keeps the
// object itself).
type dupStep struct {
	member string
	index  int
}

// spliceDuplicateMember returns the sealed bytes with one member
// pair duplicated inside the object at path: the pair bytes repeat
// verbatim, so the strict frame must refuse a duplicate member. The
// walk tracks the byte path instead of first-occurrence search, so
// same-named members in other shapes cannot misplace the surgery.
func spliceDuplicateMember(t *testing.T, sealed []byte, path []dupStep, member string) []byte {
	t.Helper()
	raw := string(sealed)
	end := len(raw)

	skipSpace := func(offset int) int {
		for offset < end && (raw[offset] == ' ' || raw[offset] == '\n' || raw[offset] == '\r' || raw[offset] == '\t') {
			offset++
		}
		return offset
	}
	// scanValue returns the offset just past the JSON value starting
	// at offset.
	var scanValue func(offset int) int
	scanString := func(offset int) int {
		// offset points at the opening quote.
		offset++
		for offset < end {
			switch raw[offset] {
			case '\\':
				offset += 2
			case '"':
				return offset + 1
			default:
				offset++
			}
		}
		t.Fatalf("unterminated string at %d", offset)
		return end
	}
	scanValue = func(offset int) int {
		offset = skipSpace(offset)
		if offset >= end {
			t.Fatalf("value expected at end of input")
		}
		switch raw[offset] {
		case '"':
			return scanString(offset)
		case '{', '[':
			open, close := raw[offset], byte('}')
			if raw[offset] == '[' {
				close = ']'
			}
			depth := 0
			for offset < end {
				switch raw[offset] {
				case '"':
					offset = scanString(offset)
					continue
				case open:
					depth++
				case close:
					depth--
					if depth == 0 {
						return offset + 1
					}
				}
				offset++
			}
			t.Fatalf("unterminated %c", open)
		default:
			for offset < end && raw[offset] != ',' && raw[offset] != ']' && raw[offset] != '}' {
				offset++
			}
			return offset
		}
		return end
	}
	// scanObjectPairs yields the byte ranges of every pair of the
	// object starting at offset (which points at '{').
	scanObjectPairs := func(offset int) [][2]int {
		offset = skipSpace(offset)
		if raw[offset] != '{' {
			t.Fatalf("object expected at %d", offset)
		}
		offset++
		var pairs [][2]int
		for {
			offset = skipSpace(offset)
			if raw[offset] == '}' {
				return pairs
			}
			pairStart := offset
			keyEnd := scanString(offset)
			_ = keyEnd
			offset = skipSpace(keyEnd)
			if raw[offset] != ':' {
				t.Fatalf("colon expected at %d", offset)
			}
			offset = scanValue(offset + 1)
			pairs = append(pairs, [2]int{pairStart, offset})
			offset = skipSpace(offset)
			if raw[offset] == ',' {
				offset++
				continue
			}
			if raw[offset] == '}' {
				return pairs
			}
			t.Fatalf("comma or close expected at %d", offset)
		}
	}
	pairKey := func(pair [2]int) string {
		// The pair starts with the key string; decode exactly its
		// extent.
		var key string
		if err := json.Unmarshal([]byte(raw[pair[0]:scanString(pair[0])]), &key); err != nil {
			t.Fatalf("decode pair key: %v", err)
		}
		return key
	}
	// descend walks the path and returns the object offset.
	offset := skipSpace(0)
	for _, step := range path {
		pairs := scanObjectPairs(offset)
		found := false
		for _, pair := range pairs {
			if pairKey(pair) != step.member {
				continue
			}
			keyEnd := scanString(pair[0])
			valueStart := skipSpace(keyEnd + 1)
			if step.index < 0 {
				offset = valueStart
			} else {
				// Enter the array element.
				arrayOffset := skipSpace(valueStart)
				if raw[arrayOffset] != '[' {
					t.Fatalf("array expected at path member %q", step.member)
				}
				arrayOffset++
				for element := 0; ; element++ {
					arrayOffset = skipSpace(arrayOffset)
					elementEnd := scanValue(arrayOffset)
					if element == step.index {
						offset = arrayOffset
						_ = elementEnd
						break
					}
					arrayOffset = skipSpace(elementEnd)
					if raw[arrayOffset] != ',' {
						t.Fatalf("path member %q has no row %d", step.member, step.index)
					}
					arrayOffset++
				}
			}
			found = true
			break
		}
		if !found {
			t.Fatalf("path member %q not found", step.member)
		}
	}
	pairs := scanObjectPairs(offset)
	for _, pair := range pairs {
		if pairKey(pair) != member {
			continue
		}
		dup := raw[pair[0]:pair[1]]
		mutated := raw[:pair[1]] + "," + dup + raw[pair[1]:]
		if strings.Count(mutated, `"`+member+`":`) != strings.Count(raw, `"`+member+`":`)+1 {
			t.Fatalf("splice for %q added %d occurrences, want exactly one", member, strings.Count(mutated, `"`+member+`":`)-strings.Count(raw, `"`+member+`":`))
		}
		return []byte(mutated)
	}
	t.Fatalf("member %q not found in the shape object", member)
	return nil
}

// TestMemberCensusDuplicated pins that a duplicated member refuses at
// the strict frame for EVERY derived member of every shape: raw JSON
// surgery duplicates the pair inside the shape object, and the
// refusal names the shape owner and the duplicate frame fault.
func TestMemberCensusDuplicated(t *testing.T) {
	for _, shape := range planCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			sealed := shape.seal(t)
			for _, member := range shape.members {
				mutated := spliceDuplicateMember(t, sealed, shape.path, member.name)
				err := shape.decode(mutated)
				requireRefusal(t, err, shape.owner, "duplicate member")
			}
		})
	}
}

// TestCensusShapeInventory pins the census coverage itself: 13
// shapes and the exact member counts retyped from the spec.
func TestCensusShapeInventory(t *testing.T) {
	want := map[string]int{
		"plan": 36, "manifest": 10, "mapping": 6, "operation": 5,
		"resource": 6, "synth-event": 4, "transaction": 4,
		"read-back": 5, "resume": 4, "rollback": 4, "contract": 2,
		"blob-entry": 7, "directory-entry": 4,
	}
	shapes := planCensusShapes(t)
	if len(shapes) != len(want) {
		t.Fatalf("census shapes = %d, want %d", len(shapes), len(want))
	}
	var labels []string
	for _, shape := range shapes {
		labels = append(labels, shape.label)
		if len(shape.members) != want[shape.label] {
			t.Errorf("shape %s members = %d, want %d", shape.label, len(shape.members), want[shape.label])
		}
	}
	sort.Strings(labels)
	t.Logf("census shapes (%d): %s", len(labels), strings.Join(labels, ", "))
}

// TestMemberCensusNoLiteralList guards the derivation itself: no
// composite literal in the package test files may hold a
// census-scale string-literal member list. A composite trips when it
// carries at least 10 distinct derived-universe members at 40% or
// higher member purity (member literals over all string literals).
// Closure bodies are skipped: probe tables legitimately name one
// member per row inside mutate closures. The committed survey
// (results) shows every legitimate composite well under the line,
// and the N-guard-literal-list harness row plants a 36-name list
// that must redden this test.
func TestMemberCensusNoLiteralList(t *testing.T) {
	universe := map[string]bool{}
	for _, shape := range planCensusShapes(t) {
		for _, member := range shape.members {
			universe[member.name] = true
		}
	}
	if len(universe) == 0 {
		t.Fatal("derived member universe is empty")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(file), "*_test.go"))
	if err != nil {
		t.Fatalf("glob test files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			composite, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			members := map[string]bool{}
			memberLiterals := 0
			stringLiterals := 0
			var walk func(node ast.Node)
			walk = func(node ast.Node) {
				if node == nil {
					return
				}
				if _, isFunc := node.(*ast.FuncLit); isFunc {
					return
				}
				if literal, isLit := node.(*ast.BasicLit); isLit && literal.Kind == token.STRING {
					if value, err := strconv.Unquote(literal.Value); err == nil {
						stringLiterals++
						if universe[value] {
							members[value] = true
							memberLiterals++
						}
					}
					return
				}
				switch typed := node.(type) {
				case *ast.CompositeLit:
					for _, element := range typed.Elts {
						walk(element)
					}
				case *ast.KeyValueExpr:
					walk(typed.Key)
					walk(typed.Value)
				case *ast.CallExpr:
					walk(typed.Fun)
					for _, argument := range typed.Args {
						walk(argument)
					}
				case *ast.ParenExpr:
					walk(typed.X)
				case *ast.UnaryExpr:
					walk(typed.X)
				case *ast.BinaryExpr:
					walk(typed.X)
					walk(typed.Y)
				case *ast.IndexExpr:
					walk(typed.X)
					walk(typed.Index)
				case *ast.SelectorExpr:
					walk(typed.X)
				case *ast.StarExpr:
					walk(typed.X)
				}
			}
			for _, element := range composite.Elts {
				walk(element)
			}
			if len(members) >= 10 && stringLiterals > 0 && float64(memberLiterals)/float64(stringLiterals) >= 0.4 {
				names := make([]string, 0, len(members))
				for name := range members {
					names = append(names, name)
				}
				sort.Strings(names)
				t.Errorf("%s:%d: composite literal holds %d member names at %.0f%% purity, want a derived census: %s",
					filepath.Base(path), fset.Position(composite.Pos()).Line, len(members),
					100*float64(memberLiterals)/float64(stringLiterals), strings.Join(names, ", "))
			}
			return true
		})
	}
}
