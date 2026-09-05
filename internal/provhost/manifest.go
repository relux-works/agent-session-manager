package provhost

import (
	"encoding/json"
	"regexp"
)

// This file validates the Section 7.3 Provider Manifest: the closed
// urn:ax:schema:provider-manifest 1.0.0 object a manifest operation
// returns as its success body. The manifest declares possible surfaces,
// not runtime availability; every member below is required and the
// object is closed.

// manifestSchema is the exact schema identifier the manifest carries.
const manifestSchema = "urn:ax:schema:provider-manifest"

// manifestSchemaVersion is the only manifest version this host accepts.
const manifestSchemaVersion = "1.0.0"

// capabilityOrder is the exact seven-name ordered capability registry
// Section 7.3 shows. The manifest and every probe body carry exactly
// these names in this order; an eighth name advertises a surface the
// contract never defined, and a missing name hides one it did.
var capabilityOrder = []string{
	"native_resume",
	"portable_store",
	"managed_pty",
	"appserver",
	"task_board_primary",
	"prompt_spawn",
	"native_goal_binding",
}

// Capabilities returns the Section 7.3 capability registry in order.
// The result is a copy; the registry cannot be mutated through it.
func Capabilities() []string {
	return append([]string(nil), capabilityOrder...)
}

// manifestPlatforms is the closed platform vocabulary Section 7.3
// allows: a sorted, unique, non-empty subset of these four.
var manifestPlatforms = []string{"linux", "macos", "windows", "wsl2"}

// semverPattern is the SemVer grammar plugin_version and adapter
// versions satisfy: three dot-separated non-negative integers with no
// leading zeros, an optional pre-release, and optional build metadata.
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// providerIDPattern is the Section 7.1 provider-id grammar the
// manifest and identity layers read: [a-z][a-z0-9-]{0,31}.
var providerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// manifestMembers is the exact required member set DecodeManifest
// accepts: the schema envelope, identity, versioning, platforms, the
// Section 7.5 operation registry in manifest order, and the capability
// registry above.
var manifestMembers = map[string]bool{
	"schema":                 true,
	"schema_version":         true,
	"provider_id":            true,
	"display_name":           true,
	"plugin_version":         true,
	"provider_version_range": true,
	"platforms":              true,
	"operations":             true,
	"capability_names":       true,
}

// manifestRequired lists manifestMembers in a fixed order so a body
// missing several members always names the same one.
var manifestRequired = []string{
	"schema",
	"schema_version",
	"provider_id",
	"display_name",
	"plugin_version",
	"provider_version_range",
	"platforms",
	"operations",
	"capability_names",
}

// DecodeManifest validates one manifest-operation success body as the
// closed Section 7.3 Provider Manifest. It returns nil on a
// well-formed manifest and a provider_protocol_error naming the
// offending member otherwise. A manifest whose operations differ from
// the dispatch registry, or whose capability names differ from the
// registry above, advertises a surface the host cannot invoke or hides
// one it must, so both are refused outright.
func DecodeManifest(body []byte) error {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if name, unknown := unknownMember(members, manifestMembers); unknown {
		failure, err := failProtocol("manifest carries unknown member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if name, missing := missingMember(members, manifestRequired); missing {
		failure, err := failProtocol("manifest misses a required member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != manifestSchema {
		failure, err := failProtocol("manifest schema is not the provider manifest", "schema")
		if err != nil {
			return err
		}
		return failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != manifestSchemaVersion {
		failure, err := failProtocol("manifest schema_version is not 1.0.0", "schema_version")
		if err != nil {
			return err
		}
		return failure
	}
	if provider, ok := rawString(members["provider_id"]); !ok || !validProviderID(provider) {
		failure, err := failProtocol("manifest provider_id is not a provider id", "provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if name, ok := rawString(members["display_name"]); !ok || runeLength(name) < 1 || runeLength(name) > 128 {
		failure, err := failProtocol("manifest display_name is not 1..128 characters", "display_name")
		if err != nil {
			return err
		}
		return failure
	}
	if version, ok := rawString(members["plugin_version"]); !ok || !semverPattern.MatchString(version) {
		failure, err := failProtocol("manifest plugin_version is not SemVer", "plugin_version")
		if err != nil {
			return err
		}
		return failure
	}
	if constraint, ok := rawString(members["provider_version_range"]); !ok || runeLength(constraint) < 1 || runeLength(constraint) > 256 {
		failure, err := failProtocol("manifest provider_version_range is not 1..256 characters", "provider_version_range")
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkManifestPlatforms(members["platforms"]); err != nil {
		return err
	}
	if err := checkManifestOperations(members["operations"]); err != nil {
		return err
	}
	if err := checkManifestCapabilities(members["capability_names"]); err != nil {
		return err
	}
	return nil
}

// validProviderID reports whether the value satisfies the Section 7.1
// provider-id grammar without importing the trust package: the manifest
// layer reads the grammar, never the filesystem.
func validProviderID(value string) bool {
	return providerIDPattern.MatchString(value)
}

// checkManifestPlatforms enforces the sorted, unique, non-empty subset
// rule over the four platform enums. Each shape carries its own
// refusal so narrowing any one check reddens its own fixture.
func checkManifestPlatforms(raw json.RawMessage) error {
	platforms, ok := rawStringArray(raw)
	if !ok || len(platforms) == 0 {
		failure, err := failProtocol("manifest platforms is empty or not an array", "platforms")
		if err != nil {
			return err
		}
		return failure
	}
	allowed := map[string]bool{}
	for _, platform := range manifestPlatforms {
		allowed[platform] = true
	}
	for _, platform := range platforms {
		if !allowed[platform] {
			failure, err := failProtocol("manifest platforms names an unknown platform", "platforms")
			if err != nil {
				return err
			}
			return failure
		}
	}
	if !sortedUniqueStrings(platforms) {
		failure, err := failProtocol("manifest platforms are not sorted unique", "platforms")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// checkManifestOperations requires the ordered registry Section 7.5
// shows with no duplicates: the manifest must name exactly what the
// host can dispatch, in the order the manifest order defines.
func checkManifestOperations(raw json.RawMessage) error {
	operations, ok := rawStringArray(raw)
	if !ok || len(operations) != len(operationOrder) {
		failure, err := failProtocol("manifest operations differ from the registry", "operations")
		if err != nil {
			return err
		}
		return failure
	}
	for index, operation := range operationOrder {
		if operations[index] != string(operation) {
			failure, err := failProtocol("manifest operations differ from the registry", "operations")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// checkManifestCapabilities requires the exact seven-name ordered
// registry: no eighth advertised surface, no hidden one.
func checkManifestCapabilities(raw json.RawMessage) error {
	names, ok := rawStringArray(raw)
	if !ok || len(names) != len(capabilityOrder) {
		failure, err := failProtocol("manifest capability_names differ from the registry", "capability_names")
		if err != nil {
			return err
		}
		return failure
	}
	for index, name := range capabilityOrder {
		if names[index] != name {
			failure, err := failProtocol("manifest capability_names differ from the registry", "capability_names")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}
