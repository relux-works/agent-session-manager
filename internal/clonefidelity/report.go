package clonefidelity

import (
	"encoding/json"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Fidelity Report 1.0.0: the
// closed archive/target report Section 13.14.2 states with its
// per-item disposition rows and reconciled aggregate maps. Counts
// and reason counts derive from the rows and are never accepted;
// the per-kind and per-block breakdowns reconcile to those derived
// totals per disposition; byte values admit the exact
// seven-disposition key set with uint53 values (rows carry no byte
// measure, so no row reconciliation is expressible — a stated
// bound).

const (
	fidelityReportSchema  = "urn:ax:schema:fidelity-report"
	fidelityReportVersion = "1.0.0"
	fidelityReportSelf    = "fidelity_report_id"
)

// maxReportRows is the Fidelity Report dispositions bound:
// FidelityDispositionRecord[1..1000000].
const maxReportRows = 1000000

var fidelityReportMembers = map[string]bool{
	"schema": true, "schema_version": true, "fidelity_report_id": true,
	"scope": true, "operation_id": true, "bundle_id": true,
	"source_snapshot_digest": true, "capture_manifest_id": true, "canonical_session_id": true,
	"projection_plan_id": true, "source_environment": true, "target_environment": true,
	"profile": true, "required_dispositions": true, "forbid_reasons": true,
	"dispositions": true, "counts": true, "event_kind_counts": true,
	"content_block_counts": true, "byte_counts": true, "reason_counts": true,
	"raw_bundle_complete": true, "canonical_complete": true,
	"target_semantically_continuable": true, "target_natively_resumable": true,
	"staged_read_back_evidence_manifest_id": true, "live_read_back_evidence_manifest_id": true,
	"adapter_attestations": true, "extensions": true,
}

var fidelityReportRequired = []string{
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

// FidelityCounts is the exact seven-member count record Section
// 13.14.2 states: exact, semantic, summarized, opaque_preserved,
// synthesized, omitted, and unrecoverable.
type FidelityCounts struct {
	Exact           uint64
	Semantic        uint64
	Summarized      uint64
	OpaquePreserved uint64
	Synthesized     uint64
	Omitted         uint64
	Unrecoverable   uint64
}

// countOf returns the count for one disposition. The caller gates
// vocabulary; an unknown disposition reads zero.
func (counts FidelityCounts) countOf(disposition string) uint64 {
	switch disposition {
	case "exact":
		return counts.Exact
	case "semantic":
		return counts.Semantic
	case "summarized":
		return counts.Summarized
	case "opaque_preserved":
		return counts.OpaquePreserved
	case "synthesized":
		return counts.Synthesized
	case "omitted":
		return counts.Omitted
	case "unrecoverable":
		return counts.Unrecoverable
	default:
		return 0
	}
}

// addRow folds one row disposition into the totals. Every counts
// derivation both entries share flows through this switch.
func (counts *FidelityCounts) addRow(disposition string) {
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
}

func (counts FidelityCounts) object() map[string]any {
	return map[string]any{
		"exact":            counts.Exact,
		"semantic":         counts.Semantic,
		"summarized":       counts.Summarized,
		"opaque_preserved": counts.OpaquePreserved,
		"synthesized":      counts.Synthesized,
		"omitted":          counts.Omitted,
		"unrecoverable":    counts.Unrecoverable,
	}
}

// deriveCounts counts the rows per disposition. Row vocabularies are
// gated before derivation, so every row folds into exactly one arm.
func deriveCounts(rows []DispositionRecord) FidelityCounts {
	var counts FidelityCounts
	for _, row := range rows {
		counts.addRow(row.Disposition)
	}
	return counts
}

// deriveReasonCounts counts, per reason, the rows naming it. Row
// reason sets are sorted unique, so each row contributes at most
// one to each reason.
func deriveReasonCounts(rows []DispositionRecord) map[string]uint64 {
	derived := map[string]uint64{}
	for _, row := range rows {
		for _, reason := range row.ReasonCodes {
			derived[reason]++
		}
	}
	return derived
}

// checkRowCount admits the dispositions cardinality both entries
// share: FidelityDispositionRecord[1..1000000]. It is factored so
// the exact bound is pinned at the edge and edge+1 without sealing
// a million-row report in the suite; entry reachability is proven
// by the entry-level refusal rows (see TRACEABILITY.md).
func checkRowCount(count int) error {
	if count < 1 || count > maxReportRows {
		return invalid("fidelity report carries %d dispositions, want [1..1000000]", count)
	}
	return nil
}

// checkRowOrder enforces the row ordering both entries share:
// source rows sorted unique by item key, then synthesized rows
// sorted unique by item key, with keys unique across all rows.
func checkRowOrder(rows []DispositionRecord) error {
	seen := make(map[string]bool, len(rows))
	seenSynthesized := false
	previousSource := ""
	previousSynthesized := ""
	for _, row := range rows {
		if seen[row.SourceItemKey] {
			return invalid("fidelity report dispositions carry duplicate source item key %q", row.SourceItemKey)
		}
		seen[row.SourceItemKey] = true
		if row.Disposition == "synthesized" {
			seenSynthesized = true
			if previousSynthesized != "" && row.SourceItemKey <= previousSynthesized {
				return invalid("fidelity report synthesized dispositions are not sorted unique by source item key")
			}
			previousSynthesized = row.SourceItemKey
			continue
		}
		if seenSynthesized {
			return invalid("fidelity report dispositions order a source row after synthesized rows")
		}
		if previousSource != "" && row.SourceItemKey <= previousSource {
			return invalid("fidelity report source dispositions are not sorted unique by source item key")
		}
		previousSource = row.SourceItemKey
	}
	return nil
}

// checkBreakdownSums reconciles one closed breakdown map to the
// derived totals: for every disposition, the values summed over
// every key equal the row-derived count. The sums cannot overflow:
// 26 closed keys times (2^53-1) stays below 2^57.
func checkBreakdownSums(owner string, breakdown map[string]FidelityCounts, totals FidelityCounts) error {
	for _, disposition := range dispositions {
		var sum uint64
		for _, counts := range breakdown {
			sum += counts.countOf(disposition)
		}
		if sum != totals.countOf(disposition) {
			return invalid("%s reconciles %d %s rows, want %d from the disposition rows", owner, sum, disposition, totals.countOf(disposition))
		}
	}
	return nil
}

// FidelityReport is one validated Fidelity Report 1.0.0.
type FidelityReport struct {
	ReportID                         scalar.Digest
	Scope                            string
	OperationID                      scalar.UUIDv7
	BundleID                         scalar.UUIDv7
	SourceSnapshotDigest             scalar.Digest
	CaptureManifestID                scalar.Digest
	CanonicalSessionID               scalar.Digest
	ProjectionPlanID                 *scalar.Digest
	SourceEnvironment                sessadapter.Tuple
	TargetEnvironment                *sessadapter.Tuple
	Profile                          string
	RequiredDispositions             map[string][]string
	ForbidReasons                    []string
	Rows                             []DispositionRecord
	Counts                           FidelityCounts
	EventKindCounts                  map[string]FidelityCounts
	ContentBlockCounts               map[string]FidelityCounts
	ByteCounts                       map[string]uint64
	ReasonCounts                     map[string]uint64
	RawBundleComplete                bool
	CanonicalComplete                bool
	TargetSemanticallyContinuable    bool
	TargetNativelyResumable          bool
	StagedReadBackEvidenceManifestID *scalar.Digest
	LiveReadBackEvidenceManifestID   *scalar.Digest
	AdapterAttestations              []scalar.Digest
}

// FidelityReportInput is the caller-supplied fidelity report
// candidate for Build. Counts and reason counts are derived from
// Rows and are never accepted; the per-kind, per-block, and
// per-disposition breakdowns are caller-supplied and reconciled.
// Nil digest pointers and empty TargetEnvironment seal as null.
type FidelityReportInput struct {
	Scope                            string
	OperationID                      string
	BundleID                         string
	SourceSnapshotDigest             string
	CaptureManifestID                string
	CanonicalSessionID               string
	ProjectionPlanID                 *string
	SourceEnvironment                []byte
	TargetEnvironment                []byte
	Profile                          string
	RequiredDispositions             map[string][]string
	ForbidReasons                    []string
	Rows                             []DispositionRecordInput
	EventKindCounts                  map[string]FidelityCounts
	ContentBlockCounts               map[string]FidelityCounts
	ByteCounts                       map[string]uint64
	RawBundleComplete                bool
	CanonicalComplete                bool
	TargetSemanticallyContinuable    bool
	TargetNativelyResumable          bool
	StagedReadBackEvidenceManifestID *string
	LiveReadBackEvidenceManifestID   *string
	AdapterAttestations              []string
	Extensions                       map[string]any
}

// BuildFidelityReport constructs one closed Fidelity Report 1.0.0
// as canonical bytes. Identical inputs produce byte-identical
// outputs. Counts and reason counts derive from the rows; every
// other closed-shape, vocabulary, branch, order, and reconciliation
// rule is a refusal, never a repair.
func BuildFidelityReport(input FidelityReportInput) ([]byte, error) {
	_, object, err := buildFidelityReport(input)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	reportID := scalar.SHA256Digest(omitted)
	object[fidelityReportSelf] = reportID.String()
	return canonicalizeObject(object)
}

func buildFidelityReport(input FidelityReportInput) (FidelityReport, map[string]any, error) {
	owner := "fidelity report"
	archive := false
	switch input.Scope {
	case "archive":
		archive = true
	case "target":
	default:
		return FidelityReport{}, nil, invalid("%s scope is outside archive|target", owner)
	}
	operation, err := scalar.ParseUUIDv7(input.OperationID)
	if err != nil {
		return FidelityReport{}, nil, invalid("%s operation_id is not a UUIDv7: %v", owner, err)
	}
	bundle, err := scalar.ParseUUIDv7(input.BundleID)
	if err != nil {
		return FidelityReport{}, nil, invalid("%s bundle_id is not a UUIDv7: %v", owner, err)
	}
	snapshot, err := scalar.ParseDigest(input.SourceSnapshotDigest)
	if err != nil {
		return FidelityReport{}, nil, invalid("%s source_snapshot_digest is not a digest: %v", owner, err)
	}
	captureID, err := scalar.ParseDigest(input.CaptureManifestID)
	if err != nil {
		return FidelityReport{}, nil, invalid("%s capture_manifest_id is not a digest: %v", owner, err)
	}
	canonicalID, err := scalar.ParseDigest(input.CanonicalSessionID)
	if err != nil {
		return FidelityReport{}, nil, invalid("%s canonical_session_id is not a digest: %v", owner, err)
	}
	var plan *scalar.Digest
	var planValue any
	if archive {
		if input.ProjectionPlanID != nil {
			return FidelityReport{}, nil, invalid("%s projection_plan_id is not null for archive", owner)
		}
	} else {
		if input.ProjectionPlanID == nil {
			return FidelityReport{}, nil, invalid("%s projection_plan_id is null for target", owner)
		}
		id, err := scalar.ParseDigest(*input.ProjectionPlanID)
		if err != nil {
			return FidelityReport{}, nil, invalid("%s projection_plan_id is not a digest: %v", owner, err)
		}
		plan = &id
		planValue = id.String()
	}
	sourceTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.SourceEnvironment))
	if err != nil {
		return FidelityReport{}, nil, invalid("%s source_environment is not an Environment Tuple: %v", owner, err)
	}
	var targetTuple *sessadapter.Tuple
	var targetValue any
	if archive {
		if len(input.TargetEnvironment) != 0 {
			return FidelityReport{}, nil, invalid("%s target_environment is not null for archive", owner)
		}
	} else {
		tuple, err := sessadapter.DecodeTuple(json.RawMessage(input.TargetEnvironment))
		if err != nil {
			return FidelityReport{}, nil, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
		}
		targetTuple = &tuple
		targetValue = tupleObject(tuple)
	}
	if !ValidProfile(input.Profile) {
		return FidelityReport{}, nil, invalid("%s profile %q is outside strict_exact|maximal_safe|compact|messages_only|archive_only", owner, input.Profile)
	}
	if archive && input.Profile != "archive_only" {
		return FidelityReport{}, nil, invalid("%s profile is %q for archive, want archive_only", owner, input.Profile)
	}
	if !archive && input.Profile == "archive_only" {
		return FidelityReport{}, nil, invalid("%s profile is archive_only for target", owner)
	}
	required, err := buildRequiredDispositions(input.RequiredDispositions)
	if err != nil {
		return FidelityReport{}, nil, err
	}
	forbidden, err := checkSortedUniqueBoundedStrings(input.ForbidReasons, 1, 128, 0, 128, owner, "forbid_reasons")
	if err != nil {
		return FidelityReport{}, nil, err
	}
	if err := checkRowCount(len(input.Rows)); err != nil {
		return FidelityReport{}, nil, err
	}
	rows := make([]DispositionRecord, 0, len(input.Rows))
	objects := make([]any, 0, len(input.Rows))
	for index, candidate := range input.Rows {
		row, object, err := buildDispositionRecord(candidate, index)
		if err != nil {
			return FidelityReport{}, nil, err
		}
		rows = append(rows, row)
		objects = append(objects, object)
	}
	if err := checkRowOrder(rows); err != nil {
		return FidelityReport{}, nil, err
	}
	counts := deriveCounts(rows)
	reasonCounts := deriveReasonCounts(rows)
	eventKinds, err := buildBreakdown(input.EventKindCounts, clonebundle.EventKinds(), "event_kind_counts")
	if err != nil {
		return FidelityReport{}, nil, err
	}
	if err := checkBreakdownSums("fidelity report event_kind_counts", eventKinds, counts); err != nil {
		return FidelityReport{}, nil, err
	}
	contentBlocks, err := buildBreakdown(input.ContentBlockCounts, clonebundle.ContentBlockTypes(), "content_block_counts")
	if err != nil {
		return FidelityReport{}, nil, err
	}
	if err := checkBreakdownSums("fidelity report content_block_counts", contentBlocks, counts); err != nil {
		return FidelityReport{}, nil, err
	}
	byteCounts, err := buildByteCounts(input.ByteCounts)
	if err != nil {
		return FidelityReport{}, nil, err
	}
	if archive {
		if input.TargetSemanticallyContinuable {
			return FidelityReport{}, nil, invalid("%s target_semantically_continuable is true for archive", owner)
		}
		if input.TargetNativelyResumable {
			return FidelityReport{}, nil, invalid("%s target_natively_resumable is true for archive", owner)
		}
	}
	var staged *scalar.Digest
	var stagedValue any
	var live *scalar.Digest
	var liveValue any
	if archive {
		if input.StagedReadBackEvidenceManifestID != nil {
			return FidelityReport{}, nil, invalid("%s staged_read_back_evidence_manifest_id is not null for archive", owner)
		}
		if input.LiveReadBackEvidenceManifestID != nil {
			return FidelityReport{}, nil, invalid("%s live_read_back_evidence_manifest_id is not null for archive", owner)
		}
	} else {
		if input.StagedReadBackEvidenceManifestID == nil {
			return FidelityReport{}, nil, invalid("%s staged_read_back_evidence_manifest_id is null for target", owner)
		}
		stagedID, err := scalar.ParseDigest(*input.StagedReadBackEvidenceManifestID)
		if err != nil {
			return FidelityReport{}, nil, invalid("%s staged_read_back_evidence_manifest_id is not a digest: %v", owner, err)
		}
		staged = &stagedID
		stagedValue = stagedID.String()
		if input.LiveReadBackEvidenceManifestID == nil {
			return FidelityReport{}, nil, invalid("%s live_read_back_evidence_manifest_id is null for target", owner)
		}
		liveID, err := scalar.ParseDigest(*input.LiveReadBackEvidenceManifestID)
		if err != nil {
			return FidelityReport{}, nil, invalid("%s live_read_back_evidence_manifest_id is not a digest: %v", owner, err)
		}
		live = &liveID
		liveValue = liveID.String()
	}
	attestations, err := parseSortedUniqueDigestStrings(input.AdapterAttestations, 0, 64, owner, "adapter_attestations")
	if err != nil {
		return FidelityReport{}, nil, err
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return FidelityReport{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	report := FidelityReport{
		Scope:                            input.Scope,
		OperationID:                      operation,
		BundleID:                         bundle,
		SourceSnapshotDigest:             snapshot,
		CaptureManifestID:                captureID,
		CanonicalSessionID:               canonicalID,
		ProjectionPlanID:                 plan,
		SourceEnvironment:                sourceTuple,
		TargetEnvironment:                targetTuple,
		Profile:                          input.Profile,
		RequiredDispositions:             required,
		ForbidReasons:                    forbidden,
		Rows:                             rows,
		Counts:                           counts,
		EventKindCounts:                  eventKinds,
		ContentBlockCounts:               contentBlocks,
		ByteCounts:                       byteCounts,
		ReasonCounts:                     reasonCounts,
		RawBundleComplete:                input.RawBundleComplete,
		CanonicalComplete:                input.CanonicalComplete,
		TargetSemanticallyContinuable:    input.TargetSemanticallyContinuable,
		TargetNativelyResumable:          input.TargetNativelyResumable,
		StagedReadBackEvidenceManifestID: staged,
		LiveReadBackEvidenceManifestID:   live,
		AdapterAttestations:              attestations,
	}
	object := map[string]any{
		"schema":                                fidelityReportSchema,
		"schema_version":                        fidelityReportVersion,
		"scope":                                 input.Scope,
		"operation_id":                          operation.String(),
		"bundle_id":                             bundle.String(),
		"source_snapshot_digest":                snapshot.String(),
		"capture_manifest_id":                   captureID.String(),
		"canonical_session_id":                  canonicalID.String(),
		"projection_plan_id":                    planValue,
		"source_environment":                    tupleObject(sourceTuple),
		"target_environment":                    targetValue,
		"profile":                               input.Profile,
		"required_dispositions":                 requiredObject(required),
		"forbid_reasons":                        stringValues(forbidden),
		"dispositions":                          objects,
		"counts":                                counts.object(),
		"event_kind_counts":                     breakdownObject(eventKinds),
		"content_block_counts":                  breakdownObject(contentBlocks),
		"byte_counts":                           byteCountsObject(byteCounts),
		"reason_counts":                         reasonCountsObject(reasonCounts),
		"raw_bundle_complete":                   input.RawBundleComplete,
		"canonical_complete":                    input.CanonicalComplete,
		"target_semantically_continuable":       input.TargetSemanticallyContinuable,
		"target_natively_resumable":             input.TargetNativelyResumable,
		"staged_read_back_evidence_manifest_id": stagedValue,
		"live_read_back_evidence_manifest_id":   liveValue,
		"adapter_attestations":                  digestStrings(attestations),
		"extensions":                            extensions,
	}
	return report, object, nil
}

// buildRequiredDispositions validates the caller-supplied policy
// map: closed capture-class keys (the spec word for a closed class
// vocabulary) to sorted unique non-empty disposition sets. Keys
// validate in sorted order so multi-key refusals are deterministic.
func buildRequiredDispositions(input map[string][]string) (map[string][]string, error) {
	owner := "fidelity report required_dispositions"
	keys := make([]string, 0, len(input))
	for class := range input {
		keys = append(keys, class)
	}
	sort.Strings(keys)
	required := make(map[string][]string, len(input))
	for _, class := range keys {
		if !clonebundle.ValidCaptureClass(class) {
			return nil, invalid("%s carry unknown class %q", owner, class)
		}
		values := input[class]
		if len(values) < 1 {
			return nil, invalid("%s[%q] carry no dispositions, want a non-empty set", owner, class)
		}
		previous := ""
		for index, disposition := range values {
			if !ValidDisposition(disposition) {
				return nil, invalid("%s[%q][%d] disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, class, index)
			}
			if index > 0 && disposition <= previous {
				return nil, invalid("%s[%q] are not sorted unique", owner, class)
			}
			previous = disposition
		}
		required[class] = append([]string(nil), values...)
	}
	return required, nil
}

// buildBreakdown validates one caller-supplied closed breakdown map:
// exactly the owner vocabulary keys, each an exact FidelityCounts
// of uint53 values.
func buildBreakdown(input map[string]FidelityCounts, keys []string, member string) (map[string]FidelityCounts, error) {
	owner := "fidelity report " + member
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	for key := range input {
		if !allowed[key] {
			return nil, invalid("%s carry unknown key %q", owner, key)
		}
	}
	breakdown := make(map[string]FidelityCounts, len(keys))
	for _, key := range keys {
		counts, present := input[key]
		if !present {
			return nil, invalid("%s miss key %q", owner, key)
		}
		for _, disposition := range dispositions {
			if counts.countOf(disposition) > maxUint53 {
				return nil, invalid("%s[%q] %s exceeds uint53", owner, key, disposition)
			}
		}
		breakdown[key] = counts
	}
	return breakdown, nil
}

// buildByteCounts validates the caller-supplied byte map: exactly
// the seven dispositions to uint53 values. Values admit any uint53:
// rows carry no byte measure, so no row reconciliation is
// expressible (a stated bound).
func buildByteCounts(input map[string]uint64) (map[string]uint64, error) {
	owner := "fidelity report byte_counts"
	for key := range input {
		if !ValidDisposition(key) {
			return nil, invalid("%s carry unknown disposition %q", owner, key)
		}
	}
	counts := make(map[string]uint64, len(dispositions))
	for _, disposition := range dispositions {
		value, present := input[disposition]
		if !present {
			return nil, invalid("%s miss disposition %q", owner, disposition)
		}
		if value > maxUint53 {
			return nil, invalid("%s[%q] exceeds uint53", owner, disposition)
		}
		counts[disposition] = value
	}
	return counts, nil
}

func tupleObject(tuple sessadapter.Tuple) map[string]any {
	return map[string]any{
		"environment_id":           tuple.EnvironmentID,
		"environment_version":      tuple.Version,
		"platform":                 tuple.Platform,
		"architecture":             tuple.Architecture,
		"store_schema_fingerprint": tuple.StoreFingerprint,
		"adapter_version":          tuple.AdapterVersion,
	}
}

func requiredObject(required map[string][]string) map[string]any {
	object := make(map[string]any, len(required))
	for class, values := range required {
		object[class] = stringValues(values)
	}
	return object
}

func breakdownObject(breakdown map[string]FidelityCounts) map[string]any {
	object := make(map[string]any, len(breakdown))
	for key, counts := range breakdown {
		object[key] = counts.object()
	}
	return object
}

func byteCountsObject(counts map[string]uint64) map[string]any {
	object := make(map[string]any, len(counts))
	for disposition, value := range counts {
		object[disposition] = value
	}
	return object
}

func reasonCountsObject(counts map[string]uint64) map[string]any {
	object := make(map[string]any, len(counts))
	for reason, value := range counts {
		object[reason] = value
	}
	return object
}

// DecodeFidelityReport validates one closed Fidelity Report 1.0.0:
// exact members, schema/version, scalar bounds, closed
// vocabularies, sorted-unique arrays, the archive/target
// nullability branches, row ordering, aggregate reconciliation
// against the rows, and self-digest agreement.
func DecodeFidelityReport(data []byte) (FidelityReport, error) {
	owner := "fidelity report"
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return FidelityReport{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, fidelityReportMembers); unknown {
		return FidelityReport{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, fidelityReportRequired); missing {
		return FidelityReport{}, invalid("%s misses a required member %q", owner, name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != fidelityReportSchema {
		return FidelityReport{}, invalid("%s schema is not urn:ax:schema:fidelity-report", owner)
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != fidelityReportVersion {
		return FidelityReport{}, invalid("%s schema_version is not 1.0.0", owner)
	}
	reportID, ok := checkDigest(members[fidelityReportSelf])
	if !ok {
		return FidelityReport{}, invalid("%s fidelity_report_id is not a digest", owner)
	}
	scope, ok := rawString(members["scope"])
	if !ok {
		return FidelityReport{}, invalid("%s scope is outside archive|target", owner)
	}
	archive := false
	switch scope {
	case "archive":
		archive = true
	case "target":
	default:
		return FidelityReport{}, invalid("%s scope is outside archive|target", owner)
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return FidelityReport{}, invalid("%s operation_id is not a UUIDv7", owner)
	}
	bundle, ok := checkUUIDv7(members["bundle_id"])
	if !ok {
		return FidelityReport{}, invalid("%s bundle_id is not a UUIDv7", owner)
	}
	snapshot, ok := checkDigest(members["source_snapshot_digest"])
	if !ok {
		return FidelityReport{}, invalid("%s source_snapshot_digest is not a digest", owner)
	}
	captureID, ok := checkDigest(members["capture_manifest_id"])
	if !ok {
		return FidelityReport{}, invalid("%s capture_manifest_id is not a digest", owner)
	}
	canonicalID, ok := checkDigest(members["canonical_session_id"])
	if !ok {
		return FidelityReport{}, invalid("%s canonical_session_id is not a digest", owner)
	}
	var plan *scalar.Digest
	if archive {
		if !isNull(members["projection_plan_id"]) {
			return FidelityReport{}, invalid("%s projection_plan_id is not null for archive", owner)
		}
	} else {
		id, ok := checkDigest(members["projection_plan_id"])
		if !ok {
			return FidelityReport{}, invalid("%s projection_plan_id is not a digest for target", owner)
		}
		plan = &id
	}
	sourceTuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil {
		return FidelityReport{}, invalid("%s source_environment is not an Environment Tuple: %v", owner, err)
	}
	var targetTuple *sessadapter.Tuple
	if archive {
		if !isNull(members["target_environment"]) {
			return FidelityReport{}, invalid("%s target_environment is not null for archive", owner)
		}
	} else {
		tuple, err := sessadapter.DecodeTuple(members["target_environment"])
		if err != nil {
			return FidelityReport{}, invalid("%s target_environment is not an Environment Tuple: %v", owner, err)
		}
		targetTuple = &tuple
	}
	profile, ok := rawString(members["profile"])
	if !ok || !ValidProfile(profile) {
		return FidelityReport{}, invalid("%s profile is outside strict_exact|maximal_safe|compact|messages_only|archive_only", owner)
	}
	if archive && profile != "archive_only" {
		return FidelityReport{}, invalid("%s profile is %q for archive, want archive_only", owner, profile)
	}
	if !archive && profile == "archive_only" {
		return FidelityReport{}, invalid("%s profile is archive_only for target", owner)
	}
	required, err := decodeRequiredDispositions(members["required_dispositions"])
	if err != nil {
		return FidelityReport{}, err
	}
	forbidden, ok := checkSortedUniqueStrings(members["forbid_reasons"], 1, 128, 0, 128)
	if !ok {
		return FidelityReport{}, invalid("%s forbid_reasons are not sorted unique string[1..128][0..128]", owner)
	}
	elements, ok := decodeArray(members["dispositions"])
	if !ok {
		return FidelityReport{}, invalid("%s dispositions are not an array", owner)
	}
	if err := checkRowCount(len(elements)); err != nil {
		return FidelityReport{}, err
	}
	rows := make([]DispositionRecord, 0, len(elements))
	for index, element := range elements {
		row, err := decodeDispositionRecord(element, index)
		if err != nil {
			return FidelityReport{}, err
		}
		rows = append(rows, row)
	}
	if err := checkRowOrder(rows); err != nil {
		return FidelityReport{}, err
	}
	counts := deriveCounts(rows)
	sealedCounts, err := decodeFidelityCounts(members["counts"])
	if err != nil {
		return FidelityReport{}, err
	}
	for _, disposition := range dispositions {
		if sealedCounts.countOf(disposition) != counts.countOf(disposition) {
			return FidelityReport{}, invalid("%s counts %s reconciles %d rows, want %d from the disposition rows", owner, disposition, sealedCounts.countOf(disposition), counts.countOf(disposition))
		}
	}
	eventKinds, err := decodeBreakdown(members["event_kind_counts"], clonebundle.EventKinds(), "event_kind_counts")
	if err != nil {
		return FidelityReport{}, err
	}
	if err := checkBreakdownSums("fidelity report event_kind_counts", eventKinds, counts); err != nil {
		return FidelityReport{}, err
	}
	contentBlocks, err := decodeBreakdown(members["content_block_counts"], clonebundle.ContentBlockTypes(), "content_block_counts")
	if err != nil {
		return FidelityReport{}, err
	}
	if err := checkBreakdownSums("fidelity report content_block_counts", contentBlocks, counts); err != nil {
		return FidelityReport{}, err
	}
	byteCounts, err := decodeByteCounts(members["byte_counts"])
	if err != nil {
		return FidelityReport{}, err
	}
	reasonCounts, err := decodeReasonCounts(members["reason_counts"])
	if err != nil {
		return FidelityReport{}, err
	}
	derivedReasons := deriveReasonCounts(rows)
	for reason, sealed := range reasonCounts {
		derived, present := derivedReasons[reason]
		if !present || sealed != derived {
			return FidelityReport{}, invalid("%s reason_counts %q reconciles %d rows, want %d from the disposition rows", owner, reason, sealed, derived)
		}
	}
	for reason, derived := range derivedReasons {
		if _, present := reasonCounts[reason]; !present {
			return FidelityReport{}, invalid("%s reason_counts miss reason %q with %d rows", owner, reason, derived)
		}
	}
	rawComplete, ok := rawBool(members["raw_bundle_complete"])
	if !ok {
		return FidelityReport{}, invalid("%s raw_bundle_complete is not a boolean", owner)
	}
	canonicalComplete, ok := rawBool(members["canonical_complete"])
	if !ok {
		return FidelityReport{}, invalid("%s canonical_complete is not a boolean", owner)
	}
	continuable, ok := rawBool(members["target_semantically_continuable"])
	if !ok {
		return FidelityReport{}, invalid("%s target_semantically_continuable is not a boolean", owner)
	}
	resumable, ok := rawBool(members["target_natively_resumable"])
	if !ok {
		return FidelityReport{}, invalid("%s target_natively_resumable is not a boolean", owner)
	}
	if archive {
		if continuable {
			return FidelityReport{}, invalid("%s target_semantically_continuable is true for archive", owner)
		}
		if resumable {
			return FidelityReport{}, invalid("%s target_natively_resumable is true for archive", owner)
		}
	}
	var staged *scalar.Digest
	var live *scalar.Digest
	if archive {
		if !isNull(members["staged_read_back_evidence_manifest_id"]) {
			return FidelityReport{}, invalid("%s staged_read_back_evidence_manifest_id is not null for archive", owner)
		}
		if !isNull(members["live_read_back_evidence_manifest_id"]) {
			return FidelityReport{}, invalid("%s live_read_back_evidence_manifest_id is not null for archive", owner)
		}
	} else {
		stagedID, ok := checkDigest(members["staged_read_back_evidence_manifest_id"])
		if !ok {
			return FidelityReport{}, invalid("%s staged_read_back_evidence_manifest_id is not a digest for target", owner)
		}
		staged = &stagedID
		liveID, ok := checkDigest(members["live_read_back_evidence_manifest_id"])
		if !ok {
			return FidelityReport{}, invalid("%s live_read_back_evidence_manifest_id is not a digest for target", owner)
		}
		live = &liveID
	}
	attestations, ok := checkSortedUniqueDigests(members["adapter_attestations"], 0, 64)
	if !ok {
		return FidelityReport{}, invalid("%s adapter_attestations are not sorted unique digest[0..64]", owner)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return FidelityReport{}, invalid("%s extensions %s", owner, extensionsFault)
	}
	if err := verifySelfDigest(members, fidelityReportSelf, reportID); err != nil {
		return FidelityReport{}, err
	}
	return FidelityReport{
		ReportID:                         reportID,
		Scope:                            scope,
		OperationID:                      operation,
		BundleID:                         bundle,
		SourceSnapshotDigest:             snapshot,
		CaptureManifestID:                captureID,
		CanonicalSessionID:               canonicalID,
		ProjectionPlanID:                 plan,
		SourceEnvironment:                sourceTuple,
		TargetEnvironment:                targetTuple,
		Profile:                          profile,
		RequiredDispositions:             required,
		ForbidReasons:                    forbidden,
		Rows:                             rows,
		Counts:                           counts,
		EventKindCounts:                  eventKinds,
		ContentBlockCounts:               contentBlocks,
		ByteCounts:                       byteCounts,
		ReasonCounts:                     reasonCounts,
		RawBundleComplete:                rawComplete,
		CanonicalComplete:                canonicalComplete,
		TargetSemanticallyContinuable:    continuable,
		TargetNativelyResumable:          resumable,
		StagedReadBackEvidenceManifestID: staged,
		LiveReadBackEvidenceManifestID:   live,
		AdapterAttestations:              attestations,
	}, nil
}

// decodeFidelityCounts validates one exact FidelityCounts record:
// exactly the seven disposition members, each a uint53.
func decodeFidelityCounts(raw json.RawMessage) (FidelityCounts, error) {
	owner := "fidelity report counts"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return FidelityCounts{}, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	allowed := map[string]bool{}
	for _, disposition := range dispositions {
		allowed[disposition] = true
	}
	if name, unknown := unknownMember(members, allowed); unknown {
		return FidelityCounts{}, invalid("%s carries unknown member %q", owner, name)
	}
	if name, missing := missingMember(members, dispositions); missing {
		return FidelityCounts{}, invalid("%s misses a required member %q", owner, name)
	}
	var counts FidelityCounts
	for _, disposition := range dispositions {
		value, ok := checkUint53Bounds(members[disposition], 0, maxUint53)
		if !ok {
			return FidelityCounts{}, invalid("%s %s is not a uint53", owner, disposition)
		}
		switch disposition {
		case "exact":
			counts.Exact = value
		case "semantic":
			counts.Semantic = value
		case "summarized":
			counts.Summarized = value
		case "opaque_preserved":
			counts.OpaquePreserved = value
		case "synthesized":
			counts.Synthesized = value
		case "omitted":
			counts.Omitted = value
		case "unrecoverable":
			counts.Unrecoverable = value
		}
	}
	return counts, nil
}

// decodeBreakdown validates one closed breakdown map: exactly the
// owner vocabulary keys, each an exact FidelityCounts record.
func decodeBreakdown(raw json.RawMessage, keys []string, member string) (map[string]FidelityCounts, error) {
	owner := "fidelity report " + member
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return nil, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	if name, unknown := unknownMember(members, allowed); unknown {
		return nil, invalid("%s carry unknown key %q", owner, name)
	}
	if name, missing := missingMember(members, keys); missing {
		return nil, invalid("%s miss key %q", owner, name)
	}
	breakdown := make(map[string]FidelityCounts, len(keys))
	for _, key := range keys {
		counts, err := decodeBreakdownCounts(members[key], owner, key)
		if err != nil {
			return nil, err
		}
		breakdown[key] = counts
	}
	return breakdown, nil
}

// decodeBreakdownCounts validates one FidelityCounts value inside a
// breakdown map. It mirrors decodeFidelityCounts with the
// breakdown key in every detail so a tampered cell names itself.
func decodeBreakdownCounts(raw json.RawMessage, owner, key string) (FidelityCounts, error) {
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return FidelityCounts{}, invalid("%s[%q] %s (%s)", owner, key, fault.detail, memberField(fault.member))
	}
	allowed := map[string]bool{}
	for _, disposition := range dispositions {
		allowed[disposition] = true
	}
	if name, unknown := unknownMember(members, allowed); unknown {
		return FidelityCounts{}, invalid("%s[%q] carries unknown member %q", owner, key, name)
	}
	if name, missing := missingMember(members, dispositions); missing {
		return FidelityCounts{}, invalid("%s[%q] misses a required member %q", owner, key, name)
	}
	var counts FidelityCounts
	for _, disposition := range dispositions {
		value, ok := checkUint53Bounds(members[disposition], 0, maxUint53)
		if !ok {
			return FidelityCounts{}, invalid("%s[%q] %s is not a uint53", owner, key, disposition)
		}
		switch disposition {
		case "exact":
			counts.Exact = value
		case "semantic":
			counts.Semantic = value
		case "summarized":
			counts.Summarized = value
		case "opaque_preserved":
			counts.OpaquePreserved = value
		case "synthesized":
			counts.Synthesized = value
		case "omitted":
			counts.Omitted = value
		case "unrecoverable":
			counts.Unrecoverable = value
		}
	}
	return counts, nil
}

// decodeByteCounts validates the byte map: exactly the seven
// dispositions to uint53 values.
func decodeByteCounts(raw json.RawMessage) (map[string]uint64, error) {
	owner := "fidelity report byte_counts"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return nil, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	allowed := map[string]bool{}
	for _, disposition := range dispositions {
		allowed[disposition] = true
	}
	if name, unknown := unknownMember(members, allowed); unknown {
		return nil, invalid("%s carry unknown disposition %q", owner, name)
	}
	if name, missing := missingMember(members, dispositions); missing {
		return nil, invalid("%s miss disposition %q", owner, name)
	}
	counts := make(map[string]uint64, len(dispositions))
	for _, disposition := range dispositions {
		value, ok := checkUint53Bounds(members[disposition], 0, maxUint53)
		if !ok {
			return nil, invalid("%s[%q] is not a uint53", owner, disposition)
		}
		counts[disposition] = value
	}
	return counts, nil
}

// decodeReasonCounts validates the reason map shape: string[1..128]
// keys to uint53 values above zero. Reconciliation against the rows
// happens at the call site.
func decodeReasonCounts(raw json.RawMessage) (map[string]uint64, error) {
	owner := "fidelity report reason_counts"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return nil, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	counts := make(map[string]uint64, len(members))
	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if length := stringLength(key); length < 1 || length > 128 {
			return nil, invalid("%s carry key outside string[1..128]", owner)
		}
		value, ok := checkUint53Bounds(members[key], 1, maxUint53)
		if !ok {
			return nil, invalid("%s[%q] is not a uint53 above zero", owner, key)
		}
		counts[key] = value
	}
	return counts, nil
}

// decodeRequiredDispositions validates the policy map: closed
// capture-class keys to sorted unique non-empty disposition sets.
func decodeRequiredDispositions(raw json.RawMessage) (map[string][]string, error) {
	owner := "fidelity report required_dispositions"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return nil, invalid("%s %s (%s)", owner, fault.detail, memberField(fault.member))
	}
	required := make(map[string][]string, len(members))
	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, class := range keys {
		if !clonebundle.ValidCaptureClass(class) {
			return nil, invalid("%s carry unknown class %q", owner, class)
		}
		elements, ok := decodeArray(members[class])
		if !ok || len(elements) < 1 {
			return nil, invalid("%s[%q] carry no dispositions, want a non-empty set", owner, class)
		}
		values := make([]string, 0, len(elements))
		previous := ""
		for index, element := range elements {
			disposition, ok := rawString(element)
			if !ok || !ValidDisposition(disposition) {
				return nil, invalid("%s[%q][%d] disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, class, index)
			}
			if index > 0 && disposition <= previous {
				return nil, invalid("%s[%q] are not sorted unique", owner, class)
			}
			previous = disposition
			values = append(values, disposition)
		}
		required[class] = values
	}
	return required, nil
}
