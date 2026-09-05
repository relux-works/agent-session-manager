// Black-box contract tests for Manifest/Probe/Capability-Evidence admission.
//
// Every fixture is built as a JSON member map by an independent test-side
// transcription of the Section 4.B/4.D recipe (own identity computation, own
// signature construction over the documented byte recipe), so agreement with
// production proves byte-exact conformance rather than shared-code
// coincidence. Negative cases mutate exactly one arm, re-finalize the
// document identity (and re-sign evidence when an unsigned field moves), and
// assert the exact wire code at the exact gate.
package terminalbackend_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gowebpki/jcs"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// requireRefusal asserts err is a registry refusal carrying exactly
// wantCode with the exact static wantDetail clause. Code-only assertions
// cannot hold a gate: ~110 of the 121 refusal arms share one wire code, so
// a deleted gate slides its fixture onto a lower arm and a code-only test
// stays green. Naming the clause makes the gate deletion fail.
func requireRefusal(t *testing.T, err error, wantCode, wantDetail string) {
	t.Helper()

	var refusal *terminalbackend.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("error = %v, want *Error with code %q at %q", err, wantCode, wantDetail)
	}
	if refusal.Code != wantCode || refusal.Detail != wantDetail {
		t.Errorf("refusal = %q at %q, want %q at %q", refusal.Code, refusal.Detail, wantCode, wantDetail)
	}
}

const (
	codeMismatch           = "terminal_backend_manifest_probe_mismatch"
	codeCapabilityUnproven = "terminal_backend_capability_unproven"
	codeIntegrityFailure   = "terminal_backend_integrity_failure"
	codeUntrusted          = "terminal_backend_untrusted"
	codeDrift              = "terminal_backend_implementation_drift"
	codeStaleGeneration    = "terminal_backend_stale_generation"
	codeNotFound           = "terminal_backend_not_found"
)

// testDigest mints a deterministic fake sha256: digest from one seed byte.
func testSeedDigest(seed byte) string {
	return "sha256:" + strings.Repeat(fmt.Sprintf("%02x", seed), 32)
}

// testIdentity computes the Section 4.B omit-self identity with the test's
// own recipe implementation: JCS bytes of the object with the self field
// omitted, SHA-256, lowercase sha256: prefix.
func testIdentity(t *testing.T, object map[string]any, selfField string) string {
	t.Helper()

	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != selfField {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		t.Fatalf("marshal omit-self object: %v", err)
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		t.Fatalf("canonicalize omit-self object: %v", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// testEvidenceMessage builds the exact bytes an attestation signs per
// Section 4.D: ASCII domain, one zero byte, JCS of the evidence object with
// exactly evidence_id and attestation_signature omitted.
func testEvidenceMessage(t *testing.T, object map[string]any) []byte {
	t.Helper()

	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != "evidence_id" && name != "attestation_signature" {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		t.Fatalf("marshal unsigned evidence: %v", err)
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		t.Fatalf("canonicalize unsigned evidence: %v", err)
	}
	message := append([]byte("ax-terminal-capability-evidence-v1"), 0x00)
	return append(message, canonical...)
}

// testKey generates one RSA attestation key for the test run.
func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate attestation key: %v", err)
	}
	return key
}

// testSignBody signs the Section 4.D message for object with key and writes
// the attestation_signature member.
func testSignBody(t *testing.T, key *rsa.PrivateKey, object map[string]any) {
	t.Helper()

	message := testEvidenceMessage(t, object)
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign evidence: %v", err)
	}
	object["attestation_signature"] = "rsa-sha256:" + base64.StdEncoding.EncodeToString(signature)
}

// finalizeEvidence signs object with key and stamps its recomputed
// evidence_id, yielding a fully valid evidence document map.
func finalizeEvidence(t *testing.T, key *rsa.PrivateKey, object map[string]any) {
	t.Helper()

	testSignBody(t, key, object)
	object["evidence_id"] = testIdentity(t, object, "evidence_id")
}

// finalizeDocument stamps the recomputed omit-self identity on a manifest
// or probe document map.
func finalizeDocument(t *testing.T, object map[string]any, selfField string) {
	t.Helper()

	object[selfField] = testIdentity(t, object, selfField)
}

// mustMarshal encodes a fixture map to JSON bytes.
func mustMarshal(t *testing.T, object map[string]any) []byte {
	t.Helper()

	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

// testVerifier authenticates evidence through one trusted key: it verifies
// the RSA signature and fails on anything else, mirroring the production
// contract that a caller-supplied issuer ID alone is never authentication.
func testVerifier(key *rsa.PrivateKey) terminalbackend.SignatureVerifier {
	return func(issuerID string, message, signature []byte) error {
		digest := sha256.Sum256(message)
		return rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature)
	}
}

