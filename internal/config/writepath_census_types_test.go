package config

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// censusTypedViolations is the authoritative write-path gate. It resolves
// package, method, alias-import and function-value identity through go/types;
// spelling a package "os" or hiding a writer behind a closure does not evade
// the gate. The loader evaluates the current Go build context; build-tagged
// or platform-specific sources require a separate invocation under that
// context. Unknown indirect callees fail closed unless they are inside one of
// the exact censused definitions. The older syntax walker remains in
// writepath_census_test.go as a small reference implementation for historical
// evidence, but all tests use this symbol-aware implementation.
func censusTypedViolations(t *testing.T, dirs ...string) []string {
	t.Helper()
	var units []censusTypeUnit
	for _, dir := range dirs {
		unit, err := loadCensusTypeUnit(dir)
		if err != nil {
			t.Fatalf("load write-path census package %s: %v", dir, err)
		}
		units = append(units, unit)
	}
	gated := censusGatedFunctions(units)
	var violations []string
	for _, unit := range units {
		scanner := newCensusTypeScanner(unit, gated)
		violations = append(violations, scanner.scan()...)
	}
	sort.Strings(violations)
	return violations
}

type censusTypeUnit struct {
	dir   string
	fset  *token.FileSet
	files []*ast.File
	info  *types.Info
	pkg   *types.Package
}

func loadCensusTypeUnit(dir string) (censusTypeUnit, error) {
	root := censusModuleRoot()
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return censusTypeUnit{}, err
	}
	if absolute == root || strings.HasPrefix(absolute, root+string(filepath.Separator)) {
		loaded, err := packages.Load(&packages.Config{
			Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps,
			Dir:   absolute,
			Tests: false,
		}, ".")
		if err != nil {
			return censusTypeUnit{}, err
		}
		if len(loaded) != 1 {
			return censusTypeUnit{}, fmt.Errorf("expected one package, got %d", len(loaded))
		}
		pkg := loaded[0]
		if len(pkg.Errors) != 0 {
			return censusTypeUnit{}, fmt.Errorf("type-check errors: %v", pkg.Errors)
		}
		if pkg.Fset == nil || pkg.Types == nil || pkg.TypesInfo == nil {
			return censusTypeUnit{}, fmt.Errorf("incomplete go/packages type information")
		}
		return censusTypeUnit{dir: absolute, fset: pkg.Fset, files: pkg.Syntax, info: pkg.TypesInfo, pkg: pkg.Types}, nil
	}

	entries, err := filepath.Glob(filepath.Join(absolute, "*.go"))
	if err != nil {
		return censusTypeUnit{}, err
	}
	sort.Strings(entries)
	fset := token.NewFileSet()
	var files []*ast.File
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return censusTypeUnit{}, err
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		return censusTypeUnit{}, fmt.Errorf("no production Go files")
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("censuscontrol", fset, files, info)
	if err != nil {
		return censusTypeUnit{}, err
	}
	return censusTypeUnit{dir: absolute, fset: fset, files: files, info: info, pkg: pkg}, nil
}

func censusModuleRoot() string {
	_, file, _, ok := runtimeCensusCaller()
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// Kept as a tiny indirection so the production census code remains easy to
// execute in a test without importing runtime in the old syntax-only file.
var runtimeCensusCaller = func() (uintptr, string, int, bool) {
	return runtimeCaller(0)
}

func runtimeCaller(skip int) (uintptr, string, int, bool) {
	return runtime.Caller(skip + 1)
}

type censusFunction struct {
	name     string
	pkgPath  string
	receiver string
}

func censusGatedFunctions(units []censusTypeUnit) map[*types.Func]string {
	result := map[*types.Func]string{}
	for _, unit := range units {
		for _, file := range unit.files {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Name == nil {
					continue
				}
				object, ok := unit.info.Defs[function.Name].(*types.Func)
				if !ok || !censusGatedName(object.Name()) {
					continue
				}
				result[object] = object.Name()
			}
		}
	}
	return result
}

func censusGatedName(name string) bool {
	if censusPairHelpers[name] || censusGatedIdents[name] {
		return true
	}
	if _, ok := censusTrustHelpers[name]; ok {
		return true
	}
	if _, ok := censusCustodyHelpers[name]; ok {
		return true
	}
	return false
}

type censusTypeScanner struct {
	unit                    censusTypeUnit
	gated                   map[*types.Func]string
	allowedIndirectObjects  map[*types.Func]map[types.Object]bool
	allowedInterfaceMethods map[*types.Func]map[types.Object]bool
	allowedInterfaceSites   map[*types.Func]map[censusInterfaceCallSite]bool
	allowedFunctionValues   map[types.Object]bool
	allowedInitializers     map[string]bool
	functionDecls           map[*types.Func]*ast.FuncDecl
	result                  []string
}

type censusIndirectPolicy struct {
	parameters map[string]bool
	locals     map[string]bool
}

// censusInterfaceMethod identifies one explicitly required interface edge.
// The scanner resolves this descriptor to the loaded selection's concrete
// *types.Func object and stores that object in the per-definition allowlist.
// Calls are admitted only when that object and its exact source call site are
// both present in the reviewed inventory, never by a boolean attached to the
// containing definition or by a method-name heuristic.
type censusInterfaceMethod struct {
	pkgPath  string
	receiver string
	name     string
}

func censusInterfaceMethods(pkgPath, receiver string, names ...string) []censusInterfaceMethod {
	methods := make([]censusInterfaceMethod, 0, len(names))
	for _, name := range names {
		methods = append(methods, censusInterfaceMethod{pkgPath: pkgPath, receiver: receiver, name: name})
	}
	return methods
}

