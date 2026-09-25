package clonereadback_test

import (
	"strings"
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// This file proves the semantic gates by whole-domain oracles
// written from the pinned text, never from the implementation: the
// valid rule over every combination of check outcomes, the mode
// vocabulary over its closed set and neighbors, and the native-ID
// equality over equal/unequal pairs.

// oracleChecks renders the seven check booleans for one combination
// index: bit 0 is staged structural, 1 live structural, 2 semantic
// marker, 3 identity, 4 workspace binding, 5 resume surface, 6
// source generation. Every check is always applicable in this
// schema — the report carries no nullable checks — so the oracle
// enumerates outcomes with applicability fixed, and the wrong-type
// census pins that a null check is a wrong type, never an
// inapplicable one.
func oracleChecks(staged, live clonereadback.ValidatedReadBack, combo int) (clonereadback.ValidationReportInput, bool) {
	input := validReportInput(staged, live)
	input.StagedStructuralValid = combo&1 != 0
	input.LiveStructuralValid = combo&2 != 0
	input.SemanticMarkerValid = combo&4 != 0
	input.IdentityValid = combo&8 != 0
	input.WorkspaceBindingValid = combo&16 != 0
	input.ResumeSurfaceValid = combo&32 != 0
	input.SourceGenerationRevalidated = combo&64 != 0
	return input, combo == 127
}

// TestValidWholeDomainOracle enumerates every combination of the
// seven check outcomes (128) by native-ID equality (2) by finding
// class (none, info, warning, error: 4): 1024 combinations. At the
// production Build entry the decided valid bit seals and the
// flipped bit refuses; at the production Decode entry the sealed
// document decodes and the flipped-bit document (under a
// recomputed independent id) refuses.
func TestValidWholeDomainOracle(t *testing.T) {
	staged, live := mustReadPair(t)
	findingClasses := [][]string{nil, {"info"}, {"warning"}, {"info", "warning"}, {"error"}, {"info", "error"}}
	checked := 0
	for combo := 0; combo < 128; combo++ {
		for _, idsEqual := range []bool{true, false} {
			for _, severities := range findingClasses {
				input, allTrue := oracleChecks(staged, live, combo)
				if !idsEqual {
					input.ObservedTargetNativeSession = "a-different-native-session"
				}
				input.Findings = fixtureFindings(severities...)
				want := allTrue && idsEqual && !hasErrorSeverity(severities)
				input.Valid = want
				sealed, err := clonereadback.BuildValidationReport(input, staged, live)
				if err != nil {
					t.Fatalf("combo=%d idsEqual=%v findings=%v: decided valid=%v refused: %v", combo, idsEqual, severities, want, err)
				}
				if _, err := clonereadback.DecodeValidationReport(sealed, staged, live); err != nil {
					t.Fatalf("combo=%d idsEqual=%v findings=%v: sealed doc refuses: %v", combo, idsEqual, severities, err)
				}
				input.Valid = !want
				if _, err := clonereadback.BuildValidationReport(input, staged, live); err == nil {
					t.Fatalf("combo=%d idsEqual=%v findings=%v: flipped valid=%v admitted at Build", combo, idsEqual, severities, !want)
				} else if !strings.Contains(err.Error(), "valid disagrees with the check outcome") {
					t.Fatalf("combo=%d: flipped-bit error %q misses the literal", combo, err.Error())
				}
				document := decodeDocument(t, sealed)
				document["valid"] = !want
				document["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, document), "validation_report_id")
				if _, err := clonereadback.DecodeValidationReport(marshalDocument(t, document), staged, live); err == nil {
					t.Fatalf("combo=%d idsEqual=%v findings=%v: flipped valid=%v admitted at Decode", combo, idsEqual, severities, !want)
				} else if !strings.Contains(err.Error(), "valid disagrees with the check outcome") {
					t.Fatalf("combo=%d: flipped-bit decode error %q misses the literal", combo, err.Error())
				}
				checked++
			}
		}
	}
	t.Logf("valid oracle: %d combinations driven through Build+Decode", checked)
}

func hasErrorSeverity(severities []string) bool {
	for _, severity := range severities {
		if severity == "error" {
			return true
		}
	}
	return false
}

// TestNativeIdentityEqualityOracle pins "equal expected and observed
// native Session IDs" at both read-back entries and both report
// entries: equal pairs seal, and unequal pairs — including the
// same-length neighbor the length-comparing narrowing would admit —
// refuse with the literal.
func TestNativeIdentityEqualityOracle(t *testing.T) {
	staged, live := mustReadPair(t)
	pairs := [][2]string{
		{"session-a", "session-a"},
		{"session-a", "session-b"},
		{"session-a", "session-aa"},
		{"x", "y"},
		{"", ""},
	}
	for _, pair := range pairs {
		want := pair[0] == pair[1] && pair[0] != ""
		input := validReadBackInput()
		input.ExpectedTargetNativeSession = pair[0]
		input.ObservedTargetNativeSession = pair[1]
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
		if want && err != nil {
			t.Fatalf("readback pair %q refused: %v", pair, err)
		}
		if !want && err == nil {
			t.Fatalf("readback pair %q admitted at Build", pair)
		}
		// The report decides valid from the equality: an unequal
		// pair seals with valid=false and refuses with
		// valid=true; an equal pair seals with valid=true.
		// Empty pairs refuse on bounds either way.
		rinput := validReportInput(staged, live)
		rinput.ExpectedTargetNativeSession = pair[0]
		rinput.ObservedTargetNativeSession = pair[1]
		if pair[0] == "" {
			rinput.Valid = false
			if _, err := clonereadback.BuildValidationReport(rinput, staged, live); err == nil {
				t.Fatalf("report pair %q admitted at Build", pair)
			}
			continue
		}
		rinput.Valid = false
		if _, err := clonereadback.BuildValidationReport(rinput, staged, live); pair[0] == pair[1] {
			if err == nil {
				t.Fatalf("report equal pair %q with valid=false admitted at Build", pair)
			}
		} else if err != nil {
			t.Fatalf("report unequal pair %q with valid=false refused: %v", pair, err)
		}
		rinput.Valid = true
		if _, err := clonereadback.BuildValidationReport(rinput, staged, live); pair[0] == pair[1] {
			if err != nil {
				t.Fatalf("report equal pair %q with valid=true refused: %v", pair, err)
			}
		} else if err == nil {
			t.Fatalf("report unequal pair %q with valid=true admitted at Build", pair)
		}
	}
	// The decode-side equality gate refuses a doc whose IDs differ
	// even under a consistent resealed id: the gate reads members,
	// not bytes.
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	document := decodeDocument(t, sealed)
	document["observed_target_native_session_id"] = "a-different-native-session"
	document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
	_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "expected and observed native Session IDs differ")
	rsealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	rdocument := decodeDocument(t, rsealed)
	rdocument["observed_target_native_session_id"] = "a-different-native-session"
	rdocument["valid"] = false
	rdocument["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, rdocument), "validation_report_id")
	report, err := clonereadback.DecodeValidationReport(marshalDocument(t, rdocument), staged, live)
	if err != nil {
		t.Fatalf("differing IDs with valid=false refused: %v", err)
	}
	if report.Valid {
		t.Fatal("differing IDs decoded valid=true")
	}
}

// TestModeVocabularyOracle pins mode:staged|live at both read-back
// entries: the two closed modes seal under their matching
// authorities and every neighbor — case, whitespace, empty,
// archived, both, and invalid UTF-8 — refuses with the literal.
func TestModeVocabularyOracle(t *testing.T) {
	for _, mode := range []string{"staged", "live"} {
		input := validReadBackInput()
		input.Mode = mode
		authority := stagedAuthority(t)
		if mode == "live" {
			authority = liveAuthority(t)
		}
		mustBuildReadBack(t, authority, input)
		if !clonereadback.ValidReadBackMode(mode) {
			t.Fatalf("ValidReadBackMode(%q) = false", mode)
		}
	}
	neighbors := []struct {
		mode    string
		literal string
	}{
		{"", "mode is outside staged|live"}, {" ", "mode is outside staged|live"},
		{"staged ", "mode is outside staged|live"}, {" staged", "mode is outside staged|live"},
		{"STAGED", "mode is outside staged|live"}, {"Staged", "mode is outside staged|live"},
		{"live ", "mode is outside staged|live"}, {"LIVE", "mode is outside staged|live"},
		{"archived", "mode is outside staged|live"}, {"both", "mode is outside staged|live"},
		{"staged,live", "mode is outside staged|live"}, {"null", "mode is outside staged|live"},
		{"dormant", "mode is outside staged|live"}, {"validated", "mode is outside staged|live"},
		{"stagged", "mode is outside staged|live"}, {"liv", "mode is outside staged|live"},
		{"stage", "mode is outside staged|live"},
		{string([]byte{0xff}), "mode is not valid UTF-8"}, {"staged\xff", "mode is not valid UTF-8"},
	}
	for _, probe := range neighbors {
		input := validReadBackInput()
		input.Mode = probe.mode
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
		requireRefusal(t, err, "read-back evidence manifest", probe.literal)
		if clonereadback.ValidReadBackMode(probe.mode) {
			t.Fatalf("ValidReadBackMode(%q) = true", probe.mode)
		}
	}
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range neighbors {
		document := decodeDocument(t, sealed)
		if probe.mode == string([]byte{0xff}) || probe.mode == "staged\xff" {
			continue // unmarshalable bytes cannot ride a JSON document; the Build row above pins them.
		}
		document["mode"] = probe.mode
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		requireRefusal(t, err, "read-back evidence manifest", "mode is outside staged|live")
	}
	if modes := clonereadback.ReadBackModes(); len(modes) != 2 || modes[0] != "staged" || modes[1] != "live" {
		t.Fatalf("ReadBackModes() = %q, want [staged live]", modes)
	}
}

// TestEvidenceKindVocabularyOracle pins
// evidence_kind:native_sample|parser_trace|marker_observation at
// both read-back entries: the three closed kinds seal and every
// neighbor refuses with the literal.
func TestEvidenceKindVocabularyOracle(t *testing.T) {
	for _, kind := range []string{"native_sample", "parser_trace", "marker_observation"} {
		input := validReadBackInput()
		input.EvidenceObjects = []clonereadback.EvidenceObjectInput{validEvidenceInput(kind, kind)}
		mustBuildReadBack(t, stagedAuthority(t), input)
		if !clonereadback.ValidEvidenceKind(kind) {
			t.Fatalf("ValidEvidenceKind(%q) = false", kind)
		}
	}
	neighbors := []struct {
		kind    string
		literal string
	}{
		{"", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"native", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"sample", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"NATIVE_SAMPLE", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"Native_Sample", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"parser", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"trace", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"marker", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"observation", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"native-sample", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"blob", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"evidence", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{"null", "evidence_kind is outside native_sample|parser_trace|marker_observation"},
		{string([]byte{0xff}), "evidence_kind is not valid UTF-8"},
	}
	for _, probe := range neighbors {
		input := validReadBackInput()
		input.EvidenceObjects = []clonereadback.EvidenceObjectInput{validEvidenceInput(probe.kind, "neighbor")}
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
		requireRefusal(t, err, "evidence object", probe.literal)
		if clonereadback.ValidEvidenceKind(probe.kind) {
			t.Fatalf("ValidEvidenceKind(%q) = true", probe.kind)
		}
	}
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range neighbors {
		if probe.kind == string([]byte{0xff}) {
			continue
		}
		document := decodeDocument(t, sealed)
		documentAt(t, document, "evidence_objects", 0)["evidence_kind"] = probe.kind
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		requireRefusal(t, err, "evidence object", "evidence_kind is outside native_sample|parser_trace|marker_observation")
	}
	if kinds := clonereadback.EvidenceKinds(); len(kinds) != 3 || kinds[0] != "native_sample" || kinds[1] != "parser_trace" || kinds[2] != "marker_observation" {
		t.Fatalf("EvidenceKinds() = %q", kinds)
	}
}
