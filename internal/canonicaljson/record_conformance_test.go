package canonicaljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file is the record-level conformance sweep for the six dimensions the
// pinned specification declares over an authoritative record and that no
// single-shape test can establish on its own:
//
//   - round trip: canonicalization is identity-preserving and idempotent for
//     every registered shape, and it does not launder a malformed member;
//   - unknown field: Section 1.5 admits an unknown key only under `extensions`,
//     so the same key must be refused at the top level and accepted below it;
//   - historical major: Section 17 retains every wire-contract version from
//     immutable history, so every v0.4.3 contract version must still resolve at
//     the v0.5.0 production identity entry with an unchanged self contract;
//   - enum union closure: no closed vocabulary position may admit a value that
//     belongs to a different record's vocabulary;
//   - provenance: every Section 10.1 identity-addressed record family carries
//     the four envelope members, and each one is refused when absent, null, or
//     wrongly typed;
//   - cross-record reference: a stored reference is the recomputed omit-self
//     digest of the referenced record, never a name, so substituting the
//     referenced bytes breaks the reference.
//
// The in-object couplings those records carry - subject/session scope, lease
// issuer, the checkpoint's exactly-one manifest rule - are already pinned by the
// derived coupling inventory in coupling_proofs_test.go and by the clause
// refusal proofs, and are not restated here.
//
// Each sweep carries its own anti-degenerate bound. A sweep that walks zero
// positions, or that would still pass with the gate narrowed, fails here
// instead of reporting a green count.

// section101RecordFamilies is the Section 10.1 sentence, verbatim: "Session,
// event, lease, checkpoint, workspace-group, provider-identity, and tombstone
// objects, plus Tombstone Acknowledgements, are immutable identity-addressed
// objects."
//
// Tombstone and Tombstone Acknowledgement schemas are registered in the catalog
// but have no complete shape validator in this package yet, so they are derived
// out below rather than named as exemptions: a family without a complete
// validator cannot be swept for its envelope, and a family that gains one must
// gain a fixture or TestEveryCompletelyValidatedSchemaVersionHasAValidFixture
// reddens first.
var section101RecordFamilies = map[string]string{
	"urn:ax:schema:session-record":    "Session",
	"urn:ax:schema:session-event":     "event",
	leaseSchema:                       "lease",
	checkpointSchema:                  "checkpoint",
	workspaceGroupSchema:              "workspace-group",
	providerIdentitySchema:            "provider-identity",
	"urn:ax:schema:tombstone":         "tombstone",
	"urn:ax:schema:tombstone-ack":     "Tombstone Acknowledgement",
	"urn:ax:schema:session-tombstone": "tombstone",
}

// section101EnvelopeMembers is the Section 10.1 obligation list for those
// families: "its canonical digest ID; subject_id, the logical session or
// workspace scope; created_by_host_id; diagnostic created_at; and optional
// namespaced extensions." The digest ID is the self field and is swept by the
// identity entries themselves, so the four listed here are the members every
// family must additionally carry.
var section101EnvelopeMembers = []string{"subject_id", "created_by_host_id", "created_at", "extensions"}

// identityAddressedFixtures returns the fixtures that address themselves by
// digest. The Section 18.1 Observation Event fixture is not identity-addressed
// and is excluded by its empty self field, not by name.
func identityAddressedFixtures() []identityFixture {
	fixtures := make([]identityFixture, 0, 32)
	for _, fixture := range everyValidIdentityFixture() {
		if fixture.selfField == "" {
			continue
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures
}

func fixtureSchemaVersion(fixture identityFixture) (string, string) {
	schema, _ := fixture.object["schema"].(string)
	version, _ := fixture.object["schema_version"].(string)
	return schema, version
}

// TestEveryIdentityFixtureSurvivesACanonicalRoundTripAtProductionEntries drives
// the production serializer as the storage form and requires it to be identity
// preserving.
//
// The failure this prevents: a canonicalizer that is correct on one hand-picked
// record but rewrites a number token, a multibyte string, or a nested array in
// some other registered shape would silently change that record's storage
// identity on rewrite. Section 10.1 derives the storage path from the digest, so
// a representation-dependent digest is a storage corruption, not a formatting
// difference.
func TestEveryIdentityFixtureSurvivesACanonicalRoundTripAtProductionEntries(t *testing.T) {
	t.Parallel()

	fixtures := identityAddressedFixtures()
	if len(fixtures) == 0 {
		t.Fatal("derived zero identity-addressed fixtures; the derivation is broken, not the package")
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			original := mustJSON(t, fixture.object)
			digest, field, err := CalculateObjectIdentity(original)
			if err != nil {
				t.Fatalf("CalculateObjectIdentity(%s) error = %v", fixture.name, err)
			}

			canonical, err := Canonicalize(original)
			if err != nil {
				t.Fatalf("Canonicalize(%s) error = %v", fixture.name, err)
			}
			again, err := Canonicalize(canonical)
			if err != nil {
				t.Fatalf("Canonicalize(canonical %s) error = %v", fixture.name, err)
			}
			if string(again) != string(canonical) {
				t.Fatalf("Canonicalize is not idempotent for %s:\n  first : %s\n  second: %s",
					fixture.name, canonical, again)
			}

			// The canonical bytes are the storage form. Reading them back must
			// reach the same identity through the same production entry.
			roundTripped, roundTrippedField, err := CalculateObjectIdentity(canonical)
			if err != nil || roundTripped != digest || roundTrippedField != field {
				t.Fatalf("CalculateObjectIdentity(canonical %s) = %q/%q, %v; want %q/%q",
					fixture.name, roundTripped, roundTrippedField, err, digest, field)
			}

			// A stored record carries its claim. Verification must accept the
			// canonical form of the claimed record and return the same digest.
			claimed := withCorrectIdentityClaimForTest(t, canonical, fixture.selfField)
			claimedCanonical, err := Canonicalize(claimed)
			if err != nil {
				t.Fatalf("Canonicalize(claimed %s) error = %v", fixture.name, err)
			}
			verified, verifiedField, err := VerifyObjectIdentity(claimedCanonical)
			if err != nil || verified != digest || verifiedField != fixture.selfField {
				t.Fatalf("VerifyObjectIdentity(canonical claimed %s) = %q/%q, %v; want %q/%q",
					fixture.name, verified, verifiedField, err, digest, fixture.selfField)
			}
		})
	}
}

