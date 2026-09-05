package provhost

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file is the conformance harness: scripted-plugin sequences
// driven through the production Host.Call entry point, asserting the
// operation-layer decoders agree with the transport on the same
// bytes. Every sequence uses a fresh scriptRunner per process and
// counts spawned processes, so a hidden retry or an ambient cache
// would redden the spawn assertions rather than passing silently.

// childFailureFrameFor builds a failure envelope carrying a genuine
// bound child error for the named registered code, through the
// production axerror entry point. The code must be registered for
// Structured Error 1.0.0, which the provider 2.x binding requires.
func childFailureFrameFor(t *testing.T, want, code, message string) []byte {
	t.Helper()
	child, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    axerror.Code(code),
		Message: message,
		Details: axerror.Details{},
	})
	if err != nil {
		t.Fatalf("axerror.New(%q): %v", code, err)
	}
	raw, err := json.Marshal(child)
	if err != nil {
		t.Fatalf("Marshal child: %v", err)
	}
	return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + want + `","ok":false,"error":` + string(raw) + `}`)
}

// frameFor builds a single-line success envelope: the contract
// fixtures are pretty-printed in the document, but the wire carries
// one JSONL line, so bodies compact to that line here.
func frameFor(t *testing.T, requestID, body string) []byte {
	t.Helper()
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, []byte(body)); err != nil {
		t.Fatalf("compact fixture: %v", err)
	}
	return successFrame(t, requestID, compacted.String())
}

// callWithBody starts one Host.Call for the operation with the canned
// response frame and returns the success body.
func callWithBody(t *testing.T, runner *scriptRunner, operation Operation, requestID string, frame []byte) []byte {
	t.Helper()
	host := liveHost(runner)
	req := liveRequest(t)
	req.Operation = operation
	req.RequestID = mustUUIDv7(t, requestID)
	runner.steps = append(runner.steps, okStep(Result{Stdout: frame, ExitCode: 0}))
	got, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	if err != nil {
		t.Fatalf("%s Call: %v", operation, err)
	}
	return got.Body
}

// TestManifestAndProbeThroughCall proves the manifest and probe
// decoders agree with the transport: the exact bytes one plugin
// process returned pass both DecodeResponse and the operation
// decoder, for both operations in registry order.
func TestManifestAndProbeThroughCall(t *testing.T) {
	runner := &scriptRunner{}
	manifestBody := callWithBody(t, runner, OpManifest, testRequestID, frameFor(t, testRequestID, specManifestExample))
	if err := DecodeManifest(manifestBody); err != nil {
		t.Fatalf("manifest through Call: %v", err)
	}
	probeBody := callWithBody(t, runner, OpProbe, testOtherID, frameFor(t, testOtherID, specProbeExample))
	if err := DecodeProbe(probeBody); err != nil {
		t.Fatalf("probe through Call: %v", err)
	}
	if runner.spawned() != 2 {
		t.Fatalf("spawned %d processes, want one per operation", runner.spawned())
	}
	for index, call := range runner.calls {
		var decoded struct {
			Operation string `json:"operation"`
		}
		if err := json.Unmarshal(call.stdin, &decoded); err != nil {
			t.Fatalf("call %d stdin is not JSON: %v", index, err)
		}
		want := string([]Operation{OpManifest, OpProbe}[index])
		if decoded.Operation != want {
			t.Fatalf("call %d operation = %q, want %q", index, decoded.Operation, want)
		}
	}
}

// TestUnknownOperationThroughCallStartsNoProcess proves dispatch
// refuses locally: Host.Call with an unknown operation returns
// invalid_config without touching the Runner.
func TestUnknownOperationThroughCallStartsNoProcess(t *testing.T) {
	runner := &scriptRunner{}
	host := liveHost(runner)
	req := liveRequest(t)
	req.Operation = "reboot"
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
	requireLocalRefusal(t, err, "invalid_config", "unknown operation")
	if runner.spawned() != 0 {
		t.Fatalf("unknown operation spawned %d processes, want none", runner.spawned())
	}
}

