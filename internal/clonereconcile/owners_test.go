package clonereconcile_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// This file pins owner delegation structurally: the exported entry
// set is exactly {ReadBackHistory, Reconcile}, every entry reaches
// its required owner validators on the static call graph, and no
// production file re-implements an owner rule (shape parsing,
// digests, UUIDs, tuples, vocabularies, or the valid decision).
// Behavior itself is proven by the gate table (gates_test.go),
// which drives violating inputs through the production entries and
// asserts literal codes. Each structural check ships a control
// plant proving it bites.

// packageDir returns the clonereconcile source directory.
func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller: no caller")
	}
	return filepath.Dir(file)
}

// productionFiles parses every non-test Go file in dir.
func productionFiles(t *testing.T, dir string) map[string]*ast.File {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	files := make(map[string]*ast.File)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files[name] = parsed
	}
	return files
}

// TestEntryCensus pins the exported production entry set: exactly
// ReadBackHistory and Reconcile. A new entry fails here until its
// owner edges are censused below.
func TestEntryCensus(t *testing.T) {
	files := productionFiles(t, packageDir(t))
	var entries []string
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !ast.IsExported(fn.Name.Name) {
				continue
			}
			entries = append(entries, fn.Name.Name)
		}
	}
	sort.Strings(entries)
	want := []string{"ReadBackHistory", "Reconcile"}
	if strings.Join(entries, ",") != strings.Join(want, ",") {
		t.Fatalf("exported entries = %q, want %q", entries, want)
	}
}

// callEdges maps each package-level function to the qualified
// selectors and bare identifiers it calls.
func callEdges(files map[string]*ast.File) map[string][]string {
	edges := make(map[string][]string)
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			var calls []string
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fun := call.Fun.(type) {
				case *ast.SelectorExpr:
					if ident, ok := fun.X.(*ast.Ident); ok {
						calls = append(calls, ident.Name+"."+fun.Sel.Name)
					}
				case *ast.Ident:
					calls = append(calls, fun.Name)
				}
				return true
			})
			edges[fn.Name.Name] = calls
		}
	}
	return edges
}

// reachableOwners returns the transitive owner-qualified selectors
// reachable from entry through package-level calls.
func reachableOwners(entry string, edges map[string][]string) map[string]bool {
	reached := make(map[string]bool)
	var visit func(name string)
	visit = func(name string) {
		for _, callee := range edges[name] {
			if strings.Contains(callee, ".") {
				reached[callee] = true
				continue
			}
			if _, seen := edges[callee]; seen {
				if !reached["func:"+callee] {
					reached["func:"+callee] = true
					visit(callee)
				}
			}
		}
	}
	visit(entry)
	return reached
}

// TestEntryOwnerReachability pins that every production entry
// reaches its required owner validators on the static call graph:
// read-back history decodes only through the read-back owner, and
// reconciliation derives the fidelity report, plan/projected
// inputs, and validation report only through their owners.
func TestEntryOwnerReachability(t *testing.T) {
	edges := callEdges(productionFiles(t, packageDir(t)))
	required := map[string][]string{
		"ReadBackHistory": {"clonereadback.DecodeReadBackEvidenceManifest"},
		"Reconcile": {
			"clonefidelity.BuildFidelityReport",
			"clonefidelity.DecodeFidelityReport",
			"cloneplan.DecodeProjectionPlan",
			"cloneplan.DecodeProjectedObjectManifest",
			"clonereadback.BuildValidationReport",
		},
	}
	for entry, owners := range required {
		reached := reachableOwners(entry, edges)
		for _, owner := range owners {
			if !reached[owner] {
				t.Errorf("entry %s does not reach owner %s on the call graph", entry, owner)
			}
		}
	}
}

// TestOwnerReachabilityControlPlant proves the reachability checker
// bites: an entry that never calls the owner is reported.
func TestOwnerReachabilityControlPlant(t *testing.T) {
	plant := `
package planted
func Reconcile() {}
`
	parsed, err := parser.ParseFile(token.NewFileSet(), "plant.go", plant, 0)
	if err != nil {
		t.Fatalf("parse plant: %v", err)
	}
	reached := reachableOwners("Reconcile", callEdges(map[string]*ast.File{"plant.go": parsed}))
	if reached["clonefidelity.BuildFidelityReport"] {
		t.Fatalf("control plant reached the owner without calling it")
	}
}

