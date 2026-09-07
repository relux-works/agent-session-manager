package environ

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// This file is the shape layer of the shared-implementation
// census. The name layer in census_test.go derives function and
// method declarations by identifier and requires the derived set
// to equal the ledger: it catches renames, removals, and copies
// that reuse a ledgered name, but it cannot see a copy under a
// fresh spelling, a new byte-counting measure, or a var-bound
// alias. This layer derives implementation SHAPE from production
// source and requires every shape hit to be a registered row:
// a fresh spelling with the same implementation texture fails
// here as an unregistered site.
//
// Shape space (each proven by a synthetic control in
// TestShapeCensusCatchesControls, which drives the same pure
// extractor the production test runs on the real tree):
//
//   - strict-decoder: a function that builds an encoding/json
//     decoder (NewDecoder or Unmarshal) and carries a duplicate
//     signal (a "duplicate"+"member" literal or a *uplicate
//     identifier) in its own body or in a same-package callee.
//     The callee arm covers decoders split across two local
//     functions; a decoder split across packages is reuse, not
//     a copy, and is not derived.
//   - surrogate-gate: a function whose body holds a surrogate
//     range literal (0xD800, 0xDBFF, 0xDC00, 0xDFFF in any base)
//     or an identifier naming one (a *urrogate* token), or that
//     references a package-level constant whose value is a
//     surrogate range literal. Callers of a ledgered gate (the
//     environ decoder, scalar's string decoder) trip the token
//     arm and are registered as callers, not gates.
//   - string-measure: a function that counts runes directly
//     (a RuneCountInString/RuneCount selector).
//   - byte-measure: a function with a string parameter it
//     applies len() to and exactly one int result (the
//     func(string)->int texture), or a function that applies
//     len() to a variable assigned from a rawString() call (the
//     verdict-style texture). Verdict-style measures that reuse
//     a registered measure helper (stringLength and friends)
//     are reuse, not copies, and are not derived.
//   - var-alias: a package-level var that rebinds a ledgered
//     shared-rule name, that binds a closure carrying any shape
//     above, or that aliases a shape-positive local function.
//     Binding an imported canonical helper under a fresh name
//     (`var f = environ.StringLength`) is reuse and is clean.
//
// Stated residue. A decoder that never touches encoding/json (a
// hand-rolled scanner with its own duplicate rule), a gate whose
// bounds are derived arithmetically under fresh names with no
// surrogate token anywhere, a byte measure over a non-string
// parameter type, and a new grammar literal outside the four
// known classes are not derived here. Each is named so the next
// copy in one of those textures fails loudly by extending the
// extractor, not silently.

// shapeSite is one shape-derived shared-rule implementation.
type shapeSite struct {
	class  string
	pkg    string
	file   string
	symbol string
}

func (site shapeSite) key() string {
	return site.class + "|" + site.pkg + "|" + site.file + "|" + site.symbol
}

// surrogateBounds is the UTF-16 surrogate range: any integer
// literal with one of these values is a gate bound.
var surrogateBounds = map[uint64]bool{
	0xD800: true,
	0xDBFF: true,
	0xDC00: true,
	0xDFFF: true,
}

// shapeLedger registers every shape hit in the census scope with
// the rationale that keeps it honest. Canonical owners are the
// single implementation new code must use; every other row names
// the behavioral battery that pins it to the canonical
// semantics. An unregistered hit fails the census; an orphaned
// row fails it too.
var shapeLedger = map[string]string{
	// Strict decoders: one JSON entry point per owner plus the
	// two envelope decoders that reach one.
	"strict-decoder|environ|decode.go|DecodeStrictObject":    "canonical owner: string-walk surrogate semantics, rune measure",
	"strict-decoder|provhost|protocol.go|decodeStrictObject": "string-aware walk, pinned by frame_agreement_test.go",
	"strict-decoder|provhost|protocol.go|DecodeResponse":     "response-envelope decoder over decodeStrictObject, pinned by provhost protocol tests",
	"strict-decoder|provhost|status.go|DecodeStatusOutcome":  "status-envelope decoder over decodeStrictObject, pinned by provhost status tests",
	"strict-decoder|canonicaljson|canonical.go|decodeStrict": "canonicalizer entry over decodeValue, pinned by frame_agreement_test.go",
	// Surrogate gates and their registered callers.
	"surrogate-gate|environ|decode.go|HasLoneSurrogateEscape":            "canonical owner: string-aware walk",
	"surrogate-gate|environ|decode.go|DecodeStrictObject":                "caller: owns the gate call, pinned by frame_agreement_test.go",
	"surrogate-gate|provhost|surrogate.go|hasLoneSurrogateEscape":        "string-aware walk, pinned by frame_agreement_test.go",
	"surrogate-gate|provhost|protocol.go|decodeStrictObject":             "caller: owns the gate call, pinned by frame_agreement_test.go",
	"surrogate-gate|canonicaljson|canonical.go|validateSurrogateEscapes": "canonicalizer gate: string-aware walk, pinned by frame_agreement_test.go",
	"surrogate-gate|canonicaljson|canonical.go|decodeStrict":             "caller: owns the gate call, pinned by frame_agreement_test.go",
	"surrogate-gate|scalar|scalar.go|hasLoneJSONSurrogate":               "third spelling (single-string unit, scalar-owned): retained, pinned both directions by scalar_agreement_test.go",
	"surrogate-gate|scalar|scalar.go|decodeJSONString":                   "caller: owns the scalar gate call, pinned by scalar_agreement_test.go through DecodeClosedEnumJSON",
	// Rune measures: one per package plus the canonicalizer's
	// internal bounded-string helpers, which the name census
	// never saw under their fresh spellings. The sessadapter and
	// dirnode measures converged onto environ.StringLength: their
	// delegating wrappers carry no rune count of their own, so
	// they derive no shape row here (the name layer still ledgers
	// the wrapper names, and TestCheckHelpersDelegateToEnviron
	// pins the delegation structurally). A revived local count
	// fails here as unregistered.
	"string-measure|environ|decode.go|StringLength":                               "canonical owner: runes per Section 1.6",
	"string-measure|provhost|opdecode.go|runeLength":                              "runes, pinned by measure_agreement_test.go",
	"string-measure|canonicaljson|closed_shapes.go|requireBoundedString":          "canonicalizer-internal rune bound, pinned by the canonicaljson suite",
	"string-measure|canonicaljson|closed_shapes.go|validateBlobDescriptor":        "canonicalizer-internal rune bound, pinned by the canonicaljson suite",
	"string-measure|canonicaljson|closed_shapes.go|validateManifestEntries":       "canonicalizer-internal rune bound, pinned by the canonicaljson suite",
	"string-measure|canonicaljson|core_records.go|nullableBoundedString":          "canonicalizer-internal rune bound, pinned by the canonicaljson suite",
	"string-measure|canonicaljson|core_records.go|validateProviderIdentityRecord": "canonicalizer-internal rune bound, pinned by identity_agreement_test.go opaque rows",
	"string-measure|scalar|path.go|ParseAbsolutePath":                             "grammar-owner path bound, pinned by the scalar suite",
	// Byte measures: the two Section 5.1 exec gates.
	"byte-measure|provhost|spawn.go|checkSpawnArgv":        "bytes by specification, pinned by TestByteBoundsStayBytes",
	"byte-measure|provhost|spawn.go|checkSpawnEnvLiterals": "bytes by specification, pinned by TestByteBoundsStayBytes",
}

