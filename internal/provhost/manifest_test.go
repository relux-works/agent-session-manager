package provhost

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// specManifestExample is the Section 7.3 manifest example, copied
// verbatim from the pinned document. The derivation test below
// re-reads the same lines from SPEC.md, so drift between this copy
// and the document reddens there rather than passing silently.
const specManifestExample = `{
  "schema": "urn:ax:schema:provider-manifest",
  "schema_version": "1.0.0",
  "provider_id": "pi",
  "display_name": "Pi",
  "plugin_version": "0.1.0",
  "provider_version_range": ">=0.73.1 <0.74.0",
  "platforms": ["linux", "macos", "windows", "wsl2"],
  "operations": [
    "manifest",
    "probe",
    "launch",
    "identify-session",
    "quiesce",
    "native-store-plan",
    "capture",
    "materialize",
    "materialize-status",
    "materialize-commit",
    "materialize-rollback",
    "resume",
    "fork",
    "stop",
    "doctor"
  ],
  "capability_names": [
    "native_resume",
    "portable_store",
    "managed_pty",
    "appserver",
    "task_board_primary",
    "prompt_spawn",
    "native_goal_binding"
  ]
}`

// TestCapabilitiesReturnsACopy proves the copy guarantee the
// accessor documents: mutating a returned slice and re-reading must
// leave the registry unchanged, so the registry cannot be mutated
// through it.
func TestCapabilitiesReturnsACopy(t *testing.T) {
	first := Capabilities()
	if len(first) == 0 {
		t.Fatal("Capabilities() is empty; the check is blind")
	}
	want := first[0]
	first[0] = "remote_exec"
	if got := Capabilities()[0]; got != want {
		t.Fatalf("Capabilities()[0] = %q after aliasing write, want %q", got, want)
	}
}

// TestSpecManifestExampleDecodes proves the Section 7.3 example is a
// well-formed manifest under DecodeManifest: the contract's own
// fixture passes the production entry point.
func TestSpecManifestExampleDecodes(t *testing.T) {
	if err := DecodeManifest([]byte(specManifestExample)); err != nil {
		t.Fatalf("DecodeManifest(spec example): %v", err)
	}
}

// TestManifestRegistriesAreDerivedFromSpec proves the three transcribed
// registries — operations, capability names, platforms — equal the
// Section 7.3 example: the example JSON is re-read from the pinned
// document, never from the constant above, so a drifted transcription
// reddens here.
func TestManifestRegistriesAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.3", 2751, 2785)
	start := strings.Index(window, "{")
	end := strings.LastIndex(window, "}")
	if start < 0 || end <= start {
		t.Fatal("Section 7.3 example holds no JSON object; the check is blind")
	}
	example, fault := decodeStrictObject([]byte(window[start : end+1]))
	if fault != nil {
		t.Fatalf("Section 7.3 example is not a strict object: %v %q", fault.detail, fault.member)
	}
	operations, ok := rawStringArray(example["operations"])
	if !ok {
		t.Fatal("Section 7.3 example operations are not a string array; the check is blind")
	}
	if fmt.Sprintf("%v", operations) != fmt.Sprintf("%v", Operations()) {
		t.Fatalf("Operations() = %v, want the Section 7.3 example %v", Operations(), operations)
	}
	capabilities, ok := rawStringArray(example["capability_names"])
	if !ok {
		t.Fatal("Section 7.3 example capability_names are not a string array; the check is blind")
	}
	if fmt.Sprintf("%v", capabilities) != fmt.Sprintf("%v", Capabilities()) {
		t.Fatalf("Capabilities() = %v, want the Section 7.3 example %v", Capabilities(), capabilities)
	}
	platforms, ok := rawStringArray(example["platforms"])
	if !ok {
		t.Fatal("Section 7.3 example platforms are not a string array; the check is blind")
	}
	if fmt.Sprintf("%v", platforms) != fmt.Sprintf("%v", manifestPlatforms) {
		t.Fatalf("manifestPlatforms = %v, want the Section 7.3 example %v", manifestPlatforms, platforms)
	}
	t.Logf("manifest registry coverage: %d operations, %d capabilities, %d platforms derived", len(operations), len(capabilities), len(platforms))
}

// manifestVariant rewrites one unique substring of the spec example.
// The rewritten body, not the spec fixture, is what the negative rows
// drive.
func manifestVariant(t *testing.T, old, new string) []byte {
	t.Helper()
	if strings.Count(specManifestExample, old) != 1 {
		t.Fatalf("manifest variant anchor %q is not unique", old)
	}
	return []byte(strings.Replace(specManifestExample, old, new, 1))
}

