package tmuxserver

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// fakeDialer scripts dial outcomes FIFO.
type fakeDialer struct {
	outcomes []DialOutcome
	calls    []string
}

func (dialer *fakeDialer) Dial(socket string) DialOutcome {
	dialer.calls = append(dialer.calls, socket)
	if len(dialer.outcomes) == 0 {
		return DialUnknown
	}
	next := dialer.outcomes[0]
	dialer.outcomes = dialer.outcomes[1:]
	return next
}

func TestUnixDialerMissingSocketIsStale(t *testing.T) {
	// Under a verified parent a missing socket proves no bind: stale,
	// never unknown, never live. The containing test temp directory keeps
	// the socket path short enough for the platform's unix address limit.
	dialer := UnixDialer{}
	if got := dialer.Dial(filepath.Join(shortTestTempDir(t), "ax.sock")); got != DialStale {
		t.Fatalf("missing socket = %s, want stale", got)
	}
}

func TestServerProberRefusesWithoutDependencies(t *testing.T) {
	root := shortTestTempDir(t)
	prober := ServerProber{Root: root, Platform: scalar.PlatformMacOS}
	if _, err := prober.Probe(root + "/tmux/ax.sock"); err == nil {
		t.Fatal("nil dialer/admission probed")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "probe dependencies")
	}
	prober = ServerProber{Root: root, Platform: scalar.PlatformMacOS, Dialer: &fakeDialer{}}
	if _, err := prober.Probe(root + "/tmux/ax.sock"); err == nil {
		t.Fatal("nil admission probed")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "probe dependencies")
	}
	prober = ServerProber{Root: root, Platform: scalar.PlatformMacOS, Admit: func() (RealmAdmission, error) {
		return RealmAdmission{}, nil
	}}
	if _, err := prober.Probe(root + "/tmux/ax.sock"); err == nil {
		t.Fatal("nil dialer probed")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "probe dependencies")
	}
}

func TestBrokerProberReportsMiss(t *testing.T) {
	report, err := BrokerProber{}.Probe()
	if err != nil {
		t.Fatal(err)
	}
	if report.Principal.UID != 0 || report.Principal.Generation != "" {
		t.Fatalf("broker probe names a principal: %+v", report.Principal)
	}
}

func TestReconcileAdmissionFailsClosedWithoutVerifier(t *testing.T) {
	admit := ReconcileAdmission(terminalbackend.Manifest{}, terminalbackend.Probe{}, nil, lxGeneration, lxNow(), nil)
	if _, err := admit(); err == nil {
		t.Fatal("nil verifier admitted")
	}
}

func TestProductionWiresDependencies(t *testing.T) {
	deps := Production("/root", scalar.PlatformMacOS, newFakeRunner(), &fakeDialer{}, func() (RealmAdmission, error) {
		return RealmAdmission{}, errors.New("no admission")
	})
	if deps.ProbeServer == nil || deps.ProbeBroker == nil || deps.Spawn == nil {
		t.Fatal("production dependencies incomplete")
	}
}

func TestServerSpawnerRefusesWithoutRunner(t *testing.T) {
	spawner := ServerSpawner{Root: t.TempDir(), Platform: scalar.PlatformMacOS}
	if err := spawner.Spawn("/root/tmux/ax.sock", []string{"tmux"}); err == nil {
		t.Fatal("nil runner spawned")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "spawn dependencies")
	}
}
