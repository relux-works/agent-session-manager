package traceability

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// expectedV070Source is the reviewed normative source the adopted v0.7.0
// registry must bind. It repeats the provenance the pin leaf verified from a
// fresh upstream clone (tag v0.7.0, peeled commit,LF document digest) so a
// registry that drifts to another source fails here before any binding is read.
var expectedV070Source = registrySource{
	Repository:     "relux-works/agent-session-manager-spec",
	Release:        "v0.7.0",
	Commit:         "32b3f2ba7c377248a53cd42389abbd2f1c321834",
	DocumentPath:   "SPEC.md",
	DocumentSHA256: "c6b2fe64ee79ed697a96ed27a1679c80b8ee1eba137feb99e7738b4da289ddcf",
}

// newV070Bindings is the exact set of section bindings the v0.7.0 adoption
// may add over the trunk v0.6.0 registry. Every other v0.7.0 binding must be
// carried, and every binding here must name its implementing story, so a
// self-minted implementation claim has nowhere to hide.
var newV070Bindings = map[string]string{
	"section:13.1":  "STORY-260916-3fjjtn",
	"section:13.10": "STORY-260916-3aukw3",
	"section:14.1":  "STORY-260916-3fjjtn",
}

// newV070Cases is the exact set of acceptance cases the v0.7.0 adoption may
// add. Carried cases must otherwise be byte-equal to the v0.6.0 registry,
// except for the two intended repoints asserted inline below.
var newV070Cases = map[string]struct{}{
	"catalog-v070-exact":      {},
	"source-pin-v070-exact":   {},
	"source-pin-v070-refusal": {},
}

// rewordedV070Gaps maps the carried bindings whose gap the adoption rewords
// to the story that must own the new text. appendix-d carries a split
// attribution ("pending STORY-...") rather than the owner formula.
var rewordedV070Gaps = map[string]string{
	"section:5.1":        "STORY-260916-2q85nu",
	"section:7.3":        "STORY-260916-1ea7od",
	"section:7.4":        "STORY-260916-1ea7od",
	"section:7.5":        "STORY-260916-1kp1lx",
	"section:appendix-d": "STORY-260916-vucwn0",
}

const (
	launchPlanContractKey = "Launch Plan request [urn:ax:schema:launch-plan-request]"
	launchPlanFixtureKey  = "pin:ax-launch-plan-request-v1"
)

// TestV070RegistryRederivesFromTrunkV060Registry re-derives the adopted v0.7.0
// ownership registry from the trunk-final v0.6.0 registry and fails on any
// missing binding, any unshifted clause line, or any self-minted claim.
//
// The v0.6.0 file in this tree is the trunk baseline: the suite's historical
// byte-identity gate keeps it equal to trunk, and this test treats it as the
// derivation source. Production VerifyRepository pins the exact adopted bytes
// through reviewedOwnershipCanonicalSHA256; this test pins the derivation
// relationship between the two registries, so neither a dropped carry nor an
// unreviewed addition can pass as a re-derivation.
//
// Post-adoption stories upgrade the adopted registry deliberately through
// storyUpgradedV070Bindings and storyNewV070Cases: each named upgrade is
// pinned literally and re-measured against the pinned v0.7.0 document, while
// every other binding and case still rederives byte-equal.
func TestV070RegistryRederivesFromTrunkV060Registry(t *testing.T) {
	t.Parallel()

	previous := decodeRegistryFile(t, "ownership.v0.6.0.json")
	current := decodeRegistryFile(t, "ownership.v0.7.0.json")

	if current.Source != expectedV070Source {
		t.Fatalf("v0.7.0 registry source = %+v, want %+v", current.Source, expectedV070Source)
	}

	previousBindings := sectionBindingsByKey(t, previous)
	currentBindings := sectionBindingsByKey(t, current)

	assertNoMissingBindings(t, previousBindings, currentBindings)
	assertNewBindingsAreReviewed(t, previousBindings, currentBindings)
	assertCarriedBindingsRederive(t, previousBindings, currentBindings)
	assertClauseLinesRemeasured(t, previousBindings, currentBindings)
	assertAcceptanceCasesRederive(t, previous, current)
	assertOwnershipGroupsRederive(t, previous, current)
	assertUnownedSectionsCarried(t, previous, current)
}

