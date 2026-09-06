package dirnode

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// This file derives the closed registries from the pinned
// specification text rather than listing them: a dropped, added,
// or reordered registry member reddens here rather than passing
// silently. Every derivation fails closed — an empty scan, a
// missing window, or an ambiguous marker is a test failure, never
// a vacuous pass — because a domain that silently derives nothing
// is not a measurement.
//
// The refusal-arm inventory lives in census_test.go; the import
// census pinning the identifier match lives below.

// specText loads the pinned document bytes the derivations scan.
// specdoc.Load verifies the digest first, so a substituted,
// truncated, or partially read document is refused instead of
// being silently compared against.
func specText(t *testing.T) string {
	t.Helper()
	if _, err := specdoc.Load(); err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	return string(specdoc.Bytes())
}

// specWindow returns the document text between the marker and the
// terminator. Both must be present exactly once and in order;
// otherwise the window is untrusted and the derivation fails
// closed.
func specWindow(t *testing.T, text, marker, terminator string) string {
	t.Helper()
	if strings.Count(text, marker) != 1 {
		t.Fatalf("marker %q occurs %d times, want exactly once", marker, strings.Count(text, marker))
	}
	if strings.Count(text, terminator) != 1 {
		t.Fatalf("terminator %q occurs %d times, want exactly once", terminator, strings.Count(text, terminator))
	}
	start := strings.Index(text, marker)
	end := strings.Index(text, terminator)
	if start >= end {
		t.Fatalf("marker %q does not precede terminator %q", marker, terminator)
	}
	return text[start:end]
}

// specCodeTokens extracts every <code>...</code> token from the
// window in order. An empty extraction fails closed.
func specCodeTokens(t *testing.T, window string) []string {
	t.Helper()
	var tokens []string
	for {
		open := strings.Index(window, "<code>")
		if open < 0 {
			break
		}
		rest := window[open+len("<code>"):]
		close := strings.Index(rest, "</code>")
		if close < 0 {
			t.Fatal("unbalanced <code> span in spec window")
		}
		tokens = append(tokens, rest[:close])
		window = rest[close+len("</code>"):]
	}
	if len(tokens) == 0 {
		t.Fatal("derived zero code tokens; the scan is broken, not the registry")
	}
	return tokens
}

// TestOperationRegistryDerivedFromSpec requires the production
// dispatch registry to equal the Section 7.9 operation table in
// order: every table row appears in production at the same
// position, and nothing else does.
func TestOperationRegistryDerivedFromSpec(t *testing.T) {
	text := specText(t)
	table := specWindow(t, text, "The exact operation registry and bodies are:", "Each displayed body is closed")
	var operations []string
	for _, line := range strings.Split(table, "\n") {
		if !strings.HasPrefix(line, "| <code>") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 2 {
			t.Fatalf("malformed table row %q", line)
		}
		name := strings.TrimSpace(cells[1])
		name = strings.TrimPrefix(name, "<code>")
		name = strings.TrimSuffix(name, "</code>")
		if !validOperation(name) {
			t.Fatalf("derived operation %q is not a production registry member", name)
		}
		operations = append(operations, name)
	}
	if len(operations) != 11 {
		t.Fatalf("derived %d operation rows, want 11; the table scan is broken, not the registry", len(operations))
	}
	got := Operations()
	if strings.Join(got, "\x00") != strings.Join(operations, "\x00") {
		t.Fatalf("Operations() = %v, want spec table %v", got, operations)
	}
}

// TestCapabilityRegistryDerivedFromSpec requires the production
// capability registry to equal the Section 7.9 capability
// sentence in order.
func TestCapabilityRegistryDerivedFromSpec(t *testing.T) {
	text := specText(t)
	window := specWindow(t, text, "The capability\nregistry is exactly", "and <code>native_resume</code>.")
	tokens := specCodeTokens(t, window)
	tokens = append(tokens, "native_resume")
	if len(tokens) != 8 {
		t.Fatalf("derived %d capability names, want 8", len(tokens))
	}
	got := Capabilities()
	if strings.Join(got, "\x00") != strings.Join(tokens, "\x00") {
		t.Fatalf("Capabilities() = %v, want spec sentence %v", got, tokens)
	}
}

