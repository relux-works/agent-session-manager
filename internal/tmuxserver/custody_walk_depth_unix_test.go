//go:build !windows

package tmuxserver

import (
	"context"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestCustodyAncestorWalkGeneratedDepthAtProductionEntries(t *testing.T) {
	entries := []string{"Probe", "Spawn", "Execute"}
	for extraDepth := 1; extraDepth <= 16; extraDepth++ {
		extraDepth := extraDepth
		for _, entry := range entries {
			entry := entry
			for _, position := range []string{"nearest_runtime_ancestor", "middle_ancestor", "deepest_non_root_ancestor"} {
				position := position
				t.Run(fmt.Sprintf("extra_%02d/%s/%s", extraDepth, position, entry), func(t *testing.T) {
					root, socket, fx, _ := newCustodyOracleEntryFixtureAtDepth(t, entry, extraDepth)
					ancestors := custodyAncestors(root)
					if len(ancestors) < extraDepth+2 {
						t.Fatalf("extra depth %d produced %d ancestor components, want at least %d", extraDepth, len(ancestors), extraDepth+2)
					}
					nonRoot := ancestors[:len(ancestors)-1]
					var target string
					switch position {
					case "nearest_runtime_ancestor":
						target = nonRoot[0]
					case "middle_ancestor":
						target = nonRoot[len(nonRoot)/2]
					case "deepest_non_root_ancestor":
						target = nonRoot[len(nonRoot)-1]
					}

					projectionHits := 0
					previousProjection := custodyModeProjection
					custodyModeProjection = func(path string, current os.FileMode) os.FileMode {
						if filepath.Clean(path) != filepath.Clean(target) {
							return current
						}
						projectionHits++
						return projectUnixLowMode(current, 0o777)
					}
					t.Cleanup(func() { custodyModeProjection = previousProjection })

					err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
					assertCustodyPermissionRefusal(t, err, "tmux_unsafe_socket_path", "socket ancestor")
					if effects != 0 {
						t.Fatalf("%s performed %d effect(s) before refusing mode 0777 at %s", entry, effects, target)
					}
					if projectionHits == 0 {
						t.Fatalf("mode 0777 at %s did not reach the production ancestor gate through %s", target, entry)
					}
				})
			}
		}
	}
}

func custodyAncestors(root string) []string {
	ancestors := make([]string, 0, 24)
	for ancestor := filepath.Clean(filepath.Dir(root)); ; ancestor = filepath.Dir(ancestor) {
		ancestors = append(ancestors, ancestor)
		if isFilesystemRoot(ancestor) {
			return ancestors
		}
	}
}

func TestCustodyAncestorWalkGeneratedDepthToPathMax(t *testing.T) {
	root, created, oneByteDepth := newCustodyPathMaxRoot(t)
	ancestors := custodyAncestors(root)
	if len(ancestors) < 3 {
		t.Fatalf("PATH_MAX fixture has only %d ancestors, want a deep generated walk", len(ancestors))
	}
	if len(root) > unix.PathMax-1 || len(filepath.Dir(root)) > unix.PathMax-1 {
		t.Fatalf("fixture exceeds PATH_MAX without its NUL: root=%d parent=%d PATH_MAX=%d", len(root), len(filepath.Dir(root)), unix.PathMax)
	}
	if len(root) != unix.PathMax-1 || len(filepath.Dir(root)) != unix.PathMax-3 {
		t.Fatalf("fixture did not reach the deepest valid ancestor path: root=%d ancestor=%d PATH_MAX=%d", len(root), len(filepath.Dir(root)), unix.PathMax)
	}
	if len(created) == 0 {
		t.Fatal("PATH_MAX fixture did not generate nested one-byte directory names")
	}
	t.Logf("PATH_MAX=%d; generated %d one-byte directories; runtime-root bytes=%d; deepest ancestor bytes=%d; ancestor positions=%d", unix.PathMax, oneByteDepth, len(root), len(filepath.Dir(root)), len(ancestors)-1)

	nonRootAncestors := ancestors[:len(ancestors)-1]
	if len(nonRootAncestors) < 2 {
		t.Fatalf("PATH_MAX fixture has %d non-root ancestors", len(nonRootAncestors))
	}
	deepest := nonRootAncestors[0]
	middle := nonRootAncestors[len(nonRootAncestors)/2]
	if filepath.Clean(deepest) != filepath.Clean(filepath.Dir(root)) {
		t.Fatalf("deepest ancestor = %q, want immediate parent %q", deepest, filepath.Dir(root))
	}
	if deepest == middle {
		t.Fatalf("middle ancestor %q unexpectedly equals deepest ancestor", middle)
	}

	for _, target := range []struct {
		name string
		path string
	}{
		{name: "deepest_non_root", path: deepest},
		{name: "middle", path: middle},
	} {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if err := os.Chmod(target.path, 0o777); err != nil {
				t.Fatalf("set 0777 on generated ancestor %q: %v", target.path, err)
			}
			t.Cleanup(func() { _ = os.Chmod(target.path, 0o700) })
			info, err := os.Stat(target.path)
			if err != nil {
				t.Fatalf("stat generated ancestor %q: %v", target.path, err)
			}
			if got := info.Mode().Perm(); got != 0o777 {
				t.Fatalf("generated ancestor permissions = %#o, want %#o", got, 0o777)
			}

			walkedTarget := 0
			previousProjection := custodyModeProjection
			custodyModeProjection = func(path string, mode os.FileMode) os.FileMode {
				if filepath.Clean(path) == filepath.Clean(target.path) {
					walkedTarget++
				}
				return mode
			}
			t.Cleanup(func() { custodyModeProjection = previousProjection })

			err = checkCustodyAncestors(root)
			assertCustodyPermissionRefusal(t, err, "tmux_unsafe_socket_path", "socket ancestor")
			if walkedTarget != 1 {
				t.Fatalf("shared ancestor predicate observed target %q %d times, want exactly once", target.path, walkedTarget)
			}
		})
	}

	deepSocket := SocketPath(filepath.Join(root, RuntimeDirName))
	if len(deepSocket) < unix.PathMax-1 {
		t.Fatalf("derived deep socket path has %d bytes, want the physical-depth fixture to exceed the unix socket limit", len(deepSocket))
	}
	if err := CheckSocketLength(deepSocket, scalar.PlatformMacOS); err == nil {
		t.Fatalf("deep derived socket of %d bytes unexpectedly passed sun_path limit %d", len(deepSocket), sunPathLimitDarwin)
	}
}

