package clonefidelity_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gowebpki/jcs"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// Shared fixtures for the clonefidelity suite. Every literal is
// hand-written and independent of production constants: verdicts
// always come from production; vectors never do. The oracle lists
// (dispositions, profiles, strategies, reasons, kinds, blocks,
// classes) are retyped from the pinned SPEC v0.7.0 text in each
// oracle test, never imported from production or from these
// builders.

const (
	fixtureOperationID = "0198f4c8-8e50-7f66-8f70-111111111111"
	fixtureBundleID    = "0198f4c8-8e50-7f66-8f70-222222222222"
	fixtureTargetOpID  = "0198f4c8-8e50-7f66-8f70-333333333333"
)

func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fixtureTupleJSON() string {
	return `{"environment_id":"test.env","environment_version":"2.1.0","platform":"linux","architecture":"amd64","store_schema_fingerprint":"` +
		fixtureDigest("fidelity-store-fingerprint") + `","adapter_version":"1.2.3"}`
}

func strptr(value string) *string { return &value }

// validRowInput returns a valid single-row candidate: source class,
// one evidence ID, canonical object, locator, explanation, and no
// staged/live evidence. Reasons default per disposition: exact rows
// carry none, every other row carries unknown_native_event.
func validRowInput(key, disposition string, reasons ...string) clonefidelity.DispositionRecordInput {
	if reasons == nil {
		if disposition != "exact" {
			reasons = []string{"unknown_native_event"}
		} else {
			reasons = []string{}
		}
	}
	return clonefidelity.DispositionRecordInput{
		SourceItemKey:           key,
		SourceClass:             "durable_payload",
		SourceEvidenceIDs:       []string{fixtureDigest("evidence-" + key)},
		CanonicalObjectID:       strptr(fixtureDigest("canonical-" + key)),
		TargetLocator:           strptr("target/" + key),
		Disposition:             disposition,
		ReasonCodes:             reasons,
		Explanation:             "row " + key,
		StagedEvidenceObjectIDs: []string{},
		LiveEvidenceObjectIDs:   []string{},
		Extensions:              map[string]any{},
	}
}

// validSynthInput returns a valid synthesized-row candidate: no
// source canonical object, one reason, source evidence present.
func validSynthInput(key string) clonefidelity.DispositionRecordInput {
	row := validRowInput(key, "synthesized", "derived_index_rebuilt")
	row.CanonicalObjectID = nil
	return row
}

// concentratedBreakdowns derives reconciling breakdown maps that
// concentrate every row on the first owner key. It hand-counts the
// rows independently of production derivation; production re-derives
// and reconciles at both entries.
func concentratedBreakdowns(rows []clonefidelity.DispositionRecordInput) (map[string]clonefidelity.FidelityCounts, map[string]clonefidelity.FidelityCounts, map[string]uint64) {
	eventKinds := clonebundle.EventKinds()
	blockTypes := clonebundle.ContentBlockTypes()
	events := make(map[string]clonefidelity.FidelityCounts, len(eventKinds))
	for _, kind := range eventKinds {
		events[kind] = clonefidelity.FidelityCounts{}
	}
	blocks := make(map[string]clonefidelity.FidelityCounts, len(blockTypes))
	for _, block := range blockTypes {
		blocks[block] = clonefidelity.FidelityCounts{}
	}
	bytes := map[string]uint64{
		"exact": 0, "semantic": 0, "summarized": 0, "opaque_preserved": 0,
		"synthesized": 0, "omitted": 0, "unrecoverable": 0,
	}
	bump := func(counts clonefidelity.FidelityCounts, disposition string) clonefidelity.FidelityCounts {
		switch disposition {
		case "exact":
			counts.Exact++
		case "semantic":
			counts.Semantic++
		case "summarized":
			counts.Summarized++
		case "opaque_preserved":
			counts.OpaquePreserved++
		case "synthesized":
			counts.Synthesized++
		case "omitted":
			counts.Omitted++
		case "unrecoverable":
			counts.Unrecoverable++
		}
		return counts
	}
	for _, row := range rows {
		events[eventKinds[0]] = bump(events[eventKinds[0]], row.Disposition)
		blocks[blockTypes[0]] = bump(blocks[blockTypes[0]], row.Disposition)
	}
	return events, blocks, bytes
}

// validTargetInput returns a valid target-scope report candidate
// over the given rows with reconciling breakdowns.
func validTargetInput(rows ...clonefidelity.DispositionRecordInput) clonefidelity.FidelityReportInput {
	events, blocks, byteCounts := concentratedBreakdowns(rows)
	return clonefidelity.FidelityReportInput{
		Scope:                            "target",
		OperationID:                      fixtureOperationID,
		BundleID:                         fixtureBundleID,
		SourceSnapshotDigest:             fixtureDigest("source-snapshot"),
		CaptureManifestID:                fixtureDigest("capture-manifest"),
		CanonicalSessionID:               fixtureDigest("canonical-session"),
		ProjectionPlanID:                 strptr(fixtureDigest("projection-plan")),
		SourceEnvironment:                []byte(fixtureTupleJSON()),
		TargetEnvironment:                []byte(fixtureTupleJSON()),
		Profile:                          "maximal_safe",
		RequiredDispositions:             map[string][]string{"durable_payload": {"exact", "semantic"}},
		ForbidReasons:                    []string{"credential_excluded"},
		Rows:                             rows,
		EventKindCounts:                  events,
		ContentBlockCounts:               blocks,
		ByteCounts:                       byteCounts,
		RawBundleComplete:                true,
		CanonicalComplete:                true,
		TargetSemanticallyContinuable:    true,
		TargetNativelyResumable:          true,
		StagedReadBackEvidenceManifestID: strptr(fixtureDigest("staged-manifest")),
		LiveReadBackEvidenceManifestID:   strptr(fixtureDigest("live-manifest")),
		AdapterAttestations:              []string{},
		Extensions:                       map[string]any{},
	}
}

