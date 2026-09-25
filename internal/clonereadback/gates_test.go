package clonereadback_test

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file pins the measure, order, count, delegation, and scalar
// gates: character bounds at edge and edge+1, sorted-unique sweeps,
// owner-decoded members refused with owner detail at both entries,
// and the stated bounds the spec leaves underivable.

// TestNativeSessionStringEdges pins string[1..512] in CHARACTERS at
// both entries of both shapes: 1 and 512 admit, 0 and 513 refuse,
// and 513 multibyte characters (1026 bytes) refuse — a byte
// measure would admit a shorter character count at the same byte
// length, so the multibyte edge at exactly 512 characters admits.
func TestNativeSessionStringEdges(t *testing.T) {
	staged, live := mustReadPair(t)
	cases := []struct {
		length int
		multi  bool
		want   bool
	}{
		{1, false, true}, {512, false, true}, {0, false, false}, {513, false, false},
		{512, true, true}, {513, true, false},
	}
	for _, probe := range cases {
		value := strings.Repeat("a", probe.length)
		if probe.multi {
			value = strings.Repeat("é", probe.length)
		}
		input := validReadBackInput()
		input.ExpectedTargetNativeSession = value
		input.ObservedTargetNativeSession = value
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
		if probe.want && err != nil {
			t.Fatalf("readback length=%d multi=%v refused: %v", probe.length, probe.multi, err)
		}
		if !probe.want && err == nil {
			t.Fatalf("readback length=%d multi=%v admitted at Build", probe.length, probe.multi)
		}
		rinput := validReportInput(staged, live)
		rinput.ExpectedTargetNativeSession = value
		rinput.ObservedTargetNativeSession = value
		_, err = clonereadback.BuildValidationReport(rinput, staged, live)
		if probe.want && err != nil {
			t.Fatalf("report length=%d multi=%v refused: %v", probe.length, probe.multi, err)
		}
		if !probe.want && err == nil {
			t.Fatalf("report length=%d multi=%v admitted at Build", probe.length, probe.multi)
		}
	}
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range []struct {
		value string
		want  bool
	}{
		{"a", true}, {strings.Repeat("a", 512), true},
		{"", false}, {strings.Repeat("a", 513), false},
		{strings.Repeat("é", 512), true}, {strings.Repeat("é", 513), false},
	} {
		document := decodeDocument(t, sealed)
		document["expected_target_native_session_id"] = probe.value
		document["observed_target_native_session_id"] = probe.value
		document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		if probe.want && err != nil {
			t.Fatalf("decode %d chars refused: %v", len([]rune(probe.value)), err)
		}
		if !probe.want && err == nil {
			t.Fatalf("decode %d chars admitted", len([]rune(probe.value)))
		}
	}
}

// TestMediaTypeStringEdges pins media_type:string[1..128] at both
// read-back entries.
func TestMediaTypeStringEdges(t *testing.T) {
	for _, probe := range []struct {
		value string
		want  bool
	}{
		{"a", true}, {strings.Repeat("a", 128), true}, {strings.Repeat("é", 128), true},
		{"", false}, {strings.Repeat("a", 129), false}, {strings.Repeat("é", 129), false},
	} {
		input := validReadBackInput()
		row := validEvidenceInput("native_sample", "media")
		row.MediaType = probe.value
		input.EvidenceObjects = []clonereadback.EvidenceObjectInput{row}
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
		if probe.want && err != nil {
			t.Fatalf("media %d chars refused: %v", len([]rune(probe.value)), err)
		}
		if !probe.want && err == nil {
			t.Fatalf("media %d chars admitted at Build", len([]rune(probe.value)))
		}
	}
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range []struct {
		value string
		want  bool
	}{
		{"a", true}, {strings.Repeat("a", 128), true},
		{"", false}, {strings.Repeat("a", 129), false},
	} {
		document := decodeDocument(t, sealed)
		documentAt(t, document, "evidence_objects", 0)["media_type"] = probe.value
		document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		if probe.want && err != nil {
			t.Fatalf("decode media %d chars refused: %v", len(probe.value), err)
		}
		if !probe.want && err == nil {
			t.Fatalf("decode media %d chars admitted", len(probe.value))
		}
	}
}

