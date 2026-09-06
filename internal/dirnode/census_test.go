package dirnode

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
)

// This file is the derived refusal-arm inventory. It exists
// because a new refusal through an existing site — a widened
// member set, a new detail literal, a dropped check that slides
// to a lower arm — leaves the whole suite green unless the arm
// set is derived from production source by parsing it, never by
// listing it.
//
// Derivation (deriveRefusalArms): every call to one of the eight
// refusal constructors with a string-literal first argument
// yields ctor|<constructor>|<detail>; a call whose first argument
// is a binary expression yields
// ctor|<constructor>|expr|<function>|<rendered>, where the render
// keeps literal text, identifier names, and selector paths, so a
// context-prefixed envelope arm and a fault-detail conduit never
// merge with each other or with a literal arm; every
// &frameFault{...} literal yields frame|<detail>, except the
// environ-delegation bridge, which yields one frame arm per
// environ fault detail. A constructor
// call with any other first-argument shape is a derivation
// violation, not a silent pass. Constructor references outside
// direct-call position fail the derivation outright: an aliased
// constructor hides every arm it builds. All alias spellings are
// caught — `alias := failX`, `var alias = failX` at package
// level, `var alias = failX` inside a function body, arguments,
// returns, struct fields, assignments — because the gate exempts
// only the definition's own Name in its ValueSpec, never a
// constructor identifier appearing anywhere else. The
// constructors of this package are themselves declared as
// `var failX = func...`, so their definition Names stay exempt
// while any other reference fails. A refusal call inside any
// function literal that is not a registered constructor body
// fails: an arm that fires only when a closure runs is not a
// call-site arm.
//
// The constructor denominator is itself derived, not listed on
// trust: TestRefusalConstructorsMatchProduction requires the
// registered set to equal the eight named constructors, every
// axerror.New site in production to sit inside one registered
// body, and no func-value indirection of axerror.New anywhere.
// A ninth constructor, an inline axerror.New refusal, or a
// `var newFault = axerror.New` indirection carries an additive
// arm the derivation never observes, and fails there instead of
// passing silently.
//
// Both directions are checked:
// TestDerivedRefusalArmsAreAllWitnessed (every derived arm
// carries a witness) and TestWitnessedArmsAreAllDerived (every
// witness names a derived arm, so a truncated derivation reddens
// on the orphaned witnesses instead of passing vacuously).
// TestEveryArmWitnessRefusesAtTheProductionEntry is intentionally
// a resolve-and-run check rather than prose: every witness names
// test functions that exist in this package's test files, and
// those tests drive the production entries with negative vectors
// asserting code plus distinguishing detail. Empty, single-file,
// or unparseable derivations fail closed: a domain that silently
// derives nothing is not a measurement.
//
// Shape space. An error construction in this package can only be
// carried by these AST shapes; the census derives or refuses each
// one:
//
//  1. A direct `failX("literal", ...)` call — DERIVED as an arm
//     keyed by the literal detail.
//  2. A direct `failX(<binary>, ...)` call — DERIVED as an arm
//     keyed by function and rendered expression. The two
//     sub-shapes in production are the fault-detail conduit
//     (`"prefix "+fault.detail`, eleven sites) and the
//     envelope-branch prefix (`context+" suffix"`, eleven
//     sites in checkResponseIdentity); both render distinctly,
//     so neither merges.
//  3. A direct `failX(<other>, ...)` call — REFUSED as a
//     derivation violation (zero such sites exist; the first
//     one fails the census instead of passing silently).
//  4. A package-level `var failY = func...` literal whose body
//     calls axerror.New — DERIVED as a constructor: the
//     produced set must equal refusalConstructors exactly, so
//     a ninth constructor fails as unregistered.
//  5. An axerror.New call outside every registered literal —
//     REFUSED as a construction site outside the registered
//     bodies.
//  6. An axerror.New selector in non-call position (alias var,
//     returned constructor, argument, struct field) — REFUSED
//     as a func-value indirection (the import census pins the
//     `axerror` binding it matches on).
//  7. A failX identifier in non-call, non-definition position
//     (alias var/:=, assignment, argument, return, struct
//     field) — REFUSED as a constructor alias that would bury
//     every arm built through it.
//  8. A failX call inside a non-constructor function literal —
//     REFUSED as a closure-hidden arm.
//  9. An import of the error package path under any other name
//     — REFUSED by the import census
//     (TestAxerrorImportsAreUnaliased pins the `axerror`
//     binding every identifier match above relies on).
//  10. A &frameFault{...} literal — DERIVED as a frame arm by
//     its literal detail.
//
// 10 of 10 shapes derived-or-refused. The residue, stated here
// rather than left silent: a frameFault built through a type
// alias (`type ff = frameFault`) would hide from shape 10 — no
// such alias exists in production (the derivation fails on any
// CompositeLit it cannot classify, but an alias retypes the
// node it matches, so this bound is textual: grep finds no
// `frameFault` alias); a construction via assembly, linkname,
// or reflect would hide from shapes 5-6 — the package uses none
// of those mechanisms; files outside the package directory and
// _test.go files, which the census never scans, must not carry
// production refusals — TestCensusScansEveryProductionFile
// requires the scanned set to equal the production file list
// the package build sees; a second argument that smuggles the
// distinguishing fact while the first stays constant would make
// two arms share one key — the constructors' second parameters
// are wire facts asserted selectively by the entry tests, never
// the arm identity, and the witness table records the
// code-plus-detail pair per arm.
//
// Census-only kills score separately from behavioural ones:
// every arm the derivation observes reddens the census by
// construction when its site is edited, and that reddening
// proves the census runs, not that the gate refuses. Behavioural
// proof stays with each row's named witness, which the inventory
// resolves textually and runs as part of the suite.

