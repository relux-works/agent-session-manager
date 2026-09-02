package canonicaljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

const (
	hostID      = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	peerHostID  = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	workspaceID = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	groupID     = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	leaseID     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	priorLease  = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
)

func TestCoreRecordFixturesReachBothIdentityProductionEntries(t *testing.T) {
	fixtures := []struct {
		name      string
		selfField SelfField
		object    map[string]any
	}{
		{"lease", SelfRecordID, validLeaseRecordObject()},
		{"checkpoint direct", SelfCheckpointID, validCheckpointRecordObject(true)},
		{"checkpoint task board", SelfCheckpointID, validCheckpointRecordObject(false)},
		{"provider identity", SelfRecordID, validProviderIdentityRecordObject()},
		{"workspace group", SelfRecordID, validWorkspaceGroupRecordObject()},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, mustJSON(t, fixture.object), fixture.selfField)
		})
	}
}

func TestEveryCatalogSessionEventVersionAndTypeReachesIdentityProductionEntries(t *testing.T) {
	for _, event := range catalog.Current().Events {
		if event.Family != "session_event" {
			continue
		}
		for _, version := range event.ContractVersions {
			name := string(event.Name)
			t.Run(version+"/"+name, func(t *testing.T) {
				object := validSessionEventObject(version, name)
				assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfEventID)
			})
		}
	}
}

func TestCoreRecordClosedShapesRefuseMissingAndUnknownMembersAtProductionEntries(t *testing.T) {
	fixtures := []struct {
		name      string
		selfField SelfField
		object    map[string]any
	}{
		{"lease", SelfRecordID, validLeaseRecordObject()},
		{"checkpoint", SelfCheckpointID, validCheckpointRecordObject(true)},
		{"provider identity", SelfRecordID, validProviderIdentityRecordObject()},
		{"workspace group", SelfRecordID, validWorkspaceGroupRecordObject()},
		{"session event", SelfEventID, validSessionEventObject("4.0.0", "session.resumed")},
	}

	for _, fixture := range fixtures {
		for _, path := range closedObjectMemberPaths(fixture.object) {
			if len(path) == 1 && (path[0] == "schema" || path[0] == "schema_version" || path[0] == string(fixture.selfField)) {
				continue
			}
			path := path
			t.Run(fixture.name+"/missing "+formatJSONPath(path), func(t *testing.T) {
				candidate := cloneJSONObject(t, fixture.object)
				deleteJSONObjectMemberAtPath(t, candidate, path)
				assertIdentityEntriesRefuseShape(t, mustJSON(t, candidate), fixture.selfField)
			})
		}

		t.Run(fixture.name+"/unknown top-level", func(t *testing.T) {
			candidate := cloneJSONObject(t, fixture.object)
			candidate["unknown"] = true
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate), fixture.selfField, "unknown member")
		})
	}
}

func TestCoreRecordCrossFieldAndNormativeNegativeFixturesReachProductionEntries(t *testing.T) {
	tests := []struct {
		name      string
		selfField SelfField
		object    func() map[string]any
		want      string
	}{
		{"lease subject mismatch", SelfRecordID, func() map[string]any { o := validLeaseRecordObject(); o["subject_id"] = sourceSessionID; return o }, "subject_id"},
		{"lease epoch zero", SelfRecordID, func() map[string]any { o := validLeaseRecordObject(); o["epoch"] = json.Number("0"); return o }, "epoch"},
		{"lease first epoch predecessor", SelfRecordID, func() map[string]any {
			o := validLeaseRecordObject()
			o["epoch"] = json.Number("1")
			o["reason"] = "create"
			o["checkpoint_id"] = nil
			return o
		}, "predecessor"},
		{"lease later epoch missing checkpoint", SelfRecordID, func() map[string]any { o := validLeaseRecordObject(); o["checkpoint_id"] = nil; return o }, "checkpoint_id"},
		{"lease issuer mismatch", SelfRecordID, func() map[string]any { o := validLeaseRecordObject(); o["created_by_host_id"] = hostID; return o }, "issued_by_host_id"},
		{"checkpoint CP-N1 background busy", SelfCheckpointID, func() map[string]any {
			o := validCheckpointRecordObject(true)
			o["safe_boundary"].(map[string]any)["background_idle"] = false
			return o
		}, "background_idle"},
		{"checkpoint CP-N2 both persistence refs null", SelfCheckpointID, func() map[string]any {
			o := validCheckpointRecordObject(true)
			o["provider_manifest_id"] = nil
			return o
		}, "exactly one"},
		{"checkpoint CP-N3 both persistence refs present", SelfCheckpointID, func() map[string]any {
			o := validCheckpointRecordObject(true)
			o["task_board_bundle_id"] = zeroDigest
			return o
		}, "exactly one"},
		{"checkpoint CP-N4 unknown boundary member", SelfCheckpointID, func() map[string]any {
			o := validCheckpointRecordObject(true)
			o["safe_boundary"].(map[string]any)["pid"] = json.Number("1")
			return o
		}, "unknown member"},
		{"provider identity unknown kind", SelfRecordID, func() map[string]any {
			o := validProviderIdentityRecordObject()
			o["identity_kind"] = "guessed"
			return o
		}, "identity_kind"},
		{"provider identity absolute opaque path", SelfRecordID, func() map[string]any {
			o := validProviderIdentityRecordObject()
			o["opaque_identity"] = map[string]any{"store_path": "/Users/alice/.provider"}
			return o
		}, "absolute path"},
		{"provider identity backend realm missing", SelfRecordID, func() map[string]any {
			o := validProviderIdentityRecordObject()
			o["backend_realm_fingerprint"] = nil
			return o
		}, "backend_realm_fingerprint"},
		{"workspace WG-N1 managed remote URLs", SelfRecordID, func() map[string]any {
			o := validWorkspaceGroupRecordObject()
			o["members"].([]any)[1].(map[string]any)["sanitized_remote_urls"] = []any{"https://example.com/repo.git"}
			return o
		}, "unknown member"},
		{"workspace WG-N2 git repository absent", SelfRecordID, func() map[string]any {
			o := validWorkspaceGroupRecordObject()
			delete(o["members"].([]any)[0].(map[string]any), "repository_identity")
			return o
		}, "missing required member"},
		{"workspace WG-N3 managed wrong policy", SelfRecordID, func() map[string]any {
			o := validWorkspaceGroupRecordObject()
			o["members"].([]any)[1].(map[string]any)["materialization_policy"] = "separate_worktree"
			return o
		}, "materialization_policy"},
		{"workspace WG-N4 nested unknown", SelfRecordID, func() map[string]any {
			o := validWorkspaceGroupRecordObject()
			o["members"].([]any)[0].(map[string]any)["unknown"] = true
			return o
		}, "unknown member"},
		{"workspace git absolute repository identity", SelfRecordID, func() map[string]any {
			o := validWorkspaceGroupRecordObject()
			o["members"].([]any)[0].(map[string]any)["repository_identity"] = "/Users/alice/repository"
			return o
		}, "absolute path"},
		{"workspace managed absolute tree identity", SelfRecordID, func() map[string]any {
			o := validWorkspaceGroupRecordObject()
			o["members"].([]any)[1].(map[string]any)["tree_identity"] = `C:\\Users\\alice\\tree`
			return o
		}, "absolute path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object()), test.selfField, test.want)
		})
	}
}

