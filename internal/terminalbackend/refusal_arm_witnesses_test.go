// Executed refusal-arm witnesses for the terminal backend packages.
//
// Every non-bound row of the white-box arm table
// (refusal_arm_inventory_test.go) carries exactly one witness here,
// keyed by the same identity the derivation uses. The harness runs each
// witness through a PUBLIC production entry point and requires the exact
// wire code at the exact static detail: a row that merely mentions the
// detail in some test's source proves nothing and resolves nothing.
//
// Witness construction rule for shared-clause sites: several sites share
// one (code, detail) clause (eleven sites share "document member type").
// Each prove documents the pass-earlier-sites construction — the input
// passes every earlier same-clause site on its entry path and fails at
// the named one. That construction orders the inputs, but it cannot
// observe a widened earlier guard: site attribution comes from the
// exercised-site audit in refusal_site_audit_test.go (internal test),
// which requires every derived non-bound line to fire at least once
// across the full suite run. A shadowed-dead arm reddens there even
// though its prove still passes through the sibling. Bound rows
// (defensive re-parses, unreachable vocabulary, decoder-contract and
// canonical-plumbing branches, the kind gate, the number walk) carry NO
// witness: no input refuses at their entry, and a witness would resolve
// through a sibling. Their pins live white-box in the pinned bound set.
//
// The both-direction harness runs in both packages: white-box proves
// derived-arms against declared-rows, and TestDerivedArmsAreAllWitnessed
// / TestWitnessedArmsAreAllDerived below prove declared-rows against
// executed witnesses, so a truncated witness registry reddens instead of
// passing vacuously.
package terminalbackend_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// armWitness proves one declared refusal arm at its public production
// entry. Key equals terminalbackend.ArmKey over the row identity exactly:
// TestWitnessedArmsAreAllDerived fails a witness that names nothing
// production declares. Entry names the public entry the prove function
// drives.
type armWitness struct {
	key   string
	entry string
	prove func(*testing.T)
}

// wkey renders one row identity in the shared witness key space,
// matching terminalbackend.ArmKey exactly.
func wkey(file, function, code, detail string, occurrence int) string {
	return fmt.Sprintf("%s|%s|%s|%q#%d", file, function, code, detail, occurrence)
}

// requireRegistryRefusal asserts err is a registry refusal satisfying
// check with the exact static wantDetail clause.
func requireRegistryRefusal(t *testing.T, err error, check func(error) bool, wantDetail string) {
	t.Helper()
	if !check(err) {
		t.Fatalf("error = %v, want a registry refusal at %q", err, wantDetail)
	}
	if got := refusalDetail(t, err); got != wantDetail {
		t.Fatalf("refusal detail = %q, want %q", got, wantDetail)
	}
}

