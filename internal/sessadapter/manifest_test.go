package sessadapter

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestDecodeManifestAcceptsFixture(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	if manifest.ProviderID != fixtureProviderID {
		t.Fatalf("ProviderID = %q, want %q", manifest.ProviderID, fixtureProviderID)
	}
	if manifest.EnvironmentID != fixtureEnvironmentID {
		t.Fatalf("EnvironmentID = %q, want %q", manifest.EnvironmentID, fixtureEnvironmentID)
	}
	if manifest.AdapterVersion != fixtureAdapterVer {
		t.Fatalf("AdapterVersion = %q, want %q", manifest.AdapterVersion, fixtureAdapterVer)
	}
	if len(manifest.Operations) != 14 || len(manifest.Capabilities) != 15 {
		t.Fatalf("registries = %d/%d, want 14/15", len(manifest.Operations), len(manifest.Capabilities))
	}
	digest := ManifestDigest(manifest)
	if _, err := scalar.ParseDigest(digest.String()); err != nil {
		t.Fatalf("ManifestDigest is not a digest: %v", err)
	}
	// The digest is stable: decoding the same bytes twice digests
	// identically, and it changes when any byte changes.
	again, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	if ManifestDigest(again) != digest {
		t.Fatal("manifest digest is not deterministic")
	}
	changed := mutateMember(t, []byte(fixtureManifestJSON()), "display_name", `"Changed Adapter"`)
	mutated, err := DecodeManifest(changed)
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	if ManifestDigest(mutated) == digest {
		t.Fatal("manifest digest does not change with the body")
	}
}

// TestManifestDigestGolden pins the host-computed manifest digest
// of the fixture. The value was verified against python hashlib
// over the emitted sorted-key JCS bytes, which reproduced it
// exactly. A mutant that changes canonicalization, member
// handling, or hash input changes this digest and reddens here.
func TestManifestDigestGolden(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	const golden = "sha256:079ebf3739e84cda1cc36487b359a26d0ab27377bd2dfb53d9ad6f344c26892d"
	if ManifestDigest(manifest).String() != golden {
		t.Fatalf("ManifestDigest = %s, want golden %s", ManifestDigest(manifest), golden)
	}
}

func TestDecodeManifestClosedMemberRules(t *testing.T) {
	fixture := []byte(fixtureManifestJSON())
	members := []string{"schema", "schema_version", "provider_id", "environment_id", "display_name", "adapter_version", "environment_version_range", "platforms", "operations", "capability_names", "extensions"}
	for _, member := range members {
		t.Run("missing "+member, func(t *testing.T) {
			_, err := DecodeManifest(dropMember(t, fixture, member))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "misses a required member")
		})
		t.Run("null "+member, func(t *testing.T) {
			_, err := DecodeManifest(mutateMember(t, fixture, member, `null`))
			if err == nil {
				t.Fatalf("DecodeManifest(%s=null) accepted", member)
			}
			if failureCode(t, err) != "session_adapter_protocol_error" {
				t.Fatalf("code = %v, want session_adapter_protocol_error", err)
			}
		})
	}
	t.Run("unknown member", func(t *testing.T) {
		_, err := DecodeManifest(mutateMember(t, fixture, "extensions", `{"com.example.extra":"x"}`))
		if err != nil {
			t.Fatalf("DecodeManifest with reverse-DNS extension: %v", err)
		}
		_, err = DecodeManifest(appendMember(t, fixture, "unexpected", `true`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "unexpected", "unknown member")
	})
}

func appendMember(t *testing.T, body []byte, member, value string) []byte {
	t.Helper()
	trimmed := strings.TrimSuffix(strings.TrimSpace(string(body)), "}")
	if strings.TrimSpace(trimmed) == "{" {
		return []byte(trimmed + `"` + member + `":` + value + `}`)
	}
	return []byte(trimmed + `,"` + member + `":` + value + `}`)
}

