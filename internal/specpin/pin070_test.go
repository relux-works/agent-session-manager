package specpin_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

// TestCurrentV070PinsSignedV070Source drives the v0.7.0 production entry
// point and pins every adopted identity field to the independently verified
// signed release: annotated tag object
// d4abe46fb12d9ba347c09f43efb01089530795b3 peeling to commit
// 32b3f2ba7c377248a53cd42389abbd2f1c321834, SPEC.md 1185291 bytes with
// SHA-256 c6b2fe64ee79ed697a96ed27a1679c80b8ee1eba137feb99e7738b4da289ddcf.
func TestCurrentV070PinsSignedV070Source(t *testing.T) {
	manifest, err := specpin.CurrentV070()
	if err != nil {
		t.Fatalf("CurrentV070() error = %v", err)
	}

	source := manifest.Source
	if source.Repository != specpin.Repository {
		t.Errorf("repository = %q, want %q", source.Repository, specpin.Repository)
	}
	if source.Release != specpin.ReleaseV070 || source.Tag != specpin.TagV070 {
		t.Errorf("release/tag = %q/%q, want %q", source.Release, source.Tag, specpin.ReleaseV070)
	}
	if source.TagObject != specpin.TagObjectV070 {
		t.Errorf("tag object = %q, want %q", source.TagObject, specpin.TagObjectV070)
	}
	if source.Commit != specpin.CommitV070 {
		t.Errorf("commit = %q, want %q", source.Commit, specpin.CommitV070)
	}
	if source.Document.Path != specpin.DocumentPath || source.Document.SHA256 != specpin.DocumentSHA256V070 {
		t.Errorf("document = %+v, want path %q digest %q", source.Document, specpin.DocumentPath, specpin.DocumentSHA256V070)
	}
	if source.SectionInventorySHA256 != specpin.SectionInventorySHA256V070 {
		t.Errorf("section inventory digest = %q, want %q", source.SectionInventorySHA256, specpin.SectionInventorySHA256V070)
	}
	wantScope := []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}
	if !reflect.DeepEqual(source.NormativeScope, wantScope) {
		t.Errorf("normative scope = %v, want %v", source.NormativeScope, wantScope)
	}

	if len(manifest.Contracts) != 64 {
		t.Fatalf("contract rows = %d, want 64", len(manifest.Contracts))
	}
	// The v0.7.0 launch-plan delta: one new row and five widened version
	// lists, each read from the production entry point, not retyped prose.
	// CLI Result is pinned unchanged at 5.0.0: the delta must not widen it.
	wantDelta := map[string][]string{
		"Launch Plan request\x00urn:ax:schema:launch-plan-request": {"1.0.0"},
		"Session record\x00urn:ax:schema:session-record":           {"1.0.0", "2.0.0", "3.0.0", "3.1.0"},
		"Provider protocol\x00urn:ax:protocol:provider":            {"2.0.0", "2.1.0", "3.0.0", "3.1.0"},
		"Provider manifest\x00urn:ax:schema:provider-manifest":     {"1.0.0", "1.1.0"},
		"Provider probe\x00urn:ax:schema:provider-probe":           {"1.0.0", "1.1.0"},
		"Structured error\x00urn:ax:schema:error":                  {"1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0", "1.5.0"},
		"CLI result\x00urn:ax:schema:cli-result":                   {"1.0.0", "2.0.0", "3.0.0", "4.0.0", "5.0.0"},
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
			t.Errorf("v0.7.0 delta row %q is absent from the verified registry", key)
		}
	}
	// The Launch Plan request row sits between Session record and Session
	// event in specification-table order, not appended at the end.
	for index, contract := range manifest.Contracts {
		if contract.Name != "Launch Plan request" {
			continue
		}
		if index == 0 || manifest.Contracts[index-1].Name != "Session record" ||
			index+1 >= len(manifest.Contracts) || manifest.Contracts[index+1].Name != "Session event" {
			t.Errorf("Launch Plan request row at index %d is not between Session record and Session event", index)
		}
	}

	if manifest.Compatibility.BaselineRelease != specpin.ReleaseV043 {
		t.Errorf("compatibility baseline = %q, want %q", manifest.Compatibility.BaselineRelease, specpin.ReleaseV043)
	}
	if len(manifest.Compatibility.AbsentContracts) != 9 {
		t.Errorf("absent contracts = %v, want the five Terminal Backend rows plus the three v0.6.0 rows plus Launch Plan request",
			manifest.Compatibility.AbsentContracts)
	}
	if len(manifest.Compatibility.VersionOverrides) != 9 {
		t.Fatalf("version overrides = %d rows, want 9", len(manifest.Compatibility.VersionOverrides))
	}

	wantFixtures := map[string]string{
		specpin.SessionDirectoryFixtureID:  "a6351a83e25a3a909297ed20bd1f4a75622b10f536a06b164fff3b12cb66f2ce",
		specpin.TerminalBackendFixtureID:   "67de0d78d76c9c445c742af5c4c14ffa5cecd620d4cb07dc5497d391b421ad37",
		specpin.RoadmapV043FixtureID:       "6023ec0d1562e8868b8bef3dc41cfd66ea0b4a4054fbaf13d3aec504578a7f74",
		specpin.HostChannelFixtureID:       "0d6529a8fd2ff4e03be98b694f62b58fa2f04cde7ca44b0382a1fdd56c532332",
		specpin.SessionSelectorFixtureID:   "f0f1cb2d7265bc224f57c04859cf0719c772cd08a266dc1c9abdd3bb04ecf913",
		specpin.LaunchPlanRequestFixtureID: "ec4310b8ded6562a3933960898b85b7521c8ed4fea703b8562921443334df241",
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

// TestVerifyV070RejectsWrongSourceHashAndVersionEvidence is the refusal half
// of adoption: every stale, substituted, trimmed, or invented identity the
// gate must reject is driven through the production VerifyV070 entry point.
// The stale-commit and stale-object rows preserve every v0.7.0 token and
// change only the pointed-to history, so a checker that matched tokens
// instead of binding them would admit them.
func TestVerifyV070RejectsWrongSourceHashAndVersionEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "invented release",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["release"] = "v0.7.1"
			},
		},
		{
			name: "stale tag",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["tag"] = "v0.6.0"
			},
		},
		{
			name: "stale tag object keeps v0.7.0 tokens",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["tag_object"] = specpin.TagObjectV060
			},
		},
		{
			name: "stale commit keeps v0.7.0 tokens",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["commit"] = specpin.CommitV060
			},
		},
		{
			name: "stale document digest",
			mutate: func(document map[string]any) {
				document["source"].(map[string]any)["document"].(map[string]any)["sha256"] = specpin.DocumentSHA256V060
			},
		},
		{
			name: "forged section inventory digest",
			mutate: func(document map[string]any) {
				// The v0.7.0 inventory digest is measured equal to the
				// v0.6.0 one, so a stale-digest row cannot exist; a forged
				// digest must still be refused.
				document["source"].(map[string]any)["section_inventory_sha256"] = strings.Repeat("0", 64)
			},
		},
		{
			name: "trimmed provider protocol delta",
			mutate: func(document map[string]any) {
				for _, row := range document["contracts"].([]any) {
					contract := row.(map[string]any)
					if contract["name"] == "Provider protocol" {
						contract["versions"] = []any{"2.0.0", "3.0.0"}
					}
				}
			},
		},
		{
			name: "dropped launch plan request row",
			mutate: func(document map[string]any) {
				kept := document["contracts"].([]any)[:0]
				for _, row := range document["contracts"].([]any) {
					if row.(map[string]any)["name"] == "Launch Plan request" {
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
					if contract["name"] == "Launch Plan request" {
						contract["id"] = "urn:ax:capability:launch-plan-request"
					}
				}
			},
		},
		{
			name: "substituted launch plan fixture digest",
			mutate: func(document map[string]any) {
				document["fixtures"].([]any)[5].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			},
		},
		{
			name: "stale host channel fixture digest",
			mutate: func(document map[string]any) {
				document["fixtures"].([]any)[3].(map[string]any)["sha256"] = "20aa66c363bcb660f7ed45ac4e760457e408fc030df1c784b71857f96961ae3b"
			},
		},
		{
			name: "unsupported capability claim",
			mutate: func(document map[string]any) {
				document["capabilities"] = []any{"caller-launch-plan"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := mutatePinV070(t, test.mutate)
			if _, err := specpin.VerifyV070(candidate); !errors.Is(err, specpin.ErrPinMismatch) {
				t.Fatalf("VerifyV070() error = %v, want ErrPinMismatch", err)
			}
		})
	}
}

