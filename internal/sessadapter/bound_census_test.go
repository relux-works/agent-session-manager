package sessadapter

// This file is the bound census (review rev4 B8). It exists because a
// new bound through any of the six helpers — checkStringBounds,
// requireStringBounds, checkUint53Bounds, requireUint53Bounds,
// checkSortedUniqueStrings, checkSortedUniqueDigests — or a new
// explicit len(...) guard passes every behavioral test that never
// names it, and pinning checkStringBounds while requireStringBounds
// goes unmeasured reads as complete at 35 of 35 while half the
// class stays open.
//
// The mechanism note: requireStringBounds is a transparent wrapper
// over checkStringBounds (operations.go, declared 100 lines above
// the pinned sites) carrying 13 of the production call sites, and
// requireUint53Bounds the same over checkUint53Bounds. Both
// wrappers are rostered below as mechanism rows: they forward
// their min/max to the shared checker, so their values are pinned
// at the caller rows, not at the wrapper.
//
// Shape, modelled on TestClosedVocabularyTablesAreRegistered:
// scanBoundSiteIDs derives every bound site from production source
// (helper calls with their member, len guards in any statement
// context, DecodeFindings cap instantiations), and
// TestBoundGuardsAreCensused requires the derived multiset to
// equal boundRegistrations exactly. A new bound in one of the
// derived shapes below with no row fails the census; a removed
// bound orphans its row and fails too. A helper alias
// (`var f = requireStringBounds`) fails as an indirection
// violation rather than a missed site. An empty scan fails
// closed. Site identity is file|function|member,
// never the bound values: a moved bound (+1) keeps its identity
// and must redden its behavioral driver, which the mutant battery
// proves per row.
//
// Shape space (review rev6 G-B; the brief asked what shape comes
// after the one just closed, so the answer is written here, not
// left as silence). A bound in this package can only be carried
// by these AST shapes; the census derives the marked ones and
// bounds the rest:
//
//   1. A direct call to one of the six helpers — DERIVED with
//      the bounded member.
//   2. A bound helper through a func-value indirection (var,
//      :=, assignment, argument, return, struct field, method
//      value) — DERIVED as a census violation (the alias
//      itself fails rather than each hidden call).
//   3. A comparison holding a len(...) call in any statement
//      context (if, for, switch, return, assignment) — DERIVED.
//   4. A comparison over a length variable bound from len(...)
//      or stringLength(...) (`n := len(v); if n > 4096`) —
//      DERIVED (review rev6 shape; two live mechanism rows:
//      the shared character-count enforcements in
//      checkStringBounds and checkSortedUniqueStrings).
//   5. A comparison holding a direct stringLength(...) call —
//      DERIVED by the same rule (no live instance today; the
//      only production length measures flow through named
//      variables).
//   6. A bound inside a package-level var initializer,
//      including func literals — DERIVED under the variable
//      name (review rev6 shape; zero live rows today, proven by
//      the synthetic package-literal test below).
//   7. Cross-field relational gates (maxObjects >
//      maxTotalBytes and its sibling in DecodeResourceLimits) —
//      STATED: not edge-valued sites, pinned behaviourally by
//      TestDecodeResourceLimitsBounds instead.
//   8. The maxUint53 maxima — STATED: representability bounds
//      pinned once by TestUint53RepresentabilityCeiling rather
//      than per site.
//   9. A comparison over a decoded value or caller-supplied
//      number with no length source in the function (`value <
//      minimum` over a rawUint53 result, a plain int parameter)
//      — STATED BOUND, not derived. Live instance:
//      checkUint53Bounds (decode.go:269); the bound is enforced
//      at every caller row, and every caller row carries a
//      driver or a pinned exemption.
//  10. A direct cap(...) or utf8.RuneCountInString(...)
//      comparison outside stringLength — STATED BOUND, not
//      derived. No live instance: the only production
//      rune-count use is inside stringLength, whose callers are
//      all rostered rows.
//
// 6 of 10 shapes derived, 2 stated-pinned, 2 stated bounds. A new
// bound in shapes 1-6 with no row fails
// TestBoundGuardsAreCensused; shapes 7-8 are pinned once each by
// their named suite; shapes 9-10 are the named residue.
//
// Every non-exempt row names its driver: the committed test
// proving refusal at its edges (min-1/max+1) through the
// production entry. Admission at the accepted edge itself is
// proven only where the driver says so: 21 rows drive the
// refusal side only (review rev6 accept-direction battery), and
// that is recorded here as a bound rather than claimed per site.
// Exempt rows say why: equivalent (the bound is unreachable
// behind an upstream gate), stated-bound (a 65537-element body),
// mechanism (transparent forwarder or generic guard whose values
// live at the caller rows), or plumbing (no live rows: the frame
// byte-scanner this category once covered now delegates to
// environ.DecodeStrictObject, whose guards live in that package).
//
// Stated bounds of this census: shapes 7-10 above, plus the
// refusal-side-only drivers. _test.go files, which never ship,
// are outside the derivation, as are const initializers, which
// hold no calls and whose constant comparisons are vacuous.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
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
// seventh bound helper with no row here fails the census only if
// it is called; the call-shape derivation below names the helper
// per site, so an unrostered helper surfaces as unregistered
// sites.
var boundHelpers = map[string]bool{
	"checkStringBounds":        true,
	"requireStringBounds":      true,
	"checkUint53Bounds":        true,
	"requireUint53Bounds":      true,
	"checkSortedUniqueStrings": true,
	"checkSortedUniqueDigests": true,
}

// scanBoundSiteIDs derives every bound site identity from
// production source: file|function|helper|member|ordinal for the
// six helpers and DecodeFindings instantiations, and
// file|function|len|rendered|ordinal for len guards in any
// statement context (if, for, switch, return, assignment —
// review rev5/F2), including guards over length variables bound
// from len(...) or stringLength(...) and guards holding a direct
// stringLength(...) call (review rev6). Scopes are FuncDecl
// bodies and package-level var initializers under their variable
// name. Values are deliberately not part of the identity. A
// func-value indirection of any helper (`var f =
// requireStringBounds`, at package or function scope) is a
// census violation, not a missed site: refusals built through
// the alias hide from this derivation exactly the way
// axerror.New aliases hid from the constructor census (review
// rev4/N8), and the import-census shape applies here — the alias
// itself fails rather than each hidden call. The synthetic
// TestBoundHelperIndirectionReports,
// TestBoundLenScanSeesNonIfContexts, and
// TestBoundLenScanSeesLengthVariables prove the halves.
func scanBoundSiteIDs(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("bound census: %v", err)
	}
	var ids []string
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("bound census: %v", err)
		}
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			t.Fatalf("bound census: %v", err)
		}
		sites, violations := boundSitesInSyntax(syntax, name)
		for _, violation := range violations {
			t.Errorf("bound census: %s: %s", name, violation)
		}
		ids = append(ids, sites...)
	}
	if scanned == 0 {
		t.Fatal("bound census scanned zero production files; the scanner is broken, not the package")
	}
	return ids
}

