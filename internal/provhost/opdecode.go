package provhost

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file holds parse-only helpers for the Section 7.3-7.7 operation
// bodies. Every helper reports a verdict and never refuses: all refusals
// call one of the six refusal constructors directly at the deciding site
// with a literal detail, so the refusal-arm inventory keeps seeing every
// arm. A helper that fronted a constructor would collapse distinct rules
// into one obligation and is not used here.

// maxUint53 is the largest exactly representable JSON integer: Section
// 7.5 counts and Section 5.1 limits are uint53 throughout.
const maxUint53 = uint64(1<<53 - 1)

// unknownMember names a member outside the closed set, if any.
func unknownMember(members map[string]json.RawMessage, allowed map[string]bool) (string, bool) {
	for name := range members {
		if !allowed[name] {
			return name, true
		}
	}
	return "", false
}

// missingMember names the first required member with no value, if any.
func missingMember(members map[string]json.RawMessage, required []string) (string, bool) {
	for _, name := range required {
		if _, present := members[name]; !present {
			return name, true
		}
	}
	return "", false
}

// rawString reads a JSON string member.
func rawString(raw json.RawMessage) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

// rawBool reads a JSON boolean member.
func rawBool(raw json.RawMessage) (bool, bool) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, false
	}
	return value, true
}

// rawUint53 reads an unsigned integral JSON number in [0, 2^53-1]. A
// fraction, exponent, sign, or out-of-range magnitude is not a uint53,
// even when it names the same mathematical value: the canonical bodies
// carry counts, never measurements.
func rawUint53(raw json.RawMessage) (uint64, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, false
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	literal := number.String()
	if literal == "" || strings.ContainsAny(literal, ".eE+-") {
		return 0, false
	}
	parsed, err := strconv.ParseUint(literal, 10, 64)
	if err != nil || parsed > maxUint53 {
		return 0, false
	}
	return parsed, true
}

// rawNullableString reads a JSON string-or-null member: null reports
// isNull without a value, any other non-string reports !ok.
func rawNullableString(raw json.RawMessage) (value string, isNull bool, ok bool) {
	if string(raw) == "null" {
		return "", true, true
	}
	value, ok = rawString(raw)
	return value, false, ok
}

// rawArray reads a JSON array member into its raw elements. Null is not
// an array: every array member in Sections 7.3-7.5 is required.
func rawArray(raw json.RawMessage) ([]json.RawMessage, bool) {
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, false
	}
	var elements []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&elements); err != nil {
		return nil, false
	}
	if elements == nil {
		return []json.RawMessage{}, true
	}
	return elements, true
}

// rawStringArray reads an array whose every element is a JSON string.
func rawStringArray(raw json.RawMessage) ([]string, bool) {
	elements, ok := rawArray(raw)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(elements))
	for _, element := range elements {
		value, ok := rawString(element)
		if !ok {
			return nil, false
		}
		out = append(out, value)
	}
	return out, true
}

// sortedUniqueStrings reports whether the values are in strictly
// increasing bytewise order: sorted with no duplicates. Bytewise order
// is the canonical-form order every closed array in Sections 7.3-7.5
// is written in.
func sortedUniqueStrings(values []string) bool {
	return sort.StringsAreSorted(values) && !hasDuplicateString(values)
}

func hasDuplicateString(values []string) bool {
	for i := 1; i < len(values); i++ {
		if values[i] == values[i-1] {
			return true
		}
	}
	return false
}

// runeLength reports the Unicode-character length the Section 7 string
// bounds count in. JSON transport is UTF-8, so characters are runes,
// not bytes and not code units.
func runeLength(value string) int {
	return utf8.RuneCountInString(value)
}

// isDigest reports whether the value parses as a content digest.
func isDigest(value string) bool {
	_, err := scalar.ParseDigest(value)
	return err == nil
}

// isUUIDv7 reports whether the value parses as a UUIDv7.
func isUUIDv7(value string) bool {
	_, err := scalar.ParseUUIDv7(value)
	return err == nil
}

// isTimestamp reports whether the value parses as a timestamp.
func isTimestamp(value string) bool {
	_, err := scalar.ParseTimestamp(value)
	return err == nil
}
