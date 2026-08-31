// Package cataloggen validates reviewed AX implementation metadata and emits
// the deterministic typed Go catalog consumed by internal/catalog.
package cataloggen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

const (
	metadataFormat                  = "ax-implementation-catalog"
	metadataFormatVersion           = 1
	reviewedMetadataCanonicalSHA256 = "4ddf049a7bc1abf29030283eaf7ad397555a4be2b4b4fce34d458a3ca2b089e8"
)

var ErrInvalidMetadata = errors.New("invalid implementation catalog metadata")

type metadata struct {
	Format             string             `json:"format"`
	FormatVersion      int                `json:"format_version"`
	Source             sourceMetadata     `json:"source"`
	NormativeScope     []string           `json:"normative_scope"`
	OperationFamilies  []operationFamily  `json:"operation_families"`
	CapabilityFamilies []capabilityFamily `json:"capability_families"`
	EventFamilies      []eventFamily      `json:"event_families"`
	ErrorCatalog       errorCatalog       `json:"error_catalog"`
}

type sourceMetadata struct {
	Repository     string `json:"repository"`
	Release        string `json:"release"`
	Commit         string `json:"commit"`
	DocumentPath   string `json:"document_path"`
	DocumentSHA256 string `json:"document_sha256"`
}

type operationFamily struct {
	Family           string          `json:"family"`
	ContractID       string          `json:"contract_id"`
	ContractVersions []string        `json:"contract_versions"`
	Releases         []string        `json:"releases"`
	NormativeSection string          `json:"normative_section"`
	FixtureFamilies  []string        `json:"fixture_families"`
	Operations       []string        `json:"operations"`
	IsolatedOutputs  []string        `json:"isolated_outputs"`
	MutationGroups   []mutationGroup `json:"mutation_groups"`
}

type mutationGroup struct {
	Operations       []string `json:"operations"`
	IdempotencyKey   string   `json:"idempotency_key"`
	RecoveryEvidence []string `json:"recovery_evidence"`
}

type capabilityFamily struct {
	Family           string   `json:"family"`
	ContractID       string   `json:"contract_id"`
	ContractVersions []string `json:"contract_versions"`
	Releases         []string `json:"releases"`
	NormativeSection string   `json:"normative_section"`
	FixtureFamilies  []string `json:"fixture_families"`
	Capabilities     []string `json:"capabilities"`
}

type eventFamily struct {
	Family           string       `json:"family"`
	ContractID       string       `json:"contract_id"`
	Releases         []string     `json:"releases"`
	NormativeSection string       `json:"normative_section"`
	FixtureFamilies  []string     `json:"fixture_families"`
	Groups           []eventGroup `json:"groups"`
}

type eventGroup struct {
	ContractVersions []string `json:"contract_versions"`
	Events           []string `json:"events"`
}

type errorCatalog struct {
	ContractID       string       `json:"contract_id"`
	ContractVersions []string     `json:"contract_versions"`
	NormativeSection string       `json:"normative_section"`
	FixtureFamilies  []string     `json:"fixture_families"`
	Groups           []errorGroup `json:"groups"`
}

type errorGroup struct {
	IntroducedVersion string         `json:"introduced_version"`
	Releases          []string       `json:"releases"`
	Mappings          []errorMapping `json:"mappings"`
}

type errorMapping struct {
	ExitCode int      `json:"exit_code"`
	Codes    []string `json:"codes"`
}

type expandedOperation struct {
	Family           string
	Name             string
	ContractID       string
	ContractVersions []string
	Effect           string
	IdempotencyKey   string
	RecoveryEvidence []string
	NormativeSection string
	FixtureFamilies  []string
	Releases         []string
}

type expandedCapability struct {
	Family           string
	Name             string
	ContractID       string
	ContractVersions []string
	NormativeSection string
	FixtureFamilies  []string
	Releases         []string
}

type expandedEvent struct {
	Family           string
	Name             string
	ContractID       string
	ContractVersions []string
	NormativeSection string
	FixtureFamilies  []string
	Releases         []string
}

type expandedError struct {
	Code             string
	ExitCode         int
	ContractID       string
	ContractVersions []string
	NormativeSection string
	FixtureFamilies  []string
	Releases         []string
}

var (
	namePattern       = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)
	capabilityPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	errorPattern      = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// Generate is the production generation entry point. It is pure: callers own
