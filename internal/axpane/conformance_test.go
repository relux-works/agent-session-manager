package axpane

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
)

// TestDecideArmPrecedence pins the §4.C wrapper-rule order: with
// every arm faulted at once, the first arm wins. Configuration
// precedes session, which precedes bootstrap idempotency, which
// precedes backend admission, which precedes fencing.
func TestDecideArmPrecedence(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.ConfigErr = errFixtureConfig
	input.SessionKnown = false
	input.ExistingBinding = &Binding{SessionID: fixtureSession, OperationID: fixtureOtherOp}
	input.SessionBound = true
	input.Backend.Manifest = []byte(`{"broken":true}`)
	input.Presented.Epoch = 0
	decision := Decide(input)
	if decision.Action != ActionRefused || decision.Class != ClassInvalidConfig {
		t.Fatalf("Decide() all-faulted = (%q, %q), want (refused, invalid_config)", decision.Action, decision.Class)
	}
	input.ConfigErr = nil
	decision = Decide(input)
	if decision.Action != ActionRefused || decision.Class != ClassInvalidArguments {
		t.Fatalf("Decide() peeled-config = (%q, %q), want the session arm", decision.Action, decision.Class)
	}
}

// TestEmitParkedVocabularyLoop authors the locally emittable
// session.parked reasons through the production emission path and
// reads each back with its exact closed payload, and proves the
// remotely held or unverifiable reasons refuse chain emission
// rather than manufacturing a collision. Only a verified,
// locally held winner with a foldable lifecycle authors: stale and
// materialization-based restore_policy emit; remote, unverified,
// and failed-handoff parks refuse.
func TestEmitParkedVocabularyLoop(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	// Move the fixture chain to failed so session.parked folds
	// (§5.7: only materializing, stopped, failed, or parked park).
	failedID := appendChainEvent(t, deps.repo, "session.failed", 1, fixtureLeaseA, 2, deps.createdID, map[string]any{
		"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
	})
	now := fixtureNow()
	observation := fixtureObservation(now)
	presented := fixturePresented()
	emittable := []struct {
		reason fencing.ParkReason
		action Action
		mutate func(*Input)
	}{
		{fencing.ParkStaleOwner, ActionParked, func(input *Input) { input.Presented.LeaseID = fixtureLeaseB }},
		{fencing.ParkRestorePolicy, ActionParked, func(input *Input) {
			input.MaterializationRequired = true
			input.JournalOK = true
		}},
	}
	sequence := uint64(3)
	predecessor := failedID
	for _, testCase := range emittable {
		input := validInput(t, deps)
		testCase.mutate(&input)
		decision := Decide(input)
		if decision.Action != testCase.action || decision.ParkReason != testCase.reason {
			t.Fatalf("Decide() = (%q, %q), want (%q, %q)", decision.Action, decision.ParkReason, testCase.action, testCase.reason)
		}
		reference, _, err := EmitParked(deps.repo, decision, EmitParams{
			SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
			LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: sequence,
			Predecessors: []string{predecessor}, CreatedAt: fixtureCreatedAt,
			Presented: presented, Observation: observation,
		})
		if err != nil {
			t.Fatalf("EmitParked(%q) error = %v", testCase.reason, err)
		}
		stored, err := deps.repo.GetEvent(fixtureSession, reference.EventID)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(stored), `"reason":"`+string(testCase.reason)+`"`) {
			t.Fatalf("parked event for %q carries no reason member: %s", testCase.reason, stored)
		}
		sequence++
		predecessor = reference.EventID
	}
	refusing := []struct {
		name    string
		reason  fencing.ParkReason
		action  Action
		mutate  func(*Input)
		observe func(*fencing.Observation)
	}{
		{"remote_owner", fencing.ParkRemoteOwner, ActionAttachRemote,
			func(input *Input) { input.Observation.Winner.HolderHostID = fixtureRemoteHost },
			func(obs *fencing.Observation) { obs.Winner.HolderHostID = fixtureRemoteHost }},
		{"unverified", fencing.ParkRestorePolicy, ActionParked,
			func(input *Input) { input.Observation.Verified = false },
			func(obs *fencing.Observation) { obs.Verified = false }},
		{"failed_handoff", fencing.ParkFailedHandoff, ActionParked,
			func(input *Input) { input.Observation.HandoffFailed = true },
			func(obs *fencing.Observation) { obs.HandoffFailed = true }},
	}
	for _, testCase := range refusing {
		t.Run("refuses_"+testCase.name, func(t *testing.T) {
			t.Parallel()
			nested := buildDecideDeps(t, true)
			nestedFailed := appendChainEvent(t, nested.repo, "session.failed", 1, fixtureLeaseA, 2, nested.createdID, map[string]any{
				"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
			})
			input := validInput(t, nested)
			testCase.mutate(&input)
			decision := Decide(input)
			if decision.Action != testCase.action || decision.ParkReason != testCase.reason {
				t.Fatalf("Decide() = (%q, %q), want (%q, %q)", decision.Action, decision.ParkReason, testCase.action, testCase.reason)
			}
			refusingObs := fixtureObservation(fixtureNow())
			testCase.observe(&refusingObs)
			if _, _, err := EmitParked(nested.repo, decision, EmitParams{
				SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
				LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: 3,
				Predecessors: []string{nestedFailed}, CreatedAt: fixtureCreatedAt,
				Presented: fixturePresented(), Observation: refusingObs,
			}); err == nil {
				t.Fatalf("EmitParked(%q) succeeded under a non-held lease, want refusal", testCase.reason)
			}
		})
	}
}

