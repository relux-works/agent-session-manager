package gitsnap

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

// Every refusal gate below is driven through Capture, the production
// entry, with a scripted Runner standing in for git. The named proving
// test per gate is pinned by the census in census_test.go; renaming a
// test without updating the pin breaks the build.

// TestRefuseOutsideRepository covers GateNotRepository: no repository
// means no HEAD, index, or worktree to read.
func TestRefuseOutsideRepository(t *testing.T) {
	t.Parallel()
	fake := newScriptedRunner()
	fake.Rescript([]string{"rev-parse", "--is-inside-work-tree", "--is-bare-repository", "--show-object-format", "--absolute-git-dir", "--git-common-dir", "--is-shallow-repository"},
		exitResult(128, "fatal: not a git repository"))
	_, err := Capture(context.Background(), fake, "/tmp/nowhere")
	requireRefusal(t, err, GateNotRepository)
}

// TestRefuseNilRunner covers the degenerate GateNotRepository arm.
func TestRefuseNilRunner(t *testing.T) {
	t.Parallel()
	_, err := Capture(context.Background(), nil, "/tmp/nowhere")
	requireRefusal(t, err, GateNotRepository)
}

// TestRefuseCorruptHead covers GateHeadCorrupt: neither a ref nor a
// commit behind HEAD is a state git can be in, so capture refuses.
func TestRefuseCorruptHead(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"symbolic-ref", "-q", "HEAD"}, exitResult(1, ""), exitResult(1, ""))
	fake.Rescript([]string{"rev-parse", "--verify", "--quiet", "HEAD"}, exitResult(1, ""), exitResult(1, ""))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateHeadCorrupt)
}

// TestRefuseUnbornNonBranchRef covers GateHeadCorrupt: an unborn HEAD
// must name a refs/heads/ branch, never a tag.
func TestRefuseUnbornNonBranchRef(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"symbolic-ref", "-q", "HEAD"}, okResult("refs/tags/v1\n"), okResult("refs/tags/v1\n"))
	fake.Rescript([]string{"rev-parse", "--verify", "--quiet", "HEAD"}, exitResult(1, ""), exitResult(1, ""))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateHeadCorrupt)
}

// TestRefuseBranchWithLiteralHeadRef covers GateHeadRef: the literal
// HEAD is not a fully qualified ref (TM-GIT-N3 class: a HEAD mode
// carrying a ref it must not carry is refused).
func TestRefuseBranchWithLiteralHeadRef(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"symbolic-ref", "-q", "HEAD"}, okResult("HEAD\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateHeadRef)
}

// TestRefuseCrossFormatOID covers GateHeadOIDFormat: a sha1-length HEAD
// OID in a sha256 repository is refused, never coerced.
func TestRefuseCrossFormatOID(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"rev-parse", "--is-inside-work-tree", "--is-bare-repository", "--show-object-format", "--absolute-git-dir", "--git-common-dir", "--is-shallow-repository"},
		okResult("true\nfalse\nsha256\n/tmp/repo/.git\n/tmp/repo/.git\nfalse\n"))
	// Keep every other read valid in sha256 so weakening HEAD validation
	// can reach the Capture result rather than a second unrelated refusal.
	index := stageProbeIndex("100644", strings.Repeat("2b", 32), "0", "README.md")
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateHeadOIDFormat)
}

// TestRefuseNoRemotes covers the lower GateRemotesRange arm: Section
// 10.4 requires remotes[1..16], so a remoteless repository cannot yield
// a conformant member and capture refuses instead of inventing one.
func TestRefuseNoRemotes(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult(""))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateRemotesRange)
}

// TestRefuseCredentialBearingRemote covers GateRemoteURL: a fetch URL
// carrying a password or token is refused, never sanitized by trimming.
func TestRefuseCredentialBearingRemote(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\thttps://user:token123@github.com/relux/payments-api.git (fetch)\norigin\thttps://user:token123@github.com/relux/payments-api.git (push)\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateRemoteURL)
}

