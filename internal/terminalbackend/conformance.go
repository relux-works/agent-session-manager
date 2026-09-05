// Lifecycle, attach, entrypoint, replication, and historical-translation
// conformance for the TerminalBackend registry.
//
// Normative scope is relux-works/agent-session-manager-spec@v0.5.0 §4.A
// (authority: attach changes no ownership; every backend-created runtime
// executes exactly `ax pane SESSION_ID`), §4.C (lifecycle state machine,
// transition/authorization/idempotency table), §4.D (attach authorization
// neutrality and capability dependencies, enforced by manifest.go), §4.E
// (replication classification and v0.4.3 historical translation), §4.1
// (backend interface operations and the idempotent bootstrap binding), §6.5
// (platform defaults select only new activation), and §7.A (provider-side
// descriptor binding, enforced by CheckProviderDescriptor in
// terminalbackend.go).
//
// Section 4.B names this deliverable: "M0 exposes only an internal semantic
// interface and conformance harness". This file is that harness: it proves
// the semantics over admitted registry state but performs no I/O, launches
// no backend, and holds no authority beyond refusing contract violations.
// Capability gating stays in manifest.go (CheckOperation, Reconcile); trust
// stays in terminalbackend.go (RegisterExternal, DigestFile). A refusal here
// is an *Error with a wire code from the pinned catalog error vocabulary
// and a static Detail clause, following the package convention.
package terminalbackend

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Wire error codes this file may report. Each is a member of the pinned
// catalog error vocabulary; TestWireCodesAreCatalogued proves no other code
// is advertised.
const (
	// CodeProtocolError reports an unknown operation, state, or member, a
	// body variant outside the closed §4.C matrix, or an error code the
	// backend MUST NOT emit for the operation (§4.B, §4.C). AX-local
	// parsing may additionally emit it for a syntactically valid request
	// the matrix does not cover.
	CodeProtocolError = "terminal_backend_protocol_error"
	// CodeUnauthorized reports an expired, mismatched, or otherwise
	// unsatisfiable attach authorization (§4.C attach row).
	CodeUnauthorized = "terminal_backend_unauthorized"
	// CodeUnavailable reports a backend that cannot serve the operation.
	// It appears here only inside the allowed-error sets: no conformance
	// refusal in this file emits it, so it carries no arm of its own.
	CodeUnavailable = "terminal_backend_unavailable"
	// CodePreconditionFailed reports a legal request against an illegal
	// source state, a non-conforming stable entrypoint, or any other
	// unmet local precondition (§4.C per-operation rows, §4.1).
	CodePreconditionFailed = "local_precondition_failed"
	// CodeIdempotencyMismatch reports a changed operation inside an
	// established idempotency window (§4.C, §4.1).
	CodeIdempotencyMismatch = "idempotency_mismatch"
	// CodeIncompatibleSchema reports a legacy identity with no canonical
	// projection, or a canonical identity with no legacy projection
	// (§4.E). It never falls back.
	CodeIncompatibleSchema = "incompatible_schema"
)

// IsProtocolError reports CodeProtocolError refusals.
func IsProtocolError(err error) bool { return errorCode(err, CodeProtocolError) }

// IsUnauthorized reports CodeUnauthorized refusals.
func IsUnauthorized(err error) bool { return errorCode(err, CodeUnauthorized) }

// IsPreconditionFailed reports CodePreconditionFailed refusals.
func IsPreconditionFailed(err error) bool { return errorCode(err, CodePreconditionFailed) }

// IsIdempotencyMismatch reports CodeIdempotencyMismatch refusals.
func IsIdempotencyMismatch(err error) bool { return errorCode(err, CodeIdempotencyMismatch) }

// IsIncompatibleSchema reports CodeIncompatibleSchema refusals.
func IsIncompatibleSchema(err error) bool { return errorCode(err, CodeIncompatibleSchema) }

// InstanceState is the closed §4.C TerminalInstanceState enum: local
// hosting only, never a second ownership state machine.
type InstanceState string

// Terminal Instance states.
const (
	StateAbsent      InstanceState = "absent"
	StateCreating    InstanceState = "creating"
	StateParked      InstanceState = "parked"
	StateActive      InstanceState = "active"
	StateQuiescing   InstanceState = "quiescing"
	StateStopped     InstanceState = "stopped"
	StateStaleFenced InstanceState = "stale_fenced"
	StateUnavailable InstanceState = "unavailable"
)

// ParseInstanceState admits exactly the eight §4.C states.
func ParseInstanceState(value string) (InstanceState, error) {
	switch InstanceState(value) {
	case StateAbsent, StateCreating, StateParked, StateActive,
		StateQuiescing, StateStopped, StateStaleFenced, StateUnavailable:
		return InstanceState(value), nil
	default:
		return "", &Error{Code: CodeProtocolError, Detail: "lifecycle state vocabulary"}
	}
}

// Operation is the closed §4.D TerminalBackendOperation enum.
type Operation string

