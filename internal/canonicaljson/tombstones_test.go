package canonicaljson

import (
	"encoding/json"
	"errors"
	"testing"
)

const (
	tombstoneSessionID = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	tombstoneGroupID   = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	tombstoneWorkspace = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	tombstoneHostID    = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	tombstoneTargetID  = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
)

func validTombstoneObject(scope string) map[string]any {
	targets := map[string]map[string]any{
		"session": {
			"kind": "session", "session_id": tombstoneSessionID,
			"session_record_id": digestWithDigit('1'), "predecessor_checkpoint_id": digestWithDigit('2'),
		},
		"workspace_entry": {
			"kind": "workspace_entry", "workspace_group_id": tombstoneGroupID,
			"workspace_id": tombstoneWorkspace, "predecessor_manifest_id": digestWithDigit('3'),
			"relative_path": "src/main.go", "entry_type": "file", "predecessor_entry_digest": digestWithDigit('4'),
		},
		"provider_snapshot": {
			"kind": "provider_snapshot", "session_id": tombstoneSessionID,
			"provider_identity_record_id": digestWithDigit('5'), "predecessor_manifest_id": digestWithDigit('6'),
		},
		"managed_replica": {
			"kind": "managed_replica", "workspace_group_id": tombstoneGroupID,
			"target_host_id": tombstoneTargetID, "managed_replica_id": "0198f4c8-8e50-7f66-8f70-2234567890ab",
			"logical_root": "relux", "destination_relative_path": "replicas/payments",
			"predecessor_marker_id": digestWithDigit('7'), "predecessor_checkpoint_id": digestWithDigit('8'),
		},
	}
	subjectID := tombstoneSessionID
	if scope == "workspace_entry" || scope == "managed_replica" {
		subjectID = tombstoneGroupID
	}
	return map[string]any{
		"schema": tombstoneSchema, "schema_version": "1.0.0", "tombstone_id": zeroDigest,
		"scope": scope, "subject_id": subjectID, "authorizing_session_id": tombstoneSessionID,
		"basis_event_id": digestWithDigit('9'), "lease_epoch": json.Number("4"),
		"lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "target": targets[scope],
		"created_by_host_id": tombstoneHostID, "created_at": "2026-08-19T04:20:00.000Z",
		"extensions": map[string]any{},
	}
}

func validTombstoneAckObject(disposition string, conflictCheckpointID *string) map[string]any {
	var conflict any
	if conflictCheckpointID != nil {
		conflict = *conflictCheckpointID
	}
	return map[string]any{
		"schema": tombstoneAckSchema, "schema_version": "1.0.0", "ack_id": zeroDigest,
		"subject_id": tombstoneGroupID, "tombstone_id": digestWithDigit('a'),
		"acknowledging_host_id": tombstoneTargetID, "disposition": disposition,
		"conflict_checkpoint_id": conflict, "observed_at": "2026-08-19T04:21:00.000Z",
		"created_by_host_id": tombstoneTargetID, "created_at": "2026-08-19T04:21:00.000Z",
		"extensions": map[string]any{},
	}
}

func stringPointer(value string) *string { return &value }

func TestTombstoneScopeTargetsValidateAtIdentityEntries(t *testing.T) {
	for _, scope := range []string{"session", "workspace_entry", "provider_snapshot", "managed_replica"} {
		t.Run(scope, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validTombstoneObject(scope)), SelfTombstoneID)
		})
	}
}

func TestTombstoneSchemaAndScopeSelectionRefusesAtIdentityEntries(t *testing.T) {
	t.Run("unknown Tombstone schema", func(t *testing.T) {
		object := validTombstoneObject("session")
		object["schema"] = "urn:ax:schema:unregistered-tombstone"
		assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfTombstoneID)
	})
	t.Run("unknown Tombstone version", func(t *testing.T) {
		object := validTombstoneObject("session")
		object["schema_version"] = "2.0.0"
		assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfTombstoneID)
	})
	t.Run("malformed Tombstone self digest", func(t *testing.T) {
		object := validTombstoneObject("session")
		object["tombstone_id"] = "not-a-digest"
		assertIdentityEntriesRefuseMalformedSelfDigest(t, mustJSON(t, object), SelfTombstoneID)
	})
	t.Run("unknown acknowledgement schema", func(t *testing.T) {
		object := validTombstoneAckObject("applied", nil)
		object["schema"] = "urn:ax:schema:unregistered-tombstone-ack"
		assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfAckID)
	})
	t.Run("unknown acknowledgement version", func(t *testing.T) {
		object := validTombstoneAckObject("applied", nil)
		object["schema_version"] = "2.0.0"
		assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfAckID)
	})
	t.Run("malformed acknowledgement self digest", func(t *testing.T) {
		object := validTombstoneAckObject("applied", nil)
		object["ack_id"] = "not-a-digest"
		assertIdentityEntriesRefuseMalformedSelfDigest(t, mustJSON(t, object), SelfAckID)
	})
	t.Run("unknown Tombstone scope", func(t *testing.T) {
		object := validTombstoneObject("session")
		object["scope"] = "replica_everywhere"
		assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfTombstoneID)
	})
}

