package specpin_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

// TestSectionInventoryV060MatchesAdoptedHeadingDigest drives the production
// inventory entry point and pins the measured v0.6.0 heading digest: 170
// identifiers in document order, the thirteen selector/authentication
// additions present, history present, fabrication absent.
func TestSectionInventoryV060MatchesAdoptedHeadingDigest(t *testing.T) {
	inventory := specpin.SectionInventoryV060()
	if len(inventory) != 170 {
		t.Fatalf("v0.6.0 section inventory has %d identifiers, want 170", len(inventory))
	}
	digest := sha256.Sum256([]byte(strings.Join(inventory, "\n") + "\n"))
	if got := hex.EncodeToString(digest[:]); got != specpin.SectionInventorySHA256V060 {
		t.Fatalf("v0.6.0 section inventory digest = %s, want %s", got, specpin.SectionInventorySHA256V060)
	}
	for _, section := range []string{"6.6", "11.10", "11.10.5", "14.7", "14.7.5"} {
		if !specpin.IsSectionV060(section) {
			t.Errorf("adopted v0.6.0 Section %s is absent from the immutable inventory", section)
		}
	}
	if !specpin.IsSectionV060("10.1") {
		t.Error("historical Section 10.1 is absent from the v0.6.0 inventory")
	}
	if specpin.IsSectionV060("10.999") {
		t.Error("nonexistent Section 10.999 is present in the immutable inventory")
	}
	if specpin.IsSectionV060("14.7.6") {
		t.Error("unshipped Section 14.7.6 is present in the immutable inventory")
	}

	inventory[0] = "forged"
	if !specpin.IsSectionV060("1") {
		t.Fatal("caller mutation leaked into the immutable section inventory")
	}
}

// TestSectionInventoryV060PreservesV050History proves adoption only adds:
// every v0.5.0 identifier appears in the v0.6.0 inventory in the same
// relative order, so no historical section was renamed, dropped, or moved.
func TestSectionInventoryV060PreservesV050History(t *testing.T) {
	previous := specpin.SectionInventoryV050()
	current := specpin.SectionInventoryV060()
	position := 0
	for _, section := range current {
		if position < len(previous) && section == previous[position] {
			position++
		}
	}
	if position != len(previous) {
		t.Fatalf("v0.5.0 inventory is not an ordered subsequence of the v0.6.0 inventory: matched %d of %d",
			position, len(previous))
	}
}
