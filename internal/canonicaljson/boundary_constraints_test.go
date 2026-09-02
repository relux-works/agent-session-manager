package canonicaljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// boundaryConstraintCase is one at-limit acceptance paired with one over-limit
// refusal, both driven through CalculateObjectIdentity and VerifyObjectIdentity.
//
// claims names the derived bound obligations this case discharges. The
// obligation keys are produced by deriveBoundCallSites in declared_bounds_test.go
// and asserted exactly there, so a case cannot claim a bound that no production
// call site declares, and a production bound cannot exist without a claim.
type boundaryConstraintCase struct {
	name      string
	selfField SelfField
	claims    []boundObligation
	atLimit   func() map[string]any
	overLimit func() map[string]any
}

func TestDeclaredBoundaryConstraintsReachBothIdentityEntries(t *testing.T) {
	for _, test := range declaredBoundaryConstraintCases() {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, mustJSON(t, test.atLimit()), test.selfField)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, test.overLimit()), test.selfField)
		})
	}
}

func declaredBoundaryConstraintCases() []boundaryConstraintCase {
	return []boundaryConstraintCase{
		{
			name:      "reverse DNS key requires a dot",
			selfField: SelfRecordID,
			atLimit: func() map[string]any {
				return genericExtensionIdentityObject(map[string]any{"a.b": true})
			},
			overLimit: func() map[string]any {
				return genericExtensionIdentityObject(map[string]any{"nodot": true})
			},
		},
		{
			name:      "extension object nesting depth 4",
			selfField: SelfRecordID,
			atLimit: func() map[string]any {
				return genericExtensionIdentityObject(map[string]any{"a.depth": nestedExtensionObjects(4)})
			},
			overLimit: func() map[string]any {
				return genericExtensionIdentityObject(map[string]any{"a.depth": nestedExtensionObjects(5)})
			},
		},
		{
			name:      "migration provenance canonical semver",
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return migrationProvenanceRecord("1.0.0") },
			overLimit: func() map[string]any { return migrationProvenanceRecord("01.0.0") },
		},
		{
			name:      "launch argv count 128",
			claims:    []boundObligation{{key: "validateSessionLaunchPlan|requireArray|argv|-..128", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return sessionRecordWithArgv(repeatedValues("x", 128)) },
			overLimit: func() map[string]any { return sessionRecordWithArgv(repeatedValues("x", 129)) },
		},
		{
			name:      "launch argv canonical size 65536 bytes",
			claims:    []boundObligation{{key: "validateSessionLaunchPlan|canonicalByteBound|Session Record Launch Plan argv|-..65536", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit: func() map[string]any {
				return sessionRecordWithArgv(encodedArgvBoundary(4_092))
			},
			overLimit: func() map[string]any {
				return sessionRecordWithArgv(encodedArgvBoundary(4_093))
			},
		},
		{
			name:      "extensions object canonical size 65536 bytes",
			claims:    []boundObligation{{key: "validateExtensionsObject|canonicalByteBound|extensions object|-..65536", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit: func() map[string]any {
				return genericExtensionIdentityObject(extensionsObjectOfCanonicalBytes(65_536))
			},
			overLimit: func() map[string]any {
				return genericExtensionIdentityObject(extensionsObjectOfCanonicalBytes(65_537))
			},
		},
		{
			name:      "launch env_names count 64",
			claims:    []boundObligation{{key: "validateSessionLaunchPlan|requireArray|env_names|-..64", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return sessionRecordWithEnvNames(environmentNames(64)) },
			overLimit: func() map[string]any { return sessionRecordWithEnvNames(environmentNames(65)) },
		},
		{
			name:      "launch env_names element grammar 128 characters",
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return sessionRecordWithEnvNames([]any{strings.Repeat("A", 128)}) },
			overLimit: func() map[string]any { return sessionRecordWithEnvNames([]any{strings.Repeat("A", 129)}) },
		},
		{
			name:      "launch env_literals count 64",
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return sessionRecordWithEnvLiterals(environmentLiterals(64)) },
			overLimit: func() map[string]any { return sessionRecordWithEnvLiterals(environmentLiterals(65)) },
		},
		{
			name:      "launch env_literals value 4096 bytes",
			selfField: SelfRecordID,
			atLimit: func() map[string]any {
				return sessionRecordWithEnvLiterals(map[string]any{"VALUE": strings.Repeat("x", 4_096)})
			},
			overLimit: func() map[string]any {
				return sessionRecordWithEnvLiterals(map[string]any{"VALUE": strings.Repeat("x", 4_097)})
			},
		},
		{
			name:      "task element identifier 128 bytes",
			claims:    []boundObligation{{key: "validateSessionTaskBoardReference|requirePrintableByteBoundedString|task_element_id|1..128", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return taskBoardSessionRecord(strings.Repeat("a", 128)) },
			overLimit: func() map[string]any { return taskBoardSessionRecord(strings.Repeat("a", 129)) },
		},
		{
			name:      "task element identifier printable non-control",
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return taskBoardSessionRecord("TASK-1") },
			overLimit: func() map[string]any { return taskBoardSessionRecord("TASK\n1") },
		},
		{
			name:      "board goal identifier 128 characters",
			claims:    []boundObligation{{key: "validateSessionBoardGoal|requireBoundedString|goal_id|1..128", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return primaryOwnerSessionRecord(strings.Repeat("界", 128)) },
			overLimit: func() map[string]any { return primaryOwnerSessionRecord(strings.Repeat("界", 129)) },
		},
		{
			name:      "blob media type lowercase ASCII type",
			selfField: SelfDescriptorID,
			atLimit: func() map[string]any {
				object := validBlobDescriptorObject()
				object["media_type"] = "application/octet-stream"
				return object
			},
			overLimit: func() map[string]any {
				object := validBlobDescriptorObject()
				object["media_type"] = "Application/octet-stream"
				return object
			},
		},
		{
			name:      "blob media type lowercase ASCII subtype",
			selfField: SelfDescriptorID,
			atLimit: func() map[string]any {
				object := validBlobDescriptorObject()
				object["media_type"] = "application/octet-stream"
				return object
			},
			overLimit: func() map[string]any {
				object := validBlobDescriptorObject()
				object["media_type"] = "application/Octet-Stream"
				return object
			},
		},
		{
			name:      "blob chunk count 32768",
			claims:    []boundObligation{{key: "validateBlobDescriptor|requireArray|chunks|-..32768", direction: boundMaximum}},
			selfField: SelfDescriptorID,
			atLimit:   func() map[string]any { return blobDescriptorWithChunks(32_768) },
			overLimit: func() map[string]any { return blobDescriptorWithChunks(32_769) },
		},
		{
			name:      "transfer manifest entry count 65536",
			claims:    []boundObligation{{key: "validateTransferManifest|requireArray|entries|-..65536", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithEntries(65_536) },
			overLimit: func() map[string]any { return transferManifestWithEntries(65_537) },
		},
		{
			name:      "manifest entry mode 4095",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithMode(4_095) },
			overLimit: func() map[string]any { return transferManifestWithMode(4_096) },
		},
		{
			name:      "symlink target rejects Windows separators",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithSymlinkTarget("sub/child/escape") },
			overLimit: func() map[string]any { return transferManifestWithSymlinkTarget(`sub\..\..\escape`) },
		},
		{
			name:      "symlink target rejects NUL",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithSymlinkTarget("sub/child/escape") },
			overLimit: func() map[string]any { return transferManifestWithSymlinkTarget("sub/\x00escape") },
		},
		{
			name:      "symlink target rejects absolute slash",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithSymlinkTarget("sub/child/escape") },
			overLimit: func() map[string]any { return transferManifestWithSymlinkTarget("/etc/passwd") },
		},
		{
			name:      "symlink target rejects Windows drive roots",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithSymlinkTarget("C") },
			overLimit: func() map[string]any { return transferManifestWithSymlinkTarget("C:") },
		},
		{
			name:      "manifest entries require strict path order from the second element",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return transferManifestWithDirectoryPaths("a", "b") },
			overLimit: func() map[string]any { return transferManifestWithDirectoryPaths("b", "a") },
		},
		{
			name:      "workspace snapshot members require strict identifier order from the second element",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return workspaceGroupWithManagedMembers(2) },
			overLimit: func() map[string]any {
				object := workspaceGroupWithManagedMembers(2)
				members := object["workspace_snapshot"].(map[string]any)["members"].([]any)
				members[0], members[1] = members[1], members[0]
				return object
			},
		},
		{
			name:      "child manifest identifier count 1024",
			claims:    []boundObligation{{key: "validateTransferManifest|requireArray|child_manifest_ids|-..1024", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return compositeManifestWithChildren(1_024) },
			overLimit: func() map[string]any { return compositeManifestWithChildren(1_025) },
		},
		{
			name:      "excluded class count 128",
			claims:    []boundObligation{{key: "validateTransferManifest|requireArray|excluded_classes|-..128", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return workspaceTreeWithExcludedClasses(128) },
			overLimit: func() map[string]any { return workspaceTreeWithExcludedClasses(129) },
		},
		{
			name:      "Git remote count 16",
			claims:    []boundObligation{{key: "validateGitRemotes|requireArray|remotes|-..16", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithRemotes(16) },
			overLimit: func() map[string]any { return gitWorkspaceWithRemotes(17) },
		},
		{
			name:      "workspace snapshot member count 256",
			claims:    []boundObligation{{key: "validateWorkspaceSnapshot|requireArray|members|-..256", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return workspaceGroupWithManagedMembers(256) },
			overLimit: func() map[string]any { return workspaceGroupWithManagedMembers(257) },
		},
		{
			name:      "required filter name count 64",
			claims:    []boundObligation{{key: "validateGitFeatures|requireArray|required_filter_names|-..64", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithRequiredFilters(64) },
			overLimit: func() map[string]any { return gitWorkspaceWithRequiredFilters(65) },
		},
		{
			name:      "agent project config path count 256",
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireSortedUniquePaths|agent_project_config_paths|-..256", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithProjectPaths(256) },
			overLimit: func() map[string]any { return gitWorkspaceWithProjectPaths(257) },
		},
		{
			name:      "blob chunk size 4194304",
			selfField: SelfDescriptorID,
			atLimit:   func() map[string]any { return blobDescriptorWithChunkSize(maxChunkSize) },
			overLimit: func() map[string]any { return blobDescriptorWithChunkSize(maxChunkSize + 1) },
		},
		{
			name:      "empty blob descriptor has no chunks",
			selfField: SelfDescriptorID,
			atLimit:   func() map[string]any { return emptyBlobDescriptor(false) },
			overLimit: func() map[string]any { return emptyBlobDescriptor(true) },
		},
		{
			name:      "Git index version minimum 2",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithIndexVersion(2) },
			overLimit: func() map[string]any { return gitWorkspaceWithIndexVersion(1) },
		},
		{
			name:      "Git index entry mode uint32 maximum",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithIndexEntryMode(1<<32 - 1) },
			overLimit: func() map[string]any { return gitWorkspaceWithIndexEntryMode(1 << 32) },
		},
		{
			name:      "board logical identifier 128 characters",
			claims:    []boundObligation{{key: "validateSessionBoardIdentity|requireBoundedString|logical_id|1..128", direction: boundMaximum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return taskBoardRecordWithLogicalID(strings.Repeat("a", 128)) },
			overLimit: func() map[string]any { return taskBoardRecordWithLogicalID(strings.Repeat("a", 129)) },
		},
		{
			name:      "managed tree project config path count 256",
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireSortedUniquePaths|agent_project_config_paths|-..256", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return managedTreeWithProjectPaths(256) },
			overLimit: func() map[string]any { return managedTreeWithProjectPaths(257) },
		},
		{
			name:      "Git submodule repository identity 256 characters",
			claims:    []boundObligation{{key: "validateGitSubmodule|requireBoundedString|repository_identity|1..256", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithSubmoduleIdentity("s" + strings.Repeat("界", 255)) },
			overLimit: func() map[string]any { return gitWorkspaceWithSubmoduleIdentity("s" + strings.Repeat("界", 256)) },
		},
		{
			name:      "Git submodule project config path count 256",
			claims:    []boundObligation{{key: "validateGitSubmodule|requireSortedUniquePaths|agent_project_config_paths|-..256", direction: boundMaximum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithInitializedSubmoduleProjectPaths(256) },
			overLimit: func() map[string]any { return gitWorkspaceWithInitializedSubmoduleProjectPaths(257) },
		},
		{
			name:      "task element identifier non-empty",
			claims:    []boundObligation{{key: "validateSessionTaskBoardReference|requirePrintableByteBoundedString|task_element_id|1..128", direction: boundMinimum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return taskBoardSessionRecord("T") },
			overLimit: func() map[string]any { return taskBoardSessionRecord("") },
		},
		{
			name:      "board goal identifier non-empty",
			claims:    []boundObligation{{key: "validateSessionBoardGoal|requireBoundedString|goal_id|1..128", direction: boundMinimum}},
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return primaryOwnerSessionRecord("g") },
			overLimit: func() map[string]any { return primaryOwnerSessionRecord("") },
		},
		{
			name:      "launch argv element non-empty",
			selfField: SelfRecordID,
			atLimit:   func() map[string]any { return sessionRecordWithArgv([]any{"x"}) },
			overLimit: func() map[string]any { return sessionRecordWithArgv([]any{""}) },
		},
		{
			name:      "managed tree identity non-empty",
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireBoundedString|tree_identity|1..256", direction: boundMinimum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return managedTreeWithIdentity("x") },
			overLimit: func() map[string]any { return managedTreeWithIdentity("") },
		},
		{
			name:      "Git workspace repository identity non-empty",
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireBoundedString|repository_identity|1..256", direction: boundMinimum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithRepositoryIdentity("x") },
			overLimit: func() map[string]any { return gitWorkspaceWithRepositoryIdentity("") },
		},
		{
			name:      "Git remote name non-empty",
			claims:    []boundObligation{{key: "validateGitRemote|requireBoundedString|name|1..128", direction: boundMinimum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithRemoteName("x") },
			overLimit: func() map[string]any { return gitWorkspaceWithRemoteName("") },
		},
		{
			name:      "Git submodule repository identity non-empty",
			claims:    []boundObligation{{key: "validateGitSubmodule|requireBoundedString|repository_identity|1..256", direction: boundMinimum}},
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return gitWorkspaceWithSubmoduleIdentity("s") },
			overLimit: func() map[string]any { return gitWorkspaceWithSubmoduleIdentity("") },
		},
		{
			name:      "board logical identifier non-empty",
			selfField: SelfRecordID,
			claims:    []boundObligation{{key: "validateSessionBoardIdentity|requireBoundedString|logical_id|1..128", direction: boundMinimum}},
			atLimit:   func() map[string]any { return taskBoardRecordWithLogicalID("a") },
			overLimit: func() map[string]any { return taskBoardRecordWithLogicalID("") },
		},
		{
			name:      "Git workspace repository identity 256 characters",
			selfField: SelfManifestID,
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireBoundedString|repository_identity|1..256", direction: boundMaximum}},
			atLimit:   func() map[string]any { return gitWorkspaceWithRepositoryIdentity(strings.Repeat("界", 256)) },
			overLimit: func() map[string]any { return gitWorkspaceWithRepositoryIdentity(strings.Repeat("界", 257)) },
		},
		{
			name:      "managed tree identity 256 characters",
			selfField: SelfManifestID,
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireBoundedString|tree_identity|1..256", direction: boundMaximum}},
			atLimit:   func() map[string]any { return managedTreeWithIdentity(strings.Repeat("界", 256)) },
			overLimit: func() map[string]any { return managedTreeWithIdentity(strings.Repeat("界", 257)) },
		},
		{
			name:      "Git remote name 128 characters",
			selfField: SelfManifestID,
			claims:    []boundObligation{{key: "validateGitRemote|requireBoundedString|name|1..128", direction: boundMaximum}},
			atLimit:   func() map[string]any { return gitWorkspaceWithRemoteName(strings.Repeat("界", 128)) },
			overLimit: func() map[string]any { return gitWorkspaceWithRemoteName(strings.Repeat("界", 129)) },
		},
		{
			name:      "workspace snapshot member submodule count 256",
			selfField: SelfManifestID,
			claims:    []boundObligation{{key: "validateWorkspaceSnapshotMember|requireArray|submodules|-..256", direction: boundMaximum}},
			atLimit:   func() map[string]any { return gitWorkspaceWithSubmoduleCount(256) },
			overLimit: func() map[string]any { return gitWorkspaceWithSubmoduleCount(257) },
		},
		{
			name:      "clone source native session identifier 512 characters",
			selfField: SelfRecordID,
			claims:    []boundObligation{{key: "validateSessionCrossEnvironmentCloneProvenance|requirePrintableBoundedString|source_native_session_id|1..512", direction: boundMaximum}},
			atLimit:   func() map[string]any { return cloneProvenanceWithNativeSessionID(strings.Repeat("界", 512)) },
			overLimit: func() map[string]any { return cloneProvenanceWithNativeSessionID(strings.Repeat("界", 513)) },
		},
		{
			name:      "clone source native session identifier non-empty",
			selfField: SelfRecordID,
			claims:    []boundObligation{{key: "validateSessionCrossEnvironmentCloneProvenance|requirePrintableBoundedString|source_native_session_id|1..512", direction: boundMinimum}},
			atLimit:   func() map[string]any { return cloneProvenanceWithNativeSessionID("n") },
			overLimit: func() map[string]any { return cloneProvenanceWithNativeSessionID("") },
		},
		{
			name:      "workspace snapshot members non-empty",
			selfField: SelfManifestID,
			atLimit:   func() map[string]any { return workspaceGroupWithMemberCount(1) },
			overLimit: func() map[string]any { return workspaceGroupWithMemberCount(0) },
		},
	}
}

func TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries(t *testing.T) {
	assertIdentityEntriesAcceptShape(t, mustJSON(t, blobDescriptorWithTrailingChunkSize(1)), SelfDescriptorID)

	input := mustJSON(t, blobDescriptorWithTrailingChunkSize(0))
	want := "BlobChunk[1] size must lie in [1, 4194304]"
	if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), want) {
		t.Fatalf("CalculateObjectIdentity(trailing zero-size BlobChunk) error = %v, want identity refusal containing %q", err, want)
	}

	claimed := withCorrectIdentityClaimForTest(t, input, SelfDescriptorID)
	if _, _, err := VerifyObjectIdentity(claimed); err == nil || !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), want) {
		t.Fatalf("VerifyObjectIdentity(trailing zero-size BlobChunk) error = %v, want identity refusal containing %q", err, want)
	}
}

