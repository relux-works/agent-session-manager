package sessadapter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// fixtureRequestFrame builds one valid request frame for the
// operation with the probe-shaped body replaced by the caller.
func fixtureRequestFrame(t *testing.T, operation Operation, body string) []byte {
	t.Helper()
	return []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":%q,"deadline":%q,"body":%s}`,
		ProtocolID, fixtureRequestID, string(operation), fixtureDeadline, body))
}

// wantRequest is the Request a valid doctor frame decodes to.
func wantRequest() Request {
	return Request{Operation: OpDoctor, RequestID: fixtureRequestID}
}

func TestDecodeRequestFrameAcceptsValidEnvelope(t *testing.T) {
	frame := fixtureRequestFrame(t, OpDoctor, `{"context":{},"direction":"source_read","tuple_registry_digest":"`+fixtureRegistryDigest+`","refresh_requested":false,"extensions":{}}`)
	got, err := DecodeRequestFrame(frame)
	if err != nil {
		t.Fatalf("DecodeRequestFrame: %v", err)
	}
	if got.Operation != OpDoctor || got.RequestID != fixtureRequestID {
		t.Fatalf("DecodeRequestFrame = %+v, want doctor/%s", got, fixtureRequestID)
	}
}

func TestDecodeRequestFrameRefusals(t *testing.T) {
	validBody := `{"expected_provider_id":"test-provider","expected_candidate_kind":"builtin","extensions":{}}`
	for _, test := range []struct {
		name  string
		frame func() []byte
		code  axerror.Code
		key   string
		value string
		text  string
	}{
		{
			name:  "empty frame",
			frame: func() []byte { return nil },
			code:  "session_adapter_protocol_error", key: "member", value: "", text: "request frame is empty",
		},
		{
			name:  "non object frame",
			frame: func() []byte { return []byte(`[1,2]`) },
			code:  "session_adapter_protocol_error", key: "member", value: "", text: "request envelope not a JSON object",
		},
		{
			name: "unknown envelope member",
			frame: func() []byte {
				return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "unexpected", `true`)
			},
			code: "session_adapter_protocol_error", key: "member", value: "unexpected", text: "unknown member",
		},
		{
			name:  "missing operation",
			frame: func() []byte { return dropMember(t, fixtureRequestFrame(t, OpProbe, validBody), "operation") },
			code:  "session_adapter_protocol_error", key: "member", value: "operation", text: "misses a required member",
		},
		{
			name: "wrong protocol",
			frame: func() []byte {
				return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "protocol", `"urn:ax:protocol:provider"`)
			},
			code: "session_adapter_protocol_error", key: "member", value: "protocol", text: "not the session adapter",
		},
		{
			name: "wrong version",
			frame: func() []byte {
				return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "protocol_version", `"2.0.0"`)
			},
			code: "session_adapter_protocol_error", key: "member", value: "protocol_version", text: "not 1.0.0",
		},
		{
			name: "non uuid request id",
			frame: func() []byte {
				return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "request_id", `"not-a-uuid"`)
			},
			code: "session_adapter_protocol_error", key: "member", value: "request_id", text: "not a UUIDv7",
		},
		{
			name: "unknown operation",
			frame: func() []byte {
				return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "operation", `"launch"`)
			},
			code: "operation_unknown", key: "operation", value: "launch", text: "outside the closed registry",
		},
		{
			name:  "empty operation",
			frame: func() []byte { return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "operation", `""`) },
			code:  "operation_unknown", key: "operation", value: "", text: "outside the closed registry",
		},
		{
			name: "bad deadline",
			frame: func() []byte {
				return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "deadline", `"yesterday"`)
			},
			code: "session_adapter_protocol_error", key: "member", value: "deadline", text: "not a timestamp",
		},
		{
			name:  "non object body",
			frame: func() []byte { return mutateMember(t, fixtureRequestFrame(t, OpProbe, validBody), "body", `[]`) },
			code:  "session_adapter_protocol_error", key: "member", value: "body", text: "not a JSON object",
		},
		{
			name: "duplicate member",
			frame: func() []byte {
				return []byte(`{"protocol":"urn:ax:protocol:session-adapter","protocol_version":"1.0.0","request_id":"` + fixtureRequestID + `","operation":"probe","operation":"probe","deadline":"` + fixtureDeadline + `","body":{}}`)
			},
			code: "session_adapter_protocol_error", key: "member", value: "operation", text: "duplicate member",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeRequestFrame(test.frame())
			requireRefusal(t, err, test.code, test.key, test.value, test.text)
		})
	}
}

