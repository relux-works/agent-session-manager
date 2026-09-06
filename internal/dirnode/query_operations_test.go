package dirnode

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// This file derives the Section 10.8.5 operation obligation set
// (review round 2, B2): DecodeQuery reports every operation-level
// defect through the single checkQueryOperation arm ("directory
// query operation is not a closed QueryOperation"), so the census
// stays green while individual parameter validators sit at zero
// coverage. The obligation set is the seventeen-operation union of
// queryParameterMembers — eleven reads plus six mutations — and
// every member is driven below through the production DecodeQuery
// entry, both admitted in its valid shape and refused with a
// validator-specific defect. An operation missing from the
// admission table fails TestDecodeQueryAllOperationsAdmit; a
// validator admitting its probe fails the refusal table. The
// three review-named narrowing probes are rows here:
// environments authentication_status ["root"], jobs states
// ["pwned"], and distinct field "secret" must all refuse.

// validFiltersJSON is one valid DirectoryFilters body shared by
// the filter-carrying operations.
var validFiltersJSON = `{"kinds":[],"lineage_anchors":[],"provider_ids":[],"host_ids":[],"workspace_ids":[],"states":[],"management_states":[],"reachability":[],"freshness":[],"warnings":[],"updated_before":null,"updated_after":null,"extensions":{}}`

// queryOperationJSON renders one closed QueryOperation element.
func queryOperationJSON(index int, name, params, fields, preset string, skip, take int, sort, dryRun, confirm, expectation, idempotency string) string {
	return fmt.Sprintf(`{"operation_index":%d,"name":%q,"parameters":%s,"fields":%s,"preset":%s,"skip":%d,"take":%d,"sort":%s,"dry_run":%s,"confirm":%s,"expectation_digest":%s,"idempotency_key":%s,"extensions":{}}`,
		index, name, params, fields, preset, skip, take, sort, dryRun, confirm, expectation, idempotency)
}

// singleOperationBatch wraps one operation in a valid query batch.
func singleOperationBatch(name, params, fields, preset string, skip, take int, sort, dryRun, confirm, expectation, idempotency string) string {
	return `{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":` + quote(fixtureQueryID) + `,"operations":[` + queryOperationJSON(0, name, params, fields, preset, skip, take, sort, dryRun, confirm, expectation, idempotency) + `],"caller":` + fixtureCallerJSON() + `,"extensions":{}}`
}

// readBatch builds a valid read batch: list operations page,
// singular reads require skip=0 and take=1.
func readBatch(name, params string, list bool) string {
	take := 1
	if list {
		take = 10
	}
	return singleOperationBatch(name, params, "null", "null", 0, take, "[]", "false", "false", "null", "null")
}

// sessionSubjectParams builds subject parameters for the named
// subject kind with a UUIDv7 identifier.
func sessionSubjectParams(kind string) string {
	return `{"subject_kind":` + quote(kind) + `,"subject_id":` + quote(fixtureQueryID) + `,"extensions":{}}`
}

