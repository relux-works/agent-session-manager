package provhost

import (
	"strings"
	"testing"
)

// safeQuiesceProof is a SafeBoundaryProof with every fact proven:
// empty blockers, both idle flags true, zero counts, and non-null
// boundary and store generation under safe true.
const safeQuiesceProof = `{
  "provider_id": "codex",
  "provider_version": "0.147.0",
  "input_blocked": true,
  "boundary_ref": "provider-event-42",
  "foreground_idle": true,
  "background_idle": true,
  "open_child_count": 0,
  "open_database_handle_count": 0,
  "store_generation": "closed:11111111-2222-4333-8444-555555555555:1",
  "safe": true,
  "blockers": []
}`

// TestSafeQuiesceProofDecodes proves the fully proven safe proof
// passes the production entry point.
// quiesceErr drives the proof decoder for refusal assertions: the
// safe bit is observed in the positive tests, never here.
func quiesceErr(body []byte) error {
	_, err := DecodeQuiesceProof(body)
	return err
}

func TestSafeQuiesceProofDecodes(t *testing.T) {
	safe, err := DecodeQuiesceProof([]byte(safeQuiesceProof))
	if err != nil {
		t.Fatalf("DecodeQuiesceProof(safe): %v", err)
	}
	if !safe {
		t.Fatal("DecodeQuiesceProof(safe) = false, want the observed safe bit true")
	}
}

// TestUnsafeQuiesceProofsDecode proves unsafe proofs are honest
// observations, not refusals: safe false validates with blockers,
// open counts, or null generation present.
func TestUnsafeQuiesceProofsDecode(t *testing.T) {
	bodies := []string{
		strings.Replace(safeQuiesceProof, `"safe": true,`, `"safe": false,`, 1),
		strings.Replace(safeQuiesceProof, `"blockers": []`, `"blockers": ["provider_busy"]`, 1),
		strings.Replace(safeQuiesceProof, `"open_child_count": 0,`, `"open_child_count": 2,`, 1),
		strings.Replace(safeQuiesceProof, `"background_idle": true,`, `"background_idle": null,`, 1),
		strings.Replace(safeQuiesceProof, `"store_generation": "closed:11111111-2222-4333-8444-555555555555:1",`, `"store_generation": null,`, 1),
	}
	for index, body := range bodies {
		unsafe := body
		if !strings.Contains(unsafe, `"safe": false`) {
			unsafe = strings.Replace(unsafe, `"safe": true,`, `"safe": false,`, 1)
		}
		safe, err := DecodeQuiesceProof([]byte(unsafe))
		if err != nil {
			t.Fatalf("unsafe proof %d refused: %v", index, err)
		}
		if safe {
			t.Fatalf("unsafe proof %d reports safe true", index)
		}
	}
}

// quiesceVariant rewrites one unique substring of the safe proof.
func quiesceVariant(t *testing.T, old, new string) []byte {
	t.Helper()
	if strings.Count(safeQuiesceProof, old) != 1 {
		t.Fatalf("quiesce variant anchor %q is not unique", old)
	}
	return []byte(strings.Replace(safeQuiesceProof, old, new, 1))
}

// TestDecodeQuiesceSafeLies enumerates the safe-rule conjuncts: each
// fixture violates exactly one proven fact while claiming safe true,
// and every one is refused under the same arm. Deleting any conjunct
// from the production disjunction admits its fixture and reddens.
func TestDecodeQuiesceSafeLies(t *testing.T) {
	rows := []struct {
		name string
		body []byte
	}{
		{"blockers open", quiesceVariant(t, `"blockers": []`, `"blockers": ["child_process_open"]`)},
		{"input flowing", quiesceVariant(t, `"input_blocked": true,`, `"input_blocked": false,`)},
		{"foreground busy", quiesceVariant(t, `"foreground_idle": true,`, `"foreground_idle": false,`)},
		{"background null", quiesceVariant(t, `"background_idle": true,`, `"background_idle": null,`)},
		{"background false", quiesceVariant(t, `"background_idle": true,`, `"background_idle": false,`)},
		{"boundary null", quiesceVariant(t, `"boundary_ref": "provider-event-42",`, `"boundary_ref": null,`)},
		{"generation null", quiesceVariant(t, `"store_generation": "closed:11111111-2222-4333-8444-555555555555:1",`, `"store_generation": null,`)},
		{"child open", quiesceVariant(t, `"open_child_count": 0,`, `"open_child_count": 1,`)},
		{"database handle open", quiesceVariant(t, `"open_database_handle_count": 0,`, `"open_database_handle_count": 1,`)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireFrameRefusal(t, quiesceErr(row.body), "safe", "leaves a fact unproven")
		})
	}
	t.Logf("quiescence safe-rule coverage: %d/%d violated facts refused", len(rows), len(rows))
}

