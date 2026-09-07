// Derived digit-guard census over the provider-protocol leaf.
//
// The failure this file exists to prevent: a digit-parsing gate added,
// narrowed, or widened anywhere in this leaf while the suite stays
// green. It happened four rounds in a row — a saturation guard
// witnessed far from its own edge, an exponent bound witnessed 44
// bytes above its true edge, an uncensused semverMajor, and finally a
// pre-multiply guard (`value > 100` in descriptorGeometry) whose
// narrowing to its own arithmetic edge (`value > 922337203685477580`)
// admitted a 37-digit literal as 1 while every shipped vector still
// refused. Each round closed one site with one hand-written row while
// the census listed its own sites, so the next site was invisible by
// construction.
//
// This census derives its denominator from production instead of
// listing it: it parses every production file of internal/
// terminalbackend and internal/provhost (directory-derived, so a new
// file is scanned without anyone remembering it) and enumerates both
// digit-guard shapes in one pass:
//
//   - char: an ordering comparison against a decimal digit rune
//     (`c < '0' || c > '9'` and its spellings, refuse-form or
//     accept-form, including the digit case of a hex decoder);
//   - bound: an ordering comparison against a decimal accumulator —
//     an identifier the same function accumulates through `* 10`
//     (`value > N` / `major > N`, before or after the multiply).
//
// Every derived site carries exactly one declared row: a char row
// names its reachable rejected class, a bound row names its
// arithmetic-edge threshold and its witness. The census fails closed
// in three directions: a derived site with no row, a row with no
// derived site, and a site the classifier cannot place (an equality
// against a digit rune, a chain mixing digit and accumulator
// comparisons, a chain mixing either with unrelated comparisons, or
// a comparison with accumulators on both sides). A zero-file scan, a
// zero-site derivation, an unparseable file, a row with an empty
// claim, and a row naming a test that does not exist are fatal too.
//
// The derivation keys on structure, never on identifier names: rune
// literals and `* 10` accumulations survive renaming, var bindings,
// and import aliases. Control plants prove it (see the battery
// record): a fourth digit gate through a var binding and one through
// an import-aliased file both fail as unregistered sites.
//
// Stated bounds (not inferred):
//   - Non-digit rune guards (`== '.'`, `!= '\\'`, `!= 'u'`, the
//     `0xed`/`0xa0` byte ranges) are a different class and invisible
//     to this enumerator. A digit-rune novelty (`== '0'`, `!= '9'`)
//     is derived and fails as unclassifiable.
//   - Digit spellings outside the comparison classifier, completed per
//     review N-R1 (three semantically identical spellings; the earlier
//     "no such site in either package" claim was false for the third
//     and is corrected here, not repeated):
//     1. integer code points (`c < 0x30`, `c > 57`) are ENUMERATED:
//     digitGuardDigitRune admits token.INT 48..57 in every base,
//     pinned by TestDigitGuardDigitRuneAdmitsCodePointSpellings;
//     2. named rune constants (`const zero = '0'; c < zero`) are
//     OUTSIDE: an identifier is not a literal, so the comparison
//     classifies as other. A pure named-constant chain (every leaf
//     other) prunes silently — a control plant of an additive pure
//     named-const digit gate in provhost/protocol.go leaves both
//     packages green, so this spelling is a genuine blind spot, not
//     a fail-closed one. Only a mixed chain (a named-constant leaf
//     beside a literal-digit or accumulator leaf) fails as
//     unclassifiable. No such site exists in either package today
//     (reviewer-grepped at N-R1, re-verified for this leaf by AST
//     scan: zero digit-valued named constants and zero such
//     comparisons); the absence is a stated bound, not a derived
//     pin.
//     3. strconv-delegated admission is OUTSIDE the comparison
//     classifier: it is not a comparison at all, so there is no
//     chain to classify. One such site EXISTS:
//     internal/provhost/opdecode.go rawUint53
//     (`strconv.ParseUint(literal, 10, 64)` + error branch +
//     magnitude bound `parsed > maxUint53`). It is behaviorally
//     pinned, not censused: narrowing the bound to `maxUint53+1`
//     (admitting exactly 2^53) reddens
//     TestDecodeQuiesceRefusals/count_overflow, which refuses 2^53
//     as "not a uint53" (battery row B-provhost-strconv-bound
//     reproduces exactly that plant and kill). A second delegated
//     site is covered only if it carries the same shape of
//     behavioral pin; the census still fails closed on every
//     comparison it CAN see.
//   - The hex letter cases (`>= 'a'`, `<= 'F'`) carry no digit rune
//     and are invisible; only the digit case of each hex decoder is
//     rowed. The decoder's exactness across all three cases rests
//     with the canonical-JSON agreement sweeps the hex rows name.
//   - An accumulator with no guard has no bound to row: a missing
//     guard is caught by the overflow-vector tests, not here.
//   - Shadowing an accumulator name inside its own function would
//     misattribute the comparison; no such shadowing exists today.
package terminalbackend_test

