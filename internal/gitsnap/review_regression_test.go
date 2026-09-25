package gitsnap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reviewIntercept struct {
	base Runner
	hook func([]string)
}

func (r reviewIntercept) Run(c context.Context, d string, a ...string) (GitResult, error) {
	r.hook(a)
	return r.base.Run(c, d, a...)
}

func TestReviewUpstreamReadFailure(t *testing.T) {
	f := scriptBaseline(newScriptedRunner())
	f.Rescript([]string{"for-each-ref", "--format=%(upstream)", "--", "refs/heads/feature/ax"}, GitResult{ExitCode: 128, Stderr: []byte("fatal: unable to read upstream reference: Input/output error")})
	s, e := Capture(context.Background(), f, "/tmp/repo")
	if e == nil {
		t.Fatalf("fatal upstream read accepted as absence: upstream=%v", s.UpstreamRef)
	}
}
func TestReviewMalformedConfig(t *testing.T) {
	d := initLiveRepo(t)
	gitRun(t, d, "config", "core.symlinks", "not-a-bool")
	s, e := Capture(context.Background(), ExecGitRunner{}, d)
	if e == nil {
		t.Fatalf("malformed boolean accepted as symlinks=%v", s.Features.Symlinks)
	}
}
func TestReviewSymlinkDefault(t *testing.T) {
	d := initLiveRepo(t)
	if e := os.Symlink("README.md", filepath.Join(d, "link")); e != nil {
		t.Fatal(e)
	}
	gitRun(t, d, "add", "link")
	s := captureLive(t, d)
	if !s.Features.Symlinks {
		t.Fatalf("unset core.symlinks captured false though index represents symlink: %+v", s.Index.Entries)
	}
}
func TestReviewRequiredFilterBoolean(t *testing.T) {
	d := initLiveRepo(t)
	gitRun(t, d, "config", "filter.demo.required", "yes")
	if v := strings.TrimSpace(gitRun(t, d, "config", "--bool", "--get", "filter.demo.required")); v != "true" {
		t.Fatal(v)
	}
	s := captureLive(t, d)
	if len(s.Features.RequiredFilters) != 1 || s.Features.RequiredFilters[0] != "demo" {
		t.Fatalf("required filter omitted: %v", s.Features.RequiredFilters)
	}
}
func TestReviewRemoteNameBound(t *testing.T) {
	d := initLiveRepo(t)
	gitRun(t, d, "remote", "add", strings.Repeat("r", 129), "https://example.com/team/repo.git")
	s, e := Capture(context.Background(), ExecGitRunner{}, d)
	if e == nil {
		t.Fatalf("129-character remote name accepted: remote count=%d", len(s.Remotes))
	}
}
func TestReviewSubdirectoryCapture(t *testing.T) {
	d := initLiveRepo(t)
	if e := os.Mkdir(filepath.Join(d, "sub"), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, "sub", "inner"), []byte("inner\n"), 0644); e != nil {
		t.Fatal(e)
	}
	gitRun(t, d, "add", "sub/inner")
	s := captureLive(t, filepath.Join(d, "sub"))
	paths := []string{}
	for _, v := range s.Index.Entries {
		paths = append(paths, v.Path)
	}
	if len(paths) != 3 || paths[2] != "sub/inner" {
		t.Fatalf("subdir capture lost/rebased index paths: %v; modes=%+v cwd=%q", paths, s.PathModes, s.Worktree.CWD)
	}
}
func TestReviewHeadRefRace(t *testing.T) {
	d := initLiveRepo(t)
	gitRun(t, d, "branch", "other")
	calls := 0
	fired := false
	r := reviewIntercept{ExecGitRunner{}, func(a []string) {
		if strings.Join(a, " ") == "rev-parse --verify --quiet HEAD" {
			calls++
			if calls == 2 {
				gitRun(t, d, "symbolic-ref", "HEAD", "refs/heads/other")
				fired = true
			}
		}
	}}
	s, e := Capture(context.Background(), r, d)
	if !fired {
		t.Fatal("fault not fired")
	}
	if e == nil {
		t.Fatalf("HEAD changed to other at same OID, capture accepted stale ref %q", *s.Head.Ref)
	}
}
func TestReviewIndexVersionRace(t *testing.T) {
	d := initLiveRepo(t)
	calls := 0
	fired := false
	r := reviewIntercept{ExecGitRunner{}, func(a []string) {
		if strings.Join(a, " ") == "ls-files --stage --debug -z" {
			calls++
			if calls == 2 {
				gitRun(t, d, "update-index", "--index-version", "4")
				fired = true
			}
		}
	}}
	s, e := Capture(context.Background(), r, d)
	if !fired {
		t.Fatal("fault not fired")
	}
	if e == nil {
		t.Fatalf("index changed v2 to v4, capture accepted stale version %d", s.Index.Version)
	}
}
func TestReviewRunnerLockOverride(t *testing.T) {
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	p := filepath.Join(t.TempDir(), "probe-git")
	if e := os.WriteFile(p, []byte("#!/bin/sh\nprintf '%s' \"$GIT_OPTIONAL_LOCKS\"\n"), 0755); e != nil {
		t.Fatal(e)
	}
	r, e := (ExecGitRunner{GitPath: p}).Run(context.Background(), t.TempDir(), "probe")
	if e != nil {
		t.Fatal(e)
	}
	if string(r.Stdout) != "0" {
		t.Fatalf("read-only runner permits inherited GIT_OPTIONAL_LOCKS=%s", r.Stdout)
	}
}
func TestReviewBinaryRenameControl(t *testing.T) {
	d := initLiveRepo(t)
	p := filepath.Join(d, "-binary")
	if e := os.WriteFile(p, []byte{0, 1, 2, 3}, 0644); e != nil {
		t.Fatal(e)
	}
	gitRun(t, d, "add", "--", "-binary")
	gitRun(t, d, "commit", "-qm", "binary")
	gitRun(t, d, "mv", "--", "-binary", "renamed")
	if e := os.WriteFile(filepath.Join(d, "renamed"), []byte{0, 1, 2, 4}, 0644); e != nil {
		t.Fatal(e)
	}
	s := captureLive(t, d)
	if len(s.Staged) != 1 || s.Staged[0].Status != "R" || s.Staged[0].Target == nil || *s.Staged[0].Target != "renamed" || len(s.Unstaged) != 1 {
		t.Fatalf("binary rename/control mismatch: %+v %+v", s.Staged, s.Unstaged)
	}
}