// Terminal backend operations.
const (
	OperationManifest         Operation = "manifest"
	OperationProbe            Operation = "probe"
	OperationCreate           Operation = "create"
	OperationAttach           Operation = "attach"
	OperationStatus           Operation = "status"
	OperationQuiesceInput     Operation = "quiesce-input"
	OperationWaitSafeBoundary Operation = "wait-safe-boundary"
	OperationRequestStop      Operation = "request-stop"
	OperationTerminateStale   Operation = "terminate-stale"
	OperationRestore          Operation = "restore"
)

// ParseOperation admits exactly the ten §4.D operations.
func ParseOperation(value string) (Operation, error) {
	switch Operation(value) {
	case OperationManifest, OperationProbe, OperationCreate, OperationAttach,
		OperationStatus, OperationQuiesceInput, OperationWaitSafeBoundary,
		OperationRequestStop, OperationTerminateStale, OperationRestore:
		return Operation(value), nil
	default:
		return "", &Error{Code: CodeProtocolError, Detail: "operation vocabulary"}
	}
}

// SideEffect is the closed §4.C TerminalBackendSideEffect enum.
type SideEffect string

// Terminal backend side effects.
const (
	EffectBindingPersisted           SideEffect = "binding_persisted"
	EffectWrapperStarted             SideEffect = "wrapper_started"
	EffectAttachClientCreated        SideEffect = "attach_client_created"
	EffectInputClosed                SideEffect = "input_closed"
	EffectSafeBoundaryObserved       SideEffect = "safe_boundary_observed"
	EffectGracefulStopRequested      SideEffect = "graceful_stop_requested"
	EffectProcessClosed              SideEffect = "process_closed"
	EffectBackendStoreClosed         SideEffect = "backend_store_closed"
	EffectStaleIncarnationTerminated SideEffect = "stale_incarnation_terminated"
	EffectWrapperRestored            SideEffect = "wrapper_restored"
)

// PresentationTransport is the closed §4.E presentation transport enum.
// Only the first two are admittable; a relay-required backend is
// unavailable pending explicit future product/contract approval, so no
// constructor admits it here.
type PresentationTransport string

// Presentation transports.
const (
	TransportLocalOnly          PresentationTransport = "local_only"
	TransportTrustedPrivateMesh PresentationTransport = "trusted_private_mesh"
	TransportThirdPartyRelay    PresentationTransport = "third_party_relay"
)

// parseTransport admits exactly the closed transport vocabulary. The relay
// member parses so a request naming it is classified rather than
// misreported, but CheckAttachRequest refuses it: current AX admits only
// the first two.
func parseTransport(value string) (PresentationTransport, error) {
	switch PresentationTransport(value) {
	case TransportLocalOnly, TransportTrustedPrivateMesh, TransportThirdPartyRelay:
		return PresentationTransport(value), nil
	default:
		return "", &Error{Code: CodeProtocolError, Detail: "presentation transport vocabulary"}
	}
}

// Authorization kinds carried by the transition table. "attach" names the
// ownership-neutral AttachAuthorization below; every other mutating kind
// names an AXAuthorization lease tuple, which this harness does not mint
// (no lease implementation exists here) but does name so the table stays
// exact.
const (
	authorizationNone       = "none"
	authorizationCreate     = "create"
	authorizationAttach     = "attach"
	authorizationControl    = "control"
	authorizationForceStale = "force_stale"
	authorizationRestore    = "restore"
)

// Transition is one exact §4.C table row: the allowed sources, the success
// target, and the exact side effects for one operation.
type Transition struct {
	Operation     Operation
	Sources       []InstanceState
	Target        InstanceState
	TargetParked  bool
	Effects       []SideEffect
	Authorization string
	// InstanceScoped is false for manifest and probe, which carry no
	// Terminal Instance source or target and no side effects.
	InstanceScoped bool
	// SameState marks status: every source is allowed and the success
	// state is the source itself, with no side effect.
	SameState bool
}

