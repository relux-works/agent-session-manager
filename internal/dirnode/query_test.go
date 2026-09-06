package dirnode

import (
	"strings"
	"testing"
)

// TestDecodeQueryAcceptsFixture drives the production query entry
// with the exact contract vector: the batch identifier, the two
// position-indexed operations with their halves, and the caller.
func TestDecodeQueryAcceptsFixture(t *testing.T) {
	t.Parallel()
	query, err := DecodeQuery([]byte(fixtureQueryJSON()))
	if err != nil {
		t.Fatalf("DecodeQuery error = %v", err)
	}
	if query.QueryID != fixtureQueryID {
		t.Fatalf("query id = %q", query.QueryID)
	}
	if len(query.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(query.Operations))
	}
	if query.Operations[0].Index != 0 || query.Operations[0].Name != "schema" || query.Operations[0].Kind != queryKindRead {
		t.Fatalf("operation 0 = %+v", query.Operations[0])
	}
	if query.Operations[1].Index != 1 || query.Operations[1].Name != "sessions" || query.Operations[1].Kind != queryKindRead {
		t.Fatalf("operation 1 = %+v", query.Operations[1])
	}
	if query.Caller.CallerID != "test-caller" || query.Caller.Interaction != "non_interactive" || len(query.Caller.Scopes) != 1 {
		t.Fatalf("caller = %+v", query.Caller)
	}
}

