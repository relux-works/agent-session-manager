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
	ManifestSHA256 = "6fa3a22be22525b4a0146fe9f995aad63c49c2230e86c7039f8eacb87df1ab8c"
)

// The v0.6.0 constants below adopt the independently accepted signed release
// agent-session-manager-spec v0.6.0 (annotated tag object
// 40c123eb8399efa8e05cbc009110940ed861a785 peeling to commit
// 0cbdf100dbf84df50c64f792b1f940e3a67859a6) as the adopted normative source
// authority. Every v0.5.0 constant above keeps its historical meaning: nothing
// here relabels the old lock, document, or inventory. Current/Verify still
// serve the v0.5.0 baseline for the intentionally-historical platform-path
// registry; the catalogue, traceability, and CI gates consume v0.6.0 through
// CurrentV060/VerifyV060 and derive the historical projections from it.
const (
	ReleaseV060 = "v0.6.0"
	TagV060     = "v0.6.0"
	// TagObjectV060 is the verified annotated tag object; CommitV060 is the
	// peeled commit it points to. Both were verified with git verify-tag and
	// git verify-commit before adoption.
	TagObjectV060 = "40c123eb8399efa8e05cbc009110940ed861a785"
	CommitV060    = "0cbdf100dbf84df50c64f792b1f940e3a67859a6"
	// DocumentSHA256V060 is the SHA-256 of the 1150005-byte SPEC.md blob at
	// the peeled commit, measured from the tagged blob, not the worktree.
	DocumentSHA256V060 = "74504539fb43c28ae3450622bc1002e643f116cd3e14df4567a882231e90896b"

	// HostChannelFixtureID is the fixture discriminator the upstream
	// host_channel_conformance.json file declares for itself.
	HostChannelFixtureID = "ax-host-channel-conformance-v1"
	// SessionSelectorFixtureID is pin-local: the upstream
	// session_selector_conformance.json file carries no fixture
	// discriminator (it binds specification_version 0.6.0 with contract
	// 1.0.0 instead), so the identifier follows the mechanical
	// path-stem convention of the other rows and is bound by the pinned
	// path and SHA-256 below.
	SessionSelectorFixtureID = "ax-session-selector-conformance-v1"

	// ManifestSHA256V060 pins the exact v0.6.0 lock bytes embedded below.
	ManifestSHA256V060 = "005cd4ffb5aba6792786727bf7d9fa59305191c81b210aae8d3d564d80661fb3"
)

var (
	// ErrPinMismatch reports malformed, partial, drifted, or substituted pin data.
	ErrPinMismatch = errors.New("normative source pin mismatch")

	// ErrUnsupportedRelease reports a release outside the pinned compatibility set.
	ErrUnsupportedRelease = errors.New("unsupported specification release")
)

//go:embed v0.5.0.lock.json
var embeddedPin []byte

//go:embed v0.6.0.lock.json
var embeddedPinV060 []byte

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
	Repository             string      `json:"repository"`
	Release                string      `json:"release"`
	Tag                    string      `json:"tag"`
	TagObject              string      `json:"tag_object"`
	Commit                 string      `json:"commit"`
	Document               DocumentPin `json:"document"`
	SectionInventorySHA256 string      `json:"section_inventory_sha256"`
	NormativeScope         []string    `json:"normative_scope"`
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

// BytesV060 returns an isolated copy of the exact embedded v0.6.0 lock bytes.
func BytesV060() []byte {
	return bytes.Clone(embeddedPinV060)
}

// Current returns a newly decoded copy of the embedded, verified release pin.
//
// Current still serves the v0.5.0 baseline: the platform-path registry in
// internal/localstore binds that historical source explicitly through
// CurrentV050. Catalogue, traceability, and CI gates consume the adopted
// v0.6.0 authority through CurrentV060 instead.
func Current() (Manifest, error) {
	return Verify(embeddedPin)
}

// CurrentV050 is the explicit historical v0.5.0 baseline reader. It is the
// binding intentionally-historical consumers declare, so a future change to
// what Current serves cannot silently re-point them at a newer authority.
func CurrentV050() (Manifest, error) {
	return Verify(embeddedPin)
}