func TestGitIndexEntryCountBoundaryIsBoundBelowThePublicObjectSizeGate(t *testing.T) {
	atLimit := gitIndexWithEntries(65_536)
	if _, err := validateGitIndex(atLimit); err != nil {
		t.Fatalf("validateGitIndex(65536 entries) error = %v, want declared-bound acceptance", err)
	}

	overLimit := gitIndexWithEntries(65_537)
	if _, err := validateGitIndex(overLimit); err == nil {
		t.Fatal("validateGitIndex(65537 entries) error = nil, want declared-bound refusal")
	}

	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	member["index"] = atLimit
	member["submodules"] = []any{}
	input := mustJSON(t, object)
	t.Logf("65536-entry GitIndex identity encodes to %d bytes; public limit is 5242880", len(input))
	if len(input) <= 5_242_880 {
		t.Fatalf("65536-entry GitIndex identity encodes to %d bytes, expected outer identity-size refusal", len(input))
	}
	assertIdentityEntriesRefuseShape(t, input, SelfManifestID)
}

func migrationProvenanceRecord(version string) map[string]any {
	object := validSessionRecordV1Object()
	object["extensions"] = map[string]any{
		"works.relux.ax.migrated-from": map[string]any{
			"schema_id":      "urn:ax:schema:session-record",
			"schema_version": version,
			"object_id":      digestWithDigit('3'),
		},
	}
	return object
}

