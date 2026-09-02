package canonicaljson

import (
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

// This file closes the fifth and last invented-constraint class found on this
// leaf: a validator that enforces a STRONGER array rule than the pinned
// specification declares.
//
// The instance: Section 5.2 declares Session Event `predecessors` as "a sorted
// array of one or more record/event digests". Production validated it with
// validateSortedUniqueDigests, so a duplicate predecessor was refused. Section
// 1.6 defines `sorted unique T[n..m]` as the COMPOUND phrase meaning "bytewise
// canonical ordering and no duplicate", and the document uses that phrase
// systematically - event_heads, manifest_ids, evidence_ids, object_ids,
// accepted_risks, sanitized_remote_urls, agent_project_config_paths - while
// using bare `sorted` for predecessors alone. Reading the two as synonyms
// invents a refusal the contract does not declare.
//
// Four earlier instances on this leaf were each fixed one at a time. This gate
// makes the class mechanical instead: the specification uses a CONTROLLED
// VOCABULARY for array constraints, so every ordering site in the package is
// derived from production, its strength is read off the comparison OPERATOR,
// and the pinned SPEC declaration it cites must use the phrase that carries
// exactly that strength. Judgement never enters; a strengthened or weakened
// validator cannot be written down consistently.
//
// The derivation proves its own coverage two ways. Ordering sites are found
// structurally - a comparison between a current element and a previous element,
// where both are traced through the loop's dataflow - so renaming a refusal
// message cannot hide one. And every production refusal whose message speaks
// about order must belong to a function that carries a derived site, so an
// ordering check written in a shape the tracer does not model reddens instead
// of passing unpinned.

const arrayOrderInventoryFile = "array-order-constraints.md"

// Strength names are the specification's own phrases, not implementation words.
const (
	// orderSorted is the bare `sorted` phrase: bytewise non-descending order,
	// duplicates admitted.
	orderSorted = "sorted"
	// orderSortedUnique is the Section 1.6 compound phrase: bytewise canonical
	// ordering AND no duplicate.
	orderSortedUnique = "sorted unique"
)

// orderingSite is one production comparison that decides the relative order of
// two elements of one array.
type orderingSite struct {
	file     string
	line     int
	function string
	op       token.Token
	text     string
	// members is the array member name this site orders. It is derived: for a
	// site inside a reusable helper the names come from the helper's call sites,
	// and for a site validating one fixed array they come from the call that
	// produced the ranged collection.
	members []string
}

// strength reads the declared rule off the comparison OPERATOR, which is the
// only thing that decides whether a duplicate survives.
//
//	refuse when current <= previous  =>  strictly ascending  =>  sorted unique
//	refuse when current <  previous  =>  non-descending      =>  sorted
func (site orderingSite) strength() string {
	switch site.op {
	case token.LEQ, token.GEQ:
		return orderSortedUnique
	case token.LSS, token.GTR:
		return orderSorted
	}
	return "?"
}

// orderingRow is one (site, member) pair as the pinned artifact records it.
type orderingRow struct {
	file     string
	line     int
	function string
	member   string
	enforces string
}

func (row orderingRow) key() string {
	return fmt.Sprintf("%s|%s|%s|%s", row.file, row.function, row.member, row.enforces)
}

func deriveOrderingRows(t *testing.T) []orderingRow {
	t.Helper()
	var rows []orderingRow
	for _, site := range deriveOrderingSites(t) {
		for _, member := range site.members {
			rows = append(rows, orderingRow{
				file:     site.file,
				line:     site.line,
				function: site.function,
				member:   member,
				enforces: site.strength(),
			})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].file != rows[j].file {
			return rows[i].file < rows[j].file
		}
		if rows[i].line != rows[j].line {
			return rows[i].line < rows[j].line
		}
		return rows[i].member < rows[j].member
	})
	return rows
}

// packageOrderSources parses every production file once and indexes the
// function declarations by name so member tracing can cross call boundaries.
type packageOrderSources struct {
	fileSet   *token.FileSet
	files     []*ast.File
	baseNames []string
	functions map[string]*ast.FuncDecl
	fileOf    map[string]string
}