func decodeRegistryFile(t *testing.T, name string) ownershipRegistry {
	t.Helper()

	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	registry, err := decodeOwnershipRegistry(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return registry
}

func sectionBindingsByKey(t *testing.T, registry ownershipRegistry) map[string]ownershipGroup {
	t.Helper()

	result := make(map[string]ownershipGroup)
	for _, group := range registry.Ownership {
		if group.Kind != ownershipSectionBinding {
			continue
		}
		if len(group.Keys) != 1 {
			t.Fatalf("section binding group carries %d keys %q, want exactly 1", len(group.Keys), group.Keys)
		}
		key := group.Keys[0]
		if _, duplicate := result[key]; duplicate {
			t.Fatalf("duplicate section binding %q", key)
		}
		result[key] = group
	}
	return result
}

func assertNoMissingBindings(t *testing.T, previous, current map[string]ownershipGroup) {
	t.Helper()

	var missing []string
	for key := range previous {
		if _, ok := current[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("v0.7.0 registry drops %d carried bindings: %q", len(missing), missing)
	}
}

func assertNewBindingsAreReviewed(t *testing.T, previous, current map[string]ownershipGroup) {
	t.Helper()

	for key, group := range current {
		if _, carried := previous[key]; carried {
			continue
		}
		story, ok := newV070Bindings[key]
		if !ok {
			t.Fatalf("v0.7.0 binding %q is not carried and not a reviewed addition", key)
		}
		if !strings.Contains(group.Gap, "Pending implementation owner: "+story) {
			t.Fatalf("v0.7.0 binding %q gap does not name its implementing story %s: %q", key, story, group.Gap)
		}
		if group.Coverage != coverageUnevidenced {
			t.Fatalf("v0.7.0 binding %q claims %s coverage, want unevidenced for a story-owned stub", key, group.Coverage)
		}
		if len(group.Clauses) != 0 {
			t.Fatalf("v0.7.0 binding %q enumerates %d clauses, want none for a story-owned stub", key, len(group.Clauses))
		}
	}
	for key := range newV070Bindings {
		if _, ok := current[key]; !ok {
			t.Fatalf("reviewed v0.7.0 binding %q is absent", key)
		}
	}
}

func assertCarriedBindingsRederive(t *testing.T, previous, current map[string]ownershipGroup) {
	t.Helper()

	for key, old := range previous {
		now, ok := current[key]
		if !ok {
			continue
		}
		if upgrade, upgraded := storyUpgradedV070Bindings[key]; upgraded {
			assertStoryBindingUpgrade(t, key, old, now, upgrade)
			continue
		}
		if now.Production != old.Production {
			t.Fatalf("binding %q production moved from %+v to %+v", key, old.Production, now.Production)
		}
		if !reflect.DeepEqual(now.AcceptanceCases, old.AcceptanceCases) {
			t.Fatalf("binding %q acceptance cases moved from %q to %q", key, old.AcceptanceCases, now.AcceptanceCases)
		}
		if now.Coverage != old.Coverage {
			t.Fatalf("binding %q coverage moved from %s to %s", key, old.Coverage, now.Coverage)
		}
		story, reworded := rewordedV070Gaps[key]
		if !reworded {
			if now.Gap != old.Gap {
				t.Fatalf("binding %q gap changed without a reviewed rewording", key)
			}
			continue
		}
		if now.Gap == old.Gap {
			t.Fatalf("binding %q gap was not reworded to its story owner", key)
		}
		if !strings.Contains(now.Gap, story) {
			t.Fatalf("binding %q reworded gap does not name %s: %q", key, story, now.Gap)
		}
	}
}

// assertClauseLinesRemeasured checks every carried clause against the pinned
// document it cites, using the production clause inventory and quote matcher.
// A clause whose v0.7.0 line still points at the v0.6.0 position fails twice:
// the measured line differs and the excerpt no longer begins there.
func assertClauseLinesRemeasured(t *testing.T, previous, current map[string]ownershipGroup) {
	t.Helper()

	document060, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("LoadV060: %v", err)
	}
	document070, err := specdoc.LoadV070()
	if err != nil {
		t.Fatalf("LoadV070: %v", err)
	}
	measured060 := measuredClauseLines(t, document060, previous, "v0.6.0")
	measured070 := measuredClauseLines(t, document070, current, "v0.7.0")

	for key, old := range previous {
		now, ok := current[key]
		if !ok {
			continue
		}
		if _, upgraded := storyUpgradedV070Bindings[key]; upgraded {
			assertStoryBindingClausesRemeasured(t, key, old, now, measured070)
			continue
		}
		if len(now.Clauses) != len(old.Clauses) {
			t.Fatalf("binding %q carries %d clauses, want %d", key, len(now.Clauses), len(old.Clauses))
		}
		for index, oldClause := range old.Clauses {
			newClause := now.Clauses[index]
			if newClause.ID != oldClause.ID {
				t.Fatalf("binding %q clause %d id moved from %s to %s", key, index, oldClause.ID, newClause.ID)
			}
			if newClause.Excerpt != oldClause.Excerpt {
				t.Fatalf("binding %q clause %s excerpt changed", key, oldClause.ID)
			}
			if !reflect.DeepEqual(newClause.AcceptanceCases, oldClause.AcceptanceCases) {
				t.Fatalf("binding %q clause %s acceptance cases moved", key, oldClause.ID)
			}
			if newClause.Line == oldClause.Line {
				t.Fatalf("binding %q clause %s still cites v0.6.0 line %d; every carried clause shifts in v0.7.0",
					key, oldClause.ID, oldClause.Line)
			}
			if measured060[key][oldClause.ID] != oldClause.Line {
				t.Fatalf("binding %q clause %s v0.6.0 line %d is not the measured line %d",
					key, oldClause.ID, oldClause.Line, measured060[key][oldClause.ID])
			}
			if measured070[key][newClause.ID] != newClause.Line {
				t.Fatalf("binding %q clause %s v0.7.0 line %d is not the measured line %d",
					key, newClause.ID, newClause.Line, measured070[key][newClause.ID])
			}
		}
	}
}

// measuredClauseLines returns the production-measured line of every normative
// clause the pinned document carries for the given bindings, verifying each
// binding's declared excerpts quote verbatim at their declared lines.
func measuredClauseLines(t *testing.T, document *specdoc.Document, bindings map[string]ownershipGroup, tag string) map[string]map[string]int {
	t.Helper()

	result := make(map[string]map[string]int, len(bindings))
	for key, group := range bindings {
		if len(group.Clauses) == 0 {
			continue
		}
		inventory, err := sectionClauseInventory(document, key)
		if err != nil {
			t.Fatalf("%s binding %q inventory: %v", tag, key, err)
		}
		lines := make(map[string]int, len(inventory))
		for _, clause := range inventory {
			lines[clause.ID] = clause.Line
		}
		for _, discharged := range group.Clauses {
			if _, ok := lines[discharged.ID]; !ok {
				t.Fatalf("%s binding %q clause %s is not a measured clause of its section", tag, key, discharged.ID)
			}
			if !quoteBeginsAtLine(document, discharged.Excerpt, discharged.Line) {
				t.Fatalf("%s binding %q clause %s excerpt does not begin at line %d", tag, key, discharged.ID, discharged.Line)
			}
		}
		result[key] = lines
	}
	return result
}

func assertAcceptanceCasesRederive(t *testing.T, previous, current ownershipRegistry) {
	t.Helper()

	oldCases := make(map[string]acceptanceCase, len(previous.AcceptanceCases))
	for _, c := range previous.AcceptanceCases {
		oldCases[c.ID] = c
	}
	newCases := make(map[string]acceptanceCase, len(current.AcceptanceCases))
	for _, c := range current.AcceptanceCases {
		if _, duplicate := newCases[c.ID]; duplicate {
			t.Fatalf("duplicate v0.7.0 acceptance case %q", c.ID)
		}
		newCases[c.ID] = c
	}
	for id := range oldCases {
		if _, ok := newCases[id]; !ok {
			t.Fatalf("v0.7.0 registry drops carried acceptance case %q", id)
		}
	}
	for id := range newCases {
		if _, carried := oldCases[id]; carried {
			continue
		}
		if _, ok := newV070Cases[id]; !ok {
			if _, ok := storyNewV070Cases[id]; !ok {
				t.Fatalf("v0.7.0 acceptance case %q is not carried and not a reviewed addition", id)
			}
		}
	}
	for id := range newV070Cases {
		if _, ok := newCases[id]; !ok {
			t.Fatalf("reviewed v0.7.0 acceptance case %q is absent", id)
		}
	}
	for id, want := range storyNewV070Cases {
		got, ok := newCases[id]
		if !ok {
			t.Fatalf("story acceptance case %q is absent", id)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("story acceptance case %q differs from the reviewed upgrade: %+v", id, got)
		}
	}
	for id, old := range oldCases {
		now := newCases[id]
		if reflect.DeepEqual(now, old) {
			continue
		}
		switch id {
		case "catalog-v060-exact":
			// The adopted catalog moved to v0.7.0, so the v0.6.0 exact case
			// is repointed at the historical projection pair it now proves.
			want := acceptanceCase{
				ID:         id,
				Production: codeReference{Path: "internal/catalog/catalog.go", Declaration: "ForRelease"},
				Tests: []codeReference{
					{Path: "internal/catalog/catalog_test.go", Declaration: "TestV060ProjectionMatchesHistoricalLock"},
				},
			}
			if !reflect.DeepEqual(now, want) {
				t.Fatalf("catalog-v060-exact repoint = %+v, want %+v", now, want)
			}
		case "assigned-section-binding":
			assertAssignedSectionBindingRename(t, old, now)
		default:
			t.Fatalf("carried acceptance case %q changed without a reviewed repoint", id)
		}
	}
}

// assertAssignedSectionBindingRename allows exactly one change to the carried
// case: the V060 section-identifier test becomes its V070 successor. Any other
// edit, addition, or removal fails.
func assertAssignedSectionBindingRename(t *testing.T, old, now acceptanceCase) {
	t.Helper()

	if now.ID != old.ID || now.Production != old.Production {
		t.Fatalf("assigned-section-binding identity moved: %+v vs %+v", now, old)
	}
	if len(now.Tests) != len(old.Tests) {
		t.Fatalf("assigned-section-binding test list length moved from %d to %d", len(old.Tests), len(now.Tests))
	}
	renamed := 0
	for index, oldRef := range old.Tests {
		newRef := now.Tests[index]
		if newRef == oldRef {
			continue
		}
		renamed++
		if newRef.Path != oldRef.Path {
			t.Fatalf("assigned-section-binding test %d moved path %q to %q", index, oldRef.Path, newRef.Path)
		}
		if oldRef.Declaration != "TestMainRejectsSyntacticallyValidNonexistentV060Section" ||
			newRef.Declaration != "TestMainRejectsSyntacticallyValidNonexistentV070Section" {
			t.Fatalf("assigned-section-binding test %d renamed %q to %q, want the single V060 to V070 rename",
				index, oldRef.Declaration, newRef.Declaration)
		}
	}
	if renamed != 1 {
		t.Fatalf("assigned-section-binding carries %d test edits, want exactly the V060 to V070 rename", renamed)
	}
}

func ownershipGroupByKindAndKeys(t *testing.T, registry ownershipRegistry, kind ownershipKind, firstKey string) ownershipGroup {
	t.Helper()

	for _, group := range registry.Ownership {
		if group.Kind == kind && len(group.Keys) > 0 && group.Keys[0] == firstKey {
			return group
		}
	}
	t.Fatalf("%s group starting with %q is absent", kind, firstKey)
	return ownershipGroup{}
}

func assertOwnershipGroupsRederive(t *testing.T, previous, current ownershipRegistry) {
	t.Helper()

	assertContractGroupRederives(t, previous, current)
	assertSourceGroupRederives(t, previous, current)

	// The catalog normative-section group and the catalog fixture group are
	// adoption-independent: identical keys, owners, and cases.
	for _, probe := range []struct {
		kind     ownershipKind
		firstKey string
	}{
		{ownershipNormativeSection, "catalog:10.8.5"},
		{ownershipFixture, "catalog:Appendix D: Directory Node operation body"},
	} {
		old := ownershipGroupByKindAndKeys(t, previous, probe.kind, probe.firstKey)
		now := ownershipGroupByKindAndKeys(t, current, probe.kind, probe.firstKey)
		if !reflect.DeepEqual(now, old) {
			t.Fatalf("%s group %q changed: %+v vs %+v", probe.kind, probe.firstKey, now, old)
		}
	}

	assertPinFixtureGroupRederives(t, previous, current)
}

func assertContractGroupRederives(t *testing.T, previous, current ownershipRegistry) {
	t.Helper()

	old := ownershipGroupByKindAndKeys(t, previous, ownershipContract, "Configuration [urn:ax:schema:config]")
	now := ownershipGroupByKindAndKeys(t, current, ownershipContract, "Configuration [urn:ax:schema:config]")
	if now.Production != old.Production {
		t.Fatalf("contract group production moved from %+v to %+v", old.Production, now.Production)
	}
	wantCases := []string{"catalog-v070-exact", "catalog-v060-exact", "catalog-v050-exact", "catalog-v043-compatibility"}
	if !reflect.DeepEqual(now.AcceptanceCases, wantCases) {
		t.Fatalf("contract group acceptance cases = %q, want %q", now.AcceptanceCases, wantCases)
	}
	oldKeys := oldKeySet(old.Keys)
	for _, key := range now.Keys {
		if _, carried := oldKeys[key]; !carried && key != launchPlanContractKey {
			t.Fatalf("contract group key %q is not carried and not the Launch Plan request row", key)
		}
		delete(oldKeys, key)
	}
	if len(oldKeys) != 0 {
		t.Fatalf("contract group drops %d carried keys", len(oldKeys))
	}
	found := false
	for _, key := range now.Keys {
		if key == launchPlanContractKey {
			found = true
		}
	}
	if !found {
		t.Fatalf("contract group lacks the Launch Plan request row %q", launchPlanContractKey)
	}
}

func oldKeySet(keys []string) map[string]struct{} {
	result := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		result[key] = struct{}{}
	}
	return result
}

