package specpin_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

// TestCurrentV060PinsSignedV060Source drives the v0.6.0 production entry
// point and pins every adopted identity field to the independently verified
// signed release: annotated tag object
// 40c123eb8399efa8e05cbc009110940ed861a785 peeling to commit
// 0cbdf100dbf84df50c64f792b1f940e3a67859a6, SPEC.md 1150005 bytes with
// SHA-256 74504539fb43c28ae3450622bc1002e643f116cd3e14df4567a882231e90896b.
func TestCurrentV060PinsSignedV060Source(t *testing.T) {
	manifest, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("CurrentV060() error = %v", err)
	}

	source := manifest.Source
	if source.Repository != specpin.Repository {
		t.Errorf("repository = %q, want %q", source.Repository, specpin.Repository)
	}
	if source.Release != specpin.ReleaseV060 || source.Tag != specpin.TagV060 {
		t.Errorf("release/tag = %q/%q, want %q", source.Release, source.Tag, specpin.ReleaseV060)
	}
	if source.TagObject != specpin.TagObjectV060 {
		t.Errorf("tag object = %q, want %q", source.TagObject, specpin.TagObjectV060)
	}
	if source.Commit != specpin.CommitV060 {
		t.Errorf("commit = %q, want %q", source.Commit, specpin.CommitV060)
	}
	if source.Document.Path != specpin.DocumentPath || source.Document.SHA256 != specpin.DocumentSHA256V060 {
		t.Errorf("document = %+v, want path %q digest %q", source.Document, specpin.DocumentPath, specpin.DocumentSHA256V060)
	}
	if source.SectionInventorySHA256 != specpin.SectionInventorySHA256V060 {
		t.Errorf("section inventory digest = %q, want %q", source.SectionInventorySHA256, specpin.SectionInventorySHA256V060)
	}
	wantScope := []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}
	if !reflect.DeepEqual(source.NormativeScope, wantScope) {
		t.Errorf("normative scope = %v, want %v", source.NormativeScope, wantScope)
	}

	if len(manifest.Contracts) != 63 {
		t.Fatalf("contract rows = %d, want 63", len(manifest.Contracts))
	}
	// The prepared v0.6.0 delta: three new rows and four widened version
	// lists, each read from the production entry point, not retyped prose.
	wantDelta := map[string][]string{
		"Host Channel\x00urn:ax:transport:host-channel":        {"1.0.0"},
		"Host Trust Store\x00urn:ax:schema:host-trust-store":   {"1.0.0"},
		"Session selector\x00urn:ax:contract:session-selector": {"1.0.0"},
		"Configuration\x00urn:ax:schema:config":                {"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
		"Mesh RPC\x00urn:ax:protocol:rpc":                      {"2.0.0", "3.0.0", "4.0.0", "5.0.0"},
		"Structured error\x00urn:ax:schema:error":              {"1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0"},
		"CLI result\x00urn:ax:schema:cli-result":               {"1.0.0", "2.0.0", "3.0.0", "4.0.0", "5.0.0"},
	}
	seen := make(map[string]bool, len(wantDelta))
	for _, contract := range manifest.Contracts {
		key := contract.Name + "\x00" + contract.ID
		want, ok := wantDelta[key]
		if !ok {
			continue
		}
		seen[key] = true
		if !reflect.DeepEqual(contract.Versions, want) {
			t.Errorf("%s versions = %v, want %v", contract.Name, contract.Versions, want)
		}
	}
	for key := range wantDelta {
		if !seen[key] {
			t.Errorf("v0.6.0 delta row %q is absent from the verified registry", key)
		}
	}

	if manifest.Compatibility.BaselineRelease != specpin.ReleaseV043 {
		t.Errorf("compatibility baseline = %q, want %q", manifest.Compatibility.BaselineRelease, specpin.ReleaseV043)
	}
	if len(manifest.Compatibility.AbsentContracts) != 8 {
		t.Errorf("absent contracts = %v, want the five Terminal Backend rows plus the three v0.6.0 rows",
			manifest.Compatibility.AbsentContracts)
	}

	wantFixtures := map[string]string{
		specpin.SessionDirectoryFixtureID: "a6351a83e25a3a909297ed20bd1f4a75622b10f536a06b164fff3b12cb66f2ce",
		specpin.TerminalBackendFixtureID:  "67de0d78d76c9c445c742af5c4c14ffa5cecd620d4cb07dc5497d391b421ad37",
		specpin.RoadmapV043FixtureID:      "6023ec0d1562e8868b8bef3dc41cfd66ea0b4a4054fbaf13d3aec504578a7f74",
		specpin.HostChannelFixtureID:      "20aa66c363bcb660f7ed45ac4e760457e408fc030df1c784b71857f96961ae3b",
		specpin.SessionSelectorFixtureID:  "2bda47f5ad79911a3ce3db1ec7c8fbefbd2023fcc9f39cdc0eb22ee86b605770",
	}
	if len(manifest.Fixtures) != len(wantFixtures) {
		t.Fatalf("fixtures = %d rows, want %d", len(manifest.Fixtures), len(wantFixtures))
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

// TestVerifyV060RejectsWrongSourceHashAndVersionEvidence is the refusal half
// of adoption: every stale, substituted, trimmed, or invented identity the
// gate must reject is driven through the production VerifyV060 entry point.
// The stale-commit and stale-object rows preserve every v0.6.0 token and
// change only the pointed-to history, so a checker that matched tokens
// instead of binding them would admit them.
func TestVerifyV060RejectsWrongSourceHashAndVersionEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "invented release",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["release"] = "v0.6.1"
			},
		},
		{
			name: "stale tag",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["tag"] = "v0.5.0"
			},
		},
		{
			name: "stale tag object keeps v0.6.0 tokens",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["tag_object"] = specpin.TagObjectV050
			},
		},
		{
			name: "stale commit keeps v0.6.0 tokens",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["commit"] = specpin.CommitV050
			},
		},
		{
			name: "stale document digest",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["document"].(map[string]any)["sha256"] = specpin.DocumentSHA256
			},
		},
		{
			name: "stale section inventory digest",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["section_inventory_sha256"] = specpin.SectionInventorySHA256
			},
		},
		{
			name: "trimmed configuration delta",
			mutate: func(document map[string]any) {
				for _, row := range document["contracts"].([]any) {
					contract := row.(map[string]any)
					if contract["name"] == "Configuration" {
						contract["versions"] = []any{"1.0.0", "2.0.0", "3.0.0"}
					}
				}
			},
		},
		{
			name: "dropped session selector row",
			mutate: func(document map[string]any) {
				kept := document["contracts"].([]any)[:0]
				for _, row := range document["contracts"].([]any) {
					if row.(map[string]any)["name"] == "Session selector" {
						continue
					}
					kept = append(kept, row)
				}
				document["contracts"] = kept
			},
		},
		{
			name: "forged contract namespace",
			mutate: func(document map[string]any) {
				for _, row := range document["contracts"].([]any) {
					contract := row.(map[string]any)
					if contract["name"] == "Session selector" {
						contract["id"] = "urn:ax:capability:session-selector"
					}
				}
			},
		},
		{
			name: "substituted fixture digest",
			mutate: func(document map[string]any) {
				document["fixtures"].([]any)[3].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			},
		},
		{
			name: "unsupported capability claim",
			mutate: func(document map[string]any) {
				document["capabilities"] = []any{"host-channel"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := mutatePinV060(t, test.mutate)
			if _, err := specpin.VerifyV060(candidate); !errors.Is(err, specpin.ErrPinMismatch) {
				t.Fatalf("VerifyV060() error = %v, want ErrPinMismatch", err)
			}
		})
	}
}