// refusalConstructors is the exact registered constructor set.
// Each must be a package-level `var failX = func...` whose body
// calls axerror.New; the derivation verifies both halves.
var refusalConstructors = []string{
	"failInvalid",
	"failViolation",
	"failUnknownOperation",
	"failDowngrade",
	"failIntegrity",
	"failIdempotency",
	"failQuery",
	"failTransport",
}

// parseGoFile parses one Go source file for the censuses.
func parseGoFile(path string, source []byte) (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), path, source, 0)
}

// productionFiles returns the production file paths of this
// package directory: every non-test Go file. Test files are
// never scanned: a refusal built in a test helper is not a
// production arm.
func productionFiles(t *testing.T) []string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(directory, name))
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("census scanned zero production files; the scanner is broken, not the package")
	}
	return files
}

// armDerivation is the derived inventory plus every shape
// violation found along the way.
type armDerivation struct {
	arms       map[string]bool
	ctors      map[string]bool
	violations []string
}

// deriveRefusalArms parses every production file and derives the
// constructor set, the arm set, and the violation list.
func deriveRefusalArms(t *testing.T) armDerivation {
	t.Helper()
	derived := armDerivation{arms: map[string]bool{}, ctors: map[string]bool{}}
	for _, path := range productionFiles(t) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("census: %v", err)
		}
		syntax, err := parseGoFile(path, source)
		if err != nil {
			t.Fatalf("census: %v", err)
		}
		scan := fileScan{derived: &derived, name: filepath.Base(path)}
		scan.registerConstructors(syntax)
		scan.walkFile(syntax)
	}
	if len(derived.ctors) == 0 {
		t.Fatal("census derived zero constructors; the scanner is broken, not the package")
	}
	if len(derived.arms) == 0 {
		t.Fatal("census derived zero arms; the scanner is broken, not the package")
	}
	return derived
}

// fileScan holds per-file census state.
type fileScan struct {
	derived   *armDerivation
	name      string
	ctors     map[string]bool
	functions []string
}

// callFunction names the function enclosing the call under
// visit: the nearest function declaration, the closure marker
// for a non-constructor literal, or the package marker for a
// package-level initializer. A closure-hidden arm carries a key
// no witness table row can match by accident, so the first one
// fails the witnessed-direction test instead of passing
// silently.
func (scan *fileScan) callFunction() string {
	if len(scan.functions) == 0 {
		return "package"
	}
	return scan.functions[len(scan.functions)-1]
}

// registerConstructors records every package-level
// `var failX = func...` literal whose body calls axerror.New.
func (scan *fileScan) registerConstructors(syntax *ast.File) {
	for _, declaration := range syntax.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			literal, ok := value.Values[0].(*ast.FuncLit)
			if !ok {
				continue
			}
			if !containsAxerrorNew(literal.Body) {
				continue
			}
			scan.derived.ctors[value.Names[0].Name] = true
		}
	}
}

// containsAxerrorNew reports whether the body calls axerror.New.
func containsAxerrorNew(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "axerror" && selector.Sel.Name == "New" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// walkFile derives arms and violations with parent tracking: the
// stack top is the parent of the node under visit.
func (scan *fileScan) walkFile(syntax *ast.File) {
	var stack []ast.Node
	ctorDepth := 0
	ctorLiterals := map[*ast.FuncLit]bool{}
	for _, declaration := range syntax.Decls {
		if general, ok := declaration.(*ast.GenDecl); ok && general.Tok == token.VAR {
			for _, specification := range general.Specs {
				if value, ok := specification.(*ast.ValueSpec); ok && len(value.Values) == 1 {
					if literal, ok := value.Values[0].(*ast.FuncLit); ok && scan.derived.ctors[value.Names[0].Name] {
						ctorLiterals[literal] = true
					}
				}
			}
		}
	}
	ast.Inspect(syntax, func(node ast.Node) bool {
		if node == nil {
			popped := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if literal, ok := popped.(*ast.FuncLit); ok && ctorLiterals[literal] {
				ctorDepth--
			}
			switch popped.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				scan.functions = scan.functions[:len(scan.functions)-1]
			}
			return true
		}
		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		if literal, ok := node.(*ast.FuncLit); ok && ctorLiterals[literal] {
			ctorDepth++
		}
		switch node := node.(type) {
		case *ast.FuncDecl:
			scan.functions = append(scan.functions, node.Name.Name)
		case *ast.FuncLit:
			if ctorLiterals[node] {
				scan.functions = append(scan.functions, "ctor")
			} else {
				scan.functions = append(scan.functions, "closure")
			}
		}
		scan.visit(node, parent, ctorDepth)
		stack = append(stack, node)
		return true
	})
}

