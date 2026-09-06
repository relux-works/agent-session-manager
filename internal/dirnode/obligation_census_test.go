package dirnode

import (
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file is the derived query-obligation census (review round
// 3, C2). DecodeQuery reports every operation-level defect through
// the single checkQueryOperation arm, so one witness per arm
// proves the arm exists while thirty-one removable obligations
// hide behind it. The admission half is already derived (the
// seventeen-operation union built from readOperations plus
// mutationOperations, failing closed on a missing builder); this
// file derives the refusal half the same way.
//
// Denominator (deriveQueryObligations): every false-returning site
// in query.go — a bare `return false` or a `return <ZeroLit>,
// false` — keyed query.go|<function>|obligation-<n> by occurrence
// order in its function. That is 112 sites: the reviewer's 89 bare
// sites plus the same-shaped tuple sites in checkCaller (11) and
// checkQueryOperation (12), which hide behind the caller arm and
// the operation arm respectively. Boolean helpers in other files
// funnel into message-distinct arms pinned by the reachability
// gate; query.go is the file whose validators collapse N into one
// message, so it carries its own roster.
//
// Every site carries exactly one row: a vector driven through the
// production DecodeQuery entry that refuses (or, for one
// classification site, admits). Roster fails closed in both
// directions: a derived site with no row fails, and a row naming
// a site production no longer derives fails as orphaned. One site
// is defensive (defensiveObligations) with a structural tripwire,
// not a vector.
//
// Resolution: each refusal vector violates exactly one sibling
// obligation — every other member of its validator holds a valid
// value — so two distinct obligations in one validator cannot both
// satisfy one witness. The committed suite proves each vector
// refuses through the production entry; the attached battery
// proves narrowness by flipping each site alone and recording
// which witnesses fail. Two structural overlaps are stated, not
// hidden:
//
//   - framing gates (checkQueryOperation obligations 7-12):
//     a gate fires exactly when its validator is false, so no
//     vector can violate a gate without violating one obligation
//     beneath it. Gate rows reuse a representative validator
//     witness, documented per row; the battery shows the AND
//     (the witness fails under both flips).
//   - layered delegates (filters arms over checkEnumSubset and
//     checkUUIDv7DigestSubset): a filters witness for a delegated
//     shape necessarily violates the delegate obligation too.
//     Those witnesses name both sites; the battery shows the pair.

// defensiveObligations names derived sites no public input can
// fire, with the bound that keeps them honest.
var defensiveObligations = map[string]string{
	// checkQueryParameters obligation-1 fires only when the kind
	// registry and the parameter-member table disagree on a name:
	// checkQueryOperation admits only registry names, so through
	// the production entry the lookup always succeeds while the
	// tables agree. TestQueryParameterMembersMatchRegistry fails
	// when the tables diverge, which is exactly when this arm
	// becomes reachable; a mutant removing it is
	// behavior-preserving while the tables agree.
	"query.go|checkQueryParameters|obligation-1": "fires only on registry divergence; pinned by the table-agreement tripwire",
}

// deriveQueryObligations derives every false-returning site in
// query.go from production source. A bare `return false` and a
// `return <ZeroLit>, false` both count: both refuse the batch.
// Anything else — expression returns, error returns — is outside
// the obligation shape. An empty derivation fails closed.
func deriveQueryObligations(t *testing.T) map[string]bool {
	t.Helper()
	source, err := readQuerySource(t)
	if err != nil {
		t.Fatalf("obligations: %v", err)
	}
	syntax, err := parseGoFile("query.go", source)
	if err != nil {
		t.Fatalf("obligations: %v", err)
	}
	sites := map[string]bool{}
	counts := map[string]int{}
	ast.Inspect(syntax, func(node ast.Node) bool {
		function, ok := node.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			return true
		}
		ast.Inspect(function.Body, func(inner ast.Node) bool {
			returned, ok := inner.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			if len(returned.Results) == 0 {
				return true
			}
			last, ok := returned.Results[len(returned.Results)-1].(*ast.Ident)
			if !ok || last.Name != "false" {
				return true
			}
			switch len(returned.Results) {
			case 1:
			case 2:
				if _, ok := returned.Results[0].(*ast.CompositeLit); !ok {
					return true
				}
			default:
				return true
			}
			counts[function.Name.Name]++
			sites[singleObligationKey(function.Name.Name, counts[function.Name.Name])] = true
			return true
		})
		return false
	})
	if len(sites) == 0 {
		t.Fatal("derived zero query obligations; the scanner is broken, not the package")
	}
	return sites
}

// readQuerySource reads the production query.go of this package
// directory.
func readQuerySource(t *testing.T) ([]byte, error) {
	t.Helper()
	for _, path := range productionFiles(t) {
		if filepath.Base(path) == "query.go" {
			contents, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			return contents, nil
		}
	}
	t.Fatal("production query.go is not among the scanned files; the census is blind, not complete")
	return nil, nil
}

// singleObligationKey renders one obligation site key.
func singleObligationKey(function string, ordinal int) string {
	return "query.go|" + function + "|obligation-" + strconv.Itoa(ordinal)
}

// obligationRow pairs one derived obligation site with the vector
// that exercises it through the production DecodeQuery entry.
// admit marks the single classification row whose vector must be
// admitted rather than refused.
type obligationRow struct {
	site  string
	name  string
	admit bool
	build func(t *testing.T) string
}

// TestQueryObligationRosterIsComplete requires every derived
// query obligation to carry exactly one row (or a defensive
// rationale). It is separate from the behavioral test below on
// purpose: removing a site renumbers its function's ordinals, so
// the roster reddens on any obligation mutant by construction.
// Killer-set analysis reads the behavioral subtests, never this
// roster verdict.
func TestQueryObligationRosterIsComplete(t *testing.T) {
	t.Parallel()
	derived := deriveQueryObligations(t)
	rows := obligationRows()
	seen := map[string]bool{}
	for _, row := range rows {
		if !derived[row.site] {
			t.Fatalf("orphaned obligation row %q (%s): production derives no such site", row.site, row.name)
		}
		if seen[row.site] {
			t.Fatalf("duplicate obligation row %q: one vector per site keeps the witness narrow", row.site)
		}
		seen[row.site] = true
	}
	var missing []string
	for site := range derived {
		if seen[site] {
			continue
		}
		if _, defensive := defensiveObligations[site]; defensive {
			continue
		}
		missing = append(missing, site)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("derived obligations with no witness vector:\n  %s", strings.Join(missing, "\n  "))
	}
}

// TestEveryQueryObligationIsDriven drives each obligation row's
// vector through the production DecodeQuery entry. A removed
// obligation admits its refusal vector — or, for the admission
// row, refuses it — and fails the named subtest here.
func TestEveryQueryObligationIsDriven(t *testing.T) {
	t.Parallel()
	rows := obligationRows()
	for _, row := range rows {
		row := row
		t.Run(row.site, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeQuery([]byte(row.build(t)))
			if row.admit {
				if err != nil {
					t.Fatalf("row %q (%s): admission vector refused: %v", row.site, row.name, err)
				}
				return
			}
			if failure := requireCode(t, err, "query_invalid"); failure == nil {
				t.Fatalf("row %q (%s): refusal vector admitted", row.site, row.name)
			}
		})
	}
}