func loadPackageOrderSources(t *testing.T) *packageOrderSources {
	t.Helper()
	_, paths := packageProductionFiles(t)
	sources := &packageOrderSources{
		fileSet:   token.NewFileSet(),
		functions: map[string]*ast.FuncDecl{},
		fileOf:    map[string]string{},
	}
	for _, path := range paths {
		parsed, err := parser.ParseFile(sources.fileSet, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		base := filepath.Base(path)
		sources.files = append(sources.files, parsed)
		sources.baseNames = append(sources.baseNames, base)
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil {
				continue
			}
			sources.functions[function.Name.Name] = function
			sources.fileOf[function.Name.Name] = base
		}
	}
	return sources
}

func deriveOrderingSites(t *testing.T) []orderingSite {
	t.Helper()
	sources := loadPackageOrderSources(t)
	var sites []orderingSite
	for index, parsed := range sources.files {
		base := sources.baseNames[index]
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				loop, ok := node.(*ast.RangeStmt)
				if !ok {
					return true
				}
				current, previous := loopElementRoots(loop)
				if len(previous) == 0 {
					return true
				}
				for _, comparison := range elementOrderComparisons(loop.Body, current, previous) {
					position := sources.fileSet.Position(comparison.Pos())
					sites = append(sites, orderingSite{
						file:     base,
						line:     position.Line,
						function: function.Name.Name,
						op:       comparison.Op,
						text:     renderComparison(comparison),
						members:  traceOrderedMemberNames(t, sources, function, loop.X, map[string]bool{}),
					})
				}
				return true
			})
		}
	}
	sort.SliceStable(sites, func(i, j int) bool {
		if sites[i].file != sites[j].file {
			return sites[i].file < sites[j].file
		}
		if sites[i].line != sites[j].line {
			return sites[i].line < sites[j].line
		}
		return sites[i].text < sites[j].text
	})
	return sites
}

// loopElementRoots performs the loop-local dataflow. current is the transitive
// closure of identifiers derived from the ranged element; previous is every
// identifier bound to an EARLIER element, either by carrying a current value
// forward or by indexing an accumulator at an offset from the loop key.
func loopElementRoots(loop *ast.RangeStmt) (current, previous map[string]bool) {
	current = map[string]bool{}
	previous = map[string]bool{}
	if value, ok := loop.Value.(*ast.Ident); ok && value.Name != "_" {
		current[value.Name] = true
	}
	key := ""
	if identifier, ok := loop.Key.(*ast.Ident); ok {
		key = identifier.Name
	}
	if loop.Body == nil {
		return current, previous
	}
	// Iterate to a fixed point: an element value may be derived through several
	// assignments before it reaches the comparison.
	for changed := true; changed; {
		changed = false
		ast.Inspect(loop.Body, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for index, target := range assignment.Lhs {
				identifier, ok := target.(*ast.Ident)
				if !ok || identifier.Name == "_" {
					continue
				}
				source := assignmentSource(assignment, index)
				if source == nil {
					continue
				}
				if indexesEarlierElement(source, key) && !previous[identifier.Name] {
					previous[identifier.Name] = true
					changed = true
					continue
				}
				if current[identifier.Name] {
					continue
				}
				// A plain assignment of an element-ROOTED expression to an
				// identifier that is not itself an element carries this element
				// into the next iteration. Requiring the root rather than mere
				// containment keeps `accumulator = append(accumulator, element)`
				// out: an accumulator is not the previous element.
				if assignment.Tok == token.ASSIGN && current[rootIdentifier(source)] {
					if !previous[identifier.Name] {
						previous[identifier.Name] = true
						changed = true
					}
					continue
				}
				// Any value COMPUTED from the element - a type assertion, a
				// member read, a validator call - is still this element.
				if assignment.Tok == token.DEFINE && !previous[identifier.Name] && expressionMentions(source, current) {
					current[identifier.Name] = true
					changed = true
				}
			}
			return true
		})
	}
	return current, previous
}

// assignmentSource pairs one left-hand target with its right-hand expression,
// handling the multi-value `a, err := f(x)` form where every target draws from
// the single call.
func assignmentSource(assignment *ast.AssignStmt, index int) ast.Expr {
	if len(assignment.Rhs) == len(assignment.Lhs) {
		return assignment.Rhs[index]
	}
	if len(assignment.Rhs) == 1 {
		return assignment.Rhs[0]
	}
	return nil
}

// indexesEarlierElement recognises `accumulator[index-1]`: a binding to the
// element before the current one.
func indexesEarlierElement(expression ast.Expr, key string) bool {
	indexed, ok := expression.(*ast.IndexExpr)
	if !ok || key == "" {
		return false
	}
	offset, ok := indexed.Index.(*ast.BinaryExpr)
	if !ok || offset.Op != token.SUB {
		return false
	}
	return rootIdentifier(offset.X) == key
}