// claimMap transcribes one registry-row claim. The row members are copied
// from the Section 4.D table, not from production, so row drift fails.
func claimMap(capability, origin string, value bool) map[string]any {
	rows := map[string]map[string]any{
		"durable_disconnect": {
			"generation_variable":   false,
			"dependent_operations":  []any{"create", "status"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		},
		"local_attach": {
			"generation_variable":   true,
			"dependent_operations":  []any{"attach"},
			"evidence_requirements": []any{"conformance_fixture", "policy_authorization", "runtime_probe"},
		},
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
		"credential_capable_execution_realm": {
			"generation_variable":  true,
			"dependent_operations": []any{"create", "restore"},
			"evidence_requirements": []any{
				"conformance_fixture", "credential_sentinel", "provider_auth_smoke", "runtime_probe",
			},
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

// fixtureUniverse is one coherent admission world: a builtin manifest,
// its probe (static echo, same-value override, differing override, false
// override, probed addition), and the evidence set proving the true claims.
type fixtureUniverse struct {
	rawGeneration    string
	generationDigest string
	fixtureID        string
	issuerID         string
	now              time.Time
	key              *rsa.PrivateKey
	manifest         map[string]any
	probe            map[string]any
	evidence         map[string]any
	evidenceByCap    map[string]map[string]any
}

// testUniverse builds the builtin ax.tmux world. The manifest identity
// tuple matches New("2.1.0", ["1.0.0", "1.1.0"]) admission exactly, so the
// same fixtures drive the registry-bound AdmitProbe entry point.
func testUniverse(t *testing.T) *fixtureUniverse {
	t.Helper()

	universe := &fixtureUniverse{
		rawGeneration: "generation-alpha",
		fixtureID:     testSeedDigest(0xF1),
		issuerID:      testSeedDigest(0x1D),
		now:           time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC),
		key:           testKey(t),
		evidenceByCap: make(map[string]map[string]any),
	}
	digest, err := terminalbackend.GenerationDigest(universe.rawGeneration)
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	universe.generationDigest = digest

	universe.manifest = map[string]any{
		"schema":                 terminalbackend.SchemaManifest,
		"schema_version":         "1.0.0",
		"manifest_id":            "",
		"terminal_backend_id":    terminalbackend.BuiltinTmux,
		"implementation_version": "2.1.0",
		"protocol_versions":      []any{"1.0.0", "1.1.0"},
		"platforms":              []any{"linux", "macos", "wsl2"},
		"implementation_kind":    "builtin_go",
		"executable_digest":      nil,
		"static_capability_claims": []any{
			claimMap("durable_disconnect", "static", true),
			claimMap("headless_creation", "static", true),
			claimMap("local_attach", "static", true),
		},
		"conformance_fixture_id": universe.fixtureID,
		"extensions":             map[string]any{},
	}
	finalizeDocument(t, universe.manifest, "manifest_id")

	universe.probe = map[string]any{
		"schema":                    terminalbackend.SchemaProbe,
		"schema_version":            "1.0.0",
		"probe_id":                  "",
		"terminal_backend_id":       terminalbackend.BuiltinTmux,
		"implementation_version":    "2.1.0",
		"protocol_version":          "1.1.0",
		"implementation_kind":       "builtin_go",
		"executable_digest":         nil,
		"platform":                  "linux",
		"os_version":                "14.5",
		"availability":              "available",
		"backend_generation_digest": universe.generationDigest,
		"capability_claims": []any{
			claimMap("durable_disconnect", "static", true),
			claimMap("graceful_stop", "probed", true),
			claimMap("headless_creation", "probed", true),
			claimMap("local_attach", "probed", false),
		},
		"evidence_ids": []any{},
		"probed_at":    "2026-01-15T12:00:00.000Z",
		"extensions":   map[string]any{},
	}

	for _, capability := range []string{"durable_disconnect", "graceful_stop", "headless_creation"} {
		object := universe.evidenceMap(capability, []any{"fixture_passed", "runtime_probe_passed"})
		finalizeEvidence(t, universe.key, object)
		universe.evidenceByCap[capability] = object
	}
	ids := make([]string, 0, len(universe.evidenceByCap))
	for _, object := range universe.evidenceByCap {
		ids = append(ids, object["evidence_id"].(string))
	}
	sort.Strings(ids)
	asValues := make([]any, 0, len(ids))
	for _, id := range ids {
		asValues = append(asValues, id)
	}
	universe.probe["evidence_ids"] = asValues
	finalizeDocument(t, universe.probe, "probe_id")

	return universe
}

// evidenceMap builds one unsigned evidence document map for capability with
// the given facts. Realm members stay null: realm evidence has its own
// builder in the external-kind universe test.
func (universe *fixtureUniverse) evidenceMap(capability string, facts []any) map[string]any {
	return map[string]any{
		"schema":                     terminalbackend.SchemaCapabilityEvidence,
		"schema_version":             "1.0.0",
		"evidence_id":                "",
		"terminal_backend_id":        terminalbackend.BuiltinTmux,
		"implementation_version":     "2.1.0",
		"protocol_version":           "1.1.0",
		"backend_generation_digest":  universe.generationDigest,
		"capability":                 capability,
		"value":                      true,
		"platform":                   "linux",
		"os_version":                 "14.5",
		"conformance_fixture_id":     universe.fixtureID,
		"observed_at":                "2025-06-01T00:00:00.000Z",
		"expires_at":                 "2027-06-01T00:00:00.000Z",
		"issuer":                     terminalbackend.IssuerLocalProbe,
		"issuer_id":                  universe.issuerID,
		"attestation_signature":      "",
		"facts":                      facts,
		"terminal_binding_id":        nil,
		"provider_id":                nil,
		"provider_build":             nil,
		"sentinel_result":            nil,
		"provider_auth_smoke_result": nil,
		"extensions":                 map[string]any{},
	}
}

// parsedUniverse parses the universe documents through the production entry
// points, failing the test on any parse error.
func (universe *fixtureUniverse) parsed(t *testing.T) (terminalbackend.Manifest, terminalbackend.Probe, []terminalbackend.Evidence) {
	t.Helper()

	manifest, err := terminalbackend.ParseManifest(mustMarshal(t, universe.manifest))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	probe, err := terminalbackend.ParseProbe(mustMarshal(t, universe.probe))
	if err != nil {
		t.Fatalf("ParseProbe() error = %v", err)
	}
	evidence := make([]terminalbackend.Evidence, 0, len(universe.evidenceByCap))
	for _, capability := range []string{"durable_disconnect", "graceful_stop", "headless_creation"} {
		object, err := terminalbackend.ParseEvidence(mustMarshal(t, universe.evidenceByCap[capability]))
		if err != nil {
			t.Fatalf("ParseEvidence(%s) error = %v", capability, err)
		}
		evidence = append(evidence, object)
	}
	return manifest, probe, evidence
}

// cloneMap deep-copies a fixture map through JSON so mutations never leak
// between cases.
func cloneMap(t *testing.T, object map[string]any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal clone: %v", err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(raw, &cloned); err != nil {
		t.Fatalf("unmarshal clone: %v", err)
	}
	return cloned
}

// TestParseManifestAdmitsClosedFixture proves the positive gate is
// reachable and pins the typed projection of every member.
func TestParseManifestAdmitsClosedFixture(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	manifest, err := terminalbackend.ParseManifest(mustMarshal(t, universe.manifest))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if manifest.TerminalBackendID != terminalbackend.BuiltinTmux {
		t.Errorf("TerminalBackendID = %q", manifest.TerminalBackendID)
	}
	if manifest.ImplementationVersion != "2.1.0" {
		t.Errorf("ImplementationVersion = %q", manifest.ImplementationVersion)
	}
	if len(manifest.ProtocolVersions) != 2 || manifest.ProtocolVersions[1] != "1.1.0" {
		t.Errorf("ProtocolVersions = %v", manifest.ProtocolVersions)
	}
	if len(manifest.Platforms) != 3 || manifest.Platforms[0].String() != "linux" {
		t.Errorf("Platforms = %v", manifest.Platforms)
	}
	if manifest.ImplementationKind != terminalbackend.KindBuiltinGo {
		t.Errorf("ImplementationKind = %q", manifest.ImplementationKind)
	}
	if manifest.ExecutableDigest != "" {
		t.Errorf("ExecutableDigest = %q, want null", manifest.ExecutableDigest)
	}
	if len(manifest.StaticCapabilityClaims) != 3 {
		t.Fatalf("StaticCapabilityClaims = %d, want 3", len(manifest.StaticCapabilityClaims))
	}
	if manifest.StaticCapabilityClaims[0].Capability != "durable_disconnect" {
		t.Errorf("first claim = %q", manifest.StaticCapabilityClaims[0].Capability)
	}
	if manifest.ConformanceFixtureID != universe.fixtureID {
		t.Errorf("ConformanceFixtureID = %q", manifest.ConformanceFixtureID)
	}
	if manifest.ManifestID != universe.manifest["manifest_id"] {
		t.Errorf("ManifestID = %q, want recomputed identity", manifest.ManifestID)
	}
}

// TestGenerationDigestGolden pins the domain-separated recipe: SHA-256 over
// UTF-8 "ax-terminal-backend-generation-v1" + NUL + raw generation bytes.
// The expected value was computed with an independent SHA-256 oracle, so a
// separator change fails here rather than hiding behind agreement.
func TestGenerationDigestGolden(t *testing.T) {
	t.Parallel()

	digest, err := terminalbackend.GenerationDigest("generation-alpha")
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	const want = "sha256:ad089bdcf1a41853068645fec63a0ec8e40b95c3e3a103bc326a70e8c479eab7"
	if digest != want {
		t.Errorf("GenerationDigest() = %q, want %q", digest, want)
	}
}

// TestGenerationDigestBounds proves the raw generation domain fails closed:
// empty (which would make every backend share one digest), over-long, and
// invalid UTF-8 derive nothing. string[1..256] bounds UTF-8 characters
// (SPEC.md:321), so the boundary is proved in two-byte runes as well as in
// ASCII bytes: 256 two-byte runes (512 bytes) derive a digest while 257 are
// refused.
func TestGenerationDigestBounds(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", strings.Repeat("g", 257), "ok\xffbad", strings.Repeat("g", 0), strings.Repeat("é", 257)} {
		_, err := terminalbackend.GenerationDigest(raw)
		requireRefusal(t, err, codeStaleGeneration, "backend_generation bound")
	}
	for _, raw := range []string{strings.Repeat("g", 256), strings.Repeat("é", 256)} {
		if _, err := terminalbackend.GenerationDigest(raw); err != nil {
			t.Errorf("GenerationDigest(%d runes) error = %v, want admission", len([]rune(raw)), err)
		}
	}
}

// TestMultibyteStringBounds proves string[1..256] counts UTF-8 characters
// (SPEC.md:321) at the document gates: a 256-rune os_version (512 bytes)
// and provider_build admit, while 257-rune values refuse with the document
// string bound. Admit cases re-seal the document identity (and re-sign
// evidence) after mutation so only the bound can decide.
func TestMultibyteStringBounds(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	universe.probe["os_version"] = strings.Repeat("é", 256)
	finalizeDocument(t, universe.probe, "probe_id")
	if _, err := terminalbackend.ParseProbe(mustMarshal(t, universe.probe)); err != nil {
		t.Errorf("ParseProbe(256-rune os_version) error = %v, want admission", err)
	}

	overlong := testUniverse(t)
	overlong.probe["os_version"] = strings.Repeat("é", 257)
	finalizeDocument(t, overlong.probe, "probe_id")
	_, err := terminalbackend.ParseProbe(mustMarshal(t, overlong.probe))
	requireRefusal(t, err, codeMismatch, "document string bound")

	realm := universe.evidenceMap("credential_capable_execution_realm",
		[]any{"fixture_passed", "provider_auth_passed", "runtime_probe_passed", "sentinel_passed"})
	realm["terminal_binding_id"] = testSeedDigest(0xB1)
	realm["provider_id"] = "codex"
	realm["provider_build"] = strings.Repeat("é", 256)
	realm["sentinel_result"] = "passed"
	realm["provider_auth_smoke_result"] = "passed"
	finalizeEvidence(t, universe.key, realm)
	if _, err := terminalbackend.ParseEvidence(mustMarshal(t, realm)); err != nil {
		t.Errorf("ParseEvidence(256-rune provider_build) error = %v, want admission", err)
	}

	realm["provider_build"] = strings.Repeat("é", 257)
	finalizeEvidence(t, universe.key, realm)
	_, err = terminalbackend.ParseEvidence(mustMarshal(t, realm))
	requireRefusal(t, err, codeMismatch, "document string bound")
}

// TestDocumentSurrogateEscapeRefused proves SPEC.md:289 at every production
// document entry: a lone-surrogate escape is refused with the document
// surrogate escape arm before any canonicalization, even though Go's
// encoding/json would silently decode it to U+FFFD and every later member
// check would then see valid UTF-8. The injection mutates the marshaled
// bytes (not the map) so the escape reaches the raw-byte scan; the broken
// identity that results is unreachable behind the decode gate.
func TestDocumentSurrogateEscapeRefused(t *testing.T) {
	t.Parallel()

	// The lone-surrogate escape is assembled from byte values: no literal
	// in this file spells the escape directly.
	slash := string([]byte{92})
	inject := func(raw []byte, anchor, value string) []byte {
		old := []byte(`"` + anchor + `"` + `:` + `"` + value + `"`)
		escaped := append([]byte(`"`+anchor+`"`+`:`+`"`), slash...)
		escaped = append(escaped, "ud800abc"...)
		escaped = append(escaped, '"')
		return bytes.Replace(raw, old, escaped, 1)
	}

	universe := testUniverse(t)
	_, err := terminalbackend.ParseProbe(inject(mustMarshal(t, universe.probe), "os_version", "14.5"))
	requireRefusal(t, err, codeMismatch, "document surrogate escape")

	_, err = terminalbackend.ParseManifest(inject(mustMarshal(t, universe.manifest), "implementation_kind", "builtin_go"))
	requireRefusal(t, err, codeMismatch, "document surrogate escape")

	_, err = terminalbackend.ParseEvidence(inject(mustMarshal(t, universe.evidenceByCap["durable_disconnect"]), "os_version", "14.5"))
	requireRefusal(t, err, codeMismatch, "document surrogate escape")
}

// TestDocumentWTF8SurrogateRefused proves the raw road to the same
// substitution is closed at every production document entry: a lone
// surrogate arriving as raw WTF-8 (CESU-8) bytes ED A0 80 rather than as an
// escape is refused before any canonicalization, even though Go's
// encoding/json would silently decode those bytes to U+FFFD. The encoding
// arm fires first (the bytes are not valid UTF-8); the surrogate gate sees
// the same input independently, pinned white-box by
// TestSurrogateGateAgreesWithCanonicalJSON, so the two refusals cannot
// disagree the way an escape-blind gate and canonicaljson once did.
func TestDocumentWTF8SurrogateRefused(t *testing.T) {
	t.Parallel()

	injectRaw := func(raw []byte, anchor, value string) []byte {
		old := []byte(`"` + anchor + `"` + `:` + `"` + value + `"`)
		injected := append([]byte(`"`+anchor+`"`+`:`+`"`), 0xed, 0xa0, 0x80)
		injected = append(injected, '"')
		return bytes.Replace(raw, old, injected, 1)
	}

	universe := testUniverse(t)
	_, err := terminalbackend.ParseProbe(injectRaw(mustMarshal(t, universe.probe), "os_version", "14.5"))
	requireRefusal(t, err, codeMismatch, "document encoding")

	_, err = terminalbackend.ParseManifest(injectRaw(mustMarshal(t, universe.manifest), "implementation_kind", "builtin_go"))
	requireRefusal(t, err, codeMismatch, "document encoding")

	_, err = terminalbackend.ParseEvidence(injectRaw(mustMarshal(t, universe.evidenceByCap["durable_disconnect"]), "os_version", "14.5"))
	requireRefusal(t, err, codeMismatch, "document encoding")
}

// TestReconcileAdmitsCoherentTuple is the positive admission: static echo,
// same-value override, differing override, false claim, and probed addition
// reconcile to exactly the true evidenced claims.
func TestReconcileAdmitsCoherentTuple(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	manifest, probe, evidence := universe.parsed(t)
	admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	want := []string{"durable_disconnect", "graceful_stop", "headless_creation"}
	if fmt.Sprint(admitted.Capabilities) != fmt.Sprint(want) {
		t.Errorf("Capabilities = %v, want %v", admitted.Capabilities, want)
	}
	if len(admitted.EvidenceIDs) != 3 {
		t.Errorf("EvidenceIDs = %v, want 3 proved IDs", admitted.EvidenceIDs)
	}
	if !admitted.Has("headless_creation") || admitted.Has("local_attach") {
		t.Errorf("Has() over %v misreports overrides", admitted.Capabilities)
	}
	if !admitted.HasOperation("create") || admitted.HasOperation("attach") {
		t.Errorf("HasOperation() over %v misreports dependencies", admitted.Capabilities)
	}
	if err := terminalbackend.CheckOperation("create", admitted); err != nil {
		t.Errorf("CheckOperation(create) error = %v, want admission", err)
	}
	requireRefusal(t, terminalbackend.CheckOperation("attach", admitted), codeCapabilityUnproven, "operation capability dependency")
}

// TestManifestDocumentRefusals narrows the closed-schema gate: each case
// widens exactly one member class and must fail at its own clause, not
// merely with the shared wire code.
func TestManifestDocumentRefusals(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(t *testing.T, object map[string]any)
		detail string
	}{
		"unknown member": {
			mutate: func(t *testing.T, object map[string]any) {
				object["sponsor"] = "evil"
			},
			detail: "document members",
		},
		"missing member": {
			mutate: func(t *testing.T, object map[string]any) {
				delete(object, "platforms")
			},
			detail: "document members",
		},
		"non-empty extensions": {
			mutate: func(t *testing.T, object map[string]any) {
				object["extensions"] = map[string]any{"v2": true}
			},
			detail: "document extensions",
		},
		"absent extensions": {
			mutate: func(t *testing.T, object map[string]any) {
				delete(object, "extensions")
			},
			detail: "document members",
		},
		"wrong schema": {
			mutate: func(t *testing.T, object map[string]any) {
				object["schema"] = "urn:ax:schema:terminal-backend-probe"
			},
			detail: "manifest schema",
		},
		"wrong version": {
			mutate: func(t *testing.T, object map[string]any) {
				object["schema_version"] = "2.0.0"
			},
			detail: "manifest schema version",
		},
		"unknown capability": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["capability"] = "teleportation"
			},
			detail: "capability vocabulary",
		},
		"probed origin in manifest": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["origin"] = "probed"
			},
			detail: "claim origin",
		},
		// A non-object claim element is refused by the shape guard
		// before member validation: narrowing the guard slides the
		// refusal onto the downstream members arm, so only this
		// case names the shape rule.
		"non-object claim": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0] = "durable_disconnect"
			},
			detail: "claim shape",
		},
		"flipped generation variable": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["generation_variable"] = true
			},
			detail: "capability registry binding",
		},
		"redefined operations": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["dependent_operations"] = []any{"create"}
			},
			detail: "capability registry binding",
		},
		"unknown operation in claim": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["dependent_operations"] = []any{"create", "launch"}
			},
			detail: "document vocabulary",
		},
		"empty operations": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["dependent_operations"] = []any{}
			},
			detail: "document list bound",
		},
		"unsorted operations": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["dependent_operations"] = []any{"status", "create"}
			},
			detail: "document ordering",
		},
		"unknown requirement": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				claims[0].(map[string]any)["evidence_requirements"] = []any{"conformance_fixture", "vibes"}
			},
			detail: "document vocabulary",
		},
		"duplicate claims": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				object["static_capability_claims"] = append(claims, claims[0])
			},
			detail: "claim ordering",
		},
		// An adjacent duplicate in sorted position is refused only by
		// the duplicate half of the ordering gate: narrowing >= to >
		// admits it, so this case holds that half.
		"adjacent duplicate claims": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				object["static_capability_claims"] = []any{claims[0], claims[1], claims[1], claims[2]}
			},
			detail: "claim ordering",
		},
		"unsorted claims": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				object["static_capability_claims"] = []any{claims[2], claims[1], claims[0]}
			},
			detail: "claim ordering",
		},
		"claim list bound": {
			mutate: func(t *testing.T, object map[string]any) {
				claims := object["static_capability_claims"].([]any)
				bloated := make([]any, 0, 17)
				for len(bloated) < 17 {
					bloated = append(bloated, claims...)
				}
				object["static_capability_claims"] = bloated[:17]
			},
			detail: "claim list bound",
		},
		"bad semver": {
			mutate: func(t *testing.T, object map[string]any) {
				object["implementation_version"] = "2.1"
			},
			detail: "document semver",
		},
		"protocol major 2": {
			mutate: func(t *testing.T, object map[string]any) {
				object["protocol_versions"] = []any{"1.0.0", "2.0.0"}
			},
			detail: "protocol versions major 1",
		},
		"unsorted protocols": {
			mutate: func(t *testing.T, object map[string]any) {
				object["protocol_versions"] = []any{"1.1.0", "1.0.0"}
			},
			detail: "protocol versions ordering",
		},
		"empty protocols": {
			mutate: func(t *testing.T, object map[string]any) {
				object["protocol_versions"] = []any{}
			},
			detail: "protocol versions bound",
		},
		"unknown platform": {
			mutate: func(t *testing.T, object map[string]any) {
				object["platforms"] = []any{"amigaos"}
			},
			detail: "platforms vocabulary",
		},
		"unsorted platforms": {
			mutate: func(t *testing.T, object map[string]any) {
				object["platforms"] = []any{"macos", "linux", "wsl2"}
			},
			detail: "platforms ordering",
		},
		"empty platforms": {
			mutate: func(t *testing.T, object map[string]any) {
				object["platforms"] = []any{}
			},
			detail: "platforms bound",
		},
		"unknown implementation kind": {
			mutate: func(t *testing.T, object map[string]any) {
				object["implementation_kind"] = "container_image"
			},
			detail: "manifest implementation kind",
		},
		"digest on builtin": {
			mutate: func(t *testing.T, object map[string]any) {
				object["executable_digest"] = testSeedDigest(0xAA)
			},
			detail: "manifest executable digest",
		},
		"bad digest": {
			mutate: func(t *testing.T, object map[string]any) {
				object["implementation_kind"] = "local_program"
				object["executable_digest"] = "not-a-digest"
			},
			detail: "document digest",
		},
		"reserved backend id": {
			mutate: func(t *testing.T, object map[string]any) {
				object["terminal_backend_id"] = "ax.evil"
			},
			detail: "manifest backend identity",
		},
		"bad fixture id": {
			mutate: func(t *testing.T, object map[string]any) {
				object["conformance_fixture_id"] = "sha256:zzz"
			},
			detail: "document digest",
		},
		"numeric member": {
			mutate: func(t *testing.T, object map[string]any) {
				object["implementation_version"] = float64(2)
			},
			detail: "document member type",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			universe := testUniverse(t)
			object := cloneMap(t, universe.manifest)
			tc.mutate(t, object)
			finalizeDocument(t, object, "manifest_id")
			_, err := terminalbackend.ParseManifest(mustMarshal(t, object))
			requireRefusal(t, err, codeMismatch, tc.detail)
		})
	}
}

