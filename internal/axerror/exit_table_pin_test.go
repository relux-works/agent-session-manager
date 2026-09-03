package axerror

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// pinnedExitStatusRows measures the Section 15.2 exit-code table from the
// digest-verified pinned document. It is a measurement rather than a written
// number on purpose: this repository stated the row count as "nineteen" in four
// places against a table that has eighteen body rows, and the miscount spread
// from one comment into README prose and into a landed traceability comment
// because nothing compared the word to the table.
func pinnedExitStatusRows(t *testing.T) []int {
	t.Helper()
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned document: %v", err)
	}
	var rows []int
	for line := 1; line <= document.LineCount(); line++ {
		section, ok := document.SectionID(line)
		if !ok || section != "15.2" {
			continue
		}
		row, ok := document.TableRowAt(line)
		if !ok || row.Header != "Exit" {
			continue
		}
		status, err := strconv.Atoi(row.Identifier)
		if err != nil {
			t.Fatalf("Section 15.2 row at line %d has non-numeric first cell %q", line, row.Identifier)
		}
		rows = append(rows, status)
	}
	if len(rows) == 0 {
		t.Fatalf("no Section 15.2 exit-code rows were measured")
	}
	sort.Ints(rows)
	return rows
}

// TestExitStatusRegistryMatchesThePinnedTableRowForRow measures the pinned
// Section 15.2 table and compares it to the production registry, so neither the
// count nor any individual status can drift from the document.
func TestExitStatusRegistryMatchesThePinnedTableRowForRow(t *testing.T) {
	rows := pinnedExitStatusRows(t)
	if len(rows) != len(exitMeanings) {
		t.Fatalf("pinned Section 15.2 table has %d rows and exitMeanings carries %d", len(rows), len(exitMeanings))
	}
	for _, status := range rows {
		if _, known := ExitStatusMeaning(status); !known {
			t.Fatalf("pinned exit status %d is not in the production registry", status)
		}
	}
	registered := make([]int, 0, len(exitMeanings))
	for status := range exitMeanings {
		registered = append(registered, status)
	}
	sort.Ints(registered)
	if fmt.Sprint(registered) != fmt.Sprint(rows) {
		t.Fatalf("production registry is %v, the pinned table is %v", registered, rows)
	}
	// The reviewed set, written out once so that a table row deleted from both
	// the document and the registry cannot pass unnoticed.
	reviewed := []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 130}
	if fmt.Sprint(rows) != fmt.Sprint(reviewed) {
		t.Fatalf("pinned Section 15.2 statuses are %v, the reviewed set is %v", rows, reviewed)
	}
	if len(reviewed) != 18 {
		t.Fatalf("the reviewed Section 15.2 set has %d rows", len(reviewed))
	}
}

// TestExactlyOneRegisteredExitStatusIsSuccess pins the one row that is not a
// failure class, measured from the same table.
func TestExactlyOneRegisteredExitStatusIsSuccess(t *testing.T) {
	rows := pinnedExitStatusRows(t)
	failures := 0
	for _, status := range rows {
		if IsFailureExitStatus(status) {
			failures++
		}
	}
	if failures != len(rows)-1 {
		t.Fatalf("%d of %d pinned statuses are failure classes, want all but the success row", failures, len(rows))
	}
	if IsFailureExitStatus(successExit) {
		t.Fatalf("the success status is reported as a failure class")
	}
}

// fanInHeader is the README table header this test measures. It is matched
// exactly rather than by keyword, so renaming a column reddens here instead of
// silently exempting the table from measurement.
const fanInHeader = "| Structured Error version | Failure statuses | Registered codes " +
	"| Statuses carrying more than one code | Largest class |"

// fanInRow is one published row, parsed from the README rather than restated.
type fanInRow struct {
	line       int
	version    Version
	statuses   int
	codes      int
	ambiguous  int
	largest    int
	largestFor int
}

// measuredFanIn is the same row derived from the production projection.
func measuredFanIn(t *testing.T, version Version) fanInRow {
	t.Helper()
	groups, err := CodesByExitStatus(version)
	if err != nil {
		t.Fatalf("CodesByExitStatus(%s): %v", version, err)
	}
	row := fanInRow{version: version, statuses: len(groups)}
	ties := 0
	for status, group := range groups {
		row.codes += len(group)
		if len(group) > 1 {
			row.ambiguous++
		}
		switch {
		case len(group) > row.largest:
			row.largest, row.largestFor, ties = len(group), status, 1
		case len(group) == row.largest:
			ties++
		}
	}
	// "The largest class" names one status, so the published row is only well
	// defined while exactly one status carries the maximum. A tie is reported as
	// unknown rather than resolved by picking the lower status.
	if ties != 1 {
		t.Fatalf("%s has %d statuses tied at %d codes, so no single largest class exists to publish",
			version, ties, row.largest)
	}
	return row
}

