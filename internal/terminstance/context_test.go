package terminstance

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func quiesceKey() string { return fixtureInstance + "/quiesce/" + fixtureQuiescence }

// TestParseMutationContextAdmitsClosedFixture drives the closed context
// through the production parse entry and asserts every member literally.
func TestParseMutationContextAdmitsClosedFixture(t *testing.T) {
	context := testContext(t)
	requireLiteral(t, "operation_id", context.OperationID, "0198f4c8-8e50-7f66-8f70-ccccccccccc1")
	requireLiteral(t, "session_id", context.SessionID, "0198f4c8-8e50-7f66-8f70-1234567890ab")
	requireLiteral(t, "terminal_instance_id", context.TerminalInstanceID, "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb1")
	requireLiteral(t, "terminal_backend_id", context.TerminalBackendID, "example.backend")
	requireLiteral(t, "implementation_version", context.ImplementationVersion, "1.2.3")
	requireLiteral(t, "protocol_version", context.ProtocolVersion, "1.0.0")
	requireLiteral(t, "backend_generation", context.BackendGeneration, "generation-one")
	requireLiteral(t, "idempotency_key", context.IdempotencyKey, "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb1/quiesce/0198f4c8-8e50-7f66-8f70-ddddddddddd1")
	requireLiteral(t, "deadline_at", context.Deadline.String(), "2026-09-01T18:00:00.000Z")
	requireLiteral(t, "authorization lease", context.Authorization.LeaseID, "f47ac10b-58cc-4372-a567-0e02b2c3d479")
}

// TestParseMutationContextMemberRefusals drives every context member
// grammar through the production entry.
func TestParseMutationContextMemberRefusals(t *testing.T) {
	build := func(member, raw string) string {
		members := map[string]string{
			"operation_id":           `"` + fixtureOperation + `"`,
			"session_id":             `"` + fixtureSessionA + `"`,
			"terminal_instance_id":   `"` + fixtureInstance + `"`,
			"terminal_backend_id":    `"` + fixtureBackend + `"`,
			"implementation_version": `"1.2.3"`,
			"protocol_version":       `"1.0.0"`,
			"backend_generation":     `"generation-one"`,
			"idempotency_key":        `"` + quiesceKey() + `"`,
			"deadline_at":            `"` + fixtureDeadline + `"`,
			"authorization":          authDoc("1", nil),
		}
		members[member] = raw
		order := []string{"operation_id", "session_id", "terminal_instance_id", "terminal_backend_id", "implementation_version", "protocol_version", "backend_generation", "idempotency_key", "deadline_at", "authorization"}
		var body strings.Builder
		body.WriteString("{")
		for i, name := range order {
			if i > 0 {
				body.WriteString(",")
			}
			body.WriteString(`"` + name + `":` + members[name])
		}
		body.WriteString("}")
		return body.String()
	}
	cases := []struct {
		name   string
		member string
		raw    string
		code   string
		detail string
	}{
		{"operation garbage uuid", "operation_id", `"nope"`, "terminal_backend_protocol_error", "mutation context operation"},
		{"session garbage uuid", "session_id", `"nope"`, "terminal_backend_protocol_error", "mutation context session"},
		{"instance garbage uuid", "terminal_instance_id", `"nope"`, "terminal_backend_protocol_error", "mutation context instance"},
		{"backend grammar", "terminal_backend_id", `"Has-Upper"`, "terminal_backend_not_found", "terminal_backend_id grammar"},
		{"backend reserved", "terminal_backend_id", `"ax.evil"`, "terminal_backend_not_found", "terminal_backend_id reserved namespace"},
		{"implementation not semver", "implementation_version", `"1.2"`, "terminal_backend_implementation_drift", "implementation_version semver"},
		{"protocol not semver", "protocol_version", `"v1"`, "terminal_backend_implementation_drift", "protocol_version major 1"},
		{"protocol major 2", "protocol_version", `"2.0.0"`, "terminal_backend_implementation_drift", "protocol_version major 1"},
		{"protocol huge major", "protocol_version", `"18446744073709551617.0.0"`, "terminal_backend_implementation_drift", "protocol_version major 1"},
		{"generation empty", "backend_generation", `""`, "terminal_backend_stale_generation", "backend_generation bound"},
		{"generation 257", "backend_generation", `"` + strings.Repeat("g", 257) + `"`, "terminal_backend_stale_generation", "backend_generation bound"},
		{"key empty", "idempotency_key", `""`, "terminal_backend_protocol_error", "mutation context key bound"},
		{"key 257", "idempotency_key", `"` + strings.Repeat("k", 257) + `"`, "terminal_backend_protocol_error", "mutation context key bound"},
		{"deadline garbage", "deadline_at", `"soon"`, "terminal_backend_protocol_error", "mutation context deadline"},
		{"authorization array", "authorization", `[]`, "terminal_backend_protocol_error", "mutation context authorization"},
		{"authorization bad epoch", "authorization", authDoc("0", nil), "terminal_backend_protocol_error", "ax authorization epoch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMutationContext([]byte(build(tc.member, tc.raw)))
			requireRefusal(t, err, tc.code, tc.detail)
		})
	}
}

