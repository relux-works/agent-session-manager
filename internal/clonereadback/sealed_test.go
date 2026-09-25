package clonereadback_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// This file pins the sealed-sibling invariant (SPEC v0.7.0
// §13.14.2 lines 10753-10763: the validation report aggregates
// TWO DISTINCT validated staged and live read-back manifests): a
// report entry cannot vouch for a sibling from a caller-minted
// struct that merely repeats an ID and a mode label. The report
// entries accept ONLY ValidatedReadBack — minted by the
// read-back validator, unforgeable outside the package — and
// refuse the zero value at both entries before any claim
// comparison. A validated sibling from another plan admits here:
// plan pairing is cross-input reconciliation and belongs to the
// final leaf (see TestReportTupleMatchIsReconciliationBound).

// unsealedPairs enumerates the sibling seal relation at one
// report entry: both zero, staged zero, live zero. Every cell
// refuses with the seal literal.
func unsealedPairs() [][2]bool {
	return [][2]bool{{false, false}, {false, true}, {true, false}}
}

// TestBuildReportRefusesUnsealedSiblings is the forged-sibling
// regression at the Build entry: the zero ValidatedReadBack is
// the only sibling value a caller can mint without the
// validator, and every pair carrying one refuses with the seal
// literal — the entry never reads the inner ID or mode of an
// unsealed sibling.
func TestBuildReportRefusesUnsealedSiblings(t *testing.T) {
	staged, live := mustReadPair(t)
	var zero clonereadback.ValidatedReadBack
	sealed := map[bool]clonereadback.ValidatedReadBack{true: staged, false: zero}
	liveSealed := map[bool]clonereadback.ValidatedReadBack{true: live, false: zero}
	for _, pair := range unsealedPairs() {
		input := validReportInput(staged, live)
		_, err := clonereadback.BuildValidationReport(input, sealed[pair[0]], liveSealed[pair[1]])
		if err == nil {
			t.Fatalf("Build admitted unsealed pair staged=%v live=%v", pair[0], pair[1])
		}
		requireRefusal(t, err, "clone validation report", "read-back sibling is not a validated read")
	}
}

// TestDecodeReportRefusesUnsealedSiblings is the forged-sibling
// regression at the Decode entry: the sealed document is valid,
// yet every pair carrying a zero sibling refuses with the seal
// literal — the seal gate fires before any document member is
// read.
func TestDecodeReportRefusesUnsealedSiblings(t *testing.T) {
	staged, live := mustReadPair(t)
	sealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	var zero clonereadback.ValidatedReadBack
	first := map[bool]clonereadback.ValidatedReadBack{true: staged, false: zero}
	second := map[bool]clonereadback.ValidatedReadBack{true: live, false: zero}
	for _, pair := range unsealedPairs() {
		_, err := clonereadback.DecodeValidationReport(sealed, first[pair[0]], second[pair[1]])
		if err == nil {
			t.Fatalf("Decode admitted unsealed pair staged=%v live=%v", pair[0], pair[1])
		}
		requireRefusal(t, err, "clone validation report", "read-back sibling is not a validated read")
	}
}

// mustOtherPlanStaged seals and decodes one staged read bound to
// a different projection plan: a genuinely validated sibling
// from another operation.
func mustOtherPlanStaged(t *testing.T) clonereadback.ValidatedReadBack {
	t.Helper()
	input := validReadBackInput()
	input.ProjectionPlanID = fixtureDigest("other-plan")
	other := mustDecodeReadBack(t, stagedAuthority(t), mustBuildReadBack(t, stagedAuthority(t), input))
	if other.Manifest().ProjectionPlanID.String() == fixtureDigest("plan") {
		t.Fatal("other-plan sibling carries the report plan: the probe is not cross-plan")
	}
	if other.Mode() != "staged" {
		t.Fatalf("other-plan sibling mode = %q, want staged", other.Mode())
	}
	return other
}

// TestBuildReportSiblingFromAnotherPlanIsBound pins the
// sibling-scope bound at the Build entry: a validated staged
// sibling from another plan admits here — the seal, slot, and
// reference bindings hold — because plan pairing across shapes
// needs the wider reconciliation inputs and belongs to the final
// leaf. The probe fails if the sibling ever shares the report
// plan, so the admission pins the bound, not a same-plan seal.
func TestBuildReportSiblingFromAnotherPlanIsBound(t *testing.T) {
	_, live := mustReadPair(t)
	other := mustOtherPlanStaged(t)
	input := validReportInput(other, live)
	if input.ProjectionPlanID == other.Manifest().ProjectionPlanID.String() {
		t.Fatal("report plan equals the sibling plan: the probe is not cross-plan")
	}
	mustBuildReport(t, input, other, live)
}