func assertSourceGroupRederives(t *testing.T, previous, current ownershipRegistry) {
	t.Helper()

	old := ownershipGroupByKindAndKeys(t, previous, ownershipNormativeSection, "source:1")
	now := ownershipGroupByKindAndKeys(t, current, ownershipNormativeSection, "source:1")
	if !reflect.DeepEqual(now.Keys, old.Keys) {
		t.Fatalf("source normative-section keys moved")
	}
	wantCases := []string{"source-pin-v070-exact", "source-pin-v070-refusal"}
	if !reflect.DeepEqual(now.AcceptanceCases, wantCases) {
		t.Fatalf("source group acceptance cases = %q, want %q", now.AcceptanceCases, wantCases)
	}
	// FINDING F1 (TASK-260917-10k118): the reviewed v0.7.0 bytes keep the
	// source group's production owner at VerifyV060 while its cases moved to
	// the v0.7.0 pair. The v0.5.0→v0.6.0 precedent moved Verify to VerifyV060,
	// so the adoption pattern says VerifyV070; the gate only checks that the
	// named declaration exists, which is why the bytes are green. The
	// reconstruction must not rewrite reviewed bytes, so this test pins the
	// carried reference literally: a future fix to VerifyV070 must update
	// this pin deliberately, and any other drift fails here.
	if now.Production.Declaration != "VerifyV060" || now.Production.Path != "internal/specpin/pin.go" {
		t.Fatalf("source group production = %+v, want the carried VerifyV060 reference (see F1)", now.Production)
	}
}

