package canonicaljson

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The utf8.ValidString re-checks in closed_shapes.go are documented as
// subsumed by decodeStrict rather than deleted, and that subsumption rests on
// one package-internal invariant: every exported entry point that can reach a
// re-check takes its input as bytes and decodes it through decodeStrict, which
// refuses input that is not valid UTF-8 before any value exists.
//
// A prose comment cannot notice when that invariant is broken. These tests pin
// it, so an exported entry point that accepted an already-decoded Go value and
// handed it to one of those validators reddens here instead of silently
// turning eight documented-unreachable guards back into live ones.
//
// WHAT THE CALL GRAPH BELOW MODELS, AND WHAT IT DOES NOT.
//
// The graph is derived from the AST alone, so it is an approximation, and a
// pin that does not state its bound is the false guarantee it was meant to
// replace. It records an edge for:
//
//   - a call of a package-level function by its identifier;
//   - a call of a method declared in this package, matched by method name;
//   - a call through a function value - a callee that is not a package
//     function, a predeclared identifier, or a named type - which is assumed
//     to reach every function whose identifier is used as a value anywhere in
//     this package. That is the shape this package actually dispatches
//     through: immutableObjectShapeValidators holds validator function values
//     registered in mustBuildImmutableObjectShapeValidators and is invoked as
//     validator(object) in validateImmutableObjectShape.
//
// It records no edge, and therefore proves nothing, for a callee reached
// through reflection, through a function value handed to another package and
// invoked there, or through a func-typed struct field: `x.M(...)` resolves
// only when M is a method declared in this package, so a dispatch table parked
// in a struct field and invoked as `table.validate(object)` evades this graph
// entirely. The claim these tests support is bounded to the constructions
// above; it is not "the invariant cannot be broken".

// predeclaredCallees are the identifiers a call can name without reaching any
// function declared in this package: the builtins, and the predeclared type
// names that appear as conversions.
var predeclaredCallees = map[string]bool{
	"append": true, "cap": true, "clear": true, "close": true, "complex": true,
	"copy": true, "delete": true, "imag": true, "len": true, "make": true,
	"max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true,

	"any": true, "bool": true, "byte": true, "complex64": true, "complex128": true,
	"error": true, "float32": true, "float64": true, "int": true, "int8": true,
	"int16": true, "int32": true, "int64": true, "rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true,
}

// productionDeclaration is one derived production function or method.
type productionDeclaration struct {
	key         string
	declaration *ast.FuncDecl
}

// productionFunctionKey names a declaration uniquely. Methods are included:
// skipping Recv != nil would let an exported method that takes an
// already-decoded value pass every gate below without being modelled at all.
func productionFunctionKey(function *ast.FuncDecl) string {
	if function.Recv == nil {
		return function.Name.Name
	}
	return receiverTypeName(function.Recv) + "." + function.Name.Name
}

func receiverTypeName(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) == 0 {
		return "?"
	}
	expression := receiver.List[0].Type
	for {
		switch typed := expression.(type) {
		case *ast.StarExpr:
			expression = typed.X
		case *ast.ParenExpr:
			expression = typed.X
		case *ast.IndexExpr:
			expression = typed.X
		case *ast.IndexListExpr:
			expression = typed.X
		case *ast.Ident:
			return typed.Name
		default:
			return "?"
		}
	}
}

// utf8GuardedFunctions derives, rather than names, the production functions
// that re-check UTF-8. Deriving them means a new re-check is covered by this
// pin the moment it is written, and methods are derived alongside plain
// functions so a re-check moved onto a method stays covered.
func utf8GuardedFunctions(t *testing.T) map[string]bool {
	t.Helper()

	_, files := parsedProductionPackage(t)
	guarded := make(map[string]bool)
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if !isUTF8ValidStringCall(node) {
					return true
				}
				guarded[productionFunctionKey(function)] = true
				return true
			})
		}
	}
	return guarded
}

func isUTF8ValidStringCall(node ast.Node) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "ValidString" {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	return ok && packageName.Name == "utf8"
}