// TestCanonicalRoundTripDoesNotLaunderAMalformedMember is the round-trip
// negative half.
//
// Section 1.5 declares it directly: "A malformed value remains invalid after a
// caller recomputes the containing object's self-ID." Recomputing the claim and
// canonicalizing the result is exactly the laundering path a writer has, so the
// refusal has to survive both.
func TestCanonicalRoundTripDoesNotLaunderAMalformedMember(t *testing.T) {
	t.Parallel()

	for _, fixture := range identityAddressedFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			mutated := cloneJSONObject(t, fixture.object)
			// created_at is a Section 10.1 envelope member on the record
			// families and an ordinary declared member elsewhere; every fixture
			// that has it must reject an impossible calendar value.
			if _, present := mutated["created_at"]; !present {
				t.Skip("fixture has no created_at member to malform")
			}
			mutated["created_at"] = "2026-02-30T00:00:00.000Z"

			candidate := mustJSON(t, mutated)
			if _, _, err := CalculateObjectIdentity(candidate); err == nil || !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("CalculateObjectIdentity(%s malformed created_at) error = %v, want identity refusal", fixture.name, err)
			}

			// Recompute the self-ID over the malformed bytes and canonicalize
			// them, which is the strongest laundering a caller can perform.
			claimed := withCorrectIdentityClaimForTest(t, candidate, fixture.selfField)
			canonical, err := Canonicalize(claimed)
			if err != nil {
				t.Fatalf("Canonicalize(%s malformed claimed) error = %v", fixture.name, err)
			}
			if _, _, err := VerifyObjectIdentity(canonical); err == nil || !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("VerifyObjectIdentity(%s malformed after self-ID recomputation and canonicalization) error = %v, want refusal",
					fixture.name, err)
			}
		})
	}
}

// TestDistinctIdentityFixturesDoNotShareAnOmitSelfDigest is the anti-degenerate
// bound under the round-trip sweep.
//
// Every assertion above compares a digest to another digest computed the same
// way. A calculation that ignored object content, hashed only the schema, or
// returned a constant would satisfy all of them. Requiring the fixture digests
// to be pairwise distinct is what makes the round-trip sweep mean that identity
// tracks content.
func TestDistinctIdentityFixturesDoNotShareAnOmitSelfDigest(t *testing.T) {
	t.Parallel()

	seen := make(map[scalar.Digest]string)
	for _, fixture := range identityAddressedFixtures() {
		digest, _, err := CalculateObjectIdentity(mustJSON(t, fixture.object))
		if err != nil {
			t.Fatalf("CalculateObjectIdentity(%s) error = %v", fixture.name, err)
		}
		if previous, collision := seen[digest]; collision {
			t.Errorf("fixtures %q and %q share omit-self digest %q; identity does not track content",
				previous, fixture.name, digest)
			continue
		}
		seen[digest] = fixture.name
	}
	if len(seen) < 2 {
		t.Fatalf("compared %d distinct fixture digests; the sweep proves nothing below two", len(seen))
	}
}

// TestUnknownTopLevelMemberIsRefusedWhileTheSameKeyIsAdmittedUnderExtensions
// binds both halves of the Section 1.5 extension boundary at one call.
//
// Section 1.5: "Unknown fields MAY be retained only under an extensions map
// whose keys are reverse-DNS names. A reader MUST reject an unknown top-level
// field in a major version 1 object." A refusal test alone is satisfied by a
// validator that rejects the key everywhere, which would break the only declared
// forward-compatibility channel; an acceptance test alone is satisfied by a
// validator that accepts it everywhere, which is the fail-open the clause exists
// to prevent. The same key and the same value are used on both sides so the only
// difference between the accepted and the refused object is its position.
func TestUnknownTopLevelMemberIsRefusedWhileTheSameKeyIsAdmittedUnderExtensions(t *testing.T) {
	t.Parallel()

	const extensionKey = "works.relux.ax.unknown-control"
	extensionValue := "retained"

	swept := 0
	for _, fixture := range identityAddressedFixtures() {
		if _, extensible := fixture.object["extensions"]; !extensible {
			continue
		}
		swept++
		t.Run(fixture.name, func(t *testing.T) {
			refused := cloneJSONObject(t, fixture.object)
			refused[extensionKey] = extensionValue
			assertIdentityEntriesRefuseShape(t, mustJSON(t, refused), fixture.selfField)

			// A reverse-DNS key at the top level is still unknown there. The
			// refusal must not depend on the key looking un-namespaced.
			admitted := cloneJSONObject(t, fixture.object)
			extensions, ok := admitted["extensions"].(map[string]any)
			if !ok {
				t.Fatalf("fixture %s extensions member is %T, want object", fixture.name, admitted["extensions"])
			}
			extensions[extensionKey] = extensionValue
			assertIdentityEntriesAcceptShape(t, mustJSON(t, admitted), fixture.selfField)
		})
	}
	if swept == 0 {
		t.Fatal("swept zero extensible fixtures; the sweep is broken, not the package")
	}
}

// contractVersionsByID projects one release's Section 1.5 contract registry
// into contract ID -> declared versions.
func contractVersionsByID(t *testing.T, release catalog.Release) map[string][]string {
	t.Helper()

	projection, err := catalog.ForRelease(release)
	if err != nil {
		t.Fatalf("catalog.ForRelease(%s) error = %v", release, err)
	}
	versions := make(map[string][]string, len(projection.Contracts))
	for _, contract := range projection.Contracts {
		versions[string(contract.ID)] = append([]string{}, contract.Versions...)
	}
	return versions
}

// retentionViolations returns every way the current registry fails to retain a
// historical one: a dropped contract, or a contract that survived while losing
// one of the versions immutable history published under it.
//
// It is a pure function over two release projections so the gate below can be
// proven to have teeth without mutating a shipped registry, and so the
// comparison is between two independently generated release rows rather than
// between a table and itself.
func retentionViolations(historical, current map[string][]string) []string {
	var problems []string
	for contract, versions := range historical {
		retained, present := current[contract]
		if !present {
			problems = append(problems, fmt.Sprintf("%s is no longer registered", contract))
			continue
		}
		index := make(map[string]struct{}, len(retained))
		for _, version := range retained {
			index[version] = struct{}{}
		}
		for _, version := range versions {
			if _, kept := index[version]; !kept {
				problems = append(problems, fmt.Sprintf("%s dropped historical version %s", contract, version))
			}
		}
	}
	sort.Strings(problems)
	return problems
}

