package canonicaljson

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// This file carries the named negative cases for the Section 5, 10.1 and 18.1
// refusals that no derived sweep can reach: cross-field couplings, ordering and
// uniqueness, closed vocabularies, and literal value gates. Each case supplies a
// COMPLETE valid member set and violates exactly one clause, because
// requireExactMembers runs first in every validator and any omission stops
// there.
//
// TestEveryProductionRefusalGuardIsExecuted is what makes this file impossible
// to under-fill: every guard below is a derived obligation, and deleting a case
// here reddens that gate rather than quietly lowering the evidence.

// lowerUUIDv7 and higherUUIDv7 bracket every UUIDv7 the fixtures carry, so an
// identity-equality coupling can be broken in BOTH lexical directions.
//
// One direction is not enough. The revision-4 narrowing sweep confirmed it:
// with only a lexically smaller counterpart, `subjectID != sessionID` narrowed
// to `subjectID > sessionID` left the whole suite green, and the coupling would
// have silently admitted half of its own violation class.
const (
	lowerUUIDv7  = "0198f4c8-0000-7000-8000-000000000000"
	higherUUIDv7 = "0198f4c8-ffff-7fff-bfff-ffffffffffff"
)

// TestIdentityBracketsAreOnEitherSideOfEveryFixtureUUID keeps the brackets
// honest: a bracket that stopped bracketing would turn both-direction cases
// into two copies of the same direction without failing anything.
func TestIdentityBracketsAreOnEitherSideOfEveryFixtureUUID(t *testing.T) {
	t.Parallel()

	for _, reference := range []string{sessionID, sourceSessionID, groupID, hostID, peerHostID, workspaceID, operationID, bundleID} {
		if !(lowerUUIDv7 < reference) {
			t.Errorf("lowerUUIDv7 %q does not sort before fixture UUID %q", lowerUUIDv7, reference)
		}
		if !(higherUUIDv7 > reference) {
			t.Errorf("higherUUIDv7 %q does not sort after fixture UUID %q", higherUUIDv7, reference)
		}
	}
	for _, bracket := range []string{lowerUUIDv7, higherUUIDv7} {
		object := validLeaseRecordObject()
		object["subject_id"], object["session_id"] = bracket, bracket
		assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
	}
}

// TestCoreRecordCrossFieldCouplingsRefuseAtBothIdentityEntries pins the
// subject-identity couplings the review found narrowable with a green suite.
// Each case keeps every member valid and breaks exactly one equality, in both
// lexical directions, so no single comparison operator satisfies the gate.
func TestCoreRecordCrossFieldCouplingsRefuseAtBothIdentityEntries(t *testing.T) {
	tests := []struct {
		name      string
		selfField SelfField
		object    func(counterpart string) map[string]any
		want      string
	}{
		{"lease subject differs from session", SelfRecordID, func(counterpart string) map[string]any {
			object := validLeaseRecordObject()
			object["session_id"] = counterpart
			return object
		}, "Lease Record subject_id must equal session_id"},
		{"lease issuer differs from creator", SelfRecordID, func(counterpart string) map[string]any {
			object := validLeaseRecordObject()
			object["created_by_host_id"] = counterpart
			return object
		}, "issued_by_host_id"},
		{"checkpoint subject differs from session", SelfCheckpointID, func(counterpart string) map[string]any {
			object := validCheckpointRecordObject(true)
			object["session_id"] = counterpart
			return object
		}, "Checkpoint Record subject_id must equal session_id"},
		{"provider identity subject differs from session", SelfRecordID, func(counterpart string) map[string]any {
			object := validProviderIdentityRecordObject()
			object["session_id"] = counterpart
			return object
		}, "Provider Identity Record subject_id must equal session_id"},
		{"workspace group subject differs from group", SelfRecordID, func(counterpart string) map[string]any {
			object := validWorkspaceGroupRecordObject()
			object["workspace_group_id"] = counterpart
			return object
		}, "Workspace Group Record subject_id must equal workspace_group_id"},
		{"session event subject differs from session", SelfEventID, func(counterpart string) map[string]any {
			object := validSessionEventObject("4.0.0", "session.resumed")
			object["session_id"] = counterpart
			return object
		}, "subject_id"},
	}

	for _, test := range tests {
		for _, direction := range []struct {
			name        string
			counterpart string
		}{
			{"counterpart sorts lower", lowerUUIDv7},
			{"counterpart sorts higher", higherUUIDv7},
		} {
			t.Run(test.name+"/"+direction.name, func(t *testing.T) {
				assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object(direction.counterpart)), test.selfField, test.want)
			})
		}
	}
}

