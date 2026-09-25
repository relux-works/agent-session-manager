package gitsnap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestContentIgnoreCensusWarningRefuses(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	writeContent(t, root, ".gitignore", []byte("ignored-private\nother-ignored\n"))
	writeContent(t, root, "ignored-private", []byte("must stay excluded"))
	for _, tc := range []struct {
		name   string
		stdout string
		stderr string
	}{
		{"empty", "", "warning: unable to access policy: Permission denied\n"},
		{"partial", "other-ignored\x00", "warning: could not open directory\n"},
		{"unclassified-diagnostic", "", "unrecognized diagnostic\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			runner := contentFaultRunner{
				Runner: ExecGitRunner{}, command: "ls-files --others --ignored --exclude-standard --directory -z",
				result: GitResult{ExitCode: 0, Stdout: []byte(tc.stdout), Stderr: []byte(tc.stderr)},
				action: func() { called = true },
			}
			value, err := Capture(context.Background(), runner, root, options)
			if !called {
				t.Fatal("Capture did not reach the ignore census")
			}
			requireRefusal(t, err, GateContentRead)
			if value != nil {
				t.Fatal("incomplete census returned a snapshot")
			}
		})
	}
	if entriesByPath(captureContent(t, root, options).Content)["ignored-private"] != nil {
		t.Fatal("retry included ignored bytes")
	}
}

// Exercise Git's actual successful-exit permission warning, not a synthetic
// nonzero exit. Permission semantics unavailable to this process are a skip,
// never evidence that an unreadable policy was tested.
func TestContentExcludePolicyReadCompleteness(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode 000 does not establish unreadability on Windows")
	}
	for _, source := range []string{"git-info-exclude", "core-excludesFile"} {
		t.Run(source, func(t *testing.T) {
			root := initLiveRepo(t)
			options := contentOptions(t)
			writeContent(t, root, "ignored-private", []byte("must stay excluded"))
			policy := filepath.Join(root, ".git", "info", "exclude")
			if source == "core-excludesFile" {
				policy = filepath.Join(t.TempDir(), "exclusion-policy")
				gitRun(t, root, "config", "core.excludesFile", policy)
			}
			if err := os.MkdirAll(filepath.Dir(policy), 0700); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(policy); !os.IsNotExist(err) {
				t.Fatalf("missing-policy control is not absent: %v", err)
			}
			indexPath := filepath.Join(root, ".git", "index")
			indexBefore, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				after, err := os.ReadFile(indexPath)
				if err != nil || !bytes.Equal(indexBefore, after) {
					t.Errorf("capture changed the real index: %v", err)
				}
			})
			if entriesByPath(captureContent(t, root, options).Content)["ignored-private"] == nil {
				t.Fatal("legitimately missing policy refused or hid untracked content")
			}
			if err := os.WriteFile(policy, []byte("ignored-private\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if entriesByPath(captureContent(t, root, options).Content)["ignored-private"] != nil {
				t.Fatal("readable policy did not exclude private content")
			}
			if err := os.Chmod(policy, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Chmod(policy, 0600); err != nil {
					t.Error(err)
				}
			})
			if _, err := os.ReadFile(policy); err == nil {
				t.Skip("process can read mode-000 policy; permission refusal not exercised")
			} else if !os.IsPermission(err) {
				t.Fatalf("fixture did not establish permission denial: %v", err)
			}
			raw, err := (ExecGitRunner{}).Run(context.Background(), root, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z")
			t.Logf("unreadable policy: Git exit=%d stdout=%q stderr=%q error=%v", raw.ExitCode, raw.Stdout, raw.Stderr, err)
			if err != nil || raw.ExitCode != 0 || len(raw.Stderr) == 0 || len(raw.Stdout) != 0 {
				t.Fatal("Git did not exercise the successful-exit incomplete census witness")
			}
			value, err := Capture(context.Background(), ExecGitRunner{}, root, options)
			requireRefusal(t, err, GateContentRead)
			if value != nil {
				t.Fatal("unreadable policy returned a snapshot")
			}
			if err := os.Chmod(policy, 0600); err != nil {
				t.Fatal(err)
			}
			if entriesByPath(captureContent(t, root, options).Content)["ignored-private"] != nil {
				t.Fatal("restoration/retry included private content")
			}
		})
	}
}