import (
	"bytes"
	"github.com/relux-works/agent-session-manager/internal/invcore"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	terminalbackend "github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// digitGuardSite is one derived production guard: a maximal ||/&&
// chain of comparisons (or one lone comparison) mentioning a digit
// rune or a decimal accumulator, keyed by package directory, file,
// enclosing function, shape, and printer-normalized expression. Two
// identical guards in one function share a key and fail the census
// until they are disambiguated: hiding behind a sibling row is a
// silent admit.
type digitGuardSite struct {
	dir      string
	function string
	file     string
	kind     string
	expr     string
}

// digitGuardRow is the declared counterpart of one site. A char row
// states the reachable rejected class; a bound row states the
// arithmetic-edge threshold and the witness literal that pins it.
// Tests names the driving tests, which must exist.
type digitGuardRow struct {
	dir       string
	file      string
	function  string
	kind      string
	expr      string
	rejected  string
	threshold string
	witness   string
	tests     []string
}

// digitGuardRows declares every digit guard in the leaf. The expr
// column is the printer-normalized chain text the derivation
// computes; it is exact on purpose, so editing a bound orphans its
// row instead of passing silently.
var digitGuardRows = []digitGuardRow{
	{
		dir: "terminalbackend", file: "descriptor.go", function: "descriptorGeometry",
		kind: "char", expr: `digit < '0' || digit > '9'`,
		rejected: "reachable {-, +, ., e, E}: json.Number literals spell only [0-9.+eE-], so / (0x2F) and : (0x3A) die in the strict decoder before the gate runs (pinned by TestDigitCensusAdjacentsNeverReachGeometryGate); + is equivalent by reachability, legal only after e/E which refuse first",
		tests:    []string{"TestParseProviderDescriptorValueRefusals", "TestDigitCensusAdjacentsNeverReachGeometryGate"},
	},
	{
		dir: "terminalbackend", file: "manifest.go", function: "readUTF16EscapeUnit",
		kind: "char", expr: `digit >= '0' && digit <= '9'`,
		rejected: "the digit case of the three-case hex decoder: bytes outside 0-9a-fA-F reach the refuse arm (return false, refused as a document syntax/surrogate escape); the case admits exactly 0-9",
		tests:    []string{"TestDocumentSurrogateEscapeRefused", "TestSurrogateGateAgreesWithCanonicalJSON", "TestDocumentHexEscapeDigitBoundaryRefused"},
	},
	{
		dir: "provhost", file: "protocol.go", function: "parseMajor",
		kind: "char", expr: `digit < '0' || digit > '9'`,
		rejected: "every non-digit byte in the major component (JSON string domain, so any byte is reachable): adjacents / and : witnessed in major position",
		tests:    []string{"TestParseMajorDigitBoundariesRefuseAtEntry"},
	},
	{
		dir: "provhost", file: "protocol.go", function: "parseMajor",
		kind: "char", expr: `rest[i] < '0' || rest[i] > '9'`,
		rejected: "every non-digit byte in each rest component: adjacents / and : witnessed in both minor and patch positions",
		tests:    []string{"TestParseMajorDigitBoundariesRefuseAtEntry"},
	},
	{
		dir: "provhost", file: "surrogate.go", function: "readHexUnit",
		kind: "char", expr: `b >= '0' && b <= '9'`,
		rejected: "the digit branch of the hex if-chain: bytes outside 0-9a-fA-F reach the refuse arm (return false, refused as a lone surrogate escape); the branch admits exactly 0-9",
		tests:    []string{"TestProductionEntriesRefuseLoneSurrogateEscapes", "TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON", "TestProductionEntriesRefuseHexDigitBoundaryEscapes"},
	},
	{
		dir: "terminalbackend", file: "descriptor.go", function: "descriptorGeometry",
		kind: "bound", expr: `value > 100`,
		threshold: "100: refuse on the first digit past the bound, before the multiply; value is at most 100 here so value*10+9 cannot wrap on any platform",
		witness:   "9223372036854775808000000000000000001 (37 digits): admitted as 1 when the guard narrows to floor(MaxInt/10) = 922337203685477580, refused by every threshold at or below 92233720368547758",
		tests:     []string{"TestParseProviderDescriptorGeometryOverflowVectors"},
	},
	{
		dir: "terminalbackend", file: "descriptor.go", function: "descriptorGeometry",
		kind: "bound", expr: `value < 1 || value > 1000`,
		threshold: "lower 1 / upper 1000: the post-accumulation range deciding the 1..1000 bound after the loop",
		witness:   "boundary pair 0/1 and 1000/1001, plus 65535",
		tests:     []string{"TestParseProviderDescriptorValueRefusals", "TestParseProviderDescriptorGeometryBounds"},
	},
	{
		dir: "terminalbackend", file: "terminalbackend.go", function: "semverMajor",
		kind: "bound", expr: `major > (math.MaxInt-digit)/10`,
		threshold: "floor((MaxInt-digit)/10): saturate to MaxInt, never wrap; largest admitted 9223372036854775807, MaxInt+1 saturates",
		witness:   "922337203685477580801.0.0 with neighbours 800/802: walks the accumulator to MaxInt/10 and aliases native major 1 when the guard narrows to MaxInt/10",
		tests:     []string{"TestSemverMajorSaturationEdge"},
	},
	{
		dir: "provhost", file: "protocol.go", function: "parseMajor",
		kind: "bound", expr: `major > (math.MaxInt-step)/10`,
		threshold: "floor((MaxInt-step)/10): saturate to MaxInt, never wrap; largest admitted 9223372036854775807, MaxInt+1 saturates",
		witness:   "922337203685477580802.0.0 with neighbours 801/803: walks the accumulator to MaxInt/10 and aliases native major 2 when the guard narrows to MaxInt/10",
		tests:     []string{"TestParseMajorSaturationEdge", "TestParseMajorNeverWraps"},
	},
}

// digitGuardKey identifies one site or row in both directions.
func digitGuardKey(dir, file, function, kind, expr string) string {
	return dir + "|" + file + "|" + function + "|" + kind + "|" + expr
}

// TestDigitCensusCoversEveryLeafGuard derives every digit guard from
// leaf production source and requires each to resolve to exactly one
// declared row naming a real test, and every declared row to resolve
// to exactly one derived guard.
func TestDigitCensusCoversEveryLeafGuard(t *testing.T) {
	t.Parallel()

	sites, unclassifiable, files := deriveDigitGuardSites(t)
	if len(unclassifiable) > 0 {
		sort.Strings(unclassifiable)
		t.Fatalf("unclassifiable digit-guard shapes (%d), classify them before the suite passes:\n%s",
			len(unclassifiable), strings.Join(unclassifiable, "\n"))
	}
	seen := map[string]int{}
	for _, site := range sites {
		seen[digitGuardKey(site.dir, site.file, site.function, site.kind, site.expr)]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Fatalf("derived digit guard %q occurs %d times; disambiguate the rows instead of hiding behind a sibling", key, count)
		}
	}
	byKey := map[string]digitGuardSite{}
	for _, site := range sites {
		byKey[digitGuardKey(site.dir, site.file, site.function, site.kind, site.expr)] = site
	}
	tests := digitCensusTestNames(t)
	var unregistered []string
	for _, site := range sites {
		key := digitGuardKey(site.dir, site.file, site.function, site.kind, site.expr)
		found := false
		for _, row := range digitGuardRows {
			if digitGuardKey(row.dir, row.file, row.function, row.kind, row.expr) == key {
				found = true
				break
			}
		}
		if !found {
			unregistered = append(unregistered, "unregistered "+site.kind+" site "+key)
		}
	}
	var orphan []string
	for _, row := range digitGuardRows {
		key := digitGuardKey(row.dir, row.file, row.function, row.kind, row.expr)
		if _, ok := byKey[key]; !ok {
			orphan = append(orphan, "orphan row "+key)
			continue
		}
		switch row.kind {
		case "char":
			if row.rejected == "" {
				orphan = append(orphan, "char row "+key+" declares no rejected class")
			}
			if row.threshold != "" || row.witness != "" {
				orphan = append(orphan, "char row "+key+" carries bound fields")
			}
		case "bound":
			if row.threshold == "" || row.witness == "" {
				orphan = append(orphan, "bound row "+key+" declares no threshold or witness")
			}
			if row.rejected != "" {
				orphan = append(orphan, "bound row "+key+" carries a rejected class")
			}
		default:
			orphan = append(orphan, "row "+key+" has unknown kind "+row.kind)
		}
		if len(row.tests) == 0 {
			orphan = append(orphan, "row "+key+" names no driving test")
		}
		for _, name := range row.tests {
			if !tests[name] {
				orphan = append(orphan, "row "+key+" names missing test "+name)
			}
		}
	}
	if len(unregistered) > 0 || len(orphan) > 0 {
		sort.Strings(unregistered)
		sort.Strings(orphan)
		t.Fatalf("digit-guard census mismatch:\n%s\n%s",
			strings.Join(unregistered, "\n"), strings.Join(orphan, "\n"))
	}
	t.Logf("digit-guard census: %d/%d derived sites rowed across %d production files",
		len(sites), len(digitGuardRows), files)
}