// boundSitesInSyntax derives the bound site identities of one
// parsed production file and reports every helper indirection.
// Direct helper calls claim file|function|helper|member|ordinal;
// DecodeFindings instantiations claim their member; every len
// comparison in any statement context claims
// file|function|len|rendered|ordinal — including comparisons over
// length variables (idents bound from len(...) or
// stringLength(...)) and comparisons holding a direct
// stringLength(...) call (review rev6). Scopes are FuncDecl
// bodies and package-level var initializers under their variable
// name; the per-scope ordinal keeps every existing FuncDecl row
// identical. A use of a helper name that is neither a direct
// call nor one of the six function definitions is an indirection
// violation.
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
// scope body: helper calls and DecodeFindings instantiations by
// member, len comparisons by rendered shape. Length variables
// are collected first over the same body, so a guard written
// through a variable claims the same len site the direct
// spelling would.
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
		switch {
		case boundHelpers[ident.Name]:
			claim(name + "|" + scope + "|" + ident.Name + "|" + boundMemberOf(call, ident.Name))
		case ident.Name == "DecodeFindings":
			member := renderBoundExpr(call.Args[0])
			if index, ok := call.Args[0].(*ast.IndexExpr); ok {
				if literal, ok := index.Index.(*ast.BasicLit); ok && literal.Kind == token.STRING {
					member = literal.Value
				}
			}
			claim(name + "|" + scope + "|DecodeFindings|" + member)
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
// sources (shape 10 of the census header): neither is used as a
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
// review-rev6 plant shape.
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
// of the six bound helpers in one parsed file: any use of a
// helper name that is neither a direct call (the identifier is
// the Fun of its enclosing call) nor one of the six function
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
// string-literal member for the require wrappers, the indexed
// member name for direct checks, or the rendered expression when
// the call bounds a computed value (element, identifier,
// members[name]).
func boundMemberOf(call *ast.CallExpr, helper string) string {
	if len(call.Args) == 0 {
		return "noargs"
	}
	var first ast.Expr
	if helper == "requireStringBounds" || helper == "requireUint53Bounds" {
		if len(call.Args) < 2 {
			return "short"
		}
		first = call.Args[1]
	} else {
		first = call.Args[0]
	}
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

// hasLenCall reports whether the subtree calls len.
func hasLenCall(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(inner ast.Node) bool {
		call, ok := inner.(*ast.CallExpr)
		if ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "len" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// boundRegistrations rosters every derived site. driver names the
// committed test proving the edges at the production entry;
// exempt rows carry the rationale instead. Ordinals disambiguate
// repeated members in one function: checkRequestScalars carries
// four expected_target_native_session_id bounds (ordinal 0
// projection-plan, 1 read-back, 2 validate-nullable, 3
// resume-plan) and checkSuccessScalars three
// source_store_generation bounds (ordinal 0 inspect, 1
// capture-plan, 2 capture).
var boundRegistrations = []boundRegistration{
	// context.go: direct checks with existing drivers.
	{id: `context.go|DecodeCapturePlanItem|checkStringBounds|"native_item_key"|0`, driver: "TestStringBoundEdges", reason: "capture item key 1..512"},
	{id: `context.go|DecodeFinding|checkStringBounds|"code"|0`, driver: "TestStringBoundEdges", reason: "finding code 1..128"},
	{id: `context.go|DecodeFinding|checkStringBounds|"message"|0`, driver: "TestStringBoundEdges", reason: "finding message 1..4096"},
	{id: `context.go|DecodeFinding|checkStringBounds|"remediation"|0`, driver: "TestStringBoundEdges", reason: "finding remediation 1..4096"},
	{id: `context.go|DecodeSourceSelector|checkStringBounds|"native_session_id"|0`, driver: "TestStringBoundEdges", reason: "selector session id 1..512"},
	{id: `context.go|DecodeSourceSelector|checkStringBounds|"opaque_source_ref"|0`, driver: "TestStringBoundEdges", reason: "selector opaque ref 1..512"},
	// context.go: resource limits. The 0..65536 pair is pinned by
	// the existing bounds test; the 1..maxUint53 trio drives its
	// minima here and its maxima once at the shared ceiling.
	{id: `context.go|DecodeResourceLimits|checkUint53Bounds|"max_events"|0`, driver: "TestDecodeResourceLimitsBounds", reason: "event limit 0..65536"},
	{id: `context.go|DecodeResourceLimits|checkUint53Bounds|"max_target_resources"|0`, driver: "TestDecodeResourceLimitsBounds", reason: "target limit 0..65536"},
	{id: `context.go|DecodeResourceLimits|checkUint53Bounds|"max_objects"|0`, driver: "TestResourceLimitsMinimumEdges", reason: "object limit 1..maxUint53; maximum pinned by TestUint53RepresentabilityCeiling"},
	{id: `context.go|DecodeResourceLimits|checkUint53Bounds|"max_total_bytes"|0`, driver: "TestResourceLimitsMinimumEdges", reason: "total limit 1..maxUint53; maximum pinned by TestUint53RepresentabilityCeiling"},
	{id: `context.go|DecodeResourceLimits|checkUint53Bounds|"max_single_object_bytes"|0`, driver: "TestResourceLimitsMinimumEdges", reason: "single limit 1..maxUint53; maximum pinned by TestUint53RepresentabilityCeiling"},
	// context.go: sorted-unique handle names and the generic
	// findings guard behind the four cap instantiations.
	{id: `context.go|DecodeReadAuthority|checkSortedUniqueStrings|"root_handle_names"|0`, driver: "TestSortedUniqueBoundEdges", reason: "handle names element 1..128 count 1..128"},
	{id: `context.go|DecodeFindings|len|uint64(len(elements)) > maximum|0`, driver: "TestFindingsCapEdges", exempt: true, reason: "mechanism: generic guard; caps pinned at the four DecodeFindings instantiation rows below"},
	// decode.go: reverse-DNS key bounds and shared mechanisms.
	{id: `decode.go|checkExtensions|len|len(name) < N|0`, exempt: true, reason: "equivalent: no 2-character key satisfies the reverse-DNS grammar (minimum \"a.b\" is 3), so narrowing the floor changes no verdict; the ceiling and the refusal are pinned by TestExtensionKeyBoundEdges"},
	{id: `decode.go|checkExtensions|len|len(name) > N|0`, driver: "TestExtensionKeyBoundEdges", reason: "extension key maximum 253"},
	{id: `decode.go|checkSortedUniqueDigests|len|index < len(values)|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "plumbing: pairwise sortedness scan index, not a domain bound; element and count edges pinned at the caller rows"},
	// decode.go: the shared character-count enforcements behind
	// every string bound (review rev6 shape 4). Both compare a
	// length variable bound from stringLength, so the
	// length-variable derivation sees them; they are mechanism,
	// not sites — their values live at the caller rows, which
	// the named drivers pin at both refusal edges.
	{id: `decode.go|checkStringBounds|len|length < minimum|0`, driver: "TestStringBoundEdges", exempt: true, reason: "mechanism: shared minimum enforcement; values pinned at the caller rows"},
	{id: `decode.go|checkStringBounds|len|length > maximum|0`, driver: "TestStringBoundEdges", exempt: true, reason: "mechanism: shared maximum enforcement; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueStrings|len|length < minimumLength|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "mechanism: shared element-minimum enforcement; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueStrings|len|length > maximumLength|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "mechanism: shared element-maximum enforcement; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueDigests|len|uint64(len(elements)) < minimumCount|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "mechanism: shared count floor; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueDigests|len|uint64(len(elements)) > maximumCount|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "mechanism: shared count ceiling; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueStrings|len|index < len(values)|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "plumbing: pairwise sortedness scan index, not a domain bound; element and count edges pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueStrings|len|uint64(len(elements)) < minimumCount|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "mechanism: shared count floor; values pinned at the caller rows"},
	{id: `decode.go|checkSortedUniqueStrings|len|uint64(len(elements)) > maximumCount|0`, driver: "TestSortedUniqueBoundEdges", exempt: true, reason: "mechanism: shared count ceiling; values pinned at the caller rows"},
	{id: `decode.go|parseUint53Literal|len|index < len(literal)|0`, exempt: true, reason: "plumbing: digit-scan index, not a domain bound; value edges pinned by TestUint53RepresentabilityCeiling"},
	// manifest.go.
	{id: `manifest.go|DecodeManifest|checkSortedUniqueStrings|"platforms"|0`, exempt: true, reason: "equivalent: element 16 and count 4 unreachable behind scalar.ParsePlatform and the four-name closed set"},
	{id: `manifest.go|DecodeManifest|checkStringBounds|"display_name"|0`, driver: "TestDecodeManifestValueRules", reason: "display name 1..128"},
	{id: `manifest.go|DecodeManifest|checkStringBounds|"environment_version_range"|0`, driver: "TestStringBoundEdges", reason: "version range 1..256"},
	{id: `manifest.go|checkCapabilityRegistry|len|len(elements) != len(capabilityOrder)|0`, driver: "TestDecodeManifestRegistryMutations", reason: "fifteen-name registry length; zero pinned by TestRegistryLengthEmptyRefuses"},
	{id: `manifest.go|checkOperationRegistry|len|len(elements) != len(operationOrder)|0`, driver: "TestDecodeManifestRegistryMutations", reason: "fourteen-name registry length; zero pinned by TestRegistryLengthEmptyRefuses"},
	// operations.go request scalars: the requireStringBounds
	// wrapper class. The cursor minimum was pinned by the
	// existing witness suite only through nullability, never at
	// the 1..1024 edges; all rows below drive both edges.
	{id: `operations.go|checkDiscoverSources|len|uint64(len(elements)) > N|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", reason: "discover sources cap via too_many_sources witness"},
	{id: `operations.go|checkExcludedClasses|len|uint64(len(elements)) > N|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", reason: "excluded classes cap via ten_classes witness"},
	{id: `operations.go|checkRequestScalars|checkSortedUniqueDigests|"canonical_event_ids"|0`, exempt: true, reason: "stated bound: 65537 digests; count 0..65536"},
	{id: `operations.go|checkRequestScalars|checkSortedUniqueStrings|"forbid_reasons"|0`, driver: "TestSortedUniqueBoundEdges", reason: "forbid reasons element 1..128 count 0..128"},
	{id: `operations.go|checkRequestScalars|requireStringBounds|"cursor"|0`, driver: "TestRequestScalarBoundEdges", reason: "discover cursor 1..1024"},
	{id: `operations.go|checkRequestScalars|requireStringBounds|"expected_source_store_generation"|0`, driver: "TestRequestScalarBoundEdges", reason: "generation 1..512; minimum also witnessed by empty_generation"},
	{id: `operations.go|checkRequestScalars|requireStringBounds|"expected_target_native_session_id"|0`, driver: "TestRequestScalarBoundEdges", reason: "ordinal 0: projection-plan target id 1..512"},
	{id: `operations.go|checkRequestScalars|requireStringBounds|"expected_target_native_session_id"|1`, driver: "TestRequestScalarBoundEdges", reason: "ordinal 1: read-back target id 1..512"},
	{id: `operations.go|checkRequestScalars|requireStringBounds|"expected_target_native_session_id"|2`, driver: "TestRequestScalarBoundEdges", reason: "ordinal 2: validate nullable target id 1..512"},
	{id: `operations.go|checkRequestScalars|requireStringBounds|"expected_target_native_session_id"|3`, driver: "TestRequestScalarBoundEdges", reason: "ordinal 3: resume-plan target id 1..512"},
	{id: `operations.go|checkRequestScalars|requireUint53Bounds|"limit"|0`, driver: "TestRequestScalarBoundEdges", reason: "discover limit 1..65536"},
	{id: `operations.go|checkRequestScalars|requireUint53Bounds|"max_items"|0`, driver: "TestRequestScalarBoundEdges", reason: "max items 1..65536"},
	{id: `operations.go|checkRequestScalars|requireUint53Bounds|"max_total_bytes"|0`, driver: "TestRequestScalarBoundEdges", reason: "max bytes minimum 1; maximum maxUint53 pinned by TestUint53RepresentabilityCeiling"},
	{id: `operations.go|checkRequiredDispositions|len|len(elements) == N|0`, driver: "TestArrayBoundEdges", reason: "disposition list lower edge"},
	{id: `operations.go|checkRequiredDispositions|len|len(elements) > N|0`, driver: "TestArrayBoundEdges", reason: "disposition list cap 7"},
	{id: `operations.go|checkRequiredDispositions|len|len(members) == N|0`, driver: "TestRequiredDispositionsEmptyMapRefuses", reason: "disposition map lower edge"},
	// operations.go success scalars: the second requireStringBounds
	// front. source_store_generation ordinals: 0 inspect, 1
	// capture-plan, 2 capture.
	{id: `operations.go|checkSuccessScalars|DecodeFindings|"ambiguities"|0`, driver: "TestFindingsCapEdges", reason: "inspect ambiguities cap 1024"},
	{id: `operations.go|checkSuccessScalars|DecodeFindings|"findings"|0`, driver: "TestFindingsCapEdges", reason: "projection findings cap 4096"},
	{id: `operations.go|checkSuccessScalars|checkSortedUniqueDigests|"canonical_event_candidate_ids"|0`, exempt: true, reason: "stated bound: 65537 digests; count 0..65536"},
	{id: `operations.go|checkSuccessScalars|checkSortedUniqueDigests|"raw_reference_ids"|0`, exempt: true, reason: "stated bound: 65537 digests; count 0..65536"},
	{id: `operations.go|checkSuccessScalars|checkSortedUniqueDigests|"required_source_object_ids"|0`, exempt: true, reason: "stated bound: 65537 digests; count 0..65536"},
	{id: `operations.go|checkSuccessScalars|checkSortedUniqueStrings|"created_resource_keys"|0`, driver: "TestSortedUniqueBoundEdges", reason: "resource keys element 1..512; count 65536 is a stated bound (65537 keys)"},
	{id: `operations.go|checkSuccessScalars|checkSortedUniqueStrings|"environment_names"|0`, driver: "TestSortedUniqueBoundEdges", reason: "environment names element 1..256 count 0..128"},
	{id: `operations.go|checkSuccessScalars|checkSortedUniqueStrings|"parsed_head_ids"|0`, driver: "TestSortedUniqueBoundEdges", reason: "head ids element 1..512 count 0..1024"},
	{id: `operations.go|checkSuccessScalars|checkStringBounds|element|0`, driver: "TestStringBoundEdges", reason: "resume argv word 1..4096"},
	{id: `operations.go|checkSuccessScalars|len|len(argv) == N|0`, driver: "TestArrayBoundEdges", reason: "argv lower edge"},
	{id: `operations.go|checkSuccessScalars|len|len(argv) > N|0`, driver: "TestArrayBoundEdges", reason: "argv cap 128"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"cwd_relative"|0`, driver: "TestSuccessScalarBoundEdges", reason: "cwd relative 1..4096"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"next_cursor"|0`, driver: "TestSuccessScalarBoundEdges", reason: "discover next cursor 1..1024"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"observed_target_native_session_id"|0`, driver: "TestSuccessScalarBoundEdges", reason: "observed target id 1..512"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"source_store_generation"|0`, driver: "TestSuccessScalarBoundEdges", reason: "ordinal 0: inspect generation 1..512"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"source_store_generation"|1`, driver: "TestSuccessScalarBoundEdges", reason: "ordinal 1: capture-plan generation 1..512"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"source_store_generation"|2`, driver: "TestSuccessScalarBoundEdges", reason: "ordinal 2: capture generation 1..512"},
	{id: `operations.go|checkSuccessScalars|requireStringBounds|"target_native_session_id"|0`, driver: "TestSuccessScalarBoundEdges", reason: "resume target id 1..512"},
	{id: `operations.go|checkSuccessScalars|requireUint53Bounds|"candidate_count"|0`, driver: "TestSuccessScalarBoundEdges", reason: "candidate count 0..65536"},
	{id: `operations.go|checkSuccessScalars|requireUint53Bounds|"item_count"|0`, driver: "TestSuccessScalarBoundEdges", reason: "item count 0..65536"},
	{id: `operations.go|checkSuccessScalars|requireUint53Bounds|"parsed_event_count"|0`, driver: "TestSuccessScalarBoundEdges", reason: "parsed event count minimum 0; maximum maxUint53 pinned by TestUint53RepresentabilityCeiling"},
	{id: `operations.go|checkValidateResult|DecodeFindings|"findings"|0`, driver: "TestFindingsCapEdges", reason: "validate findings cap 4096"},
	// The two transparent wrappers over the shared checkers.
	{id: `operations.go|requireStringBounds|checkStringBounds|members[name]|0`, driver: "TestRequestScalarBoundEdges", exempt: true, reason: "mechanism: transparent bound forwarder; values pinned at the 13 caller rows"},
	{id: `operations.go|requireUint53Bounds|checkUint53Bounds|members[name]|0`, driver: "TestRequestScalarBoundEdges", exempt: true, reason: "mechanism: transparent bound forwarder; values pinned at the 6 caller rows"},
	// probe.go.
	{id: `probe.go|DecodeDoctorResult|DecodeFindings|"findings"|0`, driver: "TestFindingsCapEdges", reason: "doctor findings cap 4096"},
	{id: `probe.go|DecodeDoctorResult|checkUint53Bounds|"registry_sequence"|0`, driver: "TestDoctorSequenceMinimum", reason: "registry sequence minimum 0; maximum maxUint53 pinned by TestUint53RepresentabilityCeiling"},
	{id: `probe.go|DecodeProbe|checkSortedUniqueStrings|"warnings"|0`, driver: "TestSortedUniqueBoundEdges", reason: "warnings count 0..1024; element 2048 pinned by TestDecodeProbeClosedRules"},
	{id: `probe.go|decodeCapabilities|len|len(members) != len(capabilityOrder)|0`, driver: "TestDecodeProbeClosedRules", reason: "fifteen-entry capability map; zero pinned by TestRegistryLengthEmptyRefuses"},
	{id: `probe.go|decodeCapabilityValue|checkStringBounds|"detail"|0`, driver: "TestStringBoundEdges", reason: "capability detail 0..2048"},
	// protocol.go frame bounds and trim plumbing.
	{id: `protocol.go|CheckFailureEnvelope|len|len(frame) == N|0`, driver: "TestEmptyFramesRefuse", reason: "failure empty frame"},
	{id: `protocol.go|CheckFailureEnvelope|len|len(frame) > MaxFrameBytes|0`, driver: "TestFrameBoundEdges", reason: "failure 8 MiB frame"},
	{id: `protocol.go|CheckSuccessEnvelope|len|len(frame) == N|0`, driver: "TestEmptyFramesRefuse", reason: "success empty frame"},
	{id: `protocol.go|CheckSuccessEnvelope|len|len(frame) > MaxFrameBytes|0`, driver: "TestFrameBoundEdges", reason: "success 8 MiB frame"},
	{id: `protocol.go|DecodeRequestFrame|len|len(frame) == N|0`, driver: "TestEmptyFramesRefuse", reason: "request empty frame"},
	{id: `protocol.go|DecodeRequestFrame|len|len(frame) > MaxFrameBytes|0`, driver: "TestFrameBoundEdges", reason: "request 8 MiB frame"},
	{id: `protocol.go|bytesTrimSpace|len|len(trimmed) == N|0`, exempt: true, reason: "plumbing: empty trim returns its input; no refusal, no bound"},
	// tuple.go.
	{id: `tuple.go|CheckTupleAdmission|len|len(entry.Strategies) != N|0`, driver: "TestCheckTupleAdmissionRefusals", reason: "source strategies exact shape"},
	{id: `tuple.go|DecodeTuple|checkStringBounds|"environment_version"|0`, driver: "TestStringBoundEdges", reason: "tuple version 1..128"},
	{id: `tuple.go|DecodeTupleEntry|checkStringBounds|"revocation_reason"|0`, driver: "TestStringBoundEdges", reason: "revocation reason 1..4096"},
	{id: `tuple.go|DecodeTupleEntry|checkUint53Bounds|"entry_sequence"|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", reason: "entry sequence above zero via entry_sequence=0 witness"},
	{id: `tuple.go|checkContractsShape|checkStringBounds|identifier|0`, driver: "TestStringBoundEdges", reason: "contract identifier 1..256"},
	{id: `tuple.go|checkContractsShape|len|len(elements) == N|0`, driver: "TestArrayBoundEdges", reason: "contracts lower edge"},
	{id: `tuple.go|checkContractsShape|len|len(elements) > N|0`, driver: "TestArrayBoundEdges", reason: "contracts cap 64"},
	{id: `tuple.go|checkContractsShape|len|len(members) != N|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", reason: "contract row shape via three-member_row witness"},
	{id: `tuple.go|checkContractsShape|len|len(versions) == N|0`, driver: "TestArrayBoundEdges", reason: "versions lower edge"},
	{id: `tuple.go|checkContractsShape|len|len(versions) > N|0`, driver: "TestArrayBoundEdges", reason: "versions cap 32"},
	{id: `tuple.go|decodeFidelityLimits|checkStringBounds|"affected_class"|0`, driver: "TestStringBoundEdges", reason: "fidelity class 1..128"},
	{id: `tuple.go|decodeFidelityLimits|checkStringBounds|"code"|0`, driver: "TestStringBoundEdges", reason: "fidelity code 1..128"},
	{id: `tuple.go|decodeFidelityLimits|checkStringBounds|"detail"|0`, driver: "TestStringBoundEdges", reason: "fidelity detail 1..4096"},
	{id: `tuple.go|decodeFidelityLimits|len|index < len(limits)|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", exempt: true, reason: "plumbing: pairwise sortedness scan index, not a domain bound; count edge pinned at the caller row"},
	{id: `tuple.go|decodeFidelityLimits|len|uint64(len(elements)) > N|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", reason: "fidelity cap via 1025_rows witness"},
	{id: `tuple.go|decodeFixtureEvidence|checkStringBounds|"suite_revision"|0`, driver: "TestStringBoundEdges", reason: "suite revision 1..128"},
	{id: `tuple.go|decodeFixtureEvidence|checkUint53Bounds|"fixture_count"|0`, driver: "TestEveryArmWitnessRefusesAtTheProductionEntry", reason: "fixture count above zero via fixture_count=0 witness"},
	{id: `tuple.go|decodeSmokeEvidence|checkStringBounds|"native_cli_family"|0`, driver: "TestStringBoundEdges", reason: "CLI family 1..128"},
	{id: `tuple.go|decodeStrategies|len|index < len(strategies)|0`, driver: "TestStrategiesEmptyRefuses", exempt: true, reason: "plumbing: pairwise sortedness scan index, not a domain bound; lower edge pinned at the caller row"},
	{id: `tuple.go|decodeStrategies|len|len(elements) == N|0`, driver: "TestStrategiesEmptyRefuses", reason: "strategies lower edge"},
}

// boundCensusCanaries are survivor sites from the review traversal
// that the scanner must see. If any is absent from the derived
// set, the scanner is blind in exactly the direction the census
// exists to measure, and the census fails rather than reporting
// a short denominator as complete.
var boundCensusCanaries = []string{
	`operations.go|checkRequestScalars|requireStringBounds|"cursor"`,
	`operations.go|checkSuccessScalars|requireStringBounds|"cwd_relative"`,
	`operations.go|checkRequestScalars|requireStringBounds|"expected_target_native_session_id"`,
	`operations.go|checkSuccessScalars|requireUint53Bounds|"candidate_count"`,
	`probe.go|DecodeProbe|checkSortedUniqueStrings|"warnings"`,
	`decode.go|checkExtensions|len|len(name) > N`,
	`tuple.go|decodeStrategies|len|index < len(strategies)`,
	`decode.go|parseUint53Literal|len|index < len(literal)`,
	`decode.go|checkStringBounds|len|length < minimum`,
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

// TestBoundHelperIndirectionReports proves the indirection half
// of the bound census against synthetic files: a package-level
// `var f = requireStringBounds` and a function-local alias both
// report, while a direct helper call and the helper definition
// itself stay exempt. The vectors are synthetic on purpose: they
// prove the gate, not production. This is the regression test
// for review rev5/F2 (repeat of rev4/N8): the alias plant passed
// the whole suite because the scanner matched `call.Fun` only.
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
		syntax := parse("package probe\nvar zzprobeIndirect = requireStringBounds\nfunc f(m map[string]json.RawMessage) error {\nreturn zzprobeIndirect(m, \"zz_probe_member\", 1, 4096)\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
	})
	t.Run("function-local alias reports", func(t *testing.T) {
		syntax := parse("package probe\nfunc f() {\nindirect := requireStringBounds\n_ = indirect\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 1 {
			t.Fatalf("violations = %v, want the single indirection report", violations)
		}
	})
	t.Run("direct call stays exempt", func(t *testing.T) {
		syntax := parse("package probe\nfunc f(m map[string]json.RawMessage) error {\nreturn requireStringBounds(m, \"zz_probe_member\", 1, 4096)\n}\n")
		if violations := boundHelperIndirections(syntax); len(violations) != 0 {
			t.Fatalf("violations = %v, want none for a direct call", violations)
		}
	})
	t.Run("helper definition stays exempt", func(t *testing.T) {
		syntax := parse("package probe\nfunc requireStringBounds(m map[string]json.RawMessage, name string, minimum, maximum int) error {\nreturn nil\n}\n")
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
// purpose: they prove the gate, not production. This is the
// regression test for the second and third plants of review
// rev5/F2, which hid in return and for positions the if-only
// walk never visited.
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
// cap(...) or utf8.RuneCountInString(...) comparison, and an
// element comparison over a slice built with a len capacity do
// not. The vectors are synthetic on purpose: they prove the
// gate, not production. This is the regression test for the
// rev6 G-B plants (`length := len(value); if length > 4096` and
// the package-literal bound passed the whole suite unrostered),
// and the negatives pin the stated residue: shapes 9 and 10 of
// the census header.
func TestBoundLenScanSeesLengthVariables(t *testing.T) {
	source := "package probe\nfunc zzLenVariable(value string) error {\nlength := len(value)\nif length < 1 || length > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzRuneVariable(value string) error {\nlength := stringLength(value)\nif length < 1 || length > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzRuneDirect(value string) error {\nif stringLength(value) > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzConverted(value string) error {\nn := uint64(len(value))\nif n > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzPlainParam(count int) error {\nif count < 1 || count > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzCapBound(values []string) error {\nif cap(values) > 4096 { return nil }\nreturn nil\n}\n" +
		"func zzCapacitySlice(elements []string) error {\nvalues := make([]string, 0, len(elements))\nif values[0] == values[1] { return nil }\nreturn nil\n}\n" +
		"var zzProbeLiteralBound = func(observed string) error {\nif len(observed) > 4096 { return nil }\nlength := len(observed)\nif length < 1 { return nil }\nreturn nil\n}\n"
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
			t.Errorf("sites = %v, want no site under %q (stated residue, shapes 9-10)", sites, want)
		}
	}
}

// boundCensusDriverCount returns the number of non-exempt rows
// naming the given driver. Sibling tables with fixed tripwires
// (TestStringBoundEdges) assert against it so they cannot go
// stale when the roster grows.
func boundCensusDriverCount(driver string) int {
	count := 0
	for _, row := range boundRegistrations {
		if !row.exempt && row.driver == driver {
			count++
		}
	}
	return count
}

// requestStringBoundRow is one requireStringBounds site on the
// request path: operation, member, and inclusive character
// bounds, driven through CheckRequestBody.
type requestStringBoundRow struct {
	operation Operation
	member    string
	min       int
	max       int
}

// TestRequestScalarBoundEdges drives every requireStringBounds
// and finite requireUint53Bounds request site at its edges
// through CheckRequestBody: min-1 refuses, min admits, max
// admits, max+1 refuses. A mutant widening any bound by one
// admits its max+1 row and reddens here.
func TestRequestScalarBoundEdges(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	srows := []requestStringBoundRow{
		{OpDiscover, "cursor", 1, 1024},
		{OpSnapshotProof, "expected_source_store_generation", 1, 512},
		{OpProjectionPlan, "expected_target_native_session_id", 1, 512},
		{OpReadBack, "expected_target_native_session_id", 1, 512},
		{OpValidate, "expected_target_native_session_id", 1, 512},
		{OpResumePlan, "expected_target_native_session_id", 1, 512},
	}
	for _, row := range srows {
		row := row
		name := string(row.operation) + "/" + row.member
		t.Run(name, func(t *testing.T) {
			drive := func(literal string) error {
				body := fixtureRequestBody(t, row.operation, manifestDigest)
				_, err := CheckRequestBody(row.operation, mutateMember(t, body, row.member, literal))
				return err
			}
			if err := drive(quotedRepeat(row.min - 1)); err == nil {
				t.Fatalf("%s admitted length %d below the %d minimum", name, row.min-1, row.min)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
			if err := drive(quotedRepeat(row.min)); err != nil {
				t.Fatalf("%s refused length %d at the minimum: %v", name, row.min, err)
			}
			if err := drive(quotedRepeat(row.max)); err != nil {
				t.Fatalf("%s refused length %d at the maximum: %v", name, row.max, err)
			}
			if err := drive(quotedRepeat(row.max + 1)); err == nil {
				t.Fatalf("%s admitted length %d past the %d maximum", name, row.max+1, row.max)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
		})
	}
	uints := []struct {
		operation Operation
		member    string
		min       uint64
		max       uint64
		finiteMax bool
	}{
		{OpDiscover, "limit", 1, 65536, true},
		{OpCapturePlan, "max_items", 1, 65536, true},
		{OpCapturePlan, "max_total_bytes", 1, maxUint53, false},
	}
	for _, row := range uints {
		row := row
		name := string(row.operation) + "/" + row.member
		t.Run(name, func(t *testing.T) {
			drive := func(literal string) error {
				body := fixtureRequestBody(t, row.operation, manifestDigest)
				_, err := CheckRequestBody(row.operation, mutateMember(t, body, row.member, literal))
				return err
			}
			below := strconv.FormatUint(row.min-1, 10)
			if row.min == 0 {
				below = "-1"
			}
			if err := drive(below); err == nil {
				t.Fatalf("%s admitted %s below the %d minimum", name, below, row.min)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
			if err := drive(strconv.FormatUint(row.min, 10)); err != nil {
				t.Fatalf("%s refused %d at the minimum: %v", name, row.min, err)
			}
			if !row.finiteMax {
				return
			}
			if err := drive(strconv.FormatUint(row.max, 10)); err != nil {
				t.Fatalf("%s refused %d at the maximum: %v", name, row.max, err)
			}
			if err := drive(strconv.FormatUint(row.max+1, 10)); err == nil {
				t.Fatalf("%s admitted %d past the %d maximum", name, row.max+1, row.max)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
		})
	}
}

// successFactsFor builds the SuccessFacts one success body is
// checked against: the decoded request context with staged mode,
// target-1 identity, and source_read direction.
func successFactsFor(t *testing.T, operation Operation, manifestDigest string) (CallContext, SuccessFacts) {
	t.Helper()
	request := fixtureRequestBody(t, operation, manifestDigest)
	decoded, err := CheckRequestBody(operation, request)
	if err != nil {
		t.Fatalf("CheckRequestBody(%q): %v", operation, err)
	}
	facts := SuccessFacts{Context: decoded, ValidateMode: "staged", ResumeTargetID: "target-1", DoctorDirection: DirectionSourceRead}
	return decoded, facts
}

// TestSuccessScalarBoundEdges drives every requireStringBounds
// and finite requireUint53Bounds success site at its edges
// through CheckSuccessBody. The OpResumePlan target row aligns
// argv with the admit values: the identity gate runs after the
// bound, so only the admit values must also satisfy it.
func TestSuccessScalarBoundEdges(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	srows := []struct {
		operation Operation
		member    string
		min       int
		max       int
	}{
		{OpDiscover, "next_cursor", 1, 1024},
		{OpInspect, "source_store_generation", 1, 512},
		{OpCapturePlan, "source_store_generation", 1, 512},
		{OpCapture, "source_store_generation", 1, 512},
		{OpReadBack, "observed_target_native_session_id", 1, 512},
		{OpResumePlan, "target_native_session_id", 1, 512},
		{OpResumePlan, "cwd_relative", 1, 4096},
	}
	for _, row := range srows {
		row := row
		name := string(row.operation) + "/" + row.member
		t.Run(name, func(t *testing.T) {
			_, facts := successFactsFor(t, row.operation, manifestDigest)
			drive := func(t *testing.T, literal string) error {
				request := fixtureRequestBody(t, row.operation, manifestDigest)
				decoded, err := CheckRequestBody(row.operation, request)
				if err != nil {
					t.Fatalf("CheckRequestBody: %v", err)
				}
				local := facts
				local.Context = decoded
				base := fixtureSuccessBody(t, row.operation, requestContextOf(t, request), manifestDigest)
				mutated := mutateMember(t, base, row.member, literal)
				if row.operation == OpDiscover && literal != `null` {
					mutated = mutateMember(t, mutated, "partial", `true`)
				}
				if row.operation == OpResumePlan && row.member == "target_native_session_id" && (literal == quotedRepeat(row.min) || literal == quotedRepeat(row.max)) {
					identity := strings.Trim(literal, `"`)
					mutated = mutateMember(t, mutated, "argv", `["ax","open",`+strconv.Quote(identity)+`]`)
					local.ResumeTargetID = identity
				}
				return CheckSuccessBody(row.operation, mutated, local)
			}
			if err := drive(t, quotedRepeat(row.min-1)); err == nil {
				t.Fatalf("%s admitted length %d below the %d minimum", name, row.min-1, row.min)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
			if err := drive(t, quotedRepeat(row.min)); err != nil {
				t.Fatalf("%s refused length %d at the minimum: %v", name, row.min, err)
			}
			if err := drive(t, quotedRepeat(row.max)); err != nil {
				t.Fatalf("%s refused length %d at the maximum: %v", name, row.max, err)
			}
			if err := drive(t, quotedRepeat(row.max+1)); err == nil {
				t.Fatalf("%s admitted length %d past the %d maximum", name, row.max+1, row.max)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
		})
	}
	uints := []struct {
		operation Operation
		member    string
		finiteMax bool
	}{
		{OpCapturePlan, "candidate_count", true},
		{OpCapture, "item_count", true},
		{OpReadBack, "parsed_event_count", false},
	}
	for _, row := range uints {
		row := row
		name := string(row.operation) + "/" + row.member
		t.Run(name, func(t *testing.T) {
			_, facts := successFactsFor(t, row.operation, manifestDigest)
			drive := func(t *testing.T, literal string) error {
				request := fixtureRequestBody(t, row.operation, manifestDigest)
				decoded, err := CheckRequestBody(row.operation, request)
				if err != nil {
					t.Fatalf("CheckRequestBody: %v", err)
				}
				local := facts
				local.Context = decoded
				base := fixtureSuccessBody(t, row.operation, requestContextOf(t, request), manifestDigest)
				return CheckSuccessBody(row.operation, mutateMember(t, base, row.member, literal), local)
			}
			if err := drive(t, `-1`); err == nil {
				t.Fatalf("%s admitted -1 below the zero minimum", name)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
			if err := drive(t, `0`); err != nil {
				t.Fatalf("%s refused 0 at the minimum: %v", name, err)
			}
			if !row.finiteMax {
				return
			}
			if err := drive(t, `65536`); err != nil {
				t.Fatalf("%s refused 65536 at the maximum: %v", name, err)
			}
			if err := drive(t, `65537`); err == nil {
				t.Fatalf("%s admitted 65537 past the 65536 maximum", name)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, "outside its bound")
			}
		})
	}
}

// sortedStrings builds n sorted unique JSON string literals with
// the given printf format and width.
func sortedStrings(format string, n int) string {
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, strconv.Quote(fmt.Sprintf(format, i)))
	}
	return `[` + strings.Join(parts, ",") + `]`
}

// TestSortedUniqueBoundEdges drives every checkSortedUniqueStrings
// call site at its element and count edges through the production
// entry: the empty element refuses, a max+1 element refuses, max+1
// elements refuse, and the maxima admit. A mutant widening any
// element or count bound by one admits its row and reddens here.
func TestSortedUniqueBoundEdges(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	t.Run("forbid reasons", func(t *testing.T) {
		drive := func(value string) error {
			body := fixtureRequestBody(t, OpProjectionPlan, manifestDigest)
			_, err := CheckRequestBody(OpProjectionPlan, mutateMember(t, body, "forbid_reasons", value))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty forbid reason")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "forbid_reasons", "sorted-unique-string[1..128][0..128]")
		}
		if err := drive(`[` + quotedRepeat(129) + `]`); err == nil {
			t.Fatal("admitted a 129-character forbid reason")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "forbid_reasons", "sorted-unique-string[1..128][0..128]")
		}
		if err := drive(`[` + quotedRepeat(128) + `]`); err != nil {
			t.Fatalf("refused a 128-character reason at the maximum: %v", err)
		}
		if err := drive(sortedStrings("r-%03d", 129)); err == nil {
			t.Fatal("admitted 129 forbid reasons past the 128 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "forbid_reasons", "sorted-unique-string[1..128][0..128]")
		}
		if err := drive(sortedStrings("r-%03d", 128)); err != nil {
			t.Fatalf("refused 128 reasons at the maximum: %v", err)
		}
		if err := drive(`[]`); err != nil {
			t.Fatalf("refused an empty reason list at the zero minimum: %v", err)
		}
	})
	t.Run("created resource keys", func(t *testing.T) {
		_, facts := successFactsFor(t, OpProject, manifestDigest)
		drive := func(t *testing.T, value string) error {
			request := fixtureRequestBody(t, OpProject, manifestDigest)
			decoded, err := CheckRequestBody(OpProject, request)
			if err != nil {
				t.Fatalf("CheckRequestBody: %v", err)
			}
			local := facts
			local.Context = decoded
			base := fixtureSuccessBody(t, OpProject, requestContextOf(t, request), manifestDigest)
			return CheckSuccessBody(OpProject, mutateMember(t, base, "created_resource_keys", value), local)
		}
		if err := drive(t, `[""]`); err == nil {
			t.Fatal("admitted an empty resource key")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "created_resource_keys", "sorted unique string[1..512][0..65536]")
		}
		if err := drive(t, `[`+quotedRepeat(513)+`]`); err == nil {
			t.Fatal("admitted a 513-character resource key")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "created_resource_keys", "sorted unique string[1..512][0..65536]")
		}
		if err := drive(t, `[`+quotedRepeat(512)+`]`); err != nil {
			t.Fatalf("refused a 512-character key at the maximum: %v", err)
		}
	})
	t.Run("parsed head ids", func(t *testing.T) {
		_, facts := successFactsFor(t, OpReadBack, manifestDigest)
		drive := func(t *testing.T, value string) error {
			request := fixtureRequestBody(t, OpReadBack, manifestDigest)
			decoded, err := CheckRequestBody(OpReadBack, request)
			if err != nil {
				t.Fatalf("CheckRequestBody: %v", err)
			}
			local := facts
			local.Context = decoded
			base := fixtureSuccessBody(t, OpReadBack, requestContextOf(t, request), manifestDigest)
			return CheckSuccessBody(OpReadBack, mutateMember(t, base, "parsed_head_ids", value), local)
		}
		if err := drive(t, `[""]`); err == nil {
			t.Fatal("admitted an empty head id")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "parsed_head_ids", "sorted unique string[1..512][0..1024]")
		}
		if err := drive(t, `[`+quotedRepeat(513)+`]`); err == nil {
			t.Fatal("admitted a 513-character head id")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "parsed_head_ids", "sorted unique string[1..512][0..1024]")
		}
		if err := drive(t, sortedStrings("h-%05d", 1025)); err == nil {
			t.Fatal("admitted 1025 head ids past the 1024 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "parsed_head_ids", "sorted unique string[1..512][0..1024]")
		}
		if err := drive(t, sortedStrings("h-%05d", 1024)); err != nil {
			t.Fatalf("refused 1024 head ids at the maximum: %v", err)
		}
	})
	t.Run("environment names", func(t *testing.T) {
		_, facts := successFactsFor(t, OpResumePlan, manifestDigest)
		drive := func(t *testing.T, value string) error {
			request := fixtureRequestBody(t, OpResumePlan, manifestDigest)
			decoded, err := CheckRequestBody(OpResumePlan, request)
			if err != nil {
				t.Fatalf("CheckRequestBody: %v", err)
			}
			local := facts
			local.Context = decoded
			base := fixtureSuccessBody(t, OpResumePlan, requestContextOf(t, request), manifestDigest)
			return CheckSuccessBody(OpResumePlan, mutateMember(t, base, "environment_names", value), local)
		}
		if err := drive(t, `[""]`); err == nil {
			t.Fatal("admitted an empty environment name")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "environment_names", "sorted unique string[1..256][0..128]")
		}
		if err := drive(t, `[`+quotedRepeat(257)+`]`); err == nil {
			t.Fatal("admitted a 257-character environment name")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "environment_names", "sorted unique string[1..256][0..128]")
		}
		if err := drive(t, sortedStrings("e-%05d", 129)); err == nil {
			t.Fatal("admitted 129 environment names past the 128 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "environment_names", "sorted unique string[1..256][0..128]")
		}
		if err := drive(t, sortedStrings("e-%05d", 128)); err != nil {
			t.Fatalf("refused 128 names at the maximum: %v", err)
		}
	})
	t.Run("probe warnings count", func(t *testing.T) {
		manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
		if err != nil {
			t.Fatalf("DecodeManifest: %v", err)
		}
		digest := ManifestDigest(manifest).String()
		drive := func(value string) error {
			_, err := DecodeProbe(mutateMember(t, []byte(fixtureProbeJSON(digest)), "warnings", value))
			return err
		}
		if err := drive(sortedStrings("w-%05d", 1025)); err == nil {
			t.Fatal("admitted 1025 warnings past the 1024 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "warnings", "sorted unique string[0..2048][0..1024]")
		}
		if err := drive(sortedStrings("w-%05d", 1024)); err != nil {
			t.Fatalf("refused 1024 warnings at the maximum: %v", err)
		}
	})
	t.Run("root handle names", func(t *testing.T) {
		drive := func(value string) error {
			_, err := DecodeReadAuthority(mutateMember(t, []byte(fixtureReadAuthorityJSON()), "root_handle_names", value))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty handle name")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "root_handle_names", "sorted unique string[1..128][1..128]")
		}
		if err := drive(`[` + quotedRepeat(129) + `]`); err == nil {
			t.Fatal("admitted a 129-character handle name")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "root_handle_names", "sorted unique string[1..128][1..128]")
		}
		if err := drive(sortedStrings("h-%03d", 129)); err == nil {
			t.Fatal("admitted 129 handle names past the 128 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "root_handle_names", "sorted unique string[1..128][1..128]")
		}
		if err := drive(sortedStrings("h-%03d", 128)); err != nil {
			t.Fatalf("refused 128 handle names at the maximum: %v", err)
		}
	})
}

