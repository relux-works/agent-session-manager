// Package specpin exposes the immutable upstream specification identity that
// this implementation consumes. It does not advertise product capabilities.
package specpin

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	Format                   = "ax-normative-source-pin"
	FormatVersion            = 1
	Repository               = "relux-works/agent-session-manager-spec"
	ReleaseV050              = "v0.5.0"
	ReleaseV043              = "v0.4.3"
	TagV050                  = "v0.5.0"
	TagObjectV050            = "d3da6614a6c7bf119a88c9596a86c0853c22cfb9"
	CommitV050               = "28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c"
	DocumentPath             = "SPEC.md"
	DocumentSHA256           = "562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a"
	HistoricalRegistrySHA256 = "sha256:958186993a6e59bbbc8e7fafc828f5913c4252fe964df4107132209c62f9fd83"

	SessionDirectoryFixtureID = "ax-session-directory-conformance-v1"
	TerminalBackendFixtureID  = "ax-terminal-backend-conformance-v1"
	RoadmapV043FixtureID      = "ax-v0.4.3-roadmap-terminal-realm-v1"

	// ManifestSHA256 pins the exact bytes embedded in this package.
	ManifestSHA256 = "edd67e84cb173fb66efb9d719b28ae02cf7cd5c7c1462551f5c7667b35cf78b6"
)

var (
	// ErrPinMismatch reports malformed, partial, drifted, or substituted pin data.
	ErrPinMismatch = errors.New("normative source pin mismatch")

	// ErrUnsupportedRelease reports a release outside the pinned compatibility set.
	ErrUnsupportedRelease = errors.New("unsupported specification release")
)

//go:embed v0.5.0.lock.json
var embeddedPin []byte

var (
	hex40  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hex64  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	semver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
)

// Manifest is implementation metadata, not an independently versioned AX wire
// contract. Its only accepted instance is the embedded release lock.
type Manifest struct {
	Format        string           `json:"format"`
	FormatVersion int              `json:"format_version"`
	Source        SourcePin        `json:"source"`
	Contracts     []ContractPin    `json:"contracts"`
	Compatibility CompatibilityPin `json:"compatibility"`
	Fixtures      []FixturePin     `json:"fixtures"`
}

type SourcePin struct {
	Repository     string      `json:"repository"`
	Release        string      `json:"release"`
	Tag            string      `json:"tag"`
	TagObject      string      `json:"tag_object"`
	Commit         string      `json:"commit"`
	Document       DocumentPin `json:"document"`
	NormativeScope []string    `json:"normative_scope"`
}

type DocumentPin struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type ContractPin struct {
	Name     string   `json:"name"`
	ID       string   `json:"id"`
	Versions []string `json:"versions"`
}

type CompatibilityPin struct {
	BaselineRelease  string        `json:"baseline_release"`
	RegistrySHA256   string        `json:"registry_sha256"`
	AbsentContracts  []string      `json:"absent_contracts"`
	VersionOverrides []ContractPin `json:"version_overrides"`
}

type FixturePin struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Bytes returns an isolated copy of the exact embedded lock bytes.
func Bytes() []byte {
	return bytes.Clone(embeddedPin)
}

// Current returns a newly decoded copy of the embedded, verified release pin.
func Current() (Manifest, error) {
	return Verify(embeddedPin)
}

// Verify accepts only the exact embedded release lock. It distinguishes a
// failed or partial read from absence by returning ErrPinMismatch.
func Verify(candidate []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(candidate))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, mismatch("decode pin: %v", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Manifest{}, mismatch("multiple JSON values")
		}
		return Manifest{}, mismatch("trailing JSON: %v", err)
	}

	if err := validate(manifest); err != nil {
		return Manifest{}, err
	}

	digest := sha256.Sum256(candidate)
	if hex.EncodeToString(digest[:]) != ManifestSHA256 {
		return Manifest{}, mismatch("lock digest is not %s", ManifestSHA256)
	}

	return manifest, nil
}

// ContractsForRelease returns an isolated ordered contract registry for one of
// the two releases explicitly represented by this pin.
func (manifest Manifest) ContractsForRelease(release string) ([]ContractPin, error) {
	switch release {
	case ReleaseV050:
		return cloneContracts(manifest.Contracts), nil
	case ReleaseV043:
		absent := make(map[string]struct{}, len(manifest.Compatibility.AbsentContracts))
		for _, name := range manifest.Compatibility.AbsentContracts {
			absent[name] = struct{}{}
		}

		overrides := make(map[string]ContractPin, len(manifest.Compatibility.VersionOverrides))
		for _, contract := range manifest.Compatibility.VersionOverrides {
			overrides[contractKey(contract)] = contract
		}

		contracts := make([]ContractPin, 0, len(manifest.Contracts)-len(absent))
		for _, contract := range manifest.Contracts {
			if _, removed := absent[contract.Name]; removed {
				continue
			}
			if override, ok := overrides[contractKey(contract)]; ok {
				contract = override
			}
			contracts = append(contracts, cloneContract(contract))
		}
		return contracts, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedRelease, release)
	}
}

// Fixture returns the exact pinned identity for a shipped upstream fixture.
func (manifest Manifest) Fixture(id string) (FixturePin, bool) {
	for _, fixture := range manifest.Fixtures {
		if fixture.ID == id {
			return fixture, true
		}
	}
	return FixturePin{}, false
}

