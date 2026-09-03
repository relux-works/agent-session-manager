package cliresult

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// readerSourceFile is the one production file this leaf added in full, and the
// file the inventory below is derived from exhaustively.
const readerSourceFile = "client.go"

// guardInventoryAnchor is the README heading whose two tables this test pins. It
// is matched exactly, so renaming the heading reddens here instead of silently
// exempting the inventory from measurement.
const guardInventoryAnchor = "### Refusal guard inventory"

// evidenceClasses is the closed vocabulary a row's domain-evidence cell must
// open with. A cell that opened with anything else would be prose again.
var evidenceClasses = []string{"measured", "stated bound", "declared subsumed"}

// refusalSite is one place the reader refuses, derived from the source rather
// than named by hand: an fmt.Errorf call wrapping one of this package's
// sentinels, identified by the function it sits in and the format literal it
// carries.
type refusalSite struct {
	function string
	sentinel string
	format   string
}

// derivedRefusalSites walks the reader source and returns every refusal site in
// it. This is the forced traversal: it does not consult the inventory, so a
// guard added to the file appears here whether or not anybody remembered it.
func derivedRefusalSites(t *testing.T) []refusalSite {
	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, readerSourceFile, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", readerSourceFile, err)
	}

	var sites []refusalSite
	for _, declaration := range parsed.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		if !isFunction || function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}
			selector, isSelector := call.Fun.(*ast.SelectorExpr)
			if !isSelector || selector.Sel.Name != "Errorf" {
				return true
			}
			if pkg, isIdent := selector.X.(*ast.Ident); !isIdent || pkg.Name != "fmt" {
				return true
			}
			format, sentinel := "", ""
			for index, argument := range call.Args {
				if index == 0 {
					format = concatenatedStringLiteral(t, argument)
					continue
				}
				if identifier, isIdent := argument.(*ast.Ident); isIdent &&
					strings.HasPrefix(identifier.Name, "Err") && sentinel == "" {
					sentinel = identifier.Name
				}
			}
			if sentinel == "" {
				t.Fatalf("%s: fmt.Errorf in %s at %s wraps no Err sentinel; the inventory cannot "+
					"classify a refusal it cannot name", readerSourceFile, function.Name.Name,
					fileSet.Position(call.Pos()))
			}
			if !strings.HasPrefix(format, "%w") {
				t.Fatalf("%s: fmt.Errorf in %s at %s does not wrap its sentinel first: %q",
					readerSourceFile, function.Name.Name, fileSet.Position(call.Pos()), format)
			}
			sites = append(sites, refusalSite{function: function.Name.Name, sentinel: sentinel, format: format})
			return true
		})
	}
	if len(sites) == 0 {
		t.Fatalf("derived no refusal site from %s, so this pin measures nothing", readerSourceFile)
	}
	return sites
}

// concatenatedStringLiteral flattens `"a" + "b"` into its text. A format built
// from concatenated literals is common here and must not silently derive as an
// empty marker.
func concatenatedStringLiteral(t *testing.T, expression ast.Expr) string {
	t.Helper()
	switch typed := expression.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return ""
		}
		unquoted, err := strconv.Unquote(typed.Value)
		if err != nil {
			t.Fatalf("unquote %s: %v", typed.Value, err)
		}
		return unquoted
	case *ast.BinaryExpr:
		if typed.Op != token.ADD {
			return ""
		}
		return concatenatedStringLiteral(t, typed.X) + concatenatedStringLiteral(t, typed.Y)
	default:
		return ""
	}
}

// inventoryRow is one row of the README's derived table.
type inventoryRow struct {
	number   string
	guard    string
	sentinel string
	marker   string
	evidence string
	tests    []string
	line     int
}

// siblingRow is one row of the README's enumerated table of guards this leaf
// added or reordered outside the reader source.
type siblingRow struct {
	guard    string
	file     string
	function string
	tests    []string
	line     int
}

