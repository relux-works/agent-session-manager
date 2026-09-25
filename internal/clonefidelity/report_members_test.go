package clonefidelity_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gowebpki/jcs"
	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// TestReportRoundTrip seals a valid target report and a valid
// archive report, decodes each, and asserts every field survived
// with literal schema, scope, and profile values.
func TestReportRoundTrip(t *testing.T) {
	rows := []clonefidelity.DispositionRecordInput{
		validRowInput("k-exact", "exact"),
		validRowInput("k-semantic", "semantic", "target_no_equivalent"),
		validSynthInput("k-synth"),
	}
	target := validTargetInput(rows...)
	sealed := mustBuild(t, target)
	report := mustDecode(t, sealed)
	if report.Scope != "target" {
		t.Fatalf("scope = %q, want target", report.Scope)
	}
	if report.Profile != "maximal_safe" {
		t.Fatalf("profile = %q, want maximal_safe", report.Profile)
	}
	if report.ProjectionPlanID == nil || report.TargetEnvironment == nil {
		t.Fatal("target branch members are null, want non-null")
	}
	if report.StagedReadBackEvidenceManifestID == nil || report.LiveReadBackEvidenceManifestID == nil {
		t.Fatal("target manifest IDs are null, want non-null")
	}
	if len(report.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(report.Rows))
	}
	if report.Counts.Exact != 1 || report.Counts.Semantic != 1 || report.Counts.Synthesized != 1 {
		t.Fatalf("counts = %+v, want one exact/semantic/synthesized", report.Counts)
	}
	if report.ReasonCounts["target_no_equivalent"] != 1 || report.ReasonCounts["derived_index_rebuilt"] != 1 {
		t.Fatalf("reason_counts = %v, want one per row reason", report.ReasonCounts)
	}
	if report.OperationID.String() != fixtureOperationID || report.BundleID.String() != fixtureBundleID {
		t.Fatal("operation/bundle IDs did not survive the round trip")
	}
	archive := validArchiveInput(rows...)
	sealedArchive := mustBuild(t, archive)
	decoded := mustDecode(t, sealedArchive)
	if decoded.Scope != "archive" || decoded.Profile != "archive_only" {
		t.Fatalf("scope/profile = %q/%q, want archive/archive_only", decoded.Scope, decoded.Profile)
	}
	if decoded.ProjectionPlanID != nil || decoded.TargetEnvironment != nil {
		t.Fatal("archive branch members are non-null, want null")
	}
	if decoded.TargetSemanticallyContinuable || decoded.TargetNativelyResumable {
		t.Fatal("archive target booleans are true, want false")
	}
	// Schema literals survive on the wire.
	document := decodeDocument(t, sealed)
	if document["schema"] != "urn:ax:schema:fidelity-report" {
		t.Fatalf("schema = %v, want urn:ax:schema:fidelity-report", document["schema"])
	}
	if document["schema_version"] != "1.0.0" {
		t.Fatalf("schema_version = %v, want 1.0.0", document["schema_version"])
	}
}

