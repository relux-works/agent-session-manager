package clonesnap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var errHookRefused = errors.New("test hook refused")

// Refusal table: every capture gate refuses through the production
// Capture entry with the offending member named. Each row drives the
// entry; the mutant harness weakens each gate and the named killer
// fails.

func captureRefusal(t *testing.T, mutate func(*CaptureRequest), store func(t *testing.T, root string)) error {
	t.Helper()
	storeRoot := fixtureStore(t)
	if store != nil {
		store(t, storeRoot)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	if mutate != nil {
		mutate(&request)
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	if result != nil {
		t.Fatalf("Capture() returned a result with error %v", err)
	}
	if !errors.Is(err, ErrInvalid) && !errors.Is(err, clonebundle.ErrInvalid) {
		t.Fatalf("Capture() error = %v, want ErrInvalid or the landed refusal", err)
	}
	return err
}

func requireRefusalDetail(t *testing.T, err error, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("refusal %q misses %q", err.Error(), fragment)
		}
	}
}

func TestCaptureRefusesUnplannedMember(t *testing.T) {
	err := captureRefusal(t, nil, func(t *testing.T, root string) {
		writeStoreFile(t, root, "store/zz-extra", []byte("unplanned-bytes"))
	})
	requireRefusalDetail(t, err, `"store/zz-extra"`, "not a plan candidate")
}

func TestCaptureRefusesPrefixSibling(t *testing.T) {
	// An unplanned key that merely shares a prefix with a planned
	// key is still unplanned: the lookup is exact.
	err := captureRefusal(t, nil, func(t *testing.T, root string) {
		writeStoreFile(t, root, "store/blob-a-evil", []byte("prefix-sibling-bytes"))
	})
	requireRefusalDetail(t, err, `"store/blob-a-evil"`, "not a plan candidate")
}

func TestCaptureRefusesUnplannedSpecialMember(t *testing.T) {
	// An UNPLANNED special member — a symlink to an outside file —
	// refuses as "not a plan candidate" with the member named,
	// exactly like an unplanned regular file: the plan gate runs
	// before any containment open, and no outside byte is read.
	storeRoot := fixtureStore(t)
	outside := t.TempDir()
	secret := "snap-secret-outside-special-6d2b"
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(storeRoot, "stray-link")); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want an unplanned-member refusal", result)
	}
	requireRefusalDetail(t, err, `"stray-link"`, "not a plan candidate")
	if blobs := listStoredBlobs(t, storeSink); blobsContain(blobs, secret) {
		t.Fatal("capture installed bytes from outside the store root")
	}
	if blobs := listStoredBlobs(t, storeSink); len(blobs) != 0 {
		t.Fatalf("refused capture installed %d blobs, want none", len(blobs))
	}
}

func TestCaptureRefusesRequiredMissing(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.Plan = []PlanItem{
			{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
			{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
			{NativeKey: "store/must-have", Class: "durable_payload", Required: true},
			{NativeKey: "store/token-cache", Class: "credential", Required: false},
		}
	}, nil)
	requireRefusalDetail(t, err, `"store/must-have"`, "required but absent")
}

func TestCaptureRefusesDuplicatePlanKey(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.Plan = []PlanItem{
			{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
			{NativeKey: "store/blob-a", Class: "durable_sidecar", Required: true},
			{NativeKey: "store/token-cache", Class: "credential", Required: false},
		}
	}, nil)
	requireRefusalDetail(t, err, "capture plan keys are not sorted unique")
	// An unsorted plan still refuses after the duplicate-only
	// narrowing: the second case pins the other half.
	unsorted := captureRefusal(t, func(request *CaptureRequest) {
		request.Plan = []PlanItem{
			{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
			{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		}
	}, nil)
	requireRefusalDetail(t, unsorted, "capture plan keys are not sorted unique")
}

func TestCaptureRefusesUnsanitizedPlanKey(t *testing.T) {
	// A UUIDv7 native key is indistinguishable from a fabricated AX
	// Session identity: the sanitizer refuses it, and nothing is
	// installed before the refusal.
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: fixtureUUIDKey, Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, fixtureUUIDKey, "UUIDv7")
	if blobs := listStoredBlobs(t, storeSink); len(blobs) != 0 {
		t.Fatalf("refused capture installed %d blobs, want none", len(blobs))
	}
}

func TestCaptureRefusesAbsolutePlanKey(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.Plan = []PlanItem{
			{NativeKey: "/abs-plan-key", Class: "durable_payload", Required: true},
		}
	}, nil)
	requireRefusalDetail(t, err, "absolute source path")
}

func TestCaptureRefusesParentPlanKey(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.Plan = []PlanItem{
			{NativeKey: "store/../escape", Class: "durable_payload", Required: true},
		}
	}, nil)
	requireRefusalDetail(t, err, `"store/../escape"`, "escapes the store root")
}

