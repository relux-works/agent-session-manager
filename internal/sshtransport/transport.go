// Package sshtransport owns a single foreground OpenSSH stdio process and its
// bounded duplex streams. It does not implement Mesh RPC hello or operations.
// Frames are untrusted protocol input, never authentication or capability facts.
package sshtransport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/peeridentity"
)

const (
	MaxLineBytes = 8 * 1024 * 1024 // Section 11.2, excluding LF.
	StderrLimit  = 64 * 1024       // Local diagnostic budget; never protocol input.
	drainTimeout = time.Second     // Also bounds inherited pipes after direct exit.
)

var (
	ErrLine        = errors.New("SSH RPC line is empty, oversized, or unterminated")
	ErrStderrLimit = errors.New("SSH diagnostic byte budget exceeded")
	ErrStream      = errors.New("SSH stream failed")
	ErrStart       = errors.New("SSH process could not start")
	ErrExit        = errors.New("SSH process exited unsuccessfully")
	ErrClosed      = errors.New("SSH session closed")
	ErrDrain       = errors.New("SSH pipes remained open after process exit")
)

// Failure deliberately excludes argv, raw stderr and operating-system errors
// that may disclose machine-local paths. ExitCode is -1 when no exit was seen.
// OpenSSH exit 255 cannot distinguish authentication from other SSH failures;
// it is never interpreted as an absent peer or as an authenticated empty result.
type Failure struct {
	Kind     error
	ExitCode int
}

func (e *Failure) Error() string { return e.Kind.Error() }
func (e *Failure) Unwrap() error { return e.Kind }

// Client retains the validated config timeout and allowlist snapshot. The zero
// value cannot start a process. New uses the installed OpenSSH from PATH; this
// executable and the local operator's SSH configuration are trusted inputs.
type Client struct {
	directory      peeridentity.Directory
	connectSeconds uint64
	rpcTimeout     time.Duration
	executable     string
	env            []string
}

func New(snapshot config.Snapshot) (*Client, error) {
	directory, err := peeridentity.FromSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	loaded, _ := snapshot.Configuration()
	return &Client{directory: directory, connectSeconds: loaded.Value.Mesh.ConnectTimeoutSeconds,
		rpcTimeout: time.Duration(loaded.Value.Mesh.RPCTimeoutSeconds) * time.Second, executable: "ssh"}, nil
}

// policy precedes configured options because OpenSSH uses the first value.
// Keep external known_hosts/user-key selection, but prohibit implicit host trust,
// reuse of a potentially differently authenticated control socket, forwarding,
// local commands, background persistence and replacement of the fixed command.
func (c *Client) argv(target peeridentity.Target) ([]string, error) {
	args, err := target.RPCArgv()
	if err != nil {
		return nil, err
	}
	policy := []string{
		"StrictHostKeyChecking=yes", "NoHostAuthenticationForLocalhost=no", "VerifyHostKeyDNS=no",
		"UpdateHostKeys=no", "BatchMode=yes", "NumberOfPasswordPrompts=0",
		"ControlMaster=no", "ControlPath=none", "ControlPersist=no",
		"ClearAllForwardings=yes", "ForwardAgent=no", "ForwardX11=no", "Tunnel=no",
		"PermitLocalCommand=no", "RemoteCommand=none", "RequestTTY=no", "SessionType=default",
		"ForkAfterAuthentication=no", "StdinNull=no", "ConnectionAttempts=1",
		"ConnectTimeout=" + strconv.FormatUint(c.connectSeconds, 10),
	}
	out := make([]string, 0, 2*len(policy)+len(args))
	for _, option := range policy {
		out = append(out, "-o", option)
	}
	out = append(out, args...)
	size := 0
	for _, arg := range out {
		size += len(arg)
	}
	if size > 65_536 {
		return nil, peeridentity.ErrSSHArgvTooLarge
	}
	return out, nil
}

// Open resolves the explicit allowlist and starts the fixed remote command via
// the native process API (also on Windows). Success means only process start.
// OpenSSH itself checks the host key before it can run the remote command.
// The RPC owner MUST validate hello, including Target.CheckProtocolHost, before
// admitting operations. No remote shell argument comes from protocol payloads.
// The caller owns Close; its context and configured RPC timeout bound the whole
// session, including blocked reads/writes. It may impose a shorter deadline.
func (c *Client) Open(ctx context.Context, selector string) (*Session, error) {
	target, err := c.directory.Resolve(selector)
	if err != nil {
		return nil, err
	}
	argv, err := c.argv(target)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	life, cancel := context.WithTimeout(ctx, c.rpcTimeout)
	cmd := exec.Command(c.executable, argv...)
	cmd.Env = c.env
	configureProcess(cmd)
	// Own all six pipe endpoints, including failures before Start. exec.Wait
	// must never close a read endpoint while our pump is still draining it.
	var pipes []*os.File
	pipe := func() (*os.File, *os.File, error) {
		r, w, e := os.Pipe()
		if e == nil {
			pipes = append(pipes, r, w)
		}
		return r, w, e
	}
	inR, inW, err := pipe()
	if err == nil {
		var outR, outW, errR, errW *os.File
		outR, outW, err = pipe()
		if err == nil {
			errR, errW, err = pipe()
		}
		if err == nil {
			cmd.Stdin, cmd.Stdout, cmd.Stderr = inR, outW, errW
			if err = cmd.Start(); err == nil {
				_ = inR.Close()
				_ = outW.Close()
				_ = errW.Close()
				s := &Session{target: target, cmd: cmd, ctx: life, cancel: cancel,
					stdin: inW, stdout: outR, stderr: errR, frames: make(chan []byte, 1),
					done: make(chan struct{}), fault: make(chan error, 1)}
				go s.supervise()
				return s, nil
			}
		}
	}
	for _, p := range pipes {
		_ = p.Close()
	}
	cancel()
	return nil, &Failure{Kind: ErrStart, ExitCode: -1}
}

