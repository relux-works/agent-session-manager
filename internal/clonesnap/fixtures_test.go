package clonesnap

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Shared fixtures for the clonesnap suite. Every literal is
// hand-written and independent of production constants; digests are
// recomputed with crypto/sha256, never copied from production. The
// verdict always comes from production; the vectors never do.

const (
	fixtureOperationID = "0198f4d1-9e60-7f11-8a31-abcdef012345"
	fixtureBundleID    = "0198f4d1-9e60-7f22-8b42-bcdef0123456"
	fixtureSessionID   = "0198f4d1-9e60-7f33-8c53-cdef01234567"
	fixtureHostID      = "0198f4d1-9e60-7f44-8d64-def012345678"
	fixtureWorkspaceID = "0198f4d1-9e60-7f55-8e75-ef0123456789"
	fixtureCreatedAt   = "2026-09-18T05:10:00.000Z"
	fixtureUUIDKey     = "0198f4d1-9e60-7f66-8f86-f0123456789a"
)

func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fixtureTupleJSON() string {
	return `{"environment_id":"snap.env","environment_version":"3.0.1","platform":"linux","architecture":"arm64","store_schema_fingerprint":` +
		`"` + fixtureDigest("snap-store-fingerprint") + `","adapter_version":"2.0.0"}`
}

func fixtureNativeIdentity(t *testing.T) clonebundle.NativeIdentity {
	t.Helper()
	workspace, err := scalar.ParseUUIDv7(fixtureWorkspaceID)
	if err != nil {
		t.Fatalf("ParseUUIDv7() error = %v", err)
	}
	return clonebundle.NativeIdentity{
		NativeSessionID:    "native-session-snap",
		IdentityKind:       "provider_native",
		LogicalWorkspaceID: workspace,
	}
}

func fixturePlatform() scalar.Platform {
	if runtime.GOOS == "darwin" {
		return scalar.PlatformMacOS
	}
	if runtime.GOOS == "windows" {
		return scalar.PlatformWindows
	}
	return scalar.PlatformLinux
}

func openTestStore(t *testing.T) *localstore.ObjectStore {
	t.Helper()
	home := t.TempDir()
	config := filepath.Join(home, "config")
	if err := os.Mkdir(config, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := openStoreAt(home)
	if err != nil {
		t.Fatalf("openStoreAt() error = %v", err)
	}
	return store
}

// openStoreAt opens the object store rooted at an existing home
// directory. The crash child uses it without a testing context.
func openStoreAt(home string) (*localstore.ObjectStore, error) {
	resolved, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: fixturePlatform(),
		Flags: map[string]string{
			"--config":      filepath.Join(home, "config", "config.toml"),
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
		return nil, err
	}
	return localstore.OpenObjectStore(resolved)
}

// writeStoreFile creates one regular store member with the given
// bytes, creating parent directories as needed.
func writeStoreFile(t *testing.T, root, key string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// fixtureStore builds the standard provider store: two included
// members, one credential member, and a nested directory.
func fixtureStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytes"))
	writeStoreFile(t, root, "store/blob-b", []byte("beta-payload-bytes"))
	writeStoreFile(t, root, "store/token-cache", []byte("credential-secret-bytes"))
	return root
}

func fixturePlan() []PlanItem {
	return []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
	}
}

// fixtureWorkspace builds a real workspace tree and returns its
// checkpoint request: the root, an existing nested cwd, one remote
// fingerprint, a branch, and a head digest.
func fixtureWorkspace(t *testing.T) WorkspaceRequest {
	t.Helper()
	root := t.TempDir()
	cwd := filepath.Join(root, "work", "trees", "alpha")
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	branch := "main"
	head := fixtureDigest("snap-head")
	return WorkspaceRequest{
		LogicalWorkspaceID: fixtureWorkspaceID,
		WorkspaceRoot:      root,
		Cwd:                cwd,
		RemoteFingerprints: []string{fixtureDigest("snap-remote")},
		Branch:             &branch,
		HeadDigest:         &head,
		Platform:           fixturePlatform(),
	}
}

