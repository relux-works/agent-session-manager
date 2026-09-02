// Package specdoc exposes the exact pinned upstream AX specification document
// to repository fidelity gates.
//
// The upstream specification remains normative. The embedded copy is a
// verification input only: it is accepted solely when its SHA-256 equals the
// document digest already pinned in internal/specpin, so a substituted,
// truncated, edited, or partially read document is refused instead of being
// silently compared against. Nothing here advertises runtime capabilities or
// mutates durable state.
//
// The package exists so that enumeration artifacts can be compared against the
// specification text itself rather than against the implementation those
// artifacts are supposed to constrain. Only test binaries import it, so the
// embedded document never reaches a shipped command.
package specdoc

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

//go:embed SPEC.md
var embedded []byte

// ErrDocumentMismatch reports an absent, empty, or non-pinned document. A
// failed or partial read is reported as a mismatch, never as an absence that
// some caller could treat as satisfied.
var ErrDocumentMismatch = errors.New("pinned specification document mismatch")

// Document is a verified pinned specification together with the single
// whitespace-normalized projection used for excerpt comparison, an index from
// normalized offsets back to 1-based SPEC.md line numbers, and an index of the
// document's Markdown tables.
//
// The projection collapses each run of ASCII whitespace to one space, except at
// a hard boundary, where it collapses to BlockSeparator instead. A normalized
// excerpt can never contain BlockSeparator, so a quote cannot be satisfied by
// stitching text across such a boundary. Two kinds of boundary are hard:
//
//   - a blank line, which separates a paragraph from the paragraph after it and
//     a heading from its body;
//   - the newline between two adjacent Markdown table rows, because a table row
//     is a complete line by construction, so no honest excerpt spans two of
//     them, while a stitched one silently imports the next row's constraint.
//
// Hard line wrapping and table indentation inside one block are still forgiven,
// as is the newline between two adjacent list items or between two adjacent
// lines of one paragraph.
type Document struct {
	raw            []byte
	lines          []string
	normalized     string
	lineOf         []int
	sectionOfLine  []string
	tableRowOfLine []TableRow
}

// BlockSeparator marks a hard boundary in the normalized projection. It is a
// newline, which Normalize never emits, so it is unmatchable by any excerpt.
const BlockSeparator = '\n'

// TableRow describes one body row of a Markdown table in the pinned document:
// the first-column header of the table it belongs to, and the text of its own
// first cell. Together they say what the row declares — "Field"/"reason" is the
// declaration of a member, "Type"/"GitIndex" the declaration of a type, and
// "Tag"/"kind = git" the declaration of a variant.
type TableRow struct {
	// Header is the trimmed text of the table's first column header.
	Header string
	// FirstCell is the trimmed text of this row's first cell.
	FirstCell string
	// Identifier is FirstCell with a single enclosing <code> element removed,
	// which is how the pinned document writes every declaration.
	Identifier string
	// Line is the 1-based SPEC.md line of the row.
	Line int
}

// Load returns the embedded pinned document after verifying its digest.
func Load() (*Document, error) {
	return Parse(embedded)
}

// Bytes returns an isolated copy of the exact embedded document bytes.
func Bytes() []byte {
	return bytes.Clone(embedded)
}

