package config

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
)

// This file enforces the rev9 write-path census statically: every
// filesystem mutation of the config-plus-trust pair must happen through
// an allowlisted helper from a lock-held context, and every pair writer
// additionally requires a live exclusive-hold capability at runtime (see
// hosttrust.HeldExclusive). The census itself is the board outcome
// TASK-260909-2ez769_write-path-census-rev8.md; this gate fails when a
// pair mutation appears outside the censused rows.
//
// Rules (see the census for the per-row rationale):
//   A. config pair writes (replaceDurably, writeTempReplace) run only
//      inside a closure passed directly to a hold entry (hosttrust
//      JointCommit or WithExclusiveHold), which holds the exclusive
//      authorization lock across revalidation, replacement and
//      compensation. Legacy migrate coordinates through
//      WithExclusiveHold like every other writer; there is no unlocked
//      exemption.
//   B. trust/marker writes (commitDocument, removeMarkerLocked) run only
//      in JointCommit, transactLocked, Initialize, recoverLocked,
//      resolveFailedReplace and compensateLocked, all under the
//      exclusive hold.
//   C. credential-file writes (writeCustodyFile) run only in Issue and
//      Rotate, which stage inert files before the admitting trust commit.
//   D. raw filesystem entry mutations (Rename, Remove, CreateTemp,
//      CreateExclusive, Link) run only inside the helpers above, their
//      staging/cleanup sites, and the atomic lock publication. Mode and
//      directory setup (Chmod, MkdirAll) is custody, covered by the
//      custody tests, not by this gate.
//   E. custody constructors (ensureOwnerDir, secureStaged) run only at
//      the censused setup/install sites.
//   F. config staging (writeTempFile) runs only inside the pair writers.
//   G. direct standard-library mutations (os.Rename, os.Remove,
//      os.RemoveAll, os.Create, os.CreateTemp, os.OpenFile, os.WriteFile,
//      os.MkdirTemp, os.Link, os.Symlink, os.Truncate) run only inside
//      the filesystem backend definitions (the osFileSystem and
//      osMigrationFileSystem method sets); every other occurrence,
//      including method receivers and wrapper indirection, is a
//      violation.
//   H. every resolved *types.Func that is a gated helper or filesystem
//      mutation is a writer in every value position: binding one to a
//      variable, field or parameter, passing one as an argument, returning
//      one, storing it in a composite literal or importing it with dot syntax
//      is a violation unless the binding sits in a censused definition.
//      Writer-shaped calls through a *types.Var, parameter, field or
//      interface method are rejected unless the call is inside a censused
//      definition; the gate does not do dataflow. The gate resolves local
//      aliases transitively and scans package variable initializers as well
//      as function bodies.
//
// The control TestWritePathGateFlagsUnlockedCaller proves the gate sees
// new unlocked callers in the three original shapes; the executable
// controls TestWritePathGateFlagsExecutableRogues prove it sees the earlier
// reviewer evasion shapes plus promoted methods, package-level writer slices,
// function-typed fields and parameters, and dot-imported composite values,
// each paired with proof that the flagged construct really mutates. The
// hold-require-skip and replace-require-skip mutation plants prove the runtime
// capability behaviorally.

var censusPairHelpers = map[string]bool{
	"replaceDurably":   true,
	"writeTempReplace": true,
}

var censusTrustHelpers = map[string]map[string]bool{
	"commitDocument": {
		"transactLocked": true, "Initialize": true, "JointCommit": true, "compensateLocked": true, "ensureConfigBindingLocked": true,
	},
	"commitConfigBinding": {"ensureConfigBindingLocked": true},
	"removeMarkerLocked": {
		"JointCommit": true, "recoverLocked": true, "compensateLocked": true, "resolveFailedReplace": true,
	},
}

var censusCustodyHelpers = map[string]map[string]bool{
	"writeCustodyFile": {"Issue": true, "Rotate": true},
	"writeTempFile":    {"replaceDurably": true, "writeTempReplace": true},
	"ensureOwnerDir":   {"openStore": true, "Issue": true, "Rotate": true},
	"secureStaged":     {"commitDocument": true, "commitConfigBinding": true, "writeCustodyFile": true, "ensureOwnerDir": true, "openLock": true},
}

var censusRawMutations = map[string]map[string]bool{
	"Rename": {
		"commitDocument": true, "commitConfigBinding": true, "writeCustodyFile": true, "replaceDurably": true, "writeTempReplace": true,
	},
	"Remove": {
		"commitDocument": true, "writeCustodyFile": true, "Issue": true, "Rotate": true,
		"removeMarkerLocked": true, "writeTempFile": true, "writeTempReplace": true, "replaceDurably": true,
		"cleanupStagedFile": true, "cleanupCredentialDirectory": true,
	},
	"CreateTemp": {
		"commitDocument": true, "commitConfigBinding": true, "writeCustodyFile": true, "writeTempFile": true,
	},
	"CreateExclusive": {"openLock": true},
	"Link":            {"replaceDurably": true},
}

// censusGatedIdents are the helper names that must never escape as
// values (rule H) and whose local aliases resolve to the helper itself.
var censusGatedIdents = map[string]bool{
	"replaceDurably": true, "writeTempReplace": true, "writeTempFile": true,
	"commitDocument": true, "commitConfigBinding": true, "removeMarkerLocked": true,
	"writeCustodyFile": true, "ensureOwnerDir": true, "secureStaged": true,
}

// censusOSMutations are the standard-library entry points that create,
// replace, remove or rename files. They run only inside the backend
// definitions (rule G). Reads (ReadFile, Stat, Lstat, ReadDir, Open),
// custody setup (Mkdir, MkdirAll, Chmod, Chtimes) and process control
// (Getenv, Exit, Args) are outside this rule.
var censusOSMutations = map[string]bool{
	"Rename": true, "Remove": true, "RemoveAll": true, "Create": true,
	"CreateTemp": true, "OpenFile": true, "WriteFile": true, "MkdirTemp": true,
	"Link": true, "Symlink": true, "Truncate": true,
}