func TestReviewFeatureReadFailure(t *testing.T) {
	f := scriptBaseline(newScriptedRunner())
	f.Rescript([]string{"config", "--bool", "--get", "core.symlinks"}, exitResult(128, ""))
	s, e := Capture(context.Background(), f, "/tmp/repo")
	if e == nil {
		t.Fatalf("fatal feature read accepted as symlinks=%v", s.Features.Symlinks)
	}
}
func TestReviewRequiredFilterReadFailure(t *testing.T) {
	f := scriptBaseline(newScriptedRunner())
	f.Rescript([]string{"config", "-z", "--name-only", "--get-regexp", `^filter\..*\.required$`}, exitResult(128, ""))
	s, e := Capture(context.Background(), f, "/tmp/repo")
	if e == nil {
		t.Fatalf("fatal required-filter read accepted as empty: %v", s.Features.RequiredFilters)
	}
}

func TestReviewLockOverrideMutatesIndex(t *testing.T) {
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	d := initLiveRepo(t)
	p := filepath.Join(d, "README.md")
	// Replace a tracked file with identical contents to invalidate cached stat data.
	if e := os.Remove(p); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte("base\n"), 0644); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(filepath.Join(d, ".git", "index"))
	if e != nil {
		t.Fatal(e)
	}
	snapshot, captureErr := Capture(context.Background(), ExecGitRunner{}, d)
	after, e := os.ReadFile(filepath.Join(d, ".git", "index"))
	if e != nil {
		t.Fatal(e)
	}
	if captureErr != nil {
		t.Fatalf("Capture: %v", captureErr)
	}
	if len(snapshot.Unstaged) != 0 {
		t.Fatalf("stat-only change became a delta: %+v", snapshot.Unstaged)
	}
	if string(before) != string(after) {
		t.Fatalf("Capture changed durable index bytes under inherited locks=1; Capture error=%v", captureErr)
	}
}

func TestReviewSymbolicRefReadFailure(t *testing.T) {
	f := scriptBaseline(newScriptedRunner())
	f.Rescript([]string{"symbolic-ref", "-q", "HEAD"}, GitResult{ExitCode: 128, Stderr: []byte("fatal: could not read HEAD: Input/output error")})
	s, e := Capture(context.Background(), f, "/tmp/repo")
	if e == nil {
		t.Fatalf("fatal symbolic-ref read accepted as head mode=%s", s.Head.Mode)
	}
}