// TestParsedHeadIDsSweep pins parsed_head_ids as sorted unique
// string[1..512][0..1024] at both read-back entries: sorted admits,
// unsorted/duplicated/overlong/overcount refuse with the literal,
// and the count edge 1024 admits while 1025 refuses.
func TestParsedHeadIDsSweep(t *testing.T) {
	input := validReadBackInput()
	input.ParsedHeadIDs = nil
	mustBuildReadBack(t, stagedAuthority(t), input)
	for _, probe := range [][]string{
		{"b", "a"}, {"a", "a"}, {"a", "b", "b"}, {""}, {strings.Repeat("a", 513)},
	} {
		bad := validReadBackInput()
		bad.ParsedHeadIDs = probe
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(bad, stagedAuthority(t))
		requireRefusal(t, err, "read-back evidence manifest", "parsed_head_ids")
	}
	edge := validReadBackInput()
	edge.ParsedHeadIDs = make([]string, 0, 1024)
	for index := 0; index < 1024; index++ {
		edge.ParsedHeadIDs = append(edge.ParsedHeadIDs, fmt.Sprintf("head-%04d", index))
	}
	mustBuildReadBack(t, stagedAuthority(t), edge)
	over := validReadBackInput()
	over.ParsedHeadIDs = append(append([]string{}, edge.ParsedHeadIDs...), "zzz-overflow-tail")
	sort.Strings(over.ParsedHeadIDs)
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(over, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_head_ids carries 1025 items, want [0..1024]")
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range [][]string{{"b", "a"}, {"a", "a"}} {
		document := decodeDocument(t, sealed)
		document["parsed_head_ids"] = probe
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		requireRefusal(t, err, "read-back evidence manifest", "parsed_head_ids are not sorted unique string[1..512][0..1024]")
	}
	// An overlong head and an over-count array refuse at Decode
	// under a consistent resealed id: the gate reads members, not
	// bytes. The over-count array stays sorted, so only the count
	// bound fires.
	document := decodeDocument(t, sealed)
	document["parsed_head_ids"] = []string{strings.Repeat("a", 513)}
	document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_head_ids are not sorted unique string[1..512][0..1024]")
	edgeSealed := mustBuildReadBack(t, stagedAuthority(t), edge)
	edgeDoc := decodeDocument(t, edgeSealed)
	heads := edgeDoc["parsed_head_ids"].([]any)
	heads = append(heads, "zzz-overflow-tail")
	rendered := make([]string, 0, len(heads))
	for _, head := range heads {
		rendered = append(rendered, head.(string))
	}
	sort.Strings(rendered)
	edgeDoc["parsed_head_ids"] = rendered
	edgeDoc["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, edgeDoc), "read_back_evidence_manifest_id")
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, edgeDoc), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_head_ids are not sorted unique string[1..512][0..1024]")
}

// TestEvidenceOrderSweep pins "sorted unique by evidence kind/blob
// ID" at both read-back entries: the six permutations of three rows
// admit exactly the sorted one, same-kind rows order by blob ID,
// and duplicated (kind, blob) pairs refuse with the literal.
func TestEvidenceOrderSweep(t *testing.T) {
	rows := []clonereadback.EvidenceObjectInput{
		validEvidenceInput("marker_observation", "order-c"),
		validEvidenceInput("native_sample", "order-a"),
		validEvidenceInput("parser_trace", "order-b"),
	}
	sorted := append([]clonereadback.EvidenceObjectInput{}, rows...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].EvidenceKind != sorted[j].EvidenceKind {
			return sorted[i].EvidenceKind < sorted[j].EvidenceKind
		}
		return sorted[i].BlobID < sorted[j].BlobID
	})
	admitted := 0
	for _, perm := range permutations3() {
		input := validReadBackInput()
		input.EvidenceObjects = []clonereadback.EvidenceObjectInput{rows[perm[0]], rows[perm[1]], rows[perm[2]]}
		_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
		isSorted := rows[perm[0]].EvidenceKind == sorted[0].EvidenceKind && rows[perm[1]].EvidenceKind == sorted[1].EvidenceKind && rows[perm[2]].EvidenceKind == sorted[2].EvidenceKind
		if isSorted {
			if err != nil {
				t.Fatalf("sorted permutation %v refused: %v", perm, err)
			}
			admitted++
		} else if err == nil {
			t.Fatalf("unsorted permutation %v admitted at Build", perm)
		} else if !strings.Contains(err.Error(), "evidence_objects are not sorted unique by evidence kind/blob ID") {
			t.Fatalf("permutation %v error %q misses the literal", perm, err.Error())
		}
	}
	if admitted != 1 {
		t.Fatalf("admitted permutations = %d, want 1", admitted)
	}
	// Same-kind rows order by blob ID string order.
	low := validEvidenceInput("native_sample", "same-kind-low")
	high := validEvidenceInput("native_sample", "same-kind-high")
	pair := []clonereadback.EvidenceObjectInput{low, high}
	sort.Slice(pair, func(i, j int) bool { return pair[i].BlobID < pair[j].BlobID })
	ordered := validReadBackInput()
	ordered.EvidenceObjects = pair
	mustBuildReadBack(t, stagedAuthority(t), ordered)
	flipped := validReadBackInput()
	flipped.EvidenceObjects = []clonereadback.EvidenceObjectInput{pair[1], pair[0]}
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(flipped, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "evidence_objects are not sorted unique by evidence kind/blob ID")
	// Duplicated pairs refuse even when adjacent-sorted.
	dup := validReadBackInput()
	dup.EvidenceObjects = []clonereadback.EvidenceObjectInput{low, low}
	_, _, err = clonereadback.BuildReadBackEvidenceManifest(dup, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "evidence_objects are not sorted unique by evidence kind/blob ID")
	// Decode pins the same gate over sealed bytes.
	sealed := mustBuildReadBack(t, stagedAuthority(t), ordered)
	document := decodeDocument(t, sealed)
	rows2 := document["evidence_objects"].([]any)
	rows2[0], rows2[1] = rows2[1], rows2[0]
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "evidence_objects are not sorted unique by evidence kind/blob ID")
}

