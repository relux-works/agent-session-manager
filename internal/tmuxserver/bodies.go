package tmuxserver

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// This file admits the four closed operation bodies the landed engine
// leaves out: the full create body (the engine runs create in receipt
// scope only), attach, terminate-stale, and restore. Quiesce-input,
// wait-safe-boundary, and request-stop bodies stay owned by
// terminstance.ParseOperationBody and status bodies by
// terminstance.ParseStatusBody; those parsers are reused, never
// re-implemented. Every nested document (mutation context, binding,
// attach authorization) is admitted by its landed owner; only the
// member lists — which only these schemas can name — are local.

// CreateBody is the closed §4.C create body: context, binding,
// bootstrap_operation_id, entrypoint, presentation_transport, and
// interactive. BindingRaw carries the binding member's exact bytes so
// the backend keeps the carried document for the restore read without
// re-encoding it.
type CreateBody struct {
	Context              terminstance.MutationContext
	Binding              termbind.Binding
	BindingRaw           json.RawMessage
	BootstrapOperationID string
	Transport            string
	Interactive          bool
}

// createBodyMembers is the exact closed member set.
var createBodyMembers = []string{
	"context",
	"binding",
	"bootstrap_operation_id",
	"entrypoint",
	"presentation_transport",
	"interactive",
}

// parseCreateBody admits one closed create body. The nested mutation
// context, terminal instance binding, and entrypoint are admitted by
// their landed owners; the transport admits the three-literal closed
// vocabulary here and the relay member refuses at the handler (create
// carries no attach authorization, so no landed relay arm stands
// downstream).
func parseCreateBody(raw []byte) (CreateBody, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return CreateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body frame"}
	}
	if len(members) != len(createBodyMembers) {
		return CreateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body members"}
	}
	for _, member := range createBodyMembers {
		if _, known := members[member]; !known {
			return CreateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body members"}
		}
	}
	context, err := terminstance.ParseMutationContext(members["context"])
	if err != nil {
		return CreateBody{}, err
	}
	binding, err := termbind.ParseTerminalBinding(members["binding"])
	if err != nil {
		return CreateBody{}, err
	}
	bootstrap, ok := environ.CheckUUIDv7(members["bootstrap_operation_id"])
	if !ok {
		return CreateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body bootstrap"}
	}
	argv, err := parseEntrypointArgv(members["entrypoint"])
	if err != nil {
		return CreateBody{}, err
	}
	if err := terminalbackend.CheckEntrypoint(argv, context.SessionID); err != nil {
		return CreateBody{}, err
	}
	transport, err := parsePresentationTransport(members["presentation_transport"])
	if err != nil {
		return CreateBody{}, err
	}
	var interactive bool
	if err := json.Unmarshal(members["interactive"], &interactive); err != nil {
		return CreateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body interactive"}
	}
	return CreateBody{
		Context:              context,
		Binding:              binding,
		BindingRaw:           members["binding"],
		BootstrapOperationID: bootstrap.String(),
		Transport:            transport,
		Interactive:          interactive,
	}, nil
}

// parseEntrypointArgv admits the closed Entrypoint object {argv} and
// reads its three-element vector. Length and literals are the landed
// CheckEntrypoint's verdict, not this function's.
func parseEntrypointArgv(raw json.RawMessage) ([]string, error) {
	object, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return nil, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body entrypoint"}
	}
	if len(object) != 1 {
		return nil, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body entrypoint"}
	}
	member, known := object["argv"]
	if !known {
		return nil, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body entrypoint"}
	}
	var argv []string
	if err := json.Unmarshal(member, &argv); err != nil {
		return nil, &Error{Code: terminalbackend.CodeProtocolError, Detail: "create body entrypoint"}
	}
	return argv, nil
}

// parsePresentationTransport admits the closed three-literal transport
// vocabulary. Admission is not authorization: the relay member parses
// and refuses downstream (landed attach arm for attach, the create
// relay arm for create).
func parsePresentationTransport(raw json.RawMessage) (string, error) {
	transport, ok := bodyString(raw)
	if !ok {
		return "", &Error{Code: terminalbackend.CodeProtocolError, Detail: "presentation transport vocabulary"}
	}
	switch transport {
	case string(terminalbackend.TransportLocalOnly),
		string(terminalbackend.TransportTrustedPrivateMesh),
		string(terminalbackend.TransportThirdPartyRelay):
		return transport, nil
	default:
		return "", &Error{Code: terminalbackend.CodeProtocolError, Detail: "presentation transport vocabulary"}
	}
}

// AttachBody is the closed §4.C attach body: the flat identity tuple,
// client, transport, input boolean, deadline, and attach authorization.
// AuthRaw carries the authorization member's raw bytes so the attach
// store re-admits the exact presented document without a re-frame.
type AttachBody struct {
	SessionID             string
	TerminalInstanceID    string
	TerminalBackendID     string
	ImplementationVersion string
	ProtocolVersion       string
	BackendGeneration     string
	ClientID              string
	Transport             string
	InputAuthorized       bool
	Deadline              scalar.Timestamp
	Authorization         terminalbackend.AttachAuthorization
	AuthRaw               json.RawMessage
}

// attachBodyMembers is the exact closed member set.
var attachBodyMembers = []string{
	"session_id",
	"terminal_instance_id",
	"terminal_backend_id",
	"implementation_version",
	"protocol_version",
	"backend_generation",
	"client_id",
	"transport",
	"input_authorized",
	"deadline_at",
	"authorization",
}

