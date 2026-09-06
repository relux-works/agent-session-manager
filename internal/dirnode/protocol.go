package dirnode

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Wire constants of pinned specification Section 7.9. Directory
// Node protocol 2.0.0 is the current separately negotiated
// read-mostly façade; 1.0.0 remains an immutable legacy wire
// contract. Neither major is a negotiation: a different major is an
// invalid envelope, never a downgrade, except through the exact
// manifest downgrade tuple DecideBootstrapStep recognizes.
const (
	// ProtocolID is the exact protocol member every request and
	// response envelope carries.
	ProtocolID = "urn:ax:protocol:session-directory-node"
	// RequestSchema is the exact schema member every request
	// envelope carries.
	RequestSchema = "urn:ax:schema:session-directory-node-request"
	// ResponseSchema is the exact schema member every response
	// envelope carries. The response carries its own schema
	// member because the downgrade tuple names a response
	// schema_version distinct from every echoed request member;
	// a response without one cannot carry that tuple.
	ResponseSchema = "urn:ax:schema:session-directory-node-response"
	// ManifestSchema is the exact schema identifier the manifest
	// body carries.
	ManifestSchema = "urn:ax:schema:session-directory-node-manifest"
	// ManifestSchemaVersion is the only manifest version this host
	// accepts under either major.
	ManifestSchemaVersion = "1.0.0"
	// ResponseSchemaVersion is the only response schema_version
	// this host accepts under either major: both majors bind
	// Directory Node Response 1.0.0.
	ResponseSchemaVersion = "1.0.0"
	// MajorV1 is the legacy wire major.
	MajorV1 = 1
	// MajorV2 is the current wire major.
	MajorV2 = 2
	// ErrorVersion is the Structured Error version every failure
	// envelope carries: Directory Node 1 and 2 statically bind
	// Structured Error 1.2.0.
	ErrorVersion = axerror.Version120
	// MaxFrameBytes bounds one JSONL line in either direction:
	// Section 7.9 inherits the rule that every line is one
	// complete UTF-8 JSON object no larger than 8 MiB.
	MaxFrameBytes = 8 << 20
	// MinDeadlineMS and MaxDeadlineMS bound deadline_ms to
	// uint53[1..3600000].
	MinDeadlineMS = uint64(1)
	MaxDeadlineMS = uint64(3600000)
)

// supportedMajors is the single construction site of the local
// major registry, in strictly descending numeric order: peers
// choose the highest mutually supported major.
var supportedMajors = []int{MajorV2, MajorV1}

// SupportedMajors returns the locally supported Directory Node
// majors in strictly descending numeric order. The result is a
// copy; the registry cannot be mutated through it.
func SupportedMajors() []int {
	return append([]int(nil), supportedMajors...)
}

// VersionForMajor binds a supported major to its exact N.0.0 wire
// version. The second result is false for any other major: there is
// no mapping to derive, only the two closed bindings.
func VersionForMajor(major int) (string, bool) {
	switch major {
	case MajorV1:
		return "1.0.0", true
	case MajorV2:
		return "2.0.0", true
	default:
		return "", false
	}
}

// MajorForVersion inverts VersionForMajor: only the two exact N.0.0
// versions name a major. Any other version — including a patched
// 2.0.1 — is not a supported major under another spelling.
func MajorForVersion(version string) (int, bool) {
	switch version {
	case "1.0.0":
		return MajorV1, true
	case "2.0.0":
		return MajorV2, true
	default:
		return 0, false
	}
}

// Operation names one Section 7.9 registry entry. The registry is
// closed: dispatch refuses any other name locally, before any node
// surface is touched.
type Operation string

