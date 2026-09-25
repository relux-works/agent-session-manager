package cloneplan_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// dagOracleVerdict is the independent DAG rule retyped from the
// spec sentence "Dependencies are lower sequences and form a DAG"
// plus the pinned depends_on_sequences shape (sorted unique
// uint53): sequences strictly increase, every depends list is
// strictly increasing, and every dependency names a strictly lower
// sequence present in the set. Strict decrease along every edge
// admits no cycle and no self-edge, so the oracle needs no separate
// cycle walk: any cycle would cross a non-decreasing edge.
func dagOracleVerdict(sequences []uint64, depends [][]uint64) bool {
	for index := 1; index < len(sequences); index++ {
		if sequences[index] <= sequences[index-1] {
			return false
		}
	}
	present := make(map[uint64]bool, len(sequences))
	for _, sequence := range sequences {
		present[sequence] = true
	}
	for index, sequence := range sequences {
		for position, dependency := range depends[index] {
			if position > 0 && dependency <= depends[index][position-1] {
				return false
			}
			if dependency >= sequence || !present[dependency] {
				return false
			}
		}
	}
	return true
}

func dagInput(sequences []uint64, depends [][]uint64) cloneplan.ProjectionPlanInput {
	input := validPlanInput()
	operations := make([]cloneplan.TargetOperationInput, 0, len(sequences))
	for index, sequence := range sequences {
		operation := validOperationInput(sequence, depends[index]...)
		operation.ResourceKeys = []string{fmt.Sprintf("res/%d", sequence)}
		operations = append(operations, operation)
	}
	input.TargetOperations = operations
	return input
}

func dagVerdict(t *testing.T, sequences []uint64, depends [][]uint64, baseline []byte) (bool, bool) {
	t.Helper()
	input := dagInput(sequences, depends)
	_, buildErr := cloneplan.BuildProjectionPlan(input)
	buildAdmits := buildErr == nil
	var decodeAdmits bool
	if buildAdmits {
		sealed, err := cloneplan.BuildProjectionPlan(input)
		if err != nil {
			t.Fatalf("rebuild: %v", err)
		}
		_, decodeErr := cloneplan.DecodeProjectionPlan(sealed)
		decodeAdmits = decodeErr == nil
	} else {
		// A build refusal still reaches decode through a
		// hand-assembled document: dependency violations are
		// shape-valid except for the DAG rule, so decode must
		// refuse them too.
		decodeAdmits = decodeAdmitsHandBuilt(t, depends, baseline)
	}
	return buildAdmits, decodeAdmits
}

// decodeAdmitsHandBuilt transplants the swept dependency lists into
// the shared sealed baseline (valid operations with empty depends),
// so decode-side verdicts are measured even for graphs build
// refuses. A DAG refusal is the expected decode verdict for invalid
// graphs; any refusal counts as non-admission here because the
// document is shape-valid except for the swept edges.
func decodeAdmitsHandBuilt(t *testing.T, depends [][]uint64, baseline []byte) bool {
	t.Helper()
	document := decodeDocument(t, baseline)
	rows := document["target_operations"].([]any)
	if len(rows) != len(depends) {
		t.Fatalf("baseline carries %d operations, swept graph has %d", len(rows), len(depends))
	}
	for index := range rows {
		edges := make([]any, 0, len(depends[index]))
		for _, dependency := range depends[index] {
			edges = append(edges, float64(dependency))
		}
		rows[index].(map[string]any)["depends_on_sequences"] = edges
	}
	_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	return err == nil
}

// dagBaseline seals the shared decode baseline for n contiguous
// operations with empty dependency lists.
func dagBaseline(t *testing.T, n int) []byte {
	t.Helper()
	sequences := make([]uint64, 0, n)
	empty := make([][]uint64, n)
	for sequence := uint64(1); sequence <= uint64(n); sequence++ {
		sequences = append(sequences, sequence)
	}
	return mustBuildPlan(t, dagInput(sequences, empty))
}

