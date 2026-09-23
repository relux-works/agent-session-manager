package axpane

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/resumesmoke"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessprofile"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Action is exactly one wrapper outcome. Decide returns one and only
// one; there is no combined or fallback outcome.
type Action string

const (
	// ActionLaunch starts the provider under the winning lease with
	// the validated descriptor and fencing token.
	ActionLaunch Action = "launch"
	// ActionReattach reattaches to the one recorded wrapper/child
	// after a lost result: the bootstrap pair matches the durable
	// binding and every gate passes again.
	ActionReattach Action = "reattach"
	// ActionAttachRemote offers remote attach: another host owns the
	// session in an interactive terminal and the attach capability
	// is admitted. It authorizes no lease mutation.
	ActionAttachRemote Action = "attach_remote"
	// ActionTakeoverOffer offers takeover: another host owns the
	// session in an interactive terminal but remote attach is not
	// available (attach capability unproven). It starts no runtime.
	// A remote owner outside an interactive terminal parks instead
	// (§4.2 step 5: parked in all other cases).
	ActionTakeoverOffer Action = "takeover_offer"
	// ActionParked parks without launching the provider, carrying
	// the Section 5.2 session.parked reason vocabulary.
	ActionParked Action = "parked"
	// ActionRefused refuses: a static validation failed and there is
	// no parked direction that a state change could resolve.
	ActionRefused Action = "refused"
)

// Refusal classes are the Section 15 codes the wrapper surfaces.
// Landed-gate codes pass through verbatim; the wrapper mints no code.
const (
	// ClassInvalidConfig reports a configuration that fails
	// validation (config.Load through ValidateConfig).
	ClassInvalidConfig = "invalid_config"
	// ClassInvalidArguments reports a malformed wrapper call: an
	// unknown SESSION_ID, an unknown caller kind or mode, or a
	// malformed presented fencing token.
	ClassInvalidArguments = "invalid_arguments"
	// ClassIntegrityFailure reports profile derivation input the
	// reducer refuses: the chain cannot be trusted.
	ClassIntegrityFailure = "integrity_failure"
	// ClassProfileMappingUnavailable is the Section 2.4 resume
	// failure: the adapter cannot map the stored profile for the
	// probed provider version.
	ClassProfileMappingUnavailable = "profile_mapping_unavailable"
)

// Mode selects the wrapper path: a fresh activation (the Section 4.C
// create row) or the after-restore first start (the restore row). It
// selects both the fencing entry and the backend capability the
// admitted set must confer.
type Mode string

const (
	// ModeLaunch gates through fencing AuthorizeActivation and the
	// create capability dependencies.
	ModeLaunch Mode = "launch"
	// ModeRestore gates through fencing AuthorizeRestore and the
	// restore capability dependencies.
	ModeRestore Mode = "restore"
)

// Caller kinds are the closed §4.2 realm vocabulary.
const (
	// CallerForeground is an interactive operator-side caller.
	CallerForeground = "foreground"
	// CallerBackground is a background CLI, SSH RPC process, daemon,
	// or restore worker: it must never create a
	// credential-dependent tmux server.
	CallerBackground = "background"
)

// Realm carries the §4.2 caller realm and broker readiness the
// credential-conditional arm decides over. There is deliberately no
// member for a cached sentinel result, a managername observation,
// or any attested-server boolean: none authorizes resume, so none
// is an input. A background caller, and any caller on a path that
// requires provider credentials, authorizes only through the
// admitted credential_capable_execution_realm Capability Evidence
// row bound to the exact host binding, probed provider build, and
// current generation (§4.C conditional rows, §4.2, §4.D); the bare
// boolean previously carried here never authorizes and is removed.
type Realm struct {
	// Caller is CallerForeground or CallerBackground.
	Caller string
	// BrokerState, ServerGeneration, and Remediation are the typed
	// realm/readiness details the capability_unavailable refusal
	// carries. They are required: an empty member refuses
	// invalid_arguments instead of emitting an untyped refusal.
	// ServerGeneration must equal the current AX-known raw
	// generation the backend admission binds to; a stale generation
	// refuses capability_unavailable, never launches.
	BrokerState      string
	ServerGeneration string
	Remediation      string
}

// Closed capability names the wrapper's conditional arms evaluate.
// They repeat the §4.D registry rows the backend admission proves;
// the wrapper invents no capability.
const (
	// credentialRealmCapability is the §4.D row a background caller
	// must hold: the exact-instance sentinel plus provider-auth
	// smoke bound to the terminal binding, provider build,
	// generation digest, OS version, and expiry.
	credentialRealmCapability = "credential_capable_execution_realm"
	// headlessCreationCapability is the §4.C create-row conditional
	// a non-interactive create must hold.
	headlessCreationCapability = "headless_creation"
)

// ProviderFacts carries the probed provider build, the stored
// identity, and the optional discovery proof the provider-identity
// arm decides over.
type ProviderFacts struct {
	// Build is the exact probed build tuple.
	Build provhost.BuildTuple
	// Identity is the stored Provider Identity Record document.
	Identity []byte
	// Discovery is the NativeDiscoveryProof document, meaningful
	// only when HasDiscovery is set.
	Discovery    []byte
	HasDiscovery bool
	// DiscoveryContext binds the proof to host-side facts.
	DiscoveryContext provhost.DiscoveryContext
}

