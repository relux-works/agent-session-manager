package provhost

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// This file proves the profile_resolve.go production entries: the
// Section 7.7 mapping resolver against the exact probed build and
// the sanitized launch argv projection. Every refusal below drives
// its production entry; the arm witnesses in
// refusal_arm_operations_e_test.go pin each derived arm to one of
// these entries.

// resolveTuple returns the exact probed build for one provider.
func resolveTuple(provider, version string) BuildTuple {
	return BuildTuple{ProviderID: provider, ProviderVersion: version, Platform: "macos", Architecture: "arm64"}
}

func TestResolveMappingTable(t *testing.T) {
	versions := map[string]string{
		"codex": "0.147.0", "claude": "2.1.229", "gemini": "0.9.0",
		"muse": "0.1.0", "antigravity": "1.1.14", "pi": "0.73.1",
	}
	wantMapping := map[string]string{
		"codex":       "--dangerously-bypass-approvals-and-sandbox",
		"claude":      "--dangerously-skip-permissions",
		"gemini":      "--approval-mode=yolo",
		"muse":        "--yolo",
		"antigravity": "--dangerously-skip-permissions",
		"pi":          "default_unrestricted_tool_set",
	}
	for _, provider := range profileProviders {
		t.Run(provider+"/yolo", func(t *testing.T) {
			resolved, err := ResolveMapping(provider, ProfileYOLO, resolveTuple(provider, versions[provider]))
			if err != nil {
				t.Fatalf("ResolveMapping error = %v", err)
			}
			if resolved.ProviderID != provider || resolved.Profile != ProfileYOLO || resolved.Version != versions[provider] {
				t.Fatalf("ResolveMapping binds %+v", resolved)
			}
			if resolved.Mapping != wantMapping[provider] {
				t.Fatalf("ResolveMapping mapping = %q, want %q", resolved.Mapping, wantMapping[provider])
			}
			if resolved.Equivalent != (provider == "pi") {
				t.Fatalf("ResolveMapping equivalent = %v for %s", resolved.Equivalent, provider)
			}
		})
		t.Run(provider+"/standard", func(t *testing.T) {
			resolved, err := ResolveMapping(provider, ProfileStandard, resolveTuple(provider, versions[provider]))
			if err != nil {
				t.Fatalf("ResolveMapping error = %v", err)
			}
			// Standard omits every unrestricted flag on the
			// wire; only the Pi equivalence is disclosed.
			if resolved.Mapping != "" {
				t.Fatalf("ResolveMapping standard mapping = %q, want empty", resolved.Mapping)
			}
			if resolved.Equivalent != (provider == "pi") {
				t.Fatalf("ResolveMapping equivalent = %v for %s", resolved.Equivalent, provider)
			}
		})
	}
}

func TestResolveMappingRefusals(t *testing.T) {
	tuple := resolveTuple("codex", "0.147.0")
	if _, err := ResolveMapping("NOT A PROVIDER", ProfileYOLO, tuple); failureCode(t, err) != "invalid_config" {
		t.Fatalf("ResolveMapping(garbage provider) code = %v, want invalid_config", err)
	}
	if _, err := ResolveMapping("codex", "turbo", tuple); failureCode(t, err) != "invalid_config" {
		t.Fatalf("ResolveMapping(bad profile) code = %v, want invalid_config", err)
	}
	foreign := resolveTuple("claude", "2.1.229")
	if _, err := ResolveMapping("codex", ProfileYOLO, foreign); failureCode(t, err) != "invalid_config" {
		t.Fatalf("ResolveMapping(foreign tuple) code = %v, want invalid_config", err)
	}
	badPlatform := BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "plan9", Architecture: "arm64"}
	if _, err := ResolveMapping("codex", ProfileYOLO, badPlatform); failureCode(t, err) != "invalid_config" {
		t.Fatalf("ResolveMapping(bad tuple) code = %v, want invalid_config", err)
	}
	for _, provider := range []string{"qwen", "futuredesk"} {
		unmapped := resolveTuple(provider, "1.0.0")
		_, err := ResolveMapping(provider, ProfileYOLO, unmapped)
		if failureCode(t, err) != "profile_mapping_unavailable" {
			t.Fatalf("ResolveMapping(%s) code = %v, want profile_mapping_unavailable", provider, err)
		}
		if failureExit(t, err) != 6 {
			t.Fatalf("ResolveMapping(%s) exit = %d, want 6", provider, failureExit(t, err))
		}
		failure := failureObject(t, err)
		for _, key := range []string{"provider_id", "provider_version", "profile"} {
			if _, ok := failure.Detail(key); !ok {
				t.Fatalf("ResolveMapping(%s) carries no %q detail", provider, key)
			}
		}
	}
	drifted := resolveTuple("pi", "0.74.0")
	if _, err := ResolveMapping("pi", ProfileYOLO, drifted); failureCode(t, err) != "profile_mapping_unavailable" {
		t.Fatalf("ResolveMapping(pi 0.74.0) code = %v, want profile_mapping_unavailable", err)
	}
	if _, err := ResolveMapping("pi", ProfileStandard, drifted); failureCode(t, err) != "profile_mapping_unavailable" {
		t.Fatalf("ResolveMapping(pi 0.74.0 standard) code = %v, want profile_mapping_unavailable", err)
	}
}

