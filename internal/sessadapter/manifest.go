package sessadapter

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Section 7.8 Session Adapter Manifest: the
// closed urn:ax:schema:session-adapter-manifest 1.0.0 object a
// manifest operation returns as its success body. The manifest
// declares possible surfaces, not runtime availability; every member
// below is required and the object is closed.

// manifestSchema is the exact schema identifier the manifest carries.
const manifestSchema = "urn:ax:schema:session-adapter-manifest"

// manifestSchemaVersion is the only manifest version this host accepts.
const manifestSchemaVersion = "1.0.0"

// capabilityOrder is the exact fifteen-name ordered capability
// registry Section 7.8 shows. The manifest and every probe body
// carry exactly these names in this order; a sixteenth name
// advertises a surface the contract never defined, and a missing
// name hides one it did.
var capabilityOrder = []string{
	"native_discovery",
	"stable_snapshot",
	"raw_capture",
	"canonical_read",
	"canonical_write",
	"native_read_back",
	"native_resume_plan",
	"official_import",
	"same_environment_lossless_clone",
	"tool_history",
	"usage_history",
	"compaction_history",
	"subagent_graph",
	"opaque_reasoning_roundtrip",
	"workspace_binding",
}

// Capabilities returns the Section 7.8 capability registry in order.
// The result is a copy; the registry cannot be mutated through it.
func Capabilities() []string {
	return append([]string(nil), capabilityOrder...)
}

// manifestMembers is the exact required member set DecodeManifest
// accepts: the schema envelope, identity, versioning, platforms, the
// Section 7.8 operation registry in section order, and the
// capability registry above.
var manifestMembers = map[string]bool{
	"schema":                    true,
	"schema_version":            true,
	"provider_id":               true,
	"environment_id":            true,
	"display_name":              true,
	"adapter_version":           true,
	"environment_version_range": true,
	"platforms":                 true,
	"operations":                true,
	"capability_names":          true,
	"extensions":                true,
}

// manifestRequired lists manifestMembers in a fixed order so a body
// missing several members always names the same one.
var manifestRequired = []string{
	"schema",
	"schema_version",
	"provider_id",
	"environment_id",
	"display_name",
	"adapter_version",
	"environment_version_range",
	"platforms",
	"operations",
	"capability_names",
	"extensions",
}

// Manifest is the validated Session Adapter Manifest 1.0.0: the
// provider and environment identity plus the ordered registries the
// probe and dispatch layers check. The canonical body is retained
// for the host-computed digest.
type Manifest struct {
	ProviderID     string
	EnvironmentID  string
	AdapterVersion string
	Platforms      []string
	Operations     []string
	Capabilities   []string
	canonical      []byte
}

// Canonical returns the JCS bytes the digest was computed over.
func (manifest Manifest) Canonical() []byte {
	return append([]byte(nil), manifest.canonical...)
}

