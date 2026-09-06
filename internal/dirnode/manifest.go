package dirnode

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Section 7.9 Directory Node Manifest
// 1.0.0: the closed seventeen-member object a manifest operation
// returns as its success body under either major. The manifest
// declares possible surfaces, not runtime availability; a status of
// conditional, unavailable, or unknown is never reported as
// available, and unknown is never rewritten as unsupported.

// capabilityOrder is the exact eight-name directory-capability
// registry Section 7.9 shows. The manifest capabilities map carries
// exactly these keys; a ninth key advertises a surface the contract
// never defined, and a missing key hides one it did.
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

// Capabilities returns the Section 7.9 capability registry in
// order. The result is a copy; the registry cannot be mutated
// through it.
func Capabilities() []string {
	return append([]string(nil), capabilityOrder...)
}

// validCapability reports whether the name is a registry member. It
// is a table loop so the census derives it like every other closed
// vocabulary: an inline comparison chain here would be invisible to
// the census.
func validCapability(name string) bool {
	for _, allowed := range capabilityOrder {
		if name == allowed {
			return true
		}
	}
	return false
}

// capabilityStatuses is the closed CapabilityResult status
// vocabulary. Section 8.1 labels are not aliases: unknown must not
// be rewritten as unsupported, and conditional must not be
// advertised as available.
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

// manifestMembers is the exact required member set DecodeManifest
// accepts: the schema envelope, node identity, the three façade
// executable/module bindings, the supported versions, the operation
// and schema registries, the tuple registry binding, the eight
// capabilities, the policy bindings, the limits, and extensions.
var manifestMembers = map[string]bool{
	"schema":                          true,
	"schema_version":                  true,
	"node_id":                         true,
	"node_version":                    true,
	"host_id":                         true,
	"executable_sha256":               true,
	"provider_manifest_digest":        true,
	"session_adapter_manifest_digest": true,
	"supported_protocol_versions":     true,
	"operations":                      true,
	"schemas":                         true,
	"environment_tuple_registry_id":   true,
	"capabilities":                    true,
	"redaction_policy_ids":            true,
	"enrichment_profile_ids":          true,
	"limits":                          true,
	"extensions":                      true,
}

