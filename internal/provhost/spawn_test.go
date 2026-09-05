package provhost

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// specResumePlan is the normative resume success body (Section 7.5),
// copied verbatim from the pinned document: a codex yolo plan whose
// profile_mapping carries the exact Section 7.7 adapter flag.
const specResumePlan = `{
    "argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ],
    "cwd": "/srv/relux/payments-api/src",
    "env_names": ["OPENAI_API_KEY"],
    "env_literals": {},
    "native_session_id": "11111111-2222-4333-8444-555555555555",
    "profile_mapping": "--dangerously-bypass-approvals-and-sandbox",
    "extensions": {}
  }`

// TestSpecResumePlanDecodes proves the normative resume body passes
// the production entry point for its provider, profile, and
// platform, and that the embedded profile_mapping equals the
// Section 7.7 mapping rather than merely being present.
func TestSpecResumePlanDecodes(t *testing.T) {
	if err := DecodeSpawnPlan([]byte(specResumePlan), "codex", ProfileYOLO, scalar.PlatformLinux); err != nil {
		t.Fatalf("DecodeSpawnPlan(spec resume): %v", err)
	}
	mapping, err := ProfileMapping("codex", ProfileYOLO)
	if err != nil {
		t.Fatalf("ProfileMapping(codex, yolo): %v", err)
	}
	if !strings.Contains(specResumePlan, `"profile_mapping": "`+mapping+`"`) {
		t.Fatalf("spec resume profile_mapping does not carry the Section 7.7 mapping %q", mapping)
	}
}

// spawnVariant rewrites one unique substring of the normative plan.
func spawnVariant(t *testing.T, old, new string) []byte {
	t.Helper()
	if strings.Count(specResumePlan, old) != 1 {
		t.Fatalf("spawn variant anchor %q is not unique", old)
	}
	return []byte(strings.Replace(specResumePlan, old, new, 1))
}

