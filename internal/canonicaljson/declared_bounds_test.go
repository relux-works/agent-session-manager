package canonicaljson

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file is the self-proving coverage gate for every declared range bound in
// the package.
//
// The failure it exists to prevent: a bound is reachable and accepts valid
// input, nothing ever proves it rejects anything, and widening it leaves the
// whole configured gate set green. A review sweep found roughly twenty such
// bounds after a hand-picked twelve-mutant sweep reported zero survivors, which
// is what a hand-listed inventory always degrades to one call site later.
//
// The gate therefore derives its own subject. It walks the package production
// sources for every call to a bound helper, derives the helper set transitively
// rather than naming helpers by hand, turns each call site into one obligation
// per declared direction, and requires the obligation set to equal the set of
// proofs that actually execute. Adding a bound call site, widening a bound, or
// deleting a proof all redden the suite.

// boundDirection is one declared side of a range bound.
type boundDirection string

const (
	// boundMaximum obliges an at-maximum acceptance and an over-maximum refusal.
	boundMaximum boundDirection = "max"
	// boundMinimum obliges an at-minimum acceptance and an under-minimum refusal.
	// A declared minimum of zero carries no obligation: there is nothing below it.
	boundMinimum boundDirection = "min"
)

// boundObligation identifies one direction of one derived call site.
type boundObligation struct {
	key       string
	direction boundDirection
}

func (obligation boundObligation) String() string {
	return obligation.key + " [" + string(obligation.direction) + "]"
}

// boundHelper is a package function that refuses a value outside a declared
// range. Helpers are derived, never listed: see deriveBoundHelpers.
type boundHelper struct {
	name string
	// Parameter indexes; -1 when the helper does not take that parameter.
	nameArgument    int
	minimumArgument int
	maximumArgument int
	// fixed helpers wrap an inner helper with literal bounds instead of
	// forwarding minimum/maximum parameters, so their bounds come from the
	// wrapper body rather than from each call site.
	fixed        bool
	fixedMinimum string
	fixedMaximum string
}

// delegatedBoundProofs names the derived obligations whose at-limit acceptance
// cannot be observed through a public entry point, together with the test that
// proves them instead. The set is asserted exactly and each named test must
// exist in the package, so a delegation cannot be invented to silence the gate.
//
// GitIndex entries: a 65,536-entry index is a valid GitIndex but the smallest
// identity candidate carrying one encodes past the 5 MiB public object-size
// gate, so the public entries refuse it on size before the declared bound is
// reached. The named test pins both the declared-bound behaviour at the
// validator and the outer size refusal at the public entries.
var delegatedBoundProofs = map[boundObligation]string{
	{key: "validateGitIndex|requireArray|entries|-..65536", direction: boundMaximum}: "TestGitIndexEntryCountBoundaryIsBoundBelowThePublicObjectSizeGate",
}

