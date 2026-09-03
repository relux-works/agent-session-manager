package cliresult

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// documentWithout removes one top-level member from the normative example.
func documentWithout(t *testing.T, member string) []byte {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(normativeCLISuccess), &object); err != nil {
		t.Fatalf("unmarshal example: %v", err)
	}
	delete(object, member)
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return encoded
}

// documentWith replaces or adds one top-level member of the normative example.
func documentWith(t *testing.T, member string, raw string) []byte {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(normativeCLISuccess), &object); err != nil {
		t.Fatalf("unmarshal example: %v", err)
	}
	object[member] = json.RawMessage(raw)
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return encoded
}

// TestTopLevelObjectIsClosed narrows the closed-envelope gate one member at a
// time. Each of the eight declared members is required, and a ninth member is
// refused rather than ignored - Section 1.6 calls that fail-closed rule the
// protection against "silently ignoring a new ownership or security control".
func TestTopLevelObjectIsClosed(t *testing.T) {
	for _, member := range []string{
		"schema", "schema_version", "command", "ok", "operation_id", "session_id", "body", "extensions",
	} {
		t.Run("missing/"+member, func(t *testing.T) {
			if _, err := Decode(Version100, documentWithout(t, member)); err == nil {
				t.Fatalf("Decode admitted a document without %q", member)
			}
		})
	}
	for _, member := range []string{"error", "warning", "authority", "exit_code", "retryable", "Ok"} {
		t.Run("unknown/"+member, func(t *testing.T) {
			_, err := Decode(Version100, documentWith(t, member, `"smuggled"`))
			if err == nil || !strings.Contains(err.Error(), "unknown top-level member") {
				t.Fatalf("Decode(%q) = %v, want an unknown-member refusal", member, err)
			}
		})
	}
}

// TestFailureShapedSuccessIsRefused pins the Section 14.2 sentence "failure
// output is one Structured Error object from Section 15.1, not a CLI Result
// with ok = false". A reader that returned an ok=false document as a result
// would hand a caller a failure wearing a success type.
func TestFailureShapedSuccessIsRefused(t *testing.T) {
	_, err := Decode(Version100, documentWith(t, "ok", "false"))
	if err == nil || !strings.Contains(err.Error(), "ok is false") {
		t.Fatalf("Decode(ok=false) = %v, want a refusal", err)
	}
	for _, raw := range []string{`"true"`, `1`, `null`, `{}`} {
		if _, err := Decode(Version100, documentWith(t, "ok", raw)); err == nil {
			t.Fatalf("Decode admitted ok=%s", raw)
		}
	}
}

// TestSchemaIdentityIsSettledBeforeAnythingElseIsTrusted proves the refusal
// ordering: a document with a wrong schema or an unsupported major is refused
// even when every other member is malformed, so nothing downstream of the
// identity check can have been consulted on the way to that refusal.
func TestSchemaIdentityIsSettledBeforeAnythingElseIsTrusted(t *testing.T) {
	poisoned := func(t *testing.T, member, raw string) []byte {
		t.Helper()
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(normativeCLISuccess), &object); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		object[member] = json.RawMessage(raw)
		object["command"] = json.RawMessage(`"not-a-command"`)
		object["body"] = json.RawMessage(`{"forged":true}`)
		object["operation_id"] = json.RawMessage(`"not-a-uuid"`)
		encoded, err := json.Marshal(object)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return encoded
	}
	if _, err := Decode(Version100, poisoned(t, "schema", `"urn:ax:schema:error"`)); err == nil ||
		!strings.Contains(err.Error(), "schema is not urn:ax:schema:cli-result") {
		t.Fatalf("wrong schema = %v, want a schema refusal", err)
	}
	_, err := Decode(Version100, poisoned(t, "schema_version", `"2.0.0"`))
	if !errors.Is(err, ErrUnsupportedMajor) {
		t.Fatalf("major 2 read as major 1 = %v, want ErrUnsupportedMajor", err)
	}
	_, err = Decode(Version200, poisoned(t, "schema_version", `"1.0.0"`))
	if !errors.Is(err, ErrUnsupportedMajor) {
		t.Fatalf("major 1 read as major 2 = %v, want ErrUnsupportedMajor", err)
	}
}

// TestUnsupportedMajorIsRejectedRatherThanParsed pins Section 17.2 rule 1 for
// every major the CLI Result registry does not carry as well as for the ones it
// does but this reader is not bound to.
func TestUnsupportedMajorIsRejectedRatherThanParsed(t *testing.T) {
	for _, version := range []string{"0.9.0", "2.0.0", "3.0.0", "4.0.0", "9.9.9"} {
		t.Run(version, func(t *testing.T) {
			_, err := Decode(Version100, documentWith(t, "schema_version", `"`+version+`"`))
			if !errors.Is(err, ErrUnsupportedMajor) {
				t.Fatalf("Decode(%s) = %v, want ErrUnsupportedMajor", version, err)
			}
		})
	}
}

