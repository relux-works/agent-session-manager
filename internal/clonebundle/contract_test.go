package clonebundle

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

// Positive contract tests: every production Build entry constructs
// objects its Decode entry accepts, identical inputs produce
// byte-identical outputs, and derived flags match their derivation.

func TestBuildRawManifestRoundTrip(t *testing.T) {
	identity := fixtureNativeIdentity()
	identityDigest, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatalf("IdentityDigest() error = %v", err)
	}
	inputs := validEntryInputs(t)
	first, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
		"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil)
	if err != nil {
		t.Fatalf("BuildRawObjectManifest() error = %v", err)
	}
	second, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
		"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil)
	if err != nil {
		t.Fatalf("BuildRawObjectManifest() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("BuildRawObjectManifest identical inputs produced different bytes")
	}
	manifest, err := DecodeRawObjectManifest(first)
	if err != nil {
		t.Fatalf("DecodeRawObjectManifest() error = %v", err)
	}
	if len(manifest.Entries) != 2 {
		t.Fatalf("DecodeRawObjectManifest entries = %d, want 2", len(manifest.Entries))
	}
	var sum uint64
	for _, entry := range manifest.Entries {
		sum += entry.ByteCount
	}
	if manifest.TotalBytes != sum {
		t.Fatalf("TotalBytes = %d, entries sum to %d", manifest.TotalBytes, sum)
	}
	if manifest.SourceNativeSessionID != "native-session-alpha" {
		t.Fatalf("SourceNativeSessionID = %q", manifest.SourceNativeSessionID)
	}
	byKey := map[string]RawObjectEntry{}
	for _, entry := range manifest.Entries {
		byKey[entry.NativeItemKey] = entry
	}
	fetch := map[string][]byte{}
	for index, input := range inputs {
		fetch[manifest.Entries[index].BlobDescriptorID.String()] = input.Descriptor
	}
	if err := VerifyRawManifestDescriptors(manifest, func(id string) ([]byte, error) {
		descriptor, ok := fetch[id]
		if !ok {
			t.Fatalf("descriptor fetch missed %s", id)
		}
		return descriptor, nil
	}); err != nil {
		t.Fatalf("VerifyRawManifestDescriptors() error = %v", err)
	}
	if byKey["store/blob-a"].Class != "durable_payload" || byKey["store/blob-b"].Class != "durable_sidecar" {
		t.Fatalf("entry classes = %q, %q", byKey["store/blob-a"].Class, byKey["store/blob-b"].Class)
	}
}

func TestBuildCaptureManifestDerivesRawComplete(t *testing.T) {
	input := validCaptureInput(t)
	built, err := BuildCaptureManifest(input)
	if err != nil {
		t.Fatalf("BuildCaptureManifest() error = %v", err)
	}
	rebuilt, err := BuildCaptureManifest(input)
	if err != nil {
		t.Fatalf("BuildCaptureManifest() error = %v", err)
	}
	if !bytes.Equal(built, rebuilt) {
		t.Fatal("BuildCaptureManifest identical inputs produced different bytes")
	}
	manifest, err := DecodeCaptureManifest(built)
	if err != nil {
		t.Fatalf("DecodeCaptureManifest() error = %v", err)
	}
	if !manifest.RawComplete {
		t.Fatal("RawComplete = false, want true after complete reconciliation")
	}
	if len(manifest.ExcludedClasses) != 1 || manifest.ExcludedClasses[0] != "credential" {
		t.Fatalf("ExcludedClasses = %v, want [credential]", manifest.ExcludedClasses)
	}
	if err := VerifyCaptureReconciliation(manifest, input.PlanKeys, input.RawKeys); err != nil {
		t.Fatalf("VerifyCaptureReconciliation() error = %v", err)
	}
	if err := RefuseMaximalSafeUnlessComplete(manifest); err != nil {
		t.Fatalf("RefuseMaximalSafeUnlessComplete() error = %v", err)
	}
	if !manifest.CaptureBoundary.Stable() {
		t.Fatal("CaptureBoundary is not stable")
	}
}

func TestBuildCaptureManifestUnknownMakesIncomplete(t *testing.T) {
	input := validCaptureInput(t)
	count := uint64(9)
	descriptor := fixtureDigest("test-descriptor")
	input.Items = append(input.Items, CaptureItemInput{
		NativeItemKey: "store/zz-unknown", Class: "unknown", Disposition: "included",
		BlobDescriptorID: &descriptor, ByteCount: &count,
	})
	input.PlanKeys = append(input.PlanKeys, "store/zz-unknown")
	input.RawKeys = append(input.RawKeys, "store/zz-unknown")
	built, err := BuildCaptureManifest(input)
	if err != nil {
		t.Fatalf("BuildCaptureManifest() error = %v", err)
	}
	manifest, err := DecodeCaptureManifest(built)
	if err != nil {
		t.Fatalf("DecodeCaptureManifest() error = %v", err)
	}
	if manifest.RawComplete {
		t.Fatal("RawComplete = true with an unknown-class item")
	}
	if err := RefuseMaximalSafeUnlessComplete(manifest); !errors.Is(err, ErrInvalid) {
		t.Fatalf("RefuseMaximalSafeUnlessComplete() error = %v, want ErrInvalid", err)
	}
	if err := VerifyCaptureReconciliation(manifest, input.PlanKeys, input.RawKeys); !errors.Is(err, ErrInvalid) {
		t.Fatalf("VerifyCaptureReconciliation() error = %v, want ErrInvalid", err)
	}
}

