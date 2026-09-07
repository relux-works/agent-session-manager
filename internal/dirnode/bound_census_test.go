package dirnode

// This file is the bound census (review round 2, B1), ported from
// the sessadapter leaf's bound_census_test.go. It exists because a
// new bound through any of the eight helpers — checkStringBounds,
// checkUint53Bounds, checkSortedUniqueStrings,
// checkSortedUniqueDigests, checkSortedUniqueUUIDv7,
// checkEnumSubset, checkUUIDv7DigestSubset, checkOptionalString —
// or a new explicit len(...) guard passes every behavioral test
// that never names it, and pinning checkStringBounds while the
// query validators go unmeasured reads as complete while most of
// the class stays open: the same checkStringBounds max+1 mutant
// killed in sessadapter survived on every maximum checked here.
//
// Shape, modelled on TestClosedVocabularyTablesAreRegistered:
// scanBoundSiteIDs derives every bound site from production source
// (helper calls with their member, len guards in any statement
// context), and TestBoundGuardsAreCensused requires the derived
// multiset to equal boundRegistrations exactly. A new bound in one
// of the derived shapes below with no row fails the census; a
// removed bound orphans its row and fails too. A helper alias
// (`var f = checkStringBounds`) fails as an indirection violation
// rather than a missed site. An empty scan fails closed. Site
// identity is file|function|member, never the bound values: a
// moved bound (+1) keeps its identity and must redden its
// behavioral driver, which the mutant battery proves per row.
//
// Shape space. A bound in this package can only be carried by
// these AST shapes; the census derives the marked ones and bounds
// the rest:
//
//   1. A direct call to one of the eight helpers — DERIVED with
//      the bounded member.
//   2. A bound helper through a func-value indirection (var,
//      :=, assignment, argument, return, struct field, method
//      value) — DERIVED as a census violation (the alias
//      itself fails rather than each hidden call).
//   3. A comparison holding a len(...) call in any statement
//      context (if, for, switch, return, assignment) — DERIVED.
//   4. A comparison over a length variable bound from len(...)
//      or stringLength(...) (`n := len(v); if n > 4096`) —
//      DERIVED; live mechanism rows are the shared
//      character-count enforcements in checkStringBounds,
//      checkSortedUniqueStrings, and checkURI.
//   5. A comparison holding a direct stringLength(...) call —
//      DERIVED by the same rule; live instance:
//      CheckCursorReuse.
//   6. A bound inside a package-level var initializer,
//      including func literals — DERIVED under the variable
//      name; zero live rows today, proven by the synthetic
//      package-literal test below.
//   7. A comparison over a decoded value or caller-supplied
//      number with no length source in the function
//      (`skip > 1000000` over a rawUint53 result, `deadlineMS <
//      MinDeadlineMS` over a plain parameter, `index > 63`,
//      `take < 1`, `version != 1`, cross-field gates) — STATED
//      BOUND, not derived. Each is pinned behaviourally by its
//      named driver; the bound is enforced at the entry the
//      driver drives.
//   8. The maxUint53 maxima — STATED: representability bounds
//      pinned once by TestUint53RepresentabilityCeiling rather
//      than per site.
//   9. A direct cap(...) or utf8.RuneCountInString(...)
//      comparison outside stringLength — STATED BOUND, not
//      derived. No live instance: the only production
//      rune-count use is inside stringLength, whose callers are
//      all rostered rows.
//  10. An ordering comparison with no length source
//      (`previous >= record.Key` in Journal.Import, pairwise
//      sortedness scans `values[index-1] >= values[index]`) —
//      STATED, not derived: ordering gates, not domain bounds.
//      Both halves are pinned behaviourally: the sortedness
//      refusal tests feed unsorted vectors (["b","a"]) for the
//      ordered half, and TestSortedUniqueDuplicatesRefuse feeds a
//      duplicate at every one of the eight scans for the
//      uniqueness half — a duplicate is sorted, so only it proves
//      the `>=` (not `>`) comparison.
//
// 6 of 10 shapes derived, 4 stated. A new bound in shapes 1-6
// with no row fails TestBoundGuardsAreCensused; shapes 7-10 are
// pinned by their named suites or stated here as residue.
//
// Every non-exempt row names its driver: the committed test
// proving refusal at its edges (min-1/max+1) through the
// production entry. Admission at the accepted edge itself is
// proven only where the driver says so. Exempt rows say why:
// equivalent (the bound is unreachable behind an upstream gate),
// mechanism (transparent forwarder or generic guard whose values
// live at the caller rows), or plumbing (no live rows: the frame
// byte-scanner this category once covered now delegates to
// environ.DecodeStrictObject, whose guards live in that package).
//
// Stated bounds of this census: shapes 7-10 above. _test.go
// files, which never ship, are outside the derivation, as are
// const initializers, which hold no calls and whose constant
// comparisons are vacuous.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// boundRegistration is one production bound site with its proof:
// the behavioral driver covering its edges, or an explicit
// exemption with its rationale.
type boundRegistration struct {
	id     string
	driver string
	exempt bool
	reason string
}

// boundHelpers is the closed helper set the census derives. A
// ninth bound helper with no row here fails the census only if
// it is called; the call-shape derivation below names the helper
// per site, so an unrostered helper surfaces as unregistered
// sites. checkEnumSubset, checkUUIDv7DigestSubset,
// checkOptionalString, and checkURI are bound-carrying forwards:
// their call sites roster the member with the caller-side values,
// and their bodies roster mechanism rows each.
var boundHelpers = map[string]bool{
	"checkStringBounds":        true,
	"checkUint53Bounds":        true,
	"checkSortedUniqueStrings": true,
	"checkSortedUniqueDigests": true,
	"checkSortedUniqueUUIDv7":  true,
	"checkEnumSubset":          true,
	"checkUUIDv7DigestSubset":  true,
	"checkOptionalString":      true,
	"checkURI":                 true,
}