// validArchiveInput returns a valid archive-scope report candidate
// over the given rows: null plan/target/manifest IDs, archive_only
// profile, and false target booleans.
func validArchiveInput(rows ...clonefidelity.DispositionRecordInput) clonefidelity.FidelityReportInput {
	input := validTargetInput(rows...)
	input.Scope = "archive"
	input.Profile = "archive_only"
	input.ProjectionPlanID = nil
	input.TargetEnvironment = nil
	input.TargetSemanticallyContinuable = false
	input.TargetNativelyResumable = false
	input.StagedReadBackEvidenceManifestID = nil
	input.LiveReadBackEvidenceManifestID = nil
	return input
}

func mustBuild(t *testing.T, input clonefidelity.FidelityReportInput) []byte {
	t.Helper()
	sealed, err := clonefidelity.BuildFidelityReport(input)
	if err != nil {
		t.Fatalf("BuildFidelityReport() error = %v", err)
	}
	return sealed
}

func mustDecode(t *testing.T, sealed []byte) clonefidelity.FidelityReport {
	t.Helper()
	report, err := clonefidelity.DecodeFidelityReport(sealed)
	if err != nil {
		t.Fatalf("DecodeFidelityReport() error = %v", err)
	}
	return report
}

// requireRefusal asserts the refusal contract: the stable ErrInvalid
// code plus the literal detail retyped from the expected rule. The
// literal pins the gate; the code pins the refusal identity.
func requireRefusal(t *testing.T, err error, literal string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want refusal containing %q", literal)
	}
	if !errors.Is(err, clonefidelity.ErrInvalid) {
		t.Fatalf("error = %v, want errors.Is ErrInvalid", err)
	}
	if !strings.Contains(err.Error(), literal) {
		t.Fatalf("error = %q, want literal %q", err.Error(), literal)
	}
}

// decodeDocument decodes sealed bytes preserving number literals,
// so tampered reports re-marshal byte-identically except for the
// planted mutation.
func decodeDocument(t *testing.T, sealed []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(sealed))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("decode sealed report: %v", err)
	}
	return document
}

// tampered reseals a report with one planted mutation. Semantic
// mutations break the self digest too, but every semantic gate runs
// before digest verification, so the semantic refusal fires first
// with its literal detail.
func tampered(t *testing.T, sealed []byte, mutate func(document map[string]any)) []byte {
	t.Helper()
	document := decodeDocument(t, sealed)
	mutate(document)
	out, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal tampered report: %v", err)
	}
	return out
}

func tamperRow(document map[string]any, index int, mutate func(row map[string]any)) {
	rows := document["dispositions"].([]any)
	mutate(rows[index].(map[string]any))
}

// independentReportID recomputes the omit-self identity in the test:
// JCS over the sealed members minus fidelity_report_id, hashed with
// SHA-256. It shares only the JCS library with production; the
// omission wiring and digest comparison are the properties under
// test.
func independentReportID(t *testing.T, sealed []byte) string {
	t.Helper()
	document := decodeDocument(t, sealed)
	delete(document, "fidelity_report_id")
	plain, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal omit-self object: %v", err)
	}
	canonical, err := jcs.Transform(plain)
	if err != nil {
		t.Fatalf("JCS transform: %v", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// gridReasons returns n distinct admissible reasons in sorted
// order: the core list first, then reverse-DNS extensions.
func gridReasons(n int) []string {
	core := []string{
		"target_no_equivalent", "source_not_persisted", "source_truncated",
		"source_corrupt", "foreign_encrypted_payload", "foreign_signature_unverifiable",
		"target_schema_constraint", "target_context_limit", "target_size_limit",
		"target_version_gate", "official_importer_loss", "graph_flattened",
		"unsafe_pending_action", "credential_excluded", "secret_policy",
		"operator_policy", "unsupported_media_type", "unknown_native_event",
		"derived_index_rebuilt",
	}
	reasons := make([]string, 0, n)
	for _, reason := range core {
		if len(reasons) >= n {
			break
		}
		reasons = append(reasons, reason)
	}
	for i := 0; len(reasons) < n; i++ {
		reasons = append(reasons, "com.example.grid.r"+itoa(i))
	}
	// Sorted unique: core codes sort before the com.example extensions
	// except none collide; sort explicitly to satisfy the order gate.
	sorted := append([]string(nil), reasons...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
