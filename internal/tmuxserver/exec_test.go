package tmuxserver

import (
	"context"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestParseDirectiveAdmitsTenPrimitives(t *testing.T) {
	for _, directive := range []string{
		"new-session", "attach-session", "list-panes", "list-sessions",
		"lock-session", "detach-client", "wait-for", "send-keys", "has-session", "kill-session",
	} {
		parsed, err := ParseDirective(directive)
		if err != nil {
			t.Fatalf("ParseDirective(%q): %v", directive, err)
		}
		if string(parsed) != directive {
			t.Fatalf("ParseDirective(%q) = %q", directive, parsed)
		}
	}
}

func TestParseDirectiveRefusesUnknown(t *testing.T) {
	for _, value := range []string{"", "kill-server", "new", "attach", "NEW-SESSION", "new-session ", "run-shell"} {
		if _, err := ParseDirective(value); err == nil {
			t.Fatalf("ParseDirective(%q) admitted", value)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "command vocabulary")
		}
	}
}

func TestDirectivesForMapsEightOperations(t *testing.T) {
	cases := map[string][]Directive{
		"create":             {DirectiveNewSession},
		"attach":             {DirectiveAttachSession},
		"status":             {DirectiveListPanes, DirectiveListSessions},
		"quiesce-input":      {DirectiveLockSession, DirectiveDetachClients},
		"wait-safe-boundary": {DirectiveWaitBoundary, DirectiveListPanes},
		"request-stop":       {DirectiveSendInterrupt, DirectiveHasSession, DirectiveKillSession, DirectiveHasSession},
		"terminate-stale":    {DirectiveKillSession, DirectiveHasSession},
		"restore":            {DirectiveNewSession},
	}
	for operation, want := range cases {
		got, err := DirectivesFor(terminalbackend.Operation(operation))
		if err != nil {
			t.Fatalf("DirectivesFor(%q): %v", operation, err)
		}
		if len(got) != len(want) {
			t.Fatalf("DirectivesFor(%q) = %v, want %v", operation, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("DirectivesFor(%q) = %v, want %v", operation, got, want)
			}
		}
	}
}

func TestDirectivesForRefusesNonLifecycle(t *testing.T) {
	for _, operation := range []string{"manifest", "probe", "suspend", ""} {
		if _, err := DirectivesFor(terminalbackend.Operation(operation)); err == nil {
			t.Fatalf("DirectivesFor(%q) admitted", operation)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "lifecycle operation")
		}
	}
}

func lxCommandArgs(socket string) CommandArgs {
	return CommandArgs{
		RuntimeDir:           "/root/tmux",
		Socket:               socket,
		SessionID:            lxSession,
		InstanceID:           lxInstance,
		QuiescenceGeneration: lxQuiesce,
		AttachInput:          true,
		HasAttachInput:       true,
	}
}

