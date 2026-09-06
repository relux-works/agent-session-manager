package sessadapter

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// This file is the closed-vocabulary census. It exists because a
// new closed set in production — a map consulted for membership,
// a slice looped for admission, a switch classifying inputs, an
// inline comparison chain admitting two names — passes every
// behavioral test that never names it. The census finds the
// membership surface by parsing production source and fails
// closed on anything with no registered derivation, so the
// surface cannot grow something unwitnessed.
//
// Five checks:
// TestClosedVocabularyTablesAreRegistered scans every
// package-level var shaped like a vocabulary — a map or slice
// composite, written directly or through a named slice or map
// type declared anywhere in the package — and requires each in
// the registration table below with its consultation proof.
// TestNoUnregisteredInlineVocabularies forbids the shape the
// table scan cannot see: a boolean chain comparing one operand
// against two or more closed values (`v == "a" || v == "b"`,
// `v != "a" && v != "b"`, and the same shapes through constants
// or conversions). Every closed union lives in a registered
// table; a chain is an unregistered union and fails here.
// TestClosedMemberSetsAreDerivedFromSpec parses every closed
// body from the pinned document and requires exact ordered
// equality with the production member tables: a dropped, added,
// or reordered member reddens here.
// TestAllProductionSwitchesAreClassified allows only switches
// dispatching on the validated operation plus the tagless UTF-16
// hex classifier inside readUTF16Escape, named by function; any
// other switch — including a tagless switch anywhere else — is a
// membership gate hiding spot and fails.
// TestValueVocabulariesMatchSpec pins every semantic vocabulary
// against its verbatim spec union as a set.
//
// Stated bound: the scanner is syntactic. It sees package-level
// composite shapes (single, shared, and paired initializers),
// named-type aliases, and comparison chains — not function-local
// composites, not unions encoded in a const string or a regexp
// alternation, and not sets built at runtime (a map populated in
// a loop, a registry assembled in init). This package holds no
// const- or regexp-encoded union and builds no vocabulary at
// runtime: every registered table is a package-level composite
// literal, which the scan asserts by construction — a
// runtime-built set would have to replace a composite the
// census already pins. The five function-local obligation lists
// (the validate target list, the two validate check triples,
// the two target-write lists) are outside the table scan and
// are pinned behaviourally instead, in both directions, by the
// obligation tests.
//
// An empty scan, an empty derivation, or an unparseable file
// fails closed throughout: a domain that silently derives
// nothing is not a measurement.

// registeredVocabulary is one production vocabulary table with
// its consultation proof: the production function or call
// pattern that consults it for membership.
type registeredVocabulary struct {
	name       string
	consulted  string
	specMember string
}

// vocabularyRegistrations lists every package-level vocabulary
// table. A production table with no row here fails the census.
var vocabularyRegistrations = []registeredVocabulary{
	{name: "operationOrder", consulted: "validOperation", specMember: "Section 7.8 operation table"},
	{name: "requestMembers", consulted: "unknownMember", specMember: "Section 7.8 request envelope sentence"},
	{name: "requestRequired", consulted: "missingMember", specMember: "Section 7.8 request envelope sentence"},
	{name: "successMembers", consulted: "unknownMember", specMember: "Section 7.8 success envelope rule"},
	{name: "successRequired", consulted: "missingMember", specMember: "Section 7.8 success envelope rule"},
	{name: "failureMembers", consulted: "unknownMember", specMember: "Section 7.8 failure envelope rule"},
	{name: "failureRequired", consulted: "missingMember", specMember: "Section 7.8 failure envelope rule"},
	{name: "capabilityOrder", consulted: "decodeCapabilities", specMember: "Section 7.8 capability sentence"},
	{name: "manifestMembers", consulted: "unknownMember", specMember: "Section 7.8 manifest table"},
	{name: "manifestRequired", consulted: "missingMember", specMember: "Section 7.8 manifest table"},
	{name: "probeMembers", consulted: "unknownMember", specMember: "Section 7.8 probe sentence"},
	{name: "probeRequired", consulted: "missingMember", specMember: "Section 7.8 probe sentence"},
	{name: "capabilityValueMembers", consulted: "unknownMember", specMember: "Section 7.8 capability value sentence"},
	{name: "capabilityValueRequired", consulted: "missingMember", specMember: "Section 7.8 capability value sentence"},
	{name: "capabilityEvidences", consulted: "validCapabilityEvidence", specMember: "Section 7.8 evidence union"},
	{name: "capabilityStatuses", consulted: "validCapabilityStatus", specMember: "Section 7.8 status union"},
	{name: "contextMembers", consulted: "unknownMember", specMember: "Section 7.8 AdapterCallContext sentence"},
	{name: "contextRequired", consulted: "missingMember", specMember: "Section 7.8 AdapterCallContext sentence"},
	{name: "readAuthorityMembers", consulted: "unknownMember", specMember: "Section 7.8 ReadAuthority sentence"},
	{name: "readAuthorityRequired", consulted: "missingMember", specMember: "Section 7.8 ReadAuthority sentence"},
	{name: "objectAuthorityMembers", consulted: "unknownMember", specMember: "Section 7.8 ObjectAuthority sentence"},
	{name: "objectAuthorityRequired", consulted: "missingMember", specMember: "Section 7.8 ObjectAuthority sentence"},
	{name: "objectPurposes", consulted: "validObjectPurpose", specMember: "Section 7.8 purpose union"},
	{name: "objectModes", consulted: "validObjectMode", specMember: "Section 7.8 ObjectAuthority mode union"},
	{name: "readPurposes", consulted: "validReadPurpose", specMember: "Section 7.8 ReadAuthority purpose union"},
	{name: "findingSeverities", consulted: "validFindingSeverity", specMember: "Section 7.8 severity union"},
	{name: "sourceSelectorMembers", consulted: "unknownMember", specMember: "Section 7.8 SourceSelector sentence"},
	{name: "sourceSelectorRequired", consulted: "missingMember", specMember: "Section 7.8 SourceSelector sentence"},
	{name: "resourceLimitsMembers", consulted: "unknownMember", specMember: "Section 7.8 ResourceLimits sentence"},
	{name: "resourceLimitsRequired", consulted: "missingMember", specMember: "Section 7.8 ResourceLimits sentence"},
	{name: "findingMembers", consulted: "unknownMember", specMember: "Section 7.8 AdapterFinding sentence"},
	{name: "findingRequired", consulted: "missingMember", specMember: "Section 7.8 AdapterFinding sentence"},
	{name: "capturePlanItemMembers", consulted: "unknownMember", specMember: "Section 7.8 CapturePlanItem sentence"},
	{name: "capturePlanItemRequired", consulted: "missingMember", specMember: "Section 7.8 CapturePlanItem sentence"},
	{name: "tupleMembers", consulted: "unknownMember", specMember: "Section 7.8 Environment Tuple sentence"},
	{name: "tupleRequired", consulted: "missingMember", specMember: "Section 7.8 Environment Tuple sentence"},
	{name: "projectionStrategies", consulted: "validStrategy", specMember: "Section 13.14 strategy sentence"},
	{name: "fidelityDispositions", consulted: "validDisposition", specMember: "Section 13.14 disposition sentence"},
	{name: "captureClasses", consulted: "validCaptureClass", specMember: "Section 13.14 CaptureItem class column"},
	{name: "tupleArchitectures", consulted: "validTupleArchitecture", specMember: "Section 7.8 architecture union"},
	{name: "tupleKeyMembers", consulted: "unknownMember", specMember: "Section 13.14 key sentence"},
	{name: "tupleKeyRequired", consulted: "missingMember", specMember: "Section 13.14 key sentence"},
	{name: "tupleEntryMembers", consulted: "unknownMember", specMember: "Section 13.14 entry sentence"},
	{name: "tupleEntryRequired", consulted: "missingMember", specMember: "Section 13.14 entry sentence"},
	{name: "fixtureEvidenceMembers", consulted: "unknownMember", specMember: "Section 13.14 FixtureEvidence sentence"},
	{name: "fixtureEvidenceRequired", consulted: "missingMember", specMember: "Section 13.14 FixtureEvidence sentence"},
	{name: "smokeEvidenceMembers", consulted: "unknownMember", specMember: "Section 13.14 ResumeSmokeEvidence sentence"},
	{name: "smokeEvidenceRequired", consulted: "missingMember", specMember: "Section 13.14 ResumeSmokeEvidence sentence"},
	{name: "fidelityLimitMembers", consulted: "unknownMember", specMember: "Section 13.14 FidelityLimit sentence"},
	{name: "fidelityLimitRequired", consulted: "missingMember", specMember: "Section 13.14 FidelityLimit sentence"},
	{name: "requestBodyMembers", consulted: "CheckRequestBody", specMember: "Section 7.8 operation table"},
	{name: "successBodyMembers", consulted: "CheckSuccessBody", specMember: "Section 7.8 operation table"},
	{name: "fidelityProfiles", consulted: "validFidelityProfile", specMember: "Section 7.8 profile union"},
	{name: "validateModes", consulted: "validValidateMode", specMember: "Section 7.8 validate row"},
	{name: "contextFreeOperations", consulted: "operationSkipsRequestContext", specMember: "Section 7.8 operation table (context-free request bodies)"},
	{name: "doctorResultMembers", consulted: "unknownMember", specMember: "Section 7.8 doctor row"},
	{name: "doctorResultRequired", consulted: "missingMember", specMember: "Section 7.8 doctor row"},
	{name: "directionNames", consulted: "validDirection", specMember: "Section 13.14 key direction union and Section 7.8 doctor row"},
	{name: "roleNames", consulted: "validRole", specMember: "Section 7.8 binding role union"},
	{name: "candidateKindNames", consulted: "validCandidateKind", specMember: "Section 7.8 candidate-kind union"},
	{name: "tupleEntryStatuses", consulted: "validTupleEntryStatus", specMember: "Section 13.14 entry status union"},
	{name: "evidenceResults", consulted: "validEvidenceResult", specMember: "Sections 13.14 result=pass"},
	{name: "registryEntryStatuses", consulted: "validRegistryEntryStatus", specMember: "Section 7.8 doctor row"},
}