// elementOrderComparisons returns the comparisons that order a current element
// against a previous one, and only those. A comparison against a literal or a
// bound variable is a numeric range check owned by the declared-bounds gate.
func elementOrderComparisons(body *ast.BlockStmt, current, previous map[string]bool) []*ast.BinaryExpr {
	var found []*ast.BinaryExpr
	ast.Inspect(body, func(node ast.Node) bool {
		expression, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		switch expression.Op {
		case token.LSS, token.LEQ, token.GTR, token.GEQ:
		default:
			return true
		}
		left, right := rootIdentifier(expression.X), rootIdentifier(expression.Y)
		if left == "" || right == "" {
			return true
		}
		if current[left] && previous[right] || previous[left] && current[right] {
			found = append(found, expression)
		}
		return true
	})
	return found
}

// rootIdentifier reduces a selector, index or conversion chain to the
// identifier it is rooted at, and returns "" for anything literal-rooted.
func rootIdentifier(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return rootIdentifier(value.X)
	case *ast.IndexExpr:
		return rootIdentifier(value.X)
	case *ast.ParenExpr:
		return rootIdentifier(value.X)
	case *ast.StarExpr:
		return rootIdentifier(value.X)
	case *ast.TypeAssertExpr:
		return rootIdentifier(value.X)
	}
	return ""
}

// expressionMentions reports whether any identifier in the expression is in the
// set, which is how an element-derived value is recognised across a call.
func expressionMentions(expression ast.Expr, names map[string]bool) bool {
	mentioned := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && names[identifier.Name] {
			mentioned = true
		}
		return !mentioned
	})
	return mentioned
}

// traceOrderedMemberNames answers "which declared array member does this loop
// order?". It follows the ranged collection back to the string literal that
// named it. Two edges make it work across the package: when the value is a
// PARAMETER, the enclosing function is a reusable ordering helper and the trace
// continues at each of its call sites; when the value was produced by a call,
// the trace continues into that call's arguments, which is where every array
// accessor in this package takes its member name. Nothing here is a name list,
// so a new helper or a new call site contributes its members automatically.
func traceOrderedMemberNames(t *testing.T, sources *packageOrderSources, function *ast.FuncDecl, expression ast.Expr, visiting map[string]bool) []string {
	t.Helper()
	if literal, ok := stringLiteral(expression); ok {
		return []string{literal}
	}
	root := rootIdentifier(expression)
	if root == "" {
		return nil
	}
	guard := function.Name.Name + "|" + root
	if visiting[guard] {
		return nil
	}
	visiting[guard] = true
	defer delete(visiting, guard)

	if position, isParameter := orderParameterIndex(function, root); isParameter {
		return orderedMemberNamesFromCallers(t, sources, function, position, visiting)
	}
	call, ok := producingCall(function, root)
	if !ok {
		return nil
	}
	// A direct string-literal argument names the member; it always wins over a
	// traced one, because the object the accessor reads is itself a member of
	// some outer record and would otherwise shadow the inner name.
	for _, argument := range call.Args {
		if literal, ok := stringLiteral(argument); ok {
			return []string{literal}
		}
	}
	for _, argument := range call.Args {
		if names := traceOrderedMemberNames(t, sources, function, argument, visiting); len(names) > 0 {
			return names
		}
	}
	return nil
}

// orderedMemberNamesFromCallers resolves a helper parameter by tracing the
// matching argument at every production call site of that helper.
func orderedMemberNamesFromCallers(t *testing.T, sources *packageOrderSources, helper *ast.FuncDecl, position int, visiting map[string]bool) []string {
	t.Helper()
	seen := map[string]bool{}
	var names []string
	for _, file := range sources.files {
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee, ok := call.Fun.(*ast.Ident)
			if !ok || callee.Name != helper.Name.Name || position >= len(call.Args) {
				return true
			}
			caller := enclosingFunction(file, node)
			if caller == nil || caller == helper {
				return true
			}
			for _, name := range traceOrderedMemberNames(t, sources, caller, call.Args[position], visiting) {
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
			return true
		})
	}
	sort.Strings(names)
	return names
}

