package traceability

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
	"github.com/relux-works/agent-session-manager/internal/specpin"
)

func TestVerifyRepositoryAcceptsExactOwnership(t *testing.T) {
	t.Parallel()

	report, err := VerifyRepository(repositorySnapshot(t))
	if err != nil {
		t.Fatalf("VerifyRepository() error = %v", err)
	}
	want := Report{
		Contracts:              60,
		NormativeSections:      36,
		AcceptanceCases:        77,
		Fixtures:               30,
		CompatibilityContracts: 55,
		SectionBindings:        49,
		FullCoverage:           1,
		PartialCoverage:        3,
		SliverCoverage:         1,
		UnevidencedCoverage:    41,
		UnmeasuredCoverage:     3,
		UnownedSections:        2,
		NormativeClauses:       403,
		DischargedClauses:      17,
	}
	if !reflect.DeepEqual(report, want) {
		t.Fatalf("VerifyRepository() report = %#v, want %#v", report, want)
	}
}

// TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases pins
// the admitted arm of assigned-scope admission. Section 6.2 is admitted because
// it discharges the one normative clause its pinned section carries against an
// executable acceptance case. It is the only shipped binding that is.
//
// Section 13.14.5 used to be admitted here on the ground that its pinned
// section "carries no RFC 2119 obligation of its own". That was false: the
// obligation scanner matches uppercase keywords only, and Section 13.14.5
// states its obligations as required-member and variant tables. Nineteen of the
// 157 pinned headings are in that class, including the eighteen-row exit-code
// registry of Section 15.2 and the closed Provider Manifest of Section 7.3, and
// admitting them reproduced this bug through a different door. The assertion is
// kept rather than deleted: it moved to the refusal table below and to
// TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers, which now pins
// each of the three by its measured ratio.
func TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases(t *testing.T) {
	t.Parallel()

	report, err := VerifyAssignedSections(repositorySnapshot(t), []string{"6.2"})
	if err != nil {
		t.Fatalf("VerifyAssignedSections() error = %v", err)
	}
	if report.AssignedScopes != 1 {
		t.Fatalf("VerifyAssignedSections() assigned scopes = %d, want 1", report.AssignedScopes)
	}

	for _, scope := range []string{"6.2", "§6.2"} {
		report, err := VerifyAssignedSections(repositorySnapshot(t), []string{scope})
		if err != nil {
			t.Errorf("VerifyAssignedSections(%q) error = %v", scope, err)
			continue
		}
		if report.AssignedScopes != 1 {
			t.Errorf("VerifyAssignedSections(%q) assigned scopes = %d, want 1", scope, report.AssignedScopes)
		}
	}

	// A scope that pairs the one admitted binding with an unmeasured one is
	// refused as a whole. Admission is not per-section best-effort.
	_, err = VerifyAssignedSections(repositorySnapshot(t), []string{"6.2", "13.14.5"})
	if err == nil || !errors.Is(err, ErrTraceability) ||
		!strings.Contains(err.Error(), `binding "section:13.14.5" discharges 0/0 normative clauses, which is unmeasured coverage`) {
		t.Fatalf("VerifyAssignedSections(6.2, 13.14.5) error = %v, want unmeasured refusal", err)
	}

	for _, scope := range []string{"11.2-11.3", "§4.B-4.C", "Appendix D"} {
		_, err := VerifyAssignedSections(repositorySnapshot(t), []string{scope})
		if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), "assigned-scope admission requires full") {
			t.Errorf("VerifyAssignedSections(%q) error = %v, want coverage refusal", scope, err)
		}
	}
}

func TestVerifyAssignedSectionsRejectsPinnedSectionWithoutScopedImplementation(t *testing.T) {
	t.Parallel()

	_, err := VerifyAssignedSections(repositorySnapshot(t), []string{"10.5"})
	want := `assigned section "10.5" binding "section:10.5" has no scoped implementation owner`
	if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), want) {
		t.Fatalf("VerifyAssignedSections(10.5) error = %v, want ErrTraceability containing %q", err, want)
	}
}

func TestVerifyAssignedSectionsRejectsMalformedUnpinnedAndEmptyScope(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		sections []string
		contains string
	}{
		{name: "empty", sections: nil, contains: "assigned section scope is empty"},
		{name: "malformed", sections: []string{"10.x!"}, contains: "invalid assigned section"},
		{name: "nonexistent", sections: []string{"10.999"}, contains: "not a real v0.5.0 section identifier"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := VerifyAssignedSections(repositorySnapshot(t), test.sections)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("VerifyAssignedSections() error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
	}
}

func TestResolveAssignedSectionsRejectsRealButUnpinnedSection(t *testing.T) {
	t.Parallel()

	_, err := resolveAssignedSections([]string{"1"}, []string{"10.1"})
	if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), "outside pinned normative scope") {
		t.Fatalf("resolveAssignedSections() error = %v, want unpinned-scope refusal", err)
	}
}

