// Package traceability verifies that the pinned AX specification inventory
// remains bound to concrete production declarations and executable acceptance
// cases. It is a read-only repository gate; it does not advertise runtime
// capabilities or mutate implementation state.
package traceability

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/cataloggen"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
	"github.com/relux-works/agent-session-manager/internal/specpin"
)

const (
	ownershipRegistryPath = "internal/traceability/ownership.v0.5.0.json"
	contractLockPath      = "internal/specpin/v0.5.0.lock.json"
	catalogMetadataPath   = "internal/catalog/catalog.v0.5.0.json"
	generatedCatalogPath  = "internal/catalog/catalog_gen.go"

	ownershipFormat        = "ax-spec-to-code-ownership"
	ownershipFormatVersion = 1

	// reviewedOwnershipCanonicalSHA256 pins the semantic JSON projection. JSON
	// formatting may change, but ownership claims cannot be self-minted without
	// an explicit review of this binding.
	reviewedOwnershipCanonicalSHA256 = "8281d96bc0d30a7f9ad6f7db89eef7fef9d45d203efe5e6a4645032e8af5ea17"
)

var ErrTraceability = errors.New("spec-to-code traceability check failed")

// Report contains inventory coverage only. These counts do not claim that any
// runtime capability is available, enabled, or supported.
type Report struct {
	Contracts              int
	NormativeSections      int
	AcceptanceCases        int
	Fixtures               int
	CompatibilityContracts int
	AssignedScopes         int

	// SectionBindings and the coverage counters below are a measured ratio over
	// the registered section bindings, not a prose claim. NormativeClauses is
	// the number of RFC 2119 clause lines the pinned document itself carries in
	// the bound sections; DischargedClauses is how many of those a binding
	// enumerates with a verified excerpt and an acceptance owner.
	SectionBindings     int
	FullCoverage        int
	PartialCoverage     int
	SliverCoverage      int
	UnevidencedCoverage int
	UnmeasuredCoverage  int
	UnownedSections     int
	NormativeClauses    int
	DischargedClauses   int
}

type ownershipKind string

const (
	ownershipContract         ownershipKind = "contract"
	ownershipNormativeSection ownershipKind = "normative_section"
	ownershipSectionBinding   ownershipKind = "section_binding"
	ownershipFixture          ownershipKind = "fixture"
)

type ownershipRegistry struct {
	Format          string           `json:"format"`
	FormatVersion   int              `json:"format_version"`
	Source          registrySource   `json:"source"`
	AcceptanceCases []acceptanceCase `json:"acceptance_cases"`
	Ownership       []ownershipGroup `json:"ownership"`
	UnownedSections []unownedSection `json:"unowned_sections"`
}

// unownedSection records a real v0.5.0 section that this repository does not
// implement. It exists so an unimplemented section is disclosed by name rather
// than bound to a symbol from a neighbouring package, which is what an owned
// key would otherwise assert. It is a disclosure, never an exemption: a section
// the generated catalog requires an owner for may not be declared here, and
// assigned-scope admission refuses an unowned section and prints its gap.
type unownedSection struct {
	Key      string `json:"key"`
	Gap      string `json:"gap"`
	Evidence string `json:"evidence"`
}

type registrySource struct {
	Repository     string `json:"repository"`
	Release        string `json:"release"`
	Commit         string `json:"commit"`
	DocumentPath   string `json:"document_path"`
	DocumentSHA256 string `json:"document_sha256"`
}

type acceptanceCase struct {
	ID         string          `json:"id"`
	Production codeReference   `json:"production"`
	Tests      []codeReference `json:"tests"`
}

type ownershipGroup struct {
	Kind            ownershipKind      `json:"kind"`
	Keys            []string           `json:"keys"`
	Production      codeReference      `json:"production"`
	AcceptanceCases []string           `json:"acceptance_cases"`
	Coverage        coverageLevel      `json:"coverage,omitempty"`
	Gap             string             `json:"gap,omitempty"`
	Clauses         []dischargedClause `json:"clauses,omitempty"`
}

// coverageLevel is how much of a bound section a binding claims to discharge.
// The claim is not taken on trust: the level is recomputed from the clauses the
// binding enumerates against the clause inventory the pinned specification
// itself carries for that section, and a claim that differs from the measured
// level is refused.
type coverageLevel string

const (
	// coverageFull means every normative clause the pinned section carries is
	// enumerated with a verified excerpt and an acceptance owner.
	coverageFull coverageLevel = "full"
	// coveragePartial means at least half of them are.
	coveragePartial coverageLevel = "partial"
	// coverageSliver means fewer than half are: the binding implements a corner
	// of the section it is registered against.
	coverageSliver coverageLevel = "sliver"
	// coverageUnevidenced means the binding enumerates no clause at all. It says
	// the registry makes no clause-level claim, not that the implementation
	// covers nothing.
	coverageUnevidenced coverageLevel = "unevidenced"
	// coverageUnmeasured means normativeKeywordPattern finds no clause line
	// under the section's heading or its subheadings, so the gate cannot
	// measure the section's obligations at all.
	//
	// It is deliberately NOT called "declarative", and it does NOT mean the
	// section carries no obligation. Nineteen of the 157 pinned headings are in
	// this class and every one of them has a substantive body: Section 15.2 is a
	// eighteen-row normative exit-code registry and Section 7.3 is the closed
	// Provider Manifest, both of which state their obligations as tables rather
	// than in uppercase RFC 2119 keywords. Treating that silence as "nothing to
	// discharge" is the same free pass this gate exists to remove, reached
	// through the scanner instead of through the label, so an unmeasured binding
	// carries the same mandatory gap as every other level below full and is
	// refused for assigned-scope admission.
	coverageUnmeasured coverageLevel = "unmeasured"
)