// TestClosedVocabularyTablesAreRegistered is the census: every
// package-level vocabulary-shaped var must carry a registration
// row naming its consultation. A new table without a row fails
// here rather than passing silently.
func TestClosedVocabularyTablesAreRegistered(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	registered := map[string]registeredVocabulary{}
	for _, entry := range vocabularyRegistrations {
		if _, duplicate := registered[entry.name]; duplicate {
			t.Fatalf("census registers %q twice", entry.name)
		}
		registered[entry.name] = entry
	}
	found, scanned := scanVocabularyTables(t, directory)
	if len(scanned) == 0 {
		t.Fatal("census scanned zero production files; the scanner is broken, not the package")
	}
	if len(found) == 0 {
		t.Fatal("census found zero vocabulary tables; the scanner is broken, not the package")
	}
	var unregistered []string
	for _, name := range found {
		entry, ok := registered[name]
		if !ok {
			unregistered = append(unregistered, name)
			continue
		}
		if entry.consulted == "" || entry.specMember == "" {
			t.Errorf("census registration of %q names no consultation or spec member", name)
		}
	}
	sort.Strings(unregistered)
	if len(unregistered) > 0 {
		t.Fatalf("production vocabulary table(s) with no census registration:\n  %s", strings.Join(unregistered, "\n  "))
	}
	var orphaned []string
	for name := range registered {
		present := false
		for _, table := range found {
			if table == name {
				present = true
				break
			}
		}
		if !present {
			orphaned = append(orphaned, name)
		}
	}
	sort.Strings(orphaned)
	if len(orphaned) > 0 {
		t.Fatalf("census registration(s) naming no production table:\n  %s", strings.Join(orphaned, "\n  "))
	}
	t.Logf("vocabulary census: %d tables registered across %d files", len(found), len(scanned))
}

// scanVocabularyTables returns every package-level var shaped
// like a vocabulary: a map or slice composite, written directly
// or through a named slice or map type declared anywhere in the
// package. A named-type table (`type names []string` plus
// `var t = names{...}`) is the same closed set as a literal one,
// and the census sees both.
func scanVocabularyTables(t *testing.T, directory string) ([]string, []string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	type namedFile struct {
		name   string
		syntax *ast.File
	}
	var files []namedFile
	var scanned []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned = append(scanned, name)
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("census: %v", err)
		}
		syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			t.Fatalf("census: %v", err)
		}
		files = append(files, namedFile{name: name, syntax: syntax})
	}
	named := map[string]ast.Expr{}
	for _, file := range files {
		collectNamedCompositeTypes(file.syntax, named)
	}
	var found []string
	for _, file := range files {
		found = append(found, vocabularyTablesInFile(file.syntax, named)...)
	}
	return found, scanned
}

// collectNamedCompositeTypes records every package-level type
// name declared with a composite underlying type: a direct map or
// slice, or another named type resolved later. Struct, string,
// and other underlying types are recorded too, so a struct
// literal through a named type is never mistaken for a table.
func collectNamedCompositeTypes(syntax *ast.File, named map[string]ast.Expr) {
	for _, declaration := range syntax.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			alias, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			named[alias.Name.Name] = alias.Type
		}
	}
}