func TestVerifyRepositoryRejectsNarrowedOwnership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*ownershipRegistry)
		contains string
	}{
		{
			name: "one registered contract loses ownership",
			mutate: func(registry *ownershipRegistry) {
				group := ownershipGroupByKind(t, registry, ownershipContract)
				group.Keys = group.Keys[:len(group.Keys)-1]
			},
			contains: "registered contract",
		},
		{
			name: "one nonempty normative section loses ownership",
			mutate: func(registry *ownershipRegistry) {
				group := ownershipGroupByKind(t, registry, ownershipNormativeSection)
				group.Keys = group.Keys[1:]
			},
			contains: "registered normative_section",
		},
		{
			name: "one exact catalog section binding loses ownership",
			mutate: func(registry *ownershipRegistry) {
				removeOwnershipGroupByKey(t, registry, "section:9.2")
			},
			contains: "registered section_binding \"section:9.2\"",
		},
		{
			name: "one exact fixture loses ownership",
			mutate: func(registry *ownershipRegistry) {
				group := ownershipGroupByKind(t, registry, ownershipFixture)
				group.Keys = group.Keys[1:]
			},
			contains: "registered fixture",
		},
		{
			name: "one acceptance case loses its owner",
			mutate: func(registry *ownershipRegistry) {
				registry.AcceptanceCases = registry.AcceptanceCases[1:]
			},
			contains: "unregistered acceptance case",
		},
		{
			name: "one covered contract is duplicated instead of deleted",
			mutate: func(registry *ownershipRegistry) {
				group := ownershipGroupByKind(t, registry, ownershipContract)
				group.Keys = append(group.Keys, group.Keys[0])
			},
			contains: "duplicate contract implementation owner",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := repositorySnapshot(t)

			// Forward control. A mutant that "reddens" an already-red mask
			// proves nothing, so the unmutated snapshot has to be green at both
			// entry points before the mutant is applied.
			if _, err := VerifyRepository(repository); err != nil {
				t.Fatalf("baseline VerifyRepository() error = %v, want green mask before planting", err)
			}
			if _, err := VerifyAssignedSections(repository, []string{"6.2"}); err != nil {
				t.Fatalf("baseline VerifyAssignedSections(6.2) error = %v, want green mask before planting", err)
			}

			rewriteRegistry(t, repository, test.mutate)
			_, err := VerifyRepository(repository)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("VerifyRepository() error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
	}
}

func TestCatalogSectionBindingCoverageIsExactAndDoesNotClaimUnimplementedScope(t *testing.T) {
	t.Parallel()

	current, err := catalog.ForRelease(catalog.ReleaseV050)
	if err != nil {
		t.Fatalf("ForRelease(v0.5.0) error = %v", err)
	}
	bindings, err := expectedCatalogSectionBindings(current)
	if err != nil {
		t.Fatalf("expectedCatalogSectionBindings() error = %v", err)
	}
	if len(bindings) != 24 {
		t.Fatalf("catalog-scoped binding inventory has %d entries, want 24", len(bindings))
	}
	for _, required := range []string{"section:4.B", "section:9.2", "section:18.1", "section:appendix-d"} {
		if _, ok := bindings[required]; !ok {
			t.Errorf("catalog-scoped binding inventory lacks %q", required)
		}
	}
	if _, claimed := bindings["section:10.1"]; claimed {
		t.Fatal("catalog-scoped binding inventory falsely claims unimplemented Section 10.1")
	}
}

func TestVerifyRepositoryRejectsAbsentAcceptanceTestDeclaration(t *testing.T) {
	t.Parallel()

	repository := repositorySnapshot(t)
	replaceRepositorySourceOnce(
		t,
		repository,
		"internal/traceability/cmd/tracecheck/main_test.go",
		"func TestRunReportsExactCoverageAndFailsClosed(",
		"func TestRunReportsExactCoverageAndFailsClosedWithoutOwnership(",
	)

	_, err := VerifyRepository(repository)
	want := "acceptance case \"ci-entrypoint\" test owner: declaration \"TestRunReportsExactCoverageAndFailsClosed\" is absent from \"internal/traceability/cmd/tracecheck/main_test.go\""
	if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), want) {
		t.Fatalf("VerifyRepository() error = %v, want ErrTraceability containing %q", err, want)
	}
}

func TestSourceCheckerAcceptsNativeFuzzAndRejectsNonExecutableHelpers(t *testing.T) {
	t.Parallel()

	repository := fstest.MapFS{
		"boundary_test.go": &fstest.MapFile{Data: []byte(`package boundary

import "testing"

func FuzzBoundary(f *testing.F) {}
func Fuzzhelper(f *testing.F) {}
func BenchmarkBoundary(b *testing.B) {}
`)},
	}
	checker := newSourceChecker(repository)
	if err := checker.verify(codeReference{Path: "boundary_test.go", Declaration: "FuzzBoundary"}, true); err != nil {
		t.Fatalf("native fuzz reference refused: %v", err)
	}
	for _, declaration := range []string{"Fuzzhelper", "BenchmarkBoundary"} {
		err := checker.verify(codeReference{Path: "boundary_test.go", Declaration: declaration}, true)
		if err == nil || !strings.Contains(err.Error(), "not an executable Go test reference") {
			t.Errorf("sourceChecker.verify(%s) error = %v, want executable-test refusal", declaration, err)
		}
	}
}

func TestVerifyRepositoryRejectsAbsentAcceptanceProductionDeclaration(t *testing.T) {
	t.Parallel()

	repository := repositorySnapshot(t)
	replaceRepositorySourceOnce(
		t,
		repository,
		"internal/catalog/cmd/cataloggen/main.go",
		"func writeIfChanged(",
		"func writeIfChangedWithoutOwnership(",
	)

	_, err := VerifyRepository(repository)
	want := "acceptance case \"catalog-generation-idempotency-recovery\" production owner: declaration \"writeIfChanged\" is absent from \"internal/catalog/cmd/cataloggen/main.go\""
	if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), want) {
		t.Fatalf("VerifyRepository() error = %v, want ErrTraceability containing %q", err, want)
	}
}

func TestVerifyRepositoryRejectsAbsentOwnershipGroupProductionDeclaration(t *testing.T) {
	t.Parallel()

	repository := repositorySnapshot(t)
	replaceRepositorySourceOnce(
		t,
		repository,
		"internal/specpin/pin.go",
		"func (manifest Manifest) Fixture(",
		"func (manifest Manifest) FixtureWithoutOwnership(",
	)

	_, err := VerifyRepository(repository)
	want := "(fixture) production owner: declaration \"Fixture\" is absent from \"internal/specpin/pin.go\""
	if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), want) {
		t.Fatalf("VerifyRepository() error = %v, want ErrTraceability containing %q", err, want)
	}
}

