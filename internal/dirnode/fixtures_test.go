package dirnode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// Shared fixtures for the dirnode suite. Every builder emits
// hand-written JSON with independently chosen literals: no builder
// derives its probe points from production constants, so a mutated
// production constant cannot take its fixture with it. The verdict
// always comes from production; the vectors never do. In
// particular the schema URNs, protocol URN, version strings,
// operation names, and member names below are retyped from the
// pinned specification, not referenced from the package under
// test.

// Fixed fixture identities. The UUIDs carry a 7 version nibble;
// the digests are syntactically pinned sha256 hex.
const (
	fixtureNodeID      = "test-directory-node"
	fixtureNodeVersion = "2.4.1"
	fixtureHostID      = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	fixtureRequestID   = "0198f4c8-8e50-7f66-8f70-1234567890ac"
	fixtureOperationID = "0198f4c8-8e50-7f66-8f70-1234567890ad"
	fixtureQueryID     = "0198f4c8-8e50-7f66-8f70-1234567890ae"
	fixtureOriginHost  = "0198f4c8-8e50-7f66-8f70-1234567890af"
	fixtureObservedAt  = "2026-08-19T04:05:00.000Z"
)

// fixtureDigest returns the pinned digest of one seed.
func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Standard fixture digests, each naming its seed.
var (
	fixtureExecutableDigest = fixtureDigest("test-node-executable")
	fixtureProviderDigest   = fixtureDigest("test-provider-manifest")
	fixtureAdapterDigest    = fixtureDigest("test-adapter-manifest")
	fixtureTupleRegistry    = fixtureDigest("test-tuple-registry")
	fixturePolicyDigest     = fixtureDigest("test-policy")
	fixtureRedactionDigest  = fixtureDigest("test-redaction-policy")
	fixtureEnrichmentDigest = fixtureDigest("test-enrichment-profile")
	fixtureEvidenceDigest   = fixtureDigest("test-evidence")
	fixtureInstallDigestA   = fixtureDigest("test-install-a")
	fixtureInstallDigestB   = fixtureDigest("test-install-b")
	fixtureBatchDigest      = fixtureDigest("test-batch")
	fixtureNativeDigestA    = fixtureDigest("test-native-a")
	fixtureHeadDigest       = fixtureDigest("test-head")
	fixturePlanDigest       = fixtureDigest("test-plan")
)

// fixtureRequestFrame builds one hand-written request envelope for
// the given wire version string, operation, and body.
func fixtureRequestFrame(version, operation, requestID string, deadline uint64, body string) []byte {
	return []byte(fmt.Sprintf(`{"schema":"urn:ax:schema:session-directory-node-request","schema_version":%q,"protocol":"urn:ax:protocol:session-directory-node","protocol_version":%q,"request_id":%q,"operation":%q,"deadline_ms":%d,"body":%s}`,
		version, version, requestID, operation, deadline, body))
}

