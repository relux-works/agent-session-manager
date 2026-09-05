package provhost

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// futureDeadline returns a deadline duration ahead of the real clock,
// so Host deadline accounting runs against live time.
func futureDeadline(t *testing.T, ahead time.Duration) scalar.Timestamp {
	t.Helper()
	parsed, err := scalar.ParseTimestamp(time.Now().Add(ahead).UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	return parsed
}

// liveRequest is testRequest with a live future deadline for Host calls,
// which enforce the deadline against a real context.
func liveRequest(t *testing.T) Request {
	t.Helper()
	req := testRequest(t)
	req.Deadline = futureDeadline(t, 5*time.Minute)
	return req
}

// liveHost builds a Host on the real clock.
func liveHost(runner *scriptRunner) Host {
	return Host{Runner: runner, Now: time.Now}
}

// scriptCall records one Runner invocation.
type scriptCall struct {
	executable string
	stdin      []byte
}

// scriptRunner is a scripted Runner: it replays canned results, records
// every invocation, and optionally blocks until the context ends to
// simulate a hung plugin. It proves Host behavior without real processes;
// the ExecRunner integration test below drives the production call site.
// scriptStep is one canned invocation outcome: either a result or a
// transport error, consumed in listed order.
type scriptStep struct {
	result Result
	err    error
}

func okStep(result Result) scriptStep { return scriptStep{result: result} }

func failStep(err error) scriptStep { return scriptStep{err: err} }

type scriptRunner struct {
	mu    sync.Mutex
	calls []scriptCall
	steps []scriptStep
	block bool
	// delay sleeps past the request deadline before replaying, so the
	// runner returns a result after the deadline instead of an error.
	delay time.Duration
}

func (runner *scriptRunner) Run(ctx context.Context, executable string, stdin []byte) (Result, error) {
	runner.mu.Lock()
	runner.calls = append(runner.calls, scriptCall{executable: executable, stdin: append([]byte(nil), stdin...)})
	index := len(runner.calls) - 1
	runner.mu.Unlock()
	if runner.block {
		<-ctx.Done()
		return Result{}, ctx.Err()
	}
	if runner.delay > 0 {
		time.Sleep(runner.delay)
	}
	if index < len(runner.steps) {
		step := runner.steps[index]
		return step.result, step.err
	}
	return Result{}, errors.New("script: no step canned")
}

func (runner *scriptRunner) spawned() int {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	return len(runner.calls)
}

func successResult(t *testing.T, id, body string) Result {
	t.Helper()
	return Result{Stdout: successFrame(t, id, body), ExitCode: 0}
}

// TestCallSendsOneFrameToOneProcess proves the dispatch path: the exact
// request frame reaches exactly one process of the named executable.
func TestCallSendsOneFrameToOneProcess(t *testing.T) {
	runner := &scriptRunner{steps: []scriptStep{okStep(successResult(t, testRequestID, `{"provider_id":"pi"}`))}}
	host := liveHost(runner)
	req := liveRequest(t)
	got, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if string(got.Body) != `{"provider_id":"pi"}` {
		t.Fatalf("Call body = %s", got.Body)
	}
	if runner.spawned() != 1 {
		t.Fatalf("Call spawned %d processes, want exactly one per operation", runner.spawned())
	}
	call := runner.calls[0]
	if call.executable != "/plugins/ax-provider-pi" {
		t.Fatalf("Call executable = %q", call.executable)
	}
	frame, err := EncodeRequest(req, time.Now())
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	if string(call.stdin) != string(frame) {
		t.Fatalf("Call stdin = %q, want the single request frame", call.stdin)
	}
}

// TestCallUnknownOperationStartsNoProcess proves dispatch refuses an
// unregistered operation locally: the failure is invalid_config and the
// Runner is never touched.
func TestCallUnknownOperationStartsNoProcess(t *testing.T) {
	runner := &scriptRunner{}
	host := liveHost(runner)
	req := liveRequest(t)
	req.Operation = "reboot"
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	requireLocalRefusal(t, err, "invalid_config", "unknown operation")
	if runner.spawned() != 0 {
		t.Fatalf("Call spawned %d processes for an unknown operation, want none", runner.spawned())
	}
}

// TestCallNilRunnerRefuses proves a host without a Runner fails closed
// before framing.
func TestCallNilRunnerRefuses(t *testing.T) {
	host := Host{Now: time.Now}
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
	requireLocalRefusal(t, err, "invalid_config", "host has no runner")
}

// TestCallUsesSystemClockWhenUnset proves the nil-clock branch: a Host
// without Now enforces live deadlines.
func TestCallUsesSystemClockWhenUnset(t *testing.T) {
	runner := &scriptRunner{steps: []scriptStep{okStep(successResult(t, testRequestID, `{"provider_id":"pi"}`))}}
	host := Host{Runner: runner}
	req := testRequest(t)
	req.Deadline = futureDeadline(t, 5*time.Minute)
	if _, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req); err != nil {
		t.Fatalf("Call with unset clock: %v", err)
	}
}