// validFindingLiteral is one closed AdapterFinding used to fill
// count-bound arrays without tripping element gates.
const validFindingLiteral = `{"severity":"info","code":"c","message":"m","remediation":null,"extensions":{}}`

// findingArray builds n valid findings as a JSON array.
func findingArray(n int) string {
	return `[` + strings.Join(repeatLiteral(validFindingLiteral, n), ",") + `]`
}

func repeatLiteral(literal string, n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, literal)
	}
	return out
}

// TestFindingsCapEdges drives every DecodeFindings cap
// instantiation at its edge through the production entry: the
// cap admits, cap+1 refuses naming the count bound. The four
// instantiations share one generic guard; each value is pinned
// where it is passed.
func TestFindingsCapEdges(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	t.Run("inspect ambiguities 1024", func(t *testing.T) {
		_, facts := successFactsFor(t, OpInspect, manifestDigest)
		drive := func(t *testing.T, value string) error {
			request := fixtureRequestBody(t, OpInspect, manifestDigest)
			decoded, err := CheckRequestBody(OpInspect, request)
			if err != nil {
				t.Fatalf("CheckRequestBody: %v", err)
			}
			local := facts
			local.Context = decoded
			base := fixtureSuccessBody(t, OpInspect, requestContextOf(t, request), manifestDigest)
			return CheckSuccessBody(OpInspect, mutateMember(t, base, "ambiguities", value), local)
		}
		if err := drive(t, findingArray(1025)); err == nil {
			t.Fatal("admitted 1025 ambiguities past the 1024 cap")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "findings", "exceed the count bound")
		}
		if err := drive(t, findingArray(1024)); err != nil {
			t.Fatalf("refused 1024 ambiguities at the cap: %v", err)
		}
	})
	for _, operation := range []Operation{OpProjectionPlan, OpValidate, OpDoctor} {
		operation := operation
		t.Run(string(operation)+" findings 4096", func(t *testing.T) {
			_, facts := successFactsFor(t, operation, manifestDigest)
			if operation == OpValidate {
				facts.ValidateMode = "staged"
			}
			if operation == OpDoctor {
				facts.DoctorDirection = DirectionSourceRead
			}
			drive := func(t *testing.T, value string) error {
				request := fixtureRequestBody(t, operation, manifestDigest)
				decoded, err := CheckRequestBody(operation, request)
				if err != nil {
					t.Fatalf("CheckRequestBody: %v", err)
				}
				local := facts
				local.Context = decoded
				base := fixtureSuccessBody(t, operation, requestContextOf(t, request), manifestDigest)
				return CheckSuccessBody(operation, mutateMember(t, base, "findings", value), local)
			}
			if err := drive(t, findingArray(4097)); err == nil {
				t.Fatalf("%s admitted 4097 findings past the 4096 cap", operation)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", "findings", "exceed the count bound")
			}
			if err := drive(t, findingArray(4096)); err != nil {
				t.Fatalf("%s refused 4096 findings at the cap: %v", operation, err)
			}
		})
	}
}