func newCustodyPathMaxRoot(t *testing.T) (string, []string, int) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve PATH_MAX fixture base: %v", err)
	}
	created := make([]string, 0, unix.PathMax/2)
	t.Cleanup(func() {
		for index := len(created) - 1; index >= 0; index-- {
			if err := os.Remove(created[index]); err != nil && !os.IsNotExist(err) {
				t.Errorf("remove deep custody fixture %q: %v", created[index], err)
			}
		}
	})

	parent := base
	if len(parent)%2 == 0 {
		parityBase := filepath.Join(parent, "xy")
		if err := os.Mkdir(parityBase, 0o700); err != nil {
			t.Fatalf("create PATH_MAX parity directory: %v", err)
		}
		created = append(created, parityBase)
		parent = parityBase
	}
	oneByteDepth := 0
	for {
		next := filepath.Join(parent, "x")
		rootCandidate := filepath.Join(next, "r")
		if len(rootCandidate) > unix.PathMax-1 {
			break
		}
		if err := os.Mkdir(next, 0o700); err != nil {
			t.Fatalf("create one-byte path component at %d bytes: %v", len(next), err)
		}
		created = append(created, next)
		parent = next
		oneByteDepth++
	}
	root := filepath.Join(parent, "r")
	if len(root) > unix.PathMax-1 {
		t.Fatalf("runtime-root fixture path has %d bytes, exceeds PATH_MAX-1=%d", len(root), unix.PathMax-1)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create PATH_MAX runtime-root fixture at %d bytes: %v", len(root), err)
	}
	created = append(created, root)
	return root, created, oneByteDepth
}

func TestCustodyAncestorWalkReachableFromProductionEntries(t *testing.T) {
	graph := loadCustodyProductionCallGraph(t)
	entries := []struct {
		name string
		key  string
	}{
		{name: "Probe", key: "tmuxserver.ServerProber.Probe"},
		{name: "Spawn", key: "tmuxserver.ServerSpawner.Spawn"},
		{name: "Execute", key: "tmuxserver.Lifecycle.Execute"},
	}
	for _, entry := range entries {
		entry := entry
		t.Run(entry.name, func(t *testing.T) {
			reachable := custodyReachableFunctions(graph, entry.key)
			if _, ok := reachable["tmuxserver.CheckSocketCustody"]; !ok {
				t.Fatalf("production entry %s no longer reaches CheckSocketCustody", entry.key)
			}
			if _, ok := reachable["tmuxserver.checkCustodyAncestors"]; !ok {
				t.Fatalf("production entry %s no longer reaches checkCustodyAncestors", entry.key)
			}
		})
	}
}