// TestDAGWholeDomainCensus classifies EVERY dependency graph through
// four operations (all 2^(n^2) edge sets, valid and invalid) against
// the independent oracle through both production entries. At five
// operations the full 33M-graph enumeration is infeasible, so the
// n==5 branch drives every edge class instead — self/equal, higher,
// lower, missing, unsorted, duplicate — with each class at every
// position, plus the valid-DAG admission sweep in
// TestDAGValidEnumerationFive and the locality proof in
// TestDAGLocalityStructure: validity is a per-edge local rule, so
// the position sweep covers the five-operation domain by structure.
func TestDAGWholeDomainCensus(t *testing.T) {
	for n := 1; n <= 5; n++ {
		sequences := make([]uint64, 0, n)
		for sequence := uint64(1); sequence <= uint64(n); sequence++ {
			sequences = append(sequences, sequence)
		}
		baseline := dagBaseline(t, n)
		if n == 5 {
			sweepFiveEdgeClasses(t, sequences, baseline)
			continue
		}
		total := 1 << (n * n)
		admitted := 0
		disagreements := 0
		for mask := 0; mask < total; mask++ {
			depends := make([][]uint64, n)
			for target := 0; target < n; target++ {
				for source := 0; source < n; source++ {
					bit := target*n + source
					if mask&(1<<bit) != 0 {
						// Edge stored sorted: sources ascend,
						// so the shape gate never fires here.
						depends[target] = append(depends[target], uint64(source+1))
					}
				}
			}
			want := dagOracleVerdict(sequences, depends)
			buildAdmits, decodeAdmits := dagVerdict(t, sequences, depends, baseline)
			if buildAdmits != want {
				disagreements++
				t.Errorf("n=%d mask=%b: build admits = %v, oracle = %v (depends=%v)", n, mask, buildAdmits, want, depends)
			}
			if decodeAdmits != want {
				disagreements++
				t.Errorf("n=%d mask=%b: decode admits = %v, oracle = %v (depends=%v)", n, mask, decodeAdmits, want, depends)
			}
			// A weakened gate disagrees on thousands of
			// graphs; the first 25 prove the kill and bound
			// the failure output. Green runs sweep every
			// graph and report the admitted count below.
			if disagreements >= 25 {
				t.Fatalf("n=%d: stopping after %d disagreements", n, disagreements)
			}
			if want {
				admitted++
			}
		}
		t.Logf("n=%d: %d of %d graphs admitted", n, admitted, total)
	}
}

// sweepFiveEdgeClasses drives every dependency edge class at every
// position of a five-operation plan against the oracle through both
// entries. The base graph is empty (valid); each probe adds edges at
// exactly one position.
func sweepFiveEdgeClasses(t *testing.T, sequences []uint64, baseline []byte) {
	t.Helper()
	probes := 0
	check := func(label string, position int, edges []uint64) {
		t.Helper()
		depends := make([][]uint64, len(sequences))
		depends[position] = edges
		want := dagOracleVerdict(sequences, depends)
		buildAdmits, decodeAdmits := dagVerdict(t, sequences, depends, baseline)
		if buildAdmits != want {
			t.Errorf("n=5 %s at op %d (%v): build admits = %v, oracle = %v", label, position+1, edges, buildAdmits, want)
		}
		if decodeAdmits != want {
			t.Errorf("n=5 %s at op %d (%v): decode admits = %v, oracle = %v", label, position+1, edges, decodeAdmits, want)
		}
		probes++
	}
	for position, sequence := range sequences {
		check("self-equal", position, []uint64{sequence})
		check("higher", position, []uint64{sequence + 1})
		if position == 0 {
			// No lower sequence exists at op 1: the empty
			// list is the vacuous lower case and admits.
			check("lower-vacuous", position, nil)
		} else {
			check("lower", position, []uint64{1})
		}
		check("missing", position, []uint64{0})
		if position == 0 {
			check("unsorted", position, []uint64{2, 1})
		} else {
			check("unsorted", position, []uint64{sequence + 1, 1})
		}
		check("duplicate", position, []uint64{1, 1})
	}
	t.Logf("n=5: %d edge-class probes classified", probes)
}

