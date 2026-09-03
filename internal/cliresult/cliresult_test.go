package cliresult

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// normativeCLISuccess is the Section 14.2 "normative CLI success" example,
// copied verbatim from the pinned document.
const normativeCLISuccess = `{
  "schema": "urn:ax:schema:cli-result",
  "schema_version": "1.0.0",
  "command": "takeover",
  "ok": true,
  "operation_id": "0198f4c8-17e0-78ff-8879-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "body": {
    "mode": "force",
    "workspace_mode": "whole_group",
    "destination_host_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab",
    "source_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
    "affected_session_ids": [
      "0198f4c8-3e70-7a11-8a2b-1234567890ab"
    ],
    "lease_epoch": 5,
    "lease_id": "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff",
    "checkpoint_id": "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656",
    "state": "running",
    "materialized": true,
    "adopted": false,
    "resumed": true,
    "warnings": ["previous_owner_may_still_be_running"]
  },
  "extensions": {}
}`

// TestNormativeExampleDecodesAndReEncodesToTheSameCanonicalBytes drives the
// pinned example through the production reader and the production encoder. A
// reader that admitted the example while an encoder produced different bytes
// would mean the two halves of this package disagree about the same object.
func TestNormativeExampleDecodesAndReEncodesToTheSameCanonicalBytes(t *testing.T) {
	result, err := Decode(Version100, []byte(normativeCLISuccess))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if result.Command() != CommandTakeover {
		t.Fatalf("command = %q, want takeover", result.Command())
	}
	if result.Version() != Version100 {
		t.Fatalf("version = %q, want 1.0.0", result.Version())
	}
	operation, known := result.IDs().Operation()
	if !known || operation.String() != fixtureOperationID {
		t.Fatalf("operation_id = %q/%t", operation.String(), known)
	}
	session, known := result.IDs().Session()
	if !known || session.String() != fixtureSessionID {
		t.Fatalf("session_id = %q/%t", session.String(), known)
	}
	// The example is a direct takeover: adopted is false and resumed is true.
	if err := result.VerifyTakeoverAdoption(KindDirect); err != nil {
		t.Fatalf("VerifyTakeoverAdoption(direct): %v", err)
	}
	encoded, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	expected, err := canonicaljson.Canonicalize([]byte(normativeCLISuccess))
	if err != nil {
		t.Fatalf("canonicalize example: %v", err)
	}
	actual, err := canonicaljson.Canonicalize(encoded)
	if err != nil {
		t.Fatalf("canonicalize encoding: %v", err)
	}
	if !bytes.Equal(expected, actual) {
		t.Fatalf("re-encoded example differs:\n want %s\n  got %s", expected, actual)
	}
}

// TestEveryImplementedCommandRoundTripsThroughItsOwnReader drives New, the
// encoder, and Decode for every command whose body this repository builds. The
// assertion is that the writer cannot emit an object its own reader refuses,
// which is the defect this package's shared validator exists to remove.
func TestEveryImplementedCommandRoundTripsThroughItsOwnReader(t *testing.T) {
	implemented := ImplementedCommands()
	if len(implemented) != 18 {
		t.Fatalf("implemented commands = %d, want the 18 Section 14.2 tags: %v", len(implemented), implemented)
	}
	for _, command := range implemented {
		t.Run(string(command), func(t *testing.T) {
			spec := validSpec(t, command)
			result := mustResult(t, spec)
			encoded, err := result.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			decoded, err := Decode(result.Version(), encoded)
			if err != nil {
				t.Fatalf("Decode: %v\n%s", err, encoded)
			}
			if decoded.Command() != command {
				t.Fatalf("command = %q, want %q", decoded.Command(), command)
			}
			if !reflect.DeepEqual(decoded.Body(), result.Body()) {
				t.Fatalf("body differs after round trip")
			}
			reEncoded, err := decoded.MarshalJSON()
			if err != nil {
				t.Fatalf("re-MarshalJSON: %v", err)
			}
			if !bytes.Equal(canonicalize(t, encoded), canonicalize(t, reEncoded)) {
				t.Fatalf("round trip is not canonical-stable")
			}
		})
	}
}

func canonicalize(t *testing.T, data []byte) []byte {
	t.Helper()
	canonical, err := canonicaljson.Canonicalize(data)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	return canonical
}