func censusInterfaceMethodSet(sets ...[]censusInterfaceMethod) []censusInterfaceMethod {
	var methods []censusInterfaceMethod
	for _, set := range sets {
		methods = append(methods, set...)
	}
	return methods
}

// censusInterfaceCallSite is a source-positioned, callee-specific edge. The
// position is deliberately part of the policy: adding the same interface
// method at another position is a new edge and is rejected until it is
// reviewed into this inventory. The expected method descriptor prevents a
// source edit at an existing position from inheriting another method's
// authorization.
type censusInterfaceCallSite struct {
	file   string
	line   int
	column int
	method censusInterfaceMethod
}

func censusInterfaceCallSites(pkgPath, receiver, name string, locations ...any) []censusInterfaceCallSite {
	if len(locations)%3 != 0 {
		panic("census interface call-site locations must be file/line/column triples")
	}
	sites := make([]censusInterfaceCallSite, 0, len(locations)/3)
	for index := 0; index < len(locations); index += 3 {
		file, fileOK := locations[index].(string)
		line, lineOK := locations[index+1].(int)
		column, columnOK := locations[index+2].(int)
		if !fileOK || !lineOK || !columnOK {
			panic("census interface call-site locations have invalid types")
		}
		sites = append(sites, censusInterfaceCallSite{
			file: file, line: line, column: column,
			method: censusInterfaceMethod{pkgPath: pkgPath, receiver: receiver, name: name},
		})
	}
	return sites
}

func censusInterfaceCallSiteSet(sets ...[]censusInterfaceCallSite) []censusInterfaceCallSite {
	var sites []censusInterfaceCallSite
	for _, set := range sets {
		sites = append(sites, set...)
	}
	return sites
}

// These are the only production indirect edges that the scanner may admit.
// The descriptor is used only while loading the package; the scanner stores
// and compares the resulting *types.Func/*types.Var objects, so a same-named
// function, receiver, parameter or interface in another package cannot inherit
// an exemption. Every other unresolved/function-value/type-parameter edge is
// a violation.
var censusIndirectPolicies = map[censusFunction]censusIndirectPolicy{
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeAll"}:       {},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "replaceDurably"}: {},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempFile"}: {
		locals: map[string]bool{"clean": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempReplace"}: {},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "applyV4"}: {
		parameters: map[string]bool{"openStore": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "rollbackV4"}: {
		parameters: map[string]bool{"openStore": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "validateRootKinds"}: {
		parameters: map[string]bool{"stat": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "admitSSHArguments"}: {
		locals: map[string]bool{"rule": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "admitSSHOption"}: {
		locals: map[string]bool{"rule": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "sshFlagValueRule"}: {
		parameters: map[string]bool{"permits": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "writeAll"}: {},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "withExclusiveHold"}: {
		parameters: map[string]bool{"fn": true},
		locals:     map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "fileLock", name: "exclusiveHold"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithExclusiveHold"}: {
		parameters: map[string]bool{"fn": true},
		locals:     map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithExclusiveHoldForConfig"}: {
		parameters: map[string]bool{"fn": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithSharedSnapshot"}: {
		parameters: map[string]bool{"fn": true},
		locals:     map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithSharedSnapshotForConfig"}: {
		parameters: map[string]bool{"fn": true},
		locals:     map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "JointCommit"}: {
		parameters: map[string]bool{"revalidate": true, "replace": true, "restore": true},
		locals:     map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "transactLocked"}: {
		parameters: map[string]bool{"fn": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithMutationAuthorization"}: {
		parameters: map[string]bool{"boundary": true},
		locals:     map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "AuthorizeDispatch"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "EnsureConfigBinding"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "ensureConfigBindingLocked"}: {
		locals: map[string]bool{"unlockBootstrap": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "ValidateBoundPaths"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "ValidateBoundConfigPath"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readSnapshot"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "lockForConvergedRead"}: {
		locals: map[string]bool{"exclusiveUnlock": true, "unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "transact"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "Initialize"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "Recover"}: {
		locals: map[string]bool{"unlock": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "decodeEntryString"}: {
		parameters: map[string]bool{"accept": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "compensateLocked"}: {
		parameters: map[string]bool{"restore": true},
		locals:     map[string]bool{"failed": true},
	},
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "resolveFailedReplace"}: {
		parameters: map[string]bool{"restore": true},
	},
}

// censusRequiredInterfaceMethods is the explicit interface/type-parameter
// edge inventory for production definitions. Each descriptor is resolved to
// the loaded selection object and stored in a per-definition map. Adding a
// new method, receiver interface, or call site therefore requires an
// explicit review entry; a spelling match or an enclosing-function boolean
// cannot authorize it.
var censusRequiredInterfaceMethods = map[censusFunction][]censusInterfaceMethod{
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "Load"}:                                      censusInterfaceMethods("io/fs", "FileInfo", "Mode"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "loadAbsentConfig"}:                          censusInterfaceMethods("io/fs", "FileInfo", "IsDir"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "validateRootKinds"}:                         censusInterfaceMethods("io/fs", "FileInfo", "IsDir"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "replaceDurably"}:                            censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Stat", "Link", "Remove", "ReadFile", "Rename"), censusInterfaceMethods("io/fs", "FileInfo", "Mode")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempFile"}:                             censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "CreateTemp", "Remove"), censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationFile", "Name", "Close", "Chmod", "Sync")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeAll"}:                                  censusInterfaceMethods("io", "Writer", "Write"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "syncDirectory"}:                             censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "OpenDirectory"), censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationDirectory", "Sync", "Close")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "rollbackV4"}:                                censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempReplace"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Rename", "Remove"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "restorePresentEmptyMaps"}:                   censusInterfaceMethods("reflect", "Type", "Kind", "Key", "Field"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "validateTerminal"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/config", "BackendSettingsValidator", "ValidateBackendSettings"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "writeAll"}:                               censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "interface{Write([]byte) (int, error)}", "Write"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readCommitted"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "loadBindingAt"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "ensureOwnerDir"}:                         censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "MkdirAll", "Chmod"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyOwnerDir"}:                         censusInterfaceMethods("io/fs", "FileInfo", "IsDir", "Mode"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyLockFile"}:                         censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat"), censusInterfaceMethods("io/fs", "FileInfo", "Mode")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyOwnerFile"}:                        censusInterfaceMethods("io/fs", "FileInfo", "Mode"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "platformModeOK"}:                         censusInterfaceMethods("io/fs", "FileInfo", "Mode"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyOwner"}:                            censusInterfaceMethods("io/fs", "FileInfo", "Sys"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "writeCustodyFile"}:                       censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateTemp", "Remove", "Rename", "Lstat"), censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Name", "Close", "Sync")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "cleanupCredentialDirectory"}:             censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readOwnerFile"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "JointCommit"}:         censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "resolveFailedReplace"}:                   censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "compensateLocked"}:                       censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readCompensationMarker"}:                 censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "recoverLocked"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "markerPresent"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "removeMarkerLocked"}:                     censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "openLock"}:                               censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "CreateExclusive", "Chmod"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "checkP256Point"}:                         censusInterfaceMethods("crypto/elliptic", "Curve", "IsOnCurve"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readSnapshotLocked"}:                     censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "loadCommitted"}:                          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "ReadFile"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "Initialize"}:          censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "commitDocument"}:                         censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateTemp", "Chmod", "Rename"), censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Name", "Sync", "Close")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "commitConfigBinding"}: censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateTemp", "Chmod", "Rename"), censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Name", "Sync", "Close")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "cleanupStagedFile"}:                      censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "syncDir"}:                                censusInterfaceMethodSet(censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "OpenDirectory"), censusInterfaceMethods("github.com/relux-works/agent-session-manager/internal/hosttrust", "SyncedDirectory", "Sync", "Close")),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "TrustStore", name: "Format"}:         censusInterfaceMethods("fmt", "State", "Write"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "CredentialEntry", name: "Format"}:    censusInterfaceMethods("fmt", "State", "Write"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "AuthRequest", name: "Format"}:        censusInterfaceMethods("fmt", "State", "Write"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Snapshot", name: "Format"}:           censusInterfaceMethods("fmt", "State", "Write"),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "EnrollmentMaterial", name: "Format"}: censusInterfaceMethods("fmt", "State", "Write"),
}