func sessionRecordWithArgv(argv []any) map[string]any {
	object := validSessionRecordV1Object()
	object["launch_plan"].(map[string]any)["argv"] = argv
	return object
}

// extensionsObjectOfCanonicalBytes returns a one-member extensions object whose
// canonical encoding is exactly target bytes. A canonical one-member object is
// "{" plus the quoted key, ":", the quoted value and "}", so its length is the
// key length plus the value length plus seven.
func extensionsObjectOfCanonicalBytes(target int) map[string]any {
	const key = "works.relux.bytes"
	return map[string]any{key: strings.Repeat("x", target-len(key)-7)}
}

func encodedArgvBoundary(lastLength int) []any {
	argv := make([]any, 16)
	for index := range 15 {
		argv[index] = strings.Repeat("x", 4_093)
	}
	argv[15] = strings.Repeat("x", lastLength)
	return argv
}

func sessionRecordWithEnvNames(names []any) map[string]any {
	object := validSessionRecordV1Object()
	launchPlan := object["launch_plan"].(map[string]any)
	launchPlan["env_names"] = names
	launchPlan["env_literals"] = map[string]any{}
	return object
}

func environmentNames(count int) []any {
	names := make([]any, count)
	for index := range count {
		names[index] = fmt.Sprintf("A_%03d", index)
	}
	return names
}