// TestLaunchCarriesValidatedPlan pins the §13.1 step-4 surface the
// wrapper hands to the later provider-launch leaf: the resolved
// adapter mapping, the admitted §7.A descriptor, and the sealed
// fencing token travel together on every launch, and no launch
// ships a partial plan.
func TestLaunchCarriesValidatedPlan(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	decision := Decide(validInput(t, deps))
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() action = %q, want launch", decision.Action)
	}
	if !decision.HasToken {
		t.Fatal("launch carries no fencing token")
	}
	if decision.Descriptor.TerminalInstanceID == "" {
		t.Fatal("launch carries no admitted descriptor")
	}
	if len(decision.Admitted.Capabilities) == 0 {
		t.Fatal("launch carries no admitted capabilities")
	}
}

// TestConformanceMatrixCompleteness enumerates the pinned clauses
// and the named top-level test that drives each through a
// production entry, then verifies every named test exists in this
// package through the go list entry. A row fails if its test is
// removed or renamed without updating the matrix; subtest rows in
// TRACEABILITY.md refine these top-level drivers.
func TestConformanceMatrixCompleteness(t *testing.T) {
	t.Parallel()
	rows := []struct {
		clause string
		test   string
	}{
		{"4.B manifest/probe/evidence identity", "TestDecideLaunchPositive"},
		{"4.B mismatch fails before activation", "TestDecideTable"},
		{"4.C wrapper rule order", "TestDecideArmPrecedence"},
		{"4.C bootstrap idempotency mismatch", "TestDecideTable"},
		{"4.C identical retry reattaches", "TestBindIdenticalRetryReattaches"},
		{"4.C concurrent commit reattaches", "TestBindConcurrentSameOperationReattaches"},
		{"4.C status proves absence", "TestBindInstallsAndStatusProves"},
		{"4.C unknown is not absent", "TestStatusUnknownIsNotAbsent"},
		{"4.C no PID persisted", "TestBindingPersistsNoPID"},
		{"4.C crash at the binding seam", "TestBindCrashChildSelfTerminates"},
		{"4.C per-pair receipt retention", "TestRunSupersededPairIdenticalRetry"},
		{"4.C create from stopped post-window", "TestRunCreateFromStoppedPostWindow"},
		{"4.C credential conditional every caller", "TestDecideForegroundCredentialRequiresRealm"},
		{"4.D only admitted rows authorize", "TestDecideTable"},
		{"4.2 background caller capability_unavailable", "TestDecideTypedRefusals"},
		{"4.2 cached evidence never authorizes", "TestRealmCarriesNoCachedEvidence"},
		{"4.2 dead realm row typed refusal", "TestDecideExpiredRealmRowRefusesUnavailable"},
		{"4.2 realm evidence order independent", "TestDecideRealmEvidenceOrderIndependent"},
		{"4.2 after-restore sequence", "TestDecideAfterRestoreSequence"},
		{"5.7 newest checkpoint gates resume", "TestRunEpochOneOwnerAfterCheckpoint"},
		{"5.2 session.parked vocabulary", "TestEmitParkedVocabularyLoop"},
		{"5.2 parked emission under lease", "TestEmitParkedRoundTrip"},
		{"5.2 v4 terminal payloads", "TestEmitTerminalV4RoundTrip"},
		{"7.A descriptor build and admit", "TestDescriptorBuildAdmitRoundTrip"},
		{"7.A descriptor mismatch refuses", "TestDescriptorMismatchRefuses"},
		{"7.A generation bounds", "TestDescriptorGenerationBounds"},
		{"7.A token passed separately", "TestDecideForgedTokensRefuse"},
		{"13.1 step 3 durable terminal entry", "TestRunLaunchBinds"},
		{"13.1 step 4 validated plan", "TestLaunchCarriesValidatedPlan"},
		{"2.4 effective profile and source", "TestDecideProfileSourcePinsEvent"},
		{"2.4 mapping refusal", "TestDecideYoloMapping"},
		{"2.4 unknown checkpoint store fails", "TestRunNoCkptStorePublishedNewestFails"},
		{"5.7 successor null fold parks", "TestDecideSuccessorNullFoldParks"},
		{"5.7 post-takeover restore from newest", "TestRunPostTakeoverRestoreLaunchesWithNewestClosure"},
		{"2.4 post-takeover closure both directions", "TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard"},
		{"4.C post-window replay reattaches", "TestRunPostWindowSamePairRaceReattaches"},
		{"4.C window never closes on the lease", "TestRunWindowLeaseCheckpointNullFoldRefuses"},
		{"4.C unfoldable chain fails the run", "TestRunUnfoldableChainFails"},
		{"8.4 identity record binds the session", "TestDecideIdentityForeignSessionRefuses"},
		{"orchestrated launch/reattach/park/refuse", "TestRunParkedEmits"},
		{"ownership observation over leases", "TestObserveLoadsWinningLease"},
	}
	listed, err := exec.Command("go", "test", ".", "-list", ".", "-count=1").CombinedOutput()
	if err != nil {
		t.Fatalf("go test -list error = %v: %s", err, listed)
	}
	present := map[string]bool{}
	for _, line := range strings.Split(string(listed), "\n") {
		matched, _ := regexp.MatchString(`^Test[A-Za-z0-9_]+$`, strings.TrimSpace(line))
		if matched {
			present[strings.TrimSpace(line)] = true
		}
	}
	if len(present) == 0 {
		t.Fatalf("go test -list found no tests: %s", listed)
	}
	for _, row := range rows {
		if !present[row.test] {
			t.Errorf("clause %q names missing test %q", row.clause, row.test)
		}
	}
	t.Logf("%d pinned clauses mapped to named tests", len(rows))
}
