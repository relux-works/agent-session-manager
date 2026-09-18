package terminstance

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// MutationContext is the closed §4.C MutationContext object: exactly
// operation_id, session_id, terminal_instance_id, terminal_backend_id,
// implementation_version, protocol_version, backend_generation,
// idempotency_key, deadline_at and authorization. Every mutating request
// carries it.
type MutationContext struct {
	OperationID           string
	SessionID             string
	TerminalInstanceID    string
	TerminalBackendID     string
	ImplementationVersion string
	ProtocolVersion       string
	BackendGeneration     string
	IdempotencyKey        string
	Deadline              scalar.Timestamp
	Authorization         AXAuthorization
}

// mutationContextMembers is the exact closed member set, in the order the
// pinned section lists it.
var mutationContextMembers = []string{
	"operation_id",
	"session_id",
	"terminal_instance_id",
	"terminal_backend_id",
	"implementation_version",
	"protocol_version",
	"backend_generation",
	"idempotency_key",
	"deadline_at",
	"authorization",
}

// ParseMutationContext admits one closed MutationContext document. The
// nested authorization is admitted by the shared
// parseAXAuthorizationObject arms, never a second implementation.
// Malformed documents are terminal_backend_protocol_error (AX-local
// parse); semantic binding (key material, version tuple, generation
// staleness) is CheckMutationContext, not this function.
func ParseMutationContext(raw []byte) (MutationContext, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context frame")
	}
	return parseMutationContextObject(members)
}

