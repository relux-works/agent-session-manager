package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// saturatedVersion1Document carries every Configuration 1.0.0 member. Each
// member that the validators do not pin to a single value is set away from the
// default it would decay to, so a member dropped by an encoder is observable
// instead of silently reappearing as its default.
// TestSaturatedVersion1FixtureCoversEveryConfiguration1Member derives the
// member set from rawV1 and fails if this document stops being saturated.
func saturatedVersion1Document() []byte {
	return []byte(fmt.Sprintf(`schema = %q
schema_version = %q
host_id = %q
host_name = "saturated-fixture-host"
platform = "macos"

[mesh]
transport = "ssh"
sync_interval_seconds = 61
connect_timeout_seconds = 11
rpc_timeout_seconds = 301
workspace_replication = false
payload_encryption = "none"

[[mesh.peers]]
host_id = %q
name = "saturated-peer"
endpoint = "peer.example"
platform = "linux"
ssh_args = ["-o", "BatchMode=yes"]

[[mesh.peers.workspace_roots]]
logical_root = "peer-root"
path = "/srv/peer"

[[workspace_roots]]
logical_root = "relux"
path = "/Users/test/Developer"

[providers]
plugin_dirs = ["/Users/test/.local/libexec/ax/providers"]
allow_path_plugins = false
require_explicit_trust = true

[sync]
chunk_bytes = 4194304
max_parallel_chunks = 5
staging_retention_hours = 73
tombstone_min_retention_days = 91

[terminal]
backend = "tmux"
safe_boundary_timeout_seconds = 42
graceful_stop_timeout_seconds = 61

[service]
enabled = false
health_interval_seconds = 31

[restore]
auto_resume = true

[profiles.yolo]
require_first_use_confirmation = false
`, SchemaID, Version1, testHostID, testPeerID))
}

// version1SaturationExemption records a Configuration 1.0.0 member that cannot
// be set away from its default and still load. The exemption is never taken on
// trust: alternate is driven through the real Load entry and must be refused,
// so an exemption that stops being true reddens instead of hiding a member.
type version1SaturationExemption struct {
	reason string
	// key is the TOML assignment key whose first occurrence is replaced.
	key string
	// alternate is the replacement literal, or the empty string when the
	// falsifying edit is to remove the member entirely.
	alternate string
}

var version1SaturationExemptions = map[string]version1SaturationExemption{
	"schema": {
		reason:    "the document schema identity, not an operator-tunable member",
		key:       "schema",
		alternate: `"urn:example:other"`,
	},
	"schema_version": {
		reason:    "the migration source marker; the migrated version is asserted separately by SourceVersion",
		key:       "schema_version",
		alternate: `"9.0.0"`,
	},
	"host_id": {
		reason:    "required member with no default to decay to",
		key:       "host_id",
		alternate: "",
	},
	"host_name": {
		reason:    "required member with no default to decay to",
		key:       "host_name",
		alternate: "",
	},
	"platform": {
		reason:    "must equal the runtime platform probe, so the fixture platform is the only accepted value",
		key:       "platform",
		alternate: `"linux"`,
	},
	"mesh.transport": {
		reason:    "the validator admits exactly one transport",
		key:       "transport",
		alternate: `"quic"`,
	},
	"mesh.payload_encryption": {
		reason:    "the validator admits exactly one payload encryption value",
		key:       "payload_encryption",
		alternate: `"aead"`,
	},
	"providers.require_explicit_trust": {
		reason:    "the validator admits only the trust-requiring value",
		key:       "require_explicit_trust",
		alternate: "false",
	},
	"sync.chunk_bytes": {
		reason:    "the validator admits exactly one chunk size",
		key:       "chunk_bytes",
		alternate: "8388608",
	},
	"terminal.backend": {
		reason:    "the legacy backend vocabulary is pinned to the runtime platform, so macOS admits only tmux",
		key:       "backend",
		alternate: `"conpty"`,
	},
}