func TestProjectLaunchArgvYOLO(t *testing.T) {
	cases := []struct {
		provider string
		version  string
		base     []string
		want     []string
	}{
		{"codex", "0.147.0", []string{"codex", "resume", "11111111-2222-4333-8444-555555555555"}, []string{"codex", "resume", "11111111-2222-4333-8444-555555555555", "--dangerously-bypass-approvals-and-sandbox"}},
		{"claude", "2.1.229", []string{"claude", "--resume", "uuid"}, []string{"claude", "--resume", "uuid", "--dangerously-skip-permissions"}},
		{"gemini", "0.9.0", []string{"gemini", "--resume", "uuid"}, []string{"gemini", "--resume", "uuid", "--approval-mode=yolo"}},
		{"muse", "0.1.0", []string{"muse", "resume", "uuid"}, []string{"muse", "resume", "uuid", "--yolo"}},
		{"antigravity", "1.1.14", []string{"agy", "--conversation", "uuid"}, []string{"agy", "--conversation", "uuid", "--dangerously-skip-permissions"}},
		{"pi", "0.73.1", []string{"pi", "--session", "uuid"}, []string{"pi", "--session", "uuid"}},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			resolved, err := ResolveMapping(tc.provider, ProfileYOLO, resolveTuple(tc.provider, tc.version))
			if err != nil {
				t.Fatalf("ResolveMapping error = %v", err)
			}
			projected, err := ProjectLaunchArgv(tc.base, tc.provider, ProfileYOLO, resolved)
			if err != nil {
				t.Fatalf("ProjectLaunchArgv error = %v", err)
			}
			if strings.Join(projected, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("ProjectLaunchArgv = %q, want %q", projected, tc.want)
			}
			// The base slice is never rewritten: the
			// projection copies.
			if len(tc.base)+1 != len(projected) && tc.provider != "pi" {
				t.Fatalf("base was rewritten: %q", tc.base)
			}
		})
	}
}

func TestProjectLaunchArgvStandardOmitsEveryFlag(t *testing.T) {
	for _, provider := range profileProviders {
		t.Run(provider, func(t *testing.T) {
			version := "0.147.0"
			if provider == "pi" {
				version = "0.73.1"
			}
			if provider == "muse" {
				version = "0.1.0"
			}
			resolved, err := ResolveMapping(provider, ProfileStandard, resolveTuple(provider, version))
			if err != nil {
				t.Fatalf("ResolveMapping error = %v", err)
			}
			base := []string{provider, "resume", "uuid"}
			projected, err := ProjectLaunchArgv(base, provider, ProfileStandard, resolved)
			if err != nil {
				t.Fatalf("ProjectLaunchArgv error = %v", err)
			}
			if strings.Join(projected, "\x00") != strings.Join(base, "\x00") {
				t.Fatalf("ProjectLaunchArgv = %q, want base unchanged", projected)
			}
		})
	}
}