func assertPinFixtureGroupRederives(t *testing.T, previous, current ownershipRegistry) {
	t.Helper()

	old := ownershipGroupByKindAndKeys(t, previous, ownershipFixture, "pin:ax-session-directory-conformance-v1")
	now := ownershipGroupByKindAndKeys(t, current, ownershipFixture, "pin:ax-session-directory-conformance-v1")
	if now.Production != old.Production {
		t.Fatalf("pin fixture group production moved from %+v to %+v", old.Production, now.Production)
	}
	wantCases := []string{"source-pin-exact", "source-pin-refusal", "source-pin-v060-exact", "source-pin-v060-refusal", "source-pin-v070-exact", "source-pin-v070-refusal"}
	if !reflect.DeepEqual(now.AcceptanceCases, wantCases) {
		t.Fatalf("pin fixture group acceptance cases = %q, want %q", now.AcceptanceCases, wantCases)
	}
	oldKeys := oldKeySet(old.Keys)
	for _, key := range now.Keys {
		if _, carried := oldKeys[key]; !carried && key != launchPlanFixtureKey {
			t.Fatalf("pin fixture group key %q is not carried and not the Launch Plan fixture", key)
		}
		delete(oldKeys, key)
	}
	if len(oldKeys) != 0 {
		t.Fatalf("pin fixture group drops %d carried keys", len(oldKeys))
	}
}

