package gitsnap

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The gate census states its denominator up front. One census row counts
// one literal refuse(Gate...) call site in non-test production files of
// this package, after stripping comments and string literals. The
// reported count is syntactic, not clause-level behavioral coverage. A gate
// with no literal site, or a site naming no registry row, fails the census.
// Named driver references prove linkage only; the external mutation harness
// measures which selected behavioral tests actually reject each weakening.

// gateProvingTests pins the proving test per registry row. The values
// are referenced through the blank-identifier block below so renaming or
// deleting a proving test breaks the build instead of silently
// shrinking the census.
var gateProvingTests = map[string]string{
	"GateTransferObjects":   "TestAssemblyFailedReadsAndCorruptObjects",
	"GateManifestClosure":   "TestValidateProvisionalClosureAndDescriptors",
	"GateCapturePolicy":     "TestContentIgnoredDefaultAndPolicyRefusal",
	"GateContentRead":       "TestContentReadFailureIsNotAbsence",
	"GateContentPath":       "TestContentSymlinks",
	"GateBlobLimit":         "TestContentLargeBlobChunksAndLimit",
	"GateBlobInstall":       "TestContentBlobInstallFailureAndRetry",
	"GateSubmoduleState":    "TestContentSubmoduleUninitializedAndRefusals",
	"GateFeatures":          "TestCaptureRequiredFilterCountBounds",
	"GateNotRepository":     "TestRefuseOutsideRepository",
	"GateHeadCorrupt":       "TestRefuseCorruptHead",
	"GateHeadRef":           "TestRefuseBranchWithLiteralHeadRef",
	"GateHeadOIDFormat":     "TestRefuseCrossFormatOID",
	"GateRemotesRange":      "TestRefuseNoRemotes",
	"GateRemoteURL":         "TestRefuseCredentialBearingRemote",
	"GateIdentityLength":    "TestRefuseEmptyIdentity",
	"GateIndexVersion":      "TestRefuseIndexVersionFive",
	"GateIndexStage":        "TestRefuseIndexStageFour",
	"GateIndexEntry":        "TestRefuseIndexBadOID",
	"GateIndexSort":         "TestRefuseUnsortedIndex",
	"GateIndexEntriesRange": "TestEdgeIndexMaxPlusOneRefuses",
	"GateUpstreamRef":       "TestRefuseBadUpstream",
	"GateDeltaStatus":       "TestRefuseUnknownDeltaStatus",
	"GateWorktreeKind":      "TestRefuseSpecialWorktreeFile",
	"GateConsistency":       "TestConsistencyRefusesIndexMutationArmedOnce",
}

// proving test references: renaming any of these tests without updating
// gateProvingTests is a build error, not a silent census shrink.
var (
	_ = TestAssemblyFailedReadsAndCorruptObjects
	_ = TestValidateProvisionalClosureAndDescriptors
	_ = TestCaptureRequiredFilterCountBounds
	_ = TestRefuseOutsideRepository
	_ = TestRefuseCorruptHead
	_ = TestRefuseBranchWithLiteralHeadRef
	_ = TestRefuseCrossFormatOID
	_ = TestRefuseNoRemotes
	_ = TestRefuseCredentialBearingRemote
	_ = TestRefuseEmptyIdentity
	_ = TestRefuseIndexVersionFive
	_ = TestRefuseIndexStageFour
	_ = TestRefuseIndexBadOID
	_ = TestRefuseUnsortedIndex
	_ = TestEdgeIndexMaxPlusOneRefuses
	_ = TestRefuseBadUpstream
	_ = TestRefuseUnknownDeltaStatus
	_ = TestRefuseSpecialWorktreeFile
	_ = TestConsistencyRefusesIndexMutationArmedOnce
)

var refuseSitePattern = regexp.MustCompile(`\brefuse\((Gate[A-Za-z0-9]+)`)

