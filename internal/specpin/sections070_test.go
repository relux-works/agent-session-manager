package specpin_test

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

// TestSectionInventoryV070MatchesAdoptedHeadingDigest drives the production
// inventory entry point and pins the measured v0.7.0 heading digest: 170
// identifiers in document order — the v0.7.0 revision adds no numbered
// section, so the digest is measured equal to the v0.6.0 one — with history
// present and fabrication absent.
func TestSectionInventoryV070MatchesAdoptedHeadingDigest(t *testing.T) {
	inventory := specpin.SectionInventoryV070()
	if len(inventory) != 170 {
		t.Fatalf("v0.7.0 section inventory has %d identifiers, want 170", len(inventory))
	}
	digest := sha256.Sum256([]byte(strings.Join(inventory, "\n") + "\n"))
	if got := hex.EncodeToString(digest[:]); got != specpin.SectionInventorySHA256V070 {
		t.Fatalf("v0.7.0 section inventory digest = %s, want %s", got, specpin.SectionInventorySHA256V070)
	}
	for _, section := range []string{"6.6", "11.10", "14.7", "15.3"} {
		if !specpin.IsSectionV070(section) {
			t.Errorf("adopted v0.7.0 Section %s is absent from the immutable inventory", section)
		}
	}
	if !specpin.IsSectionV070("10.1") {
		t.Error("historical Section 10.1 is absent from the v0.7.0 inventory")
	}
	if specpin.IsSectionV070("10.999") {
		t.Error("nonexistent Section 10.999 is present in the immutable inventory")
	}
	if specpin.IsSectionV070("A.12") {
		t.Error("appendix subsection A.12 is present in the inventory; appendix subsections are excluded by the rule")
	}
	if specpin.IsSectionV070("appendix-e") {
		t.Error("unshipped appendix-e is present in the immutable inventory")
	}

	inventory[0] = "forged"
	if !specpin.IsSectionV070("1") {
		t.Fatal("caller mutation leaked into the immutable section inventory")
	}
}

// TestSectionInventoryV070PreservesV060History proves adoption only carries
// history forward: the v0.7.0 inventory equals the v0.6.0 inventory
// identifier for identifier, in the same order, because the revision adds no
// numbered section — no historical section was renamed, dropped, or moved.
func TestSectionInventoryV070PreservesV060History(t *testing.T) {
	previous := specpin.SectionInventoryV060()
	current := specpin.SectionInventoryV070()
	if !reflect.DeepEqual(current, previous) {
		t.Fatal("v0.7.0 inventory differs from the v0.6.0 inventory; the revision adds no numbered section")
	}
	position := 0
	for _, section := range current {
		if position < len(previous) && section == previous[position] {
			position++
		}
	}
	if position != len(previous) {
		t.Fatalf("v0.6.0 inventory is not an ordered subsequence of the v0.7.0 inventory: matched %d of %d",
			position, len(previous))
	}
}
