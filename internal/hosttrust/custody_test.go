package hosttrust

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func openForTest(t *testing.T, stateDir string) *Store {
	t.Helper()
	return openPathsForTest(t, filepath.Join(stateDir, "config.toml"), stateDir)
}

func resolvedPathsForTest(t *testing.T, configPath, stateDir string) localstore.ResolvedPaths {
	t.Helper()
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: scalar.PlatformLinux,
		Flags: map[string]string{
			"--config":      configPath,
			"--data-dir":    filepath.Join(stateDir, "data"),
			"--state-dir":   stateDir,
			"--cache-dir":   filepath.Join(stateDir, "cache"),
			"--runtime-dir": filepath.Join(stateDir, "runtime"),
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths(%s,%s) error = %v", configPath, stateDir, err)
	}
	return paths
}

func openPathsForTest(t *testing.T, configPath, stateDir string) *Store {
	t.Helper()
	store, err := Open(resolvedPathsForTest(t, configPath, stateDir))
	if err != nil {
		t.Fatalf("Open(%s) error = %v", stateDir, err)
	}
	return store
}

func storeStateDirForTest(store *Store) string {
	return filepath.Dir(store.Root())
}

func TestOpenCreatesOwnerOnlyLayout(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	root := filepath.Join(state, hostChannelDir)
	for _, dir := range []string{root, filepath.Join(root, credentialsDir)} {
		info, err := os.Lstat(dir)
		if err != nil {
			t.Fatalf("Lstat(%s) error = %v", dir, err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("directory %s mode = %04o, want 0700", dir, info.Mode().Perm())
		}
	}
	snapshot, err := store.Initialize()
	if err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	if snapshot.Generation != 1 || len(snapshot.Trust.Entries) != 0 {
		t.Fatalf("Initialize = gen %d entries %d, want 1/0", snapshot.Generation, len(snapshot.Trust.Entries))
	}
	info, err := os.Lstat(store.TrustPath())
	if err != nil {
		t.Fatalf("Lstat(trust.json) error = %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("trust.json mode = %04o, want 0600", info.Mode().Perm())
	}
	lockInfo, err := os.Lstat(filepath.Join(root, lockFileName))
	if err != nil {
		t.Fatalf("Lstat(lock) error = %v", err)
	}
	if lockInfo.Mode().Perm() != 0o600 {
		t.Fatalf("lock mode = %04o, want 0600", lockInfo.Mode().Perm())
	}
	// A second explicit setup refuses: it never overwrites committed authority.
	if _, err := store.Initialize(); err == nil {
		t.Fatal("second Initialize succeeded, want refusal")
	}
}

func TestOpenRefusesUnresolvedPathPair(t *testing.T) {
	if _, err := Open(localstore.ResolvedPaths{}); err == nil {
		t.Fatal("Open(zero paths) succeeded, want refusal")
	} else if !errors.Is(err, ErrInvalidStoreContext) {
		t.Fatalf("Open(zero paths) error = %v", err)
	}
}

func TestReadSnapshotMissingStore(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.ReadSnapshot(); err == nil {
		t.Fatal("ReadSnapshot(missing) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustStoreMissing) {
		t.Fatalf("ReadSnapshot(missing) error = %v, want ErrTrustStoreMissing", err)
	}
}

func TestCustodyRefusals(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-bit refusals do not bind the superuser")
	}
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(state, hostChannelDir)
	tests := []struct {
		name  string
		apply func() func()
	}{
		{"group readable directory", func() func() {
			if err := os.Chmod(root, 0o750); err != nil {
				t.Fatal(err)
			}
			return func() { _ = os.Chmod(root, 0o700) }
		}},
		{"world readable trust file", func() func() {
			if err := os.Chmod(store.TrustPath(), 0o644); err != nil {
				t.Fatal(err)
			}
			return func() { _ = os.Chmod(store.TrustPath(), 0o600) }
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restore := test.apply()
			defer restore()
			fresh, err := Open(resolvedPathsForTest(t, filepath.Join(state, "config.toml"), state))
			if err == nil {
				_ = fresh
				if _, err := fresh.ReadSnapshot(); err == nil {
					t.Fatalf("ReadSnapshot(%s) succeeded, want refusal", test.name)
				} else if !errors.Is(err, ErrTrustUnsafeCustody) {
					t.Fatalf("ReadSnapshot(%s) error = %v, want ErrTrustUnsafeCustody", test.name, err)
				}
			} else if !errors.Is(err, ErrTrustUnsafeCustody) {
				t.Fatalf("Open(%s) error = %v, want ErrTrustUnsafeCustody", test.name, err)
			}
		})
	}
}

func TestValidateCustodyBindsDirectory(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	credential, err := store.Issue(testHostA, testNow)
	if err != nil {
		t.Fatal(err)
	}
	leafHex := strings.TrimPrefix(credential, "sha256:")
	if err := store.ValidateCustody(leafHex, testHostA, testNow); err != nil {
		t.Fatalf("ValidateCustody(matching directory) error = %v", err)
	}
	// A directory named by the root digest holding the same valid files
	// selects no identity: the contained leaf must reproduce the directory.
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	entry, found := findEntry(snapshot.Trust, credential)
	if !found {
		t.Fatal("issued credential has no trust entry")
	}
	rootHex := entry.RootID.Hex()
	if rootHex == leafHex {
		t.Fatal("fixture digests collide")
	}
	credentialsRoot := filepath.Join(state, hostChannelDir, credentialsDir)
	if err := os.Rename(filepath.Join(credentialsRoot, leafHex), filepath.Join(credentialsRoot, rootHex)); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateCustody(rootHex, testHostA, testNow); err == nil {
		t.Fatal("ValidateCustody(root-named directory) succeeded, want refusal")
	} else if !errors.Is(err, ErrCredentialCustody) {
		t.Fatalf("ValidateCustody(root-named directory) error = %v, want ErrCredentialCustody", err)
	}
}

func TestTrustSymlinkRefused(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	shadow := filepath.Join(state, "shadow.json")
	if err := os.WriteFile(shadow, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(store.TrustPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(shadow, store.TrustPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadSnapshot(); err == nil {
		t.Fatal("ReadSnapshot(symlink) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustUnsafeCustody) {
		t.Fatalf("ReadSnapshot(symlink) error = %v, want ErrTrustUnsafeCustody", err)
	}
}

func TestCorruptTrustRefusedNeverEmpty(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.TrustPath(), []byte(`{"schema":"urn:ax:schema:host-trust-store","schema_version":"1.0.0","generation":2,"entries":[]}TRUNC`), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.ReadSnapshot()
	if err == nil {
		t.Fatalf("ReadSnapshot(corrupt) succeeded as gen %d, want refusal", snapshot.Generation)
	}
	if snapshot.Generation != 0 || len(snapshot.Trust.Entries) != 0 {
		t.Fatal("refused snapshot must be zero, never a partial store")
	}
}

func TestStagingFilesIgnored(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(state, hostChannelDir)
	if err := os.WriteFile(filepath.Join(root, stagePrefix+"crashed"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot with staged leftovers error = %v", err)
	}
	if snapshot.Generation != 1 {
		t.Fatalf("ReadSnapshot selected gen %d, want last committed 1", snapshot.Generation)
	}
}

func TestCommitFailureLeavesCommittedBytes(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.TrustPath())
	if err != nil {
		t.Fatal(err)
	}
	failing := &faultFileSystem{FileSystem: store.fs, failSync: true}
	shadow := &Store{root: store.root, fs: failing, lock: store.lock}
	err = shadow.transact(func(trust *TrustStore) error {
		trust.Entries = append(trust.Entries, entryForTest(t, issueForTest(t, testHostA, testNow), EntryActive, testNow, nil))
		return nil
	})
	if err == nil {
		t.Fatal("transact(failing sync) succeeded, want failure")
	} else if !errors.Is(err, ErrTrustDurability) {
		t.Fatalf("transact(failing sync) error = %v, want ErrTrustDurability", err)
	}
	after, err := os.ReadFile(store.TrustPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("failed commit changed the committed bytes")
	}
	entries, err := os.ReadDir(filepath.Join(state, hostChannelDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stagePrefix) {
			t.Fatalf("failed commit left staging file %s", entry.Name())
		}
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot after failed commit error = %v", err)
	}
	if snapshot.Generation != 1 || len(snapshot.Trust.Entries) != 0 {
		t.Fatal("failed commit changed the generation or entries")
	}
}

func TestGenerationExhaustionRefusesMutation(t *testing.T) {
	state := t.TempDir()
	store := openForTest(t, state)
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	ceiling, err := EncodeTrust(TrustStore{Generation: MaxGeneration})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.TrustPath(), ceiling, 0o600); err != nil {
		t.Fatal(err)
	}
	err = store.transact(func(trust *TrustStore) error { return nil })
	if err == nil {
		t.Fatal("transact at uint53 ceiling succeeded, want refusal")
	} else if !errors.Is(err, ErrGenerationExhausted) {
		t.Fatalf("transact at ceiling error = %v, want ErrGenerationExhausted", err)
	}
}

// faultFileSystem fails the staging-file fsync to prove a failed commit is a
// failure that leaves committed bytes, generation and directory clean.
type faultFileSystem struct {
	FileSystem
	failSync bool
}

func (filesystem *faultFileSystem) CreateTemp(dir, pattern string, mode fs.FileMode) (StagedFile, error) {
	staged, err := filesystem.FileSystem.CreateTemp(dir, pattern, mode)
	if err != nil {
		return nil, err
	}
	if filesystem.failSync {
		return &faultStagedFile{StagedFile: staged}, nil
	}
	return staged, nil
}

type faultStagedFile struct {
	StagedFile
}

func (file *faultStagedFile) Sync() error { return errors.New("injected trust sync failure") }