// SmokeFacts carries the provider-auth smoke precondition: the
// stored smoke record plus the typed target-auth details the
// target_auth_missing refusal carries when the smoke is required
// and fails.
type SmokeFacts struct {
	// Required reports that provider credentials are required, so
	// the smoke must pass (the §4.C create/restore rows).
	Required bool
	// Record is the stored smoke record document.
	Record []byte
	// Target carries the typed provider/build/version/generation
	// details. Every member is required when the smoke is
	// required: an empty member refuses invalid_arguments instead
	// of emitting an untyped refusal.
	Target axerror.TargetAuth
	// MacOSVersion is recorded here because TargetAuth carries it;
	// it is the version the smoke evidence binds, not a fresh
	// observation.
}

// BackendFacts carries the backend identity documents the §4.B arm
// decides over.
type BackendFacts struct {
	// Manifest, Probe, and Evidence are the raw documents parsed by
	// the terminalbackend owner.
	Manifest []byte
	Probe    []byte
	Evidence [][]byte
	// RawGeneration is the AX-known raw generation the admission
	// binds to; Verify checks evidence signatures.
	RawGeneration string
	Verify        terminalbackend.SignatureVerifier
	// Now is the admission instant for evidence liveness.
	Now time.Time
}

// Input is the complete Decide surface: validated configuration,
// the loaded session, the local lease view, materialization and
// checkpoint state, the profile chain, provider identity, backend
// identity, realm readiness, and the bootstrap binding. I/O-free
// callers build it by hand; Run builds it through the gates.go
// adapters over the durable stores.
type Input struct {
	// SessionID is the logical session under decision.
	SessionID string
	// BootstrapOperationID is the caller-stable bootstrap operation.
	BootstrapOperationID string
	// Mode selects the fencing entry and capability row.
	Mode Mode

	// ConfigErr is the ValidateConfig outcome; non-nil refuses
	// invalid_config.
	ConfigErr error
	// SessionKnown reports that the logical session loaded; false
	// refuses invalid_arguments.
	SessionKnown bool

	// ExistingBinding is the proven bootstrap receipt for THIS
	// (session_id, bootstrap_operation_id) pair, or nil when the
	// pair is unrecorded. A recorded pair reattaches; an
	// unrecorded pair on a bound session inside the bootstrap
	// window refuses idempotency_mismatch; an unrecorded pair
	// after the window closed installs a new receipt and
	// launches. Superseded pairs keep their receipts, so an
	// identical retry of any recorded pair replays its own
	// receipt even after later pairs superseded it.
	ExistingBinding *Binding
	// SessionBound reports that the session has any recorded
	// binding (the first-binding anchor or any pair receipt).
	// Together with the window predicate it drives the
	// changed-operation arm: an unrecorded pair on a bound
	// session inside the window refuses idempotency_mismatch,
	// while an unbound session binds fresh.
	SessionBound bool
	// HasNewestCheckpoint and NewestCheckpointID carry the
	// landed chain fold's newest checkpoint (sessstate.Reduce
	// over the authoritative chain: Projection.HasCheckpoint
	// and Newest.ID from checkpoint.created, checkpointed
	// session.stopped, and session.resumed). This is the
	// authoritative "newest checkpoint" (§5.7
	// newest_checkpoint_id, §14.4): the window predicate,
	// checkpoint admission, and journal source binding all read
	// this fact — never the lease record's own checkpoint,
	// which is null for every epoch-1 owner. Run loads it
	// through the fold adapter; I/O-free callers supply it by
	// hand. A successor lease's Checkpoint is the handoff base
	// at takeover time and never moves, so it is never
	// compared against this fact: the fold is the sole newest
	// authority, and the only implication the winner's
	// checkpoint carries — a checkpoint-carrying winner means
	// the chain published a newest — is asserted once, directly
	// after the fencing arm.
	HasNewestCheckpoint bool
	NewestCheckpointID  string

	// Presented is the local fencing token; Observation is the
	// verified lease view, built by fencing.Observe.
	Presented   fencing.PresentedToken
	Observation fencing.Observation

	// Backend carries the §4.B identity documents.
	Backend BackendFacts

	// Journal is the loaded materialization journal; JournalOK
	// reports that the matjournal owner admitted it. When
	// MaterializationRequired is set (the after-restore step 3),
	// a local launch additionally requires the committed phase
	// sourced from the fold's newest published checkpoint. A
	// fresh launch (§13.1 steps 3-4) runs before the journal
	// commits and leaves the requirement unset.
	MaterializationRequired bool
	Journal                 matjournal.Journal
	JournalOK               bool

	// Checkpoint carries the resume checkpoint: Required when the
	// path resumes from a checkpoint, Doc with its bytes, ID with
	// the expected digest. Admission attests through the sessrepo
	// owner, requires exactly the fold's newest published
	// checkpoint, and binds the checkpoint's own session_id to
	// this session. The winning lease's handoff base is never
	// compared here: the fold is the sole newest authority.
	CheckpointRequired bool
	CheckpointDoc      []byte
	CheckpointID       string

	// ProfileData carries the decoded session surface together with
	// the durable winning-lease and handoff-closure facts used by
	// sessprofile to select an effective source. ProfileHeads, when
	// non-empty, further narrows derivation to the checkpoint actually
	// resumed; it never widens the lease-authorized source set.
	ProfileData  sessprofile.Derivation
	ProfileHeads []string

	// Provider carries the probed build, stored identity, and
	// optional discovery proof.
	Provider ProviderFacts

	// Smoke carries the provider-auth smoke precondition.
	Smoke SmokeFacts

	// Realm carries the caller realm and broker readiness.
	Realm Realm

	// Entrypoint is the wrapper argv, checked against the §4.A
	// stable entrypoint rule.
	Entrypoint []string

	// Descriptor is the §7.A provider descriptor document the
	// wrapper supplies; HostBinding is the AX-validated host-local
	// binding it must match. Admission runs on the launch and
	// reattach paths only: parked, refused, and remote-offer
	// outcomes start no local provider.
	Descriptor  []byte
	HostBinding terminalbackend.InstanceBinding

	// RemoteInteractive reports that the remote owner holds the
	// session in an interactive terminal (after-restore step 4).
	RemoteInteractive bool

	// Interactive reports that the local pane runs in an
	// interactive terminal. A non-interactive create must hold the
	// headless_creation conditional (§4.C create row); the field
	// selects that arm and shapes the supplied descriptor.
	Interactive bool
}

