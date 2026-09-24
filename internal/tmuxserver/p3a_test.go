package tmuxserver

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestAcquireForegroundRefusesEveryCatalogDecoyRunning closes the rev7
// P3-A residue: the foreground attach wiring refuses every catalog
// decoy — each alone and all together — when the server reports
// running. A running server whose admission lacks the realm row never
// attaches, however fresh its generation; the membership arm, not the
// staleness arm, refuses. Probe K31's shape, lifted to the committed
// suite with spawn-silence on every member.
func TestAcquireForegroundRefusesEveryCatalogDecoyRunning(t *testing.T) {
	decoys := catalogDecoyCapabilities(t)
	drive := func(t *testing.T, capabilities []string) {
		t.Helper()
		root := runtimeRoot(t)
		runtimeDir(t, root)
		req := validRequest(t, root)
		req.Caller = CallerForeground
		req.Platform = scalar.PlatformMacOS
		spy := &spawnSpy{}
		req.Deps.Spawn = spy.spawn
		req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
			return ServerReport{
				Running: true,
				Admission: RealmAdmission{
					Admitted:      terminalbackend.Admitted{Capabilities: capabilities},
					RawGeneration: fixtureGeneration,
				},
			}, nil
		}
		_, err := Acquire(req)
		requireLocalError(t, err, "tmux_readiness_not_authorizing", "server attestation")
		if spy.calls != 0 {
			t.Fatalf("spawn calls = %d, want 0", spy.calls)
		}
	}
	for _, decoy := range decoys {
		t.Run(decoy, func(t *testing.T) {
			drive(t, []string{decoy})
		})
	}
	t.Run("all decoys together", func(t *testing.T) {
		drive(t, append([]string(nil), decoys...))
	})
}
