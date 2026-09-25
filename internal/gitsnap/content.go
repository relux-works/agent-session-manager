package gitsnap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// ContentOptions enables working-copy content capture in Capture. Without it,
// Capture retains the repository/index observation API. Store must be outside
// the checkout. Policies are trusted project/provider configuration, not proof
// supplied by files being captured. Paths are relative to the outer repository,
// including submodule prefixes. No content-level confidentiality is asserted.
type ContentOptions struct {
	Store *localstore.ObjectStore
	// IncludeIgnored names exact files, positively classified as non-secret by
	// the project/provider owner. False classifications cannot authorize capture.
	IncludeIgnored map[string]bool
	// Exclude names files or directory subtrees and their diagnostic classes.
	// Exclusions always win over includes, including for tracked content.
	Exclude map[string]string
	// LocalPaths is the existing AX path owner's resolved layout. Its paths are
	// excluded when nested inside this checkout; local runtime state is not history.
	LocalPaths *localstore.ResolvedPaths
}

// Content is the captured working-copy leaf, not a workspace-group wire object.
// Entries use the exact Section 10.4 tagged shapes; descriptors and bytes are
// installed in Store before this value can be returned. AssembleProvisional
// separately constructs packs, raw indexes and workspace-root/tree manifests.
type Content struct {
	Entries         []map[string]any `json:"entries"`
	Blobs           []BlobDescriptor `json:"blobs"`
	ExcludedClasses []string         `json:"excluded_classes"`
	Submodules      []Submodule      `json:"submodules"`
	SparsePatterns  *BlobDescriptor  `json:"sparse_patterns"`
}

type BlobChunk struct {
	Index   uint32 `json:"index"`
	Offset  uint64 `json:"offset"`
	Size    uint64 `json:"size"`
	ChunkID string `json:"chunk_id"`
}

type BlobDescriptor struct {
	Schema        string      `json:"schema"`
	SchemaVersion string      `json:"schema_version"`
	DescriptorID  string      `json:"descriptor_id"`
	BlobID        string      `json:"blob_id"`
	Size          uint64      `json:"size"`
	MediaType     string      `json:"media_type"`
	Chunks        []BlobChunk `json:"chunks"`
}

// Submodule preserves the stage-0 pointer independently of the recursive child
// HEAD and index. State is nil only for an observed uninitialized checkout.
// The parent HEAD-tree pointer remains derivable from its HEAD; no child object
// is read through the parent object database.
type Submodule struct {
	Path               string    `json:"path"`
	RepositoryIdentity string    `json:"repository_identity"`
	SanitizedURL       string    `json:"sanitized_url"`
	GitlinkOID         string    `json:"gitlink_oid"`
	Initialized        bool      `json:"initialized"`
	State              *Snapshot `json:"state"`
}

type contentState struct {
	localRoots []string
	options    ContentOptions
	active     map[string]bool
	count      int
}

const chunkSize = 4 * 1024 * 1024
const maxBlobSize int64 = 128 * 1024 * 1024 * 1024

func (s *contentState) validate() error {
	if s.options.Store == nil {
		return refuse(GateCapturePolicy, "content capture requires an immutable object store")
	}
	for name, nonSecret := range s.options.IncludeIgnored {
		if _, err := scalar.ParseRelativePath(name); err != nil || !nonSecret {
			return refuse(GateCapturePolicy, "ignored include requires a valid path and non-secret classification")
		}
	}
	for name, class := range s.options.Exclude {
		if _, err := scalar.ParseRelativePath(name); err != nil || class == "" || !utf8.ValidString(class) {
			return refuse(GateCapturePolicy, "exclusion requires a valid path and class")
		}
	}
	if s.options.LocalPaths != nil {
		for _, local := range s.options.LocalPaths.Paths() {
			root, err := canonicalPolicyPath(local.Value.String())
			if err != nil {
				return refuse(GateCapturePolicy, "cannot resolve machine-local exclusion root")
			}
			s.localRoots = append(s.localRoots, root)
		}
	}
	return nil
}

