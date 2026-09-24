//go:build !windows

package tmuxserver

import (
	"testing"
)

// A foreign-owned leaf on the background path keeps its custody code
// at the top level — tmux_unsafe_runtime_dir at runtime ownership,
// never the capability_unavailable wrapper — with no broker contact
// and neither foreground dependency invoked. The ownership gate is
// staged through the effectiveUID seam: a genuinely foreign-owned
// directory needs privilege no test may assume.
func TestAcquireBackgroundStagesOwnershipRefusal(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	previous := effectiveUID
	effectiveUID = func() int { return previous() + 100000 }
	defer func() { effectiveUID = previous }()
	req := validRequest(t, root)
	req.Caller = CallerBackground
	spy := &spawnSpy{}
	probe := &probeServerSpy{}
	probed := false
	req.Deps.Spawn = spy.spawn
	req.Deps.ProbeServer = probe.probe
	req.Deps.ProbeBroker = func() (BrokerReport, error) {
		probed = true
		return brokerReport(), nil
	}
	_, err := Acquire(req)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime ownership")
	if probed {
		t.Fatal("broker was probed past a foreign-owned leaf")
	}
	if spy.calls != 0 {
		t.Fatalf("spawn calls = %d, want 0", spy.calls)
	}
	probe.requireSilent(t)
}
