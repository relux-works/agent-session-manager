package clonesnap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Positive contract tests: Capture seals both manifests from real
// store bytes through the landed builders, every sealed manifest
// re-decodes, identical runs produce byte-identical manifests, and
// the sealed members match independently recomputed values.

func TestCaptureSealsBothManifestsFromStoreBytes(t *testing.T) {
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	result := mustCapture(t, validCaptureRequest(t, storeRoot, store))

	if result.BoundaryKind != "stable" {
		t.Fatalf("BoundaryKind = %q, want stable", result.BoundaryKind)
	}
	if result.PreDigest != result.PostDigest {
		t.Fatalf("PreDigest %q != PostDigest %q on a quiet store", result.PreDigest, result.PostDigest)
	}
	if !result.RawInstalled || !result.CaptureInstalled {
		t.Fatalf("Installed = (%v, %v), want (true, true) on first capture", result.RawInstalled, result.CaptureInstalled)
	}

	// The raw manifest carries exactly the two included members,
	// sorted by native key, with byte counts recomputed here.
	raw, err := clonebundle.DecodeRawObjectManifest(result.RawManifest)
	if err != nil {
		t.Fatalf("DecodeRawObjectManifest() error = %v", err)
	}
	if len(raw.Entries) != 2 {
		t.Fatalf("raw entries = %d, want 2", len(raw.Entries))
	}
	if raw.Entries[0].NativeItemKey != "store/blob-a" || raw.Entries[1].NativeItemKey != "store/blob-b" {
		t.Fatalf("raw entry keys = %q, %q", raw.Entries[0].NativeItemKey, raw.Entries[1].NativeItemKey)
	}
	wantA := sha256.Sum256([]byte("alpha-payload-bytes"))
	wantB := sha256.Sum256([]byte("beta-payload-bytes"))
	if raw.Entries[0].BlobID.String() != "sha256:"+hex.EncodeToString(wantA[:]) {
		t.Fatalf("blob-a blob_id = %q", raw.Entries[0].BlobID)
	}
	if raw.Entries[1].BlobID.String() != "sha256:"+hex.EncodeToString(wantB[:]) {
		t.Fatalf("blob-b blob_id = %q", raw.Entries[1].BlobID)
	}
	if raw.Entries[0].ByteCount != uint64(len("alpha-payload-bytes")) {
		t.Fatalf("blob-a byte_count = %d", raw.Entries[0].ByteCount)
	}
	var sum uint64
	for _, entry := range raw.Entries {
		sum += entry.ByteCount
	}
	if raw.TotalBytes != sum {
		t.Fatalf("TotalBytes = %d, entries sum to %d", raw.TotalBytes, sum)
	}
	if raw.SourceNativeSessionID != "native-session-snap" {
		t.Fatalf("SourceNativeSessionID = %q", raw.SourceNativeSessionID)
	}

	// The capture manifest reconciles: one item per plan candidate,
	// the credential excluded with its stable reason, raw_complete
	// derived true, and the exact raw manifest identity.
	manifest, err := clonebundle.DecodeCaptureManifest(result.CaptureManifest)
	if err != nil {
		t.Fatalf("DecodeCaptureManifest() error = %v", err)
	}
	if len(manifest.Items) != 3 {
		t.Fatalf("capture items = %d, want 3", len(manifest.Items))
	}
	byKey := map[string]clonebundle.CaptureItem{}
	for _, item := range manifest.Items {
		byKey[item.NativeItemKey] = item
	}
	if !byKey["store/blob-a"].Included() || !byKey["store/blob-b"].Included() {
		t.Fatal("included members did not seal as included")
	}
	credential := byKey["store/token-cache"]
	if credential.Included() {
		t.Fatal("credential sealed as included")
	}
	if credential.ExclusionReason == nil || *credential.ExclusionReason != "credential_excluded" {
		t.Fatalf("credential exclusion reason = %v", credential.ExclusionReason)
	}
	if len(manifest.ExcludedClasses) != 1 || manifest.ExcludedClasses[0] != "credential" {
		t.Fatalf("ExcludedClasses = %v, want [credential]", manifest.ExcludedClasses)
	}
	if !manifest.RawComplete {
		t.Fatal("RawComplete = false, want true after complete reconciliation")
	}
	if manifest.SourceRawObjectManifest != raw.ManifestID {
		t.Fatalf("SourceRawObjectManifest %q != raw %q", manifest.SourceRawObjectManifest, raw.ManifestID)
	}
	if !manifest.CaptureBoundary.Stable() {
		t.Fatal("CaptureBoundary is not stable")
	}
	proof := manifest.CaptureBoundary.Proof
	if proof.ProofKind != "closed_store" {
		t.Fatalf("ProofKind = %q, want closed_store", proof.ProofKind)
	}
	if proof.PreCaptureDigest != proof.PostCaptureDigest {
		t.Fatal("sealed proof digests differ on a quiet store")
	}
	if proof.PreCaptureDigest != result.PreDigest {
		t.Fatal("sealed proof pre digest is not the measured source digest")
	}
	if proof.SourceGeneration.String() != "generation-7" {
		t.Fatalf("SourceGeneration = %q", proof.SourceGeneration)
	}

	// The workspace checkpoint seals the observed repository state.
	binding, err := clonebundle.DecodeWorkspaceBinding(result.WorkspaceBinding)
	if err != nil {
		t.Fatalf("DecodeWorkspaceBinding() error = %v", err)
	}
	if binding.CwdRelative != "work/trees/alpha" {
		t.Fatalf("CwdRelative = %q", binding.CwdRelative)
	}
	if binding.Branch == nil || *binding.Branch != "main" {
		t.Fatalf("Branch = %v", binding.Branch)
	}
	if binding.HeadDigest == nil || binding.HeadDigest.String() != fixtureDigest("snap-head") {
		t.Fatalf("HeadDigest = %v", binding.HeadDigest)
	}
}

