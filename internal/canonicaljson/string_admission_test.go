package canonicaljson

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file pins the SEMANTICS of the string-admission matcher, which is the
// second factor of the admitted set.
//
// The failure it exists to prevent: closed-vocabularies.md pins the MEMBER LIST
// of every closed vocabulary, and constraint-enumeration.md pins every closed
// member set, but neither says anything about the COMPARISON that decides
// whether a candidate string is one of those members. A derived sweep confirmed
// the gap at the real entry point: case-folding the comparison inside
// requireEnum makes CalculateObjectIdentity accept and attest a Lease Record
// whose reason is "CREATE"; trimming makes it attest " create "; relaxing
// requireExactString from equality to a prefix test makes it attest a Checkpoint
// Record whose status is "validated_but_not_really". Every one of those mutants
// left `go test ./...`, all four seeded fuzz corpora, tracecheck, the 47-row
// closed-vocabulary pin and the derived refusal-coverage gate green, because the
// pinned rows still match the source and the refusal branch is still executed
// for whatever outside value a hand-written case happens to pick.
//
// The shape is the one this package already closed once for member lists: a test
// that refuses ONE outside value proves the gate is REACHABLE, not that the
// admitted set is the DECLARED set. The admitted set is
// (member list) INTERSECT (matcher semantics). This file derives the second
// factor: every literal-admitted string in every valid fixture is attacked at
// the production entry with a fixed adversarial family built from the value
// itself - case variants, leading and trailing whitespace, a proper prefix, a
// proper superstring, and an embedded NUL - so widening the comparison in any of
// those directions reddens the suite at every call site at once.

// stringAdmissionDeciders is the pinned set of package functions that decide a
// string member's admission IN THEIR OWN BODY - by comparing the member against
// a caller-supplied string parameter, by ranging over one, or by handing one to
// an admission primitive outside this package.
//
// The value says whether the parameter carries a LITERAL admitted set. Only
// literal deciders are walked by the site inventory below; the one false entry
// takes a parameter that SELECTS a grammar rather than enumerating the admitted
// values, so there is no literal set to pin and its admitted set is proven by
// the declared-grammar proofs instead.
//
// This is not a convenience list:
// TestStringAdmissionDerivationCoversEveryLiteralAdmittingHelper derives the
// same set from the sources and fails when they disagree, so a byte-identical
// duplicate of requireEnum or requireExactString - the exact defect this Story
// already shipped once as requireMemberSet - reddens instead of quietly carrying
// unpinned admission semantics. The derivation is deliberately broad about HOW a
// decider compares: an equality test, a range, and a string predicate such as
// strings.EqualFold or strings.HasPrefix all keep the function inside the
// inventory, so relaxing the comparison cannot make a decider disappear from its
// own gate.
//
// Functions that merely FORWARD a string parameter into one of these are not
// pinned: the site walk visits the inner call and resolveStringArguments follows
// the parameter out through the forwarder's own callers, so a new wrapper is
// covered without editing anything here.
var stringAdmissionDeciders = map[string]bool{
	"requireEnum":        true,
	"requireExactString": true,
	// objectFormat selects the sha1 or sha256 Git OID grammar; it is not the
	// admitted set. The admitted set is the OID grammar, proven separately.
	"requireGitOIDForFormat": false,
}

// stringAdmissionSite is one production call that admits a closed set of string
// literals for one named member.
type stringAdmissionSite struct {
	file     string
	function string
	helper   string
	member   string
	values   []string
}

func (site stringAdmissionSite) String() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", site.file, site.function, site.helper, site.member, strings.Join(site.values, ","))
}

// unanchoredStringAdmissionSites discloses a derived admission site that no
// valid fixture reaches with one of its admitted values, together with the
// reason. An entry here is a hole in the family gate and must name why the site
// is unreachable from everyValidIdentityFixture.
var unanchoredStringAdmissionSites = map[string]string{}

// adversarialCase is one widening of the matcher semantics, named by the
// dimension it widens so a failure says which comparison was relaxed.
type adversarialCase struct {
	dimension string
	value     string
}

