// White-box pins for reconciliation arms no document can deliver an input
// to. Parse enforces registry-row equality for every claim (a Manifest or
// Probe cannot redefine a row), so a drifted override or addition fails at
// the document gate before Reconcile runs; likewise, same-ID
// differing-bytes evidence cannot pass parsing because the ID is a hash of
// the bytes. These arms are measured here, at the guard, with typed values
// mutated after parsing, rather than left as prose; the reachability bound
// is stated, not inferred.
package terminalbackend

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"
)

// pinVerifier accepts every attestation: signature validity is owned by the
// black-box suite with real cryptography, and these pins must not depend
// on it.
func pinVerifier(issuerID string, message, signature []byte) error { return nil }

// requirePinRefusal asserts the exact refusal clause, for the same reason
// the black-box suite names its arms: most gates share one wire code, so a
// code-only pin cannot hold the gate it names.
func requirePinRefusal(t *testing.T, err error, wantCode, wantDetail string) {
	t.Helper()

	var refusal *Error
	if !errors.As(err, &refusal) {
		t.Fatalf("error = %v, want *Error with code %q at %q", err, wantCode, wantDetail)
	}
	if refusal.Code != wantCode || refusal.Detail != wantDetail {
		t.Errorf("refusal = %q at %q, want %q at %q", refusal.Code, refusal.Detail, wantCode, wantDetail)
	}
}