func TestCaptureInstallsEveryPayloadBlob(t *testing.T) {
	// Section 10.2 MUST: before publishing a record that
	// references a blob, the blob is installed. Every raw entry's
	// blob is present in the store after Capture.
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	result := mustCapture(t, validCaptureRequest(t, storeRoot, store))
	assertEveryRawBlobInstalled(t, store, result.Raw)
}

func TestCaptureIsByteIdenticalAcrossRuns(t *testing.T) {
	storeRoot := fixtureStore(t)
	first := mustCapture(t, validCaptureRequest(t, storeRoot, openTestStore(t)))
	second := mustCapture(t, validCaptureRequest(t, storeRoot, openTestStore(t)))
	if !bytes.Equal(first.RawManifest, second.RawManifest) {
		t.Fatal("RawManifest bytes differ across identical runs")
	}
	if !bytes.Equal(first.CaptureManifest, second.CaptureManifest) {
		t.Fatal("CaptureManifest bytes differ across identical runs")
	}
	if first.PreDigest != second.PreDigest {
		t.Fatal("PreDigest differs across identical runs")
	}
}

func TestCaptureUnknownMakesRawIncomplete(t *testing.T) {
	storeRoot := fixtureStore(t)
	writeStoreFile(t, storeRoot, "store/zz-unknown", []byte("unknown-kind-bytes"))
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
		{NativeKey: "store/zz-unknown", Class: "unknown", Required: true},
	}
	result := mustCapture(t, request)
	if result.Capture.RawComplete {
		t.Fatal("RawComplete = true with an unknown-class item, want false")
	}
	if len(result.Raw.Entries) != 3 {
		t.Fatalf("raw entries = %d, want 3 (unknown is captured)", len(result.Raw.Entries))
	}
	if err := clonebundle.RefuseMaximalSafeUnlessComplete(result.Capture); err == nil {
		t.Fatal("RefuseMaximalSafeUnlessComplete() = nil with an unknown-class item")
	} else if !strings.Contains(err.Error(), "maximal_safe") {
		t.Fatalf("RefuseMaximalSafeUnlessComplete() error = %v, want the maximal_safe refusal", err)
	}
}

