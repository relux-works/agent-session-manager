package gitsnap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// liveGit reports whether the git executable is available. Tests that
// need a real repository skip otherwise; scripted tests cover the same
// gates without it.
func liveGit(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git executable not available")
	}
	return path
}

// gitRun runs git with repository-local identity and signing disabled so
// the test never depends on ambient user configuration.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := []string{"-c", "user.email=test@example.com", "-c", "user.name=test", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main", "-c", "init.templateDir="}
	full = append(full, args...)
	command := exec.Command("git", full...)
	command.Dir = dir
	command.Env = isolatedGitEnv()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

// initLiveRepo creates a committed repository with one sanitized remote
// that never touches the network: git remote add performs no fetch.
func initLiveRepo(t *testing.T) string {
	t.Helper()
	liveGit(t)
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main", ".")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "README.md", "AGENTS.md")
	gitRun(t, dir, "commit", "-qm", "init")
	gitRun(t, dir, "remote", "add", "origin", "ssh://git@github.com/relux/payments-api.git")
	return dir
}

func captureLive(t *testing.T, dir string) *Snapshot {
	t.Helper()
	snapshot, err := Capture(context.Background(), ExecGitRunner{}, dir)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	return snapshot
}

// TestCaptureLiveBranchRepository drives the full production entry over a
// real branch checkout: identity, HEAD/ref, worktree metadata, index
// stages/flags, staged and unstaged deltas, and file modes.
func TestCaptureLiveBranchRepository(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)
	gitRun(t, dir, "update-index", "--assume-unchanged", "AGENTS.md")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "README.md")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("staged\nworking\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot := captureLive(t, dir)

	if snapshot.RepositoryIdentity != "relux/payments-api" {
		t.Errorf("RepositoryIdentity = %q, want relux/payments-api", snapshot.RepositoryIdentity)
	}
	if len(snapshot.Remotes) != 1 || snapshot.Remotes[0].Name != "origin" {
		t.Fatalf("Remotes = %+v, want one origin remote", snapshot.Remotes)
	}
	if snapshot.Remotes[0].FetchURL != "ssh://git@github.com/relux/payments-api.git" {
		t.Errorf("FetchURL = %q", snapshot.Remotes[0].FetchURL)
	}
	if snapshot.Head.Mode != "branch" || snapshot.Head.Ref == nil || *snapshot.Head.Ref != "refs/heads/main" {
		t.Errorf("Head = %+v, want branch refs/heads/main", snapshot.Head)
	}
	if snapshot.Head.OID == nil || !strings.HasPrefix(*snapshot.Head.OID, "sha1:") {
		t.Errorf("Head.OID = %+v, want sha1: OID", snapshot.Head.OID)
	}
	if snapshot.ObjectFormat != "sha1" {
		t.Errorf("ObjectFormat = %q", snapshot.ObjectFormat)
	}
	if snapshot.Worktree.IsBare || snapshot.Worktree.RepoRoot == "" || snapshot.Worktree.GitDir == "" {
		t.Errorf("Worktree = %+v, want non-bare with roots", snapshot.Worktree)
	}
	if len(snapshot.Worktree.Worktrees) != 1 {
		t.Errorf("Worktrees = %v, want the main checkout listed", snapshot.Worktree.Worktrees)
	}
	if snapshot.Index.Format != "git_index" || snapshot.Index.Version != 2 {
		t.Errorf("Index = %+v, want git_index version 2", snapshot.Index)
	}
	if snapshot.Index.EntryCount != len(snapshot.Index.Entries) || len(snapshot.Index.Entries) != 2 {
		t.Fatalf("Index entries = %+v", snapshot.Index)
	}
	byPath := map[string]IndexEntry{}
	for _, entry := range snapshot.Index.Entries {
		byPath[entry.Path] = entry
	}
	agents, ok := byPath["AGENTS.md"]
	if !ok {
		t.Fatalf("index entries = %+v, want AGENTS.md", snapshot.Index.Entries)
	}
	if agents.Stage != 0 || agents.Mode != 0o100644 || !agents.AssumeUnchanged {
		t.Errorf("AGENTS.md entry = %+v, want stage 0 mode 100644 assume-unchanged", agents)
	}
	if agents.IntentToAdd || agents.SkipWorktree {
		t.Errorf("AGENTS.md entry = %+v, want no intent-to-add or skip-worktree", agents)
	}
	if len(snapshot.Staged) != 1 || snapshot.Staged[0].Path != "README.md" || snapshot.Staged[0].Status != "M" {
		t.Errorf("Staged = %+v, want one M README.md", snapshot.Staged)
	}
	if len(snapshot.Unstaged) != 1 || snapshot.Unstaged[0].Path != "README.md" || snapshot.Unstaged[0].Status != "M" {
		t.Errorf("Unstaged = %+v, want one M README.md", snapshot.Unstaged)
	}
	modes := map[string]PathMode{}
	for _, mode := range snapshot.PathModes {
		modes[mode.Path] = mode
	}
	readme, ok := modes["README.md"]
	if !ok || readme.Kind != KindFile || readme.PermBits&0o444 != 0o444 {
		t.Errorf("PathModes[README.md] = %+v, want file with read bits", readme)
	}
	if _, untracked := modes["notes.txt"]; untracked {
		t.Errorf("PathModes must not contain untracked notes.txt: %+v", snapshot.PathModes)
	}
}

