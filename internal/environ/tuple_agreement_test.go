package environ_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file is the Environment Tuple agreement battery. The
// tuple is the one type every state-bearing adapter call
// carries, and its admission model is shared with the Section
// 10.8.1 observation rather than restated there. The battery
// proves sessadapter.DecodeTuple and environ.DecodeTuple agree
// verdict-for-verdict over a corpus that mutates each member
// alone, and proves the two member tables are literally one
// table: the required lists are derived from both productions
// and required equal in order as well as content, because
// missing-member determinism (which member is named when several
// are absent) is itself shared behavior.

// deriveRequiredList derives the string list behind a
// package-level []string variable in one production file. The
// comparison travels with the derivation: a reordered,
// extended, or shortened table fails here, not silently.
func deriveRequiredList(t *testing.T, pkg, file, symbol string) []string {
	t.Helper()
	path := filepath.Join(mustInternalRoot(t), pkg, file)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("members: %v", err)
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatalf("members: %v", err)
	}
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != symbol {
				continue
			}
			if len(value.Values) != 1 {
				t.Fatalf("members: %s %s is not a single literal; the extractor cannot see through indirection", path, symbol)
			}
			composite, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				t.Fatalf("members: %s %s is not a composite literal; the extractor cannot see through indirection", path, symbol)
			}
			var members []string
			for _, element := range composite.Elts {
				literal, ok := element.(*ast.BasicLit)
				if !ok {
					t.Fatalf("members: %s %s holds a non-literal; the extractor cannot see through indirection", path, symbol)
				}
				members = append(members, strings.Trim(literal.Value, `"`))
			}
			return members
		}
	}
	t.Fatalf("members: %s %s not found; the extractor is blind, not the table absent", path, symbol)
	return nil
}

func mustInternalRoot(t *testing.T) string {
	t.Helper()
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("members: %v", err)
	}
	return root
}

// internalRoot returns the internal/ source root above this
// package directory. It lives here (rather than reusing the
// in-package census helper) because this battery is an external
// test package: the facades it drives import environ, so an
// in-package battery importing them would be an import cycle.
func internalRoot(t *testing.T) (string, error) {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(directory), nil
}

// TestTupleMemberTablesAreOneTable requires the environ and
// sessadapter required-member tables to be literally equal:
// same members, same order. A seventh member admitted on one
// side, or a renamed member, reddens here before any vector
// runs.
func TestTupleMemberTablesAreOneTable(t *testing.T) {
	mine := deriveRequiredList(t, "environ", "tuple.go", "tupleRequired")
	theirs := deriveRequiredList(t, "sessadapter", "tuple.go", "tupleRequired")
	if !reflect.DeepEqual(mine, theirs) {
		t.Fatalf("tuple required tables differ:\nenviron:     %q\nsessadapter: %q", mine, theirs)
	}
	if len(mine) != 6 {
		t.Fatalf("tuple required table has %d members, want the exact six", len(mine))
	}
}

// tupleRow is one agreement vector: the body and the rule phrase
// both refusals must carry. An empty phrase means both entries
// accept.
type tupleRow struct {
	name string
	body []byte
	text string
}