// TestSafeBoundaryEvidenceRefusesANonQuiescentBoundaryAtBothIdentityEntries
// pins Section 5.4. The published Checkpoint attestation says the session was
// quiescent; the review showed the counters-zero and foreground_idle gates could
// be deleted together with the whole suite still green.
func TestSafeBoundaryEvidenceRefusesANonQuiescentBoundaryAtBothIdentityEntries(t *testing.T) {
	tests := []struct {
		name   string
		member string
		value  any
		want   string
	}{
		{"foreground busy", "foreground_idle", false, "Safe Boundary Evidence foreground_idle must be true"},
		{"background busy", "background_idle", false, "Safe Boundary Evidence background_idle must be true"},
		{"input not blocked", "input_blocked", false, "Safe Boundary Evidence input_blocked must be true"},
		{"open process", "open_processes", json.Number("1"), "Safe Boundary Evidence open_processes must be zero"},
		{"open database handle", "open_database_handles", json.Number("1"), "Safe Boundary Evidence open_database_handles must be zero"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validCheckpointRecordObject(true)
			object["safe_boundary"].(map[string]any)[test.member] = test.value
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfCheckpointID, test.want)
		})
	}
}

// TestSafeBoundaryEvidenceRefusesAnUnregisteredProviderIDAtBothIdentityEntries
// pins the provider-id grammar at its Section 5.4 call site. A grammar violation
// is used rather than an unknown-but-well-formed ID, because provider-id is a
// scalar form and not a closed vocabulary.
func TestSafeBoundaryEvidenceRefusesAnUnregisteredProviderIDAtBothIdentityEntries(t *testing.T) {
	object := validCheckpointRecordObject(true)
	object["safe_boundary"].(map[string]any)["provider_id"] = "Codex"
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfCheckpointID, "member provider_id")
}

// TestProviderIdentityOpaqueIdentityRefusesNonStringValuesAtBothIdentityEntries
// pins the opaque_identity value type. The member is an open map, so no derived
// member sweep reaches inside it.
func TestProviderIdentityOpaqueIdentityRefusesNonStringValuesAtBothIdentityEntries(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"number", json.Number("1")},
		{"boolean", true},
		{"null", nil},
		{"object", map[string]any{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validProviderIdentityRecordObject()
			object["opaque_identity"] = map[string]any{"native_id": test.value}
			assertIdentityEntriesRefuseWithReason(
				t, mustJSON(t, object), SelfRecordID,
				`Provider Identity Record opaque_identity["native_id"] must be a UTF-8 string`,
			)
		})
	}
}

// TestSessionEventPredecessorsAreSortedAndAdmitDuplicates drives both arms of
// the Section 5.2 clause "<code>predecessors</code> as a sorted array of one or
// more record/event digests" (SPEC.md:1728) at both identity entries, for every
// registered Session Event version.
//
// The declared phrase is bare `sorted`. Section 1.6 defines `sorted unique
// T[n..m]` as the COMPOUND phrase meaning "bytewise canonical ordering and no
// duplicate", and Section 5.2 uses it for evidence_ids and manifest_ids in the
// same document while using bare `sorted` for predecessors alone. Production
// validated predecessors with validateSortedUniqueDigests, so it refused a
// duplicate predecessor the contract admits; the descending arm was the only
// one anything drove, and a strengthened validator is invisible to it.
//
// The version set comes from the catalog, never from a list here, so a new
// Session Event version is driven without anyone editing this test.
func TestSessionEventPredecessorsAreSortedAndAdmitDuplicates(t *testing.T) {
	first, second := numberedDigests(2)[0], numberedDigests(2)[1]
	for _, version := range registeredSessionEventVersions(t) {
		t.Run("version "+version, func(t *testing.T) {
			t.Run("ascending is admitted", func(t *testing.T) {
				object := validSessionEventObject(version, "session.created")
				object["predecessors"] = []any{first, second}
				assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfEventID)
			})

			t.Run("duplicate is admitted", func(t *testing.T) {
				object := validSessionEventObject(version, "session.created")
				object["predecessors"] = []any{first, first}
				assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfEventID)
			})

			t.Run("descending is refused", func(t *testing.T) {
				object := validSessionEventObject(version, "session.created")
				object["predecessors"] = []any{second, first}
				assertIdentityEntriesRefuseWithReason(
					t, mustJSON(t, object), SelfEventID, "member predecessors must be sorted")
			})

			t.Run("a duplicate out of order is still refused", func(t *testing.T) {
				object := validSessionEventObject(version, "session.created")
				object["predecessors"] = []any{second, second, first}
				assertIdentityEntriesRefuseWithReason(
					t, mustJSON(t, object), SelfEventID, "member predecessors must be sorted")
			})
		})
	}
}