// TestSaturatedVersion1FixtureCoversEveryConfiguration1Member derives the
// Configuration 1.0.0 member set from the versioned wire type rather than
// listing it, so a member added to rawV1 later is covered without editing this
// file. It proves the fixture is complete, that every non-exempt member is
// distinguishable from the value it would decay to, and that every exemption
// is real: the declared alternate is refused at the production Load entry.
func TestSaturatedVersion1FixtureCoversEveryConfiguration1Member(t *testing.T) {
	fixture := flattenTOMLDocument(t, saturatedVersion1Document())
	defaults := flattenTOMLDocument(t, migratedDefaultDocument(t))
	declared := wireLeafTOMLPaths(reflect.TypeOf(rawV1{}), "")
	if len(declared) == 0 {
		t.Fatal("derived no Configuration 1.0.0 member paths from rawV1")
	}

	for _, path := range declared {
		value, present := fixture[path]
		if !present {
			t.Errorf("Configuration 1.0.0 member %q is declared by rawV1 but absent from the saturated fixture", path)
			continue
		}
		if exemption, exempt := version1SaturationExemptions[path]; exempt {
			assertVersion1ExemptionIsReal(t, path, exemption)
			continue
		}
		if strings.Contains(path, "[]") {
			// The enclosing collection is empty by default, so a populated
			// entry is itself the value that distinguishes the fixture.
			if _, defaulted := defaults[path]; defaulted {
				t.Errorf("collection member %q is present in a defaults-only document, so a populated entry proves nothing", path)
			}
			continue
		}
		if defaultValue, defaulted := defaults[path]; defaulted && reflect.DeepEqual(value, defaultValue) {
			t.Errorf("member %q carries its default value %v, so a dropped member would be invisible; set it away from the default or declare an exemption", path, defaultValue)
		}
	}

	for path := range fixture {
		if !containsString(declared, path) {
			t.Errorf("saturated fixture carries %q, which rawV1 does not declare", path)
		}
	}
	for path := range version1SaturationExemptions {
		if !containsString(declared, path) {
			t.Errorf("exemption declared for %q, which rawV1 does not declare", path)
		}
	}
}

// TestMigrationRetainsEveryConfiguration1MemberOnEveryTarget drives the real
// Migrate entry with a source that sets every Configuration 1.0.0 member away
// from its default and compares the whole loaded Configuration before and
// after. A whole-struct comparison is what makes a later added member fail
// closed: any member an encoder drops decays to its default and reddens here.
func TestMigrationRetainsEveryConfiguration1MemberOnEveryTarget(t *testing.T) {
	const choice = "mesh_sanitized"
	source := saturatedVersion1Document()

	directory, filename := seedMigrationDocument(t, source)
	before := loadMigrated(t, directory, filename)
	if before.SourceVersion != Version1 {
		t.Fatalf("saturated fixture source version = %q, want %s", before.SourceVersion, Version1)
	}
	if before.Value.Terminal.SafeBoundaryTimeoutSeconds == 300 || before.Value.Terminal.GracefulStopTimeoutSeconds == 60 {
		t.Fatalf("saturated fixture terminal timeouts are still the defaults: %#v", before.Value.Terminal)
	}
	want := before.Value
	want.Directory.GeneratedSummaryUpgradeChoice = choice

	stepped, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
		TargetVersion: Version2, GeneratedSummaryUpgradeChoice: choice,
	})
	if err != nil || !stepped.Changed {
		t.Fatalf("Migrate(saturated v1 to %s) = %#v, %v", Version2, stepped, err)
	}
	afterVersion2 := loadMigrated(t, directory, filename)
	if afterVersion2.SourceVersion != Version2 {
		t.Fatalf("migrated source version = %q, want %s", afterVersion2.SourceVersion, Version2)
	}
	assertConfigurationsMatch(t, Version1+" to "+Version2, want, afterVersion2.Value)

	stepped, err = Migrate(migrationInputs(directory, filename), nil, MigrationOptions{TargetVersion: CurrentVersion})
	if err != nil || !stepped.Changed {
		t.Fatalf("Migrate(saturated %s to %s) = %#v, %v", Version2, CurrentVersion, stepped, err)
	}
	afterCurrent := loadMigrated(t, directory, filename)
	if afterCurrent.SourceVersion != CurrentVersion {
		t.Fatalf("migrated source version = %q, want %s", afterCurrent.SourceVersion, CurrentVersion)
	}
	assertConfigurationsMatch(t, Version2+" to "+CurrentVersion, want, afterCurrent.Value)

	directDirectory, directFilename := seedMigrationDocument(t, source)
	direct, err := Migrate(migrationInputs(directDirectory, directFilename), nil, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: choice,
	})
	if err != nil || !direct.Changed {
		t.Fatalf("Migrate(saturated v1 to %s) = %#v, %v", CurrentVersion, direct, err)
	}
	assertConfigurationsMatch(t, Version1+" to "+CurrentVersion, want, loadMigrated(t, directDirectory, directFilename).Value)
}