// TestVerifyRefusesForeignReleaseBytes proves the two gates are
// version-bound: the v0.5.0 gate refuses the adopted v0.6.0 bytes (a stale
// consumer cannot be silently promoted), and the v0.6.0 gate refuses the
// historical v0.5.0 bytes (the new authority admits no old evidence).
func TestVerifyRefusesForeignReleaseBytes(t *testing.T) {
	if _, err := specpin.Verify(specpin.BytesV060()); !errors.Is(err, specpin.ErrPinMismatch) {
		t.Fatalf("Verify(v0.6.0 bytes) error = %v, want ErrPinMismatch", err)
	}
	if _, err := specpin.VerifyV060(specpin.Bytes()); !errors.Is(err, specpin.ErrPinMismatch) {
		t.Fatalf("VerifyV060(v0.5.0 bytes) error = %v, want ErrPinMismatch", err)
	}
}

func TestVerifyV060RejectsPartialMalformedAndByteDifferentReads(t *testing.T) {
	raw := specpin.BytesV060()
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
			if _, err := specpin.VerifyV060(test.candidate); !errors.Is(err, specpin.ErrPinMismatch) {
				t.Fatalf("VerifyV060() error = %v, want ErrPinMismatch", err)
			}
		})
	}
}

// TestContractsForReleaseV060 drives the production projection entry point
// on the verified v0.6.0 manifest. The historical projections are not
// restated: the v0.5.0 rows must equal the real v0.5.0 lock row for row, and
// the v0.4.3 rows must equal the v0.5.0 manifest's own v0.4.3 projection, so
// adoption cannot rewrite history without failing here.
func TestContractsForReleaseV060(t *testing.T) {
	manifest, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("CurrentV060() error = %v", err)
	}

	current, err := manifest.ContractsForRelease(specpin.ReleaseV060)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.6.0) error = %v", err)
	}
	if len(current) != 63 {
		t.Fatalf("v0.6.0 rows = %d, want 63", len(current))
	}

	historical050, err := manifest.ContractsForRelease(specpin.ReleaseV050)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.5.0) error = %v", err)
	}
	pinned050, err := specpin.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !reflect.DeepEqual(historical050, pinned050.Contracts) {
		t.Fatal("v0.6.0 manifest's v0.5.0 projection differs from the real v0.5.0 lock; history was rewritten")
	}

	historical043, err := manifest.ContractsForRelease(specpin.ReleaseV043)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.4.3) error = %v", err)
	}
	pinned043, err := pinned050.ContractsForRelease(specpin.ReleaseV043)
	if err != nil {
		t.Fatalf("v0.5.0 ContractsForRelease(v0.4.3) error = %v", err)
	}
	if !reflect.DeepEqual(historical043, pinned043) {
		t.Fatal("v0.6.0 manifest's v0.4.3 projection differs from the v0.5.0 manifest's; the baseline moved")
	}
	if len(historical043) != 55 {
		t.Fatalf("v0.4.3 rows = %d, want 55", len(historical043))
	}

	for _, release := range []string{"v0.4.2", "v0.6.1", "v0.6.0正式"} {
		if _, err := manifest.ContractsForRelease(release); !errors.Is(err, specpin.ErrUnsupportedRelease) {
			t.Errorf("ContractsForRelease(%q) error = %v, want ErrUnsupportedRelease", release, err)
		}
	}
}