// failingReader always errors, exercising the capped-read failure path.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("fake: read failed") }

// TestReadCappedReportsStreamErrors proves a stream failure surfaces
// instead of an empty capture.
func TestReadCappedReportsStreamErrors(t *testing.T) {
	if _, err := readCapped(failingReader{}, 1024); err == nil {
		t.Fatal("readCapped swallowed a stream error")
	}
}

// TestReadCappedEnforcesCap proves the memory bound runner.go promises:
// a stream far beyond the cap is cut to exactly cap bytes carrying the
// stream prefix, while under-cap and exact-cap streams pass through
// untouched. Removing the limiter returns the whole stream; removing the
// truncation returns cap+1 bytes; both redden here.
func TestReadCappedEnforcesCap(t *testing.T) {
	const cap = 16
	over := strings.Repeat("a", 4*cap) + strings.Repeat("b", 4*cap)
	got, err := readCapped(strings.NewReader(over), cap)
	if err != nil {
		t.Fatalf("readCapped: %v", err)
	}
	if len(got) != cap {
		t.Fatalf("readCapped cut %d input bytes to %d, want exactly %d", len(over), len(got), cap)
	}
	if string(got) != strings.Repeat("a", cap) {
		t.Fatalf("readCapped kept %q, want the stream prefix", got)
	}
	for _, size := range []int{0, 1, cap - 1, cap} {
		want := strings.Repeat("c", size)
		got, err := readCapped(strings.NewReader(want), cap)
		if err != nil {
			t.Fatalf("readCapped(%d): %v", size, err)
		}
		if string(got) != want {
			t.Fatalf("readCapped(%d) = %q, want passthrough", size, got)
		}
	}
}

// errAfterReader yields filler bytes up to a budget, then fails. It proves
// readCapped stops consuming at cap+1: with the limiter the error past the
// cap is never reached, while a limiter removed (reading the whole stream
// into memory first) surfaces it.
type errAfterReader struct {
	remaining int
	filler    byte
}

func (reader *errAfterReader) Read(into []byte) (int, error) {
	if reader.remaining <= 0 {
		return 0, errors.New("fake: past the cap")
	}
	n := len(into)
	if n > reader.remaining {
		n = reader.remaining
	}
	for i := 0; i < n; i++ {
		into[i] = reader.filler
	}
	reader.remaining -= n
	return n, nil
}

// TestReadCappedStopsConsumingAtTheCap proves the limiter half of the
// memory bound: a stream that fails past four caps is still cut cleanly,
// because reads stop one byte past the cap instead of draining the
// stream. Removing the limiter surfaces the stream error here.
func TestReadCappedStopsConsumingAtTheCap(t *testing.T) {
	const cap = 16
	got, err := readCapped(&errAfterReader{remaining: 4 * cap, filler: 'd'}, cap)
	if err != nil {
		t.Fatalf("readCapped reached past the cap: %v", err)
	}
	if string(got) != strings.Repeat("d", cap) {
		t.Fatalf("readCapped kept %q, want %d prefix bytes", got, cap)
	}
}