// forbiddenReimplementations are owner-rule tokens no production
// file may carry: shape parsing/rendering, digest/UUID/tuple
// decoding, JCS, vocabulary re-checks, and the read-back seal.
var forbiddenReimplementations = []string{
	"urn:ax:schema:",
	"json.Marshal",
	"json.Unmarshal",
	"sha256.",
	"jcs.",
	"Canonicalize",
	"ParseUUID",
	"DecodeStrictObject",
	"CheckDigest",
	"CheckUUIDv7",
	"CheckStringBounds",
	"CheckUint53Bounds",
	"ValidReadBackMode",
	"ValidEvidenceKind",
	"ValidDisposition(",
	"ValidCaptureClass(",
	"sealReadBack",
	"decideValid",
	"ValidatedReadBack{",
}

// allowedOwnerPredicates are the two vocabulary predicates
// production legitimately calls (tier-1 exclusions, tier-2
// normalizations); every other vocabulary token above is a fork.
var allowedOwnerPredicates = []string{
	"clonebundle.ValidCaptureClass(",
	"clonefidelity.ValidDisposition(",
}

// guardReimplementation scans one source for forked owner rules.
func guardReimplementation(source string) []string {
	var hits []string
	for _, token := range forbiddenReimplementations {
		allowed := false
		for _, predicate := range allowedOwnerPredicates {
			if strings.Contains(predicate, token) {
				allowed = true
				break
			}
		}
		for _, line := range strings.Split(source, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if !strings.Contains(line, token) {
				continue
			}
			if allowed {
				rest := line
				for _, predicate := range allowedOwnerPredicates {
					rest = strings.ReplaceAll(rest, predicate, "")
				}
				if !strings.Contains(rest, token) {
					continue
				}
			}
			hits = append(hits, token)
			break
		}
	}
	return hits
}

// TestNoOwnerReimplementation pins that production never
// re-implements an owner rule: no shape URNs, no JSON
// marshal/unmarshal, no digest/UUID/tuple decoding outside the
// delegated scalar/tuple calls, no vocabulary re-checks beyond the
// two allowlisted predicates, and no read-back seal construction.
func TestNoOwnerReimplementation(t *testing.T) {
	dir := packageDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if hits := guardReimplementation(string(source)); len(hits) > 0 {
			t.Errorf("%s re-implements owner rules: %q", name, hits)
		}
	}
}

// TestNoOwnerReimplementationControlPlant proves the guard bites: a
// planted local digest check and a planted seal literal are both
// reported, while the allowlisted predicates pass.
func TestNoOwnerReimplementationControlPlant(t *testing.T) {
	plant := `
package planted
import "crypto/sha256"
func check(input string) bool {
	sum := sha256.Sum256([]byte(input))
	_ = sum
	return true
}
var _ = "urn:ax:schema:clone-validation-report"
`
	if hits := guardReimplementation(plant); len(hits) != 2 {
		t.Fatalf("control plant hits = %q, want the sha256 and URN tokens", hits)
	}
	clean := `
package planted
func check(class, disposition string) bool {
	return clonebundle.ValidCaptureClass(class) && clonefidelity.ValidDisposition(disposition)
}
`
	if hits := guardReimplementation(clean); len(hits) != 0 {
		t.Fatalf("allowlisted predicates hit = %q, want none", hits)
	}
}

// TestSealedReadsNeverMinted pins that production never constructs
// a ValidatedReadBack composite literal: seals arrive only from
// the read-back owner's entries.
func TestSealedReadsNeverMinted(t *testing.T) {
	// Enforced by TestNoOwnerReimplementation (ValidatedReadBack{
	// is a forbidden token); this test pins the positive shape:
	// History flows from ReadBackHistory into Reconcile.
	docs := sealSharedWorldDocs(t)
	input := gateBaseline(t, docs)
	if input.History.Staged.Mode() != "staged" || input.History.Live.Mode() != "live" {
		t.Fatalf("baseline history modes = %q/%q", input.History.Staged.Mode(), input.History.Live.Mode())
	}
}
