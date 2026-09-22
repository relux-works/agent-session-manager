package sessrepo

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// This file is the sessrepo refusal census. Every rejection in this package
// funnels through the refuse var, and the census derives its denominator
// from production source instead of listing it: a new refuse call without a
// boundary-driven negative path fails the gate, and a row without a site
// fails too. The gate has three halves:
//
//  1. Static derivation: every direct refuse(SENTINEL, ...) call in the
//     production files is derived as file:line with its sentinel. Calls
//     must sit on a single line with a bare sentinel identifier, or the
//     derivation fails closed.
//  2. Runtime recording: TestMain wraps the refuse var, attributing each
//     exercised refusal to its production file:line plus the sentinel it
//     carried, and whether a production boundary entry (Open,
//     CreateSession, AppendEvent, GetRecord, GetEvent, ListEvents,
//     ListSessions, Resolve) is on the stack. A direct helper call proves
//     the helper refuses and nothing about the boundary, so only
//     boundary-driven sites count.
//  3. Alias audit: every refuse reference outside direct-call position,
//     and every import-aliased or var-bound reference to the watched owner
//     delegations, fails through invcore.AuditConstructorReferences, with
//     the funnel's own declaration allowlisted by derived position.
//
// Control plants below prove each half fails closed, including an import
// alias and a var binding in both the passing and the failing direction.
var sessrepoRefusalRecorder = invcore.NewSiteRecorder()

var sessrepoBoundarySites = map[string]struct{}{}

var sessrepoBoundaryMarkers = []string{
	"sessrepo.Open",
	"sessrepo.CheckFencingExpiry",
	").CreateSession",
	").AppendEvent",
	").GetRecord",
	").GetEvent",
	").ListEvents",
	").ListSessions",
	").Resolve",
	").CreateLease",
	").CompareAndSwapLease",
	").GetLease",
	").ListLeases",
	").WinningLease",
	").VerifyFencingToken",
}

func TestMain(main *testing.M) {
	original := refuse
	refuse = func(sentinel error, format string, arguments ...any) error {
		err := original(sentinel, format, arguments...)
		sessrepoRefusalRecorder.Record(sentinel.Error(), 1)
		if beneathSessrepoBoundary() {
			if site := callerSessrepoSite(); site != "" {
				sessrepoBoundarySites[site] = struct{}{}
			}
		}
		return err
	}
	code := main.Run()
	if fullSessrepoPackageTestRun() {
		// The content-equality audit is static: derivation over production
		// source diffed against the ledger, with no dependence on which
		// tests passed. It runs on every full run, green or red, so a
		// plant reddens the gate even when an unrelated test is already
		// red — a red suite masks nothing. The refusal runtime audit below
		// instead compares exercised sites against derived sites, which is
		// only meaningful when every test ran, so it stays gated on green.
		files, fileSet, scanFailures := invcore.ScanProduction(".")
		var failures []string
		failures = append(failures, scanFailures...)
		if len(scanFailures) == 0 {
			failures = append(failures, auditSessrepoEquality(files, fileSet)...)
		}
		if code == 0 {
			failures = append(failures, auditSessrepoRefusals()...)
		}
		if len(failures) != 0 {
			for _, failure := range failures {
				fmt.Fprintln(os.Stderr, failure)
			}
			code = 1
		}
	}
	os.Exit(code)
}

func fullSessrepoPackageTestRun() bool {
	selected := flag.Lookup("test.run")
	return selected == nil || selected.Value.String() == ""
}

// callerSessrepoSite reports the production file:line of the refuse call
// site: the wrapper calls this helper, so skip lands past Callers, this
// helper, and the wrapper, on the production line that refused.
func callerSessrepoSite() string {
	frames := make([]uintptr, 4)
	if count := runtime.Callers(3, frames); count == 0 {
		return ""
	}
	located := runtime.CallersFrames(frames)
	frame, _ := located.Next()
	if frame.Function == "" {
		return ""
	}
	return filepath.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
}

func beneathSessrepoBoundary() bool {
	frames := make([]uintptr, 64)
	for {
		count := runtime.Callers(2, frames)
		located := runtime.CallersFrames(frames[:count])
		for range frames[:count] {
			frame, _ := located.Next()
			for _, marker := range sessrepoBoundaryMarkers {
				if strings.Contains(frame.Function, marker) {
					return true
				}
			}
		}
		if count < len(frames) {
			return false
		}
		frames = make([]uintptr, 2*len(frames))
	}
}

