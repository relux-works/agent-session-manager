// Package rpcwire encodes and inspects untrusted Mesh RPC frames. Structural
// validity and correlation are never authentication, handshake completion or
// permission to dispatch. No connection, responder or operation executor lives
// here. The eventual bilateral admission consumer must supply those boundaries.
package rpcwire

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sshtransport"
)

const Protocol = "urn:ax:protocol:rpc"
const MaxLineBytes = sshtransport.MaxLineBytes
const MinObjectBytes = 5 * 1024 * 1024

var (
	ErrFrame       = errors.New("invalid RPC frame")
	ErrVersion     = errors.New("unsupported RPC wire version")
	ErrCorrelation = errors.New("RPC response does not match request")
	ErrHello       = errors.New("incompatible RPC hello structure")
	ErrInventory   = errors.New("invalid RPC inventory structure")
)

// Request retains an isolated envelope. Body bytes for operations other than
// hello and inventory.roots are opaque: their operation owners must validate
// them. A zero Request cannot be encoded or used as a response expectation.
type Request struct {
	version, id, operation string
	body                   json.RawMessage
}

func (r Request) Version() string       { return r.version }
func (r Request) ID() string            { return r.id }
func (r Request) Operation() string     { return r.operation }
func (r Request) Body() json.RawMessage { return bytes.Clone(r.body) }

// Response is correlated but still untrusted. A success body has the same
// operation-validation bounds as Request. A failure uses the containing major's
// static Structured Error binding, never a version supplied by its body.
type Response struct {
	request Request
	body    json.RawMessage
	failure *axerror.Error
}

func (r Response) OK() bool                { return r.failure == nil && r.body != nil }
func (r Response) Body() json.RawMessage   { return bytes.Clone(r.body) }
func (r Response) Failure() *axerror.Error { return r.failure }

func major(version string) (int, error) {
	switch version {
	case "2.0.0":
		return 2, nil
	case "3.0.0":
		return 3, nil
	case "4.0.0":
		return 4, nil
	case "5.0.0":
		return 5, nil
	default:
		return 0, ErrVersion
	}
}

// object refuses duplicate members, non-integral numbers and other
// non-I-JSON input using the existing common data-model owner. Keep original
// bytes; canonicalization must not launder invalid numeric lexical forms.
func object(data []byte) (map[string]json.RawMessage, error) {
	if _, err := canonicaljson.Canonicalize(data); err != nil {
		return nil, ErrFrame
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) != nil || m == nil {
		return nil, ErrFrame
	}
	return m, nil
}
func exact(m map[string]json.RawMessage, keys ...string) bool {
	if len(m) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}
func stringValue(raw json.RawMessage) string {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}
func envelope(data []byte) (map[string]json.RawMessage, string, string, error) {
	if len(data) == 0 || len(data) > MaxLineBytes || bytes.ContainsAny(data, "\n") {
		return nil, "", "", ErrFrame
	}
	m, err := object(data)
	if err != nil {
		return nil, "", "", err
	}
	if stringValue(m["protocol"]) != Protocol {
		return nil, "", "", ErrFrame
	}
	version := stringValue(m["protocol_version"])
	if !semver.MatchString(version) {
		return nil, "", "", ErrFrame
	}
	if _, err = major(version); err != nil {
		return nil, "", "", err
	}
	id := stringValue(m["request_id"])
	if _, err = scalar.ParseUUIDv7(id); err != nil {
		return nil, "", "", ErrFrame
	}
	return m, version, id, nil
}

// DecodeRequest inspects one LF-free line, as returned by SSH Session.Receive.
// It never marks a hello successful or opens an operation-dispatch path.
func DecodeRequest(line []byte) (Request, error) {
	m, version, id, err := envelope(line)
	if err != nil {
		return Request{}, err
	}
	if !exact(m, "protocol", "protocol_version", "request_id", "operation", "body") {
		return Request{}, ErrFrame
	}
	op := stringValue(m["operation"])
	if op == "" {
		return Request{}, ErrFrame
	}
	if _, err = object(m["body"]); err != nil {
		return Request{}, err
	}
	if op == "hello" {
		_, err = decodeHello(version, m["body"], false)
	}
	if op == "inventory.roots" {
		_, err = decodeNamespaces(version, m["body"])
	}
	if err != nil {
		return Request{}, err
	}
	return Request{version, id, op, bytes.Clone(m["body"])}, nil
}