// productionCallGraph maps each production function and method to the ones it
// can call, under the model documented at the top of this file.
func productionCallGraph(t *testing.T) (map[string]*ast.FuncDecl, map[string]map[string]bool) {
	t.Helper()

	_, files := parsedProductionPackage(t)

	declarations := make(map[string]*ast.FuncDecl)
	keysByName := make(map[string][]string)
	declaredTypes := make(map[string]bool)
	importNames := make(map[string]bool)
	declarationNameIdentifiers := make(map[*ast.Ident]bool)
	for _, file := range files {
		for _, specification := range file.Imports {
			importNames[importLocalName(specification)] = true
		}
		for _, declaration := range file.Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				if typed.Body == nil {
					continue
				}
				key := productionFunctionKey(typed)
				declarations[key] = typed
				keysByName[typed.Name.Name] = append(keysByName[typed.Name.Name], key)
				declarationNameIdentifiers[typed.Name] = true
			case *ast.GenDecl:
				for _, specification := range typed.Specs {
					if typeSpecification, ok := specification.(*ast.TypeSpec); ok {
						declaredTypes[typeSpecification.Name.Name] = true
					}
				}
			}
		}
	}

	staticCalleeIdentifiers := make(map[*ast.Ident]bool)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch callee := unparenthesize(call.Fun).(type) {
			case *ast.Ident:
				staticCalleeIdentifiers[callee] = true
			case *ast.SelectorExpr:
				staticCalleeIdentifiers[callee.Sel] = true
			}
			return true
		})
	}

	// A function whose identifier is used as a value - passed as an argument,
	// stored in a table, returned - can be invoked through any function value.
	// This is the edge the previous revision of this pin was missing, and the
	// one this package's own dispatch table needs.
	addressTaken := make(map[string]bool)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok || staticCalleeIdentifiers[identifier] || declarationNameIdentifiers[identifier] {
				return true
			}
			for _, key := range keysByName[identifier.Name] {
				addressTaken[key] = true
			}
			return true
		})
	}

	edges := make(map[string]map[string]bool, len(declarations))
	for key, function := range declarations {
		called := make(map[string]bool)
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			static, indirect := resolveCallee(call, keysByName, declaredTypes, importNames)
			for _, callee := range static {
				called[callee] = true
			}
			if indirect {
				for callee := range addressTaken {
					called[callee] = true
				}
			}
			return true
		})
		edges[key] = called
	}
	return declarations, edges
}

// resolveCallee classifies one call site. It returns the production
// declarations the call reaches statically, and whether the call goes through a
// function value, in which case the caller is treated as reaching every
// address-taken function.
func resolveCallee(
	call *ast.CallExpr,
	keysByName map[string][]string,
	declaredTypes map[string]bool,
	importNames map[string]bool,
) ([]string, bool) {
	switch callee := unparenthesize(call.Fun).(type) {
	case *ast.Ident:
		if keys, ok := keysByName[callee.Name]; ok {
			return keys, false
		}
		if predeclaredCallees[callee.Name] || declaredTypes[callee.Name] {
			return nil, false
		}
		// A local function-typed variable, parameter, or closure.
		return nil, true
	case *ast.SelectorExpr:
		if packageName, ok := unparenthesize(callee.X).(*ast.Ident); ok && importNames[packageName.Name] {
			return nil, false
		}
		var methods []string
		for _, key := range keysByName[callee.Sel.Name] {
			if strings.Contains(key, ".") {
				methods = append(methods, key)
			}
		}
		// An external method, or a func-typed field. The second is the
		// construction this graph does not model; see the note above.
		return methods, false
	case *ast.FuncLit:
		// Immediately invoked; its body is walked as part of the enclosing
		// declaration already.
		return nil, false
	case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.StructType, *ast.InterfaceType, *ast.StarExpr:
		// A conversion to a composite or pointer type.
		return nil, false
	default:
		// A computed callee: a table lookup, or the result of another call.
		return nil, true
	}
}

func unparenthesize(expression ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parenthesized.X
	}
}

func importLocalName(specification *ast.ImportSpec) string {
	if specification.Name != nil {
		return specification.Name.Name
	}
	path := strings.Trim(specification.Path.Value, `"`)
	if index := strings.LastIndex(path, "/"); index >= 0 {
		return path[index+1:]
	}
	return path
}