func sessionRecordWithEnvLiterals(literals map[string]any) map[string]any {
	object := validSessionRecordV1Object()
	launchPlan := object["launch_plan"].(map[string]any)
	launchPlan["env_names"] = []any{}
	launchPlan["env_literals"] = literals
	return object
}

func environmentLiterals(count int) map[string]any {
	literals := make(map[string]any, count)
	for index := range count {
		literals[fmt.Sprintf("VALUE_%03d", index)] = "x"
	}
	return literals
}

func taskBoardSessionRecord(taskElementID string) map[string]any {
	object := validSessionRecordV1Object()
	object["kind"] = "task_board"
	object["task_board"] = map[string]any{
		"bridge_protocol_version": "1.0.0",
		"board": map[string]any{
			"kind":       "local",
			"logical_id": "agent-session-manager",
			"remote_url": nil,
			"extensions": map[string]any{},
		},
		"task_element_id":     taskElementID,
		"launch_mode":         "tracked_prompt",
		"manager_session_ref": nil,
		"board_goal":          nil,
		"native_goal_binding": "prompt",
		"extensions":          map[string]any{},
	}
	return object
}

func primaryOwnerSessionRecord(goalID string) map[string]any {
	object := taskBoardSessionRecord("TASK-260830-8x76g1")
	reference := object["task_board"].(map[string]any)
	reference["launch_mode"] = "primary_owner"
	reference["native_goal_binding"] = "bound"
	reference["board_goal"] = map[string]any{
		"schema":     "board-goal-v2",
		"goal_id":    goalID,
		"revision":   json.Number("1"),
		"extensions": map[string]any{},
	}
	return object
}