// Exact source locations for every admitted indirect interface call in the
// production config and hosttrust packages. This is intentionally a closed
// inventory rather than a list of methods per enclosing function: a new call
// site, escaped method value, or moved call must fail the gate until its
// production justification and hold relationship are reviewed.
var censusRequiredInterfaceCallSites = map[censusFunction][]censusInterfaceCallSite{
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "Load"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "Mode", "loader.go", 260, 6),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "loadAbsentConfig"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "IsDir", "loader.go", 326, 6),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "replaceDurably"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Link", "migration.go", 417, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "ReadFile", "migration.go", 422, 24),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Remove", "migration.go", 419, 8, "migration.go", 425, 8, "migration.go", 429, 12, "migration.go", 443, 7, "migration.go", 459, 9),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Rename", "migration.go", 442, 12, "migration.go", 452, 18),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Stat", "migration.go", 406, 15, "migration.go", 423, 20),
		censusInterfaceCallSites("io/fs", "FileInfo", "Mode", "migration.go", 410, 6, "migration.go", 424, 78, "migration.go", 438, 100, "migration.go", 450, 104),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "restorePresentEmptyMaps"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("reflect", "Type", "Field", "schema.go", 562, 17),
		censusInterfaceCallSites("reflect", "Type", "Key", "schema.go", 558, 38, "schema.go", 567, 65),
		censusInterfaceCallSites("reflect", "Type", "Kind", "schema.go", 558, 38),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "rollbackV4"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "ReadFile", "migration_v4_apply.go", 212, 22),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "syncDirectory"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationDirectory", "Close", "migration.go", 514, 7, "migration.go", 517, 9),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationDirectory", "Sync", "migration.go", 513, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "OpenDirectory", "migration.go", 509, 17),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "validateRootKinds"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "IsDir", "loader.go", 518, 7),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "validateTerminal"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "BackendSettingsValidator", "ValidateBackendSettings", "validation.go", 849, 13),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeAll"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io", "Writer", "Write", "migration.go", 496, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempFile"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFile", "Chmod", "migration.go", 478, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFile", "Close", "migration.go", 474, 7, "migration.go", 487, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFile", "Name", "migration.go", 472, 9),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFile", "Sync", "migration.go", 484, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "CreateTemp", "migration.go", 468, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Remove", "migration.go", 475, 7, "migration.go", 488, 7),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempReplace"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Remove", "migration_v4_helpers.go", 138, 7),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/config", "migrationFileSystem", "Rename", "migration_v4_helpers.go", 137, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "AuthRequest", name: "Format"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("fmt", "State", "Write", "authorize.go", 35, 9),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "CredentialEntry", name: "Format"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("fmt", "State", "Write", "trust.go", 67, 9),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "EnrollmentMaterial", name: "Format"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("fmt", "State", "Write", "export.go", 27, 9),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Snapshot", name: "Format"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("fmt", "State", "Write", "authorize.go", 42, 9),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "TrustStore", name: "Format"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("fmt", "State", "Write", "trust.go", 62, 63),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "Initialize"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "transact.go", 168, 15),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "JointCommit"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "joint.go", 254, 15),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "checkP256Point"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("crypto/elliptic", "Curve", "IsOnCurve", "profile_checks.go", 45, 6),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "cleanupCredentialDirectory"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove", "enroll.go", 206, 6, "enroll.go", 207, 6, "enroll.go", 208, 6, "enroll.go", 209, 6),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "cleanupStagedFile"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove", "transact.go", 313, 7),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "commitConfigBinding"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Chmod", "binding.go", 336, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateTemp", "binding.go", 319, 17),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Rename", "binding.go", 342, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Close", "binding.go", 330, 7, "binding.go", 333, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Name", "binding.go", 323, 10),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Sync", "binding.go", 329, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "commitDocument"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Chmod", "transact.go", 293, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateTemp", "transact.go", 276, 17),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Rename", "transact.go", 301, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Close", "transact.go", 287, 7, "transact.go", 290, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Name", "transact.go", 280, 10),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Sync", "transact.go", 286, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "compensateLocked"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "joint.go", 356, 24, "joint.go", 365, 20),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "ensureOwnerDir"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Chmod", "custody.go", 283, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "custody.go", 269, 15, "custody.go", 292, 14),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "MkdirAll", "custody.go", 278, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "loadBindingAt"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "binding.go", 146, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "binding.go", 156, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "loadCommitted"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "transact.go", 246, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "transact.go", 256, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "markerPresent"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "joint.go", 532, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "openLock"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Chmod", "lock_unix.go", 62, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateExclusive", "lock_unix.go", 51, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "lock_unix.go", 41, 12, "lock_unix.go", 68, 15),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "platformModeOK"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "Mode", "custody_unix.go", 15, 9),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readCommitted"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "authorize.go", 89, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "authorize.go", 96, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readCompensationMarker"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "joint.go", 401, 19),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "joint.go", 411, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readOwnerFile"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "enroll.go", 272, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "enroll.go", 279, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "readSnapshotLocked"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "transact.go", 133, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "transact.go", 143, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "recoverLocked"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "joint.go", 467, 15, "joint.go", 473, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "joint.go", 480, 19, "joint.go", 493, 24),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "removeMarkerLocked"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove", "joint.go", 580, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "resolveFailedReplace"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "ReadFile", "joint.go", 301, 24),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "syncDir"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "OpenDirectory", "transact.go", 332, 17),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "SyncedDirectory", "Close", "transact.go", 337, 7, "transact.go", 340, 9),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "SyncedDirectory", "Sync", "transact.go", 336, 12),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyLockFile"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "custody.go", 334, 16),
		censusInterfaceCallSites("io/fs", "FileInfo", "Mode", "custody.go", 338, 7, "custody.go", 338, 34),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyOwner"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "Sys", "custody_unix.go", 26, 14),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyOwnerDir"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "IsDir", "custody.go", 300, 6),
		censusInterfaceCallSites("io/fs", "FileInfo", "Mode", "custody.go", 303, 5),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "verifyOwnerFile"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("io/fs", "FileInfo", "Mode", "custody.go", 351, 6, "custody.go", 354, 5),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "writeAll"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "interface{Write([]byte) (int, error)}", "Write", "transact.go", 319, 19),
	),
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "writeCustodyFile"}: censusInterfaceCallSiteSet(
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "CreateTemp", "enroll.go", 162, 17),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Lstat", "enroll.go", 191, 15),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Remove", "enroll.go", 171, 7),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "FileSystem", "Rename", "enroll.go", 188, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Close", "enroll.go", 170, 7, "enroll.go", 177, 7, "enroll.go", 181, 7, "enroll.go", 184, 12),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Name", "enroll.go", 166, 13),
		censusInterfaceCallSites("github.com/relux-works/agent-session-manager/internal/hosttrust", "StagedFile", "Sync", "enroll.go", 180, 12),
	),
}

