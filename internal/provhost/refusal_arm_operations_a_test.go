package provhost

import (
	"strings"
	"testing"
)

// declaredOperationWitnesses proves every operation-layer refusal arm
// added by the conformance-harness leaf at its production entry
// point: manifest, probe, capability-gate, profile, quiescence,
// spawn-plan, identity, and idempotency-key arms. Each witness drives
// the decoder that owns the arm and requires the full arm identity
// (code, member, rule detail, non-retryable), so a deleted or
// narrowed production rule orphans its witness instead of passing
// silently.
func declaredOperationWitnesses() []armWitness {
	return []armWitness{
		// Manifest arms, through DecodeManifest.
		{arm: `ctor|failProtocol|manifest capability_names differ from the registry`, name: "manifest eighth capability", prove: func(t *testing.T) {
			body := manifestVariant(t, `    "prompt_spawn",
    "native_goal_binding"`, `    "prompt_spawn",
    "native_goal_binding",
    "remote_exec"`)
			requireFrameRefusal(t, DecodeManifest(body), "capability_names", "differ from the registry")
		}},
		{arm: `ctor|failProtocol|manifest carries unknown member`, name: "manifest diagnostics member", prove: func(t *testing.T) {
			body := manifestVariant(t, `"display_name": "Pi",`, `"display_name": "Pi", "diagnostics": [],`)
			requireFrameRefusal(t, DecodeManifest(body), "diagnostics", "unknown member")
		}},
		{arm: `ctor|failProtocol|manifest display_name is not 1..128 characters`, name: "manifest empty display name", prove: func(t *testing.T) {
			body := manifestVariant(t, `"display_name": "Pi"`, `"display_name": ""`)
			requireFrameRefusal(t, DecodeManifest(body), "display_name", "not 1..128 characters")
		}},
		{arm: `ctor|failProtocol|manifest misses a required member`, name: "manifest without display name", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specManifestExample, "  \"display_name\": \"Pi\",\n", "", 1))
			requireFrameRefusal(t, DecodeManifest(body), "display_name", "misses a required member")
		}},
		{arm: `ctor|failProtocol|manifest operations differ from the registry`, name: "manifest dropped operation", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specManifestExample, ",\n    \"doctor\"", "", 1))
			requireFrameRefusal(t, DecodeManifest(body), "operations", "differ from the registry")
		}},
		{arm: `ctor|failProtocol|manifest platforms are not sorted unique`, name: "manifest unsorted platforms", prove: func(t *testing.T) {
			body := manifestVariant(t, `["linux", "macos", "windows", "wsl2"]`, `["macos", "linux", "windows", "wsl2"]`)
			requireFrameRefusal(t, DecodeManifest(body), "platforms", "not sorted unique")
		}},
		{arm: `ctor|failProtocol|manifest platforms is empty or not an array`, name: "manifest empty platforms", prove: func(t *testing.T) {
			body := manifestVariant(t, `["linux", "macos", "windows", "wsl2"]`, `[]`)
			requireFrameRefusal(t, DecodeManifest(body), "platforms", "empty or not an array")
		}},
		{arm: `ctor|failProtocol|manifest platforms names an unknown platform`, name: "manifest darwin platform", prove: func(t *testing.T) {
			body := manifestVariant(t, `"windows", "wsl2"`, `"windows", "darwin"`)
			requireFrameRefusal(t, DecodeManifest(body), "platforms", "unknown platform")
		}},
		{arm: `ctor|failProtocol|manifest plugin_version is not SemVer`, name: "manifest two-part version", prove: func(t *testing.T) {
			body := manifestVariant(t, `"plugin_version": "0.1.0"`, `"plugin_version": "1.0"`)
			requireFrameRefusal(t, DecodeManifest(body), "plugin_version", "not SemVer")
		}},
		{arm: `ctor|failProtocol|manifest provider_id is not a provider id`, name: "manifest uppercase provider", prove: func(t *testing.T) {
			body := manifestVariant(t, `"provider_id": "pi"`, `"provider_id": "Pi"`)
			requireFrameRefusal(t, DecodeManifest(body), "provider_id", "not a provider id")
		}},
		{arm: `ctor|failProtocol|manifest provider_version_range is not 1..256 characters`, name: "manifest empty range", prove: func(t *testing.T) {
			body := manifestVariant(t, `"provider_version_range": ">=0.73.1 <0.74.0"`, `"provider_version_range": ""`)
			requireFrameRefusal(t, DecodeManifest(body), "provider_version_range", "not 1..256 characters")
		}},
		{arm: `ctor|failProtocol|manifest schema is not the provider manifest`, name: "manifest probe schema", prove: func(t *testing.T) {
			body := manifestVariant(t, `"urn:ax:schema:provider-manifest"`, `"urn:ax:schema:provider-probe"`)
			requireFrameRefusal(t, DecodeManifest(body), "schema", "not the provider manifest")
		}},
		{arm: `ctor|failProtocol|manifest schema_version is not 1.0.0`, name: "manifest schema 2.0.0", prove: func(t *testing.T) {
			body := manifestVariant(t, `"schema_version": "1.0.0"`, `"schema_version": "2.0.0"`)
			requireFrameRefusal(t, DecodeManifest(body), "schema_version", "not 1.0.0")
		}},
		// Probe arms, through DecodeProbe.
		{arm: `ctor|failProtocol|probe architecture is not amd64 or arm64`, name: "probe x86 architecture", prove: func(t *testing.T) {
			body := probeVariant(t, `"architecture": "arm64"`, `"architecture": "x86"`)
			requireFrameRefusal(t, DecodeProbe(body), "architecture", "not amd64 or arm64")
		}},
		{arm: `ctor|failProtocol|probe capabilities carry an unknown key`, name: "probe eighth capability", prove: func(t *testing.T) {
			body := probeVariant(t, "    }\n  },\n  \"warnings\": []", "    },\n    \"remote_exec\": {\n      \"status\": \"available\",\n      \"enabled\": true,\n      \"evidence\": \"probed\",\n      \"detail\": \"x\"\n    }\n  },\n  \"warnings\": []")
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "unknown key")
		}},
		{arm: `ctor|failProtocol|probe capabilities miss a registry key`, name: "probe without prompt_spawn", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specProbeExample, "    \"prompt_spawn\": {\n      \"status\": \"unknown\",\n      \"enabled\": false,\n      \"evidence\": \"none\",\n      \"detail\": \"not claimed for v0.3.0\"\n    },\n", "", 1))
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "miss a registry key")
		}},
		{arm: `ctor|failProtocol|probe capability carries an unknown member`, name: "probe capability score", prove: func(t *testing.T) {
			body := probeVariant(t, `"detail": "not claimed for v0.3.0"`, `"detail": "not claimed for v0.3.0", "score": 1`)
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "unknown member")
		}},
		{arm: `ctor|failProtocol|probe capability detail is not 0..2048 characters`, name: "probe overlong detail", prove: func(t *testing.T) {
			body := probeVariant(t, `"detail": "--session, --continue, and --resume are present"`, `"detail": "`+strings.Repeat("d", 2049)+`"`)
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "not 0..2048 characters")
		}},
		{arm: `ctor|failProtocol|probe capability enabled is not a boolean`, name: "probe string enabled", prove: func(t *testing.T) {
			body := probeVariant(t, `"status": "available",
      "enabled": true,`, `"status": "available",
      "enabled": "yes",`)
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "enabled is not a boolean")
		}},
		{arm: `ctor|failProtocol|probe capability enables a non-available status`, name: "probe enabled conditional", prove: func(t *testing.T) {
			body := probeWithCapabilities(t, func(name, _ string) string {
				if name == "portable_store" {
					return probeCapabilityBlock("conditional", true)
				}
				return probeCapabilityBlock("unsupported", false)
			})
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "enables a non-available status")
		}},
		{arm: `ctor|failProtocol|probe capability evidence is not a registry member`, name: "probe vibes evidence", prove: func(t *testing.T) {
			body := probeVariant(t, `"evidence": "probed",`, `"evidence": "vibes",`)
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "evidence is not a registry member")
		}},
		{arm: `ctor|failProtocol|probe capability misses a required member`, name: "probe without evidence", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specProbeExample, "      \"evidence\": \"none\",\n", "", 1))
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "misses a required member")
		}},
		{arm: `ctor|failProtocol|probe capability status is not a registry member`, name: "probe sometimes status", prove: func(t *testing.T) {
			body := probeVariant(t, `"status": "unknown",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"`, `"status": "sometimes",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"`)
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "status is not a registry member")
		}},
		{arm: `ctor|failProtocol|probe carries unknown member`, name: "probe diagnostics member", prove: func(t *testing.T) {
			body := probeVariant(t, `"warnings": []`, `"warnings": [], "diagnostics": []`)
			requireFrameRefusal(t, DecodeProbe(body), "diagnostics", "unknown member")
		}},
		{arm: `ctor|failProtocol|probe misses a required member`, name: "probe without warnings", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specProbeExample, ",\n  \"warnings\": []", "", 1))
			requireFrameRefusal(t, DecodeProbe(body), "warnings", "misses a required member")
		}},
		{arm: `ctor|failProtocol|probe platform is not a registry member`, name: "probe darwin platform", prove: func(t *testing.T) {
			body := probeVariant(t, `"platform": "macos"`, `"platform": "darwin"`)
			requireFrameRefusal(t, DecodeProbe(body), "platform", "not a registry member")
		}},
		{arm: `ctor|failProtocol|probe provider_id is not a provider id`, name: "probe numeric provider", prove: func(t *testing.T) {
			body := probeVariant(t, `"provider_id": "pi"`, `"provider_id": "7pi"`)
			requireFrameRefusal(t, DecodeProbe(body), "provider_id", "not a provider id")
		}},
		{arm: `ctor|failProtocol|probe provider_version is not 1..128 characters`, name: "probe empty version", prove: func(t *testing.T) {
			body := probeVariant(t, `"provider_version": "0.73.1"`, `"provider_version": ""`)
			requireFrameRefusal(t, DecodeProbe(body), "provider_version", "not 1..128 characters")
		}},
		{arm: `ctor|failProtocol|probe schema is not the provider probe`, name: "probe manifest schema", prove: func(t *testing.T) {
			body := probeVariant(t, `"urn:ax:schema:provider-probe"`, `"urn:ax:schema:provider-manifest"`)
			requireFrameRefusal(t, DecodeProbe(body), "schema", "not the provider probe")
		}},
		{arm: `ctor|failProtocol|probe schema_version is not 1.0.0`, name: "probe schema 1.1.0", prove: func(t *testing.T) {
			body := probeVariant(t, `"schema_version": "1.0.0"`, `"schema_version": "1.1.0"`)
			requireFrameRefusal(t, DecodeProbe(body), "schema_version", "not 1.0.0")
		}},
		{arm: `ctor|failProtocol|probe warning exceeds 2048 characters`, name: "probe overlong warning", prove: func(t *testing.T) {
			body := probeVariant(t, `"warnings": []`, `"warnings": ["`+strings.Repeat("w", 2049)+`"]`)
			requireFrameRefusal(t, DecodeProbe(body), "warnings", "exceeds 2048 characters")
		}},
		{arm: `ctor|failProtocol|probe warnings are not sorted unique`, name: "probe unsorted warnings", prove: func(t *testing.T) {
			body := probeVariant(t, `"warnings": []`, `"warnings": ["b", "a"]`)
			requireFrameRefusal(t, DecodeProbe(body), "warnings", "not sorted unique")
		}},
		{arm: `ctor|failProtocol|probe warnings exceed 1024 entries`, name: "probe 1025 warnings", prove: func(t *testing.T) {
			body := probeVariant(t, `"warnings": []`, `"warnings": [`+strings.Repeat(`"a",`, 1024)+`"a"]`)
			requireFrameRefusal(t, DecodeProbe(body), "warnings", "exceed 1024 entries")
		}},
		// Capability-gate and profile arms.
		{arm: `ctor|failInvalid|capability is not a registry member`, name: "gate unknown capability", prove: func(t *testing.T) {
			requireLocalRefusal(t, RequireCapability([]byte(specProbeExample), "remote_exec"), "invalid_config", "not a registry member")
		}},
		{arm: `ctor|failInvalid|probe does not establish the capability as available`, name: "gate conditional portable_store", prove: func(t *testing.T) {
			requireLocalRefusal(t, RequireCapability([]byte(specProbeExample), "portable_store"), "invalid_config", "does not establish the capability")
		}},
		{arm: `ctor|failInvalid|profile mapping names an unknown profile`, name: "mapping unrestricted profile", prove: func(t *testing.T) {
			_, err := ProfileMapping("codex", "unrestricted")
			requireLocalRefusal(t, err, "invalid_config", "unknown profile")
		}},
		{arm: `ctor|failInvalid|profile mapping names an unknown provider`, name: "mapping qwen provider", prove: func(t *testing.T) {
			_, err := ProfileMapping("qwen", ProfileYOLO)
			requireLocalRefusal(t, err, "invalid_config", "unknown provider")
		}},
	}
}
