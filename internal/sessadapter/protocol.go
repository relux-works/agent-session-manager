package sessadapter

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Wire constants of pinned specification Section 7.8. The protocol
// identifier and version are exact: the adapter speaks 1.0.0 in the
// same trusted executable as Provider Protocol 2.0.0, and anything
// else on the wire is a session_adapter_protocol_error, never a
// negotiation.
const (
	// ProtocolID is the exact protocol member every request and
	// response envelope carries.
	ProtocolID = "urn:ax:protocol:session-adapter"
	// ProtocolVersion is the exact protocol_version member this host
	// writes and the only one it accepts a response under.
	ProtocolVersion = "1.0.0"
	// ProtocolMajor is the major this host implements. Section 7.8
	// defines no multi-major negotiation for the adapter, so a
	// different major is an invalid envelope, not a downgrade.
	ProtocolMajor = 1
	// ErrorVersion is the Structured Error version every failure
	// envelope carries: Session Adapter 1.0 statically binds
	// Structured Error 1.1.0.
	ErrorVersion = axerror.Version110
	// MaxFrameBytes bounds one JSONL line in either direction:
	// Section 7.8 inherits Section 7.2's rule that every line is one
	// complete UTF-8 JSON object no larger than 8 MiB.
	MaxFrameBytes = 8 << 20
)

// Operation names one Section 7.8 registry entry. The registry is
// closed: dispatch refuses any other name locally, before any adapter
// surface is touched.
type Operation string

// Section 7.8 operation registry in the section's table order. The
// inventory test derives this list from the pinned specification
// text and requires exact ordered equality, so a dropped, added, or
// reordered entry reddens there rather than passing silently.
const (
	OpManifest       Operation = "manifest"
	OpProbe          Operation = "probe"
	OpDiscover       Operation = "discover"
	OpInspect        Operation = "inspect"
	OpSnapshotProof  Operation = "snapshot-proof"
	OpCapturePlan    Operation = "capture-plan"
	OpCapture        Operation = "capture"
	OpNormalize      Operation = "normalize"
	OpProjectionPlan Operation = "projection-plan"
	OpProject        Operation = "project"
	OpReadBack       Operation = "read-back"
	OpValidate       Operation = "validate"
	OpResumePlan     Operation = "resume-plan"
	OpDoctor         Operation = "doctor"
)

// operationOrder is the single construction site of the dispatch
// registry.
var operationOrder = []Operation{
	OpManifest,
	OpProbe,
	OpDiscover,
	OpInspect,
	OpSnapshotProof,
	OpCapturePlan,
	OpCapture,
	OpNormalize,
	OpProjectionPlan,
	OpProject,
	OpReadBack,
	OpValidate,
	OpResumePlan,
	OpDoctor,
}

// Operations returns the dispatch registry in section order. The
// result is a copy; the registry cannot be mutated through it.
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
// inventory gate can observe every exercised refusal site. Each is
// the single construction site for its failure class; production
// code must not build *axerror.Error any other way.
//
// failInvalid reports a caller error caught before any adapter
// surface is touched: a required set naming an unknown capability, a
// binding the caller built from unobserved facts. failProtocol
// reports an unusable adapter frame: an invalid envelope, a body
// outside its closed shape, a digest mismatch, a partial,
// malformed, over-limit, or escaped result. failUnknownOperation
// reports a name outside the closed registry. failCapability
// reports an operation the adapter does not implement.
// failUnsupportedTuple reports an environment the signed registry
// does not admit. failIntegrity reports a binding observation that
// cannot be reconciled to trusted candidate facts: a failed or
// partial read, or a before-call fact that no longer equals the
// freshly read fact and the Journal binding.
//
// Every constructor takes the human detail first as a string literal
// at the call site and the distinguishing wire fact second, so the
// inventory derives one obligation per literal detail and every
// witness can assert which arm fired through the code plus the
// distinguishing detail, not merely that something refused.
var failInvalid = func(detail string, field string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "invalid_config", Message: "session adapter host refused the call: " + detail, Details: axerror.Details{"field": field}})
}

var failProtocol = func(detail string, member string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "session_adapter_protocol_error", Message: "session adapter host rejected the adapter frame: " + detail, Details: axerror.Details{"member": member}})
}

var failUnknownOperation = func(detail string, operation string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "operation_unknown", Message: "session adapter host refused an unknown operation: " + detail, Details: axerror.Details{"operation": operation}})
}

var failCapability = func(detail string, capability string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "capability_unavailable", Message: "session adapter host refused an unavailable operation: " + detail, Details: axerror.Details{"capability": capability}})
}