// Decision is the one outcome Decide returns.
type Decision struct {
	// Action is the outcome.
	Action Action
	// ParkReason and WinningLeaseID carry the session.parked
	// vocabulary when Action is parked. The lease ID is empty when
	// no winner is established.
	ParkReason     fencing.ParkReason
	WinningLeaseID string
	// Class is the Section 15 refusal code when Action is refused.
	Class string
	// Detail is the static refusal or park detail. It carries
	// member names and the landed cause, never credentials,
	// generations, or native references.
	Detail string
	// Cause is the underlying landed error.
	Cause error
	// Token is the sealed fencing token the passed gate minted,
	// set on launch and reattach only. The provider receives it
	// separately from the descriptor (§7.A).
	Token fencing.LeaseToken
	// HasToken reports that Token is minted. The zero token never
	// authorizes: decisions without a passed gate carry none.
	HasToken bool
	// Profile is the effective persisted pair the launch carries.
	Profile sessprofile.Pair
	// Mapping is the resolved provider adapter flag (empty for
	// standard).
	Mapping string
	// Admitted is the reconciled backend capability set.
	Admitted terminalbackend.Admitted
	// Descriptor is the admitted §7.A provider descriptor, set on
	// launch and reattach only. On reattach the recorded Binding
	// terminal instance ID is authoritative: the descriptor names the
	// request's own freshly minted instance, never the recorded child,
	// so no caller may target the descriptor instance on reattach.
	Descriptor terminalbackend.ProviderDescriptor
	// RealmFailure is the composed typed Structured Error for the
	// capability_unavailable and target_auth_missing refusals.
	RealmFailure *axerror.Error
}

func refused(class, detail string, cause error) Decision {
	return Decision{Action: ActionRefused, Class: class, Detail: detail, Cause: cause}
}

func parked(reason fencing.ParkReason, winningLeaseID string, cause error) Decision {
	return Decision{
		Action:         ActionParked,
		ParkReason:     reason,
		WinningLeaseID: winningLeaseID,
		Detail:         "park (" + string(reason) + ") without launching the provider",
		Cause:          cause,
	}
}