// TestDigitCensusAdjacentsNeverReachGeometryGate proves `/` (0x2F) and
// `:` (0x3A) — the characters adjacent to the digit range — never reach
// the descriptorGeometry gate: neither spells inside a JSON number, so
// the strict decoder refuses first with the document syntax arm. A gate
// mutant shifting either bound (`digit > ':'`, `digit < '/'`) or
// skipping either character token-preservingly changes no disposition
// here; the rows exist so the census claim "unreachable" is measured,
// not asserted. Both members are driven because the geometry site is
// shared and the decoder runs before either call.
func TestDigitCensusAdjacentsNeverReachGeometryGate(t *testing.T) {
	t.Parallel()

	for _, member := range []string{"columns", "rows"} {
		for _, literal := range []string{`1/2`, `1:2`} {
			_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, member, literal))
			if err == nil || !terminalbackend.IsMismatch(err) {
				t.Fatalf("ParseProviderDescriptor(%s=%s) error = %v, want a document mismatch refusal", member, literal, err)
			}
			if got := refusalDetail(t, err); got != "document syntax" {
				t.Fatalf("ParseProviderDescriptor(%s=%s) detail = %q, want document syntax", member, literal, got)
			}
		}
	}
}

// deriveDigitGuardSites parses every production file of the two leaf
// packages and returns every digit-guard site in source order, every
// unclassifiable shape, and the scanned file count. The file set is
// directory-derived: a new production file is scanned without anyone
// remembering it.
func deriveDigitGuardSites(t *testing.T) (sites []digitGuardSite, unclassifiable []string, files int) {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("derive digit guards: %v", err)
	}
	dirs := []struct {
		label string
		path  string
	}{
		{"terminalbackend", root},
		{"provhost", filepath.Join(root, "..", "provhost")},
	}
	for _, dir := range dirs {
		// Selection and fail-closed parsing are the shared core's;
		// the core returns names sorted, preserving source order.
		productions, _ := invcore.MustScanProduction(t, dir.path)
		files += len(productions)
		for _, production := range productions {
			name, syntax := production.Name, production.Syntax
			for _, decl := range syntax.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					if decl.Body == nil {
						continue
					}
					function := decl.Name.Name
					if decl.Recv != nil && len(decl.Recv.List) > 0 {
						function = digitGuardReceiver(decl.Recv.List[0].Type) + "." + function
					}
					more, bad := digitGuardSitesInBody(dir.label, name, function, decl.Body)
					sites = append(sites, more...)
					unclassifiable = append(unclassifiable, bad...)
				case *ast.GenDecl:
					more, bad := digitGuardSitesInInit(dir.label, name, decl)
					sites = append(sites, more...)
					unclassifiable = append(unclassifiable, bad...)
				}
			}
		}
	}
	if files == 0 {
		t.Fatal("derived digit guards from zero production files; the scanner is broken, not the leaf")
	}
	if len(sites) == 0 {
		t.Fatal("derived zero digit guards from the leaf sources; the scanner is broken, not the leaf")
	}
	return sites, unclassifiable, files
}