func blobDescriptorWithChunks(count int) map[string]any {
	object := validBlobDescriptorObject()
	chunks := make([]any, count)
	for index := range count {
		chunks[index] = map[string]any{
			"index":    json.Number(strconv.Itoa(index)),
			"offset":   json.Number(strconv.FormatInt(int64(index)*maxChunkSize, 10)),
			"size":     json.Number(strconv.FormatInt(maxChunkSize, 10)),
			"chunk_id": digestWithDigit('1'),
		}
	}
	object["size"] = json.Number(strconv.FormatInt(int64(count)*maxChunkSize, 10))
	object["chunks"] = chunks
	return object
}

func blobDescriptorWithChunkSize(size int) map[string]any {
	object := validBlobDescriptorObject()
	object["size"] = json.Number(strconv.Itoa(size))
	firstChunk(object)["size"] = json.Number(strconv.Itoa(size))
	return object
}

func emptyBlobDescriptor(withChunk bool) map[string]any {
	object := validBlobDescriptorObject()
	object["size"] = json.Number("0")
	if !withChunk {
		object["chunks"] = []any{}
	}
	return object
}

func blobDescriptorWithTrailingChunkSize(size int) map[string]any {
	object := validBlobDescriptorObject()
	object["size"] = json.Number(strconv.Itoa(maxChunkSize + size))
	object["chunks"] = []any{
		map[string]any{
			"index":    json.Number("0"),
			"offset":   json.Number("0"),
			"size":     json.Number(strconv.Itoa(maxChunkSize)),
			"chunk_id": digestWithDigit('1'),
		},
		map[string]any{
			"index":    json.Number("1"),
			"offset":   json.Number(strconv.Itoa(maxChunkSize)),
			"size":     json.Number(strconv.Itoa(size)),
			"chunk_id": digestWithDigit('2'),
		},
	}
	return object
}

