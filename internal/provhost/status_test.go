package provhost

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const (
	testMaterializationID = "0198f4c8-c290-73aa-9374-1234567890ab"
	testTransactionID     = "0198f4c8-f5c0-76dd-9677-1234567890ab"
	testAuthorityID       = "provider_transaction"
	testPlanID            = "sha256:64644a5ad573d36c0c13f44f56ef25ab93cff33001ff2a3371b082603910f2dd"
	testRollbackToken     = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5QUI"
	testDiscovery         = `{"native_session_id":"11111111-2222-4333-8444-555555555555","discovered":true,"discovery_root":"/srv/provider/sessions","backend_resolved":false}`
)

func testStatusIDs() StatusIDs {
	return StatusIDs{
		MaterializationID:      testMaterializationID,
		TransactionID:          testTransactionID,
		TransactionAuthorityID: testAuthorityID,
	}
}

// statusBody builds a ProviderTransactionStatus body. Null members are
// passed as the literal null.
func statusBody(materializationID, transactionID, authorityID, planID, state, token, discovery string) []byte {
	quote := func(value, fallback string) string {
		if value == "null" {
			return "null"
		}
		if value == "" {
			return fallback
		}
		return `"` + value + `"`
	}
	return []byte(`{"materialization_id":` + quote(materializationID, `"x"`) +
		`,"transaction_id":` + quote(transactionID, `"x"`) +
		`,"transaction_authority_id":` + quote(authorityID, `"x"`) +
		`,"plan_id":` + quote(planID, `"x"`) +
		`,"state":"` + state + `"` +
		`,"rollback_token":` + quote(token, `"x"`) +
		`,"native_discovery":` + discovery + `}`)
}

func preparedBody() []byte {
	return statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery)
}

// TestDecodeStatusOutcomeAcceptsTerminalStates proves the positive
// recovery path: prepared, committed, and rolled_back observations with
// the state their nullability rules require are returned as-is.
func TestDecodeStatusOutcomeAcceptsTerminalStates(t *testing.T) {
	for _, kase := range []struct {
		name  string
		body  []byte
		state StatusState
	}{
		{"prepared", preparedBody(), StatusPrepared},
		{"committed", statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "committed", "null", testDiscovery), StatusCommitted},
		{"rolled back", statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "rolled_back", "null", testDiscovery), StatusRolledBack},
	} {
		t.Run(kase.name, func(t *testing.T) {
			got, err := DecodeStatusOutcome(kase.body, testStatusIDs())
			if err != nil {
				t.Fatalf("DecodeStatusOutcome: %v", err)
			}
			if got != kase.state {
				t.Fatalf("DecodeStatusOutcome = %q, want %q", got, kase.state)
			}
		})
	}
}

// TestDecodeStatusOutcomeUnknownFailsClosed proves the quarantine rule:
// state unknown fails with integrity_failure and names the state in the
// details, and never returns a usable status.
func TestDecodeStatusOutcomeUnknownFailsClosed(t *testing.T) {
	body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, "null", "unknown", "null", "null")
	got, err := DecodeStatusOutcome(body, testStatusIDs())
	if got != "" {
		t.Fatalf("DecodeStatusOutcome = %q for unknown, want no status", got)
	}
	requireLocalRefusal(t, err, "integrity_failure", "transaction state is unknown")
	if state, ok := failureObject(t, err).Detail("status_state"); !ok || state != "unknown" {
		t.Fatalf("DecodeStatusOutcome status_state = %v, want unknown", state)
	}
	if failureExit(t, err) != 9 {
		t.Fatalf("DecodeStatusOutcome exit = %d, want 9", failureExit(t, err))
	}
}