// packageShapes is the per-package extractor state: which local
// functions carry a duplicate signal or a shape, and which
// package-level constants hold surrogate bounds.
type packageShapes struct {
	funcDups   map[string]bool
	funcShapes map[string][]string
	constBound map[string]bool
}

// TestSharedShapesAreLedgered requires every shape-derived site
// to be a registered row, and every row to be derived: an
// unregistered copy fails as a unification violation, and a row
// naming an implementation production no longer derives fails
// as orphaned. Both directions run through the invcore harness.
func TestSharedShapesAreLedgered(t *testing.T) {
	derived := deriveShapeSites(t)
	rows := make(map[string]struct{}, len(shapeLedger))
	for key := range shapeLedger {
		rows[key] = struct{}{}
	}
	derivedSet := make(map[string]struct{}, len(derived))
	for key := range derived {
		derivedSet[key] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "shared-rule shape", derivedSet, rows)
}

// deriveShapeSites walks the census scope and derives every
// shape site from production source.
func deriveShapeSites(t *testing.T) map[string]bool {
	t.Helper()
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	sources := readScopeSources(t, root)
	sites := map[string]bool{}
	for pkg, files := range sources {
		for _, site := range shapeSitesInPackage(pkg, files) {
			sites[site.key()] = true
		}
	}
	if len(sites) == 0 {
		t.Fatal("shapes derived zero sites; the scanner is blind, not the tree clean")
	}
	return sites
}

// readScopeSources reads every production Go file in the census
// packages into memory, keyed by package then file name.
// Production file selection and fail-closed parsing come from
// invcore: an unreadable or unparseable production file fails the
// suite instead of scanning as clean.
func readScopeSources(t *testing.T, root string) map[string]map[string][]byte {
	t.Helper()
	sources := map[string]map[string][]byte{}
	packages := make([]string, 0, len(censusPackages))
	for pkg := range censusPackages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	for _, pkg := range packages {
		files, _ := invcore.MustScanProduction(t, filepath.Join(root, pkg))
		if sources[pkg] == nil {
			sources[pkg] = map[string][]byte{}
		}
		for _, production := range files {
			sources[pkg][production.Name] = production.Source
		}
	}
	return sources
}

// shapeSitesInPackage derives the shape sites of one package. It
// is pure over its inputs, so the synthetic controls drive this
// exact function on planted sources: a control that fails here
// fails the production census by construction.
func shapeSitesInPackage(pkg string, files map[string][]byte) []shapeSite {
	parsed := map[string]*ast.File{}
	for file, source := range files {
		syntax, err := parser.ParseFile(token.NewFileSet(), file, source, 0)
		if err != nil {
			return []shapeSite{{class: "parse-failure", pkg: pkg, file: file, symbol: "unparseable"}}
		}
		parsed[file] = syntax
	}
	state := &packageShapes{funcDups: map[string]bool{}, funcShapes: map[string][]string{}, constBound: map[string]bool{}}
	for _, syntax := range parsed {
		for _, decl := range syntax.Decls {
			switch node := decl.(type) {
			case *ast.FuncDecl:
				if node.Body == nil {
					continue
				}
				if bodyHasDupSignal(node.Body) {
					state.funcDups[node.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for index, name := range value.Names {
						if index >= len(value.Values) {
							continue
						}
						if literal, ok := value.Values[index].(*ast.BasicLit); ok && literal.Kind == token.INT {
							if magnitude, err := strconv.ParseUint(literal.Value, 0, 32); err == nil && surrogateBounds[magnitude] {
								state.constBound[name.Name] = true
							}
						}
					}
				}
			}
		}
	}
	for _, syntax := range parsed {
		for _, decl := range syntax.Decls {
			node, ok := decl.(*ast.FuncDecl)
			if !ok || node.Body == nil {
				continue
			}
			// Same-named methods union their shapes so the
			// alias resolution below is file-order
			// independent.
			union := map[string]bool{}
			for _, class := range state.funcShapes[node.Name.Name] {
				union[class] = true
			}
			for _, class := range funcShapeClasses(node.Body, node.Type, state) {
				union[class] = true
			}
			var merged []string
			for class := range union {
				merged = append(merged, class)
			}
			sort.Strings(merged)
			state.funcShapes[node.Name.Name] = merged
		}
	}
	var sites []shapeSite
	add := func(class, file, symbol string) {
		sites = append(sites, shapeSite{class: class, pkg: pkg, file: file, symbol: symbol})
	}
	for file, syntax := range parsed {
		for _, decl := range syntax.Decls {
			switch node := decl.(type) {
			case *ast.FuncDecl:
				if node.Body == nil {
					continue
				}
				// Emission uses the file-local shapes; the
				// union map above serves alias resolution
				// only, so same-named methods never bleed
				// sites across files.
				for _, class := range funcShapeClasses(node.Body, node.Type, state) {
					add(class, file, node.Name.Name)
				}
				ast.Inspect(node.Body, func(inner ast.Node) bool {
					literal, ok := inner.(*ast.FuncLit)
					if !ok {
						return true
					}
					for _, class := range funcShapeClasses(literal.Body, literal.Type, state) {
						add(class, file, node.Name.Name+"$closure")
					}
					return true
				})
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range value.Names {
						if class, ok := sharedFunctionSymbols[name.Name]; ok {
							add(class, file, name.Name)
						}
						for _, rhs := range value.Values {
							switch bound := rhs.(type) {
							case *ast.FuncLit:
								for _, class := range funcShapeClasses(bound.Body, bound.Type, state) {
									add(class, file, name.Name)
								}
							case *ast.Ident:
								// `var f = localHelper` rebinds a
								// local implementation: it inherits
								// the helper's shapes. Binding an
								// imported helper (`var f =
								// otherpkg.Helper`) is a selector,
								// not an identifier, and is reuse.
								for _, class := range state.funcShapes[bound.Name] {
									add(class, file, name.Name)
								}
							}
						}
					}
				}
			}
		}
	}
	return sites
}

