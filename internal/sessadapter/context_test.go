package sessadapter

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// fixtureRequestWithDigest builds a discover request body whose
// context carries the digest of the body itself: the fixpoint the
// host computes (digest over the body with request_digest
// omitted, then written in). The context names the given adapter
// manifest digest so the body binds to a real sealed binding.
func fixtureRequestWithDigest(t *testing.T, manifestDigest string) []byte {
	t.Helper()
	contextWith := func(requestDigest string) string {
		return fmt.Sprintf(`{"operation_id":%q,"provider_id":%q,"environment":%s,"session_adapter_manifest_digest":%q,"executable_sha256":%q,"request_digest":%q,"extensions":{}}`,
			fixtureOperationID, fixtureProviderID, fixtureTupleJSON(), manifestDigest, fixtureExecutableDigest, requestDigest)
	}
	skeleton := fmt.Sprintf(`{"context":%s,"authority":%s,"workspace_filter":null,"limit":50,"cursor":null,"extensions":{}}`,
		contextWith("sha256:"+strings.Repeat("0", 64)), fixtureReadAuthorityJSON())
	digest, err := RequestDigestFor([]byte(skeleton))
	if err != nil {
		t.Fatalf("RequestDigestFor: %v", err)
	}
	return []byte(fmt.Sprintf(`{"context":%s,"authority":%s,"workspace_filter":null,"limit":50,"cursor":null,"extensions":{}}`,
		contextWith(digest.String()), fixtureReadAuthorityJSON()))
}

// fixtureManifestDigestText decodes the fixture manifest and returns
// its host-computed digest text.
func fixtureManifestDigestText(t *testing.T) string {
	t.Helper()
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	return ManifestDigest(manifest).String()
}

func TestRequestDigestFixpoint(t *testing.T) {
	body := fixtureRequestWithDigest(t, fixtureDigest("test-adapter-manifest"))
	context, err := CheckRequestBody(OpDiscover, body)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	// The digest verifies against the sent bytes: writing the
	// digest into the context did not change the digest, because
	// the digested form omits that member.
	if err := VerifyRequestDigest(context, body); err != nil {
		t.Fatalf("VerifyRequestDigest: %v", err)
	}
	// A one-byte change anywhere else breaks the binding.
	mutated := mutateMember(t, body, "limit", `51`)
	if err := VerifyRequestDigest(context, mutated); err == nil {
		t.Fatal("VerifyRequestDigest admitted a changed body under a replayed context")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "request_digest", "does not match")
	}
	// The digest is deterministic: the same canonical body
	// digests identically across calls.
	again, err := RequestDigestFor(body)
	if err != nil {
		t.Fatalf("RequestDigestFor: %v", err)
	}
	if again.String() != context.RequestDigest {
		t.Fatal("request digest is not deterministic")
	}
}

func TestVerifyRequestDigestOmitsOnlyItsMember(t *testing.T) {
	body := fixtureRequestWithDigest(t, fixtureDigest("test-adapter-manifest"))
	context, err := CheckRequestBody(OpDiscover, body)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	// Changing the operation_id changes the digest: the omission
	// covers only request_digest, not its neighbours.
	mutated := []byte(strings.Replace(string(body), fixtureOperationID, "0198f4c8-8e50-7f66-8f70-1234567890aa", 1))
	if err := VerifyRequestDigest(context, mutated); err == nil {
		t.Fatal("VerifyRequestDigest admitted a body with a different operation_id")
	}
}

