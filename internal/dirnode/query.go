package dirnode

import (
	"encoding/json"
	"regexp"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Section 10.8.5 Session Directory Query:
// the closed request envelope, the caller context shape, the
// seventeen-operation registry with its name-matched parameter
// union, the field/preset projection rules, the read/mutation flag
// rules, and the cursor binding.
//
// The parser rejects the whole batch before execution on unknown
// syntax, operation, parameter, field, preset, filter, sort key, or
// bound: every check below runs before any operation is admitted,
// and DecodeQuery returns either a fully validated batch or one
// refusal. CallerContext is shape-checked here and authenticated
// server-side by the caller; it is never accepted from an
// unverified body alone.

const (
	// QuerySchema is the exact schema member a directory query
	// carries.
	QuerySchema = "urn:ax:schema:session-directory-query"
	// QuerySchemaVersion is the only query schema_version this
	// host accepts.
	QuerySchemaVersion = "1.0.0"
)

// queryMembers is the exact Session Directory Query member set.
var queryMembers = map[string]bool{
	"schema":         true,
	"schema_version": true,
	"query_id":       true,
	"operations":     true,
	"caller":         true,
	"extensions":     true,
}

// queryRequired lists queryMembers in a fixed order.
var queryRequired = []string{
	"schema",
	"schema_version",
	"query_id",
	"operations",
	"caller",
	"extensions",
}

// callerMembers is the exact CallerContext member set.
var callerMembers = map[string]bool{
	"caller_id":                true,
	"authentication_subject":   true,
	"origin_host_id":           true,
	"interaction":              true,
	"scopes":                   true,
	"disclosure_policy_digest": true,
	"extensions":               true,
}

// callerRequired lists callerMembers in a fixed order.
var callerRequired = []string{
	"caller_id",
	"authentication_subject",
	"origin_host_id",
	"interaction",
	"scopes",
	"disclosure_policy_digest",
	"extensions",
}

// operationMembers is the exact QueryOperation member set.
var operationMembers = map[string]bool{
	"operation_index":    true,
	"name":               true,
	"parameters":         true,
	"fields":             true,
	"preset":             true,
	"skip":               true,
	"take":               true,
	"sort":               true,
	"dry_run":            true,
	"confirm":            true,
	"expectation_digest": true,
	"idempotency_key":    true,
	"extensions":         true,
}

// operationRequired lists operationMembers in a fixed order.
var operationRequired = []string{
	"operation_index",
	"name",
	"parameters",
	"fields",
	"preset",
	"skip",
	"take",
	"sort",
	"dry_run",
	"confirm",
	"expectation_digest",
	"idempotency_key",
	"extensions",
}

// filterMembers is the exact DirectoryFilters member set.
var filterMembers = map[string]bool{
	"kinds":             true,
	"lineage_anchors":   true,
	"provider_ids":      true,
	"host_ids":          true,
	"workspace_ids":     true,
	"states":            true,
	"management_states": true,
	"reachability":      true,
	"freshness":         true,
	"warnings":          true,
	"updated_before":    true,
	"updated_after":     true,
	"extensions":        true,
}

// filterRequired lists filterMembers in a fixed order.
var filterRequired = []string{
	"kinds",
	"lineage_anchors",
	"provider_ids",
	"host_ids",
	"workspace_ids",
	"states",
	"management_states",
	"reachability",
	"freshness",
	"warnings",
	"updated_before",
	"updated_after",
	"extensions",
}

// sortMembers is the exact QuerySort member set.
var sortMembers = map[string]bool{
	"field":      true,
	"direction":  true,
	"extensions": true,
}

// sortRequired lists sortMembers in a fixed order.
var sortRequired = []string{
	"field",
	"direction",
	"extensions",
}

// readOperations is the closed read registry of Section 10.8.5 in
// the section's listed order.
var readOperations = []string{
	"schema",
	"sessions",
	"session",
	"lineage",
	"hosts",
	"environments",
	"jobs",
	"plans",
	"count",
	"distinct",
	"directory_summary",
}

// mutationOperations is the closed mutation registry in the
// section's listed order.
var mutationOperations = []string{
	"set_title",
	"set_tags",
	"set_pin",
	"enrich",
	"plan_continue",
	"execute_plan",
}

// queryOperationKind names which half of the registry an operation
// belongs to. Reads and mutations carry different flag,
// projection, and pagination rules.
type queryOperationKind string

const (
	queryKindRead     queryOperationKind = "read"
	queryKindMutation queryOperationKind = "mutation"
)

// queryOperationKindOf reports the registry half a name belongs to.
// Both registries are table loops so the census derives them: an
// inline chain would hide a widened vocabulary from the inventory.
func queryOperationKindOf(name string) (queryOperationKind, bool) {
	for _, allowed := range readOperations {
		if name == allowed {
			return queryKindRead, true
		}
	}
	for _, allowed := range mutationOperations {
		if name == allowed {
			return queryKindMutation, true
		}
	}
	return "", false
}

// listOperations names the read operations that may use nonzero
// skip or take other than 1: the entity-listing reads. The
// singular, schema, and aggregate reads require skip=0 and take=1.
// The section states the rule without enumerating the set, so the
// choice is recorded here: schema, session, count, distinct, and
// directory_summary address one object, one registry, or one number
// and have no list to page. A different reading would move lineage
// or distinct across the line; the refusal detail names the
// operation either way.
var listOperations = []string{
	"sessions",
	"lineage",
	"hosts",
	"environments",
	"jobs",
	"plans",
}

// isListOperation reports whether the read operation pages a list.
func isListOperation(name string) bool {
	for _, allowed := range listOperations {
		if name == allowed {
			return true
		}
	}
	return false
}

// annotationMutations names the mutations that require an
// expectation digest and idempotency key when confirmed.
var annotationMutations = []string{
	"set_title",
	"set_tags",
	"set_pin",
}

// isAnnotationMutation reports whether the mutation is an
// annotation mutation.
func isAnnotationMutation(name string) bool {
	for _, allowed := range annotationMutations {
		if name == allowed {
			return true
		}
	}
	return false
}

// queryParameterMembers maps each operation to its exact
// parameters member set in Section 10.8.5 table order. The schema
// and directory_summary bodies carry extensions only: there is no
// untyped parameter bag.
var queryParameterMembers = map[string][]string{
	"schema":            {"extensions"},
	"directory_summary": {"extensions"},
	"sessions":          {"filters", "extensions"},
	"count":             {"filters", "extensions"},
	"session":           {"subject_kind", "subject_id", "extensions"},
	"lineage":           {"anchor_id", "include_suggestions", "extensions"},
	"hosts":             {"host_ids", "reachable", "extensions"},
	"environments":      {"host_ids", "environment_ids", "authentication_status", "extensions"},
	"jobs":              {"job_ids", "profile_ids", "states", "extensions"},
	"plans":             {"plan_ids", "operation_ids", "include_expired", "extensions"},
	"distinct":          {"field", "filters", "extensions"},
	"set_title":         {"subject_kind", "subject_id", "title", "supersedes_annotation_ids", "extensions"},
	"set_tags":          {"subject_kind", "subject_id", "tags", "supersedes_annotation_ids", "extensions"},
	"set_pin":           {"subject_kind", "subject_id", "value", "supersedes_annotation_ids", "extensions"},
	"enrich":            {"subject_kind", "subject_id", "profile_id", "kinds", "expected_head_digest", "extensions"},
	"plan_continue":     {"subject_kind", "subject_id", "source_instance_id", "to_host_id", "to_installation_id", "intent", "workspace_policy", "source_after_success", "extensions"},
	"execute_plan":      {"plan_id", "operation_id", "confirmations", "extensions"},
}

// queryFields is the closed directory-field registry: the
// twenty-four projection fields plus the three explicitly lazy
// ones. Fields and presets apply only to the projecting reads;
// other operations require both null.
var queryFields = []string{
	"id",
	"kind",
	"lineage_anchor",
	"management_state",
	"display_title",
	"title_source",
	"provider",
	"host",
	"workspace",
	"state",
	"owner",
	"local_role",
	"updated_at",
	"summary",
	"recent_activity",
	"last_user_intent",
	"open_loops",
	"annotation_freshness",
	"inventory_freshness",
	"reachability",
	"branch_count",
	"clone_count",
	"warnings",
	"available_intents",
	"lineage_graph",
	"live_runtime",
	"preview",
}

// validQueryField reports whether the field is a registry member.
func validQueryField(field string) bool {
	for _, allowed := range queryFields {
		if field == allowed {
			return true
		}
	}
	return false
}

// queryPresets is the closed preset registry.
var queryPresets = []string{
	"minimal",
	"overview",
	"activity",
	"routing",
	"full",
}

// validQueryPreset reports whether the preset is a registry member.
func validQueryPreset(preset string) bool {
	for _, allowed := range queryPresets {
		if preset == allowed {
			return true
		}
	}
	return false
}

// querySortFields is the closed QuerySort field registry.
var querySortFields = []string{
	"display_title",
	"provider",
	"host",
	"workspace",
	"state",
	"updated_at",
	"annotation_freshness",
	"inventory_freshness",
	"reachability",
	"stable_id",
}

// queryScopes is the closed CallerContext scope registry.
var queryScopes = []string{
	"directory.read",
	"directory.preview",
	"directory.mutate",
	"directory.execute",
	"directory.admin",
}

// providerIDPattern is the provider-id grammar:
// [a-z][a-z0-9-]{0,31}.
var providerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// CallerContext is one validated query caller: the shape-checked
// identity the caller authenticates server-side.
type CallerContext struct {
	CallerID         string
	AuthSubject      string
	OriginHostID     string
	Interaction      string
	Scopes           []string
	DisclosurePolicy string
}

// QueryOperation is one validated batch operation: its position,
// registry name, and half.
type QueryOperation struct {
	Index int
	Name  string
	Kind  queryOperationKind
}

// Query is one validated Session Directory Query: the batch
// identifier, the position-indexed operations, and the caller.
type Query struct {
	QueryID    string
	Operations []QueryOperation
	Caller     CallerContext
}

// DecodeQuery validates one Session Directory Query body as the
// closed Section 10.8.5 contract and rejects the whole batch
// before execution on the first defect: unknown syntax, operation,
// parameter, field, preset, filter, sort key, or bound. For a
// batch of N operations each operation_index must equal its
// zero-based array position; duplicate, sparse, reordered, or
// position-mismatched indexes invalidate the whole query.
func DecodeQuery(body []byte) (Query, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(body))
	if fault != nil {
		failure, err := failQuery("directory query "+fault.detail, fault.member)
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	if name, unknown := unknownMember(members, queryMembers); unknown {
		failure, err := failQuery("directory query carries unknown member", name)
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	if name, missing := missingMember(members, queryRequired); missing {
		failure, err := failQuery("directory query misses a required member", name)
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	if schema, ok := rawString(members["schema"]); !ok || schema != QuerySchema {
		failure, err := failQuery("directory query schema is not the session directory query", "schema")
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	if version, ok := rawString(members["schema_version"]); !ok || version != QuerySchemaVersion {
		failure, err := failQuery("directory query version is not 1.0.0", "schema_version")
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	queryID, ok := checkUUIDv7(members["query_id"])
	if !ok {
		failure, err := failQuery("directory query identifier is not a UUIDv7", "query_id")
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	elements, ok := decodeArray(members["operations"])
	if !ok || len(elements) < 1 || len(elements) > 64 {
		failure, err := failQuery("directory query operations are not QueryOperation[1..64]", "operations")
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	caller, ok := checkCaller(members["caller"])
	if !ok {
		failure, err := failQuery("directory query caller is not a closed CallerContext", "caller")
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failQuery("directory query extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return Query{}, err
		}
		return Query{}, failure
	}
	operations := make([]QueryOperation, 0, len(elements))
	for position, element := range elements {
		operation, ok := checkQueryOperation(element, position)
		if !ok {
			failure, err := failQuery("directory query operation is not a closed QueryOperation", "operations")
			if err != nil {
				return Query{}, err
			}
			return Query{}, failure
		}
		operations = append(operations, operation)
	}
	return Query{QueryID: queryID.String(), Operations: operations, Caller: caller}, nil
}

// checkCaller validates one CallerContext object.
func checkCaller(raw json.RawMessage) (CallerContext, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return CallerContext{}, false
	}
	if name, unknown := unknownMember(members, callerMembers); unknown {
		_ = name
		return CallerContext{}, false
	}
	if name, missing := missingMember(members, callerRequired); missing {
		_ = name
		return CallerContext{}, false
	}
	callerID, ok := checkStringBounds(members["caller_id"], 1, 256)
	if !ok {
		return CallerContext{}, false
	}
	subject, ok := checkStringBounds(members["authentication_subject"], 1, 512)
	if !ok {
		return CallerContext{}, false
	}
	origin, ok := checkUUIDv7(members["origin_host_id"])
	if !ok {
		return CallerContext{}, false
	}
	interaction, ok := rawString(members["interaction"])
	if !ok || (interaction != "interactive" && interaction != "non_interactive") {
		return CallerContext{}, false
	}
	scopes, ok := checkSortedUniqueStrings(members["scopes"], 1, 64, 1, 5)
	if !ok {
		return CallerContext{}, false
	}
	for _, scope := range scopes {
		allowed := false
		for _, candidate := range queryScopes {
			if scope == candidate {
				allowed = true
				break
			}
		}
		if !allowed {
			return CallerContext{}, false
		}
	}
	policy, ok := checkDigest(members["disclosure_policy_digest"])
	if !ok {
		return CallerContext{}, false
	}
	if !checkExtensions(members["extensions"]) {
		return CallerContext{}, false
	}
	return CallerContext{
		CallerID:         callerID,
		AuthSubject:      subject,
		OriginHostID:     origin.String(),
		Interaction:      interaction,
		Scopes:           scopes,
		DisclosurePolicy: policy.String(),
	}, true
}

// checkQueryOperation validates one batch element against its
// position: the exact closed members, the index equal to the
// zero-based array position, a registry name, the name-matched
// parameter union, the projection rules, the pagination rule, and
// the read/mutation flag rules.
func checkQueryOperation(raw json.RawMessage, position int) (QueryOperation, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return QueryOperation{}, false
	}
	if name, unknown := unknownMember(members, operationMembers); unknown {
		_ = name
		return QueryOperation{}, false
	}
	if name, missing := missingMember(members, operationRequired); missing {
		_ = name
		return QueryOperation{}, false
	}
	index, ok := rawUint53(members["operation_index"])
	if !ok || index > 63 || int(index) != position {
		return QueryOperation{}, false
	}
	name, ok := rawString(members["name"])
	if !ok {
		return QueryOperation{}, false
	}
	kind, ok := queryOperationKindOf(name)
	if !ok {
		return QueryOperation{}, false
	}
	if !checkQueryParameters(name, members["parameters"]) {
		return QueryOperation{}, false
	}
	if !checkQueryProjection(name, kind, members["fields"], members["preset"]) {
		return QueryOperation{}, false
	}
	if !checkQueryPagination(name, members["skip"], members["take"]) {
		return QueryOperation{}, false
	}
	if !checkQuerySort(members["sort"]) {
		return QueryOperation{}, false
	}
	if !checkQueryFlags(name, kind, members) {
		return QueryOperation{}, false
	}
	if !checkExtensions(members["extensions"]) {
		return QueryOperation{}, false
	}
	return QueryOperation{Index: position, Name: name, Kind: kind}, true
}

// checkQueryParameters validates the name-matched parameter union:
// the exact member set for the operation plus the per-member
// shapes. There is no untyped parameter bag.
func checkQueryParameters(name string, raw json.RawMessage) bool {
	want, known := queryParameterMembers[name]
	if !known {
		return false
	}
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return false
	}
	allowed := map[string]bool{}
	for _, member := range want {
		allowed[member] = true
	}
	if member, unknown := unknownMember(members, allowed); unknown {
		_ = member
		return false
	}
	if member, missing := missingMember(members, want); missing {
		_ = member
		return false
	}
	if !checkExtensions(members["extensions"]) {
		return false
	}
	switch name {
	case "sessions", "count":
		return checkFilters(members["filters"])
	case "session":
		return checkSubject(members["subject_kind"], members["subject_id"], false)
	case "lineage":
		return checkLineageParameters(members)
	case "hosts":
		return checkHostsParameters(members)
	case "environments":
		return checkEnvironmentsParameters(members)
	case "jobs":
		return checkJobsParameters(members)
	case "plans":
		return checkPlansParameters(members)
	case "distinct":
		return checkDistinctParameters(members)
	case "set_title":
		return checkSubject(members["subject_kind"], members["subject_id"], true) &&
			checkTitleParameter(members) &&
			checkSupersedes(members["supersedes_annotation_ids"])
	case "set_tags":
		return checkSubject(members["subject_kind"], members["subject_id"], true) &&
			checkTagsParameter(members) &&
			checkSupersedes(members["supersedes_annotation_ids"])
	case "set_pin":
		return checkSubject(members["subject_kind"], members["subject_id"], true) &&
			checkPinParameter(members) &&
			checkSupersedes(members["supersedes_annotation_ids"])
	case "enrich":
		return checkSubject(members["subject_kind"], members["subject_id"], true) &&
			checkEnrichParameters(members)
	case "plan_continue":
		return checkSubject(members["subject_kind"], members["subject_id"], true) &&
			checkPlanContinueParameters(members)
	case "execute_plan":
		return checkExecutePlanParameters(members)
	default:
		return true
	}
}

// checkSubject validates a subject_kind/subject_id pair. Mutations
// additionally admit the lineage kind.
func checkSubject(kindRaw, idRaw json.RawMessage, mutation bool) bool {
	kind, ok := rawString(kindRaw)
	if !ok {
		return false
	}
	switch kind {
	case "ax_session", "native_instance":
	case "lineage":
		if !mutation {
			return false
		}
	default:
		return false
	}
	identifier, ok := rawString(idRaw)
	if !ok {
		return false
	}
	if _, err := scalar.ParseUUIDv7(identifier); err == nil {
		return true
	}
	_, digestErr := checkDigestString(identifier)
	return digestErr
}

// checkLineageParameters validates the lineage parameter union
// member.
func checkLineageParameters(members map[string]json.RawMessage) bool {
	anchor, ok := rawString(members["anchor_id"])
	if !ok {
		return false
	}
	if _, err := scalar.ParseUUIDv7(anchor); err != nil {
		if _, digestErr := checkDigestString(anchor); !digestErr {
			return false
		}
	}
	_, ok = rawBool(members["include_suggestions"])
	return ok
}

// checkHostsParameters validates the hosts parameter union member.
func checkHostsParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkSortedUniqueUUIDv7(members["host_ids"], 0, 256); !ok {
		return false
	}
	if isNull(members["reachable"]) {
		return true
	}
	_, ok := rawBool(members["reachable"])
	return ok
}

// checkEnvironmentsParameters validates the environments parameter
// union member.
func checkEnvironmentsParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkSortedUniqueUUIDv7(members["host_ids"], 0, 256); !ok {
		return false
	}
	environments, ok := checkSortedUniqueStrings(members["environment_ids"], 1, 64, 0, 64)
	if !ok {
		return false
	}
	for _, environment := range environments {
		if !environmentIDPattern.MatchString(environment) {
			return false
		}
	}
	statuses, ok := checkSortedUniqueStrings(members["authentication_status"], 1, 32, 0, 4)
	if !ok {
		return false
	}
	for _, status := range statuses {
		switch status {
		case "available", "missing", "expired", "unknown":
		default:
			return false
		}
	}
	return true
}

// checkJobsParameters validates the jobs parameter union member.
func checkJobsParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkSortedUniqueUUIDv7(members["job_ids"], 0, 256); !ok {
		return false
	}
	if _, ok := checkSortedUniqueDigests(members["profile_ids"], 0, 256); !ok {
		return false
	}
	states, ok := checkSortedUniqueStrings(members["states"], 1, 32, 0, 7)
	if !ok {
		return false
	}
	for _, state := range states {
		switch state {
		case "queued", "claimed", "running", "succeeded", "superseded", "failed", "canceled":
		default:
			return false
		}
	}
	return true
}

// checkPlansParameters validates the plans parameter union member.
func checkPlansParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkSortedUniqueDigests(members["plan_ids"], 0, 256); !ok {
		return false
	}
	if _, ok := checkSortedUniqueUUIDv7(members["operation_ids"], 0, 256); !ok {
		return false
	}
	_, ok := rawBool(members["include_expired"])
	return ok
}