// TestManifestEncodingRefusals proves malformed reads fail closed: syntax,
// duplicate, encoding, depth, shape, and identity failures are mismatch,
// never absence.
func TestManifestEncodingRefusals(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	valid := mustMarshal(t, universe.manifest)
	deep := strings.Repeat("[", 40) + strings.Repeat("]", 40)

	cases := map[string]struct {
		raw    []byte
		detail string
	}{
		"empty":        {raw: []byte{}, detail: "document syntax"},
		"truncated":    {raw: valid[:len(valid)/2], detail: "document syntax"},
		"not json":     {raw: []byte("manifest"), detail: "document syntax"},
		"top array":    {raw: []byte("[]"), detail: "document shape"},
		"top string":   {raw: []byte(`"manifest"`), detail: "document shape"},
		"trailing":     {raw: append(append([]byte{}, valid...), []byte(" {}")...), detail: "document trailing data"},
		"duplicate":    {raw: []byte(`{"schema":"urn:ax:schema:terminal-backend-manifest","schema":"urn:ax:schema:terminal-backend-manifest"}`), detail: "document duplicate member"},
		"invalid utf8": {raw: append(append([]byte{}, valid...), 0xff), detail: "document encoding"},
		"deep nesting": {raw: []byte(`{"schema":"x","nest":` + deep + `}`), detail: "document nesting"},
		"null":         {raw: []byte("null"), detail: "document shape"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := terminalbackend.ParseManifest(tc.raw)
			requireRefusal(t, err, codeMismatch, tc.detail)
		})
	}
}