// TestVersionSelectionIsStaticPerCommandTag pins the Section 14.2 selection
// rule as a reviewed literal table compared against production. Ranging over
// the production registry instead would let a mutant that deletes a row delete
// its own test case and survive.
func TestVersionSelectionIsStaticPerCommandTag(t *testing.T) {
	expected := map[Command]Version{
		"cancel": Version100, "start": Version100, "list": Version100, "status": Version100,
		"attach": Version100, "takeover": Version100, "fork": Version100, "stop": Version100,
		"resume": Version100, "sync": Version100, "diff": Version100, "materialize": Version100,
		"doctor": Version100, "logs": Version100, "peer.list": Version100, "peer.probe": Version100,
		"session.set_profile": Version100, "pane": Version100,

		"session.clone.adapters": Version200, "session.clone.doctor": Version200,
		"session.clone.list": Version200, "session.clone.inspect": Version200,
		"session.clone.plan": Version200, "session.clone.run": Version200,
		"session.clone.verify": Version200, "session.clone.open": Version200,

		"sessions.list": Version300, "sessions.grep": Version300, "sessions.inspect": Version300,
		"sessions.lineage": Version300, "sessions.scan": Version300, "sessions.enrich": Version300,
		"sessions.jobs": Version300, "sessions.plan": Version300, "sessions.continue": Version300,
		"sessions.operation": Version300, "sessions.attach": Version300, "sessions.doctor": Version300,
		"sessions.query": Version300, "sessions.mutate": Version300,

		"terminal.backend.list": Version400, "terminal.backend.show": Version400,
		"terminal.backend.probe": Version400, "terminal.backend.doctor": Version400,
	}
	registered := Commands()
	if len(registered) != len(expected) {
		t.Fatalf("registered commands = %d, reviewed table = %d", len(registered), len(expected))
	}
	for command, want := range expected {
		got, err := RegisteredVersionForCommand(command)
		if err != nil {
			t.Fatalf("RegisteredVersionForCommand(%q): %v", command, err)
		}
		if got != want {
			t.Fatalf("command %q selects %q, want %q", command, got, want)
		}
	}
}

// TestUnimplementedSurfacesAreRefusedRatherThanEmitted proves that a registered
// tag this repository cannot build is refused with a distinguishable error, not
// admitted with an unchecked body. An unimplemented surface that quietly
// produced an object would advertise a capability that does not exist.
func TestUnimplementedSurfacesAreRefusedRatherThanEmitted(t *testing.T) {
	cases := []struct {
		command Command
		version Version
	}{
		{CommandCloneAdapters, Version200},
		{CommandClonePlan, Version200},
		{CommandCloneRun, Version200},
		{CommandCloneOpen, Version200},
		{CommandSessionsList, Version300},
		{CommandSessionsQuery, Version300},
		{CommandTerminalBackendList, Version400},
		{CommandTerminalBackendDoctor, Version400},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.command), func(t *testing.T) {
			if _, err := VersionForCommand(testCase.command); !errors.Is(err, ErrUnimplementedVersion) {
				t.Fatalf("VersionForCommand(%q) = %v, want ErrUnimplementedVersion", testCase.command, err)
			}
			_, err := New(Spec{Command: testCase.command, IDs: NoIDs(), Body: map[string]any{}})
			if !errors.Is(err, ErrUnimplementedVersion) {
				t.Fatalf("New(%q) = %v, want ErrUnimplementedVersion", testCase.command, err)
			}
			registered, err := RegisteredVersionForCommand(testCase.command)
			if err != nil || registered != testCase.version {
				t.Fatalf("registered version = %q/%v, want %q", registered, err, testCase.version)
			}
		})
	}
}

// TestUnknownCommandIsRefusedAsUnknownRatherThanUnimplemented separates the two
// facts. A tag AX does not register is not the same as a tag this build cannot
// produce, and collapsing them would let a later slice mistake one for the
// other.
func TestUnknownCommandIsRefusedAsUnknownRatherThanUnimplemented(t *testing.T) {
	for _, command := range []Command{"", "resolve", "Takeover", "peer_probe", "session.clone"} {
		if _, err := VersionForCommand(command); !errors.Is(err, ErrUnknownCommand) {
			t.Fatalf("VersionForCommand(%q) = %v, want ErrUnknownCommand", command, err)
		}
		if _, err := RegisteredVersionForCommand(command); !errors.Is(err, ErrUnknownCommand) {
			t.Fatalf("RegisteredVersionForCommand(%q) = %v, want ErrUnknownCommand", command, err)
		}
	}
}

