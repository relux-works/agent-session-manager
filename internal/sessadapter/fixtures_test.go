package sessadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// Shared fixtures for the sessadapter suite. Every builder emits
// hand-written JSON with independently chosen literals: no builder
// derives its probe points from production constants, so a mutated
// production constant cannot take its fixture with it. The verdict
// always comes from production; the vectors never do.

// Fixed fixture identities. The UUIDs carry a 7 version nibble;
// the digests are syntactically pinned sha256 hex.
const (
	fixtureProviderID    = "test-provider"
	fixtureEnvironmentID = "test.env"
	fixtureAdapterVer    = "1.2.3"
	fixtureRequestID     = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	fixtureOperationID   = "0198f4c8-8e50-7f66-8f70-1234567890ac"
	fixtureAuthorityID   = "0198f4c8-8e50-7f66-8f70-1234567890ad"
	fixtureDeadline      = "2026-08-19T04:05:00.000Z"
	fixtureValidFrom     = "2026-01-01T00:00:00.000Z"
	fixtureValidUntil    = "2027-01-01T00:00:00.000Z"
	fixtureCallTime      = "2026-06-01T00:00:00.000Z"
	fixtureExecutedAt    = "2026-05-01T00:00:00.000Z"
	fixtureExpiresAt     = "2026-08-19T05:05:00.000Z"
)

// fixtureDigest returns the pinned digest of one seed.
func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Standard fixture digests, each naming its seed.
var (
	fixtureExecutableDigest = fixtureDigest("test-executable")
	fixtureProviderManifest = fixtureDigest("test-provider-manifest")
	fixtureStorePrint       = fixtureDigest("test-store-fingerprint")
	fixtureSuiteDigest      = fixtureDigest("test-suite")
	fixtureEvidenceDigest   = fixtureDigest("test-evidence")
	fixtureSmokeDigest      = fixtureDigest("test-smoke")
	fixtureRegistryDigest   = fixtureDigest("test-registry")
	fixturePlanDigest       = fixtureDigest("test-plan")
	fixtureCaptureDigest    = fixtureDigest("test-capture-manifest")
	fixtureSessionDigest    = fixtureDigest("test-canonical-session")
	fixtureProjectedDigest  = fixtureDigest("test-projected-manifest")
	fixtureStructuralDigest = fixtureDigest("test-structural")
	fixtureReadBackDigest   = fixtureDigest("test-read-back-evidence")
)

// fixtureTupleJSON is one valid Environment Tuple.
func fixtureTupleJSON() string {
	return fmt.Sprintf(`{"environment_id":%q,"environment_version":"2.1.0","platform":"linux","architecture":"amd64","store_schema_fingerprint":%q,"adapter_version":%q}`,
		fixtureEnvironmentID, fixtureStorePrint, fixtureAdapterVer)
}

// fixtureContextJSON is one valid AdapterCallContext carrying the
// given request digest.
func fixtureContextJSON(requestDigest string) string {
	return fmt.Sprintf(`{"operation_id":%q,"provider_id":%q,"environment":%s,"session_adapter_manifest_digest":%q,"executable_sha256":%q,"request_digest":%q,"extensions":{}}`,
		fixtureOperationID, fixtureProviderID, fixtureTupleJSON(), fixtureDigest("test-adapter-manifest"), fixtureExecutableDigest, requestDigest)
}

// fixtureReadAuthorityJSON is one valid source_native ReadAuthority.
func fixtureReadAuthorityJSON() string {
	return fmt.Sprintf(`{"authority_id":%q,"purpose":"source_native","root_handle_names":["handle-a"],"expires_at":%q,"extensions":{}}`,
		fixtureAuthorityID, fixtureExpiresAt)
}

// fixtureFreshSinkJSON is one valid fresh_sink ObjectAuthority.
func fixtureFreshSinkJSON(purpose string) string {
	return fmt.Sprintf(`{"authority_id":%q,"purpose":%q,"mode":"fresh_sink","max_objects":100,"max_total_bytes":1000000,"extensions":{}}`,
		fixtureAuthorityID, purpose)
}

