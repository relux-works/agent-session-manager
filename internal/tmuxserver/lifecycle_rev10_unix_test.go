//go:build !windows

package tmuxserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestAttachPeerFilenameMismatchFailsClosed pins the attach
// fail-closed on a corrupt receipt namespace: B holds a valid
// receipt, its bytes are moved to A's durable key (so B's own key
// is empty and this is not a validated durable replay), and B
// attaches again with multi_attach absent. The census must refuse
// the misfiled bytes through the keyed binding before the
// requesting-client exclusion can erase them — no receipt, no
// vector, no input authorization.
func TestAttachPeerFilenameMismatchFailsClosed(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	first := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: first, Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance)
	raw, err := os.ReadFile(filepath.Join(dir, lxClientB+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, lxClientB+".json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, lxClient+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	second := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: withoutCapability("multi_attach"),
	})
	if err == nil {
		t.Fatalf("attach admitted past a receipt misfiled under another client's key: %+v", outcome.Attach)
	}
	if !strings.Contains(err.Error(), "attach receipt names another client pair") {
		t.Fatalf("refusal = %v, want the keyed-binding corruption refusal", err)
	}
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("lookup = %v, %v; want no receipt past the corrupt census", found, err)
	}
	if len(outcome.Argv) != 0 {
		t.Fatalf("refused attach built argv %q", outcome.Argv)
	}
	if outcome.Attach != nil {
		t.Fatalf("refused attach authorized input: %+v", outcome.Attach)
	}
}

// contextHonoringRunner delegates to a scripted fake runner but
// refuses a cancelled call first, matching OSRunner and
// exec.CommandContext cancellation semantics: a probe whose bound
// already fired never runs.
type contextHonoringRunner struct {
	inner *fakeRunner
}

func (runner *contextHonoringRunner) Run(ctx context.Context, argv []string) (RunResult, error) {
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	return runner.inner.Run(ctx, argv)
}

// TestExecuteStopEscalationSucceedsWithContextHonoringRunner pins
// the post-escalation observation budget: after the 5ms graceful
// wait expires on a live session, the escalation kill issues under
// the still-valid operation deadline and authorization, and the
// re-confirmation observes the closure — even though the
// cancellation-honouring executor would refuse any probe still
// bound by the exhausted graceful wait. The graceful budget and
// the post-escalation observation budget are distinct; a timeout
// still concludes unknown, never closure.
func TestExecuteStopEscalationSucceedsWithContextHonoringRunner(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.runner.queue("send-keys", "", 0).queue("has-session", "", 0).
		queue("kill-session", "", 0).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	seen := false
	fx.runner.onRun = func(argv []string) {
		if len(argv) >= 4 && argv[3] == "has-session" && !seen {
			seen = true
			fx.clock.Sleep(10 * time.Millisecond)
		}
	}
	fx.lc.Runner = &contextHonoringRunner{inner: fx.runner}
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "request-stop", Body: lxStopBody(t, 5, nil), Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.runner.subcommands(); len(got) != 4 || got[0] != "send-keys" || got[1] != "has-session" || got[2] != "kill-session" || got[3] != "has-session" {
		t.Fatalf("calls = %v, want the graceful poll, one escalation, and the observed re-confirm", got)
	}
	if outcome.Mutation.After != terminalbackend.StateStopped {
		t.Fatalf("after = %s", outcome.Mutation.After)
	}
}