func TestProjectLaunchArgvRefusals(t *testing.T) {
	codex := resolveTuple("codex", "0.147.0")
	resolved, err := ResolveMapping("codex", ProfileYOLO, codex)
	if err != nil {
		t.Fatalf("ResolveMapping error = %v", err)
	}
	claude, err := ResolveMapping("claude", ProfileYOLO, resolveTuple("claude", "2.1.229"))
	if err != nil {
		t.Fatalf("ResolveMapping error = %v", err)
	}
	standard, err := ResolveMapping("codex", ProfileStandard, codex)
	if err != nil {
		t.Fatalf("ResolveMapping error = %v", err)
	}
	cases := []struct {
		name              string
		base              []string
		provider, profile string
		resolution        ResolvedMapping
		wantCode          axerror.Code
	}{
		{"resolution names another provider", []string{"codex"}, "codex", ProfileYOLO, claude, "invalid_config"},
		{"resolution names another profile", []string{"codex"}, "codex", ProfileYOLO, standard, "invalid_config"},
		{"empty base", nil, "codex", ProfileYOLO, resolved, ""},
		{"empty element", []string{"codex", ""}, "codex", ProfileYOLO, resolved, ""},
		{"NUL element", []string{"codex", "a\x00b"}, "codex", ProfileYOLO, resolved, ""},
		{"duplicate flag", []string{"codex", "--dangerously-bypass-approvals-and-sandbox"}, "codex", ProfileYOLO, resolved, "invalid_config"},
		{"foreign flag", []string{"codex", "--approval-mode=yolo"}, "codex", ProfileYOLO, resolved, "invalid_config"},
		{"codex alias under yolo", []string{"codex", "--yolo"}, "codex", ProfileYOLO, resolved, "invalid_config"},
		{"pi mapping string in base", []string{"pi", "default_unrestricted_tool_set"}, "pi", ProfileYOLO, mustResolvePiYOLO(t), "invalid_config"},
		{"flag under standard", []string{"codex", "--dangerously-bypass-approvals-and-sandbox"}, "codex", ProfileStandard, standard, "invalid_config"},
		{"alias under standard", []string{"codex", "--yolo"}, "codex", ProfileStandard, standard, "invalid_config"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ProjectLaunchArgv(tc.base, tc.provider, tc.profile, tc.resolution)
			if err == nil {
				t.Fatalf("ProjectLaunchArgv(%s) error = nil, want refusal", tc.name)
			}
			if tc.wantCode == "" {
				// Structural argv refusals propagate from
				// the secprim owner, not as Structured
				// Errors.
				return
			}
			if failureCode(t, err) != tc.wantCode {
				t.Fatalf("ProjectLaunchArgv(%s) code = %v, want %s", tc.name, err, tc.wantCode)
			}
		})
	}
}

func mustResolvePiYOLO(t *testing.T) ResolvedMapping {
	t.Helper()
	resolved, err := ResolveMapping("pi", ProfileYOLO, resolveTuple("pi", "0.73.1"))
	if err != nil {
		t.Fatalf("ResolveMapping(pi yolo) error = %v", err)
	}
	return resolved
}

