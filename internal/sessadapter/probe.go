package sessadapter

import (
	"encoding/json"
)

// This file validates the Section 7.8 Session Adapter Probe 1.0.0,
// the capability-value rules, the probe equality gates against the
// verified manifest and host values, the doctor health gate, and
// the target-write capability gates.

// probeSchema is the exact schema identifier the probe carries.
const probeSchema = "urn:ax:schema:session-adapter-probe"

// probeSchemaVersion is the only probe version this host accepts.
const probeSchemaVersion = "1.0.0"

// probeMembers is the exact Session Adapter Probe 1.0.0 member set:
// schema, schema_version, provider_id, adapter_manifest_digest,
// adapter_version, environment, capabilities, warnings, and
// extensions.
var probeMembers = map[string]bool{
	"schema":                  true,
	"schema_version":          true,
	"provider_id":             true,
	"adapter_manifest_digest": true,
	"adapter_version":         true,
	"environment":             true,
	"capabilities":            true,
	"warnings":                true,
	"extensions":              true,
}

// probeRequired lists probeMembers in a fixed order.
var probeRequired = []string{
	"schema",
	"schema_version",
	"provider_id",
	"adapter_manifest_digest",
	"adapter_version",
	"environment",
	"capabilities",
	"warnings",
	"extensions",
}

// capabilityValueMembers is the exact per-capability value member
// set: status, enabled, evidence, and detail. Only
// status=available permits enabled=true.
var capabilityValueMembers = map[string]bool{
	"status":   true,
	"enabled":  true,
	"evidence": true,
	"detail":   true,
}

// capabilityValueRequired lists capabilityValueMembers in order.
var capabilityValueRequired = []string{
	"status",
	"enabled",
	"evidence",
	"detail",
}

// Capability is one validated capability value.
type Capability struct {
	Status   string
	Enabled  bool
	Evidence string
	Detail   string
}

// Probe is one validated Session Adapter Probe 1.0.0.
type Probe struct {
	ProviderID     string
	ManifestDigest string
	AdapterVersion string
	Environment    Tuple
	Capabilities   map[string]Capability
	Warnings       []string
}