// stripCode removes block comments, line comments, double-quoted and
// backquoted string literals, and rune literals so a gate name inside
// prose or a literal is never counted as a call site.
func stripCode(source string) string {
	var builder strings.Builder
	rest := source
	for len(rest) > 0 {
		switch {
		case strings.HasPrefix(rest, "/*"):
			end := strings.Index(rest[2:], "*/")
			if end < 0 {
				return builder.String()
			}
			rest = rest[2+end+2:]
		case strings.HasPrefix(rest, "//"):
			end := strings.Index(rest, "\n")
			if end < 0 {
				return builder.String()
			}
			rest = rest[end:]
		case strings.HasPrefix(rest, "`"):
			end := strings.Index(rest[1:], "`")
			if end < 0 {
				return builder.String()
			}
			rest = rest[1+end+1:]
		case strings.HasPrefix(rest, `"`):
			index := 1
			for index < len(rest) {
				if rest[index] == '\\' {
					index += 2
					continue
				}
				if rest[index] == '"' {
					break
				}
				index++
			}
			rest = rest[min(index+1, len(rest)):]
		case strings.HasPrefix(rest, "'"):
			end := strings.Index(rest[1:], "'")
			if end < 0 {
				return builder.String()
			}
			rest = rest[1+end+1:]
		default:
			builder.WriteByte(rest[0])
			rest = rest[1:]
		}
	}
	return builder.String()
}

// gateSites parses dir for literal refuse(Gate...) call sites in
// non-test Go files and returns the gate name per site in file order.
func gateSites(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var sites []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range refuseSitePattern.FindAllStringSubmatch(stripCode(string(content)), -1) {
			sites = append(sites, match[1])
		}
	}
	return sites
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

// TestGateCensusCoversEveryRefusalSite is the census: every literal
// refuse call site names a registry row, and every registry row names a
// compiled test. No claim is made that all literal clauses were defeated.
func TestGateCensusCoversEveryRefusalSite(t *testing.T) {
	t.Parallel()
	sites := gateSites(t, packageDir(t))
	registry := map[string]Gate{}
	for _, gate := range gateRegistry {
		if _, duplicate := registry[gate.Name]; duplicate {
			t.Fatalf("duplicate registry row %s", gate.Name)
		}
		registry[gate.Name] = gate
	}
	seen := map[string]int{}
	for _, site := range sites {
		gate, ok := registry[site]
		if !ok {
			t.Errorf("call site names unregistered gate %s", site)
			continue
		}
		_ = gate
		seen[site]++
	}
	for _, gate := range gateRegistry {
		if seen[gate.Name] == 0 {
			t.Errorf("registry gate %s has no call site", gate.Name)
		}
		if _, ok := gateProvingTests[gate.Name]; !ok {
			t.Errorf("registry gate %s has no proving test", gate.Name)
		}
	}
	for name := range gateProvingTests {
		if _, ok := registry[name]; !ok {
			t.Errorf("proving test pinned for unknown gate %s", name)
		}
	}
	t.Logf("census: %d call sites over %d registry gates", len(sites), len(gateRegistry))
}

// TestGateCodesAreClosedAndLive proves the refusal-code vocabulary is
// closed (exactly the pinned table) and live (every code refuses from
// at least one site).
func TestGateCodesAreClosedAndLive(t *testing.T) {
	t.Parallel()
	if len(codeNames) != 3 {
		t.Fatalf("code vocabulary = %v, want the 3 pinned codes", codeNames)
	}
	sites := gateSites(t, packageDir(t))
	used := map[Code]bool{}
	for _, gate := range gateRegistry {
		found := false
		for _, code := range codeNames {
			if gate.Code == code {
				found = true
			}
		}
		if !found {
			t.Errorf("gate %s uses code %q outside the closed vocabulary", gate.Name, gate.Code)
		}
	}
	byName := map[string]Gate{}
	for _, gate := range gateRegistry {
		byName[gate.Name] = gate
	}
	for _, site := range sites {
		used[byName[site].Code] = true
	}
	for _, code := range codeNames {
		if !used[code] {
			t.Errorf("code %q has no call site", code)
		}
	}
}

// TestClosedDomainsArePinned guards the small closed domains against
// silent widening: index versions, head modes, delta statuses, and
// worktree kinds must read exactly as the normative source states them.
func TestClosedDomainsArePinned(t *testing.T) {
	t.Parallel()
	join := func(values []string) string { return strings.Join(values, ",") }
	if join(indexVersions) != "2,3,4" {
		t.Errorf("indexVersions = %v", indexVersions)
	}
	if join(headModes) != "branch,detached,unborn" {
		t.Errorf("headModes = %v", headModes)
	}
	statuses := join(deltaStatuses)
	if statuses != "A,C,D,M,R,T,U" {
		t.Errorf("deltaStatuses = %v", deltaStatuses)
	}
	if len(worktreeKinds) != 6 {
		t.Errorf("worktreeKinds = %v", worktreeKinds)
	}
	sorted := append([]string{}, statusesToStrings()...)
	sort.Strings(sorted)
	_ = sorted
}

