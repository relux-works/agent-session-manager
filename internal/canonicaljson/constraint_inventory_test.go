package canonicaljson

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type documentedConstraintRow struct {
	shape       string
	member      string
	constraint  string
	callSite    string
	specExcerpt string
}

// The shape names are documentation identities. Member names are deliberately
// absent: TestConstraintEnumerationMatchesRequireExactMembers derives those
// from the production requireExactMembers argument lists.
var documentedClosedShapeCalls = map[string][]string{
	"validateSessionRecordV1":                        {"Session Record 1.0.0"},
	"validateSessionRecordWithDerivation":            {"Session Record 2.0.0 and 3.0.0"},
	"validateSessionLaunchPlan":                      {"Session Record Launch Plan"},
	"validateSessionTaskBoardReference":              {"Session Record Task-board Reference"},
	"validateSessionBoardIdentity":                   {"Session Record Board Identity"},
	"validateSessionBoardGoal":                       {"Session Record Board Goal"},
	"validateSessionForkProvenance":                  {"Session Record Fork Provenance"},
	"validateSessionOriginProvenance":                {"Session Record origin provenance"},
	"validateSessionSameProviderForkProvenance":      {"Session Record same-provider-fork provenance"},
	"validateSessionCrossEnvironmentCloneProvenance": {"Session Record cross-environment-clone provenance"},
	"validateSessionNativeAdoptionProvenance":        {"Session Record native-adoption provenance"},
	"validateEnvironmentTuple":                       {"EnvironmentTuple"},
	"validateBlobDescriptor":                         {"Blob Descriptor", "BlobChunk"},
	"validateTransferManifest":                       {"Transfer Manifest"},
	"validateManifestEntries":                        {"ManifestEntry.directory", "ManifestEntry.file", "ManifestEntry.symlink", "ManifestEntry.hardlink"},
	"validateWorkspaceSnapshot":                      {"WorkspaceSnapshot"},
	"validateWorkspaceSnapshotMember":                {"WorkspaceSnapshotMember.managed_tree", "WorkspaceSnapshotMember.git"},
	"validateGitRemote":                              {"GitRemote"},
	"validateGitHead":                                {"GitHead"},
	"validateGitObjectPack":                          {"GitObjectPack"},
	"validateGitIndex":                               {"GitIndex"},
	"validateGitIndexEntry":                          {"GitIndexEntry"},
	"validateGitSubmodule":                           {"GitSubmodule"},
	"validateGitFeatures":                            {"GitFeatures"},
	"validateMigrationExtensionObject":               {"MigrationProvenance"},
	"validateLeaseRecord":                            {"Lease Record"},
	"validateCheckpointRecord":                       {"Checkpoint Record"},
	"validateSafeBoundaryEvidence":                   {"Safe Boundary Evidence"},
	"validateProviderIdentityRecord":                 {"Provider Identity Record"},
	"validateWorkspaceGroupRecord":                   {"Workspace Group Record"},
	"validateWorkspaceMember":                        {"WorkspaceMember.git", "WorkspaceMember.managed_tree"},
	"validateSessionEvent":                           {"Session Event"},
	"validateObservationEvent":                       {"Observation Event"},
	"validateObservationCountsMember":                {"ObservationCounts"},
}

// derivedMemberSetCallSites names every production requireExactMembers call
// whose member slice is computed rather than written as string literals, so the
// literal-argument inventory below cannot read it. Each entry names the test
// that pins that call's members instead. The set is asserted exactly: a new
// computed call site fails until it is declared here and given a pin.
var derivedMemberSetCallSites = map[string]string{
	"validateSessionEvent": "TestSessionEventPayloadMembersMatchPinnedSpecInventory",
}

// closedMemberRefusals are the refusal messages that define a closed-member
// gate. TestRequireExactMembersIsTheOnlyClosedMemberGate derives every function
// that emits them and requires that set to be exactly requireExactMembers, so a
// second copy of the helper cannot carry unenumerated member sets again.
var closedMemberRefusals = []string{"contains unknown member", "is missing required member"}

// packageProductionFiles derives the production sources of this package from
// the directory rather than a hand-written list, so a new non-test file is
// scanned by every inventory gate without anyone remembering to add it.
func packageProductionFiles(t *testing.T) (string, []string) {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve constraint inventory test source path")
	}
	packageDirectory := filepath.Dir(source)
	entries, err := os.ReadDir(packageDirectory)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(packageDirectory, name))
	}
	if len(files) == 0 {
		t.Fatal("derived zero production files for the canonicaljson package")
	}
	sort.Strings(files)
	return packageDirectory, files
}

