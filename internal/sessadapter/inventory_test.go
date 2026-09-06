package sessadapter

import (
	"fmt"
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
// Derivation (deriveRefusalArms): every call to one of the seven
// refusal constructors with a string-literal first argument
// yields ctor|<constructor>|<detail>; a call whose first
// argument is a literal prefix plus fault.detail yields
// ctor|<constructor>|conduit|<function>|<prefix>, so conduits in
// different functions never merge; every &frameFault{...}
// literal yields frame|<detail>, except the environ-delegation
// bridge, which yields one frame arm per environ fault detail.
// A constructor call with any
// other first-argument shape lands as
// ctor|<constructor>|expr|<function> rather than passing
// silently. Constructor references outside direct-call position
// fail the derivation outright: an aliased constructor hides
// every arm it builds. All three alias spellings are caught —
// `alias := failInvalid`, `var alias = failInvalid` at package
// level, and `var alias = failInvalid` inside a function body —
// because the gate exempts only the definition's own Name in its
// ValueSpec, never a constructor identifier appearing as a
// ValueSpec value. The constructors of this package are
// themselves declared as `var failX = func...`, so their
// definition Names must stay exempt while any other var-held
// reference fails.
// Package-level var initializers scan under their variable name,
// so a refusal arm hiding in a package-level func literal is
// derived, not missed.
//
// The constructor denominator is itself derived, not listed on
// trust: TestRefusalConstructorsMatchProduction requires the
// registered set to equal the package-level vars whose bodies
// call axerror.New, every axerror.New site in production to sit
// inside one registered body, and no func-value indirection of
// axerror.New anywhere. An eighth constructor, an inline
// axerror.New refusal, or a `var newFault = axerror.New`
// indirection carries an additive arm the derivation never
// observes, and fails there instead of passing silently.
//
// Both directions are checked:
// TestDerivedRefusalArmsAreAllWitnessed (every derived arm
// carries a witness) and TestWitnessedArmsAreAllDerived (every
// witness names a derived arm, so a truncated derivation reddens
// on the orphaned witnesses instead of passing vacuously).
// TestEveryArmWitnessRefusesAtTheProductionEntry drives each
// witness through the production entry and requires the refusal
// to come from the arm under test: code, distinguishing detail,
// and rule text. Empty, single-file, or unparseable derivations
// fail closed: a domain that silently derives nothing is not a
// measurement.
//
// Stated bounds: the five timestamp pairs share one
// (constructor, detail) identity across their checkTimestamp and
// Time halves by construction (one rule, two halves); deleting
// one half is a behavioral mutant the entry-point tests must
// catch, and every witness below proves both halves. Arms
// sharing one identity merge to one obligation only there and
// nowhere else: any other merge fails the derivation. The
// open-class member gates (unknown-member and duplicate-member
// refusals) range over an unbounded name class, so one witness
// per arm proves the gate fires, not that every name is gated:
// a mutant exempting a different name than the witnessed one is
// outside what the arm inventory narrows, and is recorded here
// rather than proven.
//
// Shape space (review rev6 G-B; the brief asked what shape comes
// after the one just closed, so the answer is written here). An
// error construction in this package can only be carried by
// these AST shapes; the census derives or refuses each one:
//
//   1. A direct `failX(...)` call — DERIVED as an arm keyed by
//      the literal detail (or conduit/expr shape).
//   2. A package-level `var failY = func...` literal whose body
//      calls axerror.New — DERIVED as a constructor: the
//      produced set must equal refusalConstructors exactly, so
//      an eighth constructor fails as unregistered.
//   3. An axerror.New call outside every registered literal
//      (inline in a function, in a function-local literal even
//      under a registered name) — REFUSED as a construction
//      site outside the registered bodies.
//   4. An axerror.New selector in non-call position (alias var,
//      returned constructor, argument, struct field) — REFUSED
//      as a func-value indirection (the import census pins the
//      `axerror` binding it matches on).
//   5. A failX identifier in non-call, non-definition position
//      (alias var/:=, assignment, argument, return, struct
//      field, method value) — REFUSED as a constructor alias
//      that would bury every arm built through it (proven per
//      position by TestConstructorAliasSpellingsFailDerivation).
//   6. An import of the error package path under any other name
//      — REFUSED by the import census
//      (TestAxerrorImportsAreUnaliased pins the `axerror`
//      binding every identifier match above relies on).
//   7. A &frameFault{...} literal — DERIVED as a frame arm by
//      its type name and literal detail, except the
//      environ-delegation bridge (detail: fault.Detail), which
//      derives one arm per fault detail in environ production
//      source (proven by the bridge row in the derivation, not
//      by retyping the space here).
//
// 7 of 7 shapes derived-or-refused, each with a synthetic proof.
// The residue, stated here rather than left silent: a frameFault
// built through a type alias (`type ff = frameFault`) would hide
// from shape 7 — no such alias exists in production; a
// construction via assembly, linkname, or reflect would hide
// from shapes 3-4 — the package uses none of those mechanisms;
// files outside the package directory and _test.go files, which
// never ship, are not scanned.

// refusalArmCensusFloor is the derived arm count this test was
// written against. It is a tripwire, not an enumeration: a
// derivation returning fewer arms fails closed even when every
// surviving arm is witnessed, which is what catches a silently
// truncated scan. Raising it is routine when production gains a
// refusal; lowering it requires saying why.
const refusalArmCensusFloor = 323

// refusalConstructors is the closed constructor set the
// derivation observes, pinned against production by
// TestRefusalConstructorsMatchProduction: the registered names
// must equal the package-level error-constructing vars exactly,
// and every axerror.New site in production must sit inside one
// registered constructor body. A refusal built any other way —
// through an unregistered eighth constructor or an inline
// axerror.New — fails that census; a refusal built through an
// aliased constructor fails the alias gate below.
var refusalConstructors = []string{
	"failInvalid",
	"failProtocol",
	"failUnknownOperation",
	"failCapability",
	"failUnavailable",
	"failUnsupportedTuple",
	"failIntegrity",
}

func deriveRefusalArms(t *testing.T) map[string]struct{} {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("derive refusal arms: %v", err)
	}
	arms, scanned, err := refusalArmsIn(directory)
	if err != nil {
		t.Fatalf("derive refusal arms: %v", err)
	}
	if len(scanned) == 0 {
		t.Fatal("derived refusal arms from zero production files; the scanner is broken, not the package")
	}
	if len(arms) == 0 {
		t.Fatal("derived zero refusal arms from the package sources; the scanner is broken, not the package")
	}
	if len(arms) < refusalArmCensusFloor {
		t.Fatalf("derived %d refusal arms, below the %d census floor; the derivation is short, not the package", len(arms), refusalArmCensusFloor)
	}
	t.Logf("refusal arm coverage domain: %d derived arms across %d production files", len(arms), len(scanned))
	return arms
}

