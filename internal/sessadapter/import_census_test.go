package sessadapter

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file is the import census the constructor census stands on
// (review rev3 B6). TestRefusalConstructorsMatchProduction matches
// error constructions by the package identifier `axerror`, so a
// production file importing the error package under any other name
// hides every refusal it builds from both the constructor census
// and the arm derivation at once. This test requires every
// production import of the error package path to bind the name
// `axerror` — the default spelling or an explicit identical alias —
// and fails on any other binding. An empty scan, or a tree with no
// production importer at all, fails closed: a census that sees
// nothing measures nothing.

// axerrorImportPath is the only import path the constructor census
// resolves. It is spelled here rather than derived because the
// census must pin production, not follow it: a moved package with
// no row here fails the zero-importer guard below.
const axerrorImportPath = "github.com/relux-works/agent-session-manager/internal/axerror"

// TestAxerrorImportsAreUnaliased pins the import spelling every
// identifier-matched census in this package relies on.
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
		syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
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

// TestAxerrorImportCensusSeesAliases proves the census shape
// against synthetic imports: the default binding and an explicit
// identical alias pass, while a renamed, blank, or dot import
// fails. The vectors are synthetic on purpose: they prove the
// gate, not production.
func TestAxerrorImportCensusSeesAliases(t *testing.T) {
	aliased := func(source string) bool {
		t.Helper()
		syntax, err := parser.ParseFile(token.NewFileSet(), "probe.go", []byte(source), parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		for _, declaration := range syntax.Imports {
			target, err := strconv.Unquote(declaration.Path.Value)
			if err != nil || target != axerrorImportPath {
				continue
			}
			if declaration.Name != nil && declaration.Name.Name != "axerror" {
				return true
			}
		}
		return false
	}
	path := strconv.Quote(axerrorImportPath)
	for _, probe := range []struct {
		name   string
		source string
		want   bool
	}{
		{"default binding passes", "package probe\nimport " + path + "\n", false},
		{"explicit identical alias passes", "package probe\nimport axerror " + path + "\n", false},
		{"renamed import fails", "package probe\nimport axe " + path + "\n", true},
		{"blank import fails", "package probe\nimport _ " + path + "\n", true},
		{"dot import fails", "package probe\nimport . " + path + "\n", true},
	} {
		t.Run(probe.name, func(t *testing.T) {
			if got := aliased(probe.source); got != probe.want {
				t.Fatalf("aliased(%q) = %v, want %v", probe.source, got, probe.want)
			}
		})
	}
}