// TestDerivedArmsAreAllWitnessed is the forward direction: every derived
// non-bound arm carries an executed witness. A planted arm (a new
// production site through an existing constructor) lands here as an
// unwitnessed arm.
func TestDerivedArmsAreAllWitnessed(t *testing.T) {
	t.Parallel()

	derived := terminalbackend.DerivedArmIdentities(t)
	declared := terminalbackend.DeclaredArmIdentities()
	bound := map[string]bool{}
	for _, row := range declared {
		if row.Bound != "" {
			bound[terminalbackend.ArmKey(row)] = true
		}
	}
	witnessed := map[string]bool{}
	for _, witness := range declaredArmWitnesses() {
		witnessed[witness.key] = true
	}
	var missing []string
	for _, arm := range derived {
		key := terminalbackend.ArmKey(arm)
		if bound[key] {
			continue
		}
		if !witnessed[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("derived refusal arm(s) with no executed witness at the production entry:\n  %s",
			strings.Join(missing, "\n  "))
	}
	t.Logf("refusal-arm witnesses: %d/%d derived arms executed", len(derived)-len(missing)-len(bound), len(derived))
}

// TestWitnessedArmsAreAllDerived is the reverse direction: every witness
// names a derived arm, and no witness names a bound arm (a bound arm has
// no refusing input by construction; witnessing it would resolve through
// a sibling). A deleted or narrowed production branch orphans its witness
// and fails here, which is also what makes a truncated derivation fail
// instead of passing vacuously.
func TestWitnessedArmsAreAllDerived(t *testing.T) {
	t.Parallel()

	derived := map[string]bool{}
	for _, arm := range terminalbackend.DerivedArmIdentities(t) {
		derived[terminalbackend.ArmKey(arm)] = true
	}
	bound := map[string]bool{}
	for _, row := range terminalbackend.DeclaredArmIdentities() {
		if row.Bound != "" {
			bound[terminalbackend.ArmKey(row)] = true
		}
	}
	var orphans []string
	for _, witness := range declaredArmWitnesses() {
		if bound[witness.key] {
			orphans = append(orphans, witness.key+" (bound arm must not carry a witness)")
			continue
		}
		if !derived[witness.key] {
			orphans = append(orphans, witness.key+" ("+witness.entry+")")
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Fatalf("witness(es) naming no derived production arm:\n  %s", strings.Join(orphans, "\n  "))
	}
}

// TestEveryDeclaredArmRefusesAtItsEntry drives every witness through its
// public production entry point and requires the attributed refusal.
func TestEveryDeclaredArmRefusesAtItsEntry(t *testing.T) {
	t.Parallel()

	for _, witness := range declaredArmWitnesses() {
		t.Run(witness.key, func(t *testing.T) {
			t.Parallel()
			witness.prove(t)
		})
	}
}

// conformanceWitnesses proves every non-bound conformance.go arm at its
// public entry. Multi-occurrence groups use ordered inputs: each prove
// passes the earlier same-clause sites and fails at the named one.
func conformanceWitnesses() []armWitness {
	return []armWitness{
		{
			key:   wkey("conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach authorization binding", 1),
			entry: "CheckAttachRequest",
			prove: func(t *testing.T) {
				// Wrong transport against a matching policy: the
				// transport parse passes, the binding comparison fires.
				err := terminalbackend.CheckAttachRequest(testAttachAuthorization(t), "trusted_private_mesh", true, conformanceNow())
				requireConformanceRefusal(t, err, "terminal_backend_unauthorized", "attach authorization binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach authorization expiry", 1),
			entry: "CheckAttachRequest",
			prove: func(t *testing.T) {
				// Matching transport and input, instant past expiry: the
				// binding comparison passes, the expiry comparison fires.
				expired := conformanceNow().Add(48 * time.Hour)
				err := terminalbackend.CheckAttachRequest(testAttachAuthorization(t), "local_only", true, expired)
				requireConformanceRefusal(t, err, "terminal_backend_unauthorized", "attach authorization expiry")
			},
		},
		{
			key:   wkey("conformance.go", "CheckAttachRequest", "CodeUnauthorized", "attach relay transport", 1),
			entry: "CheckAttachRequest",
			prove: func(t *testing.T) {
				// A relay policy matches on transport and input, so only
				// the relay bar can refuse.
				raw := `{"policy_evidence_id":"` + testDigest + `",` +
					`"authorizing_host_id":"` + conformanceHost + `",` +
					`"transport":"third_party_relay",` +
					`"input_authorized":false,` +
					`"issued_at":"` + conformanceIssued + `",` +
					`"expires_at":"` + conformanceExpires + `"}`
				auth, err := terminalbackend.ParseAttachAuthorization([]byte(raw))
				if err != nil {
					t.Fatalf("ParseAttachAuthorization(relay) error = %v", err)
				}
				err = terminalbackend.CheckAttachRequest(auth, "third_party_relay", false, conformanceNow())
				requireConformanceRefusal(t, err, "terminal_backend_unauthorized", "attach relay transport")
			},
		},
		{
			key:   wkey("conformance.go", "CheckAttachResult", "CodeUnauthorized", "attach input binding", 1),
			entry: "CheckAttachResult",
			prove: func(t *testing.T) {
				// Result grants input the request withheld: the triple
				// equality fails on the result leg.
				err := terminalbackend.CheckAttachResult(false, true, testAttachAuthorization(t))
				requireConformanceRefusal(t, err, "terminal_backend_unauthorized", "attach input binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint argv", 1),
			entry: "CheckEntrypoint",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckEntrypoint([]string{"codex", conformanceSessionA}, conformanceSessionA)
				requireConformanceRefusal(t, err, "local_precondition_failed", "entrypoint argv")
			},
		},
		{
			// Occurrence order in CheckEntrypoint: #1 parses the session
			// ID, #2 parses argv[2], #3 compares the two. Each prove
			// passes the earlier parses.
			key:   wkey("conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 1),
			entry: "CheckEntrypoint",
			prove: func(t *testing.T) {
				// Unparseable session ID: the argv shape passes, the
				// first session parse fires.
				err := terminalbackend.CheckEntrypoint([]string{"ax", "pane", conformanceSessionA}, "bogus")
				requireConformanceRefusal(t, err, "local_precondition_failed", "entrypoint session binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 2),
			entry: "CheckEntrypoint",
			prove: func(t *testing.T) {
				// Valid session ID, unparseable carried ID: #1 passes,
				// the second parse fires.
				err := terminalbackend.CheckEntrypoint([]string{"ax", "pane", "bogus"}, conformanceSessionA)
				requireConformanceRefusal(t, err, "local_precondition_failed", "entrypoint session binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckEntrypoint", "CodePreconditionFailed", "entrypoint session binding", 3),
			entry: "CheckEntrypoint",
			prove: func(t *testing.T) {
				// Two valid but differing IDs: both parses pass, the
				// comparison fires.
				err := terminalbackend.CheckEntrypoint([]string{"ax", "pane", conformanceSessionB}, conformanceSessionA)
				requireConformanceRefusal(t, err, "local_precondition_failed", "entrypoint session binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckErrorAllowed", "CodeProtocolError", "operation error vocabulary", 1),
			entry: "CheckErrorAllowed",
			prove: func(t *testing.T) {
				// A listed operation with an unlisted code: the operation
				// parse passes, the allowed-set lookup fires.
				err := terminalbackend.CheckErrorAllowed("create", "bogus_code")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "operation error vocabulary")
			},
		},
		{
			key:   wkey("conformance.go", "CheckReplicable", "CodeProtocolError", "replication exclusion", 1),
			entry: "CheckReplicable",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckReplicable([]string{"manifest_id", "binding_id"})
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "replication exclusion")
			},
		},
		{
			key:   wkey("conformance.go", "CheckStatusResult", "CodePreconditionFailed", "status attachability", 1),
			entry: "CheckStatusResult",
			prove: func(t *testing.T) {
				// Evidenced attach capability outside parked/active: the
				// identity, observation, and state parses pass, the
				// two-state attachability rule fires.
				present := true
				lastOp := "0198f4c8-8e50-7f66-8f70-1234567890ab"
				lastEffect := terminalbackend.EffectWrapperStarted
				result := terminalbackend.StatusResult{
					State: trueState(), IdentityMatch: true, WrapperPresent: true,
					ProviderPresent: &present, Attachable: true,
					LastOperationID: &lastOp, LastEffect: &lastEffect,
					ProviderRequested: true, ProviderEvidenced: true, AttachEvidenced: true,
				}
				err := terminalbackend.CheckStatusResult(true, result)
				requireConformanceRefusal(t, err, "local_precondition_failed", "status attachability")
			},
		},
		{
			// Occurrence order: #1 is the identityMatch contradiction
			// (false lookup, true report), #2 is the non-canonical false
			// form. #1's input fails the first comparison; #2's input
			// passes it (both false) and fails the form check.
			key:   wkey("conformance.go", "CheckStatusResult", "CodeProtocolError", "status identity binding", 1),
			entry: "CheckStatusResult",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckStatusResult(false, terminalbackend.StatusResult{State: "absent", IdentityMatch: true})
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckStatusResult", "CodeProtocolError", "status identity binding", 2),
			entry: "CheckStatusResult",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckStatusResult(false, terminalbackend.StatusResult{State: "active"})
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")
			},
		},
		{
			key:   wkey("conformance.go", "CheckStatusResult", "CodeProtocolError", "status provider observation", 1),
			entry: "CheckStatusResult",
			prove: func(t *testing.T) {
				// Provider observation carried without request: the
				// identity checks pass, the request-and-evidence
				// conjunction fires.
				present := true
				lastOp := "0198f4c8-8e50-7f66-8f70-1234567890ab"
				lastEffect := terminalbackend.EffectWrapperStarted
				result := terminalbackend.StatusResult{
					State: trueState(), IdentityMatch: true, WrapperPresent: true,
					ProviderPresent: &present, Attachable: false,
					LastOperationID: &lastOp, LastEffect: &lastEffect,
					ProviderRequested: false, ProviderEvidenced: true, AttachEvidenced: false,
				}
				err := terminalbackend.CheckStatusResult(true, result)
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "status provider observation")
			},
		},
		{
			key:   wkey("conformance.go", "CheckTransition", "CodePreconditionFailed", "lifecycle instance scope", 1),
			entry: "CheckTransition",
			prove: func(t *testing.T) {
				// manifest carries no instance: the operation parse
				// passes, the scope comparison fires.
				_, _, err := terminalbackend.CheckTransition("manifest", "active", false)
				requireConformanceRefusal(t, err, "local_precondition_failed", "lifecycle instance scope")
			},
		},
		{
			key:   wkey("conformance.go", "CheckTransition", "CodePreconditionFailed", "lifecycle transition", 1),
			entry: "CheckTransition",
			prove: func(t *testing.T) {
				// attach from a known but disallowed source: operation,
				// scope, and state parses pass, the source-membership
				// comparison fires.
				_, _, err := terminalbackend.CheckTransition("attach", "stopped", false)
				requireConformanceRefusal(t, err, "local_precondition_failed", "lifecycle transition")
			},
		},
		{
			// Occurrence order in IdempotencyKey: #1 is the empty-segment
			// loop, #2 the variable-count floor (status), #3 the exact
			// count. Each prove passes the earlier comparisons.
			key:   wkey("conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 1),
			entry: "IdempotencyKey",
			prove: func(t *testing.T) {
				_, err := terminalbackend.IdempotencyKey("create", "", "bootstrap-1")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency key shape")
			},
		},
		{
			key:   wkey("conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 2),
			entry: "IdempotencyKey",
			prove: func(t *testing.T) {
				_, err := terminalbackend.IdempotencyKey("status")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency key shape")
			},
		},
		{
			key:   wkey("conformance.go", "IdempotencyKey", "CodeProtocolError", "idempotency key shape", 3),
			entry: "IdempotencyKey",
			prove: func(t *testing.T) {
				_, err := terminalbackend.IdempotencyKey("manifest")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency key shape")
			},
		},
		{
			// Occurrence order in ImportLedger: #1 is the line-count
			// shape, #2 the empty key/result shape, #3 the operation
			// parse. Each prove passes the earlier shapes.
			key:   wkey("conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 1),
			entry: "ImportLedger",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ImportLedger([]byte("only-one-line\n"))
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency ledger image")
			},
		},
		{
			key:   wkey("conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 2),
			entry: "ImportLedger",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ImportLedger([]byte("key\ncreate\n\n"))
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency ledger image")
			},
		},
		{
			key:   wkey("conformance.go", "ImportLedger", "CodeProtocolError", "idempotency ledger image", 3),
			entry: "ImportLedger",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ImportLedger([]byte("key\nlaunch\nresult-1\n"))
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency ledger image")
			},
		},
		{
			key:   wkey("conformance.go", "ImportLedger", "CodeIdempotencyMismatch", "idempotency ledger image", 1),
			entry: "ImportLedger",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ImportLedger([]byte("key\ncreate\nresult-1\nkey\nrestore\nresult-2\n"))
				requireConformanceRefusal(t, err, "idempotency_mismatch", "idempotency ledger image")
			},
		},
		{
			key:   wkey("conformance.go", "Ledger.Bind", "CodeIdempotencyMismatch", "idempotency key conflict", 1),
			entry: "Bind",
			prove: func(t *testing.T) {
				ledger := terminalbackend.NewLedger()
				if _, err := ledger.Bind("session-1/bootstrap-1", "create", "result-1"); err != nil {
					t.Fatalf("Bind() error = %v", err)
				}
				_, err := ledger.Bind("session-1/bootstrap-1", "restore", "result-1")
				requireConformanceRefusal(t, err, "idempotency_mismatch", "idempotency key conflict")
			},
		},
		{
			key:   wkey("conformance.go", "Ledger.Bind", "CodeProtocolError", "idempotency key shape", 1),
			entry: "Bind",
			prove: func(t *testing.T) {
				_, err := terminalbackend.NewLedger().Bind("", "create", "result-1")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency key shape")
			},
		},
		{
			key:   wkey("conformance.go", "Ledger.Bind", "CodeProtocolError", "idempotency ledger unavailable", 1),
			entry: "Bind",
			prove: func(t *testing.T) {
				var nilLedger *terminalbackend.Ledger
				_, err := nilLedger.Bind("session-1/bootstrap-1", "create", "result-1")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "idempotency ledger unavailable")
			},
		},
		{
			// Occurrence order in ParseAttachAuthorization: #1 is the
			// host-ID grammar check, #2 the input-authorized type check.
			// #2's input carries a valid host ID, passing #1.
			key:   wkey("conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document member type", 1),
			entry: "ParseAttachAuthorization",
			prove: func(t *testing.T) {
				raw := `{"policy_evidence_id":"` + testDigest + `",` +
					`"authorizing_host_id":"not-a-uuid",` +
					`"transport":"local_only",` +
					`"input_authorized":true,` +
					`"issued_at":"` + conformanceIssued + `",` +
					`"expires_at":"` + conformanceExpires + `"}`
				_, err := terminalbackend.ParseAttachAuthorization([]byte(raw))
				requireConformanceRefusal(t, err, "terminal_backend_manifest_probe_mismatch", "document member type")
			},
		},
		{
			key:   wkey("conformance.go", "ParseAttachAuthorization", "CodeMismatch", "document member type", 2),
			entry: "ParseAttachAuthorization",
			prove: func(t *testing.T) {
				raw := `{"policy_evidence_id":"` + testDigest + `",` +
					`"authorizing_host_id":"` + conformanceHost + `",` +
					`"transport":"local_only",` +
					`"input_authorized":"yes",` +
					`"issued_at":"` + conformanceIssued + `",` +
					`"expires_at":"` + conformanceExpires + `"}`
				_, err := terminalbackend.ParseAttachAuthorization([]byte(raw))
				requireConformanceRefusal(t, err, "terminal_backend_manifest_probe_mismatch", "document member type")
			},
		},
		{
			key:   wkey("conformance.go", "ParseAttachAuthorization", "CodeUnauthorized", "attach authorization expiry", 1),
			entry: "ParseAttachAuthorization",
			prove: func(t *testing.T) {
				// Expiry equal to issue: every member rule passes, the
				// strict-after comparison fires.
				raw := `{"policy_evidence_id":"` + testDigest + `",` +
					`"authorizing_host_id":"` + conformanceHost + `",` +
					`"transport":"local_only",` +
					`"input_authorized":true,` +
					`"issued_at":"` + conformanceIssued + `",` +
					`"expires_at":"` + conformanceIssued + `"}`
				_, err := terminalbackend.ParseAttachAuthorization([]byte(raw))
				requireConformanceRefusal(t, err, "terminal_backend_unauthorized", "attach authorization expiry")
			},
		},
		{
			key:   wkey("conformance.go", "ParseInstanceState", "CodeProtocolError", "lifecycle state vocabulary", 1),
			entry: "ParseInstanceState",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseInstanceState("running")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "lifecycle state vocabulary")
			},
		},
		{
			key:   wkey("conformance.go", "ParseOperation", "CodeProtocolError", "operation vocabulary", 1),
			entry: "ParseOperation",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseOperation("launch")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "operation vocabulary")
			},
		},
		{
			key:   wkey("conformance.go", "ParseSideEffect", "CodeProtocolError", "side effect vocabulary", 1),
			entry: "ParseSideEffect",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseSideEffect("side_effect_unknown")
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "side effect vocabulary")
			},
		},
		{
			key:   wkey("conformance.go", "ProjectToLegacy", "CodeIncompatibleSchema", "legacy reverse projection", 1),
			entry: "ProjectToLegacy",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ProjectToLegacy("vendor.term")
				requireConformanceRefusal(t, err, "incompatible_schema", "legacy reverse projection")
			},
		},
		{
			key:   wkey("conformance.go", "TranslateLegacyBackend", "CodeIncompatibleSchema", "legacy backend identity", 1),
			entry: "TranslateLegacyBackend",
			prove: func(t *testing.T) {
				_, err := terminalbackend.TranslateLegacyBackend("screen")
				requireConformanceRefusal(t, err, "incompatible_schema", "legacy backend identity")
			},
		},
		{
			key:   wkey("conformance.go", "parseTransport", "CodeProtocolError", "presentation transport vocabulary", 1),
			entry: "ParseAttachAuthorization",
			prove: func(t *testing.T) {
				// Unknown transport: member rules pass, the closed
				// transport vocabulary fires inside ParseAttachAuthorization.
				raw := `{"policy_evidence_id":"` + testDigest + `",` +
					`"authorizing_host_id":"` + conformanceHost + `",` +
					`"transport":"courier_pigeon",` +
					`"input_authorized":true,` +
					`"issued_at":"` + conformanceIssued + `",` +
					`"expires_at":"` + conformanceExpires + `"}`
				_, err := terminalbackend.ParseAttachAuthorization([]byte(raw))
				requireConformanceRefusal(t, err, "terminal_backend_protocol_error", "presentation transport vocabulary")
			},
		},
	}
}

