package secprim

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// exercisedRefusalSites records the production file:line that constructed a
// refusal, keyed with the rule it carried, for every instrumented refusal
// constructor call made during the test run.
var exercisedRefusalSites sync.Map // file:line -> rule

func recordRefusalSite(rule string) {
	if _, file, line, ok := runtime.Caller(2); ok {
		exercisedRefusalSites.Store(fmt.Sprintf("%s:%d", filepath.Base(file), line), rule)
	}
}

func TestMain(main *testing.M) {
	origPath, origArgv, origEnv := failPath, failArgv, failEnv
	failPath = func(rule, detail string) *Error {
		err := origPath(rule, detail)
		recordRefusalSite("path:" + rule)
		return err
	}
	failArgv = func(rule, detail string) *Error {
		err := origArgv(rule, detail)
		recordRefusalSite("argv:" + rule)
		return err
	}
	failEnv = func(rule, detail string) *Error {
		err := origEnv(rule, detail)
		recordRefusalSite("env:" + rule)
		return err
	}
	code := main.Run()
	if code == 0 && fullPackageTestRun() {
		if failures := auditRefusalInventory(); len(failures) != 0 {
			for _, failure := range failures {
				fmt.Fprintln(os.Stderr, failure)
			}
			code = 1
		}
	}
	os.Exit(code)
}

func fullPackageTestRun() bool {
	selected := flag.Lookup("test.run")
	return selected == nil || selected.Value.String() == ""
}

// defensiveSites names derived refusal sites no public input can fire,
// with the bound that keeps each honest. The member-containment conjunct
// in Resolve is unreachable because the member grammar refuses every
// parent segment before the prefix check runs; it stays as defense in
// depth against a future grammar relaxation, and
// TestGuardPrefixCheckIsLoadBearing proves the containment property over
// a generated corpus, so a weakened grammar reddens there first. A
// defensive entry that resolves to no derived site fails as stale.
var defensiveSites = map[string]string{
	"failPath:member containment": "unreachable while the member grammar refuses parent segments; pinned by TestGuardPrefixCheckIsLoadBearing",
}

// platformExempt names rules that need not fire on one GOOS, keyed
// file|constructor:rule. The key is rule-level, not site-level: every
// "open failed" site in nofollow.go shares it. The post-check directory
// open fails deterministically only where mode bits are enforceable: the
// chmod vector in TestOpenNoFollowDirRefusesUnreadableDir is skipped on
// Windows (ACLs do not express a mode-000 directory) and as root (which
// bypasses mode bits), so on Windows the rule is exempt. An exemption
// that resolves to no derived site fails as stale.
var platformExempt = map[string]string{
	"nofollow.go|failPath:open failed": "windows",
}

// rootExempt names derived sites excused when the suite runs as root,
// which bypasses the mode bits the chmod vector depends on. Same key
// shape and staleness discipline as platformExempt; the audit below
// checks both tables for orphans.
var rootExempt = map[string]bool{
	"nofollow.go|failPath:open failed": true,
}

// siteKey renders the platform-exemption identity of a derived site: the
// file basename plus the constructor and rule. Line numbers are left out
// so edits above or below a site do not rename it.
func siteKey(site, constructor string) string {
	file := site
	if index := strings.LastIndex(site, ":"); index >= 0 {
		file = site[:index]
	}
	return file + "|" + constructor
}

// exemptionResolves reports whether the exemption key matches at least
// one derived site.
func exemptionResolves(derived map[string]string, key string) bool {
	for derivedSite, constructor := range derived {
		if siteKey(derivedSite, constructor) == key {
			return true
		}
	}
	return false
}

// ctorNames is the closed refusal-constructor set. errors.go states the
// rule these names enforce: production builds Error values only through
// them, never through an alias, a literal, new(Error), or a wrapper
// factory.
var ctorNames = map[string]bool{"failPath": true, "failArgv": true, "failEnv": true}

// refusalDerivation is the parsed shape of one source set: the call
// sites that must each fire during the suite, and the construction
// shapes outside the constructors that must not exist at all.
type refusalDerivation struct {
	sites   map[string]string // file:line -> "constructor:rule" (aliases resolved)
	strays  []string          // file:line plus the offending shape, sorted
	scanned int
}