// checkDistinctParameters validates the distinct parameter union
// member.
func checkDistinctParameters(members map[string]json.RawMessage) bool {
	field, ok := rawString(members["field"])
	if !ok {
		return false
	}
	switch field {
	case "kind", "lineage_anchor", "provider", "host", "workspace", "state", "management_state", "reachability", "freshness", "warning":
	default:
		return false
	}
	return checkFilters(members["filters"])
}

// checkTitleParameter validates the set_title title member.
func checkTitleParameter(members map[string]json.RawMessage) bool {
	_, ok := checkStringBounds(members["title"], 1, 512)
	return ok
}

// checkTagsParameter validates the set_tags tags member: a sorted
// unique string array in the count bound. The section bounds the
// count, not the items, so items cross as strings.
func checkTagsParameter(members map[string]json.RawMessage) bool {
	elements, ok := decodeArray(members["tags"])
	if !ok || len(elements) > 256 {
		return false
	}
	var previous string
	for index, element := range elements {
		tag, ok := rawString(element)
		if !ok {
			return false
		}
		if index > 0 && previous >= tag {
			return false
		}
		previous = tag
	}
	return true
}

// checkPinParameter validates the set_pin value member.
func checkPinParameter(members map[string]json.RawMessage) bool {
	_, ok := rawBool(members["value"])
	return ok
}