// TestExtensionKeyBoundEdges drives the reverse-DNS extension key
// bounds at both edges through DecodeManifest: 2 refuses, 3
// admits, 253 admits, 254 refuses.
func TestExtensionKeyBoundEdges(t *testing.T) {
	valid253 := "a" + strings.Repeat(".b", 126)
	if len(valid253) != 253 {
		t.Fatalf("scaffold key = %d chars, want 253", len(valid253))
	}
	drive := func(value string) error {
		_, err := DecodeManifest(mutateMember(t, []byte(fixtureManifestJSON()), "extensions", value))
		return err
	}
	if err := drive(`{"ab":{}}`); err == nil {
		t.Fatal("admitted a 2-character extension key")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "extensions", "reverse-DNS keyed")
	}
	if err := drive(`{"a.b":{}}`); err != nil {
		t.Fatalf("refused a 3-character key at the minimum: %v", err)
	}
	if err := drive(`{` + strconv.Quote(valid253) + `: {}}`); err != nil {
		t.Fatalf("refused a 253-character key at the maximum: %v", err)
	}
	if err := drive(`{` + strconv.Quote(valid253+"c") + `: {}}`); err == nil {
		t.Fatal("admitted a 254-character extension key")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "extensions", "reverse-DNS keyed")
	}
}

