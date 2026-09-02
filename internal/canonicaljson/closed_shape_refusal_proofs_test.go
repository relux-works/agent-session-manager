package canonicaljson

import (
	"encoding/json"
	"strings"
	"testing"
)

// This file carries the named negative cases for the Section 10.2-10.4 refusals
// that the derived member sweeps cannot reach: git object-format agreement,
// path grammar inside arrays, ordering inside a nested shape, and the migration
// provenance gate that only an explicitly unsupported schema reaches.
//
// Every case supplies a COMPLETE valid member set and violates exactly one
// clause, so the refusal asserted is the one under test and not the closed-member
// sweep firing first.

// TestGitWorkspaceRefusesAnObjectIDThatDisagreesWithTheDeclaredObjectFormat
// pins the object-format agreement at every place a git OID is read. The
// declared format is switched to sha256 while each OID in turn stays a valid
// sha1, so the refusal cannot come from OID grammar: only the format
// cross-check can produce it.
func TestGitWorkspaceRefusesAnObjectIDThatDisagreesWithTheDeclaredObjectFormat(t *testing.T) {
	const sha256OID = "sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"

	tests := []struct {
		name   string
		build  func(t *testing.T) map[string]any
		mutate func(member map[string]any)
		want   string
	}{
		{"repository head oid", func(t *testing.T) map[string]any { return validGitWorkspaceGroupObject() }, func(member map[string]any) {
			member["features"].(map[string]any)["object_format"] = "sha256"
			member["object_pack"].(map[string]any)["object_format"] = "sha256"
		}, "GitHead oid"},
		{"repository index entry oid", func(t *testing.T) map[string]any { return validGitWorkspaceGroupObject() }, func(member map[string]any) {
			member["features"].(map[string]any)["object_format"] = "sha256"
			member["object_pack"].(map[string]any)["object_format"] = "sha256"
			member["head"].(map[string]any)["oid"] = sha256OID
		}, "GitIndexEntry[0] oid"},
		{"submodule gitlink oid", gitWorkspaceWithInitializedSubmoduleObject, func(member map[string]any) {
			submodule := member["submodules"].([]any)[0].(map[string]any)
			submodule["features"].(map[string]any)["object_format"] = "sha256"
			submodule["object_pack"].(map[string]any)["object_format"] = "sha256"
			submodule["head"].(map[string]any)["oid"] = sha256OID
			for _, entry := range submodule["index"].(map[string]any)["entries"].([]any) {
				entry.(map[string]any)["oid"] = sha256OID
			}
		}, "GitSubmodule gitlink_oid"},
		{"submodule head oid", gitWorkspaceWithInitializedSubmoduleObject, func(member map[string]any) {
			member["submodules"].([]any)[0].(map[string]any)["head"].(map[string]any)["oid"] = sha256OID
		}, "GitHead oid"},
		{"submodule index entry oid", gitWorkspaceWithNestedSubmoduleObject, func(member map[string]any) {
			submodule := member["submodules"].([]any)[0].(map[string]any)
			submodule["index"].(map[string]any)["entries"].([]any)[0].(map[string]any)["oid"] = sha256OID
		}, "GitIndexEntry[0] oid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := test.build(t)
			test.mutate(gitWorkspaceMember(object))
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfManifestID, test.want)
		})
	}
}

func gitWorkspaceWithInitializedSubmoduleObject(t *testing.T) map[string]any {
	t.Helper()
	return gitWorkspaceWithInitializedSubmodule()
}

// gitWorkspaceWithNestedSubmoduleObject carries a submodule whose own index has
// an entry, which the depth-one fixture does not.
func gitWorkspaceWithNestedSubmoduleObject(t *testing.T) map[string]any {
	t.Helper()
	object := validGitWorkspaceGroupObject()
	gitWorkspaceMember(object)["submodules"] = []any{validInitializedSubmoduleShape(1, 2)}
	return object
}

// TestGitFeaturesRequiredFilterNamesRefuseUnsortedAndDuplicateNames pins the
// ordering clause on the one git-features collection that carries it, in both
// failing directions.
func TestGitFeaturesRequiredFilterNamesRefuseUnsortedAndDuplicateNames(t *testing.T) {
	tests := []struct {
		name  string
		names []any
	}{
		{"descending", []any{"lfs", "crlf"}},
		{"duplicate", []any{"crlf", "crlf"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validGitWorkspaceGroupObject()
			gitWorkspaceMember(object)["features"].(map[string]any)["required_filter_names"] = test.names
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfManifestID, "required_filter_names")
		})
	}
}

