package provhost

import (
	"strings"
	"testing"
)

// declaredOperationWitnessesIdentity proves the identity,
// identify-result, and idempotency-key arms. It extends
// declaredOperationWitnesses in refusal_arm_operations_a_test.go; the
// split is file size only.
func declaredOperationWitnessesIdentity() []armWitness {
	return []armWitness{
		// Identity arms, through CheckIdentity.
		{arm: `ctor|failProtocol|identity carries unknown member`, name: "identity score member", prove: func(t *testing.T) {
			body := identityVariant(t, `"extensions": {}`, `"extensions": {}, "score": 1`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "score", "unknown member")
		}},
		{arm: `ctor|failProtocol|identity extensions is not an object`, name: "identity array extensions", prove: func(t *testing.T) {
			body := identityVariant(t, `"extensions": {}`, `"extensions": []`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "extensions", "not an object")
		}},
		{arm: `ctor|failProtocol|identity host is not a UUIDv7`, name: "identity bogus host", prove: func(t *testing.T) {
			body := identityVariant(t, `"created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab"`, `"created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890aG"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "created_by_host_id", "not a UUIDv7")
		}},
		{arm: `ctor|failProtocol|identity kind is not a registry member`, name: "identity window handle kind", prove: func(t *testing.T) {
			body := identityVariant(t, `"identity_kind": "backend_conversation_uuid"`, `"identity_kind": "window_handle"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "identity_kind", "not a registry member")
		}},
		{arm: `ctor|failProtocol|identity misses a required member`, name: "identity without opaque", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specIdentityExample, "  \"opaque_identity\": {},\n", "", 1))
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "opaque_identity", "misses a required member")
		}},
		{arm: `ctor|failProtocol|identity native_session_id is not 1..512 characters`, name: "identity empty native session", prove: func(t *testing.T) {
			body := identityVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": ""`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "native_session_id", "not 1..512 characters")
		}},
		{arm: `ctor|failProtocol|identity opaque key is not a provider key`, name: "identity bad opaque key", prove: func(t *testing.T) {
			body := identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"Bad Key!": "x"}`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "opaque_identity", "not a provider key")
		}},
		{arm: `ctor|failProtocol|identity opaque value begins with an absolute path`, name: "identity absolute opaque value", prove: func(t *testing.T) {
			body := identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"workdir": "/Users/iv/work"}`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "opaque_identity", "begins with an absolute path")
		}},
		{arm: `ctor|failProtocol|identity opaque value is not 1..1024 characters`, name: "identity object opaque value", prove: func(t *testing.T) {
			body := identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"adapter": {"nested": true}}`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "opaque_identity", "not 1..1024 characters")
		}},
		{arm: `ctor|failProtocol|identity opaque_identity exceeds 32 entries`, name: "identity 33 opaque entries", prove: func(t *testing.T) {
			var entries strings.Builder
			for i := 0; i < 33; i++ {
				if i > 0 {
					entries.WriteString(", ")
				}
				entries.WriteString(`"k` + pad2(i) + `": "x"`)
			}
			body := identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {`+entries.String()+`}`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "opaque_identity", "exceeds 32 entries")
		}},
		{arm: `ctor|failProtocol|identity provider_id is not a provider id`, name: "identity uppercase provider", prove: func(t *testing.T) {
			body := identityVariant(t, `"provider_id": "antigravity"`, `"provider_id": "Antigravity"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "provider_id", "not a provider id")
		}},
		{arm: `ctor|failProtocol|identity provider_version is not 1..128 characters`, name: "identity overlong version", prove: func(t *testing.T) {
			body := identityVariant(t, `"provider_version": "1.1.14"`, `"provider_version": "`+strings.Repeat("v", 129)+`"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "provider_version", "not 1..128 characters")
		}},
		{arm: `ctor|failProtocol|identity provider_version_range is not 1..256 characters`, name: "identity empty range", prove: func(t *testing.T) {
			body := identityVariant(t, `"provider_version_range": ">=1.1.14 <1.2.0"`, `"provider_version_range": ""`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "provider_version_range", "not 1..256 characters")
		}},
		{arm: `ctor|failProtocol|identity realm is not a digest or null`, name: "identity garbage realm", prove: func(t *testing.T) {
			body := identityVariant(t, `"backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"`, `"backend_realm_fingerprint": "nope"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "backend_realm_fingerprint", "not a digest or null")
		}},
		{arm: `ctor|failProtocol|identity realm is required for this backend kind`, name: "identity antigravity null realm", prove: func(t *testing.T) {
			body := identityVariant(t, `"backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"`, `"backend_realm_fingerprint": null`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "backend_realm_fingerprint", "required for this backend kind")
		}},
		{arm: `ctor|failProtocol|identity record_id is not a digest`, name: "identity garbage digest", prove: func(t *testing.T) {
			body := identityVariant(t, `"record_id": "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2"`, `"record_id": "sha256:zzzz"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "record_id", "not a digest")
		}},
		{arm: `ctor|failProtocol|identity schema is not the provider identity`, name: "identity manifest schema", prove: func(t *testing.T) {
			body := identityVariant(t, `"urn:ax:schema:provider-identity"`, `"urn:ax:schema:provider-manifest"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "schema", "not the provider identity")
		}},
		{arm: `ctor|failProtocol|identity schema_version is not 1.0.0`, name: "identity schema 1.1.0", prove: func(t *testing.T) {
			body := identityVariant(t, `"schema_version": "1.0.0"`, `"schema_version": "1.1.0"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "schema_version", "not 1.0.0")
		}},
		{arm: `ctor|failProtocol|identity session_id is not a UUIDv7`, name: "identity bogus session", prove: func(t *testing.T) {
			body := identityVariant(t, `"session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ax"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "session_id", "not a UUIDv7")
		}},
		{arm: `ctor|failProtocol|identity subject_id does not equal session_id`, name: "identity split subject", prove: func(t *testing.T) {
			body := identityVariant(t, `"subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"subject_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "subject_id", "does not equal session_id")
		}},
		{arm: `ctor|failProtocol|identity subject_id is not a UUIDv7`, name: "identity bogus subject", prove: func(t *testing.T) {
			body := identityVariant(t, `"subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"subject_id": "not-a-uuid"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "subject_id", "not a UUIDv7")
		}},
		{arm: `ctor|failProtocol|identity timestamp is not a timestamp`, name: "identity relative timestamp", prove: func(t *testing.T) {
			body := identityVariant(t, `"created_at": "2026-08-19T04:09:45.000Z"`, `"created_at": "yesterday"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "created_at", "not a timestamp")
		}},
		{arm: `ctor|failProtocol|identity workspace is not a UUIDv7`, name: "identity bogus workspace", prove: func(t *testing.T) {
			body := identityVariant(t, `"logical_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab"`, `"logical_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890aX"`)
			requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "logical_workspace_id", "not a UUIDv7")
		}},
		// Identify-result arms, through DecodeIdentifyResult.
		{arm: `ctor|failProtocol|identify confidence is not exact strong or weak`, name: "identify certain confidence", prove: func(t *testing.T) {
			body := []byte(strings.Replace(string(specIdentifyCallBody()), `"confidence": "exact"`, `"confidence": "certain"`, 1))
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "confidence", "not exact strong or weak")
		}},
		{arm: `ctor|failProtocol|identify evidence is not 1..4 members`, name: "identify empty evidence", prove: func(t *testing.T) {
			body := []byte(strings.Replace(string(specIdentifyCallBody()), `["native_id", "store_path"]`, `[]`, 1))
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "matched_evidence", "not 1..4 members")
		}},
		{arm: `ctor|failProtocol|identify evidence is not sorted unique`, name: "identify unsorted evidence", prove: func(t *testing.T) {
			body := []byte(strings.Replace(string(specIdentifyCallBody()), `["native_id", "store_path"]`, `["store_path", "native_id"]`, 1))
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "matched_evidence", "not sorted unique")
		}},
		{arm: `ctor|failProtocol|identify evidence names an unknown member`, name: "identify window title evidence", prove: func(t *testing.T) {
			body := []byte(strings.Replace(string(specIdentifyCallBody()), `["native_id", "store_path"]`, `["window_title"]`, 1))
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "matched_evidence", "unknown member")
		}},
		{arm: `ctor|failProtocol|identify identity is not an object`, name: "identify array identity", prove: func(t *testing.T) {
			body := []byte(`{"identity": [], "confidence": "exact", "matched_evidence": ["native_id"]}`)
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "identity", "not an object")
		}},
		{arm: `ctor|failProtocol|identify result carries unknown member`, name: "identify score member", prove: func(t *testing.T) {
			body := []byte(strings.Replace(string(specIdentifyCallBody()), `"matched_evidence": ["native_id", "store_path"]`, `"matched_evidence": ["native_id", "store_path"], "score": 1`, 1))
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "score", "unknown member")
		}},
		{arm: `ctor|failProtocol|identify result misses a required member`, name: "identify without confidence", prove: func(t *testing.T) {
			body := []byte(strings.Replace(string(specIdentifyCallBody()), `, "confidence": "exact"`, "", 1))
			requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "confidence", "misses a required member")
		}},
		// Idempotency-key arms, through IdempotencyKeyFor.
		{arm: `ctor|failInvalid|idempotency key names an operation without operation_id`, name: "key for doctor", prove: func(t *testing.T) {
			_, err := IdempotencyKeyFor(OpDoctor, testRequestID)
			requireLocalRefusal(t, err, "invalid_config", "without operation_id")
		}},
		{arm: `ctor|failInvalid|idempotency key names an unknown operation`, name: "key for reboot", prove: func(t *testing.T) {
			_, err := IdempotencyKeyFor(Operation("reboot"), testRequestID)
			requireLocalRefusal(t, err, "invalid_config", "unknown operation")
		}},
		{arm: `ctor|failInvalid|idempotency operation_id is not a UUIDv7`, name: "key with garbage id", prove: func(t *testing.T) {
			_, err := IdempotencyKeyFor(OpMaterialize, "bogus")
			requireLocalRefusal(t, err, "invalid_config", "not a UUIDv7")
		}},
		{arm: `ctor|failInvalid|identity names another provider`, name: "antigravity record for codex", prove: func(t *testing.T) {
			requireLocalRefusal(t, CheckIdentity([]byte(specIdentityExample), "codex"), "invalid_config", "names another provider")
		}},
	}
}

// pad2 renders the index zero-padded to two digits for generated
// opaque-identity keys: k00..k32 sort bytewise, so the 33-entry
// fixture trips the count rule rather than key order.
func pad2(index int) string {
	if index < 10 {
		return "0" + string(rune('0'+index))
	}
	return string(rune('0'+index/10)) + string(rune('0'+index%10))
}