// transitionTable is the exact §4.C transition matrix. create carries two
// targets: active when interactive, otherwise parked.
var transitionTable = []Transition{
	{Operation: OperationManifest, InstanceScoped: false, Authorization: authorizationNone},
	{Operation: OperationProbe, InstanceScoped: false, Authorization: authorizationNone},
	{
		Operation: OperationCreate,
		Sources:   []InstanceState{StateAbsent, StateStopped},
		Target:    StateActive, TargetParked: true,
		Effects:        []SideEffect{EffectBindingPersisted, EffectWrapperStarted},
		Authorization:  authorizationCreate,
		InstanceScoped: true,
	},
	{
		Operation:      OperationAttach,
		Sources:        []InstanceState{StateParked, StateActive},
		Target:         StateActive,
		Effects:        []SideEffect{EffectAttachClientCreated},
		Authorization:  authorizationAttach,
		InstanceScoped: true,
	},
	{
		Operation: OperationStatus,
		Sources: []InstanceState{
			StateAbsent, StateCreating, StateParked, StateActive,
			StateQuiescing, StateStopped, StateStaleFenced, StateUnavailable,
		},
		SameState:      true,
		Authorization:  authorizationNone,
		InstanceScoped: true,
	},
	{
		Operation:      OperationQuiesceInput,
		Sources:        []InstanceState{StateActive, StateParked},
		Target:         StateQuiescing,
		Effects:        []SideEffect{EffectInputClosed},
		Authorization:  authorizationControl,
		InstanceScoped: true,
	},
	{
		Operation:      OperationWaitSafeBoundary,
		Sources:        []InstanceState{StateQuiescing},
		Target:         StateQuiescing,
		Effects:        []SideEffect{EffectSafeBoundaryObserved},
		Authorization:  authorizationControl,
		InstanceScoped: true,
	},
	{
		Operation:      OperationRequestStop,
		Sources:        []InstanceState{StateQuiescing},
		Target:         StateStopped,
		Effects:        []SideEffect{EffectGracefulStopRequested, EffectProcessClosed, EffectBackendStoreClosed},
		Authorization:  authorizationControl,
		InstanceScoped: true,
	},
	{
		Operation:      OperationTerminateStale,
		Sources:        []InstanceState{StateStaleFenced, StateUnavailable},
		Target:         StateStopped,
		Effects:        []SideEffect{EffectStaleIncarnationTerminated, EffectProcessClosed},
		Authorization:  authorizationForceStale,
		InstanceScoped: true,
	},
	{
		Operation:      OperationRestore,
		Sources:        []InstanceState{StateAbsent, StateStopped, StateUnavailable},
		Target:         StateParked,
		Effects:        []SideEffect{EffectBindingPersisted, EffectWrapperRestored},
		Authorization:  authorizationRestore,
		InstanceScoped: true,
	},
}

// lookupTransition returns the table row for one operation.
func lookupTransition(operation Operation) (Transition, bool) {
	for _, row := range transitionTable {
		if row.Operation == operation {
			return row, true
		}
	}
	return Transition{}, false
}

// CheckTransition enforces the §4.C transition matrix. It returns the
// success target state and the exact side effects. create resolves its
// target from interactive: active when interactive, otherwise parked. A
// state reached only by AX fencing observation (stale_fenced) is never
// entered here: no backend operation targets it, so a row naming it as a
// target cannot exist and CheckTransition cannot return it except as a
// status observation of the source itself.
//
// An unknown operation or state is a protocol error; a known operation
// against a disallowed source is a failed local precondition. Only
// quiesce-input enters quiescing, and creating is never a success target:
// it exists only between the idempotency receipt and the first side
// effect, which the Ledger below owns.
func CheckTransition(operation, source string, interactive bool) (InstanceState, []SideEffect, error) {
	parsedOperation, err := ParseOperation(operation)
	if err != nil {
		return "", nil, err
	}
	row, known := lookupTransition(parsedOperation)
	if !known {
		return "", nil, &Error{Code: CodeProtocolError, Detail: "operation vocabulary"}
	}
	if !row.InstanceScoped {
		if source != "" {
			return "", nil, &Error{Code: CodePreconditionFailed, Detail: "lifecycle instance scope"}
		}
		return "", nil, nil
	}
	parsedSource, err := ParseInstanceState(source)
	if err != nil {
		return "", nil, err
	}
	allowed := false
	for _, candidate := range row.Sources {
		if candidate == parsedSource {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", nil, &Error{Code: CodePreconditionFailed, Detail: "lifecycle transition"}
	}
	if row.SameState {
		return parsedSource, nil, nil
	}
	if parsedOperation == OperationCreate && !interactive {
		return StateParked, append([]SideEffect(nil), row.Effects...), nil
	}
	return row.Target, append([]SideEffect(nil), row.Effects...), nil
}

// allowedOperationErrors is the exact §4.C "Allowed error codes" column.
// An error not listed for an operation MUST NOT be emitted by the backend
// for a syntactically valid request; AX-local parsing may additionally
// emit terminal_backend_protocol_error, which every row therefore carries.
var allowedOperationErrors = map[Operation][]string{
	OperationManifest: {
		CodeProtocolError, "terminal_backend_protocol_incompatible",
		"terminal_backend_process_failed", "terminal_backend_timeout",
		CodeIntegrityFailure,
	},
	OperationProbe: {
		CodeUntrusted, CodeMismatch, CodeDrift,
		CodeProtocolError, "terminal_backend_process_failed", "terminal_backend_timeout",
		CodeIntegrityFailure,
	},
	OperationCreate: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven, CodeUnavailable,
		"terminal_backend_timeout", "terminal_backend_process_failed",
		CodeIntegrityFailure, CodeProtocolError,
	},
	OperationAttach: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven, CodeUnavailable,
		"terminal_backend_timeout", "terminal_backend_process_failed",
		CodeIntegrityFailure, CodeProtocolError,
	},
	OperationStatus: {
		CodeStaleGeneration, CodeCapabilityUnproven, CodeUnavailable,
		"terminal_backend_timeout", CodeIntegrityFailure, CodeProtocolError,
	},
	OperationQuiesceInput: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven,
		"terminal_backend_timeout", CodeIntegrityFailure, CodeProtocolError,
	},
	OperationWaitSafeBoundary: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven,
		"quiesce_timeout", CodeIntegrityFailure, CodeProtocolError,
	},
	OperationRequestStop: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven,
		"stop_timeout", "terminal_backend_process_failed",
		CodeIntegrityFailure, CodeProtocolError,
	},
	OperationTerminateStale: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven,
		"terminal_backend_timeout", "terminal_backend_process_failed",
		CodeIntegrityFailure, CodeProtocolError,
	},
	OperationRestore: {
		CodeIdempotencyMismatch, CodePreconditionFailed, CodeUnauthorized,
		CodeStaleGeneration, CodeCapabilityUnproven, CodeRestoreMismatch,
		CodeUnavailable, "terminal_backend_timeout", "terminal_backend_process_failed",
		CodeIntegrityFailure, CodeProtocolError,
	},
}

