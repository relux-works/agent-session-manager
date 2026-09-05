package provider

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// discoverSourceSites are the external-discovery call sites derived from
// the production Discover body: every directory enumeration must flow
// through collectDirectory, and no external Candidate may be built inline.
type discoverSourceSites struct {
	// collectClasses are the source-label classes of the collectDirectory
	// call sites: "plugin_dirs" or "path".
	collectClasses []string
	// directReads counts system.ReadDir calls inside Discover itself.
	directReads int
	// directNameParses counts externalID calls inside Discover itself.
	directNameParses int
	// inlineExternals counts KindExternal Candidate literals inside
	// Discover itself.
	inlineExternals int
	// inlineBuiltins counts KindBuiltin Candidate literals inside
	// Discover itself.
	inlineBuiltins int
}

// deriveDiscoverSourceSites parses the production Discover function out of
// provider.go, never from memory: a discovery source added to production
// without flowing through collectDirectory changes these counts and
// reddens the inventory test below.
func deriveDiscoverSourceSites(t *testing.T) discoverSourceSites {
	t.Helper()
	path := filepath.Join(mustProviderDir(t), "provider.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read provider.go: %v", err)
	}
	fileSet := token.NewFileSet()
	syntax, err := parser.ParseFile(fileSet, path, source, 0)
	if err != nil {
		t.Fatalf("parse provider.go: %v", err)
	}
	var body *ast.BlockStmt
	for _, decl := range syntax.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "Discover" && fn.Recv == nil {
			body = fn.Body
		}
	}
	if body == nil {
		t.Fatal("production func Discover not found in provider.go")
	}
	// Resolve the source-label variables assigned inside Discover (the
	// plugin loop builds `source := fmt.Sprintf("plugin_dirs[%d]", index)`
	// and passes the variable): each such assignment maps a name to its
	// class so call-site arguments written as identifiers stay classified.
	assigned := map[string]string{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.AssignStmt:
			for i, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && i < len(stmt.Rhs) {
					if class, ok := sprintfSourceClass(stmt.Rhs[i]); ok {
						assigned[ident.Name] = class
					}
				}
			}
		case *ast.ValueSpec:
			for i, name := range stmt.Names {
				if i < len(stmt.Values) {
					if class, ok := sprintfSourceClass(stmt.Values[i]); ok {
						assigned[name.Name] = class
					}
				}
			}
		}
		return true
	})
	var sites discoverSourceSites
	ast.Inspect(body, func(node ast.Node) bool {
		if literal, ok := node.(*ast.CompositeLit); ok {
			if ident, ok := literal.Type.(*ast.Ident); ok && ident.Name == "Candidate" {
				for _, elt := range literal.Elts {
					field, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := field.Key.(*ast.Ident)
					if !ok || key.Name != "kind" {
						continue
					}
					if value, ok := field.Value.(*ast.Ident); ok {
						switch value.Name {
						case "KindExternal":
							sites.inlineExternals++
						case "KindBuiltin":
							sites.inlineBuiltins++
						}
					}
				}
			}
			return true
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			switch fun.Name {
			case "collectDirectory":
				if len(call.Args) < 4 {
					t.Fatalf("collectDirectory call has %d args, want >= 4", len(call.Args))
				}
				sites.collectClasses = append(sites.collectClasses, classifySourceArg(t, call.Args[3], assigned))
			case "externalID":
				sites.directNameParses++
			}
		case *ast.SelectorExpr:
			if fun.Sel.Name == "ReadDir" {
				sites.directReads++
			}
		}
		return true
	})
	return sites
}

// sprintfSourceClass reports the source class of a fmt.Sprintf source
// expression: the plugin_dirs index format maps to the plugin_dirs class.
func sprintfSourceClass(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Sprintf" || len(call.Args) == 0 {
		return "", false
	}
	format, ok := call.Args[0].(*ast.BasicLit)
	if !ok || !strings.Contains(format.Value, "plugin_dirs[") {
		return "", false
	}
	return "plugin_dirs", true
}