func TestVerifyRepositoryDistinguishesMissingAndMalformedEvidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(fstest.MapFS)
		contains string
	}{
		{
			name: "ownership registry absent",
			mutate: func(repository fstest.MapFS) {
				delete(repository, ownershipRegistryPath)
			},
			contains: "read required evidence",
		},
		{
			name: "ownership registry malformed",
			mutate: func(repository fstest.MapFS) {
				repository[ownershipRegistryPath] = &fstest.MapFile{Data: []byte("{")}
			},
			contains: "decode ownership registry",
		},
		{
			name: "production owner source absent",
			mutate: func(repository fstest.MapFS) {
				delete(repository, "internal/catalog/catalog.go")
			},
			contains: "read \"internal/catalog/catalog.go\"",
		},
		{
			name: "normative lock partial read",
			mutate: func(repository fstest.MapFS) {
				value := repository[contractLockPath].Data
				repository[contractLockPath] = &fstest.MapFile{Data: bytes.Clone(value[:len(value)/2])}
			},
			contains: "verify normative source lock",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := repositorySnapshot(t)
			test.mutate(repository)
			_, err := VerifyRepository(repository)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("VerifyRepository() error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
	}
}

func TestVerifyRepositoryRejectsForgedOwnershipAndCapabilityClaims(t *testing.T) {
	t.Parallel()

	t.Run("valid declaration cannot self-mint ownership", func(t *testing.T) {
		t.Parallel()
		repository := repositorySnapshot(t)
		rewriteRegistry(t, repository, func(registry *ownershipRegistry) {
			group := ownershipGroupByKind(t, registry, ownershipContract)
			group.Production.Declaration = "Current"
		})
		_, err := VerifyRepository(repository)
		if err == nil || !strings.Contains(err.Error(), "projection digest") {
			t.Fatalf("VerifyRepository(forged owner) error = %v, want reviewed-digest refusal", err)
		}
	})

	t.Run("unknown contract owner is refused", func(t *testing.T) {
		t.Parallel()
		repository := repositorySnapshot(t)
		rewriteRegistry(t, repository, func(registry *ownershipRegistry) {
			group := ownershipGroupByKind(t, registry, ownershipContract)
			group.Keys = append(group.Keys, "Forged [urn:ax:schema:forged]")
		})
		_, err := VerifyRepository(repository)
		if err == nil || !strings.Contains(err.Error(), "self-minted") {
			t.Fatalf("VerifyRepository(forged contract) error = %v, want self-minted refusal", err)
		}
	})

	t.Run("capability availability claim is not a registry field", func(t *testing.T) {
		t.Parallel()
		repository := repositorySnapshot(t)
		var document map[string]any
		if err := json.Unmarshal(repository[ownershipRegistryPath].Data, &document); err != nil {
			t.Fatalf("decode ownership registry: %v", err)
		}
		group := document["ownership"].([]any)[0].(map[string]any)
		group["supported"] = true
		candidate, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("encode forged registry: %v", err)
		}
		repository[ownershipRegistryPath] = &fstest.MapFile{Data: candidate}
		_, err = VerifyRepository(repository)
		if err == nil || !strings.Contains(err.Error(), "unknown field \"supported\"") {
			t.Fatalf("VerifyRepository(capability claim) error = %v, want unknown-field refusal", err)
		}
	})
}

func TestVerifyRepositoryIsIdempotentAndReadOnly(t *testing.T) {
	t.Parallel()

	repository := repositorySnapshot(t)
	before := snapshotDigest(repository)
	first, err := VerifyRepository(repository)
	if err != nil {
		t.Fatalf("first VerifyRepository() error = %v", err)
	}
	second, err := VerifyRepository(repository)
	if err != nil {
		t.Fatalf("second VerifyRepository() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical verification reports differ: first=%#v second=%#v", first, second)
	}
	after := snapshotDigest(repository)
	if before != after {
		t.Fatalf("read-only verification mutated repository evidence: before=%x after=%x", before, after)
	}
}

func TestCIWorkflowInvokesTraceabilityGate(t *testing.T) {
	t.Parallel()

	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	for _, command := range []string{
		"go run ./internal/traceability/cmd/tracecheck",
		"go test ./... -v",
		"go vet ./...",
		"go build ./...",
	} {
		if !bytes.Contains(workflow, []byte(command)) {
			t.Errorf("CI workflow does not invoke %q", command)
		}
	}
}

func repositorySnapshot(t *testing.T) fstest.MapFS {
	t.Helper()

	root := os.DirFS(filepath.Join("..", ".."))
	registryBytes, err := fs.ReadFile(root, ownershipRegistryPath)
	if err != nil {
		t.Fatalf("read ownership registry: %v", err)
	}
	registry, err := decodeOwnershipRegistry(registryBytes)
	if err != nil {
		t.Fatalf("decode ownership registry: %v", err)
	}
	required := map[string]struct{}{
		contractLockPath:      {},
		catalogMetadataPath:   {},
		generatedCatalogPath:  {},
		ownershipRegistryPath: {},
	}
	for _, acceptance := range registry.AcceptanceCases {
		required[acceptance.Production.Path] = struct{}{}
		for _, test := range acceptance.Tests {
			required[test.Path] = struct{}{}
		}
	}
	for _, group := range registry.Ownership {
		required[group.Production.Path] = struct{}{}
	}

	result := make(fstest.MapFS, len(required))
	for filename := range required {
		value, err := fs.ReadFile(root, filename)
		if err != nil {
			t.Fatalf("read repository fixture %q: %v", filename, err)
		}
		result[filename] = &fstest.MapFile{Data: bytes.Clone(value)}
	}
	return result
}

