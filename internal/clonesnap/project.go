package clonesnap

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file owns target-branch admission and the projection output
// whose existence pins the race-before-projection ordering.
// CaptureAndProject runs the capture pipeline, then admits the
// sealed capture manifest: the unstable_archive form can never enter
// a target branch (the landed G2 gate decides), and the maximal_safe
// fidelity profile requires a reconciled complete manifest (the
// landed maximal-safe gate decides). Admission emits one admission
// receipt into the target sink. Projection planning past admission
// is the final leaf's scope; this receipt exists to prove admission
// and ordering, not to plan a projection.

// FidelityProfile is one Section 7.8 fidelity profile:
// strict_exact|maximal_safe|compact|messages_only.
type FidelityProfile string

const (
	// FidelityStrictExact requests exact projection.
	FidelityStrictExact FidelityProfile = "strict_exact"
	// FidelityMaximalSafe requests maximal safe projection: it
	// requires a reconciled complete manifest with no unknown class.
	FidelityMaximalSafe FidelityProfile = "maximal_safe"
	// FidelityCompact requests compact projection.
	FidelityCompact FidelityProfile = "compact"
	// FidelityMessagesOnly requests messages-only projection.
	FidelityMessagesOnly FidelityProfile = "messages_only"
)

// validFidelityProfile reports whether the profile is in the Section
// 7.8 closed vocabulary. Bound (P3-ζ′): sessadapter owns an
// unexported copy of this four-profile table (operations.go
// fidelityProfiles, pinned by TestValueVocabulariesMatchSpec) and
// this package keeps its own closed switch (pinned by
// TestProjectAdmitsStableEveryProfile and
// TestProjectRefusesUnknownProfile). The two spellings are not
// unified: unification is deferred to a follow-up on
// internal/sessadapter and internal/clonesnap.
func validFidelityProfile(profile string) bool {
	switch FidelityProfile(profile) {
	case FidelityStrictExact, FidelityMaximalSafe, FidelityCompact, FidelityMessagesOnly:
		return true
	default:
		return false
	}
}

// ProjectRequest is one capture-and-project candidate: the capture
// request, the target sink that receives the admission receipt, and
// the fidelity profile admission decides.
type ProjectRequest struct {
	Capture CaptureRequest
	// TargetSink receives the admission receipt. It stays empty
	// when capture refuses, races, or seals archive-only output.
	TargetSink *localstore.ObjectStore
	// FidelityProfile is the requested Section 7.8 profile.
	FidelityProfile string
}

// ProjectResult is one admitted capture: the capture result plus the
// sealed admission receipt.
type ProjectResult struct {
	Capture *CaptureResult
	// Receipt carries the sealed admission receipt bytes.
	Receipt []byte
	// ReceiptDigest is the blob identity of the receipt.
	ReceiptDigest scalar.Digest
	// ReceiptInstalled reports whether this run installed the
	// receipt blob (a replay verifies and reuses).
	ReceiptInstalled bool
}

// CaptureAndProject runs capture, then admits the sealed capture to
// a target branch and emits the admission receipt. The race gate
// precedes admission and emission (see ORDER-PIN below): a mutated
// source refuses, or seals archive-only output that admission
// rejects — no projection output is produced before the race is
// decided.
func CaptureAndProject(request ProjectRequest) (*ProjectResult, error) {
	if request.TargetSink == nil {
		return nil, invalid("capture projection requires a target sink")
	}
	state, err := newPipeline(request.Capture)
	if err != nil {
		return nil, err
	}
	defer state.close()
	if err := state.walkAndMeasure(); err != nil {
		return nil, err
	}
	// ORDER-PIN: the source-race gate precedes any seal, admission,
	// or projection output. N-race-order-project moves this gate
	// after admitAndEmit: the early seal then refuses with a seal
	// error instead of the race refusal, so that row's killer
	// reddens on the wrong error (the seal error message, not a
	// sink). N-race-order-project-seal keeps the race error while
	// sealing first: its killer reddens on the sealed raw manifest
	// in the capture store.
	if err := state.gateSourceRace(); err != nil {
		captured, raceErr := state.raceOutcome(err)
		if raceErr != nil {
			return nil, raceErr
		}
		// Archive-only output never enters a target branch: the
		// admission gate below refuses it, so no receipt exists.
		if _, err := AdmitForTarget(captured.CaptureManifest, request.FidelityProfile); err != nil {
			return nil, err
		}
		return nil, invalid("capture archive unexpectedly admitted to a target branch")
	}
	sealed, err := state.sealAndPublish(false)
	if err != nil {
		return nil, err
	}
	return state.admitAndEmit(request, sealed)
}

