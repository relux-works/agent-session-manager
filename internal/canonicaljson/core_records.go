package canonicaljson

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	leaseSchema            = "urn:ax:schema:lease"
	checkpointSchema       = "urn:ax:schema:checkpoint"
	providerIdentitySchema = "urn:ax:schema:provider-identity"
	workspaceGroupSchema   = "urn:ax:schema:workspace-group"
	sessionEventSchema     = "urn:ax:schema:session-event"
	observationSchema      = "urn:ax:schema:observation"
)

var (
	providerIdentityKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
	observationNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}$`)
	lowerSnakePattern          = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	// terminalBackendIDPattern is the Section 4.B declared grammar, verbatim:
	// "terminal_backend_id is 1-128 ASCII bytes matching
	// [a-z][a-z0-9]*(?:[.-][a-z0-9]+)*".
	terminalBackendIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	windowsAbsolutePathPattern = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

	// ErrInvalidObservation reports an Observation Event or stream outside the
	// pinned Section 18.1 contract. Observation Events are not identity-addressed.
	ErrInvalidObservation = errors.New("invalid observation event")
)

func validateLeaseRecord(object map[string]any) error {
	if err := requireExactMembers("Lease Record", object,
		"schema", "schema_version", "record_id", "subject_id", "lease_id", "session_id", "epoch",
		"holder_host_id", "predecessor_lease_id", "reason", "checkpoint_id", "issued_by_host_id",
		"created_by_host_id", "created_at", "extensions"); err != nil {
		return err
	}
	if err := validateIdentityRecordEnvelope(object, leaseSchema, "1.0.0", "record_id"); err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return err
	}
	sessionID, err := requireUUIDv7(object, "session_id")
	if err != nil {
		return err
	}
	if subjectID != sessionID {
		return invalidIdentity("Lease Record subject_id must equal session_id")
	}
	if err := requireUUIDv4(object, "lease_id"); err != nil {
		return err
	}
	epoch, err := requirePositiveUint(object, "epoch")
	if err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "holder_host_id"); err != nil {
		return err
	}
	predecessorPresent, err := requireNullableUUIDv4(object, "predecessor_lease_id")
	if err != nil {
		return err
	}
	reason, err := requireEnum(object, "reason", "create", "graceful_takeover", "force_takeover", "recovery")
	if err != nil {
		return err
	}
	checkpointPresent, err := requireNullableDigestPresence(object, "checkpoint_id")
	if err != nil {
		return err
	}
	// Each of the three couplings below quotes the clause that licenses it.
	// Section 5.3 does not declare an epoch-1 => create direction anywhere, so
	// an epoch-1 recovery or takeover lease is admitted: where the pinned
	// specification is silent this validator stays permissive.
	epochOneCreate := epoch == 1 && reason == "create"
	// predecessor_lease_id: "Null only at epoch 1".
	if epoch != 1 && !predecessorPresent {
		return invalidIdentity("Lease Record predecessor_lease_id must be non-null after epoch 1")
	}
	// "An epoch-1 create lease MUST have a null predecessor".
	if epochOneCreate && predecessorPresent {
		return invalidIdentity("Lease Record epoch-1 create predecessor_lease_id must be null")
	}
	// checkpoint_id: "Null only for epoch-1 create; otherwise the validated
	// materialized handoff base". The complement of epoch-1 create therefore
	// requires a non-null checkpoint, including at epoch 1 for any other reason.
	if !epochOneCreate && !checkpointPresent {
		return invalidIdentity("Lease Record checkpoint_id must be non-null unless the lease is an epoch-1 create")
	}
	issuedBy, err := requireUUIDv7(object, "issued_by_host_id")
	if err != nil {
		return err
	}
	createdBy, _ := object["created_by_host_id"].(string)
	if createdBy != issuedBy {
		return invalidIdentity("Lease Record created_by_host_id must equal issued_by_host_id")
	}
	return nil
}

func validateCheckpointRecord(object map[string]any) error {
	if err := requireExactMembers("Checkpoint Record", object,
		"schema", "schema_version", "checkpoint_id", "subject_id", "session_id", "lease_epoch", "lease_id",
		"safe_boundary", "event_heads", "workspace_manifest_id", "provider_manifest_id", "task_board_bundle_id",
		"created_by_host_id", "created_at", "status", "extensions"); err != nil {
		return err
	}
	if err := validateIdentityRecordEnvelope(object, checkpointSchema, "1.0.0", "checkpoint_id"); err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return err
	}
	sessionID, err := requireUUIDv7(object, "session_id")
	if err != nil {
		return err
	}
	if subjectID != sessionID {
		return invalidIdentity("Checkpoint Record subject_id must equal session_id")
	}
	if _, err := requirePositiveUint(object, "lease_epoch"); err != nil {
		return err
	}
	if err := requireUUIDv4(object, "lease_id"); err != nil {
		return err
	}
	boundary, err := requireObject(object, "safe_boundary")
	if err != nil {
		return err
	}
	if err := validateSafeBoundaryEvidence(boundary); err != nil {
		return err
	}
	heads, err := requireArrayRange(object, "event_heads", 1, 64)
	if err != nil {
		return err
	}
	if err := validateSortedUniqueDigests(heads, "event_heads"); err != nil {
		return err
	}
	if err := requireDigest(object, "workspace_manifest_id"); err != nil {
		return err
	}
	providerPresent, err := requireNullableDigestPresence(object, "provider_manifest_id")
	if err != nil {
		return err
	}
	boardPresent, err := requireNullableDigestPresence(object, "task_board_bundle_id")
	if err != nil {
		return err
	}
	if providerPresent == boardPresent {
		return invalidIdentity("Checkpoint Record requires exactly one of provider_manifest_id and task_board_bundle_id")
	}
	return requireExactString(object, "status", "validated")
}

func validateSafeBoundaryEvidence(object map[string]any) error {
	if err := requireExactMembers("Safe Boundary Evidence", object,
		"provider_id", "provider_version", "evidence", "input_blocked", "foreground_idle", "background_idle",
		"open_processes", "open_database_handles"); err != nil {
		return err
	}
	if err := requireProviderID(object, "provider_id"); err != nil {
		return err
	}
	if _, err := requireBoundedString(object, "provider_version", 1, 128); err != nil {
		return err
	}
	if _, err := requireEnum(object, "evidence", "provider_api", "provider_event", "managed_pty", "task_board_bridge", "accepted_test"); err != nil {
		return err
	}
	for _, name := range []string{"input_blocked", "foreground_idle", "background_idle"} {
		value, err := requireBool(object, name)
		if err != nil {
			return err
		}
		if !value {
			return invalidIdentity("Safe Boundary Evidence %s must be true", name)
		}
	}
	for _, name := range []string{"open_processes", "open_database_handles"} {
		value, err := requireUint(object, name, uint64(maxSafeInteger))
		if err != nil {
			return err
		}
		if value != 0 {
			return invalidIdentity("Safe Boundary Evidence %s must be zero", name)
		}
	}
	return nil
}

func validateProviderIdentityRecord(object map[string]any) error {
	if err := requireExactMembers("Provider Identity Record", object,
		"schema", "schema_version", "record_id", "subject_id", "session_id", "provider_id", "provider_version",
		"provider_version_range", "native_session_id", "identity_kind", "logical_workspace_id",
		"backend_realm_fingerprint", "opaque_identity", "created_by_host_id", "created_at", "extensions"); err != nil {
		return err
	}
	if err := validateIdentityRecordEnvelope(object, providerIdentitySchema, "1.0.0", "record_id"); err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return err
	}
	sessionID, err := requireUUIDv7(object, "session_id")
	if err != nil {
		return err
	}
	if subjectID != sessionID {
		return invalidIdentity("Provider Identity Record subject_id must equal session_id")
	}
	if err := requireProviderID(object, "provider_id"); err != nil {
		return err
	}
	// Spelled out as three ordered calls rather than a map range: a map range
	// reports a random member when several bounds are violated at once, and it
	// hides the declared bounds from the derived bounds inventory.
	if _, err := requireBoundedString(object, "provider_version", 1, 128); err != nil {
		return err
	}
	if _, err := requireBoundedString(object, "provider_version_range", 1, 256); err != nil {
		return err
	}
	if _, err := requireBoundedString(object, "native_session_id", 1, 512); err != nil {
		return err
	}
	kind, err := requireEnum(object, "identity_kind", "session_uuid", "session_path_or_id", "backend_conversation_uuid", "task_board_managed", "provider_defined")
	if err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "logical_workspace_id"); err != nil {
		return err
	}
	realmPresent, err := requireNullableDigestPresence(object, "backend_realm_fingerprint")
	if err != nil {
		return err
	}
	providerID, _ := object["provider_id"].(string)
	if providerID == "antigravity" && kind == "backend_conversation_uuid" && !realmPresent {
		return invalidIdentity("Provider Identity Record backend_realm_fingerprint must be non-null for Antigravity backend_conversation_uuid")
	}
	opaque, err := requireObject(object, "opaque_identity")
	if err != nil {
		return err
	}
	if len(opaque) > 32 {
		return invalidIdentity("Provider Identity Record opaque_identity exceeds maximum length 32")
	}
	// Iterate in sorted key order so a record violating several opaque_identity
	// bounds always reports the same member rather than a random map key.
	opaqueKeys := make([]string, 0, len(opaque))
	for key := range opaque {
		opaqueKeys = append(opaqueKeys, key)
	}
	sort.Strings(opaqueKeys)
	for _, key := range opaqueKeys {
		value := opaque[key]
		if !providerIdentityKeyPattern.MatchString(key) {
			return invalidIdentity("Provider Identity Record opaque_identity key %q must match [a-z][a-z0-9_.-]{0,63}", key)
		}
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) {
			return invalidIdentity("Provider Identity Record opaque_identity[%q] must be a UTF-8 string", key)
		}
		length := utf8.RuneCountInString(text)
		if length < 1 || length > 1024 {
			return invalidIdentity("Provider Identity Record opaque_identity[%q] must contain 1..1024 Unicode characters", key)
		}
		// Section 2 declares the map "MUST NOT contain an absolute path,
		// credential, environment value, PID, socket, terminal ID, or mutable
		// cache selector". Only the leading-absolute-path form of that sentence
		// is decidable from one candidate: a value such as `cwd=/Users/x`
		// embeds an absolute path and is ADMITTED here. The message says
		// "begin with" rather than "contain" so the refusal states what it
		// enforces, and constraint-enumeration.md routes the undecidable
		// remainder to the same external column as the credential and PID
		// classes in that sentence.
		if strings.HasPrefix(text, "/") || strings.HasPrefix(text, `\\`) || windowsAbsolutePathPattern.MatchString(text) {
			return invalidIdentity("Provider Identity Record opaque_identity[%q] must not begin with an absolute path", key)
		}
	}
	return nil
}

func validateWorkspaceGroupRecord(object map[string]any) error {
	if err := requireExactMembers("Workspace Group Record", object,
		"schema", "schema_version", "record_id", "subject_id", "workspace_group_id", "display_name", "members",
		"created_by_host_id", "created_at", "extensions"); err != nil {
		return err
	}
	if err := validateIdentityRecordEnvelope(object, workspaceGroupSchema, "1.0.0", "record_id"); err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return err
	}
	groupID, err := requireUUIDv7(object, "workspace_group_id")
	if err != nil {
		return err
	}
	if subjectID != groupID {
		return invalidIdentity("Workspace Group Record subject_id must equal workspace_group_id")
	}
	if _, err := requireBoundedString(object, "display_name", 1, 128); err != nil {
		return err
	}
	members, err := requireArrayRange(object, "members", 1, 256)
	if err != nil {
		return err
	}
	previousID := ""
	paths := make(map[string]string, len(members))
	for index, value := range members {
		member, err := requireObjectValue(value, fmt.Sprintf("Workspace Group Record members[%d]", index))
		if err != nil {
			return err
		}
		workspaceID, relativePath, err := validateWorkspaceMember(member)
		if err != nil {
			return err
		}
		// Section 2 declares "Members are sorted by workspace_id, and no two
		// members may have an equal or case-colliding group_relative_path". The
		// declared uniqueness is on group_relative_path, enforced below; the
		// workspace_id rule is bare `sorted`, so an equal workspace_id is
		// admitted rather than refused on an undeclared uniqueness.
		if index > 0 && workspaceID < previousID {
			return invalidIdentity("Workspace Group Record members must be sorted by workspace_id")
		}
		previousID = workspaceID
		folded := simpleFoldKey(relativePath)
		if prior, collision := paths[folded]; collision {
			return invalidIdentity("Workspace Group Record member paths %q and %q are equal or case-colliding", prior, relativePath)
		}
		paths[folded] = relativePath
	}
	return nil
}

func validateWorkspaceMember(object map[string]any) (string, string, error) {
	kind, err := requireString(object, "kind")
	if err != nil {
		return "", "", err
	}
	// Both member lists are spelled out literally rather than appended to a
	// shared prefix so the per-member constraint inventory can read them.
	switch kind {
	case "git":
		if err := requireExactMembers("WorkspaceMember.git", object,
			"workspace_id", "kind", "group_relative_path", "repo_relative_cwd", "agent_project_config_paths",
			"materialization_policy", "repository_identity", "sanitized_remote_urls"); err != nil {
			return "", "", err
		}
	case "managed_tree":
		if err := requireExactMembers("WorkspaceMember.managed_tree", object,
			"workspace_id", "kind", "group_relative_path", "repo_relative_cwd", "agent_project_config_paths",
			"materialization_policy", "tree_identity"); err != nil {
			return "", "", err
		}
	default:
		return "", "", invalidIdentity("WorkspaceMember kind %q is not git or managed_tree", kind)
	}
	workspaceID, err := requireUUIDv7(object, "workspace_id")
	if err != nil {
		return "", "", err
	}
	relativePath, err := requireRelativePathValue(object, "group_relative_path")
	if err != nil {
		return "", "", err
	}
	if err := requireDotOrRelativePath(object, "repo_relative_cwd"); err != nil {
		return "", "", err
	}
	if err := requireSortedUniquePaths(object, "agent_project_config_paths", 256); err != nil {
		return "", "", err
	}
	if kind == "git" {
		if err := requireBoundedLogicalIdentity(object, "repository_identity"); err != nil {
			return "", "", err
		}
		urls, err := requireArrayRange(object, "sanitized_remote_urls", 1, 16)
		if err != nil {
			return "", "", err
		}
		if err := validateSortedUniqueSanitizedGitURLs(urls); err != nil {
			return "", "", err
		}
		if _, err := requireEnum(object, "materialization_policy", "shared_checkout", "separate_worktree"); err != nil {
			return "", "", err
		}
	} else {
		if err := requireBoundedLogicalIdentity(object, "tree_identity"); err != nil {
			return "", "", err
		}
		if _, err := requireEnum(object, "materialization_policy", "shared_tree", "separate_copy"); err != nil {
			return "", "", err
		}
	}
	return workspaceID, relativePath, nil
}

func validateSortedUniqueSanitizedGitURLs(values []any) error {
	previous := ""
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			return invalidIdentity("member sanitized_remote_urls[%d] must be a string", index)
		}
		if _, err := scalar.ParseSanitizedGitURL(text); err != nil {
			return invalidIdentity("member sanitized_remote_urls[%d]: %v", index, err)
		}
		if index > 0 && text <= previous {
			return invalidIdentity("member sanitized_remote_urls must be strictly sorted and unique")
		}
		previous = text
	}
	return nil
}

type eventPayloadShape struct {
	members []string
	check   func(map[string]any) error
}

var sessionEventPayloadShapes = mustBuildSessionEventPayloadShapes()

func validateSessionEventV1(object map[string]any) error {
	return validateSessionEvent(object, "1.0.0")
}
func validateSessionEventV2(object map[string]any) error {
	return validateSessionEvent(object, "2.0.0")
}
func validateSessionEventV3(object map[string]any) error {
	return validateSessionEvent(object, "3.0.0")
}
func validateSessionEventV4(object map[string]any) error {
	return validateSessionEvent(object, "4.0.0")
}

func validateSessionEvent(object map[string]any, version string) error {
	if err := requireExactMembers("Session Event", object,
		"schema", "schema_version", "event_id", "subject_id", "session_id", "event_type", "created_by_host_id",
		"lease_epoch", "lease_id", "lease_sequence", "predecessors", "created_at", "payload", "extensions"); err != nil {
		return err
	}
	if err := validateIdentityRecordEnvelope(object, sessionEventSchema, version, "event_id"); err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return err
	}
	sessionID, err := requireUUIDv7(object, "session_id")
	if err != nil {
		return err
	}
	if subjectID != sessionID {
		return invalidIdentity("Session Event subject_id must equal session_id")
	}
	eventType, err := requireString(object, "event_type")
	if err != nil {
		return err
	}
	if _, err := requirePositiveUint(object, "lease_epoch"); err != nil {
		return err
	}
	if err := requireUUIDv4(object, "lease_id"); err != nil {
		return err
	}
	if _, err := requirePositiveUint(object, "lease_sequence"); err != nil {
		return err
	}
	predecessors, err := requireArrayMinimum(object, "predecessors", 1)
	if err != nil {
		return err
	}
	// Section 5.2 declares predecessors as "a sorted array of one or more
	// record/event digests" - bare `sorted`, not the Section 1.6 compound phrase
	// `sorted unique`. Duplicates are therefore admitted here.
	if err := validateSortedDigests(predecessors, "predecessors"); err != nil {
		return err
	}
	payload, err := requireObject(object, "payload")
	if err != nil {
		return err
	}
	shape, known := sessionEventPayloadShapes[version][eventType]
	if !known {
		if version == "1.0.0" {
			// Section 5.2 requires an unknown v1 event to remain retainable but
			// inert. Only its closed top-level envelope and object payload are read.
			return nil
		}
		return invalidIdentity("Session Event %s event_type %q is not registered", version, eventType)
	}
	if err := requireExactMembers("Session Event "+version+" payload "+eventType, payload, shape.members...); err != nil {
		return err
	}
	if shape.check != nil {
		return shape.check(payload)
	}
	return nil
}

func mustBuildSessionEventPayloadShapes() map[string]map[string]eventPayloadShape {
	base := map[string]eventPayloadShape{
		"session.created":           shape([]string{"session_record_id", "bootstrap_operation_id", "first_checkpoint_operation_id"}, checks(digests("session_record_id"), uuid7s("bootstrap_operation_id", "first_checkpoint_operation_id"))),
		"terminal.created":          shape([]string{"backend", "terminal_id"}, checks(enum("backend", "tmux", "conpty"), bounded("terminal_id", 1, 512))),
		"provider.launched":         shape([]string{"provider_id", "provider_version", "execution_profile", "profile_source_event_id", "profile_mapping"}, checks(providerIDs("provider_id"), bounded("provider_version", 1, 128), enum("execution_profile", "standard", "yolo"), nullableDigests("profile_source_event_id"), bounded("profile_mapping", 1, 512))),
		"provider.identified":       shape([]string{"provider_identity_record_id", "confidence"}, checks(digests("provider_identity_record_id"), enum("confidence", "exact", "strong", "weak"))),
		"session.idle":              shape([]string{"boundary_ref", "foreground_idle", "background_idle"}, checks(bounded("boundary_ref", 1, 1024), booleans("foreground_idle", "background_idle"))),
		"session.quiescing":         shape([]string{"operation_id", "reason", "input_blocked"}, checks(uuid7s("operation_id"), enum("reason", "graceful_takeover", "stop", "checkpoint"), booleans("input_blocked"))),
		"checkpoint.created":        shape([]string{"checkpoint_id", "kind"}, checks(digests("checkpoint_id"), enum("kind", "periodic", "pre_stop", "closure", "fork_base", "manual"))),
		"sync.completed":            shape([]string{"peer_host_id", "checkpoint_id", "manifest_ids", "materialized"}, validateSyncCompletedPayload),
		"session.stopped":           shape([]string{"graceful", "checkpoint_id", "resumable", "closure_kind", "process_closed", "store_closed"}, validateSessionStoppedPayload),
		"session.resumed":           shape([]string{"checkpoint_id", "execution_profile", "profile_source_event_id", "terminal_backend", "native_session_id"}, checks(digests("checkpoint_id"), enum("execution_profile", "standard", "yolo"), nullableDigests("profile_source_event_id"), enum("terminal_backend", "tmux", "conpty"), bounded("native_session_id", 1, 512))),
		"session.bootstrap_aborted": shape([]string{"operation_id", "failure_phase", "provider_identity_record_id", "manager_session_ref", "process_closed", "store_closed", "resume_allowed"}, validateBootstrapAbortedPayload),
		"lease.transferred":         shape([]string{"operation_id", "from_host_id", "to_host_id", "predecessor_lease_id", "new_lease_id"}, checks(uuid7s("operation_id", "from_host_id", "to_host_id"), uuid4s("predecessor_lease_id", "new_lease_id"))),
		"lease.forced":              shape([]string{"operation_id", "expected_owner_host_id", "expected_epoch", "new_lease_id", "checkpoint_id"}, checks(uuid7s("operation_id", "expected_owner_host_id"), positiveUints("expected_epoch"), uuid4s("new_lease_id"), digests("checkpoint_id"))),
		"session.parked":            shape([]string{"reason", "winning_lease_id"}, checks(enum("reason", "remote_owner", "stale_owner", "restore_policy", "failed_handoff"), uuid4s("winning_lease_id"))),
		"session.failed":            shape([]string{"error_code", "retryable", "operation_id"}, checks(bounded("error_code", 1, 128), booleans("retryable"), nullableUUIDv7s("operation_id"))),
		"fork.created":              shape([]string{"source_session_id", "source_checkpoint_id", "new_session_record_id", "provider_fork_mode", "execution_profile", "profile_source_event_id", "source_profile_event_id"}, checks(uuid7s("source_session_id"), digests("source_checkpoint_id", "new_session_record_id"), enum("provider_fork_mode", "native", "supported_import", "task_board_clone"), enum("execution_profile", "standard", "yolo"), nullMember("profile_source_event_id"), nullableDigests("source_profile_event_id"))),
		"profile.changed":           shape([]string{"from", "to", "confirmed"}, validateProfileChangedPayload),
		"session.tombstoned":        shape([]string{"tombstone_id"}, digests("tombstone_id")),
		"takeover.force_confirmed":  shape([]string{"operation_id", "expected_owner_host_id", "expected_epoch", "checkpoint_id", "accepted_risks", "confirmation_mode"}, validateForceConfirmedPayload),
		"replica.replace_confirmed": shape([]string{"operation_id", "workspace_group_id", "target_host_id", "managed_replica_id", "expected_marker_id", "expected_checkpoint_id", "replacement_checkpoint_id", "confirmation_mode"}, checks(uuid7s("operation_id", "workspace_group_id", "target_host_id", "managed_replica_id"), digests("expected_marker_id", "expected_checkpoint_id", "replacement_checkpoint_id"), enum("confirmation_mode", "interactive", "non_interactive"))),
		"task_board.launched":       shape([]string{"operation_id", "manager_session_ref", "provider_id", "launch_mode", "lease_epoch", "lease_id", "execution_profile", "profile_source_event_id", "board_goal_id", "board_goal_revision", "state"}, validateTaskBoardLaunchedPayload),
		"task_board.adopted":        shape([]string{"operation_id", "bundle_id", "manager_session_ref", "board_goal_id", "board_goal_revision"}, validateTaskBoardAdoptedPayload),
		"tombstone.issued":          shape([]string{"tombstone_id", "scope", "subject_id", "target_ref"}, checks(digests("tombstone_id"), enum("scope", "session", "workspace_entry", "provider_snapshot", "managed_replica"), uuid7s("subject_id"), bounded("target_ref", 1, 1024))),
		"tombstone.resolved":        shape([]string{"tombstone_id", "resolution", "target_ref", "resulting_entry_digest"}, validateTombstoneResolvedPayload),
	}
	clone := map[string]eventPayloadShape{
		"clone.planned":                  shape([]string{"operation_id", "bundle_manifest_id", "projection_plan_id", "migration_checkpoint_id", "materialization_id", "target_environment", "expected_target_native_session_id"}, validateClonePlannedPayload),
		"clone.target_prepared":          shape([]string{"operation_id", "materialization_id", "plan_id", "provider_transaction_id", "provider_prepared_result_digest", "staged_read_back_evidence_manifest_id", "rollback_retained"}, checks(uuid7s("operation_id", "materialization_id", "provider_transaction_id"), digests("plan_id", "provider_prepared_result_digest", "staged_read_back_evidence_manifest_id"), literalBool("rollback_retained", true))),
		"clone.target_published":         shape([]string{"operation_id", "materialization_id", "provider_identity_record_id", "target_provider_manifest_id", "live_read_back_evidence_manifest_id", "fidelity_report_id", "validation_report_id", "source_generation_revalidated", "rollback_retained"}, checks(uuid7s("operation_id", "materialization_id"), digests("provider_identity_record_id", "target_provider_manifest_id", "live_read_back_evidence_manifest_id", "fidelity_report_id", "validation_report_id"), literalBool("source_generation_revalidated", true), literalBool("rollback_retained", true))),
		"clone.target_validation_failed": shape([]string{"operation_id", "materialization_id", "phase", "error_code", "validation_report_id", "rollback_required", "transaction_unknown"}, checks(uuid7s("operation_id", "materialization_id"), enum("phase", "prepublication_source_recheck", "provider_prepare", "postpublication_source_recheck", "live_discovery", "live_read_back", "resume_plan"), bounded("error_code", 1, 128), nullableDigests("validation_report_id"), booleans("rollback_required", "transaction_unknown"))),
		"clone.rolled_back":              shape([]string{"operation_id", "materialization_id", "provider_rolled_back_result_digest", "retained_bundle_manifest_id", "reason_code"}, checks(uuid7s("operation_id", "materialization_id"), digests("provider_rolled_back_result_digest", "retained_bundle_manifest_id"), bounded("reason_code", 1, 128))),
		"clone.committed":                shape([]string{"operation_id", "materialization_id", "provider_identity_record_id", "provider_committed_result_digest", "target_checkpoint_id", "fidelity_report_id", "validation_report_id", "native_resumable"}, checks(uuid7s("operation_id", "materialization_id"), digests("provider_identity_record_id", "provider_committed_result_digest", "target_checkpoint_id", "fidelity_report_id", "validation_report_id"), literalBool("native_resumable", true))),
		"clone.lineage_published":        shape([]string{"operation_id", "target_checkpoint_id", "lineage_receipt_id", "bundle_manifest_id"}, checks(uuid7s("operation_id"), digests("target_checkpoint_id", "lineage_receipt_id", "bundle_manifest_id"))),
		"clone.failed":                   shape([]string{"operation_id", "phase", "error_code", "retryable", "retained_bundle_manifest_id", "materialization_id", "transaction_unknown"}, checks(uuid7s("operation_id", "materialization_id"), exact("phase", "checkpoint"), exact("error_code", "target_checkpoint_failed"), literalBool("retryable", true), digests("retained_bundle_manifest_id"), literalBool("transaction_unknown", false))),
	}
	directory := map[string]eventPayloadShape{
		"adoption.planned":              shape([]string{"operation_id", "plan_id", "source_instance_id", "source_observation_id", "source_head_digest"}, checks(uuid7s("operation_id"), digests("plan_id", "source_instance_id", "source_observation_id", "source_head_digest"))),
		"adoption.committed":            shape([]string{"operation_id", "provider_identity_record_id", "initial_checkpoint_id", "native_resumable"}, checks(uuid7s("operation_id"), digests("provider_identity_record_id", "initial_checkpoint_id"), literalBool("native_resumable", true))),
		"move.planned":                  shape([]string{"operation_id", "plan_id", "source_session_id", "target_session_id"}, checks(uuid7s("operation_id", "source_session_id", "target_session_id"), digests("plan_id"))),
		"move.target_committed":         shape([]string{"operation_id", "target_session_id", "target_checkpoint_id", "clone_lineage_receipt_id"}, checks(uuid7s("operation_id", "target_session_id"), digests("target_checkpoint_id", "clone_lineage_receipt_id"))),
		"move.source_release_requested": shape([]string{"operation_id", "target_committed_event_id", "source_lease_epoch", "source_lease_id"}, checks(uuid7s("operation_id"), digests("target_committed_event_id"), positiveUints("source_lease_epoch"), uuid4s("source_lease_id"))),
		"move.source_released":          shape([]string{"operation_id", "target_session_id", "source_stop_event_id", "source_release_receipt_id", "outcome"}, checks(uuid7s("operation_id", "target_session_id"), digests("source_stop_event_id", "source_release_receipt_id"), exact("outcome", "moved_cross_environment"))),
		"move.source_release_failed":    shape([]string{"operation_id", "target_session_id", "error_code", "source_still_resumable", "outcome"}, checks(uuid7s("operation_id", "target_session_id"), bounded("error_code", 1, 128), booleans("source_still_resumable"), exact("outcome", "cloned_source_still_active"))),
	}
	versions := map[string]map[string]eventPayloadShape{
		"1.0.0": clonePayloadShapes(base),
		"2.0.0": mergePayloadShapes(base, clone),
		"3.0.0": mergePayloadShapes(base, clone, directory),
		"4.0.0": mergePayloadShapes(base, clone, directory),
	}
	versions["4.0.0"]["terminal.created"] = shape([]string{"terminal_binding_id", "terminal_backend_id", "implementation_version", "protocol_version", "evidence_ids"}, validateTerminalV4Payload)
	versions["4.0.0"]["session.resumed"] = shape([]string{"checkpoint_id", "execution_profile", "profile_source_event_id", "terminal_binding_id", "terminal_backend_id", "implementation_version", "protocol_version", "evidence_ids"}, checks(digests("checkpoint_id"), enum("execution_profile", "standard", "yolo"), nullableDigests("profile_source_event_id"), validateTerminalV4Payload))
	if err := validateSessionEventPayloadShapeCompleteness(versions, catalog.Current().Events); err != nil {
		panic(err)
	}
	return versions
}

func validateSessionEventPayloadShapeCompleteness(versions map[string]map[string]eventPayloadShape, events []catalog.Event) error {
	expected := make(map[string]map[string]struct{})
	for _, event := range events {
		if event.Family != "session_event" {
			continue
		}
		for _, version := range event.ContractVersions {
			if expected[version] == nil {
				expected[version] = make(map[string]struct{})
			}
			expected[version][string(event.Name)] = struct{}{}
		}
	}
	if len(versions) != len(expected) {
		return fmt.Errorf("Session Event payload registry has %d versions, catalog has %d", len(versions), len(expected))
	}
	for version, names := range expected {
		shapes, ok := versions[version]
		if !ok || len(shapes) != len(names) {
			return fmt.Errorf("Session Event %s payload registry has %d entries, catalog has %d", version, len(shapes), len(names))
		}
		for name := range names {
			shape, ok := shapes[name]
			if !ok || len(shape.members) == 0 {
				return fmt.Errorf("Session Event %s payload registry is missing %s", version, name)
			}
		}
	}
	return nil
}

func shape(members []string, check func(map[string]any) error) eventPayloadShape {
	return eventPayloadShape{members: append([]string(nil), members...), check: check}
}

func clonePayloadShapes(source map[string]eventPayloadShape) map[string]eventPayloadShape {
	result := make(map[string]eventPayloadShape, len(source))
	for name, value := range source {
		result[name] = value
	}
	return result
}

func mergePayloadShapes(sources ...map[string]eventPayloadShape) map[string]eventPayloadShape {
	result := make(map[string]eventPayloadShape)
	for _, source := range sources {
		for name, value := range source {
			if _, duplicate := result[name]; duplicate {
				panic("duplicate Session Event payload shape " + name)
			}
			result[name] = value
		}
	}
	return result
}

func checks(validators ...func(map[string]any) error) func(map[string]any) error {
	return func(object map[string]any) error {
		for _, validator := range validators {
			if err := validator(object); err != nil {
				return err
			}
		}
		return nil
	}
}

func digests(names ...string) func(map[string]any) error { return each(names, requireDigest) }
func nullableDigests(names ...string) func(map[string]any) error {
	return each(names, requireNullableDigest)
}
func uuid7s(names ...string) func(map[string]any) error {
	return each(names, func(o map[string]any, n string) error { _, err := requireUUIDv7(o, n); return err })
}
func nullableUUIDv7s(names ...string) func(map[string]any) error {
	return each(names, func(o map[string]any, n string) error { _, _, err := requireNullableUUIDv7(o, n); return err })
}
func uuid4s(names ...string) func(map[string]any) error      { return each(names, requireUUIDv4) }
func providerIDs(names ...string) func(map[string]any) error { return each(names, requireProviderID) }
func positiveUints(names ...string) func(map[string]any) error {
	return each(names, func(o map[string]any, n string) error { _, err := requirePositiveUint(o, n); return err })
}
func booleans(names ...string) func(map[string]any) error {
	return each(names, func(o map[string]any, n string) error { _, err := requireBool(o, n); return err })
}

func each(names []string, validator func(map[string]any, string) error) func(map[string]any) error {
	return func(object map[string]any) error {
		for _, name := range names {
			if err := validator(object, name); err != nil {
				return err
			}
		}
		return nil
	}
}

func bounded(name string, minimum, maximum int) func(map[string]any) error {
	return func(object map[string]any) error {
		_, err := requireBoundedString(object, name, minimum, maximum)
		return err
	}
}
func enum(name string, allowed ...string) func(map[string]any) error {
	return func(object map[string]any) error { _, err := requireEnum(object, name, allowed...); return err }
}
func exact(name, expected string) func(map[string]any) error {
	return func(object map[string]any) error { return requireExactString(object, name, expected) }
}
func literalBool(name string, expected bool) func(map[string]any) error {
	return func(object map[string]any) error {
		value, err := requireBool(object, name)
		if err != nil {
			return err
		}
		if value != expected {
			return invalidIdentity("member %s must be %t", name, expected)
		}
		return nil
	}
}

func nullMember(name string) func(map[string]any) error {
	return func(object map[string]any) error {
		value, ok := object[name]
		if !ok {
			return invalidIdentity("identity input requires member %s", name)
		}
		if value != nil {
			return invalidIdentity("member %s must be null", name)
		}
		return nil
	}
}

func validateSyncCompletedPayload(object map[string]any) error {
	if err := uuid7s("peer_host_id")(object); err != nil {
		return err
	}
	if err := digests("checkpoint_id")(object); err != nil {
		return err
	}
	values, err := requireArrayRange(object, "manifest_ids", 1, 1024)
	if err != nil {
		return err
	}
	if err := validateSortedUniqueDigests(values, "manifest_ids"); err != nil {
		return err
	}
	return booleans("materialized")(object)
}

func validateSessionStoppedPayload(object map[string]any) error {
	if err := booleans("graceful", "resumable", "process_closed", "store_closed")(object); err != nil {
		return err
	}
	present, err := requireNullableDigestPresence(object, "checkpoint_id")
	if err != nil {
		return err
	}
	kind, err := requireEnum(object, "closure_kind", "checkpointed", "bootstrap_abort")
	if err != nil {
		return err
	}
	graceful, _ := object["graceful"].(bool)
	resumable, _ := object["resumable"].(bool)
	if kind == "checkpointed" {
		if !present || !resumable {
			return invalidIdentity("session.stopped checkpointed requires non-null checkpoint_id and resumable true")
		}
	} else if present || resumable || graceful {
		return invalidIdentity("session.stopped bootstrap_abort requires null checkpoint_id and false resumable and graceful")
	}
	return nil
}

func validateBootstrapAbortedPayload(object map[string]any) error {
	if err := uuid7s("operation_id")(object); err != nil {
		return err
	}
	phase, err := requireEnum(object, "failure_phase", "before_terminal", "after_terminal", "after_process", "after_identity", "before_checkpoint")
	if err != nil {
		return err
	}
	providerPresent, err := requireNullableDigestPresence(object, "provider_identity_record_id")
	if err != nil {
		return err
	}
	_, managerPresent, err := nullableBoundedString(object, "manager_session_ref", 1, 512)
	if err != nil {
		return err
	}
	if err := literalBool("process_closed", true)(object); err != nil {
		return err
	}
	if err := literalBool("store_closed", true)(object); err != nil {
		return err
	}
	if err := literalBool("resume_allowed", false)(object); err != nil {
		return err
	}
	identityEstablished := phase == "after_identity" || phase == "before_checkpoint"
	if identityEstablished && providerPresent == managerPresent {
		return invalidIdentity("session.bootstrap_aborted after identity requires exactly one identity field")
	}
	if !identityEstablished && (providerPresent || managerPresent) {
		return invalidIdentity("session.bootstrap_aborted before identity requires both identity fields null")
	}
	return nil
}

func validateProfileChangedPayload(object map[string]any) error {
	from, err := requireEnum(object, "from", "standard", "yolo")
	if err != nil {
		return err
	}
	to, err := requireEnum(object, "to", "standard", "yolo")
	if err != nil {
		return err
	}
	if from == to {
		return invalidIdentity("profile.changed from and to must differ")
	}
	return booleans("confirmed")(object)
}

func validateForceConfirmedPayload(object map[string]any) error {
	if err := uuid7s("operation_id", "expected_owner_host_id")(object); err != nil {
		return err
	}
	if err := positiveUints("expected_epoch")(object); err != nil {
		return err
	}
	if err := digests("checkpoint_id")(object); err != nil {
		return err
	}
	risks, err := requireArrayRange(object, "accepted_risks", 3, 3)
	if err != nil {
		return err
	}
	want := []string{"divergent_history", "split_brain", "stale_process"}
	for index, value := range risks {
		if value != want[index] {
			return invalidIdentity("takeover.force_confirmed accepted_risks must be exactly %v", want)
		}
	}
	return enum("confirmation_mode", "interactive", "non_interactive")(object)
}

func validateTaskBoardLaunchedPayload(object map[string]any) error {
	if err := uuid7s("operation_id")(object); err != nil {
		return err
	}
	if err := bounded("manager_session_ref", 1, 512)(object); err != nil {
		return err
	}
	if err := providerIDs("provider_id")(object); err != nil {
		return err
	}
	if err := enum("launch_mode", "primary_owner", "tracked_prompt")(object); err != nil {
		return err
	}
	if err := positiveUints("lease_epoch")(object); err != nil {
		return err
	}
	if err := uuid4s("lease_id")(object); err != nil {
		return err
	}
	if err := enum("execution_profile", "standard", "yolo")(object); err != nil {
		return err
	}
	if err := nullableDigests("profile_source_event_id")(object); err != nil {
		return err
	}
	if err := validateGoalPair(object); err != nil {
		return err
	}
	return enum("state", "running", "idle")(object)
}

func validateTaskBoardAdoptedPayload(object map[string]any) error {
	if err := uuid7s("operation_id")(object); err != nil {
		return err
	}
	if err := digests("bundle_id")(object); err != nil {
		return err
	}
	if err := bounded("manager_session_ref", 1, 512)(object); err != nil {
		return err
	}
	return validateGoalPair(object)
}

func validateGoalPair(object map[string]any) error {
	_, goalPresent, err := nullableBoundedString(object, "board_goal_id", 1, 128)
	if err != nil {
		return err
	}
	revisionPresent, err := nullablePositiveUint(object, "board_goal_revision")
	if err != nil {
		return err
	}
	if goalPresent != revisionPresent {
		return invalidIdentity("task-board goal ID and revision must both be null or both non-null")
	}
	return nil
}

func validateTombstoneResolvedPayload(object map[string]any) error {
	if err := digests("tombstone_id")(object); err != nil {
		return err
	}
	resolution, err := requireEnum(object, "resolution", "deleted", "already_absent", "resurrected", "retained_conflict")
	if err != nil {
		return err
	}
	if err := bounded("target_ref", 1, 1024)(object); err != nil {
		return err
	}
	present, err := requireNullableDigestPresence(object, "resulting_entry_digest")
	if err != nil {
		return err
	}
	if (resolution == "resurrected") != present {
		return invalidIdentity("tombstone.resolved resulting_entry_digest must be non-null exactly for resurrected")
	}
	return nil
}

func validateClonePlannedPayload(object map[string]any) error {
	if err := uuid7s("operation_id", "materialization_id")(object); err != nil {
		return err
	}
	if err := digests("bundle_manifest_id", "projection_plan_id", "migration_checkpoint_id")(object); err != nil {
		return err
	}
	environment, err := requireObject(object, "target_environment")
	if err != nil {
		return err
	}
	if err := validateEnvironmentTuple(environment); err != nil {
		return err
	}
	return bounded("expected_target_native_session_id", 1, 512)(object)
}

func validateTerminalV4Payload(object map[string]any) error {
	if err := digests("terminal_binding_id")(object); err != nil {
		return err
	}
	if err := requireTerminalBackendID(object, "terminal_backend_id"); err != nil {
		return err
	}
	for _, name := range []string{"implementation_version", "protocol_version"} {
		version, err := requireString(object, name)
		if err != nil {
			return err
		}
		if !semverPattern.MatchString(version) {
			return invalidIdentity("member %s must be canonical semver", name)
		}
	}
	values, err := requireArrayRange(object, "evidence_ids", 1, 256)
	if err != nil {
		return err
	}
	return validateSortedUniqueDigests(values, "evidence_ids")
}

// ValidateObservationEvent validates one complete Section 18.1 JSON object.
// It is read-only and deliberately separate from immutable object identity.
func ValidateObservationEvent(input []byte) error {
	value, err := decodeStrict(input)
	if err != nil {
		return invalidObservation("decode: %v", err)
	}
	if err := validateAXNumbers(value); err != nil {
		return invalidObservation("numbers: %v", err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return invalidObservation("input must be a JSON object")
	}
	if err := validateObservationEvent(object); err != nil {
		return invalidObservation("%v", err)
	}
	return nil
}

// ValidateObservationStream validates one ordered stream snapshot. Each event
// still passes ValidateObservationEvent; stream identity and exact +1 sequence
// continuity are then checked without consulting timestamps.
func ValidateObservationStream(events [][]byte) error {
	if len(events) == 0 {
		return invalidObservation("stream must contain at least one event")
	}
	streamID := ""
	var previous uint64
	for index, input := range events {
		value, err := decodeStrict(input)
		if err != nil {
			return invalidObservation("event %d decode: %v", index, err)
		}
		if err := validateAXNumbers(value); err != nil {
			return invalidObservation("event %d numbers: %v", index, err)
		}
		object, ok := value.(map[string]any)
		if !ok {
			return invalidObservation("event %d must be a JSON object", index)
		}
		if err := validateObservationEvent(object); err != nil {
			return invalidObservation("event %d: %v", index, err)
		}
		currentStream, _ := object["stream_id"].(string)
		sequence, _ := requireUint(object, "sequence", uint64(maxSafeInteger))
		if index == 0 {
			streamID = currentStream
			if sequence != 1 {
				return invalidObservation("stream sequence must start at 1, got %d", sequence)
			}
		} else {
			if currentStream != streamID {
				return invalidObservation("stream_id changed within stream")
			}
			if sequence != previous+1 {
				return invalidObservation("stream sequence must increase by exactly one: got %d after %d", sequence, previous)
			}
		}
		previous = sequence
	}
	return nil
}

func validateObservationEvent(object map[string]any) error {
	if err := requireExactMembers("Observation Event", object,
		"schema", "schema_version", "stream_id", "sequence", "timestamp", "level", "event", "operation_id",
		"session_id", "host_id", "peer_host_id", "phase", "result", "duration_ms", "counts", "object_ids",
		"error_code", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "schema", observationSchema); err != nil {
		return err
	}
	if err := requireExactString(object, "schema_version", "1.0.0"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "stream_id"); err != nil {
		return err
	}
	if _, err := requirePositiveUint(object, "sequence"); err != nil {
		return err
	}
	if err := requireTimestamp(object, "timestamp"); err != nil {
		return err
	}
	if _, err := requireEnum(object, "level", "debug", "info", "warn", "error"); err != nil {
		return err
	}
	eventName, err := requireBoundedString(object, "event", 3, 128)
	if err != nil {
		return err
	}
	if !observationNamePattern.MatchString(eventName) {
		return invalidIdentity("Observation Event event must match [a-z][a-z0-9_]*(.[a-z][a-z0-9_]*){1,7}")
	}
	if err := nullableUUIDv7s("operation_id", "session_id", "peer_host_id")(object); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "host_id"); err != nil {
		return err
	}
	phase, phasePresent, err := nullableBoundedString(object, "phase", 1, 128)
	if err != nil {
		return err
	}
	if phasePresent && !lowerSnakePattern.MatchString(phase) {
		return invalidIdentity("Observation Event phase must use lower_snake_case")
	}
	result, err := requireEnum(object, "result", "started", "success", "partial", "failure", "cancelled")
	if err != nil {
		return err
	}
	_, durationPresent, err := nullableUintPresence(object, "duration_ms")
	if err != nil {
		return err
	}
	if result == "started" && durationPresent {
		return invalidIdentity("Observation Event started result requires null duration_ms")
	}
	if err := validateObservationCountsMember(object); err != nil {
		return err
	}
	objectIDs, err := requireArrayRange(object, "object_ids", 0, 4096)
	if err != nil {
		return err
	}
	if err := validateSortedUniqueDigests(objectIDs, "object_ids"); err != nil {
		return err
	}
	_, errorPresent, err := nullableBoundedString(object, "error_code", 1, 128)
	if err != nil {
		return err
	}
	requiresError := result == "partial" || result == "failure"
	if requiresError != errorPresent {
		return invalidIdentity("Observation Event partial/failure requires non-null error_code and every other result requires null")
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateObservationCountsMember(parent map[string]any) error {
	value, ok := parent["counts"]
	if !ok {
		return invalidIdentity("identity input requires member counts")
	}
	if value == nil {
		return nil
	}
	object, err := requireObjectValue(value, "ObservationCounts")
	if err != nil {
		return err
	}
	if err := requireExactMembers("ObservationCounts", object, "records", "events", "manifests", "blobs", "chunks", "bytes", "retries"); err != nil {
		return err
	}
	for _, name := range []string{"records", "events", "manifests", "blobs", "chunks", "bytes", "retries"} {
		if _, err := requireUint(object, name, uint64(maxSafeInteger)); err != nil {
			return err
		}
	}
	return nil
}

func validateIdentityRecordEnvelope(object map[string]any, schema, version, selfField string) error {
	if err := requireExactString(object, "schema", schema); err != nil {
		return err
	}
	if err := requireExactString(object, "schema_version", version); err != nil {
		return err
	}
	if err := requireDigest(object, selfField); err != nil {
		return err
	}
	return validateCommonRecordEnvelope(object)
}

func requireArrayRange(object map[string]any, name string, minimum, maximum int) ([]any, error) {
	values, err := requireArray(object, name, maximum)
	if err != nil {
		return nil, err
	}
	if len(values) < minimum {
		return nil, invalidIdentity("member %s requires at least %d entries", name, minimum)
	}
	return values, nil
}

func requireArrayMinimum(object map[string]any, name string, minimum int) ([]any, error) {
	value, ok := object[name]
	if !ok {
		return nil, invalidIdentity("identity input requires member %s", name)
	}
	values, ok := value.([]any)
	if !ok {
		return nil, invalidIdentity("member %s must be an array", name)
	}
	if len(values) < minimum {
		return nil, invalidIdentity("member %s requires at least %d entries", name, minimum)
	}
	return values, nil
}

func requirePositiveUint(object map[string]any, name string) (uint64, error) {
	value, err := requireUint(object, name, uint64(maxSafeInteger))
	if err != nil {
		return 0, err
	}
	if value == 0 {
		return 0, invalidIdentity("member %s must be greater than zero", name)
	}
	return value, nil
}

func nullablePositiveUint(object map[string]any, name string) (bool, error) {
	value, ok := object[name]
	if !ok {
		return false, invalidIdentity("identity input requires member %s", name)
	}
	if value == nil {
		return false, nil
	}
	_, err := requirePositiveUint(object, name)
	return err == nil, err
}

func nullableUintPresence(object map[string]any, name string) (uint64, bool, error) {
	value, ok := object[name]
	if !ok {
		return 0, false, invalidIdentity("identity input requires member %s", name)
	}
	if value == nil {
		return 0, false, nil
	}
	parsed, err := requireUint(object, name, uint64(maxSafeInteger))
	return parsed, err == nil, err
}

func requireUUIDv4(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if _, err := scalar.ParseUUIDv4(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

func requireNullableUUIDv4(object map[string]any, name string) (bool, error) {
	value, ok := object[name]
	if !ok {
		return false, invalidIdentity("identity input requires member %s", name)
	}
	if value == nil {
		return false, nil
	}
	return true, requireUUIDv4(object, name)
}

func requireProviderID(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if _, err := scalar.ParseProviderID(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

// requireTerminalBackendID enforces the Section 4.B terminal-backend-id scalar
// type. The declared bound is "1-128 ASCII bytes"; the declared grammar admits
// only ASCII, so the byte count and the Unicode character count coincide for
// every admitted value and the bound is measured in bytes as declared.
func requireTerminalBackendID(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	// The declared minimum of 1 is subsumed by requireString, which already
	// refuses the empty string; only the maximum needs its own branch here so no
	// unreachable check is left behind.
	if len(value) > 128 {
		return invalidIdentity("member %s must contain 1..128 ASCII bytes", name)
	}
	if !terminalBackendIDPattern.MatchString(value) {
		return invalidIdentity("member %s must match the terminal-backend-id grammar", name)
	}
	return nil
}

func requireBoundedLogicalIdentity(object map[string]any, name string) error {
	value, err := requireBoundedString(object, name, 1, 256)
	if err != nil {
		return err
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || windowsAbsolutePathPattern.MatchString(value) {
		return invalidIdentity("member %s must be a logical identity, not an absolute path", name)
	}
	return nil
}

func requireTimestamp(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if _, err := scalar.ParseTimestamp(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

func nullableBoundedString(object map[string]any, name string, minimum, maximum int) (string, bool, error) {
	value, present, err := nullableString(object, name)
	if err != nil || !present {
		return value, present, err
	}
	length := utf8.RuneCountInString(value)
	if length < minimum || length > maximum {
		return "", false, invalidIdentity("member %s must be null or contain %d..%d Unicode characters", name, minimum, maximum)
	}
	return value, true, nil
}

func invalidObservation(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidObservation, fmt.Sprintf(format, arguments...))
}
