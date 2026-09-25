package cloneplanning

import (
	"go/ast"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// This file pins the record-rule single ownership structurally,
// the regression answer to review finding
// effects-reason-set-coupling-missing (CR rev4): PlanTargetEffects
// re-validated disposition and reason vocabulary by hand and
// skipped the reason-set coupling the owner enforces. The Section
// 13.14.2 record rules now have one owner,
// internal/clonefidelity, reached through the
// ValidateDispositionRow seam:
//
//   - the census derives every exported entry whose parameters
//     carry a disposition, reason set, mapping, or record from the
//     real parameter types and asserts the set is exactly
//     {PlanTargetEffects};
//   - the reachability pin proves PlanTargetEffects, PlanItem
//     (which self-checks every emitted mapping), and PlanSession
//     (which inherits the check per item) all reach the owner
//     seam on the static call graph;
//   - the no-fork guards prove the package contains no second
//     disposition/reason rule implementation: no calls to the
//     owner's vocabulary predicates and no switch on a
//     Disposition in the validation closures.
//
// Behavior itself is proven by the rule x entry table in
// session_test.go, which drives violating inputs through the
// production entry and asserts the owner's literal codes. Control
// plants (a new disposition-carrying entry, a removed PlanItem
// self-check, a re-added vocabulary call, a disposition switch in
// the closure) redden these tests; logs ride the evidence tar.

// A parameter carries a disposition row when its transitive type
// closure contains a struct pairing a disposition member
// (Disposition or ExpectedDispos) with a reason-set member
// (Reasons or ReasonCodes): exactly what the owner's row
// validator consumes. A lone reason set without a disposition
// sibling is not a validatable row — the seam needs both — so it
// does not census. In particular ClassifyItem's event evidence
// (SourceEvidence.ReasonCodes, validated by the landed decoder
// and ignored by the classifier) is excluded by this rule, and
// the exclusion is measured: the census pins the evidence shape
// and the ignore test below proves adversarial evidence reasons
// never change the item. Resolution and protection facts are
// likewise not rows: they are closed-vocabulary classification
// inputs with their own gates.
func structCarriesDispositionRow(typ reflect.Type) bool {
	hasDisposition := false
	hasReasons := false
	for i := 0; i < typ.NumField(); i++ {
		switch typ.Field(i).Name {
		case "Disposition", "ExpectedDispos":
			hasDisposition = true
		case "Reasons", "ReasonCodes":
			hasReasons = true
		}
	}
	return hasDisposition && hasReasons
}

// reflectCarriesDispositionRow reports whether a disposition row
// is reachable from the type through fields, elements, keys, and
// pointees. Open any-typed members do not count: an any carries
// no NAMED row, and members the entry ignores (non-fact payload
// keys) can never reach a decision by type — the all-kinds sweep
// proves adversarial payload text cannot change the item, and
// the closed-fact guard pins ClassifyItem to the two fact keys.
func reflectCarriesDispositionRow(typ reflect.Type, visited map[reflect.Type]bool) bool {
	if typ == nil {
		return false
	}
	switch typ.Kind() {
	case reflect.Pointer:
		return reflectCarriesDispositionRow(typ.Elem(), visited)
	case reflect.Slice, reflect.Array:
		return reflectCarriesDispositionRow(typ.Elem(), visited)
	case reflect.Map:
		return reflectCarriesDispositionRow(typ.Key(), visited) || reflectCarriesDispositionRow(typ.Elem(), visited)
	case reflect.Struct:
		if structCarriesDispositionRow(typ) {
			return true
		}
		if visited[typ] {
			return false
		}
		visited[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			if reflectCarriesDispositionRow(typ.Field(i).Type, visited) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// reflectEntryCarriesDispositionRow reports whether any parameter
// of the function value carries a disposition row.
func reflectEntryCarriesDispositionRow(value any) bool {
	typ := reflect.TypeOf(value)
	for i := 0; i < typ.NumIn(); i++ {
		if reflectCarriesDispositionRow(typ.In(i), map[reflect.Type]bool{}) {
			return true
		}
	}
	return false
}

// structFieldNames lists the field names of one struct type.
func structFieldNames(value any) []string {
	typ := reflect.TypeOf(value)
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		names = append(names, typ.Field(i).Name)
	}
	sort.Strings(names)
	return names
}

func TestDispositionEntryCensus(t *testing.T) {
	// The census derives from the real parameter types over the
	// pinned exported-entry list: a new exported entry breaks
	// loudly here (count pin) and in the UTF-8 census (AST set
	// linkage), so no entry slips past unclassified.
	if len(exportedEntryFuncs) != 8 {
		t.Fatalf("exported entry list carries %d entries, want 8", len(exportedEntryFuncs))
	}
	// The list linkage: the AST-derived exported set equals the
	// reflection list, so a new exported entry breaks loudly
	// here instead of slipping past the census unclassified.
	listed := map[string]bool{}
	for _, entry := range exportedEntryFuncs {
		listed[entry.name] = true
	}
	for _, entry := range inspectEntryCensus(t) {
		if !listed[entry.name] {
			t.Fatalf("exported %s is missing from the reflection cross-check list", entry.name)
		}
		delete(listed, entry.name)
	}
	for name := range listed {
		t.Fatalf("reflection list names %s, which the AST census does not export", name)
	}
	var carriers []string
	for _, entry := range exportedEntryFuncs {
		if reflectEntryCarriesDispositionRow(entry.value) {
			carriers = append(carriers, entry.name)
		}
	}
	sort.Strings(carriers)
	if len(carriers) != 1 || carriers[0] != "PlanTargetEffects" {
		t.Fatalf("disposition-row entries %q, want exactly [PlanTargetEffects]", carriers)
	}
	// The non-carrier pins prove the negative is not vacuous:
	// the neighboring input structs exist with their known
	// members and pair no disposition with a reason set.
	pins := []struct {
		name  string
		value any
		known []string
	}{
		{"Item", Item{}, []string{"Block", "Kind", "Protection", "Resolution"}},
		{"VisibleInput", VisibleInput{}, []string{"EventIDs", "Kind", "Ordinal", "Texts"}},
	}
	for _, pin := range pins {
		fields := structFieldNames(pin.value)
		if strings.Join(fields, ",") != strings.Join(pin.known, ",") {
			t.Fatalf("%s fields %q, want %q", pin.name, fields, pin.known)
		}
		if structCarriesDispositionRow(reflect.TypeOf(pin.value)) {
			t.Fatalf("%s carries a disposition row", pin.name)
		}
	}
	// Mapping is the carrier: the census above is not vacuous.
	mappingFields := structFieldNames(Mapping{})
	if strings.Join(mappingFields, ",") != "Disposition,Reasons" {
		t.Fatalf("Mapping fields %q, want [Disposition Reasons]", mappingFields)
	}
	// The measured exclusion: event evidence carries a reason
	// set with no disposition sibling, so it is not a row and
	// ClassifyItem stays outside the census. If evidence ever
	// gains a disposition member this pin reddens and the census
	// must be revisited.
	evidenceFields := structFieldNames(clonebundle.SourceEvidence{})
	hasReasons := false
	for _, field := range evidenceFields {
		if field == "ReasonCodes" {
			hasReasons = true
		}
		if field == "Disposition" || field == "ExpectedDispos" {
			t.Fatalf("SourceEvidence gained disposition member %q; the census exclusion must be revisited", field)
		}
	}
	if !hasReasons {
		t.Fatalf("SourceEvidence carries no ReasonCodes; the exclusion pin is vacuous")
	}
}

// callEdges describes one function's outgoing calls: local callee
// names and external package-path-qualified calls.
type callEdges struct {
	local    []string
	external []string
}

// inspectCallGraph resolves the imports of each production file
// and records every call edge per function. A dot-import of the
// owner package fails closed: a bare ValidateDispositionRow call
// would otherwise hide from the seam pin.
func inspectCallGraph(t *testing.T) map[string]callEdges {
	t.Helper()
	graph := map[string]callEdges{}
	for _, file := range productionFiles(t) {
		localName := map[string]string{}
		for _, imported := range file.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if imported.Name != nil && imported.Name.Name == "." {
				if path == "github.com/relux-works/agent-session-manager/internal/clonefidelity" {
					t.Fatalf("dot-import of the owner package hides the seam pin")
				}
				continue
			}
			name := ""
			if imported.Name == nil {
				name = path[strings.LastIndex(path, "/")+1:]
			} else {
				name = imported.Name.Name
			}
			localName[name] = path
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			var edges callEdges
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					edges.local = append(edges.local, fun.Name)
				case *ast.SelectorExpr:
					if ident, ok := fun.X.(*ast.Ident); ok {
						if path, known := localName[ident.Name]; known {
							edges.external = append(edges.external, path+"."+fun.Sel.Name)
						}
					}
				}
				return true
			})
			graph[fn.Name.Name] = edges
		}
	}
	return graph
}

