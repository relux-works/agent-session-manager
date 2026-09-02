package canonicaljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestSpecDerivedRefusalClausesReachBothIdentityEntries(t *testing.T) {
	tests := []struct {
		name       string
		selfField  SelfField
		makeObject func(*testing.T) map[string]any
		want       string
	}{
		{
			name:      "launch environment literal name grammar",
			selfField: SelfRecordID,
			makeObject: func(*testing.T) map[string]any {
				object := validSessionRecordV1Object()
				object["launch_plan"].(map[string]any)["env_literals"] = map[string]any{"9BAD-NAME": "value"}
				return object
			},
			want: `Session Record Launch Plan env_literals key "9BAD-NAME" has invalid environment-name grammar`,
		},
		{
			name:      "board logical identifier grammar",
			selfField: SelfRecordID,
			makeObject: func(*testing.T) map[string]any {
				object := taskBoardSessionRecord("TASK-260830-8x76g1")
				board := object["task_board"].(map[string]any)["board"].(map[string]any)
				board["logical_id"] = "-leading-hyphen"
				return object
			},
			want: "Session Record Board Identity logical_id has invalid grammar",
		},
		{
			name:      "local board remote URL nullability",
			selfField: SelfRecordID,
			makeObject: func(*testing.T) map[string]any {
				object := taskBoardSessionRecord("TASK-260830-8x76g1")
				board := object["task_board"].(map[string]any)["board"].(map[string]any)
				board["remote_url"] = "https://example.com/board"
				return object
			},
			want: "local Session Record Board Identity remote_url must be null",
		},
		{
			name:      "workspace snapshot subject equality",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validTransferManifestObject("workspace_group")
				object["workspace_snapshot"].(map[string]any)["workspace_group_id"] = "0198f4c8-5b20-7c33-8c4d-2234567890ab"
				return object
			},
			want: "WorkspaceSnapshot workspace_group_id must equal manifest subject_id",
		},
		{
			name:      "head ref check-ref-format grammar",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validGitWorkspaceGroupObject()
				gitWorkspaceMember(object)["head"].(map[string]any)["ref"] = "refs/heads/bad..name"
				return object
			},
			want: "member ref: git-ref: does not satisfy git check-ref-format grammar",
		},
		{
			name:      "branch head requires ref",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validGitWorkspaceGroupObject()
				gitWorkspaceMember(object)["head"].(map[string]any)["ref"] = nil
				return object
			},
			want: "branch GitHead requires non-null oid and ref",
		},
		{
			name:      "unborn head requires branch ref",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validGitWorkspaceGroupObject()
				head := gitWorkspaceMember(object)["head"].(map[string]any)
				head["mode"] = "unborn"
				head["oid"] = nil
				head["ref"] = "refs/tags/main"
				return object
			},
			want: "unborn GitHead requires null oid and refs/heads/ ref",
		},
		{
			name:       "submodule sibling path case collision",
			selfField:  SelfManifestID,
			makeObject: gitWorkspaceWithCaseCollidingSubmodules,
			want:       "GitSubmodule paths must not duplicate or case-collide",
		},
		{
			name:      "initialized submodule requires repository cwd",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validGitWorkspaceGroupObject()
				submodule := validInitializedSubmoduleShape(1, 1)
				submodule["repo_relative_cwd"] = nil
				gitWorkspaceMember(object)["submodules"] = []any{submodule}
				return object
			},
			want: "initialized GitSubmodule member repo_relative_cwd must be non-null",
		},
		{
			name:      "initialized submodule refuses unborn head",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validGitWorkspaceGroupObject()
				submodule := validInitializedSubmoduleShape(1, 1)
				head := submodule["head"].(map[string]any)
				head["mode"] = "unborn"
				head["oid"] = nil
				head["ref"] = "refs/heads/main"
				gitWorkspaceMember(object)["submodules"] = []any{submodule}
				return object
			},
			want: "initialized GitSubmodule head must be branch or detached",
		},
		{
			name:      "submodule pack and features formats agree",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validGitWorkspaceGroupObject()
				submodule := validInitializedSubmoduleShape(1, 1)
				submodule["features"].(map[string]any)["object_format"] = "sha256"
				gitWorkspaceMember(object)["submodules"] = []any{submodule}
				return object
			},
			want: "GitSubmodule pack and features object formats must match",
		},
		{
			name:      "symlink target requires string",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validTransferManifestObject("workspace_tree")
				object["entries"] = []any{map[string]any{
					"path": "link", "type": "symlink", "mode": json.Number("511"), "target": json.Number("1"),
				}}
				return object
			},
			want: "member target must be a UTF-8 string",
		},
		{
			name:      "hardlink target requires string",
			selfField: SelfManifestID,
			makeObject: func(*testing.T) map[string]any {
				object := validTransferManifestObject("workspace_tree")
				object["entries"] = []any{map[string]any{
					"path": "hard", "type": "hardlink", "mode": json.Number("420"), "target_path": json.Number("1"),
				}}
				return object
			},
			want: "member target_path must be a UTF-8 string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.makeObject(t)), test.selfField, test.want)
		})
	}
}

