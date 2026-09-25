package gitsnap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Closed domain bounds from the normative source. Each bound is enforced
// at the production entry and edged in both directions by
// bounds_edge_test.go and rework_test.go. The separate source census counts
// literal refusal sites; it does not prove behavior at every clause.
const (
	maxRemotes         = 16
	maxRemoteNameRune  = 128
	maxRequiredFilters = 64
	maxIndexEntries    = 65536
	maxIdentityRune    = 256
)

// indexVersions is the closed GitIndex version domain of Section 10.4.
var indexVersions = []string{"2", "3", "4"}

// headModes is the closed GitHead mode domain of Section 10.4.
var headModes = []string{"branch", "detached", "unborn"}

// deltaStatuses is the closed staged/unstaged delta status vocabulary.
// A raw-diff letter outside this table (git documents X for unknown) has
// no snapshot member and is refused by GateDeltaStatus.
var deltaStatuses = []string{"A", "C", "D", "M", "R", "T", "U"}

// WorktreeKind names the filesystem kind observed at a tracked path.
type WorktreeKind string

// Closed worktree-kind vocabulary.
const (
	KindFile    WorktreeKind = "file"
	KindDir     WorktreeKind = "directory"
	KindSymlink WorktreeKind = "symlink"
	KindGitlink WorktreeKind = "gitlink"
	KindMissing WorktreeKind = "missing"
	KindOther   WorktreeKind = "other"
)

// worktreeKinds is the closed kind vocabulary as a table for the census.
var worktreeKinds = []WorktreeKind{
	KindFile,
	KindDir,
	KindSymlink,
	KindGitlink,
	KindMissing,
	KindOther,
}

// Remote is one sanitized Git remote: the Section 10.4 GitRemote member
// without any credential-bearing URL the sanitizer would have refused.
type Remote struct {
	Name     string  `json:"name"`
	FetchURL string  `json:"fetch_url"`
	PushURL  *string `json:"push_url"`
}

// Head is the Section 10.4 GitHead member: branch carries OID and ref,
// detached carries only the OID, unborn carries only a refs/heads/ ref.
type Head struct {
	Mode string  `json:"mode"`
	OID  *string `json:"oid"`
	Ref  *string `json:"ref"`
}

// IndexEntry is the Section 10.4 GitIndexEntry member.
type IndexEntry struct {
	Path            string `json:"path"`
	Stage           int    `json:"stage"`
	Mode            uint32 `json:"mode"`
	OID             string `json:"oid"`
	IntentToAdd     bool   `json:"intent_to_add"`
	SkipWorktree    bool   `json:"skip_worktree"`
	AssumeUnchanged bool   `json:"assume_unchanged"`
	FSMonitorValid  bool   `json:"fsmonitor_valid"`
}

// Index is the Section 10.4 GitIndex member minus the raw index blob,
// which AssembleProvisional supplies separately. Capture records
// the logical entries and version. EntryCount always equals
// len(Entries); both are set from the one parsed length.
type Index struct {
	Format     string       `json:"format"`
	Version    int          `json:"version"`
	Entries    []IndexEntry `json:"entries"`
	EntryCount int          `json:"entry_count"`
}

// Delta is one staged or unstaged file change. Staged deltas compare the
// index against HEAD; unstaged deltas compare the working tree against
// the index, so the two arrays stay distinct per Section 12.3. A nil OID
// marks the all-zero side of an addition or deletion.
type Delta struct {
	Path    string  `json:"path"`
	Status  string  `json:"status"`
	Target  *string `json:"target,omitempty"`
	OldMode uint32  `json:"old_mode"`
	NewMode uint32  `json:"new_mode"`
	OldOID  *string `json:"old_oid"`
	NewOID  *string `json:"new_oid"`
}

// PathMode is the observed worktree mode of one tracked path: the kind
// from lstat and the Unix permission bits. Symlink targets and untracked
// paths belong to the sibling leaf and are not recorded here.
type PathMode struct {
	Path     string       `json:"path"`
	Kind     WorktreeKind `json:"kind"`
	PermBits uint32       `json:"perm_bits"`
	Exec     bool         `json:"exec"`
}

// Worktree describes the checkout the snapshot was read from.
type Worktree struct {
	RepoRoot  string   `json:"repo_root"`
	GitDir    string   `json:"git_dir"`
	CommonDir string   `json:"common_dir"`
	IsBare    bool     `json:"is_bare"`
	IsShallow bool     `json:"is_shallow"`
	CWD       string   `json:"cwd"`
	Worktrees []string `json:"worktrees"`
}

// Features carries the captured Section 10.4 GitFeatures scalar fields.
// Sparse-pattern and filter blob identities belong to the blob sibling
// leaf and are not produced here.
type Features struct {
	ObjectFormat      string   `json:"object_format"`
	FileMode          bool     `json:"filemode"`
	Symlinks          bool     `json:"symlinks"`
	CaseSensitive     bool     `json:"case_sensitive"`
	PrecomposeUnicode bool     `json:"precompose_unicode"`
	SparseCheckout    bool     `json:"sparse_checkout"`
	RequiredFilters   []string `json:"required_filter_names"`
	LFSRequired       bool     `json:"lfs_required"`
}