// vocabularyTablesInFile returns the package-level vars of one
// file whose initializer is a vocabulary composite. One shared
// initializer serves every name; paired initializers serve their
// own name each (`var a, b = []string{...}, []string{...}`), so
// a multi-name declaration never hides a table from the scan.
func vocabularyTablesInFile(syntax *ast.File, named map[string]ast.Expr) []string {
	var found []string
	for _, declaration := range syntax.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Values) == 0 {
				continue
			}
			for index, ident := range value.Names {
				var initializer ast.Expr
				switch {
				case len(value.Values) == 1:
					initializer = value.Values[0]
				case index < len(value.Values):
					initializer = value.Values[index]
				default:
					continue
				}
				if isVocabularyShape(initializer, named) {
					found = append(found, ident.Name)
				}
			}
		}
	}
	return found
}

// isVocabularyShape reports whether the initializer is a
// vocabulary composite: a map or slice literal, or a composite
// through a named type whose underlying type — resolved across
// the package, through any alias chain — is a map or slice.
// Element type is irrelevant; registrations name the table, not
// its shape. A composite through any other named type (a struct
// literal, a string conversion) is not a table.
func isVocabularyShape(node ast.Expr, named map[string]ast.Expr) bool {
	literal, ok := node.(*ast.CompositeLit)
	if !ok {
		return false
	}
	switch typ := literal.Type.(type) {
	case *ast.MapType, *ast.ArrayType:
		return true
	case *ast.Ident:
		return resolvesToComposite(typ.Name, named, 0)
	}
	return false
}

// resolvesToComposite reports whether the named type ultimately
// denotes a map or slice. Alias chains resolve up to a fixed
// depth; a cycle or an unknown name is not a table.
func resolvesToComposite(name string, named map[string]ast.Expr, depth int) bool {
	if depth > 8 {
		return false
	}
	underlying, ok := named[name]
	if !ok {
		return false
	}
	switch typ := underlying.(type) {
	case *ast.MapType, *ast.ArrayType:
		return true
	case *ast.Ident:
		return resolvesToComposite(typ.Name, named, depth+1)
	}
	return false
}

// TestAllProductionSwitchesAreClassified allows only switches
// dispatching on the validated operation plus the tagless UTF-16
// hex classifier inside readUTF16Escape. Any other switch —
// including a tagless switch in any other function — is a
// membership gate hiding spot and fails: membership lives in
// tables the census sees, never in case arms it cannot.
func TestAllProductionSwitchesAreClassified(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("switch census: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("switch census: %v", err)
	}
	scanned := 0
	classified := 0
	var unclassified []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("switch census: %v", err)
		}
		syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			t.Fatalf("switch census: %v", err)
		}
		var stack []ast.Node
		ast.Inspect(syntax, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if statement, ok := node.(*ast.SwitchStmt); ok {
				position := filepath.Base(path)
				if classifySwitch(statement.Tag, enclosingFunc(stack)) {
					classified++
				} else {
					unclassified = append(unclassified, position)
				}
			}
			stack = append(stack, node)
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("switch census scanned zero production files")
	}
	sort.Strings(unclassified)
	if len(unclassified) > 0 {
		t.Fatalf("unclassified production switch(es): %s", strings.Join(unclassified, ", "))
	}
	t.Logf("switch census: %d classified switches across %d files", classified, scanned)
}

// enclosingFunc names the innermost function declaration above
// the current walk position, or "" outside any function. The
// switch census uses it to confine the tagless-switch exemption
// to readUTF16Escape by name rather than exempting the shape.
func enclosingFunc(stack []ast.Node) string {
	for index := len(stack) - 1; index >= 0; index-- {
		if declaration, ok := stack[index].(*ast.FuncDecl); ok {
			return declaration.Name.Name
		}
	}
	return ""
}

// classifySwitch reports whether a switch with the given tag in
// the named function is an allowed classifier: a dispatch on
// the validated operation, or the tagless UTF-16 hex classifier
// inside readUTF16Escape — which switches over digit ranges and
// classifies characters rather than admitting members. Only
// this function earns the tagless exemption; a tagless switch
// anywhere else is a membership gate hiding spot.
func classifySwitch(tag ast.Expr, function string) bool {
	if ident, ok := tag.(*ast.Ident); ok && ident.Name == "operation" {
		return true
	}
	return tag == nil && function == "readUTF16Escape"
}

// TestSwitchClassifierNamesItsExemption proves the switch gate
// shape against synthetic inputs: the operation dispatch passes
// anywhere, the tagless classifier passes only inside
// readUTF16Escape, and a tagless switch in any other function
// fails — the sentence the census header states.
func TestSwitchClassifierNamesItsExemption(t *testing.T) {
	operation := &ast.Ident{Name: "operation"}
	other := &ast.Ident{Name: "mode"}
	for _, probe := range []struct {
		name     string
		tag      ast.Expr
		function string
		want     bool
	}{
		{"operation dispatch passes", operation, "checkRequestScalars", true},
		{"tagless classifier passes in readUTF16Escape", nil, "readUTF16Escape", true},
		{"tagless switch fails elsewhere", nil, "CheckSuccessBody", false},
		{"other tag fails", other, "checkRequestScalars", false},
	} {
		t.Run(probe.name, func(t *testing.T) {
			if got := classifySwitch(probe.tag, probe.function); got != probe.want {
				t.Fatalf("classifySwitch(tag, %q) = %v, want %v", probe.function, got, probe.want)
			}
		})
	}
}

// TestNoUnregisteredInlineVocabularies is the chain census: no
// production boolean chain may compare one operand against two
// or more closed values. `v == "a" || v == "b"`,
// `v != "a" && v != "b"`, and the same shapes through constants
// (`role != RoleSource && role != RoleTarget`) or conversions are
// unions outside every table, and a widened chain survives every
// behavioral test that never names the new member. Single
// comparisons — one dispatch member (`mode == "archive"`), one
// sentinel (`== ""`), one echo — are not unions and pass.
func TestNoUnregisteredInlineVocabularies(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("chain census: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("chain census: %v", err)
	}
	scanned := 0
	var unregistered []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("chain census: %v", err)
		}
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			t.Fatalf("chain census: %v", err)
		}
		for _, chain := range inlineVocabulariesInFile(fileSet, syntax) {
			unregistered = append(unregistered, name+": "+chain)
		}
	}
	if scanned == 0 {
		t.Fatal("chain census scanned zero production files")
	}
	sort.Strings(unregistered)
	if len(unregistered) > 0 {
		t.Fatalf("unregistered inline vocabular%s:\n  %s",
			map[bool]string{true: "y", false: "ies"}[len(unregistered) == 1],
			strings.Join(unregistered, "\n  "))
	}
	t.Logf("chain census: no inline vocabularies across %d files", scanned)
}

