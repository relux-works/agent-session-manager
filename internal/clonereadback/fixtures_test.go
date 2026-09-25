package clonereadback_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file builds the valid baseline inputs every conformance test
// mutates: deterministic digests, UUIDv7 operation IDs, owner-valid
// tuple/binding/finding documents, and the must-build/must-decode
// seams.

// fixtureDigest renders a deterministic pinned-form digest for one
// seed.
func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte("clonereadback:" + seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// fixtureOperationID renders a deterministic lowercase UUIDv7: the
// time field pins the seed hash, the version nibble pins 7, and the
// variant bits pin 10.
func fixtureOperationID(seed string) string {
	sum := sha256.Sum256([]byte("clonereadback-operation:" + seed))
	hexed := hex.EncodeToString(sum[:])
	timeField := hexed[:12]
	randA := hexed[12:15]
	randB := hexed[16:28]
	return timeField[:8] + "-" + timeField[8:12] + "-7" + randA + "-8" + randB[:3] + "-" + randB[3:12] + hexed[28:31]
}

// fixtureTuple is one owner-valid Environment Tuple document.
func fixtureTuple() []byte {
	return []byte(`{"adapter_version":"1.2.3","architecture":"arm64","environment_id":"relux.target","environment_version":"2026.09","platform":"macos","store_schema_fingerprint":"` + fixtureDigest("store-fingerprint") + `"}`)
}

// fixtureBinding is one owner-valid Workspace Binding document.
func fixtureBinding() []byte {
	first, second := fixtureDigest("remote-a"), fixtureDigest("remote-b")
	if second < first {
		first, second = second, first
	}
	return []byte(`{"branch":"main","cwd_relative":"work/session","extensions":{},"head_digest":"` + fixtureDigest("head") + `","index_digest":null,"logical_workspace_id":"` + fixtureOperationID("workspace") + `","repository_remote_fingerprints":["` + first + `","` + second + `"],"working_tree_digest":null}`)
}

// fixtureFinding renders one owner-valid Adapter Finding document
// with the given severity.
func fixtureFinding(severity, code string) string {
	return `{"code":"` + code + `","extensions":{},"message":"finding ` + code + `","remediation":null,"severity":"` + severity + `"}`
}

// fixtureFindings renders one owner-valid findings array over the
// given severities.
func fixtureFindings(severities ...string) []byte {
	var rows []string
	for index, severity := range severities {
		rows = append(rows, fixtureFinding(severity, fmt.Sprintf("probe-%04d", index)))
	}
	return []byte("[" + strings.Join(rows, ",") + "]")
}

// validEvidenceInput renders one valid evidence row candidate with a
// caller-chosen kind and blob seed.
func validEvidenceInput(kind, seed string) clonereadback.EvidenceObjectInput {
	return clonereadback.EvidenceObjectInput{
		EvidenceKind:     kind,
		MediaType:        "application/octet-stream",
		ByteCount:        64,
		BlobID:           fixtureDigest("blob-" + seed),
		BlobDescriptorID: fixtureDigest("descriptor-" + seed),
	}
}

// validReadBackInput renders one valid read-back manifest candidate:
// staged mode, two evidence rows in (kind, blob) order.
func validReadBackInput() clonereadback.ReadBackManifestInput {
	second := validEvidenceInput("parser_trace", "trace")
	first := validEvidenceInput("native_sample", "sample")
	return clonereadback.ReadBackManifestInput{
		OperationID:                 fixtureOperationID("read-back"),
		Mode:                        "staged",
		ProjectionPlanID:            fixtureDigest("plan"),
		ProjectedObjectManifestID:   fixtureDigest("projected"),
		ExpectedTargetNativeSession: "target-native-session-1",
		ObservedTargetNativeSession: "target-native-session-1",
		ObservedEnvironment:         fixtureTuple(),
		ParsedEventCount:            41,
		ParsedHeadIDs:               []string{"head-a", "head-b"},
		WorkspaceBinding:            fixtureBinding(),
		StructuralDigest:            fixtureDigest("structural"),
		EvidenceObjects:             []clonereadback.EvidenceObjectInput{first, second},
		Extensions:                  map[string]any{},
	}
}

// fixtureAuthority decodes one owner-valid ReadAuthority carrying
// the given purpose through the sessadapter owner: the fixtures
// thread genuinely validated authorities, never hand-built
// structs, mirroring how the caller chain mints them.
func fixtureAuthority(t *testing.T, purpose string) sessadapter.ReadAuthority {
	t.Helper()
	raw := []byte(`{"authority_id":"` + fixtureOperationID("authority-"+purpose) + `","purpose":"` + purpose + `","root_handle_names":["handle-a"],"expires_at":"2026-09-24T00:00:00.000Z","extensions":{}}`)
	authority, err := sessadapter.DecodeReadAuthority(raw)
	if err != nil {
		t.Fatalf("DecodeReadAuthority(%s): %v", purpose, err)
	}
	return authority
}

// stagedAuthority renders the trusted target_staged authority every
// staged read-back case threads.
func stagedAuthority(t *testing.T) sessadapter.ReadAuthority {
	t.Helper()
	return fixtureAuthority(t, "target_staged")
}

// liveAuthority renders the trusted target_live authority every
// live read-back case threads.
func liveAuthority(t *testing.T) sessadapter.ReadAuthority {
	t.Helper()
	return fixtureAuthority(t, "target_live")
}

// sourceAuthority renders the trusted source_native authority the
// purpose grid refuses for target read-back.
func sourceAuthority(t *testing.T) sessadapter.ReadAuthority {
	t.Helper()
	return fixtureAuthority(t, "source_native")
}

// validLiveReadBackInput renders one valid live read-back manifest
// candidate: the staged baseline claimed under live.
func validLiveReadBackInput() clonereadback.ReadBackManifestInput {
	input := validReadBackInput()
	input.Mode = "live"
	return input
}

// validReportInput renders one valid validation report candidate:
// every check true, matching IDs, one info finding, valid true,
// and read references naming the two trusted sibling reads.
func validReportInput(staged, live clonereadback.ValidatedReadBack) clonereadback.ValidationReportInput {
	return clonereadback.ValidationReportInput{
		OperationID:                      fixtureOperationID("report"),
		ProjectionPlanID:                 fixtureDigest("plan"),
		ProjectedObjectManifestID:        fixtureDigest("projected"),
		StagedReadBackEvidenceManifestID: staged.ManifestID().String(),
		LiveReadBackEvidenceManifestID:   live.ManifestID().String(),
		TargetProviderManifestID:         fixtureDigest("provider"),
		FidelityReportID:                 fixtureDigest("fidelity"),
		ExpectedTargetNativeSession:      "target-native-session-1",
		ObservedTargetNativeSession:      "target-native-session-1",
		TargetEnvironment:                fixtureTuple(),
		StagedStructuralValid:            true,
		LiveStructuralValid:              true,
		SemanticMarkerValid:              true,
		IdentityValid:                    true,
		WorkspaceBindingValid:            true,
		ResumeSurfaceValid:               true,
		SourceGenerationRevalidated:      true,
		Findings:                         fixtureFindings("info"),
		Valid:                            true,
		Extensions:                       map[string]any{},
	}
}

func mustBuildReadBack(t *testing.T, authority sessadapter.ReadAuthority, input clonereadback.ReadBackManifestInput) []byte {
	t.Helper()
	_, sealed, err := clonereadback.BuildReadBackEvidenceManifest(input, authority)
	if err != nil {
		t.Fatalf("BuildReadBackEvidenceManifest: %v", err)
	}
	return sealed
}

func mustDecodeReadBack(t *testing.T, authority sessadapter.ReadAuthority, sealed []byte) clonereadback.ValidatedReadBack {
	t.Helper()
	manifest, err := clonereadback.DecodeReadBackEvidenceManifest(sealed, authority)
	if err != nil {
		t.Fatalf("DecodeReadBackEvidenceManifest: %v", err)
	}
	return manifest
}

// mustReadPair seals and decodes the staged and live reads the
// report cases aggregate: one operation, two stages. The pair
// rides the sealed decode path, so every report case aggregates
// genuinely validated reads.
func mustReadPair(t *testing.T) (clonereadback.ValidatedReadBack, clonereadback.ValidatedReadBack) {
	t.Helper()
	staged := mustDecodeReadBack(t, stagedAuthority(t), mustBuildReadBack(t, stagedAuthority(t), validReadBackInput()))
	live := mustDecodeReadBack(t, liveAuthority(t), mustBuildReadBack(t, liveAuthority(t), validLiveReadBackInput()))
	if staged.ManifestID() == live.ManifestID() {
		t.Fatal("staged and live reads share an id: the pair is not distinct")
	}
	return staged, live
}

func mustBuildReport(t *testing.T, input clonereadback.ValidationReportInput, staged, live clonereadback.ValidatedReadBack) []byte {
	t.Helper()
	sealed, err := clonereadback.BuildValidationReport(input, staged, live)
	if err != nil {
		t.Fatalf("BuildValidationReport: %v", err)
	}
	return sealed
}

func mustDecodeReport(t *testing.T, sealed []byte, staged, live clonereadback.ValidatedReadBack) clonereadback.ValidationReport {
	t.Helper()
	report, err := clonereadback.DecodeValidationReport(sealed, staged, live)
	if err != nil {
		t.Fatalf("DecodeValidationReport: %v", err)
	}
	return report
}

// decodeDocument parses sealed bytes into a generic document for
// member surgery.
func decodeDocument(t *testing.T, sealed []byte) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(sealed, &document); err != nil {
		t.Fatalf("unmarshal sealed: %v", err)
	}
	return document
}

// marshalDocument renders a surgered document back to bytes.
func marshalDocument(t *testing.T, document map[string]any) []byte {
	t.Helper()
	rendered, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	return rendered
}

// documentAt navigates a nested object member for row-level census
// probes.
func documentAt(t *testing.T, document map[string]any, member string, index int) map[string]any {
	t.Helper()
	rows, ok := document[member].([]any)
	if !ok {
		t.Fatalf("member %s is not an array", member)
	}
	row, ok := rows[index].(map[string]any)
	if !ok {
		t.Fatalf("member %s[%d] is not an object", member, index)
	}
	return row
}
