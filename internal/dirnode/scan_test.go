package dirnode

import (
	"sort"
	"testing"
)

// TestCheckScanRequestAcceptsFixture drives the production scan
// request entry with the exact contract vector.
func TestCheckScanRequestAcceptsFixture(t *testing.T) {
	t.Parallel()
	request, err := CheckScanRequest([]byte(fixtureScanRequestJSON()))
	if err != nil {
		t.Fatalf("CheckScanRequest error = %v", err)
	}
	if request.OperationID != fixtureOperationID || len(request.Installations) != 2 {
		t.Fatalf("CheckScanRequest = %+v", request)
	}
	if request.HasPriorBatch || request.HasCursor || request.MaxInstances != 100 {
		t.Fatalf("CheckScanRequest optionals = %+v", request)
	}
}

// TestCheckScanRequestRefusals drives the production scan request
// entry with one structural vector per rule.
func TestCheckScanRequestRefusals(t *testing.T) {
	t.Parallel()
	good := fixtureScanRequestJSON()
	sortedPair := []string{fixtureInstallDigestA, fixtureInstallDigestB}
	sort.Strings(sortedPair)
	unsorted := replaceOnce(t, good, `"installation_ids":[`+quote(sortedPair[0])+`,`+quote(sortedPair[1])+`]`, `"installation_ids":[`+quote(sortedPair[1])+`,`+quote(sortedPair[0])+`]`, 1)
	for _, probe := range []struct {
		name string
		body string
	}{
		{"unknown member", replaceOnce(t, good, `"extensions":{}}`, `"extensions":{},"extra":1}`, 1)},
		{"missing member", replaceOnce(t, good, `"cursor":null,`, ``, 1)},
		{"bad operation id", replaceOnce(t, good, fixtureOperationID, "not-a-uuid", 1)},
		{"empty installations", replaceOnce(t, good, `"installation_ids":[`+quote(sortedPair[0])+`,`+quote(sortedPair[1])+`]`, `"installation_ids":[]`, 1)},
		{"unsorted installations", unsorted},
		{"bad installation digest", replaceOnce(t, good, fixtureInstallDigestA, "sha256:zzz", 1)},
		{"bad prior batch", replaceOnce(t, good, `"prior_batch_id":null`, `"prior_batch_id":0`, 1)},
		{"empty cursor", replaceOnce(t, good, `"cursor":null`, `"cursor":""`, 1)},
		{"non string cursor", replaceOnce(t, good, `"cursor":null`, `"cursor":0`, 1)},
		{"bad extensions", replaceOnce(t, good, `"extensions":{}}`, `"extensions":{"x":1}}`, 1)},
		{"non object", `[]`},
		{"zero max instances", replaceOnce(t, good, `"max_instances":100`, `"max_instances":0`, 1)},
		{"max instances past bound", replaceOnce(t, good, `"max_instances":100`, `"max_instances":65537`, 1)},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckScanRequest([]byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// quote renders one JSON string literal for fixture surgery.
func quote(value string) string {
	return `"` + value + `"`
}

// TestCheckScanRequestNullableOptions drives the production scan
// request entry through the nullable members: a digest prior
// batch and a bounded cursor admit, while wrong-typed values are
// not absences.
func TestCheckScanRequestNullableOptions(t *testing.T) {
	t.Parallel()
	body := replaceOnce(t, fixtureScanRequestJSON(), `"prior_batch_id":null`, `"prior_batch_id":`+quote(fixtureBatchDigest), 1)
	body = replaceOnce(t, body, `"cursor":null`, `"cursor":"abc"`, 1)
	request, err := CheckScanRequest([]byte(body))
	if err != nil {
		t.Fatalf("CheckScanRequest(optionals) error = %v", err)
	}
	if !request.HasPriorBatch || request.PriorBatchID != fixtureBatchDigest || !request.HasCursor || request.Cursor != "abc" {
		t.Fatalf("CheckScanRequest(optionals) = %+v", request)
	}
}

// TestCheckScanResponseAcceptsFixture drives the production scan
// response entry with the exact contract vector.
func TestCheckScanResponseAcceptsFixture(t *testing.T) {
	t.Parallel()
	response, err := CheckScanResponse([]byte(fixtureScanResponseJSON()))
	if err != nil {
		t.Fatalf("CheckScanResponse error = %v", err)
	}
	if !response.HasBatch || len(response.EnvironmentIDs) != 1 || len(response.NativeIDs) != 1 || response.HasNextCursor {
		t.Fatalf("CheckScanResponse = %+v", response)
	}
}

// TestCheckScanResponseRefusals drives the production scan
// response entry with one structural vector per rule.
func TestCheckScanResponseRefusals(t *testing.T) {
	t.Parallel()
	good := fixtureScanResponseJSON()
	nativePair := []string{fixtureNativeDigestA, fixtureEvidenceDigest}
	sort.Strings(nativePair)
	unsortedNatives := replaceOnce(t, good, `"native_observation_ids":[`+quote(fixtureNativeDigestA)+`]`, `"native_observation_ids":[`+quote(nativePair[1])+`,`+quote(nativePair[0])+`]`, 1)
	for _, probe := range []struct {
		name string
		body string
	}{
		{"unknown member", replaceOnce(t, good, `"extensions":{}}`, `"extensions":{},"extra":1}`, 1)},
		{"missing member", replaceOnce(t, good, `"next_cursor":null,`, ``, 1)},
		{"non object batch", replaceOnce(t, good, `"batch":{"batch_id":`+quote(fixtureBatchDigest)+`}`, `"batch":[]`, 1)},
		{"empty environment ids", replaceOnce(t, good, `"environment_observation_ids":[`+quote(fixtureEvidenceDigest)+`]`, `"environment_observation_ids":[]`, 1)},
		{"unsorted natives", unsortedNatives},
		{"bad next cursor type", replaceOnce(t, good, `"next_cursor":null`, `"next_cursor":0`, 1)},
		{"empty next cursor", replaceOnce(t, good, `"next_cursor":null`, `"next_cursor":""`, 1)},
		{"bad native digest", replaceOnce(t, good, quote(fixtureNativeDigestA), `"zzz"`, 1)},
		{"bad extensions", replaceOnce(t, good, `"extensions":{}}`, `"extensions":{"x":1}}`, 1)},
		{"non object", `[]`},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckScanResponse([]byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// journalBody renders one canonical scan request body for the
// journal tests with independently chosen literals.
func journalBody(operationID string) []byte {
	return []byte(`{"cursor":null,"extensions":{},"installation_ids":["` + fixtureInstallDigestA + `"],"max_instances":100,"operation_id":"` + operationID + `","prior_batch_id":null}`)
}

// TestJournalReplaysIdenticalBody drives the production
// idempotency entry for the first-seen path and the
// canonical-identical retry: the first call stores, the repeat
// replays the recorded bytes with no new record.
func TestJournalReplaysIdenticalBody(t *testing.T) {
	t.Parallel()
	journal := NewJournal()
	body := journalBody(fixtureOperationID)
	result := []byte(`{"batch":"first"}`)
	stored, prior, err := journal.CheckAndRecord(OpScan, fixtureOperationID, body, result)
	if err != nil || !stored || prior != nil {
		t.Fatalf("CheckAndRecord(first) = %v %q %v", stored, prior, err)
	}
	stored, prior, err = journal.CheckAndRecord(OpScan, fixtureOperationID, body, []byte(`{"batch":"second"}`))
	if err != nil || stored || string(prior) != string(result) {
		t.Fatalf("CheckAndRecord(repeat) = %v %q %v, want replay of first", stored, prior, err)
	}
}

// TestJournalRefusesChangedBody drives the production idempotency
// entry with a changed body under a recorded key: the retry is an
// idempotency_mismatch and records nothing — the prior result
// still replays afterwards.
func TestJournalRefusesChangedBody(t *testing.T) {
	t.Parallel()
	journal := NewJournal()
	body := journalBody(fixtureOperationID)
	result := []byte(`{"batch":"first"}`)
	if _, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, body, result); err != nil {
		t.Fatalf("CheckAndRecord(first) error = %v", err)
	}
	changed := journalBody(fixtureOperationID)
	changed = []byte(replaceOnce(t, string(changed), `"max_instances":100`, `"max_instances":101`, 1))
	if _, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, changed, []byte(`{"batch":"evil"}`)); requireCode(t, err, "idempotency_mismatch") == nil {
		t.Fatal("changed body must refuse")
	}
	_, prior, err := journal.CheckAndRecord(OpScan, fixtureOperationID, body, nil)
	if err != nil || string(prior) != string(result) {
		t.Fatalf("prior result after mismatch = %q %v", prior, err)
	}
}

// TestJournalScopesKeysByOperation drives the production
// idempotency entry across operations: one operation's
// operation_id never authorizes another operation's retry, and
// non-registry operations and non-UUID identifiers refuse before
// any record is touched.
func TestJournalScopesKeysByOperation(t *testing.T) {
	t.Parallel()
	journal := NewJournal()
	body := journalBody(fixtureOperationID)
	if _, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, body, []byte(`{}`)); err != nil {
		t.Fatalf("CheckAndRecord(scan) error = %v", err)
	}
	stored, _, err := journal.CheckAndRecord(OpEnrichmentRun, fixtureOperationID, body, []byte(`{}`))
	if err != nil || !stored {
		t.Fatalf("CheckAndRecord(enrichment-run) = %v %v, want fresh store", stored, err)
	}
	if _, _, err := journal.CheckAndRecord("query", fixtureOperationID, body, []byte(`{}`)); requireCode(t, err, "operation_unknown") == nil {
		t.Fatal("unknown operation must refuse")
	}
	if _, _, err := journal.CheckAndRecord(OpScan, "bad", body, []byte(`{}`)); requireCode(t, err, "invalid_config") == nil {
		t.Fatal("bad identifier must refuse")
	}
	if _, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, []byte(`{`), []byte(`{}`)); requireCode(t, err, "adapter_protocol_violation") == nil {
		t.Fatal("non-canonical body must refuse")
	}
}

// TestJournalSurvivesRestart drives the production journal export
// and import entries for crash recovery: a restarted host
// recovers the same record, replays the identical body, still
// refuses a changed body, and refuses corrupt bytes as an
// integrity failure rather than an empty journal.
func TestJournalSurvivesRestart(t *testing.T) {
	t.Parallel()
	journal := NewJournal()
	body := journalBody(fixtureOperationID)
	result := []byte(`{"batch":"durable"}`)
	if _, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, body, result); err != nil {
		t.Fatalf("CheckAndRecord error = %v", err)
	}
	exported, err := journal.Export()
	if err != nil {
		t.Fatalf("Export error = %v", err)
	}
	restarted := NewJournal()
	if err := restarted.Import(exported); err != nil {
		t.Fatalf("Import error = %v", err)
	}
	stored, replayed, err := restarted.CheckAndRecord(OpScan, fixtureOperationID, body, nil)
	if err != nil || stored || string(replayed) != string(result) {
		t.Fatalf("replay after restart = %v %q %v", stored, replayed, err)
	}
	changed := []byte(replaceOnce(t, string(body), `"max_instances":100`, `"max_instances":101`, 1))
	if _, _, err := restarted.CheckAndRecord(OpScan, fixtureOperationID, changed, nil); requireCode(t, err, "idempotency_mismatch") == nil {
		t.Fatal("changed body after restart must refuse")
	}
	empty := NewJournal()
	requireCode(t, empty.Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":2,"records":[]}`)), "integrity_failure")
	requireCode(t, empty.Import([]byte(`{"schema":"urn:ax:other","version":1,"records":[]}`)), "integrity_failure")
	requireCode(t, empty.Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1}`)), "integrity_failure")
	requireCode(t, empty.Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":{}}`)), "integrity_failure")
	requireCode(t, empty.Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[{"key":1,"body_digest":"eA==","result":"e30="}]}`)), "integrity_failure")
	requireCode(t, empty.Import([]byte(`not json`)), "integrity_failure")
	requireCode(t, empty.Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[{"key":"k","body_digest":"sha256:zzz","result":"e30="}]}`)), "integrity_failure")
	if err := empty.Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[]}`)); err != nil {
		t.Fatalf("Import(empty valid) error = %v", err)
	}
}

// TestJournalImportRefusesUnsortedRecords drives the production
// import entry with a two-record document in decreasing key
// order: the import refuses rather than silently accepting an
// order the exporter never emits.
func TestJournalImportRefusesUnsortedRecords(t *testing.T) {
	t.Parallel()
	low := `{"key":"scan\u00000198f4c8-8e50-7f66-8f70-1234567890ac","body_digest":` + quote(fixtureBatchDigest) + `,"result":"e30="}`
	high := `{"key":"scan\u00000198f4c8-8e50-7f66-8f70-1234567890ae","body_digest":` + quote(fixtureBatchDigest) + `,"result":"e30="}`
	sorted := `{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[` + low + `,` + high + `]}`
	reversed := `{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[` + high + `,` + low + `]}`
	journal := NewJournal()
	if err := journal.Import([]byte(sorted)); err != nil {
		t.Fatalf("Import(sorted) error = %v", err)
	}
	requireCode(t, NewJournal().Import([]byte(reversed)), "integrity_failure")
}

// TestJournalExportIsDeterministic drives the production export
// entry twice and requires identical bytes: recovery evidence
// must not depend on map iteration order.
func TestJournalExportIsDeterministic(t *testing.T) {
	t.Parallel()
	journal := NewJournal()
	for _, id := range []string{fixtureOperationID, fixtureQueryID} {
		if _, _, err := journal.CheckAndRecord(OpScan, id, journalBody(id), []byte(`{}`)); err != nil {
			t.Fatalf("CheckAndRecord(%s) error = %v", id, err)
		}
	}
	first, err := journal.Export()
	if err != nil {
		t.Fatalf("Export error = %v", err)
	}
	for range 20 {
		again, err := journal.Export()
		if err != nil {
			t.Fatalf("Export error = %v", err)
		}
		if string(again) != string(first) {
			t.Fatal("export bytes differ across calls: map iteration leaks into recovery evidence")
		}
	}
}
