package environ_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// This file is the Provider Identity agreement battery. Two
// validators implement the Section 5.5 shape rule-for-rule:
// provhost.CheckIdentity mints provider-stdio refusals while
// canonicaljson validates the same record through the closed
// identity shape on the way to its object identity. A shape-rule
// change must land in both validators in the same change, and
// this battery is what fails when it lands in only one: every
// row drives one shared vector through both production entries
// and requires identical verdicts with the full rendering on
// each side.
//
// One split is by design, not drift: provider equality (the
// record naming the expected provider) is a caller-correlation
// rule the host enforces, not a content rule the identity
// calculation owns. The another-provider row ledgers that split
// exactly — provhost refuses invalid_config while the
// canonicalizer accepts — so a content-rule divergence cannot
// hide behind it and a change to the split reddens here.

// identityRow is one agreement vector: the body, the provhost
// rule phrase, and the canonicalizer rule phrase. Empty phrases
// mean both entries accept.
type identityRow struct {
	name      string
	body      []byte
	provText  string
	canonText string
}

// identityCorpus mutates each identity member alone against an
// otherwise-valid record over the shared fixture identities.
func identityCorpus(t *testing.T) []identityRow {
	t.Helper()
	valid := validIdentityJSON()
	drop := func(member string) []byte { return dropMember(t, valid, member) }
	set := func(member, literal string) []byte { return mutateMember(t, valid, member, literal) }
	otherSubject := jsonQuote(fixtureCreatorHostID)
	return []identityRow{
		{"valid", valid, "", ""},
		{"unknown member", appendMember(t, valid, "score", `1`), "identity carries unknown member", "unknown member"},
		{"missing opaque", drop("opaque_identity"), "identity misses a required member", "required member"},
		// Schema dispatch precedes shape validation in the
		// canonicalizer: a wrong schema or version refuses at
		// the dispatch registry, never reaching the identity
		// rules provhost enforces as member arms. Same
		// verdict, different arm by construction; a schema
		// change must update the dispatch registry and both
		// validators together.
		{"wrong schema", set("schema", `"urn:ax:schema:provider-manifest"`), "identity schema is not the provider identity", "has no supported immutable self-identity contract"},
		{"wrong version", set("schema_version", `"1.1.0"`), "identity schema_version is not 1.0.0", "has no supported immutable self-identity contract"},
		{"bad record digest", set("record_id", `"sha256:zzzz"`), "identity record_id is not a digest", "record_id"},
		{"bad subject uuid", set("subject_id", `"not-a-uuid"`), "identity subject_id is not a UUIDv7", "subject_id"},
		{"subject differs", set("subject_id", otherSubject), "identity subject_id does not equal session_id", "subject_id must equal session_id"},
		{"session differs", set("session_id", otherSubject), "identity subject_id does not equal session_id", "subject_id must equal session_id"},
		{"bad provider grammar", set("provider_id", `"Antigravity"`), "identity provider_id is not a provider id", "provider_id"},
		{"empty version", set("provider_version", `""`), "identity provider_version is not 1..128 characters", "provider_version"},
		{"long version", set("provider_version", jsonQuote(strings.Repeat("v", 129))), "identity provider_version is not 1..128 characters", "provider_version"},
		{"empty range", set("provider_version_range", `""`), "identity provider_version_range is not 1..256 characters", "provider_version_range"},
		{"empty native id", set("native_session_id", `""`), "identity native_session_id is not 1..512 characters", "native_session_id"},
		{"unknown kind", set("identity_kind", `"telepathy"`), "identity kind is not a registry member", "identity_kind"},
		{"bad workspace uuid", set("logical_workspace_id", `"not-a-uuid"`), "identity workspace is not a UUIDv7", "logical_workspace_id"},
		{"bad realm", set("backend_realm_fingerprint", `"sha256:zzzz"`), "identity realm is not a digest or null", "backend_realm_fingerprint"},
		{"null realm refused", set("backend_realm_fingerprint", `null`), "identity realm is required for this backend kind", "must be non-null for Antigravity"},
		{"opaque array refused", set("opaque_identity", `[]`), "not a JSON object", "opaque_identity"},
		{"opaque 33 keys refused", set("opaque_identity", opaqueKeysJSON(33)), "exceeds 32 entries", "exceeds maximum length 32"},
		{"opaque bad key", set("opaque_identity", `{"K":"v"}`), "identity opaque key is not a provider key", "must match"},
		{"opaque empty value", set("opaque_identity", `{"k":""}`), "identity opaque value is not 1..1024 characters", "1..1024"},
		{"opaque absolute path", set("opaque_identity", `{"k":"/etc/passwd"}`), "begins with an absolute path", "must not begin with an absolute path"},
		{"bad creator uuid", set("created_by_host_id", `"not-a-uuid"`), "identity host is not a UUIDv7", "created_by_host_id"},
		{"bad timestamp", set("created_at", `"yesterday"`), "identity timestamp is not a timestamp", "created_at"},
		{"extensions array refused", set("extensions", `[]`), "identity extensions is not an object", "extensions"},
		{"lone escape refused", set("opaque_identity", `{"k":"A`+rawLoneHigh+`B"}`), "lone surrogate escape", "surrogate"},
		{"escaped text admits", set("opaque_identity", `{"k":"A`+rawEscapedHigh+`B"}`), "", ""},
		// The rows below pin the deliberate copy against its owner
		// (canonicaljson) on the classes where the dialect is blind:
		// extensions content, numbers, and nesting the dialect never
		// reads. The provhost phrase on each is the conjoined owner
		// gate, not a member arm: deleting the owner call admits the
		// body and fails the row on the provhost side.
		{"extensions open key refused", set("extensions", `{"x": "y"}`), "identity is not a valid provider identity", "extensions key"},
		{"extensions unsafe integer refused", set("extensions", `{"works.relux.ax.x": 9007199254740993}`), "identity is not a valid provider identity", "outside the AX safe-integer interval"},
		{"extensions float refused", set("extensions", `{"works.relux.ax.x": 1.5}`), "identity is not a valid provider identity", "forbidden by the AX common model"},
		{"extensions 65 keys refused", set("extensions", extensionKeysJSON(65)), "identity is not a valid provider identity", "maximum is 64"},
		{"extensions deep nesting refused", set("extensions", `{"works.relux.ax.a": {"b": {"c": {"d": {"e": {"f": 1}}}}}}`), "identity is not a valid provider identity", "maximum nesting depth 4"},
		// Bound edges in both directions: the admit rows prove the gate
		// is not refusing everything, the refuse rows prove the bound.
		{"opaque 64-char key admits", set("opaque_identity", `{"k`+strings.Repeat("e", 63)+`": "v"}`), "", ""},
		{"version 128 admits", set("provider_version", jsonQuote(strings.Repeat("v", 128))), "", ""},
		{"version 128 wide admits", set("provider_version", jsonQuote(strings.Repeat("é", 128))), "", ""},
		{"version 129 wide refused", set("provider_version", jsonQuote(strings.Repeat("é", 129))), "identity provider_version is not 1..128 characters", "provider_version"},
		// A number where the dialect already reads the type is refused
		// by the dialect arm first: the owner gate never fires, and the
		// member attribution survives the conjunction.
		{"opaque unsafe integer refused", set("opaque_identity", `{"k": 9007199254740993}`), "identity opaque value is not 1..1024 characters", "outside the AX safe-integer interval"},
		// Decoder vectors both entries must refuse with their own arm.
		{"duplicate member refused", duplicateMemberJSON(t, valid), "duplicate member", "duplicate object member"},
		{"trailing data refused", append(append([]byte{}, valid...), ' ', '{', '}'), "trailing data", "trailing JSON"},
	}
}

