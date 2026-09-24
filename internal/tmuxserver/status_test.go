package tmuxserver

import (
	"context"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// stageStatusBinding records the bootstrap binding receipt plus the
// create-time binding document the identity observation reads: the
// recorded (session, bootstrap) pair carrying the lifecycle identity
// tuple. Exact-instance observation requires it; a query without a
// recorded binding reads unknown, never an echo.
func stageStatusBinding(t *testing.T, reqBackend *backend) {
	t.Helper()
	if _, _, err := reqBackend.bindings.Bind(lxSession, lxBootstrap, axpane.Binding{
		SessionID:          lxSession,
		OperationID:        lxBootstrap,
		TerminalInstanceID: lxInstance,
		BindingDigest:      lxDigestA,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reqBackend.states.RecordBindingDoc(lxSession, lxBootstrap, lxBindingDoc(t, nil)); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyStatusAbsentProbe(t *testing.T) {
	cases := []struct {
		name      string
		memory    terminalbackend.InstanceState
		found     bool
		wantState terminalbackend.InstanceState
	}{
		{"no-memory", "", false, terminalbackend.StateAbsent},
		{"memory-absent", terminalbackend.StateAbsent, true, terminalbackend.StateAbsent},
		{"memory-stopped", terminalbackend.StateStopped, true, terminalbackend.StateStopped},
		{"memory-fenced", terminalbackend.StateStaleFenced, true, terminalbackend.StateStaleFenced},
		{"memory-active", terminalbackend.StateActive, true, terminalbackend.StateUnavailable},
		{"memory-parked", terminalbackend.StateParked, true, terminalbackend.StateUnavailable},
		{"memory-quiescing", terminalbackend.StateQuiescing, true, terminalbackend.StateUnavailable},
		{"memory-unavailable", terminalbackend.StateUnavailable, true, terminalbackend.StateUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, wrapper, provider, attachable := ClassifyStatus(ClassifyInput{Memory: tc.memory, MemoryFound: tc.found})
			if state != tc.wantState || wrapper || provider || attachable {
				t.Fatalf("classify = %s/%v/%v/%v, want %s/false/false/false", state, wrapper, provider, attachable, tc.wantState)
			}
		})
	}
}

func TestClassifyStatusPresentProbe(t *testing.T) {
	cases := []struct {
		name      string
		memory    terminalbackend.InstanceState
		found     bool
		attached  int
		rows      int
		capable   bool
		wantState terminalbackend.InstanceState
		wantWrap  bool
		wantProv  bool
		wantAtt   bool
	}{
		{"parked", terminalbackend.StateParked, true, 0, 2, true, terminalbackend.StateParked, true, true, true},
		{"active", terminalbackend.StateActive, true, 3, 1, true, terminalbackend.StateActive, true, true, true},
		{"no-memory-parked", "", false, 0, 1, true, terminalbackend.StateParked, true, true, true},
		{"no-memory-active", "", false, 1, 1, true, terminalbackend.StateActive, true, true, true},
		{"memory-quiescing", terminalbackend.StateQuiescing, true, 1, 2, true, terminalbackend.StateQuiescing, true, true, false},
		{"memory-fenced", terminalbackend.StateStaleFenced, true, 0, 1, true, terminalbackend.StateStaleFenced, true, true, false},
		{"memory-stopped", terminalbackend.StateStopped, true, 0, 1, true, terminalbackend.StateUnavailable, false, false, false},
		{"memory-unavailable", terminalbackend.StateUnavailable, true, 0, 1, true, terminalbackend.StateUnavailable, false, false, false},
		{"memory-creating", terminalbackend.StateCreating, true, 0, 1, true, terminalbackend.StateUnavailable, false, false, false},
		{"no-attach-cap", terminalbackend.StateParked, true, 0, 1, false, terminalbackend.StateParked, true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, wrapper, provider, attachable := ClassifyStatus(ClassifyInput{
				Present: true, Attached: tc.attached, ProviderRows: tc.rows,
				Memory: tc.memory, MemoryFound: tc.found, AttachCapable: tc.capable,
			})
			if state != tc.wantState || wrapper != tc.wantWrap || provider != tc.wantProv || attachable != tc.wantAtt {
				t.Fatalf("classify = %s/%v/%v/%v, want %s/%v/%v/%v",
					state, wrapper, provider, attachable, tc.wantState, tc.wantWrap, tc.wantProv, tc.wantAtt)
			}
		})
	}
}

func TestStatusEvidenceSortedUniqueDigests(t *testing.T) {
	evidence := statusEvidence(lxInstance, true, []string{"sh", "ax"}, 2)
	if len(evidence) != 2 {
		t.Fatalf("evidence = %q", evidence)
	}
	for _, id := range evidence {
		if _, err := scalar.ParseDigest(id); err != nil {
			t.Fatalf("evidence %q: %v", id, err)
		}
	}
	if evidence[0] > evidence[1] {
		t.Fatalf("evidence unsorted: %q", evidence)
	}
	absent := statusEvidence(lxInstance, false, nil, 0)
	if len(absent) != 1 {
		t.Fatalf("absent evidence = %q", absent)
	}
	if _, err := scalar.ParseDigest(absent[0]); err != nil {
		t.Fatalf("absent evidence: %v", err)
	}
}

func TestObserveStatusNilStoreRefuses(t *testing.T) {
	// Identity observation reads the durable binding: a backend
	// without the binding store cannot observe identity, so it
	// refuses instead of classifying from the live probe alone.
	// (Replaces TestObserveStatusNilMemoryReadsLive: the nil-memory
	// tolerance died with the echo — rev2 P1-B.)
	runner := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	reqBackend, _ := statusBackend(t, runner, admittedWith())
	stageStatusBinding(t, reqBackend)
	reqBackend.states = nil
	_, err := reqBackend.ObserveStatus(context.Background(), exactStatusBody(t))
	if err == nil {
		t.Fatal("nil binding store observed identity")
	}
	requireEngineCode(t, err, "terminal_backend_protocol_error")
}

func TestProviderTripleRule(t *testing.T) {
	// provider_present is non-null exactly when observation was
	// requested, capable, and observed; the flags mirror the rule for
	// the landed result check.
	cases := []struct {
		requested bool
		capable   bool
		observed  bool
		wantNull  bool
		wantFlags [2]bool
	}{
		{true, true, true, false, [2]bool{true, true}},
		{true, true, false, true, [2]bool{true, false}},
		{true, false, true, true, [2]bool{true, false}},
		{true, false, false, true, [2]bool{true, false}},
		{false, true, true, true, [2]bool{false, false}},
		{false, true, false, true, [2]bool{false, false}},
		{false, false, true, true, [2]bool{false, false}},
		{false, false, false, true, [2]bool{false, false}},
	}
	for _, tc := range cases {
		present, requested, evidenced := providerTriple(tc.requested, tc.capable, tc.observed)
		if (present == nil) != tc.wantNull || (present != nil && !*present) {
			t.Fatalf("triple(%v,%v,%v) = %+v, want null=%v", tc.requested, tc.capable, tc.observed, present, tc.wantNull)
		}
		if requested != tc.wantFlags[0] || evidenced != tc.wantFlags[1] {
			t.Fatalf("flags(%v,%v,%v) = %v/%v", tc.requested, tc.capable, tc.observed, requested, evidenced)
		}
	}
}

func TestCutRowShapes(t *testing.T) {
	name, count, ok := cutRow(lxInstance + "|3")
	if !ok || name != lxInstance || count != "3" {
		t.Fatalf("cut = %q %q %v", name, count, ok)
	}
	for _, bad := range []string{"", "no-pipe", "|3", lxInstance + "|", "|"} {
		if _, _, ok := cutRow(bad); ok {
			t.Fatalf("cut %q admitted", bad)
		}
	}
}

// statusBackend builds a request backend for ObserveStatus unit tests:
// fake runner, real states, admitted set, no exec of tmux.
func statusBackend(t *testing.T, runner *fakeRunner, admitted terminalbackend.Admitted) (*backend, *InstanceStates) {
	t.Helper()
	states, err := OpenInstanceStates(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := testAxpaneStore(t)
	if err != nil {
		t.Fatal(err)
	}
	return &backend{
		runner:     runner,
		runtimeDir: "/root/tmux",
		socket:     "/root/tmux/ax.sock",
		sessionID:  lxSession,
		instanceID: lxInstance,
		now:        lxNow,
		sleep:      func(time.Duration) {},
		poll:       0,
		admitted:   admitted,
		bindings:   bindings,
		states:     states,
		killExit:   -1,
	}, states
}

func exactStatusBody(t *testing.T) terminstance.StatusBody {
	t.Helper()
	body, err := terminstance.ParseStatusBody(lxStatusBody(t, true, true, nil))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestObserveStatusPresentParked(t *testing.T) {
	runner := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	reqBackend, _ := statusBackend(t, runner, admittedWith("provider_process_observation", "local_attach"))
	stageStatusBinding(t, reqBackend)
	observed, err := reqBackend.ObserveStatus(context.Background(), exactStatusBody(t))
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != terminalbackend.StateParked || !observed.WrapperPresent || !observed.Attachable {
		t.Fatalf("observed = %+v", observed)
	}
	if observed.ProviderPresent == nil || !*observed.ProviderPresent {
		t.Fatalf("provider = %+v", observed.ProviderPresent)
	}
	if !observed.ProviderRequested || !observed.ProviderEvidenced || !observed.AttachEvidenced {
		t.Fatalf("flags = %+v", observed)
	}
}

func TestObserveStatusAbsent(t *testing.T) {
	runner := newFakeRunner().queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
	reqBackend, _ := statusBackend(t, runner, admittedWith())
	stageStatusBinding(t, reqBackend)
	observed, err := reqBackend.ObserveStatus(context.Background(), exactStatusBody(t))
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != terminalbackend.StateAbsent || observed.WrapperPresent || observed.Attachable {
		t.Fatalf("observed = %+v", observed)
	}
	if observed.ProviderPresent != nil {
		t.Fatalf("provider = %+v", observed.ProviderPresent)
	}
}

func TestObserveStatusUnknowns(t *testing.T) {
	cases := []struct {
		name      string
		script    func(*fakeRunner) *fakeRunner
		transport bool
	}{
		{"transport", func(r *fakeRunner) *fakeRunner { return r.failTransport("list-panes", errFakeTransport) }, true},
		{"empty-rows", func(r *fakeRunner) *fakeRunner { return r.queue("list-panes", "\n", 0) }, false},
		{"sessions-exit", func(r *fakeRunner) *fakeRunner {
			return r.queue("list-panes", "sh\n", 0).queue("list-sessions", "", 1)
		}, false},
		{"sessions-missing-row", func(r *fakeRunner) *fakeRunner {
			return r.queue("list-panes", "sh\n", 0).queue("list-sessions", lxSessionB+"|0\n", 0)
		}, false},
		{"sessions-malformed-row", func(r *fakeRunner) *fakeRunner {
			return r.queue("list-panes", "sh\n", 0).queue("list-sessions", "garbage\n", 0)
		}, false},
		{"sessions-bad-count", func(r *fakeRunner) *fakeRunner {
			return r.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|many\n", 0)
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqBackend, _ := statusBackend(t, tc.script(newFakeRunner()), admittedWith())
			stageStatusBinding(t, reqBackend)
			_, err := reqBackend.ObserveStatus(context.Background(), exactStatusBody(t))
			if err == nil {
				t.Fatalf("%s reported", tc.name)
			}
			// Transport proves nothing and passes through uncoded;
			// every contradictory probe refuses the unknown verdict.
			if tc.transport {
				if err != errFakeTransport {
					t.Fatalf("err = %v, want the transport failure", err)
				}
				return
			}
			requireEngineCode(t, err, "terminal_backend_unavailable")
		})
	}
}

func TestObserveStatusMemoryReconciles(t *testing.T) {
	// Stopped memory plus a live session is a resurrection: unavailable.
	runner := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|0\n", 0)
	reqBackend, states := statusBackend(t, runner, admittedWith())
	stageStatusBinding(t, reqBackend)
	if err := states.Record(lxInstance, terminalbackend.StateStopped); err != nil {
		t.Fatal(err)
	}
	observed, err := reqBackend.ObserveStatus(context.Background(), exactStatusBody(t))
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != terminalbackend.StateUnavailable {
		t.Fatalf("state = %s", observed.State)
	}
	// Quiescing memory plus a live session adopts quiescing with no attach.
	runner2 := newFakeRunner().
		queue("list-panes", "sh\n", 0).
		queue("list-sessions", lxInstance+"|2\n", 0)
	reqBackend2, states2 := statusBackend(t, runner2, admittedWith("local_attach"))
	stageStatusBinding(t, reqBackend2)
	if err := states2.Record(lxInstance, terminalbackend.StateQuiescing); err != nil {
		t.Fatal(err)
	}
	observed2, err := reqBackend2.ObserveStatus(context.Background(), exactStatusBody(t))
	if err != nil {
		t.Fatal(err)
	}
	if observed2.State != terminalbackend.StateQuiescing || !observed2.WrapperPresent || observed2.Attachable {
		t.Fatalf("observed = %+v", observed2)
	}
}

func TestObserveStatusSessionScoped(t *testing.T) {
	// Unbound session reads absent without probing tmux.
	reqBackend, _ := statusBackend(t, newFakeRunner(), admittedWith())
	scoped, err := terminstance.ParseStatusBody(lxStatusBody(t, false, false, nil))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := reqBackend.ObserveStatus(context.Background(), scoped)
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != terminalbackend.StateAbsent || !observed.IdentityMatch {
		t.Fatalf("observed = %+v", observed)
	}
}
