package clonebundle

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Capture Source Basis and Capture Boundary
// closed unions of Clone Capture Manifest 1.0.0, including the
// StableSnapshotProof. The stable form carries proof; the unstable
// form is core-created archive-only output that can never enter a
// target branch.

// SourceBasis is one validated Capture Source Basis.
type SourceBasis struct {
	Kind                     string
	SourceSessionID          *scalar.UUIDv7
	SourceSessionRecordID    *scalar.Digest
	SourceCheckpointID       *scalar.Digest
	SourceProviderIdentityID *scalar.Digest
	ExternalSourceRef        *string
}

var sourceBasisAXMembers = map[string]bool{
	"kind": true, "source_session_id": true, "source_session_record_id": true,
	"source_checkpoint_id": true, "source_provider_identity_record_id": true, "extensions": true,
}

var sourceBasisAXRequired = []string{
	"kind", "source_session_id", "source_session_record_id",
	"source_checkpoint_id", "source_provider_identity_record_id", "extensions",
}

var sourceBasisExternalMembers = map[string]bool{
	"kind": true, "external_source_ref": true, "extensions": true,
}

var sourceBasisExternalRequired = []string{
	"kind", "external_source_ref", "extensions",
}

// DecodeSourceBasis validates one closed Capture Source Basis.
func DecodeSourceBasis(raw json.RawMessage) (SourceBasis, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return SourceBasis{}, invalid("source basis %s (%s)", fault.detail, memberField(fault.member))
	}
	kindValue, present := members["kind"]
	if !present {
		return SourceBasis{}, invalid("source basis misses a required member %q", "kind")
	}
	kind, ok := rawString(kindValue)
	if !ok {
		return SourceBasis{}, invalid("source basis kind is not a string")
	}
	switch kind {
	case "ax_session":
		return decodeAXSessionBasis(members)
	case "external_native":
		return decodeExternalNativeBasis(members)
	default:
		return SourceBasis{}, invalid("source basis kind is outside ax_session|external_native")
	}
}

func decodeAXSessionBasis(members map[string]json.RawMessage) (SourceBasis, error) {
	if name, unknown := unknownMember(members, sourceBasisAXMembers); unknown {
		return SourceBasis{}, invalid("source basis carries unknown member %q", name)
	}
	if name, missing := missingMember(members, sourceBasisAXRequired); missing {
		return SourceBasis{}, invalid("source basis misses a required member %q", name)
	}
	sessionID, ok := checkUUIDv7(members["source_session_id"])
	if !ok {
		return SourceBasis{}, invalid("source basis source_session_id is not a UUIDv7")
	}
	recordID, ok := checkDigest(members["source_session_record_id"])
	if !ok {
		return SourceBasis{}, invalid("source basis source_session_record_id is not a digest")
	}
	checkpointID, ok := checkDigest(members["source_checkpoint_id"])
	if !ok {
		return SourceBasis{}, invalid("source basis source_checkpoint_id is not a digest")
	}
	providerID, ok := checkDigest(members["source_provider_identity_record_id"])
	if !ok {
		return SourceBasis{}, invalid("source basis source_provider_identity_record_id is not a digest")
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return SourceBasis{}, invalid("source basis extensions %s", extensionsFault)
	}
	return SourceBasis{
		Kind:                     "ax_session",
		SourceSessionID:          &sessionID,
		SourceSessionRecordID:    &recordID,
		SourceCheckpointID:       &checkpointID,
		SourceProviderIdentityID: &providerID,
	}, nil
}