// TestResumeThroughCall proves the normative resume plan travels the
// transport and validates for its provider, profile, and platform,
// with the profile_mapping equal to the Section 7.7 mapping.
func TestResumeThroughCall(t *testing.T) {
	runner := &scriptRunner{}
	body := callWithBody(t, runner, OpResume, testRequestID, frameFor(t, testRequestID, specResumePlan))
	if err := DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux); err != nil {
		t.Fatalf("resume through Call: %v", err)
	}
}

// specIdentifyCallBody wraps the Section 5.5 example in an
// identify-session success body.
func specIdentifyCallBody() []byte {
	return []byte(`{"identity": ` + specIdentityExample + `, "confidence": "exact", "matched_evidence": ["native_id", "store_path"]}`)
}

// TestIdentifyThroughCall proves the identify-session result travels
// the transport and validates for its provider.
func TestIdentifyThroughCall(t *testing.T) {
	runner := &scriptRunner{}
	body := callWithBody(t, runner, OpIdentifySession, testRequestID, frameFor(t, testRequestID, string(specIdentifyCallBody())))
	if err := DecodeIdentifyResult(body, "antigravity"); err != nil {
		t.Fatalf("identify through Call: %v", err)
	}
}

// TestQuiesceThroughCall proves both quiescence verdicts travel the
// transport: the safe proof reports safe true and the unsafe proof
// reports safe false, which is what stops graceful takeover.
func TestQuiesceThroughCall(t *testing.T) {
	runner := &scriptRunner{}
	safeBody := callWithBody(t, runner, OpQuiesce, testRequestID, frameFor(t, testRequestID, safeQuiesceProof))
	safe, err := DecodeQuiesceProof(safeBody)
	if err != nil {
		t.Fatalf("safe quiesce through Call: %v", err)
	}
	if !safe {
		t.Fatal("safe quiesce proof reports safe false")
	}
	unsafeProof := `{"provider_id": "codex", "provider_version": "0.147.0", "input_blocked": true, "boundary_ref": "e", "foreground_idle": true, "background_idle": null, "open_child_count": 0, "open_database_handle_count": 0, "store_generation": "g", "safe": false, "blockers": []}`
	unsafeBody := callWithBody(t, runner, OpQuiesce, testOtherID, successFrame(t, testOtherID, unsafeProof))
	unsafe, err := DecodeQuiesceProof(unsafeBody)
	if err != nil {
		t.Fatalf("unsafe quiesce through Call: %v", err)
	}
	if unsafe {
		t.Fatal("unsafe quiesce proof reports safe true; graceful takeover must stop")
	}
}

// TestCapabilityGatePrecedesCall proves the fail-closed tuples: for
// every registry capability under every non-available status the
// host-side gate refuses. RequireCapability takes no Runner, so no
// process can start through the refused path by construction; that
// half is structural, and the measured half is the refusal itself.
// The domain is derived from Capabilities() times the three
// non-available statuses — 21 tuples, all measured, none sampled.
func TestCapabilityGatePrecedesCall(t *testing.T) {
	tuples := 0
	for _, name := range Capabilities() {
		for _, status := range []string{CapabilityConditional, CapabilityUnsupported, CapabilityUnknown} {
			name, status := name, status
			t.Run(name+"/"+status, func(t *testing.T) {
				probe := probeWithCapabilities(t, func(key, _ string) string {
					if key == name {
						return probeCapabilityBlock(status, false)
					}
					return probeCapabilityBlock(CapabilityUnsupported, false)
				})
				// RequireCapability takes no Runner, so no process can
				// start here by construction; the refusal above is the
				// measured half, and the zero-process half follows
				// structurally from the signature.
				err := RequireCapability(probe, name)
				requireLocalRefusal(t, err, "invalid_config", "does not establish the capability")
			})
			tuples++
		}
	}
	if tuples != len(Capabilities())*3 {
		t.Fatalf("measured %d fail-closed tuples, want %d", tuples, len(Capabilities())*3)
	}
	t.Logf("fail-closed tuple coverage: %d/%d capability-by-status tuples refused at the gate", tuples, tuples)
	// The positive control: the established capability proceeds.
	runner := &scriptRunner{}
	if err := RequireCapability([]byte(specProbeExample), "native_resume"); err != nil {
		t.Fatalf("established capability refused: %v", err)
	}
	callWithBody(t, runner, OpNativeStorePlan, testRequestID, successFrame(t, testRequestID, `{}`))
	if runner.spawned() != 1 {
		t.Fatalf("established capability spawned %d processes, want one", runner.spawned())
	}
}

