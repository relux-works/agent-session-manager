package canonicaljson

import (
	"errors"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// boundProof proves one declared range bound at both of its limits through a
// production entry point.
//
// build(size) returns a candidate whose bounded member has exactly `size`
// characters, entries, or keys. The runner accepts build(minimum) and
// build(maximum) and refuses build(minimum-1) and build(maximum+1).
//
// minimum and maximum are the numbers written in the pinned SPEC declaration
// quoted in spec, never the implementation constants they are compared against:
// an assertion against the implementation's own constant cannot fail. The key
// carries the implementation's literal bounds, so if the two ever diverge the
// coverage assertion in declared_bounds_test.go reddens before any case runs.
type boundProof struct {
	key       string
	spec      string
	selfField SelfField // empty selects the Observation Event entry point
	minimum   int       // -1 when the helper declares no minimum
	maximum   int       // -1 when the helper declares no maximum
	build     func(t *testing.T, size int) map[string]any
}

// boundSubsumption records a declared bound whose at-limit acceptance is
// unreachable because an earlier production refusal always fires first. The
// subsuming refusal is named here and pinned by the named test, so "subsumed"
// cannot be used to wave a bound through. Widening such a bound is still caught:
// the derived obligation key carries the literal bound.
type boundSubsumption struct {
	subsumingRefusal string
	provingTest      string
}

// subsumedBoundProofs is asserted exactly by the coverage test, and every named
// test must exist in this package.
var subsumedBoundProofs = map[boundObligation]boundSubsumption{
	{key: "validateGitSubmodule|requireArray|submodules|-..256", direction: boundMaximum}: {
		subsumingRefusal: "GitSubmodule tree exceeds maximum total count 256",
		provingTest:      "TestNestedSubmoduleArrayBoundIsSubsumedByTheWholeTreeCount",
	},
}

// collectBoundProofClaims returns every obligation discharged by a proof that
// actually runs: the core-record table below, the declared boundary-constraint
// table, and the pinned subsumptions.
func collectBoundProofClaims(t *testing.T) []boundObligation {
	t.Helper()

	var claims []boundObligation
	for _, proof := range coreRecordBoundProofs() {
		if proof.maximum >= 0 {
			claims = append(claims, boundObligation{key: proof.key, direction: boundMaximum})
		}
		if proof.minimum >= 1 {
			claims = append(claims, boundObligation{key: proof.key, direction: boundMinimum})
		}
	}
	for _, boundary := range declaredBoundaryConstraintCases() {
		claims = append(claims, boundary.claims...)
	}
	for obligation := range subsumedBoundProofs {
		claims = append(claims, obligation)
	}
	return claims
}

// TestSubsumedBoundProofsNameATestThatExists keeps the subsumption escape hatch
// honest the same way the delegation one is kept honest.
func TestSubsumedBoundProofsNameATestThatExists(t *testing.T) {
	t.Parallel()

	names := packageTestFunctionNames(t)
	for obligation, subsumption := range subsumedBoundProofs {
		if _, ok := names[subsumption.provingTest]; !ok {
			t.Errorf("subsumed bound %s names %s, which does not exist in this package", obligation, subsumption.provingTest)
		}
	}
}

// TestEveryCoreRecordDeclaredBoundAcceptsAtItsLimitAndRefusesPastIt runs the
// table. Every case drives CalculateObjectIdentity and VerifyObjectIdentity, or
// ValidateObservationEvent for the Observation Event bounds, and asserts the
// refusal names the bound rather than any earlier disjunct.
func TestEveryCoreRecordDeclaredBoundAcceptsAtItsLimitAndRefusesPastIt(t *testing.T) {
	for _, proof := range coreRecordBoundProofs() {
		t.Run(proof.key, func(t *testing.T) {
			helper := boundHelperFromKey(t, proof.key)
			member := boundMemberFromKey(t, proof.key)
			if proof.maximum >= 0 {
				assertBoundAccepts(t, proof, proof.maximum)
				assertBoundRefuses(t, proof, proof.maximum+1,
					boundRefusalText(helper, member, proof.minimum, proof.maximum, boundMaximum))
			}
			if proof.minimum >= 1 {
				assertBoundAccepts(t, proof, proof.minimum)
				assertBoundRefuses(t, proof, proof.minimum-1,
					boundRefusalText(helper, member, proof.minimum, proof.maximum, boundMinimum))
			}
		})
	}
}

func assertBoundAccepts(t *testing.T, proof boundProof, size int) {
	t.Helper()
	candidate := mustJSON(t, proof.build(t, size))
	if proof.selfField == "" {
		if err := ValidateObservationEvent(candidate); err != nil {
			t.Fatalf("ValidateObservationEvent(%s at %d) error = %v, want acceptance", proof.key, size, err)
		}
		return
	}
	assertIdentityEntriesAcceptShape(t, candidate, proof.selfField)
}

func assertBoundRefuses(t *testing.T, proof boundProof, size int, want string) {
	t.Helper()
	candidate := mustJSON(t, proof.build(t, size))
	if proof.selfField == "" {
		err := ValidateObservationEvent(candidate)
		if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateObservationEvent(%s at %d) error = %v, want observation refusal containing %q", proof.key, size, err, want)
		}
		return
	}
	assertIdentityEntriesRefuseWithReason(t, candidate, proof.selfField, want)
}

