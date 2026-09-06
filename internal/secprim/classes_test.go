package secprim

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// derivedExclusionClasses derives the Section 16.2 mandatory-exclusion
// classes from the pinned specification document itself: every body row of
// the Class table in Section 16.2. A class added to or removed from the
// specification fails the roster below instead of passing silently.
func derivedExclusionClasses(t *testing.T) []string {
	t.Helper()
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	var classes []string
	seen := map[string]bool{}
	for number := 1; number <= document.LineCount(); number++ {
		row, ok := document.TableRowAt(number)
		if !ok || row.Header != "Class" {
			continue
		}
		section, ok := document.SectionID(row.Line)
		if !ok || section != "16.2" {
			continue
		}
		if seen[row.FirstCell] {
			continue
		}
		seen[row.FirstCell] = true
		classes = append(classes, row.FirstCell)
	}
	if len(classes) == 0 {
		t.Fatal("derived no Section 16.2 exclusion classes; the scanner is broken, not the document")
	}
	sort.Strings(classes)
	return classes
}

// exclusionRow pairs one derived Section 16.2 class with its disposition:
// which mechanism keeps that class out of artifacts, and which test
// proves the mechanism. Every derived class carries exactly one row.
type exclusionRow struct {
	class       string
	disposition string
	witnesses   []string
}

// exclusionRows is the reviewed disposition half of the Section 16.2
// census. Dispositions are one of: scrub (the Redact stream arms),
// refuse (a boundary gate), structural (never logged by construction),
// or nopath (no in-repo emission path carries the class — the M0-stage
// bound, proven by TestNoArchiveEmissionPath plus the secret-site
// roster).
func exclusionRows() []exclusionRow {
	return []exclusionRow{
		{
			class: "Provider credentials",
			disposition: "scrub for free text (Redact corpus and key=value arms) and refuse for structured " +
				"diagnostics (the axerror excluded-key gate)",
			witnesses: []string{
				"secprim.TestRedactSensitivePairs",
				"secprim.TestSecretAxerrorTableRefused",
			},
		},
		{
			class: "SSH/private identity",
			disposition: "scrub for free text (the Redact private-key block arm for key material, corpus and " +
				"key arms for the rest) and refuse for structured diagnostics",
			witnesses: []string{
				"secprim.TestRedactPrivateKeyBlocks",
				"secprim.TestSecretAxerrorTableRefused",
			},
		},
		{
			class: "Environment secrets",
			disposition: "scrub for free text and refuse for structured diagnostics; SpawnPlan literals are " +
				"additionally bounded and disjoint by the provhost wire gate, whose secrecy itself is an " +
				"operator obligation no content inspection could prove",
			witnesses: []string{
				"secprim.TestRedactCorpusValues",
				"secprim.TestSecretAxerrorTableRefused",
				"provhost.TestDecodeSpawnPlanRefusals",
			},
		},
		{
			class: "Machine authentication",
			disposition: "nopath (archive-format bound): the recursive import scan proves no production file " +
				"imports an archive writer that could carry keychain, secret-service, credential-manager, or " +
				"login-token handling into a bundle — not that no such handling exists anywhere; the " +
				"secret-site roster proves no field whose name carries one of the 13 census tokens exists " +
				"outside the dispositioned tables (names outside that vocabulary are outside the witness, " +
				"stated in the census bound); key-shaped text is still scrubbed where it matches",
			witnesses: []string{
				"secprim.TestSecretSiteRosterIsComplete",
				"secprim.TestNoArchiveEmissionPath",
			},
		},
		{
			class: "Live process identity",
			disposition: "nopath (archive-format bound) for transfer plus structural for diagnostics: the " +
				"recursive import scan proves no archive-writer path exists to audit for PID or handle " +
				"transfer — not the absence of every non-archive emission shape — and PIDs and handles " +
				"never enter Structured Errors because the causal-leak gate refuses cause text verbatim",
			witnesses: []string{
				"secprim.TestNoArchiveEmissionPath",
				"axerror.TestCauseNeverReachesTheWire",
			},
		},
		{
			class: "IPC/runtime",
			disposition: "nopath (archive-format bound): the recursive import scan proves no archive-writer " +
				"transfer path exists — not the absence of every socket, pipe, or tmux-server-socket shape; " +
				"socket-shaped names in diagnostics are inert text",
			witnesses: []string{
				"secprim.TestNoArchiveEmissionPath",
				"secprim.TestSecretSiteRosterIsComplete",
			},
		},
		{
			class: "TerminalBackend runtime",
			disposition: "refuse by vocabulary: bindings, native references, endpoints, tokens, relay " +
				"credentials, and observations are classified sensitive-owner-only or forbidden and never " +
				"replicate as live authority",
			witnesses: []string{
				"terminalbackend.TestReplicationClassificationIsClosed",
			},
		},
		{
			class: "Transient locking",
			disposition: "nopath (archive-format bound): the recursive import scan proves lock files, WAL/SHM " +
				"journals, and PID locks have no archive-writer emission path — not the absence of every " +
				"non-archive export shape; the owner-only staging roots they guard are validated by the " +
				"no-follow gates",
			witnesses: []string{
				"secprim.TestNoArchiveEmissionPath",
				"secprim.TestOpenNoFollowDir",
			},
		},
		{
			class: "Mutable derived indexes",
			disposition: "nopath (archive-format bound): the recursive import scan proves SQLite projections " +
				"and provider caches have no archive-writer export path — not the absence of every " +
				"non-archive export shape; they are rebuilt locally",
			witnesses: []string{
				"secprim.TestNoArchiveEmissionPath",
				"secprim.TestSecretSiteRosterIsComplete",
			},
		},
	}
}

