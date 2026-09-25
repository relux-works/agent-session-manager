package clonefidelity_test

import (
	"fmt"
	"testing"

	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// TestRecordGridOracle sweeps the full record-rule grid from SPEC
// v0.7.0 lines 10601-10614: disposition (7) x reason-set size (0, 1,
// 2, 128, 129) x canonical_object_id (present, null) x source
// evidence (present, absent) = 140 cells, each driven through both
// production entries. The verdict oracle is the four rules in the
// spec text plus the shape bounds, written here independently of
// production:
//
//  1. exact requires an empty reason set;
//  2. every other disposition requires at least one reason;
//  3. synthesized requires no source canonical object;
//  4. every non-synthesized row traces to captured source evidence
//     (the shape bound source_evidence_ids[1..65536] enforces the
//     presence half at both entries; resolution against the Capture
//     Manifest needs the manifest and is a stated bound).
//
// Admit cells round-trip build-then-decode. Refuse cells refuse at
// Build from the row input, and at Decode from a tampered sealed
// report; the tamper breaks the self digest too, but the row gate
// fires first with its literal detail.
func TestRecordGridOracle(t *testing.T) {
	dispositions := []string{"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable"}
	sizes := []int{0, 1, 2, 128, 129}
	admitted := 0
	for _, disposition := range dispositions {
		for _, size := range sizes {
			for _, canonical := range []bool{true, false} {
				for _, evidence := range []bool{true, false} {
					admit, buildLiteral, decodeLiteral := gridOracle(disposition, size, canonical, evidence)
					name := fmt.Sprintf("%s/reasons=%d/canonical=%t/evidence=%t", disposition, size, canonical, evidence)
					t.Run(name, func(t *testing.T) {
						reasons := gridReasons(size)
						row := validRowInput("grid-row", disposition)
						// Set explicitly: spreading an empty slice
						// would select the disposition defaults.
						row.ReasonCodes = reasons
						if !canonical {
							row.CanonicalObjectID = nil
						}
						if !evidence {
							row.SourceEvidenceIDs = []string{}
						}
						if admit {
							admitted++
							sealed, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
							if err != nil {
								t.Fatalf("BuildFidelityReport() error = %v, want admit", err)
							}
							report, err := clonefidelity.DecodeFidelityReport(sealed)
							if err != nil {
								t.Fatalf("DecodeFidelityReport() error = %v, want admit", err)
							}
							if len(report.Rows[0].ReasonCodes) != size {
								t.Fatalf("reasons = %d, want %d", len(report.Rows[0].ReasonCodes), size)
							}
							return
						}
						_, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
						requireRefusal(t, err, buildLiteral)
						// Decode side: tamper a valid sealed report
						// into the cell shape. The base row is
						// semantic with one reason; only the
						// four grid axes move.
						base := validRowInput("grid-row", "semantic", "unknown_native_event")
						sealed := mustBuild(t, validTargetInput(base))
						reasonValues := make([]any, 0, size)
						for _, reason := range reasons {
							reasonValues = append(reasonValues, reason)
						}
						mutated := tampered(t, sealed, func(document map[string]any) {
							tamperRow(document, 0, func(r map[string]any) {
								r["disposition"] = disposition
								r["reason_codes"] = reasonValues
								if canonical {
									r["canonical_object_id"] = fixtureDigest("canonical-grid-row")
								} else {
									r["canonical_object_id"] = nil
								}
								if evidence {
									r["source_evidence_ids"] = []any{fixtureDigest("evidence-grid-row")}
								} else {
									r["source_evidence_ids"] = []any{}
								}
							})
						})
						_, err = clonefidelity.DecodeFidelityReport(mutated)
						requireRefusal(t, err, decodeLiteral)
					})
				}
			}
		}
	}
	// Oracle census: exact admits (0 reasons, either canonical,
	// evidence present) = 2 cells; each of the 5 non-exact
	// non-synthesized dispositions admits (1, 2, 128 reasons) x
	// (canonical either way) x (evidence present) = 3*2*1 = 6 cells
	// each = 30; synthesized admits (1, 2, 128) x (canonical null)
	// x (evidence present) = 3 cells. Total 35 of 140.
	if admitted != 35 {
		t.Fatalf("admitted cells = %d, want 35 of 140", admitted)
	}
}

// gridOracle returns the cell verdict plus the first-fire literal
// detail per entry. Gate order is identical on both sides:
// evidence shape, reasons shape, reason rule, canonical rule.
func gridOracle(disposition string, size int, canonical, evidence bool) (admit bool, buildLiteral, decodeLiteral string) {
	if !evidence {
		return false,
			"carries 0 source_evidence_ids, want [1..65536]",
			"source_evidence_ids are not sorted unique digest[1..65536]"
	}
	if size > 128 {
		return false,
			fmt.Sprintf("carries %d reason_codes, want [0..128]", size),
			"reason_codes are not sorted unique string[1..128][0..128]"
	}
	if disposition == "exact" {
		if size != 0 {
			detail := fmt.Sprintf("exact carries %d reason_codes, want an empty reason set", size)
			return false, detail, detail
		}
		return true, "", ""
	}
	if size < 1 {
		detail := fmt.Sprintf("%s carries no reason_codes, want at least one reason", disposition)
		return false, detail, detail
	}
	if disposition == "synthesized" && canonical {
		detail := "synthesized carries a source canonical object"
		return false, detail, detail
	}
	return true, "", ""
}

// TestRecordFieldBounds pins every record string bound at the edge
// and edge+1 through both entries: source_item_key[1..512],
// source_class[1..128], target_locator[1..1024]|null,
// explanation[1..4096], and reason items[1..128].
func TestRecordFieldBounds(t *testing.T) {
	cases := []struct {
		name          string
		mutateInput   func(row *clonefidelity.DispositionRecordInput)
		mutateSealed  func(row map[string]any)
		buildLiteral  string
		decodeLiteral string
	}{
		{
			name: "key_too_long",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.SourceItemKey = repeat("k", 513)
			},
			mutateSealed:  func(row map[string]any) { row["source_item_key"] = repeat("k", 513) },
			buildLiteral:  "source_item_key is not a string[1..512]",
			decodeLiteral: "source_item_key is not a string[1..512]",
		},
		{
			name: "key_empty",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.SourceItemKey = ""
			},
			mutateSealed:  func(row map[string]any) { row["source_item_key"] = "" },
			buildLiteral:  "source_item_key is not a string[1..512]",
			decodeLiteral: "source_item_key is not a string[1..512]",
		},
		{
			name: "class_too_long",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.SourceClass = repeat("c", 129)
			},
			mutateSealed:  func(row map[string]any) { row["source_class"] = repeat("c", 129) },
			buildLiteral:  "source_class is not a string[1..128]",
			decodeLiteral: "source_class is not a string[1..128]",
		},
		{
			name: "class_empty",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.SourceClass = ""
			},
			mutateSealed:  func(row map[string]any) { row["source_class"] = "" },
			buildLiteral:  "source_class is not a string[1..128]",
			decodeLiteral: "source_class is not a string[1..128]",
		},
		{
			name: "locator_too_long",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.TargetLocator = strptr(repeat("l", 1025))
			},
			mutateSealed:  func(row map[string]any) { row["target_locator"] = repeat("l", 1025) },
			buildLiteral:  "target_locator is not a string[1..1024]",
			decodeLiteral: "target_locator is not a string[1..1024] or null",
		},
		{
			name: "locator_empty",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.TargetLocator = strptr("")
			},
			mutateSealed:  func(row map[string]any) { row["target_locator"] = "" },
			buildLiteral:  "target_locator is not a string[1..1024]",
			decodeLiteral: "target_locator is not a string[1..1024] or null",
		},
		{
			name: "explanation_too_long",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.Explanation = repeat("e", 4097)
			},
			mutateSealed:  func(row map[string]any) { row["explanation"] = repeat("e", 4097) },
			buildLiteral:  "explanation is not a string[1..4096]",
			decodeLiteral: "explanation is not a string[1..4096]",
		},
		{
			name: "explanation_empty",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.Explanation = ""
			},
			mutateSealed:  func(row map[string]any) { row["explanation"] = "" },
			buildLiteral:  "explanation is not a string[1..4096]",
			decodeLiteral: "explanation is not a string[1..4096]",
		},
		{
			name: "reason_item_too_long",
			mutateInput: func(row *clonefidelity.DispositionRecordInput) {
				row.ReasonCodes = []string{"a." + repeat("r", 127)}
			},
			mutateSealed:  func(row map[string]any) { row["reason_codes"] = []any{"a." + repeat("r", 127)} },
			buildLiteral:  "reason_codes[0] is not a string[1..128]",
			decodeLiteral: "reason_codes are not sorted unique string[1..128][0..128]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/build", func(t *testing.T) {
			row := validRowInput("bound-row", "semantic")
			tc.mutateInput(&row)
			_, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
			requireRefusal(t, err, tc.buildLiteral)
		})
		t.Run(tc.name+"/decode", func(t *testing.T) {
			sealed := mustBuild(t, validTargetInput(validRowInput("bound-row", "semantic")))
			mutated := tampered(t, sealed, func(document map[string]any) {
				tamperRow(document, 0, tc.mutateSealed)
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.decodeLiteral)
		})
	}
	// Edges admit: 512/128/1024/4096/128-character values seal and
	// decode at both entries.
	edgeRow := validRowInput("edge-row", "semantic")
	edgeRow.SourceItemKey = repeat("k", 512)
	edgeRow.SourceClass = repeat("c", 128)
	edgeRow.TargetLocator = strptr(repeat("l", 1024))
	edgeRow.Explanation = repeat("e", 4096)
	edgeRow.ReasonCodes = []string{repeat("r", 63) + "." + repeat("s", 62) + ".t"}
	report := mustDecode(t, mustBuild(t, validTargetInput(edgeRow)))
	if len(report.Rows[0].SourceItemKey) != 512 {
		t.Fatalf("edge key length = %d, want 512", len(report.Rows[0].SourceItemKey))
	}
	// Null locator admits at both entries.
	nullRow := validRowInput("null-locator", "semantic")
	nullRow.TargetLocator = nil
	mustDecode(t, mustBuild(t, validTargetInput(nullRow)))
}