// trueState returns a non-parked, non-active admitted state for the
// attachability witness: stopped parses, but attachable-stopped is refused.
func trueState() terminalbackend.InstanceState {
	return terminalbackend.InstanceState("stopped")
}

// declaredArmWitnesses aggregates every witness group. Groups live in
// this file by entry family so each prove sits beside its construction
// argument.
func declaredArmWitnesses() []armWitness {
	var witnesses []armWitness
	for _, group := range [][]armWitness{
		conformanceWitnesses(),
		registryWitnesses(),
		manifestWitnesses(),
		probeWitnesses(),
		evidenceWitnesses(),
		reconcileWitnesses(),
		admitWitnesses(),
		descriptorWitnesses(),
	} {
		witnesses = append(witnesses, group...)
	}
	return witnesses
}

// proveManifestDoc mutates a valid manifest map, re-stamps its identity,
// and requires ParseManifest to refuse with wantDetail. The mutation must
// keep every earlier-checked member valid so the refusal attributes to
// the named site.
func proveManifestDoc(t *testing.T, mutate func(map[string]any), wantDetail string) {
	t.Helper()
	universe := testUniverse(t)
	object := cloneMap(t, universe.manifest)
	mutate(object)
	finalizeDocument(t, object, "manifest_id")
	_, err := terminalbackend.ParseManifest(mustMarshal(t, object))
	requireRefusal(t, err, codeMismatch, wantDetail)
}

// proveProbeDoc mutates a valid probe map, re-stamps its identity, and
// requires ParseProbe to refuse with wantDetail.
func proveProbeDoc(t *testing.T, mutate func(map[string]any), wantDetail string) {
	t.Helper()
	universe := testUniverse(t)
	object := cloneMap(t, universe.probe)
	mutate(object)
	finalizeDocument(t, object, "probe_id")
	_, err := terminalbackend.ParseProbe(mustMarshal(t, object))
	requireRefusal(t, err, codeMismatch, wantDetail)
}

// proveEvidenceDoc mutates a valid evidence map, re-stamps only its
// identity (parsing never verifies the attestation, so the original
// signature stays a valid bystander), and requires ParseEvidence to
// refuse with wantDetail.
func proveEvidenceDoc(t *testing.T, mutate func(map[string]any), wantDetail string) {
	t.Helper()
	universe := testUniverse(t)
	object := cloneMap(t, universe.evidenceByCap["durable_disconnect"])
	mutate(object)
	object["evidence_id"] = testIdentity(t, object, "evidence_id")
	_, err := terminalbackend.ParseEvidence(mustMarshal(t, object))
	requireRefusal(t, err, codeMismatch, wantDetail)
}

// probeWitnesses proves every non-bound ParseProbe arm. The shared
// member-set, digest, identity, timestamp, ordering, string-bound, and
// extension sites carry no per-entry rows: one witness per site lives in
// manifestWitnesses, and the probe entry reaches the same static sites.
func probeWitnesses() []armWitness {
	return []armWitness{
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "document digest", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["evidence_ids"] = []any{"nope"}
				}, "document digest")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "evidence list bound", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					ids := make([]any, 0, 257)
					for seed := 0; seed < 257; seed++ {
						ids = append(ids, testSeedDigest(byte(seed)))
					}
					object["evidence_ids"] = ids
				}, "evidence list bound")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe backend identity", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["terminal_backend_id"] = "ax.evil"
				}, "probe backend identity")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe platform", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["platform"] = "plan9"
				}, "probe platform")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe protocol major 1", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["protocol_version"] = "2.0.0"
				}, "probe protocol major 1")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe schema", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["schema"] = "urn:ax:schema:terminal-backend-manifest"
				}, "probe schema")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe schema version", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["schema_version"] = "2.0.0"
				}, "probe schema version")
			},
		},
		{
			key:   wkey("manifest.go", "boundedStringMember", "CodeMismatch", "document string bound", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["os_version"] = strings.Repeat("o", 257)
				}, "document string bound")
			},
		},
		{
			// stringArrayMember #1 is the not-an-array shape,
			// #2 the non-string-element shape: #2's input is an
			// array, passing #1.
			key:   wkey("manifest.go", "stringArrayMember", "CodeMismatch", "document member type", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["dependent_operations"] = "create"
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "stringArrayMember", "CodeMismatch", "document member type", 2),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["facts"] = []any{"fixture_passed", float64(7)}
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "stringMember", "CodeMismatch", "document member type", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				// A number where the first string member belongs: the
				// member set passes, the string assertion fires before
				// any value rule reads it.
				proveManifestDoc(t, func(object map[string]any) {
					object["implementation_version"] = float64(2)
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe availability", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["availability"] = "sometimes"
				}, "probe availability")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe executable digest", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				// Local-program kind without a digest: the member set,
				// kind, and type shapes pass, the needs-digest rule
				// fires.
				proveProbeDoc(t, func(object map[string]any) {
					object["implementation_kind"] = "local_program"
					object["executable_digest"] = nil
				}, "probe executable digest")
			},
		},
		{
			key:   wkey("manifest.go", "ParseProbe", "CodeMismatch", "probe implementation kind", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["implementation_kind"] = "container_image"
				}, "probe implementation kind")
			},
		},
		{
			key:   wkey("manifest.go", "semverMember", "CodeMismatch", "document semver", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["implementation_version"] = "2.1"
				}, "document semver")
			},
		},
		{
			key:   wkey("manifest.go", "timestampMember", "CodeMismatch", "document timestamp", 1),
			entry: "ParseProbe",
			prove: func(t *testing.T) {
				proveProbeDoc(t, func(object map[string]any) {
					object["probed_at"] = "not-a-timestamp"
				}, "document timestamp")
			},
		},
		{
			key:   wkey("manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions bound", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["protocol_versions"] = []any{}
				}, "protocol versions bound")
			},
		},
		{
			key:   wkey("manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions major 1", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["protocol_versions"] = []any{"2.0.0"}
				}, "protocol versions major 1")
			},
		},
		{
			key:   wkey("manifest.go", "validateProtocolList", "CodeMismatch", "protocol versions ordering", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["protocol_versions"] = []any{"1.0.0", "1.0.0"}
				}, "protocol versions ordering")
			},
		},
	}
}