func TestBuildCaptureManifestUnstableArchive(t *testing.T) {
	input := validCaptureInput(t)
	input.Boundary = BoundaryInput{
		Kind: "unstable_archive", Generation: "generation-9",
		PreCaptureDigest: fixtureDigest("test-pre"), PostCaptureDigest: fixtureDigest("test-post"),
		Core: true,
	}
	built, err := BuildCaptureManifest(input)
	if err != nil {
		t.Fatalf("BuildCaptureManifest(unstable) error = %v", err)
	}
	manifest, err := DecodeCaptureManifest(built)
	if err != nil {
		t.Fatalf("DecodeCaptureManifest(unstable) error = %v", err)
	}
	if manifest.CaptureBoundary.Stable() {
		t.Fatal("unstable boundary reports stable")
	}
	if err := RefuseUnstableForTarget(manifest.CaptureBoundary); !errors.Is(err, ErrInvalid) {
		t.Fatalf("RefuseUnstableForTarget(unstable) error = %v, want ErrInvalid", err)
	}
	stable, err := DecodeCaptureBoundary([]byte(`{"kind":"stable","proof":{"proof_kind":"closed_store","source_generation":"g","snapshot_identity_digest":null,"pre_capture_digest":` +
		`"` + fixtureDigest("test-pre-capture") + `","post_capture_digest":"` + fixtureDigest("test-pre-capture") +
		`","input_blocked":true,"foreground_idle":true,"background_idle":true,"extensions":{}},"extensions":{}}`))
	if err != nil {
		t.Fatalf("DecodeCaptureBoundary(stable) error = %v", err)
	}
	if err := RefuseUnstableForTarget(stable); err != nil {
		t.Fatalf("RefuseUnstableForTarget(stable) error = %v", err)
	}
}

func TestBuildCanonicalSessionRoundTrip(t *testing.T) {
	input := validSessionInput()
	first, err := BuildCanonicalSession(input)
	if err != nil {
		t.Fatalf("BuildCanonicalSession() error = %v", err)
	}
	second, err := BuildCanonicalSession(input)
	if err != nil {
		t.Fatalf("BuildCanonicalSession() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("BuildCanonicalSession identical inputs produced different bytes")
	}
	session, err := DecodeCanonicalSession(first)
	if err != nil {
		t.Fatalf("DecodeCanonicalSession() error = %v", err)
	}
	if len(session.Actors) != 2 || len(session.EventIDs) != 2 || len(session.HeadEventIDs) != 1 {
		t.Fatalf("session actors/events/heads = %d/%d/%d", len(session.Actors), len(session.EventIDs), len(session.HeadEventIDs))
	}
	if session.Title == nil || *session.Title != "alpha session" {
		t.Fatalf("session title = %v", session.Title)
	}
}

func TestBuildCanonicalEventRoundTrip(t *testing.T) {
	for _, status := range []string{"exact", "partial", "unavailable", "synthesized"} {
		input := validEventInput(status)
		first, err := BuildCanonicalEvent(input)
		if err != nil {
			t.Fatalf("BuildCanonicalEvent(%s) error = %v", status, err)
		}
		second, err := BuildCanonicalEvent(input)
		if err != nil {
			t.Fatalf("BuildCanonicalEvent(%s) error = %v", status, err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("BuildCanonicalEvent(%s) identical inputs produced different bytes", status)
		}
		event, err := DecodeCanonicalEvent(first)
		if err != nil {
			t.Fatalf("DecodeCanonicalEvent(%s) error = %v", status, err)
		}
		if event.Evidence.CaptureStatus != status {
			t.Fatalf("CaptureStatus = %q, want %q", event.Evidence.CaptureStatus, status)
		}
		if status == "synthesized" && event.Evidence.CoreOperationID == nil {
			t.Fatal("synthesized evidence lost its core operation")
		}
	}
}

func TestIdentityDigestIsStable(t *testing.T) {
	identity := fixtureNativeIdentity()
	first, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatalf("IdentityDigest() error = %v", err)
	}
	second, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatalf("IdentityDigest() error = %v", err)
	}
	if first != second {
		t.Fatal("IdentityDigest identical inputs produced different digests")
	}
	other := identity
	other.NativeSessionID = "native-session-beta"
	third, err := IdentityDigest(other, nil)
	if err != nil {
		t.Fatalf("IdentityDigest() error = %v", err)
	}
	if third == first {
		t.Fatal("IdentityDigest did not change with the native session ID")
	}
}