// TestDecodeQueryIndexRules drives the production query entry
// with index defects: duplicate, sparse, reordered, and
// position-mismatched indexes invalidate the whole batch before
// any operation executes.
func TestDecodeQueryIndexRules(t *testing.T) {
	t.Parallel()
	good := fixtureQueryJSON()
	schemaOp := `{"operation_index":0,"name":"schema","parameters":{"extensions":{}},"fields":null,"preset":null,"skip":0,"take":1,"sort":[],"dry_run":false,"confirm":false,"expectation_digest":null,"idempotency_key":null,"extensions":{}}`
	duplicate := replaceOnce(t, good, `"operation_index":1`, `"operation_index":0`, 1)
	sparse := replaceOnce(t, good, `"operation_index":1`, `"operation_index":2`, 1)
	reordered := replaceOnce(t, good, `"operations":[`+schemaOp+`,`, `"operations":[`, 1)
	reordered = replaceOnce(t, reordered, `],"caller"`, `,`+schemaOp+`],"caller"`, 1)
	empty := `{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":"` + fixtureQueryID + `","operations":[],"caller":` + fixtureCallerJSON() + `,"extensions":{}}`
	for _, probe := range []struct {
		name string
		body string
	}{
		{"duplicate index", duplicate},
		{"sparse index", sparse},
		{"reordered operations", reordered},
		{"dropped first operation", replaceOnce(t, good, `"operations":[`+schemaOp+`,`, `"operations":[`, 1)},
		{"empty batch", empty},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeQuery([]byte(probe.body)); requireCode(t, err, "query_invalid") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestDecodeQueryEnvelopeRules drives the production query entry
// with one envelope vector per rule.
func TestDecodeQueryEnvelopeRules(t *testing.T) {
	t.Parallel()
	good := fixtureQueryJSON()
	for _, probe := range []struct {
		name string
		body string
	}{
		{"unknown member", replaceOnce(t, good, `],"caller":`, `],"extra":1,"caller":`, 1)},
		{"missing caller", replaceOnce(t, good, `,"caller":`+fixtureCallerJSON(), ``, 1)},
		{"wrong schema", replaceOnce(t, good, "session-directory-query", "session-directory-node-request", 1)},
		{"wrong version", replaceOnce(t, good, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)},
		{"bad query id", replaceOnce(t, good, fixtureQueryID, "not-a-uuid", 1)},
		{"bad caller scope", replaceOnce(t, good, `"directory.read"`, `"directory.root"`, 1)},
		{"empty scopes", replaceOnce(t, good, `"scopes":["directory.read"]`, `"scopes":[]`, 1)},
		{"bad interaction", replaceOnce(t, good, `"non_interactive"`, `"whenever"`, 1)},
		{"bad origin host", replaceOnce(t, good, fixtureOriginHost, "not-a-uuid", 1)},
		{"non object", `[]`},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeQuery([]byte(probe.body)); requireCode(t, err, "query_invalid") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestDecodeQueryRegistryRules drives the production query entry
// with unknown operations, unknown parameters, and unknown
// filters: the parser rejects the whole batch on unknown syntax,
// operation, parameter, field, preset, filter, sort key, or
// bound.
func TestDecodeQueryRegistryRules(t *testing.T) {
	t.Parallel()
	good := fixtureQueryJSON()
	for _, probe := range []struct {
		name string
		body string
	}{
		{"unknown operation", replaceOnce(t, good, `"name":"sessions"`, `"name":"search"`, 1)},
		{"unknown parameter", replaceOnce(t, good, `"filters":{"kinds"`, `"filters":{"kindsX"`, 1)},
		{"unknown filter", replaceOnce(t, good, `"freshness":[]`, `"freshness":[],"fuzz":[]`, 1)},
		{"unknown field", replaceOnce(t, good, `"fields":null,"preset":"overview"`, `"fields":["nope"],"preset":null`, 1)},
		{"unknown preset", replaceOnce(t, good, `"preset":"overview"`, `"preset":"everything"`, 1)},
		{"unknown sort field", replaceOnce(t, good, `"field":"stable_id"`, `"field":"vibes"`, 1)},
		{"unknown sort direction", replaceOnce(t, good, `"direction":"asc"`, `"direction":"sideways"`, 1)},
		{"fields and preset together", replaceOnce(t, good, `"fields":null,"preset":"overview"`, `"fields":["id"],"preset":"overview"`, 1)},
		{"bad filter enum", replaceOnce(t, good, `"reachability":[]`, `"reachability":["everywhere"]`, 1)},
		{"bad provider id", replaceOnce(t, good, `"provider_ids":[]`, `"provider_ids":["BAD"]`, 1)},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeQuery([]byte(probe.body)); requireCode(t, err, "query_invalid") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// mutationBatch builds a one-operation mutation batch with the
// given parameters, flags, and digests for the flag-rule tests.
func mutationBatch(name, parameters string, dryRun, confirm bool, expectation, idempotency string) string {
	return `{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":"` + fixtureQueryID + `","operations":[{"operation_index":0,"name":"` + name + `","parameters":` + parameters + `,"fields":null,"preset":null,"skip":0,"take":1,"sort":[],"dry_run":` + boolString(dryRun) + `,"confirm":` + boolString(confirm) + `,"expectation_digest":` + expectation + `,"idempotency_key":` + idempotency + `,"extensions":{}}],"caller":` + fixtureCallerJSON() + `,"extensions":{}}`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// TestDecodeQueryMutationFlagRules drives the production query
// entry through the mutation flag rules: dry run xor confirm,
// execute_plan requiring confirm plus both digests, annotation
// mutations requiring both digests when confirmed, and
// plan_continue requiring pure dry run.
func TestDecodeQueryMutationFlagRules(t *testing.T) {
	t.Parallel()
	title := `{"subject_kind":"ax_session","subject_id":"` + fixtureQueryID + `","title":"T","supersedes_annotation_ids":[],"extensions":{}}`
	digest := `"` + fixtureBatchDigest + `"`
	execute := `{"plan_id":` + digest + `,"operation_id":"` + fixtureOperationID + `","confirmations":[],"extensions":{}}`
	continueParams := `{"subject_kind":"ax_session","subject_id":"` + fixtureQueryID + `","source_instance_id":` + digest + `,"to_host_id":"` + fixtureOriginHost + `","to_installation_id":` + digest + `,"intent":"resume","workspace_policy":"exact_checkpoint","source_after_success":"retain","extensions":{}}`
	for _, probe := range []struct {
		name    string
		body    string
		refuses bool
	}{
		{"dry run title plans", mutationBatch("set_title", title, true, false, "null", "null"), false},
		{"confirmed title without digests", mutationBatch("set_title", title, false, true, "null", "null"), true},
		{"confirmed title with digests", mutationBatch("set_title", title, false, true, digest, digest), false},
		{"both flags", mutationBatch("set_title", title, true, true, digest, digest), true},
		{"neither flag", mutationBatch("set_title", title, false, false, "null", "null"), true},
		{"execute plan confirmed with digests", mutationBatch("execute_plan", execute, false, true, digest, digest), false},
		{"execute plan dry run", mutationBatch("execute_plan", execute, true, false, "null", "null"), true},
		{"execute plan missing digests", mutationBatch("execute_plan", execute, false, true, "null", digest), true},
		{"plan continue dry run", mutationBatch("plan_continue", continueParams, true, false, "null", "null"), false},
		{"plan continue confirmed", mutationBatch("plan_continue", continueParams, false, true, "null", "null"), true},
		{"read with dry run", replaceOnce(t, fixtureQueryJSON(), `"dry_run":false,"confirm":false`, `"dry_run":true,"confirm":false`, 1), true},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeQuery([]byte(probe.body))
			if probe.refuses && err == nil {
				t.Fatal("must refuse")
			}
			if !probe.refuses && err != nil {
				t.Fatalf("must admit: %v", err)
			}
			if probe.refuses {
				requireCode(t, err, "query_invalid")
			}
		})
	}
}

// TestDecodeQueryPaginationRules drives the production query
// entry through the skip/take rule: list operations page, while
// the schema read requires skip=0 and take=1.
func TestDecodeQueryPaginationRules(t *testing.T) {
	t.Parallel()
	paged := replaceOnce(t, fixtureQueryJSON(), `"skip":0,"take":10`, `"skip":5,"take":10`, 1)
	if _, err := DecodeQuery([]byte(paged)); err != nil {
		t.Fatalf("paged sessions must admit: %v", err)
	}
	unpaged := replaceOnce(t, fixtureQueryJSON(), `"skip":0,"take":1,"sort":[],"dry_run":false,"confirm":false,"expectation_digest":null,"idempotency_key":null,"extensions":{}},{"operation_index":1`, `"skip":1,"take":1,"sort":[],"dry_run":false,"confirm":false,"expectation_digest":null,"idempotency_key":null,"extensions":{}},{"operation_index":1`, 1)
	if _, err := DecodeQuery([]byte(unpaged)); requireCode(t, err, "query_invalid") == nil {
		t.Fatal("schema read with skip must refuse")
	}
}

// TestCheckCursorReuse drives the production cursor entry: a
// well-formed cursor admits under unchanged bounds, reuse after
// any bound change refuses with query_invalid, and a malformed
// cursor refuses before any bound is consulted.
func TestCheckCursorReuse(t *testing.T) {
	t.Parallel()
	if err := CheckCursorReuse("opaque-cursor", false); err != nil {
		t.Fatalf("CheckCursorReuse(unchanged) error = %v", err)
	}
	// Edges admit: the bound census pins the character measure
	// at 1 and 1024 through this entry.
	if err := CheckCursorReuse("x", false); err != nil {
		t.Fatalf("CheckCursorReuse(1 char) error = %v", err)
	}
	if err := CheckCursorReuse(strings.Repeat("x", 1024), false); err != nil {
		t.Fatalf("CheckCursorReuse(1024 chars) error = %v", err)
	}
	requireCode(t, CheckCursorReuse("opaque-cursor", true), "query_invalid")
	requireCode(t, CheckCursorReuse("", false), "query_invalid")
	requireCode(t, CheckCursorReuse(strings.Repeat("x", 1025), false), "query_invalid")
}