func TestCustodyAncestorWalkHasNoLengthDependentControlFlow(t *testing.T) {
	graph := loadCustodyProductionCallGraph(t)
	walkKey := "tmuxserver.checkCustodyAncestors"
	walk := graph.functions[walkKey]
	if walk == nil {
		t.Fatalf("production call graph has no %s", walkKey)
	}
	reachable := custodyReachableFunctions(graph, walkKey)
	keys := make([]string, 0, len(reachable))
	for key := range reachable {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	t.Logf("transitive custody decision call graph: %s", strings.Join(keys, ", "))

	for _, key := range keys {
		if custodyDiagnosticOnlyStringFunction(graph, reachable, key) {
			t.Logf("%s is reached only to build failPath detail text; its existing display truncation does not affect custody decisions", key)
			continue
		}
		assertNoCustodyPathLengthBranch(t, graph, reachable[key])
	}

	assertCustodyAncestorWalkStructure(t, graph, walk)
}

type custodyASTFunction struct {
	key         string
	packageName string
	file        *ast.File
	declaration *ast.FuncDecl
	imports     map[string]string
}

type custodyASTCallGraph struct {
	functions map[string]*custodyASTFunction
	topLevel  map[string]string
	methods   map[string][]string
	fset      *token.FileSet
}

func loadCustodyProductionCallGraph(t *testing.T) *custodyASTCallGraph {
	t.Helper()
	moduleBytes, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read module path for custody call graph: %v", err)
	}
	modulePath := ""
	for _, line := range strings.Split(string(moduleBytes), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			modulePath = fields[1]
			break
		}
	}
	if modulePath == "" {
		t.Fatal("go.mod has no module declaration")
	}

	graph := &custodyASTCallGraph{
		functions: make(map[string]*custodyASTFunction),
		topLevel:  make(map[string]string),
		methods:   make(map[string][]string),
		fset:      token.NewFileSet(),
	}
	for _, packageDir := range []string{".", "../secprim"} {
		entries, err := os.ReadDir(packageDir)
		if err != nil {
			t.Fatalf("read custody source directory %q: %v", packageDir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			matches, err := build.Default.MatchFile(packageDir, name)
			if err != nil {
				t.Fatalf("match production source %s: %v", filepath.Join(packageDir, name), err)
			}
			if !matches {
				continue
			}
			path := filepath.Join(packageDir, name)
			file, err := parser.ParseFile(graph.fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse production custody source %s: %v", path, err)
			}
			imports := make(map[string]string)
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatalf("unquote import path in %s: %v", path, err)
				}
				packageName := filepath.Base(importPath)
				if !strings.HasPrefix(importPath, modulePath+"/internal/") {
					continue
				}
				alias := packageName
				if spec.Name != nil {
					alias = spec.Name.Name
				}
				imports[alias] = packageName
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Body == nil {
					continue
				}
				packageName := file.Name.Name
				key := packageName + "." + function.Name.Name
				if receiver := custodyReceiverName(function); receiver != "" {
					key = packageName + "." + receiver + "." + function.Name.Name
					graph.methods[packageName+"."+function.Name.Name] = append(graph.methods[packageName+"."+function.Name.Name], key)
				} else {
					graph.topLevel[packageName+"."+function.Name.Name] = key
				}
				graph.functions[key] = &custodyASTFunction{key: key, packageName: packageName, file: file, declaration: function, imports: imports}
			}
		}
	}
	return graph
}

func custodyReceiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	typeExpression := function.Recv.List[0].Type
	for {
		switch current := typeExpression.(type) {
		case *ast.StarExpr:
			typeExpression = current.X
		case *ast.ParenExpr:
			typeExpression = current.X
		default:
			if identifier, ok := current.(*ast.Ident); ok {
				return identifier.Name
			}
			return ""
		}
	}
}

func custodyReachableFunctions(graph *custodyASTCallGraph, start string) map[string]*custodyASTFunction {
	reachable := make(map[string]*custodyASTFunction)
	queue := []string{start}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, seen := reachable[key]; seen {
			continue
		}
		function := graph.functions[key]
		if function == nil {
			continue
		}
		reachable[key] = function
		ast.Inspect(function.declaration.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			queue = append(queue, custodyASTCallTargets(graph, function, call.Fun)...)
			return true
		})
	}
	return reachable
}

func custodyASTCallTargets(graph *custodyASTCallGraph, caller *custodyASTFunction, function ast.Expr) []string {
	switch current := function.(type) {
	case *ast.Ident:
		if target := graph.topLevel[caller.packageName+"."+current.Name]; target != "" {
			return []string{target}
		}
	case *ast.SelectorExpr:
		if receiver, ok := current.X.(*ast.Ident); ok {
			if importedPackage := caller.imports[receiver.Name]; importedPackage != "" {
				if target := graph.topLevel[importedPackage+"."+current.Sel.Name]; target != "" {
					return []string{target}
				}
				return nil
			}
		}
		return graph.methods[caller.packageName+"."+current.Sel.Name]
	}
	return nil
}