// registeredSessionEventVersions derives the Session Event version set from the
// pinned catalog, so a new version is driven by every proof below it without an
// edit here.
func registeredSessionEventVersions(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	var versions []string
	for _, event := range catalog.Current().Events {
		if event.Family != "session_event" {
			continue
		}
		for _, version := range event.ContractVersions {
			if !seen[version] {
				seen[version] = true
				versions = append(versions, version)
			}
		}
	}
	if len(versions) == 0 {
		t.Fatal("derived zero Session Event versions from the pinned catalog")
	}
	sort.Strings(versions)
	return versions
}

// TestSessionEventEvidenceIdsStillRefuseADuplicate is the contrast case, and it
// is what makes the previous test evidence rather than a weakening. In the SAME
// record, on a member the pinned document declares as `sorted unique digest
// [1..256]` (SPEC.md:1871), a duplicate must still be refused. If someone
// "fixes" the predecessors test by weakening the shared digest-ordering helper,
// this reddens.
func TestSessionEventEvidenceIdsStillRefuseADuplicate(t *testing.T) {
	digest := numberedDigests(1)[0]
	object := validSessionEventObject("4.0.0", "terminal.created")
	payload := object["payload"].(map[string]any)
	payload["evidence_ids"] = []any{digest, digest}
	assertIdentityEntriesRefuseWithReason(
		t, mustJSON(t, object), SelfEventID, "member evidence_ids must be strictly sorted and unique")
}

// TestWorkspaceGroupMemberOrderingIsSortedAndAdmitsADuplicateWorkspaceID drives
// BOTH arms of the Section 2 clause "Members are sorted by <code>workspace_id</code>,
// and no two members may have an equal or case-colliding
// <code>group_relative_path</code>" (SPEC.md:2146) at both identity entries.
//
// The declared ordering is bare `sorted`, not the Section 1.6 compound phrase
// `sorted unique`, and the declared uniqueness is on group_relative_path. A
// descending pair must therefore be refused while an equal workspace_id on two
// members with distinct paths must be ADMITTED. The validator previously
// refused the duplicate, which is the invented-constraint class this leaf was
// reworked for five times; array-order-constraints.md is what now makes it
// mechanical rather than a reading.
func TestWorkspaceGroupMemberOrderingIsSortedAndAdmitsADuplicateWorkspaceID(t *testing.T) {
	t.Run("descending is refused", func(t *testing.T) {
		object := validWorkspaceGroupRecordObject()
		members := object["members"].([]any)
		object["members"] = []any{members[1], members[0]}
		assertIdentityEntriesRefuseWithReason(
			t, mustJSON(t, object), SelfRecordID, "Workspace Group Record members must be sorted by workspace_id")
	})

	t.Run("equal workspace_id with distinct paths is admitted", func(t *testing.T) {
		object := validWorkspaceGroupRecordObject()
		members := object["members"].([]any)
		duplicate := cloneJSONObject(t, members[1].(map[string]any))
		duplicate["workspace_id"] = members[0].(map[string]any)["workspace_id"]
		object["members"] = []any{members[0], duplicate}
		assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
	})

	t.Run("equal workspace_id still refuses a colliding path", func(t *testing.T) {
		object := validWorkspaceGroupRecordObject()
		members := object["members"].([]any)
		duplicate := cloneJSONObject(t, members[1].(map[string]any))
		duplicate["workspace_id"] = members[0].(map[string]any)["workspace_id"]
		duplicate["group_relative_path"] = members[0].(map[string]any)["group_relative_path"]
		object["members"] = []any{members[0], duplicate}
		assertIdentityEntriesRefuseWithReason(
			t, mustJSON(t, object), SelfRecordID, "are equal or case-colliding")
	})
}