func TestCheckContextEcho(t *testing.T) {
	body := fixtureRequestWithDigest(t, fixtureDigest("test-adapter-manifest"))
	sent, err := CheckRequestBody(OpDiscover, body)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	// Byte-identical echo passes.
	echoed, err := CheckContextEcho(sent, rawMember(t, body, "context"))
	if err != nil {
		t.Fatalf("CheckContextEcho: %v", err)
	}
	if echoed.OperationID != sent.OperationID {
		t.Fatal("echoed context lost the operation identifier")
	}
	// A re-serialized context with a changed value fails at the
	// echo: decode accepts the valid shape, and only byte
	// identity proves the adapter answered this call.
	altered := mutateMember(t, rawMember(t, body, "context"), "provider_id", `"other-provider"`)
	if _, err := CheckContextEcho(sent, altered); err == nil {
		t.Fatal("CheckContextEcho admitted a context for another provider")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "context", "byte-for-byte")
	}
	// A same-length forgery — one flipped hex digit in the
	// operation identifier, still a valid UUIDv7 — fails at the
	// echo too. A length-only comparison admits it, so this case
	// pins the byte-for-byte half of the gate, not just the
	// length half.
	sameLength := []byte(strings.Replace(string(rawMember(t, body, "context")), fixtureOperationID, "0198f4c8-8e50-7f66-8f70-1234567890ad", 1))
	if len(sameLength) != len(rawMember(t, body, "context")) {
		t.Fatal("same-length forgery changed length; the probe is broken, not the gate")
	}
	if _, err := DecodeCallContext(sameLength); err != nil {
		t.Fatalf("same-length forgery is not a valid context: %v", err)
	}
	if _, err := CheckContextEcho(sent, sameLength); err == nil {
		t.Fatal("CheckContextEcho admitted a same-length rebuilt context")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "context", "byte-for-byte")
	}
	// A malformed context fails at decode, before the echo.
	malformed := mutateMember(t, rawMember(t, body, "context"), "provider_id", `"Other Provider"`)
	if _, err := CheckContextEcho(sent, malformed); err == nil {
		t.Fatal("CheckContextEcho admitted a malformed context")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "provider_id", "not a provider-id")
	}
}

// rawMember extracts one top-level member's raw bytes.
func rawMember(t *testing.T, body []byte, member string) []byte {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(body, &members); err != nil {
		t.Fatalf("rawMember: %v", err)
	}
	raw, present := members[member]
	if !present {
		t.Fatalf("rawMember: %q not in body", member)
	}
	return raw
}

func TestDecodeCallContextRules(t *testing.T) {
	valid := fixtureContextJSON(fixtureDigest("req"))
	if _, err := DecodeCallContext([]byte(valid)); err != nil {
		t.Fatalf("DecodeCallContext: %v", err)
	}
	for _, test := range []struct {
		name   string
		member string
		value  string
		text   string
	}{
		{"bad operation id", "operation_id", `"not-a-uuid"`, "not a UUIDv7"},
		{"bad provider id", "provider_id", `"Test"`, "not a provider-id"},
		{"bad manifest digest", "session_adapter_manifest_digest", `"nope"`, "not a digest"},
		{"bad executable digest", "executable_sha256", `"nope"`, "not a digest"},
		{"bad request digest", "request_digest", `"nope"`, "not a digest"},
		{"bad extensions", "extensions", `{"x":"y"}`, "not reverse-DNS keyed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeCallContext(mutateMember(t, []byte(valid), test.member, test.value))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", test.member, test.text)
		})
	}
	t.Run("missing member", func(t *testing.T) {
		_, err := DecodeCallContext(dropMember(t, []byte(valid), "environment"))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "environment", "misses a required member")
	})
	t.Run("unknown member", func(t *testing.T) {
		_, err := DecodeCallContext(appendMember(t, []byte(valid), "unexpected", `1`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "unexpected", "unknown member")
	})
}

func TestDecodeReadAuthorityRules(t *testing.T) {
	valid := fixtureReadAuthorityJSON()
	if _, err := DecodeReadAuthority([]byte(valid)); err != nil {
		t.Fatalf("DecodeReadAuthority: %v", err)
	}
	for _, purpose := range []string{"source_native", "target_staged", "target_live"} {
		if _, err := DecodeReadAuthority(mutateMember(t, []byte(valid), "purpose", `"`+purpose+`"`)); err != nil {
			t.Fatalf("DecodeReadAuthority(%q): %v", purpose, err)
		}
	}
	for _, test := range []struct {
		name   string
		member string
		value  string
		text   string
	}{
		{"bad purpose", "purpose", `"live_store"`, "outside source_native|target_staged|target_live"},
		{"empty handles", "root_handle_names", `[]`, "not sorted unique"},
		{"unsorted handles", "root_handle_names", `["b","a"]`, "not sorted unique"},
		{"duplicate handles", "root_handle_names", `["a","a"]`, "not sorted unique"},
		{"empty handle", "root_handle_names", `[""]`, "not sorted unique"},
		{"bad authority id", "authority_id", `"x"`, "not a UUIDv7"},
		{"bad expiry", "expires_at", `"never"`, "not a timestamp"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeReadAuthority(mutateMember(t, []byte(valid), test.member, test.value))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", test.member, test.text)
		})
	}
}