func rewriteRegistry(t *testing.T, repository fstest.MapFS, mutate func(*ownershipRegistry)) {
	t.Helper()

	registry, err := decodeOwnershipRegistry(repository[ownershipRegistryPath].Data)
	if err != nil {
		t.Fatalf("decode ownership registry: %v", err)
	}
	mutate(&registry)
	value, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("encode ownership registry: %v", err)
	}
	repository[ownershipRegistryPath] = &fstest.MapFile{Data: value}
}

func replaceRepositorySourceOnce(t *testing.T, repository fstest.MapFS, filename, from, to string) {
	t.Helper()

	file, ok := repository[filename]
	if !ok {
		t.Fatalf("repository fixture %q not found", filename)
	}
	if count := bytes.Count(file.Data, []byte(from)); count != 1 {
		t.Fatalf("repository fixture %q contains source declaration %q %d times, want exactly once", filename, from, count)
	}
	repository[filename] = &fstest.MapFile{Data: bytes.Replace(file.Data, []byte(from), []byte(to), 1)}
}

func ownershipGroupByKind(t *testing.T, registry *ownershipRegistry, kind ownershipKind) *ownershipGroup {
	t.Helper()

	for index := range registry.Ownership {
		if registry.Ownership[index].Kind == kind {
			return &registry.Ownership[index]
		}
	}
	t.Fatalf("ownership group %q not found", kind)
	return nil
}

func removeOwnershipGroupByKey(t *testing.T, registry *ownershipRegistry, key string) {
	t.Helper()
	for index, group := range registry.Ownership {
		for _, candidate := range group.Keys {
			if candidate == key {
				registry.Ownership = append(registry.Ownership[:index], registry.Ownership[index+1:]...)
				return
			}
		}
	}
	t.Fatalf("ownership key %q not found", key)
}

func snapshotDigest(repository fstest.MapFS) [32]byte {
	names := make([]string, 0, len(repository))
	for name := range repository {
		names = append(names, name)
	}
	sort.Strings(names)
	var content bytes.Buffer
	for _, name := range names {
		content.WriteString(name)
		content.WriteByte(0)
		content.Write(repository[name].Data)
		content.WriteByte(0)
	}
	return sha256.Sum256(content.Bytes())
}

// TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers is the negative
// arm of the coverage gate at the production entry point. Every section listed
// here used to be admitted by VerifyAssignedSections because its binding named
// a real Go declaration and an executable acceptance case. None of them
// discharges its section, so a Story assigned one of them could legitimately do
// nothing and still leave the gate green. The refusal now names the measured
// ratio, so this table is a disclosure of the shipped state and not prose: a
// section that later becomes fully covered has to be removed from it
// deliberately.
//
// The three 0/0 rows - 7.3, 13.14.5 and 15.2 - are the sections the obligation
// scanner cannot see. They were admitted by the first revision of this gate on
// the false ground that they carry no obligation; they are refused here because
// a section the gate cannot measure is not a section the gate may certify.
func TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers(t *testing.T) {
	t.Parallel()

	repository := repositorySnapshot(t)
	for _, test := range []struct {
		section string
		want    string
	}{
		{"1.6", `binding "section:1.6" discharges 0/31 normative clauses, which is unevidenced coverage`},
		{"2.1", `binding "section:2.1" discharges 0/1 normative clauses, which is unevidenced coverage`},
		{"2.2", `binding "section:2.2" is recorded unowned:`},
		{"2.3", `binding "section:2.3" discharges 0/7 normative clauses, which is unevidenced coverage`},
		{"2.4", `binding "section:2.4" discharges 0/4 normative clauses, which is unevidenced coverage`},
		{"3.2", `binding "section:3.2" discharges 0/13 normative clauses, which is unevidenced coverage`},
		{"3.3", `binding "section:3.3" discharges 0/4 normative clauses, which is unevidenced coverage`},
		{"5.1", `binding "section:5.1" discharges 0/9 normative clauses, which is unevidenced coverage`},
		{"6.1", `binding "section:6.1" discharges 0/2 normative clauses, which is unevidenced coverage`},
		{"6.3", `binding "section:6.3" discharges 0/11 normative clauses, which is unevidenced coverage`},
		{"6.4", `binding "section:6.4" discharges 0/2 normative clauses, which is unevidenced coverage`},
		{"6.5", `binding "section:6.5" discharges 0/3 normative clauses, which is unevidenced coverage`},
		{"7.3", `binding "section:7.3" discharges 0/0 normative clauses, which is unmeasured coverage`},
		{"7.9", `binding "section:7.9" discharges 0/8 normative clauses, which is unevidenced coverage`},
		{"9.2", `binding "section:9.2" discharges 0/35 normative clauses, which is unevidenced coverage`},
		{"10.1", `binding "section:10.1" discharges 0/3 normative clauses, which is unevidenced coverage`},
		{"10.2", `binding "section:10.2" discharges 0/5 normative clauses, which is unevidenced coverage`},
		{"10.3", `binding "section:10.3" discharges 1/3 normative clauses, which is sliver coverage`},
		{"10.4", `binding "section:10.4" discharges 0/25 normative clauses, which is unevidenced coverage`},
		{"13.14.5", `binding "section:13.14.5" discharges 0/0 normative clauses, which is unmeasured coverage`},
		{"14.2", `binding "section:14.2" discharges 8/9 normative clauses, which is partial coverage`},
		{"15.1", `binding "section:15.1" discharges 5/7 normative clauses, which is partial coverage`},
		{"15.2", `binding "section:15.2" discharges 0/0 normative clauses, which is unmeasured coverage`},
		{"15.3", `binding "section:15.3" discharges 2/3 normative clauses, which is partial coverage`},
		{"17.1", `binding "section:17.1" discharges 0/6 normative clauses, which is unevidenced coverage`},
		{"17.2", `binding "section:17.2" discharges 0/1 normative clauses, which is unevidenced coverage`},
		{"17.3", `binding "section:17.3" discharges 0/3 normative clauses, which is unevidenced coverage`},
		{"18.1", `binding "section:18.1" discharges 0/5 normative clauses, which is unevidenced coverage`},
		{"18.4", `binding "section:18.4" is recorded unowned:`},
		{"Appendix D", `binding "section:appendix-d" discharges 0/16 normative clauses, which is unevidenced coverage`},
	} {
		test := test
		t.Run(test.section, func(t *testing.T) {
			t.Parallel()
			_, err := VerifyAssignedSections(repository, []string{test.section})
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("VerifyAssignedSections(%q) error = %v, want ErrTraceability containing %q", test.section, err, test.want)
			}
		})
	}
}

// TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted is the
// anti-vacuity proof for the coverage gate. A gate that only ever reddens
// proves nothing, and a gate that only reddens on a deleted binding says
// nothing about the class it is supposed to cover. Both arms drive the same
// production function verifySectionBindingCoverage, which verifyOwnership calls
// through verifySectionCoverage on both VerifyRepository and
// VerifyAssignedSections.
func TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load() error = %v", err)
	}
	acceptanceIDs := map[string]struct{}{"config-versioned-readers": {}, "canonical-identity-refusal": {}}

	// Section 6.3 carries eleven normative clauses. The adequate binding
	// enumerates all eleven from the pinned document itself.
	inventory, err := sectionClauseInventory(document, "section:6.3")
	if err != nil {
		t.Fatalf("sectionClauseInventory() error = %v", err)
	}
	if len(inventory) != 11 {
		t.Fatalf("Section 6.3 clause inventory = %d, want 11", len(inventory))
	}
	adequate := ownershipGroup{
		Kind:            ownershipSectionBinding,
		Keys:            []string{"section:6.3"},
		Production:      codeReference{Path: "internal/config/validation.go", Declaration: "validateConfiguration"},
		AcceptanceCases: []string{"config-versioned-readers"},
		Coverage:        coverageFull,
	}
	for _, clause := range inventory {
		adequate.Clauses = append(adequate.Clauses, dischargedClause{
			ID:              clause.ID,
			Line:            clause.Line,
			Excerpt:         clause.Text,
			AcceptanceCases: []string{"config-versioned-readers"},
		})
	}
	measured, err := verifySectionBindingCoverage(document, adequate, acceptanceIDs)
	if err != nil {
		t.Fatalf("adequate binding refused: %v", err)
	}
	if measured.Level != coverageFull || measured.Ratio() != "11/11" {
		t.Fatalf("adequate binding measured %s %s, want full 11/11", measured.Level, measured.Ratio())
	}

	// Every mutant below keeps the "full" claim and breaks exactly one thing.
	for _, test := range []struct {
		name     string
		mutate   func(*ownershipGroup)
		contains string
	}{
		{
			name:     "planted sliver keeps the full claim with one clause",
			mutate:   func(group *ownershipGroup) { group.Clauses = group.Clauses[:1] },
			contains: `claims full coverage but discharges 1 of the 11 normative clauses`,
		},
		{
			name:     "planted sliver keeps the full claim with no clause",
			mutate:   func(group *ownershipGroup) { group.Clauses = nil },
			contains: `claims full coverage but discharges 0 of the 11 normative clauses`,
		},
		{
			name:     "planted partial keeps the full claim",
			mutate:   func(group *ownershipGroup) { group.Clauses = group.Clauses[:6] },
			contains: `claims full coverage but discharges 6 of the 11 normative clauses of the pinned section, which is partial coverage`,
		},
		{
			name: "invented clause pads the enumeration",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].ID = "6.3#12"
			},
			contains: `discharges clause "6.3#12", which is not one of the 11 normative clauses`,
		},
		{
			name: "clause quotes text the pinned section does not carry",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].Excerpt = "The implementation MUST do whatever this registry says."
			},
			contains: `does not quote the pinned document verbatim at line`,
		},
		{
			name: "clause quotes a real obligation from another section",
			mutate: func(group *ownershipGroup) {
				foreign, err := sectionClauseInventory(document, "section:10.3")
				if err != nil {
					t.Fatalf("sectionClauseInventory(10.3) error = %v", err)
				}
				group.Clauses[0].Excerpt = foreign[0].Text
			},
			contains: `does not quote the pinned document verbatim at line`,
		},
		{
			name: "clause quotes prose that carries no obligation",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].Excerpt = "The transfer unit is a fixed 4 MiB chunk except the last chunk."
			},
			contains: `quotes text that carries no RFC 2119 obligation`,
		},
		{
			name: "clause moves to a line it does not occupy",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].Line = group.Clauses[1].Line
			},
			contains: `but the pinned clause is at line`,
		},
		{
			name: "clause discharges through no acceptance case",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].AcceptanceCases = nil
			},
			contains: `names no acceptance case that discharges it`,
		},
		{
			name: "clause discharges through an unregistered acceptance case",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].AcceptanceCases = []string{"invented-case"}
			},
			contains: `references unregistered acceptance case "invented-case"`,
		},
		{
			name: "clause borrows an acceptance case the binding does not own",
			mutate: func(group *ownershipGroup) {
				group.Clauses[0].AcceptanceCases = []string{"canonical-identity-refusal"}
			},
			contains: `which the binding does not own`,
		},
		{
			name: "the same clause is counted twice",
			mutate: func(group *ownershipGroup) {
				group.Clauses[1] = group.Clauses[0]
			},
			contains: `repeats discharged clause`,
		},
		{
			name: "coverage level is omitted entirely",
			mutate: func(group *ownershipGroup) {
				group.Coverage = ""
			},
			contains: `declares no coverage level`,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mutant := cloneOwnershipGroup(adequate)
			test.mutate(&mutant)
			_, err := verifySectionBindingCoverage(document, mutant, acceptanceIDs)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("verifySectionBindingCoverage(mutant) error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
	}
}