func permutations3() [][]int {
	return [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
}

// TestEvidenceCountEdges pins EvidenceObject[0..65536] at both
// read-back entries: empty admits, 65536 admits, 65537 refuses with
// the literal count.
func TestEvidenceCountEdges(t *testing.T) {
	empty := validReadBackInput()
	empty.EvidenceObjects = nil
	mustBuildReadBack(t, stagedAuthority(t), empty)
	full := validReadBackInput()
	full.EvidenceObjects = make([]clonereadback.EvidenceObjectInput, 0, 65536)
	seen := map[string]bool{}
	for counter := 0; len(full.EvidenceObjects) < 65536; counter++ {
		row := validEvidenceInput("native_sample", fmt.Sprintf("edge-%d", counter))
		if seen[row.BlobID] {
			continue
		}
		seen[row.BlobID] = true
		full.EvidenceObjects = append(full.EvidenceObjects, row)
	}
	sort.Slice(full.EvidenceObjects, func(i, j int) bool {
		return full.EvidenceObjects[i].BlobID < full.EvidenceObjects[j].BlobID
	})
	sealed := mustBuildReadBack(t, stagedAuthority(t), full)
	if _, err := clonereadback.DecodeReadBackEvidenceManifest(sealed, stagedAuthority(t)); err != nil {
		t.Fatalf("65536 rows refused at Decode: %v", err)
	}
	over := validReadBackInput()
	over.EvidenceObjects = append(append([]clonereadback.EvidenceObjectInput{}, full.EvidenceObjects...), validEvidenceInput("parser_trace", "overflow"))
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(over, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "carries 65537 evidence_objects, want [0..65536]")
	document := decodeDocument(t, sealed)
	rows := document["evidence_objects"].([]any)
	rows = append(rows, map[string]any{
		"evidence_kind": "parser_trace", "media_type": "text/plain", "byte_count": float64(1),
		"blob_id": fixtureDigest("overflow"), "blob_descriptor_id": fixtureDigest("overflow-desc"),
	})
	document["evidence_objects"] = rows
	document["read_back_evidence_manifest_id"] = independentSelfDigest(t, marshalDocument(t, document), "read_back_evidence_manifest_id")
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "carries 65537 evidence_objects, want [0..65536]")
}