// funcShapeClasses reports the shape classes of one function or
// closure body.
func funcShapeClasses(body *ast.BlockStmt, sig *ast.FuncType, state *packageShapes) []string {
	if body == nil {
		return nil
	}
	seen := map[string]bool{}
	if bodyBuildsJSONDecoder(body) && bodyCarriesDupSignal(body, state) {
		seen["strict-decoder"] = true
	}
	if bodyCarriesSurrogateSignal(body, state) {
		seen["surrogate-gate"] = true
	}
	if bodyCountsRunes(body) {
		seen["string-measure"] = true
	}
	if bodyIsByteMeasure(body, sig) {
		seen["byte-measure"] = true
	}
	if _, ok := bodyIsVerdictByteMeasure(body); ok {
		seen["byte-measure"] = true
	}
	var classes []string
	for class := range seen {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	return classes
}

// bodyBuildsJSONDecoder reports whether the body constructs an
// encoding/json decoder or unmarshaler.
func bodyBuildsJSONDecoder(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != "json" {
			return true
		}
		if selector.Sel.Name == "NewDecoder" || selector.Sel.Name == "Unmarshal" {
			found = true
			return false
		}
		return true
	})
	return found
}

// bodyHasDupSignal reports whether the body carries a duplicate
// signal: a literal naming both halves, or an identifier with
// the duplicate root (a FaultDuplicate-style constant, a
// duplicate map probe, a dup-check helper name).
func bodyHasDupSignal(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.BasicLit:
			if n.Kind == token.STRING {
				if text, err := strconv.Unquote(n.Value); err == nil {
					if strings.Contains(text, "duplicate") && strings.Contains(text, "member") {
						found = true
						return false
					}
				}
			}
		case *ast.Ident:
			if strings.Contains(n.Name, "uplicate") {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// bodyCarriesDupSignal extends the own-body signal across
// same-package callees: a decoder split into an entry plus a
// local duplicate helper still derives on the entry.
func bodyCarriesDupSignal(body *ast.BlockStmt, state *packageShapes) bool {
	if bodyHasDupSignal(body) {
		return true
	}
	callee := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if state.funcDups[fun.Name] {
				callee = true
				return false
			}
		case *ast.SelectorExpr:
			if state.funcDups[fun.Sel.Name] {
				callee = true
				return false
			}
		}
		return true
	})
	return callee
}

// bodyCarriesSurrogateSignal reports whether the body holds a
// surrogate range literal, names one, or references a
// package-level bound constant.
func bodyCarriesSurrogateSignal(body *ast.BlockStmt, state *packageShapes) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.BasicLit:
			if n.Kind == token.INT {
				if magnitude, err := strconv.ParseUint(n.Value, 0, 32); err == nil && surrogateBounds[magnitude] {
					found = true
					return false
				}
			}
		case *ast.Ident:
			if strings.Contains(strings.ToLower(n.Name), "urrogate") {
				found = true
				return false
			}
			if state.constBound[n.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// bodyCountsRunes reports whether the body counts runes
// directly through the utf8 selector.
func bodyCountsRunes(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selector.Sel.Name == "RuneCountInString" || selector.Sel.Name == "RuneCount" {
			found = true
			return false
		}
		return true
	})
	return found
}

// stringParamNames returns the names of the string-typed
// parameters.
func stringParamNames(sig *ast.FuncType) map[string]bool {
	names := map[string]bool{}
	if sig == nil || sig.Params == nil {
		return names
	}
	for _, field := range sig.Params.List {
		identifier, ok := field.Type.(*ast.Ident)
		if !ok || identifier.Name != "string" {
			continue
		}
		for _, name := range field.Names {
			names[name.Name] = true
		}
	}
	return names
}

// returnsSingleInt reports whether the signature returns exactly
// one int.
func returnsSingleInt(sig *ast.FuncType) bool {
	if sig == nil || sig.Results == nil || len(sig.Results.List) != 1 {
		return false
	}
	identifier, ok := sig.Results.List[0].Type.(*ast.Ident)
	return ok && identifier.Name == "int"
}

