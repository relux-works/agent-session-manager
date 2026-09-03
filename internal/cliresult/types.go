package cliresult

import (
	"fmt"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// sessionStates is the Section 5.7 SessionState registry in its declared order.
// Section 5.7 says "every RPC and CLI field typed SessionState uses exactly this
// registry", and that the spellings created, starting, and quiesced are not
// session lifecycle states.
var sessionStates = []string{
	"creating", "running", "idle", "quiescing", "checkpointing", "stopped",
	"materializing", "parked", "failed", "stale", "tombstoned",
}

// providerCapabilityNames is the capability-name vocabulary the Section 14.2
// SessionSummary bound "map(capability-name,CapabilitySummary)[0..7]" ranges
// over. It is projected from the reviewed catalog rather than retyped, so a
// name added to the pinned registry cannot exist in one place only, and the
// projection is asserted to be the seven-member Section 8.3 provider family
// that the [0..7] bound names.
var providerCapabilityNames = mustProviderCapabilityNames()

func mustProviderCapabilityNames() map[string]struct{} {
	names, err := providerCapabilityNamesFrom(catalog.Current().Capabilities)
	if err != nil {
		panic(fmt.Sprintf("cli result capability vocabulary is invalid: %v", err))
	}
	return names
}

func providerCapabilityNamesFrom(capabilities []catalog.Capability) (map[string]struct{}, error) {
	names := make(map[string]struct{})
	for _, capability := range capabilities {
		if capability.Family != "provider" {
			continue
		}
		if _, duplicate := names[string(capability.Name)]; duplicate {
			return nil, fmt.Errorf("capability %q is registered twice", capability.Name)
		}
		names[string(capability.Name)] = struct{}{}
	}
	if len(names) != maxSessionCapabilities {
		return nil, fmt.Errorf(
			"provider capability family has %d names, the Section 14.2 SessionSummary bound is %d",
			len(names), maxSessionCapabilities)
	}
	return names, nil
}

// maxSessionCapabilities is the Section 14.2 upper bound on the SessionSummary
// capabilities map.
const maxSessionCapabilities = 7

// CapabilityNames returns the capability-name vocabulary in sorted order. It is
// a vocabulary only: nothing here reports that a capability is available,
// enabled, or supported.
func CapabilityNames() []string {
	result := make([]string, 0, len(providerCapabilityNames))
	for name := range providerCapabilityNames {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// helloContractNames is the Section 11.2 contract-name vocabulary: "the
// contracts map contains exactly the fourteen displayed lower-snake-case keys".
// The same section states that "Configuration, provider protocol/manifest/probe,
// task-board bridge, materialization recovery state, Structured Error,
// Observation Event, and CLI Result MUST NOT appear in this map", so restricting
// the peer.probe contract map to these fourteen implements that prohibition
// rather than inventing a bound: any other key, including error and cli_result,
// is refused by construction.
var helloContractNames = map[string]struct{}{
	"rpc": {}, "session_record": {}, "session_event": {}, "lease": {},
	"checkpoint": {}, "workspace_group": {}, "provider_identity": {},
	"blob": {}, "transfer_manifest": {}, "chunk": {},
	"materialization_plan": {}, "tombstone": {}, "tombstone_ack": {},
	"task_board_bundle": {},
}

// ContractNames returns the contract-name vocabulary in sorted order.
func ContractNames() []string {
	result := make([]string, 0, len(helloContractNames))
	for name := range helloContractNames {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// validateCapabilitySummary checks the closed Section 14.2 CapabilitySummary:
// status, enabled, and detail, and nothing else.
//
// The relation between status and enabled is deliberately not enforced here.
// Section 7.4 states "only status=available permits enabled=true" for the
// Provider capability result, and Section 17.2 states that a binding mismatch
// "changes it to conditional/unsupported and prevents dependent activation";
// neither sentence is stated over this CLI type, and this validator does not
// see the negotiated tuple those sentences are evaluated against. Refusing the
// pair here would be an invented constraint, and claiming to check it would be
// worse: the check belongs to the capability resolver that owns the binding.
func validateCapabilitySummary(where string, object map[string]any) error {
	if err := requireClosedMembers(object, where, []string{"status", "enabled", "detail"}); err != nil {
		return err
	}
	if _, err := memberEnum(object, where, "status", "available", "conditional", "unsupported", "unknown"); err != nil {
		return err
	}
	if _, err := memberBool(object, where, "enabled"); err != nil {
		return err
	}
	_, err := memberString(object, where, "detail", 0, 2048)
	return err
}

var sessionSummaryMembers = []string{
	"session_id", "name", "kind", "provider_id", "owner_host_id", "owner_host_name",
	"lease_epoch", "lease_id", "local_role", "state", "newest_checkpoint_id",
	"newest_checkpoint_created_at", "workspace_status", "capabilities", "warnings",
}

// validateSessionSummary checks the closed Section 14.2 SessionSummary and
// returns its session_id, which is the ID a SessionSummary array is sorted by.
func validateSessionSummary(where string, object map[string]any) (string, error) {
	if err := requireClosedMembers(object, where, sessionSummaryMembers); err != nil {
		return "", err
	}
	sessionID, err := memberUUIDv7(object, where, "session_id")
	if err != nil {
		return "", err
	}
	if _, err := memberString(object, where, "name", 1, 64); err != nil {
		return "", err
	}
	if _, err := memberEnum(object, where, "kind", "direct", "task_board"); err != nil {
		return "", err
	}
	providerID, ok := object["provider_id"].(string)
	if !ok {
		return "", failf("%s.provider_id is not a JSON string", where)
	}
	if _, err := scalar.ParseProviderID(providerID); err != nil {
		return "", failf("%s.provider_id: %v", where, err)
	}
	if _, err := memberUUIDv7(object, where, "owner_host_id"); err != nil {
		return "", err
	}
	if _, err := memberString(object, where, "owner_host_name", 1, 64); err != nil {
		return "", err
	}
	if _, err := memberUint53(object, where, "lease_epoch", 1); err != nil {
		return "", err
	}
	if err := memberUUIDv4(object, where, "lease_id"); err != nil {
		return "", err
	}
	if _, err := memberEnum(object, where, "local_role", "owner", "replica"); err != nil {
		return "", err
	}
	state, err := memberEnum(object, where, "state", sessionStates...)
	if err != nil {
		return "", err
	}
	checkpointPresent, err := memberDigestOrNull(object, where, "newest_checkpoint_id")
	if err != nil {
		return "", err
	}
	if err := memberTimestampOrNull(object, where, "newest_checkpoint_created_at"); err != nil {
		return "", err
	}
	// Section 5.7: "No event or RPC result may derive stopped while
	// newest_checkpoint_id is null." A CLI Result is neither an event nor an
	// RPC result, but the SessionSummary it carries reports the same derived
	// state, and a summary that contradicts the state engine is not a
	// conforming projection of it.
	if state == "stopped" && !checkpointPresent {
		return "", failf("%s reports state stopped with a null newest_checkpoint_id", where)
	}
	if _, err := memberEnum(
		object, where, "workspace_status",
		"absent", "current", "staged", "conflict", "unsupported"); err != nil {
		return "", err
	}
	if err := validateCapabilityMap(where, object); err != nil {
		return "", err
	}
	if err := requireSortedUnique(object, where, "warnings", 0, 1024, stringElement(1, 1024)); err != nil {
		return "", err
	}
	return sessionID.String(), nil
}

// validateCapabilityMap checks map(capability-name,CapabilitySummary)[0..7].
// Section 1.6 says a map's "member names are data, not schema fields", so the
// bound is on the member count and the names are checked against the closed
// capability-name vocabulary.
func validateCapabilityMap(where string, object map[string]any) error {
	capabilities, err := memberObject(object, where, "capabilities")
	if err != nil {
		return err
	}
	// This count bound is subsumed by the vocabulary check below: the
	// capability-name vocabulary has exactly maxSessionCapabilities members, so
	// a map whose names are all admitted cannot exceed the count, and the
	// eighth entry is refused by name first. It is retained rather than deleted
	// so the Section 14.2 bound stays visible in the code that implements the
	// type, and TestCapabilityCountBoundIsSubsumedByTheVocabulary pins the
	// invariant that makes it subsumed.
	if len(capabilities) > maxSessionCapabilities {
		return failf("%s.capabilities has %d members, the bound is 0..%d",
			where, len(capabilities), maxSessionCapabilities)
	}
	for _, name := range sortedKeys(capabilities) {
		if _, known := providerCapabilityNames[name]; !known {
			return failf("%s.capabilities carries %q, which is not a capability name", where, name)
		}
		summary, ok := capabilities[name].(map[string]any)
		if !ok {
			return failf("%s.capabilities[%q] is not a JSON object", where, name)
		}
		if err := validateCapabilitySummary(fmt.Sprintf("%s.capabilities[%q]", where, name), summary); err != nil {
			return err
		}
	}
	return nil
}

// validatePathDiff checks the closed Section 14.2 PathDiff. It returns the
// empty ordering key: a PathDiff is keyed by a path, not by an ID, so the
// Section 14.2 sentence "digest arrays and object arrays keyed by an ID are
// sorted bytewise by that ID" does not reach it and no ordering is imposed.
func validatePathDiff(where string, object map[string]any) (string, error) {
	members := []string{"path", "classification", "source_digest", "destination_digest"}
	if err := requireClosedMembers(object, where, members); err != nil {
		return "", err
	}
	path, ok := object["path"].(string)
	if !ok {
		return "", failf("%s.path is not a JSON string", where)
	}
	if _, err := scalar.ParseRelativePath(path); err != nil {
		return "", failf("%s.path: %v", where, err)
	}
	if _, err := memberEnum(
		object, where, "classification",
		"added", "removed", "modified", "type_changed", "mode_changed", "conflict"); err != nil {
		return "", err
	}
	if _, err := memberDigestOrNull(object, where, "source_digest"); err != nil {
		return "", err
	}
	if _, err := memberDigestOrNull(object, where, "destination_digest"); err != nil {
		return "", err
	}
	return "", nil
}

// validatePeerSummary checks the closed Section 14.2 PeerSummary and returns
// its host_id, which is the ID a PeerSummary array is sorted by.
func validatePeerSummary(where string, object map[string]any) (string, error) {
	members := []string{"host_id", "name", "platform", "reachable", "last_successful_sync_at", "degraded_codes"}
	if err := requireClosedMembers(object, where, members); err != nil {
		return "", err
	}
	hostID, err := memberUUIDv7(object, where, "host_id")
	if err != nil {
		return "", err
	}
	if _, err := memberString(object, where, "name", 1, 64); err != nil {
		return "", err
	}
	if _, err := memberEnum(object, where, "platform", "macos", "linux", "wsl2", "windows"); err != nil {
		return "", err
	}
	if _, err := memberBool(object, where, "reachable"); err != nil {
		return "", err
	}
	if err := memberTimestampOrNull(object, where, "last_successful_sync_at"); err != nil {
		return "", err
	}
	if err := requireSortedUnique(object, where, "degraded_codes", 0, 1024, stringElement(1, 1024)); err != nil {
		return "", err
	}
	return hostID.String(), nil
}

// validateCLIFinding checks the closed Section 14.2 CLIFinding. It returns the
// empty ordering key: a finding carries no ID, so no ordering is imposed.
func validateCLIFinding(where string, object map[string]any) (string, error) {
	members := []string{"severity", "code", "message", "remediation", "source"}
	if err := requireClosedMembers(object, where, members); err != nil {
		return "", err
	}
	if _, err := memberEnum(object, where, "severity", "info", "warning", "error"); err != nil {
		return "", err
	}
	if _, err := memberString(object, where, "code", 1, 128); err != nil {
		return "", err
	}
	if _, err := memberString(object, where, "message", 1, 4096); err != nil {
		return "", err
	}
	if err := memberStringOrNull(object, where, "remediation", 1, 4096); err != nil {
		return "", err
	}
	_, err := memberEnum(
		object, where, "source",
		"core", "terminal", "provider", "mesh", "workspace", "task_board")
	return "", err
}

var materializationSummaryMembers = []string{
	"session_id", "checkpoint_id", "materialization_id", "mode", "destination_path",
	"destination_classification", "preserved_checkpoint_id", "committed", "ownership_changed",
}

// destinationClasses is the Section 11.7 ordered five-value DestinationClass
// registry. The former spellings matching_managed, divergent_managed, and
// unmanaged are invalid in version 1.0.0.
var destinationClasses = []string{"absent", "empty", "managed_unchanged", "managed_divergent", "unmanaged_nonempty"}

// validateMaterializationSummary checks the closed Section 14.2
// MaterializationSummary together with the success rules the same section
// states over it.
func validateMaterializationSummary(where string, object map[string]any) error {
	if err := requireClosedMembers(object, where, materializationSummaryMembers); err != nil {
		return err
	}
	if _, err := memberUUIDv7(object, where, "session_id"); err != nil {
		return err
	}
	if err := memberDigest(object, where, "checkpoint_id"); err != nil {
		return err
	}
	if _, err := memberUUIDv7(object, where, "materialization_id"); err != nil {
		return err
	}
	mode, err := memberEnum(object, where, "mode", "default", "copy", "worktree", "replace_managed_replica")
	if err != nil {
		return err
	}
	if err := validateAbsolutePathMember(object, where, "destination_path"); err != nil {
		return err
	}
	classification, err := memberEnum(object, where, "destination_classification", destinationClasses...)
	if err != nil {
		return err
	}
	preserved, err := memberDigestOrNull(object, where, "preserved_checkpoint_id")
	if err != nil {
		return err
	}
	// "A successful MaterializationSummary requires committed = true and
	// ownership_changed = false."
	if err := requireTrue(object, where, "committed"); err != nil {
		return err
	}
	if err := requireFalse(object, where, "ownership_changed"); err != nil {
		return err
	}
	// "Replacement requires a non-null preserved checkpoint; every other mode
	// requires null."
	replacement := mode == "replace_managed_replica"
	if replacement != preserved {
		if replacement {
			return failf("%s replaces a managed replica with a null preserved_checkpoint_id", where)
		}
		return failf("%s uses mode %q with a non-null preserved_checkpoint_id", where, mode)
	}
	// "replacement reports managed_divergent, and no success may report
	// unmanaged_nonempty." The copy/worktree sentence is qualified with
	// "normally" and states no rule, so it is not enforced.
	if replacement && classification != "managed_divergent" {
		return failf("%s replaces a managed replica but reports destination_classification %q", where, classification)
	}
	if classification == "unmanaged_nonempty" {
		return failf("%s reports destination_classification unmanaged_nonempty in a success object", where)
	}
	return nil
}

// validateAbsolutePathMember checks a Section 1.6 absolute-path member.
//
// Section 1.6 defines absolute-path as "absolute and lexically normalized for
// the platform named by its containing request". A CLI Result names no
// platform, so the member is admitted when it is a valid absolute path for at
// least one supported platform. That is a stated bound rather than a hidden
// one: a caller that knows the emitting host's platform must re-check it there,
// and VerifyDestinationPlatform is the narrowing hook for exactly that.
func validateAbsolutePathMember(object map[string]any, where, name string) error {
	value, ok := object[name].(string)
	if !ok {
		return failf("%s.%s is not a JSON string", where, name)
	}
	for _, platform := range supportedPlatforms {
		if _, err := scalar.ParseAbsolutePath(platform, value); err == nil {
			return nil
		}
	}
	return failf("%s.%s is not an absolute path on any supported platform", where, name)
}

var supportedPlatforms = []scalar.Platform{
	scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2, scalar.PlatformWindows,
}