// evidenceWitnesses proves every non-bound ParseEvidence arm. Mutations
// keep the attestation-adjacent members valid so value rules fire before
// identity, matching parseEvidenceObject order.
func evidenceWitnesses() []armWitness {
	return []armWitness{
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "capability vocabulary", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "teleportation"
				}, "capability vocabulary")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence backend identity", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["terminal_backend_id"] = "ax.evil"
				}, "evidence backend identity")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence expiry", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["expires_at"] = "2024-01-01T00:00:00.000Z"
				}, "evidence expiry")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence issuer", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["issuer"] = "self"
				}, "evidence issuer")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence platform", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["platform"] = "plan9"
				}, "evidence platform")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence protocol major 1", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["protocol_version"] = "2.0.0"
				}, "evidence protocol major 1")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence schema", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["schema"] = "urn:ax:schema:terminal-backend-manifest"
				}, "evidence schema")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence schema version", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["schema_version"] = "2.0.0"
				}, "evidence schema version")
			},
		},
		{
			key:   wkey("manifest.go", "ParseEvidence", "CodeMismatch", "evidence value", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["value"] = "maybe"
				}, "evidence value")
			},
		},
		{
			key:   wkey("manifest.go", "parseAttestationSignature", "CodeMismatch", "evidence signature encoding", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["attestation_signature"] = "rsa-sha256:!!!"
				}, "evidence signature encoding")
			},
		},
		{
			key:   wkey("manifest.go", "parseAttestationSignature", "CodeMismatch", "evidence signature scheme", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["attestation_signature"] = "sha256:abcd"
				}, "evidence signature scheme")
			},
		},
	}
}

// descriptorWitnesses proves every non-bound ParseProviderDescriptor arm
// through setDescriptorMember mutations on the valid descriptor.
func descriptorWitnesses() []armWitness {
	return []armWitness{
		{
			// Occurrence order: #1 is the count shape (extra member),
			// #2 the missing-member shape.
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor member set", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				extra := strings.Replace(validDescriptorDoc, `"rows": 24`, `"rows": 24, "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, 1)
				_, err := terminalbackend.ParseProviderDescriptor([]byte(extra))
				requireDescriptorRefusal(t, err, "descriptor member set")
			},
		},
		{
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor member set", 2),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				// Same member count with "rows" renamed away: the
				// count shape (#1) passes, the per-member loop fires.
				// Deleting the member instead shortens the document
				// and refuses at #1, proving only the sibling.
				renamed := strings.Replace(validDescriptorDoc, `"rows": 24`, `"rowsx": 24`, 1)
				_, err := terminalbackend.ParseProviderDescriptor([]byte(renamed))
				requireDescriptorRefusal(t, err, "descriptor member set")
			},
		},
		{
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor digest", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "terminal_binding_id", `"not-a-digest"`))
				requireDescriptorRefusal(t, err, "descriptor digest")
			},
		},
		{
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor instance", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "terminal_instance_id", `"1234"`))
				requireDescriptorRefusal(t, err, "descriptor instance")
			},
		},
		{
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor implementation version", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "implementation_version", `"1.2"`))
				requireDescriptorRefusal(t, err, "descriptor implementation version")
			},
		},
		{
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor protocol version", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "protocol_version", `"one"`))
				requireDescriptorRefusal(t, err, "descriptor protocol version")
			},
		},
		{
			key:   wkey("descriptor.go", "ParseProviderDescriptor", "CodeProtocolError", "descriptor interactive", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "interactive", `1`))
				requireDescriptorRefusal(t, err, "descriptor interactive")
			},
		},
		{
			key:   wkey("descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry bound", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				// The 37-digit witness pins the pre-multiply guard at
				// its own arithmetic edge: it refuses here, before any
				// accumulator exists.
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "columns", `9223372036854775808000000000000000001`))
				requireDescriptorRefusal(t, err, "descriptor geometry bound")
			},
		},
		{
			key:   wkey("descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry bound", 2),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "columns", `0`))
				requireDescriptorRefusal(t, err, "descriptor geometry bound")
			},
		},
		{
			key:   wkey("descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry digits", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "columns", `80.0`))
				requireDescriptorRefusal(t, err, "descriptor geometry digits")
			},
		},
		{
			key:   wkey("descriptor.go", "descriptorGeometry", "CodeProtocolError", "descriptor geometry type", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "columns", `"80"`))
				requireDescriptorRefusal(t, err, "descriptor geometry type")
			},
		},
		{
			key:   wkey("descriptor.go", "descriptorText", "CodeProtocolError", "descriptor member type", 1),
			entry: "ParseProviderDescriptor",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "terminal_binding_id", `null`))
				requireDescriptorRefusal(t, err, "descriptor member type")
			},
		},
		{
			key:   wkey("manifest.go", "GenerationDigest", "CodeStaleGeneration", "backend_generation bound", 1),
			entry: "GenerationDigest",
			prove: func(t *testing.T) {
				// Empty generation image: the UTF-8/length rule fires
				// before any digest exists.
				_, err := terminalbackend.GenerationDigest("")
				requireRefusal(t, err, codeStaleGeneration, "backend_generation bound")
			},
		},
		{
			key:   wkey("manifest.go", "CapabilitiesForOperation", "CodeMismatch", "operation vocabulary", 1),
			entry: "CapabilitiesForOperation",
			prove: func(t *testing.T) {
				_, err := terminalbackend.CapabilitiesForOperation("launch")
				requireRefusal(t, err, codeMismatch, "operation vocabulary")
			},
		},
		{
			key:   wkey("manifest.go", "CheckOperation", "CodeMismatch", "operation vocabulary", 1),
			entry: "CheckOperation",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
				if err != nil {
					t.Fatalf("Reconcile() error = %v", err)
				}
				err = terminalbackend.CheckOperation("launch", admitted)
				requireRefusal(t, err, codeMismatch, "operation vocabulary")
			},
		},
		{
			key:   wkey("manifest.go", "CheckOperation", "CodeCapabilityUnproven", "operation capability dependency", 1),
			entry: "CheckOperation",
			prove: func(t *testing.T) {
				// attach is a closed operation no proved claim
				// supports: the vocabulary parse passes, the
				// dependency check fires.
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
				if err != nil {
					t.Fatalf("Reconcile() error = %v", err)
				}
				err = terminalbackend.CheckOperation("attach", admitted)
				requireRefusal(t, err, codeCapabilityUnproven, "operation capability dependency")
			},
		},
	}
}

// proveReconcileDoc mutates the universe maps, then drives the public
// Reconcile entry with the freshly parsed documents, requiring the named
// refusal. Parsing must succeed, so mutations stay schema-valid and only
// the reconciled RELATION breaks.
func proveReconcileDoc(t *testing.T, mutate func(*fixtureUniverse), wantCode, wantDetail string) {
	t.Helper()
	universe := testUniverse(t)
	mutate(universe)
	manifest, probe, evidence := reconcileFixture(t, universe)
	_, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
	requireRefusal(t, err, wantCode, wantDetail)
}