func TestSessionEventVersionIsolationAndCrossFieldRefusalsReachProductionEntries(t *testing.T) {
	tests := []struct {
		name   string
		object func() map[string]any
		want   string
	}{
		{"v2 refuses v3 adoption event", func() map[string]any { return validSessionEventObject("2.0.0", "adoption.planned") }, "event_type"},
		{"v3 refuses v4 terminal payload", func() map[string]any {
			o := validSessionEventObject("4.0.0", "terminal.created")
			o["schema_version"] = "3.0.0"
			return o
		}, "terminal.created"},
		{"v4 refuses v3 terminal payload", func() map[string]any {
			o := validSessionEventObject("3.0.0", "terminal.created")
			o["schema_version"] = "4.0.0"
			return o
		}, "terminal.created"},
		{"subject differs from session", func() map[string]any {
			o := validSessionEventObject("1.0.0", "session.created")
			o["subject_id"] = sourceSessionID
			return o
		}, "subject_id"},
		{"lease epoch zero", func() map[string]any {
			o := validSessionEventObject("1.0.0", "session.created")
			o["lease_epoch"] = json.Number("0")
			return o
		}, "lease_epoch"},
		{"lease sequence zero", func() map[string]any {
			o := validSessionEventObject("1.0.0", "session.created")
			o["lease_sequence"] = json.Number("0")
			return o
		}, "lease_sequence"},
		{"predecessors empty", func() map[string]any {
			o := validSessionEventObject("1.0.0", "session.created")
			o["predecessors"] = []any{}
			return o
		}, "predecessors"},
		{"force risks widened", func() map[string]any {
			o := validSessionEventObject("1.0.0", "takeover.force_confirmed")
			o["payload"].(map[string]any)["accepted_risks"] = []any{"data_loss", "divergent_history", "split_brain", "stale_process"}
			return o
		}, "accepted_risks"},
		{"task board goal half absent", func() map[string]any {
			o := validSessionEventObject("1.0.0", "task_board.launched")
			o["payload"].(map[string]any)["board_goal_revision"] = nil
			return o
		}, "goal"},
		{"tombstone resurrection missing digest", func() map[string]any {
			o := validSessionEventObject("1.0.0", "tombstone.resolved")
			o["payload"].(map[string]any)["resulting_entry_digest"] = nil
			return o
		}, "resulting_entry_digest"},
		{"checkpointed stop missing checkpoint", func() map[string]any {
			o := validSessionEventObject("1.0.0", "session.stopped")
			o["payload"].(map[string]any)["checkpoint_id"] = nil
			return o
		}, "checkpointed"},
		{"fork new-session profile source is non-null", func() map[string]any {
			o := validSessionEventObject("1.0.0", "fork.created")
			o["payload"].(map[string]any)["profile_source_event_id"] = zeroDigest
			return o
		}, "profile_source_event_id"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object()), SelfEventID, test.want)
		})
	}
}