// reachesSeam reports whether the entry reaches the owner seam on
// the static call graph, following local calls transitively.
func reachesSeam(entry string, graph map[string]callEdges, seam string) bool {
	visited := map[string]bool{}
	queue := []string{entry}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if visited[name] {
			continue
		}
		visited[name] = true
		edges, ok := graph[name]
		if !ok {
			continue
		}
		for _, external := range edges.external {
			if external == seam {
				return true
			}
		}
		queue = append(queue, edges.local...)
	}
	return false
}

func TestClassifyItemIgnoresEvidenceReasons(t *testing.T) {
	// The behavioral half of the census exclusion: event
	// evidence reasons are validated by the landed decoder and
	// ignored by the classifier, which bridges kind plus closed
	// facts only. Adversarial evidence (bogus codes, unsorted
	// and duplicated sets, over-count sets, empty and invalid
	// UTF-8 members) across every valid event item neither
	// refuses nor changes the item. Hand-built events: the
	// sealed builder cannot carry malformed evidence at all, so
	// only the direct struct path exercises these members.
	many := make([]string, 129)
	for i := range many {
		many[i] = "bogus_reason"
	}
	adversarial := [][]string{
		nil,
		{},
		{"bogus_reason"},
		{"operator_policy", "bogus_reason"},
		{"unsafe_pending_action", "operator_policy"},
		{"operator_policy", "operator_policy"},
		{""},
		{"\xff\xfe"},
		many,
	}
	cells := 0
	for _, want := range eventSweepItems() {
		payload := map[string]any{}
		if want.Resolution != "" {
			payload["resolution"] = want.Resolution
		}
		if want.Protection != "" {
			payload["protection"] = want.Protection
		}
		for i, reasons := range adversarial {
			event := clonebundle.CanonicalEvent{
				Kind:     want.Kind,
				Payload:  payload,
				Evidence: clonebundle.SourceEvidence{ReasonCodes: reasons},
			}
			got, err := ClassifyItem(event)
			label := want.Kind + "/evidence"
			if err != nil {
				t.Fatalf("evidence[%d] %s refused: %v", i, label, err)
			}
			if got != want {
				t.Fatalf("evidence[%d] %s = %+v, want %+v", i, label, got, want)
			}
			cells++
		}
	}
	if cells == 0 {
		t.Fatalf("no evidence cells swept")
	}
	t.Logf("classify evidence-ignore: %d cells", cells)
}

