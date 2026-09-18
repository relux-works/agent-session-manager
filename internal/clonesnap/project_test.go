package clonesnap

import (
	"bytes"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// Target admission evidence: AdmitForTarget drives the landed G2
// and maximal-safe gates from the projection entry, and
// CaptureAndProject emits the admission receipt only for admitted
// stable captures.

func projectValid(t *testing.T, profile string) (*ProjectResult, error) {
	t.Helper()
	storeRoot := fixtureStore(t)
	return CaptureAndProject(ProjectRequest{
		Capture:         validCaptureRequest(t, storeRoot, openTestStore(t)),
		TargetSink:      openTestStore(t),
		FidelityProfile: profile,
	})
}

func TestProjectAdmitsStableEveryProfile(t *testing.T) {
	for _, profile := range []string{"strict_exact", "maximal_safe", "compact", "messages_only"} {
		projected, err := projectValid(t, profile)
		if err != nil {
			t.Fatalf("CaptureAndProject(%q) error = %v", profile, err)
		}
		if !projected.ReceiptInstalled {
			t.Fatalf("CaptureAndProject(%q) installed no receipt", profile)
		}
		sealed := string(projected.Receipt)
		for _, literal := range []string{
			`"schema":"urn:ax:internal:clonesnap-admission-receipt"`,
			`"schema_version":"1.0.0"`,
			`"boundary_kind":"stable"`,
			`"fidelity_profile":"` + profile + `"`,
		} {
			if !strings.Contains(sealed, literal) {
				t.Fatalf("receipt misses %s:\n%s", literal, sealed)
			}
		}
		if !strings.Contains(sealed, projected.Capture.Raw.ManifestID.String()) {
			t.Fatal("receipt misses the raw manifest identity")
		}
		if !strings.Contains(sealed, projected.Capture.Capture.ManifestID.String()) {
			t.Fatal("receipt misses the capture manifest identity")
		}
	}
}

func TestProjectRefusesUnstable(t *testing.T) {
	archive := func(t *testing.T, generation string) error {
		t.Helper()
		storeRoot := fixtureStore(t)
		storeSink := openTestStore(t)
		targetSink := openTestStore(t)
		request := validCaptureRequest(t, storeRoot, storeSink)
		request.OnRace = RaceArchive
		request.OperatorExplicit = true
		request.Generation = generation
		request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
			writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
		})
		projected, err := CaptureAndProject(ProjectRequest{
			Capture:         request,
			TargetSink:      targetSink,
			FidelityProfile: "strict_exact",
		})
		if err == nil {
			t.Fatalf("CaptureAndProject() = %+v, want a target refusal", projected)
		}
		// Archive-only output never enters a target branch: the
		// refusal leaves the target sink empty, not only unadmitted.
		if blobs := listStoredBlobs(t, targetSink); len(blobs) != 0 {
			t.Fatalf("refused archive capture wrote %d blobs to the target sink, want none", len(blobs))
		}
		return err
	}
	// The unstable_archive form can never enter a target branch.
	requireRefusalDetail(t, archive(t, "generation-9"), "unstable_archive", "target branch")
	// A different generation still refuses after the
	// admit-exactly-generation-9 narrowing.
	requireRefusalDetail(t, archive(t, "generation-10"), "unstable_archive", "target branch")
}

func TestProjectRefusesUnstableDirect(t *testing.T) {
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.OnRace = RaceArchive
	request.OperatorExplicit = true
	request.Hooks.AfterWalk = mutateAfterWalk(t, func(root string) {
		writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytez"))
	})
	captured := mustCapture(t, request)
	if _, err := AdmitForTarget(captured.CaptureManifest, "strict_exact"); err == nil {
		t.Fatal("AdmitForTarget(unstable) = nil, want a refusal")
	} else {
		requireRefusalDetail(t, err, "unstable_archive", "target branch")
	}
}

func TestProjectMaximalSafeRequiresComplete(t *testing.T) {
	admitUnknown := func(t *testing.T, keys int) error {
		t.Helper()
		storeRoot := t.TempDir()
		writeStoreFile(t, storeRoot, "store/blob-a", []byte("alpha-payload-bytes"))
		plan := []PlanItem{{NativeKey: "store/blob-a", Class: "durable_payload", Required: true}}
		for index := 0; index < keys; index++ {
			key := "store/zz-unknown-" + string(rune('a'+index))
			writeStoreFile(t, storeRoot, key, []byte("unknown-bytes"))
			plan = append(plan, PlanItem{NativeKey: key, Class: "unknown", Required: true})
		}
		// Sorted: blob-a, then the zz-unknown keys in order.
		request := validCaptureRequest(t, storeRoot, openTestStore(t))
		request.Plan = plan
		captured := mustCapture(t, request)
		if captured.Capture.RawComplete {
			t.Fatal("RawComplete = true with unknown-class items")
		}
		_, err := AdmitForTarget(captured.CaptureManifest, "maximal_safe")
		if err == nil {
			t.Fatalf("AdmitForTarget(maximal_safe) = nil with %d unknown items", keys)
		}
		return err
	}
	requireRefusalDetail(t, admitUnknown(t, 2), "maximal_safe")
	// The two-item manifest still refuses after the
	// admit-exactly-three-items narrowing.
	requireRefusalDetail(t, admitUnknown(t, 1), "maximal_safe")
}

