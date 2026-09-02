package canonicaljson

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file derives the subject of two proof gates that a deletion-only or
// operator-rewrite mutation sweep structurally cannot reach.
//
// The failures they exist to prevent, both found by review on this leaf after a
// 135-mutant and then a 26-mutant sweep each reported a strong kill rate:
//
//   - A coupling written as one boolean comparison — `requiresError !=
//     errorPresent` — proven in ONE lexical direction only. Narrowing it to
//     `requiresError && !errorPresent` left the whole configured gate set green
//     while ValidateObservationEvent attested a `success` event carrying a
//     non-null error_code. An operator-rewrite grammar cannot express that
//     mutant: there is no operator to rewrite, only a boolean identity to split.
//
//   - An integer comparison against a literal — `epoch != 1` — proven only at
//     epoch 4. Narrowing it to `epoch >= 3` left the gate set green while
//     CalculateObjectIdentity attested an epoch-2 lease with a null
//     predecessor. `epoch != 1` narrowed to `epoch > 1` is an EQUIVALENT
//     mutant, so no deletion or operator sweep could ever have surfaced this;
//     only a case at the literal's own boundary can.
//
// Both subjects are derived from the production sources rather than listed, so
// a new coupling or a new literal comparison becomes an obligation without
// anyone editing a table, and moving a literal reddens the gate before any case
// runs because the derived key carries the literal.

// ---------------------------------------------------------------------------
// Presence couplings
// ---------------------------------------------------------------------------

// presenceDirection names one of the two single-sided violations of a coupling.
// Which pair applies is derived from the operator, never chosen:
//
//	a != b requires the two sides to agree, so its violations are
//	  (a true, b false) and (a false, b true).
//	a == b requires the two sides to differ, so its violations are
//	  (both true) and (both false).
type presenceDirection string

const (
	presenceLeftOnly  presenceDirection = "left-only"
	presenceRightOnly presenceDirection = "right-only"
	presenceBoth      presenceDirection = "both"
	presenceNeither   presenceDirection = "neither"
)

// presenceObligation identifies one direction of one derived coupling site.
type presenceObligation struct {
	key       string
	direction presenceDirection
}

func (obligation presenceObligation) String() string {
	return obligation.key + " [" + string(obligation.direction) + "]"
}

// derivePresenceCouplingObligations returns every derived obligation with the
// number of distinct production sites that declare it.
//
// The scan is package-wide. A boolean-valued equality is rare enough — seven
// sites across three production files — that scoping the gate to this leaf's
// own file would leave a known instance of the same class unproven in a file
// this package also ships.
func derivePresenceCouplingObligations(t *testing.T) map[presenceObligation]int {
	t.Helper()

	obligations := make(map[presenceObligation]int)
	for _, site := range derivePresenceCouplingSites(t) {
		left, right := presenceLeftOnly, presenceRightOnly
		if site.op == token.EQL {
			left, right = presenceBoth, presenceNeither
		}
		obligations[presenceObligation{key: site.key, direction: left}]++
		obligations[presenceObligation{key: site.key, direction: right}]++
	}
	if len(obligations) == 0 {
		t.Fatal("derived zero presence-coupling obligations; the scanner is broken, not the package")
	}
	return obligations
}

type presenceCouplingSite struct {
	key string
	op  token.Token
}

// derivePresenceCouplingSites walks every production source and returns one
// entry per `==`/`!=` whose BOTH operands are boolean-valued.
//
// Boolean-valued is decided from the source, not from a name convention: a
// negation, a nested comparison, a short-circuit, or an identifier whose
// enclosing function binds it from a package function whose result at that
// position is declared `bool`.
func derivePresenceCouplingSites(t *testing.T) []presenceCouplingSite {
	t.Helper()

	var sites []presenceCouplingSite
	forEachProductionComparison(t, func(context comparisonContext) {
		if context.expression.Op != token.EQL && context.expression.Op != token.NEQ {
			return
		}
		if !context.isBoolean(context.expression.X) || !context.isBoolean(context.expression.Y) {
			return
		}
		sites = append(sites, presenceCouplingSite{key: context.key(), op: context.expression.Op})
	})
	sort.Slice(sites, func(i, j int) bool { return sites[i].key < sites[j].key })
	return sites
}

// ---------------------------------------------------------------------------
// Literal boundaries
// ---------------------------------------------------------------------------

