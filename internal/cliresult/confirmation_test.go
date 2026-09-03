package cliresult

import (
	"strings"
	"testing"
)

// TestDestructiveOperationsCarryTheirDocumentedExpectationFlags pins the
// reviewed table against the Section 14.1 grammar it is quoted from.
func TestDestructiveOperationsCarryTheirDocumentedExpectationFlags(t *testing.T) {
	reviewed := map[DestructiveOperation][]string{
		OperationForceTakeover:         {"--expect-epoch", "--expect-owner"},
		OperationReplaceManagedReplica: {"--expect-checkpoint"},
	}
	operations := DestructiveOperations()
	if len(operations) != len(reviewed) {
		t.Fatalf("governed operations = %v, reviewed table has %d rows", operations, len(reviewed))
	}
	for operation, want := range reviewed {
		got, err := ExpectationFlags(operation)
		if err != nil {
			t.Fatalf("ExpectationFlags(%q): %v", operation, err)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%q expects %v, reviewed table says %v", operation, got, want)
		}
	}
	if _, err := ExpectationFlags("stop --force"); err == nil {
		t.Fatalf("ExpectationFlags admitted an operation the pinned document does not govern")
	}
}

// TestInteractiveDestructiveOperationMustPrompt narrows the Section 14.2
// sentence "destructive or split-brain-risk operations MUST prompt in
// interactive mode".
func TestInteractiveDestructiveOperationMustPrompt(t *testing.T) {
	invocation, failure := ParseCommonFlags(SurfaceTakeover, []string{
		"--force", "--expect-owner", fixtureSourceHostID, "--expect-epoch", "5",
	})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	confirmation, refusal := RequireConfirmation(
		OperationForceTakeover, invocation, []string{"--expect-owner", "--expect-epoch"})
	if refusal != nil {
		t.Fatalf("RequireConfirmation: %s", refusal.Message())
	}
	if !confirmation.PromptRequired {
		t.Fatalf("an interactive force takeover was allowed to proceed without a prompt")
	}
	// --yes does not remove the interactive prompt obligation, because Section
	// 14.2 states the prompt for interactive mode and --yes for non-interactive
	// mode as two separate requirements.
	invocation, failure = ParseCommonFlags(SurfaceTakeover, []string{"--force", "--yes"})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	confirmation, refusal = RequireConfirmation(
		OperationForceTakeover, invocation, []string{"--expect-owner", "--expect-epoch"})
	if refusal != nil || !confirmation.PromptRequired {
		t.Fatalf("--yes suppressed the interactive prompt: %v/%v", confirmation, refusal)
	}
}

// TestNonInteractiveDestructiveOperationRequiresYes narrows the second half of
// the same sentence: such an operation must "require --yes plus every
// documented expectation flag in non-interactive mode".
func TestNonInteractiveDestructiveOperationRequiresYes(t *testing.T) {
	complete := []string{"--expect-owner", "--expect-epoch"}
	withoutYes, failure := ParseCommonFlags(SurfaceTakeover, []string{"--force", "--non-interactive"})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	_, refusal := RequireConfirmation(OperationForceTakeover, withoutYes, complete)
	if refusal == nil {
		t.Fatalf("a non-interactive force takeover proceeded without --yes")
	}
	if refusal.Code() != "confirmation_required" || refusal.ExitCode() != 16 {
		t.Fatalf("refusal = %q/%d, want confirmation_required/16", refusal.Code(), refusal.ExitCode())
	}

	withYes, failure := ParseCommonFlags(SurfaceTakeover, []string{"--force", "--non-interactive", "--yes"})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	confirmation, refusal := RequireConfirmation(OperationForceTakeover, withYes, complete)
	if refusal != nil {
		t.Fatalf("RequireConfirmation: %s", refusal.Message())
	}
	if confirmation.PromptRequired {
		t.Fatalf("a non-interactive run was told to prompt, which --non-interactive forbids")
	}
}

// TestYesAloneNeverBypassesAnExpectedOwnerEpochOrCheckpointCheck narrows the
// Section 14.2 sentence "--yes alone MUST NOT bypass an expected
// owner/epoch/checkpoint check". Each expectation flag is withheld on its own
// row, so a gate narrowed to require only one of a pair still fails.
func TestYesAloneNeverBypassesAnExpectedOwnerEpochOrCheckpointCheck(t *testing.T) {
	cases := []struct {
		name      string
		surface   SurfaceCommand
		operation DestructiveOperation
		supplied  []string
		missing   string
	}{
		{"force takeover without either", SurfaceTakeover, OperationForceTakeover, nil, "--expect-epoch"},
		{"force takeover without the owner", SurfaceTakeover, OperationForceTakeover,
			[]string{"--expect-epoch"}, "--expect-owner"},
		{"force takeover without the epoch", SurfaceTakeover, OperationForceTakeover,
			[]string{"--expect-owner"}, "--expect-epoch"},
		{"replacement without the checkpoint", SurfaceMaterialize, OperationReplaceManagedReplica,
			nil, "--expect-checkpoint"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, argv := range [][]string{
				{"--yes"},
				{"--yes", "--non-interactive"},
				{"--non-interactive"},
				nil,
			} {
				invocation, failure := ParseCommonFlags(testCase.surface, argv)
				if failure != nil {
					t.Fatalf("ParseCommonFlags(%v): %s", argv, failure.Message())
				}
				_, refusal := RequireConfirmation(testCase.operation, invocation, testCase.supplied)
				if refusal == nil {
					t.Fatalf("%v proceeded without %s", argv, testCase.missing)
				}
				if refusal.Code() != "invalid_arguments" || refusal.ExitCode() != 2 {
					t.Fatalf("refusal = %q/%d, want invalid_arguments/2", refusal.Code(), refusal.ExitCode())
				}
				if !strings.Contains(refusal.Message(), testCase.missing) {
					t.Fatalf("refusal %q does not name the missing %s", refusal.Message(), testCase.missing)
				}
			}
		})
	}
}

// TestRequireConfirmationRefusesAnUngovernedOperationOrMissingInvocation proves
// the gate has no permissive default.
func TestRequireConfirmationRefusesAnUngovernedOperationOrMissingInvocation(t *testing.T) {
	invocation, failure := ParseCommonFlags(SurfaceStop, []string{"--yes", "--non-interactive"})
	if failure != nil {
		t.Fatalf("ParseCommonFlags: %s", failure.Message())
	}
	if _, refusal := RequireConfirmation("stop --force", invocation, nil); refusal == nil {
		t.Fatalf("an ungoverned operation was allowed through the confirmation gate")
	}
	if _, refusal := RequireConfirmation(OperationForceTakeover, nil, nil); refusal == nil {
		t.Fatalf("a nil invocation was allowed through the confirmation gate")
	}
}