// TestReportMemberSet pins the exact 29-member report shape from the
// SPEC v0.7.0 Fidelity Report 1.0.0 table: one unknown member
// refuses, and removing any single member refuses naming it.
func TestReportMemberSet(t *testing.T) {
	members := []string{
		"schema", "schema_version", "fidelity_report_id",
		"scope", "operation_id", "bundle_id",
		"source_snapshot_digest", "capture_manifest_id", "canonical_session_id",
		"projection_plan_id", "source_environment", "target_environment",
		"profile", "required_dispositions", "forbid_reasons",
		"dispositions", "counts", "event_kind_counts",
		"content_block_counts", "byte_counts", "reason_counts",
		"raw_bundle_complete", "canonical_complete",
		"target_semantically_continuable", "target_natively_resumable",
		"staged_read_back_evidence_manifest_id", "live_read_back_evidence_manifest_id",
		"adapter_attestations", "extensions",
	}
	if len(members) != 29 {
		t.Fatalf("oracle member count = %d, want 29", len(members))
	}
	sealed := mustBuild(t, validTargetInput(validRowInput("member-row", "exact")))
	document := decodeDocument(t, sealed)
	if len(document) != 29 {
		t.Fatalf("sealed member count = %d, want 29", len(document))
	}
	for _, member := range members {
		if _, present := document[member]; !present {
			t.Errorf("sealed report misses member %q", member)
		}
	}
	mutated := tampered(t, sealed, func(document map[string]any) { document["frobnicate"] = 1 })
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `carries unknown member "frobnicate"`)
	for _, member := range members {
		mutated := tampered(t, sealed, func(document map[string]any) { delete(document, member) })
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, `misses a required member "`+member+`"`)
	}
	// Schema/version/scope literals refuse with literal codes.
	schemaCases := []struct {
		name    string
		member  string
		value   any
		literal string
	}{
		{"schema_wrong", "schema", "urn:ax:schema:fidelity-repor", "schema is not urn:ax:schema:fidelity-report"},
		{"schema_empty", "schema", "", "schema is not urn:ax:schema:fidelity-report"},
		{"schema_number", "schema", 7, "schema is not urn:ax:schema:fidelity-report"},
		{"version_wrong", "schema_version", "1.0.1", "schema_version is not 1.0.0"},
		{"version_number", "schema_version", 1, "schema_version is not 1.0.0"},
		{"scope_wrong", "scope", "staging", "scope is outside archive|target"},
		{"scope_empty", "scope", "", "scope is outside archive|target"},
		{"scope_number", "scope", 7, "scope is outside archive|target"},
		{"scope_null", "scope", nil, "scope is outside archive|target"},
	}
	for _, tc := range schemaCases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := tampered(t, sealed, func(document map[string]any) { document[tc.member] = tc.value })
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.literal)
		})
	}
	// Build side refuses a bad scope literal too.
	input := validTargetInput(validRowInput("member-row", "exact"))
	input.Scope = "staging"
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "scope is outside archive|target")
}

// TestRecordMemberSet pins the exact 11-member disposition row shape:
// one unknown member refuses, and removing any single member refuses
// naming it.
func TestRecordMemberSet(t *testing.T) {
	members := []string{
		"source_item_key", "source_class", "source_evidence_ids",
		"canonical_object_id", "target_locator", "disposition",
		"reason_codes", "explanation", "staged_evidence_object_ids",
		"live_evidence_object_ids", "extensions",
	}
	if len(members) != 11 {
		t.Fatalf("oracle member count = %d, want 11", len(members))
	}
	sealed := mustBuild(t, validTargetInput(validRowInput("member-row", "exact")))
	mutated := tampered(t, sealed, func(document map[string]any) {
		tamperRow(document, 0, func(row map[string]any) { row["frobnicate"] = 1 })
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `carries unknown member "frobnicate"`)
	for _, member := range members {
		mutated := tampered(t, sealed, func(document map[string]any) {
			tamperRow(document, 0, func(row map[string]any) { delete(row, member) })
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, `misses a required member "`+member+`"`)
	}
}

// TestCountsMemberSet pins the exact seven-member FidelityCounts
// shape: one unknown member refuses, and removing any single member
// refuses naming it.
func TestCountsMemberSet(t *testing.T) {
	members := []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"}
	sealed := mustBuild(t, validTargetInput(validRowInput("member-row", "exact")))
	mutated := tampered(t, sealed, func(document map[string]any) {
		document["counts"].(map[string]any)["frobnicate"] = 1
	})
	_, err := clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, `carries unknown member "frobnicate"`)
	for _, member := range members {
		mutated := tampered(t, sealed, func(document map[string]any) {
			delete(document["counts"].(map[string]any), member)
		})
		_, err := clonefidelity.DecodeFidelityReport(mutated)
		requireRefusal(t, err, `misses a required member "`+member+`"`)
	}
}

// TestReportIdentityIndependent recomputes fidelity_report_id in the
// test (JCS over the sealed members minus only that member,
// SHA-256): the recomputed digest equals the sealed claim and the
// decoded identity. Omitting any other member instead yields a
// different digest, pinning the omission set exactly.
func TestReportIdentityIndependent(t *testing.T) {
	sealed := mustBuild(t, validTargetInput(validRowInput("identity-row", "exact")))
	document := decodeDocument(t, sealed)
	claimed, ok := document["fidelity_report_id"].(string)
	if !ok || claimed == "" {
		t.Fatalf("sealed fidelity_report_id = %v, want a digest string", document["fidelity_report_id"])
	}
	recomputed := independentReportID(t, sealed)
	if recomputed != claimed {
		t.Fatalf("independent digest = %q, sealed claim = %q", recomputed, claimed)
	}
	report := mustDecode(t, sealed)
	if report.ReportID.String() != claimed {
		t.Fatalf("decoded identity = %q, sealed claim = %q", report.ReportID.String(), claimed)
	}
	// Omitting a different member instead must disagree: only the
	// self member is omitted.
	document["fidelity_report_id"] = claimed
	delete(document, "scope")
	plain, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	canonical, err := jcs.Transform(plain)
	if err != nil {
		t.Fatalf("JCS: %v", err)
	}
	sum := sha256.Sum256(canonical)
	wrongOmission := "sha256:" + hex.EncodeToString(sum[:])
	if wrongOmission == claimed {
		t.Fatal("omitting scope instead of the self member agrees: omission set unpinned")
	}
	// The all-zero claim is well-formed but wrong: it refuses with
	// the mismatch detail (and anchors the self-digest narrowing).
	zero := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	mutated := tampered(t, sealed, func(document map[string]any) { document["fidelity_report_id"] = zero })
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "does not match omit-self digest")
	// A tampered claim refuses with the literal mismatch detail.
	last := claimed[len(claimed)-1:]
	flippedHex := "0"
	if last == "0" {
		flippedHex = "1"
	}
	flipped := claimed[:len(claimed)-1] + flippedHex
	mutated = tampered(t, sealed, func(document map[string]any) { document["fidelity_report_id"] = flipped })
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "does not match omit-self digest")
	// A non-digest claim refuses at the shape gate.
	mutated = tampered(t, sealed, func(document map[string]any) { document["fidelity_report_id"] = "not-a-digest" })
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "fidelity_report_id is not a digest")
}