// TestResourceLimitsMinimumEdges drives the three 1..maxUint53
// resource minima: zero refuses naming the above-zero arm.
func TestResourceLimitsMinimumEdges(t *testing.T) {
	base := `{"max_objects":10,"max_total_bytes":1000,"max_single_object_bytes":100,"max_events":5,"max_target_resources":5}`
	if _, err := DecodeResourceLimits(json.RawMessage(base)); err != nil {
		t.Fatalf("DecodeResourceLimits baseline: %v", err)
	}
	for _, member := range []string{"max_objects", "max_total_bytes", "max_single_object_bytes"} {
		member := member
		t.Run(member, func(t *testing.T) {
			var members map[string]json.RawMessage
			if err := json.Unmarshal([]byte(base), &members); err != nil {
				t.Fatalf("scaffold: %v", err)
			}
			members[member] = json.RawMessage(`0`)
			rebuilt, err := json.Marshal(members)
			if err != nil {
				t.Fatalf("scaffold: %v", err)
			}
			_, err = DecodeResourceLimits(rebuilt)
			if err == nil {
				t.Fatalf("admitted %s=0 below the one minimum", member)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "above zero")
			}
		})
	}
}

// TestUint53RepresentabilityCeiling pins the shared uint53
// ceiling once: 2^53-1 admits, 2^53 refuses. Every maxUint53
// maximum in the roster points here instead of repeating the
// literal per site.
func TestUint53RepresentabilityCeiling(t *testing.T) {
	if _, ok := rawUint53(json.RawMessage(`9007199254740991`)); !ok {
		t.Fatal("rawUint53 refused 2^53-1 at the ceiling")
	}
	if _, ok := rawUint53(json.RawMessage(`9007199254740992`)); ok {
		t.Fatal("rawUint53 admitted 2^53 past the ceiling")
	}
	if _, ok := checkUint53Bounds(json.RawMessage(`9007199254740992`), 0, maxUint53); ok {
		t.Fatal("checkUint53Bounds admitted 2^53 past the ceiling")
	}
}