// TestManifestIdentityMismatch proves the reader recomputes the ID: a valid
// shape with a foreign digest is refused.
func TestManifestIdentityMismatch(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	object := cloneMap(t, universe.manifest)
	object["manifest_id"] = testSeedDigest(0xDD)
	_, err := terminalbackend.ParseManifest(mustMarshal(t, object))
	requireRefusal(t, err, codeMismatch, "document identity binding")
}

// TestProbeDocumentRefusals narrows the probe closed-schema gate: the same
// schema, version, identity, kind, and digest classes the manifest parser
// pins, at the probe's own clauses, plus the probe-only members.
func TestProbeDocumentRefusals(t *testing.T) {
	t.Parallel()

	// Note: a static probe claim that merely differs from the manifest
	// parses by design; cross-document reconciliation owns that refusal.
	cases := map[string]struct {
		mutate func(t *testing.T, object map[string]any)
		detail string
	}{
		"wrong schema": {
			mutate: func(t *testing.T, object map[string]any) {
				object["schema"] = "urn:ax:schema:terminal-backend-manifest"
			},
			detail: "probe schema",
		},
		"wrong schema version": {
			mutate: func(t *testing.T, object map[string]any) {
				object["schema_version"] = "2.0.0"
			},
			detail: "probe schema version",
		},
		"reserved backend id": {
			mutate: func(t *testing.T, object map[string]any) {
				object["terminal_backend_id"] = "ax.evil"
			},
			detail: "probe backend identity",
		},
		"unknown implementation kind": {
			mutate: func(t *testing.T, object map[string]any) {
				object["implementation_kind"] = "container_image"
			},
			detail: "probe implementation kind",
		},
		"digest on builtin": {
			mutate: func(t *testing.T, object map[string]any) {
				object["executable_digest"] = testSeedDigest(0xAA)
			},
			detail: "probe executable digest",
		},
		"missing digest on external": {
			mutate: func(t *testing.T, object map[string]any) {
				object["implementation_kind"] = "local_program"
				object["executable_digest"] = nil
			},
			detail: "probe executable digest",
		},
		"unknown availability": {
			mutate: func(t *testing.T, object map[string]any) {
				object["availability"] = "sometimes"
			},
			detail: "probe availability",
		},
		"bad os version": {
			mutate: func(t *testing.T, object map[string]any) {
				object["os_version"] = ""
			},
			detail: "document string bound",
		},
		"bad platform": {
			mutate: func(t *testing.T, object map[string]any) {
				object["platform"] = "plan9"
			},
			detail: "probe platform",
		},
		"protocol major 2": {
			mutate: func(t *testing.T, object map[string]any) {
				object["protocol_version"] = "2.0.0"
			},
			detail: "probe protocol major 1",
		},
		"unsorted evidence ids": {
			mutate: func(t *testing.T, object map[string]any) {
				ids := object["evidence_ids"].([]any)
				for left, right := 0, len(ids)-1; left < right; left, right = left+1, right-1 {
					ids[left], ids[right] = ids[right], ids[left]
				}
			},
			detail: "document ordering",
		},
		"duplicate evidence ids": {
			mutate: func(t *testing.T, object map[string]any) {
				ids := object["evidence_ids"].([]any)
				object["evidence_ids"] = append(ids, ids[0])
			},
			detail: "document ordering",
		},
		// An adjacent duplicate in sorted position is refused only by
		// the duplicate half of the ordering gate: narrowing >= to >
		// admits it, so this case holds that half.
		"adjacent duplicate evidence ids": {
			mutate: func(t *testing.T, object map[string]any) {
				ids := object["evidence_ids"].([]any)
				object["evidence_ids"] = []any{ids[0], ids[1], ids[1], ids[2]}
			},
			detail: "document ordering",
		},
		"evidence list bound": {
			mutate: func(t *testing.T, object map[string]any) {
				ids := make([]any, 0, 257)
				for seed := 0; seed < 257; seed++ {
					ids = append(ids, testSeedDigest(byte(seed)))
				}
				object["evidence_ids"] = ids
			},
			detail: "evidence list bound",
		},
		"malformed evidence id": {
			mutate: func(t *testing.T, object map[string]any) {
				object["evidence_ids"] = []any{"nope"}
			},
			detail: "document digest",
		},
		"bad timestamp": {
			mutate: func(t *testing.T, object map[string]any) {
				object["probed_at"] = "2026-01-15 12:00:00"
			},
			detail: "document timestamp",
		},
		"timestamp without fraction": {
			mutate: func(t *testing.T, object map[string]any) {
				object["probed_at"] = "2026-01-15T12:00:00Z"
			},
			detail: "document timestamp",
		},
		"non-empty extensions": {
			mutate: func(t *testing.T, object map[string]any) {
				object["extensions"] = map[string]any{"x": 1}
			},
			detail: "document extensions",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			universe := testUniverse(t)
			object := cloneMap(t, universe.probe)
			tc.mutate(t, object)
			finalizeDocument(t, object, "probe_id")
			_, err := terminalbackend.ParseProbe(mustMarshal(t, object))
			requireRefusal(t, err, codeMismatch, tc.detail)
		})
	}
}

