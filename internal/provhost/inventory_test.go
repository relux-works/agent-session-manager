package provhost

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// exercisedRefusalSites records the production file:line that constructed
// a refusal, for every instrumented refusal constructor call made during
// the test run. observedCodes records every stable code those refusals
// carried.
var exercisedRefusalSites sync.Map
var observedCodes sync.Map

func recordRefusalSite(code string) {
	observedCodes.Store(code, struct{}{})
	if _, file, line, ok := runtime.Caller(2); ok {
		exercisedRefusalSites.Store(fmt.Sprintf("%s:%d", filepath.Base(file), line), struct{}{})
	}
}

func TestMain(main *testing.M) {
	origInvalid, origProtocol, origMismatch := failInvalid, failProtocol, failMismatch
	origProcess, origTimeout, origIntegrity := failProcess, failTimeout, failIntegrity
	failInvalid = func(detail string) (*axerror.Error, error) {
		failure, err := origInvalid(detail)
		if err == nil {
			recordRefusalSite(string(failure.Code()))
		}
		return failure, err
	}
	failProtocol = func(detail, member string) (*axerror.Error, error) {
		failure, err := origProtocol(detail, member)
		if err == nil {
			recordRefusalSite(string(failure.Code()))
		}
		return failure, err
	}
	failMismatch = func(detail, observed string) (*axerror.Error, error) {
		failure, err := origMismatch(detail, observed)
		if err == nil {
			recordRefusalSite(string(failure.Code()))
		}
		return failure, err
	}
	failProcess = func(detail string, cause error) (*axerror.Error, error) {
		failure, err := origProcess(detail, cause)
		if err == nil {
			recordRefusalSite(string(failure.Code()))
		}
		return failure, err
	}
	failTimeout = func(millisDetail string, millis int64) (*axerror.Error, error) {
		failure, err := origTimeout(millisDetail, millis)
		if err == nil {
			recordRefusalSite(string(failure.Code()))
		}
		return failure, err
	}
	failIntegrity = func(detail, statusState, materializationID, transactionID string) (*axerror.Error, error) {
		failure, err := origIntegrity(detail, statusState, materializationID, transactionID)
		if err == nil {
			recordRefusalSite(string(failure.Code()))
		}
		return failure, err
	}
	code := main.Run()
	if code == 0 && fullPackageTestRun() {
		if failures := auditRefusalInventory(); len(failures) != 0 {
			for _, failure := range failures {
				fmt.Fprintln(os.Stderr, failure)
			}
			code = 1
		}
	}
	os.Exit(code)
}

func fullPackageTestRun() bool {
	selected := flag.Lookup("test.run")
	return selected == nil || selected.Value.String() == ""
}