// TestRefuseCredentialPushURL covers the GateRemoteURL push arm: a clean
// fetch URL does not launder a credential-bearing push URL.
func TestRefuseCredentialPushURL(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/relux/payments-api.git (fetch)\norigin\thttps://user:token123@github.com/relux/push-mirror.git (push)\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateRemoteURL)
}

// TestRefuseMalformedRemoteLine covers the GateRemoteURL grammar arm.
func TestRefuseMalformedRemoteLine(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin ssh://git@github.com/relux/x.git (fetch)\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateRemoteURL)
}

// TestRefuseEmptyIdentity covers GateIdentityLength: a remote URL with
// no path component yields no repository_identity.
func TestRefuseEmptyIdentity(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/.git (fetch)\norigin\tssh://git@github.com/.git (push)\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIdentityLength)
}

// TestRefuseIndexVersionFive covers GateIndexVersion: only 2, 3, and 4
// are admitted.
func TestRefuseIndexVersionFive(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"update-index", "--show-index-version"}, okResult("5\n"), okResult("5\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexVersion)
}

// TestRefuseIndexVersionOne covers the lower GateIndexVersion arm.
func TestRefuseIndexVersionOne(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"update-index", "--show-index-version"}, okResult("1\n"), okResult("1\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexVersion)
}

const stageProbeDebug = "  ctime: 1:2\n  mtime: 3:4\n  dev: 5\tino: 6\n  uid: 7\tgid: 8\n  size: 5\tflags: 0\n"

// stageProbeIndex builds one ls-files --debug -z entry with free stage
// digits so the stage gate, not the grammar, decides.
func stageProbeIndex(mode, oid, stage, path string) string {
	return mode + " " + oid + " " + stage + "\t" + path + "\x00" + stageProbeDebug
}

// TestRefuseIndexStageFour covers GateIndexStage with the exact TM-GIT-N2
// mutation: stage 4 is refused as incompatible_schema.
func TestRefuseIndexStageFour(t *testing.T) {
	t.Parallel()
	bad := stageProbeIndex("100644", "19d9cc8584ac2c7dcf57d2680375e80f099dc481", "4", "README.md")
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(bad), okResult(bad))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexStage)
}

// TestRefuseIndexBadOID covers GateIndexEntry: a truncated OID is a
// malformed member, not stage content.
func TestRefuseIndexBadOID(t *testing.T) {
	t.Parallel()
	bad := stageProbeIndex("100644", "19d9cc85", "0", "README.md")
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(bad), okResult(bad))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexEntry)
}

// TestRefuseIndexBadMode covers GateIndexEntry: a non-octal mode is a
// malformed member.
func TestRefuseIndexBadMode(t *testing.T) {
	t.Parallel()
	bad := stageProbeIndex("888888", "19d9cc8584ac2c7dcf57d2680375e80f099dc481", "0", "README.md")
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(bad), okResult(bad))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexEntry)
}

// TestRefuseIndexEscapingPath covers GateIndexEntry: an escaping path is
// never admitted into the snapshot.
func TestRefuseIndexEscapingPath(t *testing.T) {
	t.Parallel()
	bad := stageProbeIndex("100644", "19d9cc8584ac2c7dcf57d2680375e80f099dc481", "0", "../evil.md")
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(bad), okResult(bad))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexEntry)
}

// TestRefuseIndexMissingFlags covers GateIndexEntry: a record without a
// flags line is a partial read, never an entry with clear flags. A
// failed, partial, or malformed read is never a legitimate absence.
func TestRefuseIndexMissingFlags(t *testing.T) {
	t.Parallel()
	bad := "100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md\x00  ctime: 1:2\n"
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(bad), okResult(bad))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexEntry)
}

// TestRefuseUnsortedIndex covers GateIndexSort: entries must be strictly
// sorted by path then stage.
func TestRefuseUnsortedIndex(t *testing.T) {
	t.Parallel()
	first := "100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md\x00" + stageProbeDebug
	second := "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md\x00" + stageProbeDebug
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(first+second), okResult(first+second))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexSort)
}

// TestRefuseDuplicateIndexEntry covers GateIndexSort: the same
// (path, stage) twice is not strictly increasing.
func TestRefuseDuplicateIndexEntry(t *testing.T) {
	t.Parallel()
	one := "100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md\x00" + stageProbeDebug
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(one+one), okResult(one+one))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexSort)
}