// auditSessrepoRefusals derives the expected inventory from production AST
// and requires the exercised sets to match it in both directions, the alias
// audit to be clean, and the observed sentinel set to equal the derived
// sentinel roster exactly. It runs only on a green suite (see TestMain):
// the exercised sets are meaningless when tests failed. The static
// content-equality audit lives outside it and runs unconditionally.
func auditSessrepoRefusals() []string {
	files, fileSet, failures := invcore.ScanProduction(".")
	if len(failures) != 0 {
		return failures
	}
	derived, sentinels, sentinelNames, derivationFailures := deriveSessrepoRefusalSites(files, fileSet)
	failures = append(failures, derivationFailures...)
	aliasFailures := auditSessrepoAliases(files, fileSet)
	failures = append(failures, aliasFailures...)
	failures = append(failures, auditSessrepoUnrouted(files, fileSet, sentinelNames)...)
	derivedSet := make(map[string]struct{}, len(derived))
	for _, site := range derived {
		derivedSet[site.key()] = struct{}{}
	}
	unregistered, orphaned, diffFailures := invcore.DiffSets(derivedSet, sessrepoRefusalRecorder.Sites())
	for _, failure := range diffFailures {
		failures = append(failures, "sessrepo refusal census: "+failure)
	}
	for _, site := range unregistered {
		failures = append(failures, "sessrepo refusal site without an exercised negative path: "+site)
	}
	for _, site := range orphaned {
		failures = append(failures, "sessrepo exercised refusal outside the derived inventory: "+site)
	}
	var missingBoundary []string
	for _, site := range derived {
		if _, ok := sessrepoBoundarySites[site.key()]; !ok {
			missingBoundary = append(missingBoundary, site.key())
		}
	}
	sort.Strings(missingBoundary)
	if len(missingBoundary) != 0 {
		failures = append(failures, "sessrepo refusal sites never reached beneath a production boundary entry (a direct helper call does not prove the production effect): "+strings.Join(missingBoundary, ", "))
	}
	var extraBoundary []string
	for site := range sessrepoBoundarySites {
		if _, ok := derivedSet[site]; !ok {
			extraBoundary = append(extraBoundary, site)
		}
	}
	sort.Strings(extraBoundary)
	if len(extraBoundary) != 0 {
		failures = append(failures, "sessrepo boundary-exercised sites outside the derived inventory: "+strings.Join(extraBoundary, ", "))
	}
	wantCodes := make([]string, 0, len(sentinels))
	for code := range sentinels {
		wantCodes = append(wantCodes, code)
	}
	sort.Strings(wantCodes)
	gotCodes := sessrepoRefusalRecorder.Codes()
	if fmt.Sprintf("%v", gotCodes) != fmt.Sprintf("%v", wantCodes) {
		failures = append(failures, fmt.Sprintf("sessrepo observed refusal codes = %v, want closed roster %v", gotCodes, wantCodes))
	}
	return failures
}

// sessrepoRefusalSite is one derived call site with its sentinel.
type sessrepoRefusalSite struct {
	file     string
	line     int
	sentinel string
}

func (site sessrepoRefusalSite) key() string {
	return site.file + ":" + strconv.Itoa(site.line)
}

// deriveSessrepoRefusalSites derives every direct refuse(SENTINEL, ...)
// call in the production files plus the closed sentinel roster. The roster
// derives from package-level Err* vars initialized with errors.New, so a
// new sentinel without a refusal — or a refusal carrying an undeclared
// sentinel — fails the gate. Sentinel names feed the unrouted-refusal
// audit; sentinel texts feed the observed-code audit.
func deriveSessrepoRefusalSites(files []invcore.ProductionFile, fileSet *token.FileSet) ([]sessrepoRefusalSite, map[string]struct{}, map[string]struct{}, []string) {
	var sites []sessrepoRefusalSite
	sentinels := map[string]struct{}{}
	sentinelNames := map[string]struct{}{}
	var failures []string
	for _, production := range files {
		for _, declaration := range production.Syntax.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range value.Names {
					if !strings.HasPrefix(name.Name, "Err") || index >= len(value.Values) {
						continue
					}
					call, ok := value.Values[index].(*ast.CallExpr)
					if !ok {
						continue
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || selector.Sel.Name != "New" {
						continue
					}
					receiver, ok := selector.X.(*ast.Ident)
					if !ok || receiver.Name != "errors" || len(call.Args) != 1 {
						continue
					}
					literal, ok := call.Args[0].(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						failures = append(failures, production.Name+": sentinel "+name.Name+" is not an errors.New string literal")
						continue
					}
					text, err := strconv.Unquote(literal.Value)
					if err != nil {
						failures = append(failures, production.Name+": sentinel "+name.Name+": "+err.Error())
						continue
					}
					sentinels[text] = struct{}{}
					sentinelNames[name.Name] = struct{}{}
				}
			}
		}
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok || identifier.Name != "refuse" || len(call.Args) < 1 {
				return true
			}
			position := fileSet.Position(identifier.Pos())
			if !position.IsValid() {
				failures = append(failures, production.Name+": refuse call at an invalid position")
				return true
			}
			if end := fileSet.Position(call.End()); end.Line != position.Line {
				failures = append(failures, production.Name+":"+strconv.Itoa(position.Line)+": refuse call spans several lines; the runtime site and the parsed site would disagree")
				return true
			}
			sentinel, ok := call.Args[0].(*ast.Ident)
			if !ok {
				failures = append(failures, production.Name+":"+strconv.Itoa(position.Line)+": refuse sentinel is not a bare identifier; the extractor cannot see through indirection")
				return true
			}
			sites = append(sites, sessrepoRefusalSite{file: production.Name, line: position.Line, sentinel: sentinel.Name})
			return true
		})
	}
	if len(sites) == 0 {
		failures = append(failures, "sessrepo refusal census derived zero sites; the scanner is blind, not the package empty")
	}
	if len(sentinels) == 0 {
		failures = append(failures, "sessrepo refusal census derived zero sentinels; the roster extractor is blind")
	}
	return sites, sentinels, sentinelNames, failures
}

