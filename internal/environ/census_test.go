package environ

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This file is the name layer of the shared-implementation
// census. The story's premise is that the Provider, Session
// Adapter, and Directory Node facades sit on one environment
// library; each facade carried its own copy of the shared rules,
// and independent copies of the same rule drift in ways no
// per-facade reviewer can see. This census is the boundary that
// fails when the facades diverge: it derives every shared-rule
// implementation known by identifier from production source and
// requires the derived set to equal the ledger below exactly.
// The shape layer in shape_census_test.go derives implementations
// by implementation texture and catches copies under fresh
// spellings, new byte measures, and var-bound aliases that this
// name layer cannot see; the two layers are complementary by
// construction.
//
// Removing or renaming a ledgered implementation orphans its row
// and fails too. An empty scan fails closed. The ledger records
// the single canonical owner per rule (this package, or scalar
// where the grammar already has one) plus the copies this leaf
// does not unify; each copy names the battery file that pins it
// to the canonical semantics. The sessadapter and dirnode frame
// decoders are delegating wrappers onto environ.DecodeStrictObject
// (their raw-scan divergence, once ledgered here, was closed by
// delegation and is proven in both directions by the
// frame-agreement battery); the remaining provhost, canonicaljson,
// and scalar copies are unification residue tracked in the
// cross-story follow-up, not harmless divergence.
//
// Scope. The scan covers the story packages (environ, provider,
// provhost, sessadapter, dirnode) plus the grammar owners
// (scalar, canonicaljson). Copies in config, cliresult,
// terminalbackend, axerror, and the other stories' packages are
// outside this story's ownership: each package derives its own
// inventories in its own test files, and gating another story's
// package from here would couple stories that must land
// independently. That exclusion is a stated bound, not a blind
// spot: the scan fails closed inside its scope, and a copy that
// moves into scope (a new definition in a scanned package)
// fails as unregistered.

// sharedSite is one derived shared-rule implementation.
type sharedSite struct {
	class  string
	pkg    string
	file   string
	symbol string
}

// sharedSiteKey renders the ledger key for one site.
func (site sharedSite) sharedSiteKey() string {
	return site.class + "|" + site.pkg + "|" + site.file + "|" + site.symbol
}

// censusPackages is the exact scan scope. A production file in
// any other package is outside this story's jurisdiction.
var censusPackages = map[string]bool{
	"environ":       true,
	"provider":      true,
	"provhost":      true,
	"sessadapter":   true,
	"dirnode":       true,
	"scalar":        true,
	"canonicaljson": true,
}

// sharedLedger maps every known shared-rule implementation to
// the rationale that keeps it honest. Canonical owners are the
// single implementation new code must use; every other row names
// the battery file that pins the copy to the canonical
// semantics.
var sharedLedger = map[string]string{
	// Frame decoders: one JSON entry point per package.
	"strict-decoder|environ|decode.go|DecodeStrictObject":     "canonical owner: string-walk surrogate semantics, rune measure",
	"strict-decoder|provhost|protocol.go|decodeStrictObject":  "pinned by frame_agreement_test.go",
	"strict-decoder|sessadapter|decode.go|decodeStrictObject": "delegating wrapper onto environ.DecodeStrictObject, pinned by TestDelegatingWrappersCallEnviron + frame_agreement_test.go",
	"strict-decoder|dirnode|decode.go|decodeStrictObject":     "delegating wrapper onto environ.DecodeStrictObject, pinned by TestDelegatingWrappersCallEnviron + frame_agreement_test.go",
	"strict-decoder|canonicaljson|canonical.go|decodeStrict":  "canonicalizer entry: distinct JCS pipeline role, surrogate verdicts pinned by frame_agreement_test.go",
	// Lone-surrogate gates.
	"surrogate-gate|environ|decode.go|HasLoneSurrogateEscape":            "canonical owner: string-aware walk",
	"surrogate-gate|provhost|surrogate.go|hasLoneSurrogateEscape":        "string-aware walk, pinned by frame_agreement_test.go",
	"surrogate-gate|canonicaljson|canonical.go|validateSurrogateEscapes": "canonicalizer gate: string-aware walk, pinned by frame_agreement_test.go",
	// String measures.
	"string-measure|environ|decode.go|StringLength":     "canonical owner: runes per Section 1.6",
	"string-measure|provhost|opdecode.go|runeLength":    "runes, pinned by measure_agreement_test.go",
	"string-measure|sessadapter|decode.go|stringLength": "byte-frozen leaf 1: runes, pinned by measure_agreement_test.go",
	"string-measure|dirnode|decode.go|stringLength":     "byte-frozen leaf 2: runes, pinned by measure_agreement_test.go",
	// Identity validators: two refusal dialects over one rule.
	"identity-validator|provhost|identity.go|CheckIdentity":                           "provider-stdio refusal dialect, pinned by identity_agreement_test.go",
	"identity-validator|canonicaljson|core_records.go|validateProviderIdentityRecord": "identity refusal dialect, pinned by identity_agreement_test.go",
}