// operationParams builds the valid parameter body per operation in
// Section 10.8.5 table order. Every literal is retyped from the
// contract, never referenced from production constants.
func operationParams(name string) string {
	digest := quote(fixtureBatchDigest)
	uuid := quote(fixtureQueryID)
	host := quote(fixtureOriginHost)
	switch name {
	case "schema", "directory_summary":
		return `{"extensions":{}}`
	case "sessions", "count":
		return `{"filters":` + validFiltersJSON + `,"extensions":{}}`
	case "session":
		return sessionSubjectParams("ax_session")
	case "lineage":
		return `{"anchor_id":` + uuid + `,"include_suggestions":false,"extensions":{}}`
	case "hosts":
		return `{"host_ids":[],"reachable":null,"extensions":{}}`
	case "environments":
		return `{"host_ids":[],"environment_ids":[],"authentication_status":[],"extensions":{}}`
	case "jobs":
		return `{"job_ids":[],"profile_ids":[],"states":[],"extensions":{}}`
	case "plans":
		return `{"plan_ids":[],"operation_ids":[],"include_expired":false,"extensions":{}}`
	case "distinct":
		return `{"field":"kind","filters":` + validFiltersJSON + `,"extensions":{}}`
	case "set_title":
		return `{"subject_kind":"ax_session","subject_id":` + uuid + `,"title":"T","supersedes_annotation_ids":[],"extensions":{}}`
	case "set_tags":
		return `{"subject_kind":"ax_session","subject_id":` + uuid + `,"tags":[],"supersedes_annotation_ids":[],"extensions":{}}`
	case "set_pin":
		return `{"subject_kind":"ax_session","subject_id":` + uuid + `,"value":true,"supersedes_annotation_ids":[],"extensions":{}}`
	case "enrich":
		return `{"subject_kind":"ax_session","subject_id":` + uuid + `,"profile_id":` + digest + `,"kinds":["summary"],"expected_head_digest":` + digest + `,"extensions":{}}`
	case "plan_continue":
		return `{"subject_kind":"ax_session","subject_id":` + uuid + `,"source_instance_id":` + digest + `,"to_host_id":` + host + `,"to_installation_id":` + digest + `,"intent":"resume","workspace_policy":"exact_checkpoint","source_after_success":"retain","extensions":{}}`
	case "execute_plan":
		return `{"plan_id":` + digest + `,"operation_id":` + quote(fixtureOperationID) + `,"confirmations":[],"extensions":{}}`
	default:
		return ""
	}
}

// refuseBatch wraps caller-supplied parameters in the valid
// framing for the named operation, so the refusal is attributable
// to the parameters rather than the envelope.
func refuseBatch(name, params string) string {
	switch name {
	case "sessions", "lineage", "hosts", "environments", "jobs", "plans":
		return readBatch(name, params, true)
	case "schema", "session", "count", "distinct", "directory_summary":
		return readBatch(name, params, false)
	case "execute_plan":
		digest := quote(fixtureBatchDigest)
		return mutationBatch(name, params, false, true, digest, digest)
	default:
		return mutationBatch(name, params, true, false, "null", "null")
	}
}

// operationBatch builds the valid single-operation batch for the
// named operation with its pagination and flag rules.
func operationBatch(name string) string {
	return refuseBatch(name, operationParams(name))
}

// badParams substitutes one needle inside the valid parameters of
// the named operation, failing when the needle is absent so the
// vector cannot pass vacuously against a still-valid body.
func badParams(t *testing.T, name, needle, replacement string) string {
	t.Helper()
	good := operationParams(name)
	if !strings.Contains(good, needle) {
		t.Fatalf("badParams(%s): needle %q absent; the vector mutates nothing", name, needle)
	}
	return strings.Replace(good, needle, replacement, 1)
}

// sortedLiterals builds n sorted unique JSON string literals with
// zero-padded numeric suffixes, so the sortedness gate stays
// green while the count sweeps.
func sortedLiterals(prefix string, n int) string {
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, fmt.Sprintf("%q", fmt.Sprintf("%s-%05d", prefix, i)))
	}
	return `[` + strings.Join(parts, ",") + `]`
}

// TestDecodeQueryAllOperationsAdmit drives every member of the
// seventeen-operation union through the production DecodeQuery
// entry in its valid shape. The table is checked against the
// production registry union, so an operation with no row fails
// here rather than sitting at zero coverage behind the shared
// operation arm.
func TestDecodeQueryAllOperationsAdmit(t *testing.T) {
	t.Parallel()
	union := append(append([]string(nil), readOperations...), mutationOperations...)
	if len(union) != 17 {
		t.Fatalf("query union = %d operations, want 17; the derivation is broken, not the registry", len(union))
	}
	for _, name := range union {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			batch := operationBatch(name)
			if batch == "" {
				t.Fatalf("no parameter builder for %q; the obligation set is short, not the registry", name)
			}
			query, err := DecodeQuery([]byte(batch))
			if err != nil {
				t.Fatalf("DecodeQuery(%s) error = %v", name, err)
			}
			if len(query.Operations) != 1 || query.Operations[0].Name != name || query.Operations[0].Index != 0 {
				t.Fatalf("operations = %+v, want the single %q operation at index 0", query.Operations, name)
			}
			kind, known := queryOperationKindOf(name)
			if !known {
				t.Fatalf("%q is not a registry member", name)
			}
			if query.Operations[0].Kind != kind {
				t.Fatalf("kind = %q, want %q", query.Operations[0].Kind, kind)
			}
		})
	}
}