// visit derives or refuses one node.
func (scan *fileScan) visit(node, parent ast.Node, ctorDepth int) {
	switch node := node.(type) {
	case *ast.CallExpr:
		scan.visitCall(node, parent, ctorDepth)
	case *ast.Ident:
		scan.visitIdent(node, parent)
	case *ast.SelectorExpr:
		scan.visitSelector(node, parent)
	case *ast.CompositeLit:
		scan.visitComposite(node, parent)
	}
}

// visitCall handles constructor calls and axerror.New calls.
func (scan *fileScan) visitCall(call *ast.CallExpr, parent ast.Node, ctorDepth int) {
	if identifier, ok := call.Fun.(*ast.Ident); ok && isConstructorName(identifier.Name) {
		scan.visitConstructorCall(call, identifier.Name)
		return
	}
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "axerror" && selector.Sel.Name == "New" {
			if ctorDepth == 0 {
				scan.derived.violations = append(scan.derived.violations, scan.name+": axerror.New call outside every registered constructor body")
			}
		}
	}
}

// isConstructorName reports whether the identifier names a
// derived constructor. The import census pins the axerror
// binding; the constructor census pins this set against the
// registered table, so matching on the name cannot drift from
// either without reddening its own gate first.
func isConstructorName(name string) bool {
	for _, registered := range refusalConstructors {
		if name == registered {
			return true
		}
	}
	return false
}

// visitConstructorCall derives one constructor-call arm. A call
// inside a non-constructor function literal is refused outright:
// an arm that fires only when a closure runs is not a call-site
// arm, and keying it would pretend otherwise.
func (scan *fileScan) visitConstructorCall(call *ast.CallExpr, name string) {
	if scan.callFunction() == "closure" {
		scan.derived.violations = append(scan.derived.violations, scan.name+": "+name+" call inside a function literal")
		return
	}
	if len(call.Args) == 0 {
		scan.derived.violations = append(scan.derived.violations, scan.name+": "+name+" call with no arguments")
		return
	}
	switch first := call.Args[0].(type) {
	case *ast.BasicLit:
		if first.Kind != token.STRING {
			scan.derived.violations = append(scan.derived.violations, scan.name+": "+name+" call with non-string literal first argument")
			return
		}
		detail, err := strconv.Unquote(first.Value)
		if err != nil {
			scan.derived.violations = append(scan.derived.violations, scan.name+": "+name+" call with unquotable detail")
			return
		}
		scan.derived.arms["ctor|"+name+"|"+detail] = true
	case *ast.BinaryExpr:
		rendered, ok := renderExpr(first)
		if !ok {
			scan.derived.violations = append(scan.derived.violations, scan.name+": "+name+" call with unrenderable first argument")
			return
		}
		scan.derived.arms["ctor|"+name+"|expr|"+scan.callFunction()+"|"+rendered] = true
	default:
		scan.derived.violations = append(scan.derived.violations, scan.name+": "+name+" call with non-literal, non-binary first argument")
	}
}

// renderExpr renders a first-argument expression into a stable
// key: string literals by value, identifiers and selectors by
// path. Only + compositions render; anything else is a
// violation, not a silent bucket.
func renderExpr(node ast.Expr) (string, bool) {
	switch node := node.(type) {
	case *ast.BasicLit:
		if node.Kind != token.STRING {
			return "", false
		}
		detail, err := strconv.Unquote(node.Value)
		if err != nil {
			return "", false
		}
		return strconv.Quote(detail), true
	case *ast.Ident:
		return node.Name, true
	case *ast.SelectorExpr:
		identifier, ok := node.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		return identifier.Name + "." + node.Sel.Name, true
	case *ast.BinaryExpr:
		if node.Op != token.ADD {
			return "", false
		}
		left, ok := renderExpr(node.X)
		if !ok {
			return "", false
		}
		right, ok := renderExpr(node.Y)
		if !ok {
			return "", false
		}
		return left + "+" + right, true
	default:
		return "", false
	}
}

// visitIdent refuses constructor aliases: any constructor
// identifier outside direct-call position and outside its own
// definition Name.
func (scan *fileScan) visitIdent(identifier *ast.Ident, parent ast.Node) {
	if !isConstructorName(identifier.Name) {
		return
	}
	if call, ok := parent.(*ast.CallExpr); ok {
		if fun, ok := call.Fun.(*ast.Ident); ok && fun == identifier {
			return
		}
	}
	if specification, ok := parent.(*ast.ValueSpec); ok {
		for _, name := range specification.Names {
			if name == identifier {
				return
			}
		}
	}
	scan.derived.violations = append(scan.derived.violations, scan.name+": constructor alias of "+identifier.Name)
}