// auditSessrepoUnrouted fails every fmt.Errorf call that wraps a package
// sentinel outside the refuse funnel. A routed refusal and an identical
// direct wrap behave the same, so without this half a bypassed funnel
// passes the behavioral suite and the site inventory together.
func auditSessrepoUnrouted(files []invcore.ProductionFile, fileSet *token.FileSet, sentinelNames map[string]struct{}) []string {
	var unrouted []string
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Errorf" {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok || receiver.Name != "fmt" {
				return true
			}
			for _, argument := range call.Args {
				identifier, ok := argument.(*ast.Ident)
				if !ok {
					continue
				}
				if _, watched := sentinelNames[identifier.Name]; watched {
					position := fileSet.Position(call.Pos())
					unrouted = append(unrouted, production.Name+":"+strconv.Itoa(position.Line))
				}
			}
			return true
		})
	}
	sort.Strings(unrouted)
	if len(unrouted) == 0 {
		return nil
	}
	return []string{"sessrepo refusals raised outside the refuse funnel (the derived inventory cannot require a negative path for them): " + strings.Join(unrouted, ", ")}
}

// sessrepoConstructorSpec is the alias-audit policy: the local funnel must
// stay a direct call, and the watched owner delegations must stay direct
// qualified calls resolved by import path, so neither an import alias nor
// a var binding can hide a refusal or a delegation.
func sessrepoConstructorSpec() invcore.ConstructorSpec {
	return invcore.ConstructorSpec{
		Local: map[string]bool{"refuse": true},
		Qualified: map[string]map[string]bool{
			"github.com/relux-works/agent-session-manager/internal/canonicaljson": {"VerifyObjectIdentity": true},
			"github.com/relux-works/agent-session-manager/internal/environ":       {"DecodeStrictObject": true, "CheckUUIDv7": true, "CheckUint53Bounds": true},
			"github.com/relux-works/agent-session-manager/internal/scalar":        {"ParseDigest": true, "ParseUUIDv7": true, "ParseUUIDv4": true},
		},
	}
}

// auditSessrepoAliases runs the alias audit over every production file,
// allowlisting the funnel's own declaration position: a package-level
// `var refuse = func...` binding a function literal.
func auditSessrepoAliases(files []invcore.ProductionFile, fileSet *token.FileSet) []string {
	var failures []string
	spec := sessrepoConstructorSpec()
	for _, production := range files {
		allowed := sessrepoFunnelPositions(production.Syntax)
		for _, failure := range invcore.AuditConstructorReferences(production.Syntax, fileSet, production.Name, spec) {
			if atAllowedPosition(failure, production.Name, fileSet, allowed) {
				continue
			}
			failures = append(failures, failure)
		}
	}
	return failures
}

func sessrepoFunnelPositions(syntax *ast.File) map[token.Pos]bool {
	positions := map[token.Pos]bool{}
	for _, declaration := range syntax.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range value.Names {
				if name.Name != "refuse" || index >= len(value.Values) {
					continue
				}
				if _, ok := value.Values[index].(*ast.FuncLit); ok {
					positions[name.Pos()] = true
				}
			}
		}
	}
	return positions
}

func atAllowedPosition(failure, display string, fileSet *token.FileSet, allowed map[token.Pos]bool) bool {
	for position := range allowed {
		located := fileSet.Position(position)
		if !located.IsValid() {
			continue
		}
		prefix := display + ":" + strconv.Itoa(located.Line) + ":" + strconv.Itoa(located.Column) + ":"
		if strings.Contains(failure, prefix) {
			return true
		}
	}
	return false
}

