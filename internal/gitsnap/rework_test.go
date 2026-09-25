package gitsnap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCaptureRootAndSubdirectoryAgree(t *testing.T) {
	dir := initLiveRepo(t)
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "inner"), []byte("inside\n"), 0755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "sub/inner")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("outside changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	root, child := captureLive(t, dir), captureLive(t, sub)
	if root.Worktree.CWD != "." || child.Worktree.CWD != "sub" {
		t.Fatalf("cwd root=%q sub=%q", root.Worktree.CWD, child.Worktree.CWD)
	}
	child.Worktree.CWD = "."
	// The private assembly witness includes native stat metadata such as atime.
	// Compare its file identity/size/mtime/digest contract, then compare every
	// snapshot field independently of incidental access-time changes on reads.
	if !root.observedIndex.equal(child.observedIndex) {
		t.Fatal("root/sub index observations differ")
	}
	root.observedIndex, child.observedIndex = indexState{}, indexState{}
	if !reflect.DeepEqual(root, child) {
		t.Fatalf("root/sub differ:\n%+v\n%+v", root, child)
	}
	for _, mode := range child.PathModes {
		if mode.Kind != KindFile {
			t.Fatalf("wrong root stat: %+v", mode)
		}
	}
}

func TestCaptureLinkedWorktreeSubdirectory(t *testing.T) {
	dir := initLiveRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitRun(t, dir, "worktree", "add", "-qb", "linked", linked)
	if err := os.Mkdir(filepath.Join(linked, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	snapshot := captureLive(t, filepath.Join(linked, "sub"))
	root, _ := filepath.EvalSymlinks(linked)
	common, _ := filepath.EvalSymlinks(filepath.Join(dir, ".git"))
	if snapshot.Worktree.RepoRoot != root || snapshot.Worktree.CommonDir != common || snapshot.Worktree.CWD != "sub" || len(snapshot.Index.Entries) != 2 {
		t.Fatalf("linked capture: %+v", snapshot)
	}
}

func TestCaptureSameOIDHeadModeRace(t *testing.T) {
	dir := initLiveRepo(t)
	count := 0
	runner := reviewIntercept{ExecGitRunner{}, func(args []string) {
		if strings.Join(args, " ") == "rev-parse --verify --quiet HEAD" {
			count++
			if count == 2 {
				gitRun(t, dir, "update-ref", "--no-deref", "HEAD", strings.TrimSpace(gitRun(t, dir, "rev-parse", "HEAD")))
			}
		}
	}}
	snapshot, err := Capture(context.Background(), runner, dir)
	requireRefusal(t, err, GateConsistency)
	if snapshot != nil || count != 2 {
		t.Fatalf("snapshot=%v calls=%d", snapshot, count)
	}
	captureLive(t, dir)
}

func TestCaptureIndexReplacementRace(t *testing.T) {
	dir := initLiveRepo(t)
	count := 0
	runner := reviewIntercept{ExecGitRunner{}, func(args []string) {
		if strings.Join(args, " ") == "ls-files --stage --debug -z" {
			count++
			if count == 2 {
				path := filepath.Join(dir, ".git", "index")
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path+".replacement", content, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(path+".replacement", path); err != nil {
					t.Fatal(err)
				}
			}
		}
	}}
	_, err := Capture(context.Background(), runner, dir)
	requireRefusal(t, err, GateConsistency)
	if count != 2 {
		t.Fatalf("fault count=%d", count)
	}
	captureLive(t, dir)
}

type failingReadRunner struct {
	base   Runner
	match  []string
	result GitResult
	err    error
	fired  int
	at     int
	seen   int
}

func (r *failingReadRunner) Run(ctx context.Context, dir string, args ...string) (GitResult, error) {
	if reflect.DeepEqual(args, r.match) {
		r.seen++
		if r.at == 0 || r.seen == r.at {
			r.fired++
			return r.result, r.err
		}
	}
	return r.base.Run(ctx, dir, args...)
}

// Every failure is injected into a real Capture conversation, at the exact
// fallback-bearing read. Later reads cannot masquerade as the injected refusal.
func TestCaptureReadFailuresNeverBecomeAbsence(t *testing.T) {
	dir := initLiveRepo(t)
	reads := [][]string{
		{"symbolic-ref", "-q", "HEAD"},
		{"rev-parse", "--verify", "--quiet", "HEAD"},
		{"for-each-ref", "--format=%(upstream)", "--", "refs/heads/main"},
		{"config", "--bool", "--get", "core.filemode"},
		{"config", "--bool", "--get", "core.symlinks"},
		{"config", "--bool", "--get", "core.ignorecase"},
		{"config", "--bool", "--get", "core.precomposeunicode"},
		{"config", "--bool", "--get", "core.sparseCheckout"},
		{"config", "-z", "--name-only", "--get-regexp", `^filter\..*\.required$`},
	}
	for _, args := range reads {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			for _, fault := range []struct {
				name   string
				result GitResult
				err    error
			}{
				{"fatal", exitResult(128, ""), nil},
				{"transport", GitResult{}, errors.New("injected read failure")},
				{"partial_absence", exitResult(1, "partial"), nil},
				{"diagnostic_absence", GitResult{ExitCode: 1, Stderr: []byte("read failure")}, nil},
			} {
				t.Run(fault.name, func(t *testing.T) {
					runner := &failingReadRunner{base: ExecGitRunner{}, match: args, result: fault.result, err: fault.err}
					snapshot, err := Capture(context.Background(), runner, dir)
					requireRefusal(t, err, GateNotRepository)
					if snapshot != nil || runner.fired != 1 {
						t.Fatalf("snapshot=%v fired=%d", snapshot, runner.fired)
					}
				})
			}
		})
	}
	captureLive(t, dir)
}

