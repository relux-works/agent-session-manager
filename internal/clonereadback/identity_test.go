package clonereadback_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/gowebpki/jcs"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// independentSelfDigest recomputes the omit-self identity without
// touching production: plain marshal, JCS transform, SHA-256. The
// canonical-bytes path is shared infrastructure (JCS itself); the
// omission choice and digest comparison are independent.
func independentSelfDigest(t *testing.T, sealed []byte, self string) string {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(sealed, &document); err != nil {
		t.Fatalf("unmarshal sealed: %v", err)
	}
	delete(document, self)
	plain, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal omitted: %v", err)
	}
	canonical, err := jcs.Transform(plain)
	if err != nil {
		t.Fatalf("JCS transform: %v", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// TestReadBackIdentityIndependent pins that
// read_back_evidence_manifest_id is the JCS digest with only itself
// omitted: the independent recompute agrees, omitting a different
// member disagrees, and a tampered claim refuses.
func TestReadBackIdentityIndependent(t *testing.T) {
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	document := decodeDocument(t, sealed)
	claimed, ok := document["read_back_evidence_manifest_id"].(string)
	if !ok {
		t.Fatal("read_back_evidence_manifest_id is not a string")
	}
	if got := independentSelfDigest(t, sealed, "read_back_evidence_manifest_id"); got != claimed {
		t.Fatalf("independent digest = %s, sealed claim = %s", got, claimed)
	}
	if got := independentSelfDigest(t, sealed, "mode"); got == claimed {
		t.Fatal("omitting mode instead of the id agrees: the omission rule is not pinned")
	}
	document["read_back_evidence_manifest_id"] = fixtureDigest("tampered-claim")
	_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read_back_evidence_manifest_id", "does not match omit-self digest")
	document = decodeDocument(t, sealed)
	document["read_back_evidence_manifest_id"] = "not-a-digest"
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read_back_evidence_manifest_id", "is not a digest")
}

// TestReportIdentityIndependent pins that validation_report_id is
// the JCS digest with only itself omitted.
func TestReportIdentityIndependent(t *testing.T) {
	staged, live := mustReadPair(t)
	sealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	document := decodeDocument(t, sealed)
	claimed, ok := document["validation_report_id"].(string)
	if !ok {
		t.Fatal("validation_report_id is not a string")
	}
	if got := independentSelfDigest(t, sealed, "validation_report_id"); got != claimed {
		t.Fatalf("independent digest = %s, sealed claim = %s", got, claimed)
	}
	if got := independentSelfDigest(t, sealed, "valid"); got == claimed {
		t.Fatal("omitting valid instead of the id agrees: the omission rule is not pinned")
	}
	document["validation_report_id"] = fixtureDigest("tampered-claim")
	_, err := clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
	requireRefusal(t, err, "validation_report_id", "does not match omit-self digest")
	document = decodeDocument(t, sealed)
	document["validation_report_id"] = "not-a-digest"
	_, err = clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
	requireRefusal(t, err, "validation_report_id", "is not a digest")
}

// TestModeRelabelRefused pins "Modes cannot be relabeled"
// (SPEC v0.7.0 §13.14.2 line 10580 and lines 10753-10754):
// flipping the mode under a sealed id refuses at the production
// decode entry; under the producing authority the refusal is the
// authority mismatch (the gate reads the claim against trusted
// purpose, not bytes), and under the flipped authority — where
// the claim agrees — the stale id refuses as an identity
// mismatch. Either way the relabeled bytes are a different
// document, never the same manifest.
func TestModeRelabelRefused(t *testing.T) {
	for _, mode := range []string{"staged", "live"} {
		authority := stagedAuthority(t)
		flippedAuthority := liveAuthority(t)
		flipped := "live"
		if mode == "live" {
			authority = liveAuthority(t)
			flippedAuthority = stagedAuthority(t)
			flipped = "staged"
		}
		input := validReadBackInput()
		input.Mode = mode
		sealed := mustBuildReadBack(t, authority, input)
		document := decodeDocument(t, sealed)
		document["mode"] = flipped
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), authority)
		requireRefusal(t, err, "read-back evidence manifest", "disagrees with read authority purpose")
		_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), flippedAuthority)
		requireRefusal(t, err, "read_back_evidence_manifest_id", "does not match omit-self digest")
	}
}

// TestModeBindsIdentity pins that the mode binds the authority,
// not just the bytes (SPEC v0.7.0 §13.14.2 lines 10753-10754:
// "Staged and live manifests are distinct and cannot be
// relabeled."): the same logical inputs sealed under staged and
// live carry different ids, and resealing the flipped mode under
// its own recomputed id REFUSES under the producing authority —
// the digest alone does not carry the stage, so a recomputed
// digest never authorizes the opposite mode.
func TestModeBindsIdentity(t *testing.T) {
	stagedInput := validReadBackInput()
	stagedInput.Mode = "staged"
	liveInput := validReadBackInput()
	liveInput.Mode = "live"
	staged := mustBuildReadBack(t, stagedAuthority(t), stagedInput)
	live := mustBuildReadBack(t, liveAuthority(t), liveInput)
	stagedDoc := decodeDocument(t, staged)
	liveDoc := decodeDocument(t, live)
	if stagedDoc["read_back_evidence_manifest_id"] == liveDoc["read_back_evidence_manifest_id"] {
		t.Fatal("staged and live manifests share an id: modes do not bind identity")
	}
	relabeled := decodeDocument(t, staged)
	relabeled["mode"] = "live"
	relabeled["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, relabeled), "read_back_evidence_manifest_id")
	if relabeled["read_back_evidence_manifest_id"] != liveDoc["read_back_evidence_manifest_id"] {
		t.Fatal("recomputed relabel id disagrees with the live seal: identity is not a pure function of members")
	}
	_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, relabeled), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "disagrees with read authority purpose")
}