// TestStalePinCannotAuthorizeNewRelease pins the direction of trust: a
// verified v0.5.0 manifest answers for v0.5.0 and v0.4.3 only. Asking it for
// v0.6.0 is refused, so old evidence can never stand in for the adopted
// release.
func TestStalePinCannotAuthorizeNewRelease(t *testing.T) {
	manifest, err := specpin.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if _, err := manifest.ContractsForRelease(specpin.ReleaseV060); !errors.Is(err, specpin.ErrUnsupportedRelease) {
		t.Fatalf("v0.5.0 ContractsForRelease(v0.6.0) error = %v, want ErrUnsupportedRelease", err)
	}
}

func TestCurrentV060IsIdempotentAndReturnsIsolatedData(t *testing.T) {
	first, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("first CurrentV060() error = %v", err)
	}
	second, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("second CurrentV060() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated CurrentV060() calls returned different manifests")
	}

	first.Contracts[0].Versions[0] = "9.9.9"
	third, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("third CurrentV060() error = %v", err)
	}
	if third.Contracts[0].Versions[0] != "1.0.0" {
		t.Fatalf("caller mutation leaked into embedded pin: %v", third.Contracts[0].Versions)
	}

	raw := specpin.BytesV060()
	raw[0] = ' '
	if _, err := specpin.CurrentV060(); err != nil {
		t.Fatalf("caller mutation of BytesV060 leaked into the embed: %v", err)
	}
}

func mutatePinV060(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()

	var document map[string]any
	if err := json.Unmarshal(specpin.BytesV060(), &document); err != nil {
		t.Fatalf("decode embedded v0.6.0 pin: %v", err)
	}
	mutate(document)
	candidate, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mutated pin: %v", err)
	}
	return candidate
}
