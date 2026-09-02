package canonicaljson

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file binds every closed vocabulary this package admits to a reviewed pin.
//
// The failure it exists to prevent: a derived mutation sweep over the production
// sources admitted one extra member at each of the 47 vocabulary call sites in
// turn, and 47 of 47 mutants SURVIVED. `go test ./...`, all four seeded fuzz
// corpora, `tracecheck`, the derived declared-bounds gate and the derived
// refusal-guard coverage gate were all green while, for example,
// validateLeaseRecord admitted a fifth lease reason.
//
// The cause is that a "refuses one value outside the vocabulary" test proves the
// gate is REACHABLE, not that the admitted set is the DECLARED set. Coverage
// cannot see it either: the widened call site still executes its refusal for the
// outside value the test happens to pick. Only binding the admitted set to a
// reviewed pin makes widening fail, which is the same shape as the declared
// bounds inventory: the obligation carries the literal, so the mutant dies at
// derivation before any case runs.

const closedVocabularyInventoryFile = "closed-vocabularies.md"

// vocabularySite is one production call that admits a closed set of strings.
type vocabularySite struct {
	file     string
	line     int
	function string
	// scope is the enclosing composite-literal key, which is the Session Event
	// type when the vocabulary is declared inside the payload registry.
	scope  string
	member string
	values []string
}

func (site vocabularySite) row() string {
	quoted := make([]string, 0, len(site.values))
	for _, value := range site.values {
		quoted = append(quoted, "`"+value+"`")
	}
	scope := site.scope
	if scope == "" {
		scope = "-"
	}
	return fmt.Sprintf("%s|%s|%s|%s|%s", site.file, site.function, scope, site.member, strings.Join(quoted, ", "))
}

// TestEveryClosedVocabularyAdmitsExactlyItsPinnedSet is the gate. Production is
// derived; the expected set is pinned. Adding an admitted member, removing one,
// reordering them, adding an unpinned call site, or deleting a pinned one all
// fail here.
func TestEveryClosedVocabularyAdmitsExactlyItsPinnedSet(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	derived := deriveClosedVocabularySites(t)
	pinned := readClosedVocabularyInventory(t, filepath.Join(directory, "testdata", closedVocabularyInventoryFile))

	if len(derived) == 0 {
		t.Fatal("derived zero closed vocabularies from the package sources; the scanner is broken, not the package")
	}
	for index := 0; index < len(derived) || index < len(pinned); index++ {
		switch {
		case index >= len(pinned):
			t.Errorf("production vocabulary %s:%d has no pinned row:\n  %s",
				derived[index].file, derived[index].line, derived[index].row())
		case index >= len(derived):
			t.Errorf("pinned vocabulary row %d has no production call site:\n  %s", index+1, pinned[index])
		case derived[index].row() != pinned[index]:
			t.Errorf(
				"closed vocabulary %s:%d does not match its pinned row. A widened set admits a value the "+
					"pinned specification does not declare; a narrowed set refuses one it does.\n"+
					"  production: %s\n  pinned    : %s",
				derived[index].file, derived[index].line, derived[index].row(), pinned[index])
		}
	}
}

// TestClosedVocabularyDerivationCoversEveryAdmittingHelper is the gate's own
// coverage proof, the property this Story was rejected for lacking twice: an
// inventory that scans one form of a construct is blind to an equivalent
// duplicate. It derives every function in the package that admits a closed set
// of string literals and fails unless that set is exactly what the scanner
// walks, so a second copy of requireEnum reddens instead of quietly carrying
// unpinned vocabularies.
func TestClosedVocabularyDerivationCoversEveryAdmittingHelper(t *testing.T) {
	t.Parallel()

	found := deriveVocabularyAdmittingHelpers(t)
	want := map[string]bool{"requireEnum": true, "enum": true}
	for name := range found {
		if !want[name] {
			t.Errorf(
				"%s admits a closed vocabulary but the inventory does not walk its call sites, so every "+
					"vocabulary it carries is unpinned. Collapse it into requireEnum or extend the scanner.",
				name)
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("vocabulary-admitting helper %s was not derived; every pinned row below it is unsound", name)
		}
	}
}

