package clonebundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Shared fixtures for the clonebundle suite. Every literal is
// hand-written and independent of production constants: the verdict
// always comes from production; the vectors never do.

const (
	fixtureOperationID = "0198f4c8-8e50-7f66-8f70-1234567890ac"
	fixtureBundleID    = "0198f4c8-8e50-7f66-8f70-1234567890b1"
	fixtureSessionID   = "0198f4c8-8e50-7f66-8f70-1234567890c2"
	fixtureRecordSeed  = "test-session-record"
	fixtureHostID      = "0198f4c8-8e50-7f66-8f70-1234567890d3"
	fixtureActorMain   = "0198f4c8-8e50-7f66-8f70-1234567890e4"
	fixtureActorSub    = "0198f4c8-8e50-7f66-8f70-1234567890f5"
	fixtureTurnID      = "0198f4c8-8e50-7f66-8f70-1234567890a6"
	fixtureCreatedAt   = "2026-09-17T04:05:00.000Z"
	fixtureWorkspaceID = "0198f4c8-8e50-7f66-8f70-1234567890b7"
)

func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fixtureTupleJSON() string {
	return fmt.Sprintf(`{"environment_id":"test.env","environment_version":"2.1.0","platform":"linux","architecture":"amd64","store_schema_fingerprint":%q,"adapter_version":"1.2.3"}`,
		fixtureDigest("test-store-fingerprint"))
}

func fixtureWorkspaceJSON() string {
	return fmt.Sprintf(`{"logical_workspace_id":%q,"cwd_relative":"work/trees/alpha","repository_remote_fingerprints":[%q],"branch":"main","head_digest":%q,"index_digest":null,"working_tree_digest":null,"extensions":{}}`,
		fixtureWorkspaceID, fixtureDigest("test-remote"), fixtureDigest("test-head"))
}

func fixtureNativeIdentity() NativeIdentity {
	workspace, err := scalar.ParseUUIDv7(fixtureWorkspaceID)
	if err != nil {
		panic(err)
	}
	return NativeIdentity{
		NativeSessionID:    "native-session-alpha",
		IdentityKind:       "provider_native",
		LogicalWorkspaceID: workspace,
	}
}

func mustDigest(t *testing.T, value string) scalar.Digest {
	t.Helper()
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		t.Fatalf("ParseDigest(%q) error = %v", value, err)
	}
	return digest
}

func strptr(value string) *string { return &value }

func u64ptr(value uint64) *uint64 { return &value }

// makeDescriptor builds one valid Section 10.2 Blob Descriptor for
// the given payload through the canonicaljson production entries:
// the shape validates and the identity verifies before the fixture
// is returned.
func makeDescriptor(t *testing.T, payload []byte) (descriptor []byte, blobID, descriptorID string, size uint64) {
	t.Helper()
	sum := sha256.Sum256(payload)
	blobID = "sha256:" + hex.EncodeToString(sum[:])
	size = uint64(len(payload))
	var chunks []any
	if len(payload) > 0 {
		chunks = []any{map[string]any{
			"index":    0,
			"offset":   0,
			"size":     size,
			"chunk_id": blobID,
		}}
	} else {
		chunks = []any{}
	}
	object := map[string]any{
		"schema":         "urn:ax:schema:blob",
		"schema_version": "1.0.0",
		"descriptor_id":  fixtureDigest("placeholder-descriptor"),
		"blob_id":        blobID,
		"size":           size,
		"media_type":     "application/octet-stream",
		"chunks":         chunks,
	}
	plain, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	calculated, _, err := canonicaljson.CalculateObjectIdentity(plain)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity(fixture descriptor) error = %v", err)
	}
	object["descriptor_id"] = calculated.String()
	descriptor, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(descriptor); err != nil {
		t.Fatalf("VerifyObjectIdentity(fixture descriptor) error = %v", err)
	}
	return descriptor, blobID, calculated.String(), size
}

func openFixtureStore(t *testing.T) *localstore.ObjectStore {
	t.Helper()
	home := t.TempDir()
	config := filepath.Join(home, "config")
	if err := os.Mkdir(config, 0o700); err != nil {
		t.Fatal(err)
	}
	platform := scalar.PlatformLinux
	if runtime.GOOS == "darwin" {
		platform = scalar.PlatformMacOS
	} else if runtime.GOOS == "windows" {
		platform = scalar.PlatformWindows
	}
	resolved, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: platform,
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
	store, err := localstore.OpenObjectStore(resolved)
	if err != nil {
		t.Fatalf("OpenObjectStore() error = %v", err)
	}
	return store
}