func assertCustodyAncestorWalkStructure(t *testing.T, graph *custodyASTCallGraph, function *custodyASTFunction) {
	t.Helper()
	fset := graph.fset

	var loops []*ast.ForStmt
	var conditions []*ast.IfStmt
	var returns []*ast.ReturnStmt
	var forbiddenBranch string
	var integerLiteral string
	ast.Inspect(function.declaration.Body, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.ForStmt:
			loops = append(loops, current)
		case *ast.IfStmt:
			conditions = append(conditions, current)
		case *ast.ReturnStmt:
			returns = append(returns, current)
		case *ast.BranchStmt:
			if current.Tok == token.BREAK || current.Tok == token.CONTINUE || current.Tok == token.GOTO {
				forbiddenBranch = current.Tok.String()
			}
		case *ast.BasicLit:
			if current.Kind == token.INT {
				integerLiteral = current.Value
			}
		}
		return true
	})
	if len(loops) != 1 {
		t.Fatalf("checkCustodyAncestors has %d loops, want one unconditional parent walk", len(loops))
	}
	if loop := loops[0]; loop.Init != nil || loop.Cond != nil || loop.Post != nil {
		t.Fatal("checkCustodyAncestors loop has a counter, condition, or post clause")
	}
	if forbiddenBranch != "" {
		t.Fatalf("checkCustodyAncestors uses %s to leave or skip the parent walk", forbiddenBranch)
	}
	if integerLiteral != "" {
		t.Fatalf("checkCustodyAncestors contains integer literal %s that can encode a depth limit", integerLiteral)
	}
	if len(conditions) != 5 {
		t.Fatalf("checkCustodyAncestors has %d conditional branches, want the two root exits and three refusal gates", len(conditions))
	}

	rootGuardLines := make([][2]int, 0, 2)
	rootGuardCount := 0
	for _, condition := range conditions {
		if !isCustodyRootExitCondition(condition.Cond) {
			continue
		}
		rootGuardCount++
		rootGuardLines = append(rootGuardLines, [2]int{
			fset.Position(condition.Body.Lbrace).Line,
			fset.Position(condition.Body.Rbrace).Line,
		})
	}
	if rootGuardCount != 2 {
		t.Fatalf("checkCustodyAncestors has %d recognized filesystem-root guards, want both parent == cleaned and filesystem-root checks", rootGuardCount)
	}

	nilReturns := 0
	for _, result := range returns {
		if len(result.Results) != 1 {
			continue
		}
		identifier, ok := result.Results[0].(*ast.Ident)
		if !ok || identifier.Name != "nil" {
			continue
		}
		nilReturns++
		line := fset.Position(result.Pos()).Line
		insideRootGuard := false
		for _, bounds := range rootGuardLines {
			if line > bounds[0] && line < bounds[1] {
				insideRootGuard = true
				break
			}
		}
		if !insideRootGuard {
			t.Fatalf("checkCustodyAncestors returns nil at line %d outside a filesystem-root guard", line)
		}
	}
	if nilReturns != 2 {
		t.Fatalf("checkCustodyAncestors has %d nil exits, want only the two filesystem-root exits", nilReturns)
	}
}

func assertNoCustodyPathLengthBranch(t *testing.T, graph *custodyASTCallGraph, function *custodyASTFunction) {
	t.Helper()
	pathValues := make(map[string]bool)
	metricValues := make(map[string]bool)
	if function.declaration.Type.Params != nil {
		for _, field := range function.declaration.Type.Params.List {
			for _, name := range field.Names {
				if isCustodyPathLikeType(field.Type) {
					pathValues[name.Name] = true
				}
			}
		}
	}

	changed := true
	for changed {
		changed = false
		ast.Inspect(function.declaration.Body, func(node ast.Node) bool {
			if current, ok := node.(*ast.RangeStmt); ok && custodyExprReturnsPathValue(graph, function, current.X, pathValues, metricValues) {
				for _, expression := range []ast.Expr{current.Key, current.Value} {
					identifier, ok := expression.(*ast.Ident)
					if ok && identifier.Name != "_" && !pathValues[identifier.Name] {
						pathValues[identifier.Name] = true
						changed = true
					}
				}
			}
			var names []*ast.Ident
			var values []ast.Expr
			switch current := node.(type) {
			case *ast.ValueSpec:
				names = current.Names
				values = current.Values
			case *ast.AssignStmt:
				for _, left := range current.Lhs {
					if identifier, ok := left.(*ast.Ident); ok {
						names = append(names, identifier)
					}
				}
				values = current.Rhs
			default:
				return true
			}
			for index, name := range names {
				if name == nil || len(values) == 0 {
					continue
				}
				expressionIndex := index
				if len(values) == 1 && len(names) > 1 {
					expressionIndex = 0
				} else if expressionIndex >= len(values) {
					continue
				}
				expression := values[expressionIndex]
				if custodyExprIsPathMetric(graph, function, expression, pathValues, metricValues) && !metricValues[name.Name] {
					metricValues[name.Name] = true
					changed = true
				} else if custodyExprReturnsPathValue(graph, function, expression, pathValues, metricValues) && !pathValues[name.Name] {
					pathValues[name.Name] = true
					changed = true
				} else if custodyExprHasPathMetric(graph, function, expression, pathValues, metricValues) && !metricValues[name.Name] {
					metricValues[name.Name] = true
					changed = true
				}
			}
			return true
		})
	}

	var violation string
	var violationLine int
	ast.Inspect(function.declaration.Body, func(node ast.Node) bool {
		if violation != "" {
			return false
		}
		switch current := node.(type) {
		case *ast.BinaryExpr:
			if !isComparisonOperator(current.Op) {
				return true
			}
			if custodyExprIsPathMetric(graph, function, current.X, pathValues, metricValues) || custodyExprIsPathMetric(graph, function, current.Y, pathValues, metricValues) {
				violation = "path-derived length/depth comparison"
			} else if custodyIntegerLiteral(current.X) && custodyExprHasPath(current.Y, pathValues, metricValues) || custodyIntegerLiteral(current.Y) && custodyExprHasPath(current.X, pathValues, metricValues) {
				violation = "literal threshold over a path-derived value"
			}
		case *ast.ForStmt:
			if current.Init != nil || current.Post != nil {
				violation = "counter or bounded loop"
			}
		case *ast.IncDecStmt:
			violation = "counter update"
		case *ast.SwitchStmt:
			if custodyExprIsPathMetric(graph, function, current.Tag, pathValues, metricValues) {
				violation = "path-derived length/depth switch"
			}
		}
		if violation != "" {
			violationLine = graph.fset.Position(node.Pos()).Line
			return false
		}
		return true
	})
	if violation != "" {
		t.Fatalf("%s contains %s at line %d", function.key, violation, violationLine)
	}
}