// TestEveryV043ContractVersionIsRetainedByTheV050Registry is the Section 17
// retention gate at the registry level.
//
// Section 17 requires every wire-contract version from immutable history to be
// retained, and Section 1.5 fixes the v0.4.3 registry as the v0.5.0 table with
// the five TerminalBackend rows absent and six rows pinned to their then-active
// versions. Every other row, identifier, version, order and meaning is exactly
// the v0.4.3 registry. The compatibility test in internal/specpin checks the six
// named rows and the five absent ones; this checks the rule those exceptions are
// exceptions to, over all 55 historical rows, so a v0.5.0 release that quietly
// drops some other historical major fails here rather than at the first
// unreadable v0.4.3 object.
func TestEveryV043ContractVersionIsRetainedByTheV050Registry(t *testing.T) {
	t.Parallel()

	historical := contractVersionsByID(t, catalog.ReleaseV043)
	current := contractVersionsByID(t, catalog.ReleaseV050)
	if len(historical) == 0 || len(current) == 0 {
		t.Fatalf("projected %d historical and %d current contracts; the projection is broken, not the package",
			len(historical), len(current))
	}
	for _, problem := range retentionViolations(historical, current) {
		t.Errorf("v0.5.0 registry %s", problem)
	}
}

// TestRetentionViolationDetectionReportsADroppedVersionAndContract proves the
// gate above has teeth.
//
// Emptying the current registry would fail any check. The mutants here are the
// two that actually happen: one historical major removed from an otherwise
// intact contract row, and one whole contract row removed. Both must be reported
// by name, and a registry that only ADDS rows and versions must stay silent.
func TestRetentionViolationDetectionReportsADroppedVersionAndContract(t *testing.T) {
	t.Parallel()

	historical := contractVersionsByID(t, catalog.ReleaseV043)
	current := contractVersionsByID(t, catalog.ReleaseV050)

	if problems := retentionViolations(historical, current); len(problems) != 0 {
		t.Fatalf("baseline retention reported %v; the mutants below prove nothing from a red baseline", problems)
	}

	const eventContract = sessionEventSchema
	if len(historical[eventContract]) < 2 {
		t.Fatalf("v0.4.3 %s declares %v; the mutant needs a multi-version historical row",
			eventContract, historical[eventContract])
	}

	narrowed := copyContractVersions(current)
	narrowed[eventContract] = narrowed[eventContract][1:]
	dropped := historical[eventContract][0]
	problems := retentionViolations(historical, narrowed)
	if len(problems) != 1 || !strings.Contains(problems[0], "dropped historical version "+dropped) {
		t.Fatalf("dropping %s@%s reported %v, want exactly one dropped-version problem", eventContract, dropped, problems)
	}

	removed := copyContractVersions(current)
	delete(removed, eventContract)
	problems = retentionViolations(historical, removed)
	if len(problems) != 1 || !strings.Contains(problems[0], eventContract+" is no longer registered") {
		t.Fatalf("removing %s reported %v, want exactly one dropped-contract problem", eventContract, problems)
	}

	// A registry that grows must not be reported. A detector that flagged every
	// difference would pass both mutants above while blocking every release.
	widened := copyContractVersions(current)
	widened[eventContract] = append(append([]string{}, widened[eventContract]...), "9.0.0")
	widened["urn:ax:schema:example-future-contract"] = []string{"1.0.0"}
	if problems := retentionViolations(historical, widened); len(problems) != 0 {
		t.Fatalf("a registry that only adds rows reported %v, want silence", problems)
	}
}

func copyContractVersions(source map[string][]string) map[string][]string {
	copied := make(map[string][]string, len(source))
	for contract, versions := range source {
		copied[contract] = append([]string{}, versions...)
	}
	return copied
}

// TestV050OnlyContractVersionsAreAbsentFromTheHistoricalV043Registry is the
// other direction of the same rule.
//
// Retention is only meaningful if the historical projection is genuinely
// historical. If catalog.ForRelease(v0.4.3) returned the current registry, the
// gate above would be vacuously green forever. Section 1.5 names the delta:
// v0.5.0 activates the five TerminalBackend contracts plus Configuration 3.0.0,
// Provider Protocol 3.0.0, Mesh RPC 4.0.0, Session Event 4.0.0, CLI Result
// 4.0.0, and Structured Error 1.3.0.
func TestV050OnlyContractVersionsAreAbsentFromTheHistoricalV043Registry(t *testing.T) {
	t.Parallel()

	historical := contractVersionsByID(t, catalog.ReleaseV043)
	current := contractVersionsByID(t, catalog.ReleaseV050)

	added := make(map[string][]string)
	for contract, versions := range current {
		retained := make(map[string]struct{}, len(historical[contract]))
		for _, version := range historical[contract] {
			retained[version] = struct{}{}
		}
		for _, version := range versions {
			if _, present := retained[version]; !present {
				added[contract] = append(added[contract], version)
			}
		}
	}
	if len(added) == 0 {
		t.Fatal("the v0.4.3 registry carries every v0.5.0 contract version; it is the current registry under " +
			"another name and every retention assertion built on it is vacuous")
	}
	if _, historicallyPresent := historical[sessionEventSchema]; !historicallyPresent {
		t.Fatalf("%s is absent from the v0.4.3 registry; the projection is broken, not the package", sessionEventSchema)
	}
	if !containsString(added[sessionEventSchema], "4.0.0") {
		t.Errorf("Section 1.5 activates Session Event 4.0.0 in v0.5.0, but the v0.4.3 registry already carries it; "+
			"added versions were %v", added[sessionEventSchema])
	}
	// The activated major must also be resolvable in production, or the
	// implementation advertises a release it cannot read.
	if _, resolved := schemaIdentityContracts[schemaIdentityKey{schema: sessionEventSchema, version: "4.0.0"}]; !resolved {
		t.Errorf("%s@4.0.0 is activated by v0.5.0 but does not resolve at the production identity entry", sessionEventSchema)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// TestEveryHistoricalMajorFixtureIsAcceptedUnderItsOwnRegisteredVersion drives
// the retention rule through the production entry rather than through the
// registry table.
//
// A registry row proves the version is known. It does not prove an object
// carrying that version is still readable: a shape validator bound to the newest
// major would resolve the row and then refuse every historical object. Every
// registered version below the highest major of a multi-version schema is driven
// here with its own fixture, and a version one major above the highest
// registered one must be refused so the sweep cannot pass by admitting anything.
func TestEveryHistoricalMajorFixtureIsAcceptedUnderItsOwnRegisteredVersion(t *testing.T) {
	t.Parallel()

	// Compare majors numerically. A lexicographic maximum over version strings
	// silently picks "9.0.0" over "10.0.0" the day a contract reaches two digits,
	// which would move every registered version below the wrong ceiling and quietly
	// empty this sweep.
	highest := make(map[string]int)
	for key := range schemaIdentityContracts {
		major := majorVersion(t, key.version)
		if current, seen := highest[key.schema]; !seen || major > current {
			highest[key.schema] = major
		}
	}

	historical := make(map[schemaIdentityKey]struct{})
	for _, fixture := range identityAddressedFixtures() {
		schema, version := fixtureSchemaVersion(fixture)
		if version == "" || majorVersion(t, version) == highest[schema] {
			continue
		}
		historical[schemaIdentityKey{schema: schema, version: version}] = struct{}{}
		t.Run(fixture.name, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, mustJSON(t, fixture.object), fixture.selfField)
		})
	}
	if len(historical) == 0 {
		t.Fatal("swept zero historical-major fixtures; the sweep is broken, not the package")
	}

	// The complement: one major above the highest registered version of each
	// multi-version schema must not resolve. Retention is not permission to
	// accept an unregistered future major.
	for schema, major := range highest {
		next := fmt.Sprintf("%d.0.0", major+1)
		if _, registered := schemaIdentityContracts[schemaIdentityKey{schema: schema, version: next}]; registered {
			continue
		}
		t.Run("unregistered future major "+schema+"@"+next, func(t *testing.T) {
			candidate := fmt.Sprintf(`{"schema":%q,"schema_version":%q,"record_id":%q}`, schema, next, zeroDigest)
			if _, _, err := CalculateObjectIdentity([]byte(candidate)); err == nil || !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("CalculateObjectIdentity(%s@%s) error = %v, want identity refusal", schema, next, err)
			}
		})
	}
}