// visitSelector refuses axerror.New in non-call position.
func (scan *fileScan) visitSelector(selector *ast.SelectorExpr, parent ast.Node) {
	identifier, ok := selector.X.(*ast.Ident)
	if !ok || identifier.Name != "axerror" || selector.Sel.Name != "New" {
		return
	}
	if call, ok := parent.(*ast.CallExpr); ok {
		if fun, ok := call.Fun.(*ast.SelectorExpr); ok && fun == selector {
			return
		}
	}
	scan.derived.violations = append(scan.derived.violations, scan.name+": axerror.New in non-call position")
}

// visitComposite derives &frameFault{...} literals by their
// literal detail, except the environ-delegation bridge in
// decodeStrictObject (detail: fault.Detail), which derives one
// arm per fault detail in environ production source. A composite
// literal that names frameFault through any other shape is a
// violation.
func (scan *fileScan) visitComposite(literal *ast.CompositeLit, parent ast.Node) {
	identifier, ok := literal.Type.(*ast.Ident)
	if !ok || identifier.Name != "frameFault" {
		return
	}
	if _, ok := parent.(*ast.UnaryExpr); !ok {
		scan.derived.violations = append(scan.derived.violations, scan.name+": frameFault literal outside address-of")
		return
	}
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			scan.derived.violations = append(scan.derived.violations, scan.name+": frameFault literal with positional element")
			return
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok || key.Name != "detail" {
			continue
		}
		if literal, ok := pair.Value.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			unquoted, err := strconv.Unquote(literal.Value)
			if err != nil {
				scan.derived.violations = append(scan.derived.violations, scan.name+": frameFault literal with unquotable detail")
				return
			}
			scan.derived.arms["frame|"+unquoted] = true
			return
		}
		if selector, ok := pair.Value.(*ast.SelectorExpr); ok && isEnvironFaultDetail(selector) && scan.name == "decode.go" && scan.callFunction() == "decodeStrictObject" {
			// The environ-delegation bridge forwards environ's
			// whole fault space: one arm per fault detail
			// derived from environ production source, never
			// retyped here.
			for _, detail := range environFaultDetails() {
				scan.derived.arms["frame|"+detail] = true
			}
			return
		}
		scan.derived.violations = append(scan.derived.violations, scan.name+": frameFault literal with non-literal detail")
		return
	}
	scan.derived.violations = append(scan.derived.violations, scan.name+": frameFault literal with no detail")
}

// isEnvironFaultDetail reports whether the detail expression is
// the environ-fault forward (fault.Detail) the decodeStrictObject
// delegation bridge carries. Any other selector is a new shape,
// not the bridge, and stays a violation.
func isEnvironFaultDetail(selector *ast.SelectorExpr) bool {
	identifier, ok := selector.X.(*ast.Ident)
	return ok && identifier.Name == "fault" && selector.Sel.Name == "Detail"
}

// environFaultDetails derives the frame-fault space from environ
// production: every Fault* constant value in environ/decode.go.
// The delegation bridge forwards exactly this space, so the arm
// set tracks environ source rather than a retyped list. A read
// or parse failure panics: a scanner that cannot read the fault
// space must fail loudly, never derive an empty set.
func environFaultDetails() []string {
	path := filepath.Join("..", "environ", "decode.go")
	source, err := os.ReadFile(path)
	if err != nil {
		panic("frame arms: read environ decode.go: " + err.Error())
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		panic("frame arms: parse environ decode.go: " + err.Error())
	}
	var details []string
	for _, declaration := range syntax.Decls {
		node, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range node.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range value.Names {
				if !strings.HasPrefix(name.Name, "Fault") || index >= len(value.Values) {
					continue
				}
				literal, ok := value.Values[index].(*ast.BasicLit)
				if !ok {
					panic("frame arms: environ " + name.Name + " is not a literal")
				}
				detail, err := strconv.Unquote(literal.Value)
				if err != nil {
					panic("frame arms: environ " + name.Name + ": " + err.Error())
				}
				details = append(details, detail)
			}
		}
	}
	if len(details) == 0 {
		panic("frame arms: derived zero environ fault details")
	}
	sort.Strings(details)
	return details
}

// defensiveArms names derived arms no public input can fire:
// encoder-produced values that cannot fail encoding. Each stays
// in production as fail-closed handling of a theoretically
// fallible call, and each is recorded here rather than witnessed
// by a refusal test that cannot exist. A new defensive arm must
// extend this map with its rationale; an arm missing from both
// this map and the witness table fails
// TestDerivedRefusalArmsAreAllWitnessed.
var defensiveArms = map[string]string{
	"ctor|failViolation|request frame is not encodable":        "json.Marshal over validated strings, integers, and raw JSON cannot fail; the arm stays so a future member type cannot fail silently",
	"ctor|failIntegrity|idempotency journal is not exportable": "Canonicalize over encoder-produced bytes cannot fail short of memory corruption; the arm stays so export never returns a partial journal",
}