// bodyIsByteMeasure reports the func(string)->int texture: a
// string parameter measured by len().
func bodyIsByteMeasure(body *ast.BlockStmt, sig *ast.FuncType) bool {
	if !returnsSingleInt(sig) {
		return false
	}
	params := stringParamNames(sig)
	if len(params) == 0 {
		return false
	}
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok || identifier.Name != "len" || len(call.Args) != 1 {
			return true
		}
		if argument, ok := call.Args[0].(*ast.Ident); ok && params[argument.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// bodyIsVerdictByteMeasure reports the verdict-style texture: a
// len() applied to a variable assigned from a rawString() call.
func bodyIsVerdictByteMeasure(body *ast.BlockStmt) (string, bool) {
	measured := map[string]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, rhs := range assignment.Rhs {
			if !expressionCallsRawString(rhs) {
				continue
			}
			for _, lhs := range assignment.Lhs {
				if identifier, ok := lhs.(*ast.Ident); ok && identifier.Name != "_" && identifier.Name != "ok" {
					measured[identifier.Name] = true
				}
			}
		}
		return true
	})
	if len(measured) == 0 {
		return "", false
	}
	name := ""
	hit := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok || identifier.Name != "len" || len(call.Args) != 1 {
			return true
		}
		if argument, ok := call.Args[0].(*ast.Ident); ok && measured[argument.Name] {
			name, hit = argument.Name, true
			return false
		}
		return true
	})
	return name, hit
}

// expressionCallsRawString reports whether the expression holds
// a rawString() call.
func expressionCallsRawString(expression ast.Expr) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "rawString" {
			found = true
			return false
		}
		return true
	})
	return found
}

// shapeControl is one synthetic control plant: a source file
// whose derived shape and name sites must equal the want sets
// exactly, and whose every derived site must be unregistered —
// the census verdict on a genuine new copy. The extractor under
// test is shapeSitesInPackage, the same pure function the
// production census runs on the real tree, so a control that
// fails here fails the production census by construction.
type shapeControl struct {
	name      string
	source    string
	wantShape []string
	wantName  []string
	// allowEmpty marks the negatives control: clean derivation
	// is the verdict, so the non-empty guard does not apply.
	allowEmpty bool
}

// shapeControls plants every shape the census claims to catch:
// the reviewer's seven in-scope shapes (A, B1-B3, C1-C3) plus
// the split-decoder, const-bound gate, rune measure,
// verdict-style byte measure, local-alias, and fresh-name
// method variants, with negatives that must stay clean.
func shapeControls() []shapeControl {
	decoderBody := `decoder := json.NewDecoder(bytes.NewReader(data))
	seen := map[string]int{}
	for decoder.More() {
		name := "k"
		if _, duplicate := seen[name]; duplicate {
			panic("duplicate member")
		}
		seen[name] = 1
	}
	return seen`
	return []shapeControl{
		{
			name: "fresh decoder spelling",
			source: `package provider
func parseStrictObject(data []byte) map[string]int {
	` + decoderBody + `
}`,
			wantShape: []string{"strict-decoder|provider|control.go|parseStrictObject"},
		},
		{
			name: "split decoder entry plus helper",
			source: `package provider
func decodeEntry(data []byte) map[string]int {
	decoder := json.NewDecoder(bytes.NewReader(data))
	_ = decoder
	return checkNoDup(map[string]int{})
}
func checkNoDup(members map[string]int) map[string]int {
	for name := range members {
		_ = name
		_ = "duplicate member check"
	}
	return members
}`,
			wantShape: []string{"strict-decoder|provider|control.go|decodeEntry"},
		},
		{
			name: "fresh surrogate gate spelling",
			source: `package provider
func scanForLoneSurrogate(data []byte) bool {
	for _, unit := range data {
		if unit == 0xD800 || unit == 0xDC00 {
			return true
		}
	}
	return false
}`,
			wantShape: []string{"surrogate-gate|provider|control.go|scanForLoneSurrogate"},
		},
		{
			name: "const-bound gate under fresh names",
			source: `package provider
const hiStart = 0xD800
const hiEnd = 0xDBFF
func scanFrame(data []byte) bool {
	for _, unit := range data {
		if unit >= hiStart && unit <= hiEnd {
			return true
		}
	}
	return false
}`,
			wantShape: []string{"surrogate-gate|provider|control.go|scanFrame"},
		},
		{
			name: "fresh rune measure",
			source: `package provider
func measureRunes(value string) int { return utf8.RuneCountInString(value) }`,
			wantShape: []string{"string-measure|provider|control.go|measureRunes"},
		},
		{
			name: "fresh byte measure",
			source: `package provider
func measureString(value string) int { return len(value) }`,
			wantShape: []string{"byte-measure|provider|control.go|measureString"},
		},
		{
			name: "verdict-style byte measure",
			source: `package provider
func checkByteBounds(raw json.RawMessage, minimum, maximum int) (string, bool) {
	value, ok := rawString(raw)
	if !ok || len(value) < minimum || len(value) > maximum {
		return "", false
	}
	return value, true
}`,
			wantShape: []string{"byte-measure|provider|control.go|checkByteBounds"},
		},
		{
			name: "var alias of ledgered name",
			source: `package provider
var stringLength = byteLength`,
			wantShape: []string{"string-measure|provider|control.go|stringLength"},
		},
		{
			// The name layer stays blind to the var binding
			// (it derives FuncDecl only) while the shape
			// layer fires on the closure: the exact B1 gap,
			// closed by derivation rather than assertion.
			name: "var alias closure with decoder shape",
			source: `package provider
var decodeStrict = func(data []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	_ = decoder
	_ = "duplicate member"
	return false
}`,
			wantShape: []string{"strict-decoder|provider|control.go|decodeStrict"},
		},
		{
			name: "var alias of shape-positive local",
			source: `package provider
func byteLength(value string) int { return len(value) }
var byteLen = byteLength`,
			wantShape: []string{
				"byte-measure|provider|control.go|byteLen",
				"byte-measure|provider|control.go|byteLength",
			},
		},
		{
			name: "fresh gate method spelling",
			source: `package provider
type frameScanner struct{}
func (frameScanner) scanEscapes(data []byte) bool {
	for _, unit := range data {
		if unit >= 0xDC00 && unit <= 0xDFFF {
			return true
		}
	}
	return false
}`,
			wantShape: []string{"surrogate-gate|provider|control.go|scanEscapes"},
		},
		{
			name: "ledgered name method still fires by name",
			source: `package provider
type surrogateScanner struct{}
func (surrogateScanner) hasLoneSurrogateEscape(data []byte) bool { return len(data) == 0 }`,
			wantName: []string{"surrogate-gate|provider|control.go|hasLoneSurrogateEscape"},
		},
		{
			name: "ledgered decoder name still fires by name",
			source: `package provider
func decodeStrictObject(data []byte) bool { return len(data) == 0 }`,
			wantName: []string{"strict-decoder|provider|control.go|decodeStrictObject"},
		},
		{
			name: "negatives stay clean",
			source: `package provider
func decodeArrayShaped(raw json.RawMessage) []int {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_ = decoder
	return nil
}
func asciiNameOk(name string) bool { return len(name) >= 1 && len(name) <= 128 }
func countElements(values []string) int { return len(values) }
var reuse = environ.StringLength`,
			allowEmpty: true,
		},
	}
}