// parseMutationContextObject admits one already-framed context member map.
// ParseMutationContext and ParseOperationBody share it so the nested
// context inside an operation body is admitted by the same arms.
func parseMutationContextObject(members map[string]json.RawMessage) (MutationContext, error) {
	if len(members) != len(mutationContextMembers) {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context members")
	}
	for _, member := range mutationContextMembers {
		if _, known := members[member]; !known {
			return MutationContext{}, refuse(CodeProtocolError, "mutation context members")
		}
	}
	operationID, ok := environ.CheckUUIDv7(members["operation_id"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context operation")
	}
	sessionID, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context session")
	}
	instanceID, ok := environ.CheckUUIDv7(members["terminal_instance_id"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context instance")
	}
	backendID, ok := rawString(members["terminal_backend_id"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context backend")
	}
	if _, err := terminalbackend.ParseID(backendID); err != nil {
		return MutationContext{}, wrapLanded(err)
	}
	implementationVersion, ok := rawString(members["implementation_version"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context versions")
	}
	protocolVersion, ok := rawString(members["protocol_version"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context versions")
	}
	if err := checkVersionTuple(backendID, implementationVersion, protocolVersion); err != nil {
		return MutationContext{}, err
	}
	generation, ok := environ.CheckStringBounds(members["backend_generation"], 1, 256)
	if !ok {
		return MutationContext{}, refuse(CodeStaleGeneration, "backend_generation bound")
	}
	idempotencyKey, ok := environ.CheckStringBounds(members["idempotency_key"], 1, 256)
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context key bound")
	}
	deadline, ok := environ.CheckTimestamp(members["deadline_at"])
	if !ok {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context deadline")
	}
	authMembers, fault := environ.DecodeStrictObject(members["authorization"])
	if fault != nil {
		return MutationContext{}, refuse(CodeProtocolError, "mutation context authorization")
	}
	authorization, err := parseAXAuthorizationObject(authMembers)
	if err != nil {
		return MutationContext{}, err
	}
	return MutationContext{
		OperationID:           operationID.String(),
		SessionID:             sessionID.String(),
		TerminalInstanceID:    instanceID.String(),
		TerminalBackendID:     backendID,
		ImplementationVersion: implementationVersion,
		ProtocolVersion:       protocolVersion,
		BackendGeneration:     generation,
		IdempotencyKey:        idempotencyKey,
		Deadline:              deadline,
		Authorization:         authorization,
	}, nil
}

// checkVersionTuple enforces the operation-body version rule:
// implementation_version is semver and protocol_version is semver in
// Terminal Backend Protocol major 1. It delegates both arms to the
// landed CheckVersionTuple with the carried protocol version as the
// membership list: membership against the admitted probe list is the
// registry's business at admission (landed RequireRestoreBinding,
// CheckVersionTuple, Reconcile), so the operation gate contributes only
// the grammar and major arms and states that bound explicitly instead of
// re-deriving the saturating major or the semver grammar.
func checkVersionTuple(backendID, implementationVersion, protocolVersion string) error {
	if err := terminalbackend.CheckVersionTuple(backendID, implementationVersion, protocolVersion, []string{protocolVersion}); err != nil {
		return wrapLanded(err)
	}
	return nil
}

// checkGenerationBound enforces string[1..256] in characters over valid
// UTF-8 on a Go-level generation value by delegating to the landed
// GenerationDigest bound arm; the digest half is discarded because §4.C
// results carry the raw string, not its digest. The refusal class is the
// landed terminal_backend_stale_generation, never a fresh code.
func checkGenerationBound(generation string) error {
	if _, err := terminalbackend.GenerationDigest(generation); err != nil {
		return wrapLanded(err)
	}
	return nil
}

// KeySegments derives the canonical idempotency key for one engine
// operation from its identity material through the landed IdempotencyKey
// shapes. The infix literals are fixed per row, never caller input: a
// carried key that differs from the derived material is internally
// inconsistent and refused as terminal_backend_protocol_error, while a
// carried key that matches its material but collides with a different
// bound operation in the store is idempotency_mismatch at the receipt
// seam. create and restore share the (session, bootstrap) shape; only
// create executes in this engine.
func KeySegments(operation terminalbackend.Operation, context MutationContext, params Params) (string, error) {
	switch operation {
	case terminalbackend.OperationCreate:
		return terminalbackend.IdempotencyKey(string(operation), context.SessionID, params.BootstrapOperationID)
	case terminalbackend.OperationQuiesceInput:
		return terminalbackend.IdempotencyKey(string(operation), context.TerminalInstanceID, "quiesce", params.QuiescenceGeneration)
	case terminalbackend.OperationWaitSafeBoundary:
		return terminalbackend.IdempotencyKey(string(operation), context.TerminalInstanceID, "boundary", params.QuiescenceGeneration, string(params.ProviderProofKind))
	case terminalbackend.OperationRequestStop:
		return terminalbackend.IdempotencyKey(string(operation), context.TerminalInstanceID, "stop", params.SafeBoundaryEvidenceID)
	default:
		return "", refuse(CodeProtocolError, "operation lifecycle scope")
	}
}

// CheckMutationContext enforces the semantic half of the mutating-request
// contract for one already-parsed context: the carried key equals the
// canonical material for the operation and the Go-level values re-admit
// through their landed gates, so a hand-built context cannot bypass the
// document arms. Generation staleness against the validated binding is
// the engine's per-effect recheck, not this function.
func CheckMutationContext(operation terminalbackend.Operation, context MutationContext, params Params) error {
	if _, err := terminalbackend.ParseOperation(string(operation)); err != nil {
		return wrapLanded(err)
	}
	if _, err := scalar.ParseUUIDv7(context.OperationID); err != nil {
		return refuse(CodeProtocolError, "mutation context operation")
	}
	if _, err := scalar.ParseUUIDv7(context.SessionID); err != nil {
		return refuse(CodeProtocolError, "mutation context session")
	}
	if _, err := scalar.ParseUUIDv7(context.TerminalInstanceID); err != nil {
		return refuse(CodeProtocolError, "mutation context instance")
	}
	if _, err := terminalbackend.ParseID(context.TerminalBackendID); err != nil {
		return wrapLanded(err)
	}
	if err := checkVersionTuple(context.TerminalBackendID, context.ImplementationVersion, context.ProtocolVersion); err != nil {
		return err
	}
	if err := checkGenerationBound(context.BackendGeneration); err != nil {
		return err
	}
	if err := checkParams(operation, params); err != nil {
		return err
	}
	if _, err := context.Deadline.Time(); err != nil {
		return refuse(CodeProtocolError, "mutation context deadline")
	}
	derived, err := KeySegments(operation, context, params)
	if err != nil {
		return err
	}
	if context.IdempotencyKey != derived {
		return refuse(CodeProtocolError, "idempotency key material")
	}
	return nil
}

// checkParams validates the per-operation parameters: quiescence
// generations are UUIDv7, proof kinds parse, timeouts lie in
// uint53[1..3600000] and the stop evidence is a digest.
func checkParams(operation terminalbackend.Operation, params Params) error {
	switch operation {
	case terminalbackend.OperationCreate:
		if _, err := scalar.ParseUUIDv7(params.BootstrapOperationID); err != nil {
			return refuse(CodeProtocolError, "mutation context bootstrap")
		}
		return nil
	case terminalbackend.OperationQuiesceInput:
		if _, err := scalar.ParseUUIDv7(params.QuiescenceGeneration); err != nil {
			return refuse(CodeProtocolError, "quiescence generation")
		}
		return nil
	case terminalbackend.OperationWaitSafeBoundary:
		if _, err := scalar.ParseUUIDv7(params.QuiescenceGeneration); err != nil {
			return refuse(CodeProtocolError, "quiescence generation")
		}
		if _, err := ParseProviderProofKind(string(params.ProviderProofKind)); err != nil {
			return err
		}
		if params.TimeoutMs < 1 || params.TimeoutMs > 3600000 {
			return refuse(CodeProtocolError, "wait timeout bound")
		}
		return nil
	case terminalbackend.OperationRequestStop:
		if _, err := scalar.ParseDigest(params.SafeBoundaryEvidenceID); err != nil {
			return refuse(CodeProtocolError, "stop boundary evidence")
		}
		if params.GracefulTimeoutMs < 1 || params.GracefulTimeoutMs > 3600000 {
			return refuse(CodeProtocolError, "stop timeout bound")
		}
		return nil
	default:
		return refuse(CodeProtocolError, "operation lifecycle scope")
	}
}
