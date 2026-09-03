package cliresult

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// The eighteen closed bodies of the Section 14.2 command-tag table. Each
// validator reproduces exactly one row of that table plus the cross-member
// sentences the same section states over it, and nothing else.

// validateCancelBody checks {name:string[1..64],cancelled:boolean} "with true
// required".
func validateCancelBody(body map[string]any) error {
	if err := requireClosedMembers(body, "body", []string{"name", "cancelled"}); err != nil {
		return err
	}
	if _, err := memberString(body, "body", "name", 1, 64); err != nil {
		return err
	}
	return requireTrue(body, "body", "cancelled")
}

// validateStartBody checks
// {session:SessionSummary,execution_profile:standard|yolo,terminal_backend:tmux|conpty}.
func validateStartBody(body map[string]any) error {
	members := []string{"session", "execution_profile", "terminal_backend"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if err := validateSessionMember(body, "body"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "execution_profile", "standard", "yolo"); err != nil {
		return err
	}
	_, err := memberEnum(body, "body", "terminal_backend", "tmux", "conpty")
	return err
}

// validateListBody checks
// {sessions:SessionSummary[0..65536],partial:boolean,unreachable_peer_ids:sorted unique UUIDv7[0..1024]}.
//
// The sessions array is an object array keyed by an ID, so Section 14.2's
// "digest arrays and object arrays keyed by an ID are sorted bytewise by that
// ID" applies to it.
func validateListBody(body map[string]any) error {
	members := []string{"sessions", "partial", "unreachable_peer_ids"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if err := requireSortedUnique(
		body, "body", "sessions", 0, 65536, objectElement(validateSessionSummary)); err != nil {
		return err
	}
	if _, err := memberBool(body, "body", "partial"); err != nil {
		return err
	}
	return requireSortedUnique(body, "body", "unreachable_peer_ids", 0, 1024, uuidv7Element)
}

// validateStatusBody checks
// {session:SessionSummary,process_present:boolean,active_operation_id:UUIDv7|null,last_successful_sync:map(UUIDv7,timestamp)[0..1024]}.
func validateStatusBody(body map[string]any) error {
	members := []string{"session", "process_present", "active_operation_id", "last_successful_sync"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if err := validateSessionMember(body, "body"); err != nil {
		return err
	}
	if _, err := memberBool(body, "body", "process_present"); err != nil {
		return err
	}
	if _, _, err := memberUUIDv7OrNull(body, "body", "active_operation_id"); err != nil {
		return err
	}
	return validateSyncMap(body, "body", "last_successful_sync")
}

// validateSyncMap checks map(UUIDv7,timestamp)[0..1024]. Section 1.6 says a
// map's "member names are data, not schema fields", so the key grammar is
// checked on the name and the bound is on the member count.
func validateSyncMap(body map[string]any, where, name string) error {
	entries, err := memberObject(body, where, name)
	if err != nil {
		return err
	}
	if len(entries) > 1024 {
		return failf("%s.%s has %d members, the bound is 0..1024", where, name, len(entries))
	}
	for _, key := range sortedKeys(entries) {
		if _, err := uuidv7Element(where+"."+name+" key", key); err != nil {
			return err
		}
		if err := memberTimestampOrNull(entries, where+"."+name, key); err != nil {
			return err
		}
		if entries[key] == nil {
			return failf("%s.%s[%q] is null; the declared value type is timestamp", where, name, key)
		}
	}
	return nil
}

// validateAttachBody checks
// {session:SessionSummary,mode:local|remote,attached_owner_host_id:UUIDv7,detached:boolean,provider_exit_code:int32|null}.
func validateAttachBody(body map[string]any) error {
	members := []string{"session", "mode", "attached_owner_host_id", "detached", "provider_exit_code"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if err := validateSessionMember(body, "body"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "mode", "local", "remote"); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "attached_owner_host_id"); err != nil {
		return err
	}
	if _, err := memberBool(body, "body", "detached"); err != nil {
		return err
	}
	return memberInt32OrNull(body, "body", "provider_exit_code")
}

var takeoverMembers = []string{
	"mode", "workspace_mode", "destination_host_id", "source_host_id", "affected_session_ids",
	"lease_epoch", "lease_id", "checkpoint_id", "state", "materialized", "adopted", "resumed", "warnings",
}

// validateTakeoverBody checks the closed takeover row. The adoption rule that
// depends on the session kind is not checked here; see verifyTakeoverAdoption.
func validateTakeoverBody(body map[string]any) error {
	if err := requireClosedMembers(body, "body", takeoverMembers); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "mode", "graceful", "force"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "workspace_mode", "whole_group", "separate_worktrees"); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "destination_host_id"); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "source_host_id"); err != nil {
		return err
	}
	if err := requireSortedUnique(body, "body", "affected_session_ids", 1, 1024, uuidv7Element); err != nil {
		return err
	}
	if _, err := memberUint53(body, "body", "lease_epoch", 1); err != nil {
		return err
	}
	if err := memberUUIDv4(body, "body", "lease_id"); err != nil {
		return err
	}
	if err := memberDigest(body, "body", "checkpoint_id"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "state", "running", "idle", "stopped", "failed"); err != nil {
		return err
	}
	for _, name := range []string{"materialized", "adopted", "resumed"} {
		if _, err := memberBool(body, "body", name); err != nil {
			return err
		}
	}
	return requireSortedUnique(body, "body", "warnings", 0, 1024, stringElement(1, 1024))
}