// These definitions perform or directly authorize durable writes. Their
// indirect interface edges must be lexically dominated by a top-level
// fail-fast hold verification in the same function. The checker intentionally
// accepts only that conservative guard shape; a branch-local or later hold
// check is not evidence for a preceding call.
var censusInterfaceCallHoldRequirements = map[censusFunction]bool{
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "replaceDurably"}:                            true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", name: "writeTempReplace"}:                          true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", name: "commitDocument"}:                         true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "commitConfigBinding"}: true,
}

// These package variables are test seams or immutable error constructors,
// not filesystem writers. Their declaration objects are admitted explicitly;
// arbitrary function-valued variables remain rejected at call sites.
var censusAllowedFunctionValues = map[string]map[string]bool{
	"github.com/relux-works/agent-session-manager/internal/config": {
		"tomlDecode": true, "migrationError": true, "configError": true, "loaderError": true,
		"terminalCapabilitySet": true,
	},
	"github.com/relux-works/agent-session-manager/internal/hosttrust": {
		"trustError": true,
	},
}

// These standard-library function variables are immutable constructors or
// curve selectors, not mutation entry points. They are admitted by the exact
// declaration object resolved from the loaded package, never by a signature.
var censusAllowedExternalFunctionValues = map[string]map[string]bool{
	"crypto/elliptic": {"P256": true},
}

// The terminal capability set is an immutable package initializer whose
// expression is an immediately-invoked function literal. Keep that one
// initializer explicit; arbitrary immediate function calls remain rejected.
var censusAllowedFunctionInitializers = map[string]map[string]bool{
	"github.com/relux-works/agent-session-manager/internal/config": {"var terminalCapabilitySet": true},
}

type censusFunctionField struct {
	pkgPath  string
	typeName string
	name     string
}

// sshValueRule.permits is a pure validator callback. The scanner records its
// resolved field object below; an unrelated function-valued field remains
// rejected even if it has the same shape elsewhere.
var censusAllowedFunctionFields = map[censusFunctionField]bool{
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", typeName: "sshOptionRule", name: "permits"}: true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", typeName: "Inputs", name: "LookupEnv"}:      true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", typeName: "Inputs", name: "Stat"}:           true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/config", typeName: "Inputs", name: "ReadFile"}:       true,
}

