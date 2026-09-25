package clonereadback_test

import (
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

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// TestReadBackRoundTrip seals a full read-back manifest and a full
// validation report, decodes each, and asserts every field survived
// with literal schema, mode, and kind values.
func TestReadBackRoundTrip(t *testing.T) {
	staged, live := mustReadPair(t)
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	manifest := mustDecodeReadBack(t, stagedAuthority(t), sealed)
	if manifest.Manifest().Mode != "staged" {
		t.Fatalf("mode = %q, want staged", manifest.Manifest().Mode)
	}
	if len(manifest.Manifest().EvidenceObjects) != 2 {
		t.Fatalf("evidence_objects = %d, want 2", len(manifest.Manifest().EvidenceObjects))
	}
	if manifest.Manifest().EvidenceObjects[0].EvidenceKind != "native_sample" {
		t.Fatalf("row[0] kind = %q, want native_sample", manifest.Manifest().EvidenceObjects[0].EvidenceKind)
	}
	if manifest.Manifest().EvidenceObjects[1].EvidenceKind != "parser_trace" {
		t.Fatalf("row[1] kind = %q, want parser_trace", manifest.Manifest().EvidenceObjects[1].EvidenceKind)
	}
	if manifest.Manifest().ExpectedTargetNativeSession != manifest.Manifest().ObservedTargetNativeSession {
		t.Fatal("native Session IDs did not survive equal")
	}
	document := decodeDocument(t, sealed)
	if document["schema"] != "urn:ax:schema:clone-read-back-evidence-manifest" {
		t.Fatalf("schema = %v, want urn:ax:schema:clone-read-back-evidence-manifest", document["schema"])
	}
	if document["schema_version"] != "1.0.0" {
		t.Fatalf("schema_version = %v, want 1.0.0", document["schema_version"])
	}
	rsealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	report := mustDecodeReport(t, rsealed, staged, live)
	if !report.Valid {
		t.Fatal("valid did not survive true")
	}
	if len(report.Findings) != 1 || report.Findings[0].Severity != "info" {
		t.Fatal("findings did not survive")
	}
	rdocument := decodeDocument(t, rsealed)
	if rdocument["schema"] != "urn:ax:schema:clone-validation-report" {
		t.Fatalf("schema = %v, want urn:ax:schema:clone-validation-report", rdocument["schema"])
	}
	if rdocument["schema_version"] != "1.0.0" {
		t.Fatalf("schema_version = %v, want 1.0.0", rdocument["schema_version"])
	}
}

// memberProbe describes one shape member for the census grid: its
// admitted JSON type and whether the shape carries it.
type memberProbe struct {
	name string
	typ  string // string, uint, boolean, array, object
}

// censusShape binds one closed shape to its DERIVED member list, a
// baseline sealer, a navigator to the shape object inside the sealed
// document, and the production decode entry.
type censusShape struct {
	label   string
	owner   string // error owner phrase
	members []memberProbe
	seal    func(t *testing.T) []byte
	// navigate returns the shape object and a rewriter that
	// remarshals the mutated document.
	navigate func(t *testing.T, document map[string]any) map[string]any
	rewrite  func(t *testing.T, document map[string]any) []byte
	// dupZone locates the byte zone (object body) where a member
	// pair is duplicated: top document or first evidence row.
	dupZone func(t *testing.T, sealed []byte) (start, end int)
	decode  func([]byte) error
}

// irregularMembers maps the Go input fields whose JSON member names
// do not follow the mechanical snake_case rule, each quoted from
// pinned SPEC v0.7.0 §13.14.2: both top shapes carry
// expected/observed_target_native_session_id (with the _id suffix
// the Go field drops). Every other field maps mechanically. The
// exact-set cross-check against the sealed bytes fails if this
// table or the snake rule drifts from production.
var irregularMembers = map[string]string{
	"ExpectedTargetNativeSession": "expected_target_native_session_id",
	"ObservedTargetNativeSession": "observed_target_native_session_id",
}

// snakeMember renders the mechanical member name for a Go input
// field: a word boundary opens before every uppercase rune that
// follows a lowercase rune or a digit, and the words join with
// underscores in lowercase. Trailing acronym runs stay one word, so
// OperationID becomes operation_id and ParsedHeadIDs becomes
// parsed_head_ids.
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
// structs decode into.
var rowShapeOf = map[string]string{
	"EvidenceObjects": "evidence",
}

// deriveReflectedCensus walks the two production input types by
// reflection and returns every shape member with its JSON type.
// Nested structs and slices of structs recurse; any field kind the
// walker cannot classify fails loudly, so the set grows with the
// types and never shrinks silently.
func deriveReflectedCensus(t *testing.T) map[string][]memberProbe {
	t.Helper()
	census := map[string][]memberProbe{}
	walkCensusFields(t, reflect.TypeOf(clonereadback.ReadBackManifestInput{}), "readback", census)
	walkCensusFields(t, reflect.TypeOf(clonereadback.ValidationReportInput{}), "report", census)
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
	emit := func(typ string) {
		census[shape] = append(census[shape], memberProbe{name: member, typ: typ})
	}
	fieldType := field.Type
	switch fieldType.Kind() {
	case reflect.String:
		emit("string")
	case reflect.Bool:
		emit("boolean")
	case reflect.Uint64:
		emit("uint")
	case reflect.Slice, reflect.Array:
		element := fieldType.Elem()
		switch element.Kind() {
		case reflect.Uint8:
			// Raw JSON members: tuples and bindings are
			// objects, findings are an array. The field
			// name disambiguates; anything else fails
			// loudly.
			switch field.Name {
			case "Findings":
				emit("array")
			case "ObservedEnvironment", "WorkspaceBinding", "TargetEnvironment":
				emit("object")
			default:
				t.Fatalf("raw JSON field %s.%s is neither findings nor a known object: extend the walker", shape, field.Name)
			}
		case reflect.String:
			emit("array")
		case reflect.Struct:
			emit("array")
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
		value := fieldType.Elem()
		if value.Kind() != reflect.Interface || value.NumMethod() != 0 {
			t.Fatalf("unhandled map value kind %s at %s.%s: extend the walker", value.Kind(), shape, field.Name)
		}
		emit("object")
	default:
		t.Fatalf("unhandled kind %s at %s.%s: extend the walker", fieldType.Kind(), shape, field.Name)
	}
}

// deriveEnvelopeMembers derives the sealed-envelope set for one
// shape: sealed members minus the reflection-derived input
// members, with JSON types classified from the sealed values.
// Every top-level envelope field the sealer renders — schema,
// schema_version, the self id — joins the attack grid through
// this derivation, never through a hand-typed list. A value kind
// the classifier cannot type fails loudly.
func deriveEnvelopeMembers(t *testing.T, object map[string]any, derived []memberProbe) []memberProbe {
	t.Helper()
	known := map[string]bool{}
	for _, member := range derived {
		known[member.name] = true
	}
	var names []string
	for name := range object {
		if !known[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var envelope []memberProbe
	for _, name := range names {
		switch object[name].(type) {
		case string:
			envelope = append(envelope, memberProbe{name: name, typ: "string"})
		default:
			t.Fatalf("envelope member %q has unclassified JSON type %T: extend the classifier", name, object[name])
		}
	}
	return envelope
}

// readBackCensusShapes binds the three derived shapes to their
// sealers, navigators, byte zones, and production decode entries.
// The member lists come from reflection over the input types plus
// the derived sealed envelope, so every member — inputs and
// envelope alike — rides the full attack grid.
func readBackCensusShapes(t *testing.T) []censusShape {
	t.Helper()
	derived := deriveReflectedCensus(t)
	stagedAuth := stagedAuthority(t)
	staged, live := mustReadPair(t)
	readbackSealed := mustBuildReadBack(t, stagedAuth, validReadBackInput())
	reportSealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	readbackMembers := append(append([]memberProbe{}, derived["readback"]...), deriveEnvelopeMembers(t, decodeDocument(t, readbackSealed), derived["readback"])...)
	evidenceMembers := append(append([]memberProbe{}, derived["evidence"]...), deriveEnvelopeMembers(t, documentAt(t, decodeDocument(t, readbackSealed), "evidence_objects", 0), derived["evidence"])...)
	reportMembers := append(append([]memberProbe{}, derived["report"]...), deriveEnvelopeMembers(t, decodeDocument(t, reportSealed), derived["report"])...)
	rewriteDoc := func(t *testing.T, document map[string]any) []byte {
		return marshalDocument(t, document)
	}
	topZone := func(t *testing.T, sealed []byte) (int, int) {
		t.Helper()
		return 0, len(sealed)
	}
	return []censusShape{
		{
			label:   "readback",
			owner:   "read-back evidence manifest",
			members: readbackMembers,
			seal:    func(t *testing.T) []byte { return mustBuildReadBack(t, stagedAuthority(t), validReadBackInput()) },
			navigate: func(t *testing.T, document map[string]any) map[string]any {
				return document
			},
			rewrite: rewriteDoc,
			dupZone: topZone,
			decode: func(data []byte) error {
				_, err := clonereadback.DecodeReadBackEvidenceManifest(data, stagedAuth)
				return err
			},
		},
		{
			label:   "evidence",
			owner:   "evidence object",
			members: evidenceMembers,
			seal:    func(t *testing.T) []byte { return mustBuildReadBack(t, stagedAuthority(t), validReadBackInput()) },
			navigate: func(t *testing.T, document map[string]any) map[string]any {
				return documentAt(t, document, "evidence_objects", 0)
			},
			rewrite: rewriteDoc,
			dupZone: func(t *testing.T, sealed []byte) (int, int) {
				t.Helper()
				return firstRowZone(t, sealed, "evidence_objects")
			},
			decode: func(data []byte) error {
				_, err := clonereadback.DecodeReadBackEvidenceManifest(data, stagedAuth)
				return err
			},
		},
		{
			label:   "report",
			owner:   "clone validation report",
			members: reportMembers,
			seal:    func(t *testing.T) []byte { return mustBuildReport(t, validReportInput(staged, live), staged, live) },
			navigate: func(t *testing.T, document map[string]any) map[string]any {
				return document
			},
			rewrite: rewriteDoc,
			dupZone: topZone,
			decode: func(data []byte) error {
				_, err := clonereadback.DecodeValidationReport(data, staged, live)
				return err
			},
		},
	}
}

// requireRefusal asserts the production entry refused with the
// owner phrase and the literal gate fragment.
func requireRefusal(t *testing.T, err error, owner, fragment string) {
	t.Helper()
	if err == nil {
		t.Fatalf("entry admitted what it must refuse (want %q ... %q)", owner, fragment)
	}
	if !strings.Contains(err.Error(), owner) {
		t.Fatalf("error %q misses owner phrase %q", err.Error(), owner)
	}
	if !strings.Contains(err.Error(), fragment) {
		t.Fatalf("error %q misses literal %q", err.Error(), fragment)
	}
}

// TestMemberCensusExactSets cross-checks the derived member set
// against the sealed bytes: every derived member — reflected
// inputs and derived envelope alike — is present in the sealed
// shape, and the sealed shape carries no underived member. The
// inventory test pins the exact per-shape counts, so sealed bloat
// beyond the derivation reddens there.
func TestMemberCensusExactSets(t *testing.T) {
	for _, shape := range readBackCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			document := decodeDocument(t, shape.seal(t))
			object := shape.navigate(t, document)
			derived := map[string]bool{}
			for _, member := range shape.members {
				derived[member.name] = true
				if _, present := object[member.name]; !present {
					t.Errorf("derived member %q missing from sealed %s", member.name, shape.label)
				}
			}
			for name := range object {
				if !derived[name] {
					t.Errorf("sealed %s carries underived member %q", shape.label, name)
				}
			}
		})
	}
}

// TestMemberCensusUnknownMissing drives every derived member
// through the production decode entry twice: once deleted (missing
// refusal naming the member) and once beside an extra member
// (unknown refusal naming the extra).
func TestMemberCensusUnknownMissing(t *testing.T) {
	for _, shape := range readBackCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			for _, member := range shape.members {
				document := decodeDocument(t, shape.seal(t))
				object := shape.navigate(t, document)
				saved := object[member.name]
				delete(object, member.name)
				err := shape.decode(shape.rewrite(t, document))
				requireRefusal(t, err, shape.owner, "misses a required member \""+member.name+"\"")
				object[member.name] = saved
				object["zz_extra_member"] = "smuggled"
				err = shape.decode(shape.rewrite(t, document))
				requireRefusal(t, err, shape.owner, "carries unknown member \"zz_extra_member\"")
			}
		})
	}
}