// fixtureManifestJSON builds one complete valid manifest. The
// supported versions name both majors; the operations list is the
// sorted eleven-name registry; the schemas list carries fifteen
// minimal assertions; every capability is available with null
// reason except conditional and unknown rows that prove the
// reason-coherence rule in the positive direction.
func fixtureManifestJSON() string {
	var assertions []string
	for index := 0; index < 15; index++ {
		assertions = append(assertions, fmt.Sprintf(`{"contract_id":"urn:ax:schema:contract-%02d","exact_version":"1.0.%d","extensions":{}}`, index, index))
	}
	operations := []string{
		"continuation-inspect", "doctor", "enrichment-plan", "enrichment-run",
		"enrichment-status", "inventory", "manifest", "preview", "probe",
		"runtime-observe", "scan",
	}
	ops := make([]string, 0, len(operations))
	for _, name := range operations {
		ops = append(ops, fmt.Sprintf("%q", name))
	}
	capabilities := []string{
		`"directory_discovery":{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
		fmt.Sprintf(`"directory_incremental_scan":{"status":"available","reason_code":null,"evidence_ids":[%q],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`, fixtureEvidenceDigest),
		`"directory_head_digest":{"status":"conditional","reason_code":"needs-quiesce","evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
		`"directory_tail_preview":{"status":"unavailable","reason_code":"not-probed","evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
		`"native_title_read":{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
		`"native_runtime_observation":{"status":"unknown","reason_code":"no-evidence","evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
		`"existing_session_adoption":{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
		`"native_resume":{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`,
	}
	return fmt.Sprintf(`{"schema":"urn:ax:schema:session-directory-node-manifest","schema_version":"1.0.0","node_id":%q,"node_version":%q,"host_id":%q,"executable_sha256":%q,"provider_manifest_digest":%q,"session_adapter_manifest_digest":%q,"supported_protocol_versions":["1.0.0","2.0.0"],"operations":[%s],"schemas":[%s],"environment_tuple_registry_id":%q,"capabilities":{%s},"redaction_policy_ids":[%q],"enrichment_profile_ids":[%q],"limits":{"max_frame_bytes":8388608,"max_scan_instances":65536,"max_inventory_take":1000,"max_excerpt_count":20,"max_excerpt_bytes":4096,"max_enrichment_events":5000,"max_enrichment_bytes":4194304,"extensions":{}},"extensions":{}}`,
		fixtureNodeID, fixtureNodeVersion, fixtureHostID,
		fixtureExecutableDigest, fixtureProviderDigest, fixtureAdapterDigest,
		strings.Join(ops, ","), strings.Join(assertions, ","),
		fixtureTupleRegistry, strings.Join(capabilities, ","),
		fixtureRedactionDigest, fixtureEnrichmentDigest)
}

// fixtureObservedFacade returns the observed facts matching the
// fixture manifest bindings.
func fixtureObservedFacade() ObservedFacade {
	return ObservedFacade{
		ExecutableSHA256:       fixtureExecutableDigest,
		ProviderManifestDigest: fixtureProviderDigest,
		SessionAdapterDigest:   fixtureAdapterDigest,
	}
}

// fixtureSuccessFrame wraps a body in a hand-written success
// envelope echoing the given request identity.
func fixtureSuccessFrame(version, operation, requestID, body string) []byte {
	return []byte(fmt.Sprintf(`{"schema":"urn:ax:schema:session-directory-node-response","schema_version":"1.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":%q,"request_id":%q,"operation":%q,"ok":true,"body":%s}`,
		version, requestID, operation, body))
}

// fixtureErrorObject builds one hand-written Structured Error 1.2.0
// object with independently chosen literals.
func fixtureErrorObject(code string, exit int, retryable bool) string {
	return fmt.Sprintf(`{"schema":"urn:ax:schema:error","schema_version":"1.2.0","code":%q,"message":"fixture %s","exit_code":%d,"retryable":%v,"details":{}}`,
		code, code, exit, retryable)
}

// fixtureFailureFrame wraps an error object in a hand-written
// failure envelope echoing the given request identity.
func fixtureFailureFrame(version, operation, requestID, errorObject string) []byte {
	return []byte(fmt.Sprintf(`{"schema":"urn:ax:schema:session-directory-node-response","schema_version":"1.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":%q,"request_id":%q,"operation":%q,"ok":false,"error":%s}`,
		version, requestID, operation, errorObject))
}

// fixtureDowngradeFrame is the exact downgrade tuple for the given
// major: failure echo plus incompatible_protocol, exit 6,
// retryable=false.
func fixtureDowngradeFrame(version, requestID string) []byte {
	return fixtureFailureFrame(version, "manifest", requestID, fixtureErrorObject("incompatible_protocol", 6, false))
}

// fixtureProbeRequestV2 is one valid Request 2.0.0 probe body.
func fixtureProbeRequestV2() string {
	return `{"platform":"linux","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":["directory_discovery"],"extensions":{}}`
}

// fixtureProbeRequestV1 is one valid Request 1.0.0 probe body.
func fixtureProbeRequestV1() string {
	return `{"platform":"darwin","architecture":"arm64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`
}

// fixtureProbeResponseJSON is one valid probe success body whose
// node build equals the fixture manifest.
func fixtureProbeResponseJSON() string {
	return fmt.Sprintf(`{"host_id":%q,"node_build":{"node_id":%q,"node_version":%q,"executable_sha256":%q,"provider_manifest_digest":%q,"session_adapter_manifest_digest":%q,"extensions":{}},"policy_digest":%q,"environments":[{"environment_id":"test.env"}],"findings":[{"severity":"info","code":"probe-ok","message":"probe completed","remediation":null,"extensions":{}}],"extensions":{}}`,
		fixtureHostID, fixtureNodeID, fixtureNodeVersion,
		fixtureExecutableDigest, fixtureProviderDigest, fixtureAdapterDigest,
		fixturePolicyDigest)
}

// fixtureScanRequestJSON is one valid scan request body. The
// installation pair emits in sorted order: the sortedness verdict
// belongs to production, and the builder arranges a valid vector
// without retyping production's table.
func fixtureScanRequestJSON() string {
	pair := []string{fixtureInstallDigestA, fixtureInstallDigestB}
	sort.Strings(pair)
	return fmt.Sprintf(`{"operation_id":%q,"installation_ids":[%q,%q],"prior_batch_id":null,"cursor":null,"max_instances":100,"extensions":{}}`,
		fixtureOperationID, pair[0], pair[1])
}

// fixtureScanResponseJSON is one valid scan success body.
func fixtureScanResponseJSON() string {
	return fmt.Sprintf(`{"batch":{"batch_id":%q},"environment_observation_ids":[%q],"native_observation_ids":[%q],"next_cursor":null,"extensions":{}}`,
		fixtureBatchDigest, fixtureEvidenceDigest, fixtureNativeDigestA)
}

// fixtureCallerJSON is one valid CallerContext.
func fixtureCallerJSON() string {
	return fmt.Sprintf(`{"caller_id":"test-caller","authentication_subject":"test-subject","origin_host_id":%q,"interaction":"non_interactive","scopes":["directory.read"],"disclosure_policy_digest":%q,"extensions":{}}`,
		fixtureOriginHost, fixturePolicyDigest)
}

// fixtureQueryJSON is one valid two-operation batch: a schema read
// and a sessions read with filters.
func fixtureQueryJSON() string {
	return fmt.Sprintf(`{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":%q,"operations":[{"operation_index":0,"name":"schema","parameters":{"extensions":{}},"fields":null,"preset":null,"skip":0,"take":1,"sort":[],"dry_run":false,"confirm":false,"expectation_digest":null,"idempotency_key":null,"extensions":{}},{"operation_index":1,"name":"sessions","parameters":{"filters":{"kinds":[],"lineage_anchors":[],"provider_ids":[],"host_ids":[],"workspace_ids":[],"states":[],"management_states":[],"reachability":[],"freshness":[],"warnings":[],"updated_before":null,"updated_after":null,"extensions":{}},"extensions":{}},"fields":null,"preset":"overview","skip":0,"take":10,"sort":[{"field":"stable_id","direction":"asc","extensions":{}}],"dry_run":false,"confirm":false,"expectation_digest":null,"idempotency_key":null,"extensions":{}}],"caller":%s,"extensions":{}}`,
		fixtureQueryID, fixtureCallerJSON())
}

// replaceOnce substitutes the first occurrence like strings.Replace,
// but fails the test when the needle is absent or the count is not
// one: a fixture-surgery vector that mutates nothing would otherwise
// pass vacuously against a still-valid body, measuring the scanner
// instead of the gate.
func replaceOnce(t *testing.T, haystack, needle, replacement string, n int) string {
	t.Helper()
	if n != 1 {
		t.Fatalf("replaceOnce: count %d is not 1", n)
	}
	if !strings.Contains(haystack, needle) {
		t.Fatalf("replaceOnce: needle %q absent; the vector mutates nothing", needle)
	}
	return strings.Replace(haystack, needle, replacement, n)
}

// requireAxError unwraps a production refusal into its Structured
// Error, failing the test when the value is a transport-level Go
// error instead of a wire refusal.
func requireAxError(t *testing.T, err error) *axerror.Error {
	t.Helper()
	if err == nil {
		t.Fatal("want refusal, got nil error")
	}
	failure, ok := err.(*axerror.Error)
	if !ok {
		t.Fatalf("want *axerror.Error, got %T (%v)", err, err)
	}
	return failure
}

// requireCode asserts the refusal code and returns the failure for
// further assertions.
func requireCode(t *testing.T, err error, code axerror.Code) *axerror.Error {
	t.Helper()
	failure := requireAxError(t, err)
	if failure.Code() != code {
		t.Fatalf("refusal code = %q, want %q (detail keys %v)", failure.Code(), code, failure.DetailKeys())
	}
	return failure
}