// literalBoundaryObligation requires a case driving the production entry with
// the compared integer at exactly this value. The key carries the literal, so
// moving the literal reddens the coverage assertion before any case runs.
type literalBoundaryObligation struct {
	key   string
	value int
}

func (obligation literalBoundaryObligation) String() string {
	return fmt.Sprintf("%s [at %d]", obligation.key, obligation.value)
}

// literalBoundaryScope is the production file whose integer literal
// comparisons this leaf owns and proves.
//
// Scope, stated rather than implied: `canonical.go` and `closed_shapes.go` are
// the preceding leaves' deliverables. Their literal comparisons were measured
// and disclosed in TASK-260830-1tax26_rev4-preceding-leaf-survivors.md and are
// owned by the sibling conformance-test leaf TASK-260830-uqnwmi. Widening this
// gate to them would claim proofs this leaf does not ship.
const literalBoundaryScope = "core_records.go"

// deriveLiteralBoundaryObligations returns every derived obligation with the
// number of distinct production sites that declare it.
//
// Obligations are the values at which the derived comparison flips, which is
// read off the operator rather than chosen:
//
//	x >  K  flips between K and K+1
//	x <= K  flips between K and K+1
//	x >= K  flips between K-1 and K
//	x <  K  flips between K-1 and K
//	x != K  and x == K single out K, so both neighbours are obliged
//
// A value below zero is never obliged for a length, a range index, or an
// unsigned local, because it is not in the compared expression's domain.
func deriveLiteralBoundaryObligations(t *testing.T) map[literalBoundaryObligation]int {
	t.Helper()

	obligations := make(map[literalBoundaryObligation]int)
	for _, site := range deriveLiteralBoundarySites(t) {
		for _, value := range site.values {
			obligations[literalBoundaryObligation{key: site.key, value: value}]++
		}
	}
	if len(obligations) == 0 {
		t.Fatal("derived zero literal-boundary obligations; the scanner is broken, not the package")
	}
	return obligations
}

type literalBoundarySite struct {
	key     string
	literal int
	values  []int
}

// deriveLiteralBoundarySites walks the owned production file and returns one
// entry per comparison between an integer-valued expression and an integer
// literal, excluding the registry-completeness comparisons that run at package
// initialization against the pinned catalog rather than against a candidate.
func deriveLiteralBoundarySites(t *testing.T) []literalBoundarySite {
	t.Helper()

	var sites []literalBoundarySite
	forEachProductionComparison(t, func(context comparisonContext) {
		if context.file != literalBoundaryScope {
			return
		}
		if _, exempt := initializationTimeComparisonFunctions[context.function]; exempt {
			return
		}
		literal, compared, ok := context.integerLiteralComparison()
		if !ok {
			return
		}
		values := boundaryValues(context.expression.Op, literal, context.literalOnLeft)
		if context.isNonNegative(compared) {
			values = filterNonNegative(values)
		}
		if len(values) == 0 {
			return
		}
		sites = append(sites, literalBoundarySite{key: context.key(), literal: literal, values: values})
	})
	sort.Slice(sites, func(i, j int) bool { return sites[i].key < sites[j].key })
	return sites
}

// initializationTimeComparisonFunctions names the production functions whose
// integer comparisons cannot be driven with a candidate value because they run
// once at package initialization over the pinned catalog and the registry built
// from it. The set is asserted exactly by
// TestInitializationTimeComparisonExemptionsAreRealAndPanicking, so an exemption
// cannot be invented to silence the gate.
var initializationTimeComparisonFunctions = map[string]string{
	"validateSessionEventPayloadShapeCompleteness": "runs from mustBuildSessionEventPayloadShapes at package initialization over catalog.Current(); its comparisons count registry shapes, not candidate members, and any mismatch panics before any entry point exists",
}

func boundaryValues(op token.Token, literal int, literalOnLeft bool) []int {
	if literalOnLeft {
		switch op {
		case token.LSS:
			op = token.GTR
		case token.LEQ:
			op = token.GEQ
		case token.GTR:
			op = token.LSS
		case token.GEQ:
			op = token.LEQ
		}
	}
	switch op {
	case token.GTR, token.LEQ:
		return []int{literal, literal + 1}
	case token.GEQ, token.LSS:
		return []int{literal - 1, literal}
	case token.EQL, token.NEQ:
		return []int{literal - 1, literal, literal + 1}
	}
	return nil
}

func filterNonNegative(values []int) []int {
	kept := values[:0:0]
	for _, value := range values {
		if value >= 0 {
			kept = append(kept, value)
		}
	}
	return kept
}

