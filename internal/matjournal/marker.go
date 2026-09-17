package matjournal

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Marker is the typed Managed Replica Marker document: the immutable
// second variant of the Section 10.6 recovery contract. It is the
// authority for classifying a destination as managed.
type Marker struct {
	Schema                 string             `json:"schema"`
	SchemaVersion          string             `json:"schema_version"`
	DocumentKind           string             `json:"document_kind"`
	MarkerID               string             `json:"marker_id"`
	ManagedReplicaID       string             `json:"managed_replica_id"`
	HostID                 string             `json:"host_id"`
	Platform               string             `json:"platform"`
	WorkspaceGroupID       string             `json:"workspace_group_id"`
	PrimarySessionID       string             `json:"primary_session_id"`
	SourceCheckpointID     string             `json:"source_checkpoint_id"`
	WorkspaceGroupRecordID string             `json:"workspace_group_record_id"`
	WorkspaceManifestID    string             `json:"workspace_manifest_id"`
	PlanID                 string             `json:"plan_id"`
	MaterializationID      string             `json:"materialization_id"`
	Destination            ReplicaDestination `json:"destination"`
	PredecessorMarkerID    *string            `json:"predecessor_marker_id"`
	CommittedAt            string             `json:"committed_at"`
	Extensions             map[string]any     `json:"extensions"`
}

// ReplicaDestination is the closed destination shape.
type ReplicaDestination struct {
	LogicalRoot             string `json:"logical_root"`
	WorkspaceRelativePath   string `json:"workspace_relative_path"`
	ResolvedPathFingerprint string `json:"resolved_path_fingerprint"`
}

// markerMembers is the exact closed 18-member marker shape.
var markerMembers = []string{
	"schema", "schema_version", "document_kind", "marker_id",
	"managed_replica_id", "host_id", "platform", "workspace_group_id",
	"primary_session_id", "source_checkpoint_id",
	"workspace_group_record_id", "workspace_manifest_id", "plan_id",
	"materialization_id", "destination", "predecessor_marker_id",
	"committed_at", "extensions",
}

// destinationMembers is the exact closed 3-member destination shape.
var destinationMembers = []string{
	"logical_root", "workspace_relative_path", "resolved_path_fingerprint",
}

// ValidateMarker validates the closed marker shape and attests its
// omit-self identity through canonical JSON. The marker_id is the
// SHA-256 of the JCS form with marker_id omitted; a drift between the
// claim and recomputation fails closed. The shape owner still refuses
// this schema, so the shape is validated here (see doc.go).
func ValidateMarker(raw []byte) (Marker, error) {
	var marker Marker
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return Marker{}, invalid("decode marker frame: %v", fault)
	}
	if err := closedMembers(members, markerMembers, "marker"); err != nil {
		return Marker{}, err
	}
	destination, fault := environ.DecodeStrictObject(members["destination"])
	if fault != nil {
		return Marker{}, invalid("decode marker destination frame: %v", fault)
	}
	if err := closedMembers(destination, destinationMembers, "marker destination"); err != nil {
		return Marker{}, err
	}
	decoder := strictDecoder(raw)
	if err := decoder.Decode(&marker); err != nil {
		return Marker{}, invalid("decode marker members: %v", err)
	}
	if err := checkMarkerGrammar(&marker); err != nil {
		return Marker{}, err
	}
	omitted := make(map[string]any, len(members)-1)
	framed, err := json.Marshal(marker)
	if err != nil {
		return Marker{}, invalid("encode marker for identity: %v", err)
	}
	var logical map[string]any
	if err := json.Unmarshal(framed, &logical); err != nil {
		return Marker{}, invalid("decode marker for identity: %v", err)
	}
	for name, value := range logical {
		if name != "marker_id" {
			omitted[name] = value
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		return Marker{}, invalid("encode omit-self marker: %v", err)
	}
	canonical, err := canonicaljson.Canonicalize(serialized)
	if err != nil {
		return Marker{}, invalid("canonicalize omit-self marker: %v", err)
	}
	if computed := scalar.SHA256Digest(canonical).String(); computed != marker.MarkerID {
		return Marker{}, invalid("marker_id %q disagrees with recomputation %q", marker.MarkerID, computed)
	}
	return marker, nil
}

// checkMarkerGrammar validates every marker member grammar rule.
func checkMarkerGrammar(marker *Marker) error {
	if marker.Schema != journalSchema {
		return invalid("marker schema %q, want %q", marker.Schema, journalSchema)
	}
	if marker.SchemaVersion != journalVersion {
		return invalid("marker schema_version %q, want %q", marker.SchemaVersion, journalVersion)
	}
	if marker.DocumentKind != markerKind {
		return invalid("marker document_kind %q, want %q", marker.DocumentKind, markerKind)
	}
	var err error
	if marker.MarkerID, err = checkDigest(marker.MarkerID, "marker_id"); err != nil {
		return err
	}
	if marker.ManagedReplicaID, err = checkUUIDv7(marker.ManagedReplicaID, "managed_replica_id"); err != nil {
		return err
	}
	if marker.HostID, err = checkUUIDv7(marker.HostID, "host_id"); err != nil {
		return err
	}
	if marker.Platform, err = checkPlatform(marker.Platform, "platform"); err != nil {
		return err
	}
	if marker.WorkspaceGroupID, err = checkUUIDv7(marker.WorkspaceGroupID, "workspace_group_id"); err != nil {
		return err
	}
	if marker.PrimarySessionID, err = checkUUIDv7(marker.PrimarySessionID, "primary_session_id"); err != nil {
		return err
	}
	if marker.SourceCheckpointID, err = checkDigest(marker.SourceCheckpointID, "source_checkpoint_id"); err != nil {
		return err
	}
	if marker.WorkspaceGroupRecordID, err = checkDigest(marker.WorkspaceGroupRecordID, "workspace_group_record_id"); err != nil {
		return err
	}
	if marker.WorkspaceManifestID, err = checkDigest(marker.WorkspaceManifestID, "workspace_manifest_id"); err != nil {
		return err
	}
	if marker.PlanID, err = checkDigest(marker.PlanID, "plan_id"); err != nil {
		return err
	}
	if marker.MaterializationID, err = checkUUIDv7(marker.MaterializationID, "materialization_id"); err != nil {
		return err
	}
	if err := checkReplicaDestination(&marker.Destination); err != nil {
		return err
	}
	if marker.PredecessorMarkerID, err = checkNullableDigestPointer(marker.PredecessorMarkerID, "predecessor_marker_id"); err != nil {
		return err
	}
	if marker.CommittedAt, err = checkTimestamp(marker.CommittedAt, "committed_at"); err != nil {
		return err
	}
	return checkExtensionsMap(marker.Extensions)
}