// TestCensusDiffSetsFailsClosed plants the three derivation outcomes
// against the production DiffSets harness: an unregistered site, an orphan
// row, and a zero-site derivation each fail instead of passing silently.
func TestCensusDiffSetsFailsClosed(t *testing.T) {
	unregistered, orphaned, failures := invcore.DiffSets(map[string]struct{}{"chain.go:1": {}}, map[string]struct{}{})
	if len(failures) != 0 {
		t.Fatalf("plant failures = %v", failures)
	}
	if len(unregistered) != 1 || unregistered[0] != "chain.go:1" {
		t.Fatalf("unregistered = %v, want [chain.go:1]", unregistered)
	}
	if len(orphaned) != 0 {
		t.Fatalf("orphaned = %v, want none", orphaned)
	}
	unregistered, orphaned, failures = invcore.DiffSets(map[string]struct{}{"chain.go:1": {}}, map[string]struct{}{"chain.go:1": {}, "chain.go:2": {}})
	if len(failures) != 0 {
		t.Fatalf("plant failures = %v", failures)
	}
	if len(unregistered) != 0 {
		t.Fatalf("unregistered = %v, want none", unregistered)
	}
	if len(orphaned) != 1 || orphaned[0] != "chain.go:2" {
		t.Fatalf("orphaned = %v, want [chain.go:2]", orphaned)
	}
	_, _, failures = invcore.DiffSets(map[string]struct{}{}, map[string]struct{}{})
	if len(failures) == 0 {
		t.Fatal("zero-site derivation passed clean; the harness would bless a blind scan")
	}
}

// sessrepoAliasPlant is one control for the alias audit: wantFailure names
// the fragment a reporting plant must carry, empty for a clean plant.
type sessrepoAliasPlant struct {
	name        string
	source      string
	wantFailure string
}

func sessrepoAliasPlants() []sessrepoAliasPlant {
	canonical := `"github.com/relux-works/agent-session-manager/internal/canonicaljson"`
	scalar := `"github.com/relux-works/agent-session-manager/internal/scalar"`
	return []sessrepoAliasPlant{
		{
			name:        "direct local call is clean",
			source:      "package plant\nfunc run() error {\n\treturn refuse(errExample, \"no\")\n}\n",
			wantFailure: "",
		},
		{
			name:        "import alias direct call resolves by path",
			source:      "package plant\nimport cjson " + canonical + "\nfunc run(raw []byte) {\n\t_, _, _ = cjson.VerifyObjectIdentity(raw)\n}\n",
			wantFailure: "",
		},
		{
			name:        "import alias var binding fails",
			source:      "package plant\nimport cjson " + canonical + "\nfunc run(raw []byte) {\n\tverify := cjson.VerifyObjectIdentity\n\t_, _, _ = verify(raw)\n}\n",
			wantFailure: "outside direct-call position",
		},
		{
			name:        "local var binding fails",
			source:      "package plant\nfunc run() error {\n\tdeny := refuse\n\treturn deny(errExample, \"no\")\n}\n",
			wantFailure: "outside direct-call position",
		},
		{
			name:        "dot import of a watched path fails closed",
			source:      "package plant\nimport . " + scalar + "\nfunc run(value string) {\n\t_, _ = ParseDigest(value)\n}\n",
			wantFailure: "dot-imports watched path",
		},
		{
			name:        "shadowed funnel name fails closed",
			source:      "package plant\nfunc run(refuse int) int {\n\treturn refuse\n}\n",
			wantFailure: "outside direct-call position",
		},
	}
}

// TestCensusAliasPlants drives every control through the production audit
// spec and requires the ledgered outcome. A plant that stops reporting
// fails here by construction.
func TestCensusAliasPlants(t *testing.T) {
	spec := sessrepoConstructorSpec()
	for _, plant := range sessrepoAliasPlants() {
		t.Run(plant.name, func(t *testing.T) {
			syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(plant.source))
			if failure != "" {
				t.Fatalf("control does not parse: %s", failure)
			}
			failures := invcore.AuditConstructorReferences(syntax, fileSet, "plant.go", spec)
			if plant.wantFailure == "" {
				if len(failures) != 0 {
					t.Fatalf("clean plant reported: %s", strings.Join(failures, "; "))
				}
				return
			}
			matched := false
			for _, failure := range failures {
				if strings.Contains(failure, plant.wantFailure) {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("plant reported %q, want a failure containing %q", failures, plant.wantFailure)
			}
		})
	}
}

