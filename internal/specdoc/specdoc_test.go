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

func TestLoadAcceptsOnlyThePinnedDocumentDigest(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	digest := sha256.Sum256(specdoc.Bytes())
	if got := hex.EncodeToString(digest[:]); got != specpin.DocumentSHA256 {
		t.Fatalf("embedded document digest = %s, want pinned %s", got, specpin.DocumentSHA256)
	}
	if document.LineCount() < 1000 {
		t.Fatalf("pinned document has %d lines, want the full specification", document.LineCount())
	}
}

// TestParseRefusesEveryNonPinnedDocument is the anti-vacuity proof for the
// digest gate. A comparison against a swapped specification would confirm
// whatever that swapped text happens to say, so every perturbation must be
// refused rather than parsed.
func TestParseRefusesEveryNonPinnedDocument(t *testing.T) {
	t.Parallel()

	pinned := specdoc.Bytes()

	appended := append(bytes.Clone(pinned), '\n')
	truncated := bytes.Clone(pinned)[:len(pinned)-1]

	perturbed := bytes.Clone(pinned)
	index := bytes.Index(perturbed, []byte("urn:ax:schema:blob"))
	if index < 0 {
		t.Fatal("pinned document does not contain the Blob Descriptor schema identifier")
	}
	perturbed[index] = 'U'

	whitespaceOnly := bytes.Clone(pinned)
	space := bytes.IndexByte(whitespaceOnly, ' ')
	if space < 0 {
		t.Fatal("pinned document contains no space byte")
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
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := specdoc.Parse(test.candidate); !errors.Is(err, specdoc.ErrDocumentMismatch) {
				t.Fatalf("Parse(%s) error = %v, want ErrDocumentMismatch", test.name, err)
			}
		})
	}
}

func TestQuoteLinesFindsWrappedTextAtItsStartingLine(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// A quote that the specification hard-wraps across two lines must resolve
	// to the line where it starts, otherwise every wrapped excerpt would have
	// to be shortened until it fit one line.
	const wrapped = "Every transferred blob has a Blob Descriptor with schema <code>urn:ax:schema:blob</code> version <code>1.0.0</code>."
	lines := document.QuoteLines(wrapped)
	if len(lines) != 1 {
		t.Fatalf("QuoteLines(wrapped) = %v, want exactly one line", lines)
	}
	line, ok := document.Line(lines[0])
	if !ok || !strings.Contains(line, "Every transferred blob has a Blob Descriptor") {
		t.Fatalf("line %d = %q, want the wrapped quote's first line", lines[0], line)
	}
	if _, ok := document.Line(lines[0] + 1); !ok {
		t.Fatalf("line %d has no successor, the quote cannot wrap", lines[0])
	}
}

func TestLineRefusesOutOfRangeNumbers(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// A paraphrase that names a line outside the document must be reported as
	// absent rather than silently resolving to some other line.
	for _, number := range []int{0, -1, document.LineCount() + 1, 1 << 30} {
		if text, ok := document.Line(number); ok {
			t.Errorf("Line(%d) = %q, true; want not found", number, text)
		}
	}
	if _, ok := document.Line(1); !ok {
		t.Fatal("Line(1) is not readable")
	}
	if _, ok := document.Line(document.LineCount()); !ok {
		t.Fatal("the last line is not readable")
	}
}

func TestQuoteLinesRefusesAbsentAndEmptyText(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, text := range []string{
		"",
		"   \n\t ",
		"Every transferred blob has a Blob Descriptor with schema <code>urn:ax:schema:digest</code>",
		"the quoted word digest reached the Pinned SPEC declaration column",
	} {
		if lines := document.QuoteLines(text); len(lines) != 0 {
			t.Errorf("QuoteLines(%q) = %v, want no match", text, lines)
		}
		if document.Contains(text) {
			t.Errorf("Contains(%q) = true, want false", text)
		}
	}
}

// TestNormalizeForgivesOnlyWhitespace pins the normalization rule. A looser
// rule would let an excerpt match text the specification does not contain.
func TestNormalizeForgivesOnlyWhitespace(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "wrapped", input: "sorted,\ncontiguous", want: "sorted, contiguous"},
		{name: "table indent", input: "  | `a` |  `b` |  ", want: "| `a` | `b` |"},
		{name: "tabs and returns", input: "a\t\r\nb", want: "a b"},
		{name: "runs collapse", input: "a      b", want: "a b"},
		{name: "case preserved", input: "MUST Be", want: "MUST Be"},
		{name: "punctuation preserved", input: "uint53[1..4194304];", want: "uint53[1..4194304];"},
		{name: "markup preserved", input: "<code>size</code>", want: "<code>size</code>"},
	} {
		if got := specdoc.Normalize(test.input); got != test.want {
			t.Errorf("Normalize(%s) = %q, want %q", test.name, got, test.want)
		}
	}

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Case, punctuation, and markup differences must not be normalized away.
	for _, text := range []string{
		"every transferred blob has a blob descriptor",
		"Every transferred blob has a Blob Descriptor with schema urn:ax:schema:blob",
		"Every transferred blob has a Blob Descriptor with schema `urn:ax:schema:blob`",
	} {
		if document.Contains(text) {
			t.Errorf("Contains(%q) = true; normalization is too loose", text)
		}
	}
}