// censusBackendMethods are the production filesystem backend
// definitions: their bodies implement the seams, so calls inside them
// are the definition of the interface, not new write paths. The
// exemption matches exact (receiver, method) pairs: a new method on a
// backend receiver is scanned like any other function, so a rogue
// cannot hide behind a backend receiver.
var censusBackendMethods = map[string]map[string]bool{
	"osFileSystem": {
		"CreateTemp": true, "CreateExclusive": true, "ReadFile": true,
		"Lstat": true, "Stat": true, "Remove": true, "Rename": true,
		"MkdirAll": true, "Chmod": true, "OpenDirectory": true, "ReadDir": true,
	},
	"osMigrationFileSystem": {
		"CreateTemp": true, "Link": true, "ReadFile": true, "Stat": true,
		"Remove": true, "Rename": true, "OpenDirectory": true,
	},
}

// censusPackageIdents are package qualifiers whose selectors are judged
// by the os-direct rule (for os) or ignored (pure helpers with no
// filesystem entry points).
var censusPackageIdents = map[string]bool{
	"os": true, "filepath": true, "unix": true, "windows": true,
	"strings": true, "bytes": true, "errors": true, "fmt": true,
}

func censusViolation(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// censusViolations scans the non-test Go sources under dirs and reports
// every pair mutation outside the censused rows.
func censusViolations(t *testing.T, dirs ...string) []string {
	return censusTypedViolations(t, dirs...)
}

func censusFileViolations(fileset *token.FileSet, file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if value.Body == nil {
				continue
			}
			if censusIsBackendMethod(value) {
				continue
			}
			violations = append(violations, censusBodyViolations(fileset, value.Body, value.Name.Name)...)
		case *ast.GenDecl:
			// Package variable initializers can hide executable writers
			// (a function literal calling a gated helper) or leak a
			// helper as a value; both are violations outside censused
			// definitions.
			for _, spec := range value.Specs {
				values, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				name := "package var"
				if len(values.Names) > 0 {
					name = "var " + values.Names[0].Name
				}
				for _, initializer := range values.Values {
					violations = append(violations, censusExprViolations(fileset, initializer, name, nil)...)
					if literal, ok := initializer.(*ast.FuncLit); ok {
						violations = append(violations, censusBodyViolations(fileset, literal.Body, name)...)
					}
				}
			}
		}
	}
	return violations
}

func censusIsBackendMethod(function *ast.FuncDecl) bool {
	if function.Recv == nil {
		return false
	}
	for _, field := range function.Recv.List {
		name := censusTypeName(field.Type)
		if methods, ok := censusBackendMethods[name]; ok && methods[function.Name.Name] {
			return true
		}
	}
	return false
}

func censusTypeName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return censusTypeName(value.X)
	default:
		return ""
	}
}

// censusBodyViolations attributes every gated call and helper escape in
// one function body to its enclosing declaration. Aliases bind
// body-wide (fail closed): any local name once bound to a gated helper
// resolves to that helper for the whole body.
func censusBodyViolations(fileset *token.FileSet, body *ast.BlockStmt, funcName string) []string {
	aliases := censusCollectAliases(body)
	var violations []string
	violations = append(violations, censusWalk(fileset, body, []ast.Node{body}, funcName, aliases)...)
	return violations
}

// censusCollectAliases binds local names to gated helpers through
// :=, var and plain assignments, transitively (an alias of an alias
// resolves to the helper). Only simple single-value bindings count;
// anything else is ignored, never assumed benign elsewhere.
func censusCollectAliases(body *ast.BlockStmt) map[string]string {
	bound := map[string]string{}
	resolve := func(expression ast.Expr) (string, bool) {
		ident, ok := expression.(*ast.Ident)
		if !ok {
			return "", false
		}
		if censusGatedIdents[ident.Name] {
			return ident.Name, true
		}
		target, ok := bound[ident.Name]
		return target, ok
	}
	bind := func(target ast.Expr, value ast.Expr) {
		ident, ok := target.(*ast.Ident)
		if !ok {
			return
		}
		if helper, ok := resolve(value); ok {
			bound[ident.Name] = helper
		}
	}
	for range 4 {
		changed := false
		ast.Inspect(body, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.AssignStmt:
				if len(value.Lhs) == 1 && len(value.Rhs) == 1 {
					before := len(bound)
					bind(value.Lhs[0], value.Rhs[0])
					changed = changed || len(bound) != before
				}
			case *ast.ValueSpec:
				if len(value.Names) == 1 && len(value.Values) == 1 {
					before := len(bound)
					bind(value.Names[0], value.Values[0])
					changed = changed || len(bound) != before
				}
			}
			return true
		})
		if !changed {
			break
		}
	}
	return bound
}