// CheckErrorAllowed enforces the §4.C error mapping: a backend reporting a
// code outside the operation's allowed set violates the contract, and the
// report itself is refused as a protocol error. It names the production
// reporting site, not the underlying failure.
func CheckErrorAllowed(operation, code string) error {
	parsedOperation, err := ParseOperation(operation)
	if err != nil {
		return err
	}
	for _, allowed := range allowedOperationErrors[parsedOperation] {
		if code == allowed {
			return nil
		}
	}
	return &Error{Code: CodeProtocolError, Detail: "operation error vocabulary"}
}

// IdempotencyKey derives the canonical UTF-8 idempotency key for one
// operation from its identity segments (§4.C authorization and idempotency
// column, §4.1 bootstrap binding). The key is never caller-chosen free
// text: the segment count and order are fixed per operation, and every
// segment must be non-empty.
//
// The per-operation forms are: manifest and probe take the request ID
// alone; create and restore take session_id + bootstrap_operation_id;
// attach takes terminal_instance_id + client_id; quiesce-input takes
// terminal_instance_id + "quiesce" + quiescence_generation;
// wait-safe-boundary takes terminal_instance_id + "boundary" +
// quiescence_generation + provider_proof_kind; request-stop takes
// terminal_instance_id + "stop" + safe_boundary_evidence_id;
// terminate-stale takes terminal_instance_id + "terminate" +
// stale_lease_id + decimal stale_epoch; status takes its full queried
// identity tuple, of at least one segment.
func IdempotencyKey(operation string, segments ...string) (string, error) {
	parsedOperation, err := ParseOperation(operation)
	if err != nil {
		return "", err
	}
	for _, segment := range segments {
		if segment == "" {
			return "", &Error{Code: CodeProtocolError, Detail: "idempotency key shape"}
		}
	}
	want := idempotencyKeySegments[parsedOperation]
	if want < 0 {
		if len(segments) < 1 {
			return "", &Error{Code: CodeProtocolError, Detail: "idempotency key shape"}
		}
	} else if len(segments) != want {
		return "", &Error{Code: CodeProtocolError, Detail: "idempotency key shape"}
	}
	return strings.Join(segments, "/"), nil
}

// idempotencyKeySegments is the fixed per-operation segment count. status
// carries its full queried identity tuple, so its count varies and only
// the non-empty floor is enforced.
var idempotencyKeySegments = map[Operation]int{
	OperationManifest: 1, OperationProbe: 1,
	OperationCreate: 2, OperationRestore: 2, OperationAttach: 2,
	OperationQuiesceInput: 3, OperationRequestStop: 3,
	OperationWaitSafeBoundary: 4, OperationTerminateStale: 4,
	OperationStatus: -1,
}

// Receipt is one durably bound idempotency record: the canonical key, the
// operation it was bound for, and the recorded result identity. AX and the
// backend bind the pair before the first child side effect (§4.1); an
// identical retry returns the recorded result instead of acting again.
type Receipt struct {
	Key       string
	Operation Operation
	ResultID  string
}

// Ledger is the process-local idempotency receipt table backing the §4.C
// recovery rules: persist the receipt before the first side effect,
// identical retry replays it, a changed operation in the window is
// idempotency_mismatch, and uncertainty requires status before any start
// retry. The zero value is unusable; construct with NewLedger.
//
// The ledger is the semantic half of durability: Export and Import move
// the table across a controller crash as stable bytes, and Import refuses
// a malformed or ambiguous image instead of inventing an empty table.
type Ledger struct {
	mutex    sync.Mutex
	receipts map[string]Receipt
}

// NewLedger opens an empty receipt table.
func NewLedger() *Ledger {
	return &Ledger{receipts: make(map[string]Receipt)}
}