func TestCaptureClosingHeadReadFailure(t *testing.T) {
	dir := initLiveRepo(t)
	for _, args := range [][]string{{"symbolic-ref", "-q", "HEAD"}, {"rev-parse", "--verify", "--quiet", "HEAD"}} {
		runner := &failingReadRunner{base: ExecGitRunner{}, match: args, result: exitResult(128, ""), at: 2}
		_, err := Capture(context.Background(), runner, dir)
		requireRefusal(t, err, GateNotRepository)
		if runner.fired != 1 {
			t.Fatal("closing fault not fired")
		}
	}
}

func TestCaptureRequiredFilterReadFailureAfterCensus(t *testing.T) {
	dir := initLiveRepo(t)
	gitRun(t, dir, "config", "filter.demo.required", "true")
	for _, code := range []int{1, 128} {
		runner := &failingReadRunner{base: ExecGitRunner{}, match: []string{"config", "--bool", "--get", "filter.demo.required"}, result: exitResult(code, "")}
		_, err := Capture(context.Background(), runner, dir)
		requireRefusal(t, err, GateNotRepository)
		if runner.fired != 1 {
			t.Fatal("filter fault not fired")
		}
	}
}

func TestCaptureRequiredFilterBooleanSpellings(t *testing.T) {
	dir := initLiveRepo(t)
	for _, row := range []struct {
		value string
		want  bool
	}{{"true", true}, {"yes", true}, {"on", true}, {"1", true}, {"2", true}, {"-1", true}, {"false", false}, {"no", false}, {"off", false}, {"0", false}, {"", false}} {
		gitRun(t, dir, "config", "filter.demo.dotted.required", row.value)
		snapshot := captureLive(t, dir)
		want := []string(nil)
		if row.want {
			want = []string{"demo.dotted"}
		}
		if !reflect.DeepEqual(snapshot.Features.RequiredFilters, want) {
			t.Fatalf("%q: filters=%v want=%v", row.value, snapshot.Features.RequiredFilters, want)
		}
	}
	gitRun(t, dir, "config", "--add", "filter.demo.dotted.required", "yes")
	gitRun(t, dir, "config", "--add", "filter.demo.dotted.required", "no")
	if got := captureLive(t, dir).Features.RequiredFilters; len(got) != 0 {
		t.Fatalf("last value did not win: %v", got)
	}
}