// reconcileWitnesses proves every non-bound Reconcile arm at the public
// Reconcile entry. Document-level mutations mirror the refusal table;
// typed post-parse mutations (registry drift, conflicting evidence)
// reuse the white-box pins' construction through the same public entry.
func reconcileWitnesses() []armWitness {
	return []armWitness{
		{
			key:   wkey("manifest.go", "checkClaimRelation", "CodeMismatch", "probe omission of manifest claim", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					probe := universe.probe["capability_claims"].([]any)
					universe.probe["capability_claims"] = probe[1:]
					delete(universe.evidenceByCap, "durable_disconnect")
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "probe omission of manifest claim")
			},
		},
		{
			key:   wkey("manifest.go", "checkClaimRelation", "CodeMismatch", "probe static claim echo", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					probe := universe.probe["capability_claims"].([]any)
					probe[0].(map[string]any)["value"] = false
					delete(universe.evidenceByCap, "durable_disconnect")
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "probe static claim echo")
			},
		},
		{
			key:   wkey("manifest.go", "checkClaimRelation", "CodeMismatch", "probe static claim without manifest", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					probe := universe.probe["capability_claims"].([]any)
					probe[1].(map[string]any)["origin"] = "static"
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "probe static claim without manifest")
			},
		},
		{
			key:   wkey("manifest.go", "checkClaimRelation", "CodeMismatch", "probe override of stable claim", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					probe := universe.probe["capability_claims"].([]any)
					probe[0].(map[string]any)["origin"] = "probed"
					finalizeDocument(t, universe.probe, "probe_id")
				}, codeMismatch, "probe override of stable claim")
			},
		},
		{
			// Typed post-parse mutation (documents cannot carry a
			// registry-drifted row: parseClaim refuses it first): the
			// probe parses, then the typed row drifts, then the public
			// Reconcile entry refuses at the relation check.
			key:   wkey("manifest.go", "checkClaimRelation", "CodeMismatch", "probe override registry binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				for index := range probe.CapabilityClaims {
					if probe.CapabilityClaims[index].Capability == "headless_creation" {
						probe.CapabilityClaims[index].DependentOperations = []string{"create", "status"}
					}
				}
				_, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
				requireRefusal(t, err, codeMismatch, "probe override registry binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkClaimRelation", "CodeMismatch", "probe addition registry binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				for index := range probe.CapabilityClaims {
					if probe.CapabilityClaims[index].Capability == "graceful_stop" {
						probe.CapabilityClaims[index].EvidenceRequirements = []string{"conformance_fixture"}
					}
				}
				_, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
				requireRefusal(t, err, codeMismatch, "probe addition registry binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceCoverage", "CodeMismatch", "evidence requirement coverage", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					universe.evidence = nil
					delete(universe.evidenceByCap, "durable_disconnect")
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "evidence requirement coverage")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceIDs", "CodeMismatch", "evidence id set binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					// The probe still demands graceful_stop but its
					// evidence object is gone: the ID set no longer
					// covers the probe's set.
					claims := universe.probe["capability_claims"].([]any)
					claims[1].(map[string]any)["value"] = false
					delete(universe.evidenceByCap, "graceful_stop")
					finalizeDocument(t, universe.probe, "probe_id")
				}, codeMismatch, "evidence id set binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceLiveness", "CodeMismatch", "evidence liveness", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					object := universe.evidenceByCap["graceful_stop"]
					object["observed_at"] = "2024-01-01T00:00:00.000Z"
					object["expires_at"] = "2025-01-01T00:00:00.000Z"
					finalizeEvidence(t, universe.key, object)
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "evidence liveness")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceSet", "CodeMismatch", "conflicting evidence", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				// Same-ID twin with an extra fact: the conflict rule
				// precedes the attestation check, so the reused
				// signature is a valid bystander.
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				twin := evidence[0]
				twin.Facts = append(append([]string{}, evidence[0].Facts...), "fixture_extra_fact")
				evidence = append(evidence, twin)
				_, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, testVerifier(universe.key))
				requireRefusal(t, err, codeMismatch, "conflicting evidence")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceSet", "CodeMismatch", "evidence claim binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					object := universe.evidenceMap("local_attach", []any{"fixture_passed", "policy_checked", "runtime_probe_passed"})
					finalizeEvidence(t, universe.key, object)
					universe.evidenceByCap["local_attach"] = object
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "evidence claim binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceSignature", "CodeIntegrityFailure", "evidence attestation", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				// Every evidence object re-signed by a foreign key:
				// parses admit them (well-formed), the Reconcile
				// attestation check refuses.
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					other := testKey(t)
					for _, object := range universe.evidenceByCap {
						delete(object, "attestation_signature")
						object["attestation_signature"] = ""
						testSignBody(t, other, object)
						object["evidence_id"] = testIdentity(t, object, "evidence_id")
					}
					finalizeProbeIDs(t, universe)
				}, codeIntegrityFailure, "evidence attestation")
			},
		},
		{
			key:   wkey("manifest.go", "checkEvidenceTuple", "CodeMismatch", "evidence tuple binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					object := universe.evidenceByCap["graceful_stop"]
					object["backend_generation_digest"] = testSeedDigest(0x99)
					finalizeEvidence(t, universe.key, object)
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "evidence tuple binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkProbeIdentity", "CodeMismatch", "probe manifest binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					universe.probe["implementation_version"] = "1.2.4"
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "probe manifest binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkProbeMembership", "CodeMismatch", "probe protocol membership", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					universe.probe["protocol_version"] = "1.2.0"
					for _, object := range universe.evidenceByCap {
						object["protocol_version"] = "1.2.0"
						finalizeEvidence(t, universe.key, object)
					}
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "probe protocol membership")
			},
		},
		{
			key:   wkey("manifest.go", "checkProbeMembership", "CodeMismatch", "probe platform membership", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				proveReconcileDoc(t, func(universe *fixtureUniverse) {
					universe.probe["platform"] = "windows"
					for _, object := range universe.evidenceByCap {
						object["platform"] = "windows"
						finalizeEvidence(t, universe.key, object)
					}
					finalizeProbeIDs(t, universe)
				}, codeMismatch, "probe platform membership")
			},
		},
		{
			key:   wkey("manifest.go", "checkProbeGeneration", "CodeStaleGeneration", "probe generation binding", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				// A well-formed but foreign raw generation image: the
				// membership parses pass, the generation comparison
				// fires.
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				_, err := terminalbackend.Reconcile(manifest, probe, evidence, "generation-beta", universe.now, testVerifier(universe.key))
				requireRefusal(t, err, codeStaleGeneration, "probe generation binding")
			},
		},
		{
			key:   wkey("manifest.go", "Reconcile", "CodeIntegrityFailure", "evidence signature verifier", 1),
			entry: "Reconcile",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				manifest, probe, evidence := reconcileFixture(t, universe)
				_, err := terminalbackend.Reconcile(manifest, probe, evidence, universe.rawGeneration, universe.now, nil)
				requireRefusal(t, err, codeIntegrityFailure, "evidence signature verifier")
			},
		},
	}
}