// TestParseMutationContextGenerationBound256Multibyte proves the bound
// counts characters, not bytes: 256 two-byte characters are admitted in
// the context body.
func TestParseMutationContextGenerationBound256Multibyte(t *testing.T) {
	generation := strings.Repeat("é", 256)
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, generation, quiesceKey(), fixtureDeadline, authDoc("1", nil))
	context, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext(256 chars) error = %v", err)
	}
	requireLiteral(t, "backend_generation", context.BackendGeneration, generation)

	tooLong := strings.Repeat("é", 257)
	raw = contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, tooLong, quiesceKey(), fixtureDeadline, authDoc("1", nil))
	_, err = ParseMutationContext([]byte(raw))
	requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation bound")
}

// TestParseMutationContextClosedShape proves the closed context: unknown
// and missing members refuse, and frame violations never decode.
func TestParseMutationContextClosedShape(t *testing.T) {
	base := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, quiesceKey(), fixtureDeadline, authDoc("1", nil))
	extra := strings.Replace(base, `"authorization":`, `"trace_id":"smuggled","authorization":`, 1)
	_, err := ParseMutationContext([]byte(extra))
	requireRefusal(t, err, "terminal_backend_protocol_error", "mutation context members")

	_, err = ParseMutationContext([]byte(`[1]`))
	requireRefusal(t, err, "terminal_backend_protocol_error", "mutation context frame")

	_, err = ParseMutationContext([]byte(base + ` `))
	if err != nil {
		t.Fatalf("ParseMutationContext(trailing whitespace) error = %v", err)
	}
}

// TestCheckMutationContextGenerationBound proves the Go-level generation
// bound through the production entry: a hand-built context carrying a
// 257-character generation refuses with the landed stale class.
func TestCheckMutationContextGenerationBound(t *testing.T) {
	context := testContext(t)
	tampered := context
	tampered.BackendGeneration = strings.Repeat("g", 257)
	err := CheckMutationContext(terminalbackend.OperationQuiesceInput, tampered, testParams())
	requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation bound")
}

// TestKeySegmentsDerivesRowMaterial proves the exact idempotency key
// material per engine row through the production derivation, with the
// pinned UTF-8 infix literals.
func TestKeySegmentsDerivesRowMaterial(t *testing.T) {
	context := testContext(t)
	quiesce, err := KeySegments(terminalbackend.OperationQuiesceInput, context, testParams())
	if err != nil {
		t.Fatalf("KeySegments(quiesce-input) error = %v", err)
	}
	requireLiteral(t, "quiesce key", quiesce, fixtureInstance+"/quiesce/"+fixtureQuiescence)

	wait, err := KeySegments(terminalbackend.OperationWaitSafeBoundary, context, Params{QuiescenceGeneration: fixtureQuiescence, ProviderProofKind: ProofProviderQuiescence})
	if err != nil {
		t.Fatalf("KeySegments(wait-safe-boundary) error = %v", err)
	}
	requireLiteral(t, "wait key", wait, fixtureInstance+"/boundary/"+fixtureQuiescence+"/provider_quiescence")

	stop, err := KeySegments(terminalbackend.OperationRequestStop, context, Params{SafeBoundaryEvidenceID: fixtureDigestB})
	if err != nil {
		t.Fatalf("KeySegments(request-stop) error = %v", err)
	}
	requireLiteral(t, "stop key", stop, fixtureInstance+"/stop/"+fixtureDigestB)

	create, err := KeySegments(terminalbackend.OperationCreate, context, Params{BootstrapOperationID: fixtureBootstrap})
	if err != nil {
		t.Fatalf("KeySegments(create) error = %v", err)
	}
	requireLiteral(t, "create key", create, fixtureSessionA+"/"+fixtureBootstrap)

	if _, err := KeySegments(terminalbackend.OperationStatus, context, testParams()); err == nil {
		t.Error("KeySegments(status) = nil, want the scope refusal")
	} else {
		requireRefusal(t, err, "terminal_backend_protocol_error", "operation lifecycle scope")
	}
}