const ownerRecordSeam = "github.com/relux-works/agent-session-manager/internal/clonefidelity.ValidateDispositionRow"

func TestDelegationReachesOwner(t *testing.T) {
	// Every mapping-touching entry reaches the owner's record
	// validator: PlanTargetEffects validates caller-supplied
	// mappings, PlanItem self-checks every emitted mapping, and
	// PlanSession inherits the check per item through PlanItem.
	// Removing any edge (the N-planitem-owner-bypass plant
	// removes PlanItem's) reddens here.
	graph := inspectCallGraph(t)
	for _, entry := range []string{"PlanTargetEffects", "PlanItem", "PlanSession"} {
		if !reachesSeam(entry, graph, ownerRecordSeam) {
			t.Fatalf("%s does not reach the owner seam %s", entry, ownerRecordSeam)
		}
	}
}

func TestNoDispositionVocabularyFork(t *testing.T) {
	// Package-wide pin: no production code calls the owner's
	// disposition or reason vocabulary predicates. All
	// disposition/reason membership flows through the
	// ValidateDispositionRow seam; re-adding a direct predicate
	// call (a second implementation of a subset) reddens here.
	// Strategy and event-kind vocabularies are different axes
	// with their own delegations and are unaffected.
	graph := inspectCallGraph(t)
	forked := []string{}
	for fn, edges := range graph {
		for _, external := range edges.external {
			if external == "github.com/relux-works/agent-session-manager/internal/clonefidelity.ValidDisposition" ||
				external == "github.com/relux-works/agent-session-manager/internal/clonefidelity.ValidReasonCode" {
				forked = append(forked, fn+":"+external)
			}
		}
	}
	sort.Strings(forked)
	if len(forked) != 0 {
		t.Fatalf("disposition/reason vocabulary calls outside the owner seam: %q", forked)
	}
}