// unwrapParens strips parenthesized expressions so ((failPath)) and
// (failPath) resolve like the bare spelling instead of bypassing it.
func unwrapParens(expr ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = parenthesized.X
	}
}

// returnsError reports whether the result list carries a *Error: the
// factory shape the gate enumerates. WithCause is the one legitimate
// member and is excluded by its callers, not here.
func returnsError(results *ast.FieldList) bool {
	if results == nil {
		return false
	}
	for _, field := range results.List {
		star, ok := unwrapParens(field.Type).(*ast.StarExpr)
		if !ok {
			continue
		}
		if identifier, ok := unwrapParens(star.X).(*ast.Ident); ok && identifier.Name == "Error" {
			return true
		}
	}
	return false
}

// isErrorLiteral reports whether the composite literal builds an
// secprim Error: Error{...} in-package, or secprim.Error{...} from
// outside it.
func isErrorLiteral(literal *ast.CompositeLit) bool {
	switch typed := unwrapParens(literal.Type).(type) {
	case *ast.Ident:
		return typed.Name == "Error"
	case *ast.SelectorExpr:
		return typed.Sel.Name == "Error"
	}
	return false
}

// deriveRefusals parses every source and derives the refusal inventory
// in two phases. Phase one resolves the constructor-alias fixpoint
// (var x = failPath, x := failArgv, alias-of-alias), the *Error
// factory declarations, and the constructor body spans. Phase two
// walks every call, literal, binding, and name use against them:
//
//   - a call to a constructor or its alias is a site (the rule is the
//     literal first argument, labelled under the underlying
//     constructor so exemptions keep resolving);
//   - a binding of a constructor to a new name is an alias stray, even
//     when never called;
//   - an Error literal, a new(Error) call, or a call to a package
//     *Error factory (other than the WithCause decorator, which
//     refines an already-counted refusal) is a construction stray;
//   - a factory declaration whose body never calls a constructor is a
//     manufacturing stray, even when never called;
//   - any other use of a constructor name as a value is an escape
//     stray.
//
// The enumerated shape space is: bare call, parenthesized call, var
// alias, local alias, alias-of-alias, direct literal, new(Error),
// method factory call, function factory call, uncalled factory, and
// value escape. TestRefusalInventoryClosesBypassShapes plants every
// member.
//
// Stated bound: every shape above is keyed on the Error identifier
// spelling (isErrorLiteral matches Ident "Error", returnsError matches
// StarExpr of Ident "Error"). A refusal type spelled through a type
// alias (type X = Error), a defined type (type X Error), or a factory
// result spelling the type through such an alias is invisible to this
// derivation — neither a literal stray, nor a factory, nor an escape —
// by construction, not by oversight. The gate therefore closes the
// enumerated spellings, not the class; resolving spellings would need
// type information (go/types), which this syntax-only derivation
// deliberately does not load. TestRefusalInventorySpellingBoundIsStated
// pins the bound: the alias spellings must stay silent here, so a
// future spelling cannot re-claim the class without touching that test.
func deriveRefusals(sources map[string][]byte) (refusalDerivation, error) {
	derivation := refusalDerivation{sites: map[string]string{}}
	fileset := token.NewFileSet()
	type parsed struct {
		name   string
		syntax *ast.File
	}
	var files []parsed
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		syntax, err := parser.ParseFile(fileset, name, sources[name], 0)
		if err != nil {
			return derivation, err
		}
		files = append(files, parsed{name: name, syntax: syntax})
	}
	derivation.scanned = len(files)
	position := func(name string, pos token.Pos) string {
		return fmt.Sprintf("%s:%d", filepath.Base(name), fileset.Position(pos).Line)
	}
	insideCtor := func(syntax *ast.File, pos token.Pos) bool {
		inside := false
		ast.Inspect(syntax, func(node ast.Node) bool {
			general, ok := node.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range general.Specs {
				values, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, identifier := range values.Names {
					if !ctorNames[identifier.Name] || index >= len(values.Values) {
						continue
					}
					if literal, ok := values.Values[index].(*ast.FuncLit); ok {
						if pos >= literal.Pos() && pos <= literal.End() {
							inside = true
							return false
						}
					}
				}
			}
			return true
		})
		return inside
	}
	// Phase one: bindings of constructor values to new names.
	type binding struct {
		name, file, target string
		pos                token.Pos
	}
	var bindings []binding
	noteBinding := func(file string, lhs, rhs ast.Expr) {
		identifier, ok := lhs.(*ast.Ident)
		if !ok || identifier.Name == "_" {
			return
		}
		if target, ok := unwrapParens(rhs).(*ast.Ident); ok {
			bindings = append(bindings, binding{name: identifier.Name, file: file, target: target.Name, pos: identifier.Pos()})
		}
	}
	for _, file := range files {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.ValueSpec:
				for index, identifier := range typed.Names {
					if index < len(typed.Values) {
						noteBinding(file.name, identifier, typed.Values[index])
					}
				}
			case *ast.AssignStmt:
				if len(typed.Lhs) == len(typed.Rhs) {
					for index := range typed.Lhs {
						noteBinding(file.name, typed.Lhs[index], typed.Rhs[index])
					}
				}
			}
			return true
		})
	}
	aliases := map[string]string{}
	for changed := true; changed; {
		changed = false
		for _, candidate := range bindings {
			if _, known := aliases[candidate.name]; known {
				continue
			}
			if ctorNames[candidate.target] {
				aliases[candidate.name] = candidate.target
				changed = true
			} else if underlying, ok := aliases[candidate.target]; ok {
				aliases[candidate.name] = underlying
				changed = true
			}
		}
	}
	underlying := func(name string) (string, bool) {
		if ctorNames[name] {
			return name, true
		}
		target, ok := aliases[name]
		return target, ok
	}
	// Phase one: *Error factory declarations (functions and methods).
	factories := map[string]bool{}
	for _, file := range files {
		for _, declaration := range file.syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name == "WithCause" {
				continue
			}
			if returnsError(function.Type.Results) {
				factories[function.Name.Name] = true
			}
		}
	}
	// Phase two: sites, strays, and the positions that excuse a name
	// use (call-fun position and declaration position).
	callFuns := map[token.Pos]bool{}
	declared := map[token.Pos]bool{}
	for _, file := range files {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CallExpr:
				if identifier, ok := unwrapParens(typed.Fun).(*ast.Ident); ok {
					callFuns[identifier.Pos()] = true
				}
			case *ast.ValueSpec:
				for _, identifier := range typed.Names {
					declared[identifier.Pos()] = true
				}
			case *ast.AssignStmt:
				for _, bound := range typed.Lhs {
					if identifier, ok := bound.(*ast.Ident); ok {
						declared[identifier.Pos()] = true
					}
				}
			case *ast.Field:
				for _, identifier := range typed.Names {
					declared[identifier.Pos()] = true
				}
			case *ast.FuncDecl:
				declared[typed.Name.Pos()] = true
			case *ast.TypeSpec:
				declared[typed.Name.Pos()] = true
			}
			return true
		})
	}
	stray := func(file string, pos token.Pos, shape string) {
		derivation.strays = append(derivation.strays, position(file, pos)+" "+shape)
	}
	for _, file := range files {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := unwrapParens(call.Fun).(type) {
			case *ast.Ident:
				if constructor, ok := underlying(fun.Name); ok {
					// Every constructor call passes its rule as a
					// string literal first argument; a non-literal
					// rule is outside the roster shape and fails
					// the derivation.
					site := position(file.name, call.Pos())
					if len(call.Args) == 0 {
						derivation.sites[site] = ""
						return true
					}
					literal, ok := unwrapParens(call.Args[0]).(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						derivation.sites[site] = ""
						return true
					}
					rule, err := strconv.Unquote(literal.Value)
					if err != nil {
						derivation.sites[site] = ""
						return true
					}
					derivation.sites[site] = constructor + ":" + rule
					return true
				}
				if fun.Name == "new" && len(call.Args) > 0 {
					if identifier, ok := unwrapParens(call.Args[0]).(*ast.Ident); ok && identifier.Name == "Error" {
						stray(file.name, call.Pos(), "new(Error) manufactures a refusal outside the constructors")
					}
					return true
				}
				if factories[fun.Name] {
					stray(file.name, call.Pos(), "factory call "+fun.Name+" manufactures a refusal outside the constructors")
				}
			case *ast.SelectorExpr:
				if fun.Sel.Name == "WithCause" {
					return true
				}
				if factories[fun.Sel.Name] {
					stray(file.name, call.Pos(), "factory call "+fun.Sel.Name+" manufactures a refusal outside the constructors")
				}
			}
			return true
		})
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if isErrorLiteral(literal) && !insideCtor(file.syntax, literal.Pos()) {
				stray(file.name, literal.Pos(), "Error literal manufactures a refusal outside the constructors")
			}
			return true
		})
	}
	for _, candidate := range bindings {
		if _, ok := underlying(candidate.target); ok {
			if ctorNames[candidate.name] {
				continue
			}
			stray(candidate.file, candidate.pos, "alias "+candidate.name+" rebinds a refusal constructor")
		}
	}
	// Factory declarations whose body never reaches a constructor
	// manufacture refusals no inventory site can name.
	for _, file := range files {
		for _, declaration := range file.syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name == "WithCause" || function.Body == nil {
				continue
			}
			if !factories[function.Name.Name] {
				continue
			}
			reaches := false
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if identifier, ok := unwrapParens(call.Fun).(*ast.Ident); ok {
					if _, ok := underlying(identifier.Name); ok {
						reaches = true
						return false
					}
				}
				return true
			})
			if !reaches {
				stray(file.name, function.Pos(), "factory "+function.Name.Name+" manufactures *Error without the constructors")
			}
		}
	}
	// Constructor names used as values anywhere but call-fun and
	// declaration position escape the call-site inventory.
	for _, file := range files {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			if _, ok := underlying(identifier.Name); !ok {
				return true
			}
			if callFuns[identifier.Pos()] || declared[identifier.Pos()] {
				return true
			}
			stray(file.name, identifier.Pos(), "constructor "+identifier.Name+" escapes as a value outside a call")
			return true
		})
	}
	sort.Strings(derivation.strays)
	return derivation, nil
}

