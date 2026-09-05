// AST-derived refusal-arm inventory for the terminal backend packages.
//
// The failure this file exists to prevent: a brand-new production refusal
// arm carrying a brand-new unasserted detail shipped while `gofmt`,
// `go vet`, `go test ./...`, and `tracecheck` all stayed green. A
// hand-maintained arm list cannot catch that; a derived one can, but only
// if it is checked in BOTH directions. The forward direction alone passes
// vacuously on a truncated derivation: every derived arm asserted is true
// when the derivation silently dropped half the package. So this file
// derives every refusal arm from the production AST, requires each to
// resolve to exactly one declared row naming a real asserting test, and
// requires every declared row to resolve to exactly one derived arm.
//
// Modelled on internal/canonicaljson/grammar_inventory_test.go: derivation
// fails closed on an empty, short, or unparseable input, and the declared
// table cannot move its own goalposts because the derivation never
// consults it.
//
// Stated bounds (not inferred):
//   - The inventory proves every arm is DECLARED with a resolving test.
//     Execution and decision proof is each row's named test, measured by
//     this package's suite. A row resolves textually (detail mention, or
//     entry-plus-code/predicate mention); a wrongly attributed row that
//     still resolves is a review finding, not a green light.
//   - Attribution drift survives through that fallback: a
//     dead-by-construction arm can resolve through a sibling site's test
//     while pointing at the wrong guard. Live example: CheckTransition's
//     "operation vocabulary" arm (conformance.go) is unreachable —
//     ParseOperation admits exactly the ten operations transitionTable
//     covers, so lookupTransition never misses — yet the row carries no
//     bound and resolves through TestParseOperationAdmitsOnlyTheTenClosedOperations,
//     which drives ParseOperation's own arm sharing the detail. The five
//     re-parse arms are the contrast: they declare boundDefensiveReparse
//     and TestDefensiveBoundsAreExactlyThese pins the set.
//   - Kill inflation: both inventory directions fail on ANY added or
//     deleted production arm by construction, so an arm-deleting mutant
//     elsewhere reads as killed by the inventory alone with no behavioral
//     change. Mutation scores measured with this file present are NOT
//     comparable to pre-inventory scores; review the failing-test list,
//     not just the exit code.
//   - errors.New/fmt.Errorf sites are not wire refusals and carry no arm.
//     They live only in validatePlatforms (funneled into one *Error arm
//     whose detail set is pinned exactly) and DigestFile (I/O errors that
//     must never be *Error). Both properties are asserted below, in every
//     FuncDecl body plus every package-level var initializer, and any
//     reference to either constructor outside direct-call position
//     (e.g. `newPlain := errors.New`) fails the suite, so an aliased
//     construction cannot ship unattributed.
//   - The DigestFile half of that allowlist is function-wide with no
//     detail pin, unlike validatePlatforms' derived exact set: a new
//     errors.New on any path inside DigestFile passes the gate (and the
//     never-wire behavioral assertion, which only drives the current
//     paths). Disclosed, not closed.
//   - Custom error types outside the two recognised spellings (*Error,
//     errors.New, fmt.Errorf) are outside both derivations: a bespoke
//     error type smuggled as a refusal survives. Disclosed, not closed.
//   - Constructor Details are single static literals by assertion: the
//     derivation fatals on any mismatchf/integrityFailure call without
//     exactly one string-literal argument, so format-verb interpolation
//     cannot enter the table. Zero multi-argument sites today; go vet's
//     printf analyzer independently refuses a non-constant format string.
package terminalbackend

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// refusalArm is one derived production refusal: a wire-code *Error
// construction site keyed by file, enclosing function (receiver-qualified
// for methods), code symbol, static detail, and occurrence index within
// that key. The index keeps two identical arms in one function distinct:
// adding a second arm with an already-declared key fails on the missing
// #2 row instead of hiding behind #1.
type refusalArm struct {
	file       string
	function   string
	code       string
	detail     string
	occurrence int
}

// constructorFunctions build refusals instead of issuing them. Their own
// *Error sites are the funnel, not arms.
var constructorFunctions = map[string]bool{
	"mismatchf":        true,
	"integrityFailure": true,
}

// passthroughDetail marks the single dynamic-detail funnel: Registration
// validation wraps validatePlatforms errors via err.Error(). Every other
// Detail expression must be a static string literal; a new dynamic detail
// fails the derivation rather than entering the table silently.
const passthroughDetail = "passthrough:err.Error()"