// Decide is the pure wrapper core: configuration, session, bootstrap
// idempotency with the window predicate, backend identity and
// capability admission with the §4.C conditionals, fencing, the
// successor-lease null-fold arm, materialization with the
// fold-newest source binding, checkpoint admission bound to the
// fold newest and session, provider identity, provider-auth smoke
// with the target binding, realm evidence, profile derivation from
// the checkpoint closure on resume and mapping, entrypoint, and
// provider descriptor, in that precedence, then the outcome
// selection. Fencing precedes materialization: a remote owner
// offers attach/takeover even when the local materialization is
// stale (the after-restore order). The tuple gate precedes mapping
// so a tuple refusal keeps its own class instead of wearing
// profile_mapping_unavailable. Decide performs no I/O, reads no
// clock except the instants the input carries, and mints no lease,
// binding, or event: Run owns the durable effects. Every refusal
// carries the Section 15 code the arm names; every park carries
// the session.parked vocabulary.
func Decide(input Input) Decision {
	if input.ConfigErr != nil {
		return refused(ClassInvalidConfig, "configuration is invalid", input.ConfigErr)
	}
	if input.Mode != ModeLaunch && input.Mode != ModeRestore {
		return refused(ClassInvalidArguments, "wrapper mode is not launch or restore", errWrapperMode(input.Mode))
	}
	if !input.SessionKnown {
		return refused(ClassInvalidArguments, "logical session is unknown", errUnknownSession(input.SessionID))
	}
	if (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) {
		return refused(terminalbackend.CodeIdempotencyMismatch,
			"bootstrap operation changed inside the bootstrap window",
			errIdempotencyChanged(input.SessionID))
	}
	if err := checkRealmVocabulary(input.Realm.Caller); err != nil {
		return refused(ClassInvalidArguments, "caller realm is not foreground or background", err)
	}

	admitted, evidence, err := admitBackend(input)
	if err != nil {
		var malformed errInvalidArguments
		if errors.As(err, &malformed) {
			return refused(ClassInvalidArguments, "caller realm details are incomplete", err)
		}
		var typed *axerror.Error
		if errors.As(err, &typed) {
			decision := refused(string(typed.Code()), "caller has no usable realm evidence", err)
			decision.RealmFailure = typed
			return decision
		}
		return refused(backendClass(err), "terminal backend identity or capability admission failed", err)
	}
	if input.Mode == ModeLaunch && !input.Interactive && !admitted.Has(headlessCreationCapability) {
		return refused(terminalbackend.CodeCapabilityUnproven,
			"non-interactive create requires headless_creation",
			errHeadlessRequired())
	}

	token, err := authorize(input)
	if err != nil {
		if fencing.IsParked(err) {
			reason, winning, _ := fencing.ParkDetails(err)
			return decideParked(input, admitted, reason, winning, err)
		}
		return refused(fencingClass(err), "fencing refused the wrapper", err)
	}

	// A checkpoint-carrying winner implies the fold has a
	// published newest: the successor lease's checkpoint is the
	// handoff base at takeover time and never moves, while every
	// checkpoint the successor owner publishes afterwards lands
	// on the chain — so the fold is the sole newest authority
	// and this implication is the only fact the lease's base
	// asserts, at this one site every launch-class path passes
	// (launch and restore alike, with or without a required
	// checkpoint or journal). A checkpoint-carrying winner over
	// a null fold parks: the chain is behind the lease in a way
	// no local launch may paper over. Nothing else about the
	// base is asserted here — binding the base to a checkpoint
	// the chain published is the takeover leaf's obligation
	// (stated bound).
	if input.Observation.HasWinner && input.Observation.Winner.HasCheckpoint && !input.HasNewestCheckpoint {
		return parked(fencing.ParkRestorePolicy, input.Observation.Winner.LeaseID, errSuccessorNullFold())
	}

	if input.MaterializationRequired {
		if err := checkMaterialization(input); err != nil {
			return parked(fencing.ParkRestorePolicy, input.Observation.Winner.LeaseID, err)
		}
	}
	if input.CheckpointRequired {
		if _, err := admitCheckpoint(input); err != nil {
			return parked(fencing.ParkRestorePolicy, input.Observation.Winner.LeaseID, err)
		}
	}

	if err := checkProviderIdentity(input.Provider, input.SessionID); err != nil {
		return refused(providerClass(err), "provider identity is unresolved", err)
	}
	if err := checkSmoke(input); err != nil {
		decision := refused(smokeClass(err), "provider-auth smoke precondition failed", err)
		var typed *axerror.Error
		if errors.As(err, &typed) {
			decision.RealmFailure = typed
		}
		return decision
	}
	if err := checkRealm(input, admitted, evidence); err != nil {
		var malformed errInvalidArguments
		if errors.As(err, &malformed) {
			return refused(ClassInvalidArguments, "caller realm details are incomplete", err)
		}
		var typed *axerror.Error
		if errors.As(err, &typed) {
			decision := refused(string(typed.Code()), "caller has no admitted realm evidence", err)
			decision.RealmFailure = typed
			return decision
		}
		return refused(ClassInvalidArguments, "caller realm is unverifiable", err)
	}
	pair, err := deriveProfile(input)
	if err != nil {
		return refused(ClassIntegrityFailure, "effective profile derivation failed", err)
	}
	// The tuple gate above already admitted the probed build, so a
	// ResolveMapping failure here is the mapping gap Section 2.4
	// names, never a tuple refusal wearing the mapping class.
	resolved, err := provhost.ResolveMapping(input.Provider.Build.ProviderID, pair.Profile, input.Provider.Build)
	if err != nil {
		return refused(ClassProfileMappingUnavailable, "adapter cannot map the stored profile", err)
	}
	if err := terminalbackend.CheckEntrypoint(input.Entrypoint, input.SessionID); err != nil {
		return refused(backendClass(err), "stable entrypoint is not ax pane SESSION_ID", err)
	}

	decision := Decision{
		Action:   ActionLaunch,
		Token:    token,
		HasToken: true,
		Profile:  pair,
		Mapping:  resolved.Mapping,
		Admitted: admitted,
		Detail:   "launch under the winning lease",
	}
	if input.ExistingBinding != nil && input.ExistingBinding.OperationID == input.BootstrapOperationID {
		decision.Action = ActionReattach
		decision.Detail = "reattach to the one recorded wrapper/child"
	}
	descriptor, err := terminalbackend.AdmitProviderDescriptor(input.Descriptor, input.HostBinding)
	if err != nil {
		return refused(backendClass(err), "provider descriptor does not match the validated binding", err)
	}
	decision.Descriptor = descriptor
	return decision
}

