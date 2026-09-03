package axerror

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// normativeExample is the Section 15.1 failure object, copied verbatim from the
// pinned document at SPEC.md lines 10986-11000.
const normativeExample = `{
  "schema": "urn:ax:schema:error",
  "schema_version": "1.0.0",
  "code": "workspace_conflict",
  "message": "destination differs from its last materialized checkpoint",
  "exit_code": 5,
  "retryable": false,
  "operation_id": "0198f4c8-b180-7299-9273-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "details": {
    "expected_checkpoint": "sha256:2222222222222222222222222222222222222222222222222222222222222222",
    "remediations": ["diff", "copy", "worktree", "replace_managed_replica"]
  }
}`

func mustUUIDv7(test *testing.T, value string) scalar.UUIDv7 {
	test.Helper()
	parsed, err := scalar.ParseUUIDv7(value)
	if err != nil {
		test.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

// TestNewProducesTheNormativeStructuredErrorObject drives the real constructor
// and requires the encoded bytes to equal the canonical form of the pinned
// example. The exit status is not supplied by the caller anywhere in this test:
// it is resolved from the registry, so a wrong mapping would show up as a byte
// difference here rather than as a missing assertion.
func TestNewProducesTheNormativeStructuredErrorObject(test *testing.T) {
	failure, err := New(Spec{
		Version: Version100,
		Code:    "workspace_conflict",
		Message: "destination differs from its last materialized checkpoint",
		IDs: NoIDs().
			WithOperation(mustUUIDv7(test, "0198f4c8-b180-7299-9273-1234567890ab")).
			WithSession(mustUUIDv7(test, "0198f4c8-3e70-7a11-8a2b-1234567890ab")),
		Details: Details{
			"expected_checkpoint": "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			"remediations":        []any{"diff", "copy", "worktree", "replace_managed_replica"},
		},
	})
	if err != nil {
		test.Fatalf("New: %v", err)
	}
	if failure.ExitCode() != 5 {
		test.Fatalf("exit_code = %d, the pinned example carries 5", failure.ExitCode())
	}
	if failure.Retryable() {
		test.Fatal("retryable defaulted to true")
	}
	if failure.Version() != Version100 || failure.Code() != "workspace_conflict" {
		test.Fatalf("envelope identity drifted: %s %s", failure.Version(), failure.Code())
	}
	if !failure.CodeRegistered() {
		test.Fatal("a constructed code reports as unregistered")
	}

	encoded, err := json.Marshal(failure)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	got, err := canonicaljson.Canonicalize(encoded)
	if err != nil {
		test.Fatalf("canonicalize produced object: %v", err)
	}
	want, err := canonicaljson.Canonicalize([]byte(normativeExample))
	if err != nil {
		test.Fatalf("canonicalize pinned example: %v", err)
	}
	if string(got) != string(want) {
		test.Fatalf("encoded object\n got %s\nwant %s", got, want)
	}

	// The pinned example round-trips through the reader as the same object.
	decoded, err := Decode(Version100, []byte(normativeExample))
	if err != nil {
		test.Fatalf("Decode pinned example: %v", err)
	}
	roundTripped, err := json.Marshal(decoded)
	if err != nil {
		test.Fatalf("marshal decoded: %v", err)
	}
	canonicalRoundTrip, err := canonicaljson.Canonicalize(roundTripped)
	if err != nil {
		test.Fatalf("canonicalize round trip: %v", err)
	}
	if string(canonicalRoundTrip) != string(want) {
		test.Fatalf("round trip\n got %s\nwant %s", canonicalRoundTrip, want)
	}
}

// TestOptionalIdentifiersAreOmittedNotNulled checks the two optional members.
// An object built with NoIDs carries neither member, and an object that knows
// one carries exactly that one.
func TestOptionalIdentifiersAreOmittedNotNulled(test *testing.T) {
	failure, err := New(Spec{
		Version: Version100,
		Code:    "not_found",
		Message: "no such session",
		IDs:     NoIDs(),
		Details: Details{},
	})
	if err != nil {
		test.Fatalf("New: %v", err)
	}
	encoded, err := json.Marshal(failure)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	for _, absent := range []string{"operation_id", "session_id"} {
		if strings.Contains(string(encoded), absent) {
			test.Fatalf("unknown %s was emitted: %s", absent, encoded)
		}
	}
	if !strings.Contains(string(encoded), `"details":{}`) {
		test.Fatalf("required empty details member was not emitted: %s", encoded)
	}
	if _, known := failure.IDs().Operation(); known {
		test.Fatal("NoIDs reported a known operation identifier")
	}

	sessionScoped, err := New(Spec{
		Version: Version110,
		Code:    "source_not_quiescent",
		Message: "source session is still producing output",
		IDs:     NoIDs().WithSession(mustUUIDv7(test, "0198f4c8-3e70-7a11-8a2b-1234567890ab")),
		Details: Details{},
	})
	if err != nil {
		test.Fatalf("New: %v", err)
	}
	encoded, err = json.Marshal(sessionScoped)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), "operation_id") {
		test.Fatalf("unknown operation_id was emitted: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"session_id":"0198f4c8-3e70-7a11-8a2b-1234567890ab"`) {
		test.Fatalf("known session_id was not emitted: %s", encoded)
	}
}

// TestUnknownDetailKeysNeverChangeAnyMachineAnswer is the Section 15.1 reader
// obligation as a behavioural gate: readers "MUST never infer success,
// authority, or a remediation action from" unknown detail keys. Each accessor
// is compared between an object with an empty detail map and an object whose
// details assert success, authority, and a remediation. Nothing may move.
func TestUnknownDetailKeysNeverChangeAnyMachineAnswer(test *testing.T) {
	build := func(details Details) *Error {
		test.Helper()
		failure, err := New(Spec{
			Version: Version120,
			Code:    "policy_refused",
			Message: "the requested move is refused by policy",
			IDs:     NoIDs(),
			Details: details,
		})
		if err != nil {
			test.Fatalf("New: %v", err)
		}
		return failure
	}
	plain := build(Details{})
	adversarial := build(Details{
		"ok":             true,
		"success":        true,
		"authorized":     true,
		"authority":      "granted",
		"retryable":      true,
		"exit_code":      json.Number("0"),
		"remediation":    "retry the identical request",
		"schema_version": "1.3.0",
	})

	if plain.Code() != adversarial.Code() {
		test.Fatalf("details changed the code: %s vs %s", plain.Code(), adversarial.Code())
	}
	if plain.ExitCode() != adversarial.ExitCode() {
		test.Fatalf("details changed the exit status: %d vs %d", plain.ExitCode(), adversarial.ExitCode())
	}
	if plain.Retryable() != adversarial.Retryable() {
		test.Fatalf("details changed retryability: %t vs %t", plain.Retryable(), adversarial.Retryable())
	}
	if plain.Version() != adversarial.Version() {
		test.Fatalf("details changed the version: %s vs %s", plain.Version(), adversarial.Version())
	}
	if plain.CodeRegistered() != adversarial.CodeRegistered() {
		test.Fatal("details changed the registered-code answer")
	}
	if _, known := adversarial.IDs().Operation(); known {
		test.Fatal("details minted an operation identifier")
	}
	if _, known := adversarial.IDs().Session(); known {
		test.Fatal("details minted a session identifier")
	}
	if adversarial.ExitCode() == 0 {
		test.Fatal("a failure object reported the success exit status")
	}
	// The same holds after a decode, where the detail map came from a peer.
	encoded, err := json.Marshal(adversarial)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	decoded, err := Decode(Version120, encoded)
	if err != nil {
		test.Fatalf("Decode: %v", err)
	}
	if decoded.Retryable() || decoded.ExitCode() != plain.ExitCode() || decoded.Code() != plain.Code() {
		test.Fatalf("decoded object took a machine answer from its details: %+v", decoded)
	}
}

// TestErrorRenderingAndCauseChain checks that the Go rendering carries the code
// and the human text and never the cause, while errors.Is still reaches the
// cause for local handling.
func TestErrorRenderingAndCauseChain(test *testing.T) {
	sentinel := errors.New("provider stderr: token=SECRET-VALUE-0001 refused")
	failure, err := New(Spec{
		Version: Version100,
		Code:    "provider_process_failed",
		Message: "the provider plugin exited before answering",
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   sentinel,
	})
	if err != nil {
		test.Fatalf("New: %v", err)
	}
	rendered := failure.Error()
	if !strings.HasPrefix(rendered, "provider_process_failed: ") {
		test.Fatalf("rendering does not lead with the code: %q", rendered)
	}
	if strings.Contains(rendered, "SECRET-VALUE-0001") {
		test.Fatalf("rendering carried the cause: %q", rendered)
	}
	if !errors.Is(failure, sentinel) {
		test.Fatal("errors.Is cannot reach the local cause")
	}
}