// DecodeManifest validates one manifest-operation success body as the
// closed Section 7.8 Session Adapter Manifest. A manifest whose
// operations differ from the dispatch registry, or whose capability
// names differ from the registry above, advertises a surface the
// host cannot invoke or hides one it must, so both are refused
// outright.
func DecodeManifest(body []byte) (Manifest, error) {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol("adapter manifest "+fault.detail, fault.member)
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if name, unknown := unknownMember(members, manifestMembers); unknown {
		failure, err := failProtocol("adapter manifest carries unknown member", name)
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if name, missing := missingMember(members, manifestRequired); missing {
		failure, err := failProtocol("adapter manifest misses a required member", name)
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != manifestSchema {
		failure, err := failProtocol("adapter manifest schema is not the session adapter manifest", "schema")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != manifestSchemaVersion {
		failure, err := failProtocol("adapter manifest version is not 1.0.0", "schema_version")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	providerID, ok := rawString(members["provider_id"])
	if !ok || !providerIDPattern.MatchString(providerID) {
		failure, err := failProtocol("adapter manifest provider identifier is not a provider-id", "provider_id")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	environmentID, ok := rawString(members["environment_id"])
	if !ok || !environ.CheckEnvironmentID(environmentID) {
		failure, err := failProtocol("adapter manifest environment identifier is not an environment-id", "environment_id")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if _, ok := checkStringBounds(members["display_name"], 1, 128); !ok {
		failure, err := failProtocol("adapter manifest display name is not a string[1..128]", "display_name")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	adapterVersion, ok := rawString(members["adapter_version"])
	if !ok || !environ.CheckSemver(adapterVersion) {
		failure, err := failProtocol("adapter manifest adapter version is not SemVer", "adapter_version")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if _, ok := checkStringBounds(members["environment_version_range"], 1, 256); !ok {
		failure, err := failProtocol("adapter manifest environment version range is not a non-empty string[1..256]", "environment_version_range")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	platforms, ok := checkSortedUniqueStrings(members["platforms"], 1, 16, 1, 4)
	if !ok {
		failure, err := failProtocol("adapter manifest platforms are not a sorted unique non-empty subset", "platforms")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	for _, platform := range platforms {
		if _, err := scalar.ParsePlatform(platform); err != nil {
			failure, faultErr := failProtocol("adapter manifest platform is outside linux|macos|windows|wsl2", "platforms")
			if faultErr != nil {
				return Manifest{}, faultErr
			}
			return Manifest{}, failure
		}
	}
	operations, ok := checkOperationRegistry(members["operations"])
	if !ok {
		failure, err := failProtocol("adapter manifest operations are not the complete ordered fourteen-name registry", "operations")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	capabilities, ok := checkCapabilityRegistry(members["capability_names"])
	if !ok {
		failure, err := failProtocol("adapter manifest capability names are not the complete ordered fifteen-name registry", "capability_names")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failProtocol("adapter manifest extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return Manifest{}, err
		}
		return Manifest{}, failure
	}
	canonical, err := canonicaljson.Canonicalize(body)
	if err != nil {
		failure, faultErr := failProtocol("adapter manifest is not canonical JSON", "")
		if faultErr != nil {
			return Manifest{}, faultErr
		}
		return Manifest{}, failure
	}
	return Manifest{
		ProviderID:     providerID,
		EnvironmentID:  environmentID,
		AdapterVersion: adapterVersion,
		Platforms:      platforms,
		Operations:     operations,
		Capabilities:   capabilities,
		canonical:      canonical,
	}, nil
}

// checkOperationRegistry reports whether the member is exactly the
// dispatch registry in section order: same names, same positions,
// same length. Sortedness is not the rule here — order is — so the
// comparison is element-wise against operationOrder.
func checkOperationRegistry(raw json.RawMessage) ([]string, bool) {
	elements, ok := decodeArray(raw)
	if !ok || len(elements) != len(operationOrder) {
		return nil, false
	}
	operations := make([]string, 0, len(elements))
	for index, element := range elements {
		name, ok := rawString(element)
		if !ok || name != string(operationOrder[index]) {
			return nil, false
		}
		operations = append(operations, name)
	}
	return operations, true
}

// checkCapabilityRegistry reports whether the member is exactly the
// fifteen-name capability registry in section order.
func checkCapabilityRegistry(raw json.RawMessage) ([]string, bool) {
	elements, ok := decodeArray(raw)
	if !ok || len(elements) != len(capabilityOrder) {
		return nil, false
	}
	capabilities := make([]string, 0, len(elements))
	for index, element := range elements {
		name, ok := rawString(element)
		if !ok || name != capabilityOrder[index] {
			return nil, false
		}
		capabilities = append(capabilities, name)
	}
	return capabilities, true
}

// ManifestDigest returns the host-computed JCS SHA-256 of the
// validated manifest: the session_adapter_manifest_digest. It is
// computed by the host, never embedded by the adapter, so a
// manifest that names its own digest cannot mint trust.
func ManifestDigest(manifest Manifest) scalar.Digest {
	return scalar.SHA256Digest(manifest.canonical)
}
