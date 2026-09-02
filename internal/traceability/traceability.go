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
	reviewedOwnershipCanonicalSHA256 = "b5882b265b29d0c286c62a0263a6a38444d972434a5c5d42569efe6fc3af0b2c"
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
	Kind            ownershipKind `json:"kind"`
	Keys            []string      `json:"keys"`
	Production      codeReference `json:"production"`
	AcceptanceCases []string      `json:"acceptance_cases"`
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
	return verifyOwnership(repository, registry, manifest, currentCatalog, compatibilityCatalog, bindings)
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
	owned, err := verifyOwnershipGroups(checker, registry.Ownership, acceptanceIDs, expected, required, selectedSectionBindings)
	if err != nil {
		return Report{}, err
	}
	for _, binding := range assignedBindings {
		for _, key := range binding.Keys {
			if _, ok := owned[ownershipSectionBinding][key]; !ok {
				return Report{}, fail("assigned section %q binding %q has no scoped implementation owner", binding.Scope, key)
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

	return Report{
		Contracts:              len(expected[ownershipContract]),
		NormativeSections:      len(expected[ownershipNormativeSection]),
		AcceptanceCases:        len(acceptanceIDs),
		Fixtures:               len(expected[ownershipFixture]),
		CompatibilityContracts: len(compatibilityCatalog.Contracts),
		AssignedScopes:         len(assignedBindings),
	}, nil
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