// TestDecodeQueryOperationValidatorsRefuse drives one
// validator-specific defect per row through the production
// DecodeQuery entry. Every row must refuse with query_invalid: a
// validator admitting its probe — including the three
// review-named narrowing probes "root", "pwned", and "secret" —
// fails here.
func TestDecodeQueryOperationValidatorsRefuse(t *testing.T) {
	t.Parallel()
	pair := []string{fixtureInstallDigestA, fixtureInstallDigestB}
	sort.Strings(pair)
	if pair[0] == pair[1] {
		t.Fatal("fixture installation digests collide; the unsorted vector is unbuildable")
	}
	unsortedSupersedes := `[` + quote(pair[1]) + `,` + quote(pair[0]) + `]`
	// filterWith substitutes one needle inside the valid filters
	// body of the sessions parameters.
	filterWith := func(t *testing.T, needle, replacement string) string {
		t.Helper()
		params := operationParams("sessions")
		if !strings.Contains(params, needle) {
			t.Fatalf("filter needle %q absent; the vector mutates nothing", needle)
		}
		return refuseBatch("sessions", strings.Replace(params, needle, replacement, 1))
	}
	rows := []struct {
		name  string
		build func(t *testing.T) string
	}{
		{"schema extra parameter", func(t *testing.T) string {
			return refuseBatch("schema", `{"extensions":{},"extra":1}`)
		}},
		{"directory summary extra parameter", func(t *testing.T) string {
			return refuseBatch("directory_summary", `{"extensions":{},"extra":1}`)
		}},
		{"sessions bad reachability", func(t *testing.T) string {
			return filterWith(t, `"reachability":[]`, `"reachability":["everywhere"]`)
		}},
		{"sessions bad provider id", func(t *testing.T) string {
			return filterWith(t, `"provider_ids":[]`, `"provider_ids":["BAD"]`)
		}},
		{"sessions bad kind", func(t *testing.T) string {
			return filterWith(t, `"kinds":[]`, `"kinds":["bogus"]`)
		}},
		{"sessions bad management state", func(t *testing.T) string {
			return filterWith(t, `"management_states":[]`, `"management_states":["bogus"]`)
		}},
		{"sessions bad freshness", func(t *testing.T) string {
			return filterWith(t, `"freshness":[]`, `"freshness":["bogus"]`)
		}},
		{"sessions bad lineage anchor", func(t *testing.T) string {
			return filterWith(t, `"lineage_anchors":[]`, `"lineage_anchors":["zzz"]`)
		}},
		{"sessions warning past element bound", func(t *testing.T) string {
			return filterWith(t, `"warnings":[]`, `"warnings":[`+quote(strings.Repeat("w", 257))+`]`)
		}},
		{"sessions bad timestamp", func(t *testing.T) string {
			return filterWith(t, `"updated_before":null`, `"updated_before":"yesterday"`)
		}},
		{"count fields require null", func(t *testing.T) string {
			return singleOperationBatch("count", operationParams("count"), `["id"]`, "null", 0, 1, "[]", "false", "false", "null", "null")
		}},
		{"count take requires one", func(t *testing.T) string {
			return singleOperationBatch("count", operationParams("count"), "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{"session lineage kind refused", func(t *testing.T) string {
			return refuseBatch("session", sessionSubjectParams("lineage"))
		}},
		{"session bad subject id", func(t *testing.T) string {
			return refuseBatch("session", `{"subject_kind":"ax_session","subject_id":"nope","extensions":{}}`)
		}},
		{"lineage bad anchor", func(t *testing.T) string {
			return refuseBatch("lineage", badParams(t, "lineage", quote(fixtureQueryID), `"nope"`))
		}},
		{"lineage non-bool suggestions", func(t *testing.T) string {
			return refuseBatch("lineage", badParams(t, "lineage", `"include_suggestions":false`, `"include_suggestions":1`))
		}},
		{"hosts bad host id", func(t *testing.T) string {
			return refuseBatch("hosts", badParams(t, "hosts", `"host_ids":[]`, `"host_ids":["not-a-uuid"]`))
		}},
		{"hosts non-bool reachable", func(t *testing.T) string {
			return refuseBatch("hosts", badParams(t, "hosts", `"reachable":null`, `"reachable":1`))
		}},
		{"environments root status refused", func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"authentication_status":[]`, `"authentication_status":["root"]`))
		}},
		{"environments bad environment id", func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"environment_ids":[]`, `"environment_ids":["BAD"]`))
		}},
		{"environments unsorted environment ids", func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"environment_ids":[]`, `"environment_ids":["b.env","a.env"]`))
		}},
		{"jobs pwned state refused", func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"states":[]`, `"states":["pwned"]`))
		}},
		{"jobs bad job id", func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"job_ids":[]`, `"job_ids":["zzz"]`))
		}},
		{"jobs bad profile id", func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"profile_ids":[]`, `"profile_ids":["zzz"]`))
		}},
		{"plans non-bool include expired", func(t *testing.T) string {
			return refuseBatch("plans", badParams(t, "plans", `"include_expired":false`, `"include_expired":"yes"`))
		}},
		{"plans bad plan id", func(t *testing.T) string {
			return refuseBatch("plans", badParams(t, "plans", `"plan_ids":[]`, `"plan_ids":["zzz"]`))
		}},
		{"plans bad operation id", func(t *testing.T) string {
			return refuseBatch("plans", badParams(t, "plans", `"operation_ids":[]`, `"operation_ids":["zzz"]`))
		}},
		{"distinct secret field refused", func(t *testing.T) string {
			return refuseBatch("distinct", badParams(t, "distinct", `"field":"kind"`, `"field":"secret"`))
		}},
		{"distinct bad filter", func(t *testing.T) string {
			params := badParams(t, "distinct", `"reachability":[]`, `"reachability":["everywhere"]`)
			return refuseBatch("distinct", params)
		}},
		{"set title empty title", func(t *testing.T) string {
			return refuseBatch("set_title", badParams(t, "set_title", `"title":"T"`, `"title":""`))
		}},
		{"set title past title bound", func(t *testing.T) string {
			return refuseBatch("set_title", badParams(t, "set_title", `"title":"T"`, `"title":`+quote(strings.Repeat("a", 513))))
		}},
		{"set title unsorted supersedes", func(t *testing.T) string {
			return refuseBatch("set_title", badParams(t, "set_title", `"supersedes_annotation_ids":[]`, `"supersedes_annotation_ids":`+unsortedSupersedes))
		}},
		{"set title bad supersedes digest", func(t *testing.T) string {
			return refuseBatch("set_title", badParams(t, "set_title", `"supersedes_annotation_ids":[]`, `"supersedes_annotation_ids":["zzz"]`))
		}},
		{"set tags unsorted tags", func(t *testing.T) string {
			return refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":["b","a"]`))
		}},
		{"set tags past count bound", func(t *testing.T) string {
			return refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":`+sortedLiterals("tag", 257)))
		}},
		{"set pin non-bool value", func(t *testing.T) string {
			return refuseBatch("set_pin", badParams(t, "set_pin", `"value":true`, `"value":"yes"`))
		}},
		{"enrich bad kind", func(t *testing.T) string {
			return refuseBatch("enrich", badParams(t, "enrich", `"kinds":["summary"]`, `"kinds":["pwned"]`))
		}},
		{"enrich empty kinds", func(t *testing.T) string {
			return refuseBatch("enrich", badParams(t, "enrich", `"kinds":["summary"]`, `"kinds":[]`))
		}},
		{"enrich bad profile", func(t *testing.T) string {
			return refuseBatch("enrich", badParams(t, "enrich", `"profile_id":`+quote(fixtureBatchDigest), `"profile_id":"zzz"`))
		}},
		{"plan continue bad intent", func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"intent":"resume"`, `"intent":"teleport"`))
		}},
		{"plan continue bad workspace policy", func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"workspace_policy":"exact_checkpoint"`, `"workspace_policy":"whatever"`))
		}},
		{"plan continue bad source after success", func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"source_after_success":"retain"`, `"source_after_success":"vanish"`))
		}},
		{"execute plan unsorted confirmations", func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":["b","a"]`))
		}},
		{"execute plan past confirmations bound", func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":`+sortedLiterals("c", 65)))
		}},
		{"sessions unknown projection field", func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), `["id","nope"]`, "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeQuery([]byte(row.build(t))); requireCode(t, err, "query_invalid") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestDecodeQueryProjectionFields drives the field projection
// registry through the production DecodeQuery entry: a sorted
// registry subset admits, an unknown field refuses, and a
// duplicate field refuses on sortedness. The validQueryField gate
// sat at zero coverage behind the shared operation arm; these
// rows pin it.
func TestDecodeQueryProjectionFields(t *testing.T) {
	t.Parallel()
	admitted := singleOperationBatch("sessions", operationParams("sessions"), `["display_title","state"]`, "null", 0, 10, "[]", "false", "false", "null", "null")
	query, err := DecodeQuery([]byte(admitted))
	if err != nil {
		t.Fatalf("registry field subset must admit: %v", err)
	}
	if len(query.Operations) != 1 || query.Operations[0].Name != "sessions" {
		t.Fatalf("operations = %+v, want the single sessions operation", query.Operations)
	}
	for _, fields := range []string{`["nope"]`, `["id","id"]`, `["state","id"]`} {
		body := singleOperationBatch("sessions", operationParams("sessions"), fields, "null", 0, 10, "[]", "false", "false", "null", "null")
		if _, err := DecodeQuery([]byte(body)); requireCode(t, err, "query_invalid") == nil {
			t.Fatalf("fields %s must refuse", fields)
		}
	}
	// Bound edges for the projection array (element 1..64,
	// count 0..128): the empty selection admits, a 65-character
	// field refuses on length, and 129 fields refuse on count.
	drive := func(fields string) error {
		body := singleOperationBatch("sessions", operationParams("sessions"), fields, "null", 0, 10, "[]", "false", "false", "null", "null")
		_, err := DecodeQuery([]byte(body))
		return err
	}
	if err := drive(`[]`); err != nil {
		t.Fatalf("empty fields must admit: %v", err)
	}
	longField := strings.Repeat("f", 65)
	if err := drive(`["` + longField + `"]`); err == nil {
		t.Fatal("65-character field must refuse past the element maximum")
	}
	many := make([]string, 0, 129)
	for i := 0; i < 129; i++ {
		many = append(many, fmt.Sprintf(`"f-%05d"`, i))
	}
	if err := drive(`[` + strings.Join(many, ",") + `]`); err == nil {
		t.Fatal("129 fields must refuse past the count maximum")
	}
}

