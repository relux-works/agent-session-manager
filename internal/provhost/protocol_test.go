package provhost

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// asError unwraps err to an *axerror.Error without hiding the chain.
func asError(t *testing.T, err error, failure **axerror.Error) bool {
	t.Helper()
	return errors.As(err, failure)
}

const (
	testRequestID = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	testOtherID   = "0198f4c8-e4b0-75cc-9576-1234567890ab"
	testDeadline  = "2026-08-19T04:05:00.000Z"
	testNow       = "2026-08-19T04:00:00.000Z"
)

func mustUUIDv7(t *testing.T, value string) scalar.UUIDv7 {
	t.Helper()
	parsed, err := scalar.ParseUUIDv7(value)
	if err != nil {
		t.Fatalf("ParseUUIDv7(%q): %v", value, err)
	}
	return parsed
}

func mustTimestamp(t *testing.T, value string) scalar.Timestamp {
	t.Helper()
	parsed, err := scalar.ParseTimestamp(value)
	if err != nil {
		t.Fatalf("ParseTimestamp(%q): %v", value, err)
	}
	return parsed
}

func mustInstant(t *testing.T, value string) time.Time {
	t.Helper()
	instant, err := mustTimestamp(t, value).Time()
	if err != nil {
		t.Fatalf("Timestamp(%q).Time(): %v", value, err)
	}
	return instant
}

// testRequest builds the Section 7.2 request-envelope fixture: a doctor
// operation with a future deadline. The production entry point for
// framing is EncodeRequest.
func testRequest(t *testing.T) Request {
	t.Helper()
	return Request{
		Operation: OpDoctor,
		RequestID: mustUUIDv7(t, testRequestID),
		Deadline:  mustTimestamp(t, testDeadline),
		Body:      json.RawMessage(`{"platform":"macos","architecture":"arm64","provider_executable":null,"identity":null}`),
	}
}

// failureCode extracts the Structured Error code from err, failing when
// err is not an *axerror.Error. Every failure below is a 1.0.0 object,
// local or child-bound.
func failureCode(t *testing.T, err error) axerror.Code {
	t.Helper()
	if err == nil {
		t.Fatal("want a failure, got nil")
	}
	var failure *axerror.Error
	if !asError(t, err, &failure) {
		t.Fatalf("error %v is not a Structured Error", err)
	}
	return failure.Code()
}

func failureExit(t *testing.T, err error) int {
	t.Helper()
	var failure *axerror.Error
	if !asError(t, err, &failure) {
		t.Fatalf("error %v is not a Structured Error", err)
	}
	return failure.ExitCode()
}

// failureObject unwraps err to the Structured Error itself, so arm
// identity beyond the stable code can be asserted.
func failureObject(t *testing.T, err error) *axerror.Error {
	t.Helper()
	if err == nil {
		t.Fatal("want a failure, got nil")
	}
	var failure *axerror.Error
	if !asError(t, err, &failure) {
		t.Fatalf("error %v is not a Structured Error", err)
	}
	return failure
}

// failureMember returns the "member" diagnostic of a frame refusal. Every
// provider_protocol_error arm names the member it refused on, so asserting
// it pins the arm: a missing-member gate deleted from checkResponseMembers
// slides to a lower arm carrying the same code with a different member or
// detail, and the slide reddens here instead of passing silently.
func failureMember(t *testing.T, err error) string {
	t.Helper()
	failure := failureObject(t, err)
	member, ok := failure.Detail("member")
	if !ok {
		t.Fatalf("refusal %v carries no member detail", err)
	}
	text, ok := member.(string)
	if !ok {
		t.Fatalf("refusal %v member detail is %T, want string", err, member)
	}
	return text
}

// requireFrameRefusal asserts the full arm identity of a plugin-frame
// refusal: the stable code, the refused member, the detail naming the
// rule, and the non-retryable bit doc.go promises for every local
// failure this package emits.
func requireFrameRefusal(t *testing.T, err error, member, detail string) {
	t.Helper()
	if failureCode(t, err) != "provider_protocol_error" {
		t.Fatalf("DecodeResponse code = %v, want provider_protocol_error", err)
	}
	if got := failureMember(t, err); got != member {
		t.Fatalf("DecodeResponse member = %q, want %q (detail: %v)", got, member, err)
	}
	if !strings.Contains(err.Error(), detail) {
		t.Fatalf("DecodeResponse error = %v, want detail containing %q", err, detail)
	}
	if failureObject(t, err).Retryable() {
		t.Fatalf("DecodeResponse error = %v, want non-retryable", err)
	}
}