// TestDecodeStatusOutcomeNullabilityMatrix proves every state enforces
// its null/non-null rule: unknown carries nothing, prepared carries
// plan, token, and discovery, and terminal states carry plan and
// discovery with no token. Each violation is integrity_failure. The
// production entry point is DecodeStatusOutcome.
func TestDecodeStatusOutcomeNullabilityMatrix(t *testing.T) {
	for _, kase := range []struct {
		name     string
		state    string
		plan     string
		token    string
		discover string
		detail   string
	}{
		{"unknown with plan", "unknown", testPlanID, "null", "null", "unknown status carries plan, token, or discovery"},
		{"unknown with token", "unknown", "null", testRollbackToken, "null", "unknown status carries plan, token, or discovery"},
		{"unknown with discovery", "unknown", "null", "null", testDiscovery, "unknown status carries plan, token, or discovery"},
		{"unknown with all", "unknown", testPlanID, testRollbackToken, testDiscovery, "unknown status carries plan, token, or discovery"},
		{"prepared without plan", "prepared", "null", testRollbackToken, testDiscovery, "prepared status misses plan, token, or discovery"},
		{"prepared without token", "prepared", testPlanID, "null", testDiscovery, "prepared status misses plan, token, or discovery"},
		{"prepared without discovery", "prepared", testPlanID, testRollbackToken, "null", "prepared status misses plan, token, or discovery"},
		{"prepared empty", "prepared", "null", "null", "null", "prepared status misses plan, token, or discovery"},
		{"committed without plan", "committed", "null", "null", testDiscovery, "terminal status misses plan or discovery or keeps a token"},
		{"committed without discovery", "committed", testPlanID, "null", "null", "terminal status misses plan or discovery or keeps a token"},
		{"committed keeping token", "committed", testPlanID, testRollbackToken, testDiscovery, "terminal status misses plan or discovery or keeps a token"},
		{"rolled back without plan", "rolled_back", "null", "null", testDiscovery, "terminal status misses plan or discovery or keeps a token"},
		{"rolled back without discovery", "rolled_back", testPlanID, "null", "null", "terminal status misses plan or discovery or keeps a token"},
		{"rolled back keeping token", "rolled_back", testPlanID, testRollbackToken, testDiscovery, "terminal status misses plan or discovery or keeps a token"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, kase.plan, kase.state, kase.token, kase.discover)
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireLocalRefusal(t, err, "integrity_failure", kase.detail)
		})
	}
}

// TestDecodeStatusOutcomeRefusesForeignTransactions proves identity
// binding: a status naming another materialization, transaction, or
// authority is not this transaction's durable state.
func TestDecodeStatusOutcomeRefusesForeignTransactions(t *testing.T) {
	other := "0198f4c8-aaaa-73aa-9374-1234567890ab"
	for _, kase := range []struct {
		name   string
		body   []byte
		detail string
	}{
		{"other materialization", statusBody(other, testTransactionID, testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery), "status names another materialization"},
		{"other transaction", statusBody(testMaterializationID, other, testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery), "status names another transaction"},
		{"other authority", statusBody(testMaterializationID, testTransactionID, "another_authority", testPlanID, "prepared", testRollbackToken, testDiscovery), "status names another transaction authority"},
		{"non-string materialization", statusBody("", testTransactionID, testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery), "status names another materialization"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			body := kase.body
			if strings.Contains(kase.name, "non-string") {
				body = []byte(`{"materialization_id":7,"transaction_id":"` + testTransactionID + `","transaction_authority_id":"` + testAuthorityID + `","plan_id":"` + testPlanID + `","state":"prepared","rollback_token":"` + testRollbackToken + `","native_discovery":` + testDiscovery + `}`)
			}
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireLocalRefusal(t, err, "integrity_failure", kase.detail)
		})
	}
}

// TestDecodeStatusOutcomeRefusesMalformedBodies proves uninterpretable
// observations fail closed with integrity_failure rather than as absent
// or partial states.
func TestDecodeStatusOutcomeRefusesMalformedBodies(t *testing.T) {
	good := string(preparedBody())
	for _, kase := range []struct {
		name   string
		body   string
		detail string
	}{
		{"empty", ``, "status body is not a JSON object"},
		{"not JSON", `oops`, "status body is not a JSON object"},
		{"array", `[]`, "status body is not a JSON object"},
		{"scalar", `1`, "status body is not a JSON object"},
		{"trailing data", good + "x", "status body is trailing data after the object"},
		{"duplicate member", strings.Replace(good, `"state":"prepared"`, `"state":"prepared","state":"committed"`, 1), "status body is duplicate member"},
		{"unknown member", strings.Replace(good, `"state":"prepared"`, `"state":"prepared","phase":"prepared"`, 1), "status body carries unknown member"},
		{"missing plan", strings.Replace(good, `"plan_id":"`+testPlanID+`",`, ``, 1), "status body misses a required member"},
		{"missing state", strings.Replace(good, `,"state":"prepared"`, ``, 1), "status body misses a required member"},
		{"bad state", strings.Replace(good, `"prepared"`, `"preparing"`, 1), "status state is not a registry member"},
		{"non-string state", strings.Replace(good, `"state":"prepared"`, `"state":7`, 1), "status state is not a registry member"},
		{"malformed digest", strings.Replace(good, testPlanID, "sha256:xyz", 1), "status plan_id is not a digest"},
		{"non-string digest", strings.Replace(good, `"plan_id":"`+testPlanID+`"`, `"plan_id":7`, 1), "status plan_id is not a string"},
		{"short token", strings.Replace(good, testRollbackToken, "YWJj", 1), "status rollback_token is shorter than 256 bits"},
		{"non-base64 token", strings.Replace(good, testRollbackToken, "!!!not-base64!!!", 1), "status rollback_token is shorter than 256 bits"},
		{"non-string token", strings.Replace(good, `"rollback_token":"`+testRollbackToken+`"`, `"rollback_token":7`, 1), "status rollback_token is not a string"},
		{"scalar discovery", strings.Replace(good, `"native_discovery":`+testDiscovery, `"native_discovery":7`, 1), "status native_discovery is not an object"},
		{"malformed operation id", strings.Replace(good, `"materialization_id"`, `"operation_id":"bogus","materialization_id"`, 1), "status operation_id is not a UUIDv7"},
		{"non-string operation id", strings.Replace(good, `"materialization_id"`, `"operation_id":7,"materialization_id"`, 1), "status operation_id is not a string"},
	} {
		t.Run(kase.name, func(t *testing.T) {
			_, err := DecodeStatusOutcome([]byte(kase.body), testStatusIDs())
			requireLocalRefusal(t, err, "integrity_failure", kase.detail)
		})
	}
}