// auditRefusalInventory derives the refusal inventory from package
// source, never from memory: every production call to a refusal
// constructor must have an exercised negative path, no Structured Error
// may be built outside the six constructors, no raw error may be minted,
// and the observed code set must equal the closed code set exactly.
func auditRefusalInventory() []string {
	directory, err := os.Getwd()
	if err != nil {
		return []string{fmt.Sprintf("derive provhost refusal inventory: %v", err)}
	}
	inventory, err := deriveRefusalInventory(directory)
	if err != nil {
		return []string{fmt.Sprintf("derive provhost refusal inventory: %v", err)}
	}
	var failures []string
	// Floor: a derived domain that can silently derive nothing is not a
	// measurement. An empty derivation must fail closed, never pass
	// vacuously — the same "the scan is blind" floor the discovery
	// package carries on its own scans.
	if len(inventory.ScannedFiles) == 0 {
		failures = append(failures, "scanned no production sources; the check is blind")
	}
	if len(inventory.Sites) == 0 {
		failures = append(failures, "derived no refusal sites; the scan is blind")
	}
	// Reverse direction: every exercised refusal site must resolve to a
	// derived site. The exercised set comes from the test run, not from
	// a hand list, so a truncated derivation reddens here even though
	// the forward check passes vacuously.
	derived := map[string]bool{}
	for _, site := range inventory.Sites {
		derived[site] = true
	}
	var outside []string
	exercisedRefusalSites.Range(func(key, _ any) bool {
		site := key.(string)
		if !derived[site] {
			outside = append(outside, site)
		}
		return true
	})
	sort.Strings(outside)
	if len(outside) != 0 {
		failures = append(failures, "exercised refusal sites outside the derived inventory; the derivation is short: "+strings.Join(outside, ", "))
	}
	if len(inventory.StrayBuilders) != 0 {
		sort.Strings(inventory.StrayBuilders)
		failures = append(failures, "provhost failures built outside the refusal constructors: "+strings.Join(inventory.StrayBuilders, ", "))
	}
	if len(inventory.RawConstructors) != 0 {
		sort.Strings(inventory.RawConstructors)
		failures = append(failures, "provhost raw error construction: "+strings.Join(inventory.RawConstructors, ", "))
	}
	var missing []string
	for _, site := range inventory.Sites {
		if _, ok := exercisedRefusalSites.Load(site); !ok {
			missing = append(missing, site)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		failures = append(failures, "provhost refusal call sites without an exercised negative path: "+strings.Join(missing, ", "))
	}
	var codes []string
	observedCodes.Range(func(key, _ any) bool {
		codes = append(codes, key.(string))
		return true
	})
	sort.Strings(codes)
	want := []string{"incompatible_protocol", "integrity_failure", "invalid_config", "provider_process_failed", "provider_protocol_error", "provider_timeout"}
	if fmt.Sprintf("%v", codes) != fmt.Sprintf("%v", want) {
		failures = append(failures, fmt.Sprintf("observed refusal codes = %v, want closed set %v", codes, want))
	}
	return failures
}

// refusalInventory is derived from package sources.
type refusalInventory struct {
	// ScannedFiles are the production source basenames the derivation
	// parsed. The audit fails closed when this is empty.
	ScannedFiles []string
	// Sites are file:line positions of production refusal constructor
	// calls. Every constructor call must sit on a single source line so
	// the runtime site and the parsed site agree.
	Sites []string
	// StrayBuilders are axerror.New or axerror.LocalFromUntrusted calls
	// outside a constructor body.
	StrayBuilders []string
	// RawConstructors are errors.New, fmt.Errorf, or panic calls
	// anywhere in production code. There is no allowlist: dynamic facts
	// travel in Structured Error details, never in Go error text.
	RawConstructors []string
}

var refusalConstructors = map[string]bool{
	"failInvalid":   true,
	"failProtocol":  true,
	"failMismatch":  true,
	"failProcess":   true,
	"failTimeout":   true,
	"failIntegrity": true,
}

func deriveRefusalInventory(directory string) (refusalInventory, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return refusalInventory{}, err
	}
	var inventory refusalInventory
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		inventory.ScannedFiles = append(inventory.ScannedFiles, name)
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			return refusalInventory{}, err
		}
		syntax, err := parser.ParseFile(fileSet, path, source, parser.ParseComments)
		if err != nil {
			return refusalInventory{}, err
		}
		// Map each constructor's own body span so builder calls inside
		// it are not mistaken for strays.
		constructorBodies := map[*ast.FuncLit]bool{}
		for _, decl := range syntax.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				values, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, ident := range values.Names {
					if !refusalConstructors[ident.Name] || i >= len(values.Values) {
						continue
					}
					if literal, ok := values.Values[i].(*ast.FuncLit); ok {
						constructorBodies[literal] = true
					}
				}
			}
		}
		insideConstructor := func(pos token.Pos) bool {
			for body := range constructorBodies {
				if pos >= body.Pos() && pos <= body.End() {
					return true
				}
			}
			return false
		}
		ast.Inspect(syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			position := fileSet.Position(call.Pos())
			site := fmt.Sprintf("%s:%d", filepath.Base(position.Filename), position.Line)
			if ident, ok := call.Fun.(*ast.Ident); ok {
				if refusalConstructors[ident.Name] {
					inventory.Sites = append(inventory.Sites, site)
					return true
				}
				if ident.Name == "panic" {
					inventory.RawConstructors = append(inventory.RawConstructors, site+" panic")
					return true
				}
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if qualifier, ok := selector.X.(*ast.Ident); ok {
				qualified := qualifier.Name + "." + selector.Sel.Name
				switch qualified {
				case "axerror.New", "axerror.LocalFromUntrusted":
					if !insideConstructor(call.Pos()) {
						inventory.StrayBuilders = append(inventory.StrayBuilders, site+" "+qualified)
					}
					return true
				case "errors.New", "fmt.Errorf":
					inventory.RawConstructors = append(inventory.RawConstructors, site+" "+qualified)
					return true
				}
			}
			if selector.Sel.Name == "panic" {
				inventory.RawConstructors = append(inventory.RawConstructors, site+" panic")
			}
			return true
		})
	}
	return inventory, nil
}