func refusalArmsIn(directory string) (map[string]struct{}, []string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}
	arms := map[string]struct{}{}
	var scanned []string
	constructors := map[string]bool{}
	for _, name := range refusalConstructors {
		constructors[name] = true
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned = append(scanned, name)
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			return nil, nil, err
		}
		if err := collectFileArms(fileSet, syntax, constructors, arms); err != nil {
			return nil, nil, err
		}
	}
	return arms, scanned, nil
}

// TestRefusalConstructorsMatchProduction pins the constructor
// denominator against production source in both directions: the
// set of package-level func-valued vars whose body constructs
// an error with axerror.New must equal refusalConstructors
// exactly, and every axerror.New call in production must sit
// inside one registered constructor body. A func-value
// indirection of the constructor (`var newFault = axerror.New`)
// is refused outright: the reference names the constructor
// without calling it, so a refusal built through the alias
// hides from both the census and the arm derivation. The arm
// inventory observes only registered constructors, so an eighth
// constructor with an additive arm, an inline axerror.New
// refusal with an additive arm, or an indirection with an
// additive arm all survive the full suite — and all fail here,
// the first as an unregistered constructor, the second as a
// construction site outside every registered body, the third as
// an indirection. axerror.Decode is a read, not a construction,
// and is ignored. An empty scan fails closed.
func TestRefusalConstructorsMatchProduction(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("constructor census: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("constructor census: %v", err)
	}
	registered := map[string]bool{}
	for _, name := range refusalConstructors {
		registered[name] = true
	}
	produced := map[string]string{}
	scanned := 0
	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("constructor census: %v", err)
		}
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			t.Fatalf("constructor census: %v", err)
		}
		for _, violation := range constructorSitesInFile(fileSet, syntax, registered, produced, name) {
			violations = append(violations, name+": "+violation)
		}
	}
	if scanned == 0 {
		t.Fatal("constructor census scanned zero production files; the scanner is broken, not the package")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("error-construction site(s) outside the registered constructors:\n  %s", strings.Join(violations, "\n  "))
	}
	var unregistered []string
	for name, file := range produced {
		if !registered[name] {
			unregistered = append(unregistered, name+" ("+file+")")
		}
	}
	var orphaned []string
	for name := range registered {
		if _, ok := produced[name]; !ok {
			orphaned = append(orphaned, name)
		}
	}
	sort.Strings(unregistered)
	sort.Strings(orphaned)
	if len(unregistered) > 0 {
		t.Fatalf("production error constructor(s) with no census registration:\n  %s", strings.Join(unregistered, "\n  "))
	}
	if len(orphaned) > 0 {
		t.Fatalf("census registration(s) naming no production constructor:\n  %s", strings.Join(orphaned, "\n  "))
	}
	t.Logf("constructor census: %d constructors across %d files", len(produced), scanned)
}

