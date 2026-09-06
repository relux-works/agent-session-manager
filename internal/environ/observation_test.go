package environ_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file batteries the Section 10.8.1 Environment
// Observation gate. No facade validates observation content
// today — dirnode crosses observations as presence and object
// shape only, explicitly deferring content to this leaf — so
// these rows prove new behavior, not agreement. The roster is
// still derived and fail-closed: the required-member list comes
// from environ production, every required member owns at least
// one refusal row, and a row naming a member production no
// longer requires fails as orphaned.
//
// Conjoined rules count twice throughout: sorted-unique evidence
// owns an unsorted-but-unique row AND a sorted-but-duplicated
// row, and reason coherence owns an available-with-reason row
// AND a conditional-without-reason row, so a check enforcing
// only one half cannot pass both.

// observationRow is one battery vector: the required member it
// violates ("" for vocabulary and structural rules), the body,
// and the rule phrase the refusal must carry. An empty phrase
// means the entry accepts and the decoded value is asserted.
type observationRow struct {
	member string
	name   string
	body   []byte
	text   string
}

// capabilityValueJSON builds one CapabilityResult with the given
// status, reason literal (already JSON), and evidence literal.
func capabilityValueJSON(status, reason, evidence string) string {
	return `{"status":` + jsonQuote(status) + `,"reason_code":` + reason + `,"evidence_ids":` + evidence + `,"observed_at":` + jsonQuote(fixtureObservedAt) + `,"extensions":{}}`
}

// observationCorpus mutates each observation member alone
// against an otherwise-valid body.
func observationCorpus(t *testing.T) []observationRow {
	t.Helper()
	valid := validObservationJSON()
	drop := func(member string) []byte { return dropMember(t, valid, member) }
	set := func(member, literal string) []byte { return mutateMember(t, valid, member, literal) }
	evidencePair := sortedDigests(fixtureEvidenceID, fixtureExecutable)
	unsortedEvidence := `["` + evidencePair[1] + `","` + evidencePair[0] + `"]`
	duplicatedEvidence := `["` + evidencePair[0] + `","` + evidencePair[0] + `"]`
	capWith := func(status, reason, evidence string) []byte {
		capabilities := strings.Replace(validCapabilitiesJSON(), availableCapabilityJSON(), capabilityValueJSON(status, reason, evidence), 1)
		return mutateMember(t, valid, "capabilities", capabilities)
	}
	sevenCaps := sevenCapabilitiesJSON(t)
	return []observationRow{
		{"", "valid", valid, ""},
		{"schema", "missing schema", drop("schema"), "misses a required member"},
		{"schema", "wrong schema", set("schema", `"urn:ax:schema:session-adapter-manifest"`), "not the environment observation"},
		{"schema_version", "wrong version", set("schema_version", `"2.0.0"`), "not 1.0.0"},
		{"observation_id", "bad observation digest", set("observation_id", `"sha256:zzzz"`), "not a digest"},
		{"host_id", "bad host uuid", set("host_id", `"not-a-uuid"`), "not a UUIDv7"},
		{"installation_id", "bad installation digest", set("installation_id", `"42"`), "not a digest"},
		{"environment_id", "bad environment id", set("environment_id", `"Test.Env"`), "environment-id"},
		{"environment_id", "missing environment id", drop("environment_id"), "misses a required member"},
		{"environment_version", "empty version", set("environment_version", `""`), "string[1..128]"},
		{"environment_version", "long version", set("environment_version", jsonQuote(strings.Repeat("v", 129))), "string[1..128]"},
		{"provider_id", "uppercase provider", set("provider_id", `"Test-Provider"`), "provider-id"},
		{"platform", "darwin refused", set("platform", `"darwin"`), "platform"},
		{"platform", "wsl2 admits", set("platform", `"wsl2"`), ""},
		{"architecture", "x86 refused", set("architecture", `"x86"`), "architecture"},
		{"backend_realm_fingerprint", "bad realm digest", set("backend_realm_fingerprint", `"null"`), "not a digest"},
		{"capabilities", "seven capabilities refused", set("capabilities", sevenCaps), "exact eight-name"},
		{"capabilities", "ninth capability refused", set("capabilities", nineCapabilitiesJSON(t)), "exact eight-name"},
		{"capabilities", "ninth capability refused", appendMember(t, valid, "capabilities-extra", `{}`), "unknown member"},
		{"capabilities", "unknown capability key", set("capabilities", unknownCapabilityJSON(t)), "exact eight-name"},
		{"capabilities", "bad status refused", capWith("ready", `null`, `[]`), "exact eight-name"},
		{"capabilities", "available with reason refused", capWith("available", jsonQuote("why"), `[]`), "exact eight-name"},
		{"capabilities", "conditional without reason refused", capWith("conditional", `null`, `[]`), "exact eight-name"},
		{"capabilities", "conditional with reason admits", capWith("conditional", jsonQuote("why"), `[]`), ""},
		{"capabilities", "unsorted evidence refused", capWith("available", `null`, unsortedEvidence), "exact eight-name"},
		{"capabilities", "duplicated evidence refused", capWith("available", `null`, duplicatedEvidence), "exact eight-name"},
		{"capabilities", "bad evidence digest refused", capWith("available", `null`, `["sha256:zzzz"]`), "exact eight-name"},
		{"authentication_status", "bad auth status", set("authentication_status", `"signed-in"`), "authentication status"},
		{"runtime_status", "bad runtime status", set("runtime_status", `"sleeping"`), "runtime status"},
		{"observed_at", "bad timestamp", set("observed_at", `"yesterday"`), "not a timestamp"},
		{"observed_at", "missing timestamp", drop("observed_at"), "misses a required member"},
		{"extensions", "bad extensions key", set("extensions", `{"nodots":1}`), "reverse-DNS"},
		{"", "unknown top member", appendMember(t, valid, "score", `1`), "unknown member"},
	}
}