// TestDoctorSequenceMinimum drives the registry sequence lower
// edge: -1 refuses, 0 admits.
func TestDoctorSequenceMinimum(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpDoctor, manifestDigest)
	decoded, err := CheckRequestBody(OpDoctor, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	drive := func(value string) error {
		base := fixtureSuccessBody(t, OpDoctor, requestContextOf(t, request), manifestDigest)
		_, err := DecodeDoctorResult(mutateMember(t, base, "registry_sequence", value), decoded, DirectionSourceRead)
		return err
	}
	if err := drive(`-1`); err == nil {
		t.Fatal("admitted registry_sequence=-1 below the zero minimum")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "registry_sequence", "not a uint53")
	}
	if err := drive(`0`); err != nil {
		t.Fatalf("refused registry_sequence=0 at the minimum: %v", err)
	}
}

// TestEmptyFramesRefuse drives the three empty-frame guards: an
// empty request, success, or failure frame is a protocol error,
// never an absent result.
func TestEmptyFramesRefuse(t *testing.T) {
	if _, err := DecodeRequestFrame([]byte{}); err == nil {
		t.Fatal("DecodeRequestFrame admitted an empty frame")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "frame is empty")
	}
	want := wantRequest()
	if _, err := CheckSuccessEnvelope([]byte{}, want); err == nil {
		t.Fatal("CheckSuccessEnvelope admitted an empty frame")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "frame is empty")
	}
	if _, err := CheckFailureEnvelope([]byte{}, want); err == nil {
		t.Fatal("CheckFailureEnvelope admitted an empty frame")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "frame is empty")
	}
}