// Capture reads one Git working copy through runner and returns its
// validated leaf snapshot. ExecGitRunner isolates diff's index refresh in
// disposable copies: success returns a snapshot, failure returns
// a *Refusal naming the registry gate that fired, never a partial
// snapshot. The closing re-read refuses with GateConsistency when HEAD,
// the index, or either delta stream moved mid-capture; a retry after
// quiesce heals.
func Capture(ctx context.Context, runner Runner, dir string, content ...ContentOptions) (*Snapshot, error) {
	var state *contentState
	if len(content) > 1 {
		return nil, refuse(GateCapturePolicy, "one content policy is required")
	}
	if len(content) == 1 {
		state = &contentState{options: content[0], active: map[string]bool{}}
		if err := state.validate(); err != nil {
			return nil, err
		}
	}
	return capture(ctx, runner, dir, state, "", 0)
}

func capture(ctx context.Context, runner Runner, dir string, state *contentState, prefix string, depth int) (*Snapshot, error) {
	if runner == nil {
		return nil, refuse(GateNotRepository, "nil runner captures no repository")
	}
	repo, err := readRepository(ctx, runner, dir)
	if err != nil {
		return nil, err
	}
	dir = repo.worktree.RepoRoot
	head, err := readHead(ctx, runner, dir, repo.objectFormat)
	if err != nil {
		return nil, err
	}
	upstream, err := readUpstream(ctx, runner, dir, head)
	if err != nil {
		return nil, err
	}
	remotes, err := readRemotes(ctx, runner, dir)
	if err != nil {
		return nil, err
	}
	identity, err := deriveIdentity(remotes)
	if err != nil {
		return nil, err
	}
	indexState, err := readIndexState(ctx, runner, dir)
	if err != nil {
		return nil, err
	}
	version, err := readIndexVersion(ctx, runner, dir)
	if err != nil {
		return nil, err
	}
	indexBytes, indexRefusal := runOK(ctx, runner, dir, "ls-files", "--stage", "--debug", "-z")
	if indexRefusal != nil {
		return nil, refuse(GateNotRepository, "index read failed: "+indexRefusal.Detail)
	}
	entries, err := parseIndexEntries(indexBytes, repo.objectFormat)
	if err != nil {
		return nil, err
	}
	// Unsupported tracked kinds fail fast here, before the delta reads:
	// git's own diff cannot run against special files, so reaching the
	// diff first would mask GateWorktreeKind behind a process failure.
	modes, statted, err := statIndexPaths(repo.worktree.RepoRoot, entries)
	if err != nil {
		return nil, err
	}
	stagedBytes, stagedRefusal := runOK(ctx, runner, dir, "diff", "--cached", "--raw", "--no-abbrev", "-z")
	if stagedRefusal != nil {
		return nil, refuse(GateNotRepository, "staged delta read failed: "+stagedRefusal.Detail)
	}
	staged, err := parseDeltas(stagedBytes)
	if err != nil {
		return nil, err
	}
	unstagedBytes, unstagedRefusal := runOK(ctx, runner, dir, "diff", "--raw", "--no-abbrev", "-z")
	if unstagedRefusal != nil {
		return nil, refuse(GateNotRepository, "unstaged delta read failed: "+unstagedRefusal.Detail)
	}
	unstaged, err := parseDeltas(unstagedBytes)
	if err != nil {
		return nil, err
	}
	if err := statDeltaPaths(repo.worktree.RepoRoot, &modes, statted, staged, unstaged); err != nil {
		return nil, err
	}
	features, err := readFeatures(ctx, runner, dir, repo.objectFormat)
	if err != nil {
		return nil, err
	}
	var content *Content
	if state != nil {
		content, err = state.capture(ctx, runner, repo, entries, features, head, remotes, prefix, depth)
		if err != nil {
			return nil, err
		}
	}
	if err := recheckConsistency(ctx, runner, dir, head, repo.objectFormat, version, indexState, indexBytes, stagedBytes, unstagedBytes); err != nil {
		return nil, err
	}
	index := Index{Format: "git_index", Version: version, Entries: entries, EntryCount: len(entries)}
	return &Snapshot{
		observedIndex:      indexState,
		RepositoryIdentity: identity,
		Remotes:            remotes,
		Head:               head,
		UpstreamRef:        upstream,
		ObjectFormat:       repo.objectFormat,
		Worktree:           repo.worktree,
		Index:              index,
		Staged:             staged,
		Unstaged:           unstaged,
		PathModes:          modes,
		Features:           features,
		Content:            content,
	}, nil
}

// repositoryFacts bundles the worktree metadata with the object format
// every OID parse below must match.
type repositoryFacts struct {
	worktree     Worktree
	objectFormat string
}

// runOK runs one git command and refuses with GateNotRepository when the
// process cannot start or exits unsuccessfully. Optional readers must
// distinguish their documented absence status explicitly.
func runOK(ctx context.Context, runner Runner, dir string, args ...string) ([]byte, *Refusal) {
	result, err := runner.Run(ctx, dir, args...)
	if err != nil {
		return nil, refuse(GateNotRepository, "git "+strings.Join(args, " ")+": "+err.Error())
	}
	if result.ExitCode != 0 {
		return nil, refuse(GateNotRepository, "git "+strings.Join(args, " ")+" exited "+strconv.Itoa(result.ExitCode))
	}
	return result.Stdout, nil
}

// firstLine trims one trailing newline and splits the first line.
func firstLine(output []byte) string {
	line, _, _ := strings.Cut(string(output), "\n")
	return strings.TrimSuffix(line, "\r")
}