// inlineChainComparison is one equality comparison inside a
// boolean chain: the rendered operand plus the closed value it is
// compared against.
type inlineChainComparison struct {
	operand string
	value   string
}

// inlineVocabulariesInFile reports every operand compared against
// two or more distinct closed values inside one boolean chain of
// the file, as "operand <- value, value" strings. Groups merge
// only within a single chain subtree, never across statements.
func inlineVocabulariesInFile(fileSet *token.FileSet, syntax *ast.File) []string {
	seen := map[string]struct{}{}
	var unregistered []string
	ast.Inspect(syntax, func(node ast.Node) bool {
		chain, ok := node.(*ast.BinaryExpr)
		if !ok || (chain.Op != token.LOR && chain.Op != token.LAND) {
			return true
		}
		var comparisons []inlineChainComparison
		flattenChainComparisons(fileSet, chain, &comparisons)
		groups := map[string]map[string]struct{}{}
		for _, comparison := range comparisons {
			if comparison.operand == "" || comparison.value == "" {
				continue
			}
			set := groups[comparison.operand]
			if set == nil {
				set = map[string]struct{}{}
				groups[comparison.operand] = set
			}
			set[comparison.value] = struct{}{}
		}
		for operand, values := range groups {
			if len(values) < 2 {
				continue
			}
			var names []string
			for value := range values {
				names = append(names, value)
			}
			sort.Strings(names)
			key := operand + " <- " + strings.Join(names, ", ")
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			position := fileSet.Position(chain.Pos())
			unregistered = append(unregistered, fmt.Sprintf("%s compares %s", position, key))
		}
		return true
	})
	sort.Strings(unregistered)
	return unregistered
}

// flattenChainComparisons collects every equality comparison in
// one boolean-chain subtree, descending through ||, &&, and
// parentheses. Anything else contributes only itself when it is
// a qualifying comparison.
func flattenChainComparisons(fileSet *token.FileSet, node ast.Node, out *[]inlineChainComparison) {
	switch node := node.(type) {
	case *ast.ParenExpr:
		flattenChainComparisons(fileSet, node.X, out)
	case *ast.BinaryExpr:
		if node.Op == token.LOR || node.Op == token.LAND {
			flattenChainComparisons(fileSet, node.X, out)
			flattenChainComparisons(fileSet, node.Y, out)
			return
		}
		if node.Op == token.EQL || node.Op == token.NEQ {
			*out = append(*out, chainComparisons(fileSet, node)...)
		}
	}
}

// chainComparisons renders one equality comparison as operand
// comparisons. A string literal on either side makes the other
// side the operand. Two bare identifiers record both
// orientations, because either side may be the closed constant —
// except the predeclared nil, true, and false, which are presence
// checks, never union members. A bare identifier against a field,
// conversion, or call result makes the composite the operand and
// the identifier the value. Anything else is not a closed-value
// comparison: numeric bounds, rune classifiers, nil presence,
// and field-to-field equality all pass through uncounted.
func chainComparisons(fileSet *token.FileSet, comparison *ast.BinaryExpr) []inlineChainComparison {
	left, right := unwrapParens(comparison.X), unwrapParens(comparison.Y)
	if value, ok := closedValue(left); ok {
		if operand := renderExpr(fileSet, right); operand != "" {
			return []inlineChainComparison{{operand: operand, value: value}}
		}
		return nil
	}
	if value, ok := closedValue(right); ok {
		if operand := renderExpr(fileSet, left); operand != "" {
			return []inlineChainComparison{{operand: operand, value: value}}
		}
		return nil
	}
	leftIdent, leftIsIdent := left.(*ast.Ident)
	rightIdent, rightIsIdent := right.(*ast.Ident)
	if leftIsIdent && rightIsIdent {
		if predeclared(leftIdent.Name) || predeclared(rightIdent.Name) {
			return nil
		}
		return []inlineChainComparison{
			{operand: leftIdent.Name, value: rightIdent.Name},
			{operand: rightIdent.Name, value: leftIdent.Name},
		}
	}
	if leftIsIdent && isCompositeOperand(right) {
		if operand := renderExpr(fileSet, right); operand != "" {
			return []inlineChainComparison{{operand: operand, value: leftIdent.Name}}
		}
		return nil
	}
	if rightIsIdent && isCompositeOperand(left) {
		if operand := renderExpr(fileSet, left); operand != "" {
			return []inlineChainComparison{{operand: operand, value: rightIdent.Name}}
		}
		return nil
	}
	return nil
}

// unwrapParens strips parenthesized wrappers so a parenthesized
// operand still groups with its bare spelling.
func unwrapParens(node ast.Expr) ast.Expr {
	for {
		parenthesized, ok := node.(*ast.ParenExpr)
		if !ok {
			return node
		}
		node = parenthesized.X
	}
}

// predeclared reports whether the identifier is a predeclared
// constant: a nil, true, or false comparison is a presence or
// flag check, never a closed-union member.
func predeclared(name string) bool {
	return name == "nil" || name == "true" || name == "false"
}

// isCompositeOperand reports whether the expression is a field,
// conversion, or call result: the shapes a closed constant is
// compared against. Literals, identifiers, and composite
// literals are classified elsewhere or not at all.
func isCompositeOperand(node ast.Expr) bool {
	switch node.(type) {
	case *ast.SelectorExpr, *ast.CallExpr, *ast.IndexExpr:
		return true
	}
	return false
}

// closedValue returns the closed value of a string literal
// operand, unquoted so equivalent spellings group together.
func closedValue(node ast.Expr) (string, bool) {
	literal, ok := node.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	if value, err := strconv.Unquote(literal.Value); err == nil {
		return value, true
	}
	return literal.Value, true
}

// renderExpr renders one operand to its source text for grouping:
// two comparisons group only when their operands spell
// identically.
func renderExpr(fileSet *token.FileSet, node ast.Node) string {
	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, fileSet, node); err != nil {
		return ""
	}
	return buffer.String()
}