// TestQueryParameterMembersMatchRegistry is the structural
// tripwire for the defensive checkQueryParameters obligation-1:
// the parameter-member table must key exactly the operation
// union. A name the kind registry admits but the member table
// misses would reach the defensive arm; this test fails first.
func TestQueryParameterMembersMatchRegistry(t *testing.T) {
	t.Parallel()
	union := append(append([]string(nil), readOperations...), mutationOperations...)
	if len(union) != 17 {
		t.Fatalf("query union = %d operations, want 17", len(union))
	}
	if len(queryParameterMembers) != len(union) {
		t.Fatalf("parameter-member table keys = %d, want the %d union members", len(queryParameterMembers), len(union))
	}
	for _, name := range union {
		members, ok := queryParameterMembers[name]
		if !ok || len(members) == 0 {
			t.Fatalf("operation %q has no parameter-member row; the defensive lookup would fire", name)
		}
	}
}

// sessionsWith substitutes one needle inside the valid sessions
// parameters, failing when the needle is absent so the vector
// cannot pass vacuously.
func sessionsWith(t *testing.T, needle, replacement string) string {
	t.Helper()
	params := operationParams("sessions")
	if !strings.Contains(params, needle) {
		t.Fatalf("sessions needle %q absent; the vector mutates nothing", needle)
	}
	return refuseBatch("sessions", strings.Replace(params, needle, replacement, 1))
}