// mirrorNameSites replicates the name layer's FuncDecl rule on
// one synthetic file: declarations whose identifiers are already
// in sharedFunctionSymbols. The production name derivation in
// census_test.go applies this same rule to the real tree.
func mirrorNameSites(source []byte) []string {
	syntax, err := parser.ParseFile(token.NewFileSet(), "control.go", source, 0)
	if err != nil {
		return []string{"parse-failure|provider|control.go|unparseable"}
	}
	var keys []string
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.FuncDecl)
		if !ok || node.Name == nil {
			continue
		}
		if class, ok := sharedFunctionSymbols[node.Name.Name]; ok {
			keys = append(keys, shapeSite{class: class, pkg: "provider", file: "control.go", symbol: node.Name.Name}.key())
		}
	}
	sort.Strings(keys)
	return keys
}

// TestShapeCensusCatchesControls drives every control plant
// through the production extractor and requires the exact
// derived sets, with every derived site unregistered: the proof
// that each claimed shape fails the census rather than passing
// silently.
func TestShapeCensusCatchesControls(t *testing.T) {
	for _, control := range shapeControls() {
		t.Run(control.name, func(t *testing.T) {
			seenShape := map[string]bool{}
			var gotShape []string
			for _, site := range shapeSitesInPackage("provider", map[string][]byte{"control.go": []byte(control.source)}) {
				if !seenShape[site.key()] {
					seenShape[site.key()] = true
					gotShape = append(gotShape, site.key())
				}
			}
			sort.Strings(gotShape)
			if strings.Join(gotShape, "\x00") != strings.Join(control.wantShape, "\x00") {
				t.Fatalf("shape sites = %q, want %q", gotShape, control.wantShape)
			}
			gotName := mirrorNameSites([]byte(control.source))
			if strings.Join(gotName, "\x00") != strings.Join(control.wantName, "\x00") {
				t.Fatalf("name sites = %q, want %q", gotName, control.wantName)
			}
			for _, key := range append(append([]string{}, gotShape...), gotName...) {
				if _, ok := shapeLedger[key]; ok {
					t.Fatalf("control site %q is registered; the control proves nothing", key)
				}
				if _, ok := sharedLedger[key]; ok {
					t.Fatalf("control site %q is registered; the control proves nothing", key)
				}
			}
			if !control.allowEmpty && len(gotShape)+len(gotName) == 0 {
				t.Fatal("control derives no site; the control proves nothing")
			}
		})
	}
}

// TestGrammarCensusCatchesFreshName plants a fresh-name copy of
// the environment-id grammar and requires the classifier to
// report it as a copy of that class with an unregistered key:
// the proof that a renamed grammar copy fails the census.
func TestGrammarCensusCatchesFreshName(t *testing.T) {
	literals, _, _ := deriveGrammarLiterals(t)
	known := grammarKnownLiterals(literals)
	environmentID, ok := knownLiteralFor(t, literals, "env-id-grammar")
	if !ok {
		t.Fatal("env-id-grammar derives no literal; the control has nothing to copy")
	}
	source := []byte("package provider\nvar envIDCopy = regexp.MustCompile(`" + environmentID + "`)\n")
	candidates, err := grammarCandidatesInFile("control.go", source)
	if err != nil {
		t.Fatalf("control does not parse: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("control derives %d candidates, want 1", len(candidates))
	}
	class, copy := classifyGrammarCandidate(candidates[0], known)
	if !copy || class != "env-id-grammar" {
		t.Fatalf("fresh-name copy classified as (%q, %v), want (env-id-grammar, true)", class, copy)
	}
	key := shapeSite{class: class, pkg: "provider", file: "control.go", symbol: candidates[0].symbol}.key()
	if grammarSiteLedger[key] {
		t.Fatalf("control site %q is registered; the control proves nothing", key)
	}
	negative, err := grammarCandidatesInFile("control.go", []byte("package provider\nvar sitePattern = regexp.MustCompile(`^[a-z]+$`)\n"))
	if err != nil {
		t.Fatalf("negative does not parse: %v", err)
	}
	if _, copy := classifyGrammarCandidate(negative[0], known); copy {
		t.Fatal("unknown-literal grammar classified as a copy; the rule overreaches")
	}
}

// knownLiteralFor returns one derived literal of the class.
func knownLiteralFor(t *testing.T, literals map[string]map[string]bool, class string) (string, bool) {
	t.Helper()
	for text := range literals[class] {
		return text, true
	}
	return "", false
}

// TestDelegatingWrappersCallEnviron pins the B2 fix
// structurally: the sessadapter and dirnode frame decoders must
// delegate to environ, and the removed raw-scan copies must not
// come back. The wrapper body must reference the environ
// package and must not construct a decoder, a surrogate bound,
// or a rune count of its own; neither file may define the old
// gate or helper names or carry surrogate range literals.
func TestDelegatingWrappersCallEnviron(t *testing.T) {
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("delegation: %v", err)
	}
	for _, pkg := range []string{"sessadapter", "dirnode"} {
		t.Run(pkg, func(t *testing.T) {
			path := filepath.Join(root, pkg, "decode.go")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("delegation: %v", err)
			}
			syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
			if err != nil {
				t.Fatalf("delegation: %v", err)
			}
			foundWrapper := false
			for _, decl := range syntax.Decls {
				node, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				switch node.Name.Name {
				case "hasLoneSurrogateEscape", "readUTF16Escape", "readUTF16EscapeAfter":
					t.Fatalf("delegation: removed raw-scan copy %q is back in %s/decode.go", node.Name.Name, pkg)
				case "decodeStrictObject":
					foundWrapper = true
					pinsDelegation(t, pkg, node)
				}
			}
			if !foundWrapper {
				t.Fatalf("delegation: decodeStrictObject missing in %s/decode.go", pkg)
			}
			assertNoSurrogateLiterals(t, pkg, syntax)
		})
	}
}