// any output write and receive no partial artifact on validation failure.
func Generate(metadataBytes, contractLock []byte) ([]byte, error) {
	manifest, err := specpin.Verify(contractLock)
	if err != nil {
		return nil, invalid("verify normative lock: %v", err)
	}
	decoded, err := decodeMetadata(metadataBytes)
	if err != nil {
		return nil, err
	}
	if err := validateMetadata(decoded, manifest); err != nil {
		return nil, err
	}

	operations := expandOperations(decoded.OperationFamilies)
	capabilities := expandCapabilities(decoded.CapabilityFamilies)
	events := expandEvents(decoded.EventFamilies)
	errorsCatalog := expandErrors(decoded.ErrorCatalog)
	digest := sha256.Sum256(metadataBytes)

	var output bytes.Buffer
	output.WriteString("// Code generated by go generate ./internal/catalog; DO NOT EDIT.\n")
	fmt.Fprintf(&output, "// Metadata SHA-256: %s\n\n", hex.EncodeToString(digest[:]))
	output.WriteString("package catalog\n\n")
	output.WriteString("var generatedDefinition = catalogDefinition{\n")
	writeSource(&output, decoded)
	fmt.Fprintf(&output, "\tMetadataSHA256: %q,\n", hex.EncodeToString(digest[:]))
	writeContracts(&output, manifest)
	writeOperations(&output, operations)
	writeCapabilities(&output, capabilities)
	writeEvents(&output, events)
	writeErrors(&output, errorsCatalog)
	output.WriteString("}\n")

	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return nil, invalid("format generated Go: %v", err)
	}
	return formatted, nil
}

func decodeMetadata(candidate []byte) (metadata, error) {
	var decoded metadata
	decoder := json.NewDecoder(bytes.NewReader(candidate))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return metadata{}, invalid("decode metadata: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return metadata{}, invalid("multiple JSON values")
		}
		return metadata{}, invalid("trailing JSON: %v", err)
	}
	return decoded, nil
}