// requiredAdversarialDimensions is the attack surface of a string comparison.
// It is pinned rather than derived because it describes the MUTANTS the family
// must defeat, not anything production declares: each entry corresponds to one
// way an equality test over a declared member list can be relaxed, and each was
// observed to attest an undeclared value at CalculateObjectIdentity when the
// comparison was relaxed that way. Deleting a dimension from the family is
// exactly the delete-only weakening this file exists to catch, so
// TestAdversarialFamilyCoversEveryPinnedWideningDimension requires all of them.
var requiredAdversarialDimensions = []string{
	"case-fold (upper)",
	"case-fold (lower)",
	"case-fold (title)",
	"leading whitespace",
	"trailing whitespace",
	"surrounding whitespace",
	"leading tab",
	"trailing newline",
	"proper superstring (suffix)",
	"proper superstring (prefix)",
	"embedded NUL",
	"proper prefix",
	"proper suffix",
}

// TestAdversarialFamilyCoversEveryPinnedWideningDimension is the family's own
// completeness proof, and it also proves the two filters that keep the family
// honest: a variant that collapses onto the admitted value, and one that
// collapses onto another declared member of the same vocabulary, are both
// dropped instead of being asserted as refusals they are not.
func TestAdversarialFamilyCoversEveryPinnedWideningDimension(t *testing.T) {
	t.Parallel()

	// No single value can exercise all three case folds: for a lowercase member
	// the lower fold collapses onto the value itself, and for an uppercase one
	// the upper and title folds coincide. The union over both spellings is
	// required to cover every pinned dimension, and neither spelling may emit a
	// dimension that is not pinned.
	produced := make(map[string]bool)
	for _, member := range []string{"create", "CREATE"} {
		for _, attack := range adversarialFamily(member, map[string]bool{member: true}) {
			produced[attack.dimension] = true
		}
	}
	pinned := make(map[string]bool, len(requiredAdversarialDimensions))
	for _, dimension := range requiredAdversarialDimensions {
		pinned[dimension] = true
		if !produced[dimension] {
			t.Errorf(
				"the adversarial family no longer produces a %q case, so a matcher relaxed in that direction "+
					"attests an undeclared value with the whole suite green", dimension)
		}
	}
	for dimension := range produced {
		if !pinned[dimension] {
			t.Errorf("the adversarial family produces an unpinned %q case; pin it or remove it", dimension)
		}
	}

	// "AB" upper-cases to itself, so the upper case-fold variant must be
	// dropped: asserting it as a refusal would assert that the admitted value
	// itself is refused.
	for _, attack := range adversarialFamily("AB", map[string]bool{"AB": true}) {
		if attack.value == "AB" {
			t.Errorf("the family emitted the admitted value itself as a %q case", attack.dimension)
		}
	}
	// A sibling member of the same vocabulary is declared, so a variant that
	// collapses onto it must be dropped too.
	for _, attack := range adversarialFamily("stop", map[string]bool{"stop": true, "STOP": true, "stops": true}) {
		if attack.value == "STOP" || attack.value == "stops" {
			t.Errorf("the family emitted the declared sibling %q as a %q case", attack.value, attack.dimension)
		}
	}
}

// adversarialFamily builds the fixed family from an admitted value. Each entry
// is admitted by exactly one relaxation of an equality test over the declared
// member list, so refusing all of them pins the comparison in every direction a
// mutant can widen it.
func adversarialFamily(admitted string, siblings map[string]bool) []adversarialCase {
	candidates := []adversarialCase{
		{"case-fold (upper)", strings.ToUpper(admitted)},
		{"case-fold (lower)", strings.ToLower(admitted)},
		{"case-fold (title)", strings.Title(admitted)}, //nolint:staticcheck // ASCII vocabularies only
		{"leading whitespace", " " + admitted},
		{"trailing whitespace", admitted + " "},
		{"surrounding whitespace", " " + admitted + " "},
		{"leading tab", "\t" + admitted},
		{"trailing newline", admitted + "\n"},
		{"proper superstring (suffix)", admitted + "_not_really"},
		{"proper superstring (prefix)", "not_really_" + admitted},
		{"embedded NUL", admitted + "\x00"},
	}
	if runes := []rune(admitted); len(runes) > 1 {
		candidates = append(candidates,
			adversarialCase{"proper prefix", string(runes[:len(runes)-1])},
			adversarialCase{"proper suffix", string(runes[1:])},
		)
	}
	family := make([]adversarialCase, 0, len(candidates))
	seen := map[string]bool{admitted: true}
	for _, candidate := range candidates {
		// A variant that collapses back onto the admitted value, or onto another
		// declared member of the same vocabulary, proves nothing and must not be
		// asserted as a refusal.
		if seen[candidate.value] || siblings[candidate.value] {
			continue
		}
		seen[candidate.value] = true
		family = append(family, candidate)
	}
	return family
}