// delegatedCheckHelpers names every sessadapter and dirnode
// decode.go helper that must delegate to environ rather than
// reimplement the rule. The frame decoder above is pinned
// separately; these are the per-helper copies this leaf
// converged. A helper that regrows a local rule fails here
// structurally, before any behavioral battery runs.
var delegatedCheckHelpers = []string{
	"stringLength",
	"checkStringBounds",
	"checkUint53Bounds",
	"checkDigest",
	"checkUUIDv7",
	"checkTimestamp",
	"checkSortedUniqueStrings",
	"checkSortedUniqueDigests",
	"checkExtensions",
}

// removedGrammarCopies names the grammar variables the converged
// helpers deleted. A revived copy fails here as well as in the
// grammar census, so the failure names the file, not just the
// class.
var removedGrammarCopies = []string{
	"semverPattern",
	"environmentIDPattern",
	"reverseDNSPattern",
}

// TestCheckHelpersDelegateToEnviron pins the per-helper
// convergence structurally: every delegated helper must reference
// the environ package and must hold no decoder, surrogate gate,
// rune count, or compiled grammar of its own, and the removed
// grammar copies must not come back. dirnode's checkSemver and
// checkEnvironmentID delegate the same way through local names
// the shared list does not ledger.
func TestCheckHelpersDelegateToEnviron(t *testing.T) {
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("check-delegation: %v", err)
	}
	for _, pkg := range []string{"sessadapter", "dirnode"} {
		t.Run(pkg, func(t *testing.T) {
			path := filepath.Join(root, pkg, "decode.go")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("check-delegation: %v", err)
			}
			syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
			if err != nil {
				t.Fatalf("check-delegation: %v", err)
			}
			found := map[string]bool{}
			for _, decl := range syntax.Decls {
				node, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				for _, name := range delegatedCheckHelpers {
					if node.Name.Name == name {
						found[name] = true
						pinsCheckDelegation(t, pkg, node)
					}
				}
				if pkg == "dirnode" && (node.Name.Name == "checkSemver" || node.Name.Name == "checkEnvironmentID") {
					found[node.Name.Name] = true
					pinsCheckDelegation(t, pkg, node)
				}
			}
			for _, name := range delegatedCheckHelpers {
				if !found[name] {
					t.Fatalf("check-delegation: %s helper %q missing in %s/decode.go", pkg, name, pkg)
				}
			}
			if pkg == "dirnode" && (!found["checkSemver"] || !found["checkEnvironmentID"]) {
				t.Fatalf("check-delegation: dirnode grammar helpers missing in dirnode/decode.go")
			}
			assertNoRevivedGrammarCopies(t, pkg, syntax)
		})
	}
}

// checkHelperDelegation reports why the helper is not a pure
// load-bearing delegation onto its environ twin, or "" when it
// is. A pure delegation is exactly `return environ.Twin(args)`
// as the body's only statement: the helper's result IS the
// owner's answer. A discarded call (`_, _ = environ.Twin(...)`)
// plus a regrown local rule preserves the searched-for token
// while the helper's result no longer depends on the owner at
// all — the decoy the review planted green — so anything but a
// directly returned single environ call fails here. The twin
// name must match the wrapper name: delegating to the wrong
// environ member preserves the token while changing behavior.
// This check is pure over the declaration so the synthetic
// controls in TestDelegationPinsAreLoadBearing drive this exact
// function: a decoy that passes here passes the production
// census by construction.
func checkHelperDelegation(wrapper *ast.FuncDecl) string {
	if wrapper.Body == nil || len(wrapper.Body.List) != 1 {
		return "holds more than the single delegating return"
	}
	ret, ok := wrapper.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "does not return exactly one result"
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return "does not return a call"
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "does not return an environ call"
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok || identifier.Name != "environ" {
		return "does not return an environ call"
	}
	twin := wrapper.Name.Name
	if len(twin) > 0 && twin[0] >= 'a' && twin[0] <= 'z' {
		twin = string(twin[0]-'a'+'A') + twin[1:]
	}
	if selector.Sel.Name != twin {
		return "delegates to environ." + selector.Sel.Name + ", want the " + twin + " twin"
	}
	return ""
}

