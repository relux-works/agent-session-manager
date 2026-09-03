package cliresult

import (
	"strings"
	"testing"
)

// TestContractBoundStatesEveryLimitThisPackageActuallyHas asserts that the
// disclosure in the code still says what the code does. A bound widened in the
// implementation while the constant keeps promising the narrower behaviour is
// the failure this test exists to catch.
func TestContractBoundStatesEveryLimitThisPackageActuallyHas(t *testing.T) {
	required := []string{
		"does not build CLI Result 3.0.0 or 4.0.0",
		"does not build the Section 14.1 clone bodies",
		"without a session kind",
		"absolute on any supported platform",
		"does not enforce the Section 18.1 total order",
		"never decides",
	}
	for _, phrase := range required {
		if !strings.Contains(ContractBound, phrase) {
			t.Fatalf("ContractBound no longer states %q", phrase)
		}
	}

	// Each stated limit is a real behaviour, not prose.
	if _, err := VersionForCommand(CommandSessionsList); err == nil {
		t.Fatalf("CLI Result 3.0.0 is claimed as unbuilt but a tag was admitted")
	}
	if _, err := VersionForCommand(CommandTerminalBackendList); err == nil {
		t.Fatalf("CLI Result 4.0.0 is claimed as unbuilt but a tag was admitted")
	}
	if _, err := VersionForCommand(CommandCloneRun); err == nil {
		t.Fatalf("a clone body is claimed as unbuilt but a tag was admitted")
	}
	spec := validSpec(t, CommandTakeover)
	spec.SessionKind = ""
	if _, err := New(spec); err == nil {
		t.Fatalf("the adoption rule is claimed to need a session kind but New admitted none")
	}
	windows := mustResult(t, specWithBody(t, CommandMaterialize,
		mutateBody(materializationSummary(), "destination_path", `C:\work`)))
	if err := windows.VerifyDestinationPlatform("linux"); err == nil {
		t.Fatalf("the platform narrowing hook does not narrow")
	}
}

// TestRefusalInventoryIsMeasuredRatherThanClaimed reports the ratio of
// registered command tags this repository builds. It is a measured count with
// its denominator stated, not a sentence about coverage: prose cannot
// distinguish 18 of 44 from 44 of 44.
func TestRefusalInventoryIsMeasuredRatherThanClaimed(t *testing.T) {
	registered := len(Commands())
	implemented := len(ImplementedCommands())
	if registered != 44 {
		t.Fatalf("registered command tags = %d, want 44 (18 legacy, 8 clone, 14 directory, 4 terminal)", registered)
	}
	if implemented != 18 {
		t.Fatalf("implemented command tags = %d of %d, want the 18 Section 14.2 tags", implemented, registered)
	}
	versions := len(Versions())
	built := len(ImplementedVersions())
	if versions != 4 || built != 2 {
		t.Fatalf("versions = %d/%d, want 2 of 4", built, versions)
	}
	surfaces := len(Surfaces())
	users := len(UserSurfaces())
	if surfaces != 31 || users != 29 {
		t.Fatalf("command surfaces = %d with %d user commands, want 31 and 29", surfaces, users)
	}
}