func TestCaptureRefusesMysteryPlanClass(t *testing.T) {
	storeRoot := fixtureStore(t)
	writeStoreFile(t, storeRoot, "store/blob-c", []byte("mystery-class-bytes"))
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-c", Class: "mystery", Required: true},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, `"mystery"`, "nine-class vocabulary")
	if blobs := listStoredBlobs(t, storeSink); len(blobs) != 0 {
		t.Fatalf("refused capture installed %d blobs, want none", len(blobs))
	}
}

func TestCaptureRefusesDirectoryMember(t *testing.T) {
	storeRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(storeRoot, "store"), 0o700); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "store", Class: "durable_payload", Required: true},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, `"store"`)
}

func TestCaptureRefusesSymlinkedIntermediate(t *testing.T) {
	storeRoot := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "file"), []byte("outside-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(storeRoot, "link-escape")); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "link-escape/file", Class: "durable_payload", Required: true},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, "link-escape/file", "symlink")
	if blobs := listStoredBlobs(t, storeSink); blobsContain(blobs, "outside-bytes") {
		t.Fatal("capture installed bytes from outside the store root")
	}
}

func TestCaptureRefusesOptionalSymlinkedIntermediate(t *testing.T) {
	// An OPTIONAL member behind a symlinked intermediate still
	// refuses: the blocked ancestor routes the member to its Guard
	// open, which names the full member — it never seals as
	// plan_optional_absent.
	storeRoot := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "file"), []byte("outside-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(storeRoot, "link-escape")); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "link-escape/file", Class: "durable_payload", Required: false},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, "link-escape/file", "symlink")
	if blobs := listStoredBlobs(t, storeSink); blobsContain(blobs, "outside-bytes") {
		t.Fatal("capture installed bytes from outside the store root")
	}
}

func TestCaptureRefusesTrailingSymlink(t *testing.T) {
	storeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(storeRoot, "real-target"), []byte("real-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real-target", filepath.Join(storeRoot, "link-escape")); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "link-escape", Class: "durable_payload", Required: true},
		{NativeKey: "real-target", Class: "durable_payload", Required: true},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, "link-escape", "symlink")
}

func TestCaptureRefusesSymlinkedStoreRoot(t *testing.T) {
	// A symlinked STORE root refuses before anything is opened or
	// installed: the root handle never follows a trailing link,
	// and the refusal names the link.
	storeRoot := fixtureStore(t)
	link := filepath.Join(t.TempDir(), "link-root")
	if err := os.Symlink(storeRoot, link); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, link, storeSink)
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a store-root refusal", result)
	}
	requireRefusalDetail(t, err, "link-root", "symlink")
	if blobs := listStoredBlobs(t, storeSink); len(blobs) != 0 {
		t.Fatalf("refused capture installed %d blobs, want none", len(blobs))
	}
}

func TestCaptureRefusesInvalidSourceIdentity(t *testing.T) {
	// P3-η at this leaf's entry: Capture re-validates
	// SourceIdentity through the landed IdentityDigest before
	// anything seals — an identity decoding would refuse fails
	// here, and no manifest is published.
	refuseIdentity := func(t *testing.T, mutate func(*clonebundle.NativeIdentity)) error {
		t.Helper()
		storeRoot := fixtureStore(t)
		storeSink := openTestStore(t)
		request := validCaptureRequest(t, storeRoot, storeSink)
		mutate(&request.SourceIdentity)
		result, err := Capture(request)
		if err == nil {
			t.Fatalf("Capture() = %+v, want an identity refusal", result)
		}
		if result != nil {
			t.Fatalf("Capture() returned a result with error %v", err)
		}
		for _, blob := range listStoredBlobs(t, storeSink) {
			if _, err := clonebundle.DecodeRawObjectManifest(blob.bytes); err == nil {
				t.Fatal("refused identity sealed a raw manifest")
			}
			if _, err := clonebundle.DecodeCaptureManifest(blob.bytes); err == nil {
				t.Fatal("refused identity sealed a capture manifest")
			}
		}
		return err
	}
	overlong := strings.Repeat("n", 513)
	requireRefusalDetail(t, refuseIdentity(t, func(identity *clonebundle.NativeIdentity) {
		identity.NativeSessionID = overlong
	}), "native identity digest seals no undecodable identity")
	requireRefusalDetail(t, refuseIdentity(t, func(identity *clonebundle.NativeIdentity) {
		identity.IdentityKind = "bogus"
	}), "native identity digest seals no undecodable identity")
	requireRefusalDetail(t, refuseIdentity(t, func(identity *clonebundle.NativeIdentity) {
		identity.LogicalWorkspaceID = scalar.UUIDv7{}
	}), "native identity digest seals no undecodable identity")
	requireRefusalDetail(t, refuseIdentity(t, func(identity *clonebundle.NativeIdentity) {
		identity.NativeSessionID = "native\xff"
	}), "native key is not valid UTF-8")
	requireRefusalDetail(t, refuseIdentity(t, func(identity *clonebundle.NativeIdentity) {
		identity.NativeSessionID = "/abs/native"
	}), "absolute source path")
}

