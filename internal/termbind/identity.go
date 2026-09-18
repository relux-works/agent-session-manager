package termbind

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/gowebpki/jcs"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// BindingSchema is the Terminal Instance Binding 1.0.0 schema URN from
// Section 4.B.
const BindingSchema = "urn:ax:schema:terminal-instance-binding"

// BindingSchemaVersion is the only admitted binding schema version.
const BindingSchemaVersion = "1.0.0"

// CheckInstanceIdentity admits exactly the AX-allocated terminal instance
// identity: a UUIDv7. A PID, handle, socket, path, named pipe, URL, token,
// mutable endpoint, or native reference cannot spell a 36-character
// hyphenated UUIDv7, so grammar admission is identity-shape admission (the
// landed descriptor rationale, applied here as the story-level gate). Any
// other value refuses terminal_backend_protocol_error: AX-local parsing of
// a syntactically invalid identity, per the Section 4.C row note.
func CheckInstanceIdentity(value string) error {
	if _, err := scalar.ParseUUIDv7(value); err != nil {
		return &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: "terminal instance identity"}
	}
	return nil
}

// Binding is the parsed Terminal Instance Binding 1.0.0 closed object: the
// host-local durable metadata Section 4.B defines. It is never replicated;
// replicated Session Events carry only its digest as an opaque audit
// reference.
type Binding struct {
	BindingID             string
	SessionID             string
	HostID                string
	HostIncarnationID     string
	TerminalInstanceID    string
	TerminalBackendID     string
	ImplementationVersion string
	ProtocolVersion       string
	BackendGeneration     string
	NativeReference       string
	CreatedAt             string
	SupersedesBindingID   string
	HasSupersedes         bool
}

// bindingMembers is the exact closed member set of the binding document,
// in the order Section 4.B lists it.
var bindingMembers = []string{
	"schema",
	"schema_version",
	"binding_id",
	"session_id",
	"host_id",
	"host_incarnation_id",
	"terminal_instance_id",
	"terminal_backend_id",
	"implementation_version",
	"protocol_version",
	"backend_generation",
	"native_reference",
	"created_at",
	"supersedes_binding_id",
	"extensions",
}

// ParseTerminalBinding admits one closed Terminal Instance Binding 1.0.0
// document. No parser exists on trunk for this schema, so this entry owns
// it; every grammar delegates to its landed owner (frame to environ,
// UUIDv7/digest/timestamp to scalar, backend ID and generation to
// terminalbackend, semver to environ). The protocol_version member must
// additionally select Terminal Backend Protocol major 1 and the
// extensions member must be exactly the empty object {}, per the Section
// 4.B table. The terminal_instance_id member passes through the coded
// identity gate: a forbidden identity form refuses
// terminal_backend_protocol_error. The binding_id member is recomputed by
// the Section 4.B identity rule (digest of the RFC 8785 JCS bytes with the
// ID member omitted, numbers and nested duplicates refused before the
// transform) and must equal the carried value; a reader recompute
// mismatch refuses terminal_backend_manifest_probe_mismatch. Member-type
// faults refuse the same protocol class as the landed descriptor parser:
// a mistyped binding member is a malformed local document.
func ParseTerminalBinding(raw []byte) (Binding, error) {
	empty := Binding{}
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return empty, protocolRefusal("binding frame")
	}
	if len(members) != len(bindingMembers) {
		return empty, protocolRefusal("binding member set")
	}
	for _, name := range bindingMembers {
		if _, known := members[name]; !known {
			return empty, protocolRefusal("binding member set")
		}
	}
	var parsed Binding
	text := func(name string) (string, error) {
		var value string
		decoder := json.NewDecoder(bytes.NewReader(members[name]))
		if err := decoder.Decode(&value); err != nil {
			return "", protocolRefusal("binding " + name)
		}
		if decoder.More() {
			return "", protocolRefusal("binding " + name)
		}
		return value, nil
	}
	schema, err := text("schema")
	if err != nil {
		return empty, err
	}
	if schema != BindingSchema {
		return empty, protocolRefusal("binding schema")
	}
	version, err := text("schema_version")
	if err != nil {
		return empty, err
	}
	if version != BindingSchemaVersion {
		return empty, protocolRefusal("binding schema version")
	}
	bindingID, err := text("binding_id")
	if err != nil {
		return empty, err
	}
	if _, err := scalar.ParseDigest(bindingID); err != nil {
		return empty, protocolRefusal("binding digest")
	}
	parsed.BindingID = bindingID
	for _, name := range []string{"session_id", "host_id", "host_incarnation_id"} {
		value, err := text(name)
		if err != nil {
			return empty, err
		}
		if _, err := scalar.ParseUUIDv7(value); err != nil {
			return empty, protocolRefusal("binding " + name)
		}
		switch name {
		case "session_id":
			parsed.SessionID = value
		case "host_id":
			parsed.HostID = value
		case "host_incarnation_id":
			parsed.HostIncarnationID = value
		}
	}
	instance, err := text("terminal_instance_id")
	if err != nil {
		return empty, err
	}
	if err := CheckInstanceIdentity(instance); err != nil {
		return empty, err
	}
	parsed.TerminalInstanceID = instance
	backendID, err := text("terminal_backend_id")
	if err != nil {
		return empty, err
	}
	if _, err := terminalbackend.ParseID(backendID); err != nil {
		return empty, err
	}
	parsed.TerminalBackendID = backendID
	implementation, err := text("implementation_version")
	if err != nil {
		return empty, err
	}
	if !environ.CheckSemver(implementation) {
		return empty, protocolRefusal("binding implementation version")
	}
	parsed.ImplementationVersion = implementation
	protocol, err := text("protocol_version")
	if err != nil {
		return empty, err
	}
	if !environ.CheckSemver(protocol) || !isProtocolMajorOne(protocol) {
		return empty, protocolRefusal("binding protocol version")
	}
	parsed.ProtocolVersion = protocol
	generation, err := text("backend_generation")
	if err != nil {
		return empty, err
	}
	if _, err := terminalbackend.GenerationDigest(generation); err != nil {
		return empty, err
	}
	parsed.BackendGeneration = generation
	native, err := text("native_reference")
	if err != nil {
		return empty, err
	}
	if _, ok := environ.CheckStringBounds(members["native_reference"], 1, 512); !ok {
		return empty, protocolRefusal("binding native reference")
	}
	parsed.NativeReference = native
	created, err := text("created_at")
	if err != nil {
		return empty, err
	}
	if _, err := scalar.ParseTimestamp(created); err != nil {
		return empty, protocolRefusal("binding timestamp")
	}
	parsed.CreatedAt = created
	if !isExplicitNull(members["supersedes_binding_id"]) {
		supersedes, err := text("supersedes_binding_id")
		if err != nil {
			return empty, err
		}
		if _, err := scalar.ParseDigest(supersedes); err != nil {
			return empty, protocolRefusal("binding supersedes")
		}
		parsed.SupersedesBindingID = supersedes
		parsed.HasSupersedes = true
	}
	if !isEmptyExtensionsObject(members["extensions"]) {
		return empty, protocolRefusal("binding extensions")
	}
	recomputed, err := bindingIdentity(raw, "binding_id")
	if err != nil {
		return empty, err
	}
	if recomputed != bindingID {
		return empty, &terminalbackend.Error{Code: terminalbackend.CodeMismatch, Detail: "binding identity"}
	}
	return parsed, nil
}