// deriveVocabularyAdmittingHelpers returns every package function that takes a
// variadic `...string` of admitted values and either compares against it or
// forwards it to another such function. Nothing here is a name list.
func deriveVocabularyAdmittingHelpers(t *testing.T) map[string]bool {
	t.Helper()

	_, files := packageProductionFiles(t)
	fileSet := token.NewFileSet()
	admitting := map[string]bool{}
	var declarations []*ast.FuncDecl
	for _, path := range files {
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil {
				continue
			}
			if !hasVariadicStringParameter(function) {
				continue
			}
			declarations = append(declarations, function)
		}
	}
	// Seed: a function that yields the admitted value and decides admission from
	// its own variadic set — either by ranging over it, or by forwarding it to an
	// admission primitive outside this package.
	for _, function := range declarations {
		if !returnsStringFirst(function) {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if rangeStatement, ok := node.(*ast.RangeStmt); ok {
				if identifier, ok := rangeStatement.X.(*ast.Ident); ok && identifier.Name == variadicName(function) {
					admitting[function.Name.Name] = true
				}
				return true
			}
			call, ok := node.(*ast.CallExpr)
			if !ok || !call.Ellipsis.IsValid() || len(call.Args) == 0 {
				return true
			}
			last, ok := call.Args[len(call.Args)-1].(*ast.Ident)
			if !ok || last.Name != variadicName(function) {
				return true
			}
			if _, external := call.Fun.(*ast.SelectorExpr); external {
				admitting[function.Name.Name] = true
			}
			return true
		})
	}
	// Closure: a wrapper forwarding its variadic set to an already-derived one.
	for changed := true; changed; {
		changed = false
		for _, function := range declarations {
			if admitting[function.Name.Name] {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || !call.Ellipsis.IsValid() {
					return true
				}
				callee, ok := call.Fun.(*ast.Ident)
				if !ok || !admitting[callee.Name] {
					return true
				}
				admitting[function.Name.Name] = true
				changed = true
				return true
			})
		}
	}
	return admitting
}

// returnsStringFirst separates a closed VOCABULARY from a closed MEMBER SET.
// requireExactMembers also carries a variadic string set, but it returns only an
// error; pinning its member lists is the separate job of
// constraint-enumeration.md. A vocabulary helper yields the value it admitted.
func returnsStringFirst(function *ast.FuncDecl) bool {
	if function.Type.Results == nil || len(function.Type.Results.List) == 0 {
		return false
	}
	identifier, ok := function.Type.Results.List[0].Type.(*ast.Ident)
	return ok && identifier.Name == "string"
}

func hasVariadicStringParameter(function *ast.FuncDecl) bool {
	if function.Type.Params == nil || len(function.Type.Params.List) == 0 {
		return false
	}
	last := function.Type.Params.List[len(function.Type.Params.List)-1]
	ellipsis, ok := last.Type.(*ast.Ellipsis)
	if !ok {
		return false
	}
	element, ok := ellipsis.Elt.(*ast.Ident)
	return ok && element.Name == "string" && len(last.Names) == 1
}

func variadicName(function *ast.FuncDecl) string {
	last := function.Type.Params.List[len(function.Type.Params.List)-1]
	return last.Names[0].Name
}

// deriveClosedVocabularySites walks every call to a derived admitting helper and
// returns the member and the admitted values in declaration order. A call whose
// member or values are not string literals FAILS the derivation with its
// position rather than being skipped, so a computed vocabulary cannot slip past
// the inventory unnoticed.
func deriveClosedVocabularySites(t *testing.T) []vocabularySite {
	t.Helper()

	_, files := packageProductionFiles(t)
	admitting := deriveVocabularyAdmittingHelpers(t)
	fileSet := token.NewFileSet()
	var sites []vocabularySite
	for _, path := range files {
		name := filepath.Base(path)
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		scopes := enclosingLiteralKeys(parsed)
		var current string
		ast.Inspect(parsed, func(node ast.Node) bool {
			if function, ok := node.(*ast.FuncDecl); ok {
				current = function.Name.Name
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee, ok := call.Fun.(*ast.Ident)
			if !ok || !admitting[callee.Name] || call.Ellipsis.IsValid() {
				return true
			}
			// The admitted values are the trailing arguments; the member name is
			// the argument directly before them. requireEnum also takes the
			// object, so the member is located by skipping non-literal leading
			// arguments rather than by hard-coding an index.
			var literals []string
			for _, argument := range call.Args {
				literal, ok := argument.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					literals = nil
					continue
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("%s:%d unquote %s: %v", name, fileSet.Position(literal.Pos()).Line, literal.Value, err)
				}
				literals = append(literals, value)
			}
			position := fileSet.Position(call.Pos())
			if len(literals) < 2 {
				t.Fatalf(
					"%s:%d %s admits a vocabulary whose member or values are not string literals, so it cannot be "+
						"pinned. Spell it literally or extend the derivation.",
					name, position.Line, callee.Name)
			}
			sites = append(sites, vocabularySite{
				file: name, line: position.Line, function: current,
				scope: scopes[call], member: literals[0], values: literals[1:],
			})
			return true
		})
	}
	sort.SliceStable(sites, func(first, second int) bool {
		if sites[first].file != sites[second].file {
			return sites[first].file < sites[second].file
		}
		return sites[first].line < sites[second].line
	})
	return sites
}