func transferManifestWithEntries(count int) map[string]any {
	object := validTransferManifestObject("workspace_tree")
	entries := make([]any, count)
	for index := range count {
		entries[index] = map[string]any{
			"path": fmt.Sprintf("p/%05d", index),
			"type": "directory",
			"mode": json.Number("493"),
		}
	}
	object["entries"] = entries
	return object
}

func transferManifestWithMode(mode int) map[string]any {
	object := validTransferManifestObject("workspace_tree")
	object["entries"] = []any{map[string]any{
		"path": "directory",
		"type": "directory",
		"mode": json.Number(strconv.Itoa(mode)),
	}}
	return object
}

func transferManifestWithSymlinkTarget(target string) map[string]any {
	object := validTransferManifestObject("workspace_tree")
	object["entries"] = []any{map[string]any{
		"path":   "current/link",
		"type":   "symlink",
		"mode":   json.Number("511"),
		"target": target,
	}}
	return object
}

func transferManifestWithDirectoryPaths(paths ...string) map[string]any {
	object := validTransferManifestObject("workspace_tree")
	entries := make([]any, len(paths))
	for index, pathValue := range paths {
		entries[index] = map[string]any{
			"path": pathValue,
			"type": "directory",
			"mode": json.Number("493"),
		}
	}
	object["entries"] = entries
	return object
}

func compositeManifestWithChildren(count int) map[string]any {
	object := validTransferManifestObject("composite")
	children := make([]any, count)
	for index := range count {
		children[index] = fmt.Sprintf("sha256:%064x", index)
	}
	object["child_manifest_ids"] = children
	return object
}

func workspaceTreeWithExcludedClasses(count int) map[string]any {
	object := validTransferManifestObject("workspace_tree")
	object["excluded_classes"] = numberedStrings("class", count, 3)
	return object
}

func gitWorkspaceWithRemotes(count int) map[string]any {
	object := validGitWorkspaceGroupObject()
	remotes := make([]any, count)
	for index := range count {
		remotes[index] = map[string]any{
			"name":      fmt.Sprintf("r%02d", index),
			"fetch_url": fmt.Sprintf("https://example.test/r%02d.git", index),
			"push_url":  nil,
		}
	}
	gitWorkspaceMember(object)["remotes"] = remotes
	return object
}