// requireLocalRefusal asserts the arm identity of a refusal that carries
// no member detail (invalid_config framing refusals and integrity_failure
// status refusals): the stable code, the detail naming the rule in the
// human text, and the non-retryable bit.
func requireLocalRefusal(t *testing.T, err error, code axerror.Code, detail string) {
	t.Helper()
	if failureCode(t, err) != code {
		t.Fatalf("code = %v, want %s", err, code)
	}
	if !strings.Contains(err.Error(), detail) {
		t.Fatalf("error = %v, want detail containing %q", err, detail)
	}
	if failureObject(t, err).Retryable() {
		t.Fatalf("error = %v, want non-retryable", err)
	}
}

// TestEncodeRequestEmitsExactFrame pins the request wire bytes: the six
// exact members in fixed order, with the body passed through verbatim.
func TestEncodeRequestEmitsExactFrame(t *testing.T) {
	frame, err := EncodeRequest(testRequest(t), mustInstant(t, testNow))
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	want := `{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"0198f4c8-8e50-7f66-8f70-1234567890ab","operation":"doctor","deadline":"2026-08-19T04:05:00.000Z","body":{"platform":"macos","architecture":"arm64","provider_executable":null,"identity":null}}`
	if string(frame) != want {
		t.Fatalf("EncodeRequest frame = %s, want %s", frame, want)
	}
}

// TestEncodeRequestDispatchesEveryRegistryOperation proves dispatch
// reaches every registry entry: each of the 15 operations frames with
// its own name. The registry list itself is pinned against the
// specification text by the inventory test.
func TestEncodeRequestDispatchesEveryRegistryOperation(t *testing.T) {
	now := mustInstant(t, testNow)
	for _, name := range Operations() {
		req := testRequest(t)
		req.Operation = Operation(name)
		frame, err := EncodeRequest(req, now)
		if err != nil {
			t.Fatalf("EncodeRequest(%q): %v", name, err)
		}
		var decoded struct {
			Operation string `json:"operation"`
		}
		if err := json.Unmarshal(frame, &decoded); err != nil {
			t.Fatalf("EncodeRequest(%q) frame is not JSON: %v", name, err)
		}
		if decoded.Operation != name {
			t.Fatalf("EncodeRequest(%q) frame carries operation %q", name, decoded.Operation)
		}
	}
}

// TestEncodeRequestRefusals proves framing fails closed before any
// process starts: unknown operations, non-UUIDv7 request IDs, stale or
// malformed deadlines, non-object bodies, and oversize frames are all
// invalid_config from EncodeRequest. The production entry point is
// EncodeRequest.
func TestEncodeRequestRefusals(t *testing.T) {
	now := mustInstant(t, testNow)
	for _, kase := range []struct {
		name   string
		mutate func(*Request)
		detail string
	}{
		{"unknown operation", func(req *Request) { req.Operation = "reboot" }, "unknown operation"},
		{"empty operation", func(req *Request) { req.Operation = "" }, "unknown operation"},
		{"zero request id", func(req *Request) { req.RequestID = scalar.UUIDv7{} }, "request_id is not a UUIDv7"},
		{"zero deadline", func(req *Request) { req.Deadline = scalar.Timestamp{} }, "deadline is not a timestamp"},
		{"past deadline", func(req *Request) { req.Deadline = mustTimestamp(t, testNow) }, "deadline is not in the future"},
		{"deadline long past", func(req *Request) { req.Deadline = mustTimestamp(t, "2020-01-01T00:00:00.000Z") }, "deadline is not in the future"},
		{"array body", func(req *Request) { req.Body = json.RawMessage(`[1,2]`) }, "body is not a JSON object"},
		{"string body", func(req *Request) { req.Body = json.RawMessage(`"x"`) }, "body is not a JSON object"},
		{"scalar body", func(req *Request) { req.Body = json.RawMessage(`3`) }, "body is not a JSON object"},
		{"empty body", func(req *Request) { req.Body = nil }, "body is not a JSON object"},
		{"blank body", func(req *Request) { req.Body = json.RawMessage(`  `) }, "body is not a JSON object"},
		{"malformed body", func(req *Request) { req.Body = json.RawMessage(`{oops`) }, "body is not a JSON object"},
		{"oversize frame", func(req *Request) {
			req.Body = json.RawMessage(`{"pad":"` + strings.Repeat("a", specFrameLimitBytes) + `"}`)
		}, "request frame exceeds 8 MiB"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			req := testRequest(t)
			kase.mutate(&req)
			_, err := EncodeRequest(req, now)
			requireLocalRefusal(t, err, "invalid_config", kase.detail)
		})
	}
}