// TestPlantedSliverRedensTheProductionEntryPoints plants a coverage lie into a
// shipped binding and drives both production entry points with it. The declared
// level is the only thing changed, so the mutant proves the gate measures the
// enumeration rather than believing the label.
func TestPlantedSliverRedensTheProductionEntryPoints(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		mutate   func(*ownershipRegistry)
		contains string
	}{
		{
			name: "a sliver binding claims the whole section",
			mutate: func(registry *ownershipRegistry) {
				sectionBinding(t, registry, "section:10.3").Coverage = coverageFull
			},
			contains: `section binding "section:10.3" claims full coverage but discharges 1 of the 3 normative clauses of the pinned section, which is sliver coverage`,
		},
		{
			name: "an unevidenced binding claims the whole section",
			mutate: func(registry *ownershipRegistry) {
				group := sectionBinding(t, registry, "section:9.2")
				group.Coverage = coverageFull
				group.Gap = ""
			},
			contains: `section binding "section:9.2" claims full coverage but discharges 0 of the 35 normative clauses`,
		},
		{
			name: "a sliver binding drops the gap it has to name",
			mutate: func(registry *ownershipRegistry) {
				sectionBinding(t, registry, "section:10.3").Gap = "n/a"
			},
			contains: `section binding "section:10.3" claims sliver coverage without naming what 10.3 leaves unimplemented`,
		},
		{
			name: "an unmeasured binding drops the gap it now has to name",
			mutate: func(registry *ownershipRegistry) {
				sectionBinding(t, registry, "section:15.2").Gap = ""
			},
			contains: `section binding "section:15.2" claims unmeasured coverage without naming what 15.2 leaves unimplemented`,
		},
		{
			name: "an unmeasured binding pads its gap instead of disclosing",
			mutate: func(registry *ownershipRegistry) {
				sectionBinding(t, registry, "section:15.2").Gap =
					"Section 15.2 is not fully covered here yet, and it will be covered later."
			},
			contains: `gap does not name the production declaration "ExitCodeFor" the binding is registered to`,
		},
		{
			name: "an unmeasured binding relabels itself unevidenced",
			mutate: func(registry *ownershipRegistry) {
				sectionBinding(t, registry, "section:7.3").Coverage = coverageUnevidenced
			},
			contains: `section binding "section:7.3" claims unevidenced coverage but discharges 0 of the 0 normative clauses of the pinned section, which is unmeasured coverage`,
		},
		{
			name: "an unmeasured binding relabels itself full to reach admission",
			mutate: func(registry *ownershipRegistry) {
				group := sectionBinding(t, registry, "section:13.14.5")
				group.Coverage = coverageFull
				group.Gap = ""
			},
			contains: `section binding "section:13.14.5" claims full coverage but discharges 0 of the 0 normative clauses of the pinned section, which is unmeasured coverage`,
		},
		{
			name: "a gap names a neighbouring section identifier instead of its own",
			mutate: func(registry *ownershipRegistry) {
				sectionBinding(t, registry, "section:6.5").Gap =
					"translateV3 maps the legacy terminal table, but Section 6.55 is not implemented in this repository at all."
			},
			contains: `gap does not name section 6.5 as a whole identifier`,
		},
		{
			name: "an unowned entry pads its gap with a neighbouring identifier",
			mutate: func(registry *ownershipRegistry) {
				registry.UnownedSections[0].Gap =
					"Section 2.22 lease and replica invariants are not implemented in this repository at all."
			},
			contains: `unowned section "section:2.2" does not name what 2.2 leaves unimplemented: gap does not name section 2.2 as a whole identifier`,
		},
		{
			name: "a contract owner claims section coverage",
			mutate: func(registry *ownershipRegistry) {
				ownershipGroupByKind(t, registry, ownershipContract).Coverage = coverageFull
			},
			contains: "claims section coverage, which only a section binding can be measured for",
		},
		{
			name: "an unowned section is also claimed as owned",
			mutate: func(registry *ownershipRegistry) {
				registry.UnownedSections = append(registry.UnownedSections, unownedSection{
					Key:      "section:10.3",
					Gap:      "Section 10.3 chunk validation is unimplemented in this repository.",
					Evidence: "There is no transfer receiver anywhere in the repository tree.",
				})
			},
			contains: `section "section:10.3" is registered as both owned and unowned`,
		},
		{
			name: "an unowned entry covers a section the catalog requires an owner for",
			mutate: func(registry *ownershipRegistry) {
				removeOwnershipGroupByKey(t, registry, "section:9.2")
				registry.UnownedSections = append(registry.UnownedSections, unownedSection{
					Key:      "section:9.2",
					Gap:      "Section 9.2 provider reality is unimplemented in this repository.",
					Evidence: "The typed catalog references it but no production declaration implements it.",
				})
			},
			contains: `unowned section "section:9.2" is required to have an implementation owner by the generated catalog`,
		},
		{
			name: "an unowned entry discloses no gap",
			mutate: func(registry *ownershipRegistry) {
				registry.UnownedSections[0].Gap = "todo"
			},
			contains: `unowned section "section:2.2" does not name what 2.2 leaves unimplemented`,
		},
		{
			name: "an unowned entry discloses no evidence",
			mutate: func(registry *ownershipRegistry) {
				registry.UnownedSections[0].Evidence = ""
			},
			contains: `unowned section "section:2.2" states no evidence for its gap`,
		},
		{
			name: "an unowned entry is self-minted for a section that does not exist",
			mutate: func(registry *ownershipRegistry) {
				registry.UnownedSections[0].Key = "section:10.999"
			},
			contains: `unowned section "section:10.999" is self-minted for an unknown section`,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := repositorySnapshot(t)
			rewriteRegistry(t, repository, test.mutate)
			_, err := VerifyRepository(repository)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("VerifyRepository(mutant) error = %v, want ErrTraceability containing %q", err, test.contains)
			}
			_, err = VerifyAssignedSections(repository, []string{"6.2"})
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("VerifyAssignedSections(mutant) error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
	}
}

