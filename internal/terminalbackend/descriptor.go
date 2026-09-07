// Provider Protocol 3.0.0 Terminal Instance descriptor admission.
//
// Normative scope is relux-works/agent-session-manager-spec@v0.5.0 §7.A:
// Provider Protocol 3.0.0 retains the exact v2 framing and replaces every
// occurrence of the v2 TerminalDescriptor with one closed v3 object. This
// file owns the v3-specific production rule: parsing the closed descriptor
// document and matching it against the AX-validated host-local binding
// before a provider process is launched or observed. Transport framing,
// deadlines, and dispatch stay with the v2 machinery; the v3 failure-error
// binding (Structured Error 1.3.0) is resolved through axerror.BindingFor
// by the dual-stack host, which does not exist yet (see the stated bound
// in internal/provhost/doc.go).
//
// The provider-side MUST this file enforces is: reject a descriptor whose
// binding digest, backend ID/version, or generation does not match the
// AX-validated host-local binding. Shape validation (the closed member set
// and every per-member constraint) runs first so a malformed descriptor is
// refused as a protocol violation, never compared as a binding.
//
// Future caller, recorded so the next leaf inherits the question rather
// than rediscovering it: AdmitProviderDescriptor has no production caller
// yet. It belongs on the v3 launch/observe path — the dual-stack host
// follow-up named in internal/provhost/doc.go — which must admit the
// candidate TerminalDescriptor against the AX-validated host-local
// binding immediately before spawning the provider process (launch) and
// before trusting a status observation (observe), and refuse the
// operation on mismatch. Wiring it anywhere else (after spawn, on a
// retry path that reuses an admitted descriptor without re-checking
// generation) would admit a stale binding; the call order is parse,
// match, then launch, with no caching of the admitted value across
// generation changes.
package terminalbackend

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// descriptorMembers is the exact closed §7.A member set: eight table rows,
// nine members, because columns and rows are two members. No other member
// is permitted. It is package-level so the inventory derivation sees the
// single construction site; widening it must redden the member-set test,
// not pass silently.
var descriptorMembers = []string{
	"terminal_binding_id",
	"terminal_instance_id",
	"terminal_backend_id",
	"implementation_version",
	"protocol_version",
	"backend_generation",
	"interactive",
	"columns",
	"rows",
}

// ProviderDescriptor is one validated §7.A v3 TerminalDescriptor. Every
// value is trustworthy only when produced by ParseProviderDescriptor: the
// zero value is not valid (empty digest, empty generation, zero geometry),
// and no constructor outside this file may mint one. Columns and Rows
// carry the §7.A uint16[1..1000] bound; the Go type holds the uint16 half
// and the parse entry holds the 1..1000 half.
type ProviderDescriptor struct {
	TerminalBindingID     string
	TerminalInstanceID    string
	TerminalBackendID     string
	ImplementationVersion string
	ProtocolVersion       string
	BackendGeneration     string
	Interactive           bool
	Columns               uint16
	Rows                  uint16
}

// Binding projects the AX-validated binding subset §7.A matches: binding
// digest, backend ID, versions, and generation. Instance identity,
// interactivity, and geometry are carried context, matched to nothing.
func (descriptor ProviderDescriptor) Binding() InstanceBinding {
	return InstanceBinding{
		BackendID:             descriptor.TerminalBackendID,
		ImplementationVersion: descriptor.ImplementationVersion,
		ProtocolVersion:       descriptor.ProtocolVersion,
		Generation:            descriptor.BackendGeneration,
		TerminalBindingID:     descriptor.TerminalBindingID,
	}
}

// AdmitProviderDescriptor is the complete §7.A provider-side gate: parse
// the closed descriptor document, then match it against the AX-validated
// host-local binding. A malformed document is refused with
// terminal_backend_protocol_error before any comparison runs; a
// well-formed document that does not match is refused with the binding
// arm CheckProviderDescriptor reports. Admitted documents return the
// parsed descriptor for the launch path to carry.
func AdmitProviderDescriptor(doc []byte, binding InstanceBinding) (ProviderDescriptor, error) {
	descriptor, err := ParseProviderDescriptor(doc)
	if err != nil {
		return ProviderDescriptor{}, err
	}
	if err := CheckProviderDescriptor(descriptor.Binding(), binding); err != nil {
		return ProviderDescriptor{}, err
	}
	return descriptor, nil
}

