package specpin_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

func TestCurrentPinsPublishedV050Source(t *testing.T) {
	manifest, err := specpin.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if manifest.Source.Repository != specpin.Repository {
		t.Errorf("repository = %q, want %q", manifest.Source.Repository, specpin.Repository)
	}
	if manifest.Source.TagObject != specpin.TagObjectV050 {
		t.Errorf("tag object = %q, want %q", manifest.Source.TagObject, specpin.TagObjectV050)
	}
	if manifest.Source.Commit != specpin.CommitV050 {
		t.Errorf("commit = %q, want %q", manifest.Source.Commit, specpin.CommitV050)
	}
	if manifest.Source.Document.SHA256 != specpin.DocumentSHA256 {
		t.Errorf("document digest = %q, want %q", manifest.Source.Document.SHA256, specpin.DocumentSHA256)
	}
	if manifest.Source.SectionInventorySHA256 != specpin.SectionInventorySHA256 {
		t.Errorf("section inventory digest = %q, want %q", manifest.Source.SectionInventorySHA256, specpin.SectionInventorySHA256)
	}
	wantScope := []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}
	if !reflect.DeepEqual(manifest.Source.NormativeScope, wantScope) {
		t.Errorf("normative scope = %v, want %v", manifest.Source.NormativeScope, wantScope)
	}
	if len(manifest.Contracts) != 60 {
		t.Fatalf("contract rows = %d, want 60", len(manifest.Contracts))
	}

	wantFixtures := map[string]string{
		specpin.SessionDirectoryFixtureID: "a6351a83e25a3a909297ed20bd1f4a75622b10f536a06b164fff3b12cb66f2ce",
		specpin.TerminalBackendFixtureID:  "67de0d78d76c9c445c742af5c4c14ffa5cecd620d4cb07dc5497d391b421ad37",
		specpin.RoadmapV043FixtureID:      "6023ec0d1562e8868b8bef3dc41cfd66ea0b4a4054fbaf13d3aec504578a7f74",
	}
	for id, wantDigest := range wantFixtures {
		fixture, ok := manifest.Fixture(id)
		if !ok {
			t.Errorf("Fixture(%q) was not found", id)
			continue
		}
		if fixture.SHA256 != wantDigest {
			t.Errorf("Fixture(%q) digest = %q, want %q", id, fixture.SHA256, wantDigest)
		}
	}
	if _, ok := manifest.Fixture("self-minted-fixture"); ok {
		t.Error("unknown fixture was accepted")
	}
}

func TestVerifyRejectsIdentityAndContractDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "tag",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["tag"] = "v0.5.1"
			},
		},
		{
			name: "commit",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["commit"] = strings.Repeat("0", 40)
			},
		},
		{
			name: "document digest",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["document"].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			},
		},
		{
			name: "normative scope",
			mutate: func(document map[string]any) {
				source := document["source"].(map[string]any)
				source["normative_scope"] = source["normative_scope"].([]any)[1:]
			},
		},
		{
			name: "section inventory digest",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["section_inventory_sha256"] = strings.Repeat("0", 64)
			},
		},
		{
			name: "contract version",
			mutate: func(document map[string]any) {
				document["contracts"].([]any)[0].(map[string]any)["versions"] = []any{"1.0.0", "2.0.0"}
			},
		},
		{
			name: "fixture identity",
			mutate: func(document map[string]any) {
				document["fixtures"].([]any)[0].(map[string]any)["id"] = "self-minted-fixture"
			},
		},
		{
			name: "unsupported capability claim",
			mutate: func(document map[string]any) {
				document["capabilities"] = []any{"terminal-backend"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := mutatePin(t, test.mutate)
			if _, err := specpin.Verify(candidate); !errors.Is(err, specpin.ErrPinMismatch) {
				t.Fatalf("Verify() error = %v, want ErrPinMismatch", err)
			}
		})
	}
}

func TestVerifyRejectsPartialMalformedAndByteDifferentReads(t *testing.T) {
	raw := specpin.Bytes()
	tests := []struct {
		name      string
		candidate []byte
	}{
		{name: "absent", candidate: nil},
		{name: "partial", candidate: raw[:len(raw)/2]},
		{name: "trailing value", candidate: append(append([]byte(nil), raw...), []byte("{}")...)},
		{name: "byte different whitespace", candidate: append(append([]byte(nil), raw...), '\n')},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := specpin.Verify(test.candidate); !errors.Is(err, specpin.ErrPinMismatch) {
				t.Fatalf("Verify() error = %v, want ErrPinMismatch", err)
			}
		})
	}
}

func TestContractsForReleasePreservesV043Compatibility(t *testing.T) {
	manifest, err := specpin.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	current, err := manifest.ContractsForRelease(specpin.ReleaseV050)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.5.0) error = %v", err)
	}
	if len(current) != 60 {
		t.Fatalf("v0.5.0 rows = %d, want 60", len(current))
	}

	historical, err := manifest.ContractsForRelease(specpin.ReleaseV043)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.4.3) error = %v", err)
	}
	if len(historical) != 55 {
		t.Fatalf("v0.4.3 rows = %d, want 55", len(historical))
	}

	wantVersions := map[string][]string{
		"Configuration":     {"1.0.0", "2.0.0"},
		"Provider protocol": {"2.0.0"},
		"Mesh RPC":          {"2.0.0", "3.0.0"},
		"Session event":     {"1.0.0", "2.0.0", "3.0.0"},
		"Structured error":  {"1.0.0", "1.1.0", "1.2.0"},
		"CLI result":        {"1.0.0", "2.0.0", "3.0.0"},
	}
	for _, contract := range historical {
		if strings.HasPrefix(contract.Name, "Terminal Backend") || contract.Name == "Terminal Instance binding" || contract.Name == "Terminal capability evidence" {
			t.Errorf("v0.4.3 retained v0.5.0-only contract %q", contract.Name)
		}
		if expected, ok := wantVersions[contract.Name]; ok && !reflect.DeepEqual(contract.Versions, expected) {
			t.Errorf("%s versions = %v, want %v", contract.Name, contract.Versions, expected)
		}
	}

	if _, err := manifest.ContractsForRelease("v0.4.2"); !errors.Is(err, specpin.ErrUnsupportedRelease) {
		t.Fatalf("ContractsForRelease(v0.4.2) error = %v, want ErrUnsupportedRelease", err)
	}
}

func TestCurrentIsIdempotentAndReturnsIsolatedData(t *testing.T) {
	first, err := specpin.Current()
	if err != nil {
		t.Fatalf("first Current() error = %v", err)
	}
	second, err := specpin.Current()
	if err != nil {
		t.Fatalf("second Current() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated Current() calls returned different manifests")
	}

	first.Contracts[0].Versions[0] = "9.9.9"
	third, err := specpin.Current()
	if err != nil {
		t.Fatalf("third Current() error = %v", err)
	}
	if third.Contracts[0].Versions[0] != "1.0.0" {
		t.Fatalf("caller mutation leaked into embedded pin: %v", third.Contracts[0].Versions)
	}
}

func mutatePin(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()

	var document map[string]any
	if err := json.Unmarshal(specpin.Bytes(), &document); err != nil {
		t.Fatalf("decode embedded pin: %v", err)
	}
	mutate(document)
	candidate, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mutated pin: %v", err)
	}
	return candidate
}