func assertConfigurationsMatch(t *testing.T, step string, want, got Configuration) {
	t.Helper()
	differences := configurationDifferences(reflect.ValueOf(want), reflect.ValueOf(got), "")
	if len(differences) > 0 {
		t.Fatalf("migration %s did not retain every member:\n  %s", step, strings.Join(differences, "\n  "))
	}
}

// configurationDifferences reports the exact member paths that changed, so a
// dropped member is named rather than buried in a struct dump.
func configurationDifferences(want, got reflect.Value, prefix string) []string {
	if want.Type() != got.Type() {
		return []string{fmt.Sprintf("%s: type %s != %s", prefix, want.Type(), got.Type())}
	}
	switch want.Kind() {
	case reflect.Struct:
		var differences []string
		for index := 0; index < want.NumField(); index++ {
			field := want.Type().Field(index)
			if !field.IsExported() {
				continue
			}
			differences = append(differences, configurationDifferences(want.Field(index), got.Field(index), joinMemberPath(prefix, field.Name))...)
		}
		return differences
	case reflect.Slice:
		if want.Len() != got.Len() {
			return []string{fmt.Sprintf("%s: length %d != %d", prefix, want.Len(), got.Len())}
		}
		var differences []string
		for index := 0; index < want.Len(); index++ {
			differences = append(differences, configurationDifferences(want.Index(index), got.Index(index), fmt.Sprintf("%s[%d]", prefix, index))...)
		}
		return differences
	default:
		if !reflect.DeepEqual(want.Interface(), got.Interface()) {
			return []string{fmt.Sprintf("%s: %v != %v", prefix, want.Interface(), got.Interface())}
		}
		return nil
	}
}

func joinMemberPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// wireLeafTOMLPaths derives the complete leaf member set of a versioned wire
// struct. An array of tables contributes its element members under a "[]"
// segment so collection members are enumerated too.
func wireLeafTOMLPaths(wire reflect.Type, prefix string) []string {
	for wire.Kind() == reflect.Pointer {
		wire = wire.Elem()
	}
	switch wire.Kind() {
	case reflect.Struct:
		var paths []string
		for index := 0; index < wire.NumField(); index++ {
			field := wire.Field(index)
			tag := strings.Split(field.Tag.Get("toml"), ",")[0]
			if tag == "" || tag == "-" {
				continue
			}
			paths = append(paths, wireLeafTOMLPaths(field.Type, joinTOMLPath(prefix, tag))...)
		}
		return paths
	case reflect.Slice:
		element := wire.Elem()
		for element.Kind() == reflect.Pointer {
			element = element.Elem()
		}
		if element.Kind() == reflect.Struct {
			return wireLeafTOMLPaths(element, prefix+"[]")
		}
		return []string{prefix}
	default:
		return []string{prefix}
	}
}

func joinTOMLPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// flattenTOMLDocument renders a TOML document as leaf path to value, using the
// same "[]" segment convention as wireLeafTOMLPaths.
func flattenTOMLDocument(t *testing.T, document []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := toml.Unmarshal(document, &decoded); err != nil {
		t.Fatalf("flatten TOML document: %v", err)
	}
	flattened := map[string]any{}
	flattenTOMLValue(decoded, "", flattened)
	return flattened
}

func flattenTOMLValue(value any, prefix string, into map[string]any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			flattenTOMLValue(nested, joinTOMLPath(prefix, key), into)
		}
	case []any:
		tables := len(typed) > 0
		for _, element := range typed {
			if _, ok := element.(map[string]any); !ok {
				tables = false
				break
			}
		}
		if !tables {
			into[prefix] = typed
			return
		}
		for _, element := range typed {
			flattenTOMLValue(element, prefix+"[]", into)
		}
	default:
		into[prefix] = value
	}
}