func assertUnownedSectionsCarried(t *testing.T, previous, current ownershipRegistry) {
	t.Helper()

	if len(current.UnownedSections) != len(previous.UnownedSections) {
		t.Fatalf("unowned sections length moved from %d to %d", len(previous.UnownedSections), len(current.UnownedSections))
	}
	for index, old := range previous.UnownedSections {
		now := current.UnownedSections[index]
		if now.Key != old.Key {
			t.Fatalf("unowned section %d key moved from %q to %q", index, old.Key, now.Key)
		}
		if now == old {
			continue
		}
		if now.Key != "section:11.10.5" {
			t.Fatalf("unowned section %q changed without a reviewed rewording", now.Key)
		}
		if now.Evidence != old.Evidence {
			t.Fatalf("unowned section 11.10.5 evidence changed")
		}
		if !strings.Contains(now.Gap, "Disclosure owner: TASK-260830-2x16gz") {
			t.Fatalf("unowned section 11.10.5 gap lacks the Disclosure owner rewording: %q", now.Gap)
		}
	}
}

// storyBindingUpgrade is the reviewed post-adoption upgrade of one carried
// v0.7.0 binding. The mesh-rpc-framing-and-negotiation story final leaf
// (STORY-260830-4qojoz / TASK-260830-19bjfj) moves three unevidenced stubs to
// measured coverage; every field below is pinned literally, so the
// re-derivation gate keeps failing on any drift outside the reviewed
// upgrade. Section 11.3 is deliberately absent: no test drives a
// digest-identity array, so it stays the carried unevidenced stub.
type storyBindingUpgrade struct {
	Production codeReference
	Cases      []string
	Coverage   coverageLevel
	Gap        string
	Clauses    []dischargedClause
}

