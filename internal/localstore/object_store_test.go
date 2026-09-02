package localstore

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	emptyDigestText   = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	exampleDigestText = "sha256:9c21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50"
)

func TestDigestPathV1MatchesNormativePOSIXAndWindowsVectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		digest string
		style  PathStyle
		want   string
	}{
		{"empty POSIX", emptyDigestText, PathStylePOSIX, "sha256/e3/b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"example POSIX", exampleDigestText, PathStylePOSIX, "sha256/9c/21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50"},
		{"empty Windows", emptyDigestText, PathStyleWindows, `sha256\e3\b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`},
		{"example Windows", exampleDigestText, PathStyleWindows, `sha256\9c\21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			digest, err := scalar.ParseDigest(test.digest)
			if err != nil {
				t.Fatalf("ParseDigest() error = %v", err)
			}
			got, err := DigestPathV1(digest, test.style)
			if err != nil {
				t.Fatalf("DigestPathV1() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("DigestPathV1() = %q, want %q", got, test.want)
			}
			parts := strings.FieldsFunc(got, func(r rune) bool { return r == '/' || r == '\\' })
			if len(parts) != 3 || len(parts[1]) != 2 || len(parts[2]) != 62 || strings.Contains(got, "sha256:") {
				t.Fatalf("DigestPathV1() = %q, want sha256/two-hex/62-hex without colon", got)
			}
		})
	}

	digest, err := scalar.ParseDigest(emptyDigestText)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DigestPathV1(digest, PathStyle("unknown")); !errors.Is(err, ErrInvalidPathStyle) {
		t.Fatalf("DigestPathV1(unknown style) error = %v, want %v", err, ErrInvalidPathStyle)
	}
	if _, err := DigestPathV1(scalar.Digest{}, PathStylePOSIX); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("DigestPathV1(zero digest) error = %v, want %v", err, ErrInvalidPath)
	}
}

func TestPutBlobInstallsVerifiedBytesAndIdenticalRetryDoesNotRewrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	content := []byte("hello world")
	digest := scalar.SHA256Digest(content)
	result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("PutBlob() error = %v", err)
	}
	if !result.Installed || result.Digest != digest || result.Size != uint64(len(content)) {
		t.Fatalf("PutBlob() result = %+v, want installed verified blob", result)
	}
	wantRelative, err := DigestPathV1(digest, nativePathStyle())
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(store.DataRoot(), "objects", filepath.FromSlash(strings.ReplaceAll(wantRelative, `\`, "/")))
	if result.Path != wantPath {
		t.Fatalf("PutBlob() path = %q, want %q", result.Path, wantPath)
	}
	got, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("ReadFile(installed blob) error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("installed bytes = %q, want %q", got, content)
	}
	before, err := os.Lstat(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode().Perm() != 0o600 {
		t.Fatalf("installed mode = %s, want 0600", before.Mode())
	}

	retry, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("PutBlob(retry) error = %v", err)
	}
	if retry.Installed || retry.Path != result.Path {
		t.Fatalf("PutBlob(retry) result = %+v, want existing immutable object", retry)
	}
	after, err := os.Lstat(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("identical PutBlob retry replaced the immutable file")
	}
	assertNoStagedFiles(t, filepath.Dir(result.Path))
}

func TestPutBlobRefusesAndQuarantinesDigestOrSizeMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	tests := []struct {
		name             string
		content          []byte
		expectedSize     uint64
		expected         scalar.Digest
		quarantinePrefix []byte
	}{
		{"digest mismatch", []byte("changed"), uint64(len("changed")), scalar.SHA256Digest([]byte("expected")), []byte("changed")},
		{"source shorter than declared", []byte("value"), uint64(len("value") + 1), scalar.SHA256Digest([]byte("value")), []byte("value")},
		{"source longer than declared", []byte("value!trailing"), uint64(len("value")), scalar.SHA256Digest([]byte("value")), []byte("value!")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			result, err := store.PutBlob(test.expected, test.expectedSize, bytes.NewReader(test.content))
			if !errors.Is(err, ErrIntegrityMismatch) {
				t.Fatalf("PutBlob() error = %v, want %v", err, ErrIntegrityMismatch)
			}
			if result.Installed || result.QuarantinePath == "" {
				t.Fatalf("PutBlob() result = %+v, want quarantined refusal", result)
			}
			if _, parseErr := scalar.ParseUUIDv7(filepath.Base(result.QuarantinePath)); parseErr != nil {
				t.Fatalf("quarantine leaf is not UUIDv7: %v", parseErr)
			}
			quarantined, readErr := os.ReadFile(result.QuarantinePath)
			if readErr != nil {
				t.Fatalf("ReadFile(quarantine) error = %v", readErr)
			}
			if !bytes.Equal(quarantined, test.quarantinePrefix) {
				t.Fatalf("quarantine bytes = %q, want bounded diagnostic prefix %q", quarantined, test.quarantinePrefix)
			}
			if result.Size != uint64(len(test.quarantinePrefix)) || result.Digest != scalar.SHA256Digest(test.quarantinePrefix) {
				t.Fatalf("PutBlob() quarantined identity = %s/%d, want prefix identity", result.Digest, result.Size)
			}
			if info, statErr := os.Lstat(result.QuarantinePath); statErr != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("quarantine mode/error = %v/%v, want 0600", info, statErr)
			}
			wantDigestPath, pathErr := nativeDigestPath(store.quarantine, test.expected)
			if pathErr != nil {
				t.Fatal(pathErr)
			}
			if filepath.Dir(result.QuarantinePath) != wantDigestPath {
				t.Fatalf("quarantine digest directory = %q, want shared digest_path_v1 %q", filepath.Dir(result.QuarantinePath), wantDigestPath)
			}
			target, pathErr := nativeDigestPath(store.objects, test.expected)
			if pathErr != nil {
				t.Fatal(pathErr)
			}
			if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("mismatched target stat error = %v, want absent", statErr)
			}
		})
	}
}

func TestPutBlobEnforcesUint53SizeInBothDirectionsAtProductionEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	digest := scalar.SHA256Digest(nil)
	result, err := store.PutBlob(digest, MaxBlobSize, bytes.NewReader(nil))
	if !errors.Is(err, ErrIntegrityMismatch) || errors.Is(err, ErrInvalidBlobSize) {
		t.Fatalf("PutBlob(size at uint53 limit) error = %v, want admitted then size mismatch", err)
	}
	if result.QuarantinePath == "" {
		t.Fatal("PutBlob(size at limit) did not drive the production staged-write path")
	}

	result, err = store.PutBlob(digest, MaxBlobSize+1, bytes.NewReader(nil))
	if !errors.Is(err, ErrInvalidBlobSize) {
		t.Fatalf("PutBlob(size past uint53 limit) error = %v, want %v", err, ErrInvalidBlobSize)
	}
	if result != (PutResult{}) {
		t.Fatalf("PutBlob(size past limit) result = %+v, want no mutation", result)
	}
}

func TestPutBlobQuarantinesBothInputsWhenExistingDigestPathIsCorrupt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("valid")
	tests := []struct {
		name     string
		existing []byte
	}{
		{name: "different size and digest", existing: []byte("corrupt")},
		{name: "same size but different digest", existing: []byte("fraud")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			digest := scalar.SHA256Digest(content)
			target, err := nativeDigestPath(store.objects, digest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, test.existing, 0o600); err != nil {
				t.Fatal(err)
			}

			result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if !errors.Is(err, ErrImmutableConflict) {
				t.Fatalf("PutBlob(corrupt existing) error = %v, want %v", err, ErrImmutableConflict)
			}
			if result.QuarantinePath == "" || result.ExistingQuarantinePath == "" {
				t.Fatalf("PutBlob(corrupt existing) result = %+v, want both quarantine paths", result)
			}
			if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("corrupt target stat error = %v, want removed from immutable namespace", statErr)
			}
			assertFileBytes(t, result.QuarantinePath, content)
			assertFileBytes(t, result.ExistingQuarantinePath, test.existing)
		})
	}
}

func TestPutBlobRefusesCorrectBytesAtUnsafeExistingMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	for _, mode := range []os.FileMode{0o666, 0o640} {
		mode := mode
		t.Run(mode.String(), func(t *testing.T) {
			store := openTestStore(t)
			content := []byte("immutable")
			digest := scalar.SHA256Digest(content)
			target, err := nativeDigestPath(store.objects, digest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, content, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(target, mode); err != nil {
				t.Fatal(err)
			}

			result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if !errors.Is(err, ErrImmutableConflict) || result.QuarantinePath == "" {
				t.Fatalf("PutBlob(unsafe existing mode %04o) result/error = %+v/%v, want quarantined candidate refusal", mode, result, err)
			}
			if result.ExistingQuarantinePath != "" {
				t.Fatalf("unsafe existing object moved into quarantine at %q", result.ExistingQuarantinePath)
			}
			info, statErr := os.Lstat(target)
			if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != mode {
				t.Fatalf("unsafe existing object changed: info/error = %v/%v", info, statErr)
			}
			assertFileBytes(t, target, content)
		})
	}
}

func TestPutBlobCrashBoundariesLeaveRecoverableState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	content := []byte("durable")
	digest := scalar.SHA256Digest(content)
	tests := []struct {
		name       string
		mutateOps  func(*storeOperations)
		wantTarget bool
	}{
		{
			name: "file fsync fails before rename",
			mutateOps: func(operations *storeOperations) {
				operations.syncFile = func(*os.File) error { return errors.New("injected file sync failure") }
			},
		},
		{
			name: "rename fails after verified fsync",
			mutateOps: func(operations *storeOperations) {
				operations.install = func(string, string) error { return errors.New("injected rename failure") }
			},
		},
		{
			name: "directory fsync fails after rename",
			mutateOps: func(operations *storeOperations) {
				operations.syncDirectory = func(string) error { return errors.New("injected directory sync failure") }
			},
			wantTarget: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			operations := defaultStoreOperations()
			test.mutateOps(&operations)
			store.operations = operations

			_, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if !errors.Is(err, ErrDurability) {
				t.Fatalf("PutBlob(injected failure) error = %v, want %v", err, ErrDurability)
			}
			target, pathErr := nativeDigestPath(store.objects, digest)
			if pathErr != nil {
				t.Fatal(pathErr)
			}
			_, statErr := os.Lstat(target)
			if test.wantTarget && statErr != nil {
				t.Fatalf("target after post-rename sync failure stat error = %v", statErr)
			}
			if !test.wantTarget && !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("target before atomic rename stat error = %v, want absent", statErr)
			}
			assertNoStagedFiles(t, filepath.Dir(target))

			store.operations = defaultStoreOperations()
			retry, retryErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			if retryErr != nil {
				t.Fatalf("PutBlob(recovery retry) error = %v", retryErr)
			}
			if retry.Path != target {
				t.Fatalf("PutBlob(recovery retry) path = %q, want %q", retry.Path, target)
			}
			assertFileBytes(t, target, content)
		})
	}
}

func TestConcurrentObjectStoreWritersCannotReplacePublishedInode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	first := openTestStore(t)
	resolved := first.resolvedPathsForTest(t)
	second, err := OpenObjectStore(resolved)
	if err != nil {
		t.Fatalf("OpenObjectStore(second process view) error = %v", err)
	}

	// Prove the defaults wired by OpenObjectStore, rather than a test replacement.
	// An occupied destination must survive the exact install operation PutBlob uses.
	operationDirectory := t.TempDir()
	candidate := filepath.Join(operationDirectory, "candidate")
	published := filepath.Join(operationDirectory, "published")
	if err := os.WriteFile(candidate, []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(published, []byte("published"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := first.operations.install(candidate, published); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("production install over occupied target error = %v, want %v", err, fs.ErrExist)
	}
	assertFileBytes(t, candidate, []byte("candidate"))
	assertFileBytes(t, published, []byte("published"))

	closed, err := os.CreateTemp(operationDirectory, "closed-sync-")
	if err != nil {
		t.Fatal(err)
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.operations.syncFile(closed); err == nil {
		t.Fatal("production file fsync on a closed descriptor succeeded; default is a no-op")
	}
	if err := first.operations.syncDirectory(filepath.Join(operationDirectory, "missing")); err == nil {
		t.Fatal("production directory fsync on a missing directory succeeded; default is a no-op")
	}

	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	defaultSyncFile := first.operations.syncFile
	barrierSyncFile := func(file *os.File) error {
		if err := defaultSyncFile(file); err != nil {
			return err
		}
		ready <- struct{}{}
		<-release
		return nil
	}
	first.operations.syncFile = barrierSyncFile
	second.operations.syncFile = barrierSyncFile

	content := []byte("one immutable inode")
	digest := scalar.SHA256Digest(content)
	results := make(chan PutResult, 2)
	errorsSeen := make(chan error, 2)
	var writers sync.WaitGroup
	writers.Add(2)
	for _, store := range []*ObjectStore{first, second} {
		go func(store *ObjectStore) {
			defer writers.Done()
			result, putErr := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
			results <- result
			errorsSeen <- putErr
		}(store)
	}
	for writer := 0; writer < 2; writer++ {
		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			close(release)
			writers.Wait()
			t.Fatal("concurrent writer did not reach the production fsync boundary")
		}
	}
	close(release)
	writers.Wait()
	close(results)
	close(errorsSeen)

	installed := 0
	for putErr := range errorsSeen {
		if putErr != nil {
			t.Errorf("concurrent PutBlob() error = %v", putErr)
		}
	}
	for result := range results {
		if result.Installed {
			installed++
		}
	}
	if installed != 1 {
		t.Fatalf("concurrent PutBlob installed count = %d, want exactly 1", installed)
	}
	target, err := nativeDigestPath(first.objects, digest)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := second.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if err != nil || retry.Installed {
		t.Fatalf("PutBlob(post-race retry) result/error = %+v/%v, want existing", retry, err)
	}
	after, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("published inode changed after concurrent installation")
	}
}

func TestPutBlobRefusesUnsafeDigestPathWithoutFollowingOrQuarantiningSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	content := []byte("candidate")
	digest := scalar.SHA256Digest(content)
	target, err := nativeDigestPath(store.objects, digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(external, []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, target); err != nil {
		t.Fatal(err)
	}

	result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if !errors.Is(err, ErrImmutableConflict) || result.QuarantinePath == "" {
		t.Fatalf("PutBlob(symlink target) result/error = %+v/%v, want quarantined candidate refusal", result, err)
	}
	if result.ExistingQuarantinePath != "" {
		t.Fatalf("unsafe symlink moved into quarantine at %q", result.ExistingQuarantinePath)
	}
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("unsafe target was followed or replaced: info/error = %v/%v", info, err)
	}
	assertFileBytes(t, external, []byte("external"))
}

func TestPutBlobRefusesOwnerModeNonRegularDigestEntryBeforeReadingOrMovingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	content := []byte("candidate")
	digest := scalar.SHA256Digest(content)
	target, err := nativeDigestPath(store.objects, digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if !errors.Is(err, ErrImmutableConflict) || result.QuarantinePath == "" {
		t.Fatalf("PutBlob(owner-mode directory target) result/error = %+v/%v, want quarantined candidate refusal", result, err)
	}
	if result.ExistingQuarantinePath != "" {
		t.Fatalf("non-regular digest entry moved into quarantine at %q", result.ExistingQuarantinePath)
	}
	info, statErr := os.Lstat(target)
	if statErr != nil || !info.IsDir() {
		t.Fatalf("non-regular digest entry was read or moved: info/error = %v/%v", info, statErr)
	}
}

func TestObjectStoreEntryPointsRefuseUninitializedAndMalformedInputs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	if _, err := OpenObjectStore(ResolvedPaths{}); !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("OpenObjectStore(zero paths) error = %v, want %v", err, ErrUnsupportedPlatform)
	}
	content := []byte("value")
	digest := scalar.SHA256Digest(content)
	var nilStore *ObjectStore
	if _, err := nilStore.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); !errors.Is(err, ErrDurability) {
		t.Fatalf("nil PutBlob() error = %v, want %v", err, ErrDurability)
	}
	store := openTestStore(t)
	if _, err := store.PutBlob(digest, uint64(len(content)), nil); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("PutBlob(nil source) error = %v, want %v", err, ErrIntegrityMismatch)
	}
	if _, err := store.PutBlob(scalar.Digest{}, 0, bytes.NewReader(nil)); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("PutBlob(zero digest) error = %v, want %v", err, ErrIntegrityMismatch)
	}
}

func openTestStore(t *testing.T) *ObjectStore {
	t.Helper()
	home := t.TempDir()
	config := filepath.Join(home, "config")
	if err := os.Mkdir(config, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolvePaths(ResolveRequest{
		Platform: nativeTestPlatform(),
		Flags: map[string]string{
			"--config":      filepath.Join(config, "config.toml"),
			"--data-dir":    filepath.Join(home, "data"),
			"--state-dir":   filepath.Join(home, "state"),
			"--cache-dir":   filepath.Join(home, "cache"),
			"--runtime-dir": filepath.Join(home, "runtime"),
		},
		Environment:  map[string]string{},
		HomeDir:      home,
		TemporaryDir: filepath.Join(home, "temporary"),
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	store, err := OpenObjectStore(resolved)
	if err != nil {
		t.Fatalf("OpenObjectStore() error = %v", err)
	}
	return store
}

func (store *ObjectStore) resolvedPathsForTest(t *testing.T) ResolvedPaths {
	t.Helper()
	home := filepath.Dir(store.dataRoot)
	resolved, err := ResolvePaths(ResolveRequest{
		Platform: nativeTestPlatform(),
		Flags: map[string]string{
			"--config":      filepath.Join(home, "config", "config.toml"),
			"--data-dir":    store.dataRoot,
			"--state-dir":   filepath.Join(home, "state"),
			"--cache-dir":   filepath.Join(home, "cache"),
			"--runtime-dir": filepath.Join(home, "runtime"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func nativePathStyle() PathStyle {
	if runtime.GOOS == "windows" {
		return PathStyleWindows
	}
	return PathStylePOSIX
}

func assertNoStagedFiles(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stagedFilePrefix) {
			t.Errorf("staged file remains after PutBlob: %s", entry.Name())
		}
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file %q bytes = %q, want %q", path, got, want)
	}
}