// digitGuardReceiver names the receiver type of a method for the site
// key: value and pointer receivers share the key space.
func digitGuardReceiver(expr ast.Expr) string {
	expr = digitGuardUnparen(expr)
	if star, ok := expr.(*ast.StarExpr); ok {
		return digitGuardReceiver(star.X)
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "recv"
}

// digitGuardUnparen strips parentheses.
func digitGuardUnparen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

// digitGuardSitesInInit scans one package-level declaration with no
// accumulator in scope: only char chains can appear here, and any
// accumulator-shaped comparison fails as unclassifiable rather than
// entering silently.
func digitGuardSitesInInit(dir, file string, decl *ast.GenDecl) ([]digitGuardSite, []string) {
	var sites []digitGuardSite
	var bad []string
	ast.Inspect(decl, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		site, failure, done := digitGuardClassifyRoot(dir, file, "(package initializer)", expr, map[string]bool{})
		if done {
			if failure != "" {
				bad = append(bad, failure)
			} else if site != nil {
				sites = append(sites, *site)
			}
			return false
		}
		return true
	})
	return sites, bad
}

// digitGuardSitesInBody scans one function body. Accumulators are
// collected first so a guard comparison resolves against the
// function's own accumulation steps, never against a name list.
func digitGuardSitesInBody(dir, file, function string, body *ast.BlockStmt) ([]digitGuardSite, []string) {
	accumulators := digitGuardAccumulators(body)
	var sites []digitGuardSite
	var bad []string
	ast.Inspect(body, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		site, failure, done := digitGuardClassifyRoot(dir, file, function, expr, accumulators)
		if done {
			if failure != "" {
				bad = append(bad, failure)
			} else if site != nil {
				sites = append(sites, *site)
			}
			return false
		}
		return true
	})
	return sites, bad
}