func TestExcludedClassesExactGate(t *testing.T) {
	items := []CaptureItem{
		{NativeItemKey: "a", Class: "durable_payload", Disposition: "included"},
		{NativeItemKey: "b", Class: "credential", Disposition: "excluded"},
	}
	if !excludedClassesExact(items, []string{"credential"}) {
		t.Fatal("excludedClassesExact(exact) = false, want true")
	}
	if excludedClassesExact(items, []string{"credential", "unknown"}) {
		t.Fatal("excludedClassesExact(extra) = true, want false")
	}
	if excludedClassesExact(items, nil) {
		t.Fatal("excludedClassesExact(missing) = true, want false")
	}
	if excludedClassesExact(items, []string{"unknown", "credential"}) {
		t.Fatal("excludedClassesExact(unordered) = true, want false")
	}
}

func TestOrdinalContiguity(t *testing.T) {
	if err := CheckOrdinalContiguity([]uint64{0, 1, 2, 3}); err != nil {
		t.Fatalf("CheckOrdinalContiguity(contiguous) error = %v", err)
	}
	if err := CheckOrdinalContiguity([]uint64{2, 0, 1}); err != nil {
		t.Fatalf("CheckOrdinalContiguity(unordered set) error = %v", err)
	}
	for _, ordinals := range [][]uint64{{1}, {0, 2}, {0, 0, 1}, {0, 1, 5}} {
		if err := CheckOrdinalContiguity(ordinals); !errors.Is(err, ErrInvalid) {
			t.Fatalf("CheckOrdinalContiguity(%v) error = %v, want ErrInvalid", ordinals, err)
		}
	}
}

func TestInstallRawBlobIsIdempotent(t *testing.T) {
	store := openFixtureStore(t)
	payload := []byte("raw-evidence-bytes")
	descriptor, blobID, _, size := makeDescriptor(t, payload)
	first, err := InstallRawBlob(store, descriptor, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("InstallRawBlob() error = %v", err)
	}
	second, err := InstallRawBlob(store, descriptor, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("InstallRawBlob(second install) error = %v", err)
	}
	if first != second {
		t.Fatalf("InstallRawBlob agreements differ: %+v vs %+v", first, second)
	}
	if first.BlobID.String() != blobID || first.Size != size {
		t.Fatalf("InstallRawBlob agreement = %+v, want blob %s size %d", first, blobID, size)
	}
	if _, err := InstallRawBlob(store, descriptor, bytes.NewReader([]byte("tampered-evidence!"))); !errors.Is(err, ErrInvalid) {
		t.Fatalf("InstallRawBlob(tampered bytes) error = %v, want ErrInvalid", err)
	}
}

func TestCaptureClassVocabularyIsPinned(t *testing.T) {
	pinned := []string{
		"durable_payload",
		"durable_index_required",
		"durable_sidecar",
		"derived_cache_optional",
		"credential",
		"machine_auth",
		"runtime_state",
		"transient_lock",
		"unknown",
	}
	for _, class := range pinned {
		if !ValidCaptureClass(class) {
			t.Fatalf("ValidCaptureClass(%q) = false, want true", class)
		}
	}
	for _, class := range []string{"", "durable", "CREDENTIAL", "unknown ", "derived_cache"} {
		if ValidCaptureClass(class) {
			t.Fatalf("ValidCaptureClass(%q) = true, want false", class)
		}
	}
}

func TestEventKindVocabularyIsPinned(t *testing.T) {
	pinned := []string{
		"session_started", "instruction_snapshot", "user_message", "assistant_message",
		"reasoning_summary", "opaque_reasoning", "tool_definition_snapshot", "tool_call",
		"tool_result", "approval_request", "approval_response", "plan_update", "progress",
		"usage", "rate_limit", "file_change", "compaction", "turn_started", "turn_completed",
		"turn_aborted", "subagent_started", "subagent_completed", "error",
		"migration_checkpoint", "session_finished", "opaque_event",
	}
	if len(pinned) != 26 {
		t.Fatalf("pinned kinds = %d, want 26", len(pinned))
	}
	for _, kind := range pinned {
		if !ValidEventKind(kind) {
			t.Fatalf("ValidEventKind(%q) = false, want true", kind)
		}
	}
	for _, kind := range []string{"", "message", "USER_MESSAGE", "tool_call "} {
		if ValidEventKind(kind) {
			t.Fatalf("ValidEventKind(%q) = true, want false", kind)
		}
	}
}

func TestNativeIdentityRoundTrip(t *testing.T) {
	identity := fixtureNativeIdentity()
	encoded, err := EncodeNativeIdentity(identity, nil)
	if err != nil {
		t.Fatalf("EncodeNativeIdentity() error = %v", err)
	}
	decoded, err := DecodeNativeIdentity(encoded)
	if err != nil {
		t.Fatalf("DecodeNativeIdentity() error = %v", err)
	}
	if decoded != identity {
		t.Fatalf("DecodeNativeIdentity round trip = %+v, want %+v", decoded, identity)
	}
	var members map[string]any
	if err := json.Unmarshal(encoded, &members); err != nil {
		t.Fatal(err)
	}
	members["extensions"] = map[string]any{"com.example.origin": "probe"}
	again, err := json.Marshal(members)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeNativeIdentity(again); err != nil {
		t.Fatalf("DecodeNativeIdentity(reverse-DNS extensions) error = %v", err)
	}
}