// producingCall returns the call expression that assigned a local identifier.
func producingCall(function *ast.FuncDecl, name string) (*ast.CallExpr, bool) {
	var found *ast.CallExpr
	ast.Inspect(function.Body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for index, target := range assignment.Lhs {
			identifier, ok := target.(*ast.Ident)
			if !ok || identifier.Name != name {
				continue
			}
			if call, ok := assignmentSource(assignment, index).(*ast.CallExpr); ok && found == nil {
				found = call
			}
		}
		return true
	})
	return found, found != nil
}

func orderParameterIndex(function *ast.FuncDecl, name string) (int, bool) {
	if function.Type.Params == nil {
		return 0, false
	}
	position := 0
	for _, field := range function.Type.Params.List {
		for _, parameter := range field.Names {
			if parameter.Name == name {
				return position, true
			}
			position++
		}
		if len(field.Names) == 0 {
			position++
		}
	}
	return 0, false
}

func enclosingFunction(file *ast.File, node ast.Node) *ast.FuncDecl {
	var enclosing *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		if function.Pos() <= node.Pos() && node.End() <= function.End() {
			enclosing = function
		}
	}
	return enclosing
}

func stringLiteral(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	return text, true
}

// pinnedOrderRow is one row of the reviewed artifact.
type pinnedOrderRow struct {
	file       string
	function   string
	member     string
	declaredOn string
	enforces   string
	orderLine  string
	orderQuote string
	uniqueLine string
	// uniqueQuote is the sentinel "-" when the document declares no uniqueness
	// for this array, and "strengthens" when production refuses a duplicate the
	// document does not declare anywhere.
	uniqueQuote string
	index       int
}

func (row pinnedOrderRow) key() string {
	return fmt.Sprintf("%s|%s|%s|%s", row.file, row.function, row.member, row.enforces)
}

const (
	orderNoDeclaration    = "-"
	orderStrengthensNoted = "strengthens"
)

// declaredOrderStrengthenings names every ordering site that refuses a
// duplicate the pinned document does not declare at any line, together with the
// leaf that owns the section it sits in. The set is asserted EXACTLY against the
// artifact, so a fourth strengthening cannot be introduced silently: it either
// matches its declared phrase or it appears here and is reported.
var declaredOrderStrengthenings = map[string]string{
	"closed_shapes.go|validateWorkspaceSnapshot|members|sorted unique": "TASK-260830-uqnwmi",
	"closed_shapes.go|validateGitIndex|entries|sorted unique":          "TASK-260830-uqnwmi",
}

// TestEveryArrayOrderConstraintMatchesItsPinnedPhrase is the gate. Production is
// derived; the expected sequence is pinned. Strengthening a validator, weakening
// one, adding an ordering site, adding a call site to an ordering helper, or
// deleting a pinned one all fail here, because the derived row carries the
// strength read off the comparison operator.
func TestEveryArrayOrderConstraintMatchesItsPinnedPhrase(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	derived := deriveOrderingRows(t)
	pinned := readArrayOrderInventory(t, filepath.Join(directory, "testdata", arrayOrderInventoryFile))

	if len(derived) == 0 {
		t.Fatal("derived zero array order constraints from the package sources; the scanner is broken, not the package")
	}
	// An ordering site whose member cannot be traced produces no row, so it
	// would slip past the sequence comparison below entirely. A new ordering
	// helper that nothing calls is exactly that shape.
	for _, problem := range untraceableOrderingSites(deriveOrderingSites(t)) {
		t.Error(problem)
	}
	for index := 0; index < len(derived) || index < len(pinned); index++ {
		switch {
		case index >= len(pinned):
			t.Errorf("production order constraint %s:%d has no pinned row:\n  %s",
				derived[index].file, derived[index].line, derived[index].key())
		case index >= len(derived):
			t.Errorf("pinned order row %d has no production ordering site:\n  %s", index+1, pinned[index].key())
		case derived[index].key() != pinned[index].key():
			t.Errorf(
				"array order constraint %s:%d does not match its pinned row. A validator that enforces "+
					"`sorted unique` where the pinned document declares bare `sorted` refuses a duplicate the "+
					"contract admits; the reverse admits an order the contract refuses.\n"+
					"  production: %s\n  pinned    : %s",
				derived[index].file, derived[index].line, derived[index].key(), pinned[index].key())
		}
	}
}