// TestUnimplementedVersionsAreRefusedByEveryVersionEntryPoint narrows the
// version gate: 3.0.0 and 4.0.0 are registered by the pinned Section 1.5 row
// and are refused here as unimplemented, while an unregistered value is refused
// as unsupported.
func TestUnimplementedVersionsAreRefusedByEveryVersionEntryPoint(t *testing.T) {
	if got := len(Versions()); got != 4 {
		t.Fatalf("registered versions = %d, want the four pinned CLI Result versions", got)
	}
	if got := ImplementedVersions(); !reflect.DeepEqual(got, []Version{Version100, Version200}) {
		t.Fatalf("implemented versions = %v, want [1.0.0 2.0.0]", got)
	}
	for _, version := range []Version{Version300, Version400} {
		if _, err := Decode(version, []byte(normativeCLISuccess)); !errors.Is(err, ErrUnimplementedVersion) {
			t.Fatalf("Decode(%q) = %v, want ErrUnimplementedVersion", version, err)
		}
	}
	for _, version := range []Version{"", "1.0", "1.0.0.0", "2.1.0", "5.0.0", "01.0.0"} {
		if _, err := Decode(version, []byte(normativeCLISuccess)); !errors.Is(err, ErrUnsupportedVersion) {
			t.Fatalf("Decode(%q) = %v, want ErrUnsupportedVersion", version, err)
		}
	}
}

// TestConstructionDoesNotAliasTheCallerBodyGraph attacks containers at three
// depths and on both sides of an array. A copy narrowed to one level fails
// here, not only a copy deleted outright.
func TestConstructionDoesNotAliasTheCallerBodyGraph(t *testing.T) {
	spec := validSpec(t, CommandList)
	body := spec.Body.(map[string]any)
	sessions := body["sessions"].([]any)
	summary := sessions[0].(map[string]any)
	capabilities := summary["capabilities"].(map[string]any)
	native := capabilities["native_resume"].(map[string]any)

	result := mustResult(t, spec)
	encodedBefore, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	// Depth 1, 2, 3, and through an array element.
	body["partial"] = true
	sessions[0] = "not an object"
	summary["state"] = "tombstoned"
	capabilities["appserver"] = map[string]any{"status": "available", "enabled": true, "detail": ""}
	native["status"] = "unsupported"

	encodedAfter, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after mutation: %v", err)
	}
	if !bytes.Equal(encodedBefore, encodedAfter) {
		t.Fatalf("mutating the caller graph changed the result:\n before %s\n  after %s", encodedBefore, encodedAfter)
	}
}

// TestBodyAccessorDoesNotHandOutTheLiveContainer is the mirror image: a nested
// container handed back must not be writable into the validated object.
func TestBodyAccessorDoesNotHandOutTheLiveContainer(t *testing.T) {
	result := mustResult(t, validSpec(t, CommandList))
	encodedBefore, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	handed := result.Body()
	sessions := handed["sessions"].([]any)
	summary := sessions[0].(map[string]any)
	summary["capabilities"].(map[string]any)["appserver"] = map[string]any{
		"status": "available", "enabled": true, "detail": "",
	}
	summary["warnings"] = []any{"forged"}
	handed["partial"] = true

	encodedAfter, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after mutation: %v", err)
	}
	if !bytes.Equal(encodedBefore, encodedAfter) {
		t.Fatalf("writing through Body changed the result")
	}
}

// TestBodyOutsideTheCommonDataModelIsRefused proves that New does not accept a
// Go value the wire contract forbids. A float, an integer at or beyond 2^53, and
// a value type outside the model are each refused on their own row.
func TestBodyOutsideTheCommonDataModelIsRefused(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"floating point", 1.5},
		{"unsafe integer", json.Number("9007199254740992")},
		{"channel", make(chan int)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			spec := validSpec(t, CommandCancel)
			spec.Body = mutateBody(spec.Body.(map[string]any), "name", testCase.value)
			if _, err := New(spec); err == nil {
				t.Fatalf("New admitted %s", testCase.name)
			}
		})
	}
}