// dischargedClause binds one normative clause of the pinned section to the
// acceptance cases that discharge it. Every field is checked against the
// hash-verified document: the identifier must index the section's own clause
// inventory, the line must be the line that clause occupies, and the excerpt
// must be that clause's text quoted verbatim, so a clause cannot be claimed
// with a true sentence about something else.
type dischargedClause struct {
	ID              string   `json:"id"`
	Line            int      `json:"line"`
	Excerpt         string   `json:"excerpt"`
	AcceptanceCases []string `json:"acceptance_cases"`
}

// normativeClause is one RFC 2119 clause line of the pinned specification,
// measured from the document rather than declared by the registry.
type normativeClause struct {
	ID   string
	Line int
	Text string
}

type codeReference struct {
	Path        string `json:"path"`
	Declaration string `json:"declaration"`
}

type assignedSectionBinding struct {
	Scope string
	Keys  []string
}

var assignedSectionPattern = regexp.MustCompile(`^((?:20|1[0-9]|[1-9])(?:\.(?:[0-9]+|[A-Za-z]))*)(?:-((?:20|1[0-9]|[1-9])(?:\.(?:[0-9]+|[A-Za-z]))*))?$`)
var catalogAppendixPattern = regexp.MustCompile(`(?i)^Appendix ([A-D]):`)

// VerifyRepository is the production CI entry point. It verifies exact pinned
// inputs, stale generated output, compatibility coverage, and every registered
// ownership reference against the supplied repository filesystem.
func VerifyRepository(repository fs.FS) (Report, error) {
	return verifyRepository(repository, nil)
}

// VerifyAssignedSections is the production Story-scope entry point. Each exact
// assigned subsection, or every heading in a same-section range, must have its
// own scoped production declaration and executable acceptance-case link. A
// pinned top-level source owner alone never satisfies assigned-scope admission.
func VerifyAssignedSections(repository fs.FS, sections []string) (Report, error) {
	if len(sections) == 0 {
		return Report{}, fail("assigned section scope is empty")
	}
	return verifyRepository(repository, sections)
}

func verifyRepository(repository fs.FS, assignedSections []string) (Report, error) {
	if repository == nil {
		return Report{}, fail("repository filesystem is nil")
	}

	lockBytes, err := readRequired(repository, contractLockPath)
	if err != nil {
		return Report{}, err
	}
	manifest, err := specpin.Verify(lockBytes)
	if err != nil {
		return Report{}, fail("verify normative source lock: %v", err)
	}

	metadataBytes, err := readRequired(repository, catalogMetadataPath)
	if err != nil {
		return Report{}, err
	}
	generated, err := cataloggen.Generate(metadataBytes, lockBytes)
	if err != nil {
		return Report{}, fail("verify reviewed catalog metadata: %v", err)
	}
	committedGenerated, err := readRequired(repository, generatedCatalogPath)
	if err != nil {
		return Report{}, err
	}
	if !bytes.Equal(generated, committedGenerated) {
		return Report{}, fail("generated catalog is stale; run go generate ./internal/catalog")
	}

	currentCatalog, err := catalog.ForRelease(catalog.ReleaseV050)
	if err != nil {
		return Report{}, fail("load current catalog: %v", err)
	}
	if err := verifyCatalogContracts(manifest.Contracts, currentCatalog.Contracts); err != nil {
		return Report{}, err
	}
	compatibilityCatalog, err := catalog.ForRelease(catalog.ReleaseV043)
	if err != nil {
		return Report{}, fail("load compatibility catalog: %v", err)
	}

	registryBytes, err := readRequired(repository, ownershipRegistryPath)
	if err != nil {
		return Report{}, err
	}
	registry, err := decodeOwnershipRegistry(registryBytes)
	if err != nil {
		return Report{}, err
	}
	bindings, err := resolveAssignedSections(manifest.Source.NormativeScope, assignedSections)
	if err != nil {
		return Report{}, err
	}
	document, err := specdoc.Load()
	if err != nil {
		return Report{}, fail("load pinned specification document: %v", err)
	}
	if manifest.Source.Document.SHA256 != specpin.DocumentSHA256 {
		return Report{}, fail(
			"verified normative lock document digest %s differs from the pinned clause source %s",
			manifest.Source.Document.SHA256, specpin.DocumentSHA256)
	}
	return verifyOwnership(repository, document, registry, manifest, currentCatalog, compatibilityCatalog, bindings)
}

func readRequired(repository fs.FS, filename string) ([]byte, error) {
	value, err := fs.ReadFile(repository, filename)
	if err != nil {
		return nil, fail("read required evidence %q: %v", filename, err)
	}
	return value, nil
}

func decodeOwnershipRegistry(candidate []byte) (ownershipRegistry, error) {
	var decoded ownershipRegistry
	decoder := json.NewDecoder(bytes.NewReader(candidate))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return ownershipRegistry{}, fail("decode ownership registry: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return ownershipRegistry{}, fail("decode ownership registry: multiple JSON values")
		}
		return ownershipRegistry{}, fail("decode ownership registry trailing data: %v", err)
	}
	return decoded, nil
}