// TestFindingsCountEdges pins AdapterFinding[0..4096] at both report
// entries: 4096 admits, 4097 refuses.
func TestFindingsCountEdges(t *testing.T) {
	staged, live := mustReadPair(t)
	severities := make([]string, 0, 4096)
	for index := 0; index < 4096; index++ {
		severities = append(severities, "info")
	}
	full := validReportInput(staged, live)
	full.Findings = fixtureFindings(severities...)
	sealed := mustBuildReport(t, full, staged, live)
	if _, err := clonereadback.DecodeValidationReport(sealed, staged, live); err != nil {
		t.Fatalf("4096 findings refused at Decode: %v", err)
	}
	over := validReportInput(staged, live)
	over.Findings = fixtureFindings(append(severities, "info")...)
	_, err := clonereadback.BuildValidationReport(over, staged, live)
	requireRefusal(t, err, "clone validation report", "findings are not AdapterFinding[0..4096]")
	// Decode refuses the over-count document under a consistent
	// resealed id: the count gate reads members, not bytes.
	overDoc := decodeDocument(t, sealed)
	findings := overDoc["findings"].([]any)
	findings = append(findings, map[string]any{
		"code": "probe-overflow", "extensions": map[string]any{},
		"message": "overflow", "remediation": nil, "severity": "info",
	})
	overDoc["findings"] = findings
	overDoc["validation_report_id"] = independentSelfDigest(t, marshalDocument(t, overDoc), "validation_report_id")
	_, err = clonereadback.DecodeValidationReport(marshalDocument(t, overDoc), staged, live)
	requireRefusal(t, err, "clone validation report", "findings are not AdapterFinding[0..4096]")
}

// TestOwnerDelegation pins that tuple, binding, and finding members
// decode through their landed owners at every entry: a broken
// nested document refuses with the owner detail, never a local
// re-implementation verdict.
func TestOwnerDelegation(t *testing.T) {
	staged, live := mustReadPair(t)
	brokenTuple := []byte(`{"adapter_version":"1.2.3","architecture":"arm64","environment_id":"relux.target","environment_version":"2026.09","platform":"plan9","store_schema_fingerprint":"` + fixtureDigest("store") + `"}`)
	brokenBinding := []byte(`{"branch":"main","cwd_relative":"/absolute/escape","extensions":{},"head_digest":null,"index_digest":null,"logical_workspace_id":"` + fixtureOperationID("ws") + `","repository_remote_fingerprints":[],"working_tree_digest":null}`)
	brokenFinding := []byte(`[{"code":"x","extensions":{},"message":"m","remediation":null,"severity":"critical"}]`)
	input := validReadBackInput()
	input.ObservedEnvironment = brokenTuple
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "observed_environment is not an Environment Tuple")
	input = validReadBackInput()
	input.WorkspaceBinding = brokenBinding
	_, _, err = clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "workspace_binding is not a Workspace Binding")
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	document := decodeDocument(t, sealed)
	document["observed_environment"] = decodeDocument(t, brokenTuple)
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "observed_environment is not an Environment Tuple")
	document = decodeDocument(t, sealed)
	document["workspace_binding"] = decodeDocument(t, brokenBinding)
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "workspace_binding is not a Workspace Binding")
	rinput := validReportInput(staged, live)
	rinput.TargetEnvironment = brokenTuple
	_, err = clonereadback.BuildValidationReport(rinput, staged, live)
	requireRefusal(t, err, "clone validation report", "target_environment is not an Environment Tuple")
	rinput = validReportInput(staged, live)
	rinput.Findings = brokenFinding
	_, err = clonereadback.BuildValidationReport(rinput, staged, live)
	requireRefusal(t, err, "clone validation report", "findings are not AdapterFinding[0..4096]")
	rsealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	rdocument := decodeDocument(t, rsealed)
	rdocument["target_environment"] = decodeDocument(t, brokenTuple)
	_, err = clonereadback.DecodeValidationReport(marshalDocument(t, rdocument), staged, live)
	requireRefusal(t, err, "clone validation report", "target_environment is not an Environment Tuple")
	rdocument = decodeDocument(t, rsealed)
	var findingDocs []any
	if err := json.Unmarshal(brokenFinding, &findingDocs); err != nil {
		t.Fatalf("unmarshal broken findings: %v", err)
	}
	rdocument["findings"] = findingDocs
	_, err = clonereadback.DecodeValidationReport(marshalDocument(t, rdocument), staged, live)
	requireRefusal(t, err, "clone validation report", "findings are not AdapterFinding[0..4096]")
}