// TestEvidenceDocumentRefusals narrows the evidence closed-schema gate: the
// same schema, version, identity, and protocol classes the manifest parser
// pins, at the evidence's own clauses, plus the value, fact, issuer,
// signature, and realm members. The signature is re-applied after every
// unsigned-field mutation so each case reaches its own arm instead of the
// attestation check.
func TestEvidenceDocumentRefusals(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(t *testing.T, universe *fixtureUniverse, object map[string]any)
		detail string
	}{
		"wrong schema": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["schema"] = "urn:ax:schema:terminal-backend-manifest"
			},
			detail: "evidence schema",
		},
		"wrong schema version": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["schema_version"] = "2.0.0"
			},
			detail: "evidence schema version",
		},
		"reserved backend id": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["terminal_backend_id"] = "ax.evil"
			},
			detail: "evidence backend identity",
		},
		"protocol major 2": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["protocol_version"] = "2.0.0"
			},
			detail: "evidence protocol major 1",
		},
		// The platform vocabulary gate in the one parser the
		// closed-schema rework did not extend: only this arm names
		// the platform rule, so narrowing it leaves no refusal.
		"bad platform": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["platform"] = "plan9"
			},
			detail: "evidence platform",
		},
		"false value": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["value"] = false
			},
			detail: "evidence value",
		},
		"string value": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["value"] = "true"
			},
			detail: "evidence value",
		},
		"expires equal observed": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["expires_at"] = object["observed_at"]
			},
			detail: "evidence expiry",
		},
		"expires before observed": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["expires_at"] = "2024-01-01T00:00:00.000Z"
			},
			detail: "evidence expiry",
		},
		"malformed observed_at": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["observed_at"] = "2026-01-15 12:00:00"
			},
			detail: "document timestamp",
		},
		"malformed expires_at": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["expires_at"] = "not-a-timestamp"
			},
			detail: "document timestamp",
		},
		"unknown fact": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["facts"] = []any{"fixture_passed", "good_vibes"}
			},
			detail: "document vocabulary",
		},
		"unsorted facts": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["facts"] = []any{"runtime_probe_passed", "fixture_passed"}
			},
			detail: "document ordering",
		},
		"empty facts": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["facts"] = []any{}
			},
			detail: "document list bound",
		},
		"unknown issuer": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["issuer"] = "self"
			},
			detail: "evidence issuer",
		},
		"bad signature scheme": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["attestation_signature"] = "sha256:abcd"
			},
			detail: "evidence signature scheme",
		},
		// An unprefixed but well-formed Base64 body is admitted when
		// the scheme-prefix half is removed, so this case holds it.
		"unprefixed valid base64 signature": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["attestation_signature"] = "AA=="
			},
			detail: "evidence signature scheme",
		},
		"bad signature encoding": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["attestation_signature"] = "rsa-sha256:!!!"
			},
			detail: "evidence signature encoding",
		},
		"realm members on plain claim": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["terminal_binding_id"] = testSeedDigest(0xB1)
				object["provider_id"] = "codex"
				object["provider_build"] = "1.0"
				object["sentinel_result"] = "passed"
				object["provider_auth_smoke_result"] = "passed"
			},
			detail: "evidence realm binding",
		},
		// A realm claim proved by realm-less evidence is refused by the
		// required-members half; only the inverse had a case before.
		"null realm members on realm claim": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["capability"] = "credential_capable_execution_realm"
			},
			detail: "evidence realm binding",
		},
		"bad provider id": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["capability"] = "credential_capable_execution_realm"
				object["terminal_binding_id"] = testSeedDigest(0xB1)
				object["provider_id"] = "Codex!!"
				object["provider_build"] = "1.0"
				object["sentinel_result"] = "passed"
				object["provider_auth_smoke_result"] = "passed"
			},
			detail: "evidence provider identity",
		},
		"half provider pair": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["capability"] = "credential_capable_execution_realm"
				object["terminal_binding_id"] = testSeedDigest(0xB1)
				object["provider_id"] = "codex"
				object["provider_build"] = nil
				object["sentinel_result"] = "passed"
				object["provider_auth_smoke_result"] = "passed"
			},
			detail: "evidence realm binding",
		},
		"failed sentinel literal": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["capability"] = "credential_capable_execution_realm"
				object["terminal_binding_id"] = testSeedDigest(0xB1)
				object["provider_id"] = "codex"
				object["provider_build"] = "1.0"
				object["sentinel_result"] = "failed"
				object["provider_auth_smoke_result"] = "passed"
			},
			detail: "evidence realm result",
		},
		"overlong os version": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["os_version"] = strings.Repeat("o", 257)
			},
			detail: "document string bound",
		},
		"unknown capability": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["capability"] = "teleportation"
			},
			detail: "capability vocabulary",
		},
		"numeric fact": {
			mutate: func(t *testing.T, universe *fixtureUniverse, object map[string]any) {
				object["facts"] = []any{"fixture_passed", float64(7)}
			},
			detail: "document member type",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			universe := testUniverse(t)
			object := cloneMap(t, universe.evidenceByCap["durable_disconnect"])
			tc.mutate(t, universe, object)
			// Only the identity is re-stamped: parsing never verifies
			// the attestation, so the original signature stays a valid
			// bystander and every case reaches its own structural arm.
			object["evidence_id"] = testIdentity(t, object, "evidence_id")
			_, err := terminalbackend.ParseEvidence(mustMarshal(t, object))
			requireRefusal(t, err, codeMismatch, tc.detail)
		})
	}
}