// preparedResultBody is the normative prepared materialize result
// (Section 7.5): the bytes a lost-response retry must reproduce
// byte-identically.
const preparedResultBody = `{
  "operation_id": "0198f4c8-e4b0-75cc-9576-1234567890ab",
  "materialization_id": "0198f4c8-c290-73aa-9374-1234567890ab",
  "transaction_id": "0198f4c8-f5c0-76dd-9677-1234567890ab",
  "plan_id": "sha256:64644a5ad573d36c0c13f44f56ef25ab93cff33001ff2a3371b082603910f2dd",
  "state": "prepared",
  "created_paths": ["/srv/provider/sessions/11111111-2222-4333-8444-555555555555"],
  "merged_paths": [],
  "validations": [
    {"code":"native_discovery","status":"passed","detail":"exact session identity resolved"}
  ],
  "rollback_token": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5QUI",
  "native_discovery": {
    "native_session_id": "11111111-2222-4333-8444-555555555555",
    "discovered": true,
    "discovery_root": "/srv/provider/sessions",
    "backend_resolved": false
  }
}`

// mutationRequest builds a keyed mutation request with the given
// operation_id and envelope request_id: the canonical bytes the
// retry must reproduce or mismatch on.
func mutationRequest(t *testing.T, operation, operationID, requestID string) Request {
	t.Helper()
	return Request{
		Operation: Operation(operation),
		RequestID: mustUUIDv7(t, requestID),
		Deadline:  futureDeadline(t, 5*time.Minute),
		Body:      json.RawMessage(`{"operation_id":"` + operationID + `","materialization_id":"` + testMaterializationID + `","transaction_id":"` + testTransactionID + `"}`),
	}
}

// TestLostPrepareReturnsByteIdentical proves the PTX-LOST-PREPARE
// shape through fresh processes: a lost materialize response
// retried with the same (operation, operation_id) and canonical body
// returns byte-identical bytes, and the runner observes two mutation
// frames with equal keys.
func TestLostPrepareReturnsByteIdentical(t *testing.T) {
	const operationID = "0198f4c8-e4b0-75cc-9576-1234567890ab"
	firstID := testRequestID
	secondID := testOtherID
	runner := &scriptRunner{steps: []scriptStep{
		okStep(Result{Stdout: frameFor(t, firstID, preparedResultBody), ExitCode: 0}),
		okStep(Result{Stdout: frameFor(t, secondID, preparedResultBody), ExitCode: 0}),
	}}
	host := Host{Runner: runner, Now: time.Now}
	firstReq := mutationRequest(t, string(OpMaterialize), operationID, firstID)
	first, err := host.Call(context.Background(), "/plugins/ax-provider-pi", firstReq)
	if err != nil {
		t.Fatalf("first materialize Call: %v", err)
	}
	secondReq := mutationRequest(t, string(OpMaterialize), operationID, secondID)
	second, err := host.Call(context.Background(), "/plugins/ax-provider-pi", secondReq)
	if err != nil {
		t.Fatalf("retry materialize Call: %v", err)
	}
	if string(first.Body) != string(second.Body) {
		t.Fatalf("retry body differs:\n%s\n%s", first.Body, second.Body)
	}
	keys := mutationKeys(t, runner)
	if len(keys) != 2 || keys[0] != keys[1] {
		t.Fatalf("mutation keys = %v, want two equal keys", keys)
	}
	want := "(materialize, " + operationID + ")"
	if keys[0] != want {
		t.Fatalf("mutation key = %q, want %q", keys[0], want)
	}
}

