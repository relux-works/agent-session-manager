package environ

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// observationSchema is the exact schema identifier the
// Environment Observation carries.
const observationSchema = "urn:ax:schema:environment-observation"

// observationSchemaVersion is the only observation version this
// library accepts.
const observationSchemaVersion = "1.0.0"

// observationMembers is the exact Section 10.8.1 member set.
var observationMembers = map[string]bool{
	"schema":                    true,
	"schema_version":            true,
	"observation_id":            true,
	"host_id":                   true,
	"installation_id":           true,
	"environment_id":            true,
	"environment_version":       true,
	"provider_id":               true,
	"platform":                  true,
	"architecture":              true,
	"backend_realm_fingerprint": true,
	"capabilities":              true,
	"authentication_status":     true,
	"runtime_status":            true,
	"observed_at":               true,
	"extensions":                true,
}

// observationRequired lists observationMembers in a fixed order so
// a record missing several members always names the same one.
var observationRequired = []string{
	"schema",
	"schema_version",
	"observation_id",
	"host_id",
	"installation_id",
	"environment_id",
	"environment_version",
	"provider_id",
	"platform",
	"architecture",
	"backend_realm_fingerprint",
	"capabilities",
	"authentication_status",
	"runtime_status",
	"observed_at",
	"extensions",
}

// capabilityOrder is the exact eight-name directory-capability
// registry Section 7.9 states. The observation capabilities map
// carries exactly these keys: a ninth key advertises a surface
// the contract never defined, and a missing key hides one it did.
// Section 10.8.1 reuses this registry rather than creating a
// second capability authority.
var capabilityOrder = []string{
	"directory_discovery",
	"directory_incremental_scan",
	"directory_head_digest",
	"directory_tail_preview",
	"native_title_read",
	"native_runtime_observation",
	"existing_session_adoption",
	"native_resume",
}

// capabilityStatuses is the closed CapabilityResult status
// vocabulary. Unknown is never rewritten as unsupported, and
// conditional is never advertised as available.
var capabilityStatuses = []string{
	"available",
	"conditional",
	"unavailable",
	"unknown",
}

