package localstore

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
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

// The projection refusal inventory is a derived-completeness gate: the expected
// set of refusal call sites is parsed out of the production sources rather than
// listed by hand, so a new refusal clause cannot ship without a negative path.
//
// Completeness has two halves and both are derived.
//
//  1. Every refusal raised through a refusal funnel must have an exercised
//     negative path, reached beneath openProjection unless the line says why it
//     cannot be. Being *called* and having its *production effect exercised* are
//     different facts: a unit test that invokes a refusal helper directly proves
//     the helper refuses, and proves nothing about whether the production entry
//     still consults it.
//  2. Every refusal in a projection-owned source must actually go through a
//     funnel. Half 1 is only as complete as the funnels are exhaustive, so an
//     ownership guard or an ErrUnsafeOwnership wrap written inline — invisible
//     to half 1 by construction — is itself a failure of this gate.
var (
	exercisedProjectionRefusalSites        sync.Map
	productionDrivenProjectionRefusalSites sync.Map
)

const (
	// A site whose refusal is provably owned by another clause on the
	// production path.
	projectionRefusalSubsumedMarker = "projection-refusal-subsumed:"
	// A site that openProjection cannot reach, with the reason stated inline.
	projectionRefusalDirectMarker = "projection-refusal-direct:"
)

// projectionRefusalFunnels are the identifiers a projection refusal may be
// raised through. Both record their immediate caller, so one entry in this set
// yields one inventory site per call site rather than one per funnel.
var projectionRefusalFunnels = map[string]struct{}{
	"projectionRefusal":          {},
	"projectionOwnershipRefusal": {},
}

// projectionOwnershipVerifiers are the shared owner-only checks whose refusals
// are raised inside another file. A projection-owned guard around one of them
// must re-raise through a funnel, otherwise its refusal has no inventory site.
var projectionOwnershipVerifiers = map[string]struct{}{
	"verifyOwnerFileInfo":  {},
	"verifyOwnerDirectory": {},
}

// projectionUnroutedRefusalSentinels are sentinels that only ever denote a
// refusal, never operational error propagation, so wrapping one directly with
// fmt.Errorf inside a projection-owned source bypasses the inventory.
var projectionUnroutedRefusalSentinels = map[string]struct{}{
	"ErrUnsafeOwnership": {},
}

func TestMain(main *testing.M) {
	baseProjectionRefusal := projectionRefusal
	projectionRefusal = func(sentinel error, format string, arguments ...any) error {
		recordProjectionRefusalSite()
		return baseProjectionRefusal(sentinel, format, arguments...)
	}
	baseProjectionOwnershipRefusal := projectionOwnershipRefusal
	projectionOwnershipRefusal = func(sentinel error, cause error, format string, arguments ...any) error {
		recordProjectionRefusalSite()
		return baseProjectionOwnershipRefusal(sentinel, cause, format, arguments...)
	}
	code := main.Run()
	if code == 0 && fullLocalstorePackageTestRun() && runtime.GOOS != "windows" {
		failures, err := projectionRefusalInventoryFailures()
		if err != nil {
			fmt.Fprintf(os.Stderr, "derive projection refusal inventory: %v\n", err)
			code = 1
		} else if len(failures) != 0 {
			for _, failure := range failures {
				fmt.Fprintln(os.Stderr, failure)
			}
			code = 1
		}
	}
	os.Exit(code)
}

// recordProjectionRefusalSite attributes the refusal to the production line
// that decided it, which is the caller of the funnel wrapper installed above.
func recordProjectionRefusalSite() {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	site := fmt.Sprintf("%s:%d", filepath.Base(file), line)
	exercisedProjectionRefusalSites.Store(site, struct{}{})
	if calledBeneathOpenProjection() {
		productionDrivenProjectionRefusalSites.Store(site, struct{}{})
	}
}

// calledBeneathOpenProjection reports whether the current refusal was raised
// underneath the production state-engine entry point rather than by a test
// calling an internal helper directly.
func calledBeneathOpenProjection() bool {
	programCounters := make([]uintptr, 64)
	for {
		count := runtime.Callers(2, programCounters)
		frames := runtime.CallersFrames(programCounters[:count])
		for {
			frame, more := frames.Next()
			if strings.HasSuffix(frame.Function, ".openProjection") {
				return true
			}
			if !more {
				break
			}
		}
		if count < len(programCounters) {
			return false
		}
		programCounters = make([]uintptr, 2*len(programCounters))
	}
}

func fullLocalstorePackageTestRun() bool {
	selected := flag.Lookup("test.run")
	return selected == nil || selected.Value.String() == ""
}

type projectionRefusalSite struct {
	site        string
	allowDirect bool
}

