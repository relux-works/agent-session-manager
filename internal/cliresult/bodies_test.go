package cliresult

import (
	"encoding/json"
	"strings"
	"testing"
)

// specWithBody rebuilds a valid spec with a replaced body.
func specWithBody(t *testing.T, command Command, body map[string]any) Spec {
	t.Helper()
	spec := validSpec(t, command)
	spec.Body = body
	return spec
}

func refuse(t *testing.T, name string, spec Spec) {
	t.Helper()
	if _, err := New(spec); err == nil {
		t.Fatalf("New admitted %s", name)
	}
}

func admit(t *testing.T, name string, spec Spec) {
	t.Helper()
	if _, err := New(spec); err != nil {
		t.Fatalf("New refused %s: %v", name, err)
	}
}

// TestEveryBodyIsClosed narrows the closed-body gate for every implemented
// command: each declared member is required, and one extra member is refused.
// Section 1.6 makes that explicit - "every other embedded object is closed: an
// unknown member MUST be rejected even when its containing top-level object is
// otherwise valid".
func TestEveryBodyIsClosed(t *testing.T) {
	for _, command := range ImplementedCommands() {
		t.Run(string(command), func(t *testing.T) {
			base := validSpec(t, command).Body.(map[string]any)
			if len(base) == 0 {
				t.Fatalf("fixture for %q has no members", command)
			}
			for member := range base {
				body := cloneObject(base)
				delete(body, member)
				refuse(t, "a body without "+member, specWithBody(t, command, body))
			}
			body := cloneObject(base)
			body["works_relux_extra"] = "smuggled"
			refuse(t, "a body with an extra member", specWithBody(t, command, body))
		})
	}
}

// TestStopTupleNarrowsEveryDeclaredCondition attacks the Section 14.2 stop
// tuple one condition at a time. Each refusal below flips exactly one member of
// an otherwise valid object, so a gate narrowed to admit one wrong combination
// fails here rather than only a gate deleted outright.
func TestStopTupleNarrowsEveryDeclaredCondition(t *testing.T) {
	abortedSummary := sessionSummary(fixtureSessionID)
	abortedSummary["state"] = "failed"
	aborted := map[string]any{
		"session":           abortedSummary,
		"graceful":          false,
		"checkpoint_id":     nil,
		"resumable":         false,
		"bootstrap_aborted": true,
		"process_closed":    true,
		"store_closed":      true,
	}
	admit(t, "the only tuple a null checkpoint admits", specWithBody(t, CommandStop, aborted))
	admit(t, "the resumable stopped tuple", specWithBody(t, CommandStop, stopBody()))

	// "checkpoint_id = null is valid only with graceful=false, resumable=false,
	// bootstrap_aborted=true, and nested session state failed."
	refuse(t, "a null checkpoint with graceful=true",
		specWithBody(t, CommandStop, mutateBody(aborted, "graceful", true)))
	refuse(t, "a null checkpoint with resumable=true",
		specWithBody(t, CommandStop, mutateBody(aborted, "resumable", true)))
	refuse(t, "a null checkpoint with bootstrap_aborted=false",
		specWithBody(t, CommandStop, mutateBody(aborted, "bootstrap_aborted", false)))
	for _, state := range []string{"stopped", "running", "idle", "stale", "tombstoned"} {
		summary := sessionSummary(fixtureSessionID)
		summary["state"] = state
		if state == "stopped" {
			summary["newest_checkpoint_id"] = fixtureCheckpointID
		}
		refuse(t, "a null checkpoint with nested state "+state,
			specWithBody(t, CommandStop, mutateBody(aborted, "session", summary)))
	}

	// "A non-null checkpoint with nested state stopped requires resumable=true
	// and bootstrap_aborted=false."
	refuse(t, "a stopped checkpointed session that is not resumable",
		specWithBody(t, CommandStop, mutateBody(stopBody(), "resumable", false)))
	refuse(t, "a stopped checkpointed session with an aborted bootstrap",
		specWithBody(t, CommandStop, mutateBody(stopBody(), "bootstrap_aborted", true)))

	// "Process and store closure must be true in every success object;
	// otherwise the command returns Structured Error instead."
	refuse(t, "an unclosed process",
		specWithBody(t, CommandStop, mutateBody(stopBody(), "process_closed", false)))
	refuse(t, "an unclosed store",
		specWithBody(t, CommandStop, mutateBody(stopBody(), "store_closed", false)))
}