// parsePublishedFanIn reads the README table into rows. It refuses a table it
// cannot parse rather than skipping the unparsed part: a row that silently fell
// out of the comparison is exactly how the two wrong rows below survived.
func parsePublishedFanIn(t *testing.T) []fanInRow {
	t.Helper()
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	lines := strings.Split(string(readme), "\n")
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == fanInHeader {
			if start >= 0 {
				t.Fatalf("README carries the fan-in table header twice, at lines %d and %d", start+1, index+1)
			}
			start = index
		}
	}
	if start < 0 {
		t.Fatalf("README no longer carries the fan-in table header %q", fanInHeader)
	}
	var rows []fanInRow
	for index := start + 2; index < len(lines); index++ {
		text := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(text, "|") {
			break
		}
		cells := strings.Split(strings.Trim(text, "|"), "|")
		if len(cells) != 5 {
			t.Fatalf("README fan-in row at line %d has %d cells, want 5: %q", index+1, len(cells), text)
		}
		for cell := range cells {
			cells[cell] = strings.TrimSpace(cells[cell])
		}
		row := fanInRow{line: index + 1, version: Version(strings.Trim(cells[0], "`"))}
		numbers := []*int{&row.statuses, &row.codes, &row.ambiguous}
		for cell, target := range numbers {
			value, err := strconv.Atoi(cells[cell+1])
			if err != nil {
				t.Fatalf("README fan-in row at line %d has non-numeric cell %q", index+1, cells[cell+1])
			}
			*target = value
		}
		if _, err := fmt.Sscanf(cells[4], "%d codes at exit %d", &row.largest, &row.largestFor); err != nil {
			t.Fatalf("README fan-in row at line %d has largest class %q, want \"N codes at exit S\": %v",
				index+1, cells[4], err)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		t.Fatal("README fan-in table has no body rows")
	}
	return rows
}

// TestREADMEFanInTableIsDerivedFromTheMeasuredProjection makes the published
// table a measurement instead of prose beside one.
//
// This is the third time a number in this repository drifted from the thing it
// describes: "nineteen" §15.2 rows against a table with eighteen, an inflated
// 10/394 clause ratio, and now two of the four rows of this very table. The
// 1.2.0 row published "12 codes at exit 6" against a measured 14 at exit 16 -
// wrong count AND wrong status - and 1.3.0 published 15 at exit 6 against a
// measured 17. Both survived because the largest class was asserted for 1.0.0
// and 1.1.0 only, so the other two rows were never compared to anything.
//
// The fix is not to assert the corrected numbers somewhere else. It is to leave
// no row unmeasured: every registered version must appear exactly once, every
// published row must appear in the registry, and every cell of every row is
// re-derived from CodesByExitStatus here.
func TestREADMEFanInTableIsDerivedFromTheMeasuredProjection(t *testing.T) {
	published := parsePublishedFanIn(t)

	seen := make(map[Version]int, len(published))
	for _, row := range published {
		if previous, duplicate := seen[row.version]; duplicate {
			t.Fatalf("README publishes %s twice, at lines %d and %d", row.version, previous, row.line)
		}
		seen[row.version] = row.line
		if !isRegisteredVersion(row.version) {
			t.Fatalf("README line %d publishes %s, which the registry does not carry", row.line, row.version)
		}
		measured := measuredFanIn(t, row.version)
		measured.line = row.line
		if row != measured {
			t.Fatalf("README line %d publishes %s as %d statuses / %d codes / %d ambiguous / "+
				"%d codes at exit %d; measured %d / %d / %d / %d codes at exit %d",
				row.line, row.version, row.statuses, row.codes, row.ambiguous, row.largest, row.largestFor,
				measured.statuses, measured.codes, measured.ambiguous, measured.largest, measured.largestFor)
		}
	}
	for _, version := range Versions() {
		if _, present := seen[version]; !present {
			t.Fatalf("the registry carries %s and the README table has no row for it, so its fan-in "+
				"is published nowhere and cannot drift into being wrong unnoticed", version)
		}
	}
}

// TestLogbookFanInFiguresAreDerivedFromTheMeasuredProjection covers the second
// place the wrong 1.3.0 figure was copied to. Entry 1003 states the 1.0.0 and
// 1.3.0 rows in prose, and prose beside a table that measures itself is exactly
// what drifts, so the sentences are located and re-derived rather than trusted.
func TestLogbookFanInFiguresAreDerivedFromTheMeasuredProjection(t *testing.T) {
	logbook, err := os.ReadFile(filepath.Join("..", "..", "LOGBOOK.md"))
	if err != nil {
		t.Fatalf("read LOGBOOK.md: %v", err)
	}
	text := string(logbook)

	for _, claim := range []struct {
		version  Version
		sentence func(fanInRow) string
	}{
		{
			version: Version100,
			sentence: func(row fanInRow) string {
				return fmt.Sprintf("assigns %d codes to %d failure statuses, %d of which carry more than one code, "+
					"the largest being %d codes at exit %d", row.codes, row.statuses, row.ambiguous,
					row.largest, row.largestFor)
			},
		},
		{
			version: Version130,
			sentence: func(row fanInRow) string {
				return fmt.Sprintf("reaches %d codes with %d codes at exit %d",
					row.codes, row.largest, row.largestFor)
			},
		},
	} {
		want := claim.sentence(measuredFanIn(t, claim.version))
		if !strings.Contains(text, want) {
			t.Fatalf("LOGBOOK.md no longer states the measured %s fan-in %q", claim.version, want)
		}
	}
}