// TestEveryArrayOrderRowCitesTheDeclarationItsValidatorEnforces is the
// phrase-to-validator mapping itself, applied to every row. It is what turns
// "read the specification carefully" into a mechanical check.
func TestEveryArrayOrderRowCitesTheDeclarationItsValidatorEnforces(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	pinned := readArrayOrderInventory(t, filepath.Join(directory, "testdata", arrayOrderInventoryFile))
	for _, problem := range checkArrayOrderProvenance(pinned) {
		t.Error(problem)
	}
}

// TestArrayOrderProvenanceCheckReportsAMisdeclaredRow is the negative proof of
// the checker above. checkArrayOrderProvenance is a pure function so it can be
// driven with rows that are wrong in each way it claims to detect; if the check
// is disabled or narrowed, these cases stop being reported and this test fails.
func TestArrayOrderProvenanceCheckReportsAMisdeclaredRow(t *testing.T) {
	t.Parallel()

	valid := pinnedOrderRow{
		file: "core_records.go", function: "validateSortedUniqueDigests", member: "event_heads",
		declaredOn: "event_heads", enforces: orderSortedUnique,
		orderLine: "1982", orderQuote: "event_heads | sorted unique digest[1..64]",
		uniqueLine: "1982", uniqueQuote: "event_heads | sorted unique digest[1..64]",
	}
	if problems := checkArrayOrderProvenance([]pinnedOrderRow{valid}); len(problems) != 0 {
		t.Fatalf("a correct row was reported: %v", problems)
	}

	strengthened := valid
	strengthened.uniqueLine, strengthened.uniqueQuote = orderNoDeclaration, orderNoDeclaration
	weakened := valid
	weakened.enforces = orderSorted
	unnamed := valid
	unnamed.orderQuote, unnamed.uniqueQuote = "sorted unique digest[1..64]", "sorted unique digest[1..64]"
	unordered := valid
	unordered.orderQuote = "event_heads | digest[1..64] | Authoritative event DAG heads"
	notUnique := valid
	notUnique.uniqueQuote = "event_heads | digest[1..64] | Authoritative event DAG heads"
	lowercaseType := valid
	lowercaseType.declaredOn = "heads"

	// Each case names the exact problem it must produce. Asserting only "some
	// problem was reported" would let a narrowed clause pass on a neighbouring
	// clause's refusal, which is the one-directional-proof shape this leaf was
	// rejected for in round five.
	cases := []struct {
		name string
		row  pinnedOrderRow
		want string
	}{
		{"validator enforces uniqueness with no uniqueness declaration cited", strengthened,
			"production refuses a duplicate but the row cites no uniqueness declaration"},
		{"validator admits duplicates while citing a uniqueness declaration", weakened,
			"production admits duplicates while the row cites a uniqueness declaration"},
		{"quoted order declaration does not name the array it declares", unnamed,
			"the quoted order declaration does not name"},
		{"quoted uniqueness declaration does not name the array it declares", unnamed,
			"the quoted uniqueness declaration does not name"},
		{"quoted order declaration states no ordering", unordered,
			"the quoted order declaration states no ordering rule"},
		{"quoted uniqueness declaration states no uniqueness", notUnique,
			"uses none of the document's uniqueness phrases"},
		{"declared-on token differs from the member and is not a type name", lowercaseType,
			"is neither the member nor a CamelCase element type"},
	}
	for _, test := range cases {
		problems := checkArrayOrderProvenance([]pinnedOrderRow{test.row})
		if !containsProblem(problems, test.want) {
			t.Errorf("checkArrayOrderProvenance did not report %q for a row that is %s; reported %v",
				test.want, test.name, problems)
		}
	}

	sameQuoteDifferentLine := []pinnedOrderRow{valid, func() pinnedOrderRow {
		other := valid
		other.member, other.declaredOn = "event_heads", "event_heads"
		other.orderLine, other.uniqueLine = "1983", "1983"
		return other
	}()}
	if problems := checkArrayOrderProvenance(sameQuoteDifferentLine); len(problems) == 0 {
		t.Error("checkArrayOrderProvenance admitted the same declaration cited at two different lines")
	}
}