// TestExclusionClassRosterIsComplete requires every derived Section 16.2
// class to carry exactly one disposition row, and every row to resolve to
// a derived class.
func TestExclusionClassRosterIsComplete(t *testing.T) {
	derived := derivedExclusionClasses(t)
	rows := exclusionRows()
	seen := map[string]bool{}
	for _, row := range rows {
		found := false
		for _, class := range derived {
			if row.class == class {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("orphaned exclusion row %q: the specification derives no such class", row.class)
		}
		if seen[row.class] {
			t.Fatalf("duplicate exclusion row %q", row.class)
		}
		seen[row.class] = true
	}
	var missing []string
	for _, class := range derived {
		if !seen[class] {
			missing = append(missing, class)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("derived exclusion classes with no disposition:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(derived) != 9 {
		t.Fatalf("derived %d exclusion classes, want the 9 of Section 16.2: %q", len(derived), derived)
	}
}

// TestExclusionWitnessesExist verifies every cited witness names a real
// test, reusing the declaration walk from the site census.
func TestExclusionWitnessesExist(t *testing.T) {
	t.Parallel()
	needed := map[string]map[string]bool{}
	for _, row := range exclusionRows() {
		for _, witness := range row.witnesses {
			parts := strings.Split(witness, ".")
			if len(parts) != 2 {
				t.Fatalf("witness %q is not package.Test shaped", witness)
			}
			if needed[parts[0]] == nil {
				needed[parts[0]] = map[string]bool{}
			}
			needed[parts[0]][parts[1]] = true
		}
	}
	for pkg, names := range needed {
		found := testFuncNames(t, pkg)
		for name := range names {
			if !found[name] {
				t.Errorf("witness %s.%s does not exist; the census cites a ghost", pkg, name)
			}
		}
	}
}

// TestNoArchiveEmissionPath proves the archive-format bound behind the
// nopath dispositions above: no production file anywhere under
// internal/ imports an archive writer, so no code path can carry an
// excluded class into a bundle through one.
//
// The walk is recursive over every nested production package
// (internal/catalog/cmd/cataloggen and
// internal/traceability/cmd/tracecheck included): a one-level scan
// proved blind to nested trees. A file the parser cannot read fails
// the test rather than passing silently, so an unparseable package is
// a closed failure, never a gap.
//
// Stated bound, not a no-emission proof: the walk establishes the
// absence of three archive-format imports, not the absence of manifest
// or bundle emission in general — a JSON manifest needs none of those
// imports. The nopath rows cite this test for exactly that bound and
// no more. The day an archive or transfer writer lands, this test
// names it and the nopath rows must be re-dispositioned.
func TestNoArchiveEmissionPath(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "internal"))
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"archive/tar", "archive/zip", "compress/gzip"}
	var violations []string
	scanned := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		syntax, err := parser.ParseFile(token.NewFileSet(), name, contents, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, imported := range syntax.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, denied := range forbidden {
				if importPath == denied {
					rel, relErr := filepath.Rel(root, path)
					if relErr != nil {
						rel = path
					}
					violations = append(violations, "internal/"+rel+" imports "+importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("emission-path walk failed closed: %v", err)
	}
	if scanned == 0 {
		t.Fatal("scanned no production sources; the check is blind")
	}
	t.Logf("emission-path scan covered %d production files", scanned)
	if len(violations) > 0 {
		t.Fatalf("archive emission paths exist; nopath dispositions must be re-proven:\n  %s", strings.Join(violations, "\n  "))
	}
}

// Keep the AST imports referenced for the emission-path walk above.
var _ = ast.Walk