func isCustodyPathLikeType(expression ast.Expr) bool {
	switch current := expression.(type) {
	case *ast.Ident:
		return current.Name == "string" || current.Name == "byte" || current.Name == "rune"
	case *ast.ArrayType:
		return current.Len == nil && isCustodyPathLikeType(current.Elt)
	case *ast.StarExpr:
		return isCustodyPathLikeType(current.X)
	case *ast.ParenExpr:
		return isCustodyPathLikeType(current.X)
	default:
		return false
	}
}

func custodyExprHasPath(expression ast.Expr, pathValues, metricValues map[string]bool) bool {
	if expression == nil {
		return false
	}
	derived := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if derived {
			return false
		}
		identifier, ok := node.(*ast.Ident)
		if ok && (pathValues[identifier.Name] || metricValues[identifier.Name]) {
			derived = true
			return false
		}
		return true
	})
	return derived
}

func custodyExprReturnsPathValue(graph *custodyASTCallGraph, function *custodyASTFunction, expression ast.Expr, pathValues, metricValues map[string]bool) bool {
	if expression == nil {
		return false
	}
	switch current := expression.(type) {
	case *ast.Ident:
		return pathValues[current.Name]
	case *ast.ParenExpr:
		return custodyExprReturnsPathValue(graph, function, current.X, pathValues, metricValues)
	case *ast.IndexExpr:
		return custodyExprReturnsPathValue(graph, function, current.X, pathValues, metricValues)
	case *ast.SliceExpr:
		return custodyExprReturnsPathValue(graph, function, current.X, pathValues, metricValues)
	case *ast.BinaryExpr:
		if current.Op == token.ADD {
			return custodyExprReturnsPathValue(graph, function, current.X, pathValues, metricValues) || custodyExprReturnsPathValue(graph, function, current.Y, pathValues, metricValues)
		}
	case *ast.CallExpr:
		pathArgument := false
		for _, argument := range current.Args {
			if custodyExprHasPath(argument, pathValues, metricValues) {
				pathArgument = true
				break
			}
		}
		if !pathArgument {
			return false
		}
		if custodyKnownPathValueCall(custodyCallName(current.Fun)) || custodyPathByteConversion(current.Fun) {
			return true
		}
		for _, key := range custodyASTCallTargets(graph, function, current.Fun) {
			if custodyFunctionReturnsPathValue(graph.functions[key].declaration) {
				return true
			}
		}
	}
	return false
}

func custodyExprHasPathMetric(graph *custodyASTCallGraph, function *custodyASTFunction, expression ast.Expr, pathValues, metricValues map[string]bool) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if ok && custodyExprIsPathMetric(graph, function, call, pathValues, metricValues) {
			found = true
			return false
		}
		return true
	})
	return found
}

func custodyPathByteConversion(expression ast.Expr) bool {
	array, ok := expression.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	return isCustodyPathLikeType(array.Elt)
}

func custodyKnownPathValueCall(name string) bool {
	switch name {
	case "string", "Clean", "Dir", "Join", "Base", "Abs", "Rel", "EvalSymlinks", "Readlink", "Trim", "TrimPrefix", "TrimSuffix", "TrimSpace", "Replace", "ReplaceAll", "ToLower", "ToUpper", "CutPrefix", "CutSuffix", "Split", "SplitN", "SplitAfter", "SplitAfterN", "SplitList", "Fields", "FieldsFunc":
		return true
	default:
		return false
	}
}