func parseBoolFlag(value string) bool {
	return strings.TrimSpace(value) == "true"
}

func readRepository(ctx context.Context, runner Runner, dir string) (*repositoryFacts, error) {
	probe, refusal := runOK(ctx, runner, dir, "rev-parse",
		"--is-inside-work-tree", "--is-bare-repository", "--show-object-format",
		"--absolute-git-dir", "--git-common-dir", "--is-shallow-repository")
	if refusal != nil {
		return nil, refusal
	}
	fields := strings.Split(strings.TrimSuffix(string(probe), "\n"), "\n")
	if len(fields) != 6 {
		return nil, refuse(GateNotRepository, "rev-parse probe returned "+strconv.Itoa(len(fields))+" fields, want 6")
	}
	inside := parseBoolFlag(fields[0])
	bare := parseBoolFlag(fields[1])
	format := strings.TrimSpace(fields[2])
	gitDir := strings.TrimSpace(fields[3])
	commonDir := strings.TrimSpace(fields[4])
	shallow := parseBoolFlag(fields[5])
	if !inside && !bare {
		return nil, refuse(GateNotRepository, "not inside a work tree or bare repository")
	}
	if format != "sha1" && format != "sha256" {
		return nil, refuse(GateHeadOIDFormat, "unknown object format "+strconv.Quote(format))
	}
	worktree := Worktree{GitDir: gitDir, CommonDir: commonDir, IsBare: bare, IsShallow: shallow}
	if inside && !bare {
		root, rerr := runOK(ctx, runner, dir, "rev-parse", "--show-toplevel")
		if rerr != nil {
			return nil, rerr
		}
		worktree.RepoRoot = firstLine(root)
	} else {
		worktree.RepoRoot = gitDir
	}
	cwd, err := filepath.Abs(dir)
	if err != nil {
		return nil, refuse(GateNotRepository, "working directory unreadable: "+err.Error())
	}
	// Git reports canonical roots; resolve an existing caller path as well.
	if resolved, resolveErr := filepath.EvalSymlinks(cwd); resolveErr == nil {
		cwd = resolved
	}
	relative, err := filepath.Rel(worktree.RepoRoot, cwd)
	if err != nil {
		return nil, refuse(GateNotRepository, "cannot resolve repository-relative cwd")
	}
	worktree.CWD = filepath.ToSlash(relative)
	if worktree.CWD != "." {
		if _, err := scalar.ParseRelativePath(worktree.CWD); err != nil {
			return nil, refuse(GateNotRepository, "invalid repository-relative cwd")
		}
	}
	if !filepath.IsAbs(worktree.CommonDir) {
		worktree.CommonDir = filepath.Clean(filepath.Join(cwd, worktree.CommonDir))
	}
	listing, rerr := runOK(ctx, runner, dir, "worktree", "list", "--porcelain")
	if rerr != nil {
		return nil, rerr
	}
	for _, line := range strings.Split(string(listing), "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok && path != "" {
			worktree.Worktrees = append(worktree.Worktrees, path)
		}
	}
	sort.Strings(worktree.Worktrees)
	return &repositoryFacts{worktree: worktree, objectFormat: format}, nil
}

// oidValue returns the raw hex OID for the consistency re-read.
func (head Head) oidValue() string {
	if head.OID == nil {
		return ""
	}
	value := *head.OID
	if index := strings.Index(value, ":"); index >= 0 {
		return value[index+1:]
	}
	return value
}

func readHead(ctx context.Context, runner Runner, dir, objectFormat string) (Head, error) {
	revision, revErr := runner.Run(ctx, dir, "rev-parse", "--verify", "--quiet", "HEAD")
	if revErr != nil {
		return Head{}, refuse(GateNotRepository, "rev-parse HEAD failed: "+revErr.Error())
	}
	if revision.ExitCode == 0 && firstLine(revision.Stdout) == "" {
		return Head{}, refuse(GateHeadOIDFormat, "successful HEAD read returned no OID")
	}
	if revision.ExitCode != 0 && !absentResult(revision) {
		return Head{}, refuse(GateNotRepository, "HEAD read failed")
	}
	symref, symrefErr := runner.Run(ctx, dir, "symbolic-ref", "-q", "HEAD")
	if symrefErr != nil {
		return Head{}, refuse(GateNotRepository, "symbolic-ref failed: "+symrefErr.Error())
	}
	if symref.ExitCode != 0 && !absentResult(symref) {
		return Head{}, refuse(GateNotRepository, "symbolic-ref read failed")
	}
	if symref.ExitCode == 0 && firstLine(symref.Stdout) == "" {
		return Head{}, refuse(GateHeadRef, "empty symbolic HEAD")
	}
	ref := ""
	if symref.ExitCode == 0 {
		ref = strings.TrimSuffix(string(symref.Stdout), "\n")
	}
	oid := ""
	if revision.ExitCode == 0 {
		oid = strings.TrimSuffix(string(revision.Stdout), "\n")
	}
	switch {
	case ref != "" && oid != "":
		parsedRef, err := scalar.ParseGitRef(ref)
		if err != nil {
			return Head{}, refuse(GateHeadRef, "HEAD ref invalid: "+err.Error())
		}
		parsedOID, err := scalar.ParseGitOIDForObjectFormat(objectFormat+":"+oid, objectFormat)
		if err != nil {
			return Head{}, refuse(GateHeadOIDFormat, "HEAD oid invalid: "+err.Error())
		}
		refValue, oidValue := parsedRef.String(), parsedOID.String()
		return Head{Mode: "branch", OID: &oidValue, Ref: &refValue}, nil
	case ref == "" && oid != "":
		parsedOID, err := scalar.ParseGitOIDForObjectFormat(objectFormat+":"+oid, objectFormat)
		if err != nil {
			return Head{}, refuse(GateHeadOIDFormat, "detached HEAD oid invalid: "+err.Error())
		}
		oidValue := parsedOID.String()
		return Head{Mode: "detached", OID: &oidValue}, nil
	case ref != "" && oid == "":
		parsedRef, err := scalar.ParseGitRef(ref)
		if err != nil {
			return Head{}, refuse(GateHeadRef, "unborn HEAD ref invalid: "+err.Error())
		}
		if !strings.HasPrefix(ref, "refs/heads/") {
			return Head{}, refuse(GateHeadCorrupt, "unborn HEAD ref must start with refs/heads/, got "+strconv.Quote(ref))
		}
		refValue := parsedRef.String()
		return Head{Mode: "unborn", Ref: &refValue}, nil
	default:
		return Head{}, refuse(GateHeadCorrupt, "HEAD names no ref and no commit")
	}
}

