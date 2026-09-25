package clonereadback_test

import (
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file pins the mode-authority invariant (SPEC v0.7.0
// §13.14.2 lines 10753-10754: "Staged and live manifests are
// distinct and cannot be relabeled."; line 10580: "Modes cannot be
// relabeled."): at every exported read-back entry the mode is
// DERIVED from the trusted read authority purpose (lines
// 3796-3797: purpose:source_native|target_staged|target_live) the
// caller threads from its authority chain, never parsed from the
// manifest being validated. A resealed manifest carrying the
// opposite mode refuses even though its self-digest recomputes,
// and source_native grants no target read-back.

// TestModeAuthorityGrid enumerates purpose (3) × claimed mode (2) ×
// entry (2): 12 cells. target_staged admits staged only,
// target_live admits live only, and source_native refuses both
// modes at both entries. Every Decode mismatch cell first proves
// the sealed id recomputes independently, so the refusal pins the
// authority gate and not the digest. An invalid purpose refuses at
// both entries with the vocabulary literal.
func TestModeAuthorityGrid(t *testing.T) {
	purposes := []struct {
		name   string
		expect string // "" refuses every claim
	}{
		{"target_staged", "staged"},
		{"target_live", "live"},
		{"source_native", ""},
	}
	checked := 0
	for _, purpose := range purposes {
		authority := fixtureAuthority(t, purpose.name)
		for _, mode := range []string{"staged", "live"} {
			want := purpose.expect == mode
			input := validReadBackInput()
			input.Mode = mode
			_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, authority)
			if want && err != nil {
				t.Fatalf("purpose=%s mode=%s Build refused: %v", purpose.name, mode, err)
			}
			if !want {
				if err == nil {
					t.Fatalf("purpose=%s mode=%s Build admitted", purpose.name, mode)
				}
				wantLiteral := "disagrees with read authority purpose"
				if purpose.expect == "" {
					wantLiteral = "grants no target read-back"
				}
				requireRefusal(t, err, "read-back evidence manifest", wantLiteral)
			}
			sealed := sealMode(t, mode)
			if got := independentSelfDigest(t, sealed, "read_back_evidence_manifest_id"); got != decodeDocument(t, sealed)["read_back_evidence_manifest_id"] {
				t.Fatalf("purpose=%s mode=%s: sealed id does not recompute", purpose.name, mode)
			}
			manifest, err := clonereadback.DecodeReadBackEvidenceManifest(sealed, authority)
			if want && err != nil {
				t.Fatalf("purpose=%s mode=%s Decode refused: %v", purpose.name, mode, err)
			}
			if want && manifest.Mode() != mode {
				t.Fatalf("purpose=%s mode=%s decoded as %q", purpose.name, mode, manifest.Mode())
			}
			if !want {
				if err == nil {
					t.Fatalf("purpose=%s mode=%s Decode admitted", purpose.name, mode)
				}
				wantLiteral := "disagrees with read authority purpose"
				if purpose.expect == "" {
					wantLiteral = "grants no target read-back"
				}
				requireRefusal(t, err, "read-back evidence manifest", wantLiteral)
			}
			checked++
		}
	}
	// An authority purpose outside the closed vocabulary refuses at
	// both entries even though no owner decoder runs here: the
	// mapping is total.
	bogus := sessadapter.ReadAuthority{AuthorityID: "0193a5b7-4c2d-7e1f-8a3b-5c6d7e8f9012", Purpose: "target_archive"}
	for _, mode := range []string{"staged", "live"} {
		input := validReadBackInput()
		input.Mode = mode
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, bogus)
		requireRefusal(t, err, "read-back evidence manifest", "is outside source_native|target_staged|target_live")
		_, err = clonereadback.DecodeReadBackEvidenceManifest(sealMode(t, mode), bogus)
		requireRefusal(t, err, "read-back evidence manifest", "is outside source_native|target_staged|target_live")
		checked++
	}
	t.Logf("mode-authority grid: %d purpose/mode cells × 2 entries driven", checked)
}

// sealMode seals one valid manifest claiming the given mode under
// its matching authority.
func sealMode(t *testing.T, mode string) []byte {
	t.Helper()
	authority := stagedAuthority(t)
	if mode == "live" {
		authority = liveAuthority(t)
	}
	input := validReadBackInput()
	input.Mode = mode
	return mustBuildReadBack(t, authority, input)
}

// TestBuildStagedAuthorityRefusesLiveClaim is the staged→live
// regression at the Build entry: a live claim under the staged
// authority refuses as a relabel, never seals.
func TestBuildStagedAuthorityRefusesLiveClaim(t *testing.T) {
	input := validReadBackInput()
	input.Mode = "live"
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", `mode "live" disagrees with read authority purpose "target_staged"`)
}

// TestBuildLiveAuthorityRefusesStagedClaim is the live→staged
// regression at the Build entry: a staged claim under the live
// authority refuses as a relabel, never seals.
func TestBuildLiveAuthorityRefusesStagedClaim(t *testing.T) {
	input := validReadBackInput()
	input.Mode = "staged"
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, liveAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", `mode "staged" disagrees with read authority purpose "target_live"`)
}

// TestDecodeStagedAuthorityRefusesResealedLiveClaim is the
// staged→live regression at the Decode entry: staged evidence
// resealed with mode=live under a recomputed id refuses under the
// staged authority even though the self-digest recomputes.
func TestDecodeStagedAuthorityRefusesResealedLiveClaim(t *testing.T) {
	sealed := sealMode(t, "staged")
	document := decodeDocument(t, sealed)
	document["mode"] = "live"
	document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
	resealed := marshalDocument(t, document)
	if got := independentSelfDigest(t, resealed, "read_back_evidence_manifest_id"); got != decodeDocument(t, resealed)["read_back_evidence_manifest_id"] {
		t.Fatal("resealed id does not recompute: the probe is not a reseal")
	}
	_, err := clonereadback.DecodeReadBackEvidenceManifest(resealed, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", `mode "live" disagrees with read authority purpose "target_staged"`)
}

// TestDecodeLiveAuthorityRefusesResealedStagedClaim is the
// live→staged regression at the Decode entry: live evidence
// resealed with mode=staged under a recomputed id refuses under
// the live authority even though the self-digest recomputes.
func TestDecodeLiveAuthorityRefusesResealedStagedClaim(t *testing.T) {
	sealed := sealMode(t, "live")
	document := decodeDocument(t, sealed)
	document["mode"] = "staged"
	document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
	resealed := marshalDocument(t, document)
	if got := independentSelfDigest(t, resealed, "read_back_evidence_manifest_id"); got != decodeDocument(t, resealed)["read_back_evidence_manifest_id"] {
		t.Fatal("resealed id does not recompute: the probe is not a reseal")
	}
	_, err := clonereadback.DecodeReadBackEvidenceManifest(resealed, liveAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", `mode "staged" disagrees with read authority purpose "target_live"`)
}