// Bind records the receipt for a key before the first side effect. A key
// bound for the first time is stored and replayed back. A key bound again
// with the identical operation and result replays the stored receipt: it
// performs no new effect. A key bound again with a different operation or
// result is idempotency_mismatch: only a successful status read proves
// absence, never a second binding.
func (ledger *Ledger) Bind(key string, operation Operation, resultID string) (Receipt, error) {
	if ledger == nil {
		return Receipt{}, &Error{Code: CodeProtocolError, Detail: "idempotency ledger unavailable"}
	}
	if key == "" || resultID == "" {
		return Receipt{}, &Error{Code: CodeProtocolError, Detail: "idempotency key shape"}
	}
	if _, err := ParseOperation(string(operation)); err != nil {
		return Receipt{}, err
	}
	ledger.mutex.Lock()
	defer ledger.mutex.Unlock()
	if stored, known := ledger.receipts[key]; known {
		if stored.Operation != operation || stored.ResultID != resultID {
			return Receipt{}, &Error{Code: CodeIdempotencyMismatch, Detail: "idempotency key conflict"}
		}
		return stored, nil
	}
	receipt := Receipt{Key: key, Operation: operation, ResultID: resultID}
	ledger.receipts[key] = receipt
	return receipt, nil
}

// Replay returns the stored receipt for a key. A lost result replays the
// receipt; absence of a receipt proves nothing and reports false, so the
// caller must consult status before any start retry.
func (ledger *Ledger) Replay(key string) (Receipt, bool) {
	if ledger == nil {
		return Receipt{}, false
	}
	ledger.mutex.Lock()
	defer ledger.mutex.Unlock()
	receipt, known := ledger.receipts[key]
	return receipt, known
}

// Export renders the table as stable bytes: receipts sorted by key, one
// per line as key, operation, and result fields. Import refuses a
// malformed image or one binding the same key twice instead of admitting
// a half-read table.
func (ledger *Ledger) Export() []byte {
	if ledger == nil {
		return nil
	}
	ledger.mutex.Lock()
	defer ledger.mutex.Unlock()
	keys := make([]string, 0, len(ledger.receipts))
	for key := range ledger.receipts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var image strings.Builder
	for _, key := range keys {
		receipt := ledger.receipts[key]
		image.WriteString(key + "\n" + string(receipt.Operation) + "\n" + receipt.ResultID + "\n")
	}
	return []byte(image.String())
}

// AttachAuthorization is the closed, ownership-neutral AX policy object
// of §4.C: exactly policy evidence, authorizing host, transport, input
// boolean, and an expiry strictly after issue. It authorizes only the
// named presentation transport and input boolean. It contains no lease
// member of any kind and therefore cannot acquire, renew, or transfer
// ownership; ParseAttachAuthorization refuses any document carrying one
// as an unknown member before any other check runs.
type AttachAuthorization struct {
	PolicyEvidenceID  string
	AuthorizingHostID string
	Transport         PresentationTransport
	InputAuthorized   bool
	IssuedAt          scalar.Timestamp
	ExpiresAt         scalar.Timestamp
}

// attachAuthorizationMembers is the exact closed member set.
var attachAuthorizationMembers = []string{
	"policy_evidence_id",
	"authorizing_host_id",
	"transport",
	"input_authorized",
	"issued_at",
	"expires_at",
}

// ParseAttachAuthorization admits one closed AttachAuthorization document.
// Unknown members (including any lease, epoch, holder, or authorization
// kind member smuggled into an attach policy) and missing members are
// both malformed; a malformed read is an error, never an absent policy,
// so callers must not fall back to an unauthenticated attach.
func ParseAttachAuthorization(raw []byte) (AttachAuthorization, error) {
	object, err := decodeStrictObject(raw)
	if err != nil {
		return AttachAuthorization{}, err
	}
	if err := checkExactMembers(object, attachAuthorizationMembers); err != nil {
		return AttachAuthorization{}, err
	}
	policyEvidenceID, err := digestMember(object, "policy_evidence_id")
	if err != nil {
		return AttachAuthorization{}, err
	}
	authorizingHostID, err := stringMember(object, "authorizing_host_id")
	if err != nil {
		return AttachAuthorization{}, err
	}
	if _, err := scalar.ParseUUIDv7(authorizingHostID); err != nil {
		return AttachAuthorization{}, mismatchf("document member type")
	}
	transportRaw, err := stringMember(object, "transport")
	if err != nil {
		return AttachAuthorization{}, err
	}
	transport, err := parseTransport(transportRaw)
	if err != nil {
		return AttachAuthorization{}, err
	}
	inputRaw, known := object["input_authorized"]
	if !known {
		return AttachAuthorization{}, mismatchf("document members")
	}
	inputAuthorized, ok := inputRaw.(bool)
	if !ok {
		return AttachAuthorization{}, mismatchf("document member type")
	}
	issuedAt, err := timestampMember(object, "issued_at")
	if err != nil {
		return AttachAuthorization{}, err
	}
	expiresAt, err := timestampMember(object, "expires_at")
	if err != nil {
		return AttachAuthorization{}, err
	}
	issued, err := issuedAt.Time()
	if err != nil {
		return AttachAuthorization{}, mismatchf("document timestamp")
	}
	expires, err := expiresAt.Time()
	if err != nil {
		return AttachAuthorization{}, mismatchf("document timestamp")
	}
	if !expires.After(issued) {
		return AttachAuthorization{}, &Error{Code: CodeUnauthorized, Detail: "attach authorization expiry"}
	}
	return AttachAuthorization{
		PolicyEvidenceID:  policyEvidenceID,
		AuthorizingHostID: authorizingHostID,
		Transport:         transport,
		InputAuthorized:   inputAuthorized,
		IssuedAt:          issuedAt,
		ExpiresAt:         expiresAt,
	}, nil
}