func TestCaptureOptionalAbsentUnknownMakesRawIncomplete(t *testing.T) {
	// P3-ε: an EXCLUDED unknown-class item — an optional plan
	// member absent from the store — still drives
	// raw_complete=false and still blocks maximal_safe at the
	// projection entry, exactly like an included unknown item.
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
		{NativeKey: "store/zz-unknown-absent", Class: "unknown", Required: false},
	}
	result := mustCapture(t, request)
	if result.Capture.RawComplete {
		t.Fatal("RawComplete = true with an excluded unknown-class item, want false")
	}
	var found *clonebundle.CaptureItem
	for index := range result.Capture.Items {
		if result.Capture.Items[index].NativeItemKey == "store/zz-unknown-absent" {
			found = &result.Capture.Items[index]
		}
	}
	if found == nil {
		t.Fatal("optional absent unknown candidate has no item")
	}
	if found.Included() {
		t.Fatal("optional absent unknown candidate sealed as included")
	}
	if found.ExclusionReason == nil || *found.ExclusionReason != "plan_optional_absent" {
		t.Fatalf("optional absent unknown reason = %v", found.ExclusionReason)
	}
	if _, err := AdmitForTarget(result.CaptureManifest, "maximal_safe"); err == nil {
		t.Fatal("AdmitForTarget(maximal_safe) admitted a manifest with an excluded unknown item")
	} else {
		requireRefusalDetail(t, err, "maximal_safe")
	}
	if _, err := AdmitForTarget(result.CaptureManifest, "strict_exact"); err != nil {
		t.Fatalf("AdmitForTarget(strict_exact) error = %v", err)
	}
}

func TestCaptureImmutableSnapshotProof(t *testing.T) {
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.ProofKind = "immutable_snapshot"
	identity := fixtureDigest("snap-snapshot-identity")
	request.SnapshotIdentity = &identity
	result := mustCapture(t, request)
	proof := result.Capture.CaptureBoundary.Proof
	if proof.ProofKind != "immutable_snapshot" {
		t.Fatalf("ProofKind = %q", proof.ProofKind)
	}
	if proof.SnapshotIdentity == nil || proof.SnapshotIdentity.String() != identity {
		t.Fatalf("SnapshotIdentity = %v", proof.SnapshotIdentity)
	}
}

func TestCaptureExternalNativeBasis(t *testing.T) {
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.SourceBasis = clonebundle.SourceBasisInput{
		Kind:              "external_native",
		ExternalSourceRef: "provider-native-source-9",
	}
	result := mustCapture(t, request)
	if result.Capture.SourceBasis.Kind != "external_native" {
		t.Fatalf("SourceBasis kind = %q", result.Capture.SourceBasis.Kind)
	}
	if result.Capture.SourceBasis.ExternalSourceRef == nil ||
		*result.Capture.SourceBasis.ExternalSourceRef != "provider-native-source-9" {
		t.Fatalf("ExternalSourceRef = %v", result.Capture.SourceBasis.ExternalSourceRef)
	}
}

func TestCaptureOptionalAbsentSealsExcluded(t *testing.T) {
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/maybe", Class: "durable_payload", Required: false},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
	}
	result := mustCapture(t, request)
	if len(result.Capture.Items) != 4 {
		t.Fatalf("capture items = %d, want 4", len(result.Capture.Items))
	}
	var found *clonebundle.CaptureItem
	for index := range result.Capture.Items {
		if result.Capture.Items[index].NativeItemKey == "store/maybe" {
			found = &result.Capture.Items[index]
		}
	}
	if found == nil {
		t.Fatal("optional absent candidate has no item")
	}
	if found.Included() {
		t.Fatal("optional absent candidate sealed as included")
	}
	if found.ExclusionReason == nil || *found.ExclusionReason != "plan_optional_absent" {
		t.Fatalf("optional absent reason = %v", found.ExclusionReason)
	}
	if !result.Capture.RawComplete {
		t.Fatal("RawComplete = false, want true: the absent optional reconciles as excluded")
	}
}

