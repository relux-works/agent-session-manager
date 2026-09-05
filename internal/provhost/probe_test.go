package provhost

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// specProbeExample is the Section 7.4 probe example, copied verbatim
// from the pinned document. It exercises all four statuses across
// the seven capabilities: available, conditional, unsupported, and
// unknown, each with enabled set exactly as the contract allows.
const specProbeExample = `{
  "schema": "urn:ax:schema:provider-probe",
  "schema_version": "1.0.0",
  "provider_id": "pi",
  "provider_version": "0.73.1",
  "platform": "macos",
  "architecture": "arm64",
  "capabilities": {
    "native_resume": {
      "status": "available",
      "enabled": true,
      "evidence": "probed",
      "detail": "--session, --continue, and --resume are present"
    },
    "portable_store": {
      "status": "conditional",
      "enabled": false,
      "evidence": "acceptance_required",
      "detail": "closed-store cross-host fixture has not passed"
    },
    "managed_pty": {
      "status": "conditional",
      "enabled": false,
      "evidence": "acceptance_required",
      "detail": "platform PTY interruption and flush gate has not passed"
    },
    "appserver": {
      "status": "unsupported",
      "enabled": false,
      "evidence": "provider_contract",
      "detail": "Pi RPC is not claimed as the ax appserver capability"
    },
    "task_board_primary": {
      "status": "unknown",
      "enabled": false,
      "evidence": "none",
      "detail": "no reliable primary adapter is accepted"
    },
    "prompt_spawn": {
      "status": "unknown",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"
    },
    "native_goal_binding": {
      "status": "unsupported",
      "enabled": false,
      "evidence": "provider_contract",
      "detail": "no native task-board goal binding"
    }
  },
  "warnings": []
}`

// TestSpecProbeExampleDecodes proves the Section 7.4 example passes
// the production entry point with every status represented.
func TestSpecProbeExampleDecodes(t *testing.T) {
	if err := DecodeProbe([]byte(specProbeExample)); err != nil {
		t.Fatalf("DecodeProbe(spec example): %v", err)
	}
}

// probeVariant rewrites one unique substring of the spec example.
func probeVariant(t *testing.T, old, new string) []byte {
	t.Helper()
	if strings.Count(specProbeExample, old) != 1 {
		t.Fatalf("probe variant anchor %q is not unique", old)
	}
	return []byte(strings.Replace(specProbeExample, old, new, 1))
}

// probeCapabilityBlock renders one capability value with the given
// status and enabled bit, for the per-key sweep.
func probeCapabilityBlock(status string, enabled bool) string {
	return `{"status": "` + status + `", "enabled": ` + map[bool]string{true: "true", false: "false"}[enabled] + `, "evidence": "probed", "detail": "sweep"}`
}

// probeWithCapabilities builds a probe whose capabilities object is
// exactly the seven registry keys mapped through render. The domain
// is derived from Capabilities(), never from a hand-picked sample:
// every registry key answers to every rule.
func probeWithCapabilities(t *testing.T, render func(name, exampleStatus string) string) []byte {
	t.Helper()
	example, fault := decodeStrictObject([]byte(specProbeExample))
	if fault != nil {
		t.Fatalf("spec probe example is not a strict object: %v", fault.detail)
	}
	capabilities, fault := decodeStrictObject(example["capabilities"])
	if fault != nil {
		t.Fatalf("spec probe capabilities are not a strict object: %v", fault.detail)
	}
	var names []string
	for _, name := range Capabilities() {
		raw, present := capabilities[name]
		if !present {
			t.Fatalf("spec probe example misses capability %q", name)
		}
		var status struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(raw, &status); err != nil {
			t.Fatalf("spec probe capability %q status: %v", name, err)
		}
		names = append(names, `"`+name+`": `+render(name, status.Status))
	}
	start := strings.Index(specProbeExample, `"native_resume": {`)
	end := strings.Index(specProbeExample, `  },
  "warnings"`)
	if start < 0 || end <= start {
		t.Fatal("spec probe example capabilities block not found; the sweep is blind")
	}
	return []byte(specProbeExample[:start] + strings.Join(names, ",\n    ") + "\n    " + specProbeExample[end:])
}

