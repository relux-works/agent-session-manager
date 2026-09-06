package dirnode

import (
	"encoding/json"
)

// This file validates the Section 7.9 probe operation in both
// majors: the closed request body with its per-major platform
// vocabulary, and the closed success body with the node-build
// equality gate against the validated manifest.
//
// The probe row the operation table shows is the Request 2.0.0
// form; Request 1.0.0 differs only in its closed legacy platform
// vocabulary from the major-binding table. A v1 darwin request
// denotes the macOS host class of that published major, but darwin
// is not a valid v2 wire value and never appears in an Environment
// Observation; a v1 request cannot express WSL2. The host validates
// the vocabulary belonging to the negotiated major and never
// relabels, reinterprets, or coerces a token across majors.

// platformV1 is the closed Request 1.0.0 platform vocabulary.
var platformV1 = []string{
	"darwin",
	"linux",
	"windows",
}

// platformV2 is the closed Request 2.0.0 platform vocabulary.
var platformV2 = []string{
	"macos",
	"linux",
	"wsl2",
	"windows",
}

// architectures is the closed probe architecture vocabulary under
// either major.
var architectures = []string{
	"amd64",
	"arm64",
}

// validPlatformForMajor reports whether the platform token belongs
// to the negotiated major's vocabulary. Both vocabularies are table
// loops so the census derives them like every other closed
// vocabulary: an inline chain here would be invisible to the
// census, and a widened chain would silently admit a cross-major
// token.
func validPlatformForMajor(major int, platform string) bool {
	switch major {
	case MajorV1:
		for _, allowed := range platformV1 {
			if platform == allowed {
				return true
			}
		}
		return false
	case MajorV2:
		for _, allowed := range platformV2 {
			if platform == allowed {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// validArchitecture reports whether the architecture token is a
// registry member.
func validArchitecture(architecture string) bool {
	for _, allowed := range architectures {
		if architecture == allowed {
			return true
		}
	}
	return false
}

// probeRequestMembers is the exact probe request-body member set
// under either major.
var probeRequestMembers = map[string]bool{
	"platform":                  true,
	"architecture":              true,
	"requested_environment_ids": true,
	"requested_capabilities":    true,
	"extensions":                true,
}

// probeRequestRequired lists probeRequestMembers in a fixed order.
var probeRequestRequired = []string{
	"platform",
	"architecture",
	"requested_environment_ids",
	"requested_capabilities",
	"extensions",
}

// ProbeRequest is one validated probe request body: the platform
// and architecture tokens plus the sorted unique environment and
// capability filters.
type ProbeRequest struct {
	Major                 int
	Platform              string
	Architecture          string
	RequestedEnvironments []string
	RequestedCapabilities []string
}

// CheckProbeRequest validates one probe-operation request body
// under the negotiated major: the exact closed members, the
// platform token from the major's own vocabulary, a registry
// architecture, sorted unique environment identifiers in the count
// bound, and a sorted unique subset of the capability registry in
// the count bound. A v2 token on a v1 attempt — or the reverse —
// is a violation, never a relabeling.
func CheckProbeRequest(major int, body []byte) (ProbeRequest, error) {
	if _, ok := VersionForMajor(major); !ok {
		failure, err := failInvalid("probe request major is outside the locally supported registry", "protocol_version")
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	members, fault := decodeStrictObject(bytesTrimSpace(body))
	if fault != nil {
		failure, err := failViolation("probe request "+fault.detail, fault.member)
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	if name, unknown := unknownMember(members, probeRequestMembers); unknown {
		failure, err := failViolation("probe request carries unknown member", name)
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	if name, missing := missingMember(members, probeRequestRequired); missing {
		failure, err := failViolation("probe request misses a required member", name)
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	platform, ok := rawString(members["platform"])
	if !ok || !validPlatformForMajor(major, platform) {
		failure, err := failViolation("probe request platform is outside the negotiated major vocabulary", "platform")
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	architecture, ok := rawString(members["architecture"])
	if !ok || !validArchitecture(architecture) {
		failure, err := failViolation("probe request architecture is outside amd64|arm64", "architecture")
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	environments, ok := checkSortedUniqueStrings(members["requested_environment_ids"], 1, 64, 0, 64)
	if !ok {
		failure, err := failViolation("probe request environment identifiers are not sorted unique strings[0..64]", "requested_environment_ids")
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	for _, environment := range environments {
		if !environmentIDPattern.MatchString(environment) {
			failure, err := failViolation("probe request environment identifier is not an environment-id", "requested_environment_ids")
			if err != nil {
				return ProbeRequest{}, err
			}
			return ProbeRequest{}, failure
		}
	}
	capabilities, ok := checkSortedUniqueStrings(members["requested_capabilities"], 1, 64, 0, 8)
	if !ok {
		failure, err := failViolation("probe request capabilities are not sorted unique strings[0..8]", "requested_capabilities")
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	for _, capability := range capabilities {
		if !validCapability(capability) {
			failure, err := failViolation("probe request capability is outside the directory registry", "requested_capabilities")
			if err != nil {
				return ProbeRequest{}, err
			}
			return ProbeRequest{}, failure
		}
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failViolation("probe request extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return ProbeRequest{}, err
		}
		return ProbeRequest{}, failure
	}
	return ProbeRequest{
		Major:                 major,
		Platform:              platform,
		Architecture:          architecture,
		RequestedEnvironments: environments,
		RequestedCapabilities: capabilities,
	}, nil
}

// probeResponseMembers is the exact probe success-body member set.
var probeResponseMembers = map[string]bool{
	"host_id":       true,
	"node_build":    true,
	"policy_digest": true,
	"environments":  true,
	"findings":      true,
	"extensions":    true,
}

// probeResponseRequired lists probeResponseMembers in a fixed order.
var probeResponseRequired = []string{
	"host_id",
	"node_build",
	"policy_digest",
	"environments",
	"findings",
	"extensions",
}

// nodeBuildMembers is the exact DirectoryNodeBuild member set.
var nodeBuildMembers = map[string]bool{
	"node_id":                         true,
	"node_version":                    true,
	"executable_sha256":               true,
	"provider_manifest_digest":        true,
	"session_adapter_manifest_digest": true,
	"extensions":                      true,
}

// nodeBuildRequired lists nodeBuildMembers in a fixed order.
var nodeBuildRequired = []string{
	"node_id",
	"node_version",
	"executable_sha256",
	"provider_manifest_digest",
	"session_adapter_manifest_digest",
	"extensions",
}

// findingMembers is the exact AdapterFinding member set.
var findingMembers = map[string]bool{
	"severity":    true,
	"code":        true,
	"message":     true,
	"remediation": true,
	"extensions":  true,
}

// findingRequired lists findingMembers in a fixed order.
var findingRequired = []string{
	"severity",
	"code",
	"message",
	"remediation",
	"extensions",
}

// findingSeverities is the closed AdapterFinding severity
// vocabulary.
var findingSeverities = []string{
	"info",
	"warning",
	"error",
}

// validFindingSeverity reports whether the severity is a registry
// member.
func validFindingSeverity(severity string) bool {
	for _, allowed := range findingSeverities {
		if severity == allowed {
			return true
		}
	}
	return false
}

// NodeBuild is one validated DirectoryNodeBuild: the five identity
// values CheckNodeBuildEqualsManifest compares against the current
// manifest.
type NodeBuild struct {
	NodeID                 string
	NodeVersion            string
	ExecutableSHA256       string
	ProviderManifestDigest string
	AdapterManifestDigest  string
}

// Finding is one validated AdapterFinding.
type Finding struct {
	Severity    string
	Code        string
	Message     string
	Remediation string
	HasRemedy   bool
}

// ProbeResponse is one validated probe success body. Environment
// observations cross as validated array bounds of present objects:
// their content belongs to the shared-environment leaf of this
// Story, so checkProbeEnvironments requires each element to be an
// object and never inspects further — the one place that bound is
// enforced.
type ProbeResponse struct {
	HostID       string
	Build        NodeBuild
	PolicyDigest string
	Environments int
	Findings     []Finding
}

// CheckProbeResponse validates one probe-operation success body:
// the exact closed members, the host identifier, the closed node
// build, the policy digest, the bounded environment and finding
// arrays, and extensions.
func CheckProbeResponse(body []byte) (ProbeResponse, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(body))
	if fault != nil {
		failure, err := failViolation("probe response "+fault.detail, fault.member)
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	if name, unknown := unknownMember(members, probeResponseMembers); unknown {
		failure, err := failViolation("probe response carries unknown member", name)
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	if name, missing := missingMember(members, probeResponseRequired); missing {
		failure, err := failViolation("probe response misses a required member", name)
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	hostID, ok := checkUUIDv7(members["host_id"])
	if !ok {
		failure, err := failViolation("probe response host identifier is not a UUIDv7", "host_id")
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	build, ok := checkNodeBuild(members["node_build"])
	if !ok {
		failure, err := failViolation("probe response node build is not the closed DirectoryNodeBuild", "node_build")
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	policy, ok := checkDigest(members["policy_digest"])
	if !ok {
		failure, err := failViolation("probe response policy digest is not a digest", "policy_digest")
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	environments, ok := checkNestedObjects(members["environments"], 0, 256)
	if !ok {
		failure, err := failViolation("probe response environments are not EnvironmentObservation[0..256] by shape", "environments")
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	findings, ok := checkFindings(members["findings"])
	if !ok {
		failure, err := failViolation("probe response findings are not AdapterFinding[0..4096]", "findings")
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failViolation("probe response extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return ProbeResponse{}, err
		}
		return ProbeResponse{}, failure
	}
	return ProbeResponse{
		HostID:       hostID.String(),
		Build:        build,
		PolicyDigest: policy.String(),
		Environments: environments,
		Findings:     findings,
	}, nil
}

// checkNodeBuild validates one DirectoryNodeBuild object.
func checkNodeBuild(raw json.RawMessage) (NodeBuild, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return NodeBuild{}, false
	}
	if name, unknown := unknownMember(members, nodeBuildMembers); unknown {
		_ = name
		return NodeBuild{}, false
	}
	if name, missing := missingMember(members, nodeBuildRequired); missing {
		_ = name
		return NodeBuild{}, false
	}
	nodeID, ok := checkStringBounds(members["node_id"], 1, 128)
	if !ok {
		return NodeBuild{}, false
	}
	nodeVersion, ok := rawString(members["node_version"])
	if !ok || !checkSemver(nodeVersion) {
		return NodeBuild{}, false
	}
	executable, ok := checkDigest(members["executable_sha256"])
	if !ok {
		return NodeBuild{}, false
	}
	providerDigest, ok := checkDigest(members["provider_manifest_digest"])
	if !ok {
		return NodeBuild{}, false
	}
	adapterDigest, ok := checkDigest(members["session_adapter_manifest_digest"])
	if !ok {
		return NodeBuild{}, false
	}
	if !checkExtensions(members["extensions"]) {
		return NodeBuild{}, false
	}
	return NodeBuild{
		NodeID:                 nodeID,
		NodeVersion:            nodeVersion,
		ExecutableSHA256:       executable.String(),
		ProviderManifestDigest: providerDigest.String(),
		AdapterManifestDigest:  adapterDigest.String(),
	}, true
}

// checkNestedObjects reports the element count when the member is
// an array of objects in the count bound. Elements are required to
// be objects and are never inspected further here: their content
// owner is the shared-environment leaf, and inventing its
// validation here would collide with that ownership at
// integration.
func checkNestedObjects(raw json.RawMessage, minimumCount, maximumCount uint64) (int, bool) {
	elements, ok := decodeArray(raw)
	if !ok {
		return 0, false
	}
	if uint64(len(elements)) < minimumCount || uint64(len(elements)) > maximumCount {
		return 0, false
	}
	for _, element := range elements {
		if _, fault := decodeStrictObject(bytesTrimSpace(element)); fault != nil {
			return 0, false
		}
	}
	return len(elements), true
}

// checkFindings validates an AdapterFinding array in the count
// bound.
func checkFindings(raw json.RawMessage) ([]Finding, bool) {
	elements, ok := decodeArray(raw)
	if !ok || len(elements) > 4096 {
		return nil, false
	}
	findings := make([]Finding, 0, len(elements))
	for _, element := range elements {
		finding, ok := checkFinding(element)
		if !ok {
			return nil, false
		}
		findings = append(findings, finding)
	}
	return findings, true
}

// checkFinding validates one AdapterFinding object.
func checkFinding(raw json.RawMessage) (Finding, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return Finding{}, false
	}
	if name, unknown := unknownMember(members, findingMembers); unknown {
		_ = name
		return Finding{}, false
	}
	if name, missing := missingMember(members, findingRequired); missing {
		_ = name
		return Finding{}, false
	}
	severity, ok := rawString(members["severity"])
	if !ok || !validFindingSeverity(severity) {
		return Finding{}, false
	}
	code, ok := checkStringBounds(members["code"], 1, 128)
	if !ok {
		return Finding{}, false
	}
	message, ok := checkStringBounds(members["message"], 1, 4096)
	if !ok {
		return Finding{}, false
	}
	remediation, present, wasNull := checkOptionalString(members["remediation"], 1, 4096)
	if !present {
		return Finding{}, false
	}
	if !checkExtensions(members["extensions"]) {
		return Finding{}, false
	}
	return Finding{
		Severity:    severity,
		Code:        code,
		Message:     message,
		Remediation: remediation,
		HasRemedy:   !wasNull,
	}, true
}

// CheckNodeBuildEqualsManifest requires every node-build value to
// equal the current manifest: the probe answers for the manifest
// the bootstrap selected, not for a build the node invented
// mid-session. Any divergence is an integrity failure.
func CheckNodeBuildEqualsManifest(build NodeBuild, manifest Manifest) error {
	if build.NodeID != manifest.NodeID {
		failure, err := failIntegrity("probe node build node identifier differs from the manifest", "node_id")
		if err != nil {
			return err
		}
		return failure
	}
	if build.NodeVersion != manifest.NodeVersion {
		failure, err := failIntegrity("probe node build node version differs from the manifest", "node_version")
		if err != nil {
			return err
		}
		return failure
	}
	if build.ExecutableSHA256 != manifest.ExecutableSHA256 {
		failure, err := failIntegrity("probe node build executable binding differs from the manifest", "executable_sha256")
		if err != nil {
			return err
		}
		return failure
	}
	if build.ProviderManifestDigest != manifest.ProviderManifestDigest {
		failure, err := failIntegrity("probe node build provider binding differs from the manifest", "provider_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	if build.AdapterManifestDigest != manifest.AdapterManifestDigest {
		failure, err := failIntegrity("probe node build adapter binding differs from the manifest", "session_adapter_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
