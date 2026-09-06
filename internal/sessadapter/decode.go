package sessadapter

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file holds parse-only helpers for every Section 7.8 closed
// body. Every helper reports a verdict and never refuses: all refusals
// call one of the refusal constructors directly at the deciding site
// with a literal detail, so the refusal-arm inventory keeps seeing
// every arm. A helper that fronted a constructor would collapse
// distinct rules into one obligation and is not used here.

// maxUint53 is the largest exactly representable JSON integer:
// Section 7.8 counts and limits are uint53 throughout.
const maxUint53 = uint64(1<<53 - 1)

// frameFault is one strict-decode failure: what was wrong and which
// member it was found in, if any. It never carries foreign bytes.
type frameFault struct {
	detail string
	member string
}

// decodeStrictObject parses one complete JSON object with duplicate
// member rejection. It is the only JSON entry point in this package:
// every body crosses it before any member is read. The frame verdict
// itself is the shared environ verdict: this function delegates to
// environ.DecodeStrictObject and translates the fault into the
// package refusal dialect without reimplementing any rule, so the
// two can never drift. The delegated string-walk surrogate gate
// refuses a real lone escape behind an even backslash run and
// admits quoted backslash-u text; the raw byte scan this replaces
// misread the quoted text as a high surrogate and admitted the
// real lone escape behind it. A non-UTF-8 frame, a lone surrogate
// escape, a non-object top level, a duplicated member, or trailing
// data is a fault, never a partial object the caller could mistake
// for a result.
func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	members, fault := environ.DecodeStrictObject(data)
	if fault != nil {
		return nil, &frameFault{detail: fault.Detail, member: fault.Member}
	}
	return members, nil
}

// unknownMember names a member outside the closed set, if any. The
// single-unknown fixtures keep the verdict deterministic: with
// several unknowns the map order decides which is named, so tests
// never assert a name from a multi-unknown body.
func unknownMember(members map[string]json.RawMessage, allowed map[string]bool) (string, bool) {
	for name := range members {
		if !allowed[name] {
			return name, true
		}
	}
	return "", false
}

// missingMember names the first required member with no value, if
// any. The required list is fixed order, so the verdict is
// deterministic for every missing combination.
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
// even when it names the same mathematical value: the canonical
// bodies carry counts, never measurements.
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
	return parseUint53Literal(literal)
}

// parseUint53Literal reads a digit-only literal without trusting a
// float conversion: the magnitude bound is checked on the digit
// string, so 2^53 is refused even on platforms that could hold it.
func parseUint53Literal(literal string) (uint64, bool) {
	var value uint64
	for index := 0; index < len(literal); index++ {
		digit := literal[index]
		if digit < '0' || digit > '9' {
			return 0, false
		}
		value = value*10 + uint64(digit-'0')
		if value > maxUint53 {
			return 0, false
		}
	}
	return value, true
}

// stringLength is the Section 1.6 string measure in characters, not
// bytes: a bound of string[1..128] admits 128 characters of any
// width.
func stringLength(value string) int {
	return utf8.RuneCountInString(value)
}

// checkStringBounds reports whether the member is a JSON string whose
// character count falls in the inclusive bound.
func checkStringBounds(raw json.RawMessage, minimum, maximum int) (string, bool) {
	value, ok := rawString(raw)
	if !ok {
		return "", false
	}
	length := stringLength(value)
	if length < minimum || length > maximum {
		return "", false
	}
	return value, true
}

// checkUint53Bounds reports whether the member is a uint53 in the
// inclusive bound. Both edges are checked: a one-sided check would
// admit the open side.
func checkUint53Bounds(raw json.RawMessage, minimum, maximum uint64) (uint64, bool) {
	value, ok := rawUint53(raw)
	if !ok {
		return 0, false
	}
	if value < minimum || value > maximum {
		return 0, false
	}
	return value, true
}

// checkDigest reports whether the member is a digest of the pinned
// sha256 form.
func checkDigest(raw json.RawMessage) (scalar.Digest, bool) {
	value, ok := rawString(raw)
	if !ok {
		return scalar.Digest{}, false
	}
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return scalar.Digest{}, false
	}
	return digest, true
}