// TestConstructorSitesReportFuncValueIndirection proves the
// indirection case of the constructor census against synthetic
// files: a package-level `var newFault = axerror.New` reports
// even with no refusal built through it yet, while a New call
// inside the registered definition and a Decode read stay
// exempt. The vectors are synthetic on purpose: they prove the
// gate, not production.
func TestConstructorSitesReportFuncValueIndirection(t *testing.T) {
	registered := map[string]bool{"failInvalid": true}
	parse := func(source string) (*token.FileSet, *ast.File) {
		t.Helper()
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return fileSet, syntax
	}
	t.Run("package-level indirection reports", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nvar newFault = axerror.New\nfunc f() { _, _ = newFault(nil) }\n")
		produced := map[string]string{}
		violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go")
		if len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
		if !strings.Contains(violations[0], "indirection") {
			t.Fatalf("violation = %q, want the indirection sentence", violations[0])
		}
	})
	t.Run("function-local indirection reports", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f() { newFault := axerror.New\n_ = newFault }\n")
		produced := map[string]string{}
		if violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go"); len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
	})
	t.Run("returned constructor reports", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f() func() { return axerror.New }\n")
		produced := map[string]string{}
		violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go")
		if len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
		if !strings.Contains(violations[0], "indirection") {
			t.Fatalf("violation = %q, want the indirection sentence", violations[0])
		}
	})
	t.Run("call inside the definition stays exempt", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nvar failInvalid = func() { axerror.New(nil) }\n")
		produced := map[string]string{}
		if violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go"); len(violations) != 0 {
			t.Fatalf("violations = %v, want none for the definition spelling", violations)
		}
	})
	t.Run("decode read stays exempt", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f() { _, _ = axerror.Decode(nil) }\n")
		produced := map[string]string{}
		if violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go"); len(violations) != 0 {
			t.Fatalf("violations = %v, want none for a Decode read", violations)
		}
	})
}

// TestConstructorSitesReportFunctionLocalRegisteredName proves
// the function-local case of the constructor census against a
// synthetic file: a New call inside a function-local literal
// reports even when the variable carries a registered name,
// while the package-level definition spelling stays exempt. The
// vector is synthetic on purpose: it proves the gate, not
// production.
func TestConstructorSitesReportFunctionLocalRegisteredName(t *testing.T) {
	registered := map[string]bool{"failInvalid": true}
	parse := func(source string) (*token.FileSet, *ast.File) {
		t.Helper()
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return fileSet, syntax
	}
	t.Run("function-local registered name reports", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f() { var failInvalid = func() { axerror.New(nil) }\n_ = failInvalid }\n")
		produced := map[string]string{}
		violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go")
		if len(violations) != 1 {
			t.Fatalf("violations = %v, want one function-local report", violations)
		}
		if _, ok := produced["failInvalid"]; ok {
			t.Fatal("function-local literal recorded a constructor; only package-level scopes produce")
		}
	})
	t.Run("package-level definition stays exempt", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nvar failInvalid = func() { axerror.New(nil) }\n")
		produced := map[string]string{}
		if violations := constructorSitesInFile(fileSet, syntax, registered, produced, "probe.go"); len(violations) != 0 {
			t.Fatalf("violations = %v, want none for the definition spelling", violations)
		}
		if produced["failInvalid"] != "probe.go" {
			t.Fatalf("produced = %v, want the definition recorded", produced)
		}
	})
}