// tokenWithDecodedBytes builds a base64url rollback token decoding to
// exactly n bytes, so the 256-bit floor is pinned from both sides rather
// than only somewhere between the 3-byte short fixture and the 38-byte
// valid one.
func tokenWithDecodedBytes(t *testing.T, n int) string {
	t.Helper()
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x9e}, n))
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("DecodeString: %v", err)
	}
	if len(raw) != n {
		t.Fatalf("token decodes to %d bytes, want exactly %d", len(raw), n)
	}
	return token
}

// TestDecodeStatusOutcomeEnforcesTokenEntropyFloor proves the 256-bit
// rollback-token floor exactly: a token decoding to 31 bytes fails, and
// one decoding to 32 bytes is accepted. Any weakened floor at or below
// 248 bits admits the 31-byte token; any raised floor rejects the valid
// 32-byte observation. The production entry point is DecodeStatusOutcome.
func TestDecodeStatusOutcomeEnforcesTokenEntropyFloor(t *testing.T) {
	short := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "prepared", tokenWithDecodedBytes(t, 31), testDiscovery)
	_, err := DecodeStatusOutcome(short, testStatusIDs())
	requireLocalRefusal(t, err, "integrity_failure", "status rollback_token is shorter than 256 bits")

	floor := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "prepared", tokenWithDecodedBytes(t, 32), testDiscovery)
	if got, err := DecodeStatusOutcome(floor, testStatusIDs()); err != nil || got != StatusPrepared {
		t.Fatalf("32-byte-token recovery = %q, %v; want prepared", got, err)
	}
}

// TestDecodeStatusOutcomeIgnoresWellFormedOperationID proves a present
// mutation key does not alias the read: status identity is the
// materialization/transaction pair, so a well-formed operation_id is
// accepted and ignored.
func TestDecodeStatusOutcomeIgnoresWellFormedOperationID(t *testing.T) {
	body := strings.Replace(string(preparedBody()), `"materialization_id"`, `"operation_id":"0198f4c8-e4b0-75cc-9576-1234567890ab","materialization_id"`, 1)
	got, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
	if err != nil {
		t.Fatalf("DecodeStatusOutcome with operation_id: %v", err)
	}
	if got != StatusPrepared {
		t.Fatalf("DecodeStatusOutcome = %q, want prepared", got)
	}
}

// TestStatusReadsEvolve proves no byte-identical replay rule applies to
// status observations: the same transaction may report prepared under
// one envelope and committed under the next, and both readings succeed.
func TestStatusReadsEvolve(t *testing.T) {
	ids := testStatusIDs()
	first, err := DecodeStatusOutcome(preparedBody(), ids)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	second, err := DecodeStatusOutcome(statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "committed", "null", testDiscovery), ids)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if first != StatusPrepared || second != StatusCommitted {
		t.Fatalf("evolving reads = %q then %q, want prepared then committed", first, second)
	}
}