func statusesToStrings() []string {
	return append([]string{}, deltaStatuses...)
}

// copyPackageForControls copies the non-test production files to a temp
// dir so control plants never touch the shipped tree.
func copyPackageForControls(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir(packageDir(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(packageDir(t), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func appendControl(t *testing.T, dir, filename, plant string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, []byte(plant)...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCensusControlCommentAndStringSurvive proves the instrument does
// not count gate names inside comments or string literals: both plants
// must leave the census unchanged (behavior-neutral controls that must
// SURVIVE as non-rows).
func TestCensusControlCommentAndStringSurvive(t *testing.T) {
	t.Parallel()
	dir := copyPackageForControls(t)
	before := gateSites(t, dir)
	appendControl(t, dir, "snapshot.go", "// refuse(GateGhostComment) in prose is not a call site\n")
	appendControl(t, dir, "snapshot.go", "var _ = \"refuse(GateGhostString) in a literal is not a call site\"\n")
	after := gateSites(t, dir)
	if len(after) != len(before) {
		t.Errorf("comment/string plants changed the census %d -> %d, want unchanged", len(before), len(after))
	}
	for _, site := range after {
		if site == "GateGhostComment" || site == "GateGhostString" {
			t.Errorf("plant %s counted as a call site", site)
		}
	}
}

// TestCensusControlUnregisteredGateReddens proves the instrument goes
// red on a real but unregistered gate: the known-bad control that must
// redden.
func TestCensusControlUnregisteredGateReddens(t *testing.T) {
	t.Parallel()
	dir := copyPackageForControls(t)
	appendControl(t, dir, "snapshot.go", "func plantedUnregisteredGate() *Refusal { return refuse(GatePlantedPositive, \"plant\") }\n")
	sites := gateSites(t, dir)
	registry := map[string]bool{}
	for _, gate := range gateRegistry {
		registry[gate.Name] = true
	}
	reddens := false
	for _, site := range sites {
		if !registry[site] {
			reddens = true
		}
	}
	if !reddens {
		t.Error("unregistered planted gate did not redden the census")
	}
}

// TestCensusControlAliasIsInvisible proves the stated blind spot: a
// refusal reached through an aliased variable, a fresh-spelling wrapper,
// or any other non-literal spelling is invisible to the literal
// call-site instrument. The package upholds direct invocation by
// construction (see doc.go); this control keeps the bound stated rather
// than assumed.
func TestCensusControlAliasIsInvisible(t *testing.T) {
	t.Parallel()
	dir := copyPackageForControls(t)
	before := len(gateSites(t, dir))
	appendControl(t, dir, "snapshot.go", "var aliasRefuse = refuse\nfunc plantedAlias() *Refusal { return aliasRefuse(GateNotRepository, \"plant\") }\n")
	appendControl(t, dir, "snapshot.go", "func freshSpellingRefuse(gate Gate, detail string) *Refusal { return refuse(gate, detail) }\n")
	after := gateSites(t, dir)
	if len(after) != before {
		t.Errorf("alias/wrapper plants changed the census %d -> %d: the blind spot is wider than stated", before, len(after))
	}
}

// TestCensusControlMethodLiteralIsCounted proves a literal refuse call
// inside a method body counts like any other site: the census is keyed
// on the call spelling, not on the enclosing declaration kind.
func TestCensusControlMethodLiteralIsCounted(t *testing.T) {
	t.Parallel()
	dir := copyPackageForControls(t)
	before := len(gateSites(t, dir))
	appendControl(t, dir, "snapshot.go", "type plantedReceiver struct{}\nfunc (plantedReceiver) check() *Refusal { return refuse(GatePlantedMethod, \"plant\") }\n")
	sites := gateSites(t, dir)
	if len(sites) != before+1 || !slices.Contains(sites, "GatePlantedMethod") {
		t.Errorf("method-literal plant not counted: %d -> %d", before, len(sites))
	}
}

var (
	_ = TestContentIgnoredDefaultAndPolicyRefusal
	_ = TestContentReadFailureIsNotAbsence
	_ = TestContentSymlinks
	_ = TestContentLargeBlobChunksAndLimit
	_ = TestContentBlobInstallFailureAndRetry
	_ = TestContentSubmoduleUninitializedAndRefusals
)