// readUpstream uses a successful field read to establish absence. rev-parse
// @{upstream} uses the same failure status for absent tracking and fatal reads.
func readUpstream(ctx context.Context, runner Runner, dir string, head Head) (*string, error) {
	if head.Mode != "branch" {
		return nil, nil
	}
	result, err := runOK(ctx, runner, dir, "for-each-ref", "--format=%(upstream)", "--", *head.Ref)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, refuse(GateConsistency, "HEAD ref disappeared during upstream read")
	}
	ref := strings.TrimSuffix(string(result), "\n")
	if ref == "" {
		return nil, nil
	}
	parsed, perr := scalar.ParseGitRef(ref)
	if perr != nil {
		return nil, refuse(GateUpstreamRef, "upstream ref invalid: "+perr.Error())
	}
	value := parsed.String()
	return &value, nil
}

func readRemotes(ctx context.Context, runner Runner, dir string) ([]Remote, error) {
	output, refusal := runOK(ctx, runner, dir, "remote", "-v")
	if refusal != nil {
		return nil, refusal
	}
	fetch := map[string]string{}
	push := map[string]string{}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			return nil, refuse(GateRemoteURL, "malformed remote line "+strconv.Quote(line))
		}
		name := fields[0]
		if !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > maxRemoteNameRune {
			return nil, refuse(GateRemoteURL, "remote name must contain 1..128 characters")
		}
		rest := strings.TrimSpace(fields[1])
		url, kind, _ := strings.Cut(rest, " ")
		url = strings.TrimSpace(url)
		kind = strings.TrimSpace(kind)
		if name == "" || url == "" {
			return nil, refuse(GateRemoteURL, "malformed remote line "+strconv.Quote(line))
		}
		switch kind {
		case "(fetch)":
			fetch[name] = url
		case "(push)":
			push[name] = url
		default:
			return nil, refuse(GateRemoteURL, "unknown remote kind "+strconv.Quote(kind))
		}
	}
	names := make([]string, 0, len(fetch))
	for name := range fetch {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) < 1 {
		return nil, refuse(GateRemotesRange, "repository has no remotes, want 1..16")
	}
	if len(names) > maxRemotes {
		return nil, refuse(GateRemotesRange, "repository has "+strconv.Itoa(len(names))+" remotes, want 1..16")
	}
	remotes := make([]Remote, 0, len(names))
	for _, name := range names {
		fetchURL, err := scalar.ParseSanitizedGitURL(fetch[name])
		if err != nil {
			return nil, refuse(GateRemoteURL, "remote "+strconv.Quote(name)+" fetch URL: "+err.Error())
		}
		remote := Remote{Name: name, FetchURL: fetchURL.String()}
		if url, ok := push[name]; ok && url != fetch[name] {
			pushURL, err := scalar.ParseSanitizedGitURL(url)
			if err != nil {
				return nil, refuse(GateRemoteURL, "remote "+strconv.Quote(name)+" push URL: "+err.Error())
			}
			value := pushURL.String()
			remote.PushURL = &value
		}
		remotes = append(remotes, remote)
	}
	return remotes, nil
}

// deriveIdentity computes repository_identity from the remotes: the fetch
// URL of origin when present, else the lexicographically first remote
// (remotes arrive sorted). The host, scheme, user, leading path noise,
// and one trailing .git suffix are stripped; the remainder must be
// 1..256 characters, counted in characters rather than bytes.
func deriveIdentity(remotes []Remote) (string, error) {
	chosen := remotes[0].FetchURL
	for _, remote := range remotes {
		if remote.Name == "origin" {
			chosen = remote.FetchURL
			break
		}
	}
	identity := stripRemotePrefix(chosen)
	identity = strings.TrimSuffix(identity, ".git")
	identity = strings.Trim(identity, "/")
	if identity == "" || !utf8.ValidString(identity) {
		return "", refuse(GateIdentityLength, "remote URL yields no repository identity")
	}
	if count := len([]rune(identity)); count < 1 || count > maxIdentityRune {
		return "", refuse(GateIdentityLength, "repository identity is "+strconv.Itoa(count)+" characters, want 1..256")
	}
	return identity, nil
}