// TestCensusPlantHarnessFailsClosed proves the plant harness itself is
// fail-closed: an unparseable control is a harness failure, never clean.
func TestCensusPlantHarnessFailsClosed(t *testing.T) {
	_, _, failure := invcore.ParseSource("plant.go", []byte("package plant\nfunc broken( {\n"))
	if failure == "" {
		t.Fatal("unparseable control parsed clean; the harness would pass a blind plant")
	}
}

// The content-equality census derives every byte-content comparison in the
// package through invcore, the way the refusal inventory above is derived:
// over every production file, never over hard-coded names or tokens. The
// class has two classified forms:
//
//   - E1: a qualified bytes.Equal call, resolved by import path (not local
//     spelling) through invcore.WatchedCallPositions, so an import alias
//     still derives. A var-bound Equal is not a call and fails the alias
//     audit instead of deriving.
//   - E2: a == or != whose operand converts content with string(...): the
//     string(left) == string(right) spelling carries no bytes.Equal token
//     and a substring count cannot see it. Identifier comparisons between
//     already-string values (digests, lease IDs, session IDs) are not
//     content comparisons and stay outside the class.
//
// An unclassifiable spelling — bytes.Compare or bytes.EqualFold,
// reflect.DeepEqual, slices.Equal — fails closed instead of passing
// silently; widen the class when one is genuinely needed. A hand-rolled
// comparison loop is the stated blind spot: it carries no classified call
// and no conversion comparison, so this census would not see it.
func sessrepoEqualitySpec() invcore.ConstructorSpec {
	return invcore.ConstructorSpec{
		Qualified: map[string]map[string]bool{
			"bytes": {"Equal": true},
		},
	}
}

// sessrepoEqualityWideSpec is the alias-audit policy for the class: every
// spelling that compares byte content, classified or not, so an aliased
// or bound reference to any of them fails outside direct-call position.
func sessrepoEqualityWideSpec() invcore.ConstructorSpec {
	return invcore.ConstructorSpec{
		Qualified: map[string]map[string]bool{
			"bytes":   {"Equal": true, "Compare": true, "EqualFold": true},
			"reflect": {"DeepEqual": true},
			"slices":  {"Equal": true},
		},
	}
}

// equalitySite is one derived content-equality comparison with its form.
type equalitySite struct {
	file string
	line int
	form string
}

func (site equalitySite) key() string {
	return site.file + ":" + strconv.Itoa(site.line)
}

// equalitySitesInFile derives the classified sites of one source: E1
// through the invcore attribution set, E2 through the conversion-operand
// scan. Display names the file for diagnostics.
func equalitySitesInFile(syntax *ast.File, fileSet *token.FileSet, display string) ([]equalitySite, []string) {
	var sites []equalitySite
	var failures []string
	for position := range invcore.WatchedCallPositions(syntax, sessrepoEqualitySpec()) {
		located := fileSet.Position(position)
		if !located.IsValid() {
			failures = append(failures, display+": bytes.Equal call at an invalid position")
			continue
		}
		sites = append(sites, equalitySite{file: display, line: located.Line, form: "bytes.Equal"})
	}
	ast.Inspect(syntax, func(node ast.Node) bool {
		comparison, ok := node.(*ast.BinaryExpr)
		if !ok || (comparison.Op != token.EQL && comparison.Op != token.NEQ) {
			return true
		}
		if !isStringConversion(comparison.X) && !isStringConversion(comparison.Y) {
			return true
		}
		located := fileSet.Position(comparison.Pos())
		if !located.IsValid() {
			failures = append(failures, display+": string comparison at an invalid position")
			return true
		}
		sites = append(sites, equalitySite{file: display, line: located.Line, form: "string-=="})
		return true
	})
	return sites, failures
}

// isStringConversion reports a string(...) builtin conversion call.
func isStringConversion(node ast.Node) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	identifier, ok := call.Fun.(*ast.Ident)
	return ok && identifier.Name == "string"
}

// unclassifiableEqualitySpellings reports direct calls to content-equality
// spellings outside the classified set, resolved by import path so an
// import alias cannot hide them.
func unclassifiableEqualitySpellings(syntax *ast.File, fileSet *token.FileSet, display string) []string {
	wide := invcore.WatchedCallPositions(syntax, sessrepoEqualityWideSpec())
	narrow := invcore.WatchedCallPositions(syntax, sessrepoEqualitySpec())
	var failures []string
	for position := range wide {
		if narrow[position] {
			continue
		}
		located := fileSet.Position(position)
		if !located.IsValid() {
			failures = append(failures, display+": unclassifiable equality call at an invalid position")
			continue
		}
		failures = append(failures, display+":"+strconv.Itoa(located.Line)+": content-equality spelling outside the classified set (bytes.Equal, string ==); widen the census or remove it")
	}
	sort.Strings(failures)
	return failures
}