func newCensusTypeScanner(unit censusTypeUnit, gated map[*types.Func]string) censusTypeScanner {
	scanner := censusTypeScanner{
		unit: unit, gated: gated,
		allowedIndirectObjects:  map[*types.Func]map[types.Object]bool{},
		allowedInterfaceMethods: map[*types.Func]map[types.Object]bool{},
		allowedInterfaceSites:   map[*types.Func]map[censusInterfaceCallSite]bool{},
		allowedFunctionValues:   map[types.Object]bool{},
		allowedInitializers:     map[string]bool{},
		functionDecls:           map[*types.Func]*ast.FuncDecl{},
	}
	for name := range censusAllowedFunctionInitializers[unit.pkg.Path()] {
		scanner.allowedInitializers[name] = true
	}
	if unit.pkg != nil && unit.pkg.Scope() != nil {
		for name := range censusAllowedFunctionValues[unit.pkg.Path()] {
			if object := unit.pkg.Scope().Lookup(name); object != nil {
				scanner.allowedFunctionValues[object] = true
			}
		}
	}
	for _, object := range unit.info.Uses {
		variable, ok := object.(*types.Var)
		if !ok || variable.Pkg() == nil {
			continue
		}
		if censusAllowedExternalFunctionValues[variable.Pkg().Path()][variable.Name()] {
			scanner.allowedFunctionValues[variable] = true
		}
	}
	for _, file := range unit.files {
		for _, declaration := range file.Decls {
			if general, ok := declaration.(*ast.GenDecl); ok && general.Tok == token.TYPE {
				for _, specification := range general.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range structure.Fields.List {
						for _, identifier := range field.Names {
							object := unit.info.Defs[identifier]
							if censusAllowedFunctionFields[censusFunctionField{pkgPath: unit.pkg.Path(), typeName: typeSpec.Name.Name, name: identifier.Name}] {
								scanner.allowedFunctionValues[object] = true
							}
						}
					}
				}
			}
			functionDecl, ok := declaration.(*ast.FuncDecl)
			if !ok || functionDecl.Name == nil {
				continue
			}
			function, ok := unit.info.Defs[functionDecl.Name].(*types.Func)
			if !ok {
				continue
			}
			scanner.functionDecls[function] = functionDecl
			policy, hasPolicy := censusIndirectPolicies[censusFunctionOf(function)]
			methods := censusRequiredInterfaceMethods[censusFunctionOf(function)]
			if !hasPolicy && len(methods) == 0 {
				continue
			}
			if len(methods) != 0 {
				allowedMethods := map[types.Object]bool{}
				for _, selection := range unit.info.Selections {
					if censusAllowedInterfaceMethod(selection.Obj(), methods) {
						allowedMethods[selection.Obj()] = true
					}
				}
				allowedSites := map[censusInterfaceCallSite]bool{}
				for _, site := range censusRequiredInterfaceCallSites[censusFunctionOf(function)] {
					allowedSites[site] = true
				}
				scanner.allowedInterfaceMethods[function] = allowedMethods
				scanner.allowedInterfaceSites[function] = allowedSites
			}
			signature, ok := function.Type().(*types.Signature)
			if !ok {
				continue
			}
			allowed := map[types.Object]bool{}
			for index := 0; index < signature.Params().Len(); index++ {
				parameter := signature.Params().At(index)
				if policy.parameters[parameter.Name()] {
					allowed[parameter] = true
				}
			}
			if len(policy.locals) != 0 && functionDecl.Body != nil {
				ast.Inspect(functionDecl.Body, func(node ast.Node) bool {
					identifier, ok := node.(*ast.Ident)
					if !ok || !policy.locals[identifier.Name] {
						return true
					}
					if variable, ok := unit.info.Uses[identifier].(*types.Var); ok {
						allowed[variable] = true
					}
					return true
				})
			}
			if len(allowed) != 0 {
				scanner.allowedIndirectObjects[function] = allowed
			}
		}
	}
	return scanner
}

func (scanner *censusTypeScanner) scan() []string {
	for _, file := range scanner.unit.files {
		for _, declaration := range file.Decls {
			switch value := declaration.(type) {
			case *ast.FuncDecl:
				if value.Body == nil || scanner.isBackendMethod(value) {
					continue
				}
				function := censusObject(scanner.unit.info.Defs[value.Name])
				scanner.walk(value.Body, value.Name.Name, function)
			case *ast.GenDecl:
				for _, specification := range value.Specs {
					variables, ok := specification.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, initializer := range variables.Values {
						name := "package initializer"
						if len(variables.Names) == len(variables.Values) {
							for index, variable := range variables.Values {
								if variable == initializer && index < len(variables.Names) {
									name = "var " + variables.Names[index].Name
								}
							}
						}
						scanner.walk(initializer, name, nil)
					}
				}
			}
		}
	}
	return scanner.result
}

func (scanner *censusTypeScanner) isBackendMethod(function *ast.FuncDecl) bool {
	object, ok := scanner.unit.info.Defs[function.Name].(*types.Func)
	if !ok {
		return false
	}
	return censusBackendDefinition(object)
}

func censusObject(object types.Object) *types.Func {
	function, _ := object.(*types.Func)
	return function
}

func censusFunctionOf(object *types.Func) censusFunction {
	if object == nil {
		return censusFunction{}
	}
	function := censusFunction{name: object.Name()}
	if object.Pkg() != nil {
		function.pkgPath = object.Pkg().Path()
	}
	if signature, ok := object.Type().(*types.Signature); ok && signature.Recv() != nil {
		function.receiver = censusNamedType(signature.Recv().Type())
	}
	return function
}