// Section 7.9 operation registry in the section's table order. The
// inventory test derives this list from the pinned specification
// text and requires exact ordered equality, so a dropped, added, or
// reordered entry reddens there rather than passing silently.
const (
	OpManifest            Operation = "manifest"
	OpProbe               Operation = "probe"
	OpScan                Operation = "scan"
	OpInventory           Operation = "inventory"
	OpPreview             Operation = "preview"
	OpEnrichmentPlan      Operation = "enrichment-plan"
	OpEnrichmentRun       Operation = "enrichment-run"
	OpEnrichmentStatus    Operation = "enrichment-status"
	OpContinuationInspect Operation = "continuation-inspect"
	OpRuntimeObserve      Operation = "runtime-observe"
	OpDoctor              Operation = "doctor"
)

// operationOrder is the single construction site of the dispatch
// registry.
var operationOrder = []Operation{
	OpManifest,
	OpProbe,
	OpScan,
	OpInventory,
	OpPreview,
	OpEnrichmentPlan,
	OpEnrichmentRun,
	OpEnrichmentStatus,
	OpContinuationInspect,
	OpRuntimeObserve,
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

// framedOperations is the scoped subset of the registry this host
// frames in this deliverable: manifest, probe, and scan request
// and success bodies, plus the Session Directory Query contract
// of Section 10.8.5. The remaining eight registry members —
// inventory, preview, enrichment-plan, enrichment-run,
// enrichment-status, continuation-inspect, runtime-observe, and
// doctor — carry nested observation, lineage, enrichment, and
// runtime types whose content owners are the
// shared-environment and boundary leaves of this Story; this
// package frames no body for them, and the inventory test pins
// that boundary by requiring the framed set to equal exactly
// these three names. There is no dispatcher in this package that
// could route around the subset: callers reach only the entry
// points that exist.
var framedOperations = []Operation{
	OpManifest,
	OpProbe,
	OpScan,
}

// FramedOperations returns the scoped framed subset in section
// order. The result is a copy.
func FramedOperations() []string {
	out := make([]string, 0, len(framedOperations))
	for _, operation := range framedOperations {
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
// failInvalid reports a caller error caught before any node surface
// is touched: an unsupported local major, a binding the caller built
// from unobserved facts, a reused attempt process. failViolation
// reports an unusable node frame: an invalid envelope, a body
// outside its closed shape, a mislabeled major, a partial,
// malformed, over-limit, or escaped result. failUnknownOperation
// reports a name outside the closed registry.
// failDowngrade reports the terminal no-common-major outcome with
// incompatible_protocol. failIntegrity reports a binding observation
// that cannot be reconciled to trusted facts: a failed or partial
// read, or façade bindings that contradict the observed candidate.
// failIdempotency reports a changed body under a recorded
// (operation, operation_id). failQuery reports a Session Directory
// Query outside its closed shape; cursor reuse after a bound change
// lands here because the pinned error registry does not register
// the query_cursor_mismatch code the Section 10.8.5 prose names.
// failTransport reports a missing response: timeout, signal, or exit
// without one valid frame.
//
// Every constructor takes the human detail first as a string literal
// at the call site and the distinguishing wire fact second, so the
// inventory derives one obligation per literal detail and every
// witness can assert which arm fired through the code plus the
// distinguishing detail, not merely that something refused.
var failInvalid = func(detail string, field string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "invalid_config", Message: "directory node host refused the call: " + detail, Details: axerror.Details{"field": field}})
}

var failViolation = func(detail string, member string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "adapter_protocol_violation", Message: "directory node host rejected the node frame: " + detail, Details: axerror.Details{"member": member}})
}

var failUnknownOperation = func(detail string, operation string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "operation_unknown", Message: "directory node host refused an unknown operation: " + detail, Details: axerror.Details{"operation": operation}})
}

var failDowngrade = func(detail string, subject string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "incompatible_protocol", Message: "directory node host found no common major: " + detail, Details: axerror.Details{"subject": subject}})
}

var failIntegrity = func(detail string, subject string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "integrity_failure", Message: "directory node binding observation is not trusted state: " + detail, Details: axerror.Details{"subject": subject}})
}