func workspaceGroupWithManagedMembers(count int) map[string]any {
	object := validTransferManifestObject("workspace_group")
	members := make([]any, count)
	for index := range count {
		members[index] = map[string]any{
			"workspace_id":               fmt.Sprintf("0198f4c8-7d40-7e55-8e6f-%012x", index),
			"kind":                       "managed_tree",
			"group_relative_path":        fmt.Sprintf("tree-%03d", index),
			"tree_identity":              fmt.Sprintf("relux/tree-%03d", index),
			"tree_manifest_id":           digestWithDigit('5'),
			"repo_relative_cwd":          ".",
			"agent_project_config_paths": []any{},
			"materialization_policy":     "separate_copy",
		}
	}
	object["workspace_snapshot"].(map[string]any)["members"] = members
	return object
}

func gitWorkspaceWithRequiredFilters(count int) map[string]any {
	object := validGitWorkspaceGroupObject()
	features := gitWorkspaceMember(object)["features"].(map[string]any)
	features["required_filter_names"] = numberedStrings("filter", count, 2)
	return object
}

func gitWorkspaceWithProjectPaths(count int) map[string]any {
	object := validGitWorkspaceGroupObject()
	gitWorkspaceMember(object)["agent_project_config_paths"] = numberedStrings("config", count, 3)
	return object
}

func gitWorkspaceWithIndexVersion(version int) map[string]any {
	object := validGitWorkspaceGroupObject()
	gitWorkspaceMember(object)["index"].(map[string]any)["version"] = json.Number(strconv.Itoa(version))
	return object
}

func gitWorkspaceWithIndexEntryMode(mode uint64) map[string]any {
	object := validGitWorkspaceGroupObject()
	index := gitWorkspaceMember(object)["index"].(map[string]any)
	index["entries"].([]any)[0].(map[string]any)["mode"] = json.Number(strconv.FormatUint(mode, 10))
	return object
}

func taskBoardRecordWithLogicalID(logicalID string) map[string]any {
	object := taskBoardSessionRecord("TASK-260830-8x76g1")
	reference := object["task_board"].(map[string]any)
	reference["board"].(map[string]any)["logical_id"] = logicalID
	return object
}

func managedTreeWithProjectPaths(count int) map[string]any {
	object := validTransferManifestObject("workspace_group")
	gitWorkspaceMember(object)["agent_project_config_paths"] = numberedStrings("config", count, 3)
	return object
}

func managedTreeWithIdentity(identity string) map[string]any {
	object := validTransferManifestObject("workspace_group")
	gitWorkspaceMember(object)["tree_identity"] = identity
	return object
}

func gitWorkspaceWithRepositoryIdentity(identity string) map[string]any {
	object := validGitWorkspaceGroupObject()
	gitWorkspaceMember(object)["repository_identity"] = identity
	return object
}

func gitWorkspaceWithRemoteName(name string) map[string]any {
	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	member["remotes"].([]any)[0].(map[string]any)["name"] = name
	return object
}

func gitWorkspaceWithSubmoduleIdentity(identity string) map[string]any {
	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	member["submodules"].([]any)[0].(map[string]any)["repository_identity"] = identity
	return object
}

func gitWorkspaceWithInitializedSubmoduleProjectPaths(count int) map[string]any {
	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	submodule := validInitializedSubmoduleShape(1, 1)
	submodule["agent_project_config_paths"] = numberedStrings("config", count, 3)
	member["submodules"] = []any{submodule}
	return object
}

func workspaceGroupWithMemberCount(count int) map[string]any {
	object := validTransferManifestObject("workspace_group")
	snapshot := object["workspace_snapshot"].(map[string]any)
	if count == 0 {
		snapshot["members"] = []any{}
	}
	return object
}

func numberedStrings(prefix string, count, width int) []any {
	values := make([]any, count)
	for index := range count {
		values[index] = fmt.Sprintf("%s-%0*d", prefix, width, index)
	}
	return values
}

func repeatedValues(value string, count int) []any {
	values := make([]any, count)
	for index := range count {
		values[index] = value
	}
	return values
}

func nestedExtensionObjects(depth int) any {
	var value any = "value"
	for range depth {
		value = map[string]any{"value": value}
	}
	return value
}

func gitIndexWithEntries(count int) map[string]any {
	entries := make([]any, count)
	for index := range count {
		entries[index] = map[string]any{
			"path":             fmt.Sprintf("f%05d", index),
			"stage":            json.Number("0"),
			"mode":             json.Number("33188"),
			"oid":              "sha1:" + strings.Repeat("2", 40),
			"intent_to_add":    false,
			"skip_worktree":    false,
			"assume_unchanged": false,
			"fsmonitor_valid":  false,
		}
	}
	return map[string]any{
		"format":             "git_index",
		"version":            json.Number("2"),
		"blob_id":            digestWithDigit('5'),
		"blob_descriptor_id": digestWithDigit('6'),
		"entries":            entries,
		"entry_count":        json.Number(strconv.Itoa(count)),
	}
}