// admitSubstituted drives AdmitProbe with the manifest and probe
// executable digests overridden: when both track the foreign digest,
// the probe-to-manifest identity holds and only the manifest-to-record
// binding can refuse; when only the probe drifts, the probe identity
// refuses.
//
// The documents rebind to example.realm (the ax.* namespace refuses
// external trust outright): every record-compared member matches the
// admitted record exactly, so only the digests move.
func admitSubstituted(t *testing.T, manifestDigest, probeDigest string, wantCode, wantDetail string) {
	t.Helper()
	universe := testUniverse(t)
	registry, err := terminalbackend.New("2.1.0", []string{"1.0.0", "1.1.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	const backendID = "example.realm"
	if err := registerExternalForTest(t, registry, backendID, testSeedDigest(0xA1)); err != nil {
		t.Fatalf("RegisterExternal() error = %v", err)
	}
	manifest := cloneMap(t, universe.manifest)
	manifest["terminal_backend_id"] = backendID
	manifest["implementation_version"] = "3.0.0"
	manifest["protocol_versions"] = []any{"1.0.0"}
	manifest["platforms"] = []any{"linux"}
	manifest["implementation_kind"] = "local_program"
	manifest["executable_digest"] = manifestDigest
	finalizeDocument(t, manifest, "manifest_id")
	probe := cloneMap(t, universe.probe)
	probe["terminal_backend_id"] = backendID
	probe["implementation_version"] = "3.0.0"
	probe["protocol_version"] = "1.0.0"
	probe["implementation_kind"] = "local_program"
	probe["executable_digest"] = probeDigest
	finalizeDocument(t, probe, "probe_id")
	var evidenceRaws [][]byte
	for _, capability := range []string{"durable_disconnect", "graceful_stop", "headless_creation"} {
		object := cloneMap(t, universe.evidenceByCap[capability])
		object["terminal_backend_id"] = backendID
		object["implementation_version"] = "3.0.0"
		object["protocol_version"] = "1.0.0"
		finalizeEvidence(t, universe.key, object)
		evidenceRaws = append(evidenceRaws, mustMarshal(t, object))
	}
	// The probe's evidence set must list the rebound evidence identities,
	// or the ID-set rule fires before either substitution arm.
	var rebound []string
	for _, raw := range evidenceRaws {
		parsed, err := terminalbackend.ParseEvidence(raw)
		if err != nil {
			t.Fatalf("ParseEvidence() error = %v", err)
		}
		rebound = append(rebound, parsed.EvidenceID)
	}
	sort.Strings(rebound)
	asValues := make([]any, 0, len(rebound))
	for _, id := range rebound {
		asValues = append(asValues, id)
	}
	probe["evidence_ids"] = asValues
	finalizeDocument(t, probe, "probe_id")
	_, err = registry.AdmitProbe(
		mustMarshal(t, manifest), mustMarshal(t, probe),
		evidenceRaws, universe.rawGeneration, universe.now, testVerifier(universe.key),
	)
	requireRefusal(t, err, wantCode, wantDetail)
}

// admitWitnesses proves every non-bound AdmitProbe arm: the nil-registry
// arm needs no documents, the drift arm reuses the record-drift
// construction, and the two substitution arms share admitSubstituted.
func admitWitnesses() []armWitness {
	return []armWitness{
		{
			key:   wkey("manifest.go", "Registry.AdmitProbe", "CodeNotFound", "registry unavailable", 1),
			entry: "AdmitProbe",
			prove: func(t *testing.T) {
				var nilRegistry *terminalbackend.Registry
				_, err := nilRegistry.AdmitProbe(nil, nil, nil, "", time.Now().UTC(), nil)
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "registry unavailable")
			},
		},
		{
			key:   wkey("manifest.go", "checkManifestRecordBinding", "CodeDrift", "manifest implementation drift", 1),
			entry: "AdmitProbe",
			prove: func(t *testing.T) {
				// Drifted implementation version: the manifest parses
				// and the identity resolves, so the record comparison
				// fires.
				universe := testUniverse(t)
				universe.manifest["implementation_version"] = "9.9.9"
				finalizeDocument(t, universe.manifest, "manifest_id")
				registry, err := terminalbackend.New("2.1.0", []string{"1.0.0", "1.1.0"})
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				_, err = registry.AdmitProbe(
					mustMarshal(t, universe.manifest), mustMarshal(t, universe.probe),
					nil, universe.rawGeneration, universe.now, testVerifier(universe.key),
				)
				requireRefusal(t, err, codeDrift, "manifest implementation drift")
			},
		},
		{
			key:   wkey("manifest.go", "checkManifestRecordBinding", "CodeUntrusted", "executable substitution", 1),
			entry: "AdmitProbe",
			prove: func(t *testing.T) {
				admitSubstituted(t, testSeedDigest(0xE6), testSeedDigest(0xE6), codeUntrusted, "executable substitution")
			},
		},
		{
			key:   wkey("manifest.go", "checkProbeIdentity", "CodeUntrusted", "executable substitution", 1),
			entry: "AdmitProbe",
			prove: func(t *testing.T) {
				admitSubstituted(t, testSeedDigest(0xA1), testSeedDigest(0xE6), codeUntrusted, "executable substitution")
			},
		},
	}
}

// manifestWitnesses proves every non-bound ParseManifest arm. Mutations
// follow parseManifestObject order so each input passes the earlier
// same-clause sites: member-set before member rules, member rules before
// value rules, value rules before identity.
func manifestWitnesses() []armWitness {
	return []armWitness{
		{
			// checkExactMembers #1 is the count shape (extra member);
			// #2 is the same-count unknown-name shape below.
			key:   wkey("manifest.go", "checkExactMembers", "CodeMismatch", "document members", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["sponsor"] = "evil"
				}, "document members")
			},
		},
		{
			key:   wkey("manifest.go", "checkExactMembers", "CodeMismatch", "document members", 2),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					delete(object, "platforms")
					object["sponsor"] = "evil"
				}, "document members")
			},
		},
		{
			key:   wkey("manifest.go", "checkClosedList", "CodeMismatch", "document list bound", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["dependent_operations"] = []any{}
				}, "document list bound")
			},
		},
		{
			key:   wkey("manifest.go", "checkClosedList", "CodeMismatch", "document vocabulary", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["dependent_operations"] = []any{"create", "launch"}
				}, "document vocabulary")
			},
		},
		{
			key:   wkey("manifest.go", "checkExtensions", "CodeMismatch", "document extensions", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["extensions"] = map[string]any{"v2": true}
				}, "document extensions")
			},
		},
		{
			key:   wkey("manifest.go", "checkIdentity", "CodeMismatch", "document digest", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				// Malformed fixture digest: member rules pass, the
				// claimed-ID parse inside checkIdentity fires.
				proveManifestDoc(t, func(object map[string]any) {
					object["conformance_fixture_id"] = "sha256:zzz"
				}, "document digest")
			},
		},
		{
			key:   wkey("manifest.go", "checkIdentity", "CodeMismatch", "document identity binding", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				// Foreign but well-formed digest: the claimed-ID parse
				// passes, the recomputation comparison fires. The
				// identity is overwritten without re-stamping, so the
				// document is otherwise valid.
				universe := testUniverse(t)
				object := cloneMap(t, universe.manifest)
				object["manifest_id"] = testSeedDigest(0xDD)
				_, err := terminalbackend.ParseManifest(mustMarshal(t, object))
				requireRefusal(t, err, codeMismatch, "document identity binding")
			},
		},
		{
			key:   wkey("manifest.go", "checkSortedUnique", "CodeMismatch", "document ordering", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["dependent_operations"] = []any{"status", "create"}
				}, "document ordering")
			},
		},
		{
			key:   wkey("manifest.go", "decodeCappedValue", "CodeMismatch", "document duplicate member", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				raw := []byte(`{"schema":"urn:ax:schema:terminal-backend-manifest","schema":"urn:ax:schema:terminal-backend-manifest"}`)
				_, err := terminalbackend.ParseManifest(raw)
				requireRefusal(t, err, codeMismatch, "document duplicate member")
			},
		},
		{
			key:   wkey("manifest.go", "decodeCappedValue", "CodeMismatch", "document nesting", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				deep := strings.Repeat("[", 40) + strings.Repeat("]", 40)
				_, err := terminalbackend.ParseManifest([]byte(`{"schema":"x","nest":` + deep + `}`))
				requireRefusal(t, err, codeMismatch, "document nesting")
			},
		},
		{
			// Occurrence order: #1 is the top-level token error, #2 the
			// object key-token error, #4 the object close-token error,
			// #5 the array close-token error. #3 and #6 are decoder-
			// contract bound. Each prove fails at the named token read.
			key:   wkey("manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseManifest([]byte{})
				requireRefusal(t, err, codeMismatch, "document syntax")
			},
		},
		{
			key:   wkey("manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 2),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseManifest([]byte(`{1:2}`))
				requireRefusal(t, err, codeMismatch, "document syntax")
			},
		},
		{
			key:   wkey("manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 4),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseManifest([]byte(`{"a":1`))
				requireRefusal(t, err, codeMismatch, "document syntax")
			},
		},
		{
			key:   wkey("manifest.go", "decodeCappedValue", "CodeMismatch", "document syntax", 5),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseManifest([]byte(`[1,2`))
				requireRefusal(t, err, codeMismatch, "document syntax")
			},
		},
		{
			key:   wkey("manifest.go", "decodeStrictObject", "CodeMismatch", "document encoding", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				valid := mustMarshal(t, universe.manifest)
				_, err := terminalbackend.ParseManifest(append(append([]byte{}, valid...), 0xff))
				requireRefusal(t, err, codeMismatch, "document encoding")
			},
		},
		{
			key:   wkey("manifest.go", "decodeStrictObject", "CodeMismatch", "document shape", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseManifest([]byte(`[]`))
				requireRefusal(t, err, codeMismatch, "document shape")
			},
		},
		{
			key:   wkey("manifest.go", "decodeStrictObject", "CodeMismatch", "document size", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				padded := padOwnershipDocument(t, mustMarshal(t, universe.manifest), 5_242_881)
				_, err := terminalbackend.ParseManifest(padded)
				requireRefusal(t, err, codeMismatch, "document size")
			},
		},
		{
			key:   wkey("manifest.go", "decodeStrictObject", "CodeMismatch", "document surrogate escape", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				// The lone-surrogate escape is assembled from byte
				// values: no literal here spells the escape directly.
				// The raw-byte scan fires before any member rule reads
				// the value.
				universe := testUniverse(t)
				valid := string(mustMarshal(t, universe.manifest))
				slash := string([]byte{92})
				old := `"terminal_backend_id":"ax.tmux"`
				if strings.Count(valid, old) != 1 {
					t.Fatalf("surgery anchor is not unique")
				}
				injected := strings.Replace(valid, old, `"terminal_backend_id":"`+slash+`ud800"`, 1)
				_, err := terminalbackend.ParseManifest([]byte(injected))
				requireRefusal(t, err, codeMismatch, "document surrogate escape")
			},
		},
		{
			key:   wkey("manifest.go", "decodeStrictObject", "CodeMismatch", "document trailing data", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				universe := testUniverse(t)
				valid := mustMarshal(t, universe.manifest)
				_, err := terminalbackend.ParseManifest(append(append([]byte{}, valid...), []byte(" {}")...))
				requireRefusal(t, err, codeMismatch, "document trailing data")
			},
		},
		{
			key:   wkey("manifest.go", "digestMember", "CodeMismatch", "document digest", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["conformance_fixture_id"] = "sha256:zzz"
				}, "document digest")
			},
		},
		{
			// digestOrNullMember: the members and member-type shapes are
			// shadowed-lookup bound; the digest value shape below is the
			// live arm, reached through the manifest executable_digest
			// member (string-typed, so the type shape passes).
			key:   wkey("manifest.go", "digestOrNullMember", "CodeMismatch", "document digest", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["implementation_kind"] = "local_program"
					object["executable_digest"] = "not-a-digest"
				}, "document digest")
			},
		},
		{
			key:   wkey("manifest.go", "digestOrNullMember", "CodeMismatch", "document member type", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["executable_digest"] = float64(7)
				}, "document member type")
			},
		},
		{
			// parseClaim #1 is the value-bool shape, #2 the
			// generation-variable-bool shape: #2's input carries a valid
			// value, passing #1.
			key:   wkey("manifest.go", "parseClaim", "CodeMismatch", "document member type", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["value"] = "yes"
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaim", "CodeMismatch", "document member type", 2),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["generation_variable"] = "yes"
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaim", "CodeMismatch", "capability registry binding", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["generation_variable"] = true
				}, "capability registry binding")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaim", "CodeMismatch", "capability vocabulary", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["capability"] = "teleportation"
				}, "capability vocabulary")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaim", "CodeMismatch", "claim origin", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0].(map[string]any)["origin"] = "probed"
				}, "claim origin")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaim", "CodeMismatch", "claim shape", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					claims[0] = "durable_disconnect"
				}, "claim shape")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaimList", "CodeMismatch", "claim list bound", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					bloated := make([]any, 0, 17)
					for len(bloated) < 17 {
						bloated = append(bloated, claims...)
					}
					object["static_capability_claims"] = bloated[:17]
				}, "claim list bound")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaimList", "CodeMismatch", "claim ordering", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				// Adjacent duplicate in sorted position: refused only by
				// the duplicate half of the ordering gate.
				proveManifestDoc(t, func(object map[string]any) {
					claims := object["static_capability_claims"].([]any)
					object["static_capability_claims"] = []any{claims[0], claims[1], claims[1], claims[2]}
				}, "claim ordering")
			},
		},
		{
			key:   wkey("manifest.go", "parseClaimList", "CodeMismatch", "document member type", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["static_capability_claims"] = "durable_disconnect"
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "parseManifestObject", "CodeMismatch", "manifest backend identity", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["terminal_backend_id"] = "ax.evil"
				}, "manifest backend identity")
			},
		},
		{
			key:   wkey("manifest.go", "parseManifestObject", "CodeMismatch", "manifest executable digest", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["implementation_kind"] = "builtin_go"
					object["executable_digest"] = testSeedDigest(0xAA)
				}, "manifest executable digest")
			},
		},
		{
			key:   wkey("manifest.go", "parseManifestObject", "CodeMismatch", "manifest implementation kind", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				// container_image passes parseKind's closed vocabulary
				// (which wraps into this clause) — the clause names the
				// manifest rule, not the registry one.
				proveManifestDoc(t, func(object map[string]any) {
					object["implementation_kind"] = "container_image"
				}, "manifest implementation kind")
			},
		},
		{
			key:   wkey("manifest.go", "parseManifestObject", "CodeMismatch", "manifest schema", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["schema"] = "urn:ax:schema:terminal-backend-probe"
				}, "manifest schema")
			},
		},
		{
			key:   wkey("manifest.go", "parseManifestObject", "CodeMismatch", "manifest schema version", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["schema_version"] = "2.0.0"
				}, "manifest schema version")
			},
		},
		{
			key:   wkey("manifest.go", "parsePlatformList", "CodeMismatch", "platforms bound", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["platforms"] = []any{}
				}, "platforms bound")
			},
		},
		{
			key:   wkey("manifest.go", "parsePlatformList", "CodeMismatch", "platforms ordering", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["platforms"] = []any{"macos", "linux", "wsl2"}
				}, "platforms ordering")
			},
		},
		{
			key:   wkey("manifest.go", "parsePlatformList", "CodeMismatch", "platforms vocabulary", 1),
			entry: "ParseManifest",
			prove: func(t *testing.T) {
				proveManifestDoc(t, func(object map[string]any) {
					object["platforms"] = []any{"amigaos"}
				}, "platforms vocabulary")
			},
		},
		{
			key:   wkey("manifest.go", "parseRealmLiteral", "CodeMismatch", "evidence realm result", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = "codex"
					object["provider_build"] = "1.0"
					object["sentinel_result"] = "failed"
					object["provider_auth_smoke_result"] = "passed"
				}, "evidence realm result")
			},
		},
		{
			// parseRealmMembers #1 is the required-members half (realm
			// claim with null members); #2 the forbidden-members half
			// (plain claim with realm members).
			key:   wkey("manifest.go", "parseRealmMembers", "CodeMismatch", "evidence realm binding", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
				}, "evidence realm binding")
			},
		},
		{
			key:   wkey("manifest.go", "parseRealmMembers", "CodeMismatch", "evidence realm binding", 2),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = "codex"
					object["provider_build"] = "1.0"
					object["sentinel_result"] = "passed"
					object["provider_auth_smoke_result"] = "passed"
				}, "evidence realm binding")
			},
		},
		{
			// parseRealmProvider member-type #1 is the provider_id
			// shape, #2 the provider_build shape: #2's input carries a
			// valid provider ID, passing #1.
			key:   wkey("manifest.go", "parseRealmProvider", "CodeMismatch", "document member type", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = float64(7)
					object["provider_build"] = "1.0"
					object["sentinel_result"] = "passed"
					object["provider_auth_smoke_result"] = "passed"
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "parseRealmProvider", "CodeMismatch", "document member type", 2),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = "codex"
					object["provider_build"] = float64(7)
					object["sentinel_result"] = "passed"
					object["provider_auth_smoke_result"] = "passed"
				}, "document member type")
			},
		},
		{
			key:   wkey("manifest.go", "parseRealmProvider", "CodeMismatch", "document string bound", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = "codex"
					object["provider_build"] = strings.Repeat("o", 257)
					object["sentinel_result"] = "passed"
					object["provider_auth_smoke_result"] = "passed"
				}, "document string bound")
			},
		},
		{
			key:   wkey("manifest.go", "parseRealmProvider", "CodeMismatch", "evidence provider identity", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = "Codex!!"
					object["provider_build"] = "1.0"
					object["sentinel_result"] = "passed"
					object["provider_auth_smoke_result"] = "passed"
				}, "evidence provider identity")
			},
		},
		{
			// The half-pair shape: provider ID without a build. The ID
			// parses and the build member is present-but-null, passing
			// the member and type shapes, so the pair rule fires.
			key:   wkey("manifest.go", "parseRealmProvider", "CodeMismatch", "evidence realm binding", 1),
			entry: "ParseEvidence",
			prove: func(t *testing.T) {
				proveEvidenceDoc(t, func(object map[string]any) {
					object["capability"] = "credential_capable_execution_realm"
					object["terminal_binding_id"] = testSeedDigest(0xB1)
					object["provider_id"] = "codex"
					object["provider_build"] = nil
					object["sentinel_result"] = "passed"
					object["provider_auth_smoke_result"] = "passed"
				}, "evidence realm binding")
			},
		},
	}
}

