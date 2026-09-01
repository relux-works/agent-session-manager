// Package catalog exposes generated, typed vocabularies from the reviewed AX
// implementation metadata. Catalog membership defines a contract vocabulary;
// it never advertises runtime availability or implementation support.
package catalog

import (
	"errors"
	"fmt"
)

const (
	Format        = "ax-implementation-catalog"
	FormatVersion = 1

	ReleaseV043 Release = "v0.4.3"
	ReleaseV050 Release = "v0.5.0"
)

var ErrUnsupportedRelease = errors.New("unsupported catalog release")

// Release identifies a specification package release represented by the
// reviewed implementation metadata.
type Release string

// ContractID is a registered AX schema or protocol identifier.
type ContractID string

// Family scopes operation, capability, and event names that may otherwise
// overlap across independent contracts.
type Family string

// OperationName is one closed operation-registry member.
type OperationName string

// CapabilityName is one manifest/probe capability-vocabulary member. It does
// not imply that the capability is available.
type CapabilityName string

// EventName is one closed Session Event or required Observation Event member.
type EventName string

// ErrorName is one stable Structured Error code.
type ErrorName string

// Source binds every generated definition to the immutable normative source.
type Source struct {
	Repository     string
	Release        Release
	Commit         string
	DocumentPath   string
	DocumentSHA256 string
	NormativeScope []string
}

// Contract is one independently versioned schema or protocol contract.
type Contract struct {
	Name     string
	ID       ContractID
	Versions []string
}

// SelfIdentityContract defines the schema-directed JCS omit-self rule for an
// immutable logical object. Membership is reviewed metadata from the pinned
// specification; it does not imply that any wider schema behavior is
// implemented.
type SelfIdentityContract struct {
	ContractID         ContractID
	ContractVersions   []string
	SelfField          string
	DiscriminatorName  string
	DiscriminatorValue string
	NormativeSection   string
}

// OperationEffect classifies whether an operation mutates authoritative
// durable state. Isolated output is restricted to caller-created sinks and is
// not an authoritative state transition.
type OperationEffect string

const (
	// EffectNoDurableMutation makes no stronger read-only claim: an operation
	// may still plan work or control an ephemeral process.
	EffectNoDurableMutation OperationEffect = "no_durable_mutation"
	EffectIsolatedOutput    OperationEffect = "isolated_output"
	EffectDurableMutation   OperationEffect = "durable_mutation"
)

// Operation is a typed protocol/query operation definition. Durable mutations
// always carry both an exact idempotency scope and recovery evidence.
type Operation struct {
	Family           Family
	Name             OperationName
	ContractID       ContractID
	ContractVersions []string
	Effect           OperationEffect
	IdempotencyKey   string
	RecoveryEvidence []string
	NormativeSection string
	FixtureFamilies  []string
}

// Capability is a vocabulary definition only. Availability, status, enabled,
// and support claims deliberately do not exist in this type.
type Capability struct {
	Family           Family
	Name             CapabilityName
	ContractID       ContractID
	ContractVersions []string
	NormativeSection string
	FixtureFamilies  []string
}

// Event is a required event name and the exact contract versions that define
// it. Session Events and Observation Events remain separate families.
type Event struct {
	Family           Family
	Name             EventName
	ContractID       ContractID
	ContractVersions []string
	NormativeSection string
	FixtureFamilies  []string
}

// ErrorCode is one stable Structured Error code-to-exit mapping.
type ErrorCode struct {
	Code             ErrorName
	ExitCode         int
	ContractID       ContractID
	ContractVersions []string
	NormativeSection string
	FixtureFamilies  []string
}

// Catalog is an isolated release projection of the generated definitions.
type Catalog struct {
	Source         Source
	Release        Release
	MetadataSHA256 string
	Contracts      []Contract
	SelfIdentities []SelfIdentityContract
	Operations     []Operation
	Capabilities   []Capability
	Events         []Event
	Errors         []ErrorCode
}

type catalogDefinition struct {
	Source         Source
	MetadataSHA256 string
	Contracts      map[Release][]Contract
	SelfIdentities []scopedSelfIdentityContract
	Operations     []scopedOperation
	Capabilities   []scopedCapability
	Events         []scopedEvent
	Errors         []scopedError
}

type scopedSelfIdentityContract struct {
	Definition SelfIdentityContract
	Releases   []Release
}

type scopedOperation struct {
	Definition Operation
	Releases   []Release
}

type scopedCapability struct {
	Definition Capability
	Releases   []Release
}

type scopedEvent struct {
	Definition Event
	Releases   []Release
}

type scopedError struct {
	Definition ErrorCode
	Releases   []Release
}

//go:generate go run ./cmd/cataloggen -metadata catalog.v0.5.0.json -contracts ../specpin/v0.5.0.lock.json -output catalog_gen.go

// Current returns an isolated v0.5.0 catalog.
func Current() Catalog {
	catalog, err := ForRelease(ReleaseV050)
	if err != nil {
		panic(fmt.Sprintf("generated current catalog is invalid: %v", err))
	}
	return catalog
}