// extensionKeysJSON builds an extensions object with count reverse-DNS
// keys, so the count bound (not the key grammar) is what refuses it.
func extensionKeysJSON(count int) string {
	parts := make([]string, 0, count)
	for index := 0; index < count; index++ {
		parts = append(parts, `"works.relux.ax.k`+twoDigits(index)+`":"v"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// duplicateMemberJSON returns the fixture with one member doubled: a
// repeated member has no single value, so both decoders refuse it before
// any shape rule runs.
func duplicateMemberJSON(t *testing.T, valid []byte) []byte {
	t.Helper()
	body := string(valid)
	if body == "" || body[len(body)-1] != '}' {
		t.Fatalf("duplicate member fixture is not an object")
	}
	return []byte(body[:len(body)-1] + `, "score": 1, "score": 2}`)
}

// Raw escape fragments. rawLoneHigh is a real lone high escape
// in JSON source; rawEscapedHigh is an escaped backslash
// followed by text, which carries no escape at all. Both are
// package variables (not literals) so the Go compiler never
// sees an invalid \u escape in source.
var (
	rawLoneHigh    = "\\ud800"
	rawEscapedHigh = "\\\\ud800"
)

// opaqueKeysJSON builds an opaque_identity object with count
// grammar-valid keys.
func opaqueKeysJSON(count int) string {
	parts := make([]string, 0, count)
	for index := 0; index < count; index++ {
		parts = append(parts, `"k`+twoDigits(index)+`":"v"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func twoDigits(value int) string {
	digits := "0123456789"
	return string([]byte{digits[value/10%10], digits[value%10]})
}

// TestIdentityAgreementAcrossValidators drives the corpus
// through both identity entries and requires identical verdicts
// with the full rule rendering on each side.
func TestIdentityAgreementAcrossValidators(t *testing.T) {
	for _, row := range identityCorpus(t) {
		t.Run(row.name, func(t *testing.T) {
			provErr := provhost.CheckIdentity(row.body, "antigravity")
			_, _, canonErr := canonicaljson.CalculateObjectIdentity(row.body)
			if row.provText == "" && row.canonText == "" {
				if provErr != nil {
					t.Fatalf("provhost refused %s: %v", row.body, provErr)
				}
				if canonErr != nil {
					t.Fatalf("canonicaljson refused %s: %v", row.body, canonErr)
				}
				return
			}
			if provErr == nil {
				t.Fatalf("provhost admitted %s, want rule %q", row.body, row.provText)
			}
			if canonErr == nil {
				t.Fatalf("canonicaljson admitted %s, want rule %q", row.body, row.canonText)
			}
			if !strings.Contains(provErr.Error(), row.provText) {
				t.Fatalf("provhost error = %v, want rule %q", provErr, row.provText)
			}
			if !strings.Contains(canonErr.Error(), row.canonText) {
				t.Fatalf("canonicaljson error = %v, want rule %q", canonErr, row.canonText)
			}
		})
	}
}

// TestIdentityProviderEqualityIsCallerRule ledgers the one
// by-design split: a well-formed record for another provider is
// a caller-correlation failure (invalid_config) at the host,
// while the identity calculation accepts the content. Neither
// side may change its half silently.
func TestIdentityProviderEqualityIsCallerRule(t *testing.T) {
	body := mutateMember(t, validIdentityJSON(), "provider_id", `"codex"`)
	provErr := provhost.CheckIdentity(body, "antigravity")
	if provErr == nil {
		t.Fatalf("provhost admitted a record for another provider")
	}
	if !strings.Contains(provErr.Error(), "identity names another provider") {
		t.Fatalf("provhost error = %v, want the caller-correlation arm", provErr)
	}
	if _, _, canonErr := canonicaljson.CalculateObjectIdentity(body); canonErr != nil {
		t.Fatalf("canonicaljson refused well-formed content for another provider: %v", canonErr)
	}
}

// TestIdentityMemberTablesAreOneTable requires the provhost
// required-member table and the canonicalizer's exact-member
// list for the identity record to carry the same members in the
// same order: multi-missing determinism is shared behavior.
func TestIdentityMemberTablesAreOneTable(t *testing.T) {
	mine := deriveRequiredList(t, "provhost", "identity.go", "identityRequired")
	theirs := deriveIdentityExactMembers(t)
	if len(mine) != len(theirs) {
		t.Fatalf("identity member tables differ in size:\nprovhost: %q\ncanonicaljson: %q", mine, theirs)
	}
	for index := range mine {
		if mine[index] != theirs[index] {
			t.Fatalf("identity member tables differ at %d:\nprovhost: %q\ncanonicaljson: %q", index, mine, theirs)
		}
	}
}

// deriveIdentityExactMembers derives the member list from the
// requireExactMembers call for the Provider Identity Record in
// canonicaljson production.
func deriveIdentityExactMembers(t *testing.T) []string {
	t.Helper()
	path := mustInternalRoot(t) + "/canonicaljson/core_records.go"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("identity members: %v", err)
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatalf("identity members: %v", err)
	}
	found := false
	var members []string
	ast.Inspect(syntax, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || found {
			return true
		}
		target, ok := call.Fun.(*ast.Ident)
		if !ok || target.Name != "requireExactMembers" || len(call.Args) < 3 {
			return true
		}
		first, ok := call.Args[0].(*ast.BasicLit)
		if !ok || strings.Trim(first.Value, `"`) != "Provider Identity Record" {
			return true
		}
		for _, arg := range call.Args[2:] {
			literal, ok := arg.(*ast.BasicLit)
			if !ok {
				t.Fatalf("identity members: exact-member list is not literals; the extractor cannot see through indirection")
			}
			members = append(members, strings.Trim(literal.Value, `"`))
		}
		found = true
		return false
	})
	if !found {
		t.Fatal("identity members: requireExactMembers call for the Provider Identity Record not found; the extractor is blind")
	}
	return members
}