// TestDAGValidEnumerationFive admits every valid five-operation DAG:
// all 2^10 strictly-lower edge sets through both entries. Valid
// graphs through four operations are subsumed by the exhaustive
// census above.
func TestDAGValidEnumerationFive(t *testing.T) {
	const n = 5
	sequences := make([]uint64, 0, n)
	for sequence := uint64(1); sequence <= uint64(n); sequence++ {
		sequences = append(sequences, sequence)
	}
	baseline := dagBaseline(t, n)
	// Lower-triangular edge slots: (target, source) with
	// source < target.
	var slots [][2]int
	for target := 0; target < n; target++ {
		for source := 0; source < target; source++ {
			slots = append(slots, [2]int{target, source})
		}
	}
	total := 1 << len(slots)
	for mask := 0; mask < total; mask++ {
		depends := make([][]uint64, n)
		for slot, edge := range slots {
			if mask&(1<<slot) != 0 {
				depends[edge[0]] = append(depends[edge[0]], uint64(edge[1]+1))
			}
		}
		if !dagOracleVerdict(sequences, depends) {
			t.Fatalf("mask=%b: swept graph is not a valid DAG: the sweep is wrong", mask)
		}
		buildAdmits, decodeAdmits := dagVerdict(t, sequences, depends, baseline)
		if !buildAdmits || !decodeAdmits {
			t.Errorf("mask=%b: valid DAG refused (build=%v decode=%v)", mask, buildAdmits, decodeAdmits)
		}
	}
	t.Logf("n=5: %d valid DAGs admitted", total)
}

// TestDAGSingleEdgeLocality proves the per-edge locality claim
// behaviorally: around one rich valid five-operation base DAG, every
// single-edge toggle flips the verdict through both entries exactly
// when the toggled edge violates the oracle. No verdict depends on
// any other edge.
func TestDAGSingleEdgeLocality(t *testing.T) {
	base := [][]uint64{{}, {1}, {1, 2}, {2, 3}, {1, 4}}
	sequences := []uint64{1, 2, 3, 4, 5}
	if !dagOracleVerdict(sequences, base) {
		t.Fatal("locality base is not a valid DAG: the test vector is wrong")
	}
	baseline := dagBaseline(t, 5)
	toggles := 0
	for target := 0; target < 5; target++ {
		for edge := uint64(0); edge <= 6; edge++ {
			depends := make([][]uint64, 5)
			for row := range depends {
				depends[row] = append([]uint64{}, base[row]...)
			}
			// Toggle membership, keeping the list sorted.
			kept := depends[target][:0]
			present := false
			for _, current := range depends[target] {
				if current == edge {
					present = true
					continue
				}
				kept = append(kept, current)
			}
			if !present {
				kept = append(kept, edge)
				for position := len(kept) - 1; position > 0 && kept[position] < kept[position-1]; position-- {
					kept[position], kept[position-1] = kept[position-1], kept[position]
				}
			}
			depends[target] = kept
			want := dagOracleVerdict(sequences, depends)
			buildAdmits, decodeAdmits := dagVerdict(t, sequences, depends, baseline)
			if buildAdmits != want {
				t.Errorf("toggle op %d edge %d (%v): build admits = %v, oracle = %v", target+1, edge, depends[target], buildAdmits, want)
			}
			if decodeAdmits != want {
				t.Errorf("toggle op %d edge %d (%v): decode admits = %v, oracle = %v", target+1, edge, depends[target], decodeAdmits, want)
			}
			toggles++
		}
	}
	t.Logf("%d single-edge toggles classified", toggles)
}