// TestMaterializationSuccessRulesNarrowEachDeclaredCondition attacks the
// Section 14.2 MaterializationSummary success rules one member at a time.
func TestMaterializationSuccessRulesNarrowEachDeclaredCondition(t *testing.T) {
	base := materializationSummary()
	admit(t, "a committed copy materialization", specWithBody(t, CommandMaterialize, base))

	refuse(t, "an uncommitted materialization",
		specWithBody(t, CommandMaterialize, mutateBody(base, "committed", false)))
	refuse(t, "a materialization that changed ownership",
		specWithBody(t, CommandMaterialize, mutateBody(base, "ownership_changed", true)))
	refuse(t, "an unmanaged_nonempty destination",
		specWithBody(t, CommandMaterialize, mutateBody(base, "destination_classification", "unmanaged_nonempty")))

	// "Replacement requires a non-null preserved checkpoint; every other mode
	// requires null."
	for _, mode := range []string{"default", "copy", "worktree"} {
		body := mutateBody(base, "mode", mode)
		body["preserved_checkpoint_id"] = fixtureOtherDigest
		refuse(t, mode+" with a preserved checkpoint", specWithBody(t, CommandMaterialize, body))
	}
	replacement := mutateBody(base, "mode", "replace_managed_replica")
	replacement["destination_classification"] = "managed_divergent"
	refuse(t, "replacement without a preserved checkpoint",
		specWithBody(t, CommandMaterialize, replacement))
	replacement["preserved_checkpoint_id"] = fixtureOtherDigest
	admit(t, "a conforming replacement", specWithBody(t, CommandMaterialize, replacement))

	// "replacement reports managed_divergent".
	for _, classification := range []string{"absent", "empty", "managed_unchanged"} {
		refuse(t, "replacement reporting "+classification,
			specWithBody(t, CommandMaterialize, mutateBody(replacement, "destination_classification", classification)))
	}
	// The former Section 11.7 spellings are invalid in version 1.0.0.
	for _, classification := range []string{"matching_managed", "divergent_managed", "unmanaged"} {
		refuse(t, "the retired spelling "+classification,
			specWithBody(t, CommandMaterialize, mutateBody(base, "destination_classification", classification)))
	}
}

// TestTakeoverAdoptionRuleIsCheckedPerSessionKind narrows the Section 14.2
// sentence "for a task-board takeover, adopted MUST be true before resumed can
// be true; for a direct takeover it MUST be false", on the writing side and on
// the reading side.
func TestTakeoverAdoptionRuleIsCheckedPerSessionKind(t *testing.T) {
	body := takeoverBody()
	cases := []struct {
		name     string
		kind     SessionKind
		adopted  bool
		resumed  bool
		admitted bool
	}{
		{"direct, not adopted, resumed", KindDirect, false, true, true},
		{"direct, not adopted, not resumed", KindDirect, false, false, true},
		{"direct, adopted", KindDirect, true, false, false},
		{"direct, adopted and resumed", KindDirect, true, true, false},
		{"task-board, adopted and resumed", KindTaskBoard, true, true, true},
		{"task-board, adopted, not resumed", KindTaskBoard, true, false, true},
		{"task-board, not adopted, not resumed", KindTaskBoard, false, false, true},
		{"task-board, resumed before adopted", KindTaskBoard, false, true, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := mutateBody(mutateBody(body, "adopted", testCase.adopted), "resumed", testCase.resumed)
			spec := specWithBody(t, CommandTakeover, mutated)
			spec.SessionKind = testCase.kind
			result, err := New(spec)
			if testCase.admitted != (err == nil) {
				t.Fatalf("New = %v, admitted want %t", err, testCase.admitted)
			}
			if !testCase.admitted {
				return
			}
			// The reading side reaches the same verdict once the kind is known.
			encoded, err := result.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			decoded, err := Decode(Version100, encoded)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if err := decoded.VerifyTakeoverAdoption(testCase.kind); err != nil {
				t.Fatalf("VerifyTakeoverAdoption(%q): %v", testCase.kind, err)
			}
			other := KindDirect
			if testCase.kind == KindDirect {
				other = KindTaskBoard
			}
			// Only the direct arm can contradict the other kind: task-board
			// admits every direct-valid combination except an adopted one.
			if testCase.adopted {
				if err := decoded.VerifyTakeoverAdoption(other); err == nil && other == KindDirect {
					t.Fatalf("VerifyTakeoverAdoption(direct) admitted an adopted takeover")
				}
			}
		})
	}
}