func TestHelperLevelRefusalsAndUnbornHeadReachBothIdentityEntries(t *testing.T) {
	t.Run("array member type", func(t *testing.T) {
		object := validSessionRecordV1Object()
		object["launch_plan"].(map[string]any)["env_names"] = map[string]any{}
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID, "member env_names must be an array")
	})

	t.Run("sorted string element type", func(t *testing.T) {
		object := validTransferManifestObject("workspace_tree")
		object["excluded_classes"] = []any{json.Number("1")}
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfManifestID, "member excluded_classes[0] must be a UTF-8 string")
	})

	t.Run("unborn head is valid when oid is null and ref names a branch", func(t *testing.T) {
		object := validGitWorkspaceGroupObject()
		head := gitWorkspaceMember(object)["head"].(map[string]any)
		head["mode"] = "unborn"
		head["oid"] = nil
		head["ref"] = "refs/heads/main"
		assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfManifestID)
	})
}

func TestSymlinkTargetLowerBoundReachesBothIdentityEntries(t *testing.T) {
	assertIdentityEntriesAcceptShape(
		t,
		mustJSON(t, transferManifestWithSymlinkTarget("a")),
		SelfManifestID,
	)
	assertIdentityEntriesRefuseWithReason(
		t,
		mustJSON(t, transferManifestWithSymlinkTarget("")),
		SelfManifestID,
		"member target must be a non-empty UTF-8 string",
	)
}

func TestSubsumedRefusalClausesRemainBlockedAtBothIdentityEntries(t *testing.T) {
	tests := []struct {
		name       string
		selfField  SelfField
		makeObject func() map[string]any
	}{
		{"remote URL non-string reaches URL grammar", SelfRecordID, func() map[string]any {
			object := primaryOwnerSessionRecord("GOAL-260830-primary")
			board := object["task_board"].(map[string]any)["board"].(map[string]any)
			board["kind"] = "remote"
			board["remote_url"] = json.Number("1")
			return object
		}},
		{"empty blob with chunk reaches coverage", SelfDescriptorID, func() map[string]any {
			return emptyBlobDescriptor(true)
		}},
		{"non-empty blob without chunk reaches exact coverage", SelfDescriptorID, func() map[string]any {
			object := validBlobDescriptorObject()
			object["chunks"] = []any{}
			return object
		}},
		{"chunk beyond total reaches final coverage inequality", SelfDescriptorID, func() map[string]any {
			object := validBlobDescriptorObject()
			object["size"] = json.Number("10")
			return object
		}},
		{"migration extensions type reaches schema shape", SelfRecordID, func() map[string]any {
			object := validSessionRecordV1Object()
			object["extensions"] = []any{}
			return object
		}},
		{"path element type reaches relative path parser", SelfManifestID, func() map[string]any {
			object := validGitWorkspaceGroupObject()
			gitWorkspaceMember(object)["agent_project_config_paths"] = []any{json.Number("1")}
			return object
		}},
		{"nullable ref type reaches git ref parser", SelfManifestID, func() map[string]any {
			object := validGitWorkspaceGroupObject()
			gitWorkspaceMember(object)["head"].(map[string]any)["ref"] = json.Number("1")
			return object
		}},
		{"nested object type reaches nested exact members", SelfRecordID, func() map[string]any {
			object := taskBoardSessionRecord("TASK-260830-8x76g1")
			object["task_board"].(map[string]any)["board"] = []any{}
			return object
		}},
		{"empty string reaches media type grammar", SelfDescriptorID, func() map[string]any {
			object := validBlobDescriptorObject()
			object["media_type"] = ""
			return object
		}},
		{"non-number uint reaches numeric parser", SelfDescriptorID, func() map[string]any {
			object := validBlobDescriptorObject()
			object["size"] = "11"
			return object
		}},
		{"fractional uint reaches numeric parser", SelfDescriptorID, func() map[string]any {
			object := validBlobDescriptorObject()
			object["size"] = json.Number("1.5")
			return object
		}},
		{"digest array element type reaches digest parser", SelfManifestID, func() map[string]any {
			object := validTransferManifestObject("composite")
			object["child_manifest_ids"] = []any{json.Number("1")}
			return object
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseShape(t, mustJSON(t, test.makeObject()), test.selfField)
		})
	}
}