// protocolRefusal builds the AX-local document refusal: the Section 4.C
// protocol class for a malformed local parse.
func protocolRefusal(detail string) error {
	return &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: detail}
}

// isExplicitNull reports whether raw is the explicit JSON null.
func isExplicitNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// bindingIdentity computes the Section 4.B identity of one binding
// document: the lowercase sha256 digest of the RFC 8785 JCS bytes with the
// self member omitted. canonicaljson explicitly rejects this schema and
// the terminalbackend identity rule is unexported, so this is the
// schema's own instance of the rule, built on the same JCS library both
// owners use. The decode refuses a JSON number at any depth and a
// duplicate member at any depth BEFORE the JCS transform (SPEC.md:301
// NUM-UNSAFE-NUMBER): the transform rounds numbers and the plain decode
// keeps the last duplicate, so without the pre-transform walk a
// reordered caller would digest rounded or substituted bytes.
// TestBindingIdentityAgreesWithLandedAdmission pins digest equality with
// the landed manifest admission on a manifest-shaped object and verdict
// agreement on a numeric extension value and a nested duplicate member.
func bindingIdentity(raw []byte, selfField string) (string, error) {
	object, err := decodeIdentityDocument(raw)
	if err != nil {
		return "", err
	}
	if err := refuseIdentityNumbers(object); err != nil {
		return "", err
	}
	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != selfField {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		return "", protocolRefusal("binding identity")
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		return "", protocolRefusal("binding identity")
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Identity-decode bounds, mirroring the landed terminalbackend identity
// decode (maxJSONDepth, maxIdentityDocumentBytes): the depth bound sits
// far above the deepest legitimate binding nesting, the size bound far
// above the deepest legitimate binding document, and both fail closed.
const (
	maxBindingIdentityDepth = 32
	maxBindingIdentityBytes = 5_242_880
)

// decodeIdentityDocument decodes one identity document the way the landed
// terminalbackend identity decode does: invalid UTF-8, a lone surrogate
// escape, a duplicate member at ANY depth, excessive nesting, an
// oversize document, a non-object top level, and trailing data each
// refuse before any identity bytes exist. Numbers decode as json.Number
// so the refuseIdentityNumbers walk can refuse them structurally.
func decodeIdentityDocument(raw []byte) (map[string]any, error) {
	if len(raw) > maxBindingIdentityBytes {
		return nil, protocolRefusal("binding identity")
	}
	if !utf8.Valid(raw) {
		return nil, protocolRefusal("binding identity")
	}
	if environ.HasLoneSurrogateEscape(raw) {
		return nil, protocolRefusal("binding identity")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeIdentityValue(decoder, 0)
	if err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, protocolRefusal("binding identity")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, protocolRefusal("binding identity")
	}
	return object, nil
}

// decodeIdentityValue decodes one JSON value with an explicit depth
// bound and duplicate-member refusal at every object depth, mirroring
// the landed terminalbackend capped decode.
func decodeIdentityValue(decoder *json.Decoder, depth int) (any, error) {
	if depth > maxBindingIdentityDepth {
		return nil, protocolRefusal("binding identity")
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, protocolRefusal("binding identity")
	}
	if delimiter, ok := token.(json.Delim); ok {
		switch delimiter {
		case '{':
			object := make(map[string]any)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, protocolRefusal("binding identity")
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, protocolRefusal("binding identity")
				}
				if _, duplicate := object[key]; duplicate {
					return nil, protocolRefusal("binding identity")
				}
				member, err := decodeIdentityValue(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				object[key] = member
			}
			if _, err := decoder.Token(); err != nil {
				return nil, protocolRefusal("binding identity")
			}
			return object, nil
		case '[':
			array := []any{}
			for decoder.More() {
				member, err := decodeIdentityValue(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				array = append(array, member)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, protocolRefusal("binding identity")
			}
			return array, nil
		default:
			return nil, protocolRefusal("binding identity")
		}
	}
	return token, nil
}

// refuseIdentityNumbers walks a decoded identity document and refuses any
// JSON number at any depth, mirroring the landed terminalbackend walk. It
// runs inside bindingIdentity before the JCS transform, so no number
// reaches the digest bytes however the members were checked.
func refuseIdentityNumbers(value any) error {
	switch typed := value.(type) {
	case json.Number, float64:
		return protocolRefusal("binding identity")
	case map[string]any:
		for _, member := range typed {
			if err := refuseIdentityNumbers(member); err != nil {
				return err
			}
		}
	case []any:
		for _, member := range typed {
			if err := refuseIdentityNumbers(member); err != nil {
				return err
			}
		}
	}
	return nil
}

// isProtocolMajorOne reports whether a semver string selects Terminal
// Backend Protocol major 1. The grammar gate (environ.CheckSemver) runs
// first at the call site, so the major is exactly the digit run before
// the first dot and reads 1 if and only if the string starts with "1.":
// no integer parse, no saturation class for a huge foreign major.
// TestBindingProtocolMajorAgreesWithLandedTuple pins verdict agreement
// with terminalbackend.CheckVersionTuple over a major corpus.
func isProtocolMajorOne(version string) bool {
	return strings.HasPrefix(version, "1.")
}

// isEmptyExtensionsObject admits exactly the empty object {} the Section
// 4.B Binding table pins for extensions. The member decodes through the
// landed strict frame (duplicate members, lone surrogates, and trailing
// data refuse) and must carry zero members; any other value — a string,
// number, array, null, or a non-empty object — refuses.
func isEmptyExtensionsObject(raw json.RawMessage) bool {
	decoded, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return false
	}
	return len(decoded) == 0
}

// AdmitMutationContext gates the terminal_instance_id member of one
// Section 4.C mutation context document through the coded identity gate,
// then admits the document through the landed terminstance parser. The
// pre-scan only fires on a present string member; any other shape
// delegates untouched, so the landed parser remains the shape authority.
// Documented redundancy: the landed parser refuses a forbidden identity
// with the same pinned code, so the pre-scan adds only the Go error type
// (*terminalbackend.Error); its narrowing row kills on that type.
func AdmitMutationContext(raw []byte) (terminstance.MutationContext, error) {
	if err := prescanIdentity(raw, "terminal_instance_id"); err != nil {
		return terminstance.MutationContext{}, err
	}
	return terminstance.ParseMutationContext(raw)
}

// AdmitStatusBody gates the terminal_instance_id member of one Section 4.C
// status body through the coded identity gate when the member is a present
// non-null string, then admits the body through the landed terminstance
// parser. A null instance (the session-scoped lookup) skips the gate; any
// non-string shape delegates untouched to the landed shape authority.
// Documented redundancy, as AdmitMutationContext: the landed parser
// refuses the same pinned code, so the pre-scan adds only the Go error
// type (*terminalbackend.Error); its narrowing row kills on that type.
func AdmitStatusBody(raw []byte) (terminstance.StatusBody, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return terminstance.ParseStatusBody(raw)
	}
	encoded, known := members["terminal_instance_id"]
	if !known || isExplicitNull(encoded) {
		return terminstance.ParseStatusBody(raw)
	}
	var value string
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if err := decoder.Decode(&value); err != nil {
		return terminstance.ParseStatusBody(raw)
	}
	if err := CheckInstanceIdentity(value); err != nil {
		return terminstance.StatusBody{}, err
	}
	return terminstance.ParseStatusBody(raw)
}

// prescanIdentity runs the identity gate over one present string member of
// a raw document. A frame fault, a missing member, or a non-string member
// returns nil so the caller delegates to the landed shape authority,
// which refuses those shapes itself.
func prescanIdentity(raw []byte, member string) error {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return nil
	}
	encoded, known := members[member]
	if !known {
		return nil
	}
	var value string
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	return CheckInstanceIdentity(value)
}