// pinsCheckDelegation requires the helper to reference environ and
// to hold no rule of its own: no decoder, no surrogate signal, no
// rune count, and no compiled grammar. The reference alone is not
// enough — a discarded call satisfies it while regrowing the rule
// — so the helper must additionally be a pure load-bearing
// delegation per checkHelperDelegation: the owner's answer must
// be the helper's result.
func pinsCheckDelegation(t *testing.T, pkg string, wrapper *ast.FuncDecl) {
	t.Helper()
	if failure := checkHelperDelegation(wrapper); failure != "" {
		t.Fatalf("check-delegation: %s %s is not a load-bearing delegation: %s", pkg, wrapper.Name.Name, failure)
	}
	if bodyBuildsJSONDecoder(wrapper.Body) {
		t.Fatalf("check-delegation: %s %s builds its own decoder", pkg, wrapper.Name.Name)
	}
	surrogateState := &packageShapes{funcDups: map[string]bool{}, funcShapes: map[string][]string{}, constBound: map[string]bool{}}
	if bodyCarriesSurrogateSignal(wrapper.Body, surrogateState) {
		t.Fatalf("check-delegation: %s %s carries its own surrogate gate", pkg, wrapper.Name.Name)
	}
	if bodyCountsRunes(wrapper.Body) {
		t.Fatalf("check-delegation: %s %s counts runes itself", pkg, wrapper.Name.Name)
	}
	ast.Inspect(wrapper.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selector.Sel.Name == "MustCompile" {
			t.Fatalf("check-delegation: %s %s compiles its own grammar", pkg, wrapper.Name.Name)
		}
		return true
	})
}

// assertNoRevivedGrammarCopies requires the converged grammar
// variables to stay deleted from the package decode.go file.
func assertNoRevivedGrammarCopies(t *testing.T, pkg string, syntax *ast.File) {
	t.Helper()
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				for _, removed := range removedGrammarCopies {
					if name.Name == removed {
						t.Fatalf("check-delegation: removed grammar copy %q is back in %s/decode.go", removed, pkg)
					}
				}
			}
		}
	}
}

// checkWrapperDelegation reports why the frame-decoder wrapper is
// not a load-bearing delegation onto environ.DecodeStrictObject,
// or "" when it is. The wrapper's verdict must depend on the
// owner's answer: exactly one environ call, to DecodeStrictObject,
// assigned to non-blank names that the body reuses, with no other
// call in the body. A discarded call (`_, _ =
// environ.DecodeStrictObject(data)`) preserves the token while
// the wrapper's verdict no longer depends on the owner, and a
// fault-ignoring wrapper (`members, _ := ...; return members,
// nil`) admits every malformed frame while keeping the call —
// so blanks, unused answers, and extra calls all fail here. This
// check is pure over the declaration so the synthetic controls
// in TestDelegationPinsAreLoadBearing drive this exact function.
func checkWrapperDelegation(wrapper *ast.FuncDecl) string {
	if wrapper.Body == nil {
		return "has no body"
	}
	var owned []*ast.CallExpr
	ast.Inspect(wrapper.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != "environ" {
			return true
		}
		owned = append(owned, call)
		return true
	})
	if len(owned) != 1 {
		return "holds an environ call count the delegation cannot attribute"
	}
	selector := owned[0].Fun.(*ast.SelectorExpr)
	if selector.Sel.Name != "DecodeStrictObject" {
		return "delegates to environ." + selector.Sel.Name + ", want the DecodeStrictObject verdict"
	}
	assigned := map[string]int{}
	attributed := false
	for _, stmt := range wrapper.Body.List {
		assignment, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}
		for _, rhs := range assignment.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok || call != owned[0] {
				continue
			}
			attributed = true
			for _, lhs := range assignment.Lhs {
				identifier, ok := lhs.(*ast.Ident)
				if !ok {
					return "assigns the owner verdict to a non-name"
				}
				if identifier.Name == "_" {
					return "discards the owner verdict"
				}
				assigned[identifier.Name]++
			}
		}
	}
	if !attributed {
		return "never attributes the owner verdict to a name"
	}
	uses := map[string]int{}
	ast.Inspect(wrapper.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		uses[identifier.Name]++
		return true
	})
	for name := range assigned {
		if uses[name] < 2 {
			return "ignores the owner answer " + name
		}
	}
	calls := 0
	ast.Inspect(wrapper.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.CallExpr); ok {
			calls++
		}
		return true
	})
	if calls != 1 {
		return "holds a rule of its own beside the delegation"
	}
	return ""
}

// pinsDelegation requires the wrapper to reference environ and
// to hold no decoder, gate, or measure of its own. The reference
// alone is not enough — a discarded call satisfies it while the
// verdict no longer depends on the owner — so the wrapper must
// additionally be load-bearing per checkWrapperDelegation.
func pinsDelegation(t *testing.T, pkg string, wrapper *ast.FuncDecl) {
	t.Helper()
	if failure := checkWrapperDelegation(wrapper); failure != "" {
		t.Fatalf("delegation: %s decodeStrictObject is not a load-bearing delegation: %s", pkg, failure)
	}
	if bodyBuildsJSONDecoder(wrapper.Body) {
		t.Fatalf("delegation: %s decodeStrictObject builds its own decoder", pkg)
	}
	if bodyHasDupSignal(wrapper.Body) {
		t.Fatalf("delegation: %s decodeStrictObject carries its own duplicate rule", pkg)
	}
	if bodyCountsRunes(wrapper.Body) {
		t.Fatalf("delegation: %s decodeStrictObject counts runes itself", pkg)
	}
	surrogateState := &packageShapes{funcDups: map[string]bool{}, funcShapes: map[string][]string{}, constBound: map[string]bool{}}
	if bodyCarriesSurrogateSignal(wrapper.Body, surrogateState) {
		t.Fatalf("delegation: %s decodeStrictObject carries its own surrogate gate", pkg)
	}
}

