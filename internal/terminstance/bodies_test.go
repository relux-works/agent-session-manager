package terminstance

import (
	"strings"
	"testing"
)

func quiesceBody() string {
	return `{"context":` + contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, quiesceKey(), fixtureDeadline, authDoc("1", nil)) + `,"quiescence_generation":"` + fixtureQuiescence + `"}`
}

func waitBody() string {
	key := fixtureInstance + "/boundary/" + fixtureQuiescence + "/provider_quiescence"
	return `{"context":` + contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, fixtureDeadline, authDoc("1", nil)) + `,"quiescence_generation":"` + fixtureQuiescence + `","provider_proof_kind":"provider_quiescence","timeout_ms":1000}`
}

func stopBody() string {
	key := fixtureInstance + "/stop/" + fixtureDigestB
	return `{"context":` + contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, fixtureDeadline, authDoc("1", nil)) + `,"safe_boundary_evidence_id":"` + fixtureDigestB + `","graceful_timeout_ms":1000}`
}

// TestParseOperationBodyAdmitsRows drives the three closed operation
// bodies through the production entry and asserts the parsed members
// literally, including the carried key equal to the row material.
func TestParseOperationBodyAdmitsRows(t *testing.T) {
	context, params, err := ParseOperationBody("quiesce-input", []byte(quiesceBody()))
	if err != nil {
		t.Fatalf("ParseOperationBody(quiesce-input) error = %v", err)
	}
	requireLiteral(t, "quiescence_generation", params.QuiescenceGeneration, "0198f4c8-8e50-7f66-8f70-ddddddddddd1")
	requireLiteral(t, "quiesce key", context.IdempotencyKey, "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb1/quiesce/0198f4c8-8e50-7f66-8f70-ddddddddddd1")

	context, params, err = ParseOperationBody("wait-safe-boundary", []byte(waitBody()))
	if err != nil {
		t.Fatalf("ParseOperationBody(wait-safe-boundary) error = %v", err)
	}
	requireLiteral(t, "proof kind", string(params.ProviderProofKind), "provider_quiescence")
	if params.TimeoutMs != 1000 {
		t.Errorf("timeout_ms = %d, want 1000", params.TimeoutMs)
	}

	context, params, err = ParseOperationBody("request-stop", []byte(stopBody()))
	if err != nil {
		t.Fatalf("ParseOperationBody(request-stop) error = %v", err)
	}
	requireLiteral(t, "stop evidence", params.SafeBoundaryEvidenceID, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if params.GracefulTimeoutMs != 1000 {
		t.Errorf("graceful_timeout_ms = %d, want 1000", params.GracefulTimeoutMs)
	}
}

// TestParseOperationBodyRefusals drives the body member arms: wrong
// member sets, bad nested generations, bad proof kinds, out-of-bound
// timeouts (including the AX number model at timeout_ms) and bad stop
// evidence.
func TestParseOperationBodyRefusals(t *testing.T) {
	extra := strings.Replace(quiesceBody(), `"quiescence_generation"`, `"trace_id":"x","quiescence_generation"`, 1)
	_, _, err := ParseOperationBody("quiesce-input", []byte(extra))
	requireRefusal(t, err, "terminal_backend_protocol_error", "operation body members")

	_, _, err = ParseOperationBody("quiesce-input", []byte(`[1]`))
	requireRefusal(t, err, "terminal_backend_protocol_error", "operation body frame")

	badGen := strings.Replace(quiesceBody(), `"quiescence_generation":"`+fixtureQuiescence+`"`, `"quiescence_generation":"nope"`, 1)
	_, _, err = ParseOperationBody("quiesce-input", []byte(badGen))
	requireRefusal(t, err, "terminal_backend_protocol_error", "quiescence generation")

	badProof := strings.Replace(waitBody(), `"provider_proof_kind":"provider_quiescence"`, `"provider_proof_kind":"provider_restart"`, 1)
	_, _, err = ParseOperationBody("wait-safe-boundary", []byte(badProof))
	requireRefusal(t, err, "terminal_backend_protocol_error", "provider proof vocabulary")

	for _, timeout := range []string{"0", "3600001", "1.5", "1e3", `"1000"`, "null", "-5"} {
		bad := strings.Replace(waitBody(), `"timeout_ms":1000`, `"timeout_ms":`+timeout, 1)
		_, _, err = ParseOperationBody("wait-safe-boundary", []byte(bad))
		requireRefusal(t, err, "terminal_backend_protocol_error", "wait timeout bound")
	}
	for _, timeout := range []string{"1", "3600000"} {
		good := strings.Replace(waitBody(), `"timeout_ms":1000`, `"timeout_ms":`+timeout, 1)
		if _, _, err := ParseOperationBody("wait-safe-boundary", []byte(good)); err != nil {
			t.Errorf("ParseOperationBody(wait timeout %s) error = %v, want admission", timeout, err)
		}
	}

	badEvidence := strings.Replace(stopBody(), `"safe_boundary_evidence_id":"`+fixtureDigestB+`"`, `"safe_boundary_evidence_id":"not-a-digest"`, 1)
	_, _, err = ParseOperationBody("request-stop", []byte(badEvidence))
	requireRefusal(t, err, "terminal_backend_protocol_error", "stop boundary evidence")

	badStopTimeout := strings.Replace(stopBody(), `"graceful_timeout_ms":1000`, `"graceful_timeout_ms":0`, 1)
	_, _, err = ParseOperationBody("request-stop", []byte(badStopTimeout))
	requireRefusal(t, err, "terminal_backend_protocol_error", "stop timeout bound")

	// A body whose carried key disagrees with its own material is
	// internally inconsistent even though every member parses.
	wrongKey := strings.Replace(quiesceBody(), quiesceKey(), fixtureInstance+"/quiesce/"+fixtureSessionB, 1)
	_, _, err = ParseOperationBody("quiesce-input", []byte(wrongKey))
	requireRefusal(t, err, "terminal_backend_protocol_error", "idempotency key material")
}

// TestParseOperationBodyScopeGate proves operations outside the engine
// rows are refused at the production body entry.
func TestParseOperationBodyScopeGate(t *testing.T) {
	for _, operation := range []string{"status", "create", "attach", "restore", "terminate-stale", "manifest", "probe"} {
		_, _, err := ParseOperationBody(operation, []byte(quiesceBody()))
		requireRefusal(t, err, "terminal_backend_protocol_error", "operation lifecycle scope")
	}
	_, _, err := ParseOperationBody("launch", []byte(quiesceBody()))
	requireRefusal(t, err, "terminal_backend_protocol_error", "operation vocabulary")
}