// TestMemberCensusMiscased drives every derived member through the
// production decode entry renamed with an uppercase initial: the
// renamed member is unknown and the required member is missing.
func TestMemberCensusMiscased(t *testing.T) {
	for _, shape := range readBackCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			for _, member := range shape.members {
				document := decodeDocument(t, shape.seal(t))
				object := shape.navigate(t, document)
				miscased := strings.ToUpper(member.name[:1]) + member.name[1:]
				object[miscased] = object[member.name]
				delete(object, member.name)
				err := shape.decode(shape.rewrite(t, document))
				requireRefusal(t, err, shape.owner, "carries unknown member \""+miscased+"\"")
			}
		})
	}
}

// wrongTypeProbes lists the JSON values that are never the probed
// member type.
func wrongTypeProbes(typ string) []any {
	switch typ {
	case "string":
		return []any{float64(1), float64(1.5), true, nil, []any{}, map[string]any{}}
	case "uint":
		return []any{"1", float64(1.5), float64(-1), true, nil, []any{}, map[string]any{}}
	case "boolean":
		return []any{"true", float64(1), float64(0), nil, []any{}, map[string]any{}}
	case "array":
		return []any{"x", float64(1), true, nil, map[string]any{}}
	case "object":
		return []any{"x", float64(1), true, nil, []any{true}}
	default:
		return nil
	}
}