// TestCheckMutationContextKeyMaterial proves a carried key that differs
// from the canonical material is internally inconsistent: wrong infix,
// wrong order and wrong generation all refuse through the production
// entry with the protocol class, never mismatch (mismatch is the store
// seam's verdict, not this gate's).
func TestCheckMutationContextKeyMaterial(t *testing.T) {
	context := testContext(t)
	if err := CheckMutationContext(terminalbackend.OperationQuiesceInput, context, testParams()); err != nil {
		t.Fatalf("CheckMutationContext() error = %v", err)
	}
	badKeys := []string{
		fixtureInstance + "/quiesce-wrong/" + fixtureQuiescence,
		fixtureQuiescence + "/quiesce/" + fixtureInstance,
		fixtureInstance + "/quiesce/" + fixtureSessionB,
		fixtureInstance + "/quiesce/" + fixtureQuiescence + "/extra",
	}
	for _, key := range badKeys {
		tampered := context
		tampered.IdempotencyKey = key
		err := CheckMutationContext(terminalbackend.OperationQuiesceInput, tampered, testParams())
		requireRefusal(t, err, "terminal_backend_protocol_error", "idempotency key material")
	}
}

// TestCheckMutationContextParams proves the per-operation parameter arms:
// bootstrap and quiescence generations are UUIDv7, proof kinds parse,
// timeouts lie in [1..3600000] and the stop evidence is a digest.
func TestCheckMutationContextParams(t *testing.T) {
	context := testContext(t)
	create := context
	create.IdempotencyKey = fixtureSessionA + "/" + fixtureBootstrap
	if err := CheckMutationContext(terminalbackend.OperationCreate, create, Params{BootstrapOperationID: fixtureBootstrap}); err != nil {
		t.Fatalf("CheckMutationContext(create) error = %v", err)
	}
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationCreate, create, Params{BootstrapOperationID: "nope"}), "terminal_backend_protocol_error", "mutation context bootstrap")

	wait := context
	wait.IdempotencyKey = fixtureInstance + "/boundary/" + fixtureQuiescence + "/ax_checkpoint_boundary"
	waitParams := Params{QuiescenceGeneration: fixtureQuiescence, ProviderProofKind: ProofAXCheckpointBoundary, TimeoutMs: 3600000}
	if err := CheckMutationContext(terminalbackend.OperationWaitSafeBoundary, wait, waitParams); err != nil {
		t.Fatalf("CheckMutationContext(wait) error = %v", err)
	}
	bad := waitParams
	bad.TimeoutMs = 0
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationWaitSafeBoundary, wait, bad), "terminal_backend_protocol_error", "wait timeout bound")
	bad = waitParams
	bad.TimeoutMs = 3600001
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationWaitSafeBoundary, wait, bad), "terminal_backend_protocol_error", "wait timeout bound")
	bad = waitParams
	bad.ProviderProofKind = "provider_restart"
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationWaitSafeBoundary, wait, bad), "terminal_backend_protocol_error", "provider proof vocabulary")
	bad = waitParams
	bad.QuiescenceGeneration = "nope"
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationWaitSafeBoundary, wait, bad), "terminal_backend_protocol_error", "quiescence generation")

	quiesce := context
	badQuiesce := testParams()
	badQuiesce.QuiescenceGeneration = "nope"
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationQuiesceInput, quiesce, badQuiesce), "terminal_backend_protocol_error", "quiescence generation")

	stop := context
	stop.IdempotencyKey = fixtureInstance + "/stop/" + fixtureDigestB
	stopParams := Params{SafeBoundaryEvidenceID: fixtureDigestB, GracefulTimeoutMs: 1}
	if err := CheckMutationContext(terminalbackend.OperationRequestStop, stop, stopParams); err != nil {
		t.Fatalf("CheckMutationContext(request-stop) error = %v", err)
	}
	badStop := stopParams
	badStop.SafeBoundaryEvidenceID = "not-a-digest"
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationRequestStop, stop, badStop), "terminal_backend_protocol_error", "stop boundary evidence")
	badStop = stopParams
	badStop.GracefulTimeoutMs = 0
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationRequestStop, stop, badStop), "terminal_backend_protocol_error", "stop timeout bound")
	badStop = stopParams
	badStop.GracefulTimeoutMs = 3600001
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationRequestStop, stop, badStop), "terminal_backend_protocol_error", "stop timeout bound")
}

// TestCheckMutationContextVersionArms proves the Go-level version tuple
// through the production entry: a hand-built context carrying a
// non-semver implementation or a non-major-1 protocol refuses with the
// landed drift class, exactly like its document twin.
func TestCheckMutationContextVersionArms(t *testing.T) {
	context := testContext(t)
	tampered := context
	tampered.ImplementationVersion = "1.2"
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationQuiesceInput, tampered, testParams()), "terminal_backend_implementation_drift", "implementation_version semver")

	tampered = context
	tampered.ProtocolVersion = "2.0.0"
	requireRefusal(t, CheckMutationContext(terminalbackend.OperationQuiesceInput, tampered, testParams()), "terminal_backend_implementation_drift", "protocol_version major 1")
}
