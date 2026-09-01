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
	"validateSessionRecordV1":           {"Session Record 1.0.0"},
	"validateSessionLaunchPlan":         {"Session Record Launch Plan"},
	"validateSessionTaskBoardReference": {"Session Record Task-board Reference"},
	"validateSessionBoardIdentity":      {"Session Record Board Identity"},
	"validateSessionBoardGoal":          {"Session Record Board Goal"},
	"validateSessionForkProvenance":     {"Session Record Fork Provenance"},
	"validateBlobDescriptor":            {"Blob Descriptor", "BlobChunk"},
	"validateTransferManifest":          {"Transfer Manifest"},
	"validateManifestEntries":           {"ManifestEntry.directory", "ManifestEntry.file", "ManifestEntry.symlink", "ManifestEntry.hardlink"},
	"validateWorkspaceSnapshot":         {"WorkspaceSnapshot"},
	"validateWorkspaceSnapshotMember":   {"WorkspaceSnapshotMember.managed_tree", "WorkspaceSnapshotMember.git"},
	"validateGitRemote":                 {"GitRemote"},
	"validateGitHead":                   {"GitHead"},
	"validateGitObjectPack":             {"GitObjectPack"},
	"validateGitIndex":                  {"GitIndex"},
	"validateGitIndexEntry":             {"GitIndexEntry"},
	"validateGitSubmodule":              {"GitSubmodule"},
	"validateGitFeatures":               {"GitFeatures"},
	"validateMigrationExtensionObject":  {"MigrationProvenance"},
}

func TestConstraintEnumerationMatchesRequireExactMembers(t *testing.T) {
	t.Parallel()

	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve constraint inventory test source path")
	}
	packageDirectory := filepath.Dir(source)
	productionPath := filepath.Join(packageDirectory, "closed_shapes.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), productionPath, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	expected := make(map[string]string)
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		calls := requireExactMemberCalls(t, function)
		shapes, tracked := documentedClosedShapeCalls[function.Name.Name]
		if !tracked {
			if len(calls) != 0 {
				t.Fatalf("%s has undocumented requireExactMembers calls", function.Name.Name)
			}
			continue
		}
		if len(calls) != len(shapes) {
			t.Fatalf("%s has %d requireExactMembers calls, documentation mapping has %d", function.Name.Name, len(calls), len(shapes))
		}
		for index, members := range calls {
			for _, member := range members {
				key := shapes[index] + "." + member
				if previous, duplicate := expected[key]; duplicate {
					t.Fatalf("production member %s is duplicated at %s and %s", key, previous, function.Name.Name)
				}
				expected[key] = function.Name.Name
			}
		}
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

func requireExactMemberCalls(t *testing.T, function *ast.FuncDecl) [][]string {
	t.Helper()
	var calls [][]string
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
	return calls
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
