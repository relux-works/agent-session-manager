package clonereadback

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file holds parse-only helpers for the two Section 13.14.2
// validation closed shapes. Helpers report verdicts; every refusal
// is constructed at the deciding site with a literal detail so each
// rule stays a distinct gate. Strict object decoding, scalar
// bounds, sorted-unique arrays, and the character measure delegate
// to the landed environ owner; closed extension values and the
// omit-self digest delegate to the clonebundle owner; tuples,
// workspace bindings, and findings decode through their owners at
// the call site.

const maxUint53 = uint64(1<<53 - 1)

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

func memberField(member string) string {
	if member == "" {
		return "frame"
	}
	return "member " + member
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
	// encoding/json unmarshals null into a string without an error,
	// so null is rejected explicitly: every string member here is
	// required non-null.
	if isNull(raw) {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func rawBool(raw json.RawMessage) (bool, bool) {
	// encoding/json unmarshals null into a bool without an error,
	// so null is rejected explicitly: a missing boolean is never a
	// false one.
	if isNull(raw) {
		return false, false
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, false
	}
	return value, true
}

// stringLength is the Section 1.6 string measure in characters, not
// bytes. It delegates to the single environ measure so the two can
// never drift.
func stringLength(value string) int {
	return environ.StringLength(value)
}

// checkStringBounds reports whether the member is a JSON string
// whose character count falls in the inclusive bound. It delegates
// to the environ gate.
func checkStringBounds(raw json.RawMessage, minimum, maximum int) (string, bool) {
	return environ.CheckStringBounds(raw, minimum, maximum)
}

// checkUint53Bounds reports whether the member is a uint53 in the
// inclusive bound. It delegates to the environ gate.
func checkUint53Bounds(raw json.RawMessage, minimum, maximum uint64) (uint64, bool) {
	return environ.CheckUint53Bounds(raw, minimum, maximum)
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

// checkSortedUniqueStrings reports whether the member is an array of
// strings in the item bound whose order is strictly increasing with
// no duplicates. It delegates to the environ gate.
func checkSortedUniqueStrings(raw json.RawMessage, minimumLength, maximumLength int, minimumCount, maximumCount uint64) ([]string, bool) {
	return environ.CheckSortedUniqueStrings(raw, minimumLength, maximumLength, minimumCount, maximumCount)
}

func isNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
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

// validText reports whether the value is valid UTF-8 text. Section
// 1.6 requires text to be valid UTF-8, and encoding/json silently
// rewrites invalid bytes to U+FFFD instead of failing — so every
// Build-side string admission runs this gate BEFORE marshaling, and
// refuses what it cannot seal byte-exactly.
func validText(value string) bool {
	return utf8.ValidString(value)
}

// checkExtensionsClosed enforces the full closed extensions rule via
// the clonebundle owner: reverse-DNS keys and values inside the AX
// common logical data model. It returns nil when the member is
// closed, otherwise the refusal detail naming the failed rule.
func checkExtensionsClosed(raw json.RawMessage) error {
	return clonebundle.CheckExtensionsClosed(raw)
}

// canonicalizeObject renders one closed object deterministically:
// the members marshal in any order and the JCS transform sorts
// them, so identical logical inputs always produce identical bytes.
func canonicalizeObject(object map[string]any) ([]byte, error) {
	plain, err := json.Marshal(object)
	if err != nil {
		return nil, invalid("serialize validation object: %v", err)
	}
	canonical, err := canonicaljson.Canonicalize(plain)
	if err != nil {
		return nil, invalid("canonicalize validation object: %v", err)
	}
	return canonical, nil
}

// verifySelfDigest recomputes the omit-self identity through the
// clonebundle owner and refuses a mismatch against the claimed self
// digest.
func verifySelfDigest(members map[string]json.RawMessage, selfField string, claimed scalar.Digest) error {
	calculated, err := clonebundle.OmitSelfDigest(members, selfField)
	if err != nil {
		return err
	}
	if calculated != claimed {
		return invalid("%s claim %q does not match omit-self digest %q", selfField, claimed.String(), calculated.String())
	}
	return nil
}

// checkSortedUniqueBoundedStrings reports whether the
// caller-supplied Build input is sorted unique bounded strings in
// the count bound. It mirrors the decode string-array gate for Go
// inputs: UTF-8 first (so nothing seals rewritten), then character
// bounds, then strict order.
func checkSortedUniqueBoundedStrings(values []string, minimumLength, maximumLength int, minimumCount, maximumCount int, owner, member string) ([]string, error) {
	if len(values) < minimumCount || len(values) > maximumCount {
		return nil, invalid("%s %s carries %d items, want [%d..%d]", owner, member, len(values), minimumCount, maximumCount)
	}
	for index, value := range values {
		if !validText(value) {
			return nil, invalid("%s %s[%d] is not valid UTF-8", owner, member, index)
		}
		if length := stringLength(value); length < minimumLength || length > maximumLength {
			return nil, invalid("%s %s[%d] is not a string[%d..%d]", owner, member, index, minimumLength, maximumLength)
		}
		if index > 0 && value <= values[index-1] {
			return nil, invalid("%s %s are not sorted unique", owner, member)
		}
	}
	return append([]string(nil), values...), nil
}

// stringValues renders strings for the sealed object.
func stringValues(values []string) []any {
	rendered := make([]any, 0, len(values))
	for _, value := range values {
		rendered = append(rendered, value)
	}
	return rendered
}