func custodyExprIsPathMetric(graph *custodyASTCallGraph, function *custodyASTFunction, expression ast.Expr, pathValues, metricValues map[string]bool) bool {
	if expression == nil {
		return false
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return metricValues[identifier.Name]
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	name := custodyCallName(call.Fun)
	pathArgument := false
	for _, argument := range call.Args {
		if custodyExprHasPath(argument, pathValues, metricValues) {
			pathArgument = true
			break
		}
	}
	if !pathArgument {
		return false
	}
	if custodyKnownPathMetricCall(name) {
		return true
	}
	for _, key := range custodyASTCallTargets(graph, function, call.Fun) {
		if custodyFunctionReturnsInt(graph.functions[key].declaration) {
			return true
		}
	}
	return false
}

func custodyKnownPathMetricCall(name string) bool {
	switch name {
	case "len", "cap", "Count", "RuneCountInString", "ComponentCount", "PathDepth", "Index", "LastIndex", "IndexByte", "LastIndexByte":
		return true
	default:
		return false
	}
}

func custodyFunctionReturnsInt(function *ast.FuncDecl) bool {
	if function.Type.Results == nil {
		return false
	}
	for _, field := range function.Type.Results.List {
		if identifier, ok := field.Type.(*ast.Ident); ok && (identifier.Name == "int" || strings.HasPrefix(identifier.Name, "uint")) {
			return true
		}
	}
	return false
}

func custodyFunctionReturnsPathValue(function *ast.FuncDecl) bool {
	if function.Type.Results == nil {
		return false
	}
	for _, field := range function.Type.Results.List {
		if isCustodyPathLikeType(field.Type) {
			return true
		}
	}
	return false
}

func custodyFunctionReturnsString(function *ast.FuncDecl) bool {
	if function.Type.Results == nil {
		return false
	}
	for _, field := range function.Type.Results.List {
		if identifier, ok := field.Type.(*ast.Ident); ok && identifier.Name == "string" {
			return true
		}
	}
	return false
}

func custodyDiagnosticOnlyStringFunction(graph *custodyASTCallGraph, reachable map[string]*custodyASTFunction, key string) bool {
	function := graph.functions[key]
	if function == nil || !custodyFunctionReturnsString(function.declaration) {
		return false
	}
	uses, failPathDetailUses := 0, 0
	for _, caller := range reachable {
		ast.Inspect(caller.declaration.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if custodyTargetsContain(custodyASTCallTargets(graph, caller, call.Fun), key) {
				uses++
			}
			if custodyCallName(call.Fun) == "failPath" {
				failPathDetailUses += custodyCountTargetCalls(graph, caller, call, key)
			}
			return true
		})
	}
	return uses > 0 && uses == failPathDetailUses
}

func custodyTargetsContain(targets []string, expected string) bool {
	for _, target := range targets {
		if target == expected {
			return true
		}
	}
	return false
}

func custodyCountTargetCalls(graph *custodyASTCallGraph, caller *custodyASTFunction, expression ast.Node, expected string) int {
	count := 0
	ast.Inspect(expression, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && custodyTargetsContain(custodyASTCallTargets(graph, caller, call.Fun), expected) {
			count++
		}
		return true
	})
	return count
}

func custodyCallName(function ast.Expr) string {
	switch current := function.(type) {
	case *ast.Ident:
		return current.Name
	case *ast.SelectorExpr:
		return current.Sel.Name
	default:
		return ""
	}
}

func custodyIntegerLiteral(expression ast.Expr) bool {
	literal, ok := expression.(*ast.BasicLit)
	return ok && literal.Kind == token.INT
}

func isComparisonOperator(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
		return true
	default:
		return false
	}
}

func isCustodyRootExitCondition(expression ast.Expr) bool {
	switch condition := expression.(type) {
	case *ast.BinaryExpr:
		if condition.Op != token.EQL {
			return false
		}
		left, leftOK := condition.X.(*ast.Ident)
		right, rightOK := condition.Y.(*ast.Ident)
		return leftOK && rightOK && left.Name == "parent" && right.Name == "cleaned"
	case *ast.CallExpr:
		function, ok := condition.Fun.(*ast.Ident)
		if !ok || function.Name != "isFilesystemRoot" || len(condition.Args) != 1 {
			return false
		}
		argument, ok := condition.Args[0].(*ast.Ident)
		return ok && argument.Name == "parent"
	default:
		return false
	}
}

