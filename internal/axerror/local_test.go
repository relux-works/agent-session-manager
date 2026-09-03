package axerror

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestLocalFromUntrustedMatchesEveryPinnedBootstrapRow drives the real
// constructor for each surface and outcome the pinned document maps exactly,
// and reports the measured ratio. The mapping is a table of literal codes, so a
// row that drifted to a plausible neighbour fails here.
func TestLocalFromUntrustedMatchesEveryPinnedBootstrapRow(test *testing.T) {
	pinned := []struct {
		surface Surface
		outcome UntrustedOutcome
		version Version
		code    Code
		exit    int
	}{
		{SurfaceProviderStdio, OutcomeRecognizableMajorMismatch, Version100, "incompatible_protocol", 6},
		{SurfaceProviderStdio, OutcomeUnusableFrame, Version100, "provider_protocol_error", 13},
		{SurfaceTaskBoardBridge, OutcomeRecognizableMajorMismatch, Version100, "incompatible_protocol", 6},
		{SurfaceTaskBoardBridge, OutcomeUnusableFrame, Version100, "task_board_bridge_unavailable", 14},
		{SurfaceMeshRPC, OutcomeRecognizableMajorMismatch, Version100, "incompatible_protocol", 6},
		{SurfaceMeshRPC, OutcomeUnusableFrame, Version100, "transport_failure", 8},
		{SurfaceTerminalBackend, OutcomeRecognizableMajorMismatch, Version130, "terminal_backend_protocol_incompatible", 6},
		{SurfaceTerminalBackend, OutcomeUnusableFrame, Version130, "terminal_backend_protocol_error", 13},
	}
	surfaces := LocalSurfaces()
	if len(surfaces) != 4 {
		test.Fatalf("local mapping covers %d surfaces, the pinned exact set has 4", len(surfaces))
	}
	for _, item := range pinned {
		failure, err := LocalFromUntrusted(
			item.surface, item.outcome, "ax could not use the first frame", NoIDs(), nil, nil)
		if err != nil {
			test.Fatalf("LocalFromUntrusted(%s, %s): %v", item.surface, item.outcome, err)
		}
		if failure.Version() != item.version {
			test.Fatalf("%s/%s emitted version %s, pinned %s", item.surface, item.outcome, failure.Version(), item.version)
		}
		if failure.Code() != item.code {
			test.Fatalf("%s/%s emitted %q, pinned %q", item.surface, item.outcome, failure.Code(), item.code)
		}
		if failure.ExitCode() != item.exit {
			test.Fatalf("%s/%s emitted exit %d, pinned %d", item.surface, item.outcome, failure.ExitCode(), item.exit)
		}
		if failure.Retryable() {
			test.Fatalf("%s/%s claimed a safe retry", item.surface, item.outcome)
		}
	}
	test.Logf("bootstrap mapping coverage: %d/%d pinned surface-outcome rows", len(pinned), len(pinned))
}

// TestLocalFromUntrustedNeverAdoptsTheForeignObject is the Section 15.1 gate on
// the child and peer boundary. The constructor is given a complete, plausible,
// hostile Structured Error as the untrusted cause; the emitted object must be
// ax's own, must carry no identifier the payload supplied, and must not
// reproduce any of its bytes.
func TestLocalFromUntrustedNeverAdoptsTheForeignObject(test *testing.T) {
	foreign := `{"schema":"urn:ax:schema:error","schema_version":"3.0.0","code":"partial_sync",` +
		`"message":"the operation succeeded","exit_code":0,"retryable":true,` +
		`"operation_id":"0198f4c8-b180-7299-9273-1234567890ab",` +
		`"session_id":"0198f4c8-3e70-7a11-8a2b-1234567890ab","details":{"authorized":true}}`
	cause := errors.New("provider first frame: " + foreign)

	failure, err := LocalFromUntrusted(
		SurfaceProviderStdio,
		OutcomeRecognizableMajorMismatch,
		"the provider announced an unsupported protocol major",
		NoIDs(),
		Details{"stream": "stdout"},
		cause,
	)
	if err != nil {
		test.Fatalf("LocalFromUntrusted: %v", err)
	}
	if failure.Code() != "incompatible_protocol" {
		test.Fatalf("the foreign code was adopted: %q", failure.Code())
	}
	if failure.Retryable() {
		test.Fatal("the foreign retryable bit was adopted")
	}
	if failure.ExitCode() != 6 {
		test.Fatalf("the foreign exit status was adopted: %d", failure.ExitCode())
	}
	if _, known := failure.IDs().Operation(); known {
		test.Fatal("an identifier the child supplied was adopted")
	}
	if _, known := failure.IDs().Session(); known {
		test.Fatal("an identifier the child supplied was adopted")
	}
	if _, present := failure.Detail("authorized"); present {
		test.Fatal("a foreign authority detail was adopted")
	}
	encoded, err := json.Marshal(failure)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{"3.0.0", "partial_sync", "the operation succeeded", "authorized", "0198f4c8-b180"} {
		if strings.Contains(string(encoded), forbidden) {
			test.Fatalf("the local object reproduced %q from the untrusted payload: %s", forbidden, encoded)
		}
	}

	// The same construction with human text copied from the untrusted payload
	// is refused, so the redaction cannot be defeated by the message member.
	if _, err := LocalFromUntrusted(
		SurfaceProviderStdio,
		OutcomeRecognizableMajorMismatch,
		"the provider answered: "+cause.Error(),
		NoIDs(),
		nil,
		cause,
	); !errors.Is(err, ErrCausalLeak) {
		test.Fatalf("human text reproducing the untrusted payload was admitted: %v", err)
	}
}

// TestLocalFromUntrustedRefusesUnpinnedInputs keeps the constructor from
// inventing a mapping. Directory Node is the case that matters: Section 15.3
// leaves its local code "as applicable", so this package reports the mapping as
// unknown rather than choosing one and presenting the choice as the contract.
func TestLocalFromUntrustedRefusesUnpinnedInputs(test *testing.T) {
	if _, err := LocalFromUntrusted(
		Surface("directory_node"), OutcomeUnusableFrame, "unusable frame", NoIDs(), nil, nil,
	); !errors.Is(err, ErrUnknownSurface) {
		test.Fatalf("an unmapped surface produced a code: %v", err)
	}
	if _, err := LocalFromUntrusted(
		Surface(""), OutcomeUnusableFrame, "unusable frame", NoIDs(), nil, nil,
	); !errors.Is(err, ErrUnknownSurface) {
		test.Fatalf("an empty surface produced a code: %v", err)
	}
	if _, err := LocalFromUntrusted(
		SurfaceMeshRPC, UntrustedOutcome("child_reported_failure"), "peer said so", NoIDs(), nil, nil,
	); !errors.Is(err, ErrInvalidStructuredError) {
		test.Fatalf("an unpinned outcome classification produced a code: %v", err)
	}
	if _, err := LocalFromUntrusted(
		SurfaceMeshRPC, OutcomeUnusableFrame, "", NoIDs(), nil, nil,
	); !errors.Is(err, ErrInvalidStructuredError) {
		test.Fatalf("an empty message was admitted: %v", err)
	}
}