func reachableProductionFunctions(edges map[string]map[string]bool, from string) map[string]bool {
	reached := make(map[string]bool)
	queue := []string{from}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for next := range edges[current] {
			if reached[next] {
				continue
			}
			reached[next] = true
			queue = append(queue, next)
		}
	}
	return reached
}

// exportedProductionEntryPoints derives the exported functions and methods of
// the package, sorted, so the reachability walk below has a stable subject.
func exportedProductionEntryPoints(declarations map[string]*ast.FuncDecl) []productionDeclaration {
	var entryPoints []productionDeclaration
	for key, declaration := range declarations {
		if !ast.IsExported(declaration.Name.Name) {
			continue
		}
		entryPoints = append(entryPoints, productionDeclaration{key: key, declaration: declaration})
	}
	sort.Slice(entryPoints, func(i, j int) bool { return entryPoints[i].key < entryPoints[j].key })
	return entryPoints
}

// byteInputParameters reports whether every parameter of function is a byte
// slice or a slice of byte slices. Such a function cannot be handed a decoded
// Go string, map or slice by a caller; it has to decode its own input.
//
// A method never qualifies. Its receiver is an already-constructed Go value
// that the caller supplies, so an exported method reaching a re-check breaks
// the subsumption argument whatever its parameters look like.
func byteInputParameters(function *ast.FuncDecl) bool {
	if function.Recv != nil {
		return false
	}
	if function.Type.Params == nil {
		return true
	}
	for _, field := range function.Type.Params.List {
		if !isByteSliceType(field.Type) && !isByteSliceSliceType(field.Type) {
			return false
		}
	}
	return true
}

func isByteSliceType(expression ast.Expr) bool {
	array, ok := expression.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	element, ok := array.Elt.(*ast.Ident)
	return ok && element.Name == "byte"
}

func isByteSliceSliceType(expression ast.Expr) bool {
	array, ok := expression.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	return isByteSliceType(array.Elt)
}

// TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck is the first half
// of the pin the subsumption comments in closed_shapes.go depend on.
//
// Every exported production function or method that can reach a
// utf8.ValidString re-check must take its input as bytes, so a caller cannot
// supply a Go string, map or slice that never passed through this package's
// decoder. An exported entry point taking an already-decoded value into that
// reachable set fails here, which is exactly the break the comments declare -
// including one that reaches the re-checks through the package's own
// immutableObjectShapeValidators dispatch table rather than by calling a
// validator directly.
//
// TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage is the second
// half: bytes only help if decodeStrict is what decodes them.
func TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck(t *testing.T) {
	t.Parallel()

	guarded := utf8GuardedFunctions(t)
	if len(guarded) == 0 {
		t.Fatal("derived no utf8.ValidString re-check; the derivation, not the package, is wrong")
	}
	declarations, edges := productionCallGraph(t)
	if _, ok := declarations["decodeStrict"]; !ok {
		t.Fatal("decodeStrict is not a top-level production function; the subsumption argument names it")
	}

	entryPoints := exportedProductionEntryPoints(declarations)
	if len(entryPoints) == 0 {
		t.Fatal("derived no exported entry point")
	}

	guardedEntryPoints := 0
	for _, entryPoint := range entryPoints {
		reachedGuards := reachedGuardNames(edges, guarded, entryPoint.key)
		if len(reachedGuards) == 0 {
			continue
		}
		guardedEntryPoints++

		if !byteInputParameters(entryPoint.declaration) {
			t.Errorf(
				"exported entry point %s reaches the utf8.ValidString re-checks in %v "+
					"but does not take its input as bytes; the subsumption comments in "+
					"closed_shapes.go claim no caller can supply an already-decoded value",
				entryPoint.key, reachedGuards,
			)
		}
	}
	if guardedEntryPoints == 0 {
		t.Fatal("no exported entry point reaches a utf8.ValidString re-check; the pin would assert nothing")
	}
}

