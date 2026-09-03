package axerror

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// structuredErrorWith rewrites the conforming document's schema_version and
// optionally adds one further top-level member.
func structuredErrorWith(test *testing.T, version, member, raw string) []byte {
	test.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(conformingDocument), &object); err != nil {
		test.Fatalf("unmarshal conforming document: %v", err)
	}
	object["schema_version"] = json.RawMessage(`"` + version + `"`)
	if member != "" {
		object[member] = json.RawMessage(raw)
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	return encoded
}

// TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule is the Structured Error
// half of the same reviewed precedence the CLI Result reader pins.
//
// The closed nine-member shape used to be enforced before the envelope identity
// was settled, so a document of another major carrying a tenth member was
// refused with "unknown field" and errors.Is(err, ErrUnsupportedMajor) was
// false. Both orders refuse; only one tells a compatibility caller the version
// fact. Section 1.6 scopes the member rule to "a major version 1 object",
// Section 17.1 to "within any negotiated major version", Section 17.2 lists
// "rejects an unsupported major" first, and Section 15.1 forbids parsing "a
// different major's payload far enough to trust its error code, retryable bit,
// details, or authority fields".
func TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule(test *testing.T) {
	test.Parallel()

	test.Run("future major carrying an unknown member reports the major", func(test *testing.T) {
		_, err := Decode(Version120, structuredErrorWith(test, "2.0.0", "new_in_v2", `"smuggled"`))
		if !errors.Is(err, ErrUnsupportedMajor) {
			test.Fatalf("error = %v, want ErrUnsupportedMajor", err)
		}
		if strings.Contains(err.Error(), "unknown field") {
			test.Fatalf("error = %v, want the version fact rather than a structural claim about "+
				"a payload whose major was never settled", err)
		}
	})

	test.Run("future major alone still reports the major", func(test *testing.T) {
		_, err := Decode(Version120, structuredErrorWith(test, "2.0.0", "", ""))
		if !errors.Is(err, ErrUnsupportedMajor) {
			test.Fatalf("error = %v, want ErrUnsupportedMajor", err)
		}
	})

	test.Run("wrong bound minor outranks the member rule too", func(test *testing.T) {
		_, err := Decode(Version120, structuredErrorWith(test, "1.0.0", "new_in_v2", `"smuggled"`))
		if !errors.Is(err, ErrVersionMismatch) {
			test.Fatalf("error = %v, want ErrVersionMismatch", err)
		}
	})

	test.Run("bound version carrying an unknown member still reports the member", func(test *testing.T) {
		_, err := Decode(Version120, structuredErrorWith(test, "1.2.0", "new_in_v2", `"smuggled"`))
		if err == nil || !strings.Contains(err.Error(), "unknown field") {
			test.Fatalf("error = %v, want the unknown-member refusal", err)
		}
		if errors.Is(err, ErrUnsupportedMajor) || errors.Is(err, ErrVersionMismatch) {
			test.Fatalf("error = %v, want the member fact for a document of the bound version", err)
		}
	})

	test.Run("a foreign schema outranks the member rule too", func(test *testing.T) {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(conformingDocument), &object); err != nil {
			test.Fatalf("unmarshal conforming document: %v", err)
		}
		object["schema"] = json.RawMessage(`"urn:ax:schema:cli-result"`)
		object["new_in_v2"] = json.RawMessage(`"smuggled"`)
		encoded, err := json.Marshal(object)
		if err != nil {
			test.Fatalf("marshal: %v", err)
		}
		if _, err := Decode(Version120, encoded); err == nil ||
			!strings.Contains(err.Error(), "schema is not urn:ax:schema:error") {
			test.Fatalf("error = %v, want the schema refusal", err)
		}
	})
}

// TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse pins the other half:
// reading the identity from the raw bytes before the closed decode must change
// which refusal a caller sees and nothing else.
func TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse(test *testing.T) {
	test.Parallel()

	replace := func(member, raw string) []byte {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(conformingDocument), &object); err != nil {
			test.Fatalf("unmarshal conforming document: %v", err)
		}
		if raw == "" {
			delete(object, member)
		} else {
			object[member] = json.RawMessage(raw)
		}
		encoded, err := json.Marshal(object)
		if err != nil {
			test.Fatalf("marshal: %v", err)
		}
		return encoded
	}

	for _, row := range []struct {
		name     string
		document []byte
	}{
		{"unknown member at the bound version", structuredErrorWith(test, "1.2.0", "authority", `"owner"`)},
		{"missing schema", replace("schema", "")},
		{"missing schema_version", replace("schema_version", "")},
		{"missing code", replace("code", "")},
		{"missing details", replace("details", "")},
		{"schema is not a string", replace("schema", `5`)},
		{"schema_version is not a string", replace("schema_version", `1`)},
		{"schema_version is not a semver", replace("schema_version", `"1.2"`)},
		{"unregistered version of major 1", replace("schema_version", `"1.9.0"`)},
		{"exit_code as a string", replace("exit_code", `"9"`)},
		{"trailing content", []byte(conformingDocument + "\n{}")},
		{"not an object", []byte(`[]`)},
		{"empty", nil},
		// The common-data-model gate runs before the identity, so these rows
		// state the second precedence decision this reader makes: a document
		// whose members repeat has no unambiguous identity to settle, and the
		// bytes are refused as bytes rather than reported as a version or a
		// member fact. Both orders refuse; a reader that settled the identity
		// first would answer from one of two occurrences of schema_version.
		{"duplicate schema", duplicateMember(test, conformingDocument, "schema", `"urn:ax:schema:error"`)},
		{"duplicate schema_version", duplicateMember(test, conformingDocument, "schema_version", `"1.2.0"`)},
		{"duplicate code", duplicateMember(test, conformingDocument, "code", `"observation_gap"`)},
		{"duplicate retryable", duplicateMember(test, conformingDocument, "retryable", `true`)},
		{"duplicate details", duplicateMember(test, conformingDocument, "details", `{"observed_sequence": "18"}`)},
	} {
		test.Run(row.name, func(test *testing.T) {
			if _, err := Decode(Version120, row.document); err == nil {
				test.Fatalf("Decode admitted %s", row.name)
			}
		})
	}
}