func majorVersion(t *testing.T, version string) int {
	t.Helper()
	major, err := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	if err != nil {
		t.Fatalf("parse major of version %q: %v", version, err)
	}
	return major
}

// TestSection101EnvelopeProvenanceIsRequiredByEveryRecordFamilyFixture is the
// provenance sweep.
//
// Section 10.1 requires every identity-addressed record to carry subject_id,
// created_by_host_id, diagnostic created_at, and extensions in addition to its
// schema-specific fields. Per-record tests establish this one record at a time,
// which is exactly the shape that lets a newly registered family ship without
// provenance: nothing fails when a record has no envelope test of its own. This
// derives the obligation from the Section 10.1 family list and the production
// validator registry, so a registered family without a swept fixture reddens.
func TestSection101EnvelopeProvenanceIsRequiredByEveryRecordFamilyFixture(t *testing.T) {
	t.Parallel()

	// A registered family is one this package validates completely. Tombstone
	// and Tombstone Acknowledgement resolve a self field but route to a total
	// refusal, so they have no valid object to sweep. They are derived out by
	// walking the production sources, not excused by name: the day either gains
	// a complete validator it enters this sweep automatically.
	totallyRefusing := deriveTotalRefusalValidators(t)
	registered := make(map[string]struct{})
	for key, validatorName := range deriveRegisteredShapeValidators(t) {
		if totallyRefusing[validatorName] {
			continue
		}
		if _, member := section101RecordFamilies[key.schema]; member {
			registered[key.schema] = struct{}{}
		}
	}
	if len(registered) == 0 {
		t.Fatal("derived zero completely validated Section 10.1 families; the derivation is broken, not the package")
	}

	covered := make(map[string]struct{})
	for _, fixture := range identityAddressedFixtures() {
		schema, _ := fixtureSchemaVersion(fixture)
		if _, member := section101RecordFamilies[schema]; !member {
			continue
		}
		covered[schema] = struct{}{}
		t.Run(fixture.name, func(t *testing.T) {
			for _, member := range section101EnvelopeMembers {
				if _, present := fixture.object[member]; !present {
					t.Errorf("Section 10.1 family %s omits envelope member %q entirely", schema, member)
					continue
				}
				for _, mutation := range envelopeProvenanceMutations(member) {
					t.Run(member+" "+mutation.name, func(t *testing.T) {
						mutated := cloneJSONObject(t, fixture.object)
						mutation.apply(mutated, member)
						candidate := mustJSON(t, mutated)
						if mutation.reason != "" {
							assertIdentityEntriesRefuseWithReason(t, candidate, fixture.selfField, mutation.reason)
							return
						}
						assertIdentityEntriesRefuseShape(t, candidate, fixture.selfField)
					})
				}
			}
		})
	}

	for schema := range registered {
		if _, swept := covered[schema]; !swept {
			t.Errorf("Section 10.1 family %s (%s) has a registered identity contract but no swept fixture; "+
				"its envelope provenance is unproven", schema, section101RecordFamilies[schema])
		}
	}
	if len(covered) == 0 {
		t.Fatal("swept zero Section 10.1 record families; the derivation is broken, not the package")
	}
}

type envelopeMutation struct {
	name  string
	apply func(object map[string]any, member string)
	// reason, when set, is the substring the refusal message must contain. It
	// is how a case that is refused by an EARLIER gate than the one it names
	// records which gate actually caught it, instead of counting the refusal as
	// evidence for a clause it never reached.
	reason string
}