// TestQuoteLinesRefusesTextThatSpansABlankLine pins the blank-line boundary of
// the normalization rule. Collapsing every whitespace run to one space also
// collapsed blank lines, so a "verbatim" excerpt could stitch the tail of one
// block to the head of the next and still be reported as quoting the pinned
// document. That is exactly the looseness the excerpt gate exists to refuse.
//
// The stitched text below is real: both halves are verbatim SPEC.md, taken from
// the two blocks that meet at line 4613 and 4615. It must not match, while each
// half on its own must.
func TestQuoteLinesRefusesTextThatSpansABlankLine(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	const (
		before   = "sorted, contiguous, non-overlapping, start at offset zero, and cover exactly <code>size</code> bytes:"
		after    = "The descriptor is closed and contains exactly <code>schema</code>"
		stitched = before + " " + after
	)

	if lines := document.QuoteLines(before); len(lines) == 0 {
		t.Fatalf("the block before the blank line is not quotable; the fixture is stale")
	}
	if lines := document.QuoteLines(after); len(lines) == 0 {
		t.Fatalf("the block after the blank line is not quotable; the fixture is stale")
	}
	if lines := document.QuoteLines(stitched); len(lines) != 0 {
		t.Fatalf("text spanning a blank line matched at %v; a quote may not stitch two blocks", lines)
	}
	if document.Contains(stitched) {
		t.Fatal("Contains admitted text spanning a blank line")
	}
}

// TestNormalizeNeverEmitsTheBlockSeparator is why the boundary above holds. The
// document projection separates blocks with a byte no normalized excerpt can
// produce, so the refusal is structural rather than a special case in the
// caller.
func TestNormalizeNeverEmitsTheBlockSeparator(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"one\n\ntwo",
		"one\n \n\t\n two",
		"\n\n\n",
		"one\ntwo",
		"  leading and trailing  ",
	} {
		if strings.ContainsRune(specdoc.Normalize(text), specdoc.BlockSeparator) {
			t.Fatalf("Normalize(%q) emitted the block separator", text)
		}
	}
	if got, want := specdoc.Normalize("one\n\ntwo"), "one two"; got != want {
		t.Fatalf("Normalize collapsed a blank line to %q, want %q", got, want)
	}
}

// TestSectionIDResolvesTheEnclosingNumberedClause pins the clause resolver the
// enumeration's shape anchor depends on. A resolver that answered "" for a
// whole region, or that let an unnumbered subheading open a clause, would let a
// foreign citation through by accident rather than by rule.
func TestSectionIDResolvesTheEnclosingNumberedClause(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, test := range []struct {
		line int
		want string
	}{
		{line: 4622, want: "10.2"}, // BlobChunk's size clause
		{line: 4746, want: "10.4"}, // ManifestEntry.file's row
		{line: 1908, want: "5.3"},  // Lease Record reason
		{line: 363, want: "2.1"},   // Terms table name grammar
		{line: 1471, want: "5.1"},  // Session Record name row
	} {
		got, ok := document.SectionID(test.line)
		if !ok || got != test.want {
			t.Errorf("SectionID(%d) = %q, %v; want %q", test.line, got, ok, test.want)
		}
	}

	// Out of range fails closed rather than resolving to some other clause.
	for _, number := range []int{0, -1, document.LineCount() + 1} {
		if id, ok := document.SectionID(number); ok {
			t.Errorf("SectionID(%d) = %q, true; want not found", number, id)
		}
	}

	// The title line precedes every numbered clause and must resolve to none.
	if id, ok := document.SectionID(1); ok {
		t.Errorf("SectionID(1) = %q, true; the document title opens no clause", id)
	}

	// An unnumbered subheading does not open a clause: its body stays inside the
	// numbered clause above it, which is the clause a citation belongs to.
	var unnumbered int
	for number := 1; number <= document.LineCount(); number++ {
		line, _ := document.Line(number)
		if strings.HasPrefix(line, "#### Managed Replica Marker document") {
			unnumbered = number
			break
		}
	}
	if unnumbered == 0 {
		t.Fatal("the unnumbered-subheading fixture is stale")
	}
	above, aboveOK := document.SectionID(unnumbered - 1)
	at, atOK := document.SectionID(unnumbered)
	if !aboveOK || !atOK || above != at {
		t.Fatalf("an unnumbered subheading changed the clause from %q to %q", above, at)
	}

	// Nearly every line of the document belongs to some clause; a resolver that
	// silently answered "" would disable the anchor without failing anything.
	unresolved := 0
	for number := 1; number <= document.LineCount(); number++ {
		if _, ok := document.SectionID(number); !ok {
			unresolved++
		}
	}
	if unresolved > 32 {
		t.Fatalf("%d of %d lines resolve to no clause; the anchor would be mostly inert",
			unresolved, document.LineCount())
	}
}