// admitAndEmit admits one sealed capture to the target branch and
// emits its admission receipt into the target sink.
func (state *pipeline) admitAndEmit(request ProjectRequest, sealed *CaptureResult) (*ProjectResult, error) {
	if _, err := AdmitForTarget(sealed.CaptureManifest, request.FidelityProfile); err != nil {
		return nil, err
	}
	receipt, err := BuildAdmissionReceipt(sealed.Raw.ManifestID, sealed.Capture.ManifestID, request.FidelityProfile)
	if err != nil {
		return nil, err
	}
	digest := scalar.SHA256Digest(receipt)
	put, err := request.TargetSink.PutBlob(digest, uint64(len(receipt)), bytes.NewReader(receipt))
	if err != nil {
		return nil, invalid("capture projection cannot publish the admission receipt: %v", err)
	}
	return &ProjectResult{
		Capture:          sealed,
		Receipt:          receipt,
		ReceiptDigest:    digest,
		ReceiptInstalled: put.Installed,
	}, nil
}

// AdmitForTarget admits one sealed capture manifest to a target
// branch under the requested fidelity profile. The unstable_archive
// form is archive-only output and always refuses; an unknown
// fidelity profile refuses; maximal_safe requires a reconciled
// complete manifest with no unknown class.
func AdmitForTarget(captureManifest []byte, profile string) (clonebundle.CaptureManifest, error) {
	manifest, err := clonebundle.DecodeCaptureManifest(captureManifest)
	if err != nil {
		return clonebundle.CaptureManifest{}, err
	}
	if err := clonebundle.RefuseUnstableForTarget(manifest.CaptureBoundary); err != nil {
		return clonebundle.CaptureManifest{}, err
	}
	if !validFidelityProfile(profile) {
		return clonebundle.CaptureManifest{}, invalid("capture fidelity profile %q is outside strict_exact|maximal_safe|compact|messages_only", profile)
	}
	if profile == "maximal_safe" {
		if err := clonebundle.RefuseMaximalSafeUnlessComplete(manifest); err != nil {
			return clonebundle.CaptureManifest{}, err
		}
	}
	return manifest, nil
}

// admissionReceiptSchema pins the admission receipt envelope
// identity.
const admissionReceiptSchema = "urn:ax:internal:clonesnap-admission-receipt"

// BuildAdmissionReceipt seals one target admission receipt: the
// admitted raw and capture manifest identities, the stable boundary
// kind, and the fidelity profile. Only stable captures reach it;
// AdmitForTarget refused everything else before this seal ran.
func BuildAdmissionReceipt(rawManifestID, captureManifestID scalar.Digest, profile string) ([]byte, error) {
	if !validFidelityProfile(profile) {
		return nil, invalid("capture fidelity profile %q is outside strict_exact|maximal_safe|compact|messages_only", profile)
	}
	object := map[string]any{
		"schema":                 admissionReceiptSchema,
		"schema_version":         "1.0.0",
		"raw_object_manifest_id": rawManifestID.String(),
		"capture_manifest_id":    captureManifestID.String(),
		"boundary_kind":          "stable",
		"fidelity_profile":       profile,
		"extensions":             map[string]any{},
	}
	plain, err := json.Marshal(object)
	if err != nil {
		return nil, invalid("serialize admission receipt: %v", err)
	}
	sealed, err := canonicaljson.Canonicalize(plain)
	if err != nil {
		return nil, invalid("canonicalize admission receipt: %v", err)
	}
	return sealed, nil
}