// auditRefusalInventory derives every failPath/failArgv/failEnv call site
// from non-test production source and requires each to have fired during
// the suite (forward), and every fired site to resolve to a derived site
// (reverse). Constructions outside the constructors (aliases, literals,
// factories, escapes) fail outright. A constructor call no test reaches,
// or a recorded site the source no longer contains, fails instead of
// passing silently. An empty derivation fails closed: a blind scanner is
// not a complete inventory.
func auditRefusalInventory() []string {
	directory, err := os.Getwd()
	if err != nil {
		return []string{fmt.Sprintf("derive secprim refusal inventory: %v", err)}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return []string{fmt.Sprintf("derive secprim refusal inventory: %v", err)}
	}
	sources := map[string][]byte{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return []string{fmt.Sprintf("derive secprim refusal inventory: %v", err)}
		}
		sources[name] = contents
	}
	derivation, err := deriveRefusals(sources)
	if err != nil {
		return []string{fmt.Sprintf("derive secprim refusal inventory: %v", err)}
	}
	derived := derivation.sites
	var failures []string
	if derivation.scanned == 0 {
		failures = append(failures, "scanned no production sources; the check is blind")
	}
	if len(derived) == 0 {
		failures = append(failures, "derived no refusal sites; the scan is blind")
	}
	if len(derivation.strays) != 0 {
		failures = append(failures, "refusal construction outside the constructors:\n  "+strings.Join(derivation.strays, "\n  "))
	}
	for site, defense := range defensiveSites {
		_ = defense
		matched := false
		for derivedSite, constructor := range derived {
			if constructor == site {
				matched = true
				_ = derivedSite
			}
		}
		if !matched {
			failures = append(failures, fmt.Sprintf("defensive site %s resolves to no derived site; the rationale is stale", site))
		}
	}
	for key := range platformExempt {
		if !exemptionResolves(derived, key) {
			failures = append(failures, fmt.Sprintf("platform exemption %s resolves to no derived site; the exemption is stale", key))
		}
	}
	for key := range rootExempt {
		if !exemptionResolves(derived, key) {
			failures = append(failures, fmt.Sprintf("root exemption %s resolves to no derived site; the exemption is stale", key))
		}
	}
	for site, constructor := range derived {
		if constructor == "" {
			failures = append(failures, fmt.Sprintf("refusal site %s carries a non-literal rule; the roster cannot name it", site))
			continue
		}
		if _, defensive := defensiveSites[constructor]; defensive {
			continue
		}
		if only, exempt := platformExempt[siteKey(site, constructor)]; exempt && runtime.GOOS == only {
			continue
		}
		if _, excused := rootExempt[siteKey(site, constructor)]; excused && privilegedUser() {
			continue
		}
		if _, exercised := exercisedRefusalSites.Load(site); !exercised {
			failures = append(failures, fmt.Sprintf("refusal site %s never fired during the suite", site))
		}
	}
	exercisedRefusalSites.Range(func(key, _ any) bool {
		if _, ok := derived[key.(string)]; !ok {
			failures = append(failures, fmt.Sprintf("exercised refusal site %s resolves to no derived site", key.(string)))
		}
		return true
	})
	return failures
}