func TestDecodeManifestRegistryMutations(t *testing.T) {
	fixture := []byte(fixtureManifestJSON())
	operations := func() []string {
		names := make([]string, 0, len(operationOrder))
		for _, operation := range operationOrder {
			names = append(names, `"`+string(operation)+`"`)
		}
		return names
	}
	t.Run("omit operation", func(t *testing.T) {
		names := operations()[:13]
		_, err := DecodeManifest(mutateMember(t, fixture, "operations", `[`+strings.Join(names, ",")+`]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "operations", "fourteen-name registry")
	})
	t.Run("add operation", func(t *testing.T) {
		names := append(operations(), `"launch"`)
		_, err := DecodeManifest(mutateMember(t, fixture, "operations", `[`+strings.Join(names, ",")+`]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "operations", "fourteen-name registry")
	})
	t.Run("reorder operations", func(t *testing.T) {
		names := operations()
		names[0], names[1] = names[1], names[0]
		_, err := DecodeManifest(mutateMember(t, fixture, "operations", `[`+strings.Join(names, ",")+`]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "operations", "fourteen-name registry")
	})
	t.Run("duplicate operation", func(t *testing.T) {
		names := append(operations(), `"doctor"`)
		_, err := DecodeManifest(mutateMember(t, fixture, "operations", `[`+strings.Join(names, ",")+`]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "operations", "fourteen-name registry")
	})
	t.Run("omit capability", func(t *testing.T) {
		_, err := DecodeManifest(mutateMember(t, fixture, "capability_names", `["native_discovery"]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capability_names", "fifteen-name registry")
	})
	t.Run("reorder capabilities", func(t *testing.T) {
		names := make([]string, 0, len(capabilityOrder))
		for _, name := range capabilityOrder {
			names = append(names, `"`+name+`"`)
		}
		names[0], names[14] = names[14], names[0]
		_, err := DecodeManifest(mutateMember(t, fixture, "capability_names", `[`+strings.Join(names, ",")+`]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capability_names", "fifteen-name registry")
	})
}

func TestDecodeManifestValueRules(t *testing.T) {
	fixture := []byte(fixtureManifestJSON())
	for _, test := range []struct {
		name   string
		member string
		value  string
		text   string
	}{
		{"wrong schema", "schema", `"urn:ax:schema:provider-manifest"`, "not the session adapter manifest"},
		{"wrong version", "schema_version", `"2.0.0"`, "not 1.0.0"},
		{"bad provider id", "provider_id", `"Test-Provider"`, "not a provider-id"},
		{"empty provider id", "provider_id", `""`, "not a provider-id"},
		{"bad environment id", "environment_id", `"Test Env"`, "not an environment-id"},
		{"empty display name", "display_name", `""`, "not a string[1..128]"},
		{"bad adapter version", "adapter_version", `"1.2"`, "not SemVer"},
		{"empty version range", "environment_version_range", `""`, "not a non-empty string[1..256]"},
		{"empty platforms", "platforms", `[]`, "not a sorted unique non-empty subset"},
		{"unknown platform", "platforms", `["plan9"]`, "outside linux|macos|windows|wsl2"},
		{"unsorted platforms", "platforms", `["macos","linux"]`, "not a sorted unique non-empty subset"},
		{"duplicate platforms", "platforms", `["linux","linux"]`, "not a sorted unique non-empty subset"},
		{"bad extensions", "extensions", `{"not-dns":"x"}`, "not reverse-DNS keyed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeManifest(mutateMember(t, fixture, test.member, test.value))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", test.member, test.text)
		})
	}
	// display_name boundary: 128 admits, 129 refuses. Characters,
	// not bytes: 128 multi-byte runes admit.
	for _, test := range []struct {
		name  string
		value string
		ok    bool
	}{
		{"128 ascii", strings.Repeat("a", 128), true},
		{"129 ascii", strings.Repeat("a", 129), false},
		{"128 multibyte", strings.Repeat("é", 128), true},
		{"129 multibyte", strings.Repeat("é", 129), false},
	} {
		t.Run("display_name "+test.name, func(t *testing.T) {
			_, err := DecodeManifest(mutateMember(t, fixture, "display_name", `"`+test.value+`"`))
			if test.ok && err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			if !test.ok && err == nil {
				t.Fatal("DecodeManifest admitted an over-long display name")
			}
		})
	}
}