// enclosingLiteralKeys maps each node to the nearest enclosing composite-literal
// string key, so a vocabulary declared inside the Session Event payload registry
// is attributed to its event type.
func enclosingLiteralKeys(parsed *ast.File) map[ast.Node]string {
	scopes := map[ast.Node]string{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		pair, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		literal, ok := pair.Key.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		key, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		ast.Inspect(pair.Value, func(inner ast.Node) bool {
			if inner != nil {
				if _, seen := scopes[inner]; !seen {
					scopes[inner] = key
				}
			}
			return true
		})
		return true
	})
	return scopes
}

// vocabularyInventoryRow is one parsed artifact row. The declaration columns are
// kept rather than discarded so they can be checked, which is the difference
// between a provenance column and decoration.
type vocabularyInventoryRow struct {
	key         string
	member      string
	values      []string
	specLine    string
	declaration string
}

// readClosedVocabularyInventory returns the pinned rows in file order, in the
// same `file|function|scope|member|values` shape the derivation produces.
func readClosedVocabularyInventory(t *testing.T, path string) []string {
	t.Helper()

	rows := readClosedVocabularyInventoryRows(t, path)
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.key)
	}
	return keys
}

func readClosedVocabularyInventoryRows(t *testing.T, path string) []vocabularyInventoryRow {
	t.Helper()

	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = handle.Close() }()

	var rows []vocabularyInventoryRow
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 7 {
			t.Fatalf("pinned vocabulary row has %d cells, want 7: %s", len(cells), line)
		}
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		values := make([]string, 0, 8)
		for _, value := range strings.Split(cells[4], ",") {
			values = append(values, strings.Trim(strings.TrimSpace(value), "`"))
		}
		rows = append(rows, vocabularyInventoryRow{
			key: fmt.Sprintf("%s|%s|%s|%s|%s",
				strings.Trim(cells[0], "`"), strings.Trim(cells[1], "`"),
				strings.Trim(cells[2], "`"), strings.Trim(cells[3], "`"), cells[4]),
			member:      strings.Trim(cells[3], "`"),
			values:      values,
			specLine:    cells[5],
			declaration: cells[6],
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatalf("read zero pinned vocabulary rows from %s; the reader is broken, not the artifact", path)
	}
	return rows
}

// TestEveryPinnedVocabularyRowQuotesADeclarationThatContainsItsMembers checks
// the two provenance columns the reader previously parsed and threw away.
//
// The artifact cannot verify itself against SPEC.md - the document is not
// vendored, and vendoring it to satisfy a test would be a different change - but
// three findings on this Story were invented constraints written into a
// declaration column, so an unchecked declaration column is exactly the shape
// that produced them. What is checkable here is checked: the SPEC line must be a
// positive line number, the declaration must name the member and every admitted
// value as a whole token, and two rows quoting the same declaration must cite
// the same line. A row widened with an extra member now also has to have that
// member appear in the quoted declaration, where a reviewer reads it.
func TestEveryPinnedVocabularyRowQuotesADeclarationThatContainsItsMembers(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	rows := readClosedVocabularyInventoryRows(t, filepath.Join(directory, "testdata", closedVocabularyInventoryFile))
	for _, problem := range checkVocabularyProvenanceColumns(rows) {
		t.Error(problem)
	}
}