// siteBinding is one derived admission site observed at a concrete position of
// a valid fixture: the site's declared members are exactly the values the
// production entry admits at that position.
type siteBinding struct {
	site    stringAdmissionSite
	fixture identityFixture
	path    []string
	current string
}

// bindAdmissionSites locates each derived admission site in the valid fixtures
// by BEHAVIOUR rather than by name.
//
// A site binds to a fixture position when the position currently carries one of
// its declared members and every other declared member, substituted there, is
// either accepted or refused for a reason that is not the closed-enum
// membership refusal. That second condition is what separates two sites that
// share a member name: `kind` on a Session Record admits direct|task_board and
// `kind` on a Transfer Manifest admits five other values, and substituting
// "composite" into a Session Record is refused as a non-member, so the manifest
// site does not bind there.
//
// Binding by behaviour also makes the inventory prove the admitted set in both
// directions at once. If the matcher narrows and silently drops a declared
// member, that member is refused as a non-member everywhere, no position binds
// the site, and the coverage assertion below reports it as unanchored.
func bindAdmissionSites(t *testing.T, sites []stringAdmissionSite) []siteBinding {
	t.Helper()

	governed := governedAdmittedValues(sites)
	var bindings []siteBinding
	for _, fixture := range everyValidIdentityFixture() {
		for _, path := range everyCandidateValuePath(fixture.object) {
			member := governedMemberName(path)
			if member == "" {
				continue
			}
			current, ok := jsonValueAtPath(t, fixture.object, path).(string)
			if !ok {
				continue
			}
			for _, site := range governed[admittedValueKey{member: member, value: current}] {
				if !siteAdmitsEveryDeclaredMemberAt(t, fixture, path, site, current) {
					continue
				}
				bindings = append(bindings, siteBinding{site: site, fixture: fixture, path: path, current: current})
			}
		}
	}
	return bindings
}

func siteAdmitsEveryDeclaredMemberAt(t *testing.T, fixture identityFixture, path []string, site stringAdmissionSite, current string) bool {
	t.Helper()

	for _, declared := range site.values {
		if declared == current {
			continue
		}
		candidate := cloneJSONObject(t, fixture.object)
		setJSONValueAtPath(t, candidate, path, declared)
		if !fixtureEntriesRefuse(t, fixture, candidate) {
			continue
		}
		if strings.Contains(fixtureRefusalReason(t, fixture, candidate), vocabularyMembershipRefusal) {
			return false
		}
	}
	return true
}

// TestEveryLiteralAdmittedStringRefusesItsAdversarialFamilyAtTheProductionEntry
// is the gate. For every derived admission site it locates the valid-fixture
// positions the site governs and requires the whole adversarial family built
// from the admitted value there to be refused at CalculateObjectIdentity and
// VerifyObjectIdentity, or at ValidateObservationEvent for Section 18.1 objects.
//
// The obligation is derived from production and the family is derived from the
// admitted value, so no case is hand-picked: relaxing requireEnum or
// requireExactString in any of these directions fails here at every site that
// reaches it, instead of leaving 74 call sites pinned by their member list and
// unpinned by their comparison.
//
// What this cannot prove: a matcher that admits one arbitrary EXTRA string
// unrelated to any declared member. That is unbounded and no finite family
// reaches it; the closed-vocabulary pin covers the call-site argument list, and
// this file covers the comparison.
func TestEveryLiteralAdmittedStringRefusesItsAdversarialFamilyAtTheProductionEntry(t *testing.T) {
	sites := deriveStringAdmissionSites(t)
	bindings := bindAdmissionSites(t, sites)

	anchored := make(map[string]bool)
	var admitted []string
	for _, binding := range bindings {
		anchored[binding.site.String()] = true
		siblings := make(map[string]bool, len(binding.site.values))
		for _, value := range binding.site.values {
			siblings[value] = true
		}
		for _, attack := range adversarialFamily(binding.current, siblings) {
			candidate := cloneJSONObject(t, binding.fixture.object)
			setJSONValueAtPath(t, candidate, binding.path, attack.value)
			if !fixtureEntriesRefuse(t, binding.fixture, candidate) {
				admitted = append(admitted, fmt.Sprintf(
					"%s at %s (site %s): %s widening admitted %q for declared member %q",
					binding.fixture.name, formatJSONPath(binding.path), binding.site,
					attack.dimension, attack.value, binding.current))
			}
		}
	}

	sort.Strings(admitted)
	if len(admitted) > 0 {
		t.Errorf(
			"the production entry ADMITTED and attested a value that no declared vocabulary or literal contains. "+
				"The member list is pinned; the comparison that decides membership has been widened:\n  %s",
			strings.Join(admitted, "\n  "))
	}

	// The gate must not pass by covering nothing, and an unanchored site is also
	// the signature of a NARROWED matcher: a site whose declared members are no
	// longer all admitted at any position binds nowhere.
	var unreached []string
	for _, site := range sites {
		key := site.String()
		_, disclosed := unanchoredStringAdmissionSites[key]
		if anchored[key] {
			if disclosed {
				t.Errorf("admission site %s is disclosed as unanchored but a fixture position binds it; remove the disclosure", key)
			}
			continue
		}
		if disclosed {
			continue
		}
		unreached = append(unreached, key)
	}
	sort.Strings(unreached)
	if len(unreached) > 0 {
		t.Errorf(
			"admission site(s) that bind to no valid-fixture position, so their matcher semantics are unpinned. "+
				"Either the matcher no longer admits every declared member, or no fixture carries the shape: add "+
				"a fixture, or disclose the site in unanchoredStringAdmissionSites with the reason:\n  %s",
			strings.Join(unreached, "\n  "))
	}
}