func TestCaptureRefusesBadTuple(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.SourceEnvironment = []byte(`{"environment_id":"snap.env"}`)
	}, nil)
	requireRefusalDetail(t, err, "Environment Tuple")
}

func TestCaptureRefusesMixedSourceBasis(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.SourceBasis.ExternalSourceRef = "also-native"
	}, nil)
	requireRefusalDetail(t, err, "external source reference")
}

func TestCaptureRefusesUnknownBasisKind(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.SourceBasis.Kind = "maybe"
	}, nil)
	requireRefusalDetail(t, err, "ax_session|external_native")
}

func TestCaptureRefusesUnsanitizedExternalRef(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.SourceBasis = clonebundle.SourceBasisInput{
			Kind:              "external_native",
			ExternalSourceRef: "/absolute-source-ref",
		}
	}, nil)
	requireRefusalDetail(t, err, "absolute source path")
}

func TestCaptureRefusesUnknownProofKind(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.ProofKind = "maybe_stable"
	}, nil)
	requireRefusalDetail(t, err, "closed proof vocabulary")
}

func TestCaptureRefusesIdleCouplingViolation(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.InputBlocked = false
	}, nil)
	requireRefusalDetail(t, err, "input_blocked")
}

func TestCaptureRefusesSnapshotIdentityCoupling(t *testing.T) {
	identity := fixtureDigest("snap-snapshot-identity")
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.SnapshotIdentity = &identity
	}, nil)
	requireRefusalDetail(t, err, "null snapshot identity")
	missing := captureRefusal(t, func(request *CaptureRequest) {
		request.ProofKind = "immutable_snapshot"
	}, nil)
	requireRefusalDetail(t, missing, "non-null snapshot identity")
}

func TestCaptureRefusesEmptyGeneration(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.Generation = ""
	}, nil)
	requireRefusalDetail(t, err, "string[1..512]")
}

func TestCaptureRefusesBadPlanDigest(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.CapturePlanDigest = "not-a-digest"
	}, nil)
	requireRefusalDetail(t, err, "capture_plan_digest")
}

func TestCaptureRefusesBadRacePolicy(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.OnRace = RacePolicy("maybe")
	}, nil)
	requireRefusalDetail(t, err, "refuse|archive")
}

func TestCaptureRefusesNilStore(t *testing.T) {
	storeRoot := fixtureStore(t)
	request := validCaptureRequest(t, storeRoot, openTestStore(t))
	request.Store = nil
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, "object store")
}

func TestCaptureRefusesHookError(t *testing.T) {
	err := captureRefusal(t, func(request *CaptureRequest) {
		request.Hooks.AfterWalk = func(string) error { return errHookRefused }
	}, nil)
	requireRefusalDetail(t, err, "hook refused")
}

func TestCaptureRefusesOversizedMember(t *testing.T) {
	refuseSized := func(t *testing.T, payload string) error {
		t.Helper()
		storeRoot := t.TempDir()
		writeStoreFile(t, storeRoot, "store/only", []byte(payload))
		storeSink := openTestStore(t)
		request := validCaptureRequest(t, storeRoot, storeSink)
		request.MaxSingleBytes = 10
		request.Plan = []PlanItem{{NativeKey: "store/only", Class: "durable_payload", Required: true}}
		result, err := Capture(request)
		if err == nil {
			t.Fatalf("Capture() = %+v, want a refusal", result)
		}
		return err
	}
	// An 11-byte member refuses under a 10-byte bound.
	requireRefusalDetail(t, refuseSized(t, "0123456789a"), `"store/only"`, "capability_unavailable")
	// A 12-byte member still refuses after the admit-exactly-11
	// narrowing: the second case pins the other half.
	requireRefusalDetail(t, refuseSized(t, "0123456789ab"), `"store/only"`, "capability_unavailable")
}

func TestCaptureRefusesSparseOversizedMember(t *testing.T) {
	// A 128 GiB + 1 sparse member refuses with
	// capability_unavailable under the default bound without
	// reading its bytes and without publishing any manifest.
	storeRoot := fixtureStore(t)
	sparse := filepath.Join(storeRoot, "store", "sparse-huge")
	file, err := os.OpenFile(sparse, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(137438953472 + 1); err != nil {
		_ = file.Close()
		t.Skipf("sparse truncate: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/sparse-huge", Class: "durable_payload", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, `"store/sparse-huge"`, "capability_unavailable")
	for _, blob := range listStoredBlobs(t, storeSink) {
		if _, err := clonebundle.DecodeRawObjectManifest(blob.bytes); err == nil {
			t.Fatal("oversized capture published a raw manifest")
		}
		if _, err := clonebundle.DecodeCaptureManifest(blob.bytes); err == nil {
			t.Fatal("oversized capture published a capture manifest")
		}
	}
}
