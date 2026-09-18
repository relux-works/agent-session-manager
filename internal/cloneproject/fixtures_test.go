package cloneproject

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonesnap"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Shared fixtures for the cloneproject suite. Every projection proof
// runs through the real path: a fixture provider store, the
// predecessor's clonesnap.Capture entry, then Normalize over the
// sealed manifests with blob bytes fetched from the real capture
// store. Every literal is hand-written and independent of production
// constants; digests are recomputed with crypto/sha256, never copied
// from production.

const (
	fixtureOperationID = "0198f4d1-9e60-7a11-8a31-abcdef012345"
	fixtureBundleID    = "0198f4d1-9e60-7a22-8b42-bcdef0123456"
	fixtureSessionID   = "0198f4d1-9e60-7a33-8c53-cdef01234567"
	fixtureHostID      = "0198f4d1-9e60-7a44-8d64-def012345678"
	fixtureWorkspaceID = "0198f4d1-9e60-7a55-8e75-ef0123456789"
	fixtureCreatedAt   = "2026-09-18T11:05:00.000Z"
	fixtureLogicalID   = "0198f4d1-9e60-7a66-8f86-f0123456789a"
	fixtureMainActor   = "0198f4d1-9e60-7a77-8097-0123456789ab"
	fixtureSubActor    = "0198f4d1-9e60-7a88-81a8-123456789abc"
	fixtureExtActor    = "0198f4d1-9e60-7a99-82b9-23456789abcd"
)

func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fixtureTupleJSON() string {
	return `{"environment_id":"project.env","environment_version":"3.0.1","platform":"linux","architecture":"arm64","store_schema_fingerprint":` +
		`"` + fixtureDigest("project-store-fingerprint") + `","adapter_version":"2.0.0"}`
}

func fixtureNativeIdentity(t *testing.T) clonebundle.NativeIdentity {
	t.Helper()
	workspace, err := scalar.ParseUUIDv7(fixtureWorkspaceID)
	if err != nil {
		t.Fatalf("ParseUUIDv7() error = %v", err)
	}
	return clonebundle.NativeIdentity{
		NativeSessionID:    "native-session-project",
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
		t.Fatal(err)
	}
	store, err := localstore.OpenObjectStore(resolved)
	if err != nil {
		t.Fatalf("OpenObjectStore() error = %v", err)
	}
	return store
}

// errFixtureGone is the test fetch failure for the missing-blob row.
var errFixtureGone = errGone("fixture blob removed")

type errGone string

func (err errGone) Error() string { return string(err) }

