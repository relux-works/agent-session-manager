package clonereconcile_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/gowebpki/jcs"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/cloneplan"
	"github.com/relux-works/agent-session-manager/internal/clonereadback"
	clonereconcile "github.com/relux-works/agent-session-manager/internal/clonereconcile"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// Shared fixtures for the clonereconcile suite. Every literal is
// hand-written and independent of production constants: verdicts
// always come from production; vectors never do. Owner-shaped
// documents (tuple, binding, findings, plan, projected, reads,
// fidelity rows) are built through the same owner entries
// production calls, so the baseline is valid by construction and
// each test mutates exactly one gate.

// fixtureDigest renders a deterministic pinned-form digest for one
// seed.
func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte("clonereconcile:" + seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// fixtureOperationID renders a deterministic lowercase UUIDv7.
func fixtureOperationID(seed string) string {
	sum := sha256.Sum256([]byte("clonereconcile-operation:" + seed))
	hexed := hex.EncodeToString(sum[:])
	timeField := hexed[:12]
	randA := hexed[12:15]
	randB := hexed[16:28]
	return timeField[:8] + "-" + timeField[8:12] + "-7" + randA + "-8" + randB[:3] + "-" + randB[3:12] + hexed[28:31]
}

// fixtureTuple is the one shared target tuple every paired document
// carries: reads observe it, the plan targets it, and both reports
// bind it.
func fixtureTuple() []byte {
	return []byte(`{"adapter_version":"1.2.3","architecture":"arm64","environment_id":"relux.target","environment_version":"2026.09","platform":"macos","store_schema_fingerprint":"` + fixtureDigest("store-fingerprint") + `"}`)
}

// fixtureTupleOther is a distinct owner-valid tuple for cross-match
// refusal cells.
func fixtureTupleOther() []byte {
	return []byte(`{"adapter_version":"1.2.4","architecture":"amd64","environment_id":"relux.other","environment_version":"2026.09","platform":"linux","store_schema_fingerprint":"` + fixtureDigest("store-fingerprint-other") + `"}`)
}

// fixtureBinding is one owner-valid Workspace Binding document.
func fixtureBinding() []byte {
	first, second := fixtureDigest("remote-a"), fixtureDigest("remote-b")
	if second < first {
		first, second = second, first
	}
	return []byte(`{"branch":"main","cwd_relative":"work/session","extensions":{},"head_digest":"` + fixtureDigest("head") + `","index_digest":null,"logical_workspace_id":"` + fixtureOperationID("workspace") + `","repository_remote_fingerprints":["` + first + `","` + second + `"],"working_tree_digest":null}`)
}

// fixtureFindings renders one owner-valid findings array over the
// given severities.
func fixtureFindings(severities ...string) []byte {
	var rows []string
	for index, severity := range severities {
		rows = append(rows, `{"code":"probe-`+itoa(index)+`","extensions":{},"message":"finding `+itoa(index)+`","remediation":null,"severity":"`+severity+`"}`)
	}
	return []byte("[" + strings.Join(rows, ",") + "]")
}

// fixtureAuthority decodes one owner-valid ReadAuthority carrying
// the given purpose through the sessadapter owner.
func fixtureAuthority(t *testing.T, purpose string) sessadapter.ReadAuthority {
	t.Helper()
	raw := []byte(`{"authority_id":"` + fixtureOperationID("authority-"+purpose) + `","purpose":"` + purpose + `","root_handle_names":["handle-a"],"expires_at":"2026-09-24T00:00:00.000Z","extensions":{}}`)
	authority, err := sessadapter.DecodeReadAuthority(raw)
	if err != nil {
		t.Fatalf("DecodeReadAuthority(%s): %v", purpose, err)
	}
	return authority
}

func strptr(value string) *string { return &value }

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

const fixtureNativeSession = "target-native-session-1"

// evidenceFor renders the evidence rows one sealed read carries for
// the given blob seeds, sorted by (kind, blob) as the read-back
// owner requires.
func evidenceFor(t *testing.T, seeds []string) []clonereadback.EvidenceObjectInput {
	t.Helper()
	rows := make([]clonereadback.EvidenceObjectInput, 0, len(seeds))
	for _, seed := range seeds {
		rows = append(rows, clonereadback.EvidenceObjectInput{
			EvidenceKind:     "native_sample",
			MediaType:        "application/octet-stream",
			ByteCount:        64,
			BlobID:           fixtureDigest(seed),
			BlobDescriptorID: fixtureDigest("descriptor-" + seed),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].BlobID < rows[j].BlobID })
	return rows
}

// sealRead seals one read-back manifest through the read-back owner
// under the authority for its mode, observing the shared target.
func sealRead(t *testing.T, mode, planID, projectedID string, evidence []clonereadback.EvidenceObjectInput) (clonereadback.ValidatedReadBack, []byte) {
	t.Helper()
	return sealReadFull(t, mode, planID, projectedID, evidence, fixtureNativeSession, fixtureTuple())
}

// sealReadFull seals one read-back manifest observing the given
// native session and tuple: pairing cells seal reads against
// neighbor targets while the shared plan/projected stay fixed.
func sealReadFull(t *testing.T, mode, planID, projectedID string, evidence []clonereadback.EvidenceObjectInput, session string, env []byte) (clonereadback.ValidatedReadBack, []byte) {
	t.Helper()
	purpose := "target_staged"
	if mode == "live" {
		purpose = "target_live"
	}
	input := clonereadback.ReadBackManifestInput{
		OperationID:                 fixtureOperationID("read-" + mode + session),
		Mode:                        mode,
		ProjectionPlanID:            planID,
		ProjectedObjectManifestID:   projectedID,
		ExpectedTargetNativeSession: session,
		ObservedTargetNativeSession: session,
		ObservedEnvironment:         env,
		ParsedEventCount:            7,
		ParsedHeadIDs:               []string{"head-a"},
		WorkspaceBinding:            fixtureBinding(),
		StructuralDigest:            fixtureDigest("structural-" + mode),
		EvidenceObjects:             evidence,
		Extensions:                  map[string]any{},
	}
	sealed, bytes, err := clonereadback.BuildReadBackEvidenceManifest(input, fixtureAuthority(t, purpose))
	if err != nil {
		t.Fatalf("BuildReadBackEvidenceManifest(%s): %v", mode, err)
	}
	return sealed, bytes
}

// validPlanInput renders one valid projection plan candidate bound
// to the shared target.
func validPlanInput() cloneplan.ProjectionPlanInput {
	return cloneplan.ProjectionPlanInput{
		OperationID:                 fixtureOperationID("plan"),
		BundleID:                    fixtureOperationID("bundle"),
		RequestDigest:               fixtureDigest("request"),
		SourceSnapshotDigest:        fixtureDigest("snapshot"),
		CaptureManifestID:           fixtureDigest("capture"),
		CanonicalSessionID:          fixtureDigest("canonical"),
		CanonicalEventIDs:           []string{},
		SourceEnvironment:           fixtureTuple(),
		TargetEnvironment:           fixtureTuple(),
		ExpectedTargetNativeSession: fixtureNativeSession,
		TargetWorkspace:             []byte(`{"logical_workspace_id":"` + fixtureOperationID("plan-workspace") + `","cwd_relative":"work/target","repository_remote_fingerprints":[],"branch":null,"head_digest":null,"index_digest":null,"working_tree_digest":null,"extensions":{}}`),
		Strategy:                    "target_native_writer",
		StrategyRationale:           "native writer covers the target tuple",
		FidelityProfile:             "maximal_safe",
		RequiredDispositions:        map[string][]string{"durable_payload": {"exact"}},
		ForbidReasons:               []string{},
		ItemMappings: []cloneplan.ItemMappingInput{{
			SourceItemKey:     "a-item",
			CanonicalObjectID: strptr(fixtureDigest("canonical-a-item")),
			TargetResourceKey: []string{"res/a-item"},
			ExpectedDispos:    "exact",
			ReasonCodes:       []string{},
			Extensions:        map[string]any{},
		}},
		TargetOperations: []cloneplan.TargetOperationInput{{
			Sequence:           1,
			Action:             "write_blob",
			ResourceKeys:       []string{"res/op"},
			DependsOnSequences: nil,
			Extensions:         map[string]any{},
		}},
		ExpectedResources: []cloneplan.ExpectedResourceInput{{
			OperationSequence: 1,
			ResourceKey:       "a-res",
			Kind:              "blob",
			Mode:              nil,
			ExpectedBlobID:    strptr(fixtureDigest("blob-a-res")),
			Extensions:        map[string]any{},
		}},
		SynthesizedEvents:        []cloneplan.SynthesizedEventInput{},
		SecurityExclusions:       []string{},
		ResourceLimits:           []byte(`{"max_objects":100,"max_total_bytes":1000000,"max_single_object_bytes":65536,"max_events":100,"max_target_resources":100}`),
		TransactionPlan:          cloneplan.TransactionPlanInput{MaterializationIntent: "clone", TargetCollisionPolicy: "must_be_absent", Activation: "dormant_validated", Extensions: map[string]any{}},
		ReadBackPlan:             cloneplan.ReadBackPlanInput{Modes: []string{"staged", "live"}, RequireIdentityMatch: true, RequireWorkspaceMatch: true, RequireSemanticMarker: true, Extensions: map[string]any{}},
		ResumePlan:               cloneplan.ResumePlanInput{OpensExistingIdentity: true, AllowBlankFallback: false, BoundedContinuationTurnRequired: true, Extensions: map[string]any{}},
		RollbackPlan:             cloneplan.RollbackPlanInput{Required: true, RetainThrough: "live_validated", ForbiddenAfterProviderCommit: true, Extensions: map[string]any{}},
		RequiredContracts:        []cloneplan.ContractRequirementInput{{ContractID: "ax.contract.plan", Version: "1.2.3"}},
		RequiredCapabilities:     []string{"clone.target.write"},
		FidelityBasisDigest:      fixtureDigest("basis"),
		SourceAdapterBuildDigest: fixtureDigest("source-build"),
		TargetAdapterBuildDigest: fixtureDigest("target-build"),
		ControllerBuildDigest:    fixtureDigest("controller-build"),
		Extensions:               map[string]any{},
	}
}

// mustBuildPlan seals one projection plan through the plan owner.
func mustBuildPlan(t *testing.T, input cloneplan.ProjectionPlanInput) []byte {
	t.Helper()
	sealed, err := cloneplan.BuildProjectionPlan(input)
	if err != nil {
		t.Fatalf("BuildProjectionPlan() error = %v", err)
	}
	return sealed
}

// mustDecodePlan decodes one sealed plan through the plan owner.
func mustDecodePlan(t *testing.T, sealed []byte) cloneplan.ProjectionPlan {
	t.Helper()
	plan, err := cloneplan.DecodeProjectionPlan(sealed)
	if err != nil {
		t.Fatalf("DecodeProjectionPlan() error = %v", err)
	}
	return plan
}

// validProjectedInput renders one valid projected manifest candidate
// bound to the given plan digest and the shared target.
func validProjectedInput(planID string) cloneplan.ProjectedManifestInput {
	return cloneplan.ProjectedManifestInput{
		OperationID:                 fixtureOperationID("projected"),
		ProjectionPlanID:            planID,
		TargetEnvironment:           fixtureTuple(),
		ExpectedTargetNativeSession: fixtureNativeSession,
		Entries: []cloneplan.ProjectedEntryInput{
			{OperationSequence: 1, ResourceKey: "a-dir", Kind: "directory", Mode: u64ptr(0755)},
			{OperationSequence: 1, ResourceKey: "b-blob", Kind: "blob", Mode: nil, ByteCount: u64ptr(128), BlobID: strptr(fixtureDigest("blob-b-blob")), BlobDescriptorID: strptr(fixtureDigest("descriptor-b-blob"))},
		},
		TotalBytes: 128,
		Extensions: map[string]any{},
	}
}

func u64ptr(value uint64) *uint64 { return &value }

// mustBuildProjected seals one projected manifest through the plan
// owner.
func mustBuildProjected(t *testing.T, input cloneplan.ProjectedManifestInput) []byte {
	t.Helper()
	sealed, err := cloneplan.BuildProjectedObjectManifest(input)
	if err != nil {
		t.Fatalf("BuildProjectedObjectManifest() error = %v", err)
	}
	return sealed
}

// mustDecodeProjected decodes one sealed projected manifest through
// the plan owner.
func mustDecodeProjected(t *testing.T, sealed []byte) cloneplan.ProjectedManifest {
	t.Helper()
	manifest, err := cloneplan.DecodeProjectedObjectManifest(sealed)
	if err != nil {
		t.Fatalf("DecodeProjectedObjectManifest() error = %v", err)
	}
	return manifest
}

// rowReasons renders the default reason set for one disposition:
// exact rows carry none, every other row carries one core reason.
func rowReasons(disposition string) []string {
	if disposition == "exact" {
		return []string{}
	}
	return []string{"unknown_native_event"}
}

// validRowInput renders one valid fidelity row candidate over the
// given source key, canonical digest (nil for normalized rows and
// synthesized rows), disposition, and staged/live evidence seeds.
// Source evidence always names the candidate's raw digest; the
// caller passes the raw seed explicitly so tier-1 and tier-3 stay
// in agreement by construction.
func validRowInput(key, rawSeed string, canonical *string, disposition string, stagedSeeds, liveSeeds []string) clonefidelity.DispositionRecordInput {
	staged := make([]string, 0, len(stagedSeeds))
	for _, seed := range stagedSeeds {
		staged = append(staged, fixtureDigest(seed))
	}
	sort.Strings(staged)
	live := make([]string, 0, len(liveSeeds))
	for _, seed := range liveSeeds {
		live = append(live, fixtureDigest(seed))
	}
	sort.Strings(live)
	return clonefidelity.DispositionRecordInput{
		SourceItemKey:           key,
		SourceClass:             "durable_payload",
		SourceEvidenceIDs:       []string{fixtureDigest(rawSeed)},
		CanonicalObjectID:       canonical,
		TargetLocator:           strptr("target/" + key),
		Disposition:             disposition,
		ReasonCodes:             rowReasons(disposition),
		Explanation:             "row " + key,
		StagedEvidenceObjectIDs: staged,
		LiveEvidenceObjectIDs:   live,
		Extensions:              map[string]any{},
	}
}

// synthRowInput renders one valid synthesized-row candidate: no
// source canonical object, source evidence naming a digest outside
// the captured raw set, and optional resolving evidence.
func synthRowInput(key, evidenceSeed string, stagedSeeds, liveSeeds []string) clonefidelity.DispositionRecordInput {
	row := validRowInput(key, evidenceSeed, nil, "synthesized", stagedSeeds, liveSeeds)
	row.ReasonCodes = []string{"derived_index_rebuilt"}
	return row
}

// concentratedBreakdowns derives reconciling breakdown maps that
// concentrate every row on the first owner key, hand-counted
// independently of production derivation.
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

// validFidelityInput renders one valid target-scope fidelity report
// candidate over the given rows bound to the sealed reads.
func validFidelityInput(rows []clonefidelity.DispositionRecordInput, stagedID, liveID string) clonefidelity.FidelityReportInput {
	events, blocks, byteCounts := concentratedBreakdowns(rows)
	return clonefidelity.FidelityReportInput{
		Scope:                            "target",
		OperationID:                      fixtureOperationID("fidelity"),
		BundleID:                         fixtureOperationID("bundle"),
		SourceSnapshotDigest:             fixtureDigest("source-snapshot"),
		CaptureManifestID:                fixtureDigest("capture-manifest"),
		CanonicalSessionID:               fixtureDigest("canonical-session"),
		ProjectionPlanID:                 strptr(fixtureDigest("projection-plan")),
		SourceEnvironment:                fixtureTuple(),
		TargetEnvironment:                fixtureTuple(),
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
		StagedReadBackEvidenceManifestID: strptr(stagedID),
		LiveReadBackEvidenceManifestID:   strptr(liveID),
		AdapterAttestations:              []string{},
		Extensions:                       map[string]any{},
	}
}

// validValidationFields renders one valid validation member set:
// all checks true, no findings, the shared target.
func validValidationFields(expectedFidelityID string) clonereconcile.ValidationFields {
	return clonereconcile.ValidationFields{
		OperationID:                 fixtureOperationID("validation"),
		ProviderManifestID:          fixtureDigest("provider-manifest"),
		ExpectedTargetNativeSession: fixtureNativeSession,
		ObservedTargetNativeSession: fixtureNativeSession,
		TargetEnvironment:           fixtureTuple(),
		Checks: clonereadback.ValidationChecks{
			StagedStructuralValid: true, LiveStructuralValid: true,
			SemanticMarkerValid: true, IdentityValid: true,
			WorkspaceBindingValid: true, ResumeSurfaceValid: true,
			SourceGenerationRevalidated: true,
		},
		Findings:                 fixtureFindings(),
		Extensions:               map[string]any{},
		ExpectedFidelityReportID: expectedFidelityID,
	}
}

// requireRefusal asserts the refusal contract: the stable ErrInvalid
// code plus the literal detail. The literal pins the gate; the code
// pins the refusal identity.
func requireRefusal(t *testing.T, err error, literal string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want refusal containing %q", literal)
	}
	if !errors.Is(err, clonereconcile.ErrInvalid) {
		t.Fatalf("error = %v, want errors.Is ErrInvalid", err)
	}
	if !strings.Contains(err.Error(), literal) {
		t.Fatalf("error = %q, want literal %q", err.Error(), literal)
	}
}

// independentFidelityID recomputes the omit-self identity in the
// test: the sealed document minus fidelity_report_id, JCS-hashed
// with SHA-256. It shares no production helper.
func independentFidelityID(t *testing.T, sealed []byte) string {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(sealed, &document); err != nil {
		t.Fatalf("decode sealed fidelity report: %v", err)
	}
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