// TestTakeoverRequiresASessionKindAndOtherCommandsRefuseOne proves the
// constructor cannot silently skip the adoption rule, and that no other command
// can pretend to carry one.
func TestTakeoverRequiresASessionKindAndOtherCommandsRefuseOne(t *testing.T) {
	spec := validSpec(t, CommandTakeover)
	spec.SessionKind = ""
	refuse(t, "a takeover without a session kind", spec)
	spec.SessionKind = "task-board"
	refuse(t, "a takeover with a misspelled session kind", spec)

	other := validSpec(t, CommandStart)
	other.SessionKind = KindDirect
	refuse(t, "a start carrying a session kind", other)

	result := mustResult(t, validSpec(t, CommandStart))
	if err := result.VerifyTakeoverAdoption(KindDirect); err != nil {
		t.Fatalf("VerifyTakeoverAdoption on a non-takeover: %v", err)
	}
	if err := mustResult(t, validSpec(t, CommandTakeover)).VerifyTakeoverAdoption("neither"); err == nil {
		t.Fatalf("VerifyTakeoverAdoption admitted an unknown session kind")
	}
}

// TestIDKeyedArraysAreSortedAndUnique narrows the Section 14.2 rule "digest
// arrays and object arrays keyed by an ID are sorted bytewise by that ID". Each
// case reorders or repeats exactly one array of an otherwise valid body.
func TestIDKeyedArraysAreSortedAndUnique(t *testing.T) {
	low, high := fixtureSessionID, fixtureSourceHostID
	if low > high {
		low, high = high, low
	}
	t.Run("list.sessions", func(t *testing.T) {
		body := validSpec(t, CommandList).Body.(map[string]any)
		sorted := mutateBody(body, "sessions", []any{sessionSummary(low), sessionSummary(high)})
		spec := specWithBody(t, CommandList, sorted)
		spec.IDs = NoIDs()
		admit(t, "sorted sessions", spec)
		spec = specWithBody(t, CommandList, mutateBody(body, "sessions",
			[]any{sessionSummary(high), sessionSummary(low)}))
		spec.IDs = NoIDs()
		refuse(t, "unsorted sessions", spec)
		spec = specWithBody(t, CommandList, mutateBody(body, "sessions",
			[]any{sessionSummary(low), sessionSummary(low)}))
		spec.IDs = NoIDs()
		refuse(t, "repeated sessions", spec)
	})
	t.Run("peer.list.peers", func(t *testing.T) {
		admit(t, "sorted peers", specWithBody(t, CommandPeerList, map[string]any{
			"peers": []any{peerSummary(low), peerSummary(high)},
		}))
		refuse(t, "unsorted peers", specWithBody(t, CommandPeerList, map[string]any{
			"peers": []any{peerSummary(high), peerSummary(low)},
		}))
		refuse(t, "repeated peers", specWithBody(t, CommandPeerList, map[string]any{
			"peers": []any{peerSummary(low), peerSummary(low)},
		}))
	})
	t.Run("takeover.affected_session_ids", func(t *testing.T) {
		body := takeoverBody()
		spec := specWithBody(t, CommandTakeover, mutateBody(body, "affected_session_ids", []any{high, low}))
		spec.SessionKind = KindDirect
		spec.IDs = NoIDs().WithOperation(mustUUIDv7(t, fixtureOperationID))
		refuse(t, "unsorted affected sessions", spec)
		spec = specWithBody(t, CommandTakeover, mutateBody(body, "affected_session_ids", []any{}))
		spec.SessionKind = KindDirect
		spec.IDs = NoIDs().WithOperation(mustUUIDv7(t, fixtureOperationID))
		refuse(t, "an empty affected-session array", spec)
	})
	t.Run("sync.checkpoint_ids", func(t *testing.T) {
		body := validSpec(t, CommandSync).Body.(map[string]any)
		refuse(t, "unsorted checkpoint digests", specWithBody(t, CommandSync,
			mutateBody(body, "checkpoint_ids", []any{fixtureCheckpointID, fixtureOtherDigest})))
		admit(t, "sorted checkpoint digests", specWithBody(t, CommandSync,
			mutateBody(body, "checkpoint_ids", []any{fixtureOtherDigest, fixtureCheckpointID})))
		refuse(t, "an empty peer array", specWithBody(t, CommandSync,
			mutateBody(body, "peer_ids", []any{})))
	})
	t.Run("session summary warnings", func(t *testing.T) {
		summary := sessionSummary(fixtureSessionID)
		summary["warnings"] = []any{"b", "a"}
		refuse(t, "unsorted warnings", specWithBody(t, CommandStart, map[string]any{
			"session": summary, "execution_profile": "yolo", "terminal_backend": "tmux",
		}))
	})
}

