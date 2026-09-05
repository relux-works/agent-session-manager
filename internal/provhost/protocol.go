package provhost

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Wire constants of pinned specification Section 7.2. The protocol
// identifier and version are exact: a different major is never negotiated,
// and anything other than 2.0.0 on the wire is refused or reported as a
// major mismatch, never accepted.
const (
	// ProtocolID is the exact protocol member every request and response
	// envelope carries.
	ProtocolID = "urn:ax:protocol:provider"
	// ProtocolVersion is the exact protocol_version member this host
	// writes and the only one it accepts a response under.
	ProtocolVersion = "2.0.0"
	// ProtocolMajor is the major this host implements. Responses under
	// any other recognizable major are reported as incompatible_protocol
	// without trusting their payload.
	ProtocolMajor = 2
	// MaxFrameBytes bounds one JSONL line in either direction: Section
	// 7.2 requires every line to be one complete UTF-8 JSON object no
	// larger than 8 MiB.
	MaxFrameBytes = 8 << 20
	// StderrCapBytes caps captured plugin stderr. It is an implementation
	// resource bound, not a protocol limit: diagnostics beyond it are
	// dropped. Failure details carry the failure class and member names
	// only, never stderr content.
	StderrCapBytes = 1 << 20
)

// Operation names one Section 7.5 registry entry. The registry is closed:
// dispatch refuses any other name locally, before any process starts.
type Operation string

// Section 7.5 operation registry in the Section 7.3 manifest order. The
// inventory test derives this list from the pinned specification text and
// requires exact ordered equality, so a dropped, added, or reordered entry
// reddens there rather than passing silently.
const (
	OpManifest            Operation = "manifest"
	OpProbe               Operation = "probe"
	OpLaunch              Operation = "launch"
	OpIdentifySession     Operation = "identify-session"
	OpQuiesce             Operation = "quiesce"
	OpNativeStorePlan     Operation = "native-store-plan"
	OpCapture             Operation = "capture"
	OpMaterialize         Operation = "materialize"
	OpMaterializeStatus   Operation = "materialize-status"
	OpMaterializeCommit   Operation = "materialize-commit"
	OpMaterializeRollback Operation = "materialize-rollback"
	OpResume              Operation = "resume"
	OpFork                Operation = "fork"
	OpStop                Operation = "stop"
	OpDoctor              Operation = "doctor"
)

// operationOrder is the single construction site of the dispatch registry.
var operationOrder = []Operation{
	OpManifest,
	OpProbe,
	OpLaunch,
	OpIdentifySession,
	OpQuiesce,
	OpNativeStorePlan,
	OpCapture,
	OpMaterialize,
	OpMaterializeStatus,
	OpMaterializeCommit,
	OpMaterializeRollback,
	OpResume,
	OpFork,
	OpStop,
	OpDoctor,
}

// Operations returns the dispatch registry in manifest order. The result
// is a copy; the registry cannot be mutated through it.
func Operations() []string {
	out := make([]string, 0, len(operationOrder))
	for _, operation := range operationOrder {
		out = append(out, string(operation))
	}
	return out
}

// validOperation reports whether the name is a registry member.
func validOperation(name string) bool {
	for _, operation := range operationOrder {
		if string(operation) == name {
			return true
		}
	}
	return false
}

// The refusal constructors below are declared as variables so the
// inventory gate can observe every exercised refusal site. Each is the
// single construction site for its failure class; production code must
// not build *axerror.Error any other way.
//
// failInvalid reports a caller error caught before any process starts:
// an unknown operation, a stale deadline, or an unframeable request body.
// failProtocol reports an unusable plugin frame through the Section 15.1
// provider-stdio otherwise branch. failMismatch reports a recognizable
// major mismatch through the same table's mismatch branch, without
// trusting the foreign payload. failProcess reports a crash or transport
// failure. failTimeout reports a plugin that exceeded its request
// deadline. failIntegrity reports a status observation that cannot be
// reconciled to durable state.
var failInvalid = func(detail string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: axerror.Version100, Code: "invalid_config", Message: "provider host refused the request: " + detail, Details: axerror.Details{}})
}

var failProtocol = func(detail string, member string) (*axerror.Error, error) {
	return axerror.LocalFromUntrusted(axerror.SurfaceProviderStdio, axerror.OutcomeUnusableFrame, "provider host rejected the plugin frame: "+detail, axerror.NoIDs(), axerror.Details{"member": member}, nil)
}