// constructorVarScope is one var-held func literal an
// axerror.New call can attribute to: the holding variable name
// with its source range, and whether the declaration is
// package-level. Only a package-level scope holding a
// registered name is a constructor; anything else enclosing a
// New call is an unregistered construction site.
type constructorVarScope struct {
	name       string
	start, end token.Pos
	topLevel   bool
}

// constructorSitesInFile records every package-level func-valued
// var whose body calls axerror.New into produced, and reports
// every axerror.New call outside a registered constructor body:
// outside any var-held literal, inside one whose variable is not
// registered, or inside a function-local literal even when the
// variable carries a registered name. Function-local var-held
// literals are never constructors, so a New call inside one
// always reports. A func-value indirection (`var newFault =
// axerror.New`) is reported too: the reference names the
// constructor without calling it, so every refusal built through
// the alias would hide from both this census and the arm
// derivation at once. axerror.Decode is a read, not a
// construction, and is ignored.
func constructorSitesInFile(fileSet *token.FileSet, syntax *ast.File, registered map[string]bool, produced map[string]string, filename string) []string {
	var scopes []constructorVarScope
	var calls []token.Pos
	var indirections []token.Pos
	var stack []ast.Node
	funcDepth := 0
	ast.Inspect(syntax, func(node ast.Node) bool {
		if node == nil {
			popped := stack[len(stack)-1]
			switch popped.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				funcDepth--
			}
			stack = stack[:len(stack)-1]
			return true
		}
		if selector, ok := node.(*ast.SelectorExpr); ok && isAxerrorNew(selector) {
			if parent, ok := parentNode(stack).(*ast.CallExpr); !ok || parent.Fun != selector {
				indirections = append(indirections, node.Pos())
			}
		}
		switch node := node.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			funcDepth++
		case *ast.GenDecl:
			if node.Tok != token.VAR {
				break
			}
			topLevel := funcDepth == 0
			for _, specification := range node.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, ident := range value.Names {
					literal := pairedFuncLit(value, index)
					if literal == nil {
						continue
					}
					scopes = append(scopes, constructorVarScope{name: ident.Name, start: literal.Pos(), end: literal.End(), topLevel: topLevel})
				}
			}
		case *ast.CallExpr:
			selector, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok || receiver.Name != "axerror" || selector.Sel.Name != "New" {
				break
			}
			calls = append(calls, node.Pos())
		}
		stack = append(stack, node)
		return true
	})
	var violations []string
	for _, position := range indirections {
		violations = append(violations, fmt.Sprintf("func-value indirection of axerror.New at %s; refusals built through it hide from the constructor census", fileSet.Position(position)))
	}
	for _, position := range calls {
		best := ""
		var bestStart token.Pos
		initialized := false
		topLevel := false
		for _, scope := range scopes {
			if scope.start <= position && position <= scope.end {
				if !initialized || scope.start >= bestStart {
					best, bestStart, initialized, topLevel = scope.name, scope.start, true, scope.topLevel
				}
			}
		}
		switch {
		case !initialized:
			violations = append(violations, fmt.Sprintf("axerror.New outside any constructor at %s", fileSet.Position(position)))
		case !registered[best]:
			violations = append(violations, fmt.Sprintf("axerror.New inside unregistered %q at %s", best, fileSet.Position(position)))
		case !topLevel:
			violations = append(violations, fmt.Sprintf("axerror.New inside function-local %q at %s; only package-level constructors are registered", best, fileSet.Position(position)))
		default:
			produced[best] = filename
		}
	}
	return violations
}