// fixtureRefusalReason returns the refusal message of the entry that owns the
// fixture, so a refusal can be attributed to a clause rather than assumed.
func fixtureRefusalReason(t *testing.T, fixture identityFixture, candidate map[string]any) string {
	t.Helper()

	encoded := mustJSON(t, candidate)
	if fixture.selfField == "" {
		if err := ValidateObservationEvent(encoded); err != nil {
			return err.Error()
		}
		return ""
	}
	if _, _, err := CalculateObjectIdentity(encoded); err != nil {
		return err.Error()
	}
	return ""
}

// vocabularyMembershipRefusal is the message the shared closed-enum primitive
// emits when a value is outside the negotiated vocabulary. A declared member
// refused with this message means the matcher narrowed; a declared member
// refused with any other message was rejected by a cross-field coupling, which
// is a different and legitimate clause.
const vocabularyMembershipRefusal = "is not a member of the negotiated vocabulary"

type admittedValueKey struct {
	member string
	value  string
}

func governedAdmittedValues(sites []stringAdmissionSite) map[admittedValueKey][]stringAdmissionSite {
	governed := make(map[admittedValueKey][]stringAdmissionSite)
	for _, site := range sites {
		for _, value := range site.values {
			key := admittedValueKey{member: site.member, value: value}
			governed[key] = append(governed[key], site)
		}
	}
	return governed
}

// governedMemberName returns the object member a candidate path addresses, or
// "" when the path addresses an array element rather than a named member.
func governedMemberName(path []string) string {
	if len(path) == 0 {
		return ""
	}
	last := path[len(path)-1]
	if strings.HasPrefix(last, "[") {
		return ""
	}
	return last
}

// TestStringAdmissionDerivationCoversEveryLiteralAdmittingHelper is the gate's
// own coverage proof. It derives every package function that decides whether a
// named string member is one of the literal values its caller supplied, and
// fails unless that set is exactly the pinned primitives. A second copy of
// requireEnum or requireExactString - the exact defect this Story already
// shipped once with requireMemberSet - reddens here instead of carrying
// unpinned admission semantics.
func TestStringAdmissionDerivationCoversEveryLiteralAdmittingHelper(t *testing.T) {
	t.Parallel()

	found := deriveStringAdmissionHelpers(t)
	for name := range found {
		if _, pinned := stringAdmissionDeciders[name]; !pinned {
			t.Errorf(
				"%s decides a string member's admission from a caller-supplied parameter but is not pinned, so "+
					"every value it admits is outside the adversarial-family gate. Collapse it into requireEnum or "+
					"requireExactString, or pin it in stringAdmissionDeciders and say whether its parameter carries "+
					"a literal admitted set.",
				name)
		}
	}
	for name := range stringAdmissionDeciders {
		if !found[name] {
			t.Errorf(
				"pinned string-admission decider %s is no longer derivable from the sources. Either it was renamed, "+
					"or its comparison no longer reads as a decision over its own parameter - which is itself a "+
					"widening, because every site below it drops out of the adversarial-family gate.", name)
		}
	}
}

