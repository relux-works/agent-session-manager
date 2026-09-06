package cigate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
)

// FuzzTargets derives the committed secconftest fuzz-target list from the
// instrument's own FuzzTargetEntry map, sorted. CI's fuzz-smoke job derives
// its secconftest leg from the compiled test binary via `go test -list`,
// and TestListedTargetsMatchDerivation holds that list equal to this
// derivation: a target added without an entry — or an entry without a
// target — reddens the suite before CI can pass on an empty or stale list.
func FuzzTargets() ([]string, error) {
	entries := make(map[string]string, len(secconftest.FuzzTargetEntry))
	for target, entry := range secconftest.FuzzTargetEntry {
		entries[target] = string(entry)
	}
	return sortedTargets(entries)
}

// sortedTargets is the testable core of FuzzTargets: production passes the
// instrument's own map, tests pass synthetic ones.
func sortedTargets(entries map[string]string) ([]string, error) {
	targets := make([]string, 0, len(entries))
	for target, entry := range entries {
		if !strings.HasPrefix(target, "Fuzz") || entry == "" {
			return nil, fmt.Errorf("cigate: fuzz target entry %q is malformed", target)
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("cigate: fuzz target derivation is empty")
	}
	sort.Strings(targets)
	return targets, nil
}
