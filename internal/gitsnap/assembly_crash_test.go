package gitsnap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAssemblyCrashAfterObjectsAndRetry(t *testing.T) {
	t.Parallel()
	if os.Getenv("AX_ASSEMBLY_CRASH_CHILD") == "1" {
		root := os.Getenv("AX_ASSEMBLY_REPO")
		options := assemblyOptions(t, root)
		options.Members[0].Content = openCrashStore(t, os.Getenv("AX_ASSEMBLY_STORE"))
		// The first Capture precedes object construction; the second Capture starts
		// after pack/index/manifest installs. Exit before a result can be exposed.
		count := 0
		runner := assemblyIntercept{run: func(ctx context.Context, dir string, args []string) (GitResult, error) {
			if len(args) > 0 && args[0] == "rev-parse" && len(args) > 1 && args[1] == "--show-toplevel" {
				count++
				if count == 2 {
					os.Exit(74)
				}
			}
			return (ExecGitRunner{}).Run(ctx, dir, args...)
		}}
		if _, err := AssembleProvisional(context.Background(), runner, options); err != nil {
			t.Fatalf("crash path assembly returned before reaching its injected crash: %v", err)
		}
		t.Fatal("crash point not reached")
		return
	}
	root := initLiveRepo(t)
	home := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(root, ".git/index"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(exe, "-test.run=^TestAssemblyCrashAfterObjectsAndRetry$", "-test.count=1")
	command.Env = append(os.Environ(), "AX_ASSEMBLY_CRASH_CHILD=1", "AX_ASSEMBLY_REPO="+root, "AX_ASSEMBLY_STORE="+home)
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 74 {
		t.Fatalf("expected crash exit 74; err=%v output=%s", err, output)
	}
	t.Log("child assembly exit 74 is the expected post-install/pre-result failure")
	options := assemblyOptions(t, root)
	options.Members[0].Content = openCrashStore(t, home)
	value := assemble(t, ExecGitRunner{}, options)
	again := assemble(t, ExecGitRunner{}, options)
	if value.RootID != again.RootID {
		t.Fatal("crash retry not idempotent")
	}
	after, err := os.ReadFile(filepath.Join(root, ".git/index"))
	if err != nil || string(after) != string(index) {
		t.Fatal("crash changed source index")
	}
}