// deriveRefusalArms walks every production file of this package (every
// *.go except *_test.go, so a new production file is scanned without
// anyone remembering it) and returns every refusal arm in source order.
// It fails closed: an unparseable file, a non-static code or detail, and
// an empty result are all fatal, never an empty table.
//
// Coverage is declaration-complete, not FuncDecl-complete: every FuncDecl
// body (including nested func literals, methods, and init) plus every
// package-level var initializer is scanned, so a refusal arm hiding in a
// package-level `var f = func() *Error {...}` is derived like any other.
// Constructor references outside direct-call position (`refuse :=
// mismatchf`) are rejected outright by rejectConstructorAliases below:
// an aliased refusal cannot be inventoried, so it fails the suite rather
// than shipping silently.
func deriveRefusalArms(t *testing.T) []refusalArm {
	t.Helper()

	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob production files: %v", err)
	}
	var production []string
	for _, match := range matches {
		if !strings.HasSuffix(match, "_test.go") {
			production = append(production, match)
		}
	}
	if len(production) == 0 {
		t.Fatal("derived zero production files; the scanner is broken, not the package")
	}

	var arms []refusalArm
	parsed := 0
	for _, name := range production {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v; an unparseable derivation proves nothing", name, err)
		}
		parsed++
		collect := func(functionName string, root ast.Node) {
			ast.Inspect(root, func(node ast.Node) bool {
				switch expression := node.(type) {
				case *ast.UnaryExpr:
					if expression.Op != token.AND {
						return true
					}
					literal, ok := expression.X.(*ast.CompositeLit)
					if !ok {
						return true
					}
					identifier, ok := literal.Type.(*ast.Ident)
					if !ok || identifier.Name != "Error" {
						return true
					}
					code, detail := "", ""
					for _, element := range literal.Elts {
						pair, ok := element.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := pair.Key.(*ast.Ident)
						if !ok {
							continue
						}
						switch key.Name {
						case "Code":
							symbol, ok := pair.Value.(*ast.Ident)
							if !ok {
								t.Fatalf("%s: refusal code is not a static Code symbol; "+
									"a computed code cannot be inventoried", functionName)
							}
							code = symbol.Name
						case "Detail":
							detail = staticDetail(t, functionName, pair.Value)
						}
					}
					if code == "" || detail == "" {
						t.Fatalf("%s: refusal without a static code and detail; "+
							"the inventory cannot name what it cannot read", functionName)
					}
					arms = append(arms, refusalArm{
						file: name, function: functionName, code: code, detail: detail,
					})
				case *ast.CallExpr:
					identifier, ok := expression.Fun.(*ast.Ident)
					if !ok || !constructorFunctions[identifier.Name] {
						return true
					}
					if len(expression.Args) != 1 {
						t.Fatalf("%s: %s with %d arguments; Detail must be a single "+
							"static literal and never interpolate local data, and a "+
							"variadic detail cannot be inventoried",
							functionName, identifier.Name, len(expression.Args))
					}
					literal, ok := expression.Args[0].(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						t.Fatalf("%s: %s detail is not a static string literal; "+
							"a computed detail cannot be inventoried", functionName, identifier.Name)
					}
					detail, err := strconv.Unquote(literal.Value)
					if err != nil {
						t.Fatalf("%s: unquote detail: %v", functionName, err)
					}
					code := "CodeMismatch"
					if identifier.Name == "integrityFailure" {
						code = "CodeIntegrityFailure"
					}
					arms = append(arms, refusalArm{
						file: name, function: functionName, code: code, detail: detail,
					})
				}
				return true
			})
		}
		for _, declaration := range file.Decls {
			switch decl := declaration.(type) {
			case *ast.FuncDecl:
				if decl.Body == nil {
					continue
				}
				if constructorFunctions[decl.Name.Name] {
					continue
				}
				functionName := decl.Name.Name
				if decl.Recv != nil && len(decl.Recv.List) > 0 {
					functionName = receiverTypeName(decl.Recv.List[0].Type) + "." + functionName
				}
				collect(functionName, decl.Body)
			case *ast.GenDecl:
				// Package-level initializers execute outside any FuncDecl,
				// so a func literal (or a bare &Error{}) assigned to a
				// package var would otherwise ship invisible. Attribute
				// its arms to the declared variable name.
				for _, spec := range decl.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					functionName := "var"
					if len(value.Names) > 0 {
						functionName = value.Names[0].Name
					}
					for _, item := range value.Values {
						collect(functionName, item)
					}
				}
			}
		}
		rejectConstructorAliases(t, file, name)
	}
	_ = parsed
	if len(arms) == 0 {
		t.Fatal("derived zero refusal arms from the package sources; the scanner is broken, not the package")
	}
	occurrences := make(map[refusalArm]int)
	for index := range arms {
		key := arms[index]
		key.occurrence = 0
		occurrences[key]++
		arms[index].occurrence = occurrences[key]
	}
	return arms
}

// rejectConstructorAliases fails the suite when a refusal constructor
// (mismatchf, integrityFailure) is referenced outside direct-call
// position. `refuse := mismatchf; refuse("…")` ships a reachable arm the
// arm derivation above cannot attribute to a static call site, so the
// reference itself is the violation: call the constructor directly.
// The constructors' own declarations are excluded; their bodies are the
// funnel, not arms. Any other use — alias, argument, field, shadow —
// fails closed, because no legitimate production shape needs it.
func rejectConstructorAliases(t *testing.T, file *ast.File, name string) {
	t.Helper()

	allowed := map[token.Pos]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if !ok {
			if call, ok := node.(*ast.CallExpr); ok {
				if identifier, ok := call.Fun.(*ast.Ident); ok && constructorFunctions[identifier.Name] {
					allowed[identifier.Pos()] = true
				}
			}
			return true
		}
		if constructorFunctions[declaration.Name.Name] {
			allowed[declaration.Name.Pos()] = true
			return false
		}
		return true
	})
	ast.Inspect(file, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if ok && constructorFunctions[declaration.Name.Name] {
			return false
		}
		identifier, ok := node.(*ast.Ident)
		if !ok || !constructorFunctions[identifier.Name] || allowed[identifier.Pos()] {
			return true
		}
		t.Fatalf("%s: constructor %q referenced outside direct-call position; "+
			"an aliased refusal cannot be inventoried, call it directly", name, identifier.Name)
		return true
	})
}

// receiverTypeName renders a method receiver type to its bare name.
func receiverTypeName(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return receiverTypeName(typed.X)
	default:
		return "unknown"
	}
}

// staticDetail reads one Detail expression: a string literal, or the one
// dynamic funnel. Anything else fails the derivation.
func staticDetail(t *testing.T, function string, expression ast.Expr) string {
	t.Helper()

	if literal, ok := expression.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		detail, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatalf("%s: unquote detail: %v", function, err)
		}
		return detail
	}
	if call, ok := expression.(*ast.CallExpr); ok {
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Error" {
			if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == "err" {
				return passthroughDetail
			}
		}
	}
	t.Fatalf("%s: refusal Detail is not a static clause; Detail must never "+
		"interpolate local data, and a dynamic detail cannot be inventoried", function)
	return ""
}

// productionCodeSymbols resolves every Code symbol to its wire value from
// the production const declarations, so the declared table cannot drift
// from the codes production actually emits.
func productionCodeSymbols(t *testing.T) map[string]string {
	t.Helper()

	symbols := make(map[string]string)
	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob production files: %v", err)
	}
	for _, name := range matches {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
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
					if !strings.HasPrefix(name.Name, "Code") || index >= len(value.Values) {
						continue
					}
					literal, ok := value.Values[index].(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						t.Fatalf("code symbol %s is not a static string; "+
							"wire codes must be literals", name.Name)
					}
					wire, err := strconv.Unquote(literal.Value)
					if err != nil {
						t.Fatalf("unquote %s: %v", name.Name, err)
					}
					symbols[name.Name] = wire
				}
			}
		}
	}
	if len(symbols) == 0 {
		t.Fatal("resolved zero code symbols; the scanner is broken, not the package")
	}
	return symbols
}