func TestBuildCommandVectors(t *testing.T) {
	socket := "/root/tmux/ax.sock"
	cases := []struct {
		name      string
		directive Directive
		want      []string
	}{
		{"new-session", DirectiveNewSession, []string{"tmux", "-S", socket, "new-session", "-d", "-s", lxInstance, "ax", "pane", lxSession}},
		{"attach", DirectiveAttachSession, []string{"tmux", "-S", socket, "attach-session", "-t", lxInstance}},
		{"list-panes", DirectiveListPanes, []string{"tmux", "-S", socket, "list-panes", "-t", lxInstance, "-F", "#{pane_current_command}"}},
		{"list-sessions", DirectiveListSessions, []string{"tmux", "-S", socket, "list-sessions", "-F", "#{session_name}|#{session_attached}"}},
		{"lock", DirectiveLockSession, []string{"tmux", "-S", socket, "lock-session", "-t", lxInstance}},
		{"detach", DirectiveDetachClients, []string{"tmux", "-S", socket, "detach-client", "-s", lxInstance}},
		{"wait", DirectiveWaitBoundary, []string{"tmux", "-S", socket, "wait-for", "ax-boundary-" + lxQuiesce}},
		{"send-keys", DirectiveSendInterrupt, []string{"tmux", "-S", socket, "send-keys", "-t", lxInstance, "C-c"}},
		{"has", DirectiveHasSession, []string{"tmux", "-S", socket, "has-session", "-t", lxInstance}},
		{"kill", DirectiveKillSession, []string{"tmux", "-S", socket, "kill-session", "-t", lxInstance}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildCommand(tc.directive, lxCommandArgs(socket))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("argv = %q, want %q", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("argv = %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestBuildCommandRefusesUnknownDirective(t *testing.T) {
	// The builder re-parses its directive: a foreign primitive refuses
	// the vocabulary before the socket pin ever runs.
	if _, err := BuildCommand(Directive("kill-server"), CommandArgs{}); err == nil {
		t.Fatal("unknown directive built")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "command vocabulary")
	}
}

func TestBuildCommandRefusesAmbientSocket(t *testing.T) {
	sockets := []string{
		"/tmp/ax-ambient.sock",
		"/root/tmux/default",
		"/root/tmux/ax.sock.evil",
		"/other/tmux/ax.sock",
		"ax.sock",
		"",
	}
	for _, socket := range sockets {
		args := lxCommandArgs(socket)
		for _, directive := range []Directive{DirectiveNewSession, DirectiveAttachSession, DirectiveKillSession, DirectiveHasSession} {
			if _, err := BuildCommand(directive, args); err == nil {
				t.Fatalf("BuildCommand(%s, socket %q) admitted", directive, socket)
			} else {
				requireLocalCode(t, err, "tmux_ambient_server_reuse", "command socket")
			}
		}
	}
}

func TestBuildCommandAttachReadOnlyVector(t *testing.T) {
	socket := "/root/tmux/ax.sock"
	args := lxCommandArgs(socket)
	args.AttachInput = false
	got, err := BuildCommand(DirectiveAttachSession, args)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"tmux", "-S", socket, "attach-session", "-r", "-t", lxInstance}
	if len(got) != len(want) {
		t.Fatalf("argv = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv = %q, want %q", got, want)
		}
	}
}

func TestBuildCommandAttachRequiresInputFlag(t *testing.T) {
	socket := "/root/tmux/ax.sock"
	args := lxCommandArgs(socket)
	args.HasAttachInput = false
	if _, err := BuildCommand(DirectiveAttachSession, args); err == nil {
		t.Fatal("unstated attach authorization built")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command attach authorization")
	}
}

func TestBuildCommandRefusesBadIdentity(t *testing.T) {
	socket := "/root/tmux/ax.sock"
	args := lxCommandArgs(socket)
	args.InstanceID = "not-a-uuid"
	if _, err := BuildCommand(DirectiveHasSession, args); err == nil {
		t.Fatal("bad instance admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command instance")
	}
	args = lxCommandArgs(socket)
	args.SessionID = "not-a-uuid"
	if _, err := BuildCommand(DirectiveNewSession, args); err == nil {
		t.Fatal("bad session admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command session")
	}
	args = lxCommandArgs(socket)
	args.QuiescenceGeneration = "not-a-uuid"
	if _, err := BuildCommand(DirectiveWaitBoundary, args); err == nil {
		t.Fatal("bad quiescence admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command quiescence")
	}
}

func TestScrubTmuxEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"TMUX=/tmp/tmux-1000/default,1234,0",
		"TMUX_TMPDIR=/tmp/ambient",
		"TMUX_FOO=keep",
		"AX_TMUX=keep",
	}
	scrubbed := scrubTmuxEnv(env)
	for _, entry := range scrubbed {
		if entry == env[1] || entry == env[2] {
			t.Fatalf("ambient survived scrub: %q", entry)
		}
	}
	if len(scrubbed) != 3 {
		t.Fatalf("scrubbed = %q", scrubbed)
	}
}