// checkUUIDv7 reports whether the member is a lowercase UUIDv7.
func checkUUIDv7(raw json.RawMessage) (scalar.UUIDv7, bool) {
	value, ok := rawString(raw)
	if !ok {
		return scalar.UUIDv7{}, false
	}
	identifier, err := scalar.ParseUUIDv7(value)
	if err != nil {
		return scalar.UUIDv7{}, false
	}
	return identifier, true
}

// checkTimestamp reports whether the member is a pinned timestamp.
func checkTimestamp(raw json.RawMessage) (scalar.Timestamp, bool) {
	value, ok := rawString(raw)
	if !ok {
		return scalar.Timestamp{}, false
	}
	instant, err := scalar.ParseTimestamp(value)
	if err != nil {
		return scalar.Timestamp{}, false
	}
	return instant, true
}

// isNull reports whether the member is an explicit JSON null.
func isNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// decodeArray reads a JSON array member into its raw elements. A
// non-array — including null — is not an empty array.
func decodeArray(raw json.RawMessage) ([]json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil, false
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return nil, false
	}
	var elements []json.RawMessage
	for decoder.More() {
		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, false
		}
		elements = append(elements, element)
	}
	if token, err := decoder.Token(); err != nil {
		return nil, false
	} else if delim, ok := token.(json.Delim); !ok || delim != ']' {
		return nil, false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, false
	}
	return elements, true
}

// checkSortedUniqueStrings reports whether the member is an array of
// strings in the item bound whose JCS encodings order strictly
// increasingly with no duplicates. Bytewise JCS order over strings
// is plain lexicographic order, so the check compares directly; the
// count bound is checked on both edges.
func checkSortedUniqueStrings(raw json.RawMessage, minimumLength, maximumLength int, minimumCount, maximumCount uint64) ([]string, bool) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, false
	}
	if uint64(len(elements)) < minimumCount || uint64(len(elements)) > maximumCount {
		return nil, false
	}
	values := make([]string, 0, len(elements))
	for _, element := range elements {
		value, ok := rawString(element)
		if !ok {
			return nil, false
		}
		length := stringLength(value)
		if length < minimumLength || length > maximumLength {
			return nil, false
		}
		values = append(values, value)
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] >= values[index] {
			return nil, false
		}
	}
	return values, true
}

// checkSortedUniqueDigests reports whether the member is a sorted
// unique digest array in the count bound.
func checkSortedUniqueDigests(raw json.RawMessage, minimumCount, maximumCount uint64) ([]scalar.Digest, bool) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, false
	}
	if uint64(len(elements)) < minimumCount || uint64(len(elements)) > maximumCount {
		return nil, false
	}
	values := make([]scalar.Digest, 0, len(elements))
	for _, element := range elements {
		digest, ok := checkDigest(element)
		if !ok {
			return nil, false
		}
		values = append(values, digest)
	}
	for index := 1; index < len(values); index++ {
		if values[index-1].String() >= values[index].String() {
			return nil, false
		}
	}
	return values, true
}

// semverPattern is the SemVer grammar adapter and environment
// versions satisfy: three dot-separated non-negative integers with
// no leading zeros, an optional pre-release, and optional build
// metadata.
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// providerIDPattern is the Section 7.1 provider-id grammar the
// manifest, context, and binding layers read:
// [a-z][a-z0-9-]{0,31}.
var providerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// environmentIDPattern is the Section 7.8 environment-id grammar:
// [a-z][a-z0-9.-]{0,63}. One semantic native environment per
// adapter; dots separate the native hierarchy, never trust.
var environmentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}$`)

// reverseDNSPattern is the Section 1.6 extension-key grammar:
// dot-separated lowercase labels, 3..253 characters.
var reverseDNSPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$`)

// checkExtensions reports whether the member is an object whose keys
// are all reverse-DNS names. Values cross opaquely: extensions
// cannot add operations, capabilities, or trust facts, and no member
// of this package reads an extension value for any admission
// decision.
func checkExtensions(raw json.RawMessage) bool {
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return false
	}
	for name := range members {
		if len(name) < 3 || len(name) > 253 || !reverseDNSPattern.MatchString(name) {
			return false
		}
	}
	return true
}
