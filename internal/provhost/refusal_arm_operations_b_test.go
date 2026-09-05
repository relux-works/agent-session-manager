package provhost

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// declaredOperationWitnessesQuiesce proves the quiescence and
// spawn-plan arms. It extends declaredOperationWitnesses in
// refusal_arm_operations_a_test.go; the split is file size only.
func declaredOperationWitnessesQuiesce() []armWitness {
	return []armWitness{
		// Quiescence arms, through DecodeQuiesceProof.
		{arm: `ctor|failProtocol|quiesce background_idle is not a boolean or null`, name: "quiesce numeric background", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"background_idle": true,`, `"background_idle": 1,`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "background_idle", "not a boolean or null")
		}},
		{arm: `ctor|failProtocol|quiesce blockers are not sorted unique`, name: "quiesce unsorted blockers", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"safe": true,
  "blockers": []`, `"safe": false,
  "blockers": ["provider_busy", "background_unproven"]`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "blockers", "not sorted unique")
		}},
		{arm: `ctor|failProtocol|quiesce blockers exceed 5 entries`, name: "quiesce six blockers", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"blockers": []`, `"blockers": ["background_unproven", "child_process_open", "database_handle_open", "provider_busy", "store_unstable", "provider_busy"]`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "blockers", "exceed 5 entries")
		}},
		{arm: `ctor|failProtocol|quiesce blockers name an unknown blocker`, name: "quiesce process_open blocker", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"blockers": []`, `"blockers": ["process_open"]`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "blockers", "unknown blocker")
		}},
		{arm: `ctor|failProtocol|quiesce boundary_ref is not 1..1024 characters or null`, name: "quiesce numeric boundary", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"boundary_ref": "provider-event-42",`, `"boundary_ref": 42,`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "boundary_ref", "not 1..1024 characters or null")
		}},
		{arm: `ctor|failProtocol|quiesce count is not a uint53`, name: "quiesce fractional count", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"open_child_count": 0,`, `"open_child_count": 1.5,`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "open_child_count", "not a uint53")
		}},
		{arm: `ctor|failProtocol|quiesce foreground_idle is not a boolean`, name: "quiesce numeric foreground", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"foreground_idle": true,`, `"foreground_idle": 1,`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "foreground_idle", "not a boolean")
		}},
		{arm: `ctor|failProtocol|quiesce input_blocked is not a boolean`, name: "quiesce string input", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"input_blocked": true,`, `"input_blocked": "yes",`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "input_blocked", "not a boolean")
		}},
		{arm: `ctor|failProtocol|quiesce proof carries unknown member`, name: "quiesce score member", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"safe": true,`, `"safe": true, "score": 1,`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "score", "unknown member")
		}},
		{arm: `ctor|failProtocol|quiesce proof misses a required member`, name: "quiesce without safe", prove: func(t *testing.T) {
			body := []byte(strings.Replace(safeQuiesceProof, "  \"safe\": true,\n", "", 1))
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "safe", "misses a required member")
		}},
		{arm: `ctor|failProtocol|quiesce provider_id is not a provider id`, name: "quiesce uppercase provider", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"provider_id": "codex"`, `"provider_id": "Codex"`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "provider_id", "not a provider id")
		}},
		{arm: `ctor|failProtocol|quiesce provider_version is not 1..128 characters`, name: "quiesce empty version", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"provider_version": "0.147.0"`, `"provider_version": ""`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "provider_version", "not 1..128 characters")
		}},
		{arm: `ctor|failProtocol|quiesce safe is not a boolean`, name: "quiesce string safe", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"safe": true,`, `"safe": "yes",`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "safe", "not a boolean")
		}},
		{arm: `ctor|failProtocol|quiesce safe proof leaves a fact unproven`, name: "quiesce safe with open child", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"open_child_count": 0,`, `"open_child_count": 1,`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "safe", "leaves a fact unproven")
		}},
		{arm: `ctor|failProtocol|quiesce store_generation is not 1..512 characters or null`, name: "quiesce overlong generation", prove: func(t *testing.T) {
			body := quiesceVariant(t, `"store_generation": "closed:11111111-2222-4333-8444-555555555555:1",`, `"store_generation": "`+strings.Repeat("g", 513)+`",`)
			_, err := DecodeQuiesceProof(body)
			requireFrameRefusal(t, err, "store_generation", "not 1..512 characters or null")
		}},
		// Spawn-plan arms, through DecodeSpawnPlan.
		{arm: `ctor|failProtocol|spawn argv element is not 1..4096 NUL-free bytes`, name: "spawn empty argv element", prove: func(t *testing.T) {
			body := spawnVariant(t, `"resume",`, `"resume", "",`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "argv", "not 1..4096 NUL-free bytes")
		}},
		{arm: `ctor|failProtocol|spawn argv exceeds 65536 bytes total`, name: "spawn seventeen full elements", prove: func(t *testing.T) {
			elements := strings.Repeat(`"`+strings.Repeat("a", 4096)+`",`, 17)
			body := spawnVariant(t, `"argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ]`, `"argv": [`+elements[:len(elements)-1]+`]`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "argv", "exceeds 65536 bytes total")
		}},
		{arm: `ctor|failProtocol|spawn argv is empty or longer than 128`, name: "spawn empty argv", prove: func(t *testing.T) {
			body := spawnVariant(t, `"argv": [
      "codex",
      "--dangerously-bypass-approvals-and-sandbox",
      "resume",
      "11111111-2222-4333-8444-555555555555"
    ]`, `"argv": []`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "argv", "empty or longer than 128")
		}},
		{arm: `ctor|failProtocol|spawn cwd is not an absolute path`, name: "spawn relative cwd", prove: func(t *testing.T) {
			body := spawnVariant(t, `"cwd": "/srv/relux/payments-api/src"`, `"cwd": "srv/relux/payments-api/src"`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "cwd", "not an absolute path")
		}},
		{arm: `ctor|failProtocol|spawn env_literal is not at most 4096 NUL-free bytes`, name: "spawn overlong literal", prove: func(t *testing.T) {
			body := spawnVariant(t, `"env_literals": {}`, `"env_literals": {"ALPHA": "`+strings.Repeat("a", 4097)+`"}`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "env_literals", "not at most 4096 NUL-free bytes")
		}},
		{arm: `ctor|failProtocol|spawn env_literal key is not disjoint`, name: "spawn overlapping literal", prove: func(t *testing.T) {
			body := spawnVariant(t, `"env_literals": {}`, `"env_literals": {"OPENAI_API_KEY": "x"}`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "env_literals", "not disjoint")
		}},
		{arm: `ctor|failProtocol|spawn env_literals exceed 64 entries`, name: "spawn 65 literals", prove: func(t *testing.T) {
			var entries strings.Builder
			for i := 0; i < 65; i++ {
				if i > 0 {
					entries.WriteString(", ")
				}
				entries.WriteString(fmt.Sprintf(`"E%04d": "x"`, i))
			}
			body := spawnVariant(t, `"env_literals": {}`, `"env_literals": {`+entries.String()+`}`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "env_literals", "exceed 64 entries")
		}},
		{arm: `ctor|failProtocol|spawn env_names are not environment names`, name: "spawn digit name", prove: func(t *testing.T) {
			body := spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": ["9LIVES"]`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "env_names", "not environment names")
		}},
		{arm: `ctor|failProtocol|spawn env_names are not sorted unique`, name: "spawn unsorted names", prove: func(t *testing.T) {
			body := spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": ["ZED", "ALPHA"]`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "env_names", "not sorted unique")
		}},
		{arm: `ctor|failProtocol|spawn env_names exceed 64 entries`, name: "spawn 65 names", prove: func(t *testing.T) {
			body := spawnVariant(t, `"env_names": ["OPENAI_API_KEY"]`, `"env_names": [`+quotedRange(65)+`]`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "env_names", "exceed 64 entries")
		}},
		{arm: `ctor|failProtocol|spawn extensions is not an object`, name: "spawn array extensions", prove: func(t *testing.T) {
			body := spawnVariant(t, `"extensions": {}`, `"extensions": []`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "extensions", "not an object")
		}},
		{arm: `ctor|failProtocol|spawn native_session_id is not 1..512 characters or null`, name: "spawn numeric native session", prove: func(t *testing.T) {
			body := spawnVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": 7`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "native_session_id", "not 1..512 characters or null")
		}},
		{arm: `ctor|failProtocol|spawn plan carries unknown member`, name: "spawn priority member", prove: func(t *testing.T) {
			body := spawnVariant(t, `"extensions": {}`, `"extensions": {}, "priority": 1`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "priority", "unknown member")
		}},
		{arm: `ctor|failProtocol|spawn plan misses a required member`, name: "spawn without extensions", prove: func(t *testing.T) {
			body := []byte(strings.Replace(specResumePlan, `,
    "extensions": {}`, "", 1))
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "extensions", "misses a required member")
		}},
		{arm: `ctor|failProtocol|spawn profile_mapping does not match the provider profile`, name: "spawn claude flag for codex", prove: func(t *testing.T) {
			body := spawnVariant(t, `"profile_mapping": "--dangerously-bypass-approvals-and-sandbox"`, `"profile_mapping": "--yolo"`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "profile_mapping", "does not match the provider profile")
		}},
		{arm: `ctor|failProtocol|spawn profile_mapping is not at most 512 characters`, name: "spawn 513-char mapping", prove: func(t *testing.T) {
			body := spawnVariant(t, `"profile_mapping": "--dangerously-bypass-approvals-and-sandbox"`, `"profile_mapping": "`+strings.Repeat("m", 513)+`"`)
			requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "profile_mapping", "not at most 512 characters")
		}},
	}
}