// CheckAttachRequest authorizes one attach request against its policy at
// the given instant. The request transport and input boolean must equal
// the authorized ones; a relay transport is refused even when authorized
// on paper, because current AX admits only local and trusted-mesh
// presentation (§4.E). Expiry is checked against now: an expired policy
// authorizes nothing, and a lease-shaped bypass cannot exist because the
// policy has no lease member to consult.
func CheckAttachRequest(auth AttachAuthorization, transport string, inputAuthorized bool, now time.Time) error {
	requested, err := parseTransport(transport)
	if err != nil {
		return err
	}
	if requested != auth.Transport || inputAuthorized != auth.InputAuthorized {
		return &Error{Code: CodeUnauthorized, Detail: "attach authorization binding"}
	}
	if requested == TransportThirdPartyRelay {
		return &Error{Code: CodeUnauthorized, Detail: "attach relay transport"}
	}
	expires, err := auth.ExpiresAt.Time()
	if err != nil {
		return mismatchf("document timestamp")
	}
	if !now.Before(expires) {
		return &Error{Code: CodeUnauthorized, Detail: "attach authorization expiry"}
	}
	return nil
}

// CheckAttachResult enforces the §4.C attach result rule: the result
// input boolean MUST equal both the request and the AttachAuthorization.
// A backend reporting input the policy did not authorize is refused even
// when the request itself was authorized.
func CheckAttachResult(requestInputAuthorized, resultInputAuthorized bool, auth AttachAuthorization) error {
	if resultInputAuthorized != requestInputAuthorized || resultInputAuthorized != auth.InputAuthorized {
		return &Error{Code: CodeUnauthorized, Detail: "attach input binding"}
	}
	return nil
}

// CheckEntrypoint enforces the §4.A stable entrypoint rule: every
// backend-created runtime ultimately executes exactly
// `ax pane SESSION_ID`, and a raw provider command is forbidden as a
// durable entry point. argv must be exactly the three literals ax, pane,
// and the canonical request Session UUIDv7 string, and the carried
// session must equal the session under creation.
func CheckEntrypoint(argv []string, sessionID string) error {
	if len(argv) != 3 || argv[0] != "ax" || argv[1] != "pane" {
		return &Error{Code: CodePreconditionFailed, Detail: "entrypoint argv"}
	}
	session, err := scalar.ParseUUIDv7(sessionID)
	if err != nil {
		return &Error{Code: CodePreconditionFailed, Detail: "entrypoint session binding"}
	}
	carried, err := scalar.ParseUUIDv7(argv[2])
	if err != nil {
		return &Error{Code: CodePreconditionFailed, Detail: "entrypoint session binding"}
	}
	if carried.String() != session.String() {
		return &Error{Code: CodePreconditionFailed, Detail: "entrypoint session binding"}
	}
	return nil
}

// StatusResult is the §4.C status success body in harness form. Pointer
// members distinguish null from false: provider observation is null
// unless requested and evidenced, and the last-operation fields are null
// when the backend reports no prior operation.
type StatusResult struct {
	State             InstanceState
	IdentityMatch     bool
	WrapperPresent    bool
	ProviderPresent   *bool
	Attachable        bool
	LastOperationID   *string
	LastEffect        *SideEffect
	ProviderRequested bool
	ProviderEvidenced bool
	AttachEvidenced   bool
}