func censusWalk(fileset *token.FileSet, node ast.Node, ancestors []ast.Node, funcName string, aliases map[string]string) []string {
	var violations []string
	push := append(ancestors, node)
	switch value := node.(type) {
	case *ast.FuncLit:
		// Nested literals share the body's alias map (fail closed);
		// the enclosing declaration still attributes the violation.
	case *ast.CallExpr:
		if violation := censusCallViolation(fileset, value, push, funcName, aliases); violation != "" {
			violations = append(violations, violation)
		}
		// A gated helper passed as an argument escapes to whatever the
		// callee does with it; only hold-entry closures may receive
		// pair-writing behavior, and they take closures, never bare
		// helpers.
		for _, argument := range value.Args {
			if violation := censusEscapeViolation(fileset, argument, funcName, aliases, "argument"); violation != "" {
				violations = append(violations, violation)
			}
		}
	case *ast.AssignStmt:
		for _, right := range value.Rhs {
			if _, ok := right.(*ast.CallExpr); ok {
				continue
			}
			if violation := censusEscapeViolation(fileset, right, funcName, aliases, "assignment"); violation != "" {
				violations = append(violations, violation)
			}
		}
	case *ast.ValueSpec:
		for _, initializer := range value.Values {
			if _, ok := initializer.(*ast.CallExpr); ok {
				continue
			}
			if violation := censusEscapeViolation(fileset, initializer, funcName, aliases, "initializer"); violation != "" {
				violations = append(violations, violation)
			}
		}
	case *ast.ReturnStmt:
		for _, result := range value.Results {
			if _, ok := result.(*ast.CallExpr); ok {
				continue
			}
			if violation := censusEscapeViolation(fileset, result, funcName, aliases, "return"); violation != "" {
				violations = append(violations, violation)
			}
		}
	case *ast.KeyValueExpr:
		if _, ok := value.Value.(*ast.CallExpr); !ok {
			if violation := censusEscapeViolation(fileset, value.Value, funcName, aliases, "field"); violation != "" {
				violations = append(violations, violation)
			}
		}
	}
	for _, child := range censusChildren(node) {
		violations = append(violations, censusWalk(fileset, child, push, funcName, aliases)...)
	}
	return violations
}

// censusEscapeViolation flags a gated helper flowing as a value. Direct
// references and resolved aliases both count. Call targets are excluded:
// a call is judged by the call rule, so a censused call nested in the
// expression (for example inside a hold-entry closure argument) must not
// double-flag as an escape. Selector and field names are excluded as
// well: only value positions count.
func censusEscapeViolation(fileset *token.FileSet, expression ast.Expr, funcName string, aliases map[string]string, where string) string {
	excluded := map[*ast.Ident]bool{}
	ast.Inspect(expression, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.CallExpr:
			switch fun := value.Fun.(type) {
			case *ast.Ident:
				excluded[fun] = true
			case *ast.SelectorExpr:
				excluded[fun.Sel] = true
			}
		case *ast.SelectorExpr:
			excluded[value.Sel] = true
		case *ast.KeyValueExpr:
			if ident, ok := value.Key.(*ast.Ident); ok {
				excluded[ident] = true
			}
		}
		return true
	})
	found := ""
	ast.Inspect(expression, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok || excluded[ident] {
			return true
		}
		if ident.Name == "nil" {
			return true
		}
		if censusGatedIdents[ident.Name] {
			found = ident.Name
			return false
		}
		if helper, ok := aliases[ident.Name]; ok {
			found = helper + " (via alias " + ident.Name + ")"
			return false
		}
		return true
	})
	if found == "" {
		return ""
	}
	position := fileset.Position(expression.Pos())
	return censusViolation("%s:%d: gated helper %s escapes as %s in %s", position.Filename, position.Line, found, where, funcName)
}

func censusChildren(node ast.Node) []ast.Node {
	var children []ast.Node
	ast.Inspect(node, func(child ast.Node) bool {
		if child == nil || child == node {
			return true
		}
		children = append(children, child)
		return false
	})
	return children
}

// censusCallTarget resolves a call to its helper name and whether it is
// a direct standard-library call (os.X), a seam/ordinary selector call,
// or a plain identifier call (with alias resolution).
func censusCallTarget(call *ast.CallExpr, aliases map[string]string) (name string, osDirect bool, ok bool) {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		if helper, bound := aliases[function.Name]; bound {
			return helper, false, true
		}
		return function.Name, false, true
	case *ast.SelectorExpr:
		if ident, ok := function.X.(*ast.Ident); ok && ident.Name == "os" {
			return function.Sel.Name, true, true
		}
		if ident, ok := function.X.(*ast.Ident); ok && censusPackageIdents[ident.Name] {
			return "", false, false
		}
		return function.Sel.Name, false, true
	default:
		return "", false, false
	}
}

func censusCallViolation(fileset *token.FileSet, call *ast.CallExpr, ancestors []ast.Node, funcName string, aliases map[string]string) string {
	name, osDirect, ok := censusCallTarget(call, aliases)
	if !ok {
		return ""
	}
	position := fileset.Position(call.Pos())
	locate := func() string {
		return fmt.Sprintf("%s:%d: %s call in %s", position.Filename, position.Line, name, funcName)
	}
	if osDirect {
		// Backend definitions never reach this walk (their bodies are
		// skipped), so any direct os mutation here is a rogue writer.
		if censusOSMutations[name] {
			return censusViolation("%s is a direct os mutation outside the backend definitions", locate())
		}
		return ""
	}
	if censusPairHelpers[name] {
		if censusInsideHoldClosure(ancestors) {
			return ""
		}
		return censusViolation("%s is outside a JointCommit/WithExclusiveHold closure", locate())
	}
	if allowed, gated := censusTrustHelpers[name]; gated {
		if allowed[funcName] {
			return ""
		}
		return censusViolation("%s is outside the censused lock-held writers", locate())
	}
	if allowed, gated := censusCustodyHelpers[name]; gated {
		if allowed[funcName] {
			return ""
		}
		return censusViolation("%s is outside the censused custody sites", locate())
	}
	if allowed, gated := censusRawMutations[name]; gated {
		if allowed[funcName] {
			return ""
		}
		return censusViolation("%s is outside the censused mutation helpers", locate())
	}
	return ""
}