// ForRelease returns an isolated exact catalog projection for a represented
// release. It never coerces or guesses an unsupported release.
func ForRelease(release Release) (Catalog, error) {
	contracts, ok := generatedDefinition.Contracts[release]
	if !ok {
		return Catalog{}, fmt.Errorf("%w: %s", ErrUnsupportedRelease, release)
	}

	allowedVersions := make(map[ContractID]map[string]struct{}, len(contracts))
	for _, contract := range contracts {
		versions := allowedVersions[contract.ID]
		if versions == nil {
			versions = make(map[string]struct{}, len(contract.Versions))
			allowedVersions[contract.ID] = versions
		}
		for _, version := range contract.Versions {
			versions[version] = struct{}{}
		}
	}

	result := Catalog{
		Source:         cloneSource(generatedDefinition.Source),
		Release:        release,
		MetadataSHA256: generatedDefinition.MetadataSHA256,
		Contracts:      cloneContracts(contracts),
	}
	for _, scoped := range generatedDefinition.SelfIdentities {
		if !containsRelease(scoped.Releases, release) {
			continue
		}
		definition := cloneSelfIdentityContract(scoped.Definition)
		definition.ContractVersions = filterVersions(definition.ContractVersions, allowedVersions[definition.ContractID])
		if len(definition.ContractVersions) != 0 {
			result.SelfIdentities = append(result.SelfIdentities, definition)
		}
	}
	for _, scoped := range generatedDefinition.Operations {
		if !containsRelease(scoped.Releases, release) {
			continue
		}
		definition := cloneOperation(scoped.Definition)
		definition.ContractVersions = filterVersions(definition.ContractVersions, allowedVersions[definition.ContractID])
		if len(definition.ContractVersions) != 0 {
			result.Operations = append(result.Operations, definition)
		}
	}
	for _, scoped := range generatedDefinition.Capabilities {
		if !containsRelease(scoped.Releases, release) {
			continue
		}
		definition := cloneCapability(scoped.Definition)
		definition.ContractVersions = filterVersions(definition.ContractVersions, allowedVersions[definition.ContractID])
		if len(definition.ContractVersions) != 0 {
			result.Capabilities = append(result.Capabilities, definition)
		}
	}
	for _, scoped := range generatedDefinition.Events {
		if !containsRelease(scoped.Releases, release) {
			continue
		}
		definition := cloneEvent(scoped.Definition)
		definition.ContractVersions = filterVersions(definition.ContractVersions, allowedVersions[definition.ContractID])
		if len(definition.ContractVersions) != 0 {
			result.Events = append(result.Events, definition)
		}
	}
	for _, scoped := range generatedDefinition.Errors {
		if !containsRelease(scoped.Releases, release) {
			continue
		}
		definition := cloneError(scoped.Definition)
		definition.ContractVersions = filterVersions(definition.ContractVersions, allowedVersions[definition.ContractID])
		if len(definition.ContractVersions) != 0 {
			result.Errors = append(result.Errors, definition)
		}
	}
	return result, nil
}

func containsRelease(releases []Release, wanted Release) bool {
	for _, release := range releases {
		if release == wanted {
			return true
		}
	}
	return false
}

func filterVersions(versions []string, allowed map[string]struct{}) []string {
	filtered := make([]string, 0, len(versions))
	for _, version := range versions {
		if _, ok := allowed[version]; ok {
			filtered = append(filtered, version)
		}
	}
	return filtered
}

func cloneSource(source Source) Source {
	source.NormativeScope = append([]string(nil), source.NormativeScope...)
	return source
}

func cloneContracts(contracts []Contract) []Contract {
	result := make([]Contract, len(contracts))
	for index, contract := range contracts {
		contract.Versions = append([]string(nil), contract.Versions...)
		result[index] = contract
	}
	return result
}

func cloneSelfIdentityContract(contract SelfIdentityContract) SelfIdentityContract {
	contract.ContractVersions = append([]string(nil), contract.ContractVersions...)
	return contract
}

func cloneOperation(operation Operation) Operation {
	operation.ContractVersions = append([]string(nil), operation.ContractVersions...)
	operation.RecoveryEvidence = append([]string(nil), operation.RecoveryEvidence...)
	operation.FixtureFamilies = append([]string(nil), operation.FixtureFamilies...)
	return operation
}

func cloneCapability(capability Capability) Capability {
	capability.ContractVersions = append([]string(nil), capability.ContractVersions...)
	capability.FixtureFamilies = append([]string(nil), capability.FixtureFamilies...)
	return capability
}

func cloneEvent(event Event) Event {
	event.ContractVersions = append([]string(nil), event.ContractVersions...)
	event.FixtureFamilies = append([]string(nil), event.FixtureFamilies...)
	return event
}

func cloneError(item ErrorCode) ErrorCode {
	item.ContractVersions = append([]string(nil), item.ContractVersions...)
	item.FixtureFamilies = append([]string(nil), item.FixtureFamilies...)
	return item
}