// TestDecodeProbeRefusals drives every probe rule with a fixture
// carrying only that violation.
func TestDecodeProbeRefusals(t *testing.T) {
	long2048 := strings.Repeat("w", 2048)
	long2049 := strings.Repeat("w", 2049)
	long128 := strings.Repeat("v", 128)
	long129 := strings.Repeat("v", 129)
	rows := []struct {
		name   string
		body   []byte
		member string
		detail string
	}{
		{"unknown member", probeVariant(t, `"warnings": []`, `"warnings": [], "diagnostics": []`), "diagnostics", "unknown member"},
		{"missing member", []byte(strings.Replace(specProbeExample, `,
  "warnings": []`, ``, 1)), "warnings", "misses a required member"},
		{"wrong schema", probeVariant(t, `"urn:ax:schema:provider-probe"`, `"urn:ax:schema:provider-manifest"`), "schema", "not the provider probe"},
		{"wrong schema version", probeVariant(t, `"schema_version": "1.0.0"`, `"schema_version": "1.1.0"`), "schema_version", "not 1.0.0"},
		{"bad provider id", probeVariant(t, `"provider_id": "pi"`, `"provider_id": "7pi"`), "provider_id", "not a provider id"},
		{"empty provider version", probeVariant(t, `"provider_version": "0.73.1"`, `"provider_version": ""`), "provider_version", "not 1..128 characters"},
		{"provider version 129", probeVariant(t, `"provider_version": "0.73.1"`, `"provider_version": "`+long129+`"`), "provider_version", "not 1..128 characters"},
		{"bad platform", probeVariant(t, `"platform": "macos"`, `"platform": "darwin"`), "platform", "not a registry member"},
		{"bad architecture", probeVariant(t, `"architecture": "arm64"`, `"architecture": "x86"`), "architecture", "not amd64 or arm64"},
		{"capabilities missing key", []byte(strings.Replace(specProbeExample, `    "prompt_spawn": {
      "status": "unknown",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"
    },
`, ``, 1)), "capabilities", "miss a registry key"},
		{"capabilities missing first key", []byte(strings.Replace(specProbeExample, `    "native_resume": {
      "status": "available",
      "enabled": true,
      "evidence": "probed",
      "detail": "--session, --continue, and --resume are present"
    },
`, ``, 1)), "capabilities", "miss a registry key"},
		{"capabilities missing last key", []byte(strings.Replace(specProbeExample, `,
    "native_goal_binding": {
      "status": "unsupported",
      "enabled": false,
      "evidence": "provider_contract",
      "detail": "no native task-board goal binding"
    }`, ``, 1)), "capabilities", "miss a registry key"},
		{"capabilities unknown key", probeVariant(t, "    }\n  },\n  \"warnings\": []", "    },\n    \"remote_exec\": {\n      \"status\": \"available\",\n      \"enabled\": true,\n      \"evidence\": \"probed\",\n      \"detail\": \"x\"\n    }\n  },\n  \"warnings\": []"), "capabilities", "unknown key"},
		{"capability unknown member", probeVariant(t, `"detail": "not claimed for v0.3.0"`, `"detail": "not claimed for v0.3.0", "score": 1`), "capabilities", "unknown member"},
		{"capability missing member", []byte(strings.Replace(specProbeExample, `      "evidence": "none",
`, ``, 1)), "capabilities", "misses a required member"},
		{"capability bad status", probeVariant(t, `"status": "unknown",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"`, `"status": "sometimes",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"`), "capabilities", "status is not a registry member"},
		{"capability non-boolean enabled", probeVariant(t, `"status": "available",
      "enabled": true,`, `"status": "available",
      "enabled": "yes",`), "capabilities", "enabled is not a boolean"},
		{"capability bad evidence", probeVariant(t, `"evidence": "probed",`, `"evidence": "vibes",`), "capabilities", "evidence is not a registry member"},
		{"capability long detail", probeVariant(t, `"detail": "--session, --continue, and --resume are present"`, `"detail": "`+long2049+`"`), "capabilities", "not 0..2048 characters"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := DecodeProbe(row.body)
			requireFrameRefusal(t, err, row.member, row.detail)
		})
	}
	if err := DecodeProbe(probeVariant(t, `"detail": "--session, --continue, and --resume are present"`, `"detail": "`+long2048+`"`)); err != nil {
		t.Fatalf("2048-character detail refused: %v", err)
	}
	if err := DecodeProbe(probeVariant(t, `"provider_version": "0.73.1"`, `"provider_version": "`+long128+`"`)); err != nil {
		t.Fatalf("128-character provider version refused: %v", err)
	}
}