// digitGuardClassifyRoot decides whether expr is a chain root worth
// classifying: a ||/&& whose parent the Inspect walk already consumed
// is part of the outer chain, and a comparison inside any ||/&&
// chain belongs to its root. Non-root nodes report done=false so the
// walk descends. A root with no digit or accumulator content reports
// done=true with no site, pruning the walk below it.
func digitGuardClassifyRoot(dir, file, function string, expr ast.Expr, accumulators map[string]bool) (*digitGuardSite, string, bool) {
	binary, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return nil, "", false
	}
	if binary.Op == token.LOR || binary.Op == token.LAND {
		return digitGuardClassifyChain(dir, file, function, binary, accumulators)
	}
	if digitGuardIsComparison(binary.Op) {
		leaf := digitGuardClassifyLeaf(binary, accumulators)
		switch leaf {
		case digitLeafChar:
			return &digitGuardSite{dir: dir, file: file, function: function, kind: "char", expr: digitGuardCanon(binary)}, "", true
		case digitLeafBound:
			return &digitGuardSite{dir: dir, file: file, function: function, kind: "bound", expr: digitGuardCanon(binary)}, "", true
		case digitLeafBad:
			return nil, "unclassifiable guard in " + file + " (" + function + "): " + digitGuardCanon(binary), true
		default:
			return nil, "", false
		}
	}
	return nil, "", false
}

