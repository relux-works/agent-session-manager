package clonesnap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// Source-race evidence: a changed byte, a changed size, an appended
// record, and a replaced file each refuse — or seal the
// unstable_archive form with source_not_quiescent — and the mutation
// check runs before any seal or projection output. The AfterWalk
// hook injects each mutation deterministically between the capture
// walk and the post-capture measurement.

func mutateAfterWalk(t *testing.T, mutate func(root string)) func(string) error {
	t.Helper()
	return func(root string) error {
		mutate(root)
		return nil
	}
}

func raceRefusal(t *testing.T, mutate func(root string)) error {
	t.Helper()
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Hooks.AfterWalk = mutateAfterWalk(t, mutate)
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a race refusal", result)
	}
	requireRefusalDetail(t, err, "source mutated during capture")
	return err
}

func TestCaptureRaceChangedByte(t *testing.T) {
	// The first case mutates blob-a at equal size: size equality is
	// never proof, so the content digests still disagree.
	err := raceRefusal(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
	})
	requireRefusalDetail(t, err, `"store/blob-a"`, "capture digests differ")
	// The second case mutates blob-b: other members still refuse
	// after the admit-exactly-blob-a narrowing.
	other := raceRefusal(t, func(root string) {
		writeStoreFile(t, root, "store/blob-b", []byte("beta-payload-bytez"))
	})
	requireRefusalDetail(t, other, `"store/blob-b"`, "capture digests differ")
}

func TestCaptureRaceChangedSize(t *testing.T) {
	err := raceRefusal(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytes-extended"))
	})
	requireRefusalDetail(t, err, `"store/blob-a"`)
}

func TestCaptureRaceExcludedSizeChange(t *testing.T) {
	// An EXCLUDED member's size change still races: excluded
	// files contribute key and size to the source record, so the
	// digests disagree and the refusal names the member. (An
	// equal-size content change inside an excluded member stays
	// below the detection floor: excluded bytes are never opened.)
	err := raceRefusal(t, func(root string) {
		writeStoreFile(t, root, "store/token-cache", []byte("credential-secret-bytes-extended"))
	})
	requireRefusalDetail(t, err, `"store/token-cache"`)
}

func TestCaptureRaceAppendedRecord(t *testing.T) {
	err := raceRefusal(t, func(root string) {
		writeStoreFile(t, root, "store/blob-appended", []byte("appended-bytes"))
	})
	requireRefusalDetail(t, err, `"store/blob-appended"`)
}

func TestCaptureRaceReplacedFile(t *testing.T) {
	err := raceRefusal(t, func(root string) {
		if err := os.Remove(filepath.Join(root, "store", "blob-a")); err != nil {
			t.Fatal(err)
		}
		writeStoreFile(t, root, "store/blob-a", []byte("replacement-bytes"))
	})
	requireRefusalDetail(t, err, `"store/blob-a"`)
}

func TestCaptureRaceUnreadableAtPost(t *testing.T) {
	// A member that grows past MaxSingleBytes between the walk
	// and the post measurement keeps its shape — the same
	// regular key — but cannot be re-read: the race refuses
	// naming the member instead of sealing the captured hash.
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.MaxSingleBytes = 32
	request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", bytes.Repeat([]byte("g"), 64))
	})
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a race refusal", result)
	}
	requireRefusalDetail(t, err, "source mutated during capture", `"store/blob-a"`, "capture digests differ")
}

func TestCaptureRaceRemovedFile(t *testing.T) {
	err := raceRefusal(t, func(root string) {
		if err := os.Remove(filepath.Join(root, "store", "blob-b")); err != nil {
			t.Fatal(err)
		}
	})
	requireRefusalDetail(t, err, `"store/blob-b"`)
}