// boundRefusalText builds the refusal an out-of-range value must produce, from
// the pinned SPEC numbers carried by the proof rather than from the production
// constants, so a widened bound cannot satisfy its own assertion.
func boundRefusalText(helper, member string, minimum, maximum int, direction boundDirection) string {
	switch helper {
	case "requireBoundedString", "bounded", "requireBoundedLogicalIdentity", "requirePrintableBoundedString":
		if direction == boundMinimum && minimum == 1 {
			// A declared minimum of one is exactly "non-empty", and these
			// helpers read the member through requireString, which refuses the
			// empty string before the range comparison. The subsuming refusal
			// is named here rather than left as an unreachable branch, and the
			// case still proves that minimum-1 is rejected at the entry point.
			return fmt.Sprintf("member %s must be a non-empty UTF-8 string", member)
		}
		return fmt.Sprintf("member %s must contain %d..%d Unicode characters", member, minimum, maximum)
	case "requirePrintableByteBoundedString":
		// This helper reads through requireUTF8String, which admits the empty
		// string, so its own byte-count refusal is the first gate in both
		// directions.
		return fmt.Sprintf("member %s must contain %d..%d UTF-8 bytes", member, minimum, maximum)
	case "nullableBoundedString":
		if direction == boundMinimum && minimum == 1 {
			// Subsumed by nullableString for the same reason as above.
			return fmt.Sprintf("member %s must be null or a non-empty UTF-8 string", member)
		}
		return fmt.Sprintf("member %s must be null or contain %d..%d Unicode characters", member, minimum, maximum)
	case "requireArrayRange":
		if direction == boundMinimum {
			return fmt.Sprintf("member %s requires at least %d entries", member, minimum)
		}
		return fmt.Sprintf("member %s exceeds maximum length %d", member, maximum)
	case "requireArrayMinimum":
		return fmt.Sprintf("member %s requires at least %d entries", member, minimum)
	case "requireArray", "requireSortedUniquePaths":
		return fmt.Sprintf("member %s exceeds maximum length %d", member, maximum)
	}
	return "member " + member
}

func boundHelperFromKey(t *testing.T, key string) string {
	t.Helper()
	parts := strings.Split(key, "|")
	if len(parts) != 4 {
		t.Fatalf("malformed bound key %q", key)
	}
	return parts[1]
}

func boundMemberFromKey(t *testing.T, key string) string {
	t.Helper()
	parts := strings.Split(key, "|")
	if len(parts) != 4 {
		t.Fatalf("malformed bound key %q", key)
	}
	return parts[2]
}

// packageTestFunctionNames derives every test function declared in this
// package's test sources, so a delegated or subsumed obligation cannot name a
// test that does not exist.
func packageTestFunctionNames(t *testing.T) map[string]struct{} {
	t.Helper()

	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve declared bounds test source path")
	}
	directory := filepath.Dir(source)
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed := parseProductionFile(t, filepath.Join(directory, entry.Name()))
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			names[function.Name.Name] = struct{}{}
		}
	}
	if len(names) == 0 {
		t.Fatal("derived zero test functions for the canonicaljson package")
	}
	return names
}

