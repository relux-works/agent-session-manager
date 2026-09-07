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
// Resolution is EXECUTED, not textual: every non-bound row carries a
// witness executed through a public production entry point by
// TestEveryDeclaredArmRefusesAtItsEntry (owns its proof in
// refusal_arm_witnesses_test.go), which requires the exact wire code at
// the exact static detail. A row that merely mentions the detail in some
// test's source proves nothing and resolves nothing.
//
// A pair prove alone does NOT attribute the arm: 62 of the 184 witnessed
// arms share their (code, detail) clause with a sibling, so a widened
// earlier guard swallows a later arm's inputs and the later witness
// still passes through the sibling (P8: CheckEntrypoint's first session
// parse widened to also compare, killing the second parse and the
// comparison arms, left every witness green). Attribution is the pair
// prove PLUS the exercised-site audit in refusal_site_audit_test.go:
// every production refusal construction records its file:line at
// runtime, the post-run audit requires every derived non-bound line to
// have fired, and TestDerivedSiteLinesAreExactlyRowed maps every row to
// exactly one derived line. A dead-by-construction arm never fires, so
// it reddens in the audit even though its pair still refuses through its
// sibling. Live example of the bound alternative: CheckTransition's
// "operation vocabulary" arm (conformance.go) is unreachable —
// CheckTransition parses through ParseOperation first and
// lookupTransition covers every admitted operation — so the row declares
// boundUnreachableVocabulary, pinned by
// TestCheckTransitionOperationVocabularyIsUnreachable, instead of
// resolving through ParseOperation's sibling arm sharing the detail.
// The five re-parse arms declare boundDefensiveReparse the same way, and
// TestDefensiveBoundsAreExactlyThese pins the full bound set.
//
// Stated residual: the audit binds every ROW to a fired LINE, but not
// every WITNESS to its row's line — two witnesses firing each other's
// sites in complementary swap would stay green. No such swap exists
// (each prove documents its pass-earlier-sites construction), and the
// shape is perverse rather than adjacent, but it is disclosed, not
// closed. provider and provhost share the same residual by construction:
// their witnesses bind codes, their audits bind lines.
//
// Stated bounds (not inferred):
//   - The inventory proves every arm is DECLARED and every non-bound arm
//     is WITNESSED at a public entry with its exact code and detail.
//     Bound arms (defensive re-parses, the unreachable vocabulary arm)
//     are proved by their pinning tests, which fail closed when the
//     production shape they assume changes.
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
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// refusalArm is one derived production refusal: a wire-code *Error
// construction site keyed by file, enclosing function (receiver-qualified
// for methods), code symbol, static detail, and occurrence index within
// that key. The index keeps two identical arms in one function distinct:
// adding a second arm with an already-declared key fails on the missing
// #2 row instead of hiding behind #1.
//
// Line carries the production file:line of the construction call: the
// refuse() call line for &Error literals, the constructor call line for
// mismatchf/integrityFailure. The exercised-site audit requires every
// derived line to fire at least once across the full suite run, so an
// arm shadowed dead by a widened sibling guard reddens there even though
// its (code, detail) pair still refuses through the sibling. Two arms
// sharing one line fail the derivation: one line cannot attribute two
// arms.
type refusalArm struct {
	file       string
	function   string
	code       string
	detail     string
	occurrence int
	line       int
	// backendID records whether the refuse-wrapped &Error literal
	// carries a BackendID key, read off the literal's AST keys at
	// derivation (compositeLitNamesBackendID). Spelling-independent:
	// a multi-line composite literal names the same keys as a
	// single-line one.
	backendID bool
}

// compositeLitNamesBackendID reports whether a refuse-wrapped &Error
// literal carries a BackendID key. The check reads the literal's AST
// keys, so every source layout spells the same answer; a line-text
// filter ("BackendID:" on the arm's line) misses every other layout.
// Positional (unkeyed) literals cannot reach here: errorLiteral fails
// the derivation on a refusal without static Code/Detail keys, and
// refuse takes exactly one &Error literal, never a variable.
func compositeLitNamesBackendID(literal *ast.CompositeLit) bool {
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := pair.Key.(*ast.Ident); ok && key.Name == "BackendID" {
			return true
		}
	}
	return false
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

	return deriveRefusalArmsIn(t, ".")
}