func TestSessionEventV1RetainsUnknownTypeWithoutInterpretingPayload(t *testing.T) {
	object := validSessionEventObject("1.0.0", "future.event")
	object["payload"] = map[string]any{"future_member": true}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfEventID)
}

func TestEveryCatalogSessionEventPayloadMemberIsRequiredAndTypedAtProductionEntries(t *testing.T) {
	for version, shapes := range sessionEventPayloadShapes {
		for eventType, shape := range shapes {
			version, eventType, shape := version, eventType, shape
			t.Run(version+"/"+eventType, func(t *testing.T) {
				fixture := validSessionEventObject(version, eventType)
				payload := fixture["payload"].(map[string]any)
				if len(payload) != len(shape.members) {
					t.Fatalf("fixture has %d payload members, production registry has %d", len(payload), len(shape.members))
				}
				for _, member := range shape.members {
					member := member
					t.Run("missing "+member, func(t *testing.T) {
						candidate := cloneJSONObject(t, fixture)
						delete(candidate["payload"].(map[string]any), member)
						assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate), SelfEventID, member)
					})
					t.Run("wrong type "+member, func(t *testing.T) {
						candidate := cloneJSONObject(t, fixture)
						candidate["payload"].(map[string]any)[member] = map[string]any{}
						assertIdentityEntriesRefuseShape(t, mustJSON(t, candidate), SelfEventID)
					})
				}
				t.Run("unknown member", func(t *testing.T) {
					candidate := cloneJSONObject(t, fixture)
					candidate["payload"].(map[string]any)["unknown"] = true
					assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate), SelfEventID, "unknown member")
				})
			})
		}
	}
}

// TestEverySessionEventClosedVocabularyRefusesComplementAtProductionEntries
// derives its subject from the pinned payload inventory instead of listing it.
//
// The previous hand-maintained list said "Every" and was missing
// takeover.force_confirmed accepted_risks, a closed three-value enum array: the
// test could not fail when a vocabulary was added without a row. The subject is
// now every inventory row whose pinned declaration is a closed vocabulary, so a
// new vocabulary becomes an obligation with no one adding anything.
func TestEverySessionEventClosedVocabularyRefusesComplementAtProductionEntries(t *testing.T) {
	packageDirectory, _ := packageProductionFiles(t)
	inventory := readSessionEventPayloadInventory(t, filepath.Join(packageDirectory, "testdata", "session-event-payload-members.md"))

	swept := 0
	for _, subject := range closedVocabularyPayloadMembers(t, inventory) {
		subject := subject
		swept++
		t.Run(subject.version+"/"+subject.eventType+"/"+subject.member, func(t *testing.T) {
			candidate := validSessionEventObject(subject.version, subject.eventType)
			payload := candidate["payload"].(map[string]any)
			if existing, isArray := payload[subject.member].([]any); isArray {
				// An enum array is declared with an exact length and bytewise
				// order, so the outside value replaces an element rather than
				// the array: a shorter array would be refused on length and
				// prove nothing about the vocabulary.
				replaced := append([]any{}, existing...)
				replaced[0] = "aaa_outside_vocabulary"
				payload[subject.member] = replaced
			} else {
				payload[subject.member] = "outside_vocabulary"
			}
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate), SelfEventID, subject.member)
		})
	}
	if swept == 0 {
		t.Fatal("derived zero closed payload vocabularies from the pinned inventory; the derivation is broken, not the package")
	}
}

type closedVocabularySubject struct {
	version   string
	eventType string
	member    string
}

// closedVocabularyPayloadMembers derives every payload member the pinned
// inventory declares as a closed vocabulary, paired with a contract version
// that carries it.
//
// A declaration is a closed vocabulary when its type is an alternation of two
// or more bare lower-snake tokens, or when it declares an enum collection.
// A single alternative plus null — "digest|null", "string[1..512]|null" — is a
// nullable scalar, not a vocabulary, and is excluded.
func closedVocabularyPayloadMembers(t *testing.T, inventory payloadInventory) []closedVocabularySubject {
	t.Helper()

	versions := map[string][]string{}
	for _, event := range catalog.Current().Events {
		if event.Family == "session_event" {
			versions[string(event.Name)] = event.ContractVersions
		}
	}
	overrides := inventory.byDefinition[payloadInventoryOverrideVersion+"-override"]

	var subjects []closedVocabularySubject
	for key, declaration := range inventory.declarations {
		definition, rest, found := strings.Cut(key, " ")
		if !found {
			t.Fatalf("malformed payload inventory key %q", key)
		}
		// Event types carry a dot, so the member is whatever follows the last one.
		separator := strings.LastIndex(rest, ".")
		if separator < 0 {
			t.Fatalf("malformed payload inventory key %q", key)
		}
		eventType, member := rest[:separator], rest[separator+1:]
		if !isClosedVocabularyDeclaration(declaration) {
			continue
		}
		version := ""
		if definition == payloadInventoryOverrideVersion+"-override" {
			version = payloadInventoryOverrideVersion
		} else {
			if _, overridden := overrides[eventType]; overridden {
				// The 4.0.0 override replaces this payload, so the default rows
				// are proven at a version that still selects them.
				for _, candidate := range versions[eventType] {
					if candidate != payloadInventoryOverrideVersion {
						version = candidate
						break
					}
				}
			} else if declared := versions[eventType]; len(declared) > 0 {
				version = declared[0]
			}
		}
		if version == "" {
			t.Fatalf("pinned inventory row %s has no contract version that selects it", key)
		}
		subjects = append(subjects, closedVocabularySubject{version: version, eventType: eventType, member: member})
	}
	sort.Slice(subjects, func(first, second int) bool {
		left := subjects[first].version + subjects[first].eventType + subjects[first].member
		right := subjects[second].version + subjects[second].eventType + subjects[second].member
		return left < right
	})
	return subjects
}