// validCapabilityStatus reports whether the status is a registry
// member.
func validCapabilityStatus(status string) bool {
	for _, allowed := range capabilityStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// capabilityMembers is the exact CapabilityResult member set.
var capabilityMembers = map[string]bool{
	"status":       true,
	"reason_code":  true,
	"evidence_ids": true,
	"observed_at":  true,
	"extensions":   true,
}

// capabilityRequired lists capabilityMembers in a fixed order.
var capabilityRequired = []string{
	"status",
	"reason_code",
	"evidence_ids",
	"observed_at",
	"extensions",
}

// authenticationStatuses is the closed authentication_status
// vocabulary: status only, never credentials.
var authenticationStatuses = []string{
	"available",
	"missing",
	"expired",
	"unknown",
}

// validAuthenticationStatus reports whether the status is a
// registry member.
func validAuthenticationStatus(status string) bool {
	for _, allowed := range authenticationStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// runtimeStatuses is the closed runtime_status vocabulary.
var runtimeStatuses = []string{
	"available",
	"degraded",
	"unavailable",
}

// validRuntimeStatus reports whether the status is a registry
// member.
func validRuntimeStatus(status string) bool {
	for _, allowed := range runtimeStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// CapabilityResult is one validated directory capability result.
type CapabilityResult struct {
	Status     string
	ReasonCode string
	HasReason  bool
	Evidence   []string
	ObservedAt string
}

// Observation is one validated Environment Observation.
type Observation struct {
	ObservationID    string
	HostID           string
	InstallationID   string
	EnvironmentID    string
	EnvironmentVer   string
	ProviderID       string
	Platform         string
	Architecture     string
	RealmFingerprint string
	Capabilities     map[string]CapabilityResult
	Authentication   string
	Runtime          string
	ObservedAt       string
}

// DecodeEnvironmentObservation validates one closed Environment
// Observation: the exact schema envelope, digest and UUIDv7
// identities, the environment-id grammar with
// environment_version as string[1..128], the provider-id grammar
// (an explicit manifest mapping, never string inference), the
// platform and architecture vocabularies through the reused
// Section 13.14 tuple admission model, the non-secret realm
// digest, the exact eight-name capability map, the two status
// vocabularies, the diagnostic timestamp, and reverse-DNS
// extensions.
func DecodeEnvironmentObservation(body []byte) (Observation, error) {
	members, fault := DecodeStrictObject(bytesTrimSpace(body))
	if fault != nil {
		return Observation{}, refuse("observation "+fault.Detail, fault.Member)
	}
	if name, unknown := unknownMember(members, observationMembers); unknown {
		return Observation{}, refuse("observation carries unknown member", name)
	}
	if name, missing := missingMember(members, observationRequired); missing {
		return Observation{}, refuse("observation misses a required member", name)
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != observationSchema {
		return Observation{}, refuse("observation schema is not the environment observation", "schema")
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != observationSchemaVersion {
		return Observation{}, refuse("observation schema version is not 1.0.0", "schema_version")
	}
	observationID, ok := CheckDigest(members["observation_id"])
	if !ok {
		return Observation{}, refuse("observation identifier is not a digest", "observation_id")
	}
	hostID, ok := CheckUUIDv7(members["host_id"])
	if !ok {
		return Observation{}, refuse("observation host identifier is not a UUIDv7", "host_id")
	}
	installationID, ok := CheckDigest(members["installation_id"])
	if !ok {
		return Observation{}, refuse("observation installation identifier is not a digest", "installation_id")
	}
	environmentID, ok := rawString(members["environment_id"])
	if !ok || !CheckEnvironmentID(environmentID) {
		return Observation{}, refuse("observation environment identifier is not an environment-id", "environment_id")
	}
	environmentVersion, ok := CheckStringBounds(members["environment_version"], 1, 128)
	if !ok {
		return Observation{}, refuse("observation environment version is not a string[1..128]", "environment_version")
	}
	provider, ok := rawString(members["provider_id"])
	if !ok {
		return Observation{}, refuse("observation provider identifier is not a string", "provider_id")
	}
	providerID, err := scalar.ParseProviderID(provider)
	if err != nil {
		return Observation{}, refuse("observation provider identifier is not a provider-id", "provider_id")
	}
	platform, ok := rawString(members["platform"])
	if !ok {
		return Observation{}, refuse("observation platform is not a string", "platform")
	}
	if _, err := scalar.ParsePlatform(platform); err != nil {
		return Observation{}, refuse("observation platform is outside linux|macos|windows|wsl2", "platform")
	}
	architecture, ok := rawString(members["architecture"])
	if !ok || !validTupleArchitecture(architecture) {
		return Observation{}, refuse("observation architecture is outside amd64|arm64", "architecture")
	}
	realm, ok := CheckDigest(members["backend_realm_fingerprint"])
	if !ok {
		return Observation{}, refuse("observation realm fingerprint is not a digest", "backend_realm_fingerprint")
	}
	capabilities, ok := checkCapabilities(members["capabilities"])
	if !ok {
		return Observation{}, refuse("observation capabilities are not the exact eight-name result map", "capabilities")
	}
	authentication, ok := rawString(members["authentication_status"])
	if !ok || !validAuthenticationStatus(authentication) {
		return Observation{}, refuse("observation authentication status is outside available|missing|expired|unknown", "authentication_status")
	}
	runtime, ok := rawString(members["runtime_status"])
	if !ok || !validRuntimeStatus(runtime) {
		return Observation{}, refuse("observation runtime status is outside available|degraded|unavailable", "runtime_status")
	}
	observed, ok := CheckTimestamp(members["observed_at"])
	if !ok {
		return Observation{}, refuse("observation timestamp is not a timestamp", "observed_at")
	}
	if !CheckExtensions(members["extensions"]) {
		return Observation{}, refuse("observation extensions are not reverse-DNS keyed", "extensions")
	}
	return Observation{
		ObservationID:    observationID.String(),
		HostID:           hostID.String(),
		InstallationID:   installationID.String(),
		EnvironmentID:    environmentID,
		EnvironmentVer:   environmentVersion,
		ProviderID:       providerID.String(),
		Platform:         platform,
		Architecture:     architecture,
		RealmFingerprint: realm.String(),
		Capabilities:     capabilities,
		Authentication:   authentication,
		Runtime:          runtime,
		ObservedAt:       observed.String(),
	}, nil
}

// checkCapabilities validates the exact eight-name capability
// map: exactly the registry keys, each a closed CapabilityResult.
func checkCapabilities(raw json.RawMessage) (map[string]CapabilityResult, bool) {
	members, fault := DecodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return nil, false
	}
	if len(members) != len(capabilityOrder) {
		return nil, false
	}
	capabilities := make(map[string]CapabilityResult, len(capabilityOrder))
	for _, name := range capabilityOrder {
		member, present := members[name]
		if !present {
			return nil, false
		}
		capability, ok := checkCapabilityResult(member)
		if !ok {
			return nil, false
		}
		capabilities[name] = capability
	}
	return capabilities, true
}

// checkCapabilityResult validates one CapabilityResult object.
// Available carries a null reason and every other status carries
// one: the two directions are separate obligations, so an
// available-with-reason vector and a conditional-without-reason
// vector are separate rows in the battery.
func checkCapabilityResult(raw json.RawMessage) (CapabilityResult, bool) {
	members, fault := DecodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return CapabilityResult{}, false
	}
	if name, unknown := unknownMember(members, capabilityMembers); unknown {
		_ = name
		return CapabilityResult{}, false
	}
	if name, missing := missingMember(members, capabilityRequired); missing {
		_ = name
		return CapabilityResult{}, false
	}
	status, ok := rawString(members["status"])
	if !ok || !validCapabilityStatus(status) {
		return CapabilityResult{}, false
	}
	reasonMember, present := members["reason_code"]
	if !present {
		return CapabilityResult{}, false
	}
	hasReason := false
	var reasonCode string
	if isNull(reasonMember) {
		hasReason = false
	} else {
		value, ok := CheckStringBounds(reasonMember, 1, 128)
		if !ok {
			return CapabilityResult{}, false
		}
		hasReason = true
		reasonCode = value
	}
	if status == "available" && hasReason {
		return CapabilityResult{}, false
	}
	if status != "available" && !hasReason {
		return CapabilityResult{}, false
	}
	evidence, ok := CheckSortedUniqueDigests(members["evidence_ids"], 0, 64)
	if !ok {
		return CapabilityResult{}, false
	}
	observed, ok := CheckTimestamp(members["observed_at"])
	if !ok {
		return CapabilityResult{}, false
	}
	if !CheckExtensions(members["extensions"]) {
		return CapabilityResult{}, false
	}
	texts := make([]string, 0, len(evidence))
	for _, digest := range evidence {
		texts = append(texts, digest.String())
	}
	return CapabilityResult{
		Status:     status,
		ReasonCode: reasonCode,
		HasReason:  hasReason,
		Evidence:   texts,
		ObservedAt: observed.String(),
	}, true
}