// digitGuardClassifyChain gathers the leaves of one maximal ||/&&
// chain and places the chain: pure digit chains are char sites,
// pure accumulator chains are bound sites, anything mixed or
// degenerate is unclassifiable, and chains with no digit or
// accumulator content are pruned.
func digitGuardClassifyChain(dir, file, function string, root *ast.BinaryExpr, accumulators map[string]bool) (*digitGuardSite, string, bool) {
	var chars, bounds, other, bad int
	var visit func(expr ast.Expr)
	visit = func(expr ast.Expr) {
		expr = digitGuardUnparen(expr)
		if inner, ok := expr.(*ast.BinaryExpr); ok && (inner.Op == token.LOR || inner.Op == token.LAND) {
			visit(inner.X)
			visit(inner.Y)
			return
		}
		if negated, ok := expr.(*ast.UnaryExpr); ok && negated.Op == token.NOT {
			if digitGuardMentionsDigitOrAccumulator(negated.X, accumulators) {
				bad++
			} else {
				other++
			}
			return
		}
		if comparison, ok := expr.(*ast.BinaryExpr); ok && digitGuardIsComparison(comparison.Op) {
			switch digitGuardClassifyLeaf(comparison, accumulators) {
			case digitLeafChar:
				chars++
			case digitLeafBound:
				bounds++
			case digitLeafBad:
				bad++
			default:
				other++
			}
			return
		}
		other++
	}
	visit(root)
	canon := digitGuardCanon(root)
	where := "unclassifiable guard in " + file + " (" + function + "): " + canon
	switch {
	case bad > 0:
		return nil, where, true
	case chars > 0 && bounds > 0:
		return nil, where + " (mixes digit and accumulator comparisons)", true
	case chars > 0 && other > 0:
		return nil, where + " (mixes digit comparisons with unrelated comparisons)", true
	case bounds > 0 && other > 0:
		return nil, where + " (mixes accumulator comparisons with unrelated comparisons)", true
	case chars > 0:
		return &digitGuardSite{dir: dir, file: file, function: function, kind: "char", expr: canon}, "", true
	case bounds > 0:
		return &digitGuardSite{dir: dir, file: file, function: function, kind: "bound", expr: canon}, "", true
	default:
		return nil, "", true
	}
}

// digit leaf classes.
const (
	digitLeafOther = iota
	digitLeafChar
	digitLeafBound
	digitLeafBad
)

// digitGuardIsComparison reports whether op is a comparison,
// including the equality operators the classifier refuses.
func digitGuardIsComparison(op token.Token) bool {
	switch op {
	case token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL, token.NEQ:
		return true
	default:
		return false
	}
}

// digitGuardClassifyLeaf places one comparison: an ordering against
// a digit rune is char, an ordering with exactly one accumulator
// side is bound, an equality against a digit rune and every
// degenerate shape (digit against accumulator, accumulators on both
// sides) is bad, and everything else is other.
func digitGuardClassifyLeaf(comparison *ast.BinaryExpr, accumulators map[string]bool) int {
	_, xDigit := digitGuardDigitRune(comparison.X)
	_, yDigit := digitGuardDigitRune(comparison.Y)
	xAcc := digitGuardIsAccumulator(comparison.X, accumulators)
	yAcc := digitGuardIsAccumulator(comparison.Y, accumulators)
	switch comparison.Op {
	case token.LSS, token.GTR, token.LEQ, token.GEQ:
		switch {
		case (xDigit || yDigit) && (xAcc || yAcc):
			return digitLeafBad
		case xDigit || yDigit:
			return digitLeafChar
		case xAcc && yAcc:
			return digitLeafBad
		case xAcc || yAcc:
			return digitLeafBound
		default:
			return digitLeafOther
		}
	case token.EQL, token.NEQ:
		if xDigit || yDigit {
			return digitLeafBad
		}
		return digitLeafOther
	default:
		return digitLeafOther
	}
}