func isClosedVocabularyDeclaration(declaration string) bool {
	_, declared, found := strings.Cut(declaration, ":")
	if !found {
		return false
	}
	if strings.Contains(declared, "enum") {
		return true
	}
	var alternatives []string
	for _, alternative := range strings.Split(declared, "|") {
		alternative = strings.TrimSpace(alternative)
		if alternative == "null" || alternative == "" {
			continue
		}
		if !lowerSnakePattern.MatchString(alternative) {
			return false
		}
		alternatives = append(alternatives, alternative)
	}
	return len(alternatives) >= 2
}

func TestSessionEventLiteralClaimsRefuseOppositeValueAtProductionEntries(t *testing.T) {
	tests := []struct {
		version   string
		eventType string
		member    string
	}{
		{"2.0.0", "clone.target_prepared", "rollback_retained"},
		{"2.0.0", "clone.target_published", "source_generation_revalidated"},
		{"2.0.0", "clone.target_published", "rollback_retained"},
		{"2.0.0", "clone.committed", "native_resumable"},
		{"2.0.0", "clone.failed", "retryable"},
		{"3.0.0", "adoption.committed", "native_resumable"},
	}
	for _, test := range tests {
		t.Run(test.eventType+"/"+test.member, func(t *testing.T) {
			candidate := validSessionEventObject(test.version, test.eventType)
			candidate["payload"].(map[string]any)[test.member] = false
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate), SelfEventID, test.member)
		})
	}

	for _, test := range []struct{ eventType, member string }{
		{"clone.failed", "transaction_unknown"},
	} {
		candidate := validSessionEventObject("2.0.0", test.eventType)
		candidate["payload"].(map[string]any)[test.member] = true
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate), SelfEventID, test.member)
	}
}

func TestCoreRecordCollectionAndMapBoundsReachProductionEntries(t *testing.T) {
	t.Run("checkpoint event heads 64 and 65", func(t *testing.T) {
		at := validCheckpointRecordObject(true)
		at["event_heads"] = numberedDigests(64)
		assertIdentityEntriesAcceptShape(t, mustJSON(t, at), SelfCheckpointID)
		over := cloneJSONObject(t, at)
		over["event_heads"] = numberedDigests(65)
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, over), SelfCheckpointID, "maximum length 64")
	})

	t.Run("provider opaque map 32 and 33", func(t *testing.T) {
		at := validProviderIdentityRecordObject()
		at["opaque_identity"] = numberedOpaqueIdentity(32)
		assertIdentityEntriesAcceptShape(t, mustJSON(t, at), SelfRecordID)
		over := cloneJSONObject(t, at)
		over["opaque_identity"] = numberedOpaqueIdentity(33)
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, over), SelfRecordID, "maximum length 32")
	})

	t.Run("workspace members 1 and 257", func(t *testing.T) {
		one := validWorkspaceGroupRecordObject()
		one["members"] = one["members"].([]any)[:1]
		assertIdentityEntriesAcceptShape(t, mustJSON(t, one), SelfRecordID)
		empty := cloneJSONObject(t, one)
		empty["members"] = []any{}
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, empty), SelfRecordID, "at least 1")
		over := cloneJSONObject(t, one)
		over["members"] = numberedManagedWorkspaceMembers(257)
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, over), SelfRecordID, "maximum length 256")
	})

	t.Run("workspace remote URLs 1 and 17", func(t *testing.T) {
		at := validWorkspaceGroupRecordObject()
		at["members"] = at["members"].([]any)[:1]
		at["members"].([]any)[0].(map[string]any)["sanitized_remote_urls"] = numberedSanitizedURLs(16)
		assertIdentityEntriesAcceptShape(t, mustJSON(t, at), SelfRecordID)
		over := cloneJSONObject(t, at)
		over["members"].([]any)[0].(map[string]any)["sanitized_remote_urls"] = numberedSanitizedURLs(17)
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, over), SelfRecordID, "maximum length 16")
	})
}

