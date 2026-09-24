package tmuxserver

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// RunResult is one completed process execution: the exit code and both
// streams are data, never an error. A nonzero exit alone proves
// nothing: tmux reports every failure with exit 1 (verified on 3.6a),
// so absence is proven only by an exit paired with its absence-shaped
// stderr (see classifyAbsence); only a transport failure — the process
// never ran, or died by signal — returns an error, and an error proves
// nothing, not even non-commit.
type RunResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner executes one argv vector to completion. Production callers pass
// vectors built by BuildCommand only; tests inject fakes. No Runner
// implementation in the test suite starts a tmux process.
type Runner interface {
	Run(ctx context.Context, argv []string) (RunResult, error)
}

// OSRunner is the production Runner: native process start with no
// shell. The child environment scrubs the two ambient tmux variables
// (EnvTMUX, EnvTMUXTmpDir) so an ambient session can never leak into a
// -S-addressed invocation; -S addressing already ignores them, and the
// scrub makes that independence structural rather than relied-upon.
type OSRunner struct {
	// Command builds the process. Nil means exec.CommandContext.
	// Tests inject a helper-process fake; production leaves it nil.
	Command func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// Run executes argv and reports its exit code as data. A vector that
// fails the landed argv shape gate refuses before any process starts.
func (runner OSRunner) Run(ctx context.Context, argv []string) (RunResult, error) {
	if err := secprim.CheckArgv(argv); err != nil {
		return RunResult{}, &Error{Code: CodeInvalidArguments, Detail: "command argv"}
	}
	command := exec.CommandContext
	if runner.Command != nil {
		command = runner.Command
	}
	proc := command(ctx, argv[0], argv[1:]...)
	proc.Env = scrubTmuxEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	proc.Stdout = &stdout
	proc.Stderr = &stderr
	if err := proc.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// A signal-killed child reports ExitCode -1: the process
			// never answered, so the run is a transport failure
			// (unknown), never exit data. Returning -1 as data
			// would let a killed probe manufacture absence or
			// closure downstream; every absence classifier also
			// refuses negative exits, so both layers fail closed.
			if exitErr.ExitCode() < 0 {
				return RunResult{}, err
			}
			return RunResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: exitErr.ExitCode()}, nil
		}
		return RunResult{}, err
	}
	return RunResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: 0}, nil
}

// scrubTmuxEnv removes the ambient tmux session variables from a process
// environment. It matches exact NAME= prefixes, so TMUX_FOO survives
// while TMUX and TMUX_TMPDIR are dropped.
func scrubTmuxEnv(env []string) []string {
	scrubbed := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, EnvTMUX+"=") || strings.HasPrefix(entry, EnvTMUXTmpDir+"=") {
			continue
		}
		scrubbed = append(scrubbed, entry)
	}
	return scrubbed
}

// Directive is the closed tmux primitive vocabulary this leaf executes.
// Each lifecycle operation maps to a fixed directive sequence (see the
// backend drivers); no operation execs a primitive outside this enum,
// and no primitive addresses any server but the derived -S socket.
type Directive string

// tmux primitives. The string values are the literal tmux(1) command
// names verified against tmux 3.6a list-commands.
const (
	DirectiveNewSession    Directive = "new-session"
	DirectiveAttachSession Directive = "attach-session"
	DirectiveListPanes     Directive = "list-panes"
	DirectiveListSessions  Directive = "list-sessions"
	DirectiveLockSession   Directive = "lock-session"
	DirectiveDetachClients Directive = "detach-client"
	DirectiveWaitBoundary  Directive = "wait-for"
	DirectiveSendInterrupt Directive = "send-keys"
	DirectiveHasSession    Directive = "has-session"
	DirectiveKillSession   Directive = "kill-session"
)

// ParseDirective admits exactly the ten tmux primitives.
func ParseDirective(value string) (Directive, error) {
	switch Directive(value) {
	case DirectiveNewSession, DirectiveAttachSession, DirectiveListPanes,
		DirectiveListSessions, DirectiveLockSession, DirectiveDetachClients,
		DirectiveWaitBoundary, DirectiveSendInterrupt, DirectiveHasSession,
		DirectiveKillSession:
		return Directive(value), nil
	default:
		return "", &Error{Code: terminalbackend.CodeProtocolError, Detail: "command vocabulary"}
	}
}

// PanesFormat is the list-panes row shape: the foreground command name
// only. It deliberately excludes pane_pid: identity MUST NOT be a PID
// (pinned SPEC 4.B), so provider presence is proven by a non-empty
// command row, never by a process number.
const PanesFormat = "#{pane_current_command}"

// SessionsFormat is the list-sessions row shape: session name and
// attached-client count, pipe-separated.
const SessionsFormat = "#{session_name}|#{session_attached}"

// BoundaryChannelPrefix prefixes every wait-for channel this leaf
// waits on. The channel is ax-boundary-<quiescence_generation UUIDv7>:
// fixed prefix plus caller-bound generation, never caller free text.
// The observation runs the bare `wait-for <channel>` form, which
// blocks until the wrapper wakes it with `wait-for -S`: the `-L` form
// acquires a channel lock and succeeds immediately on a fresh channel
// without any wrapper signal, so it can never evidence a boundary.
const BoundaryChannelPrefix = "ax-boundary-"

