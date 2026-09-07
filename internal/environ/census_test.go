package environ

import (
	"errors"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
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
// where the grammar already has one) plus the retained copies;
// each copy names the battery file that pins it to the canonical
// semantics in both directions. The sessadapter and dirnode frame
// decoders are delegating wrappers onto environ.DecodeStrictObject
// (their raw-scan divergence, once ledgered here, was closed by
// delegation and is proven in both directions by the
// frame-agreement battery), and their per-helper Check*,
// stringLength, and grammar copies converged the same way. The
// provhost decoder and surrogate gate stay independent because
// their owning assembly is frozen by the accepted story leaves
// (protocol.go); the canonicaljson gates stay independent
// because the canonicalizer validates any JSON value with its
// own depth, number, and control rules rather than closed
// objects; the scalar gate stays independent because it
// validates a single JSON string scalar rather than a frame.
// Each retained copy carries a bidirectional battery row that
// fails when the two diverge in either direction, and the
// outcome names the freeze or role split behind every one.
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
	"string-measure|sessadapter|decode.go|stringLength": "delegating wrapper onto environ.StringLength, pinned by TestCheckHelpersDelegateToEnviron + measure_agreement_test.go",
	"string-measure|dirnode|decode.go|stringLength":     "delegating wrapper onto environ.StringLength, pinned by TestCheckHelpersDelegateToEnviron + measure_agreement_test.go",
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
// symbol map, and returns ledger keys for each hit. Production
// file selection and fail-closed parsing come from invcore: an
// unreadable or unparseable production file fails the suite
// instead of scanning as clean, and a package with zero
// production files fails instead of passing vacuously.
func scanSharedSymbols(t *testing.T, symbols map[string]string, functions bool) []string {
	t.Helper()
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	packages := make([]string, 0, len(censusPackages))
	for pkg := range censusPackages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	var keys []string
	for _, pkg := range packages {
		files, _ := invcore.MustScanProduction(t, filepath.Join(root, pkg))
		for _, production := range files {
			syntax, file := production.Syntax, production.Name
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
		}
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
// Both directions run through the invcore harness, so a truncated
// derivation fails instead of passing vacuously.
func TestSharedImplementationsAreCensused(t *testing.T) {
	derived := deriveSharedSites(t)
	rows := make(map[string]struct{}, len(sharedLedger))
	for key := range sharedLedger {
		rows[key] = struct{}{}
	}
	derivedSet := make(map[string]struct{}, len(derived))
	for key := range derived {
		derivedSet[key] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "shared-rule copy", derivedSet, rows)
}

// grammarSiteLedger names every grammar definition site in the
// census scope. A new grammar definition fails here even when it
// carries an identical literal: a second copy is a second copy,
// and the next edit drifts exactly one of them.
// grammarSiteLedger names every grammar definition site in the
// census scope. The sessadapter and dirnode env-id, semver, and
// extensions copies converged onto the environ grammars (their
// check sites call environ.CheckEnvironmentID, environ.CheckSemver,
// and environ.CheckExtensions directly), so their rows are GONE,
// not retained: a revived copy fails here as unregistered. The
// retained copies each name their owner: provider-id is scalar's
// rule everywhere, while the provhost and canonicaljson copies of
// the environ-owned grammars are pinned by the one-language
// battery in both directions.
var grammarSiteLedger = map[string]bool{
	"env-id-grammar|environ|tuple.go|environmentIDPattern":                true,
	"env-id-grammar|canonicaljson|closed_shapes.go|environmentIDPattern":  true,
	"semver-grammar|environ|tuple.go|semverPattern":                       true,
	"semver-grammar|canonicaljson|closed_shapes.go|semverPattern":         true,
	"semver-grammar|provhost|manifest.go|semverPattern":                   true,
	"extensions-grammar|environ|decode.go|reverseDNSPattern":              true,
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
// function on planted sources. Parsing is fail-closed through
// invcore: an unparseable file fails instead of scanning as
// clean.
func grammarCandidatesInFile(path string, source []byte) ([]grammarCandidate, error) {
	syntax, _, failure := invcore.ParseBytes(path, source, parser.ParseComments)
	if failure != "" {
		return nil, errors.New(failure)
	}
	return grammarCandidatesInSyntax(syntax), nil
}

// grammarCandidatesInSyntax extracts every package-level single
// MustCompile definition from one parsed file. Production
// derivation drives this over invcore-scanned files; the
// synthetic controls reach it through grammarCandidatesInFile,
// so both prove the same extractor.
func grammarCandidatesInSyntax(syntax *ast.File) []grammarCandidate {
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
	return candidates
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
	packages := make([]string, 0, len(censusPackages))
	for pkg := range censusPackages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	for _, pkg := range packages {
		files, _ := invcore.MustScanProduction(t, filepath.Join(root, pkg))
		for _, production := range files {
			for _, candidate := range grammarCandidatesInSyntax(production.Syntax) {
				collected = append(collected, collectedCandidate{pkg: pkg, file: production.Name, candidate: candidate})
			}
		}
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
	derived := make(map[string]struct{}, len(sites))
	for key := range sites {
		derived[key] = struct{}{}
	}
	rows := make(map[string]struct{}, len(grammarSiteLedger))
	for key := range grammarSiteLedger {
		rows[key] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "grammar copy", derived, rows)
	for key := range fresh {
		t.Errorf("unregistered grammar copy under a fresh name %q: a second copy is a second copy even when the symbol is new", key)
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