// successFrame builds a minimal valid success envelope for want.
func successFrame(t *testing.T, want, body string) []byte {
	t.Helper()
	return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + want + `","ok":true,"body":` + body + `}`)
}

// childFailureFrame builds a failure envelope carrying a genuine bound
// child error, constructed through the production axerror entry point.
func childFailureFrame(t *testing.T, want string) []byte {
	t.Helper()
	child, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    "capability_unavailable",
		Message: "portable store is not available for this provider build",
		Details: axerror.Details{},
	})
	if err != nil {
		t.Fatalf("axerror.New: %v", err)
	}
	raw, err := json.Marshal(child)
	if err != nil {
		t.Fatalf("Marshal child: %v", err)
	}
	return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + want + `","ok":false,"error":` + string(raw) + `}`)
}

// TestDecodeResponseAcceptsSuccess proves the positive path returns the
// body verbatim for correlation-matched frames.
func TestDecodeResponseAcceptsSuccess(t *testing.T) {
	body := `{"provider_id":"pi","provider_version":"0.73.1","findings":[]}`
	got, err := DecodeResponse(successFrame(t, testRequestID, body), mustUUIDv7(t, testRequestID))
	if err != nil {
		t.Fatalf("DecodeResponse: %v", err)
	}
	if string(got.Body) != body {
		t.Fatalf("DecodeResponse body = %s, want %s", got.Body, body)
	}
}

// TestDecodeResponseReturnsChildFailure proves a failure envelope
// surfaces the bound child error itself, with its own code and exit.
func TestDecodeResponseReturnsChildFailure(t *testing.T) {
	_, err := DecodeResponse(childFailureFrame(t, testRequestID), mustUUIDv7(t, testRequestID))
	if failureCode(t, err) != "capability_unavailable" {
		t.Fatalf("DecodeResponse code = %v, want the bound child code capability_unavailable", err)
	}
	if failureExit(t, err) != 6 {
		t.Fatalf("DecodeResponse exit = %d, want 6", failureExit(t, err))
	}
}

// TestDecodeResponseRecognizableMajorMismatch proves a foreign major is
// reported as incompatible_protocol without trusting the payload: the
// frame carries a forged error object whose code must never surface.
func TestDecodeResponseRecognizableMajorMismatch(t *testing.T) {
	for _, version := range []string{"3.0.0", "1.0.0", "10.2.3"} {
		frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"` + version + `","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"9.9.9","code":"forged_code","message":"forged","exit_code":0,"retryable":true,"details":{}}}`)
		_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
		if failureCode(t, err) != "incompatible_protocol" {
			t.Fatalf("DecodeResponse(%s) code = %v, want incompatible_protocol", version, err)
		}
		observed, ok := failureObject(t, err).Detail("observed")
		if !ok || observed != version {
			t.Fatalf("DecodeResponse(%s) observed = %v, want the foreign version %q", version, observed, version)
		}
		if failureExit(t, err) != 6 {
			t.Fatalf("DecodeResponse(%s) exit = %d, want 6", version, failureExit(t, err))
		}
		if strings.Contains(err.Error(), "forged_code") {
			t.Fatalf("foreign payload leaked into the local failure: %v", err)
		}
	}
}

