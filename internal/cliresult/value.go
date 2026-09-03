package cliresult

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// The helpers below read one closed object member at a time. They exist so that
// every member of every Section 14.2 shape is checked by the same code on the
// writing and the reading side: New canonicalizes its caller's body into the
// same value model Decode produces, and both then run the identical validator.
// A writer that could emit an object its own reader refuses is the defect this
// arrangement removes rather than documents.

// requireClosedMembers enforces a closed object: exactly the declared members,
// no absent one and no extra one. Section 1.6 is explicit that "every other
// embedded object is closed: an unknown member MUST be rejected even when its
// containing top-level object is otherwise valid".
func requireClosedMembers(object map[string]any, where string, members []string) error {
	declared := make(map[string]struct{}, len(members))
	for _, member := range members {
		declared[member] = struct{}{}
		if _, present := object[member]; !present {
			return failf("%s is missing required member %q", where, member)
		}
	}
	for _, key := range sortedKeys(object) {
		if _, ok := declared[key]; !ok {
			return failf("%s carries unknown member %q", where, key)
		}
	}
	return nil
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func memberObject(object map[string]any, where, name string) (map[string]any, error) {
	nested, ok := object[name].(map[string]any)
	if !ok {
		return nil, failf("%s.%s is not a JSON object", where, name)
	}
	return nested, nil
}

func memberArray(object map[string]any, where, name string) ([]any, error) {
	members, ok := object[name].([]any)
	if !ok {
		return nil, failf("%s.%s is not a JSON array", where, name)
	}
	return members, nil
}

func memberBool(object map[string]any, where, name string) (bool, error) {
	value, ok := object[name].(bool)
	if !ok {
		return false, failf("%s.%s is not a JSON boolean", where, name)
	}
	return value, nil
}

// requireTrue enforces a member the pinned document fixes to the literal true.
func requireTrue(object map[string]any, where, name string) error {
	value, err := memberBool(object, where, name)
	if err != nil {
		return err
	}
	if !value {
		return failf("%s.%s must be true", where, name)
	}
	return nil
}

// requireFalse enforces a member the pinned document fixes to the literal false.
func requireFalse(object map[string]any, where, name string) error {
	value, err := memberBool(object, where, name)
	if err != nil {
		return err
	}
	if value {
		return failf("%s.%s must be false", where, name)
	}
	return nil
}

// memberString reads a string member and bounds it in UTF-8 characters, which
// is the unit Section 1.6 declares for string[n..m].
func memberString(object map[string]any, where, name string, minimum, maximum int) (string, error) {
	value, ok := object[name].(string)
	if !ok {
		return "", failf("%s.%s is not a JSON string", where, name)
	}
	if err := boundString(value, fmt.Sprintf("%s.%s", where, name), minimum, maximum); err != nil {
		return "", err
	}
	return value, nil
}

func boundString(value, where string, minimum, maximum int) error {
	if !utf8.ValidString(value) {
		return failf("%s is not valid UTF-8", where)
	}
	count := utf8.RuneCountInString(value)
	if count < minimum || count > maximum {
		return failf("%s is %d UTF-8 characters, the bound is %d..%d", where, count, minimum, maximum)
	}
	return nil
}

// memberEnum reads a case-sensitive lower-snake-case enumeration member and
// refuses any value outside the closed vocabulary the section declares.
func memberEnum(object map[string]any, where, name string, allowed ...string) (string, error) {
	value, ok := object[name].(string)
	if !ok {
		return "", failf("%s.%s is not a JSON string", where, name)
	}
	if _, err := scalar.ParseClosedEnum(value, allowed...); err != nil {
		return "", failf("%s.%s %q is not one of %s", where, name, value, strings.Join(allowed, "|"))
	}
	return value, nil
}

// memberUint53 reads a uint53 member and refuses a value below the declared
// floor. Section 14.2 writes lease_epoch as uint53>0, so the floor is a
// parameter rather than a special case bolted on at each call site.
func memberUint53(object map[string]any, where, name string, floor uint64) (uint64, error) {
	number, ok := object[name].(json.Number)
	if !ok {
		return 0, failf("%s.%s is not a JSON number", where, name)
	}
	text := number.String()
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, failf("%s.%s %s is not a uint53", where, name, text)
	}
	if _, err := scalar.NewUint53(value); err != nil {
		return 0, failf("%s.%s %s is outside the safe-integer interval", where, name, text)
	}
	if value < floor {
		return 0, failf("%s.%s is %d, the bound is >= %d", where, name, value, floor)
	}
	return value, nil
}

// memberInt32OrNull reads an int32|null member. Section 14.2 uses it only for
// provider_exit_code, where a negative value is a real signal-derived status.
func memberInt32OrNull(object map[string]any, where, name string) error {
	if object[name] == nil {
		return nil
	}
	number, ok := object[name].(json.Number)
	if !ok {
		return failf("%s.%s is neither null nor a JSON number", where, name)
	}
	if _, err := strconv.ParseInt(number.String(), 10, 32); err != nil {
		return failf("%s.%s %s is not an int32", where, name, number.String())
	}
	return nil
}

func memberUUIDv7(object map[string]any, where, name string) (scalar.UUIDv7, error) {
	value, ok := object[name].(string)
	if !ok {
		return scalar.UUIDv7{}, failf("%s.%s is not a JSON string", where, name)
	}
	parsed, err := scalar.ParseUUIDv7(value)
	if err != nil {
		return scalar.UUIDv7{}, failf("%s.%s: %v", where, name, err)
	}
	return parsed, nil
}