// TestRecordMultibyteMeasure pins the Section 1.6 character measure
// at the shared string gate: 512 multibyte characters admit, 513
// refuse, at both entries.
func TestRecordMultibyteMeasure(t *testing.T) {
	admit := validRowInput("multi-row", "semantic")
	admit.SourceItemKey = repeat("é", 512)
	mustDecode(t, mustBuild(t, validTargetInput(admit)))
	refuse := validRowInput("multi-row", "semantic")
	refuse.SourceItemKey = repeat("é", 513)
	_, err := clonefidelity.BuildFidelityReport(validTargetInput(refuse))
	requireRefusal(t, err, "source_item_key is not a string[1..512]")
	sealed := mustBuild(t, validTargetInput(validRowInput("multi-row", "semantic")))
	mutated := tampered(t, sealed, func(document map[string]any) {
		tamperRow(document, 0, func(row map[string]any) { row["source_item_key"] = repeat("é", 513) })
	})
	_, err = clonefidelity.DecodeFidelityReport(mutated)
	requireRefusal(t, err, "source_item_key is not a string[1..512]")
}

// TestRecordSortedUnique pins the sorted-unique arrays with an
// unsorted array and with a duplicate at both entries:
// reason_codes, source_evidence_ids, staged and live evidence.
func TestRecordSortedUnique(t *testing.T) {
	// Hand digests (literals, also anchoring the sorted narrowings).
	handLow := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	handHigh := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	evidenceUnsorted := []string{handHigh, handLow}
	duplicate := []string{handLow, handLow}
	cases := []struct {
		name          string
		reasons       []string
		evidence      []string
		staged        []string
		live          []string
		buildLiteral  string
		decodeLiteral string
	}{
		{"reasons_unsorted",
			[]string{"unknown_native_event", "credential_excluded"},
			nil, nil, nil,
			"reason_codes are not sorted unique",
			"reason_codes are not sorted unique string[1..128][0..128]"},
		{"reasons_duplicate",
			[]string{"unknown_native_event", "unknown_native_event"},
			nil, nil, nil,
			"reason_codes are not sorted unique",
			"reason_codes are not sorted unique string[1..128][0..128]"},
		{"evidence_unsorted",
			nil, evidenceUnsorted, nil, nil,
			"source_evidence_ids are not sorted unique",
			"source_evidence_ids are not sorted unique digest[1..65536]"},
		{"evidence_duplicate",
			nil, duplicate, nil, nil,
			"source_evidence_ids are not sorted unique",
			"source_evidence_ids are not sorted unique digest[1..65536]"},
		{"staged_unsorted",
			nil, nil, evidenceUnsorted, nil,
			"staged_evidence_object_ids are not sorted unique",
			"staged_evidence_object_ids are not sorted unique digest[0..65536]"},
		{"staged_duplicate",
			nil, nil, duplicate, nil,
			"staged_evidence_object_ids are not sorted unique",
			"staged_evidence_object_ids are not sorted unique digest[0..65536]"},
		{"live_unsorted",
			nil, nil, nil, evidenceUnsorted,
			"live_evidence_object_ids are not sorted unique",
			"live_evidence_object_ids are not sorted unique digest[0..65536]"},
		{"live_duplicate",
			nil, nil, nil, duplicate,
			"live_evidence_object_ids are not sorted unique",
			"live_evidence_object_ids are not sorted unique digest[0..65536]"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/build", func(t *testing.T) {
			row := validRowInput("sorted-row", "semantic")
			if tc.reasons != nil {
				row.ReasonCodes = tc.reasons
			}
			if tc.evidence != nil {
				row.SourceEvidenceIDs = tc.evidence
			}
			if tc.staged != nil {
				row.StagedEvidenceObjectIDs = tc.staged
			}
			if tc.live != nil {
				row.LiveEvidenceObjectIDs = tc.live
			}
			_, err := clonefidelity.BuildFidelityReport(validTargetInput(row))
			requireRefusal(t, err, tc.buildLiteral)
		})
		t.Run(tc.name+"/decode", func(t *testing.T) {
			sealed := mustBuild(t, validTargetInput(validRowInput("sorted-row", "semantic")))
			mutated := tampered(t, sealed, func(document map[string]any) {
				tamperRow(document, 0, func(row map[string]any) {
					if tc.reasons != nil {
						values := make([]any, 0, len(tc.reasons))
						for _, reason := range tc.reasons {
							values = append(values, reason)
						}
						row["reason_codes"] = values
					}
					if tc.evidence != nil {
						values := make([]any, 0, len(tc.evidence))
						for _, id := range tc.evidence {
							values = append(values, id)
						}
						row["source_evidence_ids"] = values
					}
					if tc.staged != nil {
						values := make([]any, 0, len(tc.staged))
						for _, id := range tc.staged {
							values = append(values, id)
						}
						row["staged_evidence_object_ids"] = values
					}
					if tc.live != nil {
						values := make([]any, 0, len(tc.live))
						for _, id := range tc.live {
							values = append(values, id)
						}
						row["live_evidence_object_ids"] = values
					}
				})
			})
			_, err := clonefidelity.DecodeFidelityReport(mutated)
			requireRefusal(t, err, tc.decodeLiteral)
		})
	}
}

func repeat(char string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += char
	}
	return out
}