// parseAttachBody admits one closed attach body. The nested attach
// authorization is admitted by its landed owner; version-tuple and
// generation grammars delegate to the landed gates the status body
// already uses.
func parseAttachBody(raw []byte) (AttachBody, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body frame"}
	}
	if len(members) != len(attachBodyMembers) {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body members"}
	}
	for _, member := range attachBodyMembers {
		if _, known := members[member]; !known {
			return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body members"}
		}
	}
	session, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body session"}
	}
	instance, ok := environ.CheckUUIDv7(members["terminal_instance_id"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body instance"}
	}
	backendID, ok := bodyString(members["terminal_backend_id"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body backend"}
	}
	if _, err := terminalbackend.ParseID(backendID); err != nil {
		return AttachBody{}, err
	}
	implementationVersion, ok := bodyString(members["implementation_version"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body versions"}
	}
	protocolVersion, ok := bodyString(members["protocol_version"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body versions"}
	}
	if err := terminalbackend.CheckVersionTuple(backendID, implementationVersion, protocolVersion, []string{protocolVersion}); err != nil {
		return AttachBody{}, err
	}
	generation, ok := environ.CheckStringBounds(members["backend_generation"], 1, 256)
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeStaleGeneration, Detail: "backend_generation bound"}
	}
	client, ok := environ.CheckUUIDv7(members["client_id"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body client"}
	}
	transport, err := parsePresentationTransport(members["transport"])
	if err != nil {
		return AttachBody{}, err
	}
	var inputAuthorized bool
	if err := json.Unmarshal(members["input_authorized"], &inputAuthorized); err != nil {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body input"}
	}
	deadline, ok := environ.CheckTimestamp(members["deadline_at"])
	if !ok {
		return AttachBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "attach body deadline"}
	}
	authorization, err := terminalbackend.ParseAttachAuthorization(members["authorization"])
	if err != nil {
		return AttachBody{}, err
	}
	return AttachBody{
		SessionID:             session.String(),
		TerminalInstanceID:    instance.String(),
		TerminalBackendID:     backendID,
		ImplementationVersion: implementationVersion,
		ProtocolVersion:       protocolVersion,
		BackendGeneration:     generation,
		ClientID:              client.String(),
		Transport:             transport,
		InputAuthorized:       inputAuthorized,
		Deadline:              deadline,
		Authorization:         authorization,
		AuthRaw:               members["authorization"],
	}, nil
}

// TerminateBody is the closed §4.C terminate-stale body: context,
// stale_lease_id, stale_epoch, and diagnostic_evidence_id.
type TerminateBody struct {
	Context              terminstance.MutationContext
	StaleLeaseID         string
	StaleEpoch           uint64
	DiagnosticEvidenceID string
}

// terminateBodyMembers is the exact closed member set.
var terminateBodyMembers = []string{
	"context",
	"stale_lease_id",
	"stale_epoch",
	"diagnostic_evidence_id",
}

// parseTerminateBody admits one closed terminate-stale body.
func parseTerminateBody(raw []byte) (TerminateBody, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body frame"}
	}
	if len(members) != len(terminateBodyMembers) {
		return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body members"}
	}
	for _, member := range terminateBodyMembers {
		if _, known := members[member]; !known {
			return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body members"}
		}
	}
	context, err := terminstance.ParseMutationContext(members["context"])
	if err != nil {
		return TerminateBody{}, err
	}
	leaseRaw, ok := bodyString(members["stale_lease_id"])
	if !ok {
		return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body lease"}
	}
	lease, err := scalar.ParseUUIDv4(leaseRaw)
	if err != nil {
		return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body lease"}
	}
	epoch, ok := environ.CheckUint53Bounds(members["stale_epoch"], 1, scalar.MaxUint53)
	if !ok {
		return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body epoch"}
	}
	diagnostic, ok := environ.CheckDigest(members["diagnostic_evidence_id"])
	if !ok {
		return TerminateBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "terminate body evidence"}
	}
	return TerminateBody{
		Context:              context,
		StaleLeaseID:         lease.String(),
		StaleEpoch:           epoch,
		DiagnosticEvidenceID: diagnostic.String(),
	}, nil
}

// RestoreBody is the closed §4.C restore body: context,
// prior_binding_id, checkpoint_id, and bootstrap_operation_id.
type RestoreBody struct {
	Context              terminstance.MutationContext
	PriorBindingID       string
	CheckpointID         string
	BootstrapOperationID string
}

// restoreBodyMembers is the exact closed member set.
var restoreBodyMembers = []string{
	"context",
	"prior_binding_id",
	"checkpoint_id",
	"bootstrap_operation_id",
}

// parseRestoreBody admits one closed restore body.
func parseRestoreBody(raw []byte) (RestoreBody, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return RestoreBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "restore body frame"}
	}
	if len(members) != len(restoreBodyMembers) {
		return RestoreBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "restore body members"}
	}
	for _, member := range restoreBodyMembers {
		if _, known := members[member]; !known {
			return RestoreBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "restore body members"}
		}
	}
	context, err := terminstance.ParseMutationContext(members["context"])
	if err != nil {
		return RestoreBody{}, err
	}
	prior, ok := environ.CheckDigest(members["prior_binding_id"])
	if !ok {
		return RestoreBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "restore body binding"}
	}
	checkpoint, ok := environ.CheckDigest(members["checkpoint_id"])
	if !ok {
		return RestoreBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "restore body checkpoint"}
	}
	bootstrap, ok := environ.CheckUUIDv7(members["bootstrap_operation_id"])
	if !ok {
		return RestoreBody{}, &Error{Code: terminalbackend.CodeProtocolError, Detail: "restore body bootstrap"}
	}
	return RestoreBody{
		Context:              context,
		PriorBindingID:       prior.String(),
		CheckpointID:         checkpoint.String(),
		BootstrapOperationID: bootstrap.String(),
	}, nil
}

// bodyString reads a JSON string member. Non-string members are
// malformed, never coerced.
func bodyString(raw json.RawMessage) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}