// TestInlineVocabularyDetectorSeesBothShapes proves the table
// scan against synthetic sources: a named-type table resolves
// through the alias, a struct literal through a named type does
// not, multi-name declarations resolve every name (shared,
// paired, and mixed initializers), an inline chain in either
// boolean shape flags, and a single comparison passes. The
// vectors are synthetic on purpose: they prove the scanner, not
// production.
func TestInlineVocabularyDetectorSeesBothShapes(t *testing.T) {
	parse := func(source string) (*token.FileSet, *ast.File) {
		t.Helper()
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return fileSet, syntax
	}
	t.Run("named type table resolves", func(t *testing.T) {
		_, syntax := parse("package probe\ntype names []string\nvar t = names{\"a\", \"b\"}\n")
		named := map[string]ast.Expr{}
		collectNamedCompositeTypes(syntax, named)
		tables := vocabularyTablesInFile(syntax, named)
		if len(tables) != 1 || tables[0] != "t" {
			t.Fatalf("named-type table = %v, want [t]", tables)
		}
	})
	t.Run("named type alias chain resolves", func(t *testing.T) {
		_, syntax := parse("package probe\ntype a []string\ntype b a\nvar t = b{\"a\"}\n")
		named := map[string]ast.Expr{}
		collectNamedCompositeTypes(syntax, named)
		tables := vocabularyTablesInFile(syntax, named)
		if len(tables) != 1 || tables[0] != "t" {
			t.Fatalf("aliased table = %v, want [t]", tables)
		}
	})
	t.Run("struct literal through named type is not a table", func(t *testing.T) {
		_, syntax := parse("package probe\ntype point struct{ x int }\nvar p = point{1}\n")
		named := map[string]ast.Expr{}
		collectNamedCompositeTypes(syntax, named)
		if tables := vocabularyTablesInFile(syntax, named); len(tables) != 0 {
			t.Fatalf("struct literal = %v, want no tables", tables)
		}
	})
	t.Run("multi-name shared table resolves both names", func(t *testing.T) {
		_, syntax := parse("package probe\nvar a, b = []string{\"x\", \"y\"}\n")
		named := map[string]ast.Expr{}
		collectNamedCompositeTypes(syntax, named)
		tables := vocabularyTablesInFile(syntax, named)
		sort.Strings(tables)
		if len(tables) != 2 || tables[0] != "a" || tables[1] != "b" {
			t.Fatalf("shared table = %v, want [a b]", tables)
		}
	})
	t.Run("multi-name paired tables resolve each name", func(t *testing.T) {
		_, syntax := parse("package probe\nvar a, b = []string{\"x\"}, []string{\"y\"}\n")
		named := map[string]ast.Expr{}
		collectNamedCompositeTypes(syntax, named)
		tables := vocabularyTablesInFile(syntax, named)
		sort.Strings(tables)
		if len(tables) != 2 || tables[0] != "a" || tables[1] != "b" {
			t.Fatalf("paired tables = %v, want [a b]", tables)
		}
	})
	t.Run("multi-name mixed pair resolves only the table", func(t *testing.T) {
		_, syntax := parse("package probe\ntype point struct{ x int }\nvar a, b = point{1}, []string{\"y\"}\n")
		named := map[string]ast.Expr{}
		collectNamedCompositeTypes(syntax, named)
		tables := vocabularyTablesInFile(syntax, named)
		if len(tables) != 1 || tables[0] != "b" {
			t.Fatalf("mixed pair = %v, want [b]", tables)
		}
	})
	t.Run("or chain of literals flags", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f(v string) bool { return v == \"a\" || v == \"b\" }\n")
		if chains := inlineVocabulariesInFile(fileSet, syntax); len(chains) != 1 {
			t.Fatalf("or chain = %v, want one unregistered vocabulary", chains)
		}
	})
	t.Run("and chain of negations flags", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f(v string) bool { return v != \"a\" && v != \"b\" }\n")
		if chains := inlineVocabulariesInFile(fileSet, syntax); len(chains) != 1 {
			t.Fatalf("and chain = %v, want one unregistered vocabulary", chains)
		}
	})
	t.Run("chain through constants flags", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f(v string) bool { return v != A && v != B }\n")
		if chains := inlineVocabulariesInFile(fileSet, syntax); len(chains) != 1 {
			t.Fatalf("constant chain = %v, want one unregistered vocabulary", chains)
		}
	})
	t.Run("single comparison passes", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f(v string) bool { return v == \"a\" }\n")
		if chains := inlineVocabulariesInFile(fileSet, syntax); len(chains) != 0 {
			t.Fatalf("single comparison = %v, want no unregistered vocabulary", chains)
		}
	})
	t.Run("field to field equality passes", func(t *testing.T) {
		fileSet, syntax := parse("package probe\nfunc f(a, b point) bool { return a.x != b.x || a.y != b.y }\n")
		if chains := inlineVocabulariesInFile(fileSet, syntax); len(chains) != 0 {
			t.Fatalf("field equality = %v, want no unregistered vocabulary", chains)
		}
	})
}

// sentenceEnd scans the first sentence terminator in text after a
// member-list marker: the first period outside a code span that is
// not a digit-period-digit abbreviation like Section 7.5 or
// Structured Error 1.1. Normative member sentences never end
// mid-list any other way, so any other period ends the scan.
func sentenceEnd(rest string) int {
	inCode := false
	for offset := 0; offset < len(rest); offset++ {
		if strings.HasPrefix(rest[offset:], "<code>") {
			inCode = true
			continue
		}
		if strings.HasPrefix(rest[offset:], "</code>") {
			inCode = false
			continue
		}
		if rest[offset] != '.' || inCode {
			continue
		}
		previous, next := byte(0), byte(0)
		if offset > 0 {
			previous = rest[offset-1]
		}
		if offset+1 < len(rest) {
			next = rest[offset+1]
		}
		if previous >= '0' && previous <= '9' && next >= '0' && next <= '9' {
			continue
		}
		return offset
	}
	return -1
}

// sentenceMembers collects the member names of one scanned
// sentence: every code span contributes the name before its first
// colon or equals sign, in document order.
func sentenceMembers(t *testing.T, marker, sentence string) []string {
	t.Helper()
	var members []string
	for {
		open := strings.Index(sentence, "<code>")
		if open < 0 {
			break
		}
		sentence = sentence[open+len("<code>"):]
		close := strings.Index(sentence, "</code>")
		if close < 0 {
			t.Fatalf("unbalanced code span after %q", marker)
		}
		span := sentence[:close]
		sentence = sentence[close+len("</code>"):]
		name := span
		if index := strings.IndexAny(span, ":="); index >= 0 {
			name = span[:index]
		}
		members = append(members, name)
	}
	if len(members) == 0 {
		t.Fatalf("derived no members for %q; the parser is broken", marker)
	}
	return members
}