// CheckStatusResult enforces the §4.C exact-instance lookup rules. A
// non-match sets identity_match false, state absent, wrapper absent,
// provider null, attachable false, and both last fields null, and does
// not authorize fallback: any deviation from that canonical false form
// is refused. provider_present is null unless provider observation was
// requested and evidenced. attachable is true only in parked or active
// with a currently evidenced matching attach capability. A timeout or
// read failure is unknown, never absent: this check validates reported
// results and never synthesizes absence from a missing read.
func CheckStatusResult(identityMatch bool, result StatusResult) error {
	if _, err := ParseInstanceState(string(result.State)); err != nil {
		return err
	}
	if result.LastEffect != nil {
		if _, err := ParseSideEffect(string(*result.LastEffect)); err != nil {
			return err
		}
	}
	if !identityMatch || !result.IdentityMatch {
		if result.IdentityMatch {
			return &Error{Code: CodeProtocolError, Detail: "status identity binding"}
		}
		if result.State != StateAbsent || result.WrapperPresent ||
			result.ProviderPresent != nil || result.Attachable ||
			result.LastOperationID != nil || result.LastEffect != nil {
			return &Error{Code: CodeProtocolError, Detail: "status identity binding"}
		}
		return nil
	}
	if result.ProviderPresent != nil && !(result.ProviderRequested && result.ProviderEvidenced) {
		return &Error{Code: CodeProtocolError, Detail: "status provider observation"}
	}
	if result.Attachable && !((result.State == StateParked || result.State == StateActive) && result.AttachEvidenced) {
		return &Error{Code: CodePreconditionFailed, Detail: "status attachability"}
	}
	return nil
}

// ParseSideEffect admits exactly the ten §4.C side effects.
func ParseSideEffect(value string) (SideEffect, error) {
	switch SideEffect(value) {
	case EffectBindingPersisted, EffectWrapperStarted, EffectAttachClientCreated,
		EffectInputClosed, EffectSafeBoundaryObserved, EffectGracefulStopRequested,
		EffectProcessClosed, EffectBackendStoreClosed,
		EffectStaleIncarnationTerminated, EffectWrapperRestored:
		return SideEffect(value), nil
	default:
		return "", &Error{Code: CodeProtocolError, Detail: "side effect vocabulary"}
	}
}

// ReplicationClass is the §4.E persistence classification. Only
// safe_evidence MAY persist and replicate after validation; every other
// class is host-local, rebuildable, owner-only, or forbidden.
type ReplicationClass string

// Replication classes.
const (
	ReplicationSafeEvidence       ReplicationClass = "safe_evidence"
	ReplicationHostLocal          ReplicationClass = "host_local"
	ReplicationDerivedCache       ReplicationClass = "derived_cache"
	ReplicationSensitiveOwnerOnly ReplicationClass = "sensitive_owner_only"
	ReplicationForbidden          ReplicationClass = "forbidden"
)

// replicationMembers is the closed §4.E classification table. Member
// names are wire vocabulary, never local data: classifying a member the
// table does not know is refused rather than defaulted.
var replicationMembers = buildReplicationMembers()

// buildReplicationMembers assembles the closed §4.E classification table.
// Member names are wire vocabulary, never local data: classifying a member
// the table does not know is refused rather than defaulted.
func buildReplicationMembers() map[string]ReplicationClass {
	// Safe immutable evidence: sanitized backend identity, versions,
	// platform, conformance, and capability evidence MAY persist and
	// replicate after validation.
	safe := []string{
		"manifest_id", "probe_id", "evidence_id",
		"terminal_backend_id", "implementation_version",
		"protocol_version", "protocol_versions",
		"platform", "platforms", "os_version",
		"conformance_fixture_id", "capability_claims", "evidence_ids",
		"backend_generation_digest", "capability", "dependent_operations",
		"evidence_requirements", "facts", "issuer", "issuer_id",
		"observed_at", "expires_at",
	}
	// Host-local durable metadata: instance/binding IDs, sanitized native
	// reference, generation, idempotency receipt, last effect.
	hostLocal := []string{
		"binding_id", "terminal_instance_id", "session_id",
		"host_id", "host_incarnation_id", "backend_generation",
		"native_reference", "idempotency_receipt", "last_effect",
		"last_operation_id", "supersedes_binding_id", "created_at",
	}
	// Derived cache: discovery, availability, attachability, and process
	// observations with probe views; rebuildable, non-authoritative.
	derived := []string{
		"availability", "attachable", "wrapper_present", "provider_present",
		"identity_match", "state", "probed_at",
	}
	// Sensitive runtime state: IPC, sockets, pipes, attach/relay
	// credentials, backend auth and live databases, GUI/login
	// attestations, provider credentials; owner-only, non-replicable.
	sensitive := []string{
		"attach_descriptor", "ipc_handle", "tmux_socket",
		"named_pipe", "attach_credential", "relay_credential",
		"backend_auth", "auth_database", "gui_attestation",
		"login_attestation", "provider_credential", "evidence_secret",
		"environment_values",
	}
	// Forbidden: raw native reference, PID/handle, endpoint/token,
	// terminal output/scrollback, unrestricted environment, credential
	// detail, and live-process facts MUST NOT replicate.
	forbidden := []string{
		"native_pid", "process_handle", "endpoint", "token",
		"terminal_output", "scrollback", "unrestricted_environment",
		"credential_detail", "live_process_fact", "raw_native_reference",
	}
	table := make(map[string]ReplicationClass,
		len(safe)+len(hostLocal)+len(derived)+len(sensitive)+len(forbidden))
	for _, member := range safe {
		table[member] = ReplicationSafeEvidence
	}
	for _, member := range hostLocal {
		table[member] = ReplicationHostLocal
	}
	for _, member := range derived {
		table[member] = ReplicationDerivedCache
	}
	for _, member := range sensitive {
		table[member] = ReplicationSensitiveOwnerOnly
	}
	for _, member := range forbidden {
		table[member] = ReplicationForbidden
	}
	return table
}