// digitGuardMentionsDigitOrAccumulator reports whether expr contains
// a digit rune or a decimal accumulator: a negation over either is
// unclassifiable, over neither is unrelated.
func digitGuardMentionsDigitOrAccumulator(expr ast.Expr, accumulators map[string]bool) bool {
	mentions := false
	ast.Inspect(expr, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BasicLit:
			if _, ok := digitGuardDigitRune(node); ok {
				mentions = true
				return false
			}
		case *ast.Ident:
			if accumulators[node.Name] {
				mentions = true
				return false
			}
		case *ast.BinaryExpr:
			if node.Op == token.MUL {
				if id, ok := digitGuardUnparen(node.X).(*ast.Ident); ok && accumulators[id.Name] {
					if lit, ok := digitGuardUnparen(node.Y).(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "10" {
						mentions = true
						return false
					}
				}
				if id, ok := digitGuardUnparen(node.Y).(*ast.Ident); ok && accumulators[id.Name] {
					if lit, ok := digitGuardUnparen(node.X).(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "10" {
						mentions = true
						return false
					}
				}
			}
		}
		return true
	})
	return mentions
}

// digitGuardDigitRune reports whether expr spells a decimal digit code
// point. Rune literals count in every spelling (the gate sees the value,
// not the spelling: '0', '\x30', '\u0030'), as do integer code points in
// every base (48, 0x30, 0o60, 0X30, 48 with underscores): a gate written
// `c < 0x30 || c > 0x39` is the same gate. Named rune constants and
// strconv-delegated admission are explicitly OUTSIDE this classifier (see
// the stated bounds above): an identifier is not a literal, and an
// error-branch gate is not a comparison.
func digitGuardDigitRune(expr ast.Expr) (rune, bool) {
	lit, ok := digitGuardUnparen(expr).(*ast.BasicLit)
	if !ok {
		return 0, false
	}
	switch lit.Kind {
	case token.CHAR:
		inner := strings.TrimPrefix(strings.TrimSuffix(lit.Value, "'"), "'")
		value, _, _, err := strconv.UnquoteChar(inner, '\'')
		if err != nil {
			return 0, false
		}
		if value < '0' || value > '9' {
			return 0, false
		}
		return value, true
	case token.INT:
		value, err := strconv.ParseInt(lit.Value, 0, 32)
		if err != nil {
			return 0, false
		}
		if value < '0' || value > '9' {
			return 0, false
		}
		return rune(value), true
	default:
		return 0, false
	}
}

// digitGuardIsAccumulator reports whether expr is a bare reference
// to a decimal accumulator of the enclosing function.
func digitGuardIsAccumulator(expr ast.Expr, accumulators map[string]bool) bool {
	ident, ok := digitGuardUnparen(expr).(*ast.Ident)
	return ok && accumulators[ident.Name]
}

// digitGuardAccumulators collects the decimal accumulators of one
// function: identifiers assigned from an expression multiplying the
// identifier itself by the integer literal 10. The name is never
// matched against a list; a renamed accumulator is derived like any
// other.
func digitGuardAccumulators(body *ast.BlockStmt) map[string]bool {
	accumulators := map[string]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 {
			return true
		}
		target, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		for _, rhs := range assign.Rhs {
			if digitGuardMultipliesByTen(rhs, target.Name) {
				accumulators[target.Name] = true
			}
		}
		return true
	})
	return accumulators
}

// digitGuardMultipliesByTen reports whether expr contains the
// identifier multiplying itself by ten, in either operand order.
func digitGuardMultipliesByTen(expr ast.Expr, name string) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		product, ok := node.(*ast.BinaryExpr)
		if !ok || product.Op != token.MUL {
			return true
		}
		if digitGuardIsTenTimes(product.X, product.Y, name) || digitGuardIsTenTimes(product.Y, product.X, name) {
			found = true
			return false
		}
		return true
	})
	return found
}

