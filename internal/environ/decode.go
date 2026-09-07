package environ

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// maxUint53 is the largest exactly representable JSON integer:
// counts and limits are uint53 throughout the pinned
// specification.
const maxUint53 = uint64(1<<53 - 1)

// Fault is one strict-decode failure: what was wrong and which
// member it was found in, if any. It never carries foreign bytes.
// The five details are the complete fault space: every body this
// package refuses at the frame layer carries exactly one of them,
// and the frame-agreement battery derives its denominator from
// this list, so a sixth fault shape fails the census instead of
// passing silently.
type Fault struct {
	Detail string
	Member string
}

func (fault *Fault) Error() string {
	if fault.Member == "" {
		return "environment frame " + fault.Detail
	}
	return "environment frame " + fault.Detail + " in member " + fault.Member
}

// Frame fault details. Each is a constant so the agreement battery
// derives the fault space from production source instead of
// retyping it: a reworded fault fails its row rather than passing
// against a stale copy.
const (
	FaultNotUTF8       = "not valid UTF-8"
	FaultLoneSurrogate = "lone surrogate escape"
	FaultNotObject     = "not a JSON object"
	FaultDuplicate     = "duplicate member"
	FaultTrailing      = "trailing data after the object"
)

// DecodeStrictObject parses one complete JSON object with
// duplicate member rejection. It is the only JSON entry point in
// this package: every body crosses it before any member is read.
// A non-UTF-8 frame, a lone surrogate escape, a non-object top
// level, a duplicated member, or trailing data is a fault, never
// a partial object the caller could mistake for a result.
func DecodeStrictObject(data []byte) (map[string]json.RawMessage, *Fault) {
	if !utf8.Valid(data) {
		return nil, &Fault{Detail: FaultNotUTF8}
	}
	if HasLoneSurrogateEscape(data) {
		return nil, &Fault{Detail: FaultLoneSurrogate}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, &Fault{Detail: FaultNotObject}
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil, &Fault{Detail: FaultNotObject}
	}
	members := map[string]json.RawMessage{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, &Fault{Detail: FaultNotObject}
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, &Fault{Detail: FaultNotObject}
		}
		if _, duplicate := members[key]; duplicate {
			return nil, &Fault{Detail: FaultDuplicate, Member: key}
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, &Fault{Detail: FaultNotObject, Member: key}
		}
		members[key] = raw
	}
	if token, err := decoder.Token(); err != nil {
		return nil, &Fault{Detail: FaultNotObject}
	} else if delim, ok := token.(json.Delim); !ok || delim != '}' {
		return nil, &Fault{Detail: FaultNotObject}
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, &Fault{Detail: FaultTrailing}
	}
	return members, nil
}

// HasLoneSurrogateEscape reports whether the raw frame carries a
// UTF-16 escape the JSON string grammar cannot pair. Go's
// encoding/json would otherwise replace the lone escape with
// U+FFFD silently, so wire bodies carrying lone surrogates would
// be admitted with rewritten bytes.
//
// The walk is string-aware: only escapes inside JSON strings
// count, and a backslash escape consumes both bytes, so the text
// "\\ud800" (backslash-plus-text, JSON source `"\\ud800"`) is
// not an escape and is admitted at the frame gate. A scanner
// that matches every `\u` byte pair regardless of string context
// refuses that text, and worse, misreads it as a high surrogate
// paired with a following real escape — admitting a real lone
// escape the string walk refuses. The sessadapter and dirnode
// facades once carried that raw scan and diverged in both
// directions; they now delegate their frame verdict to
// DecodeStrictObject, and the frame-agreement battery proves
// both directions with run-parity rows plus even-run-plus-real-
// escape compositions, so any change to this walk fails there in
// either direction.
func HasLoneSurrogateEscape(data []byte) bool {
	for index := 0; index < len(data); index++ {
		if data[index] != '"' {
			continue
		}
		for index++; index < len(data); index++ {
			switch data[index] {
			case '"':
				goto stringComplete
			case '\\':
				index++
				if index >= len(data) || data[index] != 'u' {
					continue
				}
				unit, end, ok := readUTF16Escape(data, index-1)
				if !ok {
					// Not a well-formed \uXXXX escape: the
					// JSON-syntax arms own it, not this gate.
					continue
				}
				index = end - 1
				switch {
				case unit >= 0xd800 && unit <= 0xdbff:
					second, _, ok := readUTF16Escape(data, end)
					if !ok || second < 0xdc00 || second > 0xdfff {
						return true
					}
					index = end + 6 - 1
				case unit >= 0xdc00 && unit <= 0xdfff:
					return true
				}
			}
		}
		// An unterminated string owns no surrogate verdict: the
		// frame is not JSON at all, so the syntax arms refuse it
		// as not-a-JSON-object rather than misattributing it
		// here.
		return false
	stringComplete:
	}
	return false
}

// readUTF16Escape reads one \uXXXX escape starting at the offset.
// It reports the unit value and the offset just past the escape,
// or false when the escape is truncated or not hexadecimal.
func readUTF16Escape(data []byte, offset int) (uint16, int, bool) {
	if offset+6 > len(data) || data[offset] != '\\' || data[offset+1] != 'u' {
		return 0, offset, false
	}
	value, err := strconv.ParseUint(string(data[offset+2:offset+6]), 16, 16)
	if err != nil {
		return 0, offset, false
	}
	return uint16(value), offset + 6, true
}