// checkSupersedes validates a supersedes_annotation_ids member.
func checkSupersedes(raw json.RawMessage) bool {
	_, ok := checkSortedUniqueDigests(raw, 0, 1024)
	return ok
}

// checkEnrichParameters validates the enrich parameter union
// member beyond the subject pair.
func checkEnrichParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkDigest(members["profile_id"]); !ok {
		return false
	}
	kinds, ok := checkSortedUniqueStrings(members["kinds"], 1, 64, 1, 3)
	if !ok {
		return false
	}
	for _, kind := range kinds {
		switch kind {
		case "generated_title", "summary", "recent_activity":
		default:
			return false
		}
	}
	_, ok = checkDigest(members["expected_head_digest"])
	return ok
}

// checkPlanContinueParameters validates the plan_continue
// parameter union member beyond the subject pair.
func checkPlanContinueParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkDigest(members["source_instance_id"]); !ok {
		return false
	}
	if _, ok := checkUUIDv7(members["to_host_id"]); !ok {
		return false
	}
	if _, ok := checkDigest(members["to_installation_id"]); !ok {
		return false
	}
	intent, ok := rawString(members["intent"])
	if !ok {
		return false
	}
	switch intent {
	case "attach", "resume", "takeover", "fork", "adopt", "clone", "move", "open_unmanaged", "archive_context":
	default:
		return false
	}
	policy, ok := rawString(members["workspace_policy"])
	if !ok {
		return false
	}
	switch policy {
	case "refuse", "exact_checkpoint", "materialize_copy":
	default:
		return false
	}
	after, ok := rawString(members["source_after_success"])
	if !ok {
		return false
	}
	switch after {
	case "retain", "stop_and_release":
	default:
		return false
	}
	return true
}