func verifyOwnership(
	repository fs.FS,
	document *specdoc.Document,
	registry ownershipRegistry,
	manifest specpin.Manifest,
	currentCatalog catalog.Catalog,
	compatibilityCatalog catalog.Catalog,
	assignedBindings []assignedSectionBinding,
) (Report, error) {
	if registry.Format != ownershipFormat || registry.FormatVersion != ownershipFormatVersion {
		return Report{}, fail("unsupported ownership registry format %q version %d", registry.Format, registry.FormatVersion)
	}
	source := registry.Source
	if source.Repository != manifest.Source.Repository ||
		source.Release != manifest.Source.Release ||
		source.Commit != manifest.Source.Commit ||
		source.DocumentPath != manifest.Source.Document.Path ||
		source.DocumentSHA256 != manifest.Source.Document.SHA256 {
		return Report{}, fail("ownership registry source differs from verified normative lock")
	}

	expected := map[ownershipKind]map[string]struct{}{
		ownershipContract:         expectedContracts(manifest.Contracts),
		ownershipNormativeSection: expectedNormativeSections(manifest, currentCatalog),
		ownershipSectionBinding:   expectedSectionBindingInventory(),
		ownershipFixture:          expectedFixtures(manifest, currentCatalog),
	}
	requiredSectionBindings, err := expectedCatalogSectionBindings(currentCatalog)
	if err != nil {
		return Report{}, err
	}
	required := map[ownershipKind]map[string]struct{}{
		ownershipContract:         expected[ownershipContract],
		ownershipNormativeSection: expected[ownershipNormativeSection],
		ownershipSectionBinding:   requiredSectionBindings,
		ownershipFixture:          expected[ownershipFixture],
	}
	selectedSectionBindings := map[string]struct{}(nil)
	if assignedBindings != nil {
		required = cloneOwnershipSets(expected)
		selectedSectionBindings = make(map[string]struct{})
		for _, binding := range assignedBindings {
			for _, key := range binding.Keys {
				selectedSectionBindings[key] = struct{}{}
			}
		}
		required[ownershipSectionBinding] = make(map[string]struct{})
	}
	checker := newSourceChecker(repository)
	acceptanceIDs, err := verifyAcceptanceCases(checker, registry.AcceptanceCases)
	if err != nil {
		return Report{}, err
	}
	// Coverage is measured over every registered section binding, not only the
	// assigned ones, so a sliver planted anywhere in the registry reddens both
	// production entry points rather than only the repository-wide one. It is
	// measured before ownership resolution so that a section moved to the
	// unowned disclosure to dodge the catalog's owner requirement is reported
	// as that dodge rather than as an ordinary missing owner.
	coverage, err := verifySectionCoverage(document, registry.Ownership, acceptanceIDs)
	if err != nil {
		return Report{}, err
	}
	ownedSections := make(map[string]struct{}, len(coverage))
	for key := range coverage {
		ownedSections[key] = struct{}{}
	}
	unowned, err := verifyUnownedSections(
		registry.UnownedSections, expected[ownershipSectionBinding], requiredSectionBindings, ownedSections)
	if err != nil {
		return Report{}, err
	}

	owned, err := verifyOwnershipGroups(checker, registry.Ownership, acceptanceIDs, expected, required, selectedSectionBindings)
	if err != nil {
		return Report{}, err
	}

	for _, binding := range assignedBindings {
		for _, key := range binding.Keys {
			if entry, disclosed := unowned[key]; disclosed {
				return Report{}, fail(
					"assigned section %q binding %q is recorded unowned: %s", binding.Scope, key, entry.Gap)
			}
			if _, ok := owned[ownershipSectionBinding][key]; !ok {
				return Report{}, fail("assigned section %q binding %q has no scoped implementation owner", binding.Scope, key)
			}
			// Admission requires full and nothing else. coverageUnmeasured is
			// deliberately not admitted: the gate cannot see the section's
			// obligations, and admitting on an unverifiable justification
			// sentence would be self-minted evidence for exactly the class this
			// gate exists to refuse. Absence of a scanner-visible clause is a
			// failure to measure, not a measured absence of obligation.
			measured := coverage[key]
			if measured.Level != coverageFull {
				return Report{}, fail(
					"assigned section %q binding %q discharges %s normative clauses, which is %s coverage; assigned-scope admission requires full: %s",
					binding.Scope, key, measured.Ratio(), measured.Level, measured.Gap)
			}
		}
	}
	for _, contract := range compatibilityCatalog.Contracts {
		key := contractKey(contract.Name, string(contract.ID))
		if _, ok := owned[ownershipContract][key]; !ok {
			return Report{}, fail("compatibility contract %q has no implementation owner", key)
		}
	}

	canonical, err := json.Marshal(registry)
	if err != nil {
		return Report{}, fail("encode canonical ownership registry: %v", err)
	}
	digest := sha256.Sum256(canonical)
	gotDigest := hex.EncodeToString(digest[:])
	if gotDigest != reviewedOwnershipCanonicalSHA256 {
		return Report{}, fail("ownership registry projection digest %s differs from reviewed %s", gotDigest, reviewedOwnershipCanonicalSHA256)
	}

	report := Report{
		Contracts:              len(expected[ownershipContract]),
		NormativeSections:      len(expected[ownershipNormativeSection]),
		AcceptanceCases:        len(acceptanceIDs),
		Fixtures:               len(expected[ownershipFixture]),
		CompatibilityContracts: len(compatibilityCatalog.Contracts),
		AssignedScopes:         len(assignedBindings),
		SectionBindings:        len(coverage),
		UnownedSections:        len(unowned),
	}
	for _, measured := range coverage {
		report.NormativeClauses += measured.Total
		report.DischargedClauses += measured.Discharged
		switch measured.Level {
		case coverageFull:
			report.FullCoverage++
		case coveragePartial:
			report.PartialCoverage++
		case coverageSliver:
			report.SliverCoverage++
		case coverageUnevidenced:
			report.UnevidencedCoverage++
		case coverageUnmeasured:
			report.UnmeasuredCoverage++
		}
	}
	return report, nil
}

// verifySectionCoverage measures every registered section binding against the
// pinned document and refuses a coverage claim on any other ownership kind. A
// contract or fixture owner has no section to be measured against, so a
// coverage field there would be a claim the gate cannot check.
func verifySectionCoverage(
	document *specdoc.Document,
	groups []ownershipGroup,
	acceptanceIDs map[string]struct{},
) (map[string]sectionCoverage, error) {
	result := make(map[string]sectionCoverage)
	for index, group := range groups {
		if group.Kind != ownershipSectionBinding {
			if group.Coverage != "" || group.Gap != "" || len(group.Clauses) != 0 {
				return nil, fail(
					"ownership group %d (%s) claims section coverage, which only a section binding can be measured for",
					index, group.Kind)
			}
			continue
		}
		if len(group.Keys) != 1 {
			return nil, fail("section binding ownership group %d has %d keys, want exactly one", index, len(group.Keys))
		}
		measured, err := verifySectionBindingCoverage(document, group, acceptanceIDs)
		if err != nil {
			return nil, err
		}
		if _, duplicate := result[measured.Key]; duplicate {
			return nil, fail("duplicate section_binding implementation owner for %q", measured.Key)
		}
		result[measured.Key] = measured
	}
	return result, nil
}