// TestSectionClauseInventoryIsMeasuredFromThePinnedDocument proves the
// denominator is read from the hash-verified specification rather than declared
// by the registry, and that a parent heading is measured over its subheadings
// rather than over the lines above its first child.
func TestSectionClauseInventoryIsMeasuredFromThePinnedDocument(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load() error = %v", err)
	}
	for _, test := range []struct {
		key   string
		total int
		first int
	}{
		{key: "section:6.2", total: 1, first: 2417},
		{key: "section:10.3", total: 3, first: 4663},
		{key: "section:2.2", total: 22, first: 392},
		{key: "section:appendix-d", total: 16},
		{key: "section:13.14.5", total: 0},
	} {
		inventory, err := sectionClauseInventory(document, test.key)
		if err != nil {
			t.Errorf("sectionClauseInventory(%q) error = %v", test.key, err)
			continue
		}
		if len(inventory) != test.total {
			t.Errorf("sectionClauseInventory(%q) = %d clauses, want %d", test.key, len(inventory), test.total)
			continue
		}
		if test.first != 0 && inventory[0].Line != test.first {
			t.Errorf("sectionClauseInventory(%q) first clause at line %d, want %d", test.key, inventory[0].Line, test.first)
		}
		for index, clause := range inventory {
			owner, ok := document.SectionID(clause.Line)
			if !ok {
				t.Errorf("sectionClauseInventory(%q) clause %d has no enclosing section", test.key, index)
				continue
			}
			identifier, err := sectionDocumentIdentifier(test.key)
			if err != nil {
				t.Fatalf("sectionDocumentIdentifier(%q) error = %v", test.key, err)
			}
			if owner != identifier && !strings.HasPrefix(owner, identifier+".") {
				t.Errorf("sectionClauseInventory(%q) imported clause from section %q", test.key, owner)
			}
		}
	}

	if _, err := sectionClauseInventory(document, "9.2"); err == nil {
		t.Fatal("sectionClauseInventory accepted a key without the section prefix")
	}
}

// TestCoverageBucketNamesTheMeasuredRatio pins the boundary between the levels,
// including the two that admit an assigned scope.
func TestCoverageBucketNamesTheMeasuredRatio(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		discharged int
		total      int
		want       coverageLevel
	}{
		{0, 0, coverageUnmeasured},
		{0, 1, coverageUnevidenced},
		{0, 22, coverageUnevidenced},
		{1, 1, coverageFull},
		{1, 3, coverageSliver},
		{1, 22, coverageSliver},
		{5, 11, coverageSliver},
		{6, 11, coveragePartial},
		{2, 3, coveragePartial},
		{11, 11, coverageFull},
	} {
		if got := coverageBucket(test.discharged, test.total); got != test.want {
			t.Errorf("coverageBucket(%d, %d) = %s, want %s", test.discharged, test.total, got, test.want)
		}
	}
}

func sectionBinding(t *testing.T, registry *ownershipRegistry, key string) *ownershipGroup {
	t.Helper()
	for index := range registry.Ownership {
		if registry.Ownership[index].Kind != ownershipSectionBinding {
			continue
		}
		if registry.Ownership[index].Keys[0] == key {
			return &registry.Ownership[index]
		}
	}
	t.Fatalf("section binding %q not found", key)
	return nil
}

func cloneOwnershipGroup(group ownershipGroup) ownershipGroup {
	cloned := group
	cloned.Keys = append([]string(nil), group.Keys...)
	cloned.AcceptanceCases = append([]string(nil), group.AcceptanceCases...)
	cloned.Clauses = make([]dischargedClause, len(group.Clauses))
	for index, clause := range group.Clauses {
		cloned.Clauses[index] = clause
		cloned.Clauses[index].AcceptanceCases = append([]string(nil), clause.AcceptanceCases...)
	}
	return cloned
}

// TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation is the
// measurement behind the coverageUnmeasured rename and behind refusing that
// bucket for assigned-scope admission.
//
// The first revision of this gate called the bucket "declarative" and admitted
// it, on the stated ground that such a section "carries no RFC 2119 clause of
// its own". This test measures whether that is true of the pinned document. It
// is not: every pinned heading the obligation scanner scores at zero still has
// a substantive body, so not one of them is a heading with nothing to
// discharge. The scanner is blind to them because they state their obligations
// as tables, not because the obligations are absent.
func TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load() error = %v", err)
	}
	identifiers := specpin.SectionInventoryV050()
	if len(identifiers) != 157 {
		t.Fatalf("pinned heading inventory = %d, want 157", len(identifiers))
	}
	var blind []string
	for _, identifier := range identifiers {
		key := sectionBindingKey(identifier)
		inventory, err := sectionClauseInventory(document, key)
		if err != nil {
			t.Fatalf("sectionClauseInventory(%q) error = %v", key, err)
		}
		if len(inventory) != 0 {
			continue
		}
		blind = append(blind, identifier)
		if body := sectionBodyLineCount(t, document, key); body < 8 {
			t.Errorf("heading %s scores zero clauses and has only %d body lines; it may genuinely carry nothing", identifier, body)
		}
	}
	want := []string{
		"7.3", "10.8.1", "13.5", "13.12", "13.14.1", "13.14.2", "13.14.3", "13.14.4", "13.14.5",
		"14.3", "14.6", "15.2", "16.6", "16.7", "18.2", "19.4", "appendix-a", "appendix-b", "appendix-c",
	}
	sort.Strings(blind)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if !reflect.DeepEqual(blind, sortedWant) {
		t.Fatalf("scanner-blind headings = %v, want %v", blind, sortedWant)
	}
	if len(blind) != 19 {
		t.Fatalf("scanner-blind headings = %d, want 19 of 157", len(blind))
	}
}