// TestUnorderedArraysAreNotGivenAnInventedOrdering is the complement: PathDiff
// and CLIFinding carry no ID, so Section 14.2's sort rule does not reach them
// and this package must not impose one. A validator that sorted them anyway
// would refuse conforming documents.
func TestUnorderedArraysAreNotGivenAnInventedOrdering(t *testing.T) {
	body := validSpec(t, CommandDiff).Body.(map[string]any)
	entries := []any{
		map[string]any{"path": "z/last.go", "classification": "added",
			"source_digest": nil, "destination_digest": fixtureCheckpointID},
		map[string]any{"path": "a/first.go", "classification": "removed",
			"source_digest": fixtureCheckpointID, "destination_digest": nil},
	}
	admit(t, "path diffs in document order", specWithBody(t, CommandDiff, mutateBody(body, "entries", entries)))

	findings := []any{
		map[string]any{"severity": "warning", "code": "z_last", "message": "later",
			"remediation": nil, "source": "core"},
		map[string]any{"severity": "info", "code": "a_first", "message": "earlier",
			"remediation": "run ax doctor", "source": "mesh"},
	}
	admit(t, "findings in document order", specWithBody(t, CommandDoctor, map[string]any{
		"healthy": false, "findings": findings,
	}))
}

// TestClosedEnumsRefuseAPlausibleNeighbour narrows every closed vocabulary the
// Section 14.2 bodies declare. Section 17.2 rule 4 requires a reader to reject
// "an unknown ownership/security enum that would affect behavior", and each
// value below is a spelling a careless writer would reach for.
func TestClosedEnumsRefuseAPlausibleNeighbour(t *testing.T) {
	cases := []struct {
		command Command
		member  string
		value   string
	}{
		{CommandStart, "execution_profile", "standard_yolo"},
		{CommandStart, "terminal_backend", "screen"},
		{CommandAttach, "mode", "ssh"},
		{CommandTakeover, "mode", "forced"},
		{CommandTakeover, "workspace_mode", "whole-group"},
		{CommandTakeover, "state", "quiescing"},
		{CommandFork, "provider_fork_mode", "import"},
		{CommandResume, "terminal_backend", "tmux2"},
		{CommandDiff, "classification", "conflicting"},
		{CommandMaterialize, "mode", "replace"},
		{CommandSessionSetProfile, "new_profile", "YOLO"},
		{CommandPane, "result", "detached"},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.command)+"/"+testCase.member, func(t *testing.T) {
			base := validSpec(t, testCase.command).Body.(map[string]any)
			spec := specWithBody(t, testCase.command, mutateBody(base, testCase.member, testCase.value))
			refuse(t, testCase.value, spec)
		})
	}
}