// sortedDigests returns two distinct fixture digests in JCS
// (lexicographic) order.
func sortedDigests(first, second string) []string {
	pair := []string{first, second}
	sort.Strings(pair)
	if pair[0] == pair[1] {
		panic("sortedDigests: fixtures collide")
	}
	return pair
}

// sevenCapabilitiesJSON returns the capability map without its
// first registry name.
func sevenCapabilitiesJSON(t *testing.T) string {
	t.Helper()
	full := validCapabilitiesJSON()
	without := strings.Replace(full, `"directory_discovery":`+availableCapabilityJSON()+`,`, "", 1)
	if without == full {
		t.Fatal("sevenCapabilitiesJSON: anchor not found")
	}
	return without
}

// nineCapabilitiesJSON returns the capability map with a ninth
// valid-valued key: the length arm, distinct from the unknown-key
// arm the swap below exercises.
func nineCapabilitiesJSON(t *testing.T) string {
	t.Helper()
	full := validCapabilitiesJSON()
	nine := strings.Replace(full, `"native_resume":`, `"extra_surface":`+availableCapabilityJSON()+`,"native_resume":`, 1)
	if nine == full {
		t.Fatal("nineCapabilitiesJSON: anchor not found")
	}
	return nine
}

// unknownCapabilityJSON returns the eight-name map with one key
// swapped for an unregistered name.
func unknownCapabilityJSON(t *testing.T) string {
	t.Helper()
	full := validCapabilitiesJSON()
	swapped := strings.Replace(full, `"directory_discovery"`, `"directory_exploration"`, 1)
	if swapped == full {
		t.Fatal("unknownCapabilityJSON: anchor not found")
	}
	return swapped
}

// TestObservationRosterIsClosed requires every derived required
// member to own at least one row, and every row to name a
// derived member (or "" for vocabulary and structural rules).
func TestObservationRosterIsClosed(t *testing.T) {
	required := deriveRequiredList(t, "environ", "observation.go", "observationRequired")
	derived := map[string]bool{}
	for _, member := range required {
		derived[member] = true
	}
	covered := map[string]bool{}
	for _, row := range observationCorpus(t) {
		if row.member == "" {
			continue
		}
		if !derived[row.member] {
			t.Fatalf("row %q names underived member %q", row.name, row.member)
		}
		covered[row.member] = true
	}
	for _, member := range required {
		if !covered[member] {
			t.Errorf("derived required member %q owns no battery row", member)
		}
	}
}