// TestEveryDeclaredBoundCallSiteIsProvenInBothDirections is the coverage
// assertion. It fails when a derived obligation has no proof, when a proof
// claims an obligation that no longer exists, and — because the derived key
// carries the literal bound — when any bound is silently widened or narrowed.
func TestEveryDeclaredBoundCallSiteIsProvenInBothDirections(t *testing.T) {
	t.Parallel()

	obligations := deriveBoundObligations(t)
	proven := make(map[boundObligation]int)
	for _, claim := range collectBoundProofClaims(t) {
		proven[claim]++
	}
	for obligation := range delegatedBoundProofs {
		proven[obligation]++
	}

	var missing, extra []string
	for obligation, required := range obligations {
		if proven[obligation] < required {
			missing = append(missing, fmt.Sprintf("%s: %d call site(s), %d proof(s)", obligation, required, proven[obligation]))
		}
	}
	for obligation, count := range proven {
		if required := obligations[obligation]; count > required {
			extra = append(extra, fmt.Sprintf("%s: %d proof(s), %d call site(s)", obligation, count, required))
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("declared bounds with no at-limit/over-limit proof:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("bound proofs claiming obligations that no call site declares:\n  %s", strings.Join(extra, "\n  "))
	}
}

// TestDelegatedBoundProofsNameATestThatExists keeps the delegation escape hatch
// honest: a delegated obligation must point at a test function present in this
// package's sources.
func TestDelegatedBoundProofsNameATestThatExists(t *testing.T) {
	t.Parallel()

	names := packageTestFunctionNames(t)
	for obligation, test := range delegatedBoundProofs {
		if _, ok := names[test]; !ok {
			t.Errorf("delegated bound %s names %s, which does not exist in this package", obligation, test)
		}
	}
}

// deriveBoundObligations returns every derived obligation with the number of
// distinct production call sites that declare it.
func deriveBoundObligations(t *testing.T) map[boundObligation]int {
	t.Helper()

	obligations := make(map[boundObligation]int)
	for _, site := range deriveBoundCallSites(t) {
		if site.maximum != "" {
			obligations[boundObligation{key: site.key, direction: boundMaximum}]++
		}
		if minimum, err := strconv.Atoi(site.minimum); err == nil && minimum >= 1 {
			obligations[boundObligation{key: site.key, direction: boundMinimum}]++
		}
	}
	if len(obligations) == 0 {
		t.Fatal("derived zero declared-bound obligations; the scanner is broken, not the package")
	}
	return obligations
}

type boundCallSite struct {
	key     string
	minimum string
	maximum string
}

// deriveBoundCallSites walks every production source of the package and returns
// one entry per call to a derived bound helper.
//
// The key names the lexical context, the helper, the bounded member, and the
// literal bounds. Context is the enclosing composite-literal string key when the
// call sits in one — that is how the Session Event payload registry declares a
// bound per event type — and the enclosing function name otherwise.
func deriveBoundCallSites(t *testing.T) []boundCallSite {
	t.Helper()

	_, paths := packageProductionFiles(t)
	files := make([]*ast.File, 0, len(paths))
	for _, path := range paths {
		files = append(files, parseProductionFile(t, path))
	}
	constants := derivePackageIntegerConstants(files)
	helpers := deriveBoundHelpers(t, files)

	var sites []boundCallSite
	for _, file := range files {
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, node)
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			helper, ok := helpers[identifier.Name]
			if !ok {
				return true
			}
			context, enclosing := callContext(stack)
			if _, forwarding := helpers[enclosing]; forwarding {
				// A bound helper calling another bound helper forwards its own
				// parameters; the outer helper's call sites carry the bounds.
				return true
			}
			member := literalStringArgument(call, helper.nameArgument)
			minimum, maximum := "", ""
			if helper.fixed {
				minimum, maximum = helper.fixedMinimum, helper.fixedMaximum
			} else {
				if helper.minimumArgument >= 0 {
					minimum = literalIntegerArgument(call, helper.minimumArgument, constants)
				}
				if helper.maximumArgument >= 0 {
					maximum = literalIntegerArgument(call, helper.maximumArgument, constants)
				}
			}
			if member == "" || (helper.minimumArgument >= 0 && minimum == "") || (helper.maximumArgument >= 0 && maximum == "") {
				t.Fatalf(
					"%s: call to %s at %s does not declare a literal member name and bounds, so it cannot be proven at its limit; "+
						"spell the member and bounds literally or extend the derivation",
					identifier.Name, context, callPosition(t, file, call),
				)
			}
			sites = append(sites, boundCallSite{
				key:     fmt.Sprintf("%s|%s|%s|%s..%s", context, identifier.Name, member, orDash(minimum), orDash(maximum)),
				minimum: minimum,
				maximum: maximum,
			})
			return true
		})
	}
	return sites
}

// deriveBoundHelpers derives the bound-helper set from the package instead of
// naming it. A seed helper declares a bounded member through `name string` plus
// an int `minimum` and/or `maximum` parameter. A wrapper helper then declares
// `name string` with no bound parameters and forwards that name to an already
// derived helper with literal bounds; its own bounds are those literals. The
// closure repeats so a wrapper around a wrapper is still derived.
//
// Nothing here is a name list, so a seventh helper — the exact shape that let a
// duplicate closed-member gate carry forty unenumerated member sets — is picked
// up automatically and its call sites become obligations.
func deriveBoundHelpers(t *testing.T, files []*ast.File) map[string]boundHelper {
	t.Helper()

	helpers := make(map[string]boundHelper)
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			helper, seed := seedBoundHelper(function)
			if seed {
				helpers[helper.name] = helper
			}
		}
	}
	for range len(files) + 4 {
		grew := false
		for _, file := range files {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Body == nil {
					continue
				}
				if _, known := helpers[function.Name.Name]; known {
					continue
				}
				helper, wrapper := wrapperBoundHelper(function, helpers)
				if wrapper {
					helpers[helper.name] = helper
					grew = true
				}
			}
		}
		if !grew {
			break
		}
	}
	if len(helpers) == 0 {
		t.Fatal("derived zero bound helpers for the canonicaljson package")
	}
	return helpers
}