// TestWorkspaceMemberRefusesAnUndeclaredKind pins the tag that selects the
// member's closed shape. An unknown kind must be refused rather than falling
// through to a shape that happens to validate.
func TestWorkspaceMemberRefusesAnUndeclaredKind(t *testing.T) {
	object := validWorkspaceGroupRecordObject()
	object["members"].([]any)[0].(map[string]any)["kind"] = "worktree"
	assertIdentityEntriesRefuseWithReason(
		t, mustJSON(t, object), SelfRecordID,
		`WorkspaceMember kind "worktree" is not git or managed_tree`,
	)
}

// TestWorkspaceMemberSanitizedRemoteURLOrderingRefusesUnsortedAndDuplicateURLs
// pins the Section 5.6 "sorted unique sanitized-git-URL[1..16]" clause in both
// failing directions.
func TestWorkspaceMemberSanitizedRemoteURLOrderingRefusesUnsortedAndDuplicateURLs(t *testing.T) {
	const want = "member sanitized_remote_urls must be strictly sorted and unique"
	const first = "ssh://git@github.com/relux/alpha.git"
	const second = "ssh://git@github.com/relux/beta.git"

	tests := []struct {
		name string
		urls []any
	}{
		{"descending", []any{second, first}},
		{"duplicate", []any{first, first}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validWorkspaceGroupRecordObject()
			object["members"].([]any)[0].(map[string]any)["sanitized_remote_urls"] = test.urls
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID, want)
		})
	}
}

// TestSessionStoppedBootstrapAbortRefusesEveryNonAbortingWitness pins the
// Section 5.2 "false resumable and graceful values" clause. Each of the three
// witnesses is violated on its own, so narrowing the guard to any two of them
// still reddens.
func TestSessionStoppedBootstrapAbortRefusesEveryNonAbortingWitness(t *testing.T) {
	const want = "session.stopped bootstrap_abort requires null checkpoint_id and false resumable and graceful"

	tests := []struct {
		name    string
		payload func(map[string]any)
	}{
		{"checkpoint present", func(payload map[string]any) { payload["checkpoint_id"] = zeroDigest }},
		{"resumable true", func(payload map[string]any) { payload["resumable"] = true }},
		{"graceful true", func(payload map[string]any) { payload["graceful"] = true }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validSessionEventObject("4.0.0", "session.stopped")
			payload := object["payload"].(map[string]any)
			payload["closure_kind"] = "bootstrap_abort"
			payload["checkpoint_id"] = nil
			payload["resumable"] = false
			payload["graceful"] = false
			test.payload(payload)
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfEventID, want)
		})
	}
}

// TestProfileChangedRefusesAnUnchangedProfile pins the from/to inequality in
// both directions of the closed vocabulary, so narrowing the guard to one value
// pair still reddens.
func TestProfileChangedRefusesAnUnchangedProfile(t *testing.T) {
	for _, profile := range []string{"standard", "yolo"} {
		t.Run(profile, func(t *testing.T) {
			object := validSessionEventObject("4.0.0", "profile.changed")
			payload := object["payload"].(map[string]any)
			payload["from"], payload["to"] = profile, profile
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfEventID, "profile.changed from and to must differ")
		})
	}
}

// TestObservationEventPhaseRefusesNonLowerSnakeCase pins the Section 18.1
// "Stable lower-snake-case phase or null" clause. Section 18.1 objects are not
// identity-addressed, so the entry point is ValidateObservationEvent.
func TestObservationEventPhaseRefusesNonLowerSnakeCase(t *testing.T) {
	for _, phase := range []string{"Destination_Validated", "destination-validated", "destination validated", "_leading"} {
		t.Run(phase, func(t *testing.T) {
			object := validObservationEventObject()
			object["phase"] = phase
			assertObservationRefusesWithReason(t, object, "Observation Event phase must use lower_snake_case")
		})
	}
}

// assertObservationRefusesWithReason drives the Section 18.1 production entry
// and requires the specific refusal, so a case cannot be satisfied by an
// unrelated earlier disjunct.
func assertObservationRefusesWithReason(t *testing.T, object map[string]any, want string) {
	t.Helper()

	err := ValidateObservationEvent(mustJSON(t, object))
	if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), want) {
		t.Fatalf("ValidateObservationEvent error = %v, want observation refusal containing %q", err, want)
	}
}