func projectionRefusalInventoryFailures() ([]string, error) {
	expected, unrouted, err := declaredProjectionRefusalSites()
	if err != nil {
		return nil, err
	}
	var uncalled, unexercised []string
	for _, site := range expected {
		if _, ok := exercisedProjectionRefusalSites.Load(site.site); !ok {
			uncalled = append(uncalled, site.site)
			continue
		}
		if site.allowDirect {
			continue
		}
		if _, ok := productionDrivenProjectionRefusalSites.Load(site.site); !ok {
			unexercised = append(unexercised, site.site)
		}
	}
	sort.Strings(uncalled)
	sort.Strings(unexercised)
	sort.Strings(unrouted)

	var failures []string
	if len(unrouted) != 0 {
		failures = append(failures, fmt.Sprintf(
			"projection refusals raised outside the refusal funnels %v, so the derived inventory cannot require a negative path for them; re-raise them through a funnel at: %s",
			sortedNames(projectionRefusalFunnels), strings.Join(unrouted, ", ")))
	}
	if len(uncalled) != 0 {
		failures = append(failures, "projection refusal call sites without an exercised negative path: "+strings.Join(uncalled, ", "))
	}
	if len(unexercised) != 0 {
		failures = append(failures, fmt.Sprintf(
			"projection refusal call sites never reached beneath openProjection (a direct helper call does not prove the production effect); drive OpenProjection or annotate the line with %q: %s",
			projectionRefusalDirectMarker, strings.Join(unexercised, ", ")))
	}
	return failures, nil
}

func sortedNames(set map[string]struct{}) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// projectionOwnedSource reports whether a production file belongs to the
// projection deliverable. The routing half of the gate is scoped to these files
// so it constrains this task's own refusals rather than the shared path and
// object-store sources owned elsewhere.
func projectionOwnedSource(name string) bool {
	return strings.HasPrefix(name, "projection")
}

// declaredProjectionRefusalSites derives the expected refusal inventory and the
// unrouted refusals from the production AST. Only files the current build
// context actually compiles are considered, so a platform-specific refusal is
// never demanded from a platform that cannot execute it.
func declaredProjectionRefusalSites() ([]projectionRefusalSite, []string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}
	context := build.Default
	fileSet := token.NewFileSet()
	var expected []projectionRefusalSite
	var unrouted []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		matches, err := context.MatchFile(directory, entry.Name())
		if err != nil {
			return nil, nil, err
		}
		if !matches {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		lines := strings.Split(string(source), "\n")
		parsed, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			return nil, nil, err
		}
		site := func(position token.Position) string {
			return fmt.Sprintf("%s:%d", entry.Name(), position.Line)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CallExpr:
				if identifier, ok := typed.Fun.(*ast.Ident); ok {
					if _, funnel := projectionRefusalFunnels[identifier.Name]; funnel {
						position := fileSet.Position(typed.Pos())
						line := lines[position.Line-1]
						if strings.Contains(line, projectionRefusalSubsumedMarker) {
							return true
						}
						expected = append(expected, projectionRefusalSite{
							site:        site(position),
							allowDirect: strings.Contains(line, projectionRefusalDirectMarker),
						})
						return true
					}
				}
				// fmt.Errorf is a selector, not an identifier, so the unrouted
				// check must not sit behind the funnel-identifier match above.
				if !projectionOwnedSource(entry.Name()) {
					return true
				}
				if isProjectionRefusalFunnelDefinition(parsed, typed) {
					return true
				}
				if wrapsUnroutedRefusalSentinel(typed) {
					unrouted = append(unrouted, site(fileSet.Position(typed.Pos())))
				}
			case *ast.IfStmt:
				if !projectionOwnedSource(entry.Name()) {
					return true
				}
				if !guardsOwnershipVerifier(typed) || bodyRaisesThroughFunnel(typed.Body) {
					return true
				}
				unrouted = append(unrouted, site(fileSet.Position(typed.Pos())))
			}
			return true
		})
	}
	return expected, unrouted, nil
}

// isProjectionRefusalFunnelDefinition excludes the funnels' own bodies, which
// legitimately wrap a refusal sentinel because they are what every routed call
// site delegates to.
func isProjectionRefusalFunnelDefinition(file *ast.File, call *ast.CallExpr) bool {
	inside := false
	ast.Inspect(file, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for index, name := range value.Names {
			if _, funnel := projectionRefusalFunnels[name.Name]; !funnel {
				continue
			}
			if index >= len(value.Values) {
				continue
			}
			if call.Pos() >= value.Values[index].Pos() && call.End() <= value.Values[index].End() {
				inside = true
			}
		}
		return true
	})
	return inside
}

func wrapsUnroutedRefusalSentinel(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Errorf" {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	if !ok || packageName.Name != "fmt" {
		return false
	}
	for _, argument := range call.Args {
		identifier, ok := argument.(*ast.Ident)
		if !ok {
			continue
		}
		if _, refusal := projectionUnroutedRefusalSentinels[identifier.Name]; refusal {
			return true
		}
	}
	return false
}

func guardsOwnershipVerifier(statement *ast.IfStmt) bool {
	found := false
	if statement.Init != nil {
		ast.Inspect(statement.Init, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if _, verifier := projectionOwnershipVerifiers[identifier.Name]; verifier {
				found = true
			}
			return true
		})
	}
	return found
}

func bodyRaisesThroughFunnel(body *ast.BlockStmt) bool {
	raises := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if _, funnel := projectionRefusalFunnels[identifier.Name]; funnel {
			raises = true
		}
		return true
	})
	return raises
}