// deriveTypeMembers parses one "<code>Type</code> contains
// exactly ..." sentence from the pinned document into its member
// list in document order. A missing marker or an empty derivation
// fails closed.
func deriveTypeMembers(t *testing.T, text, marker string) []string {
	t.Helper()
	return deriveTypeMembersWith(t, text, marker, "")
}

// deriveTypeMembersWith selects among repeated homonymous
// sentences: the pinned document states the request envelope and
// the capability value in more than one section with different
// members, so the selector names a structural fragment only the
// Session Adapter sentence carries. An empty selector keeps the
// first occurrence. The derivation still parses the whole
// selected sentence; no production value is matched.
func deriveTypeMembersWith(t *testing.T, text, marker, selector string) []string {
	t.Helper()
	remaining := text
	for {
		index := strings.Index(remaining, marker)
		if index < 0 {
			t.Fatalf("spec marker %q with selector %q not found; the derivation is broken", marker, selector)
		}
		rest := remaining[index+len(marker):]
		end := sentenceEnd(rest)
		if end < 0 {
			t.Fatalf("spec sentence for %q never terminates", marker)
		}
		sentence := rest[:end]
		if selector == "" || strings.Contains(sentence, selector) {
			return sentenceMembers(t, marker, sentence)
		}
		remaining = rest
	}
}

// memberSetKeys returns the keys of one production member map.
func memberSetKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for name := range set {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

// TestClosedMemberSetsAreDerivedFromSpec requires every closed
// body to equal its pinned definition in order: the production
// Required list must match the document sequence exactly, and
// the Members map must hold exactly the same names. A dropped,
// added, or reordered member reddens here. Both halves matter:
// the Required list drives the missing-member refusal, the
// Members map drives the unknown-member gate, and pinning only
// the list leaves a widened map admitting an extra member with
// every test green. SupportedContractVersions carries no
// production gate map, so its row pins the Required list only.
func TestClosedMemberSetsAreDerivedFromSpec(t *testing.T) {
	if _, err := specdoc.Load(); err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	text := string(specdoc.Bytes())
	// selectors disambiguate the two member sentences the pinned
	// document repeats across sections with different members: the
	// request envelope (Sections 7.5-style, 7.8, and 7.9 each state
	// one) and the capability value (the Provider Probe and Session
	// Adapter Probe sentences differ). Each selector is a structural
	// fragment only the Session Adapter sentence carries.
	selectors := map[string]string{
		"request envelope": "<code>deadline</code>, and <code>body</code>",
		"capability value": "<code>detail:string[0..2048]</code>",
	}
	ordered := []struct {
		name     string
		marker   string
		required []string
		members  map[string]bool
	}{
		{"request envelope", "The request envelope contains exactly", requestRequired, requestMembers},
		{"AdapterCallContext", "<code>AdapterCallContext</code> contains exactly", contextRequired, contextMembers},
		{"ReadAuthority", "<code>ReadAuthority</code> contains exactly", readAuthorityRequired, readAuthorityMembers},
		{"ObjectAuthority", "<code>ObjectAuthority</code> contains exactly", objectAuthorityRequired, objectAuthorityMembers},
		{"SourceSelector", "<code>SourceSelector</code> contains exactly", sourceSelectorRequired, sourceSelectorMembers},
		{"AdapterFinding", "<code>AdapterFinding</code> contains exactly", findingRequired, findingMembers},
		{"ResourceLimits", "<code>ResourceLimits</code> contains exactly", resourceLimitsRequired, resourceLimitsMembers},
		{"CapturePlanItem", "<code>CapturePlanItem</code> contains exactly", capturePlanItemRequired, capturePlanItemMembers},
		{"Session Adapter Probe", "Session Adapter Probe 1.0.0 is closed and contains exactly", probeRequired, probeMembers},
		{"Environment Tuple", "Environment Tuple contains exactly", tupleRequired, tupleMembers},
		{"SupportedEnvironmentTupleKey", "<code>SupportedEnvironmentTupleKey</code> contains exactly", tupleKeyRequired, tupleKeyMembers},
		{"SupportedEnvironmentTupleEntry", "<code>SupportedEnvironmentTupleEntry</code> contains exactly", tupleEntryRequired, tupleEntryMembers},
		{"SupportedContractVersions", "<code>SupportedContractVersions</code> contains exactly", []string{"contract_id", "versions"}, nil},
		{"FixtureEvidence", "<code>FixtureEvidence</code> contains exactly", fixtureEvidenceRequired, fixtureEvidenceMembers},
		{"ResumeSmokeEvidence", "<code>ResumeSmokeEvidence</code> contains exactly", smokeEvidenceRequired, smokeEvidenceMembers},
		{"FidelityLimit", "<code>FidelityLimit</code> contains exactly", fidelityLimitRequired, fidelityLimitMembers},
		{"capability value", "Each capability value contains exactly", capabilityValueRequired, capabilityValueMembers},
	}
	mapComparisons := 0
	for _, test := range ordered {
		t.Run(test.name, func(t *testing.T) {
			derived := deriveTypeMembersWith(t, text, test.marker, selectors[test.name])
			if strings.Join(derived, ",") != strings.Join(test.required, ",") {
				t.Errorf("%s required = %v, want derived %v", test.name, test.required, derived)
			}
			if test.members != nil {
				mapComparisons++
				derivedSet := append([]string(nil), derived...)
				sort.Strings(derivedSet)
				if strings.Join(derivedSet, ",") != strings.Join(memberSetKeys(test.members), ",") {
					t.Errorf("%s members = %v, want derived %v", test.name, memberSetKeys(test.members), derivedSet)
				}
			}
		})
	}
	// The success and failure envelopes reuse the request's first
	// four identity fields by an explicit rule, so their
	// derivation transforms the request list rather than
	// re-parsing prose. The gate maps are pinned as sets against
	// the same derivation: a widened successMembers admits a
	// frame carrying the planted member while the control
	// refuses, with the Required comparison green.
	request := deriveTypeMembers(t, text, "The request envelope contains exactly")
	firstFour := request[:4]
	wantSuccess := append(append([]string(nil), firstFour...), "ok", "body")
	if strings.Join(wantSuccess, ",") != strings.Join(successRequired, ",") {
		t.Errorf("success required = %v, want %v", successRequired, wantSuccess)
	}
	mapComparisons++
	wantSuccessSet := append([]string(nil), wantSuccess...)
	sort.Strings(wantSuccessSet)
	if strings.Join(wantSuccessSet, ",") != strings.Join(memberSetKeys(successMembers), ",") {
		t.Errorf("success members = %v, want derived %v", memberSetKeys(successMembers), wantSuccessSet)
	}
	wantFailure := append(append([]string(nil), firstFour...), "ok", "error")
	if strings.Join(wantFailure, ",") != strings.Join(failureRequired, ",") {
		t.Errorf("failure required = %v, want %v", failureRequired, wantFailure)
	}
	mapComparisons++
	wantFailureSet := append([]string(nil), wantFailure...)
	sort.Strings(wantFailureSet)
	if strings.Join(wantFailureSet, ",") != strings.Join(memberSetKeys(failureMembers), ",") {
		t.Errorf("failure members = %v, want derived %v", memberSetKeys(failureMembers), wantFailureSet)
	}
	// The manifest member column comes from its own table, not a
	// sentence: parse the first-column code spans in the table
	// window. The local is named derivedManifest so it cannot
	// shadow the production manifestMembers gate map the
	// comparison below must read.
	derivedManifest := deriveManifestTableMembers(t, text)
	if strings.Join(derivedManifest, ",") != strings.Join(manifestRequired, ",") {
		t.Errorf("manifest required = %v, want derived %v", manifestRequired, derivedManifest)
	}
	mapComparisons++
	derivedManifestSet := append([]string(nil), derivedManifest...)
	sort.Strings(derivedManifestSet)
	if strings.Join(derivedManifestSet, ",") != strings.Join(memberSetKeys(manifestMembers), ",") {
		t.Errorf("manifest members = %v, want derived %v", memberSetKeys(manifestMembers), derivedManifestSet)
	}
	// The doctor result members come from the operation table
	// like every other success body.
	_, _, successes := deriveOperationTable(t)
	doctor, ok := successes[string(OpDoctor)]
	if !ok {
		t.Fatal("derived no doctor success body from the operation table")
	}
	if strings.Join(doctor, ",") != strings.Join(doctorResultRequired, ",") {
		t.Errorf("doctor required = %v, want derived %v", doctorResultRequired, doctor)
	}
	mapComparisons++
	doctorSet := append([]string(nil), doctor...)
	sort.Strings(doctorSet)
	if strings.Join(doctorSet, ",") != strings.Join(memberSetKeys(doctorResultMembers), ",") {
		t.Errorf("doctor members = %v, want derived %v", memberSetKeys(doctorResultMembers), doctorSet)
	}
	t.Logf("member-set census: %d member maps pinned across %d bodies", mapComparisons, len(ordered)+4)
}

// deriveManifestTableMembers parses the Session Adapter Manifest
// member column from its table window: rows starting with
// "| <code>" contribute their first-column name in order.
func deriveManifestTableMembers(t *testing.T, text string) []string {
	t.Helper()
	table, ok := sectionTableWindow(text, "Session Adapter Manifest 1.0.0 is closed and contains exactly:", "Session Adapter Probe 1.0.0 is closed and contains exactly")
	if !ok {
		t.Fatal("manifest table window not found")
	}
	var members []string
	for _, line := range strings.Split(table, "\n") {
		if !strings.HasPrefix(line, "| <code>") {
			continue
		}
		cells := splitTableCells(line)
		if len(cells) < 2 {
			continue
		}
		// The first cell carries one member per code span: three
		// rows pair two members with a slash (schema /
		// schema_version, display_name / adapter_version,
		// operations / capability_names). Every span that is a
		// bare snake_case name is a member, in order.
		cell := cells[1]
		for {
			open := strings.Index(cell, "<code>")
			if open < 0 {
				break
			}
			cell = cell[open+len("<code>"):]
			close := strings.Index(cell, "</code>")
			if close < 0 {
				t.Fatal("unbalanced code span in the manifest table")
			}
			name := cell[:close]
			cell = cell[close+len("</code>"):]
			snake := len(name) > 0
			for _, character := range name {
				if character != '_' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
					snake = false
					break
				}
			}
			if snake {
				members = append(members, name)
			}
		}
	}
	if len(members) == 0 {
		t.Fatal("derived no manifest members; the parser is broken")
	}
	return members
}