// sharedFunctionSymbols maps a function name to its census
// class.
var sharedFunctionSymbols = map[string]string{
	"decodeStrictObject":             "strict-decoder",
	"DecodeStrictObject":             "strict-decoder",
	"decodeStrict":                   "strict-decoder",
	"hasLoneSurrogateEscape":         "surrogate-gate",
	"HasLoneSurrogateEscape":         "surrogate-gate",
	"validateSurrogateEscapes":       "surrogate-gate",
	"stringLength":                   "string-measure",
	"runeLength":                     "string-measure",
	"StringLength":                   "string-measure",
	"CheckIdentity":                  "identity-validator",
	"validateProviderIdentityRecord": "identity-validator",
}

// grammarSymbols maps a package-level variable name to its
// grammar class. Only same-rule grammars share a class: the
// three capabilityOrder tables (directory eight, adapter
// fifteen, provider seven) and the per-major platform tables are
// deliberately absent here and are pinned by literal-list
// derivation in grammar_test.go instead, where the rule identity
// travels with the comparison.
var grammarSymbols = map[string]string{
	"environmentIDPattern": "env-id-grammar",
	"semverPattern":        "semver-grammar",
	"reverseDNSPattern":    "extensions-grammar",
	"providerIDPattern":    "provider-id-grammar",
}

// deriveSharedSites scans every production Go file in the census
// packages for shared-rule implementations. Test files are never
// scanned: a helper in a test is not a production copy. An
// unreadable file or a parse failure fails closed: a scanner that
// cannot read a file must not report the file as clean.
func deriveSharedSites(t *testing.T) map[string]bool {
	t.Helper()
	sites := map[string]bool{}
	for _, site := range scanSharedSymbols(t, sharedFunctionSymbols, true) {
		sites[site] = true
	}
	if len(sites) == 0 {
		t.Fatal("census derived zero shared-rule sites from a tree known to duplicate them; the scanner is blind, not the tree clean")
	}
	return sites
}