// TestMemberCensusWrongTypes drives every derived member through
// the production decode entry under six wrong JSON types: each is
// refused with the member name in the literal.
func TestMemberCensusWrongTypes(t *testing.T) {
	for _, shape := range readBackCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			for _, member := range shape.members {
				probes := wrongTypeProbes(member.typ)
				if len(probes) == 0 {
					t.Fatalf("member %s has unprobed type %q", member.name, member.typ)
				}
				for _, probe := range probes {
					document := decodeDocument(t, shape.seal(t))
					object := shape.navigate(t, document)
					object[member.name] = probe
					err := shape.decode(shape.rewrite(t, document))
					if err == nil {
						t.Fatalf("member %s admits wrong type %v", member.name, probe)
					}
					if !strings.Contains(err.Error(), shape.owner) || !strings.Contains(err.Error(), member.name) {
						t.Fatalf("member %s wrong-type error %q misses owner or member literal", member.name, err.Error())
					}
				}
			}
		})
	}
}

// scanJSONValue scans one JSON value starting at start (an object,
// array, string, or scalar) and returns the index just past its
// end.
func scanJSONValue(t *testing.T, raw []byte, start int) int {
	t.Helper()
	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(raw); index++ {
		char := raw[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			if depth == 0 {
				return index
			}
			depth--
			if depth == 0 {
				return index + 1
			}
		case ',':
			if depth == 0 {
				return index
			}
		}
	}
	t.Fatalf("unterminated JSON value at %d", start)
	return -1
}