// TestSessionSummaryNarrowsEveryDeclaredMember attacks the closed
// SessionSummary member by member.
func TestSessionSummaryNarrowsEveryDeclaredMember(t *testing.T) {
	start := func(summary map[string]any) Spec {
		spec := specWithBody(t, CommandStart, map[string]any{
			"session": summary, "execution_profile": "yolo", "terminal_backend": "tmux",
		})
		return spec
	}
	cases := []struct {
		name   string
		member string
		value  any
	}{
		{"an empty name", "name", ""},
		{"a 65 character name", "name", strings.Repeat("n", 65)},
		{"an unknown kind", "kind", "taskboard"},
		{"a provider id with an underscore", "provider_id", "task_board"},
		{"a UUIDv4 owner host", "owner_host_id", fixtureLeaseID},
		{"a zero lease epoch", "lease_epoch", json.Number("0")},
		{"a negative lease epoch", "lease_epoch", json.Number("-1")},
		{"an unsafe lease epoch", "lease_epoch", json.Number("9007199254740992")},
		{"a UUIDv7 lease id", "lease_id", fixtureSessionID},
		{"an unknown local role", "local_role", "replica_owner"},
		{"a retired session state", "state", "created"},
		{"another retired session state", "state", "quiesced"},
		{"a digest without its algorithm", "newest_checkpoint_id", strings.TrimPrefix(fixtureCheckpointID, "sha256:")},
		{"a timestamp without milliseconds", "newest_checkpoint_created_at", "2026-08-19T04:30:00Z"},
		{"an impossible date", "newest_checkpoint_created_at", "2026-02-30T04:30:00.000Z"},
		{"an unknown workspace status", "workspace_status", "dirty"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			summary := sessionSummary(fixtureSessionID)
			summary[testCase.member] = testCase.value
			refuse(t, testCase.name, start(summary))
		})
	}
	t.Run("an exactly 64 character name", func(t *testing.T) {
		summary := sessionSummary(fixtureSessionID)
		summary["name"] = strings.Repeat("n", 64)
		admit(t, "a name at the bound", start(summary))
	})
	t.Run("stopped with a null checkpoint", func(t *testing.T) {
		summary := sessionSummary(fixtureSessionID)
		summary["state"] = "stopped"
		summary["newest_checkpoint_id"] = nil
		refuse(t, "a stopped session with no checkpoint", start(summary))
	})
}