// decideParked resolves a fencing park into the parked outcome or,
// for a remote owner in an interactive terminal, the attach/takeover
// offer of after-restore step 4. Attach requires the admitted attach
// capability; an interactive remote owner without it offers takeover
// instead of starting a runtime. A remote owner outside an
// interactive terminal parks with the remote_owner reason (§4.2 step
// 5: parked in all other cases). Non-remote parks carry the fencing
// reason unchanged.
func decideParked(input Input, admitted terminalbackend.Admitted, reason fencing.ParkReason, winning string, cause error) Decision {
	if reason != fencing.ParkRemoteOwner {
		return parked(reason, winning, cause)
	}
	if !input.RemoteInteractive {
		return parked(reason, winning, cause)
	}
	if terminalbackend.CheckOperation("attach", admitted) == nil {
		return Decision{
			Action:         ActionAttachRemote,
			ParkReason:     reason,
			WinningLeaseID: winning,
			Admitted:       admitted,
			Detail:         "offer remote attach; no lease mutation",
			Cause:          cause,
		}
	}
	return Decision{
		Action:         ActionTakeoverOffer,
		ParkReason:     reason,
		WinningLeaseID: winning,
		Admitted:       admitted,
		Detail:         "offer takeover; no runtime started",
		Cause:          cause,
	}
}

// bootstrapWindowClosed reports whether the §13.1 bootstrap window
// has closed: the chain fold published a newest checkpoint, so a
// new bootstrap operation is a legitimate post-window binding, not
// an in-window mismatch. The predicate reads only the fold fact,
// never a caller claim and never the lease record's own
// checkpoint, which is null for every epoch-1 owner. A successor
// lease (epoch 2 or later) carries the handoff checkpoint and so
// implies a published newest, but the fold is the authority the
// window closes on.
func bootstrapWindowClosed(input Input) bool {
	return input.HasNewestCheckpoint
}

// authorize selects the fencing entry by mode: activation for a
// fresh launch, restore for the after-restore first start. Both are
// launch-class entries: ownership that is remote, ambiguous,
// unverified, or absent parks with the session.parked vocabulary,
// while malformed calls, foreign sessions, and lapsed grants refuse.
func authorize(input Input) (fencing.LeaseToken, error) {
	if input.Mode == ModeRestore {
		return fencing.AuthorizeRestore(input.Presented, input.Observation)
	}
	return fencing.AuthorizeActivation(input.Presented, input.Observation)
}

// admitBackend parses the §4.B identity documents through the
// terminalbackend owner, reconciles the Probe against the Manifest
// with the returned evidence bound to the AX-known raw generation,
// and gates the mode operation against the admitted set. Only the
// exact admitted capability rows authorize the action; no
// capability map is invented here. The parsed evidence is returned
// alongside the admitted set so the realm arm can bind the
// credential-realm row to the host binding and probed build
// without re-parsing.
func admitBackend(input Input) (terminalbackend.Admitted, []terminalbackend.Evidence, error) {
	manifest, err := terminalbackend.ParseManifest(input.Backend.Manifest)
	if err != nil {
		return terminalbackend.Admitted{}, nil, err
	}
	probe, err := terminalbackend.ParseProbe(input.Backend.Probe)
	if err != nil {
		return terminalbackend.Admitted{}, nil, err
	}
	evidence := make([]terminalbackend.Evidence, 0, len(input.Backend.Evidence))
	for _, doc := range input.Backend.Evidence {
		parsed, err := terminalbackend.ParseEvidence(doc)
		if err != nil {
			return terminalbackend.Admitted{}, nil, err
		}
		evidence = append(evidence, parsed)
	}
	admitted, err := terminalbackend.Reconcile(manifest, probe, evidence, input.Backend.RawGeneration, input.Backend.Now, input.Backend.Verify)
	if err != nil {
		// A caller that needs the realm row but submitted only
		// dead realm evidence (expired at the admission instant
		// or bound to a superseded generation) refuses the §4.2
		// capability_unavailable with typed realm/readiness
		// details, not the backend reconcile class: the realm
		// evidence is unusable however the reconcile failed.
		if realmErr := deadRealmRefusal(input, evidence, err); realmErr != nil {
			return terminalbackend.Admitted{}, nil, realmErr
		}
		return terminalbackend.Admitted{}, nil, err
	}
	operation := "create"
	if input.Mode == ModeRestore {
		operation = "restore"
	}
	if err := terminalbackend.CheckOperation(operation, admitted); err != nil {
		return terminalbackend.Admitted{}, nil, err
	}
	return admitted, evidence, nil
}

// checkMaterialization enforces after-restore step 3: the required
// journal must be admitted by the matjournal owner, committed, and
// sourced from the fold's newest published checkpoint. A committed
// journal one generation behind the newest checkpoint is stale,
// never valid; with no published checkpoint the requirement parks,
// never launches. The winning lease's handoff base is never
// consulted: the fold is the sole newest authority.
func checkMaterialization(input Input) error {
	if !input.JournalOK || input.Journal.Phase != matjournal.PhaseCommitted {
		return errMaterializationNotValid(input.Journal.Phase, input.JournalOK)
	}
	if !input.HasNewestCheckpoint {
		return errMaterializationNoCheckpoint()
	}
	if input.Journal.SourceCheckpointID != input.NewestCheckpointID {
		return errMaterializationStale()
	}
	return nil
}

