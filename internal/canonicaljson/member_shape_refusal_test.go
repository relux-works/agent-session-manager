package canonicaljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file is the sweep that reaches the per-member refusals.
//
// The reviewer's root cause: requireExactMembers runs first in every validator,
// so a negative fixture that OMITS a member short-circuits on the closed-member
// sweep and never reaches the type, format or coupling refusal it claims to
// pin. Every case here therefore supplies a COMPLETE valid member set and
// violates exactly one member's JSON type.
//
// The subject is derived, not listed: every value position of every fixture in
// everyValidIdentityFixture, which is itself required to cover every completely
// validated schema/version registered in production. A new member, a new array
// element shape, or a new record type becomes an obligation without anyone
// adding a row.

// untypedFixtureMembers names the members whose pinned declaration assigns no
// JSON type, so accepting a value of any type there is correct rather than a
// gap. Requiring a refusal for these would invent a constraint the pinned SPEC
// does not declare — the same class of error this Story has already been
// rejected for three times.
//
// The key is the member name. The set is asserted EXACTLY in both directions by
// TestEveryFixtureMemberRefusesAWrongJSONTypeAtItsProductionEntry: a member
// listed here that turns out to be refused fails as an obsolete exemption, and
// a member accepted without being listed fails as an unenforced type.
//
// Pinned SPEC v0.5.0 at 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c declares:
// "Environment Tuple contains exactly environment_id, environment_version,
// platform=linux|macos|windows|wsl2, architecture=amd64|arm64,
// store_schema_fingerprint, and adapter_version". The three members below are
// the only ones in that list carrying neither a type nor a format, while their
// platform and architecture siblings carry explicit vocabularies, so presence
// is all the clause declares.
var untypedFixtureMembers = map[string]string{
	"environment_version":      "EnvironmentTuple environment_version is declared by name only; the string[1..128] bound belongs to Environment Observation",
	"store_schema_fingerprint": "EnvironmentTuple store_schema_fingerprint is declared by name only and is not declared as a digest",
	"adapter_version":          "EnvironmentTuple adapter_version is declared by name only; the SemVer word belongs to the Session Adapter Manifest row of a different schema",
}

// wrongJSONTypeFor returns a value of a different JSON type from the one the
// fixture carries at a position. A typed closed member must refuse it —
// including a nullable member, where a number is neither null nor the declared
// type.
func wrongJSONTypeFor(value any) (any, string) {
	switch value.(type) {
	case string:
		return json.Number("1"), "number for a string"
	case json.Number:
		return "1", "string for a number"
	case bool:
		return "true", "string for a boolean"
	case []any:
		return map[string]any{}, "object for an array"
	case map[string]any:
		return []any{}, "array for an object"
	case nil:
		return json.Number("1"), "number for a null"
	}
	return json.Number("1"), "number"
}

// TestEveryFixtureMemberRefusesAWrongJSONTypeAtItsProductionEntry drives the
// sweep. It fails when a typed member is accepted with a wrong-typed value,
// which is the exact shape of an unenforced member: the gate is reachable, the
// valid fixture passes, and nothing ever proves the declared type is checked.
func TestEveryFixtureMemberRefusesAWrongJSONTypeAtItsProductionEntry(t *testing.T) {
	accepted := make(map[string][]string)
	for _, fixture := range everyValidIdentityFixture() {
		for _, path := range everyCandidateValuePath(fixture.object) {
			path := path
			member := path[len(path)-1]
			replacement, description := wrongJSONTypeFor(jsonValueAtPath(t, fixture.object, path))
			name := fmt.Sprintf("%s/%s %s", fixture.name, formatJSONPath(path), description)
			t.Run(name, func(t *testing.T) {
				candidate := cloneJSONObject(t, fixture.object)
				setJSONValueAtPath(t, candidate, path, replacement)
				refused := fixtureEntriesRefuse(t, fixture, candidate)
				if _, untyped := untypedFixtureMembers[member]; untyped {
					if refused {
						t.Fatalf(
							"%s is declared untyped in untypedFixtureMembers but production refuses a wrong-typed value; "+
								"the exemption is obsolete and must be removed", member,
						)
					}
					return
				}
				if !refused {
					accepted[member] = append(accepted[member], name)
				}
			})
		}
	}

	if len(accepted) == 0 {
		return
	}
	members := make([]string, 0, len(accepted))
	for member, cases := range accepted {
		sort.Strings(cases)
		members = append(members, fmt.Sprintf("%s (%d case(s), e.g. %s)", member, len(cases), cases[0]))
	}
	sort.Strings(members)
	t.Errorf(
		"member(s) accepted with a wrong JSON type and not declared untyped. Either production does not enforce "+
			"the declared type, or the pinned SPEC declares the member by name only and it belongs in "+
			"untypedFixtureMembers with the clause quoted:\n  %s",
		strings.Join(members, "\n  "),
	)
}