// TestReconcileRefusesDriftedOverride pins the override registry-binding
// arm: a probed override of a variable claim whose row members drift from
// the manifest row is refused, even though Parse would already refuse the
// document carrying it.
func TestReconcileRefusesDriftedOverride(t *testing.T) {
	t.Parallel()

	manifest, probe, evidence := validReconcileTriple(t)
	for index := range probe.CapabilityClaims {
		if probe.CapabilityClaims[index].Capability == "headless_creation" {
			probe.CapabilityClaims[index].DependentOperations = []string{"create", "status"}
		}
	}
	_, err := Reconcile(manifest, probe, evidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	requirePinRefusal(t, err, CodeMismatch, "probe override registry binding")
}

// TestReconcileRefusesDriftedAddition pins the addition registry-binding
// arm: a probed addition whose row members drift from the closed registry
// row is refused, even though Parse would already refuse the document.
func TestReconcileRefusesDriftedAddition(t *testing.T) {
	t.Parallel()

	manifest, probe, evidence := validReconcileTriple(t)
	for index := range probe.CapabilityClaims {
		if probe.CapabilityClaims[index].Capability == "graceful_stop" {
			probe.CapabilityClaims[index].EvidenceRequirements = []string{"conformance_fixture"}
		}
	}
	_, err := Reconcile(manifest, probe, evidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	requirePinRefusal(t, err, CodeMismatch, "probe addition registry binding")
}

// TestReconcileRefusesConflictingEvidence pins the same-ID conflict arm:
// two evidence objects sharing an ID but carrying different facts
// invalidate the whole probe. This input cannot arrive through Parse
// (identical IDs over different bytes fail the identity recomputation),
// so it is constructed at the typed level.
func TestReconcileRefusesConflictingEvidence(t *testing.T) {
	t.Parallel()

	manifest, probe, evidence := validReconcileTriple(t)
	twin := evidence[0]
	twin.Facts = []string{"fixture_passed", "runtime_probe_passed", "ui_absent"}
	evidence = append(evidence, twin)
	_, err := Reconcile(manifest, probe, evidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	requirePinRefusal(t, err, CodeMismatch, "conflicting evidence")
}

// TestIdenticalDuplicateEvidenceDeduped pins the stated dedup bound: an
// evidence object supplied twice byte-for-byte contributes its ID once and
// admits. (Attestation signatures are deterministic for one key and
// message, so identical documents are genuinely identical inputs, not two
// observations.) Same-ID differing-bytes input stays refused above.
func TestIdenticalDuplicateEvidenceDeduped(t *testing.T) {
	t.Parallel()

	manifest, probe, evidence := validReconcileTriple(t)
	evidence = append(evidence, evidence[0], evidence[1])
	admitted, err := Reconcile(manifest, probe, evidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(admitted.EvidenceIDs) != 2 {
		t.Errorf("EvidenceIDs = %v, want 2 deduplicated IDs", admitted.EvidenceIDs)
	}
}

// TestReconcileRefusesExecutableSubstitution pins the digest-identity arm:
// a probe whose executable digest differs from the manifest member is
// untrusted, even though both documents parse (each is valid against its
// own kind rule). Parse cannot see across documents, so this arm is
// reachable only here and in production admission.
func TestReconcileRefusesExecutableSubstitution(t *testing.T) {
	t.Parallel()

	manifest, probe, evidence := validExternalPinTriple(t)
	probe.ExecutableDigest = testPinDigest(0xE6)
	_, err := Reconcile(manifest, probe, evidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	requirePinRefusal(t, err, CodeUntrusted, "executable substitution")
}

// TestParsedClaimSlicesShareNoRegistryBacking pins the interior-isolation
// half of the closed capability registry. Claim DependentOperations and
// EvidenceRequirements are the only exported slices on the new surface
// that could alias the package-wide capabilityRegistry rows, so the pin
// scribbles on every parsed claim slice and then re-reads the interior:
// the rows themselves, a registry-derived query, and a fresh admission.
// A parseClaim that returned the row slices directly would rewrite the
// process-wide registry here while every code-only test stayed green.
func TestParsedClaimSlicesShareNoRegistryBacking(t *testing.T) {
	t.Parallel()

	manifest, probe, _ := validReconcileTriple(t)

	before := make(map[string]capabilityRow, len(capabilityRegistry))
	for capability, row := range capabilityRegistry {
		before[capability] = capabilityRow{
			generationVariable:   row.generationVariable,
			dependentOperations:  append([]string{}, row.dependentOperations...),
			evidenceRequirements: append([]string{}, row.evidenceRequirements...),
		}
	}
	beforeCreate, err := CapabilitiesForOperation("create")
	if err != nil {
		t.Fatalf("CapabilitiesForOperation(create) error = %v", err)
	}
	sort.Strings(beforeCreate)

	poison := func(claims []Claim) {
		for index := range claims {
			for element := range claims[index].DependentOperations {
				claims[index].DependentOperations[element] = "REVIEW-POISON"
			}
			for element := range claims[index].EvidenceRequirements {
				claims[index].EvidenceRequirements[element] = "REVIEW-POISON"
			}
		}
	}
	poison(manifest.StaticCapabilityClaims)
	poison(probe.CapabilityClaims)

	for capability, row := range capabilityRegistry {
		want := before[capability]
		if row.generationVariable != want.generationVariable ||
			!equalStrings(row.dependentOperations, want.dependentOperations) ||
			!equalStrings(row.evidenceRequirements, want.evidenceRequirements) {
			t.Errorf("capabilityRegistry[%q] = %+v after claim mutation, want %+v", capability, row, want)
		}
	}
	afterCreate, err := CapabilitiesForOperation("create")
	if err != nil {
		t.Fatalf("CapabilitiesForOperation(create) error = %v", err)
	}
	sort.Strings(afterCreate)
	if !equalStrings(afterCreate, beforeCreate) {
		t.Errorf("CapabilitiesForOperation(create) = %v after claim mutation, want %v", afterCreate, beforeCreate)
	}

	freshManifest, freshProbe, freshEvidence := validReconcileTriple(t)
	admitted, err := Reconcile(freshManifest, freshProbe, freshEvidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	if err != nil {
		t.Fatalf("Reconcile() after claim mutation error = %v", err)
	}
	if len(admitted.Capabilities) != 2 || !admitted.Has("graceful_stop") || !admitted.Has("headless_creation") {
		t.Errorf("Admitted = %v after claim mutation, want graceful_stop and headless_creation", admitted.Capabilities)
	}
}

// validExternalPinTriple is the external-kind variant of the pin triple:
// local_program with a digest, one static realm claim echoed by probe,
// proven by realm evidence.
func validExternalPinTriple(t *testing.T) (Manifest, Probe, []Evidence) {
	t.Helper()

	generation, err := GenerationDigest("generation-alpha")
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	digest := testPinDigest(0xE5)
	fixture := testPinDigest(0xF1)
	issuer := testPinDigest(0x1D)

	manifest := map[string]any{
		"schema":                 SchemaManifest,
		"schema_version":         SchemaVersion100,
		"manifest_id":            "",
		"terminal_backend_id":    "example.term",
		"implementation_version": "2.1.0",
		"protocol_versions":      []any{"1.0.0"},
		"platforms":              []any{"linux"},
		"implementation_kind":    "local_program",
		"executable_digest":      digest,
		"static_capability_claims": []any{
			pinRealmClaim(OriginStatic),
		},
		"conformance_fixture_id": fixture,
		"extensions":             map[string]any{},
	}
	evidenceObject := map[string]any{
		"schema":                     SchemaCapabilityEvidence,
		"schema_version":             SchemaVersion100,
		"evidence_id":                "",
		"terminal_backend_id":        "example.term",
		"implementation_version":     "2.1.0",
		"protocol_version":           "1.0.0",
		"backend_generation_digest":  generation,
		"capability":                 credentialRealmCapability,
		"value":                      true,
		"platform":                   "linux",
		"os_version":                 "14.5",
		"conformance_fixture_id":     fixture,
		"observed_at":                "2025-06-01T00:00:00.000Z",
		"expires_at":                 "2027-06-01T00:00:00.000Z",
		"issuer":                     IssuerLocalProbe,
		"issuer_id":                  issuer,
		"attestation_signature":      "rsa-sha256:AA==",
		"facts":                      []any{"fixture_passed", "provider_auth_passed", "runtime_probe_passed", "sentinel_passed"},
		"terminal_binding_id":        digest,
		"provider_id":                "codex",
		"provider_build":             "1.2.3",
		"sentinel_result":            "passed",
		"provider_auth_smoke_result": "passed",
		"extensions":                 map[string]any{},
	}
	stampPinIdentity(t, evidenceObject, "evidence_id")
	parsedEvidence, err := ParseEvidence(mustPinJSON(t, evidenceObject))
	if err != nil {
		t.Fatalf("ParseEvidence() error = %v", err)
	}
	probe := map[string]any{
		"schema":                    SchemaProbe,
		"schema_version":            SchemaVersion100,
		"probe_id":                  "",
		"terminal_backend_id":       "example.term",
		"implementation_version":    "2.1.0",
		"protocol_version":          "1.0.0",
		"implementation_kind":       "local_program",
		"executable_digest":         digest,
		"platform":                  "linux",
		"os_version":                "14.5",
		"availability":              AvailabilityAvailable,
		"backend_generation_digest": generation,
		"capability_claims": []any{
			pinRealmClaim(OriginProbed),
		},
		"evidence_ids": []any{parsedEvidence.EvidenceID},
		"probed_at":    "2026-01-15T12:00:00.000Z",
		"extensions":   map[string]any{},
	}
	stampPinIdentity(t, manifest, "manifest_id")
	stampPinIdentity(t, probe, "probe_id")
	parsedManifest, err := ParseManifest(mustPinJSON(t, manifest))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	parsedProbe, err := ParseProbe(mustPinJSON(t, probe))
	if err != nil {
		t.Fatalf("ParseProbe() error = %v", err)
	}
	return parsedManifest, parsedProbe, []Evidence{parsedEvidence}
}

// pinRealmClaim returns one credential-realm claim map.
func pinRealmClaim(origin string) map[string]any {
	return map[string]any{
		"capability":            credentialRealmCapability,
		"origin":                origin,
		"value":                 true,
		"generation_variable":   true,
		"dependent_operations":  []any{"create", "restore"},
		"evidence_requirements": []any{"conformance_fixture", "credential_sentinel", "provider_auth_smoke", "runtime_probe"},
	}
}

// TestReconcileAdmitsPinTriple proves the unmutated white-box triple is
// coherent: the pins above fail on their mutation, not on a broken base.
func TestReconcileAdmitsPinTriple(t *testing.T) {
	t.Parallel()

	manifest, probe, evidence := validReconcileTriple(t)
	admitted, err := Reconcile(manifest, probe, evidence, "generation-alpha", testAdmissionTime(), pinVerifier)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(admitted.Capabilities) != 2 || !admitted.Has("graceful_stop") || !admitted.Has("headless_creation") {
		t.Errorf("Admitted = %v, want graceful_stop and headless_creation", admitted.Capabilities)
	}
}

// validReconcileTriple parses one coherent triple: a manifest with one
// static claim, a probe with a same-value override plus one probed
// addition, and the two evidences proving the true claims.
func validReconcileTriple(t *testing.T) (Manifest, Probe, []Evidence) {
	t.Helper()

	generation, err := GenerationDigest("generation-alpha")
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	fixture := testPinDigest(0xF1)
	issuer := testPinDigest(0x1D)

	manifest := pinManifest(fixture)
	evidenceMaps := []map[string]any{
		pinEvidence("graceful_stop", generation, fixture, issuer),
		pinEvidence("headless_creation", generation, fixture, issuer),
	}
	parsedEvidence := make([]Evidence, 0, len(evidenceMaps))
	var ids []string
	for _, object := range evidenceMaps {
		stampPinIdentity(t, object, "evidence_id")
		parsed, err := ParseEvidence(mustPinJSON(t, object))
		if err != nil {
			t.Fatalf("ParseEvidence() error = %v", err)
		}
		parsedEvidence = append(parsedEvidence, parsed)
		ids = append(ids, parsed.EvidenceID)
	}
	sort.Strings(ids)

	probe := pinProbe(generation, ids)
	stampPinIdentity(t, manifest, "manifest_id")
	stampPinIdentity(t, probe, "probe_id")
	parsedManifest, err := ParseManifest(mustPinJSON(t, manifest))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	parsedProbe, err := ParseProbe(mustPinJSON(t, probe))
	if err != nil {
		t.Fatalf("ParseProbe() error = %v", err)
	}
	return parsedManifest, parsedProbe, parsedEvidence
}

// pinManifest returns the white-box manifest document map with a blank ID.
func pinManifest(fixture string) map[string]any {
	return map[string]any{
		"schema":                 SchemaManifest,
		"schema_version":         SchemaVersion100,
		"manifest_id":            "",
		"terminal_backend_id":    "example.term",
		"implementation_version": "2.1.0",
		"protocol_versions":      []any{"1.0.0"},
		"platforms":              []any{"linux"},
		"implementation_kind":    "builtin_go",
		"executable_digest":      nil,
		"static_capability_claims": []any{
			pinClaim("headless_creation", OriginStatic, true),
		},
		"conformance_fixture_id": fixture,
		"extensions":             map[string]any{},
	}
}

// pinProbe returns the white-box probe document map with a blank ID.
func pinProbe(generation string, ids []string) map[string]any {
	asValues := make([]any, 0, len(ids))
	for _, id := range ids {
		asValues = append(asValues, id)
	}
	return map[string]any{
		"schema":                    SchemaProbe,
		"schema_version":            SchemaVersion100,
		"probe_id":                  "",
		"terminal_backend_id":       "example.term",
		"implementation_version":    "2.1.0",
		"protocol_version":          "1.0.0",
		"implementation_kind":       "builtin_go",
		"executable_digest":         nil,
		"platform":                  "linux",
		"os_version":                "14.5",
		"availability":              AvailabilityAvailable,
		"backend_generation_digest": generation,
		"capability_claims": []any{
			pinClaim("graceful_stop", OriginProbed, true),
			pinClaim("headless_creation", OriginProbed, true),
		},
		"evidence_ids": asValues,
		"probed_at":    "2026-01-15T12:00:00.000Z",
		"extensions":   map[string]any{},
	}
}

// pinEvidence returns one white-box evidence document map with a blank ID
// and a placeholder signature the pin verifier accepts unconditionally.
func pinEvidence(capability, generation, fixture, issuer string) map[string]any {
	return map[string]any{
		"schema":                     SchemaCapabilityEvidence,
		"schema_version":             SchemaVersion100,
		"evidence_id":                "",
		"terminal_backend_id":        "example.term",
		"implementation_version":     "2.1.0",
		"protocol_version":           "1.0.0",
		"backend_generation_digest":  generation,
		"capability":                 capability,
		"value":                      true,
		"platform":                   "linux",
		"os_version":                 "14.5",
		"conformance_fixture_id":     fixture,
		"observed_at":                "2025-06-01T00:00:00.000Z",
		"expires_at":                 "2027-06-01T00:00:00.000Z",
		"issuer":                     IssuerLocalProbe,
		"issuer_id":                  issuer,
		"attestation_signature":      "rsa-sha256:AA==",
		"facts":                      []any{"fixture_passed", "runtime_probe_passed"},
		"terminal_binding_id":        nil,
		"provider_id":                nil,
		"provider_build":             nil,
		"sentinel_result":            nil,
		"provider_auth_smoke_result": nil,
		"extensions":                 map[string]any{},
	}
}

// pinClaim returns one white-box claim map with exact registry-row members.
func pinClaim(capability, origin string, value bool) map[string]any {
	rows := map[string]map[string]any{
		"headless_creation": {
			"generation_variable":   true,
			"dependent_operations":  []any{"create"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		},
		"graceful_stop": {
			"generation_variable":   true,
			"dependent_operations":  []any{"request-stop"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		},
	}
	row := rows[capability]
	return map[string]any{
		"capability":            capability,
		"origin":                origin,
		"value":                 value,
		"generation_variable":   row["generation_variable"],
		"dependent_operations":  row["dependent_operations"],
		"evidence_requirements": row["evidence_requirements"],
	}
}

// stampPinIdentity stamps the recomputed omit-self identity on a document.
func stampPinIdentity(t *testing.T, object map[string]any, selfField string) {
	t.Helper()

	identity, err := objectIdentity(object, selfField)
	if err != nil {
		t.Fatalf("objectIdentity() error = %v", err)
	}
	object[selfField] = identity
}

// mustPinJSON encodes a fixture map to JSON bytes.
func mustPinJSON(t *testing.T, object map[string]any) []byte {
	t.Helper()

	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

// testPinDigest mints a deterministic fake digest for white-box fixtures.
func testPinDigest(seed byte) string {
	const hexdigits = "0123456789abcdef"
	var digits [64]byte
	for index := range digits {
		digits[index] = hexdigits[(int(seed)+index)%16]
	}
	return "sha256:" + string(digits[:])
}

// testAdmissionTime is the fixed admission instant for white-box pins.
func testAdmissionTime() time.Time {
	return time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)
}