// TestAcceptsVersionImplementsTheSameOrLowerSupportedMinorRule pins Section
// 17.2 rule 2 as a function of its own.
//
// The pinned CLI Result registry carries no two versions sharing a major, so
// Decode cannot reach the lower-minor arm today; the comparison is therefore
// tested directly rather than claimed through a path the registry cannot
// exercise. That limit is stated in RefusalBound.
func TestAcceptsVersionImplementsTheSameOrLowerSupportedMinorRule(t *testing.T) {
	cases := []struct {
		expected, candidate Version
		accepted            bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.2.0", "1.1.0", true},
		{"1.2.0", "1.2.0", true},
		{"1.2.3", "1.2.2", true},
		{"1.2.0", "1.3.0", false},
		{"1.2.0", "1.2.1", false},
		{"1.0.0", "2.0.0", false},
		{"2.0.0", "1.0.0", false},
		{"1.0.0", "1.0", false},
		{"", "1.0.0", false},
	}
	for _, testCase := range cases {
		if got := acceptsVersion(testCase.expected, testCase.candidate); got != testCase.accepted {
			t.Fatalf("acceptsVersion(%q, %q) = %t, want %t",
				testCase.expected, testCase.candidate, got, testCase.accepted)
		}
	}
}

// TestCommandTagAndDocumentVersionMustAgree pins the Section 14.2 sentence "no
// command may emit another registered version or retry a different major after
// parsing begins". A clone tag inside a 1.0.0 envelope is refused rather than
// reinterpreted under 2.0.0.
func TestCommandTagAndDocumentVersionMustAgree(t *testing.T) {
	_, err := Decode(Version100, documentWith(t, "command", `"session.clone.run"`))
	if err == nil || !strings.Contains(err.Error(), "selects CLI Result 2.0.0") {
		t.Fatalf("Decode(clone tag in 1.0.0) = %v, want a selection refusal", err)
	}
	_, err = Decode(Version100, documentWith(t, "command", `"terminal.backend.list"`))
	if err == nil || !strings.Contains(err.Error(), "selects CLI Result 4.0.0") {
		t.Fatalf("Decode(terminal tag in 1.0.0) = %v, want a selection refusal", err)
	}

	// The decisive case is a tag whose body this repository does build carried
	// inside another registered version: nothing else in the reader refuses it,
	// so only the selection rule can. A 1.0.0 takeover body inside a 2.0.0
	// envelope is a complete, otherwise valid object.
	promoted := documentWith(t, "schema_version", `"2.0.0"`)
	_, err = Decode(Version200, promoted)
	if err == nil {
		t.Fatalf("Decode admitted a 1.0.0 takeover body inside a 2.0.0 envelope")
	}
	if !strings.Contains(err.Error(), `command "takeover" selects CLI Result 1.0.0, the document is 2.0.0`) {
		t.Fatalf("Decode = %v, want the selection refusal", err)
	}
}

// TestRequiredNullableIdentifiersMustBePresent narrows the Section 1.6 rule that
// "a required T|null field MUST be present and MAY contain JSON null". An
// omitted identifier is refused, an explicit null is admitted, and a malformed
// one is refused.
func TestRequiredNullableIdentifiersMustBePresent(t *testing.T) {
	spec := validSpec(t, CommandCancel)
	result := mustResult(t, spec)
	encoded, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(encoded), `"operation_id":null`) ||
		!strings.Contains(string(encoded), `"session_id":null`) {
		t.Fatalf("encoder omitted a required nullable identifier: %s", encoded)
	}
	if _, err := Decode(Version100, encoded); err != nil {
		t.Fatalf("Decode of an explicit-null document: %v", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	delete(object, "session_id")
	trimmed, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := Decode(Version100, trimmed); err == nil {
		t.Fatalf("Decode admitted a document with an omitted session_id")
	}
	for _, raw := range []string{`"not-a-uuid"`, `5`, `{}`, `"0198f4c8-3e70-4a11-8a2b-1234567890ab"`} {
		if _, err := Decode(Version100, documentWith(t, "session_id", raw)); err == nil {
			t.Fatalf("Decode admitted session_id=%s", raw)
		}
	}
}

// TestDuplicateAndMalformedDocumentsAreRefused proves that the reader runs the
// strict common data model over the whole document before it reads any member:
// a duplicate key, a float, an integer at or beyond 2^53, and trailing content
// are each refused.
func TestDuplicateAndMalformedDocumentsAreRefused(t *testing.T) {
	cases := map[string]string{
		"duplicate key":    strings.Replace(normativeCLISuccess, `"ok": true,`, `"ok": true, "ok": true,`, 1),
		"floating point":   strings.Replace(normativeCLISuccess, `"lease_epoch": 5`, `"lease_epoch": 5.5`, 1),
		"unsafe integer":   strings.Replace(normativeCLISuccess, `"lease_epoch": 5`, `"lease_epoch": 9007199254740992`, 1),
		"trailing content": normativeCLISuccess + "\n{}",
		"not an object":    `["schema"]`,
		"empty":            ``,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(Version100, []byte(document)); err == nil {
				t.Fatalf("Decode admitted %s", name)
			}
		})
	}
}