// ParseProviderDescriptor validates one closed §7.A descriptor document.
// Member-set violations (an unknown member or a missing member) are one
// arm: the rule is the closed set itself, and the member-set test
// enumerates the extra, missing, and duplicate shapes against it. Member
// type violations (a non-string string member) are one arm for the same
// reason: JSON type discipline is one obligation, enumerated across all
// six string members by the member-type test.
func ParseProviderDescriptor(doc []byte) (ProviderDescriptor, error) {
	object, err := decodeStrictObject(doc)
	if err != nil {
		return ProviderDescriptor{}, err
	}
	if len(object) != len(descriptorMembers) {
		return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor member set"})
	}
	for _, member := range descriptorMembers {
		if _, known := object[member]; !known {
			return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor member set"})
		}
	}
	bindingID, err := descriptorText(object, "terminal_binding_id")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	if _, err := scalar.ParseDigest(bindingID); err != nil {
		return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor digest"})
	}
	instanceID, err := descriptorText(object, "terminal_instance_id")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	// The UUIDv7 grammar is the decidable half of the §7.A identity rule:
	// a PID, socket, path, pipe, URL, handle, token, or native reference
	// cannot spell a 36-character hyphenated UUIDv7, so grammar admission
	// is identity-shape admission. The undecidable half (a UUIDv7 that
	// names no live instance) is the binding's business, not the
	// document's.
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor instance"})
	}
	backendID, err := descriptorText(object, "terminal_backend_id")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	if _, err := ParseID(backendID); err != nil {
		return ProviderDescriptor{}, err
	}
	implementationVersion, err := descriptorText(object, "implementation_version")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	// Shape only: equality to the admitted Manifest and Probe is the
	// binding match CheckProviderDescriptor performs against versions the
	// registry validated at admission, so a well-formed version here is
	// compared, never trusted, there.
	if !semverPattern.MatchString(implementationVersion) {
		return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor implementation version"})
	}
	protocolVersion, err := descriptorText(object, "protocol_version")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	if !semverPattern.MatchString(protocolVersion) || semverMajor(protocolVersion) != 1 {
		return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor protocol version"})
	}
	generation, err := descriptorText(object, "backend_generation")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	// The string[1..256] bound is the registry's own rule, enforced by
	// its own arm; a descriptor generation outside it is the same
	// violation wherever it is read.
	if err := checkGeneration(generation); err != nil {
		return ProviderDescriptor{}, err
	}
	interactive, ok := object["interactive"].(bool)
	if !ok {
		return ProviderDescriptor{}, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor interactive"})
	}
	columns, err := descriptorGeometry(object, "columns")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	rows, err := descriptorGeometry(object, "rows")
	if err != nil {
		return ProviderDescriptor{}, err
	}
	return ProviderDescriptor{
		TerminalBindingID:     bindingID,
		TerminalInstanceID:    instanceID,
		TerminalBackendID:     backendID,
		ImplementationVersion: implementationVersion,
		ProtocolVersion:       protocolVersion,
		BackendGeneration:     generation,
		Interactive:           interactive,
		Columns:               columns,
		Rows:                  rows,
	}, nil
}

// descriptorText extracts a required string member. Numbers, booleans,
// nulls, arrays, and objects are refused: no string member of the §7.A
// descriptor takes them.
func descriptorText(object map[string]any, name string) (string, error) {
	value, ok := object[name].(string)
	if !ok {
		return "", refuse(&Error{Code: CodeProtocolError, Detail: "descriptor member type"})
	}
	return value, nil
}

// descriptorGeometry extracts a required uint16[1..1000] member. The
// strict decoder delivers numbers as json.Number, so the literal must
// spell an unsigned integer: a fraction, exponent, or sign is not one,
// even when it names the same mathematical value. Canonical geometry
// carries counts, never measurements.
func descriptorGeometry(object map[string]any, name string) (uint16, error) {
	// Three arms, one site each, shared by both geometry members: the JSON
	// number type, the unsigned-integer literal shape, and the 1..1000
	// bound are three distinct rules, and the geometry test enumerates the
	// columns shapes and the rows shapes against each of them.
	literal, ok := object[name].(json.Number)
	if !ok {
		return 0, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor geometry type"})
	}
	digits := literal.String()
	value := 0
	for i := 0; i < len(digits); i++ {
		digit := digits[i]
		if digit < '0' || digit > '9' {
			return 0, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor geometry digits"})
		}
		// Refuse on the first digit that would exceed the bound rather
		// than accumulating past it: value is at most 100 here, so the
		// multiply below cannot wrap on any platform, and a literal
		// naming more than 1000 is refused before its low bits can
		// alias an in-range number downstream would trust as read.
		if value > 100 {
			return 0, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor geometry bound"})
		}
		value = value*10 + int(digit-'0')
	}
	// An empty literal spells no digits, so value stays 0 and lands on
	// the bound arm rather than needing an arm of its own.
	if value < 1 || value > 1000 {
		return 0, refuse(&Error{Code: CodeProtocolError, Detail: "descriptor geometry bound"})
	}
	return uint16(value), nil
}