// deriveRefusalArmsIn runs the inventory derivation against an arbitrary
// directory. Production runs use "."; derivation tests stage a synthetic
// package copy under t.TempDir() to prove the derivation against shapes
// the live tree does not contain.
func deriveRefusalArmsIn(t *testing.T, dir string) []refusalArm {
	t.Helper()

	// Production-file selection and fail-closed parsing are the shared
	// core's: a new production file is scanned without anyone
	// remembering it, and an unparseable file fails, never skips.
	files, fileSet := invcore.MustScanProduction(t, dir)

	var arms []refusalArm
	parsed := 0
	for _, production := range files {
		file, name := production.Syntax, production.Name
		parsed++
		lineOf := func(position token.Pos) int {
			return fileSet.Position(position).Line
		}
		// errorLiteral reads one &Error{Code:, Detail:} literal into its
		// static code symbol and detail. Anything else fails the
		// derivation: a refusal the inventory cannot name is a refusal it
		// cannot witness.
		errorLiteral := func(functionName string, literal *ast.CompositeLit) (string, string) {
			identifier, ok := literal.Type.(*ast.Ident)
			if !ok || identifier.Name != "Error" {
				t.Fatalf("%s: refuse wraps a non-Error literal; only &Error literals route through refuse", functionName)
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
			return code, detail
		}
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
					if identifier, ok := literal.Type.(*ast.Ident); !ok || identifier.Name != "Error" {
						return true
					}
					// A wrapped literal never reaches here: the refuse
					// CallExpr case below consumes it and prunes the
					// descent. An &Error literal built outside refuse is
					// an unregistered construction: it refuses without
					// recording its site, so the exercised-site audit
					// cannot see it, and it fails here instead of
					// shipping silently.
					t.Fatalf("%s (%s:%d): &Error literal outside refuse(); "+
						"route every production refusal through refuse so the site audit can attribute it",
						functionName, name, lineOf(expression.Pos()))
					return false
				case *ast.CallExpr:
					if identifier, ok := expression.Fun.(*ast.Ident); ok && identifier.Name == "refuse" {
						if len(expression.Args) != 1 {
							t.Fatalf("%s: refuse with %d arguments; refuse wraps exactly one &Error literal",
								functionName, len(expression.Args))
						}
						unary, ok := expression.Args[0].(*ast.UnaryExpr)
						if !ok || unary.Op != token.AND {
							t.Fatalf("%s (%s:%d): refuse wraps a non-literal; refuse takes exactly one &Error literal, never a variable",
								functionName, name, lineOf(expression.Pos()))
						}
						literal, ok := unary.X.(*ast.CompositeLit)
						if !ok {
							t.Fatalf("%s (%s:%d): refuse wraps a non-literal; refuse takes exactly one &Error literal, never a variable",
								functionName, name, lineOf(expression.Pos()))
						}
						code, detail := errorLiteral(functionName, literal)
						arms = append(arms, refusalArm{
							file: name, function: functionName, code: code, detail: detail,
							line:      lineOf(expression.Pos()),
							backendID: compositeLitNamesBackendID(literal),
						})
						return false
					}
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
					// mismatchf/integrityFailure construct
					// &Error{Code, Detail} with no BackendID key (their
					// Detail is a static clause by derivation), so these
					// arms never carry BackendID.
					arms = append(arms, refusalArm{
						file: name, function: functionName, code: code, detail: detail,
						line:      lineOf(expression.Pos()),
						backendID: false,
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
		invcore.MustAuditConstructorReferences(t, file, fileSet, name, refusalConstructorSpec())
	}
	_ = parsed
	if len(arms) == 0 {
		t.Fatal("derived zero refusal arms from the package sources; the scanner is broken, not the package")
	}
	occurrences := make(map[refusalArm]int)
	for index := range arms {
		// The occurrence index counts identical (file, function, code,
		// detail) arms, so the line (which distinguishes them at
		// runtime) and the derivation-only backendID flag stay out of
		// the key: zero them alongside the index.
		key := arms[index]
		key.occurrence = 0
		key.line = 0
		key.backendID = false
		occurrences[key]++
		arms[index].occurrence = occurrences[key]
	}
	// One line attributes one arm. Two arms sharing a file:line would
	// record the same exercised site, so a shadowed twin would hide
	// behind its sibling exactly the way P8 did before the site audit
	// existed. Keep one construction per line instead.
	seenLines := make(map[string]string)
	for _, arm := range arms {
		site := arm.file + ":" + strconv.Itoa(arm.line)
		if first, duplicate := seenLines[site]; duplicate {
			t.Fatalf("derived arms %s and %s share production line %s; "+
				"one line cannot attribute two arms, split the constructions",
				first, describeArm(arm), site)
		}
		seenLines[site] = describeArm(arm)
	}
	return arms
}

// refusalConstructorSpec is the core watch list for this package:
// package-local refusal constructors plus the plain-error spellings by
// import path, so an import alias cannot change what the audit sees. The
// union keeps terminalbackend's outright refusal and resolves the
// import-alias and var-binding shapes that walked through every
// identifier-keyed gate in this repository.
//
// refuse joins the local set: every production &Error literal routes
// through it, so a `refuse := refuse` binding or any other use outside
// direct-call position fails outright instead of shipping an
// unattributed refusal. TestRefuseFunnelRejectsAliases pins that.
func refusalConstructorSpec() invcore.ConstructorSpec {
	return invcore.ConstructorSpec{
		Local: map[string]bool{
			"mismatchf":        true,
			"integrityFailure": true,
			"refuse":           true,
		},
		Qualified: map[string]map[string]bool{
			"errors": {"New": true},
			"fmt":    {"Errorf*": true},
		},
	}
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
	files, _ := invcore.MustScanProduction(t, ".")
	for _, production := range files {
		file := production.Syntax
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

// declaredRefusalArm is one inventoried arm: the derived key (file,
// function, code symbol, static detail, occurrence), the public
// production entry its witness drives, an optional stated bound for
// arms no input can reach, and the funneled detail set (nil except the
// funnel row). Non-bound rows are proved by the executed witness of the
// same identity in refusal_arm_witnesses_test.go; bound rows are proved
// by their pinning tests in the pinned set below.
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
}

// boundDefensiveReparse is the stated bound for re-parse arms after an
// already-validated member: timestampMember validated the member, and the
// Time re-parse cannot fail while scalar owns that contract. The arms
// stay as fail-closed defense; the bound says why no document reaches
// them, and TestDefensiveBoundsAreExactlyThese pins the set.
const boundDefensiveReparse = "defensive re-parse after validation; scalar owns the re-parse contract"

// boundUnreachableVocabulary is the stated bound for CheckTransition's
// "operation vocabulary" arm: CheckTransition parses its operation
// through ParseOperation first, and lookupTransition covers every
// admitted operation, so the miss branch is unreachable by
// construction. TestCheckTransitionOperationVocabularyIsUnreachable pins
// the coverage; adding an operation to ParseOperation without a
// transition-table row fails the pin and promotes this row back to a
// witnessed arm.
const boundUnreachableVocabulary = "unreachable: ParseOperation admits exactly the tabled operations, so lookupTransition never misses"

// boundDecoderContract is the stated bound for decodeCappedValue's two
// defensive syntax branches: a non-string object key (#3) and a stray
// delimiter at a value position (#6). encoding/json Token() errors at
// both positions instead of yielding those values, so neither branch has
// an input. TestDecodeCappedValueDefensiveSitesAreUnreachable pins the
// decoder contract; a Go release that yields either value fails the pin
// and promotes both rows back to witnessed arms.
const boundDecoderContract = "unreachable: encoding/json Token() errors on non-string keys and stray delimiters, never yielding them"

// boundCanonicalPlumbing is the stated bound for the four Marshal/JCS
// error branches in objectIdentity and UnsignedEvidenceBytes: their maps
// are built from JSON-decoded values (numbers refused upstream by
// refuseIdentityNumbers, encoding refused by the decode gate) or from
// typed string-only structs, so json.Marshal cannot fail and jcs has no
// numbers, duplicates, or malformed bytes to refuse.
// TestCanonicalPlumbingIsTotalOnDecodedMaps pins totality over the
// reachable input class; a new failing shape fails the pin and promotes
// these rows back to witnessed arms.
const boundCanonicalPlumbing = "unreachable: json.Marshal and jcs.Transform cannot fail on number-free decoded maps"

// boundUnreachableKind is the stated bound for parseKind's vocabulary
// arm: both document callers wrap its error in their own clause, and
// RegisterExternal's external-kind gate fires first, so the arm's own
// clause never surfaces at any public entry. TestParseKindAdmitsOnlyThe-
// ClosedVocabulary pins the vocabulary white-box; TestRegisterExternal-
// RefusesUnknownKindAsUntrusted pins the gate order, so removing the
// gate fails there and promotes this row back to a witnessed arm.
const boundUnreachableKind = "unreachable: document callers wrap the error and the registry kind gate fires first"

// boundUnreachableNumbers is the stated bound for refuseIdentityNumbers'
// arm: every parse entry validates member types before checkIdentity, so
// a JSON number never reaches the walk; the walk holds that order for
// callers that skip the member checks.
// TestObjectIdentityRefusesNumbersBeforeCanonicalization pins the walk
// white-box. The sibling-swallow residual for this arm's shared clause
// is measured by the exercised-site audit, not by arm-deletion mutants:
// deleting the arm orphans its row (bijection kill), but only the audit
// sees an existing arm go dead while its pair still refuses through a
// sibling (battery row S-tb-shadow reproduces exactly that plant and
// kill; the old C-tb-dead-arm add-a-function row killed for the wrong
// reason and no longer counts as dead-arm coverage).
const boundUnreachableNumbers = "unreachable: member-type validation runs before checkIdentity on every parse entry"

// boundUnreachableDigestNull is the stated bound for Registration.validate's
// "executable_digest must be null" arm: RegisterExternal's external-kind
// gate admits exactly the digest-carrying kinds, so validate never sees a
// builtin-kind record with a digest at any public entry; New builds its
// builtins without digests. TestValidateRefusesBuiltinKindWithDigest pins
// the arm white-box by calling validate directly. Narrowing the kind gate
// fails TestRegisterExternalRefusesUnknownKindAsUntrusted and promotes
// this row back to a witnessed arm.
const boundUnreachableDigestNull = "unreachable: the registry kind gate admits exactly the digest-carrying kinds"

// boundShadowedLookup is the stated bound for the nine direct-miss arms
// shadowed by a preceding checkExactMembers over the same member list:
// when the exact check passes, the object keys equal the list exactly,
// so the later lookup of a listed member cannot miss.
// TestShadowedLookupsHaveNoInput proves the shadowing per bound row from
// the production AST, never from a hand list: it derives every
// "document members" miss, every exact check, and every helper call
// site, and requires each miss to classify covered (bound) or uncovered
// (witnessed) with its row kind agreeing in both directions. A list that
// loses a member un-covers its miss arm into a live one; a removed exact
// call does the same through the extra-member witnesses that pin the
// exact calls themselves.
const boundShadowedLookup = "unreachable: a preceding checkExactMembers over the same list fires first"

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
	{"conformance.go", "CheckAttachRequest", "CodeMismatch", "document timestamp", 1, "CheckAttachRequest", boundDefensiveReparse, nil},
	{"conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach authorization binding", 1, "CheckAttachRequest", "", nil},
	{"conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach authorization expiry", 1, "CheckAttachRequest", "", nil},
	{"conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach relay transport", 1, "CheckAttachRequest", "", nil},
	{"conformance.go", "CheckAttachResult", "CodeUnauthorized", "attach input binding", 1, "CheckAttachResult", "", nil},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint argv", 1, "CheckEntrypoint", "", nil},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 1, "CheckEntrypoint", "", nil},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 2, "CheckEntrypoint", "", nil},
	{"conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 3, "CheckEntrypoint", "", nil},
	{"conformance.go", "CheckErrorAllowed", "CodeProtocolError", "operation error vocabulary", 1, "CheckErrorAllowed", "", nil},
	{"conformance.go", "CheckReplicable", "CodeProtocolError", "replication exclusion", 1, "CheckReplicable", "", nil},
	{"conformance.go", "CheckStatusResult", "CodePreconditionFailed", "status attachability", 1, "CheckStatusResult", "", nil},
	{"conformance.go", "CheckStatusResult", "CodeProtocolError", "status identity binding", 1, "CheckStatusResult", "", nil},
	{"conformance.go", "CheckStatusResult", "CodeProtocolError", "status identity binding", 2, "CheckStatusResult", "", nil},
	{"conformance.go", "CheckStatusResult", "CodeProtocolError", "status provider observation", 1, "CheckStatusResult", "", nil},
	{"conformance.go", "CheckTransition", "CodePreconditionFailed", "lifecycle instance scope", 1, "CheckTransition", "", nil},
	{"conformance.go", "CheckTransition", "CodePreconditionFailed", "lifecycle transition", 1, "CheckTransition", "", nil},
	{"conformance.go", "CheckTransition", "CodeProtocolError", "operation vocabulary", 1, "CheckTransition", boundUnreachableVocabulary, nil},
	{"conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 1, "IdempotencyKey", "", nil},
	{"conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 2, "IdempotencyKey", "", nil},
	{"conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 3, "IdempotencyKey", "", nil},
	{"conformance.go", "ImportLedger", "CodeIdempotencyMismatch", "idempotency ledger image", 1, "ImportLedger", "", nil},
	{"conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 1, "ImportLedger", "", nil},
	{"conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 2, "ImportLedger", "", nil},
	{"conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 3, "ImportLedger", "", nil},
	{"conformance.go", "Ledger.Bind", "CodeIdempotencyMismatch", "idempotency key conflict", 1, "Bind", "", nil},
	{"conformance.go", "Ledger.Bind", "CodeProtocolError", "idempotency key shape", 1, "Bind", "", nil},
	{"conformance.go", "Ledger.Bind", "CodeProtocolError", "idempotency ledger unavailable", 1, "Bind", "", nil},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document member type", 1, "ParseAttachAuthorization", "", nil},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document member type", 2, "ParseAttachAuthorization", "", nil},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document members", 1, "ParseAttachAuthorization", boundShadowedLookup, nil},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document timestamp", 1, "ParseAttachAuthorization", boundDefensiveReparse, nil},
	{"conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document timestamp", 2, "ParseAttachAuthorization", boundDefensiveReparse, nil},
	{"conformance.go", "ParseAttachAuthorization", "CodeUnauthorized", "attach authorization expiry", 1, "ParseAttachAuthorization", "", nil},
	{"conformance.go", "ParseInstanceState", "CodeProtocolError", "lifecycle state vocabulary", 1, "ParseInstanceState", "", nil},
	{"conformance.go", "ParseOperation", "CodeProtocolError", "operation vocabulary", 1, "ParseOperation", "", nil},
	{"conformance.go", "ParseSideEffect", "CodeProtocolError", "side effect vocabulary", 1, "ParseSideEffect", "", nil},
	{"conformance.go", "ProjectToLegacy", "CodeIncompatibleSchema", "legacy reverse projection", 1, "ProjectToLegacy", "", nil},
	{"conformance.go", "TranslateLegacyBackend", "CodeIncompatibleSchema", "legacy backend identity", 1, "TranslateLegacyBackend", "", nil},
	{"conformance.go", "parseTransport", "CodeProtocolError", "presentation transport vocabulary", 1, "ParseAttachAuthorization", "", nil},
	{"manifest.go", "CapabilitiesForOperation", "CodeMismatch", "operation vocabulary", 1, "CapabilitiesForOperation", "", nil},
	{"manifest.go", "CheckOperation", "CodeMismatch", "operation vocabulary", 1, "CheckOperation", "", nil},
	{"manifest.go", "CheckOperation", "CodeCapabilityUnproven", "operation capability dependency", 1, "CheckOperation", "", nil},
	{"manifest.go", "GenerationDigest", "CodeStaleGeneration", "backend_generation bound", 1, "GenerationDigest", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "capability vocabulary", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "document timestamp", 1, "ParseEvidence", boundDefensiveReparse, nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "document timestamp", 2, "ParseEvidence", boundDefensiveReparse, nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence backend identity", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence expiry", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence issuer", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence platform", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence protocol major 1", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence schema", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence schema version", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseEvidence", "CodeMismatch", "evidence value", 1, "ParseEvidence", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "document digest", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "document members", 1, "ParseProbe", boundShadowedLookup, nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "evidence list bound", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe availability", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe backend identity", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe executable digest", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe implementation kind", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe platform", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe protocol major 1", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe schema", 1, "ParseProbe", "", nil},
	{"manifest.go", "ParseProbe", "CodeMismatch", "probe schema version", 1, "ParseProbe", "", nil},
	{"manifest.go", "Reconcile", "CodeIntegrityFailure", "evidence signature verifier", 1, "Reconcile", "", nil},
	{"manifest.go", "Registry.AdmitProbe", "CodeNotFound", "registry unavailable", 1, "AdmitProbe", "", nil},
	{"manifest.go", "UnsignedEvidenceBytes", "CodeIntegrityFailure", "evidence canonical bytes", 1, "UnsignedEvidenceBytes", boundCanonicalPlumbing, nil},
	{"manifest.go", "UnsignedEvidenceBytes", "CodeIntegrityFailure", "evidence canonical bytes", 2, "UnsignedEvidenceBytes", boundCanonicalPlumbing, nil},
	{"manifest.go", "boundedStringMember", "CodeMismatch", "document string bound", 1, "ParseProbe", "", nil},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe addition registry binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe omission of manifest claim", 1, "Reconcile", "", nil},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe override of stable claim", 1, "Reconcile", "", nil},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe override registry binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe static claim echo", 1, "Reconcile", "", nil},
	{"manifest.go", "checkClaimRelation", "CodeMismatch", "probe static claim without manifest", 1, "Reconcile", "", nil},
	{"manifest.go", "checkClosedList", "CodeMismatch", "document list bound", 1, "ParseManifest", "", nil},
	{"manifest.go", "checkClosedList", "CodeMismatch", "document vocabulary", 1, "ParseManifest", "", nil},
	{"manifest.go", "checkEvidenceCoverage", "CodeMismatch", "evidence requirement coverage", 1, "Reconcile", "", nil},
	{"manifest.go", "checkEvidenceIDs", "CodeMismatch", "evidence id set binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkEvidenceLiveness", "CodeMismatch", "document timestamp", 1, "Reconcile", boundDefensiveReparse, nil},
	{"manifest.go", "checkEvidenceLiveness", "CodeMismatch", "document timestamp", 2, "Reconcile", boundDefensiveReparse, nil},
	{"manifest.go", "checkEvidenceLiveness", "CodeMismatch", "evidence liveness", 1, "Reconcile", "", nil},
	{"manifest.go", "checkEvidenceSet", "CodeMismatch", "conflicting evidence", 1, "Reconcile", "", nil},
	{"manifest.go", "checkEvidenceSet", "CodeMismatch", "evidence claim binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkEvidenceSignature", "CodeIntegrityFailure", "evidence attestation", 1, "Reconcile", "", nil},
	{"manifest.go", "checkEvidenceTuple", "CodeMismatch", "evidence tuple binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkExactMembers", "CodeMismatch", "document members", 1, "ParseManifest", "", nil},
	{"manifest.go", "checkExactMembers", "CodeMismatch", "document members", 2, "ParseManifest", "", nil},
	{"manifest.go", "checkExtensions", "CodeMismatch", "document extensions", 1, "ParseManifest", "", nil},
	{"manifest.go", "checkExtensions", "CodeMismatch", "document members", 1, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "checkIdentity", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil},
	{"manifest.go", "checkIdentity", "CodeMismatch", "document identity binding", 1, "ParseManifest", "", nil},
	{"manifest.go", "checkManifestRecordBinding", "CodeDrift", "manifest implementation drift", 1, "AdmitProbe", "", nil},
	{"manifest.go", "checkManifestRecordBinding", "CodeUntrusted", "executable substitution", 1, "AdmitProbe", "", nil},
	{"manifest.go", "checkProbeGeneration", "CodeStaleGeneration", "probe generation binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkProbeIdentity", "CodeMismatch", "probe manifest binding", 1, "Reconcile", "", nil},
	{"manifest.go", "checkProbeIdentity", "CodeUntrusted", "executable substitution", 1, "AdmitProbe", "", nil},
	{"manifest.go", "checkProbeMembership", "CodeMismatch", "probe platform membership", 1, "Reconcile", "", nil},
	{"manifest.go", "checkProbeMembership", "CodeMismatch", "probe protocol membership", 1, "Reconcile", "", nil},
	{"manifest.go", "checkSortedUnique", "CodeMismatch", "document ordering", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document duplicate member", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document nesting", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 2, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 3, "ParseManifest", boundDecoderContract, nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 4, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 5, "ParseManifest", "", nil},
	{"manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 6, "ParseManifest", boundDecoderContract, nil},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document encoding", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document size", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document shape", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document surrogate escape", 1, "ParseManifest", "", nil},
	{"manifest.go", "decodeStrictObject", "CodeMismatch", "document trailing data", 1, "ParseManifest", "", nil},
	{"manifest.go", "digestMember", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil},
	{"manifest.go", "digestOrNullMember", "CodeMismatch", "document digest", 1, "ParseManifest", "", nil},
	{"manifest.go", "digestOrNullMember", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil},
	{"manifest.go", "digestOrNullMember", "CodeMismatch", "document members", 1, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "objectIdentity", "CodeMismatch", "document identity", 1, "ParseManifest", boundCanonicalPlumbing, nil},
	{"manifest.go", "objectIdentity", "CodeMismatch", "document identity", 2, "ParseManifest", boundCanonicalPlumbing, nil},
	{"manifest.go", "parseAttestationSignature", "CodeMismatch", "evidence signature encoding", 1, "ParseEvidence", "", nil},
	{"manifest.go", "parseAttestationSignature", "CodeMismatch", "evidence signature scheme", 1, "ParseEvidence", "", nil},
	{"manifest.go", "parseClaim", "CodeMismatch", "capability registry binding", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaim", "CodeMismatch", "capability vocabulary", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaim", "CodeMismatch", "claim origin", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaim", "CodeMismatch", "claim shape", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaim", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaim", "CodeMismatch", "document member type", 2, "ParseManifest", "", nil},
	{"manifest.go", "parseClaimList", "CodeMismatch", "claim list bound", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaimList", "CodeMismatch", "claim ordering", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseClaimList", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "document members", 1, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest backend identity", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest executable digest", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest implementation kind", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest schema", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseManifestObject", "CodeMismatch", "manifest schema version", 1, "ParseManifest", "", nil},
	{"manifest.go", "parsePlatformList", "CodeMismatch", "platforms bound", 1, "ParseManifest", "", nil},
	{"manifest.go", "parsePlatformList", "CodeMismatch", "platforms ordering", 1, "ParseManifest", "", nil},
	{"manifest.go", "parsePlatformList", "CodeMismatch", "platforms vocabulary", 1, "ParseManifest", "", nil},
	{"manifest.go", "parseRealmLiteral", "CodeMismatch", "document members", 1, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "parseRealmLiteral", "CodeMismatch", "evidence realm result", 1, "ParseEvidence", "", nil},
	{"manifest.go", "parseRealmMembers", "CodeMismatch", "evidence realm binding", 1, "ParseEvidence", "", nil},
	{"manifest.go", "parseRealmMembers", "CodeMismatch", "evidence realm binding", 2, "ParseEvidence", "", nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document member type", 1, "ParseEvidence", "", nil},
	{"manifest.go", "refuseIdentityNumbers", "CodeMismatch", "document member type", 1, "ParseManifest", boundUnreachableNumbers, nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document member type", 2, "ParseEvidence", "", nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document members", 1, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document members", 2, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "document string bound", 1, "ParseEvidence", "", nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "evidence provider identity", 1, "ParseEvidence", "", nil},
	{"manifest.go", "parseRealmProvider", "CodeMismatch", "evidence realm binding", 1, "ParseEvidence", "", nil},
	{"manifest.go", "semverMember", "CodeMismatch", "document semver", 1, "ParseManifest", "", nil},
	{"manifest.go", "stringArrayMember", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil},
	{"manifest.go", "stringArrayMember", "CodeMismatch", "document member type", 2, "ParseManifest", "", nil},
	{"manifest.go", "stringArrayMember", "CodeMismatch", "document members", 1, "ParseManifest", boundShadowedLookup, nil},
	{"manifest.go", "stringMember", "CodeMismatch", "document member type", 1, "ParseManifest", "", nil},
	{"manifest.go", "timestampMember", "CodeMismatch", "document timestamp", 1, "ParseProbe", "", nil},
	{"manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions bound", 1, "ParseManifest", "", nil},
	{"manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions major 1", 1, "ParseManifest", "", nil},
	{"manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions ordering", 1, "ParseManifest", "", nil},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeDrift", "descriptor version binding", 1, "CheckProviderDescriptor", "", nil},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor backend binding", 1, "CheckProviderDescriptor", "", nil},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor binding digest", 1, "CheckProviderDescriptor", "", nil},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor binding digest", 2, "CheckProviderDescriptor", "", nil},
	{"terminalbackend.go", "CheckProviderDescriptor", "CodeStaleGeneration", "descriptor generation binding", 1, "CheckProviderDescriptor", "", nil},
	{"terminalbackend.go", "CheckVersionTuple", "CodeDrift", "implementation_version semver", 1, "CheckVersionTuple", "", nil},
	{"terminalbackend.go", "CheckVersionTuple", "CodeDrift", "protocol_version major 1", 1, "CheckVersionTuple", "", nil},
	{"terminalbackend.go", "CheckVersionTuple", "CodeDrift", "protocol_version membership", 1, "CheckVersionTuple", "", nil},
	{"terminalbackend.go", "DefaultForPlatform", "CodeNotFound", "platform vocabulary", 1, "DefaultForPlatform", "", nil},
	{"terminalbackend.go", "New", "CodeDrift", "implementation_version semver", 1, "New", "", nil},
	{"terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id bound", 1, "ParseID", "", nil},
	{"terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id grammar", 1, "ParseID", "", nil},
	{"terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id reserved namespace", 1, "ParseID", "", nil},
	{"terminalbackend.go", "Registration.validate", "CodeDrift", "executable_digest must be null", 1, "RegisterExternal", boundUnreachableDigestNull, nil},
	{"terminalbackend.go", "Registration.validate", "CodeDrift", "implementation_version semver", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registration.validate", "CodeUntrusted", "executable_digest", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeAmbiguous", "duplicate backend_id", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeAmbiguous", "external_trust identity binding", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeDrift", "implementation drift", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeNotFound", "registry unavailable", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "executable substitution", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "external implementation_kind", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "external_trust disabled", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "Registry.RequireRestoreBinding", "CodeNotFound", "registry unavailable", 1, "RequireRestoreBinding", "", nil},
	{"terminalbackend.go", "Registry.RequireRestoreBinding", "CodeRestoreMismatch", "restore requires the prior binding", 1, "RequireRestoreBinding", "", nil},
	{"terminalbackend.go", "Registry.Resolve", "CodeNotFound", "registry unavailable", 1, "Resolve", "", nil},
	{"terminalbackend.go", "Registry.Resolve", "CodeNotFound", "unregistered terminal_backend_id", 1, "Resolve", "", nil},
	{"terminalbackend.go", "TrustEntry.validate", "CodeAmbiguous", "external_trust reserved namespace", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "TrustEntry.validate", "CodeUntrusted", "external_trust executable_digest", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "TrustEntry.validate", "CodeUntrusted", "external_trust executable_path", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "checkGeneration", "CodeStaleGeneration", "backend_generation bound", 1, "CheckProviderDescriptor", "", nil},
	{"terminalbackend.go", "parseKind", "CodeNotFound", "implementation_kind vocabulary", 1, "RegisterExternal", boundUnreachableKind, nil},
	{"terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions bound", 1, "New", "", nil},
	{"terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions major 1", 1, "RegisterExternal", "", nil},
	{"terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions sorted unique", 1, "New", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor member set", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor member set", 2, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor digest", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor instance", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor implementation version", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor protocol version", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor interactive", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "descriptorText", "CodeProtocolError", "descriptor member type", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry type", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry digits", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry bound", 1, "ParseProviderDescriptor", "", nil},
	{"descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry bound", 2, "ParseProviderDescriptor", "", nil},
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

// armInventoryKeys renders derived arms and declared rows into one shared
// key space for the core both-direction harness.
func armInventoryKeys(derived []refusalArm, rows []declaredRefusalArm) (map[string]struct{}, map[string]struct{}) {
	derivedKeys := make(map[string]struct{}, len(derived))
	for _, arm := range derived {
		derivedKeys[describeArm(arm)] = struct{}{}
	}
	rowKeys := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := describeRow(row)
		if _, duplicate := rowKeys[key]; duplicate {
			rowKeys["DUPLICATE-ROW:"+key] = struct{}{}
		}
		rowKeys[key] = struct{}{}
	}
	return derivedKeys, rowKeys
}

// TestDerivedRefusalArmsAreAllDeclared is the forward direction: every
// arm the production AST derives resolves to exactly one declared row.
// A brand-new arm with a brand-new detail fails here until its witness
// is recorded.
func TestDerivedRefusalArmsAreAllDeclared(t *testing.T) {
	t.Parallel()

	derived, rows := armInventoryKeys(deriveRefusalArms(t), declaredRows())
	invcore.MustCheckBothDirections(t, "refusal-arm inventory", derived, rows)
}

// TestDeclaredRefusalArmsAreAllDerived is the reverse direction: every
// declared row resolves to exactly one derived arm. Without it the
// forward direction passes vacuously on a truncated derivation, and a
// deleted production guard lingers as a row pointing at nothing.
func TestDeclaredRefusalArmsAreAllDerived(t *testing.T) {
	t.Parallel()

	derived, rows := armInventoryKeys(deriveRefusalArms(t), declaredRows())
	invcore.MustCheckBothDirections(t, "refusal-arm inventory", derived, rows)
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

// ArmIdentity is the export bridge to the executed-witness registry in
// refusal_arm_witnesses_test.go (package terminalbackend_test): one row
// identity with the entry its witness drives, its bound (empty except
// for arms no input can reach), and the funneled detail set for the
// single dynamic-detail row.
type ArmIdentity struct {
	File       string
	Function   string
	Code       string
	Detail     string
	Occurrence int
	Entry      string
	Bound      string
	DetailSet  []string
}

// DeclaredArmIdentities exports every declared row identity for the
// witness registry's reverse direction.
func DeclaredArmIdentities() []ArmIdentity {
	rows := declaredRows()
	identities := make([]ArmIdentity, 0, len(rows))
	for _, row := range rows {
		identities = append(identities, ArmIdentity{
			File: row.file, Function: row.function, Code: row.code,
			Detail: row.detail, Occurrence: row.occurrence,
			Entry: row.entry, Bound: row.bound, DetailSet: row.detailSet,
		})
	}
	return identities
}

// DerivedArmIdentities exports every derived arm identity for the
// witness registry's forward direction.
func DerivedArmIdentities(t *testing.T) []ArmIdentity {
	t.Helper()
	derived := deriveRefusalArms(t)
	identities := make([]ArmIdentity, 0, len(derived))
	for _, arm := range derived {
		identities = append(identities, ArmIdentity{
			File: arm.file, Function: arm.function, Code: arm.code,
			Detail: arm.detail, Occurrence: arm.occurrence,
		})
	}
	return identities
}

// ArmKey renders one identity in the shared witness key space.
func ArmKey(identity ArmIdentity) string {
	return identity.File + "|" + identity.Function + "|" + identity.Code + "|" +
		strconv.Quote(identity.Detail) + "#" + strconv.Itoa(identity.Occurrence)
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

// TestDefensiveBoundsAreExactlyThese pins the set of bound rows: a new
// defensive arm must join this set deliberately, with its bound stated,
// rather than slipping in as an ordinary row whose test cannot reach it.
func TestDefensiveBoundsAreExactlyThese(t *testing.T) {
	t.Parallel()

	want := map[string]string{
		"conformance.go ParseAttachAuthorization CodeMismatch \"document timestamp\" #1":           boundDefensiveReparse,
		"conformance.go ParseAttachAuthorization CodeMismatch \"document timestamp\" #2":           boundDefensiveReparse,
		"conformance.go CheckAttachRequest CodeMismatch \"document timestamp\" #1":                 boundDefensiveReparse,
		"manifest.go ParseEvidence CodeMismatch \"document timestamp\" #1":                         boundDefensiveReparse,
		"manifest.go ParseEvidence CodeMismatch \"document timestamp\" #2":                         boundDefensiveReparse,
		"manifest.go checkEvidenceLiveness CodeMismatch \"document timestamp\" #1":                 boundDefensiveReparse,
		"manifest.go checkEvidenceLiveness CodeMismatch \"document timestamp\" #2":                 boundDefensiveReparse,
		"conformance.go CheckTransition CodeProtocolError \"operation vocabulary\" #1":             boundUnreachableVocabulary,
		"manifest.go decodeCappedValue CodeMismatch \"document syntax\" #3":                        boundDecoderContract,
		"manifest.go decodeCappedValue CodeMismatch \"document syntax\" #6":                        boundDecoderContract,
		"manifest.go objectIdentity CodeMismatch \"document identity\" #1":                         boundCanonicalPlumbing,
		"manifest.go objectIdentity CodeMismatch \"document identity\" #2":                         boundCanonicalPlumbing,
		"manifest.go UnsignedEvidenceBytes CodeIntegrityFailure \"evidence canonical bytes\" #1":   boundCanonicalPlumbing,
		"manifest.go UnsignedEvidenceBytes CodeIntegrityFailure \"evidence canonical bytes\" #2":   boundCanonicalPlumbing,
		"terminalbackend.go parseKind CodeNotFound \"implementation_kind vocabulary\" #1":          boundUnreachableKind,
		"manifest.go refuseIdentityNumbers CodeMismatch \"document member type\" #1":               boundUnreachableNumbers,
		"terminalbackend.go Registration.validate CodeDrift \"executable_digest must be null\" #1": boundUnreachableDigestNull,
		"conformance.go ParseAttachAuthorization CodeMismatch \"document members\" #1":             boundShadowedLookup,
		"manifest.go parseManifestObject CodeMismatch \"document members\" #1":                     boundShadowedLookup,
		"manifest.go ParseProbe CodeMismatch \"document members\" #1":                              boundShadowedLookup,
		"manifest.go digestOrNullMember CodeMismatch \"document members\" #1":                      boundShadowedLookup,
		"manifest.go stringArrayMember CodeMismatch \"document members\" #1":                       boundShadowedLookup,
		"manifest.go checkExtensions CodeMismatch \"document members\" #1":                         boundShadowedLookup,
		"manifest.go parseRealmProvider CodeMismatch \"document members\" #1":                      boundShadowedLookup,
		"manifest.go parseRealmProvider CodeMismatch \"document members\" #2":                      boundShadowedLookup,
		"manifest.go parseRealmLiteral CodeMismatch \"document members\" #1":                       boundShadowedLookup,
	}
	for _, row := range declaredRows() {
		if row.bound == "" {
			continue
		}
		wantBound, pinned := want[describeRow(row)]
		if !pinned {
			t.Errorf("declared row %s claims a bound outside the pinned set; "+
				"a new defensive arm joins deliberately or not at all", describeRow(row))
			continue
		}
		if row.bound != wantBound {
			t.Errorf("declared row %s carries bound %q, want the pinned %q",
				describeRow(row), row.bound, wantBound)
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

// TestCheckTransitionOperationVocabularyIsUnreachable proves the bound on
// CheckTransition's "operation vocabulary" arm: CheckTransition parses
// its operation through ParseOperation first, so the lookupTransition
// miss branch fires only for an operation ParseOperation admits that the
// table does not cover. Every admitted operation resolves, so no input
// reaches the arm; adding an operation to ParseOperation without a table
// row fails here and promotes the row back to a witnessed arm.
func TestCheckTransitionOperationVocabularyIsUnreachable(t *testing.T) {
	t.Parallel()

	// The admitted set is derived from ParseOperation's own switch, never
	// listed: an eleventh admitted operation without a table row fails
	// below and promotes the vocabulary row back to a witnessed arm.
	admitted := parseOperationAdmittedSet(t)
	if len(admitted) != 10 {
		t.Fatalf("ParseOperation admits %d operations, want the ten closed operations; the pin must move with the vocabulary", len(admitted))
	}
	for _, operation := range admitted {
		if _, err := ParseOperation(operation); err != nil {
			t.Errorf("ParseOperation(%q) = %v, want admitted", operation, err)
		}
		if _, known := lookupTransition(Operation(operation)); !known {
			t.Errorf("lookupTransition(%q) misses, so CheckTransition's vocabulary arm is reachable: witness it", operation)
		}
	}
}

// parseOperationAdmittedSet derives the exact string set ParseOperation
// admits from its production switch cases: the case clauses name the
// Operation constants, whose values resolve from the production const
// declarations in the same scan.
func parseOperationAdmittedSet(t *testing.T) []string {
	t.Helper()

	files, _ := invcore.MustScanProduction(t, ".")
	constants := map[string]string{}
	for _, production := range files {
		for _, declaration := range production.Syntax.Decls {
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
					if !ok || literal.Kind != token.STRING {
						continue
					}
					text, err := strconv.Unquote(literal.Value)
					if err != nil {
						t.Fatalf("unquote const %s: %v", name.Name, err)
					}
					constants[name.Name] = text
				}
			}
		}
	}
	var admitted []string
	for _, production := range files {
		if production.Name != "conformance.go" {
			continue
		}
		for _, declaration := range production.Syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != "ParseOperation" {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				clause, ok := node.(*ast.CaseClause)
				if !ok || clause.List == nil {
					return true
				}
				for _, expression := range clause.List {
					identifier, ok := expression.(*ast.Ident)
					if !ok {
						t.Fatalf("ParseOperation admits a non-constant case; the pin cannot name what it cannot read")
					}
					text, known := constants[identifier.Name]
					if !known {
						t.Fatalf("ParseOperation admits unresolved constant %s", identifier.Name)
					}
					admitted = append(admitted, text)
				}
				return true
			})
		}
	}
	if len(admitted) == 0 {
		t.Fatal("derived zero admitted operations; the scanner is broken, not the package")
	}
	return admitted
}

// TestDecodeCappedValueDefensiveSitesAreUnreachable pins the decoder
// contract boundDecoderContract relies on: encoding/json Token() errors
// on non-string object keys and on stray delimiters at value positions
// instead of yielding them, so decodeCappedValue's key-shape (#3) and
// stray-delimiter (#6) branches have no input. A Go release that yields
// either value fails here and promotes both rows back to witnessed arms.
func TestDecodeCappedValueDefensiveSitesAreUnreachable(t *testing.T) {
	t.Parallel()

	nonStringKeys := []string{
		`{1:2}`, `{true:1}`, `{null:1}`, `{[1]:2}`,
	}
	for _, raw := range nonStringKeys {
		decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
		if _, err := decoder.Token(); err != nil {
			t.Fatalf("Token() opener at %q = %v, want the object delimiter", raw, err)
		}
		if token, err := decoder.Token(); err == nil {
			t.Errorf("Token() key at %q = %#v, want an error: a yielded key would reach the defensive branch", raw, token)
		}
		if _, err := decodeCappedValue(json.NewDecoder(bytes.NewReader([]byte(raw))), 0); err == nil {
			t.Errorf("decodeCappedValue(%q) = nil, want a document syntax refusal", raw)
		}
	}
	strayDelimiters := []string{`]`, `}`, `{"a":}`, `[1,]`, `{"a":1,}`, `{"a" "b"}`}
	for _, raw := range strayDelimiters {
		decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
		decoder.Token()
		if _, err := decodeCappedValue(json.NewDecoder(bytes.NewReader([]byte(raw))), 0); err == nil {
			t.Errorf("decodeCappedValue(%q) = nil, want a document syntax refusal", raw)
		}
	}
}

// TestCanonicalPlumbingIsTotalOnDecodedMaps pins the totality
// boundCanonicalPlumbing relies on: objectIdentity succeeds over every
// JSON-decodable shape (nested, empty, unicode, boundary strings) and
// UnsignedEvidenceBytes succeeds over adversarial typed evidence, so the
// four Marshal/JCS error branches have no input in the reachable class.
// A new shape that breaks the plumbing fails here and promotes those
// rows back to witnessed arms.
func TestCanonicalPlumbingIsTotalOnDecodedMaps(t *testing.T) {
	t.Parallel()

	adversarial := []map[string]any{
		{},
		{"empty": map[string]any{}, "list": []any{}, "nothing": nil},
		{"nested": map[string]any{"deeper": map[string]any{"deepest": []any{"a", true, nil}}}},
		{"unicode": "é界🎉", "controls": "a\x00b\nc\td", "empty": ""},
		{"long": strings.Repeat("g", 100000)},
		{"bools": []any{true, false}, "mixed": map[string]any{"x": "y"}},
	}
	for index, object := range adversarial {
		if _, err := objectIdentity(object, "missing-self"); err != nil {
			t.Errorf("objectIdentity(adversarial[%d]) = %v, want success: the plumbing arms are dead on decoded maps", index, err)
		}
	}
	platform, err := scalar.ParsePlatform("linux")
	if err != nil {
		t.Fatalf("ParsePlatform() error = %v", err)
	}
	observed, err := scalar.ParseTimestamp("2026-01-15T12:00:00.000Z")
	if err != nil {
		t.Fatalf("ParseTimestamp() error = %v", err)
	}
	for _, osVersion := range []string{"", strings.Repeat("é", 1000), "ok\xffbad", "controls-\x00\n\t"} {
		evidence := Evidence{
			TerminalBackendID: "com.example.term", ImplementationVersion: "1.2.3",
			ProtocolVersion: "1.0.0", BackendGenerationDigest: "sha256:00",
			Capability: "graceful_stop", Platform: platform, OSVersion: osVersion,
			ConformanceFixtureID: "sha256:00", ObservedAt: observed, ExpiresAt: observed,
			Issuer: "ax_local_probe", IssuerID: "sha256:00",
			Facts: []string{"fixture_passed", "", "é"},
		}
		if _, err := UnsignedEvidenceBytes(evidence); err != nil {
			t.Errorf("UnsignedEvidenceBytes(os_version=%q) = %v, want success", osVersion, err)
		}
	}
}

// TestShadowedLookupsHaveNoInput pins boundShadowedLookup structurally:
// the member lists equal the exact expected slices, and every shadowed
// lookup names a member of its shadowing list. A list that loses a
// member may un-shadow its miss arm into a live one; a lookup of a
// member outside its shadowing list was never shadowed at all. Either
// change fails here and promotes the affected rows back to witnessed
// arms. The shadowing exact CALLS are pinned by the extra-member
// witnesses (one per entry), which fail if an exact call is removed.
func TestShadowedLookupsHaveNoInput(t *testing.T) {
	t.Parallel()

	equalLists := func(name string, got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s = %v, want exactly %v: a lost member may un-shadow a bound miss arm", name, got, want)
		}
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("%s = %v, want exactly %v: a moved member may un-shadow a bound miss arm", name, got, want)
			}
		}
	}
	equalLists("manifestMembers", manifestMembers, []string{
		"schema", "schema_version", "manifest_id", "terminal_backend_id",
		"implementation_version", "protocol_versions", "platforms",
		"implementation_kind", "executable_digest", "static_capability_claims",
		"conformance_fixture_id", "extensions",
	})
	equalLists("probeMembers", probeMembers, []string{
		"schema", "schema_version", "probe_id", "terminal_backend_id",
		"implementation_version", "protocol_version", "implementation_kind",
		"executable_digest", "platform", "os_version", "availability",
		"backend_generation_digest", "capability_claims", "evidence_ids",
		"probed_at", "extensions",
	})
	equalLists("evidenceMembers", evidenceMembers, []string{
		"schema", "schema_version", "evidence_id", "terminal_backend_id",
		"implementation_version", "protocol_version", "backend_generation_digest",
		"capability", "value", "platform", "os_version", "conformance_fixture_id",
		"observed_at", "expires_at", "issuer", "issuer_id",
		"attestation_signature", "facts", "terminal_binding_id", "provider_id",
		"provider_build", "sentinel_result", "provider_auth_smoke_result",
		"extensions",
	})
	equalLists("attachAuthorizationMembers", attachAuthorizationMembers, []string{
		"policy_evidence_id", "authorizing_host_id", "transport",
		"input_authorized", "issued_at", "expires_at",
	})
	// parseClaim's inline list is derived, never listed; the value pin
	// below fails a member lost from the call the same way equalLists
	// fails a package list.
	_ = parseClaimMemberList(t)
	// Every shadowed lookup resolves to its shadowing check through the
	// derived cover analysis below: the (list, member) pairs are
	// computed from the production AST per bound row, never hand-listed,
	// and every "document members" miss classifies covered (bound) or
	// uncovered (witnessed) with its row kind agreeing in both
	// directions.
	assertShadowedLookupsAreCovered(t)
}

// shadowMiss is one derived `mismatchf("document members")` site: the
// production file, the enclosing function, the construction line, the
// map variable missed on, and the looked-up members determinable at the
// site itself (fixed literals like object["extensions"]; empty for a
// parameter lookup like object[name], whose members come from the
// helper's call sites).
type shadowMiss struct {
	file     string
	function string
	line     int
	position token.Pos
	mapVar   string
	fixed    []string
	// param reports a parameter lookup (object[name]): the members come
	// from the helper's call sites, and every name argument must be a
	// string literal. A fixed lookup (object["lit"]) names its members
	// at the site; a lookup with no map index at all (the length check)
	// has no members and is uncovered by construction.
	param bool
}

// shadowExact is one derived checkExactMembers call: the enclosing
// function, the checked map variable, the call position, and the exact
// member list (a package-level var or an inline literal, resolved to
// values; anything else fails closed).
type shadowExact struct {
	function string
	mapVar   string
	position token.Pos
	members  []string
	source   string
}

// assertShadowedLookupsAreCovered derives every "document members" miss
// from the production AST and requires each to classify covered
// (unreachable: bound) or uncovered (reachable: witnessed), with the
// declared row kind agreeing in both directions. Covered means every
// looked-up member passes a checkExactMembers over a list containing it
// on every production path to the miss: a same-function preceding check
// on the same map variable, or — for member helpers without one — every
// production caller passing a covered map, recursively. A miss with no
// determinable members, a non-literal helper argument, an unresolvable
// list, or an unresolvable caller is uncovered-or-fatal, never silently
// covered: the analysis fails toward witnessed, and a bound row that
// cannot prove cover reddens here instead of hiding behind a hand list.
func assertShadowedLookupsAreCovered(t *testing.T) {
	t.Helper()

	files, fileSet := invcore.MustScanProduction(t, ".")
	misses := shadowMissSites(t, files, fileSet)
	exacts := shadowExactChecks(t, files, fileSet)
	calls := shadowHelperCalls(files)

	// Join misses to derived arms by (file, function, line): every miss
	// must be a derived arm and vice versa. A miss the analysis cannot
	// see is a bound without a pin; an arm the analysis cannot see is a
	// pin that proves nothing.
	derivedByKey := map[string]refusalArm{}
	for _, arm := range deriveRefusalArms(t) {
		if arm.detail != "document members" {
			continue
		}
		derivedByKey[arm.file+"|"+arm.function+":"+strconv.Itoa(arm.line)] = arm
	}
	missByKey := map[string]shadowMiss{}
	for _, miss := range misses {
		key := miss.file + "|" + miss.function + ":" + strconv.Itoa(miss.line)
		if _, duplicate := missByKey[key]; duplicate {
			t.Errorf("shadow analysis derives %s twice", key)
			continue
		}
		missByKey[key] = miss
		if _, ok := derivedByKey[key]; !ok {
			t.Errorf("shadow miss %s names no derived refusal arm; the cover analysis sees a site the inventory does not", key)
		}
	}
	for key := range derivedByKey {
		if _, ok := missByKey[key]; !ok {
			t.Errorf("derived refusal arm %s names no shadow miss; the inventory sees a site the cover analysis does not", key)
		}
	}

	// Row linkage, both directions: covered misses carry bound rows,
	// uncovered misses carry witnessed rows.
	rows := declaredRows()
	rowByKey := map[string]declaredRefusalArm{}
	for _, row := range rows {
		rowByKey[describeRow(row)] = row
	}
	covered, uncovered := 0, 0
	for key, miss := range missByKey {
		arm, ok := derivedByKey[key]
		if !ok {
			continue
		}
		row, known := rowByKey[describeArm(arm)]
		if !known {
			t.Errorf("shadow miss %s resolves to no declared row", key)
			continue
		}
		lookups := shadowMissLookups(t, miss, calls)
		isCovered := len(lookups) > 0 && shadowLookupsCovered(exacts, calls, lookups)
		if isCovered {
			covered++
			if row.bound != boundShadowedLookup {
				t.Errorf("shadow miss %s is covered but its row carries bound %q, want %q",
					key, row.bound, boundShadowedLookup)
			}
		} else {
			uncovered++
			if row.bound != "" {
				t.Errorf("shadow miss %s is uncovered but its row claims bound %q: the shadowing does not hold, witness the arm",
					key, row.bound)
			}
		}
	}
	boundRows := 0
	for _, row := range rows {
		if row.bound == boundShadowedLookup {
			boundRows++
		}
	}
	if covered != boundRows {
		t.Errorf("cover analysis covers %d misses, want exactly the %d boundShadowedLookup rows", covered, boundRows)
	}
	t.Logf("shadowed lookups: %d covered (bound), %d uncovered (witnessed)", covered, uncovered)
}

// shadowMissSites derives every `mismatchf("document members")` site:
// its file, enclosing function (receiver-qualified for methods), call
// line, missed map variable, and fixed looked-up literals. The miss
// shape is `v, known := object["lit"]` (or object[name]) with the
// mismatchf call inside the `if !known` body: the fixed literals are the
// string indices in that if statement's init and condition. A miss whose
// lookup cannot be read this way fails closed: an unreadable shadowing
// is not a shadowing.
func shadowMissSites(t *testing.T, files []invcore.ProductionFile, fileSet *token.FileSet) []shadowMiss {
	t.Helper()

	var misses []shadowMiss
	for _, production := range files {
		name, syntax := production.Name, production.Syntax
		for _, declaration := range syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			functionName := function.Name.Name
			if function.Recv != nil && len(function.Recv.List) > 0 {
				functionName = receiverTypeName(function.Recv.List[0].Type) + "." + functionName
			}
			// Block-level walk, not a bare Inspect: the lookup
			// (`v, known := object["lit"]`) is the statement
			// preceding the `if !known`, or the if's own init, and
			// only the sibling position identifies it. Every
			// *ast.BlockStmt in the body is visited, so nested
			// misses classify like top-level ones.
			ast.Inspect(function.Body, func(node ast.Node) bool {
				block, ok := node.(*ast.BlockStmt)
				if !ok {
					return true
				}
				for index, statement := range block.List {
					branch, ok := statement.(*ast.IfStmt)
					if !ok {
						continue
					}
					callPos, refuses := shadowBranchRefusalCall(branch)
					if !refuses {
						continue
					}
					var prev ast.Stmt
					if index > 0 {
						prev = block.List[index-1]
					}
					// A guard with no readable map index (the length
					// check, an exotic lookup, a lookup further back
					// than the preceding sibling) carries no members
					// and classifies uncovered: an unreadable
					// shadowing is not a shadowing, and the row must
					// witness the arm.
					mapVar, fixed, param := shadowLookupAround(branch.Init, prev)
					misses = append(misses, shadowMiss{
						file: name, function: functionName,
						line:     fileSet.Position(callPos).Line,
						position: callPos,
						mapVar:   mapVar, fixed: fixed, param: param,
					})
				}
				return true
			})
		}
	}
	if len(misses) == 0 {
		t.Fatal("derived zero document-members misses; the scanner is broken, not the package")
	}
	return misses
}

// shadowLookupAround reads the lookup guarding a miss: the if's own
// init (`if _, known := object["lit"]; !known`), else the preceding
// sibling statement (`v, known := object["lit"]` on the line above).
func shadowLookupAround(init ast.Stmt, prev ast.Stmt) (string, []string, bool) {
	if init != nil {
		if mapVar, fixed, param := shadowLookupIn(init); mapVar != "" {
			return mapVar, fixed, param
		}
	}
	if prev != nil {
		return shadowLookupIn(prev)
	}
	return "", nil, false
}

// shadowLookupIn reads one statement's map index expressions into the
// missed variable, fixed literals, and parameter flag. Statements with
// no map index, or indices over disagreeing variables, yield an empty
// variable: uncovered, never an error here (classification, not
// enumeration, decides).
func shadowLookupIn(statement ast.Stmt) (string, []string, bool) {
	var mapVar string
	var fixed []string
	param := false
	ast.Inspect(statement, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		index, ok := expression.(*ast.IndexExpr)
		if !ok {
			return true
		}
		target, ok := index.X.(*ast.Ident)
		if !ok {
			return true
		}
		if mapVar == "" {
			mapVar = target.Name
		} else if mapVar != target.Name {
			mapVar = "\x00"
			return true
		}
		literal, ok := index.Index.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			param = true
			return true
		}
		member, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		fixed = append(fixed, member)
		return true
	})
	if mapVar == "\x00" {
		return "", nil, false
	}
	return mapVar, fixed, param
}

// shadowBranchRefusalCall returns the position of the
// `mismatchf("document members")` call in the if body, if one exists.
func shadowBranchRefusalCall(branch *ast.IfStmt) (token.Pos, bool) {
	var callPos token.Pos
	ast.Inspect(branch.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok || identifier.Name != "mismatchf" || len(call.Args) != 1 {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		detail, err := strconv.Unquote(literal.Value)
		if err != nil || detail != "document members" {
			return true
		}
		callPos = call.Pos()
		return false
	})
	return callPos, callPos.IsValid()
}

// shadowExactChecks derives every checkExactMembers call: its enclosing
// function, checked map variable (a bare identifier, never an
// expression), call position, and resolved member list. The list is a
// package-level var literal or an inline slice literal; anything else
// fails closed.
func shadowExactChecks(t *testing.T, files []invcore.ProductionFile, fileSet *token.FileSet) []shadowExact {
	t.Helper()

	lists := map[string][]string{}
	for _, production := range files {
		for _, declaration := range production.Syntax.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range value.Names {
					if len(value.Values) == 0 {
						continue
					}
					literal, ok := value.Values[0].(*ast.CompositeLit)
					if !ok {
						continue
					}
					var members []string
					for _, element := range literal.Elts {
						basic, ok := element.(*ast.BasicLit)
						if !ok || basic.Kind != token.STRING {
							members = nil
							break
						}
						member, err := strconv.Unquote(basic.Value)
						if err != nil {
							members = nil
							break
						}
						members = append(members, member)
					}
					if members != nil {
						lists[name.Name] = members
					}
				}
			}
		}
	}
	var exacts []shadowExact
	for _, production := range files {
		name, syntax := production.Name, production.Syntax
		for _, declaration := range syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			functionName := function.Name.Name
			if function.Recv != nil && len(function.Recv.List) > 0 {
				functionName = receiverTypeName(function.Recv.List[0].Type) + "." + functionName
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				if !ok || identifier.Name != "checkExactMembers" || len(call.Args) != 2 {
					return true
				}
				target, ok := call.Args[0].(*ast.Ident)
				if !ok {
					t.Errorf("%s (%s:%d): checkExactMembers over a non-identifier map; "+
						"the cover analysis cannot thread it", functionName, name, fileSet.Position(call.Pos()).Line)
					return true
				}
				var members []string
				switch list := call.Args[1].(type) {
				case *ast.Ident:
					resolved, known := lists[list.Name]
					if !known {
						t.Errorf("%s (%s:%d): checkExactMembers over unresolvable list %q",
							functionName, name, fileSet.Position(call.Pos()).Line, list.Name)
						return true
					}
					members = resolved
				case *ast.CompositeLit:
					for _, element := range list.Elts {
						basic, ok := element.(*ast.BasicLit)
						if !ok || basic.Kind != token.STRING {
							t.Errorf("%s (%s:%d): checkExactMembers over a non-literal inline list",
								functionName, name, fileSet.Position(call.Pos()).Line)
							return true
						}
						member, err := strconv.Unquote(basic.Value)
						if err != nil {
							t.Errorf("%s (%s:%d): unquote inline list member: %v",
								functionName, name, fileSet.Position(call.Pos()).Line, err)
							return true
						}
						members = append(members, member)
					}
				default:
					t.Errorf("%s (%s:%d): checkExactMembers over an unreadable list shape",
						functionName, name, fileSet.Position(call.Pos()).Line)
					return true
				}
				exacts = append(exacts, shadowExact{
					function: functionName, mapVar: target.Name,
					position: call.Pos(), members: members,
					source: name + ":" + strconv.Itoa(fileSet.Position(call.Pos()).Line),
				})
				return true
			})
		}
	}
	if len(exacts) == 0 {
		t.Fatal("derived zero exact-member checks; the scanner is broken, not the package")
	}
	return exacts
}

// shadowCall records one production call: the caller, the passed map
// variable (empty when the first argument is not a bare identifier),
// the call position, and the raw arguments (a parameter helper's member
// name is args[1]).
type shadowCall struct {
	caller   string
	mapVar   string
	position token.Pos
	args     []ast.Expr
}

// shadowHelperCalls indexes every production Ident call by callee name.
// Selector calls, method values, and func variables resolve to no callee
// here: a helper reached that way is uncovered, never silently covered.
func shadowHelperCalls(files []invcore.ProductionFile) map[string][]shadowCall {
	calls := map[string][]shadowCall{}
	for _, production := range files {
		for _, declaration := range production.Syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			functionName := function.Name.Name
			if function.Recv != nil && len(function.Recv.List) > 0 {
				functionName = receiverTypeName(function.Recv.List[0].Type) + "." + functionName
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				entry := shadowCall{caller: functionName, position: call.Pos(), args: call.Args}
				if len(call.Args) > 0 {
					if target, ok := call.Args[0].(*ast.Ident); ok {
						entry.mapVar = target.Name
					}
				}
				calls[identifier.Name] = append(calls[identifier.Name], entry)
				return true
			})
		}
	}
	return calls
}

// shadowLookup is one looked-up member at the position that must be
// covered: a fixed literal at its miss site, or a parameter-helper name
// literal at the call site passing it. Cover is per lookup, not per
// member: "executable_digest" is shadowed on the manifest path and never
// flows down the evidence path, so requiring it covered there would fail
// a sound shadowing.
type shadowLookup struct {
	member   string
	function string
	variable string
	position token.Pos
}

// shadowMissLookups resolves the covered lookups of one miss: its fixed
// literals at the miss position, plus — for a parameter lookup — every
// (call site, name literal) pair. A parameter lookup with a non-literal
// name argument fails the test: an unreadable member cannot prove
// cover. The exact checks' own misses resolve to nothing: they range
// over the list parameter, so no call-site literal names them, and they
// are uncovered by construction.
func shadowMissLookups(t *testing.T, miss shadowMiss, calls map[string][]shadowCall) []shadowLookup {
	t.Helper()

	var lookups []shadowLookup
	for _, member := range miss.fixed {
		lookups = append(lookups, shadowLookup{
			member: member, function: miss.function,
			variable: miss.mapVar, position: miss.position,
		})
	}
	if !miss.param || miss.function == "checkExactMembers" {
		return lookups
	}
	for _, call := range calls[miss.function] {
		if len(call.args) < 2 {
			t.Errorf("shadow helper %s called with %d arguments at %s; "+
				"the member-name argument is missing", miss.function, len(call.args), call.caller)
			continue
		}
		literal, ok := call.args[1].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			t.Errorf("shadow helper %s takes a non-literal member name at %s; "+
				"an unreadable member cannot prove cover", miss.function, call.caller)
			continue
		}
		member, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Errorf("unquote member name at %s: %v", call.caller, err)
			continue
		}
		lookups = append(lookups, shadowLookup{
			member: member, function: call.caller,
			variable: call.mapVar, position: call.position,
		})
	}
	return lookups
}

