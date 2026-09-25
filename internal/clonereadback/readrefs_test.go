package clonereadback_test

import (
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// This file pins that a Clone Validation Report aggregates TWO
// DISTINCT reads (SPEC v0.7.0 §13.14.2 lines 10761-10762: the
// staged and live read-back evidence manifest ids; lines
// 10580-10582: two reads, every applicable check must pass): the
// staged and live references must name the trusted sibling reads
// in their own slots. The same digest in both slots, a swapped
// pair, a missing reference, or a twice-named read refuses at
// both entries.

// TestBuildReportRefusesEqualReadRefs is the equal-references
// regression at the Build entry: the same digest in both slots
// refuses, and naming one read twice refuses as not distinct.
func TestBuildReportRefusesEqualReadRefs(t *testing.T) {
	staged, live := mustReadPair(t)
	same := [][2]string{
		{staged.ManifestID().String(), staged.ManifestID().String()},
		{live.ManifestID().String(), live.ManifestID().String()},
	}
	for _, pair := range same {
		input := validReportInput(staged, live)
		input.StagedReadBackEvidenceManifestID = pair[0]
		input.LiveReadBackEvidenceManifestID = pair[1]
		_, err := clonereadback.BuildValidationReport(input, staged, live)
		if err == nil {
			t.Fatalf("Build admitted equal read refs %q", pair)
		}
		requireRefusal(t, err, "clone validation report", "does not match")
	}
	input := validReportInput(staged, live)
	input.LiveReadBackEvidenceManifestID = input.StagedReadBackEvidenceManifestID
	_, err := clonereadback.BuildValidationReport(input, staged, staged)
	requireRefusal(t, err, "clone validation report", "are not distinct")
}

// TestDecodeReportRefusesEqualReadRefs is the equal-references
// regression at the Decode entry: the same digest in both slots
// refuses under a recomputed id, and naming one read twice
// refuses as not distinct.
func TestDecodeReportRefusesEqualReadRefs(t *testing.T) {
	staged, live := mustReadPair(t)
	sealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	same := [][2]string{
		{staged.ManifestID().String(), staged.ManifestID().String()},
		{live.ManifestID().String(), live.ManifestID().String()},
	}
	for _, pair := range same {
		document := decodeDocument(t, sealed)
		document["staged_read_back_evidence_manifest_id"] = pair[0]
		document["live_read_back_evidence_manifest_id"] = pair[1]
		document["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, document), "validation_report_id")
		_, err := clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
		if err == nil {
			t.Fatalf("Decode admitted equal read refs %q", pair)
		}
		requireRefusal(t, err, "clone validation report", "does not match")
	}
	document := decodeDocument(t, sealed)
	document["live_read_back_evidence_manifest_id"] = document["staged_read_back_evidence_manifest_id"]
	document["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, document), "validation_report_id")
	_, err := clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, staged)
	requireRefusal(t, err, "clone validation report", "are not distinct")
}

// TestBuildReportRefusesSwappedReadRefs is the swapped-slots
// regression at the Build entry: the live read id in the staged
// slot (and vice versa) refuses.
func TestBuildReportRefusesSwappedReadRefs(t *testing.T) {
	staged, live := mustReadPair(t)
	input := validReportInput(staged, live)
	input.StagedReadBackEvidenceManifestID = live.ManifestID().String()
	input.LiveReadBackEvidenceManifestID = staged.ManifestID().String()
	_, err := clonereadback.BuildValidationReport(input, staged, live)
	requireRefusal(t, err, "clone validation report", "staged_read_back_evidence_manifest_id does not match the staged read")
}

// TestDecodeReportRefusesSwappedReadRefs is the swapped-slots
// regression at the Decode entry: the live read id in the staged
// slot (and vice versa) refuses under a recomputed id.
func TestDecodeReportRefusesSwappedReadRefs(t *testing.T) {
	staged, live := mustReadPair(t)
	sealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	document := decodeDocument(t, sealed)
	document["staged_read_back_evidence_manifest_id"] = live.ManifestID().String()
	document["live_read_back_evidence_manifest_id"] = staged.ManifestID().String()
	document["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, document), "validation_report_id")
	_, err := clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
	requireRefusal(t, err, "clone validation report", "staged_read_back_evidence_manifest_id does not match the staged read")
}

// TestReportReadRefRelationOracle enumerates the whole reference
// relation class at both report entries: claims over
// {staged-id, live-id, unrelated} × {staged-id, live-id,
// unrelated} admit exactly the (staged, live) cell; a missing
// reference refuses; and the sibling relation (twice-named read,
// swapped slots) refuses.
func TestReportReadRefRelationOracle(t *testing.T) {
	staged, live := mustReadPair(t)
	stageID := staged.ManifestID().String()
	liveID := live.ManifestID().String()
	other := fixtureDigest("unrelated-manifest")
	pairs := []struct {
		staged  string
		live    string
		want    bool
		literal string
	}{
		{stageID, liveID, true, ""},
		{stageID, stageID, false, "does not match the live read"},
		{liveID, liveID, false, "does not match the staged read"},
		{liveID, stageID, false, "does not match the staged read"},
		{stageID, other, false, "does not match the live read"},
		{other, liveID, false, "does not match the staged read"},
		{other, other, false, "does not match the staged read"},
	}
	checked := 0
	sealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	for _, pair := range pairs {
		input := validReportInput(staged, live)
		input.StagedReadBackEvidenceManifestID = pair.staged
		input.LiveReadBackEvidenceManifestID = pair.live
		_, err := clonereadback.BuildValidationReport(input, staged, live)
		if pair.want && err != nil {
			t.Fatalf("claims (%s, %s) Build refused: %v", shortDigest(pair.staged), shortDigest(pair.live), err)
		}
		if !pair.want {
			requireRefusal(t, err, "clone validation report", pair.literal)
		}
		document := decodeDocument(t, sealed)
		document["staged_read_back_evidence_manifest_id"] = pair.staged
		document["live_read_back_evidence_manifest_id"] = pair.live
		document["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, document), "validation_report_id")
		_, err = clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
		if pair.want && err != nil {
			t.Fatalf("claims (%s, %s) Decode refused: %v", shortDigest(pair.staged), shortDigest(pair.live), err)
		}
		if !pair.want {
			requireRefusal(t, err, "clone validation report", pair.literal)
		}
		checked++
	}
	// A missing reference refuses at Build as a malformed digest
	// and at Decode as a missing member (the census pins the
	// literal; the id staleness never reads first).
	for _, member := range []string{"staged_read_back_evidence_manifest_id", "live_read_back_evidence_manifest_id"} {
		input := validReportInput(staged, live)
		if member == "staged_read_back_evidence_manifest_id" {
			input.StagedReadBackEvidenceManifestID = ""
		} else {
			input.LiveReadBackEvidenceManifestID = ""
		}
		_, err := clonereadback.BuildValidationReport(input, staged, live)
		requireRefusal(t, err, "clone validation report", member+" is not a digest")
		document := decodeDocument(t, sealed)
		delete(document, member)
		_, err = clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live)
		requireRefusal(t, err, "clone validation report", "misses a required member \""+member+"\"")
		checked++
	}
	// The sibling relation refuses at both entries: a twice-named
	// read is not two distinct reads, and siblings in swapped
	// slots fail their own slot mode.
	siblings := []struct {
		label  string
		first  clonereadback.ValidatedReadBack
		second clonereadback.ValidatedReadBack
		claims [2]string
		want   string
	}{
		{"twice-staged", staged, staged, [2]string{stageID, stageID}, "are not distinct"},
		{"twice-live", live, live, [2]string{liveID, liveID}, "are not distinct"},
		{"swapped-siblings", live, staged, [2]string{stageID, liveID}, "staged read is not a staged manifest"},
	}
	for _, probe := range siblings {
		input := validReportInput(staged, live)
		input.StagedReadBackEvidenceManifestID = probe.claims[0]
		input.LiveReadBackEvidenceManifestID = probe.claims[1]
		_, err := clonereadback.BuildValidationReport(input, probe.first, probe.second)
		requireRefusal(t, err, "clone validation report", probe.want)
		document := decodeDocument(t, sealed)
		document["staged_read_back_evidence_manifest_id"] = probe.claims[0]
		document["live_read_back_evidence_manifest_id"] = probe.claims[1]
		document["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, document), "validation_report_id")
		_, err = clonereadback.DecodeValidationReport(marshalDocument(t, document), probe.first, probe.second)
		requireRefusal(t, err, "clone validation report", probe.want)
		checked++
	}
	t.Logf("read-ref relation oracle: %d cells × 2 entries driven", checked)
}

// shortDigest abbreviates a digest for failure messages.
func shortDigest(digest string) string {
	if len(digest) > 16 {
		return digest[:16]
	}
	return digest
}