// failUnavailable reports an operation the adapter does not
// implement. The operation-to-capability mapping is not stated in
// Section 7.8, so the refusal names the operation rather than
// inventing a capability: the detail key is operation, never a
// capability the contract never defined.
var failUnavailable = func(detail string, operation string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "capability_unavailable", Message: "session adapter host refused an unimplemented operation: " + detail, Details: axerror.Details{"operation": operation}})
}

var failUnsupportedTuple = func(detail string, environment string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "unsupported_environment_tuple", Message: "session adapter host refused an unadmitted environment: " + detail, Details: axerror.Details{"environment": environment}})
}

var failIntegrity = func(detail string, subject string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "integrity_failure", Message: "session adapter binding observation is not trusted state: " + detail, Details: axerror.Details{"subject": subject}})
}

// requestMembers is the exact request envelope member set Section 7.8
// requires: protocol, protocol_version, request_id, operation,
// deadline, and body.
var requestMembers = map[string]bool{
	"protocol":         true,
	"protocol_version": true,
	"request_id":       true,
	"operation":        true,
	"deadline":         true,
	"body":             true,
}

// requestRequired lists requestMembers in a fixed order so an
// envelope missing several members always names the same one.
var requestRequired = []string{
	"protocol",
	"protocol_version",
	"request_id",
	"operation",
	"deadline",
	"body",
}

// successMembers is the exact success envelope member set: the four
// echoed identity fields, ok=true, and body, with no error member.
var successMembers = map[string]bool{
	"protocol":         true,
	"protocol_version": true,
	"request_id":       true,
	"operation":        true,
	"ok":               true,
	"body":             true,
}

// successRequired lists successMembers in a fixed order.
var successRequired = []string{
	"protocol",
	"protocol_version",
	"request_id",
	"operation",
	"ok",
	"body",
}

// failureMembers is the exact failure envelope member set: the four
// echoed identity fields, ok=false, and error, with no body member.
var failureMembers = map[string]bool{
	"protocol":         true,
	"protocol_version": true,
	"request_id":       true,
	"operation":        true,
	"ok":               true,
	"error":            true,
}

// failureRequired lists failureMembers in a fixed order.
var failureRequired = []string{
	"protocol",
	"protocol_version",
	"request_id",
	"operation",
	"ok",
	"error",
}

// Request is one Section 7.8 request envelope as the host wrote it:
// the identity fields plus the canonical body bytes. Responses are
// checked against it, never trusted on their own echo.
type Request struct {
	Operation Operation
	RequestID string
	Body      []byte
}

