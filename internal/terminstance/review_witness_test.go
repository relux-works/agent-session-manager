package terminstance

// Reviewer witnesses committed verbatim (RUN-260918-f6b44c): each
// passes on the pristine tree and fails under the narrowing mutant it
// witnesses. TestRV3W_M1 pins the expiry axis of the per-effect
// recheck (E3); TestRV3W_M2/M3/M4/M8 pin the P3-1..P3-4 arms.
// TestRV3_FencingNoLocalTupleComparison pins the fencing composition
// (no local tuple comparison in this package).

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// rv3StopContextWithDeadline builds a request-stop context whose
// deadline_at lies AFTER the authorization expires_at, so that an
// authorization expiring mid-operation is observable independently of
// the deadline arm (the shipped fixture has deadline < expiry, so the
// deadline always fires first).
func rv3StopContextWithDeadline(t *testing.T, deadline string) (MutationContext, Params) {
	t.Helper()
	key := fixtureInstance + "/stop/" + fixtureDigestB
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, deadline, authDoc("1", nil))
	context, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext(stop, deadline %s) error = %v", deadline, err)
	}
	return context, Params{SafeBoundaryEvidenceID: fixtureDigestB, GracefulTimeoutMs: 1000}
}

// TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect (WITNESS for RV3-M1):
// the authorization EXPIRES between the first and second stop effect
// while the lease tuple is unchanged and the deadline is still ahead.
// The per-effect recheck must refuse before the second effect with the
// literal expiry arm, after=unavailable and new_authorization, leaving
// exactly one committed effect.
func TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	// deadline 2026-09-03 > auth expiry 2026-09-02.
	context, params := rv3StopContextWithDeadline(t, "2026-09-03T00:00:00.000Z")
	afterExpiry, err := time.Parse(time.RFC3339Nano, "2026-09-02T01:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	effects := 0
	engine.Hooks = &EngineHooks{AfterEffect: func(terminalbackend.SideEffect) { effects++ }}
	engine.Now = func() time.Time {
		if effects >= 1 {
			return afterExpiry
		}
		return fixtureNow()
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, mustParseState(t, "quiescing"), testAdmitted("graceful_stop"))
	requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
	if performed := backend.performed(); len(performed) != 1 || performed[0] != terminalbackend.EffectGracefulStopRequested {
		t.Errorf("performed = %v, want exactly [graceful_stop_requested] before the expired authorization is caught", performed)
	}
}

// rv3Digests mints n distinct valid digests in ascending bytewise order.
func rv3Digests(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("sha256:%064x", i))
	}
	return out
}

// TestRV3W_M2_EvidenceBoundEdge (WITNESS for RV3-M2): the sorted-unique
// digest[0..256] bound is witnessed at its edge on both surfaces: 256
// admitted, 257 refused, for a replayed MutationResult (CheckResult) and
// for a status report (ExecuteStatus).
func TestRV3W_M2_EvidenceBoundEdge(t *testing.T) {
	context, _, source, _ := quiesceFixture(t)
	base := Result{
		OperationID: context.OperationID, SessionID: context.SessionID,
		TerminalInstanceID: context.TerminalInstanceID, TerminalBackendID: context.TerminalBackendID,
		ImplementationVersion: context.ImplementationVersion, ProtocolVersion: context.ProtocolVersion,
		BackendGeneration: context.BackendGeneration, Before: source, After: terminalbackend.StateQuiescing,
		Effects: []terminalbackend.SideEffect{terminalbackend.EffectInputClosed}, Disposition: DispositionReplaySame,
	}
	admitted := base
	admitted.EvidenceIDs = rv3Digests(256)
	if err := CheckResult(context, source, admitted); err != nil {
		t.Errorf("CheckResult(256 evidence IDs) error = %v, want admission", err)
	}
	over := base
	over.EvidenceIDs = rv3Digests(257)
	requireRefusal(t, CheckResult(context, source, over), "terminal_backend_protocol_error", "result evidence bound")

	backend := newMockBackend()
	observed := matchingObservation()
	observed.EvidenceIDs = rv3Digests(256)
	backend.observation = observed
	engine := testEngine(t, backend, testLease(), fixtureGen)
	if _, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect")); err != nil {
		t.Errorf("ExecuteStatus(256 evidence IDs) error = %v, want admission", err)
	}
	observed.EvidenceIDs = rv3Digests(257)
	backend.observation = observed
	_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status evidence bound")
}