func classifySourceArg(t *testing.T, expr ast.Expr, assigned map[string]string) string {
	t.Helper()
	if literal, ok := expr.(*ast.BasicLit); ok {
		if literal.Value == `"path"` {
			return "path"
		}
		t.Fatalf("collectDirectory source literal = %s, want \"path\"", literal.Value)
	}
	if _, ok := expr.(*ast.CallExpr); ok {
		if class, ok := sprintfSourceClass(expr); ok {
			return class
		}
		t.Fatalf("collectDirectory source expression is not the plugin_dirs index format")
	}
	if ident, ok := expr.(*ast.Ident); ok {
		if class, ok := assigned[ident.Name]; ok {
			return class
		}
		t.Fatalf("collectDirectory source variable %q has no classified assignment in Discover", ident.Name)
	}
	t.Fatalf("collectDirectory source argument has an unrecognized form %T", expr)
	return ""
}

func mustProviderDir(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return directory
}

// sourceClassOfTableRow maps a trust-gate table row name to its production
// source class, failing on names that match no derived class.
func sourceClassOfTableRow(t *testing.T, name string) string {
	t.Helper()
	if name == "path" {
		return "path"
	}
	var index int
	if _, err := fmt.Sscanf(name, "plugin_dirs[%d]", &index); err == nil {
		return "plugin_dirs"
	}
	t.Fatalf("trust-gate table row %q matches no production discovery source class", name)
	return ""
}

// TestTrustGateSourceInventoryIsDerivedFromDiscover requires a bijection
// between the production discovery sources and the trust-gate table rows:
// the source labels are enumerated from the Discover body with go/ast, so
// a fourth source added to production without a table row reddens, as does
// a table row naming a source production no longer has. The ratio below is
// a count of derived labels covered over derived labels found — the
// denominator comes from production, not from the table literal.
func TestTrustGateSourceInventoryIsDerivedFromDiscover(t *testing.T) {
	sites := deriveDiscoverSourceSites(t)

	if sites.directReads != 0 {
		t.Fatalf("Discover reads %d directories outside collectDirectory: every external source must flow through the gated path", sites.directReads)
	}
	if sites.directNameParses != 0 {
		t.Fatalf("Discover parses %d executable names outside collectDirectory", sites.directNameParses)
	}
	if sites.inlineExternals != 0 {
		t.Fatalf("Discover builds %d external candidates outside trustCandidate: every external must carry trust-time facts", sites.inlineExternals)
	}
	if sites.inlineBuiltins != 1 {
		t.Fatalf("Discover builds %d builtin candidate literals, want exactly 1", sites.inlineBuiltins)
	}
	if len(sites.collectClasses) != 2 {
		t.Fatalf("Discover has %d collectDirectory sites %v, want exactly 2", len(sites.collectClasses), sites.collectClasses)
	}
	derived := map[string]bool{}
	for _, class := range sites.collectClasses {
		derived[class] = true
	}
	var derivedLabels []string
	for class := range derived {
		derivedLabels = append(derivedLabels, class)
	}
	sort.Strings(derivedLabels)
	if fmt.Sprintf("%v", derivedLabels) != "[path plugin_dirs]" {
		t.Fatalf("derived discovery source classes = %v, want [path plugin_dirs]", derivedLabels)
	}

	rows := trustGateSources()
	covered := map[string]bool{}
	pluginIndexes := map[int]bool{}
	for _, row := range rows {
		class := sourceClassOfTableRow(t, row.name)
		if !derived[class] {
			t.Fatalf("trust-gate table row %q names a source production does not enumerate", row.name)
		}
		covered[class] = true
		var index int
		if _, err := fmt.Sscanf(row.name, "plugin_dirs[%d]", &index); err == nil {
			pluginIndexes[index] = true
		}
	}
	for class := range derived {
		if !covered[class] {
			t.Fatalf("production discovery source %q has no trust-gate table row", class)
		}
	}
	if !pluginIndexes[0] {
		t.Fatalf("indexed source plugin_dirs has no index-0 table row")
	}
	aboveZero := false
	for index := range pluginIndexes {
		if index > 0 {
			aboveZero = true
		}
	}
	if !aboveZero {
		t.Fatal("indexed source plugin_dirs is covered at index 0 only: an index-scoped gate bypass survives")
	}
	t.Logf("trust-gate source inventory: %d/%d derived source classes covered", len(covered), len(derived))
}