// TestHelperProcess re-execs the test binary as a canned child process
// for OSRunner tests: no shell, no tmux, deterministic exit/stdout.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if os.Getenv("HELPER_KILLSELF") == "1" {
		proc, err := os.FindProcess(os.Getpid())
		if err != nil {
			os.Exit(43)
		}
		// The kill signal lands asynchronously: yield until death
		// instead of exiting, so the parent always observes the
		// signal death rather than racing it to the exit.
		for i := 0; i < 1000; i++ {
			_ = proc.Kill()
			time.Sleep(10 * time.Millisecond)
		}
		os.Exit(43)
	}
	if os.Getenv("TMUX") != "" || os.Getenv("TMUX_TMPDIR") != "" {
		os.Exit(42)
	}
	os.Stdout.WriteString(os.Getenv("HELPER_STDOUT"))
	os.Stderr.WriteString(os.Getenv("HELPER_STDERR"))
	code, _ := strconv.Atoi(os.Getenv("HELPER_EXIT"))
	os.Exit(code)
}

func helperCommand(record *[][]string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		*record = append(*record, append([]string{name}, args...))
		return exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
	}
}

func TestOSRunnerReportsExitAsData(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_STDOUT", "out\n")
	t.Setenv("HELPER_EXIT", "3")
	var recorded [][]string
	runner := OSRunner{Command: helperCommand(&recorded)}
	result, err := runner.Run(context.Background(), []string{"tmux", "-S", "sock", "has-session"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 3 || string(result.Stdout) != "out\n" {
		t.Fatalf("result = %+v", result)
	}
	if len(recorded) != 1 || recorded[0][0] != "tmux" {
		t.Fatalf("recorded = %q", recorded)
	}
}

func TestOSRunnerScrubsAmbientInChild(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	t.Setenv("TMUX_TMPDIR", "/tmp/ambient")
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_STDOUT", "")
	t.Setenv("HELPER_EXIT", "0")
	var recorded [][]string
	runner := OSRunner{Command: helperCommand(&recorded)}
	result, err := runner.Run(context.Background(), []string{"tmux", "ls"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 42 {
		t.Fatal("child observed ambient tmux variables")
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit = %d", result.ExitCode)
	}
}

func TestOSRunnerRefusesBadArgv(t *testing.T) {
	runner := OSRunner{}
	if _, err := runner.Run(context.Background(), nil); err == nil {
		t.Fatal("empty argv admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command argv")
	}
	if _, err := runner.Run(context.Background(), []string{"tmux", ""}); err == nil {
		t.Fatal("empty element admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "command argv")
	}
}

func TestOSRunnerCapturesStderr(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_STDOUT", "")
	t.Setenv("HELPER_STDERR", "can't find session: x\n")
	t.Setenv("HELPER_EXIT", "1")
	var recorded [][]string
	runner := OSRunner{Command: helperCommand(&recorded)}
	result, err := runner.Run(context.Background(), []string{"tmux", "-S", "sock", "has-session"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 1 || string(result.Stderr) != "can't find session: x\n" {
		t.Fatalf("result = %+v", result)
	}
}

func TestOSRunnerSignalDeathProvesNothing(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_KILLSELF", "1")
	var recorded [][]string
	runner := OSRunner{Command: helperCommand(&recorded)}
	if result, err := runner.Run(context.Background(), []string{"tmux", "-S", "sock", "has-session"}); err == nil {
		t.Fatalf("killed child reported data: %+v", result)
	}
}

func TestOSRunnerTransportErrorProvesNothing(t *testing.T) {
	runner := OSRunner{Command: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "/nonexistent-ax-binary-xyz", args...)
	}}
	if _, err := runner.Run(context.Background(), []string{"/nonexistent-ax-binary-xyz"}); err == nil {
		t.Fatal("missing binary reported success")
	}
}