// reconcileFixture parses one possibly-mutated universe into typed values.
func reconcileFixture(t *testing.T, universe *fixtureUniverse) (terminalbackend.Manifest, terminalbackend.Probe, []terminalbackend.Evidence) {
	t.Helper()

	manifest, err := terminalbackend.ParseManifest(mustMarshal(t, universe.manifest))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	probe, err := terminalbackend.ParseProbe(mustMarshal(t, universe.probe))
	if err != nil {
		t.Fatalf("ParseProbe() error = %v", err)
	}
	// Every entry in the evidence set parses, including extras the case
	// added: unlisted or dangling objects are reconciliation refusals,
	// not parse failures.
	names := make([]string, 0, len(universe.evidenceByCap))
	for name := range universe.evidenceByCap {
		names = append(names, name)
	}
	sort.Strings(names)
	evidence := make([]terminalbackend.Evidence, 0, len(names))
	for _, name := range names {
		parsed, err := terminalbackend.ParseEvidence(mustMarshal(t, universe.evidenceByCap[name]))
		if err != nil {
			t.Fatalf("ParseEvidence(%s) error = %v", name, err)
		}
		evidence = append(evidence, parsed)
	}
	return manifest, probe, evidence
}

// TestReconcileRefusals proves every admission arm fails closed at its own
// clause. Each case moves exactly one rule's input and re-finalizes
// identities (and re-signs evidence when an unsigned field moves) so the
// failure lands on the intended arm — and the assertion names that arm, so
// deleting the gate fails the test instead of sliding onto a lower arm.
// Where a downstream arm could catch the same fixture, the fixture is
// narrowed until only the named rule can reject it.
func TestReconcileRefusals(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(t *testing.T, universe *fixtureUniverse)
		code   string
		detail string
	}{
		"version mismatch": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				universe.probe["implementation_version"] = "2.2.0"
				finalizeDocument(t, universe.probe, "probe_id")
			},
			code:   codeMismatch,
			detail: "probe manifest binding",
		},
		"kind mismatch": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				universe.probe["implementation_kind"] = "native_runtime"
				finalizeDocument(t, universe.probe, "probe_id")
			},
			code:   codeMismatch,
			detail: "probe manifest binding",
		},
		// The evidence moves with the probe protocol: without that, a
		// deleted membership gate slides onto the tuple binding and the
		// suite stays green. Narrowed, only the membership rule rejects.
		"protocol not member": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				universe.probe["protocol_version"] = "1.2.0"
				for _, object := range universe.evidenceByCap {
					object["protocol_version"] = "1.2.0"
					finalizeEvidence(t, universe.key, object)
				}
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "probe protocol membership",
		},
		"platform not member": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				universe.probe["platform"] = "windows"
				for _, object := range universe.evidenceByCap {
					object["platform"] = "windows"
					finalizeEvidence(t, universe.key, object)
				}
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "probe platform membership",
		},
		"stale generation": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {},
			code:   codeStaleGeneration,
			detail: "probe generation binding",
		},
		// The orphaned durable evidence is withdrawn with the claim:
		// without that, a deleted omission gate slides onto the claim
		// binding and the suite stays green. Narrowed, only the
		// omission rule rejects.
		"omitted manifest claim": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				claims := universe.probe["capability_claims"].([]any)
				universe.probe["capability_claims"] = claims[1:]
				delete(universe.evidenceByCap, "durable_disconnect")
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "probe omission of manifest claim",
		},
		// The orphaned durable evidence is withdrawn with the flipped
		// value: without that, a deleted echo gate slides onto the
		// claim binding. Narrowed, only the echo rule rejects.
		"static echo drift": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				claims := universe.probe["capability_claims"].([]any)
				claims[0].(map[string]any)["value"] = false
				delete(universe.evidenceByCap, "durable_disconnect")
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "probe static claim echo",
		},
		// A static-origin probe claim for a capability the manifest
		// never declared is self-minted: the manifest-to-probe
		// omission loop checks the opposite direction and cannot
		// catch it. The value stays true with its evidence intact,
		// so only this arm can reject; deleting it admits the
		// undeclared capability.
		"static claim without manifest": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				claims := universe.probe["capability_claims"].([]any)
				claims[1].(map[string]any)["origin"] = "static"
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "probe static claim without manifest",
		},
		"override of stable claim": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				claims := universe.probe["capability_claims"].([]any)
				claims[0].(map[string]any)["origin"] = "probed"
				finalizeDocument(t, universe.probe, "probe_id")
			},
			code:   codeMismatch,
			detail: "probe override of stable claim",
		},
		"true claim without evidence": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				delete(universe.evidenceByCap, "graceful_stop")
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence requirement coverage",
		},
		// The withdrawn claim is flipped false so coverage still holds
		// and only the ID-set rule can reject the stale listing: a bare
		// deletion lands on coverage instead, which already has a case.
		"dangling evidence id": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				claims := universe.probe["capability_claims"].([]any)
				claims[1].(map[string]any)["value"] = false
				delete(universe.evidenceByCap, "graceful_stop")
				finalizeDocument(t, universe.probe, "probe_id")
			},
			code:   codeMismatch,
			detail: "evidence id set binding",
		},
		"evidence id superset": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				ids := universe.probe["evidence_ids"].([]any)
				ids = append(ids, testSeedDigest(0xE1))
				names := make([]string, 0, len(ids))
				for _, id := range ids {
					names = append(names, id.(string))
				}
				sort.Strings(names)
				sorted := make([]any, 0, len(names))
				for _, id := range names {
					sorted = append(sorted, id)
				}
				universe.probe["evidence_ids"] = sorted
				finalizeDocument(t, universe.probe, "probe_id")
			},
			code:   codeMismatch,
			detail: "evidence id set binding",
		},
		"unreferenced evidence object": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				// A distinct usable object (supplementary fact changes
				// the bytes and the ID) that evidence_ids never lists.
				extra := universe.evidenceMap("durable_disconnect", []any{"fixture_passed", "runtime_probe_passed", "ui_absent"})
				finalizeEvidence(t, universe.key, extra)
				universe.evidenceByCap["durable-extra"] = extra
			},
			code:   codeMismatch,
			detail: "evidence id set binding",
		},
		// The false claim's evidence ID is listed on the probe: without
		// that, a deleted false-claim half slides onto the ID-set rule.
		// Narrowed, only the claim binding rejects.
		"evidence for false claim": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceMap("local_attach", []any{"fixture_passed", "policy_checked", "runtime_probe_passed"})
				finalizeEvidence(t, universe.key, object)
				universe.evidenceByCap["local_attach"] = object
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence claim binding",
		},
		"expired evidence": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["observed_at"] = "2024-01-01T00:00:00.000Z"
				object["expires_at"] = "2025-01-01T00:00:00.000Z"
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence liveness",
		},
		// The upper boundary instant itself: observed <= now ==
		// expires_at is expired (now < expires_at is strict). A gate
		// narrowed to now.After(expires) admits this row.
		"evidence expires at admission instant": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["expires_at"] = "2026-01-20T00:00:00.000Z"
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence liveness",
		},
		"future observed evidence": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["observed_at"] = "2027-01-01T00:00:00.000Z"
				object["expires_at"] = "2028-01-01T00:00:00.000Z"
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence liveness",
		},
		"wrong generation evidence": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["backend_generation_digest"] = testSeedDigest(0x99)
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence tuple binding",
		},
		"wrong fixture evidence": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["conformance_fixture_id"] = testSeedDigest(0x98)
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence tuple binding",
		},
		"insufficient facts": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["facts"] = []any{"fixture_passed"}
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence requirement coverage",
		},
		"supplementary facts satisfy nothing": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				object := universe.evidenceByCap["graceful_stop"]
				object["facts"] = []any{"prompt_absent", "ui_absent"}
				finalizeEvidence(t, universe.key, object)
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence requirement coverage",
		},
		"split facts across objects": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				first := universe.evidenceByCap["graceful_stop"]
				first["facts"] = []any{"fixture_passed"}
				finalizeEvidence(t, universe.key, first)
				second := universe.evidenceMap("graceful_stop", []any{"runtime_probe_passed"})
				finalizeEvidence(t, universe.key, second)
				universe.evidenceByCap["graceful-second"] = second
				finalizeProbeIDs(t, universe)
			},
			code:   codeMismatch,
			detail: "evidence requirement coverage",
		},
		// Note: same-ID differing-bytes evidence cannot pass parsing (the
		// ID is a hash of the bytes), so the conflict arm is pinned
		// white-box in manifest_pin_test.go instead.
		"wrong key signature": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {
				other := testKey(t)
				for _, object := range universe.evidenceByCap {
					delete(object, "attestation_signature")
					object["attestation_signature"] = ""
					testSignBody(t, other, object)
					object["evidence_id"] = testIdentity(t, object, "evidence_id")
				}
				finalizeProbeIDs(t, universe)
			},
			code:   codeIntegrityFailure,
			detail: "evidence attestation",
		},
		"nil verifier": {
			mutate: func(t *testing.T, universe *fixtureUniverse) {},
			code:   codeIntegrityFailure,
			detail: "evidence signature verifier",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			universe := testUniverse(t)
			tc.mutate(t, universe)
			manifest, probe, evidence := reconcileFixture(t, universe)
			verify := testVerifier(universe.key)
			if name == "nil verifier" {
				verify = nil
			}
			rawGeneration := universe.rawGeneration
			if name == "stale generation" {
				rawGeneration = "generation-beta"
			}
			_, err := terminalbackend.Reconcile(manifest, probe, evidence, rawGeneration, universe.now, verify)
			requireRefusal(t, err, tc.code, tc.detail)
		})
	}
}