// TestProvenanceProblemMatchingIsExact keeps the message pinning above load
// bearing. Without it, widening containsProblem to "any problem was reported"
// lets every case below pass on a neighbouring clause's refusal, which is
// exactly the failure the message pinning exists to prevent.
func TestProvenanceProblemMatchingIsExact(t *testing.T) {
	t.Parallel()

	problems := []string{"array order row 1 (x): the quoted order declaration states no ordering rule: \"x\""}
	if !containsProblem(problems, "states no ordering rule") {
		t.Error("containsProblem missed the problem it was given")
	}
	if containsProblem(problems, "production refuses a duplicate but the row cites no uniqueness declaration") {
		t.Error("containsProblem matched a problem that was not reported, so every case that uses it proves nothing")
	}
	if containsProblem(nil, "anything") {
		t.Error("containsProblem matched against an empty problem set")
	}
}

func containsProblem(problems []string, want string) bool {
	for _, problem := range problems {
		if strings.Contains(problem, want) {
			return true
		}
	}
	return false
}

// checkArrayOrderProvenance is the mapping, as a pure function.
//
//	enforces `sorted unique`  =>  a uniqueness declaration MUST be cited
//	enforces `sorted`         =>  no uniqueness declaration may be cited
//
// A validator that refuses a duplicate the document does not declare cannot
// satisfy the first rule by citing anything, which is why the only remaining
// exit is the disclosed strengthening sentinel.
func checkArrayOrderProvenance(rows []pinnedOrderRow) []string {
	var problems []string
	quoteLines := map[string]string{}
	for _, row := range rows {
		label := fmt.Sprintf("array order row %d (%s)", row.index+1, row.key())

		switch {
		case row.declaredOn == "":
			problems = append(problems, label+": no declared-on token")
		case row.declaredOn != row.member && !startsUpper(row.declaredOn):
			problems = append(problems, fmt.Sprintf(
				"%s: declared-on token %q is neither the member nor a CamelCase element type; a member name is "+
					"lower_snake in the pinned document, so this row cites a declaration for something else",
				label, row.declaredOn))
		}
		if !namesWholeToken(row.orderQuote, row.declaredOn) {
			problems = append(problems, fmt.Sprintf(
				"%s: the quoted order declaration does not name %q as a whole token, so it does not declare this array",
				label, row.declaredOn))
		}
		if !declaresOrder(row.orderQuote) {
			problems = append(problems, fmt.Sprintf(
				"%s: the quoted order declaration states no ordering rule: %q", label, row.orderQuote))
		}

		switch row.enforces {
		case orderSortedUnique:
			switch {
			case row.uniqueQuote == orderStrengthensNoted:
				if _, disclosed := declaredOrderStrengthenings[row.key()]; !disclosed {
					problems = append(problems, label+
						": claims an undeclared strengthening that declaredOrderStrengthenings does not disclose")
				}
			case row.uniqueQuote == orderNoDeclaration:
				problems = append(problems, label+
					": production refuses a duplicate but the row cites no uniqueness declaration. Either the "+
					"pinned document declares uniqueness for this array and the row must quote it, or the "+
					"validator invents a refusal the contract does not declare")
			default:
				if !namesWholeToken(row.uniqueQuote, row.declaredOn) {
					problems = append(problems, fmt.Sprintf(
						"%s: the quoted uniqueness declaration does not name %q as a whole token", label, row.declaredOn))
				}
				if !declaresUniqueness(row.uniqueQuote) {
					problems = append(problems, fmt.Sprintf(
						"%s: the quoted uniqueness declaration uses none of the document's uniqueness phrases: %q",
						label, row.uniqueQuote))
				}
			}
		case orderSorted:
			if row.uniqueQuote != orderNoDeclaration || row.uniqueLine != orderNoDeclaration {
				problems = append(problems, label+
					": production admits duplicates while the row cites a uniqueness declaration. One of the two is wrong")
			}
		default:
			problems = append(problems, fmt.Sprintf("%s: unknown enforced phrase %q", label, row.enforces))
		}

		for _, cited := range []struct{ quote, line string }{
			{row.orderQuote, row.orderLine}, {row.uniqueQuote, row.uniqueLine},
		} {
			if cited.quote == orderNoDeclaration || cited.quote == orderStrengthensNoted {
				continue
			}
			if previous, seen := quoteLines[cited.quote]; seen && previous != cited.line {
				problems = append(problems, fmt.Sprintf(
					"%s: the declaration %q is cited at line %s here and at line %s elsewhere",
					label, cited.quote, cited.line, previous))
			}
			quoteLines[cited.quote] = cited.line
		}
	}
	return problems
}