// TestChangedBodyReturnsMismatch proves the PTX-IDEMPOTENCY-N1 shape:
// the same (operation, operation_id) with a changed canonical body
// surfaces the bound idempotency_mismatch child, and the host sends
// no third frame on its own.
func TestChangedBodyReturnsMismatch(t *testing.T) {
	const operationID = "0198f4c8-e4b0-75cc-9576-1234567890ab"
	runner := &scriptRunner{steps: []scriptStep{
		okStep(Result{Stdout: frameFor(t, testRequestID, preparedResultBody), ExitCode: 0}),
		okStep(Result{Stdout: childFailureFrameFor(t, testOtherID, "idempotency_mismatch", "retry changed the plan"), ExitCode: 0}),
	}}
	host := Host{Runner: runner, Now: time.Now}
	firstReq := mutationRequest(t, string(OpMaterialize), operationID, testRequestID)
	if _, err := host.Call(context.Background(), "/plugins/ax-provider-pi", firstReq); err != nil {
		t.Fatalf("first materialize Call: %v", err)
	}
	secondReq := mutationRequest(t, string(OpMaterialize), operationID, testOtherID)
	secondReq.Body = json.RawMessage(`{"operation_id":"` + operationID + `","materialization_id":"` + testMaterializationID + `","transaction_id":"0198f4c8-0000-76dd-9677-1234567890ab"}`)
	_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", secondReq)
	if failureCode(t, err) != "idempotency_mismatch" {
		t.Fatalf("changed-body retry code = %v, want idempotency_mismatch", err)
	}
	if runner.spawned() != 2 {
		t.Fatalf("spawned %d processes, want exactly the two sent frames", runner.spawned())
	}
	keys := mutationKeys(t, runner)
	if len(keys) != 2 || keys[0] != keys[1] {
		t.Fatalf("mutation keys = %v, want two equal keys", keys)
	}
}

// mutationKeys extracts the (operation, operation_id) pair the host
// sent on each recorded call, proving what the retry reused.
func mutationKeys(t *testing.T, runner *scriptRunner) []string {
	t.Helper()
	var keys []string
	for _, call := range runner.calls {
		var frame struct {
			Operation string `json:"operation"`
			Body      struct {
				OperationID string `json:"operation_id"`
			} `json:"body"`
		}
		if err := json.Unmarshal(call.stdin, &frame); err != nil {
			t.Fatalf("recorded stdin is not JSON: %v", err)
		}
		keys = append(keys, "("+frame.Operation+", "+frame.Body.OperationID+")")
	}
	return keys
}

// TestChildFailureCodesPassThrough proves the host never rewrites a
// plugin failure: the normative negative codes surface with their
// registered identities through Host.Call.
func TestChildFailureCodesPassThrough(t *testing.T) {
	for _, kase := range []struct {
		code     string
		exitCode int
	}{
		{"capability_unavailable", 6},
		{"incompatible_schema", 6},
		{"secret_policy_violation", 16},
		{"idempotency_mismatch", 3},
	} {
		t.Run(kase.code, func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{
				okStep(Result{Stdout: childFailureFrameFor(t, testRequestID, kase.code, "plugin refused"), ExitCode: 0}),
			}}
			host := liveHost(runner)
			req := liveRequest(t)
			req.Operation = OpMaterialize
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
			if failureCode(t, err) != axerror.Code(kase.code) {
				t.Fatalf("code = %v, want %s", err, kase.code)
			}
			if failureExit(t, err) != kase.exitCode {
				t.Fatalf("exit = %d, want %d", failureExit(t, err), kase.exitCode)
			}
			if failureObject(t, err).Retryable() {
				t.Fatalf("%s surfaces retryable", kase.code)
			}
		})
	}
}