var failIdempotency = func(detail string, operation string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "idempotency_mismatch", Message: "directory node host refused a changed idempotent body: " + detail, Details: axerror.Details{"operation": operation}})
}

var failQuery = func(detail string, member string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "query_invalid", Message: "directory node host refused the directory query: " + detail, Details: axerror.Details{"member": member}})
}

var failTransport = func(detail string, subject string) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{Version: ErrorVersion, Code: "transport_failure", Message: "directory node attempt produced no usable response: " + detail, Details: axerror.Details{"subject": subject}})
}

// RefuseUnknownOperation refuses a dispatch name outside the closed
// registry with operation_unknown. Dispatch refuses locally, before
// any node surface is touched.
func RefuseUnknownOperation(name string) error {
	failure, err := failUnknownOperation("dispatch names an operation outside the closed registry", name)
	if err != nil {
		return err
	}
	return failure
}

// requestMembers is the exact request envelope member set Section
// 7.9 requires: schema, schema_version, protocol, protocol_version,
// request_id, operation, deadline_ms, and body.
var requestMembers = map[string]bool{
	"schema":           true,
	"schema_version":   true,
	"protocol":         true,
	"protocol_version": true,
	"request_id":       true,
	"operation":        true,
	"deadline_ms":      true,
	"body":             true,
}

// requestRequired lists requestMembers in a fixed order so an
// envelope missing several members always names the same one.
var requestRequired = []string{
	"schema",
	"schema_version",
	"protocol",
	"protocol_version",
	"request_id",
	"operation",
	"deadline_ms",
	"body",
}

// responseMembers is the exact response envelope member set both
// success and failure carry: the response schema identity, the four
// echoed request fields, ok, and exactly one of body or error.
var responseMembers = map[string]bool{
	"schema":           true,
	"schema_version":   true,
	"protocol":         true,
	"protocol_version": true,
	"request_id":       true,
	"operation":        true,
	"ok":               true,
	"body":             true,
	"error":            true,
}

// responseRequired lists the members every response carries
// regardless of branch.
var responseRequired = []string{
	"schema",
	"schema_version",
	"protocol",
	"protocol_version",
	"request_id",
	"operation",
	"ok",
}

// Request is one Section 7.9 request envelope as the host wrote it:
// the negotiated major, the identity fields, and the canonical body
// bytes. Responses are checked against it, never trusted on their
// own echo.
type Request struct {
	Major     int
	Operation Operation
	RequestID string
	Body      []byte
}