func TestDecodeObjectAuthorityRules(t *testing.T) {
	for _, purpose := range []string{"capture_plan", "raw_source", "canonical_source", "projection_plan", "projected_target", "read_back_evidence"} {
		if _, err := DecodeObjectAuthority([]byte(fixtureFreshSinkJSON(purpose))); err != nil {
			t.Fatalf("DecodeObjectAuthority(%q): %v", purpose, err)
		}
	}
	valid := fixtureFreshSinkJSON("capture_plan")
	for _, test := range []struct {
		name       string
		member     string
		value      string
		wantMember string
		text       string
	}{
		{"bad purpose", "purpose", `"live_store"`, "purpose", "six-purpose vocabulary"},
		{"bad mode", "mode", `"append"`, "mode", "outside read|fresh_sink"},
		{"zero objects on fresh sink", "max_objects", `0`, "mode", "both limits above zero"},
		{"zero bytes on fresh sink", "max_total_bytes", `0`, "mode", "both limits above zero"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeObjectAuthority(mutateMember(t, []byte(valid), test.member, test.value))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", test.wantMember, test.text)
		})
	}
	// Read mode exposes only request-named objects: zero limits
	// are admissible there because the authority grants no sink.
	read := mutateMember(t, []byte(valid), "mode", `"read"`)
	read = mutateMember(t, read, "max_objects", `0`)
	if _, err := DecodeObjectAuthority(read); err != nil {
		t.Fatalf("DecodeObjectAuthority read with zero objects: %v", err)
	}
	// The fresh sink must be empty: a reused sink under a
	// fresh_sink authority is refused, and read mode takes no
	// sink claim either way.
	authority, err := DecodeObjectAuthority([]byte(valid))
	if err != nil {
		t.Fatalf("DecodeObjectAuthority: %v", err)
	}
	if err := CheckFreshSink(authority, true); err != nil {
		t.Fatalf("CheckFreshSink empty: %v", err)
	}
	err = CheckFreshSink(authority, false)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "mode", "not empty")
	readAuthority, err := DecodeObjectAuthority(read)
	if err != nil {
		t.Fatalf("DecodeObjectAuthority: %v", err)
	}
	if err := CheckFreshSink(readAuthority, false); err != nil {
		t.Fatalf("CheckFreshSink read mode: %v", err)
	}
}

