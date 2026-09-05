package provhost

import (
	"encoding/json"
)

// This file validates the SafeBoundaryProof a quiesce operation
// returns as its success body (Section 7.5 row, Section 7.6 rules). A
// safe proof requires empty blockers, input_blocked, foreground_idle,
// background_idle true, both counts zero, and non-null boundary and
// store generation. If background idleness, process exit, or store
// closure cannot be proven, safe MUST be false and graceful takeover
// MUST stop; the plugin MUST NOT silently convert the request. An
// unsafe proof is an honest observation and validates; only a safe
// claim over unproven facts is refused.

// quiesceBlockers is the closed blocker enum Section 7.5 defines.
var quiesceBlockers = []string{
	"background_unproven",
	"child_process_open",
	"database_handle_open",
	"provider_busy",
	"store_unstable",
}

// quiesceMembers is the exact required member set of a
// SafeBoundaryProof.
var quiesceMembers = map[string]bool{
	"provider_id":                true,
	"provider_version":           true,
	"input_blocked":              true,
	"boundary_ref":               true,
	"foreground_idle":            true,
	"background_idle":            true,
	"open_child_count":           true,
	"open_database_handle_count": true,
	"store_generation":           true,
	"safe":                       true,
	"blockers":                   true,
}

// quiesceRequired lists quiesceMembers in a fixed order so a proof
// missing several members always names the same one.
var quiesceRequired = []string{
	"provider_id",
	"provider_version",
	"input_blocked",
	"boundary_ref",
	"foreground_idle",
	"background_idle",
	"open_child_count",
	"open_database_handle_count",
	"store_generation",
	"safe",
	"blockers",
}

// DecodeQuiesceProof validates one quiesce-operation success body as a
// SafeBoundaryProof. It returns the observed safe bit on an honest
// proof — safe with every fact proven, or unsafe — and a
// provider_protocol_error naming the offending member on any
// malformed proof or any safe claim over unproven facts. The caller
// stops graceful takeover on safe false; the proof itself never
// converts the request.
func DecodeQuiesceProof(body []byte) (bool, error) {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return false, err
		}
		return false, failure
	}
	if name, unknown := unknownMember(members, quiesceMembers); unknown {
		failure, err := failProtocol("quiesce proof carries unknown member", name)
		if err != nil {
			return false, err
		}
		return false, failure
	}
	if name, missing := missingMember(members, quiesceRequired); missing {
		failure, err := failProtocol("quiesce proof misses a required member", name)
		if err != nil {
			return false, err
		}
		return false, failure
	}
	if provider, ok := rawString(members["provider_id"]); !ok || !validProviderID(provider) {
		failure, err := failProtocol("quiesce provider_id is not a provider id", "provider_id")
		if err != nil {
			return false, err
		}
		return false, failure
	}
	if version, ok := rawString(members["provider_version"]); !ok || runeLength(version) < 1 || runeLength(version) > 128 {
		failure, err := failProtocol("quiesce provider_version is not 1..128 characters", "provider_version")
		if err != nil {
			return false, err
		}
		return false, failure
	}
	inputBlocked, ok := rawBool(members["input_blocked"])
	if !ok {
		failure, err := failProtocol("quiesce input_blocked is not a boolean", "input_blocked")
		if err != nil {
			return false, err
		}
		return false, failure
	}
	foregroundIdle, ok := rawBool(members["foreground_idle"])
	if !ok {
		failure, err := failProtocol("quiesce foreground_idle is not a boolean", "foreground_idle")
		if err != nil {
			return false, err
		}
		return false, failure
	}
	safe, ok := rawBool(members["safe"])
	if !ok {
		failure, err := failProtocol("quiesce safe is not a boolean", "safe")
		if err != nil {
			return false, err
		}
		return false, failure
	}
	boundaryNull, generationNull, background, backgroundNull, err := checkQuiesceNullable(members)
	if err != nil {
		return false, err
	}
	childCount, databaseCount, err := checkQuiesceCounts(members)
	if err != nil {
		return false, err
	}
	blockers, err := checkQuiesceBlockers(members["blockers"])
	if err != nil {
		return false, err
	}
	// A safe claim over unproven facts is the one lie Section 7.6
	// names: every conjunct below carries a fixture with only it
	// violated, except backgroundNull, which rawNullableBool folds
	// into !background (null reads as false), so no fixture can
	// isolate it. Narrowing any other conjunct reddens its fixture.
	if safe {
		if len(blockers) != 0 || !inputBlocked || !foregroundIdle || backgroundNull || !background || boundaryNull || generationNull || childCount != 0 || databaseCount != 0 {
			failure, fault := failProtocol("quiesce safe proof leaves a fact unproven", "safe")
			if fault != nil {
				return false, fault
			}
			return false, failure
		}
	}
	return safe, nil
}