// unknownMember names a member outside the closed set, if any.
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

// rawUint53 reads an unsigned integral JSON number in [0,
// 2^53-1]. A fraction, exponent, sign, or out-of-range magnitude
// is not a uint53, even when it names the same mathematical
// value: the bodies carry counts, never measurements.
func rawUint53(raw json.RawMessage) (uint64, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, false
	}
	// Decode reads one value and stops: without the trailing
	// check a hostile slice like `12a` would read as 12. Member
	// slices arriving through DecodeStrictObject can never carry
	// trailing data, but this entry takes raw slices and must
	// not trust them.
	if _, err := decoder.Token(); err != io.EOF {
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

// parseUint53Literal reads a digit-only literal without trusting
// a float conversion: the magnitude bound is checked on the digit
// string, so 2^53 is refused even on platforms that could hold
// it.
func parseUint53Literal(literal string) (uint64, bool) {
	if literal == "" {
		return 0, false
	}
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

// StringLength is the Section 1.6 string measure in characters,
// not bytes: a bound of string[1..128] admits 128 characters of
// any width. Byte-counted exec rules (Provider SpawnPlan argv and
// env literals, Section 5.1) do not go through here; the measure
// battery ledgers that split.
func StringLength(value string) int {
	return utf8.RuneCountInString(value)
}

// CheckStringBounds reports whether the member is a JSON string
// whose character count falls in the inclusive bound.
func CheckStringBounds(raw json.RawMessage, minimum, maximum int) (string, bool) {
	value, ok := rawString(raw)
	if !ok {
		return "", false
	}
	length := StringLength(value)
	if length < minimum || length > maximum {
		return "", false
	}
	return value, true
}

// CheckUint53Bounds reports whether the member is a uint53 in the
// inclusive bound. Both edges are checked: a one-sided check
// would admit the open side.
func CheckUint53Bounds(raw json.RawMessage, minimum, maximum uint64) (uint64, bool) {
	value, ok := rawUint53(raw)
	if !ok {
		return 0, false
	}
	if value < minimum || value > maximum {
		return 0, false
	}
	return value, true
}

// CheckDigest reports whether the member is a digest of the
// pinned sha256 form.
func CheckDigest(raw json.RawMessage) (scalar.Digest, bool) {
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

// CheckUUIDv7 reports whether the member is a lowercase UUIDv7.
func CheckUUIDv7(raw json.RawMessage) (scalar.UUIDv7, bool) {
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

// CheckTimestamp reports whether the member is a pinned
// timestamp.
func CheckTimestamp(raw json.RawMessage) (scalar.Timestamp, bool) {
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

func bytesTrimSpace(raw json.RawMessage) []byte {
	return bytes.TrimSpace(raw)
}

// reverseDNSPattern is the extensions-key grammar: at least two
// dot-separated labels, each starting with a lowercase letter.
var reverseDNSPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$`)

// CheckExtensions reports whether the member is an object whose
// every key is reverse-DNS. Extensions never add operations,
// capabilities, or trust facts; the key grammar is the part of
// that rule decidable from one body.
func CheckExtensions(raw json.RawMessage) bool {
	members, fault := DecodeStrictObject(bytesTrimSpace(raw))
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

// CheckSortedUniqueStrings reports whether the member is an array
// of strings in the item bound whose JCS encodings order strictly
// increasingly with no duplicates. Bytewise JCS order over
// strings is plain lexicographic order, so the check compares
// directly; the count bound is checked on both edges.
//
// Ordering and uniqueness are two obligations, not one: an
// unsorted-but-unique vector and a sorted-but-duplicated vector
// are separate rows in this package's sorted-strings battery
// (TestCheckSortedUniqueStringsHalves drives this function
// directly), so a check that enforced only one half could not
// pass both. The observation battery covers the digest twin,
// not this function.
func CheckSortedUniqueStrings(raw json.RawMessage, minimumLength, maximumLength int, minimumCount, maximumCount uint64) ([]string, bool) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, false
	}
	if uint64(len(elements)) < minimumCount || uint64(len(elements)) > maximumCount {
		return nil, false
	}
	values := make([]string, 0, len(elements))
	for _, element := range elements {
		value, ok := CheckStringBounds(element, minimumLength, maximumLength)
		if !ok {
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

// CheckSortedUniqueDigests reports whether the member is an array
// of digests in the count bound, ordered strictly increasingly
// with no duplicates.
func CheckSortedUniqueDigests(raw json.RawMessage, minimumCount, maximumCount uint64) ([]scalar.Digest, bool) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, false
	}
	if uint64(len(elements)) < minimumCount || uint64(len(elements)) > maximumCount {
		return nil, false
	}
	values := make([]scalar.Digest, 0, len(elements))
	for _, element := range elements {
		digest, ok := CheckDigest(element)
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

// refuse builds the package error for one refused member: the
// rule in human text automation must never branch on, naming the
// member. Tests assert the full text, so a reworded rule fails
// its row instead of passing against a stale copy.
//
// refuse is a variable, not a function, so the refusal-site audit
// (refusal_site_audit_test.go) can swap it in TestMain and record
// the production file:line behind every exercised refusal. Every
// call site invokes it directly; an aliased call records its real
// use site and fails the reverse direction of the audit.
var refuse = func(rule, member string) error {
	if member == "" {
		return fmt.Errorf("environment %s", rule)
	}
	return fmt.Errorf("environment %s: %s", rule, member)
}