// TestDecodeSpawnPlanRefusals drives every SpawnPlan rule.
func TestDecodeSpawnPlanRefusals(t *testing.T) {
	long4096 := strings.Repeat("a", 4096)
	long4097 := strings.Repeat("a", 4097)
	long512 := strings.Repeat("m", 512)
	long513 := strings.Repeat("m", 513)
	rows := []struct {
		name   string
		body   []byte
		member string
		detail string
	}{
		{"unknown member", spawnVariant(t, `"extensions": {}`, `"extensions": {}, "priority": 1`), "priority", "unknown member"},
		{"missing member", []byte(strings.Replace(specResumePlan, `,
    "extensions": {}`, ``, 1)), "extensions", "misses a required member"},
		{"argv empty", spawnVariant(t, `"argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ]`, `"argv": []`), "argv", "empty or longer than 128"},
		{"argv empty element", spawnVariant(t, `"resume",`, `"resume", "",`), "argv", "not 1..4096 NUL-free bytes"},
		{"argv long element", spawnVariant(t, `"resume",`, `"resume", "`+long4097+`",`), "argv", "not 1..4096 NUL-free bytes"},
		{"argv non-string", spawnVariant(t, `"resume",`, `"resume", 7,`), "argv", "not 1..4096 NUL-free bytes"},
		{"argv NUL element", spawnVariant(t, `"resume",`, `"resume", "a\u0000b",`), "argv", "not 1..4096 NUL-free bytes"},
		{"argv 129 elements", spawnVariant(t, `"argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ]`, `"argv": [`+strings.Repeat(`"a", `, 128)+`"a"]`), "argv", "empty or longer than 128"},
		{"argv exceeds 65536 total", spawnVariant(t, `"argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ]`, `"argv": [`+strings.Repeat(`"`+long4096+`", `, 16)+`"a"]`), "argv", "exceeds 65536 bytes total"},
		{"cwd relative", spawnVariant(t, `"cwd": "/srv/relux/payments-api/src"`, `"cwd": "srv/relux/payments-api/src"`), "cwd", "not an absolute path"},
		{"cwd windows on linux", spawnVariant(t, `"cwd": "/srv/relux/payments-api/src"`, `"cwd": "C:\\srv\\relux"`), "cwd", "not an absolute path"},
		{"env names 65", spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": [`+quotedRange(65)+`]`), "env_names", "exceed 64 entries"},
		{"env name bad grammar", spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": ["9LIVES"]`), "env_names", "not environment names"},
		{"env name 129", spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": ["`+strings.Repeat("E", 129)+`"]`), "env_names", "not environment names"},
		{"env names unsorted", spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": ["ZED", "ALPHA"]`), "env_names", "not sorted unique"},
		{"env names duplicated", spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": ["ALPHA", "ALPHA"]`), "env_names", "not sorted unique"},
		{"env literal overlaps", spawnVariant(t, `"env_literals": {}`, `"env_literals": {"OPENAI_API_KEY": "x"}`), "env_literals", "not disjoint"},
		{"env literal bad key", spawnVariant(t, `"env_literals": {}`, `"env_literals": {"9LIVES": "x"}`), "env_literals", "not disjoint"},
		{"env literal long value", spawnVariant(t, `"env_literals": {}`, `"env_literals": {"ALPHA": "`+long4097+`"}`), "env_literals", "not at most 4096 NUL-free bytes"},
		{"env literal non-string", spawnVariant(t, `"env_literals": {}`, `"env_literals": {"ALPHA": 7}`), "env_literals", "not at most 4096 NUL-free bytes"},
		{"env literal NUL value", spawnVariant(t, `"env_literals": {}`, `"env_literals": {"ALPHA": "a\u0000b"}`), "env_literals", "not at most 4096 NUL-free bytes"},
		{"profile mapping 513", spawnVariant(t, `"profile_mapping": "--dangerously-bypass-approvals-and-sandbox"`, `"profile_mapping": "`+long513+`"`), "profile_mapping", "not at most 512 characters"},
		{"native session 513", spawnVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": "`+long513+`"`), "native_session_id", "not 1..512 characters or null"},
		{"native session number", spawnVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": 7`), "native_session_id", "not 1..512 characters or null"},
		{"profile mapping mismatch", spawnVariant(t, `"profile_mapping": "--dangerously-bypass-approvals-and-sandbox"`, `"profile_mapping": "--yolo"`), "profile_mapping", "does not match the provider profile"},
		{"profile mapping standard lie", spawnVariant(t, `"profile_mapping": "--dangerously-bypass-approvals-and-sandbox"`, `"profile_mapping": ""`), "profile_mapping", "does not match the provider profile"},
		{"extensions array", spawnVariant(t, `"extensions": {}`, `"extensions": []`), "extensions", "not an object"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireFrameRefusal(t, DecodeSpawnPlan(row.body, "codex", ProfileYOLO, scalar.PlatformLinux), row.member, row.detail)
		})
	}
	// Boundary acceptances: 4,096-byte argv elements and literals,
	// 512-character mappings, 128-element and 65,536-byte totals, and
	// 64 sorted names validate.
	if err := DecodeSpawnPlan(spawnVariant(t, `"resume",`, `"resume", "`+long4096+`",`), "codex", ProfileYOLO, scalar.PlatformLinux); err != nil {
		t.Fatalf("4096-byte argv element refused: %v", err)
	}
	if err := DecodeSpawnPlan(spawnVariant(t, `"argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ]`, `"argv": [`+strings.Repeat(`"`+long4096+`", `, 15)+`"`+long4096+`"]`), "codex", ProfileYOLO, scalar.PlatformLinux); err != nil {
		t.Fatalf("65536-byte argv total refused: %v", err)
	}
	if err := DecodeSpawnPlan(spawnVariant(t, `"env_literals": {}`, `"env_literals": {"ALPHA": "`+long4096+`"}`), "codex", ProfileYOLO, scalar.PlatformLinux); err != nil {
		t.Fatalf("4096-byte literal refused: %v", err)
	}
	// A 512-character mapping clears the length gate and falls
	// through to the mismatch rule: the bound is exact, and only the
	// Section 7.7 strings clear both.
	requireFrameRefusal(t, DecodeSpawnPlan(spawnVariant(t, `"profile_mapping": "--dangerously-bypass-approvals-and-sandbox"`, `"profile_mapping": "`+long512+`"`), "codex", ProfileYOLO, scalar.PlatformLinux), "profile_mapping", "does not match the provider profile")
}

// quotedRange renders "E0000", "E0001", ... quoted and comma-joined:
// zero-padded so the sequence is bytewise sorted.
func quotedRange(count int) string {
	names := make([]string, 0, count)
	for i := 0; i < count; i++ {
		names = append(names, fmt.Sprintf(`"E%04d"`, i))
	}
	return strings.Join(names, ", ")
}

// TestDecodeSpawnPlanMappingSeats proves the caller seats of the
// mapping check: an unknown provider or profile for codex-shaped
// bodies is a caller error, and a yolo body checked under standard
// is a mapping mismatch, not an acceptance.
func TestDecodeSpawnPlanMappingSeats(t *testing.T) {
	requireLocalRefusal(t, DecodeSpawnPlan([]byte(specResumePlan), "qwen", ProfileYOLO, scalar.PlatformLinux), "invalid_config", "unknown provider")
	requireLocalRefusal(t, DecodeSpawnPlan([]byte(specResumePlan), "codex", "unrestricted", scalar.PlatformLinux), "invalid_config", "unknown profile")
	requireFrameRefusal(t, DecodeSpawnPlan([]byte(specResumePlan), "codex", ProfileStandard, scalar.PlatformLinux), "profile_mapping", "does not match the provider profile")
	requireFrameRefusal(t, DecodeSpawnPlan([]byte(specResumePlan), "claude", ProfileYOLO, scalar.PlatformLinux), "profile_mapping", "does not match the provider profile")
}