// packageProductionAST parses every production file of the package once, so
// derivations that need cross-function resolution share positions.
func packageProductionAST(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()

	_, paths := packageProductionFiles(t)
	fileSet := token.NewFileSet()
	files := make([]*ast.File, 0, len(paths))
	for _, path := range paths {
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, parsed)
	}
	return fileSet, files
}

type functionParameter struct {
	name     string
	kind     string
	variadic bool
}

func flattenedParameters(function *ast.FuncDecl) []functionParameter {
	var parameters []functionParameter
	if function.Type.Params == nil {
		return parameters
	}
	for _, field := range function.Type.Params.List {
		kind, variadic := parameterKind(field.Type)
		if len(field.Names) == 0 {
			parameters = append(parameters, functionParameter{kind: kind, variadic: variadic})
			continue
		}
		for _, name := range field.Names {
			parameters = append(parameters, functionParameter{name: name.Name, kind: kind, variadic: variadic})
		}
	}
	return parameters
}

func parameterKind(expression ast.Expr) (string, bool) {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name, false
	case *ast.Ellipsis:
		kind, _ := parameterKind(typed.Elt)
		return kind, true
	case *ast.MapType:
		key, _ := parameterKind(typed.Key)
		value, _ := parameterKind(typed.Value)
		return "map[" + key + "]" + value, false
	case *ast.InterfaceType:
		return "any", false
	}
	return "", false
}

// deriveMemberReaders returns every package function that reads a named member
// out of a candidate object, mapped to the parameter index that names it.
func deriveMemberReaders(t *testing.T, files []*ast.File) map[string]int {
	t.Helper()

	functions := packageFunctionDeclarations(files)
	readers := make(map[string]int)
	for name, function := range functions {
		parameters := flattenedParameters(function)
		objects := map[string]bool{}
		for _, parameter := range parameters {
			if parameter.kind == "map[string]any" {
				objects[parameter.name] = true
			}
		}
		if len(objects) == 0 {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			index, ok := node.(*ast.IndexExpr)
			if !ok {
				return true
			}
			container, ok := index.X.(*ast.Ident)
			if !ok || !objects[container.Name] {
				return true
			}
			key, ok := index.Index.(*ast.Ident)
			if !ok {
				return true
			}
			if position := parameterIndex(parameters, key.Name, "string"); position >= 0 {
				readers[name] = position
			}
			return true
		})
	}
	for changed := true; changed; {
		changed = false
		for name, function := range functions {
			if _, done := readers[name]; done {
				continue
			}
			parameters := flattenedParameters(function)
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee, ok := call.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				position, ok := readers[callee.Name]
				if !ok || position >= len(call.Args) {
					return true
				}
				argument, ok := call.Args[position].(*ast.Ident)
				if !ok {
					return true
				}
				if index := parameterIndex(parameters, argument.Name, "string"); index >= 0 {
					readers[name] = index
					changed = true
				}
				return true
			})
		}
	}
	if len(readers) == 0 {
		t.Fatal("derived zero member readers from the package sources; the scanner is broken, not the package")
	}
	return readers
}

func parameterIndex(parameters []functionParameter, name, kind string) int {
	for index, parameter := range parameters {
		if parameter.name == name && parameter.kind == kind {
			return index
		}
	}
	return -1
}

// admissionPrimitive records the parameter positions of a derived primitive.
type admissionPrimitive struct {
	memberIndex int
	valueIndex  int
	variadic    bool
	// local marks a function that decides admission itself rather than
	// forwarding the decision to another derived function.
	local bool
}

// deriveStringAdmissionHelpers returns every package function that decides
// whether a named string member equals one of the literal values its caller
// supplied. Nothing here is a name list: a function qualifies when it reads a
// named member from a candidate object AND decides that member's admission from
// a string parameter of its own, either by comparing against it, ranging over
// it, forwarding it to an admission primitive outside this package, or
// forwarding it to an already-derived helper.
func deriveStringAdmissionHelpers(t *testing.T) map[string]bool {
	t.Helper()

	_, files := packageProductionAST(t)
	return deriveStringAdmissionPrimitives(t, files).localNames()
}