// TestDecodeReportSiblingFromAnotherPlanIsBound pins the
// sibling-scope bound at the Decode entry: the report sealed
// over a validated other-plan staged sibling decodes with that
// sibling — plan pairing stays reconciliation scope.
func TestDecodeReportSiblingFromAnotherPlanIsBound(t *testing.T) {
	_, live := mustReadPair(t)
	other := mustOtherPlanStaged(t)
	sealed := mustBuildReport(t, validReportInput(other, live), other, live)
	mustDecodeReport(t, sealed, other, live)
}

// siblingBareType reports whether one parameter type spelling is
// a caller-mintable sibling: the exported ReadBackManifest
// value/pointer or a bare scalar digest value/pointer.
func siblingBareType(expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "ReadBackManifest"
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "scalar" && selector.Sel.Name == "Digest"
}

// TestReportEntriesTakeOnlySealedSiblings pins that no exported
// entry takes a caller-mintable sibling: neither the exported
// ReadBackManifest value/pointer nor a bare digest
// value/pointer appears in any exported parameter list —
// functions or methods. The report entries take ONLY
// ValidatedReadBack. The P-seal-param-* harness rows plant one
// violation each and must redden this test.
func TestReportEntriesTakeOnlySealedSiblings(t *testing.T) {
	dir := packageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var violations []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
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
			if !ok || !ast.IsExported(fn.Name.Name) {
				continue
			}
			if fn.Type.Params == nil {
				continue
			}
			for _, field := range fn.Type.Params.List {
				if siblingBareType(field.Type) {
					violations = append(violations, fn.Name.Name)
					break
				}
			}
		}
	}
	sort.Strings(violations)
	if len(violations) != 0 {
		t.Fatalf("exported entries taking a caller-mintable sibling = %v, want none: report siblings must be ValidatedReadBack", violations)
	}
}

// sealedResultType reports whether one result type spelling is a
// sealed read: ValidatedReadBack by value or pointer.
func sealedResultType(expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "ValidatedReadBack"
}

// TestValidatedReadBackSealedConstruction pins that the seal is
// unforgeable outside the package: the type carries no exported
// field, the only exported constructors are the two read-back
// validator entries, and the zero value refuses at both report
// entries. The P-seal-ctor harness row plants a third
// constructor and the P-seal-exported-field row plants an
// exported field; both must redden this test.
func TestValidatedReadBackSealedConstruction(t *testing.T) {
	sealed := reflect.TypeOf(clonereadback.ValidatedReadBack{})
	if sealed.NumField() == 0 {
		t.Fatal("ValidatedReadBack has no fields: the seal carries nothing")
	}
	for i := 0; i < sealed.NumField(); i++ {
		field := sealed.Field(i)
		if field.PkgPath == "" {
			t.Errorf("ValidatedReadBack.%s is exported: callers can forge the seal field by field", field.Name)
		}
	}
	dir := packageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	want := map[string]bool{
		"BuildReadBackEvidenceManifest": true, "DecodeReadBackEvidenceManifest": true,
	}
	var found []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
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
			if !ok || !ast.IsExported(fn.Name.Name) {
				continue
			}
			if fn.Type.Results == nil {
				continue
			}
			mints := false
			for _, field := range fn.Type.Results.List {
				if sealedResultType(field.Type) {
					mints = true
					break
				}
			}
			if mints {
				found = append(found, fn.Name.Name)
			}
		}
	}
	sort.Strings(found)
	if len(found) != len(want) {
		t.Fatalf("seal constructors = %v, want the two read-back entries: no third mint, no missing mint", found)
	}
	for _, name := range found {
		if !want[name] {
			t.Fatalf("seal constructors = %v, want the two read-back entries: %s mints outside the validator", found, name)
		}
	}
	// The zero value is the only caller-mintable seal and both
	// report entries refuse it (the full unsealed grid rides the
	// dedicated per-entry regressions above).
	staged, live := mustReadPair(t)
	var zero clonereadback.ValidatedReadBack
	_, err = clonereadback.BuildValidationReport(validReportInput(staged, live), zero, zero)
	requireRefusal(t, err, "clone validation report", "read-back sibling is not a validated read")
	sealedDoc := mustBuildReport(t, validReportInput(staged, live), staged, live)
	_, err = clonereadback.DecodeValidationReport(sealedDoc, zero, zero)
	requireRefusal(t, err, "clone validation report", "read-back sibling is not a validated read")
}

// TestNoGeneratedBytecode pins that no generated Python bytecode
// rides the package: the mutant harness must run with bytecode
// writing disabled (python3 -B), so a __pycache__ blob can never
// join the candidate again.
func TestNoGeneratedBytecode(t *testing.T) {
	dir := packageDir(t)
	var offenders []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "__pycache__" {
				offenders = append(offenders, path)
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".pyc") {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	sort.Strings(offenders)
	if len(offenders) != 0 {
		t.Fatalf("generated bytecode present = %v: run the harness with python3 -B and remove it", offenders)
	}
}