// ---------------------------------------------------------------------------
// Shared comparison walk
// ---------------------------------------------------------------------------

type comparisonContext struct {
	file          string
	function      string
	expression    *ast.BinaryExpr
	locals        map[string]string
	nonNegative   map[string]bool
	constants     map[string]string
	literalOnLeft bool
}

func (context comparisonContext) key() string {
	return fmt.Sprintf("%s|%s|%s", context.file, context.function, renderComparison(context.expression))
}

func (context comparisonContext) isBoolean(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.ParenExpr:
		return context.isBoolean(value.X)
	case *ast.UnaryExpr:
		return value.Op == token.NOT
	case *ast.BinaryExpr:
		switch value.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
			return true
		}
		return false
	case *ast.Ident:
		return context.locals[value.Name] == "bool"
	}
	return false
}

// integerLiteralComparison reports whether exactly one side is an integer
// literal, spelled out or as a package integer constant.
//
// The other side needs no inference and gets none: Go refuses to compile a
// comparison between an untyped integer constant and a non-numeric operand, so
// the compiler has already established that the other side is integer-valued.
// A heuristic here could only be wrong in the direction of dropping a site.
func (context *comparisonContext) integerLiteralComparison() (literal int, compared ast.Expr, ok bool) {
	left, leftLiteral := context.integerLiteral(context.expression.X)
	right, rightLiteral := context.integerLiteral(context.expression.Y)
	switch {
	case leftLiteral && !rightLiteral:
		context.literalOnLeft = true
		return left, context.expression.Y, true
	case rightLiteral && !leftLiteral:
		return right, context.expression.X, true
	}
	return 0, nil, false
}

func (context comparisonContext) integerLiteral(expression ast.Expr) (int, bool) {
	switch value := expression.(type) {
	case *ast.ParenExpr:
		return context.integerLiteral(value.X)
	case *ast.BasicLit:
		if value.Kind != token.INT {
			return 0, false
		}
		parsed, err := strconv.Atoi(strings.ReplaceAll(value.Value, "_", ""))
		return parsed, err == nil
	case *ast.Ident:
		text, known := context.constants[value.Name]
		if !known {
			return 0, false
		}
		parsed, err := strconv.Atoi(text)
		return parsed, err == nil
	}
	return 0, false
}

// isNonNegative reports whether the compared expression cannot take a negative
// value: a len(), a range index, or an unsigned local.
func (context comparisonContext) isNonNegative(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.ParenExpr:
		return context.isNonNegative(value.X)
	case *ast.CallExpr:
		identifier, ok := value.Fun.(*ast.Ident)
		return ok && identifier.Name == "len"
	case *ast.Ident:
		if context.nonNegative[value.Name] {
			return true
		}
		switch context.locals[value.Name] {
		case "uint", "uint64":
			return true
		}
	}
	return false
}

// forEachProductionComparison visits every comparison expression in the package
// production sources with the enclosing function's local types resolved.
func forEachProductionComparison(t *testing.T, visit func(comparisonContext)) {
	t.Helper()

	_, paths := packageProductionFiles(t)
	files := make([]*ast.File, 0, len(paths))
	bases := make([]string, 0, len(paths))
	for _, path := range paths {
		files = append(files, parseProductionFile(t, path))
		bases = append(bases, filepath.Base(path))
	}
	constants := derivePackageIntegerConstants(files)
	results := derivePackageFunctionResults(files)

	for index, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			locals, nonNegative := deriveLocalTypes(function, results)
			ast.Inspect(function.Body, func(node ast.Node) bool {
				expression, ok := node.(*ast.BinaryExpr)
				if !ok {
					return true
				}
				switch expression.Op {
				case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
				default:
					return true
				}
				visit(comparisonContext{
					file:        bases[index],
					function:    function.Name.Name,
					expression:  expression,
					locals:      locals,
					nonNegative: nonNegative,
					constants:   constants,
				})
				return true
			})
		}
	}
}

// derivePackageFunctionResults maps every package function to the declared type
// name of each of its results, so an identifier bound from a call can be typed
// without a type checker.
func derivePackageFunctionResults(files []*ast.File) map[string][]string {
	results := make(map[string][]string)
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			var names []string
			if function.Type.Results != nil {
				for _, field := range function.Type.Results.List {
					name := "?"
					if identifier, ok := field.Type.(*ast.Ident); ok {
						name = identifier.Name
					}
					count := max(len(field.Names), 1)
					for range count {
						names = append(names, name)
					}
				}
			}
			results[function.Name.Name] = names
		}
	}
	return results
}