func verifyAcceptanceCases(checker *sourceChecker, cases []acceptanceCase) (map[string]struct{}, error) {
	if len(cases) == 0 {
		return nil, fail("acceptance case registry is empty")
	}
	seen := make(map[string]struct{}, len(cases))
	for index, acceptance := range cases {
		if acceptance.ID == "" {
			return nil, fail("acceptance case %d has an empty id", index)
		}
		if _, duplicate := seen[acceptance.ID]; duplicate {
			return nil, fail("duplicate acceptance case %q", acceptance.ID)
		}
		seen[acceptance.ID] = struct{}{}
		if err := checker.verify(acceptance.Production, false); err != nil {
			return nil, fail("acceptance case %q production owner: %v", acceptance.ID, err)
		}
		if len(acceptance.Tests) == 0 {
			return nil, fail("acceptance case %q has no executable test owner", acceptance.ID)
		}
		testSeen := make(map[codeReference]struct{}, len(acceptance.Tests))
		for _, test := range acceptance.Tests {
			if _, duplicate := testSeen[test]; duplicate {
				return nil, fail("acceptance case %q repeats test owner %s:%s", acceptance.ID, test.Path, test.Declaration)
			}
			testSeen[test] = struct{}{}
			if err := checker.verify(test, true); err != nil {
				return nil, fail("acceptance case %q test owner: %v", acceptance.ID, err)
			}
		}
	}
	return seen, nil
}

func verifyOwnershipGroups(
	checker *sourceChecker,
	groups []ownershipGroup,
	acceptanceIDs map[string]struct{},
	allowed map[ownershipKind]map[string]struct{},
	required map[ownershipKind]map[string]struct{},
	selectedSectionBindings map[string]struct{},
) (map[ownershipKind]map[string]struct{}, error) {
	owned := map[ownershipKind]map[string]struct{}{
		ownershipContract:         make(map[string]struct{}),
		ownershipNormativeSection: make(map[string]struct{}),
		ownershipSectionBinding:   make(map[string]struct{}),
		ownershipFixture:          make(map[string]struct{}),
	}
	if len(groups) == 0 {
		return nil, fail("implementation ownership registry is empty")
	}
	for index, group := range groups {
		if _, ok := allowed[group.Kind]; !ok {
			return nil, fail("ownership group %d has unknown kind %q", index, group.Kind)
		}
		if group.Kind == ownershipSectionBinding && selectedSectionBindings != nil {
			selected := false
			for _, key := range group.Keys {
				if _, ok := selectedSectionBindings[key]; ok {
					selected = true
					break
				}
			}
			if !selected {
				continue
			}
		}
		if group.Kind == ownershipSectionBinding && len(group.Keys) != 1 {
			return nil, fail("section binding ownership group %d has %d keys, want exactly one", index, len(group.Keys))
		}
		if len(group.Keys) == 0 {
			return nil, fail("ownership group %d (%s) has no registered keys", index, group.Kind)
		}
		if err := checker.verify(group.Production, false); err != nil {
			if group.Kind == ownershipSectionBinding {
				return nil, fail("section binding %q production owner: %v", group.Keys[0], err)
			}
			return nil, fail("ownership group %d (%s) production owner: %v", index, group.Kind, err)
		}
		if len(group.AcceptanceCases) == 0 {
			if group.Kind == ownershipSectionBinding {
				return nil, fail("section binding %q has no scope-specific acceptance owner", group.Keys[0])
			}
			return nil, fail("ownership group %d (%s) has no acceptance owner", index, group.Kind)
		}
		caseSeen := make(map[string]struct{}, len(group.AcceptanceCases))
		for _, acceptanceID := range group.AcceptanceCases {
			if _, duplicate := caseSeen[acceptanceID]; duplicate {
				return nil, fail("ownership group %d (%s) repeats acceptance case %q", index, group.Kind, acceptanceID)
			}
			caseSeen[acceptanceID] = struct{}{}
			if _, ok := acceptanceIDs[acceptanceID]; !ok {
				if group.Kind == ownershipSectionBinding {
					return nil, fail("section binding %q references unregistered acceptance case %q", group.Keys[0], acceptanceID)
				}
				return nil, fail("ownership group %d (%s) references unregistered acceptance case %q", index, group.Kind, acceptanceID)
			}
		}
		for _, key := range group.Keys {
			if key == "" {
				return nil, fail("ownership group %d (%s) has an empty key", index, group.Kind)
			}
			if _, duplicate := owned[group.Kind][key]; duplicate {
				return nil, fail("duplicate %s implementation owner for %q", group.Kind, key)
			}
			owned[group.Kind][key] = struct{}{}
		}
	}
	for kind, wanted := range required {
		for key := range wanted {
			if _, ok := owned[kind][key]; !ok {
				return nil, fail("registered %s %q has no implementation owner", kind, key)
			}
		}
	}
	for kind, registered := range owned {
		for key := range registered {
			if _, ok := allowed[kind][key]; !ok {
				return nil, fail("implementation owner is self-minted for unknown %s %q", kind, key)
			}
		}
	}
	return owned, nil
}