// TestDecodeProbeWarnings proves the warning-array bounds exactly:
// 1,024 sorted unique entries pass, 1,025 are refused, and order and
// duplication are refused independently of length.
func TestDecodeProbeWarnings(t *testing.T) {
	sorted := make([]string, 0, 1024)
	for i := 0; i < 1024; i++ {
		sorted = append(sorted, fmt.Sprintf("w-%04d", i))
	}
	encode := func(warnings []string) []byte {
		raw, err := json.Marshal(warnings)
		if err != nil {
			t.Fatalf("marshal warnings: %v", err)
		}
		return probeVariant(t, `"warnings": []`, `"warnings": `+string(raw))
	}
	if err := DecodeProbe(encode(sorted)); err != nil {
		t.Fatalf("1024 sorted warnings refused: %v", err)
	}
	requireFrameRefusal(t, DecodeProbe(encode(append(sorted, "w-last"))), "warnings", "exceed 1024 entries")
	requireFrameRefusal(t, DecodeProbe(encode([]string{"b", "a"})), "warnings", "not sorted unique")
	requireFrameRefusal(t, DecodeProbe(encode([]string{"a", "a"})), "warnings", "not sorted unique")
	requireFrameRefusal(t, DecodeProbe(probeVariant(t, `"warnings": []`, `"warnings": ["`+strings.Repeat("w", 2049)+`"]`)), "warnings", "exceeds 2048 characters")
}

// TestDecodeProbeEnabledRequiresAvailable sweeps the full
// status-by-enabled domain for one capability: only
// available+true validates. The sweep is the complement the
// sibling story's harness missed by sampling: 4 statuses times 2
// enabled values, all 8 measured.
func TestDecodeProbeEnabledRequiresAvailable(t *testing.T) {
	for _, status := range []string{"available", "conditional", "unsupported", "unknown"} {
		for _, enabled := range []bool{false, true} {
			status, enabled := status, enabled
			name := fmt.Sprintf("%s enabled=%v", status, enabled)
			t.Run(name, func(t *testing.T) {
				body := probeWithCapabilities(t, func(name, _ string) string {
					if name == "native_resume" {
						return probeCapabilityBlock(status, enabled)
					}
					return probeCapabilityBlock("unsupported", false)
				})
				err := DecodeProbe(body)
				if status == "available" && enabled {
					if err != nil {
						t.Fatalf("available+enabled refused: %v", err)
					}
					return
				}
				if status != "available" && enabled {
					requireFrameRefusal(t, err, "capabilities", "enables a non-available status")
					return
				}
				if err != nil {
					t.Fatalf("%s enabled=%v refused: %v", status, enabled, err)
				}
			})
		}
	}
}

// TestDecodeProbeSweepsEveryCapabilityKey proves the capability
// rules answer per key, not per sample: each registry key carries
// the enabled-non-available violation in turn, and every turn is
// refused. A validation narrowed to the keys the hand-written
// fixtures touch would pass six of these seven turns.
func TestDecodeProbeSweepsEveryCapabilityKey(t *testing.T) {
	for _, key := range Capabilities() {
		key := key
		t.Run(key, func(t *testing.T) {
			body := probeWithCapabilities(t, func(name, _ string) string {
				if name == key {
					return probeCapabilityBlock("conditional", true)
				}
				return probeCapabilityBlock("unsupported", false)
			})
			requireFrameRefusal(t, DecodeProbe(body), "capabilities", "enables a non-available status")
		})
	}
	t.Logf("capability key coverage: %d/%d registry keys swept", len(Capabilities()), len(Capabilities()))
}

// TestCapabilityUsableIsDerivedFromStatus sweeps the pure gate over
// the same 8-cell domain: usable exactly for available+true.
func TestCapabilityUsableIsDerivedFromStatus(t *testing.T) {
	for _, status := range []string{"available", "conditional", "unsupported", "unknown"} {
		for _, enabled := range []bool{false, true} {
			want := status == "available" && enabled
			if got := CapabilityUsable(status, enabled); got != want {
				t.Fatalf("CapabilityUsable(%q, %v) = %v, want %v", status, enabled, got, want)
			}
		}
	}
}

// TestRequireCapabilityRefusesUnproven Surfaces proves the host-side
// gate: the spec example establishes native_resume only, so every
// other registry key is refused without starting a process, and an
// unknown name is refused as a caller error.
func TestRequireCapabilityRefusesUnprovenSurfaces(t *testing.T) {
	if err := RequireCapability([]byte(specProbeExample), "native_resume"); err != nil {
		t.Fatalf("RequireCapability(native_resume): %v", err)
	}
	refused := 0
	for _, name := range Capabilities() {
		if name == "native_resume" {
			continue
		}
		err := RequireCapability([]byte(specProbeExample), name)
		requireLocalRefusal(t, err, "invalid_config", "does not establish the capability")
		refused++
	}
	if refused != len(Capabilities())-1 {
		t.Fatalf("refused %d capabilities, want %d", refused, len(Capabilities())-1)
	}
	requireLocalRefusal(t, RequireCapability([]byte(specProbeExample), "remote_exec"), "invalid_config", "not a registry member")
	requireFrameRefusal(t, RequireCapability([]byte(`{"schema":"x"}`), "native_resume"), "schema_version", "misses a required member")
	t.Logf("capability gate coverage: 1 usable, %d refused of %d registry keys", refused, len(Capabilities()))
}