// ---------------------------------------------------------------------------
// Fixture builders
// ---------------------------------------------------------------------------

func sizedString(size int) any { return strings.Repeat("x", size) }

// lowestSessionEventVersion derives the earliest contract version that carries
// an event type from the pinned catalog rather than hard-coding the version
// groups, so a catalog change moves these proofs with it.
func lowestSessionEventVersion(t *testing.T, eventType string) string {
	t.Helper()
	best := ""
	for _, event := range catalog.Current().Events {
		if event.Family != "session_event" || string(event.Name) != eventType {
			continue
		}
		for _, version := range event.ContractVersions {
			if best == "" || version < best {
				best = version
			}
		}
	}
	if best == "" {
		t.Fatalf("catalog declares no contract version for session event %q", eventType)
	}
	return best
}

func sessionEventPayloadBound(eventType, member string, value func(int) any) func(*testing.T, int) map[string]any {
	return func(t *testing.T, size int) map[string]any {
		t.Helper()
		object := validSessionEventObject(lowestSessionEventVersion(t, eventType), eventType)
		object["payload"].(map[string]any)[member] = value(size)
		return object
	}
}

func sessionEventPayloadBoundAtVersion(version, eventType, member string, value func(int) any) func(*testing.T, int) map[string]any {
	return func(t *testing.T, size int) map[string]any {
		t.Helper()
		object := validSessionEventObject(version, eventType)
		object["payload"].(map[string]any)[member] = value(size)
		return object
	}
}

func observationBound(member string, value func(int) any, prepare func(map[string]any)) func(*testing.T, int) map[string]any {
	return func(t *testing.T, size int) map[string]any {
		t.Helper()
		object := validObservationEventObject()
		if prepare != nil {
			prepare(object)
		}
		object[member] = value(size)
		return object
	}
}

func sizedObservationName(size int) any {
	// The declared grammar is [a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}, so the
	// shortest admitted name is three characters. Below that the length refusal
	// is the first gate reached, which is exactly what the minimum proof asserts.
	if size < 3 {
		return strings.Repeat("a", size)
	}
	return "a." + strings.Repeat("b", size-2)
}

func sizedRelativePaths(size int) any {
	values := make([]any, size)
	for index := range size {
		values[index] = fmt.Sprintf("config-%04d.md", index)
	}
	return values
}

func workspaceGroupMemberBound(index int, member string, value func(int) any) func(*testing.T, int) map[string]any {
	return func(t *testing.T, size int) map[string]any {
		t.Helper()
		object := validWorkspaceGroupRecordObject()
		object["members"].([]any)[index].(map[string]any)[member] = value(size)
		return object
	}
}

func acceptedRisksOfSize(size int) any {
	declared := []any{"divergent_history", "split_brain", "stale_process"}
	if size <= len(declared) {
		return declared[:size]
	}
	values := append([]any(nil), declared...)
	for index := len(declared); index < size; index++ {
		values = append(values, fmt.Sprintf("extra_risk_%d", index))
	}
	return values
}

// ---------------------------------------------------------------------------
// The proofs
// ---------------------------------------------------------------------------

