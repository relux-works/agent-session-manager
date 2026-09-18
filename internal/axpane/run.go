package axpane

import (
	"errors"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Request is the complete Run surface: the session and bootstrap
// operation under decision plus every modeled-caller fact the
// adapters cannot load from the durable stores (sync state,
// provider build and identity, backend documents, realm readiness,
// presentation geometry). The stores supply configuration,
// sessions, leases, journals, checkpoints, and bindings.
type Request struct {
	SessionID            string
	BootstrapOperationID string
	Mode                 Mode

	ConfigInputs    config.Inputs
	ConfigOverrides config.Overrides

	// Presented is the local fencing token; Observe carries the
	// caller-reported sync, handoff, grant, and clock facts.
	Presented fencing.PresentedToken
	Observe   fencing.ObserveInput

	// Materialization selects the required journal: when Required
	// is set the adapter loads ID and the decision requires the
	// committed phase (after-restore step 3).
	MaterializationID       string
	MaterializationRequired bool

	// Checkpoint selects the resume checkpoint: when Required is
	// set the adapter loads ID and the decision admits it.
	CheckpointID       string
	CheckpointRequired bool

	Provider ProviderFacts
	Smoke    SmokeFacts
	Realm    Realm
	Backend  BackendFacts

	// Entrypoint is the wrapper argv.
	Entrypoint []string

	// HostBinding is the AX-validated host-local binding subset
	// the supplied descriptor must match. Geometry and
	// interactivity shape the supplied descriptor; InstanceID
	// pins the terminal instance, or mints a fresh UUIDv7 when
	// empty.
	HostBinding terminalbackend.InstanceBinding
	Interactive bool
	Columns     uint16
	Rows        uint16
	InstanceID  string

	// RemoteInteractive reports that the remote owner holds the
	// session in an interactive terminal (after-restore step 4).
	RemoteInteractive bool

	// LocalHostID authors parked events; CreatedAt stamps them.
	LocalHostID string
	CreatedAt   string

	// Hooks arms the bootstrap binding crash boundaries.
	Hooks *Hooks
}

// Outcome is the one Run result: the decision plus the durable
// effects it authorized. Exactly one effect applies: a launch binds
// the bootstrap pair (superseding after the window closed), a
// reattach returns the recorded binding, a parked or remote-offer
// decision with a locally held, verified winner and a foldable
// lifecycle authors the parked event, and every other outcome
// writes nothing.
type Outcome struct {
	Decision Decision
	// Binding is the bound receipt on launch and the recorded
	// receipt on reattach, nil otherwise.
	Binding *Binding
	// Event is the authored parked event reference and bytes;
	// Emitted is false when the decision parks without a locally
	// held winner, without a foldable lifecycle, or when the
	// outcome is neither parked nor a remote offer. A park without
	// chain authority parks without evidence, never by manufacturing
	// a sequence collision under another host's lease.
	Event     *sessrepo.EventRef
	EventJSON []byte
	Emitted   bool
}

// Run orchestrates the wrapper: validate configuration, load the
// logical session, derive the fold's newest checkpoint, prove the
// bootstrap binding state, observe the winning lease, load
// materialization, checkpoint, and profile state, decide through
// the pure core, then apply exactly the authorized durable
// effect. A store failure propagates as an error with no decision:
// unknown is never absence and never a launch.
func Run(stores Stores, request Request) (Outcome, error) {
	empty := Outcome{}
	if stores.Repo == nil || stores.Pane == nil {
		return empty, fmt.Errorf("axpane run carries no repository or binding store")
	}
	// The request hooks arm a copy of the binding handle: Run never
	// mutates the caller's store.
	pane := *stores.Pane
	paneStore := pane.WithHooks(request.Hooks)
	input := Input{
		SessionID:               request.SessionID,
		BootstrapOperationID:    request.BootstrapOperationID,
		Mode:                    request.Mode,
		Presented:               request.Presented,
		Provider:                request.Provider,
		Smoke:                   request.Smoke,
		Realm:                   request.Realm,
		Backend:                 request.Backend,
		Entrypoint:              request.Entrypoint,
		HostBinding:             request.HostBinding,
		RemoteInteractive:       request.RemoteInteractive,
		Interactive:             request.Interactive,
		CheckpointRequired:      request.CheckpointRequired,
		CheckpointID:            request.CheckpointID,
		MaterializationRequired: request.MaterializationRequired,
	}
	input.ConfigErr = ValidateConfig(request.ConfigInputs, request.ConfigOverrides)

	known, _, err := LoadSession(stores.Repo, request.SessionID)
	if err != nil {
		return empty, err
	}
	input.SessionKnown = known

	pair, pairFound, err := paneStore.Lookup(request.SessionID, request.BootstrapOperationID)
	if err != nil {
		return empty, err
	}
	if pairFound {
		duplicate := pair
		input.ExistingBinding = &duplicate
	}
	bound, err := paneStore.SessionBound(request.SessionID)
	if err != nil {
		return empty, err
	}
	input.SessionBound = bound

	if known {
		has, newest, err := LoadNewestCheckpoint(stores.Repo, request.SessionID)
		if err != nil {
			return empty, err
		}
		input.HasNewestCheckpoint = has
		input.NewestCheckpointID = newest
	}

	var observation fencing.Observation
	if known {
		observed, err := ObserveOwnership(stores.Repo, request.SessionID, request.Observe)
		if err != nil {
			return empty, err
		}
		observation = observed
	}
	input.Observation = observation

	// The fold's newest checkpoint drives the resume profile
	// closure, so a published newest with no checkpoint store
	// bound is unknown, not "no closure": deriving from the
	// session head instead would silently promote a head-only
	// change to the launch profile. Fail the run. (A required
	// checkpoint with no store fails at its own load below; a
	// null fold needs no store at all.)
	if known && input.HasNewestCheckpoint && stores.Ckpt == nil {
		return empty, fmt.Errorf("axpane run requires the newest checkpoint closure with no checkpoint store")
	}

	if request.MaterializationRequired {
		if stores.Mat == nil {
			return empty, fmt.Errorf("axpane run requires a materialization with no journal store")
		}
		journal, admitted, err := LoadMaterialization(stores.Mat, request.MaterializationID)
		if err != nil {
			return empty, err
		}
		input.Journal = journal
		input.JournalOK = admitted
	}
	if request.CheckpointRequired {
		if stores.Ckpt == nil {
			return empty, fmt.Errorf("axpane run requires a checkpoint with no checkpoint store")
		}
		doc, err := LoadCheckpoint(stores.Ckpt, request.CheckpointID)
		if err != nil {
			return empty, err
		}
		input.CheckpointDoc = doc
	}
	if known {
		record, events, err := LoadProfile(stores.Repo, request.SessionID)
		if err != nil {
			return empty, err
		}
		input.ProfileRecord = record
		input.ProfileEvents = events
		// Resume derives the effective profile from the
		// checkpoint actually resumed — the required checkpoint
		// when one is required, else the fold's newest — never
		// the session head and never the winning lease's handoff
		// base, so a head-only or losing-lease change cannot
		// become the launch profile (§13.10, §2.4). With no
		// required checkpoint and no published newest (the
		// epoch-1 bootstrap-retry path) the session head stays
		// the source.
		closureDoc := input.CheckpointDoc
		if !request.CheckpointRequired || len(input.CheckpointDoc) == 0 {
			closureDoc = nil
			if input.HasNewestCheckpoint && stores.Ckpt != nil {
				loaded, err := LoadCheckpoint(stores.Ckpt, input.NewestCheckpointID)
				if err != nil {
					return empty, err
				}
				if len(loaded) == 0 {
					return empty, fmt.Errorf("axpane run requires the newest checkpoint closure with the newest absent from the store")
				}
				closureDoc = loaded
			}
		}
		if len(closureDoc) > 0 {
			heads, err := checkpointHeads(closureDoc)
			if err != nil {
				return empty, err
			}
			input.ProfileHeads = heads
		}
	}

	instance := request.InstanceID
	if instance == "" {
		minted, err := MintInstanceIDRandom()
		if err != nil {
			return empty, err
		}
		instance = minted
	}
	descriptorDoc, err := BuildDescriptor(DescriptorParams{
		Binding: BindingRef{
			BindingDigest:         request.HostBinding.TerminalBindingID,
			BackendID:             request.HostBinding.BackendID,
			ImplementationVersion: request.HostBinding.ImplementationVersion,
			ProtocolVersion:       request.HostBinding.ProtocolVersion,
			Generation:            request.HostBinding.Generation,
		},
		InstanceID:  instance,
		Interactive: request.Interactive,
		Columns:     request.Columns,
		Rows:        request.Rows,
	})
	if err != nil {
		return empty, err
	}
	input.Descriptor = descriptorDoc

	decision := Decide(input)
	outcome := Outcome{Decision: decision}
	switch decision.Action {
	case ActionLaunch:
		candidate := Binding{
			SessionID:          request.SessionID,
			OperationID:        request.BootstrapOperationID,
			TerminalInstanceID: instance,
		}
		candidate.BindingDigest = BindingDigest(candidate)
		// A launch reaches here only for an unrecorded pair (a
		// recorded pair reattaches instead). On a bound session
		// the decision already applied the window predicate, so
		// a launch here is post-window and records its own pair
		// receipt through Supersede, keeping every superseded
		// receipt; on an unbound session the launch binds
		// no-replace. A mismatch from Bind is a concurrent
		// changed-operation commit the retry observes through
		// Lookup.
		if input.SessionBound {
			bound, replayed, err := paneStore.Supersede(request.SessionID, request.BootstrapOperationID, candidate)
			if err != nil {
				return empty, err
			}
			if replayed {
				// A concurrent wrapper recorded this pair
				// between our Lookup and our Supersede: the
				// pair is bound, so this run reattaches to
				// the recorded child instead of launching a
				// second one — the post-window mirror of the
				// Bind race below.
				outcome.Decision.Action = ActionReattach
				outcome.Decision.Detail = "reattach to the one recorded wrapper/child"
			}
			duplicate := bound
			outcome.Binding = &duplicate
			break
		}
		bound, reattached, err := paneStore.Bind(request.SessionID, request.BootstrapOperationID, candidate)
		if err != nil {
			// A concurrent changed-operation commit surfaces
			// here as idempotency_mismatch: the caller's retry
			// observes the winner through Status and decides
			// the refused outcome.
			return empty, err
		}
		if reattached {
			// A concurrent wrapper won the bind between Status
			// and Bind: the pair is bound, so this run
			// reattaches to the recorded child instead of
			// launching a second one.
			outcome.Decision.Action = ActionReattach
			outcome.Decision.Detail = "reattach to the one recorded wrapper/child"
		}
		duplicate := bound
		outcome.Binding = &duplicate
	case ActionReattach:
		duplicate := *input.ExistingBinding
		outcome.Binding = &duplicate
	case ActionParked, ActionAttachRemote, ActionTakeoverOffer:
		if !observation.HasWinner {
			// No winner establishes no authority: session.parked
			// requires a winning lease ID, so the decision
			// parks without authoring an event.
			return outcome, nil
		}
		if observation.Winner.HolderHostID != observation.LocalHostID {
			// A remote winner holds the lease: the local
			// wrapper parks without manufacturing a sequence
			// collision under another host's lease (§5.2: the
			// winning owner serializes state-changing
			// events). No host-local chain record exists for
			// a remote park; the decision itself is the
			// parked evidence.
			return outcome, nil
		}
		foldable, err := canAuthorParked(stores.Repo, request.SessionID)
		if err != nil {
			return empty, err
		}
		if !foldable {
			// The §5.7 table cannot fold session.parked from
			// the derived state: the decision parks without
			// authoring an unfolderable lifecycle event.
			return outcome, nil
		}
		reference, raw, err := emitParked(stores.Repo, decision, observation, request)
		if err != nil {
			// The mutation gate refused (unverified,
			// ambiguous, failed handoff, lapsed grant): the
			// decision parks without chain evidence rather
			// than failing the run. A store failure also
			// lands here; unknown is never a launch, but the
			// parked decision stands with no event.
			// Emit-level refusals are distinguished by the
			// fencing owner; repository failures propagate.
			if fencing.IsParked(err) || isFencingRefusal(err) {
				return outcome, nil
			}
			outcome.Emitted = false
			return outcome, err
		}
		outcome.Event = &reference
		outcome.EventJSON = raw
		outcome.Emitted = true
	}
	return outcome, nil
}

// isFencingRefusal reports whether err is a fencing gate refusal
// (as opposed to a repository failure): the parked decision stands
// with no chain evidence when the mutation gate refuses.
func isFencingRefusal(err error) bool {
	if err == nil {
		return false
	}
	if fencing.IsParked(err) {
		return true
	}
	return errors.Is(err, fencing.ErrInvalidArguments) ||
		errors.Is(err, fencing.ErrLeaseConflict) ||
		errors.Is(err, fencing.ErrNotOwner) ||
		errors.Is(err, fencing.ErrStaleOwner)
}

// emitParked authors the parked event under the winning lease the
// local host holds, with the current chain linkage: the next
// sequence under the winning lease, or sequence 1 referencing the
// predecessor head when the winner succeeded the tail lease. The
// presented token is the winning tuple itself, so the mutation
// gate authorizes the locally held winner; the decision's stale or
// unverified presented token never authors.
func emitParked(repository *sessrepo.Repository, decision Decision, observation fencing.Observation, request Request) (sessrepo.EventRef, []byte, error) {
	recordID, tail, err := ChainTail(repository, request.SessionID)
	if err != nil {
		return sessrepo.EventRef{}, nil, err
	}
	var sequence uint64 = 1
	predecessors := []string{recordID}
	if tail != nil {
		if tail.LeaseID == observation.Winner.LeaseID {
			sequence = tail.LeaseSequence + 1
		}
		predecessors = []string{tail.EventID}
	}
	presented := fencing.PresentedToken{
		SessionID: request.SessionID,
		Epoch:     observation.Winner.Epoch,
		LeaseID:   observation.Winner.LeaseID,
	}
	return EmitParked(repository, decision, EmitParams{
		SessionID:     request.SessionID,
		CreatedByHost: request.LocalHostID,
		LeaseEpoch:    observation.Winner.Epoch,
		LeaseID:       observation.Winner.LeaseID,
		LeaseSequence: sequence,
		Predecessors:  predecessors,
		CreatedAt:     request.CreatedAt,
		Presented:     presented,
		Observation:   observation,
	})
}