// armWitnesses maps every derived arm to the test functions that
// drive it through a production entry with a negative vector
// asserting code plus distinguishing detail. Shared arms list
// every branch test: the context-prefixed envelope arms fire on
// both the success and failure paths, so both refusal tests are
// named. The table is verified in both directions against the
// derivation; behavioural proof stays with the named tests,
// which run as part of the suite.
var armWitnesses = map[string][]string{
	"ctor|failInvalid|request major is outside the locally supported registry":                              {"TestEncodeRequestRefusals"},
	"ctor|failInvalid|request identifier is not a UUIDv7":                                                   {"TestEncodeRequestRefusals"},
	"ctor|failInvalid|request deadline is outside uint53[1..3600000]":                                       {"TestEncodeRequestRefusals"},
	"ctor|failInvalid|probe request major is outside the locally supported registry":                        {"TestProbeRequestRefusals"},
	"ctor|failInvalid|observed executable digest is not a digest":                                           {"TestCheckManifestBindings"},
	"ctor|failInvalid|observed provider manifest digest is not a digest":                                    {"TestCheckManifestBindings"},
	"ctor|failInvalid|observed adapter manifest digest is not a digest":                                     {"TestCheckManifestBindings"},
	"ctor|failInvalid|idempotency operation identifier is not a UUIDv7":                                     {"TestJournalScopesKeysByOperation"},
	"ctor|failInvalid|bootstrap attempt does not carry operation manifest":                                  {"TestBootstrapRefusesNonManifestAttempt"},
	"ctor|failInvalid|bootstrap process token is empty":                                                     {"TestProcessGuardRefusesReuse"},
	"ctor|failInvalid|bootstrap process was already used for an attempt":                                    {"TestProcessGuardRefusesReuse"},
	"ctor|failViolation|request frame is empty":                                                             {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request frame exceeds the 8 MiB bound":                                              {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request envelope carries unknown member":                                            {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request envelope misses a required member":                                          {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request schema is not the directory node request":                                   {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request protocol version is not a string":                                           {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request schema version is not a string":                                             {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request protocol is not the directory node":                                         {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request identifier is not a string":                                                 {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request identifier is not a UUIDv7":                                                 {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request operation is not a string":                                                  {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request versions do not bind one supported major":                                   {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|request deadline is not uint53[1..3600000]":                                         {"TestDecodeRequestFrameRefusals"},
	"ctor|failViolation|success envelope does not carry ok=true":                                            {"TestCheckSuccessEnvelopeRefusals"},
	"ctor|failViolation|success envelope carries both body and error":                                       {"TestCheckSuccessEnvelopeRefusals"},
	"ctor|failViolation|success envelope carries neither body nor error":                                    {"TestCheckSuccessEnvelopeRefusals"},
	"ctor|failViolation|failure envelope does not carry ok=false":                                           {"TestCheckFailureEnvelopeRefusals"},
	"ctor|failViolation|failure envelope carries both body and error":                                       {"TestCheckFailureEnvelopeRefusals"},
	"ctor|failViolation|failure envelope carries neither body nor error":                                    {"TestCheckFailureEnvelopeRefusals"},
	"ctor|failViolation|node manifest carries unknown member":                                               {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest misses a required member":                                             {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest schema is not the directory node manifest":                            {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest version is not 1.0.0":                                                 {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest node identifier is not a string[1..128]":                              {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest node version is not SemVer":                                           {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest host identifier is not a UUIDv7":                                      {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest executable binding is not a digest":                                   {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest provider binding is not a digest":                                     {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest adapter binding is not a digest":                                      {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest supported versions are not sorted unique SemVer[1..16]":               {"TestDecodeManifestRegistryRules"},
	"ctor|failViolation|node manifest operations are not the complete sorted eleven-name registry":          {"TestDecodeManifestRegistryRules"},
	"ctor|failViolation|node manifest schemas are not sorted unique ContractAssertion[15..64]":              {"TestDecodeManifestRegistryRules"},
	"ctor|failViolation|node manifest tuple registry binding is not a digest":                               {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest capabilities are not the exact eight-name result map":                 {"TestDecodeManifestClosedMemberRules", "TestManifestCapabilityReasonCoherence"},
	"ctor|failViolation|node manifest redaction policies are not sorted unique digest[1..64]":               {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest enrichment profiles are not sorted unique digest[0..256]":             {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest limits are not the closed DirectoryNodeLimits":                        {"TestManifestLimitsBounds"},
	"ctor|failViolation|node manifest extensions are not reverse-DNS keyed":                                 {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|node manifest is not canonical JSON":                                                {"TestDecodeManifestClosedMemberRules"},
	"ctor|failViolation|probe request carries unknown member":                                               {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request misses a required member":                                             {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request platform is outside the negotiated major vocabulary":                  {"TestProbeRequestMajorVocabularies"},
	"ctor|failViolation|probe request architecture is outside amd64|arm64":                                  {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request environment identifiers are not sorted unique strings[0..64]":         {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request environment identifier is not an environment-id":                      {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request capabilities are not sorted unique strings[0..8]":                     {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request capability is outside the directory registry":                         {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe request extensions are not reverse-DNS keyed":                                 {"TestProbeRequestRefusals"},
	"ctor|failViolation|probe response carries unknown member":                                              {"TestProbeResponseRefusals"},
	"ctor|failViolation|probe response misses a required member":                                            {"TestProbeResponseRefusals"},
	"ctor|failViolation|probe response host identifier is not a UUIDv7":                                     {"TestProbeResponseRefusals"},
	"ctor|failViolation|probe response node build is not the closed DirectoryNodeBuild":                     {"TestProbeResponseRefusals"},
	"ctor|failViolation|probe response policy digest is not a digest":                                       {"TestProbeResponseRefusals"},
	"ctor|failViolation|probe response environments are not EnvironmentObservation[0..256] by shape":        {"TestProbeResponseRefusals"},
	"ctor|failViolation|probe response findings are not AdapterFinding[0..4096]":                            {"TestProbeResponseRefusals", "TestProbeResponseFindingBound"},
	"ctor|failViolation|probe response extensions are not reverse-DNS keyed":                                {"TestProbeResponseRefusals"},
	"ctor|failViolation|scan request carries unknown member":                                                {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request misses a required member":                                              {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request operation identifier is not a UUIDv7":                                  {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request installations are not sorted unique digest[1..256]":                    {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request prior batch is not a digest|null":                                      {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request cursor is not a string[1..4096]|null":                                  {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request max_instances is not uint53[1..65536]":                                 {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan request extensions are not reverse-DNS keyed":                                  {"TestCheckScanRequestRefusals"},
	"ctor|failViolation|scan response carries unknown member":                                               {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|scan response misses a required member":                                             {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|scan response batch is not an object":                                               {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|scan response environment observations are not sorted unique digest[1..256]":        {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|scan response native observations are not sorted unique digest[0..65536]":           {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|scan response next cursor is not a string[1..4096]|null":                            {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|scan response extensions are not reverse-DNS keyed":                                 {"TestCheckScanResponseRefusals"},
	"ctor|failViolation|idempotency body is not canonical JSON":                                             {"TestJournalScopesKeysByOperation"},
	"ctor|failViolation|bootstrap manifest does not contain the attempted version":                          {"TestBootstrapTerminalFailuresNeverDowngrade"},
	"ctor|failViolation|bootstrap success carries a nonmatching exit status":                                {"TestBootstrapTerminalFailuresNeverDowngrade"},
	"ctor|failViolation|bootstrap failure is not the exact downgrade tuple":                                 {"TestBootstrapRefusesNonExactDowngradeTuple"},
	"ctor|failViolation|bootstrap downgrade response carries a nonmatching exit status":                     {"TestBootstrapRefusesNonExactDowngradeTuple"},
	"ctor|failUnknownOperation|dispatch names an operation outside the closed registry":                     {"TestRefuseUnknownOperation"},
	"ctor|failUnknownOperation|request names an operation outside the closed registry":                      {"TestDecodeRequestFrameRefusals"},
	"ctor|failUnknownOperation|idempotency key names an operation outside the closed registry":              {"TestJournalScopesKeysByOperation"},
	"ctor|failTransport|bootstrap attempt produced no response frame":                                       {"TestBootstrapTerminalFailuresNeverDowngrade"},
	"ctor|failDowngrade|no lower locally supported major remains":                                           {"TestBootstrapEndsWithoutCommonMajor"},
	"ctor|failIntegrity|failure error is not a Structured Error 1.2.0":                                      {"TestCheckFailureEnvelopeRefusals"},
	"ctor|failIntegrity|node manifest executable binding contradicts the observed executable":               {"TestCheckManifestBindings"},
	"ctor|failIntegrity|node manifest provider binding contradicts the observed manifest":                   {"TestCheckManifestBindings"},
	"ctor|failIntegrity|node manifest adapter binding contradicts the observed manifest":                    {"TestCheckManifestBindings"},
	"ctor|failIntegrity|probe node build node identifier differs from the manifest":                         {"TestNodeBuildEqualityRefusesDrift"},
	"ctor|failIntegrity|probe node build node version differs from the manifest":                            {"TestNodeBuildEqualityRefusesDrift"},
	"ctor|failIntegrity|probe node build executable binding differs from the manifest":                      {"TestNodeBuildEqualityRefusesDrift"},
	"ctor|failIntegrity|probe node build provider binding differs from the manifest":                        {"TestNodeBuildEqualityRefusesDrift"},
	"ctor|failIntegrity|probe node build adapter binding differs from the manifest":                         {"TestNodeBuildEqualityRefusesDrift"},
	"ctor|failIdempotency|idempotent body changed under a recorded operation identifier":                    {"TestJournalRefusesChangedBody", "TestJournalSurvivesRestart"},
	"ctor|failQuery|directory query carries unknown member":                                                 {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query misses a required member":                                               {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query schema is not the session directory query":                              {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query version is not 1.0.0":                                                   {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query identifier is not a UUIDv7":                                             {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query operations are not QueryOperation[1..64]":                               {"TestDecodeQueryIndexRules"},
	"ctor|failQuery|directory query caller is not a closed CallerContext":                                   {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query extensions are not reverse-DNS keyed":                                   {"TestDecodeQueryEnvelopeRules"},
	"ctor|failQuery|directory query operation is not a closed QueryOperation":                               {"TestDecodeQueryIndexRules", "TestDecodeQueryRegistryRules", "TestDecodeQueryMutationFlagRules", "TestDecodeQueryPaginationRules"},
	"ctor|failQuery|directory cursor is not a string[1..1024]":                                              {"TestCheckCursorReuse"},
	"ctor|failQuery|directory cursor is bound to a changed query":                                           {"TestCheckCursorReuse"},
	`ctor|failViolation|expr|EncodeRequest|"request body "+fault.detail`:                                    {"TestEncodeRequestRefusals"},
	`ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`:                           {"TestDecodeRequestFrameRefusals", "TestDecodeRequestFrameStrictFaults"},
	`ctor|failViolation|expr|DecodeRequestFrame|"request body "+fault.detail`:                               {"TestDecodeRequestFrameRefusals"},
	`ctor|failViolation|expr|CheckSuccessEnvelope|"success body "+fault.detail`:                             {"TestCheckSuccessEnvelopeRefusals"},
	`ctor|failViolation|expr|DecodeManifest|"node manifest "+fault.detail`:                                  {"TestDecodeManifestClosedMemberRules"},
	`ctor|failViolation|expr|CheckProbeRequest|"probe request "+fault.detail`:                               {"TestProbeRequestRefusals"},
	`ctor|failViolation|expr|CheckProbeResponse|"probe response "+fault.detail`:                             {"TestProbeResponseRefusals"},
	`ctor|failViolation|expr|CheckScanRequest|"scan request "+fault.detail`:                                 {"TestCheckScanRequestRefusals"},
	`ctor|failViolation|expr|CheckScanResponse|"scan response "+fault.detail`:                               {"TestCheckScanResponseRefusals"},
	`ctor|failIntegrity|expr|Import|"idempotency journal "+fault.detail`:                                    {"TestJournalSurvivesRestart"},
	"ctor|failIntegrity|idempotency journal schema is not the journal export":                               {"TestJournalSurvivesRestart"},
	"ctor|failIntegrity|idempotency journal version is not 1":                                               {"TestJournalSurvivesRestart"},
	"ctor|failIntegrity|idempotency journal records are not an array":                                       {"TestJournalSurvivesRestart"},
	"ctor|failIntegrity|idempotency journal record is not a keyed digest binding":                           {"TestJournalSurvivesRestart", "TestJournalImportRefusesUnsortedRecords"},
	"ctor|failIntegrity|idempotency journal records are not sorted unique":                                  {"TestJournalImportRefusesUnsortedRecords"},
	`ctor|failQuery|expr|DecodeQuery|"directory query "+fault.detail`:                                       {"TestDecodeQueryEnvelopeRules"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" frame is empty"`:                               {"TestResponseIdentitySharedArms"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" frame exceeds the 8 MiB bound"`:                {"TestResponseIdentitySharedArms"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" envelope "+fault.detail`:                       {"TestResponseIdentitySharedArms"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" envelope carries unknown member"`:              {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" envelope misses a required member"`:            {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" schema is not the directory node response"`:    {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" schema version is not the bound 1.0.0"`:        {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" protocol does not echo the request"`:           {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" protocol version does not echo the request"`:   {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" request identifier does not echo the request"`: {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	`ctor|failViolation|expr|checkResponseIdentity|context+" operation does not echo the request"`:          {"TestCheckSuccessEnvelopeRefusals", "TestCheckFailureEnvelopeRefusals"},
	"frame|not valid UTF-8":                {"TestDecodeRequestFrameStrictFaults"},
	"frame|lone surrogate escape":          {"TestDecodeRequestFrameStrictFaults"},
	"frame|not a JSON object":              {"TestDecodeRequestFrameStrictFaults", "TestDecodeRequestFrameRefusals"},
	"frame|duplicate member":               {"TestDecodeRequestFrameRefusals"},
	"frame|trailing data after the object": {"TestDecodeRequestFrameStrictFaults"},
}

// TestRefusalConstructorsMatchProduction requires the registered
// constructor set to equal the derived one and every axerror.New
// site to sit inside a registered body with no violations: the
// denominator is derived from production, not from the test
// file. Any violation fails the census instead of passing
// silently.
func TestRefusalConstructorsMatchProduction(t *testing.T) {
	derived := deriveRefusalArms(t)
	if len(derived.violations) > 0 {
		sort.Strings(derived.violations)
		t.Fatalf("census violations:\n  %s", strings.Join(derived.violations, "\n  "))
	}
	var got []string
	for name := range derived.ctors {
		got = append(got, name)
	}
	sort.Strings(got)
	want := append([]string(nil), refusalConstructors...)
	sort.Strings(want)
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("derived constructors = %v, want registered %v", got, want)
	}
}

// TestCensusScansEveryProductionFile requires the scanned set to
// equal the package's production files: a production file the
// census never opens is a file whose arms never derive.
func TestCensusScansEveryProductionFile(t *testing.T) {
	files := productionFiles(t)
	names := map[string]bool{
		"bootstrap.go": true,
		"decode.go":    true,
		"doc.go":       true,
		"manifest.go":  true,
		"probe.go":     true,
		"protocol.go":  true,
		"query.go":     true,
		"scan.go":      true,
	}
	if len(files) != len(names) {
		t.Fatalf("scanned %d files %v, want %d named files", len(files), files, len(names))
	}
	for _, path := range files {
		if !names[filepath.Base(path)] {
			t.Fatalf("unexpected production file %q: register it here or the census ignores it", path)
		}
	}
}

// TestDerivedRefusalArmsAreAllWitnessed requires every derived
// arm to carry a witness or a recorded defensive rationale: an
// arm nobody drives is next round's finding, stated here instead.
func TestDerivedRefusalArmsAreAllWitnessed(t *testing.T) {
	derived := deriveRefusalArms(t)
	if len(derived.violations) > 0 {
		sort.Strings(derived.violations)
		t.Fatalf("census violations:\n  %s", strings.Join(derived.violations, "\n  "))
	}
	var missing []string
	for arm := range derived.arms {
		if _, witnessed := armWitnesses[arm]; witnessed {
			continue
		}
		if _, defensive := defensiveArms[arm]; defensive {
			continue
		}
		missing = append(missing, arm)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("unwitnessed arms:\n  %s", strings.Join(missing, "\n  "))
	}
	t.Logf("census: %d arms witnessed, %d defensive, %d constructors", len(armWitnesses), len(defensiveArms), len(derived.ctors))
}

// TestWitnessedArmsAreAllDerived requires every witness row to
// name a derived arm: a truncated derivation reddens on the
// orphaned witnesses instead of passing vacuously.
func TestWitnessedArmsAreAllDerived(t *testing.T) {
	derived := deriveRefusalArms(t)
	if len(derived.violations) > 0 {
		sort.Strings(derived.violations)
		t.Fatalf("census violations:\n  %s", strings.Join(derived.violations, "\n  "))
	}
	var orphaned []string
	for arm := range armWitnesses {
		if !derived.arms[arm] {
			orphaned = append(orphaned, arm)
		}
	}
	for arm := range defensiveArms {
		if !derived.arms[arm] {
			orphaned = append(orphaned, arm+" (defensive)")
		}
	}
	sort.Strings(orphaned)
	if len(orphaned) > 0 {
		t.Fatalf("orphaned witness rows:\n  %s", strings.Join(orphaned, "\n  "))
	}
}

// TestWitnessesResolve requires every named witness to exist as
// a test function in this package's test files: a witness that
// names nothing proves nothing.
func TestWitnessesResolve(t *testing.T) {
	defined := map[string]bool{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("witness resolve: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("witness resolve: %v", err)
		}
		syntax, err := parseGoFile(name, source)
		if err != nil {
			t.Fatalf("witness resolve: %v", err)
		}
		for _, declaration := range syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			defined[function.Name.Name] = true
		}
	}
	var missing []string
	seen := map[string]bool{}
	for _, witnesses := range armWitnesses {
		for _, witness := range witnesses {
			if seen[witness] {
				continue
			}
			seen[witness] = true
			if !defined[witness] {
				missing = append(missing, witness)
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("unresolved witnesses:\n  %s", strings.Join(missing, "\n  "))
	}
}

// TestConstructorAliasSpellingsFailDerivation proves the alias
// shapes against synthetic setters: a package-level var alias, a
// function-local := alias, and a constructor passed as an
// argument all fail the derivation, while the definition Name
// itself stays exempt. The vectors are synthetic on purpose:
// they prove the gate, not production.
func TestConstructorAliasSpellingsFailDerivation(t *testing.T) {
	scan := &fileScan{derived: &armDerivation{arms: map[string]bool{}, ctors: map[string]bool{}}, name: "synthetic.go"}
	_ = scan
	for _, probe := range []struct {
		name   string
		source string
		want   int
	}{
		{"definition name exempt", "package probe\nvar failInvalid = func(detail string, field string) {}\n", 0},
		{"package var alias fails", "package probe\nvar alias = failInvalid\n", 1},
		{"local alias fails", "package probe\nfunc f() { alias := failInvalid; _ = alias }\n", 1},
		{"argument alias fails", "package probe\nfunc f() { g(failInvalid) }\nfunc g(v any) {}\n", 1},
		{"return alias fails", "package probe\nfunc f() any { return failInvalid }\n", 1},
		{"assignment alias fails", "package probe\nfunc f() { var v any; v = failInvalid; _ = v }\n", 1},
	} {
		t.Run(probe.name, func(t *testing.T) {
			syntax, err := parseGoFile("synthetic.go", []byte(probe.source))
			if err != nil {
				t.Fatalf("parse synthetic source: %v", err)
			}
			derived := &armDerivation{arms: map[string]bool{}, ctors: map[string]bool{}}
			inner := &fileScan{derived: derived, name: "synthetic.go"}
			var stack []ast.Node
			ast.Inspect(syntax, func(node ast.Node) bool {
				if node == nil {
					stack = stack[:len(stack)-1]
					return true
				}
				var parent ast.Node
				if len(stack) > 0 {
					parent = stack[len(stack)-1]
				}
				inner.visit(node, parent, 0)
				stack = append(stack, node)
				return true
			})
			if len(derived.violations) != probe.want {
				t.Fatalf("violations = %v, want %d", derived.violations, probe.want)
			}
		})
	}
}