func TestCaptureUpstreamConfiguredMissingTarget(t *testing.T) {
	dir := initLiveRepo(t)
	gitRun(t, dir, "config", "branch.main.remote", "origin")
	gitRun(t, dir, "config", "branch.main.merge", "refs/heads/missing")
	snapshot := captureLive(t, dir)
	if snapshot.UpstreamRef == nil || *snapshot.UpstreamRef != "refs/remotes/origin/missing" {
		t.Fatalf("configured upstream lost: %v", snapshot.UpstreamRef)
	}
}

func TestCaptureFeatureDefaultsAndOverrides(t *testing.T) {
	dir := initLiveRepo(t)
	gitRun(t, dir, "config", "--unset", "core.filemode")
	snapshot := captureLive(t, dir)
	if !snapshot.Features.FileMode || !snapshot.Features.Symlinks {
		t.Fatalf("Git defaults lost: %+v", snapshot.Features)
	}
	gitRun(t, dir, "config", "core.symlinks", "false")
	gitRun(t, dir, "config", "core.filemode", "false")
	snapshot = captureLive(t, dir)
	if snapshot.Features.FileMode || snapshot.Features.Symlinks {
		t.Fatalf("explicit overrides lost: %+v", snapshot.Features)
	}
}

func TestCaptureRemoteNameBounds(t *testing.T) {
	for _, char := range []string{"r", "é"} {
		for _, size := range []int{128, 129} {
			t.Run(fmt.Sprintf("%s_%d", char, size), func(t *testing.T) {
				dir := initLiveRepo(t)
				name := strings.Repeat(char, size)
				gitRun(t, dir, "remote", "add", name, "https://example.com/team/repo.git")
				snapshot, err := Capture(context.Background(), ExecGitRunner{}, dir)
				if size == 129 {
					requireRefusal(t, err, GateRemoteURL)
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, remote := range snapshot.Remotes {
					if remote.Name == name {
						found = true
					}
				}
				if !found {
					t.Fatal("128-character name absent")
				}
			})
		}
	}
}

func TestCaptureRefGrammarAtEachRead(t *testing.T) {
	for _, ref := range []string{"HEAD", "refs/heads/bad..name"} {
		t.Run("head_"+ref, func(t *testing.T) {
			fake := scriptBaseline(newScriptedRunner())
			fake.Rescript([]string{"symbolic-ref", "-q", "HEAD"}, okResult(ref+"\n"), okResult(ref+"\n"))
			_, err := Capture(context.Background(), fake, "/tmp/repo")
			requireRefusal(t, err, GateHeadRef)
		})
		t.Run("upstream_"+ref, func(t *testing.T) {
			fake := scriptBaseline(newScriptedRunner())
			fake.Rescript([]string{"for-each-ref", "--format=%(upstream)", "--", "refs/heads/feature/ax"}, okResult(ref+"\n"))
			_, err := Capture(context.Background(), fake, "/tmp/repo")
			requireRefusal(t, err, GateUpstreamRef)
		})
	}
}

func TestCaptureFixtureEnvironmentIsolation(t *testing.T) {
	config := filepath.Join(t.TempDir(), "ambient.config")
	if err := os.WriteFile(config, []byte("[core]\nsymlinks = false\n[filter \"ambient\"]\nrequired = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_SYSTEM", config)
	dir := initLiveRepo(t)
	if value := strings.TrimSpace(gitRun(t, dir, "config", "--bool", "--default", "true", "core.symlinks")); value != "true" {
		t.Fatal("fixture inherited ambient config")
	}
}

func TestExecRunnerStartFailure(t *testing.T) {
	result, err := (ExecGitRunner{GitPath: filepath.Join(t.TempDir(), "missing-git")}).Run(context.Background(), t.TempDir(), "status")
	if err == nil || result.ExitCode != -1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCaptureAbsentIndexConfigReadFailure(t *testing.T) {
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"update-index", "--show-index-version"}, okResult("0\n"))
	fake.Script([]string{"config", "--get", "index.version"}, exitResult(128, ""))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateNotRepository)
}