func resolveAssignedSections(pinned []string, assigned []string) ([]assignedSectionBinding, error) {
	if assigned == nil {
		return nil, nil
	}
	pinnedSet := make(map[string]struct{}, len(pinned))
	for _, section := range pinned {
		pinnedSet[section] = struct{}{}
	}
	bindings := make([]assignedSectionBinding, 0, len(assigned))
	seen := make(map[string]struct{}, len(assigned))
	for _, raw := range assigned {
		scope := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "§"))
		if scope == "" {
			return nil, fail("invalid assigned section %q", raw)
		}
		canonical := strings.ToLower(strings.Join(strings.Fields(scope), "-"))
		root := canonical
		identifiers := []string{canonical}
		if strings.HasPrefix(canonical, "appendix-") {
			if !specpin.IsSectionV050(canonical) {
				return nil, fail("assigned section %q is not a real v0.5.0 section identifier", raw)
			}
		} else {
			matches := assignedSectionPattern.FindStringSubmatch(scope)
			if matches == nil {
				return nil, fail("invalid assigned section %q", raw)
			}
			start := canonicalNumericSection(matches[1])
			end := start
			if matches[2] != "" {
				end = canonicalNumericSection(matches[2])
			}
			root = strings.Split(start, ".")[0]
			if strings.Split(end, ".")[0] != root {
				return nil, fail("invalid assigned section %q", raw)
			}
			for _, identifier := range []string{start, end} {
				if !specpin.IsSectionV050(identifier) {
					return nil, fail("assigned section %q is not a real v0.5.0 section identifier", raw)
				}
			}
			expanded, expandErr := expandV050SectionRange(start, end)
			if expandErr != nil {
				return nil, fail("invalid assigned section %q: %v", raw, expandErr)
			}
			identifiers = expanded
		}
		if _, ok := pinnedSet[root]; !ok {
			return nil, fail("assigned section %q is outside pinned normative scope", raw)
		}
		keys := make([]string, len(identifiers))
		for index, identifier := range identifiers {
			keys[index] = sectionBindingKey(identifier)
		}
		bindingIdentity := strings.Join(keys, "\x00")
		if _, duplicate := seen[bindingIdentity]; duplicate {
			continue
		}
		seen[bindingIdentity] = struct{}{}
		bindings = append(bindings, assignedSectionBinding{Scope: raw, Keys: keys})
	}
	return bindings, nil
}

func expandV050SectionRange(start, end string) ([]string, error) {
	inventory := specpin.SectionInventoryV050()
	startIndex := -1
	endIndex := -1
	for index, identifier := range inventory {
		if identifier == start {
			startIndex = index
		}
		if identifier == end {
			endIndex = index
		}
	}
	if startIndex == -1 || endIndex == -1 {
		return nil, fmt.Errorf("range endpoint is not a real v0.5.0 section identifier")
	}
	if endIndex < startIndex {
		return nil, fmt.Errorf("section range descends from %s to %s", start, end)
	}
	return append([]string(nil), inventory[startIndex:endIndex+1]...), nil
}

func sectionBindingKey(identifier string) string {
	return "section:" + identifier
}

func expectedSectionBindingInventory() map[string]struct{} {
	result := make(map[string]struct{})
	for _, identifier := range specpin.SectionInventoryV050() {
		result[sectionBindingKey(identifier)] = struct{}{}
	}
	return result
}

func expectedCatalogSectionBindings(value catalog.Catalog) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	addDefinition := func(normativeSection string, fixtures []string) error {
		for _, expression := range strings.Split(normativeSection, ",") {
			expression = strings.TrimSpace(expression)
			parts := strings.Split(expression, "-")
			if len(parts) > 2 || len(parts) == 0 {
				return fail("catalog normative section %q is not an exact section or range", normativeSection)
			}
			start := canonicalNumericSection(strings.TrimSpace(parts[0]))
			end := start
			if len(parts) == 2 {
				end = canonicalNumericSection(strings.TrimSpace(parts[1]))
			}
			identifiers, err := expandV050SectionRange(start, end)
			if err != nil {
				return fail("catalog normative section %q: %v", normativeSection, err)
			}
			for _, identifier := range identifiers {
				result[sectionBindingKey(identifier)] = struct{}{}
			}
		}
		for _, fixture := range fixtures {
			match := catalogAppendixPattern.FindStringSubmatch(fixture)
			if match != nil {
				result[sectionBindingKey("appendix-"+strings.ToLower(match[1]))] = struct{}{}
			}
		}
		return nil
	}
	for _, operation := range value.Operations {
		if err := addDefinition(operation.NormativeSection, operation.FixtureFamilies); err != nil {
			return nil, err
		}
	}
	for _, capability := range value.Capabilities {
		if err := addDefinition(capability.NormativeSection, capability.FixtureFamilies); err != nil {
			return nil, err
		}
	}
	for _, event := range value.Events {
		if err := addDefinition(event.NormativeSection, event.FixtureFamilies); err != nil {
			return nil, err
		}
	}
	for _, item := range value.Errors {
		if err := addDefinition(item.NormativeSection, item.FixtureFamilies); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func canonicalNumericSection(identifier string) string {
	parts := strings.Split(identifier, ".")
	for index := 1; index < len(parts); index++ {
		if len(parts[index]) != 1 {
			continue
		}
		character := parts[index][0]
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') {
			parts[index] = strings.ToUpper(parts[index])
		}
	}
	return strings.Join(parts, ".")
}

func cloneOwnershipSets(source map[ownershipKind]map[string]struct{}) map[ownershipKind]map[string]struct{} {
	result := make(map[ownershipKind]map[string]struct{}, len(source))
	for kind, values := range source {
		cloned := make(map[string]struct{}, len(values))
		for value := range values {
			cloned[value] = struct{}{}
		}
		result[kind] = cloned
	}
	return result
}

func verifyCatalogContracts(want []specpin.ContractPin, got []catalog.Contract) error {
	if len(got) != len(want) {
		return fail("current catalog has %d contracts, verified source has %d", len(got), len(want))
	}
	for index := range want {
		if got[index].Name != want[index].Name || string(got[index].ID) != want[index].ID ||
			!equalStrings(got[index].Versions, want[index].Versions) {
			return fail("current catalog contract row %d differs from verified source", index)
		}
	}
	return nil
}

