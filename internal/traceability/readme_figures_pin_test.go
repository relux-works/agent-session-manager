package traceability

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// ownershipParagraphAnchor is the README heading whose first paragraph this
// test measures. It is matched exactly rather than by keyword, so renaming the
// heading reddens here instead of silently exempting the paragraph from
// measurement.
const ownershipParagraphAnchor = "## Specification-to-Code Ownership Gate"

// publishedFigure is one number the ownership paragraph states, located by the
// noun phrase that follows it and bound to the Report field it must equal. The
// phrase is the identity: a figure is proven by naming what it counts, not by
// its position in the sentence, so re-ordering the clause does not silently
// re-point a pin at a different number.
type publishedFigure struct {
	phrase  string
	measure func(Report) int
}

// publishedOwnershipFigures enumerates every figure the paragraph states. The
// unpinned-figure guard below turns a deletion from this list into a red rather
// than into a quietly smaller comparison, so the list is measured, not trusted.
var publishedOwnershipFigures = []publishedFigure{
	{"current contract rows", func(report Report) int { return report.Contracts }},
	{"pinned or catalog-referenced normative section keys", func(report Report) int { return report.NormativeSections }},
	{"executable acceptance cases", func(report Report) int { return report.AcceptanceCases }},
	{"exact section bindings", func(report Report) int { return report.SectionBindings }},
	{"disclosed unowned sections", func(report Report) int { return report.UnownedSections }},
	{"exact fixture identities", func(report Report) int { return report.Fixtures }},
	{"-contract subset", func(report Report) int { return report.CompatibilityContracts }},
}

// ownershipParagraph returns the paragraph under the anchor, unwrapped onto one
// line, together with the README line it starts at. It refuses a target it
// cannot locate rather than skipping the comparison: a paragraph that silently
// fell out of the measurement is exactly how the acceptance-case figure below
// drifted from 73 to 74 without anything reddening.
func ownershipParagraph(t *testing.T) (string, int) {
	t.Helper()

	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	lines := strings.Split(string(readme), "\n")
	anchor := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == ownershipParagraphAnchor {
			if anchor >= 0 {
				t.Fatalf("README carries the heading %q twice, at lines %d and %d; the paragraph this test measures is ambiguous",
					ownershipParagraphAnchor, anchor+1, index+1)
			}
			anchor = index
		}
	}
	if anchor < 0 {
		t.Fatalf("README no longer carries the heading %q", ownershipParagraphAnchor)
	}
	start := anchor + 1
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	var body []string
	for index := start; index < len(lines); index++ {
		text := strings.TrimSpace(lines[index])
		if text == "" {
			break
		}
		body = append(body, text)
	}
	if len(body) == 0 {
		t.Fatalf("README carries no paragraph under %q", ownershipParagraphAnchor)
	}
	return strings.Join(body, " "), start + 1
}

// isVersionDigits reports whether the digit run beginning at offset belongs to a
// dotted version or path token such as v0.5.0 or ownership.v0.5.0.json, rather
// than being a figure the paragraph publishes. The rule is deliberately narrow:
// only a run introduced by "v" or preceded by a dot that itself follows a digit
// is exempt, so a published figure can never be excused as version text.
func isVersionDigits(paragraph string, offset int) bool {
	if offset == 0 {
		return false
	}
	switch paragraph[offset-1] {
	case 'v', 'V':
		return true
	case '.':
		return offset >= 2 && paragraph[offset-2] >= '0' && paragraph[offset-2] <= '9'
	}
	return false
}

// TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport makes the published
// ownership paragraph a measurement instead of prose beside one.
//
// This is the third time in this artifact that a figure published next to a
// measurement drifted from it: LOGBOOK 1015 records the "nineteen §15.2 rows"
// claim against an eighteen-row table, LOGBOOK 1157 records two of the four
// published fan-in rows being wrong, and this paragraph published 73 executable
// acceptance cases while the registry held 74 and tracecheck printed 74. The
// tool for the class already existed - parsePublishedFanIn in
// internal/axerror/exit_table_pin_test.go - and had simply never been pointed at
// the neighbouring sentence.
//
// The fix is not to assert the corrected number somewhere else. It is to leave
// no figure of the paragraph unmeasured: each is located by the phrase naming
// what it counts, re-derived from VerifyRepository, and any number the paragraph
// states that no pinned phrase consumed is reported as unmeasured rather than
// ignored.
func TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport(t *testing.T) {
	t.Parallel()

	report, err := VerifyRepository(repositorySnapshot(t))
	if err != nil {
		t.Fatalf("VerifyRepository() error = %v", err)
	}
	paragraph, line := ownershipParagraph(t)

	consumed := make([]bool, len(paragraph))
	for _, figure := range publishedOwnershipFigures {
		pattern := regexp.MustCompile(`([0-9]+) ?` + regexp.QuoteMeta(figure.phrase))
		matches := pattern.FindAllStringSubmatchIndex(paragraph, -1)
		if len(matches) != 1 {
			t.Fatalf("README ownership paragraph at line %d states a figure for %q %d times, want exactly 1",
				line, figure.phrase, len(matches))
		}
		digits := paragraph[matches[0][2]:matches[0][3]]
		published, convErr := strconv.Atoi(digits)
		if convErr != nil {
			t.Fatalf("README ownership paragraph at line %d states a non-numeric figure %q for %q",
				line, digits, figure.phrase)
		}
		if measured := figure.measure(report); published != measured {
			t.Fatalf("README ownership paragraph at line %d publishes %d %s; VerifyRepository measures %d",
				line, published, strings.TrimPrefix(figure.phrase, "-"), measured)
		}
		for offset := matches[0][2]; offset < matches[0][3]; offset++ {
			consumed[offset] = true
		}
	}

	// Every remaining digit run must be version or path text. A figure nobody
	// pinned is reported here, so adding an unmeasured number to the paragraph -
	// or deleting a row from publishedOwnershipFigures - is a red.
	for _, run := range regexp.MustCompile(`[0-9]+`).FindAllStringIndex(paragraph, -1) {
		if consumed[run[0]] || isVersionDigits(paragraph, run[0]) {
			continue
		}
		t.Fatalf("README ownership paragraph at line %d states the figure %q, which no pinned phrase measures; "+
			"give it a row in publishedOwnershipFigures or remove it from the paragraph",
			line, paragraph[run[0]:run[1]])
	}
}
