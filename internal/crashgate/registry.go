package crashgate

// Path names the Section 13.13 direct/task-board dimension. Every
// boundary lists its applicable paths; a boundary that applies to one
// path only (for example the CR-FORCE-D direct-force rows) lists
// that path only.
const (
	PathDirect    = "direct"
	PathTaskBoard = "task_board"
)

// Durable-write classes. The first four are the story's reachable
// classes (checkpoint capture, journal transition, operation
// receipt, managed-replica marker); the rest name the unlanded
// owner class of the boundary's durable fact.
const (
	WriteCheckpoint = "checkpoint"
	WriteJournal    = "journal"
	WriteExternal   = "external"
)

// Driver keys name the conformance driver (test-side) that injects
// the crash and runs the recovery for a reachable path.
const (
	DriverCheckpointCapture = "checkpoint-capture"
	DriverJournalCreate     = "journal-create"
	DriverJournalTransfer   = "journal-transfer"
	DriverJournalValidate   = "journal-validate"
	DriverJournalPrepareOp  = "journal-prepare-op"
	DriverJournalToPrepared = "journal-to-prepared"
	DriverJournalActivation = "journal-activation"
	DriverJournalFinalize   = "journal-finalize"
	DriverJournalDormant    = "journal-dormant"
)

// BoundaryPath classifies one direct/task-board path of one crash
// boundary. Exactly one of Driver (reachable) and Owner (not
// applicable) is set; Note carries the honest why.
type BoundaryPath struct {
	Name      string
	Reachable bool
	Driver    string
	Owner     string
	Note      string
}

// Boundary is one crash boundary of the Section 13.13 registry:
// its ID, family, phase, durable-write class, and per-path
// classification.
type Boundary struct {
	ID           string
	Family       string
	Phase        string
	DurableWrite string
	Paths        []BoundaryPath
}

// Path returns the named path classification of the boundary.
func (boundary Boundary) Path(name string) (BoundaryPath, bool) {
	for _, path := range boundary.Paths {
		if path.Name == name {
			return path, true
		}
	}
	return BoundaryPath{}, false
}

// Reachable reports whether any path of the boundary is reachable
// through the landed owners.
func (boundary Boundary) Reachable() bool {
	for _, path := range boundary.Paths {
		if path.Reachable {
			return true
		}
	}
	return false
}