func TestEveryFixtureClosedShapeMemberIsRequiredAtBothIdentityEntries(t *testing.T) {
	fixtures := []struct {
		name      string
		selfField SelfField
		object    map[string]any
	}{
		{"session direct", SelfRecordID, validSessionRecordV1Object()},
		{"session primary owner", SelfRecordID, primaryOwnerSessionRecord("GOAL-260830-primary")},
		{"session fork", SelfRecordID, sessionRecordWithForkProvenance()},
		{"session migration provenance", SelfRecordID, sessionRecordWithMigrationProvenance()},
		{"session v2 origin", SelfRecordID, validSessionRecordV2Object(validOriginProvenance())},
		{"session v2 same-provider fork", SelfRecordID, validSessionRecordV2Object(validSameProviderForkProvenance())},
		{"session v2 cross-environment clone", SelfRecordID, validSessionRecordV2Object(validCrossEnvironmentCloneProvenance("external_native"))},
		{"session v3 native adoption", SelfRecordID, validSessionRecordV3Object(validNativeAdoptionProvenance())},
		{"blob descriptor", SelfDescriptorID, validBlobDescriptorObject()},
		{"workspace tree entries", SelfManifestID, workspaceTreeWithEveryEntryVariant()},
		{"managed workspace group", SelfManifestID, validTransferManifestObject("workspace_group")},
		{"git workspace group", SelfManifestID, gitWorkspaceWithInitializedSubmodule()},
	}

	for _, fixture := range fixtures {
		paths := closedObjectMemberPaths(fixture.object)
		for _, path := range paths {
			if len(path) == 1 && (path[0] == "schema" || path[0] == "schema_version" || path[0] == string(fixture.selfField)) {
				continue
			}
			path := path
			t.Run(fixture.name+"/missing "+formatJSONPath(path), func(t *testing.T) {
				object := cloneJSONObject(t, fixture.object)
				deleteJSONObjectMemberAtPath(t, object, path)
				assertIdentityEntriesRefuseShape(t, mustJSON(t, object), fixture.selfField)
			})
		}
	}
}

func TestSessionRecordRequiredNestedObjectsRefuseNullAndScalarAtBothIdentityEntries(t *testing.T) {
	replacements := []struct {
		name  string
		value any
	}{
		{"null", nil},
		{"scalar", "not-an-object"},
	}

	for _, fixture := range sessionRecordProvenanceFixtures() {
		object := validSessionRecordV2Object(fixture.value)
		object["schema_version"] = fixture.version
		for _, path := range closedObjectMemberPaths(object) {
			if _, isObject := jsonValueAtPath(t, object, path).(map[string]any); !isObject {
				continue
			}
			path := path
			for _, replacement := range replacements {
				replacement := replacement
				t.Run(fixture.name+"/"+replacement.name+" "+formatJSONPath(path), func(t *testing.T) {
					candidate := cloneJSONObject(t, object)
					setJSONObjectMemberAtPath(t, candidate, path, replacement.value)
					assertIdentityEntriesRefuseShape(t, mustJSON(t, candidate), SelfRecordID)
				})
			}
		}
	}
}