// fixtureEntriesRefuse drives the production entry that owns the fixture and
// reports whether it refused.
//
// Identity-addressed records are driven at both CalculateObjectIdentity and
// VerifyObjectIdentity; Section 18.1 Observation Events are not
// identity-addressed and are driven at ValidateObservationEvent. A refusal at
// one entry and acceptance at the other is a hard failure, not a "refused":
// that divergence is the bypass path this pair exists to close.
func fixtureEntriesRefuse(t *testing.T, fixture identityFixture, candidate map[string]any) bool {
	t.Helper()

	encoded := mustJSON(t, candidate)
	if fixture.selfField == "" {
		err := ValidateObservationEvent(encoded)
		if err != nil && !errors.Is(err, ErrInvalidObservation) {
			t.Fatalf("ValidateObservationEvent(%s) error = %v, want ErrInvalidObservation or acceptance", fixture.name, err)
		}
		return err != nil
	}

	_, _, calculateErr := CalculateObjectIdentity(encoded)
	if calculateErr != nil && !errors.Is(calculateErr, ErrInvalidIdentity) {
		t.Fatalf("CalculateObjectIdentity(%s) error = %v, want ErrInvalidIdentity or acceptance", fixture.name, calculateErr)
	}

	// The self member carries the claimed identity. Rewriting it to a correct
	// claim, as the shared helper does, would erase the very value under test,
	// so the wrong-typed candidate is handed to VerifyObjectIdentity verbatim.
	verified := encoded
	if calculateErr == nil {
		verified = withCorrectIdentityClaimForTest(t, encoded, fixture.selfField)
	}
	_, _, verifyErr := VerifyObjectIdentity(verified)
	if verifyErr != nil && !errors.Is(verifyErr, ErrInvalidIdentity) {
		t.Fatalf("VerifyObjectIdentity(%s) error = %v, want ErrInvalidIdentity or acceptance", fixture.name, verifyErr)
	}

	if (calculateErr == nil) != (verifyErr == nil) {
		t.Fatalf(
			"identity entries disagree for %s: CalculateObjectIdentity = %v, VerifyObjectIdentity = %v; "+
				"a shape refused at one entry and admitted at the other is a bypass path",
			fixture.name, calculateErr, verifyErr,
		)
	}
	return calculateErr != nil
}

// everyCandidateValuePath returns every value position of a candidate that
// production validates: every closed member, and every array element.
//
// Open members are excluded because they carry no declared type: the extensions
// object admits any member value, and env_literals admits any environment value.
// Both containers themselves are still emitted, so their own declared object
// type stays under the sweep, and the one closed subtree inside extensions — the
// migration provenance object — is walked like any other closed shape.
func everyCandidateValuePath(object map[string]any) [][]string {
	var paths [][]string
	var visit func(any, []string)
	visit = func(value any, prefix []string) {
		switch typed := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				path := append(append([]string{}, prefix...), key)
				paths = append(paths, path)
				if key == "env_literals" {
					continue
				}
				if key == "extensions" {
					if extensions, ok := typed[key].(map[string]any); ok {
						if provenance, ok := extensions["works.relux.ax.migrated-from"]; ok {
							provenancePath := append(append([]string{}, path...), "works.relux.ax.migrated-from")
							paths = append(paths, provenancePath)
							visit(provenance, provenancePath)
						}
					}
					continue
				}
				visit(typed[key], path)
			}
		case []any:
			for index, member := range typed {
				path := append(append([]string{}, prefix...), fmt.Sprintf("[%d]", index))
				paths = append(paths, path)
				visit(member, path)
			}
		}
	}
	visit(object, nil)
	return paths
}

// setJSONValueAtPath writes a value at a path that may end in an array index,
// which setJSONObjectMemberAtPath does not support.
func setJSONValueAtPath(t *testing.T, root map[string]any, path []string, value any) {
	t.Helper()

	var current any = root
	for _, component := range path[:len(path)-1] {
		if strings.HasPrefix(component, "[") {
			current = current.([]any)[arrayIndexComponent(t, component)]
			continue
		}
		current = current.(map[string]any)[component]
	}
	last := path[len(path)-1]
	if strings.HasPrefix(last, "[") {
		current.([]any)[arrayIndexComponent(t, last)] = value
		return
	}
	current.(map[string]any)[last] = value
}

func arrayIndexComponent(t *testing.T, component string) int {
	t.Helper()
	var index int
	if _, err := fmt.Sscanf(component, "[%d]", &index); err != nil {
		t.Fatal(err)
	}
	return index
}