// TestReportFraming pins the strict frame at both object layers: a
// duplicate top-level member, trailing bytes, a non-object
// document, lone surrogates, and invalid UTF-8 refuse; so does a
// duplicate member inside a row.
func TestReportFraming(t *testing.T) {
	sealed := mustBuild(t, validTargetInput(validRowInput("framing-row", "exact")))
	duplicate := append(append([]byte{}, sealed[:len(sealed)-1]...), []byte(`,"scope":"target"}`)...)
	_, err := clonefidelity.DecodeFidelityReport(duplicate)
	requireRefusal(t, err, "duplicate member")
	trailing := append(append([]byte{}, sealed...), '7')
	_, err = clonefidelity.DecodeFidelityReport(trailing)
	requireRefusal(t, err, "trailing data after the object")
	_, err = clonefidelity.DecodeFidelityReport([]byte(`[1,2]`))
	requireRefusal(t, err, "not a JSON object")
	_, err = clonefidelity.DecodeFidelityReport([]byte("\xff\xfe"))
	requireRefusal(t, err, "not valid UTF-8")
	lone := `{"schema":"urn:ax:schema:fidelity-report","k":"` + "\\ud800" + `"}`
	_, err = clonefidelity.DecodeFidelityReport([]byte(lone))
	requireRefusal(t, err, "lone surrogate escape")
	// Duplicate member inside a row refuses at the row frame gate.
	text := string(sealed)
	marker := `"source_class":"durable_payload"`
	if !strings.Contains(text, marker) {
		t.Fatalf("sealed report lacks %q", marker)
	}
	dupRow := strings.Replace(text, marker, marker+`,"source_class":"durable_payload"`, 1)
	_, err = clonefidelity.DecodeFidelityReport([]byte(dupRow))
	requireRefusal(t, err, "duplicate member")
}

// TestReportDeterminism builds the same input ten times: sealing is
// pure bytes, so every output is byte-identical (Go map iteration
// order cannot leak through JCS).
func TestReportDeterminism(t *testing.T) {
	rows := []clonefidelity.DispositionRecordInput{
		validRowInput("k-exact", "exact"),
		validRowInput("k-semantic", "semantic", "target_no_equivalent"),
		validSynthInput("k-synth"),
	}
	input := validTargetInput(rows...)
	first := mustBuild(t, input)
	for i := 0; i < 10; i++ {
		next := mustBuild(t, input)
		if string(next) != string(first) {
			t.Fatalf("build %d differs from build 0", i+1)
		}
	}
}

// TestEntriesArePure pins the no-durable-write bound operationally:
// Decode never mutates its input bytes, and Build output always
// decodes clean.
func TestEntriesArePure(t *testing.T) {
	sealed := mustBuild(t, validTargetInput(validRowInput("pure-row", "exact")))
	before := append([]byte(nil), sealed...)
	mustDecode(t, sealed)
	if string(sealed) != string(before) {
		t.Fatal("DecodeFidelityReport mutated its input bytes")
	}
}