// TestDecodeResponseForeignMajorPrecedesMemberRules proves the version
// gate runs before the v2 member vocabulary: a recognizable foreign
// major is incompatible_protocol even when it carries members v2
// never allows, or omits members v2 requires. Judging those shapes
// through the v2 member rules first would misreport them as
// provider_protocol_error unknown member or missing member, which is
// what the code did before the gate moved ahead. The production entry
// point is DecodeResponse.
func TestDecodeResponseForeignMajorPrecedesMemberRules(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	envelope := func(version, members string) []byte {
		return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"` + version + `","request_id":"` + testRequestID + `",` + members + `}`)
	}
	for _, kase := range []struct {
		name    string
		frame   []byte
		version string
	}{
		{"v3 member on success", envelope("3.0.0", `"ok":true,"body":{},"capabilities":[]`), "3.0.0"},
		{"v3 member on failure", envelope("3.0.0", `"ok":false,"error":{"code":"x"},"descriptor":{}`), "3.0.0"},
		{"missing body on success", envelope("3.0.0", `"ok":true`), "3.0.0"},
		{"missing error on failure", envelope("3.0.0", `"ok":false`), "3.0.0"},
		{"missing ok", envelope("3.0.0", `"body":{}`), "3.0.0"},
		{"major one with v3 member", envelope("1.0.0", `"ok":true,"body":{},"capabilities":[]`), "1.0.0"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			_, err := DecodeResponse(kase.frame, want)
			if failureCode(t, err) != "incompatible_protocol" {
				t.Fatalf("DecodeResponse(%s) code = %v, want incompatible_protocol", kase.name, err)
			}
			if observed, ok := failureObject(t, err).Detail("observed"); !ok || observed != kase.version {
				t.Fatalf("DecodeResponse(%s) observed = %v, want the foreign version %q", kase.name, observed, kase.version)
			}
			if failureExit(t, err) != 6 {
				t.Fatalf("DecodeResponse(%s) exit = %d, want 6", kase.name, failureExit(t, err))
			}
		})
	}
	// The gate needs the envelope identity: without our protocol id,
	// or without a readable version, the member rules keep judging.
	// (Member-shape faults still precede the protocol identity
	// check, as before: a foreign protocol id carrying a v2-unknown
	// member is an unknown member, not a foreign envelope.)
	_, err := DecodeResponse([]byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"3.0.0","request_id":"`+testRequestID+`","ok":true,"body":{}}`), want)
	requireFrameRefusal(t, err, "protocol", "not a provider envelope")
	_, err = DecodeResponse([]byte(`{"protocol_version":"3.0.0","request_id":"`+testRequestID+`","ok":true,"body":{}}`), want)
	requireFrameRefusal(t, err, "protocol", "missing member")
	_, err = DecodeResponse([]byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":3,"request_id":"`+testRequestID+`","ok":true,"body":{}}`), want)
	requireFrameRefusal(t, err, "protocol_version", "member is not a string")
}

