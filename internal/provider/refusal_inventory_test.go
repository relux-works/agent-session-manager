package provider

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

// exercisedRefusalSites records the production file:line that constructed a
// refusal, for every instrumented refusal constructor call made during the
// test run. observedCodes records every stable code those refusals carried.
var exercisedRefusalSites sync.Map
var observedCodes sync.Map

func recordRefusalSite(code string) {
	observedCodes.Store(code, struct{}{})
	if _, file, line, ok := runtime.Caller(2); ok {
		exercisedRefusalSites.Store(fmt.Sprintf("%s:%d", filepath.Base(file), line), struct{}{})
	}
}

func TestMain(main *testing.M) {
	origDuplicate, origInvalid, origPrecondition, origIntegrity := failDuplicate, failInvalid, failPrecondition, failIntegrity
	failDuplicate = func(providerID, firstSource, secondSource string) Error {
		err := origDuplicate(providerID, firstSource, secondSource)
		recordRefusalSite(err.Code())
		return err
	}
	failInvalid = func(detail string) Error {
		err := origInvalid(detail)
		recordRefusalSite(err.Code())
		return err
	}
	failPrecondition = func(detail string, cause error) Error {
		err := origPrecondition(detail, cause)
		recordRefusalSite(err.Code())
		return err
	}
	failIntegrity = func(detail string, cause error) Error {
		err := origIntegrity(detail, cause)
		recordRefusalSite(err.Code())
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

// auditRefusalInventory derives the refusal inventory from package source,
// never from memory: every production call to a refusal constructor must
// have an exercised negative path, no Error value may be built outside the
// four constructors, and the observed code set must equal the closed code
// set exactly.
func auditRefusalInventory() []string {
	directory, err := os.Getwd()
	if err != nil {
		return []string{fmt.Sprintf("derive provider refusal inventory: %v", err)}
	}
	inventory, err := deriveRefusalInventory(directory)
	if err != nil {
		return []string{fmt.Sprintf("derive provider refusal inventory: %v", err)}
	}
	var failures []string
	// Floor: a derived domain that can silently derive nothing is not a
	// measurement. An empty or short derivation must fail closed, never
	// pass vacuously — the same "the scan is blind" floor the package's
	// other derived scans carry.
	if len(inventory.ScannedFiles) == 0 {
		failures = append(failures, "scanned no production sources; the check is blind")
	}
	if len(inventory.Sites) == 0 {
		failures = append(failures, "derived no refusal sites; the scan is blind")
	}
	// Reverse direction: every exercised refusal site must resolve to a
	// derived site. The exercised set is derived from the test run, not
	// hand-listed, so a truncated derivation (1 of N sites, provider.go
	// skipped, or an empty-but-successful read of the wrong directory)
	// reddens here even though the forward check passes vacuously.
	derived := map[string]bool{}
	for _, site := range inventory.Sites {
		derived[site] = true
	}
	var outside []string
	exercisedRefusalSites.Range(func(key, _ any) bool {
		site := key.(string)
		if !derived[site] {
			outside = append(outside, site)
		}
		return true
	})
	sort.Strings(outside)
	if len(outside) != 0 {
		failures = append(failures, "exercised refusal sites outside the derived inventory; the derivation is short: "+strings.Join(outside, ", "))
	}
	if len(inventory.StrayLiterals) != 0 {
		sort.Strings(inventory.StrayLiterals)
		failures = append(failures, "provider refusals built outside an instrumented constructor: "+strings.Join(inventory.StrayLiterals, ", "))
	}
	if len(inventory.RawConstructors) != 0 {
		sort.Strings(inventory.RawConstructors)
		failures = append(failures, "provider raw error construction outside documented cause sites: "+strings.Join(inventory.RawConstructors, ", "))
	}
	var missing []string
	for _, site := range inventory.Sites {
		if _, ok := exercisedRefusalSites.Load(site); !ok {
			missing = append(missing, site)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		failures = append(failures, "provider refusal call sites without an exercised negative path: "+strings.Join(missing, ", "))
	}
	var codes []string
	observedCodes.Range(func(key, _ any) bool {
		codes = append(codes, key.(string))
		return true
	})
	sort.Strings(codes)
	want := []string{codeIntegrityFailure, codeInvalidConfig, codeLocalPrecondition}
	if fmt.Sprintf("%v", codes) != fmt.Sprintf("%v", want) {
		failures = append(failures, fmt.Sprintf("observed refusal codes = %v, want closed set %v", codes, want))
	}
	return failures
}

// refusalInventory is derived from package sources.
type refusalInventory struct {
	// ScannedFiles are the production source basenames the derivation
	// parsed. The audit fails closed when this is empty: a successful
	// read of the wrong directory must not pass as a clean inventory.
	ScannedFiles []string
	// Sites are file:line positions of production refusal constructor calls.
	Sites []string
	// StrayLiterals are Error composite literals outside the constructors.
	StrayLiterals []string
	// RawConstructors are errors.New, fmt.Errorf, or panic calls outside
	// the documented owner-attestation cause sites.
	RawConstructors []string
}

var refusalConstructors = map[string]bool{
	"failDuplicate":    true,
	"failInvalid":      true,
	"failPrecondition": true,
	"failIntegrity":    true,
}

// causeSiteFiles are the only production files allowed to mint raw cause
// errors: the platform owner-attestation seams whose failures production
// code always wraps in a refusal constructor.
var causeSiteFiles = map[string]bool{
	"os_unix.go":    true,
	"os_windows.go": true,
}

func deriveRefusalInventory(directory string) (refusalInventory, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return refusalInventory{}, err
	}
	var inventory refusalInventory
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		inventory.ScannedFiles = append(inventory.ScannedFiles, name)
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			return refusalInventory{}, err
		}
		syntax, err := parser.ParseFile(fileSet, path, source, parser.ParseComments)
		if err != nil {
			return refusalInventory{}, err
		}
		// Map each constructor's own body span so literals inside it are
		// not mistaken for strays.
		constructorBodies := map[*ast.FuncLit]bool{}
		for _, decl := range syntax.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				values, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, ident := range values.Names {
					if !refusalConstructors[ident.Name] || i >= len(values.Values) {
						continue
					}
					if literal, ok := values.Values[i].(*ast.FuncLit); ok {
						constructorBodies[literal] = true
					}
				}
			}
		}
		ast.Inspect(syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			position := fileSet.Position(call.Pos())
			site := fmt.Sprintf("%s:%d", filepath.Base(position.Filename), position.Line)
			if ident, ok := call.Fun.(*ast.Ident); ok {
				if refusalConstructors[ident.Name] {
					inventory.Sites = append(inventory.Sites, site)
					return true
				}
				if ident.Name == "panic" {
					inventory.RawConstructors = append(inventory.RawConstructors, site+" panic")
					return true
				}
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if qualifier, ok := selector.X.(*ast.Ident); ok {
				qualified := qualifier.Name + "." + selector.Sel.Name
				if qualified == "errors.New" || qualified == "fmt.Errorf" {
					if !causeSiteFiles[name] {
						inventory.RawConstructors = append(inventory.RawConstructors, site+" "+qualified)
					}
					return true
				}
			}
			if selector.Sel.Name == "panic" {
				inventory.RawConstructors = append(inventory.RawConstructors, site+" panic")
			}
			return true
		})
		// Stray Error literals: composite literals of type Error outside
		// a constructor body. Bodies are matched by source span.
		ast.Inspect(syntax, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			ident, ok := literal.Type.(*ast.Ident)
			if !ok || ident.Name != "Error" {
				return true
			}
			for body := range constructorBodies {
				if literal.Pos() >= body.Pos() && literal.End() <= body.End() {
					return true
				}
			}
			position := fileSet.Position(literal.Pos())
			inventory.StrayLiterals = append(inventory.StrayLiterals, fmt.Sprintf("%s:%d", filepath.Base(position.Filename), position.Line))
			return true
		})
	}
	return inventory, nil
}