// censusInsideHoldClosure reports whether the call sits in a function
// literal passed directly as a hold-entry argument (JointCommit or
// WithExclusiveHold). Ancestors run outermost-first, so the literal's
// parent is the preceding entry; the nearest literal decides,
// conservatively.
func censusInsideHoldClosure(ancestors []ast.Node) bool {
	for index := len(ancestors) - 1; index >= 0; index-- {
		literal, ok := ancestors[index].(*ast.FuncLit)
		if !ok {
			continue
		}
		if index == 0 {
			return false
		}
		parent, ok := ancestors[index-1].(*ast.CallExpr)
		if !ok || !censusIsHoldEntryCall(parent) {
			return false
		}
		for _, argument := range parent.Args {
			if argument == ast.Node(literal) {
				return true
			}
		}
		return false
	}
	return false
}

func censusIsHoldEntryCall(call *ast.CallExpr) bool {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		return function.Name == "JointCommit" || function.Name == "WithExclusiveHold" || function.Name == "WithExclusiveHoldForConfig"
	case *ast.SelectorExpr:
		return function.Sel.Name == "JointCommit" || function.Sel.Name == "WithExclusiveHold" || function.Sel.Name == "WithExclusiveHoldForConfig"
	default:
		return false
	}
}

// censusExprViolations judges one package-level initializer expression:
// a bare gated helper there escapes, and a call there is judged by the
// call rule against the variable name.
func censusExprViolations(fileset *token.FileSet, expression ast.Expr, name string, aliases map[string]string) []string {
	var violations []string
	if call, ok := expression.(*ast.CallExpr); ok {
		if violation := censusCallViolation(fileset, call, []ast.Node{expression}, name, aliases); violation != "" {
			violations = append(violations, violation)
		}
		return violations
	}
	if _, ok := expression.(*ast.FuncLit); !ok {
		if violation := censusEscapeViolation(fileset, expression, name, aliases, "package initializer"); violation != "" {
			violations = append(violations, violation)
		}
	}
	return violations
}

func censusPackageDirs(t *testing.T) (configDir, trustDir string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	configDir = filepath.Dir(file)
	trustDir = filepath.Join(configDir, "..", "hosttrust")
	return configDir, trustDir
}

func TestWritePathCensusGate(t *testing.T) {
	configDir, trustDir := censusPackageDirs(t)
	violations := censusViolations(t, configDir, trustDir)
	for _, violation := range violations {
		t.Errorf("write-path census violation: %s", violation)
	}
}

// TestWritePathCensusRejectsUnlistedInterfaceCallSite copies the production
// source into an isolated module, plants two real filesystem renames in
// replaceDurably at new positions, and runs the production census plus a
// valid-hold witness. The pre-hold plant proves ordering; the post-hold plant
// proves that an unlisted site cannot inherit authorization from the same
// interface method. The test expects the child census to fail while the
// witness succeeds, so a site-admission mutant is observable behaviorally.
func TestWritePathCensusRejectsUnlistedInterfaceCallSite(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	copyCensusProject(t, censusModuleRoot(), project)
	applyCensusMutationOverlay(t, project)

	migrationPath := filepath.Join(project, "internal", "config", "migration.go")
	source, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	const holdGuard = "\tif err := requireHoldForConfig(hold, paths); err != nil {\n\t\treturn err\n\t}"
	const plantedGuard = "\tif backup == \"reviewer-pre-hold\" { return filesystem.Rename(string(original), string(replacement)) }; if err := requireHoldForConfig(hold, paths); err != nil {\n\t\treturn err\n\t}; if backup == \"reviewer-post-hold\" { return filesystem.Rename(string(original), string(replacement)) }"
	if !strings.Contains(string(source), holdGuard) {
		t.Fatal("replaceDurably hold guard was not found in the census plant source")
	}
	planted := strings.Replace(string(source), holdGuard, plantedGuard, 1)
	if err := os.WriteFile(migrationPath, []byte(planted), 0o600); err != nil {
		t.Fatal(err)
	}

	witness := `package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestReviewerUnlistedSiteExecutes(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	statePath := filepath.Join(directory, "state")
	if err := os.WriteFile(configPath, []byte("configuration"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: scalar.PlatformLinux,
		Flags: map[string]string{
			"--config": configPath, "--data-dir": filepath.Join(statePath, "data"),
			"--state-dir": statePath, "--cache-dir": filepath.Join(statePath, "cache"),
			"--runtime-dir": filepath.Join(statePath, "runtime"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := hosttrust.Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatal(err)
	}
	for _, branch := range []string{"reviewer-pre-hold", "reviewer-post-hold"} {
		original := filepath.Join(directory, branch+"-original")
		replacement := filepath.Join(directory, branch+"-replacement")
		if err := os.WriteFile(original, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := store.WithExclusiveHoldForConfig(paths, func(hold hosttrust.HeldExclusive) error {
			return replaceDurably(hold, osMigrationFileSystem{}, paths, branch, []byte(original), []byte(replacement))
		}); err != nil {
			t.Fatalf("replaceDurably(%s) = %v", branch, err)
		}
		contents, err := os.ReadFile(replacement)
		if err != nil {
			t.Fatal(err)
		}
		if string(contents) != "old" {
			t.Fatalf("%s witness contents = %q, want old", branch, contents)
		}
	}
}
`
	witnessPath := filepath.Join(project, "internal", "config", "reviewer_callsite_regression_test.go")
	if err := os.WriteFile(witnessPath, []byte(witness), 0o600); err != nil {
		t.Fatal(err)
	}
	child := exec.Command("go", "test", "./internal/config", "-run", "^(TestWritePathCensusGate|TestReviewerUnlistedSiteExecutes)$", "-count=1", "-v")
	child.Dir = project
	child.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	output, childErr := child.CombinedOutput()
	if childErr == nil {
		t.Fatalf("planted census child succeeded; gate admitted an unlisted site:\n%s", output)
	}
	if !strings.Contains(string(output), "--- PASS: TestReviewerUnlistedSiteExecutes") {
		t.Fatalf("planted child did not execute the real rename witness (err=%v):\n%s", childErr, output)
	}
	if !strings.Contains(string(output), "write-path census violation") {
		t.Fatalf("planted child failed without a census violation (err=%v):\n%s", childErr, output)
	}
}