func TestCaptureArchiveSealsUnstable(t *testing.T) {
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.OnRace = RaceArchive
	request.OperatorExplicit = true
	request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
	})
	result := mustCapture(t, request)
	if result.BoundaryKind != "unstable_archive" {
		t.Fatalf("BoundaryKind = %q, want unstable_archive", result.BoundaryKind)
	}
	if result.PreDigest == result.PostDigest {
		t.Fatal("PreDigest == PostDigest despite the injected mutation")
	}
	manifest, err := clonebundle.DecodeCaptureManifest(result.CaptureManifest)
	if err != nil {
		t.Fatalf("DecodeCaptureManifest() error = %v", err)
	}
	if manifest.CaptureBoundary.Stable() {
		t.Fatal("sealed boundary is stable after a detected mutation")
	}
	sealed := string(result.CaptureManifest)
	if !strings.Contains(sealed, `"kind":"unstable_archive"`) {
		t.Fatal("sealed manifest misses the unstable_archive kind")
	}
	if !strings.Contains(sealed, `"reason_code":"source_not_quiescent"`) {
		t.Fatal("sealed manifest misses reason_code source_not_quiescent")
	}
	if !strings.Contains(sealed, `"operator_explicit":true`) {
		t.Fatal("sealed manifest misses operator_explicit true")
	}
	if !strings.Contains(sealed, `"target_projection_forbidden":true`) {
		t.Fatal("sealed manifest misses target_projection_forbidden true")
	}
	// The form exists to carry the measured evidence: the sealed
	// pre/post digests equal the pipeline's measured digests (not
	// the pre digest twice), and the sealed generation equals the
	// request generation.
	if manifest.CaptureBoundary.PreCaptureDigest == nil ||
		manifest.CaptureBoundary.PreCaptureDigest.String() != result.PreDigest.String() {
		t.Fatalf("sealed pre_capture_digest = %v, measured pre = %q",
			manifest.CaptureBoundary.PreCaptureDigest, result.PreDigest.String())
	}
	if manifest.CaptureBoundary.PostCaptureDigest == nil ||
		manifest.CaptureBoundary.PostCaptureDigest.String() != result.PostDigest.String() {
		t.Fatalf("sealed post_capture_digest = %v, measured post = %q",
			manifest.CaptureBoundary.PostCaptureDigest, result.PostDigest.String())
	}
	if manifest.CaptureBoundary.SourceGeneration == nil ||
		manifest.CaptureBoundary.SourceGeneration.String() != request.Generation {
		t.Fatalf("sealed source_generation = %v, request generation = %q",
			manifest.CaptureBoundary.SourceGeneration, request.Generation)
	}
	if err := clonebundle.RefuseUnstableForTarget(manifest.CaptureBoundary); err == nil {
		t.Fatal("RefuseUnstableForTarget() = nil for archive output")
	}
}

func TestCaptureArchiveRequiresExplicit(t *testing.T) {
	archive := func(t *testing.T, generation string) error {
		t.Helper()
		storeRoot := fixtureStore(t)
		storeSink := openTestStore(t)
		request := validCaptureRequest(t, storeRoot, storeSink)
		request.OnRace = RaceArchive
		request.OperatorExplicit = false
		request.Generation = generation
		request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
			writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
		})
		result, err := Capture(request)
		if err == nil {
			t.Fatalf("Capture() = %+v, want an acknowledgement refusal", result)
		}
		return err
	}
	requireRefusalDetail(t, archive(t, "generation-9"), "explicit operator acknowledgement")
	// A different generation still refuses after the
	// admit-exactly-generation-9 narrowing.
	requireRefusalDetail(t, archive(t, "generation-10"), "explicit operator acknowledgement")
}

func TestCaptureRaceSealsNoManifest(t *testing.T) {
	// The race gate precedes the seal: a mutated source leaves no
	// manifest blob behind, only the orphaned payload prefix.
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
	})
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a race refusal", result)
	}
	if result != nil {
		t.Fatalf("Capture() returned a result with error %v", err)
	}
	for _, blob := range listStoredBlobs(t, storeSink) {
		if _, err := clonebundle.DecodeRawObjectManifest(blob.bytes); err == nil {
			t.Fatal("mutated capture sealed a raw manifest before the race gate")
		}
		if _, err := clonebundle.DecodeCaptureManifest(blob.bytes); err == nil {
			t.Fatal("mutated capture sealed a capture manifest before the race gate")
		}
	}
}

func TestCaptureRaceEmitsNoReceipt(t *testing.T) {
	// The race gate precedes projection output: a mutated source
	// is refused BY the gate — with the race refusal, not a seal
	// error from a seal that ran too early — and leaves the
	// target sink empty. The gate also precedes any seal at this
	// entry: the capture store holds no sealed manifest either.
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	targetSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
	})
	projected, err := CaptureAndProject(ProjectRequest{
		Capture:         request,
		TargetSink:      targetSink,
		FidelityProfile: "strict_exact",
	})
	if err == nil {
		t.Fatalf("CaptureAndProject() = %+v, want a race refusal", projected)
	}
	requireRefusalDetail(t, err, "source mutated during capture", `"store/blob-a"`)
	if blobs := listStoredBlobs(t, targetSink); len(blobs) != 0 {
		t.Fatalf("mutated capture emitted %d receipt blobs before the race gate", len(blobs))
	}
	for _, blob := range listStoredBlobs(t, storeSink) {
		if _, err := clonebundle.DecodeRawObjectManifest(blob.bytes); err == nil {
			t.Fatal("mutated capture sealed a raw manifest before the race gate")
		}
		if _, err := clonebundle.DecodeCaptureManifest(blob.bytes); err == nil {
			t.Fatal("mutated capture sealed a capture manifest before the race gate")
		}
	}
}

func TestCaptureHookFires(t *testing.T) {
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	fired := false
	request.Hooks.AfterRawPublish = func() error {
		fired = true
		return nil
	}
	mustCapture(t, request)
	if !fired {
		t.Fatal("AfterRawPublish hook did not fire")
	}
}