// finalizeProbeIDs rebuilds evidence_ids from the current evidence set and
// re-stamps the probe identity.
func finalizeProbeIDs(t *testing.T, universe *fixtureUniverse) {
	t.Helper()

	ids := make([]string, 0, len(universe.evidenceByCap))
	for _, object := range universe.evidenceByCap {
		id, ok := object["evidence_id"].(string)
		if !ok || id == "" {
			t.Fatalf("evidence without identity in set")
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	asValues := make([]any, 0, len(ids))
	for _, id := range ids {
		asValues = append(asValues, id)
	}
	universe.probe["evidence_ids"] = asValues
	finalizeDocument(t, universe.probe, "probe_id")
}

// TestCapabilitiesForOperation pins the registry-derived dependency map:
// which capabilities confer each operation. manifest and probe confer
// through no capability and return an empty set by design.
func TestCapabilitiesForOperation(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"manifest": {}, "probe": {},
		"create":             {"credential_capable_execution_realm", "durable_disconnect", "headless_creation", "scrollback_retention", "terminal_state_retention"},
		"attach":             {"local_attach", "multi_attach", "multiple_input_clients", "remote_attach", "web_attach"},
		"status":             {"durable_disconnect", "provider_process_observation", "scrollback_retention", "terminal_state_retention"},
		"quiesce-input":      {"input_quiescence"},
		"wait-safe-boundary": {"provider_process_observation", "safe_boundary_observation"},
		"request-stop":       {"graceful_stop"},
		"terminate-stale":    {"provider_process_observation", "stale_process_termination"},
		"restore":            {"credential_capable_execution_realm", "reboot_restoration", "scrollback_retention", "terminal_state_retention"},
	}
	for operation, expected := range want {
		got, err := terminalbackend.CapabilitiesForOperation(operation)
		if err != nil {
			t.Errorf("CapabilitiesForOperation(%q) error = %v", operation, err)
			continue
		}
		sort.Strings(got)
		if fmt.Sprint(got) != fmt.Sprint(expected) {
			t.Errorf("CapabilitiesForOperation(%q) = %v, want %v", operation, got, expected)
		}
	}
	_, err := terminalbackend.CapabilitiesForOperation("launch")
	requireRefusal(t, err, codeMismatch, "operation vocabulary")
	requireRefusal(t, terminalbackend.CheckOperation("launch", terminalbackend.Admitted{}), codeMismatch, "operation vocabulary")
}

// TestCheckOperationAdmitsDependencyFreeOperations pins the §4.D rule that
// manifest and probe carry no capability dependency: both admit against the
// empty set and against a full set, while a capability-gated operation is
// still refused with the dependency clause.
func TestCheckOperationAdmitsDependencyFreeOperations(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	manifest, probe, evidence := universe.parsed(t)
	full, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	for _, admitted := range []terminalbackend.Admitted{{}, full} {
		if err := terminalbackend.CheckOperation("manifest", admitted); err != nil {
			t.Errorf("CheckOperation(manifest, %v) error = %v, want admission: manifest carries no capability dependency", admitted.Capabilities, err)
		}
		if err := terminalbackend.CheckOperation("probe", admitted); err != nil {
			t.Errorf("CheckOperation(probe, %v) error = %v, want admission: probe carries no capability dependency", admitted.Capabilities, err)
		}
	}
	requireRefusal(t, terminalbackend.CheckOperation("create", terminalbackend.Admitted{}), codeCapabilityUnproven, "operation capability dependency")
}

// TestEmptyTupleAdmitsEmptySet pins the vacuous boundary explicitly: a
// claim-free probe with no evidence admits an empty set without error. The
// set is empty because nothing was claimed, not because a filter removed
// anything; callers must not read this as proof of filtering.
func TestEmptyTupleAdmitsEmptySet(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	universe.manifest["static_capability_claims"] = []any{}
	finalizeDocument(t, universe.manifest, "manifest_id")
	universe.probe["capability_claims"] = []any{}
	universe.probe["evidence_ids"] = []any{}
	finalizeDocument(t, universe.probe, "probe_id")
	universe.evidenceByCap = map[string]map[string]any{}

	manifest, probe, evidence := reconcileFixture(t, universe)
	admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(admitted.Capabilities) != 0 || len(admitted.EvidenceIDs) != 0 {
		t.Errorf("Admitted = %+v, want empty", admitted)
	}
	requireRefusal(t, terminalbackend.CheckOperation("status", admitted), codeCapabilityUnproven, "operation capability dependency")
}

// TestAdmittedSharesNoBackingArrays keeps leaf 1's pin true for the new
// surface: admitted outputs are freshly allocated copies, so mutating the
// typed inputs after admission cannot rewrite the admitted set, and no
// exported path reaches registry or document interior arrays.
func TestAdmittedSharesNoBackingArrays(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	manifest, probe, evidence := universe.parsed(t)
	admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	beforeCapabilities := append([]string{}, admitted.Capabilities...)
	beforeEvidence := append([]string{}, admitted.EvidenceIDs...)
	probe.EvidenceIDs[0] = beforeEvidence[0] + "-mutated"
	manifest.ProtocolVersions[0] = "9.9.9"
	evidence[0].Facts[0] = "mutated_fact"
	if fmt.Sprint(admitted.Capabilities) != fmt.Sprint(beforeCapabilities) ||
		fmt.Sprint(admitted.EvidenceIDs) != fmt.Sprint(beforeEvidence) {
		t.Errorf("Admitted = %+v after input mutation, want immutable output", admitted)
	}
}

// TestUnavailableProbeStillAdmits pins the deliberate non-gate:
// availability is an observation, not admission. An unavailable probe with
// coherent evidence admits; activation policy owns the availability read.
func TestUnavailableProbeStillAdmits(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	universe.probe["availability"] = "unavailable"
	finalizeDocument(t, universe.probe, "probe_id")
	manifest, probe, evidence := reconcileFixture(t, universe)
	if _, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key)); err != nil {
		t.Errorf("Reconcile(unavailable) error = %v, want admission", err)
	}
}