// spliceDuplicateMember duplicates one member pair inside the shape
// byte zone, producing a document whose strict decode must refuse
// the duplicate. The duplicate carries the same bytes, so only the
// duplication — never a changed value — triggers the refusal.
func spliceDuplicateMember(t *testing.T, sealed []byte, zone func(t *testing.T, sealed []byte) (int, int), member string) []byte {
	t.Helper()
	start, end := zone(t, sealed)
	window := sealed[start:end]
	key := `"` + member + `":`
	at := strings.Index(string(window), key)
	if at < 0 {
		t.Fatalf("member %q not found in dup zone", member)
	}
	valueStart := start + at + len(key)
	valueEnd := scanJSONValue(t, sealed, valueStart)
	pair := sealed[valueStart-len(key) : valueEnd]
	mutated := append([]byte{}, sealed[:valueEnd]...)
	mutated = append(mutated, ',')
	mutated = append(mutated, pair...)
	mutated = append(mutated, sealed[valueEnd:]...)
	return mutated
}

// firstRowZone locates the first row object of an array member in
// the sealed bytes.
func firstRowZone(t *testing.T, sealed []byte, member string) (int, int) {
	t.Helper()
	key := `"` + member + `":[{`
	at := strings.Index(string(sealed), key)
	if at < 0 {
		t.Fatalf("array member %q with a first row not found", member)
	}
	start := at + len(key) - 1
	end := scanJSONValue(t, sealed, start)
	return start, end
}