// TestCapabilityMapNarrowsItsVocabularyAndBound proves the capability map is
// checked against the closed Section 8.3 provider vocabulary and the Section
// 14.2 [0..7] bound, and that the vocabulary is the reviewed seven names.
func TestCapabilityMapNarrowsItsVocabularyAndBound(t *testing.T) {
	reviewed := []string{
		"appserver", "managed_pty", "native_goal_binding", "native_resume",
		"portable_store", "prompt_spawn", "task_board_primary",
	}
	names := CapabilityNames()
	if len(names) != len(reviewed) {
		t.Fatalf("capability vocabulary = %v, want the seven Section 8.3 provider names", names)
	}
	for index, name := range reviewed {
		if names[index] != name {
			t.Fatalf("capability %d = %q, want %q", index, names[index], name)
		}
	}
	start := func(summary map[string]any) Spec {
		return specWithBody(t, CommandStart, map[string]any{
			"session": summary, "execution_profile": "yolo", "terminal_backend": "tmux",
		})
	}
	full := map[string]any{}
	for _, name := range reviewed {
		full[name] = map[string]any{"status": "unknown", "enabled": false, "detail": ""}
	}
	summary := sessionSummary(fixtureSessionID)
	summary["capabilities"] = full
	admit(t, "all seven capability names", start(summary))

	// A Session Adapter and a TerminalBackend capability name are real AX
	// vocabulary from other families, and neither belongs in this map.
	for _, foreign := range []string{"native_discovery", "durable_disconnect", "task_board", ""} {
		summary = sessionSummary(fixtureSessionID)
		summary["capabilities"] = map[string]any{
			foreign: map[string]any{"status": "unknown", "enabled": false, "detail": ""},
		}
		refuse(t, "the foreign capability name "+foreign, start(summary))
	}
	summary = sessionSummary(fixtureSessionID)
	eight := map[string]any{}
	for name, value := range full {
		eight[name] = value
	}
	eight["native_discovery"] = map[string]any{"status": "unknown", "enabled": false, "detail": ""}
	summary["capabilities"] = eight
	refuse(t, "eight capability entries", start(summary))

	summary = sessionSummary(fixtureSessionID)
	summary["capabilities"] = map[string]any{
		"native_resume": map[string]any{"status": "available", "enabled": true},
	}
	refuse(t, "a capability summary without detail", start(summary))
	summary = sessionSummary(fixtureSessionID)
	summary["capabilities"] = map[string]any{
		"native_resume": map[string]any{
			"status": "available", "enabled": true, "detail": "", "evidence": "self-minted",
		},
	}
	refuse(t, "a capability summary with an extra member", start(summary))
	summary = sessionSummary(fixtureSessionID)
	summary["capabilities"] = map[string]any{
		"native_resume": map[string]any{"status": "supported", "enabled": true, "detail": ""},
	}
	refuse(t, "an unknown capability status", start(summary))
}

// TestPeerProbeContractMapRefusesTheContractsSection112Excludes narrows the
// contract-name vocabulary. Section 11.2 states that "Structured Error,
// Observation Event, and CLI Result MUST NOT appear in this map", so each of
// those names is refused on its own row rather than by a single blanket case.
func TestPeerProbeContractMapRefusesTheContractsSection112Excludes(t *testing.T) {
	reviewed := []string{
		"blob", "checkpoint", "chunk", "lease", "materialization_plan", "provider_identity",
		"rpc", "session_event", "session_record", "task_board_bundle", "tombstone",
		"tombstone_ack", "transfer_manifest", "workspace_group",
	}
	names := ContractNames()
	if len(names) != 14 {
		t.Fatalf("contract vocabulary = %v, want the fourteen Section 11.2 keys", names)
	}
	for index, name := range reviewed {
		if names[index] != name {
			t.Fatalf("contract %d = %q, want %q", index, names[index], name)
		}
	}
	base := validSpec(t, CommandPeerProbe).Body.(map[string]any)
	full := map[string]any{}
	for _, name := range reviewed {
		full[name] = []any{"1.0.0"}
	}
	admit(t, "the fourteen hello contracts", specWithBody(t, CommandPeerProbe, mutateBody(base, "contracts", full)))

	excluded := []string{
		"error", "cli_result", "observation", "configuration", "provider",
		"provider_manifest", "task_board_bridge", "materialization_recovery_state",
	}
	for _, name := range excluded {
		t.Run(name, func(t *testing.T) {
			refuse(t, "the excluded contract "+name, specWithBody(t, CommandPeerProbe,
				mutateBody(base, "contracts", map[string]any{name: []any{"1.0.0"}})))
		})
	}
	refuse(t, "an unsorted version array", specWithBody(t, CommandPeerProbe,
		mutateBody(base, "contracts", map[string]any{"rpc": []any{"2.0.0", "1.0.0"}})))
	refuse(t, "an empty version array", specWithBody(t, CommandPeerProbe,
		mutateBody(base, "contracts", map[string]any{"rpc": []any{}})))
	refuse(t, "a non-semver version", specWithBody(t, CommandPeerProbe,
		mutateBody(base, "contracts", map[string]any{"rpc": []any{"2.0"}})))
}