// verifyTakeoverAdoption checks the Section 14.2 sentence "for a task-board
// takeover, adopted MUST be true before resumed can be true; for a direct
// takeover it MUST be false".
//
// It runs only for the takeover tag, and only once a session kind is known. New
// requires that kind; Decode cannot have it, because the body carries none.
func verifyTakeoverAdoption(command Command, body map[string]any, kind SessionKind) error {
	if command != CommandTakeover {
		return nil
	}
	adopted, err := memberBool(body, "body", "adopted")
	if err != nil {
		return err
	}
	resumed, err := memberBool(body, "body", "resumed")
	if err != nil {
		return err
	}
	switch kind {
	case KindDirect:
		if adopted {
			return failf("a direct takeover reports adopted true; Section 14.2 requires false")
		}
		return nil
	case KindTaskBoard:
		if resumed && !adopted {
			return failf("a task-board takeover reports resumed true before adopted")
		}
		return nil
	default:
		return failf("session kind %q is not direct or task_board", kind)
	}
}

// validateForkBody checks
// {source_session_id:UUIDv7,source_checkpoint_id:digest,session:SessionSummary,workspace_group_id:UUIDv7,provider_fork_mode:native|supported_import|task_board_clone}.
func validateForkBody(body map[string]any) error {
	members := []string{
		"source_session_id", "source_checkpoint_id", "session", "workspace_group_id", "provider_fork_mode",
	}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "source_session_id"); err != nil {
		return err
	}
	if err := memberDigest(body, "body", "source_checkpoint_id"); err != nil {
		return err
	}
	if err := validateSessionMember(body, "body"); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "workspace_group_id"); err != nil {
		return err
	}
	_, err := memberEnum(body, "body", "provider_fork_mode", "native", "supported_import", "task_board_clone")
	return err
}

var stopMembers = []string{
	"session", "graceful", "checkpoint_id", "resumable", "bootstrap_aborted", "process_closed", "store_closed",
}

// validateStopBody checks the closed stop row together with the Section 14.2
// stop tuple, which is "mapped losslessly from RPC session.stop".
func validateStopBody(body map[string]any) error {
	if err := requireClosedMembers(body, "body", stopMembers); err != nil {
		return err
	}
	if err := validateSessionMember(body, "body"); err != nil {
		return err
	}
	graceful, err := memberBool(body, "body", "graceful")
	if err != nil {
		return err
	}
	checkpointPresent, err := memberDigestOrNull(body, "body", "checkpoint_id")
	if err != nil {
		return err
	}
	resumable, err := memberBool(body, "body", "resumable")
	if err != nil {
		return err
	}
	bootstrapAborted, err := memberBool(body, "body", "bootstrap_aborted")
	if err != nil {
		return err
	}
	// "Process and store closure must be true in every success object;
	// otherwise the command returns Structured Error instead."
	if err := requireTrue(body, "body", "process_closed"); err != nil {
		return err
	}
	if err := requireTrue(body, "body", "store_closed"); err != nil {
		return err
	}
	summary, err := memberObject(body, "body", "session")
	if err != nil {
		return err
	}
	state, err := memberEnum(summary, "body.session", "state", sessionStates...)
	if err != nil {
		return err
	}
	// "checkpoint_id = null is valid only with graceful=false, resumable=false,
	// bootstrap_aborted=true, and nested session state failed."
	if !checkpointPresent {
		if graceful || resumable || !bootstrapAborted || state != "failed" {
			return failf(
				"body.checkpoint_id is null with graceful=%t resumable=%t bootstrap_aborted=%t state=%q; "+
					"Section 14.2 admits a null checkpoint only with false, false, true, and failed",
				graceful, resumable, bootstrapAborted, state)
		}
		return nil
	}
	// "A non-null checkpoint with nested state stopped requires resumable=true
	// and bootstrap_aborted=false."
	if state == "stopped" && (!resumable || bootstrapAborted) {
		return failf(
			"body carries a checkpoint with nested state stopped, resumable=%t and bootstrap_aborted=%t; "+
				"Section 14.2 requires true and false",
			resumable, bootstrapAborted)
	}
	return nil
}