// applyCensusMutationOverlay propagates the mutation harness's source overlay
// into the isolated child module. Without this handoff the child would test
// the on-disk baseline while the parent tests the mutant, making a behavioral
// mutant appear killed for the wrong reason. Non-census overlays are ignored.
func applyCensusMutationOverlay(t *testing.T, project string) {
	t.Helper()
	overlayPath := os.Getenv("CENSUS_MUTATION_OVERLAY")
	if overlayPath == "" {
		return
	}
	contents, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read census mutation overlay: %v", err)
	}
	var overlay struct {
		Replace map[string]string `json:"Replace"`
	}
	if err := json.Unmarshal(contents, &overlay); err != nil {
		t.Fatalf("decode census mutation overlay: %v", err)
	}
	for source, replacement := range overlay.Replace {
		if filepath.Base(source) != "writepath_census_types_test.go" {
			continue
		}
		mutated, err := os.ReadFile(replacement)
		if err != nil {
			t.Fatalf("read census mutation replacement: %v", err)
		}
		target := filepath.Join(project, "internal", "config", "writepath_census_types_test.go")
		if err := os.WriteFile(target, mutated, 0o600); err != nil {
			t.Fatalf("install census mutation replacement: %v", err)
		}
		return
	}
}

func copyCensusProject(t *testing.T, sourceRoot, destinationRoot string) {
	t.Helper()
	if err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destinationRoot, 0o700)
		}
		if relative == ".git" || relative == ".temp" || relative == ".task-board" {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if relative != "go.mod" && relative != "go.sum" && relative != "internal" && !strings.HasPrefix(relative, "internal"+string(filepath.Separator)) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		target := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, contents, 0o600)
	}); err != nil {
		t.Fatalf("copy census project: %v", err)
	}
}

// The type-aware gate fails closed when it cannot resolve a callee. A
// source-only scanner would silently treat this unresolved call as harmless.
func TestWritePathCensusFailsClosedOnUnresolvedCallee(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unresolved.go")
	if err := os.WriteFile(path, []byte("package censuscontrol\nfunc rogue() { unknownCallee() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCensusTypeUnit(dir); err == nil {
		t.Fatal("write-path census accepted an unresolved callee")
	}
}

// The control plant: the gate must flag a new unlocked caller in a
// synthetic package while passing the censused shapes.
func TestWritePathGateFlagsUnlockedCaller(t *testing.T) {
	t.Helper()
	write := func(t *testing.T, dir, name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		content := "package censuscontrol\n\n" + body
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("flags unlocked pair write", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "rogue.go", "func replaceDurably() {}\nfunc rogueApply() {\n\treplaceDurably()\n}\n")
		violations := censusViolations(t, dir)
		if len(violations) != 1 || !strings.Contains(violations[0], "rogueApply") {
			t.Fatalf("gate violations = %v, want exactly the rogueApply caller", violations)
		}
	})
	t.Run("flags unlocked trust write", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "rogue.go", "func commitDocument() {}\nfunc rogueBump() {\n\tcommitDocument()\n}\n")
		violations := censusViolations(t, dir)
		if len(violations) != 1 || !strings.Contains(violations[0], "rogueBump") {
			t.Fatalf("gate violations = %v, want exactly the rogueBump caller", violations)
		}
	})
	t.Run("flags unlocked raw rename", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "rogue.go", "type seam interface{ Rename(a, b string) }\nfunc rogueRename(s seam) {\n\ts.Rename(\"a\", \"b\")\n}\n")
		violations := censusViolations(t, dir)
		if len(violations) != 1 || !strings.Contains(violations[0], "rogueRename") {
			t.Fatalf("gate violations = %v, want exactly the rogueRename caller", violations)
		}
	})
	t.Run("passes censused shapes", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "clean.go", "func replaceDurably() {}\nfunc writeTempReplace() {}\n"+
			"type store struct{}\nfunc (s store) JointCommit(a, b, c func() error) {}\n"+
			"func (s store) WithExclusiveHold(fn func(int)) {}\n"+
			"func migrate(s store) {\n\ts.WithExclusiveHold(func(hold int) {\n\t\treplaceDurably()\n\t})\n}\n"+
			"func applyV4(s store) {\n\ts.JointCommit(func() error { return nil }, func() error {\n\t\treplaceDurably()\n\t\treturn nil\n\t}, func() error {\n\t\twriteTempReplace()\n\t\treturn nil\n\t})\n}\n")
		if violations := censusViolations(t, dir); len(violations) != 0 {
			t.Fatalf("gate violations = %v, want none for censused shapes", violations)
		}
	})
	t.Run("does not trust same-named hold on another receiver", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "impostor.go", "func replaceDurably() {}\ntype impostor struct{}\nfunc (impostor) WithExclusiveHold(fn func(int)) {}\nfunc rogue(s impostor) { s.WithExclusiveHold(func(int) { replaceDurably() }) }\n")
		violations := censusViolations(t, dir)
		if len(violations) == 0 || !strings.Contains(strings.Join(violations, "\n"), "rogue") {
			t.Fatalf("gate violations = %v, want the impostor receiver to fail", violations)
		}
	})
}