// envelopeProvenanceMutations returns the ways a writer can drop or forge
// provenance while leaving a syntactically well-formed object.
//
// Absence, null, and the wrong JSON type are the degenerate drops. The one that
// matters more is the last: a value of the right JSON type in the wrong
// identity grammar. A host that stamps a UUIDv4 into created_by_host_id, or an
// empty string into subject_id, produces an object that any "is it a string"
// check accepts and that names no host at all. Section 10.1 types these members,
// so each must be refused on its declared grammar rather than on its JSON type.
func envelopeProvenanceMutations(member string) []envelopeMutation {
	mutations := []envelopeMutation{
		// Deleting the member never reaches validateCommonRecordEnvelope:
		// requireExactMembers runs first in every Section 10.1 validator and
		// refuses the incomplete member set. The case is kept because absence
		// must be refused, and its refusal reason is pinned to the closed
		// member set so it cannot be read as evidence about the envelope's own
		// typing. Every mutation below supplies a COMPLETE member set and
		// violates exactly the envelope clause it names.
		{
			name:   "absent",
			apply:  func(object map[string]any, name string) { delete(object, name) },
			reason: fmt.Sprintf("is missing required member %q", member),
		},
		{name: "null", apply: func(object map[string]any, name string) { object[name] = nil }},
	}
	if member == "extensions" {
		return append(mutations,
			envelopeMutation{name: "wrong type", apply: func(object map[string]any, name string) { object[name] = []any{} }},
			envelopeMutation{name: "non reverse-DNS key", apply: func(object map[string]any, name string) {
				object[name] = map[string]any{"unknown": "value"}
			}},
		)
	}
	mutations = append(mutations,
		envelopeMutation{name: "wrong type", apply: func(object map[string]any, name string) { object[name] = json.Number("1") }},
		envelopeMutation{name: "empty string", apply: func(object map[string]any, name string) { object[name] = "" }},
	)
	if member == "created_at" {
		// A well-formed RFC 3339 shape naming a day that does not exist.
		return append(mutations, envelopeMutation{
			name: "impossible calendar date",
			apply: func(object map[string]any, name string) {
				object[name] = "2026-02-30T00:00:00.000Z"
			},
		})
	}
	// subject_id and created_by_host_id are UUIDv7. A UUIDv4 is a valid UUID of
	// the wrong version and is the exact shape a forged or copied provenance
	// stamp takes.
	return append(mutations,
		envelopeMutation{name: "UUIDv4 in a UUIDv7 member", apply: func(object map[string]any, name string) {
			object[name] = "11111111-2222-4333-8444-555555555555"
		}},
		envelopeMutation{name: "non-UUID string", apply: func(object map[string]any, name string) {
			object[name] = "host-alpha"
		}},
	)
}

// ambiguousVocabularyPositions resolves the fixture positions whose member name
// alone does not select one pinned vocabulary row.
//
// The declared set of every other position is derived: it is the single pinned
// row whose member name matches and whose admitted values contain the value the
// fixture carries. Four positions have two such rows, because two different
// records declare a same-named member over overlapping values, and that overlap
// is exactly the contamination this sweep exists to find. Each is resolved here
// against the pinned specification declaration, and the resolution must still be
// one of the candidate rows, so a widened production vocabulary invalidates the
// resolution instead of being absorbed by it.
//
// Keys are `schema|scope|path|value`; scope is the Session Event type or `-`.
var ambiguousVocabularyPositions = map[string][]string{
	// Section 5.3 lease reason, not the Section 5.2 session.quiescing reason.
	"urn:ax:schema:lease|-|reason|graceful_takeover": {"create", "graceful_takeover", "force_takeover", "recovery"},
	// Section 5.1 session kind, not the Section 10.4 Transfer Manifest kind.
	"urn:ax:schema:session-record|-|kind|task_board": {"direct", "task_board"},
	// Section 10.4 Transfer Manifest kind, not the Section 5.1 session kind.
	"urn:ax:schema:transfer-manifest|-|kind|task_board": {
		"workspace_group", "workspace_tree", "provider", "task_board", "composite",
	},
}

// vocabularyPositionKey names a fixture position the way
// ambiguousVocabularyPositions keys it.
func vocabularyPositionKey(fixture identityFixture, position enumPosition) string {
	schema, _ := fixtureSchemaVersion(fixture)
	scope := "-"
	if eventType, tagged := fixture.object["event_type"].(string); tagged {
		scope = eventType
	}
	return fmt.Sprintf("%s|%s|%s|%s", schema, scope, strings.Join(position.path, "."), position.value)
}

// declaredVocabularyAtPosition returns the exact set the pinned inventory
// declares for one fixture position, and whether the position is a closed
// vocabulary at all.
func declaredVocabularyAtPosition(
	t *testing.T,
	fixture identityFixture,
	position enumPosition,
	rows []vocabularyInventoryRow,
) (map[string]struct{}, bool) {
	t.Helper()

	// Several rows can declare the same member over the same values - six
	// production sites carry the standard/yolo execution profile. Those are one
	// vocabulary, not six candidates, so candidates are deduplicated by their
	// admitted set and only a genuinely different set makes a position ambiguous.
	var candidates [][]string
	seen := make(map[string]struct{})
	for _, row := range rows {
		if row.member != position.member {
			continue
		}
		if !containsString(row.values, position.value) {
			continue
		}
		sorted := append([]string{}, row.values...)
		sort.Strings(sorted)
		signature := strings.Join(sorted, ",")
		if _, duplicate := seen[signature]; duplicate {
			continue
		}
		seen[signature] = struct{}{}
		candidates = append(candidates, row.values)
	}

	key := vocabularyPositionKey(fixture, position)
	switch len(candidates) {
	case 0:
		// A free string member can coincidentally hold a pinned value - a
		// manifest entry whose path is literally "file" - and a shape can pin a
		// member to one exact literal where another shape carries a vocabulary,
		// as `clone.failed` pins phase=checkpoint against the six-member
		// `clone.target_validation_failed` phase enum. Neither is a vocabulary
		// position; both are swept as single-value sites, which is the stronger
		// bound.
		if _, member := indexValues(rows, position.member); !member {
			return nil, false
		}
		return map[string]struct{}{position.value: {}}, true
	case 1:
		return valueSet(candidates[0]), true
	default:
		resolved, reviewed := ambiguousVocabularyPositions[key]
		if !reviewed {
			t.Errorf("position %s matches %d pinned vocabulary rows and has no reviewed resolution; "+
				"add one to ambiguousVocabularyPositions naming the declaration that governs it", key, len(candidates))
			return valueSet(candidates[0]), true
		}
		matched := false
		for _, candidate := range candidates {
			if equalValueSets(candidate, resolved) {
				matched = true
			}
		}
		if !matched {
			t.Errorf("reviewed resolution for %s is %v, which is no longer one of the %d pinned candidate rows; "+
				"a production vocabulary changed under it", key, resolved, len(candidates))
		}
		return valueSet(resolved), true
	}
}