func validEntryInputs(t *testing.T) []EntryInput {
	t.Helper()
	first, firstBlob, firstDescriptor, firstSize := makeDescriptor(t, []byte("first-raw-object"))
	second, secondBlob, secondDescriptor, secondSize := makeDescriptor(t, []byte("second-raw-object"))
	return []EntryInput{
		{NativeItemKey: "store/blob-a", Class: "durable_payload", ByteCount: firstSize, BlobID: firstBlob, DescriptorID: firstDescriptor, Descriptor: first},
		{NativeItemKey: "store/blob-b", Class: "durable_sidecar", ByteCount: secondSize, BlobID: secondBlob, DescriptorID: secondDescriptor, Descriptor: second},
	}
}

func validCaptureItems(t *testing.T) []CaptureItemInput {
	t.Helper()
	_, _, firstDescriptor, firstSize := makeDescriptor(t, []byte("first-raw-object"))
	_, _, secondDescriptor, secondSize := makeDescriptor(t, []byte("second-raw-object"))
	firstCount := firstSize
	secondCount := secondSize
	return []CaptureItemInput{
		{NativeItemKey: "store/blob-a", Class: "durable_payload", Disposition: "included", BlobDescriptorID: &firstDescriptor, ByteCount: &firstCount},
		{NativeItemKey: "store/blob-b", Class: "durable_sidecar", Disposition: "included", BlobDescriptorID: &secondDescriptor, ByteCount: &secondCount},
		{NativeItemKey: "store/token-cache", Class: "credential", Disposition: "excluded", ExclusionReason: strptr("credential_excluded")},
	}
}

func validBoundaryInput() BoundaryInput {
	return BoundaryInput{
		Kind:              "stable",
		ProofKind:         "closed_store",
		Generation:        "generation-42",
		PreCaptureDigest:  fixtureDigest("test-pre-capture"),
		PostCaptureDigest: fixtureDigest("test-pre-capture"),
		InputBlocked:      true,
		ForegroundIdle:    true,
		BackgroundIdle:    true,
	}
}

func validCaptureInput(t *testing.T) CaptureManifestInput {
	t.Helper()
	return CaptureManifestInput{
		OperationID: fixtureOperationID,
		BundleID:    fixtureBundleID,
		SourceBasis: SourceBasisInput{
			Kind:                     "ax_session",
			SourceSessionID:          fixtureSessionID,
			SourceSessionRecordID:    fixtureDigest(fixtureRecordSeed),
			SourceCheckpointID:       fixtureDigest("test-checkpoint"),
			SourceProviderIdentityID: fixtureDigest("test-provider-identity"),
		},
		SourceEnvironment:   []byte(fixtureTupleJSON()),
		SourceIdentity:      fixtureNativeIdentity(),
		CapturePlanDigest:   fixtureDigest("test-capture-plan"),
		Boundary:            validBoundaryInput(),
		SourceRawManifestID: fixtureDigest("test-raw-manifest"),
		Items:               validCaptureItems(t),
		PlanKeys:            []string{"store/blob-a", "store/blob-b", "store/token-cache"},
		RawKeys:             []string{"store/blob-a", "store/blob-b"},
		CreatedByHostID:     fixtureHostID,
		CreatedAt:           fixtureCreatedAt,
	}
}

func validSessionInput() CanonicalSessionInput {
	return CanonicalSessionInput{
		LogicalSessionID:      fixtureSessionID,
		SourceEnvironment:     []byte(fixtureTupleJSON()),
		SourceNativeSessionID: "native-session-alpha",
		Title:                 strptr("alpha session"),
		Workspace:             []byte(fixtureWorkspaceJSON()),
		Actors: []ActorInput{
			{ActorID: fixtureActorMain, Kind: "main", Name: strptr("main")},
			{ActorID: fixtureActorSub, Kind: "subagent", ParentActorID: strptr(fixtureActorMain)},
		},
		EventIDs:     []string{fixtureDigest("test-event-0"), fixtureDigest("test-event-1")},
		HeadEventIDs: []string{fixtureDigest("test-event-1")},
		CreatedAt:    strptr(fixtureCreatedAt),
	}
}

func validEventInput(status string) CanonicalEventInput {
	input := CanonicalEventInput{
		LogicalSessionID: fixtureSessionID,
		Ordinal:          0,
		ActorID:          fixtureActorMain,
		Kind:             "user_message",
		Visibility:       "public",
		Payload:          []byte(`{"content_blocks":[{"type":"text","content":"hello"}],"extensions":{}}`),
		Evidence: EvidenceInput{
			Environment:     []byte(fixtureTupleJSON()),
			NativeSessionID: "native-session-alpha",
			NativeEventID:   strptr("native-event-1"),
			NativeType:      strptr("chat.message"),
			RawRefs: []RawRefInput{
				{ManifestID: fixtureDigest("test-raw-manifest"), BlobDescriptorID: fixtureDigest("test-descriptor"), Offset: 0, Length: 128},
			},
			CaptureStatus: status,
			ReasonCodes:   []string{"source_complete"},
		},
	}
	if status == "synthesized" {
		input.Evidence.NativeEventID = nil
		input.Evidence.CoreOperationID = strptr(fixtureOperationID)
	}
	return input
}
