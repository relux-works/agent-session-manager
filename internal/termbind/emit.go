package termbind

import (
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Terminal event types and schema version from Section 5.2. The literals
// are asserted at the emission entries, not referenced from the axpane
// owner, so a drift between the two fails here.
const (
	eventTerminalCreated = "terminal.created"
	eventSessionResumed  = "session.resumed"
	eventSchemaV4        = "4.0.0"
)

// EvidenceAdmission carries the local evidence admission inputs: the
// backend registry, the host-local evidence universe, the host-local raw
// backend generation (never written to the event), the clock, and the
// caller-supplied signature verifier.
type EvidenceAdmission struct {
	Registry      *terminalbackend.Registry
	Universe      EvidenceUniverse
	RawGeneration string
	Now           time.Time
	Verify        terminalbackend.SignatureVerifier
}

// EmitTerminalCreated authors one Session Event 4.0.0 terminal.created
// event through the sessrepo owner under a lease the local host holds. The
// evidence IDs resolve first through the landed admission bound to the
// payload's own backend tuple; only an admitted backend authors. The
// payload is built by the axpane owner and appended by the axpane Emit
// entry, which gates through fencing.AuthorizeMutation and the canonical
// closed-shape boundary. The binding digest travels as the opaque audit
// reference: the entry takes no binding object, generation string, or
// native reference, so none can reach the event bytes.
func EmitTerminalCreated(repository *sessrepo.Repository, params axpane.EmitParams, facts axpane.TerminalBindingFacts, admission EvidenceAdmission) (sessrepo.EventRef, []byte, ResolvedEvidence, error) {
	empty := sessrepo.EventRef{}
	resolved, err := ResolveEvidence(facts.EvidenceIDs, tupleFromFacts(facts), admission.Registry, admission.Universe, admission.RawGeneration, admission.Now, admission.Verify)
	if err != nil {
		return empty, nil, ResolvedEvidence{}, err
	}
	payload, err := axpane.TerminalCreatedPayload(facts)
	if err != nil {
		return empty, nil, ResolvedEvidence{}, err
	}
	params.EventType = eventTerminalCreated
	params.SchemaVersion = eventSchemaV4
	params.Payload = payload
	reference, raw, err := axpane.Emit(repository, params)
	if err != nil {
		return empty, nil, ResolvedEvidence{}, err
	}
	return reference, raw, resolved, nil
}

// EmitResumed authors one Session Event 4.0.0 session.resumed event through
// the same admission and lease authority as EmitTerminalCreated. The
// checkpoint, execution profile, and nullable profile source arrive from
// the caller (the effective Section 2.4 pair the decision carries); the
// terminal binding and evidence members resolve and append exactly as the
// created event does.
//
// Stated bound: EmitResumed binds neither checkpoint_id to the chain nor
// the (profile, source) pair to the referenced checkpoint's closure via
// sessprofile.CheckResumedPair. Through the wrapper decision both values
// come from the fold, so the bound is unreachable on the composed path;
// a direct caller is trusted and can author a newest the chain never
// published. TestEmitResumedCheckpointBindingIsCallerBound pins the
// boundary: a fabricated checkpoint appends and the landed fold reports
// it newest.
func EmitResumed(repository *sessrepo.Repository, params axpane.EmitParams, checkpointID, profile, source string, hasSource bool, facts axpane.TerminalBindingFacts, admission EvidenceAdmission) (sessrepo.EventRef, []byte, ResolvedEvidence, error) {
	empty := sessrepo.EventRef{}
	resolved, err := ResolveEvidence(facts.EvidenceIDs, tupleFromFacts(facts), admission.Registry, admission.Universe, admission.RawGeneration, admission.Now, admission.Verify)
	if err != nil {
		return empty, nil, ResolvedEvidence{}, err
	}
	payload, err := axpane.ResumedPayload(checkpointID, profile, source, hasSource, facts)
	if err != nil {
		return empty, nil, ResolvedEvidence{}, err
	}
	params.EventType = eventSessionResumed
	params.SchemaVersion = eventSchemaV4
	params.Payload = payload
	reference, raw, err := axpane.Emit(repository, params)
	if err != nil {
		return empty, nil, ResolvedEvidence{}, err
	}
	return reference, raw, resolved, nil
}

// tupleFromFacts derives the event tuple from the payload's own backend
// members, so resolution always binds the resolved objects to the tuple
// the emitted event carries. No second tuple is expressible.
func tupleFromFacts(facts axpane.TerminalBindingFacts) EventTuple {
	return EventTuple{
		BackendID:             facts.BackendID,
		ImplementationVersion: facts.ImplementationVersion,
		ProtocolVersion:       facts.ProtocolVersion,
	}
}