// censusAllowedDefinition matches the allowlist by the resolved definition
// identity, not by a bare spelling. The package path and receiver are part of
// the key; a same-named helper in another package or a newly added receiver
// method is scanned as a new write path. Synthetic plants use the empty
// censuscontrol package as a shape-only package, so they retain the original
// receiver-independent control semantics.
func censusAllowedDefinition(helper, current *types.Func, allowed map[string]bool) bool {
	if helper == nil || current == nil || helper.Pkg() == nil || current.Pkg() == nil {
		return false
	}
	if helper.Pkg().Path() != current.Pkg().Path() || !allowed[current.Name()] {
		return false
	}
	if current.Pkg().Path() == "censuscontrol" {
		return true
	}
	expected, hasExpected := censusExpectedReceivers[current.Name()]
	if !hasExpected {
		return current.Type().(*types.Signature).Recv() == nil
	}
	return censusNamedType(current.Type().(*types.Signature).Recv().Type()) == expected
}

var censusExpectedReceivers = map[string]string{
	"Initialize":                "Store",
	"JointCommit":               "Store",
	"Issue":                     "Store",
	"Rotate":                    "Store",
	"MarkRetiring":              "Store",
	"Revoke":                    "Store",
	"ensureConfigBindingLocked": "Store",
	"commitConfigBinding":       "Store",
}

func censusNamedType(typ types.Type) string {
	switch value := typ.(type) {
	case *types.Pointer:
		return censusNamedType(value.Elem())
	case *types.Named:
		return value.Obj().Name()
	default:
		return ""
	}
}

func censusBackendDefinition(object *types.Func) bool {
	function := censusFunctionOf(object)
	if !censusBackendPackages[function.pkgPath] {
		return false
	}
	methods, ok := censusBackendMethods[function.receiver]
	return ok && methods[function.name]
}

// censusIsCensusedDefinition identifies the narrow set of production and
// synthetic definitions whose bodies are already covered by a stronger
var censusBackendPackages = map[string]bool{
	"github.com/relux-works/agent-session-manager/internal/config":    true,
	"github.com/relux-works/agent-session-manager/internal/hosttrust": true,
	"censuscontrol": true,
}

func (scanner *censusTypeScanner) walk(root ast.Node, functionName string, current *types.Func) {
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			if len(stack) != 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		stack = append(stack, node)
		scanner.visit(node, stack, functionName, current)
		return true
	})
}

func (scanner *censusTypeScanner) visit(node ast.Node, stack []ast.Node, functionName string, current *types.Func) {
	switch value := node.(type) {
	case *ast.CallExpr:
		scanner.visitCall(value, stack, functionName, current)
	case *ast.Ident:
		scanner.visitIdent(value, stack, functionName, current)
	case *ast.SelectorExpr:
		if !scanner.isCallCallee(value, stack) {
			scanner.visitSelectorValue(value, stack, functionName, current)
		}
	}
}

func (scanner *censusTypeScanner) visitCall(call *ast.CallExpr, stack []ast.Node, functionName string, current *types.Func) {
	if object, isInterface := scanner.interfaceMethodObject(call.Fun); isInterface {
		if !scanner.allowedInterfaceCall(current, call, object) {
			scanner.add(call, "interface or type-parameter call target/site is not an allowlisted edge in "+functionName)
		}
		return
	}
	object, resolved := scanner.callObject(call)
	if !resolved {
		// A conversion has no callable object and is harmless by definition;
		// every other unresolved callee, including an immediate function
		// literal, is an unknown indirect edge and fails closed.
		if typeAndValue, ok := scanner.unit.info.Types[call.Fun]; ok && typeAndValue.IsType() {
			return
		}
		if _, immediate := call.Fun.(*ast.FuncLit); immediate && scanner.allowedInitializers[functionName] {
			return
		}
		scanner.add(call, "unresolved call callee in "+functionName)
		return
	}
	function, ok := object.(*types.Func)
	if !ok {
		switch object.(type) {
		case *types.Builtin, *types.TypeName:
			return
		}
		if scanner.allowedFunctionValues[object] {
			return
		}
		if !scanner.allowedIndirectObjects[current][object] {
			scanner.add(call, "indirect call callee is not an allowlisted declaration in "+functionName)
		}
		return
	}
	if helper, isGated := scanner.gated[function]; isGated {
		scanner.checkGatedCall(call, stack, functionName, helper, function, current)
		return
	}
	if censusIsOSMutation(function) {
		scanner.add(call, "direct os mutation outside the exact backend definition in "+functionName)
		return
	}
	if censusIsRawMutation(function) {
		allowed := censusRawMutations[function.Name()]
		if !censusAllowedDefinition(function, current, allowed) {
			scanner.add(call, "filesystem mutation method outside the censused writer in "+functionName)
		}
	}
}

func (scanner *censusTypeScanner) checkGatedCall(call *ast.CallExpr, stack []ast.Node, functionName, helper string, helperFunction, current *types.Func) {
	position := scanner.unit.fset.Position(call.Pos())
	if censusPairHelpers[helper] {
		if scanner.insideHoldClosure(stack) {
			return
		}
		scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: %s call is outside a resource-bound hold closure in %s", position.Filename, position.Line, helper, functionName))
		return
	}
	if allowed, ok := censusTrustHelpers[helper]; ok {
		if censusAllowedDefinition(helperFunction, current, allowed) {
			return
		}
	}
	if allowed, ok := censusCustodyHelpers[helper]; ok {
		if censusAllowedDefinition(helperFunction, current, allowed) {
			return
		}
	}
	scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: %s call is outside the censused writer in %s", position.Filename, position.Line, helper, functionName))
}