// TestScalarGates pins digests, UUIDs, uint53, and the AX number
// model at every entry: malformed scalars refuse, 2^53 refuses, and
// float/exponent/string numbers refuse at decode.
func TestScalarGates(t *testing.T) {
	staged, live := mustReadPair(t)
	input := validReadBackInput()
	input.OperationID = "not-a-uuid"
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "operation_id is not a UUIDv7")
	input = validReadBackInput()
	input.StructuralDigest = "not-a-digest"
	_, _, err = clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "structural_digest is not a digest")
	input = validReadBackInput()
	input.ParsedEventCount = 1 << 53
	_, _, err = clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_event_count exceeds uint53")
	input = validReadBackInput()
	input.ParsedEventCount = 1<<53 - 1
	mustBuildReadBack(t, stagedAuthority(t), input)
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	for _, probe := range []any{float64(1.5), "41", true, nil} {
		document := decodeDocument(t, sealed)
		document["parsed_event_count"] = probe
		_, err := clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
		requireRefusal(t, err, "read-back evidence manifest", "parsed_event_count is not a uint53")
	}
	document := decodeDocument(t, sealed)
	document["parsed_event_count"] = float64(1 << 53)
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_event_count is not a uint53")
	// Exponent form must refuse even when its value is integral:
	// the AX number model admits canonical integers only.
	raw := strings.Replace(string(sealed), `"parsed_event_count":41`, `"parsed_event_count":4.1e1`, 1)
	if raw == string(sealed) {
		t.Fatal("could not splice exponent count: fixture changed")
	}
	_, err = clonereadback.DecodeReadBackEvidenceManifest([]byte(raw), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_event_count is not a uint53")
	rinput := validReportInput(staged, live)
	rinput.OperationID = "01935934-5b8a-4f3c-9e1d-2a4b6c8d0e1f" // v4, not v7
	_, err = clonereadback.BuildValidationReport(rinput, staged, live)
	requireRefusal(t, err, "clone validation report", "operation_id is not a UUIDv7")
}

