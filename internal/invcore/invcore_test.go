package invcore

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// terminalSpec mirrors the terminalbackend watch list: package-local
// refusal constructors plus the errors/fmt plain-error spellings by
// import path, so an import alias cannot change what the audit sees.
func terminalSpec() ConstructorSpec {
	return ConstructorSpec{
		Local: map[string]bool{"mismatchf": true, "integrityFailure": true},
		Qualified: map[string]map[string]bool{
			"errors": {"New": true},
			"fmt":    {"Errorf": true},
		},
	}
}

// mustParsePlant parses synthetic control-plant source or fails the test:
// a plant that does not parse proves nothing about the audit.
func mustParsePlant(t *testing.T, name, source string) (*ast.File, *token.FileSet) {
	t.Helper()
	syntax, fileSet, failure := ParseSource(name, []byte(source))
	if failure != "" {
		t.Fatalf("control plant %s does not parse: %s", name, failure)
	}
	return syntax, fileSet
}

// TestDiffSetsFailsBothDirections plants one unregistered site and one
// orphan row and requires the harness to report both: the forward
// direction alone passes vacuously on a truncated derivation, so the
// reverse direction is the measurement.
func TestDiffSetsFailsBothDirections(t *testing.T) {
	t.Parallel()

	derived := map[string]struct{}{"site-a": {}, "site-b": {}}
	rows := map[string]struct{}{"site-b": {}, "row-c": {}}
	unregistered, orphaned, failures := DiffSets(derived, rows)
	if len(failures) != 0 {
		t.Fatalf("DiffSets = failures %v, want none", failures)
	}
	if len(unregistered) != 1 || unregistered[0] != "site-a" {
		t.Fatalf("DiffSets unregistered = %v, want [site-a]", unregistered)
	}
	if len(orphaned) != 1 || orphaned[0] != "row-c" {
		t.Fatalf("DiffSets orphaned = %v, want [row-c]", orphaned)
	}
}

// TestDiffSetsFailsClosedOnEmptyDerivation requires an empty derived set
// to fail even when rows are empty too: a domain that silently derives
// nothing is not a measurement.
func TestDiffSetsFailsClosedOnEmptyDerivation(t *testing.T) {
	t.Parallel()

	_, _, failures := DiffSets(map[string]struct{}{}, map[string]struct{}{})
	if len(failures) == 0 {
		t.Fatal("DiffSets(empty, empty) = no failure, want fail-closed on a blind derivation")
	}
}

// TestScanProductionFailsClosedOnZeroFiles requires an empty directory to
// fail instead of passing vacuously.
func TestScanProductionFailsClosedOnZeroFiles(t *testing.T) {
	t.Parallel()

	_, _, failures := ScanProduction(t.TempDir())
	if len(failures) == 0 {
		t.Fatal("ScanProduction(empty dir) = no failure, want fail-closed")
	}
}

// TestScanProductionFailsClosedOnUnparseableFile plants one broken
// production file and requires the scan to fail, never skip: a file the
// derivation cannot read proves nothing.
func TestScanProductionFailsClosedOnUnparseableFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.go"), []byte("package plant\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package plant\nfunc (\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, failures := ScanProduction(dir)
	if len(failures) == 0 {
		t.Fatal("ScanProduction(dir with unparseable file) = no failure, want fail-closed")
	}
	found := false
	for _, failure := range failures {
		if strings.Contains(failure, "broken.go") {
			found = true
		}
	}
	if !found {
		t.Fatalf("ScanProduction failures = %v, want the broken file named", failures)
	}
}

// TestScanProductionSkipsTestFiles requires *_test.go to stay outside the
// production set while production files are selected directory-wide.
func TestScanProductionSkipsTestFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "prod.go"), []byte("package plant\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prod_test.go"), []byte("package plant\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, _, failures := ScanProduction(dir)
	if len(failures) != 0 {
		t.Fatalf("ScanProduction = failures %v, want none", failures)
	}
	if len(files) != 1 || files[0].Name != "prod.go" {
		t.Fatalf("ScanProduction selected %v, want exactly [prod.go]", files)
	}
}

// TestAuditAdmitsDirectCalls requires clean production-shaped source —
// direct constructor calls plus the constructors' own declarations — to
// audit silent.
func TestAuditAdmitsDirectCalls(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "clean.go", `package plant

import (
	"errors"
	"fmt"
)

func mismatchf(detail string) error { return errors.New(detail) }

func build(ok bool) error {
	if !ok {
		return mismatchf("static detail")
	}
	if ok {
		return fmt.Errorf("static %s", "detail")
	}
	return nil
}
`)
	if failures := AuditConstructorReferences(syntax, fileSet, "clean.go", terminalSpec()); len(failures) != 0 {
		t.Fatalf("AuditConstructorReferences(clean) = %v, want none", failures)
	}
}