// TestCallTimesOutHungPlugin proves the deadline path: a plugin that
// never answers is terminated at the request deadline and reported as
// provider_timeout with exit 13.
func TestCallTimesOutHungPlugin(t *testing.T) {
	runner := &scriptRunner{block: true}
	host := liveHost(runner)
	req := liveRequest(t)
	req.Deadline = futureDeadline(t, 150*time.Millisecond)
	start := time.Now()
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Call took %v, want deadline enforcement near 100ms", elapsed)
	}
	requireLocalRefusal(t, err, "provider_timeout", "no response before the request deadline")
	if failureExit(t, err) != 13 {
		t.Fatalf("Call exit = %d, want 13", failureExit(t, err))
	}
}

// TestCallEmptyResultAfterDeadlineIsTimeout proves the late-empty
// path: a runner that answers with nothing after the deadline still
// reports provider_timeout, even though the process exited on its own.
func TestCallEmptyResultAfterDeadlineIsTimeout(t *testing.T) {
	runner := &scriptRunner{steps: []scriptStep{okStep(Result{ExitCode: 1})}, delay: 400 * time.Millisecond}
	host := liveHost(runner)
	req := liveRequest(t)
	req.Deadline = futureDeadline(t, 150*time.Millisecond)
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	requireLocalRefusal(t, err, "provider_timeout", "no response before the request deadline")
}

// TestCallRunnerErrorBeforeDeadlineIsProcessFailure proves a transport
// failure with time remaining is provider_process_failed, not a timeout.
func TestCallRunnerErrorBeforeDeadlineIsProcessFailure(t *testing.T) {
	runner := &scriptRunner{steps: []scriptStep{failStep(errors.New("fake: fork failed"))}}
	host := liveHost(runner)
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
	requireLocalRefusal(t, err, "provider_process_failed", "runner reported no result")
}

// TestCallParentCancelIsNotTimeout proves a caller abort with time
// remaining is provider_process_failed: the deadline never passed, so no
// timeout may be claimed.
func TestCallParentCancelIsNotTimeout(t *testing.T) {
	runner := &scriptRunner{block: true}
	host := liveHost(runner)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := host.Call(ctx, "/plugins/ax-provider-pi", liveRequest(t))
	requireLocalRefusal(t, err, "provider_process_failed", "runner reported no result")
}

// TestCallMissingResponse proves an empty stdout is a missing response:
// provider_process_failed on a crash exit, provider_protocol_error when
// the plugin claimed success by exiting 0.
func TestCallMissingResponse(t *testing.T) {
	for _, kase := range []struct {
		name   string
		code   int
		want   string
		member string
	}{
		{"crash exit", 3, "provider_process_failed", ""},
		{"signal-like exit", 137, "provider_process_failed", ""},
		{"clean exit", 0, "provider_protocol_error", ""},
	} {
		t.Run(kase.name, func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{okStep(Result{ExitCode: kase.code})}}
			host := liveHost(runner)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			if kase.want == "provider_protocol_error" {
				requireFrameRefusal(t, err, kase.member, "plugin exited without a response")
			} else {
				requireLocalRefusal(t, err, axerror.Code(kase.want), "plugin exited without a response")
			}
		})
	}
}

// TestCallRefusesUnusableStdout proves crashes, garbage, and
// correlation breaks map to the Section 7.2 host-failure classes. The
// production entry point is Host.Call.
func TestCallRefusesUnusableStdout(t *testing.T) {
	for _, kase := range []struct {
		name   string
		result Result
		want   string
		member string
		detail string
	}{
		{"garbage", Result{Stdout: []byte("not json\n"), Stderr: []byte("traceback\n"), ExitCode: 1}, "provider_protocol_error", "", "not a JSON object"},
		{"crash no output", Result{ExitCode: 1}, "provider_process_failed", "", "plugin exited without a response"},
		{"wrong request id", Result{Stdout: successFrame(t, testOtherID, `{}`), ExitCode: 0}, "provider_protocol_error", "request_id", "request_id does not match the request"},
		{"extra line", Result{Stdout: append(successFrame(t, testRequestID, `{}`), '\n', 'x', '\n'), ExitCode: 0}, "provider_protocol_error", "", "stdout carries more than one frame"},
		{"foreign major", Result{Stdout: []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"3.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`), ExitCode: 0}, "incompatible_protocol", "", ""},
	} {
		t.Run(kase.name, func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{okStep(kase.result)}}
			host := liveHost(runner)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			switch kase.want {
			case "provider_protocol_error":
				requireFrameRefusal(t, err, kase.member, kase.detail)
			case "incompatible_protocol":
				if failureCode(t, err) != "incompatible_protocol" {
					t.Fatalf("Call code = %v, want incompatible_protocol", err)
				}
				if observed, ok := failureObject(t, err).Detail("observed"); !ok || observed != "3.0.0" {
					t.Fatalf("Call observed = %v, want the foreign version", observed)
				}
			default:
				requireLocalRefusal(t, err, axerror.Code(kase.want), kase.detail)
			}
		})
	}
}

