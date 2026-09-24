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

// TestStoryV070AcceptanceCasesHaveClauseEdges prevents a declaration-only
// addition from being mistaken for measured coverage. Each Story leaf case
// added by the tmux backend Story must be listed on an executed clause edge.
func TestStoryV070AcceptanceCasesHaveClauseEdges(t *testing.T) {
	t.Parallel()

	current := decodeRegistryFile(t, "ownership.v0.7.0.json")
	bindings := sectionBindingsByKey(t, current)
	wantEdges := map[string]map[string]string{
		"tmux-private-server-management-v070": {
			"section:4.2": "4.2#4,4.2#5,4.2#6,4.2#7",
		},
		"tmux-lifecycle-operations-v070": {
			"section:4.2": "4.2#9",
			"section:4.C": "4.C#3,4.C#4,4.C#7",
		},
		"tmux-attach-overlap-v070": {
			"section:4.C": "4.C#6",
		},
		"tmux-attach-semantics-v070": {
			"section:4.C": "4.C#3,4.C#5,4.C#6,4.C#7",
		},
		"tmux-socket-custody-v070": {
			"section:3.2": "3.2#8",
		},
	}

	for acceptanceID, sections := range wantEdges {
		for bindingKey, clauseIDs := range sections {
			binding, ok := bindings[bindingKey]
			if !ok {
				t.Fatalf("acceptance case %q requires absent binding %s", acceptanceID, bindingKey)
			}
			groupHasCase := false
			for _, id := range binding.AcceptanceCases {
				if id == acceptanceID {
					groupHasCase = true
					break
				}
			}
			if !groupHasCase {
				t.Fatalf("acceptance case %q is absent from binding %s", acceptanceID, bindingKey)
			}
			for _, clause := range binding.Clauses {
				if !strings.Contains(","+clauseIDs+",", ","+clause.ID+",") {
					continue
				}
				for _, id := range clause.AcceptanceCases {
					if id == acceptanceID {
						clauseIDs = strings.ReplaceAll(clauseIDs, clause.ID, "")
						break
					}
				}
			}
			for _, clauseID := range strings.Split(clauseIDs, ",") {
				if clauseID != "" {
					t.Errorf("acceptance case %q has no decoded clause edge at %s %s", acceptanceID, bindingKey, clauseID)
				}
			}
		}
	}
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

func TestSection24LosingLeaseClausePointsAtDerivationAuthority(t *testing.T) {
	registry := decodeRegistryFile(t, "ownership.v0.7.0.json")
	group := ownershipGroupByKindAndKeys(t, registry, ownershipSectionBinding, "section:2.4")
	wantOwner := codeReference{Path: "internal/sessprofile/authority.go", Declaration: "Derive"}
	if group.Production != wantOwner {
		t.Fatalf("Section 2.4 owner = %+v, want derivation authority %+v", group.Production, wantOwner)
	}
	wantCase := "story-260922-derivation-side-profile-source"
	if !stringMember(group.AcceptanceCases, wantCase) {
		t.Fatalf("Section 2.4 acceptance cases omit %q: %q", wantCase, group.AcceptanceCases)
	}
	var clause *dischargedClause
	for index := range group.Clauses {
		if group.Clauses[index].ID == "2.4#2" {
			clause = &group.Clauses[index]
			break
		}
	}
	if clause == nil || !stringMember(clause.AcceptanceCases, wantCase) {
		t.Fatalf("decoded 2.4#2 edge = %+v, want acceptance case %q", clause, wantCase)
	}
	var acceptance *acceptanceCase
	for index := range registry.AcceptanceCases {
		if registry.AcceptanceCases[index].ID == wantCase {
			acceptance = &registry.AcceptanceCases[index]
			break
		}
	}
	if acceptance == nil || acceptance.Production != wantOwner {
		t.Fatalf("decoded acceptance case %q = %+v, want production owner %+v", wantCase, acceptance, wantOwner)
	}
}

func stringMember(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
		if upgrade, ok := storyNewV070Bindings[key]; ok {
			assertStoryNewBinding(t, key, group, upgrade)
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
	for key := range storyNewV070Bindings {
		if _, ok := current[key]; !ok {
			t.Fatalf("reviewed story binding %q is absent", key)
		}
	}
}

// assertStoryNewBinding pins a reviewed new-with-proof section binding
// literally: a post-adoption story may introduce a section the trunk
// baseline never bound when the binding carries enumerated proof, and
// the clause lines remeasure in assertClauseLinesRemeasured.
func assertStoryNewBinding(t *testing.T, key string, group ownershipGroup, upgrade storyBindingUpgrade) {
	t.Helper()

	if group.Production != upgrade.Production {
		t.Fatalf("binding %q production = %+v, want %+v", key, group.Production, upgrade.Production)
	}
	if !reflect.DeepEqual(group.AcceptanceCases, upgrade.Cases) {
		t.Fatalf("binding %q acceptance cases = %q, want %q", key, group.AcceptanceCases, upgrade.Cases)
	}
	if group.Coverage != upgrade.Coverage {
		t.Fatalf("binding %q coverage = %s, want %s", key, group.Coverage, upgrade.Coverage)
	}
	if group.Gap != upgrade.Gap {
		t.Fatalf("binding %q gap differs from the reviewed addition", key)
	}
	if !reflect.DeepEqual(group.Clauses, upgrade.Clauses) {
		t.Fatalf("binding %q clauses differ from the reviewed addition", key)
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
	for key := range storyNewV070Bindings {
		now, ok := current[key]
		if !ok {
			continue
		}
		for _, newClause := range now.Clauses {
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
// measured coverage, and the canonical-session-capture-and-evidence story
// final leaf (STORY-260830-1cyj0q / TASK-260830-32ypr2) moves two more stubs
// to full coverage; every field below is pinned literally, so the
// re-derivation gate keeps failing on any drift outside the reviewed
// upgrades. Section 11.3 is deliberately absent: no test drives a
// digest-identity array, so it stays the carried unevidenced stub. Section
// 13.14.1 is likewise absent: it carries no registry binding at all, so
// there is no carried stub to upgrade.
type storyBindingUpgrade struct {
	Production codeReference
	Cases      []string
	Coverage   coverageLevel
	Gap        string
	Clauses    []dischargedClause
}

// storyNewV070Bindings is the exact set of new-with-proof section bindings a
// post-adoption story may introduce for a section the trunk baseline never
// bound. STORY-260830-ptxkqe introduces section 4.1 with lost-create recovery;
// STORY-260830-2t4g7i introduces section 4.2 with its executed tmux clauses.
var storyNewV070Bindings = map[string]storyBindingUpgrade{
	"section:4.2": {
		Production: codeReference{Path: "internal/tmuxserver/acquire.go", Declaration: "Acquire"},
		Cases:      []string{"tmux-lifecycle-operations-v070", "tmux-private-server-management-v070"},
		Coverage:   coverageSliver,
		Gap:        "Acquire discharges Section 4.2 clauses 4.2#4 through 4.2#7 for the dedicated -S server, background no-create policy, broker contact, typed capability_unavailable refusal, and no direct-creation fallback; ExecuteWrapperRestore discharges 4.2#9 for the bounded lease refresh and local-resume/remote-offer/park decision. The remaining clauses stay unenumerated: this repository builds no ax pane command or tmux-resurrect migration, and it does not perform a real GUI/Aqua sentinel plus provider-auth smoke or re-establish that realm after logout/reboot.",
		Clauses: []dischargedClause{
			{ID: "4.2#4", Line: 1417, Excerpt: "AX MUST use its dedicated <code>-S</code> server and MUST NOT discover or reuse", AcceptanceCases: []string{"tmux-private-server-management-v070"}},
			{ID: "4.2#5", Line: 1419, Excerpt: "creation is credential-sensitive. A Background caller MUST NOT create a", AcceptanceCases: []string{"tmux-private-server-management-v070"}},
			{ID: "4.2#6", Line: 1422, Excerpt: "its attested AX tmux server; if neither exists it MUST return", AcceptanceCases: []string{"tmux-private-server-management-v070"}},
			{ID: "4.2#7", Line: 1423, Excerpt: "<code>capability_unavailable</code> with typed realm/readiness details and MUST", AcceptanceCases: []string{"tmux-private-server-management-v070"}},
			{ID: "4.2#9", Line: 1434, Excerpt: "After restore, the wrapper MUST:", AcceptanceCases: []string{"tmux-lifecycle-operations-v070"}},
		},
	},
	"section:4.1": {
		Production: codeReference{Path: "internal/termbind/recover.go", Declaration: "RecoverCreate"},
		Cases:      []string{"terminal-create-recovery-exact"},
		Coverage:   coverageSliver,
		Gap:        "RecoverCreate discharges Section 4.1 clause 4.1#5: after a lost create result, one status read plus the durable (session_id, bootstrap_operation_id) binding proves absence or identifies the ONE child (unavailable with status_first when the effect cannot be disproven, never a second child, never a false absence claim). The other Section 4.1 clauses (backend semantic operations, wrapper validation and safe parking, the durable pair bind) belong to the operation matrix, the axpane wrapper, and the terminal backend receipt owners and stay unenumerated here.",
		Clauses: []dischargedClause{
			{ID: "4.1#5", Line: 1399, Excerpt: "<code>status</code> plus that binding MUST prove absence or identify the one", AcceptanceCases: []string{"terminal-create-recovery-exact"}},
		},
	},
}

var storyUpgradedV070Bindings = map[string]storyBindingUpgrade{
	"section:3.2": {
		Production: codeReference{Path: "internal/tmuxserver/bind.go", Declaration: "CheckSocketCustody"},
		Cases:      []string{"AC-PATH-001", "localstore-path-registry", "localstore-layout-owner-only", "localstore-digest-path-v1", "localstore-immutable-blob-install", "localstore-sqlite-projection", "tmux-socket-custody-v070"},
		Coverage:   coverageSliver,
		Gap:        "CheckSocketCustody discharges Section 3.2 clause 3.2#8 for the socket's kind, owner, private mode, and identity recheck before Probe connects, Lifecycle dispatches, or Spawner unlinks. ResolvePaths and the listed localstore cases remain the path/layout owners; the other Section 3.2 clauses stay unenumerated here.",
		Clauses: []dischargedClause{
			{ID: "3.2#8", Line: 810, Excerpt: "symlink. Before bind, connect, rename, or unlink, AX MUST reject a socket path,", AcceptanceCases: []string{"tmux-socket-custody-v070"}},
		},
	},
	"section:2.4": {
		Production: codeReference{Path: "internal/sessprofile/authority.go", Declaration: "Derive"},
		Cases:      []string{"sessprofile-derive", "sessprofile-derive-heads", "sessprofile-fork-pair", "provhost-mapping-resolution", "story-260917-losing-lease-profile-source", "story-260922-derivation-side-profile-source"},
		Coverage:   coverageFull,
		Gap:        "",
		Clauses: []dischargedClause{
			{ID: "2.4#1", Line: 661, Excerpt: "MUST fail with <code>profile_mapping_unavailable</code> if the adapter cannot map the stored profile for the probed provider version.", AcceptanceCases: []string{"provhost-mapping-resolution"}},
			{ID: "2.4#2", Line: 666, Excerpt: "Losing-lease or ambiguous events MUST NOT change it.", AcceptanceCases: []string{"sessprofile-derive", "story-260917-losing-lease-profile-source", "story-260922-derivation-side-profile-source"}},
			{ID: "2.4#3", Line: 670, Excerpt: "a bundle, resume, or fork MUST NOT fall back to the Session Record creation value when the closure contains a later change.", AcceptanceCases: []string{"sessprofile-derive", "sessprofile-derive-heads"}},
			{ID: "2.4#4", Line: 678, Excerpt: "MUST NOT be treated as a profile event in the new session's event chain.", AcceptanceCases: []string{"sessprofile-fork-pair"}},
		},
	},
	"section:5.3": {
		Production: codeReference{Path: "internal/sessrepo/lease_store.go", Declaration: "CompareAndSwapLease"},
		Cases:      []string{"lease-record-lifecycle", "lease-fencing-revalidation", "lease-checkpoint-admission", "lease-divergent-preservation", "lease-union-resolution", "lease-fencing-gates", "core-record-identity-validation", "lease-ownership-union-properties", "lease-ownership-store-properties", "story-260917-append-winning-lease-admission"},
		Coverage:   coveragePartial,
		Gap:        "CompareAndSwapLease with the fencing gates discharges 7 of the 8 Section 5.3 clauses; clause 5.3#5 (every new takeover lease MUST use max_observed_epoch + 1 from the initiator's union) stays unimplemented because the union maximum is caller-side input and no takeover flow exists in this repository yet. The ownership property cases (union-order independence and loser preservation over Reduce, clock non-authority and single authority over the lease store) add executable proofs for the discharged 5.3#6 and 5.3#7 divergent-preservation evidence without changing the 7-of-8 ratio.",
		Clauses: []dischargedClause{
			{ID: "5.3#1", Line: 2006, Excerpt: "| <code>created_by_host_id</code> | UUIDv7 | MUST equal <code>issued_by_host_id</code> |", AcceptanceCases: []string{"core-record-identity-validation"}},
			{ID: "5.3#2", Line: 2036, Excerpt: "tie-break. A valid epoch greater than 1 MUST name a known predecessor for the", AcceptanceCases: []string{"lease-record-lifecycle", "lease-checkpoint-admission"}},
			{ID: "5.3#3", Line: 2037, Excerpt: "same session, MUST equal that predecessor's epoch plus one, and MUST reference", AcceptanceCases: []string{"lease-record-lifecycle", "lease-checkpoint-admission"}},
			{ID: "5.3#4", Line: 2039, Excerpt: "<code>create</code> lease MUST have a null predecessor and MAY have a null", AcceptanceCases: []string{"lease-record-lifecycle"}},
			{ID: "5.3#6", Line: 2046, Excerpt: "lease wins. Events under the losing same-epoch lease and all lower epochs MUST", AcceptanceCases: []string{"lease-divergent-preservation", "lease-union-resolution", "lease-ownership-union-properties", "story-260917-append-winning-lease-admission"}},
			{ID: "5.3#7", Line: 2047, Excerpt: "be preserved in a divergent branch and MUST NOT affect authoritative state.", AcceptanceCases: []string{"lease-divergent-preservation", "lease-union-resolution", "lease-ownership-union-properties", "story-260917-append-winning-lease-admission"}},
			{ID: "5.3#8", Line: 2049, Excerpt: "An owner process MUST revalidate its fencing token before:", AcceptanceCases: []string{"lease-fencing-revalidation", "lease-fencing-gates"}},
		},
	},
	"section:4.B": {
		Production: codeReference{Path: "internal/termbind/identity.go", Declaration: "CheckInstanceIdentity"},
		Cases:      []string{"terminal-instance-identity-exact"},
		Coverage:   coverageSliver,
		Gap:        "CheckInstanceIdentity enforces Section 4.B clause 4.B#11 (terminal instance identity is never a PID, handle, socket, path, pipe, URL, token, or endpoint) across the Binding, operation bodies, bootstrap binding, and attach surfaces; the Terminal Instance Binding 1.0.0 closed object itself is parsed by ParseTerminalBinding with ID recompute (terminal-instance-binding-exact covers the unmeasured table rules). The other 11 Section 4.B clauses (Manifest/Probe/evidence reconciliation rules, reader ID recompute, generation boundary cases) are implemented in internal/terminalbackend Reconcile/AdmitProbe and stay unenumerated here.",
		Clauses: []dischargedClause{
			{ID: "4.B#11", Line: 1076, Excerpt: "Identity MUST NOT be a PID, handle, socket, path, named pipe, URL, token, or", AcceptanceCases: []string{"terminal-instance-identity-exact"}},
		},
	},
	"section:4.D": {
		Production: codeReference{Path: "internal/termbind/resolve.go", Declaration: "ResolveEvidence"},
		Cases:      []string{"terminal-evidence-resolution-exact"},
		Coverage:   coverageSliver,
		Gap:        "ResolveEvidence discharges Section 4.D clause 4.D#2 by admitting every resolved Manifest, Probe, and Capability Evidence object through the landed Registry.AdmitProbe, which verifies the attestation signature before the object is treated as evidence; forged signatures refuse. The other Section 4.D clauses (registry-row equality 4.D#1 and the no-silent-fallback backend selection rule 4.D#3) are landed Reconcile behavior and explicit-selection behavior respectively and stay unenumerated here.",
		Clauses: []dischargedClause{
			{ID: "4.D#2", Line: 1296, Excerpt: "key registry and MUST verify the signature before treating the object as", AcceptanceCases: []string{"terminal-evidence-resolution-exact"}},
		},
	},
	"section:4.C": {
		Production: codeReference{Path: "internal/tmuxserver/lifecycle.go", Declaration: "Execute"},
		Cases:      []string{"tmux-attach-overlap-v070", "tmux-attach-semantics-v070", "tmux-lifecycle-operations-v070"},
		Coverage:   coveragePartial,
		Gap:        "Lifecycle.Execute discharges Section 4.C clauses 4.C#3, 4.C#4, 4.C#5, 4.C#6, and 4.C#7 for repeated identity, literal ax pane argv, host-only attach descriptors, AttachAuthorization equality, and listed refusal behavior. The creating uncertainty/status-first rule and per-side-effect winning-lease comparison remain outside this attach/lifecycle leaf's measured clauses.",
		Clauses: []dischargedClause{
			{ID: "4.C#3", Line: 1155, Excerpt: "Every repeated identity MUST equal the request context.", AcceptanceCases: []string{"tmux-attach-semantics-v070", "tmux-lifecycle-operations-v070"}},
			{ID: "4.C#4", Line: 1159, Excerpt: "<code>{argv:string[3]}</code>, and its three values MUST be literal", AcceptanceCases: []string{"tmux-lifecycle-operations-v070"}},
			{ID: "4.C#5", Line: 1162, Excerpt: "<code>string[1..4096]</code> usable only on the responding host and MUST NOT be", AcceptanceCases: []string{"tmux-attach-semantics-v070"}},
			{ID: "4.C#6", Line: 1189, Excerpt: "capability. In <code>attach</code>, the result input boolean MUST equal both the", AcceptanceCases: []string{"tmux-attach-overlap-v070", "tmux-attach-semantics-v070"}},
			{ID: "4.C#7", Line: 1195, Excerpt: "An error not listed for an operation MUST NOT be emitted by the backend for a", AcceptanceCases: []string{"tmux-attach-semantics-v070", "tmux-lifecycle-operations-v070"}},
		},
	},
	"section:5.2": {
		Production: codeReference{Path: "internal/termbind/resolve.go", Declaration: "ResolveEvidence"},
		Cases:      []string{"catalog-v050-exact", "core-record-identity-validation", "terminal-evidence-resolution-exact", "terminal-binding-events-exact"},
		Coverage:   coverageSliver,
		Gap:        "ResolveEvidence discharges Section 5.2 clauses 5.2#16 and 5.2#17 (evidence IDs resolve locally to validated Manifest/Probe/Evidence bound to the event tuple, never to a native reference or live-process fact); Section 5.2 clause 5.2#18 (v4 retained as inert immutable history, no derived state, no lower-version write) is pinned for Terminal Events by the EmitTerminalCreated emission path with sessrepo retention and the sessstate/sessquery readers (the retained-verbatim half, driven by TestV4RetainedAsImmutableHistory — the v1-only-reader half is a stated bound: no v1-v3-only reader exists on trunk). The other Section 5.2 envelope, table, authorship, ordering, and epoch clauses stay with the landed validateSessionEvent shape authority and sessrepo ordering, unenumerated here.",
		Clauses: []dischargedClause{
			{ID: "5.2#16", Line: 1971, Excerpt: "replicated. The evidence IDs MUST resolve locally to validated, sanitized", AcceptanceCases: []string{"terminal-evidence-resolution-exact"}},
			{ID: "5.2#17", Line: 1973, Excerpt: "backend ID and versions. They MUST NOT resolve through the event to a native", AcceptanceCases: []string{"terminal-evidence-resolution-exact"}},
			{ID: "5.2#18", Line: 1984, Excerpt: "as inert immutable history but MUST NOT derive runtime state or write a lower-", AcceptanceCases: []string{"terminal-binding-events-exact"}},
		},
	},
	"section:7.A": {
		Production: codeReference{Path: "internal/terminalbackend/descriptor.go", Declaration: "AdmitProviderDescriptor"},
		Cases:      []string{"terminal-provider-descriptor-7a", "terminal-instance-identity-exact"},
		Coverage:   coveragePartial,
		Gap:        "AdmitProviderDescriptor discharges Section 7.A clause 7.A#1 (the Provider rejects a descriptor whose binding digest, backend, version, or generation does not match the AX-validated binding), pinned by the landed terminal-provider-descriptor-7a mismatch tests and the terminal-instance-identity-exact eight-form identity pin through the same entry. Section 7.A clause 7.A#2 (LeaseToken v2 fencing rules and the 3.x transport envelope) stays with provhost v2 machinery and the dual-stack follow-up, undisclosed here.",
		Clauses: []dischargedClause{
			{ID: "7.A#1", Line: 3724, Excerpt: "TerminalBackend. The Provider MUST reject a descriptor whose binding digest,", AcceptanceCases: []string{"terminal-provider-descriptor-7a", "terminal-instance-identity-exact"}},
		},
	},
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
	"section:7.8": {
		Production: codeReference{Path: "internal/sessadapter/protocol.go", Declaration: "DecodeRequestFrame"},
		Cases:      []string{"session-adapter-frame-bound", "session-adapter-call-binding"},
		Coverage:   coverageFull,
		Gap:        "",
		Clauses: []dischargedClause{
			{ID: "7.8#1", Line: 3785, Excerpt: "Large data is referenced by manifest or Blob Descriptor ID and MUST NOT be embedded in the 8 MiB frame.", AcceptanceCases: []string{"session-adapter-frame-bound"}},
			{ID: "7.8#2", Line: 3912, Excerpt: "target mutation, these facts MUST equal freshly read trusted-candidate facts and the Journal binding.", AcceptanceCases: []string{"session-adapter-call-binding"}},
		},
	},
	"section:10.2": {
		Production: codeReference{Path: "internal/clonesnap/capture.go", Declaration: "Capture"},
		Cases:      []string{"scalar-digest-validation", "scalar-bounded-integer-validation", "canonical-jcs-rfc8785", "canonical-object-identity", "canonical-identity-refusal", "localstore-digest-path-v1", "localstore-immutable-blob-install", "localstore-sqlite-projection", "clone-capture-contracts", "clone-native-capture", "clone-projection-fidelity"},
		Coverage:   coverageFull,
		Gap:        "",
		Clauses: []dischargedClause{
			{ID: "10.2#1", Line: 4854, Excerpt: "MUST live in a manifest, not in the blob path.", AcceptanceCases: []string{"clone-capture-contracts"}},
			{ID: "10.2#2", Line: 4855, Excerpt: "references a blob, the writer MUST fsync the blob, verify its size and digest, and atomically install it in the object store.", AcceptanceCases: []string{"clone-native-capture", "clone-projection-fidelity", "localstore-immutable-blob-install"}},
			{ID: "10.2#3", Line: 4859, Excerpt: "metrics MUST record digest, size, and media type only, never blob contents.", AcceptanceCases: []string{"clone-native-capture"}},
			{ID: "10.2#4", Line: 4862, Excerpt: "<code>urn:ax:schema:blob</code> version <code>1.0.0</code>. Chunks MUST be sorted, contiguous, non-overlapping, start at offset zero, and cover exactly <code>size</code> bytes:", AcceptanceCases: []string{"clone-native-capture"}},
			{ID: "10.2#5", Line: 4902, Excerpt: "A larger file MUST fail capture with <code>capability_unavailable</code> before publishing a partial manifest.", AcceptanceCases: []string{"clone-native-capture"}},
		},
	},
}

// storyBindingExtensions are the two already-measured bindings that this
// Story extends with executed admission/source cases. Unlike the historical
// post-adoption upgrades above, these do not promote an unevidenced stub; the
// extension is limited to adding the new Story's acceptance owners to clauses
// already discharged by the carried binding.
var storyBindingExtensions = map[string]struct{}{
	"section:2.4": {},
	"section:5.3": {},
}

// storyNewV070Cases is the exact set of acceptance cases the post-adoption
// story final leaves add to the adopted registry. Each is pinned literally;
// any further addition still fails as an unreviewed claim.
var storyNewV070Cases = map[string]acceptanceCase{
	"tmux-attach-overlap-v070": {
		ID:         "tmux-attach-overlap-v070",
		Production: codeReference{Path: "internal/tmuxserver/lifecycle.go", Declaration: "Execute"},
		Tests: []codeReference{
			{Path: "internal/tmuxserver/lifecycle_active_clients_unix_test.go", Declaration: "TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances"},
			{Path: "internal/tmuxserver/lifecycle_active_clients_unix_test.go", Declaration: "TestAttachOverlapInputGateChecksEveryPeer"},
			{Path: "internal/tmuxserver/lifecycle_active_clients_unix_test.go", Declaration: "TestAttachConcurrentInputRequiresAXPolicy"},
			{Path: "internal/tmuxserver/lifecycle_active_clients_unix_test.go", Declaration: "TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer"},
			{Path: "internal/tmuxserver/lifecycle_rev8_unix_test.go", Declaration: "TestAttachOverlapRequiresMultiAttach"},
			{Path: "internal/tmuxserver/lifecycle_rev8_unix_test.go", Declaration: "TestAttachOverlapInputRequiresMultipleInputClients"},
			{Path: "internal/tmuxserver/lifecycle_rev8_unix_test.go", Declaration: "TestAttachSameClientRetryIgnoresOverlap"},
			{Path: "internal/tmuxserver/lifecycle_rev9_unix_test.go", Declaration: "TestAttachSameClientRetryWithPeerPresent"},
		},
	},
	"tmux-attach-semantics-v070": {
		ID:         "tmux-attach-semantics-v070",
		Production: codeReference{Path: "internal/tmuxserver/lifecycle.go", Declaration: "Execute"},
		Tests: []codeReference{
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestExecuteAttachLostResponseReplaysRecordedOutcome"},
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable"},
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestExecuteAttachDoesNotReadOrChangeSessionLease"},
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestForegroundAcquireComposesProductionProbeWithAttach"},
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch"},
			{Path: "internal/tmuxserver/lifecycle_extra_unix_test.go", Declaration: "TestAttachDescriptorNeverPersisted"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteAttachIdempotent"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteAttachRefusals"},
		},
	},
	"tmux-lifecycle-operations-v070": {
		ID:         "tmux-lifecycle-operations-v070",
		Production: codeReference{Path: "internal/tmuxserver/lifecycle.go", Declaration: "Execute"},
		Tests: []codeReference{
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteCreateInteractive"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteCreateHeadless"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteCreateRefusals"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteAttach"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteAttachIdempotent"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteStatusPresent"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteQuiesce"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteBoundary"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteStop"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteTerminate"},
			{Path: "internal/tmuxserver/lifecycle_ops_unix_test.go", Declaration: "TestExecuteRestore"},
			{Path: "internal/tmuxserver/wrapper_unix_test.go", Declaration: "TestWrapperRestoreLocalWinResumes"},
			{Path: "internal/tmuxserver/wrapper_unix_test.go", Declaration: "TestWrapperRestoreLapsedGrantRemoteOffer"},
			{Path: "internal/tmuxserver/wrapper_unix_test.go", Declaration: "TestWrapperRestoreParksWithoutEffects"},
			{Path: "internal/tmuxserver/wrapper_unix_test.go", Declaration: "TestWrapperRestoreExpiredRefreshParksUnverified"},
		},
	},
	"tmux-private-server-management-v070": {
		ID:         "tmux-private-server-management-v070",
		Production: codeReference{Path: "internal/tmuxserver/acquire.go", Declaration: "Acquire"},
		Tests: []codeReference{
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireForegroundSpawnsDedicatedServer"},
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireForegroundAttachesToRunningAttested"},
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireForegroundRefusesWrongGenerationAdmission"},
			{Path: "internal/tmuxserver/p3a_test.go", Declaration: "TestAcquireForegroundRefusesEveryCatalogDecoyRunning"},
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireBackgroundContactsBroker"},
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireBackgroundMissReturnsTypedUnavailable"},
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireBackgroundVerifiesWithoutCreating"},
			{Path: "internal/tmuxserver/acquire_test.go", Declaration: "TestAcquireBackgroundRefusesEveryCatalogDecoy"},
		},
	},
	"tmux-socket-custody-v070": {
		ID:         "tmux-socket-custody-v070",
		Production: codeReference{Path: "internal/tmuxserver/bind.go", Declaration: "CheckSocketCustody"},
		Tests: []codeReference{
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch"},
			{Path: "internal/tmuxserver/lifecycle_attach_properties_unix_test.go", Declaration: "TestExecuteEveryOperationRefusesForeignOrPermissiveSocket"},
			{Path: "internal/tmuxserver/probe_unix_test.go", Declaration: "TestServerProberRefusesSocketSubstitutionBeforeConnect"},
			{Path: "internal/tmuxserver/probe_unix_test.go", Declaration: "TestServerProberRefusesSocketSymlinkBeforeConnect"},
			{Path: "internal/tmuxserver/probe_unix_test.go", Declaration: "TestServerProberRefusesForeignSocketBeforeConnect"},
			{Path: "internal/tmuxserver/probe_unix_test.go", Declaration: "TestServerProberRefusesPermissiveSocketBeforeConnect"},
			{Path: "internal/tmuxserver/probe_unix_test.go", Declaration: "TestServerSpawnerRefusesSocketSubstitutionBeforeUnlinkOrSpawn"},
			{Path: "internal/tmuxserver/probe_unix_test.go", Declaration: "TestServerSpawnerRefusesForeignOrPermissiveSocketBeforeMutation"},
		},
	},
	"story-260917-append-winning-lease-admission": {
		ID:         "story-260917-append-winning-lease-admission",
		Production: codeReference{Path: "internal/sessrepo/sessrepo.go", Declaration: "AppendEvent"},
		Tests: []codeReference{
			{Path: "internal/sessrepo/sessrepo_test.go", Declaration: "TestAppendEventRefusesSupersededLeaseWhileTailStillMatches"},
			{Path: "internal/sessrepo/sessrepo_test.go", Declaration: "TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches"},
			{Path: "internal/sessprofile/setprofile_test.go", Declaration: "TestSetProfileRefusesSupersededLeaseWhileTailStillMatches"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestEmitReachesAppendAdmissionGateAfterStaleObservation"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestEmitReachesSameEpochAppendAdmissionGate"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestEmitParkedReachesSameEpochAppendAdmissionGate"},
		},
	},
	"story-260917-losing-lease-profile-source": {
		ID:         "story-260917-losing-lease-profile-source",
		Production: codeReference{Path: "internal/sessrepo/sessrepo.go", Declaration: "AppendEvent"},
		Tests: []codeReference{
			{Path: "internal/axpane/rework_test.go", Declaration: "TestAppendGateRefusesLosingLeaseProfileEvent"},
		},
	},
	"story-260922-derivation-side-profile-source": {
		ID:         "story-260922-derivation-side-profile-source",
		Production: codeReference{Path: "internal/sessprofile/authority.go", Declaration: "Derive"},
		Tests: []codeReference{
			{Path: "internal/axpane/rework_test.go", Declaration: "TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestProbe15DisabledAppendGateProjector"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestSetProfileFromEndDoesNotReplayLosingLeaseChange"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple"},
			{Path: "internal/sessprofile/authority_test.go", Declaration: "TestProjectorRejectsNeverMintedHigherEpochProfileSource"},
			{Path: "internal/sessprofile/authority_test.go", Declaration: "TestDerivationForHeadsRejectsUnmintedHigherEpochSource"},
			{Path: "internal/sessprofile/authority_test.go", Declaration: "TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource"},
			{Path: "internal/sessprofile/authority_test.go", Declaration: "TestProjectorEmptyLeaseStoreKeepsRecordAuthority"},
			{Path: "internal/axpane/rework_test.go", Declaration: "TestLoadProfileEmptyLeaseStoreKeepsSessionRecordAuthority"},
			{Path: "internal/sessprofile/authority_test.go", Declaration: "TestProjectorKeepsPriorLeaseSourceFromWinningHandoffClosure"},
			{Path: "internal/sessprofile/authority_test.go", Declaration: "TestProjectorMissingWinningHandoffStoreIsNotTreatedAsEmptyClosure"},
		},
	},
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
	"terminal-instance-binding-exact": {
		ID:         "terminal-instance-binding-exact",
		Production: codeReference{Path: "internal/termbind/identity.go", Declaration: "ParseTerminalBinding"},
		Tests: []codeReference{
			{Path: "internal/termbind/identity_test.go", Declaration: "TestParseTerminalBindingAdmitsClosedFixture"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestParseTerminalBindingClosedShape"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestBindingIdentityAgreesWithLandedAdmission"},
			{Path: "internal/termbind/binding_arms_test.go", Declaration: "TestParseTerminalBindingMemberArms"},
			{Path: "internal/termbind/binding_arms_test.go", Declaration: "TestParseTerminalBindingRawExtensions"},
			{Path: "internal/termbind/binding_arms_test.go", Declaration: "TestBindingIdentityRefusesHostileBytes"},
			{Path: "internal/termbind/binding_arms_test.go", Declaration: "TestBindingIdentityVerdictAgreesWithLandedAdmission"},
			{Path: "internal/termbind/binding_arms_test.go", Declaration: "TestBindingProtocolMajorAgreesWithLandedTuple"},
		},
	},
	"terminal-instance-identity-exact": {
		ID:         "terminal-instance-identity-exact",
		Production: codeReference{Path: "internal/termbind/identity.go", Declaration: "CheckInstanceIdentity"},
		Tests: []codeReference{
			{Path: "internal/termbind/identity_test.go", Declaration: "TestCheckInstanceIdentityAdmitsUUIDv7"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestCheckInstanceIdentityRefusesForbiddenForms"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestParseTerminalBindingRefusesForbiddenIdentity"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestAdmitMutationContextRefusesForbiddenIdentity"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestAdmitStatusBodyRefusesForbiddenIdentity"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachRefusesForbiddenIdentity"},
			{Path: "internal/termbind/identity_test.go", Declaration: "TestAdmitDescriptorRefusesForbiddenIdentity"},
		},
	},
	"terminal-evidence-resolution-exact": {
		ID:         "terminal-evidence-resolution-exact",
		Production: codeReference{Path: "internal/termbind/resolve.go", Declaration: "ResolveEvidence"},
		Tests: []codeReference{
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceAdmitsUniverse"},
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceShapeBounds"},
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceRefusesForeignKinds"},
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceRequiresManifestAndProbe"},
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceBindsEventTuple"},
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceRejectsSubstitution"},
			{Path: "internal/termbind/resolve_test.go", Declaration: "TestResolveEvidenceDrivesLandedAdmission"},
		},
	},
	"terminal-binding-events-exact": {
		ID:         "terminal-binding-events-exact",
		Production: codeReference{Path: "internal/termbind/emit.go", Declaration: "EmitTerminalCreated"},
		Tests: []codeReference{
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitTerminalCreatedRoundTrip"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitClosedShapeRefusals"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitResolvesEvidence"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitUnderLosingLeaseRefuses"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitAppendsIdempotently"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmittedEventCarriesNoBindingObject"},
			{Path: "internal/termbind/skew_test.go", Declaration: "TestTakeoverSelectsNewBackendNewEvent"},
			{Path: "internal/termbind/skew_test.go", Declaration: "TestV4RetainedAsImmutableHistory"},
			{Path: "internal/termbind/crash_unix_test.go", Declaration: "TestEmitAppendHookSeams"},
		},
	},
	"terminal-binding-resumed-exact": {
		ID:         "terminal-binding-resumed-exact",
		Production: codeReference{Path: "internal/termbind/emit.go", Declaration: "EmitResumed"},
		Tests: []codeReference{
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitResumedRoundTrip"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitResolvesEvidence"},
			{Path: "internal/termbind/emit_test.go", Declaration: "TestEmitResumedCheckpointBindingIsCallerBound"},
		},
	},
	"terminal-create-recovery-exact": {
		ID:         "terminal-create-recovery-exact",
		Production: codeReference{Path: "internal/termbind/recover.go", Declaration: "RecoverCreate"},
		Tests: []codeReference{
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateProvesAbsence"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateIdentifiesOneChild"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateDuplicateRetrySameVerdict"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateChangedOperationInWindowRefuses"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateChangedOperationPostWindowProceeds"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateLostResult"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateNeverClaimsAbsent"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateRecordedChildBackendAbsentReplays"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverCreateUnknownIsNotAbsent"},
			{Path: "internal/termbind/crash_unix_test.go", Declaration: "TestRecoverCreateAfterKillRecoversChild"},
			{Path: "internal/termbind/recover_test.go", Declaration: "TestRecoverGenerationBoundBeforeStatusRead"},
		},
	},
	"terminal-attach-receipt-exact": {
		ID:         "terminal-attach-receipt-exact",
		Production: codeReference{Path: "internal/termbind/attach.go", Declaration: "Attach"},
		Tests: []codeReference{
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachRoundTrip"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachIdenticalRetryReplays"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachSecondClientRecordsAlongside"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachConflictRefusesMismatch"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachRefusesForbiddenIdentity"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachTransportVocabulary"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachRequiresLiveAuth"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachEmitsNeitherEventNorStateChange"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachUnknownIsNotAbsent"},
			{Path: "internal/termbind/attach_test.go", Declaration: "TestAttachSessionMismatchRefuses"},
			{Path: "internal/termbind/crash_unix_test.go", Declaration: "TestAttachCrashChildSelfTerminates"},
		},
	},
	"clone-capture-contracts": {
		ID:         "clone-capture-contracts",
		Production: codeReference{Path: "internal/clonebundle/rawmanifest.go", Declaration: "BuildRawObjectManifest"},
		Tests: []codeReference{
			{Path: "internal/clonebundle/contract_test.go", Declaration: "TestBuildRawManifestRoundTrip"},
			{Path: "internal/clonebundle/contract_test.go", Declaration: "TestBuildCaptureManifestDerivesRawComplete"},
			{Path: "internal/clonebundle/contract_test.go", Declaration: "TestBuildCanonicalEventRoundTrip"},
			{Path: "internal/clonebundle/refusal_test.go", Declaration: "TestRawManifestRefusals"},
			{Path: "internal/clonebundle/refusal_test.go", Declaration: "TestCaptureManifestRefusals"},
		},
	},
	"clone-native-capture": {
		ID:         "clone-native-capture",
		Production: codeReference{Path: "internal/clonesnap/capture.go", Declaration: "Capture"},
		Tests: []codeReference{
			{Path: "internal/clonesnap/capture_test.go", Declaration: "TestCaptureSealsBothManifestsFromStoreBytes"},
			{Path: "internal/clonesnap/capture_test.go", Declaration: "TestCaptureInstallsEveryPayloadBlob"},
			{Path: "internal/clonesnap/capture_test.go", Declaration: "TestCaptureMultiChunkMember"},
			{Path: "internal/clonesnap/exclusion_test.go", Declaration: "TestCaptureLogRecordsDigestsOnly"},
			{Path: "internal/clonesnap/refusal_test.go", Declaration: "TestCaptureRefusesOversizedMember"},
		},
	},
	"clone-projection-fidelity": {
		ID:         "clone-projection-fidelity",
		Production: codeReference{Path: "internal/cloneproject/normalize.go", Declaration: "Normalize"},
		Tests: []codeReference{
			{Path: "internal/cloneproject/normalize_test.go", Declaration: "TestNormalizeUnknownRecordBecomesOpaqueEvent"},
			{Path: "internal/cloneproject/normalize_test.go", Declaration: "TestNormalizeInlineContentBoundary"},
			{Path: "internal/cloneproject/foreign_test.go", Declaration: "TestNormalizeForeignEncryptedReasoningStaysOpaque"},
			{Path: "internal/cloneproject/tools_test.go", Declaration: "TestNormalizeToolMatrix"},
			{Path: "internal/cloneproject/fidelity_test.go", Declaration: "TestNormalizeIsIdempotentIntoSharedSink"},
		},
	},
	"session-adapter-call-binding": {
		ID:         "session-adapter-call-binding",
		Production: codeReference{Path: "internal/sessadapter/discovery.go", Declaration: "CheckCallBinding"},
		Tests: []codeReference{
			{Path: "internal/sessadapter/discovery_test.go", Declaration: "TestCheckCallBinding"},
			{Path: "internal/sessadapter/discovery_test.go", Declaration: "TestCheckCallBindingZeroFactsRefuse"},
		},
	},
	"session-adapter-frame-bound": {
		ID:         "session-adapter-frame-bound",
		Production: codeReference{Path: "internal/sessadapter/protocol.go", Declaration: "DecodeRequestFrame"},
		Tests: []codeReference{
			{Path: "internal/sessadapter/envelope_test.go", Declaration: "TestDecodeRequestFrameRefusesOversizeFrame"},
			{Path: "internal/sessadapter/envelope_test.go", Declaration: "TestFrameBoundEdges"},
		},
	},
}

// assertStoryBindingUpgrade pins one reviewed post-adoption upgrade. The
// historical upgrades promote unevidenced stubs; the explicitly listed Story
// extensions retain their measured base and only add reviewed acceptance
// owners to already discharged clauses. Every resulting field is still exact.
func assertStoryBindingUpgrade(t *testing.T, key string, old, now ownershipGroup, upgrade storyBindingUpgrade) {
	t.Helper()

	if _, extension := storyBindingExtensions[key]; !extension && (old.Coverage != coverageUnevidenced || len(old.Clauses) != 0) {
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

	if _, extension := storyBindingExtensions[key]; !extension && len(old.Clauses) != 0 {
		t.Fatalf("binding %q upgrade base carries %d clauses, want the unevidenced stub", key, len(old.Clauses))
	}
	for _, newClause := range now.Clauses {
		if measured070[key][newClause.ID] != newClause.Line {
			t.Fatalf("binding %q clause %s v0.7.0 line %d is not the measured line %d",
				key, newClause.ID, newClause.Line, measured070[key][newClause.ID])
		}
	}
}