// TestDecodeResponseRefusals proves every unusable frame shape fails with
// provider_protocol_error, never with a partial result. The production
// entry point is DecodeResponse.
func TestDecodeResponseRefusals(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	okFrame := func(members string) []byte {
		return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `",` + members + `}`)
	}
	validBody := `{"provider_id":"pi"}`
	// Each row names the refusal arm it must reach: the member detail
	// and the rule detail. A deleted required-member gate slides the
	// missing member to a lower arm with the same code but a different
	// member or detail, and the slide reddens here. The malformed
	// versions below carry a foreign major ("3" or empty) so a deleted
	// parseMajor arm promotes them to incompatible_protocol instead of
	// collapsing back onto this code. The non-numeric-major rows carry a
	// numeric rest (".0.0") so only the major-digit branch can refuse
	// them: with a non-numeric rest the rest-digit branch would fire
	// first and the misclassification would collapse back to this code.
	// The full five-branch parseMajor enumeration lives in the derived
	// refusal-arm inventory; these rows are its entry-point witnesses.
	versionFrame := func(version string) []byte {
		return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"` + version + `","request_id":"` + testRequestID + `","ok":true,"body":{}}`)
	}
	for _, kase := range []struct {
		name   string
		frame  []byte
		member string
		detail string
	}{
		{"empty", []byte{}, "", "not a JSON object"},
		{"not JSON", []byte(`not json`), "", "not a JSON object"},
		{"truncated", []byte(`{"protocol":`), "protocol", "not a JSON object"},
		{"array", []byte(`[1,2]`), "", "not a JSON object"},
		{"scalar", []byte(`true`), "", "not a JSON object"},
		{"trailing data", append(successFrame(t, testRequestID, validBody), 'x'), "", "trailing data after the object"},
		{"trailing valid value", append(successFrame(t, testRequestID, validBody), ' ', '{', '}'), "", "trailing data after the object"},
		{"duplicate member", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "request_id", "duplicate member"},
		{"missing protocol", []byte(`{"protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol", "missing member"},
		{"missing version", []byte(`{"protocol":"urn:ax:protocol:provider","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "missing member"},
		{"missing request id", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","ok":true,"body":{}}`), "request_id", "missing member"},
		{"missing ok", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `"}`), "ok", "missing member"},
		{"missing body on success", okFrame(`"ok":true`), "body", "missing member"},
		{"missing error on failure", okFrame(`"ok":false`), "error", "missing member"},
		{"body on failure", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}},"body":{}}`), "body", "failure envelope carries body"},
		{"error on success", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{},"error":{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}}}`), "error", "success envelope carries error"},
		{"unknown member", okFrame(`"ok":true,"body":{},"diagnostics":[]`), "diagnostics", "unknown member"},
		{"ok not boolean", okFrame(`"ok":"yes","body":{}`), "ok", "member is not a boolean"},
		{"wrong protocol", []byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol", "not a provider envelope"},
		{"protocol not string", []byte(`{"protocol":7,"protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol", "not a provider envelope"},
		{"minor version", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.1.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "unsupported protocol version"},
		{"garbage version", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"v2","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "unsupported protocol version"},
		{"short version", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "unsupported protocol version"},
		{"empty version part", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2..0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "unsupported protocol version"},
		{"non-numeric version", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"a.b.c","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "unsupported protocol version"},
		{"foreign two-part version", versionFrame("3.0"), "protocol_version", "unsupported protocol version"},
		{"foreign four-part version", versionFrame("3.0.0.0"), "protocol_version", "unsupported protocol version"},
		{"foreign empty major", versionFrame(".0.0"), "protocol_version", "unsupported protocol version"},
		{"foreign empty minor", versionFrame("3..0"), "protocol_version", "unsupported protocol version"},
		{"foreign non-numeric rest", versionFrame("3.b.c"), "protocol_version", "unsupported protocol version"},
		{"foreign non-numeric major", versionFrame("a.0.0"), "protocol_version", "unsupported protocol version"},
		{"foreign alphanumeric major", versionFrame("2a.0.0"), "protocol_version", "unsupported protocol version"},
		{"foreign negative major", versionFrame("-1.0.0"), "protocol_version", "unsupported protocol version"},
		{"foreign plus major", versionFrame("+3.0.0"), "protocol_version", "unsupported protocol version"},
		{"version not string", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":2,"request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "member is not a string"},
		{"empty version", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"","request_id":"` + testRequestID + `","ok":true,"body":{}}`), "protocol_version", "unsupported protocol version"},
		{"request id not string", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":7,"ok":true,"body":{}}`), "request_id", "member is not a string"},
		{"request id not UUIDv7", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"not-a-uuid","ok":true,"body":{}}`), "request_id", "request_id is not a UUIDv7"},
		{"request id mismatch", successFrame(t, testOtherID, validBody), "request_id", "request_id does not match the request"},
		{"body array on success", okFrame(`"ok":true,"body":[]`), "body", "body is not a JSON object"},
		{"body scalar on success", okFrame(`"ok":true,"body":1`), "body", "body is not a JSON object"},
		{"error scalar on failure", okFrame(`"ok":false,"error":1`), "error", "error is not a JSON object"},
		{"error unbound version", []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"9.9.9","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}}}`), "error", "error is not a bound Structured Error 1.0.0"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			_, err := DecodeResponse(kase.frame, want)
			requireFrameRefusal(t, err, kase.member, kase.detail)
		})
	}
}