func (scanner *censusTypeScanner) visitIdent(identifier *ast.Ident, stack []ast.Node, functionName string, current *types.Func) {
	object := scanner.unit.info.Uses[identifier]
	if variable, ok := object.(*types.Var); ok {
		if scanner.isCompositeLiteralKey(identifier, stack) {
			return
		}
		if _, functionValue := variable.Type().(*types.Signature); functionValue && !scanner.isCallCallee(identifier, stack) && !scanner.isSelectorName(identifier, stack) && !scanner.allowedFunctionValues[variable] && !scanner.allowedIndirectObjects[current][variable] {
			scanner.add(identifier, "unallowlisted function-valued identifier escapes in "+functionName)
		}
		return
	}
	function, ok := object.(*types.Func)
	if !ok {
		return
	}
	if scanner.isCallCallee(identifier, stack) || scanner.isSelectorName(identifier, stack) {
		return
	}
	helper, gated := scanner.gated[function]
	if !gated && !censusIsOSMutation(function) && !censusIsRawMutation(function) {
		return
	}
	position := scanner.unit.fset.Position(identifier.Pos())
	if gated {
		scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: gated helper %s escapes as a value in %s", position.Filename, position.Line, helper, functionName))
		return
	}
	scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: filesystem mutation function %s escapes as a value in %s", position.Filename, position.Line, function.Name(), functionName))
}

func (scanner *censusTypeScanner) isCompositeLiteralKey(identifier *ast.Ident, stack []ast.Node) bool {
	for index := len(stack) - 1; index >= 0; index-- {
		keyValue, ok := stack[index].(*ast.KeyValueExpr)
		if ok && keyValue.Key == identifier {
			return true
		}
	}
	return false
}

func (scanner *censusTypeScanner) visitSelectorValue(selector *ast.SelectorExpr, stack []ast.Node, functionName string, current *types.Func) {
	object := scanner.selectorObject(selector)
	if scanner.isInterfaceSelector(selector) {
		if !scanner.allowedInterfaceCall(current, selector, object) {
			position := scanner.unit.fset.Position(selector.Pos())
			scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: interface or type-parameter method value/site escapes in %s", position.Filename, position.Line, functionName))
		}
		return
	}
	function, ok := object.(*types.Func)
	if !ok {
		if variable, ok := object.(*types.Var); ok {
			if _, functionValue := variable.Type().(*types.Signature); functionValue && !scanner.allowedFunctionValues[variable] && !scanner.allowedIndirectObjects[current][variable] {
				position := scanner.unit.fset.Position(selector.Pos())
				scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: unallowlisted function-valued selector escapes in %s", position.Filename, position.Line, functionName))
			}
		}
		return
	}
	if helper, gated := scanner.gated[function]; gated {
		position := scanner.unit.fset.Position(selector.Pos())
		scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: gated helper %s escapes as a value in %s", position.Filename, position.Line, helper, functionName))
		return
	}
	if censusIsOSMutation(function) || censusIsRawMutation(function) {
		position := scanner.unit.fset.Position(selector.Pos())
		scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: filesystem mutation function %s escapes as a value in %s", position.Filename, position.Line, function.Name(), functionName))
	}
}

func (scanner *censusTypeScanner) allowedInterfaceCall(current *types.Func, node ast.Node, object types.Object) bool {
	if !scanner.allowedInterfaceMethods[current][object] {
		return false
	}
	position := scanner.unit.fset.Position(node.Pos())
	for site := range scanner.allowedInterfaceSites[current] {
		if filepath.Base(position.Filename) != site.file || position.Line != site.line || position.Column != site.column {
			continue
		}
		if !censusAllowedInterfaceMethod(object, []censusInterfaceMethod{site.method}) {
			continue
		}
		if censusInterfaceCallHoldRequirements[censusFunctionOf(current)] && !scanner.interfaceCallDominatedByHold(current, node) {
			return false
		}
		return true
	}
	return false
}

func (scanner *censusTypeScanner) interfaceCallDominatedByHold(current *types.Func, node ast.Node) bool {
	functionDecl := scanner.functionDecls[current]
	if functionDecl == nil || functionDecl.Body == nil {
		return false
	}
	targetIndex := -1
	for index, statement := range functionDecl.Body.List {
		if containsNode(statement, node) {
			targetIndex = index
			break
		}
	}
	if targetIndex < 0 {
		return false
	}
	for _, statement := range functionDecl.Body.List[:targetIndex] {
		if censusIsFailFastHoldGuard(statement, scanner) {
			return true
		}
	}
	return false
}

