package gitsnap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func openCrashStore(t *testing.T, home string) ContentOptions {
	t.Helper()
	platform := scalar.PlatformLinux
	if runtime.GOOS == "darwin" {
		platform = scalar.PlatformMacOS
	}
	if runtime.GOOS == "windows" {
		platform = scalar.PlatformWindows
	}
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{Platform: platform, HomeDir: home, TemporaryDir: home})
	if err != nil {
		t.Fatal(err)
	}
	store, err := localstore.OpenObjectStore(paths)
	if err != nil {
		t.Fatal(err)
	}
	return ContentOptions{Store: store}
}
func TestContentCrashAfterInstall(t *testing.T) {
	if os.Getenv("AX_CAPTURE_CRASH_CHILD") == "1" {
		options := openCrashStore(t, os.Getenv("AX_CAPTURE_STORE_HOME"))
		runner := contentFaultRunner{Runner: ExecGitRunner{}, command: "rev-parse --path-format=absolute --git-path info/sparse-checkout", action: func() { os.Exit(73) }}
		Capture(context.Background(), runner, os.Getenv("AX_CAPTURE_REPO"), options)
		t.Fatal("crash point not reached")
		return
	}
	root := initLiveRepo(t)
	home := t.TempDir()
	options := openCrashStore(t, home)
	gitRun(t, root, "sparse-checkout", "set", "src")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(exe, "-test.run=^TestContentCrashAfterInstall$", "-test.count=1")
	command.Env = append(os.Environ(), "AX_CAPTURE_CRASH_CHILD=1", "AX_CAPTURE_STORE_HOME="+home, "AX_CAPTURE_REPO="+root)
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 73 {
		t.Fatalf("expected crash exit 73; err=%v output=%s", err, output)
	}
	t.Log("child capture exited 73 at the post-install/pre-result crash point (expected failure)")
	value := captureContent(t, root, options)
	entry := entriesByPath(value.Content)["README.md"]
	if string(storedBytes(t, options, entry["blob_id"].(string))) != "base\n" {
		t.Fatal("recovery bytes differ")
	}
	err = filepath.WalkDir(options.Store.DataRoot(), func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && len(entry.Name()) >= 17 && entry.Name()[:17] == ".ax-object-stage-" {
			t.Error("staged blob leaked")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