// memberUUIDv7OrNull reads a required UUIDv7|null member. Section 1.6 states
// that "a required T|null field MUST be present and MAY contain JSON null", so
// an absent member is refused by requireClosedMembers before this runs.
func memberUUIDv7OrNull(object map[string]any, where, name string) (scalar.UUIDv7, bool, error) {
	if object[name] == nil {
		return scalar.UUIDv7{}, false, nil
	}
	parsed, err := memberUUIDv7(object, where, name)
	if err != nil {
		return scalar.UUIDv7{}, false, err
	}
	return parsed, true, nil
}

func memberUUIDv4(object map[string]any, where, name string) error {
	value, ok := object[name].(string)
	if !ok {
		return failf("%s.%s is not a JSON string", where, name)
	}
	if _, err := scalar.ParseUUIDv4(value); err != nil {
		return failf("%s.%s: %v", where, name, err)
	}
	return nil
}

func memberDigest(object map[string]any, where, name string) error {
	value, ok := object[name].(string)
	if !ok {
		return failf("%s.%s is not a JSON string", where, name)
	}
	if _, err := scalar.ParseDigest(value); err != nil {
		return failf("%s.%s: %v", where, name, err)
	}
	return nil
}

// memberDigestOrNull reads a required digest|null member and reports whether a
// digest is present, so a caller can enforce a cross-member rule on it.
func memberDigestOrNull(object map[string]any, where, name string) (bool, error) {
	if object[name] == nil {
		return false, nil
	}
	if err := memberDigest(object, where, name); err != nil {
		return false, err
	}
	return true, nil
}

func memberTimestampOrNull(object map[string]any, where, name string) error {
	if object[name] == nil {
		return nil
	}
	value, ok := object[name].(string)
	if !ok {
		return failf("%s.%s is neither null nor a JSON string", where, name)
	}
	if _, err := scalar.ParseTimestamp(value); err != nil {
		return failf("%s.%s: %v", where, name, err)
	}
	return nil
}

func memberStringOrNull(object map[string]any, where, name string, minimum, maximum int) error {
	if object[name] == nil {
		return nil
	}
	if _, err := memberString(object, where, name, minimum, maximum); err != nil {
		return err
	}
	return nil
}

// elementValidator checks one array element and returns the value the array's
// ordering is keyed on. A type whose elements carry no ordering key returns the
// empty string, and requireSortedUnique then only bounds the array.
type elementValidator func(where string, element any) (string, error)

// requireSortedUnique enforces an array bound and, when the element validator
// yields an ordering key, the Section 14.2 rule that "digest arrays and object
// arrays keyed by an ID are sorted bytewise by that ID" with no duplicate.
//
// The comparison is bytewise on the key string, which is what "sorted bytewise"
// names. Go's default string ordering is bytewise, so no separate collation
// step is derived here.
func requireSortedUnique(
	object map[string]any,
	where, name string,
	minimum, maximum int,
	validate elementValidator,
) error {
	members, err := memberArray(object, where, name)
	if err != nil {
		return err
	}
	if len(members) < minimum || len(members) > maximum {
		return failf("%s.%s has %d members, the bound is %d..%d", where, name, len(members), minimum, maximum)
	}
	previous := ""
	ordered := false
	for index, element := range members {
		key, err := validate(fmt.Sprintf("%s.%s[%d]", where, name, index), element)
		if err != nil {
			return err
		}
		if key == "" {
			continue
		}
		if ordered {
			if key == previous {
				return failf("%s.%s repeats %q", where, name, key)
			}
			if key < previous {
				return failf("%s.%s is not sorted bytewise: %q follows %q", where, name, key, previous)
			}
		}
		previous = key
		ordered = true
	}
	return nil
}

// requireUnorderedArray bounds an array whose element order the pinned document
// does not fix. It is separate from requireSortedUnique so that an ordering
// rule is never applied to a shape that does not declare one: inventing a sort
// where the specification is silent is the same class of defect as omitting one
// where it is not.
func requireUnorderedArray(
	object map[string]any,
	where, name string,
	minimum, maximum int,
	validate elementValidator,
) error {
	members, err := memberArray(object, where, name)
	if err != nil {
		return err
	}
	if len(members) < minimum || len(members) > maximum {
		return failf("%s.%s has %d members, the bound is %d..%d", where, name, len(members), minimum, maximum)
	}
	for index, element := range members {
		if _, err := validate(fmt.Sprintf("%s.%s[%d]", where, name, index), element); err != nil {
			return err
		}
	}
	return nil
}

func stringElement(minimum, maximum int) elementValidator {
	return func(where string, element any) (string, error) {
		value, ok := element.(string)
		if !ok {
			return "", failf("%s is not a JSON string", where)
		}
		if err := boundString(value, where, minimum, maximum); err != nil {
			return "", err
		}
		return value, nil
	}
}

func uuidv7Element(where string, element any) (string, error) {
	value, ok := element.(string)
	if !ok {
		return "", failf("%s is not a JSON string", where)
	}
	if _, err := scalar.ParseUUIDv7(value); err != nil {
		return "", failf("%s: %v", where, err)
	}
	return value, nil
}

func digestElement(where string, element any) (string, error) {
	value, ok := element.(string)
	if !ok {
		return "", failf("%s is not a JSON string", where)
	}
	if _, err := scalar.ParseDigest(value); err != nil {
		return "", failf("%s: %v", where, err)
	}
	return value, nil
}

// objectElement adapts a closed-object validator to an array element and
// reports the ordering key that validator selected.
func objectElement(validate func(where string, object map[string]any) (string, error)) elementValidator {
	return func(where string, element any) (string, error) {
		object, ok := element.(map[string]any)
		if !ok {
			return "", failf("%s is not a JSON object", where)
		}
		return validate(where, object)
	}
}
