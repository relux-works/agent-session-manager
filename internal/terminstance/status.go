package terminstance

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// StatusBody is the closed §4.C status request body: exactly session_id,
// terminal_instance_id (UUIDv7 or null), terminal_backend_id,
// implementation_version, protocol_version, backend_generation
// (string[1..256] or null), include_provider_observation and deadline_at.
// The instance and generation are either both null (Session-scoped
// lookup) or both non-null (exact-instance lookup). Status carries no
// authorization: its transition row authorizes none.
type StatusBody struct {
	SessionID                  string
	TerminalInstanceID         string
	HasTerminalInstanceID      bool
	TerminalBackendID          string
	ImplementationVersion      string
	ProtocolVersion            string
	BackendGeneration          string
	HasBackendGeneration       bool
	IncludeProviderObservation bool
	Deadline                   scalar.Timestamp
}

// statusBodyMembers is the exact closed member set, in the order the
// pinned section lists it.
var statusBodyMembers = []string{
	"session_id",
	"terminal_instance_id",
	"terminal_backend_id",
	"implementation_version",
	"protocol_version",
	"backend_generation",
	"include_provider_observation",
	"deadline_at",
}

// ParseStatusBody admits one closed status body. Null is the explicit
// JSON null only: missing, empty-string and wrong-typed members are
// malformed, never null.
func ParseStatusBody(raw []byte) (StatusBody, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return StatusBody{}, refuse(CodeProtocolError, "status body frame")
	}
	if len(members) != len(statusBodyMembers) {
		return StatusBody{}, refuse(CodeProtocolError, "status body members")
	}
	for _, member := range statusBodyMembers {
		if _, known := members[member]; !known {
			return StatusBody{}, refuse(CodeProtocolError, "status body members")
		}
	}
	session, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return StatusBody{}, refuse(CodeProtocolError, "status body session")
	}
	instance, hasInstance, err := uuidOrNull(members["terminal_instance_id"])
	if err != nil {
		return StatusBody{}, refuse(CodeProtocolError, "status body instance")
	}
	backendID, ok := rawString(members["terminal_backend_id"])
	if !ok {
		return StatusBody{}, refuse(CodeProtocolError, "status body backend")
	}
	if _, err := terminalbackend.ParseID(backendID); err != nil {
		return StatusBody{}, wrapLanded(err)
	}
	implementationVersion, ok := rawString(members["implementation_version"])
	if !ok {
		return StatusBody{}, refuse(CodeProtocolError, "status body versions")
	}
	protocolVersion, ok := rawString(members["protocol_version"])
	if !ok {
		return StatusBody{}, refuse(CodeProtocolError, "status body versions")
	}
	if err := checkVersionTuple(backendID, implementationVersion, protocolVersion); err != nil {
		return StatusBody{}, err
	}
	generation, hasGeneration, err := generationOrNull(members["backend_generation"])
	if err != nil {
		return StatusBody{}, refuse(CodeStaleGeneration, "backend_generation bound")
	}
	if hasInstance != hasGeneration {
		return StatusBody{}, refuse(CodeProtocolError, "status identity scope")
	}
	var include bool
	if err := json.Unmarshal(members["include_provider_observation"], &include); err != nil {
		return StatusBody{}, refuse(CodeProtocolError, "status body observation")
	}
	deadline, ok := environ.CheckTimestamp(members["deadline_at"])
	if !ok {
		return StatusBody{}, refuse(CodeProtocolError, "status body deadline")
	}
	return StatusBody{
		SessionID:                  session.String(),
		TerminalInstanceID:         instance,
		HasTerminalInstanceID:      hasInstance,
		TerminalBackendID:          backendID,
		ImplementationVersion:      implementationVersion,
		ProtocolVersion:            protocolVersion,
		BackendGeneration:          generation,
		HasBackendGeneration:       hasGeneration,
		IncludeProviderObservation: include,
		Deadline:                   deadline,
	}, nil
}

// uuidOrNull reads a UUIDv7-or-null member: explicit JSON null reads
// absent, a UUIDv7 string reads present, anything else is malformed.
func uuidOrNull(raw json.RawMessage) (string, bool, error) {
	if isExplicitNull(raw) {
		return "", false, nil
	}
	value, ok := rawString(raw)
	if !ok {
		return "", false, refuse(CodeProtocolError, "status body instance")
	}
	parsed, err := scalar.ParseUUIDv7(value)
	if err != nil {
		return "", false, refuse(CodeProtocolError, "status body instance")
	}
	return parsed.String(), true, nil
}

// generationOrNull reads a string[1..256]-or-null member in characters
// through the landed bound. A present but out-of-bound generation is the
// landed terminal_backend_stale_generation class, matching the descriptor
// and digest arms.
func generationOrNull(raw json.RawMessage) (string, bool, error) {
	if isExplicitNull(raw) {
		return "", false, nil
	}
	value, ok := environ.CheckStringBounds(raw, 1, 256)
	if !ok {
		return "", false, refuse(CodeStaleGeneration, "backend_generation bound")
	}
	return value, true, nil
}

// isExplicitNull reports whether the member is the explicit JSON null.
// Only the null literal reads null: missing members never reach here
// (the exact-member check owns absence) and every other literal is the
// typed member the caller parses next.
func isExplicitNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}