// TestVerifyV070RefusesForeignReleaseBytes proves the three gates are
// version-bound: each older gate refuses the adopted v0.7.0 bytes (a stale
// consumer cannot be silently promoted), and the v0.7.0 gate refuses both
// historical byte strings (the new authority admits no old evidence).
func TestVerifyV070RefusesForeignReleaseBytes(t *testing.T) {
	if _, err := specpin.Verify(specpin.BytesV070()); !errors.Is(err, specpin.ErrPinMismatch) {
		t.Fatalf("Verify(v0.7.0 bytes) error = %v, want ErrPinMismatch", err)
	}
	if _, err := specpin.VerifyV060(specpin.BytesV070()); !errors.Is(err, specpin.ErrPinMismatch) {
		t.Fatalf("VerifyV060(v0.7.0 bytes) error = %v, want ErrPinMismatch", err)
	}
	if _, err := specpin.VerifyV070(specpin.BytesV060()); !errors.Is(err, specpin.ErrPinMismatch) {
		t.Fatalf("VerifyV070(v0.6.0 bytes) error = %v, want ErrPinMismatch", err)
	}
	if _, err := specpin.VerifyV070(specpin.Bytes()); !errors.Is(err, specpin.ErrPinMismatch) {
		t.Fatalf("VerifyV070(v0.5.0 bytes) error = %v, want ErrPinMismatch", err)
	}
}