// TestExtensionsGateNarrowsEachDeclaredBound refuses one step past each Section
// 1.6 bound and admits exactly at it, so a bound that was widened rather than
// deleted still fails.
func TestExtensionsGateNarrowsEachDeclaredBound(t *testing.T) {
	admitted := []any{
		map[string]any{"works.relux.ax.note": "kept"},
		map[string]any{"a.b": nil},
		// Four opened containers is the limit; the fifth is refused below.
		map[string]any{"works.relux.ax.depth": map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": "at the limit"}}}}},
		map[string]any{"works.relux.ax.array": []any{[]any{[]any{[]any{"at the limit"}}}}},
		map[string]any{"works.relux.ax.count": json.Number("9007199254740991")},
	}
	for index, extensions := range admitted {
		spec := validSpec(t, CommandCancel)
		spec.Extensions = extensions
		if _, err := New(spec); err != nil {
			t.Fatalf("admitted case %d refused: %v", index, err)
		}
	}
	refused := map[string]any{
		"no dot":           map[string]any{"works": "x"},
		"uppercase label":  map[string]any{"works.Relux": "x"},
		"leading digit":    map[string]any{"1works.relux": "x"},
		"too short":        map[string]any{"a.": "x"},
		"depth five":       map[string]any{"a.b": map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": map[string]any{"e": "past the limit"}}}}}},
		"depth five array": map[string]any{"a.b": []any{[]any{[]any{[]any{[]any{"past the limit"}}}}}},
		"depth five mixed": map[string]any{"a.b": []any{map[string]any{"a": []any{map[string]any{"b": []any{"past the limit"}}}}}},
	}
	for name, extensions := range refused {
		t.Run(name, func(t *testing.T) {
			spec := validSpec(t, CommandCancel)
			spec.Extensions = extensions
			if _, err := New(spec); err == nil {
				t.Fatalf("New admitted extensions %s", name)
			}
		})
	}
	t.Run("sixty five keys", func(t *testing.T) {
		atLimit := map[string]any{}
		for index := 0; index < 64; index++ {
			atLimit[extensionKey(index)] = "x"
		}
		spec := validSpec(t, CommandCancel)
		spec.Extensions = atLimit
		if _, err := New(spec); err != nil {
			t.Fatalf("64 keys refused: %v", err)
		}
		atLimit[extensionKey(64)] = "x"
		spec = validSpec(t, CommandCancel)
		spec.Extensions = atLimit
		if _, err := New(spec); err == nil {
			t.Fatalf("New admitted 65 extension keys")
		}
	})
	t.Run("canonical byte bound", func(t *testing.T) {
		spec := validSpec(t, CommandCancel)
		spec.Extensions = map[string]any{"works.relux.ax.blob": strings.Repeat("a", 65_600)}
		if _, err := New(spec); err == nil {
			t.Fatalf("New admitted an extensions object past 65,536 canonical bytes")
		}
	})
}

func extensionKey(index int) string {
	return "works.relux.ax" + strings.Repeat("a", 1+index%40) + "." + string(rune('a'+index%26)) + string(rune('a'+index/26))
}

// TestUnknownExtensionsSurviveTheRoundTripAsInertData proves that an extension
// this build does not understand is preserved and never consulted: no accessor
// changes behaviour because of it.
func TestUnknownExtensionsSurviveTheRoundTripAsInertData(t *testing.T) {
	spec := validSpec(t, CommandCancel)
	spec.Extensions = map[string]any{
		"com.example.unknown": map[string]any{"nested": []any{"a", json.Number("1"), true, nil}},
	}
	result := mustResult(t, spec)
	encoded, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	decoded, err := Decode(Version100, encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if keys := decoded.ExtensionKeys(); len(keys) != 1 || keys[0] != "com.example.unknown" {
		t.Fatalf("extension keys = %v", keys)
	}
	value, ok := decoded.Extension("com.example.unknown")
	if !ok {
		t.Fatalf("extension lost")
	}
	// The value is inert: writing through the returned copy cannot reach the
	// validated object.
	value.(map[string]any)["nested"] = "overwritten"
	again, _ := decoded.Extension("com.example.unknown")
	if _, still := again.(map[string]any)["nested"].([]any); !still {
		t.Fatalf("Extension handed out the live container")
	}
	if decoded.Command() != CommandCancel {
		t.Fatalf("an extension changed the command tag")
	}
}