// fixtureManifestJSON is one valid Session Adapter Manifest 1.0.0
// with the registries in section order.
func fixtureManifestJSON() string {
	operations := make([]string, 0, len(operationOrder))
	for _, operation := range operationOrder {
		operations = append(operations, `"`+string(operation)+`"`)
	}
	capabilities := make([]string, 0, len(capabilityOrder))
	for _, name := range capabilityOrder {
		capabilities = append(capabilities, `"`+name+`"`)
	}
	return fmt.Sprintf(`{"schema":"urn:ax:schema:session-adapter-manifest","schema_version":"1.0.0","provider_id":%q,"environment_id":%q,"display_name":"Test Adapter","adapter_version":%q,"environment_version_range":">=1.0.0","platforms":["linux","macos"],"operations":[%s],"capability_names":[%s],"extensions":{}}`,
		fixtureProviderID, fixtureEnvironmentID, fixtureAdapterVer,
		strings.Join(operations, ","), strings.Join(capabilities, ","))
}

// fixtureCapabilityJSON is one available+enabled capability value.
func fixtureCapabilityJSON() string {
	return `{"status":"available","enabled":true,"evidence":"probed","detail":""}`
}

// fixtureProbeJSON is one valid Session Adapter Probe 1.0.0 with
// every capability available and enabled.
func fixtureProbeJSON(manifestDigest string) string {
	values := make([]string, 0, len(capabilityOrder))
	for _, name := range capabilityOrder {
		values = append(values, `"`+name+`":`+fixtureCapabilityJSON())
	}
	return fmt.Sprintf(`{"schema":"urn:ax:schema:session-adapter-probe","schema_version":"1.0.0","provider_id":%q,"adapter_manifest_digest":%q,"adapter_version":%q,"environment":%s,"capabilities":{%s},"warnings":[],"extensions":{}}`,
		fixtureProviderID, manifestDigest, fixtureAdapterVer, fixtureTupleJSON(), strings.Join(values, ","))
}

// fixtureEntryJSON is one valid accepted registry entry for the
// direction: source_read carries strategies=[archive_only] with
// null smoke, target_write carries the native-writer strategy
// with passing smoke.
func fixtureEntryJSON(direction Direction) string {
	strategies := `["archive_only"]`
	smoke := `null`
	if direction == DirectionTargetWrite {
		strategies = `["target_native_writer"]`
		smoke = fmt.Sprintf(`{"result":"pass","executed_at":%q,"evidence_digest":%q,"native_cli_family":"claude","bounded_continuation_turn_passed":true}`,
			fixtureExecutedAt, fixtureSmokeDigest)
	}
	return fmt.Sprintf(`{"key":{"direction":%q,"environment":%s,"provider_id":%q,"candidate_kind":"builtin","executable_sha256":%q,"provider_manifest_digest":%q,"session_adapter_manifest_digest":%q},"entry_sequence":7,"contracts":[{"contract_id":"urn:ax:protocol:session-adapter","versions":["1.0.0"]}],"strategies":%s,"fixture_evidence":{"suite_revision":"rev-9","suite_digest":%q,"result":"pass","executed_at":%q,"evidence_digest":%q,"fixture_count":12},"resume_smoke_evidence":%s,"known_fidelity_limits":[],"valid_from":%q,"valid_until":%q,"status":"accepted","revocation_reason":null,"revoked_at":null,"extensions":{}}`,
		string(direction), fixtureTupleJSON(), fixtureProviderID,
		fixtureExecutableDigest, fixtureProviderManifest, fixtureDigest("test-adapter-manifest"),
		strategies, fixtureSuiteDigest, fixtureExecutedAt, fixtureEvidenceDigest, smoke,
		fixtureValidFrom, fixtureValidUntil)
}

// fixtureBindingFacts are the host-observed facts matching every
// fixture above.
func fixtureBindingFacts() BindingFacts {
	return BindingFacts{
		ProviderID:             fixtureProviderID,
		CandidateKind:          CandidateBuiltin,
		ExecutableSHA256:       fixtureExecutableDigest,
		ProviderManifestDigest: fixtureProviderManifest,
		AdapterManifestDigest:  fixtureDigest("test-adapter-manifest"),
	}
}