func parseProductionFile(t *testing.T, path string) *ast.File {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// TestRequireExactMembersIsTheOnlyClosedMemberGate is the self-coverage
// assertion for the constraint inventory. The inventory walks requireExactMembers
// calls; that walk is only complete while requireExactMembers is the sole
// function enforcing a closed member set. A duplicate helper previously carried
// the WorkspaceMember, Session Event payload, Observation Event, and
// ObservationCounts member sets outside the inventory entirely, which is how an
// unenforced scalar type survived a green suite.
func TestRequireExactMembersIsTheOnlyClosedMemberGate(t *testing.T) {
	t.Parallel()

	_, files := packageProductionFiles(t)
	found := make(map[string]struct{})
	for _, path := range files {
		for _, declaration := range parseProductionFile(t, path).Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if emitsClosedMemberRefusal(function) {
				found[function.Name.Name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) != 1 || names[0] != "requireExactMembers" {
		t.Fatalf("closed-member enforcement must live in requireExactMembers alone, found %v; "+
			"every such function's member sets must appear in the constraint inventory", names)
	}
}

func emitsClosedMemberRefusal(function *ast.FuncDecl) bool {
	emits := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		text, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		for _, refusal := range closedMemberRefusals {
			if strings.Contains(text, refusal) {
				emits = true
			}
		}
		return true
	})
	return emits
}

// TestEveryDocumentedShapeValidatorIsProductionReachable refuses an inventory
// row whose validator is never reached from an exported entry point. An
// enumerated member set on a function nothing calls proves nothing about the
// identity gate.
func TestEveryDocumentedShapeValidatorIsProductionReachable(t *testing.T) {
	t.Parallel()

	_, files := packageProductionFiles(t)
	// Nodes are package-level functions and package-level variables. Variables
	// matter because several validators reach production only as function values
	// inside a package-level dispatch registry, never as a direct call.
	references := make(map[string]map[string]struct{})
	declaredFunctions := make(map[string]struct{})
	var exported []string
	collect := func(node ast.Node) map[string]struct{} {
		used := make(map[string]struct{})
		ast.Inspect(node, func(inner ast.Node) bool {
			if identifier, ok := inner.(*ast.Ident); ok {
				used[identifier.Name] = struct{}{}
			}
			return true
		})
		return used
	}
	for _, path := range files {
		for _, declaration := range parseProductionFile(t, path).Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				if typed.Body == nil {
					continue
				}
				name := typed.Name.Name
				declaredFunctions[name] = struct{}{}
				if typed.Recv == nil && typed.Name.IsExported() {
					exported = append(exported, name)
				}
				references[name] = collect(typed.Body)
			case *ast.GenDecl:
				if typed.Tok != token.VAR {
					continue
				}
				for _, specification := range typed.Specs {
					value, ok := specification.(*ast.ValueSpec)
					if !ok {
						continue
					}
					used := make(map[string]struct{})
					for _, initializer := range value.Values {
						for identifier := range collect(initializer) {
							used[identifier] = struct{}{}
						}
					}
					for _, name := range value.Names {
						references[name.Name] = used
					}
				}
			}
		}
	}
	if len(exported) == 0 {
		t.Fatal("derived zero exported entry points for the canonicaljson package")
	}

	reachable := make(map[string]struct{})
	queue := append([]string(nil), exported...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, seen := reachable[name]; seen {
			continue
		}
		reachable[name] = struct{}{}
		for referenced := range references[name] {
			if _, known := references[referenced]; !known {
				continue
			}
			queue = append(queue, referenced)
		}
	}

	inventoried := make(map[string]struct{})
	for name := range documentedClosedShapeCalls {
		inventoried[name] = struct{}{}
	}
	for name := range derivedMemberSetCallSites {
		inventoried[name] = struct{}{}
	}
	var unreachable []string
	for name := range inventoried {
		if _, isFunction := declaredFunctions[name]; !isFunction {
			unreachable = append(unreachable, name+" (not a production function)")
			continue
		}
		if _, ok := reachable[name]; !ok {
			unreachable = append(unreachable, name)
		}
	}
	sort.Strings(unreachable)
	if len(unreachable) != 0 {
		t.Fatalf("inventoried validators are not reachable from any exported entry point: %s",
			strings.Join(unreachable, ", "))
	}
}

