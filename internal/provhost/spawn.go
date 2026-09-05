package provhost

import (
	"encoding/json"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the SpawnPlan a launch or resume operation
// returns as its success body (Section 7.5 rows). The trusted
// terminal backend performs process creation from this plan, so the
// plan MUST NOT carry secret environment values, and its
// profile_mapping MUST equal the Section 7.7 mapping for the calling
// provider and profile. Literal secrecy is structural, not
// content-admitted: values are bounded and disjoint from inherited
// names, and the host never logs bodies; no content inspection could
// prove a literal non-secret, so none is attempted.

// spawnMembers is the exact required member set of a SpawnPlan.
var spawnMembers = map[string]bool{
	"argv":              true,
	"cwd":               true,
	"env_names":         true,
	"env_literals":      true,
	"native_session_id": true,
	"profile_mapping":   true,
	"extensions":        true,
}

// spawnRequired lists spawnMembers in a fixed order so a plan missing
// several members always names the same one.
var spawnRequired = []string{
	"argv",
	"cwd",
	"env_names",
	"env_literals",
	"native_session_id",
	"profile_mapping",
	"extensions",
}

// maxSpawnArgvBytes bounds the total encoded argv Section 5.1 states.
const maxSpawnArgvBytes = 65536

// DecodeSpawnPlan validates one launch- or resume-operation success
// body as a SpawnPlan for the calling provider under the calling
// profile on the destination platform. It returns nil on a
// well-formed plan and a refusal naming the offending member
// otherwise: a provider_protocol_error for a malformed plan, and an
// invalid_config caller error when the caller names an unknown
// provider or profile for the mapping check.
func DecodeSpawnPlan(body []byte, providerID, profile string, platform scalar.Platform) error {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if name, unknown := unknownMember(members, spawnMembers); unknown {
		failure, err := failProtocol("spawn plan carries unknown member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if name, missing := missingMember(members, spawnRequired); missing {
		failure, err := failProtocol("spawn plan misses a required member", name)
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkSpawnArgv(members["argv"]); err != nil {
		return err
	}
	if cwd, ok := rawString(members["cwd"]); !ok || !isAbsoluteOn(cwd, platform) {
		failure, err := failProtocol("spawn cwd is not an absolute path", "cwd")
		if err != nil {
			return err
		}
		return failure
	}
	names, err := checkSpawnEnvNames(members["env_names"])
	if err != nil {
		return err
	}
	if err := checkSpawnEnvLiterals(members["env_literals"], names); err != nil {
		return err
	}
	if _, isNull, ok := rawNullableString(members["native_session_id"]); !ok || (!isNull && !isBoundedString(members["native_session_id"], 1, 512)) {
		failure, err := failProtocol("spawn native_session_id is not 1..512 characters or null", "native_session_id")
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkSpawnProfileMapping(members["profile_mapping"], providerID, profile); err != nil {
		return err
	}
	if !isJSONObject(members["extensions"]) {
		failure, err := failProtocol("spawn extensions is not an object", "extensions")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// isAbsoluteOn reports whether the path is absolute on the
// destination platform. Destination-native is the Section 7.5 rule;
// the platform arrives from the caller because the plan carries none.
func isAbsoluteOn(path string, platform scalar.Platform) bool {
	_, err := scalar.ParseAbsolutePath(platform, path)
	return err == nil
}

// isBoundedString reports whether the raw JSON string counts 1..max
// Unicode characters. The caller already established it is a string.
func isBoundedString(raw json.RawMessage, min, max int) bool {
	value, ok := rawString(raw)
	if !ok {
		return false
	}
	length := runeLength(value)
	return length >= min && length <= max
}

// checkSpawnArgv enforces the Section 5.1 argv limits: 1..128
// elements, each a NUL-free string of 1..4,096 bytes, 65,536 bytes
// total. Byte length is the exec-bound quantity, so elements count
// bytes here while the character-bounded members count runes.
func checkSpawnArgv(raw json.RawMessage) error {
	elements, ok := rawArray(raw)
	if !ok || len(elements) == 0 || len(elements) > 128 {
		failure, err := failProtocol("spawn argv is empty or longer than 128", "argv")
		if err != nil {
			return err
		}
		return failure
	}
	total := 0
	for _, element := range elements {
		value, ok := rawString(element)
		if !ok || len(value) < 1 || len(value) > 4096 || strings.ContainsRune(value, 0) {
			failure, err := failProtocol("spawn argv element is not 1..4096 NUL-free bytes", "argv")
			if err != nil {
				return err
			}
			return failure
		}
		total += len(value)
	}
	if total > maxSpawnArgvBytes {
		failure, err := failProtocol("spawn argv exceeds 65536 bytes total", "argv")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// validEnvName reports whether the name satisfies the Section 5.1
// environment-name grammar: [A-Za-z_][A-Za-z0-9_]{0,127}.
func validEnvName(name string) bool {
	if len(name) < 1 || len(name) > 128 {
		return false
	}
	for index := 0; index < len(name); index++ {
		char := name[index]
		valid := char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || index > 0 && char >= '0' && char <= '9'
		if !valid {
			return false
		}
	}
	return true
}

// checkSpawnEnvNames enforces sorted unique environment names
// [0..64] and returns them for the disjointness check.
func checkSpawnEnvNames(raw json.RawMessage) ([]string, error) {
	names, ok := rawStringArray(raw)
	if !ok || len(names) > 64 {
		failure, err := failProtocol("spawn env_names exceed 64 entries", "env_names")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	for _, name := range names {
		if !validEnvName(name) {
			failure, err := failProtocol("spawn env_names are not environment names", "env_names")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
	}
	if !sortedUniqueStrings(names) {
		failure, err := failProtocol("spawn env_names are not sorted unique", "env_names")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	return names, nil
}

// checkSpawnEnvLiterals enforces the literal map: at most 64
// entries, environment-name keys disjoint from env_names, string
// values of at most 4,096 bytes. Keys sort in canonical form on the
// wire; the host reads the set, so order is not rechecked here.
func checkSpawnEnvLiterals(raw json.RawMessage, names []string) error {
	literals, fault := decodeStrictObject(raw)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return err
		}
		return failure
	}
	if len(literals) > 64 {
		failure, err := failProtocol("spawn env_literals exceed 64 entries", "env_literals")
		if err != nil {
			return err
		}
		return failure
	}
	inherited := map[string]bool{}
	for _, name := range names {
		inherited[name] = true
	}
	for key, value := range literals {
		if !validEnvName(key) || inherited[key] {
			failure, err := failProtocol("spawn env_literal key is not disjoint", "env_literals")
			if err != nil {
				return err
			}
			return failure
		}
		text, ok := rawString(value)
		if !ok || len(text) > 4096 || strings.ContainsRune(text, 0) {
			failure, err := failProtocol("spawn env_literal is not at most 4096 NUL-free bytes", "env_literals")
			if err != nil {
				return err
			}
			return failure
		}
	}
	return nil
}

// checkSpawnProfileMapping requires profile_mapping to equal the
// Section 7.7 mapping for the calling provider and profile. A
// provider that answers yolo with the standard omission, or standard
// with a yolo flag, fails here rather than executing under the wrong
// sandbox. Unknown providers and profiles are caller errors that
// propagate from the mapping table.
func checkSpawnProfileMapping(raw json.RawMessage, providerID, profile string) error {
	mapping, ok := rawString(raw)
	if !ok || runeLength(mapping) > 512 {
		failure, err := failProtocol("spawn profile_mapping is not at most 512 characters", "profile_mapping")
		if err != nil {
			return err
		}
		return failure
	}
	want, err := ProfileMapping(providerID, profile)
	if err != nil {
		return err
	}
	if mapping != want {
		failure, fault := failProtocol("spawn profile_mapping does not match the provider profile", "profile_mapping")
		if fault != nil {
			return fault
		}
		return failure
	}
	return nil
}
