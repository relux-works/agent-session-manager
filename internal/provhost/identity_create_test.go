package provhost

import (
	"bytes"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// This file proves the CreateIdentity production entry: every record
// it emits validates through CheckIdentity, the owner's shape entry,
// and the binding attestation; identical params emit byte-identical
// records; and every refusal names its rule through invalid_config.
// The conformance test at the end drives a created record through the
// landed Host.Call harness, so creation and transport agree on the
// same bytes.

// validCreateParams returns params for a codex session_uuid identity
// with no backend realm: the baseline every creation test mutates.
func validCreateParams() IdentityParams {
	return IdentityParams{
		SessionID:            "0198f4c8-3e70-7a11-8a2b-1234567890ab",
		ProviderID:           "codex",
		ProviderVersion:      "0.147.0",
		ProviderVersionRange: ">=0.147.0 <0.148.0",
		NativeSessionID:      "11111111-2222-4333-8444-555555555555",
		IdentityKind:         "session_uuid",
		LogicalWorkspaceID:   "0198f4c8-6c30-7d44-8d5e-1234567890ab",
		BackendRealm:         "",
		OpaqueIdentity:       map[string]string{},
		CreatedByHostID:      "0198f4c8-4a10-7b22-8b3c-1234567890ab",
		CreatedAt:            "2026-08-19T04:03:00.000Z",
		Extensions:           map[string]any{},
	}
}

func mustCreate(t *testing.T, params IdentityParams) []byte {
	t.Helper()
	record, err := CreateIdentity(params)
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	return record
}

// TestCreateIdentityEmitsAttestedRecord proves a created record is
// accepted everywhere a plugin-returned record is: the host dialect,
// the owner's shape entry, and the binding attestation, which must
// return the claimed record_id itself.
func TestCreateIdentityEmitsAttestedRecord(t *testing.T) {
	record := mustCreate(t, validCreateParams())
	if err := CheckIdentity(record, "codex"); err != nil {
		t.Fatalf("created record fails CheckIdentity: %v", err)
	}
	if _, _, err := canonicaljson.CalculateObjectIdentity(record); err != nil {
		t.Fatalf("created record fails the owner shape entry: %v", err)
	}
	digest, field, err := canonicaljson.VerifyObjectIdentity(record)
	if err != nil {
		t.Fatalf("created record fails binding attestation: %v", err)
	}
	if field != "record_id" {
		t.Fatalf("binding self field = %q, want record_id", field)
	}
	members, fault := decodeStrictObject(record)
	if fault != nil {
		t.Fatalf("created record is not a strict object: %v %q", fault.detail, fault.member)
	}
	claimed, ok := rawString(members["record_id"])
	if !ok || claimed != digest.String() {
		t.Fatalf("created record_id = %q, want the attested digest %q", claimed, digest.String())
	}
}

// TestCreateIdentityAntigravityRecord proves the backend-kind positive
// control: an Antigravity backend_conversation_uuid with a realm, an
// opaque entry, and an extension is created and attested.
func TestCreateIdentityAntigravityRecord(t *testing.T) {
	params := validCreateParams()
	params.ProviderID = "antigravity"
	params.ProviderVersion = "1.1.14"
	params.ProviderVersionRange = ">=1.1.14 <1.2.0"
	params.IdentityKind = "backend_conversation_uuid"
	params.BackendRealm = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"
	params.OpaqueIdentity = map[string]string{"adapter": "agy-cli"}
	params.Extensions = map[string]any{"com.example.adapter": "v1"}
	record := mustCreate(t, params)
	if err := CheckIdentity(record, "antigravity"); err != nil {
		t.Fatalf("created antigravity record fails CheckIdentity: %v", err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(record); err != nil {
		t.Fatalf("created antigravity record fails binding attestation: %v", err)
	}
}

// TestCreateIdentityIsByteIdentical proves creation idempotency: the
// same params emit the same bytes on every call, despite Go map
// iteration order randomizing beneath the marshal.
func TestCreateIdentityIsByteIdentical(t *testing.T) {
	params := validCreateParams()
	params.OpaqueIdentity = map[string]string{"b": "2", "a": "1", "c": "3"}
	params.Extensions = map[string]any{"com.example.b": 2, "com.example.a": 1}
	first := mustCreate(t, params)
	for i := 0; i < 25; i++ {
		if again := mustCreate(t, params); !bytes.Equal(again, first) {
			t.Fatalf("creation %d differs from the first emission", i+1)
		}
	}
}

// TestCreateIdentityPlaceholderDigestInParams proves the record_id
// substitution cannot be misdirected by hostile params: a native ID
// repeating the placeholder digest text, and an opaque value framing
// it as a record_id member, still emit a record whose binding
// attests. The opaque framing attempt marshals with escaped quotes,
// so it never matches the bare-quote substitution frame.
func TestCreateIdentityPlaceholderDigestInParams(t *testing.T) {
	params := validCreateParams()
	params.NativeSessionID = placeholderDigestID
	params.OpaqueIdentity = map[string]string{"trap": `"record_id":"` + placeholderDigestID + `"`}
	record := mustCreate(t, params)
	digest, _, err := canonicaljson.VerifyObjectIdentity(record)
	if err != nil {
		t.Fatalf("hostile params fail binding attestation: %v", err)
	}
	members, fault := decodeStrictObject(record)
	if fault != nil {
		t.Fatalf("created record is not a strict object: %v %q", fault.detail, fault.member)
	}
	claimed, _ := rawString(members["record_id"])
	if claimed != digest.String() {
		t.Fatalf("created record_id = %q, want the attested digest %q", claimed, digest.String())
	}
	if n := bytes.Count(record, []byte(`"record_id":"`)); n != 1 {
		t.Fatalf("created record carries %d record_id frames, want exactly one", n)
	}
}

// TestCreateIdentityRefusals drives every creation refusal through
// the production entry: each row mutates one param of the valid
// baseline and requires the invalid_config arm naming that rule.
func TestCreateIdentityRefusals(t *testing.T) {
	deepExtensions := func() map[string]any {
		return map[string]any{"com.example.deep": map[string]any{"l1": map[string]any{"l2": map[string]any{"l3": map[string]any{"l4": map[string]any{"l5": "x"}}}}}}
	}
	manyExtensions := func() map[string]any {
		out := map[string]any{}
		for i := 0; i < 65; i++ {
			out["com.example.k"+pad2(i)] = i
		}
		return out
	}
	manyOpaque := func() map[string]string {
		out := map[string]string{}
		for i := 0; i < 33; i++ {
			out["k"+pad2(i)] = "x"
		}
		return out
	}
	rows := []struct {
		name   string
		mutate func(*IdentityParams)
		detail string
	}{
		{"session id", func(p *IdentityParams) { p.SessionID = "not-a-uuid" }, "create session_id is not a UUIDv7"},
		{"provider id", func(p *IdentityParams) { p.ProviderID = "Codex" }, "create provider_id is not a provider id"},
		{"version empty", func(p *IdentityParams) { p.ProviderVersion = "" }, "create provider_version is not 1..128 characters"},
		{"version overlong", func(p *IdentityParams) { p.ProviderVersion = strings.Repeat("v", 129) }, "create provider_version is not 1..128 characters"},
		{"range empty", func(p *IdentityParams) { p.ProviderVersionRange = "" }, "create provider_version_range is not 1..256 characters"},
		{"native empty", func(p *IdentityParams) { p.NativeSessionID = "" }, "create native_session_id is not 1..512 characters"},
		{"kind", func(p *IdentityParams) { p.IdentityKind = "window_handle" }, "create identity_kind is not a registry member"},
		{"workspace", func(p *IdentityParams) { p.LogicalWorkspaceID = "0198f4c8-6c30-7d44-8d5e-1234567890aG" }, "create logical_workspace_id is not a UUIDv7"},
		{"realm shape", func(p *IdentityParams) { p.BackendRealm = "nope" }, "create backend_realm_fingerprint is not a digest or empty"},
		{"realm required", func(p *IdentityParams) {
			p.ProviderID = "antigravity"
			p.IdentityKind = "backend_conversation_uuid"
			p.BackendRealm = ""
		}, "create backend_realm_fingerprint is required for this backend kind"},
		{"opaque count", func(p *IdentityParams) { p.OpaqueIdentity = manyOpaque() }, "create opaque_identity exceeds 32 entries"},
		{"opaque key", func(p *IdentityParams) { p.OpaqueIdentity = map[string]string{"Bad Key!": "x"} }, "create opaque key is not a provider key"},
		{"opaque value bounds", func(p *IdentityParams) { p.OpaqueIdentity = map[string]string{"ok": ""} }, "create opaque value is not 1..1024 characters"},
		{"opaque absolute prefix", func(p *IdentityParams) { p.OpaqueIdentity = map[string]string{"workdir": "/Users/iv/work"} }, "create opaque value begins with an absolute path"},
		{"host id", func(p *IdentityParams) { p.CreatedByHostID = "0198f4c8-4a10-7b22-8b3c-1234567890aG" }, "create created_by_host_id is not a UUIDv7"},
		{"timestamp", func(p *IdentityParams) { p.CreatedAt = "yesterday" }, "create created_at is not a timestamp"},
		{"extensions count", func(p *IdentityParams) { p.Extensions = manyExtensions() }, "create extensions exceed 64 entries"},
		{"extensions key", func(p *IdentityParams) { p.Extensions = map[string]any{"NoDNS": true} }, "create extension key is not reverse-DNS"},
		{"extensions value", func(p *IdentityParams) { p.Extensions = map[string]any{"com.example.bad": make(chan int)} }, "create extensions carry a non-JSON value"},
		{"owner backstop", func(p *IdentityParams) { p.Extensions = deepExtensions() }, "create params are not a valid provider identity"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			params := validCreateParams()
			row.mutate(&params)
			_, err := CreateIdentity(params)
			requireLocalRefusal(t, err, "invalid_config", row.detail)
		})
	}
}

// TestCreatedIdentityRoundTripsIdentifyCall proves creation agrees
// with the transport: a created record embedded in an
// identify-session result travels one Host.Call and validates for
// its provider at DecodeIdentifyResult.
func TestCreatedIdentityRoundTripsIdentifyCall(t *testing.T) {
	runner := &scriptRunner{}
	record := mustCreate(t, validCreateParams())
	result := `{"identity": ` + string(record) + `, "confidence": "exact", "matched_evidence": ["native_id"]}`
	body := callWithBody(t, runner, OpIdentifySession, testRequestID, frameFor(t, testRequestID, result))
	if err := DecodeIdentifyResult(body, "codex"); err != nil {
		t.Fatalf("created identity through Call: %v", err)
	}
	if runner.spawned() != 1 {
		t.Fatalf("identify Call spawned %d processes, want one", runner.spawned())
	}
}