var storyUpgradedV070Bindings = map[string]storyBindingUpgrade{
	"section:11.2": {
		Production: codeReference{Path: "internal/rpcwire/envelope.go", Declaration: "DecodeRequest"},
		Cases:      []string{"rpcwire-closed-bodies", "rpcwire-framing", "catalog-v050-exact"},
		Coverage:   coveragePartial,
		Gap:        "Section 11.2 clauses 11.2#1 and 11.2#5 (hello-first sequencing) are executed only for the RPC-5 Host Channel responder: DecodeRequest is a stateless codec with no handshake memory, and no legacy `ax rpc serve` responder exists in the tree, so the legacy sequencing lane stays a stated bound while the codec-enforced clauses discharge above.",
		Clauses: []dischargedClause{
			{ID: "11.2#2", Line: 7364, Excerpt: "state, Structured Error, Observation Event, and CLI Result MUST NOT appear in", AcceptanceCases: []string{"rpcwire-closed-bodies"}},
			{ID: "11.2#3", Line: 7408, Excerpt: "<code>ok = false</code> and one Structured Error and MUST omit body. Unknown", AcceptanceCases: []string{"rpcwire-framing"}},
			{ID: "11.2#4", Line: 7420, Excerpt: "at least the values shown and MUST refuse a peer below them. The 5 MiB object", AcceptanceCases: []string{"rpcwire-closed-bodies"}},
		},
	},
	"section:17.1": {
		Production: codeReference{Path: "internal/meshneg/negotiate.go", Declaration: "Negotiate"},
		Cases:      []string{"meshneg-selection", "rpcwire-version-boundary", "config-versioned-readers"},
		Coverage:   coveragePartial,
		Gap:        "Section 17.1 clauses 17.1#2 (minor-version semantic preservation) and 17.1#6 (v1 materialization upgrade) are outside Negotiate, which selects majors only: no minor-selection entry exists, immutable-object namespaced extensions (17.1#3) belong to the object-exchange owners, and materialization upgrade belongs to the matjournal and provhost owners. The retained config-versioned-readers case keeps the config Decode lanes traceable.",
		Clauses: []dischargedClause{
			{ID: "17.1#1", Line: 15567, Excerpt: "- a major increment MAY break syntax or semantics and MUST require explicit", AcceptanceCases: []string{"meshneg-selection"}},
			{ID: "17.1#4", Line: 15589, Excerpt: "highest mutually supported minor within a common major; they MUST NOT select a", AcceptanceCases: []string{"meshneg-selection"}},
			{ID: "17.1#5", Line: 15602, Excerpt: "immutable v0.1.0 contracts and MUST NOT be interpreted using the corrected", AcceptanceCases: []string{"meshneg-selection", "rpcwire-version-boundary"}},
		},
	},
	"section:17.4": {
		Production: codeReference{Path: "internal/meshneg/negotiate.go", Declaration: "Negotiate"},
		Cases:      []string{"story-4qojoz-mesh-negotiation", "config-read-only-downgrade", "config-durable-migration"},
		Coverage:   coverageSliver,
		Gap:        "Section 17.4 clauses 17.4#1 through 17.4#3 (upgrade checks, downgraded-binary read-only mode, no resume or transfer) are outside Negotiate: no upgrade, downgrade, or auto-resume flow exists in the tree, so only the activation-unavailable exposure clause discharges above. The retained config cases keep the AssessCompatibility downgrade-assessment lanes traceable.",
		Clauses: []dischargedClause{
			{ID: "17.4#4", Line: 15661, Excerpt: "immutable evidence; it MUST report activation unavailable rather than omit the", AcceptanceCases: []string{"story-4qojoz-mesh-negotiation"}},
		},
	},
}

