package provhost

import (
	"encoding/json"
)

// This file validates the Section 7.4 Capability result: the closed
// urn:ax:schema:provider-probe 1.0.0 object a probe operation returns
// as its success body. It also owns the host-side availability gate:
// only available MAY be enabled, and the host MUST NOT treat a
// conditional, unsupported, or unknown capability as usable, no matter
// what enabled claims.

// probeSchema is the exact schema identifier the probe carries.
const probeSchema = "urn:ax:schema:provider-probe"

// probeSchemaVersion is the only probe version this host accepts.
const probeSchemaVersion = "1.0.0"

// Probe capability statuses Section 7.4 defines. Only Available MAY
// set enabled true; every other status MUST NOT be advertised as
// usable, which CapabilityUsable enforces.
const (
	CapabilityAvailable   = "available"
	CapabilityConditional = "conditional"
	CapabilityUnsupported = "unsupported"
	CapabilityUnknown     = "unknown"
)

// probeCapabilityStatuses is the closed status vocabulary in the
// Section 7.4 order.
var probeCapabilityStatuses = []string{
	CapabilityAvailable,
	CapabilityConditional,
	CapabilityUnsupported,
	CapabilityUnknown,
}

// Probe capability evidence Section 7.4 defines, in section order.
var probeCapabilityEvidence = []string{
	"documented",
	"probed",
	"accepted_test",
	"provider_contract",
	"inferred",
	"acceptance_required",
	"none",
}

// probeArchitectures is the closed architecture vocabulary.
var probeArchitectures = []string{"amd64", "arm64"}

// probePlatforms is the closed platform vocabulary: the four Section
// 7.3 enums. WSL2 and native Windows are distinct rows; one never
// stands for the other.
var probePlatforms = []string{"linux", "macos", "windows", "wsl2"}

// probeMembers is the exact required member set DecodeProbe accepts.
var probeMembers = map[string]bool{
	"schema":           true,
	"schema_version":   true,
	"provider_id":      true,
	"provider_version": true,
	"platform":         true,
	"architecture":     true,
	"capabilities":     true,
	"warnings":         true,
}

// probeRequired lists probeMembers in a fixed order so a body missing
// several members always names the same one.
var probeRequired = []string{
	"schema",
	"schema_version",
	"provider_id",
	"provider_version",
	"platform",
	"architecture",
	"capabilities",
	"warnings",
}

// probeCapabilityMembers is the exact member set of one capability
// value: status, enabled, evidence, and detail, nothing else.
var probeCapabilityMembers = map[string]bool{
	"status":   true,
	"enabled":  true,
	"evidence": true,
	"detail":   true,
}

// probeCapabilityRequired lists probeCapabilityMembers in a fixed
// order so a value missing several members always names the same one.
var probeCapabilityRequired = []string{"status", "enabled", "evidence", "detail"}

