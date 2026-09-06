package dirnode

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// TestDecodeRequestFrameAcceptsBothMajors drives the production
// request entry for each supported major with an exact contract
// vector: the decoded major, operation, identifier, and body must
// match the frame.
func TestDecodeRequestFrameAcceptsBothMajors(t *testing.T) {
	t.Parallel()
	for _, probe := range []struct {
		version string
		major   int
	}{
		{"1.0.0", MajorV1},
		{"2.0.0", MajorV2},
	} {
		frame := fixtureRequestFrame(probe.version, "manifest", fixtureRequestID, 5000, `{}`)
		request, err := DecodeRequestFrame(frame)
		if err != nil {
			t.Fatalf("DecodeRequestFrame(%s) error = %v", probe.version, err)
		}
		if request.Major != probe.major || request.Operation != OpManifest || request.RequestID != fixtureRequestID {
			t.Fatalf("DecodeRequestFrame(%s) = %+v, want major %d manifest %s", probe.version, request, probe.major, fixtureRequestID)
		}
		if string(request.Body) != `{}` {
			t.Fatalf("DecodeRequestFrame(%s) body = %q, want {}", probe.version, request.Body)
		}
	}
}

// TestDecodeRequestFrameRefusals drives the production request
// entry with one negative vector per envelope rule. Each vector
// names the production call site (DecodeRequestFrame) and the
// expected code; the detail member is asserted only where the
// envelope names one deterministically.
func TestDecodeRequestFrameRefusals(t *testing.T) {
	t.Parallel()
	oversize := append(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`), make([]byte, frameBoundBytes)...)
	for _, probe := range []struct {
		name   string
		frame  []byte
		code   axerror.Code
		member string
	}{
		{"empty frame", []byte{}, "adapter_protocol_violation", ""},
		{"oversize frame", oversize, "adapter_protocol_violation", ""},
		{"non object", []byte(`[]`), "adapter_protocol_violation", ""},
		{"duplicate member", []byte(`{"schema":"urn:ax:schema:session-directory-node-request","schema_version":"2.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","deadline_ms":1,"body":{},"body":{}}`), "adapter_protocol_violation", "body"},
		{"unknown member", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), `"deadline_ms":1`, `"deadline_ms":1,"extra":1`, 1)), "adapter_protocol_violation", "extra"},
		{"missing member", []byte(`{"schema":"urn:ax:schema:session-directory-node-request","schema_version":"2.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","body":{}}`), "adapter_protocol_violation", "deadline_ms"},
		{"wrong schema", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), "session-directory-node-request", "session-directory-node-response", 1)), "adapter_protocol_violation", "schema"},
		{"non string protocol version", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), `"protocol_version":"2.0.0"`, `"protocol_version":2`, 1)), "adapter_protocol_violation", "protocol_version"},
		{"non string schema version", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), `"schema_version":"2.0.0"`, `"schema_version":2`, 1)), "adapter_protocol_violation", "schema_version"},
		{"wrong protocol", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), `"protocol":"urn:ax:protocol:session-directory-node"`, `"protocol":"urn:ax:protocol:provider"`, 1)), "adapter_protocol_violation", "protocol"},
		{"non string request id", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), `"request_id":"`+fixtureRequestID+`"`, `"request_id":7`, 1)), "adapter_protocol_violation", "request_id"},
		{"non string operation", []byte(replaceOnce(t, string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`)), `"operation":"manifest"`, `"operation":7`, 1)), "adapter_protocol_violation", "operation"},
		{"relabeled major pair", fixtureRequestFrame("1.0.0", "manifest", fixtureRequestID, 1, `{}`)[:0:0], "adapter_protocol_violation", ""},
		{"unknown operation", fixtureRequestFrame("2.0.0", "query", fixtureRequestID, 1, `{}`), "operation_unknown", ""},
		{"bad request id", fixtureRequestFrame("2.0.0", "manifest", "not-a-uuid", 1, `{}`), "adapter_protocol_violation", "request_id"},
		{"zero deadline", fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 0, `{}`), "adapter_protocol_violation", "deadline_ms"},
		{"deadline past bound", fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 3600001, `{}`), "adapter_protocol_violation", "deadline_ms"},
		{"non object body", fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `[]`), "adapter_protocol_violation", "body"},
	} {
		frame := probe.frame
		if probe.name == "relabeled major pair" {
			raw := string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
			raw = replaceOnce(t, raw, `"protocol_version":"2.0.0"`, `"protocol_version":"1.0.0"`, 1)
			frame = []byte(raw)
		}
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeRequestFrame(frame)
			failure := requireCode(t, err, probe.code)
			if probe.member != "" {
				detail, _ := failure.Detail("member")
				if detail != probe.member {
					t.Fatalf("refusal detail member = %v, want %q", detail, probe.member)
				}
			}
		})
	}
}

