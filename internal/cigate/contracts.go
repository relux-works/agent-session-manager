package cigate

import (
	"fmt"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/specpin"
)

// ContractVersions is one pinned contract with its exact ordered versions.
type ContractVersions struct {
	Name     string
	ID       string
	Versions []string
}

// ReleaseContracts binds one release to its full derived contract set.
type ReleaseContracts struct {
	Release   string
	Contracts []ContractVersions
}

// PinnedReleases returns the releases both authorities represent, after
// requiring the two packages to agree on the release roots themselves. It
// lists no versions: every version below these roots is derived, never
// retyped.
//
// The current root is the adopted v0.6.0 source and the historical root is
// v0.4.3. The v0.5.0 registry is a derived projection of the adopted lock,
// not a root: its agreement is checked by the catalog projection tests and
// the traceability legacy check rather than by this gate.
func PinnedReleases() ([]string, error) {
	return checkReleaseRoots(specpin.ReleaseV060, specpin.ReleaseV043,
		string(catalog.ReleaseV060), string(catalog.ReleaseV043))
}

// checkReleaseRoots is the testable core of PinnedReleases: production passes
// the two packages' release constants, tests pass synthetic roots. Either
// side renaming a release without the other is drift, not a wider registry,
// and is refused.
func checkReleaseRoots(pinCurrent, pinHistorical, catalogCurrent, catalogHistorical string) ([]string, error) {
	if pinCurrent != catalogCurrent {
		return nil, fmt.Errorf("cigate: current release root drift: specpin %q catalog %q",
			pinCurrent, catalogCurrent)
	}
	if pinHistorical != catalogHistorical {
		return nil, fmt.Errorf("cigate: historical release root drift: specpin %q catalog %q",
			pinHistorical, catalogHistorical)
	}
	if pinCurrent == "" || pinHistorical == "" || pinCurrent == pinHistorical {
		return nil, fmt.Errorf("cigate: release roots are empty or indistinct")
	}
	return []string{pinCurrent, pinHistorical}, nil
}

// PinContractSets derives every pinned (contract, version) set from the
// embedded specification lock, keyed by release. The historical projection
// comes from ContractsForRelease, so absent contracts and version overrides
// are the lock's own, not a retyped copy.
func PinContractSets() (map[string][]ContractVersions, error) {
	manifest, err := specpin.CurrentV060()
	if err != nil {
		return nil, fmt.Errorf("cigate: load pinned source: %w", err)
	}
	return pinContractSets(manifest)
}

// pinContractSets is the testable core over an explicit manifest: production
// passes the embedded one, tests pass synthetic ones.
func pinContractSets(manifest specpin.Manifest) (map[string][]ContractVersions, error) {
	releases, err := PinnedReleases()
	if err != nil {
		return nil, err
	}
	sets := make(map[string][]ContractVersions, len(releases))
	for _, release := range releases {
		contracts, err := manifest.ContractsForRelease(release)
		if err != nil {
			return nil, fmt.Errorf("cigate: derive pinned %s contracts: %w", release, err)
		}
		sets[release] = fromPinContracts(contracts)
	}
	return sets, nil
}

// CatalogContractSets derives every (contract, version) set from the
// generated catalog projections, keyed by release.
func CatalogContractSets() (map[string][]ContractVersions, error) {
	releases, err := PinnedReleases()
	if err != nil {
		return nil, err
	}
	sets := make(map[string][]ContractVersions, len(releases))
	for _, release := range releases {
		projected, err := catalog.ForRelease(catalog.Release(release))
		if err != nil {
			return nil, fmt.Errorf("cigate: derive catalog %s contracts: %w", release, err)
		}
		converted := make([]ContractVersions, 0, len(projected.Contracts))
		for _, contract := range projected.Contracts {
			converted = append(converted, ContractVersions{
				Name:     contract.Name,
				ID:       string(contract.ID),
				Versions: append([]string(nil), contract.Versions...),
			})
		}
		sets[release] = converted
	}
	return sets, nil
}

// VerifyContractPreservation requires the pinned lock and the generated
// catalog to agree exactly on every release: same contracts, same
// identifiers, same ordered versions. A version either side cannot account
// for is refused with its name, not averaged away. It reports the first
// disagreement; CI reruns are cheap and deterministic.
func VerifyContractPreservation() error {
	pinned, err := PinContractSets()
	if err != nil {
		return err
	}
	projected, err := CatalogContractSets()
	if err != nil {
		return err
	}
	return checkAgreement(pinned, projected)
}

// checkAgreement is the testable core over explicit sets: production passes
// the derived ones, tests pass synthetic ones.
func checkAgreement(pinned, projected map[string][]ContractVersions) error {
	releases, err := PinnedReleases()
	if err != nil {
		return err
	}
	for _, release := range releases {
		want, ok := pinned[release]
		if !ok {
			return fmt.Errorf("cigate: pinned authority accounts for no %s set", release)
		}
		got, ok := projected[release]
		if !ok {
			return fmt.Errorf("cigate: catalog authority accounts for no %s set", release)
		}
		if err := checkReleaseSet(release, want, got); err != nil {
			return err
		}
	}
	return nil
}

func checkReleaseSet(release string, pinned, projected []ContractVersions) error {
	want := indexContracts(pinned)
	got := indexContracts(projected)
	for key, wantContract := range want {
		gotContract, ok := got[key]
		if !ok {
			return fmt.Errorf("cigate: %s contract %q (%s) is pinned but absent from the catalog",
				release, wantContract.Name, wantContract.ID)
		}
		if len(wantContract.Versions) != len(gotContract.Versions) {
			return fmt.Errorf("cigate: %s contract %q (%s) pins %d versions, catalog carries %d",
				release, wantContract.Name, wantContract.ID,
				len(wantContract.Versions), len(gotContract.Versions))
		}
		for index, version := range wantContract.Versions {
			if gotContract.Versions[index] != version {
				return fmt.Errorf("cigate: %s contract %q (%s) version %d is pinned %q, catalog carries %q",
					release, wantContract.Name, wantContract.ID,
					index, version, gotContract.Versions[index])
			}
		}
	}
	for key, gotContract := range got {
		if _, ok := want[key]; !ok {
			return fmt.Errorf("cigate: %s contract %q (%s) is in the catalog but no pin accounts for it",
				release, gotContract.Name, gotContract.ID)
		}
	}
	return nil
}

func indexContracts(contracts []ContractVersions) map[string]ContractVersions {
	indexed := make(map[string]ContractVersions, len(contracts))
	for _, contract := range contracts {
		indexed[contract.Name+"\x00"+contract.ID] = contract
	}
	return indexed
}

func fromPinContracts(contracts []specpin.ContractPin) []ContractVersions {
	converted := make([]ContractVersions, 0, len(contracts))
	for _, contract := range contracts {
		converted = append(converted, ContractVersions{
			Name:     contract.Name,
			ID:       contract.ID,
			Versions: append([]string(nil), contract.Versions...),
		})
	}
	return converted
}

// SortedReleases reports the release keys of a derived set in sorted order,
// so command output is deterministic.
func SortedReleases(sets map[string][]ContractVersions) []string {
	releases := make([]string, 0, len(sets))
	for release := range sets {
		releases = append(releases, release)
	}
	sort.Strings(releases)
	return releases
}