// scanSharedSymbols scans the census packages for function or
// package-level variable definitions whose names appear in the
// symbol map, and returns ledger keys for each hit.
func scanSharedSymbols(t *testing.T, symbols map[string]string, functions bool) []string {
	t.Helper()
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	var keys []string
	count := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(rel, string(filepath.Separator))
		pkg := parts[0]
		if !censusPackages[pkg] {
			return nil
		}
		count++
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("census: read %s: %v", path, err)
		}
		syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			t.Fatalf("census: parse %s: %v", path, err)
		}
		file := parts[len(parts)-1]
		for _, decl := range syntax.Decls {
			switch node := decl.(type) {
			case *ast.FuncDecl:
				if !functions || node.Name == nil {
					continue
				}
				if class, ok := symbols[node.Name.Name]; ok {
					keys = append(keys, sharedSite{class, pkg, file, node.Name.Name}.sharedSiteKey())
				}
			case *ast.GenDecl:
				if functions {
					continue
				}
				for _, spec := range node.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range value.Names {
						if class, ok := symbols[name.Name]; ok {
							keys = append(keys, sharedSite{class, pkg, file, name.Name}.sharedSiteKey())
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("census: walk: %v", err)
	}
	if count == 0 {
		t.Fatal("census scanned zero files; the scanner is broken, not the tree")
	}
	return keys
}

// internalRoot returns the internal/ source root above this
// package directory.
func internalRoot(t *testing.T) (string, error) {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(directory), nil
}

// TestSharedImplementationsAreCensused requires the derived
// implementation set to equal the ledger exactly: an unregistered
// copy fails as a unification violation, and a row naming an
// implementation production no longer derives fails as orphaned.
func TestSharedImplementationsAreCensused(t *testing.T) {
	derived := deriveSharedSites(t)
	for key := range derived {
		if _, ok := sharedLedger[key]; !ok {
			t.Errorf("unregistered shared-rule copy %q: add one canonical implementation in internal/environ instead, or ledger the divergence with its agreement battery", key)
		}
	}
	for key, rationale := range sharedLedger {
		if !derived[key] {
			t.Errorf("orphaned ledger row %q (%s): production no longer derives it", key, rationale)
		}
	}
}

// grammarSiteLedger names every grammar definition site in the
// census scope. A new grammar definition fails here even when it
// carries an identical literal: a second copy is a second copy,
// and the next edit drifts exactly one of them.
var grammarSiteLedger = map[string]bool{
	"env-id-grammar|environ|tuple.go|environmentIDPattern":                true,
	"env-id-grammar|sessadapter|decode.go|environmentIDPattern":           true,
	"env-id-grammar|dirnode|decode.go|environmentIDPattern":               true,
	"env-id-grammar|canonicaljson|closed_shapes.go|environmentIDPattern":  true,
	"semver-grammar|environ|tuple.go|semverPattern":                       true,
	"semver-grammar|sessadapter|decode.go|semverPattern":                  true,
	"semver-grammar|dirnode|decode.go|semverPattern":                      true,
	"semver-grammar|canonicaljson|closed_shapes.go|semverPattern":         true,
	"semver-grammar|provhost|manifest.go|semverPattern":                   true,
	"extensions-grammar|environ|decode.go|reverseDNSPattern":              true,
	"extensions-grammar|sessadapter|decode.go|reverseDNSPattern":          true,
	"extensions-grammar|dirnode|decode.go|reverseDNSPattern":              true,
	"extensions-grammar|canonicaljson|closed_shapes.go|reverseDNSPattern": true,
	"provider-id-grammar|scalar|names.go|providerIDPattern":               true,
	"provider-id-grammar|provhost|manifest.go|providerIDPattern":          true,
	"provider-id-grammar|sessadapter|decode.go|providerIDPattern":         true,
	"provider-id-grammar|dirnode|query.go|providerIDPattern":              true,
}

// grammarCandidate is one package-level single MustCompile
// definition: the defined symbol and its pattern literal.
type grammarCandidate struct {
	symbol  string
	literal string
}

// grammarCandidatesInFile extracts every package-level single
// MustCompile definition from one source file. It is pure over
// its inputs, so the synthetic controls drive this exact
// function on planted sources.
func grammarCandidatesInFile(path string, source []byte) ([]grammarCandidate, error) {
	syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		return nil, err
	}
	var candidates []grammarCandidate
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			call, ok := value.Values[0].(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "MustCompile" {
				continue
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				continue
			}
			text, err := strconv.Unquote(literal.Value)
			if err != nil {
				continue
			}
			candidates = append(candidates, grammarCandidate{symbol: value.Names[0].Name, literal: text})
		}
	}
	return candidates, nil
}