func indexValues(rows []vocabularyInventoryRow, member string) (map[string]struct{}, bool) {
	values := make(map[string]struct{})
	for _, row := range rows {
		if row.member != member {
			continue
		}
		for _, value := range row.values {
			values[value] = struct{}{}
		}
	}
	return values, len(values) != 0
}

func valueSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func equalValueSets(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	set := valueSet(first)
	for _, value := range second {
		if _, present := set[value]; !present {
			return false
		}
	}
	return true
}

// TestNoRecordEnumPositionAdmitsAValueFromAnotherPinnedVocabulary is the union
// closure gate.
//
// testdata/closed-vocabularies.md pins the admitted set of every production
// vocabulary call site, which kills a widening mutant at the source. It does not
// establish that a given JSON path in a real record routes to the vocabulary its
// row names: a validator that read a lease `reason` through the
// `session.quiescing` reason gate would leave every pinned row unchanged and
// admit `stop` on a lease. This walks every closed-vocabulary position of every
// fixture, feeds it the union of every other pinned vocabulary, and requires
// refusal for everything outside that position's own declared set.
func TestNoRecordEnumPositionAdmitsAValueFromAnotherPinnedVocabulary(t *testing.T) {
	t.Parallel()

	directory, _ := packageProductionFiles(t)
	rows := readClosedVocabularyInventoryRows(t, filepath.Join(directory, "testdata", closedVocabularyInventoryFile))

	union := make(map[string]struct{})
	for _, row := range rows {
		for _, value := range row.values {
			union[value] = struct{}{}
		}
	}
	if len(union) < 2 {
		t.Fatalf("pinned vocabularies contribute %d distinct values; the union proves nothing", len(union))
	}

	positions := 0
	foreign := 0
	resolved := make(map[string]struct{})
	for _, fixture := range identityAddressedFixtures() {
		for _, position := range enumValuedPositions(fixture.object, nil, union) {
			declared, closed := declaredVocabularyAtPosition(t, fixture, position, rows)
			if !closed {
				continue
			}
			positions++
			resolved[vocabularyPositionKey(fixture, position)] = struct{}{}
			substituted, problems := foreignAdmissionsAtPosition(t, fixture, position, declared, union)
			foreign += substituted
			for _, problem := range problems {
				t.Error(problem)
			}
		}
	}
	if positions == 0 {
		t.Fatal("found zero closed-vocabulary positions across the fixtures; the walker is broken, not the package")
	}
	if foreign == 0 {
		t.Fatal("substituted zero foreign vocabulary values; the sweep proves nothing")
	}
	for key := range ambiguousVocabularyPositions {
		if _, reached := resolved[key]; !reached {
			t.Errorf("reviewed resolution %s names no fixture position; it is stale", key)
		}
	}
	t.Logf("union closure: %d closed positions x %d foreign substitutions", positions, foreign)
}

// foreignAdmissionsAtPosition substitutes every union value the position does
// not declare and returns how many substitutions ran plus every value production
// still admitted. It is shared with the mutant proof below so the gate and its
// teeth check exercise the same comparison.
func foreignAdmissionsAtPosition(
	t *testing.T,
	fixture identityFixture,
	position enumPosition,
	declared map[string]struct{},
	union map[string]struct{},
) (int, []string) {
	t.Helper()

	values := make([]string, 0, len(union))
	for value := range union {
		if _, own := declared[value]; own {
			continue
		}
		values = append(values, value)
	}
	sort.Strings(values)

	var problems []string
	for _, value := range values {
		mutated := cloneJSONObject(t, fixture.object)
		setAtPath(t, mutated, position.path, value)
		if _, _, err := CalculateObjectIdentity(mustJSON(t, mutated)); err == nil {
			problems = append(problems, fmt.Sprintf(
				"fixture %s admits %q at %s, which belongs to another record's vocabulary and not to %q",
				fixture.name, value, strings.Join(position.path, "."), position.member))
		}
	}
	return len(values), problems
}