// TestDecodeSourceSelectorComplement drives the complete domain:
// zero, one, two, and three non-null members. Exactly one admits;
// every other count refuses. There is no fifth case to sample.
func TestDecodeSourceSelectorComplement(t *testing.T) {
	base := `{"native_session_id":null,"logical_workspace_id":null,"opaque_source_ref":null}`
	cases := []struct {
		name  string
		body  string
		count int
	}{
		{"none", base, 0},
		{"native only", `{"native_session_id":"sess-1","logical_workspace_id":null,"opaque_source_ref":null}`, 1},
		{"workspace only", `{"native_session_id":null,"logical_workspace_id":"0198f4c8-8e50-7f66-8f70-1234567890ab","opaque_source_ref":null}`, 1},
		{"opaque only", `{"native_session_id":null,"logical_workspace_id":null,"opaque_source_ref":"ref-1"}`, 1},
		{"native plus workspace", `{"native_session_id":"sess-1","logical_workspace_id":"0198f4c8-8e50-7f66-8f70-1234567890ab","opaque_source_ref":null}`, 2},
		{"native plus opaque", `{"native_session_id":"sess-1","logical_workspace_id":null,"opaque_source_ref":"ref-1"}`, 2},
		{"workspace plus opaque", `{"native_session_id":null,"logical_workspace_id":"0198f4c8-8e50-7f66-8f70-1234567890ab","opaque_source_ref":"ref-1"}`, 2},
		{"all three", `{"native_session_id":"sess-1","logical_workspace_id":"0198f4c8-8e50-7f66-8f70-1234567890ab","opaque_source_ref":"ref-1"}`, 3},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			selector, err := DecodeSourceSelector([]byte(test.body))
			if test.count == 1 {
				if err != nil {
					t.Fatalf("DecodeSourceSelector: %v", err)
				}
				named := 0
				if selector.NativeSessionID != nil {
					named++
				}
				if selector.LogicalWorkspaceID != nil {
					named++
				}
				if selector.OpaqueSourceRef != nil {
					named++
				}
				if named != 1 {
					t.Fatalf("selector names %d sources, want 1", named)
				}
				return
			}
			if err == nil {
				t.Fatalf("DecodeSourceSelector admitted %d sources", test.count)
			}
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "native_session_id", "exactly one source")
		})
	}
	// Malformed members refuse with the member named: an empty
	// native ID and a non-UUID workspace ID.
	if _, err := DecodeSourceSelector([]byte(`{"native_session_id":"","logical_workspace_id":null,"opaque_source_ref":null}`)); err == nil {
		t.Fatal("admitted an empty native session identifier")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "native_session_id", "string[1..512]")
	}
	if _, err := DecodeSourceSelector([]byte(`{"native_session_id":null,"logical_workspace_id":"nope","opaque_source_ref":null}`)); err == nil {
		t.Fatal("admitted a non-UUID workspace identifier")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "logical_workspace_id", "not a UUIDv7")
	}
}

// TestDecodeResourceLimitsBounds sweeps every numeric bound on
// both edges: below-minimum, minimum, maximum, above-maximum.
// The vectors are generated, not listed, and the per-item rule is
// attacked from both directions.
func TestDecodeResourceLimitsBounds(t *testing.T) {
	valid := func() string {
		return `{"max_objects":10,"max_total_bytes":1000,"max_single_object_bytes":100,"max_events":5,"max_target_resources":5}`
	}
	if _, err := DecodeResourceLimits([]byte(valid())); err != nil {
		t.Fatalf("DecodeResourceLimits: %v", err)
	}
	bounds := []struct {
		member  string
		minimum uint64
		maximum uint64
		current uint64
	}{
		{"max_objects", 1, 9007199254740991, 10},
		{"max_total_bytes", 1, 9007199254740991, 1000},
		{"max_single_object_bytes", 1, 9007199254740991, 100},
		{"max_events", 0, 65536, 5},
		{"max_target_resources", 0, 65536, 5},
	}
	setMember := func(body, member string, value uint64) []byte {
		return mutateMember(t, []byte(body), member, fmt.Sprintf(`%d`, value))
	}
	for _, bound := range bounds {
		for _, test := range []struct {
			name  string
			value uint64
			ok    bool
		}{
			{"below minimum", bound.minimum - 1, false},
			{"minimum", bound.minimum, true},
			{"maximum", bound.maximum, true},
			{"above maximum", bound.maximum + 1, false},
		} {
			// Skip the underflow case: minimum 0 has no
			// below-minimum uint53.
			if bound.minimum == 0 && test.name == "below minimum" {
				continue
			}
			t.Run(bound.member+" "+test.name, func(t *testing.T) {
				body := valid()
				// Keep the per-item/total coherence while
				// sweeping: raise the total when the
				// swept member would otherwise exceed it.
				if (bound.member == "max_objects" || bound.member == "max_single_object_bytes") && test.value > 1000 {
					body = string(setMember(body, "max_total_bytes", test.value))
				}
				if bound.member == "max_total_bytes" && test.value < 100 {
					body = string(setMember(body, "max_single_object_bytes", test.value))
					body = string(setMember(body, "max_objects", minU64(test.value, 10)))
				}
				_, err := DecodeResourceLimits(setMember(body, bound.member, test.value))
				if test.ok && err != nil {
					t.Fatalf("DecodeResourceLimits: %v", err)
				}
				if !test.ok && err == nil {
					t.Fatalf("DecodeResourceLimits admitted %s=%d", bound.member, test.value)
				}
			})
		}
	}
	// Non-integral numbers are not uint53 even at the same
	// magnitude.
	for _, literal := range []string{`10.0`, `"10"`, `1e2`, `-1`, `true`, `null`} {
		_, err := DecodeResourceLimits(mutateMember(t, []byte(valid()), "max_objects", literal))
		if err == nil {
			t.Fatalf("DecodeResourceLimits admitted max_objects=%s", literal)
		}
	}
	// The per-item rule fails in both directions: a single-object
	// maximum above the total, and an object count above the
	// total, each refuse.
	_, err := DecodeResourceLimits([]byte(`{"max_objects":10,"max_total_bytes":100,"max_single_object_bytes":101,"max_events":0,"max_target_resources":0}`))
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "max_single_object_bytes", "exceeds the total")
	_, err = DecodeResourceLimits([]byte(`{"max_objects":101,"max_total_bytes":100,"max_single_object_bytes":100,"max_events":0,"max_target_resources":0}`))
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "max_objects", "exceeds the total")
}