func validateMetadata(value metadata, manifest specpin.Manifest) error {
	if value.Format != metadataFormat || value.FormatVersion != metadataFormatVersion {
		return invalid("unsupported format %q version %d", value.Format, value.FormatVersion)
	}
	if value.Source.Repository != manifest.Source.Repository ||
		value.Source.Release != manifest.Source.Release ||
		value.Source.Commit != manifest.Source.Commit ||
		value.Source.DocumentPath != manifest.Source.Document.Path ||
		value.Source.DocumentSHA256 != manifest.Source.Document.SHA256 {
		return invalid("source identity differs from verified normative lock")
	}
	if !equalStrings(value.NormativeScope, manifest.Source.NormativeScope) {
		return invalid("normative scope differs from verified source pin")
	}
	if len(value.OperationFamilies) == 0 || len(value.CapabilityFamilies) == 0 || len(value.EventFamilies) == 0 {
		return invalid("operation, capability, and event catalogs must all be present")
	}

	contracts := make(map[string]map[string]struct{})
	for _, contract := range manifest.Contracts {
		versions := contracts[contract.ID]
		if versions == nil {
			versions = make(map[string]struct{})
			contracts[contract.ID] = versions
		}
		for _, version := range contract.Versions {
			versions[version] = struct{}{}
		}
	}
	releaseContracts, err := releaseContractVersions(manifest)
	if err != nil {
		return invalid("build release contract projections: %v", err)
	}

	seenFamilies := make(map[string]struct{})
	for index, family := range value.OperationFamilies {
		if err := validateFamilyHeader("operation", index, family.Family, family.ContractID, family.ContractVersions, family.Releases, family.NormativeSection, family.FixtureFamilies, contracts, releaseContracts, seenFamilies); err != nil {
			return err
		}
		if len(family.Operations) == 0 {
			return invalid("operation family %q is empty", family.Family)
		}
		operationSet, err := uniqueNames("operation "+family.Family, family.Operations, namePattern)
		if err != nil {
			return err
		}
		isolated, err := uniqueNames("isolated output "+family.Family, family.IsolatedOutputs, namePattern)
		if err != nil {
			return err
		}
		for name := range isolated {
			if _, ok := operationSet[name]; !ok {
				return invalid("operation family %q isolates unknown operation %q", family.Family, name)
			}
		}
		mutations := make(map[string]struct{})
		for groupIndex, group := range family.MutationGroups {
			if len(group.Operations) == 0 || group.IdempotencyKey == "" || len(group.RecoveryEvidence) == 0 {
				return invalid("operation family %q mutation group %d lacks operations, idempotency, or recovery evidence", family.Family, groupIndex)
			}
			if hasEmpty(group.RecoveryEvidence) {
				return invalid("operation family %q mutation group %d has empty recovery evidence", family.Family, groupIndex)
			}
			for _, name := range group.Operations {
				if _, ok := operationSet[name]; !ok {
					return invalid("operation family %q marks unknown operation %q durable", family.Family, name)
				}
				if _, ok := isolated[name]; ok {
					return invalid("operation family %q operation %q is both isolated and durable", family.Family, name)
				}
				if _, duplicate := mutations[name]; duplicate {
					return invalid("operation family %q repeats durable operation %q", family.Family, name)
				}
				mutations[name] = struct{}{}
			}
		}
	}

	seenFamilies = make(map[string]struct{})
	for index, family := range value.CapabilityFamilies {
		if err := validateFamilyHeader("capability", index, family.Family, family.ContractID, family.ContractVersions, family.Releases, family.NormativeSection, family.FixtureFamilies, contracts, releaseContracts, seenFamilies); err != nil {
			return err
		}
		if len(family.Capabilities) == 0 {
			return invalid("capability family %q is empty", family.Family)
		}
		if _, err := uniqueNames("capability "+family.Family, family.Capabilities, capabilityPattern); err != nil {
			return err
		}
	}

	seenFamilies = make(map[string]struct{})
	for index, family := range value.EventFamilies {
		if family.Family == "" || family.ContractID == "" || family.NormativeSection == "" || len(family.FixtureFamilies) == 0 {
			return invalid("event family %d has incomplete header", index)
		}
		if _, duplicate := seenFamilies[family.Family]; duplicate {
			return invalid("duplicate event family %q", family.Family)
		}
		seenFamilies[family.Family] = struct{}{}
		if err := validateReleases(family.Releases); err != nil {
			return invalid("event family %q: %v", family.Family, err)
		}
		seenEvents := make(map[string]struct{})
		if len(family.Groups) == 0 {
			return invalid("event family %q is empty", family.Family)
		}
		for groupIndex, group := range family.Groups {
			if err := validateContractBinding(family.ContractID, group.ContractVersions, contracts); err != nil {
				return invalid("event family %q group %d: %v", family.Family, groupIndex, err)
			}
			if err := validateReleaseBinding(family.ContractID, group.ContractVersions, family.Releases, releaseContracts); err != nil {
				return invalid("event family %q group %d: %v", family.Family, groupIndex, err)
			}
			if len(group.Events) == 0 {
				return invalid("event family %q group %d is empty", family.Family, groupIndex)
			}
			for _, name := range group.Events {
				if !namePattern.MatchString(name) {
					return invalid("event family %q has invalid event %q", family.Family, name)
				}
				if _, duplicate := seenEvents[name]; duplicate {
					return invalid("event family %q repeats event %q", family.Family, name)
				}
				seenEvents[name] = struct{}{}
			}
		}
	}

	if value.ErrorCatalog.ContractID == "" || value.ErrorCatalog.NormativeSection == "" || len(value.ErrorCatalog.FixtureFamilies) == 0 {
		return invalid("error catalog has incomplete header")
	}
	if err := validateContractBinding(value.ErrorCatalog.ContractID, value.ErrorCatalog.ContractVersions, contracts); err != nil {
		return invalid("error catalog: %v", err)
	}
	versionIndex := make(map[string]int, len(value.ErrorCatalog.ContractVersions))
	for index, version := range value.ErrorCatalog.ContractVersions {
		versionIndex[version] = index
	}
	if len(value.ErrorCatalog.Groups) != len(value.ErrorCatalog.ContractVersions) {
		return invalid("error catalog must define one introduced-code group per contract version")
	}
	seenCodes := make(map[string]struct{})
	for groupIndex, group := range value.ErrorCatalog.Groups {
		if group.IntroducedVersion != value.ErrorCatalog.ContractVersions[groupIndex] {
			return invalid("error group %d is not the ordered contract version %q", groupIndex, value.ErrorCatalog.ContractVersions[groupIndex])
		}
		if _, ok := versionIndex[group.IntroducedVersion]; !ok {
			return invalid("error group %d introduces unsupported version %q", groupIndex, group.IntroducedVersion)
		}
		if err := validateReleases(group.Releases); err != nil {
			return invalid("error group %d: %v", groupIndex, err)
		}
		introducedVersions := value.ErrorCatalog.ContractVersions[versionIndex[group.IntroducedVersion]:]
		if err := validateReleaseBinding(value.ErrorCatalog.ContractID, introducedVersions, group.Releases, releaseContracts); err != nil {
			return invalid("error group %d: %v", groupIndex, err)
		}
		if len(group.Mappings) == 0 {
			return invalid("error group %d is empty", groupIndex)
		}
		for _, mapping := range group.Mappings {
			if mapping.ExitCode <= 0 || len(mapping.Codes) == 0 {
				return invalid("error group %d has invalid exit mapping", groupIndex)
			}
			for _, code := range mapping.Codes {
				if !errorPattern.MatchString(code) {
					return invalid("invalid error code %q", code)
				}
				if _, duplicate := seenCodes[code]; duplicate {
					return invalid("duplicate error code %q", code)
				}
				seenCodes[code] = struct{}{}
			}
		}
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return invalid("encode canonical metadata: %v", err)
	}
	digest := sha256.Sum256(canonical)
	gotDigest := hex.EncodeToString(digest[:])
	if gotDigest != reviewedMetadataCanonicalSHA256 {
		return invalid("metadata projection digest %s differs from reviewed %s", gotDigest, reviewedMetadataCanonicalSHA256)
	}
	return nil
}

