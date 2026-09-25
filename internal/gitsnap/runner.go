package gitsnap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitResult retains separate stdout, stderr and exit status. Readers whose
// completeness depends on diagnostics must assess stderr even on exit 0.
type GitResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner starts one Git process for one argument vector. Production code
// supplies ExecGitRunner; tests supply a scripted fake. A Runner moves no
// durable state: every command Capture issues is read-only.
type Runner interface {
	Run(ctx context.Context, dir string, args ...string) (GitResult, error)
}

// ExecGitRunner is the production Runner: it starts the trusted git
// executable from PATH with no shell in between, one process per call,
// under a per-call deadline. The caller provides the repository working
// directory; every command runs with GIT_OPTIONAL_LOCKS=0 so a capture
// does not perform optional index refresh writes. Mandatory writes issued
// by other callers of Run are not prevented by this setting.
type ExecGitRunner struct {
	// GitPath is the executable to start. Empty means "git" from PATH.
	GitPath string
	// Timeout bounds one Git invocation. Zero means DefaultTimeout.
	Timeout time.Duration
}

// DefaultTimeout bounds a single Git subprocess call.
const DefaultTimeout = 30 * time.Second

func (runner ExecGitRunner) executable() string {
	if runner.GitPath != "" {
		return runner.GitPath
	}
	return "git"
}

func (runner ExecGitRunner) timeout() time.Duration {
	if runner.Timeout > 0 {
		return runner.Timeout
	}
	return DefaultTimeout
}

// Run starts one git process and collects its output.
func (runner ExecGitRunner) Run(ctx context.Context, dir string, args ...string) (GitResult, error) {
	return runner.run(ctx, dir, nil, false, args...)
}

// RunInput supplies binary standard input for repository-local pack construction.
func (runner ExecGitRunner) RunInput(ctx context.Context, dir string, input []byte, args ...string) (GitResult, error) {
	return runner.run(ctx, dir, input, false, args...)
}

// RunIsolated runs in a disposable repository without ambient object databases,
// indexes, templates or Git configuration. It is used to verify offline closure.
func (runner ExecGitRunner) RunIsolated(ctx context.Context, dir string, input []byte, args ...string) (GitResult, error) {
	return runner.run(ctx, dir, input, true, args...)
}

func (runner ExecGitRunner) run(ctx context.Context, dir string, input []byte, isolated bool, args ...string) (GitResult, error) {
	deadlineCtx, cancel := context.WithTimeout(ctx, runner.timeout())
	defer cancel()
	env := append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	if isolated {
		env = nil
		for _, item := range os.Environ() {
			if !strings.HasPrefix(item, "GIT_") {
				env = append(env, item)
			}
		}
		env = append(env, "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_OPTIONAL_LOCKS=0")
	}
	env = append(env, "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0")
	argv := args
	if len(args) > 0 && args[0] == "diff" {
		// Git porcelain diff refreshes stat-only entries even with optional
		// locks disabled on some versions. A private index retains Git's exact
		// content comparison without changing the repository's durable index.
		index, err := runner.run(deadlineCtx, dir, nil, isolated, "rev-parse", "--path-format=absolute", "--git-path", "index")
		if err != nil {
			return GitResult{ExitCode: -1}, err
		}
		if index.ExitCode != 0 {
			return index, nil
		}
		source := strings.TrimSuffix(string(index.Stdout), "\n")
		if !filepath.IsAbs(source) {
			return GitResult{ExitCode: -1}, fmt.Errorf("gitsnap: index path is not absolute")
		}
		temporary, err := os.MkdirTemp("", "gitsnap-index-")
		if err != nil {
			return GitResult{ExitCode: -1}, fmt.Errorf("gitsnap: private index directory: %w", err)
		}
		defer os.RemoveAll(temporary)
		target := filepath.Join(temporary, "index")
		if err := copyIndex(source, target); err != nil {
			return GitResult{ExitCode: -1}, err
		}
		env = append(env, "GIT_INDEX_FILE="+target)
		argv = append([]string{"-c", "diff.autoRefreshIndex=true"}, args...)
	}
	command := exec.CommandContext(deadlineCtx, runner.executable(), argv...)
	command.Dir = dir
	// The child inherits the host environment (HOME for git config
	// lookup included) plus the lock guard. URLs that carry secrets are
	// never passed on the command line; Capture reads them back from
	// git output and refuses them through the sanitizer instead.
	command.Env = env
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := GitResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: -1}
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	if err != nil {
		if deadlineCtx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("gitsnap: git %s: %w", strings.Join(args, " "), deadlineCtx.Err())
		}
		if _, ok := err.(*exec.ExitError); ok {
			return result, nil
		}
		return result, fmt.Errorf("gitsnap: git %s: %w", strings.Join(args, " "), err)
	}
	return result, nil
}

// copyIndex preserves an absent index as absence; all other read failures fail.
func copyIndex(source, target string) error {
	input, err := os.Open(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("gitsnap: read index: %w", err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return fmt.Errorf("gitsnap: stat index: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("gitsnap: index is not a regular file")
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("gitsnap: create private index: %w", err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("gitsnap: copy index: %w", copyErr)
	}
	if closeErr != nil {
		return closeErr
	}
	// Git uses the index timestamp to decide whether cached stat data is racy.
	// Preserve it so separate diff copies make the same content/OID decision.
	return os.Chtimes(target, info.ModTime(), info.ModTime())
}