func TestCustodyPathComponentByteLengthsAtProductionEntries(t *testing.T) {
	entries := []string{"Probe", "Spawn", "Execute"}
	classes := []string{"runtime_root", "ancestor"}
	for _, entry := range entries {
		entry := entry
		t.Run(entry, func(t *testing.T) {
			baseRoot, _, _, _ := newCustodyOracleEntryFixtureAtDepth(t, entry, 0)
			base := filepath.Dir(baseRoot)
			for _, class := range classes {
				class := class
				componentBase := filepath.Join(base, class)
				if err := os.Mkdir(componentBase, 0o700); err != nil {
					t.Fatalf("create %s component base: %v", class, err)
				}
				maxBytes := maxCustodyComponentBytes(componentBase, class)
				if maxBytes < 1 {
					t.Fatalf("no component-name byte value reaches custody through %s/%s", entry, class)
				}
				t.Logf("%s/%s admits component-name byte lengths 1..%d; CheckSocketLength rejects %d bytes", entry, class, maxBytes, maxBytes+1)
				if CheckSocketLength(custodyNamedSocketPath(componentBase, class, strings.Repeat("x", maxBytes+1)), scalar.PlatformMacOS) == nil {
					t.Fatalf("%s/%s byte length %d unexpectedly passes the platform socket-path bound", entry, class, maxBytes+1)
				}
				t.Run(fmt.Sprintf("%s/bytes_%03d_refused", class, maxBytes+1), func(t *testing.T) {
					root, socket, fx := newCustodyNamedEntryFixtureAtBase(t, entry, class, strings.Repeat("x", maxBytes+1), componentBase)
					if err := CheckSocketLength(socket, scalar.PlatformMacOS); err == nil {
						t.Fatalf("the %d-byte derived path unexpectedly passes the platform socket-path bound", maxBytes+1)
					}
					err, effects := runCustodyPathLengthEntry(t, entry, root, socket, fx)
					assertCustodyPermissionRefusal(t, err, "tmux_socket_path_too_long", "socket path length")
					if effects != 0 {
						t.Fatalf("%s performed %d effect(s) before refusing the over-limit %s path", entry, effects, class)
					}
				})
				for byteLength := 1; byteLength <= maxBytes; byteLength++ {
					byteLength := byteLength
					t.Run(fmt.Sprintf("%s/bytes_%03d", class, byteLength), func(t *testing.T) {
						name := strings.Repeat("x", byteLength)
						root, socket, fx := newCustodyNamedEntryFixtureAtBase(t, entry, class, name, componentBase)
						if len([]byte(name)) != byteLength {
							t.Fatalf("component has %d bytes, want %d", len([]byte(name)), byteLength)
						}
						if err := CheckSocketLength(socket, scalar.PlatformMacOS); err != nil {
							t.Fatalf("socket path for %d-byte %s component is out of contract: %v", byteLength, class, err)
						}
						installSyntheticCustodySocket(t, socket)
						err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
						if err != nil {
							t.Fatalf("valid %s component name of %d bytes refused by %s: %v", class, byteLength, entry, err)
						}
						if effects == 0 {
							t.Fatalf("%s did not reach its fake next-stage effect for %s component name of %d bytes", entry, class, byteLength)
						}
					})
				}
			}
		})
	}
}

func maxCustodyComponentBytes(base, class string) int {
	maxBytes := 0
	for byteLength := 1; byteLength <= 255; byteLength++ {
		if CheckSocketLength(custodyNamedSocketPath(base, class, strings.Repeat("x", byteLength)), scalar.PlatformMacOS) != nil {
			return maxBytes
		}
		maxBytes = byteLength
	}
	return maxBytes
}

func runCustodyPathLengthEntry(t *testing.T, entry, root, socket string, fx *lxFixture) (error, int) {
	t.Helper()
	switch entry {
	case "Probe":
		dialer := &fakeDialer{outcomes: []DialOutcome{DialStale}}
		prober := ServerProber{Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer, Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil }}
		_, err := prober.Probe(socket)
		return err, len(dialer.calls)
	case "Spawn":
		runner := newFakeRunner().queue("new-session", "", 0)
		spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
		argv := []string{"tmux", "-S", socket, "new-session", "-d", "-s", lxSession}
		return spawner.Spawn(socket, argv), runner.callCount()
	case "Execute":
		if fx == nil {
			t.Fatal("Execute fixture is required")
		}
		recordBinding(t, fx, lxDigestA)
		fx.runner.queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		})
		return err, fx.runner.callCount()
	default:
		t.Fatalf("unhandled entry %q", entry)
		return nil, 0
	}
}

func custodyNamedSocketPath(base, class, name string) string {
	root := filepath.Join(base, name)
	if class == "ancestor" {
		root = filepath.Join(base, name, "r")
	}
	return filepath.Join(root, RuntimeDirName, SocketName)
}

func newCustodyNamedEntryFixture(t *testing.T, entry, class, name string) (string, string, *lxFixture) {
	t.Helper()
	baseRoot, _, fx := newCustodyOracleEntryFixture(t, entry)
	base := filepath.Dir(baseRoot)
	return newCustodyNamedEntryFixtureWithFX(t, entry, class, name, base, fx)
}

func newCustodyNamedEntryFixtureAtBase(t *testing.T, entry, class, name, base string) (string, string, *lxFixture) {
	t.Helper()
	_, _, fx := newCustodyOracleEntryFixture(t, entry)
	return newCustodyNamedEntryFixtureWithFX(t, entry, class, name, base, fx)
}

func newCustodyNamedEntryFixtureWithFX(t *testing.T, entry, class, name, base string, fx *lxFixture) (string, string, *lxFixture) {
	t.Helper()
	root := filepath.Join(base, name)
	if class == "ancestor" {
		parent := root
		if err := os.Mkdir(parent, 0o700); err != nil {
			t.Fatalf("create named ancestor %q: %v", name, err)
		}
		root = filepath.Join(parent, "r")
	} else if class != "runtime_root" {
		t.Fatalf("unhandled custody component class %q", class)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create named runtime root %q: %v", name, err)
	}
	runtimeDir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("create runtime directory for component %q: %v", name, err)
	}
	socket := SocketPath(runtimeDir)
	if fx != nil {
		fx.root = root
		fx.runtime = runtimeDir
		fx.socket = socket
		fx.lc.Root = root
		fx.lc.RuntimeDir = runtimeDir
	}
	return root, socket, fx
}