// decodeCapabilityValue validates one closed capability value:
// status in available|conditional|unsupported|unknown, evidence in
// the seven-vocabulary, detail as string[0..2048], and the
// coherence rule that only available permits enabled=true. A
// conditional, unsupported, or unknown capability with
// enabled=true is a contradictory fact and invalidates the whole
// probe, never a usable surface.
func decodeCapabilityValue(raw json.RawMessage, name string) (Capability, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("adapter capability "+fault.detail, name)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	if member, unknown := unknownMember(members, capabilityValueMembers); unknown {
		failure, err := failProtocol("adapter capability value carries unknown member", member)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	if member, missing := missingMember(members, capabilityValueRequired); missing {
		failure, err := failProtocol("adapter capability value misses a required member", member)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	status, ok := rawString(members["status"])
	if !ok || !validCapabilityStatus(status) {
		failure, err := failProtocol("adapter capability status is outside available|conditional|unsupported|unknown", name)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	enabled, ok := rawBool(members["enabled"])
	if !ok {
		failure, err := failProtocol("adapter capability enabled flag is not a boolean", name)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	if enabled && status != "available" {
		failure, err := failProtocol("adapter capability is enabled without available status", name)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	evidence, ok := rawString(members["evidence"])
	if !ok || !validCapabilityEvidence(evidence) {
		failure, err := failProtocol("adapter capability evidence is outside the seven-evidence vocabulary", name)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	detail, ok := checkStringBounds(members["detail"], 0, 2048)
	if !ok {
		failure, err := failProtocol("adapter capability detail is not a string[0..2048]", name)
		if err != nil {
			return Capability{}, err
		}
		return Capability{}, failure
	}
	return Capability{
		Status:   status,
		Enabled:  enabled,
		Evidence: evidence,
		Detail:   detail,
	}, nil
}

// capabilityEvidences is the seven-evidence vocabulary Section 7.8
// states. Membership is a table loop so the census derives it like
// every other closed vocabulary.
var capabilityEvidences = []string{
	"documented",
	"probed",
	"accepted_test",
	"provider_contract",
	"inferred",
	"acceptance_required",
	"none",
}

// validCapabilityEvidence reports whether the evidence token is one
// of the seven Section 7.8 values.
func validCapabilityEvidence(evidence string) bool {
	for _, allowed := range capabilityEvidences {
		if evidence == allowed {
			return true
		}
	}
	return false
}

// capabilityStatuses is the four-status vocabulary Section 7.8
// states. Membership is a table loop so the census derives it like
// every other closed vocabulary.
var capabilityStatuses = []string{
	"available",
	"conditional",
	"unsupported",
	"unknown",
}

// validCapabilityStatus reports whether the status token is one of
// the four Section 7.8 values.
func validCapabilityStatus(status string) bool {
	for _, allowed := range capabilityStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// registryEntryStatuses is the three-status doctor vocabulary the
// Section 7.8 doctor row states: accepted, revoked, and absent.
// Membership is a table loop so the census derives it like every
// other closed vocabulary.
var registryEntryStatuses = []string{
	"accepted",
	"revoked",
	"absent",
}

// validRegistryEntryStatus reports whether the status is one of
// the three Section 7.8 doctor entry statuses.
func validRegistryEntryStatus(status string) bool {
	for _, allowed := range registryEntryStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// DecodeProbe validates one probe-operation success body as the
// closed Section 7.8 Session Adapter Probe: the exact members, all
// fifteen capabilities with no omission and no extra, and the
// value rules above. Missing, extra, duplicated, malformed, or
// contradictory facts invalidate the whole probe and never mean
// unsupported. Equality against the manifest and host values is
// the separate CheckProbe gate: decoding proves shape, not
// binding.
func DecodeProbe(body []byte) (Probe, error) {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol("adapter probe "+fault.detail, fault.member)
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	if name, unknown := unknownMember(members, probeMembers); unknown {
		failure, err := failProtocol("adapter probe carries unknown member", name)
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	if name, missing := missingMember(members, probeRequired); missing {
		failure, err := failProtocol("adapter probe misses a required member", name)
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != probeSchema {
		failure, err := failProtocol("adapter probe schema is not the session adapter probe", "schema")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != probeSchemaVersion {
		failure, err := failProtocol("adapter probe version is not 1.0.0", "schema_version")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	providerID, ok := rawString(members["provider_id"])
	if !ok || !providerIDPattern.MatchString(providerID) {
		failure, err := failProtocol("adapter probe provider identifier is not a provider-id", "provider_id")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	manifestDigest, ok := checkDigest(members["adapter_manifest_digest"])
	if !ok {
		failure, err := failProtocol("adapter probe manifest digest is not a digest", "adapter_manifest_digest")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	adapterVersion, ok := rawString(members["adapter_version"])
	if !ok || !semverPattern.MatchString(adapterVersion) {
		failure, err := failProtocol("adapter probe adapter version is not SemVer", "adapter_version")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	environment, err := DecodeTuple(members["environment"])
	if err != nil {
		return Probe{}, err
	}
	capabilities, err := decodeCapabilities(members["capabilities"])
	if err != nil {
		return Probe{}, err
	}
	warnings, ok := checkSortedUniqueStrings(members["warnings"], 0, 2048, 0, 1024)
	if !ok {
		failure, err := failProtocol("adapter probe warnings are not sorted unique string[0..2048][0..1024]", "warnings")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("adapter probe extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return Probe{}, err
		}
		return Probe{}, failure
	}
	return Probe{
		ProviderID:     providerID,
		ManifestDigest: manifestDigest.String(),
		AdapterVersion: adapterVersion,
		Environment:    environment,
		Capabilities:   capabilities,
		Warnings:       warnings,
	}, nil
}

// decodeCapabilities validates the closed capability map: exactly
// the fifteen registry names, each with a closed value. The map
// carries no order — names are the keys — so omission and extra
// are the two refused directions, and duplication is already
// refused by the strict decode.
func decodeCapabilities(raw json.RawMessage) (map[string]Capability, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("adapter capabilities "+fault.detail, "capabilities")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if len(members) != len(capabilityOrder) {
		failure, err := failProtocol("adapter capabilities do not carry exactly fifteen entries", "capabilities")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	capabilities := make(map[string]Capability, len(capabilityOrder))
	for _, name := range capabilityOrder {
		rawValue, present := members[name]
		if !present {
			failure, err := failProtocol("adapter capabilities omit a registry capability", name)
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		value, err := decodeCapabilityValue(rawValue, name)
		if err != nil {
			return nil, err
		}
		capabilities[name] = value
	}
	return capabilities, nil
}

// ProbeHostFacts are the host-held values the probe must equal:
// the provider identifier and candidate kind from the probe
// request the host built, the manifest digest the host computed,
// and the manifest the host verified. Nothing here comes from the
// adapter.
type ProbeHostFacts struct {
	ExpectedProviderID string
	ExpectedKind       CandidateKind
	ManifestDigest     string
	Manifest           Manifest
}

// CheckProbe enforces the Section 7.8 probe equalities: provider
// ID, manifest digest, and adapter version equal the verified
// manifest and host values, and the reported tuple is the
// manifest's tuple — the adapter's one semantic native
// environment, so a probe reporting any other tuple is an
// unrequested tuple and is refused. The tuple's adapter version
// must equal the probe's own adapter version: both name the same
// probed software, and a probe disagreeing with itself is
// refused. Whether the tuple's environment version satisfies the
// manifest's version-range constraint language is a stated bound:
// the range grammar has no evaluation rule in Section 7.8.
func CheckProbe(probe Probe, facts ProbeHostFacts) error {
	if probe.ProviderID != facts.ExpectedProviderID {
		failure, err := failProtocol("adapter probe provider identifier does not match the requested provider", "provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if probe.ProviderID != facts.Manifest.ProviderID {
		failure, err := failProtocol("adapter probe provider identifier does not match the verified manifest", "provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if probe.ManifestDigest != facts.ManifestDigest {
		failure, err := failProtocol("adapter probe manifest digest does not match the host-computed digest", "adapter_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	if probe.AdapterVersion != facts.Manifest.AdapterVersion {
		failure, err := failProtocol("adapter probe adapter version does not match the verified manifest", "adapter_version")
		if err != nil {
			return err
		}
		return failure
	}
	if probe.Environment.EnvironmentID != facts.Manifest.EnvironmentID {
		failure, err := failUnsupportedTuple("adapter probe reports a tuple outside the manifest environment", probe.Environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	if probe.Environment.AdapterVersion != probe.AdapterVersion {
		failure, err := failProtocol("adapter probe tuple adapter version does not match the probe adapter version", "environment")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// CheckProbeRequest binds the probe request the host built to the
// trusted candidate: the expected provider identifier and
// candidate kind must equal the candidate the host will invoke.
// A request naming a different provider or kind would attest a
// probe for a candidate the host never observed.
func CheckProbeRequest(expectedProviderID string, expectedKind CandidateKind, candidate TrustedCandidate) error {
	if expectedProviderID != candidate.ProviderID {
		failure, err := failInvalid("probe request provider identifier does not match the trusted candidate", "expected_provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if expectedKind != candidate.Kind {
		failure, err := failInvalid("probe request candidate kind does not match the trusted candidate", "expected_candidate_kind")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// CapabilityUsable reports whether the named capability is a usable
// surface: present in the closed registry with status=available and
// enabled=true. Conditional, unsupported, and unknown surfaces are
// never usable, and neither is a name outside the registry.
func CapabilityUsable(probe Probe, name string) bool {
	value, present := probe.Capabilities[name]
	if !present {
		return false
	}
	return value.Status == "available" && value.Enabled
}

// ProviderCapability is one provider-probe capability the
// target-write gate reads: the provider's portable-store and
// native-resume facts live in the provider probe, not the adapter
// probe, so the caller projects exactly those two surfaces.
type ProviderCapability struct {
	Status  string
	Enabled bool
}

// CheckTargetWriteGates enforces the Section 7.8 target-write
// conjunction: available canonical-write or official-import,
// native-read-back, native-resume-plan, and workspace-binding on
// the adapter side, plus provider portable-store and native-resume
// on the provider side. Exact execution bindings, accepted
// non-revoked tuple entries, and current fixture plus bounded
// resume-smoke evidence are the caller's adjacent gates
// (CheckBinding, CheckTupleAdmission, CheckDoctorResult): this
// gate decides the capability conjunction only. --force,
// experimental profiles, and environment-name-only matches cannot
// bypass these gates: the function takes no bypass parameter, so a
// caller cannot express one. Unknown sources may be archived only
// after safe byte enumeration; unknown targets never write —
// which this gate enforces by refusing every capability it cannot
// see as available and enabled.
func CheckTargetWriteGates(adapter map[string]Capability, provider map[string]ProviderCapability) error {
	if !capabilityMapUsable(adapter, "canonical_write") && !capabilityMapUsable(adapter, "official_import") {
		failure, err := failCapability("target write has neither usable canonical-write nor official-import", "canonical_write|official_import")
		if err != nil {
			return err
		}
		return failure
	}
	for _, name := range []string{"native_read_back", "native_resume_plan", "workspace_binding"} {
		if !capabilityMapUsable(adapter, name) {
			failure, err := failCapability("target write misses a usable adapter capability", name)
			if err != nil {
				return err
			}
			return failure
		}
	}
	for _, name := range []string{"portable_store", "native_resume"} {
		value, present := provider[name]
		if !present || value.Status != "available" || !value.Enabled {
			failure, err := failCapability("target write misses a usable provider capability", "provider:"+name)
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// capabilityMapUsable reports whether the named adapter capability
// is usable. An absent name is not usable: the map must be the
// decoded probe map, which always carries all fifteen.
func capabilityMapUsable(adapter map[string]Capability, name string) bool {
	value, present := adapter[name]
	if !present {
		return false
	}
	return value.Status == "available" && value.Enabled
}

// doctorResultMembers is the exact doctor success-body member set.
var doctorResultMembers = map[string]bool{
	"context":               true,
	"direction":             true,
	"registry_sequence":     true,
	"registry_entry_status": true,
	"findings":              true,
	"healthy":               true,
	"extensions":            true,
}

// doctorResultRequired lists doctorResultMembers in order.
var doctorResultRequired = []string{
	"context",
	"direction",
	"registry_sequence",
	"registry_entry_status",
	"findings",
	"healthy",
	"extensions",
}

// DoctorResult is one validated doctor success body.
type DoctorResult struct {
	Context          CallContext
	Direction        Direction
	RegistrySequence uint64
	EntryStatus      string
	Findings         []Finding
	Healthy          bool
}

// DecodeDoctorResult validates one closed doctor success body: the
// echoed context, the direction in the closed vocabulary, the
// registry sequence as uint53, the entry status in
// accepted|revoked|absent, the findings, and the health flag. The
// health rule itself is CheckDoctorHealthy: decoding proves shape,
// not health.
func DecodeDoctorResult(body []byte, sent CallContext, sentDirection Direction) (DoctorResult, error) {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol("doctor result "+fault.detail, fault.member)
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	if name, unknown := unknownMember(members, doctorResultMembers); unknown {
		failure, err := failProtocol("doctor result carries unknown member", name)
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	if name, missing := missingMember(members, doctorResultRequired); missing {
		failure, err := failProtocol("doctor result misses a required member", name)
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	context, err := CheckContextEcho(sent, members["context"])
	if err != nil {
		return DoctorResult{}, err
	}
	direction, ok := rawString(members["direction"])
	if !ok || !validDirection(direction) {
		failure, err := failProtocol("doctor result direction is outside source_read|target_write", "direction")
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	if Direction(direction) != sentDirection {
		failure, err := failProtocol("doctor result direction does not answer the request direction", "direction")
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	sequence, ok := checkUint53Bounds(members["registry_sequence"], 0, maxUint53)
	if !ok {
		failure, err := failProtocol("doctor result registry sequence is not a uint53", "registry_sequence")
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	status, ok := rawString(members["registry_entry_status"])
	if !ok || !validRegistryEntryStatus(status) {
		failure, err := failProtocol("doctor result entry status is outside accepted|revoked|absent", "registry_entry_status")
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	findings, err := DecodeFindings(members["findings"], 4096)
	if err != nil {
		return DoctorResult{}, err
	}
	healthy, ok := rawBool(members["healthy"])
	if !ok {
		failure, err := failProtocol("doctor result health flag is not a boolean", "healthy")
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("doctor result extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return DoctorResult{}, err
		}
		return DoctorResult{}, failure
	}
	return DoctorResult{
		Context:          context,
		Direction:        Direction(direction),
		RegistrySequence: sequence,
		EntryStatus:      status,
		Findings:         findings,
		Healthy:          healthy,
	}, nil
}

// CheckDoctorHealthy enforces the doctor health rule: healthy
// requires an accepted exact binding and every required
// capability. A healthy=true result over a revoked or absent entry
// is refused, and so is a healthy=true result with any required
// capability unusable. healthy=false is always structurally
// admissible: the rule states necessary conditions, not
// sufficient ones. Every required name must belong to the closed
// fifteen-registry; a required set naming anything else is a
// caller error, because the host cannot demand a surface the
// contract never defined.
func CheckDoctorHealthy(result DoctorResult, probe Probe, required []string) error {
	for _, name := range required {
		if !validCapabilityName(name) {
			failure, err := failInvalid("doctor required set names a capability outside the closed registry", "required")
			if err != nil {
				return err
			}
			return failure
		}
	}
	if !result.Healthy {
		return nil
	}
	if result.EntryStatus != "accepted" {
		failure, err := failCapability("doctor reports healthy without an accepted registry entry", "tuple_registry")
		if err != nil {
			return err
		}
		return failure
	}
	for _, name := range required {
		if !CapabilityUsable(probe, name) {
			failure, err := failCapability("doctor reports healthy without a usable required capability", name)
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// validCapabilityName reports whether the name belongs to the
// fifteen-capability registry.
func validCapabilityName(name string) bool {
	for _, capability := range capabilityOrder {
		if capability == name {
			return true
		}
	}
	return false
}

// DoctorRequiredCapabilities returns the closed required-capability
// set for one doctor direction. For target_write it is the Section
// 7.8 target-write adapter set: the import strategy selects which
// member of the canonical-write/official-import pair is required,
// and native-read-back, native-resume-plan, and workspace-binding
// are always required. For source_read it is empty: Section 7.8
// names no per-direction source set, so the caller carries its
// tuple-registry-derived requirements explicitly rather than
// reading them from an invented list. The strategy-to-pair mapping
// is an interpretation recorded here, not a derived rule: the
// target_official_import strategy authorizes the official-import
// member, every other target strategy the canonical-write member.
func DoctorRequiredCapabilities(direction Direction, useOfficialImport bool) []string {
	if direction != DirectionTargetWrite {
		return nil
	}
	writer := "canonical_write"
	if useOfficialImport {
		writer = "official_import"
	}
	return []string{writer, "native_read_back", "native_resume_plan", "workspace_binding"}
}