// checkExecutePlanParameters validates the execute_plan parameter
// union member.
func checkExecutePlanParameters(members map[string]json.RawMessage) bool {
	if _, ok := checkDigest(members["plan_id"]); !ok {
		return false
	}
	if _, ok := checkUUIDv7(members["operation_id"]); !ok {
		return false
	}
	elements, ok := decodeArray(members["confirmations"])
	if !ok || len(elements) > 64 {
		return false
	}
	var previous string
	for index, element := range elements {
		confirmation, ok := rawString(element)
		if !ok {
			return false
		}
		if index > 0 && previous >= confirmation {
			return false
		}
		previous = confirmation
	}
	return true
}

// checkFilters validates one DirectoryFilters object: the exact
// closed members with per-member shapes.
func checkFilters(raw json.RawMessage) bool {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return false
	}
	if name, unknown := unknownMember(members, filterMembers); unknown {
		_ = name
		return false
	}
	if name, missing := missingMember(members, filterRequired); missing {
		_ = name
		return false
	}
	if !checkEnumSubset(members["kinds"], []string{"lineage", "managed_session", "native_instance"}, 1, 64, 0, 3) {
		return false
	}
	if !checkUUIDv7DigestSubset(members["lineage_anchors"], 0, 256) {
		return false
	}
	providers, ok := checkSortedUniqueStrings(members["provider_ids"], 1, 32, 0, 64)
	if !ok {
		return false
	}
	for _, provider := range providers {
		if !providerIDPattern.MatchString(provider) {
			return false
		}
	}
	if _, ok := checkSortedUniqueUUIDv7(members["host_ids"], 0, 256); !ok {
		return false
	}
	if _, ok := checkSortedUniqueUUIDv7(members["workspace_ids"], 0, 256); !ok {
		return false
	}
	if _, ok := checkSortedUniqueStrings(members["states"], 1, 128, 0, 64); !ok {
		return false
	}
	if !checkEnumSubset(members["management_states"], []string{"managed", "unmanaged", "conflicted"}, 1, 32, 0, 3) {
		return false
	}
	if !checkEnumSubset(members["reachability"], []string{"local", "reachable", "unreachable", "unknown"}, 1, 32, 0, 4) {
		return false
	}
	if !checkEnumSubset(members["freshness"], []string{"current", "aging", "stale", "offline", "partial", "conflicted", "unknown"}, 1, 32, 0, 7) {
		return false
	}
	if _, ok := checkSortedUniqueStrings(members["warnings"], 1, 256, 0, 128); !ok {
		return false
	}
	if !isNull(members["updated_before"]) {
		if _, ok := checkTimestamp(members["updated_before"]); !ok {
			return false
		}
	}
	if !isNull(members["updated_after"]) {
		if _, ok := checkTimestamp(members["updated_after"]); !ok {
			return false
		}
	}
	return checkExtensions(members["extensions"])
}