// stripRemotePrefix removes the scheme, authority, and scp-like host
// prefix from a sanitized remote URL, leaving the path component.
func stripRemotePrefix(url string) string {
	if index := strings.Index(url, "://"); index >= 0 {
		rest := url[index+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			return rest[slash+1:]
		}
		return ""
	}
	if host, _, _ := strings.Cut(url, ":"); strings.Contains(host, "@") || !strings.Contains(host, "/") {
		if _, path, ok := strings.Cut(url, ":"); ok {
			return strings.TrimPrefix(path, "/")
		}
	}
	return strings.TrimPrefix(url, "/")
}

func readIndexVersion(ctx context.Context, runner Runner, dir string) (int, error) {
	output, refusal := runOK(ctx, runner, dir, "update-index", "--show-index-version")
	if refusal != nil {
		return 0, refusal
	}
	version := strings.TrimSpace(string(output))
	for _, allowed := range indexVersions {
		if version == allowed {
			number, _ := strconv.Atoi(version)
			return number, nil
		}
	}
	// Version 0 means no index file exists yet (fresh unborn checkout).
	// The snapshot then carries the version git itself would write: the
	// configured index.version when it names 2, 3, or 4, else the Git
	// default 2. This is planned empty-index metadata, not a claim that
	// a raw index blob already exists or that its bytes are captured.
	if version == "0" {
		configured, err := readOptionalConfig(ctx, runner, dir, "config", "--get", "index.version")
		if err != nil {
			return 0, refuse(GateNotRepository, "index version config read failed: "+err.Error())
		}
		if configured.ExitCode == 0 {
			for _, allowed := range indexVersions {
				if strings.TrimSpace(string(configured.Stdout)) == allowed {
					number, _ := strconv.Atoi(allowed)
					return number, nil
				}
			}
		}
		return 2, nil
	}
	return 0, refuse(GateIndexVersion, "index version "+strconv.Quote(version)+", want 2, 3, or 4")
}

// ceFlag bits of the index extended flags as reported by git ls-files
// --debug. Observed against real repositories: assume-unchanged sets
// 0x8000, intent-to-add sets 0x20000000, skip-worktree sets
// 0x40000000 alongside the 0x4000 extended marker, and the fsmonitor
// valid bit is 0x10000000. The flag word layout is git-version
// sensitive; the parser pins these four bits and ignores the rest.
const (
	flagAssumeUnchanged = 0x8000
	flagSkipWorktree    = 0x40000000
	flagIntentToAdd     = 0x20000000
	flagFSMonitorValid  = 0x10000000
)

// parseIndexEntries parses git ls-files --stage --debug -z output. The
// NUL byte terminates the entry path, not the record: each entry is a
// stage line "<mode> SP <oid> SP <stage> TAB <path>" closed by NUL,
// followed by \n-terminated debug lines ending with the line carrying
// "flags: <hex>". The cursor parser honors that layout so a path
// containing a newline still parses. Stage digits are accepted freely by
// the grammar and gated to 0..3 below so a stage-4 entry reaches
// GateIndexStage instead of dying as a parse error.
func parseIndexEntries(output []byte, objectFormat string) ([]IndexEntry, error) {
	if len(bytes.TrimSpace(output)) == 0 {
		return nil, nil
	}
	entries := []IndexEntry{}
	cursor := output
	for len(bytes.TrimSpace(cursor)) > 0 {
		entry, rest, err := parseIndexRecord(cursor, objectFormat)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		cursor = rest
	}
	if len(entries) > maxIndexEntries {
		return nil, refuse(GateIndexEntriesRange, "index holds "+strconv.Itoa(len(entries))+" entries, want 0..65536")
	}
	for index := 1; index < len(entries); index++ {
		previous, current := entries[index-1], entries[index]
		if current.Path < previous.Path || (current.Path == previous.Path && current.Stage <= previous.Stage) {
			return nil, refuse(GateIndexSort, "entries not strictly sorted by path then stage at "+strconv.Quote(current.Path))
		}
	}
	return entries, nil
}

func parseIndexRecord(cursor []byte, objectFormat string) (IndexEntry, []byte, error) {
	tab := bytes.IndexByte(cursor, '\t')
	if tab < 0 {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "stage line without path")
	}
	header, afterTab := string(cursor[:tab]), cursor[tab+1:]
	nul := bytes.IndexByte(afterTab, 0)
	if nul < 0 {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "stage line without NUL path terminator")
	}
	path, afterPath := string(afterTab[:nul]), afterTab[nul+1:]
	parts := strings.Split(header, " ")
	if len(parts) != 3 {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "stage header without mode/oid/stage "+strconv.Quote(header))
	}
	modeBits, err := strconv.ParseUint(parts[0], 8, 32)
	if err != nil {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "index mode not octal uint32 "+strconv.Quote(parts[0]))
	}
	stageValue, serr := strconv.Atoi(parts[2])
	if serr != nil {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "index stage not numeric "+strconv.Quote(parts[2]))
	}
	if stageValue < 0 || stageValue > 3 {
		return IndexEntry{}, nil, refuse(GateIndexStage, "index stage "+strconv.Itoa(stageValue)+", want 0..3")
	}
	parsedPath, err := scalar.ParseRelativePath(path)
	if err != nil {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "index path invalid: "+err.Error())
	}
	parsedOID, err := scalar.ParseGitOIDForObjectFormat(objectFormat+":"+parts[1], objectFormat)
	if err != nil {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "index oid invalid: "+err.Error())
	}
	flags, rest, ok := debugFlags(afterPath)
	if !ok {
		return IndexEntry{}, nil, refuse(GateIndexEntry, "index record without flags for "+strconv.Quote(path))
	}
	return IndexEntry{
		Path:            parsedPath.String(),
		Stage:           stageValue,
		Mode:            uint32(modeBits),
		OID:             parsedOID.String(),
		IntentToAdd:     flags&flagIntentToAdd != 0,
		SkipWorktree:    flags&flagSkipWorktree != 0,
		AssumeUnchanged: flags&flagAssumeUnchanged != 0,
		FSMonitorValid:  flags&flagFSMonitorValid != 0,
	}, rest, nil
}