// sessrepoEqualityLedger registers every derived content-equality site
// with the same-length vector and narrowing mutant that witness it. The
// full-package audit diffs this ledger against the derived set in both
// directions: a site without a row is an unwitnessed comparison, a row
// without a site is a witness without a guard.
func sessrepoEqualityLedger() map[string]string {
	return map[string]string{
		"sessrepo.go:262":    "TestCreateSessionRefusesResumeWithDifferingBytes // N10 length-only",
		"chain.go:442":       "TestAppendEventRefusesSameLengthDisagreeingBytes // N13 length-only",
		"lease_store.go:650": "TestInstallDisagreeingBytesRefuses/same_length // N length-only",
	}
}

// deriveSessrepoEqualitySites derives the classified sites over every
// production file. Zero derived sites is a blind scan, never a pass.
func deriveSessrepoEqualitySites(files []invcore.ProductionFile, fileSet *token.FileSet) (map[string]struct{}, []string) {
	derived := map[string]struct{}{}
	var failures []string
	for _, production := range files {
		sites, siteFailures := equalitySitesInFile(production.Syntax, fileSet, production.Name)
		failures = append(failures, siteFailures...)
		for _, site := range sites {
			derived[site.key()] = struct{}{}
		}
	}
	if len(derived) == 0 {
		failures = append(failures, "sessrepo content-equality census derived zero sites; the scanner is blind, not the package empty")
	}
	return derived, failures
}

// auditSessrepoEquality runs the class census over every production file:
// derivation, the wide alias audit (import aliases resolve by path, var
// bindings and dot imports fail), the unclassifiable-spelling check, and
// the both-direction diff against the ledger. TestMain runs it on every
// full unfiltered package run, green or red, so a census-only plant stays
// green under -run and reddens the full gate.
func auditSessrepoEquality(files []invcore.ProductionFile, fileSet *token.FileSet) []string {
	derived, failures := deriveSessrepoEqualitySites(files, fileSet)
	for _, production := range files {
		for _, failure := range invcore.AuditConstructorReferences(production.Syntax, fileSet, production.Name, sessrepoEqualityWideSpec()) {
			failures = append(failures, failure)
		}
		failures = append(failures, unclassifiableEqualitySpellings(production.Syntax, fileSet, production.Name)...)
	}
	unregistered, orphaned, diffFailures := invcore.DiffSets(derived, sessrepoEqualityLedgerRows())
	for _, failure := range diffFailures {
		failures = append(failures, "sessrepo content-equality census: "+failure)
	}
	for _, site := range unregistered {
		failures = append(failures, "sessrepo content-equality site without a registered vector: "+site)
	}
	for _, row := range orphaned {
		failures = append(failures, "sessrepo content-equality row without a derived site: "+row)
	}
	return failures
}

// sessrepoEqualityLedgerRows keys the ledger for the both-direction diff.
func sessrepoEqualityLedgerRows() map[string]struct{} {
	rows := map[string]struct{}{}
	for key := range sessrepoEqualityLedger() {
		rows[key] = struct{}{}
	}
	return rows
}

// parseEqualityPlant parses one synthetic control source for the equality
// census plants below. An unparseable control is a harness failure.
func parseEqualityPlant(t *testing.T, name, source string) (*ast.File, *token.FileSet) {
	t.Helper()
	syntax, fileSet, failure := invcore.ParseSource(name, []byte(source))
	if failure != "" {
		t.Fatalf("control does not parse: %s", failure)
	}
	return syntax, fileSet
}

// derivedEqualityKeys derives classified site keys from one control source.
func derivedEqualityKeys(t *testing.T, name, source string) map[string]struct{} {
	t.Helper()
	syntax, fileSet := parseEqualityPlant(t, name, source)
	sites, failures := equalitySitesInFile(syntax, fileSet, name)
	if len(failures) != 0 {
		t.Fatalf("derivation reported %q on a classified control", failures)
	}
	keys := map[string]struct{}{}
	for _, site := range sites {
		keys[site.key()] = struct{}{}
	}
	return keys
}

// TestEqualityCensusDerivesBytesEqualThroughImportAlias is the alias-clean
// plant: a bytes.Equal call through an import alias resolves by path and
// still derives, so renaming the import cannot hide a comparison.
func TestEqualityCensusDerivesBytesEqualThroughImportAlias(t *testing.T) {
	keys := derivedEqualityKeys(t, "plant.go", "package plant\nimport be \"bytes\"\nfunc equal(left, right []byte) bool {\n\treturn be.Equal(left, right)\n}\n")
	if len(keys) != 1 {
		t.Fatalf("aliased bytes.Equal derived %v, want one site", keys)
	}
	syntax, fileSet := parseEqualityPlant(t, "plant.go", "package plant\nimport be \"bytes\"\nfunc equal(left, right []byte) bool {\n\treturn be.Equal(left, right)\n}\n")
	if failures := invcore.AuditConstructorReferences(syntax, fileSet, "plant.go", sessrepoEqualityWideSpec()); len(failures) != 0 {
		t.Fatalf("aliased direct call reported: %s", strings.Join(failures, "; "))
	}
}