// TestStrategiesEmptyRefuses drives the strategy-array lower
// edge: an entry with no strategies is refused, never a default
// permission.
func TestStrategiesEmptyRefuses(t *testing.T) {
	entry := mutateMember(t, []byte(fixtureEntryJSON(DirectionTargetWrite)), "strategies", `[]`)
	if _, err := DecodeTupleEntry(entry); err == nil {
		t.Fatal("DecodeTupleEntry admitted an empty strategy list")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "strategies", "are empty")
	}
}

// TestRegistryLengthEmptyRefuses drives the registry-length
// gates at zero: an empty operation, capability, or capability
// map is refused, never a vacuous registry.
func TestRegistryLengthEmptyRefuses(t *testing.T) {
	if _, err := DecodeManifest(mutateMember(t, []byte(fixtureManifestJSON()), "operations", `[]`)); err == nil {
		t.Fatal("DecodeManifest admitted an empty operation registry")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "operations", "fourteen-name registry")
	}
	if _, err := DecodeManifest(mutateMember(t, []byte(fixtureManifestJSON()), "capability_names", `[]`)); err == nil {
		t.Fatal("DecodeManifest admitted an empty capability registry")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capability_names", "fifteen-name registry")
	}
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	digest := ManifestDigest(manifest).String()
	if _, err := DecodeProbe(mutateMember(t, []byte(fixtureProbeJSON(digest)), "capabilities", `{}`)); err == nil {
		t.Fatal("DecodeProbe admitted an empty capability map")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capabilities", "exactly fifteen")
	}
}

