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

// TestLoadV070AcceptsOnlyTheAdoptedDocumentDigest drives the v0.7.0
// production entry point: the embedded bytes must be the exact 1185291-byte
// SPEC.md blob from the verified signed tag, not a historical document and
// not a worktree copy.
func TestLoadV070AcceptsOnlyTheAdoptedDocumentDigest(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV070()
	if err != nil {
		t.Fatalf("LoadV070() error = %v", err)
	}
	raw := specdoc.BytesV070()
	if len(raw) != 1185291 {
		t.Fatalf("embedded v0.7.0 document has %d bytes, want 1185291", len(raw))
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != specpin.DocumentSHA256V070 {
		t.Fatalf("embedded v0.7.0 document digest = %s, want pinned %s", got, specpin.DocumentSHA256V070)
	}
	if got := document.LineCount(); got != 16909 {
		t.Fatalf("adopted document has %d lines, want 16909", got)
	}
}

// TestParseV070RefusesEveryNonAdoptedDocument is the anti-vacuity proof for
// the v0.7.0 digest gate. The stale-document rows are the adoption-critical
// ones: the historical v0.6.0 and v0.5.0 bytes keep every old token and must
// still be refused, so a gate that has not been deliberately re-pointed
// cannot silently compare against any text but the adopted one.
func TestParseV070RefusesEveryNonAdoptedDocument(t *testing.T) {
	t.Parallel()

	pinned := specdoc.BytesV070()

	appended := append(bytes.Clone(pinned), '\n')
	truncated := bytes.Clone(pinned)[:len(pinned)-1]

	perturbed := bytes.Clone(pinned)
	index := bytes.Index(perturbed, []byte("urn:ax:schema:launch-plan-request"))
	if index < 0 {
		t.Fatal("adopted document does not contain the Launch Plan request contract identifier")
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
		{name: "stale v0.6.0 document", candidate: specdoc.BytesV060()},
		{name: "stale v0.5.0 document", candidate: specdoc.Bytes()},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := specdoc.ParseV070(test.candidate); !errors.Is(err, specdoc.ErrDocumentMismatch) {
				t.Fatalf("ParseV070(%s) error = %v, want ErrDocumentMismatch", test.name, err)
			}
		})
	}
}

// TestParseRefusesAdoptedV070Document keeps the old gates version-bound from
// the other side: neither the v0.5.0 nor the v0.6.0 entry point may accept
// the adopted text, so a comparison cannot drift onto the new document
// without its owning task re-pointing it.
func TestParseRefusesAdoptedV070Document(t *testing.T) {
	t.Parallel()

	if _, err := specdoc.Parse(specdoc.BytesV070()); !errors.Is(err, specdoc.ErrDocumentMismatch) {
		t.Fatalf("Parse(v0.7.0 bytes) error = %v, want ErrDocumentMismatch", err)
	}
	if _, err := specdoc.ParseV060(specdoc.BytesV070()); !errors.Is(err, specdoc.ErrDocumentMismatch) {
		t.Fatalf("ParseV060(v0.7.0 bytes) error = %v, want ErrDocumentMismatch", err)
	}
}

// TestQuoteLinesFindsLaunchPlanExcerptAtItsStartingLine proves the adopted
// document is really the launch-plan revision text: a hard-wrapped Section
// 15.3 excerpt about the one new literal code resolves to the line where it
// starts.
func TestQuoteLinesFindsLaunchPlanExcerptAtItsStartingLine(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV070()
	if err != nil {
		t.Fatalf("LoadV070() error = %v", err)
	}

	const wrapped = "The revision carrying Section 14.1 <code>ax start --launch-plan</code> adds exactly one literal code:"
	lines := document.QuoteLines(wrapped)
	if len(lines) != 1 || lines[0] != 15304 {
		t.Fatalf("QuoteLines(launch-plan) = %v, want exactly [15304]", lines)
	}
	line, ok := document.Line(lines[0])
	if !ok || !strings.Contains(line, "The revision carrying Section 14.1") {
		t.Fatalf("line %d = %q, want the launch-plan quote's first line", lines[0], line)
	}
}

// TestSectionIDResolvesV070Clauses pins the clause resolver on the new
// content: a citation to the Launch Plan registry row, the unnumbered
// caller-launch-plan-code table, or the new appendix subsection must resolve
// to its clause, not to a neighbouring one.
func TestSectionIDResolvesV070Clauses(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV070()
	if err != nil {
		t.Fatalf("LoadV070() error = %v", err)
	}

	for _, test := range []struct {
		line int
		want string
	}{
		{line: 141, want: "1.5"},    // Session Record registry row, widened with 3.1.0
		{line: 142, want: "1.5"},    // Launch Plan request registry row
		{line: 15302, want: "15.3"}, // unnumbered Caller launch-plan code heading stays in 15.3
		{line: 15309, want: "15.3"}, // launch_plan_invalid table row
		{line: 16518, want: "A.12"}, // new appendix subsection heading
		{line: 16530, want: "A.12"}, // first traceability row of A.12
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

// TestTableRowAtV070DeclaresAdoptedRow pins the table index on the new
// caller-launch-plan-code table: the exit-2 body row declares itself, while
// the header and delimiter above it declare nothing.
func TestTableRowAtV070DeclaresAdoptedRow(t *testing.T) {
	t.Parallel()

	document, err := specdoc.LoadV070()
	if err != nil {
		t.Fatalf("LoadV070() error = %v", err)
	}

	row, ok := document.TableRowAt(15309)
	if !ok {
		t.Fatal("TableRowAt(15309) reports no declaration for the launch_plan_invalid row")
	}
	if row.Header != "Exit" || row.Identifier != "2" || row.Line != 15309 {
		t.Fatalf("TableRowAt(15309) = %+v, want header %q identifier %q", row, "Exit", "2")
	}
	for _, line := range []int{15307, 15308, 0, document.LineCount() + 1} {
		if _, ok := document.TableRowAt(line); ok {
			t.Errorf("TableRowAt(%d) = true, want not found (header, delimiter, or out of range)", line)
		}
	}
}