// debugFlags consumes \n-terminated --debug lines up to and including
// the line carrying the flags word ("size: N\tflags: H" shares one
// line, so the search keys on the marker anywhere in the line) and
// returns the unconsumed remainder.
func debugFlags(cursor []byte) (uint64, []byte, bool) {
	for len(cursor) > 0 {
		newline := bytes.IndexByte(cursor, '\n')
		var line []byte
		if newline < 0 {
			line, cursor = cursor, nil
		} else {
			line, cursor = cursor[:newline], cursor[newline+1:]
		}
		if _, value, ok := strings.Cut(string(line), "flags:"); ok {
			flags, err := strconv.ParseUint(strings.TrimSpace(value), 16, 64)
			if err != nil {
				return 0, nil, false
			}
			return flags, cursor, true
		}
	}
	return 0, nil, false
}

// parseDeltas parses git diff --raw --no-abbrev -z output: one record per
// change, ":oldmode SP newmode SP oldoid SP newoid SP STATUS" NUL path
// [NUL target for renames/copies]. The status letter must belong to the
// closed deltaStatuses table; score suffixes on R/C are accepted and
// dropped.
func parseDeltas(output []byte) ([]Delta, error) {
	if len(output) == 0 {
		return nil, nil
	}
	fields := bytes.Split(output, []byte{0})
	deltas := []Delta{}
	index := 0
	for index < len(fields) {
		header := string(fields[index])
		index++
		if strings.TrimSpace(header) == "" {
			continue
		}
		delta, consumesTarget, err := parseDeltaHeader(header)
		if err != nil {
			return nil, err
		}
		if index >= len(fields) {
			return nil, refuse(GateDeltaStatus, "delta without path "+strconv.Quote(header))
		}
		delta.Path = string(fields[index])
		index++
		if _, err := scalar.ParseRelativePath(delta.Path); err != nil {
			return nil, refuse(GateIndexEntry, "delta path invalid: "+err.Error())
		}
		if consumesTarget {
			if index >= len(fields) {
				return nil, refuse(GateDeltaStatus, "rename delta without target "+strconv.Quote(header))
			}
			target := string(fields[index])
			index++
			if _, err := scalar.ParseRelativePath(target); err != nil {
				return nil, refuse(GateIndexEntry, "delta target invalid: "+err.Error())
			}
			delta.Target = &target
		}
		deltas = append(deltas, delta)
	}
	return deltas, nil
}

func parseDeltaHeader(header string) (Delta, bool, error) {
	if !strings.HasPrefix(header, ":") {
		return Delta{}, false, refuse(GateDeltaStatus, "delta header without colon "+strconv.Quote(header))
	}
	parts := strings.Split(strings.TrimPrefix(header, ":"), " ")
	if len(parts) != 5 {
		return Delta{}, false, refuse(GateDeltaStatus, "delta header without five fields "+strconv.Quote(header))
	}
	oldMode, err := strconv.ParseUint(parts[0], 8, 32)
	if err != nil {
		return Delta{}, false, refuse(GateDeltaStatus, "delta old mode not octal "+strconv.Quote(parts[0]))
	}
	newMode, err := strconv.ParseUint(parts[1], 8, 32)
	if err != nil {
		return Delta{}, false, refuse(GateDeltaStatus, "delta new mode not octal "+strconv.Quote(parts[1]))
	}
	status := parts[4]
	letter := status
	if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
		letter = status[:1]
	}
	known := false
	for _, allowed := range deltaStatuses {
		if letter == allowed {
			known = true
			break
		}
	}
	if !known {
		return Delta{}, false, refuse(GateDeltaStatus, "unknown delta status "+strconv.Quote(status))
	}
	delta := Delta{Status: letter, OldMode: uint32(oldMode), NewMode: uint32(newMode)}
	if !isZeroOID(parts[2]) {
		oid := parts[2]
		delta.OldOID = &oid
	}
	if !isZeroOID(parts[3]) {
		oid := parts[3]
		delta.NewOID = &oid
	}
	return delta, letter == "R" || letter == "C", nil
}

func isZeroOID(value string) bool {
	if value == "" {
		return true
	}
	for index := 0; index < len(value); index++ {
		if value[index] != '0' {
			return false
		}
	}
	return true
}