// TestLogsBodyBindsEveryEventToItsEmitter narrows the Section 14.1 sentence
// "every returned Observation Event MUST have host_id equal to that emitter",
// and proves each element is validated as a complete Section 18.1 object.
func TestLogsBodyBindsEveryEventToItsEmitter(t *testing.T) {
	base := validSpec(t, CommandLogs).Body.(map[string]any)
	admit(t, "an event from the emitter", specWithBody(t, CommandLogs, base))

	refuse(t, "an event from another host", specWithBody(t, CommandLogs,
		mutateBody(base, "events", []any{observationEvent(fixtureOtherHostID)})))

	foreign := observationEvent(fixtureSourceHostID)
	delete(foreign, "schema_version")
	refuse(t, "an event missing its schema_version", specWithBody(t, CommandLogs,
		mutateBody(base, "events", []any{foreign})))

	unknownMember := observationEvent(fixtureSourceHostID)
	unknownMember["authority"] = "forged"
	refuse(t, "an event with an unknown member", specWithBody(t, CommandLogs,
		mutateBody(base, "events", []any{unknownMember})))

	refuse(t, "an event that is not an object", specWithBody(t, CommandLogs,
		mutateBody(base, "events", []any{"not an event"})))

	admit(t, "an empty event array", specWithBody(t, CommandLogs, mutateBody(base, "events", []any{})))
	refuse(t, "an empty cursor", specWithBody(t, CommandLogs, mutateBody(base, "next_cursor", "")))
	admit(t, "an opaque cursor", specWithBody(t, CommandLogs, mutateBody(base, "next_cursor", "opaque-token")))
}

// TestAbsolutePathMemberIsCheckedAndNarrowableToAPlatform proves the stated
// bound on an absolute-path member in an object that names no platform, and
// that the narrowing hook actually narrows.
func TestAbsolutePathMemberIsCheckedAndNarrowableToAPlatform(t *testing.T) {
	base := materializationSummary()
	for _, path := range []string{"work/payments", "", "./work", "/work/../etc", "/work/./api"} {
		refuse(t, "the non-absolute path "+path, specWithBody(t, CommandMaterialize,
			mutateBody(base, "destination_path", path)))
	}
	windows := mustResult(t, specWithBody(t, CommandMaterialize,
		mutateBody(base, "destination_path", `C:\Users\operator\work`)))
	if err := windows.VerifyDestinationPlatform("windows"); err != nil {
		t.Fatalf("VerifyDestinationPlatform(windows): %v", err)
	}
	if err := windows.VerifyDestinationPlatform("macos"); err == nil {
		t.Fatalf("VerifyDestinationPlatform(macos) admitted a Windows path")
	}
	posix := mustResult(t, specWithBody(t, CommandMaterialize, base))
	if err := posix.VerifyDestinationPlatform("linux"); err != nil {
		t.Fatalf("VerifyDestinationPlatform(linux): %v", err)
	}
	if err := posix.VerifyDestinationPlatform("windows"); err == nil {
		t.Fatalf("VerifyDestinationPlatform(windows) admitted a POSIX path")
	}
	if err := posix.VerifyDestinationPlatform("plan9"); err == nil {
		t.Fatalf("VerifyDestinationPlatform admitted an unknown platform")
	}
	if err := mustResult(t, validSpec(t, CommandCancel)).VerifyDestinationPlatform("linux"); err == nil {
		t.Fatalf("VerifyDestinationPlatform admitted a command with no destination path")
	}
}

// TestCancelRequiresTheCancelledLiteral pins the Section 14.2 row "with true
// required".
func TestCancelRequiresTheCancelledLiteral(t *testing.T) {
	refuse(t, "cancelled=false", specWithBody(t, CommandCancel, map[string]any{
		"name": "payments-api", "cancelled": false,
	}))
	refuse(t, "an empty name", specWithBody(t, CommandCancel, map[string]any{
		"name": "", "cancelled": true,
	}))
}