func seedBoundHelper(function *ast.FuncDecl) (boundHelper, bool) {
	helper := boundHelper{name: function.Name.Name, nameArgument: -1, minimumArgument: -1, maximumArgument: -1}
	for index, parameter := range flatParameters(function) {
		switch {
		case parameter.name == "name" && parameter.typeName == "string":
			helper.nameArgument = index
		case parameter.name == "minimum" && parameter.typeName == "int":
			helper.minimumArgument = index
		case parameter.name == "maximum" && parameter.typeName == "int":
			helper.maximumArgument = index
		}
	}
	if helper.nameArgument < 0 || (helper.minimumArgument < 0 && helper.maximumArgument < 0) {
		return boundHelper{}, false
	}
	return helper, true
}

func wrapperBoundHelper(function *ast.FuncDecl, helpers map[string]boundHelper) (boundHelper, bool) {
	helper := boundHelper{name: function.Name.Name, nameArgument: -1, minimumArgument: -1, maximumArgument: -1}
	for index, parameter := range flatParameters(function) {
		if parameter.typeName == "int" && (parameter.name == "minimum" || parameter.name == "maximum") {
			return boundHelper{}, false
		}
		if parameter.name == "name" && parameter.typeName == "string" {
			helper.nameArgument = index
		}
	}
	if helper.nameArgument < 0 {
		return boundHelper{}, false
	}
	found := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		inner, ok := helpers[identifier.Name]
		if !ok || inner.nameArgument < 0 || inner.nameArgument >= len(call.Args) {
			return true
		}
		forwarded, ok := call.Args[inner.nameArgument].(*ast.Ident)
		if !ok || forwarded.Name != "name" {
			return true
		}
		minimum, maximum := "", ""
		if inner.minimumArgument >= 0 {
			minimum = literalIntegerArgument(call, inner.minimumArgument, nil)
			if minimum == "" {
				return true
			}
		}
		if inner.maximumArgument >= 0 {
			maximum = literalIntegerArgument(call, inner.maximumArgument, nil)
			if maximum == "" {
				return true
			}
		}
		helper.fixed, helper.fixedMinimum, helper.fixedMaximum = true, minimum, maximum
		found = true
		return false
	})
	return helper, found
}

type flatParameter struct {
	name     string
	typeName string
}

func flatParameters(function *ast.FuncDecl) []flatParameter {
	var parameters []flatParameter
	if function.Type.Params == nil {
		return parameters
	}
	for _, field := range function.Type.Params.List {
		typeName := ""
		if identifier, ok := field.Type.(*ast.Ident); ok {
			typeName = identifier.Name
		}
		if len(field.Names) == 0 {
			parameters = append(parameters, flatParameter{typeName: typeName})
			continue
		}
		for _, name := range field.Names {
			parameters = append(parameters, flatParameter{name: name.Name, typeName: typeName})
		}
	}
	return parameters
}

// derivePackageIntegerConstants resolves package-level untyped integer
// constants so a bound spelled as a named constant is still a literal bound.
func derivePackageIntegerConstants(files []*ast.File) map[string]string {
	constants := make(map[string]string)
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range value.Names {
					if index >= len(value.Values) {
						continue
					}
					literal, ok := value.Values[index].(*ast.BasicLit)
					if !ok || literal.Kind != token.INT {
						continue
					}
					constants[name.Name] = strings.ReplaceAll(literal.Value, "_", "")
				}
			}
		}
	}
	return constants
}

func callContext(stack []ast.Node) (context, enclosing string) {
	for index := len(stack) - 1; index >= 0; index-- {
		if pair, ok := stack[index].(*ast.KeyValueExpr); ok && context == "" {
			if literal, ok := pair.Key.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				if unquoted, err := strconv.Unquote(literal.Value); err == nil {
					context = unquoted
				}
			}
		}
		if function, ok := stack[index].(*ast.FuncDecl); ok {
			enclosing = function.Name.Name
			break
		}
	}
	if context == "" {
		context = enclosing
	}
	return context, enclosing
}

func literalStringArgument(call *ast.CallExpr, index int) string {
	if index < 0 || index >= len(call.Args) {
		return ""
	}
	literal, ok := call.Args[index].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	unquoted, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return unquoted
}

func literalIntegerArgument(call *ast.CallExpr, index int, constants map[string]string) string {
	if index < 0 || index >= len(call.Args) {
		return ""
	}
	switch argument := call.Args[index].(type) {
	case *ast.BasicLit:
		if argument.Kind == token.INT {
			return strings.ReplaceAll(argument.Value, "_", "")
		}
	case *ast.Ident:
		if value, ok := constants[argument.Name]; ok {
			return value
		}
	}
	return ""
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func callPosition(t *testing.T, file *ast.File, call *ast.CallExpr) string {
	t.Helper()
	_ = file
	return fmt.Sprintf("offset %d", call.Pos())
}