// TestAuditRejectsVarBinding plants the var-binding bypass: a constructor
// bound to a variable ships a reachable arm the derivation cannot
// attribute to a static call site. Dropping this direction readmits it.
func TestAuditRejectsVarBinding(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "varbind.go", `package plant

import "errors"

func build(ok bool) error {
	newPlain := errors.New
	if !ok {
		return newPlain("static detail")
	}
	return nil
}
`)
	if failures := AuditConstructorReferences(syntax, fileSet, "varbind.go", terminalSpec()); len(failures) == 0 {
		t.Fatal("AuditConstructorReferences(var binding) = no failure; the var-binding bypass walks through")
	}
}

// TestAuditRejectsLocalAlias plants the local-constructor alias bypass:
// `refuse := mismatchf` ships a reachable arm no static derivation can
// attribute. Dropping this direction readmits it.
func TestAuditRejectsLocalAlias(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "localalias.go", `package plant

func mismatchf(detail string) error { return nil }

func build(ok bool) error {
	refuse := mismatchf
	if !ok {
		return refuse("static detail")
	}
	return nil
}
`)
	if failures := AuditConstructorReferences(syntax, fileSet, "localalias.go", terminalSpec()); len(failures) == 0 {
		t.Fatal("AuditConstructorReferences(local alias) = no failure; the alias bypass walks through")
	}
}

// TestAuditRejectsImportAlias plants the import-alias bypass through a
// renamed errors import: an identifier-keyed gate sees `errs.New` and
// passes, while the construction still ships. The audit resolves the
// alias against the import path, so the binding still fails; a gate
// narrowed back to identifier spelling alone passes this plant.
func TestAuditRejectsImportAlias(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "importalias.go", `package plant

import errs "errors"

func build(ok bool) error {
	mint := errs.New
	if !ok {
		return mint("static detail")
	}
	return nil
}
`)
	if failures := AuditConstructorReferences(syntax, fileSet, "importalias.go", terminalSpec()); len(failures) == 0 {
		t.Fatal("AuditConstructorReferences(import alias binding) = no failure; the aliased import walks through")
	}
}

// TestAuditAttributesAliasedDirectCall requires a direct call through an
// import alias to audit silent: the call site is static and attributable,
// only the spelling moved. Failing it would force a needless import
// spelling rule on production.
func TestAuditAttributesAliasedDirectCall(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "aliasedcall.go", `package plant

import errs "errors"

func build(ok bool) error {
	if !ok {
		return errs.New("static detail")
	}
	return nil
}
`)
	if failures := AuditConstructorReferences(syntax, fileSet, "aliasedcall.go", terminalSpec()); len(failures) != 0 {
		t.Fatalf("AuditConstructorReferences(aliased direct call) = %v, want none: the site is static", failures)
	}
}

// TestAuditFailsClosedOnDotImport plants the shape the audit cannot
// classify: a dot import of a watched path makes bare `New("…")`
// unattributable. Pruning it would scope the defect out; failing closed
// forces the import qualified or the audit widened.
func TestAuditFailsClosedOnDotImport(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "dotimport.go", `package plant

import . "errors"

func build(ok bool) error {
	if !ok {
		return New("static detail")
	}
	return nil
}
`)
	failures := AuditConstructorReferences(syntax, fileSet, "dotimport.go", terminalSpec())
	if len(failures) == 0 {
		t.Fatal("AuditConstructorReferences(dot import) = no failure; the unattributable spelling walks through")
	}
	found := false
	for _, failure := range failures {
		if strings.Contains(failure, "dot-import") {
			found = true
		}
	}
	if !found {
		t.Fatalf("AuditConstructorReferences(dot import) = %v, want an unclassifiable-shape failure", failures)
	}
}

// TestAuditRejectsConstructorAsArgument plants the constructor-passed-as-
// value bypass: handing the constructor to another function ships arms
// the derivation never sees. Dropping this direction readmits it.
func TestAuditRejectsConstructorAsArgument(t *testing.T) {
	t.Parallel()

	syntax, fileSet := mustParsePlant(t, "asargument.go", `package plant

func mismatchf(detail string) error { return nil }

func apply(factory func(string) error) error { return factory("static detail") }

func build() error { return apply(mismatchf) }
`)
	if failures := AuditConstructorReferences(syntax, fileSet, "asargument.go", terminalSpec()); len(failures) == 0 {
		t.Fatal("AuditConstructorReferences(constructor as argument) = no failure; the value-passing bypass walks through")
	}
}

// recordViaHelper mimics the instrumented-constructor shape: the swapped
// var calls the helper, the helper calls Record.
func recordViaHelper(recorder *SiteRecorder, code string) {
	recorder.Record(code, 2)
}