func TestCustodyPathComponentNameContentAtProductionEntries(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "ascii", value: "runtime"},
		{name: "dot_prefixed", value: ".hidden"},
		{name: "dot_run", value: "..."},
		{name: "embedded_dots", value: "two..dots"},
		{name: "space", value: "two words"},
		{name: "latin_non_ascii", value: "café"},
		{name: "cjk_non_ascii", value: "猫"},
		{name: "punctuation", value: "_-."},
	}
	for _, class := range []string{"runtime_root", "ancestor"} {
		class := class
		for _, testCase := range cases {
			testCase := testCase
			for _, entry := range []string{"Probe", "Spawn", "Execute"} {
				entry := entry
				t.Run(class+"/"+testCase.name+"/"+entry, func(t *testing.T) {
					root, socket, fx := newCustodyNamedEntryFixture(t, entry, class, testCase.value)
					installSyntheticCustodySocket(t, socket)
					err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
					if err != nil {
						t.Fatalf("valid %q component name refused through %s: %v", testCase.value, entry, err)
					}
					if effects == 0 {
						t.Fatalf("%s did not reach its fake next-stage effect for component %q", entry, testCase.value)
					}
				})
			}
		}
	}
}

func TestCustodySymlinkChainLengthAtProductionEntries(t *testing.T) {
	classes := []string{"socket_leaf", "runtime_tmux_dir", "runtime_root", "ancestor"}
	entries := []string{"Probe", "Spawn", "Execute"}
	for length := 1; length <= 16; length++ {
		length := length
		for _, class := range classes {
			class := class
			for _, entry := range entries {
				entry := entry
				t.Run(fmt.Sprintf("links_%02d/%s/%s", length, class, entry), func(t *testing.T) {
					root, socket, fx := newCustodyOracleEntryFixture(t, entry)
					root, socket = installCustodySymlinkChain(t, class, root, socket, fx, length)
					err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
					code, detail := custodySymlinkRefusal(class)
					assertCustodyPermissionRefusal(t, err, code, detail)
					if effects != 0 {
						t.Fatalf("%s performed %d effect(s) before refusing %d-link %s chain", entry, effects, length, class)
					}
				})
			}
		}
	}
}

func installCustodySymlinkChain(t *testing.T, class, root, socket string, fx *lxFixture, length int) (string, string) {
	t.Helper()
	switch class {
	case "socket_leaf":
		runtimeDir := filepath.Dir(socket)
		actualSocket := filepath.Join(runtimeDir, "custody-target.sock")
		listener, err := net.Listen("unix", actualSocket)
		if err != nil {
			t.Fatalf("bind non-tmux custody target socket: %v", err)
		}
		t.Cleanup(func() { _ = listener.Close() })
		installCustodySymlinkAtPath(t, runtimeDir, socket, actualSocket, length)
	case "runtime_tmux_dir":
		runtimeDir := filepath.Join(root, RuntimeDirName)
		targetDir := filepath.Join(root, "runtime-target")
		if err := os.Rename(runtimeDir, targetDir); err != nil {
			t.Fatalf("rename runtime directory for symlink-chain fixture: %v", err)
		}
		installCustodySymlinkAtPath(t, root, runtimeDir, targetDir, length)
	case "runtime_root":
		base := filepath.Dir(root)
		targetRoot := filepath.Join(base, "root-target")
		if err := os.Rename(root, targetRoot); err != nil {
			t.Fatalf("rename runtime root for symlink-chain fixture: %v", err)
		}
		installCustodySymlinkAtPath(t, base, root, targetRoot, length)
	case "ancestor":
		base := filepath.Dir(root)
		alias := filepath.Join(base, "ancestor-alias")
		installCustodySymlinkAtPath(t, base, alias, base, length)
		root = filepath.Join(alias, filepath.Base(root))
		socket = filepath.Join(root, RuntimeDirName, SocketName)
		if fx != nil {
			fx.root = root
			fx.runtime = filepath.Join(root, RuntimeDirName)
			fx.socket = socket
			fx.lc.Root = root
			fx.lc.RuntimeDir = fx.runtime
		}
	default:
		t.Fatalf("unhandled symlink component class %q", class)
	}
	return root, socket
}

func installCustodySymlinkAtPath(t *testing.T, directory, finalLink, target string, length int) {
	t.Helper()
	if length < 1 {
		t.Fatalf("symlink chain length = %d, want at least one", length)
	}
	next := target
	for index := length - 1; index >= 1; index-- {
		link := filepath.Join(directory, fmt.Sprintf("custody-link-%02d", index))
		if err := os.Symlink(next, link); err != nil {
			t.Fatalf("create custody symlink %d of %d: %v", index, length, err)
		}
		next = link
	}
	if err := os.Symlink(next, finalLink); err != nil {
		t.Fatalf("create final custody symlink: %v", err)
	}
}

func custodySymlinkRefusal(class string) (string, string) {
	if class == "socket_leaf" {
		return "tmux_unsafe_socket_path", "socket kind"
	}
	if class == "ancestor" {
		return "tmux_unsafe_socket_path", "socket ancestor"
	}
	return "tmux_unsafe_runtime_dir", "runtime containment"
}