func assertIdentityEntriesRefuseMalformedSelfDigest(t *testing.T, input []byte, selfField SelfField) {
	t.Helper()
	if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("CalculateObjectIdentity(%s malformed self digest) error = %v, want resolver refusal", selfField, err)
	}
	if _, _, err := VerifyObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("VerifyObjectIdentity(%s malformed self digest) error = %v, want resolver refusal", selfField, err)
	}
}

func TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes(t *testing.T) {
	tests := []struct {
		name   string
		object map[string]any
		mutate func(map[string]any)
	}{
		{"target kind mismatch", validTombstoneObject("session"), func(object map[string]any) { object["target"].(map[string]any)["kind"] = "provider_snapshot" }},
		{"target scope field leakage", validTombstoneObject("session"), func(object map[string]any) { object["target"].(map[string]any)["workspace_id"] = tombstoneWorkspace }},
		{"provider target scope field leakage", validTombstoneObject("provider_snapshot"), func(object map[string]any) { object["target"].(map[string]any)["workspace_id"] = tombstoneWorkspace }},
		{"subject has invalid UUID", validTombstoneObject("session"), func(object map[string]any) { object["subject_id"] = "not-a-uuid" }},
		{"subject differs from target", validTombstoneObject("provider_snapshot"), func(object map[string]any) { object["subject_id"] = tombstoneGroupID }},
		{"workspace path traversal", validTombstoneObject("workspace_entry"), func(object map[string]any) { object["target"].(map[string]any)["relative_path"] = "../secret" }},
		{"workspace path wildcard", validTombstoneObject("workspace_entry"), func(object map[string]any) { object["target"].(map[string]any)["relative_path"] = "src/*.go" }},
		{"workspace path question wildcard", validTombstoneObject("workspace_entry"), func(object map[string]any) { object["target"].(map[string]any)["relative_path"] = "src/name?.go" }},
		{"workspace root path", validTombstoneObject("workspace_entry"), func(object map[string]any) { object["target"].(map[string]any)["relative_path"] = "." }},
		{"invalid workspace entry type", validTombstoneObject("workspace_entry"), func(object map[string]any) { object["target"].(map[string]any)["entry_type"] = "wildcard" }},
		{"managed replica root path", validTombstoneObject("managed_replica"), func(object map[string]any) { object["target"].(map[string]any)["destination_relative_path"] = "." }},
		{"managed replica path wildcard", validTombstoneObject("managed_replica"), func(object map[string]any) {
			object["target"].(map[string]any)["destination_relative_path"] = "replicas/*"
		}},
		{"empty logical root", validTombstoneObject("managed_replica"), func(object map[string]any) { object["target"].(map[string]any)["logical_root"] = "" }},
		{"unsafe lease epoch", validTombstoneObject("session"), func(object map[string]any) { object["lease_epoch"] = json.Number("9007199254740992") }},
		{"lease token is not UUIDv4", validTombstoneObject("session"), func(object map[string]any) { object["lease_id"] = "0198f4c8-3e70-7a11-8a2b-1234567890ab" }},
		{"common envelope unknown member", validTombstoneObject("session"), func(object map[string]any) { object["untrusted"] = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := cloneJSONObject(t, test.object)
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfTombstoneID)
		})
	}
}

func TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings(t *testing.T) {
	tests := []struct {
		name   string
		object map[string]any
		mutate func(map[string]any)
	}{
		{"conflict requires checkpoint", validTombstoneAckObject("retained_conflict", nil), func(map[string]any) {}},
		{"non-conflict forbids checkpoint", validTombstoneAckObject("applied", stringPointer(digestWithDigit('b'))), func(map[string]any) {}},
		{"created by must be acknowledging host", validTombstoneAckObject("applied", nil), func(object map[string]any) { object["created_by_host_id"] = tombstoneHostID }},
		{"disposition is closed", validTombstoneAckObject("applied", nil), func(object map[string]any) { object["disposition"] = "executed_delete" }},
		{"observed time is timestamp", validTombstoneAckObject("applied", nil), func(object map[string]any) { object["observed_at"] = "yesterday" }},
		{"acknowledgement is closed", validTombstoneAckObject("applied", nil), func(object map[string]any) { object["executed"] = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := cloneJSONObject(t, test.object)
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfAckID)
		})
	}
	for _, disposition := range []string{"applied", "already_absent", "retained_conflict", "not_target"} {
		t.Run("positive "+disposition, func(t *testing.T) {
			var conflict *string
			if disposition == "retained_conflict" {
				conflict = stringPointer(digestWithDigit('c'))
			}
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validTombstoneAckObject(disposition, conflict)), SelfAckID)
		})
	}
}