// admitCheckpoint admits exactly the fold's newest published
// checkpoint for resume (§13.11 step 5: validate the newest
// checkpoint). The closed shape must attest through the sessrepo
// owner, the computed identity must equal the required digest, the
// required digest must equal the newest published checkpoint, and
// the checkpoint's own session_id must name this session. With no
// published checkpoint the requirement parks: an unpublished
// checkpoint is never admitted. The winning lease's handoff base
// is never compared: the fold is the sole newest authority. It
// returns the checkpoint's event-head closure for the profile
// derivation.
func admitCheckpoint(input Input) ([]string, error) {
	doc := input.CheckpointDoc
	want := input.CheckpointID
	if len(doc) == 0 {
		return nil, errCheckpointAbsent()
	}
	digest, _, err := sessrepo.AttestCheckpointRecord(doc)
	if err != nil {
		return nil, err
	}
	if digest.String() != want {
		return nil, errCheckpointMismatch()
	}
	if !input.HasNewestCheckpoint {
		return nil, errCheckpointNoPublished()
	}
	if want != input.NewestCheckpointID {
		return nil, errCheckpointNotNewest()
	}
	sessionID, _, _, heads, err := decodeCheckpointMembers(doc)
	if err != nil {
		return nil, err
	}
	if sessionID != input.SessionID {
		return nil, errCheckpointForeign()
	}
	return heads, nil
}

// decodeCheckpointMembers reads the bound members from checkpoint
// bytes the sessrepo owner already attested: the subject session,
// the owning lease, and the event-head closure. It mirrors the
// sessquery extraction after AttestCheckpointRecord: shape and
// identity are the owner's, this is only the member read.
func decodeCheckpointMembers(doc []byte) (sessionID string, leaseEpoch uint64, leaseID string, heads []string, err error) {
	members, fault := environ.DecodeStrictObject(doc)
	if fault != nil {
		return "", 0, "", nil, errCheckpointMembers()
	}
	session, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return "", 0, "", nil, errCheckpointMembers()
	}
	sessionID = session.String()
	rawEpoch, ok := members["lease_epoch"]
	if !ok {
		return "", 0, "", nil, errCheckpointMembers()
	}
	var epochFloat float64
	if err := json.Unmarshal(rawEpoch, &epochFloat); err != nil || epochFloat < 1 {
		return "", 0, "", nil, errCheckpointMembers()
	}
	leaseEpoch = uint64(epochFloat)
	var lease string
	if rawLease, ok := members["lease_id"]; ok {
		_ = json.Unmarshal(rawLease, &lease)
	}
	if _, err := scalar.ParseUUIDv4(lease); err != nil {
		return "", 0, "", nil, errCheckpointMembers()
	}
	leaseID = lease
	rawHeads, ok := members["event_heads"]
	if !ok {
		return "", 0, "", nil, errCheckpointMembers()
	}
	var values []string
	if err := json.Unmarshal(rawHeads, &values); err != nil {
		return "", 0, "", nil, errCheckpointMembers()
	}
	for _, value := range values {
		head, err := scalar.ParseDigest(value)
		if err != nil {
			return "", 0, "", nil, errCheckpointMembers()
		}
		heads = append(heads, head.String())
	}
	return sessionID, leaseEpoch, leaseID, heads, nil
}

// checkpointHeads extracts the event-head closure from checkpoint
// bytes for the profile derivation. It attests through the sessrepo
// owner first; unattested bytes never supply a closure.
func checkpointHeads(doc []byte) ([]string, error) {
	if len(doc) == 0 {
		return nil, errCheckpointAbsent()
	}
	if _, _, err := sessrepo.AttestCheckpointRecord(doc); err != nil {
		return nil, err
	}
	_, _, _, heads, err := decodeCheckpointMembers(doc)
	return heads, err
}

// deriveProfile derives the effective persisted pair: from the
// validated checkpoint's event-head closure on resume (§13.10),
// from the session head otherwise. The closure heads arrive from
// the checkpoint actually resumed through Run (the required
// checkpoint when one is required, else the fold's newest — never
// the winning lease's handoff base), or from the required
// checkpoint's own closure when Run's heads are not supplied; a
// head-only or losing-lease change outside the closure never
// becomes the launch profile.
func deriveProfile(input Input) (sessprofile.Pair, error) {
	heads := input.ProfileHeads
	if len(heads) == 0 && input.CheckpointRequired && len(input.CheckpointDoc) > 0 {
		closure, err := checkpointHeads(input.CheckpointDoc)
		if err != nil {
			return sessprofile.Pair{}, err
		}
		heads = closure
	}
	if len(heads) > 0 {
		return input.ProfileData.DeriveForHeads(heads)
	}
	return input.ProfileData.Derive()
}

// checkProviderIdentity composes the provhost resume gate: the exact
// probed build tuple must pass the Section 8.4 direction, the stored
// identity must validate for that provider at that exact version,
// the record's own session_id must name this session, and the
// discovery proof, when present, must bind to the host-side facts.
// The session binding is the wrapper's (VerifyIdentityBuild binds
// provider and version only): an identity minted for another
// session with the same provider and version never authorizes this
// one, exactly as the checkpoint arm binds the checkpoint's own
// session_id.
func checkProviderIdentity(facts ProviderFacts, sessionID string) error {
	if err := provhost.CheckResumeTuple(facts.Build); err != nil {
		return err
	}
	if err := provhost.VerifyIdentityBuild(facts.Identity, facts.Build); err != nil {
		return err
	}
	if err := checkIdentitySession(facts.Identity, sessionID); err != nil {
		return err
	}
	if facts.HasDiscovery {
		if err := provhost.VerifyIdentityDiscovery(facts.Identity, facts.Discovery, facts.DiscoveryContext); err != nil {
			return err
		}
	}
	return nil
}