func censusIsFailFastHoldGuard(statement ast.Stmt, scanner *censusTypeScanner) bool {
	guard, ok := statement.(*ast.IfStmt)
	if !ok || guard.Else != nil {
		return false
	}
	holdCheck := false
	for _, expression := range []ast.Node{guard.Init, guard.Cond} {
		if expression == nil {
			continue
		}
		ast.Inspect(expression, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && scanner.isHoldVerificationCall(call) {
				holdCheck = true
				return false
			}
			return true
		})
	}
	if !holdCheck {
		return false
	}
	for _, nested := range guard.Body.List {
		if _, ok := nested.(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

func (scanner *censusTypeScanner) isHoldVerificationCall(call *ast.CallExpr) bool {
	object, resolved := scanner.callObject(call)
	if !resolved {
		return false
	}
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return false
	}
	switch function.Pkg().Path() {
	case "github.com/relux-works/agent-session-manager/internal/config":
		return function.Name() == "requireHold" || function.Name() == "requireHoldForConfig"
	case "github.com/relux-works/agent-session-manager/internal/hosttrust":
		return function.Name() == "requireHold" || function.Name() == "requireHoldForRoot"
	default:
		return false
	}
}

func (scanner *censusTypeScanner) callObject(call *ast.CallExpr) (types.Object, bool) {
	object := scanner.expressionObject(call.Fun)
	if object != nil {
		return object, true
	}
	return nil, false
}

func (scanner *censusTypeScanner) expressionObject(expression ast.Expr) types.Object {
	switch value := expression.(type) {
	case *ast.Ident:
		return scanner.unit.info.Uses[value]
	case *ast.SelectorExpr:
		return scanner.selectorObject(value)
	case *ast.ParenExpr:
		return scanner.expressionObject(value.X)
	case *ast.IndexExpr:
		return scanner.expressionObject(value.X)
	case *ast.IndexListExpr:
		return scanner.expressionObject(value.X)
	}
	return nil
}

func (scanner *censusTypeScanner) selectorObject(selector *ast.SelectorExpr) types.Object {
	if selection := scanner.unit.info.Selections[selector]; selection != nil {
		return selection.Obj()
	}
	return scanner.unit.info.Uses[selector.Sel]
}

func (scanner *censusTypeScanner) interfaceMethodObject(expression ast.Expr) (types.Object, bool) {
	selector, ok := interfaceSelector(expression)
	if !ok {
		return nil, false
	}
	selection := scanner.unit.info.Selections[selector]
	if selection == nil || !censusIsInterfaceType(selection.Recv()) {
		return nil, false
	}
	return selection.Obj(), true
}

func (scanner *censusTypeScanner) isInterfaceSelector(selector *ast.SelectorExpr) bool {
	_, ok := scanner.interfaceMethodObject(selector)
	return ok
}

func interfaceSelector(expression ast.Expr) (*ast.SelectorExpr, bool) {
	for {
		switch value := expression.(type) {
		case *ast.SelectorExpr:
			return value, true
		case *ast.ParenExpr:
			expression = value.X
		default:
			return nil, false
		}
	}
}

func censusAllowedInterfaceMethod(object types.Object, methods []censusInterfaceMethod) bool {
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return false
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiver := censusNamedType(signature.Recv().Type())
	if receiver == "" {
		receiver = types.TypeString(signature.Recv().Type(), nil)
	}
	for _, method := range methods {
		if method.pkgPath == function.Pkg().Path() && method.receiver == receiver && method.name == function.Name() {
			return true
		}
	}
	return false
}

func censusIsInterfaceType(typ types.Type) bool {
	switch value := typ.(type) {
	case *types.Pointer:
		return censusIsInterfaceType(value.Elem())
	case *types.Named:
		return censusIsInterfaceType(value.Underlying())
	case *types.Interface:
		return true
	case *types.TypeParam:
		return true
	default:
		return false
	}
}

func (scanner *censusTypeScanner) isCallCallee(node ast.Node, stack []ast.Node) bool {
	for index := len(stack) - 1; index > 0; index-- {
		call, ok := stack[index-1].(*ast.CallExpr)
		if ok && call.Fun == node {
			return true
		}
		if call, ok := stack[index-1].(*ast.CallExpr); ok {
			if containsNode(call.Fun, node) {
				return true
			}
		}
		if _, ok := stack[index-1].(*ast.FuncLit); ok {
			break
		}
	}
	return false
}

func (scanner *censusTypeScanner) isSelectorName(identifier *ast.Ident, stack []ast.Node) bool {
	for index := len(stack) - 1; index >= 0; index-- {
		selector, ok := stack[index].(*ast.SelectorExpr)
		if ok && selector.Sel == identifier {
			return true
		}
	}
	return false
}

func containsNode(root ast.Node, needle ast.Node) bool {
	found := false
	ast.Inspect(root, func(node ast.Node) bool {
		if node == needle {
			found = true
			return false
		}
		return !found
	})
	return found
}

func (scanner *censusTypeScanner) insideHoldClosure(stack []ast.Node) bool {
	for index := len(stack) - 1; index >= 0; index-- {
		literal, ok := stack[index].(*ast.FuncLit)
		if !ok {
			continue
		}
		if index == 0 {
			return false
		}
		call, ok := stack[index-1].(*ast.CallExpr)
		if !ok || !scanner.isHoldEntryCall(call) {
			return false
		}
		for _, argument := range call.Args {
			if argument == literal {
				return true
			}
		}
		return false
	}
	return false
}

func (scanner *censusTypeScanner) isHoldEntryCall(call *ast.CallExpr) bool {
	object, resolved := scanner.callObject(call)
	if !resolved {
		return false
	}
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	return censusHoldEntries[censusFunctionOf(function)]
}

func (scanner *censusTypeScanner) add(node ast.Node, message string) {
	position := scanner.unit.fset.Position(node.Pos())
	scanner.result = append(scanner.result, fmt.Sprintf("%s:%d: %s", position.Filename, position.Line, message))
}

func censusIsOSMutation(function *types.Func) bool {
	return function != nil && function.Pkg() != nil && function.Pkg().Path() == "os" && censusOSMutations[function.Name()]
}

func censusIsRawMutation(function *types.Func) bool {
	if function == nil || censusIsOSMutation(function) {
		return false
	}
	if _, ok := censusRawMutations[function.Name()]; ok {
		return function.Type() != nil && function.Type().(*types.Signature).Recv() != nil
	}
	return function.Name() == "CreateExclusive"
}

var censusHoldEntries = map[censusFunction]bool{
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "JointCommit"}:                true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithExclusiveHold"}:          true,
	{pkgPath: "github.com/relux-works/agent-session-manager/internal/hosttrust", receiver: "Store", name: "WithExclusiveHoldForConfig"}: true,
	{pkgPath: "censuscontrol", receiver: "store", name: "JointCommit"}:                                                                  true,
	{pkgPath: "censuscontrol", receiver: "store", name: "WithExclusiveHold"}:                                                            true,
}