func TestPublicIdentityRefusesNilShapeValidatorRegistryEntry(t *testing.T) {
	key := schemaIdentityKey{schema: "urn:ax:schema:session-record", version: "1.0.0"}
	original := immutableObjectShapeValidators[key]
	immutableObjectShapeValidators[key] = nil
	t.Cleanup(func() { immutableObjectShapeValidators[key] = original })

	assertIdentityEntriesRefuseWithReason(
		t,
		mustJSON(t, validSessionRecordV1Object()),
		SelfRecordID,
		"no immutable-object shape validator for urn:ax:schema:session-record@1.0.0",
	)
}

func gitWorkspaceWithCaseCollidingSubmodules(t *testing.T) map[string]any {
	t.Helper()
	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	index := member["index"].(map[string]any)
	entries := index["entries"].([]any)
	caseCollisionEntry := cloneJSONObject(t, entries[1].(map[string]any))
	caseCollisionEntry["path"] = "vendor/LIB"
	index["entries"] = []any{entries[0], caseCollisionEntry, entries[1]}
	index["entry_count"] = json.Number("3")

	submodules := member["submodules"].([]any)
	caseCollisionSubmodule := cloneJSONObject(t, submodules[0].(map[string]any))
	caseCollisionSubmodule["path"] = "vendor/LIB"
	caseCollisionSubmodule["repository_identity"] = "relux/LIB"
	caseCollisionSubmodule["sanitized_url"] = "https://example.com/LIB.git"
	member["submodules"] = []any{submodules[0], caseCollisionSubmodule}
	return object
}

func sessionRecordWithForkProvenance() map[string]any {
	object := validSessionRecordV1Object()
	object["fork_provenance"] = map[string]any{
		"source_session_id":         "0198f4c8-9f60-7077-8071-1234567890ab",
		"source_checkpoint_id":      digestWithDigit('3'),
		"source_workspace_group_id": "0198f4c8-af70-7188-8172-1234567890ab",
		"operation_id":              "0198f4c8-b080-7299-8273-1234567890ab",
		"provider_fork_mode":        "native",
		"extensions":                map[string]any{},
	}
	return object
}

func sessionRecordWithMigrationProvenance() map[string]any {
	object := validSessionRecordV1Object()
	object["extensions"] = map[string]any{
		"works.relux.ax.migrated-from": map[string]any{
			"schema_id":      "urn:ax:schema:session-record",
			"schema_version": "0.9.0",
			"object_id":      digestWithDigit('3'),
		},
	}
	return object
}

func workspaceTreeWithEveryEntryVariant() map[string]any {
	object := validTransferManifestObject("workspace_tree")
	object["entries"] = []any{
		map[string]any{"path": "dir", "type": "directory", "mode": json.Number("493")},
		map[string]any{
			"path": "file", "type": "file", "mode": json.Number("420"), "size": json.Number("1"),
			"blob_id": digestWithDigit('1'), "blob_descriptor_id": digestWithDigit('2'),
		},
		map[string]any{"path": "hard", "type": "hardlink", "mode": json.Number("420"), "target_path": "file"},
		map[string]any{"path": "link", "type": "symlink", "mode": json.Number("511"), "target": "file"},
	}
	return object
}

func gitWorkspaceWithInitializedSubmodule() map[string]any {
	object := validGitWorkspaceGroupObject()
	gitWorkspaceMember(object)["submodules"] = []any{validInitializedSubmoduleShape(1, 1)}
	return object
}

