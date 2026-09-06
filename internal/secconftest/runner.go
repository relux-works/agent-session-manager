package secconftest

import (
	"fmt"
	"hash/fnv"
	"sort"
	"sync/atomic"
)

// Calls counts production entry-point invocations. A fixture receives a
// counter and must record every production call it makes; the Runner
// refuses fixtures that record none, so a corpus that never reaches the
// entry point cannot pass as coverage.
type Calls struct {
	count atomic.Int64
}

// Add records n production calls.
func (calls *Calls) Add(n int64) {
	calls.count.Add(n)
}

// Count reports the recorded calls.
func (calls *Calls) Count() int64 {
	return calls.count.Load()
}

// Fixture is one named deterministic case. Run must be a pure function
// of Seed: same seed, same report string. It receives a fresh Calls
// counter and must record each production entry-point invocation it
// performs.
type Fixture struct {
	Name string
	Run  func(seed uint64, calls *Calls) (report string, err error)
}

// Result is the executed record of one fixture.
type Result struct {
	Name   string
	Report string
	Calls  int64
}

// Runner executes fixtures in sorted name order over an explicit seed.
// Registration order and Go map iteration order never affect execution:
// names are sorted before every run, so identical inputs produce one
// answer no matter how the caller stored the fixtures.
type Runner struct {
	// AllowUnreached disables the arrival check. It exists only so the
	// suite can prove the check is load-bearing: with it set, an
	// unreached fixture passes, and TestRunnerRefusesUnreachedFixture
	// pins that the default refuses. Production use leaves the zero
	// value.
	AllowUnreached bool
}

// Run executes every fixture once and returns results in sorted name
// order. A fixture that records zero production calls is refused with
// an error naming it, unless AllowUnreached is set.
func (runner Runner) Run(fixtures []Fixture, seed uint64) ([]Result, error) {
	ordered := append([]Fixture(nil), fixtures...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	results := make([]Result, 0, len(ordered))
	for _, fixture := range ordered {
		var calls Calls
		report, err := fixture.Run(seed, &calls)
		if err != nil {
			return nil, fmt.Errorf("secconftest runner: fixture %q: %w", fixture.Name, err)
		}
		if !runner.AllowUnreached && calls.Count() == 0 {
			return nil, fmt.Errorf("secconftest runner: fixture %q recorded no production call", fixture.Name)
		}
		results = append(results, Result{Name: fixture.Name, Report: report, Calls: calls.Count()})
	}
	return results, nil
}

// Digest folds the execution log into one FNV-64 hex value: fixture
// names, reports, and call counts in execution order. Two runs with
// identical inputs produce identical digests; any dropped, reordered,
// or altered fixture changes it.
func Digest(results []Result) string {
	sum := fnv.New64a()
	for _, result := range results {
		_, _ = fmt.Fprintf(sum, "%s\x00%s\x00%d\x00", result.Name, result.Report, result.Calls)
	}
	return fmt.Sprintf("%016x", sum.Sum64())
}

// SortedNames reports the execution order for a fixture list. It shares
// the sort with Run so the order claim is testable in isolation.
func SortedNames(fixtures []Fixture) []string {
	ordered := append([]Fixture(nil), fixtures...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	names := make([]string, 0, len(ordered))
	for _, fixture := range ordered {
		names = append(names, fixture.Name)
	}
	return names
}