// TestDecodeScalarStrings drives every sealed digest and UUID
// member through the production decode entries with a malformed
// string of the right JSON type: each refuses with its member
// literal. Scalar members are classified from the sealed values —
// "sha256:"-prefixed values are digests, owner-parseable UUIDs
// are UUIDs — never from a hand-typed list; the per-shape counts
// pin the classification itself.
func TestDecodeScalarStrings(t *testing.T) {
	want := map[string]int{"readback": 5, "evidence": 2, "report": 8}
	for _, shape := range readBackCensusShapes(t) {
		t.Run(shape.label, func(t *testing.T) {
			probed := 0
			for _, member := range shape.members {
				if member.typ != "string" {
					continue
				}
				document := decodeDocument(t, shape.seal(t))
				object := shape.navigate(t, document)
				value, ok := object[member.name].(string)
				if !ok {
					t.Fatalf("member %s sealed a non-string %T", member.name, object[member.name])
				}
				probe := ""
				if strings.HasPrefix(value, "sha256:") {
					probe = "not-a-digest"
				} else if _, err := scalar.ParseUUIDv7(value); err == nil {
					probe = "not-a-uuid"
				} else {
					continue
				}
				mutated := decodeDocument(t, shape.seal(t))
				shape.navigate(t, mutated)[member.name] = probe
				err := shape.decode(shape.rewrite(t, mutated))
				if err == nil {
					t.Fatalf("member %s admits malformed scalar %q", member.name, probe)
				}
				if !strings.Contains(err.Error(), shape.owner) || !strings.Contains(err.Error(), member.name) {
					t.Fatalf("member %s scalar error %q misses owner or member literal", member.name, err.Error())
				}
				probed++
			}
			if probed != want[shape.label] {
				t.Fatalf("shape %s probed %d scalar members, want %d", shape.label, probed, want[shape.label])
			}
		})
	}
}

// TestExtensionsClosed pins reverse-DNS-only extensions at every
// entry: open keys refuse with the literal.
func TestExtensionsClosed(t *testing.T) {
	staged, live := mustReadPair(t)
	input := validReadBackInput()
	input.Extensions = map[string]any{"not-a-reverse-dns-key": "x"}
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "extensions invalid")
	input = validReadBackInput()
	input.Extensions = map[string]any{"com.example.note": "x"}
	mustBuildReadBack(t, stagedAuthority(t), input)
	sealed := mustBuildReadBack(t, stagedAuthority(t), validReadBackInput())
	document := decodeDocument(t, sealed)
	document["extensions"] = map[string]any{"open": "x"}
	_, err = clonereadback.DecodeReadBackEvidenceManifest(marshalDocument(t, document), stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "extensions")
	rinput := validReportInput(staged, live)
	rinput.Extensions = map[string]any{"not-a-reverse-dns-key": "x"}
	_, err = clonereadback.BuildValidationReport(rinput, staged, live)
	requireRefusal(t, err, "clone validation report", "extensions invalid")
	rsealed := mustBuildReport(t, validReportInput(staged, live), staged, live)
	rdocument := decodeDocument(t, rsealed)
	rdocument["extensions"] = map[string]any{"open": "x"}
	_, err = clonereadback.DecodeValidationReport(marshalDocument(t, rdocument), staged, live)
	requireRefusal(t, err, "clone validation report", "extensions")
}

// TestParsedCountAndStructuralDigestAreStatedBounds pins what the
// spec leaves underivable: parsed_event_count carries no stated
// derivation from the heads or the rows, and structural_digest
// carries no stated preimage, so any uint53 count and any digest
// admit. A future derivation would be a spec change, not a gate
// fix.
func TestParsedCountAndStructuralDigestAreStatedBounds(t *testing.T) {
	input := validReadBackInput()
	input.ParsedEventCount = 0
	input.ParsedHeadIDs = []string{"head-a", "head-b", "head-c"}
	mustBuildReadBack(t, stagedAuthority(t), input)
	input.ParsedEventCount = 1<<53 - 1
	input.StructuralDigest = fixtureDigest("unrelated-preimage")
	mustBuildReadBack(t, stagedAuthority(t), input)
}

// TestReportTupleMatchIsReconciliationBound pins the sibling-scope
// bound: the report carries one target tuple and the "matching
// tuple" conjunct of the valid rule needs the read manifests'
// observed tuples as cross-inputs, so any owner-valid tuple admits
// here and the cross-match belongs to the final reconciliation
// leaf.
func TestReportTupleMatchIsReconciliationBound(t *testing.T) {
	staged, live := mustReadPair(t)
	other := []byte(`{"adapter_version":"9.9.9","architecture":"amd64","environment_id":"relux.other","environment_version":"1.0","platform":"linux","store_schema_fingerprint":"` + fixtureDigest("other-store") + `"}`)
	input := validReportInput(staged, live)
	input.TargetEnvironment = other
	mustBuildReport(t, input, staged, live)
}