func expectedContracts(contracts []specpin.ContractPin) map[string]struct{} {
	result := make(map[string]struct{}, len(contracts))
	for _, contract := range contracts {
		result[contractKey(contract.Name, contract.ID)] = struct{}{}
	}
	return result
}

func contractKey(name, id string) string {
	return name + " [" + id + "]"
}

func expectedNormativeSections(manifest specpin.Manifest, value catalog.Catalog) map[string]struct{} {
	result := make(map[string]struct{})
	for _, section := range manifest.Source.NormativeScope {
		result["source:"+section] = struct{}{}
	}
	for _, operation := range value.Operations {
		result["catalog:"+operation.NormativeSection] = struct{}{}
	}
	for _, capability := range value.Capabilities {
		result["catalog:"+capability.NormativeSection] = struct{}{}
	}
	for _, event := range value.Events {
		result["catalog:"+event.NormativeSection] = struct{}{}
	}
	for _, item := range value.Errors {
		result["catalog:"+item.NormativeSection] = struct{}{}
	}
	return result
}

func expectedFixtures(manifest specpin.Manifest, value catalog.Catalog) map[string]struct{} {
	result := make(map[string]struct{})
	for _, fixture := range manifest.Fixtures {
		result["pin:"+fixture.ID] = struct{}{}
	}
	add := func(fixtures []string) {
		for _, fixture := range fixtures {
			result["catalog:"+fixture] = struct{}{}
		}
	}
	for _, operation := range value.Operations {
		add(operation.FixtureFamilies)
	}
	for _, capability := range value.Capabilities {
		add(capability.FixtureFamilies)
	}
	for _, event := range value.Events {
		add(event.FixtureFamilies)
	}
	for _, item := range value.Errors {
		add(item.FixtureFamilies)
	}
	return result
}

type sourceChecker struct {
	repository fs.FS
	files      map[string]*ast.File
}

func newSourceChecker(repository fs.FS) *sourceChecker {
	return &sourceChecker{repository: repository, files: make(map[string]*ast.File)}
}

func (checker *sourceChecker) verify(reference codeReference, test bool) error {
	if reference.Path == "" || reference.Declaration == "" {
		return errors.New("code reference has an empty path or declaration")
	}
	clean := path.Clean(reference.Path)
	if clean != reference.Path || clean == "." || path.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe code reference path %q", reference.Path)
	}
	if path.Ext(clean) != ".go" {
		return fmt.Errorf("code reference %q is not a Go source file", clean)
	}
	if test {
		if !strings.HasSuffix(clean, "_test.go") || !isExecutableTestName(reference.Declaration) {
			return fmt.Errorf("test owner %s:%s is not an executable Go test reference", clean, reference.Declaration)
		}
	} else if strings.HasSuffix(clean, "_test.go") {
		return fmt.Errorf("production owner %s:%s points to test code", clean, reference.Declaration)
	}

	file, ok := checker.files[clean]
	if !ok {
		source, err := fs.ReadFile(checker.repository, clean)
		if err != nil {
			return fmt.Errorf("read %q: %w", clean, err)
		}
		file, err = parser.ParseFile(token.NewFileSet(), clean, source, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse %q: %w", clean, err)
		}
		checker.files[clean] = file
	}
	if !hasDeclaration(file, reference.Declaration, test) {
		return fmt.Errorf("declaration %q is absent from %q", reference.Declaration, clean)
	}
	return nil
}

func isExecutableTestName(name string) bool {
	for _, prefix := range []string{"Test", "Fuzz"} {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		suffix := name[len(prefix):]
		if suffix == "" {
			return true
		}
		first, _ := utf8.DecodeRuneInString(suffix)
		return !unicode.IsLower(first)
	}
	return false
}