type derivedPrimitives map[string]admissionPrimitive

// localNames returns the derived functions that decide admission themselves.
func (primitives derivedPrimitives) localNames() map[string]bool {
	names := make(map[string]bool, len(primitives))
	for name, primitive := range primitives {
		if primitive.local {
			names[name] = true
		}
	}
	return names
}

func deriveStringAdmissionPrimitives(t *testing.T, files []*ast.File) derivedPrimitives {
	t.Helper()

	functions := packageFunctionDeclarations(files)
	readers := deriveMemberReaders(t, files)
	primitives := derivedPrimitives{}

	valueParameterOf := func(function *ast.FuncDecl) (int, bool, bool, bool) {
		parameters := flattenedParameters(function)
		best, bestVariadic, found, local := -1, false, false, false
		record := func(index int, decidedHere bool) {
			if index < 0 || parameters[index].kind != "string" {
				return
			}
			if !found || index > best {
				best, bestVariadic, found = index, parameters[index].variadic, true
			}
			if decidedHere {
				local = true
			}
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.BinaryExpr:
				if typed.Op != token.EQL && typed.Op != token.NEQ {
					return true
				}
				for _, side := range []ast.Expr{typed.X, typed.Y} {
					if identifier, ok := side.(*ast.Ident); ok {
						record(parameterIndex(parameters, identifier.Name, "string"), true)
					}
				}
			case *ast.RangeStmt:
				if identifier, ok := typed.X.(*ast.Ident); ok {
					record(parameterIndex(parameters, identifier.Name, "string"), true)
				}
			case *ast.CallExpr:
				// A parameter handed to an admission primitive outside this
				// package decides admission here, whether it is forwarded as a
				// variadic vocabulary or compared by a string predicate such as
				// strings.HasPrefix. Both forms are matcher semantics, so both
				// keep the function inside the derivation.
				if _, external := typed.Fun.(*ast.SelectorExpr); external {
					for _, argument := range typed.Args {
						if identifier, ok := argument.(*ast.Ident); ok {
							record(parameterIndex(parameters, identifier.Name, "string"), true)
						}
					}
				}
				callee, ok := typed.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				primitive, derived := primitives[callee.Name]
				if !derived {
					return true
				}
				for position := primitive.valueIndex; position < len(typed.Args); position++ {
					if identifier, ok := typed.Args[position].(*ast.Ident); ok {
						record(parameterIndex(parameters, identifier.Name, "string"), false)
					}
				}
			}
			return true
		})
		return best, bestVariadic, found, local
	}

	for changed := true; changed; {
		changed = false
		for name, function := range functions {
			if _, done := primitives[name]; done {
				continue
			}
			member, reads := readers[name]
			if !reads {
				continue
			}
			value, variadic, found, local := valueParameterOf(function)
			if !found || value == member {
				continue
			}
			primitives[name] = admissionPrimitive{memberIndex: member, valueIndex: value, variadic: variadic, local: local}
			changed = true
		}
	}
	if len(primitives) == 0 {
		t.Fatal("derived zero string-admission primitives from the package sources; the scanner is broken, not the package")
	}
	return primitives
}

func packageFunctionDeclarations(files []*ast.File) map[string]*ast.FuncDecl {
	functions := make(map[string]*ast.FuncDecl)
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil {
				continue
			}
			functions[function.Name.Name] = function
		}
	}
	return functions
}

// callEnvironment is one frame of the resolution call chain. `args` are the
// argument expressions the caller supplied for `function`, evaluated in `outer`.
// A frame with nil args is the frontier: its parameters are still free, and the
// walk branches over that function's own call sites to bind them.
//
// Member and values are always resolved in the SAME frame chain, so a wrapper
// such as enum(name, allowed...) contributes one site per concrete caller rather
// than the Cartesian product of every member with every value list.
type callEnvironment struct {
	function string
	args     []ast.Expr
	ellipsis bool
	outer    *callEnvironment
}

func (environment *callEnvironment) frontier() string {
	for current := environment; current != nil; current = current.outer {
		if current.args == nil {
			return current.function
		}
	}
	return ""
}