// DecodeProbe validates one probe-operation success body as the closed
// Section 7.4 probe object. It returns nil on a well-formed probe and
// a provider_protocol_error naming the offending member otherwise. A
// capability that claims enabled under any status but available is
// refused here, not at use time: an advertised-but-unusable surface
// must never reach the caller as a probe result.
func DecodeProbe(body []byte) error {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if name, unknown := unknownMember(members, probeMembers); unknown {
		failure, err := failProtocol("probe carries unknown member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if name, missing := missingMember(members, probeRequired); missing {
		failure, err := failProtocol("probe misses a required member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != probeSchema {
		failure, err := failProtocol("probe schema is not the provider probe", "schema")
		if err != nil {
			return err
		}
		return failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != probeSchemaVersion {
		failure, err := failProtocol("probe schema_version is not 1.0.0", "schema_version")
		if err != nil {
			return err
		}
		return failure
	}
	if provider, ok := rawString(members["provider_id"]); !ok || !validProviderID(provider) {
		failure, err := failProtocol("probe provider_id is not a provider id", "provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if version, ok := rawString(members["provider_version"]); !ok || runeLength(version) < 1 || runeLength(version) > 128 {
		failure, err := failProtocol("probe provider_version is not 1..128 characters", "provider_version")
		if err != nil {
			return err
		}
		return failure
	}
	if platform, ok := rawString(members["platform"]); !ok || !isProbePlatform(platform) {
		failure, err := failProtocol("probe platform is not a registry member", "platform")
		if err != nil {
			return err
		}
		return failure
	}
	if architecture, ok := rawString(members["architecture"]); !ok || !isProbeArchitecture(architecture) {
		failure, err := failProtocol("probe architecture is not amd64 or arm64", "architecture")
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkProbeCapabilities(members["capabilities"]); err != nil {
		return err
	}
	if err := checkProbeWarnings(members["warnings"]); err != nil {
		return err
	}
	return nil
}

func isProbePlatform(value string) bool {
	for _, platform := range probePlatforms {
		if value == platform {
			return true
		}
	}
	return false
}

func isProbeArchitecture(value string) bool {
	for _, architecture := range probeArchitectures {
		if value == architecture {
			return true
		}
	}
	return false
}

// checkProbeCapabilities requires exactly the seven registry keys,
// each carrying a closed status/enabled/evidence/detail value. The
// per-key loop is derived from capabilityOrder, never from a sample:
// a validation that held for three capabilities and skipped the rest
// would pass a probe this function must refuse.
func checkProbeCapabilities(raw json.RawMessage) error {
	decoded, fault := decodeStrictObject(raw)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	for _, name := range capabilityOrder {
		if _, present := decoded[name]; !present {
			failure, err := failProtocol("probe capabilities miss a registry key", "capabilities")
			if err != nil {
				return err
			}
			return failure
		}
	}
	for name := range decoded {
		known := false
		for _, registry := range capabilityOrder {
			if name == registry {
				known = true
			}
		}
		if !known {
			failure, err := failProtocol("probe capabilities carry an unknown key", "capabilities")
			if err != nil {
				return err
			}
			return failure
		}
	}
	for _, name := range capabilityOrder {
		if err := checkProbeCapabilityValue(decoded[name]); err != nil {
			return err
		}
	}
	return nil
}

// checkProbeCapabilityValue validates one capability value object. The
// member parameter of every refusal below is "capabilities": the
// detail names the violated rule, and the per-key sweep in the
// conformance tests proves each registry key answers to it.
func checkProbeCapabilityValue(raw json.RawMessage) error {
	value, fault := decodeStrictObject(raw)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if _, unknown := unknownMember(value, probeCapabilityMembers); unknown {
		failure, err := failProtocol("probe capability carries an unknown member", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	if _, missing := missingMember(value, probeCapabilityRequired); missing {
		failure, err := failProtocol("probe capability misses a required member", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	status, ok := rawString(value["status"])
	if !ok || !isProbeStatus(status) {
		failure, err := failProtocol("probe capability status is not a registry member", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	enabled, ok := rawBool(value["enabled"])
	if !ok {
		failure, err := failProtocol("probe capability enabled is not a boolean", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	if evidence, ok := rawString(value["evidence"]); !ok || !isProbeEvidence(evidence) {
		failure, err := failProtocol("probe capability evidence is not a registry member", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	if detail, ok := rawString(value["detail"]); !ok || runeLength(detail) > 2048 {
		failure, err := failProtocol("probe capability detail is not 0..2048 characters", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	if enabled && status != CapabilityAvailable {
		failure, err := failProtocol("probe capability enables a non-available status", "capabilities")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

func isProbeStatus(value string) bool {
	for _, status := range probeCapabilityStatuses {
		if value == status {
			return true
		}
	}
	return false
}

func isProbeEvidence(value string) bool {
	for _, evidence := range probeCapabilityEvidence {
		if value == evidence {
			return true
		}
	}
	return false
}

// checkProbeWarnings enforces the sorted, unique, bounded warning
// array: at most 1,024 strings of at most 2,048 characters each.
func checkProbeWarnings(raw json.RawMessage) error {
	warnings, ok := rawStringArray(raw)
	if !ok || len(warnings) > 1024 {
		failure, err := failProtocol("probe warnings exceed 1024 entries", "warnings")
		if err != nil {
			return err
		}
		return failure
	}
	for _, warning := range warnings {
		if runeLength(warning) > 2048 {
			failure, err := failProtocol("probe warning exceeds 2048 characters", "warnings")
			if err != nil {
				return err
			}
			return failure
		}
	}
	if !sortedUniqueStrings(warnings) {
		failure, err := failProtocol("probe warnings are not sorted unique", "warnings")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// CapabilityUsable reports whether the probe establishes the named
// capability as usable: status available and enabled true. It is the
// host-side reading of the Section 8 matrices: conditional,
// unsupported, and unknown are never usable, however enabled reads,
// and unknown MUST NOT be rewritten as unsupported — both refuse the
// same way here, by returning false.
func CapabilityUsable(status string, enabled bool) bool {
	return status == CapabilityAvailable && enabled
}

// RequireCapability decodes one probe body and requires the named
// capability usable. A malformed probe is a provider_protocol_error;
// a well-formed probe that does not establish the capability is an
// invalid_config caller error: the caller asked for a surface the
// probe plane never proved. Either way no plugin process starts for
// the gated operation.
func RequireCapability(probeBody []byte, name string) error {
	if err := DecodeProbe(probeBody); err != nil {
		return err
	}
	// The re-decodes below are exact replays over bytes DecodeProbe
	// just validated, in this order: their faults are nil by
	// construction, so discarding them is a coupling to the call
	// above, not to untrusted input. Moving DecodeProbe after them
	// would silently drop faults; keep the validation first.
	members, _ := decodeStrictObject(probeBody)
	capabilities, _ := decodeStrictObject(members["capabilities"])
	known := false
	for _, registry := range capabilityOrder {
		if name == registry {
			known = true
		}
	}
	if !known {
		failure, err := failInvalid("capability is not a registry member")
		if err != nil {
			return err
		}
		return failure
	}
	value, _ := decodeStrictObject(capabilities[name])
	status, _ := rawString(value["status"])
	enabled, _ := rawBool(value["enabled"])
	if !CapabilityUsable(status, enabled) {
		failure, err := failInvalid("probe does not establish the capability as available")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
