package clonebundle

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file holds parse-only helpers for every Section 13.14.1 closed
// shape. Helpers report verdicts; every refusal is constructed at the
// deciding site with a literal detail so each rule stays a distinct
// gate. Strict object decoding and extension admission delegate to the
// landed environ owner; tuple decoding delegates to sessadapter at the
// call site.

const maxUint53 = uint64(1<<53 - 1)

// maxSafeInteger is the AX safe-integer magnitude bound: extension
// values must fall in [-maxSafeInteger, maxSafeInteger], mirroring
// the canonicaljson identity guarantee.
const maxSafeInteger = int64(1<<53 - 1)

type frameFault struct {
	detail string
	member string
}

func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	members, fault := environ.DecodeStrictObject(data)
	if fault != nil {
		return nil, &frameFault{detail: fault.Detail, member: fault.Member}
	}
	return members, nil
}

func unknownMember(members map[string]json.RawMessage, allowed map[string]bool) (string, bool) {
	for name := range members {
		if !allowed[name] {
			return name, true
		}
	}
	return "", false
}

func missingMember(members map[string]json.RawMessage, required []string) (string, bool) {
	for _, name := range required {
		if _, present := members[name]; !present {
			return name, true
		}
	}
	return "", false
}

func rawString(raw json.RawMessage) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func rawBool(raw json.RawMessage) (bool, bool) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, false
	}
	return value, true
}

// checkUint53Bounds reports whether the member is a uint53 in the
// inclusive bound. Both edges are checked: a one-sided check would
// admit the open side. It delegates to the environ gate.
func checkUint53Bounds(raw json.RawMessage, minimum, maximum uint64) (uint64, bool) {
	return environ.CheckUint53Bounds(raw, minimum, maximum)
}

// stringLength is the Section 1.6 string measure in characters, not
// bytes: a bound of string[1..512] admits 512 characters of any
// width. It delegates to the single environ measure so the two can
// never drift.
func stringLength(value string) int {
	return environ.StringLength(value)
}

// checkStringBounds reports whether the member is a JSON string whose
// character count falls in the inclusive bound. It delegates to the
// environ gate.
func checkStringBounds(raw json.RawMessage, minimum, maximum int) (string, bool) {
	return environ.CheckStringBounds(raw, minimum, maximum)
}

// checkDigest reports whether the member is a digest of the pinned
// sha256 form. It delegates to the environ gate.
func checkDigest(raw json.RawMessage) (scalar.Digest, bool) {
	return environ.CheckDigest(raw)
}

// checkUUIDv7 reports whether the member is a lowercase UUIDv7. It
// delegates to the environ gate.
func checkUUIDv7(raw json.RawMessage) (scalar.UUIDv7, bool) {
	return environ.CheckUUIDv7(raw)
}

// checkTimestamp reports whether the member is a pinned timestamp.
// It delegates to the environ gate.
func checkTimestamp(raw json.RawMessage) (scalar.Timestamp, bool) {
	return environ.CheckTimestamp(raw)
}

// validText reports whether the value is valid UTF-8 text. Section
// 1.6 requires text to be valid UTF-8, and encoding/json silently
// rewrites invalid bytes to U+FFFD instead of failing — so every
// Build-side string admission runs this gate BEFORE marshaling, and
// refuses what it cannot seal byte-exactly. Without it two distinct
// inputs (e.g. "native\xff" vs "native\xfe") would seal
// byte-identical objects, and a native key would launder past the
// sanitizer's own invalid-UTF-8 refusal. Every refusal is
// constructed at the deciding site with a literal detail.
func validText(value string) bool {
	return utf8.ValidString(value)
}

// validExtensionText reports whether every string under an
// extensions value — keys and string values at any depth, through
// maps, slices, arrays, pointers, interfaces, and exported struct
// fields — is valid UTF-8 text. The build helper runs it before
// marshaling for the same rewrite-and-continue reason as validText:
// a single invalid byte anywhere in the map would otherwise seal as
// U+FFFD. Numbers, booleans, and nulls cross opaquely; values whose
// marshaled form is computed rather than textual (custom
// json.Marshaler implementations) are trusted to render valid text.
func validExtensionText(value any) bool {
	return extensionTextValid(reflect.ValueOf(value))
}