func TestDecodeRequestFrameRefusesOversizeFrame(t *testing.T) {
	padding := strings.Repeat("a", MaxFrameBytes)
	frame := fixtureRequestFrame(t, OpProbe, `{"expected_provider_id":"`+padding+`","expected_candidate_kind":"builtin","extensions":{}}`)
	if len(frame) <= MaxFrameBytes {
		t.Fatalf("oversize fixture is %d bytes, want above %d", len(frame), MaxFrameBytes)
	}
	_, err := DecodeRequestFrame(frame)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "exceeds the 8 MiB bound")
}

// TestFrameBoundEdges pins the 8 MiB bound at exactly
// MaxFrameBytes (admitted past the length gate) and
// MaxFrameBytes+1 (refused) for all three envelope entry
// points. A frame of exactly the bound must fail on content,
// never on length; one byte more must fail on length. Either
// off-by-one — `> MaxFrameBytes+1` admitting the neighbour, or
// `>= MaxFrameBytes` refusing the edge — reddens here.
func TestFrameBoundEdges(t *testing.T) {
	padTo := func(size int) []byte {
		base := []byte(`{"pad":""}`)
		if size < len(base) {
			t.Fatalf("bound %d is below the %d-byte probe skeleton", size, len(base))
		}
		probe := []byte(`{"pad":"` + strings.Repeat("a", size-len(base)) + `"}`)
		if len(probe) != size {
			t.Fatalf("padded probe is %d bytes, want %d", len(probe), size)
		}
		return probe
	}
	exact, over := padTo(MaxFrameBytes), padTo(MaxFrameBytes+1)
	for _, gate := range []struct {
		name  string
		check func(frame []byte) error
	}{
		{"request", func(frame []byte) error { _, err := DecodeRequestFrame(frame); return err }},
		{"success", func(frame []byte) error { _, err := CheckSuccessEnvelope(frame, wantRequest()); return err }},
		{"failure", func(frame []byte) error { _, err := CheckFailureEnvelope(frame, wantRequest()); return err }},
	} {
		t.Run(gate.name+"/exact bound admits past length", func(t *testing.T) {
			err := gate.check(exact)
			if err == nil {
				t.Fatal("exact-bound probe passed every gate; the probe is broken, not the bound")
			}
			if strings.Contains(err.Error(), "exceeds the 8 MiB bound") {
				t.Fatalf("exact-bound frame refused on length: %v", err)
			}
		})
		t.Run(gate.name+"/one over refuses on length", func(t *testing.T) {
			err := gate.check(over)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "exceeds the 8 MiB bound")
		})
	}
}

func TestCheckSuccessEnvelopeAcceptsValidFrame(t *testing.T) {
	body := `{"direction":"source_read"}`
	frame := []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":true,"body":%s}`,
		ProtocolID, fixtureRequestID, body))
	got, err := CheckSuccessEnvelope(frame, wantRequest())
	if err != nil {
		t.Fatalf("CheckSuccessEnvelope: %v", err)
	}
	if string(got) != body {
		t.Fatalf("CheckSuccessEnvelope body = %s, want %s", got, body)
	}
}