func fixtureSourceBasis() clonebundle.SourceBasisInput {
	return clonebundle.SourceBasisInput{
		Kind:                     "ax_session",
		SourceSessionID:          fixtureSessionID,
		SourceSessionRecordID:    fixtureDigest("snap-session-record"),
		SourceCheckpointID:       fixtureDigest("snap-checkpoint"),
		SourceProviderIdentityID: fixtureDigest("snap-provider-identity"),
	}
}

func validCaptureRequest(t *testing.T, storeRoot string, store *localstore.ObjectStore) CaptureRequest {
	t.Helper()
	return CaptureRequest{
		StoreRoot:         storeRoot,
		Platform:          fixturePlatform(),
		Plan:              fixturePlan(),
		OperationID:       fixtureOperationID,
		BundleID:          fixtureBundleID,
		SourceBasis:       fixtureSourceBasis(),
		SourceEnvironment: []byte(fixtureTupleJSON()),
		SourceIdentity:    fixtureNativeIdentity(t),
		CapturePlanDigest: fixtureDigest("snap-capture-plan"),
		NativeSessionID:   "native-session-snap",
		CreatedByHostID:   fixtureHostID,
		CreatedAt:         fixtureCreatedAt,
		Generation:        "generation-7",
		ProofKind:         "closed_store",
		InputBlocked:      true,
		ForegroundIdle:    true,
		BackgroundIdle:    true,
		OperatorExplicit:  false,
		OnRace:            RaceRefuse,
		Workspace:         fixtureWorkspace(t),
		Store:             store,
	}
}

// storedBlob is one content-addressed file observed under a store's
// objects namespace.
type storedBlob struct {
	digest scalar.Digest
	bytes  []byte
}

// listStoredBlobs enumerates every blob file under the store's
// objects namespace. It is the test-side oracle for what capture
// published: production never lists the namespace.
func listStoredBlobs(t *testing.T, store *localstore.ObjectStore) []storedBlob {
	t.Helper()
	objects := filepath.Join(store.DataRoot(), "objects")
	var blobs []storedBlob
	err := filepath.Walk(objects, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		sum := sha256.Sum256(raw)
		digest, err := scalar.ParseDigest("sha256:" + hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatalf("ParseDigest() error = %v", err)
		}
		blobs = append(blobs, storedBlob{digest: digest, bytes: raw})
		return nil
	})
	if err != nil {
		t.Fatalf("Walk(objects) error = %v", err)
	}
	return blobs
}

// blobsContain reports whether any stored blob carries the needle.
func blobsContain(blobs []storedBlob, needle string) bool {
	for _, blob := range blobs {
		if strings.Contains(string(blob.bytes), needle) {
			return true
		}
	}
	return false
}

func mustCapture(t *testing.T, request CaptureRequest) *CaptureResult {
	t.Helper()
	result, err := Capture(request)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if result == nil {
		t.Fatal("Capture() returned nil result without an error")
	}
	return result
}

// assertEveryRawBlobInstalled proves the Section 10.2 install rule
// at the store: every raw-manifest entry's blob is present under
// its content digest. A manifest that references a blob Capture
// never installed fails here.
func assertEveryRawBlobInstalled(t *testing.T, store *localstore.ObjectStore, raw clonebundle.RawObjectManifest) {
	t.Helper()
	if len(raw.Entries) == 0 {
		t.Fatal("raw manifest carries no entries")
	}
	installed := map[string]bool{}
	for _, blob := range listStoredBlobs(t, store) {
		installed[blob.digest.String()] = true
	}
	for _, entry := range raw.Entries {
		if !installed[entry.BlobID.String()] {
			t.Fatalf("raw entry %q references blob %q with no installed blob", entry.NativeItemKey, entry.BlobID)
		}
	}
}