func TestConstraintEnumerationMatchesRequireExactMembers(t *testing.T) {
	t.Parallel()

	packageDirectory, files := packageProductionFiles(t)
	expected := make(map[string]string)
	derived := make(map[string]int)
	for _, path := range files {
		for _, declaration := range parseProductionFile(t, path).Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			literalCalls, derivedCalls := requireExactMemberCalls(t, function)
			if derivedCalls > 0 {
				derived[function.Name.Name] = derivedCalls
			}
			shapes, tracked := documentedClosedShapeCalls[function.Name.Name]
			if !tracked {
				if len(literalCalls) != 0 {
					t.Fatalf("%s has undocumented requireExactMembers calls", function.Name.Name)
				}
				continue
			}
			if len(literalCalls) != len(shapes) {
				t.Fatalf("%s has %d literal requireExactMembers calls, documentation mapping has %d", function.Name.Name, len(literalCalls), len(shapes))
			}
			for index, members := range literalCalls {
				for _, member := range members {
					key := shapes[index] + "." + member
					if previous, duplicate := expected[key]; duplicate {
						t.Fatalf("production member %s is duplicated at %s and %s", key, previous, function.Name.Name)
					}
					expected[key] = function.Name.Name
				}
			}
		}
	}

	// Every computed member slice must be declared and pinned elsewhere; an
	// undeclared one would otherwise pass through the inventory unenumerated.
	var derivedNames, declaredNames []string
	for name := range derived {
		derivedNames = append(derivedNames, name)
	}
	for name := range derivedMemberSetCallSites {
		declaredNames = append(declaredNames, name)
	}
	sort.Strings(derivedNames)
	sort.Strings(declaredNames)
	if strings.Join(derivedNames, ",") != strings.Join(declaredNames, ",") {
		t.Fatalf("production computed requireExactMembers call sites are %v, declared %v; "+
			"a computed member slice must be declared and pinned by a named derivation test", derivedNames, declaredNames)
	}

	rows := readConstraintRows(t, filepath.Join(packageDirectory, "testdata", "constraint-enumeration.md"))
	actual := make(map[string]documentedConstraintRow, len(rows))
	for _, row := range rows {
		key := row.shape + "." + row.member
		if _, duplicate := actual[key]; duplicate {
			t.Fatalf("constraint enumeration duplicates %s", key)
		}
		if row.constraint == "" || row.specExcerpt == "" {
			t.Fatalf("constraint enumeration row %s requires a disposition and quoted spec text", key)
		}
		actual[key] = row
	}

	var mismatches []string
	for key, callSite := range expected {
		row, ok := actual[key]
		if !ok {
			mismatches = append(mismatches, "missing artifact row "+key)
			continue
		}
		if row.callSite != callSite {
			mismatches = append(mismatches, key+" names call site "+row.callSite+", want "+callSite)
		}
	}
	for key := range actual {
		if _, ok := expected[key]; !ok {
			mismatches = append(mismatches, "artifact row has no production requireExactMembers member "+key)
		}
	}
	sort.Strings(mismatches)
	if len(mismatches) != 0 {
		t.Fatalf("constraint enumeration drift:\n%s", strings.Join(mismatches, "\n"))
	}
}

// requireExactMemberCalls returns the member lists of every literal-argument
// requireExactMembers call in the function, plus the number of calls whose
// member slice is computed at run time.
func requireExactMemberCalls(t *testing.T, function *ast.FuncDecl) ([][]string, int) {
	t.Helper()
	var calls [][]string
	computed := 0
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok || identifier.Name != "requireExactMembers" {
			return true
		}
		if len(call.Args) < 3 {
			t.Fatalf("%s has malformed requireExactMembers call", function.Name.Name)
		}
		if call.Ellipsis.IsValid() {
			computed++
			return true
		}
		members := make([]string, 0, len(call.Args)-2)
		for _, argument := range call.Args[2:] {
			literal, ok := argument.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatalf("%s requireExactMembers member is not a string literal", function.Name.Name)
			}
			member, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			members = append(members, member)
		}
		calls = append(calls, members)
		return true
	})
	return calls, computed
}

func readConstraintRows(t *testing.T, path string) []documentedConstraintRow {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var rows []documentedConstraintRow
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) != 7 {
			t.Fatalf("constraint enumeration row has %d cells, want 5: %s", len(cells)-2, line)
		}
		trimCode := func(value string) string {
			return strings.Trim(strings.TrimSpace(value), "`")
		}
		rows = append(rows, documentedConstraintRow{
			shape:       trimCode(cells[1]),
			member:      trimCode(cells[2]),
			constraint:  strings.TrimSpace(cells[3]),
			callSite:    trimCode(cells[4]),
			specExcerpt: strings.TrimSpace(cells[5]),
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}