func TestObservationDeclaredBoundsAndStreamBranchesReachProductionEntries(t *testing.T) {
	event := validObservationEventObject()
	for _, test := range []struct {
		name string
		at   string
		over string
	}{
		{"event name 128 characters", "a." + strings.Repeat("b", 126), "a." + strings.Repeat("b", 127)},
		{"phase 128 characters", strings.Repeat("a", 128), strings.Repeat("a", 129)},
		{"error code 128 characters", strings.Repeat("a", 128), strings.Repeat("a", 129)},
	} {
		t.Run(test.name, func(t *testing.T) {
			at := cloneJSONObject(t, event)
			member := "event"
			if strings.HasPrefix(test.name, "phase") {
				member = "phase"
			} else if strings.HasPrefix(test.name, "error") {
				member = "error_code"
				at["result"] = "failure"
			}
			at[member] = test.at
			if err := ValidateObservationEvent(mustJSON(t, at)); err != nil {
				t.Fatalf("at-limit observation rejected: %v", err)
			}
			over := cloneJSONObject(t, at)
			over[member] = test.over
			if err := ValidateObservationEvent(mustJSON(t, over)); err == nil {
				t.Fatal("over-limit observation accepted")
			}
		})
	}

	t.Run("started duration must be null", func(t *testing.T) {
		candidate := cloneJSONObject(t, event)
		candidate["result"] = "started"
		if err := ValidateObservationEvent(mustJSON(t, candidate)); err == nil {
			t.Fatal("started observation accepted non-null duration")
		}
	})

	t.Run("stream starts at one", func(t *testing.T) {
		candidate := cloneJSONObject(t, event)
		candidate["sequence"] = json.Number("2")
		if err := ValidateObservationStream([][]byte{mustJSON(t, candidate)}); err == nil {
			t.Fatal("stream accepted first sequence 2")
		}
	})

	t.Run("stream ID stays stable", func(t *testing.T) {
		second := cloneJSONObject(t, event)
		second["stream_id"] = operationID
		second["sequence"] = json.Number("2")
		if err := ValidateObservationStream([][]byte{mustJSON(t, event), mustJSON(t, second)}); err == nil {
			t.Fatal("stream accepted changing stream_id")
		}
	})

	t.Run("errors preserve observation sentinel", func(t *testing.T) {
		candidate := cloneJSONObject(t, event)
		delete(candidate, "sequence")
		err := ValidateObservationEvent(mustJSON(t, candidate))
		if !errors.Is(err, ErrInvalidObservation) {
			t.Fatalf("error = %v, want ErrInvalidObservation", err)
		}
	})
}

func TestObservationEventAndStreamProductionEntries(t *testing.T) {
	event := validObservationEventObject()
	if err := ValidateObservationEvent(mustJSON(t, event)); err != nil {
		t.Fatalf("ValidateObservationEvent(valid) error = %v", err)
	}
	second := cloneJSONObject(t, event)
	second["sequence"] = json.Number("2")
	if err := ValidateObservationStream([][]byte{mustJSON(t, event), mustJSON(t, second)}); err != nil {
		t.Fatalf("ValidateObservationStream(valid) error = %v", err)
	}

	for _, mutate := range []struct {
		name string
		fn   func(map[string]any)
	}{
		{"missing sequence", func(o map[string]any) { delete(o, "sequence") }},
		{"missing nullable peer host", func(o map[string]any) { delete(o, "peer_host_id") }},
		{"result ok", func(o map[string]any) { o["result"] = "ok" }},
		{"partial null error", func(o map[string]any) { o["result"] = "partial"; o["error_code"] = nil }},
		{"counts missing chunks", func(o map[string]any) { delete(o["counts"].(map[string]any), "chunks") }},
		{"negative bytes", func(o map[string]any) { o["counts"].(map[string]any)["bytes"] = json.Number("-1") }},
		{"duplicate object IDs", func(o map[string]any) { o["object_ids"] = []any{zeroDigest, zeroDigest} }},
		{"unknown provider", func(o map[string]any) { o["provider"] = "codex" }},
		{"event grammar widened uppercase", func(o map[string]any) { o["event"] = "Takeover.phase" }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			candidate := cloneJSONObject(t, event)
			mutate.fn(candidate)
			if err := ValidateObservationEvent(mustJSON(t, candidate)); err == nil {
				t.Fatal("ValidateObservationEvent accepted normative negative fixture")
			}
		})
	}

	t.Run("repeated sequence", func(t *testing.T) {
		if err := ValidateObservationStream([][]byte{mustJSON(t, event), mustJSON(t, event)}); err == nil || !strings.Contains(err.Error(), "sequence") {
			t.Fatalf("ValidateObservationStream repeated sequence error = %v", err)
		}
	})
}

func TestCoreRecordDeclaredUnicodeBoundsCountCharactersAtProductionEntries(t *testing.T) {
	provider := validProviderIdentityRecordObject()
	provider["native_session_id"] = strings.Repeat("界", 512)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, provider), SelfRecordID)
	provider["native_session_id"] = strings.Repeat("界", 513)
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, provider), SelfRecordID, "1..512 Unicode characters")

	workspace := validWorkspaceGroupRecordObject()
	workspace["display_name"] = strings.Repeat("界", 128)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, workspace), SelfRecordID)
	workspace["display_name"] = strings.Repeat("界", 129)
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, workspace), SelfRecordID, "1..128 Unicode characters")
}