// Import rebuilds a ledger from Export bytes. A truncated, malformed, or
// ambiguous image is an error, never an empty table, so a failed read
// cannot masquerade as proven absence. An empty image rebuilds the empty
// table, which is exactly what Export of a fresh ledger produces.
func ImportLedger(image []byte) (*Ledger, error) {
	lines := strings.Split(string(image), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines)%3 != 0 {
		return nil, &Error{Code: CodeProtocolError, Detail: "idempotency ledger image"}
	}
	ledger := NewLedger()
	for index := 0; index < len(lines); index += 3 {
		key, operation, resultID := lines[index], lines[index+1], lines[index+2]
		if key == "" || resultID == "" {
			return nil, &Error{Code: CodeProtocolError, Detail: "idempotency ledger image"}
		}
		parsed, err := ParseOperation(operation)
		if err != nil {
			return nil, &Error{Code: CodeProtocolError, Detail: "idempotency ledger image"}
		}
		if _, duplicate := ledger.receipts[key]; duplicate {
			return nil, &Error{Code: CodeIdempotencyMismatch, Detail: "idempotency ledger image"}
		}
		ledger.receipts[key] = Receipt{Key: key, Operation: parsed, ResultID: resultID}
	}
	return ledger, nil
}

// ClassifyReplication reports the §4.E persistence class of one wire
// member. An unlisted member reports false: the table is closed, and an
// unknown member is refused rather than defaulted into any class.
func ClassifyReplication(member string) (ReplicationClass, bool) {
	class, known := replicationMembers[member]
	return class, known
}

// CheckReplicable refuses a replication set carrying any member outside
// the safe immutable evidence class, or any member the closed table does
// not know. Host-local metadata, derived cache, sensitive runtime state,
// and forbidden members MUST NOT replicate; unknown members fail the
// same way, because an unclassified member replicating by default would
// silently widen the §4.E exclusion. An empty set replicates nothing and
// is admitted. The refusal is a protocol error: the set is a body
// variant the replication contract does not cover.
func CheckReplicable(members []string) error {
	for _, member := range members {
		if class, known := replicationMembers[member]; !known || class != ReplicationSafeEvidence {
			return &Error{Code: CodeProtocolError, Detail: "replication exclusion"}
		}
	}
	return nil
}

// LegacyUnreported is the version and generation a historical v0.4.3
// translation carries. The past reported no versions and no generation,
// so the translation says exactly that instead of inferring values no
// observation supports.
const LegacyUnreported = "legacy_unreported"

// LegacyBinding is the §4.E forward translation of one historical v0.4.3
// backend identity: the canonical backend ID with unreported version and
// generation. It carries no capability field of any kind: a translation
// MUST NOT infer capabilities the historical record never evidenced, and
// a struct that cannot name them cannot smuggle them. The field count is
// pinned by TestLegacyBindingCarriesNoCapabilities.
type LegacyBinding struct {
	BackendID             string
	ImplementationVersion string
	Generation            string
}

// legacyForward is the exact §4.E forward map. Only the two immutable
// v0.4.3 values translate; anything else is incompatible, never fallback.
var legacyForward = map[string]string{
	"tmux":   BuiltinTmux,
	"conpty": BuiltinConpty,
}

// TranslateLegacyBackend deterministically maps a historical v0.4.3
// backend name to its canonical binding. Version and generation are
// legacy_unreported and no capabilities are inferred. Translation is a
// pure function over its argument: it never rewrites or re-digests
// history, and the input is unchanged by construction.
func TranslateLegacyBackend(legacy string) (LegacyBinding, error) {
	canonical, known := legacyForward[legacy]
	if !known {
		return LegacyBinding{}, &Error{Code: CodeIncompatibleSchema, Detail: "legacy backend identity"}
	}
	return LegacyBinding{
		BackendID:             canonical,
		ImplementationVersion: LegacyUnreported,
		Generation:            LegacyUnreported,
	}, nil
}

// ProjectToLegacy is the §4.E reverse projection. It exists only for the
// two canonical built-ins; every other backend ID, including a
// well-formed third-party ID, returns incompatible_schema, never a
// fallback. A malformed identity is not an ID at all and is refused by
// the registry grammar first.
func ProjectToLegacy(backendID string) (string, error) {
	if _, err := ParseID(backendID); err != nil {
		return "", err
	}
	for legacy, canonical := range legacyForward {
		if backendID == canonical {
			return legacy, nil
		}
	}
	return "", &Error{Code: CodeIncompatibleSchema, Detail: "legacy reverse projection"}
}
