package sessprofile

import (
	"errors"
	"strings"
	"testing"
)

const (
	fixtureSourceE1 = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fixtureSourceE0 = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestResolveCreationProfile(t *testing.T) {
	for _, profile := range []string{ProfileStandard, ProfileYOLO} {
		got, err := ResolveCreationProfile(profile)
		if err != nil {
			t.Fatalf("ResolveCreationProfile(%q) error = %v", profile, err)
		}
		if got != profile {
			t.Fatalf("ResolveCreationProfile(%q) = %q", profile, got)
		}
	}
	// The pinned specification states no absent-flag default, so
	// the CLI leaf passes an explicit value and anything else —
	// including empty — refuses rather than guessing one.
	for _, flag := range []string{"", "YOLO", "STANDARD", "turbo", "standard "} {
		if _, err := ResolveCreationProfile(flag); !errors.Is(err, ErrInvalidProfile) {
			t.Fatalf("ResolveCreationProfile(%q) error = %v, want invalid execution profile", flag, err)
		}
	}
}

func TestBundlePairProjection(t *testing.T) {
	want := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	got := BundlePair(want)
	mustPairEqual(t, got, want, "BundlePair(copy)")
	if err := CheckBundlePair(ProfileYOLO, fixtureSourceE1, true, want); err != nil {
		t.Fatalf("CheckBundlePair(CHANGED-POS) error = %v", err)
	}
	null := Pair{Profile: ProfileStandard}
	if err := CheckBundlePair(ProfileStandard, "", false, null); err != nil {
		t.Fatalf("CheckBundlePair(null source) error = %v", err)
	}
	cases := []struct {
		name            string
		profile, source string
		hasSource       bool
	}{
		{"N1 stale creation value", ProfileStandard, fixtureSourceE1, true},
		{"N1 stale value and null", ProfileStandard, "", false},
		{"N2 null source", ProfileYOLO, "", false},
		{"N3 losing source", ProfileYOLO, fixtureSourceE0, true},
		{"N3 unknown source", ProfileYOLO, zeroDigest, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckBundlePair(tc.profile, tc.source, tc.hasSource, want); !errors.Is(err, ErrIntegrity) {
				t.Fatalf("CheckBundlePair(%s) error = %v, want integrity_failure", tc.name, err)
			}
		})
	}
	if err := CheckBundlePair(ProfileStandard, fixtureSourceE1, true, null); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckBundlePair(spurious source) error = %v, want integrity_failure", err)
	}
	// The source-presence direction pins its arm by message: a
	// mutant that admits a missing source still refuses at the
	// source-equality arm with a degraded message, so the
	// sentinel alone cannot tell the arms apart.
	if err := CheckBundlePair(ProfileYOLO, "", false, want); !strings.Contains(err.Error(), "carries no profile source") {
		t.Fatalf("CheckBundlePair(N2) error = %v, want the no-source arm named", err)
	}
	if err := CheckBundlePair(ProfileStandard, fixtureSourceE1, true, null); !strings.Contains(err.Error(), "want no profile source") {
		t.Fatalf("CheckBundlePair(spurious source) error = %v, want the no-source arm named", err)
	}
}

func TestCheckLaunchPair(t *testing.T) {
	creation := Pair{Profile: ProfileStandard}
	if err := CheckLaunchPair(Pair{Profile: ProfileStandard}, creation); err != nil {
		t.Fatalf("CheckLaunchPair(first) error = %v", err)
	}
	if err := CheckLaunchPair(Pair{Profile: ProfileStandard, Source: fixtureSourceE1, HasSource: true}, creation); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckLaunchPair(first with source) error = %v, want integrity_failure", err)
	}
	later := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	if err := CheckLaunchPair(later, later); err != nil {
		t.Fatalf("CheckLaunchPair(later) error = %v", err)
	}
	if err := CheckLaunchPair(Pair{Profile: ProfileStandard}, later); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckLaunchPair(fallback to creation) error = %v, want integrity_failure", err)
	}
	if err := CheckLaunchPair(Pair{Profile: ProfileYOLO}, later); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckLaunchPair(missing source) error = %v, want integrity_failure", err)
	}
	if err := CheckLaunchPair(Pair{Profile: ProfileYOLO, Source: fixtureSourceE0, HasSource: true}, later); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckLaunchPair(non-newest source) error = %v, want integrity_failure", err)
	}
}

func TestCheckResumedPair(t *testing.T) {
	want := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	if err := CheckResumedPair(want, want); err != nil {
		t.Fatalf("CheckResumedPair error = %v", err)
	}
	if err := CheckResumedPair(Pair{Profile: ProfileStandard}, want); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckResumedPair(P0 after C1) error = %v, want integrity_failure", err)
	}
	if err := CheckResumedPair(Pair{Profile: ProfileYOLO}, want); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckResumedPair(missing E1) error = %v, want integrity_failure", err)
	}
	fresh := Pair{Profile: ProfileStandard}
	if err := CheckResumedPair(fresh, fresh); err != nil {
		t.Fatalf("CheckResumedPair(pre-change) error = %v", err)
	}
}