// TestEncodeRequestRoundTrip drives the production request builder
// and requires its output to validate through the production
// decoder with identical fields.
func TestEncodeRequestRoundTrip(t *testing.T) {
	t.Parallel()
	for _, major := range SupportedMajors() {
		frame, err := EncodeRequest(major, OpProbe, fixtureRequestID, 1000, []byte(fixtureProbeRequestV2()))
		if err != nil {
			t.Fatalf("EncodeRequest(%d) error = %v", major, err)
		}
		request, err := DecodeRequestFrame(frame)
		if err != nil {
			t.Fatalf("DecodeRequestFrame(EncodeRequest(%d)) error = %v", major, err)
		}
		if request.Major != major || request.Operation != OpProbe || request.RequestID != fixtureRequestID {
			t.Fatalf("round trip(%d) = %+v", major, request)
		}
	}
}

// TestEncodeRequestRefusals drives the production builder with
// caller defects: an unsupported major is invalid_config, an
// unknown operation is operation_unknown, and bound defects refuse
// before any byte is emitted.
func TestEncodeRequestRefusals(t *testing.T) {
	t.Parallel()
	if _, err := EncodeRequest(3, OpManifest, fixtureRequestID, 1, []byte(`{}`)); requireCode(t, err, "invalid_config") == nil {
		t.Fatal("EncodeRequest(3) must refuse")
	}
	if _, err := EncodeRequest(MajorV2, "query", fixtureRequestID, 1, []byte(`{}`)); requireCode(t, err, "operation_unknown") == nil {
		t.Fatal("EncodeRequest(query) must refuse")
	}
	if _, err := EncodeRequest(MajorV2, OpManifest, "bad", 1, []byte(`{}`)); requireCode(t, err, "invalid_config") == nil {
		t.Fatal("EncodeRequest(bad id) must refuse")
	}
	if _, err := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, 0, []byte(`{}`)); requireCode(t, err, "invalid_config") == nil {
		t.Fatal("EncodeRequest(zero deadline) must refuse")
	}
	if _, err := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, 3600001, []byte(`{}`)); requireCode(t, err, "invalid_config") == nil {
		t.Fatal("EncodeRequest(deadline past bound) must refuse")
	}
	if _, err := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, 1, []byte(`[]`)); requireCode(t, err, "adapter_protocol_violation") == nil {
		t.Fatal("EncodeRequest(non-object body) must refuse")
	}
}

// TestDecodeRequestFrameStrictFaults drives the production
// request entry through the strict-decoder faults: invalid
// UTF-8, a lone surrogate escape, trailing data, and a lone
// top-level value are faults, never partial objects.
func TestDecodeRequestFrameStrictFaults(t *testing.T) {
	t.Parallel()
	good := string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
	for _, probe := range []struct {
		name  string
		frame []byte
	}{
		{"invalid utf8", []byte{0xff, 0xfe}},
		{"lone surrogate", []byte(replaceOnce(t, good, `"manifest"`, `mani\ud800fest`, 1))},
		{"trailing data", []byte(good + " {}")},
		{"empty after trim", []byte("   ")},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeRequestFrame(probe.frame); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestCheckSuccessEnvelopeAcceptsExactEcho drives the production
// success entry with an exact contract vector.
func TestCheckSuccessEnvelopeAcceptsExactEcho(t *testing.T) {
	t.Parallel()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	body, err := CheckSuccessEnvelope(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()), request)
	if err != nil {
		t.Fatalf("CheckSuccessEnvelope error = %v", err)
	}
	if _, err := DecodeManifest(body); err != nil {
		t.Fatalf("DecodeManifest(success body) error = %v", err)
	}
}

// TestCheckSuccessEnvelopeRefusals drives the production success
// entry with one negative vector per branch rule.
func TestCheckSuccessEnvelopeRefusals(t *testing.T) {
	t.Parallel()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	manifest := fixtureManifestJSON()
	for _, probe := range []struct {
		name  string
		frame []byte
	}{
		{"wrong protocol version echo", fixtureSuccessFrame("1.0.0", "manifest", fixtureRequestID, manifest)},
		{"wrong request echo", fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID[:len(fixtureRequestID)-1]+"0", manifest)},
		{"wrong operation echo", fixtureSuccessFrame("2.0.0", "probe", fixtureRequestID, manifest)},
		{"relabeled response schema", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1))},
		{"ok false", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `"ok":true`, `"ok":false`, 1))},
		{"both branches", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `"ok":true,`, `"ok":true,"error":`+fixtureErrorObject("incompatible_protocol", 6, false)+`,`, 1))},
		{"neither branch", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `,"body":`, `,"nobody":`, 1))},
		{"non object body", fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, `[]`)},
		{"unknown member", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `"ok":true`, `"ok":true,"extra":1`, 1))},
		{"missing member", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `,"ok":true`, ``, 1))},
		{"wrong schema", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), "session-directory-node-response", "session-directory-node-request", 1))},
		{"wrong protocol", []byte(replaceOnce(t, string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, manifest)), `"protocol":"urn:ax:protocol:session-directory-node"`, `"protocol":"urn:ax:protocol:provider"`, 1))},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckSuccessEnvelope(probe.frame, request); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestCheckFailureEnvelopeAcceptsExactEcho drives the production
