package dirnode

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// TestDecodeManifestAcceptsFixture drives the production manifest
// entry with the exact contract vector: every retained field must
// match the fixture literals, and the host-computed digest must be
// stable across decodes.
func TestDecodeManifestAcceptsFixture(t *testing.T) {
	t.Parallel()
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest error = %v", err)
	}
	if manifest.NodeID != fixtureNodeID || manifest.NodeVersion != fixtureNodeVersion || manifest.HostID != fixtureHostID {
		t.Fatalf("manifest identity = %q %q %q", manifest.NodeID, manifest.NodeVersion, manifest.HostID)
	}
	if manifest.ExecutableSHA256 != fixtureExecutableDigest || manifest.ProviderManifestDigest != fixtureProviderDigest || manifest.AdapterManifestDigest != fixtureAdapterDigest {
		t.Fatal("manifest façade bindings do not match the fixture")
	}
	if manifest.TupleRegistryID != fixtureTupleRegistry {
		t.Fatal("manifest tuple registry binding does not match the fixture")
	}
	if len(manifest.Operations) != 11 || len(manifest.Schemas) != 15 {
		t.Fatalf("manifest registries = %d ops %d schemas", len(manifest.Operations), len(manifest.Schemas))
	}
	if len(manifest.Capabilities) != 8 {
		t.Fatalf("manifest capabilities = %d, want 8", len(manifest.Capabilities))
	}
	if !manifest.Supports("1.0.0") || !manifest.Supports("2.0.0") || manifest.Supports("3.0.0") {
		t.Fatal("manifest version support is wrong")
	}
	again, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest again error = %v", err)
	}
	if ManifestDigest(manifest) != ManifestDigest(again) {
		t.Fatal("manifest digest is not stable across decodes")
	}
	if len(manifest.Canonical()) == 0 {
		t.Fatal("manifest canonical bytes are empty")
	}
}