// writeStoreFile rewrites one store member (race injection).
func writeStoreFile(root, key string, data []byte) error {
	path := filepath.Join(root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// fixtureMember is one provider store member: key, class, and bytes.
type fixtureMember struct {
	key     string
	class   string
	content []byte
}

// writeFixtureStore builds a provider store holding exactly the
// given members.
func writeFixtureStore(t *testing.T, members []fixtureMember) string {
	t.Helper()
	root := t.TempDir()
	for _, member := range members {
		path := filepath.Join(root, filepath.FromSlash(member.key))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(member.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// fixtureWorkspace builds a real workspace tree and its checkpoint
// request.
func fixtureWorkspace(t *testing.T) clonesnap.WorkspaceRequest {
	t.Helper()
	root := t.TempDir()
	cwd := filepath.Join(root, "work", "trees", "alpha")
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	branch := "main"
	head := fixtureDigest("project-head")
	return clonesnap.WorkspaceRequest{
		LogicalWorkspaceID: fixtureWorkspaceID,
		WorkspaceRoot:      root,
		Cwd:                cwd,
		RemoteFingerprints: []string{fixtureDigest("project-remote")},
		Branch:             &branch,
		HeadDigest:         &head,
		Platform:           fixturePlatform(),
	}
}

// validCaptureRequest builds a capture request over the fixture
// store through the real predecessor entry.
func validCaptureRequest(t *testing.T, storeRoot string, store *localstore.ObjectStore, members []fixtureMember) clonesnap.CaptureRequest {
	t.Helper()
	plan := make([]clonesnap.PlanItem, 0, len(members))
	for _, member := range members {
		plan = append(plan, clonesnap.PlanItem{NativeKey: member.key, Class: member.class, Required: true})
	}
	return clonesnap.CaptureRequest{
		StoreRoot:   storeRoot,
		Platform:    fixturePlatform(),
		Plan:        plan,
		OperationID: fixtureOperationID,
		BundleID:    fixtureBundleID,
		SourceBasis: clonebundle.SourceBasisInput{
			Kind:                     "ax_session",
			SourceSessionID:          fixtureSessionID,
			SourceSessionRecordID:    fixtureDigest("project-session-record"),
			SourceCheckpointID:       fixtureDigest("project-checkpoint"),
			SourceProviderIdentityID: fixtureDigest("project-provider-identity"),
		},
		SourceEnvironment: []byte(fixtureTupleJSON()),
		SourceIdentity:    fixtureNativeIdentity(t),
		CapturePlanDigest: fixtureDigest("project-capture-plan"),
		NativeSessionID:   "native-session-project",
		CreatedByHostID:   fixtureHostID,
		CreatedAt:         fixtureCreatedAt,
		Generation:        "generation-11",
		ProofKind:         "closed_store",
		InputBlocked:      true,
		ForegroundIdle:    true,
		BackgroundIdle:    true,
		OnRace:            clonesnap.RaceRefuse,
		Workspace:         fixtureWorkspace(t),
		Store:             store,
	}
}

// storeFetch returns a blob fetcher reading through the store's real
// digest-to-path layout: the bytes Normalize sees are the bytes the
// capture installed, addressed by content digest.
func storeFetch(t *testing.T, store *localstore.ObjectStore) func(blobID string) ([]byte, error) {
	t.Helper()
	style := localstore.PathStylePOSIX
	if runtime.GOOS == "windows" {
		style = localstore.PathStyleWindows
	}
	return func(blobID string) ([]byte, error) {
		digest, err := scalar.ParseDigest(blobID)
		if err != nil {
			return nil, err
		}
		relative, err := localstore.DigestPathV1(digest, style)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(filepath.Join(store.DataRoot(), "objects", filepath.FromSlash(relative)))
	}
}

// normalizeRequest wires a Normalize request over a real capture
// result: the sealed manifests, the capture store fetcher, a fresh
// canonical sink, and the capture's workspace checkpoint.
func normalizeRequest(t *testing.T, capture *clonesnap.CaptureResult, captureStore, canonicalSink *localstore.ObjectStore) NormalizeRequest {
	t.Helper()
	return NormalizeRequest{
		CaptureManifest:  capture.CaptureManifest,
		RawManifest:      capture.RawManifest,
		Fetch:            storeFetch(t, captureStore),
		CanonicalSink:    canonicalSink,
		LogicalSessionID: fixtureLogicalID,
		MainActorID:      fixtureMainActor,
		Workspace:        capture.WorkspaceBinding,
	}
}

// captureMembers runs the real capture entry over the given members
// and returns the result with its store.
func captureMembers(t *testing.T, members []fixtureMember) (*clonesnap.CaptureResult, *localstore.ObjectStore) {
	t.Helper()
	root := writeFixtureStore(t, members)
	store := openTestStore(t)
	result, err := clonesnap.Capture(validCaptureRequest(t, root, store, members))
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	return result, store
}

// projectionBundle bundles one full real-path run: the capture result,
// its store, the canonical sink, the member bytes, and the
// normalized bundle.
type projectionBundle struct {
	capture *clonesnap.CaptureResult
	store   *localstore.ObjectStore
	sink    *localstore.ObjectStore
	members []fixtureMember
	result  *NormalizeResult
}

// projectMembers runs the full real path: capture, then Normalize
// over the sealed manifests with a fresh canonical sink.
func projectMembers(t *testing.T, members []fixtureMember) projectionBundle {
	t.Helper()
	capture, store := captureMembers(t, members)
	sink := openTestStore(t)
	result, err := Normalize(normalizeRequest(t, capture, store, sink))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	return projectionBundle{capture: capture, store: store, sink: sink, members: members, result: result}
}

// mustWorkspaceBytes captures one throwaway member and returns its
// sealed workspace checkpoint for builder-constructed requests.
func mustWorkspaceBytes(t *testing.T) []byte {
	t.Helper()
	capture, _ := captureMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-ws", "message/user", "native", "none", "main", `{"text":"x"}`),
		)},
	})
	return capture.WorkspaceBinding
}

// memberBytes returns the original bytes of one fixture member.
func (bundle projectionBundle) memberBytes(t *testing.T, key string) []byte {
	t.Helper()
	for _, member := range bundle.members {
		if member.key == key {
			return member.content
		}
	}
	t.Fatalf("no fixture member %q", key)
	return nil
}

// rec builds one fixture-native record line.
func rec(id, nativeType, origin, protection, actor, body string) string {
	return `{"v":1,"native_event_id":"` + id + `","native_type":"` + nativeType + `","origin":"` + origin +
		`","protection":"` + protection + `","actor":"` + actor + `","body":` + body + `}`
}

// jsonl joins record lines into member bytes with a trailing newline.
func jsonl(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

// eventByNativeID finds the decoded event carrying a native event ID.
func eventByNativeID(t *testing.T, result *NormalizeResult, id string) clonebundle.CanonicalEvent {
	t.Helper()
	for _, decoded := range result.DecodedEvents {
		if decoded.Evidence.NativeEventID != nil && *decoded.Evidence.NativeEventID == id {
			return decoded
		}
	}
	t.Fatalf("no projected event carries native_event_id %q", id)
	return clonebundle.CanonicalEvent{}
}

// mustResolveRef proves a raw reference resolves through the real
// path: the manifest ID names the sealed raw manifest, the
// descriptor ID names its raw entry, the entry names the installed
// blob, and the byte range slices back out exactly.
func mustResolveRef(t *testing.T, capture *clonesnap.CaptureResult, store *localstore.ObjectStore, ref clonebundle.RawReference) []byte {
	t.Helper()
	if ref.ManifestID.String() != capture.Raw.ManifestID.String() {
		t.Fatalf("raw reference manifest %q names %q, want the sealed raw manifest %q",
			ref.ManifestID.String(), ref.ManifestID.String(), capture.Raw.ManifestID.String())
	}
	blobID := ""
	for _, entry := range capture.Raw.Entries {
		if entry.BlobDescriptorID.String() == ref.BlobDescriptorID.String() {
			blobID = entry.BlobID.String()
		}
	}
	if blobID == "" {
		t.Fatalf("raw reference descriptor %q names no raw entry", ref.BlobDescriptorID.String())
	}
	payload, err := storeFetch(t, store)(blobID)
	if err != nil {
		t.Fatalf("fetch referenced blob %q: %v", blobID, err)
	}
	if ref.Offset+ref.Length > uint64(len(payload)) {
		t.Fatalf("raw reference [%d:%d] exceeds blob of %d bytes", ref.Offset, ref.Offset+ref.Length, len(payload))
	}
	return payload[ref.Offset : ref.Offset+ref.Length]
}

// listStoreBlobs enumerates installed blob bytes for test-side
// resolution.
func listStoreBlobs(t *testing.T, store *localstore.ObjectStore) [][]byte {
	t.Helper()
	objects := filepath.Join(store.DataRoot(), "objects")
	var blobs [][]byte
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
		blobs = append(blobs, raw)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk(objects) error = %v", err)
	}
	return blobs
}

// mustResolveOverflow proves an overflow content reference resolves:
// one installed sink blob carries descriptor bytes whose omit-self
// identity is the referenced ID, and the descriptor's blob holds the
// exact content bytes.
func mustResolveOverflow(t *testing.T, sink *localstore.ObjectStore, descriptorID string) []byte {
	t.Helper()
	for _, blob := range listStoreBlobs(t, sink) {
		calculated, field, err := canonicaljson.CalculateObjectIdentity(blob)
		if err != nil || string(field) != "descriptor_id" {
			continue
		}
		if calculated.String() != descriptorID {
			continue
		}
		var members map[string]json.RawMessage
		if err := json.Unmarshal(blob, &members); err != nil {
			t.Fatalf("Unmarshal(descriptor) error = %v", err)
		}
		var blobID string
		if err := json.Unmarshal(members["blob_id"], &blobID); err != nil {
			t.Fatalf("Unmarshal(descriptor blob_id) error = %v", err)
		}
		payload, err := storeFetch(t, sink)(blobID)
		if err != nil {
			t.Fatalf("fetch overflow payload %q: %v", blobID, err)
		}
		return payload
	}
	t.Fatalf("overflow descriptor %q resolves to no installed descriptor", descriptorID)
	return nil
}

// payloadText extracts one string payload fact.
func payloadText(t *testing.T, event clonebundle.CanonicalEvent, name string) string {
	t.Helper()
	raw, ok := event.Payload[name]
	if !ok {
		t.Fatalf("event kind %q payload misses member %q", event.Kind, name)
	}
	text, ok := raw.(string)
	if !ok {
		t.Fatalf("event kind %q payload member %q is %T, want string", event.Kind, name, raw)
	}
	return text
}

// payloadInt extracts one integer payload fact.
func payloadInt(t *testing.T, event clonebundle.CanonicalEvent, name string) int64 {
	t.Helper()
	raw, ok := event.Payload[name]
	if !ok {
		t.Fatalf("event kind %q payload misses member %q", event.Kind, name)
	}
	number, ok := raw.(int64)
	if !ok {
		t.Fatalf("event kind %q payload member %q is %T, want int64", event.Kind, name, raw)
	}
	return number
}

// payloadBool extracts one boolean payload fact.
func payloadBool(t *testing.T, event clonebundle.CanonicalEvent, name string) bool {
	t.Helper()
	raw, ok := event.Payload[name]
	if !ok {
		t.Fatalf("event kind %q payload misses member %q", event.Kind, name)
	}
	value, ok := raw.(bool)
	if !ok {
		t.Fatalf("event kind %q payload member %q is %T, want bool", event.Kind, name, raw)
	}
	return value
}

// sealedPayload decodes the sealed event bytes to a member map for
// absence assertions (the decoded form normalizes numbers).
func sealedPayload(t *testing.T, sealed []byte) map[string]any {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(sealed, &members); err != nil {
		t.Fatalf("Unmarshal(sealed event) error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(members["payload"], &payload); err != nil {
		t.Fatalf("Unmarshal(sealed payload) error = %v", err)
	}
	return payload
}
