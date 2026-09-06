package secconftest

import (
	"strings"
	"testing"
)

func TestRunnerExecutesInSortedOrder(t *testing.T) {
	t.Parallel()
	var order []string
	fixtures := []Fixture{
		{Name: "zeta", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(1)
			order = append(order, "zeta")
			return "z", nil
		}},
		{Name: "alpha", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(2)
			order = append(order, "alpha")
			return "a", nil
		}},
		{Name: "mid", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(3)
			order = append(order, "mid")
			return "m", nil
		}},
	}
	var runner Runner
	results, err := runner.Run(fixtures, 7)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"alpha", "mid", "zeta"}
	got := SortedNames(fixtures)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("SortedNames = %v, want %v", got, want)
	}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("execution order = %v, want sorted %v", order, want)
	}
	for index, name := range want {
		if results[index].Name != name {
			t.Fatalf("results[%d].Name = %q, want %q", index, results[index].Name, name)
		}
	}
}

func TestRunnerDigestIsDeterministic(t *testing.T) {
	t.Parallel()
	fixtures := []Fixture{
		{Name: "b", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(1)
			return "report-b", nil
		}},
		{Name: "a", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(1)
			return "report-a", nil
		}},
	}
	var runner Runner
	first, err := runner.Run(fixtures, 99)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	second, err := runner.Run(fixtures, 99)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Registration order must not matter: reverse and rerun.
	reversed := []Fixture{fixtures[1], fixtures[0]}
	third, err := runner.Run(reversed, 99)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Digest(first) != Digest(second) || Digest(first) != Digest(third) {
		t.Fatalf("digest unstable: %q %q %q", Digest(first), Digest(second), Digest(third))
	}
	t.Logf("deterministic digest: %s", Digest(first))
}

func TestRunnerRefusesUnreachedFixture(t *testing.T) {
	t.Parallel()
	var runner Runner
	fixtures := []Fixture{
		{Name: "reached", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(1)
			return "ok", nil
		}},
		{Name: "unreached", Run: func(seed uint64, calls *Calls) (string, error) {
			return "never touched production", nil
		}},
	}
	_, err := runner.Run(fixtures, 1)
	if err == nil || !strings.Contains(err.Error(), `"unreached"`) {
		t.Fatalf("Run with an unreached fixture = %v, want an error naming it", err)
	}
}

func TestRunnerAllowsUnreachedWhenOptedIn(t *testing.T) {
	t.Parallel()
	runner := Runner{AllowUnreached: true}
	fixtures := []Fixture{
		{Name: "unreached", Run: func(seed uint64, calls *Calls) (string, error) {
			return "opted in", nil
		}},
	}
	results, err := runner.Run(fixtures, 1)
	if err != nil {
		t.Fatalf("AllowUnreached Run: %v", err)
	}
	if len(results) != 1 || results[0].Calls != 0 {
		t.Fatalf("AllowUnreached results = %+v, want one zero-call result", results)
	}
}

func TestRunnerDigestCoversAllFixtures(t *testing.T) {
	t.Parallel()
	var runner Runner
	fixtures := []Fixture{
		{Name: "a", Run: func(seed uint64, calls *Calls) (string, error) { calls.Add(1); return "ra", nil }},
		{Name: "b", Run: func(seed uint64, calls *Calls) (string, error) { calls.Add(1); return "rb", nil }},
	}
	full, err := runner.Run(fixtures, 3)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	partial, err := runner.Run(fixtures[:1], 3)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Digest(full) == Digest(partial) {
		t.Fatal("digest ignores a dropped fixture; the inventory is unwitnessed")
	}
	altered := []Fixture{
		{Name: "a", Run: func(seed uint64, calls *Calls) (string, error) { calls.Add(1); return "ra", nil }},
		{Name: "b", Run: func(seed uint64, calls *Calls) (string, error) { calls.Add(9); return "rb", nil }},
	}
	rerun, err := runner.Run(altered, 3)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Digest(full) == Digest(rerun) {
		t.Fatal("digest ignores a changed call count; the execution log is unwitnessed")
	}
}

func TestRunnerDigestDistinguishesReports(t *testing.T) {
	t.Parallel()
	// Same names, same call counts, different reports: a digest that
	// drops the report witnesses nothing and must fail here.
	run := func(report string) []Result {
		var runner Runner
		results, err := runner.Run([]Fixture{
			{Name: "a", Run: func(seed uint64, calls *Calls) (string, error) { calls.Add(1); return report, nil }},
		}, 3)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		return results
	}
	if Digest(run("ra")) == Digest(run("rb")) {
		t.Fatal("digest ignores the report; the witness is call counts only")
	}
}

func TestRunnerDigestDistinguishesNames(t *testing.T) {
	t.Parallel()
	// Same reports, same call counts, different names: a digest that
	// drops the name witnesses nothing and must fail here.
	run := func(name string) []Result {
		var runner Runner
		results, err := runner.Run([]Fixture{
			{Name: name, Run: func(seed uint64, calls *Calls) (string, error) { calls.Add(1); return "r", nil }},
		}, 3)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		return results
	}
	if Digest(run("a")) == Digest(run("b")) {
		t.Fatal("digest ignores the name; the witness is call counts only")
	}
}

func TestRunnerPropagatesFixtureError(t *testing.T) {
	t.Parallel()
	var runner Runner
	fixtures := []Fixture{
		{Name: "broken", Run: func(seed uint64, calls *Calls) (string, error) {
			calls.Add(1)
			return "", errTestSentinel
		}},
	}
	_, err := runner.Run(fixtures, 1)
	if err == nil || !strings.Contains(err.Error(), `"broken"`) {
		t.Fatalf("Run with a failing fixture = %v, want an error naming it", err)
	}
}
