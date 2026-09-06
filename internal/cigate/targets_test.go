package cigate

import (
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestFuzzTargetsAreDerivedSorted(t *testing.T) {
	t.Parallel()
	targets, err := FuzzTargets()
	if err != nil {
		t.Fatalf("FuzzTargets() error = %v", err)
	}
	if len(targets) == 0 {
		t.Fatal("FuzzTargets() is empty; CI would pass on an empty matrix")
	}
	sorted := append([]string(nil), targets...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(targets, sorted) {
		t.Fatalf("FuzzTargets() = %v, want sorted", targets)
	}
	for _, target := range targets {
		if len(target) < 5 || target[:4] != "Fuzz" {
			t.Errorf("target %q is not a Fuzz entry point", target)
		}
	}
}

func TestSortedTargetsRefusals(t *testing.T) {
	t.Parallel()
	if _, err := sortedTargets(map[string]string{}); err == nil {
		t.Error("empty derivation admitted; CI would expand an empty matrix")
	}
	if _, err := sortedTargets(map[string]string{"NotAFuzz": "secprim.X"}); err == nil {
		t.Error("non-Fuzz target admitted")
	}
	if _, err := sortedTargets(map[string]string{"FuzzX": ""}); err == nil {
		t.Error("target without an entry admitted")
	}
	got, err := sortedTargets(map[string]string{"FuzzB": "e", "FuzzA": "e"})
	if err != nil {
		t.Fatalf("sortedTargets() error = %v", err)
	}
	if !reflect.DeepEqual(got, []string{"FuzzA", "FuzzB"}) {
		t.Fatalf("sortedTargets() = %v, want sorted", got)
	}
}

// TestListedTargetsMatchDerivation is the CI wiring for FuzzTargets: the
// fuzz-smoke job smokes whatever `go test -list` reports, so this test
// holds that reported set equal to the entry-mapped derivation. A target
// added to the binary without a FuzzTargetEntry — or an entry without a
// target — fails here before CI can smoke a list the derivation does not
// cover. The absolute package path keeps the check independent of the
// test's working directory; listing compiles only and touches no network.
func TestListedTargetsMatchDerivation(t *testing.T) {
	t.Parallel()
	targets, err := FuzzTargets()
	if err != nil {
		t.Fatalf("FuzzTargets() error = %v", err)
	}
	out, err := exec.Command("go", "test",
		"github.com/relux-works/agent-session-manager/internal/secconftest",
		"-list", "^Fuzz").CombinedOutput()
	if err != nil {
		t.Fatalf("go test -list error = %v, output:\n%s", err, out)
	}
	var listed []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Fuzz") {
			listed = append(listed, line)
		}
	}
	sort.Strings(listed)
	if !reflect.DeepEqual(listed, targets) {
		t.Fatalf("listed targets = %v, derivation = %v; add the FuzzTargetEntry or remove the target", listed, targets)
	}
}