func coreRecordBoundProofs() []boundProof {
	return []boundProof{
		{
			key: "terminal.created|bounded|terminal_id|1..512", spec: "terminal_id:string[1..512]",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: sessionEventPayloadBound("terminal.created", "terminal_id", sizedString),
		},
		{
			key: "provider.launched|bounded|provider_version|1..128", spec: "provider_version:string[1..128]",
			selfField: SelfEventID, minimum: 1, maximum: 128,
			build: sessionEventPayloadBound("provider.launched", "provider_version", sizedString),
		},
		{
			key: "provider.launched|bounded|profile_mapping|1..512", spec: "profile_mapping:string[1..512]",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: sessionEventPayloadBound("provider.launched", "profile_mapping", sizedString),
		},
		{
			key: "session.idle|bounded|boundary_ref|1..1024", spec: "boundary_ref:string[1..1024]",
			selfField: SelfEventID, minimum: 1, maximum: 1024,
			build: sessionEventPayloadBound("session.idle", "boundary_ref", sizedString),
		},
		{
			key: "session.resumed|bounded|native_session_id|1..512", spec: "native_session_id:string[1..512]",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: sessionEventPayloadBound("session.resumed", "native_session_id", sizedString),
		},
		{
			key: "session.failed|bounded|error_code|1..128", spec: "error_code:string[1..128]",
			selfField: SelfEventID, minimum: 1, maximum: 128,
			build: sessionEventPayloadBound("session.failed", "error_code", sizedString),
		},
		{
			key: "tombstone.issued|bounded|target_ref|1..1024", spec: "target_ref:string[1..1024]",
			selfField: SelfEventID, minimum: 1, maximum: 1024,
			build: sessionEventPayloadBound("tombstone.issued", "target_ref", sizedString),
		},
		{
			key: "clone.target_validation_failed|bounded|error_code|1..128", spec: "error_code:string[1..128]",
			selfField: SelfEventID, minimum: 1, maximum: 128,
			build: sessionEventPayloadBound("clone.target_validation_failed", "error_code", sizedString),
		},
		{
			key: "clone.rolled_back|bounded|reason_code|1..128", spec: "reason_code:string[1..128]",
			selfField: SelfEventID, minimum: 1, maximum: 128,
			build: sessionEventPayloadBound("clone.rolled_back", "reason_code", sizedString),
		},
		{
			key: "move.source_release_failed|bounded|error_code|1..128", spec: "error_code:string[1..128]",
			selfField: SelfEventID, minimum: 1, maximum: 128,
			build: sessionEventPayloadBound("move.source_release_failed", "error_code", sizedString),
		},
		{
			key:       "validateClonePlannedPayload|bounded|expected_target_native_session_id|1..512",
			spec:      "expected_target_native_session_id:string[1..512]",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: sessionEventPayloadBound("clone.planned", "expected_target_native_session_id", sizedString),
		},
		{
			key: "validateTombstoneResolvedPayload|bounded|target_ref|1..1024", spec: "target_ref:string[1..1024]",
			selfField: SelfEventID, minimum: 1, maximum: 1024,
			build: sessionEventPayloadBound("tombstone.resolved", "target_ref", sizedString),
		},
		{
			key:       "validateTaskBoardLaunchedPayload|bounded|manager_session_ref|1..512",
			spec:      "manager_session_ref:string[1..512]",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: sessionEventPayloadBound("task_board.launched", "manager_session_ref", sizedString),
		},
		{
			key:       "validateTaskBoardAdoptedPayload|bounded|manager_session_ref|1..512",
			spec:      "manager_session_ref:string[1..512]",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: sessionEventPayloadBound("task_board.adopted", "manager_session_ref", sizedString),
		},
		{
			key:       "validateGoalPair|nullableBoundedString|board_goal_id|1..128",
			spec:      "board_goal_id:string[1..128]|null",
			selfField: SelfEventID, minimum: 1, maximum: 128,
			build: sessionEventPayloadBound("task_board.launched", "board_goal_id", sizedString),
		},
		{
			key:       "validateBootstrapAbortedPayload|nullableBoundedString|manager_session_ref|1..512",
			spec:      "manager_session_ref:string[1..512]|null",
			selfField: SelfEventID, minimum: 1, maximum: 512,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validSessionEventObject(lowestSessionEventVersion(t, "session.bootstrap_aborted"), "session.bootstrap_aborted")
				payload := object["payload"].(map[string]any)
				// The after-identity branch admits exactly one identity field;
				// clearing the record digest isolates the length refusal from
				// the pairing refusal.
				payload["provider_identity_record_id"] = nil
				payload["manager_session_ref"] = sizedString(size)
				return object
			},
		},
		{
			key:       "validateSyncCompletedPayload|requireArrayRange|manifest_ids|1..1024",
			spec:      "manifest_ids:sorted unique digest[1..1024]",
			selfField: SelfEventID, minimum: 1, maximum: 1024,
			build: sessionEventPayloadBound("sync.completed", "manifest_ids", func(size int) any { return numberedDigests(size) }),
		},
		{
			key:       "validateForceConfirmedPayload|requireArrayRange|accepted_risks|3..3",
			spec:      "accepted_risks:sorted unique divergent_history|split_brain|stale_process[3..3]",
			selfField: SelfEventID, minimum: 3, maximum: 3,
			build: sessionEventPayloadBound("takeover.force_confirmed", "accepted_risks", acceptedRisksOfSize),
		},
		{
			key:       "validateTerminalV4Payload|requireArrayRange|evidence_ids|1..256",
			spec:      "evidence_ids:sorted unique digest[1..256]",
			selfField: SelfEventID, minimum: 1, maximum: 256,
			build: sessionEventPayloadBoundAtVersion("4.0.0", "terminal.created", "evidence_ids",
				func(size int) any { return numberedDigests(size) }),
		},
		{
			key:       "validateSessionEvent|requireArrayMinimum|predecessors|1..-",
			spec:      "predecessors as a sorted array of one or more record/event digests",
			selfField: SelfEventID, minimum: 1, maximum: -1,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validSessionEventObject(lowestSessionEventVersion(t, "session.created"), "session.created")
				object["predecessors"] = numberedDigests(size)
				return object
			},
		},
		{
			key:       "validateCheckpointRecord|requireArrayRange|event_heads|1..64",
			spec:      "event_heads | sorted unique digest[1..64]",
			selfField: SelfCheckpointID, minimum: 1, maximum: 64,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validCheckpointRecordObject(true)
				object["event_heads"] = numberedDigests(size)
				return object
			},
		},
		{
			key:       "validateSafeBoundaryEvidence|requireBoundedString|provider_version|1..128",
			spec:      "provider_version | string[1..128] | Exact probed version",
			selfField: SelfCheckpointID, minimum: 1, maximum: 128,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validCheckpointRecordObject(true)
				object["safe_boundary"].(map[string]any)["provider_version"] = sizedString(size)
				return object
			},
		},
		{
			key:       "validateProviderIdentityRecord|requireBoundedString|provider_version|1..128",
			spec:      "provider_version | string[1..128] | Exact probed version",
			selfField: SelfRecordID, minimum: 1, maximum: 128,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validProviderIdentityRecordObject()
				object["provider_version"] = sizedString(size)
				return object
			},
		},
		{
			key:       "validateProviderIdentityRecord|requireBoundedString|provider_version_range|1..256",
			spec:      "provider_version_range | string[1..256] | Adapter compatibility range used for this identity",
			selfField: SelfRecordID, minimum: 1, maximum: 256,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validProviderIdentityRecordObject()
				object["provider_version_range"] = sizedString(size)
				return object
			},
		},
		{
			key:       "validateProviderIdentityRecord|requireBoundedString|native_session_id|1..512",
			spec:      "native_session_id:string[1..512]",
			selfField: SelfRecordID, minimum: 1, maximum: 512,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validProviderIdentityRecordObject()
				object["native_session_id"] = sizedString(size)
				return object
			},
		},
		{
			key:       "validateWorkspaceGroupRecord|requireBoundedString|display_name|1..128",
			spec:      "display_name is 1-128 UTF-8 characters",
			selfField: SelfRecordID, minimum: 1, maximum: 128,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validWorkspaceGroupRecordObject()
				object["display_name"] = sizedString(size)
				return object
			},
		},
		{
			key:       "validateWorkspaceGroupRecord|requireArrayRange|members|1..256",
			spec:      "members:WorkspaceMember[1..256]",
			selfField: SelfRecordID, minimum: 1, maximum: 256,
			build: func(t *testing.T, size int) map[string]any {
				t.Helper()
				object := validWorkspaceGroupRecordObject()
				object["members"] = numberedManagedWorkspaceMembers(size)
				return object
			},
		},
		{
			key:       "validateWorkspaceMember|requireArrayRange|sanitized_remote_urls|1..16",
			spec:      "sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]",
			selfField: SelfRecordID, minimum: 1, maximum: 16,
			build: workspaceGroupMemberBound(0, "sanitized_remote_urls", func(size int) any { return numberedSanitizedURLs(size) }),
		},
		{
			key:       "validateWorkspaceMember|requireBoundedLogicalIdentity|repository_identity|1..256",
			spec:      "repository_identity:string[1..256]",
			selfField: SelfRecordID, minimum: 1, maximum: 256,
			build: workspaceGroupMemberBound(0, "repository_identity", sizedString),
		},
		{
			key:       "validateWorkspaceMember|requireBoundedLogicalIdentity|tree_identity|1..256",
			spec:      "tree_identity:string[1..256]",
			selfField: SelfRecordID, minimum: 1, maximum: 256,
			build: workspaceGroupMemberBound(1, "tree_identity", sizedString),
		},
		{
			key:       "validateWorkspaceMember|requireSortedUniquePaths|agent_project_config_paths|-..256",
			spec:      "agent_project_config_paths:sorted unique path[0..256]",
			selfField: SelfRecordID, minimum: -1, maximum: 256,
			build: workspaceGroupMemberBound(0, "agent_project_config_paths", sizedRelativePaths),
		},
		{
			key:     "validateObservationEvent|requireBoundedString|event|3..128",
			spec:    "event | observation-name | [a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*){1,7}, 3-128 characters",
			minimum: 3, maximum: 128,
			build: observationBound("event", sizedObservationName, nil),
		},
		{
			key:     "validateObservationEvent|nullableBoundedString|phase|1..128",
			spec:    "phase | string[1..128] or null | Stable lower-snake-case phase or null",
			minimum: 1, maximum: 128,
			build: observationBound("phase", func(size int) any { return strings.Repeat("a", size) }, nil),
		},
		{
			key:     "validateObservationEvent|nullableBoundedString|error_code|1..128",
			spec:    "error_code | string[1..128] or null",
			minimum: 1, maximum: 128,
			build: observationBound("error_code", sizedString, func(object map[string]any) {
				// error_code is non-null only for partial and failure results.
				object["result"] = "failure"
			}),
		},
		{
			key:     "validateObservationEvent|requireArrayRange|object_ids|0..4096",
			spec:    "object_ids | sorted unique digest[0..4096] | Redacted object identities only",
			minimum: 0, maximum: 4096,
			build: observationBound("object_ids", func(size int) any { return numberedDigests(size) }, nil),
		},
	}
}

