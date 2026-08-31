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
)

func TestVerifyRepositoryAcceptsExactOwnership(t *testing.T) {
	t.Parallel()

	report, err := VerifyRepository(repositorySnapshot(t))
	if err != nil {
		t.Fatalf("VerifyRepository() error = %v", err)
	}
	want := Report{
		Contracts:              60,
		NormativeSections:      17,
		AcceptanceCases:        15,
		Fixtures:               30,
		CompatibilityContracts: 55,
	}
	if !reflect.DeepEqual(report, want) {
		t.Fatalf("VerifyRepository() report = %#v, want %#v", report, want)
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
			rewriteRegistry(t, repository, test.mutate)
			_, err := VerifyRepository(repository)
			if err == nil || !errors.Is(err, ErrTraceability) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("VerifyRepository() error = %v, want ErrTraceability containing %q", err, test.contains)
			}
		})
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
	want := "ownership group 3 (fixture) production owner: declaration \"Fixture\" is absent from \"internal/specpin/pin.go\""
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