// shadowLookupsCovered reports whether every lookup is covered: a
// same-function preceding exact check on the same map variable, or —
// for helpers without one — coverage through every production caller,
// recursively.
func shadowLookupsCovered(exacts []shadowExact, calls map[string][]shadowCall, lookups []shadowLookup) bool {
	visiting := map[string]bool{}
	for _, lookup := range lookups {
		if !shadowMemberCovered(exacts, calls, lookup.function, lookup.variable, lookup.position, lookup.member, visiting) {
			return false
		}
	}
	return true
}

// shadowMemberCovered reports whether member passes an exact check over
// a list containing it on every production path reaching (function,
// variable) at position: a preceding same-function check on the same
// variable, else coverage through every production caller passing a
// covered variable, recursively. Anything unresolvable is uncovered:
// the analysis fails toward witnessed, never toward bound.
func shadowMemberCovered(exacts []shadowExact, calls map[string][]shadowCall, function, variable string, position token.Pos, member string, visiting map[string]bool) bool {
	for _, exact := range exacts {
		if exact.function != function || exact.mapVar != variable {
			continue
		}
		if !(exact.position < position) {
			continue
		}
		for _, candidate := range exact.members {
			if candidate == member {
				return true
			}
		}
	}
	if visiting[function] {
		return false
	}
	visiting[function] = true
	defer delete(visiting, function)
	callers, ok := calls[function]
	if !ok || len(callers) == 0 {
		return false
	}
	for _, call := range callers {
		if call.mapVar == "" {
			return false
		}
		if !shadowMemberCovered(exacts, calls, call.caller, call.mapVar, call.position, member, visiting) {
			return false
		}
	}
	return true
}