// TestRegistryElementEmptyRefuses drives the registry-element
// gates at zero: a manifest registry naming "" refuses, never
// an invocable surface.
func TestRegistryElementEmptyRefuses(t *testing.T) {
	operations := make([]string, 0, len(operationOrder))
	for _, operation := range operationOrder {
		operations = append(operations, `"`+string(operation)+`"`)
	}
	operations[0] = `""`
	capabilities := make([]string, 0, len(capabilityOrder))
	for _, name := range capabilityOrder {
		capabilities = append(capabilities, `"`+name+`"`)
	}
	capabilities[0] = `""`
	if _, err := DecodeManifest(mutateMember(t, []byte(fixtureManifestJSON()), "operations", `[`+strings.Join(operations, ",")+`]`)); err == nil {
		t.Fatal("DecodeManifest admitted an empty operation name")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "operations", "fourteen-name registry")
	}
	if _, err := DecodeManifest(mutateMember(t, []byte(fixtureManifestJSON()), "capability_names", `[`+strings.Join(capabilities, ",")+`]`)); err == nil {
		t.Fatal("DecodeManifest admitted an empty capability name")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capability_names", "fifteen-name registry")
	}
}

// TestRequiredDispositionsEmptyMapRefuses drives the
// disposition-map lower edge: a projection request with no
// class is refused.
func TestRequiredDispositionsEmptyMapRefuses(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	body := fixtureRequestBody(t, OpProjectionPlan, manifestDigest)
	if _, err := CheckRequestBody(OpProjectionPlan, mutateMember(t, body, "required_dispositions", `{}`)); err == nil {
		t.Fatal("CheckRequestBody admitted an empty disposition map")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "required_dispositions", "no class")
	}
}