// statIndexPaths stats every index path: lstat kind plus permission
// bits, in sorted order. Gitlink entries (mode 160000) report
// KindGitlink without descending. A device, FIFO, socket, or other
// special file is refused: Section 10.4 carries no member for it and no
// exclusion policy lives in this leaf. It runs before the delta reads so
// an unsupported kind surfaces as GateWorktreeKind instead of hiding
// behind a diff subprocess failure.
func statIndexPaths(repoRoot string, entries []IndexEntry) ([]PathMode, map[string]bool, error) {
	modes := make([]PathMode, 0, len(entries))
	statted := map[string]bool{}
	for _, entry := range entries {
		if statted[entry.Path] {
			continue
		}
		kind, perm, err := statOne(repoRoot, entry.Path, entry.Mode)
		if err != nil {
			return nil, nil, err
		}
		modes = append(modes, PathMode{Path: entry.Path, Kind: kind, PermBits: perm, Exec: perm&0o111 != 0})
		statted[entry.Path] = true
	}
	return modes, statted, nil
}

// statDeltaPaths stats delta paths the index did not already cover
// (deletions name worktree-absent paths; renames name new targets) and
// appends them in sorted order so PathModes stays sorted.
func statDeltaPaths(repoRoot string, modes *[]PathMode, statted map[string]bool, staged, unstaged []Delta) error {
	extra := map[string]uint32{}
	for _, delta := range append(append([]Delta{}, staged...), unstaged...) {
		if !statted[delta.Path] {
			extra[delta.Path] = delta.NewMode
		}
		if delta.Target != nil && !statted[*delta.Target] {
			extra[*delta.Target] = delta.NewMode
		}
	}
	ordered := make([]string, 0, len(extra))
	for name := range extra {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	merged := append(*modes, []PathMode{}...)
	for _, name := range ordered {
		kind, perm, err := statOne(repoRoot, name, extra[name])
		if err != nil {
			return err
		}
		merged = append(merged, PathMode{Path: name, Kind: kind, PermBits: perm, Exec: perm&0o111 != 0})
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Path < merged[j].Path })
	*modes = merged
	return nil
}

func statOne(repoRoot, name string, indexMode uint32) (WorktreeKind, uint32, error) {
	if indexMode == 0o160000 {
		return KindGitlink, 0, nil
	}
	full := filepath.Join(repoRoot, filepath.FromSlash(name))
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return KindMissing, 0, nil
		}
		return "", 0, refuse(GateNotRepository, "stat failed for "+strconv.Quote(name)+": "+err.Error())
	}
	mode := info.Mode()
	switch {
	case mode.IsDir():
		return KindDir, 0, nil
	case mode&os.ModeSymlink != 0:
		return KindSymlink, 0o777, nil
	case mode.IsRegular():
		perm := uint32(mode.Perm())
		return KindFile, perm, nil
	default:
		return "", 0, refuse(GateWorktreeKind, "unsupported special file "+strconv.Quote(name))
	}
}

// absentResult accepts only the documented quiet not-present result. Partial
// output and diagnostics cannot attest absence.
func absentResult(result GitResult) bool {
	return result.ExitCode == 1 && len(result.Stdout) == 0 && len(result.Stderr) == 0
}

func readOptionalConfig(ctx context.Context, runner Runner, dir string, args ...string) (GitResult, error) {
	result, err := runner.Run(ctx, dir, args...)
	if err != nil {
		return GitResult{}, refuse(GateNotRepository, "config read failed: "+err.Error())
	}
	if result.ExitCode != 0 && !absentResult(result) {
		return GitResult{}, refuse(GateNotRepository, "config read failed")
	}
	return result, nil
}

func readFeatureBool(ctx context.Context, runner Runner, dir, key string, fallback bool) (bool, error) {
	result, err := readOptionalConfig(ctx, runner, dir, "config", "--bool", "--get", key)
	if err != nil {
		return false, err
	}
	if absentResult(result) {
		return fallback, nil
	}
	switch string(result.Stdout) {
	case "true\n":
		return true, nil
	case "false\n":
		return false, nil
	default:
		return false, refuse(GateNotRepository, "noncanonical boolean config read")
	}
}

func readFeatures(ctx context.Context, runner Runner, dir, objectFormat string) (Features, error) {
	features := Features{ObjectFormat: objectFormat}
	// Defaults describe Git configuration behavior, not a filesystem capability probe.
	for _, field := range []struct {
		key      string
		target   *bool
		fallback bool
	}{
		{"core.filemode", &features.FileMode, true},
		{"core.symlinks", &features.Symlinks, true},
		{"core.ignorecase", &features.CaseSensitive, false},
		{"core.precomposeunicode", &features.PrecomposeUnicode, false},
		{"core.sparseCheckout", &features.SparseCheckout, false},
	} {
		value, err := readFeatureBool(ctx, runner, dir, field.key, field.fallback)
		if err != nil {
			return Features{}, err
		}
		*field.target = value
	}
	features.CaseSensitive = !features.CaseSensitive
	required, err := readOptionalConfig(ctx, runner, dir, "config", "-z", "--name-only", "--get-regexp", `^filter\..*\.required$`)
	if err != nil {
		return Features{}, err
	}
	if required.ExitCode == 0 {
		if len(required.Stdout) == 0 || required.Stdout[len(required.Stdout)-1] != 0 {
			return Features{}, refuse(GateNotRepository, "malformed filter config list")
		}
		names := map[string]bool{}
		for _, key := range strings.Split(string(required.Stdout[:len(required.Stdout)-1]), "\x00") {
			if !strings.HasPrefix(key, "filter.") || !strings.HasSuffix(key, ".required") {
				return Features{}, refuse(GateNotRepository, "malformed filter key")
			}
			name := strings.TrimSuffix(strings.TrimPrefix(key, "filter."), ".required")
			if name == "" || !utf8.ValidString(name) {
				return Features{}, refuse(GateNotRepository, "invalid filter name")
			}
			// Let Git parse every spelling and apply last-value precedence. A census
			// match disappearing before this read is a failed observation, not false.
			result, err := runOK(ctx, runner, dir, "config", "--bool", "--get", key)
			if err != nil {
				return Features{}, err
			}
			switch string(result) {
			case "true\n":
				names[name] = true
			case "false\n":
				names[name] = false
			default:
				return Features{}, refuse(GateNotRepository, "noncanonical required-filter boolean")
			}
		}
		for name, required := range names {
			if required {
				features.RequiredFilters = append(features.RequiredFilters, name)
			}
		}
		if len(features.RequiredFilters) > maxRequiredFilters {
			return Features{}, refuse(GateFeatures, "required filters exceed 64")
		}
		sort.Strings(features.RequiredFilters)
	}
	for _, name := range features.RequiredFilters {
		if name == "lfs" {
			features.LFSRequired = true
		}
	}
	return features, nil
}