// Executable rogue controls: each evasion shape the CR6 review
// demonstrated (plus two producer shapes) is committed here twice —
// once as the exact production-position source the gate must flag, and
// once as proof that the flagged construct really mutates. Shapes that
// call a token-gated helper cannot compile in production position after
// rev8 (the helper requires a live hold only the lock path forges);
// their behavioral backstop is the zero-hold runtime refusal pinned
// below. Shapes that compile (direct os mutations) execute here against
// real temporary files and must change bytes.
func TestWritePathGateFlagsExecutableRogues(t *testing.T) {
	write := func(t *testing.T, dir, name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		content := "package censuscontrol\n\n" + body
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	assertFlagged := func(t *testing.T, shape, source, marker string) {
		t.Helper()
		dir := t.TempDir()
		write(t, dir, "reviewer_rogue.go", source+censusPlantPrelude)
		violations := censusViolations(t, dir)
		matched := false
		for _, violation := range violations {
			if strings.Contains(violation, marker) {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("%s: gate violations = %v, want one mentioning %q", shape, violations, marker)
		}
	}

	t.Run("direct control", func(t *testing.T) {
		assertFlagged(t, "direct control",
			"func reviewerRogue(path string, data []byte) error { _, err := writeTempReplace(osMigrationFileSystem{}, path, data); return err }\n",
			"reviewerRogue")
	})
	t.Run("package var closure", func(t *testing.T) {
		assertFlagged(t, "package var closure",
			"var reviewerRogue = func(path string, data []byte) error { _, err := writeTempReplace(osMigrationFileSystem{}, path, data); return err }\n",
			"reviewerRogue")
	})
	t.Run("local helper alias", func(t *testing.T) {
		assertFlagged(t, "local helper alias",
			"func reviewerRogue(path string, data []byte) error { writer := writeTempReplace; _, err := writer(osMigrationFileSystem{}, path, data); return err }\n",
			"reviewerRogue")
	})
	t.Run("alias import", func(t *testing.T) {
		assertFlagged(t, "alias import",
			"import censusOS \"os\"\nfunc reviewerRogue(path string, data []byte) error { return censusOS.WriteFile(path, data, 0600) }\n",
			"reviewerRogue")
	})
	t.Run("map function value", func(t *testing.T) {
		assertFlagged(t, "map function value",
			"import \"os\"\nvar reviewerWriters = map[string]func(string, []byte, os.FileMode) error{\"write\": os.WriteFile}\nfunc reviewerRogue(path string, data []byte) error { return reviewerWriters[\"write\"](path, data, 0600) }\n",
			"reviewerWriters")
	})
	t.Run("returned closure initializer", func(t *testing.T) {
		assertFlagged(t, "returned closure initializer",
			"import \"os\"\nvar reviewerFactory = func() func(string, []byte) error { return func(path string, data []byte) error { return os.WriteFile(path, data, 0600) } }()\nfunc reviewerRogue(path string, data []byte) error { return reviewerFactory(path, data) }\n",
			"reviewerFactory")
	})
	t.Run("method os rename", func(t *testing.T) {
		assertFlagged(t, "method os rename",
			"import \"os\"\ntype reviewerRogueWriter struct{}\nfunc (reviewerRogueWriter) replace(path string, data []byte) error { return os.Rename(path+\".replacement\", path) }\nfunc reviewerRogue(path string, data []byte) error { return (reviewerRogueWriter{}).replace(path,data) }\n",
			"replace")
		// The flagged construct executes: the same method shape replaces
		// a real file through os.Rename.
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("old configuration"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".replacement", []byte("unlocked replacement"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := (censusRogueWriter{}).replace(path, nil); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "unlocked replacement" {
			t.Fatal("method-receiver os.Rename witness did not change the file")
		}
	})
	t.Run("direct os write", func(t *testing.T) {
		assertFlagged(t, "direct os write",
			"import \"os\"\nfunc reviewerRogue(path string, data []byte) error { return os.WriteFile(path, data, 0600) }\n",
			"reviewerRogue")
		// The flagged construct executes: os.WriteFile replaces a real
		// file with no helper involved.
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("old configuration"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("direct replacement"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "direct replacement" {
			t.Fatal("direct os.WriteFile witness did not change the file")
		}
	})
	t.Run("backend receiver rogue", func(t *testing.T) {
		assertFlagged(t, "backend receiver rogue",
			"func (osMigrationFileSystem) rogueWrite(path string, data []byte) error { _, err := writeTempReplace(osMigrationFileSystem{}, path, data); return err }\n",
			"rogueWrite")
	})
	t.Run("backend method value", func(t *testing.T) {
		assertFlagged(t, "backend method value",
			"import \"os\"\nfunc (osMigrationFileSystem) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }\nvar reviewerRogue = osMigrationFileSystem{}.Rename\n",
			"Rename")
	})
	t.Run("dot-imported function value", func(t *testing.T) {
		assertFlagged(t, "dot-imported function value",
			"import . \"os\"\nvar reviewerWrite = WriteFile\nfunc reviewerRogue(path string, data []byte) error { return reviewerWrite(path, data, 0600) }\n",
			"WriteFile")
	})
	t.Run("promoted embedded backend method", func(t *testing.T) {
		assertFlagged(t, "promoted embedded backend method",
			"import \"os\"\ntype reviewerEmbeddedBackend struct{}\nfunc (reviewerEmbeddedBackend) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }\ntype reviewerPromotedBackend struct{ reviewerEmbeddedBackend }\nvar reviewerRogue = reviewerPromotedBackend{}.Rename\n",
			"Rename")
	})
	t.Run("package-level slice writer", func(t *testing.T) {
		assertFlagged(t, "package-level slice writer",
			"import \"os\"\nvar reviewerWriters = []func(string, []byte, os.FileMode) error{os.WriteFile}\nfunc reviewerRogue(path string, data []byte) error { return reviewerWriters[0](path, data, 0600) }\n",
			"reviewerWriters")
	})
	t.Run("function-typed struct field", func(t *testing.T) {
		assertFlagged(t, "function-typed struct field",
			"import \"os\"\ntype reviewerTable struct{ Write func(string, []byte, os.FileMode) error }\nvar reviewerWriters = reviewerTable{Write: os.WriteFile}\nfunc reviewerRogue(path string, data []byte) error { return reviewerWriters.Write(path, data, 0600) }\n",
			"reviewerWriters")
	})
	t.Run("function-typed parameter", func(t *testing.T) {
		assertFlagged(t, "function-typed parameter",
			"import \"os\"\nfunc reviewerInvoke(writer func(string, []byte, os.FileMode) error, path string, data []byte) error { return writer(path, data, 0600) }\nfunc reviewerRogue(path string, data []byte) error { return reviewerInvoke(os.WriteFile, path, data) }\n",
			"reviewerInvoke")
	})
	t.Run("dot-imported composite function value", func(t *testing.T) {
		assertFlagged(t, "dot-imported composite function value",
			"import . \"os\"\ntype reviewerTable struct{ Write func(string, []byte, FileMode) error }\nvar reviewerRecord = reviewerTable{Write: WriteFile}\nfunc reviewerRogue(path string, data []byte) error { return reviewerRecord.Write(path, data, 0600) }\n",
			"WriteFile")
	})
	t.Run("interface implementation", func(t *testing.T) {
		assertFlagged(t, "interface implementation",
			"import \"os\"\ntype reviewerWriter interface { Write(string, []byte) error }\ntype reviewerRogueWriter struct{}\nfunc (reviewerRogueWriter) Write(path string, data []byte) error { return os.WriteFile(path, data, 0600) }\nfunc reviewerRogue(w reviewerWriter, path string, data []byte) error { return w.Write(path, data) }\n",
			"Write")
	})
	t.Run("interface method spelling cannot evade the boundary", func(t *testing.T) {
		assertFlagged(t, "interface method spelling cannot evade the boundary",
			"type reviewerAction interface { Execute(string, []byte) error }\nfunc reviewerRogue(action reviewerAction, path string, data []byte) error { return action.Execute(path, data) }\n",
			"reviewerRogue")
	})
	t.Run("parenthesized interface callee cannot evade the boundary", func(t *testing.T) {
		assertFlagged(t, "parenthesized interface callee cannot evade the boundary",
			"type reviewerAction interface { Execute(string, []byte) error }\nfunc reviewerRogue(action reviewerAction, path string, data []byte) error { return (action.Execute)(path, data) }\n",
			"reviewerRogue")
	})
	t.Run("generic interface method cannot evade the boundary", func(t *testing.T) {
		assertFlagged(t, "generic interface method cannot evade the boundary",
			"type reviewerAction interface { Execute(string, []byte) error }\nfunc reviewerInvoke[T reviewerAction](action T, path string, data []byte) error { return action.Execute(path, data) }\nfunc reviewerRogue(action reviewerAction, path string, data []byte) error { return reviewerInvoke(action, path, data) }\n",
			"reviewerInvoke")
	})
	t.Run("generic helper", func(t *testing.T) {
		assertFlagged(t, "generic helper",
			"import \"os\"\nfunc reviewerGeneric[T ~string](path T, data []byte) error { return os.WriteFile(string(path), data, 0600) }\nfunc reviewerRogue(path string, data []byte) error { return reviewerGeneric(path, data) }\n",
			"reviewerGeneric")
	})
	t.Run("type-parameter method value", func(t *testing.T) {
		assertFlagged(t, "type-parameter method value",
			"type reviewerWriter interface { Write(string, []byte) error }\ntype reviewerWriterImpl struct{}\nfunc (reviewerWriterImpl) Write(string, []byte) error { return nil }\nfunc reviewerGenericMethodValue[T reviewerWriter](writer T, path string, data []byte) error { method := writer.Write; return method(path, data) }\nfunc reviewerRogue(path string, data []byte) error { return reviewerGenericMethodValue(reviewerWriterImpl{}, path, data) }\n",
			"reviewerGenericMethodValue")
	})
	t.Run("channel callback", func(t *testing.T) {
		assertFlagged(t, "channel callback",
			"import \"os\"\nfunc reviewerChannelRogue(path string, data []byte) error { callbacks := make(chan func(string, []byte, os.FileMode) error, 1); callbacks <- func(string, []byte, os.FileMode) error { return nil }; return (<-callbacks)(path, data, 0600) }\n",
			"reviewerChannelRogue")
	})
	t.Run("immediate function literal", func(t *testing.T) {
		assertFlagged(t, "immediate function literal",
			"func reviewerImmediateRogue(path string, data []byte) error { return (func(string, []byte) error { return nil })(path, data) }\n",
			"reviewerImmediateRogue")
	})
	t.Run("token gated shapes refuse at runtime", func(t *testing.T) {
		// The helper-calling shapes above cannot forge a live hold, so
		// even their compilable zero-token form fails closed here.
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("old configuration"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := writeTempReplace(zeroHoldForCensusTest(), osMigrationFileSystem{}, resolvedPathsForStoreTest(t, path, filepath.Join(filepath.Dir(path), "state")), []byte("rogue replacement")); err == nil {
			t.Fatal("writeTempReplace(zero hold) succeeded, want refusal")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "old configuration" {
			t.Fatal("zero-hold writeTempReplace changed the file")
		}
	})
	t.Run("function-value witnesses mutate real files", func(t *testing.T) {
		directory := t.TempDir()
		path := filepath.Join(directory, "config.toml")
		if err := os.WriteFile(path, []byte("old configuration"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := censusAliasImportWitness(path, []byte("alias replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusDotImportWitness(path, []byte("dot replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusMapFunctionWitness(path, []byte("map replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusReturnedClosureWitness(path, []byte("closure replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusInterfaceWitness(path, []byte("interface replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusGenericWitness(path, []byte("generic replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusGenericMethodWitness(path, []byte("generic method replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusChannelFunctionWitness(path, []byte("channel replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusImmediateFunctionWitness(path, []byte("immediate replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusSliceFunctionWitness(path, []byte("slice replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusStructFieldWitness(path, []byte("field replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusParameterFunctionWitness(path, []byte("parameter replacement")); err != nil {
			t.Fatal(err)
		}
		if err := censusCompositeFunctionWitness(path, []byte("composite replacement")); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "composite replacement" {
			t.Fatalf("function-value witnesses installed %q, want composite replacement", got)
		}
		if err := censusPromotedMethodWitness(path, []byte("promoted replacement")); err != nil {
			t.Fatal(err)
		}
		got, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "promoted replacement" {
			t.Fatalf("promoted method witness installed %q, want promoted replacement", got)
		}
		if err := os.WriteFile(path+".replacement", []byte("method value replacement"), 0o600); err != nil {
			t.Fatal(err)
		}
		methodValue := censusRogueWriter{}.replace
		if err := methodValue(path, nil); err != nil {
			t.Fatal(err)
		}
		got, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "method value replacement" {
			t.Fatalf("function-value witness installed %q, want method value replacement", got)
		}
	})
	t.Run("interface method witness mutates a real file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("old configuration"), 0o600); err != nil {
			t.Fatal(err)
		}
		var action censusExecuteAction = censusExecuteWriter{}
		if err := action.Execute(path, []byte("interface replacement")); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "interface replacement" {
			t.Fatalf("interface method witness installed %q, want interface replacement", got)
		}
		if err := censusExecuteInvoke(action, path, []byte("generic interface replacement")); err != nil {
			t.Fatal(err)
		}
		got, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "generic interface replacement" {
			t.Fatalf("generic interface method witness installed %q, want generic interface replacement", got)
		}
	})
}

const censusPlantPrelude = `
type osMigrationFileSystem struct{}
func writeTempReplace(osMigrationFileSystem, string, []byte) (string, error) { return "", nil }
`

type censusExecuteAction interface {
	Execute(string, []byte) error
}

type censusExecuteWriter struct{}

func (censusExecuteWriter) Execute(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func censusExecuteInvoke[T censusExecuteAction](action T, path string, data []byte) error {
	return action.Execute(path, data)
}

// censusRogueWriter mirrors the method-receiver evasion shape so the
// control above executes the construct the gate flags. It lives in a
// test file: fixtures write directly by design and are outside the gate.
type censusRogueWriter struct{}

func (censusRogueWriter) replace(path string, data []byte) error {
	return os.Rename(path+".replacement", path)
}

type censusEmbeddedWriter struct{}

func (censusEmbeddedWriter) write(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

type censusPromotedWriter struct{ censusEmbeddedWriter }

func censusPromotedMethodWitness(path string, data []byte) error {
	return censusPromotedWriter{}.write(path, data)
}

func censusMapFunctionWitness(path string, data []byte) error {
	writers := map[string]func(string, []byte, os.FileMode) error{"write": os.WriteFile}
	return writers["write"](path, data, 0o600)
}

func censusReturnedClosureWitness(path string, data []byte) error {
	writer := func() func(string, []byte) error {
		return func(path string, data []byte) error { return os.WriteFile(path, data, 0o600) }
	}()
	return writer(path, data)
}

type censusWriterInterface interface{ Write(string, []byte) error }

type censusInterfaceWriter struct{}

func (censusInterfaceWriter) Write(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func censusInterfaceWitness(path string, data []byte) error {
	var writer censusWriterInterface = censusInterfaceWriter{}
	return writer.Write(path, data)
}

func censusGenericWitness[T ~string](path T, data []byte) error {
	return os.WriteFile(string(path), data, 0o600)
}

type censusGenericMethodWriter interface{ Write(string, []byte) error }

type censusGenericMethodWriterImpl struct{}

func (censusGenericMethodWriterImpl) Write(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func censusGenericMethod[T censusGenericMethodWriter](writer T, path string, data []byte) error {
	method := writer.Write
	return method(path, data)
}

func censusGenericMethodWitness(path string, data []byte) error {
	return censusGenericMethod(censusGenericMethodWriterImpl{}, path, data)
}

func censusChannelFunctionWitness(path string, data []byte) error {
	callbacks := make(chan func(string, []byte, os.FileMode) error, 1)
	callbacks <- os.WriteFile
	return (<-callbacks)(path, data, 0o600)
}

func censusImmediateFunctionWitness(path string, data []byte) error {
	return (func(path string, data []byte) error { return os.WriteFile(path, data, 0o600) })(path, data)
}

var censusWriterSlice = []func(string, []byte, os.FileMode) error{os.WriteFile}

func censusSliceFunctionWitness(path string, data []byte) error {
	return censusWriterSlice[0](path, data, 0o600)
}

type censusFunctionFieldTable struct {
	Write func(string, []byte, os.FileMode) error
}

func censusStructFieldWitness(path string, data []byte) error {
	table := censusFunctionFieldTable{Write: os.WriteFile}
	return table.Write(path, data, 0o600)
}

func censusInvokeParameterWriter(writer func(string, []byte, os.FileMode) error, path string, data []byte) error {
	return writer(path, data, 0o600)
}

func censusParameterFunctionWitness(path string, data []byte) error {
	return censusInvokeParameterWriter(os.WriteFile, path, data)
}

type censusCompositeFunctionRecord struct {
	Write func(string, []byte, os.FileMode) error
}

func censusCompositeFunctionWitness(path string, data []byte) error {
	record := censusCompositeFunctionRecord{Write: os.WriteFile}
	return record.Write(path, data, 0o600)
}

// zeroHoldForCensusTest is the compilable form of a forged hold: the
// zero value, which every token-gated helper must refuse.
func zeroHoldForCensusTest() hosttrust.HeldExclusive {
	return hosttrust.HeldExclusive{}
}