// filtersWith substitutes one needle inside the valid filters
// body of the sessions parameters.
func filtersWith(t *testing.T, needle, replacement string) string {
	t.Helper()
	if !strings.Contains(validFiltersJSON, needle) {
		t.Fatalf("filters needle %q absent; the vector mutates nothing", needle)
	}
	return sessionsWith(t, needle, replacement)
}

// callerBatch wraps one caller object in the valid two-operation
// batch, so the refusal is attributable to the caller.
func callerBatch(t *testing.T, caller string) string {
	t.Helper()
	good := fixtureQueryJSON()
	if !strings.Contains(good, fixtureCallerJSON()) {
		t.Fatal("callerBatch: fixture caller absent; the vector mutates nothing")
	}
	return strings.Replace(good, fixtureCallerJSON(), caller, 1)
}

// callerWith substitutes one needle inside the valid caller
// object.
func callerWith(t *testing.T, needle, replacement string) string {
	t.Helper()
	caller := fixtureCallerJSON()
	if !strings.Contains(caller, needle) {
		t.Fatalf("caller needle %q absent; the vector mutates nothing", needle)
	}
	return callerBatch(t, strings.Replace(caller, needle, replacement, 1))
}

// obligationRows pairs every fireable derived obligation site
// with its narrow vector. Each vector violates exactly the named
// sibling obligation (layered delegates and shared framing gates
// are documented per row); every other member holds a valid
// value.
func obligationRows() []obligationRow {
	return []obligationRow{
		{site: "query.go|isListOperation|obligation-1", name: "count with list pagination", build: func(t *testing.T) string {
			// Shared with the pagination-gate row: the gate
			// admits list pagination exactly when this
			// classification does, so one vector witnesses
			// both; the battery shows the AND.
			return singleOperationBatch("count", operationParams("count"), "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|isAnnotationMutation|obligation-1", name: "enrich confirmed without expectation admits", admit: true, build: func(t *testing.T) string {
			// Removing the classification admits nothing new
			// here: it reroutes enrich into the annotation
			// branch, which refuses a confirmed mutation
			// without expectation and idempotency digests.
			return mutationBatch("enrich", operationParams("enrich"), false, true, "null", "null")
		}},
		{site: "query.go|validQueryField|obligation-1", name: "unknown projection field", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), `["id","nope"]`, "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|validQueryPreset|obligation-1", name: "unknown preset", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", `"everything"`, 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkCaller|obligation-1", name: "caller non-object", build: func(t *testing.T) string {
			good := fixtureQueryJSON()
			return strings.Replace(good, fixtureCallerJSON(), `[]`, 1)
		}},
		{site: "query.go|checkCaller|obligation-2", name: "caller unknown member", build: func(t *testing.T) string {
			return callerWith(t, `"extensions":{}}`, `"extensions":{},"extra":1}`)
		}},
		{site: "query.go|checkCaller|obligation-3", name: "caller missing member", build: func(t *testing.T) string {
			return callerWith(t, `"caller_id":"test-caller",`, ``)
		}},
		{site: "query.go|checkCaller|obligation-4", name: "caller empty identifier", build: func(t *testing.T) string {
			return callerWith(t, `"caller_id":"test-caller"`, `"caller_id":""`)
		}},
		{site: "query.go|checkCaller|obligation-5", name: "caller empty subject", build: func(t *testing.T) string {
			return callerWith(t, `"authentication_subject":"test-subject"`, `"authentication_subject":""`)
		}},
		{site: "query.go|checkCaller|obligation-6", name: "caller bad origin host", build: func(t *testing.T) string {
			return callerWith(t, fixtureOriginHost, "not-a-uuid")
		}},
		{site: "query.go|checkCaller|obligation-7", name: "caller bad interaction", build: func(t *testing.T) string {
			return callerWith(t, `"non_interactive"`, `"whenever"`)
		}},
		{site: "query.go|checkCaller|obligation-8", name: "caller empty scopes", build: func(t *testing.T) string {
			return callerWith(t, `"scopes":["directory.read"]`, `"scopes":[]`)
		}},
		{site: "query.go|checkCaller|obligation-9", name: "caller bad scope", build: func(t *testing.T) string {
			return callerWith(t, `"directory.read"`, `"directory.root"`)
		}},
		{site: "query.go|checkCaller|obligation-10", name: "caller bad policy digest", build: func(t *testing.T) string {
			return callerWith(t, fixturePolicyDigest, "sha256:zzz")
		}},
		{site: "query.go|checkCaller|obligation-11", name: "caller bad extensions", build: func(t *testing.T) string {
			return callerWith(t, `"extensions":{}}`, `"extensions":{"x":1}}`)
		}},
		{site: "query.go|checkQueryOperation|obligation-1", name: "operation non-object element", build: func(t *testing.T) string {
			schema := queryOperationJSON(0, "schema", `{"extensions":{}}`, "null", "null", 0, 1, "[]", "false", "false", "null", "null")
			return `{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":` + quote(fixtureQueryID) + `,"operations":[` + schema + `,[]],"caller":` + fixtureCallerJSON() + `,"extensions":{}}`
		}},
		{site: "query.go|checkQueryOperation|obligation-2", name: "operation unknown member", build: func(t *testing.T) string {
			batch := singleOperationBatch("sessions", operationParams("sessions"), "null", `"overview"`, 0, 10, "[]", "false", "false", "null", "null")
			return strings.Replace(batch, `"preset":"overview"`, `"preset":"overview","extra":1`, 1)
		}},
		{site: "query.go|checkQueryOperation|obligation-3", name: "operation missing member", build: func(t *testing.T) string {
			batch := operationBatch("sessions")
			if !strings.Contains(batch, `"take":10,`) {
				t.Fatal("take needle absent; the vector mutates nothing")
			}
			return strings.Replace(batch, `"take":10,`, ``, 1)
		}},
		{site: "query.go|checkQueryOperation|obligation-4", name: "operation duplicate index", build: func(t *testing.T) string {
			return strings.Replace(fixtureQueryJSON(), `"operation_index":1`, `"operation_index":0`, 1)
		}},
		{site: "query.go|checkQueryOperation|obligation-5", name: "operation non-string name", build: func(t *testing.T) string {
			batch := operationBatch("sessions")
			return strings.Replace(batch, `"name":"sessions"`, `"name":0`, 1)
		}},
		{site: "query.go|checkQueryOperation|obligation-6", name: "operation unknown name", build: func(t *testing.T) string {
			batch := operationBatch("sessions")
			return strings.Replace(batch, `"name":"sessions"`, `"name":"search"`, 1)
		}},
		{site: "query.go|checkQueryOperation|obligation-7", name: "parameters gate shares the schema-extra witness", build: func(t *testing.T) string {
			// Framing gate over the parameter union: no vector
			// violates the gate without violating one
			// obligation beneath it, so the row reuses that
			// obligation's witness. The battery shows the AND:
			// the witness fails under both flips.
			return refuseBatch("schema", `{"extensions":{},"extra":1}`)
		}},
		{site: "query.go|checkQueryOperation|obligation-8", name: "projection gate shares the fields-and-preset witness", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), `["id"]`, `"overview"`, 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryOperation|obligation-9", name: "pagination gate shares the count-take witness", build: func(t *testing.T) string {
			return singleOperationBatch("count", operationParams("count"), "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryOperation|obligation-10", name: "sort gate shares the unknown-sort-field witness", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":"vibes","direction":"asc","extensions":{}}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryOperation|obligation-11", name: "flags gate shares the unconfirmed-execute-plan witness", build: func(t *testing.T) string {
			return mutationBatch("execute_plan", operationParams("execute_plan"), true, false, "null", "null")
		}},
		{site: "query.go|checkQueryOperation|obligation-12", name: "operation extensions gate", build: func(t *testing.T) string {
			batch := operationBatch("sessions")
			return strings.Replace(batch, `"idempotency_key":null,"extensions":{}}]`, `"idempotency_key":null,"extensions":{"x":1}}]`, 1)
		}},
		{site: "query.go|checkQueryParameters|obligation-2", name: "parameters non-object", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", `[]`, "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryParameters|obligation-3", name: "parameters unknown member", build: func(t *testing.T) string {
			return refuseBatch("schema", `{"extensions":{},"extra":1}`)
		}},
		{site: "query.go|checkQueryParameters|obligation-4", name: "parameters missing member", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", `{"filters":`+validFiltersJSON+`}`, "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryParameters|obligation-5", name: "parameters bad extensions", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", `{"filters":`+validFiltersJSON+`,"extensions":{"x":1}}`, "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkSubject|obligation-1", name: "subject non-string kind", build: func(t *testing.T) string {
			return refuseBatch("session", badParams(t, "session", `"subject_kind":"ax_session"`, `"subject_kind":0`))
		}},
		{site: "query.go|checkSubject|obligation-2", name: "lineage kind on a read", build: func(t *testing.T) string {
			return refuseBatch("session", sessionSubjectParams("lineage"))
		}},
		{site: "query.go|checkSubject|obligation-3", name: "subject bogus kind", build: func(t *testing.T) string {
			return refuseBatch("session", badParams(t, "session", `"subject_kind":"ax_session"`, `"subject_kind":"bogus"`))
		}},
		{site: "query.go|checkSubject|obligation-4", name: "subject non-string identifier", build: func(t *testing.T) string {
			return refuseBatch("session", badParams(t, "session", quote(fixtureQueryID), `0`))
		}},
		{site: "query.go|checkLineageParameters|obligation-1", name: "lineage non-string anchor", build: func(t *testing.T) string {
			return refuseBatch("lineage", badParams(t, "lineage", quote(fixtureQueryID), `0`))
		}},
		{site: "query.go|checkLineageParameters|obligation-2", name: "lineage bad anchor", build: func(t *testing.T) string {
			return refuseBatch("lineage", badParams(t, "lineage", quote(fixtureQueryID), `"nope"`))
		}},
		{site: "query.go|checkHostsParameters|obligation-1", name: "hosts bad host identifier", build: func(t *testing.T) string {
			return refuseBatch("hosts", badParams(t, "hosts", `"host_ids":[]`, `"host_ids":["not-a-uuid"]`))
		}},
		{site: "query.go|checkEnvironmentsParameters|obligation-1", name: "environments bad host identifier", build: func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"host_ids":[]`, `"host_ids":["not-a-uuid"]`))
		}},
		{site: "query.go|checkEnvironmentsParameters|obligation-2", name: "environments unsorted identifiers", build: func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"environment_ids":[]`, `"environment_ids":["b.env","a.env"]`))
		}},
		{site: "query.go|checkEnvironmentsParameters|obligation-3", name: "environments bad identifier", build: func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"environment_ids":[]`, `"environment_ids":["BAD"]`))
		}},
		{site: "query.go|checkEnvironmentsParameters|obligation-4", name: "environments unsorted statuses", build: func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"authentication_status":[]`, `"authentication_status":["missing","available"]`))
		}},
		{site: "query.go|checkEnvironmentsParameters|obligation-5", name: "environments root status refused", build: func(t *testing.T) string {
			return refuseBatch("environments", badParams(t, "environments", `"authentication_status":[]`, `"authentication_status":["root"]`))
		}},
		{site: "query.go|checkJobsParameters|obligation-1", name: "jobs bad job identifier", build: func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"job_ids":[]`, `"job_ids":["zzz"]`))
		}},
		{site: "query.go|checkJobsParameters|obligation-2", name: "jobs bad profile identifier", build: func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"profile_ids":[]`, `"profile_ids":["zzz"]`))
		}},
		{site: "query.go|checkJobsParameters|obligation-3", name: "jobs unsorted states", build: func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"states":[]`, `"states":["running","queued"]`))
		}},
		{site: "query.go|checkJobsParameters|obligation-4", name: "jobs pwned state refused", build: func(t *testing.T) string {
			return refuseBatch("jobs", badParams(t, "jobs", `"states":[]`, `"states":["pwned"]`))
		}},
		{site: "query.go|checkPlansParameters|obligation-1", name: "plans bad plan identifier", build: func(t *testing.T) string {
			return refuseBatch("plans", badParams(t, "plans", `"plan_ids":[]`, `"plan_ids":["zzz"]`))
		}},
		{site: "query.go|checkPlansParameters|obligation-2", name: "plans bad operation identifier", build: func(t *testing.T) string {
			return refuseBatch("plans", badParams(t, "plans", `"operation_ids":[]`, `"operation_ids":["zzz"]`))
		}},
		{site: "query.go|checkDistinctParameters|obligation-1", name: "distinct non-string field", build: func(t *testing.T) string {
			return refuseBatch("distinct", badParams(t, "distinct", `"field":"kind"`, `"field":0`))
		}},
		{site: "query.go|checkDistinctParameters|obligation-2", name: "distinct secret field refused", build: func(t *testing.T) string {
			return refuseBatch("distinct", badParams(t, "distinct", `"field":"kind"`, `"field":"secret"`))
		}},
		{site: "query.go|checkTagsParameter|obligation-1", name: "tags past count bound", build: func(t *testing.T) string {
			return refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":`+sortedLiterals("tag", 257)))
		}},
		{site: "query.go|checkTagsParameter|obligation-2", name: "tags non-string element", build: func(t *testing.T) string {
			return refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":["a",0]`))
		}},
		{site: "query.go|checkTagsParameter|obligation-3", name: "tags unsorted", build: func(t *testing.T) string {
			return refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":["b","a"]`))
		}},
		{site: "query.go|checkEnrichParameters|obligation-1", name: "enrich bad profile", build: func(t *testing.T) string {
			return refuseBatch("enrich", badParams(t, "enrich", `"profile_id":`+quote(fixtureBatchDigest), `"profile_id":"zzz"`))
		}},
		{site: "query.go|checkEnrichParameters|obligation-2", name: "enrich empty kinds", build: func(t *testing.T) string {
			return refuseBatch("enrich", badParams(t, "enrich", `"kinds":["summary"]`, `"kinds":[]`))
		}},
		{site: "query.go|checkEnrichParameters|obligation-3", name: "enrich bad kind", build: func(t *testing.T) string {
			return refuseBatch("enrich", badParams(t, "enrich", `"kinds":["summary"]`, `"kinds":["pwned"]`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-1", name: "plan continue bad source digest", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"source_instance_id":`+quote(fixtureBatchDigest), `"source_instance_id":"zzz"`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-2", name: "plan continue bad host identifier", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"to_host_id":`+quote(fixtureOriginHost), `"to_host_id":"not-a-uuid"`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-3", name: "plan continue bad installation digest", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"to_installation_id":`+quote(fixtureBatchDigest), `"to_installation_id":"zzz"`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-4", name: "plan continue non-string intent", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"intent":"resume"`, `"intent":0`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-5", name: "plan continue bad intent", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"intent":"resume"`, `"intent":"teleport"`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-6", name: "plan continue non-string workspace policy", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"workspace_policy":"exact_checkpoint"`, `"workspace_policy":0`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-7", name: "plan continue bad workspace policy", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"workspace_policy":"exact_checkpoint"`, `"workspace_policy":"whatever"`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-8", name: "plan continue non-string after-success", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"source_after_success":"retain"`, `"source_after_success":0`))
		}},
		{site: "query.go|checkPlanContinueParameters|obligation-9", name: "plan continue bad after-success", build: func(t *testing.T) string {
			return refuseBatch("plan_continue", badParams(t, "plan_continue", `"source_after_success":"retain"`, `"source_after_success":"vanish"`))
		}},
		{site: "query.go|checkExecutePlanParameters|obligation-1", name: "execute plan bad plan digest", build: func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"plan_id":`+quote(fixtureBatchDigest), `"plan_id":"zzz"`))
		}},
		{site: "query.go|checkExecutePlanParameters|obligation-2", name: "execute plan bad operation identifier", build: func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"operation_id":`+quote(fixtureOperationID), `"operation_id":"not-a-uuid"`))
		}},
		{site: "query.go|checkExecutePlanParameters|obligation-3", name: "execute plan past confirmations bound", build: func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":`+sortedLiterals("c", 65)))
		}},
		{site: "query.go|checkExecutePlanParameters|obligation-4", name: "execute plan non-string confirmation", build: func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":["a",0]`))
		}},
		{site: "query.go|checkExecutePlanParameters|obligation-5", name: "execute plan unsorted confirmations", build: func(t *testing.T) string {
			return refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":["b","a"]`))
		}},
		{site: "query.go|checkFilters|obligation-1", name: "filters non-object", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", `{"filters":[],"extensions":{}}`, "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkFilters|obligation-2", name: "filters unknown member", build: func(t *testing.T) string {
			return filtersWith(t, `"freshness":[]`, `"freshness":[],"fuzz":[]`)
		}},
		{site: "query.go|checkFilters|obligation-3", name: "filters missing member", build: func(t *testing.T) string {
			filters := strings.TrimSuffix(validFiltersJSON, `,"extensions":{}}`) + `}`
			return singleOperationBatch("sessions", `{"filters":`+filters+`,"extensions":{}}`, "null", "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkFilters|obligation-4", name: "filters bad kind", build: func(t *testing.T) string {
			// Layered delegate: the kinds shape fails inside
			// checkEnumSubset, so this witness exercises the
			// filters gate and the enum vocabulary obligation
			// together; the battery shows the pair.
			return filtersWith(t, `"kinds":[]`, `"kinds":["bogus"]`)
		}},
		{site: "query.go|checkFilters|obligation-5", name: "filters bad lineage anchor", build: func(t *testing.T) string {
			// Layered delegate over checkUUIDv7DigestSubset:
			// exercises the filters gate and the union
			// obligation together.
			return filtersWith(t, `"lineage_anchors":[]`, `"lineage_anchors":["zzz"]`)
		}},
		{site: "query.go|checkFilters|obligation-6", name: "filters unsorted provider identifiers", build: func(t *testing.T) string {
			return filtersWith(t, `"provider_ids":[]`, `"provider_ids":["b-1","a-1"]`)
		}},
		{site: "query.go|checkFilters|obligation-7", name: "filters bad provider identifier", build: func(t *testing.T) string {
			return filtersWith(t, `"provider_ids":[]`, `"provider_ids":["BAD"]`)
		}},
		{site: "query.go|checkFilters|obligation-8", name: "filters bad host identifier", build: func(t *testing.T) string {
			return filtersWith(t, `"host_ids":[]`, `"host_ids":["not-a-uuid"]`)
		}},
		{site: "query.go|checkFilters|obligation-9", name: "filters bad workspace identifier", build: func(t *testing.T) string {
			return filtersWith(t, `"workspace_ids":[]`, `"workspace_ids":["not-a-uuid"]`)
		}},
		{site: "query.go|checkFilters|obligation-10", name: "filters unsorted states", build: func(t *testing.T) string {
			return filtersWith(t, `"states":[]`, `"states":["b","a"]`)
		}},
		{site: "query.go|checkFilters|obligation-11", name: "filters bad management state", build: func(t *testing.T) string {
			// Layered delegate over checkEnumSubset.
			return filtersWith(t, `"management_states":[]`, `"management_states":["bogus"]`)
		}},
		{site: "query.go|checkFilters|obligation-12", name: "filters bad reachability", build: func(t *testing.T) string {
			// Layered delegate over checkEnumSubset.
			return filtersWith(t, `"reachability":[]`, `"reachability":["everywhere"]`)
		}},
		{site: "query.go|checkFilters|obligation-13", name: "filters bad freshness", build: func(t *testing.T) string {
			// Layered delegate over checkEnumSubset.
			return filtersWith(t, `"freshness":[]`, `"freshness":["bogus"]`)
		}},
		{site: "query.go|checkFilters|obligation-14", name: "filters warning past element bound", build: func(t *testing.T) string {
			return filtersWith(t, `"warnings":[]`, `"warnings":[`+quote(strings.Repeat("w", 257))+`]`)
		}},
		{site: "query.go|checkFilters|obligation-15", name: "filters bad updated-before timestamp", build: func(t *testing.T) string {
			return filtersWith(t, `"updated_before":null`, `"updated_before":"yesterday"`)
		}},
		{site: "query.go|checkFilters|obligation-16", name: "filters bad updated-after timestamp", build: func(t *testing.T) string {
			return filtersWith(t, `"updated_after":null`, `"updated_after":"yesterday"`)
		}},
		{site: "query.go|checkEnumSubset|obligation-1", name: "enum subset past count bound", build: func(t *testing.T) string {
			return filtersWith(t, `"freshness":[]`, `"freshness":`+sortedLiterals("f", 8))
		}},
		{site: "query.go|checkEnumSubset|obligation-2", name: "enum subset outside vocabulary", build: func(t *testing.T) string {
			return filtersWith(t, `"reachability":[]`, `"reachability":["everywhere"]`)
		}},
		{site: "query.go|checkUUIDv7DigestSubset|obligation-1", name: "union subset non-array", build: func(t *testing.T) string {
			return filtersWith(t, `"lineage_anchors":[]`, `"lineage_anchors":{}`)
		}},
		{site: "query.go|checkUUIDv7DigestSubset|obligation-2", name: "union subset past count bound", build: func(t *testing.T) string {
			many := make([]string, 0, 257)
			for i := 0; i < 257; i++ {
				many = append(many, `"a"`)
			}
			return filtersWith(t, `"lineage_anchors":[]`, `"lineage_anchors":[`+strings.Join(many, ",")+`]`)
		}},
		{site: "query.go|checkUUIDv7DigestSubset|obligation-3", name: "union subset non-string element", build: func(t *testing.T) string {
			return filtersWith(t, `"lineage_anchors":[]`, `"lineage_anchors":[`+quote(fixtureBatchDigest)+`,0]`)
		}},
		{site: "query.go|checkUUIDv7DigestSubset|obligation-4", name: "union subset outside the union", build: func(t *testing.T) string {
			return filtersWith(t, `"lineage_anchors":[]`, `"lineage_anchors":["zzz"]`)
		}},
		{site: "query.go|checkUUIDv7DigestSubset|obligation-5", name: "union subset unsorted", build: func(t *testing.T) string {
			pair := []string{fixtureInstallDigestA, fixtureInstallDigestB}
			sort.Strings(pair)
			return filtersWith(t, `"lineage_anchors":[]`, `"lineage_anchors":[`+quote(pair[1])+`,`+quote(pair[0])+`]`)
		}},
		{site: "query.go|checkQueryProjection|obligation-1", name: "fields and preset together", build: func(t *testing.T) string {
			// Shared with the projection-gate row: the gate
			// fires exactly when this obligation does, so one
			// vector witnesses both; the battery shows the AND.
			return singleOperationBatch("sessions", operationParams("sessions"), `["id"]`, `"overview"`, 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryProjection|obligation-2", name: "fields past count bound", build: func(t *testing.T) string {
			// 135 fields cycling the 27-member registry: every
			// element is registry-valid, so only the
			// sorted/count gate can fire — count, sortedness,
			// or both. Removing the gate admits.
			cycle := []string{"id", "kind", "lineage_anchor", "management_state", "display_title", "title_source", "provider", "host", "workspace", "state", "owner", "local_role", "updated_at", "summary", "recent_activity", "last_user_intent", "open_loops", "annotation_freshness", "inventory_freshness", "reachability", "branch_count", "clone_count", "warnings", "available_intents", "lineage_graph", "live_runtime", "preview"}
			fields := make([]string, 0, 135)
			for i := 0; i < 135; i++ {
				fields = append(fields, quote(cycle[i%len(cycle)]))
			}
			return singleOperationBatch("sessions", operationParams("sessions"), `[`+strings.Join(fields, ",")+`]`, "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryProjection|obligation-3", name: "projection unknown field", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), `["id","nope"]`, "null", 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryProjection|obligation-4", name: "projection unknown preset", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", `"everything"`, 0, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryPagination|obligation-1", name: "skip past bound", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 1000001, 10, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryPagination|obligation-2", name: "take below bound", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 0, "[]", "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-1", name: "sort past count bound", build: func(t *testing.T) string {
			tuple := `{"field":"stable_id","direction":"asc","extensions":{}}`
			sorts := `[` + strings.Repeat(tuple+`,`, 8) + tuple + `]`
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, sorts, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-2", name: "sort non-object element", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[0]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-3", name: "sort unknown member", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":"stable_id","direction":"asc","extensions":{},"extra":1}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-4", name: "sort missing member", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":"stable_id","extensions":{}}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-5", name: "sort non-string field", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":0,"direction":"asc","extensions":{}}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-6", name: "sort unknown field", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":"vibes","direction":"asc","extensions":{}}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-7", name: "sort unknown direction", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":"stable_id","direction":"sideways","extensions":{}}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQuerySort|obligation-8", name: "sort bad extensions", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[{"field":"stable_id","direction":"asc","extensions":{"x":1}}]`, "false", "false", "null", "null")
		}},
		{site: "query.go|checkQueryFlags|obligation-1", name: "flags non-bool dry run", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, "[]", "0", "false", "null", "null")
		}},
		{site: "query.go|checkQueryFlags|obligation-2", name: "flags non-bool confirm", build: func(t *testing.T) string {
			return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, "[]", "false", "0", "null", "null")
		}},
		{site: "query.go|checkQueryFlags|obligation-3", name: "mutation flags agree without effect", build: func(t *testing.T) string {
			return mutationBatch("set_title", operationParams("set_title"), false, false, "null", "null")
		}},
	}
}
