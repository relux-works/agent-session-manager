package provhost

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// TestEnvNameAgreementIsBidirectional pins the shared environment-name
// grammar from both directions: validEnvName (the Section 5.1 SpawnPlan
// call sites) must equal secprim.IsEnvName (the launch-allowlist owner)
// on every swept name. A one-directional test — mine refuses what theirs
// refuses — cannot see an over-permissive divergence, so this test
// asserts equality, not implication, over an exhaustive small alphabet
// plus length boundaries and hostile shapes.
func TestEnvNameAgreementIsBidirectional(t *testing.T) {
	t.Parallel()
	alphabet := []byte{'a', 'Z', '0', '_', '-', '.', ' ', '=', 0xe9}
	var names []string
	names = append(names, "")
	var build func(prefix []byte, depth int)
	build = func(prefix []byte, depth int) {
		if depth == 3 {
			return
		}
		for _, letter := range alphabet {
			next := append(append([]byte(nil), prefix...), letter)
			names = append(names, string(next))
			build(next, depth+1)
		}
	}
	build(nil, 0)
	names = append(names,
		strings.Repeat("x", 127),
		strings.Repeat("x", 128),
		strings.Repeat("x", 129),
		"_"+strings.Repeat("y", 127),
		"9"+strings.Repeat("y", 127),
		"A\nB", "A\x00B", "AX_CONFIG", "PATH", "a-b", "a.b", "a=b",
	)
	mismatches := 0
	for _, name := range names {
		if validEnvName(name) != secprim.IsEnvName(name) {
			mismatches++
			t.Errorf("validEnvName(%q) != secprim.IsEnvName: the grammar drifted", name)
		}
	}
	if mismatches != 0 {
		t.Fatalf("%d grammar mismatches; the shared name rule drifted", mismatches)
	}
	t.Logf("agreement holds over %d swept names", len(names))
}
