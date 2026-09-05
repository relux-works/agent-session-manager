package provhost

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

// waitDrainDelay bounds how long Wait lingers for pipe EOF after the
// deadline kill before giving up on detached grandchildren.
const waitDrainDelay = 2 * time.Second

// stdoutCap bounds captured plugin stdout: one maximal frame, its line
// terminator, and one probe byte, so a maximal frame followed by any
// trailing byte is still distinguishable from a clean end. Anything
// beyond is cut; the splitFrame length checks stay exact on the capped
// capture.
const stdoutCap = MaxFrameBytes + 2

// Result is what a plugin process produced: captured stdout and stderr
// and the process exit code. Stdout and stderr are kept separate end to
// end: diagnostics never enter frame parsing, and frame bytes never
// enter diagnostics. Captures are capped (stdout at one frame plus a
// probe byte, stderr at StderrCapBytes); beyond the cap the stream is cut
// and the failure is decided from what was captured, which the length
// checks keep exact.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner starts one plugin process for one operation frame. Production
// code supplies ExecRunner; tests supply a scripted fake. A Runner moves
// bytes and reports process fate; it never parses frames and never
// decides failures.
type Runner interface {
	Run(ctx context.Context, executable string, stdin []byte) (Result, error)
}

// ExecRunner is the production Runner: it starts the trusted executable
// discovered by internal/provider, writes exactly one frame to its stdin,
// closes stdin, and waits under the caller's context. Environment and
// stderr content never enter failure human text: Result carries the raw
// streams, and failures built from a Result carry the failure class and
// member names only, never content.
type ExecRunner struct{}