// TestDecodeManifestClosedMemberRules drives the production
// manifest entry with one structural vector per rule: unknown
// members, missing members, wrong schema identity, and shape
// defects in every member class.
func TestDecodeManifestClosedMemberRules(t *testing.T) {
	t.Parallel()
	good := fixtureManifestJSON()
	for _, probe := range []struct {
		name  string
		body  string
		fatal bool
	}{
		{"unknown member", replaceOnce(t, good, `"max_enrichment_bytes":4194304,"extensions":{}},"extensions":{}}`, `"max_enrichment_bytes":4194304,"extensions":{}},"extensions":{},"extra":1}`, 1), true},
		{"missing member", replaceOnce(t, good, `"node_version":"2.4.1",`, ``, 1), true},
		{"wrong schema", replaceOnce(t, good, "session-directory-node-manifest", "session-directory-node-response", 1), true},
		{"wrong version", replaceOnce(t, good, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1), true},
		{"empty node id", replaceOnce(t, good, `"node_id":"test-directory-node"`, `"node_id":""`, 1), true},
		{"bad node version", replaceOnce(t, good, `"node_version":"2.4.1"`, `"node_version":"2.4"`, 1), true},
		{"bad host id", replaceOnce(t, good, fixtureHostID, "not-a-uuid", 1), true},
		{"bad executable digest", replaceOnce(t, good, fixtureExecutableDigest, "sha256:zzz", 1), true},
		{"bad tuple registry", replaceOnce(t, good, fixtureTupleRegistry, "null", 1), true},
		{"duplicate member", good[:len(good)-1] + `,"node_id":"x"}`, true},
		{"non object", `[]`, true},
		{"empty redaction policies", replaceOnce(t, good, `"redaction_policy_ids":[`+quote(fixtureRedactionDigest)+`]`, `"redaction_policy_ids":[]`, 1), true},
		{"bad enrichment profile", replaceOnce(t, good, fixtureEnrichmentDigest, "sha256:zzz", 1), true},
		{"bad top-level extensions", replaceOnce(t, good, `"extensions":{}},"extensions":{}}`, `"extensions":{}},"extensions":{"x":1}}`, 1), true},
		{"deep nesting past canonical bound", deepExtensionsManifest(t, good), true},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeManifest([]byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// deepExtensionsManifest nests one extensions value past the
// canonicalizer's depth bound: the member checks cross extensions
// values opaquely, so the body validates structurally and then
// fails the host-computed canonicalization — the "not canonical
// JSON" arm fires through the production entry, not only in
// theory.
func deepExtensionsManifest(t *testing.T, good string) string {
	t.Helper()
	nested := `{"a":0}`
	for range 300 {
		nested = `{"a":` + nested + `}`
	}
	return deepExtensionsSwap(t, good, nested)
}

// deepExtensionsSwap replaces the manifest top-level extensions
// object with a deeply nested one.
func deepExtensionsSwap(t *testing.T, good, nested string) string {
	t.Helper()
	needle := `"extensions":{}},"extensions":{}}`
	if !strings.Contains(good, needle) {
		t.Fatalf("manifest tail needle absent")
	}
	return replaceOnce(t, good, `"extensions":{}},"extensions":{}}`, `"extensions":{}},"extensions":{"com.example.deep":`+nested+`}}`, 1)
}

// TestDecodeManifestRegistryRules drives the production manifest
// entry against the registry members: reordered, dropped, padded,
// and mis-sorted registries refuse, as do out-of-bound schema
// counts and malformed assertions.
func TestDecodeManifestRegistryRules(t *testing.T) {
	t.Parallel()
	good := fixtureManifestJSON()
	swapped := replaceOnce(t, good, `"operations":["continuation-inspect","doctor"`, `"operations":["doctor","continuation-inspect"`, 1)
	dropped := replaceOnce(t, good, `,"scan"`, ``, 1)
	padded := replaceOnce(t, good, `,"scan"]`, `,"scan","query"]`, 1)
	unsortedSchemas := replaceOnce(t, good, `schema:contract-00`, `schema:contract-zz`, 1)
	fewSchemas := replaceOnce(t, good, `{"contract_id":"urn:ax:schema:contract-00","exact_version":"1.0.0","extensions":{}},`, ``, 1)
	badAssertion := replaceOnce(t, good, `"exact_version":"1.0.0"`, `"exact_version":"one"`, 1)
	extraAssertionMember := replaceOnce(t, good, `"exact_version":"1.0.1"`, `"exact_version":"1.0.1","tier":"gold"`, 1)
	for _, probe := range []struct {
		name string
		body string
	}{
		{"reordered operations", swapped},
		{"dropped operation", dropped},
		{"padded operation", padded},
		{"unsorted schemas", unsortedSchemas},
		{"fourteen schemas", fewSchemas},
		{"non semver assertion", badAssertion},
		{"assertion extra member", extraAssertionMember},
		{"empty versions", replaceOnce(t, good, `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":[]`, 1)},
		{"unsorted versions", replaceOnce(t, good, `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["2.0.0","1.0.0"]`, 1)},
		{"non semver version", replaceOnce(t, good, `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["1.0.0","two"]`, 1)},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeManifest([]byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestManifestCapabilityReasonCoherence drives the production
// manifest entry through the reason-coherence rule in both
// directions: available with a reason refuses, and every
// non-available status without a reason refuses.
func TestManifestCapabilityReasonCoherence(t *testing.T) {
	t.Parallel()
	good := fixtureManifestJSON()
	availableWithReason := replaceOnce(t, good, `"directory_discovery":{"status":"available","reason_code":null`, `"directory_discovery":{"status":"available","reason_code":"x"`, 1)
	conditionalWithoutReason := replaceOnce(t, good, `"directory_head_digest":{"status":"conditional","reason_code":"needs-quiesce"`, `"directory_head_digest":{"status":"conditional","reason_code":null`, 1)
	unavailableWithoutReason := replaceOnce(t, good, `"directory_tail_preview":{"status":"unavailable","reason_code":"not-probed"`, `"directory_tail_preview":{"status":"unavailable","reason_code":null`, 1)
	unknownStatus := replaceOnce(t, good, `"native_runtime_observation":{"status":"unknown"`, `"native_runtime_observation":{"status":"maybe"`, 1)
	ninthCapability := replaceOnce(t, good, `"native_resume":{"status":"available"`, `"native_resume":{"status":"available"},"directory_time_travel":{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`, 1)
	missingCapability := replaceOnce(t, good, `,"native_resume":{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`, ``, 1)
	for _, probe := range []struct {
		name string
		body string
	}{
		{"available with reason", availableWithReason},
		{"conditional without reason", conditionalWithoutReason},
		{"unavailable without reason", unavailableWithoutReason},
		{"unknown status token", unknownStatus},
		{"ninth capability", ninthCapability},
		{"missing capability", missingCapability},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeManifest([]byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestManifestLimitsBounds drives the production manifest entry
// through the DirectoryNodeLimits bounds: each bound refuses past
// its edge in both directions, and a frame ceiling above the 8
// MiB transport bound refuses.
func TestManifestLimitsBounds(t *testing.T) {
	t.Parallel()
	good := fixtureManifestJSON()
	limit := `"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`
	for _, probe := range []struct {
		name   string
		limits string
	}{
		{"frame above transport", `"max_frame_bytes":8388609,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`},
		{"zero frame", `"max_frame_bytes":0,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`},
		{"zero scan instances", `"max_frame_bytes":8388608,"max_scan_instances":0,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`},
		{"take past bound", `"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1001,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`},
		{"excerpt count past bound", `"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":21,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`},
		{"excerpt bytes past bound", `"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4097,"max_enrichment_events":5000,"max_enrichment_bytes":4194304`},
		{"enrichment events past bound", `"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5001,"max_enrichment_bytes":4194304`},
		{"enrichment bytes past bound", `"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194305`},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			body := replaceOnce(t, good, limit, probe.limits, 1)
			if _, err := DecodeManifest([]byte(body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestAvailableCapabilitiesNeverUpgrades drives the production
// advertisement entry: only available capabilities list, and
// conditional, unavailable, and unknown never appear no matter
// their reasons or evidence.
func TestAvailableCapabilitiesNeverUpgrades(t *testing.T) {
	t.Parallel()
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest error = %v", err)
	}
	available := AvailableCapabilities(manifest)
	want := map[string]bool{
		"directory_discovery":        true,
		"directory_incremental_scan": true,
		"native_title_read":          true,
		"existing_session_adoption":  true,
		"native_resume":              true,
	}
	if len(available) != len(want) {
		t.Fatalf("AvailableCapabilities() = %v, want %d entries", available, len(want))
	}
	for _, name := range available {
		if !want[name] {
			t.Fatalf("AvailableCapabilities() upgrades %q", name)
		}
	}
}

// TestCheckManifestBindings drives the production binding entry:
// matching observed facts bind cleanly, malformed observed facts
// refuse before comparison, and any contradicting binding is an
// integrity failure.
func TestCheckManifestBindings(t *testing.T) {
	t.Parallel()
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest error = %v", err)
	}
	if err := CheckManifestBindings(manifest, fixtureObservedFacade()); err != nil {
		t.Fatalf("CheckManifestBindings(matching) error = %v", err)
	}
	contradicted := fixtureObservedFacade()
	contradicted.ExecutableSHA256 = fixtureDigest("other-executable")
	requireCode(t, CheckManifestBindings(manifest, contradicted), "integrity_failure")
	contradicted = fixtureObservedFacade()
	contradicted.ProviderManifestDigest = fixtureDigest("other-provider")
	requireCode(t, CheckManifestBindings(manifest, contradicted), "integrity_failure")
	contradicted = fixtureObservedFacade()
	contradicted.SessionAdapterDigest = fixtureDigest("other-adapter")
	requireCode(t, CheckManifestBindings(manifest, contradicted), "integrity_failure")
	malformed := fixtureObservedFacade()
	malformed.ExecutableSHA256 = "not-a-digest"
	requireCode(t, CheckManifestBindings(manifest, malformed), "invalid_config")
	malformed = fixtureObservedFacade()
	malformed.ProviderManifestDigest = "not-a-digest"
	requireCode(t, CheckManifestBindings(manifest, malformed), "invalid_config")
	malformed = fixtureObservedFacade()
	malformed.SessionAdapterDigest = "not-a-digest"
	requireCode(t, CheckManifestBindings(manifest, malformed), "invalid_config")
}

// TestManifestErrorCodesAreBound verifies the refusal taxonomy the
// suite asserts: every code the manifest path can emit is
// registered under Structured Error 1.2.0, so no test above
// asserts an unmintable code.
func TestManifestErrorCodesAreBound(t *testing.T) {
	t.Parallel()
	for _, code := range []axerror.Code{"adapter_protocol_violation", "integrity_failure", "invalid_config"} {
		if _, err := axerror.ExitCodeFor(axerror.Version120, code); err != nil {
			t.Fatalf("ExitCodeFor(1.2.0, %q) error = %v", code, err)
		}
	}
}

// TestStructuredErrorBinding verifies the package's static error
// binding: the bound version is Structured Error 1.2.0 for both
// majors, and every code any production refusal can emit is
// registered under it with the specified exit class. A code the
// registry does not carry cannot be emitted — axerror.New
// refuses to mint it — so this table is the closed refusal
// taxonomy, not prose.
func TestStructuredErrorBinding(t *testing.T) {
	t.Parallel()
	if ErrorVersion != axerror.Version120 {
		t.Fatalf("ErrorVersion = %q, want 1.2.0", ErrorVersion)
	}
	for _, probe := range []struct {
		code axerror.Code
		exit int
	}{
		{"invalid_config", 3},
		{"adapter_protocol_violation", 9},
		{"operation_unknown", 6},
		{"incompatible_protocol", 6},
		{"integrity_failure", 9},
		{"idempotency_mismatch", 3},
		{"query_invalid", 2},
		{"transport_failure", 8},
	} {
		exit, err := axerror.ExitCodeFor(ErrorVersion, probe.code)
		if err != nil {
			t.Fatalf("ExitCodeFor(1.2.0, %q) error = %v", probe.code, err)
		}
		if exit != probe.exit {
			t.Fatalf("ExitCodeFor(1.2.0, %q) = %d, want %d", probe.code, exit, probe.exit)
		}
	}
}