// testBodies indexes every test declaration in this package by file and
// name to its body source. A duplicate test name across files fails: row
// references would be ambiguous.
func testBodies(t *testing.T) map[string]string {
	t.Helper()

	bodies := make(map[string]string)
	matches, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob test files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("found zero test files; the scanner is broken, not the package")
	}
	for _, name := range matches {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(function.Name.Name, "Test") {
				continue
			}
			start := fileSet.Position(function.Body.Pos()).Offset
			end := fileSet.Position(function.Body.End()).Offset
			key := name + "\n" + function.Name.Name
			if _, duplicate := bodies[key]; duplicate {
				t.Fatalf("test %s is declared twice; row references would be ambiguous", key)
			}
			bodies[key] = string(raw[start:end])
		}
	}
	if len(bodies) == 0 {
		t.Fatal("indexed zero test declarations, so every row below would resolve vacuously")
	}
	return bodies
}

// predicateCodes resolves every Is predicate to the code symbol it
// reports, verifying the shape: a single return of errorCode(err, Code).
// A predicate in any other shape fails: the inventory leans on these to
// resolve code-only assertions, so an unrecognized shape proves nothing.
func predicateCodes(t *testing.T) map[string]string {
	t.Helper()

	predicates := make(map[string]string)
	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob production files: %v", err)
	}
	for _, name := range matches {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(function.Name.Name, "Is") || function.Body == nil {
				continue
			}
			if len(function.Body.List) != 1 {
				t.Fatalf("predicate %s is not a single return; the inventory cannot resolve it",
					function.Name.Name)
			}
			returned, ok := function.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 1 {
				t.Fatalf("predicate %s is not a single return; the inventory cannot resolve it",
					function.Name.Name)
			}
			call, ok := returned.Results[0].(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				t.Fatalf("predicate %s does not call errorCode(err, Code); "+
					"the inventory cannot resolve it", function.Name.Name)
			}
			code, ok := call.Args[1].(*ast.Ident)
			if !ok {
				t.Fatalf("predicate %s reports a non-static code; the inventory cannot resolve it",
					function.Name.Name)
			}
			predicates[function.Name.Name] = code.Name
		}
	}
	if len(predicates) == 0 {
		t.Fatal("resolved zero predicates; the scanner is broken, not the package")
	}
	return predicates
}

// refusalTestRef names one asserting test by file and declaration name.
type refusalTestRef struct {
	file string
	test string
}

// declaredRefusalArm is one inventoried arm: the derived key (file,
// function, code symbol, static detail, occurrence), the public
// production entry its test drives, an optional stated bound for
// defensive arms no document can reach, and the tests asserting it.
type declaredRefusalArm struct {
	file       string
	function   string
	code       string
	detail     string
	occurrence int
	entry      string
	bound      string
	// detailSet carries the funneled detail set for the single
	// dynamic-detail row; it is empty for every literal row.
	detailSet []string
	tests     []refusalTestRef
}

// boundDefensiveReparse is the stated bound for re-parse arms after an
// already-validated member: timestampMember validated the member, and the
// Time re-parse cannot fail while scalar owns that contract. The arms
// stay as fail-closed defense; the bound says why no document reaches
// them, and TestDefensiveBoundsAreExactlyThese pins the set.
const boundDefensiveReparse = "defensive re-parse after validation; scalar owns the re-parse contract"