// TestSessionEventPayloadRegistryCompletenessRefusesADivergentRegistry drives
// validateSessionEventPayloadShapeCompleteness — the init-time cross-check that
// binds the payload registry to the generated catalog — with a registry that
// diverges in each of the three ways the check names. The production call site
// is the package initialiser in core_records.go, which panics on a non-nil
// error; the check is exercised here as the pure function it is, so its three
// refusals are proven rather than declared unreachable.
func TestSessionEventPayloadRegistryCompletenessRefusesADivergentRegistry(t *testing.T) {
	t.Parallel()

	events := catalog.Current().Events
	complete := sessionEventPayloadShapes
	if err := validateSessionEventPayloadShapeCompleteness(complete, events); err != nil {
		t.Fatalf("unmutated payload registry is already refused: %v; every case below would be vacuous", err)
	}

	var anyVersion, anyName string
	for version, shapes := range complete {
		anyVersion = version
		for name := range shapes {
			anyName = name
			break
		}
		break
	}

	tests := []struct {
		name     string
		registry func() map[string]map[string]eventPayloadShape
		want     string
	}{
		{"missing a whole contract version", func() map[string]map[string]eventPayloadShape {
			registry := copyPayloadRegistry(complete)
			delete(registry, anyVersion)
			return registry
		}, "payload registry has"},
		{"missing one event in a version", func() map[string]map[string]eventPayloadShape {
			registry := copyPayloadRegistry(complete)
			delete(registry[anyVersion], anyName)
			return registry
		}, "payload registry has"},
		{"an event registered with an empty member set", func() map[string]map[string]eventPayloadShape {
			registry := copyPayloadRegistry(complete)
			registry[anyVersion][anyName] = eventPayloadShape{}
			return registry
		}, "payload registry is missing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSessionEventPayloadShapeCompleteness(test.registry(), events)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateSessionEventPayloadShapeCompleteness error = %v, want a refusal containing %q", err, test.want)
			}
		})
	}
}

func copyPayloadRegistry(source map[string]map[string]eventPayloadShape) map[string]map[string]eventPayloadShape {
	copied := make(map[string]map[string]eventPayloadShape, len(source))
	for version, shapes := range source {
		copied[version] = clonePayloadShapes(shapes)
	}
	return copied
}

// TestObservationStreamContinuityRefusesBothFailingDirections pins the Section
// 18.1 "increases by exactly one" clause.
//
// A repeated sequence alone does not prove it: narrowing the guard from
// `sequence != previous+1` to `sequence < previous+1` still refuses a repeat
// while silently admitting a gap, and the normative negative the clause names
// is a MISSING sequence. Both directions are therefore pinned, and the
// out-of-order direction is pinned too, so no single comparison can satisfy the
// gate on its own.
func TestObservationStreamContinuityRefusesBothFailingDirections(t *testing.T) {
	tests := []struct {
		name      string
		sequences []int
		want      string
	}{
		{"gap forward", []int{1, 3}, "sequence"},
		{"repeat", []int{1, 1}, "sequence"},
		{"backwards", []int{1, 2, 2}, "sequence"},
		{"does not start at one", []int{2, 3}, "stream sequence must start at 1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := make([][]byte, 0, len(test.sequences))
			for _, sequence := range test.sequences {
				event := validObservationEventObject()
				event["sequence"] = json.Number(strconv.Itoa(sequence))
				events = append(events, mustJSON(t, event))
			}
			err := ValidateObservationStream(events)
			if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateObservationStream(%v) error = %v, want observation refusal containing %q", test.sequences, err, test.want)
			}
		})
	}

	t.Run("a contiguous stream is accepted", func(t *testing.T) {
		events := make([][]byte, 0, 3)
		for sequence := 1; sequence <= 3; sequence++ {
			event := validObservationEventObject()
			event["sequence"] = json.Number(strconv.Itoa(sequence))
			events = append(events, mustJSON(t, event))
		}
		if err := ValidateObservationStream(events); err != nil {
			t.Fatalf("ValidateObservationStream(contiguous) error = %v, want acceptance", err)
		}
	})
}