func TestCoreRecordValidationIsRepeatableAndReadOnly(t *testing.T) {
	object := validSessionEventObject("4.0.0", "session.resumed")
	before := string(mustJSON(t, object))
	for range 2 {
		if _, _, err := CalculateObjectIdentity(mustJSON(t, object)); err != nil {
			t.Fatal(err)
		}
	}
	if after := string(mustJSON(t, object)); after != before {
		t.Fatalf("identity validation mutated caller object\nbefore: %s\nafter:  %s", before, after)
	}

	observation := validObservationEventObject()
	observationBefore := string(mustJSON(t, observation))
	for range 2 {
		if err := ValidateObservationEvent(mustJSON(t, observation)); err != nil {
			t.Fatal(err)
		}
	}
	if after := string(mustJSON(t, observation)); after != observationBefore {
		t.Fatalf("observation validation mutated caller object\nbefore: %s\nafter:  %s", observationBefore, after)
	}
}

func validLeaseRecordObject() map[string]any {
	return map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "lease_id": leaseID, "epoch": json.Number("4"),
		"holder_host_id": peerHostID, "predecessor_lease_id": priorLease, "reason": "graceful_takeover",
		"checkpoint_id": zeroDigest, "issued_by_host_id": peerHostID, "created_by_host_id": peerHostID,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
}

func validCheckpointRecordObject(direct bool) map[string]any {
	providerManifest, boardBundle := any(zeroDigest), any(nil)
	if !direct {
		providerManifest, boardBundle = nil, zeroDigest
	}
	return map[string]any{
		"schema": "urn:ax:schema:checkpoint", "schema_version": "1.0.0", "checkpoint_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "lease_epoch": json.Number("4"), "lease_id": leaseID,
		"safe_boundary": map[string]any{
			"provider_id": "codex", "provider_version": "0.147.0", "evidence": "accepted_test",
			"input_blocked": true, "foreground_idle": true, "background_idle": true,
			"open_processes": json.Number("0"), "open_database_handles": json.Number("0"),
		},
		"event_heads": []any{zeroDigest}, "workspace_manifest_id": zeroDigest,
		"provider_manifest_id": providerManifest, "task_board_bundle_id": boardBundle,
		"created_by_host_id": hostID, "created_at": "2026-08-19T04:09:30.000Z", "status": "validated", "extensions": map[string]any{},
	}
}

func validProviderIdentityRecordObject() map[string]any {
	return map[string]any{
		"schema": "urn:ax:schema:provider-identity", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "provider_id": "antigravity",
		"provider_version": "1.1.14", "provider_version_range": ">=1.1.14 <1.2.0",
		"native_session_id": "11111111-2222-4333-8444-555555555555", "identity_kind": "backend_conversation_uuid",
		"logical_workspace_id": workspaceID, "backend_realm_fingerprint": zeroDigest,
		// One opaque_identity member is carried so the declared key grammar is
		// reachable from this fixture: with an empty map, every derived sweep
		// that walks the fixtures skips providerIdentityKeyPattern entirely and
		// the grammar is proven nowhere.
		"opaque_identity":    map[string]any{"backend.conversation-id": "conv-0198f4c8"},
		"created_by_host_id": hostID,
		"created_at":         "2026-08-19T04:09:45.000Z", "extensions": map[string]any{},
	}
}

func validWorkspaceGroupRecordObject() map[string]any {
	return map[string]any{
		"schema": "urn:ax:schema:workspace-group", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": groupID, "workspace_group_id": groupID, "display_name": "payments",
		"members": []any{
			map[string]any{
				"workspace_id": workspaceID, "kind": "git", "group_relative_path": "payments-api",
				"repository_identity": "relux/payments-api", "sanitized_remote_urls": []any{"ssh://git@github.com/relux/payments-api.git"},
				"repo_relative_cwd": "src", "agent_project_config_paths": []any{"AGENTS.md"}, "materialization_policy": "separate_worktree",
			},
			map[string]any{
				"workspace_id": "0198f4c8-7d40-7e55-8e6f-2234567890ab", "kind": "managed_tree", "group_relative_path": "design-notes",
				"tree_identity": "relux/design-notes", "repo_relative_cwd": "drafts",
				"agent_project_config_paths": []any{"AGENTS.md"}, "materialization_policy": "separate_copy",
			},
		},
		"created_by_host_id": hostID, "created_at": "2026-08-19T04:09:50.000Z", "extensions": map[string]any{},
	}
}

func validSessionEventObject(version, eventType string) map[string]any {
	return map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": version, "event_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "event_type": eventType, "created_by_host_id": hostID,
		"lease_epoch": json.Number("4"), "lease_id": leaseID, "lease_sequence": json.Number("12"),
		"predecessors": []any{zeroDigest}, "created_at": "2026-08-19T04:08:00.000Z",
		"payload": validSessionEventPayload(version, eventType), "extensions": map[string]any{},
	}
}