func validate(manifest Manifest) error {
	if manifest.Format != Format || manifest.FormatVersion != FormatVersion {
		return mismatch("unsupported pin format %q version %d", manifest.Format, manifest.FormatVersion)
	}

	source := manifest.Source
	if source.Repository != Repository || source.Release != ReleaseV050 || source.Tag != TagV050 ||
		source.TagObject != TagObjectV050 || source.Commit != CommitV050 ||
		source.Document.Path != DocumentPath || source.Document.SHA256 != DocumentSHA256 {
		return mismatch("source identity drift")
	}
	if !hex40.MatchString(source.TagObject) || !hex40.MatchString(source.Commit) || !hex64.MatchString(source.Document.SHA256) {
		return mismatch("malformed source digest")
	}
	if !reflect.DeepEqual(source.NormativeScope, []string{"1", "17", "20", "appendix-a", "appendix-d"}) {
		return mismatch("normative scope drift")
	}

	if len(manifest.Contracts) != 60 {
		return mismatch("contract registry has %d rows, want 60", len(manifest.Contracts))
	}
	seenContracts := make(map[string]struct{}, len(manifest.Contracts))
	for index, contract := range manifest.Contracts {
		key := contractKey(contract)
		if contract.Name == "" || !(strings.HasPrefix(contract.ID, "urn:ax:schema:") || strings.HasPrefix(contract.ID, "urn:ax:protocol:")) {
			return mismatch("contract row %d has invalid name or identifier", index)
		}
		if _, duplicate := seenContracts[key]; duplicate {
			return mismatch("duplicate contract row %q", contract.Name)
		}
		seenContracts[key] = struct{}{}
		if err := validateVersions(contract); err != nil {
			return err
		}
	}

	compatibility := manifest.Compatibility
	if compatibility.BaselineRelease != ReleaseV043 || compatibility.RegistrySHA256 != HistoricalRegistrySHA256 {
		return mismatch("compatibility baseline drift")
	}
	if !reflect.DeepEqual(compatibility.AbsentContracts, expectedAbsentContracts()) {
		return mismatch("v0.4.3 absent-contract set drift")
	}
	if !reflect.DeepEqual(compatibility.VersionOverrides, expectedVersionOverrides()) {
		return mismatch("v0.4.3 version overrides drift")
	}

	if !reflect.DeepEqual(manifest.Fixtures, expectedFixtures()) {
		return mismatch("fixture identity drift")
	}
	for _, fixture := range manifest.Fixtures {
		if fixture.ID == "" || fixture.Path == "" || !hex64.MatchString(fixture.SHA256) {
			return mismatch("malformed fixture identity %q", fixture.ID)
		}
	}

	return nil
}

func validateVersions(contract ContractPin) error {
	if len(contract.Versions) == 0 {
		return mismatch("contract %q has no versions", contract.Name)
	}
	seen := make(map[string]struct{}, len(contract.Versions))
	for index, version := range contract.Versions {
		if !semver.MatchString(version) {
			return mismatch("contract %q has invalid version %q", contract.Name, version)
		}
		if _, duplicate := seen[version]; duplicate {
			return mismatch("contract %q repeats version %q", contract.Name, version)
		}
		seen[version] = struct{}{}
		if index > 0 && compareSemver(contract.Versions[index-1], version) >= 0 {
			return mismatch("contract %q versions are not strictly increasing", contract.Name)
		}
	}
	return nil
}

func compareSemver(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for index := range leftParts {
		leftValue, _ := strconv.Atoi(leftParts[index])
		rightValue, _ := strconv.Atoi(rightParts[index])
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return 0
}

func expectedAbsentContracts() []string {
	return []string{
		"Terminal Backend protocol",
		"Terminal Backend manifest",
		"Terminal Backend probe",
		"Terminal Instance binding",
		"Terminal capability evidence",
	}
}

func expectedVersionOverrides() []ContractPin {
	return []ContractPin{
		{Name: "Configuration", ID: "urn:ax:schema:config", Versions: []string{"1.0.0", "2.0.0"}},
		{Name: "Provider protocol", ID: "urn:ax:protocol:provider", Versions: []string{"2.0.0"}},
		{Name: "Mesh RPC", ID: "urn:ax:protocol:rpc", Versions: []string{"2.0.0", "3.0.0"}},
		{Name: "Session event", ID: "urn:ax:schema:session-event", Versions: []string{"1.0.0", "2.0.0", "3.0.0"}},
		{Name: "Structured error", ID: "urn:ax:schema:error", Versions: []string{"1.0.0", "1.1.0", "1.2.0"}},
		{Name: "CLI result", ID: "urn:ax:schema:cli-result", Versions: []string{"1.0.0", "2.0.0", "3.0.0"}},
	}
}

func expectedFixtures() []FixturePin {
	return []FixturePin{
		{ID: SessionDirectoryFixtureID, Path: "fixtures/session_directory_conformance.json", SHA256: "a6351a83e25a3a909297ed20bd1f4a75622b10f536a06b164fff3b12cb66f2ce"},
		{ID: TerminalBackendFixtureID, Path: "fixtures/terminal_backend_conformance.json", SHA256: "67de0d78d76c9c445c742af5c4c14ffa5cecd620d4cb07dc5497d391b421ad37"},
		{ID: RoadmapV043FixtureID, Path: "fixtures/v0_4_3_roadmap_terminal_realm.json", SHA256: "6023ec0d1562e8868b8bef3dc41cfd66ea0b4a4054fbaf13d3aec504578a7f74"},
	}
}

func contractKey(contract ContractPin) string {
	return contract.Name + "\x00" + contract.ID
}

func cloneContracts(contracts []ContractPin) []ContractPin {
	cloned := make([]ContractPin, len(contracts))
	for index, contract := range contracts {
		cloned[index] = cloneContract(contract)
	}
	return cloned
}

func cloneContract(contract ContractPin) ContractPin {
	contract.Versions = append([]string(nil), contract.Versions...)
	return contract
}

func mismatch(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrPinMismatch, fmt.Sprintf(format, arguments...))
}