// TestDecodeEnvironmentObservation drives the corpus through the
// production entry: accepts decode to the fixture values, and
// every refusal carries its full rule rendering.
func TestDecodeEnvironmentObservation(t *testing.T) {
	for _, row := range observationCorpus(t) {
		t.Run(row.name, func(t *testing.T) {
			observation, err := environ.DecodeEnvironmentObservation(row.body)
			if row.text == "" {
				if err != nil {
					t.Fatalf("refused %s: %v", row.body, err)
				}
				assertObservationFixture(t, observation)
				return
			}
			if err == nil {
				t.Fatalf("admitted %s, want rule %q", row.body, row.text)
			}
			if !strings.Contains(err.Error(), row.text) {
				t.Fatalf("error = %v, want rule %q", err, row.text)
			}
		})
	}
}

// assertObservationFixture requires the decoded accept to carry
// the shared fixture identities: the verdict is accept AND the
// values are the fixture's, so a decoder that accepts while
// misreading a member cannot pass.
func assertObservationFixture(t *testing.T, observation environ.Observation) {
	t.Helper()
	if observation.ObservationID != fixtureObserveID {
		t.Fatalf("observation id = %q, want %q", observation.ObservationID, fixtureObserveID)
	}
	if observation.HostID != fixtureHostID {
		t.Fatalf("host id = %q, want %q", observation.HostID, fixtureHostID)
	}
	if observation.InstallationID != fixtureInstallID {
		t.Fatalf("installation id = %q, want %q", observation.InstallationID, fixtureInstallID)
	}
	if observation.EnvironmentID != fixtureEnvironmentID {
		t.Fatalf("environment id = %q, want %q", observation.EnvironmentID, fixtureEnvironmentID)
	}
	if observation.ProviderID != fixtureProviderID {
		t.Fatalf("provider id = %q, want %q", observation.ProviderID, fixtureProviderID)
	}
	if len(observation.Capabilities) != 8 {
		t.Fatalf("capabilities = %d entries, want the exact eight", len(observation.Capabilities))
	}
}

// TestDirectoryCapabilityTableIsShared requires the environ and
// dirnode directory-capability tables to be literally equal. The
// adapter fifteen-name and provider seven-name tables share the
// symbol name but are different rules in different stories, so
// they are out of scope here by rule identity, not by silence:
// this comparison names its files.
func TestDirectoryCapabilityTableIsShared(t *testing.T) {
	mine := deriveRequiredList(t, "environ", "observation.go", "capabilityOrder")
	theirs := deriveRequiredList(t, "dirnode", "manifest.go", "capabilityOrder")
	if !reflect.DeepEqual(mine, theirs) {
		t.Fatalf("directory capability tables differ:\nenviron: %q\ndirnode:  %q", mine, theirs)
	}
	if len(mine) != 8 {
		t.Fatalf("directory capability table has %d names, want the exact eight", len(mine))
	}
}

// TestArchitectureTableIsShared requires the three architecture
// tables in scope to be literally equal.
func TestArchitectureTableIsShared(t *testing.T) {
	mine := deriveRequiredList(t, "environ", "tuple.go", "tupleArchitectures")
	adapter := deriveRequiredList(t, "sessadapter", "tuple.go", "tupleArchitectures")
	node := deriveRequiredList(t, "dirnode", "probe.go", "architectures")
	if !reflect.DeepEqual(mine, adapter) || !reflect.DeepEqual(mine, node) {
		t.Fatalf("architecture tables differ:\nenviron: %q\nsessadapter: %q\ndirnode: %q", mine, adapter, node)
	}
}

// TestNegotiatedPlatformsParse requires every platform token in
// the derived dirnode v2 vocabulary to parse as a scalar
// platform, and requires the legacy v1 darwin token to refuse:
// the vocabularies are per-major by design, and a token that
// crossed majors silently would break the Section 7.9 binding.
func TestNegotiatedPlatformsParse(t *testing.T) {
	v2 := deriveRequiredList(t, "dirnode", "probe.go", "platformV2")
	if len(v2) != 4 {
		t.Fatalf("platformV2 has %d tokens, want four", len(v2))
	}
	for _, token := range v2 {
		if _, err := scalar.ParsePlatform(token); err != nil {
			t.Fatalf("v2 token %q does not parse as a platform: %v", token, err)
		}
	}
	if _, err := scalar.ParsePlatform("darwin"); err == nil {
		t.Fatal("legacy darwin parses as a platform; the majors are no longer separated")
	}
}