// TestStatusSyncMapNarrowsItsKeyGrammarAndValueType checks
// map(UUIDv7,timestamp)[0..1024].
func TestStatusSyncMapNarrowsItsKeyGrammarAndValueType(t *testing.T) {
	base := validSpec(t, CommandStatus).Body.(map[string]any)
	refuse(t, "a non-UUID sync key", specWithBody(t, CommandStatus,
		mutateBody(base, "last_successful_sync", map[string]any{"laptop": fixtureTimestamp})))
	refuse(t, "a UUIDv4 sync key", specWithBody(t, CommandStatus,
		mutateBody(base, "last_successful_sync", map[string]any{fixtureLeaseID: fixtureTimestamp})))
	refuse(t, "a null sync value", specWithBody(t, CommandStatus,
		mutateBody(base, "last_successful_sync", map[string]any{fixtureSourceHostID: nil})))
	refuse(t, "a non-timestamp sync value", specWithBody(t, CommandStatus,
		mutateBody(base, "last_successful_sync", map[string]any{fixtureSourceHostID: "yesterday"})))
	admit(t, "an empty sync map", specWithBody(t, CommandStatus,
		mutateBody(base, "last_successful_sync", map[string]any{})))
}

// TestProviderExitCodeIsAnInt32OrNull narrows the one signed numeric member the
// Section 14.2 bodies declare.
func TestProviderExitCodeIsAnInt32OrNull(t *testing.T) {
	base := validSpec(t, CommandAttach).Body.(map[string]any)
	admit(t, "a null exit code", specWithBody(t, CommandAttach, base))
	admit(t, "a negative exit code", specWithBody(t, CommandAttach,
		mutateBody(base, "provider_exit_code", json.Number("-9"))))
	admit(t, "the int32 maximum", specWithBody(t, CommandAttach,
		mutateBody(base, "provider_exit_code", json.Number("2147483647"))))
	refuse(t, "one past the int32 maximum", specWithBody(t, CommandAttach,
		mutateBody(base, "provider_exit_code", json.Number("2147483648"))))
	refuse(t, "a stringly typed exit code", specWithBody(t, CommandAttach,
		mutateBody(base, "provider_exit_code", "9")))
}

// TestCLIFindingNarrowsItsDeclaredBounds attacks the closed finding shape.
func TestCLIFindingNarrowsItsDeclaredBounds(t *testing.T) {
	finding := func(overrides map[string]any) map[string]any {
		object := map[string]any{
			"severity": "warning", "code": "tmux_missing", "message": "tmux is not installed",
			"remediation": "brew install tmux", "source": "terminal",
		}
		for key, value := range overrides {
			object[key] = value
		}
		return map[string]any{"healthy": false, "findings": []any{object}}
	}
	admit(t, "a complete finding", specWithBody(t, CommandDoctor, finding(nil)))
	admit(t, "a finding without a remediation", specWithBody(t, CommandDoctor,
		finding(map[string]any{"remediation": nil})))
	refuse(t, "an empty remediation", specWithBody(t, CommandDoctor,
		finding(map[string]any{"remediation": ""})))
	refuse(t, "an unknown severity", specWithBody(t, CommandDoctor,
		finding(map[string]any{"severity": "critical"})))
	refuse(t, "an unknown source", specWithBody(t, CommandDoctor,
		finding(map[string]any{"source": "terminal_backend"})))
	refuse(t, "an empty code", specWithBody(t, CommandDoctor, finding(map[string]any{"code": ""})))
	refuse(t, "a 129 character code", specWithBody(t, CommandDoctor,
		finding(map[string]any{"code": strings.Repeat("c", 129)})))
	admit(t, "a 128 character code", specWithBody(t, CommandDoctor,
		finding(map[string]any{"code": strings.Repeat("c", 128)})))
}