// readmeInventory locates the two tables under the anchor and parses them. It
// refuses a heading it cannot locate exactly once rather than skipping the
// comparison, which is how a published figure drifts while a suite stays green.
func readmeInventory(t *testing.T) ([]inventoryRow, []siblingRow) {
	t.Helper()

	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	lines := strings.Split(string(readme), "\n")
	anchor := -1
	for index, line := range lines {
		if strings.TrimSpace(line) != guardInventoryAnchor {
			continue
		}
		if anchor >= 0 {
			t.Fatalf("README carries the heading %q twice, at lines %d and %d", guardInventoryAnchor, anchor+1, index+1)
		}
		anchor = index
	}
	if anchor < 0 {
		t.Fatalf("README carries no heading %q, so the inventory this test pins is not published", guardInventoryAnchor)
	}

	var derived []inventoryRow
	var siblings []siblingRow
	for index := anchor + 1; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if strings.HasPrefix(line, "## ") {
			break
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := tableCells(line)
		switch {
		case len(cells) == 6 && cells[0] != "#" && !strings.HasPrefix(cells[0], "---"):
			derived = append(derived, inventoryRow{
				number:   cells[0],
				guard:    cells[1],
				sentinel: strings.Trim(cells[2], "`"),
				marker:   strings.Trim(cells[3], "`"),
				evidence: cells[4],
				tests:    backtickedNames(cells[5]),
				line:     index + 1,
			})
		case len(cells) == 4 && cells[0] != "Guard" && !strings.HasPrefix(cells[0], "---"):
			siblings = append(siblings, siblingRow{
				guard:    cells[0],
				file:     strings.Trim(cells[1], "`"),
				function: strings.Trim(cells[2], "`"),
				tests:    backtickedNames(cells[3]),
				line:     index + 1,
			})
		}
	}
	if len(derived) == 0 || len(siblings) == 0 {
		t.Fatalf("parsed %d derived rows and %d sibling rows out of the inventory section",
			len(derived), len(siblings))
	}
	return derived, siblings
}