// TestCallSeparatesStderr proves stdout/stderr separation two ways: a
// valid frame succeeds despite noisy diagnostics, and a failing
// invocation never carries stderr content in its human text.
func TestCallSeparatesStderr(t *testing.T) {
	secret := "sk-fake-secret-in-stderr"
	runner := &scriptRunner{steps: []scriptStep{okStep(Result{
		Stdout:   successFrame(t, testRequestID, `{"provider_id":"pi"}`),
		Stderr:   []byte("diagnostic: using " + secret + "\n"),
		ExitCode: 0,
	})}}
	host := liveHost(runner)
	if _, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t)); err != nil {
		t.Fatalf("Call with stderr diagnostics: %v", err)
	}

	bad := &scriptRunner{steps: []scriptStep{okStep(Result{Stdout: []byte("garbage\n"), Stderr: []byte(secret + "\n"), ExitCode: 1})}}
	host = liveHost(bad)
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
	requireFrameRefusal(t, err, "", "not a JSON object")
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("stderr content leaked into the failure text: %v", err)
	}
}

// TestStdoutCapIsPinned pins the stdout capture bound in both
// directions: one maximal frame, its line terminator, and one probe
// byte. Narrowing it to MaxFrameBytes+1 makes the host accept a maximal
// frame followed by one junk byte (behaviorally proven by
// TestExecRunnerRefusesMaximalFramePlusJunk); widening it holds 4x the
// memory bound with the suite green. Neither direction is observable
// from the scripted runner, which never passes through readCapped, so
// the constant itself is asserted here.
func TestStdoutCapIsPinned(t *testing.T) {
	const specStdoutCapBytes = (8 << 20) + 2
	if stdoutCap != specStdoutCapBytes {
		t.Fatalf("stdoutCap = %d, want the specified %d", stdoutCap, specStdoutCapBytes)
	}
	if stdoutCap != MaxFrameBytes+2 {
		t.Fatalf("stdoutCap = %d, want one maximal frame plus terminator plus probe byte", stdoutCap)
	}
}