func hasDeclaration(file *ast.File, name string, test bool) bool {
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if value.Name.Name == name && (!test || value.Recv == nil) {
				return true
			}
		case *ast.GenDecl:
			if test {
				continue
			}
			for _, specification := range value.Specs {
				switch item := specification.(type) {
				case *ast.TypeSpec:
					if item.Name.Name == name {
						return true
					}
				case *ast.ValueSpec:
					for _, identifier := range item.Names {
						if identifier.Name == name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func fail(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrTraceability, fmt.Sprintf(format, arguments...))
}

// normativeKeywordPattern matches the RFC 2119 obligation keywords that make a
// specification line a clause an implementation has to discharge. MAY and
// SHOULD are deliberately absent: they create no obligation, so counting them
// would inflate the denominator and make coverage look worse than it is.
var normativeKeywordPattern = regexp.MustCompile(`\b(MUST NOT|MUST|SHALL NOT|SHALL|REQUIRED)\b`)

// clauseIdentifierPattern matches the "<section>#<index>" form of a clause id.
var clauseIdentifierPattern = regexp.MustCompile(`^(.+)#([1-9][0-9]*)$`)

// minimumGapLength is the shortest gap sentence the gate accepts. A gap has to
// say what is missing; "n/a", "todo" and an empty string are not disclosures.
const minimumGapLength = 32

// verifyGapDiscloses checks that a gap sentence is a disclosure rather than
// padding. Length and a substring match alone were not enough: "Section 9.2 is
// not fully covered here yet." satisfied both while saying nothing. A real gap
// says what the binding's own implementation does and does not reach, so it has
// to name the production declaration the binding is registered to as well as
// the section, and it has to name the section as a whole identifier so that a
// gap about 6.55 cannot pass for a gap about 6.5.
//
// This is a tightening, not a proof: a sentence that names both and still says
// nothing useful is admitted, and the gate cannot decide otherwise. That limit
// is stated in README.md.
func verifyGapDiscloses(gap, display, declaration string) error {
	if len(gap) < minimumGapLength {
		return fmt.Errorf("gap %q is shorter than the %d characters a disclosure needs", gap, minimumGapLength)
	}
	if !mentionsSection(gap, display) {
		return fmt.Errorf("gap does not name section %s as a whole identifier", display)
	}
	if declaration != "" && !strings.Contains(gap, declaration) {
		return fmt.Errorf("gap does not name the production declaration %q the binding is registered to", declaration)
	}
	return nil
}

// mentionsSection reports whether text names the section display identifier as
// a whole identifier. A bare substring match accepts "6.55" as a mention of
// "6.5" and "13.15" as a mention of "13.1", which would let a gap about one
// section stand in for another.
func mentionsSection(text, display string) bool {
	pattern, err := regexp.Compile(`(?:^|[^0-9.])` + regexp.QuoteMeta(display) + `(?:[^0-9]|$)`)
	if err != nil {
		return false
	}
	return pattern.MatchString(text)
}

// sectionDocumentIdentifier maps an ownership key to the heading identifier the
// pinned document uses. "section:10.3" is heading 10.3; "section:appendix-d" is
// heading D.
func sectionDocumentIdentifier(key string) (string, error) {
	identifier := strings.TrimPrefix(key, "section:")
	if identifier == key || identifier == "" {
		return "", fmt.Errorf("ownership key %q is not a section binding key", key)
	}
	if rest, ok := strings.CutPrefix(identifier, "appendix-"); ok {
		if len(rest) != 1 {
			return "", fmt.Errorf("ownership key %q is not a real appendix key", key)
		}
		return strings.ToUpper(rest), nil
	}
	return identifier, nil
}

// sectionDisplayName is how a gap sentence has to name the section it is about.
func sectionDisplayName(key string) (string, error) {
	identifier := strings.TrimPrefix(key, "section:")
	if rest, ok := strings.CutPrefix(identifier, "appendix-"); ok {
		if len(rest) != 1 {
			return "", fmt.Errorf("ownership key %q is not a real appendix key", key)
		}
		return "Appendix " + strings.ToUpper(rest), nil
	}
	if identifier == "" || identifier == key {
		return "", fmt.Errorf("ownership key %q is not a section binding key", key)
	}
	return identifier, nil
}

// sectionClauseInventory measures every normative clause the pinned document
// carries under a heading, including its subheadings. A parent heading owns its
// children's obligations, so Appendix D is measured over D.1..D.n rather than
// over the two lines between its title and its first subheading.
func sectionClauseInventory(document *specdoc.Document, key string) ([]normativeClause, error) {
	identifier, err := sectionDocumentIdentifier(key)
	if err != nil {
		return nil, err
	}
	prefix := identifier + "."
	var result []normativeClause
	for line := 1; line <= document.LineCount(); line++ {
		owner, ok := document.SectionID(line)
		if !ok || (owner != identifier && !strings.HasPrefix(owner, prefix)) {
			continue
		}
		text, ok := document.Line(line)
		if !ok {
			return nil, fmt.Errorf("pinned document line %d is unreadable", line)
		}
		if !normativeKeywordPattern.MatchString(text) {
			continue
		}
		result = append(result, normativeClause{
			ID:   fmt.Sprintf("%s#%d", identifier, len(result)+1),
			Line: line,
			Text: strings.TrimSpace(text),
		})
	}
	return result, nil
}

// coverageBucket is the measured coverage level. It is computed from the two
// counts and never read from the registry.
func coverageBucket(discharged, total int) coverageLevel {
	switch {
	case total == 0:
		return coverageUnmeasured
	case discharged == 0:
		return coverageUnevidenced
	case discharged >= total:
		return coverageFull
	case discharged*2 >= total:
		return coveragePartial
	default:
		return coverageSliver
	}
}

// sectionCoverage is the measured result for one section binding.
type sectionCoverage struct {
	Key        string
	Level      coverageLevel
	Discharged int
	Total      int
	Gap        string
}

// Ratio renders the measured coverage as the ratio it was computed from, so a
// report never has to describe coverage in prose.
func (coverage sectionCoverage) Ratio() string {
	return fmt.Sprintf("%d/%d", coverage.Discharged, coverage.Total)
}

// verifySectionBindingCoverage measures one section binding against the pinned
// document and refuses a declared level that differs from the measured one.
//
// This is the check that stops a binding from claiming a whole section while
// implementing a corner of it. It verifies what it can decide: that every
// enumerated clause is a real obligation of the claimed section, quoted
// verbatim at the line it occupies, and named by an acceptance case the binding
// itself registers; and that the declared level equals the measured ratio. It
// cannot decide that the named acceptance case exercises the clause's meaning —
// that residual is stated in README.md and in the coverage artifact.
func verifySectionBindingCoverage(
	document *specdoc.Document,
	group ownershipGroup,
	acceptanceIDs map[string]struct{},
) (sectionCoverage, error) {
	key := group.Keys[0]
	inventory, err := sectionClauseInventory(document, key)
	if err != nil {
		return sectionCoverage{}, fail("section binding %q clause inventory: %v", key, err)
	}
	byIdentifier := make(map[string]normativeClause, len(inventory))
	for _, clause := range inventory {
		byIdentifier[clause.ID] = clause
	}
	groupCases := make(map[string]struct{}, len(group.AcceptanceCases))
	for _, acceptanceID := range group.AcceptanceCases {
		groupCases[acceptanceID] = struct{}{}
	}

	seen := make(map[string]struct{}, len(group.Clauses))
	for _, declared := range group.Clauses {
		clause, known := byIdentifier[declared.ID]
		if !known {
			return sectionCoverage{}, fail(
				"section binding %q discharges clause %q, which is not one of the %d normative clauses of the pinned section",
				key, declared.ID, len(inventory))
		}
		if _, duplicate := seen[declared.ID]; duplicate {
			return sectionCoverage{}, fail("section binding %q repeats discharged clause %q", key, declared.ID)
		}
		seen[declared.ID] = struct{}{}
		if declared.Line != clause.Line {
			return sectionCoverage{}, fail(
				"section binding %q clause %q declares line %d, but the pinned clause is at line %d",
				key, declared.ID, declared.Line, clause.Line)
		}
		excerpt := specdoc.Normalize(declared.Excerpt)
		if excerpt == "" {
			return sectionCoverage{}, fail("section binding %q clause %q quotes an empty excerpt", key, declared.ID)
		}
		if !normativeKeywordPattern.MatchString(excerpt) {
			return sectionCoverage{}, fail(
				"section binding %q clause %q quotes text that carries no RFC 2119 obligation", key, declared.ID)
		}
		if !quoteBeginsAtLine(document, declared.Excerpt, clause.Line) {
			return sectionCoverage{}, fail(
				"section binding %q clause %q does not quote the pinned document verbatim at line %d",
				key, declared.ID, clause.Line)
		}
		if len(declared.AcceptanceCases) == 0 {
			return sectionCoverage{}, fail(
				"section binding %q clause %q names no acceptance case that discharges it", key, declared.ID)
		}
		caseSeen := make(map[string]struct{}, len(declared.AcceptanceCases))
		for _, acceptanceID := range declared.AcceptanceCases {
			if _, duplicate := caseSeen[acceptanceID]; duplicate {
				return sectionCoverage{}, fail(
					"section binding %q clause %q repeats acceptance case %q", key, declared.ID, acceptanceID)
			}
			caseSeen[acceptanceID] = struct{}{}
			if _, registered := acceptanceIDs[acceptanceID]; !registered {
				return sectionCoverage{}, fail(
					"section binding %q clause %q references unregistered acceptance case %q", key, declared.ID, acceptanceID)
			}
			if _, owned := groupCases[acceptanceID]; !owned {
				return sectionCoverage{}, fail(
					"section binding %q clause %q discharges through acceptance case %q, which the binding does not own",
					key, declared.ID, acceptanceID)
			}
		}
	}

	measured := coverageBucket(len(group.Clauses), len(inventory))
	if group.Coverage == "" {
		return sectionCoverage{}, fail(
			"section binding %q declares no coverage level; the pinned section carries %d normative clauses and the binding enumerates %d",
			key, len(inventory), len(group.Clauses))
	}
	if group.Coverage != measured {
		return sectionCoverage{}, fail(
			"section binding %q claims %s coverage but discharges %d of the %d normative clauses of the pinned section, which is %s coverage",
			key, group.Coverage, len(group.Clauses), len(inventory), measured)
	}
	display, err := sectionDisplayName(key)
	if err != nil {
		return sectionCoverage{}, fail("section binding %q: %v", key, err)
	}
	// Every level below full owes a gap, coverageUnmeasured included. Before
	// this the gate refused a gap on the unmeasured bucket, so the bindings with
	// the least evidence behind them were the only ones structurally forbidden
	// from disclosing it.
	switch measured {
	case coverageFull:
		if group.Gap != "" {
			return sectionCoverage{}, fail("section binding %q claims %s coverage and still names a gap", key, measured)
		}
	default:
		if err := verifyGapDiscloses(group.Gap, display, group.Production.Declaration); err != nil {
			return sectionCoverage{}, fail(
				"section binding %q claims %s coverage without naming what %s leaves unimplemented: %v",
				key, measured, display, err)
		}
	}
	return sectionCoverage{
		Key:        key,
		Level:      measured,
		Discharged: len(group.Clauses),
		Total:      len(inventory),
		Gap:        group.Gap,
	}, nil
}

// quoteBeginsAtLine reports whether the normalized excerpt occurs in the pinned
// document beginning on the given line. An excerpt matching elsewhere, or
// nowhere, is not a quote of the clause it claims.
func quoteBeginsAtLine(document *specdoc.Document, excerpt string, line int) bool {
	for _, candidate := range document.QuoteLines(excerpt) {
		if candidate == line {
			return true
		}
	}
	return false
}

// verifyUnownedSections checks the disclosed unimplemented sections. An entry
// must name a real section, must not also be owned, must not cover a section
// the generated catalog requires an owner for, and must state a gap and the
// evidence for it.
func verifyUnownedSections(
	entries []unownedSection,
	allowed map[string]struct{},
	requiredByCatalog map[string]struct{},
	owned map[string]struct{},
) (map[string]unownedSection, error) {
	result := make(map[string]unownedSection, len(entries))
	for index, entry := range entries {
		if entry.Key == "" {
			return nil, fail("unowned section %d has an empty key", index)
		}
		if _, duplicate := result[entry.Key]; duplicate {
			return nil, fail("duplicate unowned section %q", entry.Key)
		}
		if _, known := allowed[entry.Key]; !known {
			return nil, fail("unowned section %q is self-minted for an unknown section", entry.Key)
		}
		if _, required := requiredByCatalog[entry.Key]; required {
			return nil, fail(
				"unowned section %q is required to have an implementation owner by the generated catalog", entry.Key)
		}
		if _, conflict := owned[entry.Key]; conflict {
			return nil, fail("section %q is registered as both owned and unowned", entry.Key)
		}
		display, err := sectionDisplayName(entry.Key)
		if err != nil {
			return nil, fail("unowned section %q: %v", entry.Key, err)
		}
		// An unowned section has no production declaration by construction, so
		// the gap is checked for length and a whole-identifier section mention
		// only, and the separate evidence field carries the rest.
		if err := verifyGapDiscloses(entry.Gap, display, ""); err != nil {
			return nil, fail("unowned section %q does not name what %s leaves unimplemented: %v", entry.Key, display, err)
		}
		if len(entry.Evidence) < minimumGapLength {
			return nil, fail("unowned section %q states no evidence for its gap", entry.Key)
		}
		result[entry.Key] = entry
	}
	return result, nil
}