// rawNullableBool reads a JSON boolean-or-null member.
func rawNullableBool(raw json.RawMessage) (value bool, isNull bool, ok bool) {
	if string(raw) == "null" {
		return false, true, true
	}
	value, ok = rawBool(raw)
	return value, false, ok
}

// checkQuiesceNullable enforces the nullability shapes outside the
// safe rule: boundary_ref is string[1..1024] or null, and
// store_generation is string[1..512] or null. Length is checked here
// so the safe rule only decides null against non-null, and the parsed
// null flags and background value are returned for that rule.
func checkQuiesceNullable(members map[string]json.RawMessage) (boundaryNull bool, generationNull bool, background bool, backgroundNull bool, err error) {
	fail := func(failure error) (bool, bool, bool, bool, error) {
		return false, false, false, false, failure
	}
	boundary, boundaryIsNull, ok := rawNullableString(members["boundary_ref"])
	if !ok || (!boundaryIsNull && (runeLength(boundary) < 1 || runeLength(boundary) > 1024)) {
		failure, fault := failProtocol("quiesce boundary_ref is not 1..1024 characters or null", "boundary_ref")
		if fault != nil {
			return fail(fault)
		}
		return fail(failure)
	}
	generation, generationIsNull, ok := rawNullableString(members["store_generation"])
	if !ok || (!generationIsNull && (runeLength(generation) < 1 || runeLength(generation) > 512)) {
		failure, fault := failProtocol("quiesce store_generation is not 1..512 characters or null", "store_generation")
		if fault != nil {
			return fail(fault)
		}
		return fail(failure)
	}
	background, backgroundNull, ok = rawNullableBool(members["background_idle"])
	if !ok {
		failure, fault := failProtocol("quiesce background_idle is not a boolean or null", "background_idle")
		if fault != nil {
			return fail(fault)
		}
		return fail(failure)
	}
	return boundaryIsNull, generationIsNull, background, backgroundNull, nil
}

// checkQuiesceCounts requires both handle counts as uint53 and
// returns them for the safe rule: a safe proof with either count
// open claims idleness it does not have.
func checkQuiesceCounts(members map[string]json.RawMessage) (uint64, uint64, error) {
	counts := make([]uint64, 0, 2)
	for _, name := range []string{"open_child_count", "open_database_handle_count"} {
		count, ok := rawUint53(members[name])
		if !ok {
			failure, err := failProtocol("quiesce count is not a uint53", name)
			if err != nil {
				return 0, 0, err
			}
			return 0, 0, failure
		}
		counts = append(counts, count)
	}
	return counts[0], counts[1], nil
}

// checkQuiesceBlockers requires the sorted unique closed-enum array of
// at most five blockers and returns it for the safe rule.
func checkQuiesceBlockers(raw json.RawMessage) ([]string, error) {
	blockers, ok := rawStringArray(raw)
	if !ok || len(blockers) > 5 {
		failure, err := failProtocol("quiesce blockers exceed 5 entries", "blockers")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	allowed := map[string]bool{}
	for _, blocker := range quiesceBlockers {
		allowed[blocker] = true
	}
	for _, blocker := range blockers {
		if !allowed[blocker] {
			failure, err := failProtocol("quiesce blockers name an unknown blocker", "blockers")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
	}
	if !sortedUniqueStrings(blockers) {
		failure, err := failProtocol("quiesce blockers are not sorted unique", "blockers")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	return blockers, nil
}
