package clonereadback_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// This file pins the single-owner rule structurally. The read-back
// and validation shapes carry Fidelity Report, disposition/reason,
// and Projection Plan data only as opaque digest references —
// never as rows — so no clonefidelity or cloneplan validator is
// reachable here by construction:
//
//   - the row census derives every exported entry whose parameters
//     carry a disposition row, a fidelity report, or a projection
//     plan from the real parameter types and asserts the set is
//     empty;
//   - the reachability pin proves every entry that DOES carry
//     owner-shaped data (tuples, bindings, findings) reaches the
//     landed owner decoder on the static call graph;
//   - the no-import guard proves the package imports neither
//     internal/clonefidelity nor internal/cloneplan.
//
// Behavior itself is proven by the owner-delegation table in
// gates_test.go, which drives broken nested documents through the
// production entries and asserts the owner detail surfaces.
// Control plants (a new row-carrying entry, a removed owner call,
// a re-added owner import) redden these tests; logs ride the
// evidence tar.

// structCarriesOwnerRow reports whether one struct type is a
// fidelity/plan row: a disposition member (Disposition or
// ExpectedDispos) paired with a reason-set member (Reasons or
// ReasonCodes), a fidelity report member, or a projection plan
// member.
func structCarriesOwnerRow(typ reflect.Type) bool {
	hasDisposition := false
	hasReasons := false
	for i := 0; i < typ.NumField(); i++ {
		switch typ.Field(i).Name {
		case "Disposition", "ExpectedDispos":
			hasDisposition = true
		case "Reasons", "ReasonCodes":
			hasReasons = true
		case "FidelityReport", "ProjectionPlan", "DispositionRecord", "ItemMapping":
			return true
		}
	}
	return hasDisposition && hasReasons
}

// reflectCarriesOwnerRow reports whether an owner row is reachable
// from the type through fields, elements, keys, and pointees. Open
// any-typed members do not count: an any carries no NAMED row.
// Digest and UUID members are opaque references, never rows.
func reflectCarriesOwnerRow(typ reflect.Type, visited map[reflect.Type]bool) bool {
	if typ == nil {
		return false
	}
	switch typ.Kind() {
	case reflect.Pointer:
		return reflectCarriesOwnerRow(typ.Elem(), visited)
	case reflect.Slice, reflect.Array:
		return reflectCarriesOwnerRow(typ.Elem(), visited)
	case reflect.Map:
		return reflectCarriesOwnerRow(typ.Key(), visited) || reflectCarriesOwnerRow(typ.Elem(), visited)
	case reflect.Struct:
		if structCarriesOwnerRow(typ) {
			return true
		}
		if visited[typ] {
			return false
		}
		visited[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			if reflectCarriesOwnerRow(typ.Field(i).Type, visited) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// exportedEntries lists every exported production entry by name and
// function value.
func exportedEntries() map[string]any {
	return map[string]any{
		"BuildReadBackEvidenceManifest":  clonereadback.BuildReadBackEvidenceManifest,
		"DecodeReadBackEvidenceManifest": clonereadback.DecodeReadBackEvidenceManifest,
		"BuildValidationReport":          clonereadback.BuildValidationReport,
		"DecodeValidationReport":         clonereadback.DecodeValidationReport,
		"ValidReadBackMode":              clonereadback.ValidReadBackMode,
		"ReadBackModes":                  clonereadback.ReadBackModes,
		"ValidEvidenceKind":              clonereadback.ValidEvidenceKind,
		"EvidenceKinds":                  clonereadback.EvidenceKinds,
	}
}

// TestExportedAPIInventory pins the exact exported production API:
// four entries plus four vocabulary helpers. A ninth export — in
// particular a new entry carrying owner-shaped data — reddens this
// test before the row census extends to it; the P-census-entry
// harness row plants exactly that.
func TestExportedAPIInventory(t *testing.T) {
	dir := packageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	want := map[string]bool{
		"BuildReadBackEvidenceManifest": true, "DecodeReadBackEvidenceManifest": true,
		"BuildValidationReport": true, "DecodeValidationReport": true,
		"ValidReadBackMode": true, "ReadBackModes": true,
		"ValidEvidenceKind": true, "EvidenceKinds": true,
	}
	found := map[string]bool{}
	for _, path := range files {
		if len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !ast.IsExported(fn.Name.Name) {
				continue
			}
			found[fn.Name.Name] = true
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("exported API misses %s", name)
		}
	}
	for name := range found {
		if !want[name] {
			t.Errorf("exported API carries unexpected %s: extend the census before shipping it", name)
		}
	}
}

// TestOwnerRowCensusIsEmpty pins that no exported entry carries a
// fidelity/plan row in its parameters: the shapes reference those
// owners by digest only.
func TestOwnerRowCensusIsEmpty(t *testing.T) {
	var carrying []string
	for name, entry := range exportedEntries() {
		typ := reflect.TypeOf(entry)
		for i := 0; i < typ.NumIn(); i++ {
			if reflectCarriesOwnerRow(typ.In(i), map[reflect.Type]bool{}) {
				carrying = append(carrying, name)
				break
			}
		}
	}
	sort.Strings(carrying)
	if len(carrying) != 0 {
		t.Fatalf("entries carrying fidelity/plan rows = %v, want empty: they must call the owner validators", carrying)
	}
	t.Log("owner-row census: 8 entries, 0 carrying (references are digests)")
}

// entryCalls collects the selector calls (package.Function) made by
// one named function across the package production files.
func entryCalls(t *testing.T, dir, function string) map[string]bool {
	t.Helper()
	calls := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		if len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		if len(filepath.Base(path)) >= 6 && filepath.Base(path)[:6] == "plant_" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != function {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				calls[pkg.Name+"."+selector.Sel.Name] = true
				return true
			})
		}
	}
	return calls
}