// deriveLocalTypes resolves the declared type of every local an enclosing
// function binds, plus the subset that cannot be negative.
func deriveLocalTypes(function *ast.FuncDecl, results map[string][]string) (map[string]string, map[string]bool) {
	locals := make(map[string]string)
	nonNegative := make(map[string]bool)
	for _, parameter := range flatParameters(function) {
		if parameter.name != "" {
			locals[parameter.name] = parameter.typeName
		}
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch statement := node.(type) {
		case *ast.RangeStmt:
			if key, ok := statement.Key.(*ast.Ident); ok && key.Name != "_" {
				locals[key.Name] = "int"
				nonNegative[key.Name] = true
			}
		case *ast.AssignStmt:
			assignLocalTypes(statement, results, locals)
		}
		return true
	})
	return locals, nonNegative
}

func assignLocalTypes(statement *ast.AssignStmt, results map[string][]string, locals map[string]string) {
	if len(statement.Rhs) == 1 {
		if call, ok := statement.Rhs[0].(*ast.CallExpr); ok {
			if identifier, ok := call.Fun.(*ast.Ident); ok {
				if identifier.Name == "len" && len(statement.Lhs) == 1 {
					bindLocal(statement.Lhs[0], "int", locals)
					return
				}
				if declared, known := results[identifier.Name]; known && len(declared) == len(statement.Lhs) {
					for index, target := range statement.Lhs {
						bindLocal(target, declared[index], locals)
					}
					return
				}
			}
			return
		}
		if _, ok := statement.Rhs[0].(*ast.TypeAssertExpr); ok && len(statement.Lhs) == 2 {
			bindLocal(statement.Lhs[1], "bool", locals)
			return
		}
		if _, ok := statement.Rhs[0].(*ast.IndexExpr); ok && len(statement.Lhs) == 2 {
			bindLocal(statement.Lhs[1], "bool", locals)
			return
		}
	}
	if len(statement.Lhs) != len(statement.Rhs) {
		return
	}
	for index, target := range statement.Lhs {
		bindLocal(target, expressionTypeName(statement.Rhs[index], locals), locals)
	}
}

func expressionTypeName(expression ast.Expr, locals map[string]string) string {
	switch value := expression.(type) {
	case *ast.ParenExpr:
		return expressionTypeName(value.X, locals)
	case *ast.UnaryExpr:
		if value.Op == token.NOT {
			return "bool"
		}
	case *ast.BinaryExpr:
		switch value.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
			return "bool"
		}
	case *ast.BasicLit:
		if value.Kind == token.INT {
			return "int"
		}
		if value.Kind == token.STRING {
			return "string"
		}
	case *ast.Ident:
		if value.Name == "true" || value.Name == "false" {
			return "bool"
		}
		return locals[value.Name]
	case *ast.CallExpr:
		if identifier, ok := value.Fun.(*ast.Ident); ok && identifier.Name == "len" {
			return "int"
		}
	}
	return ""
}

func bindLocal(target ast.Expr, typeName string, locals map[string]string) {
	identifier, ok := target.(*ast.Ident)
	if !ok || identifier.Name == "_" || typeName == "" {
		return
	}
	locals[identifier.Name] = typeName
}

// renderComparison prints a comparison back in source order so a derived key is
// readable in a failure and stable across unrelated edits.
func renderComparison(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.BasicLit:
		return strings.ReplaceAll(value.Value, "_", "")
	case *ast.ParenExpr:
		return "(" + renderComparison(value.X) + ")"
	case *ast.UnaryExpr:
		return value.Op.String() + renderComparison(value.X)
	case *ast.BinaryExpr:
		return renderComparison(value.X) + " " + value.Op.String() + " " + renderComparison(value.Y)
	case *ast.SelectorExpr:
		return renderComparison(value.X) + "." + value.Sel.Name
	case *ast.IndexExpr:
		return renderComparison(value.X) + "[" + renderComparison(value.Index) + "]"
	case *ast.CallExpr:
		arguments := make([]string, 0, len(value.Args))
		for _, argument := range value.Args {
			arguments = append(arguments, renderComparison(argument))
		}
		return renderComparison(value.Fun) + "(" + strings.Join(arguments, ", ") + ")"
	}
	return "?"
}