// CommandArgs carries the per-directive identity material. Only the
// members the directive reads are validated; the rest stay zero. Every
// tmux target session name is the terminal instance ID: the one
// dedicated server hosts every logical session, so targets are
// instance-scoped, while the wrapper tail always names the logical
// session (ax pane <session_id>). The acquisition bootstrap vector
// (BuildArgv, session-named) and lifecycle instance vectors coexist as
// distinct sessions on that server; lifecycle entries address instances
// only.
type CommandArgs struct {
	// RuntimeDir is the tmux runtime directory (<root>/tmux).
	RuntimeDir string
	// Socket must equal SocketPath(RuntimeDir): the only admissible
	// server address. Any other value refuses.
	Socket string
	// SessionID is the logical session (create/restore wrapper tail).
	SessionID string
	// InstanceID is the tmux target session name.
	InstanceID string
	// QuiescenceGeneration selects the wait-for channel (boundary).
	QuiescenceGeneration string
	// AttachInput carries the attach input authorization for the
	// attach-session vector: false emits the read-only `-r` client
	// (only detach/switch keys have any effect), true emits the
	// writable vector. HasAttachInput must be set whenever the
	// attach directive builds: an unstated authorization refuses
	// instead of defaulting to either vector.
	AttachInput    bool
	HasAttachInput bool
}

// BuildCommand builds one validated tmux -S argv vector for a
// directive. The socket pin mirrors the BuildArgv spawn-socket arm:
// the socket must be the one derived from the runtime directory, so an
// ambient socket passed here refuses instead of being addressed. The
// assembled vector passes the landed argv shape gate before return.
func BuildCommand(directive Directive, args CommandArgs) ([]string, error) {
	if _, err := ParseDirective(string(directive)); err != nil {
		return nil, err
	}
	if args.Socket != SocketPath(args.RuntimeDir) {
		return nil, &Error{Code: CodeAmbientReuse, Detail: "command socket"}
	}
	var argv []string
	switch directive {
	case DirectiveNewSession:
		session, instance, err := commandSessionInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "new-session", "-d",
			"-s", instance, "ax", "pane", session}
	case DirectiveAttachSession:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		if !args.HasAttachInput {
			return nil, &Error{Code: CodeInvalidArguments, Detail: "command attach authorization"}
		}
		argv = []string{"tmux", "-S", args.Socket, "attach-session", "-t", instance}
		if !args.AttachInput {
			argv = []string{"tmux", "-S", args.Socket, "attach-session", "-r", "-t", instance}
		}
	case DirectiveListPanes:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "list-panes", "-t", instance, "-F", PanesFormat}
	case DirectiveListSessions:
		argv = []string{"tmux", "-S", args.Socket, "list-sessions", "-F", SessionsFormat}
	case DirectiveLockSession:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "lock-session", "-t", instance}
	case DirectiveDetachClients:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "detach-client", "-s", instance}
	case DirectiveWaitBoundary:
		channel, err := commandChannel(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "wait-for", channel}
	case DirectiveSendInterrupt:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "send-keys", "-t", instance, "C-c"}
	case DirectiveHasSession:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "has-session", "-t", instance}
	case DirectiveKillSession:
		instance, err := commandInstance(args)
		if err != nil {
			return nil, err
		}
		argv = []string{"tmux", "-S", args.Socket, "kill-session", "-t", instance}
	default:
		return nil, &Error{Code: terminalbackend.CodeProtocolError, Detail: "command vocabulary"}
	}
	if err := secprim.CheckArgv(argv); err != nil {
		return nil, &Error{Code: CodeInvalidArguments, Detail: "command argv"}
	}
	return argv, nil
}

// commandInstance validates the tmux target: the terminal instance ID
// as canonical UUIDv7.
func commandInstance(args CommandArgs) (string, error) {
	instance, err := scalar.ParseUUIDv7(args.InstanceID)
	if err != nil {
		return "", &Error{Code: CodeInvalidArguments, Detail: "command instance"}
	}
	return instance.String(), nil
}

// commandSessionInstance validates the create/restore pair: the logical
// session for the wrapper tail and the instance for the target name.
func commandSessionInstance(args CommandArgs) (string, string, error) {
	session, err := scalar.ParseUUIDv7(args.SessionID)
	if err != nil {
		return "", "", &Error{Code: CodeInvalidArguments, Detail: "command session"}
	}
	instance, err := commandInstance(args)
	if err != nil {
		return "", "", err
	}
	return session.String(), instance, nil
}

// commandChannel derives the wait-for channel from the quiescence
// generation: fixed prefix plus canonical UUIDv7, never caller text.
func commandChannel(args CommandArgs) (string, error) {
	generation, err := scalar.ParseUUIDv7(args.QuiescenceGeneration)
	if err != nil {
		return "", &Error{Code: CodeInvalidArguments, Detail: "command quiescence"}
	}
	return BoundaryChannelPrefix + generation.String(), nil
}
