package cloneplanning

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// This file drives SelectStrategy and SelectProfile through the
// production entries against literal oracles. SelectStrategy
// enumerates all sixteen signal combinations; SelectProfile sweeps
// the five profiles plus invalid probes and asserts the branch.

func TestSelectStrategyOracle(t *testing.T) {
	// want[same][official][continuation][archive], retyped from
	// the R-SEL-S derivation: archive wins, then continuation,
	// then the pair/capability arm.
	want := map[[4]bool]string{}
	for _, same := range []bool{false, true} {
		for _, official := range []bool{false, true} {
			for _, continuation := range []bool{false, true} {
				for _, archive := range []bool{false, true} {
					key := [4]bool{same, official, continuation, archive}
					switch {
					case archive:
						want[key] = "archive_only"
					case continuation:
						want[key] = "continuation_context"
					case !same && official:
						want[key] = "target_official_import"
					case !same:
						want[key] = "target_native_writer"
					default:
						want[key] = "same_environment_native_rewrite"
					}
				}
			}
		}
	}
	if len(want) != 16 {
		t.Fatalf("strategy oracle covers %d cells, want 16", len(want))
	}
	for key, expected := range want {
		got, err := SelectStrategy(key[0], key[1], key[2], key[3])
		if err != nil {
			t.Fatalf("SelectStrategy%v refused: %v", key, err)
		}
		if got != expected {
			t.Fatalf("SelectStrategy%v = %q, want %q", key, got, expected)
		}
		if !clonefidelity.ValidStrategy(got) {
			t.Fatalf("SelectStrategy%v = %q outside the owner vocabulary", key, got)
		}
	}
}

func TestSelectProfileOracle(t *testing.T) {
	// R-SEL-P: archive_only routes the archive branch (no plan);
	// the four plan profiles route target; anything else refuses.
	for _, profile := range []string{"strict_exact", "maximal_safe", "compact", "messages_only"} {
		branch, got, err := SelectProfile(profile)
		if err != nil {
			t.Fatalf("SelectProfile(%q) refused: %v", profile, err)
		}
		if branch != BranchTarget || got != profile {
			t.Fatalf("SelectProfile(%q) = (%q, %q), want (target, %q)", profile, branch, got, profile)
		}
		if string(branch) != "target" {
			t.Fatalf("target branch spells %q, want target", branch)
		}
	}
	branch, got, err := SelectProfile("archive_only")
	if err != nil {
		t.Fatalf("SelectProfile(archive_only) refused: %v", err)
	}
	if branch != BranchArchive || got != "archive_only" {
		t.Fatalf("SelectProfile(archive_only) = (%q, %q), want (archive, archive_only)", branch, got)
	}
	if string(branch) != "archive" {
		t.Fatalf("archive branch spells %q, want archive", branch)
	}
	// "bogus_profile" is the narrowing probe
	// (N-select-profile-unknown admits exactly it).
	for _, profile := range []string{"", "bogus_profile", "ARCHIVE_ONLY", "maximal", "target", "archive"} {
		if _, _, err := SelectProfile(profile); err == nil {
			t.Fatalf("SelectProfile(%q) admitted", profile)
		}
	}
}

func TestSelectionAgreement(t *testing.T) {
	// Supplement to the literal oracles: every selected strategy
	// and profile stays inside the owner vocabularies, and the
	// target branch carries exactly the four plan profiles.
	seen := map[string]bool{}
	for same := 0; same < 2; same++ {
		for official := 0; official < 2; official++ {
			for continuation := 0; continuation < 2; continuation++ {
				for archive := 0; archive < 2; archive++ {
					got, err := SelectStrategy(same == 1, official == 1, continuation == 1, archive == 1)
					if err != nil {
						t.Fatal(err)
					}
					seen[got] = true
				}
			}
		}
	}
	for _, strategy := range clonefidelity.Strategies() {
		if !seen[strategy] {
			t.Fatalf("strategy %q never selected", strategy)
		}
	}
	for _, profile := range clonefidelity.Profiles() {
		branch, _, err := SelectProfile(profile)
		if err != nil {
			t.Fatalf("SelectProfile(%q) refused: %v", profile, err)
		}
		if profile == "archive_only" {
			if branch != BranchArchive {
				t.Fatalf("archive_only routes %q, want archive", branch)
			}
			continue
		}
		if branch != BranchTarget {
			t.Fatalf("%q routes %q, want target", profile, branch)
		}
		if !cloneplan.ValidPlanProfile(profile) {
			t.Fatalf("%q routes target but is outside the plan profiles", profile)
		}
	}
}