func tableCells(line string) []string {
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func backtickedNames(cell string) []string {
	var names []string
	for _, part := range strings.Split(cell, "`") {
		if strings.HasPrefix(part, "Test") {
			names = append(names, part)
		}
	}
	return names
}

// declaredTests indexes every test declaration in the repository by name,
// mapping each to the files that declare it. A row naming a test that does not
// exist, or that was renamed, is then a red rather than a row pointing at
// nothing.
func declaredTests(t *testing.T) map[string][]string {
	t.Helper()

	declarations := make(map[string][]string)
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		for _, declaration := range parsed.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || !strings.HasPrefix(function.Name.Name, "Test") {
				continue
			}
			declarations[function.Name.Name] = append(declarations[function.Name.Name], path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(declarations) == 0 {
		t.Fatal("indexed no test declarations, so every row below would resolve vacuously")
	}
	return declarations
}

// TestTheRefusalGuardInventoryIsDerivedFromTheReaderSource is the mechanism that
// exists because six rounds of review closed one guard at a time and the seventh
// found the twin of the sixth one file away.
//
// The reviewer's round-6 finding was that the ErrUnregisteredExitStatus gate in
// Read was proven at four sampled points while its twin, decodeExitStatus, had
// been swept over the same domain one round earlier. The pattern was applied to
// one of the two because the fix worked from memory rather than from a list.
// This test is that list, and it is derived rather than written: the rows are
// required to be in bijection with the refusal sites the reader source actually
// contains, so a guard added without a row reddens here before a reviewer has to
// find it.
//
// It does not measure whether a guard is well covered - that is each row's own
// named test. It measures that no guard is missing from the traversal, and that
// every row resolves to a real site, a real sentinel and a real test.
func TestTheRefusalGuardInventoryIsDerivedFromTheReaderSource(t *testing.T) {
	t.Parallel()

	sites := derivedRefusalSites(t)
	rows, siblings := readmeInventory(t)
	tests := declaredTests(t)

	// Every row resolves to exactly one derived site.
	matchedSite := make([]bool, len(sites))
	for _, row := range rows {
		matches := []int{}
		for index, site := range sites {
			if site.function == row.guardFunction() && site.sentinel == row.sentinel &&
				strings.Contains(site.format, row.marker) {
				matches = append(matches, index)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("README:%d row %s resolves to %d refusal sites in %s; a row must name exactly one "+
				"(function %q, sentinel %q, marker %q)",
				row.line, row.number, len(matches), readerSourceFile,
				row.guardFunction(), row.sentinel, row.marker)
		}
		if matchedSite[matches[0]] {
			t.Fatalf("README:%d row %s resolves to a refusal site an earlier row already claimed",
				row.line, row.number)
		}
		matchedSite[matches[0]] = true

		classified := false
		for _, class := range evidenceClasses {
			if strings.HasPrefix(row.evidence, class) {
				classified = true
			}
		}
		if !classified {
			t.Fatalf("README:%d row %s opens its domain-evidence cell with %q, which is not one of %v",
				row.line, row.number, firstWords(row.evidence), evidenceClasses)
		}
		if len(row.tests) == 0 {
			t.Fatalf("README:%d row %s names no test, so nothing measures it", row.line, row.number)
		}
		for _, name := range row.tests {
			if _, declared := tests[name]; !declared {
				t.Fatalf("README:%d row %s names %s, which is not declared anywhere under internal/",
					row.line, row.number, name)
			}
		}
	}

	// Every derived site is claimed by a row. This is the direction that would
	// have caught the round-7 finding: the gate existed in the source and no
	// row stated what measured it.
	for index, claimed := range matchedSite {
		if !claimed {
			t.Fatalf("%s carries a refusal in %s wrapping %s with the format %q, and no inventory row "+
				"claims it; a guard without a row is a guard nobody traversed",
				readerSourceFile, sites[index].function, sites[index].sentinel, sites[index].format)
		}
	}
	if len(rows) != len(sites) {
		t.Fatalf("the inventory publishes %d rows for %d derived refusal sites", len(rows), len(sites))
	}

	// The sibling table is enumerated rather than derived, so its pin is
	// resolution: each named function must be declared in its named file, and
	// each named test must exist.
	twin := false
	for _, sibling := range siblings {
		if !declaresFunction(t, filepath.Join("..", "..", sibling.file), sibling.function) {
			t.Fatalf("README:%d names %s in %s, which does not declare it",
				sibling.line, sibling.function, sibling.file)
		}
		if len(sibling.tests) == 0 {
			t.Fatalf("README:%d sibling row %q names no test", sibling.line, sibling.guard)
		}
		for _, name := range sibling.tests {
			if _, declared := tests[name]; !declared {
				t.Fatalf("README:%d sibling row names %s, which is not declared anywhere under internal/",
					sibling.line, name)
			}
		}
		if sibling.function == "decodeExitStatus" {
			twin = true
		}
	}
	// The twin is required by name. The whole point of the second table is that
	// the gate one file away sits next to its counterpart; a table that quietly
	// dropped it would satisfy every count above.
	if !twin {
		t.Fatal("the sibling table does not carry decodeExitStatus, which is the twin of row 1 and " +
			"the reason this inventory exists")
	}
}

// guardFunction extracts the function name a derived row names. It is the first
// backticked token of the guard cell, which is the convention the table follows.
func (row inventoryRow) guardFunction() string {
	for _, part := range strings.Split(row.guard, "`") {
		if part != "" && !strings.Contains(part, " ") {
			return part
		}
	}
	return ""
}

func firstWords(text string) string {
	if len(text) > 40 {
		return text[:40]
	}
	return text
}

func declaresFunction(t *testing.T, path, name string) bool {
	t.Helper()
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	for _, declaration := range parsed.Decls {
		if function, isFunction := declaration.(*ast.FuncDecl); isFunction && function.Name.Name == name {
			return true
		}
	}
	return false
}

// TestTheFanInSentenceCannotReportAMapMissAsAMeasurement settles the question
// the round-6 review left open: whether exitStatusIsNotEnough should distinguish
// "this status carries no registered code" from "this status is not in the
// table".
//
// It computes len(fanIn[status]) with no membership check, so for a status
// outside the Section 15.2 table it would report 0 in the sentence README
// publishes as a measured count - a map miss dressed as a measurement.
//
// The answer is not a defensive branch. A branch no production path can take
// promises nothing and is never measured; what makes the sentence honest is that
// the input it fabricates on cannot reach it, and that is a property to prove
// rather than to comment. This test proves it structurally and numerically:
//
//   - documentSchema, which is the only caller of exitStatusIsNotEnough, has
//     exactly one caller itself, and that caller is Read;
//   - inside Read the ErrUnregisteredExitStatus gate precedes the documentSchema
//     call, so a status outside the table never reaches the sentence; and
//   - for every registered failure status of every registered error version the
//     measured count is at least one, so a 0 in that sentence could only ever
//     have been a map miss.
//
// Reordering the gate behind documentSchema reddens the third bullet of
// TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain and the position
// assertion here.
func TestTheFanInSentenceCannotReportAMapMissAsAMeasurement(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, readerSourceFile, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", readerSourceFile, err)
	}

	callers := make(map[string][]string)
	positions := make(map[string]map[string]token.Pos)
	for _, declaration := range parsed.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		if !isFunction || function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}
			identifier, isIdent := call.Fun.(*ast.Ident)
			if !isIdent {
				return true
			}
			callers[identifier.Name] = append(callers[identifier.Name], function.Name.Name)
			if positions[function.Name.Name] == nil {
				positions[function.Name.Name] = make(map[string]token.Pos)
			}
			if _, seen := positions[function.Name.Name][identifier.Name]; !seen {
				positions[function.Name.Name][identifier.Name] = call.Pos()
			}
			return true
		})
	}

	if got := callers["documentSchema"]; len(got) != 1 || got[0] != "Read" {
		t.Fatalf("documentSchema is called from %v, want exactly [Read]; the fan-in sentence is "+
			"honest only while every path to it runs through Read's gate", got)
	}
	for _, caller := range callers["exitStatusIsNotEnough"] {
		if caller != "documentSchema" {
			t.Fatalf("exitStatusIsNotEnough is called from %q, and only documentSchema sits behind "+
				"Read's gate", caller)
		}
	}
	if len(callers["exitStatusIsNotEnough"]) == 0 {
		t.Fatal("exitStatusIsNotEnough has no caller in the reader source, so this pin measures nothing")
	}

	// The gate must precede the call. Its position is taken from the sentinel
	// reference inside Read, which is the gate's only mention there.
	var gate token.Pos
	for _, declaration := range parsed.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		if !isFunction || function.Name.Name != "Read" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			identifier, isIdent := node.(*ast.Ident)
			if isIdent && identifier.Name == "ErrUnregisteredExitStatus" && gate == token.NoPos {
				gate = identifier.Pos()
			}
			return true
		})
	}
	if gate == token.NoPos {
		t.Fatal("Read does not mention ErrUnregisteredExitStatus, so nothing gates the fan-in sentence")
	}
	call, present := positions["Read"]["documentSchema"]
	if !present {
		t.Fatal("Read does not call documentSchema")
	}
	if gate > call {
		t.Fatalf("Read reaches documentSchema at %s before the ErrUnregisteredExitStatus gate at %s, "+
			"so an unregistered status reaches the fan-in sentence and is reported as 0 registered codes",
			fileSet.Position(call), fileSet.Position(gate))
	}

	// The numeric half: 0 is never a true measurement anywhere the sentence can
	// be reached, so a 0 printed there could only ever have been a map miss.
	// Measured over every registered error version rather than over the
	// versions the command table happens to bind.
	versions := axerror.Versions()
	if len(versions) == 0 {
		t.Fatal("the registry declares no version, so this measurement has no denominator")
	}
	measured := 0
	for _, version := range versions {
		fanIn, fanInErr := axerror.CodesByExitStatus(version)
		if fanInErr != nil {
			t.Fatalf("CodesByExitStatus(%s): %v", version, fanInErr)
		}
		for _, status := range registeredFailureExitStatuses(t) {
			if len(fanIn[status]) == 0 {
				t.Fatalf("Structured Error %s assigns no code to exit status %d, so the fan-in "+
					"sentence can state 0 as a true measurement and the map-miss reading of 0 "+
					"is no longer distinguishable", version, status)
			}
			measured++
		}
	}
	if want := len(versions) * len(registeredFailureExitStatuses(t)); measured != want {
		t.Fatalf("measured %d (version, status) pairs, want %d", measured, want)
	}
}