// TestUnionClosureSweepReportsAPositionThatAdmitsAnUndeclaredValue proves the
// sweep above has teeth.
//
// The sweep is a loop of negative assertions: it stays green if every
// substitution is refused, and it would also stay green if it substituted
// nothing, walked the wrong path, or compared against an over-broad declared
// set. Narrowing one position's declared set to a single member turns the
// remaining values production genuinely admits into expected reports, so a
// sweep that cannot see an over-admitting site fails here.
func TestUnionClosureSweepReportsAPositionThatAdmitsAnUndeclaredValue(t *testing.T) {
	t.Parallel()

	fixture := identityFixture{name: "lease", selfField: SelfRecordID, object: validLeaseRecordObject()}
	position := enumPosition{path: []string{"reason"}, member: "reason", value: "graceful_takeover"}
	// Section 5.3 declares four lease reasons. The fixture is an epoch-4 lease
	// with a non-null predecessor and checkpoint, so the three takeover/recovery
	// reasons are all admissible on it; pinning only its own value must therefore
	// report the other two.
	narrowed := map[string]struct{}{"graceful_takeover": {}}
	union := map[string]struct{}{
		"graceful_takeover": {}, "force_takeover": {}, "recovery": {},
	}

	substituted, problems := foreignAdmissionsAtPosition(t, fixture, position, narrowed, union)
	if substituted != 2 {
		t.Fatalf("substituted %d values, want 2; the sweep is not driving the position", substituted)
	}
	if len(problems) != 2 {
		t.Fatalf("narrowing lease reason to %q reported %v, want both force_takeover and recovery", "graceful_takeover", problems)
	}
	for _, want := range []string{"force_takeover", "recovery"} {
		found := false
		for _, problem := range problems {
			if strings.Contains(problem, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("narrowed lease reason sweep did not report %q", want)
		}
	}

	// The same position with its full declared set must be silent, or the sweep
	// reports every enum value and its green runs mean nothing.
	full := map[string]struct{}{
		"create": {}, "graceful_takeover": {}, "force_takeover": {}, "recovery": {},
	}
	if _, problems := foreignAdmissionsAtPosition(t, fixture, position, full, union); len(problems) != 0 {
		t.Fatalf("the declared lease vocabulary reported %v, want silence", problems)
	}
}

type enumPosition struct {
	path   []string
	member string
	value  string
}

// enumValuedPositions walks a fixture and returns every string-valued position
// whose current value is in the pinned vocabulary union. Positions are found by
// value, not by a member-name list, so a new enum member in a new record is
// swept without being registered anywhere.
func enumValuedPositions(value any, path []string, union map[string]struct{}) []enumPosition {
	var positions []enumPosition
	switch typed := value.(type) {
	case map[string]any:
		names := make([]string, 0, len(typed))
		for name := range typed {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			child := append(append([]string{}, path...), name)
			if text, isString := typed[name].(string); isString {
				if _, pinned := union[text]; pinned {
					positions = append(positions, enumPosition{path: child, member: name, value: text})
				}
				continue
			}
			positions = append(positions, enumValuedPositions(typed[name], child, union)...)
		}
	case []any:
		for index, element := range typed {
			child := append(append([]string{}, path...), fmt.Sprintf("[%d]", index))
			positions = append(positions, enumValuedPositions(element, child, union)...)
		}
	}
	return positions
}

func setAtPath(t *testing.T, object map[string]any, path []string, value string) {
	t.Helper()
	var cursor any = object
	for index, step := range path {
		last := index == len(path)-1
		if strings.HasPrefix(step, "[") {
			array, ok := cursor.([]any)
			if !ok {
				t.Fatalf("path step %q expects an array, found %T", step, cursor)
			}
			var offset int
			if _, err := fmt.Sscanf(step, "[%d]", &offset); err != nil {
				t.Fatalf("parse array index %q: %v", step, err)
			}
			if last {
				array[offset] = value
				return
			}
			cursor = array[offset]
			continue
		}
		container, ok := cursor.(map[string]any)
		if !ok {
			t.Fatalf("path step %q expects an object, found %T", step, cursor)
		}
		if last {
			container[step] = value
			return
		}
		cursor = container[step]
	}
	t.Fatalf("empty path cannot be set")
}

// referencedRecord is one node of the coherent record graph built below: its
// name, its self field, and the object bytes a store would hold.
type referencedRecord struct {
	name      string
	selfField SelfField
	object    map[string]any
}

// crossRecordGraph builds one internally coherent session lineage in which every
// cross-record member holds the recomputed omit-self digest of the record it
// names, rather than a placeholder.
//
// The lineage is the Section 5 one: a session record, its provider identity, the
// workspace and provider manifests captured for it, an epoch-1 create lease, the
// session.created event under that lease, the checkpoint that closes over the
// event head and both manifests, and the epoch-2 graceful takeover whose
// handoff base is that checkpoint.
func crossRecordGraph(t *testing.T) ([]referencedRecord, map[string]scalar.Digest) {
	t.Helper()

	digests := make(map[string]scalar.Digest)
	var graph []referencedRecord

	add := func(name string, selfField SelfField, object map[string]any) scalar.Digest {
		digest, field, err := CalculateObjectIdentity(mustJSON(t, object))
		if err != nil {
			t.Fatalf("CalculateObjectIdentity(%s) error = %v", name, err)
		}
		if field != selfField {
			t.Fatalf("CalculateObjectIdentity(%s) self field = %q, want %q", name, field, selfField)
		}
		object[string(selfField)] = digest.String()
		digests[name] = digest
		graph = append(graph, referencedRecord{name: name, selfField: selfField, object: object})
		return digest
	}

	sessionRecord := validSessionRecordV1Object()
	sessionDigest := add("session record", SelfRecordID, sessionRecord)

	providerIdentity := validProviderIdentityRecordObject()
	providerIdentityDigest := add("provider identity", SelfRecordID, providerIdentity)

	providerManifest := validTransferManifestObject("provider")
	providerManifest["provider_identity_record_id"] = providerIdentityDigest.String()
	providerManifestDigest := add("provider manifest", SelfManifestID, providerManifest)

	workspaceManifest := workspaceTreeWithEveryEntryVariant()
	workspaceManifestDigest := add("workspace manifest", SelfManifestID, workspaceManifest)

	createLease := validLeaseRecordObject()
	createLease["epoch"] = json.Number("1")
	createLease["lease_id"] = priorLease
	createLease["reason"] = "create"
	createLease["predecessor_lease_id"] = nil
	createLease["checkpoint_id"] = nil
	add("epoch-1 create lease", SelfRecordID, createLease)

	createdEvent := validSessionEventObject("4.0.0", "session.created")
	createdEvent["lease_epoch"] = json.Number("1")
	createdEvent["lease_id"] = priorLease
	createdEvent["lease_sequence"] = json.Number("1")
	createdEvent["predecessors"] = []any{sessionDigest.String()}
	createdEvent["payload"].(map[string]any)["session_record_id"] = sessionDigest.String()
	eventDigest := add("session.created event", SelfEventID, createdEvent)

	checkpoint := validCheckpointRecordObject(true)
	checkpoint["lease_epoch"] = json.Number("1")
	checkpoint["lease_id"] = priorLease
	checkpoint["event_heads"] = []any{eventDigest.String()}
	checkpoint["workspace_manifest_id"] = workspaceManifestDigest.String()
	checkpoint["provider_manifest_id"] = providerManifestDigest.String()
	checkpointDigest := add("checkpoint", SelfCheckpointID, checkpoint)

	takeoverLease := validLeaseRecordObject()
	takeoverLease["epoch"] = json.Number("2")
	takeoverLease["lease_id"] = leaseID
	takeoverLease["reason"] = "graceful_takeover"
	takeoverLease["predecessor_lease_id"] = priorLease
	takeoverLease["checkpoint_id"] = checkpointDigest.String()
	add("epoch-2 graceful takeover lease", SelfRecordID, takeoverLease)

	return graph, digests
}

// crossRecordReferences are the cross-record members of that graph: the record
// that holds the reference, the JSON path to it, and the record it must name.
func crossRecordReferences() []struct{ holder, path, referenced string } {
	return []struct{ holder, path, referenced string }{
		{"provider manifest", "provider_identity_record_id", "provider identity"},
		{"session.created event", "predecessors.[0]", "session record"},
		{"session.created event", "payload.session_record_id", "session record"},
		{"checkpoint", "event_heads.[0]", "session.created event"},
		{"checkpoint", "workspace_manifest_id", "workspace manifest"},
		{"checkpoint", "provider_manifest_id", "provider manifest"},
		{"epoch-2 graceful takeover lease", "checkpoint_id", "checkpoint"},
	}
}

// TestCrossRecordReferencesResolveByRecomputedDigestAtTheProductionEntry is the
// cross-record reference gate.
//
// Every other sweep in this package validates one object in isolation, which is
// enough to prove a member is a well-formed digest and not enough to prove the
// digest is the referenced record. Section 10.1 is explicit that the storage
// path "MUST be derived from the digest, never from an untrusted display name",
// so a reference is resolvable only if recomputing the referenced record's
// omit-self identity through the production entry reproduces the stored value.
// This builds one coherent lineage, resolves every reference through a
// content-addressed store keyed by recomputed digest, and requires each hop to
// land on the record it names.
func TestCrossRecordReferencesResolveByRecomputedDigestAtTheProductionEntry(t *testing.T) {
	t.Parallel()

	graph, digests := crossRecordGraph(t)

	// Every node must be accepted at both identity entries with the claim the
	// production calculation produced, or the lineage is not a valid store.
	store := make(map[scalar.Digest]string, len(graph))
	for _, record := range graph {
		encoded := mustJSON(t, record.object)
		verified, field, err := VerifyObjectIdentity(encoded)
		if err != nil || field != record.selfField || verified != digests[record.name] {
			t.Fatalf("VerifyObjectIdentity(%s) = %q/%q, %v; want %q/%q",
				record.name, verified, field, err, digests[record.name], record.selfField)
		}
		if previous, collision := store[verified]; collision {
			t.Fatalf("records %q and %q share digest %q; the lineage is degenerate and every hop below is vacuous",
				previous, record.name, verified)
		}
		store[verified] = record.name
	}
	if len(store) != len(graph) {
		t.Fatalf("stored %d of %d lineage records", len(store), len(graph))
	}

	objects := make(map[string]map[string]any, len(graph))
	for _, record := range graph {
		objects[record.name] = record.object
	}

	references := crossRecordReferences()
	if len(references) == 0 {
		t.Fatal("declared zero cross-record references; the sweep proves nothing")
	}
	for _, reference := range references {
		t.Run(reference.holder+" "+reference.path, func(t *testing.T) {
			stored := stringAtPath(t, objects[reference.holder], strings.Split(reference.path, "."))
			parsed, err := scalar.ParseDigest(stored)
			if err != nil {
				t.Fatalf("%s %s = %q, which is not a digest: %v", reference.holder, reference.path, stored, err)
			}
			resolved, present := store[parsed]
			if !present {
				t.Fatalf("%s %s = %q resolves to no record in the lineage store", reference.holder, reference.path, stored)
			}
			if resolved != reference.referenced {
				t.Fatalf("%s %s resolves to %q, want %q", reference.holder, reference.path, resolved, reference.referenced)
			}
		})
	}
}

// TestSubstitutingAReferencedRecordBreaksEveryReferenceToIt is the cross-record
// negative half.
//
// The positive sweep proves the references resolve today. It would also pass if
// identity were derived from the schema, the subject, or any other stable label,
// which is precisely the "untrusted display name" Section 10.1 forbids as a
// storage path: under such a rule a substituted record would keep the same
// address and silently take the original's place. Here one referenced record is
// replaced by a different, individually valid record of the same schema and
// subject, and the reference must stop resolving.
func TestSubstitutingAReferencedRecordBreaksEveryReferenceToIt(t *testing.T) {
	t.Parallel()

	graph, digests := crossRecordGraph(t)
	objects := make(map[string]map[string]any, len(graph))
	for _, record := range graph {
		objects[record.name] = record.object
	}

	// A different provider manifest for the same session: same schema, same
	// kind, same subject, one different diagnostic timestamp. Nothing but the
	// content digest distinguishes it from the manifest the checkpoint names.
	substitute := cloneJSONObject(t, objects["provider manifest"])
	substitute["created_at"] = "2026-08-19T05:15:00.000Z"
	substituteDigest, field, err := CalculateObjectIdentity(mustJSON(t, substitute))
	if err != nil || field != SelfManifestID {
		t.Fatalf("CalculateObjectIdentity(substitute manifest) = %q/%q, %v", substituteDigest, field, err)
	}
	substitute[string(SelfManifestID)] = substituteDigest.String()
	assertIdentityEntriesAcceptShape(t, mustJSON(t, substitute), SelfManifestID)

	original := digests["provider manifest"]
	if substituteDigest == original {
		t.Fatalf("a manifest with a different created_at kept digest %q; identity is derived from a label, not from content",
			original)
	}

	// The checkpoint still names the original bytes, so the substitute is not
	// reachable through it.
	stored := stringAtPath(t, objects["checkpoint"], []string{"provider_manifest_id"})
	if stored != original.String() {
		t.Fatalf("checkpoint provider_manifest_id = %q, want %q", stored, original)
	}
	if stored == substituteDigest.String() {
		t.Fatalf("checkpoint provider_manifest_id resolves the substituted manifest %q", substituteDigest)
	}

	// Rewriting the reference to the substitute changes the checkpoint's own
	// identity, so its stored claim no longer verifies. A reference swap is not
	// a silent edit: it invalidates the referring record.
	tampered := cloneJSONObject(t, objects["checkpoint"])
	tampered["provider_manifest_id"] = substituteDigest.String()
	if _, _, err := VerifyObjectIdentity(mustJSON(t, tampered)); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("VerifyObjectIdentity(checkpoint with swapped manifest reference) error = %v, want identity refusal", err)
	}
}

func stringAtPath(t *testing.T, object map[string]any, path []string) string {
	t.Helper()

	var cursor any = object
	for _, step := range path {
		if strings.HasPrefix(step, "[") {
			array, ok := cursor.([]any)
			if !ok {
				t.Fatalf("path step %q expects an array, found %T", step, cursor)
			}
			var offset int
			if _, err := fmt.Sscanf(step, "[%d]", &offset); err != nil {
				t.Fatalf("parse array index %q: %v", step, err)
			}
			cursor = array[offset]
			continue
		}
		container, ok := cursor.(map[string]any)
		if !ok {
			t.Fatalf("path step %q expects an object, found %T", step, cursor)
		}
		cursor = container[step]
	}
	text, ok := cursor.(string)
	if !ok {
		t.Fatalf("path %v holds %T, want string", path, cursor)
	}
	return text
}