// parseClaimMemberList derives parseClaim's inline exact-member list from
// the production AST: the slice literal passed to checkExactMembers
// inside parseClaim. A member removed from the call may un-shadow the
// claim extractors into live miss arms.
func parseClaimMemberList(t *testing.T) []string {
	t.Helper()

	files, _ := invcore.MustScanProduction(t, ".")
	for _, production := range files {
		if production.Name != "manifest.go" {
			continue
		}
		for _, declaration := range production.Syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != "parseClaim" {
				continue
			}
			var members []string
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				if !ok || identifier.Name != "checkExactMembers" || len(call.Args) != 2 {
					return true
				}
				literal, ok := call.Args[1].(*ast.CompositeLit)
				if !ok {
					t.Fatalf("parseClaim checks members against a non-literal list; the pin cannot name it")
				}
				for _, element := range literal.Elts {
					basic, ok := element.(*ast.BasicLit)
					if !ok || basic.Kind != token.STRING {
						t.Fatalf("parseClaim member list holds a non-literal; the pin cannot name it")
					}
					text, err := strconv.Unquote(basic.Value)
					if err != nil {
						t.Fatalf("unquote claim member: %v", err)
					}
					members = append(members, text)
				}
				return true
			})
			if len(members) == 0 {
				t.Fatal("derived zero parseClaim members; the scanner is broken, not the package")
			}
			want := []string{"capability", "origin", "value", "generation_variable", "dependent_operations", "evidence_requirements"}
			if len(members) != len(want) {
				t.Fatalf("parseClaim checks %v, want exactly %v: a lost member may un-shadow a bound miss arm", members, want)
			}
			for index := range want {
				if members[index] != want[index] {
					t.Fatalf("parseClaim checks %v, want exactly %v: a moved member may un-shadow a bound miss arm", members, want)
				}
			}
			return members
		}
	}
	t.Fatal("production func parseClaim not found in manifest.go")
	return nil
}