// fixtureCandidate is the trusted candidate matching every
// fixture above.
func fixtureCandidate() TrustedCandidate {
	return TrustedCandidate{
		ProviderID:             fixtureProviderID,
		Kind:                   CandidateBuiltin,
		ExecutablePath:         "/opt/ax/bin/ax-provider-test-provider",
		OwnerIdentity:          "test-owner",
		ExecutableSHA256:       fixtureExecutableDigest,
		ProviderManifestDigest: fixtureProviderManifest,
	}
}

// mutateMember returns the object with one top-level member
// replaced by the given raw JSON. The replacement is applied to a
// fresh positive fixture, one mutation at a time, per the
// Appendix D execution rules. Members re-encode in sorted key
// order and the replacement passes through unvalidated, so
// malformed values reach the production decoder instead of
// failing the test scaffolding.
func mutateMember(t *testing.T, body []byte, member, replacement string) []byte {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(body, &members); err != nil {
		t.Fatalf("mutateMember: fixture is not JSON: %v", err)
	}
	members[member] = json.RawMessage(replacement)
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(members))
	for _, name := range names {
		quoted, err := json.Marshal(name)
		if err != nil {
			t.Fatalf("mutateMember: %v", err)
		}
		parts = append(parts, string(quoted)+":"+string(members[name]))
	}
	return []byte("{" + strings.Join(parts, ",") + "}")
}

// dropMember returns the object without one top-level member.
func dropMember(t *testing.T, body []byte, member string) []byte {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(body, &members); err != nil {
		t.Fatalf("dropMember: fixture is not JSON: %v", err)
	}
	delete(members, member)
	dropped, err := json.Marshal(members)
	if err != nil {
		t.Fatalf("dropMember: %v", err)
	}
	return dropped
}

// failureObject requires a Structured Error and returns it.
func failureObject(t *testing.T, err error) *axerror.Error {
	t.Helper()
	if err == nil {
		t.Fatal("want refusal, got nil error")
	}
	failure, ok := err.(*axerror.Error)
	if !ok {
		t.Fatalf("error %v is not a Structured Error", err)
	}
	return failure
}

// failureCode returns the stable code of a refusal.
func failureCode(t *testing.T, err error) axerror.Code {
	t.Helper()
	return failureObject(t, err).Code()
}

// failureDetail returns one string diagnostic of a refusal.
func failureDetail(t *testing.T, err error, key string) string {
	t.Helper()
	detail, ok := failureObject(t, err).Detail(key)
	if !ok {
		t.Fatalf("refusal %v carries no %q detail", err, key)
	}
	text, ok := detail.(string)
	if !ok {
		t.Fatalf("refusal %v detail %q is %T, want string", err, key, detail)
	}
	return text
}

// requireRefusal asserts the full arm identity of a refusal: the
// stable code, one distinguishing diagnostic, the detail naming
// the rule in the human text, and the non-retryable bit. Asserting
// the distinguishing diagnostic pins which arm fired: deleting a
// gate slides the refusal to a lower arm carrying the same code
// with a different diagnostic, and the slide reddens here.
func requireRefusal(t *testing.T, err error, code axerror.Code, detailKey, detailValue, text string) {
	t.Helper()
	if failureCode(t, err) != code {
		t.Fatalf("code = %v, want %s", err, code)
	}
	if got := failureDetail(t, err, detailKey); got != detailValue {
		t.Fatalf("%s detail = %q, want %q (error: %v)", detailKey, got, detailValue, err)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error = %v, want detail containing %q", err, text)
	}
	if failureObject(t, err).Retryable() {
		t.Fatalf("error = %v, want non-retryable", err)
	}
}

// canonicalJSON canonicalizes test scaffolding. It is test-only
// assembly, never a verdict: every verdict comes from production.
func canonicalJSON(t *testing.T, body []byte) []byte {
	t.Helper()
	canonical, err := canonicaljson.Canonicalize(body)
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	return canonical
}
