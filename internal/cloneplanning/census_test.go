package cloneplanning

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// This file pins the UTF-8 entry census structurally: the test
// parses the package's production files, lists every exported
// function or method with a text-carrying parameter (a string, a
// []byte, or a struct type containing either, followed
// transitively), and asserts that set equals the refusal table's
// key set exactly. A new exported text entry with no refusal row
// fails here; the control plant proves it (log rides the evidence
// tar). A reflection cross-check proves the classification
// against the real parameter types, and the Marshal guard proves
// no text value reaches encoding/json.Marshal before the shared
// validText gate.

// productionFiles returns the parsed production files of this
// package: every .go file except the tests themselves.
func productionFiles(t *testing.T) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatalf("no production files found")
	}
	return files
}

// localTypes collects the package's named type declarations for
// transitive resolution.
func localTypes(files []*ast.File) map[string]ast.Expr {
	types := map[string]ast.Expr{}
	for _, file := range files {
		for _, decl := range file.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				types[typed.Name.Name] = typed.Type
			}
		}
	}
	return types
}

var nonTextIdents = map[string]bool{
	"bool": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
}

// isTextType reports whether the parameter type can carry text:
// a string, a []byte, or a struct (or named type) containing
// either, followed transitively through the package's own
// declarations. External qualified types and open shapes (maps,
// any) classify as text conservatively; the reflection
// cross-check proves the actual reachability.
func isTextType(expr ast.Expr, types map[string]ast.Expr, visited map[string]bool) bool {
	switch typed := expr.(type) {
	case *ast.Ident:
		if typed.Name == "string" {
			return true
		}
		if nonTextIdents[typed.Name] {
			return false
		}
		if visited[typed.Name] {
			return false
		}
		underlying, ok := types[typed.Name]
		if !ok {
			return true
		}
		visited[typed.Name] = true
		return isTextType(underlying, types, visited)
	case *ast.SelectorExpr:
		return true
	case *ast.ArrayType:
		if typed.Len == nil {
			if elt, ok := typed.Elt.(*ast.Ident); ok && (elt.Name == "byte" || elt.Name == "uint8") {
				return true
			}
		}
		return isTextType(typed.Elt, types, visited)
	case *ast.MapType:
		return true
	case *ast.StarExpr:
		return isTextType(typed.X, types, visited)
	case *ast.InterfaceType:
		return true
	case *ast.Ellipsis:
		return isTextType(typed.Elt, types, visited)
	case *ast.StructType:
		for _, field := range typed.Fields.List {
			if isTextType(field.Type, types, visited) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// censusEntry is one exported function or method with its
// text-carrying verdict.
type censusEntry struct {
	name     string
	hasText  bool
	receiver string
}

// inspectEntryCensus parses the production files and classifies
// every exported function or method by its parameter types.
func inspectEntryCensus(t *testing.T) []censusEntry {
	t.Helper()
	files := productionFiles(t)
	types := localTypes(files)
	var out []censusEntry
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !ast.IsExported(fn.Name.Name) {
				continue
			}
			entry := censusEntry{name: fn.Name.Name}
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				entry.receiver = "method"
				entry.name = "method:" + fn.Name.Name
			}
			if fn.Type.Params != nil {
				for _, field := range fn.Type.Params.List {
					if isTextType(field.Type, types, map[string]bool{}) {
						entry.hasText = true
						break
					}
				}
			}
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// exportedEntryFuncs pins the package's exported function set for
// the reflection cross-check. The census test asserts these names
// equal the AST-derived exported set, so a new entry breaks
// loudly here instead of slipping past the cross-check.
var exportedEntryFuncs = []struct {
	name  string
	value any
}{
	{"SelectStrategy", SelectStrategy},
	{"SelectProfile", SelectProfile},
	{"ClassifyItem", ClassifyItem},
	{"PlanItem", PlanItem},
	{"PlanSession", PlanSession},
	{"ProjectVisibleText", ProjectVisibleText},
	{"EscapeVisibleText", EscapeVisibleText},
	{"PlanTargetEffects", PlanTargetEffects},
}

// reflectReachesText reports whether text (a string or a []byte)
// is reachable from the type through fields, elements, keys, and
// pointees. Interface members count as text conservatively: an
// any-typed member can carry a string.
func reflectReachesText(typ reflect.Type, visited map[reflect.Type]bool) bool {
	if typ == nil {
		return false
	}
	if typ.Kind() == reflect.String {
		return true
	}
	if typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 {
		return true
	}
	switch typ.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer:
		return reflectReachesText(typ.Elem(), visited)
	case reflect.Slice, reflect.Array:
		return reflectReachesText(typ.Elem(), visited)
	case reflect.Map:
		return reflectReachesText(typ.Key(), visited) || reflectReachesText(typ.Elem(), visited)
	case reflect.Struct:
		if visited[typ] {
			return false
		}
		visited[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			if reflectReachesText(typ.Field(i).Type, visited) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// reflectEntryHasText reports whether any parameter of the
// function value can carry text.
func reflectEntryHasText(value any) bool {
	typ := reflect.TypeOf(value)
	for i := 0; i < typ.NumIn(); i++ {
		if reflectReachesText(typ.In(i), map[reflect.Type]bool{}) {
			return true
		}
	}
	return false
}

func TestExportedTextEntryCensus(t *testing.T) {
	// The census derives from the production AST: a new exported
	// text entry with no refusal row fails here. Methods fail
	// explicitly: none exist, and one would need deliberate
	// table handling.
	census := inspectEntryCensus(t)
	censused := map[string]bool{}
	for _, entry := range census {
		if entry.receiver != "" {
			t.Fatalf("exported method %q needs explicit refusal-table handling", entry.name)
		}
		censused[entry.name] = entry.hasText
	}
	// The reflection list tracks the AST set exactly.
	listed := map[string]bool{}
	for _, entry := range exportedEntryFuncs {
		listed[entry.name] = true
	}
	for name := range censused {
		if !listed[name] {
			t.Fatalf("exported %s is missing from the reflection cross-check list", name)
		}
	}
	for name := range listed {
		if _, ok := censused[name]; !ok {
			t.Fatalf("reflection list names %s, which the AST census does not export", name)
		}
	}
	// The reflection cross-check proves the AST classification
	// against the real parameter types, both directions: a text
	// entry the AST missed fails as loudly as a non-text entry
	// it misclassified.
	for _, entry := range exportedEntryFuncs {
		if reflectEntryHasText(entry.value) != censused[entry.name] {
			t.Fatalf("entry %s: AST text=%v, reflection text=%v", entry.name, censused[entry.name], reflectEntryHasText(entry.value))
		}
	}
	// The external struct grounding: ClassifyItem's CanonicalEvent
	// parameter actually carries text (a string Kind field), so
	// the conservative external classification is not vacuous.
	eventType := reflect.TypeOf(clonebundle.CanonicalEvent{})
	kindField, ok := eventType.FieldByName("Kind")
	if !ok || kindField.Type.Kind() != reflect.String {
		t.Fatalf("CanonicalEvent carries no string Kind field")
	}
	// The refusal table covers the censused set exactly: a
	// censused entry missing from the table fails, and an
	// unknown table row fails.
	table := map[string]bool{}
	for _, row := range utf8RefusalTable {
		if table[row.entry] {
			t.Fatalf("refusal table duplicates %s", row.entry)
		}
		table[row.entry] = true
	}
	for name, hasText := range censused {
		if hasText && !table[name] {
			t.Fatalf("censused text entry %s is missing from the refusal table", name)
		}
	}
	for name := range table {
		hasText, ok := censused[name]
		if !ok {
			t.Fatalf("refusal table names %s, which is not an exported entry", name)
		}
		if !hasText {
			t.Fatalf("refusal table names non-text entry %s", name)
		}
	}
	// The exclusion pin: SelectStrategy is the only exported
	// entry without text (four booleans), so the census is not
	// silently empty.
	excluded := []string{}
	for name, hasText := range censused {
		if !hasText {
			excluded = append(excluded, name)
		}
	}
	if len(excluded) != 1 || excluded[0] != "SelectStrategy" {
		t.Fatalf("non-text entries %q, want exactly [SelectStrategy]", excluded)
	}
	if len(censused) != 8 {
		t.Fatalf("census carries %d exported entries, want 8", len(censused))
	}
}

// marshalCallSite is one encoding/json.Marshal call with its
// containing function and statement order evidence.
type marshalCallSite struct {
	function   string
	validator  int
	marshalIdx int
}

// inspectMarshalCallSites resolves the encoding/json import in
// each production file and locates every Marshal call with the
// validText call ordering inside its function. Statement indices
// are top-level statement positions within the function body.
func inspectMarshalCallSites(t *testing.T) []marshalCallSite {
	t.Helper()
	var sites []marshalCallSite
	for _, file := range productionFiles(t) {
		jsonName := ""
		dotImport := false
		for _, imported := range file.Imports {
			if strings.Trim(imported.Path.Value, `"`) != "encoding/json" {
				continue
			}
			if imported.Name == nil {
				jsonName = "json"
			} else if imported.Name.Name == "." {
				dotImport = true
			} else {
				jsonName = imported.Name.Name
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			isMarshalCall := func(call *ast.CallExpr) bool {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					ident, ok := sel.X.(*ast.Ident)
					return ok && jsonName != "" && ident.Name == jsonName && sel.Sel.Name == "Marshal"
				}
				if ident, ok := call.Fun.(*ast.Ident); ok {
					return dotImport && ident.Name == "Marshal"
				}
				return false
			}
			validator := -1
			marshalIdx := -1
			for index, stmt := range fn.Body.List {
				foundValidator := false
				foundMarshal := false
				ast.Inspect(stmt, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					if isMarshalCall(call) {
						foundMarshal = true
					}
					if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "validText" {
						foundValidator = true
					}
					return true
				})
				if foundValidator && validator < 0 {
					validator = index
				}
				if foundMarshal && marshalIdx < 0 {
					marshalIdx = index
				}
			}
			if marshalIdx >= 0 {
				sites = append(sites, marshalCallSite{function: fn.Name.Name, validator: validator, marshalIdx: marshalIdx})
			}
		}
	}
	return sites
}

func TestJSONMarshalFollowsValidator(t *testing.T) {
	// Structural pin: the only encoding/json.Marshal call in the
	// package's production files is in EscapeVisibleText, and the
	// shared validText gate runs in an earlier statement, so no
	// text value reaches the encoder unvalidated. A second
	// Marshal call site, a missing gate, or a reordering reddens
	// here; the behavioral refusal table proves the refusal
	// itself. Control-plant logs ride the evidence tar.
	sites := inspectMarshalCallSites(t)
	if len(sites) != 1 {
		functions := []string{}
		for _, site := range sites {
			functions = append(functions, site.function)
		}
		t.Fatalf("json.Marshal call sites %q, want exactly [EscapeVisibleText]", functions)
	}
	site := sites[0]
	if site.function != "EscapeVisibleText" {
		t.Fatalf("json.Marshal call site %q, want EscapeVisibleText", site.function)
	}
	if site.validator < 0 {
		t.Fatalf("EscapeVisibleText calls json.Marshal with no validText gate")
	}
	if site.validator >= site.marshalIdx {
		t.Fatalf("EscapeVisibleText validates at statement %d, marshals at %d: gate must run first", site.validator, site.marshalIdx)
	}
}