// storyNewV070Cases is the exact set of acceptance cases the story final leaf
// adds to the adopted registry. Each is pinned literally; any further
// addition still fails as an unreviewed claim.
var storyNewV070Cases = map[string]acceptanceCase{
	"rpcwire-closed-bodies": {
		ID:         "rpcwire-closed-bodies",
		Production: codeReference{Path: "internal/rpcwire/envelope.go", Declaration: "DecodeRequest"},
		Tests: []codeReference{
			{Path: "internal/rpcwire/adversarial_test.go", Declaration: "TestClosedBodyMemberSweep"},
			{Path: "internal/rpcwire/adversarial_test.go", Declaration: "TestClosedBodyBoundsWitnessed"},
			{Path: "internal/rpcwire/envelope_test.go", Declaration: "TestHelloRefusals"},
		},
	},
	"rpcwire-framing": {
		ID:         "rpcwire-framing",
		Production: codeReference{Path: "internal/rpcwire/envelope.go", Declaration: "DecodeResponse"},
		Tests: []codeReference{
			{Path: "internal/rpcwire/envelope_test.go", Declaration: "TestHistoricalFailureBindings"},
			{Path: "internal/rpcwire/envelope_test.go", Declaration: "TestBootstrapRejectionFraming"},
		},
	},
	"meshneg-selection": {
		ID:         "meshneg-selection",
		Production: codeReference{Path: "internal/meshneg/negotiate.go", Declaration: "Negotiate"},
		Tests: []codeReference{
			{Path: "internal/meshneg/adversarial_test.go", Declaration: "TestNoMajorSelectedByCoercion"},
			{Path: "internal/meshneg/adversarial_test.go", Declaration: "TestMajor1RejectedThroughNegotiation"},
			{Path: "internal/meshneg/negotiate_test.go", Declaration: "TestNegotiateMatrix"},
		},
	},
	"rpcwire-version-boundary": {
		ID:         "rpcwire-version-boundary",
		Production: codeReference{Path: "internal/rpcwire/envelope.go", Declaration: "DecodeRequest"},
		Tests: []codeReference{
			{Path: "internal/rpcwire/adversarial_test.go", Declaration: "TestMajor1RejectedNotReinterpreted"},
			{Path: "internal/rpcwire/adversarial_test.go", Declaration: "TestErrorSchemaNotNegotiated"},
		},
	},
	"story-4qojoz-mesh-negotiation": {
		ID:         "story-4qojoz-mesh-negotiation",
		Production: codeReference{Path: "internal/meshneg/negotiate.go", Declaration: "Negotiate"},
		Tests: []codeReference{
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestPeerExposure"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestExposureCarriesNoInventory"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestRefusalClassesResolveFromTheRegistry"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestNegotiationIsNotAdmission"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestNoFallbackSurface"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestSemverAgreementWithRPCWire"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestConfig4NeverDowngrades"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestLegacyNeverSelectsRPC5"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestNegotiationIsDeterministic"},
			{Path: "internal/meshneg/conformance_test.go", Declaration: "TestRefusalShape"},
			{Path: "internal/meshneg/adversarial_test.go", Declaration: "TestPeerOfferKeyMembershipPerKey"},
			{Path: "internal/meshneg/adversarial_test.go", Declaration: "TestNegotiateOfferRefusalCarriesLocal"},
		},
	},
}

