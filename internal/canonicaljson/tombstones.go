package canonicaljson

import (
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	tombstoneSchema    = "urn:ax:schema:tombstone"
	tombstoneAckSchema = "urn:ax:schema:tombstone-ack"
)

func validateTombstoneRecord(object map[string]any) error {
	if err := requireExactMembers("Tombstone", object,
		"schema", "schema_version", "tombstone_id", "scope", "subject_id",
		"authorizing_session_id", "basis_event_id", "lease_epoch", "lease_id",
		"target", "created_by_host_id", "created_at", "extensions",
	); err != nil {
		return err
	}
	if err := requireExactString(object, "schema", tombstoneSchema); err != nil {
		return err
	}
	if err := requireExactString(object, "schema_version", closedSchemaVersion); err != nil {
		return err
	}
	if err := requireDigest(object, "tombstone_id"); err != nil {
		return err
	}
	scope, err := requireEnum(object, "scope", "session", "workspace_entry", "provider_snapshot", "managed_replica")
	if err != nil {
		return err
	}
	if err := validateCommonRecordEnvelope(object); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "authorizing_session_id"); err != nil {
		return err
	}
	if err := requireDigest(object, "basis_event_id"); err != nil {
		return err
	}
	if _, err := requireUint(object, "lease_epoch", scalar.MaxUint53); err != nil {
		return err
	}
	if err := requireUUIDv4(object, "lease_id"); err != nil {
		return err
	}
	target, err := requireObject(object, "target")
	if err != nil {
		return err
	}
	return validateTombstoneTarget(scope, object, target)
}

func validateTombstoneTarget(scope string, envelope, target map[string]any) error {
	kind, err := requireString(target, "kind")
	if err != nil {
		return err
	}
	if kind != scope {
		return invalidIdentity("Tombstone target kind %q does not match scope %q", kind, scope)
	}
	var repeatedSubject string
	switch scope {
	case "session":
		if err := requireExactMembers("Tombstone target.session", target, "kind", "session_id", "session_record_id", "predecessor_checkpoint_id"); err != nil {
			return err
		}
		repeatedSubject = "session_id"
	case "workspace_entry":
		if err := requireExactMembers("Tombstone target.workspace_entry", target, "kind", "workspace_group_id", "workspace_id", "predecessor_manifest_id", "relative_path", "entry_type", "predecessor_entry_digest"); err != nil {
			return err
		}
		repeatedSubject = "workspace_group_id"
	case "provider_snapshot":
		if err := requireExactMembers("Tombstone target.provider_snapshot", target, "kind", "session_id", "provider_identity_record_id", "predecessor_manifest_id"); err != nil {
			return err
		}
		repeatedSubject = "session_id"
	case "managed_replica":
		if err := requireExactMembers("Tombstone target.managed_replica", target, "kind", "workspace_group_id", "target_host_id", "managed_replica_id", "logical_root", "destination_relative_path", "predecessor_marker_id", "predecessor_checkpoint_id"); err != nil {
			return err
		}
		repeatedSubject = "workspace_group_id"
	default:
		return invalidIdentity("Tombstone scope %q has no target shape", scope)
	}
	repeatedID, err := requireUUIDv7(target, repeatedSubject)
	if err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(envelope, "subject_id")
	if err != nil {
		return err
	}
	if repeatedID != subjectID {
		return invalidIdentity("Tombstone subject_id does not match target %s", repeatedSubject)
	}
	switch scope {
	case "session":
		if err := requireDigest(target, "session_record_id"); err != nil {
			return err
		}
		return requireDigest(target, "predecessor_checkpoint_id")
	case "workspace_entry":
		if _, err := requireUUIDv7(target, "workspace_id"); err != nil {
			return err
		}
		if err := requireDigest(target, "predecessor_manifest_id"); err != nil {
			return err
		}
		if err := requireTombstoneTargetPath(target, "relative_path"); err != nil {
			return err
		}
		if _, err := requireEnum(target, "entry_type", "directory", "file", "symlink", "hardlink"); err != nil {
			return err
		}
		return requireDigest(target, "predecessor_entry_digest")
	case "provider_snapshot":
		if err := requireDigest(target, "provider_identity_record_id"); err != nil {
			return err
		}
		return requireDigest(target, "predecessor_manifest_id")
	case "managed_replica":
		if _, err := requireUUIDv7(target, "target_host_id"); err != nil {
			return err
		}
		if _, err := requireUUIDv7(target, "managed_replica_id"); err != nil {
			return err
		}
		if _, err := requireBoundedString(target, "logical_root", 1, 64); err != nil {
			return err
		}
		if err := requireTombstoneTargetPath(target, "destination_relative_path"); err != nil {
			return err
		}
		if err := requireDigest(target, "predecessor_marker_id"); err != nil {
			return err
		}
		return requireDigest(target, "predecessor_checkpoint_id")
	default:
		return invalidIdentity("Tombstone scope %q has no target shape", scope)
	}
}

func requireTombstoneTargetPath(object map[string]any, name string) error {
	value, err := requireRelativePathValue(object, name)
	if err != nil {
		return err
	}
	if strings.ContainsAny(value, "*?[]") {
		return invalidIdentity("member %s must name one exact path and cannot contain wildcard metacharacters", name)
	}
	return nil
}

func validateTombstoneAckRecord(object map[string]any) error {
	if err := requireExactMembers("Tombstone Acknowledgement", object,
		"schema", "schema_version", "ack_id", "subject_id", "tombstone_id",
		"acknowledging_host_id", "disposition", "conflict_checkpoint_id",
		"observed_at", "created_by_host_id", "created_at", "extensions",
	); err != nil {
		return err
	}
	if err := requireExactString(object, "schema", tombstoneAckSchema); err != nil {
		return err
	}
	if err := requireExactString(object, "schema_version", closedSchemaVersion); err != nil {
		return err
	}
	if err := requireDigest(object, "ack_id"); err != nil {
		return err
	}
	if err := requireDigest(object, "tombstone_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "acknowledging_host_id"); err != nil {
		return err
	}
	disposition, err := requireEnum(object, "disposition", "applied", "already_absent", "retained_conflict", "not_target")
	if err != nil {
		return err
	}
	conflictPresent, err := requireNullableDigestPresence(object, "conflict_checkpoint_id")
	if err != nil {
		return err
	}
	if conflictPresent != (disposition == "retained_conflict") {
		return invalidIdentity("Tombstone Acknowledgement conflict_checkpoint_id must be non-null exactly for retained_conflict")
	}
	observedAt, err := requireString(object, "observed_at")
	if err != nil {
		return err
	}
	if _, err := scalar.ParseTimestamp(observedAt); err != nil {
		return invalidIdentity("Tombstone Acknowledgement observed_at: %v", err)
	}
	if err := validateCommonRecordEnvelope(object); err != nil {
		return err
	}
	acknowledgingHostID, _ := object["acknowledging_host_id"].(string)
	createdByHostID, _ := object["created_by_host_id"].(string)
	if createdByHostID != acknowledgingHostID {
		return invalidIdentity("Tombstone Acknowledgement created_by_host_id must equal acknowledging_host_id")
	}
	return nil
}