// TestRefuseBadUpstream covers GateUpstreamRef: an unparseable upstream
// is refused rather than carried as a string.
func TestRefuseBadUpstream(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"for-each-ref", "--format=%(upstream)", "--", "refs/heads/feature/ax"}, okResult("HEAD\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateUpstreamRef)
}

// TestRefuseUnknownDeltaStatus covers GateDeltaStatus: the closed status
// table has no X member.
func TestRefuseUnknownDeltaStatus(t *testing.T) {
	t.Parallel()
	raw := ":100644 100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 b6b0be997c9c8246cdd346dd7ece72140d74dee0 X\x00AGENTS.md\x00"
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"diff", "--cached", "--raw", "--no-abbrev", "-z"}, okResult(raw), okResult(raw))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateDeltaStatus)
}

// TestRefuseSpecialWorktreeFile covers GateWorktreeKind through the
// production entry against a live repository: a FIFO smuggled into the
// index via --cacheinfo is an unsupported kind with no snapshot member.
func TestRefuseSpecialWorktreeFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("FIFO capture refusal needs a unix fifo")
	}
	dir := initLiveRepo(t)
	fifo := dir + "/pipe"
	if err := makeFifo(fifo); err != nil {
		t.Skipf("cannot create fifo: %v", err)
	}
	gitRun(t, dir, "update-index", "--add", "--cacheinfo", "100644,e69de29bb2d1d6434b8b29ae775ad8c2e48c5391,pipe")
	_, err := Capture(context.Background(), ExecGitRunner{}, dir)
	requireRefusal(t, err, GateWorktreeKind)
}

// TestRemoteFetchPushMapping proves fetch and push URLs land on their
// own members: the remote parser keys on the (fetch)/(push) tokens, and
// swapping the assignment while preserving the tokens must redden here,
// not only in a static check.
func TestRemoteFetchPushMapping(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/relux/payments-api.git (fetch)\norigin\tssh://git@github.com/relux/push-mirror.git (push)\n"))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.Remotes[0].FetchURL != "ssh://git@github.com/relux/payments-api.git" {
		t.Errorf("FetchURL = %q", snapshot.Remotes[0].FetchURL)
	}
	if snapshot.Remotes[0].PushURL == nil || *snapshot.Remotes[0].PushURL != "ssh://git@github.com/relux/push-mirror.git" {
		t.Errorf("PushURL = %+v", snapshot.Remotes[0].PushURL)
	}
	if snapshot.RepositoryIdentity != "relux/payments-api" {
		t.Errorf("RepositoryIdentity = %q, want the fetch URL derivation", snapshot.RepositoryIdentity)
	}
}

// TestAcceptNullUpstream proves the (upstream absent) arm stays green:
// no upstream is null, not a refusal.
func TestAcceptNullUpstream(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"for-each-ref", "--format=%(upstream)", "--", "refs/heads/feature/ax"}, okResult("\n"))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.UpstreamRef != nil {
		t.Errorf("UpstreamRef = %q, want null", *snapshot.UpstreamRef)
	}
}

// TestAcceptBaselineSnapshot proves the scripted baseline itself stays
// green so a refusal test that suddenly passes cannot hide behind a
// broken fixture: the census pairs every gate with a proving test, and
// this test proves the fixture those tests mutate.
func TestAcceptBaselineSnapshot(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.RepositoryIdentity != "relux/payments-api" {
		t.Errorf("RepositoryIdentity = %q", snapshot.RepositoryIdentity)
	}
	if len(snapshot.Index.Entries) != 2 || snapshot.Index.EntryCount != 2 {
		t.Errorf("Index = %+v", snapshot.Index)
	}
	if *snapshot.UpstreamRef != "refs/remotes/origin/feature/ax" {
		t.Errorf("UpstreamRef = %+v", snapshot.UpstreamRef)
	}
	if !strings.HasPrefix(snapshot.Head.oidValue(), "602548b4") {
		t.Errorf("Head = %+v", snapshot.Head)
	}
}