// TestAdmitProbeBindsRegistry is the production entry point: the universe
// manifest resolves through a registry opened with the matching versions,
// binds member-for-member, and admits.
func TestAdmitProbeBindsRegistry(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	registry, err := terminalbackend.New("2.1.0", []string{"1.0.0", "1.1.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	manifestRaw := mustMarshal(t, universe.manifest)
	probeRaw := mustMarshal(t, universe.probe)
	evidenceRaws := make([][]byte, 0, len(universe.evidenceByCap))
	for _, capability := range []string{"durable_disconnect", "graceful_stop", "headless_creation"} {
		evidenceRaws = append(evidenceRaws, mustMarshal(t, universe.evidenceByCap[capability]))
	}
	admitted, err := registry.AdmitProbe(manifestRaw, probeRaw, evidenceRaws, universe.rawGeneration, universe.now, testVerifier(universe.key))
	if err != nil {
		t.Fatalf("AdmitProbe() error = %v", err)
	}
	if !admitted.Has("graceful_stop") || admitted.Has("local_attach") {
		t.Errorf("Admitted = %v, want proved true claims only", admitted.Capabilities)
	}
}

// TestAdmitProbeRefusesUnknownIdentity proves registry reads fail closed:
// a well-formed manifest for a never-admitted backend is not-found, never
// a default.
func TestAdmitProbeRefusesUnknownIdentity(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	universe.manifest["terminal_backend_id"] = "example.term"
	finalizeDocument(t, universe.manifest, "manifest_id")
	universe.probe["terminal_backend_id"] = "example.term"
	finalizeDocument(t, universe.probe, "probe_id")
	for _, object := range universe.evidenceByCap {
		object["terminal_backend_id"] = "example.term"
		finalizeEvidence(t, universe.key, object)
	}
	finalizeProbeIDs(t, universe)

	registry, err := terminalbackend.New("2.1.0", []string{"1.0.0", "1.1.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	evidenceRaws := make([][]byte, 0, len(universe.evidenceByCap))
	for _, name := range []string{"durable_disconnect", "graceful_stop", "headless_creation"} {
		evidenceRaws = append(evidenceRaws, mustMarshal(t, universe.evidenceByCap[name]))
	}
	_, err = registry.AdmitProbe(
		mustMarshal(t, universe.manifest), mustMarshal(t, universe.probe),
		evidenceRaws, universe.rawGeneration, universe.now, testVerifier(universe.key),
	)
	requireRefusal(t, err, codeNotFound, "unregistered terminal_backend_id")
}

// TestAdmitProbeRefusesRecordDrift proves the manifest-to-record binding:
// an implementation version the registry never admitted is drift.
func TestAdmitProbeRefusesRecordDrift(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	universe.manifest["implementation_version"] = "9.9.9"
	finalizeDocument(t, universe.manifest, "manifest_id")

	registry, err := terminalbackend.New("2.1.0", []string{"1.0.0", "1.1.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = registry.AdmitProbe(
		mustMarshal(t, universe.manifest), mustMarshal(t, universe.probe),
		nil, universe.rawGeneration, universe.now, testVerifier(universe.key),
	)
	requireRefusal(t, err, codeDrift, "manifest implementation drift")
}

// TestAdmitProbeExternalRealm exercises the external path end to end: an
// external adapter registers with trust, its manifest binds the digest, and
// realm evidence proves the credential claim. A substituted digest in the
// manifest is untrusted, not drift.
func TestAdmitProbeExternalRealm(t *testing.T) {
	t.Parallel()

	const backendID = "example.realm"
	executableDigest := testSeedDigest(0xE5)
	rawGeneration := "generation-realm"
	generationDigest, err := terminalbackend.GenerationDigest(rawGeneration)
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	fixtureID := testSeedDigest(0xF2)
	issuerID := testSeedDigest(0x2D)
	now := time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC)
	key := testKey(t)

	registry, err := terminalbackend.New("2.1.0", []string{"1.0.0", "1.1.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	manifest := map[string]any{
		"schema":                 terminalbackend.SchemaManifest,
		"schema_version":         "1.0.0",
		"manifest_id":            "",
		"terminal_backend_id":    backendID,
		"implementation_version": "3.0.0",
		"protocol_versions":      []any{"1.0.0"},
		"platforms":              []any{"linux"},
		"implementation_kind":    "local_program",
		"executable_digest":      executableDigest,
		"static_capability_claims": []any{
			claimMap("credential_capable_execution_realm", "static", true),
		},
		"conformance_fixture_id": fixtureID,
		"extensions":             map[string]any{},
	}
	finalizeDocument(t, manifest, "manifest_id")

	// Register the external adapter through the production trust gate so
	// the manifest below binds a genuinely admitted record.
	if err := registerExternalForTest(t, registry, backendID, executableDigest); err != nil {
		t.Fatalf("RegisterExternal() error = %v", err)
	}

	probe := map[string]any{
		"schema":                    terminalbackend.SchemaProbe,
		"schema_version":            "1.0.0",
		"probe_id":                  "",
		"terminal_backend_id":       backendID,
		"implementation_version":    "3.0.0",
		"protocol_version":          "1.0.0",
		"implementation_kind":       "local_program",
		"executable_digest":         executableDigest,
		"platform":                  "linux",
		"os_version":                "6.8",
		"availability":              "available",
		"backend_generation_digest": generationDigest,
		"capability_claims": []any{
			claimMap("credential_capable_execution_realm", "probed", true),
		},
		"evidence_ids": []any{},
		"probed_at":    "2026-02-01T09:00:00.000Z",
		"extensions":   map[string]any{},
	}

	evidence := map[string]any{
		"schema":                     terminalbackend.SchemaCapabilityEvidence,
		"schema_version":             "1.0.0",
		"evidence_id":                "",
		"terminal_backend_id":        backendID,
		"implementation_version":     "3.0.0",
		"protocol_version":           "1.0.0",
		"backend_generation_digest":  generationDigest,
		"capability":                 "credential_capable_execution_realm",
		"value":                      true,
		"platform":                   "linux",
		"os_version":                 "6.8",
		"conformance_fixture_id":     fixtureID,
		"observed_at":                "2026-01-01T00:00:00.000Z",
		"expires_at":                 "2027-01-01T00:00:00.000Z",
		"issuer":                     terminalbackend.IssuerRelease,
		"issuer_id":                  issuerID,
		"attestation_signature":      "",
		"facts":                      []any{"fixture_passed", "provider_auth_passed", "runtime_probe_passed", "sentinel_passed"},
		"terminal_binding_id":        testSeedDigest(0xB1),
		"provider_id":                "codex",
		"provider_build":             "1.2.3",
		"sentinel_result":            "passed",
		"provider_auth_smoke_result": "passed",
		"extensions":                 map[string]any{},
	}
	finalizeEvidence(t, key, evidence)
	probe["evidence_ids"] = []any{evidence["evidence_id"]}
	finalizeDocument(t, probe, "probe_id")

	admitted, err := registry.AdmitProbe(
		mustMarshal(t, manifest), mustMarshal(t, probe),
		[][]byte{mustMarshal(t, evidence)}, rawGeneration, now, testVerifier(key),
	)
	if err != nil {
		t.Fatalf("AdmitProbe() error = %v", err)
	}
	if !admitted.Has("credential_capable_execution_realm") {
		t.Errorf("Admitted = %v, want credential realm", admitted.Capabilities)
	}

	// Substitution: the manifest carries a foreign digest while the probe
	// tracks it, so the probe-to-manifest identity holds and only the
	// manifest-to-record binding can refuse. Without that tracking, a
	// deleted record gate slides onto the probe identity arm — which
	// reports the same code — and the suite stays green.
	substituted := cloneMap(t, manifest)
	substituted["executable_digest"] = testSeedDigest(0xE6)
	finalizeDocument(t, substituted, "manifest_id")
	trackedProbe := cloneMap(t, probe)
	trackedProbe["executable_digest"] = testSeedDigest(0xE6)
	finalizeDocument(t, trackedProbe, "probe_id")
	_, err = registry.AdmitProbe(
		mustMarshal(t, substituted), mustMarshal(t, trackedProbe),
		[][]byte{mustMarshal(t, evidence)}, rawGeneration, now, testVerifier(key),
	)
	requireRefusal(t, err, codeUntrusted, "executable substitution")
}

// registerExternalForTest admits one local_program adapter through the
// production RegisterExternal trust gate. Trust validation is grammatical
// (absolute path, well-formed digest), so no executable file is needed.
func registerExternalForTest(t *testing.T, registry *terminalbackend.Registry, backendID, digest string) error {
	t.Helper()

	return registry.RegisterExternal(
		scalar.PlatformLinux,
		terminalbackend.TrustEntry{
			BackendID:        backendID,
			ExecutablePath:   "/opt/vendor/bin/ax-backend-realm",
			ExecutableDigest: digest,
			Enabled:          true,
		},
		terminalbackend.Registration{
			ID:                    backendID,
			Kind:                  terminalbackend.KindLocalProgram,
			ImplementationVersion: "3.0.0",
			ProtocolVersions:      []string{"1.0.0"},
			Platforms:             []scalar.Platform{scalar.PlatformLinux},
			ExecutableDigest:      digest,
		},
	)
}