// delegationControl is one synthetic control for the load-bearing
// delegation pins: a function source whose check verdict must
// equal the want ("" for clean delegations, a failure fragment
// otherwise). The controls drive checkHelperDelegation and
// checkWrapperDelegation, the same pure functions the production
// census runs on the real tree, so a decoy that passes here
// passes the production gate by construction. Every reporting
// control preserves the searched-for `environ` token: the proof
// that mentioning the owner is not using it.
type delegationControl struct {
	name    string
	source  string
	helper  string
	wrapper string
	want    string
}

// delegationControls plants the decoy shapes against both pins:
// the discarded direct call plus a regrown local copy, the
// fault-ignoring wrapper, the wrong-twin delegation, and the
// token-free local rule, with clean delegations that must stay
// clean.
func delegationControls() []delegationControl {
	return []delegationControl{
		{
			name:   "clean helper delegation stays clean",
			source: "package provider\nfunc checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\treturn environ.CheckDigest(raw)\n}",
			helper: "checkDigest",
			want:   "",
		},
		{
			name:   "helper decoy discards the owner answer",
			source: "package provider\nfunc checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\t_, _ = environ.CheckDigest(raw)\n\tvalue, ok := rawString(raw)\n\tif !ok {\n\t\treturn scalar.Digest{}, false\n\t}\n\tdigest, err := scalar.ParseDigest(value)\n\tif err != nil {\n\t\treturn scalar.Digest{}, false\n\t}\n\treturn digest, true\n}",
			helper: "checkDigest",
			want:   "single delegating return",
		},
		{
			name:   "helper decoy without the drift still reports",
			source: "package provider\nfunc checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\t_, _ = environ.CheckDigest(raw)\n\treturn scalar.ParseDigest(value)\n}",
			helper: "checkDigest",
			want:   "single delegating return",
		},
		{
			name:   "helper delegating to the wrong twin reports",
			source: "package provider\nfunc checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\treturn environ.CheckUUIDv7(raw)\n}",
			helper: "checkDigest",
			want:   "want the CheckDigest twin",
		},
		{
			name:   "helper token-free local rule reports",
			source: "package provider\nfunc checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\treturn scalar.ParseDigest(value)\n}",
			helper: "checkDigest",
			want:   "environ call",
		},
		{
			name:    "clean wrapper delegation stays clean",
			source:  "package provider\nfunc decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {\n\tmembers, fault := environ.DecodeStrictObject(data)\n\tif fault != nil {\n\t\treturn nil, &frameFault{detail: fault.Detail, member: fault.Member}\n\t}\n\treturn members, nil\n}",
			wrapper: "decodeStrictObject",
			want:    "",
		},
		{
			name:    "wrapper decoy discards the owner verdict",
			source:  "package provider\nfunc decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {\n\t_, _ = environ.DecodeStrictObject(data)\n\tvar members map[string]json.RawMessage\n\tif err := json.Unmarshal(data, &members); err != nil {\n\t\treturn nil, &frameFault{detail: \"not a JSON object\"}\n\t}\n\treturn members, nil\n}",
			wrapper: "decodeStrictObject",
			want:    "discards the owner verdict",
		},
		{
			name:    "wrapper ignoring the fault reports",
			source:  "package provider\nfunc decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {\n\tmembers, _ := environ.DecodeStrictObject(data)\n\treturn members, nil\n}",
			wrapper: "decodeStrictObject",
			want:    "discards the owner verdict",
		},
		{
			name:    "wrapper delegating to the wrong member reports",
			source:  "package provider\nfunc decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {\n\tmembers, fault := environ.CheckExtensions(data)\n\tif fault != nil {\n\t\treturn nil, &frameFault{detail: \"bad\"}\n\t}\n\treturn members, nil\n}",
			wrapper: "decodeStrictObject",
			want:    "want the DecodeStrictObject verdict",
		},
	}
}

// parseControlFunc extracts one named function declaration from a
// synthetic control source.
func parseControlFunc(t *testing.T, source, name string) *ast.FuncDecl {
	t.Helper()
	syntax, err := parser.ParseFile(token.NewFileSet(), "control.go", []byte(source), 0)
	if err != nil {
		t.Fatalf("control does not parse: %v", err)
	}
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.FuncDecl)
		if !ok || node.Name.Name != name {
			continue
		}
		return node
	}
	t.Fatalf("control holds no function %q", name)
	return nil
}

// TestDelegationPinsAreLoadBearing drives every delegation
// control through the production pin functions and requires the
// ledgered verdict: clean delegations stay clean, and every
// token-preserving decoy reports. A decoy that stopped
// reporting proves nothing and fails here by construction.
func TestDelegationPinsAreLoadBearing(t *testing.T) {
	for _, control := range delegationControls() {
		t.Run(control.name, func(t *testing.T) {
			target := control.helper
			if target == "" {
				target = control.wrapper
			}
			node := parseControlFunc(t, control.source, target)
			var got string
			if control.helper != "" {
				got = checkHelperDelegation(node)
			} else {
				got = checkWrapperDelegation(node)
			}
			if control.want == "" {
				if got != "" {
					t.Fatalf("clean delegation reported: %q", got)
				}
				return
			}
			if !strings.Contains(got, control.want) {
				t.Fatalf("decoy reported %q, want a failure containing %q", got, control.want)
			}
		})
	}
}

// assertNoSurrogateLiterals requires the whole file to carry no
// surrogate range literal: the gate lives in environ now.
func assertNoSurrogateLiterals(t *testing.T, pkg string, syntax *ast.File) {
	t.Helper()
	ast.Inspect(syntax, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.INT {
			return true
		}
		if magnitude, err := strconv.ParseUint(literal.Value, 0, 32); err == nil && surrogateBounds[magnitude] {
			t.Fatalf("delegation: %s/decode.go carries surrogate literal %s; the gate lives in environ", pkg, literal.Value)
			return false
		}
		return true
	})
}