// checkIdentitySession binds the validated Provider Identity Record
// to the session under decision. It mirrors the checkpoint member
// read after attestation: shape and identity are the owner's
// (CheckIdentity inside VerifyIdentityBuild just proved the record
// well-formed), this is only the subject read.
func checkIdentitySession(identity []byte, sessionID string) error {
	members, fault := environ.DecodeStrictObject(identity)
	if fault != nil {
		return errIdentityMembers()
	}
	session, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return errIdentityMembers()
	}
	if session.String() != sessionID {
		return errIdentityForeign()
	}
	return nil
}

// checkSmoke enforces the provider-auth smoke precondition when the
// path requires provider credentials. The stored record must verify
// through the resumesmoke owner, its verdict must be pass, its exact
// claimed build must equal the probed build, and the typed target
// must bind the same build with the current generation and the
// probed OS version: a smoke result for one build, generation, or
// OS version never authorizes another. A required smoke that is
// absent, unverifiable, failing, or bound to another
// build/generation/version refuses target_auth_missing with the
// typed details Section 15.3 names.
func checkSmoke(input Input) error {
	if !input.Smoke.Required {
		return nil
	}
	record, err := resumesmoke.VerifyRecord(input.Smoke.Record)
	if err != nil {
		return smokeFailure(input, err)
	}
	if record.Verdict != resumesmoke.VerdictPass {
		return smokeFailure(input, errSmokeVerdict(record.Verdict))
	}
	build := input.Provider.Build
	if record.Tuple.ProviderID != build.ProviderID ||
		record.Tuple.ProviderVersion != build.ProviderVersion ||
		record.Tuple.Platform != build.Platform ||
		record.Tuple.Architecture != build.Architecture {
		return smokeFailure(input, errSmokeTuple())
	}
	target := input.Smoke.Target
	if target.ProviderID != build.ProviderID || target.ProviderBuild != build.ProviderVersion {
		return smokeFailure(input, errSmokeTargetBuild())
	}
	if target.TmuxServerGeneration != input.Backend.RawGeneration {
		return smokeFailure(input, errSmokeTargetGeneration())
	}
	probe, err := terminalbackend.ParseProbe(input.Backend.Probe)
	if err != nil {
		return smokeFailure(input, err)
	}
	if target.MacOSVersion != probe.OSVersion {
		return smokeFailure(input, errSmokeTargetOS())
	}
	return nil
}

// smokeFailure composes the typed target_auth_missing refusal. Typed
// details are required members of the input: an input that omits
// one is itself malformed and refuses invalid_arguments instead of
// emitting an untyped refusal.
func smokeFailure(input Input, cause error) error {
	failure, err := axerror.NewTargetAuthMissing(axerror.Version130,
		"provider-auth smoke did not pass for the probed build",
		axerror.NoIDs(), input.Smoke.Target, cause)
	if err != nil {
		return fmt.Errorf("%w: smoke refusal details are incomplete: %v", errInvalidArguments{}, err)
	}
	return failure
}

// checkRealm enforces the §4.C credential conditional with the
// §4.2 background-caller rule: a background caller, and any caller
// on a path that requires provider credentials, authorizes only
// through the admitted credential_capable_execution_realm
// Capability Evidence row bound to the exact host binding, the
// probed provider build, and the current generation, admitted
// through the landed terminalbackend reconcile path. It never falls
// back to direct server creation, and no cached sentinel,
// managername observation, or bare boolean authorizes resume. A
// foreground caller on a credential-free path passes. Expiry,
// logout, and reboot invalidation are the reconcile liveness and
// generation binding the admission already enforces; a stale
// generation in the typed details refuses as well. Typed details
// are required input members: an input that omits one is itself
// malformed and refuses invalid_arguments instead of emitting an
// untyped capability_unavailable.
func checkRealm(input Input, admitted terminalbackend.Admitted, evidence []terminalbackend.Evidence) error {
	if input.Realm.Caller != CallerBackground && !input.Smoke.Required {
		return nil
	}
	if !admitted.Has(credentialRealmCapability) {
		return realmFailure(input, errNoRealmEvidence())
	}
	if input.Realm.ServerGeneration != input.Backend.RawGeneration {
		return realmFailure(input, errRealmGenerationStale())
	}
	if err := checkRealmBinding(input, evidence); err != nil {
		return realmFailure(input, err)
	}
	return nil
}