// registry is the classified Section 13.13 boundary table in
// specification order: the twelve registry-table ranges plus the
// CR-CLONE prose range. The ID set is verified against the pinned
// text by the derivation test; the per-path classification is the
// harness's explicit, reviewed mapping.
var registry = []Boundary{
	// CR-LAUNCH-D-01..05: direct bootstrap (Section 13.1). No
	// boundary writes a checkpoint, journal, receipt, or marker:
	// the first checkpoint is published by the next phase after
	// CR-LAUNCH-D-05.
	{ID: "CR-LAUNCH-D-01", Family: "CR-LAUNCH-D", Phase: "session/lease/event persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sessrepo session/lease/event persistence (Section 13.1 bootstrap; out of checkpoint/journal story boundary)", Note: "durable write is Session/Lease/Event records, not a checkpoint or journal"},
	}},
	{ID: "CR-LAUNCH-D-02", Family: "CR-LAUNCH-D", Phase: "terminal creation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "terminal lifecycle runtime (terminalbackend manifest/probe/conformance landed; creation runtime unlanded)", Note: "durable fact is terminal creation, not a checkpoint or journal"},
	}},
	{ID: "CR-LAUNCH-D-03", Family: "CR-LAUNCH-D", Phase: "provider process start", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider plugin/process runtime (Section 7/13.1; process start unlanded)", Note: "external effect with no checkpoint/journal durable write"},
	}},
	{ID: "CR-LAUNCH-D-04", Family: "CR-LAUNCH-D", Phase: "Provider Identity persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider identity persistence (canonical shape landed; durable persistence runtime unlanded)", Note: "durable write is the Provider Identity Record, not a checkpoint or journal"},
	}},
	{ID: "CR-LAUNCH-D-05", Family: "CR-LAUNCH-D", Phase: "capture/object validation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "direct bootstrap validation (Section 13.1; validation gate unlanded)", Note: "validation produces no durable bytes; the first-checkpoint publication that follows is a future lifecycle boundary, not this one"},
	}},
	// CR-LAUNCH-TB-01..03: task-board bootstrap (Section 13.2).
	{ID: "CR-LAUNCH-TB-01", Family: "CR-LAUNCH-TB", Phase: "session/lease/event persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Owner: "sessrepo session/lease/event persistence (Section 13.2 bootstrap; out of checkpoint/journal story boundary)", Note: "durable write is Session/Lease/Event records, not a checkpoint or journal"},
	}},
	{ID: "CR-LAUNCH-TB-02", Family: "CR-LAUNCH-TB", Phase: "bridge launch may have returned a manager reference", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Owner: "task-board bridge runtime (Section 9/13.2; launch execution unlanded)", Note: "external effect with no checkpoint/journal durable write"},
	}},
	{ID: "CR-LAUNCH-TB-03", Family: "CR-LAUNCH-TB", Phase: "bridge export may have returned a bundle/proof", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Owner: "task-board bridge runtime (Section 9/13.2; export execution unlanded)", Note: "durable fact is the bridge bundle/proof, consumed by checkpoint closure as input but written by the bridge"},
	}},
	// CR-SYNC-01..07: sync (Section 13.3). The passive
	// materialization rows are journal-observable through the
	// dormant walk; transfer staging and commit belong to the
	// sync/transfer flow.
	{ID: "CR-SYNC-01", Family: "CR-SYNC", Phase: "inventory exchange", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sync transfer flow (Section 13.3; inventory exchange unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "sync transfer flow (Section 13.3; inventory exchange unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-SYNC-02", Family: "CR-SYNC", Phase: "immutable union", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sync transfer flow (Section 13.3; immutable union unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "sync transfer flow (Section 13.3; immutable union unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-SYNC-03", Family: "CR-SYNC", Phase: "missing-object selection", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sync transfer flow (Section 13.3; missing-object selection unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "sync transfer flow (Section 13.3; missing-object selection unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-SYNC-04", Family: "CR-SYNC", Phase: "staged chunk progress", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sync transfer staging (Section 13.3; staged bytes unlanded)", Note: "durable write is transfer staging bytes; the journal record of staged progress is covered by CR-MAT-03"},
		{Name: PathTaskBoard, Owner: "sync transfer staging (Section 13.3; staged bytes unlanded)", Note: "durable write is transfer staging bytes; the journal record of staged progress is covered by CR-MAT-03"},
	}},
	{ID: "CR-SYNC-05", Family: "CR-SYNC", Phase: "immutable-object commit", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sync transfer flow (Section 13.3; object commit unlanded)", Note: "durable write is the object commit, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "sync transfer flow (Section 13.3; object commit unlanded)", Note: "durable write is the object commit, not a checkpoint or journal"},
	}},
	{ID: "CR-SYNC-06", Family: "CR-SYNC", Phase: "passive materialize.prepare/commit", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: false, Owner: "passive sync is a dormant-replica (task-board staging) plan; no direct plan reaches this phase", Note: "phase skipped by the direct plan"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalDormant, Note: "dormant prepare/commit through the journal-create seam, evaluated with the passive_sync path"},
	}},
	{ID: "CR-SYNC-07", Family: "CR-SYNC", Phase: "dormant finalize", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: false, Owner: "passive sync is a dormant-replica (task-board staging) plan; no direct plan reaches this phase", Note: "phase skipped by the direct plan"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalDormant, Note: "dormant finalize through the journal-finalize seam, evaluated with the passive_sync path"},
	}},
	// CR-MAT-01..08: every materialization. Driven in full through
	// matjournal plus sessckpt, both direct and task-board
	// closures where the plan phase applies. A phase the plan
	// skips (provider/bridge halves under a workspace-only
	// closure) is not applicable to that plan, per Section 13.13.
	{ID: "CR-MAT-01", Family: "CR-MAT", Phase: "before journal creation", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalCreate, Note: "crash before the first durable byte; identical retry creates exactly one journal"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalCreate, Note: "crash before the first durable byte; identical retry creates exactly one journal"},
	}},
	{ID: "CR-MAT-02", Family: "CR-MAT", Phase: "after journal/prepare receipt", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalCreate, Note: "crash between journal install and prepare receipt, and after the receipt; replay converges"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalCreate, Note: "crash between journal install and prepare receipt, and after the receipt; replay converges"},
	}},
	{ID: "CR-MAT-03", Family: "CR-MAT", Phase: "after transfer", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalTransfer, Note: "staged chunk facts survive; resume requests only absent indexes under the same IDs"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalTransfer, Note: "staged chunk facts survive; resume requests only absent indexes under the same IDs"},
	}},
	{ID: "CR-MAT-04", Family: "CR-MAT", Phase: "after validation", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalValidate, Note: "validation-phase facts survive the restart"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalValidate, Note: "validation-phase facts survive the restart"},
	}},
	{ID: "CR-MAT-05", Family: "CR-MAT", Phase: "after provider prepare or bridge import", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: false, Owner: "workspace-only direct plan carries no provider transaction or bridge; the prepare-op phase is skipped", Note: "phase skipped by the direct plan"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalPrepareOp, Note: "provider half (prepared token durable) and bridge half (imported state durable); failed-import rollback variant included"},
	}},
	{ID: "CR-MAT-06", Family: "CR-MAT", Phase: "after bridge open or host commit enters prepared", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalToPrepared, Note: "host half: validating-to-prepared transition durable under a workspace-only closure"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalToPrepared, Note: "bridge half (opened state durable) and host half (prepared journal durable)"},
	}},
	{ID: "CR-MAT-07", Family: "CR-MAT", Phase: "after owner activation may have occurred", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: false, Owner: "workspace-only direct plan carries no provider activation or bridge adopt; the activation phase is skipped", Note: "phase skipped by the direct plan"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalActivation, Note: "adopt recorded with agreeing status continues; unreachable status parks; never guesses"},
	}},
	{ID: "CR-MAT-08", Family: "CR-MAT", Phase: "after finalize/cleanup may have occurred", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalFinalize, Note: "direct finalize has no external effect in the plan: resume under the same IDs"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalFinalize, Note: "resumed state durable completes the commit; unproven finalize parks with the blocking reason durable"},
	}},
	// CR-GRACE-01..13: graceful takeover (Section 13.6). Prepare,
	// checkpoint, prepared-materialization, activation, and
	// finalize rows are checkpoint/journal-observable; input,
	// proof, sync, stop, lease, and event rows belong to their
	// unlanded owners.
	{ID: "CR-GRACE-01", Family: "CR-GRACE", Phase: "after prepare", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalCreate, Note: "takeover prepare through the journal-create seam, evaluated with the graceful_takeover path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalCreate, Note: "takeover prepare through the journal-create seam, evaluated with the graceful_takeover path"},
	}},
	{ID: "CR-GRACE-02", Family: "CR-GRACE", Phase: "after input block", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "input gating runtime (provider/terminal input block; Section 13.6 flow unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "input gating runtime (provider/terminal input block; Section 13.6 flow unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-GRACE-03", Family: "CR-GRACE", Phase: "after quiescence proof", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider quiescence observation (provhost proof decode landed; live idle observation unlanded)", Note: "quiescence declaration is enforced at checkpoint capture; proof generation is not a checkpoint/journal write"},
		{Name: PathTaskBoard, Owner: "provider quiescence observation (provhost proof decode landed; live idle observation unlanded)", Note: "quiescence declaration is enforced at checkpoint capture; proof generation is not a checkpoint/journal write"},
	}},
	{ID: "CR-GRACE-04", Family: "CR-GRACE", Phase: "after checkpoint capture", DurableWrite: WriteCheckpoint, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverCheckpointCapture, Note: "takeover checkpoint through the capture blob/receipt seam"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverCheckpointCapture, Note: "takeover checkpoint through the capture blob/receipt seam"},
	}},
	{ID: "CR-GRACE-05", Family: "CR-GRACE", Phase: "after final sync", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sync transfer flow (Section 13.3/13.6; final sync unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "sync transfer flow (Section 13.3/13.6; final sync unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-GRACE-06", Family: "CR-GRACE", Phase: "after destination validation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lifecycle orchestration (Section 13.6 destination validation; flow unlanded)", Note: "validation produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "lifecycle orchestration (Section 13.6 destination validation; flow unlanded)", Note: "validation produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-GRACE-07", Family: "CR-GRACE", Phase: "after source stop", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider/bridge stop runtime (Section 13.6/13.9; stop execution unlanded)", Note: "external effect with no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "provider/bridge stop runtime (Section 13.6/13.9; stop execution unlanded)", Note: "external effect with no checkpoint/journal durable write"},
	}},
	{ID: "CR-GRACE-08", Family: "CR-GRACE", Phase: "after closure-delta checkpoint", DurableWrite: WriteCheckpoint, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverCheckpointCapture, Note: "closure-delta checkpoint through the capture blob/receipt seam"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverCheckpointCapture, Note: "closure-delta checkpoint through the capture blob/receipt seam"},
	}},
	{ID: "CR-GRACE-09", Family: "CR-GRACE", Phase: "after destination prepared materialization", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalToPrepared, Note: "destination prepared journal, evaluated with the graceful_takeover path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalToPrepared, Note: "destination prepared journal, evaluated with the graceful_takeover path"},
	}},
	{ID: "CR-GRACE-10", Family: "CR-GRACE", Phase: "after destination lease persistence/union", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lease arbitration beyond sessrepo persistence (lease persistence/union unlanded)", Note: "durable write is lease records, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "lease arbitration beyond sessrepo persistence (lease persistence/union unlanded)", Note: "durable write is lease records, not a checkpoint or journal"},
	}},
	{ID: "CR-GRACE-11", Family: "CR-GRACE", Phase: "after direct resume or task-board activation", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: false, Owner: "provider resume runtime (Section 13.6 direct resume; execution unlanded)", Note: "direct half is a provider effect with no journal write; the task-board half is journal-observable"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalActivation, Note: "task-board activation through the journal-activation seam, evaluated with the graceful_takeover path"},
	}},
	{ID: "CR-GRACE-12", Family: "CR-GRACE", Phase: "after finalize", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalFinalize, Note: "direct finalize has no external effect in the plan: resume under the same IDs"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalFinalize, Note: "task-board finalize through the journal-finalize seam, evaluated with the graceful_takeover path"},
	}},
	{ID: "CR-GRACE-13", Family: "CR-GRACE", Phase: "after source park/resume-event publication", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lifecycle orchestration with sessrepo event publication (Section 13.6; flow unlanded, out of story boundary)", Note: "durable write is park/resume events, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "lifecycle orchestration with sessrepo event publication (Section 13.6; flow unlanded, out of story boundary)", Note: "durable write is park/resume events, not a checkpoint or journal"},
	}},
	// CR-FORCE-01..07: common force takeover (Section 13.7). The
	// selection, staging, preflight, confirmation, and lease rows
	// write no checkpoint or journal; activation rows below carry
	// the journal-observable halves.
	{ID: "CR-FORCE-01", Family: "CR-FORCE", Phase: "after refresh/pinned-cohort selection", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lifecycle orchestration (Section 13.7 refresh/pinned-cohort selection; flow unlanded)", Note: "selection produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "lifecycle orchestration (Section 13.7 refresh/pinned-cohort selection; flow unlanded)", Note: "selection produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-FORCE-02", Family: "CR-FORCE", Phase: "after divergence preservation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "workspace divergence handling (Section 13.7/13.12; preservation runtime unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "workspace divergence handling (Section 13.7/13.12; preservation runtime unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-FORCE-03", Family: "CR-FORCE", Phase: "after inert staging", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "force staging runtime (Section 13.7; filesystem staging is coordinator work, unlanded)", Note: "staging bytes are not journal bytes"},
		{Name: PathTaskBoard, Owner: "force staging runtime (Section 13.7; filesystem staging is coordinator work, unlanded)", Note: "staging bytes are not journal bytes"},
	}},
	{ID: "CR-FORCE-04", Family: "CR-FORCE", Phase: "after final preflight", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lifecycle orchestration (Section 13.7 final preflight; flow unlanded)", Note: "validation produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "lifecycle orchestration (Section 13.7 final preflight; flow unlanded)", Note: "validation produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-FORCE-05", Family: "CR-FORCE", Phase: "after confirmation receipt", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "operator confirmation runtime (Section 13.7; confirmation receipt persistence unlanded)", Note: "an operator confirmation receipt is not a checkpoint/journal operation receipt"},
		{Name: PathTaskBoard, Owner: "operator confirmation runtime (Section 13.7; confirmation receipt persistence unlanded)", Note: "an operator confirmation receipt is not a checkpoint/journal operation receipt"},
	}},
	{ID: "CR-FORCE-06", Family: "CR-FORCE", Phase: "after new lease persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lease arbitration beyond sessrepo persistence (new lease persistence unlanded)", Note: "durable write is lease records, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "lease arbitration beyond sessrepo persistence (new lease persistence unlanded)", Note: "durable write is lease records, not a checkpoint or journal"},
	}},
	{ID: "CR-FORCE-07", Family: "CR-FORCE", Phase: "after winning-lease recomputation/events", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lease arbitration with event publication (Section 13.7; recomputation unlanded)", Note: "durable write is lease/event records, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "lease arbitration with event publication (Section 13.7; recomputation unlanded)", Note: "durable write is lease/event records, not a checkpoint or journal"},
	}},
	// CR-FORCE-D-01..05: direct force activation (Section 13.7).
	{ID: "CR-FORCE-D-01", Family: "CR-FORCE-D", Phase: "after prepare", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalCreate, Note: "direct activation prepare through the journal-create seam, evaluated with the force_takeover path"},
	}},
	{ID: "CR-FORCE-D-02", Family: "CR-FORCE-D", Phase: "after prepared commit/provider materialization", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalToPrepared, Note: "provider materialization prepared under a direct closure with a provider branch, evaluated with the force_takeover path"},
	}},
	{ID: "CR-FORCE-D-03", Family: "CR-FORCE-D", Phase: "after native discovery", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider native discovery runtime (Section 13.7; discovery unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-FORCE-D-04", Family: "CR-FORCE-D", Phase: "after plugin resume/process start", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider plugin/process runtime (Section 13.7; resume and process start unlanded)", Note: "external effect with no checkpoint/journal durable write"},
	}},
	{ID: "CR-FORCE-D-05", Family: "CR-FORCE-D", Phase: "after finalize may have occurred", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalFinalize, Note: "direct finalize has no external effect in the plan: resume under the same IDs, evaluated with the force_takeover path"},
	}},
	// CR-FORCE-TB-01..04: task-board force activation (Section 13.7).
	{ID: "CR-FORCE-TB-01", Family: "CR-FORCE-TB", Phase: "after prepare/journal creation", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalCreate, Note: "task-board activation prepare through the journal-create seam, evaluated with the force_takeover path"},
	}},
	{ID: "CR-FORCE-TB-02", Family: "CR-FORCE-TB", Phase: "after import/open/commit prepared", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalToPrepared, Note: "import/open/commit through the journal-to-prepared seam, evaluated with the force_takeover path"},
	}},
	{ID: "CR-FORCE-TB-03", Family: "CR-FORCE-TB", Phase: "after adopt may have occurred", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalActivation, Note: "adopt through the journal-activation seam, evaluated with the force_takeover path"},
	}},
	{ID: "CR-FORCE-TB-04", Family: "CR-FORCE-TB", Phase: "after resume/finalize may have occurred", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalFinalize, Note: "resume/finalize through the journal-finalize seam, evaluated with the force_takeover path"},
	}},
	// CR-FORK-01..08: fork (Section 13.8). The prepared
	// materialization, task-board adopt, and finalize rows are
	// journal-observable; allocation, projection, record/lease,
	// planning, and direct identity rows belong to their owners.
	{ID: "CR-FORK-01", Family: "CR-FORK", Phase: "after ID allocation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "fork planning (Section 13.8; ID allocation flow unlanded; scalar grammar landed)", Note: "allocation produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "fork planning (Section 13.8; ID allocation flow unlanded; scalar grammar landed)", Note: "allocation produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-FORK-02", Family: "CR-FORK", Phase: "after topology/manifest projection persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "transfer manifest projection (manifest owner; out of checkpoint/journal story boundary)", Note: "durable write is manifest projection, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "transfer manifest projection (manifest owner; out of checkpoint/journal story boundary)", Note: "durable write is manifest projection, not a checkpoint or journal"},
	}},
	{ID: "CR-FORK-03", Family: "CR-FORK", Phase: "after new Session Record/lease persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sessrepo record persistence with lease arbitration (Section 13.8; out of checkpoint/journal story boundary)", Note: "durable write is Session/Lease records, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "sessrepo record persistence with lease arbitration (Section 13.8; out of checkpoint/journal story boundary)", Note: "durable write is Session/Lease records, not a checkpoint or journal"},
	}},
	{ID: "CR-FORK-04", Family: "CR-FORK", Phase: "after provider fork plan or bundle selection", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "fork planning (Section 13.8; provider fork plan and bundle selection unlanded)", Note: "selection produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "fork planning (Section 13.8; provider fork plan and bundle selection unlanded)", Note: "selection produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-FORK-05", Family: "CR-FORK", Phase: "after prepared materialization", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalToPrepared, Note: "fork prepared journal under a workspace-only closure, evaluated with the fork path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalToPrepared, Note: "fork prepared journal, evaluated with the fork path"},
	}},
	{ID: "CR-FORK-06", Family: "CR-FORK", Phase: "after direct identity/activation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider identity and activation runtime (Section 13.8; identity persistence and activation unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-FORK-07", Family: "CR-FORK", Phase: "after task-board adopt/activation", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalActivation, Note: "fork adopt through the journal-activation seam, evaluated with the fork path"},
	}},
	{ID: "CR-FORK-08", Family: "CR-FORK", Phase: "after finalize before fork/resume events", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalFinalize, Note: "direct finalize has no external effect in the plan: resume under the same IDs, evaluated with the fork path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalFinalize, Note: "fork finalize through the journal-finalize seam, evaluated with the fork path"},
	}},
	// CR-STOP-01..05: stop (Section 13.9). The checkpoint/export
	// capture row is checkpoint-observable; input, stop effect,
	// verification, and event rows belong to their owners.
	{ID: "CR-STOP-01", Family: "CR-STOP", Phase: "after input block/quiesce", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "input gating with quiescence observation (Section 13.9; block/quiesce runtime unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "input gating with quiescence observation (Section 13.9; block/quiesce runtime unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-STOP-02", Family: "CR-STOP", Phase: "after checkpoint/export capture", DurableWrite: WriteCheckpoint, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverCheckpointCapture, Note: "stop checkpoint through the capture blob/receipt seam"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverCheckpointCapture, Note: "stop checkpoint through the capture blob/receipt seam (task-board export closure)"},
	}},
	{ID: "CR-STOP-03", Family: "CR-STOP", Phase: "after process/manager stop may have occurred", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider/bridge stop runtime (Section 13.9; stop execution unlanded)", Note: "external effect with no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "provider/bridge stop runtime (Section 13.9; stop execution unlanded)", Note: "external effect with no checkpoint/journal durable write"},
	}},
	{ID: "CR-STOP-04", Family: "CR-STOP", Phase: "after closure verification", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lifecycle orchestration (Section 13.9 closure verification; flow unlanded)", Note: "head binding is enforced at checkpoint capture/admission; flow verification writes no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "lifecycle orchestration (Section 13.9 closure verification; flow unlanded)", Note: "head binding is enforced at checkpoint capture/admission; flow verification writes no checkpoint/journal bytes"},
	}},
	{ID: "CR-STOP-05", Family: "CR-STOP", Phase: "after stopped event/result persistence", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sessrepo event with cliresult result persistence (Section 13.9; out of checkpoint/journal story boundary)", Note: "durable write is stopped event/result records, not a checkpoint or journal"},
		{Name: PathTaskBoard, Owner: "sessrepo event with cliresult result persistence (Section 13.9; out of checkpoint/journal story boundary)", Note: "durable write is stopped event/result records, not a checkpoint or journal"},
	}},
	// CR-RESUME-01..06: owner resume (Section 13.10). Prepare,
	// prepared commit, activation, and finalize rows are
	// journal-observable; validation, discovery, and
	// reconciliation rows belong to their owners.
	{ID: "CR-RESUME-01", Family: "CR-RESUME", Phase: "after owner/checkpoint/profile validation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "resume validation (sessquery admission and provhost profile libraries landed; Section 13.10 flow orchestration unlanded)", Note: "validation produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "resume validation (sessquery admission and provhost profile libraries landed; Section 13.10 flow orchestration unlanded)", Note: "validation produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-RESUME-02", Family: "CR-RESUME", Phase: "after prepare", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalCreate, Note: "resume prepare through the journal-create seam, evaluated with the owner_resume path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalCreate, Note: "resume prepare through the journal-create seam, evaluated with the owner_resume path"},
	}},
	{ID: "CR-RESUME-03", Family: "CR-RESUME", Phase: "after prepared commit", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalToPrepared, Note: "resume prepared commit under a workspace-only closure, evaluated with the owner_resume path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalToPrepared, Note: "resume prepared commit, evaluated with the owner_resume path"},
	}},
	{ID: "CR-RESUME-04", Family: "CR-RESUME", Phase: "after exact native discovery or dormant manager reconciliation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "provider native discovery runtime (Section 13.10; discovery unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "task-board bridge runtime (Section 13.10 dormant manager reconciliation; execution unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-RESUME-05", Family: "CR-RESUME", Phase: "after provider/bridge activation may have occurred", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: false, Owner: "provider activation runtime (Section 13.10 direct activation; execution unlanded) and a workspace-only plan with no journal activation phase", Note: "direct half is a provider effect with no journal write; no journal activation phase exists in the direct plan"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalActivation, Note: "resume activation through the journal-activation seam, evaluated with the owner_resume path"},
	}},
	{ID: "CR-RESUME-06", Family: "CR-RESUME", Phase: "after finalize may have occurred before session.resumed", DurableWrite: WriteJournal, Paths: []BoundaryPath{
		{Name: PathDirect, Reachable: true, Driver: DriverJournalFinalize, Note: "direct finalize has no external effect in the plan: resume under the same IDs, evaluated with the owner_resume path"},
		{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalFinalize, Note: "resume finalize through the journal-finalize seam, evaluated with the owner_resume path"},
	}},
	// CR-RESTORE-01..07: reboot restore (Section 13.11). No
	// boundary writes a checkpoint, journal, receipt, or marker;
	// validation, enumeration, lease, wrapper, and auto-resume
	// rows belong to their owners.
	{ID: "CR-RESTORE-01", Family: "CR-RESTORE", Phase: "after index validation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "localstore SQLite projection (library landed; Section 13.11 restore flow unlanded)", Note: "validation reads the index; no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "localstore SQLite projection (library landed; Section 13.11 restore flow unlanded)", Note: "validation reads the index; no checkpoint/journal durable write"},
	}},
	{ID: "CR-RESTORE-02", Family: "CR-RESTORE", Phase: "after prior-live-session enumeration", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "sessquery read library (landed; Section 13.11 restore flow orchestration unlanded)", Note: "enumeration reads state; no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "sessquery read library (landed; Section 13.11 restore flow orchestration unlanded)", Note: "enumeration reads state; no checkpoint/journal durable write"},
	}},
	{ID: "CR-RESTORE-03", Family: "CR-RESTORE", Phase: "after lease refresh", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lease arbitration beyond sessrepo persistence (lease refresh unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "lease arbitration beyond sessrepo persistence (lease refresh unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-RESTORE-04", Family: "CR-RESTORE", Phase: "after stale-owner marking", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "lease/state arbitration (Section 13.11; stale-owner marking unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "lease/state arbitration (Section 13.11; stale-owner marking unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-RESTORE-05", Family: "CR-RESTORE", Phase: "after checkpoint/profile validation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "restore validation (sessquery admission and provhost profile libraries landed; Section 13.11 flow orchestration unlanded)", Note: "validation produces no checkpoint/journal bytes"},
		{Name: PathTaskBoard, Owner: "restore validation (sessquery admission and provhost profile libraries landed; Section 13.11 flow orchestration unlanded)", Note: "validation produces no checkpoint/journal bytes"},
	}},
	{ID: "CR-RESTORE-06", Family: "CR-RESTORE", Phase: "after wrapper recreation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "terminal wrapper runtime (Section 13.11; recreation unlanded)", Note: "no checkpoint/journal durable write"},
		{Name: PathTaskBoard, Owner: "terminal wrapper runtime (Section 13.11; recreation unlanded)", Note: "no checkpoint/journal durable write"},
	}},
	{ID: "CR-RESTORE-07", Family: "CR-RESTORE", Phase: "after auto-resume may have occurred", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "auto-resume runtime with status/event reconciliation (Section 13.10/13.11 lifecycle; unlanded)", Note: "external effect with no checkpoint/journal durable write; the journal-observable finalize half is covered by CR-MAT-08 and CR-RESUME-06"},
		{Name: PathTaskBoard, Owner: "auto-resume runtime with status/event reconciliation (Section 13.10/13.11 lifecycle; unlanded)", Note: "external effect with no checkpoint/journal durable write; the journal-observable finalize half is covered by CR-MAT-08 and CR-RESUME-06"},
	}},
	// CR-CLONE-01..16: cross-environment clone fault injection
	// (Section 13.13 prose after the registry table, in order:
	// resolve; probe; inspect; snapshot; capture plan; raw
	// capture; normalize; canonical validation;
	// projection/checkpoint plan; policy gate; prepare; staged
	// read-back; source/collision recheck; publish/live
	// read-back; finalize/lineage; resume-plan/optional open).
	// The clone flow is unlanded; the finalization injection
	// prefix list (Provider commit through G4) is a stated bound
	// in TRACEABILITY.md.
	{ID: "CR-CLONE-01", Family: "CR-CLONE", Phase: "after resolve", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-02", Family: "CR-CLONE", Phase: "after probe", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-03", Family: "CR-CLONE", Phase: "after inspect", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-04", Family: "CR-CLONE", Phase: "after snapshot", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-05", Family: "CR-CLONE", Phase: "after capture plan", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-06", Family: "CR-CLONE", Phase: "after raw capture", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-07", Family: "CR-CLONE", Phase: "after normalize", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-08", Family: "CR-CLONE", Phase: "after canonical validation", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-09", Family: "CR-CLONE", Phase: "after projection/checkpoint plan", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-10", Family: "CR-CLONE", Phase: "after policy gate", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-11", Family: "CR-CLONE", Phase: "after prepare", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone prepare is not the journal materialize.prepare; the clone flow is unlanded"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone prepare is not the journal materialize.prepare; the clone flow is unlanded"},
	}},
	{ID: "CR-CLONE-12", Family: "CR-CLONE", Phase: "after staged read-back", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-13", Family: "CR-CLONE", Phase: "after source/collision recheck", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-14", Family: "CR-CLONE", Phase: "after publish/live read-back", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-15", Family: "CR-CLONE", Phase: "after finalize/lineage", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
	{ID: "CR-CLONE-16", Family: "CR-CLONE", Phase: "after resume-plan/optional open", DurableWrite: WriteExternal, Paths: []BoundaryPath{
		{Name: PathDirect, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
		{Name: PathTaskBoard, Owner: "clone flow (Section 13.14; unlanded)", Note: "clone derivation writes no checkpoint or journal in the landed owners"},
	}},
}

// Registry returns an isolated copy of the classified boundary
// table in specification order.
func Registry() []Boundary {
	out := make([]Boundary, len(registry))
	for index, boundary := range registry {
		paths := make([]BoundaryPath, len(boundary.Paths))
		copy(paths, boundary.Paths)
		boundary.Paths = paths
		out[index] = boundary
	}
	return out
}

// Lookup returns the boundary with the given ID.
func Lookup(id string) (Boundary, bool) {
	for _, boundary := range registry {
		if boundary.ID == id {
			return boundary, true
		}
	}
	return Boundary{}, false
}