// migratedDefaultDocument is a minimal Configuration 1.0.0 document migrated
// through the real entry point. Every member the migration writes is therefore
// present at the value an omitted member decays to.
func migratedDefaultDocument(t *testing.T) []byte {
	t.Helper()
	directory, filename := seedMigrationDocument(t, minimalValidConfigVersion(scalar.PlatformMacOS, Version1))
	if _, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
	}); err != nil {
		t.Fatalf("Migrate(minimal v1 to %s) error = %v", CurrentVersion, err)
	}
	document, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read migrated defaults: %v", err)
	}
	return document
}

// assertVersion1ExemptionIsReal drives the declared alternate through the real
// Load entry and requires a refusal, so a saturation exemption cannot outlive
// the constraint that justified it.
func assertVersion1ExemptionIsReal(t *testing.T, path string, exemption version1SaturationExemption) {
	t.Helper()
	edited, edits := replaceTOMLAssignment(saturatedVersion1Document(), exemption.key, exemption.alternate)
	if edits != 1 {
		t.Fatalf("exemption %q (%s): edited %d assignments of key %q, want 1", path, exemption.reason, edits, exemption.key)
	}
	directory, filename := seedMigrationDocument(t, edited)
	snapshot, err := Load(migrationInputs(directory, filename), nil)
	if err == nil {
		if _, decoded := snapshot.Configuration(); decoded {
			t.Errorf("exemption %q (%s): the alternate %q loaded successfully, so the member is not pinned and must be saturated instead", path, exemption.reason, exemption.alternate)
		}
	}
}

// replaceTOMLAssignment rewrites the first "key = ..." assignment, or removes
// the line entirely when replacement is empty.
func replaceTOMLAssignment(document []byte, key, replacement string) ([]byte, int) {
	lines := strings.Split(string(document), "\n")
	edits := 0
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if edits == 0 && strings.HasPrefix(line, key+" = ") {
			edits++
			if replacement == "" {
				continue
			}
			kept = append(kept, key+" = "+replacement)
			continue
		}
		kept = append(kept, line)
	}
	return []byte(strings.Join(kept, "\n")), edits
}

func seedMigrationDocument(t *testing.T, document []byte) (directory, filename string) {
	t.Helper()
	directory = t.TempDir()
	filename = filepath.Join(directory, "config.toml")
	if err := os.WriteFile(filename, document, 0o600); err != nil {
		t.Fatal(err)
	}
	return directory, filename
}

func containsString(values []string, want string) bool {
	return slices.Contains(values, want)
}

// TestVersion1SourcesNeverCarryDirectoryCollections names the check that
// subsumes the directory-collection passthrough in encodeVersion2. Migrate
// reaches that encoder only with a Configuration 1.0.0 source, and rawV1
// declares no directory collection, so those three members are always empty
// there and deleting the passthrough is unobservable at the production entry.
// If Configuration 1.0.0 ever gains a directory collection this reddens, and
// the passthrough then needs coverage of its own.
func TestVersion1SourcesNeverCarryDirectoryCollections(t *testing.T) {
	collections := []string{"directory_installations", "directory_enrichment_profiles", "directory_peer_disclosure"}
	for _, path := range wireLeafTOMLPaths(reflect.TypeOf(rawV1{}), "") {
		for _, collection := range collections {
			if strings.HasPrefix(path, collection) {
				t.Fatalf("Configuration 1.0.0 now declares %q, so the encodeVersion2 %s passthrough is reachable and needs its own coverage", path, collection)
			}
		}
	}
	directory, filename := seedMigrationDocument(t, saturatedVersion1Document())
	loaded := loadMigrated(t, directory, filename)
	if loaded.SourceVersion != Version1 {
		t.Fatalf("saturated fixture source version = %q, want %s", loaded.SourceVersion, Version1)
	}
	if len(loaded.Value.DirectoryInstallations) != 0 || len(loaded.Value.DirectoryEnrichmentProfiles) != 0 || len(loaded.Value.DirectoryPeerDisclosure) != 0 {
		t.Fatalf("a saturated Configuration 1.0.0 source produced directory collections: %d/%d/%d",
			len(loaded.Value.DirectoryInstallations), len(loaded.Value.DirectoryEnrichmentProfiles), len(loaded.Value.DirectoryPeerDisclosure))
	}
}