// checkEnumSubset reports whether the member is a sorted unique
// subset of the vocabulary table in the item and count bounds.
func checkEnumSubset(raw json.RawMessage, vocabulary []string, minimumLength, maximumLength int, minimumCount, maximumCount uint64) bool {
	values, ok := checkSortedUniqueStrings(raw, minimumLength, maximumLength, minimumCount, maximumCount)
	if !ok {
		return false
	}
	for _, value := range values {
		allowed := false
		for _, candidate := range vocabulary {
			if value == candidate {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

// checkUUIDv7DigestSubset reports whether the member is a sorted
// unique array of UUIDv7|digest unions in the count bound.
func checkUUIDv7DigestSubset(raw json.RawMessage, minimumCount, maximumCount uint64) bool {
	elements, ok := decodeArray(raw)
	if !ok {
		return false
	}
	if uint64(len(elements)) < minimumCount || uint64(len(elements)) > maximumCount {
		return false
	}
	var previous string
	for index, element := range elements {
		identifier, ok := rawString(element)
		if !ok {
			return false
		}
		if _, err := scalar.ParseUUIDv7(identifier); err != nil {
			if _, digestErr := checkDigestString(identifier); !digestErr {
				return false
			}
		}
		if index > 0 && previous >= identifier {
			return false
		}
		previous = identifier
	}
	return true
}

// checkQueryProjection validates the fields/preset rules: the two
// are mutually exclusive; both are null for mutations and count;
// projecting reads carry a sorted unique field subset or a
// registry preset, never both.
func checkQueryProjection(name string, kind queryOperationKind, fieldsRaw, presetRaw json.RawMessage) bool {
	fieldsNull := isNull(fieldsRaw)
	presetNull := isNull(presetRaw)
	if kind == queryKindMutation || name == "count" {
		return fieldsNull && presetNull
	}
	if !fieldsNull && !presetNull {
		return false
	}
	if !fieldsNull {
		fields, ok := checkSortedUniqueStrings(fieldsRaw, 1, 64, 0, 128)
		if !ok {
			return false
		}
		for _, field := range fields {
			if !validQueryField(field) {
				return false
			}
		}
		return true
	}
	if !presetNull {
		preset, ok := rawString(presetRaw)
		if !ok || !validQueryPreset(preset) {
			return false
		}
	}
	return true
}

// checkQueryPagination validates the skip/take rules: only list
// operations may use nonzero skip or take other than 1.
func checkQueryPagination(name string, skipRaw, takeRaw json.RawMessage) bool {
	skip, ok := rawUint53(skipRaw)
	if !ok || skip > 1000000 {
		return false
	}
	take, ok := rawUint53(takeRaw)
	if !ok || take < 1 || take > 1000 {
		return false
	}
	if isListOperation(name) {
		return true
	}
	return skip == 0 && take == 1
}

// checkQuerySort validates the sort tuple array: QuerySort[0..8]
// with registry fields and directions.
func checkQuerySort(raw json.RawMessage) bool {
	elements, ok := decodeArray(raw)
	if !ok || len(elements) > 8 {
		return false
	}
	for _, element := range elements {
		members, fault := decodeStrictObject(bytesTrimSpace(element))
		if fault != nil {
			return false
		}
		if name, unknown := unknownMember(members, sortMembers); unknown {
			_ = name
			return false
		}
		if name, missing := missingMember(members, sortRequired); missing {
			_ = name
			return false
		}
		field, ok := rawString(members["field"])
		if !ok {
			return false
		}
		allowed := false
		for _, candidate := range querySortFields {
			if field == candidate {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
		direction, ok := rawString(members["direction"])
		if !ok || (direction != "asc" && direction != "desc") {
			return false
		}
		if !checkExtensions(members["extensions"]) {
			return false
		}
	}
	return true
}

// checkQueryFlags validates the read/mutation flag rules. Read
// operations require dry_run=false, confirm=false, and null
// expectation and idempotency. Mutations require dry_run=true or
// confirm=true, never both; execute_plan requires confirm,
// expectation digest, and idempotency key; annotation mutations
// require expectation and idempotency when confirmed; planning is
// pure and uses dry run with no confirmation.
func checkQueryFlags(name string, kind queryOperationKind, members map[string]json.RawMessage) bool {
	dryRun, ok := rawBool(members["dry_run"])
	if !ok {
		return false
	}
	confirm, ok := rawBool(members["confirm"])
	if !ok {
		return false
	}
	_, expectationNull := optionalDigestNull(members["expectation_digest"])
	_, idempotencyNull := optionalDigestNull(members["idempotency_key"])
	if kind == queryKindRead {
		return !dryRun && !confirm && expectationNull && idempotencyNull
	}
	if dryRun == confirm {
		return false
	}
	switch {
	case name == "execute_plan":
		return confirm && !dryRun && !expectationNull && !idempotencyNull
	case name == "plan_continue":
		return dryRun && !confirm
	case isAnnotationMutation(name):
		if confirm {
			return !expectationNull && !idempotencyNull
		}
		return true
	default:
		return true
	}
}

// optionalDigestNull reports whether the member is an explicit
// null; otherwise it must be a digest. The second result is false
// for any other shape.
func optionalDigestNull(raw json.RawMessage) (string, bool) {
	if isNull(raw) {
		return "", true
	}
	digest, ok := checkDigest(raw)
	if !ok {
		return "", false
	}
	return digest.String(), false
}

// CheckCursorReuse validates one returned cursor against the query
// bounds it was issued under. A cursor is an opaque
// string[1..1024] integrity-bound to query schema version, caller,
// disclosure policy, operation name, parameters, projection, sort
// tuple, and last stable key; reuse after any bound value changes
// is refused with query_invalid, never a best-effort continuation.
// The pinned error registry does not register the
// query_cursor_mismatch code the Section 10.8.5 prose names, so the
// refusal carries query_invalid with a cursor-mismatch detail —
// the divergence is recorded here, not hidden.
func CheckCursorReuse(cursor string, boundsChanged bool) error {
	if stringLength(cursor) < 1 || stringLength(cursor) > 1024 {
		failure, err := failQuery("directory cursor is not a string[1..1024]", "cursor")
		if err != nil {
			return err
		}
		return failure
	}
	if boundsChanged {
		failure, err := failQuery("directory cursor is bound to a changed query", "cursor")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