// EncodeRequest builds one manifest-attempt-style request frame for
// the negotiated major: protocol_version and request schema_version
// both bind the exact attempted N.0.0 version, and the body must be
// one complete object. A major outside the local registry is an
// invalid caller request, not a peer downgrade.
func EncodeRequest(major int, operation Operation, requestID string, deadlineMS uint64, body []byte) ([]byte, error) {
	version, ok := VersionForMajor(major)
	if !ok {
		failure, err := failInvalid("request major is outside the locally supported registry", "protocol_version")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if !validOperation(string(operation)) {
		failure, err := failUnknownOperation("request names an operation outside the closed registry", string(operation))
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if _, err := scalar.ParseUUIDv7(requestID); err != nil {
		failure, err := failInvalid("request identifier is not a UUIDv7", "request_id")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if deadlineMS < MinDeadlineMS || deadlineMS > MaxDeadlineMS {
		failure, err := failInvalid("request deadline is outside uint53[1..3600000]", "deadline_ms")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	trimmed := bytes.TrimSpace(body)
	if _, fault := decodeStrictObject(trimmed); fault != nil {
		failure, err := failViolation("request body "+fault.detail, "body")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	frame, err := json.Marshal(map[string]any{
		"schema":           RequestSchema,
		"schema_version":   version,
		"protocol":         ProtocolID,
		"protocol_version": version,
		"request_id":       requestID,
		"operation":        string(operation),
		"deadline_ms":      deadlineMS,
		"body":             json.RawMessage(trimmed),
	})
	if err != nil {
		failure, faultErr := failViolation("request frame is not encodable", "")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	return frame, nil
}

// DecodeRequestFrame validates one outbound request frame before it
// reaches the node surface: one complete object within the frame
// bound, the exact envelope members, the exact schema and protocol
// identifiers, protocol_version and schema_version bound together to
// one supported N.0.0, a UUIDv7 request identifier, a registry
// operation, a deadline_ms in uint53[1..3600000], and an object
// body. An unknown operation is operation_unknown; a version pair
// that relabels one major as another is an
// adapter_protocol_violation; every other envelope defect is an
// adapter_protocol_violation. The returned body is the raw member
// for the operation-layer checks.
func DecodeRequestFrame(frame []byte) (Request, error) {
	if len(frame) == 0 {
		failure, err := failViolation("request frame is empty", "")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if len(frame) > MaxFrameBytes {
		failure, err := failViolation("request frame exceeds the 8 MiB bound", "")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	members, fault := decodeStrictObject(frame)
	if fault != nil {
		failure, err := failViolation("request envelope "+fault.detail, fault.member)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if name, unknown := unknownMember(members, requestMembers); unknown {
		failure, err := failViolation("request envelope carries unknown member", name)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if name, missing := missingMember(members, requestRequired); missing {
		failure, err := failViolation("request envelope misses a required member", name)
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != RequestSchema {
		failure, err := failViolation("request schema is not the directory node request", "schema")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	protocolVersion, ok := rawString(members["protocol_version"])
	if !ok {
		failure, err := failViolation("request protocol version is not a string", "protocol_version")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	schemaVersion, ok := rawString(members["schema_version"])
	if !ok {
		failure, err := failViolation("request schema version is not a string", "schema_version")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	major, ok := MajorForVersion(protocolVersion)
	if !ok || schemaVersion != protocolVersion {
		failure, err := failViolation("request versions do not bind one supported major", "protocol_version")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if protocol, ok := rawString(members["protocol"]); !ok || protocol != ProtocolID {
		failure, err := failViolation("request protocol is not the directory node", "protocol")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	requestID, ok := rawString(members["request_id"])
	if !ok {
		failure, err := failViolation("request identifier is not a string", "request_id")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	if _, err := scalar.ParseUUIDv7(requestID); err != nil {
		failure, err := failViolation("request identifier is not a UUIDv7", "request_id")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	operation, ok := rawString(members["operation"])
	if !ok {
		failure, err := failViolation("request operation is not a string", "operation")
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
	if _, ok := checkUint53Bounds(members["deadline_ms"], MinDeadlineMS, MaxDeadlineMS); !ok {
		failure, err := failViolation("request deadline is not uint53[1..3600000]", "deadline_ms")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	body := bytes.TrimSpace(members["body"])
	if _, fault := decodeStrictObject(body); fault != nil {
		failure, err := failViolation("request body "+fault.detail, "body")
		if err != nil {
			return Request{}, err
		}
		return Request{}, failure
	}
	return Request{Major: major, Operation: Operation(operation), RequestID: requestID, Body: body}, nil
}

// checkResponseIdentity validates one inbound response frame
// against the request that caused it: one complete object within
// the frame bound, no unknown member, every required member
// present, the response schema identity exact, and the four
// identity fields echoed exactly. It returns the members for the
// branch checks. ok and the branch member are validated by the
// caller: a frame carrying neither branch is incomplete, and one
// carrying both claims two outcomes at once.
func checkResponseIdentity(frame []byte, want Request, context string) (map[string]json.RawMessage, error) {
	if len(frame) == 0 {
		failure, err := failViolation(context+" frame is empty", "")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if len(frame) > MaxFrameBytes {
		failure, err := failViolation(context+" frame exceeds the 8 MiB bound", "")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	members, fault := decodeStrictObject(frame)
	if fault != nil {
		failure, err := failViolation(context+" envelope "+fault.detail, fault.member)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, unknown := unknownMember(members, responseMembers); unknown {
		failure, err := failViolation(context+" envelope carries unknown member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, missing := missingMember(members, responseRequired); missing {
		failure, err := failViolation(context+" envelope misses a required member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != ResponseSchema {
		failure, err := failViolation(context+" schema is not the directory node response", "schema")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != ResponseSchemaVersion {
		failure, err := failViolation(context+" schema version is not the bound 1.0.0", "schema_version")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	version, _ := VersionForMajor(want.Major)
	if protocol, ok := rawString(members["protocol"]); !ok || protocol != ProtocolID {
		failure, err := failViolation(context+" protocol does not echo the request", "protocol")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if protocolVersion, ok := rawString(members["protocol_version"]); !ok || protocolVersion != version {
		failure, err := failViolation(context+" protocol version does not echo the request", "protocol_version")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if requestID, ok := rawString(members["request_id"]); !ok || requestID != want.RequestID {
		failure, err := failViolation(context+" request identifier does not echo the request", "request_id")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if operation, ok := rawString(members["operation"]); !ok || operation != string(want.Operation) {
		failure, err := failViolation(context+" operation does not echo the request", "operation")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	return members, nil
}

// CheckSuccessEnvelope validates one inbound success frame against
// the request that caused it: exact identity echo, ok=true, an
// object body, and no error member. Body and error are disjoint,
// never nullable sentinels: a frame carrying both or neither is a
// protocol violation. The returned body is the raw member for the
// operation-layer checks.
func CheckSuccessEnvelope(frame []byte, want Request) ([]byte, error) {
	members, err := checkResponseIdentity(frame, want, "success")
	if err != nil {
		return nil, err
	}
	positive, ok := rawBool(members["ok"])
	if !ok || !positive {
		failure, faultErr := failViolation("success envelope does not carry ok=true", "ok")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	if _, present := members["error"]; present {
		failure, faultErr := failViolation("success envelope carries both body and error", "error")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	raw, present := members["body"]
	if !present {
		failure, faultErr := failViolation("success envelope carries neither body nor error", "body")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	body := bytes.TrimSpace(raw)
	if _, fault := decodeStrictObject(body); fault != nil {
		failure, faultErr := failViolation("success body "+fault.detail, "body")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	return body, nil
}

// CheckFailureEnvelope validates one inbound failure frame against
// the request that caused it: exact identity echo, ok=false, a
// closed Structured Error 1.2.0 object in error, and no body
// member. The error object itself must decode under the bound
// version; a failure that cannot be read is an integrity failure,
// never a retryable result.
func CheckFailureEnvelope(frame []byte, want Request) (*axerror.Error, error) {
	members, err := checkResponseIdentity(frame, want, "failure")
	if err != nil {
		return nil, err
	}
	positive, ok := rawBool(members["ok"])
	if !ok || positive {
		failure, faultErr := failViolation("failure envelope does not carry ok=false", "ok")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	if _, present := members["body"]; present {
		failure, faultErr := failViolation("failure envelope carries both body and error", "body")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	raw, present := members["error"]
	if !present {
		failure, faultErr := failViolation("failure envelope carries neither body nor error", "error")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	decoded, decodeErr := axerror.Decode(ErrorVersion, bytes.TrimSpace(raw))
	if decodeErr != nil {
		failure, faultErr := failIntegrity("failure error is not a Structured Error 1.2.0", "error")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	return decoded, nil
}

// requestVersion reports the exact N.0.0 version a Request was
// issued under. It cannot fail: DecodeRequestFrame and
// EncodeRequest admit only the two closed bindings.
func requestVersion(want Request) string {
	version, _ := VersionForMajor(want.Major)
	return version
}