// TestDAGLocalityStructure pins by structure that DAG validity is a
// per-edge local rule: checkOperationDAG holds no global-graph state
// beyond the presence set. It must contain exactly three range loops
// (presence build, edge loop, nested dependency loop), call nothing
// but make, len, and invalid, compare each dependency against its
// own sequence with >=, test presence with exactly one set store and
// a negated lookup, and spawn no goroutines or deferred work. The
// locality theorem then
// follows: verdict = order AND every edge (lower AND present), the
// depends lists arrive sorted unique from the shape gate, and strict
// decrease along every edge admits no cycle — a cycle would need a
// non-decreasing edge. This static pin is blind by design: the
// N-dag-locality-evasion harness row preserves every searched-for
// token while changing behavior, and the behavioral census above is
// its killer.
func TestDAGLocalityStructure(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(file), "operations.go"))
	if err != nil {
		t.Fatalf("read operations.go: %v", err)
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "operations.go", source, 0)
	if err != nil {
		t.Fatalf("parse operations.go: %v", err)
	}
	var dag *ast.FuncDecl
	var orderFound bool
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "checkOperationDAG" && function.Name.Name != "checkOperationOrder" {
			continue
		}
		if function.Name.Name == "checkOperationOrder" {
			orderFound = true
			continue
		}
		dag = function
	}
	if dag == nil {
		t.Fatal("checkOperationDAG not found: the structure moved")
	}
	if !orderFound {
		t.Fatal("checkOperationOrder not found: the order gate moved")
	}
	// Top-level shape: the presence-set declaration, the presence
	// build loop, the edge-check loop (which nests the
	// per-operation dependency loop), and the admission return.
	if len(dag.Body.List) != 4 {
		t.Fatalf("checkOperationDAG holds %d top-level statements, want 4 (presence set + presence loop + edge loop + return)", len(dag.Body.List))
	}
	if _, ok := dag.Body.List[0].(*ast.AssignStmt); !ok {
		t.Errorf("checkOperationDAG statement 1 is %T, want the presence-set assignment", dag.Body.List[0])
	}
	edgeLoop, ok := dag.Body.List[2].(*ast.RangeStmt)
	if !ok {
		t.Fatalf("checkOperationDAG statement 3 is %T, want the edge-check range loop", dag.Body.List[2])
	}
	if _, ok := dag.Body.List[3].(*ast.ReturnStmt); !ok {
		t.Errorf("checkOperationDAG statement 4 is %T, want the admission return", dag.Body.List[3])
	}
	nested := 0
	for _, statement := range edgeLoop.Body.List {
		if _, ok := statement.(*ast.RangeStmt); ok {
			nested++
		}
	}
	if nested != 1 {
		t.Errorf("edge-check loop nests %d range loops, want 1 (the dependency loop)", nested)
	}
	ranges := 0
	calls := map[string]bool{}
	presentStores := 0
	lowerCompared := false
	presenceTested := false
	ast.Inspect(dag.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.RangeStmt:
			ranges++
		case *ast.GoStmt:
			t.Errorf("checkOperationDAG spawns a goroutine: not a local rule")
		case *ast.DeferStmt:
			t.Errorf("checkOperationDAG defers work: not a local rule")
		case *ast.SendStmt:
			t.Errorf("checkOperationDAG sends on a channel: not a local rule")
		case *ast.CallExpr:
			switch function := typed.Fun.(type) {
			case *ast.Ident:
				calls[function.Name] = true
			case *ast.SelectorExpr:
				calls[function.Sel.Name] = true
			}
		case *ast.AssignStmt:
			for _, target := range typed.Lhs {
				if index, ok := target.(*ast.IndexExpr); ok {
					if ident, ok := index.X.(*ast.Ident); ok && ident.Name == "present" {
						presentStores++
					}
				}
			}
		case *ast.BinaryExpr:
			if typed.Op == token.GEQ {
				lowerCompared = lowerCompared || mentions(typed.X, "dependency") && mentions(typed.Y, "Sequence") ||
					mentions(typed.Y, "dependency") && mentions(typed.X, "Sequence")
			}
		case *ast.UnaryExpr:
			if typed.Op == token.NOT {
				if index, ok := typed.X.(*ast.IndexExpr); ok {
					if ident, ok := index.X.(*ast.Ident); ok && ident.Name == "present" {
						presenceTested = true
					}
				}
			}
		}
		return true
	})
	if ranges != 3 {
		t.Errorf("checkOperationDAG holds %d range loops, want 3 (presence build + edge loop + dependency loop)", ranges)
	}
	for name := range calls {
		if name != "make" && name != "invalid" && name != "len" {
			t.Errorf("checkOperationDAG calls %q, want only make, len, and invalid", name)
		}
	}
	if !calls["invalid"] {
		t.Errorf("checkOperationDAG never refuses: the gate is gone")
	}
	if presentStores != 1 {
		t.Errorf("checkOperationDAG stores into present %d times, want 1 (the presence build)", presentStores)
	}
	if !lowerCompared {
		t.Errorf("checkOperationDAG never compares a dependency against its sequence with >=")
	}
	if !presenceTested {
		t.Errorf("checkOperationDAG never tests the negated presence lookup")
	}
}