// Parse accepts only a byte-exact copy of the pinned SPEC.md identified by
// specpin.DocumentSHA256.
func Parse(candidate []byte) (*Document, error) {
	if len(candidate) == 0 {
		return nil, fmt.Errorf("%w: document is empty", ErrDocumentMismatch)
	}
	digest := sha256.Sum256(candidate)
	if got := hex.EncodeToString(digest[:]); got != specpin.DocumentSHA256 {
		return nil, fmt.Errorf("%w: SHA-256 is %s, want %s", ErrDocumentMismatch, got, specpin.DocumentSHA256)
	}

	document := &Document{raw: bytes.Clone(candidate)}
	document.lines = strings.Split(strings.ReplaceAll(string(document.raw), "\r\n", "\n"), "\n")

	hardBoundary := indexHardBoundaries(document.lines)

	var normalized strings.Builder
	normalized.Grow(len(candidate))
	lineOf := make([]int, 0, len(candidate))
	line := 1
	runStartLine := 1
	pendingSeparator := false
	wroteAny := false
	for index := 0; index < len(document.raw); index++ {
		character := document.raw[index]
		if isNormalizedWhitespace(character) {
			if wroteAny && !pendingSeparator {
				runStartLine = line
				pendingSeparator = true
			}
			if character == '\n' {
				line++
			}
			continue
		}
		if pendingSeparator {
			separator := byte(' ')
			if crossesHardBoundary(hardBoundary, runStartLine, line) {
				separator = BlockSeparator
			}
			normalized.WriteByte(separator)
			lineOf = append(lineOf, line)
			pendingSeparator = false
		}
		normalized.WriteByte(character)
		lineOf = append(lineOf, line)
		wroteAny = true
	}
	document.normalized = normalized.String()
	document.lineOf = lineOf
	document.sectionOfLine = indexSections(document.lines)
	document.tableRowOfLine = indexTableRows(document.lines)
	return document, nil
}

// indexHardBoundaries marks, for each 1-based line n, whether the newline
// between line n and line n+1 is a hard boundary that no excerpt may cross.
func indexHardBoundaries(lines []string) []bool {
	hard := make([]bool, len(lines)+1)
	for number := 1; number < len(lines); number++ {
		above, below := lines[number-1], lines[number]
		if strings.TrimSpace(above) == "" || strings.TrimSpace(below) == "" {
			hard[number] = true
			continue
		}
		if isTableLine(above) && isTableLine(below) {
			hard[number] = true
		}
	}
	return hard
}

// crossesHardBoundary reports whether a whitespace run that started on line
// from and ended on line to crossed any hard boundary.
func crossesHardBoundary(hard []bool, from, to int) bool {
	for number := from; number < to && number < len(hard); number++ {
		if hard[number] {
			return true
		}
	}
	return false
}

func isTableLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// tableSeparator matches the delimiter row that turns the line above it into a
// Markdown table header.
var tableSeparator = regexp.MustCompile(`^\|[\s:|-]*-{3,}[\s:|-]*\|?\s*$`)

// indexTableRows maps every 1-based line that is a body row of a Markdown table
// to what that row declares. Header and delimiter lines map to nothing: they
// declare no member, type, or variant of their own.
func indexTableRows(lines []string) []TableRow {
	rows := make([]TableRow, len(lines)+1)
	for index := 0; index+1 < len(lines); index++ {
		if !isTableLine(lines[index]) || !tableSeparator.MatchString(strings.TrimSpace(lines[index+1])) {
			continue
		}
		header := firstTableCell(lines[index])
		for body := index + 2; body < len(lines) && isTableLine(lines[body]); body++ {
			cell := firstTableCell(lines[body])
			rows[body+1] = TableRow{
				Header:     header,
				FirstCell:  cell,
				Identifier: stripCodeElement(cell),
				Line:       body + 1,
			}
		}
	}
	return rows
}

// firstTableCell returns the trimmed text of a table line's first cell.
func firstTableCell(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	if index := strings.IndexByte(trimmed, '|'); index >= 0 {
		trimmed = trimmed[:index]
	}
	return strings.TrimSpace(trimmed)
}

var codeElement = regexp.MustCompile(`^<code>(.*)</code>$`)

// stripCodeElement removes a single enclosing <code> element, which is how the
// pinned document writes every declaration cell. Text that is not exactly one
// such element is returned unchanged rather than partially rewritten.
func stripCodeElement(cell string) string {
	if match := codeElement.FindStringSubmatch(cell); match != nil {
		return match[1]
	}
	return cell
}

// TableRowAt returns what the Markdown table body row at a 1-based SPEC.md line
// declares. It reports false for every line that is not such a row, including
// table headers and delimiters.
func (document *Document) TableRowAt(number int) (TableRow, bool) {
	if number < 1 || number >= len(document.tableRowOfLine) {
		return TableRow{}, false
	}
	row := document.tableRowOfLine[number]
	if row.Line == 0 {
		return TableRow{}, false
	}
	return row, true
}

