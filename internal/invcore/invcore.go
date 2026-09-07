// Package invcore is the shared test-support core for production-derived
// inventories: production-file selection, fail-closed parsing, and the
// both-direction check harness.
//
// Every package inventory in this repository derives its denominator from
// production source instead of listing it, in three historical
// architectures for the same idea: terminalbackend resolves declared rows
// textually and refuses constructor aliasing outright; provhost executes
// every witness through the production entry point and keys on identifier
// names behind a separate call-site audit; provider instruments constructor
// vars at runtime in TestMain and audits the exercised file:line set
// against AST-derived sites. The weaker bound was the real bound, so this
// core carries the UNION of those directions, not the intersection:
//
//   - ScanProduction owns production-file selection: directory-derived,
//     every *.go except *_test.go, sorted, so a new production file is
//     scanned without anyone remembering it. Zero files is a failure,
//     never a vacuous pass.
//   - MustParse owns fail-closed parsing: an unparseable file fails, never
//     skips. A file the derivation cannot read proves nothing.
//   - DiffSets owns the both-direction harness: derived-without-row
//     (unregistered site) AND row-without-site (orphan row) fail together.
//     The forward direction alone passes vacuously on a truncated
//     derivation.
//   - AuditConstructorReferences owns the alias-bypass union: constructor
//     references outside direct-call position fail outright
//     (terminalbackend direction), import aliases and dot imports resolve
//     against the imported path instead of the local spelling (the shape
//     that walked through every identifier-keyed gate in this repository),
//     and var bindings are references, not calls (provider direction made
//     static). Anything the audit cannot classify — a dot import of a
//     watched path, an unresolved shadowing — fails closed as
//     unclassifiable, never pruned.
//   - SiteRecorder owns the runtime direction (provider/provhost shape):
//     instrumented constructor vars record the production file:line behind
//     every exercised refusal, and the audit requires the exercised set to
//     equal the derived site set in both directions.
//
// All checks come in two spellings: a pure func returning failures for
// negative tests and control plants, and a Must wrapper that fails the
// suite. A gate that can only fatal cannot be planted against, and a plant
// that cannot run proves nothing.
package invcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ProductionFile is one parsed production source: basename, full path,
// raw bytes, and syntax parsed with a directory-shared FileSet.
type ProductionFile struct {
	Name   string
	Path   string
	Source []byte
	Syntax *ast.File
	File   *token.File
}

// ScanProduction reads every production file of the directory at dir:
// every *.go except *_test.go, in sorted order. It returns the parsed
// files with their shared FileSet, plus a failure string for every
// fail-closed condition: an unreadable directory, an unreadable file, an
// unparseable file, or zero production files. An empty failure slice with
// an empty file list is impossible by construction.
func ScanProduction(dir string) ([]ProductionFile, *token.FileSet, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, []string{"scan production files: read " + dir + ": " + err.Error()}
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, nil, []string{"scanned zero production files in " + dir + "; the scanner is broken, not the package"}
	}
	fileSet := token.NewFileSet()
	var files []ProductionFile
	var failures []string
	for _, name := range names {
		path := filepath.Join(dir, name)
		source, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, "scan production files: read "+path+": "+err.Error())
			continue
		}
		syntax, err := parser.ParseFile(fileSet, path, source, parser.ParseComments)
		if err != nil {
			failures = append(failures, "scan production files: parse "+path+": "+err.Error()+"; an unparseable derivation proves nothing")
			continue
		}
		files = append(files, ProductionFile{
			Name:   name,
			Path:   path,
			Source: source,
			Syntax: syntax,
			File:   fileSet.File(syntax.Pos()),
		})
	}
	if len(files) == 0 && len(failures) == 0 {
		failures = append(failures, "derived zero parsed production files in "+dir+"; the scanner is broken, not the package")
	}
	return files, fileSet, failures
}

// ParseSource parses one synthetic source (control plants, classifier
// unit probes) with fail-closed semantics: the failure string is empty on
// success and names the cause otherwise.
func ParseSource(filename string, source []byte) (*ast.File, *token.FileSet, string) {
	return ParseBytes(filename, source, parser.ParseComments)
}