// checkRealmBinding binds the admitted realm row to the exact host
// binding and probed provider build: some submitted realm evidence
// object must carry the validated host-local terminal binding and
// the probed provider id/build. The scan accepts when any realm
// object binds, so the outcome never depends on evidence order.
// Generation, platform, OS version, expiry, and signature are the
// reconcile binding the admission already proves; this is only the
// wrapper-side cross-bind the lifecycle owner evaluates.
func checkRealmBinding(input Input, evidence []terminalbackend.Evidence) error {
	seen := false
	for _, object := range evidence {
		if object.Capability != credentialRealmCapability {
			continue
		}
		seen = true
		if object.TerminalBindingID != input.HostBinding.TerminalBindingID {
			continue
		}
		if object.ProviderID != input.Provider.Build.ProviderID ||
			object.ProviderBuild != input.Provider.Build.ProviderVersion {
			continue
		}
		return nil
	}
	if !seen {
		return errNoRealmEvidence()
	}
	return errRealmBinding()
}

// deadRealmRefusal maps a backend admission failure to the typed
// capability_unavailable refusal when the caller needs the realm
// row, submitted at least one realm evidence object, and none is
// usable (live at the admission instant and bound to the current
// generation). An expired or pre-reboot realm row fails reconcile
// first; the wrapper reports the §4.2 typed refusal the spec
// names, with the reconcile failure as the cause. It returns nil
// when the backend class stands: a caller that needs no realm
// row, no submitted realm row at all, or any usable realm row.
func deadRealmRefusal(input Input, evidence []terminalbackend.Evidence, cause error) error {
	if input.Realm.Caller != CallerBackground && !input.Smoke.Required {
		return nil
	}
	submitted := false
	for _, object := range evidence {
		if object.Capability != credentialRealmCapability {
			continue
		}
		submitted = true
		if realmRowUsable(input, object) {
			return nil
		}
	}
	if !submitted {
		return nil
	}
	return realmFailure(input, cause)
}

// realmRowUsable classifies one parsed realm evidence object for
// the refusal mapping: live at the admission instant and bound to
// the AX-known raw generation. It classifies the refusal only —
// the admission decision (refuse) is already made by Reconcile,
// which re-proves liveness and generation on the admitted path.
// This restates the reconcile liveness and generation predicate
// as a stated bound, not a derivation: the aggregate Reconcile
// error carries no per-object identity (liveness shares the
// generic mismatch code), so no per-row usability verdict can be
// read off it. The copy is classification-only and fail-closed in
// both directions — it selects the refusal class, never admission.
func realmRowUsable(input Input, object terminalbackend.Evidence) bool {
	digest, err := terminalbackend.GenerationDigest(input.Backend.RawGeneration)
	if err != nil {
		return false
	}
	if object.BackendGenerationDigest != digest {
		return false
	}
	observed, err := object.ObservedAt.Time()
	if err != nil {
		return false
	}
	expires, err := object.ExpiresAt.Time()
	if err != nil {
		return false
	}
	if observed.After(input.Backend.Now) || !input.Backend.Now.Before(expires) {
		return false
	}
	return true
}

// realmFailure composes the typed capability_unavailable refusal.
// Typed details are required members of the input: an input that
// omits one is itself malformed and refuses invalid_arguments
// instead of emitting an untyped refusal.
func realmFailure(input Input, cause error) error {
	details := axerror.RealmEvidence{
		Capability:           credentialRealmCapability,
		CallerRealm:          input.Realm.Caller,
		BrokerState:          input.Realm.BrokerState,
		TmuxServerGeneration: input.Realm.ServerGeneration,
		Remediation:          input.Realm.Remediation,
	}
	failure, err := axerror.NewRealmEvidenceUnavailable(axerror.Version130,
		"caller has no admitted realm evidence",
		axerror.NoIDs(), details, cause)
	if err != nil {
		return errInvalidArguments{detail: "background caller realm details are incomplete"}
	}
	return failure
}

// checkRealmVocabulary closes the caller vocabulary: only
// foreground and background name a caller.
func checkRealmVocabulary(caller string) error {
	if caller != CallerForeground && caller != CallerBackground {
		return errUnknownCaller(caller)
	}
	return nil
}

// backendClass reads the Section 15 code off a terminalbackend
// refusal. Landed codes pass through verbatim.
func backendClass(err error) string {
	var backend *terminalbackend.Error
	if errors.As(err, &backend) {
		return backend.Code
	}
	return terminalbackend.CodeMismatch
}

// fencingClass reads the Section 15 class off a fencing refusal:
// invalid_arguments for malformed calls, lease_conflict for
// foreign sessions and lapsed grants. Parks never reach here;
// decideParked owns them.
func fencingClass(err error) string {
	if errors.Is(err, fencing.ErrInvalidArguments) {
		return ClassInvalidArguments
	}
	return "lease_conflict"
}

// providerClass reads the Structured Error code off a provhost
// refusal: invalid_config for tuple and mapping grammar, the
// provider protocol code for identity and discovery failures.
func providerClass(err error) string {
	var failure *axerror.Error
	if errors.As(err, &failure) {
		return string(failure.Code())
	}
	return ClassInvalidConfig
}

// smokeClass reads the Structured Error code off a smoke refusal:
// target_auth_missing for a required smoke that fails, or
// invalid_arguments for a malformed smoke input.
func smokeClass(err error) string {
	var failure *axerror.Error
	if errors.As(err, &failure) {
		return string(failure.Code())
	}
	var malformed errInvalidArguments
	if errors.As(err, &malformed) {
		return ClassInvalidArguments
	}
	return "target_auth_missing"
}