func validateFamilyHeader(kind string, index int, family, contractID string, versions, releases []string, section string, fixtures []string, contracts map[string]map[string]struct{}, releaseContracts map[string]map[string]map[string]struct{}, seen map[string]struct{}) error {
	if family == "" || contractID == "" || section == "" || len(fixtures) == 0 {
		return invalid("%s family %d has incomplete header", kind, index)
	}
	if _, duplicate := seen[family]; duplicate {
		return invalid("duplicate %s family %q", kind, family)
	}
	seen[family] = struct{}{}
	if err := validateContractBinding(contractID, versions, contracts); err != nil {
		return invalid("%s family %q: %v", kind, family, err)
	}
	if err := validateReleases(releases); err != nil {
		return invalid("%s family %q: %v", kind, family, err)
	}
	if err := validateReleaseBinding(contractID, versions, releases, releaseContracts); err != nil {
		return invalid("%s family %q: %v", kind, family, err)
	}
	if hasEmpty(fixtures) {
		return invalid("%s family %q has empty fixture reference", kind, family)
	}
	return nil
}

func releaseContractVersions(manifest specpin.Manifest) (map[string]map[string]map[string]struct{}, error) {
	result := make(map[string]map[string]map[string]struct{}, 2)
	for _, release := range []string{specpin.ReleaseV043, specpin.ReleaseV050} {
		contracts, err := manifest.ContractsForRelease(release)
		if err != nil {
			return nil, err
		}
		byID := make(map[string]map[string]struct{})
		for _, contract := range contracts {
			versions := byID[contract.ID]
			if versions == nil {
				versions = make(map[string]struct{})
				byID[contract.ID] = versions
			}
			for _, version := range contract.Versions {
				versions[version] = struct{}{}
			}
		}
		result[release] = byID
	}
	return result, nil
}

func validateReleaseBinding(contractID string, versions, releases []string, contracts map[string]map[string]map[string]struct{}) error {
	for _, release := range releases {
		available := contracts[release][contractID]
		matched := false
		for _, version := range versions {
			if _, ok := available[version]; ok {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("contract %q has no represented version in release %q", contractID, release)
		}
	}
	return nil
}

func validateContractBinding(contractID string, versions []string, contracts map[string]map[string]struct{}) error {
	available, ok := contracts[contractID]
	if !ok {
		return fmt.Errorf("unknown contract %q", contractID)
	}
	if len(versions) == 0 {
		return fmt.Errorf("contract %q has no versions", contractID)
	}
	seen := make(map[string]struct{}, len(versions))
	for _, version := range versions {
		if _, ok := available[version]; !ok {
			return fmt.Errorf("contract %q does not declare version %q", contractID, version)
		}
		if _, duplicate := seen[version]; duplicate {
			return fmt.Errorf("contract %q repeats version %q", contractID, version)
		}
		seen[version] = struct{}{}
	}
	return nil
}

func validateReleases(releases []string) error {
	if len(releases) == 0 {
		return errors.New("release set is empty")
	}
	order := map[string]int{specpin.ReleaseV043: 0, specpin.ReleaseV050: 1}
	previous := -1
	for _, release := range releases {
		position, ok := order[release]
		if !ok {
			return fmt.Errorf("unsupported release %q", release)
		}
		if position <= previous {
			return errors.New("release set is duplicated or out of order")
		}
		previous = position
	}
	return nil
}

func uniqueNames(label string, names []string, pattern *regexp.Regexp) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		if !pattern.MatchString(name) {
			return nil, invalid("%s has invalid name %q", label, name)
		}
		if _, duplicate := result[name]; duplicate {
			return nil, invalid("%s repeats %q", label, name)
		}
		result[name] = struct{}{}
	}
	return result, nil
}

