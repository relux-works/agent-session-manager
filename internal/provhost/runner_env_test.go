package provhost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// runExecRunnerHelper runs inside the re-execed child (selected by the
// TestMain gate on the marker): it prints the observed environment as one
// marked JSON line and exits without running any test. It never returns.
func runExecRunnerHelper() {
	environment := os.Environ()
	sort.Strings(environment)
	document, err := json.Marshal(environment)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	fmt.Println("SECPROBE-ENV:" + string(document))
	os.Exit(0)
}

// childEnviron starts the test binary as a child through the production
// ExecRunner and returns the environment the child observed. The child
// is the test binary itself restricted to the helper test; its marked
// line is scanned out of stdout because the test framework appends its
// own trailers after the test prints.
func childEnviron(t *testing.T, runner ExecRunner) []string {
	t.Helper()
	result, err := runnerWithHelper(t, runner)
	if err != nil {
		t.Fatalf("helper run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("helper exit = %d, stderr = %q", result.ExitCode, result.Stderr)
	}
	for _, line := range strings.Split(string(result.Stdout), "\n") {
		document, found := strings.CutPrefix(line, "SECPROBE-ENV:")
		if !found {
			continue
		}
		var environment []string
		if err := json.Unmarshal([]byte(document), &environment); err != nil {
			t.Fatalf("helper line %q is not a JSON environment: %v", line, err)
		}
		return environment
	}
	t.Fatalf("helper printed no marked environment line: %q", result.Stdout)
	return nil
}

// runnerWithHelper re-execs the test binary through the production
// runner. The TestMain gate selects the helper by the marker carried in
// the environment itself, because the provider protocol starts plugins
// with no arguments and there is no argv channel for a test flag.
func runnerWithHelper(t *testing.T, runner ExecRunner) (Result, error) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return runner.Run(context.Background(), self, []byte(`{}`))
}

func helperRunner(t *testing.T, env []string) ExecRunner {
	t.Helper()
	t.Setenv("SECPROBE_HELPER_PROCESS", "1")
	return ExecRunner{Env: env}
}

// TestExecRunnerDeliversExactlyTheBuiltEnv proves the allowlist path end
// to end: the operator builds an environment with secprim.BuildEnv, hands
// it to ExecRunner, and the child observes exactly that — including the
// absence of a parent-only marker, which proves no inheritance leak.
func TestExecRunnerDeliversExactlyTheBuiltEnv(t *testing.T) {
	t.Setenv("SECPROBE_PARENT_ONLY", "present")
	// The marker goes up before BuildEnv reads the parent: the built
	// allowlist must carry it, or the child runs the suite instead of
	// the helper.
	runner := helperRunner(t, nil)
	built, err := secprim.BuildEnv(
		[]string{"SECPROBE_HELPER_PROCESS", "PATH"},
		map[string]string{"SECPROBE_LITERAL": "staged"},
		os.LookupEnv,
	)
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}
	runner.Env = built
	observed := childEnviron(t, runner)
	if !reflect.DeepEqual(observed, built) {
		t.Fatalf("child observed %q, want exactly the built %q", observed, built)
	}
	for _, entry := range observed {
		if entry == "SECPROBE_PARENT_ONLY=present" {
			t.Fatal("parent-only variable leaked into the allowlisted child")
		}
	}
}

// TestExecRunnerNilEnvInherits documents the unset policy: a nil Env
// keeps the historical inherit-everything behavior until the operator
// supplies an allowlist. The child must observe the parent-only marker,
// proving inheritance is intact rather than silently emptied.
func TestExecRunnerNilEnvInherits(t *testing.T) {
	t.Setenv("SECPROBE_PARENT_ONLY", "present")
	observed := childEnviron(t, helperRunner(t, nil))
	// The helper marker itself proves the child ran; leak the marker
	// check through membership rather than equality because the suite
	// environment is not fixed.
	found := false
	for _, entry := range observed {
		if entry == "SECPROBE_PARENT_ONLY=present" {
			found = true
		}
	}
	if !found {
		t.Fatal("nil Env did not inherit the parent environment")
	}
}

// TestExecRunnerEmptyEnvIsDenyAll proves the nil-versus-empty allowlist
// boundary the runner documents: a nil Env inherits the parent
// environment (pinned by TestExecRunnerNilEnvInherits), while an empty
// non-nil Env hands the child an exactly empty environment. The child
// is /usr/bin/env, which prints what it observes, so no helper marker
// needs to travel in the environment under test. Weakening the gate to
// len(Env) > 0 silently restores full inheritance for the deny-all
// case; this test fails then because the parent-only marker leaks.
func TestExecRunnerEmptyEnvIsDenyAll(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the env-dump probe is a unix execution check; Windows is compile-and-vet only")
	}
	t.Setenv("SECPROBE_PARENT_ONLY", "present")
	envBin, err := exec.LookPath("env")
	if err != nil {
		t.Skipf("no env binary on PATH: %v", err)
	}
	result, err := ExecRunner{Env: []string{}}.Run(context.Background(), envBin, []byte(`{}`))
	if err != nil {
		t.Fatalf("empty-env run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("empty-env exit = %d, stderr = %q", result.ExitCode, result.Stderr)
	}
	trimmed := strings.TrimSpace(string(result.Stdout))
	if trimmed != "" {
		t.Fatalf("empty Env child observed %q, want an exactly empty environment", trimmed)
	}
}

// TestExecRunnerRefusesUnusableExecutable proves the argv gate at process
// start: an empty or NUL-carrying executable is refused with the stable
// argv refusal instead of a raw exec error.
func TestExecRunnerRefusesUnusableExecutable(t *testing.T) {
	for _, executable := range []string{"", "bad\x00bin"} {
		_, err := ExecRunner{}.Run(context.Background(), executable, []byte(`{}`))
		if err == nil {
			t.Fatalf("Run(%q) started a process", executable)
		}
		if failure, ok := err.(*secprim.Error); !ok {
			t.Fatalf("Run(%q) = %T %v, want *secprim.Error", executable, err, err)
		} else if failure.Kind != secprim.ErrUnsafeArgv {
			t.Fatalf("Run(%q) kind = %v, want ErrUnsafeArgv", executable, failure.Kind)
		}
	}
}