// untraceableOrderingSites reports every derived ordering site whose member
// name could not be traced. Such a site contributes no row, so the sequence
// comparison cannot see it; the check is a pure function so its own removal is
// provable rather than only visible under a second mutation.
func untraceableOrderingSites(sites []orderingSite) []string {
	var problems []string
	for _, site := range sites {
		if len(site.members) == 0 {
			problems = append(problems, fmt.Sprintf(
				"%s:%d %s orders an array whose declared member name could not be traced, so the rule it "+
					"enforces is pinned by no row: %s",
				site.file, site.line, site.function, site.text))
		}
	}
	return problems
}

// TestUntraceableOrderingSiteIsReported is the negative proof of the check
// above. Without it, a new ordering helper that no production call site reaches
// would carry an unpinned rule and the inventory would stay green.
func TestUntraceableOrderingSiteIsReported(t *testing.T) {
	t.Parallel()

	traced := orderingSite{file: "closed_shapes.go", line: 1, function: "validateSortedUniqueDigests",
		op: token.LEQ, text: "text <= previous", members: []string{"event_heads"}}
	if problems := untraceableOrderingSites([]orderingSite{traced}); len(problems) != 0 {
		t.Fatalf("a traced ordering site was reported: %v", problems)
	}
	untraced := traced
	untraced.function, untraced.members = "validateSortedUniqueProbeDigests", nil
	if problems := untraceableOrderingSites([]orderingSite{untraced}); len(problems) != 1 {
		t.Errorf("untraceableOrderingSites reported %d problems for an untraceable site, want 1", len(problems))
	}
}

// orderPhrases and uniquenessPhrases are the specification's own controlled
// vocabulary for array constraints, not implementation words.
var orderPhrases = []string{"sorted", "in workspace-id order"}

var uniquenessPhrases = []string{"sorted unique", "sorted-unique", "sorted, unique", "no duplicate", "unique"}

func declaresOrder(quote string) bool {
	lowered := strings.ToLower(quote)
	for _, phrase := range orderPhrases {
		if strings.Contains(lowered, phrase) {
			return true
		}
	}
	return false
}

func declaresUniqueness(quote string) bool {
	lowered := strings.ToLower(quote)
	for _, phrase := range uniquenessPhrases {
		if strings.Contains(lowered, phrase) {
			return true
		}
	}
	return false
}

func namesWholeToken(quote, token string) bool {
	lowered, want := strings.ToLower(quote), strings.ToLower(token)
	for index := 0; index+len(want) <= len(lowered); index++ {
		if lowered[index:index+len(want)] != want {
			continue
		}
		if index > 0 && isTokenByte(lowered[index-1]) {
			continue
		}
		if end := index + len(want); end < len(lowered) && isTokenByte(lowered[end]) {
			continue
		}
		return true
	}
	return false
}

func isTokenByte(character byte) bool {
	return character == '_' || character >= 'a' && character <= 'z' ||
		character >= '0' && character <= '9'
}

func startsUpper(token string) bool {
	return token != "" && token[0] >= 'A' && token[0] <= 'Z'
}

// TestDeclaredOrderStrengtheningsAreExactlyTheDisclosedSet keeps the escape
// hatch closed. The disclosed set, the sentinel rows in the inventory table and
// the rows of the artifact's Strengthening section must be the same three
// facts, so an undeclared refusal cannot be parked in the artifact without also
// appearing in the source and in the reported disclosure.
func TestDeclaredOrderStrengtheningsAreExactlyTheDisclosedSet(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	path := filepath.Join(directory, "testdata", arrayOrderInventoryFile)
	sentinels := map[string]bool{}
	for _, row := range readArrayOrderInventory(t, path) {
		if row.uniqueQuote == orderStrengthensNoted {
			sentinels[row.key()] = true
		}
	}
	for key := range declaredOrderStrengthenings {
		if !sentinels[key] {
			t.Errorf("declaredOrderStrengthenings discloses %q but the inventory carries no strengthening row for it", key)
		}
	}
	for key := range sentinels {
		if _, disclosed := declaredOrderStrengthenings[key]; !disclosed {
			t.Errorf("inventory row %q claims a strengthening that the source does not disclose", key)
		}
	}
	disclosed := readArrayOrderStrengtheningSection(t, path)
	if len(disclosed) != len(sentinels) {
		t.Errorf("the artifact's Strengthening section lists %d rows for %d sentinel rows", len(disclosed), len(sentinels))
	}
	for _, entry := range disclosed {
		owner, known := declaredOrderStrengthenings[entry.key]
		if !known {
			t.Errorf("Strengthening section lists %q, which is not a disclosed strengthening", entry.key)
			continue
		}
		if entry.owner != owner {
			t.Errorf("Strengthening row %q routes to %q; the source routes it to %q", entry.key, entry.owner, owner)
		}
		if strings.TrimSpace(entry.reason) == "" {
			t.Errorf("Strengthening row %q states no undeclared refusal", entry.key)
		}
	}
}