func minU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func TestDecodeFindingRules(t *testing.T) {
	valid := `{"severity":"warning","code":"cap-1","message":"a warning","remediation":null,"extensions":{}}`
	if _, err := DecodeFinding([]byte(valid)); err != nil {
		t.Fatalf("DecodeFinding: %v", err)
	}
	for _, severity := range []string{"info", "warning", "error"} {
		if _, err := DecodeFinding(mutateMember(t, []byte(valid), "severity", `"`+severity+`"`)); err != nil {
			t.Fatalf("DecodeFinding(%q): %v", severity, err)
		}
	}
	for _, test := range []struct {
		name   string
		member string
		value  string
		text   string
	}{
		{"bad severity", "severity", `"fatal"`, "outside info|warning|error"},
		{"empty code", "code", `""`, "not a string[1..128]"},
		{"empty message", "message", `""`, "not a string[1..4096]"},
		{"empty remediation", "remediation", `""`, "not a string[1..4096]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeFinding(mutateMember(t, []byte(valid), test.member, test.value))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", test.member, test.text)
		})
	}
	if _, err := DecodeFinding(mutateMember(t, []byte(valid), "remediation", `"restart the source"`)); err != nil {
		t.Fatalf("DecodeFinding with remediation: %v", err)
	}
	if _, err := DecodeFindings([]byte(`[`+valid+`,`+valid+`]`), 4096); err != nil {
		t.Fatalf("DecodeFindings: %v", err)
	}
	if _, err := DecodeFindings([]byte(`[`+valid+`]`), 0); err == nil {
		t.Fatal("DecodeFindings admitted an entry above the count bound")
	}
}

func TestDecodeCapturePlanItemRules(t *testing.T) {
	valid := `{"native_item_key":"item-1","class":"durable_payload","byte_count":12,"required":true,"extensions":{}}`
	if _, err := DecodeCapturePlanItem([]byte(valid)); err != nil {
		t.Fatalf("DecodeCapturePlanItem: %v", err)
	}
	// All nine classes admit; a tenth does not.
	for _, class := range captureClasses {
		if _, err := DecodeCapturePlanItem(mutateMember(t, []byte(valid), "class", `"`+class+`"`)); err != nil {
			t.Fatalf("DecodeCapturePlanItem(%q): %v", class, err)
		}
	}
	_, err := DecodeCapturePlanItem(mutateMember(t, []byte(valid), "class", `"archive_only"`))
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "class", "nine-class vocabulary")
	if _, err := DecodeCapturePlanItem(mutateMember(t, []byte(valid), "byte_count", `null`)); err != nil {
		t.Fatalf("DecodeCapturePlanItem null byte count: %v", err)
	}
}
