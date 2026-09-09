package specdoc_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
	"github.com/relux-works/agent-session-manager/internal/specpin"
)

// TestLoadV060AcceptsOnlyTheAdoptedDocumentDigest drives the v0.6.0
// production entry point: the embedded bytes must be the exact 1150005-byte
// SPEC.md blob from the verified signed tag, not the historical document and
// not a worktree copy.
func TestLoadV060AcceptsOnlyTheAdoptedDocumentDigest(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("LoadV060() error = %v", err)
	}
	raw := specdoc.BytesV060()
	if len(raw) != 1150005 {
		t.Fatalf("embedded v0.6.0 document has %d bytes, want 1150005", len(raw))
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != specpin.DocumentSHA256V060 {
		t.Fatalf("embedded v0.6.0 document digest = %s, want pinned %s", got, specpin.DocumentSHA256V060)
	}
	if got := document.LineCount(); got != 16454 {
		t.Fatalf("adopted document has %d lines, want 16454", got)
	}
}

// TestParseV060RefusesEveryNonAdoptedDocument is the anti-vacuity proof for
// the v0.6.0 digest gate. The stale-document row is the adoption-critical
// one: the historical v0.5.0 bytes keep every old token and must still be
// refused, so a gate that has not been deliberately re-pointed cannot
// silently compare against either text.
func TestParseV060RefusesEveryNonAdoptedDocument(t *testing.T) {
	t.Parallel()

	pinned := specdoc.BytesV060()

	appended := append(bytes.Clone(pinned), '\n')
	truncated := bytes.Clone(pinned)[:len(pinned)-1]

	perturbed := bytes.Clone(pinned)
	index := bytes.Index(perturbed, []byte("urn:ax:contract:session-selector"))
	if index < 0 {
		t.Fatal("adopted document does not contain the Session selector contract identifier")
	}
	perturbed[index] = 'U'

	whitespaceOnly := bytes.Clone(pinned)
	space := bytes.IndexByte(whitespaceOnly, ' ')
	if space < 0 {
		t.Fatal("adopted document contains no space byte")
	}
	whitespaceOnly[space] = '\t'

	for _, test := range []struct {
		name      string
		candidate []byte
	}{
		{name: "absent", candidate: nil},
		{name: "empty", candidate: []byte{}},
		{name: "truncated read", candidate: truncated},
		{name: "appended byte", candidate: appended},
		{name: "single character substitution", candidate: perturbed},
		{name: "whitespace-only substitution", candidate: whitespaceOnly},
		{name: "unrelated document", candidate: []byte("# Not the specification\n")},
		{name: "stale v0.5.0 document", candidate: specdoc.Bytes()},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := specdoc.ParseV060(test.candidate); !errors.Is(err, specdoc.ErrDocumentMismatch) {
				t.Fatalf("ParseV060(%s) error = %v, want ErrDocumentMismatch", test.name, err)
			}
		})
	}
}

// TestParseRefusesAdoptedDocument keeps the old gate version-bound from the
// other side: the v0.5.0 entry point must not accept the adopted text, so
// the traceability comparison cannot drift onto the new document without
// its owning task re-pointing it.
func TestParseRefusesAdoptedDocument(t *testing.T) {
	t.Parallel()

	if _, err := specdoc.Parse(specdoc.BytesV060()); !errors.Is(err, specdoc.ErrDocumentMismatch) {
		t.Fatalf("Parse(v0.6.0 bytes) error = %v, want ErrDocumentMismatch", err)
	}
}

// TestQuoteLinesFindsSelectorExcerptAtItsStartingLine proves the adopted
// document is really the selector/auth/migration text: a hard-wrapped
// Section 14.7 excerpt resolves to the line where it starts.
func TestQuoteLinesFindsSelectorExcerptAtItsStartingLine(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("LoadV060() error = %v", err)
	}

	const wrapped = "This section is the v0.6.0 selector contract. It applies to the umbrella <code>ax SELECTOR</code> and every existing managed-session NAME operand in Section 14, including status, attach, takeover, fork, stop, resume, sync, diff, materialize, logs --session and session set-profile."
	lines := document.QuoteLines(wrapped)
	if len(lines) != 1 || lines[0] != 11870 {
		t.Fatalf("QuoteLines(selector) = %v, want exactly [11870]", lines)
	}
	line, ok := document.Line(lines[0])
	if !ok || !strings.Contains(line, "This section is the v0.6.0 selector contract") {
		t.Fatalf("line %d = %q, want the selector quote's first line", lines[0], line)
	}
}

// TestSectionIDResolvesAdoptedClauses pins the clause resolver on the new
// sections: a citation to the Configuration 4.0.0 table, the Host Channel
// launch prose, or the selector contract must resolve to the clause the
// v0.6.0 revision added, not to a neighbouring one.
func TestSectionIDResolvesAdoptedClauses(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("LoadV060() error = %v", err)
	}

	for _, test := range []struct {
		line int
		want string
	}{
		{line: 2644, want: "6.6"},     // schema_version row of the Config 4 table
		{line: 8016, want: "11.10.1"}, // Config-4 initiator launch prose
		{line: 11870, want: "14.7"},   // selector contract opening
		{line: 2640, want: "6.6"},     // prose just above the Config 4 table
		{line: 1487, want: "5.1"},     // Session Record name row, shifted by the v0.6.0 registry rows
	} {
		got, ok := document.SectionID(test.line)
		if !ok || got != test.want {
			t.Errorf("SectionID(%d) = %q, %v; want %q", test.line, got, ok, test.want)
		}
	}

	for _, number := range []int{0, -1, document.LineCount() + 1} {
		if id, ok := document.SectionID(number); ok {
			t.Errorf("SectionID(%d) = %q, true; want not found", number, id)
		}
	}
	if id, ok := document.SectionID(1); ok {
		t.Errorf("SectionID(1) = %q, true; the document title opens no clause", id)
	}
}

// TestTableRowAtV060DeclaresAdoptedRow pins the table index on the new
// Configuration 4.0.0 table: the schema_version body row declares itself,
// while the header and delimiter above it declare nothing.
func TestTableRowAtV060DeclaresAdoptedRow(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("LoadV060() error = %v", err)
	}

	row, ok := document.TableRowAt(2644)
	if !ok {
		t.Fatal("TableRowAt(2644) reports no declaration for the schema_version row")
	}
	if row.Header != "Key" || row.Identifier != "schema_version" || row.Line != 2644 {
		t.Fatalf("TableRowAt(2644) = %+v, want header %q identifier %q", row, "Key", "schema_version")
	}
	for _, line := range []int{2642, 2643, 0, document.LineCount() + 1} {
		if _, ok := document.TableRowAt(line); ok {
			t.Errorf("TableRowAt(%d) = true, want not found (header, delimiter, or out of range)", line)
		}
	}
}