// sectionLines joins the pinned document lines for a section, failing
// when the section boundaries move. Every citation below resolves
// against the real text, never against a restatement in this repository.
func sectionLines(t *testing.T, document *specdoc.Document, section string, first, last int) string {
	t.Helper()
	for _, line := range []int{first, last} {
		got, ok := document.SectionID(line)
		if !ok || got != section {
			t.Fatalf("SPEC.md line %d is in section %q, want %q", line, got, section)
		}
	}
	var body strings.Builder
	for line := first; line <= last; line++ {
		text, ok := document.Line(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is missing", line)
		}
		body.WriteString(text)
		body.WriteString("\n")
	}
	return body.String()
}

// requireQuote asserts the excerpt occurs verbatim in the pinned
// document and begins on a line inside the cited section.
func requireQuote(t *testing.T, document *specdoc.Document, excerpt, section string) {
	t.Helper()
	if !document.Contains(excerpt) {
		t.Fatalf("pinned document does not contain %q", excerpt)
	}
	lines := document.QuoteLines(excerpt)
	if len(lines) == 0 {
		t.Fatalf("pinned document quotes no line for %q", excerpt)
	}
	got, ok := document.SectionID(lines[0])
	if !ok || got != section {
		t.Fatalf("%q begins on SPEC.md line %d in section %q, want %q", excerpt, lines[0], got, section)
	}
	t.Logf("%q begins on SPEC.md line %d", excerpt, lines[0])
}

var operationCellPattern = regexp.MustCompile(`^\| <code>([a-z-]+)</code> \|`)

// TestOperationRegistryIsDerivedFromSpec proves the dispatch registry is
// exactly the Section 7.5 operation table in order: every table row must
// appear in Operations() at the same position, and nothing else may.
// A dropped, added, or reordered entry reddens here.
func TestOperationRegistryIsDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.5", 3070, 3086)
	var derived []string
	for _, line := range strings.Split(window, "\n") {
		if match := operationCellPattern.FindStringSubmatch(line); match != nil {
			derived = append(derived, match[1])
		}
	}
	if len(derived) == 0 {
		t.Fatal("derived no operations from the Section 7.5 table; the check is blind")
	}
	got := Operations()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", derived) {
		t.Fatalf("Operations() = %v, want the Section 7.5 table %v", got, derived)
	}
	t.Logf("operation registry coverage: %d/%d table entries dispatched", len(got), len(derived))
}

// TestSection72FramingIsPinned quotes every Section 7.2 sentence this
// package implements: the framing limits, the process model, the
// envelope shapes, the closed-member rule, and the deadline/redaction
// duties.
func TestSection72FramingIsPinned(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	for _, excerpt := range []string{
		"Each line MUST be one complete UTF-8",
		"JSON object no larger than 8 MiB",
		"Stdout MUST contain protocol frames only",
		"human diagnostics go to stderr",
		"starts one plugin process per operation",
		"After one response, <code>ax</code> closes stdin and the",
		"Exit 0 requires a successful response",
		"A structured error",
		"response SHOULD exit 0 because the protocol succeeded",
		"a crash, invalid frame,",
		"or missing response is a provider-host failure",
		"The deadline MUST be in the future when the host writes the frame",
		"A failure envelope contains",
		"and MUST NOT contain <code>body</code>",
		"Envelope and body",
		"unknown members are protocol errors under major version 2",
		"The host MUST terminate a plugin that exceeds its deadline and report",
		"<code>provider_timeout</code>",
		"It MUST redact environment and stderr before",
		"logging",
	} {
		requireQuote(t, document, excerpt, "7.2")
	}
}

// TestSection151ProviderRowIsPinned quotes the Section 15.1 provider
// row behind the local failure mapping: recognizable majors yield
// incompatible_protocol, everything else unusable yields
// provider_protocol_error, and a different major's payload is never
// trusted.
func TestSection151ProviderRowIsPinned(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	for _, excerpt := range []string{
		"The host accepts no child error object, terminates/waits for exit as applicable, and emits its own local Error 1.0.0",
		"<code>incompatible_protocol</code> for a recognizable major mismatch, otherwise <code>provider_protocol_error</code>",
		"Receivers MUST NOT parse a different major",
		"payload far enough to trust its error code, retryable bit, details, or",
	} {
		requireQuote(t, document, excerpt, "15.1")
	}
}

