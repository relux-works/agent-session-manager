package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/traceability"
)

// measuredCoverageAnchor is the README subsection this test pins to the
// tracecheck report. It is matched exactly, so renaming the heading reddens
// here instead of silently exempting the subsection from measurement.
const measuredCoverageAnchor = "### Measured coverage of this repository"

// coverageNumberWords decodes the spelled-out figures the subsection
// publishes. The map is closed: a figure spelled a new way fails the lookup
// instead of passing against a guessed value.
var coverageNumberWords = map[string]int{
	"two":         2,
	"four":        4,
	"five":        5,
	"six":         6,
	"seven":       7,
	"eight":       8,
	"nine":        9,
	"forty-five":  45,
	"forty-nine":  49,
	"forty-three": 43,
	"fifty-two":   52,
	"twelve":      12,
	"sixty-eight": 68,
	"sixty-nine":  69,
}

// TestREADMEMeasuredCoverageMatchesTracecheckReport pins the README "Measured
// coverage of this repository" subsection to the tracecheck report measured
// on the current tree. The fenced `section coverage:` line must equal the
// tool's own output line byte-for-byte, and every headline prose figure must
// equal the Report field it restates.
//
// Per-section ratios inside the subsection (13.13 at 9/11 and the like) are
// deliberately not re-derived here: TestCatalogSectionBindingCoverageIsExact-
// AndDoesNotClaimUnimplementedScope already pins each binding's measured
// ratio through the production entry point, and duplicating that table would
// test the copy, not the prose. The historical "13 sections added by v0.6.0"
// sentence is likewise out of scope: it states history, not the measured
// report.
func TestREADMEMeasuredCoverageMatchesTracecheckReport(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Join("..", "..", "..", "..")

	var output bytes.Buffer
	if err := run([]string{"-root", repositoryRoot}, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("run() printed %d lines, want the 2-line tracecheck report", len(lines))
	}
	if !strings.HasPrefix(lines[1], "section coverage: ") {
		t.Fatalf("run() second line = %q, want the section coverage line", lines[1])
	}

	report, err := traceability.VerifyRepository(os.DirFS(repositoryRoot))
	if err != nil {
		t.Fatalf("VerifyRepository() error = %v", err)
	}

	subsection, start := measuredCoverageSubsection(t, repositoryRoot)
	assertFencedCoverageLine(t, subsection, start, lines[1])
	prose := strings.Join(subsection, " ")

	assertProseFigure(t, prose, start, `(\S+) section bindings discharge ([0-9]+) of the ([0-9]+) normative clauses`,
		[]int{report.SectionBindings, report.DischargedClauses, report.NormativeClauses})
	assertProseFigure(t, prose, start, `(Four) bindings are `+"`full`", []int{report.FullCoverage})
	assertProseFigure(t, prose, start, `(nine) are `+"`partial`", []int{report.PartialCoverage})
	assertProseFigure(t, prose, start, `(nine) are `+"`sliver`", []int{report.SliverCoverage})
	assertProseFigure(t, prose, start, `(four) are `+"`unmeasured`", []int{report.UnmeasuredCoverage})
	assertProseFigure(t, prose, start, `(forty-three) are `+"`unevidenced`", []int{report.UnevidencedCoverage})
	assertProseFigure(t, prose, start, `(Seven) sections are recorded unowned`, []int{report.UnownedSections})

	fullBindings, fullClauses := fullBindingTotals(t, repositoryRoot)
	if fullBindings != report.FullCoverage {
		t.Fatalf("registry declares %d full bindings, VerifyRepository measures %d", fullBindings, report.FullCoverage)
	}
	assertProseFigure(t, prose, start, `(Four) admitted bindings out of (sixty-nine) cover (twelve) clauses`,
		[]int{report.FullCoverage, report.SectionBindings, fullClauses})
}