// TestArrayOrderDerivationCoversEveryOrderRefusal is the derivation's own
// coverage proof. Ordering sites are found by tracing element dataflow, which no
// rename can evade - but a check written in a shape the tracer does not model
// (a sort-order helper, a comparison against a value carried some other way)
// would simply not be found, and an unfound site is an unpinned one. Every
// production refusal that SPEAKS about order must therefore sit in a function
// that carries a derived site.
func TestArrayOrderDerivationCoversEveryOrderRefusal(t *testing.T) {
	t.Parallel()

	covered := map[string]bool{}
	for _, site := range deriveOrderingSites(t) {
		covered[site.file+"|"+site.function] = true
	}
	if len(covered) == 0 {
		t.Fatal("derived zero ordering sites; the tracer is broken")
	}
	sources := loadPackageOrderSources(t)
	for index, file := range sources.files {
		base := sources.baseNames[index]
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil || covered[base+"|"+function.Name.Name] {
				continue
			}
			for _, message := range refusalMessages(function) {
				if !declaresOrder(message) {
					continue
				}
				t.Errorf(
					"%s:%s refuses with an ordering message but carries no derived ordering site, so the rule it "+
						"enforces is pinned by nothing:\n  %q",
					base, function.Name.Name, message)
			}
		}
	}
}

// refusalMessages returns the format strings of every refusal a function raises.
func refusalMessages(function *ast.FuncDecl) []string {
	var messages []string
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		callee, ok := call.Fun.(*ast.Ident)
		if !ok || callee.Name != "invalidIdentity" && callee.Name != "invalidObservation" {
			return true
		}
		if text, ok := stringLiteral(call.Args[0]); ok {
			messages = append(messages, text)
		}
		return true
	})
	return messages
}

type orderStrengtheningEntry struct {
	key    string
	reason string
	owner  string
}

func readArrayOrderInventory(t *testing.T, path string) []pinnedOrderRow {
	t.Helper()
	var rows []pinnedOrderRow
	for _, cells := range readArtifactTableRows(t, path, 9) {
		rows = append(rows, pinnedOrderRow{
			file:        strings.Trim(cells[0], "`"),
			function:    strings.Trim(cells[1], "`"),
			member:      strings.Trim(cells[2], "`"),
			declaredOn:  strings.Trim(cells[3], "`"),
			enforces:    cells[4],
			orderLine:   cells[5],
			orderQuote:  decodeArtifactCell(cells[6]),
			uniqueLine:  cells[7],
			uniqueQuote: decodeArtifactCell(cells[8]),
			index:       len(rows),
		})
	}
	if len(rows) == 0 {
		t.Fatalf("read zero rows from %s", path)
	}
	return rows
}

func readArrayOrderStrengtheningSection(t *testing.T, path string) []orderStrengtheningEntry {
	t.Helper()
	var entries []orderStrengtheningEntry
	for _, cells := range readArtifactTableRows(t, path, 5) {
		entries = append(entries, orderStrengtheningEntry{
			key:    fmt.Sprintf("%s|%s|%s|%s", strings.Trim(cells[0], "`"), strings.Trim(cells[1], "`"), strings.Trim(cells[2], "`"), orderSortedUnique),
			reason: decodeArtifactCell(cells[3]),
			owner:  strings.Trim(cells[4], "`"),
		})
	}
	return entries
}

// readArtifactTableRows returns every Markdown table row of the given width,
// skipping headers and separators, so the two tables in one artifact are read
// by their shape instead of by position.
func readArtifactTableRows(t *testing.T, path string, width int) [][]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	var rows [][]string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		if len(cells) != width {
			continue
		}
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		if cells[0] == "Source" || strings.HasPrefix(cells[0], "---") {
			continue
		}
		rows = append(rows, cells)
	}
	return rows
}

func decodeArtifactCell(cell string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.Trim(cell, "`"), "&#124;", "|"), "&lt;", "<")
}