// failure entry and requires the decoded Structured Error to
// carry the exact code, exit, and retryable bit.
func TestCheckFailureEnvelopeAcceptsExactEcho(t *testing.T) {
	t.Parallel()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	decoded, err := CheckFailureEnvelope(fixtureDowngradeFrame("2.0.0", fixtureRequestID), request)
	if err != nil {
		t.Fatalf("CheckFailureEnvelope error = %v", err)
	}
	if decoded.Code() != "incompatible_protocol" || decoded.ExitCode() != 6 || decoded.Retryable() {
		t.Fatalf("decoded downgrade = %q exit %d retryable %v", decoded.Code(), decoded.ExitCode(), decoded.Retryable())
	}
}

// TestCheckFailureEnvelopeRefusals drives the production failure
// entry with one negative vector per branch rule: a failure that
// cannot be read as Structured Error 1.2.0 is an integrity
// failure, never a retryable result.
func TestCheckFailureEnvelopeRefusals(t *testing.T) {
	t.Parallel()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
	for _, probe := range []struct {
		name  string
		frame []byte
		code  axerror.Code
	}{
		{"ok true", []byte(replaceOnce(t, downgrade, `"ok":false`, `"ok":true`, 1)), "adapter_protocol_violation"},
		{"both branches", []byte(replaceOnce(t, downgrade, `"ok":false,`, `"ok":false,"body":{},`, 1)), "adapter_protocol_violation"},
		{"neither branch", []byte(replaceOnce(t, downgrade, `,"error":`, `,"nerror":`, 1)), "adapter_protocol_violation"},
		{"wrong version echo", fixtureFailureFrame("1.0.0", "manifest", fixtureRequestID, fixtureErrorObject("incompatible_protocol", 6, false)), "adapter_protocol_violation"},
		{"unreadable error", fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, `{"schema":"urn:ax:schema:error"}`), "integrity_failure"},
		{"wrong error version", fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, replaceOnce(t, fixtureErrorObject("incompatible_protocol", 6, false), `"schema_version":"1.2.0"`, `"schema_version":"1.1.0"`, 1)), "integrity_failure"},
		{"unknown member", []byte(replaceOnce(t, downgrade, `"ok":false`, `"ok":false,"extra":1`, 1)), "adapter_protocol_violation"},
		{"missing member", []byte(replaceOnce(t, downgrade, `,"ok":false`, ``, 1)), "adapter_protocol_violation"},
		{"wrong operation echo", fixtureFailureFrame("2.0.0", "probe", fixtureRequestID, fixtureErrorObject("incompatible_protocol", 6, false)), "adapter_protocol_violation"},
		{"wrong schema", []byte(replaceOnce(t, downgrade, "session-directory-node-response", "session-directory-node-request", 1)), "adapter_protocol_violation"},
		{"relabeled schema version", []byte(replaceOnce(t, downgrade, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)), "adapter_protocol_violation"},
		{"wrong protocol", []byte(replaceOnce(t, downgrade, `"protocol":"urn:ax:protocol:session-directory-node"`, `"protocol":"urn:ax:protocol:provider"`, 1)), "adapter_protocol_violation"},
		{"wrong request echo", fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID[:len(fixtureRequestID)-1]+"0", fixtureErrorObject("incompatible_protocol", 6, false)), "adapter_protocol_violation"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckFailureEnvelope(probe.frame, request); requireCode(t, err, probe.code) == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestResponseIdentitySharedArms drives the two production
// response entries through the shared identity arms both
// branches carry: an empty frame, an oversize frame, and a
// truncated frame refuse identically on the success and failure
// paths, proving the context-prefixed arms fire in both.
func TestResponseIdentitySharedArms(t *testing.T) {
	t.Parallel()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	oversize := append(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, `{}`), make([]byte, frameBoundBytes)...)
	for _, probe := range []struct {
		name  string
		frame []byte
	}{
		{"empty frame", []byte{}},
		{"oversize frame", oversize},
		{"truncated frame", []byte(`{"schema":`)},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckSuccessEnvelope(probe.frame, request); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("success must refuse")
			}
			if _, err := CheckFailureEnvelope(probe.frame, request); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("failure must refuse")
			}
		})
	}
}

// TestRefuseUnknownOperation drives the production dispatch entry:
// a name outside the closed registry refuses with
// operation_unknown naming the operation.
func TestRefuseUnknownOperation(t *testing.T) {
	t.Parallel()
	err := RefuseUnknownOperation("query")
	failure := requireCode(t, err, "operation_unknown")
	if operation, _ := failure.Detail("operation"); operation != "query" {
		t.Fatalf("refusal detail operation = %v, want query", operation)
	}
	if err := RefuseUnknownOperation(""); err == nil {
		t.Fatal("RefuseUnknownOperation(empty) must refuse")
	}
}
