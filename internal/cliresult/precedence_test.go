package cliresult

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// documentVersionedWith rewrites the normative example's schema_version and adds
// one further top-level member, which is the shape the precedence question is
// about: a document from a major this reader is not bound to, carrying a member
// this reader does not know.
func documentVersionedWith(t *testing.T, version, member, raw string) []byte {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(normativeCLISuccess), &object); err != nil {
		t.Fatalf("unmarshal example: %v", err)
	}
	object["schema_version"] = json.RawMessage(`"` + version + `"`)
	if member != "" {
		object[member] = json.RawMessage(raw)
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return encoded
}

// TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule pins the reviewed
// precedence between two refusals that both apply to the same document.
//
// A CLI Result of a major this reader is not bound to, carrying a top-level
// member this reader does not know, was refused with the structural fact - the
// member is unknown - and `errors.Is(err, ErrUnsupportedMajor)` was false. Both
// orders refuse, so nothing was ever admitted either way; what differed was
// which fact a compatibility caller was told, and a caller that has to decide
// "is my ax too old for this output" cannot decide it from a member name.
//
// The pinned document scopes the member rule to the object it governs three
// times over - Section 1.6 "a reader MUST reject an unknown top-level field in
// a major version 1 object", Section 17.1 "within any negotiated major
// version ... unknown top-level fields remain an error", Section 17.2 rule 1
// "rejects an unsupported major" listed first - so the identity is settled
// first and the member set is checked against a document whose major is known.
//
// Every row below states which of the two refusals must win, so a later change
// that reorders these checks reddens here rather than silently changing the
// answer a caller reads.
func TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule(t *testing.T) {
	t.Parallel()

	t.Run("future major carrying an unknown member reports the major", func(t *testing.T) {
		_, err := Decode(Version100, documentVersionedWith(t, "2.0.0", "new_in_v2", `"smuggled"`))
		if !errors.Is(err, ErrUnsupportedMajor) {
			t.Fatalf("error = %v, want ErrUnsupportedMajor", err)
		}
		if strings.Contains(err.Error(), "unknown top-level member") {
			t.Fatalf("error = %v, want the version fact rather than a structural claim about "+
				"a payload whose major was never settled", err)
		}
	})

	t.Run("future major alone still reports the major", func(t *testing.T) {
		_, err := Decode(Version100, documentVersionedWith(t, "2.0.0", "", ""))
		if !errors.Is(err, ErrUnsupportedMajor) {
			t.Fatalf("error = %v, want ErrUnsupportedMajor", err)
		}
	})

	t.Run("bound major carrying an unknown member still reports the member", func(t *testing.T) {
		_, err := Decode(Version100, documentVersionedWith(t, "1.0.0", "new_in_v2", `"smuggled"`))
		if err == nil || !strings.Contains(err.Error(), `unknown top-level member "new_in_v2"`) {
			t.Fatalf("error = %v, want the unknown-member refusal", err)
		}
		if errors.Is(err, ErrUnsupportedMajor) {
			t.Fatalf("error = %v, want the member fact for a document of the bound major", err)
		}
	})

	t.Run("bound major missing a member still reports the absence", func(t *testing.T) {
		_, err := Decode(Version100, documentWithout(t, "extensions"))
		if err == nil || !strings.Contains(err.Error(), `missing required member "extensions"`) {
			t.Fatalf("error = %v, want the missing-member refusal", err)
		}
	})

	t.Run("a foreign schema outranks the member rule too", func(t *testing.T) {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(normativeCLISuccess), &object); err != nil {
			t.Fatalf("unmarshal example: %v", err)
		}
		object["schema"] = json.RawMessage(`"urn:ax:schema:error"`)
		object["new_in_v2"] = json.RawMessage(`"smuggled"`)
		encoded, err := json.Marshal(object)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		_, err = Decode(Version100, encoded)
		if err == nil || !strings.Contains(err.Error(), "schema is not urn:ax:schema:cli-result") {
			t.Fatalf("error = %v, want the schema refusal", err)
		}
	})

	t.Run("the identity members themselves are still required", func(t *testing.T) {
		for _, member := range []string{"schema", "schema_version"} {
			_, err := Decode(Version100, documentWithout(t, member))
			if err == nil {
				t.Fatalf("Decode admitted a document without %q", member)
			}
			if !strings.Contains(err.Error(), "missing required member") &&
				!strings.Contains(err.Error(), "schema is not") {
				t.Fatalf("Decode(without %q) = %v, want an absence refusal", member, err)
			}
		}
	})
}

// TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse is the other half of
// the precedence decision: moving the closed-member check behind the identity
// check must change which refusal a caller sees and nothing else. Each shape
// below was refused before the reorder and must still be refused.
func TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		document []byte
	}{
		{"unknown member at the bound major", documentWith(t, "authority", `"owner"`)},
		{"unknown member beside a valid document", documentWith(t, "exit_code", `5`)},
		{"missing body", documentWithout(t, "body")},
		{"missing extensions", documentWithout(t, "extensions")},
		{"missing command", documentWithout(t, "command")},
		{"ok is false", documentWith(t, "ok", "false")},
		{"trailing content", []byte(normativeCLISuccess + "\n{}")},
		{"not an object", []byte(`[]`)},
		{"empty", nil},
		{"schema is not a string", documentWith(t, "schema", `5`)},
		{"schema_version is not a string", documentWith(t, "schema_version", `1`)},
		{"schema_version is not a semver", documentWith(t, "schema_version", `"1.0"`)},
		{"unregistered version of the bound major", documentWith(t, "schema_version", `"1.9.0"`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode(Version100, test.document); err == nil {
				t.Fatalf("Decode admitted %s", test.name)
			}
		})
	}
}