// scanBoundSiteIDs derives every bound site identity from
// production source: file|function|helper|member|ordinal for the
// eight helpers, and file|function|len|rendered|ordinal for len
// guards in any statement context (if, for, switch, return,
// assignment), including guards over length variables bound from
// len(...) or stringLength(...) and guards holding a direct
// stringLength(...) call. Scopes are FuncDecl bodies and
// package-level var initializers under their variable name.
// Values are deliberately not part of the identity. A
// func-value indirection of any helper is a census violation,
// not a missed site. The synthetic
// TestBoundHelperIndirectionReports,
// TestBoundLenScanSeesNonIfContexts, and
// TestBoundLenScanSeesLengthVariables prove the halves.
func scanBoundSiteIDs(t *testing.T, directory string) []string {
	t.Helper()
	// Production file selection and fail-closed parsing come
	// from invcore: an unreadable or unparseable production file
	// fails the suite instead of scanning as clean, and zero
	// production files fail instead of passing vacuously. The
	// bound-site extraction below is unchanged: the synthetic
	// indirection plants drive boundSitesInSyntax directly, so
	// both prove the same extractor.
	files, _ := invcore.MustScanProduction(t, directory)
	var ids []string
	for _, production := range files {
		sites, violations := boundSitesInSyntax(production.Syntax, production.Name)
		for _, violation := range violations {
			t.Errorf("bound census: %s: %s", production.Name, violation)
		}
		ids = append(ids, sites...)
	}
	return ids
}

// boundSitesInSyntax derives the bound site identities of one
// parsed production file and reports every helper indirection.
// Direct helper calls claim file|function|helper|member|ordinal;
// every len comparison in any statement context claims
// file|function|len|rendered|ordinal — including comparisons over
// length variables (idents bound from len(...) or
// stringLength(...)) and comparisons holding a direct
// stringLength(...) call. Scopes are FuncDecl bodies and
// package-level var initializers under their variable name; the
// per-scope ordinal keeps repeated members distinct. A use of a
// helper name that is neither a direct call nor one of the eight
// function definitions is an indirection violation.
func boundSitesInSyntax(syntax *ast.File, name string) (sites []string, violations []string) {
	for _, declaration := range syntax.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Body == nil {
				continue
			}
			sites = append(sites, boundSitesInScope(declaration.Body, name, declaration.Name.Name)...)
		case *ast.GenDecl:
			if declaration.Tok != token.VAR {
				continue
			}
			for _, specification := range declaration.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, ident := range value.Names {
					for _, expr := range value.Values {
						sites = append(sites, boundSitesInScope(expr, name, ident.Name)...)
					}
				}
			}
		}
	}
	violations = append(violations, boundHelperIndirections(syntax)...)
	return sites, violations
}

// boundSitesInScope derives the bound site identities of one
// scope body: helper calls by member, len comparisons by rendered
// shape. Length variables are collected first over the same body,
// so a guard written through a variable claims the same len site
// the direct spelling would.
func boundSitesInScope(body ast.Node, name, scope string) []string {
	var sites []string
	ordinal := map[string]int{}
	claim := func(base string) {
		sites = append(sites, base+"|"+strconv.Itoa(ordinal[base]))
		ordinal[base]++
	}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if boundHelpers[ident.Name] {
			claim(name + "|" + scope + "|" + ident.Name + "|" + boundMemberOf(call))
		}
		return true
	})
	lengthVars := lengthVarsInScope(body)
	ast.Inspect(body, func(node ast.Node) bool {
		comparison, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		switch comparison.Op {
		case token.GTR, token.GEQ, token.LSS, token.LEQ, token.EQL, token.NEQ:
		default:
			return true
		}
		if !hasLengthCall(comparison) && !mentionsLengthVar(comparison, lengthVars) {
			return true
		}
		claim(name + "|" + scope + "|len|" + normalizeBoundNumbers(renderBoundExpr(comparison)))
		return true
	})
	return sites
}

// lengthSourceNames are the package's two length measures: byte
// length and the Section 1.6 character measure. A variable bound
// from either is a length variable for the len-guard derivation.
// cap(...) and a direct utf8.RuneCountInString(...) call are not
// sources (shape 9 of the census header): neither is used as a
// bound in production.
var lengthSourceNames = map[string]bool{"len": true, "stringLength": true}

// isLengthSourceCall reports whether the expression calls one of
// the length measures.
func isLengthSourceCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	return ok && lengthSourceNames[ident.Name]
}

// numericConversions are the builtin conversions a length may be
// wrapped in at the binding (`n := uint64(len(v))`) without
// stopping being a length.
var numericConversions = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true, "byte": true, "rune": true, "uintptr": true,
}

// lengthBoundValue reports whether the right-hand side binds a
// length: a length-source call, possibly through parentheses or
// one builtin numeric conversion. Anything else — a slice built
// with a len capacity (`values := make([]T, 0, len(x))`), a
// length combined arithmetically before binding, a decoded
// value — binds no length variable, so element comparisons over
// such names never claim a len site.
func lengthBoundValue(rhs ast.Expr) bool {
	for {
		paren, ok := rhs.(*ast.ParenExpr)
		if !ok {
			break
		}
		rhs = paren.X
	}
	if isLengthSourceCall(rhs) {
		return true
	}
	call, ok := rhs.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	return ok && numericConversions[ident.Name] && isLengthSourceCall(call.Args[0])
}

