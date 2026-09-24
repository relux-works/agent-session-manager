package merkleinventory

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

func TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord(t *testing.T) {
	caseIndex := 0
	for skew := 0; skew < 2; skew++ {
		common := generatedSyncPropertyObjects(t, 3, skew)
		sort.Slice(common, func(left, right int) bool { return common[left].id < common[right].id })
		peerOnly := generatedPeerOnlyLeaseObjects(t, skew)
		for _, order := range syncPropertyOrders(len(common)) {
			for duplicate := 0; duplicate < 2; duplicate++ {
				for targetPosition := range common {
					for conflictRound := 1; conflictRound <= 3; conflictRound++ {
						for peerOnlyIndex := range peerOnly {
							tc := struct {
								index, skew, duplicate, targetPosition, conflictRound, peerOnlyIndex int
								order                                                                []int
							}{caseIndex, skew, duplicate, targetPosition, conflictRound, peerOnlyIndex, append([]int(nil), order...)}
							caseIndex++
							name := fmt.Sprintf("position%d-round%d-peerOnly%d-skew%d-order%v-duplicate%t",
								tc.targetPosition, tc.conflictRound, tc.peerOnlyIndex, tc.skew, tc.order, tc.duplicate == 1)
							t.Run(name, func(t *testing.T) {
								t.Parallel()
								runDurableSyncCommonIDPositionCase(t, common, peerOnly[tc.peerOnlyIndex], tc.order,
									tc.duplicate == 1, tc.targetPosition, tc.conflictRound, tc.index)
							})
						}
					}
				}
			}
		}
	}
	if caseIndex != 432 {
		t.Fatalf("generated common-ID audit shard has %d cases, want 432", caseIndex)
	}
}

func generatedPeerOnlyLeaseObjects(t *testing.T, skew int) []syncPropertyObject {
	t.Helper()
	times := [2]string{"1900-01-01T00:00:00.000Z", "2099-12-31T23:59:59.999Z"}
	ids := [2]string{
		"cccccccc-dddd-4eee-8fff-111111111111",
		"dddddddd-eeee-4fff-8aaa-222222222222",
	}
	objects := make([]syncPropertyObject, 0, len(ids))
	for index, leaseID := range ids {
		createdAt := times[(index+skew)%len(times)]
		data := unionLeaseJSON(t, leaseID, testHostID, createdAt)
		objects = append(objects, syncPropertyObject{
			namespace: "record",
			id:        referenceJSONField(t, data, "record_id"),
			data:      data,
			lease:     sessstate.LeaseHead{Epoch: 1, LeaseID: leaseID},
		})
	}
	return objects
}

func runDurableSyncCommonIDPositionCase(
	t *testing.T,
	common []syncPropertyObject,
	peerOnly syncPropertyObject,
	order []int,
	duplicate bool,
	targetPosition int,
	conflictRound int,
	caseIndex int,
) {
	t.Helper()
	target := common[targetPosition]
	variant := append(bytes.Clone(target.data), '\n')
	local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
	if err != nil {
		t.Fatalf("case %d OpenDurable(local): %v", caseIndex, err)
	}
	localOrder := append([]int(nil), order...)
	slices.Reverse(localOrder)
	for _, index := range localOrder {
		addSyncPropertyObject(t, local, common[index], duplicate, caseIndex, "local common set")
	}

	for pass := 1; pass < conflictRound; pass++ {
		peer, err := OpenDurable(filepath.Join(t.TempDir(), fmt.Sprintf("peer-before-conflict-%d", pass)))
		if err != nil {
			t.Fatalf("case %d OpenDurable(peer pass %d): %v", caseIndex, pass, err)
		}
		for _, index := range order {
			if index != targetPosition {
				addSyncPropertyObject(t, peer, common[index], duplicate, caseIndex, "peer before gap fill")
			}
		}
		before := snapshotDurableStore(t, local.root)
		if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
			t.Fatalf("case %d pass %d before conflict: %v", caseIndex, pass, err)
		}
		if after := snapshotDurableStore(t, local.root); !reflect.DeepEqual(after, before) {
			t.Fatalf("case %d pass %d partial peer changed the local durable store", caseIndex, pass)
		}
		assertSyncRootsMatchReference(t, local, common, caseIndex, pass)
	}

	peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer-conflict"))
	if err != nil {
		t.Fatalf("case %d OpenDurable(conflict peer): %v", caseIndex, err)
	}
	for _, index := range order {
		object := common[index]
		if index == targetPosition {
			object.data = variant
		}
		addSyncPropertyObject(t, peer, object, duplicate, caseIndex, "peer conflict set")
	}
	addSyncPropertyObject(t, peer, peerOnly, duplicate, caseIndex, "peer-only record")
	peerBefore := snapshotDurableStore(t, peer.root)
	err = local.SyncFrom(peer, testRequestIDs())
	if !hasLiteralSyncConflict(err, "integrity_failure") {
		t.Fatalf("case %d target position %d pass %d SyncFrom = %v, want literal integrity_failure",
			caseIndex, targetPosition, conflictRound, err)
	}
	assertSyncQuarantineBytes(t, local, peer, target, variant)
	if _, err := local.Object(peerOnly.namespace, peerOnly.id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("case %d imported peer-only record before abort: %v", caseIndex, err)
	}
	if after := snapshotDurableStore(t, peer.root); !reflect.DeepEqual(after, peerBefore) {
		t.Fatalf("case %d conflict changed peer durable bytes", caseIndex)
	}
	if err := local.SyncFrom(peer, testRequestIDs()); !hasLiteralSyncConflict(err, "integrity_failure") {
		t.Fatalf("case %d repeated SyncFrom conflict = %v, want literal integrity_failure", caseIndex, err)
	}
}