func extensionTextValid(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.String:
		return validText(value.String())
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			if !extensionTextValid(iter.Key()) || !extensionTextValid(iter.Value()) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if !extensionTextValid(value.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		own := value.Type()
		for index := 0; index < value.NumField(); index++ {
			if own.Field(index).PkgPath != "" {
				continue
			}
			if !extensionTextValid(value.Field(index)) {
				return false
			}
		}
		return true
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return true
		}
		return extensionTextValid(value.Elem())
	default:
		return true
	}
}

func bytesTrimSpace(raw json.RawMessage) json.RawMessage {
	return json.RawMessage(bytes.TrimSpace(raw))
}

func isNull(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return string(trimmed) == "null"
}

func decodeArray(raw json.RawMessage) ([]json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil, false
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '[' {
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
	token, err = decoder.Token()
	if err != nil {
		return nil, false
	}
	delimiter, ok = token.(json.Delim)
	if !ok || delimiter != ']' {
		return nil, false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, false
	}
	if elements == nil {
		elements = []json.RawMessage{}
	}
	return elements, true
}

// checkSortedUniqueStrings reports whether the member is an array of
// strings in the item bound whose JCS encodings order strictly
// increasingly with no duplicates. It delegates to the environ gate.
func checkSortedUniqueStrings(raw json.RawMessage, minimumLength, maximumLength int, minimumCount, maximumCount uint64) ([]string, bool) {
	return environ.CheckSortedUniqueStrings(raw, minimumLength, maximumLength, minimumCount, maximumCount)
}

// checkSortedUniqueDigests reports whether the member is a sorted
// unique digest array in the count bound. It delegates to the
// environ gate.
func checkSortedUniqueDigests(raw json.RawMessage, minimumCount, maximumCount uint64) ([]scalar.Digest, bool) {
	return environ.CheckSortedUniqueDigests(raw, minimumCount, maximumCount)
}

// checkExtensions reports whether the member is an object whose keys
// are all reverse-DNS names. It delegates to the environ gate. Key
// admission alone is not the closed extensions rule for this
// package: extension values flow into the omit-self identity, so
// every admission site uses checkExtensionsClosed, which additionally
// enforces the AX number model and nested duplicate refusal on the
// values.
func checkExtensions(raw json.RawMessage) bool {
	return environ.CheckExtensions(raw)
}

// maxExtensionDepth mirrors the canonicaljson nesting bound: a
// container nested deeper than this inside an extension value is
// refused, exactly as the identity owner refuses it.
const maxExtensionDepth = 256

// checkExtensionsClosed enforces the full closed extensions rule:
// reverse-DNS keys (via the environ owner) and values inside the AX
// common logical data model (integer literals only, |n| <= 2^53-1,
// no fraction or exponent, nested objects duplicate-free at every
// depth). It returns nil when the member is closed, otherwise the
// refusal detail naming the failed rule.
func checkExtensionsClosed(raw json.RawMessage) error {
	if !checkExtensions(raw) {
		return errExtensionsKeys
	}
	return checkExtensionValues(raw)
}

var (
	errExtensionsKeys     = errorString("are not reverse-DNS keyed")
	errExtensionNumber    = errorString("carry a number outside the AX safe-integer model")
	errExtensionDuplicate = errorString("carry a duplicate nested member")
	errExtensionShape     = errorString("are not a depth-bounded extension object")
	errPayloadShape       = errorString("are not a depth-bounded payload object")
)

// errorString is one fixed refusal detail. Extension faults name the
// failed rule; the deciding site names the owner.
type errorString string

func (detail errorString) Error() string { return string(detail) }

// checkExtensionValues validates every value under the extensions
// object against the AX number model with nested objects
// duplicate-free. It mirrors the canonicaljson identity guarantee
// (safe integers, no rounding, strict objects) which Canonicalize
// alone does not carry, so no value that would round or collapse
// under canonicalization is ever admitted into an omit-self
// identity.
func checkExtensionValues(raw json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(bytesTrimSpace(raw)))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return errExtensionShape
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return errExtensionShape
	}
	if err := checkExtensionMembers(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errExtensionShape
	}
	return nil
}