// tupleCorpus mutates each tuple member alone against an
// otherwise-valid body, plus the executable-provenance,
// extensions, and vocabulary edges.
func tupleCorpus(t *testing.T) []tupleRow {
	t.Helper()
	valid := validTupleJSON()
	drop := func(member string) []byte { return dropMember(t, []byte(valid), member) }
	set := func(member, literal string) []byte { return mutateMember(t, []byte(valid), member, literal) }
	return []tupleRow{
		{"valid", []byte(valid), ""},
		{"missing environment_id", drop("environment_id"), "misses a required member"},
		{"missing environment_version", drop("environment_version"), "misses a required member"},
		{"missing platform", drop("platform"), "misses a required member"},
		{"missing architecture", drop("architecture"), "misses a required member"},
		{"missing store_schema_fingerprint", drop("store_schema_fingerprint"), "misses a required member"},
		{"missing adapter_version", drop("adapter_version"), "misses a required member"},
		{"unknown member", set("adapter_version", `"1.2.3"`), ""}, // placeholder, replaced below
		{"extensions refused", appendMember(t, []byte(valid), "extensions", `{}`), "unknown member"},
		{"executable provenance refused", appendMember(t, []byte(valid), "executable_sha256", jsonQuote(fixtureExecutable)), "unknown member"},
		{"bad environment id", set("environment_id", `"Test.Env"`), "environment-id"},
		{"empty environment id", set("environment_id", `""`), "environment-id"},
		{"empty version", set("environment_version", `""`), "string[1..128]"},
		{"long version", set("environment_version", jsonQuote(strings.Repeat("v", 129))), "string[1..128]"},
		{"darwin refused", set("platform", `"darwin"`), "platform"},
		{"wsl2 admits", set("platform", `"wsl2"`), ""},
		{"uppercase refused", set("platform", `"LINUX"`), "platform"},
		{"x86 refused", set("architecture", `"x86"`), "architecture"},
		{"arm64 admits", set("architecture", `"arm64"`), ""},
		{"bad fingerprint", set("store_schema_fingerprint", `"sha256:zzzz"`), "digest"},
		{"short version", set("adapter_version", `"1.2"`), "SemVer"},
		{"prerelease admits", set("adapter_version", `"1.2.3-rc.1"`), ""},
	}
}

// TestTupleAgreementAcrossFacades drives the corpus through both
// tuple entries and requires identical verdicts with the full
// rule rendering on each side.
func TestTupleAgreementAcrossFacades(t *testing.T) {
	rows := tupleCorpus(t)
	// The unknown-member row needs a genuinely unknown member,
	// not a replaced known one.
	for index, row := range rows {
		if row.name == "unknown member" {
			rows[index].body = appendMember(t, []byte(validTupleJSON()), "score", `1`)
			rows[index].text = "unknown member"
		}
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			mine, mineErr := environ.DecodeTuple(row.body)
			theirs, theirsErr := sessadapter.DecodeTuple(row.body)
			if row.text == "" {
				if mineErr != nil {
					t.Fatalf("environ refused %s: %v", row.body, mineErr)
				}
				if theirsErr != nil {
					t.Fatalf("sessadapter refused %s: %v", row.body, theirsErr)
				}
				if mine.EnvironmentID != theirs.EnvironmentID || mine.Version != theirs.Version ||
					mine.Platform != theirs.Platform || mine.Architecture != theirs.Architecture ||
					mine.StoreFingerprint != theirs.StoreFingerprint || mine.AdapterVersion != theirs.AdapterVersion {
					t.Fatalf("decoded tuples differ: %+v vs %+v", mine, theirs)
				}
				return
			}
			if mineErr == nil {
				t.Fatalf("environ admitted %s, want rule %q", row.body, row.text)
			}
			if theirsErr == nil {
				t.Fatalf("sessadapter admitted %s, want rule %q", row.body, row.text)
			}
			if !strings.Contains(mineErr.Error(), row.text) {
				t.Fatalf("environ error = %v, want rule %q", mineErr, row.text)
			}
			if !strings.Contains(theirsErr.Error(), row.text) {
				t.Fatalf("sessadapter error = %v, want rule %q", theirsErr, row.text)
			}
		})
	}
}

// appendMember returns the object with one added top-level
// member. Every other member holds a valid value, so the added
// member alone decides the verdict.
func appendMember(t *testing.T, body []byte, member, literal string) []byte {
	t.Helper()
	members := decodeTestMembers(t, body)
	if _, present := members[member]; present {
		t.Fatalf("appendMember: %q already present", member)
	}
	members[member] = literal
	return encodeTestMembers(t, members)
}