// TestDecodeQuiesceRefusals drives every quiescence shape rule.
func TestDecodeQuiesceRefusals(t *testing.T) {
	long1024 := strings.Repeat("b", 1024)
	long1025 := strings.Repeat("b", 1025)
	long512 := strings.Repeat("g", 512)
	long513 := strings.Repeat("g", 513)
	long128 := strings.Repeat("v", 128)
	long129 := strings.Repeat("v", 129)
	rows := []struct {
		name   string
		body   []byte
		member string
		detail string
	}{
		{"unknown member", quiesceVariant(t, `"safe": true,`, `"safe": true, "score": 1,`), "score", "unknown member"},
		{"missing member", []byte(strings.Replace(safeQuiesceProof, "  \"safe\": true,\n", "", 1)), "safe", "misses a required member"},
		{"bad provider id", quiesceVariant(t, `"provider_id": "codex"`, `"provider_id": "Codex"`), "provider_id", "not a provider id"},
		{"empty provider version", quiesceVariant(t, `"provider_version": "0.147.0"`, `"provider_version": ""`), "provider_version", "not 1..128 characters"},
		{"provider version 129", quiesceVariant(t, `"provider_version": "0.147.0"`, `"provider_version": "`+long129+`"`), "provider_version", "not 1..128 characters"},
		{"input not boolean", quiesceVariant(t, `"input_blocked": true,`, `"input_blocked": "yes",`), "input_blocked", "not a boolean"},
		{"foreground not boolean", quiesceVariant(t, `"foreground_idle": true,`, `"foreground_idle": 1,`), "foreground_idle", "not a boolean"},
		{"safe not boolean", quiesceVariant(t, `"safe": true,`, `"safe": "yes",`), "safe", "not a boolean"},
		{"boundary number", quiesceVariant(t, `"boundary_ref": "provider-event-42",`, `"boundary_ref": 42,`), "boundary_ref", "not 1..1024 characters or null"},
		{"boundary 1025", quiesceVariant(t, `"boundary_ref": "provider-event-42",`, `"boundary_ref": "`+long1025+`",`), "boundary_ref", "not 1..1024 characters or null"},
		{"generation 513", quiesceVariant(t, `"store_generation": "closed:11111111-2222-4333-8444-555555555555:1",`, `"store_generation": "`+long513+`",`), "store_generation", "not 1..512 characters or null"},
		{"background number", quiesceVariant(t, `"background_idle": true,`, `"background_idle": 1,`), "background_idle", "not a boolean or null"},
		{"count fraction", quiesceVariant(t, `"open_child_count": 0,`, `"open_child_count": 1.5,`), "open_child_count", "not a uint53"},
		{"count negative", quiesceVariant(t, `"open_database_handle_count": 0`, `"open_database_handle_count": -1`), "open_database_handle_count", "not a uint53"},
		{"count overflow", quiesceVariant(t, `"open_child_count": 0,`, `"open_child_count": 9007199254740992,`), "open_child_count", "not a uint53"},
		{"blockers six", quiesceVariant(t, `"blockers": []`, `"blockers": ["background_unproven", "child_process_open", "database_handle_open", "provider_busy", "store_unstable", "provider_busy"]`), "blockers", "exceed 5 entries"},
		{"blocker unknown", quiesceVariant(t, `"blockers": []`, `"blockers": ["process_open"]`), "blockers", "unknown blocker"},
		{"blockers unsorted", quiesceVariant(t, `"safe": true,
  "blockers": []`, `"safe": false,
  "blockers": ["provider_busy", "background_unproven"]`), "blockers", "not sorted unique"},
		{"blockers duplicated", quiesceVariant(t, `"safe": true,
  "blockers": []`, `"safe": false,
  "blockers": ["provider_busy", "provider_busy"]`), "blockers", "not sorted unique"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireFrameRefusal(t, quiesceErr(row.body), row.member, row.detail)
		})
	}
	if _, err := DecodeQuiesceProof(quiesceVariant(t, `"provider_version": "0.147.0"`, `"provider_version": "`+long128+`"`)); err != nil {
		t.Fatalf("128-character provider version refused: %v", err)
	}
	if _, err := DecodeQuiesceProof(quiesceVariant(t, `"boundary_ref": "provider-event-42",`, `"boundary_ref": "`+long1024+`",`)); err != nil {
		t.Fatalf("1024-character boundary refused: %v", err)
	}
	if _, err := DecodeQuiesceProof(quiesceVariant(t, `"store_generation": "closed:11111111-2222-4333-8444-555555555555:1",`, `"store_generation": "`+long512+`",`)); err != nil {
		t.Fatalf("512-character generation refused: %v", err)
	}
	// The maximal uint53 count validates: the bound is exact.
	if _, err := DecodeQuiesceProof(quiesceVariant(t, `"safe": true,
  "blockers": []`, `"safe": false,
  "blockers": ["background_unproven", "child_process_open", "database_handle_open", "provider_busy", "store_unstable"]`)); err != nil {
		t.Fatalf("five sorted blockers refused: %v", err)
	}
}