func expandOperations(families []operationFamily) []expandedOperation {
	var result []expandedOperation
	for _, family := range families {
		isolated := make(map[string]struct{}, len(family.IsolatedOutputs))
		for _, name := range family.IsolatedOutputs {
			isolated[name] = struct{}{}
		}
		mutations := make(map[string]mutationGroup)
		for _, group := range family.MutationGroups {
			for _, name := range group.Operations {
				mutations[name] = group
			}
		}
		for _, name := range family.Operations {
			effect := "EffectNoDurableMutation"
			var idempotency string
			var recovery []string
			if _, ok := isolated[name]; ok {
				effect = "EffectIsolatedOutput"
			}
			if mutation, ok := mutations[name]; ok {
				effect = "EffectDurableMutation"
				idempotency = mutation.IdempotencyKey
				recovery = mutation.RecoveryEvidence
			}
			result = append(result, expandedOperation{
				Family: family.Family, Name: name, ContractID: family.ContractID,
				ContractVersions: family.ContractVersions, Effect: effect,
				IdempotencyKey: idempotency, RecoveryEvidence: recovery,
				NormativeSection: family.NormativeSection,
				FixtureFamilies:  family.FixtureFamilies, Releases: family.Releases,
			})
		}
	}
	return result
}

func expandCapabilities(families []capabilityFamily) []expandedCapability {
	var result []expandedCapability
	for _, family := range families {
		for _, name := range family.Capabilities {
			result = append(result, expandedCapability{
				Family: family.Family, Name: name, ContractID: family.ContractID,
				ContractVersions: family.ContractVersions, NormativeSection: family.NormativeSection,
				FixtureFamilies: family.FixtureFamilies, Releases: family.Releases,
			})
		}
	}
	return result
}

func expandEvents(families []eventFamily) []expandedEvent {
	var result []expandedEvent
	for _, family := range families {
		for _, group := range family.Groups {
			for _, name := range group.Events {
				result = append(result, expandedEvent{
					Family: family.Family, Name: name, ContractID: family.ContractID,
					ContractVersions: group.ContractVersions, NormativeSection: family.NormativeSection,
					FixtureFamilies: family.FixtureFamilies, Releases: family.Releases,
				})
			}
		}
	}
	return result
}

func expandErrors(value errorCatalog) []expandedError {
	index := make(map[string]int, len(value.ContractVersions))
	for position, version := range value.ContractVersions {
		index[version] = position
	}
	var result []expandedError
	for _, group := range value.Groups {
		versions := append([]string(nil), value.ContractVersions[index[group.IntroducedVersion]:]...)
		for _, mapping := range group.Mappings {
			for _, code := range mapping.Codes {
				result = append(result, expandedError{
					Code: code, ExitCode: mapping.ExitCode, ContractID: value.ContractID,
					ContractVersions: versions, NormativeSection: value.NormativeSection,
					FixtureFamilies: value.FixtureFamilies, Releases: group.Releases,
				})
			}
		}
	}
	return result
}

func writeSource(output *bytes.Buffer, value metadata) {
	output.WriteString("\tSource: Source{\n")
	fmt.Fprintf(output, "\t\tRepository: %q,\n", value.Source.Repository)
	fmt.Fprintf(output, "\t\tRelease: Release(%q),\n", value.Source.Release)
	fmt.Fprintf(output, "\t\tCommit: %q,\n", value.Source.Commit)
	fmt.Fprintf(output, "\t\tDocumentPath: %q,\n", value.Source.DocumentPath)
	fmt.Fprintf(output, "\t\tDocumentSHA256: %q,\n", value.Source.DocumentSHA256)
	fmt.Fprintf(output, "\t\tNormativeScope: %s,\n", stringSliceLiteral(value.NormativeScope))
	output.WriteString("\t},\n")
}