// EncodeRequest validates the same structural surface as DecodeRequest and
// returns one line without LF for SSH Session.Send. It does not send it.
func EncodeRequest(version, id, operation string, body json.RawMessage) ([]byte, error) {
	line, err := json.Marshal(map[string]any{"protocol": Protocol, "protocol_version": version, "request_id": id, "operation": operation, "body": body})
	if err != nil {
		return nil, ErrFrame
	}
	if _, err = DecodeRequest(line); err != nil {
		return nil, err
	}
	return line, nil
}

// DecodeResponse compares identity/version before examining the success or
// error payload. It does not consume a pending request or provide replay
// protection; connection sequencing remains the admission consumer's job.
func DecodeResponse(line []byte, request Request) (Response, error) {
	if request.id == "" {
		return Response{}, ErrCorrelation
	}
	m, version, id, err := envelope(line)
	if err != nil {
		return Response{}, err
	}
	if version != request.version {
		return Response{}, ErrVersion
	}
	if id != request.id {
		return Response{}, ErrCorrelation
	}
	switch string(m["ok"]) {
	case "true":
		if !exact(m, "protocol", "protocol_version", "request_id", "ok", "body") {
			return Response{}, ErrFrame
		}
		if _, err = object(m["body"]); err != nil {
			return Response{}, err
		}
		if request.operation == "hello" {
			var h Hello
			h, err = decodeHello(version, m["body"], true)
			if err == nil {
				sent, _ := request.Hello()
				if h.NonceEcho != sent.Nonce {
					err = ErrCorrelation
				}
			}
		}
		if request.operation == "inventory.roots" {
			err = validateRoots(version, m["body"], request.body)
		}
		if err != nil {
			return Response{}, err
		}
		return Response{request: request, body: bytes.Clone(m["body"])}, nil
	case "false":
		if !exact(m, "protocol", "protocol_version", "request_id", "ok", "error") {
			return Response{}, ErrFrame
		}
		n, _ := major(version)
		failure, err := axerror.DecodeBound(axerror.ContainingContract{ID: Protocol, Major: n}, m["error"])
		if err != nil {
			return Response{}, err
		}
		return Response{request: request, failure: failure}, nil
	default:
		return Response{}, ErrFrame
	}
}

func EncodeSuccess(request Request, body json.RawMessage) ([]byte, error) {
	return encodeResponse(request, true, "body", body)
}
func EncodeFailure(request Request, failure *axerror.Error) ([]byte, error) {
	return encodeResponse(request, false, "error", failure)
}
func encodeResponse(request Request, ok bool, key string, payload any) ([]byte, error) {
	line, err := json.Marshal(map[string]any{"protocol": Protocol, "protocol_version": request.version, "request_id": request.id, "ok": ok, key: payload})
	if err != nil {
		return nil, ErrFrame
	}
	if _, err = DecodeResponse(line, request); err != nil {
		return nil, err
	}
	return line, nil
}

// EncodeRejection frames a failure for a parseable supported-version request,
// even when its operation or body is invalid (Section 15.1 bootstrap framing).
// Missing/invalid protocol, version or request ID and unparseable lines return
// no frame. This only encodes bytes: the eventual responder must choose the
// correct failure and close after its one pre-handshake rejection. It never
// returns a Request that could be mistaken for a validated hello.
func EncodeRejection(line []byte, failure *axerror.Error) ([]byte, error) {
	m, version, id, err := envelope(line)
	if err != nil {
		return nil, err
	}
	// A response or arbitrary object is not a request to which we may reply.
	if _, ok := m["operation"]; !ok {
		return nil, ErrFrame
	}
	return EncodeFailure(Request{version: version, id: id}, failure)
}