// hasLengthCall reports whether the subtree calls a length
// measure: len(...) or stringLength(...).
func hasLengthCall(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(inner ast.Node) bool {
		if expr, ok := inner.(ast.Expr); ok && isLengthSourceCall(expr) {
			found = true
			return false
		}
		return true
	})
	return found
}

// lengthVarsInScope collects the idents bound from a length
// measure anywhere in one scope body: `n := len(v)`, `n =
// len(v)`, and `var n = len(v)` (and the same through
// stringLength). Only a direct binding counts (see
// lengthBoundValue): the rule is deliberately scope-wide and
// sticky — a variable bound from a length stays a length
// variable even if later reassigned, and shadowing is not
// tracked. That can only over-derive — a repurposed length name
// claims a site that needs a row — never silently miss the
// reviewer plant shape.
func lengthVarsInScope(body ast.Node) map[string]bool {
	vars := map[string]bool{}
	bindIdents := func(names []*ast.Ident, values []ast.Expr) {
		for index, ident := range names {
			if ident == nil || ident.Name == "_" {
				continue
			}
			var rhs ast.Expr
			switch {
			case len(values) == 1:
				rhs = values[0]
			case index < len(values):
				rhs = values[index]
			}
			if rhs != nil && lengthBoundValue(rhs) {
				vars[ident.Name] = true
			}
		}
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			names := make([]*ast.Ident, len(node.Lhs))
			for index, lhs := range node.Lhs {
				// A non-ident target (a selector, an index)
				// holds no variable; its paired value is
				// simply unattributed.
				ident, _ := lhs.(*ast.Ident)
				names[index] = ident
			}
			bindIdents(names, node.Rhs)
		case *ast.ValueSpec:
			bindIdents(node.Names, node.Values)
		}
		return true
	})
	return vars
}

// mentionsLengthVar reports whether the subtree reads one of the
// collected length variables.
func mentionsLengthVar(node ast.Node, vars map[string]bool) bool {
	found := false
	ast.Inspect(node, func(inner ast.Node) bool {
		if ident, ok := inner.(*ast.Ident); ok && vars[ident.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// boundHelperIndirections reports every func-value indirection
// of the eight bound helpers in one parsed file: any use of a
// helper name that is neither a direct call (the identifier is
// the Fun of its enclosing call) nor one of the eight function
// definitions. The rule is positional, so it covers var and :=
// bindings at package and function scope, plain assignments,
// arguments, and return values uniformly — the same shape as the
// constructor census indirection check it is ported from.
func boundHelperIndirections(syntax *ast.File) []string {
	definitions := map[*ast.Ident]bool{}
	for _, declaration := range syntax.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || !boundHelpers[function.Name.Name] {
			continue
		}
		definitions[function.Name] = true
	}
	var violations []string
	var stack []ast.Node
	ast.Inspect(syntax, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if ident, ok := node.(*ast.Ident); ok && boundHelpers[ident.Name] {
			if !definitions[ident] && !isDirectBoundCall(stack, ident) {
				violations = append(violations, "func-value indirection of "+ident.Name+"; bounds built through it hide from the bound census")
			}
		}
		stack = append(stack, node)
		return true
	})
	return violations
}

// isDirectBoundCall reports whether the identifier is the
// function of the call enclosing it: the stack top is a call
// whose Fun is this identifier. Anything else holding the name
// — a value spec, an assignment, an argument, a return — is an
// alias, not a bound.
func isDirectBoundCall(stack []ast.Node, ident *ast.Ident) bool {
	if len(stack) == 0 {
		return false
	}
	call, ok := stack[len(stack)-1].(*ast.CallExpr)
	return ok && call.Fun == ast.Expr(ident)
}

// boundMemberOf names the bounded member of one helper call: the
// string-literal member for member-indexed calls
// (members["node_id"]), or the rendered expression when the call
// bounds a computed value (fieldsRaw, raw, element).
func boundMemberOf(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return "noargs"
	}
	first := call.Args[0]
	if literal, ok := first.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		return literal.Value
	}
	if index, ok := first.(*ast.IndexExpr); ok {
		if literal, ok := index.Index.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			return literal.Value
		}
	}
	return renderBoundExpr(first)
}

// renderBoundExpr renders one bound expression deterministically.
func renderBoundExpr(node ast.Node) string {
	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, token.NewFileSet(), node); err != nil {
		return "RENDER-ERR"
	}
	return buffer.String()
}

// normalizeBoundNumbers replaces standalone integer literals in
// a rendered len guard with N, so the census rosters the guard
// shape rather than its bound values: a moved bound keeps its
// identity and must redden its behavioral driver. Identifiers
// holding digits (uint64, MaxFrameBytes) are untouched.
func normalizeBoundNumbers(rendered string) string {
	var out strings.Builder
	for i := 0; i < len(rendered); {
		c := rendered[i]
		if c >= '0' && c <= '9' && (i == 0 || !isBoundIdentChar(rendered[i-1])) {
			j := i
			for j < len(rendered) && rendered[j] >= '0' && rendered[j] <= '9' {
				j++
			}
			if j >= len(rendered) || !isBoundIdentChar(rendered[j]) {
				out.WriteString("N")
				i = j
				continue
			}
		}
		out.WriteByte(c)
		i++
	}
	return out.String()
}