func validSessionEventPayload(version, eventType string) map[string]any {
	digest := zeroDigest
	uuid := operationID
	switch eventType {
	case "session.created":
		return map[string]any{"session_record_id": digest, "bootstrap_operation_id": uuid, "first_checkpoint_operation_id": bundleID}
	case "terminal.created":
		if version == "4.0.0" {
			return terminalV4Payload()
		}
		return map[string]any{"backend": "tmux", "terminal_id": "ax-session"}
	case "provider.launched":
		return map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "--dangerously-bypass-approvals-and-sandbox"}
	case "provider.identified":
		return map[string]any{"provider_identity_record_id": digest, "confidence": "exact"}
	case "session.idle":
		return map[string]any{"boundary_ref": "provider-event-1", "foreground_idle": true, "background_idle": true}
	case "session.quiescing":
		return map[string]any{"operation_id": uuid, "reason": "checkpoint", "input_blocked": true}
	case "checkpoint.created":
		return map[string]any{"checkpoint_id": digest, "kind": "manual"}
	case "sync.completed":
		return map[string]any{"peer_host_id": peerHostID, "checkpoint_id": digest, "manifest_ids": []any{digest}, "materialized": true}
	case "session.stopped":
		return map[string]any{"graceful": true, "checkpoint_id": digest, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true}
	case "session.resumed":
		if version == "4.0.0" {
			p := terminalV4Payload()
			p["checkpoint_id"] = digest
			p["execution_profile"] = "yolo"
			p["profile_source_event_id"] = nil
			return p
		}
		return map[string]any{"checkpoint_id": digest, "execution_profile": "yolo", "profile_source_event_id": nil, "terminal_backend": "tmux", "native_session_id": "native-session"}
	case "session.bootstrap_aborted":
		return map[string]any{"operation_id": uuid, "failure_phase": "after_identity", "provider_identity_record_id": digest, "manager_session_ref": nil, "process_closed": true, "store_closed": true, "resume_allowed": false}
	case "lease.transferred":
		return map[string]any{"operation_id": uuid, "from_host_id": hostID, "to_host_id": peerHostID, "predecessor_lease_id": priorLease, "new_lease_id": leaseID}
	case "lease.forced":
		return map[string]any{"operation_id": uuid, "expected_owner_host_id": hostID, "expected_epoch": json.Number("3"), "new_lease_id": leaseID, "checkpoint_id": digest}
	case "session.parked":
		return map[string]any{"reason": "remote_owner", "winning_lease_id": leaseID}
	case "session.failed":
		return map[string]any{"error_code": "provider_failure", "retryable": true, "operation_id": uuid}
	case "fork.created":
		return map[string]any{"source_session_id": sourceSessionID, "source_checkpoint_id": digest, "new_session_record_id": digest, "provider_fork_mode": "native", "execution_profile": "yolo", "profile_source_event_id": nil, "source_profile_event_id": nil}
	case "profile.changed":
		return map[string]any{"from": "standard", "to": "yolo", "confirmed": true}
	case "session.tombstoned":
		return map[string]any{"tombstone_id": digest}
	case "takeover.force_confirmed":
		return map[string]any{"operation_id": uuid, "expected_owner_host_id": hostID, "expected_epoch": json.Number("3"), "checkpoint_id": digest, "accepted_risks": []any{"divergent_history", "split_brain", "stale_process"}, "confirmation_mode": "interactive"}
	case "replica.replace_confirmed":
		return map[string]any{"operation_id": uuid, "workspace_group_id": groupID, "target_host_id": peerHostID, "managed_replica_id": bundleID, "expected_marker_id": digest, "expected_checkpoint_id": digest, "replacement_checkpoint_id": digest, "confirmation_mode": "interactive"}
	case "task_board.launched":
		return map[string]any{"operation_id": uuid, "manager_session_ref": "manager-1", "provider_id": "codex", "launch_mode": "primary_owner", "lease_epoch": json.Number("4"), "lease_id": leaseID, "execution_profile": "yolo", "profile_source_event_id": nil, "board_goal_id": "GOAL-1", "board_goal_revision": json.Number("1"), "state": "running"}
	case "task_board.adopted":
		return map[string]any{"operation_id": uuid, "bundle_id": digest, "manager_session_ref": "manager-1", "board_goal_id": "GOAL-1", "board_goal_revision": json.Number("1")}
	case "tombstone.issued":
		return map[string]any{"tombstone_id": digest, "scope": "session", "subject_id": sessionID, "target_ref": "session"}
	case "tombstone.resolved":
		return map[string]any{"tombstone_id": digest, "resolution": "resurrected", "target_ref": "session", "resulting_entry_digest": digest}
	case "clone.planned":
		return map[string]any{"operation_id": uuid, "bundle_manifest_id": digest, "projection_plan_id": digest, "migration_checkpoint_id": digest, "materialization_id": bundleID, "target_environment": validEnvironmentTuple("codex", "macos", "arm64"), "expected_target_native_session_id": "native-target"}
	case "clone.target_prepared":
		return map[string]any{"operation_id": uuid, "materialization_id": bundleID, "plan_id": digest, "provider_transaction_id": operationID, "provider_prepared_result_digest": digest, "staged_read_back_evidence_manifest_id": digest, "rollback_retained": true}
	case "clone.target_published":
		return map[string]any{"operation_id": uuid, "materialization_id": bundleID, "provider_identity_record_id": digest, "target_provider_manifest_id": digest, "live_read_back_evidence_manifest_id": digest, "fidelity_report_id": digest, "validation_report_id": digest, "source_generation_revalidated": true, "rollback_retained": true}
	case "clone.target_validation_failed":
		return map[string]any{"operation_id": uuid, "materialization_id": bundleID, "phase": "live_read_back", "error_code": "validation_failed", "validation_report_id": digest, "rollback_required": true, "transaction_unknown": false}
	case "clone.rolled_back":
		return map[string]any{"operation_id": uuid, "materialization_id": bundleID, "provider_rolled_back_result_digest": digest, "retained_bundle_manifest_id": digest, "reason_code": "validation_failed"}
	case "clone.committed":
		return map[string]any{"operation_id": uuid, "materialization_id": bundleID, "provider_identity_record_id": digest, "provider_committed_result_digest": digest, "target_checkpoint_id": digest, "fidelity_report_id": digest, "validation_report_id": digest, "native_resumable": true}
	case "clone.lineage_published":
		return map[string]any{"operation_id": uuid, "target_checkpoint_id": digest, "lineage_receipt_id": digest, "bundle_manifest_id": digest}
	case "clone.failed":
		return map[string]any{"operation_id": uuid, "phase": "checkpoint", "error_code": "target_checkpoint_failed", "retryable": true, "retained_bundle_manifest_id": digest, "materialization_id": bundleID, "transaction_unknown": false}
	case "adoption.planned":
		return map[string]any{"operation_id": uuid, "plan_id": digest, "source_instance_id": digest, "source_observation_id": digest, "source_head_digest": digest}
	case "adoption.committed":
		return map[string]any{"operation_id": uuid, "provider_identity_record_id": digest, "initial_checkpoint_id": digest, "native_resumable": true}
	case "move.planned":
		return map[string]any{"operation_id": uuid, "plan_id": digest, "source_session_id": sourceSessionID, "target_session_id": sessionID}
	case "move.target_committed":
		return map[string]any{"operation_id": uuid, "target_session_id": sessionID, "target_checkpoint_id": digest, "clone_lineage_receipt_id": digest}
	case "move.source_release_requested":
		return map[string]any{"operation_id": uuid, "target_committed_event_id": digest, "source_lease_epoch": json.Number("1"), "source_lease_id": leaseID}
	case "move.source_released":
		return map[string]any{"operation_id": uuid, "target_session_id": sessionID, "source_stop_event_id": digest, "source_release_receipt_id": digest, "outcome": "moved_cross_environment"}
	case "move.source_release_failed":
		return map[string]any{"operation_id": uuid, "target_session_id": sessionID, "error_code": "source_stop_failed", "source_still_resumable": true, "outcome": "cloned_source_still_active"}
	default:
		return map[string]any{}
	}
}