// TestDecodeResponseRefusesOversizeFrame proves the 8 MiB bound from the
// decoding side: a frame one byte over the limit fails without being
// parsed, and a frame exactly at the limit is accepted.
// specFrameLimitBytes is the absolute Section 7.2 bound, stated as a
// literal rather than derived from MaxFrameBytes: the fixtures below pin
// the constant to the specification text, so weakening the constant
// reddens here instead of scaling the fixtures along with it.
const specFrameLimitBytes = 8 << 20

func TestDecodeResponseRefusesOversizeFrame(t *testing.T) {
	if MaxFrameBytes != specFrameLimitBytes {
		t.Fatalf("MaxFrameBytes = %d, want the specified %d", MaxFrameBytes, specFrameLimitBytes)
	}
	want := mustUUIDv7(t, testRequestID)
	build := func(pad int) []byte {
		return successFrame(t, testRequestID, `{"pad":"`+strings.Repeat("a", pad)+`"}`)
	}
	// The pad contributes exactly pad bytes, so base+pad is the frame
	// size. One byte over the limit must fail; exactly at it must pass.
	base := len(build(0))
	over := build(specFrameLimitBytes - base + 1)
	if len(over) != specFrameLimitBytes+1 {
		t.Fatalf("oversize fixture is %d bytes, want exactly %d", len(over), specFrameLimitBytes+1)
	}
	_, err := DecodeResponse(over, want)
	requireFrameRefusal(t, err, "", "frame exceeds 8 MiB")
	frame := build(specFrameLimitBytes - base)
	if len(frame) != specFrameLimitBytes {
		t.Fatalf("boundary fixture is %d bytes, want exactly %d", len(frame), specFrameLimitBytes)
	}
	if _, err := DecodeResponse(frame, want); err != nil {
		t.Fatalf("boundary DecodeResponse: %v", err)
	}
}

// TestDecodeResponseRefusesNonUTF8 proves the UTF-8 gate: invalid bytes
// fail even when the surrounding shape would parse.
func TestDecodeResponseRefusesNonUTF8(t *testing.T) {
	frame := append([]byte{}, successFrame(t, testRequestID, `{"provider_id":"pi"}`)...)
	frame[len(frame)-3] = 0xff
	_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
	requireFrameRefusal(t, err, "", "frame is not UTF-8")
}

// TestEncodeRequestRefusesOversizeFrame proves the 8 MiB bound from the
// framing side with the same exact-boundary technique as the decoding
// side: a request one byte over the limit fails, and a request exactly at
// the limit frames. The pad contributes exactly pad bytes, so a halved or
// doubled bound reddens here instead of scaling along.
func TestEncodeRequestRefusesOversizeFrame(t *testing.T) {
	if MaxFrameBytes != specFrameLimitBytes {
		t.Fatalf("MaxFrameBytes = %d, want the specified %d", MaxFrameBytes, specFrameLimitBytes)
	}
	now := mustInstant(t, testNow)
	build := func(pad int) (Request, int) {
		req := testRequest(t)
		req.Body = json.RawMessage(`{"pad":"` + strings.Repeat("a", pad) + `"}`)
		frame, err := EncodeRequest(req, now)
		if err != nil {
			return req, -1
		}
		return req, len(frame)
	}
	_, base := build(0)
	overPad := specFrameLimitBytes - base + 1
	overReq, overSize := build(overPad)
	if overSize != -1 {
		t.Fatalf("oversize request framed at %d bytes, want refusal over %d", overSize, specFrameLimitBytes)
	}
	if _, err := EncodeRequest(overReq, now); err == nil {
		t.Fatal("oversize EncodeRequest succeeded, want invalid_config")
	} else {
		requireLocalRefusal(t, err, "invalid_config", "request frame exceeds 8 MiB")
	}
	boundaryReq, boundarySize := build(specFrameLimitBytes - base)
	if boundarySize != specFrameLimitBytes {
		t.Fatalf("boundary request is %d bytes, want exactly %d", boundarySize, specFrameLimitBytes)
	}
	if _, err := EncodeRequest(boundaryReq, now); err != nil {
		t.Fatalf("boundary EncodeRequest: %v", err)
	}
}