// TestRefusalInventoryClosesBypassShapes plants every member of the
// enumerated (Error-identifier-spelled) refusal-construction shape
// space through the same deriveRefusals the audit runs on the real
// tree, and requires each plant to be caught. The control proves the
// instrument works (an unexercised bare call is an unfired site);
// plants A-D are the four bypasses that walked through the old
// identifier-keyed gate, and the remaining three close the rest of
// the enumerated space. Spellings outside the Error identifier are
// not members of this space: they are pinned silent by
// TestRefusalInventorySpellingBoundIsStated instead.
func TestRefusalInventoryClosesBypassShapes(t *testing.T) {
	t.Parallel()
	plants := map[string][]byte{
		// CONTROL: a bare constructor call no test fires.
		"plant_control.go": []byte("package secprim\n\nfunc plantControl() error {\n\treturn failArgv(\"plant control\", \"nowhere\")\n}\n"),
		// A: package-level var alias, then a call through it.
		"plant_a.go": []byte("package secprim\n\nvar failArgvAlias = failArgv\n\nfunc plantA() error {\n\treturn failArgvAlias(\"plant a\", \"nowhere\")\n}\n"),
		// B: a direct Error literal, the shape errors.go forbids.
		"plant_b.go": []byte("package secprim\n\nfunc plantB() error {\n\treturn &Error{Kind: ErrUnsafeArgv, Rule: \"plant b\", Detail: \"nowhere\"}\n}\n"),
		// C: a method factory plus a selector call through it.
		"plant_c.go": []byte("package secprim\n\ntype plantMaker struct{}\n\nfunc (plantMaker) argv(rule, detail string) *Error {\n\treturn &Error{Kind: ErrUnsafeArgv, Rule: rule, Detail: detail}\n}\n\nfunc plantC() error {\n\treturn plantMaker{}.argv(\"plant c\", \"nowhere\")\n}\n"),
		// D: a function-local binding, then a call through it.
		"plant_d.go": []byte("package secprim\n\nfunc plantD() error {\n\tmake := failArgv\n\treturn make(\"plant d\", \"nowhere\")\n}\n"),
		// E: the constructor escaping as a value, never called.
		"plant_e.go": []byte("package secprim\n\nfunc takeArgv(make func(string, string) *Error) func(string, string) *Error {\n\treturn make\n}\n\nvar plantEscape = takeArgv(failArgv)\n"),
		// F: new(Error) manufacturing a zero refusal.
		"plant_f.go": []byte("package secprim\n\nfunc plantF() error {\n\tfailure := new(Error)\n\tfailure.Kind = ErrUnsafeArgv\n\treturn failure\n}\n"),
		// G: an alias of an alias, then a call through it.
		"plant_g.go": []byte("package secprim\n\nvar failArgvAlias = failArgv\n\nvar failArgvAlias2 = failArgvAlias\n\nfunc plantG() error {\n\treturn failArgvAlias2(\"plant g\", \"nowhere\")\n}\n"),
	}
	// Each plant runs through the derivation alone (plus nothing):
	// plants must not see each other's declarations, the way a real
	// bypass would sit beside honest production code that must stay
	// clean.
	derive := func(t *testing.T, name string) refusalDerivation {
		t.Helper()
		derivation, err := deriveRefusals(map[string][]byte{name: plants[name]})
		if err != nil {
			t.Fatalf("derive %s: %v", name, err)
		}
		return derivation
	}
	contains := func(t *testing.T, name string, strays []string, word string) {
		t.Helper()
		for _, stray := range strays {
			if strings.Contains(stray, word) {
				return
			}
		}
		t.Fatalf("%s: no stray containing %q in %v", name, word, strays)
	}
	t.Run("control is an unfired site", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_control.go")
		if len(derivation.strays) != 0 {
			t.Fatalf("control strays %v; the honest shape must stay clean", derivation.strays)
		}
		if len(derivation.sites) != 1 {
			t.Fatalf("control sites = %v, want the one bare call", derivation.sites)
		}
	})
	t.Run("var alias", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_a.go")
		contains(t, "plant A", derivation.strays, "alias failArgvAlias")
	})
	t.Run("direct literal", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_b.go")
		contains(t, "plant B", derivation.strays, "Error literal")
	})
	t.Run("method factory", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_c.go")
		contains(t, "plant C decl", derivation.strays, "factory argv")
		contains(t, "plant C call", derivation.strays, "factory call argv")
		contains(t, "plant C literal", derivation.strays, "Error literal")
	})
	t.Run("local binding", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_d.go")
		contains(t, "plant D", derivation.strays, "alias make")
	})
	t.Run("value escape", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_e.go")
		contains(t, "plant E", derivation.strays, "escapes as a value")
	})
	t.Run("new Error", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_f.go")
		contains(t, "plant F", derivation.strays, "new(Error)")
	})
	t.Run("alias of alias", func(t *testing.T) {
		t.Parallel()
		derivation := derive(t, "plant_g.go")
		contains(t, "plant G", derivation.strays, "alias failArgvAlias2")
	})
}