// liveEqualityDerived scans the live production tree and derives its
// classified content-equality sites. The plant tests below couple to this
// live set and the live ledger instead of hard-coded copies: a test that
// diffs a synthetic plant against a hard-coded row list passes even when
// the live census it names is blind, which is exactly the round-3 defect
// repeated. Coupling to the live derivation means drift on either side
// fails the sanity half before the plant half runs.
func liveEqualityDerived(t *testing.T) map[string]struct{} {
	t.Helper()
	files, fileSet, failures := invcore.ScanProduction(".")
	if len(failures) != 0 {
		t.Fatalf("live scan reported %q", failures)
	}
	derived, siteFailures := deriveSessrepoEqualitySites(files, fileSet)
	if len(siteFailures) != 0 {
		t.Fatalf("live derivation reported %q", siteFailures)
	}
	return derived
}

// liveEqualityLedgerRows returns the live ledger rows under test.
func liveEqualityLedgerRows(t *testing.T) map[string]struct{} {
	t.Helper()
	rows := sessrepoEqualityLedgerRows()
	if len(rows) == 0 {
		t.Fatal("live ledger holds zero rows; the census would bless any plant")
	}
	return rows
}

// TestEqualityCensusFailsClosedOnNewFileSite is the P2 red-before plant: a
// bytes.Equal in a file the ledger never registered — the new-file shape
// a hard-coded filename list cannot see — is an unregistered site, not a
// pass. The derivation takes every scanned file, so the filename is
// evidence, not configuration. The plant half merges the synthetic new
// file's derived keys into the live derived set and diffs against the
// live ledger, so a regression to hard-coded filenames breaks the sanity
// half (live derived would miss the new file) or the plant half here.
func TestEqualityCensusFailsClosedOnNewFileSite(t *testing.T) {
	derived := liveEqualityDerived(t)
	rows := liveEqualityLedgerRows(t)
	if unregistered, orphaned, failures := invcore.DiffSets(derived, rows); len(failures) != 0 || len(unregistered) != 0 || len(orphaned) != 0 {
		t.Fatalf("live census not clean: unregistered=%v orphaned=%v failures=%v", unregistered, orphaned, failures)
	}
	plant := derivedEqualityKeys(t, "plantextra.go", "package sessrepo\nimport \"bytes\"\nfunc extra(left, right []byte) bool {\n\treturn bytes.Equal(left, right)\n}\n")
	if len(plant) != 1 {
		t.Fatalf("new-file plant derived %v, want one site", plant)
	}
	merged := make(map[string]struct{}, len(derived)+len(plant))
	for key := range derived {
		merged[key] = struct{}{}
	}
	for key := range plant {
		merged[key] = struct{}{}
	}
	unregistered, _, failures := invcore.DiffSets(merged, rows)
	if len(failures) != 0 {
		t.Fatalf("plant failures = %v", failures)
	}
	if len(unregistered) != 1 {
		t.Fatalf("unregistered = %v, want the plantextra.go site", unregistered)
	}
	if !strings.HasPrefix(unregistered[0], "plantextra.go:") {
		t.Fatalf("unregistered = %v, want a plantextra.go row", unregistered)
	}
}

// TestEqualityCensusFailsClosedOnStringEqualitySpelling is the P3
// red-before plant: string(left) == string(right) carries no bytes.Equal
// token and derives as the conversion-operand form, so a token count
// passes it while the class census registers an unregistered site. Like
// the new-file plant above it merges into the live derived set and diffs
// against the live ledger.
func TestEqualityCensusFailsClosedOnStringEqualitySpelling(t *testing.T) {
	derived := liveEqualityDerived(t)
	rows := liveEqualityLedgerRows(t)
	if unregistered, orphaned, failures := invcore.DiffSets(derived, rows); len(failures) != 0 || len(unregistered) != 0 || len(orphaned) != 0 {
		t.Fatalf("live census not clean: unregistered=%v orphaned=%v failures=%v", unregistered, orphaned, failures)
	}
	plant := derivedEqualityKeys(t, "store.go", "package sessrepo\nfunc equal(left, right []byte) bool {\n\treturn string(left) == string(right)\n}\n")
	if len(plant) != 1 {
		t.Fatalf("string-equality spelling derived %v, want one site", plant)
	}
	merged := make(map[string]struct{}, len(derived)+len(plant))
	for key := range derived {
		merged[key] = struct{}{}
	}
	for key := range plant {
		merged[key] = struct{}{}
	}
	unregistered, _, failures := invcore.DiffSets(merged, rows)
	if len(failures) != 0 {
		t.Fatalf("plant failures = %v", failures)
	}
	if len(unregistered) != 1 {
		t.Fatalf("unregistered = %v, want the store.go plant site", unregistered)
	}
	if !strings.HasPrefix(unregistered[0], "store.go:") {
		t.Fatalf("unregistered = %v, want a store.go row", unregistered)
	}
}