// Run starts executable with stdin on its stdin and captures both streams
// with caps. A nil error with a nonzero exit code is an ordinary result:
// a syntactically valid frame governs regardless of the exit status,
// because SHOULD exit 0 is not MUST. A non-nil error means no result
// could be judged: start failure, stream failure, context end, or a
// wait failure with empty stdout and no known exit code. In
// particular, a stdin write failure racing the plugin's exit never
// discards a judgeable result on its own.
func (ExecRunner) Run(ctx context.Context, executable string, stdin []byte) (Result, error) {
	command := newCommandContext(ctx, executable)
	command.WaitDelay = waitDrainDelay
	stdinPipe, err := command.StdinPipe()
	if err != nil {
		return Result{}, err
	}
	// Stdout and stderr travel over pipes owned here, not over
	// StdoutPipe/StderrPipe: Wait closes those read ends after seeing
	// the process exit, which races a drain still in flight and
	// surfaces as "file already closed" on a valid frame. A pipe
	// created here is closed only by this function, so a drain never
	// loses to the wait.
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return Result{}, err
	}
	defer stdoutReader.Close()
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		stdoutReader.Close()
		return Result{}, err
	}
	defer stderrReader.Close()
	command.Stdout = stdoutWriter
	command.Stderr = stderrWriter
	if err := command.Start(); err != nil {
		stdoutWriter.Close()
		stderrWriter.Close()
		stdoutReader.Close()
		stderrReader.Close()
		return Result{}, err
	}
	// The parent's copies close at once: the child holds its own, so
	// EOF still means every writer is gone while no later Close can
	// interrupt a drain.
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	// Drain both streams while stdin is written: a plugin that answers
	// without reading stdin would otherwise deadlock against a full
	// pipe once either direction exceeds its buffer.
	stdoutDone := make(chan streamResult, 1)
	stderrDone := make(chan streamResult, 1)
	go func() {
		data, err := readCapped(stdoutReader, stdoutCap)
		stdoutDone <- streamResult{data: data, err: err}
	}()
	go func() {
		data, err := readCapped(stderrReader, StderrCapBytes)
		stderrDone <- streamResult{data: data, err: err}
	}()
	wrote, writeErr := stdinPipe.Write(stdin)
	if writeErr == nil && wrote < len(stdin) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		_, writeErr = stdinPipe.Write([]byte{'\n'})
	}
	closeErr := stdinPipe.Close()
	// The wait and both drains run bounded by the call context. A
	// detached descendant that escapes the deadline group kill holds
	// the pipes open, and anything here that needs pipe EOF would
	// otherwise stay open for the descendant's whole lifetime
	// instead of ending at the deadline. Closing the read ends
	// fails the blocked drains at once; each drain sends exactly
	// once over a buffered channel, so no goroutine is left behind.
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- command.Wait()
	}()
	var waitErr error
	select {
	case waitErr = <-waitDone:
		if ctx.Err() != nil {
			// The wait ended by deadline or cancellation: the
			// kill was already signaled, but a detached
			// grandchild may hold the pipes open past
			// WaitDelay. Close them rather than waiting for
			// streams that can no longer decide anything:
			// with the deadline gone there is no valid
			// response to read.
			stdoutReader.Close()
			stderrReader.Close()
			<-stdoutDone
			<-stderrDone
			return Result{}, ctx.Err()
		}
	case <-ctx.Done():
		stdoutReader.Close()
		stderrReader.Close()
		<-stdoutDone
		<-stderrDone
		if command.Process != nil {
			// The escaper the group kill cannot reach is a
			// grandchild-or-deeper that called setsid itself
			// (see newCommandContext and the
			// detached-descendant test): the direct child is
			// a group leader, so its own setsid fails EPERM
			// and it stays in the group. Kill the direct
			// process all the same: this branch races exec's
			// group Cancel, which may not have fired yet when
			// the deadline is observed here, so the direct
			// signal guarantees the reap below cannot wait on
			// the spawned process itself. An already-reaped
			// process reports an error here that is safe to
			// ignore.
			_ = command.Process.Kill()
		}
		waitErr = <-waitDone
		return Result{}, ctx.Err()
	}
	stdoutResult, stderrResult, drained := collectDrains(ctx, stdoutReader, stderrReader, stdoutDone, stderrDone)
	if !drained {
		// The process exited but a detached descendant still
		// holds a pipe open past the deadline. The frame, if
		// any, was never fully delivered, so the deadline
		// decides: report the timeout rather than holding the
		// call open for the descendant's lifetime.
		return Result{}, ctx.Err()
	}
	if stdoutResult.err != nil {
		return Result{}, stdoutResult.err
	}
	if stderrResult.err != nil {
		return Result{}, stderrResult.err
	}
	stdout := stdoutResult.data
	stderr := stderrResult.data
	// A stdin write failure after the process ran is still a result when
	// the process produced one: the frame may have arrived whole while
	// the write or close raced the plugin's exit. Report write, close,
	// and wait failures only when no result can be judged — empty
	// stdout with no known exit code — otherwise the frame governs,
	// and an empty stdout with a known exit code classifies the crash
	// through the returned Result instead of masking it as a transport
	// failure. Returning writeErr before stdout is ever considered
	// discards valid correlated frames on an EPIPE race.
	exitCode := -1
	if command.ProcessState != nil {
		exitCode = command.ProcessState.ExitCode()
	}
	if exit, ok := waitErr.(*exec.ExitError); ok {
		exitCode = exit.ExitCode()
	}
	result := Result{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}
	if len(stdout) > 0 || command.ProcessState != nil {
		return result, nil
	}
	if writeErr != nil {
		return Result{}, writeErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	if waitErr != nil {
		return Result{}, waitErr
	}
	return result, nil
}

// streamResult is one drained stream: the captured prefix and the
// drain's own error, if the read failed before EOF.
type streamResult struct {
	data []byte
	err  error
}

// collectDrains gathers both stream captures, bounded by the call
// context. When the deadline passes while a detached descendant still
// holds a pipe open, the still-blocked read ends are closed so those
// drains fail at once instead of holding the call open past its
// deadline, and collectDrains reports false: the frame, if any, was
// never fully delivered, so the caller lets the deadline decide
// rather than judging a partial capture. Each drain channel is
// received exactly once per still-pending drain, matching the single
// buffered send of its goroutine.
func collectDrains(ctx context.Context, stdoutReader, stderrReader *os.File, stdoutDone, stderrDone <-chan streamResult) (streamResult, streamResult, bool) {
	select {
	case stdout := <-stdoutDone:
		select {
		case stderr := <-stderrDone:
			return stdout, stderr, true
		case <-ctx.Done():
			stderrReader.Close()
			stderr := <-stderrDone
			return stdout, stderr, false
		}
	case <-ctx.Done():
		stdoutReader.Close()
		stderrReader.Close()
		stdout := <-stdoutDone
		stderr := <-stderrDone
		return stdout, stderr, false
	}
}