// extendChain rebuilds the chain with the frontier frame replaced by a bound
// one, leaving the original chain untouched so sibling branches do not alias.
func extendChain(environment, replacement *callEnvironment) *callEnvironment {
	if environment == nil || environment.args == nil {
		return replacement
	}
	return &callEnvironment{
		function: environment.function,
		args:     environment.args,
		ellipsis: environment.ellipsis,
		outer:    extendChain(environment.outer, replacement),
	}
}

type siteValues struct {
	member string
	values []string
}

const stringResolutionDepth = 6

type resolution struct {
	fileSet    *token.FileSet
	functions  map[string]*ast.FuncDecl
	callers    map[string][]*ast.CallExpr
	constants  map[string]string
	primitives derivedPrimitives
}

// deriveStringAdmissionSites walks every call to a derived admission decider and
// resolves the member name and the admitted values to literals.
//
// Resolution follows string literals, package string constants, and parameters
// of the enclosing function through that function's own call sites, so a wrapper
// such as enum(name, allowed...) or the shared envelope validator that forwards
// a schema and a version contributes one site per concrete caller rather than
// dropping out of the inventory. A call that cannot be resolved to literals
// fails the derivation with its position instead of being skipped.
func deriveStringAdmissionSites(t *testing.T) []stringAdmissionSite {
	t.Helper()

	fileSet, files := packageProductionAST(t)
	context := resolution{
		fileSet:    fileSet,
		functions:  packageFunctionDeclarations(files),
		callers:    packageCallSites(files),
		constants:  derivePackageStringConstants(files),
		primitives: deriveStringAdmissionPrimitives(t, files),
	}

	unique := make(map[string]stringAdmissionSite)
	for _, file := range files {
		name := filepath.Base(fileSet.Position(file.Pos()).Filename)
		var enclosing string
		ast.Inspect(file, func(node ast.Node) bool {
			if function, ok := node.(*ast.FuncDecl); ok {
				enclosing = function.Name.Name
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			primitive, derived := context.primitives[callee.Name]
			if !derived || !primitive.local || !stringAdmissionDeciders[callee.Name] {
				return true
			}
			resolved := resolveAdmissionSite(t, context, call, primitive, &callEnvironment{function: enclosing}, 0)
			if len(resolved) == 0 {
				position := fileSet.Position(call.Pos())
				t.Fatalf(
					"%s:%d call to %s does not resolve to a literal member and literal values, so its admitted set "+
						"cannot be pinned; spell them literally or extend the resolution",
					name, position.Line, callee.Name)
			}
			for _, values := range resolved {
				site := stringAdmissionSite{
					file: name, function: enclosing, helper: callee.Name,
					member: values.member, values: values.values,
				}
				unique[site.String()] = site
			}
			return true
		})
	}

	sites := make([]stringAdmissionSite, 0, len(unique))
	for _, site := range unique {
		sites = append(sites, site)
	}
	sort.Slice(sites, func(first, second int) bool { return sites[first].String() < sites[second].String() })
	if len(sites) == 0 {
		t.Fatal("derived zero string-admission sites from the package sources; the scanner is broken, not the package")
	}
	return sites
}

func resolveAdmissionSite(t *testing.T, context resolution, call *ast.CallExpr, primitive admissionPrimitive, environment *callEnvironment, depth int) []siteValues {
	t.Helper()

	if depth > stringResolutionDepth {
		return nil
	}
	if primitive.memberIndex >= len(call.Args) || primitive.valueIndex > len(call.Args) {
		return nil
	}
	member, memberBound := resolveStringExpression(context, call.Args[primitive.memberIndex], environment, 0)
	values, valuesBound := resolveAdmittedValues(context, call, primitive, environment, 0)
	if memberBound && valuesBound {
		if len(member) != 1 || len(values) == 0 {
			return nil
		}
		return []siteValues{{member: member[0], values: values}}
	}

	frontier := environment.frontier()
	if frontier == "" {
		return nil
	}
	var resolved []siteValues
	seen := make(map[string]bool)
	for _, outer := range context.callers[frontier] {
		replacement := &callEnvironment{
			function: frontier,
			args:     outer.Args,
			ellipsis: outer.Ellipsis.IsValid(),
			outer:    &callEnvironment{function: enclosingFunctionOf(context, outer)},
		}
		for _, candidate := range resolveAdmissionSite(t, context, call, primitive, extendChain(environment, replacement), depth+1) {
			key := candidate.member + "|" + strings.Join(candidate.values, ",")
			if seen[key] {
				continue
			}
			seen[key] = true
			resolved = append(resolved, candidate)
		}
	}
	return resolved
}

// resolveAdmittedValues returns the literal values a call supplies at the value
// position. The second result reports whether every value was bound; an unbound
// value asks the caller to branch rather than silently dropping the site.
func resolveAdmittedValues(context resolution, call *ast.CallExpr, primitive admissionPrimitive, environment *callEnvironment, depth int) ([]string, bool) {
	if depth > stringResolutionDepth {
		return nil, false
	}
	if primitive.variadic && call.Ellipsis.IsValid() {
		if len(call.Args) == 0 {
			return nil, false
		}
		identifier, ok := call.Args[len(call.Args)-1].(*ast.Ident)
		if !ok {
			return nil, false
		}
		return resolveForwardedVariadic(context, identifier.Name, environment, depth+1)
	}
	last := primitive.valueIndex + 1
	if primitive.variadic {
		last = len(call.Args)
	}
	var values []string
	for position := primitive.valueIndex; position < last && position < len(call.Args); position++ {
		resolved, bound := resolveStringExpression(context, call.Args[position], environment, depth+1)
		if !bound || len(resolved) != 1 {
			return nil, false
		}
		values = append(values, resolved[0])
	}
	return values, len(values) > 0
}

// resolveForwardedVariadic resolves a `values...` forward by looking up the
// trailing arguments the current frame's caller supplied.
func resolveForwardedVariadic(context resolution, name string, environment *callEnvironment, depth int) ([]string, bool) {
	if depth > stringResolutionDepth || environment == nil {
		return nil, false
	}
	function, ok := context.functions[environment.function]
	if !ok {
		return nil, false
	}
	index := parameterIndex(flattenedParameters(function), name, "string")
	if index < 0 || environment.args == nil {
		return nil, false
	}
	if environment.ellipsis {
		identifier, ok := environment.args[len(environment.args)-1].(*ast.Ident)
		if !ok {
			return nil, false
		}
		return resolveForwardedVariadic(context, identifier.Name, environment.outer, depth+1)
	}
	var values []string
	for position := index; position < len(environment.args); position++ {
		resolved, bound := resolveStringExpression(context, environment.args[position], environment.outer, depth+1)
		if !bound || len(resolved) != 1 {
			return nil, false
		}
		values = append(values, resolved[0])
	}
	return values, len(values) > 0
}

// resolveStringExpression resolves one expression to a literal in the current
// frame chain. The second result reports whether it was bound at all.
func resolveStringExpression(context resolution, expression ast.Expr, environment *callEnvironment, depth int) ([]string, bool) {
	if depth > stringResolutionDepth {
		return nil, false
	}
	switch typed := expression.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return nil, false
		}
		value, err := strconv.Unquote(typed.Value)
		if err != nil {
			return nil, false
		}
		return []string{value}, true
	case *ast.Ident:
		if value, ok := context.constants[typed.Name]; ok {
			return []string{value}, true
		}
		if environment == nil {
			return nil, false
		}
		function, ok := context.functions[environment.function]
		if !ok {
			return nil, false
		}
		index := parameterIndex(flattenedParameters(function), typed.Name, "string")
		if index < 0 || environment.args == nil || index >= len(environment.args) {
			return nil, false
		}
		return resolveStringExpression(context, environment.args[index], environment.outer, depth+1)
	}
	return nil, false
}

// packageCallSites indexes every call to a package-level function by name.
func packageCallSites(files []*ast.File) map[string][]*ast.CallExpr {
	callers := make(map[string][]*ast.CallExpr)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if callee, ok := call.Fun.(*ast.Ident); ok {
				callers[callee.Name] = append(callers[callee.Name], call)
			}
			return true
		})
	}
	return callers
}

// enclosingFunctionOf reports which package function lexically contains a call,
// which is what makes parameter resolution follow the real call graph.
func enclosingFunctionOf(context resolution, call *ast.CallExpr) string {
	position := context.fileSet.Position(call.Pos()).Offset
	best, bestSpan := "", 1<<62
	for name, function := range context.functions {
		start := context.fileSet.Position(function.Pos()).Offset
		end := context.fileSet.Position(function.End()).Offset
		if start <= position && position <= end && end-start < bestSpan {
			best, bestSpan = name, end-start
		}
	}
	return best
}