// Session permits one sending goroutine and one receiving goroutine at a time;
// sends are serialized. Close/Wait may run concurrently with either direction.
// At most one queued line plus the current reader line is retained. Stderr is
// counted and discarded, never returned, formatted, logged or persisted.
type Session struct {
	target                peeridentity.Target
	cmd                   *exec.Cmd
	ctx                   context.Context
	cancel                context.CancelFunc
	stdin, stdout, stderr *os.File
	frames                chan []byte
	fault                 chan error
	done                  chan struct{}
	sendMu                sync.Mutex
	result                error // written once, published by closing done
}

// Target is still a plan, not an authenticated identity or successful hello.
func (s *Session) Target() peeridentity.Target { return s.target }
func (s *Session) Wait() error                 { <-s.done; return s.result }
func (s *Session) Close() error                { s.fail(ErrClosed); return s.Wait() }
func (s *Session) fail(err error) {
	select {
	case s.fault <- err:
	default:
	}
}

func (s *Session) Send(line []byte) error {
	if len(line) == 0 || len(line) > MaxLineBytes || bytes.IndexByte(line, '\n') >= 0 {
		s.fail(ErrLine)
		return ErrLine
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	select {
	case <-s.done:
		if s.result != nil {
			return s.result
		}
		return ErrClosed
	default:
	}
	// The caller retains ownership and must not mutate line until Send returns.
	for _, part := range [][]byte{line, {'\n'}} {
		if n, err := s.stdin.Write(part); err != nil || n != len(part) {
			s.fail(ErrStream)
			err := s.Wait()
			if err != nil {
				return err
			}
			return ErrStream
		}
	}
	return nil
}

// CloseWrite signals EOF while allowing pending responses to drain.
func (s *Session) CloseWrite() error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if err := s.stdin.Close(); err != nil {
		return ErrStream
	}
	return nil
}

// Receive returns complete LF-delimited lines only. EOF is returned only after
// a successful exit and both drains; partial/read/exit failures are errors.
// Earlier frames are provisional protocol input: an eventual failure cannot
// retroactively undo caller actions. The RPC layer owns transactional semantics.
func (s *Session) Receive() ([]byte, error) {
	line, ok := <-s.frames
	if ok {
		select {
		case <-s.done:
			if s.result != nil {
				return nil, s.result
			}
		default:
		}
		return line, nil
	}
	if err := s.Wait(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *Session) readLines() error {
	defer close(s.frames)
	reader := bufio.NewReaderSize(s.stdout, 32*1024)
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		complete := len(part) > 0 && part[len(part)-1] == '\n'
		if complete {
			part = part[:len(part)-1]
		}
		if len(line)+len(part) > MaxLineBytes {
			return ErrLine
		}
		line = append(line, part...)
		if complete {
			if len(line) == 0 {
				return ErrLine
			}
			select {
			case s.frames <- line:
				line = nil
			case <-s.ctx.Done():
				return s.ctx.Err()
			}
		}
		if err == nil || errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if len(line) != 0 {
				return ErrLine
			}
			return nil
		}
		return ErrStream
	}
}
func (s *Session) readStderr() error {
	n, err := io.Copy(io.Discard, io.LimitReader(s.stderr, StderrLimit+1))
	if n > StderrLimit {
		return ErrStderrLimit
	}
	if err != nil {
		return ErrStream
	}
	return nil
}

func (s *Session) supervise() {
	stdoutDone, stderrDone := make(chan error, 1), make(chan error, 1)
	waitDone := make(chan error, 1)
	go func() { stdoutDone <- s.readLines() }()
	go func() { stderrDone <- s.readStderr() }()
	go func() { waitDone <- s.cmd.Wait() }()
	var cause error
	var timer *time.Timer
	var expired <-chan time.Time
	exitCode := -1
	for stdoutDone != nil || stderrDone != nil || waitDone != nil {
		select {
		case <-s.ctx.Done():
			cause = s.ctx.Err()
		case cause = <-s.fault:
		case err := <-stdoutDone:
			stdoutDone = nil
			if err != nil {
				cause = err
			}
		case err := <-stderrDone:
			stderrDone = nil
			if err != nil {
				cause = err
			}
		case err := <-waitDone:
			waitDone = nil
			exitCode = s.cmd.ProcessState.ExitCode()
			if err != nil {
				cause = ErrExit
			} else {
				timer = time.NewTimer(drainTimeout)
				expired = timer.C
			}
		case <-expired:
			cause = ErrDrain
		}
		if cause != nil {
			break
		}
	}
	if timer != nil {
		timer.Stop()
	}
	// Stop the entire owned process group on Unix, even if the direct process
	// already exited with inherited pipes. Windows owns the direct child only.
	killProcess(s.cmd)
	s.cancel()
	_ = s.stdin.Close()
	_ = s.stdout.Close()
	_ = s.stderr.Close()
	if waitDone != nil {
		<-waitDone
		exitCode = s.cmd.ProcessState.ExitCode()
	}
	if stdoutDone != nil {
		<-stdoutDone
	}
	if stderrDone != nil {
		<-stderrDone
	}
	if cause != nil {
		s.result = &Failure{Kind: cause, ExitCode: exitCode}
	}
	close(s.done)
}