// assertStoryBindingUpgrade pins one reviewed post-adoption upgrade: the
// v0.6.0 base must be the unevidenced stub the adoption carried, and the
// v0.7.0 binding must equal the reviewed upgrade literally.
func assertStoryBindingUpgrade(t *testing.T, key string, old, now ownershipGroup, upgrade storyBindingUpgrade) {
	t.Helper()

	if old.Coverage != coverageUnevidenced || len(old.Clauses) != 0 {
		t.Fatalf("binding %q upgrade base is not the unevidenced stub: coverage %s with %d clauses", key, old.Coverage, len(old.Clauses))
	}
	if now.Production != upgrade.Production {
		t.Fatalf("binding %q upgraded production = %+v, want %+v", key, now.Production, upgrade.Production)
	}
	if !reflect.DeepEqual(now.AcceptanceCases, upgrade.Cases) {
		t.Fatalf("binding %q upgraded acceptance cases = %q, want %q", key, now.AcceptanceCases, upgrade.Cases)
	}
	if now.Coverage != upgrade.Coverage {
		t.Fatalf("binding %q upgraded coverage = %s, want %s", key, now.Coverage, upgrade.Coverage)
	}
	if now.Gap != upgrade.Gap {
		t.Fatalf("binding %q upgraded gap differs from the reviewed upgrade", key)
	}
	if !reflect.DeepEqual(now.Clauses, upgrade.Clauses) {
		t.Fatalf("binding %q upgraded clauses differ from the reviewed upgrade", key)
	}
}

// assertStoryBindingClausesRemeasured checks the upgraded binding's new
// clauses against the pinned v0.7.0 document through the production clause
// inventory. Excerpt-verbatim at the declared line is already enforced by
// measuredClauseLines over the current bindings; this pins each declared
// line to its measured position.
func assertStoryBindingClausesRemeasured(t *testing.T, key string, old, now ownershipGroup, measured070 map[string]map[string]int) {
	t.Helper()

	if len(old.Clauses) != 0 {
		t.Fatalf("binding %q upgrade base carries %d clauses, want the unevidenced stub", key, len(old.Clauses))
	}
	for _, newClause := range now.Clauses {
		if measured070[key][newClause.ID] != newClause.Line {
			t.Fatalf("binding %q clause %s v0.7.0 line %d is not the measured line %d",
				key, newClause.ID, newClause.Line, measured070[key][newClause.ID])
		}
	}
}