// DecodeRequestFrame validates one outbound request frame before it
// reaches the adapter surface: one complete object within the frame
// bound, the exact envelope members, the exact protocol identifier
// and version, a UUIDv7 request identifier, a registry operation, a
// timestamp deadline, and an object body. An unknown operation is
// operation_unknown; every other envelope defect is
// session_adapter_protocol_error. The returned body is the raw
// canonical member for the operation-layer checks.
func DecodeRequestFrame(frame []byte) (Request, error) {
	if len(frame) == 0 {
		failure, err := failProtocol("request frame is empty", "")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if len(frame) > MaxFrameBytes {
		failure, err := failProtocol("request frame exceeds the 8 MiB bound", "")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	members, fault := decodeStrictObject(frame)
	if fault != nil {
		failure, err := failProtocol("request envelope "+fault.detail, fault.member)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if name, unknown := unknownMember(members, requestMembers); unknown {
		failure, err := failProtocol("request envelope carries unknown member", name)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if name, missing := missingMember(members, requestRequired); missing {
		failure, err := failProtocol("request envelope misses a required member", name)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if protocol, ok := rawString(members["protocol"]); !ok || protocol != ProtocolID {
		failure, err := failProtocol("request protocol is not the session adapter", "protocol")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if version, ok := rawString(members["protocol_version"]); !ok || version != ProtocolVersion {
		failure, err := failProtocol("request protocol version is not 1.0.0", "protocol_version")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	requestID, ok := rawString(members["request_id"])
	if !ok {
		failure, err := failProtocol("request identifier is not a string", "request_id")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if _, err := scalar.ParseUUIDv7(requestID); err != nil {
		failure, err := failProtocol("request identifier is not a UUIDv7", "request_id")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	operation, ok := rawString(members["operation"])
	if !ok {
		failure, err := failProtocol("request operation is not a string", "operation")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if !validOperation(operation) {
		failure, err := failUnknownOperation("request names an operation outside the closed registry", operation)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if _, ok := checkTimestamp(members["deadline"]); !ok {
		failure, err := failProtocol("request deadline is not a timestamp", "deadline")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	body := bytesTrimSpace(members["body"])
	if _, fault := decodeStrictObject(body); fault != nil {
		failure, err := failProtocol("request body "+fault.detail, "body")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	return Request{Operation: Operation(operation), RequestID: requestID, Body: body}, nil
}

// CheckSuccessEnvelope validates one inbound success frame against
// the request that caused it: one complete object within the frame
// bound, the exact success members, the four identity fields echoed
// exactly, ok=true, an object body, and no error member. Body and
// error are disjoint, never nullable sentinels: a frame carrying
// both or neither is a protocol error. The returned body is the raw
// member for the operation-layer checks.
func CheckSuccessEnvelope(frame []byte, want Request) ([]byte, error) {
	if len(frame) == 0 {
		failure, err := failProtocol("success frame is empty", "")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if len(frame) > MaxFrameBytes {
		failure, err := failProtocol("success frame exceeds the 8 MiB bound", "")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	members, fault := decodeStrictObject(frame)
	if fault != nil {
		failure, err := failProtocol("success envelope "+fault.detail, fault.member)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, unknown := unknownMember(members, successMembers); unknown {
		failure, err := failProtocol("success envelope carries unknown member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, missing := missingMember(members, successRequired); missing {
		failure, err := failProtocol("success envelope misses a required member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if protocol, ok := rawString(members["protocol"]); !ok || protocol != ProtocolID {
		failure, err := failProtocol("success protocol does not echo the request", "protocol")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if version, ok := rawString(members["protocol_version"]); !ok || version != ProtocolVersion {
		failure, err := failProtocol("success protocol version does not echo the request", "protocol_version")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if requestID, ok := rawString(members["request_id"]); !ok || requestID != want.RequestID {
		failure, err := failProtocol("success request identifier does not echo the request", "request_id")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if operation, ok := rawString(members["operation"]); !ok || operation != string(want.Operation) {
		failure, err := failProtocol("success operation does not echo the request", "operation")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	positive, ok := rawBool(members["ok"])
	if !ok || !positive {
		failure, err := failProtocol("success envelope does not carry ok=true", "ok")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	body := bytesTrimSpace(members["body"])
	if _, fault := decodeStrictObject(body); fault != nil {
		failure, err := failProtocol("success body "+fault.detail, "body")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	return body, nil
}

// CheckFailureEnvelope validates one inbound failure frame against
// the request that caused it: the exact failure members, the four
// identity fields echoed exactly, ok=false, a closed error object,
// and no body member. The error object itself must decode as a
// Structured Error under the adapter's bound version; a failure that
// cannot be read is an integrity failure, never a retryable result.
func CheckFailureEnvelope(frame []byte, want Request) (*axerror.Error, error) {
	if len(frame) == 0 {
		failure, err := failProtocol("failure frame is empty", "")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if len(frame) > MaxFrameBytes {
		failure, err := failProtocol("failure frame exceeds the 8 MiB bound", "")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	members, fault := decodeStrictObject(frame)
	if fault != nil {
		failure, err := failProtocol("failure envelope "+fault.detail, fault.member)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, unknown := unknownMember(members, failureMembers); unknown {
		failure, err := failProtocol("failure envelope carries unknown member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, missing := missingMember(members, failureRequired); missing {
		failure, err := failProtocol("failure envelope misses a required member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if protocol, ok := rawString(members["protocol"]); !ok || protocol != ProtocolID {
		failure, err := failProtocol("failure protocol does not echo the request", "protocol")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if version, ok := rawString(members["protocol_version"]); !ok || version != ProtocolVersion {
		failure, err := failProtocol("failure protocol version does not echo the request", "protocol_version")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if requestID, ok := rawString(members["request_id"]); !ok || requestID != want.RequestID {
		failure, err := failProtocol("failure request identifier does not echo the request", "request_id")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if operation, ok := rawString(members["operation"]); !ok || operation != string(want.Operation) {
		failure, err := failProtocol("failure operation does not echo the request", "operation")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	positive, ok := rawBool(members["ok"])
	if !ok || positive {
		failure, err := failProtocol("failure envelope does not carry ok=false", "ok")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	reported, err := axerror.Decode(ErrorVersion, bytesTrimSpace(members["error"]))
	if err != nil {
		failure, faultErr := failIntegrity("failure error is not a Structured Error 1.1", "error")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	return reported, nil
}

// bytesTrimSpace renders one raw member without surrounding
// whitespace. json.RawMessage values parsed from a strict object
// carry no leading whitespace, but a member re-encoded by the caller
// may; trimming keeps the downstream strict decode total.
func bytesTrimSpace(raw json.RawMessage) json.RawMessage {
	trimmed := json.RawMessage(bytes.TrimSpace(raw))
	if len(trimmed) == 0 {
		return raw
	}
	return trimmed
}