// structuredScalarShape classifies a fixture value by asking the PRODUCTION
// scalar parsers, not by a member name list. A value that one of these parsers
// accepts carries a declared lexical form, so replacing it with a string that no
// parser accepts must be refused.
//
// Only unambiguous forms are classified. ParseRelativePath, ParseProviderID and
// ParsePlatform accept ordinary words, so a free-text member holding "payments"
// would be misclassified as structured and the sweep would demand a refusal the
// pinned SPEC does not declare. Those members are proven by their own named
// negative tests instead.
func structuredScalarShape(value string) string {
	if _, err := scalar.ParseDigest(value); err == nil {
		return "digest"
	}
	if _, err := scalar.ParseUUIDv7(value); err == nil {
		return "uuidv7"
	}
	if _, err := scalar.ParseUUIDv4(value); err == nil {
		return "uuidv4"
	}
	if _, err := scalar.ParseTimestamp(value); err == nil {
		return "timestamp"
	}
	if _, err := scalar.ParseGitOID(value); err == nil {
		return "git OID"
	}
	// An absolute remote URL only. ParseSanitizedGitURL also accepts the
	// scp-like `host:path` form, which matches any URN — the migration
	// provenance schema_id "urn:ax:schema:session-record" among them — so
	// classifying on the parser alone would demand a git-URL refusal from a
	// member that declares no URL form at all.
	if strings.Contains(value, "://") {
		if _, err := scalar.ParseSanitizedGitURL(value); err == nil {
			return "sanitized git URL"
		}
	}
	if semverPattern.MatchString(value) {
		return "semver"
	}
	return ""
}

// malformedScalarValue is a non-empty printable string that no structured
// scalar parser above accepts, so a member declaring any of those forms must
// refuse it. It is deliberately in range for every declared string bound, so the
// refusal that fires is the format gate and not a length gate.
const malformedScalarValue = "not-a-valid-scalar"

// unenforcedStructuredMembers declares the members that hold a
// structurally-shaped value whose form production does NOT enforce, with the
// reason. The set is asserted exactly in both directions by
// TestEveryStructuredFixtureValueRefusesAMalformedFormAtItsProductionEntry.
//
// Every entry below quotes the pinned SPEC v0.5.0 declaration at
// 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c that assigns the member a plain
// string, which is why its structurally-shaped fixture value carries no
// enforceable form. Requiring a refusal here would invent a constraint.
var unenforcedStructuredMembers = map[string]string{
	"provider_version":         `declared "provider_version:string[1..128]" and "a 1-128 character exact version string"; the fixture value is semver-shaped but no semver form is declared`,
	"native_session_id":        `declared "native_session_id:string[1..512]" and "Opaque provider handle; never interpreted by core"; the fixture value is UUID-shaped but no UUID form is declared`,
	"environment_version":      `EnvironmentTuple declares "environment_id, environment_version, platform=..., architecture=..., store_schema_fingerprint, and adapter_version" — environment_version by name only, with no type and no format`,
	"store_schema_fingerprint": `declared by name only in the same EnvironmentTuple clause; the fixture value is digest-shaped but no digest form is declared`,
	"adapter_version":          `declared by name only in the same EnvironmentTuple clause; the fixture value is semver-shaped, but the SemVer word appears on the Session Adapter Manifest row of a different schema and is not inferred across schemas here`,
}

// TestEveryStructuredFixtureValueRefusesAMalformedFormAtItsProductionEntry is
// the format half of the sweep. Where the wrong-JSON-type sweep proves a member
// is read as the declared JSON type, this proves the declared lexical form is
// parsed: a string member holding a digest, UUID, timestamp, git OID, sanitized
// git URL or semver must refuse a string that is none of them.
func TestEveryStructuredFixtureValueRefusesAMalformedFormAtItsProductionEntry(t *testing.T) {
	if shape := structuredScalarShape(malformedScalarValue); shape != "" {
		t.Fatalf("the malformed sentinel parses as %s, so every case below is vacuous", shape)
	}

	accepted := make(map[string][]string)
	swept := 0
	for _, fixture := range everyValidIdentityFixture() {
		for _, path := range everyCandidateValuePath(fixture.object) {
			path := path
			text, isString := jsonValueAtPath(t, fixture.object, path).(string)
			if !isString {
				continue
			}
			shape := structuredScalarShape(text)
			if shape == "" {
				continue
			}
			swept++
			member := path[len(path)-1]
			name := fmt.Sprintf("%s/%s malformed %s", fixture.name, formatJSONPath(path), shape)
			t.Run(name, func(t *testing.T) {
				candidate := cloneJSONObject(t, fixture.object)
				setJSONValueAtPath(t, candidate, path, malformedScalarValue)
				refused := fixtureEntriesRefuse(t, fixture, candidate)
				if _, unenforced := unenforcedStructuredMembers[member]; unenforced {
					if refused {
						t.Fatalf(
							"%s is declared unenforced in unenforcedStructuredMembers but production refuses a malformed value; "+
								"the exemption is obsolete and must be removed", member,
						)
					}
					return
				}
				if !refused {
					accepted[member] = append(accepted[member], name)
				}
			})
		}
	}
	if swept == 0 {
		t.Fatal("classified zero structured scalar values across every fixture; the classifier is broken, not the package")
	}

	if len(accepted) == 0 {
		return
	}
	members := make([]string, 0, len(accepted))
	for member, cases := range accepted {
		sort.Strings(cases)
		members = append(members, fmt.Sprintf("%s (%d case(s), e.g. %s)", member, len(cases), cases[0]))
	}
	sort.Strings(members)
	t.Errorf(
		"member(s) holding a structured scalar that production accepts in a malformed form. Either the format gate "+
			"is missing, or the pinned SPEC declares no form for the member and it belongs in "+
			"unenforcedStructuredMembers with the reason:\n  %s",
		strings.Join(members, "\n  "),
	)
}