// closureFuncs returns the entry's transitive local callee
// closure, including the entry itself.
func closureFuncs(entry string, graph map[string]callEdges) map[string]bool {
	closure := map[string]bool{}
	queue := []string{entry}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if closure[name] {
			continue
		}
		closure[name] = true
		queue = append(queue, graph[name].local...)
	}
	return closure
}

// switchOnDisposition reports whether a switch tag reads a
// Disposition selector.
func switchOnDisposition(tag ast.Expr) bool {
	found := false
	ast.Inspect(tag, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok && sel.Sel.Name == "Disposition" {
			found = true
			return false
		}
		return true
	})
	return found
}

func TestNoDispositionSwitchInValidationClosure(t *testing.T) {
	// Validation-closure pin: no switch on a Disposition
	// decides admission on the path that validates
	// caller-supplied (PlanTargetEffects) or emitted (PlanItem)
	// mappings. PlanSession's counting switch is outside this
	// closure by construction: it derives counts from
	// owner-validated emitted values (derivation, not
	// admission), and the 975-cell oracle plus the count tests
	// pin its behavior, so any admission planted there fails
	// behaviorally.
	graph := inspectCallGraph(t)
	closure := map[string]bool{}
	for _, entry := range []string{"PlanTargetEffects", "PlanItem"} {
		for name := range closureFuncs(entry, graph) {
			closure[name] = true
		}
	}
	switched := []string{}
	for _, file := range productionFiles(t) {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !closure[fn.Name.Name] {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				stmt, ok := node.(*ast.SwitchStmt)
				if !ok || stmt.Tag == nil {
					return true
				}
				if switchOnDisposition(stmt.Tag) {
					switched = append(switched, fn.Name.Name)
				}
				return true
			})
		}
	}
	sort.Strings(switched)
	if len(switched) != 0 {
		t.Fatalf("disposition switch in the validation closure: %q", switched)
	}
	// The closure pin is not vacuous: the validation closures
	// are non-empty and PlanSession's counting switch exists
	// outside them.
	if len(closure) == 0 {
		t.Fatalf("validation closure is empty")
	}
	outside := false
	for _, file := range productionFiles(t) {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "PlanSession" || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				stmt, ok := node.(*ast.SwitchStmt)
				if !ok || stmt.Tag == nil {
					return true
				}
				if switchOnDisposition(stmt.Tag) {
					outside = true
				}
				return true
			})
		}
	}
	if !outside {
		t.Fatalf("PlanSession counting switch not found; the closure scoping claim is untested")
	}
}