// TestDecodeManifestRefusals drives every manifest rule with a
// fixture carrying only that violation. Each row names the arm
// (member plus rule detail), so a deleted rule slides to a different
// arm or to acceptance and reddens here.
func TestDecodeManifestRefusals(t *testing.T) {
	long128 := strings.Repeat("a", 128)
	long129 := strings.Repeat("a", 129)
	long256 := strings.Repeat("a", 256)
	long257 := strings.Repeat("a", 257)
	id32 := "a" + strings.Repeat("b", 31)
	id33 := "a" + strings.Repeat("b", 32)
	rows := []struct {
		name   string
		body   []byte
		member string
		detail string
	}{
		{"duplicate member", []byte(`{"schema": "urn:ax:schema:provider-manifest", "schema": "urn:ax:schema:provider-manifest"}`), "schema", "duplicate member"},
		{"unknown member", manifestVariant(t, `"display_name": "Pi",`, `"display_name": "Pi", "diagnostics": [],`), "diagnostics", "unknown member"},
		{"missing member", manifestVariant(t, `  "display_name": "Pi",
`, ``), "display_name", "misses a required member"},
		{"wrong schema", manifestVariant(t, `"urn:ax:schema:provider-manifest"`, `"urn:ax:schema:provider-probe"`), "schema", "not the provider manifest"},
		{"wrong schema version", manifestVariant(t, `"schema_version": "1.0.0"`, `"schema_version": "2.0.0"`), "schema_version", "not 1.0.0"},
		{"bad provider id", manifestVariant(t, `"provider_id": "pi"`, `"provider_id": "Pi"`), "provider_id", "not a provider id"},
		{"provider id digit first", manifestVariant(t, `"provider_id": "pi"`, `"provider_id": "1pi"`), "provider_id", "not a provider id"},
		{"provider id 33", manifestVariant(t, `"provider_id": "pi"`, `"provider_id": "`+id33+`"`), "provider_id", "not a provider id"},
		{"empty display name", manifestVariant(t, `"display_name": "Pi"`, `"display_name": ""`), "display_name", "not 1..128 characters"},
		{"display name 129", manifestVariant(t, `"display_name": "Pi"`, `"display_name": "`+long129+`"`), "display_name", "not 1..128 characters"},
		{"bad semver", manifestVariant(t, `"plugin_version": "0.1.0"`, `"plugin_version": "1.0"`), "plugin_version", "not SemVer"},
		{"semver leading zero", manifestVariant(t, `"plugin_version": "0.1.0"`, `"plugin_version": "01.0.0"`), "plugin_version", "not SemVer"},
		{"empty version range", manifestVariant(t, `"provider_version_range": ">=0.73.1 <0.74.0"`, `"provider_version_range": ""`), "provider_version_range", "not 1..256 characters"},
		{"version range 257", manifestVariant(t, `"provider_version_range": ">=0.73.1 <0.74.0"`, `"provider_version_range": "`+long257+`"`), "provider_version_range", "not 1..256 characters"},
		{"platforms empty", manifestVariant(t, `["linux", "macos", "windows", "wsl2"]`, `[]`), "platforms", "empty or not an array"},
		{"platform unknown", manifestVariant(t, `"windows", "wsl2"`, `"windows", "darwin"`), "platforms", "unknown platform"},
		{"platform first unknown", manifestVariant(t, `["linux", "macos", "windows", "wsl2"]`, `["aix", "macos", "windows", "wsl2"]`), "platforms", "unknown platform"},
		{"platforms unsorted", manifestVariant(t, `["linux", "macos", "windows", "wsl2"]`, `["macos", "linux", "windows", "wsl2"]`), "platforms", "not sorted unique"},
		{"platforms duplicated", manifestVariant(t, `["linux", "macos", "windows", "wsl2"]`, `["linux", "linux", "macos", "windows", "wsl2"]`), "platforms", "not sorted unique"},
		{"operations dropped", []byte(strings.Replace(specManifestExample, `,
    "doctor"`, ``, 1)), "operations", "differ from the registry"},
		{"operations reordered", manifestVariant(t, `    "manifest",
    "probe",`, `    "probe",
    "manifest",`), "operations", "differ from the registry"},
		{"operations first substituted", manifestVariant(t, `    "manifest",
    "probe",`, `    "probe",
    "probe",`), "operations", "differ from the registry"},
		{"operations added", manifestVariant(t, `    "stop",
    "doctor"`, `    "stop",
    "doctor",
    "reboot"`), "operations", "differ from the registry"},
		{"operations last substituted", manifestVariant(t, `    "stop",
    "doctor"`, `    "stop",
    "stop"`), "operations", "differ from the registry"},
		{"capability added", manifestVariant(t, `    "prompt_spawn",
    "native_goal_binding"`, `    "prompt_spawn",
    "native_goal_binding",
    "remote_exec"`), "capability_names", "differ from the registry"},
		{"capability dropped", []byte(strings.Replace(specManifestExample, `,
    "native_goal_binding"`, ``, 1)), "capability_names", "differ from the registry"},
		{"capability reordered", manifestVariant(t, `    "native_resume",
    "portable_store",`, `    "portable_store",
    "native_resume",`), "capability_names", "differ from the registry"},
		{"capability first substituted", manifestVariant(t, `    "native_resume",
    "portable_store",`, `    "portable_store",
    "portable_store",`), "capability_names", "differ from the registry"},
		{"capability last substituted", manifestVariant(t, `    "prompt_spawn",
    "native_goal_binding"`, `    "prompt_spawn",
    "prompt_spawn"`), "capability_names", "differ from the registry"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := DecodeManifest(row.body)
			requireFrameRefusal(t, err, row.member, row.detail)
		})
	}
	// Boundary acceptances: the longest admitted values pass, so the
	// bounds above are exact rather than merely sufficient.
	if err := DecodeManifest(manifestVariant(t, `"display_name": "Pi"`, `"display_name": "`+long128+`"`)); err != nil {
		t.Fatalf("128-character display_name refused: %v", err)
	}
	if err := DecodeManifest(manifestVariant(t, `"provider_id": "pi"`, `"provider_id": "`+id32+`"`)); err != nil {
		t.Fatalf("32-character provider_id refused: %v", err)
	}
	if err := DecodeManifest(manifestVariant(t, `"provider_version_range": ">=0.73.1 <0.74.0"`, `"provider_version_range": "`+long256+`"`)); err != nil {
		t.Fatalf("256-character version range refused: %v", err)
	}
}