// openMapMembers are the members whose KEYS are data rather than schema members,
// so removing one is legitimately accepted and must not be swept as a missing
// closed member. Section 3 declares extensions as
// `map(reverse-dns,ExtensionValue)[0..64]`; Section 5.5 declares of
// opaque_identity that "Map keys are data rather than schema members"; the
// Launch Plan declares env_literals as a destination-local environment map.
// Their key grammars are proven by the declared-grammar inventory instead.
var openMapMembers = map[string]string{
	"extensions":      "reverse-DNS extension map",
	"env_literals":    "destination-local environment map",
	"opaque_identity": "provider-defined identity data map",
}

func closedObjectMemberPaths(object map[string]any) [][]string {
	var paths [][]string
	var visit func(any, []string)
	visit = func(value any, prefix []string) {
		switch typed := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			openMap := len(prefix) > 0 && openMapMembers[prefix[len(prefix)-1]] != ""
			for _, key := range keys {
				path := append(append([]string{}, prefix...), key)
				if !openMap {
					paths = append(paths, path)
				}
				if key == "env_literals" {
					continue
				}
				if key == "extensions" {
					if extensions, ok := typed[key].(map[string]any); ok {
						if provenance, ok := extensions["works.relux.ax.migrated-from"]; ok {
							visit(provenance, append(path, "works.relux.ax.migrated-from"))
						}
					}
					continue
				}
				visit(typed[key], path)
			}
		case []any:
			for index, member := range typed {
				visit(member, append(append([]string{}, prefix...), fmt.Sprintf("[%d]", index)))
			}
		}
	}
	visit(object, nil)
	return paths
}

func deleteJSONObjectMemberAtPath(t *testing.T, root map[string]any, path []string) {
	t.Helper()
	var current any = root
	for _, component := range path[:len(path)-1] {
		if strings.HasPrefix(component, "[") {
			var index int
			if _, err := fmt.Sscanf(component, "[%d]", &index); err != nil {
				t.Fatal(err)
			}
			current = current.([]any)[index]
			continue
		}
		current = current.(map[string]any)[component]
	}
	delete(current.(map[string]any), path[len(path)-1])
}

func jsonValueAtPath(t *testing.T, root map[string]any, path []string) any {
	t.Helper()
	var current any = root
	for _, component := range path {
		if strings.HasPrefix(component, "[") {
			var index int
			if _, err := fmt.Sscanf(component, "[%d]", &index); err != nil {
				t.Fatal(err)
			}
			current = current.([]any)[index]
			continue
		}
		current = current.(map[string]any)[component]
	}
	return current
}

func setJSONObjectMemberAtPath(t *testing.T, root map[string]any, path []string, value any) {
	t.Helper()
	var current any = root
	for _, component := range path[:len(path)-1] {
		if strings.HasPrefix(component, "[") {
			var index int
			if _, err := fmt.Sscanf(component, "[%d]", &index); err != nil {
				t.Fatal(err)
			}
			current = current.([]any)[index]
			continue
		}
		current = current.(map[string]any)[component]
	}
	current.(map[string]any)[path[len(path)-1]] = value
}

func formatJSONPath(path []string) string {
	var formatted strings.Builder
	for _, component := range path {
		if strings.HasPrefix(component, "[") {
			formatted.WriteString(component)
			continue
		}
		if formatted.Len() > 0 {
			formatted.WriteByte('.')
		}
		formatted.WriteString(component)
	}
	return formatted.String()
}

func assertIdentityEntriesRefuseWithReason(t *testing.T, input []byte, selfField SelfField, want string) {
	t.Helper()
	if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), want) {
		t.Fatalf("CalculateObjectIdentity(%s malformed shape) error = %v, want identity refusal containing %q", selfField, err, want)
	}
	claimed := withCorrectIdentityClaimForTest(t, input, selfField)
	if _, _, err := VerifyObjectIdentity(claimed); err == nil || !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), want) {
		t.Fatalf("VerifyObjectIdentity(%s malformed shape with correct omit-self claim) error = %v, want identity refusal containing %q", selfField, err, want)
	}
}