// TestValueVocabulariesMatchSpec pins every semantic vocabulary
// against its verbatim spec union as a set. Order is not
// normative for these sets — sortedness governs instances, not
// the listing — so both sides sort before comparing. The
// operation and capability registries are order-normative and
// are pinned in order instead.
func TestValueVocabulariesMatchSpec(t *testing.T) {
	if _, err := specdoc.Load(); err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	text := string(specdoc.Bytes())
	// normalizeUnionPipes collapses the three pipe encodings the
	// pinned document uses inside unions: escaped pipes in the
	// operation table, raw pipes in prose, and HTML entities in
	// the manifest table.
	normalizeUnionPipes := func(raw string) []string {
		raw = strings.ReplaceAll(raw, "&#124;", "|")
		raw = strings.ReplaceAll(raw, "\\|", "|")
		var values []string
		for _, value := range strings.Split(raw, "|") {
			if value == "" {
				continue
			}
			values = append(values, value)
		}
		return values
	}
	// firstSpanUnion takes the union in the first code span after
	// a prose marker. A leading member-name prefix on the first
	// element is stripped: the union starts after it.
	firstSpanUnion := func(t *testing.T, marker string) []string {
		t.Helper()
		index := strings.Index(text, marker)
		if index < 0 {
			t.Fatalf("spec union %q not found", marker)
		}
		rest := text[index+len(marker):]
		open := strings.Index(rest, "<code>")
		if open < 0 {
			t.Fatalf("no code span after %q", marker)
		}
		rest = rest[open+len("<code>"):]
		close := strings.Index(rest, "</code>")
		if close < 0 {
			t.Fatalf("unbalanced code span after %q", marker)
		}
		values := normalizeUnionPipes(rest[:close])
		if len(values) == 0 {
			t.Fatalf("empty union after %q", marker)
		}
		if prefix := strings.IndexAny(values[0], ":="); prefix >= 0 {
			values[0] = values[0][prefix+1:]
		}
		return values
	}
	// unionAfter takes the union starting exactly at the end of
	// the marker: the marker already names the sentence and the
	// member prefix, so nothing is consumed from the union and
	// nothing is stripped. This is the anchored form for unions
	// whose bare member name collides with an unrelated earlier
	// span in the document.
	unionAfter := func(t *testing.T, marker string) []string {
		t.Helper()
		index := strings.Index(text, marker)
		if index < 0 {
			t.Fatalf("spec union %q not found", marker)
		}
		rest := text[index+len(marker):]
		close := strings.Index(rest, "</code>")
		if close < 0 {
			t.Fatalf("unbalanced code span after %q", marker)
		}
		values := normalizeUnionPipes(rest[:close])
		if len(values) == 0 {
			t.Fatalf("empty union after %q", marker)
		}
		return values
	}
	// rowInlineUnion takes an inline member union from one
	// operation-table row: the first line starting with the row
	// marker, the union after the member prefix up to the next
	// comma or closing brace. The scan is windowed to the Section
	// 7.8 registry table because the probe name recurs in the
	// Terminal Backend, Provider Probe, and Directory Node tables
	// with different vocabularies.
	rowInlineUnion := func(t *testing.T, row, prefix string) []string {
		t.Helper()
		table, ok := sectionTableWindow(text, "The exact request and success <code>body</code> registry is:", "Candidate objects are addressed only by their exact")
		if !ok {
			t.Fatal("Section 7.8 registry table window not found")
		}
		var line string
		for _, candidate := range strings.Split(table, "\n") {
			if strings.HasPrefix(candidate, row) {
				line = candidate
				break
			}
		}
		if line == "" {
			t.Fatalf("spec row %q not found", row)
		}
		index := strings.Index(line, prefix)
		if index < 0 {
			t.Fatalf("spec member %q not found in row %q", prefix, row)
		}
		rest := line[index+len(prefix):]
		end := strings.IndexAny(rest, ",}")
		if end < 0 {
			t.Fatalf("unterminated union in row %q", row)
		}
		values := normalizeUnionPipes(rest[:end])
		if len(values) == 0 {
			t.Fatalf("empty union in row %q", row)
		}
		return values
	}
	setEqual := func(t *testing.T, name string, got []string, want []string) {
		t.Helper()
		sortedGot := append([]string(nil), got...)
		sortedWant := append([]string(nil), want...)
		sort.Strings(sortedGot)
		sort.Strings(sortedWant)
		if strings.Join(sortedGot, ",") != strings.Join(sortedWant, ",") {
			t.Errorf("%s = %v, want spec %v", name, got, want)
		}
	}
	t.Run("operations ordered", func(t *testing.T) {
		operations, _, _ := deriveOperationTable(t)
		var got []string
		for _, operation := range operationOrder {
			got = append(got, string(operation))
		}
		if strings.Join(got, ",") != strings.Join(operations, ",") {
			t.Errorf("operation registry = %v, want table %v", got, operations)
		}
	})
	t.Run("capabilities ordered", func(t *testing.T) {
		derived := deriveTypeMembers(t, text, "The exact capability names are")
		if strings.Join(capabilityOrder, ",") != strings.Join(derived, ",") {
			t.Errorf("capability registry = %v, want spec %v", capabilityOrder, derived)
		}
	})
	setEqual(t, "capture classes", captureClasses, firstSpanUnion(t, "Each Capture Item has one native key and class"))
	setEqual(t, "projection strategies", projectionStrategies, firstSpanUnion(t, "Strategies are"))
	setEqual(t, "fidelity dispositions", fidelityDispositions, firstSpanUnion(t, "The dispositions are"))
	setEqual(t, "capability statuses", capabilityStatuses, unionAfter(t, "Each capability value contains exactly\n<code>status:"))
	setEqual(t, "capability evidences", capabilityEvidences, unionAfter(t, "<code>enabled:boolean</code>,\n<code>evidence:"))
	setEqual(t, "read purposes", readPurposes, unionAfter(t, "<code>ReadAuthority</code> contains exactly <code>authority_id:UUIDv7</code>,\n  <code>purpose:"))
	setEqual(t, "object purposes", objectPurposes, unionAfter(t, "<code>ObjectAuthority</code> contains exactly\n  <code>authority_id:UUIDv7</code>,\n  <code>purpose:"))
	setEqual(t, "finding severities", findingSeverities, unionAfter(t, "<code>AdapterFinding</code> contains exactly\n  <code>severity:"))
	setEqual(t, "directions", directionNames, unionAfter(t, "<code>SupportedEnvironmentTupleKey</code> contains exactly\n<code>direction:"))
	setEqual(t, "doctor directions agree", directionNames, rowInlineUnion(t, "| <code>doctor</code> |", "direction:"))
	setEqual(t, "roles", roleNames, unionAfter(t, "<code>SessionAdapterExecutionBinding</code> containing exactly\n<code>role:"))
	setEqual(t, "binding candidate kinds", candidateKindNames, unionAfter(t, "<code>provider_id:provider-id</code>,\n<code>candidate_kind:"))
	setEqual(t, "probe candidate kinds agree", candidateKindNames, rowInlineUnion(t, "| <code>probe</code> |", "expected_candidate_kind:"))
	setEqual(t, "object modes", objectModes, unionAfter(t, "  <code>mode:"))
	setEqual(t, "tuple entry statuses", tupleEntryStatuses, unionAfter(t, "<code>valid_until:timestamp</code>,\n<code>status:"))
	setEqual(t, "registry entry statuses", registryEntryStatuses, rowInlineUnion(t, "| <code>doctor</code> |", "registry_entry_status:"))
	t.Run("evidence results", func(t *testing.T) {
		setEqual(t, "evidence results", evidenceResults, []string{"pass"})
		for _, marker := range []string{"<code>FixtureEvidence</code> contains exactly", "<code>ResumeSmokeEvidence</code> contains exactly"} {
			index := strings.Index(text, marker)
			if index < 0 {
				t.Fatalf("spec sentence %q not found", marker)
			}
			rest := text[index+len(marker):]
			end := sentenceEnd(rest)
			if end < 0 {
				t.Fatalf("spec sentence for %q never terminates", marker)
			}
			if !strings.Contains(rest[:end], "<code>result=pass</code>") {
				t.Errorf("spec sentence %q carries no <code>result=pass</code>", marker)
			}
		}
	})
	t.Run("context-free operations", func(t *testing.T) {
		_, requests, _ := deriveOperationTable(t)
		var want []string
		for _, operation := range operationOrder {
			context := false
			for _, member := range requests[string(operation)] {
				if member == "context" {
					context = true
					break
				}
			}
			if !context {
				want = append(want, string(operation))
			}
		}
		var got []string
		for _, operation := range contextFreeOperations {
			got = append(got, string(operation))
		}
		setEqual(t, "context-free operations", got, want)
	})
	setEqual(t, "platforms", []string{string(scalar.PlatformLinux), string(scalar.PlatformMacOS), string(scalar.PlatformWindows), string(scalar.PlatformWSL2)}, unionAfter(t, "<code>platform="))
	setEqual(t, "architectures", tupleArchitectures, unionAfter(t, "<code>architecture="))
	setEqual(t, "fidelity profiles", fidelityProfiles, rowInlineUnion(t, "| <code>projection-plan</code> |", "fidelity_profile:"))
	setEqual(t, "validate modes", validateModes, rowInlineUnion(t, "| <code>validate</code> |", "mode:"))
}