// TestExecRunnerRefusesMaximalFramePlusJunk proves the probe byte is
// load-bearing through the production path: a real plugin emitting a
// frame of exactly MaxFrameBytes, a terminator, and one junk byte is
// refused with "stdout carries more than one frame". With the cap
// narrowed by one byte the capture holds only frame plus terminator,
// the junk is cut before splitFrame can see it, and the host accepts an
// 8 MiB body the wire forbids — the narrowing reddens here with err=nil.
// The production entry point is Host.Call over ExecRunner.
func TestExecRunnerRefusesMaximalFramePlusJunk(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	req := liveRequest(t)
	prefix := `{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + req.RequestID.String() + `","ok":true,"body":{"pad":"`
	suffix := `"}}`
	padLen := MaxFrameBytes - len(prefix) - len(suffix)
	if padLen <= 0 {
		t.Fatalf("frame overhead exceeds MaxFrameBytes; cannot build a maximal frame")
	}
	if len(prefix)+padLen+len(suffix) != MaxFrameBytes {
		t.Fatalf("maximal frame is %d bytes, want exactly %d", len(prefix)+padLen+len(suffix), MaxFrameBytes)
	}
	script := filepath.Join(t.TempDir(), "ax-provider-maxframe")
	plugin := "#!/bin/sh\ncat > /dev/null\nprintf '%s' \"$AX_FRAME_PREFIX\"\nyes a | tr -d '\\n' | head -c \"$AX_PAD_LEN\"\nprintf '%s' \"$AX_FRAME_SUFFIX\"\nprintf '\\nX'\nexit 0\n"
	if err := os.WriteFile(script, []byte(plugin), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("AX_FRAME_PREFIX", prefix)
	t.Setenv("AX_FRAME_SUFFIX", suffix)
	t.Setenv("AX_PAD_LEN", strconv.Itoa(padLen))
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	_, err := host.Call(context.Background(), script, req)
	requireFrameRefusal(t, err, "", "stdout carries more than one frame")
}

// TestCallOversizeStdoutRefuses proves a first line one byte over 8 MiB
// fails as provider_protocol_error through the full Call path.
func TestCallOversizeStdoutRefuses(t *testing.T) {
	pad := strings.Repeat("a", specFrameLimitBytes)
	frame := successFrame(t, testRequestID, `{"pad":"`+pad+`"}`)
	if len(frame) <= specFrameLimitBytes {
		t.Fatalf("oversize stdout is %d bytes, want over %d", len(frame), specFrameLimitBytes)
	}
	runner := &scriptRunner{steps: []scriptStep{okStep(Result{Stdout: frame, ExitCode: 0})}}
	host := liveHost(runner)
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
	requireFrameRefusal(t, err, "", "frame exceeds 8 MiB")
}

// TestCallOneProcessPerCall proves the single-flight cross-process
// property: two calls start two processes with distinct request IDs.
func TestCallOneProcessPerCall(t *testing.T) {
	runner := &scriptRunner{steps: []scriptStep{
		okStep(successResult(t, testRequestID, `{}`)),
		okStep(successResult(t, testOtherID, `{}`)),
	}}
	host := liveHost(runner)
	first := liveRequest(t)
	if _, err := host.Call(context.Background(), "/plugins/ax-provider-pi", first); err != nil {
		t.Fatalf("first Call: %v", err)
	}
	second := liveRequest(t)
	second.RequestID = mustUUIDv7(t, testOtherID)
	if _, err := host.Call(context.Background(), "/plugins/ax-provider-pi", second); err != nil {
		t.Fatalf("second Call: %v", err)
	}
	if runner.spawned() != 2 {
		t.Fatalf("two Calls spawned %d processes, want one per operation", runner.spawned())
	}
	if string(runner.calls[0].stdin) == string(runner.calls[1].stdin) {
		t.Fatal("successive calls sent byte-identical frames; request IDs must differ")
	}
}

// TestExecRunnerEchoRoundTrip drives the production call site end to
// end: a real plugin script receives the frame on stdin and answers on
// stdout while logging diagnostics to stderr.
func TestExecRunnerEchoRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	root := t.TempDir()
	script := filepath.Join(root, "ax-provider-echo")
	plugin := `#!/bin/sh
echo "diagnostic: starting" >&2
frame=$(cat)
id=$(printf '%s' "$frame" | sed 's/.*"request_id":"\([^"]*\)".*/\1/')
printf '{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"%s","ok":true,"body":{"provider_id":"echo"}}' "$id"
exit 0
`
	if err := os.WriteFile(script, []byte(plugin), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	got, err := host.Call(context.Background(), script, liveRequest(t))
	if err != nil {
		t.Fatalf("Call through ExecRunner: %v", err)
	}
	if string(got.Body) != `{"provider_id":"echo"}` {
		t.Fatalf("Call body = %s", got.Body)
	}
}

// TestExecRunnerEnforcesDeadline drives the production kill path: a
// plugin that sleeps past the deadline is terminated and reported as
// provider_timeout.
func TestExecRunnerEnforcesDeadline(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	script := filepath.Join(t.TempDir(), "ax-provider-sleep")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	req := liveRequest(t)
	stamp, err := scalar.ParseTimestamp(time.Now().Add(300 * time.Millisecond).UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	req.Deadline = stamp
	start := time.Now()
	_, err = host.Call(context.Background(), script, req)
	elapsed := time.Since(start)
	if elapsed > 20*time.Second {
		t.Fatalf("Call took %v, want termination near the deadline", elapsed)
	}
	// Returning before the drain delay shows the wait did not linger
	// for pipe EOF after the deadline kill. What the group kill buys
	// beyond Setpgid/Kill consistency is unknown on this platform:
	// removing the Cancel alone still returns in ~300ms here, while
	// disabling Setpgid delays past 2s — so this pins termination
	// latency, not detached-descendant fate.
	if elapsed >= waitDrainDelay {
		t.Fatalf("Call took %v, want group termination well before the %v drain delay", elapsed, waitDrainDelay)
	}
	requireLocalRefusal(t, err, "provider_timeout", "no response before the request deadline")
}

// TestExecRunnerReportsCrashExit drives the production exit-code path: a
// real plugin that exits 3 without answering is provider_process_failed,
// while one that exits 0 without answering is provider_protocol_error.
// Discarding ExitCode in ExecRunner promotes the crash to a protocol
// error, and the promotion reddens here. The production entry point is
// Host.Call over ExecRunner.
func TestExecRunnerReportsCrashExit(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	for _, kase := range []struct {
		name   string
		script string
		code   string
		detail string
	}{
		{"crash exit", "#!/bin/sh\ncat > /dev/null\nexit 3\n", "provider_process_failed", "plugin exited without a response"},
		{"clean exit", "#!/bin/sh\ncat > /dev/null\nexit 0\n", "provider_protocol_error", "plugin exited without a response"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			plugin := filepath.Join(t.TempDir(), "ax-provider-exit")
			if err := os.WriteFile(plugin, []byte(kase.script), 0o700); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			host := Host{Runner: ExecRunner{}, Now: time.Now}
			_, err := host.Call(context.Background(), plugin, liveRequest(t))
			if kase.code == "provider_protocol_error" {
				requireFrameRefusal(t, err, "", kase.detail)
			} else {
				requireLocalRefusal(t, err, axerror.Code(kase.code), kase.detail)
			}
		})
	}
}

// TestExecRunnerKeepsValidFrameWhenPluginIgnoresStdin proves the F2
// rule through the production entry point: a plugin that answers a
// valid correlated frame without ever reading stdin — the shape
// ExecRunner's own comment anticipates — has every call accepted,
// even when the host-side stdin write races the plugin's exit into
// EPIPE. Returning writeErr before stdout is ever considered
// discards those frames as "runner reported no result"; the loop
// below fails on that build because a single racing call in two
// hundred is near-certain under load. The production entry point is
// Host.Call over ExecRunner.
func TestExecRunnerKeepsValidFrameWhenPluginIgnoresStdin(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	req := liveRequest(t)
	script := filepath.Join(t.TempDir(), "ax-provider-no-stdin-read")
	frame := `{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + req.RequestID.String() + `","ok":true,"body":{"provider_id":"echo"}}`
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s' \"$AX_VALID_FRAME\"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("AX_VALID_FRAME", frame)
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	const calls = 200
	discarded := 0
	for i := 0; i < calls; i++ {
		got, err := host.Call(context.Background(), script, req)
		if err != nil {
			discarded++
			t.Logf("call %d discarded its valid frame: %v", i, err)
			continue
		}
		if string(got.Body) != `{"provider_id":"echo"}` {
			t.Fatalf("call %d body = %s, want the answered frame", i, got.Body)
		}
	}
	if discarded != 0 {
		t.Fatalf("ExecRunner discarded %d of %d valid frames from a plugin that never read stdin", discarded, calls)
	}
}

// TestExecRunnerClassifiesCrashExitDeterministically proves the
// crash side of the same rule: a plugin that drains stdin and exits
// 3 without answering is provider_process_failed with "plugin exited
// without a response" on every call, never the "runner reported no
// result" misclassification the EPIPE race produced. Fifty racing
// calls fail on the old build with near certainty. The production
// entry point is Host.Call over ExecRunner.
func TestExecRunnerClassifiesCrashExitDeterministically(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	plugin := filepath.Join(t.TempDir(), "ax-provider-exit")
	if err := os.WriteFile(plugin, []byte("#!/bin/sh\ncat > /dev/null\nexit 3\n"), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	const calls = 50
	for i := 0; i < calls; i++ {
		_, err := host.Call(context.Background(), plugin, liveRequest(t))
		requireLocalRefusal(t, err, "provider_process_failed", "plugin exited without a response")
	}
}

// TestExecRunnerWritesOneTerminatedFrame drives the production stdin
// path: the exact request frame plus the Section 7.2 line terminator must
// reach the plugin. The script saves its stdin aside and answers from it,
// so the captured bytes pin what ExecRunner wrote. Omitting the newline
// reddens the byte comparison here.
func TestExecRunnerWritesOneTerminatedFrame(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	root := t.TempDir()
	capture := filepath.Join(root, "stdin-capture")
	script := filepath.Join(root, "ax-provider-capture")
	plugin := `#!/bin/sh
cat > "$AX_STDIN_CAPTURE"
frame=$(cat "$AX_STDIN_CAPTURE")
id=$(printf '%s' "$frame" | sed 's/.*"request_id":"\([^"]*\)".*/\1/')
printf '{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"%s","ok":true,"body":{"provider_id":"echo"}}' "$id"
exit 0
`
	if err := os.WriteFile(script, []byte(plugin), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("AX_STDIN_CAPTURE", capture)
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	req := liveRequest(t)
	if _, err := host.Call(context.Background(), script, req); err != nil {
		t.Fatalf("Call through ExecRunner: %v", err)
	}
	frame, err := EncodeRequest(req, time.Now())
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	want := append(append([]byte(nil), frame...), '\n')
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("ReadFile capture: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("plugin stdin is %d bytes ending %q, want the %d-byte frame plus one newline", len(got), lastByte(got), len(want))
	}
}

// lastByte renders the final stdin byte for the terminator failure text.
func lastByte(data []byte) string {
	if len(data) == 0 {
		return "empty"
	}
	return string(data[len(data)-1:])
}

// TestExecRunnerDetachedDescendantCannotHoldCallPastDeadline proves the
// deadline guarantee against a plugin that answers and then leaves a
// detached grandchild holding stdout open: the grandchild escapes the
// deadline process-group kill with setsid, so a drain that waits for
// pipe EOF would stay open for the grandchild's whole lifetime while
// the answered frame sits complete in the pipe. The call must still
// end at the request deadline with provider_timeout. Waiting on the
// drain instead returns the accepted frame after the grandchild's
// lifetime (measured 6s against sleep 6, 25s against sleep 25). The
// production entry point is Host.Call over ExecRunner.
func TestExecRunnerDetachedDescendantCannotHoldCallPastDeadline(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH; cannot run the production runner")
	}
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("no perl on PATH; cannot detach a grandchild past the group kill")
	}
	req := liveRequest(t)
	stamp, err := scalar.ParseTimestamp(time.Now().Add(2 * time.Second).UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	req.Deadline = stamp
	frame := `{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + req.RequestID.String() + `","ok":true,"body":{"provider_id":"echo"}}`
	// The plugin answers at once, then forks a setsid-detached
	// grandchild that inherits the stdout pipe and sleeps past the
	// deadline, and exits. Only the detached grandchild still writes
	// to the pipe after the deadline kill. Perl starts in
	// milliseconds, so the detach always wins the race against the
	// seconds-away deadline kill by orders of magnitude: a slow
	// interpreter start here would let the kill land before the
	// setsid and pass vacuously.
	plugin := "#!/bin/sh\nprintf '%s\\n' \"$AX_DETACHED_FRAME\"\nperl -e 'use POSIX setsid; exit if fork(); setsid(); sleep($ARGV[0]);' \"$AX_DETACHED_SLEEP\" &\nwait\nexit 0\n"
	script := filepath.Join(t.TempDir(), "ax-provider-detached")
	if err := os.WriteFile(script, []byte(plugin), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("AX_DETACHED_FRAME", frame)
	// The sleeper outlives any legitimate deadline path by an order
	// of magnitude: reaching it means the drain held the call open.
	t.Setenv("AX_DETACHED_SLEEP", "20")
	host := Host{Runner: ExecRunner{}, Now: time.Now}
	start := time.Now()
	_, err = host.Call(context.Background(), script, req)
	elapsed := time.Since(start)
	if elapsed >= 10*time.Second {
		t.Fatalf("Call took %v, want deadline enforcement near 2s, not the detached grandchild lifetime", elapsed)
	}
	requireLocalRefusal(t, err, "provider_timeout", "no response before the request deadline")
}