// isAxerrorNew reports whether the selector names the error
// constructor: axerror.New. Any other selector — axerror.Decode,
// a version constant — is not a construction.
func isAxerrorNew(selector *ast.SelectorExpr) bool {
	receiver, ok := selector.X.(*ast.Ident)
	return ok && receiver.Name == "axerror" && selector.Sel.Name == "New"
}

// parentNode returns the innermost enclosing node on the walk
// stack, or nil at the file root.
func parentNode(stack []ast.Node) ast.Node {
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

// pairedFuncLit returns the func literal holding one ValueSpec
// name: the index-paired value when every name carries its own,
// the shared value when one serves all names, and nil for any
// other shape or a non-literal value.
func pairedFuncLit(value *ast.ValueSpec, index int) *ast.FuncLit {
	var candidate ast.Expr
	switch {
	case len(value.Values) == len(value.Names) && index < len(value.Values):
		candidate = value.Values[index]
	case len(value.Values) == 1:
		candidate = value.Values[0]
	default:
		return nil
	}
	literal, ok := candidate.(*ast.FuncLit)
	if !ok {
		return nil
	}
	return literal
}

// scopeRange is one named scope constructor calls attribute to:
// a function declaration or a package-level variable holding a
// func literal.
type scopeRange struct {
	name       string
	start, end token.Pos
}

// collectFileArms derives every arm in one production file from a
// single parse, so scope containment compares positions within
// one file set: constructor calls attribute to their innermost
// enclosing scope by source range, frameFault literals derive
// frame arms, and any constructor identifier outside call-fun
// and definition position is an alias and fails the file.
func collectFileArms(fileSet *token.FileSet, syntax *ast.File, constructors map[string]bool, arms map[string]struct{}) error {
	var scopes []scopeRange
	type pendingCall struct {
		name     string
		call     *ast.CallExpr
		position token.Pos
	}
	var calls []pendingCall
	var frames []*ast.CompositeLit
	type identUse struct {
		identifier *ast.Ident
		parent     ast.Node
	}
	var uses []identUse
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
		switch node := node.(type) {
		case *ast.FuncDecl:
			scopes = append(scopes, scopeRange{name: node.Name.Name, start: node.Pos(), end: node.End()})
		case *ast.GenDecl:
			for _, specification := range node.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range value.Names {
					if index < len(value.Values) {
						if literal, ok := value.Values[index].(*ast.FuncLit); ok {
							scopes = append(scopes, scopeRange{name: "var:" + name.Name, start: literal.Pos(), end: literal.End()})
						}
					}
				}
			}
		case *ast.Ident:
			if constructors[node.Name] {
				uses = append(uses, identUse{identifier: node, parent: parent})
			}
		case *ast.CallExpr:
			if fun, ok := node.Fun.(*ast.Ident); ok && constructors[fun.Name] {
				calls = append(calls, pendingCall{name: fun.Name, call: node, position: node.Pos()})
			}
		case *ast.CompositeLit:
			if identifier, ok := node.Type.(*ast.Ident); ok && identifier.Name == "frameFault" {
				frames = append(frames, node)
			}
		}
		stack = append(stack, node)
		return true
	})
	for _, use := range uses {
		if call, ok := use.parent.(*ast.CallExpr); ok {
			if fun, ok := call.Fun.(*ast.Ident); ok && fun == use.identifier {
				continue
			}
		}
		if specification, ok := use.parent.(*ast.ValueSpec); ok {
			defined := false
			for _, name := range specification.Names {
				if name == use.identifier {
					defined = true
					break
				}
			}
			if defined {
				continue
			}
			return fmt.Errorf("constructor alias of %s buries every arm it builds at %s", use.identifier.Name, fileSet.Position(use.identifier.Pos()))
		}
		return fmt.Errorf("constructor alias or indirect use of %s at %s", use.identifier.Name, fileSet.Position(use.identifier.Pos()))
	}
	enclosing := func(position token.Pos) string {
		best := "package"
		var bestStart token.Pos
		initialized := false
		for _, scope := range scopes {
			if scope.start <= position && position <= scope.end {
				if !initialized || scope.start >= bestStart {
					best, bestStart, initialized = scope.name, scope.start, true
				}
			}
		}
		return best
	}
	for _, call := range calls {
		arm, err := callArm(call.name, call.call, enclosing(call.position))
		if err != nil {
			return err
		}
		arms[arm] = struct{}{}
	}
	for _, frame := range frames {
		details, err := frameDetails(frame)
		if err != nil {
			return err
		}
		for _, detail := range details {
			arms["frame|"+detail] = struct{}{}
		}
	}
	return nil
}