// deriveGrammarLiterals derives the regex literal behind every
// grammar definition in the census scope, keyed by ledger key.
// Two definitions carrying different literals are different
// languages even when the symbol name matches; the grammar
// battery requires one literal per class. Definitions under a
// fresh name carrying a known class literal derive as fresh
// copies of that class: a second copy is a second copy even
// when the symbol is new.
func deriveGrammarLiterals(t *testing.T) (map[string]map[string]bool, map[string]bool, map[string]bool) {
	t.Helper()
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("grammar: %v", err)
	}
	literals := map[string]map[string]bool{}
	sites := map[string]bool{}
	type collectedCandidate struct {
		pkg       string
		file      string
		candidate grammarCandidate
	}
	var collected []collectedCandidate
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(rel, string(filepath.Separator))
		pkg := parts[0]
		if !censusPackages[pkg] {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("grammar: read %s: %v", path, err)
		}
		candidates, err := grammarCandidatesInFile(path, source)
		if err != nil {
			t.Fatalf("grammar: parse %s: %v", path, err)
		}
		for _, candidate := range candidates {
			collected = append(collected, collectedCandidate{pkg: pkg, file: parts[len(parts)-1], candidate: candidate})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("grammar: walk: %v", err)
	}
	for _, entry := range collected {
		class, known := grammarSymbols[entry.candidate.symbol]
		if !known {
			continue
		}
		if literals[class] == nil {
			literals[class] = map[string]bool{}
		}
		literals[class][entry.candidate.literal] = true
		sites[sharedSite{class, entry.pkg, entry.file, entry.candidate.symbol}.sharedSiteKey()] = true
	}
	knownLiterals := grammarKnownLiterals(literals)
	fresh := map[string]bool{}
	for _, entry := range collected {
		if class, copy := classifyGrammarCandidate(entry.candidate, knownLiterals); copy {
			fresh[sharedSite{class, entry.pkg, entry.file, entry.candidate.symbol}.sharedSiteKey()] = true
		}
	}
	return literals, sites, fresh
}

// grammarKnownLiterals inverts the derived class literals into a
// literal-to-class map. It is pure over the derivation, so the
// synthetic controls classify against the same map production
// uses.
func grammarKnownLiterals(literals map[string]map[string]bool) map[string]string {
	known := map[string]string{}
	for class, texts := range literals {
		for text := range texts {
			known[text] = class
		}
	}
	return known
}

// classifyGrammarCandidate reports whether a MustCompile
// definition is a fresh-name copy of a known class: a symbol
// outside grammarSymbols carrying a known class literal. Named
// definitions and unknown literals are not copies.
func classifyGrammarCandidate(candidate grammarCandidate, known map[string]string) (string, bool) {
	if _, named := grammarSymbols[candidate.symbol]; named {
		return "", false
	}
	class, match := known[candidate.literal]
	if !match {
		return "", false
	}
	return class, true
}

// TestSharedGrammarsAreOneLanguage requires every grammar in the
// census scope to carry the identical regex literal: same symbol
// with different literals would be two languages under one name,
// and a mutant rewriting one copy's literal reddens here.
func TestSharedGrammarsAreOneLanguage(t *testing.T) {
	literals, sites, fresh := deriveGrammarLiterals(t)
	for key := range sites {
		if !grammarSiteLedger[key] {
			t.Errorf("unregistered grammar copy %q: one language means one definition site per package at most", key)
		}
	}
	for key := range fresh {
		t.Errorf("unregistered grammar copy under a fresh name %q: a second copy is a second copy even when the symbol is new", key)
	}
	for key := range grammarSiteLedger {
		if !sites[key] {
			t.Errorf("orphaned grammar row %q: production no longer derives it", key)
		}
	}
	for class, texts := range literals {
		if len(texts) != 1 {
			var variants []string
			for text := range texts {
				variants = append(variants, strconv.Quote(text))
			}
			t.Errorf("grammar class %q has %d distinct literals: %s", class, len(texts), strings.Join(variants, " vs "))
		}
	}
	if len(literals) != len(grammarSymbols) {
		t.Fatalf("grammar: derived %d classes, want %d; the extractor is blind, not the tree clean", len(literals), len(grammarSymbols))
	}
}