// indexState is an in-memory sentinel, not a raw-index deliverable. Hash the
// actual index, including extensions and format bytes, without writing a blob.
type indexState struct {
	path   string
	info   os.FileInfo
	digest [sha256.Size]byte
}

func readIndexState(ctx context.Context, runner Runner, dir string) (indexState, error) {
	output, err := runOK(ctx, runner, dir, "rev-parse", "--path-format=absolute", "--git-path", "index")
	if err != nil {
		return indexState{}, err
	}
	path := strings.TrimSuffix(string(output), "\n")
	if !filepath.IsAbs(path) {
		return indexState{}, refuse(GateNotRepository, "index path is not absolute")
	}
	state := indexState{path: path}
	file, openErr := os.Open(path)
	if os.IsNotExist(openErr) {
		return state, nil
	}
	if openErr != nil {
		return state, refuse(GateNotRepository, "index sentinel open failed")
	}
	defer file.Close()
	info, statErr := file.Stat()
	if statErr != nil || !info.Mode().IsRegular() {
		return state, refuse(GateNotRepository, "index sentinel stat failed")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return state, refuse(GateNotRepository, "index sentinel read failed")
	}
	state.info = info
	copy(state.digest[:], hash.Sum(nil))
	return state, nil
}

func (state indexState) equal(other indexState) bool {
	if state.path != other.path || state.digest != other.digest {
		return false
	}
	if state.info == nil || other.info == nil {
		return state.info == nil && other.info == nil
	}
	return os.SameFile(state.info, other.info) && state.info.Size() == other.info.Size() && state.info.ModTime().Equal(other.info.ModTime())
}

// recheckConsistency compares full HEAD state, the actual index sentinel and
// logical entries, and both raw delta streams. This bounded observation is not
// an atomic snapshot or a working-tree content digest (owned by sibling leaves).
func recheckConsistency(ctx context.Context, runner Runner, dir string, head Head, objectFormat string, version int, indexState indexState, indexBytes, stagedBytes, unstagedBytes []byte) error {
	nowHead, err := readHead(ctx, runner, dir, objectFormat)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(nowHead, head) {
		return refuse(GateConsistency, "HEAD moved during capture")
	}
	again, refusal := runOK(ctx, runner, dir, "ls-files", "--stage", "--debug", "-z")
	if refusal != nil {
		return refusal
	}
	if !bytes.Equal(again, indexBytes) {
		return refuse(GateConsistency, "index moved during capture")
	}
	nowState, stateErr := readIndexState(ctx, runner, dir)
	if stateErr != nil {
		return stateErr
	}
	nowVersion, versionErr := readIndexVersion(ctx, runner, dir)
	if versionErr != nil {
		return versionErr
	}
	if !indexState.equal(nowState) || version != nowVersion {
		return refuse(GateConsistency, "actual index moved during capture")
	}
	stagedAgain, refusal := runOK(ctx, runner, dir, "diff", "--cached", "--raw", "--no-abbrev", "-z")
	if refusal != nil {
		return refusal
	}
	if !bytes.Equal(stagedAgain, stagedBytes) {
		return refuse(GateConsistency, "staged content moved during capture")
	}
	unstagedAgain, refusal := runOK(ctx, runner, dir, "diff", "--raw", "--no-abbrev", "-z")
	if refusal != nil {
		return refusal
	}
	if !bytes.Equal(unstagedAgain, unstagedBytes) {
		return refuse(GateConsistency, "unstaged content moved during capture")
	}
	return nil
}

// digestSnapshot returns the SHA-256 hex of the snapshot encoding for
// idempotency comparisons.
func digestSnapshot(value *Snapshot) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Snapshot is an internal capture result, not a complete wire member.
// See doc.go for validated domains and the sibling-owned wire fields.
type Snapshot struct {
	observedIndex      indexState
	Content            *Content   `json:"content,omitempty"`
	RepositoryIdentity string     `json:"repository_identity"`
	Remotes            []Remote   `json:"remotes"`
	Head               Head       `json:"head"`
	UpstreamRef        *string    `json:"upstream_ref"`
	ObjectFormat       string     `json:"object_format"`
	Worktree           Worktree   `json:"worktree"`
	Index              Index      `json:"index"`
	Staged             []Delta    `json:"staged"`
	Unstaged           []Delta    `json:"unstaged"`
	PathModes          []PathMode `json:"path_modes"`
	Features           Features   `json:"features"`
}