func decodeExternalNativeBasis(members map[string]json.RawMessage) (SourceBasis, error) {
	if name, unknown := unknownMember(members, sourceBasisExternalMembers); unknown {
		return SourceBasis{}, invalid("source basis carries unknown member %q", name)
	}
	if name, missing := missingMember(members, sourceBasisExternalRequired); missing {
		return SourceBasis{}, invalid("source basis misses a required member %q", name)
	}
	reference, ok := checkStringBounds(members["external_source_ref"], 1, 512)
	if !ok {
		return SourceBasis{}, invalid("source basis external_source_ref is not a string[1..512]")
	}
	if err := SanitizeNativeKey(reference); err != nil {
		return SourceBasis{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return SourceBasis{}, invalid("source basis extensions %s", extensionsFault)
	}
	return SourceBasis{Kind: "external_native", ExternalSourceRef: &reference}, nil
}

// SourceBasisInput is one caller-supplied source basis candidate for
// Build. Exactly the kind-selected fields must be set.
type SourceBasisInput struct {
	Kind                     string
	SourceSessionID          string
	SourceSessionRecordID    string
	SourceCheckpointID       string
	SourceProviderIdentityID string
	ExternalSourceRef        string
}

func buildSourceBasis(input SourceBasisInput) (SourceBasis, map[string]any, error) {
	switch input.Kind {
	case "ax_session":
		sessionID, err := scalar.ParseUUIDv7(input.SourceSessionID)
		if err != nil {
			return SourceBasis{}, nil, invalid("source basis source_session_id is not a UUIDv7: %v", err)
		}
		recordID, err := scalar.ParseDigest(input.SourceSessionRecordID)
		if err != nil {
			return SourceBasis{}, nil, invalid("source basis source_session_record_id is not a digest: %v", err)
		}
		checkpointID, err := scalar.ParseDigest(input.SourceCheckpointID)
		if err != nil {
			return SourceBasis{}, nil, invalid("source basis source_checkpoint_id is not a digest: %v", err)
		}
		providerID, err := scalar.ParseDigest(input.SourceProviderIdentityID)
		if err != nil {
			return SourceBasis{}, nil, invalid("source basis source_provider_identity_record_id is not a digest: %v", err)
		}
		if input.ExternalSourceRef != "" {
			return SourceBasis{}, nil, invalid("source basis ax_session carries an external source reference")
		}
		basis := SourceBasis{
			Kind:                     "ax_session",
			SourceSessionID:          &sessionID,
			SourceSessionRecordID:    &recordID,
			SourceCheckpointID:       &checkpointID,
			SourceProviderIdentityID: &providerID,
		}
		return basis, map[string]any{
			"kind":                               "ax_session",
			"source_session_id":                  sessionID.String(),
			"source_session_record_id":           recordID.String(),
			"source_checkpoint_id":               checkpointID.String(),
			"source_provider_identity_record_id": providerID.String(),
			"extensions":                         map[string]any{},
		}, nil
	case "external_native":
		if stringLength(input.ExternalSourceRef) < 1 || stringLength(input.ExternalSourceRef) > 512 {
			return SourceBasis{}, nil, invalid("source basis external_source_ref is not a string[1..512]")
		}
		if err := SanitizeNativeKey(input.ExternalSourceRef); err != nil {
			return SourceBasis{}, nil, err
		}
		if input.SourceSessionID != "" || input.SourceSessionRecordID != "" ||
			input.SourceCheckpointID != "" || input.SourceProviderIdentityID != "" {
			return SourceBasis{}, nil, invalid("source basis external_native carries ax_session members")
		}
		basis := SourceBasis{Kind: "external_native", ExternalSourceRef: &input.ExternalSourceRef}
		return basis, map[string]any{
			"kind":                "external_native",
			"external_source_ref": input.ExternalSourceRef,
			"extensions":          map[string]any{},
		}, nil
	default:
		return SourceBasis{}, nil, invalid("source basis kind is outside ax_session|external_native")
	}
}

// StableSnapshotProof is one validated stable snapshot proof.
type StableSnapshotProof struct {
	ProofKind         string
	SourceGeneration  Generation
	SnapshotIdentity  *scalar.Digest
	PreCaptureDigest  scalar.Digest
	PostCaptureDigest scalar.Digest
	InputBlocked      bool
	ForegroundIdle    bool
	BackgroundIdle    bool
}

var proofKinds = []string{
	"closed_store",
	"immutable_snapshot",
	"verified_log_prefix",
	"provider_quiescence",
}

func validProofKind(kind string) bool {
	for _, allowed := range proofKinds {
		if kind == allowed {
			return true
		}
	}
	return false
}

var stableProofMembers = map[string]bool{
	"proof_kind": true, "source_generation": true, "snapshot_identity_digest": true,
	"pre_capture_digest": true, "post_capture_digest": true,
	"input_blocked": true, "foreground_idle": true, "background_idle": true, "extensions": true,
}

var stableProofRequired = []string{
	"proof_kind", "source_generation", "snapshot_identity_digest",
	"pre_capture_digest", "post_capture_digest",
	"input_blocked", "foreground_idle", "background_idle", "extensions",
}

// DecodeStableSnapshotProof validates one closed StableSnapshotProof:
// equal capture digests, the proof-kind/identity/boolean coupling,
// and the immutable generation token.
func DecodeStableSnapshotProof(raw json.RawMessage) (StableSnapshotProof, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return StableSnapshotProof{}, invalid("stable snapshot proof %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, stableProofMembers); unknown {
		return StableSnapshotProof{}, invalid("stable snapshot proof carries unknown member %q", name)
	}
	if name, missing := missingMember(members, stableProofRequired); missing {
		return StableSnapshotProof{}, invalid("stable snapshot proof misses a required member %q", name)
	}
	kind, ok := rawString(members["proof_kind"])
	if !ok || !validProofKind(kind) {
		return StableSnapshotProof{}, invalid("stable snapshot proof kind is outside the closed proof vocabulary")
	}
	generation, ok := checkGeneration(members["source_generation"])
	if !ok {
		return StableSnapshotProof{}, invalid("stable snapshot proof source_generation is not a string[1..512]")
	}
	var identity *scalar.Digest
	if !isNull(members["snapshot_identity_digest"]) {
		value, ok := checkDigest(members["snapshot_identity_digest"])
		if !ok {
			return StableSnapshotProof{}, invalid("stable snapshot proof snapshot_identity_digest is not a digest")
		}
		identity = &value
	}
	pre, ok := checkDigest(members["pre_capture_digest"])
	if !ok {
		return StableSnapshotProof{}, invalid("stable snapshot proof pre_capture_digest is not a digest")
	}
	post, ok := checkDigest(members["post_capture_digest"])
	if !ok {
		return StableSnapshotProof{}, invalid("stable snapshot proof post_capture_digest is not a digest")
	}
	inputBlocked, ok := rawBool(members["input_blocked"])
	if !ok {
		return StableSnapshotProof{}, invalid("stable snapshot proof input_blocked is not a boolean")
	}
	foregroundIdle, ok := rawBool(members["foreground_idle"])
	if !ok {
		return StableSnapshotProof{}, invalid("stable snapshot proof foreground_idle is not a boolean")
	}
	backgroundIdle, ok := rawBool(members["background_idle"])
	if !ok {
		return StableSnapshotProof{}, invalid("stable snapshot proof background_idle is not a boolean")
	}
	proof := StableSnapshotProof{
		ProofKind:         kind,
		SourceGeneration:  generation,
		SnapshotIdentity:  identity,
		PreCaptureDigest:  pre,
		PostCaptureDigest: post,
		InputBlocked:      inputBlocked,
		ForegroundIdle:    foregroundIdle,
		BackgroundIdle:    backgroundIdle,
	}
	if err := checkProofCoupling(proof); err != nil {
		return StableSnapshotProof{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return StableSnapshotProof{}, invalid("stable snapshot proof extensions %s", extensionsFault)
	}
	return proof, nil
}

// checkProofCoupling enforces the proof-kind/identity/boolean
// coupling: capture digests are equal (size equality is never
// proof and is not consulted); closed-store and
// provider-quiescence require all booleans true and null snapshot
// identity; immutable-snapshot and log-prefix require non-null
// identity.
func checkProofCoupling(proof StableSnapshotProof) error {
	if err := CheckDigestsEqual(proof.PreCaptureDigest, proof.PostCaptureDigest); err != nil {
		return invalid("stable snapshot proof: %v", err)
	}
	switch proof.ProofKind {
	case "closed_store", "provider_quiescence":
		if proof.SnapshotIdentity != nil {
			return invalid("stable snapshot proof %s requires null snapshot identity", proof.ProofKind)
		}
		if !proof.InputBlocked || !proof.ForegroundIdle || !proof.BackgroundIdle {
			return invalid("stable snapshot proof %s requires input_blocked, foreground_idle, and background_idle", proof.ProofKind)
		}
	case "immutable_snapshot", "verified_log_prefix":
		if proof.SnapshotIdentity == nil {
			return invalid("stable snapshot proof %s requires non-null snapshot identity", proof.ProofKind)
		}
	}
	return nil
}

// CaptureBoundary is one validated Capture Boundary.
type CaptureBoundary struct {
	Kind              string
	Proof             *StableSnapshotProof
	SourceGeneration  *Generation
	PreCaptureDigest  *scalar.Digest
	PostCaptureDigest *scalar.Digest
}

// Stable reports whether the boundary is the stable form.
func (boundary CaptureBoundary) Stable() bool { return boundary.Kind == "stable" }

var stableBoundaryMembers = map[string]bool{
	"kind": true, "proof": true, "extensions": true,
}

var stableBoundaryRequired = []string{"kind", "proof", "extensions"}

var unstableBoundaryMembers = map[string]bool{
	"kind": true, "source_generation": true, "pre_capture_digest": true,
	"post_capture_digest": true, "reason_code": true,
	"operator_explicit": true, "target_projection_forbidden": true, "extensions": true,
}

var unstableBoundaryRequired = []string{
	"kind", "source_generation", "pre_capture_digest",
	"post_capture_digest", "reason_code",
	"operator_explicit", "target_projection_forbidden", "extensions",
}

// DecodeCaptureBoundary validates one closed Capture Boundary.
func DecodeCaptureBoundary(raw json.RawMessage) (CaptureBoundary, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return CaptureBoundary{}, invalid("capture boundary %s (%s)", fault.detail, memberField(fault.member))
	}
	kindValue, present := members["kind"]
	if !present {
		return CaptureBoundary{}, invalid("capture boundary misses a required member %q", "kind")
	}
	kind, ok := rawString(kindValue)
	if !ok {
		return CaptureBoundary{}, invalid("capture boundary kind is not a string")
	}
	switch kind {
	case "stable":
		return decodeStableBoundary(members)
	case "unstable_archive":
		return decodeUnstableBoundary(members)
	default:
		return CaptureBoundary{}, invalid("capture boundary kind is outside stable|unstable_archive")
	}
}

func decodeStableBoundary(members map[string]json.RawMessage) (CaptureBoundary, error) {
	if name, unknown := unknownMember(members, stableBoundaryMembers); unknown {
		return CaptureBoundary{}, invalid("capture boundary carries unknown member %q", name)
	}
	if name, missing := missingMember(members, stableBoundaryRequired); missing {
		return CaptureBoundary{}, invalid("capture boundary misses a required member %q", name)
	}
	proof, err := DecodeStableSnapshotProof(members["proof"])
	if err != nil {
		return CaptureBoundary{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return CaptureBoundary{}, invalid("capture boundary extensions %s", extensionsFault)
	}
	return CaptureBoundary{Kind: "stable", Proof: &proof}, nil
}

func decodeUnstableBoundary(members map[string]json.RawMessage) (CaptureBoundary, error) {
	if name, unknown := unknownMember(members, unstableBoundaryMembers); unknown {
		return CaptureBoundary{}, invalid("capture boundary carries unknown member %q", name)
	}
	if name, missing := missingMember(members, unstableBoundaryRequired); missing {
		return CaptureBoundary{}, invalid("capture boundary misses a required member %q", name)
	}
	generation, ok := checkGeneration(members["source_generation"])
	if !ok {
		return CaptureBoundary{}, invalid("capture boundary source_generation is not a string[1..512]")
	}
	pre, ok := checkDigest(members["pre_capture_digest"])
	if !ok {
		return CaptureBoundary{}, invalid("capture boundary pre_capture_digest is not a digest")
	}
	post, ok := checkDigest(members["post_capture_digest"])
	if !ok {
		return CaptureBoundary{}, invalid("capture boundary post_capture_digest is not a digest")
	}
	reason, ok := rawString(members["reason_code"])
	if !ok || reason != "source_not_quiescent" {
		return CaptureBoundary{}, invalid("capture boundary reason_code is not source_not_quiescent")
	}
	explicit, ok := rawBool(members["operator_explicit"])
	if !ok || !explicit {
		return CaptureBoundary{}, invalid("capture boundary operator_explicit is not true")
	}
	forbidden, ok := rawBool(members["target_projection_forbidden"])
	if !ok || !forbidden {
		return CaptureBoundary{}, invalid("capture boundary target_projection_forbidden is not true")
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return CaptureBoundary{}, invalid("capture boundary extensions %s", extensionsFault)
	}
	return CaptureBoundary{
		Kind:              "unstable_archive",
		SourceGeneration:  &generation,
		PreCaptureDigest:  &pre,
		PostCaptureDigest: &post,
	}, nil
}

// RefuseUnstableForTarget is the G2 target-branch gate: the unstable
// archive form is core-created archive-only output and can never
// enter a target branch, so target admission refuses it.
func RefuseUnstableForTarget(boundary CaptureBoundary) error {
	if boundary.Kind == "unstable_archive" {
		return invalid("capture boundary unstable_archive cannot enter a target branch")
	}
	if boundary.Kind != "stable" {
		return invalid("capture boundary kind is outside stable|unstable_archive")
	}
	return nil
}

// BoundaryInput is one caller-supplied capture boundary candidate
// for Build. Core must be true to construct the unstable archive
// form; only core may construct it. Core is a caller assertion at
// this entry, so "only core may construct" is nominal here: the
// real protection is RefuseUnstableForTarget, which refuses every
// unstable archive at target-branch admission regardless of who
// constructed it.
type BoundaryInput struct {
	Kind              string
	ProofKind         string
	Generation        string
	SnapshotIdentity  *string
	PreCaptureDigest  string
	PostCaptureDigest string
	InputBlocked      bool
	ForegroundIdle    bool
	BackgroundIdle    bool
	Core              bool
}

func buildBoundary(input BoundaryInput) (CaptureBoundary, map[string]any, error) {
	switch input.Kind {
	case "stable":
		proof, object, err := buildStableProof(input)
		if err != nil {
			return CaptureBoundary{}, nil, err
		}
		boundary := CaptureBoundary{Kind: "stable", Proof: &proof}
		return boundary, map[string]any{
			"kind":       "stable",
			"proof":      object,
			"extensions": map[string]any{},
		}, nil
	case "unstable_archive":
		if !input.Core {
			return CaptureBoundary{}, nil, invalid("capture boundary unstable_archive is core-created only")
		}
		generation, err := ParseGeneration(input.Generation)
		if err != nil {
			return CaptureBoundary{}, nil, err
		}
		pre, err := scalar.ParseDigest(input.PreCaptureDigest)
		if err != nil {
			return CaptureBoundary{}, nil, invalid("capture boundary pre_capture_digest is not a digest: %v", err)
		}
		post, err := scalar.ParseDigest(input.PostCaptureDigest)
		if err != nil {
			return CaptureBoundary{}, nil, invalid("capture boundary post_capture_digest is not a digest: %v", err)
		}
		boundary := CaptureBoundary{
			Kind:              "unstable_archive",
			SourceGeneration:  &generation,
			PreCaptureDigest:  &pre,
			PostCaptureDigest: &post,
		}
		return boundary, map[string]any{
			"kind":                        "unstable_archive",
			"source_generation":           generation.String(),
			"pre_capture_digest":          pre.String(),
			"post_capture_digest":         post.String(),
			"reason_code":                 "source_not_quiescent",
			"operator_explicit":           true,
			"target_projection_forbidden": true,
			"extensions":                  map[string]any{},
		}, nil
	default:
		return CaptureBoundary{}, nil, invalid("capture boundary kind is outside stable|unstable_archive")
	}
}

func buildStableProof(input BoundaryInput) (StableSnapshotProof, map[string]any, error) {
	if !validProofKind(input.ProofKind) {
		return StableSnapshotProof{}, nil, invalid("stable snapshot proof kind is outside the closed proof vocabulary")
	}
	generation, err := ParseGeneration(input.Generation)
	if err != nil {
		return StableSnapshotProof{}, nil, err
	}
	var identity *scalar.Digest
	if input.SnapshotIdentity != nil {
		value, err := scalar.ParseDigest(*input.SnapshotIdentity)
		if err != nil {
			return StableSnapshotProof{}, nil, invalid("stable snapshot proof snapshot_identity_digest is not a digest: %v", err)
		}
		identity = &value
	}
	pre, err := scalar.ParseDigest(input.PreCaptureDigest)
	if err != nil {
		return StableSnapshotProof{}, nil, invalid("stable snapshot proof pre_capture_digest is not a digest: %v", err)
	}
	post, err := scalar.ParseDigest(input.PostCaptureDigest)
	if err != nil {
		return StableSnapshotProof{}, nil, invalid("stable snapshot proof post_capture_digest is not a digest: %v", err)
	}
	proof := StableSnapshotProof{
		ProofKind:         input.ProofKind,
		SourceGeneration:  generation,
		SnapshotIdentity:  identity,
		PreCaptureDigest:  pre,
		PostCaptureDigest: post,
		InputBlocked:      input.InputBlocked,
		ForegroundIdle:    input.ForegroundIdle,
		BackgroundIdle:    input.BackgroundIdle,
	}
	if err := checkProofCoupling(proof); err != nil {
		return StableSnapshotProof{}, nil, err
	}
	var identityValue any
	if identity != nil {
		identityValue = identity.String()
	}
	return proof, map[string]any{
		"proof_kind":               proof.ProofKind,
		"source_generation":        generation.String(),
		"snapshot_identity_digest": identityValue,
		"pre_capture_digest":       pre.String(),
		"post_capture_digest":      post.String(),
		"input_blocked":            proof.InputBlocked,
		"foreground_idle":          proof.ForegroundIdle,
		"background_idle":          proof.BackgroundIdle,
		"extensions":               map[string]any{},
	}, nil
}