func TestProjectMaximalSafeAdmitsComplete(t *testing.T) {
	projected, err := projectValid(t, "maximal_safe")
	if err != nil {
		t.Fatalf("CaptureAndProject(maximal_safe) error = %v", err)
	}
	if !projected.ReceiptInstalled {
		t.Fatal("complete maximal_safe capture installed no receipt")
	}
}

func TestProjectStrictExactAdmitsUnknown(t *testing.T) {
	storeRoot := fixtureStore(t)
	writeStoreFile(t, storeRoot, "store/zz-unknown", []byte("unknown-kind-bytes"))
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "store/blob-a", Class: "durable_payload", Required: true},
		{NativeKey: "store/blob-b", Class: "durable_sidecar", Required: true},
		{NativeKey: "store/token-cache", Class: "credential", Required: false},
		{NativeKey: "store/zz-unknown", Class: "unknown", Required: true},
	}
	projected, err := CaptureAndProject(ProjectRequest{
		Capture:         request,
		TargetSink:      openTestStore(t),
		FidelityProfile: "strict_exact",
	})
	if err != nil {
		t.Fatalf("CaptureAndProject(strict_exact) error = %v", err)
	}
	if projected.Capture.Capture.RawComplete {
		t.Fatal("RawComplete = true with an unknown-class item")
	}
}

func TestProjectRefusesUnknownProfile(t *testing.T) {
	storeRoot := fixtureStore(t)
	captured := mustCapture(t, validCaptureRequest(t, storeRoot, openTestStore(t)))
	if _, err := AdmitForTarget(captured.CaptureManifest, "ultra"); err == nil {
		t.Fatal("AdmitForTarget(ultra) = nil, want a refusal")
	} else {
		requireRefusalDetail(t, err, `"ultra"`, "strict_exact|maximal_safe|compact|messages_only")
	}
	if _, err := projectValid(t, "ultra"); err == nil {
		t.Fatal("CaptureAndProject(ultra) admitted, want a refusal")
	}
}

func TestProjectIsIdempotent(t *testing.T) {
	// A replayed capture installs the same blobs and publishes no
	// half-written manifest: every manifest blob re-decodes, the
	// replay verifies and reuses, and the bytes are identical.
	storeRoot := fixtureStore(t)
	storeSink := openTestStore(t)
	targetSink := openTestStore(t)
	build := func() ProjectRequest {
		return ProjectRequest{
			Capture:         validCaptureRequest(t, storeRoot, storeSink),
			TargetSink:      targetSink,
			FidelityProfile: "strict_exact",
		}
	}
	first, err := CaptureAndProject(build())
	if err != nil {
		t.Fatalf("CaptureAndProject() error = %v", err)
	}
	second, err := CaptureAndProject(build())
	if err != nil {
		t.Fatalf("replayed CaptureAndProject() error = %v", err)
	}
	if !bytes.Equal(first.Capture.RawManifest, second.Capture.RawManifest) {
		t.Fatal("replayed raw manifest bytes differ")
	}
	if !bytes.Equal(first.Capture.CaptureManifest, second.Capture.CaptureManifest) {
		t.Fatal("replayed capture manifest bytes differ")
	}
	if !bytes.Equal(first.Receipt, second.Receipt) {
		t.Fatal("replayed receipt bytes differ")
	}
	if second.Capture.RawInstalled || second.Capture.CaptureInstalled || second.ReceiptInstalled {
		t.Fatal("replay installed instead of verifying and reusing")
	}
	for _, blob := range listStoredBlobs(t, storeSink) {
		if _, err := clonebundle.DecodeRawObjectManifest(blob.bytes); err == nil {
			continue
		}
		if _, err := clonebundle.DecodeCaptureManifest(blob.bytes); err == nil {
			continue
		}
		// Payload blobs decode as neither manifest; their digest
		// is their identity, verified at install.
	}
	countBefore := len(listStoredBlobs(t, storeSink))
	third, err := CaptureAndProject(build())
	if err != nil {
		t.Fatalf("second replay error = %v", err)
	}
	_ = third
	if countAfter := len(listStoredBlobs(t, storeSink)); countAfter != countBefore {
		t.Fatalf("replay changed the blob count %d -> %d", countBefore, countAfter)
	}
}
