package terminstance

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Params carries the per-operation members beyond the shared mutation
// context. Only the members the operation's closed body lists are read;
// the rest stay zero. Create is receipt-scoped in this engine (its full
// binding/entrypoint body belongs to provisioning), so it carries only
// the bootstrap operation ID its idempotency key binds. TimeoutMs and
// GracefulTimeoutMs are validated uint53[1..3600000] and carried for
// the backend wait; the "lesser of request deadline and timeout" wait
// bound is enforced by that wait (modeled here), which reports a coded
// quiesce_timeout or stop_timeout proving non-commit. The engine
// enforces the MutationContext deadline, never the lesser bound.
type Params struct {
	BootstrapOperationID   string
	QuiescenceGeneration   string
	ProviderProofKind      ProviderProofKind
	TimeoutMs              uint64
	SafeBoundaryEvidenceID string
	GracefulTimeoutMs      uint64
	Interactive            bool
}

// operationBodyMembers is the exact closed member set per engine
// operation body, in pinned order. Status has its own flat body
// (ParseStatusBody); every other operation is outside the engine scope.
var operationBodyMembers = map[terminalbackend.Operation][]string{
	terminalbackend.OperationQuiesceInput:     {"context", "quiescence_generation"},
	terminalbackend.OperationWaitSafeBoundary: {"context", "quiescence_generation", "provider_proof_kind", "timeout_ms"},
	terminalbackend.OperationRequestStop:      {"context", "safe_boundary_evidence_id", "graceful_timeout_ms"},
}

// ParseOperationBody admits one closed quiesce-input, wait-safe-boundary
// or request-stop body: the exact member set, the nested mutation
// context through the shared parse arms, and the per-operation members.
// Create carries no parsed body in this engine (receipt scope only) and
// every other operation is refused by the scope gate.
func ParseOperationBody(operation string, raw []byte) (MutationContext, Params, error) {
	parsed, err := terminalbackend.ParseOperation(operation)
	if err != nil {
		return MutationContext{}, Params{}, wrapLanded(err)
	}
	if _, ok := operationBodyMembers[parsed]; !ok {
		return MutationContext{}, Params{}, refuse(CodeProtocolError, "operation lifecycle scope")
	}
	object, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return MutationContext{}, Params{}, refuse(CodeProtocolError, "operation body frame")
	}
	return parseOperationBodyObject(parsed, object)
}

// parseOperationBodyObject admits one already-framed operation body. It is
// split from ParseOperationBody only so the member-set and per-member
// arms read linearly; there is no second admission path.
func parseOperationBodyObject(operation terminalbackend.Operation, object map[string]json.RawMessage) (MutationContext, Params, error) {
	want := operationBodyMembers[operation]
	if len(object) != len(want) {
		return MutationContext{}, Params{}, refuse(CodeProtocolError, "operation body members")
	}
	for _, member := range want {
		if _, known := object[member]; !known {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "operation body members")
		}
	}
	contextMembers, fault := environ.DecodeStrictObject(object["context"])
	if fault != nil {
		return MutationContext{}, Params{}, refuse(CodeProtocolError, "operation body context")
	}
	context, err := parseMutationContextObject(contextMembers)
	if err != nil {
		return MutationContext{}, Params{}, err
	}
	var params Params
	switch operation {
	case terminalbackend.OperationQuiesceInput:
		generation, ok := environ.CheckUUIDv7(object["quiescence_generation"])
		if !ok {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "quiescence generation")
		}
		params.QuiescenceGeneration = generation.String()
	case terminalbackend.OperationWaitSafeBoundary:
		generation, ok := environ.CheckUUIDv7(object["quiescence_generation"])
		if !ok {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "quiescence generation")
		}
		kindRaw, ok := rawString(object["provider_proof_kind"])
		if !ok {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "provider proof vocabulary")
		}
		kind, err := ParseProviderProofKind(kindRaw)
		if err != nil {
			return MutationContext{}, Params{}, err
		}
		timeout, ok := environ.CheckUint53Bounds(object["timeout_ms"], 1, 3600000)
		if !ok {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "wait timeout bound")
		}
		params.QuiescenceGeneration = generation.String()
		params.ProviderProofKind = kind
		params.TimeoutMs = timeout
	case terminalbackend.OperationRequestStop:
		evidence, ok := environ.CheckDigest(object["safe_boundary_evidence_id"])
		if !ok {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "stop boundary evidence")
		}
		timeout, ok := environ.CheckUint53Bounds(object["graceful_timeout_ms"], 1, 3600000)
		if !ok {
			return MutationContext{}, Params{}, refuse(CodeProtocolError, "stop timeout bound")
		}
		params.SafeBoundaryEvidenceID = evidence.String()
		params.GracefulTimeoutMs = timeout
	default:
		return MutationContext{}, Params{}, refuse(CodeProtocolError, "operation lifecycle scope")
	}
	if err := CheckMutationContext(operation, context, params); err != nil {
		return MutationContext{}, Params{}, err
	}
	return context, params, nil
}