// checkReplicaDestination validates the closed destination shape: the
// logical root, the workspace-relative path, and the resolved-path
// fingerprint.
func checkReplicaDestination(destination *ReplicaDestination) error {
	if length := len([]rune(destination.LogicalRoot)); length < 1 || length > 64 {
		return invalid("destination logical_root must contain 1..64 characters")
	}
	relative, err := scalar.ParseRelativePath(destination.WorkspaceRelativePath)
	if err != nil {
		return invalid("destination workspace_relative_path: %v", err)
	}
	destination.WorkspaceRelativePath = relative.String()
	fingerprint, err := scalar.ParseDigest(destination.ResolvedPathFingerprint)
	if err != nil {
		return invalid("destination resolved_path_fingerprint: %v", err)
	}
	destination.ResolvedPathFingerprint = fingerprint.String()
	return nil
}

// checkMarkerBinding binds the destination marker to the committing
// journal: the managed replica, plan, source checkpoint, and
// materialization must equal the journal, and a non-null predecessor
// must resolve to valid prior marker bytes. A marker copied from
// another host, replica, or materialization is never destination
// authority.
func checkMarkerBinding(marker Marker, journal *Journal, priorMarker []byte) error {
	if journal.ManagedReplicaID == nil || marker.ManagedReplicaID != *journal.ManagedReplicaID {
		return invalid("marker managed_replica_id drifts from the journal")
	}
	if marker.PlanID != journal.PlanID {
		return invalid("marker plan_id drifts from the journal")
	}
	if marker.SourceCheckpointID != journal.SourceCheckpointID {
		return invalid("marker source_checkpoint_id drifts from the journal")
	}
	if marker.MaterializationID != journal.MaterializationID {
		return invalid("marker materialization_id drifts from the journal")
	}
	if marker.PredecessorMarkerID == nil {
		return nil
	}
	if len(priorMarker) == 0 {
		return invalid("marker predecessor %q has no prior marker evidence", *marker.PredecessorMarkerID)
	}
	prior, err := ValidateMarker(priorMarker)
	if err != nil {
		return err
	}
	if prior.MarkerID != *marker.PredecessorMarkerID {
		return invalid("prior marker %q disagrees with predecessor %q", prior.MarkerID, *marker.PredecessorMarkerID)
	}
	if prior.ManagedReplicaID != marker.ManagedReplicaID {
		return invalid("prior marker names another managed replica")
	}
	return nil
}

// Destination classes (SPEC Section 10.6).
const (
	DestinationAbsent            = "absent"
	DestinationEmpty             = "empty"
	DestinationUnmanagedNonempty = "unmanaged_nonempty"
	DestinationManagedUnchanged  = "managed_unchanged"
	DestinationManagedDivergent  = "managed_divergent"
	DestinationIntegrityFailure  = "integrity_failure"
)

// DestinationFacts is the modeled destination evidence ClassifyDestination
// decides from. TargetBytes compares a fresh workspace capture against
// the marker manifest; no filesystem scan happens here.
type DestinationFacts struct {
	// Marker carries the current marker bytes when a marker file exists.
	Marker []byte
	// MarkerPresent reports whether any marker file exists.
	MarkerPresent bool
	// TargetAbsent reports a missing target path.
	TargetAbsent bool
	// TargetEmpty reports an existing but empty target path.
	TargetEmpty bool
	// ManifestMatches reports a fresh capture equal to the marker's
	// workspace manifest.
	ManifestMatches bool
	// FingerprintMatches reports the marker fingerprint resolving to
	// this exact path under the current configuration.
	FingerprintMatches bool
}

// ClassifyDestination deterministically classifies one destination
// from its durable facts: absent, empty, unmanaged_nonempty,
// managed_unchanged, managed_divergent, or integrity_failure. A
// marker that fails validation, a fingerprint that resolves
// elsewhere, or a manifest mismatch against a resolving marker each
// fail closed into their exact class; nothing here writes.
func ClassifyDestination(facts DestinationFacts) (string, error) {
	if !facts.MarkerPresent {
		switch {
		case facts.TargetAbsent:
			return DestinationAbsent, nil
		case facts.TargetEmpty:
			return DestinationEmpty, nil
		default:
			return DestinationUnmanagedNonempty, nil
		}
	}
	marker, err := ValidateMarker(facts.Marker)
	if err != nil {
		return DestinationIntegrityFailure, nil
	}
	_ = marker
	if !facts.FingerprintMatches {
		return DestinationIntegrityFailure, nil
	}
	if facts.ManifestMatches {
		return DestinationManagedUnchanged, nil
	}
	return DestinationManagedDivergent, nil
}

// strictDecoder decodes with unknown members refused at every level.
func strictDecoder(raw []byte) *json.Decoder {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder
}