// TestQuoteLinesRefusesTextThatSpansTwoTableRows pins the second hard boundary.
// A Markdown table row is a complete line by construction, so no honest excerpt
// spans two of them, while a stitched one is individually verbatim on both
// halves and silently imports the next row's constraint. Collapsing that newline
// to a space admitted exactly that.
func TestQuoteLinesRefusesTextThatSpansTwoTableRows(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	const (
		row      = "| <code>holder_host_id</code> | UUIDv7 | Proposed owner |"
		next     = "| <code>predecessor_lease_id</code> | UUIDv4 or null | Null only at epoch 1 |"
		stitched = row + " " + next
	)

	rowLines := document.QuoteLines(row)
	if len(rowLines) == 0 {
		t.Fatal("the table row is not quotable on its own; the fixture is stale")
	}
	if lines := document.QuoteLines(next); len(lines) == 0 {
		t.Fatal("the following table row is not quotable on its own; the fixture is stale")
	}
	if lines := document.QuoteLines(stitched); len(lines) != 0 {
		t.Fatalf("text spanning two table rows matched at %v; a quote may not stitch two rows", lines)
	}
	if document.Contains(stitched) {
		t.Fatal("Contains admitted text spanning two table rows")
	}
}

// TestQuoteLinesStillForgivesHardWrappingInsideOneParagraph is the other half of
// the pair. A boundary check that refused every newline would pass the test
// above while making the whole comparison useless, because the pinned document
// hard-wraps its prose.
func TestQuoteLinesStillForgivesHardWrappingInsideOneParagraph(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wrapped := 0
	for number := 1; number < document.LineCount(); number++ {
		above, _ := document.Line(number)
		below, _ := document.Line(number + 1)
		if strings.TrimSpace(above) == "" || strings.TrimSpace(below) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(above), "|") || strings.HasPrefix(strings.TrimSpace(below), "|") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(above), "#") || strings.HasPrefix(strings.TrimSpace(below), "#") {
			continue
		}
		stitched := strings.TrimSpace(above) + "\n" + strings.TrimSpace(below)
		if !document.Contains(stitched) {
			t.Fatalf("hard-wrapped prose at lines %d-%d is no longer quotable as one excerpt", number, number+1)
		}
		wrapped++
		if wrapped >= 200 {
			break
		}
	}
	if wrapped < 200 {
		t.Fatalf("checked only %d wrapped prose boundaries, want a broad sample", wrapped)
	}
}

// TestTableRowAtNamesWhatEachRowDeclares pins the index the enumeration's
// declaring-row anchor reads. A header or delimiter line declares nothing of its
// own, and must not be reported as a declaration a citation could anchor to.
func TestTableRowAtNamesWhatEachRowDeclares(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, test := range []struct {
		line       int
		ok         bool
		header     string
		identifier string
	}{
		{line: 1908, ok: true, header: "Field", identifier: "reason"},
		{line: 4788, ok: true, header: "Type", identifier: "GitHead"},
		{line: 4791, ok: true, header: "Type", identifier: "GitIndexEntry"},
		{line: 4785, ok: false},
		{line: 4786, ok: false},
		{line: 1, ok: false},
		{line: 0, ok: false},
		{line: document.LineCount() + 1, ok: false},
	} {
		row, ok := document.TableRowAt(test.line)
		if ok != test.ok {
			t.Fatalf("TableRowAt(%d) ok = %v, want %v (row %+v)", test.line, ok, test.ok, row)
		}
		if !test.ok {
			continue
		}
		if row.Header != test.header || row.Identifier != test.identifier || row.Line != test.line {
			t.Fatalf("TableRowAt(%d) = %+v, want header %q identifier %q", test.line, row, test.header, test.identifier)
		}
	}
}
