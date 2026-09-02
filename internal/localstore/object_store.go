package localstore

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	MaxBlobSize      uint64 = 1<<53 - 1
	stagedFilePrefix        = ".ax-object-stage-"
)

var (
	ErrInvalidPathStyle  = errors.New("invalid AX digest path style")
	ErrInvalidBlobSize   = errors.New("invalid AX blob size")
	ErrIntegrityMismatch = errors.New("AX blob size or digest mismatch")
	ErrImmutableConflict = errors.New("AX immutable object conflict")
	ErrDurability        = errors.New("AX durable object installation failed")
)

type PathStyle string

const (
	PathStylePOSIX   PathStyle = "posix"
	PathStyleWindows PathStyle = "windows"
)

// DigestPathV1 is the one Section 3.2 digest-to-path projection shared by all
// durable object classes. The returned value is relative to a class root.
func DigestPathV1(digest scalar.Digest, style PathStyle) (string, error) {
	components, err := digestPathComponents(digest)
	if err != nil {
		return "", err
	}
	separator := ""
	switch style {
	case PathStylePOSIX:
		separator = "/"
	case PathStyleWindows:
		separator = `\`
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidPathStyle, style)
	}
	return strings.Join(components[:], separator), nil
}

// digestPathComponents is the single owner of the digest_path_v1 split. Every
// durable namespace in this package joins these exact components beneath its
// own root instead of re-deriving the shard and leaf independently.
func digestPathComponents(digest scalar.Digest) ([3]string, error) {
	hexDigest := digest.Hex()
	if len(hexDigest) != sha256.Size*2 {
		return [3]string{}, fmt.Errorf("%w: malformed digest", ErrInvalidPath)
	}
	return [3]string{"sha256", hexDigest[:2], hexDigest[2:]}, nil
}

func nativeDigestPath(root string, digest scalar.Digest) (string, error) {
	components, err := digestPathComponents(digest)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, components[0], components[1], components[2]), nil
}

type PutResult struct {
	Digest                 scalar.Digest
	Size                   uint64
	Path                   string
	Installed              bool
	QuarantinePath         string
	ExistingQuarantinePath string
}

type storeOperations struct {
	syncFile      func(*os.File) error
	install       func(string, string) error
	syncDirectory func(string) error
	// openExisting reads the entry already occupying a digest path. It is a
	// seam because the classification that depends on it — a read that did not
	// complete is a durability failure, never a proven mismatch — is otherwise
	// reachable only through descriptor exhaustion or real media failure.
	openExisting blobOpener
}

func defaultStoreOperations() storeOperations {
	return storeOperations{
		syncFile: func(file *os.File) error { return file.Sync() },
		install:  atomicRenameNoReplace,
		syncDirectory: func(directory string) error {
			return syncDirectory(directory)
		},
		openExisting: openBlobFile,
	}
}

type ObjectStore struct {
	dataRoot   string
	objects    string
	quarantine string
	operations storeOperations
	mu         sync.Mutex
}

// OpenObjectStore creates or verifies the owner-only local roots and opens the
// live blob namespace. It does not expose this root to a plugin or peer.
func OpenObjectStore(paths ResolvedPaths) (*ObjectStore, error) {
	if err := InitializeLayout(paths); err != nil {
		return nil, err
	}
	data, ok := paths.Path(PathData)
	if !ok {
		return nil, fmt.Errorf("%w: missing data root", ErrInvalidPath)
	}
	dataRoot := data.Value.String()
	if err := ensureOwnerChildTree(dataRoot, "objects", "sha256"); err != nil {
		return nil, fmt.Errorf("initialize object namespace: %w", err)
	}
	if err := ensureOwnerChildTree(dataRoot, "quarantine", "sha256"); err != nil {
		return nil, fmt.Errorf("initialize quarantine namespace: %w", err)
	}
	return &ObjectStore{
		dataRoot: dataRoot, objects: filepath.Join(dataRoot, "objects"),
		quarantine: filepath.Join(dataRoot, "quarantine"), operations: defaultStoreOperations(),
	}, nil
}

func (store *ObjectStore) DataRoot() string { return store.dataRoot }

// PutBlob is the production immutable-write entry point. It stages in the
// destination directory, fsyncs, verifies the declared uint53 size and SHA-256,
// atomically renames, then fsyncs the containing directory. An identical retry
// verifies and reuses the existing inode. Mismatches are installed create-new
// in quarantine and never enter the immutable namespace.
//
// Quarantine of an entry that already occupies the digest path is authorized
// only by a completed, disagreeing read, which is the SPEC.md Section 3.2
// trigger. An inspection whose read did not complete is a durability failure:
// PutBlob reports it and moves nothing, because a flaky read would otherwise
// permanently sideline a valid immutable object.
func (store *ObjectStore) PutBlob(expected scalar.Digest, expectedSize uint64, source io.Reader) (PutResult, error) {
	if store == nil || store.operations.syncFile == nil || store.operations.install == nil ||
		store.operations.syncDirectory == nil || store.operations.openExisting == nil {
		return PutResult{}, fmt.Errorf("%w: object store is not initialized", ErrDurability)
	}
	if expectedSize > MaxBlobSize {
		return PutResult{}, fmt.Errorf("%w: must be at most uint53", ErrInvalidBlobSize)
	}
	if source == nil {
		return PutResult{}, fmt.Errorf("%w: source is nil", ErrIntegrityMismatch)
	}
	if _, err := scalar.ParseDigest(expected.String()); err != nil {
		return PutResult{}, fmt.Errorf("%w: expected digest: %v", ErrIntegrityMismatch, err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	target, err := nativeDigestPath(store.objects, expected)
	if err != nil {
		return PutResult{}, fmt.Errorf("%w: object path: %v", ErrIntegrityMismatch, err)
	}
	shardDirectory := filepath.Dir(target)
	if err := ensureOwnerChildTree(store.objects, "sha256", expected.Hex()[:2]); err != nil {
		return PutResult{}, fmt.Errorf("%w: create object shard: %v", ErrDurability, err)
	}
	staged, err := os.CreateTemp(shardDirectory, stagedFilePrefix)
	if err != nil {
		return PutResult{}, fmt.Errorf("%w: create staged blob: %v", ErrDurability, err)
	}
	stagedPath := staged.Name()
	stagedClosed := false
	defer func() {
		if !stagedClosed {
			_ = staged.Close()
		}
		_ = os.Remove(stagedPath)
	}()
	if err := staged.Chmod(0o600); err != nil {
		return PutResult{}, fmt.Errorf("%w: set staged mode: %v", ErrDurability, err)
	}

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(staged, hasher), io.LimitReader(source, int64(expectedSize)+1))
	if copyErr != nil {
		return PutResult{}, fmt.Errorf("%w: stage bytes: %v", ErrDurability, copyErr)
	}
	if err := store.operations.syncFile(staged); err != nil {
		return PutResult{}, fmt.Errorf("%w: fsync staged blob: %v", ErrDurability, err)
	}
	info, err := staged.Stat()
	if err != nil {
		return PutResult{}, fmt.Errorf("%w: stat staged blob: %v", ErrDurability, err)
	}
	if err := verifyOwnerFileInfo(info, 0o600); err != nil {
		return PutResult{}, fmt.Errorf("%w: staged blob: %v", ErrDurability, err)
	}
	if err := staged.Close(); err != nil {
		return PutResult{}, fmt.Errorf("%w: close staged blob: %v", ErrDurability, err)
	}
	stagedClosed = true

	actualDigest, err := scalar.ParseDigest("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if err != nil {
		return PutResult{}, fmt.Errorf("%w: calculate staged digest: %v", ErrDurability, err)
	}
	actualSize := uint64(written)
	if info.Size() < 0 || uint64(info.Size()) != actualSize || actualSize != expectedSize || actualDigest != expected {
		quarantinePath, quarantineErr := store.quarantineStaged(expected, stagedPath)
		if quarantineErr != nil {
			return PutResult{}, fmt.Errorf("%w: mismatch and quarantine failed: %v", ErrDurability, quarantineErr)
		}
		return PutResult{Digest: actualDigest, Size: actualSize, QuarantinePath: quarantinePath},
			fmt.Errorf("%w: declared and staged identity differ", ErrIntegrityMismatch)
	}

	result := PutResult{Digest: expected, Size: expectedSize, Path: target}
	inspection := store.inspectExisting(target, expected, expectedSize)
	if inspection.verdict != blobAbsent {
		return store.resolveExistingEntry(inspection, expected, stagedPath, target, result, "")
	}

	if err := store.operations.install(stagedPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			raced := store.inspectExisting(target, expected, expectedSize)
			if raced.verdict != blobAbsent {
				return store.resolveExistingEntry(raced, expected, stagedPath, target, result, "raced ")
			}
		}
		return PutResult{}, fmt.Errorf("%w: atomic no-replace rename: %v", ErrDurability, err)
	}
	result.Installed = true
	if err := store.operations.syncDirectory(shardDirectory); err != nil {
		return result, fmt.Errorf("%w: fsync object directory after rename: %v", ErrDurability, err)
	}
	return result, nil
}

// resolveExistingEntry maps one inspection verdict onto the single consequence
// Section 3.2 allows for it. Both the first-attempt and the raced post-rename
// paths funnel through here, so the two cannot drift apart: a read that did not
// complete is a durability failure that moves nothing, and only a completed,
// disagreeing read reaches quarantineExisting.
func (store *ObjectStore) resolveExistingEntry(
	inspection blobInspection, expected scalar.Digest, stagedPath, target string,
	result PutResult, label string,
) (PutResult, error) {
	if inspection.verdict == blobMatches {
		return result, nil
	}
	if inspection.verdict == blobUnreadable {
		// The bytes on disk were never established, so nothing is proven and
		// nothing may move. The staged candidate is discarded by PutBlob's
		// deferred cleanup and the existing object stays exactly where it is.
		return PutResult{}, fmt.Errorf("%w: inspect %sexisting digest path: %v", ErrDurability, label, inspection.err)
	}
	candidateQuarantine, quarantineErr := store.quarantineStaged(expected, stagedPath)
	if quarantineErr != nil {
		return PutResult{}, fmt.Errorf("%w: %sexisting conflict and candidate quarantine failed: %v",
			ErrDurability, label, quarantineErr)
	}
	result.QuarantinePath = candidateQuarantine
	if !inspection.verdict.quarantineWarranted() {
		return result, fmt.Errorf("%w: unsafe %sdigest-path entry: %v", ErrImmutableConflict, label, inspection.err)
	}
	existingQuarantine, quarantineErr := store.quarantineExisting(expected, target)
	if quarantineErr != nil {
		return result, fmt.Errorf("%w: %v; %sexisting quarantine failed: %v",
			ErrImmutableConflict, inspection.err, label, quarantineErr)
	}
	result.ExistingQuarantinePath = existingQuarantine
	return result, fmt.Errorf("%w: %sexisting digest path failed verification: %v",
		ErrImmutableConflict, label, inspection.err)
}

// inspectExisting classifies whatever occupies the digest path. It never
// decides a consequence; it reports only what was proven. Content comparison is
// delegated to verifyBlobContent, the classifier the projection source scan
// also uses, so a failed read cannot become a mismatch on this path alone.
func (store *ObjectStore) inspectExisting(target string, expected scalar.Digest, expectedSize uint64) blobInspection {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return blobInspection{verdict: blobAbsent}
	}
	if err != nil {
		return blobInspection{verdict: blobUnreadable, err: fmt.Errorf("lstat digest path: %w", err)}
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return blobInspection{verdict: blobUnsafe, err: fmt.Errorf("%w: target is not a regular file", ErrUnsafeOwnership)}
	}
	if err := verifyOwnerFileInfo(info, 0o600); err != nil {
		if !errors.Is(err, ErrUnsafeOwnership) {
			return blobInspection{verdict: blobUnreadable, err: err}
		}
		return blobInspection{verdict: blobUnsafe, err: err}
	}
	return verifyBlobContent(store.operations.openExisting, target, expected, expectedSize)
}

func (store *ObjectStore) quarantineStaged(expected scalar.Digest, stagedPath string) (string, error) {
	directory, err := store.quarantineDigestDirectory(expected)
	if err != nil {
		return "", err
	}
	return store.moveToQuarantine(stagedPath, directory)
}

func (store *ObjectStore) quarantineExisting(expected scalar.Digest, target string) (string, error) {
	directory, err := store.quarantineDigestDirectory(expected)
	if err != nil {
		return "", err
	}
	destination, err := store.moveToQuarantine(target, directory)
	if err != nil {
		return destination, err
	}
	if err := store.operations.syncDirectory(filepath.Dir(target)); err != nil {
		return destination, err
	}
	return destination, nil
}

func (store *ObjectStore) moveToQuarantine(source, directory string) (string, error) {
	for attempts := 0; attempts < 8; attempts++ {
		destination, err := uniqueQuarantinePath(directory)
		if err != nil {
			return "", err
		}
		if err := store.operations.install(source, destination); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return "", err
		}
		if err := store.operations.syncDirectory(directory); err != nil {
			return destination, err
		}
		return destination, nil
	}
	return "", fmt.Errorf("allocate create-new quarantine destination")
}

func (store *ObjectStore) quarantineDigestDirectory(digest scalar.Digest) (string, error) {
	components, err := digestPathComponents(digest)
	if err != nil {
		return "", err
	}
	if err := ensureOwnerChildTree(store.quarantine, components[:]...); err != nil {
		return "", err
	}
	return nativeDigestPath(store.quarantine, digest)
}

func uniqueQuarantinePath(directory string) (string, error) {
	for attempts := 0; attempts < 8; attempts++ {
		identifier, err := newUUIDv7()
		if err != nil {
			return "", err
		}
		candidate := filepath.Join(directory, identifier)
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("allocate unique quarantine identifier")
}

func newUUIDv7() (string, error) {
	var value [16]byte
	milliseconds := uint64(time.Now().UnixMilli())
	value[0] = byte(milliseconds >> 40)
	value[1] = byte(milliseconds >> 32)
	value[2] = byte(milliseconds >> 24)
	value[3] = byte(milliseconds >> 16)
	value[4] = byte(milliseconds >> 8)
	value[5] = byte(milliseconds)
	if _, err := rand.Read(value[6:]); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x70
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func ensureOwnerChildTree(root string, elements ...string) error {
	if err := verifyOwnerDirectory(root); err != nil {
		return err
	}
	current := root
	for _, element := range elements {
		if element == "" || element == "." || element == ".." || strings.ContainsAny(element, `/\`) {
			return fmt.Errorf("%w: unsafe child component", ErrInvalidPath)
		}
		current = filepath.Join(current, element)
		if err := ensureOwnerDirectory(current); err != nil {
			return err
		}
	}
	return nil
}