// TestStatusRecoveryRulesArePinned quotes the Section 7.5 recovery
// sentences behind DecodeStatusOutcome: the evolving status read, the
// unknown quarantine, and the state/nullability table.
func TestStatusRecoveryRulesArePinned(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	for _, excerpt := range []string{
		"is the required evolving lost-response recovery",
		"An unknown transaction MUST return state",
		"<code>unknown</code>; the host MUST NOT infer success",
		"An unreadable or",
		"integrity-invalid transaction MUST fail with <code>integrity_failure</code> and",
		"MUST be quarantined rather than represented as a successful status object",
		"<code>unknown</code> requires null",
		"requires all three non-null",
	} {
		requireQuote(t, document, excerpt, "7.5")
	}
}

// TestStableCodesAreRegistered pins every refusal code this package
// emits against the pinned error registry and the exact exit the host
// surfaces, and agrees with the axerror wire table.
func TestStableCodesAreRegistered(t *testing.T) {
	registered := map[string]int{}
	for _, row := range catalog.Current().Errors {
		registered[string(row.Code)] = row.ExitCode
	}
	for code, wantExit := range map[string]int{
		"invalid_config":          3,
		"provider_protocol_error": 13,
		"incompatible_protocol":   6,
		"provider_process_failed": 13,
		"provider_timeout":        13,
		"integrity_failure":       9,
	} {
		gotExit, ok := registered[code]
		if !ok {
			t.Fatalf("code %q is not in the pinned error registry", code)
		}
		if gotExit != wantExit {
			t.Fatalf("code %q exit = %d, want %d", code, gotExit, wantExit)
		}
		gotWire, err := axerror.ExitCodeFor(axerror.Version100, axerror.Code(code))
		if err != nil {
			t.Fatalf("ExitCodeFor(1.0.0, %q): %v", code, err)
		}
		if gotWire != wantExit {
			t.Fatalf("wire exit for %q = %d, want %d", code, gotWire, wantExit)
		}
	}
	t.Logf("refusal code coverage: 6/6 package codes registered with pinned exits")
}

// TestRefusalConstructorsAreTotal proves the fallible-constructor error
// path never fires in practice: every constructor returns a failure and
// a nil error across representative dynamic causes. The sites recorded
// here are test lines, not production sites, so every key this test adds
// to the exercised set is removed before it returns, keeping the
// reverse-direction audit exact.
func TestRefusalConstructorsAreTotal(t *testing.T) {
	var beforeSites, beforeCodes []any
	exercisedRefusalSites.Range(func(key, _ any) bool { beforeSites = append(beforeSites, key); return true })
	observedCodes.Range(func(key, _ any) bool { beforeCodes = append(beforeCodes, key); return true })
	t.Cleanup(func() {
		keepSites := map[any]bool{}
		for _, key := range beforeSites {
			keepSites[key] = true
		}
		exercisedRefusalSites.Range(func(key, _ any) bool {
			if !keepSites[key] {
				exercisedRefusalSites.Delete(key)
			}
			return true
		})
		keepCodes := map[any]bool{}
		for _, key := range beforeCodes {
			keepCodes[key] = true
		}
		observedCodes.Range(func(key, _ any) bool {
			if !keepCodes[key] {
				observedCodes.Delete(key)
			}
			return true
		})
	})
	cause := fmt.Errorf("fake: dynamic cause")
	longCause := fmt.Errorf("fake: %s", strings.Repeat("x", 300))
	for _, kase := range []struct {
		name string
		call func() (*axerror.Error, error)
		code string
	}{
		{"invalid", func() (*axerror.Error, error) { return failInvalid("fake") }, "invalid_config"},
		{"protocol", func() (*axerror.Error, error) { return failProtocol("fake", "member") }, "provider_protocol_error"},
		{"mismatch", func() (*axerror.Error, error) { return failMismatch("fake", "3.0.0") }, "incompatible_protocol"},
		{"process with cause", func() (*axerror.Error, error) { return failProcess("fake", cause) }, "provider_process_failed"},
		{"process with long cause", func() (*axerror.Error, error) { return failProcess("fake", longCause) }, "provider_process_failed"},
		{"process without cause", func() (*axerror.Error, error) { return failProcess("fake", nil) }, "provider_process_failed"},
		{"timeout", func() (*axerror.Error, error) { return failTimeout("fake", 100) }, "provider_timeout"},
		{"integrity", func() (*axerror.Error, error) { return failIntegrity("fake", "unknown", "m", "t") }, "integrity_failure"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			failure, err := kase.call()
			if err != nil {
				t.Fatalf("constructor error = %v, want nil", err)
			}
			if failure == nil || string(failure.Code()) != kase.code {
				t.Fatalf("constructor failure = %v, want code %s", failure, kase.code)
			}
		})
	}
}