func terminalV4Payload() map[string]any {
	return map[string]any{"terminal_binding_id": zeroDigest, "terminal_backend_id": "tmux", "implementation_version": "1.0.0", "protocol_version": "1.0.0", "evidence_ids": []any{zeroDigest}}
}

func validObservationEventObject() map[string]any {
	return map[string]any{
		"schema": "urn:ax:schema:observation", "schema_version": "1.0.0", "stream_id": bundleID,
		"sequence": json.Number("1"), "timestamp": "2026-08-19T04:30:00.000Z", "level": "info", "event": "takeover.phase",
		"operation_id": operationID, "session_id": sessionID, "host_id": hostID, "peer_host_id": peerHostID,
		"phase": "destination_validated", "result": "success", "duration_ms": json.Number("1240"),
		"counts":     map[string]any{"records": json.Number("12"), "events": json.Number("0"), "manifests": json.Number("0"), "blobs": json.Number("4"), "chunks": json.Number("0"), "bytes": json.Number("8192"), "retries": json.Number("0")},
		"object_ids": []any{}, "error_code": nil, "extensions": map[string]any{},
	}
}

func numberedDigests(count int) []any {
	values := make([]any, count)
	for index := range values {
		values[index] = fmt.Sprintf("sha256:%064x", index)
	}
	return values
}

func numberedOpaqueIdentity(count int) map[string]any {
	values := make(map[string]any, count)
	for index := range count {
		values[fmt.Sprintf("key%02d", index)] = "value"
	}
	return values
}

func numberedManagedWorkspaceMembers(count int) []any {
	values := make([]any, count)
	for index := range count {
		values[index] = map[string]any{
			"workspace_id":               fmt.Sprintf("0198f4c8-%04x-7e55-8e6f-%012x", index, index),
			"kind":                       "managed_tree",
			"group_relative_path":        fmt.Sprintf("tree-%03d", index),
			"tree_identity":              fmt.Sprintf("tree-%03d", index),
			"repo_relative_cwd":          ".",
			"agent_project_config_paths": []any{},
			"materialization_policy":     "separate_copy",
		}
	}
	return values
}

func numberedSanitizedURLs(count int) []any {
	values := make([]any, count)
	for index := range count {
		values[index] = fmt.Sprintf("https://example.com/repo-%02d.git", index)
	}
	return values
}