func TestNoUnjustifiedTestSkips(t *testing.T) {
	root := os.Getenv("MERKLE_TEST_SKIP_SCAN_DIR")
	if root == "" {
		root = "."
	}
	violations, err := unallowedTestSkips(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("unallowlisted test skips: %s", strings.Join(violations, "; "))
	}
}

func TestSkipCensusRejectsAnUnjustifiedPlantedSkip(t *testing.T) {
	root := t.TempDir()
	fixture := "package merkleinventory\n" +
		"import \"testing\"\n" +
		"func TestPlantedUnjustifiedSkip(t *testing.T) {\n" +
		"\tt.Skip(\"planted without a platform or environment capability reason\")\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(root, "planted_test.go"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, err := unallowedTestSkips(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "TestPlantedUnjustifiedSkip") {
		t.Fatalf("planted skip census = %v, want one rejection for TestPlantedUnjustifiedSkip", violations)
	}
}

func TestSkipCensusRejectsAPlantedSkipWhenItsTokenRemains(t *testing.T) {
	root := t.TempDir()
	fixture := "package merkleinventory\n" +
		"import \"testing\"\n" +
		"func TestPlantedTokenPreservingSkip(t *testing.T) {\n" +
		"\tt.Skip(\"planted without a platform or environment capability reason\")\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(root, "planted_test.go"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, err := unallowedTestSkips(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "TestPlantedTokenPreservingSkip") {
		t.Fatalf("token-preserving skip census = %v, want one rejection for TestPlantedTokenPreservingSkip", violations)
	}
}

type testSkipAllowance struct {
	skipReason   string
	reason       string
	siblingTest  string
	propertyName string
}

var allowedTestSkips = map[string]testSkipAllowance{}

func unallowedTestSkips(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read skip census directory %s: %w", root, err)
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			files = append(files, filepath.Join(root, entry.Name()))
		}
	}
	slices.Sort(files)
	fset := token.NewFileSet()
	declaredTests := make(map[string]bool)
	type skipSite struct {
		file, test, method, reason string
	}
	sites := make([]skipSite, 0)
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse skip census source %s: %w", path, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if strings.HasPrefix(function.Name.Name, "Test") {
				declaredTests[function.Name.Name] = true
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !slices.Contains([]string{"Skip", "Skipf", "SkipNow"}, selector.Sel.Name) {
					return true
				}
				reason := ""
				if len(call.Args) > 0 {
					if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
						reason, _ = strconv.Unquote(literal.Value)
					}
				}
				sites = append(sites, skipSite{
					file: filepath.Base(path), test: function.Name.Name,
					method: selector.Sel.Name, reason: reason,
				})
				return true
			})
		}
	}
	violations := make([]string, 0)
	for _, site := range sites {
		key := site.file + "::" + site.test + "::" + site.method
		allowance, ok := allowedTestSkips[key]
		if !ok {
			violations = append(violations, fmt.Sprintf("%s:%s calls t.%s(%q) without an allowlist entry", site.file, site.test, site.method, site.reason))
			continue
		}
		justification := strings.ToLower(allowance.reason)
		if site.reason == "" || site.reason != allowance.skipReason ||
			!(strings.Contains(justification, "platform") || strings.Contains(justification, "environment capability")) {
			violations = append(violations, fmt.Sprintf("%s:%s has no literal reason and platform/environment capability justification", site.file, site.test))
			continue
		}
		if allowance.siblingTest == "" || allowance.propertyName == "" || !declaredTests[allowance.siblingTest] {
			violations = append(violations, fmt.Sprintf("%s:%s has no declared always-on sibling test for %q", site.file, site.test, allowance.propertyName))
		}
	}
	sort.Strings(violations)
	return violations, nil
}

func TestPackageTreeHasNoPythonBytecode(t *testing.T) {
	root := filepath.Join("..", "..")
	violations, err := pythonBytecodeArtifacts(root)
	if err != nil {
		t.Fatalf("walk repository tree for Python bytecode: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("Python bytecode artifacts found in repository tree: %s", strings.Join(violations, ", "))
	}
}

func TestPythonBytecodeHygieneRejectsPlantedArtifacts(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "__pycache__")
	if err := os.Mkdir(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "mutations.cpython-314.pyc"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "loose.pyc"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, err := pythonBytecodeArtifacts(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(violations, cache) || !slices.Contains(violations, filepath.Join(root, "loose.pyc")) {
		t.Fatalf("planted bytecode census = %v, want both cache directory and loose .pyc file", violations)
	}
}

func TestPythonBytecodeHygieneIgnoresGitignoredScratch(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{filepath.Join(root, ".temp", "__pycache__"), filepath.Join(root, ".git", "__pycache__")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "scratch.cpython-314.pyc"), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	violations, err := pythonBytecodeArtifacts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("scratch under .temp/.git reported as tracked-tree bytecode: %v", violations)
	}
}

func pythonBytecodeArtifacts(root string) ([]string, error) {
	violations := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// .git and the repository's gitignored .temp scratch are not part of
		// the tracked tree; orchestrators and producers run Python there.
		if entry.IsDir() && path != root && (entry.Name() == ".git" || entry.Name() == ".temp") {
			return filepath.SkipDir
		}
		if entry.IsDir() && entry.Name() == "__pycache__" {
			violations = append(violations, path)
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pyc") {
			violations = append(violations, path)
		}
		return nil
	})
	sort.Strings(violations)
	return violations, err
}