// TestEveryUTF8RecheckIsCoveredByTheEntryPointPin reports the pin's coverage as
// a ratio instead of leaving it to prose. A guarded function that no exported
// entry point reaches is asserted about vacuously: no entry point mutant can
// redden for it, however thorough the surrounding derivation looks.
//
// The previous revision of this graph covered 3 of 7 and said nothing about it.
func TestEveryUTF8RecheckIsCoveredByTheEntryPointPin(t *testing.T) {
	t.Parallel()

	guarded := utf8GuardedFunctions(t)
	declarations, edges := productionCallGraph(t)
	entryPoints := exportedProductionEntryPoints(declarations)

	covered := make(map[string]bool)
	for _, entryPoint := range entryPoints {
		for _, name := range reachedGuardNames(edges, guarded, entryPoint.key) {
			covered[name] = true
		}
	}

	var uncovered []string
	for name := range guarded {
		if !covered[name] {
			uncovered = append(uncovered, name)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) > 0 {
		t.Errorf(
			"utf8.ValidString re-check coverage is %d/%d; %v are reachable from no exported entry "+
				"point, so TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck asserts nothing "+
				"about them",
			len(covered), len(guarded), uncovered,
		)
	}
	t.Log(fmt.Sprintf("utf8.ValidString re-check coverage: %d/%d guarded functions", len(covered), len(guarded)))
}

// reachedGuardNames returns the sorted guarded functions reachable from one
// production declaration.
func reachedGuardNames(edges map[string]map[string]bool, guarded map[string]bool, from string) []string {
	var names []string
	for candidate := range reachableProductionFunctions(edges, from) {
		if guarded[candidate] {
			names = append(names, candidate)
		}
	}
	sort.Strings(names)
	return names
}

// TestEveryUTF8RecheckDeclaresItsSubsumption requires each utf8.ValidString
// re-check in closed_shapes.go to carry a comment naming decodeStrict, so a new
// re-check added without the reachability argument is reported rather than
// silently joining the survivors that this Story exists to remove.
func TestEveryUTF8RecheckDeclaresItsSubsumption(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	path := filepath.Join(directory, "closed_shapes.go")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(contents), "\n")

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, contents, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	found := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ValidString" {
			return true
		}
		packageName, ok := selector.X.(*ast.Ident)
		if !ok || packageName.Name != "utf8" {
			return true
		}
		found++
		line := fileSet.Position(call.Pos()).Line
		if !precedingCommentNamesDecodeStrict(lines, line) {
			t.Errorf(
				"closed_shapes.go:%d re-checks UTF-8 with no preceding comment naming decodeStrict "+
					"as the validator that subsumes it",
				line,
			)
		}
		return true
	})
	if found == 0 {
		t.Fatal("derived no utf8.ValidString re-check in closed_shapes.go; the derivation, not the package, is wrong")
	}
}

// precedingCommentNamesDecodeStrict reports whether the contiguous comment
// block immediately above the one-indexed line names decodeStrict.
func precedingCommentNamesDecodeStrict(lines []string, line int) bool {
	named := false
	for index := line - 2; index >= 0; index-- {
		text := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(text, "//") {
			break
		}
		if strings.Contains(text, "decodeStrict") {
			named = true
		}
	}
	return named
}

// TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage completes the
// subsumption argument. Byte-only entry points prove nothing on their own; they
// prove the UTF-8 re-checks unreachable only because decodeStrict, which
// refuses input that is not valid UTF-8, is the single place in the production
// package where bytes become Go values.
//
// A second decoder anywhere in the package would decode bytes this package
// never UTF-8-validated, and every subsumption comment in closed_shapes.go
// would become false. That addition reddens here.
func TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage(t *testing.T) {
	t.Parallel()

	fileSet, files := parsedProductionPackage(t)
	entryDecoders := map[string]bool{"NewDecoder": true, "Unmarshal": true}

	found := 0
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !entryDecoders[selector.Sel.Name] {
					return true
				}
				packageName, ok := selector.X.(*ast.Ident)
				if !ok || packageName.Name != "json" {
					return true
				}
				found++
				if function.Name.Name != "decodeStrict" {
					t.Errorf(
						"%s:%d decodes JSON with json.%s inside %s; decodeStrict must be the only "+
							"place bytes become Go values, or the UTF-8 subsumption comments in "+
							"closed_shapes.go stop holding",
						filepath.Base(fileSet.Position(call.Pos()).Filename),
						fileSet.Position(call.Pos()).Line,
						selector.Sel.Name,
						function.Name.Name,
					)
				}
				return true
			})
		}
	}
	if found == 0 {
		t.Fatal("derived no JSON decode in the production package; the derivation, not the package, is wrong")
	}
}