// TestCaptureLiveHeadModes covers detached and unborn HEAD through the
// production entry.
func TestCaptureLiveHeadModes(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)

	detached := captureLive(t, dir)
	_ = detached

	gitRun(t, dir, "checkout", "-q", "--detach", "HEAD")
	snapshot := captureLive(t, dir)
	if snapshot.Head.Mode != "detached" || snapshot.Head.OID == nil || snapshot.Head.Ref != nil {
		t.Errorf("detached Head = %+v", snapshot.Head)
	}

	empty := t.TempDir()
	gitRun(t, empty, "init", "-q", "-b", "main", ".")
	gitRun(t, empty, "remote", "add", "origin", "https://github.com/relux/empty.git")
	unborn := captureLive(t, empty)
	if unborn.Head.Mode != "unborn" || unborn.Head.OID != nil || unborn.Head.Ref == nil || *unborn.Head.Ref != "refs/heads/main" {
		t.Errorf("unborn Head = %+v", unborn.Head)
	}
	if unborn.RepositoryIdentity != "relux/empty" {
		t.Errorf("unborn RepositoryIdentity = %q", unborn.RepositoryIdentity)
	}
	if len(unborn.Index.Entries) != 0 || unborn.Index.EntryCount != 0 {
		t.Errorf("unborn index = %+v, want empty", unborn.Index)
	}
}

// TestCaptureLiveIndexFlags stages the worktree flags the grammar
// admits: skip-worktree and intent-to-add alongside clear entries.
func TestCaptureLiveIndexFlags(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)
	gitRun(t, dir, "update-index", "--skip-worktree", "AGENTS.md")
	if err := os.WriteFile(filepath.Join(dir, "ITA.txt"), []byte("ita\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "-N", "ITA.txt")

	snapshot := captureLive(t, dir)
	byPath := map[string]IndexEntry{}
	for _, entry := range snapshot.Index.Entries {
		byPath[entry.Path] = entry
	}
	agents, ok := byPath["AGENTS.md"]
	if !ok || !agents.SkipWorktree || agents.AssumeUnchanged || agents.IntentToAdd {
		t.Errorf("AGENTS.md entry = %+v, want only skip-worktree", agents)
	}
	ita, ok := byPath["ITA.txt"]
	if !ok || !ita.IntentToAdd {
		t.Errorf("ITA.txt entry = %+v, want intent-to-add", ita)
	}
}

// TestCaptureLiveConflictStages drives conflict stages 1-3 alongside
// stage 0 through the production entry.
func TestCaptureLiveConflictStages(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)
	gitRun(t, dir, "checkout", "-qb", "side")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("side\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "commit", "-qam", "side")
	gitRun(t, dir, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "commit", "-qam", "main")
	merge := exec.Command("git", "merge", "side")
	merge.Dir = dir
	_ = merge.Run()

	snapshot := captureLive(t, dir)
	byPathStage := map[string]IndexEntry{}
	for _, entry := range snapshot.Index.Entries {
		byPathStage[entry.Path+"|"+string(rune('0'+entry.Stage))] = entry
	}
	for _, stage := range []int{1, 2, 3} {
		key := "README.md|" + string(rune('0'+stage))
		if _, ok := byPathStage[key]; !ok {
			t.Errorf("missing conflict stage %d in %+v", stage, snapshot.Index.Entries)
		}
	}
}

// TestCaptureLiveExecBitAndSymlinkKind proves file modes distinguish the
// executable bit and name symlink versus file kinds.
func TestCaptureLiveExecBitAndSymlinkKind(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("README.md", filepath.Join(dir, "current")); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "run.sh", "current")

	snapshot := captureLive(t, dir)
	modes := map[string]PathMode{}
	for _, mode := range snapshot.PathModes {
		modes[mode.Path] = mode
	}
	script, ok := modes["run.sh"]
	if !ok || script.Kind != KindFile || !script.Exec || script.PermBits&0o111 == 0 {
		t.Errorf("PathModes[run.sh] = %+v, want exec file", script)
	}
	link, ok := modes["current"]
	if !ok || link.Kind != KindSymlink {
		t.Errorf("PathModes[current] = %+v, want symlink kind", link)
	}
	byPath := map[string]IndexEntry{}
	for _, entry := range snapshot.Index.Entries {
		byPath[entry.Path] = entry
	}
	if byPath["run.sh"].Mode != 0o100755 {
		t.Errorf("run.sh index mode = %o, want 100755", byPath["run.sh"].Mode)
	}
	if byPath["current"].Mode != 0o120000 {
		t.Errorf("current index mode = %o, want 120000", byPath["current"].Mode)
	}
}

// TestCaptureLiveMissingTrackedPath records a deleted worktree file as
// missing rather than refusing the capture.
func TestCaptureLiveMissingTrackedPath(t *testing.T) {
	t.Parallel()
	dir := initLiveRepo(t)
	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	snapshot := captureLive(t, dir)
	found := false
	for _, mode := range snapshot.PathModes {
		if mode.Path == "AGENTS.md" {
			found = true
			if mode.Kind != KindMissing {
				t.Errorf("PathModes[AGENTS.md] = %+v, want missing", mode)
			}
		}
	}
	if !found {
		t.Fatalf("PathModes lacks AGENTS.md: %+v", snapshot.PathModes)
	}
	if len(snapshot.Unstaged) != 1 || snapshot.Unstaged[0].Status != "D" {
		t.Errorf("Unstaged = %+v, want one D AGENTS.md", snapshot.Unstaged)
	}
}
