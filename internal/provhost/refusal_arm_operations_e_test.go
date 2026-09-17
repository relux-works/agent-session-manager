package provhost

import (
	"strings"
	"testing"
)

// declaredOperationWitnessesProfileResolve proves the profile
// resolution arms: the Section 7.7 mapping resolver against the
// exact probed build and the sanitized launch argv projection. It
// extends declaredOperationWitnessesIdentityCreate in
// refusal_arm_operations_d_test.go; the split is file size only.
func declaredOperationWitnessesProfileResolve() []armWitness {
	codexTuple := func() BuildTuple {
		return BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	}
	codexYOLO := func(t *testing.T) ResolvedMapping {
		t.Helper()
		resolved, err := ResolveMapping("codex", ProfileYOLO, codexTuple())
		if err != nil {
			t.Fatalf("ResolveMapping(codex yolo) error = %v", err)
		}
		return resolved
	}
	codexStandard := func(t *testing.T) ResolvedMapping {
		t.Helper()
		resolved, err := ResolveMapping("codex", ProfileStandard, codexTuple())
		if err != nil {
			t.Fatalf("ResolveMapping(codex standard) error = %v", err)
		}
		return resolved
	}
	return []armWitness{
		// Resolver arms, through ResolveMapping.
		{arm: `ctor|failInvalid|mapping provider is not a provider id`, name: "mapping garbage provider", prove: func(t *testing.T) {
			_, err := ResolveMapping("NOT A PROVIDER", ProfileYOLO, codexTuple())
			requireLocalRefusal(t, err, "invalid_config", "mapping provider is not a provider id")
		}},
		{arm: `ctor|failMappingUnavailable|mapping has no row for this provider`, name: "mapping qwen has no row", prove: func(t *testing.T) {
			qwen := BuildTuple{ProviderID: "qwen", ProviderVersion: "1.0.0", Platform: "macos", Architecture: "arm64"}
			_, err := ResolveMapping("qwen", ProfileYOLO, qwen)
			requireLocalRefusal(t, err, "profile_mapping_unavailable", "mapping has no row for this provider")
		}},
		{arm: `ctor|failInvalid|mapping tuple names another provider`, name: "mapping claude tuple for codex", prove: func(t *testing.T) {
			claude := BuildTuple{ProviderID: "claude", ProviderVersion: "2.1.229", Platform: "macos", Architecture: "arm64"}
			_, err := ResolveMapping("codex", ProfileYOLO, claude)
			requireLocalRefusal(t, err, "invalid_config", "mapping tuple names another provider")
		}},
		{arm: `ctor|failInvalid|mapping profile is not standard or yolo`, name: "mapping turbo profile", prove: func(t *testing.T) {
			_, err := ResolveMapping("codex", "turbo", codexTuple())
			requireLocalRefusal(t, err, "invalid_config", "mapping profile is not standard or yolo")
		}},
		{arm: `ctor|failMappingUnavailable|mapping pins pi to probed 0.73.1`, name: "mapping pi 0.74.0", prove: func(t *testing.T) {
			pi := BuildTuple{ProviderID: "pi", ProviderVersion: "0.74.0", Platform: "macos", Architecture: "arm64"}
			_, err := ResolveMapping("pi", ProfileYOLO, pi)
			requireLocalRefusal(t, err, "profile_mapping_unavailable", "mapping pins pi to probed 0.73.1")
		}},
		// Projection arms, through ProjectLaunchArgv.
		{arm: `ctor|failInvalid|launch mapping does not match the provider profile`, name: "launch standard resolution for yolo", prove: func(t *testing.T) {
			_, err := ProjectLaunchArgv([]string{"codex"}, "codex", ProfileYOLO, codexStandard(t))
			requireLocalRefusal(t, err, "invalid_config", "launch mapping does not match the provider profile")
		}},
		{arm: `ctor|failInvalid|launch argv already carries the profile flag`, name: "launch duplicate flag", prove: func(t *testing.T) {
			_, err := ProjectLaunchArgv([]string{"codex", "--dangerously-bypass-approvals-and-sandbox"}, "codex", ProfileYOLO, codexYOLO(t))
			requireLocalRefusal(t, err, "invalid_config", "launch argv already carries the profile flag")
		}},
		{arm: `ctor|failInvalid|launch argv carries a changed unrestricted flag`, name: "launch foreign flag", prove: func(t *testing.T) {
			_, err := ProjectLaunchArgv([]string{"codex", "--approval-mode=yolo"}, "codex", ProfileYOLO, codexYOLO(t))
			requireLocalRefusal(t, err, "invalid_config", "launch argv carries a changed unrestricted flag")
		}},
		{arm: `ctor|failInvalid|launch argv reuses an unrestricted flag under standard`, name: "launch alias under standard", prove: func(t *testing.T) {
			_, err := ProjectLaunchArgv([]string{"codex", "--yolo"}, "codex", ProfileStandard, codexStandard(t))
			requireLocalRefusal(t, err, "invalid_config", "launch argv reuses an unrestricted flag under standard")
		}},
		{arm: `ctor|failInvalid|launch argv exceeds 128 elements`, name: "launch 129 elements", prove: func(t *testing.T) {
			base := make([]string, 0, 129)
			for range 129 {
				base = append(base, "a")
			}
			_, err := ProjectLaunchArgv(base, "codex", ProfileStandard, codexStandard(t))
			requireLocalRefusal(t, err, "invalid_config", "launch argv exceeds 128 elements")
		}},
		{arm: `ctor|failInvalid|launch argv element exceeds 4096 bytes`, name: "launch 4097-byte element", prove: func(t *testing.T) {
			_, err := ProjectLaunchArgv([]string{"codex", strings.Repeat("w", 4097)}, "codex", ProfileStandard, codexStandard(t))
			requireLocalRefusal(t, err, "invalid_config", "launch argv element exceeds 4096 bytes")
		}},
		{arm: `ctor|failInvalid|launch argv exceeds 65536 bytes total`, name: "launch 65537-byte total", prove: func(t *testing.T) {
			element := strings.Repeat("f", 4096)
			base := make([]string, 0, 17)
			for range 16 {
				base = append(base, element)
			}
			base = append(base, "g")
			_, err := ProjectLaunchArgv(base, "codex", ProfileStandard, codexStandard(t))
			requireLocalRefusal(t, err, "invalid_config", "launch argv exceeds 65536 bytes total")
		}},
	}
}