// validateResumeBody checks
// {session:SessionSummary,checkpoint_id:digest,terminal_backend:tmux|conpty,native_session_id:string[1..512]}.
func validateResumeBody(body map[string]any) error {
	members := []string{"session", "checkpoint_id", "terminal_backend", "native_session_id"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if err := validateSessionMember(body, "body"); err != nil {
		return err
	}
	if err := memberDigest(body, "body", "checkpoint_id"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "terminal_backend", "tmux", "conpty"); err != nil {
		return err
	}
	_, err := memberString(body, "body", "native_session_id", 1, 512)
	return err
}

var syncMembers = []string{
	"peer_ids", "record_count", "blob_count", "byte_count",
	"checkpoint_ids", "materialized", "partial", "transfer_id",
}

// validateSyncBody checks the closed sync row.
func validateSyncBody(body map[string]any) error {
	if err := requireClosedMembers(body, "body", syncMembers); err != nil {
		return err
	}
	if err := requireSortedUnique(body, "body", "peer_ids", 1, 1024, uuidv7Element); err != nil {
		return err
	}
	for _, name := range []string{"record_count", "blob_count", "byte_count"} {
		if _, err := memberUint53(body, "body", name, 0); err != nil {
			return err
		}
	}
	if err := requireSortedUnique(body, "body", "checkpoint_ids", 0, 4096, digestElement); err != nil {
		return err
	}
	for _, name := range []string{"materialized", "partial"} {
		if _, err := memberBool(body, "body", name); err != nil {
			return err
		}
	}
	_, _, err := memberUUIDv7OrNull(body, "body", "transfer_id")
	return err
}

// validateDiffBody checks
// {session_id:UUIDv7,peer_host_id:UUIDv7|null,classification:identical|different|conflict,entries:PathDiff[0..65536]}.
func validateDiffBody(body map[string]any) error {
	members := []string{"session_id", "peer_host_id", "classification", "entries"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "session_id"); err != nil {
		return err
	}
	if _, _, err := memberUUIDv7OrNull(body, "body", "peer_host_id"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "classification", "identical", "different", "conflict"); err != nil {
		return err
	}
	return requireUnorderedArray(body, "body", "entries", 0, 65536, objectElement(validatePathDiff))
}

// validateMaterializeBody checks the materialize row, whose body is exactly a
// MaterializationSummary.
func validateMaterializeBody(body map[string]any) error {
	return validateMaterializationSummary("body", body)
}

// validateDoctorBody checks {healthy:boolean,findings:CLIFinding[0..4096]}.
func validateDoctorBody(body map[string]any) error {
	if err := requireClosedMembers(body, "body", []string{"healthy", "findings"}); err != nil {
		return err
	}
	if _, err := memberBool(body, "body", "healthy"); err != nil {
		return err
	}
	return requireUnorderedArray(body, "body", "findings", 0, 4096, objectElement(validateCLIFinding))
}

// validateLogsBody checks
// {emitting_host_id:UUIDv7,events:Observation Event[0..65536],next_cursor:string[1..512]|null}.
//
// Two rules are enforced beyond the row itself. Section 1.6 requires a field
// described as a schema object to "contain the complete object, including its
// own schema and schema_version" and to "validate against the named section",
// so each element is validated as a Section 18.1 Observation Event. Section
// 14.1 states that "every returned Observation Event MUST have host_id equal to
// that emitter", so each event is checked against emitting_host_id.
//
// The element order is not checked. Section 14.1 requires the Section 18.1
// total order, which is a property of a durable stream this package cannot see
// from one array, so the ordering is reported here as unenforced rather than
// approximated by a bytewise sort that Section 18.1 does not define.
func validateLogsBody(body map[string]any) error {
	members := []string{"emitting_host_id", "events", "next_cursor"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	emitter, err := memberUUIDv7(body, "body", "emitting_host_id")
	if err != nil {
		return err
	}
	if err := memberStringOrNull(body, "body", "next_cursor", 1, 512); err != nil {
		return err
	}
	return requireUnorderedArray(body, "body", "events", 0, 65536, observationEventElement(emitter.String()))
}

func observationEventElement(emitter string) elementValidator {
	return func(where string, element any) (string, error) {
		object, ok := element.(map[string]any)
		if !ok {
			return "", failf("%s is not a JSON object", where)
		}
		encoded, err := json.Marshal(object)
		if err != nil {
			return "", failf("%s: encode: %v", where, err)
		}
		if err := canonicaljson.ValidateObservationEvent(encoded); err != nil {
			return "", failf("%s: %v", where, err)
		}
		hostID, ok := object["host_id"].(string)
		if !ok {
			return "", failf("%s.host_id is not a JSON string", where)
		}
		if hostID != emitter {
			return "", failf("%s.host_id %s differs from body.emitting_host_id %s", where, hostID, emitter)
		}
		return "", nil
	}
}

// validatePeerListBody checks {peers:PeerSummary[0..1024]}. The peers array is
// an object array keyed by an ID, so the Section 14.2 sort rule applies.
func validatePeerListBody(body map[string]any) error {
	if err := requireClosedMembers(body, "body", []string{"peers"}); err != nil {
		return err
	}
	return requireSortedUnique(body, "body", "peers", 0, 1024, objectElement(validatePeerSummary))
}

// validatePeerProbeBody checks
// {peer:PeerSummary,contracts:map(contract-name,sorted unique semver[1..16]),round_trip_ms:uint53}.
func validatePeerProbeBody(body map[string]any) error {
	members := []string{"peer", "contracts", "round_trip_ms"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	peer, err := memberObject(body, "body", "peer")
	if err != nil {
		return err
	}
	if _, err := validatePeerSummary("body.peer", peer); err != nil {
		return err
	}
	if err := validateContractMap(body, "body", "contracts"); err != nil {
		return err
	}
	_, err = memberUint53(body, "body", "round_trip_ms", 0)
	return err
}

func validateContractMap(body map[string]any, where, name string) error {
	contracts, err := memberObject(body, where, name)
	if err != nil {
		return err
	}
	for _, key := range sortedKeys(contracts) {
		if _, known := helloContractNames[key]; !known {
			return failf("%s.%s carries %q, which is not a contract name", where, name, key)
		}
		if err := requireSortedUnique(
			contracts, where+"."+name, key, 1, 16, semverElement); err != nil {
			return err
		}
	}
	return nil
}

func semverElement(where string, element any) (string, error) {
	value, ok := element.(string)
	if !ok {
		return "", failf("%s is not a JSON string", where)
	}
	if !semverPattern.MatchString(value) {
		return "", failf("%s %q is not a semantic version", where, value)
	}
	return value, nil
}

// validateSetProfileBody checks
// {session_id:UUIDv7,previous_profile:standard|yolo,new_profile:standard|yolo,event_id:digest}.
func validateSetProfileBody(body map[string]any) error {
	members := []string{"session_id", "previous_profile", "new_profile", "event_id"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "session_id"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "previous_profile", "standard", "yolo"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "new_profile", "standard", "yolo"); err != nil {
		return err
	}
	return memberDigest(body, "body", "event_id")
}

// validatePaneBody checks
// {session_id:UUIDv7,result:attached|parked|resumed|stopped,winning_owner_host_id:UUIDv7,lease_epoch:uint53>0}.
func validatePaneBody(body map[string]any) error {
	members := []string{"session_id", "result", "winning_owner_host_id", "lease_epoch"}
	if err := requireClosedMembers(body, "body", members); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "session_id"); err != nil {
		return err
	}
	if _, err := memberEnum(body, "body", "result", "attached", "parked", "resumed", "stopped"); err != nil {
		return err
	}
	if _, err := memberUUIDv7(body, "body", "winning_owner_host_id"); err != nil {
		return err
	}
	_, err := memberUint53(body, "body", "lease_epoch", 1)
	return err
}

// validateSessionMember validates a body member typed SessionSummary.
func validateSessionMember(body map[string]any, where string) error {
	summary, err := memberObject(body, where, "session")
	if err != nil {
		return err
	}
	_, err = validateSessionSummary(where+".session", summary)
	return err
}

// VerifyDestinationPlatform re-checks a materialize result's destination_path
// against the platform of the host that emitted it. It is the narrowing hook
// for the stated bound on validateAbsolutePathMember: a CLI Result names no
// platform, so the shape validator can only require the path to be absolute on
// some supported platform, and a caller that knows the emitting host must
// re-check it there.
func (result *Result) VerifyDestinationPlatform(platform string) error {
	if result.command != CommandMaterialize {
		return failf("command %q carries no destination path", result.command)
	}
	parsed, err := parsePlatform(platform)
	if err != nil {
		return err
	}
	value, ok := result.body["destination_path"].(string)
	if !ok {
		return failf("body.destination_path is not a JSON string")
	}
	if _, err := parseAbsolutePath(parsed, value); err != nil {
		return failf("body.destination_path is not an absolute path on %s: %v", platform, err)
	}
	return nil
}