func TestCaptureIndexBytesPreservedOnReadRefusal(t *testing.T) {
	dir := initLiveRepo(t)
	path := filepath.Join(dir, ".git", "index")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	runner := &failingReadRunner{base: ExecGitRunner{}, match: []string{"config", "--bool", "--get", "core.symlinks"}, result: exitResult(128, "")}
	_, err = Capture(context.Background(), runner, dir)
	requireRefusal(t, err, GateNotRepository)
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("refused capture changed index")
	}
}

func TestCaptureRequiredFilterCountBounds(t *testing.T) {
	for _, count := range []int{0, 64, 65} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			fake := scriptBaseline(newScriptedRunner())
			var keys strings.Builder
			for i := 0; i < count; i++ {
				key := fmt.Sprintf("filter.demo%02d.required", i)
				keys.WriteString(key + "\x00")
				fake.Script([]string{"config", "--bool", "--get", key}, okResult("true\n"))
			}
			if count > 0 {
				fake.Rescript([]string{"config", "-z", "--name-only", "--get-regexp", `^filter\..*\.required$`}, okResult(keys.String()))
			}
			snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
			if count == 65 {
				requireRefusal(t, err, GateFeatures)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Features.RequiredFilters) != count {
				t.Fatalf("count=%d", len(snapshot.Features.RequiredFilters))
			}
		})
	}
}

func TestCaptureSplitIndexPreservesBytes(t *testing.T) {
	dir := initLiveRepo(t)
	gitRun(t, dir, "update-index", "--split-index")
	indexPath := filepath.Join(dir, ".git", "index")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := captureLive(t, dir)
	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || len(snapshot.Index.Entries) != 2 {
		t.Fatal("split index changed or lost entries")
	}
}

func TestCaptureDiffPrivateIndexOverridesAmbient(t *testing.T) {
	dir := initLiveRepo(t)
	alternate := filepath.Join(t.TempDir(), "alternate-index")
	original, err := os.ReadFile(filepath.Join(dir, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(alternate, original, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_INDEX_FILE", alternate)
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	gitRun(t, dir, "config", "diff.autoRefreshIndex", "false")
	path := filepath.Join(dir, "README.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	snapshot := captureLive(t, dir)
	after, err := os.ReadFile(alternate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) || len(snapshot.Unstaged) != 0 {
		t.Fatal("ambient index or diff config defeated read-only content comparison")
	}
}

func TestExecRunnerPrivateIndexCleanup(t *testing.T) {
	dir := initLiveRepo(t)
	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	for _, args := range [][]string{{"diff", "--raw"}, {"diff", "--invalid-gitsnap-option"}} {
		result, err := (ExecGitRunner{}).Run(context.Background(), dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		if args[1] == "--raw" && result.ExitCode != 0 {
			t.Fatalf("diff exit=%d", result.ExitCode)
		}
		if args[1] != "--raw" && result.ExitCode == 0 {
			t.Fatal("invalid option unexpectedly accepted")
		}
		entries, err := os.ReadDir(temporary)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "gitsnap-index-") {
				t.Fatalf("private index leaked after exit %d", result.ExitCode)
			}
		}
	}
}

func TestCaptureSuccessfulEmptyHeadReadRefuses(t *testing.T) {
	for _, output := range []string{"", "\n"} {
		fake := scriptBaseline(newScriptedRunner())
		fake.Rescript([]string{"rev-parse", "--verify", "--quiet", "HEAD"}, okResult(output), okResult(output))
		_, err := Capture(context.Background(), fake, "/tmp/repo")
		requireRefusal(t, err, GateHeadOIDFormat)
	}
}