var failMismatch = func(detail string, observed string) (*axerror.Error, error) {
	return axerror.LocalFromUntrusted(axerror.SurfaceProviderStdio, axerror.OutcomeRecognizableMajorMismatch, "provider host rejected a foreign protocol major: "+detail, axerror.NoIDs(), axerror.Details{"observed": observed}, nil)
}

var failProcess = func(detail string, cause error) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: axerror.Version100, Code: "provider_process_failed", Message: "provider plugin process failed: " + detail, Details: axerror.Details{}, Cause: cause})
}

var failTimeout = func(detail string, millis int64) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: axerror.Version100, Code: "provider_timeout", Message: "provider plugin exceeded its deadline: " + detail, Details: axerror.Details{"deadline_ms": json.Number(strconv.FormatInt(millis, 10))}})
}

var failIntegrity = func(detail string, statusState string, materializationID string, transactionID string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: axerror.Version100, Code: "integrity_failure", Message: "provider status observation is not durable state: " + detail, Details: axerror.Details{"status_state": statusState, "materialization_id": materializationID, "transaction_id": transactionID}})
}

// Request is one Section 7.2 request envelope. Body crosses the transport
// opaquely: it must be a JSON object, and its members are interpreted by
// the operation layer, except for the status recovery read.
type Request struct {
	Operation Operation
	RequestID scalar.UUIDv7
	Deadline  scalar.Timestamp
	Body      json.RawMessage
}

// Response is one accepted success envelope. Body is the raw
// operation-specific object for the operation layer to interpret. A
// failure envelope never produces a Response: DecodeResponse returns the
// bound child error instead, and a failure envelope carrying a body is
// refused outright.
type Response struct {
	Body json.RawMessage
}

// wireRequest is the exact emitted member set: protocol, protocol_version,
// request_id, operation, deadline, and body, in this fixed order.
type wireRequest struct {
	Protocol        string           `json:"protocol"`
	ProtocolVersion string           `json:"protocol_version"`
	RequestID       scalar.UUIDv7    `json:"request_id"`
	Operation       string           `json:"operation"`
	Deadline        scalar.Timestamp `json:"deadline"`
	Body            json.RawMessage  `json:"body"`
}

// EncodeRequest frames one request as a single JSONL line without the
// terminator. The deadline must be in the future at now, which the caller
// supplies so tests stay deterministic; Host passes its clock.
func EncodeRequest(req Request, now time.Time) ([]byte, error) {
	frame, _, err := encodeFrame(req, now)
	return frame, err
}

// encodeFrame validates and frames a request, also returning the parsed
// deadline instant so Host.Call validates once at a single site.
func encodeFrame(req Request, now time.Time) ([]byte, time.Time, error) {
	if !validOperation(string(req.Operation)) {
		failure, err := failInvalid("unknown operation")
		if err != nil {
			return nil, time.Time{}, err
		}
		return nil, time.Time{}, failure
	}
	if _, err := scalar.ParseUUIDv7(req.RequestID.String()); err != nil {
		failure, fault := failInvalid("request_id is not a UUIDv7")
		if fault != nil {
			return nil, time.Time{}, fault
		}
		return nil, time.Time{}, failure
	}
	deadline, err := req.Deadline.Time()
	if err != nil {
		failure, fault := failInvalid("deadline is not a timestamp")
		if fault != nil {
			return nil, time.Time{}, fault
		}
		return nil, time.Time{}, failure
	}
	if !deadline.After(now) {
		failure, fault := failInvalid("deadline is not in the future")
		if fault != nil {
			return nil, time.Time{}, fault
		}
		return nil, time.Time{}, failure
	}
	if !isJSONObject(req.Body) {
		failure, fault := failInvalid("body is not a JSON object")
		if fault != nil {
			return nil, time.Time{}, fault
		}
		return nil, time.Time{}, failure
	}
	// The operation, request ID, deadline, and body above were each
	// validated before this marshal, so it cannot fail on a reachable
	// path: the error propagates as a plain Go error rather than
	// through a refusal site no negative test could exercise.
	frame, err := json.Marshal(wireRequest{Protocol: ProtocolID, ProtocolVersion: ProtocolVersion, RequestID: req.RequestID, Operation: string(req.Operation), Deadline: req.Deadline, Body: req.Body})
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(frame) > MaxFrameBytes {
		failure, fault := failInvalid("request frame exceeds 8 MiB")
		if fault != nil {
			return nil, time.Time{}, fault
		}
		return nil, time.Time{}, failure
	}
	return frame, deadline, nil
}