// TestRV3W_M3_ConditionalCapabilityDisposition (WITNESS for RV3-M3):
// the capability-conditional refusal arm at the engine carries the
// class-mapped disposition for an uncommitted capability_unproven:
// required_operator_action, on every conditional (headless create,
// wait safe-boundary, wait provider-kind).
func TestRV3W_M3_ConditionalCapabilityDisposition(t *testing.T) {
	t.Run("headless create from stopped", func(t *testing.T) {
		create := testCreateContext(t)
		create.params.Interactive = false
		engine := testEngine(t, newMockBackend(), testLease(), fixtureGen)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "stopped"), testAdmitted("terminal_state_retention"))
		requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
		requireLiteral(t, "after", string(result.After), "stopped")
		requireLiteral(t, "disposition", string(result.Disposition), "required_operator_action")
	})
	t.Run("wait without safe_boundary_observation", func(t *testing.T) {
		context, params, source, _ := waitFixture(t, ProofAXCheckpointBoundary)
		engine := testEngine(t, newMockBackend(), testLease(), fixtureGen)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, testAdmitted("provider_process_observation"))
		requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
		requireLiteral(t, "after", string(result.After), "quiescing")
		requireLiteral(t, "disposition", string(result.Disposition), "required_operator_action")
	})
	t.Run("wait provider kind without provider_process_observation", func(t *testing.T) {
		context, params, source, admitted := waitFixture(t, ProofProviderQuiescence)
		engine := testEngine(t, newMockBackend(), testLease(), fixtureGen)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
		requireLiteral(t, "after", string(result.After), "quiescing")
		requireLiteral(t, "disposition", string(result.Disposition), "required_operator_action")
	})
}

// TestRV3W_M4_ProofKindSiblingNamesRefused (WITNESS for RV3-M4): the
// landed sibling names a caller could plausibly confuse with a proof
// kind (the capability names and the side-effect name from the same
// section) are refused at the direct entry and through the wait body.
func TestRV3W_M4_ProofKindSiblingNamesRefused(t *testing.T) {
	for _, sibling := range []string{"provider_process_observation", "safe_boundary_observation", "safe_boundary_observed", "provider_exit", "PROVIDER_QUIESCENCE", "provider_quiescence "} {
		_, err := ParseProviderProofKind(sibling)
		requireRefusal(t, err, "terminal_backend_protocol_error", "provider proof vocabulary")
		body := `{"context":` + contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, fixtureInstance+"/boundary/"+fixtureQuiescence+"/"+sibling, fixtureDeadline, authDoc("1", nil)) +
			`,"quiescence_generation":"` + fixtureQuiescence + `","provider_proof_kind":"` + sibling + `","timeout_ms":1000}`
		_, _, err = ParseOperationBody("wait-safe-boundary", []byte(body))
		requireRefusal(t, err, "terminal_backend_protocol_error", "provider proof vocabulary")
	}
}

// TestRV3W_M8_StatusLastOperationGrammar (WITNESS for RV3-M8): a status
// report whose last_operation_id is present but not a UUIDv7 (empty or
// garbage) is refused at the production status entry, never adopted.
func TestRV3W_M8_StatusLastOperationGrammar(t *testing.T) {
	for _, bad := range []string{"", "nope", "0198F4C8-8E50-7F66-8F70-CCCCCCCCCCC1"} {
		backend := newMockBackend()
		observed := matchingObservation()
		value := bad
		observed.LastOperationID = &value
		backend.observation = observed
		engine := testEngine(t, backend, testLease(), fixtureGen)
		_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
		requireRefusal(t, err, "terminal_backend_protocol_error", "status report operation")
	}
}

// TestRV3_FencingNoLocalTupleComparison pins the composition claim by
// source text: fencing.go never reads Winner.Epoch/Winner.LeaseID or the
// presented tuple outside the landed verdict calls.
func TestRV3_FencingNoLocalTupleComparison(t *testing.T) {
	src := codeLinesOnly(readProductionFile(t, "fencing.go"))
	for _, forbidden := range []string{"presented.Epoch", "presented.LeaseID", "Winner.Epoch", "Winner.LeaseID", "Grant.Token"} {
		if strings.Contains(src, forbidden) {
			t.Errorf("fencing.go references %q: a local tuple comparison would fork the landed gate", forbidden)
		}
	}
	if strings.Count(src, "fencing.Authorize(") != 2 {
		t.Errorf("fencing.go calls fencing.Authorize %d times, want exactly the two composed questions", strings.Count(src, "fencing.Authorize("))
	}
}

// readProductionFile reads one production source file of this package.
func readProductionFile(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(raw)
}

// codeLinesOnly drops full-line comments so a doc sentence naming a
// member is not mistaken for a code reference.
func codeLinesOnly(src string) string {
	var kept []string
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