// TestObservationStreamRefusesAStreamIDChangeInBothDirections pins the Section
// 18.1 single-stream rule. Only one lexical direction was proven, so narrowing
// the guard to a single comparison left the suite green while half the
// violation class was admitted.
func TestObservationStreamRefusesAStreamIDChangeInBothDirections(t *testing.T) {
	for _, replacement := range []string{lowerUUIDv7, higherUUIDv7} {
		t.Run(replacement, func(t *testing.T) {
			first := validObservationEventObject()
			second := validObservationEventObject()
			second["sequence"] = json.Number("2")
			second["stream_id"] = replacement
			events := [][]byte{mustJSON(t, first), mustJSON(t, second)}
			err := ValidateObservationStream(events)
			if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), "stream_id changed within stream") {
				t.Fatalf("ValidateObservationStream(stream_id -> %s) error = %v, want a stream_id refusal", replacement, err)
			}
		})
	}
}

// TestSessionStoppedCheckpointedRefusesEveryNonCheckpointedWitness is the other
// branch of the session.stopped coupling. A derived narrowing sweep showed the
// checkpointed arm proven in one direction only: dropping `!resumable` from
// `if !present || !resumable` left the whole suite green, so a checkpointed
// closure claiming an unresumable session was attested. Each case here violates
// exactly one conjunct.
func TestSessionStoppedCheckpointedRefusesEveryNonCheckpointedWitness(t *testing.T) {
	const want = "session.stopped checkpointed requires non-null checkpoint_id and resumable true"

	tests := []struct {
		name    string
		payload func(map[string]any)
	}{
		{"checkpoint null", func(payload map[string]any) { payload["checkpoint_id"] = nil }},
		{"resumable false", func(payload map[string]any) { payload["resumable"] = false }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validSessionEventObject("4.0.0", "session.stopped")
			payload := object["payload"].(map[string]any)
			payload["closure_kind"] = "checkpointed"
			payload["checkpoint_id"] = zeroDigest
			payload["resumable"] = true
			payload["graceful"] = true
			test.payload(payload)
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfEventID, want)
		})
	}
}

// absolutePathWitnesses is every form the absolute-path refusals reject, one per
// disjunct of the production guard. A derived narrowing sweep showed those
// guards proven by a single POSIX witness: dropping the UNC disjunct, and
// dropping the drive-letter pattern, both left the suite green.
var absolutePathWitnesses = []struct {
	name  string
	value string
}{
	{"posix root", "/Users/alice/store"},
	{"windows unc", `\\host\share\store`},
	{"windows drive letter", `C:\Users\alice\store`},
	{"windows drive letter forward slash", "D:/Users/alice/store"},
}

// TestProviderIdentityOpaqueValueRefusesEveryAbsolutePathForm drives every
// witness through both identity entries. The production call site is
// validateProviderIdentityRecord, reached from CalculateObjectIdentity and
// VerifyObjectIdentity through validateImmutableObjectShape.
func TestProviderIdentityOpaqueValueRefusesEveryAbsolutePathForm(t *testing.T) {
	for _, witness := range absolutePathWitnesses {
		t.Run(witness.name, func(t *testing.T) {
			object := validProviderIdentityRecordObject()
			object["opaque_identity"] = map[string]any{"store_path": witness.value}
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID,
				"opaque_identity[\"store_path\"] must not begin with an absolute path")
		})
	}
}

// TestWorkspaceMemberLogicalIdentityRefusesEveryAbsolutePathForm covers the same
// disjuncts at the other production call site, requireBoundedLogicalIdentity,
// for both arms of the WorkspaceMember tagged union.
func TestWorkspaceMemberLogicalIdentityRefusesEveryAbsolutePathForm(t *testing.T) {
	members := []struct {
		name   string
		index  int
		member string
	}{
		{"git repository_identity", 0, "repository_identity"},
		{"managed_tree tree_identity", 1, "tree_identity"},
	}
	for _, subject := range members {
		for _, witness := range absolutePathWitnesses {
			t.Run(subject.name+"/"+witness.name, func(t *testing.T) {
				object := validWorkspaceGroupRecordObject()
				object["members"].([]any)[subject.index].(map[string]any)[subject.member] = witness.value
				assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID,
					"member "+subject.member+" must be a logical identity, not an absolute path")
			})
		}
	}
}