// manifestRequired lists manifestMembers in a fixed order so a body
// missing several members always names the same one.
var manifestRequired = []string{
	"schema",
	"schema_version",
	"node_id",
	"node_version",
	"host_id",
	"executable_sha256",
	"provider_manifest_digest",
	"session_adapter_manifest_digest",
	"supported_protocol_versions",
	"operations",
	"schemas",
	"environment_tuple_registry_id",
	"capabilities",
	"redaction_policy_ids",
	"enrichment_profile_ids",
	"limits",
	"extensions",
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

// limitsMembers is the exact DirectoryNodeLimits member set.
var limitsMembers = map[string]bool{
	"max_frame_bytes":       true,
	"max_scan_instances":    true,
	"max_inventory_take":    true,
	"max_excerpt_count":     true,
	"max_excerpt_bytes":     true,
	"max_enrichment_events": true,
	"max_enrichment_bytes":  true,
	"extensions":            true,
}

// limitsRequired lists limitsMembers in a fixed order.
var limitsRequired = []string{
	"max_frame_bytes",
	"max_scan_instances",
	"max_inventory_take",
	"max_excerpt_count",
	"max_excerpt_bytes",
	"max_enrichment_events",
	"max_enrichment_bytes",
	"extensions",
}

// assertionMembers is the exact ContractAssertion member set.
var assertionMembers = map[string]bool{
	"contract_id":   true,
	"exact_version": true,
	"extensions":    true,
}

// assertionRequired lists assertionMembers in a fixed order.
var assertionRequired = []string{
	"contract_id",
	"exact_version",
	"extensions",
}

// Capability is one validated CapabilityResult: the status, the
// reason coherence (available carries a null reason; every other
// status requires one), the evidence digests, and the observation
// time.
type Capability struct {
	Status     string
	ReasonCode string
	HasReason  bool
	Evidence   []string
	ObservedAt string
}

// Limits is one validated DirectoryNodeLimits.
type Limits struct {
	MaxFrameBytes      uint64
	MaxScanInstances   uint64
	MaxInventoryTake   uint64
	MaxExcerptCount    uint64
	MaxExcerptBytes    uint64
	MaxEnrichmentEvent uint64
	MaxEnrichmentBytes uint64
}

// ContractAssertion is one validated schemas entry: the asserted
// contract identifier and its exact version.
type ContractAssertion struct {
	ContractID   string
	ExactVersion string
}

// Manifest is the validated Directory Node Manifest 1.0.0: the node
// identity, the façade bindings, the ordered registries, the
// capability map, and the limits. The canonical body is retained
// for the host-computed digest.
type Manifest struct {
	NodeID                 string
	NodeVersion            string
	HostID                 string
	ExecutableSHA256       string
	ProviderManifestDigest string
	AdapterManifestDigest  string
	SupportedVersions      []string
	Operations             []string
	Schemas                []ContractAssertion
	TupleRegistryID        string
	Capabilities           map[string]Capability
	RedactionPolicyIDs     []string
	EnrichmentProfileIDs   []string
	Limits                 Limits
	canonical              []byte
}

// Canonical returns the JCS bytes the digest was computed over.
func (manifest Manifest) Canonical() []byte {
	return append([]byte(nil), manifest.canonical...)
}

// Supports reports whether the manifest's
// supported_protocol_versions contains the exact selected version.
// Membership is exact: a manifest that names only 1.0.0 does not
// support a 2.0.0 attempt, and no coercion across majors is
// performed here or anywhere else.
func (manifest Manifest) Supports(version string) bool {
	for _, supported := range manifest.SupportedVersions {
		if supported == version {
			return true
		}
	}
	return false
}

// DecodeManifest validates one manifest-operation success body as
// the closed Section 7.9 Directory Node Manifest. A manifest whose
// operations are not the complete sorted eleven-name registry, or
// whose capabilities are not the exact eight-name map, advertises a
// surface the host cannot invoke or hides one it must, so both are
// refused outright. The three façade bindings are checked for shape
// here and for agreement with observed facts in
// CheckManifestBindings; a contradiction there is an integrity
// failure.
func DecodeManifest(body []byte) (Manifest, error) {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failViolation("node manifest "+fault.detail, fault.member)
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if name, unknown := unknownMember(members, manifestMembers); unknown {
		failure, err := failViolation("node manifest carries unknown member", name)
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if name, missing := missingMember(members, manifestRequired); missing {
		failure, err := failViolation("node manifest misses a required member", name)
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != ManifestSchema {
		failure, err := failViolation("node manifest schema is not the directory node manifest", "schema")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != ManifestSchemaVersion {
		failure, err := failViolation("node manifest version is not 1.0.0", "schema_version")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	nodeID, ok := checkStringBounds(members["node_id"], 1, 128)
	if !ok {
		failure, err := failViolation("node manifest node identifier is not a string[1..128]", "node_id")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	nodeVersion, ok := rawString(members["node_version"])
	if !ok || !checkSemver(nodeVersion) {
		failure, err := failViolation("node manifest node version is not SemVer", "node_version")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	hostID, ok := checkUUIDv7(members["host_id"])
	if !ok {
		failure, err := failViolation("node manifest host identifier is not a UUIDv7", "host_id")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	executable, ok := checkDigest(members["executable_sha256"])
	if !ok {
		failure, err := failViolation("node manifest executable binding is not a digest", "executable_sha256")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	providerDigest, ok := checkDigest(members["provider_manifest_digest"])
	if !ok {
		failure, err := failViolation("node manifest provider binding is not a digest", "provider_manifest_digest")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	adapterDigest, ok := checkDigest(members["session_adapter_manifest_digest"])
	if !ok {
		failure, err := failViolation("node manifest adapter binding is not a digest", "session_adapter_manifest_digest")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	supported, ok := checkSupportedVersions(members["supported_protocol_versions"])
	if !ok {
		failure, err := failViolation("node manifest supported versions are not sorted unique SemVer[1..16]", "supported_protocol_versions")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	operations, ok := checkManifestOperations(members["operations"])
	if !ok {
		failure, err := failViolation("node manifest operations are not the complete sorted eleven-name registry", "operations")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	schemas, ok := checkContractAssertions(members["schemas"])
	if !ok {
		failure, err := failViolation("node manifest schemas are not sorted unique ContractAssertion[15..64]", "schemas")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	tupleRegistry, ok := checkDigest(members["environment_tuple_registry_id"])
	if !ok {
		failure, err := failViolation("node manifest tuple registry binding is not a digest", "environment_tuple_registry_id")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	capabilities, ok := checkCapabilities(members["capabilities"])
	if !ok {
		failure, err := failViolation("node manifest capabilities are not the exact eight-name result map", "capabilities")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	redaction, ok := checkSortedUniqueDigests(members["redaction_policy_ids"], 1, 64)
	if !ok {
		failure, err := failViolation("node manifest redaction policies are not sorted unique digest[1..64]", "redaction_policy_ids")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	enrichment, ok := checkSortedUniqueDigests(members["enrichment_profile_ids"], 0, 256)
	if !ok {
		failure, err := failViolation("node manifest enrichment profiles are not sorted unique digest[0..256]", "enrichment_profile_ids")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	limits, ok := checkLimits(members["limits"])
	if !ok {
		failure, err := failViolation("node manifest limits are not the closed DirectoryNodeLimits", "limits")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failViolation("node manifest extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	canonical, err := canonicaljson.Canonicalize(body)
	if err != nil {
		failure, faultErr := failViolation("node manifest is not canonical JSON", "")
		if faultErr != nil {
			return Manifest{}, faultErr
		}
		return Manifest{}, failure
	}
	return Manifest{
		NodeID:                 nodeID,
		NodeVersion:            nodeVersion,
		HostID:                 hostID.String(),
		ExecutableSHA256:       executable.String(),
		ProviderManifestDigest: providerDigest.String(),
		AdapterManifestDigest:  adapterDigest.String(),
		SupportedVersions:      supported,
		Operations:             operations,
		Schemas:                schemas,
		TupleRegistryID:        tupleRegistry.String(),
		Capabilities:           capabilities,
		RedactionPolicyIDs:     digestStrings(redaction),
		EnrichmentProfileIDs:   digestStrings(enrichment),
		Limits:                 limits,
		canonical:              canonical,
	}, nil
}

// checkSupportedVersions reports whether the member is a sorted
// unique SemVer array in the count bound. Ordering is bytewise over
// the version strings, the same rule every sorted-unique check in
// this package applies: no member type gets its own comparison.
func checkSupportedVersions(raw json.RawMessage) ([]string, bool) {
	versions, ok := checkSortedUniqueStrings(raw, 5, 64, 1, 16)
	if !ok {
		return nil, false
	}
	for _, version := range versions {
		if !checkSemver(version) {
			return nil, false
		}
	}
	return versions, true
}

// checkManifestOperations reports whether the member is exactly the
// eleven-name operation registry in sorted unique order. Sortedness
// is the rule here — the manifest carries the registry sorted, not
// in section table order — so the comparison sorts a copy of the
// table and matches element-wise.
func checkManifestOperations(raw json.RawMessage) ([]string, bool) {
	elements, ok := decodeArray(raw)
	if !ok || len(elements) != len(operationOrder) {
		return nil, false
	}
	want := Operations()
	sort.Strings(want)
	operations := make([]string, 0, len(elements))
	for index, element := range elements {
		name, ok := rawString(element)
		if !ok || name != want[index] {
			return nil, false
		}
		operations = append(operations, name)
	}
	return operations, true
}

// checkContractAssertions reports whether the member is a sorted
// unique ContractAssertion array in the count bound. Uniqueness is
// over the canonical encoding of each entry, so two entries that
// differ only in key order or duplicate members still collide.
func checkContractAssertions(raw json.RawMessage) ([]ContractAssertion, bool) {
	elements, ok := decodeArray(raw)
	if !ok || len(elements) < 15 || len(elements) > 64 {
		return nil, false
	}
	assertions := make([]ContractAssertion, 0, len(elements))
	encodings := make([]string, 0, len(elements))
	for _, element := range elements {
		members, fault := decodeStrictObject(bytesTrimSpace(element))
		if fault != nil {
			return nil, false
		}
		if name, unknown := unknownMember(members, assertionMembers); unknown {
			_ = name
			return nil, false
		}
		if name, missing := missingMember(members, assertionRequired); missing {
			_ = name
			return nil, false
		}
		identifier, ok := rawString(members["contract_id"])
		if !ok || !checkURI(identifier) {
			return nil, false
		}
		version, ok := rawString(members["exact_version"])
		if !ok || !checkSemver(version) {
			return nil, false
		}
		if !checkExtensions(members["extensions"]) {
			return nil, false
		}
		canonical, err := canonicaljson.Canonicalize(bytesTrimSpace(element))
		if err != nil {
			return nil, false
		}
		assertions = append(assertions, ContractAssertion{ContractID: identifier, ExactVersion: version})
		encodings = append(encodings, string(canonical))
	}
	for index := 1; index < len(encodings); index++ {
		if encodings[index-1] >= encodings[index] {
			return nil, false
		}
	}
	return assertions, true
}

// checkCapabilities reports whether the member is an object with
// exactly the eight registry keys, each a closed CapabilityResult
// with coherent reason: available carries a null reason and every
// other status requires one. The coherence runs both ways: a null
// reason on a non-available status and a set reason on an
// available status both refuse.
func checkCapabilities(raw json.RawMessage) (map[string]Capability, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return nil, false
	}
	if len(members) != len(capabilityOrder) {
		return nil, false
	}
	capabilities := make(map[string]Capability, len(capabilityOrder))
	for _, name := range capabilityOrder {
		member, present := members[name]
		if !present {
			return nil, false
		}
		capability, ok := checkCapability(member)
		if !ok {
			return nil, false
		}
		capabilities[name] = capability
	}
	return capabilities, true
}

// checkCapability validates one CapabilityResult object.
func checkCapability(raw json.RawMessage) (Capability, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return Capability{}, false
	}
	if name, unknown := unknownMember(members, capabilityMembers); unknown {
		_ = name
		return Capability{}, false
	}
	if name, missing := missingMember(members, capabilityRequired); missing {
		_ = name
		return Capability{}, false
	}
	status, ok := rawString(members["status"])
	if !ok || !validCapabilityStatus(status) {
		return Capability{}, false
	}
	reason, reasonNull := members["reason_code"]
	hasReason := false
	var reasonCode string
	if reasonNull && isNull(reason) {
		hasReason = false
	} else {
		value, ok := checkStringBounds(reason, 1, 128)
		if !ok {
			return Capability{}, false
		}
		hasReason = true
		reasonCode = value
	}
	if status == "available" && hasReason {
		return Capability{}, false
	}
	if status != "available" && !hasReason {
		return Capability{}, false
	}
	evidence, ok := checkSortedUniqueDigests(members["evidence_ids"], 0, 64)
	if !ok {
		return Capability{}, false
	}
	observed, ok := checkTimestamp(members["observed_at"])
	if !ok {
		return Capability{}, false
	}
	if !checkExtensions(members["extensions"]) {
		return Capability{}, false
	}
	return Capability{
		Status:     status,
		ReasonCode: reasonCode,
		HasReason:  hasReason,
		Evidence:   digestStrings(evidence),
		ObservedAt: observed.String(),
	}, true
}

// checkLimits validates one DirectoryNodeLimits object against the
// seven manifest bounds plus the frame ceiling: a limits object
// admitting frames larger than the 8 MiB transport bound would
// license an oversize line.
func checkLimits(raw json.RawMessage) (Limits, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return Limits{}, false
	}
	if name, unknown := unknownMember(members, limitsMembers); unknown {
		_ = name
		return Limits{}, false
	}
	if name, missing := missingMember(members, limitsRequired); missing {
		_ = name
		return Limits{}, false
	}
	frame, ok := checkUint53Bounds(members["max_frame_bytes"], 1, MaxFrameBytes)
	if !ok {
		return Limits{}, false
	}
	instances, ok := checkUint53Bounds(members["max_scan_instances"], 1, 65536)
	if !ok {
		return Limits{}, false
	}
	take, ok := checkUint53Bounds(members["max_inventory_take"], 1, 1000)
	if !ok {
		return Limits{}, false
	}
	excerptCount, ok := checkUint53Bounds(members["max_excerpt_count"], 0, 20)
	if !ok {
		return Limits{}, false
	}
	excerptBytes, ok := checkUint53Bounds(members["max_excerpt_bytes"], 0, 4096)
	if !ok {
		return Limits{}, false
	}
	enrichmentEvents, ok := checkUint53Bounds(members["max_enrichment_events"], 1, 5000)
	if !ok {
		return Limits{}, false
	}
	enrichmentBytes, ok := checkUint53Bounds(members["max_enrichment_bytes"], 1, 4194304)
	if !ok {
		return Limits{}, false
	}
	if !checkExtensions(members["extensions"]) {
		return Limits{}, false
	}
	return Limits{
		MaxFrameBytes:      frame,
		MaxScanInstances:   instances,
		MaxInventoryTake:   take,
		MaxExcerptCount:    excerptCount,
		MaxExcerptBytes:    excerptBytes,
		MaxEnrichmentEvent: enrichmentEvents,
		MaxEnrichmentBytes: enrichmentBytes,
	}, true
}

// digestStrings renders validated digests in order.
func digestStrings(digests []scalar.Digest) []string {
	out := make([]string, 0, len(digests))
	for _, digest := range digests {
		out = append(out, digest.String())
	}
	return out
}

// bytesTrimSpace trims JSON-insignificant whitespace around a raw
// member. Members arrive untrimmed from the strict decoder; object
// and array checks below accept surrounding whitespace, but the
// canonicalizer and the duplicate scan need the exact span, so the
// trim happens at each nested-object entry point, never by
// rewriting the parent frame.
func bytesTrimSpace(raw json.RawMessage) []byte {
	return bytes.TrimSpace(raw)
}

// ManifestDigest returns the host-computed JCS SHA-256 of the
// validated manifest. It is computed by the host, never embedded by
// the node, so a manifest that names its own digest cannot mint
// trust.
func ManifestDigest(manifest Manifest) scalar.Digest {
	return scalar.SHA256Digest(manifest.canonical)
}

// ObservedFacade is the independently observed executable facts the
// manifest bindings must agree with: the host-observed executable
// digest and the provider and adapter manifest digests from their
// own discovery, never from the node.
type ObservedFacade struct {
	ExecutableSHA256       string
	ProviderManifestDigest string
	SessionAdapterDigest   string
}

// CheckManifestBindings requires the manifest's three façade
// bindings to equal the independently observed facts. A
// contradiction is an integrity failure, not a usable node: the
// three façade executable/module bindings and the tuple
// declarations must agree, and agreement with observed facts is
// what makes the manifest this node's. A failed or partial
// observation never means the candidate is absent.
func CheckManifestBindings(manifest Manifest, observed ObservedFacade) error {
	if _, ok := checkDigestString(observed.ExecutableSHA256); !ok {
		failure, err := failInvalid("observed executable digest is not a digest", "executable_sha256")
		if err != nil {
			return err
		}
		return failure
	}
	if _, ok := checkDigestString(observed.ProviderManifestDigest); !ok {
		failure, err := failInvalid("observed provider manifest digest is not a digest", "provider_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	if _, ok := checkDigestString(observed.SessionAdapterDigest); !ok {
		failure, err := failInvalid("observed adapter manifest digest is not a digest", "session_adapter_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	if manifest.ExecutableSHA256 != observed.ExecutableSHA256 {
		failure, err := failIntegrity("node manifest executable binding contradicts the observed executable", "executable_sha256")
		if err != nil {
			return err
		}
		return failure
	}
	if manifest.ProviderManifestDigest != observed.ProviderManifestDigest {
		failure, err := failIntegrity("node manifest provider binding contradicts the observed manifest", "provider_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	if manifest.AdapterManifestDigest != observed.SessionAdapterDigest {
		failure, err := failIntegrity("node manifest adapter binding contradicts the observed manifest", "session_adapter_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// AvailableCapabilities returns the manifest capabilities with
// status available, in registry order. Conditional, unavailable,
// and unknown are never upgraded: the Section 8.1 labels are not
// aliases, so a non-available status is reported by omission, and
// a caller that needs the reason reads the manifest map.
func AvailableCapabilities(manifest Manifest) []string {
	var out []string
	for _, name := range capabilityOrder {
		if capability, present := manifest.Capabilities[name]; present && capability.Status == "available" {
			out = append(out, name)
		}
	}
	return out
}