func TestForkProjectionAndCheck(t *testing.T) {
	source := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	if got := ForkProjection(source); got != ProfileYOLO {
		t.Fatalf("ForkProjection = %q, want yolo", got)
	}
	// The fork pair is the new-session authority (P1/null); the
	// source event survives only as provenance.
	if err := CheckForkPair(Pair{Profile: ProfileYOLO}, fixtureSourceE1, true, ProfileYOLO, fixtureSourceE1, true); err != nil {
		t.Fatalf("CheckForkPair error = %v", err)
	}
	if err := CheckForkPair(Pair{Profile: ProfileYOLO}, "", false, ProfileYOLO, "", false); err != nil {
		t.Fatalf("CheckForkPair(pre-change fork) error = %v", err)
	}
	cases := []struct {
		name                    string
		observed                Pair
		provenance              string
		hasProvenance           bool
		newCreation, wantSource string
		wantHasSource           bool
	}{
		{"pair carries a source", Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}, fixtureSourceE1, true, ProfileYOLO, fixtureSourceE1, true},
		{"pair uses creation P0", Pair{Profile: ProfileStandard}, fixtureSourceE1, true, ProfileYOLO, fixtureSourceE1, true},
		{"provenance dropped", Pair{Profile: ProfileYOLO}, "", false, ProfileYOLO, fixtureSourceE1, true},
		{"empty provenance string", Pair{Profile: ProfileYOLO}, "", true, ProfileYOLO, fixtureSourceE1, true},
		{"provenance invented", Pair{Profile: ProfileYOLO}, fixtureSourceE1, true, ProfileYOLO, "", false},
		{"provenance swapped", Pair{Profile: ProfileYOLO}, fixtureSourceE0, true, ProfileYOLO, fixtureSourceE1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckForkPair(tc.observed, tc.provenance, tc.hasProvenance, tc.newCreation, tc.wantSource, tc.wantHasSource); !errors.Is(err, ErrIntegrity) {
				t.Fatalf("CheckForkPair(%s) error = %v, want integrity_failure", tc.name, err)
			}
		})
	}
}

func TestCheckResumeRequestProfile(t *testing.T) {
	want := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	if err := CheckResumeRequestProfile(ProfileYOLO, want); err != nil {
		t.Fatalf("CheckResumeRequestProfile error = %v", err)
	}
	if err := CheckResumeRequestProfile(ProfileStandard, want); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckResumeRequestProfile(P0) error = %v, want integrity_failure", err)
	}
	if err := CheckResumeRequestProfile("turbo", want); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckResumeRequestProfile(turbo) error = %v, want integrity_failure", err)
	}
}

func TestFinalizePair(t *testing.T) {
	checkpoint := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	dormant, err := FinalizePair(ActivationDormant, checkpoint, "", false)
	if err != nil {
		t.Fatalf("FinalizePair(dormant) error = %v", err)
	}
	mustPairEqual(t, dormant, Pair{}, "FinalizePair(dormant nulls)")
	for _, activation := range []string{ActivationDirectResumed, ActivationTaskBoardResumed} {
		got, err := FinalizePair(activation, checkpoint, "", false)
		if err != nil {
			t.Fatalf("FinalizePair(%s) error = %v", activation, err)
		}
		mustPairEqual(t, got, checkpoint, "FinalizePair("+activation+")")
		forked, err := FinalizePair(activation, checkpoint, ProfileYOLO, true)
		if err != nil {
			t.Fatalf("FinalizePair(%s fork) error = %v", activation, err)
		}
		mustPairEqual(t, forked, Pair{Profile: ProfileYOLO}, "FinalizePair("+activation+" fork)")
	}
	if _, err := FinalizePair("turbo_resumed", checkpoint, "", false); !errors.Is(err, ErrDerivation) {
		t.Fatalf("FinalizePair(unknown) error = %v, want corrupt derivation input", err)
	}
	if _, err := FinalizePair(ActivationDormant, checkpoint, "", true); !errors.Is(err, ErrDerivation) {
		t.Fatalf("FinalizePair(dormant fork) error = %v, want corrupt derivation input", err)
	}
	if _, err := FinalizePair(ActivationDirectResumed, checkpoint, "turbo", true); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("FinalizePair(bad fork creation) error = %v, want invalid execution profile", err)
	}
}

func TestCheckFinalizeParams(t *testing.T) {
	want := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	if err := CheckFinalizeParams(ProfileYOLO, true, fixtureSourceE1, true, want); err != nil {
		t.Fatalf("CheckFinalizeParams error = %v", err)
	}
	if err := CheckFinalizeParams("", false, "", false, Pair{}); err != nil {
		t.Fatalf("CheckFinalizeParams(dormant) error = %v", err)
	}
	cases := []struct {
		name       string
		profile    string
		hasProfile bool
		source     string
		hasSource  bool
	}{
		{"profile diverged", ProfileStandard, true, fixtureSourceE1, true},
		{"profile dropped", "", false, fixtureSourceE1, true},
		{"source dropped", ProfileYOLO, true, "", false},
		{"source swapped", ProfileYOLO, true, fixtureSourceE0, true},
		{"dormant carries profile", ProfileYOLO, true, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := want
			if tc.name == "dormant carries profile" {
				want = Pair{}
			}
			if err := CheckFinalizeParams(tc.profile, tc.hasProfile, tc.source, tc.hasSource, want); !errors.Is(err, ErrIntegrity) {
				t.Fatalf("CheckFinalizeParams(%s) error = %v, want integrity_failure", tc.name, err)
			}
		})
	}
	if err := CheckFinalizeParams("", false, fixtureSourceE1, true, Pair{}); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckFinalizeParams(dormant carries source) error = %v, want integrity_failure", err)
	}
}

func TestCheckBridgeProfile(t *testing.T) {
	want := Pair{Profile: ProfileYOLO, Source: fixtureSourceE1, HasSource: true}
	if err := CheckBridgeProfile(ProfileYOLO, want); err != nil {
		t.Fatalf("CheckBridgeProfile error = %v", err)
	}
	if err := CheckBridgeProfile(ProfileStandard, want); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckBridgeProfile(P0) error = %v, want integrity_failure", err)
	}
}
