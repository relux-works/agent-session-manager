package gitsnap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// scriptedRunner is the test fake for Runner: exact argv (joined with
// \x00) maps to a queue of results, one consumed per call in order. When
// the queue for a command holds two different outputs, the first call
// observes the first and the re-read observes the second, which is how
// mid-capture mutation is simulated. Every call is appended to calls so
// a test can prove which reads fired inside the consistency window.
type scriptedRunner struct {
	mu      sync.Mutex
	scripts map[string][]GitResult
	calls   []string
}

func newScriptedRunner() *scriptedRunner {
	return &scriptedRunner{scripts: map[string][]GitResult{}}
}

func scriptKey(args []string) string {
	return strings.Join(args, "\x00")
}

// Script queues results for one exact argv vector.
func (fake *scriptedRunner) Script(args []string, results ...GitResult) *scriptedRunner {
	fake.scripts[scriptKey(args)] = append(fake.scripts[scriptKey(args)], results...)
	return fake
}

// Rescript replaces the queue for one exact argv vector. Baselines must
// be overridden, not extended: Capture consumes in order, so appending a
// hostile output behind the valid one would never reach the gate.
func (fake *scriptedRunner) Rescript(args []string, results ...GitResult) *scriptedRunner {
	fake.scripts[scriptKey(args)] = append([]GitResult{}, results...)
	return fake
}

// remaining reports the unconsumed queue length for one argv vector. A
// fault-injection test asserts zero afterwards to prove the armed fault
// provably fired inside the window it claims to cover.
func (fake *scriptedRunner) remaining(args ...string) int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return len(fake.scripts[scriptKey(args)])
}

func okResult(stdout string) GitResult {
	return GitResult{Stdout: []byte(stdout), ExitCode: 0}
}

func exitResult(code int, stdout string) GitResult {
	return GitResult{Stdout: []byte(stdout), ExitCode: code}
}

func (fake *scriptedRunner) Run(_ context.Context, _ string, args ...string) (GitResult, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	key := scriptKey(args)
	fake.calls = append(fake.calls, key)
	queue := fake.scripts[key]
	if len(queue) == 0 {
		return GitResult{}, fmt.Errorf("no scripted result for git %s", strings.Join(args, " "))
	}
	result := queue[0]
	fake.scripts[key] = queue[1:]
	return result, nil
}

// callCount reports how many times one exact argv vector ran.
func (fake *scriptedRunner) callCount(args ...string) int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	count := 0
	for _, call := range fake.calls {
		if call == scriptKey(args) {
			count++
		}
	}
	return count
}

// scriptBaseline programs the complete valid capture conversation. Every
// value is overridable by re-scripting the same argv after this call,
// because Script appends and Run consumes in order. The index fixture
// carries two sorted stage-0 entries with clear flags.
func scriptBaseline(fake *scriptedRunner) *scriptedRunner {
	const debugAGENTS = "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md\x00" +
		"  ctime: 1:2\n  mtime: 3:4\n  dev: 5\tino: 6\n  uid: 7\tgid: 8\n  size: 6\tflags: 0\n"
	const debugREADME = "100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md\x00" +
		"  ctime: 1:2\n  mtime: 3:4\n  dev: 5\tino: 6\n  uid: 7\tgid: 8\n  size: 5\tflags: 0\n"
	index := debugAGENTS + debugREADME
	fake.Script([]string{"rev-parse", "--path-format=absolute", "--git-path", "index"}, okResult("/tmp/repo/.git/index\n"), okResult("/tmp/repo/.git/index\n"))
	fake.Script([]string{"rev-parse", "--is-inside-work-tree", "--is-bare-repository", "--show-object-format", "--absolute-git-dir", "--git-common-dir", "--is-shallow-repository"},
		okResult("true\nfalse\nsha1\n/tmp/repo/.git\n/tmp/repo/.git\nfalse\n"))
	fake.Script([]string{"rev-parse", "--show-toplevel"}, okResult("/tmp/repo\n"))
	fake.Script([]string{"worktree", "list", "--porcelain"}, okResult("worktree /tmp/repo\nHEAD 602548b4fd46332c934667db9992b8bb00318c88\nbranch refs/heads/feature/ax\n\n"))
	fake.Script([]string{"symbolic-ref", "-q", "HEAD"}, okResult("refs/heads/feature/ax\n"), okResult("refs/heads/feature/ax\n"))
	fake.Script([]string{"rev-parse", "--verify", "--quiet", "HEAD"}, okResult("602548b4fd46332c934667db9992b8bb00318c88\n"))
	fake.Script([]string{"for-each-ref", "--format=%(upstream)", "--", "refs/heads/feature/ax"}, okResult("refs/remotes/origin/feature/ax\n"))
	fake.Script([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/relux/payments-api.git (fetch)\norigin\tssh://git@github.com/relux/payments-api.git (push)\n"))
	fake.Script([]string{"update-index", "--show-index-version"}, okResult("2\n"), okResult("2\n"))
	fake.Script([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	fake.Script([]string{"diff", "--cached", "--raw", "--no-abbrev", "-z"}, okResult(""), okResult(""))
	fake.Script([]string{"diff", "--raw", "--no-abbrev", "-z"}, okResult(""), okResult(""))
	for _, key := range []string{"core.filemode", "core.symlinks"} {
		fake.Script([]string{"config", "--bool", "--get", key}, okResult("true\n"))
	}
	for _, key := range []string{"core.ignorecase", "core.precomposeunicode", "core.sparseCheckout"} {
		fake.Script([]string{"config", "--bool", "--get", key}, exitResult(1, ""))
	}
	fake.Script([]string{"config", "-z", "--name-only", "--get-regexp", `^filter\..*\.required$`}, exitResult(1, ""))
	fake.Script([]string{"rev-parse", "--verify", "--quiet", "HEAD"}, okResult("602548b4fd46332c934667db9992b8bb00318c88\n"))
	return fake
}

// requireRefusal fails unless err is a *Refusal naming wantGate.
func requireRefusal(t *testing.T, err error, wantGate Gate) *Refusal {
	t.Helper()
	if err == nil {
		t.Fatalf("want refusal %s, got nil error", wantGate.Name)
	}
	var refusal *Refusal
	if !errors.As(err, &refusal) {
		t.Fatalf("want *Refusal %s, got %T (%v)", wantGate.Name, err, err)
	}
	if refusal.Gate != wantGate.Name || refusal.Code != wantGate.Code {
		t.Fatalf("want %s/%s, got %s/%s (%s)", wantGate.Name, wantGate.Code, refusal.Gate, refusal.Code, refusal.Detail)
	}
	return refusal
}