// readCapped drains stream up to cap bytes and cuts the rest. Memory
// stays bounded no matter what the plugin emits, and the splitFrame
// length checks stay exact on the capped capture.
func readCapped(stream io.Reader, cap int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(stream, int64(cap)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > cap {
		return data[:cap], nil
	}
	return data, nil
}

// Host invokes provider plugins: one process per operation, one frame in,
// one frame out, under the request deadline. Host keeps no cross-call
// state, so successive calls are separate processes observing durable
// state through the authority the caller passes.
type Host struct {
	// Runner starts the plugin process. It must be set; a nil Runner is
	// a caller error, refused before any frame is built.
	Runner Runner
	// Now is the clock deadlines are checked against. Nil means the
	// system clock; tests supply a fixed instant.
	Now func() time.Time
}

func (host Host) now() time.Time {
	if host.Now == nil {
		return time.Now()
	}
	return host.Now()
}

// splitFrame separates the single response line from anything following
// it. oversize reports a first line beyond MaxFrameBytes; extra reports
// any bytes after the first line terminator. An empty line is neither:
// it is a missing response, judged with the exit code by the caller.
//
// The checks are length-only and exact: stdout capture holds at most
// MaxFrameBytes+2 bytes, so a first line beyond the limit, or any
// trailing byte, is always visible in the captured length. No separate
// truncation flag participates; a narrowed or dropped disjunct here
// would be equivalent, not coverage.
func splitFrame(captured []byte) (line []byte, oversize bool, extra bool) {
	if index := bytes.IndexByte(captured, '\n'); index >= 0 {
		return captured[:index], len(captured[:index]) > MaxFrameBytes, index+1 < len(captured)
	}
	return captured, len(captured) > MaxFrameBytes, false
}

// Call dispatches one operation: it frames the request, starts one plugin
// process, enforces the request deadline, and interprets the single
// response frame. Encode failures return before the Runner is touched, so
// an unknown operation never starts a process. A success envelope returns
// its body; a failure envelope returns the bound child error; anything
// else returns a local failure that trusts nothing from the plugin.
//
// Deadline accounting is exact: the wait ends under a context cut at the
// request deadline, and provider_timeout is reported only when that
// instant actually passed. A parent cancellation first, or any other
// runner failure with time remaining, is provider_process_failed. An
// empty stdout with time remaining is a missing response:
// provider_protocol_error on exit 0, provider_process_failed otherwise.
// An empty stdout after the deadline passed is provider_timeout, and a
// valid frame governs regardless of lateness or exit status.
func (host Host) Call(ctx context.Context, executable string, req Request) (Response, error) {
	if host.Runner == nil {
		failure, err := failInvalid("host has no runner")
		if err != nil {
			return Response{}, err
		}
		return Response{}, failure
	}
	frame, deadline, err := encodeFrame(req, host.now())
	if err != nil {
		return Response{}, err
	}
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	result, err := host.Runner.Run(deadlineCtx, executable, frame)
	if err != nil {
		now := host.now()
		if !now.Before(deadline) || deadlineCtx.Err() == context.DeadlineExceeded {
			failure, fault := failTimeout("no response before the request deadline", deadline.UnixMilli())
			if fault != nil {
				return Response{}, fault
			}
			return Response{}, failure
		}
		failure, fault := failProcess("runner reported no result", err)
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	line, oversize, extra := splitFrame(result.Stdout)
	if oversize {
		failure, fault := failProtocol("frame exceeds 8 MiB", "")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	if extra {
		failure, fault := failProtocol("stdout carries more than one frame", "")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	if len(line) == 0 {
		// No valid response arrived. When the request deadline passed,
		// this is the timeout even if the process also exited badly:
		// the host waited the full allowance and got nothing usable.
		if !host.now().Before(deadline) || deadlineCtx.Err() == context.DeadlineExceeded {
			failure, fault := failTimeout("no response before the request deadline", deadline.UnixMilli())
			if fault != nil {
				return Response{}, fault
			}
			return Response{}, failure
		}
		if result.ExitCode != 0 {
			failure, fault := failProcess("plugin exited without a response", nil)
			if fault != nil {
				return Response{}, fault
			}
			return Response{}, failure
		}
		failure, fault := failProtocol("plugin exited without a response", "")
		if fault != nil {
			return Response{}, fault
		}
		return Response{}, failure
	}
	return DecodeResponse(line, req.RequestID)
}