// isBoundIdentChar reports whether the byte continues a Go
// identifier.
func isBoundIdentChar(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// boundRegistrations rosters every derived site. driver names the
// committed test proving the edges at the production entry;
// exempt rows carry the rationale instead. Ordinals disambiguate
// repeated members in one function; only checkNodeBuild-adjacent
// repetition needs none because every other repeated literal in
// this package sits in a distinct function.
var boundRegistrations = []boundRegistration{
	// decode.go: the shared character-count, sorted-unique, and
	// extensions enforcements converged onto environ this leaf
	// (checkStringBounds, checkSortedUniqueStrings,
	// checkSortedUniqueDigests, and checkExtensions delegate to
	// the environ gates), so their len comparisons no longer
	// derive here and their rows are GONE, not retained. The
	// bounds still refuse at the same production entries through
	// the same drivers (TestStringBoundEdges,
	// TestSortedUniqueStringsEdges, TestDigestArrayBoundEdges,
	// TestExtensionKeyBoundEdges), and the delegation itself is
	// pinned structurally by TestCheckHelpersDelegateToEnviron
	// in internal/environ. A revived local comparison fails here
	// as unrostered.
	{id: `decode.go|checkOptionalString|checkStringBounds|raw|0`, driver: "TestStringBoundEdges", exempt: true, reason: "mechanism: transparent bound forwarder for cursor, next_cursor, and remediation; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueUUIDv7|len|index < len(values)|0`, exempt: true, reason: "plumbing: pairwise sortedness scan index, not a domain bound; element and count edges pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueUUIDv7|len|uint64(len(elements)) < minimumCount|0`, driver: "TestSortedUniqueUUIDv7Edges", exempt: true, reason: "mechanism: shared count floor; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueUUIDv7|len|uint64(len(elements)) > maximumCount|0`, driver: "TestSortedUniqueUUIDv7Edges", exempt: true, reason: "mechanism: shared count ceiling; values pinned at the caller rows"},
	{id: `decode.go|checkURI|len|length < N|0`, driver: "TestStringBoundEdges", exempt: true, reason: "mechanism: shared URI floor; the 1..2 range is unreachable behind the colon grammar and the values live at the contract_id row"},
	{id: `decode.go|checkURI|len|length > N|0`, driver: "TestStringBoundEdges", exempt: true, reason: "mechanism: shared URI ceiling; values pinned at the contract_id row"},
	{id: `decode.go|parseUint53Literal|len|index < len(literal)|0`, exempt: true, reason: "plumbing: digit-scan index, not a domain bound; value edges pinned by TestUint53RepresentabilityCeiling"},
	// bootstrap.go.
	{id: `bootstrap.go|DecideBootstrapStep|len|len(frame) == N|0`, driver: "TestBootstrapTerminalFailuresNeverDowngrade", reason: "empty attempt frame refuses before any envelope is read"},
	{id: `bootstrap.go|NextLowerMajor|len|index+N < len(supportedMajors)|0`, exempt: true, reason: "traversal: supported-major descent, not a domain bound; the order is pinned by TestNextLowerMajorBounds"},
	// manifest.go.
	{id: `manifest.go|DecodeManifest|checkStringBounds|"node_id"|0`, driver: "TestStringBoundEdges", reason: "node identifier 1..128"},
	{id: `manifest.go|DecodeManifest|checkSortedUniqueDigests|"redaction_policy_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "redaction policies count 1..64"},
	{id: `manifest.go|DecodeManifest|checkSortedUniqueDigests|"enrichment_profile_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "enrichment profiles count 0..256"},
	{id: `manifest.go|checkCapabilities|len|len(members) != len(capabilityOrder)|0`, driver: "TestManifestCapabilityReasonCoherence", reason: "eight-name capability map via ninth and missing witnesses"},
	{id: `manifest.go|checkCapability|checkStringBounds|reason|0`, driver: "TestStringBoundEdges", reason: "capability reason 1..128"},
	{id: `manifest.go|checkCapability|checkSortedUniqueDigests|"evidence_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "capability evidence count 0..64"},
	{id: `manifest.go|checkContractAssertions|len|index < len(encodings)|0`, exempt: true, reason: "plumbing: pairwise sortedness scan index, not a domain bound; count edges pinned at the caller rows"},
	{id: `manifest.go|checkContractAssertions|len|len(elements) < N|0`, driver: "TestContractAssertionBoundEdges", reason: "schemas floor 15"},
	{id: `manifest.go|checkContractAssertions|len|len(elements) > N|0`, driver: "TestContractAssertionBoundEdges", reason: "schemas ceiling 64"},
	{id: `manifest.go|checkContractAssertions|checkURI|identifier|0`, driver: "TestStringBoundEdges", reason: "contract identifier URI 1..512"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_enrichment_bytes"|0`, driver: "TestManifestLimitsBoundEdges", reason: "enrichment bytes 1..4194304"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_enrichment_events"|0`, driver: "TestManifestLimitsBoundEdges", reason: "enrichment events 1..5000"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_excerpt_bytes"|0`, driver: "TestManifestLimitsBoundEdges", reason: "excerpt bytes 0..4096"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_excerpt_count"|0`, driver: "TestManifestLimitsBoundEdges", reason: "excerpt count 0..20"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_frame_bytes"|0`, driver: "TestManifestLimitsBoundEdges", reason: "frame bytes 1..8388608"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_inventory_take"|0`, driver: "TestManifestLimitsBoundEdges", reason: "inventory take 1..1000"},
	{id: `manifest.go|checkLimits|checkUint53Bounds|"max_scan_instances"|0`, driver: "TestManifestLimitsBoundEdges", reason: "scan instances 1..65536"},
	{id: `manifest.go|checkManifestOperations|len|len(elements) != len(operationOrder)|0`, driver: "TestDecodeManifestRegistryRules", reason: "eleven-name operation registry via dropped and padded witnesses"},
	{id: `manifest.go|checkSupportedVersions|checkSortedUniqueStrings|raw|0`, driver: "TestSortedUniqueStringsEdges", reason: "supported versions element 5..64 count 1..16; the element floor is unreachable behind SemVer (no 4-character SemVer exists)"},
	// probe.go.
	{id: `probe.go|CheckProbeRequest|checkSortedUniqueStrings|"requested_capabilities"|0`, driver: "TestSortedUniqueStringsEdges", reason: "requested capabilities element 1..64 count 0..8 over an 8-closed registry; count ceiling equivalent behind the vocabulary"},
	{id: `probe.go|CheckProbeRequest|checkSortedUniqueStrings|"requested_environment_ids"|0`, driver: "TestSortedUniqueStringsEdges", reason: "requested environments element 1..64 count 0..64"},
	{id: `probe.go|checkFindings|len|len(elements) > N|0`, driver: "TestProbeResponseFindingBound", reason: "findings ceiling 4096"},
	{id: `probe.go|checkFinding|checkOptionalString|"remediation"|0`, driver: "TestStringBoundEdges", reason: "remediation 1..4096 or null"},
	{id: `probe.go|checkFinding|checkStringBounds|"code"|0`, driver: "TestStringBoundEdges", reason: "finding code 1..128"},
	{id: `probe.go|checkFinding|checkStringBounds|"message"|0`, driver: "TestStringBoundEdges", reason: "finding message 1..4096"},
	{id: `probe.go|checkNestedObjects|len|uint64(len(elements)) < minimumCount|0`, driver: "TestProbeEnvironmentsBoundEdges", exempt: true, reason: "mechanism: shared count floor; values live at the invisible call site and are pinned by the driver"},
	{id: `probe.go|checkNestedObjects|len|uint64(len(elements)) > maximumCount|0`, driver: "TestProbeEnvironmentsBoundEdges", exempt: true, reason: "mechanism: shared count ceiling; values live at the invisible call site and are pinned by the driver"},
	{id: `probe.go|checkNodeBuild|checkStringBounds|"node_id"|0`, driver: "TestStringBoundEdges", reason: "node build identifier 1..128"},
	// protocol.go.
	{id: `protocol.go|DecodeRequestFrame|checkUint53Bounds|"deadline_ms"|0`, driver: "TestUint53BoundEdges", reason: "deadline 1..3600000"},
	{id: `protocol.go|DecodeRequestFrame|len|len(frame) == N|0`, driver: "TestFrameBoundEdges", reason: "request empty frame"},
	{id: `protocol.go|DecodeRequestFrame|len|len(frame) > MaxFrameBytes|0`, driver: "TestFrameBoundEdges", reason: "request 8 MiB frame"},
	{id: `protocol.go|checkResponseIdentity|len|len(frame) == N|0`, driver: "TestFrameBoundEdges", reason: "response empty frame on both success and failure paths"},
	{id: `protocol.go|checkResponseIdentity|len|len(frame) > MaxFrameBytes|0`, driver: "TestFrameBoundEdges", reason: "response 8 MiB frame on both success and failure paths"},
	// query.go.
	{id: `query.go|CheckCursorReuse|len|stringLength(cursor) < N|0`, driver: "TestCheckCursorReuse", reason: "cursor floor 1 in characters"},
	{id: `query.go|CheckCursorReuse|len|stringLength(cursor) > N|0`, driver: "TestCheckCursorReuse", reason: "cursor ceiling 1024 in characters"},
	{id: `query.go|DecodeQuery|len|len(elements) < N|0`, driver: "TestQueryBatchCountEdges", reason: "operations floor 1"},
	{id: `query.go|DecodeQuery|len|len(elements) > N|0`, driver: "TestQueryBatchCountEdges", reason: "operations ceiling 64"},
	{id: `query.go|checkCaller|checkSortedUniqueStrings|"scopes"|0`, driver: "TestSortedUniqueStringsEdges", reason: "caller scopes element 1..64 count 1..5 over a 5-closed registry; count ceiling equivalent behind the vocabulary"},
	{id: `query.go|checkCaller|checkStringBounds|"authentication_subject"|0`, driver: "TestStringBoundEdges", reason: "authentication subject 1..512"},
	{id: `query.go|checkCaller|checkStringBounds|"caller_id"|0`, driver: "TestStringBoundEdges", reason: "caller identifier 1..256"},
	{id: `query.go|checkEnrichParameters|checkSortedUniqueStrings|"kinds"|0`, driver: "TestSortedUniqueStringsEdges", reason: "enrich kinds element 1..64 count 1..3 over a 3-closed vocabulary; count ceiling equivalent behind the vocabulary"},
	{id: `query.go|checkEnumSubset|checkSortedUniqueStrings|raw|0`, driver: "TestSortedUniqueStringsEdges", exempt: true, reason: "mechanism: transparent subset forwarder; values pinned at the four caller rows"},
	{id: `query.go|checkEnvironmentsParameters|checkSortedUniqueStrings|"authentication_status"|0`, driver: "TestSortedUniqueStringsEdges", reason: "authentication status element 1..32 count 0..4 over a 4-closed vocabulary; count ceiling equivalent behind the vocabulary and vocabulary pinned by TestDecodeQueryOperationValidatorsRefuse"},
	{id: `query.go|checkEnvironmentsParameters|checkSortedUniqueStrings|"environment_ids"|0`, driver: "TestSortedUniqueStringsEdges", reason: "environment identifiers element 1..64 count 0..64"},
	{id: `query.go|checkEnvironmentsParameters|checkSortedUniqueUUIDv7|"host_ids"|0`, driver: "TestSortedUniqueUUIDv7Edges", reason: "environments host identifiers count 0..256"},
	{id: `query.go|checkExecutePlanParameters|len|len(elements) > N|0`, driver: "TestDecodeQueryOperationValidatorsRefuse", reason: "confirmations ceiling 64"},
	{id: `query.go|checkFilters|checkEnumSubset|"freshness"|0`, driver: "TestSortedUniqueStringsEdges", reason: "freshness element 1..32 count 0..7 over a 7-closed vocabulary; count ceiling equivalent behind the vocabulary and vocabulary pinned by TestDecodeQueryOperationValidatorsRefuse"},
	{id: `query.go|checkFilters|checkEnumSubset|"kinds"|0`, driver: "TestSortedUniqueStringsEdges", reason: "kinds element 1..64 count 0..3 over a 3-closed vocabulary; count ceiling equivalent behind the vocabulary and vocabulary pinned by TestDecodeQueryOperationValidatorsRefuse"},
	{id: `query.go|checkFilters|checkEnumSubset|"management_states"|0`, driver: "TestSortedUniqueStringsEdges", reason: "management states element 1..32 count 0..3 over a 3-closed vocabulary; count ceiling equivalent behind the vocabulary and vocabulary pinned by TestDecodeQueryOperationValidatorsRefuse"},
	{id: `query.go|checkFilters|checkEnumSubset|"reachability"|0`, driver: "TestSortedUniqueStringsEdges", reason: "reachability element 1..32 count 0..4 over a 4-closed vocabulary; count ceiling equivalent behind the vocabulary and vocabulary pinned by TestDecodeQueryOperationValidatorsRefuse"},
	{id: `query.go|checkFilters|checkSortedUniqueStrings|"provider_ids"|0`, driver: "TestSortedUniqueStringsEdges", reason: "provider identifiers element 1..32 count 0..64"},
	{id: `query.go|checkFilters|checkSortedUniqueStrings|"states"|0`, driver: "TestSortedUniqueStringsEdges", reason: "filter states element 1..128 count 0..64"},
	{id: `query.go|checkFilters|checkSortedUniqueStrings|"warnings"|0`, driver: "TestSortedUniqueStringsEdges", reason: "warnings element 1..256 count 0..128"},
	{id: `query.go|checkFilters|checkSortedUniqueUUIDv7|"host_ids"|0`, driver: "TestSortedUniqueUUIDv7Edges", reason: "filter host identifiers count 0..256"},
	{id: `query.go|checkFilters|checkSortedUniqueUUIDv7|"workspace_ids"|0`, driver: "TestSortedUniqueUUIDv7Edges", reason: "workspace identifiers count 0..256"},
	{id: `query.go|checkFilters|checkUUIDv7DigestSubset|"lineage_anchors"|0`, driver: "TestUUIDv7DigestSubsetEdges", reason: "lineage anchors count 0..256 over the UUIDv7|digest union"},
	{id: `query.go|checkHostsParameters|checkSortedUniqueUUIDv7|"host_ids"|0`, driver: "TestSortedUniqueUUIDv7Edges", reason: "hosts identifiers count 0..256"},
	{id: `query.go|checkJobsParameters|checkSortedUniqueDigests|"profile_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "job profile identifiers count 0..256"},
	{id: `query.go|checkJobsParameters|checkSortedUniqueStrings|"states"|0`, driver: "TestSortedUniqueStringsEdges", reason: "job states element 1..32 count 0..7 over a 7-closed vocabulary; count ceiling equivalent behind the vocabulary and vocabulary pinned by TestDecodeQueryOperationValidatorsRefuse (review probes pwned/root/secret die there)"},
	{id: `query.go|checkJobsParameters|checkSortedUniqueUUIDv7|"job_ids"|0`, driver: "TestSortedUniqueUUIDv7Edges", reason: "job identifiers count 0..256"},
	{id: `query.go|checkPlansParameters|checkSortedUniqueDigests|"plan_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "plan identifiers count 0..256"},
	{id: `query.go|checkPlansParameters|checkSortedUniqueUUIDv7|"operation_ids"|0`, driver: "TestSortedUniqueUUIDv7Edges", reason: "plan operation identifiers count 0..256"},
	{id: `query.go|checkQueryProjection|checkSortedUniqueStrings|fieldsRaw|0`, driver: "TestDecodeQueryProjectionFields", reason: "projection fields element 1..64 count 0..128 with registry and sortedness pinned alongside; count ceiling equivalent behind the 27-field registry"},
	{id: `query.go|checkQuerySort|len|len(elements) > N|0`, driver: "TestQuerySortBoundEdges", reason: "sort tuple ceiling 8"},
	{id: `query.go|checkSupersedes|checkSortedUniqueDigests|raw|0`, driver: "TestDigestArrayBoundEdges", reason: "supersedes count 0..1024"},
	{id: `query.go|checkTagsParameter|len|len(elements) > N|0`, driver: "TestDecodeQueryOperationValidatorsRefuse", reason: "tags ceiling 256"},
	{id: `query.go|checkTitleParameter|checkStringBounds|"title"|0`, driver: "TestStringBoundEdges", reason: "annotation title 1..512"},
	{id: `query.go|checkUUIDv7DigestSubset|len|uint64(len(elements)) < minimumCount|0`, driver: "TestUUIDv7DigestSubsetEdges", exempt: true, reason: "mechanism: shared count floor; values pinned at the lineage_anchors row"},
	{id: `query.go|checkUUIDv7DigestSubset|len|uint64(len(elements)) > maximumCount|0`, driver: "TestUUIDv7DigestSubsetEdges", exempt: true, reason: "mechanism: shared count ceiling; values pinned at the lineage_anchors row"},
	// scan.go.
	{id: `scan.go|CheckScanRequest|checkOptionalString|"cursor"|0`, driver: "TestStringBoundEdges", reason: "scan cursor 1..4096 or null"},
	{id: `scan.go|CheckScanRequest|checkSortedUniqueDigests|"installation_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "installation identifiers count 1..256"},
	{id: `scan.go|CheckScanRequest|checkUint53Bounds|"max_instances"|0`, driver: "TestUint53BoundEdges", reason: "max instances 1..65536"},
	{id: `scan.go|CheckScanResponse|checkOptionalString|"next_cursor"|0`, driver: "TestStringBoundEdges", reason: "next cursor 1..4096 or null"},
	{id: `scan.go|CheckScanResponse|checkSortedUniqueDigests|"environment_observation_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "environment observations count 1..256"},
	{id: `scan.go|CheckScanResponse|checkSortedUniqueDigests|"native_observation_ids"|0`, driver: "TestDigestArrayBoundEdges", reason: "native observations count 0..65536; the 65537-element ceiling body is a stated bound"},
	{id: `scan.go|decodeJournalRecord|checkStringBounds|"key"|0`, driver: "TestJournalBoundEdges", reason: "journal key 1..512"},
}

// boundCensusCanaries are sites from the review traversal that
// the scanner must see. If any is absent from the derived set,
// the scanner is blind in exactly the direction the census
// exists to measure, and the census fails rather than reporting
// a short denominator as complete.
var boundCensusCanaries = []string{
	`manifest.go|DecodeManifest|checkStringBounds|"node_id"`,
	`query.go|checkTitleParameter|checkStringBounds|"title"`,
	`scan.go|CheckScanRequest|checkUint53Bounds|"max_instances"`,
	`query.go|checkFilters|checkEnumSubset|"kinds"`,
	`query.go|checkFilters|checkUUIDv7DigestSubset|"lineage_anchors"`,
	`decode.go|checkURI|len|length > N`,
	`manifest.go|checkContractAssertions|len|index < len(encodings)`,
	`decode.go|parseUint53Literal|len|index < len(literal)`,
	`decode.go|checkSortedUniqueUUIDv7|len|uint64(len(elements)) < minimumCount`,
	`query.go|CheckCursorReuse|len|stringLength(cursor) > N`,
	`protocol.go|checkResponseIdentity|len|len(frame) > MaxFrameBytes`,
}

// TestBoundGuardsAreCensused is the bound census: the derived
// production site multiset must equal the registration multiset.
// An unregistered site fails as unrostered; an orphaned
// registration fails as stale. Every non-exempt row must name a
// driver and every exempt row a rationale.
func TestBoundGuardsAreCensused(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("bound census: %v", err)
	}
	derived := scanBoundSiteIDs(t, directory)
	if len(derived) == 0 {
		t.Fatal("bound census derived zero sites; the scanner is broken, not the package")
	}
	derivedCounts := map[string]int{}
	for _, id := range derived {
		derivedCounts[id]++
	}
	for _, canary := range boundCensusCanaries {
		found := false
		for id := range derivedCounts {
			if strings.HasPrefix(id, canary) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("bound census blind to canary %q; the scanner misses the class it must roster", canary)
		}
	}
	registeredCounts := map[string]int{}
	seen := map[string]bool{}
	for _, row := range boundRegistrations {
		if seen[row.id] {
			t.Errorf("bound census registers %q twice", row.id)
		}
		seen[row.id] = true
		registeredCounts[row.id]++
		if row.driver == "" && !row.exempt {
			t.Errorf("bound census row %q names no driver and no exemption", row.id)
		}
		if row.exempt && row.reason == "" {
			t.Errorf("bound census exemption %q states no rationale", row.id)
		}
	}
	var unregistered []string
	for id, count := range derivedCounts {
		for i := registeredCounts[id]; i < count; i++ {
			unregistered = append(unregistered, id)
		}
	}
	var orphaned []string
	for id, count := range registeredCounts {
		for i := derivedCounts[id]; i < count; i++ {
			orphaned = append(orphaned, id)
		}
	}
	sort.Strings(unregistered)
	sort.Strings(orphaned)
	if len(unregistered) > 0 {
		t.Fatalf("production bound site(s) with no census row:\n  %s", strings.Join(unregistered, "\n  "))
	}
	if len(orphaned) > 0 {
		t.Fatalf("census row(s) naming no production site:\n  %s", strings.Join(orphaned, "\n  "))
	}
	driven, exempt := 0, 0
	for _, row := range boundRegistrations {
		if row.exempt {
			exempt++
		} else {
			driven++
		}
	}
	t.Logf("bound census: %d sites rostered (%d driven, %d exempt) across production", len(derived), driven, exempt)
}

// boundCensusDriverCount returns the number of non-exempt rows
// naming the given driver. Sibling tables with fixed tripwires
// assert against it so they cannot go stale when the roster
// grows.
func boundCensusDriverCount(driver string) int {
	count := 0
	for _, row := range boundRegistrations {
		if !row.exempt && row.driver == driver {
			count++
		}
	}
	return count
}

// TestBoundHelperIndirectionReports proves the indirection half
// of the bound census against synthetic files: a package-level
// alias and a function-local alias both report, while a direct
// helper call and the helper definition itself stay exempt. The
// vectors are synthetic on purpose: they prove the gate, not
// production.
func TestBoundHelperIndirectionReports(t *testing.T) {
	parse := func(source string) *ast.File {
		t.Helper()
		syntax, err := parser.ParseFile(token.NewFileSet(), "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return syntax
	}
	t.Run("package-level alias reports", func(t *testing.T) {
		syntax := parse("package probe\nvar zzprobeIndirect = checkStringBounds\nfunc f(m map[string]json.RawMessage) error {\nreturn zzprobeIndirect(m, \"zz_probe_member\", 1, 128)\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
	})
	t.Run("function-local alias reports", func(t *testing.T) {
		syntax := parse("package probe\nfunc f() {\nindirect := checkSortedUniqueStrings\n_ = indirect\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
	})
	t.Run("direct call stays exempt", func(t *testing.T) {
		syntax := parse("package probe\nfunc f(m map[string]json.RawMessage) error {\nreturn checkStringBounds(m, \"zz_probe_member\", 1, 128)\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 0 {
			t.Fatalf("violations = %v, want none for a direct call", violations)
		}
	})
	t.Run("helper definition stays exempt", func(t *testing.T) {
		syntax := parse("package probe\nfunc checkStringBounds(m map[string]json.RawMessage, name string, minimum, maximum int) error {\nreturn nil\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 0 {
			t.Fatalf("violations = %v, want none for the definition spelling", violations)
		}
	})
}

// TestBoundLenScanSeesNonIfContexts proves the context half of
// the bound census against a synthetic file: len guards in a
// return, a for condition, a switch condition, and an assignment
// all derive, while a boundless len use (a slice length passed
// to make) does not claim a guard. The vectors are synthetic on
// purpose: they prove the gate, not production.
func TestBoundLenScanSeesNonIfContexts(t *testing.T) {
	source := "package probe\nfunc zzReturnBound(value string) bool {\nreturn len(value) >= 1 && len(value) <= 512\n}\n" +
		"func zzForBound(values []string) int {\nn := 0\nfor i := 0; len(values) > 128 && i < 1; i++ {\nn++\n}\nreturn n\n}\n" +
		"func zzSwitchBound(values []string) int {\nswitch {\ncase len(values) == 0:\nreturn 0\ndefault:\nreturn 1\n}\n}\n" +
		"func zzAssignBound(values []string) int {\nover := len(values) > 64\nif over {\nreturn 1\n}\nreturn 0\n}\n" +
		"func zzBoundless(values []string) []string {\nreturn make([]string, 0, len(values))\n}\n"
	syntax, err := parser.ParseFile(token.NewFileSet(), "probe.go", []byte(source), 0)
	if err != nil {
		t.Fatalf("parse synthetic source: %v", err)
	}
	sites, violations := boundSitesInSyntax(syntax, "probe.go")
	if len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
	contains := func(want string) bool {
		for _, id := range sites {
			if strings.HasPrefix(id, want) {
				return true
			}
		}
		return false
	}
	for _, want := range []string{
		"probe.go|zzReturnBound|len|",
		"probe.go|zzForBound|len|",
		"probe.go|zzSwitchBound|len|",
		"probe.go|zzAssignBound|len|",
	} {
		if !contains(want) {
			t.Errorf("sites = %v, want a site under %q", sites, want)
		}
	}
	if contains("probe.go|zzBoundless|len|") {
		t.Errorf("sites = %v, want no guard for the boundless make length", sites)
	}
}

// TestBoundLenScanSeesLengthVariables proves the length-variable
// half of the bound census against a synthetic file: a guard over
// a variable bound from len(...) derives, a guard over a
// variable bound from stringLength(...) derives, a guard holding
// a direct stringLength(...) call derives, and a bound inside a
// package-level func literal derives under its variable name —
// while a comparison over a plain numeric parameter, a direct
// cap(...) comparison, and an element comparison over a slice
// built with a len capacity do not. The vectors are synthetic on
// purpose: they prove the gate, not production.
func TestBoundLenScanSeesLengthVariables(t *testing.T) {
	source := "package probe\nfunc zzLenVariable(value string) error {\nlength := len(value)\nif length < 1 || length > 128 { return nil }\nreturn nil\n}\n" +
		"func zzRuneVariable(value string) error {\nlength := stringLength(value)\nif length < 1 || length > 128 { return nil }\nreturn nil\n}\n" +
		"func zzRuneDirect(value string) error {\nif stringLength(value) > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzConverted(value string) error {\nn := uint64(len(value))\nif n > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzPlainParam(count int) error {\nif count < 1 || count > 128 { return nil }\nreturn nil\n}\n" +
		"func zzCapBound(values []string) error {\nif cap(values) > 128 { return nil }\nreturn nil\n}\n" +
		"func zzCapacitySlice(elements []string) error {\nvalues := make([]string, 0, len(elements))\nif values[0] == values[1] { return nil }\nreturn nil\n}\n" +
		"var zzProbeLiteralBound = func(observed string) error {\nif len(observed) > 128 { return nil }\nlength := len(observed)\nif length < 1 { return nil }\nreturn nil\n}\n"
	syntax, err := parser.ParseFile(token.NewFileSet(), "probe.go", []byte(source), 0)
	if err != nil {
		t.Fatalf("parse synthetic source: %v", err)
	}
	sites, violations := boundSitesInSyntax(syntax, "probe.go")
	if len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
	contains := func(want string) bool {
		for _, id := range sites {
			if strings.HasPrefix(id, want) {
				return true
			}
		}
		return false
	}
	for _, want := range []string{
		"probe.go|zzLenVariable|len|",
		"probe.go|zzRuneVariable|len|",
		"probe.go|zzRuneDirect|len|",
		"probe.go|zzConverted|len|",
		"probe.go|zzProbeLiteralBound|len|",
	} {
		if !contains(want) {
			t.Errorf("sites = %v, want a site under %q", sites, want)
		}
	}
	for _, want := range []string{
		"probe.go|zzPlainParam|len|",
		"probe.go|zzCapBound|len|",
		"probe.go|zzCapacitySlice|len|",
	} {
		if contains(want) {
			t.Errorf("sites = %v, want no site under %q (stated residue, shapes 7 and 9)", sites, want)
		}
	}
}