// CurrentV060 returns a newly decoded copy of the embedded, verified v0.6.0
// release pin: the adopted normative source authority.
func CurrentV060() (Manifest, error) {
	return VerifyV060(embeddedPinV060)
}

func decode(candidate []byte) (Manifest, error) {
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
	return manifest, nil
}

// Verify accepts only the exact embedded release lock. It distinguishes a
// failed or partial read from absence by returning ErrPinMismatch.
func Verify(candidate []byte) (Manifest, error) {
	manifest, err := decode(candidate)
	if err != nil {
		return Manifest{}, err
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

// VerifyV060 accepts only the exact embedded v0.6.0 release lock. The v0.5.0
// lock, a partial read, and any drifted or substituted pin are refused with
// ErrPinMismatch, exactly as Verify refuses everything but the v0.5.0 bytes.
func VerifyV060(candidate []byte) (Manifest, error) {
	manifest, err := decode(candidate)
	if err != nil {
		return Manifest{}, err
	}

	if err := validateV060(manifest); err != nil {
		return Manifest{}, err
	}

	digest := sha256.Sum256(candidate)
	if hex.EncodeToString(digest[:]) != ManifestSHA256V060 {
		return Manifest{}, mismatch("lock digest is not %s", ManifestSHA256V060)
	}

	return manifest, nil
}

// ContractsForRelease returns an isolated ordered contract registry for one
// of the releases explicitly represented by this pin.
//
// A v0.5.0 manifest represents v0.5.0 and v0.4.3, exactly as before: asking
// it for v0.6.0 is refused, because a stale pin cannot authorize a release
// it predates. A v0.6.0 manifest represents v0.6.0, and derives the exact
// historical v0.5.0 and v0.4.3 registries without rewriting them: the v0.5.0
// projection drops the three v0.6.0 rows and trims the four widened version
// lists, which the cross-version test proves byte-equal to the real v0.5.0
// lock.
func (manifest Manifest) ContractsForRelease(release string) ([]ContractPin, error) {
	switch release {
	case ReleaseV060:
		if manifest.Source.Release != ReleaseV060 {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedRelease, release)
		}
		return cloneContracts(manifest.Contracts), nil
	case ReleaseV050:
		if manifest.Source.Release == ReleaseV060 {
			return derivedV050Contracts(manifest)
		}
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

// v060OnlyContractKeys are the registry rows the prepared v0.6.0 revision
// adds: the selector delta (Session selector) and the authentication delta
// (Host Channel, Host Trust Store). They are absent from every earlier
// registry, so the historical projections drop them by key, never by
// position.
func v060OnlyContractKeys() map[string]struct{} {
	keys := map[string]struct{}{}
	for _, contract := range []ContractPin{
		{Name: "Host Channel", ID: "urn:ax:transport:host-channel"},
		{Name: "Host Trust Store", ID: "urn:ax:schema:host-trust-store"},
		{Name: "Session selector", ID: "urn:ax:contract:session-selector"},
	} {
		keys[contractKey(contract)] = struct{}{}
	}
	return keys
}

// v050VersionCeilings pins the exact versions each widened v0.6.0 row carried
// in v0.5.0, per the specification's own historical-registry account: the
// selector delta adds CLI Result 5.0.0 and Structured Error 1.4.0, and the
// authentication delta adds Configuration 4.0.0 and Mesh RPC 5.0.0. Every
// other v0.6.0 row is byte-identical to its v0.5.0 self.
func v050VersionCeilings() map[string][]string {
	return map[string][]string{
		"Configuration\x00urn:ax:schema:config":   {"1.0.0", "2.0.0", "3.0.0"},
		"Mesh RPC\x00urn:ax:protocol:rpc":         {"2.0.0", "3.0.0", "4.0.0"},
		"Structured error\x00urn:ax:schema:error": {"1.0.0", "1.1.0", "1.2.0", "1.3.0"},
		"CLI result\x00urn:ax:schema:cli-result":  {"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
	}
}

// derivedV050Contracts projects the exact historical v0.5.0 registry out of
// a verified v0.6.0 manifest: drop the three v0.6.0-only rows, trim the four
// widened version lists to their v0.5.0 ceilings, keep registry order. The
// result must equal the real v0.5.0 lock row for row; the cross-version test
// enforces that, so history cannot drift here without failing loudly.
func derivedV050Contracts(manifest Manifest) ([]ContractPin, error) {
	dropped := v060OnlyContractKeys()
	ceilings := v050VersionCeilings()
	contracts := make([]ContractPin, 0, len(manifest.Contracts)-len(dropped))
	for _, contract := range manifest.Contracts {
		key := contractKey(contract)
		if _, added := dropped[key]; added {
			continue
		}
		contract = cloneContract(contract)
		if ceiling, widened := ceilings[key]; widened {
			contract.Versions = append([]string(nil), ceiling...)
		}
		contracts = append(contracts, contract)
	}
	return contracts, nil
}

func validate(manifest Manifest) error {
	if manifest.Format != Format || manifest.FormatVersion != FormatVersion {
		return mismatch("unsupported pin format %q version %d", manifest.Format, manifest.FormatVersion)
	}

	source := manifest.Source
	if source.Repository != Repository || source.Release != ReleaseV050 || source.Tag != TagV050 ||
		source.TagObject != TagObjectV050 || source.Commit != CommitV050 ||
		source.Document.Path != DocumentPath || source.Document.SHA256 != DocumentSHA256 ||
		source.SectionInventorySHA256 != SectionInventorySHA256 {
		return mismatch("source identity drift")
	}
	if !hex40.MatchString(source.TagObject) || !hex40.MatchString(source.Commit) ||
		!hex64.MatchString(source.Document.SHA256) || !hex64.MatchString(source.SectionInventorySHA256) {
		return mismatch("malformed source digest")
	}
	if !reflect.DeepEqual(source.NormativeScope, []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}) {
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

// validateV060 accepts only the exact adopted v0.6.0 source identity: the
// verified signed tag, peeled commit, document and section-inventory digests,
// the 63-row prepared registry in specification order, the v0.4.3 baseline
// with the three new rows absent, and the five shipped fixtures. The
// contract-identifier rule additionally admits the two namespaces the v0.6.0
// registry introduces (transport, contract); the v0.5.0 rule above is
// untouched, so the old gate cannot be loosened from here.
func validateV060(manifest Manifest) error {
	if manifest.Format != Format || manifest.FormatVersion != FormatVersion {
		return mismatch("unsupported pin format %q version %d", manifest.Format, manifest.FormatVersion)
	}

	source := manifest.Source
	if source.Repository != Repository || source.Release != ReleaseV060 || source.Tag != TagV060 ||
		source.TagObject != TagObjectV060 || source.Commit != CommitV060 ||
		source.Document.Path != DocumentPath || source.Document.SHA256 != DocumentSHA256V060 ||
		source.SectionInventorySHA256 != SectionInventorySHA256V060 {
		return mismatch("source identity drift")
	}
	if !hex40.MatchString(source.TagObject) || !hex40.MatchString(source.Commit) ||
		!hex64.MatchString(source.Document.SHA256) || !hex64.MatchString(source.SectionInventorySHA256) {
		return mismatch("malformed source digest")
	}
	if !reflect.DeepEqual(source.NormativeScope, []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}) {
		return mismatch("normative scope drift")
	}

	if len(manifest.Contracts) != 63 {
		return mismatch("contract registry has %d rows, want 63", len(manifest.Contracts))
	}
	seenContracts := make(map[string]struct{}, len(manifest.Contracts))
	for index, contract := range manifest.Contracts {
		key := contractKey(contract)
		if contract.Name == "" || !(strings.HasPrefix(contract.ID, "urn:ax:schema:") ||
			strings.HasPrefix(contract.ID, "urn:ax:protocol:") ||
			strings.HasPrefix(contract.ID, "urn:ax:transport:") ||
			strings.HasPrefix(contract.ID, "urn:ax:contract:")) {
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
	for _, contract := range []ContractPin{
		{Name: "Host Channel", ID: "urn:ax:transport:host-channel", Versions: []string{"1.0.0"}},
		{Name: "Host Trust Store", ID: "urn:ax:schema:host-trust-store", Versions: []string{"1.0.0"}},
		{Name: "Session selector", ID: "urn:ax:contract:session-selector", Versions: []string{"1.0.0"}},
		{Name: "Configuration", ID: "urn:ax:schema:config", Versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}},
		{Name: "Mesh RPC", ID: "urn:ax:protocol:rpc", Versions: []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"}},
		{Name: "Structured error", ID: "urn:ax:schema:error", Versions: []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0"}},
		{Name: "CLI result", ID: "urn:ax:schema:cli-result", Versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0", "5.0.0"}},
	} {
		found := false
		for _, row := range manifest.Contracts {
			if contractKey(row) == contractKey(contract) && reflect.DeepEqual(row.Versions, contract.Versions) {
				found = true
				break
			}
		}
		if !found {
			return mismatch("v0.6.0 delta row %q is missing or drifted", contract.Name)
		}
	}

	compatibility := manifest.Compatibility
	if compatibility.BaselineRelease != ReleaseV043 || compatibility.RegistrySHA256 != HistoricalRegistrySHA256 {
		return mismatch("compatibility baseline drift")
	}
	if !reflect.DeepEqual(compatibility.AbsentContracts, expectedAbsentContractsV060()) {
		return mismatch("v0.4.3 absent-contract set drift")
	}
	if !reflect.DeepEqual(compatibility.VersionOverrides, expectedVersionOverrides()) {
		return mismatch("v0.4.3 version overrides drift")
	}

	if !reflect.DeepEqual(manifest.Fixtures, expectedFixturesV060()) {
		return mismatch("fixture identity drift")
	}
	for _, fixture := range manifest.Fixtures {
		if fixture.ID == "" || fixture.Path == "" || !hex64.MatchString(fixture.SHA256) {
			return mismatch("malformed fixture identity %q", fixture.ID)
		}
	}

	return nil
}

// expectedAbsentContractsV060 is the v0.4.3 absent set as the v0.6.0
// specification states it: the five Terminal Backend rows plus the three
// rows v0.6.0 adds, in registry order. The six version overrides are
// unchanged and shared with the v0.5.0 gate.
func expectedAbsentContractsV060() []string {
	return []string{
		"Terminal Backend protocol",
		"Terminal Backend manifest",
		"Terminal Backend probe",
		"Terminal Instance binding",
		"Terminal capability evidence",
		"Host Channel",
		"Host Trust Store",
		"Session selector",
	}
}

// expectedFixturesV060 pins all five fixtures the v0.6.0 source ships. The
// first three digests are byte-identical to the v0.5.0 pin: the upstream
// revision did not touch those files, and the cross-version test proves the
// equality rather than restating it.
func expectedFixturesV060() []FixturePin {
	return []FixturePin{
		{ID: SessionDirectoryFixtureID, Path: "fixtures/session_directory_conformance.json", SHA256: "a6351a83e25a3a909297ed20bd1f4a75622b10f536a06b164fff3b12cb66f2ce"},
		{ID: TerminalBackendFixtureID, Path: "fixtures/terminal_backend_conformance.json", SHA256: "67de0d78d76c9c445c742af5c4c14ffa5cecd620d4cb07dc5497d391b421ad37"},
		{ID: RoadmapV043FixtureID, Path: "fixtures/v0_4_3_roadmap_terminal_realm.json", SHA256: "6023ec0d1562e8868b8bef3dc41cfd66ea0b4a4054fbaf13d3aec504578a7f74"},
		{ID: HostChannelFixtureID, Path: "fixtures/host_channel_conformance.json", SHA256: "20aa66c363bcb660f7ed45ac4e760457e408fc030df1c784b71857f96961ae3b"},
		{ID: SessionSelectorFixtureID, Path: "fixtures/session_selector_conformance.json", SHA256: "2bda47f5ad79911a3ce3db1ec7c8fbefbd2023fcc9f39cdc0eb22ee86b605770"},
	}
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
