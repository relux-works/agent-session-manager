package cigate

import (
	"strings"
	"testing"
)

func TestPinnedReleasesAgreeAcrossAuthorities(t *testing.T) {
	t.Parallel()
	releases, err := PinnedReleases()
	if err != nil {
		t.Fatalf("PinnedReleases() error = %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("PinnedReleases() = %v, want the current and historical roots", releases)
	}
	if releases[0] == releases[1] {
		t.Fatalf("PinnedReleases() roots are indistinct: %v", releases)
	}
}

func TestCheckReleaseRootsRefusesDrift(t *testing.T) {
	t.Parallel()
	if _, err := checkReleaseRoots("v0.5.0", "v0.4.3", "v0.5.0", "v0.4.3"); err != nil {
		t.Fatalf("agreeing roots refused: %v", err)
	}
	for _, roots := range [][4]string{
		{"v0.5.0", "v0.4.3", "v0.5.1", "v0.4.3"},
		{"v0.5.0", "v0.4.3", "v0.5.0", "v0.4.4"},
		{"", "v0.4.3", "", "v0.4.3"},
		{"v0.5.0", "v0.5.0", "v0.5.0", "v0.5.0"},
	} {
		if _, err := checkReleaseRoots(roots[0], roots[1], roots[2], roots[3]); err == nil {
			t.Errorf("checkReleaseRoots(%q) admitted drift", roots)
		}
	}
}

func TestVerifyContractPreservationLive(t *testing.T) {
	t.Parallel()
	if err := VerifyContractPreservation(); err != nil {
		t.Fatalf("VerifyContractPreservation() error = %v", err)
	}
}

// TestDerivedSetsAreComplete pins the derivation shape without retyping any
// version: every release both authorities know carries a non-empty contract
// set, and every contract carries an identity and at least one version.
func TestDerivedSetsAreComplete(t *testing.T) {
	t.Parallel()
	pinned, err := PinContractSets()
	if err != nil {
		t.Fatalf("PinContractSets() error = %v", err)
	}
	projected, err := CatalogContractSets()
	if err != nil {
		t.Fatalf("CatalogContractSets() error = %v", err)
	}
	releases, err := PinnedReleases()
	if err != nil {
		t.Fatalf("PinnedReleases() error = %v", err)
	}
	for _, release := range releases {
		for authority, sets := range map[string]map[string][]ContractVersions{
			"pin": pinned, "catalog": projected,
		} {
			contracts, ok := sets[release]
			if !ok || len(contracts) == 0 {
				t.Fatalf("%s authority accounts for no %s contracts", authority, release)
			}
			for _, contract := range contracts {
				if contract.Name == "" || contract.ID == "" || len(contract.Versions) == 0 {
					t.Errorf("%s %s contract %+v is incomplete", authority, release, contract)
				}
			}
		}
	}
}

// syntheticSets builds agreeing sets keyed by the live release roots, so the
// fixture follows a deliberate root change instead of masking it.
func syntheticSets(t *testing.T) (current, historical string, pinned, projected map[string][]ContractVersions) {
	t.Helper()
	releases, err := PinnedReleases()
	if err != nil {
		t.Fatalf("PinnedReleases() error = %v", err)
	}
	current, historical = releases[0], releases[1]
	pinned = map[string][]ContractVersions{
		current: {
			{Name: "Session Record", ID: "urn:ax:schema:session", Versions: []string{"1.0.0", "2.0.0"}},
			{Name: "Mesh RPC", ID: "urn:ax:protocol:rpc", Versions: []string{"2.0.0"}},
		},
		historical: {
			{Name: "Mesh RPC", ID: "urn:ax:protocol:rpc", Versions: []string{"2.0.0"}},
		},
	}
	projected = map[string][]ContractVersions{
		current: {
			{Name: "Session Record", ID: "urn:ax:schema:session", Versions: []string{"1.0.0", "2.0.0"}},
			{Name: "Mesh RPC", ID: "urn:ax:protocol:rpc", Versions: []string{"2.0.0"}},
		},
		historical: {
			{Name: "Mesh RPC", ID: "urn:ax:protocol:rpc", Versions: []string{"2.0.0"}},
		},
	}
	return current, historical, pinned, projected
}

// TestCheckAgreementAdmitsIdenticalSets is the positive arm every refusal row
// below mirrors: the gate must admit exact agreement, or the refusals prove
// nothing.
func TestCheckAgreementAdmitsIdenticalSets(t *testing.T) {
	t.Parallel()
	_, _, pinned, projected := syntheticSets(t)
	if err := checkAgreement(pinned, projected); err != nil {
		t.Fatalf("checkAgreement() error = %v", err)
	}
}

func TestCheckAgreementRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(current, historical string, pinned, projected map[string][]ContractVersions)
		want   func(current, historical string) string
	}{
		{
			name: "contract missing from catalog",
			mutate: func(current, _ string, _, projected map[string][]ContractVersions) {
				projected[current] = projected[current][:1]
			},
			want: func(_, _ string) string { return "absent from the catalog" },
		},
		{
			name: "contract unaccounted by pin",
			mutate: func(current, _ string, pinned, _ map[string][]ContractVersions) {
				pinned[current] = pinned[current][:1]
			},
			want: func(_, _ string) string { return "no pin accounts for it" },
		},
		{
			// NARROWING row: the catalog admits exactly one version fewer.
			// A gate that compared only contract presence would admit this.
			name: "single version dropped",
			mutate: func(current, _ string, _, projected map[string][]ContractVersions) {
				projected[current][0].Versions = []string{"1.0.0"}
			},
			want: func(_, _ string) string { return "pins 2 versions, catalog carries 1" },
		},
		{
			name: "single version added",
			mutate: func(current, _ string, _, projected map[string][]ContractVersions) {
				projected[current][1].Versions = []string{"2.0.0", "3.0.0"}
			},
			want: func(_, _ string) string { return "pins 1 versions, catalog carries 2" },
		},
		{
			// NARROWING row: same versions, different order. A gate that
			// compared version sets without order would admit this, and
			// version order selects the negotiated baseline.
			name: "version order changed",
			mutate: func(current, _ string, _, projected map[string][]ContractVersions) {
				projected[current][0].Versions = []string{"2.0.0", "1.0.0"}
			},
			want: func(_, _ string) string { return `version 0 is pinned "1.0.0", catalog carries "2.0.0"` },
		},
		{
			name: "release missing from catalog",
			mutate: func(_, historical string, _, projected map[string][]ContractVersions) {
				delete(projected, historical)
			},
			want: func(_, historical string) string { return "accounts for no " + historical + " set" },
		},
		{
			name: "release missing from pin",
			mutate: func(_, historical string, pinned, _ map[string][]ContractVersions) {
				delete(pinned, historical)
			},
			want: func(_, historical string) string { return "accounts for no " + historical + " set" },
		},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			t.Parallel()
			current, historical, pinned, projected := syntheticSets(t)
			kase.mutate(current, historical, pinned, projected)
			err := checkAgreement(pinned, projected)
			if err == nil {
				t.Fatalf("checkAgreement() admitted %s", kase.name)
			}
			want := kase.want(current, historical)
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("checkAgreement() error = %q, want substring %q", err, want)
			}
		})
	}
}