// headingSectionID matches the numbered or appendix identifier of an ATX
// heading: "## 10.4 Transfer Manifest" -> "10.4", "#### 13.14.5 Events" ->
// "13.14.5", "## Appendix A. Normative traceability" -> "A",
// "### A.3 Task acceptance traceability" -> "A.3".
var (
	atxHeading      = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	numberedSection = regexp.MustCompile(`^([0-9]+(?:\.[0-9A-Z]+)*|[A-Z](?:\.[0-9]+)+)\.?\s`)
	appendixSection = regexp.MustCompile(`^Appendix\s+([A-Z])\.`)
)

// indexSections maps every 1-based line to the identifier of the nearest
// enclosing numbered clause. An unnumbered subheading — "#### Managed Replica
// Marker document" — does not open a clause of its own; its lines stay
// attributed to the numbered section above it, which is the clause a citation
// belongs to.
func indexSections(lines []string) []string {
	sections := make([]string, len(lines)+1)
	current := ""
	for index, line := range lines {
		if match := atxHeading.FindStringSubmatch(line); match != nil {
			if id, ok := headingSection(match[2]); ok {
				current = id
			}
		}
		sections[index+1] = current
	}
	return sections
}

func headingSection(text string) (string, bool) {
	if match := appendixSection.FindStringSubmatch(text); match != nil {
		return match[1], true
	}
	if match := numberedSection.FindStringSubmatch(text); match != nil {
		return match[1], true
	}
	return "", false
}

// SectionID returns the identifier of the numbered clause containing a 1-based
// SPEC.md line, so a citation can be held to the clause of the shape it
// documents rather than only to the characters at the line.
func (document *Document) SectionID(number int) (string, bool) {
	if number < 1 || number >= len(document.sectionOfLine) {
		return "", false
	}
	id := document.sectionOfLine[number]
	if id == "" {
		return "", false
	}
	return id, true
}

// Normalize applies the single documented normalization rule to excerpt text:
// every run of ASCII whitespace collapses to one space and leading/trailing
// whitespace is removed. Nothing else is rewritten. Letter case, punctuation,
// digits, and inline <code> markup are compared exactly, so the rule forgives
// the specification's hard line wrapping and table indentation without letting
// an excerpt match unrelated text.
//
// Normalize never emits BlockSeparator. The document projection does, at every
// hard boundary — a blank line, and the newline between two adjacent table rows
// — so a normalized excerpt cannot span one.
func Normalize(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	pendingSeparator := false
	wroteAny := false
	for index := 0; index < len(text); index++ {
		character := text[index]
		if isNormalizedWhitespace(character) {
			if wroteAny {
				pendingSeparator = true
			}
			continue
		}
		if pendingSeparator {
			builder.WriteByte(' ')
			pendingSeparator = false
		}
		builder.WriteByte(character)
		wroteAny = true
	}
	return builder.String()
}

func isNormalizedWhitespace(character byte) bool {
	switch character {
	case ' ', '\t', '\r', '\n', '\v', '\f':
		return true
	default:
		return false
	}
}

// LineCount reports the number of lines in the pinned document.
func (document *Document) LineCount() int {
	return len(document.lines)
}

// Line returns the raw text of a 1-based SPEC.md line.
func (document *Document) Line(number int) (string, bool) {
	if number < 1 || number > len(document.lines) {
		return "", false
	}
	return document.lines[number-1], true
}

// QuoteLines returns every 1-based SPEC.md line on which the normalized form
// of text begins. An empty excerpt matches nothing: an empty quote would
// otherwise be satisfied by any document. A match never spans a blank line,
// because the projection separates blocks with the unmatchable BlockSeparator.
func (document *Document) QuoteLines(text string) []int {
	needle := Normalize(text)
	if needle == "" {
		return nil
	}
	var lines []int
	for offset := 0; ; {
		index := strings.Index(document.normalized[offset:], needle)
		if index < 0 {
			return lines
		}
		start := offset + index
		lines = append(lines, document.lineOf[start])
		offset = start + 1
	}
}

// Contains reports whether the normalized form of text occurs in the pinned
// document at all.
func (document *Document) Contains(text string) bool {
	return len(document.QuoteLines(text)) != 0
}