// ParseBytes parses one source with the given parser mode and fail-closed
// semantics: the failure string is empty on success and names the cause
// otherwise. Inventories with a non-standard traversal (recursive walk,
// content prefilter, imports-only scan) keep their traversal and route
// parsing here, so an unparseable file fails identically everywhere.
func ParseBytes(filename string, source []byte, mode parser.Mode) (*ast.File, *token.FileSet, string) {
	fileSet := token.NewFileSet()
	syntax, err := parser.ParseFile(fileSet, filename, source, mode)
	if err != nil {
		return nil, nil, "parse " + filename + ": " + err.Error() + "; an unparseable derivation proves nothing"
	}
	return syntax, fileSet, ""
}

// DiffSets is the both-direction harness over string keys: every derived
// key without a row is unregistered, every row without a derived key is
// orphaned. Both lists are sorted. Empty derived input is itself a
// failure: a domain that silently derives nothing is not a measurement,
// and the forward check would otherwise pass vacuously.
func DiffSets(derived, rows map[string]struct{}) (unregistered, orphaned, failures []string) {
	if len(derived) == 0 {
		failures = append(failures, "derived zero sites; the scanner is broken, not the package")
	}
	for key := range derived {
		if _, ok := rows[key]; !ok {
			unregistered = append(unregistered, key)
		}
	}
	for key := range rows {
		if _, ok := derived[key]; !ok {
			orphaned = append(orphaned, key)
		}
	}
	sort.Strings(unregistered)
	sort.Strings(orphaned)
	return unregistered, orphaned, failures
}

// ConstructorSpec declares the refusal constructors one audit watches:
// package-local bare identifiers (mismatchf) and qualified constructors
// matched by IMPORT PATH and member name (errors.New), so an import alias
// cannot change what the audit sees.
type ConstructorSpec struct {
	// Local names package-local constructor identifiers.
	Local map[string]bool
	// Qualified maps an import path to its watched member names:
	// {"errors": {"New": true}, "fmt": {"Errorf": true}}. A member name
	// with a trailing "*" matches by prefix ("Errorf*" covers Errorf
	// and its Errorf-prefixed kin).
	Qualified map[string]map[string]bool
}

// qualifiedWatches reports whether member is watched under path,
// honoring trailing-"*" prefix entries.
func qualifiedWatches(spec ConstructorSpec, path, member string) bool {
	members, ok := spec.Qualified[path]
	if !ok {
		return false
	}
	if members[member] {
		return true
	}
	for watched := range members {
		if strings.HasSuffix(watched, "*") && strings.HasPrefix(member, strings.TrimSuffix(watched, "*")) {
			return true
		}
	}
	return false
}

// AuditConstructorReferences fails every constructor reference outside
// direct-call position: aliases, var bindings, arguments, struct fields,
// and shadows. Import aliases resolve through the file's import specs, so
// `import errs "errors"; errs.New("…")` attributes to errors.New and
// `alias := errs.New` fails like any other binding. A dot import of a
// watched path or any other shape the audit cannot classify fails closed
// as unclassifiable instead of passing silently. Direct calls and the
// constructors' own declarations are the only allowed positions.
func AuditConstructorReferences(syntax *ast.File, fileSet *token.FileSet, display string, spec ConstructorSpec) []string {
	imports := resolveImports(syntax, spec)
	for path := range spec.Qualified {
		if dot, ok := imports.dot[path]; ok && dot {
			return []string{display + ": dot-imports watched path " + strconv.Quote(path) + "; the audit cannot attribute bare " + constructorMembers(spec.Qualified[path]) + ", qualify the import or widen the audit"}
		}
	}
	allowed := map[token.Pos]bool{}
	ast.Inspect(syntax, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if ok && spec.Local[declaration.Name.Name] {
			// The declaration name itself is allowed, but the body
			// still scans: a direct call nested inside the funnel
			// body is attributable and must stay allowed.
			allowed[declaration.Name.Pos()] = true
			return true
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && spec.Local[ident.Name] {
			allowed[ident.Pos()] = true
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if receiver, ok := selector.X.(*ast.Ident); ok {
				if qualifiedWatches(spec, imports.pathOf(receiver.Name), selector.Sel.Name) {
					allowed[selector.Sel.Pos()] = true
				}
			}
		}
		return true
	})
	var failures []string
	ast.Inspect(syntax, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if ok && spec.Local[declaration.Name.Name] {
			// Descend: the declaration name is allowlisted by
			// position, but references nested in the funnel body
			// still audit.
			return true
		}
		ident, ok := node.(*ast.Ident)
		if !ok || allowed[ident.Pos()] {
			return true
		}
		if spec.Local[ident.Name] {
			if _, declared := imports.names[ident.Name]; declared {
				failures = append(failures, display+":"+positionOf(fileSet, ident.Pos())+": constructor "+strconv.Quote(ident.Name)+" shadowed by an import or declaration the audit cannot classify; rename it or widen the audit")
				return true
			}
			failures = append(failures, display+":"+positionOf(fileSet, ident.Pos())+": constructor "+strconv.Quote(ident.Name)+" referenced outside direct-call position; an aliased refusal cannot be inventoried, call it directly")
			return true
		}
		return true
	})
	ast.Inspect(syntax, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || allowed[selector.Sel.Pos()] {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if qualifiedWatches(spec, imports.pathOf(receiver.Name), selector.Sel.Name) {
			failures = append(failures, display+":"+positionOf(fileSet, selector.Sel.Pos())+": constructor "+strconv.Quote(receiver.Name+"."+selector.Sel.Name)+" referenced outside direct-call position; an aliased construction cannot be inventoried, call it directly")
		}
		return true
	})
	sort.Strings(failures)
	return failures
}