func TestVerifyV070RejectsPartialMalformedAndByteDifferentReads(t *testing.T) {
	raw := specpin.BytesV070()
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
			if _, err := specpin.VerifyV070(test.candidate); !errors.Is(err, specpin.ErrPinMismatch) {
				t.Fatalf("VerifyV070() error = %v, want ErrPinMismatch", err)
			}
		})
	}
}

// TestContractsForReleaseV070 drives the production projection entry point
// on the verified v0.7.0 manifest. The historical projections are not
// restated: the v0.6.0 rows must equal the real v0.6.0 lock row for row, the
// v0.5.0 rows must equal the real v0.5.0 lock row for row, and the v0.4.3
// rows must equal the v0.6.0 manifest's own v0.4.3 projection, so adoption
// cannot rewrite history without failing here.
func TestContractsForReleaseV070(t *testing.T) {
	manifest, err := specpin.CurrentV070()
	if err != nil {
		t.Fatalf("CurrentV070() error = %v", err)
	}

	current, err := manifest.ContractsForRelease(specpin.ReleaseV070)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.7.0) error = %v", err)
	}
	if len(current) != 64 {
		t.Fatalf("v0.7.0 rows = %d, want 64", len(current))
	}

	historical060, err := manifest.ContractsForRelease(specpin.ReleaseV060)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.6.0) error = %v", err)
	}
	pinned060, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("CurrentV060() error = %v", err)
	}
	if !reflect.DeepEqual(historical060, pinned060.Contracts) {
		t.Fatal("v0.7.0 manifest's v0.6.0 projection differs from the real v0.6.0 lock; history was rewritten")
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
		t.Fatal("v0.7.0 manifest's v0.5.0 projection differs from the real v0.5.0 lock; history was rewritten")
	}

	historical043, err := manifest.ContractsForRelease(specpin.ReleaseV043)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.4.3) error = %v", err)
	}
	pinned043, err := pinned060.ContractsForRelease(specpin.ReleaseV043)
	if err != nil {
		t.Fatalf("v0.6.0 ContractsForRelease(v0.4.3) error = %v", err)
	}
	if !reflect.DeepEqual(historical043, pinned043) {
		t.Fatal("v0.7.0 manifest's v0.4.3 projection differs from the v0.6.0 manifest's; the baseline moved")
	}
	if len(historical043) != 55 {
		t.Fatalf("v0.4.3 rows = %d, want 55", len(historical043))
	}

	for _, release := range []string{"v0.4.2", "v0.7.1", "v0.7.0正式"} {
		if _, err := manifest.ContractsForRelease(release); !errors.Is(err, specpin.ErrUnsupportedRelease) {
			t.Errorf("ContractsForRelease(%q) error = %v, want ErrUnsupportedRelease", release, err)
		}
	}
}

// TestStalePinsCannotAuthorizeV070 pins the direction of trust: a verified
// v0.5.0 manifest answers for v0.5.0 and v0.4.3 only, and a verified v0.6.0
// manifest answers for v0.6.0 and below only. Asking either for v0.7.0 is
// refused, so old evidence can never stand in for the adopted release.
func TestStalePinsCannotAuthorizeV070(t *testing.T) {
	historical050, err := specpin.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if _, err := historical050.ContractsForRelease(specpin.ReleaseV070); !errors.Is(err, specpin.ErrUnsupportedRelease) {
		t.Fatalf("v0.5.0 ContractsForRelease(v0.7.0) error = %v, want ErrUnsupportedRelease", err)
	}

	historical060, err := specpin.CurrentV060()
	if err != nil {
		t.Fatalf("CurrentV060() error = %v", err)
	}
	if _, err := historical060.ContractsForRelease(specpin.ReleaseV070); !errors.Is(err, specpin.ErrUnsupportedRelease) {
		t.Fatalf("v0.6.0 ContractsForRelease(v0.7.0) error = %v, want ErrUnsupportedRelease", err)
	}
}

func TestCurrentV070IsIdempotentAndReturnsIsolatedData(t *testing.T) {
	first, err := specpin.CurrentV070()
	if err != nil {
		t.Fatalf("first CurrentV070() error = %v", err)
	}
	second, err := specpin.CurrentV070()
	if err != nil {
		t.Fatalf("second CurrentV070() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated CurrentV070() calls returned different manifests")
	}

	first.Contracts[0].Versions[0] = "9.9.9"
	third, err := specpin.CurrentV070()
	if err != nil {
		t.Fatalf("third CurrentV070() error = %v", err)
	}
	if third.Contracts[0].Versions[0] != "1.0.0" {
		t.Fatalf("caller mutation leaked into embedded pin: %v", third.Contracts[0].Versions)
	}

	raw := specpin.BytesV070()
	raw[0] = ' '
	if _, err := specpin.CurrentV070(); err != nil {
		t.Fatalf("caller mutation of BytesV070 leaked into the embed: %v", err)
	}
}

func mutatePinV070(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()

	var document map[string]any
	if err := json.Unmarshal(specpin.BytesV070(), &document); err != nil {
		t.Fatalf("decode embedded v0.7.0 pin: %v", err)
	}
	mutate(document)
	candidate, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mutated pin: %v", err)
	}
	return candidate
}