func canonicalPolicyPath(name string) (string, error) {
	resolved, err := filepath.EvalSymlinks(name)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	info, statErr := os.Lstat(name)
	if statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", err
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	parent := filepath.Dir(name)
	if parent == name {
		return "", err
	}
	resolved, err = canonicalPolicyPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(name)), nil
}

func pathWithin(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

// excluded is the one capture-selection owner shared by the parent and every
// child. It does not use the diagnostic redactor as a secret classifier.
func (s *contentState) excluded(root, name, prefix string, mode fs.FileMode) string {
	logical := path.Join(prefix, name)
	excludedNames := make([]string, 0, len(s.options.Exclude))
	for name := range s.options.Exclude {
		excludedNames = append(excludedNames, name)
	}
	sort.Strings(excludedNames)
	for _, excluded := range excludedNames {
		class := s.options.Exclude[excluded]
		if logical == excluded || strings.HasPrefix(logical, excluded+"/") {
			return class
		}
	}
	for _, localRoot := range s.localRoots {
		if pathWithin(localRoot, filepath.Join(root, filepath.FromSlash(name))) {
			return "machine_local"
		}
	}

	for _, component := range strings.Split(name, "/") {
		switch component {
		case ".git":
			return "git_metadata"
		case ".ssh", "auth.json", "credentials.json", ".git-credentials":
			return "credential"
		case ".env":
			return "environment_secret"
		}
	}
	base := path.Base(name)
	if strings.HasPrefix(base, ".env.") {
		return "environment_secret"
	}
	if strings.HasSuffix(base, ".pid") {
		return "live_pid"
	}
	if base == "updater.lock" || base == "provider.pid.lock" || strings.HasSuffix(base, "-wal") || strings.HasSuffix(base, "-shm") || strings.HasSuffix(base, "-journal") {
		return "transient_lock"
	}
	if mode&os.ModeSocket != 0 || mode&os.ModeNamedPipe != 0 {
		return "socket"
	}
	return ""
}

func (s *contentState) capture(ctx context.Context, runner Runner, repo *repositoryFacts, index []IndexEntry, features Features, head Head, remotes []Remote, prefix string, depth int) (*Content, error) {
	root := repo.worktree.RepoRoot
	if repo.worktree.IsBare {
		return nil, refuse(GateCapturePolicy, "content capture requires a working tree")
	}
	storeRoot, err := filepath.EvalSymlinks(s.options.Store.DataRoot())
	if err != nil {
		return nil, refuse(GateCapturePolicy, "cannot resolve object-store root")
	}
	if pathWithin(root, storeRoot) {
		return nil, refuse(GateCapturePolicy, "object store must be outside the captured checkout")
	}
	// Canonical Git directories detect recursion cycles even for linked worktrees.
	gitDir, err := filepath.EvalSymlinks(repo.worktree.GitDir)
	if err != nil {
		return nil, refuse(GateContentRead, "cannot resolve Git directory")
	}
	if s.active[gitDir] {
		return nil, refuse(GateSubmoduleState, "recursive repository cycle")
	}
	s.active[gitDir] = true
	defer delete(s.active, gitDir)
	// Exit 0 alone does not attest a complete ignore census: Git warns and
	// continues when an exclusion source is unreadable. Keep diagnostics at
	// this policy boundary and refuse any indeterminate census, without parsing
	// filenames or localized warning text. Legitimately absent optional policy
	// files produce no diagnostic and retain Git's normal absence semantics.
	ignoredResult, err := runner.Run(ctx, root, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z")
	if err != nil || ignoredResult.ExitCode != 0 || len(ignoredResult.Stderr) != 0 {
		return nil, refuse(GateContentRead, "ignored-path enumeration failed")
	}
	ignored, err := nulPaths(ignoredResult.Stdout)
	if err != nil {
		return nil, err
	}
	links := map[string]IndexEntry{}
	for _, entry := range index {
		if entry.Mode == 0o160000 {
			if entry.Stage != 0 {
				return nil, refuse(GateSubmoduleState, "conflicted submodule index is not representable")
			}
			links[entry.Path] = entry
		}
	}
	tree := &Content{Entries: []map[string]any{}, Blobs: []BlobDescriptor{}, ExcludedClasses: []string{}, Submodules: []Submodule{}}
	classes := map[string]bool{}
	platform := scalar.PlatformLinux
	if runtime.GOOS == "darwin" {
		platform = scalar.PlatformMacOS
	}
	if runtime.GOOS == "windows" {
		platform = scalar.PlatformWindows
	}
	guard, err := secprim.NewGuard(root, platform, nil)
	if err != nil {
		return nil, refuse(GateContentPath, "invalid content root")
	}
	rootHandle, err := secprim.OpenNoFollowDir(root)
	if err != nil {
		return nil, refuse(GateContentRead, "cannot open content root")
	}
	defer rootHandle.Close()
	captured := map[string]BlobDescriptor{}
	symlinks := map[string]string{}
	err = filepath.WalkDir(root, func(native string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return refuse(GateContentRead, "working-tree enumeration failed")
		}
		if err := ctx.Err(); err != nil {
			return refuse(GateContentRead, "capture cancelled")
		}
		if native == root {
			return nil
		}
		relative, err := filepath.Rel(root, native)
		if err != nil {
			return refuse(GateContentPath, "cannot resolve content path")
		}
		name := filepath.ToSlash(relative)
		info, err := item.Info()
		if err != nil {
			return refuse(GateContentRead, "cannot stat content path")
		}
		if class := s.excluded(root, name, prefix, info.Mode()); class != "" {
			classes[class] = true
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if _, err := scalar.ParseRelativePath(name); err != nil {
			return refuse(GateContentPath, "invalid content path")
		}
		skip := false
		for _, ignore := range ignored {
			ignore = strings.TrimSuffix(ignore, "/")
			if name == ignore || strings.HasPrefix(name, ignore+"/") {
				skip = true
				break
			}
		}
		if skip {
			wanted := s.options.IncludeIgnored[path.Join(prefix, name)]
			if item.IsDir() {
				for include := range s.options.IncludeIgnored {
					if strings.HasPrefix(include, path.Join(prefix, name)+"/") {
						wanted = true
						break
					}
				}
			}
			if !wanted {
				classes["ignored"] = true
				if item.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		mode := uint32(info.Mode().Perm())
		if info.Mode()&os.ModeSetuid != 0 {
			mode |= 0o4000
		}
		if info.Mode()&os.ModeSetgid != 0 {
			mode |= 0o2000
		}
		if info.Mode()&os.ModeSticky != 0 {
			mode |= 0o1000
		}
		entry := map[string]any{"path": name, "mode": mode}
		switch {
		case info.IsDir():
			entry["type"] = "directory"
		case info.Mode().IsRegular():
			file, err := guard.Open(rootHandle, name)
			if err != nil {
				return refuse(GateContentRead, "cannot safely open content file")
			}
			opened, statErr := file.Stat()
			if statErr != nil || !os.SameFile(info, opened) {
				file.Close()
				return refuse(GateConsistency, "content file replaced before open")
			}
			descriptor, err := s.blob(ctx, file, info.Size(), "application/octet-stream")
			closeErr := file.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return refuse(GateContentRead, "content file close failed")
			}
			captured[name] = descriptor
			tree.Blobs = append(tree.Blobs, descriptor)
			entry["type"], entry["size"], entry["blob_id"], entry["blob_descriptor_id"] = "file", descriptor.Size, descriptor.BlobID, descriptor.DescriptorID
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(native)
			if err != nil {
				return refuse(GateContentRead, "cannot read symlink")
			}
			if err := safeSymlink(root, name, target); err != nil {
				return err
			}
			symlinks[name] = target
			entry["type"], entry["target"] = "symlink", target
		default:
			return refuse(GateWorktreeKind, "unsupported included filesystem kind")
		}
		tree.Entries = append(tree.Entries, entry)
		if _, isLink := links[name]; isLink {
			if !info.IsDir() {
				return refuse(GateSubmoduleState, "submodule path is not a directory")
			}
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	tree.Submodules, err = s.submodules(ctx, runner, root, links, head, remotes, prefix, depth)
	if err != nil {
		return nil, err
	}
	if features.SparseCheckout {
		result, failure := runOK(ctx, runner, root, "rev-parse", "--path-format=absolute", "--git-path", "info/sparse-checkout")
		if failure != nil {
			return nil, refuse(GateContentRead, "cannot locate sparse patterns")
		}
		file, err := secprim.OpenNoFollowFile(strings.TrimSuffix(string(result), "\n"))
		if err != nil {
			return nil, refuse(GateContentRead, "cannot read sparse patterns")
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, refuse(GateContentRead, "cannot stat sparse patterns")
		}
		descriptor, err := s.blob(ctx, file, info.Size(), "text/plain")
		closeErr := file.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, refuse(GateContentRead, "cannot close sparse patterns")
		}
		tree.SparsePatterns = &descriptor
	}
	// Re-open through the same guard: missing/unreadable bytes never count as an
	// absent optional file. Store installs are inert if this closing check fails.
	for name, descriptor := range captured {
		file, err := guard.Open(rootHandle, name)
		if err != nil {
			return nil, refuse(GateContentRead, "cannot re-read captured file")
		}
		hash := sha256.New()
		n, readErr := io.Copy(hash, &contextReader{ctx: ctx, reader: file})
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			return nil, refuse(GateContentRead, "captured file re-read failed")
		}
		if uint64(n) != descriptor.Size || digestHex(hash.Sum(nil)) != descriptor.BlobID {
			return nil, refuse(GateConsistency, "included file changed during capture")
		}
	}
	for name, target := range symlinks {
		current, err := os.Readlink(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || current != target {
			return nil, refuse(GateConsistency, "symlink changed during capture")
		}
		if err := safeSymlink(root, name, current); err != nil {
			return nil, err
		}
	}
	sort.Slice(tree.Entries, func(i, j int) bool { return tree.Entries[i]["path"].(string) < tree.Entries[j]["path"].(string) })
	rawEntries, err := json.Marshal(tree.Entries)
	if err != nil {
		return nil, refuse(GateContentPath, "cannot encode content entries")
	}
	if err := canonicaljson.CheckManifestEntries(rawEntries); err != nil {
		return nil, refuse(GateContentPath, "content entries violate the manifest contract")
	}
	for class := range classes {
		tree.ExcludedClasses = append(tree.ExcludedClasses, class)
	}
	sort.Strings(tree.ExcludedClasses)
	if len(tree.ExcludedClasses) > 128 {
		return nil, refuse(GateCapturePolicy, "exclusion class count exceeds 128")
	}
	return tree, nil
}

func nulPaths(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[len(raw)-1] != 0 {
		return nil, refuse(GateContentRead, "partial path enumeration")
	}
	paths := strings.Split(string(raw[:len(raw)-1]), "\x00")
	for _, name := range paths {
		if _, err := scalar.ParseRelativePath(strings.TrimSuffix(name, "/")); err != nil {
			return nil, refuse(GateContentPath, "invalid enumerated path")
		}
	}
	return paths, nil
}

func safeSymlink(root, name, target string) error {
	if !utf8.ValidString(target) || utf8.RuneCountInString(target) < 1 || utf8.RuneCountInString(target) > 4096 || strings.ContainsAny(target, "\\\x00") || path.IsAbs(target) {
		return refuse(GateContentPath, "invalid symlink target")
	}
	joined := filepath.Join(root, filepath.FromSlash(path.Dir(name)), filepath.FromSlash(target))
	if !pathWithin(root, joined) {
		return refuse(GateContentPath, "symlink target escapes root")
	}
	// Resolve components without cleaning away a symlink/.. pair before
	// filesystem inspection. Missing ordinary components are safe dangling
	// targets; inaccessible components and cycles are not evidence of absence.
	pending := strings.Split(path.Dir(name)+"/"+target, "/")
	resolved := []string{}
	hops := 0
	for len(pending) > 0 {
		component := pending[0]
		pending = pending[1:]
		if component == "" || component == "." {
			continue
		}
		if component == ".." {
			if len(resolved) == 0 {
				return refuse(GateContentPath, "symlink resolves outside root")
			}
			resolved = resolved[:len(resolved)-1]
			continue
		}
		candidate := filepath.Join(append([]string{root}, append(resolved, component)...)...)
		info, err := os.Lstat(candidate)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return refuse(GateContentRead, "symlink ancestor read failed")
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			hops++
			if hops > 40 {
				return refuse(GateContentPath, "symlink cycle or excessive chain")
			}
			next, err := os.Readlink(candidate)
			if err != nil {
				return refuse(GateContentRead, "symlink chain read failed")
			}
			if filepath.IsAbs(next) || strings.ContainsAny(next, "\\\x00") {
				return refuse(GateContentPath, "symlink chain escapes root")
			}
			pending = append(strings.Split(next, "/"), pending...)
			continue
		}
		resolved = append(resolved, component)
	}
	return nil
}

func digestHex(sum []byte) string { return "sha256:" + hex.EncodeToString(sum) }

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// blob streams through fixed-size chunks and then rewinds the open file. The immutable
// store remains the sole fsync/verification/install owner. A failed capture can
// leave verified, unreferenced blobs; it never publishes a partial manifest.
func (s *contentState) blob(ctx context.Context, source io.ReadSeeker, size int64, mediaType string) (BlobDescriptor, error) {
	if size < 0 || size > maxBlobSize {
		return BlobDescriptor{}, refuse(GateBlobLimit, "blob exceeds protocol 1.0.0 size bound")
	}

	descriptor := BlobDescriptor{Schema: "urn:ax:schema:blob", SchemaVersion: "1.0.0", MediaType: mediaType, DescriptorID: "sha256:" + strings.Repeat("0", 64), Chunks: []BlobChunk{}}
	full := sha256.New()
	buffer := make([]byte, chunkSize)
	reader := &contextReader{ctx: ctx, reader: source}
	for {
		n, readErr := io.ReadFull(reader, buffer)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return BlobDescriptor{}, refuse(GateContentRead, "blob read failed")
		}
		if n > 0 {
			if descriptor.Size+uint64(n) > uint64(size) {
				return BlobDescriptor{}, refuse(GateConsistency, "blob grew during capture")
			}
			part := buffer[:n]
			sum := sha256.Sum256(part)
			id := digestHex(sum[:])
			parsed, _ := scalar.ParseDigest(id)
			if err := s.installBlob(parsed, uint64(n), bytes.NewReader(part)); err != nil {
				return BlobDescriptor{}, err
			}

			full.Write(part)
			descriptor.Chunks = append(descriptor.Chunks, BlobChunk{Index: uint32(len(descriptor.Chunks)), Offset: descriptor.Size, Size: uint64(n), ChunkID: id})
			descriptor.Size += uint64(n)
		}
		if readErr != nil {
			break
		}
	}
	if descriptor.Size != uint64(size) {
		return BlobDescriptor{}, refuse(GateConsistency, "blob size changed during capture")
	}
	descriptor.BlobID = digestHex(full.Sum(nil))
	id, _ := scalar.ParseDigest(descriptor.BlobID)
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return BlobDescriptor{}, refuse(GateContentRead, "blob source seek failed")
	}
	if err := s.installBlob(id, descriptor.Size, &contextReader{ctx: ctx, reader: source}); err != nil {
		return BlobDescriptor{}, err
	}
	raw, _ := json.Marshal(descriptor)
	identity, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		return BlobDescriptor{}, refuse(GateBlobInstall, "invalid blob descriptor")
	}
	descriptor.DescriptorID = identity.String()
	raw, _ = json.Marshal(descriptor)
	// Descriptor identity omits descriptor_id; its storage address instead hashes
	// the complete representation, as required by the uninterpreted blob store.
	sum := sha256.Sum256(raw)
	rawID, _ := scalar.ParseDigest(digestHex(sum[:]))
	if err := s.installBlob(rawID, uint64(len(raw)), bytes.NewReader(raw)); err != nil {
		return BlobDescriptor{}, err
	}
	return descriptor, nil
}

func (s *contentState) installBlob(id scalar.Digest, size uint64, source io.Reader) error {
	if _, err := s.options.Store.PutBlob(id, size, source); err != nil {
		return refuse(GateBlobInstall, "immutable blob install failed")
	}
	return nil
}