// declaredRefusalArms is the inventoried arm set. Field order is file,
// function (receiver-qualified for methods), code symbol, static detail,
// occurrence within that key, driven public entry, stated bound (empty
// except for defensive arms), funneled detail set (nil except the funnel
// row, which lives outside this table), and asserting tests.
//
// The table is generated from the derivation, never hand-enumerated:
// every row must match exactly one derived arm and vice versa, so adding
// a production arm without a row fails forward, and deleting one fails
// in reverse.
var declaredRefusalArms = []declaredRefusalArm{
	{"conformance.go", "CheckAttachRequest", "CodeMismatch", "document timestamp", 1, "CheckAttachRequest", boundDefensiveReparse, nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckAttachRequestBindsTransportInputAndExpiry"}}},
	{"conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach authorization binding", 1, "CheckAttachRequest", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckAttachRequestBindsTransportInputAndExpiry"}}},
	{"conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach authorization expiry", 1, "CheckAttachRequest", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckAttachRequestBindsTransportInputAndExpiry"}}},
	{"conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach relay transport", 1, "CheckAttachRequest", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckAttachRequestBindsTransportInputAndExpiry"}}},
	{"conformance.go", "CheckAttachResult", "CodeUnauthorized", "attach input binding", 1, "CheckAttachResult", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckAttachResultRequiresTripleEquality"}}},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint argv", 1, "CheckEntrypoint", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckEntrypointNamesItsArm"}}},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 1, "CheckEntrypoint", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckEntrypointNamesItsArm"}}},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 2, "CheckEntrypoint", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckEntrypointNamesItsArm"}}},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 3, "CheckEntrypoint", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckEntrypointNamesItsArm"}}},
	{"conformance.go", "CheckErrorAllowed", "CodeProtocolError", "operation error vocabulary", 1, "CheckErrorAllowed", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckErrorAllowedRefusesEveryUnlistedCode"}}},
	{"conformance.go", "CheckReplicable", "CodeProtocolError", "replication exclusion", 1, "CheckReplicable", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckReplicableRefusesEveryNonSafeClass"}}},
	{"conformance.go", "CheckStatusResult", "CodePreconditionFailed", "status attachability", 1, "CheckStatusResult", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckStatusResultEnforcesLookupRules"}}},
	{"conformance.go", "CheckStatusResult", "CodeProtocolError", "status identity binding", 1, "CheckStatusResult", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckStatusResultEnforcesLookupRules"}}},
	{"conformance.go", "CheckStatusResult", "CodeProtocolError", "status identity binding", 2, "CheckStatusResult", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckStatusResultEnforcesLookupRules"}}},
	{"conformance.go", "CheckStatusResult", "CodeProtocolError", "status provider observation", 1, "CheckStatusResult", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckStatusResultEnforcesLookupRules"}}},
	{"conformance.go", "CheckTransition", "CodePreconditionFailed", "lifecycle instance scope", 1, "CheckTransition", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckTransitionRefusesEveryIllegalSource"}}},
	{"conformance.go", "CheckTransition", "CodePreconditionFailed", "lifecycle transition", 1, "CheckTransition", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestCheckTransitionRefusesEveryIllegalSource"}}},
	{"conformance.go", "CheckTransition", "CodeProtocolError", "operation vocabulary", 1, "ParseOperation", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseOperationAdmitsOnlyTheTenClosedOperations"}}},
	{"conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 1, "IdempotencyKey", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestIdempotencyKeyRefusesWrongShapes"}}},
	{"conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 2, "IdempotencyKey", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestIdempotencyKeyRefusesWrongShapes"}}},
	{"conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 3, "IdempotencyKey", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestIdempotencyKeyRefusesWrongShapes"}}},
	{"conformance.go", "ImportLedger", "CodeIdempotencyMismatch", "idempotency ledger image", 1, "ImportLedger", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerSurvivesControllerCrashViaExportImport"}}},
	{"conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 1, "ImportLedger", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerSurvivesControllerCrashViaExportImport"}}},
	{"conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 2, "ImportLedger", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerSurvivesControllerCrashViaExportImport"}}},
	{"conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 3, "ImportLedger", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerSurvivesControllerCrashViaExportImport"}}},
	{"conformance.go", "Ledger.Bind", "CodeIdempotencyMismatch", "idempotency key conflict", 1, "Bind", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerBindsReplaysAndRefusesMismatch"}}},
	{"conformance.go", "Ledger.Bind", "CodeProtocolError", "idempotency key shape", 1, "Bind", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerBindsReplaysAndRefusesMismatch"}}},
	{"conformance.go", "Ledger.Bind", "CodeProtocolError", "idempotency ledger unavailable", 1, "Bind", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLedgerBindsReplaysAndRefusesMismatch"}}},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document member type", 1, "ParseAttachAuthorization", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document member type", 2, "ParseAttachAuthorization", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document members", 1, "ParseAttachAuthorization", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document timestamp", 1, "ParseAttachAuthorization", boundDefensiveReparse, nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document timestamp", 2, "ParseAttachAuthorization", boundDefensiveReparse, nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"conformance.go", "ParseAttachAuthorization", "CodeUnauthorized", "attach authorization expiry", 1, "ParseAttachAuthorization", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"conformance.go", "ParseInstanceState", "CodeProtocolError", "lifecycle state vocabulary", 1, "ParseInstanceState", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseInstanceStateAdmitsOnlyTheEightClosedStates"}}},
	{"conformance.go", "ParseOperation", "CodeProtocolError", "operation vocabulary", 1, "ParseOperation", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseOperationAdmitsOnlyTheTenClosedOperations"}}},
	{"conformance.go", "ParseSideEffect", "CodeProtocolError", "side effect vocabulary", 1, "ParseSideEffect", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseSideEffectAdmitsOnlyTheTenClosedEffects"}}},
	{"conformance.go", "ProjectToLegacy", "CodeIncompatibleSchema", "legacy reverse projection", 1, "ProjectToLegacy", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestLegacyReverseProjectionExistsOnlyForThePair"}}},
	{"conformance.go", "TranslateLegacyBackend", "CodeIncompatibleSchema", "legacy backend identity", 1, "TranslateLegacyBackend", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestHistoricalTranslationMapsOnlyTheImmutablePair"}}},
	{"conformance.go", "parseTransport", "CodeProtocolError", "presentation transport vocabulary", 1, "ParseAttachAuthorization", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestParseAttachAuthorizationRefusesNonNeutralPolicy"}}},
	{"manifest.go", "CapabilitiesForOperation", "CodeMismatch", "operation vocabulary", 1, "CapabilitiesForOperation", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestCapabilitiesForOperation"}}},
	{"manifest.go", "CheckOperation", "CodeMismatch", "operation vocabulary", 1, "CheckOperation", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestCapabilitiesForOperation"}}},
	{"manifest.go", "CheckOperation", "CodeCapabilityUnproven", "operation capability dependency", 1, "CheckOperation", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileAdmitsCoherentTuple"}}},
	{"manifest.go", "GenerationDigest", "CodeStaleGeneration", "backend_generation bound", 1, "GenerationDigest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestGenerationDigestBounds"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "capability vocabulary", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "document timestamp", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "document timestamp", 2, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence backend identity", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence expiry", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence issuer", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence platform", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence protocol major 1", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence schema", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence schema version", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence value", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "evidence list bound", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe availability", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe backend identity", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe executable digest", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe implementation kind", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe platform", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe protocol major 1", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe schema", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe schema version", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "Reconcile", "CodeIntegrityFailure", "evidence signature verifier", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "Registry.AdmitProbe", "CodeNotFound", "registry unavailable", 1, "AdmitProbe", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestAdmitProbeRefusesNilRegistry"}}},
	{"manifest.go", "UnsignedEvidenceBytes", "CodeIntegrityFailure", "evidence canonical bytes", 1, "UnsignedEvidenceBytes", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestUnsignedEvidenceBytesNeverFailsOnSchemaAdmittedInput"}}},
	{"manifest.go", "UnsignedEvidenceBytes", "CodeIntegrityFailure", "evidence canonical bytes", 2, "UnsignedEvidenceBytes", "", nil, []refusalTestRef{{file: "conformance_test.go", test: "TestUnsignedEvidenceBytesNeverFailsOnSchemaAdmittedInput"}}},
	{"manifest.go", "boundedStringMember", "CodeMismatch", "document string bound", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe addition registry binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_pin_test.go", test: "TestReconcileRefusesDriftedAddition"}}},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe omission of manifest claim", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe override of stable claim", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe override registry binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_pin_test.go", test: "TestReconcileRefusesDriftedOverride"}}},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe static claim echo", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe static claim without manifest", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkClosedList", "CodeMismatch", "document list bound", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkClosedList", "CodeMismatch", "document vocabulary", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkEvidenceCoverage", "CodeMismatch", "evidence requirement coverage", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceIDs", "CodeMismatch", "evidence id set binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceLiveness", "CodeMismatch", "document timestamp", 1, "Reconcile", boundDefensiveReparse, nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceLiveness", "CodeMismatch", "document timestamp", 2, "Reconcile", boundDefensiveReparse, nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceLiveness", "CodeMismatch", "evidence liveness", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceSet", "CodeMismatch", "conflicting evidence", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_pin_test.go", test: "TestReconcileRefusesConflictingEvidence"}}},
	{"manifest.go", "checkEvidenceSet", "CodeMismatch", "evidence claim binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceSignature", "CodeIntegrityFailure", "evidence attestation", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkEvidenceTuple", "CodeMismatch", "evidence tuple binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkExactMembers", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkExactMembers", "CodeMismatch", "document members", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkExtensions", "CodeMismatch", "document extensions", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkExtensions", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkIdentity", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "checkIdentity", "CodeMismatch", "document identity binding", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestIdentityMismatch"}}},
	{"manifest.go", "checkManifestRecordBinding", "CodeDrift", "manifest implementation drift", 1, "AdmitProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestAdmitProbeRefusesRecordDrift"}}},
	{"manifest.go", "checkManifestRecordBinding", "CodeUntrusted", "executable substitution", 1, "AdmitProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestAdmitProbeExternalRealm"}}},
	{"manifest.go", "checkProbeGeneration", "CodeStaleGeneration", "probe generation binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkProbeIdentity", "CodeMismatch", "probe manifest binding", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkProbeIdentity", "CodeUntrusted", "executable substitution", 1, "AdmitProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestAdmitProbeExternalRealm"}}},
	{"manifest.go", "checkProbeMembership", "CodeMismatch", "probe platform membership", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkProbeMembership", "CodeMismatch", "probe protocol membership", 1, "Reconcile", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestReconcileRefusals"}}},
	{"manifest.go", "checkSortedUnique", "CodeMismatch", "document ordering", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document duplicate member", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document nesting", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 3, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 4, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 5, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 6, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document encoding", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document shape", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document surrogate escape", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestDocumentSurrogateEscapeRefused"}}},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document trailing data", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestEncodingRefusals"}}},
	{"manifest.go", "digestMember", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "digestOrNullMember", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "digestOrNullMember", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "digestOrNullMember", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "objectIdentity", "CodeMismatch", "document identity", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestIdentityMismatch"}}},
	{"manifest.go", "objectIdentity", "CodeMismatch", "document identity", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestIdentityMismatch"}}},
	{"manifest.go", "parseAttestationSignature", "CodeMismatch", "evidence signature encoding", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseAttestationSignature", "CodeMismatch", "evidence signature scheme", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseClaim", "CodeMismatch", "capability registry binding", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaim", "CodeMismatch", "capability vocabulary", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaim", "CodeMismatch", "claim origin", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaim", "CodeMismatch", "claim shape", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaim", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaim", "CodeMismatch", "document member type", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaimList", "CodeMismatch", "claim list bound", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaimList", "CodeMismatch", "claim ordering", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseClaimList", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest backend identity", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest executable digest", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest implementation kind", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest schema", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest schema version", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parsePlatformList", "CodeMismatch", "platforms bound", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parsePlatformList", "CodeMismatch", "platforms ordering", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parsePlatformList", "CodeMismatch", "platforms vocabulary", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseRealmLiteral", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseRealmLiteral", "CodeMismatch", "evidence realm result", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseRealmMembers", "CodeMismatch", "evidence realm binding", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseRealmMembers", "CodeMismatch", "evidence realm binding", 2, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document member type", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document members", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document string bound", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "evidence provider identity", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "evidence realm binding", 1, "ParseEvidence", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestEvidenceDocumentRefusals"}}},
	{"manifest.go", "semverMember", "CodeMismatch", "document semver", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "stringArrayMember", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "stringArrayMember", "CodeMismatch", "document member type", 2, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "stringArrayMember", "CodeMismatch", "document members", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "stringMember", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "timestampMember", "CodeMismatch", "document timestamp", 1, "ParseProbe", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestProbeDocumentRefusals"}}},
	{"manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions bound", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions major 1", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions ordering", 1, "ParseManifest", "", nil, []refusalTestRef{{file: "manifest_test.go", test: "TestManifestDocumentRefusals"}}},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeDrift", "descriptor version binding", 1, "CheckProviderDescriptor", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckProviderDescriptor"}}},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor backend binding", 1, "CheckProviderDescriptor", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckProviderDescriptor"}}},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor binding digest", 1, "CheckProviderDescriptor", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckProviderDescriptorBindingDigestMismatch"}}},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor binding digest", 2, "CheckProviderDescriptor", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckProviderDescriptorBindingDigestMismatch"}}},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeStaleGeneration", "descriptor generation binding", 1, "CheckProviderDescriptor", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckProviderDescriptorGenerationMismatch"}}},
	{"terminalbackend.go", "CheckVersionTuple", "CodeDrift", "implementation_version semver", 1, "CheckVersionTuple", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckVersionTuple"}}},
	{"terminalbackend.go", "CheckVersionTuple", "CodeDrift", "protocol_version major 1", 1, "CheckVersionTuple", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckVersionTuple"}}},
	{"terminalbackend.go", "CheckVersionTuple", "CodeDrift", "protocol_version membership", 1, "CheckVersionTuple", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckVersionTuple"}}},
	{"terminalbackend.go", "DefaultForPlatform", "CodeNotFound", "platform vocabulary", 1, "DefaultForPlatform", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestDefaultForPlatform"}}},
	{"terminalbackend.go", "New", "CodeDrift", "implementation_version semver", 1, "New", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestNewRefusesBadVersionTuples"}}},
	{"terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id bound", 1, "ParseID", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestParseIDRefusesWidenedGrammar"}}},
	{"terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id grammar", 1, "ParseID", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestParseIDRefusesWidenedGrammar"}}},
	{"terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id reserved namespace", 1, "ParseID", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestParseIDRefusesReservedNamespace"}}},
	{"terminalbackend.go", "Registration.validate", "CodeDrift", "executable_digest must be null", 1, "validate", "", nil, []refusalTestRef{{file: "internal_pin_test.go", test: "TestValidateRefusesBuiltinKindWithDigest"}}},
	{"terminalbackend.go", "Registration.validate", "CodeDrift", "implementation_version semver", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "Registration.validate", "CodeUntrusted", "executable_digest", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeAmbiguous", "duplicate backend_id", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestDuplicateRegistrationIsRefused"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeAmbiguous", "external_trust identity binding", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeDrift", "implementation drift", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestDriftIsRefused"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeNotFound", "registry unavailable", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestResolveUnknownIsNeverAbsence"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "executable substitution", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "external implementation_kind", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusesUnknownKindAsUntrusted"}}},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "external_trust disabled", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "Registry.RequireRestoreBinding", "CodeNotFound", "registry unavailable", 1, "RequireRestoreBinding", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRequireRestoreBinding"}}},
	{"terminalbackend.go", "Registry.RequireRestoreBinding", "CodeRestoreMismatch", "restore requires the prior binding", 1, "RequireRestoreBinding", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRequireRestoreBinding"}}},
	{"terminalbackend.go", "Registry.Resolve", "CodeNotFound", "registry unavailable", 1, "Resolve", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestResolveUnknownIsNeverAbsence"}}},
	{"terminalbackend.go", "Registry.Resolve", "CodeNotFound", "unregistered terminal_backend_id", 1, "Resolve", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestResolveUnknownIsNeverAbsence"}}},
	{"terminalbackend.go", "TrustEntry.validate", "CodeAmbiguous", "external_trust reserved namespace", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusesReservedNamespaceDistinguishingValue"}}},
	{"terminalbackend.go", "TrustEntry.validate", "CodeUntrusted", "external_trust executable_digest", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "TrustEntry.validate", "CodeUntrusted", "external_trust executable_path", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "checkGeneration", "CodeStaleGeneration", "backend_generation bound", 1, "CheckProviderDescriptor", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestCheckProviderDescriptorGenerationBounds"}}},
	{"terminalbackend.go", "parseKind", "CodeNotFound", "implementation_kind vocabulary", 1, "parseKind", "", nil, []refusalTestRef{{file: "internal_pin_test.go", test: "TestParseKindAdmitsOnlyTheClosedVocabulary"}}},
	{"terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions bound", 1, "New", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestNewRefusesBadVersionTuples"}}},
	{"terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions major 1", 1, "RegisterExternal", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusals"}}},
	{"terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions sorted unique", 1, "New", "", nil, []refusalTestRef{{file: "terminalbackend_test.go", test: "TestNewRefusesBadVersionTuples"}}},
}

// funneledPlatformDetails is the exact error set validatePlatforms can
// produce, funneled through the single Registration.validate arm below.
// TestPlatformFunnelDetailsAreExactlyThese derives the set from the
// production AST and requires equality, so a fourth platform error is a
// red rather than a silently widened funnel.
var funneledPlatformDetails = []string{
	"platforms bound",
	"platforms vocabulary",
	"platforms sorted unique",
}

// funneledPlatformArm declares the single dynamic-detail arm. Its tests
// must mention every funneled detail, not just one of them.
var funneledPlatformArm = declaredRefusalArm{
	file: "terminalbackend.go", function: "Registration.validate",
	code: "CodeNotFound", detail: passthroughDetail, occurrence: 1,
	entry: "RegisterExternal",
	tests: []refusalTestRef{{file: "terminalbackend_test.go", test: "TestRegisterExternalRefusesPlatformViolations"}},
}

// armlessCodes names wire codes with no refusal arm by design.
// CodeUnavailable appears only inside the allowed-error sets; no
// conformance refusal emits it, so it carries no arm of its own.
var armlessCodes = map[string]bool{
	"CodeUnavailable": true,
}

// plainErrorFunctions names the only production functions allowed to
// construct non-wire errors. validatePlatforms feeds the pinned funnel;
// DigestFile reports I/O failures that must never be wire errors.
var plainErrorFunctions = map[string]bool{
	"validatePlatforms": true,
	"DigestFile":        true,
}

// TestDerivedRefusalArmsAreAllDeclared is the forward direction: every
// arm the production AST derives resolves to exactly one declared row.
// A brand-new arm with a brand-new detail fails here until its witness
// is recorded.
func TestDerivedRefusalArmsAreAllDeclared(t *testing.T) {
	t.Parallel()

	derived := deriveRefusalArms(t)
	rows := declaredRows()
	matched := make([]bool, len(derived))
	for _, row := range rows {
		matches := []int{}
		for index, arm := range derived {
			if armMatchesRow(arm, row) {
				matches = append(matches, index)
			}
		}
		if len(matches) != 1 {
			t.Errorf("declared row %s resolves to %d derived arms, want exactly one",
				describeRow(row), len(matches))
			continue
		}
		if matched[matches[0]] {
			t.Errorf("declared row %s claims an arm an earlier row already claimed: %s",
				describeRow(row), describeArm(derived[matches[0]]))
			continue
		}
		matched[matches[0]] = true
	}
	for index, claimed := range matched {
		if !claimed {
			t.Errorf("derived arm has no declaring row, so nothing witnesses it: %s",
				describeArm(derived[index]))
		}
	}
}

// TestDeclaredRefusalArmsAreAllDerived is the reverse direction: every
// declared row resolves to exactly one derived arm. Without it the
// forward direction passes vacuously on a truncated derivation, and a
// deleted production guard lingers as a row pointing at nothing.
func TestDeclaredRefusalArmsAreAllDerived(t *testing.T) {
	t.Parallel()

	derived := deriveRefusalArms(t)
	for _, row := range declaredRows() {
		matches := 0
		for _, arm := range derived {
			if armMatchesRow(arm, row) {
				matches++
			}
		}
		if matches != 1 {
			t.Errorf("declared row %s matches %d derived arms, want exactly one; "+
				"a row without an arm is a witness without a guard", describeRow(row), matches)
		}
	}
	if len(derived) != len(declaredRows()) {
		t.Errorf("derived %d arms but declared %d rows; the table is not a bijection",
			len(derived), len(declaredRows()))
	}
}

func armMatchesRow(arm refusalArm, row declaredRefusalArm) bool {
	return arm.file == row.file && arm.function == row.function &&
		arm.code == row.code && arm.detail == row.detail &&
		arm.occurrence == row.occurrence
}

func describeArm(arm refusalArm) string {
	return arm.file + " " + arm.function + " " + arm.code + " " +
		strconv.Quote(arm.detail) + " #" + strconv.Itoa(arm.occurrence)
}

func describeRow(row declaredRefusalArm) string {
	return row.file + " " + row.function + " " + row.code + " " +
		strconv.Quote(row.detail) + " #" + strconv.Itoa(row.occurrence)
}

// declaredRows returns the table plus the funnel row.
func declaredRows() []declaredRefusalArm {
	rows := append([]declaredRefusalArm(nil), declaredRefusalArms...)
	funnel := funneledPlatformArm
	funnel.detailSet = funneledPlatformDetails
	return append(rows, funnel)
}

// TestDeclaredRefusalWiresMatchProductionConsts binds the table to the
// production const values two ways: every row's code resolves to the
// wire its test asserts, and every Code symbol except the named armless
// one appears in at least one row, so a code nobody refuses is a red
// rather than a quiet gap.
func TestDeclaredRefusalWiresMatchProductionConsts(t *testing.T) {
	t.Parallel()

	symbols := productionCodeSymbols(t)
	covered := make(map[string]bool)
	for _, row := range declaredRows() {
		wire, known := symbols[row.code]
		if !known {
			t.Errorf("declared row %s names unknown code symbol %q",
				describeRow(row), row.code)
			continue
		}
		covered[row.code] = true
		_ = wire
	}
	for symbol := range symbols {
		if armlessCodes[symbol] {
			continue
		}
		if !covered[symbol] {
			t.Errorf("code symbol %s has no refusal arm; a wire code nothing refuses is unwitnessed", symbol)
		}
	}
}

// TestDeclaredRefusalTestsResolve requires every row to name real tests
// that textually touch the arm: the detail literal, or the driven entry
// plus the wire code (or its verified predicate). Bound rows name a test
// driving the entry; the bound itself says why no document reaches them.
func TestDeclaredRefusalTestsResolve(t *testing.T) {
	t.Parallel()

	bodies := testBodies(t)
	predicates := predicateCodes(t)
	symbols := productionCodeSymbols(t)
	for _, row := range declaredRows() {
		if len(row.tests) == 0 {
			t.Errorf("declared row %s names no test, so nothing measures it", describeRow(row))
			continue
		}
		resolved := false
		for _, reference := range row.tests {
			body, declared := bodies[reference.file+"\n"+reference.test]
			if !declared {
				t.Errorf("declared row %s names %s in %s, which is not declared",
					describeRow(row), reference.test, reference.file)
				continue
			}
			if rowResolves(row, body, predicates, symbols) {
				resolved = true
			}
		}
		if !resolved {
			t.Errorf("declared row %s names no test touching it: want the detail %q or the entry %q "+
				"with its wire code in %v", describeRow(row), row.detail, row.entry, row.tests)
		}
	}
}

// rowResolves reports whether one test body touches one row.
func rowResolves(row declaredRefusalArm, body string, predicates map[string]string, symbols map[string]string) bool {
	if row.detail == passthroughDetail {
		for _, detail := range row.detailSet {
			if !strings.Contains(body, detail) {
				return false
			}
		}
		return strings.Contains(body, row.entry)
	}
	if strings.Contains(body, row.detail) {
		return true
	}
	if row.bound != "" {
		return strings.Contains(body, row.entry)
	}
	if !strings.Contains(body, row.entry) {
		return false
	}
	if strings.Contains(body, symbols[row.code]) {
		return true
	}
	for predicate, code := range predicates {
		if code == row.code && strings.Contains(body, predicate) {
			return true
		}
	}
	return false
}

// TestDefensiveBoundsAreExactlyThese pins the set of bound rows: a new
// defensive arm must join this set deliberately, with its bound stated,
// rather than slipping in as an ordinary row whose test cannot reach it.
func TestDefensiveBoundsAreExactlyThese(t *testing.T) {
	t.Parallel()

	want := map[string]bool{
		"conformance.go ParseAttachAuthorization CodeMismatch \"document timestamp\" #1": true,
		"conformance.go ParseAttachAuthorization CodeMismatch \"document timestamp\" #2": true,
		"conformance.go CheckAttachRequest CodeMismatch \"document timestamp\" #1":       true,
		"manifest.go checkEvidenceLiveness CodeMismatch \"document timestamp\" #1":       true,
		"manifest.go checkEvidenceLiveness CodeMismatch \"document timestamp\" #2":       true,
	}
	for _, row := range declaredRows() {
		if row.bound == "" {
			continue
		}
		if row.bound != boundDefensiveReparse {
			t.Errorf("declared row %s carries an unrecognized bound %q",
				describeRow(row), row.bound)
			continue
		}
		if !want[describeRow(row)] {
			t.Errorf("declared row %s claims a bound outside the pinned set; "+
				"a new defensive arm joins deliberately or not at all", describeRow(row))
		}
	}
	bound := 0
	for _, row := range declaredRows() {
		if row.bound != "" {
			bound++
		}
	}
	if bound != len(want) {
		t.Errorf("declared %d bound rows, want exactly the pinned %d", bound, len(want))
	}
}

// TestPlatformFunnelDetailsAreExactlyThese derives the error set of
// validatePlatforms from the production AST and requires it to equal the
// funneled set: a fourth platform error widens the funnel this test pins.
func TestPlatformFunnelDetailsAreExactlyThese(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "terminalbackend.go", nil, 0)
	if err != nil {
		t.Fatalf("parse terminalbackend.go: %v", err)
	}
	details := map[string]bool{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "validatePlatforms" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "New" {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok || receiver.Name != "errors" || len(call.Args) != 1 {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatalf("validatePlatforms error is not a static literal")
				return true
			}
			detail, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatalf("unquote platform error: %v", err)
			}
			details[detail] = true
			return true
		})
	}
	if len(details) == 0 {
		t.Fatal("derived zero platform errors; the scanner is broken, not the package")
	}
	for _, want := range funneledPlatformDetails {
		if !details[want] {
			t.Errorf("funneled detail %q no longer derives from validatePlatforms", want)
		}
	}
	for detail := range details {
		known := false
		for _, want := range funneledPlatformDetails {
			if detail == want {
				known = true
			}
		}
		if !known {
			t.Errorf("validatePlatforms derives %q, outside the funneled set; "+
				"a fourth platform error widens the funnel", detail)
		}
	}
}

// TestPlainErrorsLiveOnlyInTheirTwoFunctions closes the smuggling hole:
// a non-wire error constructed anywhere but validatePlatforms (the
// pinned funnel) or DigestFile (I/O) fails, so a refusal wearing a plain
// error's clothes cannot ship unwitnessed.
//
// Coverage is declaration-complete like deriveRefusalArms above: every
// FuncDecl body (receiver-qualified for methods) plus every package-level
// var initializer is scanned, so a plain error hiding in a package-level
// `var errPlanted = errors.New(…)` or a `var f = func() error {…}`
// literal is caught like any other. An unused package-level plain error
// still fails: its initializer runs at package init, so the construction
// is real whether or not any caller returns it. An alias
// (`newPlain := errors.New`) is rejected outright by
// rejectPlainErrorAliases below: the allowlist scan matches only direct
// selector calls, so an aliased construction would ship unattributed.
// Custom error types outside the two recognised spellings are a stated
// bound above, not covered here.
func TestPlainErrorsLiveOnlyInTheirTwoFunctions(t *testing.T) {
	t.Parallel()

	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob production files: %v", err)
	}
	for _, name := range matches {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		rejectPlainErrorAliases(t, file, name)
		check := func(functionName string, root ast.Node) {
			ast.Inspect(root, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isPlainErrorConstruction(call) && !plainErrorFunctions[functionName] {
					t.Errorf("%s constructs a non-wire error in %s, outside the two allowlisted functions; "+
						"a refusal wearing a plain error's clothes ships unwitnessed", name, functionName)
				}
				return true
			})
		}
		for _, declaration := range file.Decls {
			switch decl := declaration.(type) {
			case *ast.FuncDecl:
				if decl.Body == nil {
					continue
				}
				functionName := decl.Name.Name
				if decl.Recv != nil && len(decl.Recv.List) > 0 {
					functionName = receiverTypeName(decl.Recv.List[0].Type) + "." + functionName
				}
				check(functionName, decl.Body)
			case *ast.GenDecl:
				// Package-level initializers execute outside any FuncDecl,
				// so a plain error assigned to a package var would
				// otherwise ship invisible. Attribute its constructions
				// to the declared variable name.
				for _, spec := range decl.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					functionName := "var"
					if len(value.Names) > 0 {
						functionName = value.Names[0].Name
					}
					for _, item := range value.Values {
						check(functionName, item)
					}
				}
			}
		}
	}
}

// isPlainErrorConstruction reports the two recognised plain-error
// spellings: errors.New and fmt.Errorf (any Errorf-prefixed constructor).
func isPlainErrorConstruction(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return isPlainErrorSelector(selector)
}

// isPlainErrorSelector reports the selector half of the two recognised
// plain-error spellings, mirroring isPlainErrorConstruction without
// requiring call position.
func isPlainErrorSelector(selector *ast.SelectorExpr) bool {
	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	if receiver.Name == "errors" && selector.Sel.Name == "New" {
		return true
	}
	return receiver.Name == "fmt" && strings.HasPrefix(selector.Sel.Name, "Errorf")
}

// rejectPlainErrorAliases fails the suite when a plain-error constructor
// (errors.New, fmt.Errorf and its Errorf-prefixed kin) is referenced
// outside direct-call position. `newPlain := errors.New; newPlain("…")`
// ships a reachable construction the allowlist scan above cannot attribute
// to a static call site — isPlainErrorConstruction matches only direct
// selector calls — so the reference itself is the violation: call the
// constructor directly. The alias is rejected in every function, including
// the two allowlisted ones, because an aliased construction inside
// validatePlatforms would bypass the funneled detail pin.
func rejectPlainErrorAliases(t *testing.T, file *ast.File, name string) {
	t.Helper()

	allowed := map[token.Pos]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isPlainErrorConstruction(call) {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			allowed[selector.Sel.Pos()] = true
		}
		return true
	})
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || !isPlainErrorSelector(selector) || allowed[selector.Sel.Pos()] {
			return true
		}
		t.Fatalf("%s: plain-error constructor %q referenced outside direct-call position; "+
			"an aliased plain error cannot be allowlisted, call it directly", name, selector.Sel.Name)
		return true
	})
}

// TestDigestFileRefusalsAreNeverWireErrors pins the allowlist's other
// half behaviorally: DigestFile failures are I/O errors, never *Error,
// so no caller can mistake them for a registry refusal.
func TestDigestFileRefusalsAreNeverWireErrors(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join("testdata", "no-such-executable"),
		"",
	} {
		_, err := DigestFile(path)
		if err == nil {
			t.Errorf("DigestFile(%q) = nil, want an I/O error", path)
			continue
		}
		var refusal *Error
		if errors.As(err, &refusal) {
			t.Errorf("DigestFile(%q) error = %v, want a plain I/O error, never a wire refusal", path, err)
		}
	}
}
