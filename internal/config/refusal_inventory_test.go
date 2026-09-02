package config

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
	"strings"
	"sync"
	"testing"
)

// exercisedRefusalSites records the production line that constructed a
// refusal, for every refusal constructor the package declares.
var exercisedRefusalSites sync.Map

func recordRefusalSite() {
	if _, file, line, ok := runtime.Caller(2); ok {
		exercisedRefusalSites.Store(fmt.Sprintf("%s:%d", filepath.Base(file), line), struct{}{})
	}
}

func TestMain(main *testing.M) {
	configError = func(field string, err error) error {
		recordRefusalSite()
		return &DocumentError{Clause: field, Err: err}
	}
	loaderError = func(value Error) error {
		recordRefusalSite()
		return &value
	}
	migrationError = func(value MigrationError) error {
		recordRefusalSite()
		return &value
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

func auditRefusalInventory() []string {
	directory, err := os.Getwd()
	if err != nil {
		return []string{fmt.Sprintf("derive configuration refusal inventory: %v", err)}
	}
	inventory, err := deriveRefusalInventory(directory)
	if err != nil {
		return []string{fmt.Sprintf("derive configuration refusal inventory: %v", err)}
	}
	var failures []string
	// Self-coverage: the walk must scan every refusal form the package
	// declares, so a newly added error type or a raw literal that bypasses an
	// instrumented constructor cannot report green.
	if len(inventory.Uninstrumented) != 0 {
		failures = append(failures, "configuration error types without a single refusal constructor: "+strings.Join(inventory.Uninstrumented, ", "))
	}
	if len(inventory.Duplicated) != 0 {
		failures = append(failures, "configuration error types with more than one refusal constructor: "+strings.Join(inventory.Duplicated, ", "))
	}
	if len(inventory.StrayLiterals) != 0 {
		failures = append(failures, "configuration refusals constructed outside an instrumented constructor: "+strings.Join(inventory.StrayLiterals, ", "))
	}
	var missing []string
	for _, site := range inventory.Sites {
		if _, ok := exercisedRefusalSites.Load(site); !ok {
			missing = append(missing, site)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		failures = append(failures, "configuration refusal call sites without an exercised negative path: "+strings.Join(missing, ", "))
	}
	return failures
}

// refusalInventory is derived from package sources, never hand-listed.
type refusalInventory struct {
	// ErrorTypes are the package types that implement error.
	ErrorTypes []string
	// Constructors maps a refusal constructor name to the error type it builds.
	Constructors map[string]string
	// Sites are file:line positions of production refusal constructor calls.
	Sites []string
	// StrayLiterals are error-type composite literals built anywhere other
	// than inside that type's constructor or as an argument to it.
	StrayLiterals []string
	// Uninstrumented are error types with no refusal constructor at all.
	Uninstrumented []string
	// Duplicated are error types with more than one refusal constructor.
	Duplicated []string
}

func deriveRefusalInventory(directory string) (refusalInventory, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return refusalInventory{}, err
	}
	fileSet := token.NewFileSet()
	type parsedFile struct {
		name   string
		lines  []string
		syntax *ast.File
	}
	var files []parsedFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		source, err := os.ReadFile(path)
		if err != nil {
			return refusalInventory{}, err
		}
		syntax, err := parser.ParseFile(fileSet, path, source, parser.ParseComments)
		if err != nil {
			return refusalInventory{}, err
		}
		files = append(files, parsedFile{name: entry.Name(), lines: strings.Split(string(source), "\n"), syntax: syntax})
	}

	inventory := refusalInventory{Constructors: map[string]string{}}
	errorTypes := map[string]struct{}{}
	for _, file := range files {
		for _, declaration := range file.syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Name.Name != "Error" {
				continue
			}
			if function.Type.Params.NumFields() != 0 || function.Type.Results.NumFields() != 1 {
				continue
			}
			if identifier, ok := function.Type.Results.List[0].Type.(*ast.Ident); !ok || identifier.Name != "string" {
				continue
			}
			if name := receiverTypeName(function.Recv.List[0].Type); name != "" {
				errorTypes[name] = struct{}{}
			}
		}
	}
	for name := range errorTypes {
		inventory.ErrorTypes = append(inventory.ErrorTypes, name)
	}
	sort.Strings(inventory.ErrorTypes)

	// A refusal constructor is a package-level `var name = func(...) error`
	// that builds one of the derived error types.
	constructorRanges := map[string][2]token.Pos{}
	constructorCounts := map[string]int{}
	for _, file := range files {
		for _, declaration := range file.syntax.Decls {
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
				if !ok || !returnsError(literal.Type) {
					continue
				}
				built := constructedErrorType(literal, errorTypes)
				if built == "" {
					continue
				}
				name := value.Names[0].Name
				inventory.Constructors[name] = built
				constructorRanges[name] = [2]token.Pos{literal.Pos(), literal.End()}
				constructorCounts[built]++
			}
		}
	}
	for _, name := range inventory.ErrorTypes {
		switch constructorCounts[name] {
		case 1:
		case 0:
			inventory.Uninstrumented = append(inventory.Uninstrumented, name)
		default:
			inventory.Duplicated = append(inventory.Duplicated, name)
		}
	}

	for _, file := range files {
		permitted := map[ast.Node]struct{}{}
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if _, ok := inventory.Constructors[identifier.Name]; !ok {
				return true
			}
			position := fileSet.Position(call.Pos())
			for _, argument := range call.Args {
				permitted[unwrapUnary(argument)] = struct{}{}
			}
			if !strings.Contains(file.lines[position.Line-1], "config-refusal-subsumed:") {
				inventory.Sites = append(inventory.Sites, fmt.Sprintf("%s:%d", file.name, position.Line))
			}
			return true
		})
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			composite, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			identifier, ok := composite.Type.(*ast.Ident)
			if !ok {
				return true
			}
			if _, ok := errorTypes[identifier.Name]; !ok {
				return true
			}
			if _, ok := permitted[ast.Node(composite)]; ok {
				return true
			}
			for name, bounds := range constructorRanges {
				if inventory.Constructors[name] == identifier.Name && composite.Pos() >= bounds[0] && composite.End() <= bounds[1] {
					return true
				}
			}
			position := fileSet.Position(composite.Pos())
			inventory.StrayLiterals = append(inventory.StrayLiterals, fmt.Sprintf("%s:%d %s", file.name, position.Line, identifier.Name))
			return true
		})
	}
	sort.Strings(inventory.Sites)
	sort.Strings(inventory.StrayLiterals)
	sort.Strings(inventory.Uninstrumented)
	sort.Strings(inventory.Duplicated)
	return inventory, nil
}

func receiverTypeName(expression ast.Expr) string {
	if star, ok := expression.(*ast.StarExpr); ok {
		expression = star.X
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}

func returnsError(signature *ast.FuncType) bool {
	if signature.Results.NumFields() != 1 {
		return false
	}
	identifier, ok := signature.Results.List[0].Type.(*ast.Ident)
	return ok && identifier.Name == "error"
}

func constructedErrorType(literal *ast.FuncLit, errorTypes map[string]struct{}) string {
	for _, field := range literal.Type.Params.List {
		if identifier, ok := field.Type.(*ast.Ident); ok {
			if _, ok := errorTypes[identifier.Name]; ok {
				return identifier.Name
			}
		}
	}
	built := ""
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		composite, ok := node.(*ast.CompositeLit)
		if !ok || built != "" {
			return true
		}
		if identifier, ok := composite.Type.(*ast.Ident); ok {
			if _, ok := errorTypes[identifier.Name]; ok {
				built = identifier.Name
			}
		}
		return true
	})
	return built
}

func unwrapUnary(expression ast.Expr) ast.Node {
	if unary, ok := expression.(*ast.UnaryExpr); ok {
		return unary.X
	}
	return expression
}