// TestDecodeQueryMutationLineageSubject drives the lineage subject
// through the production DecodeQuery entry: mutations admit the
// lineage kind while the session read refuses it.
func TestDecodeQueryMutationLineageSubject(t *testing.T) {
	t.Parallel()
	mutation := badParams(t, "set_title", `"subject_kind":"ax_session"`, `"subject_kind":"lineage"`)
	if _, err := DecodeQuery([]byte(refuseBatch("set_title", mutation))); err != nil {
		t.Fatalf("mutation lineage subject must admit: %v", err)
	}
}

// TestDecodeQueryCountBoundAdmissions pins the count ceilings
// from below through the production DecodeQuery entry: 256 tags
// and 64 confirmations admit, so a mutant tightening either
// ceiling reddens here while the refusal rows pin widening.
func TestDecodeQueryCountBoundAdmissions(t *testing.T) {
	t.Parallel()
	tags := badParams(t, "set_tags", `"tags":[]`, `"tags":`+sortedLiterals("tag", 256))
	if _, err := DecodeQuery([]byte(refuseBatch("set_tags", tags))); err != nil {
		t.Fatalf("256 tags at the maximum must admit: %v", err)
	}
	confirmations := badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":`+sortedLiterals("c", 64))
	digest := quote(fixtureBatchDigest)
	batch := mutationBatch("execute_plan", confirmations, false, true, digest, digest)
	if _, err := DecodeQuery([]byte(batch)); err != nil {
		t.Fatalf("64 confirmations at the maximum must admit: %v", err)
	}
}