// cloneProvenanceWithNativeSessionID builds a v2 Session Record whose
// cross-environment-clone provenance carries a source native session identifier
// of the given value.
func cloneProvenanceWithNativeSessionID(identity string) map[string]any {
	provenance := validCrossEnvironmentCloneProvenance("external_native")
	provenance["source_native_session_id"] = identity
	return validSessionRecordV2Object(provenance)
}

// gitWorkspaceWithSubmoduleCount builds a Git workspace member carrying count
// uninitialized submodules, each matched by its stage-0 gitlink index entry as
// validateGitSubmodule requires.
func gitWorkspaceWithSubmoduleCount(count int) map[string]any {
	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	entries := make([]any, 0, count)
	submodules := make([]any, 0, count)
	for index := range count {
		entries = append(entries, gitlinkIndexEntry(index))
		submodules = append(submodules, uninitializedSubmodule(index))
	}
	index := member["index"].(map[string]any)
	index["entries"] = entries
	index["entry_count"] = json.Number(strconv.Itoa(count))
	member["submodules"] = submodules
	return object
}

// gitWorkspaceWithNestedSubmoduleCount builds a Git workspace member carrying
// one initialized submodule that itself carries count nested submodules.
func gitWorkspaceWithNestedSubmoduleCount(count int) map[string]any {
	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	parent := validInitializedSubmoduleShape(1, 1)
	parent["path"] = "modules/000"
	// The parent identity must not repeat inside its own subtree: the
	// submodule tree is required to be acyclic by repository_identity.
	parent["repository_identity"] = "relux/parent-module"
	nestedEntries := make([]any, 0, count)
	nested := make([]any, 0, count)
	for index := range count {
		nestedEntries = append(nestedEntries, gitlinkIndexEntry(index))
		nested = append(nested, uninitializedSubmodule(index))
	}
	parentIndex := parent["index"].(map[string]any)
	parentIndex["entries"] = nestedEntries
	parentIndex["entry_count"] = json.Number(strconv.Itoa(count))
	parent["submodules"] = nested

	memberIndex := member["index"].(map[string]any)
	memberIndex["entries"] = []any{gitlinkIndexEntry(0)}
	memberIndex["entry_count"] = json.Number("1")
	member["submodules"] = []any{parent}
	return object
}

func gitlinkIndexEntry(index int) map[string]any {
	return map[string]any{
		"path": fmt.Sprintf("modules/%03d", index), "stage": json.Number("0"), "mode": json.Number("57344"),
		"oid": "sha1:" + strings.Repeat("3", 40), "intent_to_add": false,
		"skip_worktree": false, "assume_unchanged": false, "fsmonitor_valid": false,
	}
}

func uninitializedSubmodule(index int) map[string]any {
	return map[string]any{
		"path": fmt.Sprintf("modules/%03d", index), "repository_identity": fmt.Sprintf("relux/module-%03d", index),
		"sanitized_url": fmt.Sprintf("https://example.com/module-%03d.git", index),
		"gitlink_oid":   "sha1:" + strings.Repeat("3", 40), "initialized": false,
		"head": nil, "upstream_ref": nil, "object_pack": nil, "index": nil,
		"working_tree_manifest_id": nil, "submodules": nil, "features": nil,
		"repo_relative_cwd": nil, "agent_project_config_paths": nil,
	}
}

// manifestDirectoryEntry, manifestFileEntry, manifestSymlinkEntry and
// manifestHardlinkEntry build one complete ManifestEntry of each tag so an
// overlap fixture differs from a valid manifest only in the paths and tags it
// declares.
func manifestDirectoryEntry(path string) map[string]any {
	return map[string]any{"path": path, "type": "directory", "mode": json.Number("493")}
}

func manifestFileEntry(path string) map[string]any {
	return map[string]any{
		"path":               path,
		"type":               "file",
		"mode":               json.Number("420"),
		"size":               json.Number("11"),
		"blob_id":            digestWithDigit('4'),
		"blob_descriptor_id": digestWithDigit('5'),
	}
}

func manifestSymlinkEntry(path, target string) map[string]any {
	return map[string]any{"path": path, "type": "symlink", "mode": json.Number("511"), "target": target}
}

func manifestHardlinkEntry(path, targetPath string) map[string]any {
	return map[string]any{"path": path, "type": "hardlink", "mode": json.Number("420"), "target_path": targetPath}
}

func transferManifestWithEntryShapes(entries ...map[string]any) map[string]any {
	object := validTransferManifestObject("workspace_tree")
	values := make([]any, len(entries))
	for index, entry := range entries {
		values[index] = entry
	}
	object["entries"] = values
	return object
}