// measuredCoverageSubsection returns the subsection lines under the anchor,
// stopping before the next heading. A missing anchor, a duplicated anchor,
// or a subsection that runs to end of file without a closing heading is a
// red: each shape would otherwise silently change what this test measures.
func measuredCoverageSubsection(t *testing.T, repositoryRoot string) ([]string, int) {
	t.Helper()

	readme, err := os.ReadFile(filepath.Join(repositoryRoot, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	lines := strings.Split(string(readme), "\n")
	anchor := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == measuredCoverageAnchor {
			if anchor >= 0 {
				t.Fatalf("README carries %q twice, at lines %d and %d", measuredCoverageAnchor, anchor+1, index+1)
			}
			anchor = index
		}
	}
	if anchor < 0 {
		t.Fatalf("README no longer carries %q", measuredCoverageAnchor)
	}
	end := -1
	for index := anchor + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			end = index
			break
		}
	}
	if end < 0 {
		t.Fatalf("README subsection %q runs to end of file without a closing heading", measuredCoverageAnchor)
	}
	return lines[anchor+1 : end], anchor + 2
}

// assertFencedCoverageLine requires the subsection to carry exactly one
// fenced block and its content to equal the tool's coverage line
// byte-for-byte: no rewording, no reordered fields, no rounding.
func assertFencedCoverageLine(t *testing.T, subsection []string, start int, want string) {
	t.Helper()

	var fences []int
	for offset, line := range subsection {
		trimmed := strings.TrimSpace(line)
		if trimmed == "```text" || trimmed == "```" {
			fences = append(fences, offset)
		}
	}
	if len(fences) != 2 || strings.TrimSpace(subsection[fences[0]]) != "```text" {
		t.Fatalf("README subsection at line %d carries %d fence lines, want exactly one ```text block", start, len(fences))
	}
	got := strings.Join(subsection[fences[0]+1:fences[1]], "\n")
	if got != want {
		t.Fatalf("README fenced coverage line = %q, tracecheck printed %q", got, want)
	}
}

// assertProseFigure requires the pattern to match the unwrapped subsection
// exactly once and each capture to equal the measured figure. A capture that
// is all digits compares numerically; any other capture decodes through the
// closed number-word map.
func assertProseFigure(t *testing.T, prose string, start int, pattern string, want []int) {
	t.Helper()

	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(prose, -1)
	if len(matches) != 1 {
		t.Fatalf("README subsection at line %d matches %q %d times, want exactly 1", start, pattern, len(matches))
	}
	if len(matches[0])-1 != len(want) {
		t.Fatalf("README pattern %q captures %d figures, want %d", pattern, len(matches[0])-1, len(want))
	}
	for index, raw := range matches[0][1:] {
		var published int
		if value, err := strconv.Atoi(raw); err == nil {
			published = value
		} else if value, ok := coverageNumberWords[strings.ToLower(raw)]; ok {
			published = value
		} else {
			t.Fatalf("README subsection at line %d spells a figure %q the closed map does not decode", start, raw)
		}
		if published != want[index] {
			t.Fatalf("README subsection at line %d publishes %d for %q; measured %d", start, published, pattern, want[index])
		}
	}
}

// fullBindingTotals re-derives the admitted-sentence figures from the
// reviewed registry: how many section bindings declare full coverage and how
// many discharged clauses they enumerate. The gate has already verified each
// clause against the pinned document (VerifyRepository is green in this
// test), so the sum is measured evidence, not a restated claim.
func fullBindingTotals(t *testing.T, repositoryRoot string) (int, int) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repositoryRoot, "internal", "traceability", "ownership.v0.7.0.json"))
	if err != nil {
		t.Fatalf("read adopted ownership registry: %v", err)
	}
	var registry struct {
		Ownership []struct {
			Kind     string `json:"kind"`
			Coverage string `json:"coverage"`
			Clauses  []struct {
				ID string `json:"id"`
			} `json:"clauses"`
		} `json:"ownership"`
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatalf("decode adopted ownership registry: %v", err)
	}
	bindings, clauses := 0, 0
	for _, group := range registry.Ownership {
		if group.Kind != "section_binding" || group.Coverage != "full" {
			continue
		}
		bindings++
		clauses += len(group.Clauses)
	}
	return bindings, clauses
}
