package provhost

import (
	"reflect"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provider"
)

// TestBuiltinsEqualProfileProviders pins the cross-package agreement
// between the two hand-written six-provider literals directly: the
// Section 7.1 built-in registry behind provider.Builtins and the
// Section 7.7 provider registry behind profileProviders. It reads no
// SPEC.md line, so the agreement no longer runs only transitively
// through the two derivations' different line windows (provider's
// TestSection71BuiltinRegistryIsDocumentOrder against Section 7.1,
// this package's TestProfileYOLOMappingIsDerivedFromSpec against the
// Section 7.7 table).
//
// Set agreement already has a direct pin:
// TestSixProviderSetMatchesDiscoveryRegistry balances the two sets.
// That pin is order-insensitive, so a reorder on exactly one side
// leaves the whole suite green — the remaining structural hole. This
// test closes it with an order-sensitive comparison: both orders are
// independently spec-pinned (Section 7.1 listed order is the discovery
// enumeration order, Section 7.7 table order is the profile registry
// order), so ordered equality holds today, and a seventh member, a
// rename, a removal, or a reorder on either side alone reddens here.
// A future spec bump that legitimately reorders one table must
// consciously revisit this agreement rather than drift through a
// set-only check.
func TestBuiltinsEqualProfileProviders(t *testing.T) {
	t.Parallel()

	got := provider.Builtins()
	if !reflect.DeepEqual(got, profileProviders) {
		t.Fatalf("provider.Builtins() = %v, want provhost profileProviders %v: the two six-provider literals disagree without a spec change", got, profileProviders)
	}
}
