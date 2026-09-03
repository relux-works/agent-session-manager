package cliresult

import (
	"encoding/json"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// The identifiers below are the ones the Section 14.2 normative CLI success
// example uses, so a fixture built here is the same object the pinned document
// displays wherever the example covers it.
const (
	fixtureOperationID   = "0198f4c8-17e0-78ff-8879-1234567890ab"
	fixtureSessionID     = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	fixtureDestinationID = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	fixtureSourceHostID  = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	fixtureOtherHostID   = "0198f4c8-9999-7000-8111-1234567890ab"
	fixtureLeaseID       = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	fixtureCheckpointID  = "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656"
	fixtureOtherDigest   = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	fixtureTimestamp     = "2026-08-19T04:30:00.000Z"
	fixtureAbsolutePath  = "/Users/operator/work/payments-api"
)

func mustUUIDv7(t *testing.T, value string) scalar.UUIDv7 {
	t.Helper()
	parsed, err := scalar.ParseUUIDv7(value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

// sessionSummary builds a valid Section 14.2 SessionSummary.
func sessionSummary(sessionID string) map[string]any {
	return map[string]any{
		"session_id":                   sessionID,
		"name":                         "payments-api",
		"kind":                         "direct",
		"provider_id":                  "codex",
		"owner_host_id":                fixtureSourceHostID,
		"owner_host_name":              "workstation",
		"lease_epoch":                  json.Number("5"),
		"lease_id":                     fixtureLeaseID,
		"local_role":                   "owner",
		"state":                        "running",
		"newest_checkpoint_id":         fixtureCheckpointID,
		"newest_checkpoint_created_at": fixtureTimestamp,
		"workspace_status":             "current",
		"capabilities": map[string]any{
			"native_resume": map[string]any{
				"status": "available", "enabled": true, "detail": "",
			},
		},
		"warnings": []any{},
	}
}

func peerSummary(hostID string) map[string]any {
	return map[string]any{
		"host_id":                 hostID,
		"name":                    "laptop",
		"platform":                "macos",
		"reachable":               true,
		"last_successful_sync_at": fixtureTimestamp,
		"degraded_codes":          []any{},
	}
}

// takeoverBody is the body of the Section 14.2 normative CLI success example.
func takeoverBody() map[string]any {
	return map[string]any{
		"mode":                 "force",
		"workspace_mode":       "whole_group",
		"destination_host_id":  fixtureDestinationID,
		"source_host_id":       fixtureSourceHostID,
		"affected_session_ids": []any{fixtureSessionID},
		"lease_epoch":          json.Number("5"),
		"lease_id":             fixtureLeaseID,
		"checkpoint_id":        fixtureCheckpointID,
		"state":                "running",
		"materialized":         true,
		"adopted":              false,
		"resumed":              true,
		"warnings":             []any{"previous_owner_may_still_be_running"},
	}
}

func materializationSummary() map[string]any {
	return map[string]any{
		"session_id":                 fixtureSessionID,
		"checkpoint_id":              fixtureCheckpointID,
		"materialization_id":         fixtureDestinationID,
		"mode":                       "copy",
		"destination_path":           fixtureAbsolutePath,
		"destination_classification": "empty",
		"preserved_checkpoint_id":    nil,
		"committed":                  true,
		"ownership_changed":          false,
	}
}

func stopBody() map[string]any {
	summary := sessionSummary(fixtureSessionID)
	summary["state"] = "stopped"
	return map[string]any{
		"session":           summary,
		"graceful":          true,
		"checkpoint_id":     fixtureCheckpointID,
		"resumable":         true,
		"bootstrap_aborted": false,
		"process_closed":    true,
		"store_closed":      true,
	}
}

func observationEvent(hostID string) map[string]any {
	return map[string]any{
		"schema":         "urn:ax:schema:observation",
		"schema_version": "1.0.0",
		"stream_id":      fixtureDestinationID,
		"sequence":       json.Number("184"),
		"timestamp":      fixtureTimestamp,
		"level":          "info",
		"event":          "takeover.phase",
		"operation_id":   fixtureOperationID,
		"session_id":     fixtureSessionID,
		"host_id":        hostID,
		"peer_host_id":   nil,
		"phase":          "destination_validated",
		"result":         "success",
		"duration_ms":    json.Number("1240"),
		"counts": map[string]any{
			"records": json.Number("12"), "events": json.Number("0"),
			"manifests": json.Number("0"), "blobs": json.Number("4"),
			"chunks": json.Number("0"), "bytes": json.Number("8192"),
			"retries": json.Number("0"),
		},
		"object_ids": []any{},
		"error_code": nil,
		"extensions": map[string]any{},
	}
}

// validSpec returns a Spec whose body satisfies the command's closed shape.
// Each entry is written out rather than derived, so a change to one command's
// shape cannot be absorbed by a helper that quietly rebuilds every fixture.
func validSpec(t *testing.T, command Command) Spec {
	t.Helper()
	operation := NoIDs().WithOperation(mustUUIDv7(t, fixtureOperationID))
	session := NoIDs().WithSession(mustUUIDv7(t, fixtureSessionID))
	both := operation.WithSession(mustUUIDv7(t, fixtureSessionID))
	switch command {
	case CommandCancel:
		return Spec{Command: command, IDs: NoIDs(), Body: map[string]any{
			"name": "payments-api", "cancelled": true,
		}}
	case CommandStart:
		return Spec{Command: command, IDs: both, Body: map[string]any{
			"session":           sessionSummary(fixtureSessionID),
			"execution_profile": "yolo",
			"terminal_backend":  "tmux",
		}}
	case CommandList:
		return Spec{Command: command, IDs: NoIDs(), Body: map[string]any{
			"sessions":             []any{sessionSummary(fixtureSessionID)},
			"partial":              false,
			"unreachable_peer_ids": []any{},
		}}
	case CommandStatus:
		return Spec{Command: command, IDs: session, Body: map[string]any{
			"session":              sessionSummary(fixtureSessionID),
			"process_present":      true,
			"active_operation_id":  nil,
			"last_successful_sync": map[string]any{fixtureSourceHostID: fixtureTimestamp},
		}}
	case CommandAttach:
		return Spec{Command: command, IDs: session, Body: map[string]any{
			"session":                sessionSummary(fixtureSessionID),
			"mode":                   "local",
			"attached_owner_host_id": fixtureSourceHostID,
			"detached":               false,
			"provider_exit_code":     nil,
		}}
	case CommandTakeover:
		return Spec{
			Command: command, IDs: both, Body: takeoverBody(), SessionKind: KindDirect,
		}
	case CommandFork:
		return Spec{Command: command, IDs: both, Body: map[string]any{
			"source_session_id":    fixtureSourceHostID,
			"source_checkpoint_id": fixtureCheckpointID,
			"session":              sessionSummary(fixtureSessionID),
			"workspace_group_id":   fixtureDestinationID,
			"provider_fork_mode":   "native",
		}}
	case CommandStop:
		return Spec{Command: command, IDs: both, Body: stopBody()}
	case CommandResume:
		return Spec{Command: command, IDs: both, Body: map[string]any{
			"session":           sessionSummary(fixtureSessionID),
			"checkpoint_id":     fixtureCheckpointID,
			"terminal_backend":  "tmux",
			"native_session_id": "codex-0001",
		}}
	case CommandSync:
		return Spec{Command: command, IDs: operation, Body: map[string]any{
			"peer_ids":       []any{fixtureSourceHostID},
			"record_count":   json.Number("12"),
			"blob_count":     json.Number("4"),
			"byte_count":     json.Number("8192"),
			"checkpoint_ids": []any{fixtureCheckpointID},
			"materialized":   true,
			"partial":        false,
			"transfer_id":    nil,
		}}
	case CommandDiff:
		return Spec{Command: command, IDs: session, Body: map[string]any{
			"session_id":     fixtureSessionID,
			"peer_host_id":   nil,
			"classification": "different",
			"entries": []any{map[string]any{
				"path": "src/main.go", "classification": "modified",
				"source_digest": fixtureCheckpointID, "destination_digest": fixtureOtherDigest,
			}},
		}}
	case CommandMaterialize:
		return Spec{Command: command, IDs: both, Body: materializationSummary()}
	case CommandDoctor:
		return Spec{Command: command, IDs: NoIDs(), Body: map[string]any{
			"healthy": true,
			"findings": []any{map[string]any{
				"severity": "info", "code": "tmux_present", "message": "tmux 3.5a",
				"remediation": nil, "source": "terminal",
			}},
		}}
	case CommandLogs:
		return Spec{Command: command, IDs: NoIDs(), Body: map[string]any{
			"emitting_host_id": fixtureSourceHostID,
			"events":           []any{observationEvent(fixtureSourceHostID)},
			"next_cursor":      nil,
		}}
	case CommandPeerList:
		return Spec{Command: command, IDs: NoIDs(), Body: map[string]any{
			"peers": []any{peerSummary(fixtureSourceHostID)},
		}}
	case CommandPeerProbe:
		return Spec{Command: command, IDs: NoIDs(), Body: map[string]any{
			"peer":          peerSummary(fixtureSourceHostID),
			"contracts":     map[string]any{"rpc": []any{"2.0.0"}},
			"round_trip_ms": json.Number("12"),
		}}
	case CommandSessionSetProfile:
		return Spec{Command: command, IDs: both, Body: map[string]any{
			"session_id":       fixtureSessionID,
			"previous_profile": "standard",
			"new_profile":      "yolo",
			"event_id":         fixtureCheckpointID,
		}}
	case CommandPane:
		return Spec{Command: command, IDs: session, Body: map[string]any{
			"session_id":            fixtureSessionID,
			"result":                "attached",
			"winning_owner_host_id": fixtureSourceHostID,
			"lease_epoch":           json.Number("5"),
		}}
	default:
		t.Fatalf("no fixture for command %q", command)
		return Spec{}
	}
}

// mustResult builds a result the test expects to be valid.
func mustResult(t *testing.T, spec Spec) *Result {
	t.Helper()
	result, err := New(spec)
	if err != nil {
		t.Fatalf("New(%q): %v", spec.Command, err)
	}
	return result
}

// mutateBody returns a copy of a valid body with one member replaced. It is how
// a negative test narrows a gate: everything except the member under test stays
// exactly the value the positive test admitted.
func mutateBody(body map[string]any, key string, value any) map[string]any {
	clone := cloneObject(body)
	if value == nil {
		clone[key] = nil
		return clone
	}
	clone[key] = value
	return clone
}