func TestCheckSuccessEnvelopeRefusals(t *testing.T) {
	good := func() []byte {
		return []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":true,"body":{}}`,
			ProtocolID, fixtureRequestID))
	}
	for _, test := range []struct {
		name  string
		frame func() []byte
		key   string
		value string
		text  string
	}{
		{
			name:  "error member on success",
			frame: func() []byte { return mutateMember(t, good(), "error", `{"code":"x"}`) },
			key:   "member", value: "error", text: "unknown member",
		},
		{
			name:  "missing body",
			frame: func() []byte { return dropMember(t, good(), "body") },
			key:   "member", value: "body", text: "misses a required member",
		},
		{
			name:  "request id mismatch",
			frame: func() []byte { return mutateMember(t, good(), "request_id", `"0198f4c8-8e50-7f66-8f70-1234567890aa"`) },
			key:   "member", value: "request_id", text: "does not echo",
		},
		{
			name:  "operation mismatch",
			frame: func() []byte { return mutateMember(t, good(), "operation", `"probe"`) },
			key:   "member", value: "operation", text: "does not echo",
		},
		{
			name:  "protocol mismatch",
			frame: func() []byte { return mutateMember(t, good(), "protocol", `"urn:ax:protocol:provider"`) },
			key:   "member", value: "protocol", text: "does not echo",
		},
		{
			name:  "version mismatch",
			frame: func() []byte { return mutateMember(t, good(), "protocol_version", `"9.9.9"`) },
			key:   "member", value: "protocol_version", text: "does not echo",
		},
		{
			name:  "ok false",
			frame: func() []byte { return mutateMember(t, good(), "ok", `false`) },
			key:   "member", value: "ok", text: "does not carry ok=true",
		},
		{
			name:  "null body",
			frame: func() []byte { return mutateMember(t, good(), "body", `null`) },
			key:   "member", value: "body", text: "not a JSON object",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := CheckSuccessEnvelope(test.frame(), wantRequest())
			requireRefusal(t, err, "session_adapter_protocol_error", test.key, test.value, test.text)
		})
	}
}

func TestCheckFailureEnvelopeAcceptsBoundError(t *testing.T) {
	frame := []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.1.0","code":"capability_unavailable","message":"the adapter build cannot enumerate this source","exit_code":6,"retryable":false,"details":{}}}`,
		ProtocolID, fixtureRequestID))
	reported, err := CheckFailureEnvelope(frame, wantRequest())
	if err != nil {
		t.Fatalf("CheckFailureEnvelope: %v", err)
	}
	if reported.Code() != "capability_unavailable" || reported.ExitCode() != 6 {
		t.Fatalf("reported = %s/%d, want capability_unavailable/6", reported.Code(), reported.ExitCode())
	}
}

func TestCheckFailureEnvelopeRefusals(t *testing.T) {
	good := func() []byte {
		return []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.1.0","code":"capability_unavailable","message":"the adapter build cannot enumerate this source","exit_code":6,"retryable":false,"details":{}}}`,
			ProtocolID, fixtureRequestID))
	}
	for _, test := range []struct {
		name  string
		frame func() []byte
		code  axerror.Code
		key   string
		value string
		text  string
	}{
		{
			name:  "body member on failure",
			frame: func() []byte { return mutateMember(t, good(), "body", `{}`) },
			code:  "session_adapter_protocol_error", key: "member", value: "body", text: "unknown member",
		},
		{
			name:  "ok true",
			frame: func() []byte { return mutateMember(t, good(), "ok", `true`) },
			code:  "session_adapter_protocol_error", key: "member", value: "ok", text: "does not carry ok=false",
		},
		{
			name:  "request id mismatch",
			frame: func() []byte { return mutateMember(t, good(), "request_id", `"0198f4c8-8e50-7f66-8f70-1234567890aa"`) },
			code:  "session_adapter_protocol_error", key: "member", value: "request_id", text: "does not echo",
		},
		{
			name: "wrong error version",
			frame: func() []byte {
				return mutateMember(t, good(), "error", `{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"the adapter build cannot enumerate this source","exit_code":6,"retryable":false,"details":{}}`)
			},
			code: "integrity_failure", key: "subject", value: "error", text: "not a Structured Error 1.1",
		},
		{
			name:  "non object error",
			frame: func() []byte { return mutateMember(t, good(), "error", `null`) },
			code:  "integrity_failure", key: "subject", value: "error", text: "not a Structured Error 1.1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := CheckFailureEnvelope(test.frame(), wantRequest())
			if test.key == "" {
				if failureCode(t, err) != test.code {
					t.Fatalf("code = %v, want %s", err, test.code)
				}
				if !strings.Contains(err.Error(), test.text) {
					t.Fatalf("error = %v, want detail containing %q", err, test.text)
				}
				return
			}
			requireRefusal(t, err, test.code, test.key, test.value, test.text)
		})
	}
}