func writeContracts(output *bytes.Buffer, manifest specpin.Manifest) {
	output.WriteString("\tContracts: map[Release][]Contract{\n")
	for _, release := range []string{specpin.ReleaseV043, specpin.ReleaseV050} {
		contracts, err := manifest.ContractsForRelease(release)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(output, "\t\tRelease(%q): {\n", release)
		for _, contract := range contracts {
			fmt.Fprintf(output, "\t\t\t{Name: %q, ID: %q, Versions: %s},\n", contract.Name, contract.ID, stringSliceLiteral(contract.Versions))
		}
		output.WriteString("\t\t},\n")
	}
	output.WriteString("\t},\n")
}

func writeOperations(output *bytes.Buffer, values []expandedOperation) {
	output.WriteString("\tOperations: []scopedOperation{\n")
	for _, value := range values {
		fmt.Fprintf(output, "\t\t{Definition: Operation{Family: %q, Name: %q, ContractID: %q, ContractVersions: %s, Effect: %s, IdempotencyKey: %q, RecoveryEvidence: %s, NormativeSection: %q, FixtureFamilies: %s}, Releases: %s},\n",
			value.Family, value.Name, value.ContractID, stringSliceLiteral(value.ContractVersions), value.Effect,
			value.IdempotencyKey, stringSliceLiteral(value.RecoveryEvidence), value.NormativeSection,
			stringSliceLiteral(value.FixtureFamilies), releaseSliceLiteral(value.Releases))
	}
	output.WriteString("\t},\n")
}

func writeCapabilities(output *bytes.Buffer, values []expandedCapability) {
	output.WriteString("\tCapabilities: []scopedCapability{\n")
	for _, value := range values {
		fmt.Fprintf(output, "\t\t{Definition: Capability{Family: %q, Name: %q, ContractID: %q, ContractVersions: %s, NormativeSection: %q, FixtureFamilies: %s}, Releases: %s},\n",
			value.Family, value.Name, value.ContractID, stringSliceLiteral(value.ContractVersions), value.NormativeSection,
			stringSliceLiteral(value.FixtureFamilies), releaseSliceLiteral(value.Releases))
	}
	output.WriteString("\t},\n")
}

func writeEvents(output *bytes.Buffer, values []expandedEvent) {
	output.WriteString("\tEvents: []scopedEvent{\n")
	for _, value := range values {
		fmt.Fprintf(output, "\t\t{Definition: Event{Family: %q, Name: %q, ContractID: %q, ContractVersions: %s, NormativeSection: %q, FixtureFamilies: %s}, Releases: %s},\n",
			value.Family, value.Name, value.ContractID, stringSliceLiteral(value.ContractVersions), value.NormativeSection,
			stringSliceLiteral(value.FixtureFamilies), releaseSliceLiteral(value.Releases))
	}
	output.WriteString("\t},\n")
}

func writeErrors(output *bytes.Buffer, values []expandedError) {
	output.WriteString("\tErrors: []scopedError{\n")
	for _, value := range values {
		fmt.Fprintf(output, "\t\t{Definition: ErrorCode{Code: %q, ExitCode: %d, ContractID: %q, ContractVersions: %s, NormativeSection: %q, FixtureFamilies: %s}, Releases: %s},\n",
			value.Code, value.ExitCode, value.ContractID, stringSliceLiteral(value.ContractVersions), value.NormativeSection,
			stringSliceLiteral(value.FixtureFamilies), releaseSliceLiteral(value.Releases))
	}
	output.WriteString("\t},\n")
}

func stringSliceLiteral(values []string) string {
	if len(values) == 0 {
		return "nil"
	}
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = strconv.Quote(value)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}

func releaseSliceLiteral(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = "Release(" + strconv.Quote(value) + ")"
	}
	return "[]Release{" + strings.Join(quoted, ", ") + "}"
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

func hasEmpty(values []string) bool {
	for _, value := range values {
		if value == "" {
			return true
		}
	}
	return false
}

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidMetadata, fmt.Sprintf(format, arguments...))
}