// sectionBodyLineCount counts the non-blank lines the pinned document carries
// under a heading and its subheadings.
func sectionBodyLineCount(t *testing.T, document *specdoc.Document, key string) int {
	t.Helper()
	identifier, err := sectionDocumentIdentifier(key)
	if err != nil {
		t.Fatalf("sectionDocumentIdentifier(%q) error = %v", key, err)
	}
	prefix := identifier + "."
	count := 0
	for line := 1; line <= document.LineCount(); line++ {
		owner, ok := document.SectionID(line)
		if !ok || (owner != identifier && !strings.HasPrefix(owner, prefix)) {
			continue
		}
		text, ok := document.Line(line)
		if !ok {
			t.Fatalf("pinned document line %d is unreadable", line)
		}
		if strings.TrimSpace(text) != "" {
			count++
		}
	}
	return count
}

// TestUnmeasuredBindingWithAnHonestGapIsStillAccepted is the anti-vacuity arm
// for the tightened unmeasured bucket. Requiring a gap must not make the bucket
// impossible to declare: a binding that discloses honestly is measured as
// unmeasured 0/0 and accepted by the coverage check. It is refused only at
// assigned-scope admission, which is a separate decision pinned by
// TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers.
func TestUnmeasuredBindingWithAnHonestGapIsStillAccepted(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load() error = %v", err)
	}
	acceptanceIDs := map[string]struct{}{"core-record-shapes": {}}
	honest := ownershipGroup{
		Kind:            ownershipSectionBinding,
		Keys:            []string{"section:13.14.5"},
		Production:      codeReference{Path: "internal/canonicaljson/core_records.go", Declaration: "validateSessionEventV2"},
		AcceptanceCases: []string{"core-record-shapes"},
		Coverage:        coverageUnmeasured,
		Gap: "validateSessionEventV2 validates the Section 13.14.5 v2 Session Event shape, but the " +
			"obligation scanner measures no clause line under 13.14.5 because the section states its " +
			"obligations as tables.",
	}
	measured, err := verifySectionBindingCoverage(document, honest, acceptanceIDs)
	if err != nil {
		t.Fatalf("honest unmeasured binding refused: %v", err)
	}
	if measured.Level != coverageUnmeasured || measured.Ratio() != "0/0" {
		t.Fatalf("honest unmeasured binding measured %s %s, want unmeasured 0/0", measured.Level, measured.Ratio())
	}

	// Every mutant keeps the same binding and breaks exactly one thing.
	for _, test := range []struct {
		name     string
		mutate   func(*ownershipGroup)
		contains string
	}{
		{
			name:     "the gap is dropped entirely",
			mutate:   func(group *ownershipGroup) { group.Gap = "" },
			contains: `claims unmeasured coverage without naming what 13.14.5 leaves unimplemented`,
		},
		{
			name:     "the gap is padded to the minimum length",
			mutate:   func(group *ownershipGroup) { group.Gap = "Section 13.14.5 is not fully covered here yet." },
			contains: `gap does not name the production declaration "validateSessionEventV2" the binding is registered to`,
		},
		{
			name: "the gap names a longer neighbouring identifier",
			mutate: func(group *ownershipGroup) {
				group.Gap = "validateSessionEventV2 validates a shape, but Section 13.14.55 is unimplemented in this tree."
			},
			contains: `gap does not name section 13.14.5 as a whole identifier`,
		},
		{
			name:     "the binding relabels itself declaratively green",
			mutate:   func(group *ownershipGroup) { group.Coverage = coverageFull; group.Gap = "" },
			contains: `claims full coverage but discharges 0 of the 0 normative clauses of the pinned section, which is unmeasured coverage`,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mutant := cloneOwnershipGroup(honest)
			test.mutate(&mutant)
			_, err := verifySectionBindingCoverage(document, mutant, acceptanceIDs)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("verifySectionBindingCoverage(mutant) error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
	}
}

// TestMentionsSectionRequiresAWholeIdentifier pins the boundary rule that stops
// a gap about one section from standing in for a gap about another. A bare
// substring match accepted "6.55" as a mention of "6.5".
func TestMentionsSectionRequiresAWholeIdentifier(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		text    string
		display string
		want    bool
	}{
		{"ForRelease covers Section 6.5 partially.", "6.5", true},
		{"ForRelease covers Section 6.5, partially.", "6.5", true},
		{"ForRelease covers Section 6.5.", "6.5", true},
		{"ForRelease covers Section 6.55.", "6.5", false},
		{"ForRelease covers Section 16.5.", "6.5", false},
		{"ForRelease covers Section 13.15 only.", "13.1", false},
		{"ForRelease covers Section 13.14.5 only.", "13.14.5", true},
		{"ForRelease covers Section 13.14.55 only.", "13.14.5", false},
		{"ForRelease references Appendix D rows.", "Appendix D", true},
		{"ForRelease references 4.B rows.", "4.B", true},
		{"ForRelease references 14.B rows.", "4.B", false},
		{"6.5", "6.5", true},
	} {
		if got := mentionsSection(test.text, test.display); got != test.want {
			t.Errorf("mentionsSection(%q, %q) = %v, want %v", test.text, test.display, got, test.want)
		}
	}
}