// TestAgentProjectConfigPathsRefuseAPathOutsideTheWorkspace pins the relative
// path grammar inside the array. The array element is the only place this
// grammar is read, so a wrong-typed element proves nothing about it.
func TestAgentProjectConfigPathsRefuseAPathOutsideTheWorkspace(t *testing.T) {
	for _, path := range []string{"../escape", "/absolute", "./dot-prefixed"} {
		t.Run(path, func(t *testing.T) {
			object := validWorkspaceGroupRecordObject()
			object["members"].([]any)[0].(map[string]any)["agent_project_config_paths"] = []any{path}
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID, "member agent_project_config_paths[0]")
		})
	}
}

// TestManifestEntryPathRefusesAPathOutsideTheTree pins the same grammar at the
// Section 10.4 workspace-tree entry, which reads it through its own call site.
func TestManifestEntryPathRefusesAPathOutsideTheTree(t *testing.T) {
	for _, path := range []string{"../escape", "/absolute"} {
		t.Run(path, func(t *testing.T) {
			object := workspaceTreeWithEveryEntryVariant()
			object["entries"] = []any{map[string]any{"path": path, "type": "directory", "mode": json.Number("493")}}
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfManifestID, "ManifestEntry[0] path")
		})
	}
}

// TestUnsupportedShapeStillRefusesMalformedMigrationProvenance pins the one
// gate an explicitly unsupported schema still runs before its own refusal. The
// unsupported-shape path must not become a way to hand the identity surface an
// extensions container nothing validated.
//
// Every schema registered to a total-refusal validator is driven, and at least
// one must refuse on the migration-provenance ground specifically: the rest
// wrap that validator behind the common record envelope and refuse earlier.
func TestUnsupportedShapeStillRefusesMalformedMigrationProvenance(t *testing.T) {
	totallyRefusing := deriveTotalRefusalValidators(t)
	registered := deriveRegisteredShapeValidators(t)

	driven, provenance := 0, 0
	for key, validatorName := range registered {
		if !totallyRefusing[validatorName] {
			continue
		}
		contract, ok := schemaIdentityContracts[key]
		if !ok {
			continue
		}
		object := map[string]any{
			"schema": key.schema, "schema_version": key.version,
			string(contract.selfField): zeroDigest, "extensions": json.Number("1"),
		}
		if contract.discriminatorName != "" {
			object[contract.discriminatorName] = contract.discriminatorValue
		}
		driven++
		_, _, err := CalculateObjectIdentity(mustJSON(t, object))
		if err == nil {
			t.Fatalf("%s@%s accepted an object whose extensions member is not an object", key.schema, key.version)
		}
		if strings.Contains(err.Error(), "extensions must be a JSON object") {
			provenance++
		}
	}
	if driven == 0 {
		t.Fatal("no schema is registered to a total-refusal validator, so this gate has no subject")
	}
	if provenance == 0 {
		t.Fatal("no unsupported schema refused on the migration-provenance ground; the gate is no longer reached")
	}
}

// TestSessionRecordDerivationProvenanceRefusesAnUndeclaredKind pins the closed
// union tag that selects the provenance arm. It is the refusal that subsumes
// each arm's own kind re-check, so those re-checks are declared unreachable
// against this test.
func TestSessionRecordDerivationProvenanceRefusesAnUndeclaredKind(t *testing.T) {
	for _, version := range []string{"2.0.0", "3.0.0"} {
		t.Run(version, func(t *testing.T) {
			object := validSessionRecordV2Object(validOriginProvenance())
			object["schema_version"] = version
			object["derivation_provenance"].(map[string]any)["kind"] = "resurrection"
			assertIdentityEntriesRefuseWithReason(
				t, mustJSON(t, object), SelfRecordID,
				`derivation_provenance.kind "resurrection" is not a closed `+version+" union member",
			)
		})
	}

	t.Run("native_adoption is unavailable before 3.0.0", func(t *testing.T) {
		object := validSessionRecordV2Object(validNativeAdoptionProvenance())
		assertIdentityEntriesRefuseWithReason(
			t, mustJSON(t, object), SelfRecordID,
			`derivation_provenance.kind "native_adoption" is unavailable in 2.0.0`,
		)
	})
}