// TestStatusRecoveryAcrossProcesses proves the cross-process recovery
// shape over the real entry points: two status Calls start two separate
// processes with frames identical except request_id, and each durable
// observation reconciles independently.
func TestStatusRecoveryAcrossProcesses(t *testing.T) {
	committed := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "committed", "null", testDiscovery)
	runner := &scriptRunner{steps: []scriptStep{
		okStep(Result{Stdout: successFrame(t, testRequestID, string(preparedBody())), ExitCode: 0}),
		okStep(Result{Stdout: successFrame(t, testOtherID, string(committed)), ExitCode: 0}),
	}}
	host := liveHost(runner)
	firstReq := liveRequest(t)
	firstReq.Operation = OpMaterializeStatus
	firstReq.RequestID = mustUUIDv7(t, testRequestID)
	first, err := host.Call(context.Background(), "/plugins/ax-provider-pi", firstReq)
	if err != nil {
		t.Fatalf("first status Call: %v", err)
	}
	secondReq := liveRequest(t)
	secondReq.Operation = OpMaterializeStatus
	secondReq.RequestID = mustUUIDv7(t, testOtherID)
	second, err := host.Call(context.Background(), "/plugins/ax-provider-pi", secondReq)
	if err != nil {
		t.Fatalf("second status Call: %v", err)
	}
	if runner.spawned() != 2 {
		t.Fatalf("two status reads spawned %d processes, want one per read", runner.spawned())
	}
	if firstState, err := DecodeStatusOutcome(first.Body, testStatusIDs()); err != nil || firstState != StatusPrepared {
		t.Fatalf("first recovery = %q, %v; want prepared", firstState, err)
	}
	if secondState, err := DecodeStatusOutcome(second.Body, testStatusIDs()); err != nil || secondState != StatusCommitted {
		t.Fatalf("second recovery = %q, %v; want committed", secondState, err)
	}
}

// TestLostMutationRecoversThroughStatus proves the lost-prepare shape: a
// mutation Call whose response is lost is followed by a status read, and
// the runner observes exactly one mutation frame — the host never
// re-issues the mutation on its own, so no second destination mutation
// can occur.
func TestLostMutationRecoversThroughStatus(t *testing.T) {
	runner := &scriptRunner{
		steps: []scriptStep{
			failStep(context.DeadlineExceeded),
			okStep(Result{Stdout: successFrame(t, testOtherID, string(preparedBody())), ExitCode: 0}),
		},
	}
	// The runner error above fires with a live deadline, so it is a
	// transport failure, not a timeout.
	host := Host{Runner: runner, Now: time.Now}
	mutation := liveRequest(t)
	mutation.Operation = OpMaterialize
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", mutation)
	requireLocalRefusal(t, err, "provider_process_failed", "runner reported no result")
	status := liveRequest(t)
	status.Operation = OpMaterializeStatus
	status.RequestID = mustUUIDv7(t, testOtherID)
	got, err := host.Call(context.Background(), "/plugins/ax-provider-pi", status)
	if err != nil {
		t.Fatalf("status Call: %v", err)
	}
	state, err := DecodeStatusOutcome(got.Body, testStatusIDs())
	if err != nil || state != StatusPrepared {
		t.Fatalf("recovery = %q, %v; want prepared", state, err)
	}
	if runner.spawned() != 2 {
		t.Fatalf("recovery spawned %d processes, want mutation plus one status read", runner.spawned())
	}
	mutations := 0
	for _, call := range runner.calls {
		var decoded struct {
			Operation string `json:"operation"`
		}
		if err := json.Unmarshal(call.stdin, &decoded); err != nil {
			t.Fatalf("recorded stdin is not JSON: %v", err)
		}
		if decoded.Operation == string(OpMaterialize) {
			mutations++
		}
	}
	if mutations != 1 {
		t.Fatalf("runner observed %d mutation frames, want exactly one and no blind retry", mutations)
	}
}

// TestUnknownTransactionQuarantinesThroughCall proves the unknown
// transaction path end to end: a status body of unknown surfaces
// integrity_failure from the composed read, never a status.
func TestUnknownTransactionQuarantinesThroughCall(t *testing.T) {
	unknown := statusBody(testMaterializationID, testTransactionID, testAuthorityID, "null", "unknown", "null", "null")
	runner := &scriptRunner{steps: []scriptStep{okStep(Result{Stdout: successFrame(t, testRequestID, string(unknown)), ExitCode: 0})}}
	host := liveHost(runner)
	req := liveRequest(t)
	req.Operation = OpMaterializeStatus
	got, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	if err != nil {
		t.Fatalf("status Call: %v", err)
	}
	state, err := DecodeStatusOutcome(got.Body, testStatusIDs())
	if state != "" {
		t.Fatalf("unknown recovery = %q, want no status", state)
	}
	requireLocalRefusal(t, err, "integrity_failure", "transaction state is unknown")
}