// TestRefusalInventorySpellingBoundIsStated pins the identifier-spelling
// bound of the inventory gate: the rev2 reviewer plants H, I, and J
// manufacture refusals through a type alias, a defined type, and an
// alias-spelled factory result, and the syntax-only derivation stays
// silent about all three — no site, no stray. That silence is the
// stated bound, not a passing instrument: the gate closes the
// Error-identifier spellings enumerated above, not the class, and any
// future change that starts catching (or claiming) these spellings
// must update this test, errors.go, and the README together.
func TestRefusalInventorySpellingBoundIsStated(t *testing.T) {
	t.Parallel()
	plants := map[string][]byte{
		// H: a type alias for Error, then a literal through it.
		"plant_h.go": []byte("package secprim\n\ntype shadowAlias = Error\n\nfunc plantH() error {\n\treturn &shadowAlias{Kind: ErrUnsafeArgv, Rule: \"plant h\", Detail: \"nowhere\"}\n}\n"),
		// I: a defined type with Error's shape, converted back to *Error.
		"plant_i.go": []byte("package secprim\n\ntype shadowNamed Error\n\nfunc plantI() error {\n\tinner := shadowNamed{Kind: ErrUnsafeArgv, Rule: \"plant i\", Detail: \"nowhere\"}\n\treturn (*Error)(&inner)\n}\n"),
		// J: a factory whose result spells the type through an alias.
		"plant_j.go": []byte("package secprim\n\ntype shadowResult = Error\n\nfunc plantJFactory(rule, detail string) *shadowResult {\n\treturn &shadowResult{Kind: ErrUnsafeArgv, Rule: rule, Detail: detail}\n}\n"),
	}
	for _, name := range []string{"plant_h.go", "plant_i.go", "plant_j.go"} {
		derivation, err := deriveRefusals(map[string][]byte{name: plants[name]})
		if err != nil {
			t.Fatalf("derive %s: %v", name, err)
		}
		if len(derivation.sites) != 0 {
			t.Fatalf("%s: sites = %v, want silence (outside the witness)", name, derivation.sites)
		}
		if len(derivation.strays) != 0 {
			t.Fatalf("%s: strays = %v, want silence (outside the witness)", name, derivation.strays)
		}
	}
}