// registryWitnesses proves every non-bound terminalbackend.go arm at its
// public entry. RegisterExternal checks run in production order (nil
// registry, trust entry, enabled, identity, kind, record validation,
// digest binding, duplicate, drift), so each prove passes the earlier
// gates and fails at the named one.
func registryWitnesses() []armWitness {
	return []armWitness{
		{
			key:   wkey("terminalbackend.go", "CheckProviderDescriptor", "CodeDrift", "descriptor version binding", 1),
			entry: "CheckProviderDescriptor",
			prove: func(t *testing.T) {
				binding := validDescriptorBinding()
				descriptor := binding
				descriptor.ImplementationVersion = "1.2.4"
				err := terminalbackend.CheckProviderDescriptor(descriptor, binding)
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "descriptor version binding")
			},
		},
		{
			key:   wkey("terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor backend binding", 1),
			entry: "CheckProviderDescriptor",
			prove: func(t *testing.T) {
				binding := validDescriptorBinding()
				descriptor := binding
				descriptor.BackendID = "com.example.other"
				err := terminalbackend.CheckProviderDescriptor(descriptor, binding)
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "descriptor backend binding")
			},
		},
		{
			// Occurrence order: #1 is the digest shape (scalar parse),
			// #2 the digest equality. #2's input carries a well-formed
			// foreign digest, passing #1.
			key:   wkey("terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor binding digest", 1),
			entry: "CheckProviderDescriptor",
			prove: func(t *testing.T) {
				binding := validDescriptorBinding()
				descriptor := binding
				descriptor.TerminalBindingID = "not-a-digest"
				err := terminalbackend.CheckProviderDescriptor(descriptor, binding)
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "descriptor binding digest")
			},
		},
		{
			key:   wkey("terminalbackend.go", "CheckProviderDescriptor", "CodeNotFound", "descriptor binding digest", 2),
			entry: "CheckProviderDescriptor",
			prove: func(t *testing.T) {
				binding := validDescriptorBinding()
				descriptor := binding
				descriptor.TerminalBindingID = testDigestOther
				err := terminalbackend.CheckProviderDescriptor(descriptor, binding)
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "descriptor binding digest")
			},
		},
		{
			key:   wkey("terminalbackend.go", "CheckProviderDescriptor", "CodeStaleGeneration", "descriptor generation binding", 1),
			entry: "CheckProviderDescriptor",
			prove: func(t *testing.T) {
				binding := validDescriptorBinding()
				descriptor := binding
				descriptor.Generation = "generation-2"
				err := terminalbackend.CheckProviderDescriptor(descriptor, binding)
				requireRegistryRefusal(t, err, terminalbackend.IsStaleGeneration, "descriptor generation binding")
			},
		},
		{
			key:   wkey("terminalbackend.go", "CheckVersionTuple", "CodeDrift", "implementation_version semver", 1),
			entry: "CheckVersionTuple",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckVersionTuple("com.example.term", "v1", "1.0.0", []string{"1.0.0"})
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "implementation_version semver")
			},
		},
		{
			key:   wkey("terminalbackend.go", "CheckVersionTuple", "CodeDrift", "protocol_version major 1", 1),
			entry: "CheckVersionTuple",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckVersionTuple("com.example.term", "1.2.3", "2.0.0", []string{"2.0.0"})
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "protocol_version major 1")
			},
		},
		{
			key:   wkey("terminalbackend.go", "CheckVersionTuple", "CodeDrift", "protocol_version membership", 1),
			entry: "CheckVersionTuple",
			prove: func(t *testing.T) {
				err := terminalbackend.CheckVersionTuple("com.example.term", "1.2.3", "1.1.0", []string{"1.0.0"})
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "protocol_version membership")
			},
		},
		{
			key:   wkey("terminalbackend.go", "DefaultForPlatform", "CodeNotFound", "platform vocabulary", 1),
			entry: "DefaultForPlatform",
			prove: func(t *testing.T) {
				_, err := terminalbackend.DefaultForPlatform(scalar.Platform("plan9"))
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "platform vocabulary")
			},
		},
		{
			key:   wkey("terminalbackend.go", "New", "CodeDrift", "implementation_version semver", 1),
			entry: "New",
			prove: func(t *testing.T) {
				_, err := terminalbackend.New("v1", []string{testProto})
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "implementation_version semver")
			},
		},
		{
			key:   wkey("terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id bound", 1),
			entry: "ParseID",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseID("")
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "terminal_backend_id bound")
			},
		},
		{
			key:   wkey("terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id grammar", 1),
			entry: "ParseID",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseID("AX.TMUX")
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "terminal_backend_id grammar")
			},
		},
		{
			key:   wkey("terminalbackend.go", "ParseID", "CodeNotFound", "terminal_backend_id reserved namespace", 1),
			entry: "ParseID",
			prove: func(t *testing.T) {
				_, err := terminalbackend.ParseID("ax.evil")
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "terminal_backend_id reserved namespace")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registration.validate", "CodeDrift", "implementation_version semver", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				record := testExternalRecord("com.example.term")
				record.ImplementationVersion = "v1"
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "implementation_version semver")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registration.validate", "CodeUntrusted", "executable_digest", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				record := testExternalRecord("com.example.term")
				record.ExecutableDigest = ""
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsUntrusted, "executable_digest")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeAmbiguous", "duplicate backend_id", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				registry := testRegistry(t)
				mustRegisterExternal(t, registry, "com.example.term")
				err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), testExternalRecord("com.example.term"))
				requireRegistryRefusal(t, err, terminalbackend.IsAmbiguous, "duplicate backend_id")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeAmbiguous", "external_trust identity binding", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				record := testExternalRecord("com.example.other")
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsAmbiguous, "external_trust identity binding")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeDrift", "implementation drift", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				registry := testRegistry(t)
				mustRegisterExternal(t, registry, "com.example.term")
				record := testExternalRecord("com.example.term")
				record.ImplementationVersion = "1.2.4"
				err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "implementation drift")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeNotFound", "registry unavailable", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				var nilRegistry *terminalbackend.Registry
				err := nilRegistry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), testExternalRecord("com.example.term"))
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "registry unavailable")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "executable substitution", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				record := testExternalRecord("com.example.term")
				record.ExecutableDigest = testDigestOther
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsUntrusted, "executable substitution")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "external implementation_kind", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				record := testExternalRecord("com.example.term")
				record.Kind = terminalbackend.Kind("bogus_kind")
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsUntrusted, "external implementation_kind")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RegisterExternal", "CodeUntrusted", "external_trust disabled", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				entry := testTrustEntry("com.example.term")
				entry.Enabled = false
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, entry, testExternalRecord("com.example.term"))
				requireRegistryRefusal(t, err, terminalbackend.IsUntrusted, "external_trust disabled")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RequireRestoreBinding", "CodeNotFound", "registry unavailable", 1),
			entry: "RequireRestoreBinding",
			prove: func(t *testing.T) {
				var nilRegistry *terminalbackend.Registry
				_, err := nilRegistry.RequireRestoreBinding("com.example.term", "com.example.term")
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "registry unavailable")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.RequireRestoreBinding", "CodeRestoreMismatch", "restore requires the prior binding", 1),
			entry: "RequireRestoreBinding",
			prove: func(t *testing.T) {
				registry := testRegistry(t)
				mustRegisterExternal(t, registry, "com.example.term")
				_, err := registry.RequireRestoreBinding("com.example.term", terminalbackend.BuiltinTmux)
				requireRegistryRefusal(t, err, terminalbackend.IsRestoreMismatch, "restore requires the prior binding")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.Resolve", "CodeNotFound", "registry unavailable", 1),
			entry: "Resolve",
			prove: func(t *testing.T) {
				var nilRegistry *terminalbackend.Registry
				_, err := nilRegistry.Resolve(terminalbackend.BuiltinTmux)
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "registry unavailable")
			},
		},
		{
			key:   wkey("terminalbackend.go", "Registry.Resolve", "CodeNotFound", "unregistered terminal_backend_id", 1),
			entry: "Resolve",
			prove: func(t *testing.T) {
				_, err := testRegistry(t).Resolve("com.example.unknown")
				requireRegistryRefusal(t, err, terminalbackend.IsNotFound, "unregistered terminal_backend_id")
			},
		},
		{
			key:   wkey("terminalbackend.go", "TrustEntry.validate", "CodeAmbiguous", "external_trust reserved namespace", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				// ax.conpty passes mustParseID (canonical built-in), so
				// only the trust ax.-prefix bar can refuse it.
				entry := testTrustEntry(terminalbackend.BuiltinConpty)
				record := testExternalRecord(terminalbackend.BuiltinConpty)
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, entry, record)
				requireRegistryRefusal(t, err, terminalbackend.IsAmbiguous, "external_trust reserved namespace")
			},
		},
		{
			key:   wkey("terminalbackend.go", "TrustEntry.validate", "CodeUntrusted", "external_trust executable_digest", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				entry := testTrustEntry("com.example.term")
				entry.ExecutableDigest = "not-a-digest"
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, entry, testExternalRecord("com.example.term"))
				requireRegistryRefusal(t, err, terminalbackend.IsUntrusted, "external_trust executable_digest")
			},
		},
		{
			key:   wkey("terminalbackend.go", "TrustEntry.validate", "CodeUntrusted", "external_trust executable_path", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				entry := testTrustEntry("com.example.term")
				entry.ExecutablePath = "bin/ax-backend-term"
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, entry, testExternalRecord("com.example.term"))
				requireRegistryRefusal(t, err, terminalbackend.IsUntrusted, "external_trust executable_path")
			},
		},
		{
			key:   wkey("terminalbackend.go", "checkGeneration", "CodeStaleGeneration", "backend_generation bound", 1),
			entry: "CheckProviderDescriptor",
			prove: func(t *testing.T) {
				// Equal generations on both sides: the mismatch
				// comparison cannot fire, only the bound can refuse.
				binding := validDescriptorBinding()
				binding.Generation = ""
				err := terminalbackend.CheckProviderDescriptor(binding, binding)
				requireRegistryRefusal(t, err, terminalbackend.IsStaleGeneration, "backend_generation bound")
			},
		},
		{
			key:   wkey("terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions bound", 1),
			entry: "New",
			prove: func(t *testing.T) {
				_, err := terminalbackend.New(testImplVersion, nil)
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "protocol_versions bound")
			},
		},
		{
			key:   wkey("terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions major 1", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				// Valid semver outside major 1: the record semver check
				// passes, the major-1 rule fires inside validate.
				record := testExternalRecord("com.example.term")
				record.ProtocolVersions = []string{"2.0.0"}
				err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "protocol_versions major 1")
			},
		},
		{
			key:   wkey("terminalbackend.go", "validateProtocolVersions", "CodeDrift", "protocol_versions sorted unique", 1),
			entry: "New",
			prove: func(t *testing.T) {
				_, err := terminalbackend.New(testImplVersion, []string{testProto, testProto})
				requireRegistryRefusal(t, err, terminalbackend.IsDrift, "protocol_versions sorted unique")
			},
		},
		{
			// The funneled platform arm: one witness driving three
			// inputs, one per funneled detail, through RegisterExternal.
			// Every funneled detail must refuse here, not just one.
			key:   wkey("terminalbackend.go", "Registration.validate", "CodeNotFound", "passthrough:err.Error()", 1),
			entry: "RegisterExternal",
			prove: func(t *testing.T) {
				for _, tc := range []struct {
					name      string
					platforms []scalar.Platform
					detail    string
				}{
					{"empty platform set", nil, "platforms bound"},
					{"unknown platform member", []scalar.Platform{scalar.PlatformLinux, scalar.Platform("plan9")}, "platforms vocabulary"},
					{"duplicate platform", []scalar.Platform{scalar.PlatformLinux, scalar.PlatformLinux}, "platforms sorted unique"},
				} {
					record := testExternalRecord("com.example.term")
					record.Platforms = tc.platforms
					err := testRegistry(t).RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
					requireRegistryRefusal(t, err, terminalbackend.IsNotFound, tc.detail)
				}
			},
		},
	}
}