// callGraphReachability expands one function through same-package
// calls so the pin covers entries that reach the owner through a
// shared build helper.
func callGraphReachability(t *testing.T, dir, function string, depth int) map[string]bool {
	t.Helper()
	reached := map[string]bool{}
	var visit func(name string, remaining int)
	visit = func(name string, remaining int) {
		if remaining < 0 {
			return
		}
		for call := range entryCalls(t, dir, name) {
			reached[call] = true
			if dot := indexByte(call, '.'); dot >= 0 {
				head := call[:dot]
				_ = head
			}
		}
		// Same-package callees surface as bare identifiers, not
		// selectors; collect them with a second pass.
		for _, callee := range localCallees(t, dir, name) {
			if !reached["local."+callee] {
				reached["local."+callee] = true
				visit(callee, remaining-1)
			}
		}
	}
	visit(function, depth)
	return reached
}

func indexByte(value string, char byte) int {
	for i := 0; i < len(value); i++ {
		if value[i] == char {
			return i
		}
	}
	return -1
}

// localCallees lists the same-package functions one function calls.
func localCallees(t *testing.T, dir, function string) []string {
	t.Helper()
	var callees []string
	seen := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		if len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != function || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident, ok := call.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				if !seen[ident.Name] {
					seen[ident.Name] = true
					callees = append(callees, ident.Name)
				}
				return true
			})
		}
	}
	return callees
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

// TestOwnerReachability pins that every entry carrying owner-shaped
// data reaches the landed owner decoder on the static call graph:
// tuples through sessadapter.DecodeTuple, bindings through
// clonebundle.DecodeWorkspaceBinding, findings through
// sessadapter.DecodeFindings, identities through
// clonebundle.OmitSelfDigest, extensions through
// clonebundle.CheckExtensionsClosed/EncodeExtensions.
func TestOwnerReachability(t *testing.T) {
	dir := packageDir(t)
	want := map[string][]string{
		"BuildReadBackEvidenceManifest":  {"sessadapter.DecodeTuple", "clonebundle.DecodeWorkspaceBinding", "clonebundle.EncodeExtensions", "scalar.SHA256Digest"},
		"DecodeReadBackEvidenceManifest": {"sessadapter.DecodeTuple", "clonebundle.DecodeWorkspaceBinding", "clonebundle.OmitSelfDigest"},
		"BuildValidationReport":          {"sessadapter.DecodeTuple", "sessadapter.DecodeFindings", "clonebundle.EncodeExtensions", "scalar.SHA256Digest"},
		"DecodeValidationReport":         {"sessadapter.DecodeTuple", "sessadapter.DecodeFindings", "clonebundle.OmitSelfDigest"},
	}
	for entry, owners := range want {
		reached := callGraphReachability(t, dir, entry, 6)
		for _, owner := range owners {
			if !reached[owner] {
				t.Errorf("entry %s does not reach %s on the static call graph", entry, owner)
			}
		}
	}
	// The shared gates are reached by BOTH sibling entries: a rule
	// pinned at one entry only would leave the other unmeasured.
	for _, gate := range []struct {
		entries []string
		callee  string
	}{
		{[]string{"BuildReadBackEvidenceManifest", "DecodeReadBackEvidenceManifest"}, "checkNativeIdentityMatch"},
		{[]string{"BuildReadBackEvidenceManifest", "DecodeReadBackEvidenceManifest"}, "checkEvidenceOrder"},
		{[]string{"BuildReadBackEvidenceManifest", "DecodeReadBackEvidenceManifest"}, "checkModeAuthority"},
		{[]string{"BuildReadBackEvidenceManifest", "DecodeReadBackEvidenceManifest"}, "modeForAuthorityPurpose"},
		{[]string{"BuildValidationReport", "DecodeValidationReport"}, "checkValidAgreement"},
		{[]string{"BuildValidationReport", "DecodeValidationReport"}, "checkReadRefs"},
		{[]string{"BuildValidationReport", "DecodeValidationReport"}, "checkSiblingsSealed"},
	} {
		for _, entry := range gate.entries {
			reached := callGraphReachability(t, dir, entry, 6)
			if !reached["local."+gate.callee] {
				t.Errorf("entry %s does not reach shared gate %s", entry, gate.callee)
			}
		}
	}
}

// TestNoOwnerImports pins that production files import neither
// internal/clonefidelity nor internal/cloneplan: with no row-shaped
// data crossing the entries, any import would be a forked rule.
func TestNoOwnerImports(t *testing.T) {
	dir := packageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		if len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, spec := range parsed.Imports {
			imported := spec.Path.Value
			if imported == `"github.com/relux-works/agent-session-manager/internal/clonefidelity"` ||
				imported == `"github.com/relux-works/agent-session-manager/internal/cloneplan"` {
				t.Errorf("%s imports %s: entries carry no rows, so the import is a fork", path, imported)
			}
		}
	}
}