// TestEqualityCensusFailsClosedOnOrphanRow plants the reverse direction: a
// ledger row with no derived site is a witness without a guard. The
// derived side is the live set and the rows are the live ledger plus one
// foreign row, so ledger drift fails loudly instead of passing against a
// hard-coded copy.
func TestEqualityCensusFailsClosedOnOrphanRow(t *testing.T) {
	derived := liveEqualityDerived(t)
	rows := liveEqualityLedgerRows(t)
	rows["store.go:99"] = struct{}{}
	unregistered, orphaned, failures := invcore.DiffSets(derived, rows)
	if len(failures) != 0 {
		t.Fatalf("plant failures = %v", failures)
	}
	if len(unregistered) != 0 {
		t.Fatalf("unregistered = %v, want none", unregistered)
	}
	if len(orphaned) != 1 || orphaned[0] != "store.go:99" {
		t.Fatalf("orphaned = %v, want [store.go:99]", orphaned)
	}
}

// TestEqualityCensusFailsClosedOnVarBinding plants the var-binding
// direction in both spellings: binding bytes.Equal (even through an
// import alias) and calling the binding fails the alias audit instead of
// deriving a site.
func TestEqualityCensusFailsClosedOnVarBinding(t *testing.T) {
	for _, source := range []string{
		"package plant\nimport \"bytes\"\nfunc equal(left, right []byte) bool {\n\teq := bytes.Equal\n\treturn eq(left, right)\n}\n",
		"package plant\nimport be \"bytes\"\nfunc equal(left, right []byte) bool {\n\teq := be.Equal\n\treturn eq(left, right)\n}\n",
	} {
		syntax, fileSet := parseEqualityPlant(t, "plant.go", source)
		matched := false
		for _, failure := range invcore.AuditConstructorReferences(syntax, fileSet, "plant.go", sessrepoEqualityWideSpec()) {
			if strings.Contains(failure, "outside direct-call position") {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("var-bound equality passed clean for %q; the audit would bless a hidden comparison", source)
		}
	}
}

// TestEqualityCensusFailsClosedOnDotImport plants the unclassifiable
// import shape: a dot import of a watched path fails closed because the
// audit cannot attribute the bare member.
func TestEqualityCensusFailsClosedOnDotImport(t *testing.T) {
	syntax, fileSet := parseEqualityPlant(t, "plant.go", "package plant\nimport . \"bytes\"\nfunc equal(left, right []byte) bool {\n\treturn Equal(left, right)\n}\n")
	matched := false
	for _, failure := range invcore.AuditConstructorReferences(syntax, fileSet, "plant.go", sessrepoEqualityWideSpec()) {
		if strings.Contains(failure, "dot-imports watched path") {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("dot-imported equality passed clean; the audit would bless an unattributable comparison")
	}
}

// TestEqualityCensusFailsClosedOnUnclassifiableSpelling plants the form
// the classifier cannot classify: bytes.Compare (and reflect.DeepEqual)
// compare byte content without either classified spelling and fail the
// census instead of passing silently — including through an import alias.
func TestEqualityCensusFailsClosedOnUnclassifiableSpelling(t *testing.T) {
	for _, source := range []string{
		"package plant\nimport \"bytes\"\nfunc equal(left, right []byte) bool {\n\treturn bytes.Compare(left, right) == 0\n}\n",
		"package plant\nimport be \"bytes\"\nfunc equal(left, right []byte) bool {\n\treturn be.Compare(left, right) == 0\n}\n",
		"package plant\nimport \"reflect\"\nfunc equal(left, right []byte) bool {\n\treturn reflect.DeepEqual(left, right)\n}\n",
	} {
		syntax, fileSet := parseEqualityPlant(t, "plant.go", source)
		if failures := unclassifiableEqualitySpellings(syntax, fileSet, "plant.go"); len(failures) == 0 {
			t.Fatalf("unclassifiable spelling passed clean for %q; widen the class or keep refusing it, never bless it", source)
		}
	}
}