// callArm derives one constructor arm: a string-literal first
// argument is the detail; a literal-plus-fault.detail binary is a
// conduit keyed by function and prefix; anything else is an
// expr obligation keyed by function.
func callArm(constructor string, call *ast.CallExpr, function string) (string, error) {
	if len(call.Args) == 0 {
		return "", fmt.Errorf("constructor %s called with no arguments in %s", constructor, function)
	}
	switch first := call.Args[0].(type) {
	case *ast.BasicLit:
		if first.Kind != token.STRING {
			return "", fmt.Errorf("constructor %s first argument is not a string in %s", constructor, function)
		}
		detail, err := strconv.Unquote(first.Value)
		if err != nil {
			return "", err
		}
		return "ctor|" + constructor + "|" + detail, nil
	case *ast.BinaryExpr:
		prefix, ok := first.X.(*ast.BasicLit)
		if !ok || prefix.Kind != token.STRING {
			return "ctor|" + constructor + "|expr|" + function, nil
		}
		text, err := strconv.Unquote(prefix.Value)
		if err != nil {
			return "", err
		}
		return "ctor|" + constructor + "|conduit|" + function + "|" + text, nil
	default:
		return "ctor|" + constructor + "|expr|" + function, nil
	}
}

// frameDetails derives the frame arms of one &frameFault
// composite literal. A literal detail yields its arm. The
// environ-delegation bridge in decodeStrictObject (detail:
// fault.Detail, where fault is the environ fault) forwards
// environ's whole fault space, so it derives one arm per fault
// detail derived from environ production source — never retyped
// here, so a renamed, added, or removed fault reddens the
// derivation instead of passing against a stale copy. Any other
// non-literal detail is an expr obligation, never a silent pass.
func frameDetails(frame *ast.CompositeLit) ([]string, error) {
	for _, element := range frame.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok || key.Name != "detail" {
			continue
		}
		if literal, ok := pair.Value.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			detail, err := strconv.Unquote(literal.Value)
			if err != nil {
				return nil, err
			}
			return []string{detail}, nil
		}
		if selector, ok := pair.Value.(*ast.SelectorExpr); ok {
			if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "fault" && selector.Sel.Name == "Detail" {
				return environFaultDetails(), nil
			}
		}
		return []string{"expr"}, nil
	}
	return nil, fmt.Errorf("frameFault literal carries no detail")
}

// environFaultDetails derives the frame-fault space from environ
// production: every Fault* constant value in environ/decode.go.
// The delegation bridge forwards exactly this space, so the arm
// set tracks environ source rather than a retyped list.
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