// TestVocabularyProvenanceCheckReportsAMiscitedRow is the check's own negative
// proof. A provenance check that cannot fail is the decoration this file exists
// to replace, so the same function is run against synthetic rows that are wrong
// in each way it claims to detect, and each must be reported.
func TestVocabularyProvenanceCheckReportsAMiscitedRow(t *testing.T) {
	t.Parallel()

	valid := vocabularyInventoryRow{
		key: "f.go|validate|-|reason|`create`, `recovery`", member: "reason",
		values: []string{"create", "recovery"}, specLine: "1908",
		declaration: "reason &#124; enum &#124; create, recovery",
	}
	if problems := checkVocabularyProvenanceColumns([]vocabularyInventoryRow{valid}); len(problems) != 0 {
		t.Fatalf("a correct row was reported: %v", problems)
	}

	miscited := valid
	miscited.values = []string{"create", "recovery", "smuggled_member"}
	unnamedMember := valid
	unnamedMember.member = "not_in_the_declaration"
	badLine := valid
	badLine.specLine = "not-a-line"
	emptyDeclaration := valid
	emptyDeclaration.declaration = "  "
	// "create" is a whole token of the declaration; "creat" is only a substring,
	// and a substring test would wave it through.
	substringOnly := valid
	substringOnly.values = []string{"creat"}

	for name, row := range map[string]vocabularyInventoryRow{
		"admits a member the declaration never names": miscited,
		"member absent from the declaration":          unnamedMember,
		"SPEC line is not a line number":              badLine,
		"declaration is empty":                        emptyDeclaration,
		"admitted value is only a substring":          substringOnly,
	} {
		if problems := checkVocabularyProvenanceColumns([]vocabularyInventoryRow{row}); len(problems) == 0 {
			t.Errorf("the provenance check accepted a row that %s", name)
		}
	}

	conflicting := []vocabularyInventoryRow{valid, valid}
	conflicting[1].specLine = "9999"
	if problems := checkVocabularyProvenanceColumns(conflicting); len(problems) == 0 {
		t.Error("the provenance check accepted the same declaration cited at two different SPEC lines")
	}
}

// checkVocabularyProvenanceColumns returns one problem per unusable provenance
// citation. It is a pure function so the negative proof above can drive it.
func checkVocabularyProvenanceColumns(rows []vocabularyInventoryRow) []string {
	var problems []string
	lineByDeclaration := make(map[string]string)
	for _, row := range rows {
		line, err := strconv.Atoi(row.specLine)
		if err != nil || line <= 0 {
			problems = append(problems, fmt.Sprintf(
				"vocabulary row %s cites SPEC line %q, which is not a positive line number", row.key, row.specLine))
		}
		if strings.TrimSpace(row.declaration) == "" {
			problems = append(problems, fmt.Sprintf("vocabulary row %s quotes no pinned SPEC declaration", row.key))
			continue
		}
		tokens := declarationTokens(row.declaration)
		if !tokens[row.member] {
			problems = append(problems, fmt.Sprintf(
				"vocabulary row %s quotes a declaration that never names the member %q, so the citation cannot be "+
					"checked against the document: %s", row.key, row.member, row.declaration))
		}
		for _, value := range row.values {
			if !tokens[value] {
				problems = append(problems, fmt.Sprintf(
					"vocabulary row %s admits %q but its quoted declaration does not contain that member. Either "+
						"production admits a value the specification does not declare, or the citation is wrong: %s",
					row.key, value, row.declaration))
			}
		}
		if previous, seen := lineByDeclaration[row.declaration]; seen && previous != row.specLine {
			problems = append(problems, fmt.Sprintf(
				"the same pinned declaration %q is cited at both SPEC line %s and line %s; at most one can be right",
				row.declaration, previous, row.specLine))
		}
		lineByDeclaration[row.declaration] = row.specLine
	}
	return problems
}

// declarationTokens splits a quoted declaration into whole identifier tokens, so
// "session" does not appear to be declared merely because "session_uuid" is.
func declarationTokens(declaration string) map[string]bool {
	tokens := make(map[string]bool)
	current := strings.Builder{}
	flush := func() {
		if current.Len() > 0 {
			tokens[current.String()] = true
			current.Reset()
		}
	}
	for _, character := range declaration {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '_', character == '.':
			current.WriteRune(character)
		default:
			flush()
		}
	}
	flush()
	return tokens
}