// FunPosition reports the audited position of a call's function: the
// identifier for bare calls, the selector member for qualified calls.
// WatchedCallPositions keys on exactly these positions.
func FunPosition(call *ast.CallExpr) (token.Pos, bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Pos(), true
	case *ast.SelectorExpr:
		return fun.Sel.Pos(), true
	default:
		return token.NoPos, false
	}
}

// WatchedCallPositions returns the audited positions of every direct
// call to a watched constructor in syntax: the attribution set. Callers
// use it to attribute each watched construction to its enclosing
// function (an allowlist scan), while AuditConstructorReferences owns
// the alias direction over the same spec.
func WatchedCallPositions(syntax *ast.File, spec ConstructorSpec) map[token.Pos]bool {
	imports := resolveImports(syntax, spec)
	positions := map[token.Pos]bool{}
	ast.Inspect(syntax, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if ok && spec.Local[declaration.Name.Name] {
			return true
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && spec.Local[ident.Name] {
			positions[ident.Pos()] = true
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if receiver, ok := selector.X.(*ast.Ident); ok {
				if qualifiedWatches(spec, imports.pathOf(receiver.Name), selector.Sel.Name) {
					positions[selector.Sel.Pos()] = true
				}
			}
		}
		return true
	})
	return positions
}

// importResolution maps the file's local import names to paths.
type importResolution struct {
	names map[string]string
	dot   map[string]bool
}

func (resolution importResolution) pathOf(local string) string {
	return resolution.names[local]
}

// resolveImports reads the file's import specs into local-name to path
// bindings. Explicit aliases, blank imports, and default base names all
// resolve; only dot imports are unresolvable, and the caller fails them
// closed.
func resolveImports(syntax *ast.File, spec ConstructorSpec) importResolution {
	resolution := importResolution{names: map[string]string{}, dot: map[string]bool{}}
	for _, declaration := range syntax.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.IMPORT {
			continue
		}
		for _, item := range general.Specs {
			loaded, ok := item.(*ast.ImportSpec)
			if !ok {
				continue
			}
			path, err := strconv.Unquote(loaded.Path.Value)
			if err != nil {
				continue
			}
			if loaded.Name != nil && loaded.Name.Name == "." {
				resolution.dot[path] = true
				continue
			}
			local := importBaseName(path)
			if loaded.Name != nil {
				local = loaded.Name.Name
			}
			if local == "_" {
				continue
			}
			resolution.names[local] = path
		}
	}
	return resolution
}

// importBaseName renders the default local name of an import path.
func importBaseName(path string) string {
	if index := strings.LastIndex(path, "/"); index >= 0 {
		return path[index+1:]
	}
	return path
}

// constructorMembers joins watched member names for diagnostics.
func constructorMembers(members map[string]bool) string {
	var names []string
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// positionOf renders line:column for diagnostics; never empty.
func positionOf(fileSet *token.FileSet, position token.Pos) string {
	located := fileSet.Position(position)
	if !located.IsValid() {
		return "?:?"
	}
	return strconv.Itoa(located.Line) + ":" + strconv.Itoa(located.Column)
}