// mentions reports whether the expression references the named
// identifier or selector.
func mentions(expression ast.Expr, name string) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.Ident:
			if typed.Name == name {
				found = true
			}
		case *ast.SelectorExpr:
			if typed.Sel.Name == name {
				found = true
			}
		}
		return !found
	})
	return found
}

// TestDAGRefusalLiterals pins the refusal literals for each invalid
// edge class through the production entries: higher, equal,
// self-edge, dangling-lower, duplicate sequence, and unordered
// sequences.
func TestDAGRefusalLiterals(t *testing.T) {
	t.Run("higher", func(t *testing.T) {
		input := dagInput([]uint64{1, 2}, [][]uint64{{2}, {}})
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "target operation 1", "depends on sequence 2", "want a lower sequence")
	})
	t.Run("self", func(t *testing.T) {
		input := dagInput([]uint64{1, 2}, [][]uint64{{}, {2}})
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "target operation 2", "depends on sequence 2", "want a lower sequence")
	})
	t.Run("dangling", func(t *testing.T) {
		// Non-contiguous sequences: 2 is lower than 3 but no
		// operation carries it.
		input := dagInput([]uint64{1, 3}, [][]uint64{{}, {2}})
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "target operation 3", "depends on absent sequence 2")
	})
	t.Run("duplicate-sequence", func(t *testing.T) {
		input := dagInput([]uint64{1, 1}, [][]uint64{{}, {}})
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "target_operations", "not ordered by sequence")
	})
	t.Run("unordered", func(t *testing.T) {
		input := dagInput([]uint64{2, 1}, [][]uint64{{}, {}})
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "target_operations", "not ordered by sequence")
	})
	t.Run("decode-higher", func(t *testing.T) {
		sealed := mustBuildPlan(t, dagInput([]uint64{1, 2}, [][]uint64{{}, {}}))
		document := decodeDocument(t, sealed)
		rows := document["target_operations"].([]any)
		rows[0].(map[string]any)["depends_on_sequences"] = []any{float64(2)}
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "target operation 1", "depends on sequence 2", "want a lower sequence")
	})
	t.Run("decode-dangling", func(t *testing.T) {
		sealed := mustBuildPlan(t, dagInput([]uint64{1, 3}, [][]uint64{{}, {}}))
		document := decodeDocument(t, sealed)
		rows := document["target_operations"].([]any)
		rows[1].(map[string]any)["depends_on_sequences"] = []any{float64(2)}
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "target operation 3", "depends on absent sequence 2")
	})
	t.Run("cycle-is-higher", func(t *testing.T) {
		// A two-cycle 1->2->1 needs the non-decreasing edge
		// 1->2, which the lower-sequence rule refuses.
		input := dagInput([]uint64{1, 2}, [][]uint64{{2}, {1}})
		_, err := cloneplan.BuildProjectionPlan(input)
		requireRefusal(t, err, "want a lower sequence")
	})
}

// TestDAGNonContiguousValid pins that gaps in the sequence space
// admit: only strictly-lower presence matters, not contiguity.
func TestDAGNonContiguousValid(t *testing.T) {
	input := dagInput([]uint64{1, 5, 9}, [][]uint64{{}, {1}, {1, 5}})
	sealed := mustBuildPlan(t, input)
	plan := mustDecodePlan(t, sealed)
	if len(plan.TargetOperations) != 3 {
		t.Fatalf("operations = %d, want 3", len(plan.TargetOperations))
	}
}