// checkExtensionMembers validates the members of one open extension
// object: names are duplicate-free and every value validates.
func checkExtensionMembers(decoder *json.Decoder, depth int) error {
	if depth >= maxExtensionDepth {
		return errExtensionShape
	}
	seen := map[string]bool{}
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return errExtensionShape
		}
		name, ok := nameToken.(string)
		if !ok {
			return errExtensionShape
		}
		if seen[name] {
			return errExtensionDuplicate
		}
		seen[name] = true
		if err := checkExtensionValue(decoder, depth+1); err != nil {
			return err
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return errExtensionShape
	}
	return nil
}

// checkExtensionValue validates one extension value: containers
// recurse (objects duplicate-free at every depth, arrays in order),
// numbers obey the AX literal rule, and strings, booleans, and null
// cross opaquely.
func checkExtensionValue(decoder *json.Decoder, depth int) error {
	token, err := decoder.Token()
	if err != nil {
		return errExtensionShape
	}
	switch typed := token.(type) {
	case json.Delim:
		switch typed {
		case '{':
			return checkExtensionMembers(decoder, depth)
		case '[':
			if depth >= maxExtensionDepth {
				return errExtensionShape
			}
			for decoder.More() {
				if err := checkExtensionValue(decoder, depth+1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim(']') {
				return errExtensionShape
			}
			return nil
		default:
			return errExtensionShape
		}
	case json.Number:
		if !checkAXNumberLiteral(typed.String()) {
			return errExtensionNumber
		}
		return nil
	case string, bool, nil:
		return nil
	default:
		return errExtensionShape
	}
}

// checkAXNumberLiteral mirrors the canonicaljson safe-integer rule:
// a literal with a fraction or exponent is forbidden, and the
// integer value must fall in [-2^53+1, 2^53-1]. The bound is checked
// on the literal before any host conversion, so a value the host
// would round is refused instead.
func checkAXNumberLiteral(literal string) bool {
	if strings.ContainsAny(literal, ".eE") {
		return false
	}
	integer, err := strconv.ParseInt(literal, 10, 64)
	if err != nil || integer < -maxSafeInteger || integer > maxSafeInteger {
		return false
	}
	return true
}

// canonicalizeObject renders one closed object deterministically: the
// members marshal in any order and the JCS transform sorts them, so
// identical logical inputs always produce identical bytes.
func canonicalizeObject(object map[string]any) ([]byte, error) {
	plain, err := json.Marshal(object)
	if err != nil {
		return nil, invalid("serialize clone bundle object: %v", err)
	}
	canonical, err := canonicaljson.Canonicalize(plain)
	if err != nil {
		return nil, invalid("canonicalize clone bundle object: %v", err)
	}
	return canonical, nil
}

// omitSelfDigest computes the JCS SHA-256 identity of members with
// only the self field omitted. It mirrors the canonicaljson
// omit-self construction without routing through that package's
// closed-shape registry, which refuses these schemas by design (see
// doc.go).
func omitSelfDigest(members map[string]json.RawMessage, selfField string) (scalar.Digest, error) {
	omitted := make(map[string]any, len(members))
	for name, raw := range members {
		if name == selfField {
			continue
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return scalar.Digest{}, invalid("identity input member %s is not JSON: %v", name, err)
		}
		omitted[name] = value
	}
	plain, err := json.Marshal(omitted)
	if err != nil {
		return scalar.Digest{}, invalid("serialize omit-self object: %v", err)
	}
	canonical, err := canonicaljson.Canonicalize(plain)
	if err != nil {
		return scalar.Digest{}, invalid("canonicalize omit-self object: %v", err)
	}
	return scalar.SHA256Digest(canonical), nil
}

// verifySelfDigest recomputes the omit-self identity and refuses a
// mismatch against the claimed self digest.
func verifySelfDigest(members map[string]json.RawMessage, selfField string, claimed scalar.Digest) error {
	calculated, err := omitSelfDigest(members, selfField)
	if err != nil {
		return err
	}
	if calculated != claimed {
		return invalid("%s claim %q does not match omit-self digest %q", selfField, claimed.String(), calculated.String())
	}
	return nil
}