// TestQueryRegistryDerivedFromSpec requires the production query
// registries to equal the Section 10.8.5 read/mutation sentences
// in order: eleven reads, then six mutations.
func TestQueryRegistryDerivedFromSpec(t *testing.T) {
	text := specText(t)
	window := specWindow(t, text, "Read operations are exactly", "there is no delete.")
	tokens := specCodeTokens(t, window)
	split := -1
	for index, token := range tokens {
		if token == "set_title" {
			split = index
			break
		}
	}
	if split < 0 {
		t.Fatal("mutation registry not found in query window")
	}
	reads, mutations := tokens[:split], tokens[split:]
	if strings.Join(reads, "\x00") != strings.Join(readOperations, "\x00") {
		t.Fatalf("read registry = %v, want spec %v", readOperations, reads)
	}
	if strings.Join(mutations, "\x00") != strings.Join(mutationOperations, "\x00") {
		t.Fatalf("mutation registry = %v, want spec %v", mutationOperations, mutations)
	}
}

// TestFramedSubsetIsExact pins the deliverable boundary: the
// framed subset is exactly manifest, probe, and scan, and the
// unframed remainder is exactly the eight named registry members
// whose nested content owners are the sibling leaves. A fourth
// framed operation, or a ninth unframed one, reddens here.
func TestFramedSubsetIsExact(t *testing.T) {
	t.Parallel()
	framed := FramedOperations()
	want := []string{"manifest", "probe", "scan"}
	if strings.Join(framed, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("FramedOperations() = %v, want %v", framed, want)
	}
	framedSet := map[string]bool{}
	for _, name := range framed {
		framedSet[name] = true
	}
	var unframed []string
	for _, name := range Operations() {
		if !framedSet[name] {
			unframed = append(unframed, name)
		}
	}
	wantUnframed := []string{"inventory", "preview", "enrichment-plan", "enrichment-run", "enrichment-status", "continuation-inspect", "runtime-observe", "doctor"}
	sort.Strings(unframed)
	sort.Strings(wantUnframed)
	if strings.Join(unframed, "\x00") != strings.Join(wantUnframed, "\x00") {
		t.Fatalf("unframed operations = %v, want %v", unframed, wantUnframed)
	}
}

// axerrorImportPath is the only import path the constructor census
// resolves. It is spelled here rather than derived because the
// census must pin production, not follow it: a moved package with
// no row here fails the zero-importer guard below.
const axerrorImportPath = "github.com/relux-works/agent-session-manager/internal/axerror"

// TestAxerrorImportsAreUnaliased pins the import spelling every
// identifier-matched census in this package relies on. A
// production file importing the error package under any other
// name hides every refusal it builds from both the constructor
// census and the arm derivation at once. An empty scan, or a tree
// with no production importer at all, fails closed.
func TestAxerrorImportsAreUnaliased(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("import census: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("import census: %v", err)
	}
	scanned := 0
	importing := 0
	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("import census: %v", err)
		}
		syntax, err := parseGoFile(path, source)
		if err != nil {
			t.Fatalf("import census: %v", err)
		}
		for _, declaration := range syntax.Imports {
			target, err := strconv.Unquote(declaration.Path.Value)
			if err != nil || target != axerrorImportPath {
				continue
			}
			importing++
			if declaration.Name != nil && declaration.Name.Name != "axerror" {
				violations = append(violations, name+": imports the error package as "+declaration.Name.Name)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("import census scanned zero production files; the scanner is broken, not the package")
	}
	if importing == 0 {
		t.Fatal("import census found zero production importers of the error package; the census is vacuous, not green")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("aliased error-package import(s) hide constructions from the constructor census:\n  %s", strings.Join(violations, "\n  "))
	}
	t.Logf("import census: %d production importer(s) bind axerror across %d files", importing, scanned)
}