// TestMemberCensusDuplicated drives every derived member through
// the production decode entry duplicated at the byte level: strict
// decoding refuses the duplicate.
func TestMemberCensusDuplicated(t *testing.T) {
	for _, shape := range readBackCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			sealed := shape.seal(t)
			for _, member := range shape.members {
				mutated := spliceDuplicateMember(t, sealed, shape.dupZone, member.name)
				err := shape.decode(mutated)
				requireRefusal(t, err, shape.owner, "duplicate member")
			}
		})
	}
}

// TestCensusShapeInventory pins the census coverage itself: 3
// shapes and the exact member counts retyped from the spec.
func TestCensusShapeInventory(t *testing.T) {
	want := map[string]int{
		"readback": 16, "evidence": 5, "report": 23,
	}
	shapes := readBackCensusShapes(t)
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
// carries at least 8 distinct derived-universe members at 40% or
// higher member purity (member literals over all string literals).
// Closure bodies are skipped: probe tables legitimately name one
// member per row inside mutate closures. The N-guard-literal-list
// harness row plants a 10-name list that must redden this test.
func TestMemberCensusNoLiteralList(t *testing.T) {
	universe := map[string]bool{}
	for _, shape := range readBackCensusShapes(t) {
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
						}
					}
					return
				}
				for _, child := range childNodes(node) {
					walk(child)
				}
			}
			for _, element := range composite.Elts {
				walk(element)
			}
			if len(members) >= 8 && stringLiterals > 0 && float64(len(members))/float64(stringLiterals) >= 0.4 {
				t.Errorf("%s holds a census-scale member list (%d members): derive it, do not type it", path, len(members))
			}
			return true
		})
	}
}

// childNodes lists the child nodes of one AST node for the literal
// walk.
func childNodes(node ast.Node) []ast.Node {
	var children []ast.Node
	ast.Inspect(node, func(child ast.Node) bool {
		if child == node {
			return true
		}
		children = append(children, child)
		return false
	})
	return children
}

// TestEnvelopeLiterals pins the literal schema/version/self member
// names from the pinned registry at both production decode entries:
// a swapped literal is a different schema, never this one.
func TestEnvelopeLiterals(t *testing.T) {
	staged, live := mustReadPair(t)
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range []struct {
		member string
		value  string
	}{
		{"schema", "urn:ax:schema:clone-validation-report"},
		{"schema", "urn:ax:schema:projection-plan"},
		{"schema_version", "2.0.0"},
		{"schema_version", "1.0"},
	} {
		document := decodeDocument(t, sealed)
		document[probe.member] = probe.value
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		requireRefusal(t, err, "read-back evidence manifest", probe.member)
	}
	rsealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	for _, probe := range []struct {
		member string
		value  string
	}{
		{"schema", "urn:ax:schema:clone-read-back-evidence-manifest"},
		{"schema", "urn:ax:schema:fidelity-report"},
		{"schema_version", "2.0.0"},
		{"schema_version", "1.0"},
	} {
		document := decodeDocument(t, rsealed)
		document[probe.member] = probe.value
		_, err := clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
		requireRefusal(t, err, "clone validation report", probe.member)
	}
}
