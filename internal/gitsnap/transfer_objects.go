package gitsnap

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// AssemblyRunner extends the observation seam with stdin and an isolated Git
// database. Only disposable verifier repositories receive mandatory Git writes.
type AssemblyRunner interface {
	Runner
	RunInput(context.Context, string, []byte, ...string) (GitResult, error)
	RunIsolated(context.Context, string, []byte, ...string) (GitResult, error)
}

type repositoryObjects struct {
	pack, inventory, index BlobDescriptor
	count                  int
}

func assemblyOutput(result GitResult, err error) ([]byte, error) {
	if err != nil || result.ExitCode != 0 || len(result.Stderr) != 0 {
		return nil, refuse(GateContentRead, "Git assembly read or verification failed")
	}
	return result.Stdout, nil
}

func assemblyRun(ctx context.Context, runner Runner, root string, args ...string) ([]byte, error) {
	return assemblyOutput(runner.Run(ctx, root, args...))
}

func isolatedRun(ctx context.Context, runner AssemblyRunner, root string, input []byte, args ...string) ([]byte, error) {
	return assemblyOutput(runner.RunIsolated(ctx, root, input, args...))
}

// requiredObjects names only this repository's roots. Gitlinks are opaque child
// names, never parent roots. The HEAD and upstream are the refs recorded on wire.
func requiredObjects(ctx context.Context, runner Runner, snapshot *Snapshot) ([]string, error) {
	names := map[string]bool{}
	if snapshot.Head.OID != nil {
		names[strings.TrimPrefix(*snapshot.Head.OID, snapshot.ObjectFormat+":")] = true
	}
	if snapshot.UpstreamRef != nil {
		raw, err := assemblyRun(ctx, runner, snapshot.Worktree.RepoRoot, "rev-parse", "--verify", *snapshot.UpstreamRef+"^{object}")
		if err != nil {
			return nil, err
		}
		oid := strings.TrimSuffix(string(raw), "\n")
		if _, err := scalar.ParseGitOID(snapshot.ObjectFormat + ":" + oid); err != nil {
			return nil, refuse(GateHeadOIDFormat, "invalid upstream object")
		}
		names[oid] = true
	}
	for _, entry := range snapshot.Index.Entries {
		if entry.Mode != 0o160000 {
			names[strings.TrimPrefix(entry.OID, snapshot.ObjectFormat+":")] = true
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func objectLines(names []string) []byte {
	if len(names) == 0 {
		return nil
	}
	return []byte(strings.Join(names, "\n") + "\n")
}

func (s *assemblyState) objects(ctx context.Context, snapshot *Snapshot) (repositoryObjects, error) {
	var result repositoryObjects
	root := snapshot.Worktree.RepoRoot
	if snapshot.Worktree.IsShallow {
		return result, refuse(GateFeatures, "shallow history has no self-contained offline closure")
	}
	if err := checkAssemblySource(ctx, s.runner, root); err != nil {
		return result, err
	}
	roots, err := requiredObjects(ctx, s.runner, snapshot)
	if err != nil {
		return result, err
	}
	before, err := readIndexState(ctx, s.runner, root)
	if err != nil {
		return result, err
	}
	if !snapshot.observedIndex.equal(before) {
		return result, refuse(GateConsistency, "index changed after content capture")
	}
	scratch, err := os.MkdirTemp(s.options.ScratchRoot, "gitsnap-verify-")
	if err != nil {
		return result, refuse(GateContentRead, "cannot create isolated verifier")
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	if _, err := isolatedRun(ctx, s.runner, scratch, nil, "init", "--bare", "-q", "--template=", "--object-format="+snapshot.ObjectFormat, "."); err != nil {
		return result, err
	}
	pack, err := assemblyOutput(s.runner.RunInput(ctx, root, objectLines(roots), "pack-objects", "--stdout", "--revs", "--no-reuse-delta", "--no-reuse-object", "--window=0", "--threads=1", "-q"))
	if err != nil {
		return result, err
	}
	if len(pack) < 12 || string(pack[:4]) != "PACK" || binary.BigEndian.Uint32(pack[4:8]) != 2 {
		return result, refuse(GateTransferObjects, "invalid Git pack header")
	}
	result.count = int(binary.BigEndian.Uint32(pack[8:12]))
	if _, err := isolatedRun(ctx, s.runner, scratch, pack, "index-pack", "--stdin", "--strict"); err != nil {
		return result, err
	}
	// Inventory is derived from the imported pack, not the source database (which
	// may contain unreachable objects). The fresh database has no alternates.
	inventory, err := isolatedRun(ctx, s.runner, scratch, nil, "cat-file", "--batch-all-objects", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return result, err
	}
	inventory, err = checkInventory(inventory, snapshot.ObjectFormat, result.count, roots)
	if err != nil {
		return result, err
	}
	if snapshot.Head.OID != nil {
		if _, err := isolatedRun(ctx, s.runner, scratch, nil, "cat-file", "-e", strings.TrimPrefix(*snapshot.Head.OID, snapshot.ObjectFormat+":")+"^{commit}"); err != nil {
			return result, err
		}
	}
	// Derive exact membership by traversal, then independently ask Git for each
	// object's type/size in the isolated database. A plausible inventory line is
	// not proof that its metadata matches the imported bytes.
	expected := []byte{}
	if len(roots) > 0 {
		expected, err = isolatedRun(ctx, s.runner, scratch, objectLines(roots), "rev-list", "--objects", "--no-object-names", "--stdin")
		if err != nil {
			return result, err
		}
	}
	expectedNames := []string{}
	if len(expected) > 0 {
		expectedNames = strings.Split(strings.TrimSuffix(string(expected), "\n"), "\n")
	}
	sort.Strings(expectedNames)
	details, err := isolatedRun(ctx, s.runner, scratch, objectLines(expectedNames), "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return result, err
	}
	if !bytes.Equal(details, inventory) {
		return result, refuse(GateTransferObjects, "inventory differs from exact reachable pack objects")
	}
	for _, entry := range snapshot.Index.Entries {
		if entry.Mode != 0o160000 {
			oid := strings.TrimPrefix(entry.OID, snapshot.ObjectFormat+":")
			if !bytes.Contains(inventory, []byte(oid+" blob ")) {
				return result, refuse(GateTransferObjects, "non-gitlink index object is not a blob")
			}
		}
	}

	raw, err := s.rawIndex(ctx, snapshot, before, scratch)
	if err != nil {
		return result, err
	}
	after, err := readIndexState(ctx, s.runner, root)
	if err != nil {
		return result, err
	}
	again, err := requiredObjects(ctx, s.runner, snapshot)
	if err != nil {
		return result, err
	}
	if !before.equal(after) || !reflect.DeepEqual(roots, again) {
		return result, refuse(GateConsistency, "index or recorded refs changed during object assembly")
	}
	result.pack, err = s.addBlob(ctx, pack, "application/x-git-packed-objects")
	if err != nil {
		return result, err
	}
	result.inventory, err = s.addBlob(ctx, inventory, "text/plain")
	if err != nil {
		return result, err
	}
	result.index, err = s.addBlob(ctx, raw, "application/vnd.git.index")
	return result, err
}

func checkInventory(raw []byte, format string, count int, required []string) ([]byte, error) {
	lines := []string{}
	if len(raw) > 0 {
		if raw[len(raw)-1] != '\n' {
			return nil, refuse(GateTransferObjects, "partial object inventory")
		}
		lines = strings.Split(string(raw[:len(raw)-1]), "\n")
	}
	sort.Strings(lines)
	found := map[string]bool{}
	for _, line := range lines {
		fields := strings.Split(line, " ")
		if len(fields) != 3 {
			return nil, refuse(GateTransferObjects, "malformed object inventory")
		}
		oid, err := scalar.ParseGitOID(format + ":" + fields[0])
		if err != nil || oid.String() != format+":"+fields[0] || found[fields[0]] {
			return nil, refuse(GateTransferObjects, "invalid or duplicate inventory object")
		}
		switch fields[1] {
		case "commit", "tree", "blob", "tag":
		default:
			return nil, refuse(GateTransferObjects, "unsupported inventory object type")
		}
		size, err := strconv.ParseUint(fields[2], 10, 53)
		if err != nil || strconv.FormatUint(size, 10) != fields[2] {
			return nil, refuse(GateTransferObjects, "invalid inventory object size")
		}
		found[fields[0]] = true
	}
	if len(lines) != count {
		return nil, refuse(GateTransferObjects, "pack and inventory count disagree")
	}
	for _, oid := range required {
		if !found[oid] {
			return nil, refuse(GateTransferObjects, "required repository object missing from pack")
		}
	}
	return objectLines(lines), nil
}

func (s *assemblyState) rawIndex(ctx context.Context, snapshot *Snapshot, before indexState, scratch string) ([]byte, error) {
	target := filepath.Join(scratch, "index")
	if before.info == nil {
		// An unborn empty checkout has no physical index. Emit the equivalent empty
		// supported index, including the format-specific checksum, without source writes.
		raw := make([]byte, 12)
		copy(raw, "DIRC")
		binary.BigEndian.PutUint32(raw[4:8], uint32(snapshot.Index.Version))
		if snapshot.ObjectFormat == "sha256" {
			sum := sha256.Sum256(raw)
			raw = append(raw, sum[:]...)
		} else {
			sum := sha1.Sum(raw)
			raw = append(raw, sum[:]...)
		}
		if err := os.WriteFile(target, raw, 0600); err != nil {
			return nil, refuse(GateContentRead, "cannot prepare empty index")
		}
	} else {
		file, err := secprim.OpenNoFollowFile(before.path)
		if err != nil {
			return nil, refuse(GateContentRead, "raw index read failed")
		}
		raw, err := readAllContext(ctx, file)
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, refuse(GateContentRead, "raw index read failed")
		}
		if !validIndexChecksum(raw, snapshot.ObjectFormat) {
			return nil, refuse(GateTransferObjects, "raw index checksum mismatch")
		}
		if sha256.Sum256(raw) != before.digest {
			return nil, refuse(GateConsistency, "raw index changed after observation")
		}
		if err := os.WriteFile(target, raw, 0600); err != nil {
			return nil, refuse(GateContentRead, "cannot prepare raw index")
		}
		shared, err := assemblyRun(ctx, s.runner, snapshot.Worktree.RepoRoot, "rev-parse", "--path-format=absolute", "--shared-index-path")
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(shared)) != 0 {
			name := strings.TrimSuffix(string(shared), "\n")
			file, err := secprim.OpenNoFollowFile(name)
			if err != nil {
				return nil, refuse(GateContentRead, "shared index read failed")
			}
			data, err := readAllContext(ctx, file)
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				return nil, refuse(GateContentRead, "shared index read failed")
			}
			if !validIndexChecksum(data, snapshot.ObjectFormat) {
				return nil, refuse(GateTransferObjects, "shared index checksum mismatch")
			}
			if !strings.HasPrefix(filepath.Base(name), "sharedindex.") {
				return nil, refuse(GateTransferObjects, "invalid shared index path")
			}
			if err := os.WriteFile(filepath.Join(scratch, filepath.Base(name)), data, 0600); err != nil {
				return nil, refuse(GateContentRead, "cannot prepare shared index")
			}
			if _, err := isolatedRun(ctx, s.runner, scratch, nil, "update-index", "--no-split-index"); err != nil {
				return nil, err
			}
		}
	}
	logical, err := isolatedRun(ctx, s.runner, scratch, nil, "ls-files", "--stage", "--debug", "-z")
	if err != nil {
		return nil, err
	}
	entries, err := parseIndexEntries(logical, snapshot.ObjectFormat)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(entries, snapshot.Index.Entries) {
		return nil, refuse(GateTransferObjects, "raw and logical index disagree")
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return nil, refuse(GateContentRead, "prepared index read failed")
	}
	if len(raw) < 12 || string(raw[:4]) != "DIRC" || int(binary.BigEndian.Uint32(raw[4:8])) != snapshot.Index.Version {
		return nil, refuse(GateIndexVersion, "prepared index version disagrees")
	}
	return raw, nil
}

func packWire(objects repositoryObjects, format string) map[string]any {
	return map[string]any{"format": "git_pack_v2", "object_format": format, "blob_id": objects.pack.BlobID, "blob_descriptor_id": objects.pack.DescriptorID, "object_count": objects.count, "inventory_blob_id": objects.inventory.BlobID, "inventory_blob_descriptor_id": objects.inventory.DescriptorID}
}
func indexWire(index Index, descriptor BlobDescriptor) map[string]any {
	return map[string]any{"format": index.Format, "version": index.Version, "blob_id": descriptor.BlobID, "blob_descriptor_id": descriptor.DescriptorID, "entries": append([]IndexEntry{}, index.Entries...), "entry_count": index.EntryCount}
}

func validIndexChecksum(raw []byte, format string) bool {
	size := sha1.Size
	if format == "sha256" {
		size = sha256.Size
	}
	if len(raw) < 12+size {
		return false
	}
	body, checksum := raw[:len(raw)-size], raw[len(raw)-size:]
	if format == "sha256" {
		sum := sha256.Sum256(body)
		return bytes.Equal(sum[:], checksum)
	}
	sum := sha1.Sum(body)
	return bytes.Equal(sum[:], checksum)
}

// Check unsupported source modes before the first object-dependent observation,
// including nested repositories reached by the existing content scanner.
func checkAssemblySource(ctx context.Context, runner Runner, root string) error {
	// Git can otherwise substitute replacement history or lazily contact a remote.
	// These repository modes require a separate supported transfer representation.

	partial, err := readOptionalConfig(ctx, runner, root, "config", "--get", "extensions.partialClone")
	if err != nil {
		return err
	}
	if partial.ExitCode == 0 {
		return refuse(GateFeatures, "partial clone is not an offline capture source")
	}
	promisors, err := readOptionalConfig(ctx, runner, root, "config", "--type=bool", "--get-regexp", `^remote\..*\.promisor$`)
	if err != nil {
		return err
	}
	if promisors.ExitCode == 0 {
		lines := strings.Split(strings.TrimSuffix(string(promisors.Stdout), "\n"), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) != 2 || (fields[1] != "true" && fields[1] != "false") {
				return refuse(GateFeatures, "malformed promisor configuration")
			}
			if fields[1] == "true" {
				return refuse(GateFeatures, "partial clone is not an offline capture source")
			}
		}
	}
	replacements, err := assemblyRun(ctx, runner, root, "for-each-ref", "--format=%(refname)", "refs/replace/")
	if err != nil {
		return err
	}
	if len(replacements) != 0 {
		return refuse(GateFeatures, "replacement history is not representable")
	}

	return nil
}

type assemblyObservationRunner struct {
	Runner
	checked map[string]bool
}

func (r *assemblyObservationRunner) Run(ctx context.Context, root string, args ...string) (GitResult, error) {
	if !r.checked[root] {
		if err := checkAssemblySource(ctx, r.Runner, root); err != nil {
			return GitResult{ExitCode: -1}, err
		}
		r.checked[root] = true
	}
	return r.Runner.Run(ctx, root, args...)
}
func captureAssembly(ctx context.Context, runner AssemblyRunner, dir string, content ContentOptions) (*Snapshot, error) {
	return Capture(ctx, &assemblyObservationRunner{Runner: runner, checked: map[string]bool{}}, dir, content)
}