// isJSONObject reports whether data is well-formed JSON whose top-level
// value is an object. Bodies cross the transport opaquely, so this is the
// only body check here: member vocabularies belong to the operation layer.
func isJSONObject(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	return json.Valid(trimmed)
}

// frameFault classifies a malformed plugin frame without minting error
// values: dynamic facts travel in axerror details, never in Go error text,
// so production code builds no raw errors.
type frameFault struct {
	detail string
	member string
}

// decodeStrictObject parses one complete JSON object with duplicate
// member detection. A repeated member has no single value, so it is a
// fault, not a last-wins read. The UTF-8 gate runs first: encoding/json
// would silently replace non-UTF-8 bytes, including raw WTF-8 surrogate
// encodings such as ED A0 80, with U+FFFD instead of failing, so the
// raw bytes are screened before any member is trusted. The
// lone-surrogate gate runs next: encoding/json would likewise replace
// a lone \ud800-style escape with U+FFFD (Section 1.6: decoders MUST
// reject lone surrogate code points).
func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	if !utf8.Valid(data) {
		return nil, &frameFault{detail: "not valid UTF-8", member: ""}
	}
	if hasLoneSurrogateEscape(data) {
		return nil, &frameFault{detail: "lone surrogate escape", member: ""}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, &frameFault{detail: "not a JSON object", member: ""}
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil, &frameFault{detail: "not a JSON object", member: ""}
	}
	members := map[string]json.RawMessage{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, &frameFault{detail: "not a JSON object", member: ""}
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, &frameFault{detail: "not a JSON object", member: ""}
		}
		if _, duplicate := members[key]; duplicate {
			return nil, &frameFault{detail: "duplicate member", member: key}
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, &frameFault{detail: "not a JSON object", member: key}
		}
		members[key] = raw
	}
	if token, err := decoder.Token(); err != nil {
		return nil, &frameFault{detail: "not a JSON object", member: ""}
	} else if delim, ok := token.(json.Delim); !ok || delim != '}' {
		return nil, &frameFault{detail: "not a JSON object", member: ""}
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, &frameFault{detail: "trailing data after the object", member: ""}
	}
	return members, nil
}

// responseMembers is the exact Section 7.2 response member set: the four
// framing members plus body on success or error on failure, never both,
// never neither, and nothing else. It is package-level so the closed
// vocabulary census derives it: widening the accepted set here must redden
// TestResponseMembersAreDerivedFromSpec, not pass silently.
var responseMembers = map[string]bool{
	"protocol":         true,
	"protocol_version": true,
	"request_id":       true,
	"ok":               true,
	"body":             true,
	"error":            true,
}

func checkResponseMembers(members map[string]json.RawMessage, ok bool) *frameFault {
	required := []string{"protocol", "protocol_version", "request_id", "ok"}
	for _, name := range required {
		if _, present := members[name]; !present {
			return &frameFault{detail: "missing member", member: name}
		}
	}
	if ok {
		if _, present := members["body"]; !present {
			return &frameFault{detail: "missing member", member: "body"}
		}
		if _, forbidden := members["error"]; forbidden {
			return &frameFault{detail: "success envelope carries error", member: "error"}
		}
	} else {
		if _, present := members["error"]; !present {
			return &frameFault{detail: "missing member", member: "error"}
		}
		if _, forbidden := members["body"]; forbidden {
			return &frameFault{detail: "failure envelope carries body", member: "body"}
		}
	}
	for name := range members {
		if !responseMembers[name] {
			return &frameFault{detail: "unknown member", member: name}
		}
	}
	return nil
}

// parseMajor extracts the major from a strict numeric X.Y.Z version.
// Anything else is not a recognizable major, so the frame is unusable
// rather than a mismatch.
func parseMajor(version string) (int, bool) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, false
	}
	major := 0
	for i := 0; i < len(parts[0]); i++ {
		digit := parts[0][i]
		if digit < '0' || digit > '9' {
			return 0, false
		}
		major = major*10 + int(digit-'0')
	}
	if len(parts[0]) == 0 {
		return 0, false
	}
	for _, rest := range parts[1:] {
		if len(rest) == 0 {
			return 0, false
		}
		for i := 0; i < len(rest); i++ {
			if rest[i] < '0' || rest[i] > '9' {
				return 0, false
			}
		}
	}
	return major, true
}

// providerContract is the static Section 7.2 binding every failure
// envelope is decoded under: provider protocol major 2 binds Structured
// Error 1.0.0. The version is never taken from the document.
var providerContract = axerror.ContainingContract{ID: "urn:ax:protocol:provider", Major: 2}