// TestConstructorAliasSpellingsFailDerivation replays the four
// alias probes against the derivation gate on synthetic sources:
// a direct constructor call derives cleanly (the control), while
// a package-level `var alias = ctor`, a function-local
// `var alias = ctor`, and an `alias := ctor` all fail the file.
// The package's own `var failX = func...` definition spelling
// stays exempt: only the definition Name is excused, never a
// constructor in value position.
func TestConstructorAliasSpellingsFailDerivation(t *testing.T) {
	constructors := map[string]bool{"failInvalid": true}
	derive := func(source string) error {
		t.Helper()
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return collectFileArms(fileSet, syntax, constructors, map[string]struct{}{})
	}
	t.Run("direct call derives", func(t *testing.T) {
		if err := derive("package probe\nfunc f() { failInvalid(\"planted\", \"field\") }\n"); err != nil {
			t.Fatalf("direct call failed the derivation: %v", err)
		}
	})
	t.Run("definition name stays exempt", func(t *testing.T) {
		if err := derive("package probe\nvar failInvalid = func(detail, field string) (int, error) { return 0, nil }\n"); err != nil {
			t.Fatalf("constructor definition failed the derivation: %v", err)
		}
	})
	for _, probe := range []struct {
		name   string
		source string
	}{
		{"package level var alias", "package probe\nvar plantedFail = failInvalid\nfunc f() { plantedFail(\"planted\", \"field\") }\n"},
		{"function local var alias", "package probe\nfunc f() { var localFail = failInvalid\n_ = localFail }\n"},
		{"short declaration alias", "package probe\nfunc f() { localFail := failInvalid\n_ = localFail }\n"},
		{"returned constructor", "package probe\nfunc f() func(string, string) (int, error) { return failInvalid }\n"},
		{"constructor as argument", "package probe\nfunc g(h func(string, string) (int, error)) {}\nfunc f() { g(failInvalid) }\n"},
		{"constructor in struct field", "package probe\ntype holder struct { build func(string, string) (int, error) }\nfunc f() { h := holder{}\nh.build = failInvalid\n_ = h }\n"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			if err := derive(probe.source); err == nil {
				t.Fatal("aliased constructor passed the derivation with every arm it builds hidden")
			}
		})
	}
}

// TestDerivedRefusalArmsAreAllWitnessed is the forward direction:
// every arm derived from production must carry a witness. A
// planted arm (a new literal or site through an existing
// constructor) lands here as an unwitnessed arm.
func TestDerivedRefusalArmsAreAllWitnessed(t *testing.T) {
	derived := deriveRefusalArms(t)
	witnessed := map[string]int{}
	for _, witness := range declaredWitnesses(t) {
		witnessed[witness.arm]++
	}
	var missing []string
	for arm := range derived {
		if witnessed[arm] == 0 {
			missing = append(missing, arm)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("derived refusal arm(s) with no witness at the production entry:\n  %s", strings.Join(missing, "\n  "))
	}
	t.Logf("refusal arm coverage: %d/%d derived arms witnessed", len(derived)-len(missing), len(derived))
}

// TestWitnessedArmsAreAllDerived is the reverse direction: every
// witness must name an arm production declares. A deleted or
// narrowed production branch orphans its witness and fails here,
// which is also what makes a truncated derivation fail instead
// of passing vacuously.
func TestWitnessedArmsAreAllDerived(t *testing.T) {
	derived := deriveRefusalArms(t)
	var orphans []string
	for _, witness := range declaredWitnesses(t) {
		if _, ok := derived[witness.arm]; !ok {
			orphans = append(orphans, witness.arm+" ("+witness.name+")")
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Fatalf("witness(es) naming no derived production arm:\n  %s", strings.Join(orphans, "\n  "))
	}
}

// TestEveryArmWitnessRefusesAtTheProductionEntry drives every
// witness through its production entry point and requires the
// attributed refusal.
func TestEveryArmWitnessRefusesAtTheProductionEntry(t *testing.T) {
	for _, witness := range declaredWitnesses(t) {
		t.Run(witness.arm+"/"+witness.name, func(t *testing.T) {
			witness.prove(t)
		})
	}
}

// armWitness is one derived arm plus the test proving it refuses
// at the production entry with the attributed identity.
type armWitness struct {
	arm   string
	name  string
	prove func(t *testing.T)
}

// declaredWitnesses lists every witness. The group tables live
// in inventory_witnesses_test.go.
func declaredWitnesses(t *testing.T) []armWitness {
	t.Helper()
	e := witnessFixtureEntries(t)
	witnesses := []armWitness{}
	witnesses = append(witnesses, envelopeWitnesses(e)...)
	witnesses = append(witnesses, manifestWitnesses(e)...)
	witnesses = append(witnesses, probeWitnesses(e)...)
	witnesses = append(witnesses, contextWitnesses(e)...)
	witnesses = append(witnesses, tupleWitnesses(e)...)
	witnesses = append(witnesses, operationWitnesses(e)...)
	witnesses = append(witnesses, gateWitnesses(e)...)
	witnesses = append(witnesses, frameWitnesses(e)...)
	witnesses = append(witnesses, conduitWitnesses(e)...)
	return witnesses
}