// TestPlatformFunnelDetailsAreExactlyThese derives the error set of
// validatePlatforms from the production AST and requires it to equal the
// funneled set: a fourth platform error widens the funnel this test pins.
func TestPlatformFunnelDetailsAreExactlyThese(t *testing.T) {
	t.Parallel()

	// Selection and fail-closed parsing are the shared core's; the
	// relative path relies on the test working directory, as before.
	source, err := os.ReadFile("terminalbackend.go")
	if err != nil {
		t.Fatalf("parse terminalbackend.go: %v", err)
	}
	file, _, failure := invcore.ParseBytes("terminalbackend.go", source, 0)
	if failure != "" {
		t.Fatalf("parse terminalbackend.go: %s", failure)
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

	// The alias direction is the core's over the same spec, so an
	// import-aliased (`errs "errors"`) or var-bound construction fails
	// there; the allowlist below attributes every watched direct call —
	// import aliases resolved — to its enclosing function.
	spec := plainErrorSpec()
	files, fileSet := invcore.MustScanProduction(t, ".")
	for _, production := range files {
		file, name := production.Syntax, production.Name
		invcore.MustAuditConstructorReferences(t, file, fileSet, name, spec)
		watched := invcore.WatchedCallPositions(file, spec)
		check := func(functionName string, root ast.Node) {
			ast.Inspect(root, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				position, ok := invcore.FunPosition(call)
				if !ok || !watched[position] {
					return true
				}
				if !plainErrorFunctions[functionName] {
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

// plainErrorSpec watches the two recognised plain-error spellings by
// import path: errors.New and fmt.Errorf (any Errorf-prefixed
// constructor). Custom error types outside these spellings are a stated
// bound above, not covered here.
func plainErrorSpec() invcore.ConstructorSpec {
	return invcore.ConstructorSpec{
		Qualified: map[string]map[string]bool{
			"errors": {"New": true},
			"fmt":    {"Errorf*": true},
		},
	}
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
