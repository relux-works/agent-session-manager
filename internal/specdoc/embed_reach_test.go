package specdoc_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/relux-works/agent-session-manager"

// TestEmbeddedDocumentNeverReachesAProductBinary is the guard behind the claim
// this package's documentation and README.md both make: the embedded SPEC.md is
// a verification input for repository gates and never ships inside a product
// command.
//
// The claim was true by accident until internal/traceability began measuring a
// section's clause inventory from this package, which put it on the import path
// of non-test code for the first time. There is no ax main package yet, so the
// property still holds today; this test is what makes it stay true the moment
// one exists, instead of an 883 KiB embed reaching the product silently.
//
// The import graph is read from the source tree with go/parser rather than from
// the toolchain, so the test needs no network, no build cache and no external
// command. Build constraints are ignored deliberately: counting every file's
// imports over-approximates reachability, which is the safe direction for a
// guard.
func TestEmbeddedDocumentNeverReachesAProductBinary(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	packages, mains := parseModulePackages(t, root)

	// The only main package allowed to reach the embedded document. tracecheck
	// is the specification-to-code ownership gate; cataloggen touches the
	// document from its test binary only, which never ships.
	allowed := map[string]struct{}{
		modulePath + "/internal/traceability/cmd/tracecheck": {},
	}

	target := modulePath + "/internal/specdoc"
	if _, ok := packages[target]; !ok {
		t.Fatalf("package graph does not contain %q; the walk found %d packages", target, len(packages))
	}

	sort.Strings(mains)
	if len(mains) == 0 {
		t.Fatal("package graph found no main package; the walk is not reaching the command tree")
	}
	for _, violation := range unauthorizedReaches(packages, mains, allowed, target) {
		t.Errorf("%s; the embedded specification document must not ship in a product command", violation)
	}

	// Anti-vacuity, two ways. The walk has to be able to see a real path, and
	// the detector has to report one when a disallowed main has it.
	gate := modulePath + "/internal/traceability/cmd/tracecheck"
	path, reaches := importPathTo(packages, gate, target)
	if !reaches {
		t.Fatalf("import walk found no path from %q to %q, so it cannot detect one either", gate, target)
	}
	if len(path) < 2 {
		t.Fatalf("import path %v from %q is degenerate", path, gate)
	}
	if _, permitted := allowed[gate]; !permitted {
		t.Fatalf("allow list no longer contains %q, so the real path above would already be a violation", gate)
	}

	// Plant the command this guard exists for. cmd/ax does not exist yet, so it
	// is introduced into a copy of the real graph, importing a real package that
	// really reaches specdoc.
	planted := make(map[string][]string, len(packages)+1)
	for importPath, imports := range packages {
		planted[importPath] = imports
	}
	product := modulePath + "/cmd/ax"
	planted[product] = []string{modulePath + "/internal/traceability"}
	violations := unauthorizedReaches(planted, append(append([]string(nil), mains...), product), allowed, target)
	if len(violations) != 1 || !strings.Contains(violations[0], product) {
		t.Fatalf("planted cmd/ax violations = %v, want exactly one naming %q", violations, product)
	}
	if !strings.Contains(violations[0], target) {
		t.Fatalf("planted violation %q does not name the reached package %q", violations[0], target)
	}
}

// unauthorizedReaches is the detector both the real graph and the planted
// mutant drive. It returns one message per main package outside allowed that
// can reach target.
func unauthorizedReaches(packages map[string][]string, mains []string, allowed map[string]struct{}, target string) []string {
	var violations []string
	for _, main := range mains {
		if _, permitted := allowed[main]; permitted {
			continue
		}
		path, reaches := importPathTo(packages, main, target)
		if !reaches {
			continue
		}
		violations = append(violations, "main package "+main+" reaches "+target+" through "+strings.Join(path, " -> "))
	}
	sort.Strings(violations)
	return violations
}

// parseModulePackages reads every non-test Go file in the module and returns
// the import graph keyed by import path, plus every main package's import path.
func parseModulePackages(t *testing.T, root string) (map[string][]string, []string) {
	t.Helper()
	packages := make(map[string][]string)
	names := make(map[string]string)
	fileSet := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			base := entry.Name()
			if path != root && (strings.HasPrefix(base, ".") || base == "testdata" || base == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		relative, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		importPath := modulePath
		if relative != "." {
			importPath = modulePath + "/" + filepath.ToSlash(relative)
		}
		names[importPath] = file.Name.Name
		for _, specification := range file.Imports {
			value, unquoteErr := strconv.Unquote(specification.Path.Value)
			if unquoteErr != nil {
				return unquoteErr
			}
			if strings.HasPrefix(value, modulePath) {
				packages[importPath] = append(packages[importPath], value)
			}
		}
		if _, seen := packages[importPath]; !seen {
			packages[importPath] = nil
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk module source: %v", err)
	}
	if len(packages) == 0 {
		t.Fatalf("walk of %q found no Go packages", root)
	}

	var mains []string
	for importPath, name := range names {
		if name == "main" {
			mains = append(mains, importPath)
		}
	}
	return packages, mains
}

// importPathTo returns the first import chain from source to target, if any.
func importPathTo(packages map[string][]string, source, target string) ([]string, bool) {
	visited := map[string]struct{}{source: {}}
	type entry struct {
		importPath string
		chain      []string
	}
	queue := []entry{{importPath: source, chain: []string{source}}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range packages[current.importPath] {
			if next == target {
				return append(append([]string(nil), current.chain...), next), true
			}
			if _, seen := visited[next]; seen {
				continue
			}
			visited[next] = struct{}{}
			queue = append(queue, entry{importPath: next, chain: append(append([]string(nil), current.chain...), next)})
		}
	}
	return nil, false
}

// TestModuleHasNoProductCommandYet records the fact the guard above depends on
// for its present-tense claim. When an ax command lands this fails, and whoever
// lands it has to decide deliberately whether the guard's allow list changes.
func TestModuleHasNoProductCommandYet(t *testing.T) {
	t.Parallel()

	_, mains := parseModulePackages(t, filepath.Join("..", ".."))
	sort.Strings(mains)
	want := []string{
		modulePath + "/internal/catalog/cmd/cataloggen",
		modulePath + "/internal/traceability/cmd/tracecheck",
	}
	if len(mains) != len(want) {
		t.Fatalf("module main packages = %v, want exactly the two repository gates %v", mains, want)
	}
	for index := range want {
		if mains[index] != want[index] {
			t.Fatalf("module main packages = %v, want exactly the two repository gates %v", mains, want)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "..", "cmd", "ax")); err == nil {
		t.Fatal("cmd/ax exists; re-decide whether the embedded specification document may reach it")
	}
}