// TestNestedSubmoduleArrayBoundIsSubsumedByTheWholeTreeCount pins the one
// declared bound in the package whose at-limit acceptance is unreachable.
//
// validateGitSubmodule bounds a nested submodules array at 256 entries, but
// validateGitSubmodule also counts every submodule in the whole tree against a
// maximum of 256. A nested list of 256 sits under a parent that already
// consumed one, so the whole-tree refusal always fires first and the declared
// array bound can never be reached from a valid candidate. This test proves
// that ordering rather than assuming it: 255 nested submodules are accepted,
// 256 are refused, and the refusal names the subsuming whole-tree count.
func TestNestedSubmoduleArrayBoundIsSubsumedByTheWholeTreeCount(t *testing.T) {
	assertIdentityEntriesAcceptShape(t, mustJSON(t, gitWorkspaceWithNestedSubmoduleCount(255)), SelfManifestID)
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, gitWorkspaceWithNestedSubmoduleCount(256)),
		SelfManifestID, subsumedBoundProofs[boundObligation{
			key:       "validateGitSubmodule|requireArray|submodules|-..256",
			direction: boundMaximum,
		}].subsumingRefusal)
}

// TestDeclaredBoundProofsQuoteAPinnedSpecDeclaration keeps the table honest
// about provenance: every proof must carry a non-empty pinned SPEC quote and a
// key whose literal bounds agree with the numbers the proof asserts.
func TestDeclaredBoundProofsQuoteAPinnedSpecDeclaration(t *testing.T) {
	t.Parallel()

	for _, proof := range coreRecordBoundProofs() {
		if strings.TrimSpace(proof.spec) == "" {
			t.Errorf("bound proof %s has no pinned SPEC declaration", proof.key)
		}
		parts := strings.Split(proof.key, "|")
		if len(parts) != 4 {
			t.Fatalf("malformed bound key %q", proof.key)
		}
		bounds := strings.SplitN(parts[3], "..", 2)
		if got := boundNumber(bounds[0]); got != proof.minimum {
			t.Errorf("bound proof %s asserts minimum %d, call site declares %d", proof.key, proof.minimum, got)
		}
		if got := boundNumber(bounds[1]); got != proof.maximum {
			t.Errorf("bound proof %s asserts maximum %d, call site declares %d", proof.key, proof.maximum, got)
		}
	}
}

func boundNumber(text string) int {
	value, err := strconv.Atoi(text)
	if err != nil {
		return -1
	}
	return value
}