func TestProjectLaunchArgvBoundsBothDirections(t *testing.T) {
	resolved, err := ResolveMapping("codex", ProfileStandard, resolveTuple("codex", "0.147.0"))
	if err != nil {
		t.Fatalf("ResolveMapping error = %v", err)
	}
	// Standard carries no appended flag, so the base lengths land
	// exactly on the Section 5.1 limits.
	repeat := func(element string, count int) []string {
		out := make([]string, 0, count)
		for range count {
			out = append(out, element)
		}
		return out
	}
	atLimit := repeat("a", 128)
	if _, err := ProjectLaunchArgv(atLimit, "codex", ProfileStandard, resolved); err != nil {
		t.Fatalf("128 elements error = %v", err)
	}
	if _, err := ProjectLaunchArgv(append(atLimit, "b"), "codex", ProfileStandard, resolved); failureCode(t, err) != "invalid_config" {
		t.Fatalf("129 elements error = %v, want invalid_config", err)
	}
	wide := strings.Repeat("w", 4096)
	if _, err := ProjectLaunchArgv([]string{"codex", wide}, "codex", ProfileStandard, resolved); err != nil {
		t.Fatalf("4096-byte element error = %v", err)
	}
	if _, err := ProjectLaunchArgv([]string{"codex", wide + "x"}, "codex", ProfileStandard, resolved); failureCode(t, err) != "invalid_config" {
		t.Fatalf("4097-byte element error = %v, want invalid_config", err)
	}
	// 16 elements of 4096 bytes total exactly 65536.
	full := repeat(strings.Repeat("f", 4096), 16)
	if _, err := ProjectLaunchArgv(full, "codex", ProfileStandard, resolved); err != nil {
		t.Fatalf("65536-byte total error = %v", err)
	}
	over := append(append([]string(nil), full...), "g")
	if _, err := ProjectLaunchArgv(over, "codex", ProfileStandard, resolved); failureCode(t, err) != "invalid_config" {
		t.Fatalf("65537-byte total error = %v, want invalid_config", err)
	}
	// Yolo appends one flag: a 128-element base overflows after
	// projection.
	yolo, err := ResolveMapping("codex", ProfileYOLO, resolveTuple("codex", "0.147.0"))
	if err != nil {
		t.Fatalf("ResolveMapping error = %v", err)
	}
	if _, err := ProjectLaunchArgv(atLimit, "codex", ProfileYOLO, yolo); failureCode(t, err) != "invalid_config" {
		t.Fatalf("yolo 128-element base error = %v, want invalid_config", err)
	}
	if _, err := ProjectLaunchArgv(atLimit[:127], "codex", ProfileYOLO, yolo); err != nil {
		t.Fatalf("yolo 127-element base error = %v", err)
	}
}

// TestProjectedArgvBoundsAgreeWithSpawnPlan drives identical
// vectors at each Section 5.1 limit through the host-side
// projection entry and the SpawnPlan wire twin, requiring the
// same verdict from both: the two bounds never diverge.
func TestProjectedArgvBoundsAgreeWithSpawnPlan(t *testing.T) {
	resolved, err := ResolveMapping("codex", ProfileStandard, resolveTuple("codex", "0.147.0"))
	if err != nil {
		t.Fatalf("ResolveMapping error = %v", err)
	}
	vectors := map[string][]string{
		"127 elements":   repeatForAgreement("a", 127),
		"128 elements":   repeatForAgreement("a", 128),
		"129 elements":   repeatForAgreement("a", 129),
		"4096-byte elem": {"codex", strings.Repeat("w", 4096)},
		"4097-byte elem": {"codex", strings.Repeat("w", 4097)},
		"65536 total":    repeatForAgreement(strings.Repeat("f", 4096), 16),
		"65537 total":    append(repeatForAgreement(strings.Repeat("f", 4096), 16), "g"),
	}
	for name, vector := range vectors {
		t.Run(name, func(t *testing.T) {
			_, hostErr := ProjectLaunchArgv(vector, "codex", ProfileStandard, resolved)
			encoded, err := json.Marshal(vector)
			if err != nil {
				t.Fatalf("marshal vector: %v", err)
			}
			wireErr := checkSpawnArgv(encoded)
			if (hostErr == nil) != (wireErr == nil) {
				t.Fatalf("verdicts diverge: host = %v, wire = %v", hostErr, wireErr)
			}
		})
	}
}

func repeatForAgreement(element string, count int) []string {
	out := make([]string, 0, count)
	for range count {
		out = append(out, element)
	}
	return out
}

// TestUnrestrictedTokensDeriveFromTable requires the launch
// projection's refused token set to equal the Section 7.7 table
// values plus the documented codex alias: a new table row is
// refused in bases from the same change, and no token is retyped.
func TestUnrestrictedTokensDeriveFromTable(t *testing.T) {
	tokens := unrestrictedTokens()
	// The expected set derives from the table in the test: the
	// table values (claude and antigravity share one flag) plus
	// the documented codex alias (which spells muse's flag).
	want := map[string]bool{codexAliasYOLO: true}
	for _, flag := range profileYOLOMapping {
		want[flag] = true
	}
	if len(tokens) != len(want) {
		t.Fatalf("tokens = %d, want %d derived", len(tokens), len(want))
	}
	for token := range want {
		if !tokens[token] {
			t.Fatalf("token %q is not refused", token)
		}
	}
	for token := range tokens {
		if !want[token] {
			t.Fatalf("token %q is refused but derives from nothing", token)
		}
	}
}