// digitGuardIsTenTimes reports whether factor is the identifier and
// scale is the integer literal 10.
func digitGuardIsTenTimes(factor, scale ast.Expr, name string) bool {
	ident, ok := digitGuardUnparen(factor).(*ast.Ident)
	if !ok || ident.Name != name {
		return false
	}
	lit, ok := digitGuardUnparen(scale).(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "10"
}

// digitGuardCanon renders the printer-normalized chain text the rows
// declare: positions never enter the key, so moving a guard keeps
// its row while editing it orphans it.
func digitGuardCanon(expr ast.Expr) string {
	var rendered bytes.Buffer
	if err := printer.Fprint(&rendered, token.NewFileSet(), expr); err != nil {
		return ""
	}
	return rendered.String()
}

// TestDigitGuardDigitRuneAdmitsCodePointSpellings pins the N-R1
// widening: integer code points in every base spell the same gate as
// the rune literal, adjacent code points do not, and the two
// explicitly-outside spellings (named constants, strconv delegation)
// stay outside the classifier.
func TestDigitGuardDigitRuneAdmitsCodePointSpellings(t *testing.T) {
	t.Parallel()

	literal := func(kind token.Token, value string) ast.Expr {
		return &ast.BasicLit{Kind: kind, Value: value}
	}
	admitted := []ast.Expr{
		literal(token.CHAR, "'0'"), literal(token.CHAR, "'9'"),
		literal(token.CHAR, `'\x30'`), literal(token.CHAR, `'\u0039'`),
		literal(token.INT, "48"), literal(token.INT, "57"),
		literal(token.INT, "0x30"), literal(token.INT, "0x39"),
		literal(token.INT, "0X30"), literal(token.INT, "0o60"),
		literal(token.INT, "060"), literal(token.INT, "4_8"),
	}
	for _, expr := range admitted {
		if _, ok := digitGuardDigitRune(expr); !ok {
			t.Errorf("digitGuardDigitRune(%v %q) = false, want true: the spelling is the same gate",
				expr.(*ast.BasicLit).Kind, expr.(*ast.BasicLit).Value)
		}
	}
	rejected := []ast.Expr{
		literal(token.INT, "47"), literal(token.INT, "58"),
		literal(token.INT, "0x2F"), literal(token.INT, "0x3A"),
		literal(token.CHAR, "'/'"), literal(token.CHAR, "':'"),
		literal(token.FLOAT, "48.0"),
		&ast.Ident{Name: "zero"},
		&ast.CallExpr{Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "strconv"}, Sel: &ast.Ident{Name: "Atoi"}}},
	}
	for _, expr := range rejected {
		if _, ok := digitGuardDigitRune(expr); ok {
			t.Errorf("digitGuardDigitRune(%T) = true, want false: adjacents and non-literals are not digit gates", expr)
		}
	}
}

// digitCensusTestNames collects every test function defined in the
// leaf test files, so a row cannot name a test that does not exist.
func digitCensusTestNames(t *testing.T) map[string]bool {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("collect leaf test names: %v", err)
	}
	names := map[string]bool{}
	for _, dir := range []string{root, filepath.Join(root, "..", "provhost")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("collect leaf test names: read %s: %v", dir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
				continue
			}
			// Test-file selection stays local (the core selects
			// production files only); fail-closed parsing is the
			// core's.
			path := filepath.Join(dir, name)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("collect leaf test names: read %s: %v", path, err)
			}
			syntax, _, failure := invcore.ParseBytes(path, source, 0)
			if failure != "" {
				t.Fatalf("collect leaf test names: parse %s: %s", path, failure)
			}
			for _, decl := range syntax.Decls {
				function, ok := decl.(*ast.FuncDecl)
				if !ok || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "Test") {
					continue
				}
				names[function.Name.Name] = true
			}
		}
	}
	if len(names) == 0 {
		t.Fatal("collected zero leaf test names; the scanner is broken, not the leaf")
	}
	return names
}
