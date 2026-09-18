package axpane

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// DescriptorParams carries the §7.A Terminal Instance binding context
// the wrapper supplies: the validated host-local binding subset,
// interactivity, and geometry.
type DescriptorParams struct {
	// Binding is the AX-validated host-local binding subset the
	// descriptor must match.
	Binding BindingRef
	// InstanceID is the AX-allocated terminal instance UUIDv7;
	// empty mints a fresh one. It is never a PID, socket, path,
	// pipe, URL, handle, token, or native reference.
	InstanceID string
	// Interactive selects the presentation mode.
	Interactive bool
	// Columns and Rows carry the uint16[1..1000] geometry bound.
	Columns uint16
	Rows    uint16
}

// BindingRef is the validated host-local binding subset: binding
// digest, backend ID, versions, and generation.
type BindingRef struct {
	BindingDigest         string
	BackendID             string
	ImplementationVersion string
	ProtocolVersion       string
	Generation            string
}

// MintInstanceID allocates one terminal instance UUIDv7: 48-bit
// millisecond timestamp, version and variant bits, and 74
// cryptographic random bits. Time and randomness come from the
// caller-supplied clock and reader so tests pin them; production
// passes time.Now and crypto/rand.
func MintInstanceID(now time.Time, random func([]byte) (int, error)) (string, error) {
	var raw [16]byte
	millis := uint64(now.UTC().UnixMilli())
	if millis>>48 != 0 {
		return "", fmt.Errorf("instance clock is outside the UUIDv7 range")
	}
	raw[0] = byte(millis >> 40)
	raw[1] = byte(millis >> 32)
	raw[2] = byte(millis >> 24)
	raw[3] = byte(millis >> 16)
	raw[4] = byte(millis >> 8)
	raw[5] = byte(millis)
	tail := make([]byte, 10)
	if _, err := random(tail); err != nil {
		return "", fmt.Errorf("instance randomness failed: %w", err)
	}
	copy(raw[6:], tail)
	raw[6] = raw[6]&0x0f | 0x70
	raw[8] = raw[8]&0x3f | 0x80
	value := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(raw[0:4]),
		binary.BigEndian.Uint16(raw[4:6]),
		binary.BigEndian.Uint16(raw[6:8]),
		binary.BigEndian.Uint16(raw[8:10]),
		uint64(raw[10])<<40|uint64(raw[11])<<32|uint64(raw[12])<<24|uint64(raw[13])<<16|uint64(raw[14])<<8|uint64(raw[15]))
	if _, err := scalar.ParseUUIDv7(value); err != nil {
		return "", fmt.Errorf("minted instance id is not a UUIDv7: %w", err)
	}
	return value, nil
}

// MintInstanceIDRandom mints one terminal instance UUIDv7 from the
// system clock and crypto/rand.
func MintInstanceIDRandom() (string, error) {
	return MintInstanceID(time.Now(), rand.Read)
}

// BuildDescriptor renders the closed §7.A descriptor document for the
// launch path: the binding digest, the AX-allocated instance UUIDv7,
// backend identity, generation, interactivity, and geometry. No
// other member is permitted. Geometry outside uint16[1..1000]
// refuses, and the instance ID must parse as UUIDv7.
func BuildDescriptor(params DescriptorParams) ([]byte, error) {
	instance := params.InstanceID
	if instance == "" {
		minted, err := MintInstanceIDRandom()
		if err != nil {
			return nil, err
		}
		instance = minted
	}
	if _, err := scalar.ParseUUIDv7(instance); err != nil {
		return nil, fmt.Errorf("descriptor instance %q is not a UUIDv7: %w", instance, err)
	}
	if params.Columns < 1 || params.Columns > 1000 || params.Rows < 1 || params.Rows > 1000 {
		return nil, fmt.Errorf("descriptor geometry %dx%d is outside uint16[1..1000]", params.Columns, params.Rows)
	}
	document := map[string]any{
		"terminal_binding_id":    params.Binding.BindingDigest,
		"terminal_instance_id":   instance,
		"terminal_backend_id":    params.Binding.BackendID,
		"implementation_version": params.Binding.ImplementationVersion,
		"protocol_version":       params.Binding.ProtocolVersion,
		"backend_generation":     params.Binding.Generation,
		"interactive":            params.Interactive,
		"columns":                params.Columns,
		"rows":                   params.Rows,
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode provider descriptor: %w", err)
	}
	return raw, nil
}
