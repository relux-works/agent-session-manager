package provhost

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
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

// TestDecodeResponseRefusesDuplicateOfEveryMember requires the
// response envelope to refuse a duplicated member for every
// member of its closed set, plus the foreign keys the frame-gate
// sweep named. The first battery witnessed the duplicate arm at
// request_id alone; a narrowing mutant admitting exactly a
// duplicated body stayed green and took the second body
// last-wins through this production entry. The member set
// derives from responseMembers at runtime, so a seventh member
// is automatically swept; the literal table below is the only
// hand-written part, and a member without a literal fails closed
// instead of passing silently.
func TestDecodeResponseRefusesDuplicateOfEveryMember(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	literals := map[string]string{
		"protocol":         `"urn:ax:protocol:provider"`,
		"protocol_version": `"2.0.0"`,
		"request_id":       `"` + testRequestID + `"`,
		"ok":               `true`,
		"body":             `{"a":1}`,
		"error":            `{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}}`,
	}
	var members []string
	for name := range responseMembers {
		members = append(members, name)
	}
	sort.Strings(members)
	if len(members) != len(literals) {
		t.Fatalf("response member set has %d names for %d literals; the test cannot sweep what it cannot build", len(members), len(literals))
	}
	members = append(members, "v", "capabilities", "provider_id", "cursor")
	for _, name := range members {
		t.Run(name, func(t *testing.T) {
			literal, ok := literals[name]
			if !ok {
				literal = `1`
			}
			frame := successFrame(t, testRequestID, `{"a":1}`)
			// The success frame carries every member but
			// error: a present member needs one more copy to
			// double, an absent member needs two.
			copies := 1
			if !strings.Contains(string(frame), `"`+name+`":`) {
				copies = 2
			}
			for index := 0; index < copies; index++ {
				frame = frame[:len(frame)-1]
				frame = append(frame, []byte(`,"`+name+`":`+literal+`}`)...)
			}
			_, err := DecodeResponse(frame, want)
			requireFrameRefusal(t, err, name, "duplicate member")
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

// TestDecodeResponseWellFormedV3SuccessIsMismatch proves the documented
// v2-only outcome for a syntactically valid 3.0.0 success envelope: it
// is incompatible_protocol, not an admission. A v2/v3 major mismatch
// follows the Section 15.1 close/termination rule; the caller never
// trusts a different major's payload (Section 7.A). The production
// entry point is DecodeResponse.
func TestDecodeResponseWellFormedV3SuccessIsMismatch(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"3.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`)
	_, err := DecodeResponse(frame, want)
	if failureCode(t, err) != "incompatible_protocol" {
		t.Fatalf("DecodeResponse(3.0.0 success) code = %v, want incompatible_protocol", err)
	}
	if observed, ok := failureObject(t, err).Detail("observed"); !ok || observed != "3.0.0" {
		t.Fatalf("DecodeResponse(3.0.0 success) observed = %v, want 3.0.0", observed)
	}
	if failureObject(t, err).Version() != axerror.Version100 {
		t.Fatalf("DecodeResponse(3.0.0 success) version = %v, want the local Error 1.0.0", failureObject(t, err).Version())
	}
}

// TestDecodeResponseV3FailureWithValid130ErrorIsMismatch proves the v3
// binding is named but never trusted: a 3.0.0 failure envelope carrying
// a well-formed Structured Error 1.3.0 object (the exact binding
// Section 7.A assigns v3) is still incompatible_protocol, and no member
// of the foreign error — code, message, retryable bit, or details —
// reaches the local failure. The embedded error is first decoded through
// the production (provider, 3) → 1.3.0 binding, so the test proves
// refusal of a genuinely well-formed foreign error rather than refusal
// of malformed bytes that any arm would reject. The full rendering is
// asserted, never a shared code: incompatible_protocol is the documented
// mismatch arm, and the assertion pins that the v3 outcome is refusal,
// not admission.
func TestDecodeResponseV3FailureWithValid130ErrorIsMismatch(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	foreign := []byte(`{"schema":"urn:ax:schema:error","schema_version":"1.3.0","code":"terminal_backend_integrity_failure",` +
		`"message":"v3 backend binding failed","exit_code":9,"retryable":false,"details":{}}`)
	if _, err := axerror.DecodeBound(axerror.ContainingContract{ID: ProtocolID, Major: 3}, foreign); err != nil {
		t.Fatalf("embedded v3 error is not well-formed 1.3.0 under the (provider, 3) binding: %v", err)
	}
	frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"3.0.0","request_id":"` + testRequestID + `","ok":false,` +
		`"error":` + string(foreign) + `}`)
	_, err := DecodeResponse(frame, want)
	if failureCode(t, err) != "incompatible_protocol" {
		t.Fatalf("DecodeResponse(3.0.0 failure) code = %v, want incompatible_protocol", err)
	}
	rendered := err.Error()
	for _, foreign := range []string{"terminal_backend_integrity_failure", "v3 backend binding failed"} {
		if strings.Contains(rendered, foreign) {
			t.Fatalf("foreign v3 payload leaked into the local failure: %v", err)
		}
	}
	if observed, ok := failureObject(t, err).Detail("observed"); !ok || observed != "3.0.0" {
		t.Fatalf("DecodeResponse(3.0.0 failure) observed = %v, want 3.0.0", observed)
	}
}

// TestDecodeResponseV2FailureWith130ErrorIsBoundTo100 proves the
// failure-error contract is selected by the observed envelope major,
// never the document: a well-formed 1.3.0 error under a 2.0.0 envelope
// is refused as unbound under 1.0.0 rather than decoded as 1.3.0. A
// gate that decoded whatever version the document claimed would admit
// this frame; the named arm below is what keeps the 1.0.0 binding
// exact.
func TestDecodeResponseV2FailureWith130ErrorIsBoundTo100(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":false,` +
		`"error":{"schema":"urn:ax:schema:error","schema_version":"1.3.0","code":"invalid_config",` +
		`"message":"wrong version for this envelope","exit_code":3,"retryable":false,"details":{}}}`)
	_, err := DecodeResponse(frame, want)
	requireFrameRefusal(t, err, "error", "error is not a bound Structured Error 1.0.0")
	if strings.Contains(err.Error(), "wrong version for this envelope") {
		t.Fatalf("foreign-version payload leaked into the local failure: %v", err)
	}
}

// TestParseMajorNeverWraps proves the major is decided on the digit
// string, never on a wrapped accumulator: 18446744073709551617.0.0
// wraps to major 1 and 18446744073709551618.0.0 wraps to major 2 in 64
// bits, so an accumulating gate mistakes both foreign majors for native
// ones. A wrapped major is a version-confusion primitive: the envelope
// gate below decides admission on this number. All four literals are
// recognized (all-numeric X.Y.Z) and must report a major other than 1
// or 2; the wrap-to-2 row additionally must refuse at DecodeResponse as
// a foreign major (incompatible_protocol, exit 6), not as an unusable
// frame (exit 13) and never as 2.0.0.
func TestParseMajorNeverWraps(t *testing.T) {
	for _, version := range []string{
		"18446744073709551617.0.0",
		"18446744073709551618.0.0",
		"18446744073709551616.0.0",
		"99999999999999999999999999999999999999999.0.0",
	} {
		major, recognized := parseMajor(version)
		if !recognized {
			t.Fatalf("parseMajor(%q) = unrecognized, want a recognized foreign major", version)
		}
		if major == 1 || major == 2 {
			t.Fatalf("parseMajor(%q) = %d, want a major other than 1 or 2", version, major)
		}
	}

	want := mustUUIDv7(t, testRequestID)
	_, err := DecodeResponse(armVersionFrame("18446744073709551618.0.0"), want)
	if failureCode(t, err) != "incompatible_protocol" {
		t.Fatalf("DecodeResponse(wrapped-to-2 major) code = %v, want incompatible_protocol", err)
	}
	if failureExit(t, err) != 6 {
		t.Fatalf("DecodeResponse(wrapped-to-2 major) exit = %d, want 6", failureExit(t, err))
	}
}

// TestParseMajorSaturationStillValidatesTheRest proves saturation never
// short-circuits the X.Y.Z shape check: a version whose first component
// overflows but whose rest is malformed is unrecognized, so
// DecodeResponse refuses it as an unusable frame
// (provider_protocol_error, exit 13), not as a foreign major
// (incompatible_protocol, exit 6). The saturating early return this
// test guards against reported exactly these inputs as recognized; the
// rows below pin the two classes apart so a future short-circuit
// reddens here. The production entry point is DecodeResponse.
func TestParseMajorSaturationStillValidatesTheRest(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	for _, version := range []string{
		"99999999999999999999.abc.def",
		"99999999999999999999.0.x",
		"99999999999999999999..0",
		"18446744073709551618..0",
		"18446744073709551618.0.x",
	} {
		if major, recognized := parseMajor(version); recognized {
			t.Fatalf("parseMajor(%q) = (%d, true), want unrecognized", version, major)
		}
		_, err := DecodeResponse(armVersionFrame(version), want)
		if failureCode(t, err) != "provider_protocol_error" {
			t.Fatalf("DecodeResponse(%q) code = %v, want provider_protocol_error", version, err)
		}
		if failureExit(t, err) != 13 {
			t.Fatalf("DecodeResponse(%q) exit = %d, want 13", version, failureExit(t, err))
		}
	}
}

// TestParseMajorSaturationEdge proves the saturation guard at its exact
// arithmetic edge, through the production entry point DecodeResponse.
// math.MaxInt is 9223372036854775807 and MaxInt/10 is 922337203685477580.
// The guard `major > (math.MaxInt-step)/10` must admit 9223372036854775807
// exactly (the largest value that fits) and saturate 9223372036854775808
// (MaxInt+1) to MaxInt. The weakened guard `major > math.MaxInt/10` differs
// only when the accumulator sits at exactly MaxInt/10 and the next digit is
// 8 or 9: it computes major*10+step in int64, which wraps. The aliasing
// witness 922337203685477580802.0.0 walks the accumulator to MaxInt/10, wraps
// to -2^63 on the `8`, resets to 0 on the following `0` (0 mod 2^64), and lands
// on 2 — a foreign major aliasing native major 2, the exact invariant the
// parseMajor doc comment asserts. Its one-step neighbours 801 (wraps to 1)
// and 803 (wraps to 3) pin that the fix is at the guard, not a literal
// blocklist of one string. Moving the guard literal by one term must redden
// here: every row requires parseMajor == MaxInt (not merely recognized) and
// DecodeResponse incompatible_protocol exit 6 with observed echoed.
func TestParseMajorSaturationEdge(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	for _, version := range []string{
		"9223372036854775807.0.0",
		"9223372036854775808.0.0",
		"922337203685477580801.0.0",
		"922337203685477580802.0.0",
		"922337203685477580803.0.0",
	} {
		major, recognized := parseMajor(version)
		if !recognized {
			t.Fatalf("parseMajor(%q) = unrecognized, want saturated foreign major", version)
		}
		if major != math.MaxInt {
			t.Fatalf("parseMajor(%q) = %d, want saturation to math.MaxInt", version, major)
		}
		if major == ProtocolMajor {
			t.Fatalf("parseMajor(%q) aliased to native major %d", version, major)
		}
		_, err := DecodeResponse(armVersionFrame(version), want)
		if failureCode(t, err) != "incompatible_protocol" {
			t.Fatalf("DecodeResponse(%q) code = %v, want incompatible_protocol", version, err)
		}
		if failureExit(t, err) != 6 {
			t.Fatalf("DecodeResponse(%q) exit = %d, want 6", version, failureExit(t, err))
		}
		if observed, ok := failureObject(t, err).Detail("observed"); !ok || observed != version {
			t.Fatalf("DecodeResponse(%q) observed = %v, want the foreign version", version, observed)
		}
	}
}

// TestParseMajorDigitBoundariesRefuseAtEntry proves the digit-range arms at
// their exact edges, through the production entry point DecodeResponse.
// The major arm computes `digit < '0' || digit > '9'` and the rest arm
// computes `rest[i] < '0' || rest[i] > '9'`; the two characters adjacent to
// that range are '/' (0x2F, one below '0') and ':' (0x3A, one above '9').
// A narrowing that admits exactly one of them — by editing the bound to
// `rest[i] < '/'` or `digit > ':'`, or by inserting `if digit == ':' {
// continue }` / `if rest[i] == '/' { continue }` above the unchanged
// condition (condition text preserved, so the source-text census stays
// byte-identical) — migrates these rows from provider_protocol_error exit 13
// to incompatible_protocol exit 6, and must redden here rather than only in
// the inventory bijection. Rows cover '/' and ':' in the major and in each
// rest position (minor and patch), plus the leading single-character majors,
// so the fix is verified one step away from any single finding vector. The
// correlating controls ('a.0.0' major-letter, '3.b.c' rest-letter) stay
// refused under every such mutant and are covered by the existing corpus.
func TestParseMajorDigitBoundariesRefuseAtEntry(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	for _, version := range []string{
		"3/.0.0",
		"/.0.0",
		"3:.0.0",
		":.0.0",
		"3./.0",
		"3.0./",
		"3.:.0",
		"3.0.:",
	} {
		if major, recognized := parseMajor(version); recognized {
			t.Fatalf("parseMajor(%q) = (%d, true), want unrecognized", version, major)
		}
		_, err := DecodeResponse(armVersionFrame(version), want)
		if failureCode(t, err) != "provider_protocol_error" {
			t.Fatalf("DecodeResponse(%q) code = %v, want provider_protocol_error", version, err)
		}
		if failureExit(t, err) != 13 {
			t.Fatalf("DecodeResponse(%q) exit = %d, want 13", version, failureExit(t, err))
		}
		requireFrameRefusal(t, err, "protocol_version", "unsupported protocol version")
	}
}

// TestParseMajorLeadingZeroIsClassifiedAsForeign pins the stated bound
// on parseMajor's leading-zero looseness (see the parseMajor doc
// comment): "03.0.0" reports major 3 and takes the foreign-major
// mismatch arm, "02.0.0" reports major 2 and takes the unusable-frame
// arm. Neither is admitted — the version gate admits exactly "2.0.0"
// by string equality — so the looseness only chooses the refusal.
// The production entry point is DecodeResponse.
func TestParseMajorLeadingZeroIsClassifiedAsForeign(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	if major, recognized := parseMajor("03.0.0"); !recognized || major != 3 {
		t.Fatalf("parseMajor(%q) = (%d, %v), want (3, true): leading zeros stay classified, not rejected", "03.0.0", major, recognized)
	}
	_, err := DecodeResponse(armVersionFrame("03.0.0"), want)
	if failureCode(t, err) != "incompatible_protocol" {
		t.Fatalf("DecodeResponse(%q) code = %v, want incompatible_protocol", "03.0.0", err)
	}
	if failureExit(t, err) != 6 {
		t.Fatalf("DecodeResponse(%q) exit = %d, want 6", "03.0.0", failureExit(t, err))
	}
	if major, recognized := parseMajor("02.0.0"); !recognized || major != 2 {
		t.Fatalf("parseMajor(%q) = (%d, %v), want (2, true): leading zeros stay classified, not rejected", "02.0.0", major, recognized)
	}
	_, err = DecodeResponse(armVersionFrame("02.0.0"), want)
	if failureCode(t, err) != "provider_protocol_error" {
		t.Fatalf("DecodeResponse(%q) code = %v, want provider_protocol_error", "02.0.0", err)
	}
	if failureExit(t, err) != 13 {
		t.Fatalf("DecodeResponse(%q) exit = %d, want 13", "02.0.0", failureExit(t, err))
	}
	requireFrameRefusal(t, err, "protocol_version", "unsupported protocol version")
}

// TestDecodeResponseUnrecognizedVersionRunsMemberRules proves the foreignMajor
// peek arms only on recognizable foreign majors: a bare frame (protocol,
// protocol_version, request_id, body — no `ok`) carrying an unrecognized
// version still runs the v2 member-vocabulary check and reports the missing
// `ok` member, rather than skipping to the version gate. Widening the peek
// predicate from `recognized && major != ProtocolMajor` to `recognized ||
// major != ProtocolMajor` arms it for unrecognized versions too, sliding this
// row from member `ok` / `missing member` to member `protocol_version` /
// `unsupported protocol version` with code and exit unchanged. That slide is
// a detail-level weakening outside the F1 delta, pinned here so a future
// widening reddens behaviourally. The production entry point is
// DecodeResponse.
func TestDecodeResponseUnrecognizedVersionRunsMemberRules(t *testing.T) {
	want := mustUUIDv7(t, testRequestID)
	for _, version := range []string{
		"2.abc.def",
		"99999999999999999999.abc.def",
	} {
		if _, recognized := parseMajor(version); recognized {
			t.Fatalf("parseMajor(%q) = recognized, want unrecognized for the peek control", version)
		}
		frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"` + version + `","request_id":"` + testRequestID + `","body":{}}`)
		_, err := DecodeResponse(frame, want)
		requireFrameRefusal(t, err, "ok", "missing member")
		if failureExit(t, err) != 13 {
			t.Fatalf("DecodeResponse(%q) exit = %d, want 13", version, failureExit(t, err))
		}
	}
}