// swappedVarMimic is the wrapper level between the call site and the
// helper, matching the swapped constructor var in TestMain.
func swappedVarMimic(recorder *SiteRecorder, code string) {
	recordViaHelper(recorder, code)
}

// TestRecorderAttributesProductionFrame pins the skip convention: a
// record helper called from the swapped constructor var with skip=2
// attributes this test's file (the call site), not the helper or the
// recorder. A wrong skip silently attributes the wrong frame and the
// reverse-direction audit compares garbage.
func TestRecorderAttributesProductionFrame(t *testing.T) {
	t.Parallel()

	recorder := NewSiteRecorder()
	swappedVarMimic(recorder, "refused")
	sites := recorder.Sites()
	if len(sites) != 1 {
		t.Fatalf("Sites() = %v, want exactly one attributed site", sites)
	}
	for site := range sites {
		if !strings.HasPrefix(site, "invcore_test.go:") {
			t.Fatalf("Sites() = %v, want this test's file:line, not the helper frame", sites)
		}
	}
}

// TestRecorderAuditRequiresBothDirections plants a derived site the run
// never exercised and requires the runtime audit to fail forward, plus an
// exercised site outside the derivation to fail in reverse.
func TestRecorderAuditRequiresBothDirections(t *testing.T) {
	t.Parallel()

	derived := map[string]struct{}{"prod.go:10": {}, "prod.go:20": {}}
	recorder := NewSiteRecorder()
	recorder.sites["prod.go:10"] = struct{}{}
	recorder.sites["prod.go:99"] = struct{}{}
	recorder.codes["refused"] = struct{}{}
	failures := recorder.AuditSites("plant", 1, derived, []string{"refused"})
	if len(failures) != 2 {
		t.Fatalf("AuditSites = %v, want exactly the missing-site and outside-site failures", failures)
	}
}

// TestRecorderAuditFailsClosedOnBlindScan requires empty scanned and
// derived inputs to fail even with no exercised sites.
func TestRecorderAuditFailsClosedOnBlindScan(t *testing.T) {
	t.Parallel()

	recorder := NewSiteRecorder()
	if failures := recorder.AuditSites("plant", 0, map[string]struct{}{}, nil); len(failures) == 0 {
		t.Fatal("AuditSites(blind) = no failure, want fail-closed")
	}
}

// TestRecorderAuditPinsTheCodeSet requires an observed code outside the
// closed set to fail: a new refusal code without a ledgered row ships
// unwitnessed otherwise.
func TestRecorderAuditPinsTheCodeSet(t *testing.T) {
	t.Parallel()

	recorder := NewSiteRecorder()
	recorder.sites["prod.go:10"] = struct{}{}
	recorder.codes["refused"] = struct{}{}
	recorder.codes["brand_new_code"] = struct{}{}
	derived := map[string]struct{}{"prod.go:10": {}}
	failures := recorder.AuditSites("plant", 1, derived, []string{"refused"})
	if len(failures) != 1 || !strings.Contains(failures[0], "brand_new_code") {
		t.Fatalf("AuditSites = %v, want exactly the code-set failure naming the new code", failures)
	}
}

// TestQualifiedWatchesAdmitsPrefixSpellings pins the Errorf* union member
// the terminalbackend watch list carries: a trailing-star spelling matches
// every member sharing the stem and nothing else. Dropping the wildcard
// direction leaves the DigestFile fmt.Errorf sites unattributed without a
// single test reddening, so the star has its own pin, not just the exact
// Errorf plant in TestAuditAdmitsDirectCalls.
func TestQualifiedWatchesAdmitsPrefixSpellings(t *testing.T) {
	t.Parallel()

	star := ConstructorSpec{Qualified: map[string]map[string]bool{"fmt": {"Errorf*": true}}}
	for _, member := range []string{"Errorf", "ErrorfWrap"} {
		if !qualifiedWatches(star, "fmt", member) {
			t.Errorf("qualifiedWatches(Errorf* spec, fmt.%s) = false, want true: the star carries the stem", member)
		}
	}
	for _, member := range []string{"New", "Sprintf", "Error"} {
		if qualifiedWatches(star, "fmt", member) {
			t.Errorf("qualifiedWatches(Errorf* spec, fmt.%s) = true, want false: the star is a stem, not a globe", member)
		}
	}
	exact := ConstructorSpec{Qualified: map[string]map[string]bool{"fmt": {"Errorf": true}}}
	if qualifiedWatches(exact, "fmt", "ErrorfWrap") {
		t.Error("qualifiedWatches(exact Errorf spec, fmt.ErrorfWrap) = true, want false: only the star admits longer stems")
	}
}