// DecodeResponse validates one stdout line as the single response frame
// for the request carrying wantRequestID. It returns the success body, or
// the bound child failure, or a local failure that trusts nothing from
// the frame. A recognizable foreign major yields incompatible_protocol;
// every other unusable frame yields provider_protocol_error.
func DecodeResponse(frame []byte, wantRequestID scalar.UUIDv7) (Response, error) {
	if len(frame) > MaxFrameBytes {
		failure, err := failProtocol("frame exceeds 8 MiB", "")
		if err != nil {
			return Response{}, err
		}
		return Response{}, failure
	}
	if !utf8.Valid(frame) {
		failure, err := failProtocol("frame is not UTF-8", "")
		if err != nil {
			return Response{}, err
		}
		return Response{}, failure
	}
	members, fault := decodeStrictObject(frame)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return Response{}, err
		}
		return Response{}, failure
	}
	// foreignMajor peeks at the envelope identity before any v2
	// member rule runs. A recognizable foreign major is
	// incompatible_protocol no matter what members it carries, so
	// the v2 member vocabulary below must not judge it first: a
	// 3.0.0 envelope carrying v3 members, or omitting the v2 body,
	// is a foreign major, not an unknown or missing member. The
	// peek trusts nothing and refuses nothing: it only arms when
	// the protocol member is exactly this protocol and the version
	// member is a string with a recognizable foreign major. A frame
	// without our protocol identity, or without a readable version,
	// falls through to the member rules, which keep their verdicts.
	foreignMajor := ""
	if rawProtocol, present := members["protocol"]; present {
		var protocol string
		if err := json.Unmarshal(rawProtocol, &protocol); err == nil && protocol == ProtocolID {
			if rawVersion, present := members["protocol_version"]; present {
				var version string
				if err := json.Unmarshal(rawVersion, &version); err == nil && version != ProtocolVersion {
					if major, recognized := parseMajor(version); recognized && major != ProtocolMajor {
						foreignMajor = version
					}
				}
			}
		}
	}
	var ok bool
	if foreignMajor == "" {
		if raw, present := members["ok"]; !present {
			failure, err := failProtocol("missing member", "ok")
			if err != nil {
				return Response{}, err
			}
			return Response{}, failure
		} else if err := json.Unmarshal(raw, &ok); err != nil {
			failure, fault := failProtocol("member is not a boolean", "ok")
			if fault != nil {
				return Response{}, fault
			}
			return Response{}, failure
		}
		if fault := checkResponseMembers(members, ok); fault != nil {
			failure, err := failProtocol(fault.detail, fault.member)
			if err != nil {
				return Response{}, err
			}
			return Response{}, failure
		}
	}
	var protocol string
	if err := json.Unmarshal(members["protocol"], &protocol); err != nil || protocol != ProtocolID {
		failure, fault := failProtocol("not a provider envelope", "protocol")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	var version string
	if err := json.Unmarshal(members["protocol_version"], &version); err != nil {
		failure, fault := failProtocol("member is not a string", "protocol_version")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	if version != ProtocolVersion {
		if major, recognized := parseMajor(version); recognized && major != ProtocolMajor {
			failure, err := failMismatch("foreign protocol major", version)
			if err != nil {
				return Response{}, err
			}
			return Response{}, failure
		}
		failure, fault := failProtocol("unsupported protocol version", "protocol_version")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	var requestID string
	if err := json.Unmarshal(members["request_id"], &requestID); err != nil {
		failure, fault := failProtocol("member is not a string", "request_id")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	if _, err := scalar.ParseUUIDv7(requestID); err != nil {
		failure, fault := failProtocol("request_id is not a UUIDv7", "request_id")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	if requestID != wantRequestID.String() {
		failure, fault := failProtocol("request_id does not match the request", "request_id")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	if ok {
		if !isJSONObject(members["body"]) {
			failure, fault := failProtocol("body is not a JSON object", "body")
			if fault != nil {
				return Response{}, fault
			}
			return Response{}, failure
		}
		return Response{Body: append(json.RawMessage(nil), members["body"]...)}, nil
	}
	if !isJSONObject(members["error"]) {
		failure, fault := failProtocol("error is not a JSON object", "error")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	child, err := axerror.DecodeBound(providerContract, members["error"])
	if err != nil {
		failure, fault := failProtocol("error is not a bound Structured Error 1.0.0", "error")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	return Response{}, child
}