func TestCaptureEmptyMemberSealsEmptyBlob(t *testing.T) {
	storeRoot := fixtureStore(t)
	writeStoreFile(t, storeRoot, "store/blob-empty", []byte{})
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/blob-empty", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
	}
	result := mustCapture(t, request)
	var entry *clonebundle.RawObjectEntry
	for index := range result.Raw.Entries {
		if result.Raw.Entries[index].NativeItemKey == "store/blob-empty" {
			entry = &result.Raw.Entries[index]
		}
	}
	if entry == nil {
		t.Fatal("empty member has no raw entry")
	}
	if entry.ByteCount != 0 {
		t.Fatalf("empty member byte_count = %d", entry.ByteCount)
	}
	emptySum := sha256.Sum256(nil)
	if entry.BlobID.String() != "sha256:"+hex.EncodeToString(emptySum[:]) {
		t.Fatalf("empty member blob_id = %q", entry.BlobID)
	}
	descriptor, err := BuildBlobDescriptor([]byte{})
	if err != nil {
		t.Fatalf("BuildBlobDescriptor(empty) error = %v", err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(descriptor.Descriptor, &members); err != nil {
		t.Fatal(err)
	}
	if string(members["chunks"]) != "[]" {
		t.Fatalf("empty descriptor chunks = %s, want []", members["chunks"])
	}
	if string(members["media_type"]) != `"application/octet-stream"` {
		t.Fatalf("empty descriptor media_type = %s", members["media_type"])
	}
}

func TestCaptureMultiChunkMember(t *testing.T) {
	storeRoot := fixtureStore(t)
	payload := bytes.Repeat([]byte("0123456789abcdef"), 4*1048576/16+1)
	payload = payload[:4*1048576+10]
	writeStoreFile(t, storeRoot, "store/blob-big", payload)
	store := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, store)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/blob-big", Class: "durable_payload", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
	}
	result := mustCapture(t, request)
	descriptor, err := BuildBlobDescriptor(payload)
	if err != nil {
		t.Fatalf("BuildBlobDescriptor() error = %v", err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(descriptor.Descriptor, &members); err != nil {
		t.Fatal(err)
	}
	var chunks []map[string]json.RawMessage
	if err := json.Unmarshal(members["chunks"], &chunks); err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	if string(chunks[0]["index"]) != "0" || string(chunks[0]["offset"]) != "0" ||
		string(chunks[0]["size"]) != "4194304" {
		t.Fatalf("chunk[0] = %v", chunks[0])
	}
	if string(chunks[1]["index"]) != "1" || string(chunks[1]["offset"]) != "4194304" ||
		string(chunks[1]["size"]) != "10" {
		t.Fatalf("chunk[1] = %v", chunks[1])
	}
	firstSum := sha256.Sum256(payload[:4194304])
	if string(chunks[0]["chunk_id"]) != `"sha256:`+hex.EncodeToString(firstSum[:])+`"` {
		t.Fatalf("chunk[0] chunk_id = %s", chunks[0]["chunk_id"])
	}
	var found *clonebundle.RawObjectEntry
	for index := range result.Raw.Entries {
		if result.Raw.Entries[index].NativeItemKey == "store/blob-big" {
			found = &result.Raw.Entries[index]
		}
	}
	if found == nil || found.BlobID != descriptor.BlobID || found.BlobDescriptorID != descriptor.DescriptorID {
		t.Fatal("raw entry does not reference the sealed multi-chunk descriptor")
	}
}

func TestDefaultMaxSingleBytesIs128GiB(t *testing.T) {
	if DefaultMaxSingleBytes != 137438953472 {
		t.Fatalf("DefaultMaxSingleBytes = %d, want 137438953472", DefaultMaxSingleBytes)
	}
}

func storeFixturePayload(t *testing.T, storeRoot, key string) string {
	t.Helper()
	return filepath.Join(storeRoot, filepath.FromSlash(key))
}

func TestCaptureDescriptorAgreementEndToEnd(t *testing.T) {
	storeRoot := fixtureStore(t)
	store := openTestStore(t)
	result := mustCapture(t, validCaptureRequest(t, storeRoot, store))
	fetch := map[string][]byte{}
	for _, entry := range result.Raw.Entries {
		payload, err := os.ReadFile(storeFixturePayload(t, storeRoot, entry.NativeItemKey))
		if err != nil {
			t.Fatal(err)
		}
		descriptor, err := BuildBlobDescriptor(payload)
		if err != nil {
			t.Fatal(err)
		}
		fetch[descriptor.DescriptorID.String()] = descriptor.Descriptor
	}
	if err := clonebundle.VerifyRawManifestDescriptors(result.Raw, func(id string) ([]byte, error) {
		descriptor, ok := fetch[id]
		if !ok {
			t.Fatalf("descriptor fetch missed %s", id)
		}
		return descriptor, nil
	}); err != nil {
		t.Fatalf("VerifyRawManifestDescriptors() error = %v", err)
	}
	if got := scalar.SHA256Digest([]byte("alpha-payload-bytes")).String(); result.Raw.Entries[0].BlobID.String() != got {
		t.Fatalf("blob-a blob_id = %q, want %q", result.Raw.Entries[0].BlobID, got)
	}
}
