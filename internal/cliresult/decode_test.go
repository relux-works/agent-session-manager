package cliresult

import (
	"errors"
	"strings"
	"testing"
)

// TestMalformedTopLevelMembersAreRefusedByTheReader narrows the reader's own
// type checks. Each case keeps the document otherwise conforming and gives one
// top-level member a type Section 14.2 does not declare for it.
func TestMalformedTopLevelMembersAreRefusedByTheReader(t *testing.T) {
	cases := map[string]struct{ member, raw string }{
		"numeric schema":        {"schema", `7`},
		"null schema":           {"schema", `null`},
		"object schema_version": {"schema_version", `{"major":1}`},
		"numeric command":       {"command", `7`},
		"unknown command":       {"command", `"resolve"`},
		"numeric ok":            {"ok", `7`},
		"array body":            {"body", `[]`},
		"string body":           {"body", `"forged"`},
		"null body":             {"body", `null`},
		"array extensions":      {"extensions", `[]`},
		"null extensions":       {"extensions", `null`},
		"numeric operation_id":  {"operation_id", `7`},
		"object session_id":     {"session_id", `{}`},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(Version100, documentWith(t, testCase.member, testCase.raw)); err == nil {
				t.Fatalf("Decode admitted %s", name)
			}
		})
	}
}

// TestUnknownCommandInADocumentIsReportedAsUnknown proves the reader does not
// coerce a tag it does not recognize into a neighbouring one.
func TestUnknownCommandInADocumentIsReportedAsUnknown(t *testing.T) {
	_, err := Decode(Version100, documentWith(t, "command", `"take-over"`))
	if !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("Decode = %v, want ErrUnknownCommand", err)
	}
}

// TestDecodeRefusesABodyTheWriterWouldAlsoRefuse proves the reader and the
// writer share one validator: a document that violates a body rule is refused
// on the way in, not only on the way out.
func TestDecodeRefusesABodyTheWriterWouldAlsoRefuse(t *testing.T) {
	document := strings.Replace(normativeCLISuccess, `"adopted": false`, `"adopted": false, "extra": 1`, 1)
	if _, err := Decode(Version100, []byte(document)); err == nil {
		t.Fatalf("Decode admitted a takeover body with an extra member")
	}
	document = strings.Replace(normativeCLISuccess, `"lease_epoch": 5`, `"lease_epoch": 0`, 1)
	if _, err := Decode(Version100, []byte(document)); err == nil {
		t.Fatalf("Decode admitted a zero lease epoch")
	}
	document = strings.Replace(normativeCLISuccess, `"state": "running"`, `"state": "created"`, 1)
	if _, err := Decode(Version100, []byte(document)); err == nil {
		t.Fatalf("Decode admitted a retired session state")
	}
}

// TestExtensionsAreValidatedOnTheReadingSideToo proves the extension bounds are
// not a writer-only courtesy.
func TestExtensionsAreValidatedOnTheReadingSideToo(t *testing.T) {
	for _, raw := range []string{
		`{"works":"no dot"}`,
		`{"works.Relux":"uppercase"}`,
		`{"a.b":{"a":{"b":{"c":{"d":{"e":"too deep"}}}}}}`,
	} {
		if _, err := Decode(Version100, documentWith(t, "extensions", raw)); err == nil {
			t.Fatalf("Decode admitted extensions %s", raw)
		}
	}
	if _, err := Decode(Version100, documentWith(t, "extensions", `{"works.relux.ax.note":"kept"}`)); err != nil {
		t.Fatalf("Decode refused a conforming extension: %v", err)
	}
}

// TestMajorOfRefusesAMalformedReaderVersion proves the reader does not fall
// back to a default major when it cannot parse the version it was bound to.
func TestMajorOfRefusesAMalformedReaderVersion(t *testing.T) {
	if _, err := majorOf("not-a-version"); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("majorOf = %v, want ErrUnsupportedVersion", err)
	}
	if _, err := majorOf(Version100); err != nil {
		t.Fatalf("majorOf(1.0.0): %v", err)
	}
}
